package main

import (
	"os"
	"testing"
)

// Since main() calls log.Fatalf on error, we test the testable parts
func TestMain(t *testing.T) {
	// Test if main can be called (basic smoke test)
	// We can't easily test main() directly due to log.Fatalf, but we can verify imports work

	// Set a valid environment to avoid config errors
	_ = os.Setenv("OPENAI_API_KEY", "test-key")
	_ = os.Setenv("CHROMA_ENDPOINT", "http://localhost:8000")

	// This is a basic test to ensure the package compiles and imports work
	// In a real scenario, you'd refactor main to be more testable
	if testing.Short() {
		t.Skip("Skipping main test in short mode")
	}
}
