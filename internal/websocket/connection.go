// Package websocket provides enhanced connection management for WebSocket clients
package websocket

import (
	"context"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// Enhanced Client structure with additional features (extends the existing hub.go Client)
// Note: This works alongside the existing Client struct in hub.go

// ConnectionState represents the state of a WebSocket connection
type ConnectionState int

const (
	StateConnecting ConnectionState = iota
	StateConnected
	StateDisconnecting
	StateDisconnected
	StateError
)

// String returns the string representation of connection state
func (s ConnectionState) String() string {
	switch s {
	case StateConnecting:
		return "connecting"
	case StateConnected:
		return "connected"
	case StateDisconnecting:
		return "disconnecting"
	case StateDisconnected:
		return "disconnected"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// EnhancedClient extends the basic Client with additional functionality
type EnhancedClient struct {
	*Client                    // Embed the original Client
	State       ConnectionState
	Metadata    *ConnectionMetadata
	metrics     *ClientMetrics
	rateLimiter *ClientRateLimiter
	lastActivity time.Time
	errorCount  int
	maxErrors   int
}

// ClientMetrics tracks per-client metrics
type ClientMetrics struct {
	MessagesReceived int64         `json:"messages_received"`
	MessagesSent     int64         `json:"messages_sent"`
	BytesReceived    int64         `json:"bytes_received"`
	BytesSent        int64         `json:"bytes_sent"`
	Errors           int64         `json:"errors"`
	Latency          time.Duration `json:"latency"`
	Uptime           time.Duration `json:"uptime"`
	ConnectedAt      time.Time     `json:"connected_at"`
}

// ClientRateLimiter provides per-client rate limiting
type ClientRateLimiter struct {
	messagesPerSecond int
	lastMessageTime   time.Time
	messageCount      int
	windowStart       time.Time
}

// NewEnhancedClient creates a new enhanced WebSocket client
func NewEnhancedClient(id string, conn *websocket.Conn, hub *Hub, repository, sessionID string) *EnhancedClient {
	baseClient := NewClient(id, conn, hub, repository, sessionID)
	
	return &EnhancedClient{
		Client:       baseClient,
		State:        StateConnected,
		metrics:      &ClientMetrics{ConnectedAt: time.Now()},
		rateLimiter:  NewClientRateLimiter(10), // 10 messages per second default
		lastActivity: time.Now(),
		maxErrors:    5, // Maximum errors before disconnection
	}
}

// NewClientRateLimiter creates a new client rate limiter
func NewClientRateLimiter(messagesPerSecond int) *ClientRateLimiter {
	return &ClientRateLimiter{
		messagesPerSecond: messagesPerSecond,
		windowStart:       time.Now(),
	}
}

// CheckRateLimit checks if a message is within rate limits
func (rl *ClientRateLimiter) CheckRateLimit() bool {
	now := time.Now()
	
	// Reset window if a second has passed
	if now.Sub(rl.windowStart) >= time.Second {
		rl.messageCount = 0
		rl.windowStart = now
	}
	
	// Check if within limits
	if rl.messageCount >= rl.messagesPerSecond {
		return false
	}
	
	rl.messageCount++
	rl.lastMessageTime = now
	return true
}

// EnhancedWritePump provides enhanced message writing with metrics and rate limiting
func (ec *EnhancedClient) EnhancedWritePump(ctx context.Context) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		ec.setState(StateDisconnecting)
		if err := ec.Connection.Close(); err != nil {
			log.Printf("Error closing connection in Enhanced WritePump: %v", err)
		}
		ec.setState(StateDisconnected)
	}()

	for {
		select {
		case event, ok := <-ec.Send:
			if err := ec.Connection.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				log.Printf("Error setting write deadline: %v", err)
				ec.recordError()
				return
			}
			
			if !ok {
				// The hub closed the channel
				if err := ec.Connection.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					log.Printf("Error writing close message: %v", err)
				}
				return
			}

			// Record message being sent
			messageSize := ec.estimateMessageSize(event)
			ec.recordMessageSent(messageSize)

			if err := ec.Connection.WriteJSON(event); err != nil {
				log.Printf("Error writing JSON to WebSocket: %v", err)
				ec.recordError()
				return
			}

		case <-ticker.C:
			if err := ec.Connection.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				log.Printf("Error setting write deadline for heartbeat: %v", err)
				ec.recordError()
				continue
			}
			
			heartbeat := MemoryEvent{
				Type:      "heartbeat",
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"client_id": ec.ID,
					"uptime":    time.Since(ec.metrics.ConnectedAt).String(),
				},
			}
			
			if err := ec.Connection.WriteJSON(heartbeat); err != nil {
				log.Printf("Error writing heartbeat: %v", err)
				ec.recordError()
				return
			}
			
			ec.recordMessageSent(int64(len("heartbeat")))

		case <-ctx.Done():
			return
		}
	}
}

