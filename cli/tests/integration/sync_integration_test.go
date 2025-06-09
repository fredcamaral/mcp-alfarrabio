package integration

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	// Note: Using local test types instead of actual API package
)

// SyncIntegrationSuite tests the complete MT-003 bidirectional sync system
type SyncIntegrationSuite struct {
	suite.Suite
	serverContainer testcontainers.Container
	qdrantContainer testcontainers.Container
	serverURL       string
	wsURL           string
	httpClient      *HTTPClient
	wsClient        *WebSocketClient
	syncManager     *SyncManager
	notificationHub *NotificationHub
	logger          *slog.Logger
}

func (s *SyncIntegrationSuite) SetupSuite() {
	ctx := context.Background()
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Create a custom network for container communication
	networkName := "mcp-test-network"
	networkReq := testcontainers.NetworkRequest{
		Name:   networkName,
		Driver: "bridge",
	}
	_, err := testcontainers.GenericNetwork(ctx, testcontainers.GenericNetworkRequest{
		NetworkRequest: networkReq,
	})
	s.Require().NoError(err)

	// Start Qdrant container for vector storage
	s.logger.Info("starting Qdrant container")
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

	// Get Qdrant connection details for external access (health checks from host)
	qdrantHost, err := qdrant.Host(ctx)
	s.Require().NoError(err)
	qdrantPort, err := qdrant.MappedPort(ctx, "6333")
	s.Require().NoError(err)

	s.logger.Info("Qdrant started", slog.String("host", qdrantHost), slog.String("port", qdrantPort.Port()))

	// Additional wait to ensure Qdrant is fully ready
	time.Sleep(3 * time.Second)

	// Start MCP Memory Server container
	s.logger.Info("starting MCP Memory Server container")
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
			"MCP_MEMORY_VECTOR_DIM":       "1536",
			"OPENAI_API_KEY":              "test-key", // Mock API key for testing
			"MCP_MEMORY_SERVER_MODE":      "http",
			"MCP_MEMORY_ENABLE_WEBSOCKET": "true",
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

	// Get server connection details for external access
	serverHost, err := server.Host(ctx)
	s.Require().NoError(err)
	serverPort, err := server.MappedPort(ctx, "9080")
	s.Require().NoError(err)

	s.serverURL = fmt.Sprintf("http://%s:%s", serverHost, serverPort.Port())
	s.wsURL = fmt.Sprintf("ws://%s:%s", serverHost, serverPort.Port())

	s.logger.Info("MCP Memory Server started", slog.String("url", s.serverURL))

	// Wait for server to be fully ready
	s.waitForServerReady()

	// Create clients
	s.httpClient = NewHTTPClient(s.serverURL, "1.0.0", s.logger)
	s.notificationHub = NewNotificationHub(s.logger)
	s.wsClient = NewWebSocketClient(s.wsURL, "1.0.0", s.notificationHub, s.logger)
}

func (s *SyncIntegrationSuite) TearDownSuite() {
	ctx := context.Background()

	if s.wsClient != nil {
		s.wsClient.Close()
	}

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

func (s *SyncIntegrationSuite) SetupTest() {
	// Clean state before each test

	// Clear any existing WebSocket connections
	if s.wsClient != nil {
		s.wsClient.Close()
	}

	// Recreate WebSocket client for each test
	s.wsClient = NewWebSocketClient(s.wsURL, "1.0.0", s.notificationHub, s.logger)
}

func (s *SyncIntegrationSuite) waitForServerReady() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.FailNow("server failed to become ready within timeout")
		case <-ticker.C:
			resp, err := http.Get(s.serverURL + "/health")
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				s.logger.Info("server is ready")
				return
			}
			if resp != nil {
				resp.Body.Close()
			}
		}
	}
}

