// Package intelligence provides AI-powered memory analysis and pattern recognition.
// It includes learning engines, pattern matching, conflict detection, and cross-repository insights.
package intelligence

import (
	"context"
	"fmt"
	"lerian-mcp-memory/pkg/types"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// FreshnessManager tracks and manages memory freshness and staleness
type FreshnessManager struct {
	storage StorageInterface
	config  *FreshnessConfig
}

// FreshnessConfig configures freshness tracking behavior
type FreshnessConfig struct {
	// Technology-specific decay rates (per day)
	TechnologyDecayRates map[string]float64 `json:"technology_decay_rates"`

	// Content type decay rates
	ContentTypeDecayRates map[types.ChunkType]float64 `json:"content_type_decay_rates"`

	// Freshness thresholds (in days)
	FreshnessThresholds FreshnessThresholds `json:"freshness_thresholds"`

	// Auto-refresh settings
	AutoRefreshEnabled   bool          `json:"auto_refresh_enabled"`
	RefreshCheckInterval time.Duration `json:"refresh_check_interval"`

	// Staleness alerts
	StalenessAlerts StalenessAlerts `json:"staleness_alerts"`
}

// FreshnessThresholds defines when content becomes stale
type FreshnessThresholds struct {
	ArchitectureDecisions int            `json:"architecture_decisions"` // months
	BugFixes              int            `json:"bug_fixes"`              // months
	Documentation         int            `json:"documentation"`          // months
	GeneralContent        int            `json:"general_content"`        // months
	TechnologySpecific    map[string]int `json:"technology_specific"`    // tech -> months
}

// StalenessAlerts configures when to alert about stale content
type StalenessAlerts struct {
	TechnologyVersions    bool `json:"technology_versions"`
	ArchitectureDecisions bool `json:"architecture_decisions"`
	SecurityContent       bool `json:"security_content"`
	PerformanceMetrics    bool `json:"performance_metrics"`
}

// FreshnessStatus represents the freshness status of a memory
type FreshnessStatus struct {
	IsFresh          bool              `json:"is_fresh"`
	IsStale          bool              `json:"is_stale"`
	FreshnessScore   float64           `json:"freshness_score"` // 0.0-1.0
	DaysOld          int               `json:"days_old"`
	DecayRate        float64           `json:"decay_rate"`
	Alerts           []FreshnessAlert  `json:"alerts,omitempty"`
	LastChecked      time.Time         `json:"last_checked"`
	SuggestedActions []SuggestedAction `json:"suggested_actions,omitempty"`
}

// FreshnessAlert represents an alert about stale content
type FreshnessAlert struct {
	Type         string    `json:"type"`
	Severity     string    `json:"severity"` // "low", "medium", "high", "critical"
	Message      string    `json:"message"`
	Reason       string    `json:"reason"`
	Detected     time.Time `json:"detected"`
	ActionNeeded string    `json:"action_needed"`
}

// SuggestedAction represents a suggested action for stale content
type SuggestedAction struct {
	Action     string  `json:"action"`   // "refresh", "archive", "update", "verify"
	Priority   string  `json:"priority"` // "low", "medium", "high"
	Reason     string  `json:"reason"`
	Confidence float64 `json:"confidence"` // 0.0-1.0
}

// FreshnessBatch represents a batch freshness check result
type FreshnessBatch struct {
	TotalChecked    int                    `json:"total_checked"`
	FreshCount      int                    `json:"fresh_count"`
	StaleCount      int                    `json:"stale_count"`
	AlertsGenerated int                    `json:"alerts_generated"`
	ProcessingTime  time.Duration          `json:"processing_time"`
	Results         []ChunkFreshnessResult `json:"results"`
	Summary         FreshnessSummary       `json:"summary"`
}

// ChunkFreshnessResult represents freshness check result for a single chunk
type ChunkFreshnessResult struct {
	ChunkID         string          `json:"chunk_id"`
	Type            types.ChunkType `json:"type"`
	Repository      string          `json:"repository"`
	FreshnessStatus FreshnessStatus `json:"freshness_status"`
}

// FreshnessSummary provides overall freshness statistics
type FreshnessSummary struct {
	ByRepository  map[string]FreshnessStats `json:"by_repository"`
	ByType        map[string]FreshnessStats `json:"by_type"`
	ByTechnology  map[string]FreshnessStats `json:"by_technology"`
	OverallHealth string                    `json:"overall_health"` // "excellent", "good", "poor", "critical"
}

// FreshnessStats represents statistics for a category
type FreshnessStats struct {
	TotalChunks  int     `json:"total_chunks"`
	FreshChunks  int     `json:"fresh_chunks"`
	StaleChunks  int     `json:"stale_chunks"`
	AvgFreshness float64 `json:"avg_freshness"`
	AlertCount   int     `json:"alert_count"`
}

// NewFreshnessManager creates a new freshness manager
func NewFreshnessManager(storage StorageInterface) *FreshnessManager {
	return &FreshnessManager{
		storage: storage,
		config:  DefaultFreshnessConfig(),
	}
}

// DefaultFreshnessConfig returns sensible defaults
func DefaultFreshnessConfig() *FreshnessConfig {
	return &FreshnessConfig{
		TechnologyDecayRates: map[string]float64{
			"nodejs":     0.003, // 0.3% per day (fast-moving ecosystem)
			"react":      0.003,
			"javascript": 0.003,
			"python":     0.002, // 0.2% per day (moderate pace)
			"django":     0.002,
			"golang":     0.001, // 0.1% per day (stable)
			"kubernetes": 0.002,
			"docker":     0.002,
			"aws":        0.002,
			"default":    0.001, // Default for unrecognized tech
		},
		ContentTypeDecayRates: map[types.ChunkType]float64{
			types.ChunkTypeArchitectureDecision: 0.0005, // Very slow decay
			types.ChunkTypeProblem:              0.002,  // Moderate decay
			types.ChunkTypeSolution:             0.001,  // Slow decay (solutions are valuable)
			types.ChunkTypeCodeChange:           0.003,  // Fast decay (code changes quickly)
			types.ChunkTypeAnalysis:             0.002,  // Moderate decay
		},
		FreshnessThresholds: FreshnessThresholds{
			ArchitectureDecisions: 12, // 12 months
			BugFixes:              6,  // 6 months
			Documentation:         9,  // 9 months
			GeneralContent:        6,  // 6 months
			TechnologySpecific: map[string]int{
				"nodejs":     3, // 3 months for Node.js content
				"react":      3,
				"javascript": 4,
				"python":     6,
				"golang":     12,
				"docker":     6,
				"kubernetes": 4,
			},
		},
		AutoRefreshEnabled:   true,
		RefreshCheckInterval: 24 * time.Hour, // Daily checks
		StalenessAlerts: StalenessAlerts{
			TechnologyVersions:    true,
			ArchitectureDecisions: true,
			SecurityContent:       true,
			PerformanceMetrics:    true,
		},
	}
}

// CheckFreshness checks the freshness of a single memory chunk
func (fm *FreshnessManager) CheckFreshness(ctx context.Context, chunk *types.ConversationChunk) (*FreshnessStatus, error) {
	now := time.Now()
	age := now.Sub(chunk.Timestamp)
	daysOld := int(age.Hours() / 24)

	// Determine decay rate
	decayRate := fm.determineDecayRate(chunk)

	// Calculate freshness score
	freshnessScore := fm.calculateFreshnessScore(daysOld, decayRate)

	// Determine freshness and staleness
	isFresh := fm.isFresh(chunk, daysOld)
	isStale := fm.isStale(chunk, daysOld)

	// Generate alerts
	alerts := fm.generateAlerts(chunk, daysOld, freshnessScore)

	// Generate suggested actions
	actions := fm.generateSuggestedActions(chunk, freshnessScore, isStale)

	status := &FreshnessStatus{
		IsFresh:          isFresh,
		IsStale:          isStale,
		FreshnessScore:   freshnessScore,
		DaysOld:          daysOld,
		DecayRate:        decayRate,
		Alerts:           alerts,
		LastChecked:      now,
		SuggestedActions: actions,
	}

	return status, nil
}

// CheckRepositoryFreshness checks freshness for all chunks in a repository
func (fm *FreshnessManager) CheckRepositoryFreshness(ctx context.Context, repository string) (*FreshnessBatch, error) {
	start := time.Now()

	// Get all chunks for repository (simplified - would implement proper pagination)
	chunks, err := fm.storage.ListByRepository(ctx, repository, 1000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list repository chunks: %w", err)
	}

	results := make([]ChunkFreshnessResult, 0, len(chunks))
	freshCount := 0
	staleCount := 0
	alertsGenerated := 0

	for i := range chunks {
		chunk := &chunks[i]
		status, err := fm.CheckFreshness(ctx, chunk)
		if err != nil {
			continue // Skip errors, don't fail entire batch
		}

		result := ChunkFreshnessResult{
			ChunkID:         chunk.ID,
			Type:            chunk.Type,
			Repository:      chunk.Metadata.Repository,
			FreshnessStatus: *status,
		}

		results = append(results, result)

		if status.IsFresh {
			freshCount++
		}
		if status.IsStale {
			staleCount++
		}
		alertsGenerated += len(status.Alerts)
	}

	// Generate summary
	summary := fm.generateFreshnessSummary(results)

	return &FreshnessBatch{
		TotalChecked:    len(results),
		FreshCount:      freshCount,
		StaleCount:      staleCount,
		AlertsGenerated: alertsGenerated,
		ProcessingTime:  time.Since(start),
		Results:         results,
		Summary:         summary,
	}, nil
}

// MarkRefreshed marks a memory as recently refreshed/validated
func (fm *FreshnessManager) MarkRefreshed(ctx context.Context, chunkID, validationNotes string) error {
	chunk, err := fm.storage.GetByID(ctx, chunkID)
	if err != nil {
		return fmt.Errorf("chunk not found: %w", err)
	}

	// Update metadata to mark as refreshed
	if chunk.Metadata.ExtendedMetadata == nil {
		chunk.Metadata.ExtendedMetadata = make(map[string]interface{})
	}

	chunk.Metadata.ExtendedMetadata["last_refreshed"] = time.Now().Format(time.RFC3339)
	chunk.Metadata.ExtendedMetadata["refresh_validation_notes"] = validationNotes

	// Reset freshness tracking
	if chunk.Metadata.Quality != nil {
		chunk.Metadata.Quality.FreshnessScore = 1.0
		chunk.Metadata.Quality.RelevanceDecay = 0.0
		chunk.Metadata.Quality.CalculateOverallQuality()
	}

	return fm.storage.Update(ctx, chunk)
}

// GetStaleMemories returns memories that are considered stale
func (fm *FreshnessManager) GetStaleMemories(ctx context.Context, repository string, thresholdDays int) ([]types.ConversationChunk, error) {
	// Get chunks for repository
	chunks, err := fm.storage.ListByRepository(ctx, repository, 1000, 0)
	if err != nil {
		return nil, err
	}

	staleChunks := make([]types.ConversationChunk, 0)
	cutoffTime := time.Now().AddDate(0, 0, -thresholdDays)

	for i := range chunks {
		chunk := &chunks[i]
		if chunk.Timestamp.Before(cutoffTime) {
			// Check if it's actually stale based on type and content
			if fm.isStale(chunk, int(time.Since(chunk.Timestamp).Hours()/24)) {
				staleChunks = append(staleChunks, *chunk)
			}
		}
	}

	return staleChunks, nil
}

// Private helper methods

func (fm *FreshnessManager) determineDecayRate(chunk *types.ConversationChunk) float64 {
	// Start with content type rate
	rate, exists := fm.config.ContentTypeDecayRates[chunk.Type]
	if !exists {
		return fm.config.TechnologyDecayRates["default"]
	}

	// Adjust for detected technology
	return fm.adjustDecayRateForTechnology(rate, chunk)
}

// adjustDecayRateForTechnology adjusts decay rate based on detected technology
func (fm *FreshnessManager) adjustDecayRateForTechnology(baseRate float64, chunk *types.ConversationChunk) float64 {
	tech := fm.detectTechnology(chunk)
	if tech == "" {
		return baseRate
	}

	techRate, exists := fm.config.TechnologyDecayRates[tech]
	if !exists {
		return baseRate
	}

	// Use higher decay rate (more aggressive)
	if techRate > baseRate {
		return techRate
	}

	return baseRate
}

func (fm *FreshnessManager) detectTechnology(chunk *types.ConversationChunk) string {
	content := strings.ToLower(chunk.Content + " " + chunk.Summary)

	// Technology detection patterns
	techPatterns := map[string][]string{
		"nodejs":     {"node.js", "nodejs", "npm", "package.json"},
		"react":      {"react", "jsx", "component", "usestate", "useeffect"},
		"javascript": {"javascript", "js", "es6", "es2015", "typescript"},
		"python":     {"python", "pip", "django", "flask", "pandas"},
		"golang":     {"golang", "go", "go.mod", "go.sum"},
		"docker":     {"docker", "dockerfile", "container", "docker-compose"},
		"kubernetes": {"kubernetes", "k8s", "kubectl", "pod", "deployment"},
		"aws":        {"aws", "s3", "ec2", "lambda", "cloudformation"},
	}

	for tech, patterns := range techPatterns {
		for _, pattern := range patterns {
			if strings.Contains(content, pattern) {
				return tech
			}
		}
	}

	// Check tags for technology hints
	for _, tag := range chunk.Metadata.Tags {
		tagLower := strings.ToLower(tag)
		for tech, patterns := range techPatterns {
			for _, pattern := range patterns {
				if strings.Contains(tagLower, pattern) {
					return tech
				}
			}
		}
	}

	return ""
}

func (fm *FreshnessManager) calculateFreshnessScore(daysOld int, decayRate float64) float64 {
	// Exponential decay: score = e^(-rate * days)
	score := 1.0
	for i := 0; i < daysOld; i++ {
		score *= (1.0 - decayRate)
	}

	// Ensure score is between 0 and 1
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score
}

func (fm *FreshnessManager) isFresh(chunk *types.ConversationChunk, daysOld int) bool {
	// Check if recently refreshed
	if lastRefreshed := fm.getLastRefreshed(chunk); lastRefreshed != nil {
		refreshAge := time.Since(*lastRefreshed).Hours() / 24
		if refreshAge < 30 { // Refreshed within 30 days
			return true
		}
	}

	// Check against type-specific thresholds
	threshold := fm.getFreshnessThreshold(chunk)
	return daysOld < threshold*30 // Convert months to days (rough)
}

func (fm *FreshnessManager) isStale(chunk *types.ConversationChunk, daysOld int) bool {
	threshold := fm.getFreshnessThreshold(chunk)
	staleThreshold := threshold * 45 // 1.5x the freshness threshold

	return daysOld > staleThreshold
}

func (fm *FreshnessManager) getFreshnessThreshold(chunk *types.ConversationChunk) int {
	// Check technology-specific thresholds first
	tech := fm.detectTechnology(chunk)
	if tech != "" {
		if threshold, exists := fm.config.FreshnessThresholds.TechnologySpecific[tech]; exists {
			return threshold
		}
	}

	// Check content type thresholds
	switch chunk.Type {
	case types.ChunkTypeArchitectureDecision:
		return fm.config.FreshnessThresholds.ArchitectureDecisions
	case types.ChunkTypeSolution:
		// Check if it's a bug fix
		if fm.isBugFix(chunk) {
			return fm.config.FreshnessThresholds.BugFixes
		}
		return fm.config.FreshnessThresholds.GeneralContent
	case types.ChunkTypeProblem, types.ChunkTypeCodeChange, types.ChunkTypeDiscussion,
		types.ChunkTypeSessionSummary, types.ChunkTypeAnalysis, types.ChunkTypeVerification,
		types.ChunkTypeQuestion:
		return fm.config.FreshnessThresholds.GeneralContent
	// Task-oriented chunk types
	case types.ChunkTypeTask:
		// High priority tasks get architecture decision threshold (longer retention)
		if chunk.Metadata.TaskPriority != nil && *chunk.Metadata.TaskPriority == "high" {
			return fm.config.FreshnessThresholds.ArchitectureDecisions
		}
		return fm.config.FreshnessThresholds.GeneralContent
	case types.ChunkTypeTaskUpdate, types.ChunkTypeTaskProgress:
		// Task progress and updates are general content
		return fm.config.FreshnessThresholds.GeneralContent
	default:
		return fm.config.FreshnessThresholds.GeneralContent
	}
}

func (fm *FreshnessManager) isBugFix(chunk *types.ConversationChunk) bool {
	content := strings.ToLower(chunk.Content + " " + chunk.Summary)
	bugPatterns := []string{"bug", "fix", "error", "issue", "problem", "broken"}

	for _, pattern := range bugPatterns {
		if strings.Contains(content, pattern) {
			return true
		}
	}

	// Check tags
	for _, tag := range chunk.Metadata.Tags {
		if strings.Contains(strings.ToLower(tag), "bug") || strings.Contains(strings.ToLower(tag), "fix") {
			return true
		}
	}

	return false
}

func (fm *FreshnessManager) generateAlerts(chunk *types.ConversationChunk, daysOld int, freshnessScore float64) []FreshnessAlert {
	alerts := make([]FreshnessAlert, 0)

	// Use freshness score to determine alert severity
	severity := "low"
	if freshnessScore < 0.3 {
		severity = "high"
	} else if freshnessScore < 0.6 {
		severity = "medium"
	}

	// Technology version alerts
	if fm.config.StalenessAlerts.TechnologyVersions {
		if alert := fm.checkTechnologyVersionAlert(chunk, daysOld); alert != nil {
			alerts = append(alerts, *alert)
		}
	}

	// Architecture decision alerts - use freshness score for severity
	if fm.config.StalenessAlerts.ArchitectureDecisions && chunk.Type == types.ChunkTypeArchitectureDecision {
		if daysOld > fm.config.FreshnessThresholds.ArchitectureDecisions*30 {
			alerts = append(alerts, FreshnessAlert{
				Type:         "architecture_decision_stale",
				Severity:     severity,
				Message:      "Architecture decision may be outdated",
				Reason:       fmt.Sprintf("Decision is %d days old (freshness: %.2f)", daysOld, freshnessScore),
				Detected:     time.Now(),
				ActionNeeded: "Review and validate current relevance",
			})
		}
	}

	// Security content alerts
	if fm.config.StalenessAlerts.SecurityContent {
		if alert := fm.checkSecurityContentAlert(chunk, daysOld); alert != nil {
			alerts = append(alerts, *alert)
		}
	}

	// Performance metrics alerts
	if fm.config.StalenessAlerts.PerformanceMetrics {
		if alert := fm.checkPerformanceMetricsAlert(chunk, daysOld); alert != nil {
			alerts = append(alerts, *alert)
		}
	}

	// General freshness alert based on score
	if freshnessScore < 0.4 && daysOld > 30 {
		alerts = append(alerts, FreshnessAlert{
			Type:         "low_freshness_score",
			Severity:     severity,
			Message:      "Memory freshness score is low",
			Reason:       fmt.Sprintf("Freshness score %.2f is below threshold", freshnessScore),
			Detected:     time.Now(),
			ActionNeeded: "Consider refreshing or archiving this memory",
		})
	}

	return alerts
}

func (fm *FreshnessManager) checkTechnologyVersionAlert(chunk *types.ConversationChunk, daysOld int) *FreshnessAlert {
	content := strings.ToLower(chunk.Content)

	// Look for version patterns
	versionPattern := regexp.MustCompile(`\b\d+\.\d+(\.\d+)?\b`)
	if !versionPattern.MatchString(content) {
		return nil
	}

	tech := fm.detectTechnology(chunk)
	if tech == "" {
		return nil
	}

	threshold := fm.getTechnologyVersionThreshold(tech)
	if daysOld <= threshold {
		return nil
	}

	return &FreshnessAlert{
		Type:         "technology_version_stale",
		Severity:     "high",
		Message:      tech + " version information may be outdated",
		Reason:       "Technology reference is " + strconv.Itoa(daysOld) + " days old",
		Detected:     time.Now(),
		ActionNeeded: "Verify current version compatibility",
	}
}

// getTechnologyVersionThreshold returns the staleness threshold for technology versions
func (fm *FreshnessManager) getTechnologyVersionThreshold(tech string) int {
	// Fast-moving technologies have shorter thresholds
	if tech == "nodejs" || tech == "react" {
		return 60 // 2 months for fast-moving tech
	}
	return 90 // 3 months for general tech versions
}

func (fm *FreshnessManager) checkSecurityContentAlert(chunk *types.ConversationChunk, daysOld int) *FreshnessAlert {
	content := strings.ToLower(chunk.Content + " " + chunk.Summary)
	securityKeywords := []string{"security", "vulnerability", "cve", "exploit", "patch", "auth", "encryption"}

	for _, keyword := range securityKeywords {
		if strings.Contains(content, keyword) {
			if daysOld > 180 { // 6 months for security content
				return &FreshnessAlert{
					Type:         "security_content_stale",
					Severity:     "high",
					Message:      "Security-related content may be outdated",
					Reason:       "Security content is " + strconv.Itoa(daysOld) + " days old",
					Detected:     time.Now(),
					ActionNeeded: "Review for current security best practices",
				}
			}
			break
		}
	}

	return nil
}

func (fm *FreshnessManager) checkPerformanceMetricsAlert(chunk *types.ConversationChunk, daysOld int) *FreshnessAlert {
	content := strings.ToLower(chunk.Content + " " + chunk.Summary)
	perfKeywords := []string{"performance", "benchmark", "metrics", "latency", "throughput", "response time"}

	for _, keyword := range perfKeywords {
		if strings.Contains(content, keyword) {
			if daysOld > 120 { // 4 months for performance metrics
				return &FreshnessAlert{
					Type:         "performance_metrics_stale",
					Severity:     "medium",
					Message:      "Performance metrics may be outdated",
					Reason:       "Performance data is " + strconv.Itoa(daysOld) + " days old",
					Detected:     time.Now(),
					ActionNeeded: "Re-measure current performance",
				}
			}
			break
		}
	}

	return nil
}

func (fm *FreshnessManager) generateSuggestedActions(chunk *types.ConversationChunk, freshnessScore float64, isStale bool) []SuggestedAction {
	actions := make([]SuggestedAction, 0)

	if isStale {
		// High priority: refresh stale content
		actions = append(actions, SuggestedAction{
			Action:     "refresh",
			Priority:   "high",
			Reason:     "Content is considered stale and may be outdated",
			Confidence: 0.8,
		})

		// Consider archiving if very old and low value
		if freshnessScore < 0.1 && chunk.Metadata.Outcome != types.OutcomeSuccess {
			actions = append(actions, SuggestedAction{
				Action:     "archive",
				Priority:   "medium",
				Reason:     "Very old content with uncertain value",
				Confidence: 0.6,
			})
		}
	} else if freshnessScore < 0.5 {
		// Medium priority: verify aging content
		actions = append(actions, SuggestedAction{
			Action:     "verify",
			Priority:   "medium",
			Reason:     "Content is aging and should be verified",
			Confidence: 0.7,
		})
	}

	// Technology-specific suggestions
	tech := fm.detectTechnology(chunk)
	if tech != "" && (tech == "nodejs" || tech == "react") {
		if freshnessScore < 0.7 {
			actions = append(actions, SuggestedAction{
				Action:     "update",
				Priority:   "high",
				Reason:     tech + " content in fast-moving ecosystem",
				Confidence: 0.9,
			})
		}
	}

	return actions
}

func (fm *FreshnessManager) getLastRefreshed(chunk *types.ConversationChunk) *time.Time {
	if chunk.Metadata.ExtendedMetadata == nil {
		return nil
	}

	if refreshedStr, ok := chunk.Metadata.ExtendedMetadata["last_refreshed"].(string); ok {
		if refreshed, err := time.Parse(time.RFC3339, refreshedStr); err == nil {
			return &refreshed
		}
	}

	return nil
}

func (fm *FreshnessManager) generateFreshnessSummary(results []ChunkFreshnessResult) FreshnessSummary {
	summary := FreshnessSummary{
		ByRepository: make(map[string]FreshnessStats),
		ByType:       make(map[string]FreshnessStats),
		ByTechnology: make(map[string]FreshnessStats),
	}

	// Process results to build summary
	for i := range results {
		result := &results[i]
		// By repository
		repoStats := summary.ByRepository[result.Repository]
		repoStats.TotalChunks++
		if result.FreshnessStatus.IsFresh {
			repoStats.FreshChunks++
		}
		if result.FreshnessStatus.IsStale {
			repoStats.StaleChunks++
		}
		repoStats.AlertCount += len(result.FreshnessStatus.Alerts)
		summary.ByRepository[result.Repository] = repoStats

		// By type
		typeKey := string(result.Type)
		typeStats := summary.ByType[typeKey]
		typeStats.TotalChunks++
		if result.FreshnessStatus.IsFresh {
			typeStats.FreshChunks++
		}
		if result.FreshnessStatus.IsStale {
			typeStats.StaleChunks++
		}
		typeStats.AlertCount += len(result.FreshnessStatus.Alerts)
		summary.ByType[typeKey] = typeStats
	}

	// Calculate averages and overall health
	totalFresh := 0
	totalStale := 0
	totalChunks := len(results)

	for _, stats := range summary.ByRepository {
		totalFresh += stats.FreshChunks
		totalStale += stats.StaleChunks
	}

	if totalChunks > 0 {
		freshRatio := float64(totalFresh) / float64(totalChunks)
		switch {
		case freshRatio > 0.8:
			summary.OverallHealth = "excellent"
		case freshRatio > 0.6:
			summary.OverallHealth = "good"
		case freshRatio > 0.4:
			summary.OverallHealth = "poor"
		default:
			summary.OverallHealth = "critical"
		}
	}

	return summary
}
