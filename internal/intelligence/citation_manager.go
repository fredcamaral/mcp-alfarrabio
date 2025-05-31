package intelligence

import (
	"context"
	"crypto/sha256"
	"fmt"
	"mcp-memory/pkg/types"
	"sort"
	"strings"
	"time"
)

// CitationManager manages citations for AI responses
type CitationManager struct {
	storage StorageInterface
	config  *CitationConfig
}

// CitationConfig configures citation behavior
type CitationConfig struct {
	CitationStyle       string                      `json:"citation_style"`       // "apa", "mla", "chicago", "simple"
	IncludeTimestamps   bool                        `json:"include_timestamps"`
	IncludeRepository   bool                        `json:"include_repository"`
	IncludeConfidence   bool                        `json:"include_confidence"`
	MaxCitationsPerResponse int                     `json:"max_citations_per_response"`
	MinConfidenceForCitation float64               `json:"min_confidence_for_citation"`
	CustomFormats       map[string]CitationFormat  `json:"custom_formats"`
	GroupSimilarSources bool                       `json:"group_similar_sources"`
}

// CitationFormat defines how to format citations
type CitationFormat struct {
	Template     string            `json:"template"`     // Template with placeholders
	Fields       []string          `json:"fields"`       // Required fields
	Separator    string            `json:"separator"`    // Separator for multiple citations
	Prefix       string            `json:"prefix"`       // Prefix for citation list
	Suffix       string            `json:"suffix"`       // Suffix for citation list
}

// Citation represents a single citation
type Citation struct {
	ID          string                 `json:"id"`
	ChunkID     string                 `json:"chunk_id"`
	Type        types.ChunkType        `json:"type"`
	Repository  string                 `json:"repository"`
	Timestamp   time.Time              `json:"timestamp"`
	Summary     string                 `json:"summary"`
	Confidence  float64                `json:"confidence"`
	Relevance   float64                `json:"relevance"`
	UsageCount  int                    `json:"usage_count"`
	Context     string                 `json:"context"`     // Quoted or relevant portion
	Metadata    map[string]interface{} `json:"metadata"`
	FormattedText string               `json:"formatted_text"`
}

// CitationGroup represents grouped citations
type CitationGroup struct {
	GroupID     string      `json:"group_id"`
	GroupType   string      `json:"group_type"`   // "repository", "session", "type", "topic"
	GroupName   string      `json:"group_name"`
	Citations   []Citation  `json:"citations"`
	Summary     string      `json:"summary"`
	Weight      float64     `json:"weight"`       // Combined weight of citations
}

// ResponseCitations represents citations for a complete AI response
type ResponseCitations struct {
	ResponseID        string          `json:"response_id"`
	Query             string          `json:"query"`
	TotalCitations    int             `json:"total_citations"`
	Groups            []CitationGroup `json:"groups,omitempty"`
	IndividualCitations []Citation    `json:"individual_citations,omitempty"`
	FormattedBibliography string      `json:"formatted_bibliography"`
	InlineCitations   map[string]string `json:"inline_citations"` // text_hash -> citation_reference
	GeneratedAt       time.Time       `json:"generated_at"`
	Style             string          `json:"style"`
}

// NewCitationManager creates a new citation manager
func NewCitationManager(storage StorageInterface) *CitationManager {
	return &CitationManager{
		storage: storage,
		config:  DefaultCitationConfig(),
	}
}

// DefaultCitationConfig returns sensible defaults
func DefaultCitationConfig() *CitationConfig {
	return &CitationConfig{
		CitationStyle:            "simple",
		IncludeTimestamps:        true,
		IncludeRepository:        true,
		IncludeConfidence:        false,
		MaxCitationsPerResponse:  10,
		MinConfidenceForCitation: 0.3,
		GroupSimilarSources:      true,
		CustomFormats: map[string]CitationFormat{
			"simple": {
				Template:  "[{id}] {type} from {repository} ({timestamp})",
				Fields:    []string{"id", "type", "repository", "timestamp"},
				Separator: "\n",
				Prefix:    "Sources:\n",
				Suffix:    "",
			},
			"apa": {
				Template:  "{repository}. ({timestamp}). {summary}. Memory System.",
				Fields:    []string{"repository", "timestamp", "summary"},
				Separator: "\n",
				Prefix:    "References:\n",
				Suffix:    "",
			},
			"inline": {
				Template:  "[{id}]",
				Fields:    []string{"id"},
				Separator: ", ",
				Prefix:    "",
				Suffix:    "",
			},
		},
	}
}

