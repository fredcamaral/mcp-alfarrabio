//go:build integration
// +build integration

package internal_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"lerian-mcp-memory/internal/ai"
	"lerian-mcp-memory/internal/config"
	"lerian-mcp-memory/internal/documents"
	"lerian-mcp-memory/internal/logging"
	"lerian-mcp-memory/internal/repl"
)

// TestDocumentGenerationFlow tests the complete document generation workflow
func TestDocumentGenerationFlow(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	ctx := context.Background()
	logger := logging.NewLogger(logging.DEBUG)

	// Create test configuration
	cfg := &config.Config{
		AI: config.AIConfig{
			// Empty API keys will trigger mock mode
			Claude: config.ClaudeClientConfig{
				APIKey: "",
			},
			Perplexity: config.PerplexityClientConfig{
				APIKey: "",
			},
			OpenAI: config.OpenAIClientConfig{
				APIKey: "",
			},
		},
	}

	// Initialize components
	aiService, _ := ai.NewService(cfg, logger)
	ruleManager := documents.NewRuleManager(logger)
	docGen := ai.NewDocumentGenerator(aiService, ruleManager)
	taskGen := documents.NewTaskGenerator(logger)
	processor := documents.NewProcessor(logger)

	// Test PRD creation
	t.Run("PRD Creation", func(t *testing.T) {
		req := &ai.DocumentGenerationRequest{
			Type:       ai.DocumentTypePRD,
			Input:      "AI-powered task management system",
			Repository: "test-repo",
			Context: map[string]string{
				"project_type": "web-app",
			},
		}

		resp, err := docGen.GenerateDocument(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp.Document)

		prd, ok := resp.Document.(*documents.PRDEntity)
		require.True(t, ok)
		assert.NotEmpty(t, prd.Title)
		assert.NotEmpty(t, prd.Content)
		assert.Greater(t, prd.ComplexityScore, 0)
	})

	// Test TRD generation from PRD
	t.Run("TRD Generation", func(t *testing.T) {
		// First create a PRD
		prdReq := &ai.DocumentGenerationRequest{
			Type:       ai.DocumentTypePRD,
			Input:      "E-commerce platform with payment processing",
			Repository: "test-repo",
		}

		prdResp, err := docGen.GenerateDocument(ctx, prdReq)
		require.NoError(t, err)

		prd := prdResp.Document.(*documents.PRDEntity)

		// Generate TRD from PRD
		trdReq := &ai.DocumentGenerationRequest{
			Type:       ai.DocumentTypeTRD,
			Repository: "test-repo",
			SourcePRD:  prd,
		}

		trdResp, err := docGen.GenerateDocument(ctx, trdReq)
		require.NoError(t, err)
		assert.NotNil(t, trdResp.Document)

		trd, ok := trdResp.Document.(*documents.TRDEntity)
		require.True(t, ok)
		assert.NotEmpty(t, trd.Title)
		assert.NotEmpty(t, trd.Architecture)
		assert.NotEmpty(t, trd.TechnicalStack)
	})

	// Test task generation
	t.Run("Task Generation", func(t *testing.T) {
		// Create PRD and TRD
		prd := &documents.PRDEntity{
			Title:             "Test Project",
			Content:           "A test project for task generation",
			ComplexityScore:   50,
			EstimatedDuration: "3 months",
			ParsedContent: documents.ParsedPRDContent{
				ProjectName:  "Test Project",
				Goals:        []string{"Goal 1", "Goal 2"},
				Requirements: []string{"Req 1", "Req 2"},
			},
		}

		trd := &documents.TRDEntity{
			Title:          "Test Project TRD",
			Architecture:   "Microservices",
			TechnicalStack: []string{"Go", "PostgreSQL", "Docker"},
		}

		// Generate main tasks
		mainTasks, err := taskGen.GenerateMainTasks(prd, trd)
		require.NoError(t, err)
		assert.NotEmpty(t, mainTasks)

		// Verify task structure
		for _, task := range mainTasks {
			assert.NotEmpty(t, task.TaskID)
			assert.NotEmpty(t, task.Name)
			assert.NotEmpty(t, task.Phase)
			assert.Greater(t, task.ComplexityScore, 0)
		}

		// Generate sub-tasks for first main task
		if len(mainTasks) > 0 {
			subTasks, err := taskGen.GenerateSubTasks(mainTasks[0], prd, trd)
			require.NoError(t, err)
			assert.NotEmpty(t, subTasks)

			for _, subTask := range subTasks {
				assert.NotEmpty(t, subTask.SubTaskID)
				assert.NotEmpty(t, subTask.Name)
				assert.Greater(t, subTask.EstimatedHours, 0)
			}
		}
	})

	// Test document processing
	t.Run("Document Processing", func(t *testing.T) {
		// Create test PRD file
		tempDir := t.TempDir()
		prdFile := filepath.Join(tempDir, "test_prd.md")

		prdContent := `# Test Project PRD

## Overview
This is a test project for integration testing.

## Goals
- Goal 1: Test document processing
- Goal 2: Validate workflow

## Requirements
### Functional Requirements
- FR1: User authentication
- FR2: Data persistence

### Non-Functional Requirements
- NFR1: Performance
- NFR2: Security
`

		err := os.WriteFile(prdFile, []byte(prdContent), 0644)
		require.NoError(t, err)

		// Process PRD file
		prd, err := processor.ProcessPRDFile(prdFile, "test-repo")
		require.NoError(t, err)
		assert.NotNil(t, prd)
		assert.Equal(t, "Test Project PRD", prd.Title)
		assert.Contains(t, prd.ParsedContent.Goals, "Test document processing")

		// Validate PRD
		err = processor.ValidatePRD(prd)
		assert.NoError(t, err)

		// Export PRD
		exportFile := filepath.Join(tempDir, "exported_prd.json")
		file, err := os.Create(exportFile)
		require.NoError(t, err)
		defer file.Close()

		err = processor.ExportPRD(prd, "json", file)
		assert.NoError(t, err)

		// Verify export exists
		_, err = os.Stat(exportFile)
		assert.NoError(t, err)
	})

	// Test rule management
	t.Run("Rule Management", func(t *testing.T) {
		// List rules
		rules := ruleManager.ListRules()
		assert.NotEmpty(t, rules)

		// Get specific rule type
		prdRules := ruleManager.GetRulesByType(documents.RuleTypePRD)
		assert.NotEmpty(t, prdRules)

		// Get active rules
		activeRules := ruleManager.GetActiveRules()
		assert.NotEmpty(t, activeRules)

		// Load custom rule
		customRule := &documents.Rule{
			Name:        "custom-test-rule",
			Type:        documents.RuleTypePRD,
			Description: "Custom test rule",
			Content:     "Test rule content",
			Priority:    100,
			Active:      true,
		}

		err := ruleManager.LoadCustomRule(customRule)
		assert.NoError(t, err)

		// Verify custom rule is loaded
		rule, err := ruleManager.GetRuleByName("custom-test-rule")
		assert.NoError(t, err)
		assert.Equal(t, customRule.Name, rule.Name)
	})

	// Test complexity analysis
	t.Run("Complexity Analysis", func(t *testing.T) {
		prd := &documents.PRDEntity{
			Title:           "Complex Project",
			ComplexityScore: 75,
			ParsedContent: documents.ParsedPRDContent{
				Requirements: []string{
					"User authentication",
					"Payment processing",
					"Real-time notifications",
					"Machine learning",
				},
			},
		}

		trd := &documents.TRDEntity{
			Architecture:   "Microservices",
			TechnicalStack: []string{"Go", "Python", "TensorFlow", "Kubernetes"},
		}

		analyzer := documents.NewComplexityAnalyzer()
		analysis := analyzer.AnalyzeProject(prd, trd)

		assert.Greater(t, analysis.TotalComplexity, 50)
		assert.NotEmpty(t, analysis.CoreFeatures)
		assert.True(t, analysis.RequiresIntegration)
	})
}

