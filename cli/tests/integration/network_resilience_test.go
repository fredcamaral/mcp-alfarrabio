package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// NetworkResilienceSuite tests network failure scenarios and recovery
type NetworkResilienceSuite struct {
	suite.Suite
	serverContainer testcontainers.Container
	qdrantContainer testcontainers.Container
	toxiContainer   testcontainers.Container
	serverURL       string
	proxyURL        string
	directURL       string
	logger          *slog.Logger
}

func (s *NetworkResilienceSuite) SetupSuite() {
	ctx := context.Background()
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Start Qdrant container
	s.logger.Info("starting Qdrant container for network tests")
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
	s.logger.Info("starting MCP Memory Server for network tests")
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

	s.directURL = fmt.Sprintf("http://%s:%s", serverHost, serverPort.Port())

	// Start Toxiproxy for network simulation
	s.logger.Info("starting Toxiproxy container")
	toxiReq := testcontainers.ContainerRequest{
		Image:        "ghcr.io/shopify/toxiproxy:2.5.0",
		ExposedPorts: []string{"8474/tcp", "8475/tcp"}, // 8474 = API, 8475 = proxy
		WaitingFor:   wait.ForHTTP("/").WithPort("8474/tcp").WithStartupTimeout(30 * time.Second),
	}

	toxi, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: toxiReq,
		Started:          true,
	})
	s.Require().NoError(err)
	s.toxiContainer = toxi

	toxiHost, err := toxi.Host(ctx)
	s.Require().NoError(err)
	toxiAPIPort, err := toxi.MappedPort(ctx, "8474")
	s.Require().NoError(err)
	toxiProxyPort, err := toxi.MappedPort(ctx, "8475")
	s.Require().NoError(err)

	// Configure Toxiproxy to proxy our server
	toxiAPIURL := fmt.Sprintf("http://%s:%s", toxiHost, toxiAPIPort.Port())
	s.proxyURL = fmt.Sprintf("http://%s:%s", toxiHost, toxiProxyPort.Port())

	s.logger.Info("configuring Toxiproxy",
		slog.String("api_url", toxiAPIURL),
		slog.String("proxy_url", s.proxyURL),
		slog.String("upstream", s.directURL))

	// Create proxy configuration
	err = s.createToxiProxy(toxiAPIURL, serverHost, serverPort.Port())
	s.Require().NoError(err)

	s.serverURL = s.proxyURL

	s.logger.Info("network resilience test environment ready")
}

func (s *NetworkResilienceSuite) TearDownSuite() {
	ctx := context.Background()

	containers := []testcontainers.Container{
		s.toxiContainer,
		s.serverContainer,
		s.qdrantContainer,
	}

	for _, container := range containers {
		if container != nil {
			if err := container.Terminate(ctx); err != nil {
				s.logger.Error("failed to terminate container", slog.Any("error", err))
			}
		}
	}
}

