// Package mcp provides MCP client implementation
// for the lerian-mcp-memory CLI application.
package mcp

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

// RetryConfig defines retry behavior for MCP operations
type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
}

// HTTPMCPClient implements the MCPClient interface using HTTP
type HTTPMCPClient struct {
	baseURL      string
	httpClient   *http.Client
	logger       *slog.Logger
	online       atomic.Bool
	retryConfig  RetryConfig
	healthTicker *time.Ticker
	stopChan     chan struct{}
}

// MCPRequest represents a JSON-RPC request to the MCP server
type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      int         `json:"id"`
}

// MCPResponse represents a JSON-RPC response from the MCP server
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
	ID      int         `json:"id"`
}

// MCPError represents an error in the MCP protocol
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

var (
	ErrMCPOffline  = errors.New("MCP server is offline")
	ErrMCPProtocol = errors.New("MCP protocol error")
	ErrMCPTimeout  = errors.New("MCP request timeout")
)

// NewHTTPMCPClient creates a new HTTP-based MCP client
func NewHTTPMCPClient(config *entities.Config, logger *slog.Logger) ports.MCPClient {
	client := &HTTPMCPClient{
		baseURL: config.Server.URL,
		httpClient: &http.Client{
			Timeout: time.Duration(config.Server.Timeout) * time.Second,
		},
		logger: logger,
		retryConfig: RetryConfig{
			MaxRetries: 3,
			BaseDelay:  100 * time.Millisecond,
			MaxDelay:   5 * time.Second,
		},
		stopChan: make(chan struct{}),
	}

	// Start periodic health checks
	go client.periodicHealthCheck()

	// Temporarily set as online to bypass health check issues
	client.online.Store(true)

	return client
}

// SyncTask syncs a task with the MCP server using the new memory_store tool
func (c *HTTPMCPClient) SyncTask(ctx context.Context, task *entities.Task) error {
	if !c.IsOnline() {
		return ErrMCPOffline
	}

	// Convert task to MCP format
	mcpTask := c.convertToMCPFormat(task)

	request := MCPRequest{
		JSONRPC: "2.0",
		Method:  "memory_create",
		Params: map[string]interface{}{
			"operation": "store_chunk",
			"scope":     "single",
			"options": map[string]interface{}{
				"repository": task.Repository,
				"session_id": c.getSessionID(task.Repository),
				"content":    mcpTask["content"],
				"type":       "task",
				"metadata":   mcpTask,
			},
		},
		ID: 1,
	}

	return c.executeWithRetry(ctx, func() error {
		var response MCPResponse
		return c.sendMCPRequest(ctx, request, &response)
	})
}

// GetTasks retrieves tasks from the MCP server using the new memory_retrieve tool
func (c *HTTPMCPClient) GetTasks(ctx context.Context, repository string) ([]*entities.Task, error) {
	if !c.IsOnline() {
		return nil, ErrMCPOffline
	}

	request := MCPRequest{
		JSONRPC: "2.0",
		Method:  "memory_read",
		Params: map[string]interface{}{
			"operation": "search",
			"scope":     "single",
			"options": map[string]interface{}{
				"repository": repository,
				"session_id": c.getSessionID(repository),
				"query":      "type:task",
				"limit":      100,
			},
		},
		ID: 2,
	}

	var response MCPResponse
	err := c.executeWithRetry(ctx, func() error {
		return c.sendMCPRequest(ctx, request, &response)
	})

	if err != nil {
		return nil, err
	}

	// Convert MCP response to tasks
	return c.convertFromMCPFormat(response.Result)
}

// UpdateTaskStatus updates a task's status on the MCP server using the new memory_store tool
func (c *HTTPMCPClient) UpdateTaskStatus(ctx context.Context, taskID string, status entities.Status) error {
	if !c.IsOnline() {
		return ErrMCPOffline
	}

	// First get the current task to determine project_id
	// For now, use a default project ID - this would need task context in a real implementation
	projectID := c.getProjectIDFromTaskID(taskID)

	request := MCPRequest{
		JSONRPC: "2.0",
		Method:  "memory_update",
		Params: map[string]interface{}{
			"operation": "update_thread",
			"scope":     "single",
			"options": map[string]interface{}{
				"repository": projectID,
				"session_id": c.getSessionID(projectID),
				"thread_id":  taskID,
				"metadata": map[string]interface{}{
					"status": string(status),
				},
			},
		},
		ID: 3,
	}

	return c.executeWithRetry(ctx, func() error {
		var response MCPResponse
		return c.sendMCPRequest(ctx, request, &response)
	})
}

