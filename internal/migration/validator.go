// Package migration provides validation for database schema migrations.
package migration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"lerian-mcp-memory/pkg/types"
)

// ValidationRule represents a migration validation rule
type ValidationRule struct {
	Name        string
	Description string
	Severity    ValidationSeverity
	Pattern     *regexp.Regexp
	CheckFunc   func(ctx context.Context, migration *Migration, db *sql.DB) error
}

// ValidationSeverity represents the severity of a validation issue
type ValidationSeverity string

const (
	SeverityError   ValidationSeverity = "error"
	SeverityWarning ValidationSeverity = "warning"
	SeverityInfo    ValidationSeverity = "info"
)

// ValidationResult represents the result of migration validation
type ValidationResult struct {
	Migration   *Migration
	Passed      bool
	Issues      []ValidationIssue
	Duration    time.Duration
	Warnings    int
	Errors      int
	Suggestions []string
}

// ValidationIssue represents a specific validation issue
type ValidationIssue struct {
	Rule       string
	Severity   ValidationSeverity
	Message    string
	Line       int
	Column     int
	Suggestion string
	CanAutoFix bool
}

// MigrationValidator handles validation of database migrations
type MigrationValidator struct {
	db    *sql.DB
	rules []ValidationRule
}

// NewMigrationValidator creates a new migration validator
func NewMigrationValidator(db *sql.DB) *MigrationValidator {
	validator := &MigrationValidator{
		db:    db,
		rules: getDefaultValidationRules(),
	}
	return validator
}

// ValidateMigration performs comprehensive validation of a migration
func (v *MigrationValidator) ValidateMigration(ctx context.Context, migration *Migration) (*ValidationResult, error) {
	startTime := time.Now()

	result := &ValidationResult{
		Migration: migration,
		Passed:    true,
		Issues:    []ValidationIssue{},
		Warnings:  0,
		Errors:    0,
	}

	// Run all validation rules
	for _, rule := range v.rules {
		if err := v.runValidationRule(ctx, rule, migration, result); err != nil {
			return result, fmt.Errorf("validation rule %s failed: %w", rule.Name, err)
		}
	}

	// Additional context-aware validations
	if err := v.validateDependencies(ctx, migration, result); err != nil {
		return result, fmt.Errorf("dependency validation failed: %w", err)
	}

	v.validateSchemaChanges(ctx, migration, result)

	v.validatePerformanceImpact(ctx, migration, result)

	// Generate suggestions based on issues found
	result.Suggestions = v.generateSuggestions(result.Issues)

	// Set final status
	result.Passed = result.Errors == 0
	result.Duration = time.Since(startTime)

	return result, nil
}

// runValidationRule executes a single validation rule
func (v *MigrationValidator) runValidationRule(ctx context.Context, rule ValidationRule, migration *Migration, result *ValidationResult) error {
	// Pattern-based validation
	if rule.Pattern != nil {
		if err := v.validatePattern(rule, migration, result); err != nil {
			return err
		}
	}

	// Function-based validation
	if rule.CheckFunc != nil {
		if err := rule.CheckFunc(ctx, migration, v.db); err != nil {
			issue := ValidationIssue{
				Rule:     rule.Name,
				Severity: rule.Severity,
				Message:  err.Error(),
			}
			result.Issues = append(result.Issues, issue)

			switch rule.Severity {
			case SeverityError:
				result.Errors++
			case SeverityWarning:
				result.Warnings++
			}
		}
	}

	return nil
}

// validatePattern performs pattern-based validation
func (v *MigrationValidator) validatePattern(rule ValidationRule, migration *Migration, result *ValidationResult) error {
	sqlQuery := strings.ToUpper(migration.UpSQL)

	if rule.Pattern.MatchString(sqlQuery) {
		lines := strings.Split(migration.UpSQL, "\n")
		for lineNum, line := range lines {
			if rule.Pattern.MatchString(strings.ToUpper(line)) {
				issue := ValidationIssue{
					Rule:     rule.Name,
					Severity: rule.Severity,
					Message:  fmt.Sprintf("Line %d: %s", lineNum+1, rule.Description),
					Line:     lineNum + 1,
				}

				result.Issues = append(result.Issues, issue)

				switch rule.Severity {
				case SeverityError:
					result.Errors++
				case SeverityWarning:
					result.Warnings++
				}
				break
			}
		}
	}

	return nil
}

// validateDependencies checks migration dependencies
func (v *MigrationValidator) validateDependencies(ctx context.Context, migration *Migration, result *ValidationResult) error {
	for _, depID := range migration.Dependencies {
		exists, err := v.checkMigrationExists(ctx, depID)
		if err != nil {
			return fmt.Errorf("failed to check dependency %s: %w", depID, err)
		}

		if !exists {
			issue := ValidationIssue{
				Rule:     "dependency_check",
				Severity: SeverityError,
				Message:  "Dependency migration not found: " + depID,
			}
			result.Issues = append(result.Issues, issue)
			result.Errors++
		}
	}

	return nil
}

