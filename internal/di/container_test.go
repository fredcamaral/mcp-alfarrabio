package di

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"lerian-mcp-memory/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewContainer(t *testing.T) {
	tests := []struct {
		name          string
		setupEnv      func()
		cleanupEnv    func()
		config        *config.Config
		expectedError bool
		validate      func(*testing.T, *Container)
	}{
		{
			name: "successful_container_creation",
			setupEnv: func() {
				_ = os.Setenv("OPENAI_API_KEY", "test-key") // Test environment setup
			},
			cleanupEnv: func() {
				_ = os.Unsetenv("OPENAI_API_KEY") // Test cleanup
			},
			config:        config.DefaultConfig(),
			expectedError: false,
			validate: func(t *testing.T, c *Container) {
				// Verify main services are initialized (some may be nil if not properly set up)
				assert.NotNil(t, c.GetVectorStore())
				assert.NotNil(t, c.GetEmbeddingService())
				assert.NotNil(t, c.GetChunkingService())
				assert.NotNil(t, c.GetBackupManager())
				assert.NotNil(t, c.GetLearningEngine())
				assert.NotNil(t, c.GetMultiRepoEngine())
				assert.NotNil(t, c.GetThreadStore())
				assert.NotNil(t, c.GetThreadManager())
				assert.NotNil(t, c.GetPatternAnalyzer())
				assert.NotNil(t, c.GetContextSuggester())
				assert.NotNil(t, c.GetMemoryAnalytics())
				// Note: AuditLogger may be nil if directory creation fails
				// Note: ChainBuilder and ChainStore may be nil (not initialized in current code)
				assert.NotNil(t, c.GetRelationshipManager())
			},
		},
		{
			name: "container_with_circuit_breaker_enabled",
			setupEnv: func() {
				_ = os.Setenv("OPENAI_API_KEY", "test-key")  // Test environment setup
				_ = os.Setenv("USE_CIRCUIT_BREAKER", "true") // Test environment setup
			},
			cleanupEnv: func() {
				_ = os.Unsetenv("OPENAI_API_KEY")      // Test cleanup
				_ = os.Unsetenv("USE_CIRCUIT_BREAKER") // Test cleanup
			},
			config:        config.DefaultConfig(),
			expectedError: false,
			validate: func(t *testing.T, c *Container) {
				// Circuit breaker should be enabled
				assert.NotNil(t, c.GetVectorStore())
				// Additional validation that circuit breaker wrapper is applied would require
				// examining the underlying implementation
			},
		},
		{
			name: "container_with_custom_backup_directory",
			setupEnv: func() {
				_ = os.Setenv("OPENAI_API_KEY", "test-key") // Test environment setup
				// Create a temporary directory for testing
				tmpDir := filepath.Join(os.TempDir(), "mcp-memory-test-backup")
				_ = os.MkdirAll(tmpDir, 0o750)                       // Test directory creation - safe to ignore error
				_ = os.Setenv("MCP_MEMORY_BACKUP_DIRECTORY", tmpDir) // Test environment setup
			},
			cleanupEnv: func() {
				_ = os.Unsetenv("OPENAI_API_KEY") // Test cleanup
				tmpDir := os.Getenv("MCP_MEMORY_BACKUP_DIRECTORY")
				_ = os.Unsetenv("MCP_MEMORY_BACKUP_DIRECTORY") // Test cleanup
				if tmpDir != "" {
					_ = os.RemoveAll(tmpDir) // Test cleanup - safe to ignore error
				}
			},
			config:        config.DefaultConfig(),
			expectedError: false,
			validate: func(t *testing.T, c *Container) {
				assert.NotNil(t, c.GetBackupManager())
			},
		},
		{
			name: "container_with_custom_audit_directory",
			setupEnv: func() {
				_ = os.Setenv("OPENAI_API_KEY", "test-key") // Test environment setup
				tmpDir := filepath.Join(os.TempDir(), "mcp-memory-test-audit")
				_ = os.MkdirAll(tmpDir, 0o750)                      // Test directory creation - safe to ignore error
				_ = os.Setenv("MCP_MEMORY_AUDIT_DIRECTORY", tmpDir) // Test environment setup
			},
			cleanupEnv: func() {
				_ = os.Unsetenv("OPENAI_API_KEY") // Test cleanup
				tmpDir := os.Getenv("MCP_MEMORY_AUDIT_DIRECTORY")
				_ = os.Unsetenv("MCP_MEMORY_AUDIT_DIRECTORY") // Test cleanup
				if tmpDir != "" {
					_ = os.RemoveAll(tmpDir) // Test cleanup - safe to ignore error
				}
			},
			config:        config.DefaultConfig(),
			expectedError: false,
			validate: func(t *testing.T, c *Container) {
				assert.NotNil(t, c.GetAuditLogger())
			},
		},
		{
			name: "container_without_openai_key",
			setupEnv: func() {
				// Don't set OPENAI_API_KEY - container should still create but services may be limited
			},
			cleanupEnv: func() {
				// Nothing to clean
			},
			config:        config.DefaultConfig(),
			expectedError: false,
			validate: func(t *testing.T, c *Container) {
				// Container should create but some services might have limited functionality
				assert.NotNil(t, c.GetVectorStore())
				assert.NotNil(t, c.GetEmbeddingService()) // May work but fail on actual calls
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.setupEnv != nil {
				tt.setupEnv()
			}
			defer func() {
				if tt.cleanupEnv != nil {
					tt.cleanupEnv()
				}
			}()

			// Execute
			container, err := NewContainer(tt.config)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, container)
			} else {
				require.NoError(t, err)
				require.NotNil(t, container)

				if tt.validate != nil {
					tt.validate(t, container)
				}

				// Cleanup
				if container != nil {
					_ = container.Shutdown() // Test cleanup - safe to ignore error
				}
			}
		})
	}
}

