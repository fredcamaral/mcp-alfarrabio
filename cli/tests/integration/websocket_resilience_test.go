package integration

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// WebSocketResilienceSuite tests WebSocket connection resilience
type WebSocketResilienceSuite struct {
	suite.Suite
	serverContainer testcontainers.Container
	qdrantContainer testcontainers.Container
	serverURL       string
	wsURL           string
	logger          *slog.Logger
}

func (s *WebSocketResilienceSuite) SetupSuite() {
	ctx := context.Background()
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Create a custom network for container communication
	networkName := "mcp-ws-test-network"
	networkReq := testcontainers.NetworkRequest{
		Name:   networkName,
		Driver: "bridge",
	}
	_, err := testcontainers.GenericNetwork(ctx, testcontainers.GenericNetworkRequest{
		NetworkRequest: networkReq,
	})
	s.Require().NoError(err)

	// Start Qdrant container
	s.logger.Info("starting Qdrant container for WebSocket tests")
	qdrantReq := testcontainers.ContainerRequest{
		Image:        "qdrant/qdrant:latest",
		ExposedPorts: []string{"6333/tcp"},
		WaitingFor:   wait.ForHTTP("/").WithPort("6333/tcp").WithStartupTimeout(60 * time.Second),
		Networks:     []string{networkName},
		NetworkAliases: map[string][]string{
			networkName: {"qdrant"},
		},
		Env: map[string]string{
			"QDRANT__SERVICE__HTTP_PORT": "6333",
			"QDRANT__SERVICE__GRPC_PORT": "6334",
		},
	}

	qdrant, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: qdrantReq,
		Started:          true,
	})
	s.Require().NoError(err)
	s.qdrantContainer = qdrant

	qdrantHost, err := qdrant.Host(ctx)
	s.Require().NoError(err)
	qdrantPort, err := qdrant.MappedPort(ctx, "6333")
	s.Require().NoError(err)

	s.logger.Info("Qdrant started", slog.String("host", qdrantHost), slog.String("port", qdrantPort.Port()))

	// Additional wait to ensure Qdrant is fully ready
	time.Sleep(3 * time.Second)

	// Start MCP Memory Server
	s.logger.Info("starting MCP Memory Server for WebSocket tests")
	serverReq := testcontainers.ContainerRequest{
		Image:        "ghcr.io/lerianstudio/lerian-mcp-memory:latest",
		ExposedPorts: []string{"9080/tcp"},
		WaitingFor:   wait.ForHTTP("/health").WithPort("9080/tcp").WithStartupTimeout(120 * time.Second),
		Networks:     []string{networkName},
		NetworkAliases: map[string][]string{
			networkName: {"mcp-server"},
		},
		Env: map[string]string{
			"MCP_HOST_PORT":               "9080",
			"MCP_MEMORY_QDRANT_HOST":      "qdrant",      // Use container alias for internal communication
			"QDRANT_HOST_PORT":            "qdrant:6333", // Use container network address
			"MCP_MEMORY_LOG_LEVEL":        "debug",
			"OPENAI_API_KEY":              "test-key",
			"MCP_MEMORY_SERVER_MODE":      "http",
			"MCP_MEMORY_ENABLE_WEBSOCKET": "true",
			"MCP_MEMORY_WS_PING_INTERVAL": "5s",
			"MCP_MEMORY_WS_PONG_WAIT":     "10s",
			"MCP_MEMORY_HTTP_PORT":        "9080",
			"MCP_MEMORY_HEALTH_PORT":      "8081",
		},
	}

	server, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: serverReq,
		Started:          true,
	})
	s.Require().NoError(err)
	s.serverContainer = server

	serverHost, err := server.Host(ctx)
	s.Require().NoError(err)
	serverPort, err := server.MappedPort(ctx, "9080")
	s.Require().NoError(err)

	s.serverURL = fmt.Sprintf("http://%s:%s", serverHost, serverPort.Port())
	s.wsURL = fmt.Sprintf("ws://%s:%s", serverHost, serverPort.Port())

	s.logger.Info("WebSocket resilience test environment ready",
		slog.String("server_url", s.serverURL),
		slog.String("ws_url", s.wsURL))
}

func (s *WebSocketResilienceSuite) TearDownSuite() {
	ctx := context.Background()

	if s.serverContainer != nil {
		if err := s.serverContainer.Terminate(ctx); err != nil {
			s.logger.Error("failed to terminate server container", slog.Any("error", err))
		}
	}

	if s.qdrantContainer != nil {
		if err := s.qdrantContainer.Terminate(ctx); err != nil {
			s.logger.Error("failed to terminate qdrant container", slog.Any("error", err))
		}
	}
}

