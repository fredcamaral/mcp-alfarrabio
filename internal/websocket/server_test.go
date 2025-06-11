package websocket

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_NewServer(t *testing.T) {
	tests := []struct {
		name   string
		config *ServerConfig
		want   *ServerConfig
	}{
		{
			name:   "with default config",
			config: nil,
			want:   DefaultServerConfig(),
		},
		{
			name: "with custom config",
			config: &ServerConfig{
				MaxConnections:  500,
				ReadBufferSize:  2048,
				WriteBufferSize: 2048,
				EnableAuth:      false,
				RequiredVersion: "2.0.0",
				AllowedOrigins:  []string{"http://localhost:3000"},
			},
			want: &ServerConfig{
				MaxConnections:  500,
				ReadBufferSize:  2048,
				WriteBufferSize: 2048,
				EnableAuth:      false,
				RequiredVersion: "2.0.0",
				AllowedOrigins:  []string{"http://localhost:3000"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(tt.config)
			assert.NotNil(t, server)
			assert.NotNil(t, server.hub)
			assert.NotNil(t, server.pool)
			assert.NotNil(t, server.metricsCollector)
			assert.NotNil(t, server.heartbeat)
			assert.False(t, server.running)

			if tt.config != nil {
				assert.Equal(t, tt.config.MaxConnections, server.config.MaxConnections)
				assert.Equal(t, tt.config.EnableAuth, server.config.EnableAuth)
			}
		})
	}
}

func TestServer_StartStop(t *testing.T) {
	server := NewServer(nil)

	// Test start
	err := server.Start()
	require.NoError(t, err)
	assert.True(t, server.IsRunning())

	// Test double start
	err = server.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Test stop
	err = server.Stop()
	require.NoError(t, err)
	assert.False(t, server.IsRunning())

	// Test double stop
	err = server.Stop()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestServer_HandleUpgrade(t *testing.T) {
	server := NewServer(&ServerConfig{
		EnableAuth:     false,
		MaxConnections: 10,
		AllowedOrigins: []string{"*"},
	})

	// Start server
	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Create test HTTP server
	ts := httptest.NewServer(http.HandlerFunc(server.HandleUpgrade))
	defer ts.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

	tests := []struct {
		name       string
		headers    map[string]string
		query      string
		wantStatus int
	}{
		{
			name:       "successful connection",
			headers:    map[string]string{},
			query:      "?repository=test-repo&session_id=test-session",
			wantStatus: http.StatusSwitchingProtocols,
		},
		{
			name: "with CLI version",
			headers: map[string]string{
				"X-CLI-Version": "1.0.0",
			},
			query:      "",
			wantStatus: http.StatusSwitchingProtocols,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create WebSocket connection
			header := http.Header{}
			for k, v := range tt.headers {
				header.Set(k, v)
			}

			conn, resp, err := websocket.DefaultDialer.Dial(wsURL+tt.query, header)
			if tt.wantStatus == http.StatusSwitchingProtocols {
				require.NoError(t, err)
				assert.NotNil(t, conn)
				defer conn.Close()

				// Verify we receive welcome message
				var msg MemoryEvent
				err = conn.ReadJSON(&msg)
				require.NoError(t, err)
				assert.Equal(t, "connection", msg.Type)
				assert.Equal(t, "connected", msg.Action)
			} else {
				assert.Error(t, err)
				if resp != nil {
					assert.Equal(t, tt.wantStatus, resp.StatusCode)
				}
			}
		})
	}
}

func TestServer_Authentication(t *testing.T) {
	server := NewServer(&ServerConfig{
		EnableAuth:      true,
		RequiredVersion: "1.0.0",
		AllowedOrigins:  []string{"*"},
	})

	// Start server
	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Create test HTTP server
	ts := httptest.NewServer(http.HandlerFunc(server.HandleUpgrade))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

	tests := []struct {
		name      string
		headers   map[string]string
		wantError bool
	}{
		{
			name:      "missing CLI version",
			headers:   map[string]string{},
			wantError: true,
		},
		{
			name: "wrong CLI version",
			headers: map[string]string{
				"X-CLI-Version": "2.0.0",
			},
			wantError: true,
		},
		{
			name: "correct CLI version",
			headers: map[string]string{
				"X-CLI-Version": "1.0.0",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := http.Header{}
			for k, v := range tt.headers {
				header.Set(k, v)
			}

			conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
			if tt.wantError {
				assert.Error(t, err)
				if resp != nil {
					assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, conn)
				conn.Close()
			}
		})
	}
}

func TestServer_ConnectionLimit(t *testing.T) {
	server := NewServer(&ServerConfig{
		EnableAuth:     false,
		MaxConnections: 2, // Very low limit for testing
		AllowedOrigins: []string{"*"},
	})

	// Start server
	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Create test HTTP server
	ts := httptest.NewServer(http.HandlerFunc(server.HandleUpgrade))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

	// Connect first client
	conn1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn1.Close()

	// Connect second client
	conn2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn2.Close()

	// Try to connect third client (should fail)
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.Error(t, err)
	if resp != nil {
		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	}
}

func TestServer_MessageBroadcast(t *testing.T) {
	server := NewServer(&ServerConfig{
		EnableAuth:     false,
		AllowedOrigins: []string{"*"},
	})

	// Start server
	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Create test HTTP server
	ts := httptest.NewServer(http.HandlerFunc(server.HandleUpgrade))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

	// Connect client
	conn, _, err := websocket.DefaultDialer.Dial(wsURL+"?repository=test-repo", nil)
	require.NoError(t, err)
	defer conn.Close()

	// Skip welcome message
	var welcomeMsg MemoryEvent
	err = conn.ReadJSON(&welcomeMsg)
	require.NoError(t, err)

	// Broadcast event
	testEvent := &MemoryEvent{
		Type:       "memory",
		Action:     "created",
		ChunkID:    "test-chunk-123",
		Repository: "test-repo",
		Content:    "Test memory content",
		Timestamp:  time.Now(),
	}

	server.BroadcastEvent(testEvent)

	// Receive broadcast
	var receivedEvent MemoryEvent
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	err = conn.ReadJSON(&receivedEvent)
	require.NoError(t, err)

	assert.Equal(t, testEvent.Type, receivedEvent.Type)
	assert.Equal(t, testEvent.Action, receivedEvent.Action)
	assert.Equal(t, testEvent.ChunkID, receivedEvent.ChunkID)
	assert.Equal(t, testEvent.Content, receivedEvent.Content)
}

func TestServer_ClientMessages(t *testing.T) {
	server := NewServer(&ServerConfig{
		EnableAuth:     false,
		AllowedOrigins: []string{"*"},
	})

	// Start server
	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Create test HTTP server
	ts := httptest.NewServer(http.HandlerFunc(server.HandleUpgrade))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

	// Connect client
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Skip welcome message
	var welcomeMsg MemoryEvent
	err = conn.ReadJSON(&welcomeMsg)
	require.NoError(t, err)

	// Test ping-pong
	pingMsg := map[string]interface{}{
		"type":      "ping",
		"timestamp": time.Now().Unix(),
	}

	err = conn.WriteJSON(pingMsg)
	require.NoError(t, err)

	// Receive pong
	var pongMsg MemoryEvent
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	err = conn.ReadJSON(&pongMsg)
	require.NoError(t, err)
	assert.Equal(t, "pong", pongMsg.Type)

	// Test subscription
	subMsg := map[string]interface{}{
		"type":       "subscribe",
		"repository": "new-repo",
		"session_id": "new-session",
	}

	err = conn.WriteJSON(subMsg)
	require.NoError(t, err)

	// Allow time for subscription to process
	time.Sleep(100 * time.Millisecond)

	// Verify filtering works
	// Broadcast to different repo (should not receive)
	server.BroadcastEvent(&MemoryEvent{
		Type:       "memory",
		Action:     "created",
		Repository: "other-repo",
		Timestamp:  time.Now(),
	})

	// Broadcast to subscribed repo (should receive)
	server.BroadcastEvent(&MemoryEvent{
		Type:       "memory",
		Action:     "created",
		Repository: "new-repo",
		Timestamp:  time.Now(),
	})

	// Should only receive the second event
	var event MemoryEvent
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	err = conn.ReadJSON(&event)
	require.NoError(t, err)
	assert.Equal(t, "new-repo", event.Repository)
}

func TestServer_Heartbeat(t *testing.T) {
	// Create server with short heartbeat interval for testing
	config := DefaultServerConfig()
	config.PingInterval = 100 * time.Millisecond
	config.EnableAuth = false

	server := NewServer(config)

	// Start server
	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Create test HTTP server
	ts := httptest.NewServer(http.HandlerFunc(server.HandleUpgrade))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

	// Connect client
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Skip welcome message
	var welcomeMsg MemoryEvent
	err = conn.ReadJSON(&welcomeMsg)
	require.NoError(t, err)

	// Wait for heartbeat
	heartbeatReceived := false
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))

	for i := 0; i < 5; i++ {
		var msg MemoryEvent
		err = conn.ReadJSON(&msg)
		if err != nil {
			break
		}
		if msg.Type == "heartbeat" {
			heartbeatReceived = true
			break
		}
	}

	assert.True(t, heartbeatReceived, "Should receive heartbeat message")
}

