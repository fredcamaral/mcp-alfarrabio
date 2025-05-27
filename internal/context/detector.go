package context

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	
	"mcp-memory/pkg/types"
)

const (
	// presentValue is used to indicate a file or feature is present
	presentValue = "present"
)

// Detector provides context detection capabilities
type Detector struct {
	workingDir string
}

// NewDetector creates a new context detector
func NewDetector() (*Detector, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}
	
	return &Detector{
		workingDir: wd,
	}, nil
}

// DetectLocationContext detects location-related context
func (d *Detector) DetectLocationContext() map[string]interface{} {
	context := make(map[string]interface{})
	
	// Working directory
	context[types.EMKeyWorkingDir] = d.workingDir
	
	// Git information
	if gitInfo := d.detectGitInfo(); gitInfo != nil {
		if branch, ok := gitInfo["branch"]; ok {
			context[types.EMKeyGitBranch] = branch
		}
		if commit, ok := gitInfo["commit"]; ok {
			context[types.EMKeyGitCommit] = commit
		}
		if repoRoot, ok := gitInfo["repo_root"]; ok {
			// Calculate relative path from repo root
			relPath, _ := filepath.Rel(repoRoot.(string), d.workingDir)
			context[types.EMKeyRelativePath] = relPath
		}
	}
	
	// Project type detection
	context[types.EMKeyProjectType] = d.detectProjectType()
	
	return context
}

// DetectClientContext detects client-related context
func (d *Detector) DetectClientContext(clientType string) map[string]interface{} {
	context := make(map[string]interface{})
	
	// Client type (provided by caller)
	if clientType != "" {
		context[types.EMKeyClientType] = clientType
	}
	
	// Platform information
	context[types.EMKeyPlatform] = runtime.GOOS + "/" + runtime.GOARCH
	
	// Environment variables (filtered for safety)
	env := make(map[string]string)
	safeEnvVars := []string{
		"DEBUG", "VERBOSE", "LOG_LEVEL", "ENV", "ENVIRONMENT",
		"CI", "CI_COMMIT_SHA", "CI_BRANCH", "GITHUB_ACTIONS",
		"TERM", "SHELL", "EDITOR", "LANG", "LC_ALL",
	}
	
	for _, key := range safeEnvVars {
		if value := os.Getenv(key); value != "" {
			env[key] = value
		}
	}
	
	if len(env) > 0 {
		context[types.EMKeyEnvironment] = env
	}
	
	return context
}

// DetectLanguageVersions detects programming language versions
func (d *Detector) DetectLanguageVersions() map[string]string {
	versions := make(map[string]string)
	
	// Go version
	if out, err := exec.Command("go", "version").Output(); err == nil {
		parts := strings.Fields(string(out))
		if len(parts) >= 3 {
			versions["go"] = strings.TrimPrefix(parts[2], "go")
		}
	}
	
	// Python version
	if out, err := exec.Command("python3", "--version").Output(); err == nil {
		parts := strings.Fields(string(out))
		if len(parts) >= 2 {
			versions["python"] = parts[1]
		}
	}
	
	// Node.js version
	if out, err := exec.Command("node", "--version").Output(); err == nil {
		versions["node"] = strings.TrimSpace(strings.TrimPrefix(string(out), "v"))
	}
	
	// Java version
	if out, err := exec.Command("java", "-version").CombinedOutput(); err == nil {
		lines := strings.Split(string(out), "\n")
		if len(lines) > 0 {
			// Java version is in quotes on the first line
			if parts := strings.Split(lines[0], "\""); len(parts) >= 2 {
				versions["java"] = parts[1]
			}
		}
	}
	
	return versions
}

// DetectDependencies detects project dependencies
func (d *Detector) DetectDependencies() map[string]string {
	deps := make(map[string]string)
	
	// Go modules
	if _, err := os.Stat(filepath.Join(d.workingDir, "go.mod")); err == nil {
		deps["go.mod"] = presentValue
		// Could parse go.mod for specific versions if needed
	}
	
	// Node.js
	if _, err := os.Stat(filepath.Join(d.workingDir, "package.json")); err == nil {
		deps["package.json"] = presentValue
	}
	
	// Python
	if _, err := os.Stat(filepath.Join(d.workingDir, "requirements.txt")); err == nil {
		deps["requirements.txt"] = presentValue
	}
	if _, err := os.Stat(filepath.Join(d.workingDir, "pyproject.toml")); err == nil {
		deps["pyproject.toml"] = presentValue
	}
	
	// Rust
	if _, err := os.Stat(filepath.Join(d.workingDir, "Cargo.toml")); err == nil {
		deps["Cargo.toml"] = presentValue
	}
	
	return deps
}

// detectGitInfo detects git repository information
func (d *Detector) detectGitInfo() map[string]interface{} {
	info := make(map[string]interface{})
	
	// Check if we're in a git repository
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = d.workingDir
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	
	repoRoot := strings.TrimSpace(string(out))
	info["repo_root"] = repoRoot
	
	// Get current branch
	cmd = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = d.workingDir
	if out, err := cmd.Output(); err == nil {
		info["branch"] = strings.TrimSpace(string(out))
	}
	
	// Get current commit
	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = d.workingDir
	if out, err := cmd.Output(); err == nil {
		info["commit"] = strings.TrimSpace(string(out))[:8] // Short SHA
	}
	
	return info
}

// detectProjectType detects the type of project based on files present
func (d *Detector) detectProjectType() string {
	// Check for various project indicators
	checks := []struct {
		file     string
		projType string
	}{
		{"go.mod", types.ProjectTypeGo},
		{"go.sum", types.ProjectTypeGo},
		{"package.json", types.ProjectTypeJavaScript},
		{"tsconfig.json", types.ProjectTypeTypeScript},
		{"requirements.txt", types.ProjectTypePython},
		{"pyproject.toml", types.ProjectTypePython},
		{"setup.py", types.ProjectTypePython},
		{"Cargo.toml", types.ProjectTypeRust},
		{"pom.xml", types.ProjectTypeJava},
		{"build.gradle", types.ProjectTypeJava},
	}
	
	for _, check := range checks {
		if _, err := os.Stat(filepath.Join(d.workingDir, check.file)); err == nil {
			return check.projType
		}
	}
	
	// Check for file extensions as fallback
	entries, err := os.ReadDir(d.workingDir)
	if err == nil {
		extCounts := make(map[string]int)
		for _, entry := range entries {
			if !entry.IsDir() {
				ext := filepath.Ext(entry.Name())
				extCounts[ext]++
			}
		}
		
		// Determine by most common extension
		maxCount := 0
		var dominantType string
		
		for ext, count := range extCounts {
			if count > maxCount {
				maxCount = count
				switch ext {
				case ".go":
					dominantType = types.ProjectTypeGo
				case ".py":
					dominantType = types.ProjectTypePython
				case ".js", ".jsx":
					dominantType = types.ProjectTypeJavaScript
				case ".ts", ".tsx":
					dominantType = types.ProjectTypeTypeScript
				case ".rs":
					dominantType = types.ProjectTypeRust
				case ".java":
					dominantType = types.ProjectTypeJava
				}
			}
		}
		
		if dominantType != "" {
			return dominantType
		}
	}
	
	return types.ProjectTypeUnknown
}