// TestContainerHealthCheck is disabled for now due to external service dependencies
// TODO: Mock external services to enable health check testing
func TestContainerHealthCheck_Disabled(t *testing.T) {
	t.Skip("Health check test disabled - requires external services (Qdrant)")
}

func TestContainerShutdown(t *testing.T) {
	// Setup
	_ = os.Setenv("OPENAI_API_KEY", "test-key")          // Test environment setup
	defer func() { _ = os.Unsetenv("OPENAI_API_KEY") }() // Test cleanup

	cfg := config.DefaultConfig()
	container, err := NewContainer(cfg)
	require.NoError(t, err)
	require.NotNil(t, container)

	// Execute shutdown (without health check due to external service dependencies)
	err = container.Shutdown()
	assert.NoError(t, err)

	// Verify services are still accessible (getters should not fail)
	// but their state may have changed
	assert.NotNil(t, container.GetVectorStore())
	assert.NotNil(t, container.GetEmbeddingService())
	assert.NotNil(t, container.GetMemoryAnalytics())
	// Note: AuditLogger may be nil depending on initialization
}

func TestContainerGetters(t *testing.T) {
	// Setup
	_ = os.Setenv("OPENAI_API_KEY", "test-key")          // Test environment setup
	defer func() { _ = os.Unsetenv("OPENAI_API_KEY") }() // Test cleanup

	cfg := config.DefaultConfig()
	container, err := NewContainer(cfg)
	require.NoError(t, err)
	require.NotNil(t, container)
	defer func() { _ = container.Shutdown() }() // Test cleanup

	// Test getter methods - some may return nil if not properly initialized
	tests := []struct {
		name      string
		getter    func() interface{}
		expectNil bool
	}{
		{"GetVectorStore", func() interface{} { return container.GetVectorStore() }, false},
		{"GetEmbeddingService", func() interface{} { return container.GetEmbeddingService() }, false},
		{"GetChunkingService", func() interface{} { return container.GetChunkingService() }, false},
		{"GetBackupManager", func() interface{} { return container.GetBackupManager() }, false},
		{"GetLearningEngine", func() interface{} { return container.GetLearningEngine() }, false},
		{"GetMultiRepoEngine", func() interface{} { return container.GetMultiRepoEngine() }, false},
		{"GetThreadStore", func() interface{} { return container.GetThreadStore() }, false},
		{"GetThreadManager", func() interface{} { return container.GetThreadManager() }, false},
		{"GetChainBuilder", func() interface{} { return container.GetChainBuilder() }, true}, // Not initialized
		{"GetChainStore", func() interface{} { return container.GetChainStore() }, true},     // Not initialized
		{"GetPatternAnalyzer", func() interface{} { return container.GetPatternAnalyzer() }, false},
		{"GetContextSuggester", func() interface{} { return container.GetContextSuggester() }, false},
		{"GetRelationshipManager", func() interface{} { return container.GetRelationshipManager() }, false},
		{"GetMemoryAnalytics", func() interface{} { return container.GetMemoryAnalytics() }, false},
		{"GetAuditLogger", func() interface{} { return container.GetAuditLogger() }, true}, // May be nil if dir creation fails
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.getter()
			if tt.expectNil {
				// Some services may not be initialized - that's OK for now
				t.Logf("Getter %s returned %v (nil expected)", tt.name, result == nil)
			} else {
				assert.NotNil(t, result, "Getter %s should return non-nil value", tt.name)
			}
		})
	}
}

