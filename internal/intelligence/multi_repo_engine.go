package intelligence

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"mcp-memory/pkg/types"
)

// RepositoryContext represents context about a specific repository
type RepositoryContext struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	URL            string         `json:"url"`
	Language       string         `json:"language"`
	Framework      string         `json:"framework"`
	Architecture   string         `json:"architecture"`
	TeamSize       int            `json:"team_size"`
	LastActivity   time.Time      `json:"last_activity"`
	TotalSessions  int            `json:"total_sessions"`
	SuccessRate    float64        `json:"success_rate"`
	CommonPatterns []string       `json:"common_patterns"`
	TechStack      []string       `json:"tech_stack"`
	Configuration  map[string]any `json:"configuration"`
	Metadata       map[string]any `json:"metadata"`
}

// CrossRepoPattern represents a pattern that spans multiple repositories
type CrossRepoPattern struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	Description  string             `json:"description"`
	Repositories []string           `json:"repositories"`
	Frequency    int                `json:"frequency"`
	SuccessRate  float64            `json:"success_rate"`
	Confidence   float64            `json:"confidence"`
	Keywords     []string           `json:"keywords"`
	TechStacks   []string           `json:"tech_stacks"`
	Frameworks   []string           `json:"frameworks"`
	PatternType  PatternType        `json:"pattern_type"`
	Examples     []CrossRepoExample `json:"examples"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
	LastUsed     time.Time          `json:"last_used"`
}

// CrossRepoExample represents an example of a cross-repository pattern
type CrossRepoExample struct {
	ID         string         `json:"id"`
	Repository string         `json:"repository"`
	ChunkIDs   []string       `json:"chunk_ids"`
	Context    map[string]any `json:"context"`
	Outcome    string         `json:"outcome"`
	Timestamp  time.Time      `json:"timestamp"`
}

// RepositoryRelation represents a relationship between repositories
type RepositoryRelation struct {
	ID           string    `json:"id"`
	FromRepo     string    `json:"from_repo"`
	ToRepo       string    `json:"to_repo"`
	RelationType string    `json:"relation_type"` // "similar", "dependent", "shared_tech", "team_overlap"
	Strength     float64   `json:"strength"`
	Evidence     []string  `json:"evidence"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TeamKnowledge represents knowledge about development teams
type TeamKnowledge struct {
	TeamID         string         `json:"team_id"`
	TeamName       string         `json:"team_name"`
	Repositories   []string       `json:"repositories"`
	Expertise      []string       `json:"expertise"`
	Preferences    map[string]any `json:"preferences"`
	CommonPatterns []string       `json:"common_patterns"`
	SuccessRate    float64        `json:"success_rate"`
	ActivityLevel  string         `json:"activity_level"`
	LastActive     time.Time      `json:"last_active"`
}

// MultiRepoQuery represents a query across multiple repositories
type MultiRepoQuery struct {
	Query          string        `json:"query"`
	Repositories   []string      `json:"repositories,omitempty"`
	TechStacks     []string      `json:"tech_stacks,omitempty"`
	Frameworks     []string      `json:"frameworks,omitempty"`
	PatternTypes   []PatternType `json:"pattern_types,omitempty"`
	TimeRange      *TimeRange    `json:"time_range,omitempty"`
	MinConfidence  float64       `json:"min_confidence"`
	MaxResults     int           `json:"max_results"`
	IncludeSimilar bool          `json:"include_similar"`
}

// TimeRange represents a time range for queries
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// MultiRepoResult represents a result from multi-repository search
type MultiRepoResult struct {
	Repository string                    `json:"repository"`
	Chunks     []types.ConversationChunk `json:"chunks"`
	Patterns   []CrossRepoPattern        `json:"patterns"`
	Relevance  float64                   `json:"relevance"`
	Context    map[string]any            `json:"context"`
}

// MultiRepoEngine manages intelligence across multiple repositories
type MultiRepoEngine struct {
	patternEngine  *PatternEngine
	knowledgeGraph *GraphBuilder
	learningEngine *LearningEngine

	// Repository data
	repositories      map[string]*RepositoryContext
	crossRepoPatterns map[string]*CrossRepoPattern
	repoRelations     map[string]*RepositoryRelation
	teamKnowledge     map[string]*TeamKnowledge

	// Configuration
	maxRepositories     int
	similarityThreshold float64
	patternMinFreq      int
	enableTeamLearning  bool

	// State
	lastAnalysis     time.Time
	analysisInterval time.Duration
}

