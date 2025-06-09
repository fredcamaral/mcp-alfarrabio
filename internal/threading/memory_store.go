// Package threading provides memory threading and conversation flow management.
// It organizes memories into logical threads and tracks conversation relationships.
package threading

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// InMemoryThreadStore provides in-memory storage for memory threads
type InMemoryThreadStore struct {
	threads map[string]*MemoryThread
	mutex   sync.RWMutex
}

// NewInMemoryThreadStore creates a new in-memory thread store
func NewInMemoryThreadStore() *InMemoryThreadStore {
	return &InMemoryThreadStore{
		threads: make(map[string]*MemoryThread),
	}
}

// StoreThread stores a thread in memory
func (s *InMemoryThreadStore) StoreThread(ctx context.Context, thread *MemoryThread) error {
	if thread == nil {
		return errors.New("thread cannot be nil")
	}

	if thread.ID == "" {
		return errors.New("thread ID cannot be empty")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Update timestamp
	thread.LastUpdate = time.Now()

	// Store thread (copy to avoid external mutations)
	threadCopy := *thread
	s.threads[thread.ID] = &threadCopy

	return nil
}

// GetThread retrieves a thread by ID
func (s *InMemoryThreadStore) GetThread(ctx context.Context, threadID string) (*MemoryThread, error) {
	if threadID == "" {
		return nil, errors.New("thread ID cannot be empty")
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	thread, exists := s.threads[threadID]
	if !exists {
		return nil, fmt.Errorf("thread not found: %s", threadID)
	}

	// Return copy to avoid external mutations
	threadCopy := *thread
	return &threadCopy, nil
}

// GetThreadsByRepository retrieves all threads for a repository
func (s *InMemoryThreadStore) GetThreadsByRepository(ctx context.Context, repository string) ([]*MemoryThread, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var threads []*MemoryThread
	for _, thread := range s.threads {
		if thread.Repository == repository {
			threadCopy := *thread
			threads = append(threads, &threadCopy)
		}
	}

	return threads, nil
}

// GetActiveThreads retrieves all active threads for a repository
func (s *InMemoryThreadStore) GetActiveThreads(ctx context.Context, repository string) ([]*MemoryThread, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var activeThreads []*MemoryThread
	for _, thread := range s.threads {
		if thread.Repository == repository && thread.Status == ThreadStatusActive {
			threadCopy := *thread
			activeThreads = append(activeThreads, &threadCopy)
		}
	}

	return activeThreads, nil
}

// UpdateThreadStatus updates the status of a thread
func (s *InMemoryThreadStore) UpdateThreadStatus(ctx context.Context, threadID string, status ThreadStatus) error {
	if threadID == "" {
		return errors.New("thread ID cannot be empty")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	thread, exists := s.threads[threadID]
	if !exists {
		return fmt.Errorf("thread not found: %s", threadID)
	}

	thread.Status = status
	thread.LastUpdate = time.Now()

	// Set end time if thread is being completed
	if status == ThreadStatusComplete && thread.EndTime == nil {
		now := time.Now()
		thread.EndTime = &now
	}

	return nil
}

// DeleteThread removes a thread
func (s *InMemoryThreadStore) DeleteThread(ctx context.Context, threadID string) error {
	if threadID == "" {
		return errors.New("thread ID cannot be empty")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exists := s.threads[threadID]; !exists {
		return fmt.Errorf("thread not found: %s", threadID)
	}

	delete(s.threads, threadID)
	return nil
}

// ListThreads retrieves threads based on filters
func (s *InMemoryThreadStore) ListThreads(ctx context.Context, filters ThreadFilters) ([]*MemoryThread, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var filteredThreads []*MemoryThread

	for _, thread := range s.threads {
		if s.matchesFilters(thread, filters) {
			threadCopy := *thread
			filteredThreads = append(filteredThreads, &threadCopy)
		}
	}

	return filteredThreads, nil
}

// matchesFilters checks if a thread matches the given filters
func (s *InMemoryThreadStore) matchesFilters(thread *MemoryThread, filters ThreadFilters) bool {
	return s.matchesBasicFilters(thread, filters) &&
		s.matchesSessionFilter(thread, filters) &&
		s.matchesTagsFilter(thread, filters) &&
		s.matchesTimeFilters(thread, filters)
}

// matchesBasicFilters checks basic string filters
func (s *InMemoryThreadStore) matchesBasicFilters(thread *MemoryThread, filters ThreadFilters) bool {
	if filters.Repository != nil && thread.Repository != *filters.Repository {
		return false
	}
	if filters.Type != nil && thread.Type != *filters.Type {
		return false
	}
	if filters.Status != nil && thread.Status != *filters.Status {
		return false
	}
	return true
}

// matchesSessionFilter checks if thread matches session ID filter
func (s *InMemoryThreadStore) matchesSessionFilter(thread *MemoryThread, filters ThreadFilters) bool {
	if filters.SessionID == nil {
		return true
	}

	for _, sessionID := range thread.SessionIDs {
		if sessionID == *filters.SessionID {
			return true
		}
	}
	return false
}

// matchesTagsFilter checks if thread has all required tags
func (s *InMemoryThreadStore) matchesTagsFilter(thread *MemoryThread, filters ThreadFilters) bool {
	if len(filters.Tags) == 0 {
		return true
	}

	threadTagSet := make(map[string]bool)
	for _, tag := range thread.Tags {
		threadTagSet[tag] = true
	}

	for _, requiredTag := range filters.Tags {
		if !threadTagSet[requiredTag] {
			return false
		}
	}
	return true
}

// matchesTimeFilters checks time range filters
func (s *InMemoryThreadStore) matchesTimeFilters(thread *MemoryThread, filters ThreadFilters) bool {
	if filters.Since != nil && thread.LastUpdate.Before(*filters.Since) {
		return false
	}
	if filters.Until != nil && thread.StartTime.After(*filters.Until) {
		return false
	}
	return true
}

// GetThreadCount returns the total number of threads
func (s *InMemoryThreadStore) GetThreadCount() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return len(s.threads)
}

// Clear removes all threads (useful for testing)
func (s *InMemoryThreadStore) Clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.threads = make(map[string]*MemoryThread)
}
