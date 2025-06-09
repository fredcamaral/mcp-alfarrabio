// Package prd provides PRD storage interface and in-memory implementation.
package prd

import (
	"context"
	"fmt"
	"sync"
	"time"

	"lerian-mcp-memory/pkg/types"
)

// Storage defines the interface for PRD storage operations
type Storage interface {
	Store(ctx context.Context, doc *types.PRDDocument) error
	Get(ctx context.Context, id string) (*types.PRDDocument, error)
	List(ctx context.Context, filters StorageFilters) ([]*types.PRDDocument, error)
	Update(ctx context.Context, doc *types.PRDDocument) error
	Delete(ctx context.Context, id string) error
}

// StorageFilters represents filters for PRD listing
type StorageFilters struct {
	Status      types.PRDStatus   `json:"status,omitempty"`
	Priority    types.PRDPriority `json:"priority,omitempty"`
	ProjectType types.ProjectType `json:"project_type,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Limit       int               `json:"limit,omitempty"`
	Offset      int               `json:"offset,omitempty"`
}

// InMemoryStorage provides an in-memory implementation of PRD storage
type InMemoryStorage struct {
	documents map[string]*types.PRDDocument
	mutex     sync.RWMutex
}

// NewInMemoryStorage creates a new in-memory storage instance
func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		documents: make(map[string]*types.PRDDocument),
	}
}

// Store stores a PRD document
func (s *InMemoryStorage) Store(ctx context.Context, doc *types.PRDDocument) error {
	if doc == nil {
		return fmt.Errorf("document cannot be nil")
	}

	if doc.ID == "" {
		return fmt.Errorf("document ID cannot be empty")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Create a copy to avoid external modifications
	docCopy := *doc
	docCopy.Timestamps.Created = time.Now()
	docCopy.Timestamps.Updated = time.Now()

	s.documents[doc.ID] = &docCopy
	return nil
}

// Get retrieves a PRD document by ID
func (s *InMemoryStorage) Get(ctx context.Context, id string) (*types.PRDDocument, error) {
	if id == "" {
		return nil, fmt.Errorf("document ID cannot be empty")
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	doc, exists := s.documents[id]
	if !exists {
		return nil, fmt.Errorf("document with ID '%s' not found", id)
	}

	// Return a copy to avoid external modifications
	docCopy := *doc
	return &docCopy, nil
}

// List retrieves PRD documents based on filters
func (s *InMemoryStorage) List(ctx context.Context, filters *StorageFilters) ([]*types.PRDDocument, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var results []*types.PRDDocument

	for _, doc := range s.documents {
		if s.matchesFilters(doc, filters) {
			// Add a copy to avoid external modifications
			docCopy := *doc
			results = append(results, &docCopy)
		}
	}

	// Apply offset and limit
	if filters.Offset > 0 && filters.Offset < len(results) {
		results = results[filters.Offset:]
	} else if filters.Offset >= len(results) {
		results = []*types.PRDDocument{}
	}

	if filters.Limit > 0 && filters.Limit < len(results) {
		results = results[:filters.Limit]
	}

	return results, nil
}

// Update updates a PRD document
func (s *InMemoryStorage) Update(ctx context.Context, doc *types.PRDDocument) error {
	if doc == nil {
		return fmt.Errorf("document cannot be nil")
	}

	if doc.ID == "" {
		return fmt.Errorf("document ID cannot be empty")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exists := s.documents[doc.ID]; !exists {
		return fmt.Errorf("document with ID '%s' not found", doc.ID)
	}

	// Create a copy and update timestamp
	docCopy := *doc
	docCopy.Timestamps.Updated = time.Now()

	s.documents[doc.ID] = &docCopy
	return nil
}

// Delete deletes a PRD document
func (s *InMemoryStorage) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("document ID cannot be empty")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exists := s.documents[id]; !exists {
		return fmt.Errorf("document with ID '%s' not found", id)
	}

	delete(s.documents, id)
	return nil
}

// matchesFilters checks if a document matches the given filters
func (s *InMemoryStorage) matchesFilters(doc *types.PRDDocument, filters *StorageFilters) bool {
	// Status filter
	if filters.Status != "" && doc.Status != filters.Status {
		return false
	}

	// Priority filter
	if filters.Priority != "" && doc.Metadata.Priority != filters.Priority {
		return false
	}

	// Project type filter
	if filters.ProjectType != "" && doc.Metadata.ProjectType != filters.ProjectType {
		return false
	}

	// Tags filter (document must have all specified tags)
	if len(filters.Tags) > 0 {
		docTags := make(map[string]bool)
		for _, tag := range doc.Metadata.Tags {
			docTags[tag] = true
		}

		for _, requiredTag := range filters.Tags {
			if !docTags[requiredTag] {
				return false
			}
		}
	}

	return true
}

// GetCount returns the total number of stored documents
func (s *InMemoryStorage) GetCount() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return len(s.documents)
}

// Clear removes all documents (useful for testing)
func (s *InMemoryStorage) Clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.documents = make(map[string]*types.PRDDocument)
}
