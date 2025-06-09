// +build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"lerian-mcp-memory-cli/internal/domain/entities"
)

func TestCLIEndToEnd(t *testing.T) {
	// Skip if not in CI or explicitly requested
	if os.Getenv("RUN_E2E_TESTS") != "true" {
		t.Skip("Skipping E2E tests. Set RUN_E2E_TESTS=true to run")
	}

	// Build CLI binary for testing
	binaryPath := buildTestBinary(t)
	defer os.Remove(binaryPath)

	// Setup test environment
	tempDir := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)

	// Test complete user workflow
	t.Run("CompleteTaskWorkflow", func(t *testing.T) {
		// Add task with high priority
		output := runCLICommand(t, binaryPath, tempDir, "add", "Test e2e task", "--priority=high")
		assert.Contains(t, output, "Test e2e task")
		assert.Contains(t, output, "high")

		// List tasks in JSON format to get ID
		output = runCLICommand(t, binaryPath, tempDir, "list", "--output=json")
		var listResult struct {
			Tasks []entities.Task `json:"tasks"`
			Count int             `json:"count"`
		}
		err := json.Unmarshal([]byte(extractJSON(output)), &listResult)
		require.NoError(t, err)
		require.Equal(t, 1, listResult.Count)
		taskID := listResult.Tasks[0].ID

		// Start task
		output = runCLICommand(t, binaryPath, tempDir, "start", taskID)
		assert.Contains(t, output, "in_progress")

		// Complete task
		output = runCLICommand(t, binaryPath, tempDir, "done", taskID)
		assert.Contains(t, output, "completed")

		// Verify final state
		output = runCLICommand(t, binaryPath, tempDir, "list", "--status=completed", "--output=json")
		err = json.Unmarshal([]byte(extractJSON(output)), &listResult)
		require.NoError(t, err)
		require.Equal(t, 1, listResult.Count)
		assert.Equal(t, entities.StatusCompleted, listResult.Tasks[0].Status)
	})

	t.Run("TaskEditingWorkflow", func(t *testing.T) {
		// Add task
		output := runCLICommand(t, binaryPath, tempDir, "add", "Task to edit")

		// Get task ID
		output = runCLICommand(t, binaryPath, tempDir, "list", "--output=json")
		var listResult struct {
			Tasks []entities.Task `json:"tasks"`
		}
		json.Unmarshal([]byte(extractJSON(output)), &listResult)
		taskID := listResult.Tasks[0].ID

		// Edit task content
		output = runCLICommand(t, binaryPath, tempDir, "edit", taskID, "--content", "Edited task content")
		assert.Contains(t, output, "Edited task content")

		// Update priority
		output = runCLICommand(t, binaryPath, tempDir, "priority", taskID, "low")
		assert.Contains(t, output, "low")

		// Add tags
		output = runCLICommand(t, binaryPath, tempDir, "edit", taskID, "--add-tag", "test", "--add-tag", "e2e")

		// Verify changes
		output = runCLICommand(t, binaryPath, tempDir, "list", "--output=json")
		json.Unmarshal([]byte(extractJSON(output)), &listResult)
		task := listResult.Tasks[0]
		assert.Equal(t, "Edited task content", task.Content)
		assert.Equal(t, entities.PriorityLow, task.Priority)
		assert.Contains(t, task.Tags, "test")
		assert.Contains(t, task.Tags, "e2e")
	})

	t.Run("SearchAndFilterWorkflow", func(t *testing.T) {
		// Add multiple tasks
		runCLICommand(t, binaryPath, tempDir, "add", "Fix authentication bug", "--priority=high", "--tag=bug")
		runCLICommand(t, binaryPath, tempDir, "add", "Add user authentication", "--priority=medium", "--tag=feature")
		runCLICommand(t, binaryPath, tempDir, "add", "Update documentation", "--priority=low", "--tag=docs")

		// Search for "authentication"
		output := runCLICommand(t, binaryPath, tempDir, "search", "authentication", "--output=json")
		var searchResult struct {
			Tasks []entities.Task `json:"tasks"`
			Count int             `json:"count"`
		}
		json.Unmarshal([]byte(extractJSON(output)), &searchResult)
		assert.Equal(t, 2, searchResult.Count)

		// Filter by priority
		output = runCLICommand(t, binaryPath, tempDir, "list", "--priority=high", "--output=json")
		json.Unmarshal([]byte(extractJSON(output)), &searchResult)
		assert.Equal(t, 1, searchResult.Count)

		// Filter by tag
		output = runCLICommand(t, binaryPath, tempDir, "list", "--tag=feature", "--output=json")
		json.Unmarshal([]byte(extractJSON(output)), &searchResult)
		assert.Equal(t, 1, searchResult.Count)
	})

	t.Run("ConfigurationManagement", func(t *testing.T) {
		// Set configuration value
		output := runCLICommand(t, binaryPath, tempDir, "config", "set", "cli.output_format", "plain")
		assert.Contains(t, output, "Configuration updated")

		// Get configuration value
		output = runCLICommand(t, binaryPath, tempDir, "config", "get", "cli.output_format")
		assert.Contains(t, output, "plain")

		// List tasks with new format
		output = runCLICommand(t, binaryPath, tempDir, "list")
		// Should be in plain format, not table
		assert.NotContains(t, output, "│")
		assert.NotContains(t, output, "┌")
	})

	t.Run("StatisticsCommand", func(t *testing.T) {
		// Get statistics
		output := runCLICommand(t, binaryPath, tempDir, "stats", "--output=json")
		var stats map[string]interface{}
		err := json.Unmarshal([]byte(extractJSON(output)), &stats)
		require.NoError(t, err)

		// Verify stats exist
		assert.NotNil(t, stats["TotalTasks"])
		assert.NotNil(t, stats["PendingTasks"])
		assert.NotNil(t, stats["CompletedTasks"])
	})
}

