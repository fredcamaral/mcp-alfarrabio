package intelligence

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// ResolutionStrategy represents a strategy for resolving conflicts
type ResolutionStrategy struct {
	Type        ConflictResolutionType `json:"type"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Confidence  float64                `json:"confidence"`
	Rationale   string                 `json:"rationale"`
	Steps       []ResolutionStep       `json:"steps"`
	Risks       []string               `json:"risks"`
	Benefits    []string               `json:"benefits"`
	Context     map[string]any         `json:"context"`
}

// ResolutionStep represents a step in conflict resolution
type ResolutionStep struct {
	Order       int    `json:"order"`
	Action      string `json:"action"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// ResolutionRecommendation contains recommendations for resolving conflicts
type ResolutionRecommendation struct {
	ConflictID   string               `json:"conflict_id"`
	ConflictType ConflictType         `json:"conflict_type"`
	Severity     ConflictSeverity     `json:"severity"`
	Strategies   []ResolutionStrategy `json:"strategies"`
	Recommended  *ResolutionStrategy  `json:"recommended,omitempty"`
	Context      ConflictContext      `json:"context"`
	GeneratedAt  time.Time            `json:"generated_at"`
	ValidUntil   time.Time            `json:"valid_until"`
}

// ConflictContext provides additional context for resolution
type ConflictContext struct {
	Repository        string            `json:"repository"`
	AffectedFiles     []string          `json:"affected_files"`
	RelatedPatterns   []string          `json:"related_patterns"`
	StakeholderImpact map[string]string `json:"stakeholder_impact"`
	BusinessContext   map[string]any    `json:"business_context"`
	TechnicalContext  map[string]any    `json:"technical_context"`
}

// ConflictResolver provides conflict resolution strategies and recommendations
type ConflictResolver struct {
	// Configuration
	defaultStrategyCount int
	maxResolutionAge     time.Duration

	// Strategy weights for different conflict types
	strategyWeights map[ConflictType]map[ConflictResolutionType]float64
}

// NewConflictResolver creates a new conflict resolution engine
func NewConflictResolver() *ConflictResolver {
	return &ConflictResolver{
		defaultStrategyCount: 3,
		maxResolutionAge:     7 * 24 * time.Hour, // 7 days
		strategyWeights:      initializeStrategyWeights(),
	}
}

// ResolveConflicts generates resolution recommendations for conflicts
func (cr *ConflictResolver) ResolveConflicts(ctx context.Context, conflicts []Conflict) ([]ResolutionRecommendation, error) {
	recommendations := make([]ResolutionRecommendation, 0, len(conflicts))

	for i := range conflicts {
		recommendation := cr.generateResolutionRecommendation(&conflicts[i])
		recommendations = append(recommendations, recommendation)
	}

	return recommendations, nil
}

// GenerateResolutionStrategies creates multiple resolution strategies for a conflict
func (cr *ConflictResolver) GenerateResolutionStrategies(conflict *Conflict) []ResolutionStrategy {
	var strategies []ResolutionStrategy

	// Generate different types of strategies based on conflict type
	switch conflict.Type {
	case ConflictTypeArchitectural:
		strategies = cr.generateArchitecturalStrategies(conflict)
	case ConflictTypeTechnical:
		strategies = cr.generateTechnicalStrategies(conflict)
	case ConflictTypeTemporal:
		strategies = cr.generateTemporalStrategies(conflict)
	case ConflictTypeOutcome:
		strategies = cr.generateOutcomeStrategies(conflict)
	case ConflictTypeDecision:
		strategies = cr.generateDecisionStrategies(conflict)
	case ConflictTypeMethodology:
		strategies = cr.generateMethodologyStrategies(conflict)
	case ConflictTypePattern:
		strategies = cr.generatePatternStrategies(conflict)
	default:
		strategies = cr.generateGenericStrategies(conflict)
	}

	// Score and sort strategies
	for i := range strategies {
		strategies[i].Confidence = cr.calculateStrategyConfidence(conflict, strategies[i])
	}

	sort.Slice(strategies, func(i, j int) bool {
		return strategies[i].Confidence > strategies[j].Confidence
	})

	// Limit to default count
	if len(strategies) > cr.defaultStrategyCount {
		strategies = strategies[:cr.defaultStrategyCount]
	}

	return strategies
}

// generateResolutionRecommendation creates a comprehensive resolution recommendation
func (cr *ConflictResolver) generateResolutionRecommendation(conflict *Conflict) ResolutionRecommendation {
	strategies := cr.GenerateResolutionStrategies(conflict)

	var recommended *ResolutionStrategy
	if len(strategies) > 0 {
		recommended = &strategies[0]
	}

	resolutionContext := cr.buildConflictContext(conflict)

	return ResolutionRecommendation{
		ConflictID:   conflict.ID,
		ConflictType: conflict.Type,
		Severity:     conflict.Severity,
		Strategies:   strategies,
		Recommended:  recommended,
		Context:      resolutionContext,
		GeneratedAt:  time.Now(),
		ValidUntil:   time.Now().Add(cr.maxResolutionAge),
	}
}

// Strategy generation methods for different conflict types

func (cr *ConflictResolver) generateArchitecturalStrategies(_ *Conflict) []ResolutionStrategy {
	return []ResolutionStrategy{
		// Strategy 1: Accept latest architectural decision
		{
			Type:        ResolutionAcceptLatest,
			Title:       "Accept Latest Architecture Decision",
			Description: "Adopt the most recent architectural decision as the current standard",
			Rationale:   "Architectural decisions evolve over time. The latest decision likely incorporates lessons learned.",
			Steps: []ResolutionStep{
				{Order: 1, Action: "review", Description: "Review the context of the latest decision", Required: true},
				{Order: 2, Action: "document", Description: "Document the rationale for accepting the latest decision", Required: true},
				{Order: 3, Action: "update", Description: "Update architectural documentation", Required: true},
				{Order: 4, Action: "communicate", Description: "Communicate the decision to stakeholders", Required: true},
			},
			Benefits: []string{
				"Maintains consistency with current direction",
				"Avoids confusion about current standards",
				"Respects evolution of architectural thinking",
			},
			Risks: []string{
				"May discard valuable insights from earlier decisions",
				"Could introduce regression if latest decision is flawed",
			},
			Context: map[string]any{
				"strategy_type": "temporal_precedence",
				"applicable_to": []string{"architecture", "design_patterns"},
			},
		},
		// Strategy 2: Merge architectural approaches
		{
			Type:        ResolutionMerge,
			Title:       "Merge Architectural Approaches",
			Description: "Combine the best aspects of conflicting architectural decisions",
			Rationale:   "Both decisions may have valid points that can be integrated into a hybrid approach.",
			Steps: []ResolutionStep{
				{Order: 1, Action: "analyze", Description: "Analyze the strengths of each architectural approach", Required: true},
				{Order: 2, Action: "design", Description: "Design a hybrid approach that incorporates the best elements", Required: true},
				{Order: 3, Action: "validate", Description: "Validate the hybrid approach with prototyping or modeling", Required: true},
				{Order: 4, Action: "document", Description: "Document the new merged architectural decision", Required: true},
			},
			Benefits: []string{
				"Leverages strengths of multiple approaches",
				"May result in superior solution",
				"Demonstrates thorough consideration of alternatives",
			},
			Risks: []string{
				"May result in complexity",
				"Could compromise the clarity of each individual approach",
				"Requires additional design and validation effort",
			},
			Context: map[string]any{
				"strategy_type": "synthesis",
				"complexity":    "high",
			},
		},
		// Strategy 3: Context-specific resolution
		{
			Type:        ResolutionContextual,
			Title:       "Apply Context-Specific Architecture",
			Description: "Use different architectural approaches for different contexts or components",
			Rationale:   "Different parts of the system may benefit from different architectural approaches.",
			Steps: []ResolutionStep{
				{Order: 1, Action: "segment", Description: "Identify system components or contexts", Required: true},
				{Order: 2, Action: "map", Description: "Map appropriate architectural approaches to each context", Required: true},
				{Order: 3, Action: "define", Description: "Define clear boundaries and interfaces", Required: true},
				{Order: 4, Action: "govern", Description: "Establish governance for context-specific decisions", Required: true},
			},
			Benefits: []string{
				"Optimizes architecture for specific use cases",
				"Allows coexistence of different approaches",
				"Provides flexibility for future evolution",
			},
			Risks: []string{
				"Increases system complexity",
				"May lead to inconsistent patterns across the system",
				"Requires careful interface management",
			},
			Context: map[string]any{
				"strategy_type":       "contextual_separation",
				"governance_required": true,
			},
		},
	}
}

func (cr *ConflictResolver) generateTechnicalStrategies(_ *Conflict) []ResolutionStrategy {
	return []ResolutionStrategy{
		// Strategy 1: Benchmark and choose
		{
			Type:        ResolutionAcceptHighest,
			Title:       "Benchmark and Choose Best Performance",
			Description: "Test both approaches and select the one with better performance metrics",
			Rationale:   "Empirical testing provides objective basis for technical decisions.",
			Steps: []ResolutionStep{
				{Order: 1, Action: "define", Description: "Define benchmarking criteria and metrics", Required: true},
				{Order: 2, Action: "implement", Description: "Implement test scenarios for both approaches", Required: true},
				{Order: 3, Action: "measure", Description: "Execute benchmarks and collect metrics", Required: true},
				{Order: 4, Action: "analyze", Description: "Analyze results and make evidence-based decision", Required: true},
			},
			Benefits: []string{
				"Objective, data-driven decision making",
				"Validates performance assumptions",
				"Provides documentation for future reference",
			},
			Risks: []string{
				"Benchmarks may not reflect real-world usage",
				"Time and resource intensive",
				"May not capture all relevant factors",
			},
			Context: map[string]any{
				"strategy_type":      "empirical_testing",
				"resources_required": "medium",
			},
		},
		// Strategy 2: Evolutionary approach
		{
			Type:        ResolutionEvolutionary,
			Title:       "Implement Evolutionary Migration",
			Description: "Gradually migrate from one approach to another based on evidence",
			Rationale:   "Allows testing and validation in production with reduced risk.",
			Steps: []ResolutionStep{
				{Order: 1, Action: "plan", Description: "Plan migration phases with rollback capability", Required: true},
				{Order: 2, Action: "pilot", Description: "Implement pilot with subset of functionality", Required: true},
				{Order: 3, Action: "monitor", Description: "Monitor performance and stability metrics", Required: true},
				{Order: 4, Action: "decide", Description: "Decide on full migration based on pilot results", Required: true},
			},
			Benefits: []string{
				"Reduces risk through incremental change",
				"Provides real-world validation",
				"Allows learning and adjustment during migration",
			},
			Risks: []string{
				"May result in temporary inconsistency",
				"Requires dual support during transition",
				"Can extend timeline for resolution",
			},
			Context: map[string]any{
				"strategy_type": "gradual_migration",
				"risk_level":    "low",
			},
		},
	}
}

func (cr *ConflictResolver) generateTemporalStrategies(conflict *Conflict) []ResolutionStrategy {
	strategies := []ResolutionStrategy{}

	// Strategy 1: Accept latest
	strategies = append(strategies, ResolutionStrategy{
		Type:        ResolutionAcceptLatest,
		Title:       "Accept Most Recent Information",
		Description: "Use the most recent information as the current truth",
		Rationale:   "Latest information likely reflects current state and recent learnings.",
		Steps: []ResolutionStep{
			{Order: 1, Action: "verify", Description: "Verify the accuracy of the latest information", Required: true},
			{Order: 2, Action: "document", Description: "Document the change in understanding", Required: true},
			{Order: 3, Action: "update", Description: "Update dependent systems or processes", Required: false},
		},
		Benefits: []string{
			"Reflects current understanding",
			"Simple resolution approach",
			"Maintains forward momentum",
		},
		Risks: []string{
			"May discard still-relevant historical context",
			"Could be based on incomplete recent information",
		},
		Context: map[string]any{
			"strategy_type":   "temporal_precedence",
			"time_difference": conflict.TimeDifference.String(),
		},
	})

	return strategies
}

func (cr *ConflictResolver) generateOutcomeStrategies(_ *Conflict) []ResolutionStrategy {
	strategies := []ResolutionStrategy{}

	// Strategy 1: Investigate root cause
	strategies = append(strategies, ResolutionStrategy{
		Type:        ResolutionManualReview,
		Title:       "Investigate Outcome Discrepancy",
		Description: "Analyze why similar approaches had different outcomes",
		Rationale:   "Understanding outcome differences can reveal important context or conditions.",
		Steps: []ResolutionStep{
			{Order: 1, Action: "identify", Description: "Identify factors that may explain outcome difference", Required: true},
			{Order: 2, Action: "analyze", Description: "Analyze environmental or contextual differences", Required: true},
			{Order: 3, Action: "document", Description: "Document findings and updated understanding", Required: true},
			{Order: 4, Action: "generalize", Description: "Extract general principles or conditions", Required: false},
		},
		Benefits: []string{
			"Improves understanding of success factors",
			"May reveal important contextual dependencies",
			"Prevents future similar conflicts",
		},
		Risks: []string{
			"Time-intensive investigation",
			"May not yield clear conclusions",
		},
		Context: map[string]any{
			"strategy_type":          "root_cause_analysis",
			"investigation_required": true,
		},
	})

	return strategies
}

func (cr *ConflictResolver) generateDecisionStrategies(_ *Conflict) []ResolutionStrategy {
	strategies := []ResolutionStrategy{}

	// Strategy 1: Re-evaluate with current context
	strategies = append(strategies, ResolutionStrategy{
		Type:        ResolutionContextual,
		Title:       "Re-evaluate Decision with Current Context",
		Description: "Reassess the decision considering current circumstances and knowledge",
		Rationale:   "Decisions should be periodically re-evaluated as context changes.",
		Steps: []ResolutionStep{
			{Order: 1, Action: "gather", Description: "Gather current context and constraints", Required: true},
			{Order: 2, Action: "evaluate", Description: "Evaluate options against current criteria", Required: true},
			{Order: 3, Action: "decide", Description: "Make updated decision", Required: true},
			{Order: 4, Action: "communicate", Description: "Communicate decision rationale", Required: true},
		},
		Benefits: []string{
			"Ensures decisions remain relevant",
			"Incorporates latest knowledge and constraints",
			"Provides clear rationale for current choice",
		},
		Risks: []string{
			"May lead to frequent decision changes",
			"Requires effort to re-evaluate",
		},
		Context: map[string]any{
			"strategy_type":       "contextual_re_evaluation",
			"decision_governance": true,
		},
	})

	return strategies
}

func (cr *ConflictResolver) generateMethodologyStrategies(_ *Conflict) []ResolutionStrategy {
	strategies := []ResolutionStrategy{}

	// Strategy 1: Establish standard methodology
	strategies = append(strategies, ResolutionStrategy{
		Type:        ResolutionDomain,
		Title:       "Establish Standard Methodology",
		Description: "Define and adopt a standard methodology for consistency",
		Rationale:   "Consistent methodology improves quality and reduces confusion.",
		Steps: []ResolutionStep{
			{Order: 1, Action: "research", Description: "Research industry best practices", Required: true},
			{Order: 2, Action: "adapt", Description: "Adapt methodology to organization context", Required: true},
			{Order: 3, Action: "document", Description: "Document standard methodology", Required: true},
			{Order: 4, Action: "train", Description: "Train team on standard methodology", Required: true},
		},
		Benefits: []string{
			"Provides consistency across projects",
			"Reduces decision fatigue",
			"Improves quality and predictability",
		},
		Risks: []string{
			"May not fit all situations",
			"Could stifle innovation or flexibility",
		},
		Context: map[string]any{
			"strategy_type": "standardization",
			"scope":         "organization_wide",
		},
	})

	return strategies
}

func (cr *ConflictResolver) generateGenericStrategies(_ *Conflict) []ResolutionStrategy {
	strategies := []ResolutionStrategy{}

	// Generic strategy 1: Manual review
	strategies = append(strategies, ResolutionStrategy{
		Type:        ResolutionManualReview,
		Title:       "Manual Expert Review",
		Description: "Have domain experts review and resolve the conflict",
		Rationale:   "Complex conflicts may require human judgment and domain expertise.",
		Steps: []ResolutionStep{
			{Order: 1, Action: "assign", Description: "Assign qualified reviewers", Required: true},
			{Order: 2, Action: "review", Description: "Conduct thorough review of conflicting information", Required: true},
			{Order: 3, Action: "decide", Description: "Make resolution decision", Required: true},
			{Order: 4, Action: "document", Description: "Document decision and rationale", Required: true},
		},
		Benefits: []string{
			"Leverages human expertise",
			"Can handle complex edge cases",
			"Provides authoritative resolution",
		},
		Risks: []string{
			"Time and resource intensive",
			"May introduce human bias",
			"Dependent on reviewer availability and expertise",
		},
		Context: map[string]any{
			"strategy_type":      "human_review",
			"expertise_required": true,
		},
	})

	return strategies
}

// Helper methods

// Helper function to get base confidence from strategy weights
func (cr *ConflictResolver) getBaseConfidenceFromWeights(conflict *Conflict, strategy ResolutionStrategy) float64 {
	baseConfidence := 0.5

	if weights, exists := cr.strategyWeights[conflict.Type]; exists {
		if weight, exists := weights[strategy.Type]; exists {
			baseConfidence = weight
		}
	}

	return baseConfidence
}

// Helper function to adjust confidence based on severity
func (cr *ConflictResolver) adjustConfidenceForSeverity(baseConfidence float64, conflict *Conflict, strategy ResolutionStrategy) float64 {
	switch conflict.Severity {
	case SeverityCritical:
		if strategy.Type == ResolutionManualReview {
			return baseConfidence + 0.2
		}
	case SeverityHigh:
		if strategy.Type == ResolutionAcceptHighest || strategy.Type == ResolutionManualReview {
			return baseConfidence + 0.1
		}
	case SeverityMedium:
		if strategy.Type == ResolutionMerge || strategy.Type == ResolutionContextual {
			return baseConfidence + 0.1
		}
	case SeverityLow:
		if strategy.Type == ResolutionAcceptLatest || strategy.Type == ResolutionMerge {
			return baseConfidence + 0.05
		}
	case SeverityInfo:
		return baseConfidence + 0.02 // Small boost for informational conflicts
	}

	return baseConfidence
}

// Helper function to clamp confidence within valid bounds
func (cr *ConflictResolver) clampConfidence(confidence float64) float64 {
	if confidence > 1.0 {
		return 1.0
	}
	if confidence < 0.0 {
		return 0.0
	}
	return confidence
}

func (cr *ConflictResolver) calculateStrategyConfidence(conflict *Conflict, strategy ResolutionStrategy) float64 {
	// Get base confidence from strategy weights
	baseConfidence := cr.getBaseConfidenceFromWeights(conflict, strategy)

	// Adjust based on conflict severity
	baseConfidence = cr.adjustConfidenceForSeverity(baseConfidence, conflict, strategy)

	// Adjust based on conflict confidence
	confidenceBonus := (conflict.Confidence - 0.5) * 0.2
	baseConfidence += confidenceBonus

	// Ensure confidence is within bounds
	return cr.clampConfidence(baseConfidence)
}

func (cr *ConflictResolver) buildConflictContext(conflict *Conflict) ConflictContext {
	conflictCtx := ConflictContext{
		Repository:        extractRepository(conflict),
		AffectedFiles:     extractAffectedFiles(conflict),
		RelatedPatterns:   []string{}, // Would be populated by pattern analysis
		StakeholderImpact: extractStakeholderImpact(conflict),
		BusinessContext:   map[string]any{},
		TechnicalContext:  conflict.Context,
	}

	return conflictCtx
}

func extractRepository(conflict *Conflict) string {
	if repo := conflict.PrimaryChunk.Metadata.Repository; repo != "" {
		return repo
	}
	if repo := conflict.ConflictChunk.Metadata.Repository; repo != "" {
		return repo
	}
	return "unknown"
}

func extractAffectedFiles(conflict *Conflict) []string {
	files := make(map[string]bool)

	// Collect files from all related chunks
	for _, file := range conflict.PrimaryChunk.Metadata.FilesModified {
		files[file] = true
	}
	for _, file := range conflict.ConflictChunk.Metadata.FilesModified {
		files[file] = true
	}
	for i := range conflict.RelatedChunks {
		chunk := &conflict.RelatedChunks[i]
		for _, file := range chunk.Metadata.FilesModified {
			files[file] = true
		}
	}

	result := make([]string, 0, len(files))
	for file := range files {
		result = append(result, file)
	}

	return result
}

func extractStakeholderImpact(conflict *Conflict) map[string]string {
	impact := make(map[string]string)

	// Determine impact based on conflict type
	switch conflict.Type {
	case ConflictTypeArchitectural:
		impact["architects"] = string(SeverityHigh)
		impact["developers"] = string(SeverityMedium)
		impact["operations"] = string(SeverityMedium)
	case ConflictTypeTechnical:
		impact["developers"] = string(SeverityHigh)
		impact["qa"] = string(SeverityMedium)
		impact["operations"] = string(SeverityLow)
	case ConflictTypeDecision:
		impact["leadership"] = string(SeverityHigh)
		impact["product"] = string(SeverityMedium)
		impact["engineering"] = string(SeverityMedium)
	case ConflictTypeTemporal:
		impact["team"] = string(SeverityMedium)
		impact["stakeholders"] = string(SeverityLow)
	case ConflictTypeMethodology:
		impact["team"] = string(SeverityHigh)
		impact["process"] = string(SeverityHigh)
	case ConflictTypeOutcome:
		impact["leadership"] = string(SeverityHigh)
		impact["stakeholders"] = string(SeverityMedium)
	case ConflictTypePattern:
		impact["team"] = string(SeverityMedium)
		impact["analysts"] = string(SeverityHigh)
	default:
		impact["team"] = string(SeverityMedium)
	}

	return impact
}

// initializeStrategyWeights sets up strategy weights for different conflict types
func initializeStrategyWeights() map[ConflictType]map[ConflictResolutionType]float64 {
	weights := make(map[ConflictType]map[ConflictResolutionType]float64)

	weights[ConflictTypeArchitectural] = map[ConflictResolutionType]float64{
		ResolutionAcceptLatest: 0.7,
		ResolutionMerge:        0.8,
		ResolutionContextual:   0.9,
		ResolutionManualReview: 0.8,
		ResolutionEvolutionary: 0.6,
		ResolutionDomain:       0.7,
	}

	weights[ConflictTypeTechnical] = map[ConflictResolutionType]float64{
		ResolutionAcceptHighest: 0.9,
		ResolutionEvolutionary:  0.8,
		ResolutionMerge:         0.6,
		ResolutionManualReview:  0.7,
		ResolutionAcceptLatest:  0.5,
	}

	weights[ConflictTypeTemporal] = map[ConflictResolutionType]float64{
		ResolutionAcceptLatest: 0.9,
		ResolutionManualReview: 0.7,
		ResolutionContextual:   0.6,
	}

	weights[ConflictTypeOutcome] = map[ConflictResolutionType]float64{
		ResolutionManualReview:  0.9,
		ResolutionContextual:    0.8,
		ResolutionAcceptHighest: 0.7,
	}

	weights[ConflictTypeDecision] = map[ConflictResolutionType]float64{
		ResolutionContextual:   0.9,
		ResolutionManualReview: 0.8,
		ResolutionAcceptLatest: 0.7,
		ResolutionMerge:        0.6,
	}

	weights[ConflictTypeMethodology] = map[ConflictResolutionType]float64{
		ResolutionDomain:       0.9,
		ResolutionContextual:   0.8,
		ResolutionManualReview: 0.7,
	}

	weights[ConflictTypePattern] = map[ConflictResolutionType]float64{
		ResolutionContextual:   0.9,
		ResolutionMerge:        0.8,
		ResolutionManualReview: 0.7,
	}

	return weights
}

// generatePatternStrategies generates resolution strategies for pattern conflicts
func (cr *ConflictResolver) generatePatternStrategies(conflict *Conflict) []ResolutionStrategy {
	strategies := []ResolutionStrategy{}

	// Strategy 1: Merge patterns - customize based on conflict severity
	mergeTitle := "Merge Pattern Information"
	mergeDesc := "Combine insights from conflicting patterns"
	if conflict.Severity == SeverityHigh || conflict.Severity == SeverityCritical {
		mergeDesc = fmt.Sprintf("Carefully merge %s conflict: %s", conflict.Type, conflict.Description)
	}

	strategies = append(strategies, ResolutionStrategy{
		Type:        ResolutionMerge,
		Title:       mergeTitle,
		Description: mergeDesc,
		Rationale:   "Patterns may complement each other or represent different aspects.",
		Steps: []ResolutionStep{
			{Order: 1, Action: "analyze", Description: "Analyze both patterns for commonalities", Required: true},
			{Order: 2, Action: "identify", Description: "Identify unique aspects of each pattern", Required: true},
			{Order: 3, Action: "merge", Description: "Create unified pattern representation", Required: true},
		},
		Benefits: []string{
			"Preserves valuable insights from both patterns",
			"Creates more comprehensive understanding",
			"Maintains pattern completeness",
		},
		Risks: []string{
			"May create overly complex patterns",
			"Could dilute specific insights",
		},
	})

	// Strategy 2: Contextual resolution - adapt based on conflict context
	contextTitle := "Context-Based Pattern Selection"
	contextDesc := "Choose pattern based on specific context"
	if len(conflict.ConflictPoints) > 0 {
		contextDesc = fmt.Sprintf("Resolve conflict on '%s' using context", conflict.ConflictPoints[0].Aspect)
	}

	strategies = append(strategies, ResolutionStrategy{
		Type:        ResolutionContextual,
		Title:       contextTitle,
		Description: contextDesc,
		Rationale:   "Different patterns may apply to different contexts or scenarios.",
		Steps: []ResolutionStep{
			{Order: 1, Action: "context", Description: "Identify the specific context of application", Required: true},
			{Order: 2, Action: "evaluate", Description: "Evaluate which pattern fits the context best", Required: true},
			{Order: 3, Action: "document", Description: "Document pattern applicability contexts", Required: true},
		},
		Benefits: []string{
			"Provides context-appropriate solutions",
			"Maintains pattern specificity",
			"Enables situational application",
		},
		Risks: []string{
			"May create confusion about when to apply each pattern",
			"Requires clear context documentation",
		},
	})

	return strategies
}
