// Package http provides HTTP REST client implementation
// for the lerian-mcp-memory CLI application.
package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

// HTTPRestClient implements the ports.MCPClient interface using HTTP REST
type HTTPRestClient struct {
	baseURL      string
	httpClient   *http.Client
	logger       *slog.Logger
	online       atomic.Bool
	healthTicker *time.Ticker
	stopChan     chan struct{}
}

// TaskListResponse represents the response from GET /api/v1/tasks
type TaskListResponse struct {
	Tasks []entities.Task `json:"tasks"`
	Total int             `json:"total"`
}

// HealthResponse represents the response from GET /api/v1/health
type HealthResponse struct {
	Status string `json:"status"`
	Server string `json:"server"`
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Details string `json:"details"`
	} `json:"error"`
	Timestamp string `json:"timestamp"`
}

// CallMCPTool is not implemented for HTTP REST client - use MCP client instead
func (c *HTTPRestClient) CallMCPTool(ctx context.Context, tool string, params map[string]interface{}) (map[string]interface{}, error) {
	return nil, errors.New("MCP tools not available via HTTP REST API - use MCP protocol instead")
}

var (
	ErrServerOffline = errors.New("server is offline")
	ErrServerError   = errors.New("server error")
	ErrTimeout       = errors.New("request timeout")
)

// NewHTTPRestClient creates a new HTTP REST client
func NewHTTPRestClient(config *entities.Config, logger *slog.Logger) ports.MCPClient {
	client := &HTTPRestClient{
		baseURL: config.Server.URL,
		httpClient: &http.Client{
			Timeout: time.Duration(config.Server.Timeout) * time.Second,
		},
		logger:   logger,
		stopChan: make(chan struct{}),
	}

	// Start periodic health checks
	go client.periodicHealthCheck()

	return client
}

// SyncTask creates a new task on the server using POST /api/v1/tasks
func (c *HTTPRestClient) SyncTask(ctx context.Context, task *entities.Task) error {
	if !c.IsOnline() {
		return ErrServerOffline
	}

	// Convert CLI task entity to server API format
	serverTask := c.convertTaskToServerFormat(task)

	// Marshal server task to JSON
	taskJSON, err := json.Marshal(serverTask)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/tasks", bytes.NewBuffer(taskJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.setOffline()
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Handle response
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		// Read response body for debugging
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			c.logger.Error("Failed to read error response", slog.Any("error", err))
		} else {
			c.logger.Error("Task creation failed",
				slog.Int("status", resp.StatusCode),
				slog.String("request_body", string(taskJSON)),
				slog.String("response_body", string(body)))
		}
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	c.setOnline()
	c.logger.Debug("Task synced successfully", slog.String("task_id", task.ID))
	return nil
}