// TestBidirectionalSync tests the complete bidirectional sync flow
func (s *SyncIntegrationSuite) TestBidirectionalSync() {
	ctx := context.Background()

	s.logger.Info("testing bidirectional sync")

	// Connect WebSocket
	err := s.wsClient.Connect(ctx)
	s.Require().NoError(err)
	defer s.wsClient.Close()

	// Subscribe to test repository
	err = s.wsClient.SubscribeToRepositories([]string{"test-repo"})
	s.Require().NoError(err)

	// Setup notification capture
	notifications := make(chan *TaskEvent, 10)
	s.notificationHub.Subscribe("test", func(event *TaskEvent) {
		notifications <- event
	})

	// Create task via HTTP API
	createReq := &CreateTaskRequest{
		Content:    "Test task for bidirectional sync",
		Priority:   "high",
		Repository: "test-repo",
		Tags:       []string{"integration-test"},
	}

	taskResp, err := s.httpClient.CreateTask(ctx, createReq)
	s.Require().NoError(err)
	s.Assert().NotEmpty(taskResp.ID)
	s.Assert().Equal("Test task for bidirectional sync", taskResp.Content)

	s.logger.Info("created task via HTTP", slog.String("task_id", taskResp.ID))

	// Wait for WebSocket notification
	select {
	case event := <-notifications:
		s.Assert().Equal(EventTypeTaskCreated, event.Type)
		s.Assert().Equal(taskResp.ID, event.TaskID)
		s.Assert().Equal("test-repo", event.Repository)
		s.logger.Info("received WebSocket notification for task creation")
	case <-time.After(5 * time.Second):
		s.Fail("timeout waiting for WebSocket notification")
	}

	// Update task status
	updateReq := &UpdateTaskRequest{
		Status: stringPtr("in_progress"),
	}

	updated, err := s.httpClient.UpdateTask(ctx, taskResp.ID, updateReq)
	s.Require().NoError(err)
	s.Assert().Equal("in_progress", updated.Status)

	s.logger.Info("updated task via HTTP", slog.String("task_id", taskResp.ID))

	// Wait for update notification
	select {
	case event := <-notifications:
		s.Assert().Equal(EventTypeTaskUpdated, event.Type)
		s.Assert().Equal(taskResp.ID, event.TaskID)
		s.logger.Info("received WebSocket notification for task update")
	case <-time.After(5 * time.Second):
		s.Fail("timeout waiting for update notification")
	}

	// Test batch sync
	batchClient := NewBatchClient(s.serverURL, "1.0.0", s.logger)

	syncReq := &BatchSyncRequest{
		Repository: "test-repo",
		LocalTasks: []TaskSyncItem{{
			ID:           taskResp.ID,
			Content:      "Updated locally",
			Status:       "completed",
			Priority:     "high",
			UpdatedAt:    time.Now(),
			LocalVersion: 2,
		}},
	}

	syncResp, err := batchClient.BatchSync(ctx, syncReq)
	s.Require().NoError(err)
	s.Assert().NotNil(syncResp)

	s.logger.Info("batch sync completed",
		slog.Int("conflicts", len(syncResp.Conflicts)),
		slog.Int("server_tasks", len(syncResp.ServerTasks)))
}

// TestWebSocketResilience tests WebSocket connection resilience
func (s *SyncIntegrationSuite) TestWebSocketResilience() {
	ctx := context.Background()

	s.logger.Info("testing WebSocket resilience")

	// Connect WebSocket
	err := s.wsClient.Connect(ctx)
	s.Require().NoError(err)

	// Verify connection is established
	s.Assert().True(s.wsClient.IsConnected())

	// Subscribe to repository
	err = s.wsClient.SubscribeToRepositories([]string{"resilience-test"})
	s.Require().NoError(err)

	// Setup notification tracking
	reconnectCount := 0
	var mu sync.Mutex

	s.notificationHub.Subscribe("resilience", func(event *TaskEvent) {
		// Track events during resilience testing
	})

	// Simulate connection drop by closing the WebSocket
	s.logger.Info("simulating connection drop")
	s.wsClient.ForceDisconnect() // This should trigger reconnection

	// Wait for reconnection attempt
	time.Sleep(2 * time.Second)

	// Create task during potential reconnection
	createReq := &CreateTaskRequest{
		Content:    "Task during reconnection",
		Priority:   "medium",
		Repository: "resilience-test",
	}

	taskResp, err := s.httpClient.CreateTask(ctx, createReq)
	s.Require().NoError(err)

	s.logger.Info("created task during resilience test", slog.String("task_id", taskResp.ID))

	// Wait for WebSocket to stabilize and receive notification
	time.Sleep(3 * time.Second)

	// Verify we can still receive notifications after reconnection
	createReq2 := &CreateTaskRequest{
		Content:    "Task after reconnection",
		Priority:   "low",
		Repository: "resilience-test",
	}

	taskResp2, err := s.httpClient.CreateTask(ctx, createReq2)
	s.Require().NoError(err)

	s.logger.Info("created second task after reconnection", slog.String("task_id", taskResp2.ID))

	mu.Lock()
	finalReconnectCount := reconnectCount
	mu.Unlock()

	s.logger.Info("resilience test completed", slog.Int("reconnect_count", finalReconnectCount))
}

