package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"mcp-memory/pkg/mcp/protocol"
	"mcp-memory/pkg/mcp/server"
)

// TestInputValidation tests various input validation scenarios
func TestInputValidation(t *testing.T) {

	tests := []struct {
		name      string
		input     string
		wantError bool
		errorType string
	}{
		{
			name:      "valid JSON-RPC request",
			input:     `{"jsonrpc":"2.0","method":"test","params":{},"id":1}`,
			wantError: false,
		},
		{
			name:      "invalid JSON",
			input:     `{"jsonrpc":"2.0","method":"test"`,
			wantError: true,
			errorType: "parse error",
		},
		{
			name:      "missing jsonrpc version",
			input:     `{"method":"test","params":{},"id":1}`,
			wantError: true,
			errorType: "invalid request",
		},
		{
			name:      "SQL injection attempt",
			input:     `{"jsonrpc":"2.0","method":"test","params":{"query":"'; DROP TABLE users; --"},"id":1}`,
			wantError: false, // Should be handled safely
		},
		{
			name:      "oversized payload",
			input:     `{"jsonrpc":"2.0","method":"test","params":{"data":"` + strings.Repeat("x", 10*1024*1024) + `"},"id":1}`,
			wantError: true,
			errorType: "payload too large",
		},
		{
			name:      "null byte injection",
			input:     `{"jsonrpc":"2.0","method":"test\x00malicious","params":{},"id":1}`,
			wantError: true,
			errorType: "invalid characters",
		},
		{
			name:      "unicode control characters",
			input:     `{"jsonrpc":"2.0","method":"test\u0001\u0002","params":{},"id":1}`,
			wantError: true,
			errorType: "invalid characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req protocol.JSONRPCRequest
			err := json.Unmarshal([]byte(tt.input), &req)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error for input: %s", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestPathTraversalProtection tests protection against directory traversal attacks
func TestPathTraversalProtection(t *testing.T) {
	paths := []struct {
		input    string
		expected string
		blocked  bool
	}{
		{"/valid/path/file.txt", "/valid/path/file.txt", false},
		{"../../../etc/passwd", "", true},
		{"/path/../../../etc/passwd", "", true},
		{"/path/..\\..\\..\\windows\\system32", "", true},
		{"/path/%2e%2e%2f%2e%2e%2f", "", true},
		{"/path/\x00/malicious", "", true},
	}

	for _, p := range paths {
		t.Run(p.input, func(t *testing.T) {
			safe := isPathSafe(p.input)
			if safe && p.blocked {
				t.Errorf("path should have been blocked: %s", p.input)
			}
			if !safe && !p.blocked {
				t.Errorf("path should have been allowed: %s", p.input)
			}
		})
	}
}

// TestRateLimiting tests rate limiting functionality
func TestRateLimiting(t *testing.T) {
	s := server.NewServer("test-server", "1.0.0") // TODO: Add rate limiting middleware

	// Register a simple tool
	tool := protocol.Tool{
		Name:        "test_tool",
		Description: "Test tool",
		InputSchema: map[string]interface{}{"type": "object"},
	}
	handler := protocol.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		return map[string]string{"result": "ok"}, nil
	})
	s.AddTool(tool, handler)

	// Simulate rapid requests
	successCount := 0
	rateLimitCount := 0

	for i := 0; i < 20; i++ {
		req := protocol.JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      i,
			Method:  "tools/call",
			Params: json.RawMessage(`{
				"name": "test_tool",
				"arguments": {}
			}`),
		}

		resp := s.HandleRequest(context.Background(), &req)
		if resp.Error != nil && resp.Error.Code == -32429 { // Rate limit error
			rateLimitCount++
		} else if resp.Error == nil {
			successCount++
		}
	}

	if successCount > 10 {
		t.Errorf("rate limiting not working: %d requests succeeded (expected max 10)", successCount)
	}
	if rateLimitCount == 0 {
		t.Error("no rate limit errors received")
	}
}

