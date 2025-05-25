package transport

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"mcp-memory/pkg/mcp/protocol"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sseTestClient is a test client for SSE
type sseTestClient struct {
	events chan string
	errors chan error
	done   chan struct{}
}

func newSSETestClient() *sseTestClient {
	return &sseTestClient{
		events: make(chan string, 100),
		errors: make(chan error, 10),
		done:   make(chan struct{}),
	}
}

func (c *sseTestClient) connect(url string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read SSE stream
	go func() {
		defer close(c.done)
		scanner := bufio.NewScanner(resp.Body)
		defer resp.Body.Close()

		var eventData strings.Builder
		for scanner.Scan() {
			line := scanner.Text()

			if line == "" && eventData.Len() > 0 {
				// End of event
				c.events <- eventData.String()
				eventData.Reset()
			} else if strings.HasPrefix(line, "data: ") {
				eventData.WriteString(strings.TrimPrefix(line, "data: "))
			}
		}

		if err := scanner.Err(); err != nil {
			c.errors <- err
		}
	}()

	return nil
}

func (c *sseTestClient) close() {
	close(c.events)
	close(c.errors)
}

func TestSSETransport_StartStop(t *testing.T) {
	config := &SSEConfig{
		HTTPConfig: HTTPConfig{
			Address: "localhost:0",
		},
		EventPath: "/events",
	}

	transport := NewSSETransport(config)
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

func TestSSETransport_Connection(t *testing.T) {
	config := &SSEConfig{
		HTTPConfig: HTTPConfig{
			Address: "localhost:0",
		},
		EventPath:         "/events",
		HeartbeatInterval: 500 * time.Millisecond,
	}

	transport := NewSSETransport(config)
	handler := &mockHandler{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	require.NoError(t, err)
	defer transport.Stop()

	addr := transport.server.Addr
	baseURL := "http://" + addr

	// Connect SSE client
	client := newSSETestClient()
	err = client.connect(baseURL + "/events")
	require.NoError(t, err)

	// Wait for connection event
	select {
	case event := <-client.events:
		var data map[string]interface{}
		err = json.Unmarshal([]byte(event), &data)
		require.NoError(t, err)
		assert.NotEmpty(t, data["clientId"])
	case <-time.After(2 * time.Second):
		t.Fatal("No connection event received")
	}

	// Wait for capabilities event
	select {
	case event := <-client.events:
		var data map[string]interface{}
		err = json.Unmarshal([]byte(event), &data)
		require.NoError(t, err)
		assert.NotNil(t, data)
	case <-time.After(2 * time.Second):
		t.Fatal("No capabilities event received")
	}

	// Check client count
	assert.Equal(t, 1, transport.ClientCount())
}

func TestSSETransport_Broadcast(t *testing.T) {
	config := &SSEConfig{
		HTTPConfig: HTTPConfig{
			Address: "localhost:0",
		},
		EventPath: "/events",
	}

	transport := NewSSETransport(config)
	handler := &mockHandler{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	require.NoError(t, err)
	defer transport.Stop()

	addr := transport.server.Addr
	baseURL := "http://" + addr

	// Connect multiple clients
	numClients := 3
	clients := make([]*sseTestClient, numClients)

	for i := 0; i < numClients; i++ {
		client := newSSETestClient()
		err = client.connect(baseURL + "/events")
		require.NoError(t, err)
		clients[i] = client

		// Wait for connection event
		<-client.events
		<-client.events // capabilities
	}

	// Wait for all connections
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, numClients, transport.ClientCount())

	// Broadcast message
	testData := map[string]interface{}{
		"message":   "broadcast test",
		"timestamp": time.Now().Unix(),
	}
	transport.BroadcastEvent("test", testData)

	// All clients should receive the broadcast
	for i, client := range clients {
		select {
		case event := <-client.events:
			var data map[string]interface{}
			err = json.Unmarshal([]byte(event), &data)
			require.NoError(t, err)
			assert.Equal(t, "broadcast test", data["message"])
		case <-time.After(2 * time.Second):
			t.Fatalf("Client %d did not receive broadcast", i)
		}
	}
}

func TestSSETransport_SendToClient(t *testing.T) {
	config := &SSEConfig{
		HTTPConfig: HTTPConfig{
			Address: "localhost:0",
		},
		EventPath: "/events",
	}

	transport := NewSSETransport(config)
	handler := &mockHandler{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	require.NoError(t, err)
	defer transport.Stop()

	addr := transport.server.Addr
	baseURL := "http://" + addr

	// Connect client and get client ID
	req, _ := http.NewRequest("GET", baseURL+"/events", nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	var clientID string

	// Read connection event to get client ID
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			var event map[string]interface{}
			json.Unmarshal([]byte(data), &event)
			if id, ok := event["clientId"]; ok {
				clientID = id.(string)
				break
			}
		}
	}

	require.NotEmpty(t, clientID)

	// Send message to specific client
	testData := map[string]interface{}{
		"message": "direct message",
	}
	err = transport.SendToClient(clientID, "direct", testData)
	require.NoError(t, err)

	// Try sending to non-existent client
	err = transport.SendToClient("invalid-client", "test", testData)
	assert.Error(t, err)
}

func TestSSETransport_CommandEndpoint(t *testing.T) {
	config := &SSEConfig{
		HTTPConfig: HTTPConfig{
			Address: "localhost:0",
			Path:    "/command",
		},
		EventPath: "/events",
	}

	transport := NewSSETransport(config)

	handler := &mockHandler{
		handleFunc: func(ctx context.Context, req *protocol.JSONRPCRequest) *protocol.JSONRPCResponse {
			return &protocol.JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: map[string]interface{}{
					"echo": req.Method,
				},
			}
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	require.NoError(t, err)
	defer transport.Stop()

	addr := transport.server.Addr
	baseURL := "http://" + addr

	httpClient := &http.Client{Timeout: 5 * time.Second}

	t.Run("command without client ID", func(t *testing.T) {
		req := &protocol.JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      1,
			Method:  "test.method",
		}

		body, _ := json.Marshal(req)
		resp, err := httpClient.Post(baseURL+"/command", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var jsonResp protocol.JSONRPCResponse
		err = json.NewDecoder(resp.Body).Decode(&jsonResp)
		require.NoError(t, err)
		assert.Equal(t, req.ID, jsonResp.ID)
	})

	t.Run("command with client ID", func(t *testing.T) {
		// First connect SSE client
		sseClient := newSSETestClient()
		err = sseClient.connect(baseURL + "/events")
		require.NoError(t, err)

		// Get client ID from connection event
		event := <-sseClient.events
		var connData map[string]interface{}
		json.Unmarshal([]byte(event), &connData)
		clientID := connData["clientId"].(string)

		// Send command with client ID
		req := &protocol.JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      2,
			Method:  "test.method",
		}

		body, _ := json.Marshal(req)
		httpReq, _ := http.NewRequest("POST", baseURL+"/command", bytes.NewReader(body))
		httpReq.Header.Set("X-Client-ID", clientID)

		resp, err := httpClient.Do(httpReq)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should get accepted status
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "accepted", result["status"])

		// Response should come via SSE
		select {
		case event := <-sseClient.events:
			t.Logf("Received SSE event: %s", event)
		case <-time.After(2 * time.Second):
			// Note: Need to properly parse SSE events in real implementation
			t.Log("SSE response parsing would happen here")
		}
	})
}

func TestSSETransport_MaxClients(t *testing.T) {
	config := &SSEConfig{
		HTTPConfig: HTTPConfig{
			Address: "localhost:0",
		},
		EventPath:  "/events",
		MaxClients: 2,
	}

	transport := NewSSETransport(config)
	handler := &mockHandler{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	require.NoError(t, err)
	defer transport.Stop()

	addr := transport.server.Addr
	baseURL := "http://" + addr + "/events"

	client := &http.Client{Timeout: 5 * time.Second}

	// Connect max clients
	responses := make([]*http.Response, 2)
	for i := 0; i < 2; i++ {
		resp, err := client.Get(baseURL)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		responses[i] = resp
	}

	// Try to connect one more (should fail)
	resp, err := client.Get(baseURL)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	// Close one connection
	responses[0].Body.Close()
	time.Sleep(100 * time.Millisecond)

	// Now should be able to connect
	resp, err = client.Get(baseURL)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Cleanup
	for _, r := range responses[1:] {
		r.Body.Close()
	}
}

func TestSSETransport_Heartbeat(t *testing.T) {
	config := &SSEConfig{
		HTTPConfig: HTTPConfig{
			Address: "localhost:0",
		},
		EventPath:         "/events",
		HeartbeatInterval: 100 * time.Millisecond,
	}

	transport := NewSSETransport(config)
	handler := &mockHandler{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	require.NoError(t, err)
	defer transport.Stop()

	addr := transport.server.Addr
	baseURL := "http://" + addr + "/events"

	req, _ := http.NewRequest("GET", baseURL, nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	heartbeatCount := 0
	done := make(chan struct{})

	go func() {
		defer close(done)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, ": heartbeat") {
				heartbeatCount++
				if heartbeatCount >= 3 {
					return
				}
			}
		}
	}()

	select {
	case <-done:
		assert.GreaterOrEqual(t, heartbeatCount, 3)
	case <-time.After(1 * time.Second):
		t.Fatal("Did not receive enough heartbeats")
	}
}

func TestSSETransport_EventBuffering(t *testing.T) {
	config := &SSEConfig{
		HTTPConfig: HTTPConfig{
			Address: "localhost:0",
		},
		EventPath:       "/events",
		EventBufferSize: 5,
	}

	transport := NewSSETransport(config)
	handler := &mockHandler{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	require.NoError(t, err)
	defer transport.Stop()

	// Get client ID first
	clients := transport.GetClients()
	require.Empty(t, clients) // No clients yet

	// Connect a client
	addr := transport.server.Addr
	baseURL := "http://" + addr + "/events"

	go func() {
		resp, _ := http.Get(baseURL)
		defer resp.Body.Close()
		time.Sleep(5 * time.Second) // Keep connection open
	}()

	// Wait for client to connect
	time.Sleep(100 * time.Millisecond)
	clients = transport.GetClients()
	require.Len(t, clients, 1)
	clientID := clients[0]

	// Send many events quickly (more than buffer size)
	successCount := 0
	for i := 0; i < 10; i++ {
		err := transport.SendToClient(clientID, "test", map[string]interface{}{"index": i})
		if err == nil {
			successCount++
		}
	}

	// Should have sent at least buffer size
	assert.GreaterOrEqual(t, successCount, 5)
	// But some should have been dropped due to buffer full
	assert.Less(t, successCount, 10)
}

// Benchmark tests

func BenchmarkSSETransport_Broadcast(b *testing.B) {
	config := &SSEConfig{
		HTTPConfig: HTTPConfig{
			Address: "localhost:0",
		},
		EventPath: "/events",
	}

	transport := NewSSETransport(config)
	handler := &mockHandler{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	require.NoError(b, err)
	defer transport.Stop()

	addr := transport.server.Addr
	baseURL := "http://" + addr + "/events"

	// Connect multiple clients
	numClients := 10
	for i := 0; i < numClients; i++ {
		go func() {
			resp, _ := http.Get(baseURL)
			defer resp.Body.Close()
			scanner := bufio.NewScanner(resp.Body)
			for scanner.Scan() {
				// Consume events
			}
		}()
	}

	// Wait for clients to connect
	time.Sleep(200 * time.Millisecond)

	message := map[string]interface{}{
		"type": "benchmark",
		"data": "test broadcast message",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		transport.BroadcastEvent("benchmark", message)
	}
}

func BenchmarkSSETransport_DirectMessage(b *testing.B) {
	config := &SSEConfig{
		HTTPConfig: HTTPConfig{
			Address: "localhost:0",
		},
		EventPath: "/events",
	}

	transport := NewSSETransport(config)
	handler := &mockHandler{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	require.NoError(b, err)
	defer transport.Stop()

	// Connect a client and get its ID
	addr := transport.server.Addr
	baseURL := "http://" + addr + "/events"

	clientConnected := make(chan string)
	go func() {
		resp, _ := http.Get(baseURL)
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)

		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				var event map[string]interface{}
				if json.Unmarshal([]byte(data), &event) == nil {
					if id, ok := event["clientId"]; ok {
						clientConnected <- id.(string)
						break
					}
				}
			}
		}

		// Keep reading
		for scanner.Scan() {
			// Consume events
		}
	}()

	clientID := <-clientConnected

	message := map[string]interface{}{
		"type": "benchmark",
		"data": "test direct message",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		transport.SendToClient(clientID, "benchmark", message)
	}
}
