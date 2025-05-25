// Package server implements the MCP server functionality
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"mcp-memory/pkg/mcp/protocol"
	"mcp-memory/pkg/mcp/transport"
	"sync"
)

// Server represents an MCP server
type Server struct {
	name         string
	version      string
	capabilities protocol.ServerCapabilities
	tools        map[string]*ToolRegistration
	resources    map[string]*ResourceRegistration
	prompts      map[string]*PromptRegistration
	transport    transport.Transport
	mutex        sync.RWMutex
	initialized  bool
}

// ToolRegistration represents a registered tool
type ToolRegistration struct {
	Tool    protocol.Tool
	Handler protocol.ToolHandler
}

// ResourceRegistration represents a registered resource
type ResourceRegistration struct {
	Resource protocol.Resource
	Handler  ResourceHandler
}

// PromptRegistration represents a registered prompt
type PromptRegistration struct {
	Prompt  protocol.Prompt
	Handler PromptHandler
}

// ResourceHandler defines the interface for resource handlers
type ResourceHandler interface {
	Handle(ctx context.Context, uri string) ([]protocol.Content, error)
}

// PromptHandler defines the interface for prompt handlers
type PromptHandler interface {
	Handle(ctx context.Context, args map[string]interface{}) ([]protocol.Content, error)
}

// NewServer creates a new MCP server
func NewServer(name, version string) *Server {
	return &Server{
		name:      name,
		version:   version,
		tools:     make(map[string]*ToolRegistration),
		resources: make(map[string]*ResourceRegistration),
		prompts:   make(map[string]*PromptRegistration),
		capabilities: protocol.ServerCapabilities{
			Tools: &protocol.ToolCapability{
				ListChanged: false,
			},
			Resources: &protocol.ResourceCapability{
				Subscribe:   false,
				ListChanged: false,
			},
			Prompts: &protocol.PromptCapability{
				ListChanged: false,
			},
		},
	}
}

// AddTool registers a new tool
func (s *Server) AddTool(tool protocol.Tool, handler protocol.ToolHandler) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.tools[tool.Name] = &ToolRegistration{
		Tool:    tool,
		Handler: handler,
	}
}

// AddResource registers a new resource
func (s *Server) AddResource(resource protocol.Resource, handler ResourceHandler) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.resources[resource.URI] = &ResourceRegistration{
		Resource: resource,
		Handler:  handler,
	}
}

// AddPrompt registers a new prompt
func (s *Server) AddPrompt(prompt protocol.Prompt, handler PromptHandler) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.prompts[prompt.Name] = &PromptRegistration{
		Prompt:  prompt,
		Handler: handler,
	}
}

// SetTransport sets the transport layer
func (s *Server) SetTransport(t transport.Transport) {
	s.transport = t
}

// Start starts the server
func (s *Server) Start(ctx context.Context) error {
	if s.transport == nil {
		return fmt.Errorf("no transport configured")
	}

	return s.transport.Start(ctx, s)
}

// HandleRequest handles an incoming JSON-RPC request
func (s *Server) HandleRequest(ctx context.Context, req *protocol.JSONRPCRequest) *protocol.JSONRPCResponse {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(ctx, req)
	case "tools/list":
		return s.handleToolsList(ctx, req)
	case "tools/call":
		return s.handleToolsCall(ctx, req)
	case "resources/list":
		return s.handleResourcesList(ctx, req)
	case "resources/read":
		return s.handleResourcesRead(ctx, req)
	case "prompts/list":
		return s.handlePromptsList(ctx, req)
	case "prompts/get":
		return s.handlePromptsGet(ctx, req)
	default:
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   protocol.NewJSONRPCError(protocol.MethodNotFound, "Method not found", nil),
		}
	}
}

