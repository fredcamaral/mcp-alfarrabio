// Package storage provides PostgreSQL-backed ContentStore implementation
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"lerian-mcp-memory/internal/types"

	"github.com/lib/pq"
)

// PostgreSQLContentStore implements ContentStore interface using PostgreSQL
type PostgreSQLContentStore struct {
	db *sql.DB
}

// NewPostgreSQLContentStore creates a new PostgreSQL-backed content store
func NewPostgreSQLContentStore(db *sql.DB) *PostgreSQLContentStore {
	return &PostgreSQLContentStore{
		db: db,
	}
}

// Store content with proper project isolation
func (s *PostgreSQLContentStore) Store(ctx context.Context, content *types.Content) error {
	if err := s.validateContent(content); err != nil {
		return fmt.Errorf("invalid content: %w", err)
	}

	// Serialize metadata and tags
	metadataJSON, err := json.Marshal(content.Metadata)
	if err != nil {
		return fmt.Errorf("failed to serialize metadata: %w", err)
	}

	tagsJSON, err := json.Marshal(content.Tags)
	if err != nil {
		return fmt.Errorf("failed to serialize tags: %w", err)
	}

	embeddingsJSON, err := json.Marshal(content.Embeddings)
	if err != nil {
		return fmt.Errorf("failed to serialize embeddings: %w", err)
	}

	query := `
		INSERT INTO content_store (
			id, project_id, session_id, type, title, content, summary,
			tags, metadata, embeddings, created_at, updated_at, accessed_at,
			quality, confidence, parent_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)
		ON CONFLICT (id, project_id) 
		DO UPDATE SET
			session_id = EXCLUDED.session_id,
			type = EXCLUDED.type,
			title = EXCLUDED.title,
			content = EXCLUDED.content,
			summary = EXCLUDED.summary,
			tags = EXCLUDED.tags,
			metadata = EXCLUDED.metadata,
			embeddings = EXCLUDED.embeddings,
			updated_at = EXCLUDED.updated_at,
			accessed_at = EXCLUDED.accessed_at,
			quality = EXCLUDED.quality,
			confidence = EXCLUDED.confidence,
			parent_id = EXCLUDED.parent_id
	`

	_, err = s.db.ExecContext(ctx, query,
		content.ID,
		string(content.ProjectID),
		string(content.SessionID),
		content.Type,
		content.Title,
		content.Content,
		content.Summary,
		string(tagsJSON),
		string(metadataJSON),
		string(embeddingsJSON),
		content.CreatedAt,
		content.UpdatedAt,
		content.AccessedAt,
		content.Quality,
		content.Confidence,
		content.ParentID,
	)

	if err != nil {
		return fmt.Errorf("failed to store content: %w", err)
	}

	return nil
}

