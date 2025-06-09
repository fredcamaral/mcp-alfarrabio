// Package websocket provides connection diagnostics for WebSocket troubleshooting
package websocket

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// DiagnosticsManager provides comprehensive connection diagnostics
type DiagnosticsManager struct {
	mu            sync.RWMutex
	config        *DiagnosticsConfig
	connections   map[string]*ConnectionDiagnostics
	systemDiag    *SystemDiagnostics
	debugSessions map[string]*DebugSession
	done          chan struct{}
}

// DiagnosticsConfig configures diagnostics behavior
type DiagnosticsConfig struct {
	EnableDebugLogging    bool          `json:"enable_debug_logging" yaml:"enable_debug_logging"`
	EnableNetworkTracing  bool          `json:"enable_network_tracing" yaml:"enable_network_tracing"`
	EnablePerformanceProf bool          `json:"enable_performance_profiling" yaml:"enable_performance_profiling"`
	DiagnosticInterval    time.Duration `json:"diagnostic_interval" yaml:"diagnostic_interval"`
	RetentionPeriod       time.Duration `json:"retention_period" yaml:"retention_period"`
	MaxLogEntries         int           `json:"max_log_entries" yaml:"max_log_entries"`
	MaxDebugSessions      int           `json:"max_debug_sessions" yaml:"max_debug_sessions"`
	DetailLevel           DetailLevel   `json:"detail_level" yaml:"detail_level"`
}

// ConnectionDiagnostics tracks diagnostic information for a connection
type ConnectionDiagnostics struct {
	mu         sync.RWMutex
	ID         string
	Connection *websocket.Conn
	StartTime  time.Time
	RemoteAddr string
	LocalAddr  string
	Protocol   string
	UserAgent  string

	// Network diagnostics
	NetworkInfo        *NetworkInfo
	ConnectionState    *DiagnosticConnectionState
	PerformanceMetrics *DiagnosticPerformanceMetrics

	// Debug information
	DebugLogs     []*DebugEntry
	EventTimeline []*DiagnosticEvent
	Traces        []*NetworkTrace

	// Health diagnostics
	HealthChecks    []*HealthCheck
	LastHealthCheck time.Time
	HealthStatus    HealthDiagStatus
}

// SystemDiagnostics tracks system-wide diagnostic information
type SystemDiagnostics struct {
	mu                 sync.RWMutex
	StartTime          time.Time
	SystemHealth       SystemHealthStatus
	ResourceUsage      *ResourceUsage
	NetworkStats       *NetworkStats
	PerformanceProfile *SystemPerformance
	LastUpdate         time.Time
}

// DebugSession represents an active debugging session
type DebugSession struct {
	ID              string
	ConnectionID    string
	StartTime       time.Time
	EnabledFeatures []DebugFeature
	LogLevel        LogLevel
	Filters         []DebugFilter
	Collectors      map[string]bool
	OutputChannels  []string
	LastActivity    time.Time
}

// NetworkInfo contains network-level diagnostic information
type NetworkInfo struct {
	LocalIP          string        `json:"local_ip"`
	RemoteIP         string        `json:"remote_ip"`
	LocalPort        int           `json:"local_port"`
	RemotePort       int           `json:"remote_port"`
	NetworkInterface string        `json:"network_interface"`
	ConnectionType   string        `json:"connection_type"`
	TLSVersion       string        `json:"tls_version,omitempty"`
	CipherSuite      string        `json:"cipher_suite,omitempty"`
	Latency          time.Duration `json:"latency"`
	Bandwidth        float64       `json:"bandwidth_bps"`
	PacketLoss       float64       `json:"packet_loss_percent"`
	Jitter           time.Duration `json:"jitter"`
}

