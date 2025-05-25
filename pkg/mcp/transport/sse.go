// Package transport implements MCP transport layers
package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"mcp-memory/pkg/mcp/protocol"
)

// SSEConfig contains configuration for Server-Sent Events transport
type SSEConfig struct {
	// Base HTTP configuration
	HTTPConfig

	// SSE-specific settings
	HeartbeatInterval time.Duration
	
	// Maximum number of clients
	MaxClients int

	// Event buffer size per client
	EventBufferSize int

	// Retry delay for client reconnection (sent in SSE)
	RetryDelay time.Duration

	// Path to handle SSE connections (default: "/events")
	EventPath string
}

// SSETransport implements Server-Sent Events transport for MCP
type SSETransport struct {
	*HTTPTransport
	sseConfig *SSEConfig
	
	// Client management
	clients map[string]*sseClient
	clientMu sync.RWMutex
	
	// Event broadcasting
	broadcast chan *sseEvent
}

// sseClient represents a connected SSE client
type sseClient struct {
	id            string
	events        chan *sseEvent
	done          chan struct{}
	lastEventID   string
	writer        http.ResponseWriter
	flusher       http.Flusher
}

// sseEvent represents an SSE event
type sseEvent struct {
	ID    string
	Type  string
	Data  interface{}
	Retry int
}

// NewSSETransport creates a new SSE transport
func NewSSETransport(config *SSEConfig) *SSETransport {
	// Set defaults
	if config.HeartbeatInterval == 0 {
		config.HeartbeatInterval = 30 * time.Second
	}
	if config.MaxClients == 0 {
		config.MaxClients = 1000
	}
	if config.EventBufferSize == 0 {
		config.EventBufferSize = 100
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 5 * time.Second
	}
	if config.EventPath == "" {
		config.EventPath = "/events"
	}

	// Create base HTTP transport
	httpTransport := NewHTTPTransport(&config.HTTPConfig)

	return &SSETransport{
		HTTPTransport: httpTransport,
		sseConfig:     config,
		clients:       make(map[string]*sseClient),
		broadcast:     make(chan *sseEvent, 1000),
	}
}

// Start starts the SSE transport
func (t *SSETransport) Start(ctx context.Context, handler RequestHandler) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.running {
		return fmt.Errorf("transport already running")
	}

	t.handler = handler

	// Start broadcast handler
	go t.broadcastHandler()

	mux := http.NewServeMux()
	
	// SSE endpoint
	mux.HandleFunc(t.sseConfig.EventPath, t.handleSSE)
	
	// Regular HTTP endpoint for sending commands
	mux.HandleFunc(t.config.Path, t.handleCommand)

	t.server = &http.Server{
		Addr:         t.config.Address,
		Handler:      t.wrapWithMiddleware(mux),
		ReadTimeout:  t.config.ReadTimeout,
		WriteTimeout: 0, // Disable write timeout for SSE
		IdleTimeout:  t.config.IdleTimeout,
		TLSConfig:    t.config.TLSConfig,
	}

	t.running = true

	// Start server
	go func() {
		var err error
		if t.certFile != "" && t.keyFile != "" {
			err = t.server.ListenAndServeTLS(t.certFile, t.keyFile)
		} else {
			err = t.server.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			t.mu.Lock()
			t.running = false
			t.mu.Unlock()
		}
	}()

	// Wait for context cancellation
	go func() {
		<-ctx.Done()
		t.Stop()
	}()

	return nil
}

// Stop stops the SSE transport
func (t *SSETransport) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.running {
		return nil
	}

	t.running = false

	// Close broadcast channel
	close(t.broadcast)

	// Disconnect all clients
	t.clientMu.Lock()
	for _, client := range t.clients {
		close(client.done)
	}
	t.clients = make(map[string]*sseClient)
	t.clientMu.Unlock()

	// Stop HTTP server
	if t.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return t.server.Shutdown(ctx)
	}

	return nil
}

// handleSSE handles SSE connections
func (t *SSETransport) handleSSE(w http.ResponseWriter, r *http.Request) {
	// Check if client supports SSE
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Check max clients
	t.clientMu.RLock()
	clientCount := len(t.clients)
	t.clientMu.RUnlock()

	if clientCount >= t.sseConfig.MaxClients {
		http.Error(w, "Maximum clients reached", http.StatusServiceUnavailable)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable Nginx buffering

	// Create client
	clientID := generateID()
	client := &sseClient{
		id:          clientID,
		events:      make(chan *sseEvent, t.sseConfig.EventBufferSize),
		done:        make(chan struct{}),
		lastEventID: r.Header.Get("Last-Event-ID"),
		writer:      w,
		flusher:     flusher,
	}

	// Register client
	t.clientMu.Lock()
	t.clients[clientID] = client
	t.clientMu.Unlock()

	// Send initial connection event
	t.sendToClient(client, &sseEvent{
		Type: "connected",
		Data: map[string]interface{}{
			"clientId": clientID,
			"retry":    int(t.sseConfig.RetryDelay / time.Millisecond),
		},
	})

	// Send server capabilities
	initReq := &protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      "sse-init-" + clientID,
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": protocol.Version,
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "SSE Client",
				"version": "1.0.0",
			},
		},
	}

	resp := t.handler.HandleRequest(r.Context(), initReq)
	if resp.Result != nil {
		t.sendToClient(client, &sseEvent{
			Type: "capabilities",
			Data: resp.Result,
		})
	}

	// Handle client
	t.handleClient(client, r.Context())

	// Cleanup
	t.clientMu.Lock()
	delete(t.clients, clientID)
	t.clientMu.Unlock()
	close(client.events)
}