// MultiRepoStorage interface for persisting multi-repository data
type MultiRepoStorage interface {
	StoreRepositoryContext(ctx context.Context, repo *RepositoryContext) error
	GetRepositoryContext(ctx context.Context, id string) (*RepositoryContext, error)
	ListRepositories(ctx context.Context) ([]*RepositoryContext, error)

	StoreCrossRepoPattern(ctx context.Context, pattern *CrossRepoPattern) error
	GetCrossRepoPattern(ctx context.Context, id string) (*CrossRepoPattern, error)
	ListCrossRepoPatterns(ctx context.Context) ([]*CrossRepoPattern, error)

	StoreRepositoryRelation(ctx context.Context, relation *RepositoryRelation) error
	ListRepositoryRelations(ctx context.Context, repoID string) ([]*RepositoryRelation, error)

	StoreTeamKnowledge(ctx context.Context, team *TeamKnowledge) error
	GetTeamKnowledge(ctx context.Context, teamID string) (*TeamKnowledge, error)
	ListTeams(ctx context.Context) ([]*TeamKnowledge, error)
}

// NewMultiRepoEngine creates a new multi-repository intelligence engine
func NewMultiRepoEngine(patternEngine *PatternEngine, knowledgeGraph *GraphBuilder, learningEngine *LearningEngine) *MultiRepoEngine {
	return &MultiRepoEngine{
		patternEngine:       patternEngine,
		knowledgeGraph:      knowledgeGraph,
		learningEngine:      learningEngine,
		repositories:        make(map[string]*RepositoryContext),
		crossRepoPatterns:   make(map[string]*CrossRepoPattern),
		repoRelations:       make(map[string]*RepositoryRelation),
		teamKnowledge:       make(map[string]*TeamKnowledge),
		maxRepositories:     getEnvInt("MCP_MEMORY_MAX_REPOSITORIES", 100),
		similarityThreshold: getEnvFloat("MCP_MEMORY_REPO_SIMILARITY_THRESHOLD", 0.6),
		patternMinFreq:      getEnvInt("MCP_MEMORY_PATTERN_MIN_FREQUENCY", 3),
		enableTeamLearning:  getEnvBool("MCP_MEMORY_ENABLE_TEAM_LEARNING", true),
		lastAnalysis:        time.Now(),
		analysisInterval:    getEnvDurationHours("MCP_MEMORY_ANALYSIS_INTERVAL_HOURS", 24),
	}
}

// AddRepository adds a new repository to the intelligence system
func (mre *MultiRepoEngine) AddRepository(ctx context.Context, repo *RepositoryContext) error {
	if repo.ID == "" {
		return errors.New("repository ID cannot be empty")
	}

	if len(mre.repositories) >= mre.maxRepositories {
		return errors.New("maximum number of repositories reached")
	}

	repo.LastActivity = time.Now()
	mre.repositories[repo.ID] = repo

	// Analyze relationships with existing repositories
	mre.analyzeRepositoryRelationships(ctx, repo)

	return nil
}

// UpdateRepositoryContext updates context for an existing repository
func (mre *MultiRepoEngine) UpdateRepositoryContext(ctx context.Context, repoID string, chunks []types.ConversationChunk) error {
	repo, exists := mre.repositories[repoID]
	if !exists {
		// Create new repository context
		repo = &RepositoryContext{
			ID:             repoID,
			Name:           repoID,
			LastActivity:   time.Now(),
			TotalSessions:  0,
			SuccessRate:    0.0,
			CommonPatterns: []string{},
			TechStack:      []string{},
			Configuration:  make(map[string]any),
			Metadata:       make(map[string]any),
		}
		mre.repositories[repoID] = repo
	}

	// Update repository context based on conversation chunks
	repo.TotalSessions++
	repo.LastActivity = time.Now()

	// Extract tech stack and patterns
	techStack := mre.extractTechStack(chunks)
	patterns := mre.extractPatterns(chunks)

	repo.TechStack = mre.mergeTechStack(repo.TechStack, techStack)
	repo.CommonPatterns = mre.mergePatterns(repo.CommonPatterns, patterns)

	// Update success rate
	successRate := mre.calculateSuccessRate(chunks)
	repo.SuccessRate = (repo.SuccessRate*float64(repo.TotalSessions-1) + successRate) / float64(repo.TotalSessions)

	return nil
}

