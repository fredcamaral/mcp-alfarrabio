package mcp_test

import (
	"context"
	"encoding/json"
	"testing"
	"mcp-memory/pkg/mcp"
	"mcp-memory/pkg/mcp/protocol"
	"mcp-memory/pkg/mcp/server"
)

// BenchmarkJSONEncoding tests JSON encoding performance
func BenchmarkJSONEncoding(b *testing.B) {
	req := &protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "tools/call",
		Params: json.RawMessage(`{
			"name": "test_tool",
			"arguments": {
				"param1": "value1",
				"param2": 42,
				"param3": true,
				"param4": {"nested": "object"},
				"param5": [1, 2, 3, 4, 5]
			}
		}`),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkJSONDecoding tests JSON decoding performance
func BenchmarkJSONDecoding(b *testing.B) {
	data := []byte(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "test_tool",
			"arguments": {
				"param1": "value1",
				"param2": 42,
				"param3": true,
				"param4": {"nested": "object"},
				"param5": [1, 2, 3, 4, 5]
			}
		}
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var req protocol.JSONRPCRequest
		err := json.Unmarshal(data, &req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkToolExecution tests tool execution performance
func BenchmarkToolExecution(b *testing.B) {
	s := server.NewServer("bench-server", "1.0.0")
	
	tool := protocol.Tool{
		Name:        "bench_tool",
		Description: "Benchmark tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"value": map[string]interface{}{
					"type": "string",
				},
			},
		},
	}

	handler := mcp.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		// Simulate some work
		result := make(map[string]interface{})
		if val, ok := params["value"].(string); ok {
			result["processed"] = val + "_processed"
		}
		return result, nil
	})

	s.AddTool(tool, handler)

	req := &protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "tools/call",
		Params: json.RawMessage(`{"name": "bench_tool", "arguments": {"value": "test"}}`),
	}

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := s.HandleRequest(ctx, req)
		if resp.Error != nil {
			b.Fatal(resp.Error)
		}
	}
}

// BenchmarkConcurrentRequests tests concurrent request handling
func BenchmarkConcurrentRequests(b *testing.B) {
	s := server.NewServer("bench-server", "1.0.0")
	
	tool := protocol.Tool{
		Name:        "concurrent_tool",
		Description: "Concurrent benchmark tool",
		InputSchema: map[string]interface{}{
			"type": "object",
		},
	}

	handler := mcp.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		return map[string]interface{}{"status": "ok"}, nil
	})

	s.AddTool(tool, handler)

	req := &protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "tools/call",
		Params: json.RawMessage(`{"name": "concurrent_tool", "arguments": {}}`),
	}

	ctx := context.Background()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp := s.HandleRequest(ctx, req)
			if resp.Error != nil {
				b.Fatal(resp.Error)
			}
		}
	})
}

// BenchmarkMemoryAllocation tests memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("SmallRequest", func(b *testing.B) {
		req := &protocol.JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  "ping",
		}
		
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			data, _ := json.Marshal(req)
			var decoded protocol.JSONRPCRequest
			json.Unmarshal(data, &decoded)
		}
	})

	b.Run("LargeRequest", func(b *testing.B) {
		largeParams := make(map[string]interface{})
		for i := 0; i < 100; i++ {
			largeParams[string(rune('a'+i))] = "value" + string(rune('0'+i))
		}
		paramsJSON, _ := json.Marshal(largeParams)
		
		req := &protocol.JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  "tools/call",
			Params:  paramsJSON,
		}
		
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			data, _ := json.Marshal(req)
			var decoded protocol.JSONRPCRequest
			json.Unmarshal(data, &decoded)
		}
	})
}

// BenchmarkRequestParsing tests request parsing performance
func BenchmarkRequestParsing(b *testing.B) {
	requests := [][]byte{
		[]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"capabilities":{}}}`),
		[]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`),
		[]byte(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"test","arguments":{"key":"value"}}}`),
		[]byte(`{"jsonrpc":"2.0","id":4,"method":"resources/list"}`),
		[]byte(`{"jsonrpc":"2.0","id":5,"method":"prompts/list"}`),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := requests[i%len(requests)]
		var parsed protocol.JSONRPCRequest
		if err := json.Unmarshal(req, &parsed); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkToolRegistry tests tool registry lookup performance
func BenchmarkToolRegistry(b *testing.B) {
	s := server.NewServer("bench-server", "1.0.0")
	
	// Add many tools to test lookup performance
	for i := 0; i < 100; i++ {
		tool := protocol.Tool{
			Name:        "tool_" + string(rune('a'+i%26)) + string(rune('0'+i/26)),
			Description: "Test tool",
			InputSchema: map[string]interface{}{"type": "object"},
		}
		handler := mcp.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return nil, nil
		})
		s.AddTool(tool, handler)
	}

	// Prepare requests for different tools
	toolNames := []string{"tool_a0", "tool_m1", "tool_z3", "tool_d0", "tool_p2"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		toolName := toolNames[i%len(toolNames)]
		params, _ := json.Marshal(map[string]interface{}{
			"name":      toolName,
			"arguments": map[string]interface{}{},
		})
		
		req := &protocol.JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  "tools/call",
			Params:  params,
		}
		
		resp := s.HandleRequest(context.Background(), req)
		if resp.Error != nil && resp.Error.Code != -32601 { // Method not found is expected
			b.Fatal(resp.Error)
		}
	}
}