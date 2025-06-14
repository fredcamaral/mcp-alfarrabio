// Package storage provides vector database and storage abstractions.
// It includes Qdrant integration, circuit breakers, retry logic, and storage interfaces.
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"lerian-mcp-memory/internal/intelligence"
	"lerian-mcp-memory/internal/logging"
)

// PatternSQLStorage implements PatternStorage using PostgreSQL
type PatternSQLStorage struct {
	db     *sql.DB
	logger logging.Logger
}

// NewPatternSQLStorage creates a new SQL-based pattern storage
func NewPatternSQLStorage(db *sql.DB, logger logging.Logger) intelligence.PatternStorage {
	return &PatternSQLStorage{
		db:     db,
		logger: logger,
	}
}

// scanPatterns is a helper function to scan pattern rows and handle common unmarshaling logic
func (p *PatternSQLStorage) scanPatterns(rows *sql.Rows) ([]intelligence.Pattern, error) {
	var patterns []intelligence.Pattern
	for rows.Next() {
		var pattern intelligence.Pattern
		var signatureBytes, metadataBytes []byte
		var embeddings pq.Float64Array

		err := rows.Scan(
			&pattern.ID,
			&pattern.Type,
			&pattern.Name,
			&pattern.Description,
			&pattern.Category,
			&signatureBytes,
			pq.Array(&pattern.Keywords),
			&pattern.RepositoryURL,
			pq.Array(&pattern.FilePatterns),
			&pattern.Language,
			&pattern.ConfidenceScore,
			&pattern.ConfidenceLevel,
			&pattern.ValidationStatus,
			&pattern.OccurrenceCount,
			&pattern.PositiveFeedback,
			&pattern.NegativeFeedback,
			&pattern.LastSeenAt,
			&pattern.ParentPatternID,
			&pattern.EvolutionReason,
			&pattern.Version,
			&metadataBytes,
			&embeddings,
			&pattern.CreatedAt,
			&pattern.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pattern: %w", err)
		}

		// Unmarshal JSON fields
		if err := json.Unmarshal(signatureBytes, &pattern.Signature); err != nil {
			p.logger.Error("Failed to unmarshal signature", "pattern_id", pattern.ID, "error", err)
			pattern.Signature = make(map[string]interface{})
		}

		if err := json.Unmarshal(metadataBytes, &pattern.Metadata); err != nil {
			p.logger.Error("Failed to unmarshal metadata", "pattern_id", pattern.ID, "error", err)
			pattern.Metadata = make(map[string]interface{})
		}

		pattern.Embeddings = []float64(embeddings)
		patterns = append(patterns, pattern)
	}

	return patterns, rows.Err()
}

