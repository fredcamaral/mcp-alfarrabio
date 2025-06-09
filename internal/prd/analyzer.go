// Package prd provides PRD content analysis functionality.
package prd

import (
	"regexp"
	"strings"

	"lerian-mcp-memory/pkg/types"
)

// Analyzer handles PRD content analysis
type Analyzer struct {
	config AnalyzerConfig
}

// AnalyzerConfig represents analyzer configuration
type AnalyzerConfig struct {
	EnableAIAnalysis     bool    `json:"enable_ai_analysis"`
	MinCompletenessScore float64 `json:"min_completeness_score"`
	MinClarityScore      float64 `json:"min_clarity_score"`
	StrictMode           bool    `json:"strict_mode"`
}

// DefaultAnalyzerConfig returns default analyzer configuration
func DefaultAnalyzerConfig() AnalyzerConfig {
	return AnalyzerConfig{
		EnableAIAnalysis:     true,
		MinCompletenessScore: 0.7,
		MinClarityScore:      0.6,
		StrictMode:           false,
	}
}

// NewAnalyzer creates a new PRD analyzer
func NewAnalyzer(config AnalyzerConfig) *Analyzer {
	return &Analyzer{
		config: config,
	}
}

// AnalyzeDocument performs comprehensive analysis of a PRD document
func (a *Analyzer) AnalyzeDocument(doc *types.PRDDocument) error {
	if doc == nil {
		return nil
	}

	analysis := &types.PRDAnalysis{
		Issues:          []types.AnalysisIssue{},
		Recommendations: []string{},
		MissingElements: []string{},
		KeyConcepts:     []string{},
		TechnicalTerms:  []string{},
		SimilarPRDs:     []string{},
	}

	// Perform different types of analysis
	analysis.ComplexityScore = a.calculateComplexityScore(doc)
	analysis.CompletenessScore = a.calculateCompletenessScore(doc)
	analysis.ClarityScore = a.calculateClarityScore(doc)
	analysis.StructureScore = a.calculateStructureScore(doc)
	analysis.ReadabilityScore = a.calculateReadabilityScore(doc)

	// Calculate overall quality score
	analysis.QualityScore = a.calculateQualityScore(analysis)

	// Identify issues
	a.identifyIssues(doc, analysis)

	// Extract key concepts and technical terms
	a.extractKeyInformation(doc, analysis)

	// Generate recommendations
	a.generateRecommendations(doc, analysis)

	// Identify missing elements
	a.identifyMissingElements(doc, analysis)

	doc.Analysis = *analysis
	return nil
}

// calculateComplexityScore calculates the complexity score of the document
func (a *Analyzer) calculateComplexityScore(doc *types.PRDDocument) float64 {
	score := 0.0
	factors := 0

	// Factor 1: Number of sections
	sectionCount := len(doc.Content.Sections)
	if sectionCount > 0 {
		sectionScore := float64(sectionCount) / 20.0 // Normalize to 0-1, assuming 20 sections = high complexity
		if sectionScore > 1.0 {
			sectionScore = 1.0
		}
		score += sectionScore
		factors++
	}

	// Factor 2: Document length (word count)
	wordCount := doc.Content.WordCount
	if wordCount > 0 {
		wordScore := float64(wordCount) / 10000.0 // Normalize to 0-1, assuming 10k words = high complexity
		if wordScore > 1.0 {
			wordScore = 1.0
		}
		score += wordScore
		factors++
	}

	// Factor 3: Technical complexity (presence of technical terms)
	techTerms := a.countTechnicalTerms(doc.Content.Raw)
	if techTerms > 0 {
		techScore := float64(techTerms) / 50.0 // Normalize to 0-1, assuming 50 tech terms = high complexity
		if techScore > 1.0 {
			techScore = 1.0
		}
		score += techScore
		factors++
	}

	// Factor 4: Structure complexity (depth and nested sections)
	depthScore := float64(doc.Content.Structure.MaxDepth) / 6.0 // Normalize to 0-1, assuming depth 6 = high complexity
	if depthScore > 1.0 {
		depthScore = 1.0
	}
	score += depthScore
	factors++

	if factors == 0 {
		return 0.0
	}

	return score / float64(factors)
}

