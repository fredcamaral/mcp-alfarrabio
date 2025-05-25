package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mcp-memory/pkg/mcp/protocol"
	"strings"
	"sync"
	"testing"
	"time"
)

// Mock request handler for testing
type mockRequestHandler struct {
	responses map[string]*protocol.JSONRPCResponse
	requests  []*protocol.JSONRPCRequest
	mu        sync.Mutex
	delay     time.Duration
}

func newMockRequestHandler() *mockRequestHandler {
	return &mockRequestHandler{
		responses: make(map[string]*protocol.JSONRPCResponse),
		requests:  make([]*protocol.JSONRPCRequest, 0),
	}
}

func (h *mockRequestHandler) HandleRequest(ctx context.Context, req *protocol.JSONRPCRequest) *protocol.JSONRPCResponse {
	h.mu.Lock()
	h.requests = append(h.requests, req)
	delay := h.delay
	h.mu.Unlock()

	// Simulate processing delay if set
	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return &protocol.JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   protocol.NewJSONRPCError(protocol.InternalError, "Context cancelled", nil),
			}
		}
	}

	h.mu.Lock()
	resp, exists := h.responses[req.Method]
	h.mu.Unlock()

	if exists {
		// Copy response and set correct ID
		respCopy := *resp
		respCopy.ID = req.ID
		return &respCopy
	}

	// Default response
	return &protocol.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  map[string]interface{}{"method": req.Method, "params": req.Params},
	}
}

func (h *mockRequestHandler) getRequests() []*protocol.JSONRPCRequest {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]*protocol.JSONRPCRequest{}, h.requests...)
}

func TestNewStdioTransport(t *testing.T) {
	transport := NewStdioTransport()

	if transport.input == nil {
		t.Error("Expected input to be initialized")
	}
	if transport.output == nil {
		t.Error("Expected output to be initialized")
	}
	if transport.scanner == nil {
		t.Error("Expected scanner to be initialized")
	}
	if transport.encoder == nil {
		t.Error("Expected encoder to be initialized")
	}
	if transport.running {
		t.Error("Expected transport to not be running initially")
	}
}

func TestNewStdioTransportWithIO(t *testing.T) {
	input := strings.NewReader("test input")
	output := &bytes.Buffer{}

	transport := NewStdioTransportWithIO(input, output)

	if transport.input != input {
		t.Error("Expected custom input to be set")
	}
	if transport.output != output {
		t.Error("Expected custom output to be set")
	}
	if transport.scanner == nil {
		t.Error("Expected scanner to be initialized")
	}
	if transport.encoder == nil {
		t.Error("Expected encoder to be initialized")
	}
}

func TestStdioTransportStart(t *testing.T) {
	tests := []struct {
		name              string
		input             string
		expectedResponses []string
		expectedError     bool
		contextTimeout    time.Duration
		handler           *mockRequestHandler
	}{
		{
			name: "single valid request",
			input: `{"jsonrpc":"2.0","id":1,"method":"test","params":{"key":"value"}}` + "\n",
			expectedResponses: []string{
				`{"jsonrpc":"2.0","id":1,"result":{"method":"test","params":{"key":"value"}}}`,
			},
			handler: newMockRequestHandler(),
		},
		{
			name: "multiple requests",
			input: `{"jsonrpc":"2.0","id":1,"method":"method1"}` + "\n" +
				`{"jsonrpc":"2.0","id":2,"method":"method2","params":{"test":true}}` + "\n",
			expectedResponses: []string{
				`{"jsonrpc":"2.0","id":1,"result":{"method":"method1","params":null}}`,
				`{"jsonrpc":"2.0","id":2,"result":{"method":"method2","params":{"test":true}}}`,
			},
			handler: newMockRequestHandler(),
		},
		{
			name: "invalid JSON",
			input: `invalid json` + "\n",
			expectedResponses: []string{
				`{"jsonrpc":"2.0","error":{"code":-32700,"message":"Parse error","data":"invalid character 'i' looking for beginning of value"}}`,
			},
			handler: newMockRequestHandler(),
		},
		{
			name: "empty lines ignored",
			input: "\n" + `{"jsonrpc":"2.0","id":1,"method":"test"}` + "\n" + "\n",
			expectedResponses: []string{
				`{"jsonrpc":"2.0","id":1,"result":{"method":"test","params":null}}`,
			},
			handler: newMockRequestHandler(),
		},
		{
			name: "notification (no ID)",
			input: `{"jsonrpc":"2.0","method":"notify"}` + "\n",
			handler: func() *mockRequestHandler {
				h := newMockRequestHandler()
				h.responses["notify"] = &protocol.JSONRPCResponse{
					JSONRPC: "2.0",
					Result:  "notification received",
				}
				return h
			}(),
			expectedResponses: []string{
				`{"jsonrpc":"2.0","result":"notification received"}`,
			},
		},
		{
			name:           "context cancellation",
			input:          `{"jsonrpc":"2.0","id":1,"method":"slow"}` + "\n",
			contextTimeout: 50 * time.Millisecond,
			handler: func() *mockRequestHandler {
				h := newMockRequestHandler()
				h.delay = 100 * time.Millisecond
				return h
			}(),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := strings.NewReader(tt.input)
			output := &bytes.Buffer{}

			transport := NewStdioTransportWithIO(input, output)

			ctx := context.Background()
			if tt.contextTimeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tt.contextTimeout)
				defer cancel()
			}

			// Start transport in goroutine
			errCh := make(chan error)
			go func() {
				errCh <- transport.Start(ctx, tt.handler)
			}()

			// Wait for completion or timeout
			select {
			case err := <-errCh:
				if tt.expectedError && err == nil {
					t.Error("Expected error but got none")
				} else if !tt.expectedError && err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			case <-time.After(200 * time.Millisecond):
				if !tt.expectedError {
					t.Error("Transport did not complete in time")
				}
			}

			// Check responses
			if len(tt.expectedResponses) > 0 {
				outputStr := output.String()
				lines := strings.Split(strings.TrimSpace(outputStr), "\n")

				if len(lines) != len(tt.expectedResponses) {
					t.Errorf("Expected %d responses, got %d", len(tt.expectedResponses), len(lines))
				}

				for i, expected := range tt.expectedResponses {
					if i < len(lines) {
						// Parse both to compare as JSON objects
						var expectedJSON, actualJSON interface{}
						if err := json.Unmarshal([]byte(expected), &expectedJSON); err != nil {
							t.Fatalf("Failed to parse expected JSON: %v", err)
						}
						if err := json.Unmarshal([]byte(lines[i]), &actualJSON); err != nil {
							t.Fatalf("Failed to parse actual JSON: %v", err)
						}

						expectedStr, _ := json.Marshal(expectedJSON)
						actualStr, _ := json.Marshal(actualJSON)

						if string(expectedStr) != string(actualStr) {
							t.Errorf("Response %d mismatch:\nExpected: %s\nActual: %s", i, expected, lines[i])
						}
					}
				}
			}
		})
	}
}