// Update existing content
func (s *PostgreSQLContentStore) Update(ctx context.Context, content *types.Content) error {
	if err := s.validateContent(content); err != nil {
		return fmt.Errorf("invalid content: %w", err)
	}

	// Serialize metadata and tags
	metadataJSON, err := json.Marshal(content.Metadata)
	if err != nil {
		return fmt.Errorf("failed to serialize metadata: %w", err)
	}

	tagsJSON, err := json.Marshal(content.Tags)
	if err != nil {
		return fmt.Errorf("failed to serialize tags: %w", err)
	}

	embeddingsJSON, err := json.Marshal(content.Embeddings)
	if err != nil {
		return fmt.Errorf("failed to serialize embeddings: %w", err)
	}

	query := `
		UPDATE content_store SET
			session_id = $3,
			type = $4,
			title = $5,
			content = $6,
			summary = $7,
			tags = $8,
			metadata = $9,
			embeddings = $10,
			updated_at = $11,
			accessed_at = $12,
			quality = $13,
			confidence = $14,
			parent_id = $15
		WHERE id = $1 AND project_id = $2
	`

	result, err := s.db.ExecContext(ctx, query,
		content.ID,
		string(content.ProjectID),
		string(content.SessionID),
		content.Type,
		content.Title,
		content.Content,
		content.Summary,
		string(tagsJSON),
		string(metadataJSON),
		string(embeddingsJSON),
		content.UpdatedAt,
		content.AccessedAt,
		content.Quality,
		content.Confidence,
		content.ParentID,
	)

	if err != nil {
		return fmt.Errorf("failed to update content: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check update result: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("content not found: id=%s, project_id=%s", content.ID, content.ProjectID)
	}

	return nil
}

// Delete content by project and content ID
func (s *PostgreSQLContentStore) Delete(ctx context.Context, projectID types.ProjectID, contentID string) error {
	query := `DELETE FROM content_store WHERE id = $1 AND project_id = $2`

	result, err := s.db.ExecContext(ctx, query, contentID, string(projectID))
	if err != nil {
		return fmt.Errorf("failed to delete content: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check delete result: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("content not found: id=%s, project_id=%s", contentID, projectID)
	}

	return nil
}

// Get content by project and content ID
func (s *PostgreSQLContentStore) Get(ctx context.Context, projectID types.ProjectID, contentID string) (*types.Content, error) {
	query := `
		SELECT 
			id, project_id, session_id, type, title, content, summary,
			tags, metadata, embeddings, created_at, updated_at, accessed_at,
			quality, confidence, parent_id
		FROM content_store 
		WHERE id = $1 AND project_id = $2
	`

	row := s.db.QueryRowContext(ctx, query, contentID, string(projectID))

	var content types.Content
	var sessionIDStr, tagsJSON, metadataJSON, embeddingsJSON sql.NullString
	var title, summary, parentID sql.NullString
	var accessedAt sql.NullTime
	var quality, confidence sql.NullFloat64

	err := row.Scan(
		&content.ID,
		&content.ProjectID,
		&sessionIDStr,
		&content.Type,
		&title,
		&content.Content,
		&summary,
		&tagsJSON,
		&metadataJSON,
		&embeddingsJSON,
		&content.CreatedAt,
		&content.UpdatedAt,
		&accessedAt,
		&quality,
		&confidence,
		&parentID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("content not found: id=%s, project_id=%s", contentID, projectID)
		}
		return nil, fmt.Errorf("failed to get content: %w", err)
	}

	// Handle nullable fields
	if sessionIDStr.Valid {
		content.SessionID = types.SessionID(sessionIDStr.String)
	}
	if title.Valid {
		content.Title = title.String
	}
	if summary.Valid {
		content.Summary = summary.String
	}
	if parentID.Valid {
		content.ParentID = parentID.String
	}
	if accessedAt.Valid {
		content.AccessedAt = &accessedAt.Time
	}
	if quality.Valid {
		content.Quality = quality.Float64
	}
	if confidence.Valid {
		content.Confidence = confidence.Float64
	}

	// Deserialize JSON fields
	if tagsJSON.Valid && tagsJSON.String != "" {
		if err := json.Unmarshal([]byte(tagsJSON.String), &content.Tags); err != nil {
			return nil, fmt.Errorf("failed to deserialize tags: %w", err)
		}
	}

	if metadataJSON.Valid && metadataJSON.String != "" {
		if err := json.Unmarshal([]byte(metadataJSON.String), &content.Metadata); err != nil {
			return nil, fmt.Errorf("failed to deserialize metadata: %w", err)
		}
	}

	if embeddingsJSON.Valid && embeddingsJSON.String != "" {
		if err := json.Unmarshal([]byte(embeddingsJSON.String), &content.Embeddings); err != nil {
			return nil, fmt.Errorf("failed to deserialize embeddings: %w", err)
		}
	}

	return &content, nil
}

// BatchStore stores multiple content items efficiently
func (s *PostgreSQLContentStore) BatchStore(ctx context.Context, contents []*types.Content) (*BatchResult, error) {
	if len(contents) == 0 {
		return &BatchResult{Success: 0, Failed: 0}, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	result := &BatchResult{
		Success:      0,
		Failed:       0,
		Errors:       []BatchError{},
		ProcessedIDs: make([]string, 0, len(contents)),
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO content_store (
			id, project_id, session_id, type, title, content, summary,
			tags, metadata, embeddings, created_at, updated_at, accessed_at,
			quality, confidence, parent_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)
		ON CONFLICT (id, project_id) 
		DO UPDATE SET
			content = EXCLUDED.content,
			summary = EXCLUDED.summary,
			updated_at = EXCLUDED.updated_at
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for i, content := range contents {
		result.ProcessedIDs = append(result.ProcessedIDs, content.ID)

		if err := s.validateContent(content); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, BatchError{
				Index: i,
				ID:    content.ID,
				Error: fmt.Sprintf("validation failed: %v", err),
			})
			continue
		}

		// Serialize JSON fields
		metadataJSON, _ := json.Marshal(content.Metadata)
		tagsJSON, _ := json.Marshal(content.Tags)
		embeddingsJSON, _ := json.Marshal(content.Embeddings)

		_, err := stmt.ExecContext(ctx,
			content.ID,
			string(content.ProjectID),
			string(content.SessionID),
			content.Type,
			content.Title,
			content.Content,
			content.Summary,
			string(tagsJSON),
			string(metadataJSON),
			string(embeddingsJSON),
			content.CreatedAt,
			content.UpdatedAt,
			content.AccessedAt,
			content.Quality,
			content.Confidence,
			content.ParentID,
		)

		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, BatchError{
				Index: i,
				ID:    content.ID,
				Error: err.Error(),
			})
		} else {
			result.Success++
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return result, nil
}

// BatchUpdate updates multiple content items
func (s *PostgreSQLContentStore) BatchUpdate(ctx context.Context, contents []*types.Content) (*BatchResult, error) {
	if len(contents) == 0 {
		return &BatchResult{Success: 0, Failed: 0}, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	result := &BatchResult{
		Success:      0,
		Failed:       0,
		Errors:       []BatchError{},
		ProcessedIDs: make([]string, 0, len(contents)),
	}

	for i, content := range contents {
		result.ProcessedIDs = append(result.ProcessedIDs, content.ID)

		// Use the single Update method for each content
		if err := s.updateContentInTx(ctx, tx, content); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, BatchError{
				Index: i,
				ID:    content.ID,
				Error: err.Error(),
			})
		} else {
			result.Success++
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return result, nil
}

// BatchDelete deletes multiple content items
func (s *PostgreSQLContentStore) BatchDelete(ctx context.Context, projectID types.ProjectID, contentIDs []string) (*BatchResult, error) {
	if len(contentIDs) == 0 {
		return &BatchResult{Success: 0, Failed: 0}, nil
	}

	// Use PostgreSQL array for efficient batch delete
	query := `DELETE FROM content_store WHERE project_id = $1 AND id = ANY($2)`

	result, err := s.db.ExecContext(ctx, query, string(projectID), pq.Array(contentIDs))
	if err != nil {
		return nil, fmt.Errorf("failed to batch delete: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check delete result: %w", err)
	}

	batchResult := &BatchResult{
		Success:      int(rowsAffected),
		Failed:       len(contentIDs) - int(rowsAffected),
		ProcessedIDs: contentIDs,
		Errors:       []BatchError{},
	}

	// If some deletes failed, we can't easily determine which ones
	// In a more sophisticated implementation, we could do individual deletes
	if batchResult.Failed > 0 {
		batchResult.Errors = append(batchResult.Errors, BatchError{
			Index: -1, // Unknown which specific IDs failed
			Error: fmt.Sprintf("%d out of %d deletes failed - some IDs may not exist", batchResult.Failed, len(contentIDs)),
		})
	}

	return batchResult, nil
}

// Helper method to update content within a transaction
func (s *PostgreSQLContentStore) updateContentInTx(ctx context.Context, tx *sql.Tx, content *types.Content) error {
	if err := s.validateContent(content); err != nil {
		return fmt.Errorf("invalid content: %w", err)
	}

	// Serialize JSON fields
	metadataJSON, err := json.Marshal(content.Metadata)
	if err != nil {
		return fmt.Errorf("failed to serialize metadata: %w", err)
	}

	tagsJSON, err := json.Marshal(content.Tags)
	if err != nil {
		return fmt.Errorf("failed to serialize tags: %w", err)
	}

	embeddingsJSON, err := json.Marshal(content.Embeddings)
	if err != nil {
		return fmt.Errorf("failed to serialize embeddings: %w", err)
	}

	query := `
		UPDATE content_store SET
			session_id = $3,
			type = $4,
			title = $5,
			content = $6,
			summary = $7,
			tags = $8,
			metadata = $9,
			embeddings = $10,
			updated_at = $11,
			accessed_at = $12,
			quality = $13,
			confidence = $14,
			parent_id = $15
		WHERE id = $1 AND project_id = $2
	`

	result, err := tx.ExecContext(ctx, query,
		content.ID,
		string(content.ProjectID),
		string(content.SessionID),
		content.Type,
		content.Title,
		content.Content,
		content.Summary,
		string(tagsJSON),
		string(metadataJSON),
		string(embeddingsJSON),
		content.UpdatedAt,
		content.AccessedAt,
		content.Quality,
		content.Confidence,
		content.ParentID,
	)

	if err != nil {
		return fmt.Errorf("failed to update content: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check update result: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("content not found: id=%s, project_id=%s", content.ID, content.ProjectID)
	}

	return nil
}

// validateContent validates content before storing
func (s *PostgreSQLContentStore) validateContent(content *types.Content) error {
	if content == nil {
		return fmt.Errorf("content cannot be nil")
	}

	if content.ID == "" {
		return fmt.Errorf("content ID cannot be empty")
	}

	if err := content.ProjectID.Validate(); err != nil {
		return fmt.Errorf("invalid project ID: %w", err)
	}

	if content.Type == "" {
		return fmt.Errorf("content type cannot be empty")
	}

	if content.Content == "" {
		return fmt.Errorf("content body cannot be empty")
	}

	if content.CreatedAt.IsZero() {
		content.CreatedAt = time.Now()
	}

	if content.UpdatedAt.IsZero() {
		content.UpdatedAt = time.Now()
	}

	return nil
}
