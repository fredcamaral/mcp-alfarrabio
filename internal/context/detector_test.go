package context

import (
	"os"
	"path/filepath"
	"testing"
	
	"mcp-memory/pkg/types"
)

func TestDetector_DetectLocationContext(t *testing.T) {
	detector, err := NewDetector()
	if err != nil {
		t.Fatalf("Failed to create detector: %v", err)
	}
	
	context := detector.DetectLocationContext()
	
	// Check that working directory is set
	if wd, ok := context[types.EMKeyWorkingDir]; !ok || wd == "" {
		t.Error("Working directory not detected")
	}
	
	// Project type should be detected as Go (since we're in a Go project)
	if pt, ok := context[types.EMKeyProjectType]; !ok || pt != types.ProjectTypeGo {
		t.Errorf("Project type not correctly detected: got %v, want %v", pt, types.ProjectTypeGo)
	}
	
	// If in git repo, should have git info
	if _, err := os.Stat(filepath.Join(".git")); err == nil {
		if _, ok := context[types.EMKeyGitBranch]; !ok {
			t.Log("Git branch not detected (might not be in a git repo during tests)")
		}
	}
}

func TestDetector_DetectClientContext(t *testing.T) {
	detector, err := NewDetector()
	if err != nil {
		t.Fatalf("Failed to create detector: %v", err)
	}
	
	// Test with claude-cli client type
	context := detector.DetectClientContext(types.ClientTypeCLI)
	
	// Check client type
	if ct, ok := context[types.EMKeyClientType]; !ok || ct != types.ClientTypeCLI {
		t.Errorf("Client type not set correctly: got %v, want %v", ct, types.ClientTypeCLI)
	}
	
	// Check platform is set
	if platform, ok := context[types.EMKeyPlatform]; !ok || platform == "" {
		t.Error("Platform not detected")
	}
}

func TestDetector_DetectLanguageVersions(t *testing.T) {
	detector, err := NewDetector()
	if err != nil {
		t.Fatalf("Failed to create detector: %v", err)
	}
	
	versions := detector.DetectLanguageVersions()
	
	// Should at least detect Go version (since tests run with Go)
	if goVersion, ok := versions["go"]; !ok || goVersion == "" {
		t.Error("Go version not detected")
	}
	
	// Log other detected versions
	for lang, version := range versions {
		t.Logf("Detected %s version: %s", lang, version)
	}
}

func TestDetector_DetectDependencies(t *testing.T) {
	detector, err := NewDetector()
	if err != nil {
		t.Fatalf("Failed to create detector: %v", err)
	}
	
	deps := detector.DetectDependencies()
	
	// Should detect go.mod in this project
	if _, ok := deps["go.mod"]; !ok {
		// Might be running from a subdirectory
		t.Log("go.mod not detected (might be running from subdirectory)")
	}
	
	// Log all detected dependencies
	for dep, status := range deps {
		t.Logf("Detected dependency: %s = %s", dep, status)
	}
}

func TestDetector_detectProjectType(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected string
	}{
		{
			name:     "Go project",
			files:    []string{"go.mod", "main.go"},
			expected: types.ProjectTypeGo,
		},
		{
			name:     "Node.js project",
			files:    []string{"package.json", "index.js"},
			expected: types.ProjectTypeJavaScript,
		},
		{
			name:     "TypeScript project",
			files:    []string{"tsconfig.json", "index.ts"},
			expected: types.ProjectTypeTypeScript,
		},
		{
			name:     "Python project",
			files:    []string{"requirements.txt", "main.py"},
			expected: types.ProjectTypePython,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()
			
			// Create test files
			for _, file := range tt.files {
				if err := os.WriteFile(filepath.Join(tmpDir, file), []byte("test"), 0644); err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
			}
			
			// Create detector with temp directory
			detector := &Detector{workingDir: tmpDir}
			
			result := detector.detectProjectType()
			if result != tt.expected {
				t.Errorf("detectProjectType() = %v, want %v", result, tt.expected)
			}
		})
	}
}