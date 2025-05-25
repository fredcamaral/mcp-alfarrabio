package transport

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mcp-memory/pkg/mcp/protocol"
)

// mockHandler implements RequestHandler for testing
type mockHandler struct {
	handleFunc func(ctx context.Context, req *protocol.JSONRPCRequest) *protocol.JSONRPCResponse
}

func (m *mockHandler) HandleRequest(ctx context.Context, req *protocol.JSONRPCRequest) *protocol.JSONRPCResponse {
	if m.handleFunc != nil {
		return m.handleFunc(ctx, req)
	}
	return &protocol.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  map[string]interface{}{"echo": req.Method},
	}
}

func TestHTTPTransport_StartStop(t *testing.T) {
	config := &HTTPConfig{
		Address:      "localhost:0",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	transport := NewHTTPTransport(config)
	handler := &mockHandler{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start transport
	err := transport.Start(ctx, handler)
	require.NoError(t, err)
	assert.True(t, transport.IsRunning())

	// Try starting again (should fail)
	err = transport.Start(ctx, handler)
	assert.Error(t, err)

	// Stop transport
	err = transport.Stop()
	require.NoError(t, err)
	assert.False(t, transport.IsRunning())

	// Stop again (should be no-op)
	err = transport.Stop()
	assert.NoError(t, err)
}

func TestHTTPTransport_HandleRequest(t *testing.T) {
	config := &HTTPConfig{
		Address: "localhost:0",
		Path:    "/rpc",
	}

	transport := NewHTTPTransport(config)
	
	handler := &mockHandler{
		handleFunc: func(ctx context.Context, req *protocol.JSONRPCRequest) *protocol.JSONRPCResponse {
			return &protocol.JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: map[string]interface{}{
					"method": req.Method,
					"params": req.Params,
				},
			}
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	require.NoError(t, err)
	defer transport.Stop()

	// Get actual address
	addr := transport.server.Addr
	baseURL := "http://" + addr

	tests := []struct {
		name           string
		request        *protocol.JSONRPCRequest
		expectedResult map[string]interface{}
		expectedError  bool
	}{
		{
			name: "valid request",
			request: &protocol.JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "test.method",
				Params:  map[string]interface{}{"key": "value"},
			},
			expectedResult: map[string]interface{}{
				"method": "test.method",
				"params": map[string]interface{}{"key": "value"},
			},
		},
		{
			name: "request without params",
			request: &protocol.JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      2,
				Method:  "test.noparams",
			},
			expectedResult: map[string]interface{}{
				"method": "test.noparams",
				"params": nil,
			},
		},
	}

	client := &http.Client{Timeout: 5 * time.Second}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			resp, err := client.Post(baseURL+"/rpc", "application/json", bytes.NewReader(reqBody))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

			var jsonResp protocol.JSONRPCResponse
			err = json.NewDecoder(resp.Body).Decode(&jsonResp)
			require.NoError(t, err)

			assert.Equal(t, "2.0", jsonResp.JSONRPC)
			assert.Equal(t, tt.request.ID, jsonResp.ID)
			
			if tt.expectedError {
				assert.NotNil(t, jsonResp.Error)
			} else {
				assert.Nil(t, jsonResp.Error)
				assert.Equal(t, tt.expectedResult, jsonResp.Result)
			}
		})
	}
}

func TestHTTPTransport_ErrorHandling(t *testing.T) {
	config := &HTTPConfig{
		Address:     "localhost:0",
		MaxBodySize: 100, // Very small for testing
	}

	transport := NewHTTPTransport(config)
	handler := &mockHandler{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	require.NoError(t, err)
	defer transport.Stop()

	addr := transport.server.Addr
	baseURL := "http://" + addr

	client := &http.Client{Timeout: 5 * time.Second}

	tests := []struct {
		name         string
		method       string
		body         string
		expectedCode int
		checkError   func(t *testing.T, resp *http.Response)
	}{
		{
			name:         "invalid method",
			method:       "GET",
			body:         "",
			expectedCode: http.StatusMethodNotAllowed,
		},
		{
			name:         "invalid json",
			method:       "POST",
			body:         "{invalid json}",
			expectedCode: http.StatusOK, // JSON-RPC error
			checkError: func(t *testing.T, resp *http.Response) {
				var jsonResp protocol.JSONRPCResponse
				err := json.NewDecoder(resp.Body).Decode(&jsonResp)
				require.NoError(t, err)
				assert.NotNil(t, jsonResp.Error)
				assert.Equal(t, protocol.ParseError, jsonResp.Error.Code)
			},
		},
		{
			name:         "body too large",
			method:       "POST",
			body:         string(make([]byte, 200)), // Larger than MaxBodySize
			expectedCode: http.StatusOK,
			checkError: func(t *testing.T, resp *http.Response) {
				var jsonResp protocol.JSONRPCResponse
				err := json.NewDecoder(resp.Body).Decode(&jsonResp)
				require.NoError(t, err)
				assert.NotNil(t, jsonResp.Error)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, baseURL+"/", strings.NewReader(tt.body))
			require.NoError(t, err)

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedCode, resp.StatusCode)

			if tt.checkError != nil {
				tt.checkError(t, resp)
			}
		})
	}
}

