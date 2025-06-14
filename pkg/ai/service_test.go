package ai

import (
	"context"
	"testing"
	"time"
)

func TestNewMockService(t *testing.T) {
	service, err := NewMockService(nil)
	if err != nil {
		t.Fatalf("Failed to create mock service: %v", err)
	}

	if service == nil {
		t.Fatal("Service should not be nil")
	}
}

func TestGeneratePRD(t *testing.T) {
	service, err := NewMockService(nil)
	if err != nil {
		t.Fatalf("Failed to create mock service: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	request := PRDRequest{
		UserInputs:  []string{"Create a task management app"},
		ProjectType: "web-app",
		Repository:  "github.com/test/repo",
	}

	response, err := service.GeneratePRD(ctx, &request)
	if err != nil {
		t.Fatalf("Failed to generate PRD: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	if response.Content == "" {
		t.Fatal("Response content should not be empty")
	}

	if response.Metadata == nil {
		t.Fatal("Response metadata should not be nil")
	}
}

func TestGenerateTRD(t *testing.T) {
	service, err := NewMockService(nil)
	if err != nil {
		t.Fatalf("Failed to create mock service: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	request := TRDRequest{
		PRDContent: "Sample PRD content for testing",
		Repository: "github.com/test/repo",
	}

	response, err := service.GenerateTRD(ctx, &request)
	if err != nil {
		t.Fatalf("Failed to generate TRD: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	if response.Content == "" {
		t.Fatal("Response content should not be empty")
	}
}

func TestGenerateMainTasks(t *testing.T) {
	service, err := NewMockService(nil)
	if err != nil {
		t.Fatalf("Failed to create mock service: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	request := TaskRequest{
		Content:    "Sample TRD content for testing",
		TaskType:   "main",
		Repository: "github.com/test/repo",
	}

	response, err := service.GenerateMainTasks(ctx, &request)
	if err != nil {
		t.Fatalf("Failed to generate main tasks: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	if len(response.Tasks) == 0 {
		t.Fatal("Response should contain tasks")
	}
}

func TestAnalyzeComplexity(t *testing.T) {
	service, err := NewMockService(nil)
	if err != nil {
		t.Fatalf("Failed to create mock service: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tests := []struct {
		content       string
		expectedRange []int // [min, max]
	}{
		{"short", []int{3, 3}},
		{"medium length content for testing complexity analysis", []int{3, 3}},
	}

	for _, tt := range tests {
		complexity, err := service.AnalyzeComplexity(ctx, tt.content)
		if err != nil {
			t.Fatalf("Failed to analyze complexity: %v", err)
		}

		if complexity < tt.expectedRange[0] || complexity > tt.expectedRange[1] {
			t.Errorf("Expected complexity in range %v, got %d", tt.expectedRange, complexity)
		}
	}
}

func TestStartInteractiveSession(t *testing.T) {
	service, err := NewMockService(nil)
	if err != nil {
		t.Fatalf("Failed to create mock service: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := service.StartInteractiveSession(ctx, "prd")
	if err != nil {
		t.Fatalf("Failed to start interactive session: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	if response.SessionID == "" {
		t.Fatal("Session ID should not be empty")
	}

	if response.IsComplete {
		t.Fatal("Session should not be complete initially")
	}
}
