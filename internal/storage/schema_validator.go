// Package storage provides schema validation functionality for database operations.
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"lerian-mcp-memory/pkg/types"
)

// SchemaValidator provides validation functionality for database schema compliance
type SchemaValidator struct {
	db *sql.DB
}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator(db *sql.DB) *SchemaValidator {
	return &SchemaValidator{db: db}
}

// ValidationResult represents the result of schema validation
type ValidationResult struct {
	IsValid      bool                `json:"is_valid"`
	Errors       []ValidationError   `json:"errors,omitempty"`
	Warnings     []ValidationWarning `json:"warnings,omitempty"`
	Suggestions  []string            `json:"suggestions,omitempty"`
	Score        float64             `json:"score"` // Overall validation score 0.0-1.0
	CheckedItems []SchemaCheckItem   `json:"checked_items"`
	Summary      ValidationSummary   `json:"summary"`
}

// ValidationError represents a schema validation error
type ValidationError struct {
	Component  string `json:"component"` // table, index, constraint, etc.
	Type       string `json:"type"`      // missing, invalid, constraint_violation
	Message    string `json:"message"`
	Severity   string `json:"severity"` // critical, major, minor
	Code       string `json:"code"`
	Suggestion string `json:"suggestion,omitempty"`
}

// Error implements the error interface
func (ve *ValidationError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", ve.Code, ve.Component, ve.Message)
}

// ValidationWarning represents a schema validation warning
type ValidationWarning struct {
	Component  string `json:"component"`
	Type       string `json:"type"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion"`
	Code       string `json:"code"`
}

// SchemaCheckItem represents an individual schema check
type SchemaCheckItem struct {
	Name        string `json:"name"`
	Type        string `json:"type"`   // table, index, constraint, trigger
	Status      string `json:"status"` // pass, fail, warning
	Expected    string `json:"expected"`
	Actual      string `json:"actual"`
	Description string `json:"description"`
}

// ValidationSummary provides a summary of validation results
type ValidationSummary struct {
	TotalChecks    int       `json:"total_checks"`
	PassedChecks   int       `json:"passed_checks"`
	FailedChecks   int       `json:"failed_checks"`
	WarningChecks  int       `json:"warning_checks"`
	CriticalErrors int       `json:"critical_errors"`
	PassPercentage float64   `json:"pass_percentage"`
	SchemaVersion  string    `json:"schema_version"`
	LastValidated  time.Time `json:"last_validated"`
}

// ValidateSchema performs comprehensive schema validation
func (sv *SchemaValidator) ValidateSchema(ctx context.Context) (*ValidationResult, error) {
	result := &ValidationResult{
		IsValid:      true,
		Errors:       []ValidationError{},
		Warnings:     []ValidationWarning{},
		Suggestions:  []string{},
		CheckedItems: []SchemaCheckItem{},
		Summary: ValidationSummary{
			LastValidated: time.Now(),
		},
	}

	// 1. Validate core tables exist
	if err := sv.validateCoreTables(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to validate core tables: %w", err)
	}

	// 2. Validate table structure
	if err := sv.validateTableStructure(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to validate table structure: %w", err)
	}

	// 3. Validate indexes
	if err := sv.validateIndexes(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to validate indexes: %w", err)
	}

	// 4. Validate constraints
	if err := sv.validateConstraints(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to validate constraints: %w", err)
	}

	// 5. Validate triggers
	if err := sv.validateTriggers(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to validate triggers: %w", err)
	}

	// 6. Validate data integrity
	if err := sv.validateDataIntegrity(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to validate data integrity: %w", err)
	}

	// 7. Validate performance
	if err := sv.validatePerformance(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to validate performance: %w", err)
	}

	// Calculate final score and summary
	sv.calculateValidationScore(result)

	return result, nil
}

