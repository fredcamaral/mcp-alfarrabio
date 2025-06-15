package storage

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

// FileSessionStorage implements session storage using file system
type FileSessionStorage struct {
	generic *GenericFileStorage[*entities.Session]
}

const (
	SessionFileVersion = "1.0.0"
	SessionsFileName   = "sessions.json"
)

// NewFileSessionStorage creates a new file-based session storage
func NewFileSessionStorage(logger *slog.Logger) (ports.SessionStorage, error) {
	config := FileStorageConfig{
		SubDir:   "sessions",
		FileName: SessionsFileName,
		Version:  SessionFileVersion,
		Logger:   logger,
	}

	generic, err := NewGenericFileStorage[*entities.Session](config)
	if err != nil {
		return nil, fmt.Errorf("failed to create generic storage: %w", err)
	}

	return &FileSessionStorage{
		generic: generic,
	}, nil
}

// Create stores a new session
func (s *FileSessionStorage) Create(ctx context.Context, session *entities.Session) error {
	return s.generic.Create(ctx, session)
}

// Update updates an existing session
func (s *FileSessionStorage) Update(ctx context.Context, session *entities.Session) error {
	return s.generic.Update(ctx, session)
}

// Delete removes a session
func (s *FileSessionStorage) Delete(ctx context.Context, id string) error {
	return s.generic.Delete(ctx, id)
}

// GetByID retrieves a session by ID
func (s *FileSessionStorage) GetByID(ctx context.Context, id string) (*entities.Session, error) {
	return s.generic.GetByID(ctx, id)
}

// GetByRepository retrieves sessions for a repository
func (s *FileSessionStorage) GetByRepository(ctx context.Context, repository string) ([]*entities.Session, error) {
	sessions, err := s.generic.GetByRepository(ctx, repository)
	if err != nil {
		return nil, err
	}

	// Sort by start time (newest first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].StartTime.After(sessions[j].StartTime)
	})

	return sessions, nil
}

// GetByTimeRange retrieves sessions within a time range
func (s *FileSessionStorage) GetByTimeRange(ctx context.Context, repository string, start, end time.Time) ([]*entities.Session, error) {
	allSessions, err := s.getAllSessionsByRepository(ctx, repository)
	if err != nil {
		return nil, err
	}

	// Filter by time range
	filtered := make([]*entities.Session, 0)
	for _, session := range allSessions {
		sessionEnd := session.EndTime
		if sessionEnd == nil {
			// For active sessions, use current time as end
			now := time.Now()
			sessionEnd = &now
		}

		// Check if session overlaps with the time range
		if session.StartTime.Before(end) && sessionEnd.After(start) {
			filtered = append(filtered, session)
		}
	}

	// Sort by start time
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].StartTime.After(filtered[j].StartTime)
	})

	return filtered, nil
}

// GetActiveSessions retrieves active sessions (sessions without end time)
func (s *FileSessionStorage) GetActiveSessions(ctx context.Context, repository string) ([]*entities.Session, error) {
	allSessions, err := s.getAllSessionsByRepository(ctx, repository)
	if err != nil {
		return nil, err
	}

	// Filter for active sessions (no end time)
	filtered := make([]*entities.Session, 0)
	for _, session := range allSessions {
		if session.EndTime == nil {
			filtered = append(filtered, session)
		}
	}

	// Sort by start time (newest first)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].StartTime.After(filtered[j].StartTime)
	})

	return filtered, nil
}

// getAllSessionsByRepository is a helper function to get sessions by repository or all repositories
// This eliminates code duplication between GetByTimeRange and GetActiveSessions
func (s *FileSessionStorage) getAllSessionsByRepository(ctx context.Context, repository string) ([]*entities.Session, error) {
	if repository != "" {
		// Get sessions for specific repository
		return s.generic.GetByRepository(ctx, repository)
	}
	// Get sessions from all repositories
	return s.generic.GetAllFromAllRepositories(ctx)
}
