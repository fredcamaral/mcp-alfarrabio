// Package migration provides database schema migration functionality with rollback support.
package migration

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"lerian-mcp-memory/pkg/types"
)

// Migrator handles database schema migrations
type Migrator struct {
	db            *sql.DB
	migrationsDir string
	environment   string
	executedBy    string
	executionHost string
	appVersion    string
}

// Migration represents a database migration
type Migration struct {
	ID          string
	Filename    string
	Version     int
	Description string
	Content     string
	Checksum    string

	// Parsed SQL
	UpSQL   string
	DownSQL string

	// Metadata
	IsDestructive bool
	HasRollback   bool
	Dependencies  []string
	Tags          []string
	Priority      string
	MigrationType string

	// Execution context
	EstimatedDuration time.Duration
	Notes             string
	ExternalRefs      map[string]interface{}
}

// MigrationResult represents the result of a migration execution
type MigrationResult struct {
	Migration       *Migration
	Success         bool
	ExecutionTime   time.Duration
	AffectedRows    int64
	Error           error
	ValidationError error
	TableChanges    map[string]interface{}
	IndexChanges    map[string]interface{}
}

// BatchResult represents the result of a batch migration
type BatchResult struct {
	BatchID         int
	TotalMigrations int
	SuccessfulCount int
	FailedCount     int
	Results         []*MigrationResult
	TotalDuration   time.Duration
	Error           error
}

// NewMigrator creates a new migrator instance
func NewMigrator(db *sql.DB, migrationsDir, environment string) *Migrator {
	hostname, _ := os.Hostname()

	return &Migrator{
		db:            db,
		migrationsDir: migrationsDir,
		environment:   environment,
		executedBy:    os.Getenv("USER"),
		executionHost: hostname,
		appVersion:    os.Getenv("APP_VERSION"),
	}
}

// LoadMigrations loads all migration files from the migrations directory
func (m *Migrator) LoadMigrations(ctx context.Context) ([]*Migration, error) {
	var migrations []*Migration

	err := filepath.WalkDir(m.migrationsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".sql") {
			return nil
		}

		migration, parseErr := m.parseMigrationFile(path)
		if parseErr != nil {
			return fmt.Errorf("failed to parse migration %s: %w", path, parseErr)
		}

		migrations = append(migrations, migration)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to load migrations: %w", err)
	}

	// Sort migrations by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// parseMigrationFile parses a migration file and extracts metadata
func (m *Migrator) parseMigrationFile(filePath string) (*Migration, error) {
	// Clean and validate the file path
	filePath = filepath.Clean(filePath)
	if strings.Contains(filePath, "..") {
		return nil, fmt.Errorf("path traversal detected: %s", filePath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read migration file: %w", err)
	}

	contentStr := string(content)
	filename := filepath.Base(filePath)

	// Parse version and description from filename (e.g., "001_create_users_table.sql")
	filenameRegex := regexp.MustCompile(`^(\d+)_(.+)\.sql$`)
	matches := filenameRegex.FindStringSubmatch(filename)
	if len(matches) != 3 {
		return nil, fmt.Errorf("invalid migration filename format: %s (expected: NNN_description.sql)", filename)
	}

	version, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil, fmt.Errorf("invalid version number in filename: %s", matches[1])
	}

	description := strings.ReplaceAll(matches[2], "_", " ")

	// Generate migration ID
	migrationID := fmt.Sprintf("%03d_%s", version, matches[2])

	// Calculate checksum
	hasher := sha256.New()
	hasher.Write(content)
	checksum := hex.EncodeToString(hasher.Sum(nil))

	// Parse UP and DOWN sections
	upSQL, downSQL := m.parseUpDownSQL(contentStr)

	// Parse metadata from comments
	metadata := m.parseMetadata(contentStr)

	migration := &Migration{
		ID:          migrationID,
		Filename:    filename,
		Version:     version,
		Description: description,
		Content:     contentStr,
		Checksum:    checksum,
		UpSQL:       upSQL,
		DownSQL:     downSQL,
		HasRollback: downSQL != "",

		// Apply metadata
		IsDestructive: metadata.IsDestructive,
		Dependencies:  metadata.Dependencies,
		Tags:          metadata.Tags,
		Priority:      metadata.Priority,
		MigrationType: metadata.MigrationType,
		Notes:         metadata.Notes,
		ExternalRefs:  metadata.ExternalRefs,
	}

	return migration, nil
}

