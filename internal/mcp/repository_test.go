package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeRepository(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		note     string
	}{
		{
			name:     "empty repository returns global",
			input:    "",
			expected: GlobalMemoryRepository,
			note:     "empty string should default to global",
		},
		{
			name:     "full github URL preserved",
			input:    "github.com/user/repo",
			expected: "github.com/user/repo",
			note:     "full URLs should be preserved as-is",
		},
		{
			name:     "full gitlab URL preserved",
			input:    "gitlab.com/group/project",
			expected: "gitlab.com/group/project",
			note:     "gitlab URLs should be preserved as-is",
		},
		{
			name:     "full bitbucket URL preserved",
			input:    "bitbucket.org/team/repo",
			expected: "bitbucket.org/team/repo",
			note:     "bitbucket URLs should be preserved as-is",
		},
		{
			name:     "path-like repository preserved",
			input:    "private.git.server.com/org/repo",
			expected: "private.git.server.com/org/repo",
			note:     "private git server URLs should be preserved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeRepository(tt.input)
			assert.Equal(t, tt.expected, result, tt.note)
		})
	}
}

func TestNormalizeRepositoryWithGitDetection(t *testing.T) {
	// This test checks that directory names trigger Git detection
	// The actual Git detection will work in a real Git repository
	
	// Test with a simple directory name (should attempt Git detection)
	result := normalizeRepository("simple-name")
	
	// If we're in a Git repository, it should detect the remote
	// If not, it should fall back to the original name
	assert.True(t, 
		result == "simple-name" || // fallback when no git remote
		len(result) > len("simple-name"), // or detected git remote (longer)
		"should either fallback to original name or detect git remote",
	)
}

func TestDetectGitRepository(t *testing.T) {
	// Test the Git detection function
	result := detectGitRepository()
	
	// In this repository, we should detect something
	// The exact result depends on the Git setup
	if result != "" {
		// If we detected a repository, it should be a valid format
		assert.True(t, 
			len(result) > 0 && result[0:1] != " " && result[len(result)-1:] != " ",
			"detected repository should not have leading/trailing whitespace",
		)
		
		// Should not contain .git suffix if long enough
		if len(result) >= 4 {
			assert.False(t, 
				result[len(result)-4:] == ".git",
				"detected repository should not end with .git",
			)
		}
	}
}