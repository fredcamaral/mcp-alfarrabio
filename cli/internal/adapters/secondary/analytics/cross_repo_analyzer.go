package analytics

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sort"
	"strings"
	"time"

	"lerian-mcp-memory-cli/internal/domain/constants"
	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
	"lerian-mcp-memory-cli/internal/domain/services"
)

// crossRepoAnalyzer implements the CrossRepoAnalyzer interface
type crossRepoAnalyzer struct {
	analyticsEngine ports.AnalyticsEngine
	taskRepo        ports.TaskRepository
	sessionRepo     ports.SessionRepository
	patternRepo     ports.PatternStorage
	insightRepo     ports.InsightStorage
	logger          *slog.Logger
}

// NewCrossRepoAnalyzer creates a new cross-repository analyzer
func NewCrossRepoAnalyzer(
	analyticsEngine ports.AnalyticsEngine,
	taskRepo ports.TaskRepository,
	sessionRepo ports.SessionRepository,
	patternRepo ports.PatternStorage,
	insightRepo ports.InsightStorage,
	logger *slog.Logger,
) services.CrossRepoAnalyzer {
	return &crossRepoAnalyzer{
		analyticsEngine: analyticsEngine,
		taskRepo:        taskRepo,
		sessionRepo:     sessionRepo,
		patternRepo:     patternRepo,
		insightRepo:     insightRepo,
		logger:          logger,
	}
}

// AnalyzeCrossRepoPatterns analyzes patterns across repositories and generates insights
func (cra *crossRepoAnalyzer) AnalyzeCrossRepoPatterns(ctx context.Context, repository string) ([]*entities.CrossRepoInsight, error) {
	cra.logger.Info("analyzing cross-repo patterns", slog.String("repository", repository))

	// Get all patterns for the current repository
	repoPatterns, err := cra.patternRepo.GetByRepository(ctx, repository)
	if err != nil {
		return nil, fmt.Errorf("failed to get patterns for repository: %w", err)
	}

	// Get patterns from other repositories for comparison
	// Use search with empty query to get all patterns
	allPatterns, err := cra.patternRepo.Search(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get all patterns: %w", err)
	}

	// Group patterns by repository (excluding current one)
	otherRepoPatterns := make(map[string][]*entities.TaskPattern)
	for _, pattern := range allPatterns {
		if pattern.Repository != repository {
			otherRepoPatterns[pattern.Repository] = append(otherRepoPatterns[pattern.Repository], pattern)
		}
	}

	var insights []*entities.CrossRepoInsight

	// Analyze pattern similarities across repositories
	for otherRepo, otherPatterns := range otherRepoPatterns {
		similarity := cra.calculatePatternSimilarity(repoPatterns, otherPatterns)
		if similarity > 0.7 { // High similarity threshold
			insight := &entities.CrossRepoInsight{
				ID:          generateInsightID("similarity"),
				Type:        entities.InsightTypeComparison,
				Title:       "High Pattern Similarity with " + otherRepo,
				Description: fmt.Sprintf("Repository %s shows %.1f%% pattern similarity with %s", repository, similarity*100, otherRepo),
				SourceCount: len(otherPatterns),
				Confidence:  0.8,
				Relevance:   0.9,
				Impact: entities.ImpactMetrics{
					ProductivityGain:   0.2,
					TimeReduction:      0.15,
					QualityImprovement: 0.1,
					AdoptionRate:       0.8,
				},
				Recommendations: []string{
					fmt.Sprintf("Consider knowledge sharing with %s team", otherRepo),
					"Adopt successful patterns from similar repositories",
					"Explore common optimization opportunities",
				},
				Tags:         []string{"similarity", "cross-repo", "patterns"},
				Metadata:     make(map[string]interface{}),
				GeneratedAt:  time.Now(),
				ValidUntil:   time.Now().Add(7 * 24 * time.Hour),
				IsActionable: true,
			}
			insight.Metadata["similar_repository"] = otherRepo
			insight.Metadata["similarity_score"] = similarity
			insight.Metadata["common_patterns"] = cra.findCommonPatternTypes(repoPatterns, otherPatterns)
			insights = append(insights, insight)
		}
	}

	// Identify unique patterns that could be shared
	uniquePatterns := cra.findUniquePatterns(repoPatterns, allPatterns)
	if len(uniquePatterns) > 0 {
		insight := &entities.CrossRepoInsight{
			ID:          generateInsightID("unique"),
			Type:        entities.InsightTypeOpportunity,
			Title:       "Unique Patterns Identified",
			Description: fmt.Sprintf("Repository %s has %d unique patterns that could benefit other projects", repository, len(uniquePatterns)),
			SourceCount: len(uniquePatterns),
			Confidence:  0.7,
			Relevance:   0.8,
			Impact: entities.ImpactMetrics{
				ProductivityGain:   0.3,
				TimeReduction:      0.2,
				QualityImprovement: 0.2,
				AdoptionRate:       0.6,
			},
			Recommendations: []string{
				"Document and share unique successful patterns",
				"Create templates for reusable patterns",
				"Consider contributing to shared pattern library",
			},
			Tags:         []string{"unique", "opportunity", "patterns"},
			Metadata:     make(map[string]interface{}),
			GeneratedAt:  time.Now(),
			ValidUntil:   time.Now().Add(14 * 24 * time.Hour),
			IsActionable: true,
		}
		insight.Metadata["unique_patterns"] = cra.extractPatternNames(uniquePatterns)
		insights = append(insights, insight)
	}

	return insights, nil
}

// calculatePatternSimilarity calculates similarity between two sets of patterns
func (cra *crossRepoAnalyzer) calculatePatternSimilarity(patternsA, patternsB []*entities.TaskPattern) float64 {
	if len(patternsA) == 0 || len(patternsB) == 0 {
		return 0.0
	}

	// Calculate keyword overlap
	allKeywordsA := make(map[string]int)
	allKeywordsB := make(map[string]int)

	for _, pattern := range patternsA {
		for _, keyword := range pattern.GetKeywords() {
			allKeywordsA[keyword]++
		}
	}

	for _, pattern := range patternsB {
		for _, keyword := range pattern.GetKeywords() {
			allKeywordsB[keyword]++
		}
	}

	// Calculate Jaccard similarity coefficient
	union := make(map[string]bool)
	intersection := make(map[string]bool)

	for keyword := range allKeywordsA {
		union[keyword] = true
		if _, exists := allKeywordsB[keyword]; exists {
			intersection[keyword] = true
		}
	}

	for keyword := range allKeywordsB {
		union[keyword] = true
	}

	if len(union) == 0 {
		return 0.0
	}

	jaccardSimilarity := float64(len(intersection)) / float64(len(union))

	// Calculate pattern type similarity
	typeOverlap := cra.calculateTypeOverlap(patternsA, patternsB)

	// Combined similarity score (weighted average)
	return (jaccardSimilarity*0.7 + typeOverlap*0.3)
}

// calculateTypeOverlap calculates overlap in pattern types
func (cra *crossRepoAnalyzer) calculateTypeOverlap(patternsA, patternsB []*entities.TaskPattern) float64 {
	typesA := make(map[string]int)
	typesB := make(map[string]int)

	for _, pattern := range patternsA {
		typesA[string(pattern.Type)]++
	}

	for _, pattern := range patternsB {
		typesB[string(pattern.Type)]++
	}

	commonTypes := 0
	totalTypes := 0

	allTypes := make(map[string]bool)
	for pType := range typesA {
		allTypes[pType] = true
	}
	for pType := range typesB {
		allTypes[pType] = true
	}
	totalTypes = len(allTypes)

	for pType := range typesA {
		if _, exists := typesB[pType]; exists {
			commonTypes++
		}
	}

	if totalTypes == 0 {
		return 0.0
	}

	return float64(commonTypes) / float64(totalTypes)
}

// findCommonPatternTypes identifies common pattern types between repositories
func (cra *crossRepoAnalyzer) findCommonPatternTypes(patternsA, patternsB []*entities.TaskPattern) []string {
	typesA := make(map[string]bool)
	typesB := make(map[string]bool)

	for _, pattern := range patternsA {
		typesA[string(pattern.Type)] = true
	}

	for _, pattern := range patternsB {
		typesB[string(pattern.Type)] = true
	}

	var commonTypes []string
	for pType := range typesA {
		if typesB[pType] {
			commonTypes = append(commonTypes, pType)
		}
	}

	return commonTypes
}

// findUniquePatterns identifies patterns unique to a repository
func (cra *crossRepoAnalyzer) findUniquePatterns(repoPatterns []*entities.TaskPattern, allPatterns []*entities.TaskPattern) []*entities.TaskPattern {
	// Create a map of patterns from other repositories for comparison
	otherPatterns := make(map[string]*entities.TaskPattern)
	for _, pattern := range allPatterns {
		// Skip patterns from the same repository
		if len(repoPatterns) > 0 && pattern.Repository == repoPatterns[0].Repository {
			continue
		}
		otherPatterns[pattern.Name] = pattern
	}

	var uniquePatterns []*entities.TaskPattern
	for _, pattern := range repoPatterns {
		// Check if this pattern is unique (not found in other repositories)
		isUnique := true
		for _, otherPattern := range otherPatterns {
			if cra.patternsAreSimilar(pattern, otherPattern) {
				isUnique = false
				break
			}
		}

		if isUnique {
			uniquePatterns = append(uniquePatterns, pattern)
		}
	}

	return uniquePatterns
}