// DiagnosticConnectionState tracks the state of a WebSocket connection
type DiagnosticConnectionState struct {
	State           string        `json:"state"`
	SubState        string        `json:"sub_state"`
	ReadyState      int           `json:"ready_state"`
	BufferedAmount  int64         `json:"buffered_amount"`
	Extensions      []string      `json:"extensions"`
	Protocol        string        `json:"protocol"`
	CloseCode       int           `json:"close_code,omitempty"`
	CloseReason     string        `json:"close_reason,omitempty"`
	LastStateChange time.Time     `json:"last_state_change"`
	StateHistory    []StateChange `json:"state_history"`
}

// DiagnosticPerformanceMetrics tracks performance-related metrics
type DiagnosticPerformanceMetrics struct {
	CPU               float64       `json:"cpu_percent"`
	Memory            int64         `json:"memory_bytes"`
	GoroutineCount    int           `json:"goroutine_count"`
	GCStats           GCStats       `json:"gc_stats"`
	MessageThroughput float64       `json:"message_throughput"`
	ProcessingLatency time.Duration `json:"processing_latency"`
	QueueDepth        int           `json:"queue_depth"`
}

// DebugEntry represents a debug log entry
type DebugEntry struct {
	Timestamp    time.Time              `json:"timestamp"`
	Level        LogLevel               `json:"level"`
	Component    string                 `json:"component"`
	Message      string                 `json:"message"`
	Context      map[string]interface{} `json:"context"`
	StackTrace   string                 `json:"stack_trace,omitempty"`
	ConnectionID string                 `json:"connection_id,omitempty"`
}

// DiagnosticEvent represents a diagnostic event
type DiagnosticEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	Type        EventType              `json:"type"`
	Category    EventCategory          `json:"category"`
	Description string                 `json:"description"`
	Data        map[string]interface{} `json:"data"`
	Severity    EventSeverity          `json:"severity"`
}

// NetworkTrace represents network-level tracing information
type NetworkTrace struct {
	Timestamp   time.Time         `json:"timestamp"`
	Direction   string            `json:"direction"`
	MessageType string            `json:"message_type"`
	Size        int               `json:"size"`
	Latency     time.Duration     `json:"latency"`
	Headers     map[string]string `json:"headers,omitempty"`
	Payload     string            `json:"payload,omitempty"`
	Error       string            `json:"error,omitempty"`
}

// HealthCheck represents a health check result
type HealthCheck struct {
	Timestamp time.Time              `json:"timestamp"`
	CheckType HealthCheckType        `json:"check_type"`
	Status    HealthCheckStatus      `json:"status"`
	Latency   time.Duration          `json:"latency"`
	Details   map[string]interface{} `json:"details"`
	Errors    []string               `json:"errors,omitempty"`
}

// Enums and constants

type DetailLevel string

const (
	DetailLevelBasic    DetailLevel = "basic"
	DetailLevelStandard DetailLevel = "standard"
	DetailLevelDetailed DetailLevel = "detailed"
	DetailLevelVerbose  DetailLevel = "verbose"
)

type HealthDiagStatus string

const (
	HealthDiagHealthy  HealthDiagStatus = "healthy"
	HealthDiagWarning  HealthDiagStatus = "warning"
	HealthDiagCritical HealthDiagStatus = "critical"
	HealthDiagUnknown  HealthDiagStatus = "unknown"
)

type SystemHealthStatus string

const (
	SystemHealthy   SystemHealthStatus = "healthy"
	SystemDegraded  SystemHealthStatus = "degraded"
	SystemUnhealthy SystemHealthStatus = "unhealthy"
)

type DebugFeature string

const (
	FeatureNetworkTracing  DebugFeature = "network_tracing"
	FeatureMessageLogging  DebugFeature = "message_logging"
	FeaturePerformanceProf DebugFeature = "performance_profiling"
	FeatureStateTracking   DebugFeature = "state_tracking"
	FeatureErrorCapture    DebugFeature = "error_capture"
)

type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

type DebugFilter struct {
	Type  FilterType `json:"type"`
	Value string     `json:"value"`
}

type FilterType string

const (
	FilterConnectionID FilterType = "connection_id"
	FilterMessageType  FilterType = "message_type"
	FilterComponent    FilterType = "component"
	FilterSeverity     FilterType = "severity"
)

