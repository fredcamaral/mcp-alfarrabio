package notifications

import (
	"encoding/json"
	"time"
)

// Notification represents an MCP notification
type Notification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// NotificationMessage wraps notification data for internal processing
type NotificationMessage struct {
	ClientID     string
	Notification *Notification
	Timestamp    time.Time
}

// ListChangedParams represents parameters for list change notifications
type ListChangedParams struct {
	// Empty for now, but could include details about changes
}

// ResourceChangedParams represents parameters for resource change notifications
type ResourceChangedParams struct {
	URI string `json:"uri"`
}

// LogMessageParams represents parameters for log notifications
type LogMessageParams struct {
	Level   string `json:"level"`
	Logger  string `json:"logger,omitempty"`
	Message string `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ProgressParams represents parameters for progress notifications
type ProgressParams struct {
	ProgressToken string  `json:"progressToken"`
	Progress      float64 `json:"progress"` // 0.0 to 1.0
	Message       string  `json:"message,omitempty"`
}

// NotificationType constants
const (
	// Standard MCP notifications
	NotificationToolsListChanged     = "notifications/tools/list_changed"
	NotificationResourcesListChanged = "notifications/resources/list_changed"
	NotificationPromptsListChanged   = "notifications/prompts/list_changed"
	NotificationRootsListChanged     = "notifications/roots/list_changed"
	
	// Resource-specific notifications
	NotificationResourceChanged = "notifications/resource/changed"
	
	// Progress notifications
	NotificationProgress = "notifications/progress"
	
	// Logging notifications
	NotificationLogMessage = "notifications/log"
	
	// Custom notifications
	NotificationCustom = "notifications/custom"
)