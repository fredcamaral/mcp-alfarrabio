package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mcp-memory/pkg/mcp/protocol"
	"mcp-memory/pkg/mcp/transport"
	"sync"
	"testing"
	"time"
)

// Mock transport for testing
type mockTransport struct {
	started  bool
	server   transport.RequestHandler
	requests []protocol.JSONRPCRequest
	mu       sync.Mutex
}

func (mt *mockTransport) Start(ctx context.Context, handler transport.RequestHandler) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	mt.started = true
	mt.server = handler
	return nil
}

func (mt *mockTransport) Stop() error {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	mt.started = false
	return nil
}

func (mt *mockTransport) simulateRequest(req *protocol.JSONRPCRequest) *protocol.JSONRPCResponse {
	mt.mu.Lock()
	mt.requests = append(mt.requests, *req)
	handler := mt.server
	mt.mu.Unlock()

	if handler != nil {
		return handler.HandleRequest(context.Background(), req)
	}
	return nil
}

// Mock handlers for testing
type mockToolHandler struct {
	result interface{}
	err    error
}

func (h *mockToolHandler) Handle(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	return h.result, h.err
}

type mockResourceHandler struct {
	content []protocol.Content
	err     error
}

func (h *mockResourceHandler) Handle(ctx context.Context, uri string) ([]protocol.Content, error) {
	return h.content, h.err
}

type mockPromptHandler struct {
	content []protocol.Content
	err     error
}

func (h *mockPromptHandler) Handle(ctx context.Context, args map[string]interface{}) ([]protocol.Content, error) {
	return h.content, h.err
}

func TestNewServer(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	if server.name != "test-server" {
		t.Errorf("Expected name 'test-server', got %s", server.name)
	}
	if server.version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %s", server.version)
	}
	if server.tools == nil {
		t.Error("Expected tools map to be initialized")
	}
	if server.resources == nil {
		t.Error("Expected resources map to be initialized")
	}
	if server.prompts == nil {
		t.Error("Expected prompts map to be initialized")
	}
	if server.initialized {
		t.Error("Expected server to not be initialized")
	}

	// Check capabilities
	if server.capabilities.Tools == nil {
		t.Error("Expected Tools capability to be initialized")
	}
	if server.capabilities.Resources == nil {
		t.Error("Expected Resources capability to be initialized")
	}
	if server.capabilities.Prompts == nil {
		t.Error("Expected Prompts capability to be initialized")
	}
}

func TestAddTool(t *testing.T) {
	server := NewServer("test", "1.0")
	
	tool := protocol.Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
		},
	}
	
	handler := &mockToolHandler{result: "test result"}
	
	server.AddTool(tool, handler)
	
	// Check tool was added
	if len(server.tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(server.tools))
	}
	
	reg, exists := server.tools["test_tool"]
	if !exists {
		t.Error("Expected tool 'test_tool' to exist")
	}
	
	if reg.Tool.Name != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got %s", reg.Tool.Name)
	}
	
	// Add another tool
	tool2 := protocol.Tool{
		Name: "test_tool2",
	}
	server.AddTool(tool2, handler)
	
	if len(server.tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(server.tools))
	}
}

func TestAddResource(t *testing.T) {
	server := NewServer("test", "1.0")
	
	resource := protocol.Resource{
		URI:         "file:///test.txt",
		Name:        "test.txt",
		Description: "A test file",
		MimeType:    "text/plain",
	}
	
	handler := &mockResourceHandler{
		content: []protocol.Content{
			protocol.NewContent("test content"),
		},
	}
	
	server.AddResource(resource, handler)
	
	// Check resource was added
	if len(server.resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(server.resources))
	}
	
	reg, exists := server.resources["file:///test.txt"]
	if !exists {
		t.Error("Expected resource 'file:///test.txt' to exist")
	}
	
	if reg.Resource.URI != "file:///test.txt" {
		t.Errorf("Expected resource URI 'file:///test.txt', got %s", reg.Resource.URI)
	}
}