// GetTasks retrieves tasks from the server using GET /api/v1/tasks
func (c *HTTPRestClient) GetTasks(ctx context.Context, repository string) ([]*entities.Task, error) {
	if !c.IsOnline() {
		return nil, ErrServerOffline
	}

	// Build URL with repository query parameter
	url := fmt.Sprintf("%s/api/v1/tasks?repository=%s", c.baseURL, repository)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.setOffline()
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Handle response
	if resp.StatusCode != http.StatusOK {
		c.handleErrorResponse(resp)
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Server may return different format, let's parse as generic first
	var serverResponse map[string]interface{}
	if err := json.Unmarshal(body, &serverResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert server tasks to CLI task entities
	tasks := c.convertServerTasksToCliTasks(serverResponse)

	c.setOnline()
	c.logger.Debug("Tasks retrieved successfully",
		slog.String("repository", repository),
		slog.Int("count", len(tasks)))

	return tasks, nil
}

// UpdateTaskStatus updates a task's status using PUT /api/v1/tasks/{id}
func (c *HTTPRestClient) UpdateTaskStatus(ctx context.Context, taskID string, status entities.Status) error {
	if !c.IsOnline() {
		return ErrServerOffline
	}

	// Map CLI "pending" status to server "todo"
	serverStatus := string(status)
	if status == entities.StatusPending {
		serverStatus = "todo"
	}

	// Create update payload
	updateData := map[string]interface{}{
		"status":     serverStatus,
		"updated_at": time.Now().Format(time.RFC3339),
	}

	// Handle completion timestamp
	if status == entities.StatusCompleted {
		updateData["completed_at"] = time.Now().Format(time.RFC3339)
	}

	// Marshal update to JSON
	updateJSON, err := json.Marshal(updateData)
	if err != nil {
		return fmt.Errorf("failed to marshal update: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/tasks/%s", c.baseURL, taskID)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(updateJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.setOffline()
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Handle response
	if resp.StatusCode != http.StatusOK {
		c.handleErrorResponse(resp)
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	c.setOnline()
	c.logger.Debug("Task status updated successfully",
		slog.String("task_id", taskID),
		slog.String("status", string(status)))

	return nil
}

// QueryIntelligence is deprecated - CLI now uses shared AI package directly
func (c *HTTPRestClient) QueryIntelligence(ctx context.Context, operation string, options map[string]interface{}) (map[string]interface{}, error) {
	c.logger.Info("QueryIntelligence is deprecated - CLI uses shared AI package directly")
	return map[string]interface{}{
		"message": "Intelligence operations handled by shared AI package",
		"status":  "deprecated",
	}, nil
}

// TestConnection tests the connection using GET /api/v1/health
func (c *HTTPRestClient) TestConnection(ctx context.Context) error {
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v1/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.setOffline()
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Handle response
	if resp.StatusCode != http.StatusOK {
		c.setOffline()
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	// Parse health response (optional)
	body, err := io.ReadAll(resp.Body)
	if err == nil {
		var health HealthResponse
		if json.Unmarshal(body, &health) == nil {
			c.logger.Debug("Health check successful", slog.String("server", health.Server))
		}
	}

	c.setOnline()
	return nil
}

// IsOnline returns true if the server is reachable
func (c *HTTPRestClient) IsOnline() bool {
	return c.online.Load()
}

// Close stops the health check goroutine
func (c *HTTPRestClient) Close() error {
	close(c.stopChan)
	if c.healthTicker != nil {
		c.healthTicker.Stop()
	}
	return nil
}

// handleErrorResponse logs detailed error information from server responses
func (c *HTTPRestClient) handleErrorResponse(resp *http.Response) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("Failed to read error response", slog.Any("error", err))
		return
	}

	var errorResp ErrorResponse
	if err := json.Unmarshal(body, &errorResp); err != nil {
		c.logger.Error("Server error",
			slog.Int("status", resp.StatusCode),
			slog.String("body", string(body)))
		return
	}

	c.logger.Error("Server error",
		slog.Int("status", resp.StatusCode),
		slog.String("code", errorResp.Error.Code),
		slog.String("message", errorResp.Error.Message),
		slog.String("details", errorResp.Error.Details))
}

// periodicHealthCheck runs periodic health checks against the server
func (c *HTTPRestClient) periodicHealthCheck() {
	c.healthTicker = time.NewTicker(30 * time.Second)
	defer c.healthTicker.Stop()

	// Initial health check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := c.TestConnection(ctx); err != nil {
		c.logger.Debug("Initial health check failed", slog.Any("error", err))
	}
	cancel()

	for {
		select {
		case <-c.healthTicker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := c.TestConnection(ctx); err != nil {
				c.setOffline()
				c.logger.Debug("Health check failed", slog.Any("error", err))
			} else {
				c.setOnline()
			}
			cancel()
		case <-c.stopChan:
			return
		}
	}
}

// setOnline marks the client as online
func (c *HTTPRestClient) setOnline() {
	if !c.online.Load() {
		c.online.Store(true)
		c.logger.Info("Server connection restored")
	}
}

// setOffline marks the client as offline
func (c *HTTPRestClient) setOffline() {
	if c.online.Load() {
		c.online.Store(false)
		c.logger.Warn("Server connection lost")
	}
}

// CreateTaskRequest matches the server's expected format exactly
type CreateTaskRequest struct {
	Title              string     `json:"title"`
	Description        string     `json:"description"`
	Type               string     `json:"type"`
	Priority           string     `json:"priority"`
	AcceptanceCriteria []string   `json:"acceptance_criteria,omitempty"`
	Dependencies       []string   `json:"dependencies,omitempty"`
	Tags               []string   `json:"tags,omitempty"`
	Assignee           string     `json:"assignee,omitempty"`
	DueDate            *time.Time `json:"due_date,omitempty"`
	SourcePRDID        string     `json:"source_prd_id,omitempty"`
	Repository         string     `json:"repository,omitempty"`
	Branch             string     `json:"branch,omitempty"`
}

// convertTaskToServerFormat converts CLI task entity to server API format
func (c *HTTPRestClient) convertTaskToServerFormat(task *entities.Task) CreateTaskRequest {
	serverTask := CreateTaskRequest{
		Title:       task.Title,            // Use new Title field
		Description: task.Description,      // Use new Description field
		Type:        task.Type,             // Keep type as-is
		Priority:    string(task.Priority), // Convert Priority enum to string
		Repository:  task.Repository,       // Keep repository as-is
		Tags:        task.Tags,             // Keep tags as-is
	}

	// Fallback to Content if Title is empty (backward compatibility)
	if serverTask.Title == "" && task.Content != "" {
		serverTask.Title = task.Content
	}

	// Map server-compatible fields
	if task.Assignee != "" {
		serverTask.Assignee = task.Assignee
	}
	if len(task.AcceptanceCriteria) > 0 {
		serverTask.AcceptanceCriteria = task.AcceptanceCriteria
	}
	if len(task.Dependencies) > 0 {
		serverTask.Dependencies = task.Dependencies
	}
	if task.SourcePRDID != "" {
		serverTask.SourcePRDID = task.SourcePRDID
	}
	if task.Branch != "" {
		serverTask.Branch = task.Branch
	}
	if task.DueDate != nil {
		serverTask.DueDate = task.DueDate
	}

	// Default type if empty - infer from tags or use default
	if serverTask.Type == "" {
		serverTask.Type = c.inferTypeFromTags(task.Tags)
	}

	return serverTask
}

// convertServerTasksToCliTasks converts server response to CLI task entities
func (c *HTTPRestClient) convertServerTasksToCliTasks(serverResponse map[string]interface{}) []*entities.Task {
	// Try to extract tasks array from response
	var taskData []interface{}

	// Check if response has "data" wrapper
	if data, ok := serverResponse["data"].(map[string]interface{}); ok {
		if tasks, ok := data["tasks"].([]interface{}); ok {
			taskData = tasks
		}
	} else if tasks, ok := serverResponse["tasks"].([]interface{}); ok {
		// Direct tasks array
		taskData = tasks
	} else if taskArray, ok := serverResponse["data"].([]interface{}); ok {
		// Direct array in data field
		taskData = taskArray
	} else {
		// Return empty array if no tasks found
		return []*entities.Task{}
	}

	// Convert each server task to CLI task
	tasks := make([]*entities.Task, 0, len(taskData))
	for _, item := range taskData {
		taskMap, ok := item.(map[string]interface{})
		if !ok {
			continue // Skip invalid items
		}

		task := c.convertServerTaskToCliTask(taskMap)
		if task != nil {
			tasks = append(tasks, task)
		}
	}

	return tasks
}

// convertServerTaskToCliTask converts a single server task to CLI task entity
func (c *HTTPRestClient) convertServerTaskToCliTask(serverTask map[string]interface{}) *entities.Task {
	task := &entities.Task{
		Metadata: make(map[string]interface{}),
	}

	// Extract basic fields
	c.extractBasicFields(serverTask, task)
	
	// Extract repository information
	c.extractRepository(serverTask, task)
	
	// Extract arrays and complex fields
	c.extractArrayFields(serverTask, task)
	
	// Extract timestamps
	c.extractTimestamps(serverTask, task)
	
	// Validate and set defaults
	return c.validateAndSetDefaults(task)
}

// extractBasicFields extracts simple string and scalar fields
func (c *HTTPRestClient) extractBasicFields(serverTask map[string]interface{}, task *entities.Task) {
	if id, ok := serverTask["id"].(string); ok {
		task.ID = id
	}
	if title, ok := serverTask["title"].(string); ok {
		task.Title = title
		task.Content = title // Keep Content for backward compatibility
	}
	if description, ok := serverTask["description"].(string); ok {
		task.Description = description
	}
	if taskType, ok := serverTask["type"].(string); ok {
		task.Type = taskType
	}
	if priority, ok := serverTask["priority"].(string); ok {
		task.Priority = entities.Priority(priority)
	}
	if status, ok := serverTask["status"].(string); ok {
		task.Status = entities.Status(status)
	}
	if assignee, ok := serverTask["assignee"].(string); ok {
		task.Assignee = assignee
	}
	if sourcePRDID, ok := serverTask["source_prd_id"].(string); ok {
		task.SourcePRDID = sourcePRDID
	}
	if branch, ok := serverTask["branch"].(string); ok {
		task.Branch = branch
	}
	if timeTracked, ok := serverTask["time_tracked"].(float64); ok {
		task.TimeTracked = int(timeTracked)
	}
	if confidence, ok := serverTask["confidence"].(float64); ok {
		task.Confidence = confidence
	}
	if complexity, ok := serverTask["complexity"].(string); ok {
		task.Complexity = complexity
	}
	if riskLevel, ok := serverTask["risk_level"].(string); ok {
		task.RiskLevel = riskLevel
	}
}

// extractRepository extracts repository information from various locations
func (c *HTTPRestClient) extractRepository(serverTask map[string]interface{}, task *entities.Task) {
	// Check for repository at top level first
	if repository, ok := serverTask["repository"].(string); ok {
		task.Repository = repository
		return
	}
	
	// Check in metadata.extended_data.repository
	if metadata, ok := serverTask["metadata"].(map[string]interface{}); ok {
		if extendedData, ok := metadata["extended_data"].(map[string]interface{}); ok {
			if repo, ok := extendedData["repository"].(string); ok {
				task.Repository = repo
			}
		}
	}
}

// extractArrayFields extracts array fields from server task
func (c *HTTPRestClient) extractArrayFields(serverTask map[string]interface{}, task *entities.Task) {
	task.AcceptanceCriteria = c.extractStringArray(serverTask["acceptance_criteria"])
	task.Dependencies = c.extractStringArray(serverTask["dependencies"])
	task.BlockedBy = c.extractStringArray(serverTask["blocked_by"])
	task.Blocking = c.extractStringArray(serverTask["blocking"])
	task.Tags = c.extractStringArray(serverTask["tags"])
}

// extractStringArray converts interface{} to []string
func (c *HTTPRestClient) extractStringArray(value interface{}) []string {
	if arr, ok := value.([]interface{}); ok {
		result := make([]string, 0, len(arr))
		for _, item := range arr {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	}
	return nil
}

// extractTimestamps handles both nested and direct timestamp formats
func (c *HTTPRestClient) extractTimestamps(serverTask map[string]interface{}, task *entities.Task) {
	// Try nested timestamps first
	if timestamps, ok := serverTask["timestamps"].(map[string]interface{}); ok {
		c.parseTimestamp(timestamps["created"], &task.CreatedAt)
		c.parseTimestamp(timestamps["updated"], &task.UpdatedAt)
	} else {
		// Fall back to direct timestamp fields
		c.parseTimestamp(serverTask["created_at"], &task.CreatedAt)
	}
	
	c.parseTimestamp(serverTask["updated_at"], &task.UpdatedAt)
	
	// Handle completed_at separately as it's a pointer
	if completedAtStr, ok := serverTask["completed_at"].(string); ok && completedAtStr != "" {
		if completedAt, err := time.Parse(time.RFC3339, completedAtStr); err == nil {
			task.CompletedAt = &completedAt
		}
	}
}

// parseTimestamp parses a timestamp string and sets the target time
func (c *HTTPRestClient) parseTimestamp(value interface{}, target *time.Time) {
	if timeStr, ok := value.(string); ok {
		if parsed, err := time.Parse(time.RFC3339, timeStr); err == nil {
			*target = parsed
		}
	}
}

// validateAndSetDefaults validates required fields and sets defaults
func (c *HTTPRestClient) validateAndSetDefaults(task *entities.Task) *entities.Task {
	// Early return for invalid tasks
	if task.ID == "" || task.Content == "" {
		return nil
	}
	
	// Set defaults for empty fields
	if task.Priority == "" {
		task.Priority = entities.PriorityMedium
	}
	if task.Status == "" {
		task.Status = entities.StatusPending
	}
	if task.Repository == "" {
		task.Repository = "default"
	}
	
	// Set default timestamps if missing
	now := time.Now()
	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
	}
	if task.UpdatedAt.IsZero() {
		task.UpdatedAt = now
	}
	
	return task
}

// inferTypeFromTags infers task type from tags
func (c *HTTPRestClient) inferTypeFromTags(tags []string) string {
	// Map of tags to task types
	typeMap := map[string]string{
		"bug":            "bugfix",
		"bugfix":         "bugfix",
		"fix":            "bugfix",
		"feature":        "implementation",
		"implementation": "implementation",
		"design":         "design",
		"ui":             "design",
		"ux":             "design",
		"test":           "testing",
		"testing":        "testing",
		"qa":             "testing",
		"docs":           "documentation",
		"documentation":  "documentation",
		"readme":         "documentation",
		"research":       "research",
		"analysis":       "analysis",
		"review":         "review",
		"pr":             "review",
		"deploy":         "deployment",
		"deployment":     "deployment",
		"release":        "deployment",
		"architecture":   "architecture",
		"arch":           "architecture",
		"refactor":       "refactoring",
		"refactoring":    "refactoring",
		"integration":    "integration",
		"api":            "integration",
	}

	for _, tag := range tags {
		if taskType, ok := typeMap[strings.ToLower(tag)]; ok {
			return taskType
		}
	}

	// Default to implementation if no matching tag found
	return "implementation"
}
