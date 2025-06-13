// Package websocket provides WebSocket hub and client management
// for real-time communication in the MCP Memory Server.
package websocket

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ConnectionMetadata tracks metadata for each connection
type ConnectionMetadata struct {
	RemoteAddr       string    `json:"remote_addr"`
	UserAgent        string    `json:"user_agent"`
	Origin           string    `json:"origin"`
	ConnectedAt      time.Time `json:"connected_at"`
	LastActivity     time.Time `json:"last_activity"`
	Repository       string    `json:"repository"`
	SessionID        string    `json:"session_id"`
	CLIVersion       string    `json:"cli_version"`
	RequestID        string    `json:"request_id"`
	BytesSent        int64     `json:"bytes_sent"`
	BytesReceived    int64     `json:"bytes_received"`
	MessagesSent     int64     `json:"messages_sent"`
	MessagesReceived int64     `json:"messages_received"`
}

// MemoryEvent represents a memory change event
type MemoryEvent struct {
	Type       string      `json:"type"`
	Action     string      `json:"action"` // "created", "updated", "deleted"
	ChunkID    string      `json:"chunk_id,omitempty"`
	Repository string      `json:"repository,omitempty"`
	SessionID  string      `json:"session_id,omitempty"`
	Content    string      `json:"content,omitempty"`
	Summary    string      `json:"summary,omitempty"`
	Tags       []string    `json:"tags,omitempty"`
	Timestamp  time.Time   `json:"timestamp"`
	Data       interface{} `json:"data,omitempty"`
}

// Client represents a WebSocket client
type Client struct {
	ID         string
	Connection *websocket.Conn
	Send       chan MemoryEvent
	Hub        *Hub
	Repository string              // Filter events by repository
	SessionID  string              // Filter events by session
	Metadata   *ConnectionMetadata // Connection metadata for enhanced features
	closed     bool                // Flag to prevent double closing
	mu         sync.Mutex          // Mutex to protect closed flag
}

// SafeClose safely closes the client's send channel
func (c *Client) SafeClose() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.closed && c.Send != nil {
		close(c.Send)
		c.closed = true
	}
}

// Hub manages WebSocket connections and broadcasts
type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan MemoryEvent
	mutex      sync.RWMutex
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan MemoryEvent, 256),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run(ctx context.Context) {
	defer func() {
		// Close all client connections when shutting down
		h.mutex.Lock()
		for client := range h.clients {
			client.SafeClose()
			if err := client.Connection.Close(); err != nil {
				log.Printf("Error closing client connection: %v", err)
			}
		}
		h.mutex.Unlock()
	}()

	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()

			log.Printf("WebSocket client %s registered (total: %d)", client.ID, len(h.clients))

			// Send welcome message
			welcomeEvent := MemoryEvent{
				Type:      "connection",
				Action:    "connected",
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"server":    "mcp-memory",
					"client_id": client.ID,
					"message":   "Connected to memory update stream",
				},
			}

			select {
			case client.Send <- welcomeEvent:
			default:
				h.removeClient(client)
			}

		case client := <-h.unregister:
			h.removeClient(client)

		case event := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				// Filter events based on client preferences
				if h.shouldSendToClient(client, &event) {
					select {
					case client.Send <- event:
					default:
						// Client's send channel is full, remove them
						h.removeClientUnsafe(client)
					}
				}
			}
			h.mutex.RUnlock()

		case <-ctx.Done():
			log.Println("WebSocket hub shutting down")
			return
		}
	}
}

// removeClient safely removes a client from the hub
func (h *Hub) removeClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.removeClientUnsafe(client)
}

// removeClientUnsafe removes a client without locking (assumes lock is held)
func (h *Hub) removeClientUnsafe(client *Client) {
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		client.SafeClose()
		if err := client.Connection.Close(); err != nil {
			log.Printf("Error closing client connection: %v", err)
		}
		log.Printf("WebSocket client %s disconnected (total: %d)", client.ID, len(h.clients))
	}
}

// shouldSendToClient determines if an event should be sent to a specific client
func (h *Hub) shouldSendToClient(client *Client, event *MemoryEvent) bool {
	// Always send connection and system events
	if event.Type == "connection" || event.Type == "system" {
		return true
	}

	// Filter by repository if client has preference
	if client.Repository != "" && event.Repository != "" && client.Repository != event.Repository {
		return false
	}

	// Filter by session if client has preference
	if client.SessionID != "" && event.SessionID != "" && client.SessionID != event.SessionID {
		return false
	}

	return true
}