func TestAddPrompt(t *testing.T) {
	server := NewServer("test", "1.0")
	
	prompt := protocol.Prompt{
		Name:        "test_prompt",
		Description: "A test prompt",
		Arguments: []protocol.PromptArgument{
			{
				Name:     "name",
				Required: true,
			},
		},
	}
	
	handler := &mockPromptHandler{
		content: []protocol.Content{
			protocol.NewContent("Hello, {{name}}!"),
		},
	}
	
	server.AddPrompt(prompt, handler)
	
	// Check prompt was added
	if len(server.prompts) != 1 {
		t.Errorf("Expected 1 prompt, got %d", len(server.prompts))
	}
	
	reg, exists := server.prompts["test_prompt"]
	if !exists {
		t.Error("Expected prompt 'test_prompt' to exist")
	}
	
	if reg.Prompt.Name != "test_prompt" {
		t.Errorf("Expected prompt name 'test_prompt', got %s", reg.Prompt.Name)
	}
}

func TestSetTransport(t *testing.T) {
	server := NewServer("test", "1.0")
	transport := &mockTransport{}
	
	server.SetTransport(transport)
	
	if server.transport != transport {
		t.Error("Expected transport to be set")
	}
}

func TestStart(t *testing.T) {
	t.Run("start with transport", func(t *testing.T) {
		server := NewServer("test", "1.0")
		transport := &mockTransport{}
		server.SetTransport(transport)
		
		ctx := context.Background()
		err := server.Start(ctx)
		
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		
		if !transport.started {
			t.Error("Expected transport to be started")
		}
	})
	
	t.Run("start without transport", func(t *testing.T) {
		server := NewServer("test", "1.0")
		
		ctx := context.Background()
		err := server.Start(ctx)
		
		if err == nil {
			t.Error("Expected error when starting without transport")
		}
		
		if err.Error() != "no transport configured" {
			t.Errorf("Expected 'no transport configured' error, got %v", err)
		}
	})
}

func TestHandleInitialize(t *testing.T) {
	server := NewServer("test-server", "1.0.0")
	transport := &mockTransport{}
	server.SetTransport(transport)
	server.Start(context.Background())
	
	tests := []struct {
		name           string
		request        *protocol.JSONRPCRequest
		expectError    bool
		expectedResult protocol.InitializeResult
	}{
		{
			name: "valid initialization",
			request: &protocol.JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "initialize",
				Params: protocol.InitializeRequest{
					ProtocolVersion: protocol.Version,
					Capabilities:    protocol.ClientCapabilities{},
					ClientInfo: protocol.ClientInfo{
						Name:    "test-client",
						Version: "1.0",
					},
				},
			},
			expectError: false,
			expectedResult: protocol.InitializeResult{
				ProtocolVersion: protocol.Version,
				Capabilities:    server.capabilities,
				ServerInfo: protocol.ServerInfo{
					Name:    "test-server",
					Version: "1.0.0",
				},
			},
		},
		{
			name: "invalid params",
			request: &protocol.JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      2,
				Method:  "initialize",
				Params:  "invalid",
			},
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := transport.simulateRequest(tt.request)
			
			if tt.expectError {
				if resp.Error == nil {
					t.Error("Expected error response")
				}
			} else {
				if resp.Error != nil {
					t.Errorf("Unexpected error: %v", resp.Error)
				}
				
				// Check that server is initialized
				if !server.IsInitialized() {
					t.Error("Expected server to be initialized")
				}
				
				// Verify result structure
				resultJSON, _ := json.Marshal(resp.Result)
				var result protocol.InitializeResult
				json.Unmarshal(resultJSON, &result)
				
				if result.ProtocolVersion != tt.expectedResult.ProtocolVersion {
					t.Errorf("Protocol version mismatch: got %s, want %s", 
						result.ProtocolVersion, tt.expectedResult.ProtocolVersion)
				}
				if result.ServerInfo.Name != tt.expectedResult.ServerInfo.Name {
					t.Errorf("Server name mismatch: got %s, want %s", 
						result.ServerInfo.Name, tt.expectedResult.ServerInfo.Name)
				}
			}
		})
	}
}

