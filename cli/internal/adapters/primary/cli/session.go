package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lerian-mcp-memory-cli/internal/domain/constants"
	"lerian-mcp-memory-cli/internal/domain/services"
)

// SessionData represents the workflow session state
type SessionData struct {
	ID             string            `json:"id"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	CurrentStep    string            `json:"current_step"`
	Repository     string            `json:"repository"`
	Values         map[string]string `json:"values"`
	WorkflowStatus string            `json:"workflow_status"`
}

// getSessionFilePath returns the path to the session file
func (c *CLI) getSessionFilePath() string {
	// Use .lmmc directory in current working directory
	wd, _ := os.Getwd()
	return filepath.Join(wd, ".lmmc", "session.json")
}

// loadSession loads the current session
func (c *CLI) loadSession() (*SessionData, error) {
	sessionFile := c.getSessionFilePath()

	// Validate session file path
	cleanPath := filepath.Clean(sessionFile)
	if !strings.HasSuffix(cleanPath, filepath.Join(".lmmc", "session.json")) {
		return nil, fmt.Errorf("invalid session file path")
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create new session
			return &SessionData{
				ID:        generateSessionID(),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Values:    make(map[string]string),
			}, nil
		}
		return nil, err
	}

	var session SessionData
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// saveSession saves the session data
func (c *CLI) saveSession(session *SessionData) error {
	sessionFile := c.getSessionFilePath()

	// Ensure directory exists
	dir := filepath.Dir(sessionFile)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}

	session.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(sessionFile, data, 0600)
}

// getSessionValue retrieves a value from the session
func (c *CLI) getSessionValue(key string) string {
	session, err := c.loadSession()
	if err != nil {
		return ""
	}

	return session.Values[key]
}

// updateSession updates a value in the session
func (c *CLI) updateSession(key, value string) {
	session, err := c.loadSession()
	if err != nil {
		session = &SessionData{
			ID:        generateSessionID(),
			CreatedAt: time.Now(),
			Values:    make(map[string]string),
		}
	}

	if session.Values == nil {
		session.Values = make(map[string]string)
	}

	session.Values[key] = value

	// Update workflow status based on key
	switch key {
	case "prd_file":
		session.CurrentStep = "prd_created"
	case "trd_file":
		session.CurrentStep = "trd_created"
	case "tasks_file":
		session.CurrentStep = "tasks_generated"
	case "subtasks_file":
		session.CurrentStep = "subtasks_generated"
	}

	if err := c.saveSession(session); err != nil {
		c.logger.Warn("Failed to save session", "error", err)
	}
}

// clearSession clears the current session
func (c *CLI) clearSession() error {
	sessionFile := c.getSessionFilePath()
	return os.Remove(sessionFile)
}

// getWorkflowStatus returns the current workflow status
func (c *CLI) getWorkflowStatus() string {
	session, err := c.loadSession()
	if err != nil {
		return "no_session"
	}

	// Determine status based on available files
	if session.Values["subtasks_file"] != "" {
		return "ready_for_implementation"
	}
	if session.Values["tasks_file"] != "" {
		return "ready_for_subtasks"
	}
	if session.Values["trd_file"] != "" {
		return "ready_for_tasks"
	}
	if session.Values["prd_file"] != "" {
		return "ready_for_trd"
	}

	return "ready_to_start"
}

// generateSessionID creates a new session ID
func generateSessionID() string {
	return time.Now().Format("20060102-150405")
}

// Helper methods that were referenced in other files

// detectLatestPRD finds the most recent PRD file
func (c *CLI) detectLatestPRD() string {
	// Check session first
	if sessionPRD := c.getSessionValue("prd_file"); sessionPRD != "" {
		if _, err := os.Stat(sessionPRD); err == nil {
			return sessionPRD
		}
	}

	// Look in standard location
	preDev := constants.DefaultPreDevelopmentDir
	if _, err := os.Stat(preDev); err == nil {
		files, _ := filepath.Glob(filepath.Join(preDev, "prd-*.md"))
		if len(files) > 0 {
			// Return the most recent
			return files[len(files)-1]
		}
	}

	return ""
}

// detectLatestTRD finds the most recent TRD file
func (c *CLI) detectLatestTRD() string {
	// Check session first
	if sessionTRD := c.getSessionValue("trd_file"); sessionTRD != "" {
		if _, err := os.Stat(sessionTRD); err == nil {
			return sessionTRD
		}
	}

	// Look in standard location
	preDev := constants.DefaultPreDevelopmentDir
	if _, err := os.Stat(preDev); err == nil {
		files, _ := filepath.Glob(filepath.Join(preDev, "trd-*.md"))
		if len(files) > 0 {
			// Return the most recent
			return files[len(files)-1]
		}
	}

	return ""
}

// detectLatestTasksFile finds the most recent tasks file
func (c *CLI) detectLatestTasksFile() string {
	// Check session first
	if sessionTasks := c.getSessionValue("tasks_file"); sessionTasks != "" {
		if _, err := os.Stat(sessionTasks); err == nil {
			return sessionTasks
		}
	}

	// Look in standard location
	preDev := constants.DefaultPreDevelopmentDir
	if _, err := os.Stat(preDev); err == nil {
		files, _ := filepath.Glob(filepath.Join(preDev, "tasks-*.md"))
		if len(files) > 0 {
			// Return the most recent
			return files[len(files)-1]
		}
	}

	return ""
}

// loadPRDFromFile loads a PRD from a file (placeholder)
func (c *CLI) loadPRDFromFile(path string) *services.PRDEntity {
	// TODO: Implement actual PRD loading from markdown
	// For now, return a mock
	return &services.PRDEntity{
		ID:          "prd-001",
		Title:       "Loaded PRD",
		Description: "PRD loaded from " + path,
		CreatedAt:   time.Now(),
	}
}

// loadTRDFromFile loads a TRD from a file (placeholder)
func (c *CLI) loadTRDFromFile(_ string) (*services.TRDEntity, error) {
	// TODO: Implement actual TRD loading from markdown
	// For now, return a mock
	return &services.TRDEntity{
		ID:        "trd-001",
		PRDID:     "prd-001",
		Title:     "Loaded TRD",
		CreatedAt: time.Now(),
	}, nil
}