// calculateCompletenessScore calculates how complete the PRD is
func (a *Analyzer) calculateCompletenessScore(doc *types.PRDDocument) float64 {
	requiredSections := []types.SectionType{
		types.SectionTypeOverview,
		types.SectionTypeObjectives,
		types.SectionTypeRequirements,
		types.SectionTypeFunctional,
		types.SectionTypeAcceptance,
	}

	presentSections := 0
	sectionsByType := doc.Content.Structure.SectionsByType

	for _, reqType := range requiredSections {
		if count, exists := sectionsByType[reqType]; exists && count > 0 {
			presentSections++
		}
	}

	baseScore := float64(presentSections) / float64(len(requiredSections))

	// Bonus points for optional but valuable sections
	bonusSections := []types.SectionType{
		types.SectionTypeNonFunctional,
		types.SectionTypeTechnical,
		types.SectionTypeUserStories,
		types.SectionTypeRisks,
		types.SectionTypeTimeline,
	}

	bonusPoints := 0
	for _, bonusType := range bonusSections {
		if count, exists := sectionsByType[bonusType]; exists && count > 0 {
			bonusPoints++
		}
	}

	bonusScore := float64(bonusPoints) / float64(len(bonusSections)) * 0.2 // Max 20% bonus

	finalScore := baseScore + bonusScore
	if finalScore > 1.0 {
		finalScore = 1.0
	}

	return finalScore
}

// calculateClarityScore calculates how clear and understandable the PRD is
func (a *Analyzer) calculateClarityScore(doc *types.PRDDocument) float64 {
	content := doc.Content.Raw
	if content == "" {
		return 0.0
	}

	score := 1.0 // Start with perfect score and deduct

	// Factor 1: Average sentence length (shorter is generally clearer)
	avgSentenceLength := a.calculateAverageSentenceLength(content)
	if avgSentenceLength > 25 {
		score -= 0.2
	} else if avgSentenceLength > 20 {
		score -= 0.1
	}

	// Factor 2: Use of jargon and technical terms without explanation
	unexplainedJargon := a.countUnexplainedJargon(content)
	if unexplainedJargon > 10 {
		score -= 0.3
	} else if unexplainedJargon > 5 {
		score -= 0.2
	}

	// Factor 3: Readability (simplified calculation)
	readabilityPenalty := a.calculateReadabilityPenalty(content)
	score -= readabilityPenalty

	// Factor 4: Structure clarity (well-organized sections)
	if doc.Content.Structure.TotalSections == 0 {
		score -= 0.2
	} else if doc.Content.Structure.MaxDepth > 4 {
		score -= 0.1 // Too deeply nested
	}

	if score < 0 {
		score = 0
	}

	return score
}

