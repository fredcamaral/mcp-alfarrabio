package subscriptions

import (
	"encoding/json"
	"time"
)

// Subscription represents an active subscription
type Subscription struct {
	ID         string          `json:"id"`
	ClientID   string          `json:"clientId"`
	Method     string          `json:"method"`
	Params     json.RawMessage `json:"params"`
	CreatedAt  time.Time       `json:"createdAt"`
	LastActive time.Time       `json:"lastActive"`
}

// SubscriptionRequest represents a subscription request
type SubscriptionRequest struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// SubscriptionResponse represents a subscription response
type SubscriptionResponse struct {
	SubscriptionID string `json:"subscriptionId"`
}

// UnsubscribeRequest represents an unsubscribe request
type UnsubscribeRequest struct {
	SubscriptionID string `json:"subscriptionId"`
}

// ResourceSubscriptionParams represents parameters for resource subscription
type ResourceSubscriptionParams struct {
	URI string `json:"uri"`
}

// Event represents a subscription event
type Event struct {
	SubscriptionID string      `json:"subscriptionId"`
	Type           string      `json:"type"`
	Data           interface{} `json:"data"`
	Timestamp      time.Time   `json:"timestamp"`
}

// EventType constants
const (
	EventTypeResourceChanged   = "resource/changed"
	EventTypeResourceDeleted   = "resource/deleted"
	EventTypeToolsListChanged  = "tools/list_changed"
	EventTypeResourcesListChanged = "resources/list_changed"
	EventTypePromptsListChanged   = "prompts/list_changed"
	EventTypeRootsListChanged     = "roots/list_changed"
)

// Filter represents subscription filter criteria
type Filter struct {
	// For resource subscriptions
	URIPattern string `json:"uriPattern,omitempty"`
	
	// For list change subscriptions
	IncludeDetails bool `json:"includeDetails,omitempty"`
}

// ListChangeEvent represents a list change event
type ListChangeEvent struct {
	Added   []string `json:"added,omitempty"`
	Removed []string `json:"removed,omitempty"`
	Updated []string `json:"updated,omitempty"`
}

// ResourceChangeEvent represents a resource change event
type ResourceChangeEvent struct {
	URI        string      `json:"uri"`
	ChangeType string      `json:"changeType"` // "created", "updated", "deleted"
	Content    interface{} `json:"content,omitempty"`
}