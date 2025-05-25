package mcp_test

import (
	"context"
	"errors"
	"fmt"
	"mcp-memory/pkg/mcp/protocol"
	"mcp-memory/pkg/mcp/testutil"
	"sync"
	"testing"
	"time"
)

// TestFullMCPFlow tests a complete MCP interaction flow
func TestFullMCPFlow(t *testing.T) {
	// Create server with various capabilities
	srv := testutil.NewServerBuilder("integration-test-server", "1.0.0").
		WithSimpleTool("echo", "echoed").
		WithTool("calculator", "Performs calculations", protocol.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			op, _ := params["operation"].(string)
			a, _ := params["a"].(float64)
			b, _ := params["b"].(float64)
			
			switch op {
			case "add":
				return fmt.Sprintf("%v", a+b), nil
			case "subtract":
				return fmt.Sprintf("%v", a-b), nil
			case "multiply":
				return fmt.Sprintf("%v", a*b), nil
			case "divide":
				if b == 0 {
					return nil, errors.New("division by zero")
				}
				return fmt.Sprintf("%v", a/b), nil
			default:
				return nil, errors.New("unknown operation")
			}
		})).
		WithResource("file:///readme.txt", "readme.txt", "This is a test file").
		WithPrompt("greeting", "Hello, {{name}}!").
		WithAutoStart().
		Build()
	
	defer srv.Stop()
	
	client := srv.Client()
	ctx := context.Background()
	
	// Test 1: Initialize
	t.Run("initialization", func(t *testing.T) {
		result, err := client.Initialize(ctx, "test-client", "1.0.0")
		if err != nil {
			t.Fatalf("Failed to initialize: %v", err)
		}
		
		if result.ProtocolVersion != protocol.Version {
			t.Errorf("Expected protocol version %s, got %s", protocol.Version, result.ProtocolVersion)
		}
		
		if result.ServerInfo.Name != "integration-test-server" {
			t.Errorf("Expected server name 'integration-test-server', got %s", result.ServerInfo.Name)
		}
	})
	
	// Test 2: List tools
	t.Run("list_tools", func(t *testing.T) {
		tools, err := client.ListTools(ctx)
		if err != nil {
			t.Fatalf("Failed to list tools: %v", err)
		}
		
		if len(tools) != 2 {
			t.Errorf("Expected 2 tools, got %d", len(tools))
		}
		
		// Check tool names
		toolNames := make(map[string]bool)
		for _, tool := range tools {
			toolNames[tool.Name] = true
		}
		
		if !toolNames["echo"] {
			t.Error("Expected 'echo' tool")
		}
		if !toolNames["calculator"] {
			t.Error("Expected 'calculator' tool")
		}
	})
	
	// Test 3: Call tools
	t.Run("call_echo_tool", func(t *testing.T) {
		result, err := client.CallTool(ctx, "echo", nil)
		if err != nil {
			t.Fatalf("Failed to call echo tool: %v", err)
		}
		
		if result.IsError {
			t.Error("Expected successful result")
		}
		
		if len(result.Content) != 1 || result.Content[0].Text != "echoed" {
			t.Errorf("Expected 'echoed' result, got %v", result.Content)
		}
	})
	
	t.Run("call_calculator_tool", func(t *testing.T) {
		// Test addition
		result, err := client.CallTool(ctx, "calculator", map[string]interface{}{
			"operation": "add",
			"a":         10.5,
			"b":         20.5,
		})
		if err != nil {
			t.Fatalf("Failed to call calculator: %v", err)
		}
		
		if result.IsError {
			t.Error("Expected successful result")
		}
		
		if len(result.Content) != 1 || result.Content[0].Text != "31" {
			t.Errorf("Expected '31' result, got %v", result.Content)
		}
		
		// Test division by zero
		result, err = client.CallTool(ctx, "calculator", map[string]interface{}{
			"operation": "divide",
			"a":         10.0,
			"b":         0.0,
		})
		if err != nil {
			t.Fatalf("Failed to call calculator: %v", err)
		}
		
		if !result.IsError {
			t.Error("Expected error result for division by zero")
		}
	})
	
	// Test 4: Resources
	t.Run("resources", func(t *testing.T) {
		// List resources
		id, err := client.SendRequest("resources/list", nil)
		if err != nil {
			t.Fatalf("Failed to send resources/list: %v", err)
		}
		
		resp, err := client.WaitForResponse(ctx, id)
		if err != nil {
			t.Fatalf("Failed to get response: %v", err)
		}
		
		if resp.Error != nil {
			t.Fatalf("Got error response: %s", resp.Error.Message)
		}
		
		// Read resource
		id, err = client.SendRequest("resources/read", map[string]interface{}{
			"uri": "file:///readme.txt",
		})
		if err != nil {
			t.Fatalf("Failed to send resources/read: %v", err)
		}
		
		resp, err = client.WaitForResponse(ctx, id)
		if err != nil {
			t.Fatalf("Failed to get response: %v", err)
		}
		
		if resp.Error != nil {
			t.Fatalf("Got error response: %s", resp.Error.Message)
		}
	})
	
	// Test 5: Prompts
	t.Run("prompts", func(t *testing.T) {
		// List prompts
		id, err := client.SendRequest("prompts/list", nil)
		if err != nil {
			t.Fatalf("Failed to send prompts/list: %v", err)
		}
		
		resp, err := client.WaitForResponse(ctx, id)
		if err != nil {
			t.Fatalf("Failed to get response: %v", err)
		}
		
		if resp.Error != nil {
			t.Fatalf("Got error response: %s", resp.Error.Message)
		}
		
		// Get prompt
		id, err = client.SendRequest("prompts/get", map[string]interface{}{
			"name": "greeting",
			"arguments": map[string]interface{}{
				"name": "World",
			},
		})
		if err != nil {
			t.Fatalf("Failed to send prompts/get: %v", err)
		}
		
		resp, err = client.WaitForResponse(ctx, id)
		if err != nil {
			t.Fatalf("Failed to get response: %v", err)
		}
		
		if resp.Error != nil {
			t.Fatalf("Got error response: %s", resp.Error.Message)
		}
	})
}