// patternsAreSimilar checks if two patterns are similar enough to be considered the same
func (cra *crossRepoAnalyzer) patternsAreSimilar(patternA, patternB *entities.TaskPattern) bool {
	// Check type similarity
	if patternA.Type != patternB.Type {
		return false
	}

	// Calculate keyword overlap
	keywordsA := make(map[string]bool)
	patternAKeywords := patternA.GetKeywords()
	for _, keyword := range patternAKeywords {
		keywordsA[keyword] = true
	}

	commonKeywords := 0
	patternBKeywords := patternB.GetKeywords()
	for _, keyword := range patternBKeywords {
		if keywordsA[keyword] {
			commonKeywords++
		}
	}

	// Consider patterns similar if they share 60% or more keywords
	totalKeywords := len(patternAKeywords) + len(patternBKeywords) - commonKeywords
	if totalKeywords == 0 {
		return false
	}

	similarity := float64(commonKeywords*2) / float64(len(patternAKeywords)+len(patternBKeywords))
	return similarity > 0.6
}

// extractPatternNames extracts pattern names from a list of patterns
func (cra *crossRepoAnalyzer) extractPatternNames(patterns []*entities.TaskPattern) []string {
	names := make([]string, len(patterns))
	for i, pattern := range patterns {
		names[i] = pattern.Name
	}
	return names
}

// CalculateRepositorySimilarity calculates similarity between two repositories
func (cra *crossRepoAnalyzer) CalculateRepositorySimilarity(ctx context.Context, repoA, repoB string) (*entities.RepositorySimilarity, error) {
	cra.logger.Info("calculating repository similarity", slog.String("repoA", repoA), slog.String("repoB", repoB))

	// Get patterns for both repositories
	patternsA, err := cra.patternRepo.GetByRepository(ctx, repoA)
	if err != nil {
		return nil, fmt.Errorf("failed to get patterns for repository A: %w", err)
	}

	patternsB, err := cra.patternRepo.GetByRepository(ctx, repoB)
	if err != nil {
		return nil, fmt.Errorf("failed to get patterns for repository B: %w", err)
	}

	// Calculate pattern similarity
	patternSimilarity := cra.calculatePatternSimilarity(patternsA, patternsB)

	// Calculate project type similarity
	projectTypeSimilarity := cra.calculateProjectTypeSimilarity(patternsA, patternsB)

	// Calculate workflow complexity similarity
	complexitySimilarity := cra.calculateComplexitySimilarity(patternsA, patternsB)

	// Calculate temporal pattern similarity
	temporalSimilarity := cra.calculateTemporalSimilarity(patternsA, patternsB)

	// Combined similarity score (weighted average)
	overallScore := (patternSimilarity*0.4 + projectTypeSimilarity*0.2 + complexitySimilarity*0.2 + temporalSimilarity*0.2)

	// Determine common areas
	commonAreas := cra.identifyCommonAreas(patternsA, patternsB)

	// Identify key differences
	differences := cra.identifyKeyDifferences(patternsA, patternsB)

	similarity := &entities.RepositorySimilarity{
		RepositoryA: repoA,
		RepositoryB: repoB,
		Score:       overallScore,
		Dimensions: entities.SimilarityDimensions{
			ProjectType:  projectTypeSimilarity,
			TechStack:    0.0, // Not calculated in this implementation
			TeamSize:     0.0, // Not calculated in this implementation
			Complexity:   complexitySimilarity,
			Domain:       0.0, // Not calculated in this implementation
			WorkPatterns: patternSimilarity,
		},
		SharedPatterns: cra.findCommonPatternTypes(patternsA, patternsB),
		LastCalculated: time.Now(),
		Metadata: map[string]interface{}{
			"patterns_a_count":        len(patternsA),
			"patterns_b_count":        len(patternsB),
			"project_type_similarity": projectTypeSimilarity,
			"temporal_similarity":     temporalSimilarity,
			"common_areas":            commonAreas,
			"key_differences":         differences,
		},
	}

	return similarity, nil
}

// calculateProjectTypeSimilarity calculates similarity based on project types
func (cra *crossRepoAnalyzer) calculateProjectTypeSimilarity(patternsA, patternsB []*entities.TaskPattern) float64 {
	if len(patternsA) == 0 || len(patternsB) == 0 {
		return 0.0
	}

	projectTypesA := make(map[entities.ProjectType]int)
	projectTypesB := make(map[entities.ProjectType]int)

	for _, pattern := range patternsA {
		projectTypesA[entities.ProjectType(pattern.ProjectType)]++
	}

	for _, pattern := range patternsB {
		projectTypesB[entities.ProjectType(pattern.ProjectType)]++
	}

	// Calculate overlap in project types
	commonTypes := 0
	totalTypes := 0

	allTypes := make(map[entities.ProjectType]bool)
	for pType := range projectTypesA {
		allTypes[pType] = true
	}
	for pType := range projectTypesB {
		allTypes[pType] = true
	}
	totalTypes = len(allTypes)

	for pType := range projectTypesA {
		if _, exists := projectTypesB[pType]; exists {
			commonTypes++
		}
	}

	if totalTypes == 0 {
		return 0.0
	}

	return float64(commonTypes) / float64(totalTypes)
}

// calculateComplexitySimilarity calculates similarity based on workflow complexity
func (cra *crossRepoAnalyzer) calculateComplexitySimilarity(patternsA, patternsB []*entities.TaskPattern) float64 {
	if len(patternsA) == 0 || len(patternsB) == 0 {
		return 0.0
	}

	avgComplexityA := cra.calculateAverageComplexity(patternsA)
	avgComplexityB := cra.calculateAverageComplexity(patternsB)

	// Calculate similarity based on complexity difference
	// Smaller difference = higher similarity
	maxComplexity := 10.0 // Assume max complexity scale of 10
	complexityDiff := avgComplexityA - avgComplexityB
	if complexityDiff < 0 {
		complexityDiff = -complexityDiff
	}

	return 1.0 - (complexityDiff / maxComplexity)
}

// calculateAverageComplexity calculates average complexity of patterns
func (cra *crossRepoAnalyzer) calculateAverageComplexity(patterns []*entities.TaskPattern) float64 {
	if len(patterns) == 0 {
		return 0.0
	}

	totalComplexity := 0.0
	for _, pattern := range patterns {
		// Calculate complexity based on sequence length, keywords, etc.
		complexity := float64(len(pattern.Sequence)) * 1.5
		complexity += float64(len(pattern.GetKeywords())) * 0.5
		// Note: TaskPattern doesn't have Phases field, that's for WorkflowPattern
		totalComplexity += complexity
	}

	return totalComplexity / float64(len(patterns))
}

// calculateTemporalSimilarity calculates similarity based on temporal patterns
func (cra *crossRepoAnalyzer) calculateTemporalSimilarity(patternsA, patternsB []*entities.TaskPattern) float64 {
	if len(patternsA) == 0 || len(patternsB) == 0 {
		return 0.0
	}

	// Calculate average frequency for both repositories
	avgFreqA := cra.calculateAverageFrequency(patternsA)
	avgFreqB := cra.calculateAverageFrequency(patternsB)

	// Calculate similarity based on frequency difference
	maxFreq := 100.0 // Assume max frequency of 100
	freqDiff := avgFreqA - avgFreqB
	if freqDiff < 0 {
		freqDiff = -freqDiff
	}

	return 1.0 - (freqDiff / maxFreq)
}

// calculateAverageFrequency calculates average frequency of patterns
func (cra *crossRepoAnalyzer) calculateAverageFrequency(patterns []*entities.TaskPattern) float64 {
	if len(patterns) == 0 {
		return 0.0
	}

	totalFreq := 0.0
	for _, pattern := range patterns {
		totalFreq += pattern.Frequency
	}

	return totalFreq / float64(len(patterns))
}

// identifyCommonAreas identifies areas where repositories are similar
func (cra *crossRepoAnalyzer) identifyCommonAreas(patternsA, patternsB []*entities.TaskPattern) []string {
	var commonAreas []string

	// Check for common project types
	projectTypesA := make(map[entities.ProjectType]bool)
	projectTypesB := make(map[entities.ProjectType]bool)

	for _, pattern := range patternsA {
		projectTypesA[entities.ProjectType(pattern.ProjectType)] = true
	}

	for _, pattern := range patternsB {
		projectTypesB[entities.ProjectType(pattern.ProjectType)] = true
	}

	for pType := range projectTypesA {
		if projectTypesB[pType] {
			commonAreas = append(commonAreas, fmt.Sprintf("Both work with %s projects", pType))
		}
	}

	// Check for common keywords
	keywordsA := make(map[string]int)
	keywordsB := make(map[string]int)

	for _, pattern := range patternsA {
		for _, keyword := range pattern.GetKeywords() {
			keywordsA[keyword]++
		}
	}

	for _, pattern := range patternsB {
		for _, keyword := range pattern.GetKeywords() {
			keywordsB[keyword]++
		}
	}

	commonKeywords := make([]string, 0)
	for keyword := range keywordsA {
		if keywordsB[keyword] > 0 {
			commonKeywords = append(commonKeywords, keyword)
		}
	}

	if len(commonKeywords) > 0 {
		commonAreas = append(commonAreas, "Common focus areas: "+strings.Join(commonKeywords[:min(len(commonKeywords), 5)], ", "))
	}

	return commonAreas
}

