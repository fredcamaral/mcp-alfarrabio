// File Browser MCP Server Example
//
// This example demonstrates a secure file browser MCP server that shows:
//   - Multiple coordinated tools
//   - Resource management for file access
//   - Security best practices (path validation, size limits)
//   - Advanced schema validation
//   - Error handling and logging
package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yourusername/mcp-go"
	"github.com/yourusername/mcp-go/protocol"
	"github.com/yourusername/mcp-go/transport"
)

const (
	// Security limits
	maxFileSize = 10 * 1024 * 1024 // 10MB
	maxSearchResults = 100
)

// Server configuration
type Config struct {
	BasePath       string
	AllowedExts    []string
	ForbiddenPaths []string
}

func main() {
	// Configure the file browser
	config := &Config{
		BasePath: ".", // Current directory by default
		AllowedExts: []string{
			".txt", ".md", ".json", ".yaml", ".yml",
			".go", ".js", ".py", ".java", ".c", ".cpp",
			".html", ".css", ".xml", ".toml",
		},
		ForbiddenPaths: []string{
			".git", "node_modules", ".env", "secrets",
		},
	}

	// Create server
	server := mcp.NewServer("file-browser", "1.0.0")
	server.SetDescription("A secure file browser MCP server")

	// Register tools
	registerListTool(server, config)
	registerReadTool(server, config)
	registerSearchTool(server, config)
	registerInfoTool(server, config)

	// Register resources
	registerFileResource(server, config)

	// Start server
	ctx := context.Background()
	log.Println("File Browser MCP server starting...")
	if err := server.Start(ctx, transport.Stdio()); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// registerListTool lists files in a directory
func registerListTool(server *mcp.Server, config *Config) {
	tool := mcp.NewTool(
		"list_files",
		"List files in a directory",
		mcp.ObjectSchema("List parameters", map[string]interface{}{
			"path": mcp.StringParam("Directory path (relative to base)", true),
			"pattern": mcp.StringParam("File pattern to match (e.g., *.txt)", false),
			"recursive": mcp.BooleanParam("Search recursively", false),
		}, []string{"path"}),
	)

	handler := mcp.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		path := params["path"].(string)
		pattern := getStringParam(params, "pattern", "*")
		recursive := getBoolParam(params, "recursive", false)

		// Validate and resolve path
		fullPath, err := validatePath(config, path)
		if err != nil {
			return nil, err
		}

		var files []map[string]interface{}

		if recursive {
			err = filepath.Walk(fullPath, func(filePath string, info fs.FileInfo, err error) error {
				if err != nil {
					return nil // Skip inaccessible files
				}

				if shouldSkipFile(config, filePath, info) {
					if info.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}

				matched, _ := filepath.Match(pattern, info.Name())
				if matched || info.IsDir() {
					relPath, _ := filepath.Rel(config.BasePath, filePath)
					files = append(files, fileInfo(relPath, info))
				}

				return nil
			})
		} else {
			entries, err := os.ReadDir(fullPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read directory: %w", err)
			}

			for _, entry := range entries {
				info, err := entry.Info()
				if err != nil {
					continue
				}

				if shouldSkipFile(config, filepath.Join(fullPath, entry.Name()), info) {
					continue
				}

				matched, _ := filepath.Match(pattern, entry.Name())
				if matched || entry.IsDir() {
					relPath := filepath.Join(path, entry.Name())
					files = append(files, fileInfo(relPath, info))
				}
			}
		}

		return map[string]interface{}{
			"files": files,
			"count": len(files),
			"path":  path,
		}, nil
	})

	server.AddTool(tool, handler)
}