type EventType string

const (
	EventConnection  EventType = "connection"
	EventMessage     EventType = "message"
	EventError       EventType = "error"
	EventPerformance EventType = "performance"
	EventSecurity    EventType = "security"
)

type EventCategory string

const (
	CategoryNetwork     EventCategory = "network"
	CategoryProtocol    EventCategory = "protocol"
	CategoryApplication EventCategory = "application"
	CategorySystem      EventCategory = "system"
)

type EventSeverity string

const (
	SeverityLow      EventSeverity = "low"
	SeverityMedium   EventSeverity = "medium"
	SeverityHigh     EventSeverity = "high"
	SeverityCritical EventSeverity = "critical"
)

type HealthCheckType string

const (
	HealthCheckPing         HealthCheckType = "ping"
	HealthCheckEcho         HealthCheckType = "echo"
	HealthCheckBandwidth    HealthCheckType = "bandwidth"
	HealthCheckLatency      HealthCheckType = "latency"
	HealthCheckConnectivity HealthCheckType = "connectivity"
)

type HealthCheckStatus string

const (
	HealthCheckPassed  HealthCheckStatus = "passed"
	HealthCheckFailed  HealthCheckStatus = "failed"
	HealthCheckWarning HealthCheckStatus = "warning"
)

// Additional supporting types

type ResourceUsage struct {
	CPUUsage    float64   `json:"cpu_usage_percent"`
	MemoryUsage int64     `json:"memory_usage_bytes"`
	DiskUsage   int64     `json:"disk_usage_bytes"`
	NetworkIO   NetworkIO `json:"network_io"`
	OpenFiles   int       `json:"open_files"`
	Threads     int       `json:"threads"`
}

type NetworkStats struct {
	BytesReceived   int64 `json:"bytes_received"`
	BytesSent       int64 `json:"bytes_sent"`
	PacketsReceived int64 `json:"packets_received"`
	PacketsSent     int64 `json:"packets_sent"`
	ErrorsReceived  int64 `json:"errors_received"`
	ErrorsSent      int64 `json:"errors_sent"`
	DroppedPackets  int64 `json:"dropped_packets"`
	Retransmissions int64 `json:"retransmissions"`
}

type SystemPerformance struct {
	Throughput       float64       `json:"throughput_msg_per_sec"`
	AverageLatency   time.Duration `json:"average_latency"`
	P95Latency       time.Duration `json:"p95_latency"`
	P99Latency       time.Duration `json:"p99_latency"`
	ErrorRate        float64       `json:"error_rate"`
	MemoryEfficiency float64       `json:"memory_efficiency"`
}

type StateChange struct {
	FromState string    `json:"from_state"`
	ToState   string    `json:"to_state"`
	Timestamp time.Time `json:"timestamp"`
	Trigger   string    `json:"trigger"`
}

type GCStats struct {
	NumGC        uint32        `json:"num_gc"`
	PauseTotal   time.Duration `json:"pause_total"`
	PauseAverage time.Duration `json:"pause_average"`
	LastGC       time.Time     `json:"last_gc"`
}

type NetworkIO struct {
	BytesRead    int64 `json:"bytes_read"`
	BytesWritten int64 `json:"bytes_written"`
	ReadOps      int64 `json:"read_ops"`
	WriteOps     int64 `json:"write_ops"`
}

// NewDiagnosticsManager creates a new diagnostics manager
func NewDiagnosticsManager(config *DiagnosticsConfig) *DiagnosticsManager {
	if config == nil {
		config = DefaultDiagnosticsConfig()
	}

	dm := &DiagnosticsManager{
		config:        config,
		connections:   make(map[string]*ConnectionDiagnostics),
		systemDiag:    &SystemDiagnostics{StartTime: time.Now()},
		debugSessions: make(map[string]*DebugSession),
		done:          make(chan struct{}),
	}

	// Start diagnostic routines
	go dm.diagnosticRoutine()

	return dm
}