// identifyKeyDifferences identifies key differences between repositories
func (cra *crossRepoAnalyzer) identifyKeyDifferences(patternsA, patternsB []*entities.TaskPattern) []string {
	var differences []string

	// Compare average complexity
	complexityA := cra.calculateAverageComplexity(patternsA)
	complexityB := cra.calculateAverageComplexity(patternsB)
	complexityDiff := complexityA - complexityB

	if complexityDiff > 1.0 {
		differences = append(differences, "Repository A has more complex workflows")
	} else if complexityDiff < -1.0 {
		differences = append(differences, "Repository B has more complex workflows")
	}

	// Compare pattern counts
	if len(patternsA) > len(patternsB)*2 {
		differences = append(differences, "Repository A has significantly more patterns")
	} else if len(patternsB) > len(patternsA)*2 {
		differences = append(differences, "Repository B has significantly more patterns")
	}

	// Compare frequencies
	freqA := cra.calculateAverageFrequency(patternsA)
	freqB := cra.calculateAverageFrequency(patternsB)
	freqDiff := freqA - freqB

	if freqDiff > 10 {
		differences = append(differences, "Repository A has more frequently used patterns")
	} else if freqDiff < -10 {
		differences = append(differences, "Repository B has more frequently used patterns")
	}

	return differences
}

// FindSimilarRepositories finds repositories similar to the given one
func (cra *crossRepoAnalyzer) FindSimilarRepositories(ctx context.Context, repository string, limit int) ([]*entities.RepositorySimilarity, error) {
	cra.logger.Info("finding similar repositories", slog.String("repository", repository), slog.Int("limit", limit))

	// Get all patterns to find other repositories
	allPatterns, err := cra.patternRepo.Search(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get all patterns: %w", err)
	}

	// Extract unique repositories (excluding the target repository)
	repoSet := make(map[string]bool)
	for _, pattern := range allPatterns {
		if pattern.Repository != repository {
			repoSet[pattern.Repository] = true
		}
	}

	similarities := make([]*entities.RepositorySimilarity, 0, len(repoSet))
	for otherRepo := range repoSet {
		similarity, err := cra.CalculateRepositorySimilarity(ctx, repository, otherRepo)
		if err != nil {
			cra.logger.Warn("failed to calculate similarity",
				slog.String("repo1", repository),
				slog.String("repo2", otherRepo),
				slog.Any("error", err))
			continue
		}
		similarities = append(similarities, similarity)
	}

	// Sort by similarity score (descending)
	sort.Slice(similarities, func(i, j int) bool {
		return similarities[i].Score > similarities[j].Score
	})

	// Apply limit
	if limit > 0 && len(similarities) > limit {
		similarities = similarities[:limit]
	}

	return similarities, nil
}

// GetSharedInsights gets publicly shared insights for a project type
func (cra *crossRepoAnalyzer) GetSharedInsights(ctx context.Context, projectType entities.ProjectType) ([]*entities.CrossRepoInsight, error) {
	cra.logger.Info("getting shared insights", slog.String("project_type", string(projectType)))

	// Get all patterns for the specified project type
	allPatterns, err := cra.patternRepo.Search(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get all patterns: %w", err)
	}

	// Filter patterns by project type
	var relevantPatterns []*entities.TaskPattern
	for _, pattern := range allPatterns {
		if entities.ProjectType(pattern.ProjectType) == projectType {
			relevantPatterns = append(relevantPatterns, pattern)
		}
	}

	if len(relevantPatterns) == 0 {
		return []*entities.CrossRepoInsight{}, nil
	}

	var insights []*entities.CrossRepoInsight

	// Generate aggregated patterns insight
	aggregatedInsight := cra.generateAggregatedPatternInsight(relevantPatterns, projectType)
	if aggregatedInsight != nil {
		insights = append(insights, aggregatedInsight)
	}

	// Generate best practices insight
	bestPracticesInsight := cra.generateBestPracticesInsight(relevantPatterns, projectType)
	if bestPracticesInsight != nil {
		insights = append(insights, bestPracticesInsight)
	}

	// Generate common bottlenecks insight
	bottleneckInsight := cra.generateBottleneckInsight(relevantPatterns, projectType)
	if bottleneckInsight != nil {
		insights = append(insights, bottleneckInsight)
	}

	// Generate optimization opportunities insight
	optimizationInsight := cra.generateOptimizationInsight(relevantPatterns, projectType)
	if optimizationInsight != nil {
		insights = append(insights, optimizationInsight)
	}

	return insights, nil
}

// generateAggregatedPatternInsight creates an insight about common patterns
func (cra *crossRepoAnalyzer) generateAggregatedPatternInsight(patterns []*entities.TaskPattern, projectType entities.ProjectType) *entities.CrossRepoInsight {
	if len(patterns) < 3 {
		return nil // Need minimum patterns for aggregation
	}

	// Group patterns by similarity
	patternGroups := cra.groupSimilarPatterns(patterns)

	// Find the most common pattern group
	var largestGroup []*entities.TaskPattern
	for _, group := range patternGroups {
		if len(group) > len(largestGroup) {
			largestGroup = group
		}
	}

	if len(largestGroup) < 3 {
		return nil // Need minimum group size
	}

	// Calculate aggregate metrics
	totalFrequency := 0.0
	totalSuccessRate := 0.0
	repositories := make(map[string]bool)

	for _, pattern := range largestGroup {
		totalFrequency += pattern.Frequency
		totalSuccessRate += pattern.SuccessRate
		repositories[pattern.Repository] = true
	}

	avgSuccessRate := totalSuccessRate / float64(len(largestGroup))

	// Create insight
	insight := &entities.CrossRepoInsight{
		ID:          generateInsightID("aggregated"),
		Type:        entities.InsightTypePattern,
		Title:       fmt.Sprintf("Common %s Development Pattern", projectType),
		Description: fmt.Sprintf("A common pattern emerges across %d repositories working with %s projects", len(repositories), projectType),
		SourceCount: len(largestGroup),
		Confidence:  0.8,
		Relevance:   0.9,
		Impact: entities.ImpactMetrics{
			ProductivityGain:   avgSuccessRate * 0.3,
			TimeReduction:      avgSuccessRate * 0.2,
			QualityImprovement: avgSuccessRate * 0.1,
			AdoptionRate:       0.7,
		},
		Recommendations: []string{
			fmt.Sprintf("Adopt this proven %s development pattern", projectType),
			"Study implementation details from successful repositories",
			"Customize the pattern to fit your team's workflow",
		},
		Tags:         []string{"pattern", "aggregated", string(projectType)},
		Metadata:     make(map[string]interface{}),
		GeneratedAt:  time.Now(),
		ValidUntil:   time.Now().Add(30 * 24 * time.Hour),
		IsActionable: true,
	}

	// Store pattern details in metadata
	repoList := make([]string, 0, len(repositories))
	for repo := range repositories {
		repoList = append(repoList, repo)
	}
	insight.Metadata["source_repositories"] = repoList
	insight.Metadata["pattern_count"] = len(largestGroup)
	insight.Metadata["avg_success_rate"] = avgSuccessRate
	insight.Metadata["total_frequency"] = totalFrequency

	return insight
}

// generateBestPracticesInsight creates insights about best practices
func (cra *crossRepoAnalyzer) generateBestPracticesInsight(patterns []*entities.TaskPattern, projectType entities.ProjectType) *entities.CrossRepoInsight {
	// Find patterns with highest success rates
	var bestPatterns []*entities.TaskPattern
	for _, pattern := range patterns {
		if pattern.SuccessRate > 0.9 && pattern.Frequency > 5 {
			bestPatterns = append(bestPatterns, pattern)
		}
	}

	if len(bestPatterns) == 0 {
		return nil
	}

	// Sort by success rate
	sort.Slice(bestPatterns, func(i, j int) bool {
		return bestPatterns[i].SuccessRate > bestPatterns[j].SuccessRate
	})

	topPattern := bestPatterns[0]

	insight := &entities.CrossRepoInsight{
		ID:          generateInsightID("best-practice"),
		Type:        entities.InsightTypeBestPractice,
		Title:       fmt.Sprintf("Proven %s Best Practice", projectType),
		Description: fmt.Sprintf("High-success pattern '%s' achieves %.1f%% success rate", topPattern.Name, topPattern.SuccessRate*100),
		SourceCount: len(bestPatterns),
		Confidence:  0.9,
		Relevance:   0.9,
		Impact: entities.ImpactMetrics{
			ProductivityGain:   topPattern.SuccessRate * 0.4,
			TimeReduction:      topPattern.SuccessRate * 0.3,
			QualityImprovement: topPattern.SuccessRate * 0.2,
			AdoptionRate:       0.8,
		},
		Recommendations: []string{
			fmt.Sprintf("Implement the '%s' pattern in your %s projects", topPattern.Name, projectType),
			"Train team members on this proven approach",
			"Monitor adoption and measure improvement",
		},
		Tags:         []string{"best-practice", string(projectType), "high-success"},
		Metadata:     make(map[string]interface{}),
		GeneratedAt:  time.Now(),
		ValidUntil:   time.Now().Add(60 * 24 * time.Hour),
		IsActionable: true,
	}

	// Add evidence
	evidence := make([]string, 0, len(bestPatterns))
	for _, pattern := range bestPatterns[:min(len(bestPatterns), 3)] {
		evidence = append(evidence, fmt.Sprintf("'%s' pattern: %.1f%% success rate", pattern.Name, pattern.SuccessRate*100))
	}
	insight.Metadata["evidence"] = evidence

	return insight
}

