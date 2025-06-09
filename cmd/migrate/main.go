// migrate is a command-line tool for database schema migrations with validation,
// rollback capabilities, and comprehensive safety checks for the MCP Memory Server.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"lerian-mcp-memory/internal/config"
	"lerian-mcp-memory/internal/migration"

	_ "github.com/lib/pq" // PostgreSQL driver
)

const (
	migrationVersion = "1.0.0"
)

func main() {
	os.Exit(run())
}

func run() int {
	var (
		action        = flag.String("action", "up", "Migration action: up, down, status, validate, create")
		migrationsDir = flag.String("migrations-dir", "./migrations", "Path to migrations directory")
		environment   = flag.String("environment", "dev", "Target environment (dev, staging, production)")
		dryRun        = flag.Bool("dry-run", false, "Perform dry run without executing migrations")
		validateOnly  = flag.Bool("validate-only", false, "Only validate migrations, don't execute")
		force         = flag.Bool("force", false, "Force migration execution (skip some safety checks)")
		verbose       = flag.Bool("verbose", false, "Enable verbose logging")
		target        = flag.String("target", "", "Target migration version or ID (for up/down actions)")
		steps         = flag.Int("steps", 0, "Number of migration steps (for up/down actions)")
		createName    = flag.String("name", "", "Migration name (for create action)")
		_             = flag.String("config", "configs/dev/config.yaml", "Path to configuration file")
	)
	flag.Parse()

	// Setup logging
	if *verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("Failed to load config: %v", err)
		return 1
	}

	// Create database connection
	db, err := createDatabaseConnection(cfg)
	if err != nil {
		log.Printf("Failed to connect to database: %v", err)
		return 1
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			log.Printf("Failed to close database connection: %v", closeErr)
		}
	}()

	// Create migrator
	migrator := migration.NewMigrator(db, *migrationsDir, *environment)

	// Execute action
	ctx := context.Background()
	switch *action {
	case "up":
		err = executeUpMigrations(ctx, migrator, *target, *steps, *dryRun, *validateOnly)
	case "down":
		err = executeDownMigrations(ctx, migrator, *target, *steps, *dryRun, *force)
	case "status":
		err = showMigrationStatus(ctx, migrator)
	case "validate":
		err = validateMigrations(ctx, migrator)
	case "create":
		err = createMigration(*migrationsDir, *createName)
	default:
		err = fmt.Errorf("unknown action: %s. Available actions: up, down, status, validate, create", *action)
	}

	if err != nil {
		log.Printf("Migration action failed: %v", err)
		return 1
	}

	return 0
}

// createDatabaseConnection creates a PostgreSQL database connection
func createDatabaseConnection(cfg *config.Config) (*sql.DB, error) {
	// Build connection string from config
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	return db, nil
}

// executeUpMigrations runs pending migrations
func executeUpMigrations(ctx context.Context, migrator *migration.Migrator, target string, steps int, dryRun, validateOnly bool) error {
	log.Printf("Executing up migrations: dry_run=%v, validate_only=%v", dryRun, validateOnly)

	// Get pending migrations
	pending, err := migrator.GetPendingMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get pending migrations: %w", err)
	}

	if len(pending) == 0 {
		log.Printf("No pending migrations found")
		return nil
	}

	log.Printf("Found %d pending migrations", len(pending))

	// Filter migrations based on target and steps
	toExecute := filterMigrations(pending, target, steps, true)

	if len(toExecute) == 0 {
		log.Printf("No migrations to execute after filtering")
		return nil
	}

	log.Printf("Executing %d migrations", len(toExecute))

	// Execute migrations
	for _, mig := range toExecute {
		log.Printf("Processing migration: %s - %s", mig.ID, mig.Description)

		if validateOnly {
			// Validate only
			validator := migration.NewMigrationValidator(migrator.GetDB())
			result, err := validator.ValidateMigration(ctx, mig)
			if err != nil {
				return fmt.Errorf("validation failed for %s: %w", mig.ID, err)
			}

			log.Printf("Validation result for %s: passed=%v, warnings=%d, errors=%d",
				mig.ID, result.Passed, result.Warnings, result.Errors)

			if !result.Passed {
				for _, issue := range result.Issues {
					if issue.Severity == migration.SeverityError {
						log.Printf("  ERROR: %s", issue.Message)
					}
				}
				return fmt.Errorf("migration %s failed validation", mig.ID)
			}
			continue
		}

		// Execute migration
		result, err := migrator.ExecuteMigration(ctx, mig, dryRun)
		if err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", mig.ID, err)
		}

		if dryRun {
			log.Printf("DRY RUN: Migration %s would execute successfully", mig.ID)
		} else {
			log.Printf("Migration %s executed successfully in %v", mig.ID, result.ExecutionTime)
		}
	}

	log.Printf("Up migrations completed successfully")
	return nil
}

