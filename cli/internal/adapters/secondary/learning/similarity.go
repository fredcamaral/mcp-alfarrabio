package learning

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/services"
)

// SimilarityAnalyzer interface for repository similarity calculations
type SimilarityAnalyzer interface {
	CalculateSimilarity(ctx context.Context, repoA, repoB *entities.RepositoryCharacteristics) (*entities.RepositorySimilarity, error)
	CalculateDimensionSimilarity(ctx context.Context, dimension string, repoA, repoB *entities.RepositoryCharacteristics) (float64, error)
	FindMostSimilarDimension(ctx context.Context, repoA, repoB *entities.RepositoryCharacteristics) (string, float64, error)
	BatchCalculateSimilarity(ctx context.Context, target *entities.RepositoryCharacteristics, candidates []*entities.RepositoryCharacteristics) ([]*entities.RepositorySimilarity, error)
}

// SimilarityWeights holds weights for different similarity dimensions
type SimilarityWeights struct {
	ProjectType  float64 `json:"project_type"`
	TechStack    float64 `json:"tech_stack"`
	TeamSize     float64 `json:"team_size"`
	Complexity   float64 `json:"complexity"`
	Domain       float64 `json:"domain"`
	WorkPatterns float64 `json:"work_patterns"`
}

// DefaultSimilarityWeights returns default weights for similarity calculation
func DefaultSimilarityWeights() SimilarityWeights {
	return SimilarityWeights{
		ProjectType:  0.25,
		TechStack:    0.20,
		TeamSize:     0.10,
		Complexity:   0.15,
		Domain:       0.15,
		WorkPatterns: 0.15,
	}
}

// SimilarityConfig holds configuration for similarity analysis
type SimilarityConfig struct {
	Weights                SimilarityWeights  `json:"weights"`
	MinSimilarityThreshold float64            `json:"min_similarity_threshold"`
	MaxCandidates          int                `json:"max_candidates"`
	CacheResults           bool               `json:"cache_results"`
	CacheTTL               time.Duration      `json:"cache_ttl"`
	LanguageWeights        map[string]float64 `json:"language_weights"`
	FrameworkWeights       map[string]float64 `json:"framework_weights"`
}

// DefaultSimilarityConfig returns default configuration
func DefaultSimilarityConfig() *SimilarityConfig {
	return &SimilarityConfig{
		Weights:                DefaultSimilarityWeights(),
		MinSimilarityThreshold: 0.3,
		MaxCandidates:          50,
		CacheResults:           true,
		CacheTTL:               2 * time.Hour,
		LanguageWeights: map[string]float64{
			"go":         1.0,
			"javascript": 0.9,
			"typescript": 0.9,
			"python":     0.8,
			"java":       0.8,
			"rust":       1.0,
			"c":          0.7,
			"cpp":        0.7,
		},
		FrameworkWeights: map[string]float64{
			"react":   1.0,
			"vue":     0.9,
			"angular": 0.8,
			"express": 0.9,
			"gin":     1.0,
			"echo":    0.9,
			"django":  0.8,
			"flask":   0.7,
			"spring":  0.8,
		},
	}
}

// similarityAnalyzerImpl implements the SimilarityAnalyzer interface
type similarityAnalyzerImpl struct {
	config *SimilarityConfig
	cache  services.Cache
	logger *slog.Logger
}

// NewSimilarityAnalyzer creates a new similarity analyzer
func NewSimilarityAnalyzer(config *SimilarityConfig, cache services.Cache, logger *slog.Logger) SimilarityAnalyzer {
	if config == nil {
		config = DefaultSimilarityConfig()
	}

	return &similarityAnalyzerImpl{
		config: config,
		cache:  cache,
		logger: logger,
	}
}

