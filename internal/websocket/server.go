// Package websocket provides WebSocket server implementation and connection management
// for real-time communication in the MCP Memory Server.
package websocket

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ServerConfig represents WebSocket server configuration
type ServerConfig struct {
	MaxConnections    int           `json:"max_connections"`
	ReadBufferSize    int           `json:"read_buffer_size"`
	WriteBufferSize   int           `json:"write_buffer_size"`
	HandshakeTimeout  time.Duration `json:"handshake_timeout"`
	PingInterval      time.Duration `json:"ping_interval"`
	PongTimeout       time.Duration `json:"pong_timeout"`
	WriteTimeout      time.Duration `json:"write_timeout"`
	ReadTimeout       time.Duration `json:"read_timeout"`
	EnableCompression bool          `json:"enable_compression"`
	MaxMessageSize    int64         `json:"max_message_size"`
	EnableAuth        bool          `json:"enable_auth"`
	RequiredVersion   string        `json:"required_version"`
	AllowedOrigins    []string      `json:"allowed_origins"`
}

// DefaultServerConfig returns default WebSocket server configuration
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		MaxConnections:    1000,
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		HandshakeTimeout:  10 * time.Second,
		PingInterval:      54 * time.Second,
		PongTimeout:       60 * time.Second,
		WriteTimeout:      10 * time.Second,
		ReadTimeout:       60 * time.Second,
		EnableCompression: true,
		MaxMessageSize:    512,
		EnableAuth:        true,
		RequiredVersion:   "1.0.0",
		AllowedOrigins:    []string{"*"},
	}
}

// Server represents the WebSocket server
type Server struct {
	config           *ServerConfig
	upgrader         websocket.Upgrader
	hub              *Hub
	pool             *ConnectionPool
	metricsCollector *MetricsCollector
	heartbeat        *HeartbeatManager
	ctx              context.Context
	cancel           context.CancelFunc
	mu               sync.RWMutex
	running          bool
}

// NewServer creates a new WebSocket server
func NewServer(config *ServerConfig) *Server {
	if config == nil {
		config = DefaultServerConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	upgrader := websocket.Upgrader{
		ReadBufferSize:    config.ReadBufferSize,
		WriteBufferSize:   config.WriteBufferSize,
		HandshakeTimeout:  config.HandshakeTimeout,
		EnableCompression: config.EnableCompression,
		CheckOrigin: func(r *http.Request) bool {
			return checkOrigin(r, config.AllowedOrigins)
		},
	}

	hub := NewHub()
	pool := NewConnectionPool(config.MaxConnections)
	metricsCollector := NewMetricsCollector(nil)
	heartbeat := NewHeartbeatManager(config.PingInterval, config.PongTimeout)

	return &Server{
		config:           config,
		upgrader:         upgrader,
		hub:              hub,
		pool:             pool,
		metricsCollector: metricsCollector,
		heartbeat:        heartbeat,
		ctx:              ctx,
		cancel:           cancel,
	}
}

// Start starts the WebSocket server
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("WebSocket server is already running")
	}

	log.Println("Starting WebSocket server...")

	// Start hub
	go s.hub.Run(s.ctx)

	// Start heartbeat manager
	go s.heartbeat.Start(s.ctx)

	// Start metrics collection
	// Metrics collector is started automatically

	s.running = true
	log.Println("WebSocket server started successfully")

	return nil
}

// Stop stops the WebSocket server gracefully
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("WebSocket server is not running")
	}

	log.Println("Stopping WebSocket server...")

	// Cancel context to stop all goroutines
	s.cancel()

	// Close all connections gracefully
	s.pool.CloseAll()

	s.running = false
	log.Println("WebSocket server stopped successfully")

	return nil
}

