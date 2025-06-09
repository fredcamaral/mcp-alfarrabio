package services

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"

	"github.com/google/uuid"
)

// CrossRepoAnalyzer interface defines cross-repository analysis capabilities
type CrossRepoAnalyzer interface {
	// Core analysis operations
	AnalyzeCrossRepoPatterns(ctx context.Context, repository string) ([]*entities.CrossRepoInsight, error)
	FindSimilarRepositories(ctx context.Context, repository string, limit int) ([]*entities.RepositorySimilarity, error)
	GetSharedInsights(ctx context.Context, projectType entities.ProjectType) ([]*entities.CrossRepoInsight, error)
	GetInsightRecommendations(ctx context.Context, repository string) ([]*entities.CrossRepoInsight, error)

	// Pattern contribution and sharing
	ContributePattern(ctx context.Context, pattern *entities.TaskPattern, optIn bool) error
	CalculateRepositorySimilarity(ctx context.Context, repoA, repoB string) (*entities.RepositorySimilarity, error)

	// Privacy and settings
	UpdatePrivacySettings(ctx context.Context, repository string, settings *entities.PrivacySettings) error
	GetPrivacySettings(ctx context.Context, repository string) (*entities.PrivacySettings, error)

	// Analytics and insights
	GetInsightAnalytics(ctx context.Context, insightID string) (*InsightAnalytics, error)
	GetCrossRepoTrends(ctx context.Context, timeRange time.Duration) ([]*entities.CrossRepoInsight, error)
}

