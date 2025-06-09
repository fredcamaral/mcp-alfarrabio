// Package websocket provides connection pooling and lifecycle management
package websocket

import (
	"log"
	"sync"
	"time"
)

// ConnectionPool manages WebSocket connections with limits and lifecycle
type ConnectionPool struct {
	connections map[string]*Client
	maxSize     int
	mu          sync.RWMutex
	metrics     *PoolMetrics
}

// PoolMetrics tracks connection pool performance
type PoolMetrics struct {
	TotalConnections    int64         `json:"total_connections"`
	ActiveConnections   int           `json:"active_connections"`
	MaxConnections      int           `json:"max_connections"`
	ConnectionsAccepted int64         `json:"connections_accepted"`
	ConnectionsRejected int64         `json:"connections_rejected"`
	ConnectionsClosed   int64         `json:"connections_closed"`
	AverageLifetime     time.Duration `json:"average_lifetime"`
	MemoryUsage         int64         `json:"memory_usage_bytes"`
	LastCleanup         time.Time     `json:"last_cleanup"`
	mu                  sync.RWMutex
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(maxSize int) *ConnectionPool {
	return &ConnectionPool{
		connections: make(map[string]*Client),
		maxSize:     maxSize,
		metrics: &PoolMetrics{
			MaxConnections: maxSize,
			LastCleanup:    time.Now(),
		},
	}
}

// AddConnection adds a new connection to the pool
func (p *ConnectionPool) AddConnection(client *Client) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if pool is full
	if len(p.connections) >= p.maxSize {
		p.metrics.ConnectionsRejected++
		return false
	}

	// Add connection
	p.connections[client.ID] = client
	p.metrics.ActiveConnections = len(p.connections)
	p.metrics.ConnectionsAccepted++
	p.metrics.TotalConnections++

	log.Printf("Connection added to pool: %s (total: %d/%d)",
		client.ID, len(p.connections), p.maxSize)

	return true
}

// RemoveConnection removes a connection from the pool
func (p *ConnectionPool) RemoveConnection(clientID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	client, exists := p.connections[clientID]
	if !exists {
		return false
	}

	// Update lifetime metrics
	if client.Metadata != nil {
		lifetime := time.Since(client.Metadata.ConnectedAt)
		p.updateAverageLifetime(lifetime)
	}

	// Remove connection
	delete(p.connections, clientID)
	p.metrics.ActiveConnections = len(p.connections)
	p.metrics.ConnectionsClosed++

	log.Printf("Connection removed from pool: %s (total: %d/%d)",
		clientID, len(p.connections), p.maxSize)

	return true
}

// GetConnection retrieves a connection by ID
func (p *ConnectionPool) GetConnection(clientID string) (*Client, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	client, exists := p.connections[clientID]
	return client, exists
}

// GetAllConnections returns all active connections
func (p *ConnectionPool) GetAllConnections() map[string]*Client {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Return a copy to avoid race conditions
	connections := make(map[string]*Client)
	for id, client := range p.connections {
		connections[id] = client
	}

	return connections
}

// GetConnectionCount returns the current number of connections
func (p *ConnectionPool) GetConnectionCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.connections)
}

// CanAcceptConnection checks if the pool can accept new connections
func (p *ConnectionPool) CanAcceptConnection() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.connections) < p.maxSize
}

// GetAvailableCapacity returns the remaining capacity
func (p *ConnectionPool) GetAvailableCapacity() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.maxSize - len(p.connections)
}

// CloseAll closes all connections gracefully
func (p *ConnectionPool) CloseAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	log.Printf("Closing all connections in pool (%d connections)", len(p.connections))

	for id, client := range p.connections {
		// Close the connection gracefully
		if client.Connection != nil {
			if err := client.Connection.Close(); err != nil {
				log.Printf("Error closing connection %s: %v", id, err)
			}
		}

		// Close the send channel
		if client.Send != nil {
			close(client.Send)
		}

		// Update metrics
		if client.Metadata != nil {
			lifetime := time.Since(client.Metadata.ConnectedAt)
			p.updateAverageLifetime(lifetime)
		}

		p.metrics.ConnectionsClosed++
	}

	// Clear the connections map
	p.connections = make(map[string]*Client)
	p.metrics.ActiveConnections = 0

	log.Println("All connections closed")
}

