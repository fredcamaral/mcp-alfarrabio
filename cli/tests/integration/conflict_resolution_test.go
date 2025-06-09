package integration

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// ConflictResolutionSuite tests various conflict resolution scenarios
type ConflictResolutionSuite struct {
	suite.Suite
	serverContainer testcontainers.Container
	qdrantContainer testcontainers.Container
	serverURL       string
	httpClient      *HTTPClient
	batchClient     *BatchClient
	logger          *slog.Logger
}

func (s *ConflictResolutionSuite) SetupSuite() {
	ctx := context.Background()
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Start Qdrant container
	s.logger.Info("starting Qdrant container for conflict resolution tests")
	qdrantReq := testcontainers.ContainerRequest{
		Image:        "qdrant/qdrant:latest",
		ExposedPorts: []string{"6333/tcp"},
		WaitingFor:   wait.ForHTTP("/").WithPort("6333/tcp").WithStartupTimeout(60 * time.Second),
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

	// Start MCP Memory Server
	s.logger.Info("starting MCP Memory Server for conflict resolution tests")
	serverReq := testcontainers.ContainerRequest{
		Image:        "ghcr.io/lerianstudio/lerian-mcp-memory:latest",
		ExposedPorts: []string{"9080/tcp"},
		WaitingFor:   wait.ForHTTP("/health").WithPort("9080/tcp").WithStartupTimeout(120 * time.Second),
		Env: map[string]string{
			"MCP_HOST_PORT":               "9080",
			"QDRANT_HOST_PORT":            fmt.Sprintf("%s:%s", qdrantHost, qdrantPort.Port()),
			"MCP_MEMORY_LOG_LEVEL":        "debug",
			"OPENAI_API_KEY":              "test-key",
			"MCP_MEMORY_SERVER_MODE":      "http",
			"MCP_MEMORY_ENABLE_WEBSOCKET": "true",
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

	// Create clients
	s.httpClient = NewHTTPClient(s.serverURL, "1.0.0", s.logger)
	s.batchClient = NewBatchClient(s.serverURL, "1.0.0", s.logger)

	s.logger.Info("conflict resolution test environment ready", slog.String("server_url", s.serverURL))
}

func (s *ConflictResolutionSuite) TearDownSuite() {
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

// TestSimpleContentConflict tests basic content conflicts
func (s *ConflictResolutionSuite) TestSimpleContentConflict() {
	ctx := context.Background()

	s.logger.Info("testing simple content conflict")

	// Create initial task
	createReq := &CreateTaskRequest{
		Content:    "Original task content",
		Priority:   "medium",
		Repository: "content-conflict-test",
		Tags:       []string{"conflict-test"},
	}

	task, err := s.httpClient.CreateTask(ctx, createReq)
	s.Require().NoError(err)

	// Update task on server
	updateReq := &UpdateTaskRequest{
		Content: stringPtr("Server updated content"),
	}

	serverTask, err := s.httpClient.UpdateTask(ctx, task.ID, updateReq)
	s.Require().NoError(err)

	s.logger.Info("server updated task",
		slog.String("task_id", task.ID),
		slog.String("server_content", serverTask.Content))

	// Simulate local changes (conflicting content)
	localSyncItem := TaskSyncItem{
		ID:           task.ID,
		Content:      "Local updated content", // Different from server
		Status:       task.Status,
		Priority:     task.Priority,
		UpdatedAt:    task.UpdatedAt.Add(-1 * time.Minute), // Older timestamp
		LocalVersion: 1,
	}

	// Perform batch sync to detect conflict
	syncReq := &BatchSyncRequest{
		Repository: "content-conflict-test",
		LocalTasks: []TaskSyncItem{localSyncItem},
	}

	syncResp, err := s.batchClient.BatchSync(ctx, syncReq)
	s.Require().NoError(err)

	// Should detect conflict
	s.Assert().Greater(len(syncResp.Conflicts), 0, "should detect content conflict")

	if len(syncResp.Conflicts) > 0 {
		conflict := syncResp.Conflicts[0]
		s.Assert().Equal(task.ID, conflict.TaskID)
		s.Assert().NotNil(conflict.Resolution)

		s.logger.Info("conflict detected and resolved",
			slog.String("strategy", conflict.Resolution.Strategy),
			slog.String("reason", conflict.Reason))

		// Server should win due to newer timestamp
		s.Assert().Contains([]string{"server_wins", "server_wins_newer"}, conflict.Resolution.Strategy)

		if conflict.Resolution.ResolvedTask != nil {
			s.Assert().Equal("Server updated content", conflict.Resolution.ResolvedTask.Content)
		}
	}
}

// TestPriorityConflict tests priority conflicts
func (s *ConflictResolutionSuite) TestPriorityConflict() {
	ctx := context.Background()

	s.logger.Info("testing priority conflict")

	// Create task
	createReq := &CreateTaskRequest{
		Content:    "Task for priority conflict",
		Priority:   "low",
		Repository: "priority-conflict-test",
	}

	task, err := s.httpClient.CreateTask(ctx, createReq)
	s.Require().NoError(err)

	// Update priority on server
	updateReq := &UpdateTaskRequest{
		Priority: stringPtr("high"),
	}

	_, err = s.httpClient.UpdateTask(ctx, task.ID, updateReq)
	s.Require().NoError(err)

	// Simulate local priority change
	localSyncItem := TaskSyncItem{
		ID:           task.ID,
		Content:      task.Content,
		Status:       task.Status,
		Priority:     "medium",                              // Different from server's "high"
		UpdatedAt:    task.UpdatedAt.Add(-30 * time.Second), // Slightly older
		LocalVersion: 1,
	}

	syncReq := &BatchSyncRequest{
		Repository: "priority-conflict-test",
		LocalTasks: []TaskSyncItem{localSyncItem},
	}

	syncResp, err := s.batchClient.BatchSync(ctx, syncReq)
	s.Require().NoError(err)

	// Should detect priority conflict
	s.Assert().Greater(len(syncResp.Conflicts), 0, "should detect priority conflict")

	if len(syncResp.Conflicts) > 0 {
		conflict := syncResp.Conflicts[0]
		s.Assert().Equal(task.ID, conflict.TaskID)

		s.logger.Info("priority conflict resolved",
			slog.String("strategy", conflict.Resolution.Strategy),
			slog.String("local_priority", localSyncItem.Priority),
			slog.String("server_priority", "high"))

		// Should resolve with server's priority
		if conflict.Resolution.ResolvedTask != nil {
			s.Assert().Equal("high", conflict.Resolution.ResolvedTask.Priority)
		}
	}
}

// TestStatusConflict tests status transition conflicts
func (s *ConflictResolutionSuite) TestStatusConflict() {
	ctx := context.Background()

	s.logger.Info("testing status conflict")

	// Create task
	createReq := &CreateTaskRequest{
		Content:    "Task for status conflict",
		Priority:   "medium",
		Repository: "status-conflict-test",
	}

	task, err := s.httpClient.CreateTask(ctx, createReq)
	s.Require().NoError(err)

	// Update status on server
	updateReq := &UpdateTaskRequest{
		Status: stringPtr("in_progress"),
	}

	_, err = s.httpClient.UpdateTask(ctx, task.ID, updateReq)
	s.Require().NoError(err)

	// Simulate local status change to different state
	localSyncItem := TaskSyncItem{
		ID:           task.ID,
		Content:      task.Content,
		Status:       "completed", // Different from server's "in_progress"
		Priority:     task.Priority,
		UpdatedAt:    task.UpdatedAt.Add(-2 * time.Minute), // Older
		LocalVersion: 1,
	}

	syncReq := &BatchSyncRequest{
		Repository: "status-conflict-test",
		LocalTasks: []TaskSyncItem{localSyncItem},
	}

	syncResp, err := s.batchClient.BatchSync(ctx, syncReq)
	s.Require().NoError(err)

	// Should detect status conflict
	s.Assert().Greater(len(syncResp.Conflicts), 0, "should detect status conflict")

	if len(syncResp.Conflicts) > 0 {
		conflict := syncResp.Conflicts[0]
		s.Assert().Equal(task.ID, conflict.TaskID)

		s.logger.Info("status conflict resolved",
			slog.String("strategy", conflict.Resolution.Strategy),
			slog.String("local_status", localSyncItem.Status),
			slog.String("server_status", "in_progress"))

		// Should prefer server's status due to newer timestamp
		if conflict.Resolution.ResolvedTask != nil {
			s.Assert().Equal("in_progress", conflict.Resolution.ResolvedTask.Status)
		}
	}
}

// TestMultipleConflicts tests tasks with multiple conflicting fields
func (s *ConflictResolutionSuite) TestMultipleConflicts() {
	ctx := context.Background()

	s.logger.Info("testing multiple field conflicts")

	// Create task
	createReq := &CreateTaskRequest{
		Content:    "Original content",
		Priority:   "low",
		Repository: "multi-conflict-test",
	}

	task, err := s.httpClient.CreateTask(ctx, createReq)
	s.Require().NoError(err)

	// Make multiple updates on server
	updateReq := &UpdateTaskRequest{
		Content:  stringPtr("Server updated content"),
		Priority: stringPtr("high"),
		Status:   stringPtr("in_progress"),
	}

	_, err = s.httpClient.UpdateTask(ctx, task.ID, updateReq)
	s.Require().NoError(err)

	// Simulate local changes to all fields (conflicting)
	localSyncItem := TaskSyncItem{
		ID:           task.ID,
		Content:      "Local updated content",              // Different
		Status:       "completed",                          // Different
		Priority:     "medium",                             // Different
		UpdatedAt:    task.UpdatedAt.Add(-1 * time.Minute), // Older
		LocalVersion: 1,
	}

	syncReq := &BatchSyncRequest{
		Repository: "multi-conflict-test",
		LocalTasks: []TaskSyncItem{localSyncItem},
	}

	syncResp, err := s.batchClient.BatchSync(ctx, syncReq)
	s.Require().NoError(err)

	// Should detect conflict
	s.Assert().Greater(len(syncResp.Conflicts), 0, "should detect multiple field conflicts")

	if len(syncResp.Conflicts) > 0 {
		conflict := syncResp.Conflicts[0]
		s.Assert().Equal(task.ID, conflict.TaskID)

		s.logger.Info("multiple conflicts resolved",
			slog.String("strategy", conflict.Resolution.Strategy),
			slog.String("reason", conflict.Reason))

		// Should resolve with server values
		if conflict.Resolution.ResolvedTask != nil {
			resolved := conflict.Resolution.ResolvedTask
			s.Assert().Equal("Server updated content", resolved.Content)
			s.Assert().Equal("high", resolved.Priority)
			s.Assert().Equal("in_progress", resolved.Status)
		}
	}
}

// TestQdrantTruthResolution tests conflict resolution using Qdrant as authoritative source
func (s *ConflictResolutionSuite) TestQdrantTruthResolution() {
	ctx := context.Background()

	s.logger.Info("testing Qdrant truth-based conflict resolution")

	// Create task and let it sync to Qdrant
	createReq := &CreateTaskRequest{
		Content:    "Task for Qdrant truth test",
		Priority:   "medium",
		Repository: "qdrant-truth-test",
		Tags:       []string{"qdrant", "truth-test"},
	}

	task, err := s.httpClient.CreateTask(ctx, createReq)
	s.Require().NoError(err)

	// Wait for task to be indexed in Qdrant
	time.Sleep(2 * time.Second)

	// Update task on server (this should also update Qdrant)
	updateReq := &UpdateTaskRequest{
		Content: stringPtr("Server updated for Qdrant test"),
	}

	serverTask, err := s.httpClient.UpdateTask(ctx, task.ID, updateReq)
	s.Require().NoError(err)

	// Wait for Qdrant to be updated
	time.Sleep(2 * time.Second)

	// Create conflicting local state
	localSyncItem := TaskSyncItem{
		ID:           task.ID,
		Content:      "Local conflicting content",
		Status:       task.Status,
		Priority:     task.Priority,
		UpdatedAt:    serverTask.UpdatedAt, // Same timestamp to force Qdrant lookup
		LocalVersion: 2,
	}

	syncReq := &BatchSyncRequest{
		Repository: "qdrant-truth-test",
		LocalTasks: []TaskSyncItem{localSyncItem},
	}

	syncResp, err := s.batchClient.BatchSync(ctx, syncReq)
	s.Require().NoError(err)

	// Should detect conflict and use Qdrant for resolution
	s.Assert().Greater(len(syncResp.Conflicts), 0, "should detect conflict for Qdrant resolution")

	if len(syncResp.Conflicts) > 0 {
		conflict := syncResp.Conflicts[0]
		s.Assert().Equal(task.ID, conflict.TaskID)

		s.logger.Info("Qdrant-based conflict resolved",
			slog.String("strategy", conflict.Resolution.Strategy),
			slog.String("reason", conflict.Reason))

		// Should resolve using Qdrant's version (which should match server)
		s.Assert().Contains([]string{"server_wins", "qdrant_truth"}, conflict.Resolution.Strategy)

		if conflict.Resolution.ResolvedTask != nil {
			s.Assert().Equal("Server updated for Qdrant test", conflict.Resolution.ResolvedTask.Content)
		}
	}
}

// TestConflictResolutionStrategies tests different resolution strategies
func (s *ConflictResolutionSuite) TestConflictResolutionStrategies() {
	ctx := context.Background()

	s.logger.Info("testing various conflict resolution strategies")

	testCases := []struct {
		name                 string
		serverUpdateTime     time.Duration // relative to base time
		localUpdateTime      time.Duration // relative to base time
		expectedStrategyType string        // prefix of expected strategy
	}{
		{
			name:                 "server_newer",
			serverUpdateTime:     0,                // newer
			localUpdateTime:      -5 * time.Minute, // older
			expectedStrategyType: "server_wins",
		},
		{
			name:                 "local_newer",
			serverUpdateTime:     -5 * time.Minute, // older
			localUpdateTime:      0,                // newer
			expectedStrategyType: "local_wins",
		},
		{
			name:                 "same_time",
			serverUpdateTime:     0,       // same
			localUpdateTime:      0,       // same
			expectedStrategyType: "merge", // should attempt merge or default to server
		},
	}

	baseTime := time.Now()

	for i, tc := range testCases {
		s.Run(tc.name, func() {
			s.logger.Info("running conflict strategy test", slog.String("test_case", tc.name))

			// Create task
			createReq := &CreateTaskRequest{
				Content:    fmt.Sprintf("Task for strategy test %d", i),
				Priority:   "medium",
				Repository: "strategy-test",
			}

			task, err := s.httpClient.CreateTask(ctx, createReq)
			s.Require().NoError(err)

			// Update on server with specific timestamp simulation
			updateReq := &UpdateTaskRequest{
				Content: stringPtr(fmt.Sprintf("Server content %d", i)),
			}

			_, err = s.httpClient.UpdateTask(ctx, task.ID, updateReq)
			s.Require().NoError(err)

			// Create local state with timestamp control
			localTime := baseTime.Add(tc.localUpdateTime)

			localSyncItem := TaskSyncItem{
				ID:           task.ID,
				Content:      fmt.Sprintf("Local content %d", i),
				Status:       task.Status,
				Priority:     task.Priority,
				UpdatedAt:    localTime,
				LocalVersion: 1,
			}

			syncReq := &BatchSyncRequest{
				Repository: "strategy-test",
				LocalTasks: []TaskSyncItem{localSyncItem},
			}

			syncResp, err := s.batchClient.BatchSync(ctx, syncReq)
			s.Require().NoError(err)

			if len(syncResp.Conflicts) > 0 {
				conflict := syncResp.Conflicts[0]

				s.logger.Info("strategy test result",
					slog.String("test_case", tc.name),
					slog.String("strategy", conflict.Resolution.Strategy),
					slog.String("expected_type", tc.expectedStrategyType))

				// Check if strategy matches expected type
				if tc.expectedStrategyType == "merge" {
					// For same timestamp, could be merge or default to server
					s.Assert().Contains([]string{"merge", "server_wins"}, conflict.Resolution.Strategy)
				} else {
					s.Assert().Contains(conflict.Resolution.Strategy, tc.expectedStrategyType)
				}
			} else if tc.expectedStrategyType == "local_wins" {
				// If local is newer, might not create conflict if server accepts local
				s.logger.Info("no conflict detected - local might have been accepted", slog.String("test_case", tc.name))
			}
		})
	}
}

// TestBatchConflictResolution tests conflict resolution in batch operations
func (s *ConflictResolutionSuite) TestBatchConflictResolution() {
	ctx := context.Background()

	s.logger.Info("testing batch conflict resolution")

	const numTasks = 5
	var tasks []*TaskResponse

	// Create multiple tasks
	for i := 0; i < numTasks; i++ {
		createReq := &CreateTaskRequest{
			Content:    fmt.Sprintf("Batch test task %d", i),
			Priority:   "medium",
			Repository: "batch-conflict-test",
		}

		task, err := s.httpClient.CreateTask(ctx, createReq)
		s.Require().NoError(err)
		tasks = append(tasks, task)
	}

	// Update some tasks on server
	for i := 0; i < 3; i++ {
		updateReq := &UpdateTaskRequest{
			Content: stringPtr(fmt.Sprintf("Server updated task %d", i)),
		}

		_, err := s.httpClient.UpdateTask(ctx, tasks[i].ID, updateReq)
		s.Require().NoError(err)
	}

	// Create conflicting local changes for all tasks
	var localTasks []TaskSyncItem
	for i, task := range tasks {
		localTask := TaskSyncItem{
			ID:           task.ID,
			Content:      fmt.Sprintf("Local updated task %d", i),
			Status:       task.Status,
			Priority:     task.Priority,
			UpdatedAt:    task.UpdatedAt.Add(-1 * time.Minute), // Older
			LocalVersion: 1,
		}
		localTasks = append(localTasks, localTask)
	}

	// Perform batch sync
	syncReq := &BatchSyncRequest{
		Repository: "batch-conflict-test",
		LocalTasks: localTasks,
	}

	syncResp, err := s.batchClient.BatchSync(ctx, syncReq)
	s.Require().NoError(err)

	// Should have conflicts for the 3 updated tasks
	s.Assert().GreaterOrEqual(len(syncResp.Conflicts), 3, "should have conflicts for updated tasks")

	conflictTaskIDs := make(map[string]bool)
	for _, conflict := range syncResp.Conflicts {
		conflictTaskIDs[conflict.TaskID] = true

		s.logger.Info("batch conflict resolved",
			slog.String("task_id", conflict.TaskID),
			slog.String("strategy", conflict.Resolution.Strategy))

		// All conflicts should be resolved with server wins (due to newer timestamp)
		s.Assert().Contains([]string{"server_wins", "server_wins_newer"}, conflict.Resolution.Strategy)
	}

	s.logger.Info("batch conflict resolution completed",
		slog.Int("total_tasks", numTasks),
		slog.Int("conflicts_detected", len(syncResp.Conflicts)))
}

func TestConflictResolution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping conflict resolution tests in short mode")
	}

	if _, err := testcontainers.NewDockerProvider(); err != nil {
		t.Skip("Docker not available, skipping conflict resolution tests")
	}

	suite.Run(t, new(ConflictResolutionSuite))
}
