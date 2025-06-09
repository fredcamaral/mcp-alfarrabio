package integration

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestBasicIntegration demonstrates the integration test infrastructure
func TestBasicIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration tests in short mode")
	}

	if _, err := testcontainers.NewDockerProvider(); err != nil {
		t.Skip("Docker not available, skipping integration tests")
	}

	ctx := context.Background()

	// Test Qdrant container startup
	t.Run("qdrant_container", func(t *testing.T) {
		req := testcontainers.ContainerRequest{
			Image:        "qdrant/qdrant:latest",
			ExposedPorts: []string{"6333/tcp"},
			WaitingFor:   wait.ForHTTP("/").WithPort("6333/tcp").WithStartupTimeout(30 * time.Second),
		}

		container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		require.NoError(t, err)
		defer container.Terminate(ctx)

		host, err := container.Host(ctx)
		require.NoError(t, err)
		port, err := container.MappedPort(ctx, "6333")
		require.NoError(t, err)

		assert.NotEmpty(t, host)
		assert.NotEmpty(t, port.Port())

		t.Logf("Qdrant container started at %s:%s", host, port.Port())
	})

	// Test HTTP client functionality
	t.Run("http_client", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
		client := NewHTTPClient("http://localhost:9080", "1.0.0", logger)
		assert.NotNil(t, client)

		// Test mock task creation
		req := &CreateTaskRequest{
			Content:    "Test integration task",
			Priority:   "medium",
			Repository: "test-repo",
		}

		task, err := client.CreateTask(ctx, req)
		require.NoError(t, err)
		assert.NotEmpty(t, task.ID)
		assert.Equal(t, "Test integration task", task.Content)
		assert.Equal(t, "pending", task.Status)

		t.Logf("Created mock task: %s", task.ID)
	})

	// Test WebSocket client functionality
	t.Run("websocket_client", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
		hub := NewNotificationHub(logger)
		client := NewWebSocketClient("ws://localhost:9080/ws", "1.0.0", hub, logger)
		assert.NotNil(t, client)

		err := client.Connect(ctx)
		require.NoError(t, err)
		assert.True(t, client.IsConnected())

		err = client.SubscribeToRepositories([]string{"test-repo"})
		require.NoError(t, err)

		client.Close()
		assert.False(t, client.IsConnected())

		t.Log("WebSocket client mock functionality validated")
	})

	// Test batch sync functionality
	t.Run("batch_sync", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
		client := NewBatchClient("http://localhost:9080", "1.0.0", logger)
		assert.NotNil(t, client)

		req := &BatchSyncRequest{
			Repository: "test-repo",
			LocalTasks: []TaskSyncItem{{
				ID:        "test-task-1",
				Content:   "Local task content",
				Status:    "pending",
				Priority:  "medium",
				UpdatedAt: time.Now().Add(-2 * time.Minute), // Old timestamp to trigger conflict
			}},
		}

		resp, err := client.BatchSync(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.SyncToken)

		// Mock should create conflict for old tasks
		assert.Len(t, resp.Conflicts, 1)
		assert.Equal(t, "server_wins", resp.Conflicts[0].Resolution.Strategy)

		t.Logf("Batch sync completed with %d conflicts", len(resp.Conflicts))
	})
}

// Dedicated test functions are in their respective files:
// - network_resilience_test.go for network resilience testing
// - conflict_resolution_test.go for conflict resolution testing