// GenerateCitations creates citations for search results
func (cm *CitationManager) GenerateCitations(ctx context.Context, results []types.SearchResult, query string) (*ResponseCitations, error) {
	if len(results) == 0 {
		return &ResponseCitations{
			ResponseID:      cm.generateResponseID(query),
			Query:           query,
			TotalCitations:  0,
			GeneratedAt:     time.Now(),
			Style:           cm.config.CitationStyle,
		}, nil
	}

	// Filter results by confidence
	filteredResults := cm.filterByConfidence(results)

	// Generate individual citations
	citations := make([]Citation, 0, len(filteredResults))
	for i, result := range filteredResults {
		citation := cm.createCitation(result, i+1)
		citations = append(citations, citation)
	}

	// Group citations if enabled
	var groups []CitationGroup
	if cm.config.GroupSimilarSources {
		groups = cm.groupCitations(citations)
	}

	// Generate formatted bibliography
	bibliography := cm.formatBibliography(citations, groups)

	// Generate inline citation map
	inlineCitations := cm.generateInlineCitationMap(citations)

	responseCitations := &ResponseCitations{
		ResponseID:            cm.generateResponseID(query),
		Query:                 query,
		TotalCitations:        len(citations),
		Groups:                groups,
		IndividualCitations:   citations,
		FormattedBibliography: bibliography,
		InlineCitations:       inlineCitations,
		GeneratedAt:           time.Now(),
		Style:                 cm.config.CitationStyle,
	}

	return responseCitations, nil
}

// CreateInlineReference creates an inline citation reference for text
func (cm *CitationManager) CreateInlineReference(text string, citations *ResponseCitations) string {
	// Generate hash for the text
	textHash := cm.generateTextHash(text)
	
	// Find matching citations
	matchingCitations := cm.findMatchingCitations(text, citations.IndividualCitations)
	
	if len(matchingCitations) == 0 {
		return text
	}

	// Generate inline reference
	inlineRef := cm.formatInlineReference(matchingCitations)
	
	// Store in inline citations map
	citations.InlineCitations[textHash] = inlineRef
	
	return text + " " + inlineRef
}

// UpdateCitationUsage updates usage statistics for citations
func (cm *CitationManager) UpdateCitationUsage(ctx context.Context, citationIDs []string) error {
	for _, citationID := range citationIDs {
		// In a real implementation, we would track usage in the database
		// For now, we'll just log it
		// This could update chunk metadata with usage statistics
		_ = citationID // Mark as used for now
	}
	return nil
}

// Private helper methods

func (cm *CitationManager) filterByConfidence(results []types.SearchResult) []types.SearchResult {
	filtered := make([]types.SearchResult, 0)
	
	for _, result := range results {
		confidence := 0.5 // Default confidence
		if result.Chunk.Metadata.Confidence != nil {
			confidence = result.Chunk.Metadata.Confidence.Score
		}
		
		if confidence >= cm.config.MinConfidenceForCitation {
			filtered = append(filtered, result)
		}
	}

	// Limit to max citations
	if len(filtered) > cm.config.MaxCitationsPerResponse {
		filtered = filtered[:cm.config.MaxCitationsPerResponse]
	}

	return filtered
}

func (cm *CitationManager) createCitation(result types.SearchResult, index int) Citation {
	chunk := result.Chunk
	
	confidence := 0.5
	if chunk.Metadata.Confidence != nil {
		confidence = chunk.Metadata.Confidence.Score
	}

	// Generate citation ID (will be used below)
	citationID := fmt.Sprintf("%d", index)
	
	// Extract context (first 200 chars of content)
	context := chunk.Content
	if len(context) > 200 {
		context = context[:200] + "..."
	}

	citation := Citation{
		ID:          citationID,
		ChunkID:     chunk.ID,
		Type:        chunk.Type,
		Repository:  chunk.Metadata.Repository,
		Timestamp:   chunk.Timestamp,
		Summary:     chunk.Summary,
		Confidence:  confidence,
		Relevance:   result.Score,
		UsageCount:  0,
		Context:     context,
		Metadata:    make(map[string]interface{}),
	}

	// Add additional metadata
	citation.Metadata["session_id"] = chunk.SessionID
	citation.Metadata["outcome"] = string(chunk.Metadata.Outcome)
	if len(chunk.Metadata.Tags) > 0 {
		citation.Metadata["tags"] = chunk.Metadata.Tags
	}

	// Format the citation text
	citation.FormattedText = cm.formatCitationText(citation)

	return citation
}

func (cm *CitationManager) groupCitations(citations []Citation) []CitationGroup {
	groups := make(map[string]*CitationGroup)

	for _, citation := range citations {
		// Group by repository
		repoKey := "repo_" + citation.Repository
		if _, exists := groups[repoKey]; !exists {
			groups[repoKey] = &CitationGroup{
				GroupID:   repoKey,
				GroupType: "repository",
				GroupName: citation.Repository,
				Citations: []Citation{},
				Weight:    0,
			}
		}
		groups[repoKey].Citations = append(groups[repoKey].Citations, citation)
		groups[repoKey].Weight += citation.Relevance

		// Group by type
		typeKey := "type_" + string(citation.Type)
		if _, exists := groups[typeKey]; !exists {
			groups[typeKey] = &CitationGroup{
				GroupID:   typeKey,
				GroupType: "type",
				GroupName: string(citation.Type),
				Citations: []Citation{},
				Weight:    0,
			}
		}
		groups[typeKey].Citations = append(groups[typeKey].Citations, citation)
		groups[typeKey].Weight += citation.Relevance
	}

	// Convert to slice and sort by weight
	result := make([]CitationGroup, 0, len(groups))
	for _, group := range groups {
		group.Summary = cm.generateGroupSummary(group)
		result = append(result, *group)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Weight > result[j].Weight
	})

	return result
}

