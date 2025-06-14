// Package testing provides test configuration and utilities for integration testing
package testing

import (
	"os"
	"strconv"
	"time"

	"lerian-mcp-memory/internal/config"
)

const (
	localhostHost = "localhost"
	trueString    = "true"
)

// IntegrationTestConfig provides configuration specific to integration testing
type IntegrationTestConfig struct {
	*config.Config
	TestDatabaseURL   string
	TestQdrantURL     string
	TestOpenAIAPIKey  string
	SkipRealAI        bool
	TestTimeout       time.Duration
	CleanupAfterTests bool
	VerboseLogging    bool
}

// LoadTestConfig loads configuration optimized for integration testing
func LoadTestConfig() (*IntegrationTestConfig, error) {
	// Load base configuration
	baseConfig, err := config.LoadConfig()
	if err != nil {
		return nil, err
	}

	testConfig := &IntegrationTestConfig{
		Config:            baseConfig,
		TestDatabaseURL:   getEnv("TEST_DATABASE_URL", ""),
		TestQdrantURL:     getEnv("TEST_QDRANT_URL", "http://localhost:6333"),
		TestOpenAIAPIKey:  getEnv("TEST_OPENAI_API_KEY", ""),
		SkipRealAI:        getBoolEnv("SKIP_REAL_AI", true),
		TestTimeout:       getDurationEnv("TEST_TIMEOUT", 5*time.Minute),
		CleanupAfterTests: getBoolEnv("CLEANUP_AFTER_TESTS", true),
		VerboseLogging:    getBoolEnv("VERBOSE_TEST_LOGGING", false),
	}

	// Override base config with test-specific settings
	if testConfig.TestDatabaseURL != "" {
		// Override database configuration for test database
		testConfig.Database.Host = localhostHost
		testConfig.Database.Port = 5432
		testConfig.Database.Name = "lerian_mcp_test"
		testConfig.Database.User = "test_user"
		testConfig.Database.Password = "test_password"
	}

	// Override Qdrant settings for testing
	testConfig.Qdrant.Host = localhostHost
	testConfig.Qdrant.Port = 6333
	testConfig.Qdrant.Collection = "lerian_mcp_test_collection"

	// Override OpenAI settings for testing
	if testConfig.TestOpenAIAPIKey != "" {
		testConfig.OpenAI.APIKey = testConfig.TestOpenAIAPIKey
	}

	// Reduce rate limits for testing
	testConfig.OpenAI.RateLimitRPM = 10

	// Override server settings for testing
	testConfig.Server.Host = localhostHost
	testConfig.Server.Port = 9081 // Different port to avoid conflicts

	// Set log level based on verbose setting
	if testConfig.VerboseLogging {
		testConfig.Logging.Level = "debug"
	} else {
		testConfig.Logging.Level = "info"
	}

	return testConfig, nil
}

// IsRealStorageAvailable checks if real storage systems are available for testing
func (tc *IntegrationTestConfig) IsRealStorageAvailable() bool {
	// Check if we have real database and Qdrant available
	return tc.TestDatabaseURL != "" || (tc.Database.Host != "" && tc.Qdrant.Host != "")
}

// ShouldSkipAI determines if AI-related tests should be skipped
func (tc *IntegrationTestConfig) ShouldSkipAI() bool {
	return tc.SkipRealAI || tc.TestOpenAIAPIKey == ""
}

// Helper functions for environment variable parsing

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// TestEnvironment provides test environment detection and setup
type TestEnvironment struct {
	IsCI              bool
	HasRealStorage    bool
	HasRealAI         bool
	CanRunIntegration bool
}

// DetectTestEnvironment analyzes the current environment for testing capabilities
func DetectTestEnvironment() *TestEnvironment {
	return &TestEnvironment{
		IsCI:              os.Getenv("CI") == trueString,
		HasRealStorage:    os.Getenv("TEST_DATABASE_URL") != "" || os.Getenv("QDRANT_URL") != "",
		HasRealAI:         os.Getenv("TEST_OPENAI_API_KEY") != "" || os.Getenv("OPENAI_API_KEY") != "",
		CanRunIntegration: os.Getenv("RUN_INTEGRATION_TESTS") == trueString,
	}
}

// ShouldRunIntegrationTests determines if integration tests should run in current environment
func (te *TestEnvironment) ShouldRunIntegrationTests() bool {
	return te.CanRunIntegration && (te.HasRealStorage || te.IsCI)
}

// GetTestingRecommendations provides recommendations for test setup
func (te *TestEnvironment) GetTestingRecommendations() []string {
	var recommendations []string

	if !te.HasRealStorage {
		recommendations = append(recommendations,
			"Set TEST_DATABASE_URL or ensure database/Qdrant are running for full integration testing")
	}

	if !te.HasRealAI {
		recommendations = append(recommendations,
			"Set TEST_OPENAI_API_KEY for AI-powered integration tests (or use SKIP_REAL_AI="+trueString+")")
	}

	if !te.CanRunIntegration {
		recommendations = append(recommendations,
			"Set RUN_INTEGRATION_TESTS="+trueString+" to enable integration test execution")
	}

	return recommendations
}