// StorePattern stores a new pattern
func (p *PatternSQLStorage) StorePattern(ctx context.Context, pattern *intelligence.Pattern) error {
	query := `
		INSERT INTO patterns (
			id, type, name, description, category, signature, keywords,
			repository_url, file_patterns, language, confidence_score,
			confidence_level, validation_status, occurrence_count,
			positive_feedback, negative_feedback, last_seen_at,
			parent_pattern_id, evolution_reason, version, metadata,
			embeddings, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
			$14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24
		)`

	signatureBytes, err := json.Marshal(pattern.Signature)
	if err != nil {
		return fmt.Errorf("failed to marshal signature: %w", err)
	}

	metadataBytes, err := json.Marshal(pattern.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = p.db.ExecContext(ctx, query,
		pattern.ID,
		string(pattern.Type),
		pattern.Name,
		pattern.Description,
		pattern.Category,
		signatureBytes,
		pq.Array(pattern.Keywords),
		pattern.RepositoryURL,
		pq.Array(pattern.FilePatterns),
		pattern.Language,
		pattern.ConfidenceScore,
		string(pattern.ConfidenceLevel),
		string(pattern.ValidationStatus),
		pattern.OccurrenceCount,
		pattern.PositiveFeedback,
		pattern.NegativeFeedback,
		pattern.LastSeenAt,
		pattern.ParentPatternID,
		pattern.EvolutionReason,
		pattern.Version,
		metadataBytes,
		pq.Array(pattern.Embeddings),
		pattern.CreatedAt,
		pattern.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to store pattern: %w", err)
	}

	p.logger.Info("Stored pattern", "id", pattern.ID, "name", pattern.Name)
	return nil
}

// GetPattern retrieves a pattern by ID
func (p *PatternSQLStorage) GetPattern(ctx context.Context, id string) (*intelligence.Pattern, error) {
	query := `
		SELECT 
			id, type, name, description, category, signature, keywords,
			repository_url, file_patterns, language, confidence_score,
			confidence_level, validation_status, occurrence_count,
			positive_feedback, negative_feedback, last_seen_at,
			parent_pattern_id, evolution_reason, version, metadata,
			embeddings, created_at, updated_at
		FROM patterns WHERE id = $1`

	var pattern intelligence.Pattern
	var signatureBytes, metadataBytes []byte
	var embeddings pq.Float64Array

	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&pattern.ID,
		&pattern.Type,
		&pattern.Name,
		&pattern.Description,
		&pattern.Category,
		&signatureBytes,
		pq.Array(&pattern.Keywords),
		&pattern.RepositoryURL,
		pq.Array(&pattern.FilePatterns),
		&pattern.Language,
		&pattern.ConfidenceScore,
		&pattern.ConfidenceLevel,
		&pattern.ValidationStatus,
		&pattern.OccurrenceCount,
		&pattern.PositiveFeedback,
		&pattern.NegativeFeedback,
		&pattern.LastSeenAt,
		&pattern.ParentPatternID,
		&pattern.EvolutionReason,
		&pattern.Version,
		&metadataBytes,
		&embeddings,
		&pattern.CreatedAt,
		&pattern.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("pattern not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get pattern: %w", err)
	}

	// Unmarshal JSON fields
	if err := json.Unmarshal(signatureBytes, &pattern.Signature); err != nil {
		return nil, fmt.Errorf("failed to unmarshal signature: %w", err)
	}

	if err := json.Unmarshal(metadataBytes, &pattern.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	pattern.Embeddings = []float64(embeddings)

	return &pattern, nil
}

// ListPatterns lists patterns by type
func (p *PatternSQLStorage) ListPatterns(ctx context.Context, patternType *intelligence.PatternType) ([]intelligence.Pattern, error) {
	var query string
	var args []interface{}

	if patternType != nil {
		query = `
			SELECT 
				id, type, name, description, category, signature, keywords,
				repository_url, file_patterns, language, confidence_score,
				confidence_level, validation_status, occurrence_count,
				positive_feedback, negative_feedback, last_seen_at,
				parent_pattern_id, evolution_reason, version, metadata,
				embeddings, created_at, updated_at
			FROM patterns WHERE type = $1
			ORDER BY confidence_score DESC, updated_at DESC`
		args = []interface{}{string(*patternType)}
	} else {
		query = `
			SELECT 
				id, type, name, description, category, signature, keywords,
				repository_url, file_patterns, language, confidence_score,
				confidence_level, validation_status, occurrence_count,
				positive_feedback, negative_feedback, last_seen_at,
				parent_pattern_id, evolution_reason, version, metadata,
				embeddings, created_at, updated_at
			FROM patterns 
			ORDER BY confidence_score DESC, updated_at DESC`
	}

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list patterns: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close rows: %v\n", closeErr)
		}
	}()

	return p.scanPatterns(rows)
}