// TestREPLIntegration tests the REPL functionality
func TestREPLIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	logger := logging.NewLogger(logging.DEBUG)
	cfg := &config.Config{
		AI: config.AIConfig{
			// Empty API keys will trigger mock mode
			Claude: config.ClaudeClientConfig{
				APIKey: "",
			},
			Perplexity: config.PerplexityClientConfig{
				APIKey: "",
			},
			OpenAI: config.OpenAIClientConfig{
				APIKey: "",
			},
		},
	}

	aiService, _ := ai.NewService(cfg, logger)
	ruleManager := documents.NewRuleManager(logger)
	docGen := ai.NewDocumentGenerator(aiService, ruleManager)
	taskGen := documents.NewTaskGenerator(logger)
	processor := documents.NewProcessor(logger)

	// Create REPL instance
	replInstance := repl.NewREPL(
		docGen,
		taskGen,
		processor,
		ruleManager,
		logger,
		"test-repo",
	)

	// Test session export/import
	t.Run("Session Management", func(t *testing.T) {
		tempDir := t.TempDir()
		sessionFile := filepath.Join(tempDir, "test_session.json")

		// Export session
		file, err := os.Create(sessionFile)
		require.NoError(t, err)

		err = replInstance.ExportSession(file)
		file.Close()
		assert.NoError(t, err)

		// Verify file exists
		_, err = os.Stat(sessionFile)
		assert.NoError(t, err)
	})

	// Test document export
	t.Run("Document Export", func(t *testing.T) {
		tempDir := t.TempDir()

		err := replInstance.ExportDocuments(tempDir)
		assert.NoError(t, err)

		// Check if directory was created
		_, err = os.Stat(tempDir)
		assert.NoError(t, err)
	})
}

