// Package intelligence provides AI-powered pattern recognition
package intelligence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"

	"lerian-mcp-memory/internal/logging"
)

// SQLPatternStorage implements PatternStorage using PostgreSQL
type SQLPatternStorage struct {
	db     *sql.DB
	logger logging.Logger
}

// NewSQLPatternStorage creates a new SQL-based pattern storage
func NewSQLPatternStorage(db *sql.DB, logger logging.Logger) PatternStorage {
	return &SQLPatternStorage{
		db:     db,
		logger: logger,
	}
}

// StorePattern stores a pattern in the database
func (s *SQLPatternStorage) StorePattern(ctx context.Context, pattern *Pattern) error {
	signatureJSON, err := json.Marshal(pattern.Signature)
	if err != nil {
		return fmt.Errorf("failed to marshal signature: %w", err)
	}

	metadataJSON, err := json.Marshal(pattern.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO patterns (
			id, name, description, pattern_type, category,
			signature, keywords, repository_url, file_patterns,
			language, confidence_score, validation_status,
			occurrence_count, positive_feedback_count, negative_feedback_count,
			last_seen_at, parent_pattern_id, evolution_reason,
			version, metadata
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12,
			$13, $14, $15,
			$16, $17, $18,
			$19, $20
		)`

	_, err = s.db.ExecContext(ctx, query,
		pattern.ID, pattern.Name, pattern.Description, pattern.Type, pattern.Category,
		signatureJSON, pq.Array(pattern.Keywords), pattern.RepositoryURL, pq.Array(pattern.FilePatterns),
		pattern.Language, pattern.ConfidenceScore, pattern.ValidationStatus,
		pattern.OccurrenceCount, pattern.PositiveFeedback, pattern.NegativeFeedback,
		pattern.LastSeenAt, pattern.ParentPatternID, pattern.EvolutionReason,
		pattern.Version, metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to store pattern: %w", err)
	}

	return nil
}

// GetPattern retrieves a pattern by ID
func (s *SQLPatternStorage) GetPattern(ctx context.Context, id string) (*Pattern, error) {
	query := `
		SELECT 
			id, name, description, pattern_type, category,
			signature, keywords, repository_url, file_patterns,
			language, confidence_score, confidence_level, validation_status,
			occurrence_count, positive_feedback_count, negative_feedback_count,
			last_seen_at, parent_pattern_id, evolution_reason,
			version, metadata, created_at, updated_at
		FROM patterns
		WHERE id = $1`

	var pattern Pattern
	var signatureJSON, metadataJSON []byte
	var keywords, filePatterns pq.StringArray

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&pattern.ID, &pattern.Name, &pattern.Description, &pattern.Type, &pattern.Category,
		&signatureJSON, &keywords, &pattern.RepositoryURL, &filePatterns,
		&pattern.Language, &pattern.ConfidenceScore, &pattern.ConfidenceLevel, &pattern.ValidationStatus,
		&pattern.OccurrenceCount, &pattern.PositiveFeedback, &pattern.NegativeFeedback,
		&pattern.LastSeenAt, &pattern.ParentPatternID, &pattern.EvolutionReason,
		&pattern.Version, &metadataJSON, &pattern.CreatedAt, &pattern.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("pattern not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get pattern: %w", err)
	}

	// Unmarshal JSON fields
	if err := json.Unmarshal(signatureJSON, &pattern.Signature); err != nil {
		return nil, fmt.Errorf("failed to unmarshal signature: %w", err)
	}
	if err := json.Unmarshal(metadataJSON, &pattern.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	pattern.Keywords = []string(keywords)
	pattern.FilePatterns = []string(filePatterns)

	return &pattern, nil
}

// ListPatterns lists patterns by type
func (s *SQLPatternStorage) ListPatterns(ctx context.Context, patternType *PatternType) ([]Pattern, error) {
	query := `
		SELECT 
			id, name, description, pattern_type, category,
			signature, keywords, repository_url, file_patterns,
			language, confidence_score, confidence_level, validation_status,
			occurrence_count, positive_feedback_count, negative_feedback_count,
			last_seen_at, parent_pattern_id, evolution_reason,
			version, metadata, created_at, updated_at
		FROM patterns`

	args := []interface{}{}
	if patternType != nil {
		query += " WHERE pattern_type = $1"
		args = append(args, *patternType)
	}
	query += " ORDER BY confidence_score DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list patterns: %w", err)
	}
	defer rows.Close()

	var patterns []Pattern
	for rows.Next() {
		var pattern Pattern
		var signatureJSON, metadataJSON []byte
		var keywords, filePatterns pq.StringArray

		err := rows.Scan(
			&pattern.ID, &pattern.Name, &pattern.Description, &pattern.Type, &pattern.Category,
			&signatureJSON, &keywords, &pattern.RepositoryURL, &filePatterns,
			&pattern.Language, &pattern.ConfidenceScore, &pattern.ConfidenceLevel, &pattern.ValidationStatus,
			&pattern.OccurrenceCount, &pattern.PositiveFeedback, &pattern.NegativeFeedback,
			&pattern.LastSeenAt, &pattern.ParentPatternID, &pattern.EvolutionReason,
			&pattern.Version, &metadataJSON, &pattern.CreatedAt, &pattern.UpdatedAt,
		)
		if err != nil {
			s.logger.Error("Failed to scan pattern", "error", err)
			continue
		}

		// Unmarshal JSON fields
		if err := json.Unmarshal(signatureJSON, &pattern.Signature); err != nil {
			s.logger.Error("Failed to unmarshal signature", "error", err)
			pattern.Signature = make(map[string]interface{})
		}
		if err := json.Unmarshal(metadataJSON, &pattern.Metadata); err != nil {
			s.logger.Error("Failed to unmarshal metadata", "error", err)
			pattern.Metadata = make(map[string]interface{})
		}

		pattern.Keywords = []string(keywords)
		pattern.FilePatterns = []string(filePatterns)

		patterns = append(patterns, pattern)
	}

	return patterns, nil
}

// UpdatePattern updates an existing pattern
func (s *SQLPatternStorage) UpdatePattern(ctx context.Context, pattern *Pattern) error {
	signatureJSON, err := json.Marshal(pattern.Signature)
	if err != nil {
		return fmt.Errorf("failed to marshal signature: %w", err)
	}

	metadataJSON, err := json.Marshal(pattern.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE patterns SET
			name = $2, description = $3, pattern_type = $4, category = $5,
			signature = $6, keywords = $7, repository_url = $8, file_patterns = $9,
			language = $10, confidence_score = $11, validation_status = $12,
			occurrence_count = $13, positive_feedback_count = $14, negative_feedback_count = $15,
			last_seen_at = $16, parent_pattern_id = $17, evolution_reason = $18,
			version = $19, metadata = $20, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`

	result, err := s.db.ExecContext(ctx, query,
		pattern.ID, pattern.Name, pattern.Description, pattern.Type, pattern.Category,
		signatureJSON, pq.Array(pattern.Keywords), pattern.RepositoryURL, pq.Array(pattern.FilePatterns),
		pattern.Language, pattern.ConfidenceScore, pattern.ValidationStatus,
		pattern.OccurrenceCount, pattern.PositiveFeedback, pattern.NegativeFeedback,
		pattern.LastSeenAt, pattern.ParentPatternID, pattern.EvolutionReason,
		pattern.Version, metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to update pattern: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("pattern not found")
	}

	return nil
}

// DeletePattern deletes a pattern by ID
func (s *SQLPatternStorage) DeletePattern(ctx context.Context, id string) error {
	query := "DELETE FROM patterns WHERE id = $1"

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete pattern: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("pattern not found")
	}

	return nil
}