// HandleUpgrade handles WebSocket upgrade requests
func (s *Server) HandleUpgrade(w http.ResponseWriter, r *http.Request) {
	// Check if server is running
	s.mu.RLock()
	if !s.running {
		s.mu.RUnlock()
		http.Error(w, "WebSocket server not running", http.StatusServiceUnavailable)
		return
	}
	s.mu.RUnlock()

	// Check connection limit
	if !s.pool.CanAcceptConnection() {
		// Connection rejected due to pool being full
		http.Error(w, "Connection limit reached", http.StatusServiceUnavailable)
		return
	}

	// Authenticate connection if enabled
	if s.config.EnableAuth {
		if err := s.authenticateConnection(r); err != nil {
			// Connection rejected due to authentication failure
			http.Error(w, "Authentication failed", http.StatusUnauthorized)
			return
		}
	}

	// Upgrade connection
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		// Connection upgrade failed
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Set connection options
	conn.SetReadLimit(s.config.MaxMessageSize)
	if err := conn.SetReadDeadline(time.Now().Add(s.config.ReadTimeout)); err != nil {
		log.Printf("Failed to set read deadline: %v", err)
	}
	conn.SetPongHandler(func(string) error {
		if err := conn.SetReadDeadline(time.Now().Add(s.config.ReadTimeout)); err != nil {
			log.Printf("Failed to set read deadline in pong handler: %v", err)
		}
		return nil
	})

	// Create connection metadata
	metadata := &ConnectionMetadata{
		RemoteAddr:  r.RemoteAddr,
		UserAgent:   r.UserAgent(),
		Origin:      r.Header.Get("Origin"),
		ConnectedAt: time.Now(),
		Repository:  r.URL.Query().Get("repository"),
		SessionID:   r.URL.Query().Get("session_id"),
		CLIVersion:  r.Header.Get("X-CLI-Version"),
		RequestID:   r.Header.Get("X-Request-ID"),
	}

	// Create client
	client := NewClient(generateClientID(), conn, s.hub, metadata.Repository, metadata.SessionID)
	client.Metadata = metadata

	// Add to pool
	s.pool.AddConnection(client)

	// Register with hub
	s.hub.RegisterClient(client)

	// Start client pumps
	go client.WritePump(s.ctx)
	go client.ReadPump(s.ctx)

	// Start heartbeat monitoring for this client
	s.heartbeat.AddClient(client)

	// Record metrics
	// Register connection with metrics collector
	s.metricsCollector.RegisterConnection(client.ID)

	log.Printf("WebSocket client %s connected from %s", client.ID, metadata.RemoteAddr)
}

// authenticateConnection validates the WebSocket connection
func (s *Server) authenticateConnection(r *http.Request) error {
	if !s.config.EnableAuth {
		return nil
	}

	// Check CLI version
	cliVersion := r.Header.Get("X-CLI-Version")
	if cliVersion == "" {
		return fmt.Errorf("missing CLI version header")
	}

	// Validate version (simplified - in production, use semantic versioning)
	if s.config.RequiredVersion != "" && cliVersion != s.config.RequiredVersion {
		return fmt.Errorf("incompatible CLI version: %s, required: %s", cliVersion, s.config.RequiredVersion)
	}

	// Additional authentication logic can be added here
	// e.g., API key validation, JWT token verification, etc.

	return nil
}

// checkOrigin validates the request origin
func checkOrigin(r *http.Request, allowedOrigins []string) bool {
	origin := r.Header.Get("Origin")

	// Allow requests without origin (e.g., from command line tools)
	if origin == "" {
		return true
	}

	// Check allowed origins
	for _, allowed := range allowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}

	return false
}

// generateClientID generates a unique client ID
func generateClientID() string {
	return fmt.Sprintf("client_%d", time.Now().UnixNano())
}

// GetHub returns the WebSocket hub
func (s *Server) GetHub() *Hub {
	return s.hub
}

// GetPool returns the connection pool
func (s *Server) GetPool() *ConnectionPool {
	return s.pool
}

// GetMetrics returns the server metrics
func (s *Server) GetMetrics() *SystemMetrics {
	return s.metricsCollector.GetSystemMetrics()
}

// GetConfig returns the server configuration
func (s *Server) GetConfig() *ServerConfig {
	return s.config
}

// IsRunning returns whether the server is running
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetConnectionCount returns the current number of connections
func (s *Server) GetConnectionCount() int {
	return s.pool.GetConnectionCount()
}

// BroadcastEvent broadcasts an event to all connected clients
func (s *Server) BroadcastEvent(event *MemoryEvent) {
	s.hub.BroadcastMemoryEvent(event)
}
