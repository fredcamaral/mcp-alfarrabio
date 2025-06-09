package integration

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/testcontainers/testcontainers-go"
)

// TestMain sets up and tears down the test environment
func TestMain(m *testing.M) {
	// Check if Docker is available
	if _, err := testcontainers.NewDockerProvider(); err != nil {
		fmt.Printf("Docker not available, skipping integration tests: %v\n", err)
		os.Exit(0)
	}

	// Set up logging for tests
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Ensure Docker containers can be cleaned up
	ctx := context.Background()

	// Pre-pull required Docker images to speed up tests
	logger.Info("pre-pulling Docker images for integration tests")

	images := []string{
		"qdrant/qdrant:latest",
		"ghcr.io/lerianstudio/lerian-mcp-memory:latest",
		"ghcr.io/shopify/toxiproxy:2.5.0",
	}

	provider, err := testcontainers.NewDockerProvider()
	if err != nil {
		fmt.Printf("Failed to create Docker provider: %v\n", err)
		os.Exit(1)
	}

	for _, image := range images {
		logger.Info("pulling image", slog.String("image", image))

		_, err := provider.CreateContainer(ctx, testcontainers.ContainerRequest{
			Image: image,
		})
		if err != nil {
			logger.Warn("failed to pull image",
				slog.String("image", image),
				slog.Any("error", err))
			// Continue with other images - some might be available locally
		}
	}

	logger.Info("starting integration tests")

	// Run tests
	code := m.Run()

	// Cleanup any remaining containers
	logger.Info("cleaning up test environment")
	cleanupContainers()

	os.Exit(code)
}

func cleanupContainers() {
	// Cleanup is handled automatically by testcontainers during test teardown
	// Individual test suites manage their own container lifecycle
	fmt.Println("Container cleanup is handled by individual test suites")
}

// Helper function for pointer conversion
func stringPtr(s string) *string {
	return &s
}

// Additional helper functions can be added here as needed

func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}