// CallMCPTool calls a generic MCP tool with the given parameters
func (c *HTTPMCPClient) CallMCPTool(ctx context.Context, tool string, params map[string]interface{}) (map[string]interface{}, error) {
	if !c.IsOnline() {
		return nil, ErrMCPOffline
	}

	request := MCPRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      tool,
			"arguments": params,
		},
		ID: int(time.Now().Unix()),
	}

	var response MCPResponse
	err := c.executeWithRetry(ctx, func() error {
		return c.sendMCPRequest(ctx, request, &response)
	})

	if err != nil {
		return nil, err
	}

	// Extract content from MCP response format
	if result, ok := response.Result.(map[string]interface{}); ok {
		// Check if it's an error response
		if isError, ok := result["isError"].(bool); ok && isError {
			if content, ok := result["content"].([]interface{}); ok && len(content) > 0 {
				if textContent, ok := content[0].(map[string]interface{}); ok {
					if text, ok := textContent["text"].(string); ok {
						return nil, fmt.Errorf("MCP tool error: %s", text)
					}
				}
			}
			return nil, fmt.Errorf("MCP tool returned error without details")
		}

		// Extract content for successful responses
		if content, ok := result["content"].([]interface{}); ok && len(content) > 0 {
			if textContent, ok := content[0].(map[string]interface{}); ok {
				if text, ok := textContent["text"].(string); ok {
					// Try to parse as JSON
					var parsedResult map[string]interface{}
					if err := json.Unmarshal([]byte(text), &parsedResult); err == nil {
						return parsedResult, nil
					}
					// Return as text if not JSON
					return map[string]interface{}{"result": text}, nil
				}
			}
		}

		return result, nil
	}

	return map[string]interface{}{"result": response.Result}, nil
}

// QueryIntelligence queries the server's intelligence capabilities using the new memory_analyze tool
func (c *HTTPMCPClient) QueryIntelligence(ctx context.Context, operation string, options map[string]interface{}) (map[string]interface{}, error) {
	if !c.IsOnline() {
		return nil, ErrMCPOffline
	}

	// Extract project_id from options or use default
	projectID, _ := options["project_id"].(string)
	if projectID == "" {
		projectID = "default" // Fallback for legacy calls
	}

	// Build options for the analyze operation
	analyzeOptions := map[string]interface{}{
		"repository": projectID,
		"session_id": c.getSessionID(projectID),
	}

	// Copy other options
	for key, value := range options {
		if key != "project_id" && key != "repository" {
			analyzeOptions[key] = value
		}
	}

	request := MCPRequest{
		JSONRPC: "2.0",
		Method:  "memory_analyze",
		Params: map[string]interface{}{
			"operation": operation,
			"scope":     "single",
			"options":   analyzeOptions,
		},
		ID: 4,
	}

	var response MCPResponse
	err := c.executeWithRetry(ctx, func() error {
		return c.sendMCPRequest(ctx, request, &response)
	})

	if err != nil {
		return nil, err
	}

	// Return the result as a map
	if result, ok := response.Result.(map[string]interface{}); ok {
		return result, nil
	}

	return map[string]interface{}{
		"result": response.Result,
	}, nil
}

// TestConnection tests the connection to the MCP server
func (c *HTTPMCPClient) TestConnection(ctx context.Context) error {
	request := MCPRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "memory_system",
			"arguments": map[string]interface{}{
				"operation": "health",
				"scope":     "system",
				"options":   map[string]interface{}{},
			},
		},
		ID: 99,
	}

	var response MCPResponse
	return c.sendMCPRequest(ctx, request, &response)
}

// IsOnline returns true if the MCP server is reachable
func (c *HTTPMCPClient) IsOnline() bool {
	return c.online.Load()
}

// Close stops the health check goroutine
func (c *HTTPMCPClient) Close() error {
	close(c.stopChan)
	if c.healthTicker != nil {
		c.healthTicker.Stop()
	}
	return nil
}