// TestConcurrentOperations tests concurrent request handling
func TestConcurrentOperations(t *testing.T) {
	srv := testutil.NewServerBuilder("concurrent-test-server", "1.0.0").
		WithTool("counter", "Increments a counter", &counterHandler{}).
		WithTool("slow", "Slow operation", protocol.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			delay, _ := params["delay"].(float64)
			if delay == 0 {
				delay = 100
			}
			
			select {
			case <-time.After(time.Duration(delay) * time.Millisecond):
				return "completed", nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		})).
		WithAutoStart().
		Build()
	
	defer srv.Stop()
	
	client := srv.Client()
	ctx := context.Background()
	
	// Initialize
	if _, err := client.Initialize(ctx, "concurrent-client", "1.0.0"); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	
	t.Run("concurrent_tool_calls", func(t *testing.T) {
		const numCalls = 50
		var wg sync.WaitGroup
		errors := make(chan error, numCalls)
		
		// Make concurrent calls to counter
		for i := 0; i < numCalls; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				
				result, err := client.CallTool(ctx, "counter", map[string]interface{}{
					"increment": 1,
				})
				if err != nil {
					errors <- err
					return
				}
				
				if result.IsError {
					errors <- fmt.Errorf("tool call failed: %v", result.Content)
				}
			}()
		}
		
		// Wait for all calls to complete
		wg.Wait()
		close(errors)
		
		// Check for errors
		for err := range errors {
			t.Errorf("Concurrent call failed: %v", err)
		}
		
		// Verify final count
		result, err := client.CallTool(ctx, "counter", map[string]interface{}{
			"get": true,
		})
		if err != nil {
			t.Fatalf("Failed to get counter: %v", err)
		}
		
		if len(result.Content) != 1 {
			t.Fatal("Expected one content item")
		}
		
		// Counter should be numCalls
		if result.Content[0].Text != fmt.Sprintf("%d", numCalls) {
			t.Errorf("Expected counter to be %d, got %s", numCalls, result.Content[0].Text)
		}
	})
	
	t.Run("mixed_concurrent_operations", func(t *testing.T) {
		var wg sync.WaitGroup
		errors := make(chan error, 100)
		
		// Mix of different operations
		for i := 0; i < 10; i++ {
			// Tool calls
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				
				_, err := client.CallTool(ctx, "slow", map[string]interface{}{
					"delay": float64(10 + idx),
				})
				if err != nil {
					errors <- err
				}
			}(i)
			
			// List operations
			wg.Add(1)
			go func() {
				defer wg.Done()
				
				_, err := client.ListTools(ctx)
				if err != nil {
					errors <- err
				}
			}()
			
			// Invalid requests
			wg.Add(1)
			go func() {
				defer wg.Done()
				
				id, err := client.SendRequest("invalid/method", nil)
				if err != nil {
					errors <- err
					return
				}
				
				resp, err := client.WaitForResponse(ctx, id)
				if err != nil {
					errors <- err
					return
				}
				
				if resp.Error == nil || resp.Error.Code != protocol.MethodNotFound {
					errors <- fmt.Errorf("expected method not found error")
				}
			}()
		}
		
		// Wait for completion
		done := make(chan bool)
		go func() {
			wg.Wait()
			close(done)
		}()
		
		select {
		case <-done:
			// Success
		case <-time.After(5 * time.Second):
			t.Fatal("Concurrent operations timed out")
		}
		
		close(errors)
		for err := range errors {
			t.Errorf("Operation failed: %v", err)
		}
	})
}