func TestContainerDependencyWiring(t *testing.T) {
	// Setup
	_ = os.Setenv("OPENAI_API_KEY", "test-key")          // Test environment setup
	defer func() { _ = os.Unsetenv("OPENAI_API_KEY") }() // Test cleanup

	cfg := config.DefaultConfig()
	container, err := NewContainer(cfg)
	require.NoError(t, err)
	require.NotNil(t, container)
	defer func() { _ = container.Shutdown() }() // Test cleanup

	// Test that services have proper dependencies wired
	t.Run("services_have_dependencies", func(t *testing.T) {
		// These are integration tests to verify that dependencies are properly injected
		// The exact nature of dependencies would depend on the internal implementation

		// Verify that embedding service is available for chunking service
		chunkingService := container.GetChunkingService()
		assert.NotNil(t, chunkingService)

		// Verify that learning engine is available
		learningEngine := container.GetLearningEngine()
		assert.NotNil(t, learningEngine)

		// Verify thread manager is available
		threadManager := container.GetThreadManager()
		assert.NotNil(t, threadManager)

		// Note: chainBuilder is currently nil (not initialized in container)

		// Verify context suggester has complex dependencies
		contextSuggester := container.GetContextSuggester()
		assert.NotNil(t, contextSuggester)
	})
}

func TestContainerConfigVariations(t *testing.T) {
	_ = os.Setenv("OPENAI_API_KEY", "test-key")          // Test environment setup
	defer func() { _ = os.Unsetenv("OPENAI_API_KEY") }() // Test cleanup

	tests := []struct {
		name           string
		configModifier func(*config.Config)
		expectedError  bool
	}{
		{
			name: "default_config",
			configModifier: func(cfg *config.Config) {
				// No modifications
			},
			expectedError: false,
		},
		{
			name: "config_with_custom_qdrant_host",
			configModifier: func(cfg *config.Config) {
				cfg.Qdrant.Host = "custom-host"
				cfg.Qdrant.Port = 9999
			},
			expectedError: false,
		},
		{
			name: "config_with_custom_openai_settings",
			configModifier: func(cfg *config.Config) {
				cfg.OpenAI.EmbeddingModel = "text-embedding-3-small"
				cfg.OpenAI.MaxTokens = 2048
			},
			expectedError: false,
		},
		{
			name: "config_with_storage_settings",
			configModifier: func(cfg *config.Config) {
				cfg.Storage.Provider = "qdrant"
				cfg.Storage.RetentionDays = 30
				cfg.Storage.BackupEnabled = true
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			tt.configModifier(cfg)

			container, err := NewContainer(cfg)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, container)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, container)

				if container != nil {
					// Verify basic functionality
					assert.NotNil(t, container.GetVectorStore())
					assert.NotNil(t, container.GetEmbeddingService())

					_ = container.Shutdown() // Test cleanup - safe to ignore error
				}
			}
		})
	}
}

