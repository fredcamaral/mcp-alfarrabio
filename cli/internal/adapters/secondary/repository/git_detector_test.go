package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitDetector_DetectFromPath(t *testing.T) {
	detector := NewGitDetector()
	ctx := context.Background()

	// Test non-git directory
	tempDir := t.TempDir()
	info, err := detector.DetectFromPath(ctx, tempDir)
	require.NoError(t, err)
	assert.False(t, info.IsGitRepo)
	assert.Equal(t, "local", info.Provider)
	assert.False(t, info.HasRemote)
	assert.Equal(t, filepath.Base(tempDir), info.Name)

	// Test with simulated git directory
	gitDir := filepath.Join(tempDir, ".git")
	err = os.MkdirAll(gitDir, 0o750)
	require.NoError(t, err)

	info, err = detector.DetectFromPath(ctx, tempDir)
	require.NoError(t, err)
	assert.True(t, info.IsGitRepo)
	assert.Equal(t, tempDir, info.Path)
}

func TestGitDetector_GetRepositoryName(t *testing.T) {
	detector := NewGitDetector()
	ctx := context.Background()

	tempDir := t.TempDir()
	name, err := detector.GetRepositoryName(ctx, tempDir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Base(tempDir), name)
}

func TestGitDetector_IsValidRepository(t *testing.T) {
	detector := NewGitDetector()
	ctx := context.Background()

	// Test non-git directory
	tempDir := t.TempDir()
	assert.False(t, detector.IsValidRepository(ctx, tempDir))

	// Test with .git directory
	gitDir := filepath.Join(tempDir, ".git")
	err := os.MkdirAll(gitDir, 0o750)
	require.NoError(t, err)
	assert.True(t, detector.IsValidRepository(ctx, tempDir))
}

func TestGitDetector_DetectProvider(t *testing.T) {
	detector := NewGitDetector()

	tests := []struct {
		url      string
		expected string
	}{
		{"https://github.com/user/repo.git", "github"},
		{"git@github.com:user/repo.git", "github"},
		{"https://gitlab.com/user/repo.git", "gitlab"},
		{"git@gitlab.com:user/repo.git", "gitlab"},
		{"https://bitbucket.org/user/repo.git", "bitbucket"},
		{"https://dev.azure.com/user/repo.git", "azure"},
		{"https://example.com/user/repo.git", "git"},
	}

	for _, test := range tests {
		result := detector.detectProvider(test.url)
		assert.Equal(t, test.expected, result, "URL: %s", test.url)
	}
}

func TestGitDetector_ExtractNameFromRemoteURL(t *testing.T) {
	detector := NewGitDetector()

	tests := []struct {
		url      string
		expected string
	}{
		{"https://github.com/user/repo.git", "repo"},
		{"git@github.com:user/repo.git", "repo"},
		{"https://gitlab.com/group/subgroup/repo.git", "repo"},
		{"git@gitlab.com:group/repo.git", "repo"},
		{"https://github.com/user/repo", "repo"},
		{"ssh://git@example.com/path/to/repo.git", "repo"},
	}

	for _, test := range tests {
		result := detector.extractNameFromRemoteURL(test.url)
		assert.Equal(t, test.expected, result, "URL: %s", test.url)
	}
}
