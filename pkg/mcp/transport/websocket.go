// Package transport implements MCP transport layers
package transport

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"mcp-memory/pkg/mcp/protocol"
)

// WebSocketConfig contains configuration for WebSocket transport
type WebSocketConfig struct {
	// Address to listen on
	Address string

	// TLS configuration (optional)
	TLSConfig *tls.Config

	// WebSocket upgrade configuration
	ReadBufferSize  int
	WriteBufferSize int

	// Timeouts
	HandshakeTimeout time.Duration
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	PingInterval     time.Duration
	PongTimeout      time.Duration

	// Maximum message size (default: 10MB)
	MaxMessageSize int64

	// Enable compression
	EnableCompression bool

	// Check origin function
	CheckOrigin func(r *http.Request) bool

	// Path to handle WebSocket connections (default: "/ws")
	Path string
}

// WebSocketTransport implements WebSocket transport for MCP
type WebSocketTransport struct {
	config   *WebSocketConfig
	server   *http.Server
	upgrader *websocket.Upgrader
	handler  RequestHandler
	mu       sync.RWMutex
	running  bool
	certFile string
	keyFile  string
	
	// Active connections
	connections map[*websocket.Conn]bool
	connMu      sync.RWMutex
}

// NewWebSocketTransport creates a new WebSocket transport
func NewWebSocketTransport(config *WebSocketConfig) *WebSocketTransport {
	// Set defaults
	if config.ReadBufferSize == 0 {
		config.ReadBufferSize = 4096
	}
	if config.WriteBufferSize == 0 {
		config.WriteBufferSize = 4096
	}
	if config.MaxMessageSize == 0 {
		config.MaxMessageSize = 10 * 1024 * 1024 // 10MB
	}
	if config.HandshakeTimeout == 0 {
		config.HandshakeTimeout = 10 * time.Second
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 60 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 10 * time.Second
	}
	if config.PingInterval == 0 {
		config.PingInterval = 30 * time.Second
	}
	if config.PongTimeout == 0 {
		config.PongTimeout = 60 * time.Second
	}
	if config.Path == "" {
		config.Path = "/ws"
	}
	if config.CheckOrigin == nil {
		config.CheckOrigin = func(r *http.Request) bool { return true }
	}

	upgrader := &websocket.Upgrader{
		ReadBufferSize:    config.ReadBufferSize,
		WriteBufferSize:   config.WriteBufferSize,
		HandshakeTimeout:  config.HandshakeTimeout,
		EnableCompression: config.EnableCompression,
		CheckOrigin:       config.CheckOrigin,
	}

	return &WebSocketTransport{
		config:      config,
		upgrader:    upgrader,
		connections: make(map[*websocket.Conn]bool),
	}
}

// NewSecureWebSocketTransport creates a new secure WebSocket transport
func NewSecureWebSocketTransport(config *WebSocketConfig, certFile, keyFile string) *WebSocketTransport {
	transport := NewWebSocketTransport(config)
	transport.certFile = certFile
	transport.keyFile = keyFile
	return transport
}

// Start starts the WebSocket transport
func (t *WebSocketTransport) Start(ctx context.Context, handler RequestHandler) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.running {
		return fmt.Errorf("transport already running")
	}

	t.handler = handler

	mux := http.NewServeMux()
	mux.HandleFunc(t.config.Path, t.handleWebSocket)

	t.server = &http.Server{
		Addr:      t.config.Address,
		Handler:   mux,
		TLSConfig: t.config.TLSConfig,
	}

	t.running = true

	// Start server in a goroutine
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

// Stop stops the WebSocket transport
func (t *WebSocketTransport) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.running {
		return nil
	}

	t.running = false

	// Close all active connections
	t.connMu.Lock()
	for conn := range t.connections {
		conn.Close()
	}
	t.connections = make(map[*websocket.Conn]bool)
	t.connMu.Unlock()

	if t.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return t.server.Shutdown(ctx)
	}

	return nil
}

// handleWebSocket handles WebSocket upgrade and connection
func (t *WebSocketTransport) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := t.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	// Register connection
	t.connMu.Lock()
	t.connections[conn] = true
	t.connMu.Unlock()

	// Handle connection
	go t.handleConnection(conn)
}

// handleConnection handles a WebSocket connection
func (t *WebSocketTransport) handleConnection(conn *websocket.Conn) {
	defer func() {
		// Unregister connection
		t.connMu.Lock()
		delete(t.connections, conn)
		t.connMu.Unlock()
		conn.Close()
	}()

	// Set connection parameters
	conn.SetReadLimit(t.config.MaxMessageSize)
	conn.SetReadDeadline(time.Now().Add(t.config.PongTimeout))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(t.config.PongTimeout))
		return nil
	})

	// Start ping ticker
	ticker := time.NewTicker(t.config.PingInterval)
	defer ticker.Stop()

	// Message handling
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				return
			}

			if messageType != websocket.TextMessage {
				continue
			}

			// Parse JSON-RPC request
			var req protocol.JSONRPCRequest
			if err := json.Unmarshal(message, &req); err != nil {
				t.sendError(conn, nil, protocol.ParseError, "Invalid JSON", nil)
				continue
			}

			// Handle request
			ctx := context.Background()
			resp := t.handler.HandleRequest(ctx, &req)

			// Send response
			if err := t.sendResponse(conn, resp); err != nil {
				return
			}
		}
	}()

	// Ping loop
	for {
		select {
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(t.config.WriteTimeout))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-done:
			return
		}
	}
}

// sendResponse sends a JSON-RPC response
func (t *WebSocketTransport) sendResponse(conn *websocket.Conn, resp *protocol.JSONRPCResponse) error {
	conn.SetWriteDeadline(time.Now().Add(t.config.WriteTimeout))
	return conn.WriteJSON(resp)
}

// sendError sends a JSON-RPC error response
func (t *WebSocketTransport) sendError(conn *websocket.Conn, id interface{}, code int, message string, data interface{}) error {
	resp := &protocol.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   protocol.NewJSONRPCError(code, message, data),
	}
	return t.sendResponse(conn, resp)
}

// Broadcast sends a message to all connected clients
func (t *WebSocketTransport) Broadcast(message interface{}) error {
	t.connMu.RLock()
	defer t.connMu.RUnlock()

	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	for conn := range t.connections {
		conn.SetWriteDeadline(time.Now().Add(t.config.WriteTimeout))
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			// Connection might be dead, will be cleaned up by handleConnection
			continue
		}
	}

	return nil
}

// IsRunning returns whether the transport is running
func (t *WebSocketTransport) IsRunning() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.running
}

// ConnectionCount returns the number of active connections
func (t *WebSocketTransport) ConnectionCount() int {
	t.connMu.RLock()
	defer t.connMu.RUnlock()
	return len(t.connections)
}