// CleanupStaleConnections removes connections that haven't been active
func (p *ConnectionPool) CleanupStaleConnections(maxIdleTime time.Duration) int {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	staleConnections := []string{}

	// Find stale connections
	for id, client := range p.connections {
		if client.Metadata != nil {
			idleTime := now.Sub(client.Metadata.LastActivity)
			if idleTime > maxIdleTime {
				staleConnections = append(staleConnections, id)
			}
		}
	}

	// Remove stale connections
	for _, id := range staleConnections {
		client := p.connections[id]

		// Close connection
		if client.Connection != nil {
			if err := client.Connection.Close(); err != nil {
				log.Printf("Error closing stale connection %s: %v", id, err)
			}
		}

		// Update metrics
		if client.Metadata != nil {
			lifetime := time.Since(client.Metadata.ConnectedAt)
			p.updateAverageLifetime(lifetime)
		}

		delete(p.connections, id)
		p.metrics.ConnectionsClosed++
	}

	p.metrics.ActiveConnections = len(p.connections)
	p.metrics.LastCleanup = now

	if len(staleConnections) > 0 {
		log.Printf("Cleaned up %d stale connections", len(staleConnections))
	}

	return len(staleConnections)
}

// GetConnectionsByRepository returns connections filtered by repository
func (p *ConnectionPool) GetConnectionsByRepository(repository string) []*Client {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var filtered []*Client
	for _, client := range p.connections {
		if client.Repository == repository {
			filtered = append(filtered, client)
		}
	}

	return filtered
}

// GetConnectionsBySession returns connections filtered by session ID
func (p *ConnectionPool) GetConnectionsBySession(sessionID string) []*Client {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var filtered []*Client
	for _, client := range p.connections {
		if client.SessionID == sessionID {
			filtered = append(filtered, client)
		}
	}

	return filtered
}

// GetMetrics returns pool metrics
func (p *ConnectionPool) GetMetrics() *PoolMetrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()

	// Return a copy to avoid race conditions
	return &PoolMetrics{
		TotalConnections:    p.metrics.TotalConnections,
		ActiveConnections:   p.metrics.ActiveConnections,
		MaxConnections:      p.metrics.MaxConnections,
		ConnectionsAccepted: p.metrics.ConnectionsAccepted,
		ConnectionsRejected: p.metrics.ConnectionsRejected,
		ConnectionsClosed:   p.metrics.ConnectionsClosed,
		AverageLifetime:     p.metrics.AverageLifetime,
		MemoryUsage:         p.metrics.MemoryUsage,
		LastCleanup:         p.metrics.LastCleanup,
	}
}

// UpdateConnectionActivity updates the last activity time for a connection
func (p *ConnectionPool) UpdateConnectionActivity(clientID string) {
	p.mu.RLock()
	client, exists := p.connections[clientID]
	p.mu.RUnlock()

	if exists && client.Metadata != nil {
		client.Metadata.LastActivity = time.Now()
	}
}

// UpdateConnectionMetrics updates the connection metrics
func (p *ConnectionPool) UpdateConnectionMetrics(clientID string, bytesSent, bytesReceived int64, messagesSent, messagesReceived int64) {
	p.mu.RLock()
	client, exists := p.connections[clientID]
	p.mu.RUnlock()

	if exists && client.Metadata != nil {
		client.Metadata.BytesSent += bytesSent
		client.Metadata.BytesReceived += bytesReceived
		client.Metadata.MessagesSent += messagesSent
		client.Metadata.MessagesReceived += messagesReceived
		client.Metadata.LastActivity = time.Now()
	}
}

// updateAverageLifetime updates the average connection lifetime
func (p *ConnectionPool) updateAverageLifetime(lifetime time.Duration) {
	// Simple moving average calculation
	if p.metrics.AverageLifetime == 0 {
		p.metrics.AverageLifetime = lifetime
	} else {
		// Weighted average with 90% weight on previous average
		p.metrics.AverageLifetime = time.Duration(
			int64(p.metrics.AverageLifetime)*9/10 + int64(lifetime)/10,
		)
	}
}

// GetConnectionStats returns detailed connection statistics
func (p *ConnectionPool) GetConnectionStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := map[string]interface{}{
		"total_connections":   len(p.connections),
		"max_connections":     p.maxSize,
		"utilization_percent": float64(len(p.connections)) / float64(p.maxSize) * 100,
		"available_slots":     p.maxSize - len(p.connections),
	}

	// Connection distribution by repository
	repoStats := make(map[string]int)
	sessionStats := make(map[string]int)
	versionStats := make(map[string]int)

	for _, client := range p.connections {
		if client.Repository != "" {
			repoStats[client.Repository]++
		}
		if client.SessionID != "" {
			sessionStats[client.SessionID]++
		}
		if client.Metadata != nil && client.Metadata.CLIVersion != "" {
			versionStats[client.Metadata.CLIVersion]++
		}
	}

	stats["by_repository"] = repoStats
	stats["by_session"] = sessionStats
	stats["by_cli_version"] = versionStats

	return stats
}
