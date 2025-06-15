package learning

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"lerian-mcp-memory-cli/internal/domain/constants"
	"lerian-mcp-memory-cli/internal/domain/entities"
)

// PatternAggregator interface for aggregating patterns across repositories
type PatternAggregator interface {
	AggregatePatterns(ctx context.Context, patterns []*entities.TaskPattern) ([]*entities.AggregatedPattern, error)
	AnonymizePattern(ctx context.Context, pattern *entities.TaskPattern, settings *entities.PrivacySettings) (*entities.TaskPattern, error)
	MergeAggregatedPattern(ctx context.Context, existing *entities.AggregatedPattern, newPattern *entities.TaskPattern, weight float64) error
	ValidatePrivacyCompliance(ctx context.Context, pattern *entities.TaskPattern, settings *entities.PrivacySettings) error
	GeneratePatternSignature(pattern *entities.TaskPattern) string
	FilterSensitiveContent(content string, settings *entities.PrivacySettings) string
}

// AggregationConfig holds configuration for pattern aggregation
type AggregationConfig struct {
	MinSourceCount             int           `json:"min_source_count"`
	MinConfidenceThreshold     float64       `json:"min_confidence_threshold"`
	MaxPatternAge              time.Duration `json:"max_pattern_age"`
	AnonymizationStrength      string        `json:"anonymization_strength"` // "basic", "medium", "high"
	SensitiveKeywords          []string      `json:"sensitive_keywords"`
	PatternSimilarityThreshold float64       `json:"pattern_similarity_threshold"`
	EnableFrequencyAnalysis    bool          `json:"enable_frequency_analysis"`
	EnableTemporalAnalysis     bool          `json:"enable_temporal_analysis"`
}

// DefaultAggregationConfig returns default configuration
func DefaultAggregationConfig() *AggregationConfig {
	return &AggregationConfig{
		MinSourceCount:         3,
		MinConfidenceThreshold: 0.6,
		MaxPatternAge:          90 * 24 * time.Hour,
		AnonymizationStrength:  "medium",
		SensitiveKeywords: []string{
			"password", "secret", "key", "token", "private", "confidential",
			"internal", "proprietary", "classified", "restricted", "sensitive",
			"api_key", "access_token", "auth_token", "session", "credential",
			"email", "phone", "address", "ssn", "credit_card", "payment",
		},
		PatternSimilarityThreshold: 0.8,
		EnableFrequencyAnalysis:    true,
		EnableTemporalAnalysis:     true,
	}
}

// PatternAggregatorDependencies holds dependencies for pattern aggregator
type PatternAggregatorDependencies struct {
	Config             *AggregationConfig
	Logger             *slog.Logger
	SimilarityAnalyzer SimilarityAnalyzer
}

// patternAggregatorImpl implements the PatternAggregator interface
type patternAggregatorImpl struct {
	config             *AggregationConfig
	logger             *slog.Logger
	similarityAnalyzer SimilarityAnalyzer
	patternCache       map[string]*entities.AggregatedPattern
}

// NewPatternAggregator creates a new pattern aggregator
func NewPatternAggregator(deps PatternAggregatorDependencies) PatternAggregator {
	if deps.Config == nil {
		deps.Config = DefaultAggregationConfig()
	}

	return &patternAggregatorImpl{
		config:             deps.Config,
		logger:             deps.Logger,
		similarityAnalyzer: deps.SimilarityAnalyzer,
		patternCache:       make(map[string]*entities.AggregatedPattern),
	}
}