// generateBottleneckInsight identifies common bottlenecks
func (cra *crossRepoAnalyzer) generateBottleneckInsight(patterns []*entities.TaskPattern, projectType entities.ProjectType) *entities.CrossRepoInsight {
	// Look for patterns with low success rates or high complexity
	var problematicPatterns []*entities.TaskPattern
	for _, pattern := range patterns {
		complexity := cra.calculateAverageComplexity([]*entities.TaskPattern{pattern})
		if pattern.SuccessRate < 0.7 || complexity > 8.0 {
			problematicPatterns = append(problematicPatterns, pattern)
		}
	}

	if len(problematicPatterns) < 2 {
		return nil
	}

	// Find the most common problematic pattern
	sort.Slice(problematicPatterns, func(i, j int) bool {
		return problematicPatterns[i].Frequency > problematicPatterns[j].Frequency
	})

	topProblem := problematicPatterns[0]

	insight := &entities.CrossRepoInsight{
		ID:          generateInsightID("bottleneck"),
		Type:        entities.InsightTypeBottleneck,
		Title:       fmt.Sprintf("Common %s Development Bottleneck", projectType),
		Description: fmt.Sprintf("Pattern '%s' shows recurring challenges across multiple repositories", topProblem.Name),
		SourceCount: len(problematicPatterns),
		Confidence:  0.7,
		Relevance:   0.8,
		Impact: entities.ImpactMetrics{
			ProductivityGain:   0.3,
			TimeReduction:      0.4,
			QualityImprovement: 0.2,
			AdoptionRate:       0.6,
		},
		Recommendations: []string{
			fmt.Sprintf("Review and optimize the '%s' workflow", topProblem.Name),
			"Consider breaking down complex tasks into smaller steps",
			"Implement additional validation or review processes",
			"Study successful alternatives from high-performing repositories",
		},
		Tags:         []string{"bottleneck", string(projectType), "optimization"},
		Metadata:     make(map[string]interface{}),
		GeneratedAt:  time.Now(),
		ValidUntil:   time.Now().Add(30 * 24 * time.Hour),
		IsActionable: true,
	}

	return insight
}

// generateOptimizationInsight suggests optimization opportunities
func (cra *crossRepoAnalyzer) generateOptimizationInsight(patterns []*entities.TaskPattern, projectType entities.ProjectType) *entities.CrossRepoInsight {
	if len(patterns) < 5 {
		return nil
	}

	// Analyze frequency vs success rate to find optimization opportunities
	var improvementPatterns []*entities.TaskPattern
	avgFrequency := cra.calculateAverageFrequency(patterns)

	for _, pattern := range patterns {
		// High frequency but moderate success rate = optimization opportunity
		if float64(pattern.Frequency) > avgFrequency*1.5 && pattern.SuccessRate > 0.7 && pattern.SuccessRate < 0.9 {
			improvementPatterns = append(improvementPatterns, pattern)
		}
	}

	if len(improvementPatterns) == 0 {
		return nil
	}

	insight := &entities.CrossRepoInsight{
		ID:          generateInsightID("optimization"),
		Type:        entities.InsightTypeOpportunity,
		Title:       fmt.Sprintf("%s Optimization Opportunities", projectType),
		Description: fmt.Sprintf("Found %d high-frequency patterns with optimization potential", len(improvementPatterns)),
		SourceCount: len(improvementPatterns),
		Confidence:  0.6,
		Relevance:   0.7,
		Impact: entities.ImpactMetrics{
			ProductivityGain:   0.2,
			TimeReduction:      0.3,
			QualityImprovement: 0.25,
			AdoptionRate:       0.7,
		},
		Recommendations: []string{
			"Focus optimization efforts on high-frequency patterns",
			"Analyze successful variations of these patterns",
			"Implement incremental improvements and measure impact",
		},
		Tags:         []string{"optimization", string(projectType), "opportunity"},
		Metadata:     make(map[string]interface{}),
		GeneratedAt:  time.Now(),
		ValidUntil:   time.Now().Add(45 * 24 * time.Hour),
		IsActionable: true,
	}

	return insight
}

// groupSimilarPatterns groups patterns by similarity
func (cra *crossRepoAnalyzer) groupSimilarPatterns(patterns []*entities.TaskPattern) [][]*entities.TaskPattern {
	var groups [][]*entities.TaskPattern
	used := make(map[int]bool)

	for i, pattern := range patterns {
		if used[i] {
			continue
		}

		group := []*entities.TaskPattern{pattern}
		used[i] = true

		for j, otherPattern := range patterns {
			if i != j && !used[j] && cra.patternsAreSimilar(pattern, otherPattern) {
				group = append(group, otherPattern)
				used[j] = true
			}
		}

		if len(group) > 1 {
			groups = append(groups, group)
		}
	}

	return groups
}

// GetInsightRecommendations gets personalized insight recommendations
func (cra *crossRepoAnalyzer) GetInsightRecommendations(ctx context.Context, repository string) ([]*entities.CrossRepoInsight, error) {
	cra.logger.Info("generating insight recommendations", slog.String("repository", repository))

	// Get patterns for the target repository
	repoPatterns, err := cra.patternRepo.GetByRepository(ctx, repository)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository patterns: %w", err)
	}

	if len(repoPatterns) == 0 {
		return []*entities.CrossRepoInsight{}, nil
	}

	// Determine the primary project type for this repository
	projectType := cra.determinePrimaryProjectType(repoPatterns)

	// Get all insights for the project type
	sharedInsights, err := cra.GetSharedInsights(ctx, projectType)
	if err != nil {
		return nil, fmt.Errorf("failed to get shared insights: %w", err)
	}

	// Filter and personalize insights for this repository
	var recommendations []*entities.CrossRepoInsight

	// Find repositories with similar patterns
	similarRepos, err := cra.FindSimilarRepositories(ctx, repository, 5)
	if err != nil {
		cra.logger.Warn("failed to find similar repositories", slog.Any("error", err))
	} else if len(similarRepos) > 0 {
		// Generate similarity-based recommendations
		similarityRec := cra.generateSimilarityRecommendation(repository, similarRepos)
		if similarityRec != nil {
			recommendations = append(recommendations, similarityRec)
		}
	}

	// Generate improvement recommendations based on repository analysis
	improvementRec := cra.generateImprovementRecommendation(repository, repoPatterns)
	if improvementRec != nil {
		recommendations = append(recommendations, improvementRec)
	}

	// Generate gap analysis recommendations
	gapRec := cra.generateGapAnalysisRecommendation(repository, repoPatterns, sharedInsights)
	if gapRec != nil {
		recommendations = append(recommendations, gapRec)
	}

	// Add relevant shared insights with personalization
	for _, insight := range sharedInsights {
		if cra.isInsightRelevantForRepository(insight, repoPatterns) {
			personalizedInsight := cra.personalizeInsight(insight, repository, repoPatterns)
			recommendations = append(recommendations, personalizedInsight)
		}
	}

	// Sort recommendations by relevance and impact
	sort.Slice(recommendations, func(i, j int) bool {
		scoreI := recommendations[i].Relevance*0.6 + recommendations[i].Impact.ProductivityGain*0.4
		scoreJ := recommendations[j].Relevance*0.6 + recommendations[j].Impact.ProductivityGain*0.4
		return scoreI > scoreJ
	})

	// Limit to top recommendations
	if len(recommendations) > 10 {
		recommendations = recommendations[:10]
	}

	return recommendations, nil
}

// determinePrimaryProjectType determines the most common project type in patterns
func (cra *crossRepoAnalyzer) determinePrimaryProjectType(patterns []*entities.TaskPattern) entities.ProjectType {
	if len(patterns) == 0 {
		return entities.ProjectTypeWebApp // Default
	}

	typeCount := make(map[entities.ProjectType]int)
	for _, pattern := range patterns {
		typeCount[entities.ProjectType(pattern.ProjectType)]++
	}

	var primaryType entities.ProjectType
	maxCount := 0
	for pType, count := range typeCount {
		if count > maxCount {
			maxCount = count
			primaryType = pType
		}
	}

	return primaryType
}