// DefaultDiagnosticsConfig returns default diagnostics configuration
func DefaultDiagnosticsConfig() *DiagnosticsConfig {
	return &DiagnosticsConfig{
		EnableDebugLogging:    true,
		EnableNetworkTracing:  true,
		EnablePerformanceProf: true,
		DiagnosticInterval:    30 * time.Second,
		RetentionPeriod:       24 * time.Hour,
		MaxLogEntries:         10000,
		MaxDebugSessions:      10,
		DetailLevel:           DetailLevelStandard,
	}
}

// RegisterConnection registers a connection for diagnostics
func (dm *DiagnosticsManager) RegisterConnection(id string, conn *websocket.Conn) error {
	if id == "" {
		return fmt.Errorf("connection ID cannot be empty")
	}

	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Extract connection information
	localAddr := ""
	remoteAddr := ""

	if conn.UnderlyingConn() != nil {
		localAddr = conn.UnderlyingConn().LocalAddr().String()
		remoteAddr = conn.UnderlyingConn().RemoteAddr().String()
	}

	diag := &ConnectionDiagnostics{
		ID:         id,
		Connection: conn,
		StartTime:  time.Now(),
		LocalAddr:  localAddr,
		RemoteAddr: remoteAddr,
		Protocol:   conn.Subprotocol(),

		NetworkInfo: &NetworkInfo{},
		ConnectionState: &DiagnosticConnectionState{
			State:           "connected",
			ReadyState:      1, // OPEN
			StateHistory:    make([]StateChange, 0),
			LastStateChange: time.Now(),
		},
		PerformanceMetrics: &DiagnosticPerformanceMetrics{},

		DebugLogs:     make([]*DebugEntry, 0),
		EventTimeline: make([]*DiagnosticEvent, 0),
		Traces:        make([]*NetworkTrace, 0),
		HealthChecks:  make([]*HealthCheck, 0),

		HealthStatus: HealthDiagHealthy,
	}

	// Initialize network info
	dm.initializeNetworkInfo(diag)

	dm.connections[id] = diag

	// Log registration event
	dm.logEvent(id, EventConnection, CategoryNetwork, "Connection registered for diagnostics",
		map[string]interface{}{
			"remote_addr": remoteAddr,
			"local_addr":  localAddr,
			"protocol":    conn.Subprotocol(),
		}, SeverityLow)

	log.Printf("Registered connection %s for diagnostics", id)
	return nil
}

// UnregisterConnection removes a connection from diagnostics
func (dm *DiagnosticsManager) UnregisterConnection(id string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if diag, exists := dm.connections[id]; exists {
		dm.logEvent(id, EventConnection, CategoryNetwork, "Connection unregistered from diagnostics",
			map[string]interface{}{
				"duration": time.Since(diag.StartTime).String(),
			}, SeverityLow)

		delete(dm.connections, id)
		log.Printf("Unregistered connection %s from diagnostics", id)
	}
}

// initializeNetworkInfo initializes network diagnostic information
func (dm *DiagnosticsManager) initializeNetworkInfo(diag *ConnectionDiagnostics) {
	if diag.Connection == nil || diag.Connection.UnderlyingConn() == nil {
		return
	}

	conn := diag.Connection.UnderlyingConn()

	// Extract IP addresses and ports
	if localAddr := conn.LocalAddr(); localAddr != nil {
		if tcpAddr, ok := localAddr.(*net.TCPAddr); ok {
			diag.NetworkInfo.LocalIP = tcpAddr.IP.String()
			diag.NetworkInfo.LocalPort = tcpAddr.Port
		}
	}

	if remoteAddr := conn.RemoteAddr(); remoteAddr != nil {
		if tcpAddr, ok := remoteAddr.(*net.TCPAddr); ok {
			diag.NetworkInfo.RemoteIP = tcpAddr.IP.String()
			diag.NetworkInfo.RemotePort = tcpAddr.Port
		}
	}

	diag.NetworkInfo.ConnectionType = "websocket"

	// Perform initial latency test
	go dm.measureLatency(diag)
}

