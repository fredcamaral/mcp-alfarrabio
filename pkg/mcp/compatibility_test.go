package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mcp-memory/pkg/mcp/protocol"
	"mcp-memory/pkg/mcp/testutil"
	"testing"
)

// TestMCPProtocolCompliance tests compliance with MCP protocol specification
func TestMCPProtocolCompliance(t *testing.T) {
	srv := testutil.NewServerBuilder("compliance-server", "1.0.0").
		WithSimpleTool("test_tool", "test result").
		WithResource("test://resource", "test resource", "resource content").
		WithPrompt("test_prompt", "prompt content").
		WithAutoStart().
		Build()
	
	defer srv.Stop()
	
	client := srv.Client()
	ctx := context.Background()
	
	t.Run("protocol_version", func(t *testing.T) {
		result, err := client.Initialize(ctx, "test-client", "1.0.0")
		if err != nil {
			t.Fatalf("Failed to initialize: %v", err)
		}
		
		// Must return the exact protocol version
		if result.ProtocolVersion != protocol.Version {
			t.Errorf("Expected protocol version %s, got %s", protocol.Version, result.ProtocolVersion)
		}
	})
	
	t.Run("jsonrpc_version", func(t *testing.T) {
		// All requests and responses must have jsonrpc: "2.0"
		id, err := client.SendRequest("tools/list", nil)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		
		resp, err := client.WaitForResponse(ctx, id)
		if err != nil {
			t.Fatalf("Failed to get response: %v", err)
		}
		
		if resp.JSONRPC != "2.0" {
			t.Errorf("Expected JSONRPC version 2.0, got %s", resp.JSONRPC)
		}
	})
	
	t.Run("id_handling", func(t *testing.T) {
		// Test different ID types
		testCases := []struct {
			name string
			id   interface{}
		}{
			{"integer_id", 123},
			{"string_id", "test-id-456"},
			{"float_id", 789.0},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Manually create request with specific ID type
				req := protocol.JSONRPCRequest{
					JSONRPC: "2.0",
					ID:      tc.id,
					Method:  "tools/list",
				}
				
				// Send via raw encoder
				input := srv.Client().GetInput().(*bytes.Buffer)
				encoder := json.NewEncoder(input)
				if err := encoder.Encode(req); err != nil {
					t.Fatalf("Failed to encode request: %v", err)
				}
				
				// Get response
				resp, err := client.GetNextResponse(ctx)
				if err != nil {
					t.Fatalf("Failed to get response: %v", err)
				}
				
				// ID in response must match request
				if fmt.Sprintf("%v", resp.ID) != fmt.Sprintf("%v", tc.id) {
					t.Errorf("ID mismatch: sent %v, got %v", tc.id, resp.ID)
				}
			})
		}
	})
}

// TestJSONRPCCompliance tests JSON-RPC 2.0 compliance
func TestJSONRPCCompliance(t *testing.T) {
	srv := testutil.NewServerBuilder("jsonrpc-server", "1.0.0").
		WithSimpleTool("echo", "echoed").
		WithAutoStart().
		Build()
	
	defer srv.Stop()
	
	client := srv.Client()
	ctx := context.Background()
	
	// Initialize first
	if _, err := client.Initialize(ctx, "jsonrpc-client", "1.0.0"); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	
	t.Run("error_codes", func(t *testing.T) {
		testCases := []struct {
			name         string
			request      string
			expectedCode int
		}{
			{
				name:         "parse_error",
				request:      `{invalid json`,
				expectedCode: protocol.ParseError,
			},
			{
				name:         "invalid_request",
				request:      `{"jsonrpc": "2.0"}`, // Missing method
				expectedCode: protocol.InvalidRequest,
			},
			{
				name:         "method_not_found",
				request:      `{"jsonrpc": "2.0", "id": 1, "method": "nonexistent"}`,
				expectedCode: protocol.MethodNotFound,
			},
			{
				name:         "invalid_params",
				request:      `{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": "string"}`,
				expectedCode: protocol.InvalidParams,
			},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Send raw request
				input := srv.Client().GetInput().(*bytes.Buffer)
				input.WriteString(tc.request + "\n")
				
				// Get response
				resp, err := client.GetNextResponse(ctx)
				if err != nil {
					t.Fatalf("Failed to get response: %v", err)
				}
				
				if resp.Error == nil {
					t.Fatal("Expected error response")
				}
				
				if resp.Error.Code != tc.expectedCode {
					t.Errorf("Expected error code %d, got %d", tc.expectedCode, resp.Error.Code)
				}
			})
		}
	})
	
	t.Run("batch_requests_not_supported", func(t *testing.T) {
		// MCP doesn't support batch requests, so array should result in parse error
		batchRequest := `[{"jsonrpc":"2.0","id":1,"method":"tools/list"},{"jsonrpc":"2.0","id":2,"method":"tools/list"}]`
		
		input := srv.Client().GetInput().(*bytes.Buffer)
		input.WriteString(batchRequest + "\n")
		
		resp, err := client.GetNextResponse(ctx)
		if err != nil {
			t.Fatalf("Failed to get response: %v", err)
		}
		
		if resp.Error == nil {
			t.Fatal("Expected error for batch request")
		}
		
		if resp.Error.Code != protocol.ParseError {
			t.Errorf("Expected parse error for batch request, got code %d", resp.Error.Code)
		}
	})
}