// AnalyzeCrossRepoPatterns identifies patterns that span multiple repositories
func (mre *MultiRepoEngine) AnalyzeCrossRepoPatterns(ctx context.Context) error {
	if time.Since(mre.lastAnalysis) < mre.analysisInterval {
		return nil // Skip if analyzed recently
	}

	// Collect all patterns from all repositories
	allPatterns := make(map[string][]PatternInfo)

	for repoID, repo := range mre.repositories {
		for _, pattern := range repo.CommonPatterns {
			if _, exists := allPatterns[pattern]; !exists {
				allPatterns[pattern] = make([]PatternInfo, 0)
			}

			allPatterns[pattern] = append(allPatterns[pattern], PatternInfo{
				Repository: repoID,
				TechStack:  repo.TechStack,
				Framework:  repo.Framework,
				Frequency:  1, // Simplified for this example
			})
		}
	}

	// Identify cross-repository patterns
	for patternName, repos := range allPatterns {
		if len(repos) >= 2 { // Pattern appears in multiple repos
			crossPattern := &CrossRepoPattern{
				ID:           fmt.Sprintf("cross_%s_%d", sanitizeID(patternName), time.Now().Unix()),
				Name:         patternName,
				Description:  fmt.Sprintf("Cross-repository pattern: %s", patternName),
				Repositories: make([]string, 0),
				Frequency:    len(repos),
				Confidence:   mre.calculateCrossPatternConfidence(repos),
				Keywords:     []string{patternName},
				TechStacks:   make([]string, 0),
				Frameworks:   make([]string, 0),
				PatternType:  PatternTypeWorkflow,
				Examples:     make([]CrossRepoExample, 0),
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}

			// Collect unique repositories, tech stacks, and frameworks
			seenRepos := make(map[string]bool)
			seenTechStacks := make(map[string]bool)
			seenFrameworks := make(map[string]bool)

			for _, info := range repos {
				if !seenRepos[info.Repository] {
					crossPattern.Repositories = append(crossPattern.Repositories, info.Repository)
					seenRepos[info.Repository] = true
				}

				for _, tech := range info.TechStack {
					if !seenTechStacks[tech] {
						crossPattern.TechStacks = append(crossPattern.TechStacks, tech)
						seenTechStacks[tech] = true
					}
				}

				if info.Framework != "" && !seenFrameworks[info.Framework] {
					crossPattern.Frameworks = append(crossPattern.Frameworks, info.Framework)
					seenFrameworks[info.Framework] = true
				}
			}

			// Calculate success rate across repositories
			totalSuccess := 0.0
			for _, repoID := range crossPattern.Repositories {
				if repo, exists := mre.repositories[repoID]; exists {
					totalSuccess += repo.SuccessRate
				}
			}
			crossPattern.SuccessRate = totalSuccess / float64(len(crossPattern.Repositories))

			mre.crossRepoPatterns[crossPattern.ID] = crossPattern
		}
	}

	mre.lastAnalysis = time.Now()
	return nil
}

// QueryMultiRepo performs queries across multiple repositories
func (mre *MultiRepoEngine) QueryMultiRepo(ctx context.Context, query MultiRepoQuery) ([]MultiRepoResult, error) {
	var results []MultiRepoResult

	// Filter repositories based on query criteria
	candidateRepos := mre.filterRepositories(query)

	// Search for patterns in each repository
	for _, repo := range candidateRepos {
		result := MultiRepoResult{
			Repository: repo.ID,
			Chunks:     []types.ConversationChunk{}, // Would be populated from actual storage
			Patterns:   []CrossRepoPattern{},
			Relevance:  mre.calculateRepositoryRelevance(repo, query),
			Context: map[string]any{
				"tech_stack":    repo.TechStack,
				"framework":     repo.Framework,
				"success_rate":  repo.SuccessRate,
				"last_activity": repo.LastActivity,
			},
		}

		// Find relevant cross-repo patterns
		for _, pattern := range mre.crossRepoPatterns {
			if mre.patternMatchesQuery(pattern, query) {
				for _, patternRepo := range pattern.Repositories {
					if patternRepo == repo.ID {
						result.Patterns = append(result.Patterns, *pattern)
						break
					}
				}
			}
		}

		if result.Relevance >= query.MinConfidence {
			results = append(results, result)
		}
	}

	// Sort by relevance
	sort.Slice(results, func(i, j int) bool {
		return results[i].Relevance > results[j].Relevance
	})

	// Apply limit
	if query.MaxResults > 0 && len(results) > query.MaxResults {
		results = results[:query.MaxResults]
	}

	return results, nil
}