// CalculateSimilarity calculates overall similarity between two repositories
func (sa *similarityAnalyzerImpl) CalculateSimilarity(
	ctx context.Context,
	repoA, repoB *entities.RepositoryCharacteristics,
) (*entities.RepositorySimilarity, error) {
	sa.logger.Debug("calculating repository similarity",
		slog.String("repo_a_type", string(repoA.ProjectType)),
		slog.String("repo_b_type", string(repoB.ProjectType)))

	// Check cache if enabled
	if sa.config.CacheResults {
		cacheKey := sa.generateCacheKey(repoA, repoB)
		if cached, found := sa.cache.Get(cacheKey); found {
			if similarity, ok := cached.(*entities.RepositorySimilarity); ok {
				return similarity, nil
			}
		}
	}

	// Calculate individual dimension similarities
	dimensions := entities.SimilarityDimensions{}

	// Project type similarity
	dimensions.ProjectType = sa.calculateProjectTypeSimilarity(repoA.ProjectType, repoB.ProjectType)

	// Tech stack similarity
	dimensions.TechStack = sa.calculateTechStackSimilarity(repoA, repoB)

	// Team size similarity (if available)
	dimensions.TeamSize = sa.calculateTeamSizeSimilarity(repoA.TeamSize, repoB.TeamSize)

	// Complexity similarity
	dimensions.Complexity = sa.calculateComplexitySimilarity(repoA.Complexity, repoB.Complexity)

	// Domain similarity
	dimensions.Domain = sa.calculateDomainSimilarity(repoA.Domain, repoB.Domain)

	// Work patterns similarity
	dimensions.WorkPatterns = sa.calculateWorkPatternsSimilarity(repoA.WorkPatterns, repoB.WorkPatterns)

	// Calculate overall weighted score
	overallScore := sa.calculateWeightedScore(dimensions)

	similarity := &entities.RepositorySimilarity{
		RepositoryA:    "repo_a_" + string(repoA.ProjectType),
		RepositoryB:    "repo_b_" + string(repoB.ProjectType),
		Score:          overallScore,
		Dimensions:     dimensions,
		SharedPatterns: sa.findSharedPatterns(repoA, repoB),
		LastCalculated: time.Now(),
		Metadata:       make(map[string]interface{}),
	}

	// Add metadata
	similarity.Metadata["calculation_method"] = "weighted_dimensions"
	similarity.Metadata["weights"] = sa.config.Weights
	similarity.Metadata["language_overlap"] = sa.calculateLanguageOverlap(repoA.Languages, repoB.Languages)
	similarity.Metadata["framework_overlap"] = sa.calculateFrameworkOverlap(repoA.Frameworks, repoB.Frameworks)

	// Cache result if enabled
	if sa.config.CacheResults {
		cacheKey := sa.generateCacheKey(repoA, repoB)
		sa.cache.Set(cacheKey, similarity, sa.config.CacheTTL)
	}

	sa.logger.Debug("similarity calculated",
		slog.Float64("score", overallScore),
		slog.Float64("project_type", dimensions.ProjectType),
		slog.Float64("tech_stack", dimensions.TechStack))

	return similarity, nil
}

// CalculateDimensionSimilarity calculates similarity for a specific dimension
func (sa *similarityAnalyzerImpl) CalculateDimensionSimilarity(
	ctx context.Context,
	dimension string,
	repoA, repoB *entities.RepositoryCharacteristics,
) (float64, error) {
	switch strings.ToLower(dimension) {
	case "project_type":
		return sa.calculateProjectTypeSimilarity(repoA.ProjectType, repoB.ProjectType), nil
	case "tech_stack":
		return sa.calculateTechStackSimilarity(repoA, repoB), nil
	case "team_size":
		return sa.calculateTeamSizeSimilarity(repoA.TeamSize, repoB.TeamSize), nil
	case "complexity":
		return sa.calculateComplexitySimilarity(repoA.Complexity, repoB.Complexity), nil
	case "domain":
		return sa.calculateDomainSimilarity(repoA.Domain, repoB.Domain), nil
	case "work_patterns":
		return sa.calculateWorkPatternsSimilarity(repoA.WorkPatterns, repoB.WorkPatterns), nil
	default:
		return 0, fmt.Errorf("unknown dimension: %s", dimension)
	}
}

// FindMostSimilarDimension finds the dimension with highest similarity
func (sa *similarityAnalyzerImpl) FindMostSimilarDimension(
	ctx context.Context,
	repoA, repoB *entities.RepositoryCharacteristics,
) (string, float64, error) {
	dimensions := map[string]float64{
		"project_type":  sa.calculateProjectTypeSimilarity(repoA.ProjectType, repoB.ProjectType),
		"tech_stack":    sa.calculateTechStackSimilarity(repoA, repoB),
		"team_size":     sa.calculateTeamSizeSimilarity(repoA.TeamSize, repoB.TeamSize),
		"complexity":    sa.calculateComplexitySimilarity(repoA.Complexity, repoB.Complexity),
		"domain":        sa.calculateDomainSimilarity(repoA.Domain, repoB.Domain),
		"work_patterns": sa.calculateWorkPatternsSimilarity(repoA.WorkPatterns, repoB.WorkPatterns),
	}

	maxDimension := ""
	maxScore := 0.0

	for dimension, score := range dimensions {
		if score > maxScore {
			maxScore = score
			maxDimension = dimension
		}
	}

	return maxDimension, maxScore, nil
}

