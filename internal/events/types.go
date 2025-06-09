// Package events provides event-driven architecture for real-time updates
package events

import (
	"encoding/json"
	"fmt"
	"time"
)

// Event represents a system event with metadata and payload
type Event struct {
	// Core event information
	ID        string    `json:"id"`
	Type      EventType `json:"type"`
	Action    string    `json:"action"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"`

	// Event routing and filtering
	Repository string   `json:"repository,omitempty"`
	SessionID  string   `json:"session_id,omitempty"`
	UserID     string   `json:"user_id,omitempty"`
	ClientID   string   `json:"client_id,omitempty"`
	Tags       []string `json:"tags,omitempty"`

	// Event payload and metadata
	Payload  interface{}            `json:"payload"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Event lifecycle
	TTL       time.Duration `json:"ttl,omitempty"`
	ExpiresAt *time.Time    `json:"expires_at,omitempty"`
	Retry     *RetryConfig  `json:"retry,omitempty"`

	// Event ordering and relationships
	SequenceNumber int64  `json:"sequence_number,omitempty"`
	CorrelationID  string `json:"correlation_id,omitempty"`
	CausationID    string `json:"causation_id,omitempty"`
	ParentID       string `json:"parent_id,omitempty"`

	// Processing information
	ProcessedAt    *time.Time `json:"processed_at,omitempty"`
	DeliveredAt    *time.Time `json:"delivered_at,omitempty"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
}

// EventType represents different types of events in the system
type EventType string

const (
	// Memory and data events
	EventTypeMemoryUpdate    EventType = "memory.update"
	EventTypeMemoryCreated   EventType = "memory.created"
	EventTypeMemoryDeleted   EventType = "memory.deleted"
	EventTypeChunkProcessed  EventType = "chunk.processed"
	EventTypeSearchPerformed EventType = "search.performed"

	// Task and workflow events
	EventTypeTaskCreated       EventType = "task.created"
	EventTypeTaskUpdated       EventType = "task.updated"
	EventTypeTaskCompleted     EventType = "task.completed"
	EventTypeTaskDeleted       EventType = "task.deleted"
	EventTypeWorkflowStarted   EventType = "workflow.started"
	EventTypeWorkflowCompleted EventType = "workflow.completed"

	// Connection and session events
	EventTypeConnectionOpened   EventType = "connection.opened"
	EventTypeConnectionClosed   EventType = "connection.closed"
	EventTypeSessionStarted     EventType = "session.started"
	EventTypeSessionEnded       EventType = "session.ended"
	EventTypeClientRegistered   EventType = "client.registered"
	EventTypeClientDeregistered EventType = "client.deregistered"

	// System and monitoring events
	EventTypeSystemAlert          EventType = "system.alert"
	EventTypeHealthCheck          EventType = "system.health_check"
	EventTypePerformanceMetric    EventType = "system.performance_metric"
	EventTypeErrorOccurred        EventType = "system.error"
	EventTypeConfigurationChanged EventType = "system.config_changed"

	// Security events
	EventTypeSecurityThreat       EventType = "security.threat_detected"
	EventTypeAuthenticationFailed EventType = "security.auth_failed"
	EventTypeAccessDenied         EventType = "security.access_denied"
	EventTypeAuditLog             EventType = "security.audit_log"

	// Notification events
	EventTypeNotificationSent      EventType = "notification.sent"
	EventTypeNotificationDelivered EventType = "notification.delivered"
	EventTypeNotificationFailed    EventType = "notification.failed"

	// Custom and extension events
	EventTypeCustom    EventType = "custom"
	EventTypeExtension EventType = "extension"
)

// RetryConfig defines retry behavior for event processing
type RetryConfig struct {
	MaxAttempts   int           `json:"max_attempts"`
	BackoffPolicy BackoffPolicy `json:"backoff_policy"`
	InitialDelay  time.Duration `json:"initial_delay"`
	MaxDelay      time.Duration `json:"max_delay"`
	Multiplier    float64       `json:"multiplier"`
}

// BackoffPolicy defines retry backoff strategies
type BackoffPolicy string

const (
	BackoffLinear      BackoffPolicy = "linear"
	BackoffExponential BackoffPolicy = "exponential"
	BackoffFixed       BackoffPolicy = "fixed"
	BackoffNone        BackoffPolicy = "none"
)

// EventFilter defines criteria for filtering events
type EventFilter struct {
	Types        []EventType `json:"types,omitempty"`
	Actions      []string    `json:"actions,omitempty"`
	Sources      []string    `json:"sources,omitempty"`
	Repositories []string    `json:"repositories,omitempty"`
	SessionIDs   []string    `json:"session_ids,omitempty"`
	UserIDs      []string    `json:"user_ids,omitempty"`
	ClientIDs    []string    `json:"client_ids,omitempty"`
	Tags         []string    `json:"tags,omitempty"`

	// Time-based filtering
	After  *time.Time `json:"after,omitempty"`
	Before *time.Time `json:"before,omitempty"`

	// Advanced filtering
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CustomFilter func(*Event) bool      `json:"-"`
}