func TestStdioTransportStop(t *testing.T) {
	transport := NewStdioTransport()

	// Stop should work even when not running
	err := transport.Stop()
	if err != nil {
		t.Errorf("Unexpected error stopping non-running transport: %v", err)
	}

	// Start transport
	input := strings.NewReader("")
	output := &bytes.Buffer{}
	transport = NewStdioTransportWithIO(input, output)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler := newMockRequestHandler()

	// Start in background
	go transport.Start(ctx, handler)

	// Give it time to start
	time.Sleep(10 * time.Millisecond)

	// Stop transport
	err = transport.Stop()
	if err != nil {
		t.Errorf("Unexpected error stopping transport: %v", err)
	}

	// Verify it's not running
	if transport.IsRunning() {
		t.Error("Expected transport to not be running after stop")
	}
}

func TestStdioTransportConcurrency(t *testing.T) {
	// Test concurrent requests
	requests := []string{
		`{"jsonrpc":"2.0","id":1,"method":"concurrent1"}`,
		`{"jsonrpc":"2.0","id":2,"method":"concurrent2"}`,
		`{"jsonrpc":"2.0","id":3,"method":"concurrent3"}`,
	}

	input := strings.NewReader(strings.Join(requests, "\n") + "\n")
	output := &bytes.Buffer{}

	transport := NewStdioTransportWithIO(input, output)
	handler := newMockRequestHandler()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := transport.Start(ctx, handler)
	if err != nil && err != context.DeadlineExceeded {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify all requests were handled
	handledRequests := handler.getRequests()
	if len(handledRequests) != len(requests) {
		t.Errorf("Expected %d requests, got %d", len(requests), len(handledRequests))
	}

	// Verify responses
	outputStr := output.String()
	lines := strings.Split(strings.TrimSpace(outputStr), "\n")
	if len(lines) != len(requests) {
		t.Errorf("Expected %d responses, got %d", len(requests), len(lines))
	}
}

func TestStdioTransportAlreadyRunning(t *testing.T) {
	transport := NewStdioTransport()
	handler := newMockRequestHandler()

	// Mark as running
	transport.running = true

	err := transport.Start(context.Background(), handler)
	if err == nil {
		t.Error("Expected error when starting already running transport")
	}
	if err.Error() != "transport already running" {
		t.Errorf("Expected 'transport already running' error, got: %v", err)
	}
}

func TestStdioTransportSendResponseError(t *testing.T) {
	// Create a writer that fails
	failingWriter := &failingWriter{
		failAfter: 0,
		err:       errors.New("write failed"),
	}

	input := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"test"}` + "\n")
	transport := NewStdioTransportWithIO(input, failingWriter)

	handler := newMockRequestHandler()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := transport.Start(ctx, handler)
	if err == nil {
		t.Error("Expected error when response sending fails")
	}
	if !strings.Contains(err.Error(), "sending response") {
		t.Errorf("Expected 'sending response' error, got: %v", err)
	}
}

func TestStdioTransportScannerError(t *testing.T) {
	// Create a reader that fails
	failingReader := &failingReader{
		failAfter: 10,
		err:       errors.New("read failed"),
	}

	output := &bytes.Buffer{}
	transport := NewStdioTransportWithIO(failingReader, output)

	handler := newMockRequestHandler()
	err := transport.Start(context.Background(), handler)
	
	if err == nil {
		t.Error("Expected error when scanner fails")
	}
	if !strings.Contains(err.Error(), "scanning input") {
		t.Errorf("Expected 'scanning input' error, got: %v", err)
	}
}

func TestIsRunning(t *testing.T) {
	transport := NewStdioTransport()

	// Initially not running
	if transport.IsRunning() {
		t.Error("Expected transport to not be running initially")
	}

	// Start transport
	input := strings.NewReader("")
	output := &bytes.Buffer{}
	transport = NewStdioTransportWithIO(input, output)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler := newMockRequestHandler()

	// Start in background
	startedCh := make(chan bool)
	go func() {
		transport.mutex.Lock()
		transport.running = true
		transport.mutex.Unlock()
		startedCh <- true
		transport.Start(ctx, handler)
	}()

	// Wait for start
	<-startedCh

	// Should be running
	if !transport.IsRunning() {
		t.Error("Expected transport to be running")
	}

	// Stop
	cancel()
	time.Sleep(50 * time.Millisecond)

	// Should not be running
	if transport.IsRunning() {
		t.Error("Expected transport to not be running after context cancel")
	}
}

// Helper types for testing

type failingWriter struct {
	failAfter int
	written   int
	err       error
}

func (w *failingWriter) Write(p []byte) (n int, err error) {
	if w.written >= w.failAfter {
		return 0, w.err
	}
	w.written += len(p)
	return len(p), nil
}

type failingReader struct {
	failAfter int
	read      int
	err       error
}

func (r *failingReader) Read(p []byte) (n int, err error) {
	if r.read >= r.failAfter {
		return 0, r.err
	}
	// Return some data before failing
	if r.read < r.failAfter {
		data := []byte("test data\n")
		n = copy(p, data)
		r.read += n
		return n, nil
	}
	return 0, r.err
}

// Benchmark tests
func BenchmarkStdioTransportSingleRequest(b *testing.B) {
	request := `{"jsonrpc":"2.0","id":1,"method":"benchmark","params":{"test":true}}` + "\n"
	handler := newMockRequestHandler()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		input := strings.NewReader(request)
		output := &bytes.Buffer{}
		transport := NewStdioTransportWithIO(input, output)
		
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		transport.Start(ctx, handler)
		cancel()
	}
}

func BenchmarkStdioTransportMultipleRequests(b *testing.B) {
	requests := strings.Repeat(`{"jsonrpc":"2.0","id":1,"method":"benchmark","params":{"test":true}}`+"\n", 100)
	handler := newMockRequestHandler()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		input := strings.NewReader(requests)
		output := &bytes.Buffer{}
		transport := NewStdioTransportWithIO(input, output)
		
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		transport.Start(ctx, handler)
		cancel()
	}
}

func BenchmarkStdioTransportConcurrentAccess(b *testing.B) {
	transport := NewStdioTransport()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Simulate concurrent IsRunning checks
			_ = transport.IsRunning()
		}
	})
}

// Test edge cases
func TestStdioTransportEdgeCases(t *testing.T) {
	t.Run("nil response from handler", func(t *testing.T) {
		input := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"test"}` + "\n")
		output := &bytes.Buffer{}
		transport := NewStdioTransportWithIO(input, output)

		// Handler that returns nil
		handler := &mockRequestHandler{
			responses: map[string]*protocol.JSONRPCResponse{
				"test": nil,
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := transport.Start(ctx, handler)
		if err != nil && err != context.DeadlineExceeded {
			t.Errorf("Unexpected error: %v", err)
		}

		// Should not have written any response
		if output.Len() > 0 {
			t.Error("Expected no output for nil response")
		}
	})

	t.Run("very large request", func(t *testing.T) {
		// Create a large params object
		largeParams := make(map[string]interface{})
		for i := 0; i < 1000; i++ {
			largeParams[strings.Repeat("key", 100)] = strings.Repeat("value", 100)
		}

		reqData, _ := json.Marshal(protocol.JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      1,
			Method:  "large",
			Params:  largeParams,
		})

		input := strings.NewReader(string(reqData) + "\n")
		output := &bytes.Buffer{}
		transport := NewStdioTransportWithIO(input, output)

		handler := newMockRequestHandler()
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		err := transport.Start(ctx, handler)
		if err != nil && err != context.DeadlineExceeded && err != io.EOF {
			t.Errorf("Unexpected error: %v", err)
		}

		// Should have handled the large request
		requests := handler.getRequests()
		if len(requests) != 1 {
			t.Errorf("Expected 1 request, got %d", len(requests))
		}
	})

	t.Run("EOF handling", func(t *testing.T) {
		// Empty input should result in immediate EOF
		input := strings.NewReader("")
		output := &bytes.Buffer{}
		transport := NewStdioTransportWithIO(input, output)

		handler := newMockRequestHandler()
		err := transport.Start(context.Background(), handler)

		// Should complete without error (EOF is normal termination)
		if err != nil {
			t.Errorf("Expected nil error for EOF, got: %v", err)
		}
	})
}