// calculateStructureScore calculates how well-structured the document is
func (a *Analyzer) calculateStructureScore(doc *types.PRDDocument) float64 {
	structure := doc.Content.Structure
	score := 0.0

	// Factor 1: Has logical section organization
	if structure.TotalSections > 0 {
		score += 0.3
	}

	// Factor 2: Appropriate depth (not too shallow, not too deep)
	if structure.MaxDepth >= 2 && structure.MaxDepth <= 4 {
		score += 0.2
	} else if structure.MaxDepth == 1 || structure.MaxDepth == 5 {
		score += 0.1
	}

	// Factor 3: Has table of contents
	if structure.HasTOC {
		score += 0.1
	}

	// Factor 4: Uses visual elements appropriately
	visualScore := 0.0
	if structure.HasImages {
		visualScore += 0.05
	}
	if structure.HasTables {
		visualScore += 0.05
	}
	if structure.HasDiagrams {
		visualScore += 0.1
	}
	if visualScore > 0.2 {
		visualScore = 0.2
	}
	score += visualScore

	// Factor 5: Section type diversity
	uniqueTypes := len(structure.SectionsByType)
	if uniqueTypes >= 5 {
		score += 0.2
	} else if uniqueTypes >= 3 {
		score += 0.1
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}

// calculateReadabilityScore calculates the readability score
func (a *Analyzer) calculateReadabilityScore(doc *types.PRDDocument) float64 {
	content := doc.Content.Raw
	if content == "" {
		return 0.0
	}

	// Simplified readability calculation
	words := strings.Fields(content)
	sentences := strings.Split(content, ".")
	syllables := a.countSyllables(content)

	if len(sentences) == 0 || len(words) == 0 {
		return 0.0
	}

	// Flesch Reading Ease approximation
	avgWordsPerSentence := float64(len(words)) / float64(len(sentences))
	avgSyllablesPerWord := float64(syllables) / float64(len(words))

	fleschScore := 206.835 - (1.015 * avgWordsPerSentence) - (84.6 * avgSyllablesPerWord)

	// Normalize to 0-1 scale (Flesch scores typically range from 0-100)
	normalizedScore := fleschScore / 100.0
	if normalizedScore > 1.0 {
		normalizedScore = 1.0
	}
	if normalizedScore < 0.0 {
		normalizedScore = 0.0
	}

	return normalizedScore
}

// calculateQualityScore calculates the overall quality score
func (a *Analyzer) calculateQualityScore(analysis *types.PRDAnalysis) float64 {
	// Weighted average of different scores
	weights := map[string]float64{
		"completeness": 0.3,
		"clarity":      0.25,
		"structure":    0.2,
		"readability":  0.15,
		"complexity":   0.1, // Lower complexity can be better for some PRDs
	}

	// Complexity score needs to be inverted for quality (lower complexity = higher quality for some aspects)
	adjustedComplexity := 1.0 - analysis.ComplexityScore
	if adjustedComplexity < 0 {
		adjustedComplexity = 0
	}

	score := (analysis.CompletenessScore * weights["completeness"]) +
		(analysis.ClarityScore * weights["clarity"]) +
		(analysis.StructureScore * weights["structure"]) +
		(analysis.ReadabilityScore * weights["readability"]) +
		(adjustedComplexity * weights["complexity"])

	return score
}

// identifyIssues identifies potential issues in the document
func (a *Analyzer) identifyIssues(doc *types.PRDDocument, analysis *types.PRDAnalysis) {
	// Check for low scores and create issues
	if analysis.CompletenessScore < a.config.MinCompletenessScore {
		analysis.Issues = append(analysis.Issues, types.AnalysisIssue{
			Type:       types.IssueTypeCompleteness,
			Severity:   types.SeverityHigh,
			Message:    "Document appears to be incomplete",
			Location:   "overall",
			Suggestion: "Add missing required sections",
		})
	}

	if analysis.ClarityScore < a.config.MinClarityScore {
		analysis.Issues = append(analysis.Issues, types.AnalysisIssue{
			Type:       types.IssueTypeClarity,
			Severity:   types.SeverityMedium,
			Message:    "Document clarity could be improved",
			Location:   "overall",
			Suggestion: "Simplify language and add explanations for technical terms",
		})
	}

	// Check for structural issues
	if doc.Content.Structure.MaxDepth > 5 {
		analysis.Issues = append(analysis.Issues, types.AnalysisIssue{
			Type:       types.IssueTypeStructure,
			Severity:   types.SeverityMedium,
			Message:    "Document structure is too deeply nested",
			Location:   "overall",
			Suggestion: "Flatten the structure or break into multiple documents",
		})
	}

	if doc.Content.Structure.TotalSections == 0 {
		analysis.Issues = append(analysis.Issues, types.AnalysisIssue{
			Type:       types.IssueTypeStructure,
			Severity:   types.SeverityHigh,
			Message:    "Document lacks clear section structure",
			Location:   "overall",
			Suggestion: "Add clear headings and organize content into sections",
		})
	}
}

// extractKeyInformation extracts key concepts and technical terms
func (a *Analyzer) extractKeyInformation(doc *types.PRDDocument, analysis *types.PRDAnalysis) {
	content := strings.ToLower(doc.Content.Raw)

	// Extract technical terms (simplified approach)
	techTerms := []string{
		"api", "database", "microservice", "authentication", "authorization",
		"scalability", "performance", "security", "integration", "deployment",
		"architecture", "framework", "library", "protocol", "encryption",
		"cache", "queue", "async", "sync", "rest", "graphql", "websocket",
	}

	foundTerms := []string{}
	for _, term := range techTerms {
		if strings.Contains(content, term) {
			foundTerms = append(foundTerms, term)
		}
	}
	analysis.TechnicalTerms = foundTerms

	// Extract key concepts (business terms)
	concepts := []string{
		"user", "customer", "business", "revenue", "cost", "roi", "kpi",
		"feature", "requirement", "goal", "objective", "stakeholder",
		"timeline", "milestone", "risk", "assumption", "constraint",
	}

	foundConcepts := []string{}
	for _, concept := range concepts {
		if strings.Contains(content, concept) {
			foundConcepts = append(foundConcepts, concept)
		}
	}
	analysis.KeyConcepts = foundConcepts
}

// generateRecommendations generates recommendations for improvement
func (a *Analyzer) generateRecommendations(doc *types.PRDDocument, analysis *types.PRDAnalysis) {
	recommendations := []string{}

	// Based on completeness score
	if analysis.CompletenessScore < 0.7 {
		recommendations = append(recommendations, "Add missing sections to improve completeness")
	}

	// Based on clarity score
	if analysis.ClarityScore < 0.6 {
		recommendations = append(recommendations, "Improve document clarity by simplifying language")
	}

	// Based on structure
	if doc.Content.Structure.TotalSections < 3 {
		recommendations = append(recommendations, "Add more detailed sections for better organization")
	}

	// Based on content analysis
	if len(analysis.TechnicalTerms) > 0 && len(analysis.KeyConcepts) == 0 {
		recommendations = append(recommendations, "Add business context to balance technical details")
	}

	if len(analysis.KeyConcepts) > 0 && len(analysis.TechnicalTerms) == 0 {
		recommendations = append(recommendations, "Add technical implementation details")
	}

	analysis.Recommendations = recommendations
}

// identifyMissingElements identifies missing elements
func (a *Analyzer) identifyMissingElements(doc *types.PRDDocument, analysis *types.PRDAnalysis) {
	missing := []string{}
	sectionsByType := doc.Content.Structure.SectionsByType

	requiredElements := map[types.SectionType]string{
		types.SectionTypeOverview:     "Project Overview",
		types.SectionTypeObjectives:   "Objectives and Goals",
		types.SectionTypeRequirements: "Requirements",
		types.SectionTypeFunctional:   "Functional Requirements",
		types.SectionTypeAcceptance:   "Acceptance Criteria",
	}

	for sectionType, description := range requiredElements {
		if count, exists := sectionsByType[sectionType]; !exists || count == 0 {
			missing = append(missing, description)
		}
	}

	analysis.MissingElements = missing
}

// Helper functions

func (a *Analyzer) calculateAverageSentenceLength(content string) float64 {
	sentences := regexp.MustCompile(`[.!?]+`).Split(content, -1)
	words := strings.Fields(content)

	if len(sentences) == 0 {
		return 0
	}

	return float64(len(words)) / float64(len(sentences))
}

func (a *Analyzer) countTechnicalTerms(content string) int {
	techPatterns := []string{
		`\bAPI\b`, `\bREST\b`, `\bJSON\b`, `\bXML\b`, `\bHTTP\b`, `\bHTTPS\b`,
		`\bSQL\b`, `\bNoSQL\b`, `\bdatabase\b`, `\bmicroservice\b`,
		`\bauthentication\b`, `\bauthorization\b`, `\bOAuth\b`, `\bJWT\b`,
	}

	count := 0
	contentLower := strings.ToLower(content)

	for _, pattern := range techPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllString(contentLower, -1)
		count += len(matches)
	}

	return count
}