// sendMCPRequest sends a request to the MCP server
func (c *HTTPMCPClient) sendMCPRequest(ctx context.Context, request MCPRequest, response *MCPResponse) error {
	requestBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/mcp", bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.setOffline()
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		c.setOffline()
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("MCP server error: %d - %s", resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for MCP protocol errors
	if response.Error != nil {
		return fmt.Errorf("%w: %s (code: %d)", ErrMCPProtocol, response.Error.Message, response.Error.Code)
	}

	c.setOnline()
	return nil
}

// executeWithRetry executes an operation with retry logic
func (c *HTTPMCPClient) executeWithRetry(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		if err := operation(); err != nil {
			lastErr = err

			// Don't retry on context cancellation
			if ctx.Err() != nil {
				return ctx.Err()
			}

			// Don't retry on protocol errors
			if errors.Is(err, ErrMCPProtocol) {
				return err
			}

			if attempt < c.retryConfig.MaxRetries {
				delay := c.calculateBackoff(attempt)
				c.logger.Warn("MCP operation failed, retrying",
					slog.Int("attempt", attempt+1),
					slog.Duration("delay", delay),
					slog.Any("error", err))

				select {
				case <-time.After(delay):
					continue
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		} else {
			return nil
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", c.retryConfig.MaxRetries+1, lastErr)
}

// calculateBackoff calculates the backoff delay for a retry attempt
func (c *HTTPMCPClient) calculateBackoff(attempt int) time.Duration {
	delay := c.retryConfig.BaseDelay * time.Duration(1<<attempt)
	if delay > c.retryConfig.MaxDelay {
		delay = c.retryConfig.MaxDelay
	}
	return delay
}

// periodicHealthCheck runs periodic health checks against the MCP server
func (c *HTTPMCPClient) periodicHealthCheck() {
	c.healthTicker = time.NewTicker(30 * time.Second)
	defer c.healthTicker.Stop()

	// Initial health check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := c.TestConnection(ctx); err != nil {
		c.logger.Debug("Initial MCP health check failed", slog.Any("error", err))
	}
	cancel()

	for {
		select {
		case <-c.healthTicker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := c.TestConnection(ctx); err != nil {
				c.setOffline()
				c.logger.Debug("MCP health check failed", slog.Any("error", err))
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
func (c *HTTPMCPClient) setOnline() {
	if !c.online.Load() {
		c.online.Store(true)
		c.logger.Info("MCP connection restored")
	}
}

// setOffline marks the client as offline
func (c *HTTPMCPClient) setOffline() {
	if c.online.Load() {
		c.online.Store(false)
		c.logger.Warn("MCP connection lost")
	}
}

// convertToMCPFormat converts a task to MCP format
func (c *HTTPMCPClient) convertToMCPFormat(task *entities.Task) map[string]interface{} {
	mcpTask := map[string]interface{}{
		"id":         task.ID,
		"content":    task.Content,
		"status":     string(task.Status),
		"priority":   string(task.Priority),
		"repository": task.Repository,
		"created_at": task.CreatedAt.Format(time.RFC3339),
		"updated_at": task.UpdatedAt.Format(time.RFC3339),
		"tags":       task.Tags,
	}

	if task.ParentTaskID != "" {
		mcpTask["parent_task_id"] = task.ParentTaskID
	}

	if task.SessionID != "" {
		mcpTask["session_id"] = task.SessionID
	}

	if task.EstimatedMins > 0 {
		mcpTask["estimated_mins"] = task.EstimatedMins
	}

	if task.ActualMins > 0 {
		mcpTask["actual_mins"] = task.ActualMins
	}

	if task.CompletedAt != nil {
		mcpTask["completed_at"] = task.CompletedAt.Format(time.RFC3339)
	}

	return mcpTask
}

// convertFromMCPFormat converts MCP response data to Task entities
func (c *HTTPMCPClient) convertFromMCPFormat(data interface{}) ([]*entities.Task, error) {
	todos, err := c.extractTodosArray(data)
	if err != nil {
		return nil, err
	}

	var tasks []*entities.Task
	for _, item := range todos {
		taskData, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		task := c.convertSingleTask(taskData)
		if err := task.Validate(); err == nil {
			tasks = append(tasks, task)
		} else {
			c.logger.Warn("Skipping invalid task from MCP",
				slog.String("task_id", task.ID),
				slog.Any("error", err))
		}
	}

	return tasks, nil
}

// extractTodosArray extracts the todos array from various MCP response formats
func (c *HTTPMCPClient) extractTodosArray(data interface{}) ([]interface{}, error) {
	// Handle the response format from memory_tasks/todo_read
	if result, ok := data.(map[string]interface{}); ok {
		if todos, ok := result["todos"].([]interface{}); ok {
			return todos, nil
		}
	}

	// Try direct array format
	if todos, ok := data.([]interface{}); ok {
		return todos, nil
	}

	return nil, errors.New("invalid todos format in MCP response")
}

// convertSingleTask converts a single task data map to Task entity
func (c *HTTPMCPClient) convertSingleTask(taskData map[string]interface{}) *entities.Task {
	task := &entities.Task{}

	c.setRequiredFields(task, taskData)
	c.setTimeFields(task, taskData)
	c.setOptionalFields(task, taskData)

	return task
}

// setRequiredFields sets the required fields for a task
func (c *HTTPMCPClient) setRequiredFields(task *entities.Task, taskData map[string]interface{}) {
	if id, ok := taskData["id"].(string); ok {
		task.ID = id
	}
	if content, ok := taskData["content"].(string); ok {
		task.Content = content
	}
	if status, ok := taskData["status"].(string); ok {
		task.Status = entities.Status(status)
	}
	if priority, ok := taskData["priority"].(string); ok {
		task.Priority = entities.Priority(priority)
	}
	if repo, ok := taskData["repository"].(string); ok {
		task.Repository = repo
	}
}

// setTimeFields sets the time-related fields for a task
func (c *HTTPMCPClient) setTimeFields(task *entities.Task, taskData map[string]interface{}) {
	if createdAt, ok := taskData["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			task.CreatedAt = t
		}
	}
	if updatedAt, ok := taskData["updated_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			task.UpdatedAt = t
		}
	}
	if completedAt, ok := taskData["completed_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, completedAt); err == nil {
			task.CompletedAt = &t
		}
	}
}

// setOptionalFields sets the optional fields for a task
func (c *HTTPMCPClient) setOptionalFields(task *entities.Task, taskData map[string]interface{}) {
	if parentID, ok := taskData["parent_task_id"].(string); ok {
		task.ParentTaskID = parentID
	}
	if sessionID, ok := taskData["session_id"].(string); ok {
		task.SessionID = sessionID
	}

	c.setTagsField(task, taskData)
	c.setNumericFields(task, taskData)
}

// setTagsField sets the tags field for a task
func (c *HTTPMCPClient) setTagsField(task *entities.Task, taskData map[string]interface{}) {
	tagsValue, exists := taskData["tags"]
	if !exists {
		return
	}

	switch tags := tagsValue.(type) {
	case []interface{}:
		for _, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				task.Tags = append(task.Tags, tagStr)
			}
		}
	case []string:
		// Handle direct []string case (for tests that don't go through JSON)
		task.Tags = tags
	}
}

// setNumericFields sets the numeric fields (estimated_mins, actual_mins) for a task
func (c *HTTPMCPClient) setNumericFields(task *entities.Task, taskData map[string]interface{}) {
	// Handle estimated_mins
	if estimatedValue, exists := taskData["estimated_mins"]; exists {
		switch estimated := estimatedValue.(type) {
		case float64:
			task.EstimatedMins = int(estimated)
		case int:
			task.EstimatedMins = estimated
		}
	}

	// Handle actual_mins
	if actualValue, exists := taskData["actual_mins"]; exists {
		switch actual := actualValue.(type) {
		case float64:
			task.ActualMins = int(actual)
		case int:
			task.ActualMins = actual
		}
	}
}

// getSessionID generates or retrieves a session ID for a project
// In the new architecture, session IDs provide expanded access
func (c *HTTPMCPClient) getSessionID(projectID string) string {
	// For now, generate a simple session ID based on project
	// In production, this would be managed properly with session storage
	return fmt.Sprintf("cli_session_%s_%d", projectID, time.Now().Unix()/3600) // Hourly sessions
}

// getProjectIDFromTaskID extracts project ID from task ID
// This is a placeholder - in production, task context would be tracked properly
func (c *HTTPMCPClient) getProjectIDFromTaskID(taskID string) string {
	// For now, return a default project ID
	// In production, this would look up the task's project context
	return "default_project"
}