// TestMethodSignatures tests that all methods follow correct signatures
func TestMethodSignatures(t *testing.T) {
	srv := testutil.NewServerBuilder("signature-server", "1.0.0").
		WithSimpleTool("test_tool", "result").
		WithResource("test://res", "res", "content").
		WithPrompt("test_prompt", "prompt").
		WithAutoStart().
		Build()
	
	defer srv.Stop()
	
	client := srv.Client()
	ctx := context.Background()
	
	// Initialize
	if _, err := client.Initialize(ctx, "signature-client", "1.0.0"); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	
	testCases := []struct {
		method         string
		params         interface{}
		validateResult func(t *testing.T, result interface{})
	}{
		{
			method: "initialize",
			params: protocol.InitializeRequest{
				ProtocolVersion: protocol.Version,
				Capabilities:    protocol.ClientCapabilities{},
				ClientInfo: protocol.ClientInfo{
					Name:    "test",
					Version: "1.0",
				},
			},
			validateResult: func(t *testing.T, result interface{}) {
				data, _ := json.Marshal(result)
				var initResult protocol.InitializeResult
				if err := json.Unmarshal(data, &initResult); err != nil {
					t.Errorf("Result doesn't match InitializeResult schema: %v", err)
				}
			},
		},
		{
			method: "tools/list",
			params: nil,
			validateResult: func(t *testing.T, result interface{}) {
				resultMap, ok := result.(map[string]interface{})
				if !ok {
					t.Fatal("Result is not a map")
				}
				
				tools, ok := resultMap["tools"]
				if !ok {
					t.Fatal("Result missing 'tools' field")
				}
				
				// Should be an array
				_, ok = tools.([]interface{})
				if !ok {
					t.Fatal("'tools' is not an array")
				}
			},
		},
		{
			method: "tools/call",
			params: protocol.ToolCallRequest{
				Name:      "test_tool",
				Arguments: map[string]interface{}{},
			},
			validateResult: func(t *testing.T, result interface{}) {
				data, _ := json.Marshal(result)
				var toolResult protocol.ToolCallResult
				if err := json.Unmarshal(data, &toolResult); err != nil {
					t.Errorf("Result doesn't match ToolCallResult schema: %v", err)
				}
			},
		},
		{
			method: "resources/list",
			params: nil,
			validateResult: func(t *testing.T, result interface{}) {
				resultMap, ok := result.(map[string]interface{})
				if !ok {
					t.Fatal("Result is not a map")
				}
				
				resources, ok := resultMap["resources"]
				if !ok {
					t.Fatal("Result missing 'resources' field")
				}
				
				// Should be an array
				_, ok = resources.([]interface{})
				if !ok {
					t.Fatal("'resources' is not an array")
				}
			},
		},
		{
			method: "resources/read",
			params: map[string]interface{}{
				"uri": "test://res",
			},
			validateResult: func(t *testing.T, result interface{}) {
				resultMap, ok := result.(map[string]interface{})
				if !ok {
					t.Fatal("Result is not a map")
				}
				
				contents, ok := resultMap["contents"]
				if !ok {
					t.Fatal("Result missing 'contents' field")
				}
				
				// Should be an array
				_, ok = contents.([]interface{})
				if !ok {
					t.Fatal("'contents' is not an array")
				}
			},
		},
		{
			method: "prompts/list",
			params: nil,
			validateResult: func(t *testing.T, result interface{}) {
				resultMap, ok := result.(map[string]interface{})
				if !ok {
					t.Fatal("Result is not a map")
				}
				
				prompts, ok := resultMap["prompts"]
				if !ok {
					t.Fatal("Result missing 'prompts' field")
				}
				
				// Should be an array
				_, ok = prompts.([]interface{})
				if !ok {
					t.Fatal("'prompts' is not an array")
				}
			},
		},
		{
			method: "prompts/get",
			params: map[string]interface{}{
				"name":      "test_prompt",
				"arguments": map[string]interface{}{},
			},
			validateResult: func(t *testing.T, result interface{}) {
				resultMap, ok := result.(map[string]interface{})
				if !ok {
					t.Fatal("Result is not a map")
				}
				
				messages, ok := resultMap["messages"]
				if !ok {
					t.Fatal("Result missing 'messages' field")
				}
				
				// Should be an array
				_, ok = messages.([]interface{})
				if !ok {
					t.Fatal("'messages' is not an array")
				}
			},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.method, func(t *testing.T) {
			id, err := client.SendRequest(tc.method, tc.params)
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}
			
			resp, err := client.WaitForResponse(ctx, id)
			if err != nil {
				t.Fatalf("Failed to get response: %v", err)
			}
			
			if resp.Error != nil {
				t.Fatalf("Got error response: %s", resp.Error.Message)
			}
			
			tc.validateResult(t, resp.Result)
		})
	}
}

