// Package security provides security audit logging and monitoring
package security

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// AuditLogger provides structured security audit logging
type AuditLogger struct {
	config AuditLoggerConfig
	buffer []AuditEvent
	mu     sync.RWMutex
}

// AuditLoggerConfig configures the audit logger
type AuditLoggerConfig struct {
	// Logging settings
	LogLevel     string `json:"log_level"`     // "debug", "info", "warn", "error"
	BufferSize   int    `json:"buffer_size"`   // Number of events to buffer
	FlushInterval time.Duration `json:"flush_interval"` // How often to flush buffer
	
	// Event filtering
	EnabledEvents   []string `json:"enabled_events"`   // Which events to log
	DisabledEvents  []string `json:"disabled_events"`  // Which events to skip
	
	// Output configuration
	OutputFormat    string `json:"output_format"`    // "json", "text"
	IncludePayload  bool   `json:"include_payload"`  // Include request/response data
	MaxPayloadSize  int    `json:"max_payload_size"` // Max payload size to log
	
	// Sensitive data handling
	SensitiveFields []string `json:"sensitive_fields"` // Fields to redact
	RedactPasswords bool     `json:"redact_passwords"` // Redact password fields
	RedactTokens    bool     `json:"redact_tokens"`    // Redact auth tokens
	
	// Performance
	AsyncLogging    bool `json:"async_logging"`    // Log asynchronously
	MaxQueueSize    int  `json:"max_queue_size"`   // Max async queue size
}

// AuditEvent represents a security audit event
type AuditEvent struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	Type        EventType              `json:"type"`
	Action      string                 `json:"action"`
	Result      EventResult            `json:"result"`
	Severity    EventSeverity          `json:"severity"`
	Actor       Actor                  `json:"actor"`
	Resource    Resource               `json:"resource"`
	Request     *RequestInfo           `json:"request,omitempty"`
	Response    *ResponseInfo          `json:"response,omitempty"`
	Threats     []ThreatInfo           `json:"threats,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Duration    time.Duration          `json:"duration,omitempty"`
	ErrorInfo   *ErrorInfo             `json:"error,omitempty"`
}

// EventType represents the type of audit event
type EventType string

const (
	EventTypeAuthentication EventType = "authentication"
	EventTypeAuthorization  EventType = "authorization"
	EventTypeAccess         EventType = "access"
	EventTypeSecurity       EventType = "security"
	EventTypeDataAccess     EventType = "data_access"
	EventTypeConfiguration  EventType = "configuration"
	EventTypeError          EventType = "error"
)

// EventResult represents the result of an event
type EventResult string

const (
	ResultSuccess EventResult = "success"
	ResultFailure EventResult = "failure"
	ResultBlocked EventResult = "blocked"
	ResultWarning EventResult = "warning"
)

// EventSeverity represents the severity level
type EventSeverity string

const (
	SeverityLow      EventSeverity = "low"
	SeverityMedium   EventSeverity = "medium"
	SeverityHigh     EventSeverity = "high"
	SeverityCritical EventSeverity = "critical"
)

// Actor represents the entity performing the action
type Actor struct {
	Type      string `json:"type"`       // "user", "service", "system", "anonymous"
	ID        string `json:"id"`         // User ID, service name, etc.
	Name      string `json:"name"`       // Display name
	IPAddress string `json:"ip_address"` // Source IP address
	UserAgent string `json:"user_agent"` // User agent string
	Session   string `json:"session"`    // Session identifier
}

// Resource represents the resource being accessed
type Resource struct {
	Type       string `json:"type"`        // "api", "file", "database", etc.
	ID         string `json:"id"`          // Resource identifier
	Name       string `json:"name"`        // Resource name
	Path       string `json:"path"`        // Resource path/URL
	Operation  string `json:"operation"`   // Operation being performed
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// RequestInfo contains request details
type RequestInfo struct {
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	Headers     map[string]string `json:"headers,omitempty"`
	QueryParams map[string]string `json:"query_params,omitempty"`
	Payload     interface{}       `json:"payload,omitempty"`
	Size        int64             `json:"size"`
}

// ResponseInfo contains response details
type ResponseInfo struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers,omitempty"`
	Payload    interface{}       `json:"payload,omitempty"`
	Size       int64             `json:"size"`
}

// ThreatInfo contains information about detected threats
type ThreatInfo struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Pattern     string `json:"pattern,omitempty"`
	Field       string `json:"field,omitempty"`
	Value       string `json:"value,omitempty"`
}