// handleInitialize handles the initialize request
func (s *Server) handleInitialize(_ context.Context, req *protocol.JSONRPCRequest) *protocol.JSONRPCResponse {
	var initReq protocol.InitializeRequest
	if err := parseParams(req.Params, &initReq); err != nil {
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   protocol.NewJSONRPCError(protocol.InvalidParams, "Invalid parameters", err.Error()),
		}
	}

	s.mutex.Lock()
	s.initialized = true
	s.mutex.Unlock()

	result := protocol.InitializeResult{
		ProtocolVersion: protocol.Version,
		Capabilities:    s.capabilities,
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

// handleToolsList handles the tools/list request
func (s *Server) handleToolsList(_ context.Context, req *protocol.JSONRPCRequest) *protocol.JSONRPCResponse {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	tools := make([]protocol.Tool, 0, len(s.tools))
	for _, registration := range s.tools {
		tools = append(tools, registration.Tool)
	}

	result := map[string]interface{}{
		"tools": tools,
	}

	return &protocol.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

// handleToolsCall handles the tools/call request
func (s *Server) handleToolsCall(ctx context.Context, req *protocol.JSONRPCRequest) *protocol.JSONRPCResponse {
	var callReq protocol.ToolCallRequest
	if err := parseParams(req.Params, &callReq); err != nil {
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   protocol.NewJSONRPCError(protocol.InvalidParams, "Invalid parameters", err.Error()),
		}
	}

	s.mutex.RLock()
	registration, exists := s.tools[callReq.Name]
	s.mutex.RUnlock()

	if !exists {
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   protocol.NewJSONRPCError(protocol.MethodNotFound, "Tool not found", nil),
		}
	}

	result, err := registration.Handler.Handle(ctx, callReq.Arguments)
	if err != nil {
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  protocol.NewToolCallError(err.Error()),
		}
	}

	// Convert result to tool call result format
	var toolResult *protocol.ToolCallResult
	switch v := result.(type) {
	case *protocol.ToolCallResult:
		toolResult = v
	case string:
		toolResult = protocol.NewToolCallResult(protocol.NewContent(v))
	default:
		// Try to marshal to JSON and use as text
		if jsonBytes, err := json.Marshal(result); err == nil {
			toolResult = protocol.NewToolCallResult(protocol.NewContent(string(jsonBytes)))
		} else {
			toolResult = protocol.NewToolCallResult(protocol.NewContent(fmt.Sprintf("%v", result)))
		}
	}

	return &protocol.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  toolResult,
	}
}

// handleResourcesList handles the resources/list request
func (s *Server) handleResourcesList(_ context.Context, req *protocol.JSONRPCRequest) *protocol.JSONRPCResponse {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	resources := make([]protocol.Resource, 0, len(s.resources))
	for _, registration := range s.resources {
		resources = append(resources, registration.Resource)
	}

	result := map[string]interface{}{
		"resources": resources,
	}

	return &protocol.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

// handleResourcesRead handles the resources/read request
func (s *Server) handleResourcesRead(ctx context.Context, req *protocol.JSONRPCRequest) *protocol.JSONRPCResponse {
	params, ok := req.Params.(map[string]interface{})
	if !ok {
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   protocol.NewJSONRPCError(protocol.InvalidParams, "Invalid parameters", nil),
		}
	}

	uri, ok := params["uri"].(string)
	if !ok {
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   protocol.NewJSONRPCError(protocol.InvalidParams, "URI parameter required", nil),
		}
	}

	s.mutex.RLock()
	registration, exists := s.resources[uri]
	s.mutex.RUnlock()

	if !exists {
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   protocol.NewJSONRPCError(protocol.MethodNotFound, "Resource not found", nil),
		}
	}

	content, err := registration.Handler.Handle(ctx, uri)
	if err != nil {
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   protocol.NewJSONRPCError(protocol.InternalError, err.Error(), nil),
		}
	}

	result := map[string]interface{}{
		"contents": content,
	}

	return &protocol.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

// handlePromptsList handles the prompts/list request
func (s *Server) handlePromptsList(_ context.Context, req *protocol.JSONRPCRequest) *protocol.JSONRPCResponse {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	prompts := make([]protocol.Prompt, 0, len(s.prompts))
	for _, registration := range s.prompts {
		prompts = append(prompts, registration.Prompt)
	}

	result := map[string]interface{}{
		"prompts": prompts,
	}

	return &protocol.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

// handlePromptsGet handles the prompts/get request
func (s *Server) handlePromptsGet(ctx context.Context, req *protocol.JSONRPCRequest) *protocol.JSONRPCResponse {
	params, ok := req.Params.(map[string]interface{})
	if !ok {
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   protocol.NewJSONRPCError(protocol.InvalidParams, "Invalid parameters", nil),
		}
	}

	name, ok := params["name"].(string)
	if !ok {
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   protocol.NewJSONRPCError(protocol.InvalidParams, "Name parameter required", nil),
		}
	}

	args, _ := params["arguments"].(map[string]interface{})

	s.mutex.RLock()
	registration, exists := s.prompts[name]
	s.mutex.RUnlock()

	if !exists {
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   protocol.NewJSONRPCError(protocol.MethodNotFound, "Prompt not found", nil),
		}
	}

	content, err := registration.Handler.Handle(ctx, args)
	if err != nil {
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   protocol.NewJSONRPCError(protocol.InternalError, err.Error(), nil),
		}
	}

	result := map[string]interface{}{
		"messages": content,
	}

	return &protocol.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

// parseParams is a helper function to parse JSON-RPC parameters
func parseParams(params interface{}, target interface{}) error {
	if params == nil {
		return nil
	}

	// Convert to JSON and back to ensure proper type conversion
	jsonBytes, err := json.Marshal(params)
	if err != nil {
		return err
	}

	return json.Unmarshal(jsonBytes, target)
}

// IsInitialized returns whether the server has been initialized
func (s *Server) IsInitialized() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.initialized
}