// validateSchemaChanges analyzes schema modification impact
func (v *MigrationValidator) validateSchemaChanges(ctx context.Context, migration *Migration, result *ValidationResult) {
	_ = ctx // unused parameter, kept for potential future context-aware validation
	sqlQuery := strings.ToUpper(migration.UpSQL)

	// Check for potentially risky operations
	riskPatterns := map[string]string{
		`ALTER TABLE .+ DROP COLUMN`:          "Dropping columns can break existing code",
		`DROP TABLE`:                          "Dropping tables is irreversible",
		`ALTER TABLE .+ ALTER COLUMN .+ TYPE`: "Changing column types can cause data loss",
		`DELETE FROM .+ WHERE`:                "DELETE operations should be carefully reviewed",
		`TRUNCATE`:                            "TRUNCATE operations remove all data",
	}

	for pattern, warning := range riskPatterns {
		matched, err := regexp.MatchString(pattern, sqlQuery)
		if err != nil {
			continue
		}

		if matched {
			v.processMatchedPattern(pattern, warning, migration, result)
		}
	}
}

// processMatchedPattern processes a matched risk pattern and adds appropriate issues
func (v *MigrationValidator) processMatchedPattern(pattern, warning string, migration *Migration, result *ValidationResult) {
	severity := SeverityWarning

	if v.isDestructivePattern(pattern) {
		severity = SeverityError
		if !migration.IsDestructive {
			v.addDestructiveOperationIssue(result)
		}
	}

	v.addSchemaRiskIssue(result, severity, warning)
}

// isDestructivePattern checks if a pattern represents a destructive operation
func (v *MigrationValidator) isDestructivePattern(pattern string) bool {
	return strings.Contains(pattern, "DROP") || strings.Contains(pattern, "TRUNCATE")
}

// addDestructiveOperationIssue adds an issue for unmarked destructive operations
func (v *MigrationValidator) addDestructiveOperationIssue(result *ValidationResult) {
	issue := ValidationIssue{
		Rule:     "destructive_operation",
		Severity: SeverityError,
		Message:  "Destructive operation found but migration not marked as destructive",
	}
	result.Issues = append(result.Issues, issue)
	result.Errors++
}

// addSchemaRiskIssue adds a schema risk issue with the given severity
func (v *MigrationValidator) addSchemaRiskIssue(result *ValidationResult, severity ValidationSeverity, warning string) {
	issue := ValidationIssue{
		Rule:     "schema_risk",
		Severity: severity,
		Message:  warning,
	}
	result.Issues = append(result.Issues, issue)

	if severity == SeverityError {
		result.Errors++
	} else {
		result.Warnings++
	}
}

// validatePerformanceImpact checks for potential performance issues
func (v *MigrationValidator) validatePerformanceImpact(ctx context.Context, migration *Migration, result *ValidationResult) {
	_ = ctx // unused parameter, kept for potential future context-aware validation
	sqlQuery := strings.ToUpper(migration.UpSQL)

	// Check for operations that might be slow on large tables
	performancePatterns := map[string]string{
		`ALTER TABLE .+ ADD COLUMN .+ NOT NULL`: "Adding NOT NULL columns to large tables can be slow",
		`CREATE INDEX(?:\s+CONCURRENTLY)?\s+`:   "Index creation can be slow on large tables",
		`ALTER TABLE .+ ADD CONSTRAINT`:         "Adding constraints can be slow on large tables",
	}

	for pattern, warning := range performancePatterns {
		matched, err := regexp.MatchString(pattern, sqlQuery)
		if err != nil {
			continue
		}

		if matched {
			// Check if CONCURRENTLY is used for index creation
			if strings.Contains(pattern, "INDEX") && !strings.Contains(sqlQuery, "CONCURRENTLY") {
				issue := ValidationIssue{
					Rule:       "performance_impact",
					Severity:   SeverityWarning,
					Message:    "Consider using CREATE INDEX CONCURRENTLY for large tables",
					Suggestion: "Add CONCURRENTLY keyword to avoid table locking",
					CanAutoFix: true,
				}
				result.Issues = append(result.Issues, issue)
				result.Warnings++
			} else {
				issue := ValidationIssue{
					Rule:     "performance_impact",
					Severity: SeverityInfo,
					Message:  warning,
				}
				result.Issues = append(result.Issues, issue)
			}
		}
	}
}