// UpdatePattern updates an existing pattern
func (p *PatternSQLStorage) UpdatePattern(ctx context.Context, pattern *intelligence.Pattern) error {
	query := `
		UPDATE patterns SET
			type = $2, name = $3, description = $4, category = $5, 
			signature = $6, keywords = $7, repository_url = $8,
			file_patterns = $9, language = $10, confidence_score = $11,
			confidence_level = $12, validation_status = $13,
			occurrence_count = $14, positive_feedback = $15,
			negative_feedback = $16, last_seen_at = $17,
			parent_pattern_id = $18, evolution_reason = $19,
			version = $20, metadata = $21, embeddings = $22,
			updated_at = $23
		WHERE id = $1`

	signatureBytes, err := json.Marshal(pattern.Signature)
	if err != nil {
		return fmt.Errorf("failed to marshal signature: %w", err)
	}

	metadataBytes, err := json.Marshal(pattern.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	result, err := p.db.ExecContext(ctx, query,
		pattern.ID,
		string(pattern.Type),
		pattern.Name,
		pattern.Description,
		pattern.Category,
		signatureBytes,
		pq.Array(pattern.Keywords),
		pattern.RepositoryURL,
		pq.Array(pattern.FilePatterns),
		pattern.Language,
		pattern.ConfidenceScore,
		string(pattern.ConfidenceLevel),
		string(pattern.ValidationStatus),
		pattern.OccurrenceCount,
		pattern.PositiveFeedback,
		pattern.NegativeFeedback,
		pattern.LastSeenAt,
		pattern.ParentPatternID,
		pattern.EvolutionReason,
		pattern.Version,
		metadataBytes,
		pq.Array(pattern.Embeddings),
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to update pattern: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("pattern not found: %s", pattern.ID)
	}

	p.logger.Info("Updated pattern", "id", pattern.ID, "name", pattern.Name)
	return nil
}

// DeletePattern deletes a pattern by ID
func (p *PatternSQLStorage) DeletePattern(ctx context.Context, id string) error {
	query := `DELETE FROM patterns WHERE id = $1`

	result, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete pattern: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("pattern not found: %s", id)
	}

	p.logger.Info("Deleted pattern", "id", id)
	return nil
}

// SearchPatterns searches for patterns by query
func (p *PatternSQLStorage) SearchPatterns(ctx context.Context, query string, limit int) ([]intelligence.Pattern, error) {
	sqlQuery := `
		SELECT 
			id, type, name, description, category, signature, keywords,
			repository_url, file_patterns, language, confidence_score,
			confidence_level, validation_status, occurrence_count,
			positive_feedback, negative_feedback, last_seen_at,
			parent_pattern_id, evolution_reason, version, metadata,
			embeddings, created_at, updated_at
		FROM patterns 
		WHERE 
			name ILIKE $1 OR 
			description ILIKE $1 OR 
			$1 = ANY(keywords) OR
			category ILIKE $1
		ORDER BY confidence_score DESC, updated_at DESC
		LIMIT $2`

	searchPattern := "%" + query + "%"
	rows, err := p.db.QueryContext(ctx, sqlQuery, searchPattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search patterns: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close rows: %v\n", closeErr)
		}
	}()

	return p.scanPatterns(rows)
}

// StoreOccurrence stores a pattern occurrence
func (p *PatternSQLStorage) StoreOccurrence(ctx context.Context, occurrence *intelligence.PatternOccurrence) error {
	query := `
		INSERT INTO pattern_occurrences (
			id, pattern_id, repository_url, file_path, line_start, line_end,
			code_snippet, surrounding_context, detection_score, detection_method,
			session_id, chunk_id, metadata, detected_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)`

	metadataBytes, err := json.Marshal(occurrence.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = p.db.ExecContext(ctx, query,
		occurrence.ID,
		occurrence.PatternID,
		occurrence.RepositoryURL,
		occurrence.FilePath,
		occurrence.LineStart,
		occurrence.LineEnd,
		occurrence.CodeSnippet,
		occurrence.SurroundingContext,
		occurrence.DetectionScore,
		occurrence.DetectionMethod,
		occurrence.SessionID,
		occurrence.ChunkID,
		metadataBytes,
		occurrence.DetectedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to store occurrence: %w", err)
	}

	p.logger.Info("Stored pattern occurrence", "id", occurrence.ID, "pattern_id", occurrence.PatternID)
	return nil
}

// GetOccurrences retrieves occurrences for a pattern
func (p *PatternSQLStorage) GetOccurrences(ctx context.Context, patternID string, limit int) ([]intelligence.PatternOccurrence, error) {
	query := `
		SELECT 
			id, pattern_id, repository_url, file_path, line_start, line_end,
			code_snippet, surrounding_context, detection_score, detection_method,
			session_id, chunk_id, metadata, detected_at
		FROM pattern_occurrences 
		WHERE pattern_id = $1
		ORDER BY detected_at DESC
		LIMIT $2`

	rows, err := p.db.QueryContext(ctx, query, patternID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get occurrences: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close rows: %v\n", closeErr)
		}
	}()

	var occurrences []intelligence.PatternOccurrence
	for rows.Next() {
		var occurrence intelligence.PatternOccurrence
		var metadataBytes []byte

		err := rows.Scan(
			&occurrence.ID,
			&occurrence.PatternID,
			&occurrence.RepositoryURL,
			&occurrence.FilePath,
			&occurrence.LineStart,
			&occurrence.LineEnd,
			&occurrence.CodeSnippet,
			&occurrence.SurroundingContext,
			&occurrence.DetectionScore,
			&occurrence.DetectionMethod,
			&occurrence.SessionID,
			&occurrence.ChunkID,
			&metadataBytes,
			&occurrence.DetectedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan occurrence: %w", err)
		}

		if err := json.Unmarshal(metadataBytes, &occurrence.Metadata); err != nil {
			p.logger.Error("Failed to unmarshal metadata", "occurrence_id", occurrence.ID, "error", err)
			occurrence.Metadata = make(map[string]interface{})
		}

		occurrences = append(occurrences, occurrence)
	}

	return occurrences, rows.Err()
}

