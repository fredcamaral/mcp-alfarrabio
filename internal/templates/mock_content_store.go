// Package templates provides mock content store for template storage adapter
package templates

import (
	"context"
	"fmt"

	"lerian-mcp-memory/internal/storage"
	"lerian-mcp-memory/internal/types"
	pkgtypes "lerian-mcp-memory/pkg/types"
)

// MockContentStore implements storage.ContentStore interface using VectorStore as backend
type MockContentStore struct {
	vectorStore storage.VectorStore
}

// NewMockContentStore creates a new mock content store using vector store as backend
func NewMockContentStore(vectorStore storage.VectorStore) storage.ContentStore {
	return &MockContentStore{
		vectorStore: vectorStore,
	}
}

// Store content using vector store backend
func (m *MockContentStore) Store(ctx context.Context, content *types.Content) error {
	// Convert Content to ConversationChunk for vector store
	chunk := &pkgtypes.ConversationChunk{
		ID:        content.ID,
		SessionID: string(content.SessionID),
		Type:      pkgtypes.ChunkTypeDiscussion, // Use discussion type for template content
		Content:   content.Content,
		Summary:   fmt.Sprintf("Template content: %s", content.Type),
		Metadata: pkgtypes.ChunkMetadata{
			Repository: string(content.ProjectID),
			Tags:       []string{content.Type, "template"},
		},
		Embeddings: []float64{}, // No embeddings for template content
		Timestamp:  content.CreatedAt,
	}

	return m.vectorStore.Store(ctx, chunk)
}

// Update existing content
func (m *MockContentStore) Update(ctx context.Context, content *types.Content) error {
	// Convert Content to ConversationChunk for vector store
	chunk := &pkgtypes.ConversationChunk{
		ID:        content.ID,
		SessionID: string(content.SessionID),
		Type:      pkgtypes.ChunkTypeDiscussion,
		Content:   content.Content,
		Summary:   fmt.Sprintf("Template content: %s", content.Type),
		Metadata: pkgtypes.ChunkMetadata{
			Repository: string(content.ProjectID),
			Tags:       []string{content.Type, "template"},
		},
		Embeddings: []float64{},
		Timestamp:  content.UpdatedAt,
	}

	return m.vectorStore.Update(ctx, chunk)
}

// Delete content by project and content ID
func (m *MockContentStore) Delete(ctx context.Context, projectID types.ProjectID, contentID string) error {
	return m.vectorStore.Delete(ctx, contentID)
}

// Get content by project and content ID
func (m *MockContentStore) Get(ctx context.Context, projectID types.ProjectID, contentID string) (*types.Content, error) {
	chunk, err := m.vectorStore.GetByID(ctx, contentID)
	if err != nil {
		return nil, err
	}

	// Convert ConversationChunk back to Content
	content := &types.Content{
		ID:        chunk.ID,
		ProjectID: types.ProjectID(chunk.Metadata.Repository),
		SessionID: types.SessionID(chunk.SessionID),
		Type:      "template", // Default type for template content
		Content:   chunk.Content,
		Metadata:  make(map[string]interface{}),
		CreatedAt: chunk.Timestamp, // Use timestamp as created time
		UpdatedAt: chunk.Timestamp, // Use timestamp as updated time
	}

	// Extract metadata from chunk tags
	for _, tag := range chunk.Metadata.Tags {
		content.Metadata[tag] = true
	}

	return content, nil
}

// BatchStore stores multiple content items
func (m *MockContentStore) BatchStore(ctx context.Context, contents []*types.Content) (*storage.BatchResult, error) {
	var chunks []*pkgtypes.ConversationChunk
	for _, content := range contents {
		chunk := &pkgtypes.ConversationChunk{
			ID:        content.ID,
			SessionID: string(content.SessionID),
			Type:      pkgtypes.ChunkTypeDiscussion,
			Content:   content.Content,
			Summary:   fmt.Sprintf("Template content: %s", content.Type),
			Metadata: pkgtypes.ChunkMetadata{
				Repository: string(content.ProjectID),
				Tags:       []string{content.Type, "template"},
			},
			Embeddings: []float64{},
			Timestamp:  content.CreatedAt,
		}
		chunks = append(chunks, chunk)
	}

	legacyResult, err := m.vectorStore.BatchStore(ctx, chunks)
	if err != nil {
		return nil, err
	}

	// Convert LegacyBatchResult to BatchResult
	result := &storage.BatchResult{
		Success:      legacyResult.Success,
		Failed:       legacyResult.Failed,
		ProcessedIDs: legacyResult.ProcessedIDs,
		Errors:       make([]storage.BatchError, len(legacyResult.Errors)),
	}

	// Convert error strings to BatchError structs
	for i, errStr := range legacyResult.Errors {
		result.Errors[i] = storage.BatchError{
			Index: i,
			Error: errStr,
		}
	}

	return result, nil
}

// BatchUpdate updates multiple content items
func (m *MockContentStore) BatchUpdate(ctx context.Context, contents []*types.Content) (*storage.BatchResult, error) {
	// For simplicity, implement as individual updates
	result := &storage.BatchResult{
		Success:      0,
		Failed:       0,
		Errors:       []storage.BatchError{},
		ProcessedIDs: []string{},
	}

	for i, content := range contents {
		result.ProcessedIDs = append(result.ProcessedIDs, content.ID)
		if err := m.Update(ctx, content); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, storage.BatchError{
				Index: i,
				ID:    content.ID,
				Error: err.Error(),
			})
		} else {
			result.Success++
		}
	}

	return result, nil
}

// BatchDelete deletes multiple content items
func (m *MockContentStore) BatchDelete(ctx context.Context, projectID types.ProjectID, contentIDs []string) (*storage.BatchResult, error) {
	// Call vector store batch delete which returns LegacyBatchResult
	legacyResult, err := m.vectorStore.BatchDelete(ctx, contentIDs)
	if err != nil {
		return nil, err
	}

	// Convert LegacyBatchResult to BatchResult
	result := &storage.BatchResult{
		Success:      legacyResult.Success,
		Failed:       legacyResult.Failed,
		ProcessedIDs: legacyResult.ProcessedIDs,
		Errors:       make([]storage.BatchError, len(legacyResult.Errors)),
	}

	// Convert string errors to BatchError structs
	for i, errStr := range legacyResult.Errors {
		result.Errors[i] = storage.BatchError{
			Index: i,
			Error: errStr,
		}
	}

	return result, nil
}