// TestCapabilitiesHandling tests proper capabilities handling
func TestCapabilitiesHandling(t *testing.T) {
	srv := testutil.NewServerBuilder("capabilities-server", "1.0.0").
		WithSimpleTool("tool1", "result1").
		WithAutoStart().
		Build()
	
	defer srv.Stop()
	
	client := srv.Client()
	ctx := context.Background()
	
	t.Run("server_capabilities", func(t *testing.T) {
		result, err := client.Initialize(ctx, "capabilities-client", "1.0.0")
		if err != nil {
			t.Fatalf("Failed to initialize: %v", err)
		}
		
		// Check required capabilities structure
		if result.Capabilities.Tools == nil {
			t.Error("Tools capability should not be nil")
		}
		
		if result.Capabilities.Resources == nil {
			t.Error("Resources capability should not be nil")
		}
		
		if result.Capabilities.Prompts == nil {
			t.Error("Prompts capability should not be nil")
		}
	})
	
	t.Run("client_capabilities", func(t *testing.T) {
		// Test with various client capabilities
		initReq := protocol.InitializeRequest{
			ProtocolVersion: protocol.Version,
			Capabilities: protocol.ClientCapabilities{
				Experimental: map[string]interface{}{
					"feature1": true,
					"feature2": "enabled",
				},
				Sampling: map[string]interface{}{
					"enabled": true,
					"rate":    0.5,
				},
			},
			ClientInfo: protocol.ClientInfo{
				Name:    "advanced-client",
				Version: "2.0.0",
			},
		}
		
		id, err := client.SendRequest("initialize", initReq)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		
		resp, err := client.WaitForResponse(ctx, id)
		if err != nil {
			t.Fatalf("Failed to get response: %v", err)
		}
		
		if resp.Error != nil {
			t.Fatalf("Initialize failed: %s", resp.Error.Message)
		}
		
		// Server should accept any client capabilities
		data, _ := json.Marshal(resp.Result)
		var result protocol.InitializeResult
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("Failed to parse result: %v", err)
		}
	})
}