// TestEndToEndWorkflow tests the complete workflow from PRD to tasks
func TestEndToEndWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	logger := logging.NewLogger(logging.DEBUG)

	// Create configuration
	cfg := &config.Config{
		AI: config.AIConfig{
			// Empty API keys will trigger mock mode
			Claude: config.ClaudeClientConfig{
				APIKey: "",
			},
			Perplexity: config.PerplexityClientConfig{
				APIKey: "",
			},
			OpenAI: config.OpenAIClientConfig{
				APIKey: "",
			},
		},
	}

	// Initialize all components
	aiService, _ := ai.NewService(cfg, logger)
	ruleManager := documents.NewRuleManager(logger)
	docGen := ai.NewDocumentGenerator(aiService, ruleManager)
	taskGen := documents.NewTaskGenerator(logger)
	processor := documents.NewProcessor(logger)

	// Step 1: Create PRD
	prdReq := &ai.DocumentGenerationRequest{
		Type:       ai.DocumentTypePRD,
		Input:      "Social media analytics platform",
		Repository: "test-analytics",
		Context: map[string]string{
			"project_type": "saas",
			"target_users": "businesses",
		},
	}

	prdResp, err := docGen.GenerateDocument(ctx, prdReq)
	require.NoError(t, err)

	prd := prdResp.Document.(*documents.PRDEntity)
	assert.NotEmpty(t, prd.Title)

	// Step 2: Generate TRD from PRD
	trdReq := &ai.DocumentGenerationRequest{
		Type:       ai.DocumentTypeTRD,
		Repository: "test-analytics",
		SourcePRD:  prd,
		Context: map[string]string{
			"architecture_preference": "microservices",
			"deployment_target":       "kubernetes",
		},
	}

	trdResp, err := docGen.GenerateDocument(ctx, trdReq)
	require.NoError(t, err)

	trd := trdResp.Document.(*documents.TRDEntity)
	assert.NotEmpty(t, trd.Architecture)

	// Step 3: Generate main tasks
	mainTasks, err := taskGen.GenerateMainTasks(prd, trd)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(mainTasks), 3) // Should have at least setup, development, deployment phases

	// Step 4: Generate sub-tasks for each main task
	totalSubTasks := 0
	totalHours := 0

	for _, mainTask := range mainTasks {
		subTasks, err := taskGen.GenerateSubTasks(mainTask, prd, trd)
		require.NoError(t, err)

		totalSubTasks += len(subTasks)
		for _, subTask := range subTasks {
			totalHours += subTask.EstimatedHours
		}
	}

	assert.Greater(t, totalSubTasks, 10) // Should have substantial sub-tasks
	assert.Greater(t, totalHours, 100)   // Non-trivial project

	// Step 5: Generate timeline and verify it's reasonable
	timeline := documents.EstimateProjectTimeline(mainTasks)
	assert.NotEmpty(t, timeline)

	// Step 6: Generate dependency graph
	graph := documents.GenerateTaskDependencyGraph(mainTasks)
	assert.Contains(t, graph, "→") // Should contain dependency arrows

	// Log summary
	t.Logf("End-to-end workflow completed:")
	t.Logf("- PRD: %s (Complexity: %d)", prd.Title, prd.ComplexityScore)
	t.Logf("- TRD: %s (Architecture: %s)", trd.Title, trd.Architecture)
	t.Logf("- Main Tasks: %d", len(mainTasks))
	t.Logf("- Sub Tasks: %d", totalSubTasks)
	t.Logf("- Total Hours: %d", totalHours)
	t.Logf("- Timeline: %s", timeline)
}

// TestConcurrentDocumentGeneration tests concurrent document generation
func TestConcurrentDocumentGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	logger := logging.NewLogger(logging.DEBUG)
	cfg := &config.Config{
		AI: config.AIConfig{
			// Empty API keys will trigger mock mode
			Claude: config.ClaudeClientConfig{
				APIKey: "",
			},
			Perplexity: config.PerplexityClientConfig{
				APIKey: "",
			},
			OpenAI: config.OpenAIClientConfig{
				APIKey: "",
			},
		},
	}

	aiService, _ := ai.NewService(cfg, logger)
	ruleManager := documents.NewRuleManager(logger)
	docGen := ai.NewDocumentGenerator(aiService, ruleManager)

	// Generate multiple PRDs concurrently
	numDocs := 5
	errChan := make(chan error, numDocs)
	docChan := make(chan *documents.PRDEntity, numDocs)

	for i := 0; i < numDocs; i++ {
		go func(index int) {
			req := &ai.DocumentGenerationRequest{
				Type:       ai.DocumentTypePRD,
				Input:      fmt.Sprintf("Project %d: Test application", index),
				Repository: fmt.Sprintf("test-repo-%d", index),
			}

			resp, err := docGen.GenerateDocument(ctx, req)
			if err != nil {
				errChan <- err
				return
			}

			prd := resp.Document.(*documents.PRDEntity)
			docChan <- prd
		}(i)
	}

	// Wait for all goroutines with timeout
	timeout := time.After(30 * time.Second)
	receivedDocs := 0

	for receivedDocs < numDocs {
		select {
		case err := <-errChan:
			t.Fatalf("Error generating document: %v", err)
		case prd := <-docChan:
			assert.NotNil(t, prd)
			assert.NotEmpty(t, prd.Title)
			receivedDocs++
		case <-timeout:
			t.Fatal("Timeout waiting for document generation")
		}
	}

	assert.Equal(t, numDocs, receivedDocs)
}