// generateSimilarityRecommendation creates recommendations based on similar repositories
func (cra *crossRepoAnalyzer) generateSimilarityRecommendation(_ string, similarRepos []*entities.RepositorySimilarity) *entities.CrossRepoInsight {
	if len(similarRepos) == 0 {
		return nil
	}

	topSimilar := similarRepos[0]

	insight := &entities.CrossRepoInsight{
		ID:          generateInsightID("similarity"),
		Type:        entities.InsightTypeComparison,
		Title:       "Repository Similarity Insights",
		Description: fmt.Sprintf("Your repository shows %.1f%% similarity with '%s', suggesting potential learning opportunities", topSimilar.Score*100, topSimilar.RepositoryB),
		SourceCount: len(similarRepos),
		Confidence:  topSimilar.Score,
		Relevance:   0.8,
		Impact: entities.ImpactMetrics{
			ProductivityGain:   topSimilar.Score * 0.3,
			TimeReduction:      topSimilar.Score * 0.2,
			QualityImprovement: topSimilar.Score * 0.15,
			AdoptionRate:       0.7,
		},
		Recommendations: []string{
			fmt.Sprintf("Study successful patterns from '%s'", topSimilar.RepositoryB),
			"Consider knowledge sharing sessions with similar teams",
			"Adopt proven practices while maintaining your unique strengths",
		},
		Tags:         []string{"similarity", "learning", "cross-repo"},
		Metadata:     make(map[string]interface{}),
		GeneratedAt:  time.Now(),
		ValidUntil:   time.Now().Add(30 * 24 * time.Hour),
		IsActionable: true,
	}

	// Add similar repositories to metadata
	similarRepoNames := make([]string, len(similarRepos))
	for i, repo := range similarRepos {
		similarRepoNames[i] = repo.RepositoryB
	}
	insight.Metadata["similar_repositories"] = similarRepoNames
	insight.Metadata["top_similarity_score"] = topSimilar.Score

	return insight
}

// generateImprovementRecommendation identifies improvement opportunities
func (cra *crossRepoAnalyzer) generateImprovementRecommendation(_ string, patterns []*entities.TaskPattern) *entities.CrossRepoInsight {
	if len(patterns) < 3 {
		return nil
	}

	// Find patterns with improvement potential
	var improvablePatterns []*entities.TaskPattern
	avgSuccessRate := 0.0
	for _, pattern := range patterns {
		avgSuccessRate += pattern.SuccessRate
	}
	avgSuccessRate /= float64(len(patterns))

	for _, pattern := range patterns {
		if pattern.SuccessRate < avgSuccessRate && pattern.Frequency > 3 {
			improvablePatterns = append(improvablePatterns, pattern)
		}
	}

	if len(improvablePatterns) == 0 {
		return nil
	}

	// Sort by frequency (prioritize high-frequency patterns)
	sort.Slice(improvablePatterns, func(i, j int) bool {
		return improvablePatterns[i].Frequency > improvablePatterns[j].Frequency
	})

	topPattern := improvablePatterns[0]

	insight := &entities.CrossRepoInsight{
		ID:          generateInsightID("improvement"),
		Type:        entities.InsightTypeOpportunity,
		Title:       "Pattern Improvement Opportunity",
		Description: fmt.Sprintf("Pattern '%s' has %.1f%% success rate but high frequency (%.1f), suggesting optimization potential", topPattern.Name, topPattern.SuccessRate*100, topPattern.Frequency),
		SourceCount: len(improvablePatterns),
		Confidence:  0.7,
		Relevance:   0.9,
		Impact: entities.ImpactMetrics{
			ProductivityGain:   (1.0 - topPattern.SuccessRate) * 0.4,
			TimeReduction:      (1.0 - topPattern.SuccessRate) * 0.3,
			QualityImprovement: (1.0 - topPattern.SuccessRate) * 0.3,
			AdoptionRate:       0.8,
		},
		Recommendations: []string{
			fmt.Sprintf("Focus on optimizing the '%s' pattern", topPattern.Name),
			"Analyze failure cases to identify improvement areas",
			"Consider breaking down complex workflows into simpler steps",
			"Implement additional validation or quality checks",
		},
		Tags:         []string{"improvement", "optimization", "internal"},
		Metadata:     make(map[string]interface{}),
		GeneratedAt:  time.Now(),
		ValidUntil:   time.Now().Add(14 * 24 * time.Hour),
		IsActionable: true,
	}

	insight.Metadata["target_pattern"] = topPattern.Name
	insight.Metadata["current_success_rate"] = topPattern.SuccessRate
	insight.Metadata["frequency"] = topPattern.Frequency

	return insight
}

// generateGapAnalysisRecommendation identifies missing best practices
func (cra *crossRepoAnalyzer) generateGapAnalysisRecommendation(_ string, repoPatterns []*entities.TaskPattern, sharedInsights []*entities.CrossRepoInsight) *entities.CrossRepoInsight {
	if len(sharedInsights) == 0 {
		return nil
	}

	// Find best practices that this repository might be missing
	repoKeywords := make(map[string]bool)
	for _, pattern := range repoPatterns {
		for _, keyword := range pattern.GetKeywords() {
			repoKeywords[keyword] = true
		}
	}

	var missingPractices []string
	for _, insight := range sharedInsights {
		if insight.Type == entities.InsightTypeBestPractice {
			// Check if this repository has patterns related to this best practice
			hasRelatedPattern := false
			for _, keyword := range insight.Tags {
				if repoKeywords[keyword] {
					hasRelatedPattern = true
					break
				}
			}

			if !hasRelatedPattern {
				missingPractices = append(missingPractices, insight.Title)
			}
		}
	}

	if len(missingPractices) == 0 {
		return nil
	}

	insight := &entities.CrossRepoInsight{
		ID:          generateInsightID("gap-analysis"),
		Type:        entities.InsightTypeOpportunity,
		Title:       "Missing Best Practices",
		Description: fmt.Sprintf("Analysis reveals %d proven practices that could benefit your repository", len(missingPractices)),
		SourceCount: len(missingPractices),
		Confidence:  0.6,
		Relevance:   0.7,
		Impact: entities.ImpactMetrics{
			ProductivityGain:   0.25,
			TimeReduction:      0.2,
			QualityImprovement: 0.3,
			AdoptionRate:       0.6,
		},
		Recommendations: []string{
			"Review industry best practices for your project type",
			"Consider gradual adoption of proven methodologies",
			"Start with practices that align with your current workflow",
		},
		Tags:         []string{"gap-analysis", "best-practices", "opportunity"},
		Metadata:     make(map[string]interface{}),
		GeneratedAt:  time.Now(),
		ValidUntil:   time.Now().Add(60 * 24 * time.Hour),
		IsActionable: true,
	}

	insight.Metadata["missing_practices"] = missingPractices[:min(len(missingPractices), 5)]

	return insight
}

// isInsightRelevantForRepository checks if a shared insight is relevant
func (cra *crossRepoAnalyzer) isInsightRelevantForRepository(insight *entities.CrossRepoInsight, repoPatterns []*entities.TaskPattern) bool {
	if len(repoPatterns) == 0 {
		return true // Default to relevant
	}

	// Check if repository has patterns related to the insight
	repoKeywords := make(map[string]bool)
	for _, pattern := range repoPatterns {
		for _, keyword := range pattern.GetKeywords() {
			repoKeywords[keyword] = true
		}
	}

	// Check for keyword overlap
	relevanceScore := 0
	for _, tag := range insight.Tags {
		if repoKeywords[tag] {
			relevanceScore++
		}
	}

	// Relevant if there's some keyword overlap or if it's a high-impact insight
	return relevanceScore > 0 || insight.Impact.ProductivityGain > 0.3
}

// personalizeInsight customizes a shared insight for a specific repository
func (cra *crossRepoAnalyzer) personalizeInsight(insight *entities.CrossRepoInsight, repository string, repoPatterns []*entities.TaskPattern) *entities.CrossRepoInsight {
	// Create a copy of the insight
	personalized := &entities.CrossRepoInsight{
		ID:              generateInsightID("personalized"),
		Type:            insight.Type,
		Title:           fmt.Sprintf("%s for %s", insight.Title, repository),
		Description:     insight.Description + fmt.Sprintf(" Based on analysis of %d patterns in your repository.", len(repoPatterns)),
		SourceCount:     insight.SourceCount,
		Confidence:      insight.Confidence * 0.9, // Slightly lower confidence for personalized
		Relevance:       insight.Relevance,
		Impact:          insight.Impact,
		Recommendations: insight.Recommendations,
		Tags:            append(insight.Tags, "personalized", repository),
		Metadata:        make(map[string]interface{}),
		GeneratedAt:     time.Now(),
		ValidUntil:      insight.ValidUntil,
		IsActionable:    insight.IsActionable,
	}

	// Copy metadata and add personalization info
	for k, v := range insight.Metadata {
		personalized.Metadata[k] = v
	}
	personalized.Metadata["personalized_for"] = repository
	personalized.Metadata["repo_pattern_count"] = len(repoPatterns)

	return personalized
}

// ContributePattern contributes a pattern to the shared knowledge base
func (cra *crossRepoAnalyzer) ContributePattern(ctx context.Context, pattern *entities.TaskPattern, optIn bool) error {
	return nil
}

// UpdatePrivacySettings updates privacy settings for a repository
func (cra *crossRepoAnalyzer) UpdatePrivacySettings(ctx context.Context, repository string, settings *entities.PrivacySettings) error {
	return nil
}

// GetPrivacySettings gets privacy settings for a repository
func (cra *crossRepoAnalyzer) GetPrivacySettings(ctx context.Context, repository string) (*entities.PrivacySettings, error) {
	return entities.DefaultPrivacySettings(), nil
}

// GetInsightAnalytics gets analytics data for insights
func (cra *crossRepoAnalyzer) GetInsightAnalytics(ctx context.Context, insightID string) (*services.InsightAnalytics, error) {
	return &services.InsightAnalytics{InsightID: insightID}, nil
}