// TestContentHandling tests proper content structure handling
func TestContentHandling(t *testing.T) {
	srv := testutil.NewServerBuilder("content-server", "1.0.0").
		WithTool("multi_content", "Returns multiple content items", protocol.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return protocol.NewToolCallResult(
				protocol.NewContent("First line"),
				protocol.NewContent("Second line"),
				protocol.NewContent("Third line"),
			), nil
		})).
		WithTool("empty_content", "Returns empty content", protocol.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return protocol.NewToolCallResult(), nil
		})).
		WithAutoStart().
		Build()
	
	defer srv.Stop()
	
	client := srv.Client()
	ctx := context.Background()
	
	// Initialize
	if _, err := client.Initialize(ctx, "content-client", "1.0.0"); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	
	t.Run("multiple_content_items", func(t *testing.T) {
		result, err := client.CallTool(ctx, "multi_content", nil)
		if err != nil {
			t.Fatalf("Failed to call tool: %v", err)
		}
		
		if len(result.Content) != 3 {
			t.Errorf("Expected 3 content items, got %d", len(result.Content))
		}
		
		// All should be text type
		for i, content := range result.Content {
			if content.Type != "text" {
				t.Errorf("Content[%d] type should be 'text', got '%s'", i, content.Type)
			}
		}
	})
	
	t.Run("empty_content", func(t *testing.T) {
		result, err := client.CallTool(ctx, "empty_content", nil)
		if err != nil {
			t.Fatalf("Failed to call tool: %v", err)
		}
		
		// Empty content array is valid
		if len(result.Content) != 0 {
			t.Errorf("Expected empty content array, got %d items", len(result.Content))
		}
	})
}

