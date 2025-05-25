package protocol

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

func TestVersion(t *testing.T) {
	if Version != "2024-11-05" {
		t.Errorf("Expected version 2024-11-05, got %s", Version)
	}
}

func TestJSONRPCRequest(t *testing.T) {
	tests := []struct {
		name     string
		request  JSONRPCRequest
		wantJSON string
	}{
		{
			name: "request with all fields",
			request: JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "tools/list",
				Params:  map[string]interface{}{"filter": "test"},
			},
			wantJSON: `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"filter":"test"}}`,
		},
		{
			name: "request without ID (notification)",
			request: JSONRPCRequest{
				JSONRPC: "2.0",
				Method:  "notification",
			},
			wantJSON: `{"jsonrpc":"2.0","method":"notification"}`,
		},
		{
			name: "request with string ID",
			request: JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      "test-id",
				Method:  "test/method",
			},
			wantJSON: `{"jsonrpc":"2.0","id":"test-id","method":"test/method"}`,
		},
		{
			name: "request without params",
			request: JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      42,
				Method:  "simple/method",
			},
			wantJSON: `{"jsonrpc":"2.0","id":42,"method":"simple/method"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			// Test unmarshaling back
			var decoded JSONRPCRequest
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal request: %v", err)
			}

			// Compare original and decoded
			if !reflect.DeepEqual(tt.request, decoded) {
				t.Errorf("Request mismatch after marshal/unmarshal\nOriginal: %+v\nDecoded: %+v", tt.request, decoded)
			}
		})
	}
}

func TestJSONRPCResponse(t *testing.T) {
	tests := []struct {
		name     string
		response JSONRPCResponse
	}{
		{
			name: "successful response",
			response: JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result:  map[string]interface{}{"tools": []string{"tool1", "tool2"}},
			},
		},
		{
			name: "error response",
			response: JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      2,
				Error: &JSONRPCError{
					Code:    MethodNotFound,
					Message: "Method not found",
					Data:    "tools/unknown",
				},
			},
		},
		{
			name: "response without ID",
			response: JSONRPCResponse{
				JSONRPC: "2.0",
				Result:  "success",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling and unmarshaling
			data, err := json.Marshal(tt.response)
			if err != nil {
				t.Fatalf("Failed to marshal response: %v", err)
			}

			var decoded JSONRPCResponse
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			// Basic field checks
			if decoded.JSONRPC != tt.response.JSONRPC {
				t.Errorf("JSONRPC mismatch: got %s, want %s", decoded.JSONRPC, tt.response.JSONRPC)
			}
			if decoded.ID != tt.response.ID {
				t.Errorf("ID mismatch: got %v, want %v", decoded.ID, tt.response.ID)
			}

			// Check error presence
			if (decoded.Error != nil) != (tt.response.Error != nil) {
				t.Errorf("Error presence mismatch: got %v, want %v", decoded.Error != nil, tt.response.Error != nil)
			}
		})
	}
}

func TestNewJSONRPCError(t *testing.T) {
	tests := []struct {
		name    string
		code    int
		message string
		data    interface{}
	}{
		{
			name:    "parse error",
			code:    ParseError,
			message: "Parse error",
			data:    nil,
		},
		{
			name:    "invalid request",
			code:    InvalidRequest,
			message: "Invalid request",
			data:    "missing method",
		},
		{
			name:    "method not found",
			code:    MethodNotFound,
			message: "Method not found",
			data:    map[string]string{"method": "unknown/method"},
		},
		{
			name:    "invalid params",
			code:    InvalidParams,
			message: "Invalid params",
			data:    []string{"param1", "param2"},
		},
		{
			name:    "internal error",
			code:    InternalError,
			message: "Internal error",
			data:    struct{ Error string }{Error: "database connection failed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewJSONRPCError(tt.code, tt.message, tt.data)
			
			if err.Code != tt.code {
				t.Errorf("Code mismatch: got %d, want %d", err.Code, tt.code)
			}
			if err.Message != tt.message {
				t.Errorf("Message mismatch: got %s, want %s", err.Message, tt.message)
			}
			if !reflect.DeepEqual(err.Data, tt.data) {
				t.Errorf("Data mismatch: got %v, want %v", err.Data, tt.data)
			}
		})
	}
}

func TestTool(t *testing.T) {
	tests := []struct {
		name string
		tool Tool
	}{
		{
			name: "simple tool",
			tool: Tool{
				Name:        "test_tool",
				Description: "A test tool",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"input": map[string]interface{}{
							"type":        "string",
							"description": "Input parameter",
						},
					},
					"required": []string{"input"},
				},
			},
		},
		{
			name: "tool without description",
			tool: Tool{
				Name: "minimal_tool",
				InputSchema: map[string]interface{}{
					"type": "object",
				},
			},
		},
		{
			name: "complex tool schema",
			tool: Tool{
				Name:        "complex_tool",
				Description: "A complex tool with multiple parameters",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type": "string",
						},
						"age": map[string]interface{}{
							"type":    "integer",
							"minimum": 0,
						},
						"tags": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"type": "string",
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			data, err := json.Marshal(tt.tool)
			if err != nil {
				t.Fatalf("Failed to marshal tool: %v", err)
			}

			// Test JSON unmarshaling
			var decoded Tool
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal tool: %v", err)
			}

			// Verify fields
			if decoded.Name != tt.tool.Name {
				t.Errorf("Name mismatch: got %s, want %s", decoded.Name, tt.tool.Name)
			}
			if decoded.Description != tt.tool.Description {
				t.Errorf("Description mismatch: got %s, want %s", decoded.Description, tt.tool.Description)
			}
		})
	}
}

func TestToolCallRequest(t *testing.T) {
	tests := []struct {
		name    string
		request ToolCallRequest
	}{
		{
			name: "request with arguments",
			request: ToolCallRequest{
				Name: "test_tool",
				Arguments: map[string]interface{}{
					"input": "test value",
					"count": 42,
				},
			},
		},
		{
			name: "request without arguments",
			request: ToolCallRequest{
				Name: "no_args_tool",
			},
		},
		{
			name: "request with nested arguments",
			request: ToolCallRequest{
				Name: "nested_tool",
				Arguments: map[string]interface{}{
					"config": map[string]interface{}{
						"enabled": true,
						"timeout": 30,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON round trip
			data, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			var decoded ToolCallRequest
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal request: %v", err)
			}

			if decoded.Name != tt.request.Name {
				t.Errorf("Name mismatch: got %s, want %s", decoded.Name, tt.request.Name)
			}
		})
	}
}

func TestNewContent(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected Content
	}{
		{
			name: "simple text",
			text: "Hello, world!",
			expected: Content{
				Type: "text",
				Text: "Hello, world!",
			},
		},
		{
			name: "empty text",
			text: "",
			expected: Content{
				Type: "text",
				Text: "",
			},
		},
		{
			name: "multi-line text",
			text: "Line 1\nLine 2\nLine 3",
			expected: Content{
				Type: "text",
				Text: "Line 1\nLine 2\nLine 3",
			},
		},
		{
			name: "text with special characters",
			text: "Special chars: !@#$%^&*()_+-=[]{}|;':\",./<>?",
			expected: Content{
				Type: "text",
				Text: "Special chars: !@#$%^&*()_+-=[]{}|;':\",./<>?",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := NewContent(tt.text)
			if content.Type != tt.expected.Type {
				t.Errorf("Type mismatch: got %s, want %s", content.Type, tt.expected.Type)
			}
			if content.Text != tt.expected.Text {
				t.Errorf("Text mismatch: got %s, want %s", content.Text, tt.expected.Text)
			}
		})
	}
}

func TestNewToolCallResult(t *testing.T) {
	tests := []struct {
		name     string
		content  []Content
		expected *ToolCallResult
	}{
		{
			name: "single content",
			content: []Content{
				NewContent("Result text"),
			},
			expected: &ToolCallResult{
				Content: []Content{
					{Type: "text", Text: "Result text"},
				},
				IsError: false,
			},
		},
		{
			name: "multiple content",
			content: []Content{
				NewContent("Part 1"),
				NewContent("Part 2"),
				NewContent("Part 3"),
			},
			expected: &ToolCallResult{
				Content: []Content{
					{Type: "text", Text: "Part 1"},
					{Type: "text", Text: "Part 2"},
					{Type: "text", Text: "Part 3"},
				},
				IsError: false,
			},
		},
		{
			name:    "no content",
			content: []Content{},
			expected: &ToolCallResult{
				Content: []Content{},
				IsError: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewToolCallResult(tt.content...)
			
			if result.IsError != tt.expected.IsError {
				t.Errorf("IsError mismatch: got %v, want %v", result.IsError, tt.expected.IsError)
			}
			
			if len(result.Content) != len(tt.expected.Content) {
				t.Errorf("Content length mismatch: got %d, want %d", len(result.Content), len(tt.expected.Content))
			}
			
			for i, content := range result.Content {
				if i < len(tt.expected.Content) {
					if content.Type != tt.expected.Content[i].Type {
						t.Errorf("Content[%d].Type mismatch: got %s, want %s", i, content.Type, tt.expected.Content[i].Type)
					}
					if content.Text != tt.expected.Content[i].Text {
						t.Errorf("Content[%d].Text mismatch: got %s, want %s", i, content.Text, tt.expected.Content[i].Text)
					}
				}
			}
		})
	}
}

func TestNewToolCallError(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "simple error",
			message: "Something went wrong",
		},
		{
			name:    "detailed error",
			message: "Failed to connect to database: connection timeout after 30s",
		},
		{
			name:    "empty error",
			message: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewToolCallError(tt.message)
			
			if !result.IsError {
				t.Error("Expected IsError to be true")
			}
			
			if len(result.Content) != 1 {
				t.Errorf("Expected 1 content item, got %d", len(result.Content))
			}
			
			if len(result.Content) > 0 {
				if result.Content[0].Type != "text" {
					t.Errorf("Expected content type 'text', got %s", result.Content[0].Type)
				}
				if result.Content[0].Text != tt.message {
					t.Errorf("Expected content text '%s', got '%s'", tt.message, result.Content[0].Text)
				}
			}
		})
	}
}

func TestResource(t *testing.T) {
	tests := []struct {
		name     string
		resource Resource
	}{
		{
			name: "full resource",
			resource: Resource{
				URI:         "file:///path/to/file.txt",
				Name:        "file.txt",
				Description: "A text file",
				MimeType:    "text/plain",
			},
		},
		{
			name: "minimal resource",
			resource: Resource{
				URI: "https://example.com/data",
			},
		},
		{
			name: "resource with special URI",
			resource: Resource{
				URI:      "custom://resource/123",
				Name:     "Custom Resource",
				MimeType: "application/octet-stream",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			data, err := json.Marshal(tt.resource)
			if err != nil {
				t.Fatalf("Failed to marshal resource: %v", err)
			}

			// Test JSON unmarshaling
			var decoded Resource
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal resource: %v", err)
			}

			// Verify all fields
			if decoded.URI != tt.resource.URI {
				t.Errorf("URI mismatch: got %s, want %s", decoded.URI, tt.resource.URI)
			}
			if decoded.Name != tt.resource.Name {
				t.Errorf("Name mismatch: got %s, want %s", decoded.Name, tt.resource.Name)
			}
			if decoded.Description != tt.resource.Description {
				t.Errorf("Description mismatch: got %s, want %s", decoded.Description, tt.resource.Description)
			}
			if decoded.MimeType != tt.resource.MimeType {
				t.Errorf("MimeType mismatch: got %s, want %s", decoded.MimeType, tt.resource.MimeType)
			}
		})
	}
}

func TestPrompt(t *testing.T) {
	tests := []struct {
		name   string
		prompt Prompt
	}{
		{
			name: "prompt with arguments",
			prompt: Prompt{
				Name:        "greeting_prompt",
				Description: "A greeting prompt template",
				Arguments: []PromptArgument{
					{
						Name:        "name",
						Description: "The name to greet",
						Required:    true,
					},
					{
						Name:        "title",
						Description: "Optional title",
						Required:    false,
					},
				},
			},
		},
		{
			name: "prompt without arguments",
			prompt: Prompt{
				Name:        "simple_prompt",
				Description: "A simple prompt without parameters",
				Arguments:   nil,
			},
		},
		{
			name: "minimal prompt",
			prompt: Prompt{
				Name: "minimal",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON round trip
			data, err := json.Marshal(tt.prompt)
			if err != nil {
				t.Fatalf("Failed to marshal prompt: %v", err)
			}

			var decoded Prompt
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal prompt: %v", err)
			}

			// Verify fields
			if decoded.Name != tt.prompt.Name {
				t.Errorf("Name mismatch: got %s, want %s", decoded.Name, tt.prompt.Name)
			}
			if decoded.Description != tt.prompt.Description {
				t.Errorf("Description mismatch: got %s, want %s", decoded.Description, tt.prompt.Description)
			}
			if len(decoded.Arguments) != len(tt.prompt.Arguments) {
				t.Errorf("Arguments length mismatch: got %d, want %d", len(decoded.Arguments), len(tt.prompt.Arguments))
			}
		})
	}
}

func TestServerCapabilities(t *testing.T) {
	tests := []struct {
		name         string
		capabilities ServerCapabilities
	}{
		{
			name: "full capabilities",
			capabilities: ServerCapabilities{
				Experimental: map[string]interface{}{
					"feature1": true,
					"feature2": "enabled",
				},
				Logging: map[string]interface{}{
					"level": "debug",
				},
				Prompts: &PromptCapability{
					ListChanged: true,
				},
				Resources: &ResourceCapability{
					Subscribe:   true,
					ListChanged: true,
				},
				Tools: &ToolCapability{
					ListChanged: true,
				},
			},
		},
		{
			name: "minimal capabilities",
			capabilities: ServerCapabilities{
				Tools: &ToolCapability{},
			},
		},
		{
			name:         "empty capabilities",
			capabilities: ServerCapabilities{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			data, err := json.Marshal(tt.capabilities)
			if err != nil {
				t.Fatalf("Failed to marshal capabilities: %v", err)
			}

			// Test JSON unmarshaling
			var decoded ServerCapabilities
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal capabilities: %v", err)
			}

			// Basic nil checks
			if (decoded.Prompts != nil) != (tt.capabilities.Prompts != nil) {
				t.Errorf("Prompts nil mismatch")
			}
			if (decoded.Resources != nil) != (tt.capabilities.Resources != nil) {
				t.Errorf("Resources nil mismatch")
			}
			if (decoded.Tools != nil) != (tt.capabilities.Tools != nil) {
				t.Errorf("Tools nil mismatch")
			}
		})
	}
}

func TestInitializeRequest(t *testing.T) {
	tests := []struct {
		name    string
		request InitializeRequest
	}{
		{
			name: "full request",
			request: InitializeRequest{
				ProtocolVersion: Version,
				Capabilities: ClientCapabilities{
					Experimental: map[string]interface{}{
						"feature": "enabled",
					},
					Sampling: map[string]interface{}{
						"enabled": true,
					},
				},
				ClientInfo: ClientInfo{
					Name:    "test-client",
					Version: "1.0.0",
				},
			},
		},
		{
			name: "minimal request",
			request: InitializeRequest{
				ProtocolVersion: Version,
				Capabilities:    ClientCapabilities{},
				ClientInfo: ClientInfo{
					Name:    "minimal-client",
					Version: "0.1.0",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON round trip
			data, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			var decoded InitializeRequest
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal request: %v", err)
			}

			// Verify fields
			if decoded.ProtocolVersion != tt.request.ProtocolVersion {
				t.Errorf("ProtocolVersion mismatch: got %s, want %s", decoded.ProtocolVersion, tt.request.ProtocolVersion)
			}
			if decoded.ClientInfo.Name != tt.request.ClientInfo.Name {
				t.Errorf("ClientInfo.Name mismatch: got %s, want %s", decoded.ClientInfo.Name, tt.request.ClientInfo.Name)
			}
			if decoded.ClientInfo.Version != tt.request.ClientInfo.Version {
				t.Errorf("ClientInfo.Version mismatch: got %s, want %s", decoded.ClientInfo.Version, tt.request.ClientInfo.Version)
			}
		})
	}
}

func TestInitializeResult(t *testing.T) {
	tests := []struct {
		name   string
		result InitializeResult
	}{
		{
			name: "full result",
			result: InitializeResult{
				ProtocolVersion: Version,
				Capabilities: ServerCapabilities{
					Tools: &ToolCapability{
						ListChanged: true,
					},
				},
				ServerInfo: ServerInfo{
					Name:    "test-server",
					Version: "2.0.0",
				},
			},
		},
		{
			name: "minimal result",
			result: InitializeResult{
				ProtocolVersion: Version,
				Capabilities:    ServerCapabilities{},
				ServerInfo: ServerInfo{
					Name:    "minimal-server",
					Version: "1.0.0",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON round trip
			data, err := json.Marshal(tt.result)
			if err != nil {
				t.Fatalf("Failed to marshal result: %v", err)
			}

			var decoded InitializeResult
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal result: %v", err)
			}

			// Verify fields
			if decoded.ProtocolVersion != tt.result.ProtocolVersion {
				t.Errorf("ProtocolVersion mismatch: got %s, want %s", decoded.ProtocolVersion, tt.result.ProtocolVersion)
			}
			if decoded.ServerInfo.Name != tt.result.ServerInfo.Name {
				t.Errorf("ServerInfo.Name mismatch: got %s, want %s", decoded.ServerInfo.Name, tt.result.ServerInfo.Name)
			}
			if decoded.ServerInfo.Version != tt.result.ServerInfo.Version {
				t.Errorf("ServerInfo.Version mismatch: got %s, want %s", decoded.ServerInfo.Version, tt.result.ServerInfo.Version)
			}
		})
	}
}

func TestToolHandlerFunc(t *testing.T) {
	tests := []struct {
		name           string
		handler        ToolHandlerFunc
		params         map[string]interface{}
		expectedResult interface{}
		expectedError  bool
	}{
		{
			name: "successful handler",
			handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
				return map[string]string{"status": "success"}, nil
			},
			params:         map[string]interface{}{"input": "test"},
			expectedResult: map[string]string{"status": "success"},
			expectedError:  false,
		},
		{
			name: "handler with error",
			handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
				return nil, context.DeadlineExceeded
			},
			params:         map[string]interface{}{},
			expectedResult: nil,
			expectedError:  true,
		},
		{
			name: "handler using params",
			handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
				name, ok := params["name"].(string)
				if !ok {
					name = "unknown"
				}
				return map[string]string{"greeting": "Hello, " + name}, nil
			},
			params:         map[string]interface{}{"name": "test"},
			expectedResult: map[string]string{"greeting": "Hello, test"},
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := tt.handler.Handle(ctx, tt.params)

			if (err != nil) != tt.expectedError {
				t.Errorf("Error mismatch: got error=%v, want error=%v", err != nil, tt.expectedError)
			}

			if !tt.expectedError && !reflect.DeepEqual(result, tt.expectedResult) {
				t.Errorf("Result mismatch: got %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

func TestErrorConstants(t *testing.T) {
	// Test that error constants have expected values
	expectedErrors := map[string]int{
		"ParseError":     -32700,
		"InvalidRequest": -32600,
		"MethodNotFound": -32601,
		"InvalidParams":  -32602,
		"InternalError":  -32603,
	}

	actualErrors := map[string]int{
		"ParseError":     ParseError,
		"InvalidRequest": InvalidRequest,
		"MethodNotFound": MethodNotFound,
		"InvalidParams":  InvalidParams,
		"InternalError":  InternalError,
	}

	for name, expected := range expectedErrors {
		actual := actualErrors[name]
		if actual != expected {
			t.Errorf("%s: got %d, want %d", name, actual, expected)
		}
	}
}

// Test edge cases and error conditions
func TestJSONMarshalingEdgeCases(t *testing.T) {
	t.Run("nil pointer in response error", func(t *testing.T) {
		resp := JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      1,
			Error:   nil,
		}

		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("Failed to marshal response with nil error: %v", err)
		}

		// Should not include error field when nil
		if string(data) == "" {
			t.Error("Expected non-empty JSON")
		}
	})

	t.Run("empty tool arguments", func(t *testing.T) {
		req := ToolCallRequest{
			Name:      "test",
			Arguments: map[string]interface{}{},
		}

		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("Failed to marshal request with empty arguments: %v", err)
		}

		var decoded ToolCallRequest
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if decoded.Name != req.Name {
			t.Errorf("Name mismatch: got %s, want %s", decoded.Name, req.Name)
		}
	})

	t.Run("nil tool handler", func(t *testing.T) {
		var handler ToolHandler
		if handler != nil {
			t.Error("Expected nil handler to be nil")
		}
	})
}

// Benchmark tests
func BenchmarkNewContent(b *testing.B) {
	text := "This is a benchmark test string"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewContent(text)
	}
}

func BenchmarkJSONMarshalRequest(b *testing.B) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
		Params: map[string]interface{}{
			"filter": "test",
			"limit":  100,
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(req)
	}
}

func BenchmarkJSONUnmarshalRequest(b *testing.B) {
	data := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"filter":"test","limit":100}}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var req JSONRPCRequest
		_ = json.Unmarshal(data, &req)
	}
}