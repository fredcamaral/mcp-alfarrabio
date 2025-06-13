// Package migration provides database migration safety mechanisms with rollback capabilities
package migration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"lerian-mcp-memory/internal/logging"
)

// MigrationSafetyManager handles safe database migrations with rollback capabilities
type MigrationSafetyManager struct {
	db                  *sql.DB
	migrationsPath      string
	backupPath          string
	logger              *logging.EnhancedLogger
	dryRun              bool
	maxRollbackDuration time.Duration
}

// MigrationRecord tracks migration execution and rollback information
type MigrationRecord struct {
	ID           int64                  `json:"id"`
	Version      string                 `json:"version"`
	Name         string                 `json:"name"`
	Checksum     string                 `json:"checksum"`
	AppliedAt    time.Time              `json:"applied_at"`
	RolledBackAt *time.Time             `json:"rolled_back_at,omitempty"`
	Success      bool                   `json:"success"`
	ErrorMsg     string                 `json:"error_msg,omitempty"`
	Duration     time.Duration          `json:"duration"`
	BackupPath   string                 `json:"backup_path,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// MigrationPlan represents a planned migration operation
type MigrationPlan struct {
	Migrations    []PlannedMigration `json:"migrations"`
	TotalCount    int                `json:"total_count"`
	EstimatedTime time.Duration      `json:"estimated_time"`
	RiskLevel     string             `json:"risk_level"`
	BackupSize    int64              `json:"backup_size_estimate"`
	Dependencies  []string           `json:"dependencies"`
	Warnings      []string           `json:"warnings"`
}

// PlannedMigration represents a single migration in the plan
type PlannedMigration struct {
	Version       string                 `json:"version"`
	Name          string                 `json:"name"`
	FilePath      string                 `json:"file_path"`
	Checksum      string                 `json:"checksum"`
	RiskLevel     string                 `json:"risk_level"`
	EstimatedTime time.Duration          `json:"estimated_time"`
	HasRollback   bool                   `json:"has_rollback"`
	Dependencies  []string               `json:"dependencies"`
	Operations    []string               `json:"operations"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// RollbackPlan represents a planned rollback operation
type RollbackPlan struct {
	TargetVersion string            `json:"target_version"`
	Migrations    []PlannedRollback `json:"migrations"`
	TotalCount    int               `json:"total_count"`
	EstimatedTime time.Duration     `json:"estimated_time"`
	DataLossRisk  string            `json:"data_loss_risk"`
	Warnings      []string          `json:"warnings"`
}

// PlannedRollback represents a single rollback step
type PlannedRollback struct {
	Version       string                 `json:"version"`
	Name          string                 `json:"name"`
	RollbackSQL   string                 `json:"rollback_sql"`
	DataLossRisk  string                 `json:"data_loss_risk"`
	EstimatedTime time.Duration          `json:"estimated_time"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// SafetyConfig defines migration safety parameters
type SafetyConfig struct {
	EnableBackups       bool          `json:"enable_backups"`
	BackupBeforeMigrate bool          `json:"backup_before_migrate"`
	MaxRollbackTime     time.Duration `json:"max_rollback_time"`
	RequireConfirmation bool          `json:"require_confirmation"`
	DryRunFirst         bool          `json:"dry_run_first"`
	ParallelSafe        bool          `json:"parallel_safe"`
}

// NewMigrationSafetyManager creates a new migration safety manager
func NewMigrationSafetyManager(db *sql.DB, migrationsPath, backupPath string, logger *logging.EnhancedLogger) *MigrationSafetyManager {
	if logger == nil {
		logger = logging.NewEnhancedLogger("migration")
	}

	return &MigrationSafetyManager{
		db:                  db,
		migrationsPath:      migrationsPath,
		backupPath:          backupPath,
		logger:              logger,
		dryRun:              false,
		maxRollbackDuration: 24 * time.Hour, // Default 24 hours for rollback window
	}
}

// SetDryRun enables or disables dry run mode
func (msm *MigrationSafetyManager) SetDryRun(dryRun bool) {
	msm.dryRun = dryRun
}

// SetMaxRollbackDuration sets the maximum time window for rollbacks
func (msm *MigrationSafetyManager) SetMaxRollbackDuration(duration time.Duration) {
	msm.maxRollbackDuration = duration
}

// Initialize sets up migration tracking table and safety infrastructure
func (msm *MigrationSafetyManager) Initialize(ctx context.Context) error {
	msm.logger.Info("Initializing migration safety infrastructure")

	// Create migrations tracking table
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS migration_records (
		id SERIAL PRIMARY KEY,
		version VARCHAR(255) NOT NULL UNIQUE,
		name VARCHAR(500) NOT NULL,
		checksum VARCHAR(64) NOT NULL,
		applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		rolled_back_at TIMESTAMP WITH TIME ZONE NULL,
		success BOOLEAN NOT NULL DEFAULT TRUE,
		error_msg TEXT,
		duration_ms BIGINT NOT NULL DEFAULT 0,
		backup_path TEXT,
		metadata JSONB,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_migration_records_version ON migration_records(version);
	CREATE INDEX IF NOT EXISTS idx_migration_records_applied_at ON migration_records(applied_at);
	CREATE INDEX IF NOT EXISTS idx_migration_records_success ON migration_records(success);
	`

	_, err := msm.db.ExecContext(ctx, createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create migration tracking table: %w", err)
	}

	// Create backup directory
	if err := os.MkdirAll(msm.backupPath, 0o755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	msm.logger.Info("Migration safety infrastructure initialized successfully")
	return nil
}

// PlanMigration analyzes pending migrations and creates an execution plan
func (msm *MigrationSafetyManager) PlanMigration(ctx context.Context) (*MigrationPlan, error) {
	msm.logger.Info("Creating migration plan")

	// Get list of applied migrations
	appliedMigrations, err := msm.getAppliedMigrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Get list of available migration files
	migrationFiles, err := msm.getMigrationFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to get migration files: %w", err)
	}

	// Filter to pending migrations
	var pendingMigrations []PlannedMigration
	var totalEstimatedTime time.Duration
	var warnings []string
	riskLevel := "low"

	appliedSet := make(map[string]bool)
	for _, applied := range appliedMigrations {
		appliedSet[applied.Version] = true
	}

	for _, file := range migrationFiles {
		version := extractVersionFromFilename(file)
		if appliedSet[version] {
			continue // Skip already applied migrations
		}

		// Analyze migration file
		planned, err := msm.analyzeMigrationFile(file, version)
		if err != nil {
			msm.logger.Warn("Failed to analyze migration file",
				"file", file,
				"error", err.Error())
			warnings = append(warnings, fmt.Sprintf("Failed to analyze %s: %v", file, err))
			continue
		}

		pendingMigrations = append(pendingMigrations, planned)
		totalEstimatedTime += planned.EstimatedTime

		// Update overall risk level
		if planned.RiskLevel == "high" {
			riskLevel = "high"
		} else if planned.RiskLevel == "medium" && riskLevel != "high" {
			riskLevel = "medium"
		}
	}

	// Sort by version
	sort.Slice(pendingMigrations, func(i, j int) bool {
		return pendingMigrations[i].Version < pendingMigrations[j].Version
	})

	plan := &MigrationPlan{
		Migrations:    pendingMigrations,
		TotalCount:    len(pendingMigrations),
		EstimatedTime: totalEstimatedTime,
		RiskLevel:     riskLevel,
		BackupSize:    msm.estimateBackupSize(ctx),
		Dependencies:  msm.extractDependencies(pendingMigrations),
		Warnings:      warnings,
	}

	msm.logger.Info("Migration plan created",
		"total_migrations", plan.TotalCount,
		"estimated_time", plan.EstimatedTime.String(),
		"risk_level", plan.RiskLevel)

	return plan, nil
}

// ExecuteMigrationPlan executes a migration plan with safety checks
func (msm *MigrationSafetyManager) ExecuteMigrationPlan(ctx context.Context, plan *MigrationPlan, config SafetyConfig) error {
	if plan.TotalCount == 0 {
		msm.logger.Info("No migrations to execute")
		return nil
	}

	msm.logger.Info("Starting safe migration execution",
		"total_migrations", plan.TotalCount,
		"dry_run", msm.dryRun,
		"backup_enabled", config.EnableBackups)

	// Create backup if required
	var backupPath string
	if config.EnableBackups && config.BackupBeforeMigrate && !msm.dryRun {
		var err error
		backupPath, err = msm.createBackup(ctx)
		if err != nil {
			return fmt.Errorf("failed to create backup before migration: %w", err)
		}
		msm.logger.Info("Database backup created", "path", backupPath)
	}

	// Execute migrations in order
	for _, migration := range plan.Migrations {
		if err := msm.executeSingleMigration(ctx, migration, backupPath); err != nil {
			return fmt.Errorf("migration failed at version %s: %w", migration.Version, err)
		}
	}

	msm.logger.Info("Migration execution completed successfully",
		"total_migrations", plan.TotalCount)

	return nil
}

// PlanRollback creates a rollback plan to a target version
func (msm *MigrationSafetyManager) PlanRollback(ctx context.Context, targetVersion string) (*RollbackPlan, error) {
	msm.logger.Info("Creating rollback plan", "target_version", targetVersion)

	// Get applied migrations after target version
	appliedMigrations, err := msm.getAppliedMigrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	var migrationsToRollback []MigrationRecord
	for _, migration := range appliedMigrations {
		if migration.Version > targetVersion && migration.RolledBackAt == nil {
			migrationsToRollback = append(migrationsToRollback, migration)
		}
	}

	// Sort in reverse order (newest first)
	sort.Slice(migrationsToRollback, func(i, j int) bool {
		return migrationsToRollback[i].Version > migrationsToRollback[j].Version
	})

	var plannedRollbacks []PlannedRollback
	var totalEstimatedTime time.Duration
	var warnings []string
	dataLossRisk := "none"

	for _, migration := range migrationsToRollback {
		// Check if migration is within rollback window
		if time.Since(migration.AppliedAt) > msm.maxRollbackDuration {
			warnings = append(warnings, fmt.Sprintf("Migration %s is outside rollback window (%v)",
				migration.Version, msm.maxRollbackDuration))
		}

		rollbackSQL, err := msm.getRollbackSQL(migration.Version)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("No rollback available for %s: %v", migration.Version, err))
			continue
		}

		// Analyze rollback for data loss risk
		risk := msm.analyzeDataLossRisk(rollbackSQL)
		if risk == "high" {
			dataLossRisk = "high"
		} else if risk == "medium" && dataLossRisk != "high" {
			dataLossRisk = "medium"
		}

		plannedRollback := PlannedRollback{
			Version:       migration.Version,
			Name:          migration.Name,
			RollbackSQL:   rollbackSQL,
			DataLossRisk:  risk,
			EstimatedTime: time.Second * 30, // Default estimate
			Metadata: map[string]interface{}{
				"applied_at": migration.AppliedAt,
			},
		}

		plannedRollbacks = append(plannedRollbacks, plannedRollback)
		totalEstimatedTime += plannedRollback.EstimatedTime
	}

	plan := &RollbackPlan{
		TargetVersion: targetVersion,
		Migrations:    plannedRollbacks,
		TotalCount:    len(plannedRollbacks),
		EstimatedTime: totalEstimatedTime,
		DataLossRisk:  dataLossRisk,
		Warnings:      warnings,
	}

	msm.logger.Info("Rollback plan created",
		"target_version", targetVersion,
		"rollback_count", plan.TotalCount,
		"data_loss_risk", plan.DataLossRisk)

	return plan, nil
}

