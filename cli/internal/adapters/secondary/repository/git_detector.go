// Package repository provides repository detection implementation
// for the lerian-mcp-memory CLI application.
package repository

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"lerian-mcp-memory-cli/internal/domain/ports"
)

// GitDetector implements repository detection for Git repositories
type GitDetector struct{}

// NewGitDetector creates a new Git repository detector
func NewGitDetector() *GitDetector {
	return &GitDetector{}
}

// DetectCurrent detects repository information from current working directory
func (g *GitDetector) DetectCurrent(ctx context.Context) (*ports.RepositoryInfo, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	return g.DetectFromPath(ctx, cwd)
}

// DetectFromPath detects repository information from specific path
func (g *GitDetector) DetectFromPath(ctx context.Context, path string) (*ports.RepositoryInfo, error) {
	// Check if we're in a git repository
	gitDir, err := g.findGitRoot(path)
	if err != nil {
		// Not a git repository, use directory name
		return &ports.RepositoryInfo{
			Path:      path,
			Name:      g.getDirectoryName(path),
			Provider:  "local",
			IsGitRepo: false,
			HasRemote: false,
		}, nil
	}

	info := &ports.RepositoryInfo{
		Path:      gitDir,
		IsGitRepo: true,
	}

	// Get repository name from directory
	info.Name = g.getDirectoryName(gitDir)

	// Get remote URL if available
	remoteURL, err := g.getRemoteURL(ctx, gitDir)
	if err == nil && remoteURL != "" {
		info.RemoteURL = remoteURL
		info.HasRemote = true
		info.Provider = g.detectProvider(remoteURL)

		// Try to extract a better name from remote URL
		if remoteName := g.extractNameFromRemoteURL(remoteURL); remoteName != "" {
			info.Name = remoteName
		}
	} else {
		info.Provider = "git"
	}

	// Get current branch
	branch, err := g.getCurrentBranch(ctx, gitDir)
	if err == nil {
		info.Branch = branch
	}

	return info, nil
}

// GetRepositoryName extracts a clean repository name for identification
func (g *GitDetector) GetRepositoryName(ctx context.Context, path string) (string, error) {
	info, err := g.DetectFromPath(ctx, path)
	if err != nil {
		return "", err
	}

	return info.Name, nil
}

// IsValidRepository checks if path contains a valid repository
func (g *GitDetector) IsValidRepository(ctx context.Context, path string) bool {
	_, err := g.findGitRoot(path)
	return err == nil
}

// Helper methods

func (g *GitDetector) findGitRoot(startPath string) (string, error) {
	path := startPath
	for {
		gitDir := filepath.Join(path, ".git")
		if info, err := os.Stat(gitDir); err == nil {
			if info.IsDir() {
				return path, nil
			}
			// .git might be a file (in case of git worktrees)
			if data, err := os.ReadFile(filepath.Clean(gitDir)); err == nil {
				content := strings.TrimSpace(string(data))
				if strings.HasPrefix(content, "gitdir: ") {
					return path, nil
				}
			}
		}

		parent := filepath.Dir(path)
		if parent == path {
			// Reached root directory
			break
		}
		path = parent
	}

	return "", fmt.Errorf("not a git repository")
}

func (g *GitDetector) getDirectoryName(path string) string {
	return filepath.Base(path)
}

func (g *GitDetector) getRemoteURL(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

func (g *GitDetector) getCurrentBranch(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

func (g *GitDetector) detectProvider(remoteURL string) string {
	lowerURL := strings.ToLower(remoteURL)

	switch {
	case strings.Contains(lowerURL, "github.com"):
		return "github"
	case strings.Contains(lowerURL, "gitlab.com"):
		return "gitlab"
	case strings.Contains(lowerURL, "bitbucket.org"):
		return "bitbucket"
	case strings.Contains(lowerURL, "azure.com") || strings.Contains(lowerURL, "visualstudio.com"):
		return "azure"
	default:
		return "git"
	}
}

func (g *GitDetector) extractNameFromRemoteURL(remoteURL string) string {
	// Handle various URL formats:
	// https://github.com/user/repo.git
	// git@github.com:user/repo.git
	// https://gitlab.com/group/subgroup/repo

	// Remove .git suffix
	url := strings.TrimSuffix(remoteURL, ".git")

	// Handle SSH format (git@host:path)
	if strings.Contains(url, "@") && strings.Contains(url, ":") {
		parts := strings.Split(url, ":")
		if len(parts) >= 2 {
			path := parts[len(parts)-1]
			return g.extractRepoNameFromPath(path)
		}
	}

	// Handle HTTPS format
	if strings.HasPrefix(url, "http") {
		// Remove protocol and host
		parts := strings.Split(url, "/")
		if len(parts) >= 2 {
			return parts[len(parts)-1]
		}
	}

	// Fallback: try to extract from path
	return g.extractRepoNameFromPath(url)
}

func (g *GitDetector) extractRepoNameFromPath(path string) string {
	// Split by / and take the last non-empty part
	parts := strings.Split(path, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" {
			return parts[i]
		}
	}
	return ""
}