// GetSimilarRepositories finds repositories similar to a given one
func (mre *MultiRepoEngine) GetSimilarRepositories(ctx context.Context, repoID string, limit int) ([]*RepositoryContext, error) {
	targetRepo, exists := mre.repositories[repoID]
	if !exists {
		return nil, fmt.Errorf("repository %s not found", repoID)
	}

	type repoSimilarity struct {
		repo       *RepositoryContext
		similarity float64
	}

	var similarities []repoSimilarity

	for id, repo := range mre.repositories {
		if id == repoID {
			continue
		}

		similarity := mre.calculateRepositorySimilarity(targetRepo, repo)
		if similarity >= mre.similarityThreshold {
			similarities = append(similarities, repoSimilarity{
				repo:       repo,
				similarity: similarity,
			})
		}
	}

	// Sort by similarity
	sort.Slice(similarities, func(i, j int) bool {
		return similarities[i].similarity > similarities[j].similarity
	})

	// Convert to result slice
	resultLen := len(similarities)
	if limit > 0 && limit < resultLen {
		resultLen = limit
	}
	result := make([]*RepositoryContext, 0, resultLen)
	for i, sim := range similarities {
		if limit > 0 && i >= limit {
			break
		}
		result = append(result, sim.repo)
	}

	return result, nil
}

// GetCrossRepoInsights provides insights across repositories
func (mre *MultiRepoEngine) GetCrossRepoInsights(ctx context.Context) (map[string]any, error) {
	insights := make(map[string]any)

	// Repository statistics
	insights["total_repositories"] = len(mre.repositories)
	insights["cross_repo_patterns"] = len(mre.crossRepoPatterns)
	insights["repository_relations"] = len(mre.repoRelations)

	// Tech stack distribution
	techStackCount := make(map[string]int)
	frameworkCount := make(map[string]int)
	languageCount := make(map[string]int)

	for _, repo := range mre.repositories {
		for _, tech := range repo.TechStack {
			techStackCount[tech]++
		}
		if repo.Framework != "" {
			frameworkCount[repo.Framework]++
		}
		if repo.Language != "" {
			languageCount[repo.Language]++
		}
	}

	insights["tech_stack_distribution"] = techStackCount
	insights["framework_distribution"] = frameworkCount
	insights["language_distribution"] = languageCount

	// Success rate analytics
	totalSuccessRate := 0.0
	activeRepos := 0
	for _, repo := range mre.repositories {
		if repo.TotalSessions > 0 {
			totalSuccessRate += repo.SuccessRate
			activeRepos++
		}
	}

	if activeRepos > 0 {
		insights["avg_success_rate"] = totalSuccessRate / float64(activeRepos)
	}

	// Most common patterns
	patternFreq := make(map[string]int)
	for _, pattern := range mre.crossRepoPatterns {
		patternFreq[pattern.Name] = pattern.Frequency
	}
	insights["common_patterns"] = patternFreq

	return insights, nil
}

// Helper types and methods

type PatternInfo struct {
	Repository string
	TechStack  []string
	Framework  string
	Frequency  int
}

func (mre *MultiRepoEngine) analyzeRepositoryRelationships(_ context.Context, newRepo *RepositoryContext) {
	for id, existingRepo := range mre.repositories {
		if id == newRepo.ID {
			continue
		}

		similarity := mre.calculateRepositorySimilarity(newRepo, existingRepo)
		if similarity >= mre.similarityThreshold {
			relation := &RepositoryRelation{
				ID:           fmt.Sprintf("rel_%s_%s", newRepo.ID, id),
				FromRepo:     newRepo.ID,
				ToRepo:       id,
				RelationType: "similar",
				Strength:     similarity,
				Evidence:     mre.generateSimilarityEvidence(newRepo, existingRepo),
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}

			mre.repoRelations[relation.ID] = relation
		}
	}
}

func (mre *MultiRepoEngine) extractTechStack(chunks []types.ConversationChunk) []string {
	techStack := make(map[string]bool)

	// Common technologies to look for
	technologies := []string{
		"go", "golang", "python", "javascript", "typescript", "java", "rust", "cpp", "c++",
		"react", "vue", "angular", "express", "flask", "django", "spring", "gin",
		"postgresql", "mysql", "mongodb", "redis", "elasticsearch",
		"docker", "kubernetes", "aws", "azure", "gcp",
		"git", "github", "gitlab", "ci/cd", "jenkins",
	}

	for _, chunk := range chunks {
		content := strings.ToLower(chunk.Content)
		for _, tech := range technologies {
			if strings.Contains(content, tech) {
				techStack[tech] = true
			}
		}
	}

	result := make([]string, 0, len(techStack))
	for tech := range techStack {
		result = append(result, tech)
	}

	return result
}