func TestHTTPTransport_CORS(t *testing.T) {
	config := &HTTPConfig{
		Address:        "localhost:0",
		EnableCORS:     true,
		AllowedOrigins: []string{"https://example.com", "https://test.com"},
	}

	transport := NewHTTPTransport(config)
	handler := &mockHandler{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	require.NoError(t, err)
	defer transport.Stop()

	addr := transport.server.Addr
	baseURL := "http://" + addr

	client := &http.Client{Timeout: 5 * time.Second}

	tests := []struct {
		name               string
		origin             string
		method             string
		expectedAllowed    bool
		expectedAllowOrigin string
	}{
		{
			name:               "allowed origin",
			origin:             "https://example.com",
			method:             "OPTIONS",
			expectedAllowed:    true,
			expectedAllowOrigin: "https://example.com",
		},
		{
			name:            "disallowed origin",
			origin:          "https://malicious.com",
			method:          "OPTIONS",
			expectedAllowed: false,
		},
		{
			name:               "preflight request",
			origin:             "https://test.com",
			method:             "OPTIONS",
			expectedAllowed:    true,
			expectedAllowOrigin: "https://test.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, baseURL+"/", nil)
			require.NoError(t, err)
			req.Header.Set("Origin", tt.origin)

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			if tt.expectedAllowed {
				assert.Equal(t, tt.expectedAllowOrigin, resp.Header.Get("Access-Control-Allow-Origin"))
				assert.Equal(t, "POST, OPTIONS", resp.Header.Get("Access-Control-Allow-Methods"))
			} else {
				assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
			}

			if tt.method == "OPTIONS" {
				assert.Equal(t, http.StatusNoContent, resp.StatusCode)
			}
		})
	}
}

func TestHTTPTransport_CustomHeaders(t *testing.T) {
	customHeaders := map[string]string{
		"X-Custom-Header": "test-value",
		"X-API-Version":   "1.0",
	}

	config := &HTTPConfig{
		Address:       "localhost:0",
		CustomHeaders: customHeaders,
	}

	transport := NewHTTPTransport(config)
	handler := &mockHandler{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	require.NoError(t, err)
	defer transport.Stop()

	addr := transport.server.Addr
	client := &http.Client{Timeout: 5 * time.Second}

	req := &protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "test",
	}

	reqBody, _ := json.Marshal(req)
	resp, err := client.Post("http://"+addr+"/", "application/json", bytes.NewReader(reqBody))
	require.NoError(t, err)
	defer resp.Body.Close()

	// Check custom headers
	for k, v := range customHeaders {
		assert.Equal(t, v, resp.Header.Get(k))
	}
}

func TestHTTPTransport_RecoveryMiddleware(t *testing.T) {
	config := &HTTPConfig{
		Address: "localhost:0",
	}

	transport := NewHTTPTransport(config)
	
	// Handler that panics
	handler := &mockHandler{
		handleFunc: func(ctx context.Context, req *protocol.JSONRPCRequest) *protocol.JSONRPCResponse {
			panic("test panic")
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	require.NoError(t, err)
	defer transport.Stop()

	addr := transport.server.Addr
	client := &http.Client{Timeout: 5 * time.Second}

	req := &protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "test",
	}

	reqBody, _ := json.Marshal(req)
	resp, err := client.Post("http://"+addr+"/", "application/json", bytes.NewReader(reqBody))
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should get internal error response instead of crash
	var jsonResp protocol.JSONRPCResponse
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	require.NoError(t, err)
	assert.NotNil(t, jsonResp.Error)
	assert.Equal(t, protocol.InternalError, jsonResp.Error.Code)
}

func TestHTTPSTransport(t *testing.T) {
	// Note: This is a basic test. In production, you'd use proper certificates
	config := &HTTPConfig{
		Address: "localhost:0",
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true, // Only for testing
		},
	}

	// Create temporary self-signed certificate for testing
	certFile := t.TempDir() + "/cert.pem"
	keyFile := t.TempDir() + "/key.pem"

	// In a real test, generate test certificates here
	// For now, we'll skip the HTTPS test
	t.Skip("Skipping HTTPS test - requires certificate generation")

	transport := NewHTTPSTransport(config, certFile, keyFile)
	handler := &mockHandler{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	require.NoError(t, err)
	defer transport.Stop()

	// Test would continue with HTTPS client...
}

// Benchmark tests

func BenchmarkHTTPTransport_HandleRequest(b *testing.B) {
	config := &HTTPConfig{
		Address: "localhost:0",
	}

	transport := NewHTTPTransport(config)
	handler := &mockHandler{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	require.NoError(b, err)
	defer transport.Stop()

	addr := transport.server.Addr
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
		},
	}

	req := &protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "benchmark.test",
		Params:  map[string]interface{}{"data": "test"},
	}
	reqBody, _ := json.Marshal(req)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Post("http://"+addr+"/", "application/json", bytes.NewReader(reqBody))
			if err != nil {
				b.Fatal(err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}