// StoreRelationship stores a pattern relationship
func (p *PatternSQLStorage) StoreRelationship(ctx context.Context, relationship *intelligence.PatternRelationship) error {
	if relationship.ID == "" {
		relationship.ID = uuid.New().String()
	}

	query := `
		INSERT INTO pattern_relationships (
			id, source_pattern_id, target_pattern_id, relationship_type,
			strength, confidence, context, examples, metadata, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		) ON CONFLICT (id) DO UPDATE SET
			strength = EXCLUDED.strength,
			confidence = EXCLUDED.confidence,
			context = EXCLUDED.context,
			examples = EXCLUDED.examples,
			metadata = EXCLUDED.metadata`

	examplesBytes, err := json.Marshal(relationship.Examples)
	if err != nil {
		return fmt.Errorf("failed to marshal examples: %w", err)
	}

	metadataBytes, err := json.Marshal(relationship.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = p.db.ExecContext(ctx, query,
		relationship.ID,
		relationship.SourcePatternID,
		relationship.TargetPatternID,
		relationship.RelationshipType,
		relationship.Strength,
		relationship.Confidence,
		relationship.Context,
		examplesBytes,
		metadataBytes,
		relationship.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to store relationship: %w", err)
	}

	p.logger.Info("Stored pattern relationship", "id", relationship.ID,
		"source", relationship.SourcePatternID, "target", relationship.TargetPatternID)
	return nil
}

// GetRelationships retrieves relationships for a pattern
func (p *PatternSQLStorage) GetRelationships(ctx context.Context, patternID string) ([]intelligence.PatternRelationship, error) {
	query := `
		SELECT 
			id, source_pattern_id, target_pattern_id, relationship_type,
			strength, confidence, context, examples, metadata, created_at
		FROM pattern_relationships 
		WHERE source_pattern_id = $1 OR target_pattern_id = $1
		ORDER BY strength DESC, created_at DESC`

	rows, err := p.db.QueryContext(ctx, query, patternID)
	if err != nil {
		return nil, fmt.Errorf("failed to get relationships: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close rows: %v\n", closeErr)
		}
	}()

	var relationships []intelligence.PatternRelationship
	for rows.Next() {
		var relationship intelligence.PatternRelationship
		var examplesBytes, metadataBytes []byte

		err := rows.Scan(
			&relationship.ID,
			&relationship.SourcePatternID,
			&relationship.TargetPatternID,
			&relationship.RelationshipType,
			&relationship.Strength,
			&relationship.Confidence,
			&relationship.Context,
			&examplesBytes,
			&metadataBytes,
			&relationship.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan relationship: %w", err)
		}

		if err := json.Unmarshal(examplesBytes, &relationship.Examples); err != nil {
			p.logger.Error("Failed to unmarshal examples", "relationship_id", relationship.ID, "error", err)
			relationship.Examples = []interface{}{}
		}

		if err := json.Unmarshal(metadataBytes, &relationship.Metadata); err != nil {
			p.logger.Error("Failed to unmarshal metadata", "relationship_id", relationship.ID, "error", err)
			relationship.Metadata = make(map[string]interface{})
		}

		relationships = append(relationships, relationship)
	}

	return relationships, rows.Err()
}