// measureLatency measures connection latency
func (dm *DiagnosticsManager) measureLatency(diag *ConnectionDiagnostics) {
	if diag.Connection == nil {
		return
	}

	start := time.Now()
	err := diag.Connection.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(5*time.Second))
	if err != nil {
		dm.logError(diag.ID, "Failed to send ping for latency measurement", err)
		return
	}

	// Note: In a real implementation, you'd wait for the pong response
	// For simplicity, we're just measuring the write latency here
	latency := time.Since(start)

	diag.mu.Lock()
	diag.NetworkInfo.Latency = latency
	diag.mu.Unlock()

	dm.logEvent(diag.ID, EventPerformance, CategoryNetwork, "Latency measured",
		map[string]interface{}{
			"latency_ms": latency.Milliseconds(),
		}, SeverityLow)
}

// LogMessage logs a message for diagnostics
func (dm *DiagnosticsManager) LogMessage(connectionID, messageType string, size int, direction string, latency time.Duration) {
	if !dm.config.EnableNetworkTracing {
		return
	}

	dm.mu.RLock()
	diag, exists := dm.connections[connectionID]
	dm.mu.RUnlock()

	if !exists {
		return
	}

	trace := &NetworkTrace{
		Timestamp:   time.Now(),
		Direction:   direction,
		MessageType: messageType,
		Size:        size,
		Latency:     latency,
	}

	diag.mu.Lock()
	diag.Traces = append(diag.Traces, trace)

	// Trim traces if needed
	if len(diag.Traces) > 1000 {
		diag.Traces = diag.Traces[1:]
	}
	diag.mu.Unlock()

	// Log event
	dm.logEvent(connectionID, EventMessage, CategoryProtocol, "Message traced",
		map[string]interface{}{
			"type":      messageType,
			"size":      size,
			"direction": direction,
			"latency":   latency.Milliseconds(),
		}, SeverityLow)
}

// LogError logs an error for diagnostics
func (dm *DiagnosticsManager) LogError(connectionID, message string, err error) {
	dm.logError(connectionID, message, err)
}

// logError internal error logging
func (dm *DiagnosticsManager) logError(connectionID, message string, err error) {
	entry := &DebugEntry{
		Timestamp:    time.Now(),
		Level:        LogLevelError,
		Component:    "websocket",
		Message:      message,
		Context:      map[string]interface{}{},
		ConnectionID: connectionID,
	}

	if err != nil {
		entry.Context["error"] = err.Error()
	}

	dm.addDebugEntry(connectionID, entry)

	// Log as event
	dm.logEvent(connectionID, EventError, CategoryApplication, message,
		map[string]interface{}{
			"error": err.Error(),
		}, SeverityHigh)
}

// logEvent logs a diagnostic event
func (dm *DiagnosticsManager) logEvent(connectionID string, eventType EventType, category EventCategory,
	description string, data map[string]interface{}, severity EventSeverity) {
	event := &DiagnosticEvent{
		Timestamp:   time.Now(),
		Type:        eventType,
		Category:    category,
		Description: description,
		Data:        data,
		Severity:    severity,
	}

	dm.mu.RLock()
	diag, exists := dm.connections[connectionID]
	dm.mu.RUnlock()

	if exists {
		diag.mu.Lock()
		diag.EventTimeline = append(diag.EventTimeline, event)

		// Trim events if needed
		if len(diag.EventTimeline) > 500 {
			diag.EventTimeline = diag.EventTimeline[1:]
		}
		diag.mu.Unlock()
	}

	if dm.config.EnableDebugLogging {
		log.Printf("Diagnostic event [%s:%s] %s: %s", eventType, category, connectionID, description)
	}
}