// registerReadTool reads file contents
func registerReadTool(server *mcp.Server, config *Config) {
	tool := mcp.NewTool(
		"read_file",
		"Read the contents of a file",
		mcp.ObjectSchema("Read parameters", map[string]interface{}{
			"path": mcp.StringParam("File path to read", true),
			"encoding": mcp.StringParam("File encoding (default: utf-8)", false),
		}, []string{"path"}),
	)

	handler := mcp.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		path := params["path"].(string)

		// Validate path
		fullPath, err := validatePath(config, path)
		if err != nil {
			return nil, err
		}

		// Check file info
		info, err := os.Stat(fullPath)
		if err != nil {
			return nil, fmt.Errorf("file not found: %w", err)
		}

		if info.IsDir() {
			return nil, fmt.Errorf("path is a directory, not a file")
		}

		if info.Size() > maxFileSize {
			return nil, fmt.Errorf("file too large: %d bytes (max: %d)", info.Size(), maxFileSize)
		}

		// Check allowed extensions
		if !isExtensionAllowed(config, fullPath) {
			return nil, fmt.Errorf("file type not allowed")
		}

		// Read file
		content, err := os.ReadFile(fullPath) // #nosec G304 -- Path is validated by validatePath()
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}

		return map[string]interface{}{
			"content": string(content),
			"path":    path,
			"size":    info.Size(),
			"modified": info.ModTime().Format(time.RFC3339),
		}, nil
	})

	server.AddTool(tool, handler)
}

// registerSearchTool searches for files containing text
func registerSearchTool(server *mcp.Server, config *Config) {
	tool := mcp.NewTool(
		"search_files",
		"Search for files containing specific text",
		mcp.ObjectSchema("Search parameters", map[string]interface{}{
			"query": mcp.StringParam("Text to search for", true),
			"path": mcp.StringParam("Directory to search in", false),
			"filePattern": mcp.StringParam("File pattern (e.g., *.txt)", false),
			"caseSensitive": mcp.BooleanParam("Case sensitive search", false),
			"maxResults": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results",
				"minimum":     1,
				"maximum":     maxSearchResults,
				"default":     20,
			},
		}, []string{"query"}),
	)

	handler := mcp.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		query := params["query"].(string)
		searchPath := getStringParam(params, "path", ".")
		filePattern := getStringParam(params, "filePattern", "*")
		caseSensitive := getBoolParam(params, "caseSensitive", false)
		maxResults := getIntParam(params, "maxResults", 20)

		// Validate path
		fullPath, err := validatePath(config, searchPath)
		if err != nil {
			return nil, err
		}

		// Prepare search
		if !caseSensitive {
			query = strings.ToLower(query)
		}

		var results []map[string]interface{}
		count := 0

		err = filepath.Walk(fullPath, func(filePath string, info fs.FileInfo, err error) error {
			if err != nil || info.IsDir() || count >= maxResults {
				return nil
			}

			if shouldSkipFile(config, filePath, info) {
				return nil
			}

			matched, _ := filepath.Match(filePattern, info.Name())
			if !matched {
				return nil
			}

			// Don't search files that are too large
			if info.Size() > maxFileSize {
				return nil
			}

			// Read and search file
			content, err := os.ReadFile(filePath) // #nosec G304 -- Path comes from filepath.Walk with validated base path
			if err != nil {
				return nil
			}

			searchContent := string(content)
			if !caseSensitive {
				searchContent = strings.ToLower(searchContent)
			}

			if strings.Contains(searchContent, query) {
				relPath, _ := filepath.Rel(config.BasePath, filePath)
				
				// Find matching lines
				lines := strings.Split(string(content), "\n")
				var matches []map[string]interface{}
				
				for i, line := range lines {
					searchLine := line
					if !caseSensitive {
						searchLine = strings.ToLower(line)
					}
					
					if strings.Contains(searchLine, query) {
						matches = append(matches, map[string]interface{}{
							"line": i + 1,
							"text": strings.TrimSpace(line),
						})
						
						if len(matches) >= 5 { // Limit matches per file
							break
						}
					}
				}

				results = append(results, map[string]interface{}{
					"path":       relPath,
					"matches":    matches,
					"matchCount": len(matches),
				})
				count++
			}

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("search failed: %w", err)
		}

		return map[string]interface{}{
			"results":      results,
			"totalMatches": count,
			"query":        params["query"].(string), // Original query
		}, nil
	})

	server.AddTool(tool, handler)
}