// EventSubscription represents a subscription to events
type EventSubscription struct {
	ID           string            `json:"id"`
	SubscriberID string            `json:"subscriber_id"`
	Filter       EventFilter       `json:"filter"`
	DeliveryMode DeliveryMode      `json:"delivery_mode"`
	CreatedAt    time.Time         `json:"created_at"`
	LastDelivery *time.Time        `json:"last_delivery,omitempty"`
	Statistics   SubscriptionStats `json:"statistics"`
}

// DeliveryMode defines how events are delivered to subscribers
type DeliveryMode string

const (
	DeliveryModeImmediate DeliveryMode = "immediate"
	DeliveryModeBatched   DeliveryMode = "batched"
	DeliveryModeQueued    DeliveryMode = "queued"
	DeliveryModeWebSocket DeliveryMode = "websocket"
	DeliveryModePush      DeliveryMode = "push"
	DeliveryModeWebhook   DeliveryMode = "webhook"
)

// SubscriptionStats tracks subscription performance
type SubscriptionStats struct {
	EventsReceived  int64         `json:"events_received"`
	EventsDelivered int64         `json:"events_delivered"`
	EventsFailed    int64         `json:"events_failed"`
	LastEventTime   *time.Time    `json:"last_event_time"`
	AverageLatency  time.Duration `json:"average_latency"`
	DeliverySuccess float64       `json:"delivery_success_rate"`
}

// EventMetadata provides additional context for events
type EventMetadata struct {
	// Request context
	RequestID string `json:"request_id,omitempty"`
	TraceID   string `json:"trace_id,omitempty"`
	SpanID    string `json:"span_id,omitempty"`

	// User context
	UserAgent string `json:"user_agent,omitempty"`
	IPAddress string `json:"ip_address,omitempty"`

	// Performance metrics
	ProcessingTime time.Duration `json:"processing_time,omitempty"`
	QueueTime      time.Duration `json:"queue_time,omitempty"`

	// Business context
	EntityID      string `json:"entity_id,omitempty"`
	EntityType    string `json:"entity_type,omitempty"`
	EntityVersion string `json:"entity_version,omitempty"`

	// Custom fields
	Custom map[string]interface{} `json:"custom,omitempty"`
}

// EventStatus represents the processing status of an event
type EventStatus string

const (
	EventStatusPending    EventStatus = "pending"
	EventStatusProcessing EventStatus = "processing"
	EventStatusProcessed  EventStatus = "processed"
	EventStatusDelivered  EventStatus = "delivered"
	EventStatusFailed     EventStatus = "failed"
	EventStatusExpired    EventStatus = "expired"
	EventStatusCancelled  EventStatus = "cancelled"
)