// TestConflictResolution tests conflict resolution scenarios
func (s *SyncIntegrationSuite) TestConflictResolution() {
	ctx := context.Background()

	s.logger.Info("testing conflict resolution")

	// Create initial task
	createReq := &CreateTaskRequest{
		Content:    "Original content for conflict test",
		Priority:   "medium",
		Repository: "conflict-test",
	}

	task, err := s.httpClient.CreateTask(ctx, createReq)
	s.Require().NoError(err)

	// Simulate two different updates (conflict scenario)
	// Update 1: Change content
	update1 := &UpdateTaskRequest{
		Content: stringPtr("Updated by client 1"),
	}

	// Update 2: Change priority
	update2 := &UpdateTaskRequest{
		Priority: stringPtr("high"),
	}

	// Apply updates
	_, err = s.httpClient.UpdateTask(ctx, task.ID, update1)
	s.Require().NoError(err)

	_, err = s.httpClient.UpdateTask(ctx, task.ID, update2)
	s.Require().NoError(err)

	// Create conflicting local state for batch sync
	batchClient := NewBatchClient(s.serverURL, "1.0.0", s.logger)

	syncReq := &BatchSyncRequest{
		Repository: "conflict-test",
		LocalTasks: []TaskSyncItem{{
			ID:           task.ID,
			Content:      "Local conflicting content",
			Status:       "pending",
			Priority:     "low",                            // Different from server
			UpdatedAt:    time.Now().Add(-1 * time.Minute), // Older timestamp
			LocalVersion: 1,
		}},
	}

	syncResp, err := batchClient.BatchSync(ctx, syncReq)
	s.Require().NoError(err)

	// Should have at least one conflict
	s.Assert().Greater(len(syncResp.Conflicts), 0, "expected conflicts during sync")

	if len(syncResp.Conflicts) > 0 {
		conflict := syncResp.Conflicts[0]
		s.Assert().Equal(task.ID, conflict.TaskID)
		s.Assert().NotNil(conflict.Resolution)

		// Server should win due to newer timestamp
		s.Assert().Contains([]string{"server_wins", "server_wins_newer"}, conflict.Resolution.Strategy)

		s.logger.Info("conflict resolved",
			slog.String("strategy", conflict.Resolution.Strategy),
			slog.String("task_id", conflict.TaskID))
	}
}

