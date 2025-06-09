// Package storage provides PRD repository implementation for database operations.
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"lerian-mcp-memory/pkg/types"
)

// PRDRepository provides database operations for PRDs
type PRDRepository struct {
	db *sql.DB
}

// NewPRDRepository creates a new PRD repository
func NewPRDRepository(db *sql.DB) *PRDRepository {
	return &PRDRepository{db: db}
}

// Create inserts a new PRD into the database
func (r *PRDRepository) Create(ctx context.Context, prd *types.EnhancedPRD) error {
	query := `
		INSERT INTO prds (
			id, repository, filename, content, task_count, complexity_score,
			validation_score, version, file_size_bytes, content_type, author,
			last_parsed_version, parse_status, parse_errors, document_type,
			priority_level, status, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
		)`

	_, err := r.db.ExecContext(ctx, query,
		prd.ID, prd.Repository, prd.Filename, prd.Content, prd.TaskCount,
		prd.ComplexityScore, prd.ValidationScore, prd.Version, prd.FileSizeBytes,
		prd.ContentType, prd.Author, prd.LastParsedVersion, prd.ParseStatus,
		prd.ParseErrors, prd.DocumentType, prd.PriorityLevel, prd.Status,
		prd.Metadata, prd.CreatedAt, prd.UpdatedAt,
	)

	return err
}

// GetByID retrieves a PRD by its ID
func (r *PRDRepository) GetByID(ctx context.Context, id string) (*types.EnhancedPRD, error) {
	query := `
		SELECT id, repository, filename, content, parsed_at, task_count,
		       complexity_score, validation_score, version, file_size_bytes,
		       file_hash, content_type, author, last_parsed_version,
		       parse_status, parse_errors, document_type, priority_level,
		       status, metadata, created_at, updated_at, deleted_at
		FROM prds
		WHERE id = $1 AND deleted_at IS NULL`

	prd := &types.EnhancedPRD{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&prd.ID, &prd.Repository, &prd.Filename, &prd.Content, &prd.ParsedAt,
		&prd.TaskCount, &prd.ComplexityScore, &prd.ValidationScore, &prd.Version,
		&prd.FileSizeBytes, &prd.FileHash, &prd.ContentType, &prd.Author,
		&prd.LastParsedVersion, &prd.ParseStatus, &prd.ParseErrors,
		&prd.DocumentType, &prd.PriorityLevel, &prd.Status, &prd.Metadata,
		&prd.CreatedAt, &prd.UpdatedAt, &prd.DeletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("PRD not found with ID: %s", id)
		}
		return nil, fmt.Errorf("failed to get PRD: %w", err)
	}

	return prd, nil
}