// SearchPatterns searches for patterns using full-text search
func (s *SQLPatternStorage) SearchPatterns(ctx context.Context, query string, limit int) ([]Pattern, error) {
	searchQuery := `
		SELECT 
			id, name, description, pattern_type, category,
			signature, keywords, repository_url, file_patterns,
			language, confidence_score, confidence_level, validation_status,
			occurrence_count, positive_feedback_count, negative_feedback_count,
			last_seen_at, parent_pattern_id, evolution_reason,
			version, metadata, created_at, updated_at,
			ts_rank_cd(
				to_tsvector('english', COALESCE(name, '') || ' ' || COALESCE(description, '') || ' ' || COALESCE(category, '')),
				plainto_tsquery('english', $1)
			) AS rank
		FROM patterns
		WHERE to_tsvector('english', COALESCE(name, '') || ' ' || COALESCE(description, '') || ' ' || COALESCE(category, ''))
			@@ plainto_tsquery('english', $1)
		ORDER BY rank DESC, confidence_score DESC
		LIMIT $2`

	rows, err := s.db.QueryContext(ctx, searchQuery, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search patterns: %w", err)
	}
	defer rows.Close()

	var patterns []Pattern
	for rows.Next() {
		var pattern Pattern
		var signatureJSON, metadataJSON []byte
		var keywords, filePatterns pq.StringArray
		var rank float64

		err := rows.Scan(
			&pattern.ID, &pattern.Name, &pattern.Description, &pattern.Type, &pattern.Category,
			&signatureJSON, &keywords, &pattern.RepositoryURL, &filePatterns,
			&pattern.Language, &pattern.ConfidenceScore, &pattern.ConfidenceLevel, &pattern.ValidationStatus,
			&pattern.OccurrenceCount, &pattern.PositiveFeedback, &pattern.NegativeFeedback,
			&pattern.LastSeenAt, &pattern.ParentPatternID, &pattern.EvolutionReason,
			&pattern.Version, &metadataJSON, &pattern.CreatedAt, &pattern.UpdatedAt,
			&rank,
		)
		if err != nil {
			s.logger.Error("Failed to scan pattern", "error", err)
			continue
		}

		// Unmarshal JSON fields
		if err := json.Unmarshal(signatureJSON, &pattern.Signature); err != nil {
			pattern.Signature = make(map[string]interface{})
		}
		if err := json.Unmarshal(metadataJSON, &pattern.Metadata); err != nil {
			pattern.Metadata = make(map[string]interface{})
		}

		pattern.Keywords = []string(keywords)
		pattern.FilePatterns = []string(filePatterns)

		patterns = append(patterns, pattern)
	}

	return patterns, nil
}