func (s *NetworkResilienceSuite) createToxiProxy(toxiAPIURL, serverHost, serverPort string) error {
	// Create proxy via Toxiproxy API
	proxyConfig := fmt.Sprintf(`{
		"name": "mcp-server",
		"listen": "0.0.0.0:8475",
		"upstream": "%s:%s",
		"enabled": true
	}`, serverHost, serverPort)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(toxiAPIURL+"/proxies", "application/json", strings.NewReader(proxyConfig))
	if err != nil {
		return fmt.Errorf("failed to create proxy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		return fmt.Errorf("unexpected status creating proxy: %d", resp.StatusCode)
	}

	s.logger.Info("Toxiproxy configured successfully")
	return nil
}

func (s *NetworkResilienceSuite) addToxic(toxicType, direction string, toxicity float32, attributes map[string]interface{}) error {
	toxiHost, _ := s.toxiContainer.Host(context.Background())
	toxiAPIPort, _ := s.toxiContainer.MappedPort(context.Background(), "8474")
	toxiAPIURL := fmt.Sprintf("http://%s:%s", toxiHost, toxiAPIPort.Port())

	toxicData := map[string]interface{}{
		"type":       toxicType,
		"stream":     direction,
		"toxicity":   toxicity,
		"attributes": attributes,
	}

	data, _ := json.Marshal(toxicData)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(toxiAPIURL+"/proxies/mcp-server/toxics", "application/json", strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (s *NetworkResilienceSuite) removeToxic(name string) error {
	toxiHost, _ := s.toxiContainer.Host(context.Background())
	toxiAPIPort, _ := s.toxiContainer.MappedPort(context.Background(), "8474")
	toxiAPIURL := fmt.Sprintf("http://%s:%s", toxiHost, toxiAPIPort.Port())

	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("DELETE", toxiAPIURL+"/proxies/mcp-server/toxics/"+name, nil)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// TestLatencyResilience tests behavior under high latency
func (s *NetworkResilienceSuite) TestLatencyResilience() {
	ctx := context.Background()

	s.logger.Info("testing latency resilience")

	// Create HTTP client through proxy
	httpClient := NewHTTPClient(s.serverURL, "1.0.0", s.logger)

	// Test normal operation first
	createReq := &CreateTaskRequest{
		Content:    "Test task before latency",
		Priority:   "medium",
		Repository: "latency-test",
	}

	start := time.Now()
	task, err := httpClient.CreateTask(ctx, createReq)
	s.Require().NoError(err)
	normalDuration := time.Since(start)

	s.logger.Info("normal operation", slog.Duration("duration", normalDuration))

	// Add latency toxic (500ms + 100ms jitter)
	err = s.addToxic("latency", "downstream", 1.0, map[string]interface{}{
		"latency": 500,
		"jitter":  100,
	})
	s.Require().NoError(err)

	// Test operation with latency
	createReq2 := &CreateTaskRequest{
		Content:    "Test task with latency",
		Priority:   "medium",
		Repository: "latency-test",
	}

	start = time.Now()
	task2, err := httpClient.CreateTask(ctx, createReq2)
	s.Require().NoError(err)
	latencyDuration := time.Since(start)

	s.logger.Info("operation with latency", slog.Duration("duration", latencyDuration))

	// Should still work but take longer
	s.Assert().Greater(latencyDuration, 400*time.Millisecond)
	s.Assert().NotEmpty(task2.ID)

	// Test WebSocket with latency
	wsClient := NewWebSocketClient(
		strings.Replace(s.serverURL, "http://", "ws://", 1),
		"1.0.0",
		NewNotificationHub(s.logger),
		s.logger)

	err = wsClient.Connect(ctx)
	s.Require().NoError(err)
	defer wsClient.Close()

	// WebSocket should still connect despite latency
	s.Assert().True(wsClient.IsConnected())

	// Remove latency toxic
	err = s.removeToxic("latency")
	s.Require().NoError(err)

	s.logger.Info("latency resilience test completed",
		slog.String("task1_id", task.ID),
		slog.String("task2_id", task2.ID))
}

// TestTimeoutResilience tests timeout and retry behavior
func (s *NetworkResilienceSuite) TestTimeoutResilience() {
	ctx := context.Background()

	s.logger.Info("testing timeout resilience")

	// Create HTTP client with retry logic
	httpClient := NewHTTPClient(s.serverURL, "1.0.0", s.logger)

	// Add timeout toxic (drop connections after 100ms)
	err := s.addToxic("timeout", "downstream", 0.5, map[string]interface{}{
		"timeout": 100,
	})
	s.Require().NoError(err)

	// Test operation with intermittent timeouts
	createReq := &CreateTaskRequest{
		Content:    "Test task with timeouts",
		Priority:   "high",
		Repository: "timeout-test",
	}

	// Should retry and eventually succeed due to 50% toxicity
	start := time.Now()
	task, err := httpClient.CreateTask(ctx, createReq)
	duration := time.Since(start)

	s.logger.Info("timeout test result",
		slog.Any("error", err),
		slog.Duration("duration", duration),
		slog.String("task_id", func() string {
			if task != nil {
				return task.ID
			}
			return "nil"
		}()))

	// Should either succeed with retries or fail gracefully
	if err != nil {
		s.logger.Info("operation failed as expected due to timeouts")
		s.Assert().Contains(err.Error(), "timeout", "error should mention timeout")
	} else {
		s.Assert().NotEmpty(task.ID)
		s.Assert().Greater(duration, 100*time.Millisecond, "should take longer due to retries")
		s.logger.Info("operation succeeded despite timeouts")
	}

	// Remove timeout toxic
	err = s.removeToxic("timeout")
	s.Require().NoError(err)

	// Verify normal operation resumes
	createReq2 := &CreateTaskRequest{
		Content:    "Test task after timeout removal",
		Priority:   "medium",
		Repository: "timeout-test",
	}

	task2, err := httpClient.CreateTask(ctx, createReq2)
	s.Require().NoError(err)
	s.Assert().NotEmpty(task2.ID)

	s.logger.Info("timeout resilience test completed")
}

// TestBandwidthLimitation tests behavior under bandwidth constraints
func (s *NetworkResilienceSuite) TestBandwidthLimitation() {
	ctx := context.Background()

	s.logger.Info("testing bandwidth limitation")

	// Create HTTP client
	httpClient := NewHTTPClient(s.serverURL, "1.0.0", s.logger)

	// Add bandwidth limitation (1KB/s)
	err := s.addToxic("bandwidth", "downstream", 1.0, map[string]interface{}{
		"rate": 1024, // 1KB/s
	})
	s.Require().NoError(err)

	// Test small request (should work)
	createReq := &CreateTaskRequest{
		Content:    "Small task",
		Priority:   "medium",
		Repository: "bandwidth-test",
	}

	start := time.Now()
	task, err := httpClient.CreateTask(ctx, createReq)
	smallDuration := time.Since(start)

	s.Require().NoError(err)
	s.Assert().NotEmpty(task.ID)

	s.logger.Info("small request completed", slog.Duration("duration", smallDuration))

	// Test larger request
	largeContent := strings.Repeat("This is a large task description with lots of text. ", 100) // ~5KB

	createReq2 := &CreateTaskRequest{
		Content:    largeContent,
		Priority:   "low",
		Repository: "bandwidth-test",
		Tags:       []string{"large", "bandwidth-test", "performance"},
	}

	start = time.Now()
	task2, err := httpClient.CreateTask(ctx, createReq2)
	largeDuration := time.Since(start)

	if err != nil {
		s.logger.Info("large request failed due to bandwidth limitation", slog.Any("error", err))
		// Acceptable if it times out due to bandwidth
		s.Assert().Contains(err.Error(), "timeout", "should fail due to timeout from bandwidth limit")
	} else {
		s.Assert().NotEmpty(task2.ID)
		s.Assert().Greater(largeDuration, smallDuration, "large request should take longer")
		s.logger.Info("large request completed despite bandwidth limitation", slog.Duration("duration", largeDuration))
	}

	// Remove bandwidth limitation
	err = s.removeToxic("bandwidth")
	s.Require().NoError(err)

	// Verify large request works normally
	task3, err := httpClient.CreateTask(ctx, createReq2)
	s.Require().NoError(err)
	s.Assert().NotEmpty(task3.ID)

	s.logger.Info("bandwidth limitation test completed")
}

// TestConnectionDropAndRecovery tests connection drops and recovery
func (s *NetworkResilienceSuite) TestConnectionDropAndRecovery() {
	ctx := context.Background()

	s.logger.Info("testing connection drop and recovery")

	// Create WebSocket client
	wsClient := NewWebSocketClient(
		strings.Replace(s.serverURL, "http://", "ws://", 1),
		"1.0.0",
		NewNotificationHub(s.logger),
		s.logger)

	// Connect initially
	err := wsClient.Connect(ctx)
	s.Require().NoError(err)
	s.Assert().True(wsClient.IsConnected())

	// Subscribe to repository
	err = wsClient.SubscribeToRepositories([]string{"drop-test"})
	s.Require().NoError(err)

	// Add reset_peer toxic (drops connections)
	err = s.addToxic("reset_peer", "downstream", 1.0, map[string]interface{}{
		"timeout": 0,
	})
	s.Require().NoError(err)

	// Wait for connection to be dropped
	time.Sleep(2 * time.Second)

	// Remove toxic to allow reconnection
	err = s.removeToxic("reset_peer")
	s.Require().NoError(err)

	// Wait for reconnection
	time.Sleep(3 * time.Second)

	// Test that we can still create tasks via HTTP
	httpClient := NewHTTPClient(s.serverURL, "1.0.0", s.logger)

	createReq := &CreateTaskRequest{
		Content:    "Task after connection recovery",
		Priority:   "high",
		Repository: "drop-test",
	}

	task, err := httpClient.CreateTask(ctx, createReq)
	s.Require().NoError(err)
	s.Assert().NotEmpty(task.ID)

	// WebSocket should eventually reconnect and work
	// Give it some time to reconnect
	time.Sleep(5 * time.Second)

	wsClient.Close()

	s.logger.Info("connection drop and recovery test completed", slog.String("task_id", task.ID))
}

// TestConcurrentRequestsUnderStress tests concurrent operations during network stress
func (s *NetworkResilienceSuite) TestConcurrentRequestsUnderStress() {
	ctx := context.Background()

	s.logger.Info("testing concurrent requests under network stress")

	// Add multiple toxics to simulate poor network conditions
	err := s.addToxic("latency", "downstream", 1.0, map[string]interface{}{
		"latency": 200,
		"jitter":  50,
	})
	s.Require().NoError(err)

	err = s.addToxic("bandwidth", "downstream", 1.0, map[string]interface{}{
		"rate": 10240, // 10KB/s
	})
	s.Require().NoError(err)

	// Create multiple clients
	httpClient := NewHTTPClient(s.serverURL, "1.0.0", s.logger)

	// Run concurrent requests
	const numWorkers = 5
	const tasksPerWorker = 3

	var wg sync.WaitGroup
	results := make(chan error, numWorkers*tasksPerWorker)

	start := time.Now()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < tasksPerWorker; j++ {
				createReq := &CreateTaskRequest{
					Content:    fmt.Sprintf("Stress test task %d-%d", workerID, j),
					Priority:   "medium",
					Repository: "stress-test",
				}

				_, err := httpClient.CreateTask(ctx, createReq)
				results <- err
			}
		}(i)
	}

	wg.Wait()
	close(results)

	duration := time.Since(start)

	// Count successes and failures
	var successes, failures int
	for err := range results {
		if err == nil {
			successes++
		} else {
			failures++
		}
	}

	s.logger.Info("concurrent stress test completed",
		slog.Int("successes", successes),
		slog.Int("failures", failures),
		slog.Duration("total_duration", duration))

	// Should have some successes even under stress
	s.Assert().Greater(successes, 0, "should have at least some successful requests")

	// If there are failures, they should be network-related
	if failures > 0 {
		s.logger.Info("some requests failed as expected under network stress")
	}

	// Remove toxics
	s.removeToxic("latency")
	s.removeToxic("bandwidth")

	s.logger.Info("network stress test completed")
}

func TestNetworkResilience(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network resilience tests in short mode")
	}

	if _, err := testcontainers.NewDockerProvider(); err != nil {
		t.Skip("Docker not available, skipping network resilience tests")
	}

	suite.Run(t, new(NetworkResilienceSuite))
}