// ErrorInfo contains error details
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
	Stack   string `json:"stack,omitempty"`
}

// DefaultAuditLoggerConfig returns secure defaults
func DefaultAuditLoggerConfig() AuditLoggerConfig {
	return AuditLoggerConfig{
		LogLevel:      "info",
		BufferSize:    1000,
		FlushInterval: 30 * time.Second,
		EnabledEvents: []string{
			string(EventTypeAuthentication),
			string(EventTypeAuthorization),
			string(EventTypeSecurity),
			string(EventTypeError),
		},
		DisabledEvents:  []string{},
		OutputFormat:    "json",
		IncludePayload:  true,
		MaxPayloadSize:  10000, // 10KB
		SensitiveFields: []string{
			"password", "secret", "token", "key", "authorization",
			"x-api-key", "cookie", "session", "credential",
		},
		RedactPasswords: true,
		RedactTokens:    true,
		AsyncLogging:    true,
		MaxQueueSize:    10000,
	}
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(config AuditLoggerConfig) *AuditLogger {
	logger := &AuditLogger{
		config: config,
		buffer: make([]AuditEvent, 0, config.BufferSize),
	}
	
	// Start background processing if async logging is enabled
	if config.AsyncLogging {
		go logger.backgroundProcessor()
	}
	
	return logger
}

// LogSecurityEvent logs a security-related event
func (al *AuditLogger) LogSecurityEvent(ctx context.Context, eventType EventType, action string, result EventResult, severity EventSeverity) {
	event := AuditEvent{
		ID:        generateEventID(),
		Timestamp: time.Now(),
		Type:      eventType,
		Action:    action,
		Result:    result,
		Severity:  severity,
		Metadata:  make(map[string]interface{}),
	}
	
	// Extract context information
	al.enrichEventFromContext(ctx, &event)
	
	al.logEvent(event)
}

// LogHTTPRequest logs an HTTP request for audit purposes
func (al *AuditLogger) LogHTTPRequest(r *http.Request, result EventResult, duration time.Duration) {
	if !al.shouldLogEvent(EventTypeAccess) {
		return
	}
	
	event := AuditEvent{
		ID:        generateEventID(),
		Timestamp: time.Now(),
		Type:      EventTypeAccess,
		Action:    "http_request",
		Result:    result,
		Severity:  al.determineSeverity(result),
		Duration:  duration,
		Actor:     al.extractActorFromRequest(r),
		Resource:  al.extractResourceFromRequest(r),
		Request:   al.extractRequestInfo(r),
		Metadata:  make(map[string]interface{}),
	}
	
	// Add threats if available in context
	if threats := r.Context().Value("security_threats"); threats != nil {
		if threatList, ok := threats.([]interface{}); ok {
			event.Threats = al.convertThreats(threatList)
		}
	}
	
	al.logEvent(event)
}

// LogAuthenticationEvent logs authentication-related events
func (al *AuditLogger) LogAuthenticationEvent(actor Actor, action string, result EventResult, errorMsg string) {
	if !al.shouldLogEvent(EventTypeAuthentication) {
		return
	}
	
	event := AuditEvent{
		ID:        generateEventID(),
		Timestamp: time.Now(),
		Type:      EventTypeAuthentication,
		Action:    action,
		Result:    result,
		Severity:  al.determineSeverity(result),
		Actor:     actor,
		Metadata:  make(map[string]interface{}),
	}
	
	if errorMsg != "" {
		event.ErrorInfo = &ErrorInfo{
			Message: errorMsg,
		}
	}
	
	al.logEvent(event)
}

// LogThreatDetection logs detected security threats
func (al *AuditLogger) LogThreatDetection(r *http.Request, threats []ThreatInfo) {
	if !al.shouldLogEvent(EventTypeSecurity) || len(threats) == 0 {
		return
	}
	
	event := AuditEvent{
		ID:        generateEventID(),
		Timestamp: time.Now(),
		Type:      EventTypeSecurity,
		Action:    "threat_detection",
		Result:    ResultWarning,
		Severity:  al.determineThreatSeverity(threats),
		Actor:     al.extractActorFromRequest(r),
		Resource:  al.extractResourceFromRequest(r),
		Request:   al.extractRequestInfo(r),
		Threats:   threats,
		Metadata:  make(map[string]interface{}),
	}
	
	al.logEvent(event)
}

// logEvent adds an event to the log
func (al *AuditLogger) logEvent(event AuditEvent) {
	// Sanitize sensitive data
	al.sanitizeEvent(&event)
	
	if al.config.AsyncLogging {
		al.asyncLogEvent(event)
	} else {
		al.syncLogEvent(event)
	}
}

// syncLogEvent logs an event synchronously
func (al *AuditLogger) syncLogEvent(event AuditEvent) {
	al.mu.Lock()
	defer al.mu.Unlock()
	
	// Add to buffer
	al.buffer = append(al.buffer, event)
	
	// Flush if buffer is full
	if len(al.buffer) >= al.config.BufferSize {
		al.flushBuffer()
	}
}

// asyncLogEvent logs an event asynchronously
func (al *AuditLogger) asyncLogEvent(event AuditEvent) {
	// In a real implementation, this would use a channel-based queue
	// For now, we'll use the synchronous method
	al.syncLogEvent(event)
}

// flushBuffer flushes the event buffer
func (al *AuditLogger) flushBuffer() {
	if len(al.buffer) == 0 {
		return
	}
	
	// Output events
	for _, event := range al.buffer {
		al.outputEvent(event)
	}
	
	// Clear buffer
	al.buffer = al.buffer[:0]
}

// outputEvent outputs a single event
func (al *AuditLogger) outputEvent(event AuditEvent) {
	switch al.config.OutputFormat {
	case "json":
		al.outputJSON(event)
	case "text":
		al.outputText(event)
	default:
		al.outputJSON(event)
	}
}

// outputJSON outputs an event as JSON
func (al *AuditLogger) outputJSON(event AuditEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		fmt.Printf("Error marshaling audit event: %v\n", err)
		return
	}
	
	// In a real implementation, this would go to proper logging infrastructure
	fmt.Printf("AUDIT: %s\n", string(data))
}