// StoreOccurrence stores a pattern occurrence
func (s *SQLPatternStorage) StoreOccurrence(ctx context.Context, occurrence *PatternOccurrence) error {
	metadataJSON, err := json.Marshal(occurrence.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO pattern_occurrences (
			id, pattern_id, repository_url, file_path,
			line_start, line_end, code_snippet, surrounding_context,
			detection_score, detection_method, session_id, chunk_id,
			metadata
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8,
			$9, $10, $11, $12,
			$13
		)`

	_, err = s.db.ExecContext(ctx, query,
		occurrence.ID, occurrence.PatternID, occurrence.RepositoryURL, occurrence.FilePath,
		occurrence.LineStart, occurrence.LineEnd, occurrence.CodeSnippet, occurrence.SurroundingContext,
		occurrence.DetectionScore, occurrence.DetectionMethod, occurrence.SessionID, occurrence.ChunkID,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to store occurrence: %w", err)
	}

	return nil
}

// GetOccurrences retrieves occurrences for a pattern
func (s *SQLPatternStorage) GetOccurrences(ctx context.Context, patternID string, limit int) ([]PatternOccurrence, error) {
	query := `
		SELECT 
			id, pattern_id, repository_url, file_path,
			line_start, line_end, code_snippet, surrounding_context,
			detection_score, detection_method, session_id, chunk_id,
			metadata, detected_at
		FROM pattern_occurrences
		WHERE pattern_id = $1
		ORDER BY detected_at DESC
		LIMIT $2`

	rows, err := s.db.QueryContext(ctx, query, patternID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get occurrences: %w", err)
	}
	defer rows.Close()

	var occurrences []PatternOccurrence
	for rows.Next() {
		var occurrence PatternOccurrence
		var metadataJSON []byte

		err := rows.Scan(
			&occurrence.ID, &occurrence.PatternID, &occurrence.RepositoryURL, &occurrence.FilePath,
			&occurrence.LineStart, &occurrence.LineEnd, &occurrence.CodeSnippet, &occurrence.SurroundingContext,
			&occurrence.DetectionScore, &occurrence.DetectionMethod, &occurrence.SessionID, &occurrence.ChunkID,
			&metadataJSON, &occurrence.DetectedAt,
		)
		if err != nil {
			s.logger.Error("Failed to scan occurrence", "error", err)
			continue
		}

		if err := json.Unmarshal(metadataJSON, &occurrence.Metadata); err != nil {
			occurrence.Metadata = make(map[string]interface{})
		}

		occurrences = append(occurrences, occurrence)
	}

	return occurrences, nil
}

// StoreRelationship stores a pattern relationship
func (s *SQLPatternStorage) StoreRelationship(ctx context.Context, relationship *PatternRelationship) error {
	examplesJSON, err := json.Marshal(relationship.Examples)
	if err != nil {
		return fmt.Errorf("failed to marshal examples: %w", err)
	}

	metadataJSON, err := json.Marshal(relationship.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO pattern_relationships (
			id, source_pattern_id, target_pattern_id, relationship_type,
			strength, confidence, context, examples, metadata
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8, $9
		)`

	_, err = s.db.ExecContext(ctx, query,
		relationship.ID, relationship.SourcePatternID, relationship.TargetPatternID, relationship.RelationshipType,
		relationship.Strength, relationship.Confidence, relationship.Context, examplesJSON, metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to store relationship: %w", err)
	}

	return nil
}

// GetRelationships retrieves relationships for a pattern
func (s *SQLPatternStorage) GetRelationships(ctx context.Context, patternID string) ([]PatternRelationship, error) {
	query := `
		SELECT 
			id, source_pattern_id, target_pattern_id, relationship_type,
			strength, confidence, context, examples, metadata, created_at
		FROM pattern_relationships
		WHERE source_pattern_id = $1 OR target_pattern_id = $1
		ORDER BY strength DESC`

	rows, err := s.db.QueryContext(ctx, query, patternID)
	if err != nil {
		return nil, fmt.Errorf("failed to get relationships: %w", err)
	}
	defer rows.Close()

	var relationships []PatternRelationship
	for rows.Next() {
		var relationship PatternRelationship
		var examplesJSON, metadataJSON []byte

		err := rows.Scan(
			&relationship.ID, &relationship.SourcePatternID, &relationship.TargetPatternID, &relationship.RelationshipType,
			&relationship.Strength, &relationship.Confidence, &relationship.Context, &examplesJSON, &metadataJSON, &relationship.CreatedAt,
		)
		if err != nil {
			s.logger.Error("Failed to scan relationship", "error", err)
			continue
		}

		if err := json.Unmarshal(examplesJSON, &relationship.Examples); err != nil {
			relationship.Examples = []interface{}{}
		}
		if err := json.Unmarshal(metadataJSON, &relationship.Metadata); err != nil {
			relationship.Metadata = make(map[string]interface{})
		}

		relationships = append(relationships, relationship)
	}

	return relationships, nil
}

// UpdateConfidence updates pattern confidence based on feedback
func (s *SQLPatternStorage) UpdateConfidence(ctx context.Context, patternID string, isPositive bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Call the stored function
	query := "SELECT update_pattern_confidence($1, $2)"
	_, err = tx.ExecContext(ctx, query, patternID, isPositive)
	if err != nil {
		return fmt.Errorf("failed to update confidence: %w", err)
	}

	return tx.Commit()
}

// GetPatternStatistics retrieves pattern statistics
func (s *SQLPatternStorage) GetPatternStatistics(ctx context.Context) (map[string]interface{}, error) {
	// Refresh materialized view
	_, err := s.db.ExecContext(ctx, "SELECT refresh_pattern_statistics()")
	if err != nil {
		s.logger.Error("Failed to refresh pattern statistics", "error", err)
	}

	// Query statistics
	query := `
		SELECT 
			COUNT(DISTINCT id) as total_patterns,
			COUNT(DISTINCT CASE WHEN validation_status = 'validated' THEN id END) as validated_patterns,
			AVG(confidence_score) as avg_confidence,
			SUM(occurrence_count) as total_occurrences,
			COUNT(DISTINCT repository_url) as total_repositories
		FROM patterns`

	var stats struct {
		TotalPatterns     int     `json:"total_patterns"`
		ValidatedPatterns int     `json:"validated_patterns"`
		AvgConfidence     float64 `json:"avg_confidence"`
		TotalOccurrences  int     `json:"total_occurrences"`
		TotalRepositories int     `json:"total_repositories"`
	}

	err = s.db.QueryRowContext(ctx, query).Scan(
		&stats.TotalPatterns,
		&stats.ValidatedPatterns,
		&stats.AvgConfidence,
		&stats.TotalOccurrences,
		&stats.TotalRepositories,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}

	// Pattern type distribution
	typeQuery := `
		SELECT pattern_type, COUNT(*) as count
		FROM patterns
		GROUP BY pattern_type`

	rows, err := s.db.QueryContext(ctx, typeQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get type distribution: %w", err)
	}
	defer rows.Close()

	typeDistribution := make(map[string]int)
	for rows.Next() {
		var patternType string
		var count int
		if err := rows.Scan(&patternType, &count); err != nil {
			continue
		}
		typeDistribution[patternType] = count
	}

	return map[string]interface{}{
		"total_patterns":     stats.TotalPatterns,
		"validated_patterns": stats.ValidatedPatterns,
		"avg_confidence":     stats.AvgConfidence,
		"total_occurrences":  stats.TotalOccurrences,
		"total_repositories": stats.TotalRepositories,
		"type_distribution":  typeDistribution,
		"timestamp":          time.Now(),
	}, nil
}
