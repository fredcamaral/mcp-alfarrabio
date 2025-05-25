package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mcp-memory/pkg/mcp/protocol"
)

func TestRESTTransport_StartStop(t *testing.T) {
	config := &RESTConfig{
		HTTPConfig: HTTPConfig{
			Address: "localhost:0",
		},
		APIPrefix: "/api/v1",
	}

	transport := NewRESTTransport(config)
	handler := &mockHandler{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start transport
	err := transport.Start(ctx, handler)
	require.NoError(t, err)
	assert.True(t, transport.IsRunning())

	// Stop transport
	err = transport.Stop()
	require.NoError(t, err)
	assert.False(t, transport.IsRunning())
}

func TestRESTTransport_ToolsEndpoint(t *testing.T) {
	config := &RESTConfig{
		HTTPConfig: HTTPConfig{
			Address: "localhost:0",
		},
		APIPrefix: "/api/v1",
	}

	transport := NewRESTTransport(config)
	
	handler := &mockHandler{
		handleFunc: func(ctx context.Context, req *protocol.JSONRPCRequest) *protocol.JSONRPCResponse {
			switch req.Method {
			case "tools/list":
				return &protocol.JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result: map[string]interface{}{
						"tools": []interface{}{
							map[string]interface{}{
								"name":        "test_tool",
								"description": "A test tool",
							},
						},
					},
				}
			case "tools/call":
				params := req.Params.(map[string]interface{})
				return &protocol.JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result: map[string]interface{}{
						"content": []interface{}{
							map[string]interface{}{
								"type": "text",
								"text": "Tool " + params["name"].(string) + " called",
							},
						},
					},
				}
			default:
				return &protocol.JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error:   protocol.NewJSONRPCError(protocol.MethodNotFound, "Method not found", nil),
				}
			}
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	require.NoError(t, err)
	defer transport.Stop()

	addr := transport.server.Addr
	baseURL := "http://" + addr + "/api/v1"
	client := &http.Client{Timeout: 5 * time.Second}

	t.Run("list tools", func(t *testing.T) {
		resp, err := client.Get(baseURL + "/tools")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		tools := result["tools"].([]interface{})
		assert.Len(t, tools, 1)
		
		tool := tools[0].(map[string]interface{})
		assert.Equal(t, "test_tool", tool["name"])
	})

	t.Run("call tool", func(t *testing.T) {
		args := map[string]interface{}{
			"param1": "value1",
		}
		
		body, _ := json.Marshal(args)
		resp, err := client.Post(baseURL+"/tools/test_tool", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		content := result["content"].([]interface{})
		assert.Len(t, content, 1)
		
		item := content[0].(map[string]interface{})
		assert.Equal(t, "Tool test_tool called", item["text"])
	})

	t.Run("invalid method", func(t *testing.T) {
		resp, err := client.Post(baseURL+"/tools", "application/json", nil)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
}

func TestRESTTransport_ResourcesEndpoint(t *testing.T) {
	config := &RESTConfig{
		HTTPConfig: HTTPConfig{
			Address: "localhost:0",
		},
		APIPrefix: "/api/v1",
	}

	transport := NewRESTTransport(config)
	
	handler := &mockHandler{
		handleFunc: func(ctx context.Context, req *protocol.JSONRPCRequest) *protocol.JSONRPCResponse {
			switch req.Method {
			case "resources/list":
				return &protocol.JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result: map[string]interface{}{
						"resources": []interface{}{
							map[string]interface{}{
								"uri":  "file:///test.txt",
								"name": "test.txt",
							},
						},
					},
				}
			case "resources/read":
				params := req.Params.(map[string]interface{})
				return &protocol.JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result: map[string]interface{}{
						"contents": []interface{}{
							map[string]interface{}{
								"uri":      params["uri"],
								"mimeType": "text/plain",
								"text":     "Resource content",
							},
						},
					},
				}
			default:
				return &protocol.JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error:   protocol.NewJSONRPCError(protocol.MethodNotFound, "Method not found", nil),
				}
			}
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	require.NoError(t, err)
	defer transport.Stop()

	addr := transport.server.Addr
	baseURL := "http://" + addr + "/api/v1"
	client := &http.Client{Timeout: 5 * time.Second}

	t.Run("list resources", func(t *testing.T) {
		resp, err := client.Get(baseURL + "/resources")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		resources := result["resources"].([]interface{})
		assert.Len(t, resources, 1)
	})

	t.Run("read resource", func(t *testing.T) {
		resp, err := client.Get(baseURL + "/resources/file:///test.txt")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		contents := result["contents"].([]interface{})
		assert.Len(t, contents, 1)
		
		content := contents[0].(map[string]interface{})
		assert.Equal(t, "Resource content", content["text"])
	})
}

func TestRESTTransport_Authentication(t *testing.T) {
	apiKey := "test-api-key-123"
	
	config := &RESTConfig{
		HTTPConfig: HTTPConfig{
			Address: "localhost:0",
		},
		APIPrefix: "/api/v1",
		APIKey:    apiKey,
	}

	transport := NewRESTTransport(config)
	handler := &mockHandler{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	require.NoError(t, err)
	defer transport.Stop()

	addr := transport.server.Addr
	baseURL := "http://" + addr + "/api/v1"
	client := &http.Client{Timeout: 5 * time.Second}

	t.Run("no api key", func(t *testing.T) {
		resp, err := client.Get(baseURL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("invalid api key", func(t *testing.T) {
		req, _ := http.NewRequest("GET", baseURL+"/health", nil)
		req.Header.Set("X-API-Key", "wrong-key")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("valid api key in header", func(t *testing.T) {
		req, _ := http.NewRequest("GET", baseURL+"/health", nil)
		req.Header.Set("X-API-Key", apiKey)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("valid api key in query", func(t *testing.T) {
		resp, err := client.Get(baseURL + "/health?api_key=" + apiKey)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestRESTTransport_RateLimiting(t *testing.T) {
	config := &RESTConfig{
		HTTPConfig: HTTPConfig{
			Address: "localhost:0",
		},
		APIPrefix: "/api/v1",
		RateLimit: 5, // 5 requests per minute
	}

	transport := NewRESTTransport(config)
	handler := &mockHandler{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	require.NoError(t, err)
	defer transport.Stop()

	addr := transport.server.Addr
	baseURL := "http://" + addr + "/api/v1"
	client := &http.Client{Timeout: 5 * time.Second}

	// Make allowed requests
	for i := 0; i < 5; i++ {
		resp, err := client.Get(baseURL + "/health")
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}

	// Next request should be rate limited
	resp, err := client.Get(baseURL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
}

func TestRESTTransport_ErrorMapping(t *testing.T) {
	config := &RESTConfig{
		HTTPConfig: HTTPConfig{
			Address: "localhost:0",
		},
		APIPrefix: "/api/v1",
	}

	transport := NewRESTTransport(config)
	
	handler := &mockHandler{
		handleFunc: func(ctx context.Context, req *protocol.JSONRPCRequest) *protocol.JSONRPCResponse {
			// Return different error codes based on method
			var errorCode int
			var message string
			
			switch req.Method {
			case "tools/parse_error":
				errorCode = protocol.ParseError
				message = "Parse error"
			case "tools/invalid_request":
				errorCode = protocol.InvalidRequest
				message = "Invalid request"
			case "tools/method_not_found":
				errorCode = protocol.MethodNotFound
				message = "Method not found"
			case "tools/invalid_params":
				errorCode = protocol.InvalidParams
				message = "Invalid params"
			default:
				errorCode = protocol.InternalError
				message = "Internal error"
			}
			
			return &protocol.JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   protocol.NewJSONRPCError(errorCode, message, nil),
			}
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	require.NoError(t, err)
	defer transport.Stop()

	addr := transport.server.Addr
	baseURL := "http://" + addr + "/api/v1"
	client := &http.Client{Timeout: 5 * time.Second}

	tests := []struct {
		toolName       string
		expectedStatus int
		expectedCode   int
	}{
		{"parse_error", http.StatusBadRequest, protocol.ParseError},
		{"invalid_request", http.StatusBadRequest, protocol.InvalidRequest},
		{"method_not_found", http.StatusNotFound, protocol.MethodNotFound},
		{"invalid_params", http.StatusBadRequest, protocol.InvalidParams},
		{"internal_error", http.StatusInternalServerError, protocol.InternalError},
	}

	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			resp, err := client.Post(baseURL+"/tools/"+tt.toolName, "application/json", bytes.NewReader([]byte("{}")))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)

			errorObj := result["error"].(map[string]interface{})
			assert.Equal(t, float64(tt.expectedCode), errorObj["code"])
		})
	}
}

func TestRESTTransport_HealthEndpoint(t *testing.T) {
	config := &RESTConfig{
		HTTPConfig: HTTPConfig{
			Address: "localhost:0",
		},
		APIPrefix: "/api/v1",
	}

	transport := NewRESTTransport(config)
	handler := &mockHandler{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	require.NoError(t, err)
	defer transport.Stop()

	addr := transport.server.Addr
	baseURL := "http://" + addr + "/api/v1"
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(baseURL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var health map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&health)
	require.NoError(t, err)

	assert.Equal(t, "healthy", health["status"])
	assert.NotNil(t, health["timestamp"])
}

func TestRESTTransport_OpenAPIDocumentation(t *testing.T) {
	config := &RESTConfig{
		HTTPConfig: HTTPConfig{
			Address: "localhost:0",
		},
		APIPrefix:  "/api/v1",
		EnableDocs: true,
	}

	transport := NewRESTTransport(config)
	handler := &mockHandler{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	require.NoError(t, err)
	defer transport.Stop()

	addr := transport.server.Addr
	baseURL := "http://" + addr + "/api/v1"
	client := &http.Client{Timeout: 5 * time.Second}

	t.Run("openapi spec", func(t *testing.T) {
		resp, err := client.Get(baseURL + "/openapi.json")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		var spec map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&spec)
		require.NoError(t, err)

		assert.Equal(t, "3.0.0", spec["openapi"])
		
		info := spec["info"].(map[string]interface{})
		assert.Equal(t, "MCP REST API", info["title"])
		
		paths := spec["paths"].(map[string]interface{})
		assert.NotNil(t, paths["/tools"])
		assert.NotNil(t, paths["/health"])
	})

	t.Run("swagger ui", func(t *testing.T) {
		resp, err := client.Get(baseURL + "/docs")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "text/html", resp.Header.Get("Content-Type"))

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		// Check for Swagger UI elements
		assert.Contains(t, string(body), "swagger-ui")
		assert.Contains(t, string(body), "/api/v1/openapi.json")
	})
}

// Benchmark tests

func BenchmarkRESTTransport_ToolCall(b *testing.B) {
	config := &RESTConfig{
		HTTPConfig: HTTPConfig{
			Address: "localhost:0",
		},
		APIPrefix: "/api/v1",
	}

	transport := NewRESTTransport(config)
	handler := &mockHandler{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	require.NoError(b, err)
	defer transport.Stop()

	addr := transport.server.Addr
	baseURL := "http://" + addr + "/api/v1"
	
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
		},
	}

	args := map[string]interface{}{
		"data": "benchmark test data",
	}
	body, _ := json.Marshal(args)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Post(baseURL+"/tools/benchmark", "application/json", bytes.NewReader(body))
			if err != nil {
				b.Fatal(err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}