// EnhancedReadPump provides enhanced message reading with metrics and rate limiting
func (ec *EnhancedClient) EnhancedReadPump(ctx context.Context) {
	defer func() {
		ec.setState(StateDisconnecting)
		ec.Hub.unregister <- ec.Client
		if err := ec.Connection.Close(); err != nil {
			log.Printf("Error closing connection in Enhanced ReadPump: %v", err)
		}
		ec.setState(StateDisconnected)
	}()

	// Set read limits and timeouts
	ec.Connection.SetReadLimit(512)
	if err := ec.Connection.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
		log.Printf("Error setting read deadline: %v", err)
	}
	
	ec.Connection.SetPongHandler(func(string) error {
		if err := ec.Connection.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
			log.Printf("Error setting read deadline in pong handler: %v", err)
		}
		ec.updateActivity()
		return nil
	})

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Check rate limit before reading
			if !ec.rateLimiter.CheckRateLimit() {
				log.Printf("Rate limit exceeded for client %s", ec.ID)
				ec.recordError()
				time.Sleep(100 * time.Millisecond) // Brief pause
				continue
			}

			// Read message from client
			var msg map[string]interface{}
			err := ec.Connection.ReadJSON(&msg)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				ec.recordError()
				return
			}

			// Record message received
			messageSize := ec.estimateJSONSize(msg)
			ec.recordMessageReceived(messageSize)
			ec.updateActivity()

			// Handle client messages
			ec.handleEnhancedClientMessage(msg)
		}
	}
}

// handleEnhancedClientMessage processes messages with enhanced features
func (ec *EnhancedClient) handleEnhancedClientMessage(msg map[string]interface{}) {
	msgType, ok := msg["type"].(string)
	if !ok {
		ec.recordError()
		return
	}

	switch msgType {
	case "subscribe":
		// Handle subscription requests
		if repo, ok := msg["repository"].(string); ok {
			ec.Repository = repo
			if ec.Metadata != nil {
				ec.Metadata.Repository = repo
			}
			log.Printf("Enhanced client %s subscribed to repository: %s", ec.ID, repo)
		}
		if session, ok := msg["session_id"].(string); ok {
			ec.SessionID = session
			if ec.Metadata != nil {
				ec.Metadata.SessionID = session
			}
			log.Printf("Enhanced client %s subscribed to session: %s", ec.ID, session)
		}

	case "unsubscribe":
		// Handle unsubscription requests
		if _, ok := msg["repository"]; ok {
			ec.Repository = ""
			if ec.Metadata != nil {
				ec.Metadata.Repository = ""
			}
			log.Printf("Enhanced client %s unsubscribed from repository", ec.ID)
		}
		if _, ok := msg["session_id"]; ok {
			ec.SessionID = ""
			if ec.Metadata != nil {
				ec.Metadata.SessionID = ""
			}
			log.Printf("Enhanced client %s unsubscribed from session", ec.ID)
		}

	case "ping":
		// Respond to ping with pong and latency info
		startTime := time.Now()
		pong := MemoryEvent{
			Type:      "pong",
			Timestamp: startTime,
			Data: map[string]interface{}{
				"client_id": ec.ID,
				"server_time": startTime.Unix(),
			},
		}
		select {
		case ec.Send <- pong:
			// Calculate and record latency
			if pingTime, ok := msg["timestamp"].(float64); ok {
				latency := startTime.Sub(time.Unix(int64(pingTime), 0))
				ec.recordLatency(latency)
			}
		default:
			// Channel full, client will be removed
			ec.recordError()
		}

	case "metrics_request":
		// Handle metrics request
		ec.sendMetrics()

	case "rate_limit_request":
		// Handle rate limit adjustment request
		if newLimit, ok := msg["messages_per_second"].(float64); ok && newLimit > 0 && newLimit <= 100 {
			ec.rateLimiter.messagesPerSecond = int(newLimit)
			log.Printf("Updated rate limit for client %s to %d msg/sec", ec.ID, int(newLimit))
		}

	default:
		// Unknown message type
		log.Printf("Unknown message type from client %s: %s", ec.ID, msgType)
	}
}