// BatchCalculateSimilarity calculates similarity between target and multiple candidates
func (sa *similarityAnalyzerImpl) BatchCalculateSimilarity(
	ctx context.Context,
	target *entities.RepositoryCharacteristics,
	candidates []*entities.RepositoryCharacteristics,
) ([]*entities.RepositorySimilarity, error) {
	sa.logger.Debug("batch calculating similarities",
		slog.Int("candidate_count", len(candidates)))

	var similarities []*entities.RepositorySimilarity

	// Limit candidates to avoid performance issues
	maxCandidates := sa.config.MaxCandidates
	if len(candidates) > maxCandidates {
		candidates = candidates[:maxCandidates]
	}

	for _, candidate := range candidates {
		similarity, err := sa.CalculateSimilarity(ctx, target, candidate)
		if err != nil {
			sa.logger.Warn("failed to calculate similarity",
				slog.Any("error", err))
			continue
		}

		// Only include similarities above threshold
		if similarity.Score >= sa.config.MinSimilarityThreshold {
			similarities = append(similarities, similarity)
		}
	}

	sa.logger.Debug("batch calculation completed",
		slog.Int("similarities_found", len(similarities)))

	return similarities, nil
}

// Helper methods for individual similarity calculations

func (sa *similarityAnalyzerImpl) calculateProjectTypeSimilarity(typeA, typeB entities.ProjectType) float64 {
	if typeA == typeB {
		return 1.0
	}

	// Define compatibility matrix for project types
	compatibility := map[entities.ProjectType]map[entities.ProjectType]float64{
		entities.ProjectTypeWebApp: {
			entities.ProjectTypeAPI:          0.7,
			entities.ProjectTypeMicroservice: 0.6,
			entities.ProjectTypeLibrary:      0.3,
		},
		entities.ProjectTypeAPI: {
			entities.ProjectTypeWebApp:       0.7,
			entities.ProjectTypeMicroservice: 0.8,
			entities.ProjectTypeCLI:          0.4,
		},
		entities.ProjectTypeMicroservice: {
			entities.ProjectTypeAPI:     0.8,
			entities.ProjectTypeWebApp:  0.6,
			entities.ProjectTypeLibrary: 0.5,
		},
		entities.ProjectTypeCLI: {
			entities.ProjectTypeLibrary: 0.6,
			entities.ProjectTypeAPI:     0.4,
		},
		entities.ProjectTypeLibrary: {
			entities.ProjectTypeCLI:          0.6,
			entities.ProjectTypeMicroservice: 0.5,
			entities.ProjectTypeWebApp:       0.3,
		},
	}

	if typeACompat, exists := compatibility[typeA]; exists {
		if score, exists := typeACompat[typeB]; exists {
			return score
		}
	}

	// Check reverse compatibility
	if typeBCompat, exists := compatibility[typeB]; exists {
		if score, exists := typeBCompat[typeA]; exists {
			return score
		}
	}

	return 0.0
}

func (sa *similarityAnalyzerImpl) calculateTechStackSimilarity(repoA, repoB *entities.RepositoryCharacteristics) float64 {
	// Calculate language similarity
	langSimilarity := sa.calculateLanguageSimilarity(repoA.Languages, repoB.Languages)

	// Calculate framework similarity
	frameworkSimilarity := sa.calculateFrameworkSimilarity(repoA.Frameworks, repoB.Frameworks)

	// Calculate dependency similarity
	depSimilarity := sa.calculateDependencySimilarity(repoA.Dependencies, repoB.Dependencies)

	// Weight the different aspects
	techStackScore := (langSimilarity * 0.4) + (frameworkSimilarity * 0.4) + (depSimilarity * 0.2)

	return techStackScore
}