// UpdateConfidence updates pattern confidence based on feedback
func (p *PatternSQLStorage) UpdateConfidence(ctx context.Context, patternID string, isPositive bool) error {
	var query string
	if isPositive {
		query = `
			UPDATE patterns 
			SET 
				positive_feedback = positive_feedback + 1,
				confidence_score = (positive_feedback + 1.0) / (positive_feedback + negative_feedback + 2.0),
				confidence_level = CASE
					WHEN (positive_feedback + 1.0) / (positive_feedback + negative_feedback + 2.0) >= 0.9 THEN 'very_high'
					WHEN (positive_feedback + 1.0) / (positive_feedback + negative_feedback + 2.0) >= 0.75 THEN 'high'
					WHEN (positive_feedback + 1.0) / (positive_feedback + negative_feedback + 2.0) >= 0.5 THEN 'medium'
					WHEN (positive_feedback + 1.0) / (positive_feedback + negative_feedback + 2.0) >= 0.25 THEN 'low'
					ELSE 'very_low'
				END,
				updated_at = $2
			WHERE id = $1`
	} else {
		query = `
			UPDATE patterns 
			SET 
				negative_feedback = negative_feedback + 1,
				confidence_score = (positive_feedback + 1.0) / (positive_feedback + negative_feedback + 2.0),
				confidence_level = CASE
					WHEN (positive_feedback + 1.0) / (positive_feedback + negative_feedback + 2.0) >= 0.9 THEN 'very_high'
					WHEN (positive_feedback + 1.0) / (positive_feedback + negative_feedback + 2.0) >= 0.75 THEN 'high'
					WHEN (positive_feedback + 1.0) / (positive_feedback + negative_feedback + 2.0) >= 0.5 THEN 'medium'
					WHEN (positive_feedback + 1.0) / (positive_feedback + negative_feedback + 2.0) >= 0.25 THEN 'low'
					ELSE 'very_low'
				END,
				updated_at = $2
			WHERE id = $1`
	}

	result, err := p.db.ExecContext(ctx, query, patternID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update confidence: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("pattern not found: %s", patternID)
	}

	p.logger.Info("Updated pattern confidence", "pattern_id", patternID, "positive", isPositive)
	return nil
}

// GetPatternStatistics retrieves pattern statistics
func (p *PatternSQLStorage) GetPatternStatistics(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	if err := p.addDistributionStatistics(ctx, stats); err != nil {
		return nil, err
	}

	p.addTotalCounts(ctx, stats)
	p.addAverageConfidence(ctx, stats)

	if err := p.addTopPatterns(ctx, stats); err != nil {
		return nil, err
	}

	return stats, nil
}

// addDistributionStatistics adds pattern distribution statistics
func (p *PatternSQLStorage) addDistributionStatistics(ctx context.Context, stats map[string]interface{}) error {
	if err := p.addPatternsByType(ctx, stats); err != nil {
		return err
	}

	if err := p.addPatternsByConfidence(ctx, stats); err != nil {
		return err
	}

	if err := p.addPatternsByValidation(ctx, stats); err != nil {
		return err
	}

	return nil
}

// addPatternsByType adds patterns by type statistics
func (p *PatternSQLStorage) addPatternsByType(ctx context.Context, stats map[string]interface{}) error {
	query := `SELECT type, COUNT(*) as count FROM patterns GROUP BY type`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to get type statistics: %w", err)
	}
	defer p.closeRows(rows, "type stats")

	typeStats := make(map[string]int)
	for rows.Next() {
		var patternType string
		var count int
		if err := rows.Scan(&patternType, &count); err != nil {
			return fmt.Errorf("failed to scan type stats: %w", err)
		}
		typeStats[patternType] = count
	}

	stats["patterns_by_type"] = typeStats
	return nil
}