// parseUpDownSQL parses UP and DOWN sections from migration content
func (m *Migrator) parseUpDownSQL(content string) (upSQL, downSQL string) {
	// Look for -- UP and -- DOWN markers
	upMarker := regexp.MustCompile(`(?i)--\s*UP\s*$`)
	downMarker := regexp.MustCompile(`(?i)--\s*DOWN\s*$`)

	lines := strings.Split(content, "\n")
	var currentSection string
	var upLines, downLines []string

	for _, line := range lines {
		if upMarker.MatchString(line) {
			currentSection = "UP"
			continue
		}
		if downMarker.MatchString(line) {
			currentSection = "DOWN"
			continue
		}

		switch currentSection {
		case "UP":
			upLines = append(upLines, line)
		case "DOWN":
			downLines = append(downLines, line)
		default:
			// If no markers found, treat entire content as UP
			if currentSection == "" {
				upLines = append(upLines, line)
			}
		}
	}

	upSQL = strings.TrimSpace(strings.Join(upLines, "\n"))
	downSQL = strings.TrimSpace(strings.Join(downLines, "\n"))

	// If no UP/DOWN markers, treat entire content as UP
	if currentSection == "" {
		upSQL = strings.TrimSpace(content)
	}

	return upSQL, downSQL
}

// MigrationMetadata represents metadata parsed from migration comments
type MigrationMetadata struct {
	IsDestructive bool
	Dependencies  []string
	Tags          []string
	Priority      string
	MigrationType string
	Notes         string
	ExternalRefs  map[string]interface{}
}

// parseMetadata extracts metadata from migration comments
func (m *Migrator) parseMetadata(content string) *MigrationMetadata {
	metadata := &MigrationMetadata{
		Dependencies:  []string{},
		Tags:          []string{},
		Priority:      "normal",
		MigrationType: "schema",
		ExternalRefs:  make(map[string]interface{}),
	}

	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "--") {
			continue
		}

		comment := strings.TrimSpace(strings.TrimPrefix(line, "--"))
		m.parseMetadataComment(comment, metadata)
	}

	return metadata
}

// parseMetadataComment parses a single metadata comment
func (m *Migrator) parseMetadataComment(comment string, metadata *MigrationMetadata) {
	switch {
	case strings.HasPrefix(comment, "DESTRUCTIVE:"):
		metadata.IsDestructive = strings.Contains(strings.ToLower(comment), "true")
	case strings.HasPrefix(comment, "DEPENDS:"):
		m.parseDependencies(comment, metadata)
	case strings.HasPrefix(comment, "TAGS:"):
		m.parseTags(comment, metadata)
	case strings.HasPrefix(comment, "PRIORITY:"):
		m.parsePriority(comment, metadata)
	case strings.HasPrefix(comment, "TYPE:"):
		m.parseType(comment, metadata)
	case strings.HasPrefix(comment, "NOTES:"):
		metadata.Notes = strings.TrimSpace(strings.TrimPrefix(comment, "NOTES:"))
	}
}

// parseDependencies parses dependency information
func (m *Migrator) parseDependencies(comment string, metadata *MigrationMetadata) {
	deps := strings.TrimSpace(strings.TrimPrefix(comment, "DEPENDS:"))
	if deps == "" {
		return
	}

	metadata.Dependencies = strings.Split(deps, ",")
	for i := range metadata.Dependencies {
		metadata.Dependencies[i] = strings.TrimSpace(metadata.Dependencies[i])
	}
}

// parseTags parses tag information
func (m *Migrator) parseTags(comment string, metadata *MigrationMetadata) {
	tags := strings.TrimSpace(strings.TrimPrefix(comment, "TAGS:"))
	if tags == "" {
		return
	}

	metadata.Tags = strings.Split(tags, ",")
	for i := range metadata.Tags {
		metadata.Tags[i] = strings.TrimSpace(metadata.Tags[i])
	}
}

