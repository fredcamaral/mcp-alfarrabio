// Package graphql provides GraphQL API for memory operations
package graphql

import (
	"fmt"
	"mcp-memory/internal/di"

	"github.com/graphql-go/graphql"
)

// Schema holds the GraphQL schema
type Schema struct {
	schema    graphql.Schema
	container *di.Container
}

// NewSchema creates a new GraphQL schema
func NewSchema(container *di.Container) (*Schema, error) {
	// Define ConversationChunk type
	chunkType := graphql.NewObject(graphql.ObjectConfig{
		Name: "ConversationChunk",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.String,
			},
			"sessionId": &graphql.Field{
				Type: graphql.String,
			},
			"repository": &graphql.Field{
				Type: graphql.String,
			},
			"branch": &graphql.Field{
				Type: graphql.String,
			},
			"timestamp": &graphql.Field{
				Type: graphql.DateTime,
			},
			"content": &graphql.Field{
				Type: graphql.String,
			},
			"summary": &graphql.Field{
				Type: graphql.String,
			},
			"type": &graphql.Field{
				Type: graphql.String,
			},
			"tags": &graphql.Field{
				Type: graphql.NewList(graphql.String),
			},
			"toolsUsed": &graphql.Field{
				Type: graphql.NewList(graphql.String),
			},
			"filePaths": &graphql.Field{
				Type: graphql.NewList(graphql.String),
			},
			"concepts": &graphql.Field{
				Type: graphql.NewList(graphql.String),
			},
			"entities": &graphql.Field{
				Type: graphql.NewList(graphql.String),
			},
			"decisionOutcome": &graphql.Field{
				Type: graphql.String,
			},
			"decisionRationale": &graphql.Field{
				Type: graphql.String,
			},
			"difficultyLevel": &graphql.Field{
				Type: graphql.String,
			},
			"problemDescription": &graphql.Field{
				Type: graphql.String,
			},
			"solutionApproach": &graphql.Field{
				Type: graphql.String,
			},
			"outcome": &graphql.Field{
				Type: graphql.String,
			},
			"lessonsLearned": &graphql.Field{
				Type: graphql.String,
			},
			"nextSteps": &graphql.Field{
				Type: graphql.String,
			},
		},
	})

	// Define ScoredChunk type
	scoredChunkType := graphql.NewObject(graphql.ObjectConfig{
		Name: "ScoredChunk",
		Fields: graphql.Fields{
			"chunk": &graphql.Field{
				Type: chunkType,
			},
			"score": &graphql.Field{
				Type: graphql.Float,
			},
		},
	})

	// Define SearchResults type
	searchResultsType := graphql.NewObject(graphql.ObjectConfig{
		Name: "SearchResults",
		Fields: graphql.Fields{
			"chunks": &graphql.Field{
				Type: graphql.NewList(scoredChunkType),
			},
		},
	})

	// Define Pattern type
	patternType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Pattern",
		Fields: graphql.Fields{
			"name": &graphql.Field{
				Type: graphql.String,
			},
			"description": &graphql.Field{
				Type: graphql.String,
			},
			"occurrences": &graphql.Field{
				Type: graphql.Int,
			},
			"confidence": &graphql.Field{
				Type: graphql.Float,
			},
			"lastSeen": &graphql.Field{
				Type: graphql.DateTime,
			},
			"examples": &graphql.Field{
				Type: graphql.NewList(graphql.String),
			},
		},
	})

	// Define ContextSuggestion type
	contextSuggestionType := graphql.NewObject(graphql.ObjectConfig{
		Name: "ContextSuggestion",
		Fields: graphql.Fields{
			"relevantChunks": &graphql.Field{
				Type: graphql.NewList(scoredChunkType),
			},
			"suggestedTasks": &graphql.Field{
				Type: graphql.NewList(graphql.String),
			},
			"relatedConcepts": &graphql.Field{
				Type: graphql.NewList(graphql.String),
			},
			"potentialIssues": &graphql.Field{
				Type: graphql.NewList(graphql.String),
			},
		},
	})

	// Define MemoryQuery input type
	memoryQueryInputType := graphql.NewInputObject(graphql.InputObjectConfig{
		Name: "MemoryQueryInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"query": &graphql.InputObjectFieldConfig{
				Type: graphql.String,
			},
			"repository": &graphql.InputObjectFieldConfig{
				Type: graphql.String,
			},
			"types": &graphql.InputObjectFieldConfig{
				Type: graphql.NewList(graphql.String),
			},
			"tags": &graphql.InputObjectFieldConfig{
				Type: graphql.NewList(graphql.String),
			},
			"limit": &graphql.InputObjectFieldConfig{
				Type:         graphql.Int,
				DefaultValue: 10,
			},
			"minRelevanceScore": &graphql.InputObjectFieldConfig{
				Type:         graphql.Float,
				DefaultValue: 0.7,
			},
			"recency": &graphql.InputObjectFieldConfig{
				Type:         graphql.String,
				DefaultValue: "recent",
			},
		},
	})

	// Define StoreChunkInput type
	storeChunkInputType := graphql.NewInputObject(graphql.InputObjectConfig{
		Name: "StoreChunkInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"content": &graphql.InputObjectFieldConfig{
				Type: graphql.NewNonNull(graphql.String),
			},
			"sessionId": &graphql.InputObjectFieldConfig{
				Type: graphql.NewNonNull(graphql.String),
			},
			"repository": &graphql.InputObjectFieldConfig{
				Type: graphql.String,
			},
			"branch": &graphql.InputObjectFieldConfig{
				Type: graphql.String,
			},
			"tags": &graphql.InputObjectFieldConfig{
				Type: graphql.NewList(graphql.String),
			},
			"toolsUsed": &graphql.InputObjectFieldConfig{
				Type: graphql.NewList(graphql.String),
			},
			"filesModified": &graphql.InputObjectFieldConfig{
				Type: graphql.NewList(graphql.String),
			},
		},
	})

	// Define StoreDecisionInput type
	storeDecisionInputType := graphql.NewInputObject(graphql.InputObjectConfig{
		Name: "StoreDecisionInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"decision": &graphql.InputObjectFieldConfig{
				Type: graphql.NewNonNull(graphql.String),
			},
			"rationale": &graphql.InputObjectFieldConfig{
				Type: graphql.NewNonNull(graphql.String),
			},
			"sessionId": &graphql.InputObjectFieldConfig{
				Type: graphql.NewNonNull(graphql.String),
			},
			"repository": &graphql.InputObjectFieldConfig{
				Type: graphql.String,
			},
			"context": &graphql.InputObjectFieldConfig{
				Type: graphql.String,
			},
		},
	})

	// Create root query
	rootQuery := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"search": &graphql.Field{
				Type: searchResultsType,
				Args: graphql.FieldConfigArgument{
					"input": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(memoryQueryInputType),
					},
				},
				Resolve: s.searchResolver(container),
			},
			"getChunk": &graphql.Field{
				Type: chunkType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: s.getChunkResolver(container),
			},
			"listChunks": &graphql.Field{
				Type: graphql.NewList(chunkType),
				Args: graphql.FieldConfigArgument{
					"repository": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"limit": &graphql.ArgumentConfig{
						Type:         graphql.Int,
						DefaultValue: 100,
					},
					"offset": &graphql.ArgumentConfig{
						Type:         graphql.Int,
						DefaultValue: 0,
					},
				},
				Resolve: s.listChunksResolver(container),
			},
			"getPatterns": &graphql.Field{
				Type: graphql.NewList(patternType),
				Args: graphql.FieldConfigArgument{
					"repository": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"timeframe": &graphql.ArgumentConfig{
						Type:         graphql.String,
						DefaultValue: "month",
					},
				},
				Resolve: s.getPatternsResolver(container),
			},
			"suggestRelated": &graphql.Field{
				Type: contextSuggestionType,
				Args: graphql.FieldConfigArgument{
					"currentContext": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"sessionId": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"repository": &graphql.ArgumentConfig{
						Type: graphql.String,
					},
					"includePatterns": &graphql.ArgumentConfig{
						Type:         graphql.Boolean,
						DefaultValue: true,
					},
					"maxSuggestions": &graphql.ArgumentConfig{
						Type:         graphql.Int,
						DefaultValue: 5,
					},
				},
				Resolve: s.suggestRelatedResolver(container),
			},
			"findSimilar": &graphql.Field{
				Type: graphql.NewList(chunkType),
				Args: graphql.FieldConfigArgument{
					"problem": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"repository": &graphql.ArgumentConfig{
						Type: graphql.String,
					},
					"limit": &graphql.ArgumentConfig{
						Type:         graphql.Int,
						DefaultValue: 5,
					},
				},
				Resolve: s.findSimilarResolver(container),
			},
			"traceSession": &graphql.Field{
				Type: graphql.NewList(chunkType),
				Args: graphql.FieldConfigArgument{
					"sessionId": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: s.traceSessionResolver(container),
			},
			"traceRelated": &graphql.Field{
				Type: graphql.NewList(chunkType),
				Args: graphql.FieldConfigArgument{
					"chunkId": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"depth": &graphql.ArgumentConfig{
						Type:         graphql.Int,
						DefaultValue: 2,
					},
				},
				Resolve: s.traceRelatedResolver(container),
			},
		},
	})

	// Create root mutation
	rootMutation := graphql.NewObject(graphql.ObjectConfig{
		Name: "Mutation",
		Fields: graphql.Fields{
			"storeChunk": &graphql.Field{
				Type: chunkType,
				Args: graphql.FieldConfigArgument{
					"input": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(storeChunkInputType),
					},
				},
				Resolve: s.storeChunkResolver(container),
			},
			"storeDecision": &graphql.Field{
				Type: chunkType,
				Args: graphql.FieldConfigArgument{
					"input": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(storeDecisionInputType),
					},
				},
				Resolve: s.storeDecisionResolver(container),
			},
			"deleteChunk": &graphql.Field{
				Type: graphql.Boolean,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: s.deleteChunkResolver(container),
			},
		},
	})

	// Create schema
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query:    rootQuery,
		Mutation: rootMutation,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create GraphQL schema: %w", err)
	}

	return &Schema{
		schema:    schema,
		container: container,
	}, nil
}

// GetSchema returns the GraphQL schema
func (s *Schema) GetSchema() graphql.Schema {
	return s.schema
}

// Helper to create a singleton schema instance
var s = &Schema{}