// EventBatch represents a collection of related events
type EventBatch struct {
	ID          string                 `json:"id"`
	Events      []*Event               `json:"events"`
	CreatedAt   time.Time              `json:"created_at"`
	ProcessedAt *time.Time             `json:"processed_at,omitempty"`
	Size        int                    `json:"size"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// NewEvent creates a new event with default values
func NewEvent(eventType EventType, action string, payload interface{}) *Event {
	now := time.Now()

	return &Event{
		ID:        generateEventID(),
		Type:      eventType,
		Action:    action,
		Version:   "1.0",
		Timestamp: now,
		Source:    "mcp-memory-server",
		Payload:   payload,
		Metadata:  make(map[string]interface{}),
	}
}

// NewMemoryUpdateEvent creates a memory update event
func NewMemoryUpdateEvent(chunkID, repository, sessionID string, content interface{}) *Event {
	event := NewEvent(EventTypeMemoryUpdate, "chunk_updated", map[string]interface{}{
		"chunk_id": chunkID,
		"content":  content,
	})

	event.Repository = repository
	event.SessionID = sessionID
	event.Tags = []string{"memory", "update"}

	return event
}

// NewTaskEvent creates a task-related event
func NewTaskEvent(eventType EventType, action, taskID string, taskData interface{}) *Event {
	event := NewEvent(eventType, action, map[string]interface{}{
		"task_id": taskID,
		"data":    taskData,
	})

	event.Tags = []string{"task", action}

	return event
}

// NewSystemEvent creates a system event
func NewSystemEvent(action string, severity string, message string, details interface{}) *Event {
	event := NewEvent(EventTypeSystemAlert, action, map[string]interface{}{
		"severity": severity,
		"message":  message,
		"details":  details,
	})

	event.Tags = []string{"system", severity}

	return event
}

// NewConnectionEvent creates a connection-related event
func NewConnectionEvent(eventType EventType, clientID, sessionID string, metadata map[string]interface{}) *Event {
	event := NewEvent(eventType, "connection_change", map[string]interface{}{
		"client_id": clientID,
		"metadata":  metadata,
	})

	event.ClientID = clientID
	event.SessionID = sessionID
	event.Tags = []string{"connection"}

	return event
}

// WithCorrelation sets correlation and causation IDs for event tracking
func (e *Event) WithCorrelation(correlationID, causationID string) *Event {
	e.CorrelationID = correlationID
	e.CausationID = causationID
	return e
}

// WithMetadata adds metadata to the event
func (e *Event) WithMetadata(key string, value interface{}) *Event {
	if e.Metadata == nil {
		e.Metadata = make(map[string]interface{})
	}
	e.Metadata[key] = value
	return e
}

// WithTags adds tags to the event
func (e *Event) WithTags(tags ...string) *Event {
	e.Tags = append(e.Tags, tags...)
	return e
}

// WithTTL sets the time-to-live for the event
func (e *Event) WithTTL(ttl time.Duration) *Event {
	e.TTL = ttl
	expiresAt := e.Timestamp.Add(ttl)
	e.ExpiresAt = &expiresAt
	return e
}

// WithRetry configures retry behavior for the event
func (e *Event) WithRetry(maxAttempts int, backoffPolicy BackoffPolicy, initialDelay time.Duration) *Event {
	e.Retry = &RetryConfig{
		MaxAttempts:   maxAttempts,
		BackoffPolicy: backoffPolicy,
		InitialDelay:  initialDelay,
		MaxDelay:      5 * time.Minute,
		Multiplier:    2.0,
	}
	return e
}

// IsExpired checks if the event has expired
func (e *Event) IsExpired() bool {
	if e.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*e.ExpiresAt)
}

// Matches checks if the event matches the given filter
func (e *Event) Matches(filter *EventFilter) bool {
	// Check event types
	if len(filter.Types) > 0 {
		matched := false
		for _, t := range filter.Types {
			if e.Type == t {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check actions
	if len(filter.Actions) > 0 {
		matched := false
		for _, action := range filter.Actions {
			if e.Action == action {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check sources
	if len(filter.Sources) > 0 {
		matched := false
		for _, source := range filter.Sources {
			if e.Source == source {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check repositories
	if len(filter.Repositories) > 0 && e.Repository != "" {
		matched := false
		for _, repo := range filter.Repositories {
			if e.Repository == repo {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check session IDs
	if len(filter.SessionIDs) > 0 && e.SessionID != "" {
		matched := false
		for _, sessionID := range filter.SessionIDs {
			if e.SessionID == sessionID {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check user IDs
	if len(filter.UserIDs) > 0 && e.UserID != "" {
		matched := false
		for _, userID := range filter.UserIDs {
			if e.UserID == userID {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check client IDs
	if len(filter.ClientIDs) > 0 && e.ClientID != "" {
		matched := false
		for _, clientID := range filter.ClientIDs {
			if e.ClientID == clientID {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check tags
	if len(filter.Tags) > 0 && len(e.Tags) > 0 {
		matched := false
		for _, filterTag := range filter.Tags {
			for _, eventTag := range e.Tags {
				if filterTag == eventTag {
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check time range
	if filter.After != nil && e.Timestamp.Before(*filter.After) {
		return false
	}

	if filter.Before != nil && e.Timestamp.After(*filter.Before) {
		return false
	}

	// Check metadata filters
	if len(filter.Metadata) > 0 {
		for key, expectedValue := range filter.Metadata {
			if actualValue, exists := e.Metadata[key]; !exists || actualValue != expectedValue {
				return false
			}
		}
	}

	// Check custom filter
	if filter.CustomFilter != nil {
		return filter.CustomFilter(e)
	}

	return true
}

// ToJSON converts the event to JSON
func (e *Event) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// FromJSON creates an event from JSON data
func FromJSON(data []byte) (*Event, error) {
	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}
	return &event, nil
}

// Clone creates a deep copy of the event
func (e *Event) Clone() *Event {
	clone := *e

	// Deep copy slices and maps
	if e.Tags != nil {
		clone.Tags = make([]string, len(e.Tags))
		copy(clone.Tags, e.Tags)
	}

	if e.Metadata != nil {
		clone.Metadata = make(map[string]interface{})
		for k, v := range e.Metadata {
			clone.Metadata[k] = v
		}
	}

	if e.Retry != nil {
		retryClone := *e.Retry
		clone.Retry = &retryClone
	}

	return &clone
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return fmt.Sprintf("evt_%d_%d", time.Now().UnixNano(), time.Now().Nanosecond())
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:   3,
		BackoffPolicy: BackoffExponential,
		InitialDelay:  time.Second,
		MaxDelay:      5 * time.Minute,
		Multiplier:    2.0,
	}
}

// EventPriority represents event processing priority
type EventPriority int

const (
	PriorityLow      EventPriority = 1
	PriorityNormal   EventPriority = 2
	PriorityHigh     EventPriority = 3
	PriorityCritical EventPriority = 4
)

// GetPriority returns the priority of an event based on its type and metadata
func (e *Event) GetPriority() EventPriority {
	// Critical events
	switch e.Type {
	case EventTypeSystemAlert, EventTypeSecurityThreat, EventTypeErrorOccurred:
		return PriorityCritical
	case EventTypeAuthenticationFailed, EventTypeAccessDenied:
		return PriorityHigh
	case EventTypeMemoryUpdate, EventTypeTaskUpdated, EventTypeConnectionOpened, EventTypeConnectionClosed:
		return PriorityNormal
	default:
		return PriorityLow
	}
}
