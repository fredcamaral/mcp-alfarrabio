// Package handlers provides HTTP request handlers for the MCP Memory Server API.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"lerian-mcp-memory/internal/api/response"
	"lerian-mcp-memory/internal/config"
	"lerian-mcp-memory/internal/websocket"
)

// WebSocketHandler manages WebSocket server and connections
type WebSocketHandler struct {
	config   *config.Config
	server   *websocket.Server
	mu       sync.RWMutex
	logger   *log.Logger
	initOnce sync.Once
	initErr  error
}

// WebSocketStatus represents the WebSocket server status
type WebSocketStatus struct {
	Running           bool                     `json:"running"`
	ActiveConnections int                      `json:"active_connections"`
	MaxConnections    int                      `json:"max_connections"`
	AvailableSlots    int                      `json:"available_slots"`
	ServerMetrics     *websocket.SystemMetrics `json:"server_metrics"`
	PoolMetrics       *websocket.PoolMetrics   `json:"pool_metrics"`
	ConnectionStats   map[string]interface{}   `json:"connection_stats"`
	Config            *websocket.ServerConfig  `json:"config"`
	Timestamp         string                   `json:"timestamp"`
}

// WebSocketError represents a WebSocket-specific error response
type WebSocketError struct {
	Error     string                 `json:"error"`
	Code      string                 `json:"code"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp string                 `json:"timestamp"`
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(cfg *config.Config, logger *log.Logger) *WebSocketHandler {
	if logger == nil {
		logger = log.Default()
	}

	return &WebSocketHandler{
		config: cfg,
		logger: logger,
	}
}

// Initialize initializes the WebSocket server (thread-safe, runs only once)
func (h *WebSocketHandler) Initialize() error {
	h.initOnce.Do(func() {
		// Create WebSocket server configuration
		wsConfig := &websocket.ServerConfig{
			MaxConnections:    h.config.WebSocket.MaxConnections,
			ReadBufferSize:    h.config.WebSocket.ReadBufferSize,
			WriteBufferSize:   h.config.WebSocket.WriteBufferSize,
			HandshakeTimeout:  time.Duration(h.config.WebSocket.HandshakeTimeout) * time.Second,
			PingInterval:      time.Duration(h.config.WebSocket.PingInterval) * time.Second,
			PongTimeout:       time.Duration(h.config.WebSocket.PongTimeout) * time.Second,
			WriteTimeout:      time.Duration(h.config.WebSocket.WriteTimeout) * time.Second,
			ReadTimeout:       time.Duration(h.config.WebSocket.ReadTimeout) * time.Second,
			EnableCompression: h.config.WebSocket.EnableCompression,
			MaxMessageSize:    int64(h.config.WebSocket.MaxMessageSize),
			EnableAuth:        h.config.WebSocket.EnableAuth,
			AllowedOrigins:    h.config.WebSocket.AllowedOrigins,
		}

		// Create WebSocket server
		h.server = websocket.NewServer(wsConfig)

		// Start the server
		if err := h.server.Start(); err != nil {
			h.initErr = fmt.Errorf("failed to start WebSocket server: %w", err)
			h.logger.Printf("Error starting WebSocket server: %v", h.initErr)
			return
		}

		h.logger.Println("WebSocket server initialized successfully")
	})

	return h.initErr
}

// HandleUpgrade handles WebSocket upgrade requests at /ws
func (h *WebSocketHandler) HandleUpgrade(w http.ResponseWriter, r *http.Request) {
	// Ensure server is initialized
	if err := h.Initialize(); err != nil {
		h.writeError(w, http.StatusServiceUnavailable, "WS_SERVER_NOT_AVAILABLE",
			"WebSocket server is not available", map[string]interface{}{
				"reason": err.Error(),
			})
		return
	}

	// Check if server is running
	if !h.server.IsRunning() {
		h.writeError(w, http.StatusServiceUnavailable, "WS_SERVER_NOT_RUNNING",
			"WebSocket server is not running", nil)
		return
	}

	// Add CORS headers if needed
	if origin := r.Header.Get("Origin"); origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	// Handle preflight requests
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Methods", "GET")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-CLI-Version, X-Request-ID")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Only allow GET requests for WebSocket upgrade
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Log upgrade attempt
	h.logger.Printf("WebSocket upgrade request from %s (user-agent: %s)",
		r.RemoteAddr, r.UserAgent())

	// Delegate to the WebSocket server
	h.server.HandleUpgrade(w, r)
}

// HandleStatus handles GET /ws/status requests
func (h *WebSocketHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	// Check if server is initialized
	if h.server == nil {
		h.writeError(w, http.StatusServiceUnavailable, "WS_SERVER_NOT_INITIALIZED",
			"WebSocket server is not initialized", nil)
		return
	}

	// Build status response
	status := h.buildStatus()

	// Write response
	response.WriteSuccess(w, status)
}

// HandleMetrics handles GET /ws/metrics requests
func (h *WebSocketHandler) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	// Check if server is initialized
	if h.server == nil {
		h.writeError(w, http.StatusServiceUnavailable, "WS_SERVER_NOT_INITIALIZED",
			"WebSocket server is not initialized", nil)
		return
	}

	// Get query parameters
	connectionID := r.URL.Query().Get("connection_id")
	since := r.URL.Query().Get("since")

	// If connection ID is provided, get specific connection metrics
	if connectionID != "" {
		// TODO: Add method to access MetricsCollector from Server
		// For now, return not implemented
		h.writeError(w, http.StatusNotImplemented, "WS_FEATURE_NOT_IMPLEMENTED",
			"Connection-specific metrics not yet implemented", nil)
		return
		/*metrics, err := h.server.GetMetrics().GetConnectionMetrics(connectionID)
		if err != nil {
			h.writeError(w, http.StatusNotFound, "WS_CONNECTION_NOT_FOUND",
				fmt.Sprintf("Connection %s not found", connectionID), nil)
			return
		}
		response.WriteSuccess(w, metrics)
		return*/
	}

	// If since parameter is provided, get time series data
	if since != "" {
		_, err := time.Parse(time.RFC3339, since)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "WS_INVALID_TIME_FORMAT",
				"Invalid time format for 'since' parameter", map[string]interface{}{
					"expected_format": "RFC3339",
					"example":         "2006-01-02T15:04:05Z07:00",
				})
			return
		}

		// TODO: Add method to access MetricsCollector from Server
		// For now, return empty time series
		var timeSeriesData []interface{}
		response.WriteSuccess(w, map[string]interface{}{
			"time_series": timeSeriesData,
			"since":       since,
			"count":       len(timeSeriesData),
			"note":        "Time series metrics not yet implemented",
		})
		return
	}

	// Return system metrics only for now
	response.WriteSuccess(w, map[string]interface{}{
		"connections": map[string]interface{}{
			"total": h.server.GetConnectionCount(),
			"note":  "Per-connection metrics not yet implemented",
		},
		"system":    h.server.GetMetrics(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// HandleBroadcast handles POST /ws/broadcast requests
func (h *WebSocketHandler) HandleBroadcast(w http.ResponseWriter, r *http.Request) {
	// Check if server is initialized
	if h.server == nil || !h.server.IsRunning() {
		h.writeError(w, http.StatusServiceUnavailable, "WS_SERVER_NOT_AVAILABLE",
			"WebSocket server is not available", nil)
		return
	}

	// Parse request body
	var event websocket.MemoryEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		h.writeError(w, http.StatusBadRequest, "WS_INVALID_REQUEST",
			"Invalid request body", map[string]interface{}{
				"error": err.Error(),
			})
		return
	}

	// Validate event
	if event.Type == "" {
		h.writeError(w, http.StatusBadRequest, "WS_MISSING_EVENT_TYPE",
			"Event type is required", nil)
		return
	}

	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Broadcast the event
	h.server.BroadcastEvent(&event)

	// Log the broadcast
	h.logger.Printf("Broadcasted %s event to all connected clients", event.Type)

	// Return success response
	response.WriteSuccess(w, map[string]interface{}{
		"status":     "broadcast_sent",
		"event_type": event.Type,
		"timestamp":  event.Timestamp.Format(time.RFC3339),
		"recipients": h.server.GetConnectionCount(),
	})
}

// BroadcastMemoryEvent broadcasts a memory-related event to all connected clients
func (h *WebSocketHandler) BroadcastMemoryEvent(eventType, action, chunkID, repository, sessionID string, data interface{}) error {
	// Check if server is initialized
	if h.server == nil || !h.server.IsRunning() {
		return fmt.Errorf("WebSocket server is not available")
	}

	// Create memory event
	event := websocket.NewMemoryEvent(eventType, action, chunkID, repository, sessionID, data)

	// Broadcast the event
	h.server.BroadcastEvent(&event)

	// Log the broadcast
	h.logger.Printf("Broadcasted memory event: type=%s, action=%s, chunk=%s, repo=%s",
		eventType, action, chunkID, repository)

	return nil
}

// Shutdown gracefully shuts down the WebSocket server
func (h *WebSocketHandler) Shutdown(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.server == nil {
		return nil
	}

	h.logger.Println("Shutting down WebSocket server...")

	// Stop the server
	if err := h.server.Stop(); err != nil {
		return fmt.Errorf("failed to stop WebSocket server: %w", err)
	}

	h.logger.Println("WebSocket server shut down successfully")
	return nil
}

// buildStatus builds the current WebSocket server status
func (h *WebSocketHandler) buildStatus() *WebSocketStatus {
	if h.server == nil {
		return &WebSocketStatus{
			Running:   false,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
	}

	pool := h.server.GetPool()
	connectionCount := h.server.GetConnectionCount()

	return &WebSocketStatus{
		Running:           h.server.IsRunning(),
		ActiveConnections: connectionCount,
		MaxConnections:    h.server.GetConfig().MaxConnections,
		AvailableSlots:    pool.GetAvailableCapacity(),
		ServerMetrics:     h.server.GetMetrics(),
		PoolMetrics:       pool.GetMetrics(),
		ConnectionStats:   pool.GetConnectionStats(),
		Config:            h.server.GetConfig(),
		Timestamp:         time.Now().UTC().Format(time.RFC3339),
	}
}

// writeError writes an error response
func (h *WebSocketHandler) writeError(w http.ResponseWriter, statusCode int, code, message string, details map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := WebSocketError{
		Error:     message,
		Code:      code,
		Details:   details,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if err := json.NewEncoder(w).Encode(errorResp); err != nil {
		h.logger.Printf("Failed to write error response: %v", err)
	}
}

// GetServer returns the underlying WebSocket server (for testing)
func (h *WebSocketHandler) GetServer() *websocket.Server {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.server
}

// Additional handler methods for backward compatibility

// HandleConnectionInfo provides information about specific connections
func (h *WebSocketHandler) HandleConnectionInfo(w http.ResponseWriter, r *http.Request) {
	if h.server == nil || !h.server.IsRunning() {
		h.writeError(w, http.StatusServiceUnavailable, "WS_SERVER_NOT_AVAILABLE",
			"WebSocket server is not available", nil)
		return
	}

	pool := h.server.GetPool()

	// Get query parameters
	clientID := r.URL.Query().Get("client_id")
	repository := r.URL.Query().Get("repository")
	sessionID := r.URL.Query().Get("session_id")

	var result interface{}

	switch {
	case clientID != "":
		// Get specific client information
		client, exists := pool.GetConnection(clientID)
		if !exists {
			h.writeError(w, http.StatusNotFound, "WS_CLIENT_NOT_FOUND",
				"Client not found", nil)
			return
		}

		result = map[string]interface{}{
			"client_id":  client.ID,
			"repository": client.Repository,
			"session_id": client.SessionID,
			"metadata":   client.Metadata,
			"connected":  true,
		}
	case repository != "":
		// Get connections by repository
		clients := pool.GetConnectionsByRepository(repository)
		connections := make([]map[string]interface{}, len(clients))

		for i, client := range clients {
			connections[i] = map[string]interface{}{
				"client_id":  client.ID,
				"session_id": client.SessionID,
				"metadata":   client.Metadata,
			}
		}

		result = map[string]interface{}{
			"repository":  repository,
			"connections": connections,
			"count":       len(connections),
		}
	case sessionID != "":
		// Get connections by session
		clients := pool.GetConnectionsBySession(sessionID)
		connections := make([]map[string]interface{}, len(clients))

		for i, client := range clients {
			connections[i] = map[string]interface{}{
				"client_id":  client.ID,
				"repository": client.Repository,
				"metadata":   client.Metadata,
			}
		}

		result = map[string]interface{}{
			"session_id":  sessionID,
			"connections": connections,
			"count":       len(connections),
		}
	default:
		// Get all connections summary
		allConnections := pool.GetAllConnections()
		summary := make([]map[string]interface{}, 0, len(allConnections))

		for _, client := range allConnections {
			summary = append(summary, map[string]interface{}{
				"client_id":     client.ID,
				"repository":    client.Repository,
				"session_id":    client.SessionID,
				"connected_at":  client.Metadata.ConnectedAt,
				"last_activity": client.Metadata.LastActivity,
			})
		}

		result = map[string]interface{}{
			"total_connections": len(allConnections),
			"connections":       summary,
			"pool_metrics":      pool.GetMetrics(),
		}
	}

	response.WriteSuccess(w, result)
}

// HandleHealthCheck provides a simple health check for the WebSocket server
func (h *WebSocketHandler) HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "ok",
		"running":   false,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	if h.server != nil && h.server.IsRunning() {
		health["running"] = true
		health["connections"] = h.server.GetConnectionCount()
		health["max_connections"] = h.server.GetConfig().MaxConnections

		// Add basic performance indicators
		metrics := h.server.GetMetrics()
		health["uptime"] = time.Duration(metrics.UptimeSeconds * int64(time.Second)).String()
		health["total_connections"] = metrics.TotalConnections
		if metrics.TotalConnections > 0 {
			health["error_rate"] = fmt.Sprintf("%.2f%%", float64(metrics.TotalErrors)/float64(metrics.TotalConnections)*100)
		} else {
			health["error_rate"] = "0.00%"
		}
	}

	response.WriteSuccess(w, health)
}