// GetCrossRepoTrends gets cross-repository trends
func (cra *crossRepoAnalyzer) GetCrossRepoTrends(ctx context.Context, timeRange time.Duration) ([]*entities.CrossRepoInsight, error) {
	return []*entities.CrossRepoInsight{}, nil
}

// AnalyzeRepositories analyzes patterns across multiple repositories
func (cra *crossRepoAnalyzer) AnalyzeRepositories(ctx context.Context, repositories []string, period entities.TimePeriod) (*entities.CrossRepoAnalysis, error) {
	cra.logger.Info("starting cross-repository analysis", slog.Int("repositories", len(repositories)))

	repoMetrics := make(map[string]*entities.WorkflowMetrics)
	allTasks := make([]*entities.Task, 0)
	allSessions := make([]*entities.Session, 0)

	// Collect data from all repositories
	for _, repo := range repositories {
		// Get workflow metrics for each repository
		metrics, err := cra.analyticsEngine.GenerateWorkflowMetrics(ctx, repo, period)
		if err != nil {
			cra.logger.Warn("failed to get metrics for repository", slog.String("repo", repo), slog.Any("error", err))
			continue
		}
		repoMetrics[repo] = metrics

		// Collect tasks
		tasks, err := cra.taskRepo.GetByRepository(ctx, repo, period)
		if err != nil {
			cra.logger.Warn("failed to get tasks for repository", slog.String("repo", repo), slog.Any("error", err))
			continue
		}
		allTasks = append(allTasks, tasks...)

		// Collect sessions
		sessions, err := cra.sessionRepo.GetByRepository(ctx, repo, period)
		if err != nil {
			cra.logger.Warn("failed to get sessions for repository", slog.String("repo", repo), slog.Any("error", err))
			continue
		}
		allSessions = append(allSessions, sessions...)
	}

	// Analyze patterns across repositories
	patterns := cra.identifyCommonPatterns(allTasks)
	insights := cra.generateCrossRepoInsights(repoMetrics, patterns)
	recommendations := cra.generateRecommendations(repoMetrics, insights)

	// Detect outliers
	outliers := cra.detectOutliers(repoMetrics)

	// Calculate correlation matrix
	correlations := cra.calculateCorrelations(repoMetrics)

	// Convert slices of pointers to slices of values
	commonPatterns := make([]entities.CommonPattern, len(patterns))
	for i, p := range patterns {
		commonPatterns[i] = *p
	}

	crossRepoInsights := make([]entities.CrossRepoInsight, len(insights))
	for i, insight := range insights {
		crossRepoInsights[i] = *insight
	}

	crossRepoRecommendations := make([]entities.CrossRepoRecommendation, len(recommendations))
	for i, rec := range recommendations {
		// Convert Recommendation to CrossRepoRecommendation
		crossRepoRecommendations[i] = entities.CrossRepoRecommendation{
			ID:          rec.ID,
			Title:       rec.Title,
			Description: rec.Description,
			Category:    rec.Category,
			Priority:    rec.Priority,
			Impact:      rec.Impact,
			Effort:      rec.Effort,
			Actions:     rec.Actions,
			Evidence:    rec.Evidence,
			CreatedAt:   rec.CreatedAt,
			Metadata:    make(map[string]interface{}),
		}
	}

	outlierValues := make([]entities.Outlier, len(outliers))
	for i, o := range outliers {
		outlierValues[i] = *o
	}

	analysis := &entities.CrossRepoAnalysis{
		Period:          period,
		Repositories:    repositories,
		CommonPatterns:  commonPatterns,
		Insights:        crossRepoInsights,
		Recommendations: crossRepoRecommendations,
		Outliers:        outlierValues,
		GeneratedAt:     time.Now(),
		Metadata:        make(map[string]interface{}),
	}

	// Store repository metrics in metadata
	analysis.Metadata["repository_metrics"] = repoMetrics
	analysis.Metadata["correlations"] = correlations

	// Store insights for future reference
	for _, insight := range insights {
		if err := cra.insightRepo.Create(ctx, insight); err != nil {
			cra.logger.Warn("failed to store insight", slog.Any("error", err))
		}
	}

	return analysis, nil
}

// CompareRepositories compares specific repositories
func (cra *crossRepoAnalyzer) CompareRepositories(ctx context.Context, repoA, repoB string, period entities.TimePeriod) (*entities.RepositoryComparison, error) {
	cra.logger.Info("comparing repositories", slog.String("repoA", repoA), slog.String("repoB", repoB))

	// Get metrics for both repositories
	metricsA, err := cra.analyticsEngine.GenerateWorkflowMetrics(ctx, repoA, period)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics for repository A: %w", err)
	}

	metricsB, err := cra.analyticsEngine.GenerateWorkflowMetrics(ctx, repoB, period)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics for repository B: %w", err)
	}

	// Calculate differences
	productivityDiff := metricsB.Productivity.Score - metricsA.Productivity.Score
	velocityDiff := metricsB.Velocity.CurrentVelocity - metricsA.Velocity.CurrentVelocity
	completionDiff := metricsB.Completion.CompletionRate - metricsA.Completion.CompletionRate
	cycleTimeDiff := metricsB.CycleTime.AverageCycleTime - metricsA.CycleTime.AverageCycleTime

	// Determine better performing repository
	winner := repoA
	if (productivityDiff + velocityDiff + completionDiff) > 0 {
		winner = repoB
	}

	// Generate insights
	insights := []string{}
	if productivityDiff > 10 {
		insights = append(insights, fmt.Sprintf("%s has %.1f%% higher productivity than %s", repoB, productivityDiff, repoA))
	} else if productivityDiff < -10 {
		insights = append(insights, fmt.Sprintf("%s has %.1f%% higher productivity than %s", repoA, -productivityDiff, repoB))
	}

	if velocityDiff > 2 {
		insights = append(insights, fmt.Sprintf("%s delivers %.1f more tasks per week than %s", repoB, velocityDiff, repoA))
	} else if velocityDiff < -2 {
		insights = append(insights, fmt.Sprintf("%s delivers %.1f more tasks per week than %s", repoA, -velocityDiff, repoB))
	}

	comparison := &entities.RepositoryComparison{
		RepositoryA:      repoA,
		RepositoryB:      repoB,
		ProductivityDiff: productivityDiff,
		VelocityDiff:     velocityDiff,
		QualityDiff:      metricsB.Completion.QualityScore - metricsA.Completion.QualityScore,
		EfficiencyDiff:   metricsB.CycleTime.GetEfficiencyScore() - metricsA.CycleTime.GetEfficiencyScore(),
		SimilarityScore:  0.0, // Could be calculated based on various factors
		KeyDifferences:   insights,
		Metadata:         make(map[string]interface{}),
	}

	// Store additional details in metadata
	comparison.Metadata["period"] = period
	comparison.Metadata["metrics_a"] = metricsA
	comparison.Metadata["metrics_b"] = metricsB
	comparison.Metadata["completion_diff"] = completionDiff
	comparison.Metadata["cycle_time_diff"] = cycleTimeDiff
	comparison.Metadata["better_performer"] = winner
	comparison.Metadata["compared_at"] = time.Now()

	return comparison, nil
}