// checkMigrationExists verifies if a migration has been executed
func (v *MigrationValidator) checkMigrationExists(ctx context.Context, migrationID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM schema_migrations 
			WHERE migration_id = $1 AND is_rolled_back = false
		)`

	var exists bool
	err := v.db.QueryRowContext(ctx, query, migrationID).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// generateSuggestions creates helpful suggestions based on validation issues
func (v *MigrationValidator) generateSuggestions(issues []ValidationIssue) []string {
	suggestions := []string{}

	hasDestructive := false
	hasPerformance := false
	hasRollback := false

	for _, issue := range issues {
		if issue.Severity == SeverityError {
			if issue.Rule == "destructive_operation" {
				hasDestructive = true
			}
			if issue.Rule == "no_rollback" {
				hasRollback = true
			}
		}
		if issue.Rule == "performance_impact" {
			hasPerformance = true
		}

		if issue.CanAutoFix && issue.Suggestion != "" {
			suggestions = append(suggestions, issue.Suggestion)
		}
	}

	if hasDestructive {
		suggestions = append(suggestions, "Mark migration as destructive using -- DESTRUCTIVE: true comment")
	}

	if hasRollback {
		suggestions = append(suggestions, "Add -- DOWN section with rollback SQL")
	}

	if hasPerformance {
		suggestions = append(suggestions, "Consider running during maintenance window", "Test migration on production-sized data first")
	}

	return suggestions
}

// ValidateMigrationBatch validates a batch of migrations
func (v *MigrationValidator) ValidateMigrationBatch(ctx context.Context, migrations []*Migration) ([]*ValidationResult, error) {
	results := make([]*ValidationResult, len(migrations))

	for i, migration := range migrations {
		result, err := v.ValidateMigration(ctx, migration)
		if err != nil {
			return nil, fmt.Errorf("failed to validate migration %s: %w", migration.ID, err)
		}
		results[i] = result
	}

	// Additional batch-level validations
	if err := v.validateBatchConsistency(ctx, migrations, results); err != nil {
		return results, fmt.Errorf("batch consistency validation failed: %w", err)
	}

	return results, nil
}

// validateBatchConsistency checks for issues across multiple migrations
func (v *MigrationValidator) validateBatchConsistency(ctx context.Context, migrations []*Migration, results []*ValidationResult) error {
	_ = ctx // unused parameter, kept for potential future context-aware validation
	// Check for dependency cycles
	if err := v.checkDependencyCycles(migrations); err != nil {
		return err
	}

	// Check for conflicting changes
	if err := v.checkConflictingChanges(migrations, results); err != nil {
		return err
	}

	return nil
}

// checkDependencyCycles detects circular dependencies
func (v *MigrationValidator) checkDependencyCycles(migrations []*Migration) error {
	// Build dependency graph
	graph := make(map[string][]string)
	for _, migration := range migrations {
		graph[migration.ID] = migration.Dependencies
	}

	// Check for cycles using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for migrationID := range graph {
		if !visited[migrationID] {
			if v.hasCycle(graph, migrationID, visited, recStack) {
				return fmt.Errorf("circular dependency detected involving migration: %s", migrationID)
			}
		}
	}

	return nil
}

// hasCycle performs DFS to detect cycles in dependency graph
func (v *MigrationValidator) hasCycle(graph map[string][]string, node string, visited, recStack map[string]bool) bool {
	visited[node] = true
	recStack[node] = true

	for _, dep := range graph[node] {
		if !visited[dep] {
			if v.hasCycle(graph, dep, visited, recStack) {
				return true
			}
		} else if recStack[dep] {
			return true
		}
	}

	recStack[node] = false
	return false
}

// checkConflictingChanges identifies potentially conflicting operations
func (v *MigrationValidator) checkConflictingChanges(migrations []*Migration, results []*ValidationResult) error {
	// Track table operations across migrations
	tableOps := make(map[string][]string)

	for i, migration := range migrations {
		if results[i].Errors > 0 {
			continue // Skip migrations with errors
		}

		// Extract table names from SQL
		tables := v.extractTableNames(migration.UpSQL)
		for _, table := range tables {
			tableOps[table] = append(tableOps[table], migration.ID)
		}
	}

	// Check for potential conflicts
	for table, migrationIDs := range tableOps {
		if len(migrationIDs) > 1 {
			// Multiple migrations affecting same table - check for conflicts
			for _, result := range results {
				if contains(migrationIDs, result.Migration.ID) {
					issue := ValidationIssue{
						Rule:     "table_conflict",
						Severity: SeverityWarning,
						Message:  fmt.Sprintf("Multiple migrations affect table '%s': %v", table, migrationIDs),
					}
					result.Issues = append(result.Issues, issue)
					result.Warnings++
				}
			}
		}
	}

	return nil
}

// extractTableNames extracts table names from SQL
func (v *MigrationValidator) extractTableNames(sqlStatement string) []string {
	tables := []string{}

	// Simple regex patterns for common table operations
	patterns := []string{
		`CREATE TABLE\s+(\w+)`,
		`ALTER TABLE\s+(\w+)`,
		`DROP TABLE\s+(\w+)`,
		`INSERT INTO\s+(\w+)`,
		`UPDATE\s+(\w+)`,
		`DELETE FROM\s+(\w+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(`(?i)` + pattern)
		matches := re.FindAllStringSubmatch(sqlStatement, -1)
		for _, match := range matches {
			if len(match) > 1 {
				tables = append(tables, strings.ToLower(match[1]))
			}
		}
	}

	return tables
}

