// Package storage provides template and pattern repository implementation for database operations.
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	"lerian-mcp-memory/pkg/types"
)

// TemplateRepository provides database operations for task templates
type TemplateRepository struct {
	db *sql.DB
}

// NewTemplateRepository creates a new template repository
func NewTemplateRepository(db *sql.DB) *TemplateRepository {
	return &TemplateRepository{db: db}
}

// CreateTemplate inserts a new template into the database
func (r *TemplateRepository) CreateTemplate(ctx context.Context, template *types.TaskTemplate) error {
	query := `
		INSERT INTO task_templates (
			id, name, description, category, template_data, applicability,
			variables, project_type, complexity_level, estimated_effort_hours,
			required_skills, usage_count, success_rate, avg_completion_time_hours,
			user_rating, feedback_count, version, parent_template_id, is_active,
			created_by, tags, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24
		)`

	_, err := r.db.ExecContext(ctx, query,
		template.ID, template.Name, template.Description, template.Category,
		template.TemplateData, template.Applicability, template.Variables,
		template.ProjectType, template.ComplexityLevel, template.EstimatedEffortHours,
		template.RequiredSkills, template.UsageCount, template.SuccessRate,
		template.AvgCompletionTimeHours, template.UserRating, template.FeedbackCount,
		template.Version, template.ParentTemplateID, template.IsActive,
		template.CreatedBy, template.Tags, template.Metadata,
		template.CreatedAt, template.UpdatedAt,
	)

	return err
}