// addPatternsByConfidence adds patterns by confidence level statistics
func (p *PatternSQLStorage) addPatternsByConfidence(ctx context.Context, stats map[string]interface{}) error {
	query := `SELECT confidence_level, COUNT(*) as count FROM patterns GROUP BY confidence_level`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to get confidence statistics: %w", err)
	}
	defer p.closeRows(rows, "confidence stats")

	confidenceStats := make(map[string]int)
	for rows.Next() {
		var level string
		var count int
		if err := rows.Scan(&level, &count); err != nil {
			return fmt.Errorf("failed to scan confidence stats: %w", err)
		}
		confidenceStats[level] = count
	}

	stats["patterns_by_confidence"] = confidenceStats
	return nil
}

// addPatternsByValidation adds patterns by validation status statistics
func (p *PatternSQLStorage) addPatternsByValidation(ctx context.Context, stats map[string]interface{}) error {
	query := `SELECT validation_status, COUNT(*) as count FROM patterns GROUP BY validation_status`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to get validation statistics: %w", err)
	}
	defer p.closeRows(rows, "validation stats")

	validationStats := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return fmt.Errorf("failed to scan validation stats: %w", err)
		}
		validationStats[status] = count
	}

	stats["patterns_by_validation"] = validationStats
	return nil
}

// addTotalCounts adds total count statistics
func (p *PatternSQLStorage) addTotalCounts(ctx context.Context, stats map[string]interface{}) {
	stats["total_patterns"] = p.getCountSafely(ctx, "SELECT COUNT(*) FROM patterns", "patterns")
	stats["total_occurrences"] = p.getCountSafely(ctx, "SELECT COUNT(*) FROM pattern_occurrences", "occurrences")
	stats["total_relationships"] = p.getCountSafely(ctx, "SELECT COUNT(*) FROM pattern_relationships", "relationships")
}

// getCountSafely safely gets a count with error logging
func (p *PatternSQLStorage) getCountSafely(ctx context.Context, query, description string) int {
	var count int
	if err := p.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		p.logger.Error("Failed to count "+description, "error", err)
		return 0
	}
	return count
}

// addAverageConfidence adds average confidence statistics
func (p *PatternSQLStorage) addAverageConfidence(ctx context.Context, stats map[string]interface{}) {
	var avgConfidence sql.NullFloat64
	if err := p.db.QueryRowContext(ctx, "SELECT AVG(confidence_score) FROM patterns").Scan(&avgConfidence); err != nil {
		p.logger.Error("Failed to get average confidence", "error", err)
		return
	}

	if avgConfidence.Valid {
		stats["average_confidence"] = avgConfidence.Float64
	}
}

// addTopPatterns adds top patterns statistics
func (p *PatternSQLStorage) addTopPatterns(ctx context.Context, stats map[string]interface{}) error {
	query := `
		SELECT name, occurrence_count, confidence_score 
		FROM patterns 
		ORDER BY occurrence_count DESC 
		LIMIT 10`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to get top patterns: %w", err)
	}
	defer p.closeRows(rows, "top patterns")

	var topPatterns []map[string]interface{}
	for rows.Next() {
		var name string
		var occurrences int
		var confidence float64
		if err := rows.Scan(&name, &occurrences, &confidence); err != nil {
			return fmt.Errorf("failed to scan top patterns: %w", err)
		}

		topPatterns = append(topPatterns, map[string]interface{}{
			"name":        name,
			"occurrences": occurrences,
			"confidence":  confidence,
		})
	}

	stats["top_patterns"] = topPatterns
	return nil
}

// closeRows safely closes database rows with logging
func (p *PatternSQLStorage) closeRows(rows *sql.Rows, description string) {
	if closeErr := rows.Close(); closeErr != nil {
		p.logger.Error("Failed to close rows", "description", description, "error", closeErr)
	}
}