// addDebugEntry adds a debug log entry
func (dm *DiagnosticsManager) addDebugEntry(connectionID string, entry *DebugEntry) {
	dm.mu.RLock()
	diag, exists := dm.connections[connectionID]
	dm.mu.RUnlock()

	if exists {
		diag.mu.Lock()
		diag.DebugLogs = append(diag.DebugLogs, entry)

		// Trim logs if needed
		if len(diag.DebugLogs) > dm.config.MaxLogEntries {
			diag.DebugLogs = diag.DebugLogs[1:]
		}
		diag.mu.Unlock()
	}
}

// StartDebugSession starts a debug session for a connection
func (dm *DiagnosticsManager) StartDebugSession(connectionID string, features []DebugFeature, logLevel LogLevel) (string, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if len(dm.debugSessions) >= dm.config.MaxDebugSessions {
		return "", fmt.Errorf("maximum debug sessions reached")
	}

	sessionID := fmt.Sprintf("debug_%s_%d", connectionID, time.Now().Unix())

	session := &DebugSession{
		ID:              sessionID,
		ConnectionID:    connectionID,
		StartTime:       time.Now(),
		EnabledFeatures: features,
		LogLevel:        logLevel,
		Collectors:      make(map[string]bool),
		LastActivity:    time.Now(),
	}

	// Enable collectors based on features
	for _, feature := range features {
		session.Collectors[string(feature)] = true
	}

	dm.debugSessions[sessionID] = session

	log.Printf("Started debug session %s for connection %s", sessionID, connectionID)
	return sessionID, nil
}

// StopDebugSession stops a debug session
func (dm *DiagnosticsManager) StopDebugSession(sessionID string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	session, exists := dm.debugSessions[sessionID]
	if !exists {
		return fmt.Errorf("debug session %s not found", sessionID)
	}

	delete(dm.debugSessions, sessionID)
	log.Printf("Stopped debug session %s for connection %s", sessionID, session.ConnectionID)
	return nil
}

// PerformHealthCheck performs a health check on a connection
func (dm *DiagnosticsManager) PerformHealthCheck(connectionID string, checkType HealthCheckType) (*HealthCheck, error) {
	dm.mu.RLock()
	diag, exists := dm.connections[connectionID]
	dm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("connection %s not found", connectionID)
	}

	start := time.Now()
	check := &HealthCheck{
		Timestamp: start,
		CheckType: checkType,
		Details:   make(map[string]interface{}),
	}

	switch checkType {
	case HealthCheckPing:
		err := dm.performPingCheck(diag, check)
		if err != nil {
			check.Status = HealthCheckFailed
			check.Errors = []string{err.Error()}
		} else {
			check.Status = HealthCheckPassed
		}
	case HealthCheckLatency:
		dm.performLatencyCheck(diag, check)
	default:
		return nil, fmt.Errorf("unsupported health check type: %s", checkType)
	}

	check.Latency = time.Since(start)

	// Store health check
	diag.mu.Lock()
	diag.HealthChecks = append(diag.HealthChecks, check)
	diag.LastHealthCheck = time.Now()

	// Trim health checks if needed
	if len(diag.HealthChecks) > 100 {
		diag.HealthChecks = diag.HealthChecks[1:]
	}
	diag.mu.Unlock()

	return check, nil
}

// performPingCheck performs a ping health check
func (dm *DiagnosticsManager) performPingCheck(diag *ConnectionDiagnostics, check *HealthCheck) error {
	if diag.Connection == nil {
		return fmt.Errorf("connection is nil")
	}

	err := diag.Connection.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(5*time.Second))
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	check.Details["ping_sent"] = true
	return nil
}

// performLatencyCheck performs a latency health check
func (dm *DiagnosticsManager) performLatencyCheck(diag *ConnectionDiagnostics, check *HealthCheck) {
	diag.mu.RLock()
	latency := diag.NetworkInfo.Latency
	diag.mu.RUnlock()

	check.Details["current_latency_ms"] = latency.Milliseconds()

	switch {
	case latency < 100*time.Millisecond:
		check.Status = HealthCheckPassed
	case latency < 500*time.Millisecond:
		check.Status = HealthCheckWarning
	default:
		check.Status = HealthCheckFailed
		check.Errors = []string{"High latency detected"}
	}
}

