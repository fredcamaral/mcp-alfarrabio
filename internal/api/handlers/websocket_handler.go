// Package handlers provides HTTP handlers for WebSocket upgrade and management
package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"lerian-mcp-memory/internal/websocket"
)

// Removed unused constants

// WebSocketHandler handles WebSocket upgrade requests and management
type WebSocketHandler struct {
	server *websocket.Server
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(server *websocket.Server) *WebSocketHandler {
	return &WebSocketHandler{
		server: server,
	}
}

// HandleUpgrade handles WebSocket upgrade requests at /api/v1/ws
func (wh *WebSocketHandler) HandleUpgrade(w http.ResponseWriter, r *http.Request) {
	// Add CORS headers if needed
	if origin := r.Header.Get("Origin"); origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	// Handle preflight requests
	if r.Method == "httpMethodOPTIONS" {
		w.Header().Set("Access-Control-Allow-Methods", "httpMethodGET")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-CLI-Version, X-Request-ID")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Only allow httpMethodGET requests for WebSocket upgrade
	if r.Method != "httpMethodGET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Log the upgrade request
	log.Printf("WebSocket upgrade request from %s (User-Agent: %s)",
		r.RemoteAddr, r.UserAgent())

	// Delegate to the WebSocket server
	wh.server.HandleUpgrade(w, r)
}

// HandleStatus provides WebSocket server status information
func (wh *WebSocketHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	if !wh.server.IsRunning() {
		http.Error(w, "WebSocket server not running", http.StatusServiceUnavailable)
		return
	}

	status := map[string]interface{}{
		"status":           "running",
		"connection_count": wh.server.GetConnectionCount(),
		"max_connections":  wh.server.GetConfig().MaxConnections,
		"server_config":    wh.server.GetConfig(),
		"timestamp":        time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// HandleMetrics provides WebSocket server metrics
func (wh *WebSocketHandler) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	if !wh.server.IsRunning() {
		http.Error(w, "WebSocket server not running", http.StatusServiceUnavailable)
		return
	}

	metrics := wh.server.GetMetrics()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		log.Printf("Error encoding metrics: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// HandleConnectionInfo provides information about specific connections
func (wh *WebSocketHandler) HandleConnectionInfo(w http.ResponseWriter, r *http.Request) {
	if !wh.server.IsRunning() {
		http.Error(w, "WebSocket server not running", http.StatusServiceUnavailable)
		return
	}

	pool := wh.server.GetPool()

	// Get query parameters
	clientID := r.URL.Query().Get("client_id")
	repository := r.URL.Query().Get("repository")
	sessionID := r.URL.Query().Get("session_id")

	var result interface{}

	if clientID != "" {
		// Get specific client information
		client, exists := pool.GetConnection(clientID)
		if !exists {
			http.Error(w, "Client not found", http.StatusNotFound)
			return
		}

		result = map[string]interface{}{
			"client_id":  client.ID,
			"repository": client.Repository,
			"session_id": client.SessionID,
			"metadata":   client.Metadata,
			"connected":  true,
		}
	} else if repository != "" {
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
	} else if sessionID != "" {
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
	} else {
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

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("Error encoding connection info: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// HandleBroadcast allows manual broadcasting of events via HTTP
func (wh *WebSocketHandler) HandleBroadcast(w http.ResponseWriter, r *http.Request) {
	if r.Method != "httpMethodPOST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !wh.server.IsRunning() {
		http.Error(w, "WebSocket server not running", http.StatusServiceUnavailable)
		return
	}

	var event websocket.MemoryEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Broadcast the event
	wh.server.BroadcastEvent(&event)

	response := map[string]interface{}{
		"status":      "broadcasted",
		"event_type":  event.Type,
		"timestamp":   event.Timestamp,
		"connections": wh.server.GetConnectionCount(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// HandleHealthCheck provides a simple health check for the WebSocket server
func (wh *WebSocketHandler) HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "ok",
		"running":   wh.server.IsRunning(),
		"timestamp": time.Now(),
	}

	if wh.server.IsRunning() {
		health["connections"] = wh.server.GetConnectionCount()
		health["max_connections"] = wh.server.GetConfig().MaxConnections

		// Add basic performance indicators
		metrics := wh.server.GetMetrics()
		health["uptime"] = time.Duration(metrics.UptimeSeconds * int64(time.Second)).String()
		health["total_connections"] = metrics.TotalConnections
		health["error_rate"] = float64(metrics.TotalErrors) / float64(metrics.TotalConnections) * 100
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(health); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// HandleConnectionClose allows manual disconnection of clients
func (wh *WebSocketHandler) HandleConnectionClose(w http.ResponseWriter, r *http.Request) {
	if r.Method != "httpMethodPOST" && r.Method != "DELETE" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !wh.server.IsRunning() {
		http.Error(w, "WebSocket server not running", http.StatusServiceUnavailable)
		return
	}

	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		http.Error(w, "client_id parameter required", http.StatusBadRequest)
		return
	}

	pool := wh.server.GetPool()
	client, exists := pool.GetConnection(clientID)
	if !exists {
		http.Error(w, "Client not found", http.StatusNotFound)
		return
	}

	// Close the connection
	if err := client.Connection.Close(); err != nil {
		log.Printf("Error closing connection %s: %v", clientID, err)
		http.Error(w, "Error closing connection", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":    "disconnected",
		"client_id": clientID,
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// HandleConnectionsList provides paginated list of connections
func (wh *WebSocketHandler) HandleConnectionsList(w http.ResponseWriter, r *http.Request) {
	if !wh.server.IsRunning() {
		http.Error(w, "WebSocket server not running", http.StatusServiceUnavailable)
		return
	}

	page, pageSize := wh.parsePaginationParams(r)
	allConnections := wh.server.GetPool().GetAllConnections()
	total := len(allConnections)

	if wh.handleEmptyPage(w, page, pageSize, total) {
		return
	}

	connections := wh.paginateConnections(allConnections, page, pageSize, total)
	response := wh.buildConnectionsResponse(connections, page, pageSize, total)

	wh.writeJSONResponse(w, response)
}

// parsePaginationParams extracts page and pageSize from request
func (wh *WebSocketHandler) parsePaginationParams(r *http.Request) (page, pageSize int) {
	page = 1
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize = 20
	if sizeStr := r.URL.Query().Get("page_size"); sizeStr != "" {
		if s, err := strconv.Atoi(sizeStr); err == nil && s > 0 && s <= 100 {
			pageSize = s
		}
	}
	return
}

// handleEmptyPage handles case when page is beyond available data
func (wh *WebSocketHandler) handleEmptyPage(w http.ResponseWriter, page, pageSize, total int) bool {
	startIndex := (page - 1) * pageSize
	if startIndex < total {
		return false
	}

	response := map[string]interface{}{
		"connections": []interface{}{},
		"pagination":  wh.buildPaginationInfo(page, pageSize, total),
	}
	wh.writeJSONResponse(w, response)
	return true
}

// paginateConnections extracts the requested page of connections
func (wh *WebSocketHandler) paginateConnections(allConnections map[string]*websocket.Client, page, pageSize, total int) []map[string]interface{} {
	startIndex := (page - 1) * pageSize
	endIndex := startIndex + pageSize
	if endIndex > total {
		endIndex = total
	}

	connections := make([]map[string]interface{}, 0, endIndex-startIndex)
	i := 0
	for _, client := range allConnections {
		if i >= startIndex && i < endIndex {
			connections = append(connections, wh.connectionToMap(client))
		}
		i++
		if i >= endIndex {
			break
		}
	}
	return connections
}

// connectionToMap converts a connection to a map for JSON response
func (wh *WebSocketHandler) connectionToMap(client *websocket.Client) map[string]interface{} {
	return map[string]interface{}{
		"client_id":      client.ID,
		"repository":     client.Repository,
		"session_id":     client.SessionID,
		"remote_addr":    client.Metadata.RemoteAddr,
		"user_agent":     client.Metadata.UserAgent,
		"connected_at":   client.Metadata.ConnectedAt,
		"last_activity":  client.Metadata.LastActivity,
		"cli_version":    client.Metadata.CLIVersion,
		"bytes_sent":     client.Metadata.BytesSent,
		"bytes_received": client.Metadata.BytesReceived,
	}
}

// buildConnectionsResponse creates the final response structure
func (wh *WebSocketHandler) buildConnectionsResponse(connections []map[string]interface{}, page, pageSize, total int) map[string]interface{} {
	return map[string]interface{}{
		"connections": connections,
		"pagination":  wh.buildPaginationInfo(page, pageSize, total),
		"timestamp":   time.Now(),
	}
}

// buildPaginationInfo creates pagination metadata
func (wh *WebSocketHandler) buildPaginationInfo(page, pageSize, total int) map[string]interface{} {
	return map[string]interface{}{
		"page":      page,
		"page_size": pageSize,
		"total":     total,
		"pages":     (total + pageSize - 1) / pageSize,
	}
}

// writeJSONResponse writes a JSON response with error handling
func (wh *WebSocketHandler) writeJSONResponse(w http.ResponseWriter, response map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
