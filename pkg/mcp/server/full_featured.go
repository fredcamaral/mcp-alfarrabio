package server

import (
	"context"
	"encoding/json"
	"fmt"
	"mcp-memory/pkg/mcp/compatibility"
	"mcp-memory/pkg/mcp/discovery"
	"mcp-memory/pkg/mcp/notifications"
	"mcp-memory/pkg/mcp/protocol"
	_ "mcp-memory/pkg/mcp/roots"    // Used by ExtendedServer
	_ "mcp-memory/pkg/mcp/sampling" // Used by ExtendedServer
	"mcp-memory/pkg/mcp/subscriptions"
)

// FullFeaturedServer implements all MCP protocol features with compatibility
type FullFeaturedServer struct {
	*ExtendedServer
	discoveryService    *discovery.Service
	subscriptionManager *subscriptions.Manager
	notifier           *notifications.Notifier
	compatDetector     *compatibility.Detector
	clientProfiles     map[string]*compatibility.ClientProfile
	fallbackHandlers   map[string]*compatibility.FallbackHandler
}

// NewFullFeaturedServer creates a server with all MCP features
func NewFullFeaturedServer(name, version string) *FullFeaturedServer {
	base := NewExtendedServer(name, version)
	
	s := &FullFeaturedServer{
		ExtendedServer:      base,
		discoveryService:    discovery.NewService(),
		subscriptionManager: subscriptions.NewManager(100), // 100 subscriptions per client
		notifier:           notifications.NewNotifier(1000), // 1000 notification queue
		compatDetector:     compatibility.NewDetector(),
		clientProfiles:     make(map[string]*compatibility.ClientProfile),
		fallbackHandlers:   make(map[string]*compatibility.FallbackHandler),
	}
	
	// Wire up discovery events to notifications
	s.setupDiscoveryNotifications()
	
	// Wire up subscription handlers
	s.setupSubscriptionHandlers()
	
	return s
}

// Start starts all server components
func (s *FullFeaturedServer) Start(ctx context.Context) error {
	// Start base server
	if err := s.ExtendedServer.Start(ctx); err != nil {
		return err
	}
	
	// Start additional services
	s.subscriptionManager.Start(ctx)
	s.notifier.Start(ctx)
	
	if err := s.discoveryService.Start(ctx); err != nil {
		return fmt.Errorf("failed to start discovery service: %w", err)
	}
	
	return nil
}

// Stop stops all server components
func (s *FullFeaturedServer) Stop() error {
	s.discoveryService.Stop()
	s.notifier.Stop()
	s.subscriptionManager.Stop()
	return nil
}

// HandleRequest handles requests with full compatibility support
func (s *FullFeaturedServer) HandleRequest(ctx context.Context, req *protocol.JSONRPCRequest) *protocol.JSONRPCResponse {
	// Get client ID from context (would be set by transport layer)
	clientID, _ := ctx.Value("clientID").(string)
	
	// Check if this is an extended method
	switch req.Method {
	case "initialize":
		return s.handleInitializeWithCompatibility(ctx, req)
		
	case "discovery/discover":
		return s.handleDiscovery(ctx, req, clientID)
		
	case "resources/subscribe":
		return s.handleResourceSubscribe(ctx, req, clientID)
		
	case "resources/unsubscribe":
		return s.handleUnsubscribe(ctx, req, clientID)
		
	case "tools/subscribe", "prompts/subscribe", "roots/subscribe":
		return s.handleListSubscribe(ctx, req, clientID)
		
	default:
		// Check client compatibility for the method
		if profile, exists := s.clientProfiles[clientID]; exists {
			feature := s.methodToFeature(req.Method)
			supported, _ := s.compatDetector.CheckFeatureSupport(profile, feature)
			
			if !supported {
				// Use fallback handler
				if handler, exists := s.fallbackHandlers[clientID]; exists {
					return handler.HandleUnsupportedMethod(req.Method)
				}
			}
		}
		
		// Fall back to extended server
		return s.ExtendedServer.HandleRequest(ctx, req)
	}
}