// GetByRepository retrieves PRDs for a specific repository
func (r *PRDRepository) GetByRepository(ctx context.Context, repository string, limit, offset int) ([]*types.EnhancedPRD, error) {
	query := `
		SELECT id, repository, filename, content, parsed_at, task_count,
		       complexity_score, validation_score, version, file_size_bytes,
		       file_hash, content_type, author, last_parsed_version,
		       parse_status, parse_errors, document_type, priority_level,
		       status, metadata, created_at, updated_at, deleted_at
		FROM prds
		WHERE repository = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, repository, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query PRDs by repository: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var prds []*types.EnhancedPRD
	for rows.Next() {
		prd := &types.EnhancedPRD{}
		err := rows.Scan(
			&prd.ID, &prd.Repository, &prd.Filename, &prd.Content, &prd.ParsedAt,
			&prd.TaskCount, &prd.ComplexityScore, &prd.ValidationScore, &prd.Version,
			&prd.FileSizeBytes, &prd.FileHash, &prd.ContentType, &prd.Author,
			&prd.LastParsedVersion, &prd.ParseStatus, &prd.ParseErrors,
			&prd.DocumentType, &prd.PriorityLevel, &prd.Status, &prd.Metadata,
			&prd.CreatedAt, &prd.UpdatedAt, &prd.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan PRD: %w", err)
		}
		prds = append(prds, prd)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate PRD rows: %w", err)
	}

	return prds, nil
}

// List retrieves PRDs with filtering and pagination
func (r *PRDRepository) List(ctx context.Context, filters *PRDFilters) ([]*types.EnhancedPRD, error) {
	query := `
		SELECT id, repository, filename, content, parsed_at, task_count,
		       complexity_score, validation_score, version, file_size_bytes,
		       file_hash, content_type, author, last_parsed_version,
		       parse_status, parse_errors, document_type, priority_level,
		       status, metadata, created_at, updated_at, deleted_at
		FROM prds
		WHERE deleted_at IS NULL`

	args := []interface{}{}
	argCount := 0

	// Add repository filter
	if filters.Repository != nil {
		argCount++
		query += fmt.Sprintf(" AND repository = $%d", argCount)
		args = append(args, *filters.Repository)
	}

	// Add status filter
	if filters.Status != nil {
		argCount++
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *filters.Status)
	}

	// Add document type filter
	if filters.DocumentType != nil {
		argCount++
		query += fmt.Sprintf(" AND document_type = $%d", argCount)
		args = append(args, *filters.DocumentType)
	}

	// Add parse status filter
	if filters.ParseStatus != nil {
		argCount++
		query += fmt.Sprintf(" AND parse_status = $%d", argCount)
		args = append(args, *filters.ParseStatus)
	}

	// Add time range filters
	if filters.CreatedAfter != nil {
		argCount++
		query += fmt.Sprintf(" AND created_at >= $%d", argCount)
		args = append(args, *filters.CreatedAfter)
	}

	if filters.CreatedBefore != nil {
		argCount++
		query += fmt.Sprintf(" AND created_at <= $%d", argCount)
		args = append(args, *filters.CreatedBefore)
	}

	// Add ordering
	query += " ORDER BY created_at DESC"

	// Add pagination
	if filters.Limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filters.Limit)

		if filters.Offset > 0 {
			argCount++
			query += fmt.Sprintf(" OFFSET $%d", argCount)
			args = append(args, filters.Offset)
		}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query PRDs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var prds []*types.EnhancedPRD
	for rows.Next() {
		prd := &types.EnhancedPRD{}
		err := rows.Scan(
			&prd.ID, &prd.Repository, &prd.Filename, &prd.Content, &prd.ParsedAt,
			&prd.TaskCount, &prd.ComplexityScore, &prd.ValidationScore, &prd.Version,
			&prd.FileSizeBytes, &prd.FileHash, &prd.ContentType, &prd.Author,
			&prd.LastParsedVersion, &prd.ParseStatus, &prd.ParseErrors,
			&prd.DocumentType, &prd.PriorityLevel, &prd.Status, &prd.Metadata,
			&prd.CreatedAt, &prd.UpdatedAt, &prd.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan PRD: %w", err)
		}
		prds = append(prds, prd)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate PRD rows: %w", err)
	}

	return prds, nil
}

// Update modifies an existing PRD
func (r *PRDRepository) Update(ctx context.Context, prd *types.EnhancedPRD) error {
	query := `
		UPDATE prds SET
			repository = $2, filename = $3, content = $4, task_count = $5,
			complexity_score = $6, validation_score = $7, version = $8,
			file_size_bytes = $9, content_type = $10, author = $11,
			last_parsed_version = $12, parse_status = $13, parse_errors = $14,
			document_type = $15, priority_level = $16, status = $17,
			metadata = $18, updated_at = $19
		WHERE id = $1 AND deleted_at IS NULL`

	prd.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(ctx, query,
		prd.ID, prd.Repository, prd.Filename, prd.Content, prd.TaskCount,
		prd.ComplexityScore, prd.ValidationScore, prd.Version, prd.FileSizeBytes,
		prd.ContentType, prd.Author, prd.LastParsedVersion, prd.ParseStatus,
		prd.ParseErrors, prd.DocumentType, prd.PriorityLevel, prd.Status,
		prd.Metadata, prd.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update PRD: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("PRD not found or already deleted: %s", prd.ID)
	}

	return nil
}

// Delete soft deletes a PRD
func (r *PRDRepository) Delete(ctx context.Context, id string) error {
	query := `
		UPDATE prds SET 
			deleted_at = $2, 
			updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, id, now)
	if err != nil {
		return fmt.Errorf("failed to delete PRD: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("PRD not found: %s", id)
	}

	return nil
}