func TestHandleToolsList(t *testing.T) {
	server := NewServer("test", "1.0")
	transport := &mockTransport{}
	server.SetTransport(transport)
	server.Start(context.Background())
	
	// Add some tools
	tool1 := protocol.Tool{
		Name:        "tool1",
		Description: "First tool",
		InputSchema: map[string]interface{}{"type": "object"},
	}
	tool2 := protocol.Tool{
		Name:        "tool2",
		Description: "Second tool",
		InputSchema: map[string]interface{}{"type": "object"},
	}
	
	server.AddTool(tool1, &mockToolHandler{})
	server.AddTool(tool2, &mockToolHandler{})
	
	// Test tools/list
	req := &protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}
	
	resp := transport.simulateRequest(req)
	
	if resp.Error != nil {
		t.Errorf("Unexpected error: %v", resp.Error)
	}
	
	// Check result
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}
	
	tools, ok := result["tools"].([]interface{})
	if !ok {
		// Try as []protocol.Tool due to type assertion
		toolsTyped, ok := result["tools"].([]protocol.Tool)
		if !ok {
			t.Fatal("Expected tools to be an array")
		}
		if len(toolsTyped) != 2 {
			t.Errorf("Expected 2 tools, got %d", len(toolsTyped))
		}
	} else {
		if len(tools) != 2 {
			t.Errorf("Expected 2 tools, got %d", len(tools))
		}
	}
}