func TestCLIPerformance(t *testing.T) {
	if os.Getenv("RUN_PERF_TESTS") != "true" {
		t.Skip("Skipping performance tests. Set RUN_PERF_TESTS=true to run")
	}

	binaryPath := buildTestBinary(t)
	defer os.Remove(binaryPath)

	tempDir := setupTestEnvironment(t)
	defer os.RemoveAll(tempDir)

	t.Run("BulkTaskCreation", func(t *testing.T) {
		start := time.Now()
		const numTasks = 100

		// Create many tasks
		for i := 0; i < numTasks; i++ {
			content := fmt.Sprintf("Performance test task %d", i)
			runCLICommand(t, binaryPath, tempDir, "add", content)
		}

		duration := time.Since(start)
		avgTime := duration / numTasks
		t.Logf("Created %d tasks in %v (avg: %v per task)", numTasks, duration, avgTime)

		// Should be reasonably fast
		assert.Less(t, avgTime, 100*time.Millisecond)

		// List all tasks
		start = time.Now()
		output := runCLICommand(t, binaryPath, tempDir, "list", "--output=json")
		duration = time.Since(start)
		t.Logf("Listed %d tasks in %v", numTasks, duration)

		var listResult struct {
			Count int `json:"count"`
		}
		json.Unmarshal([]byte(extractJSON(output)), &listResult)
		assert.Equal(t, numTasks, listResult.Count)
	})

	t.Run("SearchPerformance", func(t *testing.T) {
		// Search through all tasks
		start := time.Now()
		output := runCLICommand(t, binaryPath, tempDir, "search", "test", "--output=json")
		duration := time.Since(start)
		t.Logf("Search completed in %v", duration)

		// Should be fast even with many tasks
		assert.Less(t, duration, 500*time.Millisecond)

		var searchResult struct {
			Count int `json:"count"`
		}
		json.Unmarshal([]byte(extractJSON(output)), &searchResult)
		assert.Greater(t, searchResult.Count, 0)
	})
}

func buildTestBinary(t *testing.T) string {
	t.Helper()

	// Create temporary directory for binary
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "lmmc-test")

	// Build command
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/lerian-mcp-memory-cli")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, string(output))
	}

	return binaryPath
}

func setupTestEnvironment(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	err := os.MkdirAll(homeDir, 0755)
	require.NoError(t, err)

	// Create basic config
	configDir := filepath.Join(homeDir, ".lmmc")
	err = os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	configFile := filepath.Join(configDir, "config.yaml")
	configContent := `
server:
  url: ""
  timeout: 5
cli:
  output_format: "table"
  page_size: 20
logging:
  level: "error"
  format: "text"
`
	err = os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	return tempDir
}

func runCLICommand(t *testing.T, binaryPath, tempDir string, args ...string) string {
	t.Helper()

	cmd := exec.Command(binaryPath, args...)
	cmd.Env = append(os.Environ(),
		"HOME="+filepath.Join(tempDir, "home"),
		"NO_COLOR=1", // Disable color output for easier parsing
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Logf("Command failed: %v", err)
		t.Logf("STDOUT: %s", stdout.String())
		t.Logf("STDERR: %s", stderr.String())
		t.FailNow()
	}

	// Combine output (some messages might go to stderr)
	return stdout.String() + stderr.String()
}

func extractJSON(output string) string {
	// Extract JSON from output that might contain log messages
	lines := strings.Split(output, "\n")
	var jsonLines []string
	inJSON := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "{") || strings.HasPrefix(line, "[") {
			inJSON = true
		}
		if inJSON && line != "" {
			jsonLines = append(jsonLines, line)
		}
		if strings.HasSuffix(line, "}") || strings.HasSuffix(line, "]") {
			break
		}
	}

	return strings.Join(jsonLines, "\n")
}