// validateCoreTables checks that all required tables exist
func (sv *SchemaValidator) validateCoreTables(ctx context.Context, result *ValidationResult) error {
	requiredTables := []string{
		"tasks",
		"prds",
		"task_patterns",
		"task_templates",
		"task_effort_breakdown",
		"task_quality_issues",
		"task_audit_log",
	}

	for _, tableName := range requiredTables {
		exists, err := sv.tableExists(ctx, tableName)
		if err != nil {
			return err
		}

		checkItem := SchemaCheckItem{
			Name:        tableName,
			Type:        "table",
			Expected:    "exists",
			Description: fmt.Sprintf("Core table '%s' should exist", tableName),
		}

		if exists {
			checkItem.Status = "pass"
			checkItem.Actual = "exists"
		} else {
			checkItem.Status = "fail"
			checkItem.Actual = "missing"
			result.IsValid = false
			result.Errors = append(result.Errors, ValidationError{
				Component:  tableName,
				Type:       "missing",
				Message:    fmt.Sprintf("Required table '%s' is missing", tableName),
				Severity:   "critical",
				Code:       "MISSING_TABLE",
				Suggestion: fmt.Sprintf("Run migration to create table '%s'", tableName),
			})
		}

		result.CheckedItems = append(result.CheckedItems, checkItem)
	}

	return nil
}

// validateTableStructure checks table column structure
func (sv *SchemaValidator) validateTableStructure(ctx context.Context, result *ValidationResult) error {
	// Define expected structure for tasks table
	expectedTasksColumns := map[string]string{
		"id":                  "uuid",
		"title":               "character varying",
		"description":         "text",
		"content":             "text",
		"type":                "USER-DEFINED", // ENUM
		"status":              "USER-DEFINED", // ENUM
		"priority":            "USER-DEFINED", // ENUM
		"complexity":          "USER-DEFINED", // ENUM
		"assignee":            "character varying",
		"repository":          "character varying",
		"session_id":          "character varying",
		"created_at":          "timestamp with time zone",
		"updated_at":          "timestamp with time zone",
		"started_at":          "timestamp with time zone",
		"completed_at":        "timestamp with time zone",
		"due_date":            "timestamp with time zone",
		"estimated_hours":     "numeric",
		"complexity_score":    "numeric",
		"quality_score":       "numeric",
		"parent_task_id":      "uuid",
		"tags":                "jsonb",
		"dependencies":        "jsonb",
		"blocks":              "jsonb",
		"acceptance_criteria": "jsonb",
		"metadata":            "jsonb",
		"prd_id":              "uuid",
		"source_prd_id":       "uuid",
		"source_section":      "character varying",
		"deleted_at":          "timestamp with time zone",
		"search_vector":       "tsvector",
	}

	actualColumns, err := sv.getTableColumns(ctx, "tasks")
	if err != nil {
		return err
	}

	// Check for missing columns
	for expectedCol, expectedType := range expectedTasksColumns {
		checkItem := SchemaCheckItem{
			Name:        fmt.Sprintf("tasks.%s", expectedCol),
			Type:        "column",
			Expected:    expectedType,
			Description: fmt.Sprintf("Column '%s' should exist with type '%s'", expectedCol, expectedType),
		}

		if actualType, exists := actualColumns[expectedCol]; exists {
			checkItem.Status = "pass"
			checkItem.Actual = actualType

			// Check type compatibility
			if !sv.isTypeCompatible(expectedType, actualType) {
				checkItem.Status = "warning"
				result.Warnings = append(result.Warnings, ValidationWarning{
					Component:  fmt.Sprintf("tasks.%s", expectedCol),
					Type:       "type_mismatch",
					Message:    fmt.Sprintf("Column '%s' has type '%s', expected '%s'", expectedCol, actualType, expectedType),
					Suggestion: "Consider type migration if necessary",
					Code:       "TYPE_MISMATCH",
				})
			}
		} else {
			checkItem.Status = "fail"
			checkItem.Actual = "missing"
			result.IsValid = false
			result.Errors = append(result.Errors, ValidationError{
				Component:  fmt.Sprintf("tasks.%s", expectedCol),
				Type:       "missing",
				Message:    fmt.Sprintf("Required column '%s' is missing from tasks table", expectedCol),
				Severity:   "major",
				Code:       "MISSING_COLUMN",
				Suggestion: fmt.Sprintf("Add column '%s' with type '%s'", expectedCol, expectedType),
			})
		}

		result.CheckedItems = append(result.CheckedItems, checkItem)
	}

	return nil
}