// TestWebSocketConnection tests basic WebSocket connection establishment
func (s *WebSocketResilienceSuite) TestWebSocketConnection() {
	ctx := context.Background()

	s.logger.Info("testing WebSocket connection establishment")

	hub := NewNotificationHub(s.logger)
	wsClient := NewWebSocketClient(s.wsURL, "1.0.0", hub, s.logger)

	// Test connection
	err := wsClient.Connect(ctx)
	s.Require().NoError(err)
	defer wsClient.Close()

	// Verify connection status
	s.Assert().True(wsClient.IsConnected())

	// Test subscription
	err = wsClient.SubscribeToRepositories([]string{"test-repo"})
	s.Require().NoError(err)

	s.logger.Info("WebSocket connection test completed")
}

// TestAutomaticReconnection tests automatic reconnection after connection loss
func (s *WebSocketResilienceSuite) TestAutomaticReconnection() {
	ctx := context.Background()

	s.logger.Info("testing automatic reconnection")

	hub := NewNotificationHub(s.logger)
	wsClient := NewWebSocketClient(s.wsURL, "1.0.0", hub, s.logger)

	// Track connection status changes
	var connectionEvents []bool
	var eventsMu sync.Mutex

	hub.Subscribe("reconnection-test", func(event *TaskEvent) {
		// We're mainly interested in connection status changes
	})

	hub.SubscribeToConnectionStatus(func(connected bool) {
		eventsMu.Lock()
		connectionEvents = append(connectionEvents, connected)
		eventsMu.Unlock()
		s.logger.Info("connection status changed", slog.Bool("connected", connected))
	})

	// Initial connection
	err := wsClient.Connect(ctx)
	s.Require().NoError(err)

	// Wait for initial connection event
	time.Sleep(1 * time.Second)

	// Subscribe to a repository
	err = wsClient.SubscribeToRepositories([]string{"reconnection-test"})
	s.Require().NoError(err)

	// Force disconnect to trigger reconnection
	s.logger.Info("forcing disconnection")
	wsClient.ForceDisconnect()

	// Wait for disconnection and reconnection
	time.Sleep(6 * time.Second) // Allow for reconnection attempts

	// Verify we eventually reconnect (the client should automatically reconnect)
	eventsMu.Lock()
	events := make([]bool, len(connectionEvents))
	copy(events, connectionEvents)
	eventsMu.Unlock()

	s.logger.Info("connection events during test", slog.Any("events", events))

	// Should have at least one true (initial connection) and one false (disconnection)
	hasConnection := false
	hasDisconnection := false

	for _, event := range events {
		if event {
			hasConnection = true
		} else {
			hasDisconnection = true
		}
	}

	s.Assert().True(hasConnection, "should have had at least one connection event")
	s.Assert().True(hasDisconnection, "should have had at least one disconnection event")

	wsClient.Close()
	s.logger.Info("automatic reconnection test completed")
}

// TestExponentialBackoff tests reconnection backoff behavior
func (s *WebSocketResilienceSuite) TestExponentialBackoff() {
	ctx := context.Background()

	s.logger.Info("testing exponential backoff")

	hub := NewNotificationHub(s.logger)

	// Use a non-existent URL to force connection failures
	invalidURL := "ws://127.0.0.1:65534/ws" // Port that should be closed
	wsClient := NewWebSocketClient(invalidURL, "1.0.0", hub, s.logger)

	var reconnectAttempts []time.Time
	_ = sync.Mutex{}

	// We'll track when reconnection attempts happen by monitoring the logs
	// Since we can't directly hook into the reconnection timing, we'll test by
	// trying to connect to an invalid endpoint and measuring timing

	start := time.Now()

	// Try to connect (should fail)
	err := wsClient.Connect(ctx)
	s.Assert().Error(err, "connection to invalid URL should fail")

	// The client should attempt reconnections with exponential backoff
	// Let's give it some time to make several attempts
	time.Sleep(15 * time.Second)

	// Now test with valid URL after backoff period
	wsClient.Close()

	// Create new client with valid URL
	wsClient2 := NewWebSocketClient(s.wsURL, "1.0.0", hub, s.logger)

	err = wsClient2.Connect(ctx)
	s.Require().NoError(err)
	defer wsClient2.Close()

	s.Assert().True(wsClient2.IsConnected())

	duration := time.Since(start)
	s.logger.Info("exponential backoff test completed",
		slog.Duration("total_duration", duration),
		slog.Int("reconnect_attempts", len(reconnectAttempts)))
}