// GetTemplateByID retrieves a template by its ID
func (r *TemplateRepository) GetTemplateByID(ctx context.Context, id string) (*types.TaskTemplate, error) {
	query := `
		SELECT id, name, description, category, template_data, applicability,
		       variables, project_type, complexity_level, estimated_effort_hours,
		       required_skills, usage_count, success_rate, avg_completion_time_hours,
		       user_rating, feedback_count, version, parent_template_id, is_active,
		       created_by, tags, metadata, created_at, updated_at, deleted_at
		FROM task_templates
		WHERE id = $1 AND deleted_at IS NULL`

	template := &types.TaskTemplate{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&template.ID, &template.Name, &template.Description, &template.Category,
		&template.TemplateData, &template.Applicability, &template.Variables,
		&template.ProjectType, &template.ComplexityLevel, &template.EstimatedEffortHours,
		&template.RequiredSkills, &template.UsageCount, &template.SuccessRate,
		&template.AvgCompletionTimeHours, &template.UserRating, &template.FeedbackCount,
		&template.Version, &template.ParentTemplateID, &template.IsActive,
		&template.CreatedBy, &template.Tags, &template.Metadata,
		&template.CreatedAt, &template.UpdatedAt, &template.DeletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("template not found with ID: %s", id)
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	return template, nil
}

// ListTemplates retrieves templates with filtering and pagination
func (r *TemplateRepository) ListTemplates(ctx context.Context, filters *TemplateFilters) ([]*types.TaskTemplate, error) {
	query, args := r.buildListTemplatesQuery(filters)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query templates: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var templates []*types.TaskTemplate
	for rows.Next() {
		template, err := r.scanTemplate(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan template: %w", err)
		}
		templates = append(templates, template)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return templates, nil
}

func (r *TemplateRepository) buildListTemplatesQuery(filters *TemplateFilters) (query string, args []interface{}) {
	var queryBuilder strings.Builder
	queryBuilder.WriteString(`
		SELECT id, name, description, category, template_data, applicability,
		       variables, project_type, complexity_level, estimated_effort_hours,
		       required_skills, usage_count, success_rate, avg_completion_time_hours,
		       user_rating, feedback_count, version, parent_template_id, is_active,
		       created_by, tags, metadata, created_at, updated_at, deleted_at
		FROM task_templates
		WHERE deleted_at IS NULL`)

	args = make([]interface{}, 0)
	argCount := 0

	// Add category filter
	if filters.Category != nil {
		argCount++
		queryBuilder.WriteString(fmt.Sprintf(" AND category = $%d", argCount))
		args = append(args, *filters.Category)
	}

	// Add project type filter
	if filters.ProjectType != nil {
		argCount++
		queryBuilder.WriteString(fmt.Sprintf(" AND project_type = $%d", argCount))
		args = append(args, *filters.ProjectType)
	}

	// Add complexity filter
	if filters.ComplexityLevel != nil {
		argCount++
		queryBuilder.WriteString(fmt.Sprintf(" AND complexity_level = $%d", argCount))
		args = append(args, *filters.ComplexityLevel)
	}

	// Add active filter
	if filters.IsActive != nil {
		argCount++
		queryBuilder.WriteString(fmt.Sprintf(" AND is_active = $%d", argCount))
		args = append(args, *filters.IsActive)
	}

	// Add minimum success rate filter
	if filters.MinSuccessRate != nil {
		argCount++
		queryBuilder.WriteString(fmt.Sprintf(" AND success_rate >= $%d", argCount))
		args = append(args, *filters.MinSuccessRate)
	}

	// Add ordering
	switch filters.OrderBy {
	case "usage_count":
		queryBuilder.WriteString(" ORDER BY usage_count DESC")
	case "success_rate":
		queryBuilder.WriteString(" ORDER BY success_rate DESC")
	case "user_rating":
		queryBuilder.WriteString(" ORDER BY user_rating DESC")
	case "name":
		queryBuilder.WriteString(" ORDER BY name ASC")
	default:
		queryBuilder.WriteString(" ORDER BY created_at DESC")
	}

	// Add pagination
	if filters.Limit > 0 {
		argCount++
		queryBuilder.WriteString(fmt.Sprintf(" LIMIT $%d", argCount))
		args = append(args, filters.Limit)

		if filters.Offset > 0 {
			argCount++
			queryBuilder.WriteString(fmt.Sprintf(" OFFSET $%d", argCount))
			args = append(args, filters.Offset)
		}
	}

	query = queryBuilder.String()
	return
}

func (r *TemplateRepository) scanTemplate(rows *sql.Rows) (*types.TaskTemplate, error) {
	template := &types.TaskTemplate{}
	err := rows.Scan(
		&template.ID, &template.Name, &template.Description, &template.Category,
		&template.TemplateData, &template.Applicability, &template.Variables,
		&template.ProjectType, &template.ComplexityLevel, &template.EstimatedEffortHours,
		&template.RequiredSkills, &template.UsageCount, &template.SuccessRate,
		&template.AvgCompletionTimeHours, &template.UserRating, &template.FeedbackCount,
		&template.Version, &template.ParentTemplateID, &template.IsActive,
		&template.CreatedBy, &template.Tags, &template.Metadata,
		&template.CreatedAt, &template.UpdatedAt, &template.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return template, nil
}

// PatternRepository provides database operations for task patterns
type PatternRepository struct {
	db *sql.DB
}

func (r *PatternRepository) calculateNewAvgCompletionTime(currentAvg *int32, currentCount, newCompletionMinutes, newCount int32) *int32 {
	if currentAvg != nil {
		total := int64(*currentAvg)*int64(currentCount) + int64(newCompletionMinutes)
		avg := total / int64(newCount)
		if avg > math.MaxInt32 || avg < math.MinInt32 {
			if avg > 0 {
				avg = math.MaxInt32
			} else {
				avg = math.MinInt32
			}
		}
		avgMinutes := int32(avg) // #nosec G115 - overflow check above
		return &avgMinutes
	}
	return &newCompletionMinutes
}

// UpdateTemplate modifies an existing template
func (r *TemplateRepository) UpdateTemplate(ctx context.Context, template *types.TaskTemplate) error {
	query := `
		UPDATE task_templates SET
			name = $2, description = $3, category = $4, template_data = $5,
			applicability = $6, variables = $7, project_type = $8,
			complexity_level = $9, estimated_effort_hours = $10,
			required_skills = $11, usage_count = $12, success_rate = $13,
			avg_completion_time_hours = $14, user_rating = $15,
			feedback_count = $16, version = $17, parent_template_id = $18,
			is_active = $19, created_by = $20, tags = $21, metadata = $22,
			updated_at = $23
		WHERE id = $1 AND deleted_at IS NULL`

	template.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(ctx, query,
		template.ID, template.Name, template.Description, template.Category,
		template.TemplateData, template.Applicability, template.Variables,
		template.ProjectType, template.ComplexityLevel, template.EstimatedEffortHours,
		template.RequiredSkills, template.UsageCount, template.SuccessRate,
		template.AvgCompletionTimeHours, template.UserRating, template.FeedbackCount,
		template.Version, template.ParentTemplateID, template.IsActive,
		template.CreatedBy, template.Tags, template.Metadata, template.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update template: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("template not found or already deleted: %s", template.ID)
	}

	return nil
}

// DeleteTemplate soft deletes a template
func (r *TemplateRepository) DeleteTemplate(ctx context.Context, id string) error {
	query := `
		UPDATE task_templates SET 
			deleted_at = $2, 
			updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, id, now)
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("template not found: %s", id)
	}

	return nil
}

// SearchTemplates performs full-text search on templates
func (r *TemplateRepository) SearchTemplates(ctx context.Context, searchQuery string, limit int) ([]*types.TaskTemplate, error) {
	query := `
		SELECT id, name, description, category, template_data, applicability,
		       variables, project_type, complexity_level, estimated_effort_hours,
		       required_skills, usage_count, success_rate, avg_completion_time_hours,
		       user_rating, feedback_count, version, parent_template_id, is_active,
		       created_by, tags, metadata, created_at, updated_at, deleted_at,
		       ts_rank(search_vector, plainto_tsquery('english', $1)) as rank
		FROM task_templates
		WHERE deleted_at IS NULL
		AND search_vector @@ plainto_tsquery('english', $1)
		ORDER BY rank DESC, usage_count DESC
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, searchQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search templates: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var templates []*types.TaskTemplate
	for rows.Next() {
		template := &types.TaskTemplate{}
		var rank float64

		err := rows.Scan(
			&template.ID, &template.Name, &template.Description, &template.Category,
			&template.TemplateData, &template.Applicability, &template.Variables,
			&template.ProjectType, &template.ComplexityLevel, &template.EstimatedEffortHours,
			&template.RequiredSkills, &template.UsageCount, &template.SuccessRate,
			&template.AvgCompletionTimeHours, &template.UserRating, &template.FeedbackCount,
			&template.Version, &template.ParentTemplateID, &template.IsActive,
			&template.CreatedBy, &template.Tags, &template.Metadata,
			&template.CreatedAt, &template.UpdatedAt, &template.DeletedAt, &rank,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}
		templates = append(templates, template)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate search results: %w", err)
	}

	return templates, nil
}

// IncrementTemplateUsage increments the usage count and updates metrics
func (r *TemplateRepository) IncrementTemplateUsage(ctx context.Context, id string, completionTimeHours *float64, wasSuccessful bool) error {
	// Get current stats first
	currentTemplate, err := r.GetTemplateByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get template for usage update: %w", err)
	}

	// Calculate new metrics
	newUsageCount := currentTemplate.UsageCount + 1
	var newAvgCompletionTime *float64
	var newSuccessRate *float64

	// Update average completion time
	if completionTimeHours != nil {
		if currentTemplate.AvgCompletionTimeHours != nil {
			avgTime := ((*currentTemplate.AvgCompletionTimeHours * float64(currentTemplate.UsageCount)) + *completionTimeHours) / float64(newUsageCount)
			newAvgCompletionTime = &avgTime
		} else {
			newAvgCompletionTime = completionTimeHours
		}
	}

	// Update success rate
	if currentTemplate.SuccessRate != nil {
		currentSuccessCount := int32(float64(currentTemplate.UsageCount) * (*currentTemplate.SuccessRate))
		if wasSuccessful {
			currentSuccessCount++
		}
		successRate := float64(currentSuccessCount) / float64(newUsageCount)
		newSuccessRate = &successRate
	} else {
		if wasSuccessful {
			successRate := 1.0
			newSuccessRate = &successRate
		} else {
			successRate := 0.0
			newSuccessRate = &successRate
		}
	}

	query := `
		UPDATE task_templates SET
			usage_count = $2,
			avg_completion_time_hours = $3,
			success_rate = $4,
			updated_at = $5
		WHERE id = $1 AND deleted_at IS NULL`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, id, newUsageCount, newAvgCompletionTime, newSuccessRate, now)
	if err != nil {
		return fmt.Errorf("failed to update template usage: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("template not found: %s", id)
	}

	return nil
}

// NewPatternRepository creates a new pattern repository
func NewPatternRepository(db *sql.DB) *PatternRepository {
	return &PatternRepository{db: db}
}

// CreatePattern inserts a new pattern into the database
func (r *PatternRepository) CreatePattern(ctx context.Context, pattern *types.TaskPattern) error {
	query := `
		INSERT INTO task_patterns (
			id, name, description, pattern_type, template, conditions,
			task_sequence, occurrence_count, avg_completion_time_minutes,
			success_rate, efficiency_score, repositories, project_types,
			team_sizes, complexity_levels, parent_pattern_id, related_patterns,
			confidence_score, feature_vector, last_trained_at, last_used_at,
			auto_suggested_count, user_accepted_count, user_rejected_count,
			status, validation_status, discovered_by, created_by, tags,
			metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32
		)`

	_, err := r.db.ExecContext(ctx, query,
		pattern.ID, pattern.Name, pattern.Description, pattern.PatternType,
		pattern.Template, pattern.Conditions, pattern.TaskSequence,
		pattern.OccurrenceCount, pattern.AvgCompletionMinutes, pattern.SuccessRate,
		pattern.EfficiencyScore, pattern.Repositories, pattern.ProjectTypes,
		pattern.TeamSizes, pattern.ComplexityLevels, pattern.ParentPatternID,
		pattern.RelatedPatterns, pattern.ConfidenceScore, pattern.FeatureVector,
		pattern.LastTrainedAt, pattern.LastUsedAt, pattern.AutoSuggestedCount,
		pattern.UserAcceptedCount, pattern.UserRejectedCount, pattern.Status,
		pattern.ValidationStatus, pattern.DiscoveredBy, pattern.CreatedBy,
		pattern.Tags, pattern.Metadata, pattern.CreatedAt, pattern.UpdatedAt,
	)

	return err
}

// GetPatternByID retrieves a pattern by its ID
func (r *PatternRepository) GetPatternByID(ctx context.Context, id string) (*types.TaskPattern, error) {
	query := `
		SELECT id, name, description, pattern_type, template, conditions,
		       task_sequence, occurrence_count, avg_completion_time_minutes,
		       success_rate, efficiency_score, repositories, project_types,
		       team_sizes, complexity_levels, parent_pattern_id, related_patterns,
		       confidence_score, feature_vector, last_trained_at, last_used_at,
		       auto_suggested_count, user_accepted_count, user_rejected_count,
		       status, validation_status, discovered_by, created_by, tags,
		       metadata, created_at, updated_at, deleted_at
		FROM task_patterns
		WHERE id = $1 AND deleted_at IS NULL`

	pattern := &types.TaskPattern{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&pattern.ID, &pattern.Name, &pattern.Description, &pattern.PatternType,
		&pattern.Template, &pattern.Conditions, &pattern.TaskSequence,
		&pattern.OccurrenceCount, &pattern.AvgCompletionMinutes, &pattern.SuccessRate,
		&pattern.EfficiencyScore, &pattern.Repositories, &pattern.ProjectTypes,
		&pattern.TeamSizes, &pattern.ComplexityLevels, &pattern.ParentPatternID,
		&pattern.RelatedPatterns, &pattern.ConfidenceScore, &pattern.FeatureVector,
		&pattern.LastTrainedAt, &pattern.LastUsedAt, &pattern.AutoSuggestedCount,
		&pattern.UserAcceptedCount, &pattern.UserRejectedCount, &pattern.Status,
		&pattern.ValidationStatus, &pattern.DiscoveredBy, &pattern.CreatedBy,
		&pattern.Tags, &pattern.Metadata, &pattern.CreatedAt, &pattern.UpdatedAt,
		&pattern.DeletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("pattern not found with ID: %s", id)
		}
		return nil, fmt.Errorf("failed to get pattern: %w", err)
	}

	return pattern, nil
}

// ListPatterns retrieves patterns with filtering and pagination
func (r *PatternRepository) ListPatterns(ctx context.Context, filters *PatternFilters) ([]*types.TaskPattern, error) {
	var queryBuilder strings.Builder
	queryBuilder.WriteString(`
		SELECT id, name, description, pattern_type, template, conditions,
		       task_sequence, occurrence_count, avg_completion_time_minutes,
		       success_rate, efficiency_score, repositories, project_types,
		       team_sizes, complexity_levels, parent_pattern_id, related_patterns,
		       confidence_score, feature_vector, last_trained_at, last_used_at,
		       auto_suggested_count, user_accepted_count, user_rejected_count,
		       status, validation_status, discovered_by, created_by, tags,
		       metadata, created_at, updated_at, deleted_at
		FROM task_patterns
		WHERE deleted_at IS NULL`)

	args := []interface{}{}
	argCount := 0

	// Add pattern type filter
	if filters.PatternType != nil {
		argCount++
		queryBuilder.WriteString(fmt.Sprintf(" AND pattern_type = $%d", argCount))
		args = append(args, *filters.PatternType)
	}

	// Add status filter
	if filters.Status != nil {
		argCount++
		queryBuilder.WriteString(fmt.Sprintf(" AND status = $%d", argCount))
		args = append(args, *filters.Status)
	}

	// Add validation status filter
	if filters.ValidationStatus != nil {
		argCount++
		queryBuilder.WriteString(fmt.Sprintf(" AND validation_status = $%d", argCount))
		args = append(args, *filters.ValidationStatus)
	}

	// Add minimum confidence filter
	if filters.MinConfidence != nil {
		argCount++
		queryBuilder.WriteString(fmt.Sprintf(" AND confidence_score >= $%d", argCount))
		args = append(args, *filters.MinConfidence)
	}

	// Add ordering
	switch filters.OrderBy {
	case "confidence_score":
		queryBuilder.WriteString(" ORDER BY confidence_score DESC")
	case "occurrence_count":
		queryBuilder.WriteString(" ORDER BY occurrence_count DESC")
	case "success_rate":
		queryBuilder.WriteString(" ORDER BY success_rate DESC")
	case "last_used":
		queryBuilder.WriteString(" ORDER BY last_used_at DESC")
	default:
		queryBuilder.WriteString(" ORDER BY created_at DESC")
	}

	// Add pagination
	if filters.Limit > 0 {
		argCount++
		queryBuilder.WriteString(fmt.Sprintf(" LIMIT $%d", argCount))
		args = append(args, filters.Limit)

		if filters.Offset > 0 {
			argCount++
			queryBuilder.WriteString(fmt.Sprintf(" OFFSET $%d", argCount))
			args = append(args, filters.Offset)
		}
	}

	rows, err := r.db.QueryContext(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query patterns: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var patterns []*types.TaskPattern
	for rows.Next() {
		pattern := &types.TaskPattern{}
		err := rows.Scan(
			&pattern.ID, &pattern.Name, &pattern.Description, &pattern.PatternType,
			&pattern.Template, &pattern.Conditions, &pattern.TaskSequence,
			&pattern.OccurrenceCount, &pattern.AvgCompletionMinutes, &pattern.SuccessRate,
			&pattern.EfficiencyScore, &pattern.Repositories, &pattern.ProjectTypes,
			&pattern.TeamSizes, &pattern.ComplexityLevels, &pattern.ParentPatternID,
			&pattern.RelatedPatterns, &pattern.ConfidenceScore, &pattern.FeatureVector,
			&pattern.LastTrainedAt, &pattern.LastUsedAt, &pattern.AutoSuggestedCount,
			&pattern.UserAcceptedCount, &pattern.UserRejectedCount, &pattern.Status,
			&pattern.ValidationStatus, &pattern.DiscoveredBy, &pattern.CreatedBy,
			&pattern.Tags, &pattern.Metadata, &pattern.CreatedAt, &pattern.UpdatedAt,
			&pattern.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pattern: %w", err)
		}
		patterns = append(patterns, pattern)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate pattern rows: %w", err)
	}

	return patterns, nil
}

// UpdatePatternMetrics updates pattern metrics after usage
func (r *PatternRepository) UpdatePatternMetrics(ctx context.Context, id string, completionMinutes int32, wasSuccessful, wasAccepted bool) error {
	// Get current pattern first
	currentPattern, err := r.GetPatternByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get pattern for metrics update: %w", err)
	}

	// Calculate new metrics
	newOccurrenceCount := currentPattern.OccurrenceCount + 1
	var newAvgCompletionMinutes *int32
	var newSuccessRate *float64

	// Update average completion time
	newAvgCompletionMinutes = r.calculateNewAvgCompletionTime(currentPattern.AvgCompletionMinutes, currentPattern.OccurrenceCount, completionMinutes, newOccurrenceCount)

	// Update success rate
	if currentPattern.SuccessRate != nil {
		currentSuccessCount := int32(float64(currentPattern.OccurrenceCount) * (*currentPattern.SuccessRate))
		if wasSuccessful {
			currentSuccessCount++
		}
		successRate := float64(currentSuccessCount) / float64(newOccurrenceCount)
		newSuccessRate = &successRate
	} else {
		if wasSuccessful {
			successRate := 1.0
			newSuccessRate = &successRate
		} else {
			successRate := 0.0
			newSuccessRate = &successRate
		}
	}

	// Update acceptance counts
	newAutoSuggestedCount := currentPattern.AutoSuggestedCount + 1
	newUserAcceptedCount := currentPattern.UserAcceptedCount
	newUserRejectedCount := currentPattern.UserRejectedCount

	if wasAccepted {
		newUserAcceptedCount++
	} else {
		newUserRejectedCount++
	}

	query := `
		UPDATE task_patterns SET
			occurrence_count = $2,
			avg_completion_time_minutes = $3,
			success_rate = $4,
			auto_suggested_count = $5,
			user_accepted_count = $6,
			user_rejected_count = $7,
			last_used_at = $8,
			updated_at = $8
		WHERE id = $1 AND deleted_at IS NULL`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, id, newOccurrenceCount, newAvgCompletionMinutes, newSuccessRate, newAutoSuggestedCount, newUserAcceptedCount, newUserRejectedCount, now)
	if err != nil {
		return fmt.Errorf("failed to update pattern metrics: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("pattern not found: %s", id)
	}

	return nil
}

// Filter types

// TemplateFilters represents filters for template queries
type TemplateFilters struct {
	Category        *string  `json:"category,omitempty"`
	ProjectType     *string  `json:"project_type,omitempty"`
	ComplexityLevel *string  `json:"complexity_level,omitempty"`
	IsActive        *bool    `json:"is_active,omitempty"`
	MinSuccessRate  *float64 `json:"min_success_rate,omitempty"`
	OrderBy         string   `json:"order_by"` // usage_count, success_rate, user_rating, name, created_at
	Limit           int      `json:"limit"`
	Offset          int      `json:"offset"`
}

// PatternFilters represents filters for pattern queries
type PatternFilters struct {
	PatternType      *string  `json:"pattern_type,omitempty"`
	Status           *string  `json:"status,omitempty"`
	ValidationStatus *string  `json:"validation_status,omitempty"`
	MinConfidence    *float64 `json:"min_confidence,omitempty"`
	OrderBy          string   `json:"order_by"` // confidence_score, occurrence_count, success_rate, last_used, created_at
	Limit            int      `json:"limit"`
	Offset           int      `json:"offset"`
}