func (cm *CitationManager) formatBibliography(citations []Citation, groups []CitationGroup) string {
	format := cm.config.CustomFormats[cm.config.CitationStyle]
	
	var bibliography strings.Builder
	bibliography.WriteString(format.Prefix)

	if cm.config.GroupSimilarSources && len(groups) > 0 {
		// Format grouped bibliography
		for i, group := range groups {
			if i > 0 {
				bibliography.WriteString("\n")
			}
			bibliography.WriteString(fmt.Sprintf("\n%s:\n", group.GroupName))
			
			for _, citation := range group.Citations {
				bibliography.WriteString("  ")
				bibliography.WriteString(citation.FormattedText)
				bibliography.WriteString(format.Separator)
			}
		}
	} else {
		// Format individual bibliography
		for i, citation := range citations {
			if i > 0 {
				bibliography.WriteString(format.Separator)
			}
			bibliography.WriteString(citation.FormattedText)
		}
	}

	bibliography.WriteString(format.Suffix)
	return bibliography.String()
}

func (cm *CitationManager) generateInlineCitationMap(citations []Citation) map[string]string {
	inlineMap := make(map[string]string)
	
	for _, citation := range citations {
		// Create inline reference for this citation
		inlineFormat := cm.config.CustomFormats["inline"]
		inlineRef := cm.replacePlaceholders(inlineFormat.Template, citation)
		
		// Map citation ID to inline reference
		inlineMap[citation.ChunkID] = inlineRef
	}
	
	return inlineMap
}

func (cm *CitationManager) formatCitationText(citation Citation) string {
	format := cm.config.CustomFormats[cm.config.CitationStyle]
	return cm.replacePlaceholders(format.Template, citation)
}

func (cm *CitationManager) replacePlaceholders(template string, citation Citation) string {
	text := template
	
	replacements := map[string]string{
		"{id}":         citation.ID,
		"{chunk_id}":   citation.ChunkID,
		"{type}":       string(citation.Type),
		"{repository}": citation.Repository,
		"{timestamp}":  citation.Timestamp.Format("2006-01-02"),
		"{summary}":    citation.Summary,
		"{confidence}": fmt.Sprintf("%.2f", citation.Confidence),
		"{relevance}":  fmt.Sprintf("%.2f", citation.Relevance),
		"{context}":    citation.Context,
	}

	for placeholder, value := range replacements {
		text = strings.ReplaceAll(text, placeholder, value)
	}

	return text
}

func (cm *CitationManager) generateResponseID(query string) string {
	hash := sha256.Sum256([]byte(query + time.Now().Format(time.RFC3339)))
	return fmt.Sprintf("resp_%x", hash[:8])
}

func (cm *CitationManager) generateTextHash(text string) string {
	hash := sha256.Sum256([]byte(text))
	return fmt.Sprintf("%x", hash[:8])
}

func (cm *CitationManager) findMatchingCitations(text string, citations []Citation) []Citation {
	matching := make([]Citation, 0)
	textLower := strings.ToLower(text)
	
	for _, citation := range citations {
		// Check if citation content is referenced in the text
		if strings.Contains(textLower, strings.ToLower(citation.Summary)) ||
		   strings.Contains(textLower, strings.ToLower(citation.Context)) {
			matching = append(matching, citation)
		}
	}
	
	return matching
}

func (cm *CitationManager) formatInlineReference(citations []Citation) string {
	if len(citations) == 0 {
		return ""
	}
	
	inlineFormat := cm.config.CustomFormats["inline"]
	refs := make([]string, len(citations))
	
	for i, citation := range citations {
		refs[i] = cm.replacePlaceholders(inlineFormat.Template, citation)
	}
	
	return strings.Join(refs, inlineFormat.Separator)
}

func (cm *CitationManager) generateGroupSummary(group *CitationGroup) string {
	if len(group.Citations) == 0 {
		return ""
	}
	
	switch group.GroupType {
	case "repository":
		return fmt.Sprintf("%d sources from %s", len(group.Citations), group.GroupName)
	case "type":
		return fmt.Sprintf("%d %s entries", len(group.Citations), group.GroupName)
	default:
		return fmt.Sprintf("%d related sources", len(group.Citations))
	}
}

// SetConfig updates the citation configuration
func (cm *CitationManager) SetConfig(config *CitationConfig) {
	cm.config = config
}

// GetConfig returns the current citation configuration
func (cm *CitationManager) GetConfig() *CitationConfig {
	return cm.config
}