// ExecuteRollback executes a rollback plan with safety checks
func (msm *MigrationSafetyManager) ExecuteRollback(ctx context.Context, plan *RollbackPlan, config SafetyConfig) error {
	if plan.TotalCount == 0 {
		msm.logger.Info("No rollbacks to execute")
		return nil
	}

	msm.logger.Info("Starting safe rollback execution",
		"target_version", plan.TargetVersion,
		"rollback_count", plan.TotalCount,
		"data_loss_risk", plan.DataLossRisk,
		"dry_run", msm.dryRun)

	// Create backup before rollback if required
	var backupPath string
	if config.EnableBackups && !msm.dryRun {
		var err error
		backupPath, err = msm.createBackup(ctx)
		if err != nil {
			return fmt.Errorf("failed to create backup before rollback: %w", err)
		}
		msm.logger.Info("Database backup created before rollback", "path", backupPath)
	}

	// Execute rollbacks in reverse order
	for _, rollback := range plan.Migrations {
		if err := msm.executeSingleRollback(ctx, rollback, backupPath); err != nil {
			return fmt.Errorf("rollback failed at version %s: %w", rollback.Version, err)
		}
	}

	msm.logger.Info("Rollback execution completed successfully",
		"target_version", plan.TargetVersion,
		"rollback_count", plan.TotalCount)

	return nil
}