// validateIndexes checks that required indexes exist
func (sv *SchemaValidator) validateIndexes(ctx context.Context, result *ValidationResult) error {
	requiredIndexes := []string{
		"idx_tasks_repository",
		"idx_tasks_status",
		"idx_tasks_created_at",
		"idx_tasks_parent",
		"idx_tasks_prd",
		"idx_tasks_tags",
		"idx_tasks_search_vector",
		"idx_tasks_repo_session",
		"idx_tasks_priority",
		"idx_tasks_assignee",
	}

	existingIndexes, err := sv.getTableIndexes(ctx, "tasks")
	if err != nil {
		return err
	}

	for _, indexName := range requiredIndexes {
		checkItem := SchemaCheckItem{
			Name:        indexName,
			Type:        "index",
			Expected:    "exists",
			Description: fmt.Sprintf("Index '%s' should exist for performance", indexName),
		}

		if sv.indexExists(existingIndexes, indexName) {
			checkItem.Status = "pass"
			checkItem.Actual = "exists"
		} else {
			checkItem.Status = "warning"
			checkItem.Actual = "missing"
			result.Warnings = append(result.Warnings, ValidationWarning{
				Component:  indexName,
				Type:       "missing",
				Message:    fmt.Sprintf("Recommended index '%s' is missing", indexName),
				Suggestion: "Create index for better query performance",
				Code:       "MISSING_INDEX",
			})
		}

		result.CheckedItems = append(result.CheckedItems, checkItem)
	}

	return nil
}

// validateConstraints checks database constraints
func (sv *SchemaValidator) validateConstraints(ctx context.Context, result *ValidationResult) error {
	// Check primary key constraints
	if err := sv.checkPrimaryKey(ctx, "tasks", "id", result); err != nil {
		return err
	}

	// Check foreign key constraints
	foreignKeys := map[string]string{
		"fk_tasks_prd":        "prd_id -> prds(id)",
		"fk_tasks_source_prd": "source_prd_id -> prds(id)",
		"fk_tasks_parent":     "parent_task_id -> tasks(id)",
	}

	for fkName, description := range foreignKeys {
		exists, err := sv.constraintExists(ctx, "tasks", fkName)
		if err != nil {
			return err
		}

		checkItem := SchemaCheckItem{
			Name:        fkName,
			Type:        "constraint",
			Expected:    "exists",
			Description: fmt.Sprintf("Foreign key constraint '%s' (%s)", fkName, description),
		}

		if exists {
			checkItem.Status = "pass"
			checkItem.Actual = "exists"
		} else {
			checkItem.Status = "fail"
			checkItem.Actual = "missing"
			result.IsValid = false
			result.Errors = append(result.Errors, ValidationError{
				Component:  fkName,
				Type:       "missing",
				Message:    fmt.Sprintf("Foreign key constraint '%s' is missing", fkName),
				Severity:   "major",
				Code:       "MISSING_CONSTRAINT",
				Suggestion: fmt.Sprintf("Add foreign key constraint for %s", description),
			})
		}

		result.CheckedItems = append(result.CheckedItems, checkItem)
	}

	return nil
}

// validateTriggers checks that required triggers exist
func (sv *SchemaValidator) validateTriggers(ctx context.Context, result *ValidationResult) error {
	requiredTriggers := []string{
		"task_audit_trigger",
		"update_tasks_updated_at",
		"update_task_search_vector_trigger",
		"task_completion_handler",
	}

	for _, triggerName := range requiredTriggers {
		exists, err := sv.triggerExists(ctx, "tasks", triggerName)
		if err != nil {
			return err
		}

		checkItem := SchemaCheckItem{
			Name:        triggerName,
			Type:        "trigger",
			Expected:    "exists",
			Description: fmt.Sprintf("Trigger '%s' should exist", triggerName),
		}

		if exists {
			checkItem.Status = "pass"
			checkItem.Actual = "exists"
		} else {
			checkItem.Status = "warning"
			checkItem.Actual = "missing"
			result.Warnings = append(result.Warnings, ValidationWarning{
				Component:  triggerName,
				Type:       "missing",
				Message:    fmt.Sprintf("Trigger '%s' is missing", triggerName),
				Suggestion: "Create trigger for automated functionality",
				Code:       "MISSING_TRIGGER",
			})
		}

		result.CheckedItems = append(result.CheckedItems, checkItem)
	}

	return nil
}