// handleInitializeWithCompatibility handles initialization with client detection
func (s *FullFeaturedServer) handleInitializeWithCompatibility(ctx context.Context, req *protocol.JSONRPCRequest) *protocol.JSONRPCResponse {
	var initReq protocol.InitializeRequest
	if err := parseParamsFullFeatured(req.Params, &initReq); err != nil {
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   protocol.NewJSONRPCError(protocol.InvalidParams, "Invalid parameters", err.Error()),
		}
	}
	
	// Detect client profile
	profile := s.compatDetector.DetectClient(initReq.ClientInfo, initReq.Capabilities)
	
	// Store client profile
	clientID, _ := ctx.Value("clientID").(string)
	if clientID != "" && profile != nil {
		s.mutex.Lock()
		s.clientProfiles[clientID] = profile
		s.fallbackHandlers[clientID] = compatibility.NewFallbackHandler(profile)
		s.mutex.Unlock()
		
		// Register client with notifier
		supportsProgress := false
		supportsLogging := false
		if initReq.Capabilities.Experimental != nil {
			_, supportsProgress = initReq.Capabilities.Experimental["progress"].(bool)
			_, supportsLogging = initReq.Capabilities.Experimental["logging"].(bool)
		}
		s.notifier.RegisterClient(clientID, supportsProgress, supportsLogging)
	}
	
	// Build capabilities based on client profile
	serverCaps := s.compatDetector.GetSupportedCapabilities(profile)
	
	// Merge with our full capabilities (but filtered by client support)
	if s.capabilities.Tools != nil && serverCaps.Tools != nil {
		serverCaps.Tools = s.capabilities.Tools
	}
	if s.capabilities.Resources != nil && serverCaps.Resources != nil {
		serverCaps.Resources = s.capabilities.Resources
	}
	if s.capabilities.Prompts != nil && serverCaps.Prompts != nil {
		serverCaps.Prompts = s.capabilities.Prompts
	}
	
	s.mutex.Lock()
	s.initialized = true
	s.mutex.Unlock()
	
	result := protocol.InitializeResult{
		ProtocolVersion: protocol.Version,
		Capabilities:    serverCaps,
		ServerInfo: protocol.ServerInfo{
			Name:    s.name,
			Version: s.version,
		},
	}
	
	return &protocol.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

// handleDiscovery handles discovery requests
func (s *FullFeaturedServer) handleDiscovery(ctx context.Context, req *protocol.JSONRPCRequest, clientID string) *protocol.JSONRPCResponse {
	// Check if client supports discovery
	if profile, exists := s.clientProfiles[clientID]; exists {
		supported, workaround := s.compatDetector.CheckFeatureSupport(profile, compatibility.FeatureDiscovery)
		if !supported {
			return &protocol.JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: protocol.NewJSONRPCError(
					-32002,
					"Discovery not supported by client",
					map[string]interface{}{
						"workaround": workaround,
					},
				),
			}
		}
	}
	
	// Convert params to JSON for the handler
	paramsJSON, err := json.Marshal(req.Params)
	if err != nil {
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   protocol.NewJSONRPCError(protocol.InvalidParams, "Invalid parameters", err.Error()),
		}
	}
	
	result, err := s.discoveryService.HandleDiscover(ctx, paramsJSON)
	if err != nil {
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   protocol.NewJSONRPCError(protocol.InternalError, "Discovery failed", err.Error()),
		}
	}
	
	return &protocol.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

// handleResourceSubscribe handles resource subscription requests
func (s *FullFeaturedServer) handleResourceSubscribe(ctx context.Context, req *protocol.JSONRPCRequest, clientID string) *protocol.JSONRPCResponse {
	if clientID == "" {
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   protocol.NewJSONRPCError(protocol.InvalidRequest, "Client ID required for subscriptions", nil),
		}
	}
	
	paramsJSON, err := json.Marshal(req.Params)
	if err != nil {
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   protocol.NewJSONRPCError(protocol.InvalidParams, "Invalid parameters", err.Error()),
		}
	}
	
	sub, err := s.subscriptionManager.Subscribe(clientID, "resources/subscribe", paramsJSON)
	if err != nil {
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   protocol.NewJSONRPCError(protocol.InternalError, "Subscription failed", err.Error()),
		}
	}
	
	return &protocol.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: subscriptions.SubscriptionResponse{
			SubscriptionID: sub.ID,
		},
	}
}