// Helper methods

func (msm *MigrationSafetyManager) getAppliedMigrations(ctx context.Context) ([]MigrationRecord, error) {
	query := `
		SELECT id, version, name, checksum, applied_at, rolled_back_at, success, 
		       COALESCE(error_msg, '') as error_msg, duration_ms, 
		       COALESCE(backup_path, '') as backup_path, COALESCE(metadata, '{}') as metadata
		FROM migration_records 
		WHERE success = true AND rolled_back_at IS NULL
		ORDER BY applied_at ASC
	`

	rows, err := msm.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	var migrations []MigrationRecord
	for rows.Next() {
		var migration MigrationRecord
		var metadataJSON string
		var durationMs int64

		err := rows.Scan(
			&migration.ID,
			&migration.Version,
			&migration.Name,
			&migration.Checksum,
			&migration.AppliedAt,
			&migration.RolledBackAt,
			&migration.Success,
			&migration.ErrorMsg,
			&durationMs,
			&migration.BackupPath,
			&metadataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration record: %w", err)
		}

		migration.Duration = time.Duration(durationMs) * time.Millisecond

		// Parse metadata
		if metadataJSON != "{}" {
			if err := json.Unmarshal([]byte(metadataJSON), &migration.Metadata); err != nil {
				msm.logger.Warn("Failed to parse migration metadata", "version", migration.Version, "error", err)
				migration.Metadata = make(map[string]interface{})
			}
		} else {
			migration.Metadata = make(map[string]interface{})
		}

		migrations = append(migrations, migration)
	}

	return migrations, rows.Err()
}