// AggregatePatterns aggregates multiple task patterns into consolidated patterns
func (pa *patternAggregatorImpl) AggregatePatterns(
	ctx context.Context,
	patterns []*entities.TaskPattern,
) ([]*entities.AggregatedPattern, error) {
	pa.logger.Info("aggregating patterns", slog.Int("pattern_count", len(patterns)))

	// Group patterns by similarity
	patternGroups := pa.groupSimilarPatterns(patterns)

	var aggregatedPatterns []*entities.AggregatedPattern

	for groupKey, groupPatterns := range patternGroups {
		if len(groupPatterns) < pa.config.MinSourceCount {
			pa.logger.Debug("skipping group with insufficient sources",
				slog.String("group", groupKey),
				slog.Int("source_count", len(groupPatterns)))
			continue
		}

		aggregated, err := pa.createAggregatedPattern(ctx, groupPatterns)
		if err != nil {
			pa.logger.Error("failed to create aggregated pattern",
				slog.String("group", groupKey),
				slog.Any("error", err))
			continue
		}

		if aggregated.Confidence >= pa.config.MinConfidenceThreshold {
			aggregatedPatterns = append(aggregatedPatterns, aggregated)
		}
	}

	pa.logger.Info("pattern aggregation completed",
		slog.Int("input_patterns", len(patterns)),
		slog.Int("aggregated_patterns", len(aggregatedPatterns)))

	return aggregatedPatterns, nil
}

// AnonymizePattern removes sensitive information from a pattern
func (pa *patternAggregatorImpl) AnonymizePattern(
	ctx context.Context,
	pattern *entities.TaskPattern,
	settings *entities.PrivacySettings,
) (*entities.TaskPattern, error) {
	pa.logger.Debug("anonymizing pattern", slog.String("type", string(pattern.Type)))

	// Create a deep copy
	anonymized := pa.deepCopyPattern(pattern)

	// Remove repository identifier
	anonymized.Repository = "anonymous"

	// Anonymize sequence content
	for i, step := range anonymized.Sequence {
		// Filter sensitive content from Keywords since PatternStep uses Keywords instead of Content
		anonymized.Sequence[i].TaskType = pa.generalizeTaskType(step.TaskType)
		anonymized.Sequence[i].Keywords = pa.filterKeywords(step.Keywords, settings)

		// Clear potentially sensitive metadata
		if anonymized.Sequence[i].Metadata == nil {
			anonymized.Sequence[i].Metadata = make(map[string]interface{})
		}
		delete(anonymized.Sequence[i].Metadata, "user_id")
		delete(anonymized.Sequence[i].Metadata, "team_id")
		delete(anonymized.Sequence[i].Metadata, "project_name")
		delete(anonymized.Sequence[i].Metadata, "file_paths")
	}

	// Filter metadata
	if anonymized.Metadata == nil {
		anonymized.Metadata = make(map[string]interface{})
	}

	// Remove sensitive metadata fields
	sensitiveFields := []string{
		"user_id", "team_id", "project_name", "organization",
		"email", "phone", "address", "ip_address", "hostname",
		"file_paths", "directory_paths", "api_endpoints",
	}

	for _, field := range sensitiveFields {
		delete(anonymized.Metadata, field)
	}

	// Filter keywords if present
	if keywords, exists := anonymized.Metadata["keywords"]; exists {
		if keywordSlice, ok := keywords.([]string); ok {
			anonymized.Metadata["keywords"] = pa.filterKeywords(keywordSlice, settings)
		}
	}

	// Generalize project type if needed
	if settings.AnonymizationLevel == constants.SeverityHigh {
		anonymized.Metadata["project_type"] = pa.generalizeProjectType(
			anonymized.Metadata["project_type"],
		)
	}

	// Add anonymization metadata
	anonymized.Metadata["anonymized"] = true
	anonymized.Metadata["anonymization_level"] = settings.AnonymizationLevel
	anonymized.Metadata["anonymized_at"] = time.Now()

	pa.logger.Debug("pattern anonymization completed")

	return anonymized, nil
}