// TestErrorScenarios tests various error conditions
func TestErrorScenarios(t *testing.T) {
	srv := testutil.NewServerBuilder("error-test-server", "1.0.0").
		WithTool("error_tool", "Always errors", protocol.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			errorType, _ := params["type"].(string)
			switch errorType {
			case "panic":
				panic("test panic")
			case "nil":
				return nil, nil
			case "timeout":
				<-ctx.Done()
				return nil, ctx.Err()
			default:
				return nil, errors.New("test error")
			}
		})).
		WithAutoStart().
		Build()
	
	defer srv.Stop()
	
	client := srv.Client()
	ctx := context.Background()
	
	// Initialize
	if _, err := client.Initialize(ctx, "error-client", "1.0.0"); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	
	tests := []struct {
		name           string
		method         string
		params         interface{}
		expectedError  bool
		expectedCode   int
	}{
		{
			name:          "unknown_method",
			method:        "unknown/method",
			params:        nil,
			expectedError: true,
			expectedCode:  protocol.MethodNotFound,
		},
		{
			name:          "invalid_params_type",
			method:        "tools/call",
			params:        "invalid",
			expectedError: true,
			expectedCode:  protocol.InvalidParams,
		},
		{
			name:   "missing_required_param",
			method: "tools/call",
			params: map[string]interface{}{
				// Missing 'name' parameter
			},
			expectedError: true,
			expectedCode:  protocol.InvalidParams,
		},
		{
			name:   "tool_not_found",
			method: "tools/call",
			params: protocol.ToolCallRequest{
				Name: "nonexistent",
			},
			expectedError: true,
			expectedCode:  protocol.MethodNotFound,
		},
		{
			name:   "resource_not_found",
			method: "resources/read",
			params: map[string]interface{}{
				"uri": "file:///nonexistent",
			},
			expectedError: true,
			expectedCode:  protocol.MethodNotFound,
		},
		{
			name:   "prompt_not_found",
			method: "prompts/get",
			params: map[string]interface{}{
				"name": "nonexistent",
			},
			expectedError: true,
			expectedCode:  protocol.MethodNotFound,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := client.SendRequest(tt.method, tt.params)
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}
			
			resp, err := client.WaitForResponse(ctx, id)
			if err != nil {
				t.Fatalf("Failed to get response: %v", err)
			}
			
			if tt.expectedError {
				if resp.Error == nil {
					t.Fatal("Expected error response but got success")
				}
				if resp.Error.Code != tt.expectedCode {
					t.Errorf("Expected error code %d, got %d", tt.expectedCode, resp.Error.Code)
				}
			} else {
				if resp.Error != nil {
					t.Fatalf("Expected success but got error: %s", resp.Error.Message)
				}
			}
		})
	}
}