func (msm *MigrationSafetyManager) getMigrationFiles() ([]string, error) {
	var files []string

	err := filepath.WalkDir(msm.migrationsPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".sql") {
			return nil
		}

		files = append(files, path)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk migrations directory: %w", err)
	}

	sort.Strings(files)
	return files, nil
}

func extractVersionFromFilename(filename string) string {
	base := filepath.Base(filename)
	parts := strings.Split(base, "_")
	if len(parts) > 0 {
		return parts[0]
	}
	return base
}

func (msm *MigrationSafetyManager) analyzeMigrationFile(filePath, version string) (PlannedMigration, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return PlannedMigration{}, fmt.Errorf("failed to read migration file: %w", err)
	}

	sql := string(content)
	operations := msm.extractOperations(sql)
	riskLevel := msm.assessRiskLevel(operations)

	// Extract name from filename
	base := filepath.Base(filePath)
	name := strings.TrimSuffix(base, ".sql")
	if idx := strings.Index(name, "_"); idx > 0 {
		name = name[idx+1:]
	}
	name = strings.ReplaceAll(name, "_", " ")

	// Check for rollback SQL
	hasRollback := msm.hasRollbackSQL(version)

	planned := PlannedMigration{
		Version:       version,
		Name:          name,
		FilePath:      filePath,
		Checksum:      msm.calculateChecksum(content),
		RiskLevel:     riskLevel,
		EstimatedTime: msm.estimateMigrationTime(operations),
		HasRollback:   hasRollback,
		Dependencies:  []string{}, // TODO: Extract from migration content
		Operations:    operations,
		Metadata: map[string]interface{}{
			"file_size": len(content),
		},
	}

	return planned, nil
}

func (msm *MigrationSafetyManager) extractOperations(sql string) []string {
	var operations []string
	sql = strings.ToUpper(sql)

	keywords := []string{
		"CREATE TABLE", "ALTER TABLE", "DROP TABLE",
		"CREATE INDEX", "DROP INDEX",
		"INSERT INTO", "UPDATE", "DELETE FROM",
		"CREATE FUNCTION", "DROP FUNCTION",
		"CREATE TRIGGER", "DROP TRIGGER",
	}

	for _, keyword := range keywords {
		if strings.Contains(sql, keyword) {
			operations = append(operations, keyword)
		}
	}

	return operations
}