// MergeAggregatedPattern merges a new pattern into an existing aggregated pattern
func (pa *patternAggregatorImpl) MergeAggregatedPattern(
	ctx context.Context,
	existing *entities.AggregatedPattern,
	newPattern *entities.TaskPattern,
	weight float64,
) error {
	pa.logger.Debug("merging pattern into aggregated pattern",
		slog.String("type", existing.Type),
		slog.Float64("weight", weight))

	// Update source count
	existing.SourceCount++

	// Update success rate (weighted average)
	totalWeight := float64(existing.SourceCount)
	existing.SuccessRate = ((existing.SuccessRate * (totalWeight - 1)) + (newPattern.SuccessRate * weight)) / totalWeight

	// Update frequency (weighted average)
	existing.Frequency = ((existing.Frequency * (totalWeight - 1)) + (newPattern.Frequency * weight)) / totalWeight

	// TaskPattern uses sequence duration stats instead of direct TimeMetrics field
	// Update aggregated pattern's time metrics from sequence duration stats
	if len(newPattern.Sequence) > 0 {
		// Aggregate duration stats from pattern sequence
		for _, step := range newPattern.Sequence {
			if step.Duration != nil {
				// Update time metrics using duration stats
				pa.updateTimeMetricsFromDuration(&existing.TimeMetrics, step.Duration, weight)
			}
		}
	}

	// Merge keywords
	if newKeywords, ok := newPattern.Metadata["keywords"].([]string); ok {
		existing.Keywords = pa.mergeKeywords(existing.Keywords, newKeywords)
	}

	// Update project types
	if projectType, ok := newPattern.Metadata["project_type"].(string); ok {
		existing.ProjectTypes = pa.addUniqueProjectType(existing.ProjectTypes, projectType)
	}

	// Add pattern source information
	source := entities.PatternSource{
		Repository:      newPattern.Repository,
		Weight:          weight,
		SuccessRate:     newPattern.SuccessRate,
		Contribution:    weight / totalWeight,
		LastContributed: time.Now(),
		Metadata:        make(map[string]interface{}),
	}

	existing.Sources = append(existing.Sources, source)

	// Recalculate confidence
	existing.Confidence = pa.calculateAggregatedConfidence(existing)

	// Update timestamp
	existing.UpdatedAt = time.Now()

	pa.logger.Debug("pattern merge completed",
		slog.Int("source_count", existing.SourceCount),
		slog.Float64("confidence", existing.Confidence))

	return nil
}

// ValidatePrivacyCompliance checks if a pattern complies with privacy settings
func (pa *patternAggregatorImpl) ValidatePrivacyCompliance(
	ctx context.Context,
	pattern *entities.TaskPattern,
	settings *entities.PrivacySettings,
) error {
	// Check if pattern sharing is enabled
	if !settings.SharePatterns {
		return errors.New("pattern sharing disabled")
	}

	// Check data age
	if settings.MaxDataAge > 0 {
		maxAge := time.Duration(settings.MaxDataAge) * 24 * time.Hour
		if time.Since(pattern.CreatedAt) > maxAge {
			return fmt.Errorf("pattern too old: %v", time.Since(pattern.CreatedAt))
		}
	}

	// Check for excluded keywords
	if err := pa.checkExcludedKeywords(pattern, settings.ExcludeKeywords); err != nil {
		return fmt.Errorf("contains excluded keywords: %w", err)
	}

	// Check for excluded patterns
	for _, excludePattern := range settings.ExcludePatterns {
		if pa.matchesPattern(string(pattern.Type), excludePattern) {
			return fmt.Errorf("matches excluded pattern: %s", excludePattern)
		}
	}

	// Check minimum anonymization requirements
	if settings.MinAnonymization > 1 {
		// This would be checked against the aggregated pattern source count
		// For now, we'll assume it passes individual validation
	}

	return nil
}

// GeneratePatternSignature creates a unique signature for a pattern
func (pa *patternAggregatorImpl) GeneratePatternSignature(pattern *entities.TaskPattern) string {
	// Create a signature based on pattern characteristics
	sigData := make([]string, 0, 8) // Pre-allocate for typical signature components

	sigData = append(sigData, string(pattern.Type))
	sigData = append(sigData, fmt.Sprintf("%.2f", pattern.SuccessRate))
	sigData = append(sigData, fmt.Sprintf("%.2f", pattern.Frequency))

	// Add sequence length and types
	sigData = append(sigData, strconv.Itoa(len(pattern.Sequence)))
	for _, step := range pattern.Sequence {
		sigData = append(sigData, step.TaskType)
	}

	// Add project type if available
	if projectType, ok := pattern.Metadata["project_type"]; ok {
		sigData = append(sigData, fmt.Sprintf("%v", projectType))
	}

	// Create hash
	combined := strings.Join(sigData, "|")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])[:16] // Use first 16 characters
}

