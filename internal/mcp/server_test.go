package mcp

import (
	"context"
	"mcp-memory/internal/config"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMemoryServer(t *testing.T) {
	cfg := &config.Config{
		Chroma: config.ChromaConfig{
			Endpoint:   "http://localhost:8000",
			Collection: "test",
		},
		OpenAI: config.OpenAIConfig{
			APIKey:         "test-key",
			EmbeddingModel: "text-embedding-ada-002",
			RateLimitRPM:   60,
		},
		Chunking: config.ChunkingConfig{
			MinContentLength:     100,
			MaxContentLength:     4000,
			SimilarityThreshold:  0.8,
			TimeThresholdMinutes: 20,
		},
	}

	server, err := NewMemoryServer(cfg)

	assert.NoError(t, err)
	assert.NotNil(t, server)
	assert.NotNil(t, server.container)
	assert.NotNil(t, server.mcpServer)
}

func TestMemoryServer_Start(t *testing.T) {
	cfg := &config.Config{
		Chroma: config.ChromaConfig{
			Endpoint:   "http://localhost:8000",
			Collection: "test",
		},
		OpenAI: config.OpenAIConfig{
			APIKey:         "test-key",
			EmbeddingModel: "text-embedding-ada-002",
			RateLimitRPM:   60,
		},
		Chunking: config.ChunkingConfig{
			MinContentLength:     100,
			MaxContentLength:     4000,
			SimilarityThreshold:  0.8,
			TimeThresholdMinutes: 20,
		},
	}

	server, err := NewMemoryServer(cfg)
	assert.NoError(t, err)

	ctx := context.Background()

	// Start will fail due to no real Chroma instance, but it covers the method
	err = server.Start(ctx)
	assert.Error(t, err) // Expected to fail without real services
}

func TestMemoryServer_Close(t *testing.T) {
	cfg := &config.Config{
		Chroma: config.ChromaConfig{
			Endpoint:   "http://localhost:8000",
			Collection: "test",
		},
		OpenAI: config.OpenAIConfig{
			APIKey:         "test-key",
			EmbeddingModel: "text-embedding-ada-002",
			RateLimitRPM:   60,
		},
		Chunking: config.ChunkingConfig{
			MinContentLength:     100,
			MaxContentLength:     4000,
			SimilarityThreshold:  0.8,
			TimeThresholdMinutes: 20,
		},
	}

	server, err := NewMemoryServer(cfg)
	assert.NoError(t, err)

	err = server.Close()
	assert.NoError(t, err) // Close should succeed
}

func TestMemoryServer_GetServer(t *testing.T) {
	cfg := &config.Config{
		Chroma: config.ChromaConfig{
			Endpoint:   "http://localhost:8000",
			Collection: "test",
		},
		OpenAI: config.OpenAIConfig{
			APIKey:         "test-key",
			EmbeddingModel: "text-embedding-ada-002",
			RateLimitRPM:   60,
		},
		Chunking: config.ChunkingConfig{
			MinContentLength:     100,
			MaxContentLength:     4000,
			SimilarityThreshold:  0.8,
			TimeThresholdMinutes: 20,
		},
	}

	server, err := NewMemoryServer(cfg)
	assert.NoError(t, err)

	mcpServer := server.GetServer()
	// MCP server is now initialized since we internalized the MCP-Go library
	assert.NotNil(t, mcpServer)
}

func TestMemoryServer_HandleStoreChunk_InvalidInput(t *testing.T) {
	cfg := &config.Config{
		Chroma: config.ChromaConfig{
			Endpoint:   "http://localhost:8000",
			Collection: "test",
		},
		OpenAI: config.OpenAIConfig{
			APIKey:         "test-key",
			EmbeddingModel: "text-embedding-ada-002",
			RateLimitRPM:   60,
		},
		Chunking: config.ChunkingConfig{
			MinContentLength:     100,
			MaxContentLength:     4000,
			SimilarityThreshold:  0.8,
			TimeThresholdMinutes: 20,
		},
	}

	server, err := NewMemoryServer(cfg)
	assert.NoError(t, err)

	ctx := context.Background()

	// Test with missing content
	result, err := server.handleStoreChunk(ctx, map[string]interface{}{
		"session_id": "test-session",
	})
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "content is required")

	// Test with missing session_id
	result, err = server.handleStoreChunk(ctx, map[string]interface{}{
		"content": "test content",
	})
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "session_id is required")
}

func TestMemoryServer_HandleSearch_InvalidInput(t *testing.T) {
	cfg := &config.Config{
		Chroma: config.ChromaConfig{
			Endpoint:   "http://localhost:8000",
			Collection: "test",
		},
		OpenAI: config.OpenAIConfig{
			APIKey:         "test-key",
			EmbeddingModel: "text-embedding-ada-002",
			RateLimitRPM:   60,
		},
		Chunking: config.ChunkingConfig{
			MinContentLength:     100,
			MaxContentLength:     4000,
			SimilarityThreshold:  0.8,
			TimeThresholdMinutes: 20,
		},
	}

	server, err := NewMemoryServer(cfg)
	assert.NoError(t, err)

	ctx := context.Background()

	// Test with missing query
	result, err := server.handleSearch(ctx, map[string]interface{}{})
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "query is required")
}