// RegisterClient registers a new client with the hub
func (h *Hub) RegisterClient(client *Client) {
	h.register <- client
}

// UnregisterClient unregisters a client from the hub
func (h *Hub) UnregisterClient(client *Client) {
	h.unregister <- client
}

// BroadcastMemoryEvent sends a memory event to all connected clients
func (h *Hub) BroadcastMemoryEvent(event *MemoryEvent) {
	select {
	case h.broadcast <- *event:
	default:
		log.Printf("Warning: Broadcast channel full, dropping event %s", event.Type)
	}
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.clients)
}

// NewClient creates a new WebSocket client
func NewClient(id string, conn *websocket.Conn, hub *Hub, repository, sessionID string) *Client {
	return &Client{
		ID:         id,
		Connection: conn,
		Send:       make(chan MemoryEvent, 256),
		Hub:        hub,
		Repository: repository,
		SessionID:  sessionID,
	}
}

// WritePump pumps messages from the hub to the websocket connection
func (c *Client) WritePump(ctx context.Context) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		if err := c.Connection.Close(); err != nil {
			log.Printf("Error closing connection in WritePump: %v", err)
		}
	}()

	for {
		select {
		case event, ok := <-c.Send:
			if err := c.Connection.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				log.Printf("Error setting write deadline: %v", err)
			}
			if !ok {
				// The hub closed the channel
				if err := c.Connection.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					log.Printf("Error writing close message: %v", err)
				}
				return
			}

			if err := c.Connection.WriteJSON(event); err != nil {
				log.Printf("Error writing JSON to WebSocket: %v", err)
				return
			}

		case <-ticker.C:
			if err := c.Connection.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				log.Printf("Error setting write deadline for heartbeat: %v", err)
			}
			heartbeat := MemoryEvent{
				Type:      "heartbeat",
				Timestamp: time.Now(),
			}
			if err := c.Connection.WriteJSON(heartbeat); err != nil {
				log.Printf("Error writing heartbeat: %v", err)
				return
			}

		case <-ctx.Done():
			return
		}
	}
}

// ReadPump pumps messages from the websocket connection to the hub
func (c *Client) ReadPump(ctx context.Context) {
	defer func() {
		c.Hub.unregister <- c
		if err := c.Connection.Close(); err != nil {
			log.Printf("Error closing connection in ReadPump: %v", err)
		}
	}()

	// Set read limits and timeouts
	c.Connection.SetReadLimit(512)
	if err := c.Connection.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
		log.Printf("Error setting read deadline: %v", err)
	}
	c.Connection.SetPongHandler(func(string) error {
		if err := c.Connection.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
			log.Printf("Error setting read deadline in pong handler: %v", err)
		}
		return nil
	})

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Read message from client
			var msg map[string]interface{}
			err := c.Connection.ReadJSON(&msg)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				return
			}

			// Handle client messages (subscription preferences, etc.)
			c.handleClientMessage(msg)
		}
	}
}

// handleClientMessage processes messages from the client
func (c *Client) handleClientMessage(msg map[string]interface{}) {
	msgType, ok := msg["type"].(string)
	if !ok {
		return
	}

	switch msgType {
	case "subscribe":
		// Handle subscription requests
		if repo, ok := msg["repository"].(string); ok {
			c.Repository = repo
			log.Printf("Client %s subscribed to repository: %s", c.ID, repo)
		}
		if session, ok := msg["session_id"].(string); ok {
			c.SessionID = session
			log.Printf("Client %s subscribed to session: %s", c.ID, session)
		}

	case "unsubscribe":
		// Handle unsubscription requests
		if _, ok := msg["repository"]; ok {
			c.Repository = ""
			log.Printf("Client %s unsubscribed from repository", c.ID)
		}
		if _, ok := msg["session_id"]; ok {
			c.SessionID = ""
			log.Printf("Client %s unsubscribed from session", c.ID)
		}

	case "ping":
		// Respond to ping with pong
		pong := MemoryEvent{
			Type:      "pong",
			Timestamp: time.Now(),
		}
		select {
		case c.Send <- pong:
		default:
			// Channel full, client will be removed
		}
	}
}

// NewMemoryEvent creates a new memory event with the specified parameters
func NewMemoryEvent(eventType, action, chunkID, repository, sessionID string, data interface{}) MemoryEvent {
	return MemoryEvent{
		Type:       eventType,
		Action:     action,
		ChunkID:    chunkID,
		Repository: repository,
		SessionID:  sessionID,
		Timestamp:  time.Now(),
		Data:       data,
	}
}
