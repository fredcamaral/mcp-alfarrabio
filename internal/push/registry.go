// Package push provides push notification system for CLI endpoints
package push

import (
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"
)

// CLIEndpoint represents a registered CLI endpoint for push notifications
type CLIEndpoint struct {
	ID           string                   `json:"id"`
	URL          string                   `json:"url"`
	Version      string                   `json:"version"`
	Capabilities []string                 `json:"capabilities"`
	Metadata     map[string]string        `json:"metadata"`
	RegisteredAt time.Time                `json:"registered_at"`
	LastSeen     time.Time                `json:"last_seen"`
	Status       EndpointStatus           `json:"status"`
	Health       *EndpointHealth          `json:"health"`
	Preferences  *NotificationPreferences `json:"preferences"`
}

// EndpointStatus represents the status of a CLI endpoint
type EndpointStatus string

const (
	StatusActive      EndpointStatus = "active"
	StatusInactive    EndpointStatus = "inactive"
	StatusUnreachable EndpointStatus = "unreachable"
	StatusError       EndpointStatus = "error"
)

// EndpointHealth tracks health metrics for a CLI endpoint
type EndpointHealth struct {
	IsHealthy           bool          `json:"is_healthy"`
	LastHealthCheck     time.Time     `json:"last_health_check"`
	ConsecutiveFailures int           `json:"consecutive_failures"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	LastError           string        `json:"last_error"`
	SuccessRate         float64       `json:"success_rate"`
	TotalRequests       int64         `json:"total_requests"`
	SuccessfulRequests  int64         `json:"successful_requests"`
}

// NotificationPreferences defines CLI notification preferences
type NotificationPreferences struct {
	EnabledEvents   []string          `json:"enabled_events"`
	DisabledEvents  []string          `json:"disabled_events"`
	MaxRetries      int               `json:"max_retries"`
	RetryDelay      time.Duration     `json:"retry_delay"`
	DeliveryTimeout time.Duration     `json:"delivery_timeout"`
	Filters         map[string]string `json:"filters"`
	BatchSize       int               `json:"batch_size"`
	BatchTimeout    time.Duration     `json:"batch_timeout"`
}

// DefaultNotificationPreferences returns default notification preferences
func DefaultNotificationPreferences() *NotificationPreferences {
	return &NotificationPreferences{
		EnabledEvents:   []string{"memory_update", "task_update", "system_alert"},
		DisabledEvents:  []string{},
		MaxRetries:      3,
		RetryDelay:      time.Second,
		DeliveryTimeout: 5 * time.Second,
		Filters:         make(map[string]string),
		BatchSize:       1,
		BatchTimeout:    100 * time.Millisecond,
	}
}

// Registry manages CLI endpoint registration and discovery
type Registry struct {
	endpoints      map[string]*CLIEndpoint
	mu             sync.RWMutex
	metrics        *RegistryMetrics
	cleanupTicker  *time.Ticker
	maxInactiveAge time.Duration
}

// RegistryMetrics tracks registry performance
type RegistryMetrics struct {
	TotalEndpoints      int           `json:"total_endpoints"`
	ActiveEndpoints     int           `json:"active_endpoints"`
	InactiveEndpoints   int           `json:"inactive_endpoints"`
	RegistrationCount   int64         `json:"registration_count"`
	DeregistrationCount int64         `json:"deregistration_count"`
	CleanupCount        int64         `json:"cleanup_count"`
	LastCleanup         time.Time     `json:"last_cleanup"`
	AverageUptime       time.Duration `json:"average_uptime"`
	mu                  sync.RWMutex
}

// NewRegistry creates a new CLI endpoint registry
func NewRegistry() *Registry {
	registry := &Registry{
		endpoints:      make(map[string]*CLIEndpoint),
		metrics:        &RegistryMetrics{LastCleanup: time.Now()},
		maxInactiveAge: 10 * time.Minute, // Remove endpoints inactive for 10 minutes
	}

	// Start cleanup routine
	registry.cleanupTicker = time.NewTicker(time.Minute)
	go registry.cleanupRoutine()

	return registry
}

// Register registers a new CLI endpoint
func (r *Registry) Register(endpoint *CLIEndpoint) error {
	if endpoint.ID == "" {
		return fmt.Errorf("endpoint ID is required")
	}

	if endpoint.URL == "" {
		return fmt.Errorf("endpoint URL is required")
	}

	// Validate URL
	if _, err := url.Parse(endpoint.URL); err != nil {
		return fmt.Errorf("invalid endpoint URL: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	// Set defaults if not provided
	if endpoint.Preferences == nil {
		endpoint.Preferences = DefaultNotificationPreferences()
	}

	if endpoint.Health == nil {
		endpoint.Health = &EndpointHealth{
			IsHealthy:       true,
			LastHealthCheck: now,
			SuccessRate:     1.0,
		}
	}

	if endpoint.Metadata == nil {
		endpoint.Metadata = make(map[string]string)
	}

	endpoint.RegisteredAt = now
	endpoint.LastSeen = now
	endpoint.Status = StatusActive

	// Check if endpoint already exists
	if existing, exists := r.endpoints[endpoint.ID]; exists {
		// Update existing endpoint
		existing.URL = endpoint.URL
		existing.Version = endpoint.Version
		existing.Capabilities = endpoint.Capabilities
		existing.Metadata = endpoint.Metadata
		existing.LastSeen = now
		existing.Status = StatusActive
		existing.Preferences = endpoint.Preferences

		log.Printf("Updated CLI endpoint registration: %s at %s", endpoint.ID, endpoint.URL)
	} else {
		// Register new endpoint
		r.endpoints[endpoint.ID] = endpoint
		r.metrics.RegistrationCount++
		log.Printf("Registered new CLI endpoint: %s at %s", endpoint.ID, endpoint.URL)
	}

	r.updateMetrics()
	return nil
}

// Deregister removes a CLI endpoint from the registry
func (r *Registry) Deregister(endpointID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	endpoint, exists := r.endpoints[endpointID]
	if !exists {
		return fmt.Errorf("endpoint not found: %s", endpointID)
	}

	delete(r.endpoints, endpointID)
	r.metrics.DeregistrationCount++

	log.Printf("Deregistered CLI endpoint: %s (was at %s)", endpointID, endpoint.URL)

	r.updateMetrics()
	return nil
}

// Get retrieves a CLI endpoint by ID
func (r *Registry) Get(endpointID string) (*CLIEndpoint, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	endpoint, exists := r.endpoints[endpointID]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid race conditions
	return r.copyEndpoint(endpoint), true
}

// GetAll returns all registered CLI endpoints
func (r *Registry) GetAll() []*CLIEndpoint {
	r.mu.RLock()
	defer r.mu.RUnlock()

	endpoints := make([]*CLIEndpoint, 0, len(r.endpoints))
	for _, endpoint := range r.endpoints {
		endpoints = append(endpoints, r.copyEndpoint(endpoint))
	}

	return endpoints
}

// GetActive returns all active CLI endpoints
func (r *Registry) GetActive() []*CLIEndpoint {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var active []*CLIEndpoint
	for _, endpoint := range r.endpoints {
		if endpoint.Status == StatusActive && endpoint.Health.IsHealthy {
			active = append(active, r.copyEndpoint(endpoint))
		}
	}

	return active
}

// GetByFilter returns endpoints matching filter criteria
func (r *Registry) GetByFilter(filters map[string]string) []*CLIEndpoint {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var filtered []*CLIEndpoint
	for _, endpoint := range r.endpoints {
		if r.matchesFilters(endpoint, filters) {
			filtered = append(filtered, r.copyEndpoint(endpoint))
		}
	}

	return filtered
}

// GetByRepository returns all endpoints subscribed to a specific repository
func (r *Registry) GetByRepository(repository string) []*CLIEndpoint {
	return r.GetByFilter(map[string]string{"repository": repository})
}

// GetBySession returns all endpoints subscribed to a specific session
func (r *Registry) GetBySession(sessionID string) []*CLIEndpoint {
	return r.GetByFilter(map[string]string{"session_id": sessionID})
}

// GetByCapability returns all endpoints with a specific capability
func (r *Registry) GetByCapability(capability string) []*CLIEndpoint {
	return r.GetByFilter(map[string]string{"capability": capability})
}

// GetByVersion returns all endpoints with a specific CLI version
func (r *Registry) GetByVersion(version string) []*CLIEndpoint {
	return r.GetByFilter(map[string]string{"version": version})
}

// UpdateHealth updates the health status of an endpoint
func (r *Registry) UpdateHealth(endpointID string, health *EndpointHealth) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	endpoint, exists := r.endpoints[endpointID]
	if !exists {
		return fmt.Errorf("endpoint not found: %s", endpointID)
	}

	endpoint.Health = health
	endpoint.LastSeen = time.Now()

	// Update status based on health
	if health.IsHealthy {
		endpoint.Status = StatusActive
	} else {
		if health.ConsecutiveFailures > 5 {
			endpoint.Status = StatusUnreachable
		} else {
			endpoint.Status = StatusInactive
		}
	}

	return nil
}

// UpdateLastSeen updates the last seen timestamp for an endpoint
func (r *Registry) UpdateLastSeen(endpointID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	endpoint, exists := r.endpoints[endpointID]
	if !exists {
		return fmt.Errorf("endpoint not found: %s", endpointID)
	}

	endpoint.LastSeen = time.Now()
	return nil
}

// GetMetrics returns registry metrics
func (r *Registry) GetMetrics() *RegistryMetrics {
	r.metrics.mu.RLock()
	defer r.metrics.mu.RUnlock()

	// Return a copy
	return &RegistryMetrics{
		TotalEndpoints:      r.metrics.TotalEndpoints,
		ActiveEndpoints:     r.metrics.ActiveEndpoints,
		InactiveEndpoints:   r.metrics.InactiveEndpoints,
		RegistrationCount:   r.metrics.RegistrationCount,
		DeregistrationCount: r.metrics.DeregistrationCount,
		CleanupCount:        r.metrics.CleanupCount,
		LastCleanup:         r.metrics.LastCleanup,
		AverageUptime:       r.metrics.AverageUptime,
	}
}

// Stop stops the registry cleanup routine
func (r *Registry) Stop() {
	if r.cleanupTicker != nil {
		r.cleanupTicker.Stop()
	}
}

// cleanupRoutine removes inactive endpoints periodically
func (r *Registry) cleanupRoutine() {
	for range r.cleanupTicker.C {
		r.cleanup()
	}
}

// cleanup removes endpoints that have been inactive for too long
func (r *Registry) cleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	toRemove := []string{}

	for id, endpoint := range r.endpoints {
		// Remove endpoints inactive for more than maxInactiveAge
		if now.Sub(endpoint.LastSeen) > r.maxInactiveAge {
			toRemove = append(toRemove, id)
		}
	}

	for _, id := range toRemove {
		endpoint := r.endpoints[id]
		delete(r.endpoints, id)
		r.metrics.CleanupCount++
		log.Printf("Cleaned up inactive CLI endpoint: %s (last seen %v ago)",
			id, now.Sub(endpoint.LastSeen))
	}

	if len(toRemove) > 0 {
		r.updateMetrics()
	}

	r.metrics.mu.Lock()
	r.metrics.LastCleanup = now
	r.metrics.mu.Unlock()
}

// updateMetrics updates internal metrics
func (r *Registry) updateMetrics() {
	r.metrics.mu.Lock()
	defer r.metrics.mu.Unlock()

	activeCount := 0
	inactiveCount := 0
	totalUptime := time.Duration(0)
	now := time.Now()

	for _, endpoint := range r.endpoints {
		if endpoint.Status == StatusActive && endpoint.Health.IsHealthy {
			activeCount++
		} else {
			inactiveCount++
		}

		uptime := now.Sub(endpoint.RegisteredAt)
		totalUptime += uptime
	}

	r.metrics.TotalEndpoints = len(r.endpoints)
	r.metrics.ActiveEndpoints = activeCount
	r.metrics.InactiveEndpoints = inactiveCount

	if len(r.endpoints) > 0 {
		r.metrics.AverageUptime = totalUptime / time.Duration(len(r.endpoints))
	}
}

// copyEndpoint creates a copy of an endpoint to avoid race conditions
func (r *Registry) copyEndpoint(endpoint *CLIEndpoint) *CLIEndpoint {
	metadata := make(map[string]string)
	for k, v := range endpoint.Metadata {
		metadata[k] = v
	}

	capabilities := make([]string, len(endpoint.Capabilities))
	copy(capabilities, endpoint.Capabilities)

	health := *endpoint.Health
	preferences := *endpoint.Preferences

	// Copy filters
	filters := make(map[string]string)
	for k, v := range endpoint.Preferences.Filters {
		filters[k] = v
	}
	preferences.Filters = filters

	// Copy enabled/disabled events
	enabledEvents := make([]string, len(endpoint.Preferences.EnabledEvents))
	copy(enabledEvents, endpoint.Preferences.EnabledEvents)
	preferences.EnabledEvents = enabledEvents

	disabledEvents := make([]string, len(endpoint.Preferences.DisabledEvents))
	copy(disabledEvents, endpoint.Preferences.DisabledEvents)
	preferences.DisabledEvents = disabledEvents

	return &CLIEndpoint{
		ID:           endpoint.ID,
		URL:          endpoint.URL,
		Version:      endpoint.Version,
		Capabilities: capabilities,
		Metadata:     metadata,
		RegisteredAt: endpoint.RegisteredAt,
		LastSeen:     endpoint.LastSeen,
		Status:       endpoint.Status,
		Health:       &health,
		Preferences:  &preferences,
	}
}

// matchesFilters checks if an endpoint matches the given filters
func (r *Registry) matchesFilters(endpoint *CLIEndpoint, filters map[string]string) bool {
	for key, value := range filters {
		switch key {
		case "status":
			if string(endpoint.Status) != value {
				return false
			}
		case "version":
			if endpoint.Version != value {
				return false
			}
		case "capability":
			hasCapability := false
			for _, cap := range endpoint.Capabilities {
				if cap == value {
					hasCapability = true
					break
				}
			}
			if !hasCapability {
				return false
			}
		default:
			// Check metadata
			if metaValue, exists := endpoint.Metadata[key]; !exists || metaValue != value {
				return false
			}
		}
	}
	return true
}