func TestHandleToolsCall(t *testing.T) {
	server := NewServer("test", "1.0")
	transport := &mockTransport{}
	server.SetTransport(transport)
	server.Start(context.Background())
	
	tests := []struct {
		name          string
		setupTools    func()
		request       *protocol.JSONRPCRequest
		expectError   bool
		checkResponse func(t *testing.T, resp *protocol.JSONRPCResponse)
	}{
		{
			name: "successful tool call",
			setupTools: func() {
				tool := protocol.Tool{
					Name: "echo",
					InputSchema: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"message": map[string]interface{}{"type": "string"},
						},
					},
				}
				handler := &mockToolHandler{
					result: "echoed: test message",
				}
				server.AddTool(tool, handler)
			},
			request: &protocol.JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "tools/call",
				Params: protocol.ToolCallRequest{
					Name: "echo",
					Arguments: map[string]interface{}{
						"message": "test message",
					},
				},
			},
			expectError: false,
			checkResponse: func(t *testing.T, resp *protocol.JSONRPCResponse) {
				// Result should be a ToolCallResult
				resultJSON, _ := json.Marshal(resp.Result)
				var result protocol.ToolCallResult
				json.Unmarshal(resultJSON, &result)
				
				if result.IsError {
					t.Error("Expected successful result, got error")
				}
				if len(result.Content) != 1 {
					t.Errorf("Expected 1 content item, got %d", len(result.Content))
				}
			},
		},
		{
			name: "tool returns ToolCallResult",
			setupTools: func() {
				tool := protocol.Tool{Name: "result_tool"}
				handler := &mockToolHandler{
					result: protocol.NewToolCallResult(
						protocol.NewContent("line 1"),
						protocol.NewContent("line 2"),
					),
				}
				server.AddTool(tool, handler)
			},
			request: &protocol.JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      2,
				Method:  "tools/call",
				Params: protocol.ToolCallRequest{
					Name: "result_tool",
				},
			},
			expectError: false,
			checkResponse: func(t *testing.T, resp *protocol.JSONRPCResponse) {
				result, ok := resp.Result.(*protocol.ToolCallResult)
				if !ok {
					// Try unmarshaling
					resultJSON, _ := json.Marshal(resp.Result)
					var r protocol.ToolCallResult
					json.Unmarshal(resultJSON, &r)
					result = &r
				}
				
				if result.IsError {
					t.Error("Expected successful result")
				}
				if len(result.Content) != 2 {
					t.Errorf("Expected 2 content items, got %d", len(result.Content))
				}
			},
		},
		{
			name: "tool returns complex object",
			setupTools: func() {
				tool := protocol.Tool{Name: "complex_tool"}
				handler := &mockToolHandler{
					result: map[string]interface{}{
						"status": "success",
						"data": map[string]interface{}{
							"count": 42,
						},
					},
				}
				server.AddTool(tool, handler)
			},
			request: &protocol.JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      3,
				Method:  "tools/call",
				Params: protocol.ToolCallRequest{
					Name: "complex_tool",
				},
			},
			expectError: false,
			checkResponse: func(t *testing.T, resp *protocol.JSONRPCResponse) {
				// Should convert to JSON string
				resultJSON, _ := json.Marshal(resp.Result)
				var result protocol.ToolCallResult
				json.Unmarshal(resultJSON, &result)
				
				if result.IsError {
					t.Error("Expected successful result")
				}
				if len(result.Content) != 1 {
					t.Errorf("Expected 1 content item, got %d", len(result.Content))
				}
			},
		},
		{
			name:       "tool not found",
			setupTools: func() {},
			request: &protocol.JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      4,
				Method:  "tools/call",
				Params: protocol.ToolCallRequest{
					Name: "nonexistent",
				},
			},
			expectError: true,
		},
		{
			name: "tool handler error",
			setupTools: func() {
				tool := protocol.Tool{Name: "error_tool"}
				handler := &mockToolHandler{
					err: errors.New("tool execution failed"),
				}
				server.AddTool(tool, handler)
			},
			request: &protocol.JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      5,
				Method:  "tools/call",
				Params: protocol.ToolCallRequest{
					Name: "error_tool",
				},
			},
			expectError: false, // Tool errors return as ToolCallResult with IsError=true
			checkResponse: func(t *testing.T, resp *protocol.JSONRPCResponse) {
				resultJSON, _ := json.Marshal(resp.Result)
				var result protocol.ToolCallResult
				json.Unmarshal(resultJSON, &result)
				
				if !result.IsError {
					t.Error("Expected error result")
				}
				if len(result.Content) != 1 {
					t.Errorf("Expected 1 content item, got %d", len(result.Content))
				}
				if result.Content[0].Text != "tool execution failed" {
					t.Errorf("Expected error message 'tool execution failed', got %s", result.Content[0].Text)
				}
			},
		},
		{
			name:       "invalid params",
			setupTools: func() {},
			request: &protocol.JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      6,
				Method:  "tools/call",
				Params:  "invalid",
			},
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear tools
			server.tools = make(map[string]*ToolRegistration)
			
			// Setup tools
			tt.setupTools()
			
			// Execute request
			resp := transport.simulateRequest(tt.request)
			
			if tt.expectError {
				if resp.Error == nil {
					t.Error("Expected error response")
				}
			} else {
				if resp.Error != nil {
					t.Errorf("Unexpected error: %v", resp.Error)
				}
				if tt.checkResponse != nil {
					tt.checkResponse(t, resp)
				}
			}
		})
	}
}

func TestHandleResourcesList(t *testing.T) {
	server := NewServer("test", "1.0")
	transport := &mockTransport{}
	server.SetTransport(transport)
	server.Start(context.Background())
	
	// Add resources
	resource1 := protocol.Resource{
		URI:      "file:///file1.txt",
		Name:     "file1.txt",
		MimeType: "text/plain",
	}
	resource2 := protocol.Resource{
		URI:  "https://example.com/data",
		Name: "Remote Data",
	}
	
	server.AddResource(resource1, &mockResourceHandler{})
	server.AddResource(resource2, &mockResourceHandler{})
	
	// Test resources/list
	req := &protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "resources/list",
	}
	
	resp := transport.simulateRequest(req)
	
	if resp.Error != nil {
		t.Errorf("Unexpected error: %v", resp.Error)
	}
	
	// Check result
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}
	
	resources, ok := result["resources"].([]interface{})
	if !ok {
		// Try as []protocol.Resource
		resourcesTyped, ok := result["resources"].([]protocol.Resource)
		if !ok {
			t.Fatal("Expected resources to be an array")
		}
		if len(resourcesTyped) != 2 {
			t.Errorf("Expected 2 resources, got %d", len(resourcesTyped))
		}
	} else {
		if len(resources) != 2 {
			t.Errorf("Expected 2 resources, got %d", len(resources))
		}
	}
}