// registerInfoTool gets detailed file information
func registerInfoTool(server *mcp.Server, config *Config) {
	tool := mcp.NewTool(
		"file_info",
		"Get detailed information about a file or directory",
		mcp.ObjectSchema("Info parameters", map[string]interface{}{
			"path": mcp.StringParam("Path to file or directory", true),
		}, []string{"path"}),
	)

	handler := mcp.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		path := params["path"].(string)

		// Validate path
		fullPath, err := validatePath(config, path)
		if err != nil {
			return nil, err
		}

		info, err := os.Stat(fullPath)
		if err != nil {
			return nil, fmt.Errorf("path not found: %w", err)
		}

		result := map[string]interface{}{
			"path":         path,
			"name":         info.Name(),
			"size":         info.Size(),
			"isDirectory":  info.IsDir(),
			"permissions":  info.Mode().String(),
			"modified":     info.ModTime().Format(time.RFC3339),
		}

		if info.IsDir() {
			// Count items in directory
			entries, err := os.ReadDir(fullPath)
			if err == nil {
				fileCount := 0
				dirCount := 0
				for _, entry := range entries {
					if entry.IsDir() {
						dirCount++
					} else {
						fileCount++
					}
				}
				result["fileCount"] = fileCount
				result["directoryCount"] = dirCount
			}
		} else {
			// File-specific info
			result["extension"] = filepath.Ext(info.Name())
			result["readable"] = isExtensionAllowed(config, fullPath)
		}

		return result, nil
	})

	server.AddTool(tool, handler)
}

// registerFileResource registers file access as a resource
func registerFileResource(server *mcp.Server, config *Config) {
	resource := mcp.NewResource(
		"file:///{path}",
		"Local Files",
		"Access to local file system",
		"text/plain",
	)

	handler := mcp.ResourceHandlerFunc(func(ctx context.Context, uri string) ([]protocol.Content, error) {
		// Extract path from URI
		path := strings.TrimPrefix(uri, "file:///")
		
		// Validate path
		fullPath, err := validatePath(config, path)
		if err != nil {
			return nil, err
		}

		// Check if it's allowed
		if !isExtensionAllowed(config, fullPath) {
			return nil, fmt.Errorf("file type not allowed")
		}

		// Read file
		content, err := os.ReadFile(fullPath) // #nosec G304 -- Path is validated by validatePath()
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}

		// Determine MIME type
		mimeType := "text/plain"
		ext := filepath.Ext(fullPath)
		switch ext {
		case ".json":
			mimeType = "application/json"
		case ".xml":
			mimeType = "application/xml"
		case ".html":
			mimeType = "text/html"
		case ".md":
			mimeType = "text/markdown"
		case ".yaml", ".yml":
			mimeType = "text/yaml"
		}

		return []protocol.Content{
			{
				Type:     "text",
				Text:     string(content),
				MimeType: mimeType,
			},
		}, nil
	})

	server.AddResource(resource, handler)
}

// Helper functions

func validatePath(config *Config, path string) (string, error) {
	// Clean the path
	cleanPath := filepath.Clean(path)
	
	// Resolve to absolute path
	fullPath := filepath.Join(config.BasePath, cleanPath)
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// Ensure path is within base directory
	absBase, _ := filepath.Abs(config.BasePath)
	if !strings.HasPrefix(absPath, absBase) {
		return "", fmt.Errorf("access denied: path outside allowed directory")
	}

	// Check forbidden paths
	for _, forbidden := range config.ForbiddenPaths {
		if strings.Contains(absPath, forbidden) {
			return "", fmt.Errorf("access denied: forbidden path")
		}
	}

	return absPath, nil
}

func isExtensionAllowed(config *Config, path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, allowed := range config.AllowedExts {
		if ext == allowed {
			return true
		}
	}
	return false
}

func shouldSkipFile(config *Config, path string, info fs.FileInfo) bool {
	// Skip hidden files (starting with .)
	if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
		return true
	}

	// Skip forbidden paths
	for _, forbidden := range config.ForbiddenPaths {
		if strings.Contains(path, forbidden) {
			return true
		}
	}

	return false
}

func fileInfo(path string, info fs.FileInfo) map[string]interface{} {
	return map[string]interface{}{
		"name":        info.Name(),
		"path":        path,
		"size":        info.Size(),
		"isDirectory": info.IsDir(),
		"modified":    info.ModTime().Format(time.RFC3339),
		"permissions": info.Mode().String(),
	}
}

// Parameter extraction helpers
func getStringParam(params map[string]interface{}, key, defaultValue string) string {
	if val, ok := params[key].(string); ok {
		return val
	}
	return defaultValue
}

func getBoolParam(params map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := params[key].(bool); ok {
		return val
	}
	return defaultValue
}

func getIntParam(params map[string]interface{}, key string, defaultValue int) int {
	if val, ok := params[key].(float64); ok {
		return int(val)
	}
	return defaultValue
}