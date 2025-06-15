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
	var allSessions []*entities.Session
	var err error

	if repository != "" {
		// Get sessions for specific repository
		allSessions, err = s.generic.GetByRepository(ctx, repository)
		if err != nil {
			return nil, err
		}
	} else {
		// Get sessions from all repositories
		allSessions, err = s.generic.GetAllFromAllRepositories(ctx)
		if err != nil {
			return nil, err
		}
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
	var allSessions []*entities.Session
	var err error

	if repository != "" {
		// Get sessions for specific repository
		allSessions, err = s.generic.GetByRepository(ctx, repository)
		if err != nil {
			return nil, err
		}
	} else {
		// Get sessions from all repositories
		allSessions, err = s.generic.GetAllFromAllRepositories(ctx)
		if err != nil {
			return nil, err
		}
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

// Helper methods

func (s *FileSessionStorage) getRepositoryPath(repository string) string {
	// Create a safe directory name from repository
	hash := sha256.Sum256([]byte(repository))
	safeName := hex.EncodeToString(hash[:8]) // Use first 8 bytes of hash
	return filepath.Join(s.basePath, safeName)
}

func (s *FileSessionStorage) loadSessionsFromFile(filePath string) ([]*entities.Session, error) {
	// Clean and validate the file path
	filePath = filepath.Clean(filePath)
	if strings.Contains(filePath, "..") {
		return nil, fmt.Errorf("path traversal detected: %s", filePath)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return []*entities.Session{}, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var sessionFile SessionFile
	if err := json.Unmarshal(data, &sessionFile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sessions: %w", err)
	}

	return sessionFile.Sessions, nil
}

func (s *FileSessionStorage) saveSessionsToFile(filePath, repository string, sessions []*entities.Session) error {
	sessionFile := SessionFile{
		Version:    SessionFileVersion,
		Repository: repository,
		Sessions:   sessions,
		UpdatedAt:  time.Now(),
	}

	data, err := json.MarshalIndent(sessionFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal sessions: %w", err)
	}

	// Write atomically
	tempFile := filePath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0o600); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tempFile, filePath); err != nil {
		_ = os.Remove(tempFile)
		return fmt.Errorf("failed to move temp file: %w", err)
	}

	return nil
}