func (sa *similarityAnalyzerImpl) calculateLanguageSimilarity(langsA, langsB map[string]int) float64 {
	if len(langsA) == 0 && len(langsB) == 0 {
		return 1.0
	}

	if len(langsA) == 0 || len(langsB) == 0 {
		return 0.0
	}

	// Calculate weighted Jaccard similarity
	totalA := sa.sumLanguageCounts(langsA)
	totalB := sa.sumLanguageCounts(langsB)

	var intersection, union float64

	allLanguages := make(map[string]bool)
	for lang := range langsA {
		allLanguages[lang] = true
	}
	for lang := range langsB {
		allLanguages[lang] = true
	}

	for lang := range allLanguages {
		countA := float64(langsA[lang]) / float64(totalA)
		countB := float64(langsB[lang]) / float64(totalB)

		// Apply language weights if available
		weight := 1.0
		if w, exists := sa.config.LanguageWeights[lang]; exists {
			weight = w
		}

		min := math.Min(countA, countB) * weight
		max := math.Max(countA, countB) * weight

		intersection += min
		union += max
	}

	if union == 0 {
		return 0.0
	}

	return intersection / union
}

func (sa *similarityAnalyzerImpl) calculateFrameworkSimilarity(frameworksA, frameworksB []string) float64 {
	if len(frameworksA) == 0 && len(frameworksB) == 0 {
		return 1.0
	}

	if len(frameworksA) == 0 || len(frameworksB) == 0 {
		return 0.0
	}

	setA := make(map[string]bool)
	for _, framework := range frameworksA {
		setA[strings.ToLower(framework)] = true
	}

	intersection := 0.0
	totalWeight := 0.0

	for _, framework := range frameworksB {
		weight := 1.0
		if w, exists := sa.config.FrameworkWeights[strings.ToLower(framework)]; exists {
			weight = w
		}

		totalWeight += weight

		if setA[strings.ToLower(framework)] {
			intersection += weight
		}
	}

	// Add weights for frameworks only in A
	for _, framework := range frameworksA {
		if !contains(frameworksB, framework) {
			weight := 1.0
			if w, exists := sa.config.FrameworkWeights[strings.ToLower(framework)]; exists {
				weight = w
			}
			totalWeight += weight
		}
	}

	if totalWeight == 0 {
		return 0.0
	}

	return intersection / totalWeight
}