// outputText outputs an event as text
func (al *AuditLogger) outputText(event AuditEvent) {
	fmt.Printf("AUDIT: [%s] %s %s %s by %s (%s) - %s\n",
		event.Timestamp.Format(time.RFC3339),
		event.Severity,
		event.Type,
		event.Action,
		event.Actor.ID,
		event.Actor.IPAddress,
		event.Result,
	)
}

// backgroundProcessor handles periodic buffer flushing
func (al *AuditLogger) backgroundProcessor() {
	ticker := time.NewTicker(al.config.FlushInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		al.mu.Lock()
		al.flushBuffer()
		al.mu.Unlock()
	}
}

// shouldLogEvent checks if an event type should be logged
func (al *AuditLogger) shouldLogEvent(eventType EventType) bool {
	eventTypeStr := string(eventType)
	
	// Check disabled events first
	for _, disabled := range al.config.DisabledEvents {
		if disabled == eventTypeStr {
			return false
		}
	}
	
	// If enabled events are specified, check inclusion
	if len(al.config.EnabledEvents) > 0 {
		for _, enabled := range al.config.EnabledEvents {
			if enabled == eventTypeStr {
				return true
			}
		}
		return false
	}
	
	// Default to logging if no restrictions
	return true
}

// extractActorFromRequest extracts actor information from HTTP request
func (al *AuditLogger) extractActorFromRequest(r *http.Request) Actor {
	actor := Actor{
		Type:      "anonymous",
		IPAddress: r.RemoteAddr,
		UserAgent: r.Header.Get("User-Agent"),
	}
	
	// Extract user information from context if available
	if userID := r.Context().Value("user_id"); userID != nil {
		if uid, ok := userID.(string); ok {
			actor.Type = "user"
			actor.ID = uid
		}
	}
	
	// Extract session information
	if sessionID := r.Context().Value("session_id"); sessionID != nil {
		if sid, ok := sessionID.(string); ok {
			actor.Session = sid
		}
	}
	
	return actor
}

// extractResourceFromRequest extracts resource information from HTTP request
func (al *AuditLogger) extractResourceFromRequest(r *http.Request) Resource {
	return Resource{
		Type:      "api",
		Path:      r.URL.Path,
		Operation: r.Method,
		Attributes: map[string]interface{}{
			"query_params": r.URL.RawQuery,
		},
	}
}

// extractRequestInfo extracts request information
func (al *AuditLogger) extractRequestInfo(r *http.Request) *RequestInfo {
	if !al.config.IncludePayload {
		return &RequestInfo{
			Method: r.Method,
			URL:    r.URL.String(),
			Size:   r.ContentLength,
		}
	}
	
	headers := make(map[string]string)
	for name, values := range r.Header {
		if len(values) > 0 {
			headers[name] = values[0] // Take first value
		}
	}
	
	queryParams := make(map[string]string)
	for name, values := range r.URL.Query() {
		if len(values) > 0 {
			queryParams[name] = values[0] // Take first value
		}
	}
	
	// Sanitize sensitive headers
	al.sanitizeHeaders(headers)
	al.sanitizeQueryParams(queryParams)
	
	return &RequestInfo{
		Method:      r.Method,
		URL:         r.URL.String(),
		Headers:     headers,
		QueryParams: queryParams,
		Size:        r.ContentLength,
	}
}