func TestHandleResourcesRead(t *testing.T) {
	server := NewServer("test", "1.0")
	transport := &mockTransport{}
	server.SetTransport(transport)
	server.Start(context.Background())
	
	tests := []struct {
		name           string
		setupResources func()
		request        *protocol.JSONRPCRequest
		expectError    bool
		checkResponse  func(t *testing.T, resp *protocol.JSONRPCResponse)
	}{
		{
			name: "successful resource read",
			setupResources: func() {
				resource := protocol.Resource{
					URI:      "file:///test.txt",
					Name:     "test.txt",
					MimeType: "text/plain",
				}
				handler := &mockResourceHandler{
					content: []protocol.Content{
						protocol.NewContent("File contents"),
					},
				}
				server.AddResource(resource, handler)
			},
			request: &protocol.JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "resources/read",
				Params: map[string]interface{}{
					"uri": "file:///test.txt",
				},
			},
			expectError: false,
			checkResponse: func(t *testing.T, resp *protocol.JSONRPCResponse) {
				result, ok := resp.Result.(map[string]interface{})
				if !ok {
					t.Fatal("Expected result to be a map")
				}
				
				contents, ok := result["contents"].([]protocol.Content)
				if !ok {
					// Try unmarshaling
					contentsJSON, _ := json.Marshal(result["contents"])
					var c []protocol.Content
					json.Unmarshal(contentsJSON, &c)
					contents = c
				}
				
				if len(contents) != 1 {
					t.Errorf("Expected 1 content item, got %d", len(contents))
				}
			},
		},
		{
			name:           "resource not found",
			setupResources: func() {},
			request: &protocol.JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      2,
				Method:  "resources/read",
				Params: map[string]interface{}{
					"uri": "file:///nonexistent.txt",
				},
			},
			expectError: true,
		},
		{
			name: "handler error",
			setupResources: func() {
				resource := protocol.Resource{
					URI: "file:///error.txt",
				}
				handler := &mockResourceHandler{
					err: errors.New("read failed"),
				}
				server.AddResource(resource, handler)
			},
			request: &protocol.JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      3,
				Method:  "resources/read",
				Params: map[string]interface{}{
					"uri": "file:///error.txt",
				},
			},
			expectError: true,
		},
		{
			name:           "missing uri parameter",
			setupResources: func() {},
			request: &protocol.JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      4,
				Method:  "resources/read",
				Params:  map[string]interface{}{},
			},
			expectError: true,
		},
		{
			name:           "invalid params type",
			setupResources: func() {},
			request: &protocol.JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      5,
				Method:  "resources/read",
				Params:  "invalid",
			},
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear resources
			server.resources = make(map[string]*ResourceRegistration)
			
			// Setup resources
			tt.setupResources()
			
			// Execute request
			resp := transport.simulateRequest(tt.request)
			
			if tt.expectError {
				if resp.Error == nil {
					t.Error("Expected error response")
				}
			} else {
				if resp.Error != nil {
					t.Errorf("Unexpected error: %v", resp.Error)
				}
				if tt.checkResponse != nil {
					tt.checkResponse(t, resp)
				}
			}
		})
	}
}

