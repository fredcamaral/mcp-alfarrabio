// Package handlers provides HTTP handlers for CLI endpoint registration
package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"lerian-mcp-memory/internal/push"
)

// HTTP method constants
const (
	HTTPMethodPOST   = "POST"
	HTTPMethodGET    = "GET"
	HTTPMethodPUT    = "PUT"
	HTTPMethodPATCH  = "PATCH"
	HTTPMethodDELETE = "DELETE"
)

// String constants
const (
	StringTrue = "true"
)

// CLIRegistryHandler handles CLI endpoint registration and management
type CLIRegistryHandler struct {
	notificationService *push.NotificationService
}

// NewCLIRegistryHandler creates a new CLI registry handler
func NewCLIRegistryHandler(notificationService *push.NotificationService) *CLIRegistryHandler {
	return &CLIRegistryHandler{
		notificationService: notificationService,
	}
}

// RegisterEndpoint handles CLI endpoint registration
func (crh *CLIRegistryHandler) RegisterEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTPMethodPOST {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse JSON payload
	var registrationReq CLIRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&registrationReq); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON payload: %v", err), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if err := crh.validateRegistrationRequest(&registrationReq); err != nil {
		http.Error(w, fmt.Sprintf("Validation error: %v", err), http.StatusBadRequest)
		return
	}

	// Create CLI endpoint
	endpoint := &push.CLIEndpoint{
		ID:           registrationReq.ID,
		URL:          registrationReq.URL,
		Version:      registrationReq.Version,
		Capabilities: registrationReq.Capabilities,
		Metadata:     registrationReq.Metadata,
		Preferences:  registrationReq.Preferences,
	}

	// Set defaults if not provided
	if endpoint.Metadata == nil {
		endpoint.Metadata = make(map[string]string)
	}
	if endpoint.Preferences == nil {
		endpoint.Preferences = push.DefaultNotificationPreferences()
	}

	// Add request metadata
	endpoint.Metadata["remote_addr"] = r.RemoteAddr
	endpoint.Metadata["user_agent"] = r.UserAgent()
	if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
		endpoint.Metadata["request_id"] = requestID
	}

	// Register endpoint
	if err := crh.notificationService.RegisterEndpoint(endpoint); err != nil {
		log.Printf("Failed to register CLI endpoint %s: %v", endpoint.ID, err)
		http.Error(w, fmt.Sprintf("Registration failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	response := CLIRegistrationResponse{
		Success:      true,
		EndpointID:   endpoint.ID,
		RegisteredAt: time.Now(),
		Message:      "CLI endpoint registered successfully",
		Config: CLIConfig{
			PushEnabled:       true,
			HeartbeatInterval: 30 * time.Second,
			RetryPolicy:       endpoint.Preferences,
			SupportedEvents:   []string{"memory_update", "task_update", "system_alert"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("CLI endpoint registered: %s at %s", endpoint.ID, endpoint.URL)
}

// DeregisterEndpoint handles CLI endpoint deregistration
func (crh *CLIRegistryHandler) DeregisterEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTPMethodPOST && r.Method != HTTPMethodDELETE {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get endpoint ID from URL path or request body
	endpointID := r.URL.Query().Get("endpoint_id")
	if endpointID == "" {
		// Try to get from JSON body
		var req map[string]string
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			endpointID = req["endpoint_id"]
		}
	}

	if endpointID == "" {
		http.Error(w, "endpoint_id parameter or field required", http.StatusBadRequest)
		return
	}

	// Deregister endpoint
	if err := crh.notificationService.DeregisterEndpoint(endpointID); err != nil {
		log.Printf("Failed to deregister CLI endpoint %s: %v", endpointID, err)
		http.Error(w, fmt.Sprintf("Deregistration failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	response := map[string]interface{}{
		"success":         true,
		"endpoint_id":     endpointID,
		"deregistered_at": time.Now(),
		"message":         "CLI endpoint deregistered successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("CLI endpoint deregistered: %s", endpointID)
}

// ListEndpoints returns all registered CLI endpoints
func (crh *CLIRegistryHandler) ListEndpoints(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTPMethodGET {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	showInactive := r.URL.Query().Get("include_inactive") == StringTrue
	filterVersion := r.URL.Query().Get("version")
	filterCapability := r.URL.Query().Get("capability")

	// Get endpoints
	var endpoints []*push.CLIEndpoint
	if showInactive {
		endpoints = crh.notificationService.GetEndpoints()
	} else {
		endpoints = crh.notificationService.GetActiveEndpoints()
	}

	// Apply filters
	filtered := make([]*push.CLIEndpoint, 0)
	for _, endpoint := range endpoints {
		// Version filter
		if filterVersion != "" && endpoint.Version != filterVersion {
			continue
		}

		// Capability filter
		if filterCapability != "" {
			hasCapability := false
			for _, cap := range endpoint.Capabilities {
				if cap == filterCapability {
					hasCapability = true
					break
				}
			}
			if !hasCapability {
				continue
			}
		}

		filtered = append(filtered, endpoint)
	}

	// Prepare response
	response := CLIEndpointsListResponse{
		Endpoints: make([]CLIEndpointSummary, len(filtered)),
		Count:     len(filtered),
		Timestamp: time.Now(),
	}

	for i, endpoint := range filtered {
		response.Endpoints[i] = CLIEndpointSummary{
			ID:           endpoint.ID,
			URL:          endpoint.URL,
			Version:      endpoint.Version,
			Capabilities: endpoint.Capabilities,
			Status:       string(endpoint.Status),
			IsHealthy:    endpoint.Health.IsHealthy,
			RegisteredAt: endpoint.RegisteredAt,
			LastSeen:     endpoint.LastSeen,
			Metadata:     endpoint.Metadata,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// GetEndpoint returns details for a specific CLI endpoint
func (crh *CLIRegistryHandler) GetEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTPMethodGET {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	endpointID := r.URL.Query().Get("endpoint_id")
	if endpointID == "" {
		http.Error(w, "endpoint_id parameter required", http.StatusBadRequest)
		return
	}

	endpoint, exists := crh.notificationService.GetEndpoint(endpointID)
	if !exists {
		http.Error(w, "Endpoint not found", http.StatusNotFound)
		return
	}

	// Prepare detailed response
	response := CLIEndpointDetail{
		ID:           endpoint.ID,
		URL:          endpoint.URL,
		Version:      endpoint.Version,
		Capabilities: endpoint.Capabilities,
		Metadata:     endpoint.Metadata,
		RegisteredAt: endpoint.RegisteredAt,
		LastSeen:     endpoint.LastSeen,
		Status:       string(endpoint.Status),
		Health:       endpoint.Health,
		Preferences:  endpoint.Preferences,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// UpdateEndpointPreferences updates notification preferences for an endpoint
func (crh *CLIRegistryHandler) UpdateEndpointPreferences(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTPMethodPUT && r.Method != HTTPMethodPATCH {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	endpointID := r.URL.Query().Get("endpoint_id")
	if endpointID == "" {
		http.Error(w, "endpoint_id parameter required", http.StatusBadRequest)
		return
	}

	// Check if endpoint exists
	endpoint, exists := crh.notificationService.GetEndpoint(endpointID)
	if !exists {
		http.Error(w, "Endpoint not found", http.StatusNotFound)
		return
	}

	// Parse preferences update
	var prefsUpdate push.NotificationPreferences
	if err := json.NewDecoder(r.Body).Decode(&prefsUpdate); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON payload: %v", err), http.StatusBadRequest)
		return
	}

	// Update endpoint with new preferences
	updatedEndpoint := *endpoint
	updatedEndpoint.Preferences = &prefsUpdate

	// Re-register endpoint with updated preferences
	if err := crh.notificationService.RegisterEndpoint(&updatedEndpoint); err != nil {
		log.Printf("Failed to update preferences for endpoint %s: %v", endpointID, err)
		http.Error(w, fmt.Sprintf("Update failed: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":     true,
		"endpoint_id": endpointID,
		"updated_at":  time.Now(),
		"message":     "Endpoint preferences updated successfully",
		"preferences": prefsUpdate,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Updated preferences for CLI endpoint: %s", endpointID)
}

// CheckEndpointHealth performs a health check for a specific endpoint
func (crh *CLIRegistryHandler) CheckEndpointHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTPMethodPOST {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	endpointID := r.URL.Query().Get("endpoint_id")
	if endpointID == "" {
		http.Error(w, "endpoint_id parameter required", http.StatusBadRequest)
		return
	}

	// Perform health check
	result, err := crh.notificationService.CheckEndpointHealth(endpointID)
	if err != nil {
		log.Printf("Health check failed for endpoint %s: %v", endpointID, err)
		http.Error(w, fmt.Sprintf("Health check failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// GetServiceStatus returns push notification service status
func (crh *CLIRegistryHandler) GetServiceStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTPMethodGET {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := crh.notificationService.GetServiceStatus()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// SendTestNotification sends a test notification to verify endpoint connectivity
func (crh *CLIRegistryHandler) SendTestNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTPMethodPOST {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	endpointID := r.URL.Query().Get("endpoint_id")
	if endpointID == "" {
		http.Error(w, "endpoint_id parameter required", http.StatusBadRequest)
		return
	}

	// Create test notification
	notification := push.CreateNotification("test_notification", map[string]interface{}{
		"message":   "This is a test notification from MCP Memory Server",
		"timestamp": time.Now(),
		"test_id":   fmt.Sprintf("test_%d", time.Now().UnixNano()),
	})

	// Send to specific endpoint
	if err := crh.notificationService.SendNotificationToEndpoint(notification, endpointID); err != nil {
		log.Printf("Failed to send test notification to endpoint %s: %v", endpointID, err)
		http.Error(w, fmt.Sprintf("Failed to send notification: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":         true,
		"endpoint_id":     endpointID,
		"notification_id": notification.ID,
		"sent_at":         time.Now(),
		"message":         "Test notification sent successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Sent test notification to CLI endpoint: %s", endpointID)
}

// validateRegistrationRequest validates CLI registration request
func (crh *CLIRegistryHandler) validateRegistrationRequest(req *CLIRegistrationRequest) error {
	if req.ID == "" {
		return fmt.Errorf("endpoint ID is required")
	}
	if req.URL == "" {
		return fmt.Errorf("endpoint URL is required")
	}
	if req.Version == "" {
		return fmt.Errorf("endpoint version is required")
	}
	return nil
}

// Request/Response types

// CLIRegistrationRequest represents a CLI endpoint registration request
type CLIRegistrationRequest struct {
	ID           string                        `json:"id"`
	URL          string                        `json:"url"`
	Version      string                        `json:"version"`
	Capabilities []string                      `json:"capabilities"`
	Metadata     map[string]string             `json:"metadata"`
	Preferences  *push.NotificationPreferences `json:"preferences"`
}

// CLIRegistrationResponse represents a successful registration response
type CLIRegistrationResponse struct {
	Success      bool      `json:"success"`
	EndpointID   string    `json:"endpoint_id"`
	RegisteredAt time.Time `json:"registered_at"`
	Message      string    `json:"message"`
	Config       CLIConfig `json:"config"`
}

// CLIConfig provides configuration for the registered CLI endpoint
type CLIConfig struct {
	PushEnabled       bool                          `json:"push_enabled"`
	HeartbeatInterval time.Duration                 `json:"heartbeat_interval"`
	RetryPolicy       *push.NotificationPreferences `json:"retry_policy"`
	SupportedEvents   []string                      `json:"supported_events"`
}

// CLIEndpointsListResponse represents a list of CLI endpoints
type CLIEndpointsListResponse struct {
	Endpoints []CLIEndpointSummary `json:"endpoints"`
	Count     int                  `json:"count"`
	Timestamp time.Time            `json:"timestamp"`
}

// CLIEndpointSummary represents a summary of a CLI endpoint
type CLIEndpointSummary struct {
	ID           string            `json:"id"`
	URL          string            `json:"url"`
	Version      string            `json:"version"`
	Capabilities []string          `json:"capabilities"`
	Status       string            `json:"status"`
	IsHealthy    bool              `json:"is_healthy"`
	RegisteredAt time.Time         `json:"registered_at"`
	LastSeen     time.Time         `json:"last_seen"`
	Metadata     map[string]string `json:"metadata"`
}

// CLIEndpointDetail represents detailed information about a CLI endpoint
type CLIEndpointDetail struct {
	ID           string                        `json:"id"`
	URL          string                        `json:"url"`
	Version      string                        `json:"version"`
	Capabilities []string                      `json:"capabilities"`
	Metadata     map[string]string             `json:"metadata"`
	RegisteredAt time.Time                     `json:"registered_at"`
	LastSeen     time.Time                     `json:"last_seen"`
	Status       string                        `json:"status"`
	Health       *push.EndpointHealth          `json:"health"`
	Preferences  *push.NotificationPreferences `json:"preferences"`
}
