// Package websocket provides heartbeat monitoring and health checks for WebSocket connections
package websocket

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// HeartbeatManager manages heartbeat monitoring for WebSocket connections
type HeartbeatManager struct {
	clients      map[string]*ClientHealth
	pingInterval time.Duration
	pongTimeout  time.Duration
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	metrics      *HeartbeatMetrics
}

// ClientHealth tracks health status of a WebSocket client
type ClientHealth struct {
	Client          *Client
	LastPingSent    time.Time
	LastPongReceived time.Time
	ConsecutiveFails int
	IsHealthy       bool
	TotalPings      int64
	TotalPongs      int64
	AverageLatency  time.Duration
	LastLatency     time.Duration
}

// HeartbeatMetrics tracks heartbeat system performance
type HeartbeatMetrics struct {
	TotalPingsSent     int64         `json:"total_pings_sent"`
	TotalPongsReceived int64         `json:"total_pongs_received"`
	TimeoutCount       int64         `json:"timeout_count"`
	HealthyClients     int           `json:"healthy_clients"`
	UnhealthyClients   int           `json:"unhealthy_clients"`
	AverageLatency     time.Duration `json:"average_latency"`
	MaxLatency         time.Duration `json:"max_latency"`
	MinLatency         time.Duration `json:"min_latency"`
	LastCheck          time.Time     `json:"last_check"`
	mu                 sync.RWMutex
}

// NewHeartbeatManager creates a new heartbeat manager
func NewHeartbeatManager(pingInterval, pongTimeout time.Duration) *HeartbeatManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &HeartbeatManager{
		clients:      make(map[string]*ClientHealth),
		pingInterval: pingInterval,
		pongTimeout:  pongTimeout,
		ctx:          ctx,
		cancel:       cancel,
		metrics: &HeartbeatMetrics{
			MinLatency: time.Hour, // Initialize with high value
			LastCheck:  time.Now(),
		},
	}
}

// Start begins the heartbeat monitoring process
func (hm *HeartbeatManager) Start(ctx context.Context) {
	log.Printf("Starting heartbeat manager with interval: %v, timeout: %v", 
		hm.pingInterval, hm.pongTimeout)

	ticker := time.NewTicker(hm.pingInterval)
	defer ticker.Stop()

	cleanupTicker := time.NewTicker(30 * time.Second) // Cleanup every 30 seconds
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ticker.C:
			hm.sendPings()
		case <-cleanupTicker.C:
			hm.cleanupUnhealthyClients()
		case <-ctx.Done():
			log.Println("Heartbeat manager shutting down")
			return
		case <-hm.ctx.Done():
			log.Println("Heartbeat manager context cancelled")
			return
		}
	}
}

// Stop stops the heartbeat manager
func (hm *HeartbeatManager) Stop() {
	hm.cancel()
}

// AddClient adds a client to heartbeat monitoring
func (hm *HeartbeatManager) AddClient(client *Client) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	health := &ClientHealth{
		Client:           client,
		LastPingSent:     time.Now(),
		LastPongReceived: time.Now(),
		IsHealthy:        true,
		ConsecutiveFails: 0,
	}

	hm.clients[client.ID] = health

	// Setup pong handler for this client
	client.Connection.SetPongHandler(func(appData string) error {
		hm.handlePong(client.ID)
		return nil
	})

	log.Printf("Client %s added to heartbeat monitoring", client.ID)
}

// RemoveClient removes a client from heartbeat monitoring
func (hm *HeartbeatManager) RemoveClient(clientID string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	delete(hm.clients, clientID)
	log.Printf("Client %s removed from heartbeat monitoring", clientID)
}

// sendPings sends ping messages to all monitored clients
func (hm *HeartbeatManager) sendPings() {
	hm.mu.RLock()
	clients := make(map[string]*ClientHealth)
	for id, health := range hm.clients {
		clients[id] = health
	}
	hm.mu.RUnlock()

	for clientID, health := range clients {
		if err := hm.sendPing(health); err != nil {
			log.Printf("Failed to send ping to client %s: %v", clientID, err)
			hm.markClientUnhealthy(clientID, err)
		}
	}

	hm.updateMetrics()
}

// sendPing sends a ping message to a specific client
func (hm *HeartbeatManager) sendPing(health *ClientHealth) error {
	if health.Client.Connection == nil {
		return websocket.ErrCloseSent
	}

	// Set write deadline
	if err := health.Client.Connection.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return err
	}

	// Send ping
	if err := health.Client.Connection.WriteMessage(websocket.PingMessage, nil); err != nil {
		return err
	}

	// Update health tracking
	hm.mu.Lock()
	health.LastPingSent = time.Now()
	health.TotalPings++
	hm.mu.Unlock()

	hm.metrics.mu.Lock()
	hm.metrics.TotalPingsSent++
	hm.metrics.mu.Unlock()

	return nil
}