func (mre *MultiRepoEngine) extractPatterns(chunks []types.ConversationChunk) []string {
	patterns := make(map[string]bool)

	// Common development patterns
	commonPatterns := []string{
		"debugging", "testing", "deployment", "configuration", "refactoring",
		"api_design", "database_design", "error_handling", "performance_optimization",
		"security", "authentication", "authorization", "logging", "monitoring",
	}

	for _, chunk := range chunks {
		content := strings.ToLower(chunk.Content)
		for _, pattern := range commonPatterns {
			if strings.Contains(content, strings.ReplaceAll(pattern, "_", " ")) {
				patterns[pattern] = true
			}
		}
	}

	result := make([]string, 0, len(patterns))
	for pattern := range patterns {
		result = append(result, pattern)
	}

	return result
}

func (mre *MultiRepoEngine) mergeTechStack(existing, newItems []string) []string {
	merged := make(map[string]bool)

	for _, tech := range existing {
		merged[tech] = true
	}
	for _, tech := range newItems {
		merged[tech] = true
	}

	result := make([]string, 0, len(merged))
	for tech := range merged {
		result = append(result, tech)
	}

	return result
}

func (mre *MultiRepoEngine) mergePatterns(existing, newItems []string) []string {
	merged := make(map[string]bool)

	for _, pattern := range existing {
		merged[pattern] = true
	}
	for _, pattern := range newItems {
		merged[pattern] = true
	}

	result := make([]string, 0, len(merged))
	for pattern := range merged {
		result = append(result, pattern)
	}

	return result
}

func (mre *MultiRepoEngine) calculateSuccessRate(chunks []types.ConversationChunk) float64 {
	if len(chunks) == 0 {
		return 0.0
	}

	successCount := 0
	for _, chunk := range chunks {
		// Simple heuristic: look for success indicators
		content := strings.ToLower(chunk.Content)
		if strings.Contains(content, "success") || strings.Contains(content, "works") ||
			strings.Contains(content, "fixed") || strings.Contains(content, "resolved") {
			successCount++
		}
	}

	return float64(successCount) / float64(len(chunks))
}

func (mre *MultiRepoEngine) calculateCrossPatternConfidence(repos []PatternInfo) float64 {
	if len(repos) == 0 {
		return 0.0
	}

	// Base confidence on number of repositories
	patternDivisor := getEnvFloat("MCP_MEMORY_PATTERN_CONFIDENCE_DIVISOR", 10.0)
	baseConfidence := float64(len(repos)) / patternDivisor // Max confidence when pattern appears in divisor+ repos
	if baseConfidence > 1.0 {
		baseConfidence = 1.0
	}

	return baseConfidence
}