func (a *Analyzer) countUnexplainedJargon(content string) int {
	// This is a simplified implementation
	// In a real system, this would use NLP to identify technical terms
	// and check if they're explained in the document
	jargonTerms := []string{
		"scalability", "microservices", "middleware", "framework",
		"architecture", "deployment", "containerization", "orchestration",
	}

	count := 0
	contentLower := strings.ToLower(content)

	for _, term := range jargonTerms {
		if strings.Contains(contentLower, term) {
			// Check if it's explained (very basic check)
			explained := strings.Contains(contentLower, term+" is") ||
				strings.Contains(contentLower, term+" means") ||
				strings.Contains(contentLower, term+":") ||
				strings.Contains(contentLower, term+" refers to")

			if !explained {
				count++
			}
		}
	}

	return count
}

func (a *Analyzer) calculateReadabilityPenalty(content string) float64 {
	// Simplified readability penalty calculation
	penalty := 0.0

	// Penalty for very long paragraphs
	paragraphs := strings.Split(content, "\n\n")
	for _, paragraph := range paragraphs {
		words := strings.Fields(paragraph)
		if len(words) > 100 {
			penalty += 0.1
		}
	}

	// Penalty for excessive use of passive voice (simplified detection)
	passiveIndicators := []string{"was", "were", "been", "being"}
	totalWords := len(strings.Fields(content))
	passiveCount := 0

	for _, indicator := range passiveIndicators {
		passiveCount += strings.Count(strings.ToLower(content), " "+indicator+" ")
	}

	if totalWords > 0 {
		passiveRatio := float64(passiveCount) / float64(totalWords)
		if passiveRatio > 0.1 {
			penalty += 0.2
		}
	}

	if penalty > 0.5 {
		penalty = 0.5
	}

	return penalty
}

func (a *Analyzer) countSyllables(content string) int {
	// Very simplified syllable counting
	words := strings.Fields(strings.ToLower(content))
	totalSyllables := 0

	vowels := "aeiouy"
	for _, word := range words {
		syllables := 0
		prevWasVowel := false

		for _, char := range word {
			isVowel := strings.ContainsRune(vowels, char)
			if isVowel && !prevWasVowel {
				syllables++
			}
			prevWasVowel = isVowel
		}

		// Every word has at least one syllable
		if syllables == 0 {
			syllables = 1
		}

		totalSyllables += syllables
	}

	return totalSyllables
}
