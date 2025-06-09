package repl

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"lerian-mcp-memory/internal/logging"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// HTTPServer provides HTTP endpoints for REPL notifications
type HTTPServer struct {
	session  *Session
	port     int
	server   *http.Server
	router   *mux.Router
	upgrader websocket.Upgrader
	clients  map[string]*WebSocketClient
	mu       sync.RWMutex
	logger   logging.Logger
}

// WebSocketClient represents a connected WebSocket client
type WebSocketClient struct {
	ID        string
	Conn      *websocket.Conn
	SendChan  chan []byte
	CloseChan chan bool
	LastPing  time.Time
}

// HTTPResponse represents a standard HTTP response
type HTTPResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// NewHTTPServer creates a new HTTP server for REPL
func NewHTTPServer(session *Session, port int, logger logging.Logger) *HTTPServer {
	server := &HTTPServer{
		session: session,
		port:    port,
		clients: make(map[string]*WebSocketClient),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow connections from localhost in development
				// In production, implement proper origin checking
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		logger: logger,
	}

	server.setupRoutes()
	return server
}

// setupRoutes configures HTTP routes
func (h *HTTPServer) setupRoutes() {
	h.router = mux.NewRouter()

	// API routes
	api := h.router.PathPrefix("/api/v1").Subrouter()
	api.Use(h.jsonMiddleware)
	api.Use(h.loggingMiddleware)

	// Health check
	api.HandleFunc("/health", h.handleHealth).Methods("GET")

	// Session endpoints
	api.HandleFunc("/session", h.handleGetSession).Methods("GET")
	api.HandleFunc("/session/context", h.handleGetContext).Methods("GET")
	api.HandleFunc("/session/context", h.handleUpdateContext).Methods("POST")

	// Notification endpoints
	api.HandleFunc("/notifications", h.handleGetNotifications).Methods("GET")
	api.HandleFunc("/notifications", h.handleSendNotification).Methods("POST")

	// WebSocket endpoint
	h.router.HandleFunc("/ws", h.handleWebSocket)

	// Static file server for any UI assets
	h.router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))
}

// Start starts the HTTP server
func (h *HTTPServer) Start() error {
	h.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", h.port),
		Handler:      h.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start notification broadcaster
	go h.broadcastNotifications()

	// Start WebSocket ping/pong handler
	go h.handleWebSocketPings()

	h.logger.Info("HTTP server starting", "port", h.port)
	return h.server.ListenAndServe()
}

// Shutdown gracefully shuts down the HTTP server
func (h *HTTPServer) Shutdown(ctx context.Context) error {
	// Close all WebSocket connections
	h.mu.Lock()
	for _, client := range h.clients {
		close(client.CloseChan)
	}
	h.mu.Unlock()

	// Shutdown HTTP server
	return h.server.Shutdown(ctx)
}

// Middleware

func (h *HTTPServer) jsonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func (h *HTTPServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create response writer wrapper to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		h.logger.Info("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.statusCode,
			"duration", duration,
			"remote", r.RemoteAddr,
		)
	})
}

// HTTP Handlers

func (h *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := HTTPResponse{
		Success: true,
		Message: "REPL HTTP server is healthy",
		Data: map[string]interface{}{
			"session_id": h.session.ID,
			"uptime":     time.Since(h.session.CreatedAt).String(),
			"mode":       string(h.session.Mode),
		},
	}
	h.sendJSONResponse(w, http.StatusOK, response)
}

func (h *HTTPServer) handleGetSession(w http.ResponseWriter, r *http.Request) {
	h.session.mu.RLock()
	defer h.session.mu.RUnlock()

	response := HTTPResponse{
		Success: true,
		Data: map[string]interface{}{
			"id":         h.session.ID,
			"mode":       string(h.session.Mode),
			"repository": h.session.Repository,
			"created_at": h.session.CreatedAt,
			"updated_at": h.session.UpdatedAt,
			"commands":   len(h.session.History),
		},
	}

	if h.session.ActiveWorkflow != nil {
		response.Data["active_workflow"] = map[string]interface{}{
			"id":       h.session.ActiveWorkflow.ID,
			"type":     h.session.ActiveWorkflow.Type,
			"stage":    h.session.ActiveWorkflow.Stage,
			"progress": fmt.Sprintf("%d/%d", h.session.ActiveWorkflow.CurrentStep, h.session.ActiveWorkflow.TotalSteps),
		}
	}

	h.sendJSONResponse(w, http.StatusOK, response)
}

func (h *HTTPServer) handleGetContext(w http.ResponseWriter, r *http.Request) {
	h.session.mu.RLock()
	defer h.session.mu.RUnlock()

	response := HTTPResponse{
		Success: true,
		Data:    h.session.Context,
	}
	h.sendJSONResponse(w, http.StatusOK, response)
}

func (h *HTTPServer) handleUpdateContext(w http.ResponseWriter, r *http.Request) {
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	h.session.mu.Lock()
	for key, value := range updates {
		h.session.Context[key] = value
	}
	h.session.UpdatedAt = time.Now()
	h.session.mu.Unlock()

	// Send notification about context update
	h.session.NotificationChan <- Notification{
		Type:    "context_updated",
		Message: "Session context updated via HTTP",
		Data:    updates,
	}

	response := HTTPResponse{
		Success: true,
		Message: "Context updated successfully",
	}
	h.sendJSONResponse(w, http.StatusOK, response)
}