// InsightAnalytics represents analytics data for insights
type InsightAnalytics struct {
	InsightID        string                 `json:"insight_id"`
	ViewCount        int                    `json:"view_count"`
	ApplicationCount int                    `json:"application_count"`
	SuccessRate      float64                `json:"success_rate"`
	UserFeedback     []InsightFeedback      `json:"user_feedback"`
	TrendData        []TrendPoint           `json:"trend_data"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// InsightFeedback represents user feedback on insights
type InsightFeedback struct {
	UserID      string    `json:"user_id,omitempty"`
	Repository  string    `json:"repository"`
	Rating      int       `json:"rating"` // 1-5 stars
	Helpful     bool      `json:"helpful"`
	Applied     bool      `json:"applied"`
	Comments    string    `json:"comments,omitempty"`
	SubmittedAt time.Time `json:"submitted_at"`
}

// TrendPoint represents a data point in trend analysis
type TrendPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Metric    string    `json:"metric"`
}

// CrossRepoAnalyzerConfig holds configuration for cross-repo analysis
type CrossRepoAnalyzerConfig struct {
	MinSimilarityThreshold    float64       `json:"min_similarity_threshold"`
	MinSourceCount            int           `json:"min_source_count"`
	MinConfidenceScore        float64       `json:"min_confidence_score"`
	MaxInsightAge             time.Duration `json:"max_insight_age"`
	SimilarityCacheTTL        time.Duration `json:"similarity_cache_ttl"`
	InsightGenerationInterval time.Duration `json:"insight_generation_interval"`
	PrivacyEnforcement        bool          `json:"privacy_enforcement"`
	AnonymizationStrength     string        `json:"anonymization_strength"`
}

// DefaultCrossRepoAnalyzerConfig returns default configuration
func DefaultCrossRepoAnalyzerConfig() *CrossRepoAnalyzerConfig {
	return &CrossRepoAnalyzerConfig{
		MinSimilarityThreshold:    0.6,
		MinSourceCount:            3,
		MinConfidenceScore:        0.5,
		MaxInsightAge:             30 * 24 * time.Hour,
		SimilarityCacheTTL:        24 * time.Hour,
		InsightGenerationInterval: 6 * time.Hour,
		PrivacyEnforcement:        true,
		AnonymizationStrength:     "medium",
	}
}

// CrossRepoDependencies holds dependencies for cross-repo analyzer
type CrossRepoDependencies struct {
	PatternStore      ports.PatternStorage
	InsightStore      ports.InsightStorage
	SimilarityCache   Cache
	MCPClient         MCPClient
	ProjectClassifier ProjectClassifier
	Logger            *slog.Logger
	Config            *CrossRepoAnalyzerConfig
}

// crossRepoAnalyzerImpl implements the CrossRepoAnalyzer interface
type crossRepoAnalyzerImpl struct {
	patternStore      ports.PatternStorage
	insightStore      ports.InsightStorage
	similarityCache   Cache
	mcpClient         MCPClient
	projectClassifier ProjectClassifier
	privacySettings   map[string]*entities.PrivacySettings
	config            *CrossRepoAnalyzerConfig
	logger            *slog.Logger
}

// NewCrossRepoAnalyzer creates a new cross-repository analyzer
func NewCrossRepoAnalyzer(deps CrossRepoDependencies) CrossRepoAnalyzer {
	if deps.Config == nil {
		deps.Config = DefaultCrossRepoAnalyzerConfig()
	}

	return &crossRepoAnalyzerImpl{
		patternStore:      deps.PatternStore,
		insightStore:      deps.InsightStore,
		similarityCache:   deps.SimilarityCache,
		mcpClient:         deps.MCPClient,
		projectClassifier: deps.ProjectClassifier,
		privacySettings:   make(map[string]*entities.PrivacySettings),
		config:            deps.Config,
		logger:            deps.Logger,
	}
}

// AnalyzeCrossRepoPatterns analyzes patterns across repositories and generates insights
func (a *crossRepoAnalyzerImpl) AnalyzeCrossRepoPatterns(
	ctx context.Context,
	repository string,
) ([]*entities.CrossRepoInsight, error) {
	a.logger.Info("analyzing cross-repo patterns", slog.String("repository", repository))

	// Get repository characteristics
	repoChars, err := a.getRepositoryCharacteristics(ctx, repository)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository characteristics: %w", err)
	}

	// Find similar repositories
	similarRepos, err := a.FindSimilarRepositories(ctx, repository, 20)
	if err != nil {
		return nil, fmt.Errorf("failed to find similar repositories: %w", err)
	}

	// Aggregate patterns from similar repositories
	aggregatedPatterns := make(map[string]*entities.AggregatedPattern)

	for _, simRepo := range similarRepos {
		if simRepo.Score < a.config.MinSimilarityThreshold {
			continue
		}

		// Get patterns from similar repository (with privacy check)
		patterns, err := a.getShareablePatterns(ctx, simRepo.RepositoryB)
		if err != nil {
			a.logger.Warn("failed to get patterns",
				slog.String("repo", simRepo.RepositoryB),
				slog.Any("error", err))
			continue
		}

		// Aggregate patterns
		for _, pattern := range patterns {
			key := a.generatePatternKey(pattern)
			if agg, exists := aggregatedPatterns[key]; exists {
				agg.Merge(pattern, simRepo.Score)
			} else {
				aggregatedPatterns[key] = a.newAggregatedPattern(pattern)
			}
		}
	}

	// Generate insights from aggregated patterns
	var insights []*entities.CrossRepoInsight

	for _, aggPattern := range aggregatedPatterns {
		if aggPattern.SourceCount < a.config.MinSourceCount {
			continue
		}

		insight := a.generateInsight(aggPattern, repoChars)
		if insight != nil && insight.Confidence > a.config.MinConfidenceScore {
			insights = append(insights, insight)
		}
	}

	// Rank insights by relevance and impact
	insights = a.rankInsights(insights, repoChars)

	// Store insights for caching
	for _, insight := range insights {
		if err := a.insightStore.Create(ctx, insight); err != nil {
			a.logger.Warn("failed to store insight", slog.Any("error", err))
		}
	}

	a.logger.Info("cross-repo analysis completed",
		slog.Int("insights_generated", len(insights)),
		slog.Int("similar_repos", len(similarRepos)))

	return insights, nil
}

// FindSimilarRepositories finds repositories similar to the given one
func (a *crossRepoAnalyzerImpl) FindSimilarRepositories(
	ctx context.Context,
	repository string,
	limit int,
) ([]*entities.RepositorySimilarity, error) {
	// Check cache
	cacheKey := fmt.Sprintf("similar_repos:%s", repository)
	if cached, found := a.similarityCache.Get(cacheKey); found {
		if similarities, ok := cached.([]*entities.RepositorySimilarity); ok {
			return a.limitSimilarities(similarities, limit), nil
		}
	}

	// Call MCP server for multi-repo analysis
	response, err := a.mcpClient.Call(ctx, "memory_analyze", map[string]interface{}{
		"operation": "find_similar_repositories",
		"options": map[string]interface{}{
			"repository": repository,
			"session_id": uuid.New().String(),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to call MCP server: %w", err)
	}

	// Parse response into similarity scores
	var similarities []*entities.RepositorySimilarity
	if response.SimilarRepositories != nil {
		for _, item := range response.SimilarRepositories {
			similarity := &entities.RepositorySimilarity{
				RepositoryA:    repository,
				RepositoryB:    item.Repository,
				Score:          item.Score,
				Dimensions:     a.parseDimensions(item.Dimensions),
				SharedPatterns: item.SharedPatterns,
				LastCalculated: time.Now(),
			}
			similarities = append(similarities, similarity)
		}
	}

	// Sort by score
	sort.Slice(similarities, func(i, j int) bool {
		return similarities[i].Score > similarities[j].Score
	})

	// Cache results
	a.similarityCache.Set(cacheKey, similarities, a.config.SimilarityCacheTTL)

	return a.limitSimilarities(similarities, limit), nil
}

// GetSharedInsights gets publicly shared insights for a project type
func (a *crossRepoAnalyzerImpl) GetSharedInsights(
	ctx context.Context,
	projectType entities.ProjectType,
) ([]*entities.CrossRepoInsight, error) {
	// Query insights from storage
	insights, err := a.insightStore.GetByProjectType(ctx, entities.ProjectType(projectType))
	if err != nil {
		return nil, fmt.Errorf("failed to get shared insights: %w", err)
	}

	// Filter by validity and quality
	var validInsights []*entities.CrossRepoInsight
	for _, insight := range insights {
		if !insight.IsExpired() && insight.GetQualityScore() > 0.5 {
			validInsights = append(validInsights, insight)
		}
	}

	// Sort by quality score
	entities.SortInsightsByQuality(validInsights)

	return validInsights, nil
}

// GetInsightRecommendations gets personalized insight recommendations
func (a *crossRepoAnalyzerImpl) GetInsightRecommendations(
	ctx context.Context,
	repository string,
) ([]*entities.CrossRepoInsight, error) {
	// Get repository characteristics
	repoChars, err := a.getRepositoryCharacteristics(ctx, repository)
	if err != nil {
		return nil, err
	}

	// Get insights for the project type
	insights, err := a.GetSharedInsights(ctx, repoChars.ProjectType)
	if err != nil {
		return nil, err
	}

	// Calculate relevance for each insight
	for _, insight := range insights {
		insight.Relevance = a.calculateRelevance(insight.Pattern, repoChars)
	}

	// Filter by minimum relevance
	insights = entities.FilterInsightsByRelevance(insights, 0.6)

	// Sort by relevance
	sort.Slice(insights, func(i, j int) bool {
		return insights[i].Relevance > insights[j].Relevance
	})

	// Limit to top 10
	if len(insights) > 10 {
		insights = insights[:10]
	}

	return insights, nil
}

// ContributePattern contributes a pattern to the shared knowledge base
func (a *crossRepoAnalyzerImpl) ContributePattern(
	ctx context.Context,
	pattern *entities.TaskPattern,
	optIn bool,
) error {
	if !optIn {
		return nil // User opted out
	}

	// Get privacy settings for repository
	settings := a.getPrivacySettingsForRepo(pattern.Repository)
	if !settings.ShouldShare(pattern) {
		return nil
	}

	// Anonymize pattern
	anonymized := a.anonymizePattern(pattern, settings)

	// Send to server for aggregation
	_, err := a.mcpClient.Call(ctx, "memory_create", map[string]interface{}{
		"operation": "contribute_pattern",
		"options": map[string]interface{}{
			"pattern":    anonymized,
			"repository": "cross_repo_learning", // Special repo for shared patterns
		},
	})

	if err != nil {
		return fmt.Errorf("failed to contribute pattern: %w", err)
	}

	a.logger.Debug("pattern contributed successfully", slog.String("type", string(pattern.Type)))
	return nil
}

// CalculateRepositorySimilarity calculates similarity between two repositories
func (a *crossRepoAnalyzerImpl) CalculateRepositorySimilarity(
	ctx context.Context,
	repoA, repoB string,
) (*entities.RepositorySimilarity, error) {
	// Get characteristics for both repositories
	charsA, err := a.getRepositoryCharacteristics(ctx, repoA)
	if err != nil {
		return nil, fmt.Errorf("failed to get characteristics for %s: %w", repoA, err)
	}

	charsB, err := a.getRepositoryCharacteristics(ctx, repoB)
	if err != nil {
		return nil, fmt.Errorf("failed to get characteristics for %s: %w", repoB, err)
	}

	// Calculate similarity dimensions
	dimensions := a.calculateSimilarityDimensions(charsA, charsB)

	// Calculate overall score
	similarity := &entities.RepositorySimilarity{
		RepositoryA:    repoA,
		RepositoryB:    repoB,
		Dimensions:     dimensions,
		LastCalculated: time.Now(),
		Metadata:       make(map[string]interface{}),
	}

	similarity.Score = similarity.GetOverallSimilarity()

	return similarity, nil
}

// UpdatePrivacySettings updates privacy settings for a repository
func (a *crossRepoAnalyzerImpl) UpdatePrivacySettings(
	ctx context.Context,
	repository string,
	settings *entities.PrivacySettings,
) error {
	a.privacySettings[repository] = settings
	a.logger.Debug("privacy settings updated", slog.String("repository", repository))
	return nil
}

// GetPrivacySettings gets privacy settings for a repository
func (a *crossRepoAnalyzerImpl) GetPrivacySettings(
	ctx context.Context,
	repository string,
) (*entities.PrivacySettings, error) {
	if settings, exists := a.privacySettings[repository]; exists {
		return settings, nil
	}
	return entities.DefaultPrivacySettings(), nil
}

// Helper methods

func (a *crossRepoAnalyzerImpl) getRepositoryCharacteristics(
	ctx context.Context,
	repository string,
) (*entities.RepositoryCharacteristics, error) {
	// This would typically call the project classifier
	projectType, confidence, err := a.projectClassifier.ClassifyProject(ctx, repository)
	if err != nil {
		return nil, err
	}

	projectChars, err := a.projectClassifier.GetProjectCharacteristics(ctx, repository)
	if err != nil {
		return nil, err
	}

	// Convert to RepositoryCharacteristics
	repoChars := &entities.RepositoryCharacteristics{
		ProjectType:   projectType,
		Languages:     projectChars.Languages,
		Frameworks:    projectChars.Frameworks,
		Dependencies:  projectChars.Dependencies,
		Complexity:    projectChars.GetComplexityScore(),
		ActivityLevel: 0.8, // Placeholder
		AnalyzedAt:    time.Now(),
		Metadata:      make(map[string]interface{}),
	}

	repoChars.Metadata["classification_confidence"] = confidence

	return repoChars, nil
}

func (a *crossRepoAnalyzerImpl) getShareablePatterns(
	ctx context.Context,
	repository string,
) ([]*entities.TaskPattern, error) {
	// Get patterns from storage
	patterns, err := a.patternStore.GetByRepository(ctx, repository)
	if err != nil {
		return nil, err
	}

	// Filter by privacy settings
	settings := a.getPrivacySettingsForRepo(repository)
	var shareablePatterns []*entities.TaskPattern

	for _, pattern := range patterns {
		if settings.ShouldShare(pattern) {
			shareablePatterns = append(shareablePatterns, pattern)
		}
	}

	return shareablePatterns, nil
}

func (a *crossRepoAnalyzerImpl) getPrivacySettingsForRepo(repository string) *entities.PrivacySettings {
	if settings, exists := a.privacySettings[repository]; exists {
		return settings
	}
	return entities.DefaultPrivacySettings()
}

func (a *crossRepoAnalyzerImpl) generatePatternKey(pattern *entities.TaskPattern) string {
	// Generate a key based on pattern characteristics
	return fmt.Sprintf("%s:%s:%d", pattern.Type, pattern.Repository, len(pattern.Sequence))
}

func (a *crossRepoAnalyzerImpl) newAggregatedPattern(pattern *entities.TaskPattern) *entities.AggregatedPattern {
	return &entities.AggregatedPattern{
		Type:        string(pattern.Type),
		Frequency:   pattern.Frequency,
		SuccessRate: pattern.SuccessRate,
		TimeMetrics: a.extractTimeMetrics(pattern),
		SourceCount: 1,
		Confidence:  0.5, // Initial confidence
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}
}

func (a *crossRepoAnalyzerImpl) generateInsight(
	pattern *entities.AggregatedPattern,
	repoChars *entities.RepositoryCharacteristics,
) *entities.CrossRepoInsight {
	// Determine insight type
	insightType := a.classifyInsightType(pattern)

	// Calculate relevance to current repository
	relevance := a.calculateRelevanceFromPattern(pattern, repoChars)

	// Generate actionable recommendations
	recommendations := a.generateRecommendations(pattern, repoChars)

	// Calculate impact metrics
	impact := a.calculateImpact(pattern)

	insight := &entities.CrossRepoInsight{
		ID:          uuid.New().String(),
		Type:        insightType,
		Title:       a.generateInsightTitle(pattern),
		Description: a.generateInsightDescription(pattern),
		Pattern: &entities.AnonymizedPattern{
			Type:           pattern.Type,
			Frequency:      pattern.Frequency,
			SuccessRate:    pattern.SuccessRate,
			TimeMetrics:    pattern.TimeMetrics,
			CommonKeywords: a.filterSensitiveKeywords(pattern.Keywords),
			ProjectTypes:   pattern.ProjectTypes,
		},
		SourceCount:     pattern.SourceCount,
		Confidence:      pattern.Confidence,
		Relevance:       relevance,
		Impact:          impact,
		Applicability:   pattern.ProjectTypes,
		Prerequisites:   a.identifyPrerequisites(pattern),
		Recommendations: recommendations,
		Tags:            a.generateTags(pattern),
		Metadata:        make(map[string]interface{}),
		GeneratedAt:     time.Now(),
		ValidUntil:      time.Now().Add(a.config.MaxInsightAge),
		IsActionable:    len(recommendations) > 0,
	}

	return insight
}

func (a *crossRepoAnalyzerImpl) classifyInsightType(pattern *entities.AggregatedPattern) entities.InsightType {
	// Simple classification logic
	if pattern.SuccessRate > 0.8 && pattern.SourceCount > 5 {
		return entities.InsightTypeBestPractice
	}
	if pattern.SuccessRate < 0.4 {
		return entities.InsightTypeAntiPattern
	}
	if pattern.Frequency > 0.7 {
		return entities.InsightTypePattern
	}
	return entities.InsightTypeWorkflow
}

func (a *crossRepoAnalyzerImpl) calculateRelevance(
	pattern *entities.AnonymizedPattern,
	repoChars *entities.RepositoryCharacteristics,
) float64 {
	relevance := 0.0

	// Project type match
	for _, pt := range pattern.ProjectTypes {
		if pt == string(repoChars.ProjectType) {
			relevance += 0.4
			break
		}
	}

	// Technology match
	for _, framework := range repoChars.Frameworks {
		for _, keyword := range pattern.CommonKeywords {
			if strings.Contains(strings.ToLower(keyword), strings.ToLower(framework)) {
				relevance += 0.2
				break
			}
		}
	}

	// Complexity alignment
	complexityDiff := 1.0 - abs(repoChars.Complexity-0.5) // Assuming pattern complexity around 0.5
	relevance += complexityDiff * 0.2

	// Success rate bonus
	relevance += pattern.SuccessRate * 0.2

	if relevance > 1.0 {
		relevance = 1.0
	}

	return relevance
}

func (a *crossRepoAnalyzerImpl) calculateRelevanceFromPattern(
	pattern *entities.AggregatedPattern,
	repoChars *entities.RepositoryCharacteristics,
) float64 {
	// Convert to AnonymizedPattern for calculation
	anonymized := &entities.AnonymizedPattern{
		ProjectTypes:   pattern.ProjectTypes,
		CommonKeywords: pattern.Keywords,
		SuccessRate:    pattern.SuccessRate,
	}
	return a.calculateRelevance(anonymized, repoChars)
}

// Additional helper methods would be implemented here...

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func (a *crossRepoAnalyzerImpl) limitSimilarities(similarities []*entities.RepositorySimilarity, limit int) []*entities.RepositorySimilarity {
	if len(similarities) <= limit {
		return similarities
	}
	return similarities[:limit]
}

// Placeholder implementations for remaining helper methods
func (a *crossRepoAnalyzerImpl) parseDimensions(dimensions interface{}) entities.SimilarityDimensions {
	// Implementation would parse the dimensions from MCP response
	return entities.SimilarityDimensions{}
}

func (a *crossRepoAnalyzerImpl) rankInsights(insights []*entities.CrossRepoInsight, repoChars *entities.RepositoryCharacteristics) []*entities.CrossRepoInsight {
	// Sort by quality score
	entities.SortInsightsByQuality(insights)
	return insights
}

func (a *crossRepoAnalyzerImpl) anonymizePattern(pattern *entities.TaskPattern, settings *entities.PrivacySettings) *entities.TaskPattern {
	// Create anonymized copy
	anonymized := *pattern
	anonymized.Repository = "anonymous"
	return &anonymized
}

func (a *crossRepoAnalyzerImpl) calculateSimilarityDimensions(charsA, charsB *entities.RepositoryCharacteristics) entities.SimilarityDimensions {
	return entities.SimilarityDimensions{
		ProjectType: a.calculateProjectTypeSimilarity(charsA.ProjectType, charsB.ProjectType),
		TechStack:   a.calculateTechStackSimilarity(charsA.Frameworks, charsB.Frameworks),
		Complexity:  1.0 - abs(charsA.Complexity-charsB.Complexity),
	}
}

func (a *crossRepoAnalyzerImpl) calculateProjectTypeSimilarity(typeA, typeB entities.ProjectType) float64 {
	if typeA == typeB {
		return 1.0
	}
	return 0.0
}

func (a *crossRepoAnalyzerImpl) calculateTechStackSimilarity(frameworksA, frameworksB []string) float64 {
	if len(frameworksA) == 0 && len(frameworksB) == 0 {
		return 1.0
	}

	setA := make(map[string]bool)
	for _, f := range frameworksA {
		setA[f] = true
	}

	common := 0
	for _, f := range frameworksB {
		if setA[f] {
			common++
		}
	}

	total := len(frameworksA) + len(frameworksB) - common
	if total == 0 {
		return 1.0
	}

	return float64(common) / float64(total)
}

// Placeholder implementations for remaining methods
func (a *crossRepoAnalyzerImpl) generateRecommendations(pattern *entities.AggregatedPattern, repoChars *entities.RepositoryCharacteristics) []string {
	return []string{"Consider adopting this pattern", "Review implementation details"}
}

func (a *crossRepoAnalyzerImpl) calculateImpact(pattern *entities.AggregatedPattern) entities.ImpactMetrics {
	return entities.ImpactMetrics{
		ProductivityGain:   pattern.SuccessRate * 0.2,
		TimeReduction:      float64(pattern.TimeMetrics.AverageCycleTime) / 1e9 * 0.1, // Convert nanoseconds to seconds
		QualityImprovement: pattern.SuccessRate * 0.15,
		AdoptionRate:       float64(pattern.SourceCount) / 100.0,
	}
}

func (a *crossRepoAnalyzerImpl) generateInsightTitle(pattern *entities.AggregatedPattern) string {
	return fmt.Sprintf("Pattern: %s", pattern.Type)
}

func (a *crossRepoAnalyzerImpl) generateInsightDescription(pattern *entities.AggregatedPattern) string {
	return fmt.Sprintf("This pattern appears in %d repositories with %.1f%% success rate",
		pattern.SourceCount, pattern.SuccessRate*100)
}

func (a *crossRepoAnalyzerImpl) extractTimeMetrics(pattern *entities.TaskPattern) entities.CycleTimeMetrics {
	// Extract time metrics from pattern steps
	metrics := entities.CycleTimeMetrics{
		ByType:       make(map[string]time.Duration),
		ByPriority:   make(map[string]time.Duration),
		Distribution: []entities.CycleTimePoint{},
	}

	var totalDuration time.Duration
	for _, step := range pattern.Sequence {
		if step.Duration != nil {
			totalDuration += step.Duration.Average
			metrics.ByType[step.TaskType] = step.Duration.Average
		}
	}

	metrics.AverageCycleTime = totalDuration
	metrics.MedianCycleTime = totalDuration // Simplified
	metrics.P90CycleTime = totalDuration    // Simplified

	return metrics
}

func (a *crossRepoAnalyzerImpl) filterSensitiveKeywords(keywords []string) []string {
	sensitive := []string{"password", "secret", "key", "token", "private"}
	var filtered []string

	for _, keyword := range keywords {
		isSensitive := false
		for _, s := range sensitive {
			if strings.Contains(strings.ToLower(keyword), s) {
				isSensitive = true
				break
			}
		}
		if !isSensitive {
			filtered = append(filtered, keyword)
		}
	}

	return filtered
}

func (a *crossRepoAnalyzerImpl) identifyPrerequisites(pattern *entities.AggregatedPattern) []string {
	return []string{} // Placeholder
}

func (a *crossRepoAnalyzerImpl) generateTags(pattern *entities.AggregatedPattern) []string {
	return []string{pattern.Type, "cross-repo", "insight"}
}

// Storage interfaces are now defined in interfaces.go

// GetInsightAnalytics and GetCrossRepoTrends would be implemented similarly
func (a *crossRepoAnalyzerImpl) GetInsightAnalytics(ctx context.Context, insightID string) (*InsightAnalytics, error) {
	// Implementation placeholder
	return &InsightAnalytics{
		InsightID: insightID,
	}, nil
}

func (a *crossRepoAnalyzerImpl) GetCrossRepoTrends(ctx context.Context, timeRange time.Duration) ([]*entities.CrossRepoInsight, error) {
	// Implementation placeholder
	return []*entities.CrossRepoInsight{}, nil
}