// setState updates the connection state
func (ec *EnhancedClient) setState(state ConnectionState) {
	ec.State = state
	log.Printf("Client %s state changed to: %s", ec.ID, state.String())
}

// updateActivity updates the last activity timestamp
func (ec *EnhancedClient) updateActivity() {
	ec.lastActivity = time.Now()
	if ec.Metadata != nil {
		ec.Metadata.LastActivity = ec.lastActivity
	}
}

// recordMessageSent records metrics for a sent message
func (ec *EnhancedClient) recordMessageSent(size int64) {
	ec.metrics.MessagesSent++
	ec.metrics.BytesSent += size
	if ec.Metadata != nil {
		ec.Metadata.MessagesSent++
		ec.Metadata.BytesSent += size
	}
}

// recordMessageReceived records metrics for a received message
func (ec *EnhancedClient) recordMessageReceived(size int64) {
	ec.metrics.MessagesReceived++
	ec.metrics.BytesReceived += size
	if ec.Metadata != nil {
		ec.Metadata.MessagesReceived++
		ec.Metadata.BytesReceived += size
	}
}

// recordLatency records latency measurement
func (ec *EnhancedClient) recordLatency(latency time.Duration) {
	ec.metrics.Latency = latency
	log.Printf("Client %s latency: %v", ec.ID, latency)
}

// recordError records an error occurrence
func (ec *EnhancedClient) recordError() {
	ec.errorCount++
	ec.metrics.Errors++
	
	// Check if max errors exceeded
	if ec.errorCount >= ec.maxErrors {
		log.Printf("Client %s exceeded max errors (%d), marking for disconnection", ec.ID, ec.maxErrors)
		ec.setState(StateError)
	}
}

// sendMetrics sends current metrics to the client
func (ec *EnhancedClient) sendMetrics() {
	metrics := ec.GetMetrics()
	metricsEvent := MemoryEvent{
		Type:      "metrics",
		Timestamp: time.Now(),
		Data:      metrics,
	}
	
	select {
	case ec.Send <- metricsEvent:
		log.Printf("Sent metrics to client %s", ec.ID)
	default:
		log.Printf("Could not send metrics to client %s (channel full)", ec.ID)
	}
}

// GetMetrics returns current client metrics
func (ec *EnhancedClient) GetMetrics() *ClientMetrics {
	uptime := time.Since(ec.metrics.ConnectedAt)
	
	return &ClientMetrics{
		MessagesReceived: ec.metrics.MessagesReceived,
		MessagesSent:     ec.metrics.MessagesSent,
		BytesReceived:    ec.metrics.BytesReceived,
		BytesSent:        ec.metrics.BytesSent,
		Errors:           ec.metrics.Errors,
		Latency:          ec.metrics.Latency,
		Uptime:           uptime,
		ConnectedAt:      ec.metrics.ConnectedAt,
	}
}

// IsHealthy checks if the client is in a healthy state
func (ec *EnhancedClient) IsHealthy() bool {
	return ec.State == StateConnected && ec.errorCount < ec.maxErrors
}

// GetIdleTime returns how long the client has been idle
func (ec *EnhancedClient) GetIdleTime() time.Duration {
	return time.Since(ec.lastActivity)
}

// estimateMessageSize estimates the size of a MemoryEvent
func (ec *EnhancedClient) estimateMessageSize(event MemoryEvent) int64 {
	// Rough estimation based on string lengths
	size := int64(len(event.Type) + len(event.Action) + len(event.ChunkID) + 
		len(event.Repository) + len(event.SessionID) + len(event.Content) + len(event.Summary))
	
	// Add estimated size for tags and data
	for _, tag := range event.Tags {
		size += int64(len(tag))
	}
	
	// Add rough estimation for JSON data (this could be more precise)
	size += 100 // Estimated overhead for JSON structure and timestamp
	
	return size
}

// estimateJSONSize estimates the size of a JSON message
func (ec *EnhancedClient) estimateJSONSize(msg map[string]interface{}) int64 {
	// Very rough estimation - in production, you might want to marshal and measure
	size := int64(0)
	for key, value := range msg {
		size += int64(len(key))
		if str, ok := value.(string); ok {
			size += int64(len(str))
		} else {
			size += 20 // Rough estimate for other types
		}
	}
	return size
}