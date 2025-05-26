package server

import (
	"context"
	"encoding/json"
	"mcp-memory/pkg/mcp/protocol"
	"mcp-memory/pkg/mcp/roots"
	"mcp-memory/pkg/mcp/sampling"
)

// ExtendedServer provides additional MCP protocol features
type ExtendedServer struct {
	*Server
	samplingHandler *sampling.Handler
	rootsHandler    *roots.Handler
}

// NewExtendedServer creates a new extended MCP server with all features
func NewExtendedServer(name, version string) *ExtendedServer {
	base := NewServer(name, version)
	
	// Update capabilities to include new features
	base.capabilities.Sampling = &protocol.SamplingCapability{}
	base.capabilities.Roots = &protocol.RootsCapability{
		ListChanged: false,
	}
	
	return &ExtendedServer{
		Server:          base,
		samplingHandler: sampling.NewHandler(),
		rootsHandler:    roots.NewHandler(),
	}
}

// SetSamplingHandler sets a custom sampling handler
func (s *ExtendedServer) SetSamplingHandler(handler *sampling.Handler) {
	s.samplingHandler = handler
}

// SetRootsHandler sets a custom roots handler
func (s *ExtendedServer) SetRootsHandler(handler *roots.Handler) {
	s.rootsHandler = handler
}

// HandleRequest extends the base server to handle additional methods
func (s *ExtendedServer) HandleRequest(ctx context.Context, req *protocol.JSONRPCRequest) *protocol.JSONRPCResponse {
	// Check for extended methods first
	switch req.Method {
	case "sampling/createMessage":
		return s.handleSamplingCreateMessage(ctx, req)
	case "roots/list":
		return s.handleRootsList(ctx, req)
	default:
		// Fall back to base server methods
		return s.Server.HandleRequest(ctx, req)
	}
}

// handleSamplingCreateMessage handles the sampling/createMessage request
func (s *ExtendedServer) handleSamplingCreateMessage(ctx context.Context, req *protocol.JSONRPCRequest) *protocol.JSONRPCResponse {
	if !s.initialized {
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   protocol.NewJSONRPCError(protocol.InvalidRequest, "Server not initialized", nil),
		}
	}
	
	if s.samplingHandler == nil {
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   protocol.NewJSONRPCError(protocol.MethodNotFound, "Sampling not supported", nil),
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
	
	result, err := s.samplingHandler.CreateMessage(ctx, paramsJSON)
	if err != nil {
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   protocol.NewJSONRPCError(protocol.InternalError, "Sampling failed", err.Error()),
		}
	}
	
	return &protocol.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

// handleRootsList handles the roots/list request
func (s *ExtendedServer) handleRootsList(ctx context.Context, req *protocol.JSONRPCRequest) *protocol.JSONRPCResponse {
	if !s.initialized {
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   protocol.NewJSONRPCError(protocol.InvalidRequest, "Server not initialized", nil),
		}
	}
	
	if s.rootsHandler == nil {
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   protocol.NewJSONRPCError(protocol.MethodNotFound, "Roots not supported", nil),
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
	
	result, err := s.rootsHandler.ListRoots(ctx, paramsJSON)
	if err != nil {
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   protocol.NewJSONRPCError(protocol.InternalError, "Failed to list roots", err.Error()),
		}
	}
	
	return &protocol.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

// AddRoot adds a root to the server
func (s *ExtendedServer) AddRoot(root roots.Root) {
	if s.rootsHandler != nil {
		s.rootsHandler.AddRoot(root)
	}
}

// GetRoots returns all registered roots
func (s *ExtendedServer) GetRoots() []roots.Root {
	if s.rootsHandler != nil {
		return s.rootsHandler.GetRoots()
	}
	return []roots.Root{}
}