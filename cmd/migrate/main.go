// Package main provides the database migration CLI utility with safety mechanisms
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"lerian-mcp-memory/internal/config"
	"lerian-mcp-memory/internal/logging"
	"lerian-mcp-memory/internal/migration"

	_ "github.com/lib/pq" // PostgreSQL driver
)

func main() {
	os.Exit(run())
}

func run() int {
	var (
		configFile    = flag.String("config", "", "Path to configuration file")
		migrationsDir = flag.String("migrations", "./migrations", "Path to migrations directory")
		backupDir     = flag.String("backup", "./backups", "Path to backup directory")
		command       = flag.String("command", "status", "Command to execute: status, plan, migrate, rollback")
		targetVersion = flag.String("target", "", "Target version for rollback")
		dryRun        = flag.Bool("dry-run", false, "Execute in dry run mode")
		force         = flag.Bool("force", false, "Force execution without confirmation")
		verbose       = flag.Bool("verbose", false, "Enable verbose logging")
	)
	flag.Parse()

	// Initialize logger
	logger := logging.NewEnhancedLogger("migrate")
	if *verbose {
		// TODO: Implement log level setting if needed by EnhancedLogger
		fmt.Println("Verbose logging enabled")
	}

	// Load configuration
	cfg, err := loadConfig(*configFile)
	if err != nil {
		logger.Fatal("Failed to load configuration", "error", err)
		return 1
	}

	// Connect to database
	db, err := connectToDatabase(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to database", "error", err)
		return 1
	}
	defer func() { _ = db.Close() }()

	// Create migration safety manager
	safetyManager := migration.NewMigrationSafetyManager(db, *migrationsDir, *backupDir, logger)
	safetyManager.SetDryRun(*dryRun)

	// Initialize migration infrastructure
	ctx := context.Background()
	if err := safetyManager.Initialize(ctx); err != nil {
		logger.Fatal("Failed to initialize migration infrastructure", "error", err)
		return 1
	}

	// Execute command
	switch *command {
	case "status":
		err = executeStatus(ctx, safetyManager, logger)
	case "plan":
		err = executePlan(ctx, safetyManager, logger)
	case "migrate":
		err = executeMigrate(ctx, safetyManager, logger, *force, *dryRun)
	case "rollback":
		err = executeRollback(ctx, safetyManager, logger, *targetVersion, *force, *dryRun)
	default:
		logger.Fatal("Unknown command", "command", *command)
		return 1
	}

	if err != nil {
		logger.Error("Command execution failed", "error", err.Error())
		logger.Fatal("Migration failed")
		return 1
	}

	return 0
}

func loadConfig(configFile string) (*config.Config, error) {
	if configFile != "" {
		// Load from specified file (implementation depends on your config system)
		return config.LoadConfig()
	}
	return config.LoadConfig()
}

func connectToDatabase(cfg *config.Config) (*sql.DB, error) {
	dsn := "host=" + cfg.Database.Host + " port=" + strconv.Itoa(cfg.Database.Port) + " user=" + cfg.Database.User + " password=" + cfg.Database.Password + " dbname=" + cfg.Database.Name + " sslmode=" + cfg.Database.SSLMode

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, errors.New("failed to open database connection: " + err.Error())
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.Database.ConnMaxIdleTime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, errors.New("failed to ping database: " + err.Error())
	}

	return db, nil
}

func executeStatus(ctx context.Context, safetyManager *migration.MigrationSafetyManager, logger *logging.EnhancedLogger) error {
	logger.Info("Getting migration status")

	status, err := safetyManager.GetMigrationStatus(ctx)
	if err != nil {
		return errors.New("failed to get migration status: " + err.Error())
	}

	fmt.Printf("Migration Status:\n")
	fmt.Printf("  Applied migrations: %d\n", status.AppliedCount)
	fmt.Printf("  Pending migrations: %d\n", status.PendingCount)
	fmt.Printf("  Total migration files: %d\n", status.TotalFiles)
	fmt.Printf("  Last migration: %s\n", status.LastMigration)
	if status.LastAppliedAt != nil {
		fmt.Printf("  Last applied at: %s\n", status.LastAppliedAt.Format(time.RFC3339))
	}
	fmt.Printf("  Health status: %s\n", status.HealthStatus)

	return nil
}

func executePlan(ctx context.Context, safetyManager *migration.MigrationSafetyManager, logger *logging.EnhancedLogger) error {
	logger.Info("Creating migration plan")

	plan, err := safetyManager.PlanMigration(ctx)
	if err != nil {
		return errors.New("failed to create migration plan: " + err.Error())
	}

	fmt.Printf("Migration Plan:\n")
	fmt.Printf("  Total migrations: %d\n", plan.TotalCount)
	fmt.Printf("  Estimated time: %s\n", plan.EstimatedTime.String())
	fmt.Printf("  Risk level: %s\n", plan.RiskLevel)
	fmt.Printf("  Estimated backup size: %d bytes\n", plan.BackupSize)

	if len(plan.Warnings) > 0 {
		fmt.Printf("  Warnings:\n")
		for _, warning := range plan.Warnings {
			fmt.Printf("    - %s\n", warning)
		}
	}

	if len(plan.Dependencies) > 0 {
		fmt.Printf("  Dependencies:\n")
		for _, dep := range plan.Dependencies {
			fmt.Printf("    - %s\n", dep)
		}
	}

	fmt.Printf("\nMigrations to execute:\n")
	for i := range plan.Migrations {
		migrationEntry := &plan.Migrations[i]
		fmt.Printf("  %d. %s (%s)\n", i+1, migrationEntry.Name, migrationEntry.Version)
		fmt.Printf("     Risk: %s, Time: %s, Rollback: %t\n",
			migrationEntry.RiskLevel, migrationEntry.EstimatedTime.String(), migrationEntry.HasRollback)
		if len(migrationEntry.Operations) > 0 {
			fmt.Printf("     Operations: %v\n", migrationEntry.Operations)
		}
	}

	// Output plan as JSON for programmatic use
	if os.Getenv("OUTPUT_JSON") == "true" {
		planJSON, _ := json.MarshalIndent(plan, "", "  ")
		fmt.Printf("\nJSON Plan:\n%s\n", string(planJSON))
	}

	return nil
}