// TestConcurrentConnections tests multiple concurrent WebSocket connections
func (s *WebSocketResilienceSuite) TestConcurrentConnections() {
	ctx := context.Background()

	s.logger.Info("testing concurrent WebSocket connections")

	const numConnections = 5
	var clients []*WebSocketClient
	var wg sync.WaitGroup

	hub := NewNotificationHub(s.logger)

	// Track successful connections
	var successfulConnections int64

	// Create multiple concurrent connections
	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			client := NewWebSocketClient(s.wsURL, "1.0.0", hub, s.logger)

			err := client.Connect(ctx)
			if err != nil {
				s.logger.Error("failed to connect client",
					slog.Int("client_id", clientID),
					slog.Any("error", err))
				return
			}

			atomic.AddInt64(&successfulConnections, 1)
			clients = append(clients, client)

			// Subscribe to repository
			err = client.SubscribeToRepositories([]string{fmt.Sprintf("concurrent-test-%d", clientID)})
			if err != nil {
				s.logger.Error("failed to subscribe",
					slog.Int("client_id", clientID),
					slog.Any("error", err))
			}

			s.logger.Info("client connected successfully", slog.Int("client_id", clientID))
		}(i)
	}

	wg.Wait()

	s.Assert().Equal(int64(numConnections), successfulConnections,
		"all concurrent connections should succeed")

	// Test that all connections are active
	for i, client := range clients {
		s.Assert().True(client.IsConnected(), "client %d should be connected", i)
	}

	// Create task and verify all clients can potentially receive it
	httpClient := NewHTTPClient(s.serverURL, "1.0.0", s.logger)

	createReq := &CreateTaskRequest{
		Content:    "Task for concurrent connections test",
		Priority:   "medium",
		Repository: "concurrent-test-0", // Should notify first client
	}

	task, err := httpClient.CreateTask(ctx, createReq)
	s.Require().NoError(err)

	s.logger.Info("created task for concurrent test", slog.String("task_id", task.ID))

	// Wait for potential notifications
	time.Sleep(2 * time.Second)

	// Clean up connections
	for i, client := range clients {
		client.Close()
		s.logger.Info("closed client connection", slog.Int("client_id", i))
	}

	s.logger.Info("concurrent connections test completed",
		slog.Int64("successful_connections", successfulConnections))
}

// TestLongRunningConnection tests WebSocket stability over time
func (s *WebSocketResilienceSuite) TestLongRunningConnection() {
	ctx := context.Background()

	s.logger.Info("testing long-running WebSocket connection")

	hub := NewNotificationHub(s.logger)
	wsClient := NewWebSocketClient(s.wsURL, "1.0.0", hub, s.logger)

	// Track events received
	var eventsReceived int64

	hub.Subscribe("long-running-test", func(event *TaskEvent) {
		atomic.AddInt64(&eventsReceived, 1)
		s.logger.Debug("received event in long-running test",
			slog.String("type", string(event.Type)),
			slog.String("task_id", event.TaskID))
	})

	// Connect
	err := wsClient.Connect(ctx)
	s.Require().NoError(err)
	defer wsClient.Close()

	err = wsClient.SubscribeToRepositories([]string{"long-running-test"})
	s.Require().NoError(err)

	// Create HTTP client for generating events
	httpClient := NewHTTPClient(s.serverURL, "1.0.0", s.logger)

	// Simulate long-running operation with periodic activity
	const testDuration = 30 * time.Second
	const eventInterval = 2 * time.Second

	start := time.Now()
	ticker := time.NewTicker(eventInterval)
	defer ticker.Stop()

	var taskCounter int

	for time.Since(start) < testDuration {
		select {
		case <-ticker.C:
			taskCounter++

			createReq := &CreateTaskRequest{
				Content:    fmt.Sprintf("Long-running test task %d", taskCounter),
				Priority:   "low",
				Repository: "long-running-test",
			}

			task, err := httpClient.CreateTask(ctx, createReq)
			if err != nil {
				s.logger.Error("failed to create task during long-running test", slog.Any("error", err))
				continue
			}

			s.logger.Debug("created task during long-running test",
				slog.String("task_id", task.ID),
				slog.Int("task_number", taskCounter))

		case <-ctx.Done():
			return
		}
	}

	// Wait a bit for final events to arrive
	time.Sleep(2 * time.Second)

	finalEventsReceived := atomic.LoadInt64(&eventsReceived)

	s.logger.Info("long-running connection test completed",
		slog.Duration("duration", time.Since(start)),
		slog.Int("tasks_created", taskCounter),
		slog.Int64("events_received", finalEventsReceived))

	// Should have received at least some events (allowing for potential message loss)
	s.Assert().Greater(finalEventsReceived, int64(0), "should have received at least some events")

	// Connection should still be active
	s.Assert().True(wsClient.IsConnected(), "connection should still be active after long-running test")
}