func TestHandlePromptsList(t *testing.T) {
	server := NewServer("test", "1.0")
	transport := &mockTransport{}
	server.SetTransport(transport)
	server.Start(context.Background())
	
	// Add prompts
	prompt1 := protocol.Prompt{
		Name:        "greeting",
		Description: "Greeting prompt",
	}
	prompt2 := protocol.Prompt{
		Name: "farewell",
		Arguments: []protocol.PromptArgument{
			{Name: "name", Required: true},
		},
	}
	
	server.AddPrompt(prompt1, &mockPromptHandler{})
	server.AddPrompt(prompt2, &mockPromptHandler{})
	
	// Test prompts/list
	req := &protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "prompts/list",
	}
	
	resp := transport.simulateRequest(req)
	
	if resp.Error != nil {
		t.Errorf("Unexpected error: %v", resp.Error)
	}
	
	// Check result
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}
	
	prompts, ok := result["prompts"].([]interface{})
	if !ok {
		// Try as []protocol.Prompt
		promptsTyped, ok := result["prompts"].([]protocol.Prompt)
		if !ok {
			t.Fatal("Expected prompts to be an array")
		}
		if len(promptsTyped) != 2 {
			t.Errorf("Expected 2 prompts, got %d", len(promptsTyped))
		}
	} else {
		if len(prompts) != 2 {
			t.Errorf("Expected 2 prompts, got %d", len(prompts))
		}
	}
}

func TestHandlePromptsGet(t *testing.T) {
	server := NewServer("test", "1.0")
	transport := &mockTransport{}
	server.SetTransport(transport)
	server.Start(context.Background())
	
	tests := []struct {
		name          string
		setupPrompts  func()
		request       *protocol.JSONRPCRequest
		expectError   bool
		checkResponse func(t *testing.T, resp *protocol.JSONRPCResponse)
	}{
		{
			name: "successful prompt get",
			setupPrompts: func() {
				prompt := protocol.Prompt{
					Name: "greeting",
					Arguments: []protocol.PromptArgument{
						{Name: "name", Required: true},
					},
				}
				handler := &mockPromptHandler{
					content: []protocol.Content{
						protocol.NewContent("Hello, World!"),
					},
				}
				server.AddPrompt(prompt, handler)
			},
			request: &protocol.JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "prompts/get",
				Params: map[string]interface{}{
					"name": "greeting",
					"arguments": map[string]interface{}{
						"name": "World",
					},
				},
			},
			expectError: false,
			checkResponse: func(t *testing.T, resp *protocol.JSONRPCResponse) {
				result, ok := resp.Result.(map[string]interface{})
				if !ok {
					t.Fatal("Expected result to be a map")
				}
				
				messages, ok := result["messages"].([]protocol.Content)
				if !ok {
					// Try unmarshaling
					messagesJSON, _ := json.Marshal(result["messages"])
					var m []protocol.Content
					json.Unmarshal(messagesJSON, &m)
					messages = m
				}
				
				if len(messages) != 1 {
					t.Errorf("Expected 1 message, got %d", len(messages))
				}
			},
		},
		{
			name:         "prompt not found",
			setupPrompts: func() {},
			request: &protocol.JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      2,
				Method:  "prompts/get",
				Params: map[string]interface{}{
					"name": "nonexistent",
				},
			},
			expectError: true,
		},
		{
			name: "handler error",
			setupPrompts: func() {
				prompt := protocol.Prompt{Name: "error_prompt"}
				handler := &mockPromptHandler{
					err: errors.New("prompt failed"),
				}
				server.AddPrompt(prompt, handler)
			},
			request: &protocol.JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      3,
				Method:  "prompts/get",
				Params: map[string]interface{}{
					"name": "error_prompt",
				},
			},
			expectError: true,
		},
		{
			name:         "missing name parameter",
			setupPrompts: func() {},
			request: &protocol.JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      4,
				Method:  "prompts/get",
				Params:  map[string]interface{}{},
			},
			expectError: true,
		},
		{
			name:         "invalid params type",
			setupPrompts: func() {},
			request: &protocol.JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      5,
				Method:  "prompts/get",
				Params:  "invalid",
			},
			expectError: true,
		},
		{
			name: "without arguments parameter",
			setupPrompts: func() {
				prompt := protocol.Prompt{Name: "simple"}
				handler := &mockPromptHandler{
					content: []protocol.Content{
						protocol.NewContent("Simple prompt"),
					},
				}
				server.AddPrompt(prompt, handler)
			},
			request: &protocol.JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      6,
				Method:  "prompts/get",
				Params: map[string]interface{}{
					"name": "simple",
				},
			},
			expectError: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear prompts
			server.prompts = make(map[string]*PromptRegistration)
			
			// Setup prompts
			tt.setupPrompts()
			
			// Execute request
			resp := transport.simulateRequest(tt.request)
			
			if tt.expectError {
				if resp.Error == nil {
					t.Error("Expected error response")
				}
			} else {
				if resp.Error != nil {
					t.Errorf("Unexpected error: %v", resp.Error)
				}
				if tt.checkResponse != nil {
					tt.checkResponse(t, resp)
				}
			}
		})
	}
}