func executeMigrate(ctx context.Context, safetyManager *migration.MigrationSafetyManager, logger *logging.EnhancedLogger, force, dryRun bool) error {
	logger.Info("Executing migration", "dry_run", dryRun, "force", force)

	// Create migration plan
	plan, err := safetyManager.PlanMigration(ctx)
	if err != nil {
		return errors.New("failed to create migration plan: " + err.Error())
	}

	if plan.TotalCount == 0 {
		fmt.Println("No migrations to execute")
		return nil
	}

	// Show plan summary
	fmt.Printf("Migration Plan Summary:\n")
	fmt.Printf("  Migrations: %d\n", plan.TotalCount)
	fmt.Printf("  Estimated time: %s\n", plan.EstimatedTime.String())
	fmt.Printf("  Risk level: %s\n", plan.RiskLevel)

	// Confirmation prompt (unless force or dry run)
	if !force && !dryRun {
		fmt.Printf("\nProceed with migration? [y/N]: ")
		var response string
		_, _ = fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Migration cancelled")
			return nil
		}
	}

	// Configure safety settings
	safetyConfig := migration.SafetyConfig{
		EnableBackups:       true,
		BackupBeforeMigrate: true,
		MaxRollbackTime:     24 * time.Hour,
		RequireConfirmation: !force,
		DryRunFirst:         false,
		ParallelSafe:        false,
	}

	// Execute migration
	err = safetyManager.ExecuteMigrationPlan(ctx, plan, safetyConfig)
	if err != nil {
		return errors.New("migration execution failed: " + err.Error())
	}

	if dryRun {
		fmt.Println("DRY RUN: Migration would have been executed successfully")
	} else {
		fmt.Println("Migration executed successfully")
	}

	return nil
}

func executeRollback(ctx context.Context, safetyManager *migration.MigrationSafetyManager, logger *logging.EnhancedLogger, targetVersion string, force, dryRun bool) error {
	if targetVersion == "" {
		return errors.New("target version is required for rollback")
	}

	logger.Info("Executing rollback", "target_version", targetVersion, "dry_run", dryRun, "force", force)

	// Create rollback plan
	plan, err := safetyManager.PlanRollback(ctx, targetVersion)
	if err != nil {
		return errors.New("failed to create rollback plan: " + err.Error())
	}

	if plan.TotalCount == 0 {
		fmt.Printf("No rollbacks needed to reach version " + targetVersion + "\n")
		return nil
	}

	// Show rollback plan summary
	fmt.Printf("Rollback Plan Summary:\n")
	fmt.Printf("  Target version: %s\n", plan.TargetVersion)
	fmt.Printf("  Rollbacks: %d\n", plan.TotalCount)
	fmt.Printf("  Estimated time: %s\n", plan.EstimatedTime.String())
	fmt.Printf("  Data loss risk: %s\n", plan.DataLossRisk)

	if len(plan.Warnings) > 0 {
		fmt.Printf("  Warnings:\n")
		for _, warning := range plan.Warnings {
			fmt.Printf("    - %s\n", warning)
		}
	}

	fmt.Printf("\nRollbacks to execute:\n")
	for i, rollback := range plan.Migrations {
		fmt.Printf("  %d. %s (%s) - Risk: %s\n",
			i+1, rollback.Name, rollback.Version, rollback.DataLossRisk)
	}

	// Special confirmation for rollbacks (unless force or dry run)
	if !force && !dryRun {
		fmt.Printf("\nWARNING: Rollback may cause data loss (Risk: %s)\n", plan.DataLossRisk)
		fmt.Printf("Proceed with rollback? [y/N]: ")
		var response string
		_, _ = fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Rollback cancelled")
			return nil
		}
	}

	// Configure safety settings for rollback
	safetyConfig := migration.SafetyConfig{
		EnableBackups:       true,
		BackupBeforeMigrate: true,
		MaxRollbackTime:     24 * time.Hour,
		RequireConfirmation: !force,
		DryRunFirst:         false,
		ParallelSafe:        false,
	}

	// Execute rollback
	err = safetyManager.ExecuteRollback(ctx, plan, safetyConfig)
	if err != nil {
		return errors.New("rollback execution failed: " + err.Error())
	}

	if dryRun {
		fmt.Printf("DRY RUN: Rollback to version " + targetVersion + " would have been executed successfully\n")
	} else {
		fmt.Printf("Rollback to version " + targetVersion + " executed successfully\n")
	}

	return nil
}