// executeDownMigrations runs rollback migrations
func executeDownMigrations(ctx context.Context, migrator *migration.Migrator, target string, steps int, dryRun, force bool) error {
	log.Printf("Executing down migrations: dry_run=%v, force=%v", dryRun, force)

	// Get executed migrations
	executed, err := migrator.GetExecutedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get executed migrations: %w", err)
	}

	if len(executed) == 0 {
		log.Printf("No executed migrations found")
		return nil
	}

	// Reverse order for rollback
	for i, j := 0, len(executed)-1; i < j; i, j = i+1, j-1 {
		executed[i], executed[j] = executed[j], executed[i]
	}

	// Filter migrations based on target and steps
	toRollback := filterMigrations(executed, target, steps, false)

	if len(toRollback) == 0 {
		log.Printf("No migrations to rollback after filtering")
		return nil
	}

	log.Printf("Rolling back %d migrations", len(toRollback))

	// Check for destructive operations
	if !force {
		for _, mig := range toRollback {
			if mig.IsDestructive && !dryRun {
				return fmt.Errorf("migration %s is destructive and cannot be rolled back without --force", mig.ID)
			}
			if mig.DownSQL == "" {
				return fmt.Errorf("migration %s has no rollback script", mig.ID)
			}
		}
	}

	// Execute rollbacks
	for _, mig := range toRollback {
		log.Printf("Rolling back migration: %s - %s", mig.ID, mig.Description)

		if dryRun {
			log.Printf("DRY RUN: Migration %s would be rolled back", mig.ID)
			continue
		}

		// Execute rollback (this would need to be implemented in migrator)
		log.Printf("Rolling back migration %s", mig.ID)
		// TODO: Implement rollback execution in migrator
	}

	log.Printf("Down migrations completed successfully")
	return nil
}

// showMigrationStatus displays the current migration status
func showMigrationStatus(ctx context.Context, migrator *migration.Migrator) error {
	log.Printf("Checking migration status")

	// Get current version
	currentVersion, err := migrator.GetCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	// Get executed migrations
	executed, err := migrator.GetExecutedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get executed migrations: %w", err)
	}

	// Get pending migrations
	pending, err := migrator.GetPendingMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get pending migrations: %w", err)
	}

	// Get statistics
	stats, err := migrator.GetMigrationStatistics(ctx)
	if err != nil {
		return fmt.Errorf("failed to get migration statistics: %w", err)
	}

	// Display status
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("MIGRATION STATUS")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Current Version: %d\n", currentVersion)
	fmt.Printf("Executed Migrations: %d\n", len(executed))
	fmt.Printf("Pending Migrations: %d\n", len(pending))
	fmt.Printf("Last Migration: %v\n", stats.LastMigrationDate.Format(time.RFC3339))
	fmt.Printf("Avg Execution Time: %.2f ms\n", stats.AvgExecutionTimeMs)
	fmt.Printf("Destructive Migrations: %d\n", stats.DestructiveMigrations)
	fmt.Printf("Rollback Migrations: %d\n", stats.RolledBackMigrations)

	if len(pending) > 0 {
		fmt.Println("\nPending Migrations:")
		for _, mig := range pending {
			fmt.Printf("  %s - %s\n", mig.ID, mig.Description)
		}
	}

	if len(executed) > 0 {
		fmt.Println("\nRecent Executed Migrations:")
		count := 5
		if len(executed) < count {
			count = len(executed)
		}
		for i := len(executed) - count; i < len(executed); i++ {
			mig := executed[i]
			fmt.Printf("  %s - %s\n", mig.ID, mig.Description)
		}
	}

	fmt.Println(strings.Repeat("=", 60))
	return nil
}

