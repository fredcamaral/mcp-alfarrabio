// Package embeddings provides interfaces and types for embeddings generation
package embeddings

import (
	"context"
)

// EmbeddingService defines the interface for generating text embeddings
type EmbeddingService interface {
	// Generate creates embeddings for a single text
	Generate(ctx context.Context, text string) ([]float64, error)

	// GenerateBatch creates embeddings for multiple texts efficiently
	GenerateBatch(ctx context.Context, texts []string) ([][]float64, error)

	// GetDimensions returns the number of dimensions in embeddings
	GetDimensions() int

	// HealthCheck verifies the service is working properly
	HealthCheck(ctx context.Context) error
}

// EmbeddingConfig represents configuration for embedding services
type EmbeddingConfig struct {
	Provider string `json:"provider"` // "openai", "local", "mock"

	// OpenAI specific settings
	OpenAI *OpenAIConfig `json:"openai,omitempty"`

	// Local model specific settings (for future use)
	LocalModel *LocalModelConfig `json:"local_model,omitempty"`
}

// LocalModelConfig represents configuration for local embedding models
type LocalModelConfig struct {
	ModelPath  string `json:"model_path"`
	DeviceType string `json:"device_type"` // "cpu", "gpu"
	BatchSize  int    `json:"batch_size"`
	ModelType  string `json:"model_type"` // "sentence-transformers", "huggingface"
	Dimensions int    `json:"dimensions"`
}

// EmbeddingRequest represents a request for embeddings generation
type EmbeddingRequest struct {
	Text     string                 `json:"text,omitempty"`
	Texts    []string               `json:"texts,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// EmbeddingResponse represents the response from embeddings generation
type EmbeddingResponse struct {
	Embedding  []float64              `json:"embedding,omitempty"`
	Embeddings [][]float64            `json:"embeddings,omitempty"`
	Dimensions int                    `json:"dimensions"`
	Model      string                 `json:"model"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}