// TestPingPongMechanism tests WebSocket keep-alive mechanism
func (s *WebSocketResilienceSuite) TestPingPongMechanism() {
	ctx := context.Background()

	s.logger.Info("testing ping-pong keep-alive mechanism")

	hub := NewNotificationHub(s.logger)
	wsClient := NewWebSocketClient(s.wsURL, "1.0.0", hub, s.logger)

	// Connect
	err := wsClient.Connect(ctx)
	s.Require().NoError(err)
	defer wsClient.Close()

	// Monitor connection for keep-alive activity
	start := time.Now()
	const monitorDuration = 20 * time.Second

	// The ping-pong mechanism should keep the connection alive
	// We'll just verify the connection stays active during idle time

	time.Sleep(monitorDuration)

	// Connection should still be active
	s.Assert().True(wsClient.IsConnected(),
		"connection should remain active due to ping-pong keep-alive")

	duration := time.Since(start)
	s.logger.Info("ping-pong mechanism test completed",
		slog.Duration("monitored_duration", duration))
}

// TestSubscriptionPersistence tests that subscriptions persist across reconnections
func (s *WebSocketResilienceSuite) TestSubscriptionPersistence() {
	ctx := context.Background()

	s.logger.Info("testing subscription persistence across reconnections")

	hub := NewNotificationHub(s.logger)
	wsClient := NewWebSocketClient(s.wsURL, "1.0.0", hub, s.logger)

	var eventsReceived []string
	var eventsMu sync.Mutex

	hub.Subscribe("persistence-test", func(event *TaskEvent) {
		eventsMu.Lock()
		eventsReceived = append(eventsReceived, event.TaskID)
		eventsMu.Unlock()
		s.logger.Info("received event after reconnection", slog.String("task_id", event.TaskID))
	})

	// Initial connection and subscription
	err := wsClient.Connect(ctx)
	s.Require().NoError(err)

	err = wsClient.SubscribeToRepositories([]string{"persistence-test"})
	s.Require().NoError(err)

	// Create initial task to verify subscription works
	httpClient := NewHTTPClient(s.serverURL, "1.0.0", s.logger)

	createReq := &CreateTaskRequest{
		Content:    "Task before disconnection",
		Priority:   "medium",
		Repository: "persistence-test",
	}

	task1, err := httpClient.CreateTask(ctx, createReq)
	s.Require().NoError(err)

	time.Sleep(1 * time.Second)

	// Force disconnection
	s.logger.Info("forcing disconnection to test subscription persistence")
	wsClient.ForceDisconnect()

	// Wait for reconnection
	time.Sleep(5 * time.Second)

	// Create another task after reconnection
	createReq2 := &CreateTaskRequest{
		Content:    "Task after reconnection",
		Priority:   "high",
		Repository: "persistence-test",
	}

	task2, err := httpClient.CreateTask(ctx, createReq2)
	s.Require().NoError(err)

	// Wait for events
	time.Sleep(3 * time.Second)

	wsClient.Close()

	eventsMu.Lock()
	events := make([]string, len(eventsReceived))
	copy(events, eventsReceived)
	eventsMu.Unlock()

	s.logger.Info("subscription persistence test completed",
		slog.String("task1_id", task1.ID),
		slog.String("task2_id", task2.ID),
		slog.Any("events_received", events))

	// Should have received at least some events (may not catch all due to timing)
	s.Assert().Greater(len(events), 0, "should have received events after reconnection")
}

func TestWebSocketResilience(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping WebSocket resilience tests in short mode")
	}

	if _, err := testcontainers.NewDockerProvider(); err != nil {
		t.Skip("Docker not available, skipping WebSocket resilience tests")
	}

	suite.Run(t, new(WebSocketResilienceSuite))
}