// TestStrictJSONParsing tests strict JSON parsing requirements
func TestStrictJSONParsing(t *testing.T) {
	srv := testutil.NewServerBuilder("json-server", "1.0.0").
		WithAutoStart().
		Build()
	
	defer srv.Stop()
	
	client := srv.Client()
	ctx := context.Background()
	
	testCases := []struct {
		name          string
		request       string
		shouldSucceed bool
		description   string
	}{
		{
			name:          "trailing_comma",
			request:       `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{},}`,
			shouldSucceed: false,
			description:   "Trailing commas should cause parse error",
		},
		{
			name:          "single_quotes",
			request:       `{'jsonrpc':'2.0','id':1,'method':'tools/list'}`,
			shouldSucceed: false,
			description:   "Single quotes should cause parse error",
		},
		{
			name:          "unquoted_keys",
			request:       `{jsonrpc:"2.0",id:1,method:"tools/list"}`,
			shouldSucceed: false,
			description:   "Unquoted keys should cause parse error",
		},
		{
			name:          "comments",
			request:       `{"jsonrpc":"2.0","id":1,"method":"tools/list" /* comment */}`,
			shouldSucceed: false,
			description:   "Comments should cause parse error",
		},
		{
			name:          "nan_value",
			request:       `{"jsonrpc":"2.0","id":NaN,"method":"tools/list"}`,
			shouldSucceed: false,
			description:   "NaN should cause parse error",
		},
		{
			name:          "infinity_value",
			request:       `{"jsonrpc":"2.0","id":Infinity,"method":"tools/list"}`,
			shouldSucceed: false,
			description:   "Infinity should cause parse error",
		},
		{
			name:          "valid_json",
			request:       `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
			shouldSucceed: true,
			description:   "Valid JSON should succeed",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := srv.Client().GetInput().(*bytes.Buffer)
			input.WriteString(tc.request + "\n")
			
			resp, err := client.GetNextResponse(ctx)
			if err != nil {
				t.Fatalf("Failed to get response: %v", err)
			}
			
			if tc.shouldSucceed {
				if resp.Error != nil {
					t.Errorf("%s: %s", tc.description, resp.Error.Message)
				}
			} else {
				if resp.Error == nil {
					t.Errorf("%s", tc.description)
				} else if resp.Error.Code != protocol.ParseError {
					t.Errorf("%s (got error code %d)", tc.description, resp.Error.Code)
				}
			}
		})
	}
}

// TestInteroperabilityPatterns tests common patterns for interoperability
func TestInteroperabilityPatterns(t *testing.T) {
	srv := testutil.NewServerBuilder("interop-server", "1.0.0").
		WithTool("flexible_tool", "Handles various input formats", protocol.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			// Should handle both missing params and various types
			if params == nil {
				return "no params", nil
			}
			
			// Handle different number types
			if val, ok := params["number"]; ok {
				switch v := val.(type) {
				case int:
					return fmt.Sprintf("int: %d", v), nil
				case int64:
					return fmt.Sprintf("int64: %d", v), nil
				case float64:
					return fmt.Sprintf("float64: %f", v), nil
				case json.Number:
					return fmt.Sprintf("json.Number: %s", v), nil
				default:
					return fmt.Sprintf("unknown number type: %T", v), nil
				}
			}
			
			return "processed", nil
		})).
		WithAutoStart().
		Build()
	
	defer srv.Stop()
	
	client := srv.Client()
	ctx := context.Background()
	
	// Initialize
	if _, err := client.Initialize(ctx, "interop-client", "1.0.0"); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	
	t.Run("number_handling", func(t *testing.T) {
		// Different clients might send numbers in different formats
		numberFormats := []string{
			`{"number": 42}`,          // Integer
			`{"number": 42.0}`,        // Float
			`{"number": 4.2e1}`,       // Scientific notation
			`{"number": 9007199254740992}`, // Large integer (beyond JS safe integer)
		}
		
		for _, format := range numberFormats {
			id, err := client.SendRequest("tools/call", json.RawMessage(
				fmt.Sprintf(`{"name": "flexible_tool", "arguments": %s}`, format),
			))
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}
			
			resp, err := client.WaitForResponse(ctx, id)
			if err != nil {
				t.Fatalf("Failed to get response: %v", err)
			}
			
			if resp.Error != nil {
				t.Errorf("Tool call failed for format %s: %s", format, resp.Error.Message)
			}
		}
	})
	
	t.Run("optional_fields", func(t *testing.T) {
		// All optional fields should be truly optional
		minimalRequests := []struct {
			name   string
			method string
			params string
		}{
			{
				name:   "minimal_tool_call",
				method: "tools/call",
				params: `{"name": "flexible_tool"}`, // No arguments field
			},
			{
				name:   "empty_arguments",
				method: "tools/call",
				params: `{"name": "flexible_tool", "arguments": {}}`,
			},
			{
				name:   "null_arguments",
				method: "tools/call",
				params: `{"name": "flexible_tool", "arguments": null}`,
			},
		}
		
		for _, req := range minimalRequests {
			t.Run(req.name, func(t *testing.T) {
				id, err := client.SendRequest(req.method, json.RawMessage(req.params))
				if err != nil {
					t.Fatalf("Failed to send request: %v", err)
				}
				
				resp, err := client.WaitForResponse(ctx, id)
				if err != nil {
					t.Fatalf("Failed to get response: %v", err)
				}
				
				// Should handle gracefully without error
				if resp.Error != nil {
					t.Errorf("Request failed: %s", resp.Error.Message)
				}
			})
		}
	})
}

// TestSpecialCharacterHandling tests handling of special characters
func TestSpecialCharacterHandling(t *testing.T) {
	srv := testutil.NewServerBuilder("unicode-server", "1.0.0").
		WithTool("echo", "Echoes input", protocol.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			text, _ := params["text"].(string)
			return text, nil
		})).
		WithAutoStart().
		Build()
	
	defer srv.Stop()
	
	client := srv.Client()
	ctx := context.Background()
	
	// Initialize
	if _, err := client.Initialize(ctx, "unicode-client", "1.0.0"); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	
	specialStrings := []struct {
		name string
		text string
	}{
		{"emoji", "Hello 👋 World 🌍"},
		{"unicode", "Ḧëłłö Ẅöṛḷḋ"},
		{"cjk", "你好世界 こんにちは世界 안녕하세요"},
		{"rtl", "مرحبا بالعالم"},
		{"control_chars", "Line1\nLine2\tTabbed\rCarriage"},
		{"quotes", `"Hello" 'World' \"Escaped\"`},
		{"backslashes", `C:\Users\Test\Path`},
		{"null_char", "Before\x00After"},
		{"mixed", "🎉 Unicode: ñ CJK: 中文 مرحبا \n\t\"quoted\""},
	}
	
	for _, tc := range specialStrings {
		t.Run(tc.name, func(t *testing.T) {
			result, err := client.CallTool(ctx, "echo", map[string]interface{}{
				"text": tc.text,
			})
			if err != nil {
				t.Fatalf("Failed to call tool: %v", err)
			}
			
			if len(result.Content) != 1 {
				t.Fatal("Expected one content item")
			}
			
			// Should preserve the exact string
			if result.Content[0].Text != tc.text {
				t.Errorf("String not preserved correctly\nSent: %q\nGot:  %q", tc.text, result.Content[0].Text)
			}
		})
	}
}

// TestLineDelimitedJSONStreaming tests proper line-delimited JSON handling
func TestLineDelimitedJSONStreaming(t *testing.T) {
	srv := testutil.NewServerBuilder("streaming-server", "1.0.0").
		WithSimpleTool("test", "result").
		WithAutoStart().
		Build()
	
	defer srv.Stop()
	
	client := srv.Client()
	ctx := context.Background()
	
	// Initialize
	if _, err := client.Initialize(ctx, "streaming-client", "1.0.0"); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	
	t.Run("multiple_requests_single_line", func(t *testing.T) {
		// Send multiple requests on a single line (should each be processed)
		requests := []string{
			`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
			`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
			`{"jsonrpc":"2.0","id":3,"method":"tools/list"}`,
		}
		
		input := srv.Client().GetInput().(*bytes.Buffer)
		for _, req := range requests {
			input.WriteString(req + "\n")
		}
		
		// Should get three responses
		for i := 0; i < 3; i++ {
			resp, err := client.GetNextResponse(ctx)
			if err != nil {
				t.Fatalf("Failed to get response %d: %v", i+1, err)
			}
			
			if resp.Error != nil {
				t.Errorf("Response %d has error: %s", i+1, resp.Error.Message)
			}
		}
	})
	
	t.Run("request_split_across_lines", func(t *testing.T) {
		// JSON split across multiple lines should fail
		input := srv.Client().GetInput().(*bytes.Buffer)
		input.WriteString(`{"jsonrpc":"2.0",` + "\n")
		input.WriteString(`"id":4,"method":"tools/list"}` + "\n")
		
		// First line should produce parse error
		resp, err := client.GetNextResponse(ctx)
		if err != nil {
			t.Fatalf("Failed to get response: %v", err)
		}
		
		if resp.Error == nil || resp.Error.Code != protocol.ParseError {
			t.Error("Expected parse error for incomplete JSON")
		}
	})
	
	t.Run("whitespace_handling", func(t *testing.T) {
		// Various whitespace scenarios
		requests := []string{
			`  {"jsonrpc":"2.0","id":5,"method":"tools/list"}  `, // Leading/trailing spaces
			`	{"jsonrpc":"2.0","id":6,"method":"tools/list"}`,   // Leading tab
			`{"jsonrpc":"2.0","id":7,"method":"tools/list"}	`,   // Trailing tab
		}
		
		for _, req := range requests {
			input := srv.Client().GetInput().(*bytes.Buffer)
			input.WriteString(req + "\n")
			
			resp, err := client.GetNextResponse(ctx)
			if err != nil {
				t.Fatalf("Failed to get response: %v", err)
			}
			
			// Should handle whitespace gracefully
			if resp.Error != nil {
				t.Errorf("Request with whitespace failed: %s", resp.Error.Message)
			}
		}
	})
}

// TestRealWorldCompatibility simulates real-world client behaviors
func TestRealWorldCompatibility(t *testing.T) {
	srv := testutil.NewServerBuilder("realworld-server", "1.0.0").
		WithTool("process", "Processes data", protocol.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			// Simulate processing
			return map[string]interface{}{
				"status": "success",
				"processed": params,
			}, nil
		})).
		WithAutoStart().
		Build()
	
	defer srv.Stop()
	
	client := srv.Client()
	ctx := context.Background()
	
	// Initialize with realistic client info
	_, err := client.Initialize(ctx, "vscode-mcp-client", "0.1.0")
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	
	t.Run("typescript_style_requests", func(t *testing.T) {
		// TypeScript clients might send certain patterns
		id, err := client.SendRequest("tools/call", json.RawMessage(`{
			"name": "process",
			"arguments": {
				"data": {
					"type": "object",
					"properties": {
						"name": { "type": "string" },
						"value": { "type": "number" }
					}
				},
				"options": {
					"validate": true,
					"timeout": 5000
				}
			}
		}`))
		
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		
		resp, err := client.WaitForResponse(ctx, id)
		if err != nil {
			t.Fatalf("Failed to get response: %v", err)
		}
		
		if resp.Error != nil {
			t.Errorf("Request failed: %s", resp.Error.Message)
		}
	})
	
	t.Run("python_style_requests", func(t *testing.T) {
		// Python clients might use different conventions
		pythonStyleParams := map[string]interface{}{
			"name": "process",
			"arguments": map[string]interface{}{
				"kwargs": map[string]interface{}{
					"input_data": []interface{}{1, 2, 3, 4, 5},
					"config": map[string]interface{}{
						"debug": true,
						"max_iterations": 100,
					},
				},
			},
		}
		
		id, err := client.SendRequest("tools/call", pythonStyleParams)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		
		resp, err := client.WaitForResponse(ctx, id)
		if err != nil {
			t.Fatalf("Failed to get response: %v", err)
		}
		
		if resp.Error != nil {
			t.Errorf("Request failed: %s", resp.Error.Message)
		}
	})
	
	t.Run("browser_client_patterns", func(t *testing.T) {
		// Browser-based clients might have certain limitations
		// e.g., integer precision issues with large numbers
		browserRequest := map[string]interface{}{
			"name": "process",
			"arguments": map[string]interface{}{
				"largeInt": 9007199254740991, // MAX_SAFE_INTEGER in JS
				"float": 0.1 + 0.2,          // Classic JS floating point
				"nested": map[string]interface{}{
					"array": []interface{}{true, false, nil},
				},
			},
		}
		
		id, err := client.SendRequest("tools/call", browserRequest)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		
		resp, err := client.WaitForResponse(ctx, id)
		if err != nil {
			t.Fatalf("Failed to get response: %v", err)
		}
		
		if resp.Error != nil {
			t.Errorf("Request failed: %s", resp.Error.Message)
		}
	})
}