// diagnosticRoutine runs periodic diagnostics
func (dm *DiagnosticsManager) diagnosticRoutine() {
	ticker := time.NewTicker(dm.config.DiagnosticInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			dm.performSystemDiagnostics()
			dm.cleanupOldData()
		case <-dm.done:
			return
		}
	}
}

// performSystemDiagnostics performs system-wide diagnostics
func (dm *DiagnosticsManager) performSystemDiagnostics() {
	dm.systemDiag.mu.Lock()
	defer dm.systemDiag.mu.Unlock()

	dm.systemDiag.LastUpdate = time.Now()

	// Update system health based on connection health
	dm.mu.RLock()
	totalConnections := len(dm.connections)
	healthyConnections := 0

	for _, diag := range dm.connections {
		if diag.HealthStatus == HealthDiagHealthy {
			healthyConnections++
		}
	}
	dm.mu.RUnlock()

	if totalConnections == 0 {
		dm.systemDiag.SystemHealth = SystemHealthy
	} else {
		healthRatio := float64(healthyConnections) / float64(totalConnections)
		switch {
		case healthRatio >= 0.9:
			dm.systemDiag.SystemHealth = SystemHealthy
		case healthRatio >= 0.7:
			dm.systemDiag.SystemHealth = SystemDegraded
		default:
			dm.systemDiag.SystemHealth = SystemUnhealthy
		}
	}
}

// cleanupOldData cleans up old diagnostic data
func (dm *DiagnosticsManager) cleanupOldData() {
	cutoff := time.Now().Add(-dm.config.RetentionPeriod)

	dm.mu.Lock()
	defer dm.mu.Unlock()

	for _, diag := range dm.connections {
		diag.mu.Lock()

		// Clean old debug logs
		validLogs := make([]*DebugEntry, 0)
		for _, entry := range diag.DebugLogs {
			if entry.Timestamp.After(cutoff) {
				validLogs = append(validLogs, entry)
			}
		}
		diag.DebugLogs = validLogs

		// Clean old events
		validEvents := make([]*DiagnosticEvent, 0)
		for _, event := range diag.EventTimeline {
			if event.Timestamp.After(cutoff) {
				validEvents = append(validEvents, event)
			}
		}
		diag.EventTimeline = validEvents

		// Clean old traces
		validTraces := make([]*NetworkTrace, 0)
		for _, trace := range diag.Traces {
			if trace.Timestamp.After(cutoff) {
				validTraces = append(validTraces, trace)
			}
		}
		diag.Traces = validTraces

		diag.mu.Unlock()
	}
}

// GetConnectionDiagnostics returns diagnostic information for a connection
func (dm *DiagnosticsManager) GetConnectionDiagnostics(connectionID string) (*ConnectionDiagnostics, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	diag, exists := dm.connections[connectionID]
	if !exists {
		return nil, fmt.Errorf("connection %s not found", connectionID)
	}

	// Return a copy to avoid race conditions
	diag.mu.RLock()
	defer diag.mu.RUnlock()

	return diag, nil
}

// GetSystemDiagnostics returns system diagnostic information
func (dm *DiagnosticsManager) GetSystemDiagnostics() *SystemDiagnostics {
	dm.systemDiag.mu.RLock()
	defer dm.systemDiag.mu.RUnlock()

	return dm.systemDiag
}

// GetDebugSessions returns active debug sessions
func (dm *DiagnosticsManager) GetDebugSessions() map[string]*DebugSession {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	sessions := make(map[string]*DebugSession)
	for id, session := range dm.debugSessions {
		sessions[id] = session
	}

	return sessions
}

// Close stops the diagnostics manager
func (dm *DiagnosticsManager) Close() error {
	close(dm.done)
	return nil
}