// TestAuthenticationBypass tests for authentication bypass attempts
func TestAuthenticationBypass(t *testing.T) {
	// Create server with authentication
	// TODO: Add authentication middleware
	s := server.NewServer("secure-server", "1.0.0")

	tests := []struct {
		name      string
		token     string
		shouldFail bool
	}{
		{"valid token", "valid-token", false},
		{"invalid token", "invalid-token", true},
		{"empty token", "", true},
		{"null token", "null", true},
		{"JWT-like token", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U", true},
		{"SQL injection", "' OR '1'='1", true},
		{"special characters", "!@#$%^&*()", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), "auth_token", tt.token)
			
			req := protocol.JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "ping",
			}

			resp := s.HandleRequest(ctx, &req)
			
			if tt.shouldFail && resp.Error == nil {
				t.Errorf("expected authentication to fail for token: %s", tt.token)
			}
			if !tt.shouldFail && resp.Error != nil {
				t.Errorf("expected authentication to succeed for token: %s", tt.token)
			}
		})
	}
}

// TestMemoryExhaustion tests protection against memory exhaustion attacks
func TestMemoryExhaustion(t *testing.T) {
	s := server.NewServer("test-server", "1.0.0") // TODO: Add request size limit middleware

	// Test large array attack
	largeArray := make([]int, 1000000)
	data, _ := json.Marshal(largeArray)
	
	req := protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "test",
		Params:  json.RawMessage(data),
	}

	// This should be rejected due to size
	resp := s.HandleRequest(context.Background(), &req)
	if resp.Error == nil {
		t.Error("expected error for oversized request")
	}
}

// TestConcurrentSafety tests for race conditions
func TestConcurrentSafety(t *testing.T) {
	s := server.NewServer("test-server", "1.0.0")
	
	// Register a tool that modifies shared state
	counter := 0
	tool := protocol.Tool{
		Name:        "increment",
		Description: "Increment counter",
		InputSchema: map[string]interface{}{"type": "object"},
	}
	handler := protocol.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		// This is intentionally unsafe to test race detection
		counter++
		return map[string]int{"value": counter}, nil
	})
	s.AddTool(tool, handler)

	// Run concurrent requests
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func(id int) {
			req := protocol.JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      id,
				Method:  "tools/call",
				Params: json.RawMessage(`{
					"name": "increment",
					"arguments": {}
				}`),
			}
			s.HandleRequest(context.Background(), &req)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	// Note: This test should be run with -race flag to detect race conditions
}

// TestErrorMessageSanitization tests that error messages don't leak sensitive info
func TestErrorMessageSanitization(t *testing.T) {
	s := server.NewServer("test-server", "1.0.0")

	// Register a tool that throws errors with sensitive info
	tool := protocol.Tool{
		Name:        "database_query",
		Description: "Query database",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{"type": "string"},
			},
		},
	}
	handler := protocol.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		// Simulate database error
		return nil, fmt.Errorf("pq: password authentication failed for user 'admin' at 192.168.1.100:5432")
	})
	s.AddTool(tool, handler)

	req := protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params: json.RawMessage(`{
			"name": "database_query",
			"arguments": {"query": "SELECT * FROM users"}
		}`),
	}

	resp := s.HandleRequest(context.Background(), &req)
	
	if resp.Error == nil {
		t.Fatal("expected error response")
	}

	// Check that sensitive information is not in error message
	errMsg := resp.Error.Message
	if strings.Contains(errMsg, "192.168.1.100") {
		t.Error("error message contains IP address")
	}
	if strings.Contains(errMsg, "password") {
		t.Error("error message contains password-related information")
	}
	if strings.Contains(errMsg, "admin") {
		t.Error("error message contains username")
	}
}

// TestTimeoutProtection tests timeout handling to prevent DoS
func TestTimeoutProtection(t *testing.T) {
	s := server.NewServer("test-server", "1.0.0") // TODO: Add timeout middleware

	// Register a slow tool
	tool := protocol.Tool{
		Name:        "slow_tool",
		Description: "Intentionally slow tool",
		InputSchema: map[string]interface{}{"type": "object"},
	}
	handler := protocol.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		select {
		case <-time.After(5 * time.Second):
			return map[string]string{"result": "completed"}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})
	s.AddTool(tool, handler)

	req := protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params: json.RawMessage(`{
			"name": "slow_tool",
			"arguments": {}
		}`),
	}

	start := time.Now()
	resp := s.HandleRequest(context.Background(), &req)
	duration := time.Since(start)

	if resp.Error == nil {
		t.Error("expected timeout error")
	}

	if duration > 200*time.Millisecond {
		t.Errorf("timeout took too long: %v", duration)
	}
}

