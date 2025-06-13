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
		c.handleErrorResponse(resp)
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
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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
	tasks, err := c.convertServerTasksToCliTasks(serverResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to convert server tasks: %w", err)
	}

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

	// Create update payload
	updateData := map[string]interface{}{
		"status":     string(status),
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

// ServerTaskRequest represents the format expected by the server's POST /api/v1/tasks
type ServerTaskRequest struct {
	Title       string   `json:"title"`              // CLI: content → Server: title
	Description string   `json:"description"`        // CLI: metadata → Server: description
	Type        string   `json:"type"`               // CLI: type → Server: type
	Priority    string   `json:"priority"`           // CLI: priority → Server: priority
	Repository  string   `json:"repository"`         // CLI: repository → Server: repository
	Tags        []string `json:"tags,omitempty"`     // CLI: tags → Server: tags
	Assignee    string   `json:"assignee,omitempty"` // CLI: metadata → Server: assignee
}

// convertTaskToServerFormat converts CLI task entity to server API format
func (c *HTTPRestClient) convertTaskToServerFormat(task *entities.Task) ServerTaskRequest {
	serverTask := ServerTaskRequest{
		Title:      task.Content,          // Map content → title
		Type:       task.Type,             // Keep type as-is
		Priority:   string(task.Priority), // Convert Priority enum to string
		Repository: task.Repository,       // Keep repository as-is
		Tags:       task.Tags,             // Keep tags as-is
	}

	// Set description from metadata if available
	if desc, ok := task.GetMetadataString("description"); ok {
		serverTask.Description = desc
	}

	// Set assignee from metadata if available
	if assignee, ok := task.GetMetadataString("assignee"); ok {
		serverTask.Assignee = assignee
	}

	// Default type if empty
	if serverTask.Type == "" {
		serverTask.Type = "general"
	}

	return serverTask
}

// convertServerTasksToCliTasks converts server response to CLI task entities
func (c *HTTPRestClient) convertServerTasksToCliTasks(serverResponse map[string]interface{}) ([]*entities.Task, error) {
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
		return []*entities.Task{}, nil
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

	return tasks, nil
}

// convertServerTaskToCliTask converts a single server task to CLI task entity
func (c *HTTPRestClient) convertServerTaskToCliTask(serverTask map[string]interface{}) *entities.Task {
	task := &entities.Task{
		Metadata: make(map[string]interface{}),
	}

	// Map server fields to CLI fields
	if id, ok := serverTask["id"].(string); ok {
		task.ID = id
	}

	if title, ok := serverTask["title"].(string); ok {
		task.Content = title // Map title → content
	}

	if description, ok := serverTask["description"].(string); ok {
		task.SetMetadata("description", description)
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

	if repository, ok := serverTask["repository"].(string); ok {
		task.Repository = repository
	}

	if assignee, ok := serverTask["assignee"].(string); ok {
		task.SetMetadata("assignee", assignee)
	}

	// Handle tags array
	if tagsInterface, ok := serverTask["tags"]; ok {
		if tagsArray, ok := tagsInterface.([]interface{}); ok {
			tags := make([]string, 0, len(tagsArray))
			for _, tag := range tagsArray {
				if tagStr, ok := tag.(string); ok {
					tags = append(tags, tagStr)
				}
			}
			task.Tags = tags
		}
	}

	// Handle timestamps
	if createdAtStr, ok := serverTask["created_at"].(string); ok {
		if createdAt, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			task.CreatedAt = createdAt
		}
	}

	if updatedAtStr, ok := serverTask["updated_at"].(string); ok {
		if updatedAt, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
			task.UpdatedAt = updatedAt
		}
	}

	if completedAtStr, ok := serverTask["completed_at"].(string); ok && completedAtStr != "" {
		if completedAt, err := time.Parse(time.RFC3339, completedAtStr); err == nil {
			task.CompletedAt = &completedAt
		}
	}

	// Validate and set defaults
	if task.ID == "" {
		return nil // Skip tasks without ID
	}

	if task.Content == "" {
		return nil // Skip tasks without content
	}

	if task.Priority == "" {
		task.Priority = entities.PriorityMedium
	}

	if task.Status == "" {
		task.Status = entities.StatusPending
	}

	// Set creation time if missing
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}

	if task.UpdatedAt.IsZero() {
		task.UpdatedAt = time.Now()
	}

	return task
}