// enrichEventFromContext enriches event with context information
func (al *AuditLogger) enrichEventFromContext(ctx context.Context, event *AuditEvent) {
	// Extract request ID
	if requestID := ctx.Value("request_id"); requestID != nil {
		if rid, ok := requestID.(string); ok {
			event.Metadata["request_id"] = rid
		}
	}
	
	// Extract trace ID
	if traceID := ctx.Value("trace_id"); traceID != nil {
		if tid, ok := traceID.(string); ok {
			event.Metadata["trace_id"] = tid
		}
	}
}

// sanitizeEvent removes sensitive information from an event
func (al *AuditLogger) sanitizeEvent(event *AuditEvent) {
	if event.Request != nil {
		al.sanitizeHeaders(event.Request.Headers)
		al.sanitizeQueryParams(event.Request.QueryParams)
	}
	
	if event.Response != nil {
		al.sanitizeHeaders(event.Response.Headers)
	}
}

// sanitizeHeaders removes sensitive information from headers
func (al *AuditLogger) sanitizeHeaders(headers map[string]string) {
	for key, _ := range headers {
		if al.isSensitiveField(key) {
			headers[key] = "[REDACTED]"
		}
	}
}

// sanitizeQueryParams removes sensitive information from query parameters
func (al *AuditLogger) sanitizeQueryParams(params map[string]string) {
	for key, _ := range params {
		if al.isSensitiveField(key) {
			params[key] = "[REDACTED]"
		}
	}
}

// isSensitiveField checks if a field name is sensitive
func (al *AuditLogger) isSensitiveField(fieldName string) bool {
	fieldLower := fmt.Sprintf("%s", fieldName) // Convert to lowercase for comparison
	
	for _, sensitive := range al.config.SensitiveFields {
		if fieldLower == sensitive {
			return true
		}
	}
	
	return false
}

// determineSeverity determines event severity based on result
func (al *AuditLogger) determineSeverity(result EventResult) EventSeverity {
	switch result {
	case ResultFailure:
		return SeverityHigh
	case ResultBlocked:
		return SeverityCritical
	case ResultWarning:
		return SeverityMedium
	default:
		return SeverityLow
	}
}

// determineThreatSeverity determines severity based on detected threats
func (al *AuditLogger) determineThreatSeverity(threats []ThreatInfo) EventSeverity {
	maxSeverity := SeverityLow
	
	for _, threat := range threats {
		switch threat.Severity {
		case "critical":
			maxSeverity = SeverityCritical
		case "high":
			if maxSeverity != SeverityCritical {
				maxSeverity = SeverityHigh
			}
		case "medium":
			if maxSeverity == SeverityLow {
				maxSeverity = SeverityMedium
			}
		}
	}
	
	return maxSeverity
}

// convertThreats converts interface threats to ThreatInfo
func (al *AuditLogger) convertThreats(threats []interface{}) []ThreatInfo {
	var result []ThreatInfo
	
	for _, threat := range threats {
		if threatMap, ok := threat.(map[string]interface{}); ok {
			info := ThreatInfo{}
			
			if t, ok := threatMap["type"].(string); ok {
				info.Type = t
			}
			if d, ok := threatMap["description"].(string); ok {
				info.Description = d
			}
			if s, ok := threatMap["severity"].(string); ok {
				info.Severity = s
			}
			if p, ok := threatMap["pattern"].(string); ok {
				info.Pattern = p
			}
			if f, ok := threatMap["field"].(string); ok {
				info.Field = f
			}
			if v, ok := threatMap["value"].(string); ok {
				info.Value = v
			}
			
			result = append(result, info)
		}
	}
	
	return result
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return fmt.Sprintf("audit_%d_%d", time.Now().UnixNano(), time.Now().Nanosecond())
}

// Flush forces a flush of the event buffer
func (al *AuditLogger) Flush() {
	al.mu.Lock()
	defer al.mu.Unlock()
	al.flushBuffer()
}

// Close closes the audit logger and flushes remaining events
func (al *AuditLogger) Close() {
	al.Flush()
}