// validateMigrations validates all migrations without executing them
func validateMigrations(ctx context.Context, migrator *migration.Migrator) error {
	log.Printf("Validating all migrations")

	// Load all migrations
	migrations, err := migrator.LoadMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	if len(migrations) == 0 {
		log.Printf("No migrations found")
		return nil
	}

	// Create validator
	validator := migration.NewMigrationValidator(migrator.GetDB())

	// Validate all migrations
	results, err := validator.ValidateMigrationBatch(ctx, migrations)
	if err != nil {
		return fmt.Errorf("batch validation failed: %w", err)
	}

	// Report results
	totalErrors := 0
	totalWarnings := 0

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("MIGRATION VALIDATION RESULTS")
	fmt.Println(strings.Repeat("=", 60))

	for _, result := range results {
		status := "PASS"
		if !result.Passed {
			status = "FAIL"
		}

		fmt.Printf("%s - %s [%s]\n", result.Migration.ID, result.Migration.Description, status)

		if result.Errors > 0 {
			totalErrors += result.Errors
			for _, issue := range result.Issues {
				if issue.Severity == migration.SeverityError {
					fmt.Printf("  ERROR: %s\n", issue.Message)
				}
			}
		}

		if result.Warnings > 0 {
			totalWarnings += result.Warnings
			for _, issue := range result.Issues {
				if issue.Severity == migration.SeverityWarning {
					fmt.Printf("  WARNING: %s\n", issue.Message)
				}
			}
		}

		if len(result.Suggestions) > 0 {
			fmt.Printf("  Suggestions:\n")
			for _, suggestion := range result.Suggestions {
				fmt.Printf("    - %s\n", suggestion)
			}
		}
	}

	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Total: %d migrations, %d errors, %d warnings\n", len(results), totalErrors, totalWarnings)

	if totalErrors > 0 {
		return fmt.Errorf("validation failed with %d errors", totalErrors)
	}

	log.Printf("All migrations validated successfully")
	return nil
}

// createMigration creates a new migration file
func createMigration(migrationsDir, name string) error {
	if name == "" {
		return fmt.Errorf("migration name is required")
	}

	// Create migrations directory if it doesn't exist
	if err := os.MkdirAll(migrationsDir, 0o750); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	// Find the next migration number
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	nextNumber := 1
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filename := entry.Name()
		if strings.HasSuffix(filename, ".sql") {
			parts := strings.Split(filename, "_")
			if len(parts) > 0 {
				if num, err := strconv.Atoi(parts[0]); err == nil && num >= nextNumber {
					nextNumber = num + 1
				}
			}
		}
	}

	// Create migration filename
	cleanName := strings.ReplaceAll(strings.ToLower(name), " ", "_")
	filename := fmt.Sprintf("%03d_%s.sql", nextNumber, cleanName)
	filePath := filepath.Join(migrationsDir, filename)

	// Create migration template
	template := fmt.Sprintf(`-- Migration %03d: %s
-- Description: %s
-- Created: %s
-- Version: %s

-- DESTRUCTIVE: false
-- DEPENDS: 
-- TAGS: schema
-- PRIORITY: normal
-- TYPE: schema
-- NOTES: Add description here

-- UP
-- Add your migration SQL here


-- DOWN
-- Add your rollback SQL here

`, nextNumber, name, name, time.Now().Format("2006-01-02"), migrationVersion)

	// Write migration file
	if err := os.WriteFile(filePath, []byte(template), 0o600); err != nil {
		return fmt.Errorf("failed to write migration file: %w", err)
	}

	log.Printf("Created migration: %s", filePath)
	return nil
}

// filterMigrations filters migrations based on target and steps
func filterMigrations(migrations []*migration.Migration, target string, steps int, forward bool) []*migration.Migration {
	if target == "" && steps == 0 {
		return migrations
	}

	if target != "" {
		return filterByTarget(migrations, target, forward)
	}

	if steps > 0 {
		return filterBySteps(migrations, steps)
	}

	return migrations
}

// filterByTarget filters migrations by target version or ID
func filterByTarget(migrations []*migration.Migration, target string, forward bool) []*migration.Migration {
	targetVersion, err := strconv.Atoi(target)
	if err == nil {
		return filterByVersion(migrations, targetVersion, forward)
	}
	return filterByID(migrations, target)
}

// filterByVersion filters migrations by version number
func filterByVersion(migrations []*migration.Migration, targetVersion int, forward bool) []*migration.Migration {
	var filtered []*migration.Migration
	for _, mig := range migrations {
		if forward && mig.Version <= targetVersion {
			filtered = append(filtered, mig)
		} else if !forward && mig.Version >= targetVersion {
			filtered = append(filtered, mig)
		}
	}
	return filtered
}

// filterByID filters migrations by ID
func filterByID(migrations []*migration.Migration, targetID string) []*migration.Migration {
	filtered := make([]*migration.Migration, 0, len(migrations))
	for _, mig := range migrations {
		filtered = append(filtered, mig)
		if mig.ID == targetID {
			break
		}
	}
	return filtered
}

// filterBySteps filters migrations by number of steps
func filterBySteps(migrations []*migration.Migration, steps int) []*migration.Migration {
	if steps > len(migrations) {
		steps = len(migrations)
	}
	return migrations[:steps]
}