func (sa *similarityAnalyzerImpl) calculateDependencySimilarity(depsA, depsB []string) float64 {
	if len(depsA) == 0 && len(depsB) == 0 {
		return 1.0
	}

	if len(depsA) == 0 || len(depsB) == 0 {
		return 0.0
	}

	setA := make(map[string]bool)
	for _, dep := range depsA {
		setA[strings.ToLower(dep)] = true
	}

	intersection := 0
	for _, dep := range depsB {
		if setA[strings.ToLower(dep)] {
			intersection++
		}
	}

	union := len(depsA) + len(depsB) - intersection
	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

func (sa *similarityAnalyzerImpl) calculateTeamSizeSimilarity(sizeA, sizeB int) float64 {
	if sizeA == 0 && sizeB == 0 {
		return 1.0
	}

	if sizeA == 0 || sizeB == 0 {
		return 0.5 // Partial similarity when one is unknown
	}

	// Use inverse of relative difference
	diff := math.Abs(float64(sizeA - sizeB))
	maxSize := math.Max(float64(sizeA), float64(sizeB))

	if maxSize == 0 {
		return 1.0
	}

	similarity := 1.0 - (diff / maxSize)
	return math.Max(0.0, similarity)
}

func (sa *similarityAnalyzerImpl) calculateComplexitySimilarity(complexityA, complexityB float64) float64 {
	diff := math.Abs(complexityA - complexityB)
	similarity := 1.0 - diff
	return math.Max(0.0, similarity)
}

func (sa *similarityAnalyzerImpl) calculateDomainSimilarity(domainA, domainB string) float64 {
	if domainA == "" && domainB == "" {
		return 1.0
	}

	if domainA == "" || domainB == "" {
		return 0.5
	}

	domainA = strings.ToLower(strings.TrimSpace(domainA))
	domainB = strings.ToLower(strings.TrimSpace(domainB))

	if domainA == domainB {
		return 1.0
	}

	// Check for substring matches
	if strings.Contains(domainA, domainB) || strings.Contains(domainB, domainA) {
		return 0.7
	}

	// Check for word overlaps
	wordsA := strings.Fields(domainA)
	wordsB := strings.Fields(domainB)

	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0.0
	}

	intersection := 0
	for _, wordA := range wordsA {
		for _, wordB := range wordsB {
			if wordA == wordB {
				intersection++
				break
			}
		}
	}

	union := len(wordsA) + len(wordsB) - intersection
	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

func (sa *similarityAnalyzerImpl) calculateWorkPatternsSimilarity(patternsA, patternsB []entities.WorkPattern) float64 {
	if len(patternsA) == 0 && len(patternsB) == 0 {
		return 1.0
	}

	if len(patternsA) == 0 || len(patternsB) == 0 {
		return 0.0
	}

	// Create maps for easier comparison
	mapA := make(map[string]entities.WorkPattern)
	for _, pattern := range patternsA {
		mapA[pattern.Type] = pattern
	}

	intersection := 0.0
	totalPatterns := 0.0

	for _, patternB := range patternsB {
		totalPatterns++
		if patternA, exists := mapA[patternB.Type]; exists {
			// Calculate similarity between the specific patterns
			similarity := sa.calculateIndividualPatternSimilarity(patternA, patternB)
			intersection += similarity
		}
	}

	// Account for patterns only in A
	for _, patternA := range patternsA {
		found := false
		for _, patternB := range patternsB {
			if patternA.Type == patternB.Type {
				found = true
				break
			}
		}
		if !found {
			totalPatterns++
		}
	}

	if totalPatterns == 0 {
		return 0.0
	}

	return intersection / totalPatterns
}

func (sa *similarityAnalyzerImpl) calculateIndividualPatternSimilarity(patternA, patternB entities.WorkPattern) float64 {
	// Compare frequency
	freqSimilarity := 1.0 - math.Abs(patternA.Frequency-patternB.Frequency)

	// Compare duration
	durSimilarity := 1.0 - math.Abs(patternA.Duration-patternB.Duration)/math.Max(patternA.Duration, patternB.Duration)

	// Compare success rate
	successSimilarity := 1.0 - math.Abs(patternA.SuccessRate-patternB.SuccessRate)

	// Weighted average
	return (freqSimilarity*0.4 + durSimilarity*0.3 + successSimilarity*0.3)
}

func (sa *similarityAnalyzerImpl) calculateWeightedScore(dimensions entities.SimilarityDimensions) float64 {
	score := 0.0
	score += dimensions.ProjectType * sa.config.Weights.ProjectType
	score += dimensions.TechStack * sa.config.Weights.TechStack
	score += dimensions.TeamSize * sa.config.Weights.TeamSize
	score += dimensions.Complexity * sa.config.Weights.Complexity
	score += dimensions.Domain * sa.config.Weights.Domain
	score += dimensions.WorkPatterns * sa.config.Weights.WorkPatterns

	return score
}

func (sa *similarityAnalyzerImpl) findSharedPatterns(repoA, repoB *entities.RepositoryCharacteristics) []string {
	var shared []string

	// Find shared work patterns
	mapA := make(map[string]bool)
	for _, pattern := range repoA.WorkPatterns {
		mapA[pattern.Type] = true
	}

	for _, pattern := range repoB.WorkPatterns {
		if mapA[pattern.Type] {
			shared = append(shared, pattern.Type)
		}
	}

	return shared
}

func (sa *similarityAnalyzerImpl) calculateLanguageOverlap(langsA, langsB map[string]int) float64 {
	return sa.calculateLanguageSimilarity(langsA, langsB)
}

func (sa *similarityAnalyzerImpl) calculateFrameworkOverlap(frameworksA, frameworksB []string) float64 {
	return sa.calculateFrameworkSimilarity(frameworksA, frameworksB)
}

func (sa *similarityAnalyzerImpl) generateCacheKey(repoA, repoB *entities.RepositoryCharacteristics) string {
	return fmt.Sprintf("similarity:%s:%s:%s:%s",
		string(repoA.ProjectType),
		string(repoB.ProjectType),
		repoA.AnalyzedAt.Format("2006-01-02"),
		repoB.AnalyzedAt.Format("2006-01-02"))
}

// Helper functions

func (sa *similarityAnalyzerImpl) sumLanguageCounts(languages map[string]int) int {
	total := 0
	for _, count := range languages {
		total += count
	}
	return total
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.ToLower(s) == strings.ToLower(item) {
			return true
		}
	}
	return false
}