// handleListSubscribe handles list subscription requests
func (s *FullFeaturedServer) handleListSubscribe(ctx context.Context, req *protocol.JSONRPCRequest, clientID string) *protocol.JSONRPCResponse {
	if clientID == "" {
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   protocol.NewJSONRPCError(protocol.InvalidRequest, "Client ID required for subscriptions", nil),
		}
	}
	
	sub, err := s.subscriptionManager.Subscribe(clientID, req.Method, nil)
	if err != nil {
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   protocol.NewJSONRPCError(protocol.InternalError, "Subscription failed", err.Error()),
		}
	}
	
	return &protocol.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: subscriptions.SubscriptionResponse{
			SubscriptionID: sub.ID,
		},
	}
}

// handleUnsubscribe handles unsubscribe requests
func (s *FullFeaturedServer) handleUnsubscribe(ctx context.Context, req *protocol.JSONRPCRequest, clientID string) *protocol.JSONRPCResponse {
	var unsubReq subscriptions.UnsubscribeRequest
	if err := parseParamsFullFeatured(req.Params, &unsubReq); err != nil {
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   protocol.NewJSONRPCError(protocol.InvalidParams, "Invalid parameters", err.Error()),
		}
	}
	
	if err := s.subscriptionManager.Unsubscribe(unsubReq.SubscriptionID); err != nil {
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   protocol.NewJSONRPCError(protocol.InvalidRequest, "Unsubscribe failed", err.Error()),
		}
	}
	
	return &protocol.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  map[string]interface{}{},
	}
}

// setupDiscoveryNotifications wires discovery events to notifications
func (s *FullFeaturedServer) setupDiscoveryNotifications() {
	eventChan := s.discoveryService.Subscribe()
	
	go func() {
		for event := range eventChan {
			switch event.Category {
			case "tool":
				s.notifier.NotifyToolsListChanged()
			case "resource":
				s.notifier.NotifyResourcesListChanged()
			case "prompt":
				s.notifier.NotifyPromptsListChanged()
			}
		}
	}()
}

// setupSubscriptionHandlers sets up subscription event handlers
func (s *FullFeaturedServer) setupSubscriptionHandlers() {
	// Handler for delivering notifications to subscribed clients
	s.subscriptionManager.RegisterHandler("resources/subscribe", func(ctx context.Context, event subscriptions.Event) error {
		// Send notification to the specific client
		return s.notifier.SendToClient(
			event.SubscriptionID,
			notifications.NotificationResourceChanged,
			event.Data,
		)
	})
}

// methodToFeature maps methods to features for compatibility checking
func (s *FullFeaturedServer) methodToFeature(method string) string {
	switch {
	case hasPrefix(method, "tools/"):
		return compatibility.FeatureTools
	case hasPrefix(method, "resources/"):
		return compatibility.FeatureResources
	case hasPrefix(method, "prompts/"):
		return compatibility.FeaturePrompts
	case hasPrefix(method, "roots/"):
		return compatibility.FeatureRoots
	case hasPrefix(method, "sampling/"):
		return compatibility.FeatureSampling
	case hasPrefix(method, "discovery/"):
		return compatibility.FeatureDiscovery
	default:
		return "unknown"
	}
}

// Helper functions
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// parseParamsFullFeatured is a helper to parse request parameters
func parseParamsFullFeatured(params interface{}, target interface{}) error {
	if params == nil {
		return nil
	}
	
	// Convert to JSON and back to target type
	data, err := json.Marshal(params)
	if err != nil {
		return err
	}
	
	return json.Unmarshal(data, target)
}