// Search performs full-text search on PRDs
func (r *PRDRepository) Search(ctx context.Context, searchQuery string, limit int) ([]*types.EnhancedPRD, error) {
	query := `
		SELECT id, repository, filename, content, parsed_at, task_count,
		       complexity_score, validation_score, version, file_size_bytes,
		       file_hash, content_type, author, last_parsed_version,
		       parse_status, parse_errors, document_type, priority_level,
		       status, metadata, created_at, updated_at, deleted_at,
		       ts_rank(search_vector, plainto_tsquery('english', $1)) as rank
		FROM prds
		WHERE deleted_at IS NULL
		AND search_vector @@ plainto_tsquery('english', $1)
		ORDER BY rank DESC, created_at DESC
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, searchQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search PRDs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var prds []*types.EnhancedPRD
	for rows.Next() {
		prd := &types.EnhancedPRD{}
		var rank float64

		err := rows.Scan(
			&prd.ID, &prd.Repository, &prd.Filename, &prd.Content, &prd.ParsedAt,
			&prd.TaskCount, &prd.ComplexityScore, &prd.ValidationScore, &prd.Version,
			&prd.FileSizeBytes, &prd.FileHash, &prd.ContentType, &prd.Author,
			&prd.LastParsedVersion, &prd.ParseStatus, &prd.ParseErrors,
			&prd.DocumentType, &prd.PriorityLevel, &prd.Status, &prd.Metadata,
			&prd.CreatedAt, &prd.UpdatedAt, &prd.DeletedAt, &rank,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}
		prds = append(prds, prd)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate search results: %w", err)
	}

	return prds, nil
}

// UpdateParseStatus updates the parse status of a PRD
func (r *PRDRepository) UpdateParseStatus(ctx context.Context, id, status string, errors []string) error {
	query := `
		UPDATE prds SET 
			parse_status = $2, 
			parse_errors = $3,
			parsed_at = $4,
			updated_at = $4
		WHERE id = $1 AND deleted_at IS NULL`

	now := time.Now()
	errorJSON := types.JSONArray{}
	for _, e := range errors {
		errorJSON = append(errorJSON, e)
	}

	result, err := r.db.ExecContext(ctx, query, id, status, errorJSON, now)
	if err != nil {
		return fmt.Errorf("failed to update parse status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("PRD not found: %s", id)
	}

	return nil
}

// GetStatistics returns statistics about PRDs
func (r *PRDRepository) GetStatistics(ctx context.Context) (*PRDStatistics, error) {
	query := `
		SELECT 
			COUNT(*) as total_prds,
			COUNT(*) FILTER (WHERE status = 'active') as active_prds,
			COUNT(*) FILTER (WHERE parse_status = 'success') as successfully_parsed,
			COUNT(*) FILTER (WHERE parse_status = 'failed') as failed_parsing,
			AVG(complexity_score) as avg_complexity_score,
			AVG(task_count) as avg_task_count,
			SUM(file_size_bytes) as total_file_size,
			COUNT(DISTINCT repository) as unique_repositories
		FROM prds 
		WHERE deleted_at IS NULL`

	stats := &PRDStatistics{}
	err := r.db.QueryRowContext(ctx, query).Scan(
		&stats.TotalPRDs,
		&stats.ActivePRDs,
		&stats.SuccessfullyParsed,
		&stats.FailedParsing,
		&stats.AvgComplexityScore,
		&stats.AvgTaskCount,
		&stats.TotalFileSize,
		&stats.UniqueRepositories,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get PRD statistics: %w", err)
	}

	return stats, nil
}

// PRDFilters represents filters for PRD queries
type PRDFilters struct {
	Repository    *string    `json:"repository,omitempty"`
	Status        *string    `json:"status,omitempty"`
	DocumentType  *string    `json:"document_type,omitempty"`
	ParseStatus   *string    `json:"parse_status,omitempty"`
	CreatedAfter  *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
	Limit         int        `json:"limit"`
	Offset        int        `json:"offset"`
}

// PRDStatistics represents statistics about PRDs in the database
type PRDStatistics struct {
	TotalPRDs          int64    `json:"total_prds"`
	ActivePRDs         int64    `json:"active_prds"`
	SuccessfullyParsed int64    `json:"successfully_parsed"`
	FailedParsing      int64    `json:"failed_parsing"`
	AvgComplexityScore *float64 `json:"avg_complexity_score"`
	AvgTaskCount       *float64 `json:"avg_task_count"`
	TotalFileSize      *int64   `json:"total_file_size"`
	UniqueRepositories int64    `json:"unique_repositories"`
}