func TestHandleUnknownMethod(t *testing.T) {
	server := NewServer("test", "1.0")
	transport := &mockTransport{}
	server.SetTransport(transport)
	server.Start(context.Background())
	
	req := &protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "unknown/method",
	}
	
	resp := transport.simulateRequest(req)
	
	if resp.Error == nil {
		t.Error("Expected error for unknown method")
	}
	
	if resp.Error.Code != protocol.MethodNotFound {
		t.Errorf("Expected MethodNotFound error code, got %d", resp.Error.Code)
	}
}

func TestParseParams(t *testing.T) {
	tests := []struct {
		name        string
		params      interface{}
		target      interface{}
		expectError bool
	}{
		{
			name: "parse map to struct",
			params: map[string]interface{}{
				"name": "test",
				"age":  30,
			},
			target: &struct {
				Name string `json:"name"`
				Age  int    `json:"age"`
			}{},
			expectError: false,
		},
		{
			name:        "parse nil params",
			params:      nil,
			target:      &struct{}{},
			expectError: false,
		},
		{
			name:   "parse array",
			params: []interface{}{"a", "b", "c"},
			target: &[]string{},
			expectError: false,
		},
		{
			name: "parse incompatible types",
			params: map[string]interface{}{
				"count": "not a number",
			},
			target: &struct {
				Count int `json:"count"`
			}{},
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parseParams(tt.params, tt.target)
			
			if (err != nil) != tt.expectError {
				t.Errorf("parseParams() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestConcurrentAccess(t *testing.T) {
	server := NewServer("test", "1.0")
	
	// Simulate concurrent tool additions
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			tool := protocol.Tool{
				Name: fmt.Sprintf("tool_%d", i),
			}
			server.AddTool(tool, &mockToolHandler{})
		}(i)
	}
	
	// Simulate concurrent resource additions
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resource := protocol.Resource{
				URI: fmt.Sprintf("file:///%d.txt", i),
			}
			server.AddResource(resource, &mockResourceHandler{})
		}(i)
	}
	
	// Simulate concurrent prompt additions
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			prompt := protocol.Prompt{
				Name: fmt.Sprintf("prompt_%d", i),
			}
			server.AddPrompt(prompt, &mockPromptHandler{})
		}(i)
	}
	
	wg.Wait()
	
	// Verify all items were added
	if len(server.tools) != 10 {
		t.Errorf("Expected 10 tools, got %d", len(server.tools))
	}
	if len(server.resources) != 10 {
		t.Errorf("Expected 10 resources, got %d", len(server.resources))
	}
	if len(server.prompts) != 10 {
		t.Errorf("Expected 10 prompts, got %d", len(server.prompts))
	}
}

