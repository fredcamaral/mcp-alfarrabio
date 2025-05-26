package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// TestClient tests all MCP features
type TestClient struct {
	baseURL string
	client  *http.Client
}

// NewTestClient creates a new test client
func NewTestClient(baseURL string) *TestClient {
	return &TestClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// sendRequest sends a JSON-RPC request
func (c *TestClient) sendRequest(method string, params interface{}) (json.RawMessage, error) {
	reqID := time.Now().UnixNano()
	
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      reqID,
		"method":  method,
		"params":  params,
	}
	
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	
	resp, err := c.client.Post(c.baseURL+"/rpc", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	var response struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      interface{}     `json:"id"`
		Result  json.RawMessage `json:"result"`
		Error   *struct {
			Code    int         `json:"code"`
			Message string      `json:"message"`
			Data    interface{} `json:"data"`
		} `json:"error"`
	}
	
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	if response.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s (data: %v)", response.Error.Code, response.Error.Message, response.Error.Data)
	}
	
	return response.Result, nil
}

// RunTests runs all feature tests
func (c *TestClient) RunTests() {
	fmt.Println("🧪 MCP Feature Test Suite")
	fmt.Println("========================")
	
	// Test 1: Initialize with different client profiles
	c.testInitialize()
	
	// Test 2: Tools
	c.testTools()
	
	// Test 3: Resources
	c.testResources()
	
	// Test 4: Prompts
	c.testPrompts()
	
	// Test 5: Roots
	c.testRoots()
	
	// Test 6: Sampling
	c.testSampling()
	
	// Test 7: Discovery
	c.testDiscovery()
	
	// Test 8: Subscriptions
	c.testSubscriptions()
}

func (c *TestClient) testInitialize() {
	fmt.Println("\n1️⃣ Testing Initialize...")
	
	// Test as Claude Desktop
	params := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"experimental": map[string]interface{}{},
		},
		"clientInfo": map[string]interface{}{
			"name":    "claude-desktop",
			"version": "1.0.0",
		},
	}
	
	result, err := c.sendRequest("initialize", params)
	if err != nil {
		fmt.Printf("❌ Initialize failed: %v\n", err)
		return
	}
	
	var initResult map[string]interface{}
	json.Unmarshal(result, &initResult)
	fmt.Printf("✅ Initialized as Claude Desktop\n")
	fmt.Printf("   Server: %v\n", initResult["serverInfo"])
	fmt.Printf("   Capabilities: %v\n", initResult["capabilities"])
}

func (c *TestClient) testTools() {
	fmt.Println("\n2️⃣ Testing Tools...")
	
	// List tools
	result, err := c.sendRequest("tools/list", nil)
	if err != nil {
		fmt.Printf("❌ List tools failed: %v\n", err)
		return
	}
	
	var tools map[string]interface{}
	json.Unmarshal(result, &tools)
	fmt.Printf("✅ Found %d tools\n", len(tools["tools"].([]interface{})))
	
	// Call echo tool
	params := map[string]interface{}{
		"name": "echo",
		"arguments": map[string]interface{}{
			"message": "Hello, MCP!",
		},
	}
	
	result, err = c.sendRequest("tools/call", params)
	if err != nil {
		fmt.Printf("❌ Tool call failed: %v\n", err)
		return
	}
	
	fmt.Printf("✅ Tool call result: %s\n", string(result))
}

func (c *TestClient) testResources() {
	fmt.Println("\n3️⃣ Testing Resources...")
	
	// List resources
	result, err := c.sendRequest("resources/list", nil)
	if err != nil {
		fmt.Printf("❌ List resources failed: %v\n", err)
		return
	}
	
	var resources map[string]interface{}
	json.Unmarshal(result, &resources)
	fmt.Printf("✅ Found %d resources\n", len(resources["resources"].([]interface{})))
	
	// Read resource
	params := map[string]interface{}{
		"uri": "demo://test.txt",
	}
	
	result, err = c.sendRequest("resources/read", params)
	if err != nil {
		fmt.Printf("❌ Resource read failed: %v\n", err)
		return
	}
	
	fmt.Printf("✅ Resource content received\n")
}