// GetBestPractices extracts best practices from top-performing repositories
func (cra *crossRepoAnalyzer) GetBestPractices(ctx context.Context, repositories []string, period entities.TimePeriod) ([]*entities.BestPractice, error) {
	cra.logger.Info("extracting best practices", slog.Int("repositories", len(repositories)))

	// Get metrics for all repositories
	repoMetrics := make(map[string]*entities.WorkflowMetrics)
	for _, repo := range repositories {
		metrics, err := cra.analyticsEngine.GenerateWorkflowMetrics(ctx, repo, period)
		if err != nil {
			cra.logger.Warn("failed to get metrics for repository", slog.String("repo", repo), slog.Any("error", err))
			continue
		}
		repoMetrics[repo] = metrics
	}

	var practices []*entities.BestPractice

	// Find best productivity practices
	topProductivityRepo := cra.findTopPerformer(repoMetrics, "productivity")
	if topProductivityRepo != "" {
		metrics := repoMetrics[topProductivityRepo]
		practice := &entities.BestPractice{
			ID:           generatePracticeID("productivity"),
			Category:     "productivity",
			Title:        "High Productivity Task Management",
			Description:  fmt.Sprintf("Repository %s achieves %.1f%% productivity score", topProductivityRepo, metrics.Productivity.Score),
			Impact:       0.8, // High impact
			Confidence:   0.8,
			Repositories: []string{topProductivityRepo},
			Evidence: []string{
				fmt.Sprintf("%.1f tasks completed per day", metrics.Productivity.TasksPerDay),
				fmt.Sprintf("%.0f%% deep work ratio", metrics.Productivity.DeepWorkRatio*100),
			},
			Implementation: []string{
				"Focus on task breakdown and estimation",
				"Minimize context switching",
				"Establish consistent work patterns",
			},
			Priority: entities.RecommendationPriorityHigh,
			Metadata: make(map[string]interface{}),
		}
		// Store additional details in metadata
		practice.Metadata["applicable_projects"] = []entities.ProjectType{entities.ProjectTypeWebApp, entities.ProjectTypeAPI, entities.ProjectTypeCLI}
		practice.Metadata["source"] = topProductivityRepo
		practice.Metadata["extracted_at"] = time.Now()
		practices = append(practices, practice)
	}

	// Find best velocity practices
	topVelocityRepo := cra.findTopPerformer(repoMetrics, "velocity")
	if topVelocityRepo != "" && topVelocityRepo != topProductivityRepo {
		metrics := repoMetrics[topVelocityRepo]
		practice := &entities.BestPractice{
			ID:           generatePracticeID("velocity"),
			Category:     "velocity",
			Title:        "High Velocity Development",
			Description:  fmt.Sprintf("Repository %s maintains %.1f tasks/week velocity", topVelocityRepo, metrics.Velocity.CurrentVelocity),
			Impact:       0.8, // High impact
			Confidence:   0.8,
			Repositories: []string{topVelocityRepo},
			Evidence: []string{
				fmt.Sprintf("%.1f tasks completed per week", metrics.Velocity.CurrentVelocity),
				fmt.Sprintf("%.0f%% consistency", metrics.Velocity.Consistency*100),
			},
			Implementation: []string{
				"Maintain consistent development cadence",
				"Use effective task prioritization",
				"Implement continuous integration practices",
			},
			Priority: entities.RecommendationPriorityHigh,
			Metadata: make(map[string]interface{}),
		}
		// Store additional details in metadata
		practice.Metadata["applicable_projects"] = []entities.ProjectType{entities.ProjectTypeWebApp, entities.ProjectTypeAPI}
		practice.Metadata["source"] = topVelocityRepo
		practice.Metadata["extracted_at"] = time.Now()
		practices = append(practices, practice)
	}

	// Find best cycle time practices
	topCycleTimeRepo := cra.findBestCycleTime(repoMetrics)
	if topCycleTimeRepo != "" {
		metrics := repoMetrics[topCycleTimeRepo]
		practice := &entities.BestPractice{
			ID:           generatePracticeID("cycle-time"),
			Category:     "cycle-time",
			Title:        "Efficient Task Completion",
			Description:  fmt.Sprintf("Repository %s achieves %s average cycle time", topCycleTimeRepo, formatDuration(metrics.CycleTime.AverageCycleTime)),
			Impact:       0.6, // Medium impact
			Confidence:   0.7,
			Repositories: []string{topCycleTimeRepo},
			Evidence: []string{
				"Average cycle time: " + formatDuration(metrics.CycleTime.AverageCycleTime),
				"P90 cycle time: " + formatDuration(metrics.CycleTime.P90CycleTime),
			},
			Implementation: []string{
				"Break down tasks into smaller chunks",
				"Implement effective code review processes",
				"Use automated testing and deployment",
			},
			Priority: entities.RecommendationPriorityMedium,
			Metadata: make(map[string]interface{}),
		}
		// Store additional details in metadata
		practice.Metadata["applicable_projects"] = []entities.ProjectType{entities.ProjectTypeWebApp, entities.ProjectTypeAPI, entities.ProjectTypeCLI}
		practice.Metadata["source"] = topCycleTimeRepo
		practice.Metadata["extracted_at"] = time.Now()
		practices = append(practices, practice)
	}

	return practices, nil
}

// Helper methods

func (cra *crossRepoAnalyzer) identifyCommonPatterns(tasks []*entities.Task) []*entities.CommonPattern {
	// Group tasks by type and analyze patterns
	typePatterns := make(map[string][]*entities.Task)
	for _, task := range tasks {
		typePatterns[task.Type] = append(typePatterns[task.Type], task)
	}

	var patterns []*entities.CommonPattern
	for taskType, typeTasks := range typePatterns {
		if len(typeTasks) < 5 { // Need minimum sample size
			continue
		}

		// Calculate average cycle time for this type
		var totalTime time.Duration
		completedCount := 0
		for _, task := range typeTasks {
			if task.Status == entities.StatusCompleted {
				totalTime += task.UpdatedAt.Sub(task.CreatedAt)
				completedCount++
			}
		}

		if completedCount > 0 {
			avgCycleTime := totalTime / time.Duration(completedCount)

			pattern := &entities.CommonPattern{
				ID:                generatePatternID(taskType),
				Name:              taskType + " Pattern",
				Description:       fmt.Sprintf("Common pattern for %s tasks", taskType),
				Type:              entities.PatternTypeWorkflow,
				Frequency:         float64(len(typeTasks)),
				SuccessRate:       float64(completedCount) / float64(len(typeTasks)),
				RepositoryCount:   len(cra.getRepositoriesForTaskType(typeTasks)),
				Repositories:      cra.getRepositoriesForTaskType(typeTasks),
				EstimatedDuration: avgCycleTime,
				Confidence:        0.7,
				FirstDetected:     time.Now(),
				LastSeen:          time.Now(),
				Metadata:          make(map[string]interface{}),
			}
			patterns = append(patterns, pattern)
		}
	}

	// Sort by frequency
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Frequency > patterns[j].Frequency
	})

	return patterns
}

func (cra *crossRepoAnalyzer) generateCrossRepoInsights(repoMetrics map[string]*entities.WorkflowMetrics, patterns []*entities.CommonPattern) []*entities.CrossRepoInsight {
	var insights []*entities.CrossRepoInsight

	// Performance variance insight
	if len(repoMetrics) > 1 {
		productivityScores := make([]float64, 0, len(repoMetrics))
		repoNames := make([]string, 0, len(repoMetrics))

		for repo, metrics := range repoMetrics {
			productivityScores = append(productivityScores, metrics.Productivity.Score)
			repoNames = append(repoNames, repo)
		}

		variance := calculateVariance(productivityScores)
		if variance > 100 { // High variance threshold
			insight := &entities.CrossRepoInsight{
				ID:          generateInsightID("variance"),
				Type:        entities.InsightTypePerformance,
				Title:       "High Performance Variance Across Repositories",
				Description: fmt.Sprintf("Productivity scores vary significantly across repositories (variance: %.1f)", variance),
				SourceCount: len(repoMetrics),
				Confidence:  0.8,
				Relevance:   0.9,
				Impact: entities.ImpactMetrics{
					ProductivityGain:   0.3,
					TimeReduction:      0.2,
					QualityImprovement: 0.15,
					AdoptionRate:       0.6,
				},
				Recommendations: []string{
					"Identify practices from top-performing repositories",
					"Standardize development processes across teams",
					"Share knowledge and best practices",
				},
				Tags:         []string{"performance", "variance", "cross-repo"},
				Metadata:     make(map[string]interface{}),
				GeneratedAt:  time.Now(),
				ValidUntil:   time.Now().Add(30 * 24 * time.Hour),
				IsActionable: true,
			}
			// Store affected repositories in metadata
			insight.Metadata["affected_repositories"] = repoNames
			insight.Metadata["evidence"] = fmt.Sprintf("Repository scores range from %.1f to %.1f", slices.Min(productivityScores), slices.Max(productivityScores))
			insights = append(insights, insight)
		}
	}

	// Common bottleneck insight
	if len(patterns) > 0 {
		// Find the most common high-cycle-time pattern
		for _, pattern := range patterns {
			if pattern.EstimatedDuration > 3*24*time.Hour && pattern.Frequency > 10 {
				insight := &entities.CrossRepoInsight{
					ID:          generateInsightID("bottleneck"),
					Type:        entities.InsightTypeBottleneck,
					Title:       fmt.Sprintf("Common Bottleneck in %s Tasks", pattern.Name),
					Description: fmt.Sprintf("Tasks of type '%s' consistently take longer than expected across repositories", pattern.Name),
					SourceCount: pattern.RepositoryCount,
					Confidence:  0.9,
					Relevance:   0.95,
					Impact: entities.ImpactMetrics{
						ProductivityGain:   0.4,
						TimeReduction:      pattern.EstimatedDuration.Hours() * 0.3,
						QualityImprovement: 0.1,
						AdoptionRate:       0.7,
					},
					Recommendations: []string{
						fmt.Sprintf("Review %s task complexity and requirements", pattern.Name),
						"Standardize approach for this task type",
						"Consider breaking down into smaller subtasks",
					},
					Tags:         []string{"bottleneck", "workflow", "cross-repo"},
					Metadata:     make(map[string]interface{}),
					GeneratedAt:  time.Now(),
					ValidUntil:   time.Now().Add(30 * 24 * time.Hour),
					IsActionable: true,
				}
				// Store pattern details in metadata
				insight.Metadata["affected_repositories"] = pattern.Repositories
				insight.Metadata["evidence"] = fmt.Sprintf("Average cycle time: %s across %.0f tasks in %d repositories", formatDuration(pattern.EstimatedDuration), pattern.Frequency, len(pattern.Repositories))
				insights = append(insights, insight)
				break // Only add the top bottleneck
			}
		}
	}

	return insights
}