func TestIsInitialized(t *testing.T) {
	server := NewServer("test", "1.0")
	
	// Should not be initialized initially
	if server.IsInitialized() {
		t.Error("Expected server to not be initialized initially")
	}
	
	// Simulate initialization
	transport := &mockTransport{}
	server.SetTransport(transport)
	server.Start(context.Background())
	
	initReq := &protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: protocol.InitializeRequest{
			ProtocolVersion: protocol.Version,
			Capabilities:    protocol.ClientCapabilities{},
			ClientInfo: protocol.ClientInfo{
				Name:    "test-client",
				Version: "1.0",
			},
		},
	}
	
	transport.simulateRequest(initReq)
	
	// Should be initialized after initialize request
	if !server.IsInitialized() {
		t.Error("Expected server to be initialized after initialize request")
	}
}

// Benchmark tests
func BenchmarkAddTool(b *testing.B) {
	server := NewServer("bench", "1.0")
	handler := &mockToolHandler{}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool := protocol.Tool{
			Name: fmt.Sprintf("tool_%d", i),
		}
		server.AddTool(tool, handler)
	}
}

func BenchmarkHandleRequest(b *testing.B) {
	server := NewServer("bench", "1.0")
	
	// Add a simple tool
	tool := protocol.Tool{Name: "bench_tool"}
	handler := &mockToolHandler{result: "result"}
	server.AddTool(tool, handler)
	
	req := &protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.HandleRequest(ctx, req)
	}
}

func BenchmarkConcurrentRequests(b *testing.B) {
	server := NewServer("bench", "1.0")
	
	// Add tools
	for i := 0; i < 10; i++ {
		tool := protocol.Tool{Name: fmt.Sprintf("tool_%d", i)}
		handler := &mockToolHandler{result: "result"}
		server.AddTool(tool, handler)
	}
	
	req := &protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			server.HandleRequest(ctx, req)
		}
	})
}

// Test edge cases
func TestEdgeCases(t *testing.T) {
	t.Run("nil handler registration", func(t *testing.T) {
		server := NewServer("test", "1.0")
		
		// Should not panic with nil handler
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Unexpected panic: %v", r)
			}
		}()
		
		tool := protocol.Tool{Name: "nil_handler"}
		server.AddTool(tool, nil)
		
		// Tool should be registered even with nil handler
		if _, exists := server.tools["nil_handler"]; !exists {
			t.Error("Expected tool to be registered")
		}
	})
	
	t.Run("overwrite existing registrations", func(t *testing.T) {
		server := NewServer("test", "1.0")
		
		// Add initial tool
		tool1 := protocol.Tool{
			Name:        "duplicate",
			Description: "First version",
		}
		handler1 := &mockToolHandler{result: "v1"}
		server.AddTool(tool1, handler1)
		
		// Overwrite with new version
		tool2 := protocol.Tool{
			Name:        "duplicate",
			Description: "Second version",
		}
		handler2 := &mockToolHandler{result: "v2"}
		server.AddTool(tool2, handler2)
		
		// Should have the second version
		reg, exists := server.tools["duplicate"]
		if !exists {
			t.Fatal("Expected tool to exist")
		}
		if reg.Tool.Description != "Second version" {
			t.Error("Expected tool to be overwritten")
		}
	})
	
	t.Run("request timeout handling", func(t *testing.T) {
		server := NewServer("test", "1.0")
		transport := &mockTransport{}
		server.SetTransport(transport)
		server.Start(context.Background())
		
		// Add a slow tool
		tool := protocol.Tool{Name: "slow_tool"}
		handler := &mockToolHandler{
			result: "should timeout",
		}
		server.AddTool(tool, handler)
		
		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		
		req := &protocol.JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      1,
			Method:  "tools/call",
			Params: protocol.ToolCallRequest{
				Name: "slow_tool",
			},
		}
		
		// Simulate request with timeout context
		resp := server.HandleRequest(ctx, req)
		
		// Should still get a response (not panic)
		if resp == nil {
			t.Error("Expected response even with timeout")
		}
	})
}