func (h *HTTPServer) handleGetNotifications(w http.ResponseWriter, r *http.Request) {
	// This could be enhanced to return recent notifications
	response := HTTPResponse{
		Success: true,
		Message: "Use WebSocket connection for real-time notifications",
		Data: map[string]interface{}{
			"websocket_url": fmt.Sprintf("ws://localhost:%d/ws", h.port),
		},
	}
	h.sendJSONResponse(w, http.StatusOK, response)
}

func (h *HTTPServer) handleSendNotification(w http.ResponseWriter, r *http.Request) {
	var notification Notification
	if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, "Invalid notification format")
		return
	}

	// Set notification metadata
	notification.ID = uuid.New().String()
	notification.Timestamp = time.Now()

	// Send to REPL session
	select {
	case h.session.NotificationChan <- notification:
		response := HTTPResponse{
			Success: true,
			Message: "Notification sent successfully",
			Data: map[string]interface{}{
				"notification_id": notification.ID,
			},
		}
		h.sendJSONResponse(w, http.StatusOK, response)
	default:
		h.sendErrorResponse(w, http.StatusServiceUnavailable, "Notification queue full")
	}
}

// WebSocket Handlers

func (h *HTTPServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("WebSocket upgrade failed", "error", err)
		return
	}

	client := &WebSocketClient{
		ID:        uuid.New().String(),
		Conn:      conn,
		SendChan:  make(chan []byte, 256),
		CloseChan: make(chan bool),
		LastPing:  time.Now(),
	}

	h.mu.Lock()
	h.clients[client.ID] = client
	h.mu.Unlock()

	h.logger.Info("WebSocket client connected", "client_id", client.ID)

	// Send welcome message
	welcome := map[string]interface{}{
		"type":    "connected",
		"message": "Connected to REPL notification stream",
		"data": map[string]interface{}{
			"client_id":  client.ID,
			"session_id": h.session.ID,
		},
	}
	if data, err := json.Marshal(welcome); err == nil {
		client.SendChan <- data
	}

	go h.handleWebSocketClient(client)
}

func (h *HTTPServer) handleWebSocketClient(client *WebSocketClient) {
	defer func() {
		h.mu.Lock()
		delete(h.clients, client.ID)
		h.mu.Unlock()
		if err := client.Conn.Close(); err != nil {
			h.logger.Error("Failed to close WebSocket connection", "error", err)
		}
		h.logger.Info("WebSocket client disconnected", "client_id", client.ID)
	}()

	// Start goroutines for reading and writing
	go h.webSocketReader(client)
	go h.webSocketWriter(client)

	// Wait for close signal
	<-client.CloseChan
}

func (h *HTTPServer) webSocketReader(client *WebSocketClient) {
	defer close(client.CloseChan)

	if err := client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
		h.logger.Error("Failed to set read deadline", "error", err)
		return
	}
	client.Conn.SetPongHandler(func(string) error {
		client.LastPing = time.Now()
		if err := client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
			h.logger.Error("Failed to set read deadline in pong handler", "error", err)
		}
		return nil
	})

	for {
		messageType, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Error("WebSocket read error", "error", err)
			}
			return
		}

		// Handle incoming messages
		if messageType == websocket.TextMessage {
			h.handleWebSocketMessage(client, message)
		}
	}
}

func (h *HTTPServer) webSocketWriter(client *WebSocketClient) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-client.SendChan:
			if err := client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				h.logger.Error("Failed to set write deadline", "error", err)
				return
			}
			if !ok {
				if err := client.Conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					h.logger.Error("Failed to write close message", "error", err)
				}
				return
			}

			if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			if err := client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				h.logger.Error("Failed to set write deadline for ping", "error", err)
				return
			}
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-client.CloseChan:
			return
		}
	}
}

func (h *HTTPServer) handleWebSocketMessage(client *WebSocketClient, message []byte) {
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		h.logger.Error("Invalid WebSocket message", "error", err)
		return
	}

	// Handle different message types
	msgType, ok := msg["type"].(string)
	if !ok {
		return
	}

	switch msgType {
	case "ping":
		// Respond with pong
		pong := map[string]interface{}{
			"type":      "pong",
			"timestamp": time.Now(),
		}
		if data, err := json.Marshal(pong); err == nil {
			client.SendChan <- data
		}

	case "subscribe":
		// Handle subscription requests
		h.logger.Info("WebSocket subscription", "client_id", client.ID, "data", msg)

	default:
		h.logger.Warn("Unknown WebSocket message type", "type", msgType)
	}
}

func (h *HTTPServer) broadcastNotifications() {
	for notification := range h.session.NotificationChan {
		// Convert notification to JSON
		data, err := json.Marshal(notification)
		if err != nil {
			h.logger.Error("Failed to marshal notification", "error", err)
			continue
		}

		// Broadcast to all connected WebSocket clients
		h.mu.RLock()
		for _, client := range h.clients {
			select {
			case client.SendChan <- data:
				// Sent successfully
			default:
				// Client send channel full, close connection
				close(client.CloseChan)
			}
		}
		h.mu.RUnlock()
	}
}

func (h *HTTPServer) handleWebSocketPings() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		h.mu.Lock()
		for id, client := range h.clients {
			if time.Since(client.LastPing) > 90*time.Second {
				h.logger.Warn("WebSocket client timeout", "client_id", id)
				close(client.CloseChan)
			}
		}
		h.mu.Unlock()
	}
}

// Helper methods

func (h *HTTPServer) sendJSONResponse(w http.ResponseWriter, status int, response interface{}) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response", "error", err)
	}
}

func (h *HTTPServer) sendErrorResponse(w http.ResponseWriter, status int, message string) {
	response := HTTPResponse{
		Success: false,
		Error:   message,
	}
	h.sendJSONResponse(w, status, response)
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