// parsePriority parses priority information
func (m *Migrator) parsePriority(comment string, metadata *MigrationMetadata) {
	priority := strings.TrimSpace(strings.TrimPrefix(comment, "PRIORITY:"))
	if priority != "" {
		metadata.Priority = strings.ToLower(priority)
	}
}

// parseType parses migration type information
func (m *Migrator) parseType(comment string, metadata *MigrationMetadata) {
	migType := strings.TrimSpace(strings.TrimPrefix(comment, "TYPE:"))
	if migType != "" {
		metadata.MigrationType = strings.ToLower(migType)
	}
}

// GetExecutedMigrations returns all executed migrations for the current environment
func (m *Migrator) GetExecutedMigrations(ctx context.Context) ([]*Migration, error) {
	query := `
		SELECT migration_id, filename, version, description, executed_at,
		       execution_time_ms, checksum, is_rolled_back, rollback_sql,
		       dependencies, tags, migration_type, priority, notes,
		       is_destructive, has_rollback
		FROM schema_migrations 
		WHERE environment = $1 
		ORDER BY version ASC`

	rows, err := m.db.QueryContext(ctx, query, m.environment)
	if err != nil {
		return nil, fmt.Errorf("failed to query executed migrations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var migrations []*Migration
	for rows.Next() {
		migration := &Migration{}
		var executedAt time.Time
		var executionTimeMs sql.NullInt64
		var rollbackSQL sql.NullString
		var dependencies, tags []string

		err := rows.Scan(
			&migration.ID, &migration.Filename, &migration.Version, &migration.Description,
			&executedAt, &executionTimeMs, &migration.Checksum, &migration.IsDestructive,
			&rollbackSQL, &dependencies, &tags, &migration.MigrationType,
			&migration.Priority, &migration.Notes, &migration.IsDestructive, &migration.HasRollback,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration: %w", err)
		}

		migration.Dependencies = dependencies
		migration.Tags = tags
		migration.DownSQL = rollbackSQL.String

		migrations = append(migrations, migration)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate migration rows: %w", err)
	}

	return migrations, nil
}

// GetPendingMigrations returns migrations that haven't been executed yet
func (m *Migrator) GetPendingMigrations(ctx context.Context) ([]*Migration, error) {
	allMigrations, err := m.LoadMigrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load migrations: %w", err)
	}

	executedMigrations, err := m.GetExecutedMigrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get executed migrations: %w", err)
	}

	// Create map of executed migration IDs
	executed := make(map[string]bool)
	for _, migration := range executedMigrations {
		executed[migration.ID] = true
	}

	// Filter out executed migrations
	var pending []*Migration
	for _, migration := range allMigrations {
		if !executed[migration.ID] {
			pending = append(pending, migration)
		}
	}

	return pending, nil
}

// ExecuteMigration executes a single migration
func (m *Migrator) ExecuteMigration(ctx context.Context, migration *Migration, dryRun bool) (*MigrationResult, error) {
	result := &MigrationResult{
		Migration: migration,
		Success:   false,
	}

	startTime := time.Now()

	// Validate migration before execution
	if err := m.validateMigration(ctx, migration); err != nil {
		result.ValidationError = err
		return result, fmt.Errorf("migration validation failed: %w", err)
	}

	// If dry run, just validate and return
	if dryRun {
		result.Success = true
		result.ExecutionTime = time.Since(startTime)
		return result, nil
	}

	// Start transaction
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		result.Error = err
		return result, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if result.Success {
			if commitErr := tx.Commit(); commitErr != nil {
				result.Error = commitErr
				result.Success = false
			}
		} else {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				// Log rollback error but don't override original error
				fmt.Printf("Failed to rollback transaction: %v\n", rollbackErr)
			}
		}
	}()

	// Execute the migration SQL
	sqlResult, err := tx.ExecContext(ctx, migration.UpSQL)
	if err != nil {
		result.Error = err
		return result, fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Get affected rows
	if affectedRows, rowErr := sqlResult.RowsAffected(); rowErr == nil {
		result.AffectedRows = affectedRows
	}

	// Record migration in tracking table
	if err := m.recordMigrationExecution(ctx, tx, migration, result, startTime); err != nil {
		result.Error = err
		return result, fmt.Errorf("failed to record migration: %w", err)
	}

	result.Success = true
	result.ExecutionTime = time.Since(startTime)

	return result, nil
}