// FilterSensitiveContent removes sensitive information from content
func (pa *patternAggregatorImpl) FilterSensitiveContent(
	content string,
	settings *entities.PrivacySettings,
) string {
	filtered := content

	// Remove sensitive keywords
	sensitiveKeywords := append(pa.config.SensitiveKeywords, settings.ExcludeKeywords...)

	for _, keyword := range sensitiveKeywords {
		// Case-insensitive replacement
		lowerContent := strings.ToLower(filtered)
		lowerKeyword := strings.ToLower(keyword)

		if strings.Contains(lowerContent, lowerKeyword) {
			// Replace with generic placeholder
			placeholder := pa.generatePlaceholder(keyword, settings.AnonymizationLevel)
			filtered = strings.ReplaceAll(filtered, keyword, placeholder)
			// Also replace case variations
			caser := cases.Title(language.English)
			filtered = strings.ReplaceAll(filtered, caser.String(keyword), placeholder)
			filtered = strings.ReplaceAll(filtered, strings.ToUpper(keyword), placeholder)
		}
	}

	// Remove patterns that look like sensitive data
	filtered = pa.removeSensitivePatterns(filtered, settings.AnonymizationLevel)

	return filtered
}

// Helper methods

func (pa *patternAggregatorImpl) groupSimilarPatterns(patterns []*entities.TaskPattern) map[string][]*entities.TaskPattern {
	groups := make(map[string][]*entities.TaskPattern)

	for _, pattern := range patterns {
		// Generate a key based on pattern characteristics
		key := pa.generateGroupingKey(pattern)
		groups[key] = append(groups[key], pattern)
	}

	return groups
}

func (pa *patternAggregatorImpl) generateGroupingKey(pattern *entities.TaskPattern) string {
	var keyParts []string

	keyParts = append(keyParts, string(pattern.Type))
	keyParts = append(keyParts, fmt.Sprintf("seq_len_%d", len(pattern.Sequence)))

	// Add primary task types
	taskTypes := make(map[string]bool)
	for _, step := range pattern.Sequence {
		generalizedType := pa.generalizeTaskType(step.TaskType)
		if !taskTypes[generalizedType] {
			keyParts = append(keyParts, generalizedType)
			taskTypes[generalizedType] = true
		}
	}

	// Add project type if available
	if projectType, ok := pattern.Metadata["project_type"]; ok {
		keyParts = append(keyParts, fmt.Sprintf("proj_%v", projectType))
	}

	return strings.Join(keyParts, "_")
}