func TestEnvironmentVariableHandling(t *testing.T) {
	tests := []struct {
		name          string
		envVars       map[string]string
		unsetVars     []string
		expectedError bool
	}{
		{
			name: "circuit_breaker_enabled",
			envVars: map[string]string{
				"OPENAI_API_KEY":      "test-key",
				"USE_CIRCUIT_BREAKER": "true",
			},
			expectedError: false,
		},
		{
			name: "circuit_breaker_disabled",
			envVars: map[string]string{
				"OPENAI_API_KEY":      "test-key",
				"USE_CIRCUIT_BREAKER": "false",
			},
			expectedError: false,
		},
		{
			name: "custom_directories",
			envVars: map[string]string{
				"OPENAI_API_KEY":              "test-key",
				"MCP_MEMORY_BACKUP_DIRECTORY": "/tmp/test-backup",
				"MCP_MEMORY_AUDIT_DIRECTORY":  "/tmp/test-audit",
			},
			expectedError: false,
		},
		{
			name:    "missing_openai_api_key",
			envVars: map[string]string{
				// No OPENAI_API_KEY - container should still create
			},
			unsetVars:     []string{"OPENAI_API_KEY"},
			expectedError: false, // Container creation succeeds, but services may be limited
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment
			originalValues := make(map[string]string)
			for key, value := range tt.envVars {
				if originalValue, exists := os.LookupEnv(key); exists {
					originalValues[key] = originalValue
				}
				_ = os.Setenv(key, value) // Test environment setup
			}

			for _, key := range tt.unsetVars {
				if originalValue, exists := os.LookupEnv(key); exists {
					originalValues[key] = originalValue
				}
				_ = os.Unsetenv(key) // Test cleanup
			}

			// Cleanup function
			defer func() {
				for key := range tt.envVars {
					if originalValue, exists := originalValues[key]; exists {
						_ = os.Setenv(key, originalValue) // Test cleanup
					} else {
						_ = os.Unsetenv(key) // Test cleanup
					}
				}
				for _, key := range tt.unsetVars {
					if originalValue, exists := originalValues[key]; exists {
						_ = os.Setenv(key, originalValue) // Test cleanup
					}
				}
			}()

			// Create directories if needed
			if backupDir := os.Getenv("MCP_MEMORY_BACKUP_DIRECTORY"); backupDir != "" {
				_ = os.MkdirAll(backupDir, 0o750)              // Test directory creation - safe to ignore error
				defer func() { _ = os.RemoveAll(backupDir) }() // Test cleanup
			}
			if auditDir := os.Getenv("MCP_MEMORY_AUDIT_DIRECTORY"); auditDir != "" {
				_ = os.MkdirAll(auditDir, 0o750)              // Test directory creation - safe to ignore error
				defer func() { _ = os.RemoveAll(auditDir) }() // Test cleanup
			}

			// Execute
			cfg := config.DefaultConfig()
			container, err := NewContainer(cfg)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, container)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, container)

				if container != nil {
					_ = container.Shutdown() // Test cleanup - safe to ignore error
				}
			}
		})
	}
}

// Benchmark tests for container creation performance
func BenchmarkNewContainer(b *testing.B) {
	_ = os.Setenv("OPENAI_API_KEY", "test-key")          // Test environment setup
	defer func() { _ = os.Unsetenv("OPENAI_API_KEY") }() // Test cleanup

	cfg := config.DefaultConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		container, err := NewContainer(cfg)
		if err != nil {
			b.Fatal(err)
		}
		_ = container.Shutdown() // Test cleanup - safe to ignore error
	}
}

func BenchmarkContainerHealthCheck(b *testing.B) {
	_ = os.Setenv("OPENAI_API_KEY", "test-key")          // Test environment setup
	defer func() { _ = os.Unsetenv("OPENAI_API_KEY") }() // Test cleanup

	cfg := config.DefaultConfig()
	container, err := NewContainer(cfg)
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = container.Shutdown() }() // Test cleanup

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := container.HealthCheck(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}