// validateDataIntegrity performs data integrity checks
func (sv *SchemaValidator) validateDataIntegrity(ctx context.Context, result *ValidationResult) error {
	// Check for orphaned records
	orphanedCount, err := sv.countOrphanedTasks(ctx)
	if err != nil {
		return err
	}

	checkItem := SchemaCheckItem{
		Name:        "orphaned_tasks",
		Type:        "data_integrity",
		Expected:    "0",
		Actual:      fmt.Sprintf("%d", orphanedCount),
		Description: "Tasks with invalid parent_task_id references",
	}

	if orphanedCount == 0 {
		checkItem.Status = "pass"
	} else {
		checkItem.Status = "warning"
		result.Warnings = append(result.Warnings, ValidationWarning{
			Component:  "data_integrity",
			Type:       "orphaned_records",
			Message:    fmt.Sprintf("Found %d tasks with invalid parent references", orphanedCount),
			Suggestion: "Clean up orphaned parent_task_id references",
			Code:       "ORPHANED_RECORDS",
		})
	}

	result.CheckedItems = append(result.CheckedItems, checkItem)

	// Check for inconsistent status/timestamp combinations
	inconsistentCount, err := sv.countInconsistentStatusTasks(ctx)
	if err != nil {
		return err
	}

	checkItem = SchemaCheckItem{
		Name:        "status_timestamp_consistency",
		Type:        "data_integrity",
		Expected:    "0",
		Actual:      fmt.Sprintf("%d", inconsistentCount),
		Description: "Tasks with inconsistent status and timestamp combinations",
	}

	if inconsistentCount == 0 {
		checkItem.Status = "pass"
	} else {
		checkItem.Status = "warning"
		result.Warnings = append(result.Warnings, ValidationWarning{
			Component:  "data_integrity",
			Type:       "inconsistent_data",
			Message:    fmt.Sprintf("Found %d tasks with inconsistent status/timestamp combinations", inconsistentCount),
			Suggestion: "Review and fix status/timestamp inconsistencies",
			Code:       "INCONSISTENT_STATUS",
		})
	}

	result.CheckedItems = append(result.CheckedItems, checkItem)

	return nil
}

// validatePerformance checks performance-related aspects
func (sv *SchemaValidator) validatePerformance(ctx context.Context, result *ValidationResult) error {
	// Check table size and suggest optimizations
	tableSize, err := sv.getTableSize(ctx, "tasks")
	if err != nil {
		return err
	}

	checkItem := SchemaCheckItem{
		Name:        "table_size",
		Type:        "performance",
		Expected:    "< 5GB",
		Actual:      tableSize,
		Description: "Tasks table size for performance considerations",
	}

	if strings.Contains(tableSize, "GB") {
		// Extract numeric value for comparison (simplified)
		if strings.Contains(tableSize, "5") || strings.Contains(tableSize, "6") ||
			strings.Contains(tableSize, "7") || strings.Contains(tableSize, "8") ||
			strings.Contains(tableSize, "9") {
			checkItem.Status = "warning"
			result.Suggestions = append(result.Suggestions, "Consider implementing table partitioning for large datasets")
		} else {
			checkItem.Status = "pass"
		}
	} else {
		checkItem.Status = "pass"
	}

	result.CheckedItems = append(result.CheckedItems, checkItem)

	return nil
}

// Helper methods

func (sv *SchemaValidator) tableExists(ctx context.Context, tableName string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_name = $1 AND table_schema = 'public'
		)`

	var exists bool
	err := sv.db.QueryRowContext(ctx, query, tableName).Scan(&exists)
	return exists, err
}

func (sv *SchemaValidator) getTableColumns(ctx context.Context, tableName string) (map[string]string, error) {
	query := `
		SELECT column_name, data_type 
		FROM information_schema.columns 
		WHERE table_name = $1 AND table_schema = 'public'
		ORDER BY ordinal_position`

	rows, err := sv.db.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := make(map[string]string)
	for rows.Next() {
		var columnName, dataType string
		if err := rows.Scan(&columnName, &dataType); err != nil {
			return nil, err
		}
		columns[columnName] = dataType
	}

	return columns, rows.Err()
}

func (sv *SchemaValidator) getTableIndexes(ctx context.Context, tableName string) ([]string, error) {
	query := `
		SELECT indexname 
		FROM pg_indexes 
		WHERE tablename = $1 AND schemaname = 'public'`

	rows, err := sv.db.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []string
	for rows.Next() {
		var indexName string
		if err := rows.Scan(&indexName); err != nil {
			return nil, err
		}
		indexes = append(indexes, indexName)
	}

	return indexes, rows.Err()
}

func (sv *SchemaValidator) constraintExists(ctx context.Context, tableName, constraintName string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.table_constraints 
			WHERE table_name = $1 AND constraint_name = $2 AND table_schema = 'public'
		)`

	var exists bool
	err := sv.db.QueryRowContext(ctx, query, tableName, constraintName).Scan(&exists)
	return exists, err
}