func (c *TestClient) testPrompts() {
	fmt.Println("\n4️⃣ Testing Prompts...")
	
	// List prompts
	result, err := c.sendRequest("prompts/list", nil)
	if err != nil {
		fmt.Printf("❌ List prompts failed: %v\n", err)
		return
	}
	
	var prompts map[string]interface{}
	json.Unmarshal(result, &prompts)
	fmt.Printf("✅ Found %d prompts\n", len(prompts["prompts"].([]interface{})))
	
	// Get prompt
	params := map[string]interface{}{
		"name": "greeting",
		"arguments": map[string]interface{}{
			"name":  "Alice",
			"style": "formal",
		},
	}
	
	result, err = c.sendRequest("prompts/get", params)
	if err != nil {
		fmt.Printf("❌ Get prompt failed: %v\n", err)
		return
	}
	
	fmt.Printf("✅ Prompt result received\n")
}

func (c *TestClient) testRoots() {
	fmt.Println("\n5️⃣ Testing Roots...")
	
	// List roots
	result, err := c.sendRequest("roots/list", nil)
	if err != nil {
		fmt.Printf("❌ List roots failed: %v\n", err)
		// Check if it's an unsupported feature error
		if err.Error() == "RPC error -32601: Method not found (data: <nil>)" {
			fmt.Printf("ℹ️  Roots not implemented in base server\n")
		}
		return
	}
	
	var roots map[string]interface{}
	json.Unmarshal(result, &roots)
	fmt.Printf("✅ Found %d roots\n", len(roots["roots"].([]interface{})))
}

func (c *TestClient) testSampling() {
	fmt.Println("\n6️⃣ Testing Sampling...")
	
	// Create message
	params := map[string]interface{}{
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": map[string]interface{}{
					"type": "text",
					"text": "What is MCP?",
				},
			},
		},
		"maxTokens": 100,
	}
	
	result, err := c.sendRequest("sampling/createMessage", params)
	if err != nil {
		fmt.Printf("❌ Sampling failed: %v\n", err)
		if err.Error() == "RPC error -32601: Method not found (data: <nil>)" {
			fmt.Printf("ℹ️  Sampling not implemented in base server\n")
		}
		return
	}
	
	fmt.Printf("✅ Sampling response received\n")
	fmt.Printf("   Response: %s\n", string(result))
}

func (c *TestClient) testDiscovery() {
	fmt.Println("\n7️⃣ Testing Discovery...")
	
	// Discover with filter
	params := map[string]interface{}{
		"filter": map[string]interface{}{
			"available": true,
		},
	}
	
	result, err := c.sendRequest("discovery/discover", params)
	if err != nil {
		fmt.Printf("❌ Discovery failed: %v\n", err)
		if err.Error() == "RPC error -32601: Method not found (data: <nil>)" {
			fmt.Printf("ℹ️  Discovery not implemented in base server\n")
		}
		return
	}
	
	var discovery map[string]interface{}
	json.Unmarshal(result, &discovery)
	fmt.Printf("✅ Discovery completed\n")
}

func (c *TestClient) testSubscriptions() {
	fmt.Println("\n8️⃣ Testing Subscriptions...")
	
	// Subscribe to tools list changes
	params := map[string]interface{}{
		"method": "tools/subscribe",
	}
	
	result, err := c.sendRequest("tools/subscribe", params)
	if err != nil {
		fmt.Printf("❌ Subscription failed: %v\n", err)
		if err.Error() == "RPC error -32601: Method not found (data: <nil>)" {
			fmt.Printf("ℹ️  Subscriptions not implemented in base server\n")
		}
		return
	}
	
	var subResponse map[string]interface{}
	json.Unmarshal(result, &subResponse)
	fmt.Printf("✅ Subscribed with ID: %v\n", subResponse["subscriptionId"])
}

// RunClientTest runs the test client
func RunClientTest() {
	client := NewTestClient("http://localhost:3000")
	client.RunTests()
	fmt.Println("\n✨ Test suite completed!")
}