package main

import (
	"context"
	"encoding/json"
	"log"
	"mcp-memory/pkg/mcp/protocol"
	"mcp-memory/pkg/mcp/server"
)

func TestFeatures() {
	log.Println("🧪 Testing MCP Features Directly")
	log.Println("================================")
	
	// Create extended server
	srv := server.NewExtendedServer("Test Server", "1.0.0")
	
	// Test 1: Initialize
	log.Println("\n1️⃣ Testing Initialize...")
	req := &protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0.0",
			},
			"capabilities": map[string]interface{}{},
		},
	}
	
	ctx := context.Background()
	resp := srv.HandleRequest(ctx, req)
	printResponse("Initialize", resp)
	
	// Test 2: List roots
	log.Println("\n2️⃣ Testing Roots...")
	req = &protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "roots/list",
		Params:  nil,
	}
	resp = srv.HandleRequest(ctx, req)
	printResponse("List Roots", resp)
	
	// Test 3: Sampling
	log.Println("\n3️⃣ Testing Sampling...")
	req = &protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "sampling/createMessage",
		Params: map[string]interface{}{
			"messages": []map[string]interface{}{
				{
					"role": "user",
					"content": map[string]interface{}{
						"type": "text",
						"text": "Hello, MCP!",
					},
				},
			},
			"maxTokens": 100,
		},
	}
	resp = srv.HandleRequest(ctx, req)
	printResponse("Sampling", resp)
	
	// Test 4: Tools
	log.Println("\n4️⃣ Testing Tools...")
	
	// Add a test tool
	srv.AddTool(
		protocol.Tool{
			Name:        "test_tool",
			Description: "A test tool",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		protocol.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return protocol.NewToolCallResult(protocol.NewContent("Tool executed!")), nil
		}),
	)
	
	// List tools
	req = &protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      4,
		Method:  "tools/list",
		Params:  nil,
	}
	resp = srv.HandleRequest(ctx, req)
	printResponse("List Tools", resp)
	
	// Call tool
	req = &protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      5,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      "test_tool",
			"arguments": map[string]interface{}{},
		},
	}
	resp = srv.HandleRequest(ctx, req)
	printResponse("Call Tool", resp)
	
	log.Println("\n✅ All tests completed!")
}

func printResponse(name string, resp *protocol.JSONRPCResponse) {
	if resp.Error != nil {
		log.Printf("❌ %s failed: %s (code: %d)", name, resp.Error.Message, resp.Error.Code)
		if resp.Error.Data != nil {
			log.Printf("   Error data: %v", resp.Error.Data)
		}
	} else {
		data, _ := json.MarshalIndent(resp.Result, "   ", "  ")
		log.Printf("✅ %s succeeded:", name)
		log.Printf("   %s", string(data))
	}
}

func main() {
	TestFeatures()
}