// TestXSSPrevention tests Cross-Site Scripting prevention
func TestXSSPrevention(t *testing.T) {
	s := server.NewServer("test-server", "1.0.0")

	// Register a tool that echoes input
	tool := protocol.Tool{
		Name:        "echo",
		Description: "Echo input",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"message": map[string]interface{}{"type": "string"},
			},
		},
	}
	handler := protocol.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		message, _ := params["message"].(string)
		// Sanitize output
		sanitized := sanitizeForJSON(message)
		return map[string]string{"echo": sanitized}, nil
	})
	s.AddTool(tool, handler)

	xssPayloads := []string{
		`<script>alert('XSS')</script>`,
		`<img src=x onerror=alert('XSS')>`,
		`javascript:alert('XSS')`,
		`<iframe src="javascript:alert('XSS')"></iframe>`,
		`<svg onload=alert('XSS')>`,
	}

	for _, payload := range xssPayloads {
		t.Run(payload, func(t *testing.T) {
			req := protocol.JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "tools/call",
				Params:  json.RawMessage(fmt.Sprintf(`{"name":"echo","arguments":{"message":%q}}`, payload)),
			}

			resp := s.HandleRequest(context.Background(), &req)
			if resp.Error != nil {
				t.Fatalf("unexpected error: %v", resp.Error)
			}

			// Check that response doesn't contain raw HTML
			respJSON, _ := json.Marshal(resp)
			if bytes.Contains(respJSON, []byte("<script>")) || 
			   bytes.Contains(respJSON, []byte("<img")) ||
			   bytes.Contains(respJSON, []byte("<iframe")) {
				t.Error("response contains unsanitized HTML")
			}
		})
	}
}

// Helper functions

func isPathSafe(path string) bool {
	// Check for path traversal patterns
	dangerous := []string{
		"..",
		"..\\",
		"..%2F",
		"..%5C",
		"%2e%2e",
		"\x00",
	}
	
	for _, pattern := range dangerous {
		if strings.Contains(path, pattern) {
			return false
		}
	}
	
	return true
}

func sanitizeForJSON(input string) string {
	// Basic HTML entity encoding
	replacer := strings.NewReplacer(
		"<", "&lt;",
		">", "&gt;",
		"&", "&amp;",
		"\"", "&quot;",
		"'", "&#x27;",
		"/", "&#x2F;",
	)
	return replacer.Replace(input)
}

// FuzzJSONRPC implements fuzzing for JSON-RPC parsing
func FuzzJSONRPC(f *testing.F) {
	// Add seed corpus
	f.Add([]byte(`{"jsonrpc":"2.0","method":"test","params":{},"id":1}`))
	f.Add([]byte(`{"jsonrpc":"2.0","method":"test","id":1}`))
	f.Add([]byte(`{"jsonrpc":"2.0","method":"test","params":null,"id":1}`))
	
	f.Fuzz(func(t *testing.T, data []byte) {
		var req protocol.JSONRPCRequest
		// Should not panic
		json.Unmarshal(data, &req)
		
		// If it parsed successfully, validate it
		if req.JSONRPC != "" && req.JSONRPC != "2.0" {
			t.Skip("Invalid JSON-RPC version")
		}
	})
}

// FuzzParameterValidation implements fuzzing for parameter validation
func FuzzParameterValidation(f *testing.F) {
	// Add seed corpus
	f.Add([]byte(`{"name":"test","value":"hello"}`))
	f.Add([]byte(`{"count":123,"items":["a","b","c"]}`))
	f.Add([]byte(`{"nested":{"deep":{"value":42}}}`))
	
	f.Fuzz(func(t *testing.T, data []byte) {
		// Should not panic when validating parameters
		var params map[string]interface{}
		if err := json.Unmarshal(data, &params); err != nil {
			return // Invalid JSON is ok to skip
		}
		
		// Validate parameter types
		for k, v := range params {
			// Check for dangerous keys
			if strings.Contains(k, "\x00") || strings.Contains(k, "..") {
				t.Errorf("dangerous key detected: %q", k)
			}
			
			// Check for oversized values
			if str, ok := v.(string); ok && len(str) > 1024*1024 {
				t.Error("oversized string parameter")
			}
		}
	})
}