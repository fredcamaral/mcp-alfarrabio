// Package types provides extended types for internal use
package types

import (
	"lerian-mcp-memory/pkg/types"
	"time"
)

// ExtendedConversationChunk extends the base ConversationChunk with additional fields
// This is used internally where we need more dynamic fields
type ExtendedConversationChunk struct {
	types.ConversationChunk

	// Additional fields not in the base type
	Repository         string   `json:"repository,omitempty"`
	Branch             string   `json:"branch,omitempty"`
	Concepts           []string `json:"concepts,omitempty"`
	Entities           []string `json:"entities,omitempty"`
	DecisionOutcome    string   `json:"decision_outcome,omitempty"`
	DecisionRationale  string   `json:"decision_rationale,omitempty"`
	DifficultyLevel    string   `json:"difficulty_level,omitempty"`
	ProblemDescription string   `json:"problem_description,omitempty"`
	SolutionApproach   string   `json:"solution_approach,omitempty"`
	Outcome            string   `json:"outcome,omitempty"`
	LessonsLearned     string   `json:"lessons_learned,omitempty"`
	NextSteps          string   `json:"next_steps,omitempty"`

	// Dynamic metadata
	ExtendedMetadata map[string]interface{} `json:"extended_metadata,omitempty"`
}

// ToBase converts ExtendedConversationChunk to base ConversationChunk
func (e *ExtendedConversationChunk) ToBase() types.ConversationChunk {
	chunk := e.ConversationChunk

	// Copy repository to metadata if not already there
	if e.Repository != "" && chunk.Metadata.Repository == "" {
		chunk.Metadata.Repository = e.Repository
	}

	// Copy branch to metadata if not already there
	if e.Branch != "" && chunk.Metadata.Branch == "" {
		chunk.Metadata.Branch = e.Branch
	}

	// Merge tags with concepts
	if len(e.Concepts) > 0 {
		// Deduplicate
		tagMap := make(map[string]bool)
		for _, tag := range chunk.Metadata.Tags {
			tagMap[tag] = true
		}
		for _, concept := range e.Concepts {
			if !tagMap[concept] {
				chunk.Metadata.Tags = append(chunk.Metadata.Tags, concept)
			}
		}
	}

	return chunk
}

// FromBase creates an ExtendedConversationChunk from a base chunk
func FromBase(chunk *types.ConversationChunk) *ExtendedConversationChunk {
	extended := &ExtendedConversationChunk{
		ConversationChunk: *chunk,
		ExtendedMetadata:  make(map[string]interface{}),
	}

	// Copy metadata fields to extended fields
	extended.Repository = chunk.Metadata.Repository
	extended.Branch = chunk.Metadata.Branch

	// Extract concepts from tags (simplified)
	extended.Concepts = append(extended.Concepts, chunk.Metadata.Tags...)

	return extended
}

// MemoryQuery wraps the base types.MemoryQuery
type MemoryQuery struct {
	Query             string    `json:"query"`
	Repository        string    `json:"repository,omitempty"`
	Types             []string  `json:"types,omitempty"`
	Tags              []string  `json:"tags,omitempty"`
	Limit             int       `json:"limit,omitempty"`
	MinRelevanceScore float64   `json:"min_relevance,omitempty"`
	StartTime         time.Time `json:"start_time,omitempty"`
	EndTime           time.Time `json:"end_time,omitempty"`
}

// ScoredChunk wraps a chunk with score
type ScoredChunk struct {
	Chunk ExtendedConversationChunk `json:"chunk"`
	Score float64                   `json:"score"`
}