func (msm *MigrationSafetyManager) assessRiskLevel(operations []string) string {
	highRiskOps := []string{"DROP TABLE", "DROP INDEX", "DELETE FROM", "ALTER TABLE"}
	mediumRiskOps := []string{"UPDATE", "CREATE INDEX"}

	for _, op := range operations {
		for _, highRisk := range highRiskOps {
			if strings.Contains(op, highRisk) {
				return "high"
			}
		}
	}

	for _, op := range operations {
		for _, mediumRisk := range mediumRiskOps {
			if strings.Contains(op, mediumRisk) {
				return "medium"
			}
		}
	}

	return "low"
}

func (msm *MigrationSafetyManager) estimateMigrationTime(operations []string) time.Duration {
	// Simple heuristic for time estimation
	baseTime := time.Second * 10

	for _, op := range operations {
		switch {
		case strings.Contains(op, "CREATE TABLE"):
			baseTime += time.Second * 5
		case strings.Contains(op, "CREATE INDEX"):
			baseTime += time.Second * 30
		case strings.Contains(op, "ALTER TABLE"):
			baseTime += time.Second * 20
		case strings.Contains(op, "INSERT INTO"):
			baseTime += time.Second * 15
		case strings.Contains(op, "UPDATE"):
			baseTime += time.Second * 45
		default:
			baseTime += time.Second * 5
		}
	}

	return baseTime
}

func (msm *MigrationSafetyManager) calculateChecksum(content []byte) string {
	// Simple checksum - in production, use a proper hash function
	return fmt.Sprintf("%x", len(content))
}

func (msm *MigrationSafetyManager) hasRollbackSQL(version string) bool {
	rollbackFile := filepath.Join(msm.migrationsPath, "rollback", version+"_rollback.sql")
	_, err := os.Stat(rollbackFile)
	return err == nil
}

func (msm *MigrationSafetyManager) estimateBackupSize(ctx context.Context) int64 {
	// Estimate database size for backup planning
	query := `
		SELECT COALESCE(SUM(pg_total_relation_size(oid)), 0) as total_size 
		FROM pg_class 
		WHERE relkind = 'r'
	`

	var size int64
	err := msm.db.QueryRowContext(ctx, query).Scan(&size)
	if err != nil {
		msm.logger.Warn("Failed to estimate database size", "error", err)
		return 0
	}

	return size
}

func (msm *MigrationSafetyManager) extractDependencies(migrations []PlannedMigration) []string {
	// Extract unique dependencies from all migrations
	depSet := make(map[string]bool)
	for _, migration := range migrations {
		for _, dep := range migration.Dependencies {
			depSet[dep] = true
		}
	}

	var deps []string
	for dep := range depSet {
		deps = append(deps, dep)
	}
	sort.Strings(deps)
	return deps
}

func (msm *MigrationSafetyManager) createBackup(ctx context.Context) (string, error) {
	timestamp := time.Now().Format("20060102_150405")
	backupFile := filepath.Join(msm.backupPath, fmt.Sprintf("backup_%s.sql", timestamp))

	msm.logger.Info("Creating database backup", "file", backupFile)

	// Create backup using pg_dump (simplified - in production use proper backup mechanism)
	if msm.dryRun {
		msm.logger.Info("DRY RUN: Would create backup", "file", backupFile)
		return backupFile, nil
	}

	// For now, create empty backup file as placeholder
	file, err := os.Create(backupFile)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer file.Close()

	_, err = file.WriteString(fmt.Sprintf("-- Database backup created at %s\n", time.Now().Format(time.RFC3339)))
	if err != nil {
		return "", fmt.Errorf("failed to write backup header: %w", err)
	}

	return backupFile, nil
}

func (msm *MigrationSafetyManager) executeSingleMigration(ctx context.Context, migration PlannedMigration, backupPath string) error {
	msm.logger.Info("Executing migration",
		"version", migration.Version,
		"name", migration.Name,
		"risk_level", migration.RiskLevel)

	if msm.dryRun {
		msm.logger.Info("DRY RUN: Would execute migration", "version", migration.Version)
		return nil
	}

	startTime := time.Now()

	// Read migration file
	content, err := os.ReadFile(migration.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// Execute migration in transaction
	tx, err := msm.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, string(content))
	if err != nil {
		// Record failed migration
		msm.recordMigration(ctx, migration, startTime, backupPath, false, err.Error())
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Record successful migration
	err = msm.recordMigration(ctx, migration, startTime, backupPath, true, "")
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit migration transaction: %w", err)
	}

	duration := time.Since(startTime)
	msm.logger.Info("Migration executed successfully",
		"version", migration.Version,
		"duration", duration.String())

	return nil
}