// TestNotifications tests notification handling (no response expected)
func TestNotifications(t *testing.T) {
	notificationReceived := make(chan string, 10)
	
	srv := testutil.NewServerBuilder("notification-test-server", "1.0.0").
		WithTool("notify", "Sends notifications", protocol.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			msg, _ := params["message"].(string)
			select {
			case notificationReceived <- msg:
			default:
			}
			return "notification sent", nil
		})).
		WithAutoStart().
		Build()
	
	defer srv.Stop()
	
	client := srv.Client()
	ctx := context.Background()
	
	// Initialize
	if _, err := client.Initialize(ctx, "notification-client", "1.0.0"); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	
	t.Run("send_notification", func(t *testing.T) {
		// Send notification (no ID, so no response expected)
		err := client.SendNotification("tools/call", protocol.ToolCallRequest{
			Name: "notify",
			Arguments: map[string]interface{}{
				"message": "test notification",
			},
		})
		if err != nil {
			t.Fatalf("Failed to send notification: %v", err)
		}
		
		// Give time for processing
		time.Sleep(100 * time.Millisecond)
		
		// Check if notification was received
		select {
		case msg := <-notificationReceived:
			if msg != "test notification" {
				t.Errorf("Expected 'test notification', got '%s'", msg)
			}
		case <-time.After(time.Second):
			t.Error("Notification was not processed")
		}
	})
}

// TestLargePayloads tests handling of large request/response payloads
func TestLargePayloads(t *testing.T) {
	srv := testutil.NewServerBuilder("large-payload-server", "1.0.0").
		WithTool("process_large", "Processes large data", protocol.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			data, _ := params["data"].(string)
			// Return size and first/last 10 chars
			if len(data) > 20 {
				return fmt.Sprintf("Processed %d bytes: %s...%s", len(data), data[:10], data[len(data)-10:]), nil
			}
			return fmt.Sprintf("Processed %d bytes: %s", len(data), data), nil
		})).
		WithTool("generate_large", "Generates large data", protocol.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			size, _ := params["size"].(float64)
			if size == 0 {
				size = 1024
			}
			
			// Generate data
			data := make([]byte, int(size))
			for i := range data {
				data[i] = byte('A' + (i % 26))
			}
			
			return string(data), nil
		})).
		WithAutoStart().
		Build()
	
	defer srv.Stop()
	
	client := srv.Client()
	ctx := context.Background()
	
	// Initialize
	if _, err := client.Initialize(ctx, "large-payload-client", "1.0.0"); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	
	sizes := []int{
		1024,        // 1KB
		1024 * 10,   // 10KB
		1024 * 100,  // 100KB
		1024 * 1024, // 1MB
	}
	
	for _, size := range sizes {
		t.Run(fmt.Sprintf("payload_%d_bytes", size), func(t *testing.T) {
			// Generate large input
			largeData := make([]byte, size)
			for i := range largeData {
				largeData[i] = byte('X')
			}
			
			// Send large request
			result, err := client.CallTool(ctx, "process_large", map[string]interface{}{
				"data": string(largeData),
			})
			if err != nil {
				t.Fatalf("Failed to process large data: %v", err)
			}
			
			if result.IsError {
				t.Fatalf("Tool returned error: %v", result.Content)
			}
			
			// Test large response
			result, err = client.CallTool(ctx, "generate_large", map[string]interface{}{
				"size": float64(size),
			})
			if err != nil {
				t.Fatalf("Failed to generate large data: %v", err)
			}
			
			if result.IsError {
				t.Fatalf("Tool returned error: %v", result.Content)
			}
			
			if len(result.Content) != 1 || len(result.Content[0].Text) != size {
				t.Errorf("Expected %d bytes in response, got %d", size, len(result.Content[0].Text))
			}
		})
	}
}

// TestScenarioRunner tests using the scenario runner
func TestScenarioRunner(t *testing.T) {
	runner := testutil.NewScenarioRunner(t)
	
	// Add common scenarios
	runner.AddScenario(testutil.CommonScenarios.BasicToolCall("test_tool", "test result"))
	runner.AddScenario(testutil.CommonScenarios.ErrorHandling())
	runner.AddScenario(testutil.CommonScenarios.ConcurrentRequests(10))
	runner.AddScenario(testutil.CommonScenarios.ResourceReadWrite())
	runner.AddScenario(testutil.CommonScenarios.PromptGeneration())
	
	// Run all scenarios
	runner.Run()
}

// Helper types

type counterHandler struct {
	mu    sync.Mutex
	count int
}

func (h *counterHandler) Handle(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	if increment, ok := params["increment"].(float64); ok {
		h.count += int(increment)
		return fmt.Sprintf("%d", h.count), nil
	}
	
	if _, ok := params["get"].(bool); ok {
		return fmt.Sprintf("%d", h.count), nil
	}
	
	return nil, errors.New("invalid parameters")
}