// TestEdgeCaseHandling tests various edge cases
func TestEdgeCaseHandling(t *testing.T) {
	srv := testutil.NewServerBuilder("edge-server", "1.0.0").
		WithTool("edge_tool", "Handles edge cases", protocol.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			// Return exactly what was sent
			return params, nil
		})).
		WithAutoStart().
		Build()
	
	defer srv.Stop()
	
	client := srv.Client()
	ctx := context.Background()
	
	// Initialize
	if _, err := client.Initialize(ctx, "edge-client", "1.0.0"); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	
	edgeCases := []struct {
		name   string
		params interface{}
	}{
		{
			name:   "empty_string_values",
			params: map[string]interface{}{"empty": "", "spaces": "   "},
		},
		{
			name:   "deeply_nested",
			params: createDeeplyNested(10),
		},
		{
			name:   "large_array",
			params: map[string]interface{}{"array": createLargeArray(1000)},
		},
		{
			name:   "mixed_types",
			params: map[string]interface{}{
				"string": "text",
				"number": 42,
				"float": 3.14,
				"bool": true,
				"null": nil,
				"array": []interface{}{1, "two", 3.0, true, nil},
				"object": map[string]interface{}{"nested": true},
			},
		},
		{
			name: "special_json_values",
			params: map[string]interface{}{
				"zero": 0,
				"negative": -1,
				"empty_array": []interface{}{},
				"empty_object": map[string]interface{}{},
				"false": false,
			},
		},
	}
	
	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := client.CallTool(ctx, "edge_tool", tc.params.(map[string]interface{}))
			if err != nil {
				t.Fatalf("Failed to call tool: %v", err)
			}
			
			if result.IsError {
				t.Errorf("Tool returned error: %v", result.Content)
			}
		})
	}
}

// Helper functions

func createDeeplyNested(depth int) map[string]interface{} {
	result := map[string]interface{}{}
	current := result
	
	for i := 0; i < depth; i++ {
		next := map[string]interface{}{}
		current[fmt.Sprintf("level_%d", i)] = next
		current = next
	}
	
	current["value"] = "deeply nested value"
	return result
}

func createLargeArray(size int) []interface{} {
	result := make([]interface{}, size)
	for i := range result {
		result[i] = i
	}
	return result
}