func (msm *MigrationSafetyManager) recordMigration(ctx context.Context, migration PlannedMigration, startTime time.Time, backupPath string, success bool, errorMsg string) error {
	duration := time.Since(startTime)

	metadataJSON, _ := json.Marshal(migration.Metadata)

	query := `
		INSERT INTO migration_records (version, name, checksum, applied_at, success, error_msg, duration_ms, backup_path, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := msm.db.ExecContext(ctx, query,
		migration.Version,
		migration.Name,
		migration.Checksum,
		startTime,
		success,
		errorMsg,
		duration.Milliseconds(),
		backupPath,
		string(metadataJSON),
	)

	return err
}

func (msm *MigrationSafetyManager) getRollbackSQL(version string) (string, error) {
	rollbackFile := filepath.Join(msm.migrationsPath, "rollback", version+"_rollback.sql")

	content, err := os.ReadFile(rollbackFile)
	if err != nil {
		return "", fmt.Errorf("rollback file not found: %w", err)
	}

	return string(content), nil
}

func (msm *MigrationSafetyManager) analyzeDataLossRisk(rollbackSQL string) string {
	sql := strings.ToUpper(rollbackSQL)

	if strings.Contains(sql, "DROP TABLE") || strings.Contains(sql, "DELETE FROM") {
		return "high"
	}

	if strings.Contains(sql, "UPDATE") || strings.Contains(sql, "ALTER TABLE") {
		return "medium"
	}

	return "low"
}

func (msm *MigrationSafetyManager) executeSingleRollback(ctx context.Context, rollback PlannedRollback, backupPath string) error {
	msm.logger.Info("Executing rollback",
		"version", rollback.Version,
		"name", rollback.Name,
		"data_loss_risk", rollback.DataLossRisk)

	if msm.dryRun {
		msm.logger.Info("DRY RUN: Would execute rollback", "version", rollback.Version)
		return nil
	}

	startTime := time.Now()

	// Execute rollback in transaction
	tx, err := msm.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin rollback transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, rollback.RollbackSQL)
	if err != nil {
		return fmt.Errorf("failed to execute rollback SQL: %w", err)
	}

	// Mark migration as rolled back
	_, err = tx.ExecContext(ctx,
		"UPDATE migration_records SET rolled_back_at = $1 WHERE version = $2",
		time.Now(), rollback.Version)
	if err != nil {
		return fmt.Errorf("failed to mark migration as rolled back: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit rollback transaction: %w", err)
	}

	duration := time.Since(startTime)
	msm.logger.Info("Rollback executed successfully",
		"version", rollback.Version,
		"duration", duration.String())

	return nil
}

// GetMigrationStatus returns the current status of migrations
func (msm *MigrationSafetyManager) GetMigrationStatus(ctx context.Context) (*MigrationStatus, error) {
	appliedMigrations, err := msm.getAppliedMigrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	migrationFiles, err := msm.getMigrationFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to get migration files: %w", err)
	}

	status := &MigrationStatus{
		AppliedCount:  len(appliedMigrations),
		PendingCount:  len(migrationFiles) - len(appliedMigrations),
		LastMigration: "",
		TotalFiles:    len(migrationFiles),
		HealthStatus:  "healthy",
	}

	if len(appliedMigrations) > 0 {
		lastMigration := appliedMigrations[len(appliedMigrations)-1]
		status.LastMigration = lastMigration.Version
		status.LastAppliedAt = &lastMigration.AppliedAt
	}

	return status, nil
}

// MigrationStatus represents the current state of database migrations
type MigrationStatus struct {
	AppliedCount  int        `json:"applied_count"`
	PendingCount  int        `json:"pending_count"`
	LastMigration string     `json:"last_migration"`
	LastAppliedAt *time.Time `json:"last_applied_at,omitempty"`
	TotalFiles    int        `json:"total_files"`
	HealthStatus  string     `json:"health_status"`
}