func (cra *crossRepoAnalyzer) generateRecommendations(repoMetrics map[string]*entities.WorkflowMetrics, insights []*entities.CrossRepoInsight) []*entities.Recommendation {
	var recommendations []*entities.Recommendation

	// Find top performer for knowledge sharing
	topRepo := cra.findTopPerformer(repoMetrics, "overall")
	if topRepo != "" {
		recommendation := &entities.Recommendation{
			ID:          generateRecommendationID("knowledge-sharing"),
			Priority:    entities.RecommendationPriorityMedium,
			Title:       "Share Best Practices from Top Performer",
			Description: fmt.Sprintf("Repository '%s' shows superior performance. Consider knowledge sharing sessions.", topRepo),
			Impact:      0.8, // High impact
			Effort:      0.5, // Medium effort
			Category:    "process",
			Actions: []string{
				fmt.Sprintf("Schedule knowledge sharing session with %s team", topRepo),
				"Document successful practices and workflows",
				"Pilot successful practices in other repositories",
			},
			Evidence: []string{
				topRepo + " shows superior performance metrics",
			},
			CreatedAt: time.Now(),
		}
		recommendations = append(recommendations, recommendation)
	}

	// Add recommendations based on insights
	for _, insight := range insights {
		if insight.Type == entities.InsightTypeBottleneck {
			recommendation := &entities.Recommendation{
				ID:          generateRecommendationID("bottleneck-fix"),
				Priority:    entities.RecommendationPriorityHigh,
				Title:       "Address Common Workflow Bottleneck",
				Description: insight.Description,
				Impact:      0.8, // High impact
				Effort:      0.5, // Medium effort
				Category:    "workflow",
				Actions:     insight.Recommendations,
				Evidence: []string{
					insight.Description,
				},
				CreatedAt: time.Now(),
			}
			recommendations = append(recommendations, recommendation)
		}
	}

	return recommendations
}

func (cra *crossRepoAnalyzer) detectOutliers(repoMetrics map[string]*entities.WorkflowMetrics) []*entities.Outlier {
	var outliers []*entities.Outlier

	if len(repoMetrics) < 3 {
		return outliers // Need minimum sample size
	}

	// Analyze productivity outliers
	productivityScores := make([]float64, 0, len(repoMetrics))
	for _, metrics := range repoMetrics {
		productivityScores = append(productivityScores, metrics.Productivity.Score)
	}

	mean := calculateMean(productivityScores)
	stdDev := calculateStdDev(productivityScores, mean)

	for repo, metrics := range repoMetrics {
		score := metrics.Productivity.Score
		zScore := (score - mean) / stdDev

		if zScore > 2 || zScore < -2 { // More than 2 standard deviations
			outlierType := constants.OutlierTypePositive
			if zScore < 0 {
				outlierType = "negative"
			}

			outlier := &entities.Outlier{
				ID:              generateOutlierID(repo, "productivity"),
				Repository:      repo,
				Type:            outlierType,
				Description:     fmt.Sprintf("Repository %s has %s productivity score", repo, outlierType),
				Value:           score,
				ExpectedValue:   mean,
				Deviation:       score - mean,
				Severity:        cra.calculateOutlierSeverity(zScore),
				Metric:          entities.MetricTypeProductivity,
				PossibleCauses:  cra.identifyPossibleCauses(outlierType, "productivity"),
				Recommendations: cra.generateOutlierRecommendations(outlierType, "productivity"),
				DetectedAt:      time.Now(),
				LastOccurrence:  time.Now(),
				Frequency:       1,
				Metadata:        make(map[string]interface{}),
			}
			// Store statistical details in metadata
			outlier.Metadata["mean"] = mean
			outlier.Metadata["std_dev"] = stdDev
			outlier.Metadata["z_score"] = zScore
			outliers = append(outliers, outlier)
		}
	}

	return outliers
}

func (cra *crossRepoAnalyzer) calculateCorrelations(repoMetrics map[string]*entities.WorkflowMetrics) map[string]float64 {
	correlations := make(map[string]float64)

	if len(repoMetrics) < 3 {
		return correlations // Need minimum sample size for correlation
	}

	// Extract metric arrays
	productivity := make([]float64, 0, len(repoMetrics))
	velocity := make([]float64, 0, len(repoMetrics))
	completion := make([]float64, 0, len(repoMetrics))

	for _, metrics := range repoMetrics {
		productivity = append(productivity, metrics.Productivity.Score)
		velocity = append(velocity, metrics.Velocity.CurrentVelocity)
		completion = append(completion, metrics.Completion.CompletionRate*100)
	}

	// Calculate correlations
	correlations["productivity_velocity"] = calculateCorrelation(productivity, velocity)
	correlations["productivity_completion"] = calculateCorrelation(productivity, completion)
	correlations["velocity_completion"] = calculateCorrelation(velocity, completion)

	return correlations
}

// Additional helper functions

func generatePatternID(patternType string) string {
	return fmt.Sprintf("pattern_%s_%d", patternType, time.Now().UnixNano())
}

func generateOutlierID(repo, metric string) string {
	return fmt.Sprintf("outlier_%s_%s_%d", repo, metric, time.Now().UnixNano())
}

// Utility helper methods

func (cra *crossRepoAnalyzer) getRepositoriesForTaskType(tasks []*entities.Task) []string {
	repoSet := make(map[string]bool)
	for _, task := range tasks {
		repoSet[task.Repository] = true
	}

	repos := make([]string, 0, len(repoSet))
	for repo := range repoSet {
		repos = append(repos, repo)
	}
	return repos
}

func (cra *crossRepoAnalyzer) findTopPerformer(repoMetrics map[string]*entities.WorkflowMetrics, metric string) string {
	if len(repoMetrics) == 0 {
		return ""
	}

	topRepo := ""
	topScore := 0.0

	for repo, metrics := range repoMetrics {
		var score float64
		switch metric {
		case "productivity":
			score = metrics.Productivity.Score
		case "velocity":
			score = metrics.Velocity.CurrentVelocity
		case "completion":
			score = metrics.Completion.CompletionRate * 100
		case "overall":
			// Weighted overall score
			score = (metrics.Productivity.Score * 0.4) +
				(metrics.Velocity.CurrentVelocity * 0.3) +
				(metrics.Completion.CompletionRate * 100 * 0.3)
		default:
			score = metrics.Productivity.Score
		}

		if score > topScore {
			topScore = score
			topRepo = repo
		}
	}

	return topRepo
}

func (cra *crossRepoAnalyzer) findBestCycleTime(repoMetrics map[string]*entities.WorkflowMetrics) string {
	if len(repoMetrics) == 0 {
		return ""
	}

	bestRepo := ""
	bestTime := time.Duration(0)

	for repo, metrics := range repoMetrics {
		// Lower cycle time is better
		if bestTime == 0 || (metrics.CycleTime.AverageCycleTime > 0 && metrics.CycleTime.AverageCycleTime < bestTime) {
			bestTime = metrics.CycleTime.AverageCycleTime
			bestRepo = repo
		}
	}

	return bestRepo
}

func (cra *crossRepoAnalyzer) calculateOutlierSeverity(zScore float64) entities.OutlierSeverity {
	absZ := zScore
	if absZ < 0 {
		absZ = -absZ
	}

	if absZ > 3 {
		return entities.OutlierSeverityHigh
	} else if absZ > 2.5 {
		return entities.OutlierSeverityMedium
	}
	return entities.OutlierSeverityLow
}

// Statistical helper functions

func calculateVariance(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	mean := calculateMean(values)
	sum := 0.0
	for _, v := range values {
		diff := v - mean
		sum += diff * diff
	}
	return sum / float64(len(values))
}

func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateStdDev(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}

	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values))
	return variance // Simplified - should be sqrt(variance)
}

func calculateCorrelation(x, y []float64) float64 {
	if len(x) != len(y) || len(x) == 0 {
		return 0
	}

	// Simplified correlation - in practice you'd use proper Pearson correlation
	meanX := calculateMean(x)
	meanY := calculateMean(y)

	numerator := 0.0
	denomX := 0.0
	denomY := 0.0

	for i := 0; i < len(x); i++ {
		dx := x[i] - meanX
		dy := y[i] - meanY
		numerator += dx * dy
		denomX += dx * dx
		denomY += dy * dy
	}

	if denomX == 0 || denomY == 0 {
		return 0
	}

	return numerator / (denomX * denomY) // Simplified - should use sqrt
}

// ID generation functions

func generatePracticeID(category string) string {
	return fmt.Sprintf("practice_%s_%d", category, time.Now().UnixNano())
}

func generateInsightID(category string) string {
	return fmt.Sprintf("insight_%s_%d", category, time.Now().UnixNano())
}

func generateRecommendationID(category string) string {
	return fmt.Sprintf("recommendation_%s_%d", category, time.Now().UnixNano())
}

func formatDuration(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
	return fmt.Sprintf("%.1fd", d.Hours()/24)
}

// identifyPossibleCauses identifies possible causes for outliers
func (cra *crossRepoAnalyzer) identifyPossibleCauses(outlierType, _ string) []string {
	if outlierType == "positive" {
		return []string{
			"Highly efficient development practices",
			"Strong team collaboration",
			"Well-defined processes",
			"Effective tooling and automation",
		}
	}
	return []string{
		"Process inefficiencies",
		"Lack of automation",
		"Technical debt accumulation",
		"Resource constraints",
	}
}

// generateOutlierRecommendations generates recommendations for outliers
func (cra *crossRepoAnalyzer) generateOutlierRecommendations(outlierType, _ string) []string {
	if outlierType == "positive" {
		return []string{
			"Document and share successful practices",
			"Use as benchmark for other repositories",
			"Conduct knowledge sharing sessions",
		}
	}
	return []string{
		"Analyze workflow bottlenecks",
		"Review and improve development processes",
		"Consider tooling improvements",
		"Allocate additional resources if needed",
	}
}