func (mre *MultiRepoEngine) filterRepositories(query MultiRepoQuery) []*RepositoryContext {
	var candidates []*RepositoryContext

	for _, repo := range mre.repositories {
		matches := true

		// Filter by specified repositories
		if len(query.Repositories) > 0 {
			found := false
			for _, repoID := range query.Repositories {
				if repo.ID == repoID {
					found = true
					break
				}
			}
			if !found {
				matches = false
			}
		}

		// Filter by tech stack
		if len(query.TechStacks) > 0 && matches {
			found := false
			for _, queryTech := range query.TechStacks {
				for _, repoTech := range repo.TechStack {
					if strings.EqualFold(queryTech, repoTech) {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				matches = false
			}
		}

		// Filter by framework
		if len(query.Frameworks) > 0 && matches {
			found := false
			for _, framework := range query.Frameworks {
				if strings.EqualFold(framework, repo.Framework) {
					found = true
					break
				}
			}
			if !found {
				matches = false
			}
		}

		// Filter by time range
		if query.TimeRange != nil && matches {
			if repo.LastActivity.Before(query.TimeRange.Start) || repo.LastActivity.After(query.TimeRange.End) {
				matches = false
			}
		}

		if matches {
			candidates = append(candidates, repo)
		}
	}

	return candidates
}

func (mre *MultiRepoEngine) calculateRepositoryRelevance(repo *RepositoryContext, query MultiRepoQuery) float64 {
	relevance := 0.0

	// Base relevance on success rate
	relevance += repo.SuccessRate * 0.3

	// Add relevance for matching tech stack
	if len(query.TechStacks) > 0 {
		matches := 0
		for _, queryTech := range query.TechStacks {
			for _, repoTech := range repo.TechStack {
				if strings.EqualFold(queryTech, repoTech) {
					matches++
					break
				}
			}
		}
		relevance += (float64(matches) / float64(len(query.TechStacks))) * 0.4
	}

	// Add relevance for recent activity
	daysSinceActivity := time.Since(repo.LastActivity).Hours() / 24
	decayDays := float64(getEnvInt("MCP_MEMORY_ACTIVITY_DECAY_DAYS", 30))
	activityRelevance := 1.0 / (1.0 + daysSinceActivity/decayDays) // Decay over configured days
	relevance += activityRelevance * 0.3

	return relevance
}

func (mre *MultiRepoEngine) patternMatchesQuery(pattern *CrossRepoPattern, query MultiRepoQuery) bool {
	// Check pattern types
	if len(query.PatternTypes) > 0 {
		found := false
		for _, queryType := range query.PatternTypes {
			if pattern.PatternType == queryType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check confidence
	if pattern.Confidence < query.MinConfidence {
		return false
	}

	// Check tech stacks
	if len(query.TechStacks) > 0 {
		found := false
		for _, queryTech := range query.TechStacks {
			for _, patternTech := range pattern.TechStacks {
				if strings.EqualFold(queryTech, patternTech) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func (mre *MultiRepoEngine) calculateRepositorySimilarity(repo1, repo2 *RepositoryContext) float64 {
	similarity := 0.0

	// Tech stack similarity
	techSimilarity := mre.calculateArraySimilarity(repo1.TechStack, repo2.TechStack)
	similarity += techSimilarity * 0.4

	// Framework similarity
	if repo1.Framework != "" && repo2.Framework != "" {
		if strings.EqualFold(repo1.Framework, repo2.Framework) {
			similarity += 0.2
		}
	}

	// Language similarity
	if repo1.Language != "" && repo2.Language != "" {
		if strings.EqualFold(repo1.Language, repo2.Language) {
			similarity += 0.2
		}
	}

	// Pattern similarity
	patternSimilarity := mre.calculateArraySimilarity(repo1.CommonPatterns, repo2.CommonPatterns)
	similarity += patternSimilarity * 0.2

	return similarity
}

func (mre *MultiRepoEngine) calculateArraySimilarity(arr1, arr2 []string) float64 {
	if len(arr1) == 0 || len(arr2) == 0 {
		return 0.0
	}

	intersection := 0
	for _, item1 := range arr1 {
		for _, item2 := range arr2 {
			if strings.EqualFold(item1, item2) {
				intersection++
				break
			}
		}
	}

	union := len(arr1) + len(arr2) - intersection
	return float64(intersection) / float64(union)
}

func (mre *MultiRepoEngine) generateSimilarityEvidence(repo1, repo2 *RepositoryContext) []string {
	var evidence []string

	// Common tech stack
	for _, tech1 := range repo1.TechStack {
		for _, tech2 := range repo2.TechStack {
			if strings.EqualFold(tech1, tech2) {
				evidence = append(evidence, fmt.Sprintf("shared_tech:%s", tech1))
			}
		}
	}

	// Same framework
	if repo1.Framework != "" && strings.EqualFold(repo1.Framework, repo2.Framework) {
		evidence = append(evidence, fmt.Sprintf("same_framework:%s", repo1.Framework))
	}

	// Same language
	if repo1.Language != "" && strings.EqualFold(repo1.Language, repo2.Language) {
		evidence = append(evidence, fmt.Sprintf("same_language:%s", repo1.Language))
	}

	return evidence
}

func sanitizeID(input string) string {
	// Simple ID sanitization
	result := strings.ToLower(input)
	result = strings.ReplaceAll(result, " ", "_")
	result = strings.ReplaceAll(result, "-", "_")
	return result
}

// Helper functions for environment variables
func getEnvInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if val := os.Getenv(key); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if val := os.Getenv(key); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return defaultValue
}

func getEnvDurationHours(key string, defaultHours int) time.Duration {
	if val := os.Getenv(key); val != "" {
		if hours, err := strconv.Atoi(val); err == nil {
			return time.Duration(hours) * time.Hour
		}
	}
	return time.Duration(defaultHours) * time.Hour
}