// recordMigrationExecution records the migration execution in the tracking table
func (m *Migrator) recordMigrationExecution(ctx context.Context, tx *sql.Tx, migration *Migration, result *MigrationResult, startTime time.Time) error {
	query := `
		INSERT INTO schema_migrations (
			migration_id, filename, version, description, executed_at,
			execution_time_ms, rollback_sql, checksum, dependencies, tags,
			environment, executed_by, execution_host, application_version,
			is_destructive, has_rollback, migration_type, priority, notes,
			affected_rows, validation_status
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21
		)`

	executionTimeMs := int(result.ExecutionTime.Milliseconds())
	dependencies := migration.Dependencies
	if dependencies == nil {
		dependencies = []string{}
	}
	tags := migration.Tags
	if tags == nil {
		tags = []string{}
	}

	var rollbackSQL *string
	if migration.DownSQL != "" {
		rollbackSQL = &migration.DownSQL
	}

	_, err := tx.ExecContext(ctx, query,
		migration.ID, migration.Filename, migration.Version, migration.Description,
		startTime, executionTimeMs, rollbackSQL, migration.Checksum,
		dependencies, tags, m.environment, m.executedBy, m.executionHost,
		m.appVersion, migration.IsDestructive, migration.HasRollback,
		migration.MigrationType, migration.Priority, migration.Notes,
		result.AffectedRows, "passed",
	)

	if err != nil {
		return fmt.Errorf("failed to insert migration record: %w", err)
	}

	return nil
}

// validateMigration performs comprehensive safety checks on a migration before execution
func (m *Migrator) validateMigration(ctx context.Context, migration *Migration) error {
	// Create validator
	validator := NewMigrationValidator(m.db)

	// Run comprehensive validation
	result, err := validator.ValidateMigration(ctx, migration)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Check for validation errors
	if !result.Passed {
		var errorMessages []string
		for _, issue := range result.Issues {
			if issue.Severity == SeverityError {
				errorMessages = append(errorMessages, issue.Message)
			}
		}
		if len(errorMessages) > 0 {
			return fmt.Errorf("migration validation failed: %s", strings.Join(errorMessages, "; "))
		}
	}

	// Log warnings and suggestions
	if result.Warnings > 0 {
		fmt.Printf("Migration validation warnings for %s:\n", migration.ID)
		for _, issue := range result.Issues {
			if issue.Severity == SeverityWarning {
				fmt.Printf("  WARNING: %s\n", issue.Message)
			}
		}
		if len(result.Suggestions) > 0 {
			fmt.Printf("  Suggestions:\n")
			for _, suggestion := range result.Suggestions {
				fmt.Printf("    - %s\n", suggestion)
			}
		}
	}

	return nil
}

// Removed unused method migrationExists

// Removed unused method checkDestructiveOperations

// GetCurrentVersion returns the current migration version
func (m *Migrator) GetCurrentVersion(ctx context.Context) (int, error) {
	query := `SELECT get_current_migration_version($1)`

	var version int
	err := m.db.QueryRowContext(ctx, query, m.environment).Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("failed to get current version: %w", err)
	}

	return version, nil
}

// GetMigrationStatistics returns comprehensive migration statistics
func (m *Migrator) GetMigrationStatistics(ctx context.Context) (*types.MigrationStatistics, error) {
	query := `SELECT * FROM get_migration_statistics($1)`

	stats := &types.MigrationStatistics{}
	err := m.db.QueryRowContext(ctx, query, m.environment).Scan(
		&stats.TotalMigrations,
		&stats.SuccessfulMigrations,
		&stats.RolledBackMigrations,
		&stats.CurrentVersion,
		&stats.LastMigrationDate,
		&stats.AvgExecutionTimeMs,
		&stats.DestructiveMigrations,
		&stats.MigrationsWithoutRollback,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get migration statistics: %w", err)
	}

	return stats, nil
}

// GetDB returns the database connection (for testing and validation)
func (m *Migrator) GetDB() *sql.DB {
	return m.db
}