// handlePong processes a pong response from a client
func (hm *HeartbeatManager) handlePong(clientID string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	health, exists := hm.clients[clientID]
	if !exists {
		return
	}

	now := time.Now()
	latency := now.Sub(health.LastPingSent)

	// Update health status
	health.LastPongReceived = now
	health.LastLatency = latency
	health.TotalPongs++
	health.ConsecutiveFails = 0

	// Update average latency
	if health.AverageLatency == 0 {
		health.AverageLatency = latency
	} else {
		// Exponential moving average
		health.AverageLatency = time.Duration(
			int64(health.AverageLatency)*9/10 + int64(latency)/10,
		)
	}

	// Mark as healthy if it wasn't
	if !health.IsHealthy {
		health.IsHealthy = true
		log.Printf("Client %s is now healthy (latency: %v)", clientID, latency)
	}

	// Update global metrics
	hm.metrics.mu.Lock()
	hm.metrics.TotalPongsReceived++
	
	// Update latency metrics
	if latency > hm.metrics.MaxLatency {
		hm.metrics.MaxLatency = latency
	}
	if latency < hm.metrics.MinLatency {
		hm.metrics.MinLatency = latency
	}
	
	// Update average latency
	if hm.metrics.AverageLatency == 0 {
		hm.metrics.AverageLatency = latency
	} else {
		hm.metrics.AverageLatency = time.Duration(
			int64(hm.metrics.AverageLatency)*9/10 + int64(latency)/10,
		)
	}
	
	hm.metrics.mu.Unlock()
}

// markClientUnhealthy marks a client as unhealthy
func (hm *HeartbeatManager) markClientUnhealthy(clientID string, err error) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	health, exists := hm.clients[clientID]
	if !exists {
		return
	}

	health.ConsecutiveFails++
	
	// Check if client should be marked unhealthy
	if health.ConsecutiveFails >= 3 || websocket.IsCloseError(err) {
		if health.IsHealthy {
			health.IsHealthy = false
			log.Printf("Client %s marked as unhealthy (consecutive fails: %d, error: %v)", 
				clientID, health.ConsecutiveFails, err)
		}
	}
}

// cleanupUnhealthyClients removes clients that have timed out
func (hm *HeartbeatManager) cleanupUnhealthyClients() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	now := time.Now()
	toRemove := []string{}

	for clientID, health := range hm.clients {
		// Check if client has timed out
		if now.Sub(health.LastPongReceived) > hm.pongTimeout {
			toRemove = append(toRemove, clientID)
			hm.metrics.mu.Lock()
			hm.metrics.TimeoutCount++
			hm.metrics.mu.Unlock()
		}
	}

	// Remove timed-out clients
	for _, clientID := range toRemove {
		health := hm.clients[clientID]
		delete(hm.clients, clientID)

		// Close the connection if it's still open
		if health.Client.Connection != nil {
			if err := health.Client.Connection.Close(); err != nil {
				log.Printf("Error closing timed-out connection %s: %v", clientID, err)
			}
		}

		log.Printf("Client %s removed due to heartbeat timeout", clientID)
	}
}

// updateMetrics updates the heartbeat metrics
func (hm *HeartbeatManager) updateMetrics() {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	hm.metrics.mu.Lock()
	defer hm.metrics.mu.Unlock()

	healthyCount := 0
	unhealthyCount := 0

	for _, health := range hm.clients {
		if health.IsHealthy {
			healthyCount++
		} else {
			unhealthyCount++
		}
	}

	hm.metrics.HealthyClients = healthyCount
	hm.metrics.UnhealthyClients = unhealthyCount
	hm.metrics.LastCheck = time.Now()
}

// GetMetrics returns heartbeat metrics
func (hm *HeartbeatManager) GetMetrics() *HeartbeatMetrics {
	hm.metrics.mu.RLock()
	defer hm.metrics.mu.RUnlock()

	// Return a copy to avoid race conditions
	return &HeartbeatMetrics{
		TotalPingsSent:     hm.metrics.TotalPingsSent,
		TotalPongsReceived: hm.metrics.TotalPongsReceived,
		TimeoutCount:       hm.metrics.TimeoutCount,
		HealthyClients:     hm.metrics.HealthyClients,
		UnhealthyClients:   hm.metrics.UnhealthyClients,
		AverageLatency:     hm.metrics.AverageLatency,
		MaxLatency:         hm.metrics.MaxLatency,
		MinLatency:         hm.metrics.MinLatency,
		LastCheck:          hm.metrics.LastCheck,
	}
}

// GetClientHealth returns health status for a specific client
func (hm *HeartbeatManager) GetClientHealth(clientID string) (*ClientHealth, bool) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	health, exists := hm.clients[clientID]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid race conditions
	return &ClientHealth{
		Client:           health.Client,
		LastPingSent:     health.LastPingSent,
		LastPongReceived: health.LastPongReceived,
		ConsecutiveFails: health.ConsecutiveFails,
		IsHealthy:        health.IsHealthy,
		TotalPings:       health.TotalPings,
		TotalPongs:       health.TotalPongs,
		AverageLatency:   health.AverageLatency,
		LastLatency:      health.LastLatency,
	}, true
}

// GetAllClientHealth returns health status for all clients
func (hm *HeartbeatManager) GetAllClientHealth() map[string]*ClientHealth {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	result := make(map[string]*ClientHealth)
	for id, health := range hm.clients {
		result[id] = &ClientHealth{
			Client:           health.Client,
			LastPingSent:     health.LastPingSent,
			LastPongReceived: health.LastPongReceived,
			ConsecutiveFails: health.ConsecutiveFails,
			IsHealthy:        health.IsHealthy,
			TotalPings:       health.TotalPings,
			TotalPongs:       health.TotalPongs,
			AverageLatency:   health.AverageLatency,
			LastLatency:      health.LastLatency,
		}
	}

	return result
}

// IsClientHealthy checks if a client is healthy
func (hm *HeartbeatManager) IsClientHealthy(clientID string) bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	health, exists := hm.clients[clientID]
	return exists && health.IsHealthy
}

// GetHealthyClientCount returns the number of healthy clients
func (hm *HeartbeatManager) GetHealthyClientCount() int {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	count := 0
	for _, health := range hm.clients {
		if health.IsHealthy {
			count++
		}
	}
	return count
}