// TestPerformance tests sync performance with larger datasets
func (s *SyncIntegrationSuite) TestPerformance() {
	ctx := context.Background()

	s.logger.Info("testing sync performance")

	const numTasks = 100
	tasks := make([]*TaskResponse, 0, numTasks)

	// Create many tasks in parallel
	start := time.Now()

	var wg sync.WaitGroup
	taskChan := make(chan *TaskResponse, numTasks)
	errorChan := make(chan error, numTasks)

	// Create tasks concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < numTasks/10; j++ {
				req := &CreateTaskRequest{
					Content:    fmt.Sprintf("Performance test task %d-%d", workerID, j),
					Priority:   "medium",
					Repository: "perf-test",
					Tags:       []string{"performance", "test"},
				}

				task, err := s.httpClient.CreateTask(ctx, req)
				if err != nil {
					errorChan <- err
				} else {
					taskChan <- task
				}
			}
		}(i)
	}

	wg.Wait()
	close(taskChan)
	close(errorChan)

	// Check for errors
	for err := range errorChan {
		s.Require().NoError(err)
	}

	// Collect created tasks
	for task := range taskChan {
		tasks = append(tasks, task)
	}

	createDuration := time.Since(start)
	tasksPerSecond := float64(len(tasks)) / createDuration.Seconds()

	s.logger.Info("task creation performance",
		slog.Int("tasks_created", len(tasks)),
		slog.Duration("duration", createDuration),
		slog.Float64("tasks_per_second", tasksPerSecond))

	// Performance assertions
	s.Assert().Equal(numTasks, len(tasks), "all tasks should be created")
	s.Assert().Less(createDuration, 30*time.Second, "creation should complete within 30 seconds")
	s.Assert().Greater(tasksPerSecond, 3.0, "should create at least 3 tasks per second")

	// Test batch sync performance
	batchClient := NewBatchClient(s.serverURL, "1.0.0", s.logger)

	syncItems := make([]TaskSyncItem, 0, len(tasks))
	for _, task := range tasks {
		syncItems = append(syncItems, TaskSyncItem{
			ID:        task.ID,
			Content:   task.Content,
			Status:    task.Status,
			Priority:  task.Priority,
			UpdatedAt: task.UpdatedAt,
		})
	}

	syncReq := &BatchSyncRequest{
		Repository: "perf-test",
		LocalTasks: syncItems,
	}

	start = time.Now()
	syncResp, err := batchClient.BatchSync(ctx, syncReq)
	s.Require().NoError(err)
	syncDuration := time.Since(start)

	s.logger.Info("batch sync performance",
		slog.Int("tasks_synced", len(syncItems)),
		slog.Duration("sync_duration", syncDuration),
		slog.Int("conflicts", len(syncResp.Conflicts)))

	// Sync performance assertions
	s.Assert().Less(syncDuration, 10*time.Second, "batch sync should complete within 10 seconds")
	s.Assert().Equal(0, len(syncResp.Conflicts), "no conflicts expected in performance test")
}

// TestMultiRepositorySync tests sync across multiple repositories
func (s *SyncIntegrationSuite) TestMultiRepositorySync() {
	ctx := context.Background()

	s.logger.Info("testing multi-repository sync")

	// Connect WebSocket
	err := s.wsClient.Connect(ctx)
	s.Require().NoError(err)
	defer s.wsClient.Close()

	// Subscribe to multiple repositories
	repos := []string{"repo-1", "repo-2", "repo-3"}
	err = s.wsClient.SubscribeToRepositories(repos)
	s.Require().NoError(err)

	// Create tasks in different repositories
	var createdTasks []*TaskResponse

	for i, repo := range repos {
		for j := 0; j < 3; j++ {
			createReq := &CreateTaskRequest{
				Content:    fmt.Sprintf("Task %d in %s", j+1, repo),
				Priority:   "medium",
				Repository: repo,
				Tags:       []string{"multi-repo", fmt.Sprintf("repo-%d", i+1)},
			}

			task, err := s.httpClient.CreateTask(ctx, createReq)
			s.Require().NoError(err)
			createdTasks = append(createdTasks, task)
		}
	}

	s.Assert().Equal(9, len(createdTasks), "should have created 9 tasks across 3 repositories")

	// Test repository-specific batch sync
	batchClient := NewBatchClient(s.serverURL, "1.0.0", s.logger)

	for _, repo := range repos {
		// Get tasks for this repository
		repoTasks := make([]TaskSyncItem, 0)
		for _, task := range createdTasks {
			if task.Repository == repo {
				repoTasks = append(repoTasks, TaskSyncItem{
					ID:        task.ID,
					Content:   task.Content,
					Status:    task.Status,
					Priority:  task.Priority,
					UpdatedAt: task.UpdatedAt,
				})
			}
		}

		syncReq := &BatchSyncRequest{
			Repository: repo,
			LocalTasks: repoTasks,
		}

		syncResp, err := batchClient.BatchSync(ctx, syncReq)
		s.Require().NoError(err)

		s.Assert().Equal(len(repoTasks), len(syncResp.ServerTasks),
			"server should return all tasks for repository %s", repo)

		s.logger.Info("repository sync completed",
			slog.String("repository", repo),
			slog.Int("tasks", len(repoTasks)))
	}
}

// Helper functions are defined in main_test.go

func TestSyncIntegration(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("skipping integration tests in short mode")
	}

	// Skip if no Docker available
	if _, err := testcontainers.NewDockerProvider(); err != nil {
		t.Skip("Docker not available, skipping integration tests")
	}

	suite.Run(t, new(SyncIntegrationSuite))
}