// handleClient handles a connected SSE client
func (t *SSETransport) handleClient(client *sseClient, ctx context.Context) {
	ticker := time.NewTicker(t.sseConfig.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-client.done:
			return
		case event := <-client.events:
			if err := t.writeSSEEvent(client, event); err != nil {
				return
			}
		case <-ticker.C:
			// Send heartbeat
			if err := t.writeSSEComment(client, "heartbeat"); err != nil {
				return
			}
		}
	}
}

// handleCommand handles command requests via regular HTTP POST
func (t *SSETransport) handleCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get client ID from header or query
	clientID := r.Header.Get("X-Client-ID")
	if clientID == "" {
		clientID = r.URL.Query().Get("client_id")
	}

	// Parse request
	var req protocol.JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		t.writeError(w, protocol.ParseError, "Invalid JSON", nil)
		return
	}

	// Handle request
	resp := t.handler.HandleRequest(r.Context(), &req)

	// Send response via SSE if client ID provided
	if clientID != "" {
		t.clientMu.RLock()
		client, exists := t.clients[clientID]
		t.clientMu.RUnlock()

		if exists {
			event := &sseEvent{
				ID:   fmt.Sprintf("%v", req.ID),
				Type: "response",
				Data: resp,
			}
			
			select {
			case client.events <- event:
				// Response will be sent via SSE
				w.WriteHeader(http.StatusAccepted)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status": "accepted",
					"id":     req.ID,
				})
				return
			default:
				// Client buffer full
			}
		}
	}

	// Send response via HTTP
	t.writeResponse(w, resp)
}

// broadcastHandler handles event broadcasting
func (t *SSETransport) broadcastHandler() {
	for event := range t.broadcast {
		t.clientMu.RLock()
		clients := make([]*sseClient, 0, len(t.clients))
		for _, client := range t.clients {
			clients = append(clients, client)
		}
		t.clientMu.RUnlock()

		// Send to all clients
		for _, client := range clients {
			select {
			case client.events <- event:
				// Event queued
			default:
				// Client buffer full, skip
			}
		}
	}
}

// BroadcastEvent broadcasts an event to all connected clients
func (t *SSETransport) BroadcastEvent(eventType string, data interface{}) {
	event := &sseEvent{
		ID:   generateID(),
		Type: eventType,
		Data: data,
	}

	select {
	case t.broadcast <- event:
		// Event queued for broadcast
	default:
		// Broadcast buffer full
	}
}

// SendToClient sends an event to a specific client
func (t *SSETransport) SendToClient(clientID string, eventType string, data interface{}) error {
	t.clientMu.RLock()
	client, exists := t.clients[clientID]
	t.clientMu.RUnlock()

	if !exists {
		return fmt.Errorf("client not found: %s", clientID)
	}

	event := &sseEvent{
		ID:   generateID(),
		Type: eventType,
		Data: data,
	}

	select {
	case client.events <- event:
		return nil
	default:
		return fmt.Errorf("client buffer full")
	}
}

// sendToClient sends an event to a client (internal)
func (t *SSETransport) sendToClient(client *sseClient, event *sseEvent) {
	select {
	case client.events <- event:
		// Event queued
	default:
		// Buffer full, drop event
	}
}

// writeSSEEvent writes an SSE event
func (t *SSETransport) writeSSEEvent(client *sseClient, event *sseEvent) error {
	// Write event ID if provided
	if event.ID != "" {
		if _, err := fmt.Fprintf(client.writer, "id: %s\n", event.ID); err != nil {
			return err
		}
		client.lastEventID = event.ID
	}

	// Write event type
	if event.Type != "" {
		if _, err := fmt.Fprintf(client.writer, "event: %s\n", event.Type); err != nil {
			return err
		}
	}

	// Write retry if provided
	if event.Retry > 0 {
		if _, err := fmt.Fprintf(client.writer, "retry: %d\n", event.Retry); err != nil {
			return err
		}
	}

	// Write data
	data, err := json.Marshal(event.Data)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(client.writer, "data: %s\n\n", data); err != nil {
		return err
	}

	// Flush
	client.flusher.Flush()
	return nil
}

// writeSSEComment writes an SSE comment (for keepalive)
func (t *SSETransport) writeSSEComment(client *sseClient, comment string) error {
	if _, err := fmt.Fprintf(client.writer, ": %s\n\n", comment); err != nil {
		return err
	}
	client.flusher.Flush()
	return nil
}

// ClientCount returns the number of connected SSE clients
func (t *SSETransport) ClientCount() int {
	t.clientMu.RLock()
	defer t.clientMu.RUnlock()
	return len(t.clients)
}

// GetClients returns a list of connected client IDs
func (t *SSETransport) GetClients() []string {
	t.clientMu.RLock()
	defer t.clientMu.RUnlock()

	clients := make([]string, 0, len(t.clients))
	for id := range t.clients {
		clients = append(clients, id)
	}
	return clients
}