func (pa *patternAggregatorImpl) createAggregatedPattern(
	_ context.Context,
	patterns []*entities.TaskPattern,
) (*entities.AggregatedPattern, error) {
	if len(patterns) == 0 {
		return nil, errors.New("no patterns to aggregate")
	}

	// Use the first pattern as base
	base := patterns[0]

	aggregated := &entities.AggregatedPattern{
		Type:         string(base.Type),
		Sequence:     pa.createGeneralizedSequence(patterns),
		Frequency:    0,
		SuccessRate:  0,
		TimeMetrics:  entities.CycleTimeMetrics{},
		Keywords:     []string{},
		ProjectTypes: []string{},
		SourceCount:  len(patterns),
		Sources:      []entities.PatternSource{},
		Metadata:     make(map[string]interface{}),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Aggregate metrics
	var totalFreq, totalSuccess float64
	var timeMetrics []entities.CycleTimeMetrics
	keywordSet := make(map[string]bool)
	projectTypeSet := make(map[string]bool)

	for i, pattern := range patterns {
		weight := 1.0 / float64(len(patterns)) // Equal weight for now

		totalFreq += pattern.Frequency
		totalSuccess += pattern.SuccessRate
		// Extract time metrics from pattern sequence duration stats using extractCycleTimeMetrics
		if cycleMetrics := pa.extractCycleTimeMetrics(pattern); cycleMetrics != nil {
			timeMetrics = append(timeMetrics, *cycleMetrics)
		}

		// Collect keywords
		if keywords, ok := pattern.Metadata["keywords"].([]string); ok {
			for _, keyword := range keywords {
				if !keywordSet[keyword] {
					aggregated.Keywords = append(aggregated.Keywords, keyword)
					keywordSet[keyword] = true
				}
			}
		}

		// Collect project types
		if projectType, ok := pattern.Metadata["project_type"].(string); ok {
			if !projectTypeSet[projectType] {
				aggregated.ProjectTypes = append(aggregated.ProjectTypes, projectType)
				projectTypeSet[projectType] = true
			}
		}

		// Add source
		source := entities.PatternSource{
			Repository:      pattern.Repository,
			Weight:          weight,
			SuccessRate:     pattern.SuccessRate,
			Contribution:    weight,
			LastContributed: pattern.UpdatedAt,
			Metadata:        make(map[string]interface{}),
		}
		aggregated.Sources = append(aggregated.Sources, source)

		// Copy some metadata from first pattern
		if i == 0 {
			for key, value := range pattern.Metadata {
				if !pa.isSensitiveMetadataKey(key) {
					aggregated.Metadata[key] = value
				}
			}
		}
	}

	// Calculate averages
	aggregated.Frequency = totalFreq / float64(len(patterns))
	aggregated.SuccessRate = totalSuccess / float64(len(patterns))
	aggregated.TimeMetrics = pa.aggregateTimeMetrics(timeMetrics)
	aggregated.Confidence = pa.calculateAggregatedConfidence(aggregated)

	return aggregated, nil
}

func (pa *patternAggregatorImpl) createGeneralizedSequence(patterns []*entities.TaskPattern) []entities.PatternStep {
	if len(patterns) == 0 {
		return []entities.PatternStep{}
	}

	// Find the most common sequence length
	commonLength := pa.findMostCommonSequenceLength(patterns)

	var generalizedSequence []entities.PatternStep

	for i := 0; i < commonLength; i++ {
		step := entities.PatternStep{
			Order:    i + 1,
			TaskType: pa.findMostCommonTaskTypeAtPosition(patterns, i),
			Keywords: []string{},
			Metadata: make(map[string]interface{}),
		}

		// Collect common keywords at this position
		keywordCounts := make(map[string]int)
		for _, pattern := range patterns {
			if i < len(pattern.Sequence) {
				for _, keyword := range pattern.Sequence[i].Keywords {
					keywordCounts[keyword]++
				}
			}
		}

		// Include keywords that appear in at least 30% of patterns
		threshold := len(patterns) * 30 / 100
		for keyword, count := range keywordCounts {
			if count >= threshold {
				step.Keywords = append(step.Keywords, keyword)
			}
		}

		generalizedSequence = append(generalizedSequence, step)
	}

	return generalizedSequence
}

func (pa *patternAggregatorImpl) findMostCommonSequenceLength(patterns []*entities.TaskPattern) int {
	lengthCounts := make(map[int]int)

	for _, pattern := range patterns {
		lengthCounts[len(pattern.Sequence)]++
	}

	maxCount := 0
	commonLength := 0

	for length, count := range lengthCounts {
		if count > maxCount {
			maxCount = count
			commonLength = length
		}
	}

	return commonLength
}

func (pa *patternAggregatorImpl) findMostCommonTaskTypeAtPosition(patterns []*entities.TaskPattern, position int) string {
	typeCounts := make(map[string]int)

	for _, pattern := range patterns {
		if position < len(pattern.Sequence) {
			generalizedType := pa.generalizeTaskType(pattern.Sequence[position].TaskType)
			typeCounts[generalizedType]++
		}
	}

	maxCount := 0
	commonType := "generic_task"

	for taskType, count := range typeCounts {
		if count > maxCount {
			maxCount = count
			commonType = taskType
		}
	}

	return commonType
}

func (pa *patternAggregatorImpl) generalizeTaskType(taskType string) string {
	// Mapping of specific task types to general categories
	generalizations := map[string]string{
		"setup":       "initialization",
		"configure":   "configuration",
		"install":     "installation",
		"create":      "creation",
		"implement":   "implementation",
		"test":        "testing",
		"deploy":      "deployment",
		"debug":       "debugging",
		"optimize":    "optimization",
		"document":    "documentation",
		"refactor":    "refactoring",
		"review":      "code_review",
		"merge":       "integration",
		"release":     "release",
		"maintenance": "maintenance",
		"monitoring":  "monitoring",
		"security":    "security",
		"performance": "performance",
		"ui":          "frontend",
		"api":         "backend",
		"database":    "data",
		"auth":        "authentication",
	}

	taskType = strings.ToLower(taskType)

	// Check for exact matches first
	if general, exists := generalizations[taskType]; exists {
		return general
	}

	// Check for partial matches
	for specific, general := range generalizations {
		if strings.Contains(taskType, specific) {
			return general
		}
	}

	return "generic_task"
}

func (pa *patternAggregatorImpl) generalizeProjectType(projectType interface{}) string {
	if projectType == nil {
		return "unknown"
	}

	typeStr := fmt.Sprintf("%v", projectType)
	typeStr = strings.ToLower(typeStr)

	// Group similar project types
	if strings.Contains(typeStr, "web") || strings.Contains(typeStr, "frontend") {
		return "web_application"
	}
	if strings.Contains(typeStr, "api") || strings.Contains(typeStr, "backend") {
		return "backend_service"
	}
	if strings.Contains(typeStr, "cli") || strings.Contains(typeStr, "command") {
		return "command_line_tool"
	}
	if strings.Contains(typeStr, "mobile") || strings.Contains(typeStr, "app") {
		return "mobile_application"
	}
	if strings.Contains(typeStr, "library") || strings.Contains(typeStr, "package") {
		return "library"
	}

	return "generic_project"
}

func (pa *patternAggregatorImpl) deepCopyPattern(pattern *entities.TaskPattern) *entities.TaskPattern {
	// Create a deep copy of the pattern
	copied := &entities.TaskPattern{
		ID:          uuid.New().String(), // New ID for anonymized version
		Type:        pattern.Type,
		Repository:  pattern.Repository,
		Frequency:   pattern.Frequency,
		SuccessRate: pattern.SuccessRate,
		// TaskPattern uses sequence duration stats, copy them via metadata
		Sequence:  make([]entities.PatternStep, len(pattern.Sequence)),
		CreatedAt: pattern.CreatedAt,
		UpdatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// Copy sequence
	for i, step := range pattern.Sequence {
		copied.Sequence[i] = entities.PatternStep{
			Order:    step.Order,
			TaskType: step.TaskType,
			// PatternStep uses Keywords field for content representation
			Keywords: append([]string{}, step.Keywords...),
			Metadata: make(map[string]interface{}),
		}

		// Copy step metadata
		for k, v := range step.Metadata {
			copied.Sequence[i].Metadata[k] = v
		}
	}

	// Copy pattern metadata
	for k, v := range pattern.Metadata {
		copied.Metadata[k] = v
	}

	return copied
}

func (pa *patternAggregatorImpl) filterKeywords(keywords []string, settings *entities.PrivacySettings) []string {
	var filtered []string

	excludeSet := make(map[string]bool)
	for _, keyword := range settings.ExcludeKeywords {
		excludeSet[strings.ToLower(keyword)] = true
	}

	for _, keyword := range keywords {
		if !excludeSet[strings.ToLower(keyword)] && !pa.isSensitiveKeyword(keyword) {
			filtered = append(filtered, keyword)
		}
	}

	return filtered
}

func (pa *patternAggregatorImpl) isSensitiveKeyword(keyword string) bool {
	keyword = strings.ToLower(keyword)

	for _, sensitive := range pa.config.SensitiveKeywords {
		if strings.Contains(keyword, strings.ToLower(sensitive)) {
			return true
		}
	}

	return false
}

func (pa *patternAggregatorImpl) isSensitiveMetadataKey(key string) bool {
	sensitiveKeys := []string{
		"user_id", "team_id", "project_name", "organization",
		"email", "phone", "address", "ip_address", "hostname",
		"file_paths", "directory_paths", "api_endpoints",
	}

	key = strings.ToLower(key)
	for _, sensitive := range sensitiveKeys {
		if key == sensitive {
			return true
		}
	}

	return false
}

func (pa *patternAggregatorImpl) mergeKeywords(existing []string, newKeywords []string) []string {
	keywordSet := make(map[string]bool)

	// Add existing keywords
	for _, keyword := range existing {
		keywordSet[keyword] = true
	}

	// Add new keywords
	var result []string
	result = append(result, existing...)

	for _, keyword := range newKeywords {
		if !keywordSet[keyword] {
			result = append(result, keyword)
			keywordSet[keyword] = true
		}
	}

	return result
}

func (pa *patternAggregatorImpl) addUniqueProjectType(existing []string, newType string) []string {
	for _, existingType := range existing {
		if existingType == newType {
			return existing
		}
	}
	return append(existing, newType)
}

func (pa *patternAggregatorImpl) calculateAggregatedConfidence(pattern *entities.AggregatedPattern) float64 {
	// Base confidence from source count
	sourceConfidence := float64(pattern.SourceCount) / 20.0 // Normalize to 20 sources
	if sourceConfidence > 1.0 {
		sourceConfidence = 1.0
	}

	// Success rate factor
	successConfidence := pattern.SuccessRate

	// Frequency factor
	frequencyConfidence := pattern.Frequency

	// Weighted combination
	confidence := (sourceConfidence * 0.4) + (successConfidence * 0.3) + (frequencyConfidence * 0.3)

	return confidence
}

func (pa *patternAggregatorImpl) aggregateTimeMetrics(metrics []entities.CycleTimeMetrics) entities.CycleTimeMetrics {
	if len(metrics) == 0 {
		return entities.CycleTimeMetrics{}
	}

	var totalAvg, totalMedian float64
	var maxP90 time.Duration

	for _, metric := range metrics {
		totalAvg += float64(metric.AverageCycleTime)
		totalMedian += float64(metric.MedianCycleTime)

		if metric.P90CycleTime > maxP90 {
			maxP90 = metric.P90CycleTime
		}
	}

	return entities.CycleTimeMetrics{
		AverageCycleTime: time.Duration(totalAvg / float64(len(metrics))),
		MedianCycleTime:  time.Duration(totalMedian / float64(len(metrics))),
		P90CycleTime:     maxP90,
		LeadTime:         time.Duration(totalAvg / float64(len(metrics))),
		WaitTime:         pa.calculateWaitTime(metrics), // Calculate actual wait time from metrics
	}
}

// calculateWaitTime calculates average wait time from cycle time metrics
func (pa *patternAggregatorImpl) calculateWaitTime(metrics []entities.CycleTimeMetrics) time.Duration {
	if len(metrics) == 0 {
		return 0
	}

	var totalWaitTime float64
	validMetrics := 0

	for _, metric := range metrics {
		// Wait time is typically the difference between lead time and cycle time
		// If lead time > cycle time, the difference is wait time
		if metric.LeadTime > metric.AverageCycleTime {
			totalWaitTime += float64(metric.LeadTime - metric.AverageCycleTime)
			validMetrics++
		}
	}

	if validMetrics == 0 {
		return 0
	}

	return time.Duration(totalWaitTime / float64(validMetrics))
}

func (pa *patternAggregatorImpl) checkExcludedKeywords(pattern *entities.TaskPattern, excludedKeywords []string) error {
	// Check pattern content and metadata for excluded keywords
	for _, keyword := range excludedKeywords {
		keyword = strings.ToLower(keyword)

		// Check pattern type
		if strings.Contains(strings.ToLower(string(pattern.Type)), keyword) {
			return fmt.Errorf("excluded keyword in pattern type: %s", keyword)
		}

		// Check sequence content - PatternStep doesn't have Content field, check TaskType and Keywords
		for _, step := range pattern.Sequence {
			if strings.Contains(strings.ToLower(step.TaskType), keyword) {
				return fmt.Errorf("excluded keyword in sequence task type: %s", keyword)
			}

			for _, stepKeyword := range step.Keywords {
				if strings.Contains(strings.ToLower(stepKeyword), keyword) {
					return fmt.Errorf("excluded keyword in sequence keywords: %s", keyword)
				}
			}
		}

		// Check metadata
		if keywords, ok := pattern.Metadata["keywords"].([]string); ok {
			for _, metaKeyword := range keywords {
				if strings.Contains(strings.ToLower(metaKeyword), keyword) {
					return fmt.Errorf("excluded keyword in metadata: %s", keyword)
				}
			}
		}
	}

	return nil
}

func (pa *patternAggregatorImpl) matchesPattern(patternType, excludePattern string) bool {
	// Simple pattern matching (could be enhanced with regex or glob)
	if excludePattern == patternType {
		return true
	}

	// Handle wildcard patterns
	if strings.HasPrefix(excludePattern, "*") && strings.HasSuffix(excludePattern, "*") {
		middle := strings.Trim(excludePattern, "*")
		return strings.Contains(patternType, middle)
	}

	if strings.HasPrefix(excludePattern, "*") {
		suffix := strings.TrimPrefix(excludePattern, "*")
		return strings.HasSuffix(patternType, suffix)
	}

	if strings.HasSuffix(excludePattern, "*") {
		prefix := strings.TrimSuffix(excludePattern, "*")
		return strings.HasPrefix(patternType, prefix)
	}

	return false
}

func (pa *patternAggregatorImpl) generatePlaceholder(keyword, level string) string {
	switch level {
	case "high":
		return "[REDACTED]"
	case "medium":
		return fmt.Sprintf("[%s]", strings.ToUpper(keyword[:min(3, len(keyword))]))
	default: // basic
		return fmt.Sprintf("[%s]", keyword[:min(1, len(keyword))])
	}
}

func (pa *patternAggregatorImpl) removeSensitivePatterns(content, level string) string {
	// Remove patterns that look like sensitive data based on regex
	// This is a simplified implementation

	if level == "high" {
		// Remove email patterns
		emailRegex := `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`
		content = strings.ReplaceAll(content, emailRegex, "[EMAIL]")

		// Remove URL patterns
		urlRegex := `https?://[^\s]+`
		content = strings.ReplaceAll(content, urlRegex, "[URL]")

		// Remove potential API keys (long alphanumeric strings)
		// This is very basic - real implementation would be more sophisticated
		words := strings.Fields(content)
		for i, word := range words {
			if len(word) > 20 && pa.isAlphanumeric(word) {
				words[i] = "[API_KEY]"
			}
		}
		content = strings.Join(words, " ")
	}

	return content
}

func (pa *patternAggregatorImpl) isAlphanumeric(s string) bool {
	for _, char := range s {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')) {
			return false
		}
	}
	return true
}

// Helper function to extract cycle time metrics from a task pattern
func (pa *patternAggregatorImpl) extractCycleTimeMetrics(pattern *entities.TaskPattern) *entities.CycleTimeMetrics {
	// Since TaskPattern doesn't have TimeMetrics, we extract from sequence duration stats
	if len(pattern.Sequence) == 0 {
		return nil
	}

	var totalDurations []time.Duration
	for _, step := range pattern.Sequence {
		if step.Duration != nil && step.Duration.Average > 0 {
			totalDurations = append(totalDurations, step.Duration.Average)
		}
	}

	if len(totalDurations) == 0 {
		return nil
	}

	// Calculate cycle time metrics from duration stats
	var total time.Duration
	for _, d := range totalDurations {
		total += d
	}

	return &entities.CycleTimeMetrics{
		AverageCycleTime: total / time.Duration(len(totalDurations)),
		MedianCycleTime:  total / time.Duration(len(totalDurations)), // Simplified
		P90CycleTime:     total,                                      // Simplified
		LeadTime:         total,
		WaitTime:         0,
	}
}

// Helper function to update time metrics from duration stats
func (pa *patternAggregatorImpl) updateTimeMetricsFromDuration(existing *entities.CycleTimeMetrics, duration *entities.DurationStats, weight float64) {
	if duration == nil {
		return
	}

	// Update average cycle time
	avgExisting := float64(existing.AverageCycleTime)
	avgNew := float64(duration.Average)
	existing.AverageCycleTime = time.Duration(avgExisting*(1-weight) + avgNew*weight)

	// Update median cycle time
	medExisting := float64(existing.MedianCycleTime)
	medNew := float64(duration.Median)
	existing.MedianCycleTime = time.Duration(medExisting*(1-weight) + medNew*weight)

	// Update P90 (using max as approximation)
	if duration.Max > existing.P90CycleTime {
		existing.P90CycleTime = duration.Max
	}
}