func (sv *SchemaValidator) triggerExists(ctx context.Context, tableName, triggerName string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.triggers 
			WHERE event_object_table = $1 AND trigger_name = $2
		)`

	var exists bool
	err := sv.db.QueryRowContext(ctx, query, tableName, triggerName).Scan(&exists)
	return exists, err
}

func (sv *SchemaValidator) checkPrimaryKey(ctx context.Context, tableName, columnName string, result *ValidationResult) error {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.key_column_usage kcu
			JOIN information_schema.table_constraints tc 
			ON kcu.constraint_name = tc.constraint_name
			WHERE tc.table_name = $1 AND kcu.column_name = $2 
			AND tc.constraint_type = 'PRIMARY KEY'
		)`

	var exists bool
	err := sv.db.QueryRowContext(ctx, query, tableName, columnName).Scan(&exists)
	if err != nil {
		return err
	}

	checkItem := SchemaCheckItem{
		Name:        fmt.Sprintf("%s.%s_pk", tableName, columnName),
		Type:        "constraint",
		Expected:    "primary key",
		Description: fmt.Sprintf("Primary key constraint on %s.%s", tableName, columnName),
	}

	if exists {
		checkItem.Status = "pass"
		checkItem.Actual = "primary key"
	} else {
		checkItem.Status = "fail"
		checkItem.Actual = "missing"
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Component:  fmt.Sprintf("%s.%s", tableName, columnName),
			Type:       "missing",
			Message:    fmt.Sprintf("Primary key constraint missing on %s.%s", tableName, columnName),
			Severity:   "critical",
			Code:       "MISSING_PRIMARY_KEY",
			Suggestion: fmt.Sprintf("Add primary key constraint to %s.%s", tableName, columnName),
		})
	}

	result.CheckedItems = append(result.CheckedItems, checkItem)
	return nil
}

func (sv *SchemaValidator) countOrphanedTasks(ctx context.Context) (int, error) {
	query := `
		SELECT COUNT(*) FROM tasks 
		WHERE parent_task_id IS NOT NULL 
		AND parent_task_id NOT IN (SELECT id FROM tasks WHERE deleted_at IS NULL)
		AND deleted_at IS NULL`

	var count int
	err := sv.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}

func (sv *SchemaValidator) countInconsistentStatusTasks(ctx context.Context) (int, error) {
	query := `
		SELECT COUNT(*) FROM tasks 
		WHERE deleted_at IS NULL AND (
			(status = 'completed' AND completed_at IS NULL) OR
			(status != 'completed' AND completed_at IS NOT NULL) OR
			(status = 'in_progress' AND started_at IS NULL)
		)`

	var count int
	err := sv.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}

func (sv *SchemaValidator) getTableSize(ctx context.Context, tableName string) (string, error) {
	query := `SELECT pg_size_pretty(pg_total_relation_size($1))`

	var size string
	err := sv.db.QueryRowContext(ctx, query, tableName).Scan(&size)
	return size, err
}

func (sv *SchemaValidator) isTypeCompatible(expected, actual string) bool {
	// Simplified type compatibility check
	switch expected {
	case "USER-DEFINED":
		return strings.Contains(actual, "enum") || actual == "USER-DEFINED"
	case "character varying":
		return actual == "character varying" || actual == "varchar" || actual == "text"
	case "timestamp with time zone":
		return actual == "timestamp with time zone" || actual == "timestamptz"
	case "numeric":
		return actual == "numeric" || actual == "decimal"
	default:
		return expected == actual
	}
}