func TestServer_Metrics(t *testing.T) {
	server := NewServer(nil)

	// Start server
	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Get initial metrics
	metrics := server.GetMetrics()
	assert.NotNil(t, metrics)
	assert.Equal(t, 0, metrics.ActiveConnections)

	// Create test HTTP server
	ts := httptest.NewServer(http.HandlerFunc(server.HandleUpgrade))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

	// Connect client
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	// Allow time for connection to register
	time.Sleep(100 * time.Millisecond)

	// Check metrics updated
	assert.Equal(t, 1, server.GetConnectionCount())

	// Close connection
	conn.Close()

	// Allow time for disconnection to process
	time.Sleep(100 * time.Millisecond)

	// Check metrics updated
	assert.Equal(t, 0, server.GetConnectionCount())
}

func TestServer_CORS(t *testing.T) {
	tests := []struct {
		name           string
		allowedOrigins []string
		origin         string
		shouldConnect  bool
	}{
		{
			name:           "wildcard allows all",
			allowedOrigins: []string{"*"},
			origin:         "http://example.com",
			shouldConnect:  true,
		},
		{
			name:           "specific origin allowed",
			allowedOrigins: []string{"http://localhost:3000"},
			origin:         "http://localhost:3000",
			shouldConnect:  true,
		},
		{
			name:           "origin not in allowed list",
			allowedOrigins: []string{"http://localhost:3000"},
			origin:         "http://evil.com",
			shouldConnect:  false,
		},
		{
			name:           "no origin header allowed",
			allowedOrigins: []string{"http://localhost:3000"},
			origin:         "",
			shouldConnect:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(&ServerConfig{
				EnableAuth:     false,
				AllowedOrigins: tt.allowedOrigins,
			})

			// Start server
			err := server.Start()
			require.NoError(t, err)
			defer server.Stop()

			// Create test HTTP server
			ts := httptest.NewServer(http.HandlerFunc(server.HandleUpgrade))
			defer ts.Close()

			wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

			// Create connection with origin header
			header := http.Header{}
			if tt.origin != "" {
				header.Set("Origin", tt.origin)
			}

			conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
			if tt.shouldConnect {
				require.NoError(t, err)
				assert.NotNil(t, conn)
				conn.Close()
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestServer_ConcurrentConnections(t *testing.T) {
	server := NewServer(&ServerConfig{
		EnableAuth:     false,
		MaxConnections: 100,
		AllowedOrigins: []string{"*"},
	})

	// Start server
	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Create test HTTP server
	ts := httptest.NewServer(http.HandlerFunc(server.HandleUpgrade))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

	// Connect multiple clients concurrently
	numClients := 10
	done := make(chan bool, numClients)

	for i := 0; i < numClients; i++ {
		go func(id int) {
			conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				t.Errorf("Client %d failed to connect: %v", id, err)
				done <- false
				return
			}

			// Read welcome message
			var msg MemoryEvent
			err = conn.ReadJSON(&msg)
			if err != nil {
				t.Errorf("Client %d failed to read welcome: %v", id, err)
			}

			// Send a message
			err = conn.WriteJSON(map[string]interface{}{
				"type": "ping",
			})
			if err != nil {
				t.Errorf("Client %d failed to send ping: %v", id, err)
			}

			conn.Close()
			done <- true
		}(i)
	}

	// Wait for all clients
	successCount := 0
	for i := 0; i < numClients; i++ {
		if <-done {
			successCount++
		}
	}

	assert.Equal(t, numClients, successCount, "All clients should connect successfully")
}

func TestServer_GettersAndConfig(t *testing.T) {
	config := &ServerConfig{
		MaxConnections:  500,
		EnableAuth:      true,
		RequiredVersion: "2.0.0",
	}

	server := NewServer(config)

	// Test getters
	assert.Equal(t, config, server.GetConfig())
	assert.NotNil(t, server.GetHub())
	assert.NotNil(t, server.GetPool())
	assert.NotNil(t, server.GetMetrics())

	// Verify configuration was applied
	assert.Equal(t, 500, server.config.MaxConnections)
	assert.True(t, server.config.EnableAuth)
	assert.Equal(t, "2.0.0", server.config.RequiredVersion)
}
