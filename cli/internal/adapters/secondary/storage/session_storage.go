package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

// FileSessionStorage implements session storage using file system
type FileSessionStorage struct {
	basePath string
	logger   *slog.Logger
	mutex    sync.RWMutex
}

// SessionFile represents the structure of a sessions file
type SessionFile struct {
	Version    string              `json:"version"`
	Repository string              `json:"repository"`
	Sessions   []*entities.Session `json:"sessions"`
	UpdatedAt  time.Time           `json:"updated_at"`
}

const (
	SessionFileVersion = "1.0.0"
	SessionsFileName   = "sessions.json"
)

// NewFileSessionStorage creates a new file-based session storage
func NewFileSessionStorage(logger *slog.Logger) (ports.SessionStorage, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	basePath := filepath.Join(homeDir, ".lmmc", "sessions")
	if err := os.MkdirAll(basePath, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create sessions directory: %w", err)
	}

	return &FileSessionStorage{
		basePath: basePath,
		logger:   logger,
	}, nil
}

// Create stores a new session
func (s *FileSessionStorage) Create(ctx context.Context, session *entities.Session) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := session.Validate(); err != nil {
		return fmt.Errorf("invalid session: %w", err)
	}

	repoPath := s.getRepositoryPath(session.Repository)
	if err := os.MkdirAll(repoPath, 0o750); err != nil {
		return fmt.Errorf("failed to create repository directory: %w", err)
	}

	filePath := filepath.Join(repoPath, SessionsFileName)
	sessions, _ := s.loadSessionsFromFile(filePath)

	// Check if session already exists
	for _, existing := range sessions {
		if existing.ID == session.ID {
			return fmt.Errorf("session with ID %s already exists", session.ID)
		}
	}

	sessions = append(sessions, session)

	if err := s.saveSessionsToFile(filePath, session.Repository, sessions); err != nil {
		return fmt.Errorf("failed to save sessions: %w", err)
	}

	s.logger.Debug("session created",
		slog.String("id", session.ID),
		slog.String("repository", session.Repository))
	return nil
}

// Update updates an existing session
func (s *FileSessionStorage) Update(ctx context.Context, session *entities.Session) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := session.Validate(); err != nil {
		return fmt.Errorf("invalid session: %w", err)
	}

	repoPath := s.getRepositoryPath(session.Repository)
	filePath := filepath.Join(repoPath, SessionsFileName)
	sessions, err := s.loadSessionsFromFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to load sessions: %w", err)
	}

	found := false
	for i, existing := range sessions {
		if existing.ID == session.ID {
			sessions[i] = session
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("session with ID %s not found", session.ID)
	}

	if err := s.saveSessionsToFile(filePath, session.Repository, sessions); err != nil {
		return fmt.Errorf("failed to save sessions: %w", err)
	}

	s.logger.Debug("session updated", slog.String("id", session.ID))
	return nil
}

// Delete removes a session
func (s *FileSessionStorage) Delete(ctx context.Context, id string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check all repositories for the session
	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		return fmt.Errorf("failed to read sessions directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		filePath := filepath.Join(s.basePath, entry.Name(), SessionsFileName)
		sessions, err := s.loadSessionsFromFile(filePath)
		if err != nil {
			continue
		}

		filtered := make([]*entities.Session, 0, len(sessions))
		found := false
		repository := ""

		for _, session := range sessions {
			if session.ID == id {
				found = true
				repository = session.Repository
				continue
			}
			filtered = append(filtered, session)
		}

		if found {
			if len(filtered) == 0 {
				// Remove empty file
				if err := os.Remove(filePath); err != nil {
					return fmt.Errorf("failed to remove empty sessions file: %w", err)
				}
			} else {
				if err := s.saveSessionsToFile(filePath, repository, filtered); err != nil {
					return fmt.Errorf("failed to save sessions: %w", err)
				}
			}

			s.logger.Debug("session deleted", slog.String("id", id))
			return nil
		}
	}

	return fmt.Errorf("session with ID %s not found", id)
}

// GetByID retrieves a session by ID
func (s *FileSessionStorage) GetByID(ctx context.Context, id string) (*entities.Session, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Check all repositories for the session
	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		return nil, fmt.Errorf("session with ID %s not found", id)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		filePath := filepath.Join(s.basePath, entry.Name(), SessionsFileName)
		sessions, err := s.loadSessionsFromFile(filePath)
		if err != nil {
			continue
		}

		for _, session := range sessions {
			if session.ID == id {
				return session, nil
			}
		}
	}

	return nil, fmt.Errorf("session with ID %s not found", id)
}

// GetByRepository retrieves sessions for a repository
func (s *FileSessionStorage) GetByRepository(ctx context.Context, repository string) ([]*entities.Session, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	repoPath := s.getRepositoryPath(repository)
	filePath := filepath.Join(repoPath, SessionsFileName)
	sessions, err := s.loadSessionsFromFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load sessions: %w", err)
	}

	// Sort by start time (newest first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].StartTime.After(sessions[j].StartTime)
	})

	s.logger.Debug("sessions retrieved by repository",
		slog.String("repository", repository),
		slog.Int("count", len(sessions)))
	return sessions, nil
}

// GetByTimeRange retrieves sessions within a time range
func (s *FileSessionStorage) GetByTimeRange(ctx context.Context, repository string, start, end time.Time) ([]*entities.Session, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var allSessions []*entities.Session

	if repository != "" {
		// Get sessions for specific repository
		repoPath := s.getRepositoryPath(repository)
		filePath := filepath.Join(repoPath, SessionsFileName)
		sessions, err := s.loadSessionsFromFile(filePath)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load sessions: %w", err)
		}
		allSessions = sessions
	} else {
		// Get sessions from all repositories
		entries, err := os.ReadDir(s.basePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read sessions directory: %w", err)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			filePath := filepath.Join(s.basePath, entry.Name(), SessionsFileName)
			sessions, err := s.loadSessionsFromFile(filePath)
			if err != nil {
				continue
			}
			allSessions = append(allSessions, sessions...)
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

	s.logger.Debug("sessions retrieved by time range",
		slog.String("repository", repository),
		slog.Time("start", start),
		slog.Time("end", end),
		slog.Int("count", len(filtered)))
	return filtered, nil
}

// GetActiveSessions retrieves active sessions (sessions without end time)
func (s *FileSessionStorage) GetActiveSessions(ctx context.Context, repository string) ([]*entities.Session, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var allSessions []*entities.Session

	if repository != "" {
		// Get sessions for specific repository
		repoPath := s.getRepositoryPath(repository)
		filePath := filepath.Join(repoPath, SessionsFileName)
		sessions, err := s.loadSessionsFromFile(filePath)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load sessions: %w", err)
		}
		allSessions = sessions
	} else {
		// Get sessions from all repositories
		entries, err := os.ReadDir(s.basePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read sessions directory: %w", err)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			filePath := filepath.Join(s.basePath, entry.Name(), SessionsFileName)
			sessions, err := s.loadSessionsFromFile(filePath)
			if err != nil {
				continue
			}
			allSessions = append(allSessions, sessions...)
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

	s.logger.Debug("active sessions retrieved",
		slog.String("repository", repository),
		slog.Int("count", len(filtered)))
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