// GetValidationStatistics returns validation statistics
func (v *MigrationValidator) GetValidationStatistics(ctx context.Context, environment string) (*types.ValidationStatistics, error) {
	query := `
		SELECT 
			COUNT(*) as total_validations,
			COUNT(*) FILTER (WHERE validation_status = 'passed') as passed_validations,
			COUNT(*) FILTER (WHERE validation_status = 'failed') as failed_validations,
			COUNT(*) FILTER (WHERE validation_errors != '[]') as validations_with_errors,
			AVG(execution_time_ms) as avg_validation_time_ms
		FROM schema_migrations 
		WHERE environment = $1`

	stats := &types.ValidationStatistics{}
	err := v.db.QueryRowContext(ctx, query, environment).Scan(
		&stats.TotalValidations,
		&stats.PassedValidations,
		&stats.FailedValidations,
		&stats.ValidationsWithErrors,
		&stats.AvgValidationTimeMs,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get validation statistics: %w", err)
	}

	return stats, nil
}

// getDefaultValidationRules returns the default set of validation rules
func getDefaultValidationRules() []ValidationRule {
	return []ValidationRule{
		{
			Name:        "sql_injection_check",
			Description: "Check for potential SQL injection vulnerabilities",
			Severity:    SeverityError,
			Pattern:     regexp.MustCompile(`(?i)(union\s+select|;\s*drop|;\s*delete|;\s*update.*where\s+1=1)`),
		},
		{
			Name:        "reserved_keywords",
			Description: "Check for use of reserved SQL keywords as identifiers",
			Severity:    SeverityWarning,
			CheckFunc: func(ctx context.Context, migration *Migration, db *sql.DB) error {
				// Check for reserved keywords used as identifiers
				keywords := []string{"order", "group", "select", "from", "where", "table", "index", "constraint"}
				allowedAfter := []string{"by", "from", "into", "table", "if"}
				sqlContent := strings.ToLower(migration.UpSQL)

				for _, keyword := range keywords {
					pattern := regexp.MustCompile(`\b` + keyword + `\b\s+(\w+)`)
					if matches := pattern.FindAllStringSubmatch(sqlContent, -1); matches != nil {
						for _, match := range matches {
							nextWord := match[1]
							isAllowed := false
							for _, allowed := range allowedAfter {
								if nextWord == allowed {
									isAllowed = true
									break
								}
							}
							if !isAllowed {
								return errors.New("potential use of reserved keyword '" + keyword + "' as identifier")
							}
						}
					}
				}
				return nil
			},
		},
		{
			Name:        "missing_if_exists",
			Description: "DROP statements should use IF EXISTS for safety",
			Severity:    SeverityWarning,
			CheckFunc: func(ctx context.Context, migration *Migration, db *sql.DB) error {
				// Check for DROP statements without IF EXISTS
				dropPattern := regexp.MustCompile(`(?i)drop\s+(table|index|constraint|function)\s+(\w+)`)
				ifExistsPattern := regexp.MustCompile(`(?i)drop\s+(table|index|constraint|function)\s+if\s+exists`)

				migrationSQL := migration.UpSQL
				if dropPattern.MatchString(migrationSQL) && !ifExistsPattern.MatchString(migrationSQL) {
					return errors.New("DROP statement should use IF EXISTS for safety")
				}
				return nil
			},
		},
		{
			Name:        "concurrent_index",
			Description: "CREATE INDEX should use CONCURRENTLY to avoid locking",
			Severity:    SeverityInfo,
			CheckFunc: func(ctx context.Context, migration *Migration, db *sql.DB) error {
				// Check for CREATE INDEX without CONCURRENTLY
				createIndexPattern := regexp.MustCompile(`(?i)create\s+index`)
				concurrentlyPattern := regexp.MustCompile(`(?i)create\s+index\s+concurrently`)

				migrationSQL := migration.UpSQL
				if createIndexPattern.MatchString(migrationSQL) && !concurrentlyPattern.MatchString(migrationSQL) {
					return errors.New("CREATE INDEX should use CONCURRENTLY to avoid locking")
				}
				return nil
			},
		},
	}
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