func (sv *SchemaValidator) indexExists(indexes []string, indexName string) bool {
	for _, idx := range indexes {
		if idx == indexName {
			return true
		}
	}
	return false
}

func (sv *SchemaValidator) calculateValidationScore(result *ValidationResult) {
	totalChecks := len(result.CheckedItems)
	passedChecks := 0
	warningChecks := 0
	failedChecks := 0
	criticalErrors := 0

	for _, item := range result.CheckedItems {
		switch item.Status {
		case "pass":
			passedChecks++
		case "warning":
			warningChecks++
		case "fail":
			failedChecks++
		}
	}

	for _, err := range result.Errors {
		if err.Severity == "critical" {
			criticalErrors++
		}
	}

	result.Summary = ValidationSummary{
		TotalChecks:    totalChecks,
		PassedChecks:   passedChecks,
		FailedChecks:   failedChecks,
		WarningChecks:  warningChecks,
		CriticalErrors: criticalErrors,
		LastValidated:  time.Now(),
	}

	if totalChecks > 0 {
		result.Summary.PassPercentage = float64(passedChecks) / float64(totalChecks) * 100

		// Calculate overall score (passed checks with penalty for failures)
		score := float64(passedChecks) / float64(totalChecks)

		// Apply penalties
		if failedChecks > 0 {
			score -= float64(failedChecks) * 0.1 // 10% penalty per failed check
		}
		if criticalErrors > 0 {
			score -= float64(criticalErrors) * 0.2 // 20% penalty per critical error
		}

		// Ensure score is between 0 and 1
		if score < 0 {
			score = 0
		}

		result.Score = score
	}

	// Set schema version (could be read from a version table)
	result.Summary.SchemaVersion = "1.0.0"
}

// ValidateTask validates a single task against schema requirements
func (sv *SchemaValidator) ValidateTask(ctx context.Context, task *types.Task) (*types.TaskValidationResult, error) {
	result := &types.TaskValidationResult{
		IsValid:     true,
		Errors:      []types.ValidationError{},
		Warnings:    []types.ValidationWarning{},
		Suggestions: []string{},
		Score:       1.0,
	}

	// Validate required fields
	if task.Title == "" {
		result.IsValid = false
		result.Errors = append(result.Errors, types.ValidationError{
			Field:    "title",
			Type:     "required",
			Message:  "Title is required",
			Severity: "critical",
			Code:     "REQUIRED_FIELD",
		})
	}

	if task.Description == "" {
		result.Warnings = append(result.Warnings, types.ValidationWarning{
			Field:      "description",
			Type:       "missing",
			Message:    "Description is empty",
			Suggestion: "Add a description for better task clarity",
			Code:       "MISSING_DESCRIPTION",
		})
	}

	// Validate enums
	validStatuses := []string{"pending", "in_progress", "completed", "cancelled", "blocked", "todo"}
	if !sv.isValidEnum(string(task.Status), validStatuses) {
		result.IsValid = false
		result.Errors = append(result.Errors, types.ValidationError{
			Field:    "status",
			Type:     "invalid_value",
			Message:  fmt.Sprintf("Invalid status: %s", task.Status),
			Severity: "major",
			Code:     "INVALID_ENUM",
		})
	}

	validPriorities := []string{"low", "medium", "high", "critical", "blocking"}
	if !sv.isValidEnum(string(task.Priority), validPriorities) {
		result.IsValid = false
		result.Errors = append(result.Errors, types.ValidationError{
			Field:    "priority",
			Type:     "invalid_value",
			Message:  fmt.Sprintf("Invalid priority: %s", task.Priority),
			Severity: "major",
			Code:     "INVALID_ENUM",
		})
	}

	// Calculate score based on errors and warnings
	totalIssues := len(result.Errors) + len(result.Warnings)
	if totalIssues > 0 {
		result.Score = 1.0 - (float64(len(result.Errors))*0.2 + float64(len(result.Warnings))*0.1)
		if result.Score < 0 {
			result.Score = 0
		}
	}

	return result, nil
}

func (sv *SchemaValidator) isValidEnum(value string, validValues []string) bool {
	for _, valid := range validValues {
		if value == valid {
			return true
		}
	}
	return false
}
