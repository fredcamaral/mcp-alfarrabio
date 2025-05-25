# Building Your First MCP Server with MCP-Go

Welcome to the MCP-Go tutorial! In this guide, we'll walk through building a complete MCP server from scratch. By the end, you'll have a working file manager server that can list, read, and search files.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Setting Up Your Project](#setting-up-your-project)
3. [Creating a Basic Server](#creating-a-basic-server)
4. [Adding Your First Tool](#adding-your-first-tool)
5. [Working with Resources](#working-with-resources)
6. [Adding Prompts](#adding-prompts)
7. [Error Handling](#error-handling)
8. [Testing Your Server](#testing-your-server)
9. [Advanced Features](#advanced-features)
10. [Next Steps](#next-steps)

## Prerequisites

Before we begin, make sure you have:

- Go 1.21 or later installed
- Basic knowledge of Go programming
- A text editor or IDE
- Claude Desktop or another MCP client for testing

## Setting Up Your Project

Let's start by creating a new Go project:

```bash
mkdir mcp-file-manager
cd mcp-file-manager
go mod init file-manager
```

Install the MCP-Go library:

```bash
go get github.com/yourusername/mcp-go
```

Create the main file:

```bash
touch main.go
```

## Creating a Basic Server

Let's start with a minimal MCP server:

```go
package main

import (
    "context"
    "log"
    
    "github.com/yourusername/mcp-go"
    "github.com/yourusername/mcp-go/transport"
)

func main() {
    // Create a new MCP server with a name and version
    server := mcp.NewServer("file-manager", "1.0.0")
    
    // Configure server metadata
    server.SetDescription("A simple file management MCP server")
    
    // Start the server with stdio transport
    ctx := context.Background()
    if err := server.Start(ctx, transport.Stdio()); err != nil {
        log.Fatal("Failed to start server:", err)
    }
}
```

This creates a basic server that:
- Has a name ("file-manager") and version ("1.0.0")
- Uses stdio transport (standard input/output)
- Can handle the MCP handshake

Build and test it:

```bash
go build -o file-manager
./file-manager
```

The server is now running, but it doesn't have any tools yet!

## Adding Your First Tool

Let's add a tool to list files in a directory:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "path/filepath"
    
    "github.com/yourusername/mcp-go"
    "github.com/yourusername/mcp-go/transport"
)

func main() {
    server := mcp.NewServer("file-manager", "1.0.0")
    server.SetDescription("A simple file management MCP server")
    
    // Define the list_files tool
    listFilesTool := mcp.NewTool(
        "list_files",
        "List files in a directory",
        mcp.ObjectSchema("List files parameters", map[string]interface{}{
            "path": mcp.StringParam("Directory path to list", true),
            "pattern": mcp.StringParam("File pattern to match (e.g., *.txt)", false),
        }, []string{"path"}),
    )
    
    // Add the tool with its handler
    server.AddTool(listFilesTool, mcp.ToolHandlerFunc(listFilesHandler))
    
    // Start the server
    ctx := context.Background()
    if err := server.Start(ctx, transport.Stdio()); err != nil {
        log.Fatal("Failed to start server:", err)
    }
}

func listFilesHandler(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    // Extract parameters
    path, ok := params["path"].(string)
    if !ok {
        return nil, fmt.Errorf("path parameter is required")
    }
    
    pattern := "*" // Default pattern
    if p, ok := params["pattern"].(string); ok {
        pattern = p
    }
    
    // List files
    var files []map[string]interface{}
    
    err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
        if err != nil {
            return nil // Skip files we can't access
        }
        
        // Check if file matches pattern
        matched, _ := filepath.Match(pattern, info.Name())
        if !matched {
            return nil
        }
        
        files = append(files, map[string]interface{}{
            "name": info.Name(),
            "path": filePath,
            "size": info.Size(),
            "isDir": info.IsDir(),
            "modified": info.ModTime().Format("2006-01-02 15:04:05"),
        })
        
        return nil
    })
    
    if err != nil {
        return nil, fmt.Errorf("failed to list files: %w", err)
    }
    
    return map[string]interface{}{
        "files": files,
        "count": len(files),
    }, nil
}
```

Now let's add a tool to read file contents:

```go
// Add this after the listFilesTool definition

readFileTool := mcp.NewTool(
    "read_file",
    "Read the contents of a file",
    mcp.ObjectSchema("Read file parameters", map[string]interface{}{
        "path": mcp.StringParam("Path to the file to read", true),
        "encoding": mcp.StringParam("File encoding (default: utf-8)", false),
    }, []string{"path"}),
)

server.AddTool(readFileTool, mcp.ToolHandlerFunc(readFileHandler))

// Add this handler function
func readFileHandler(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    path, ok := params["path"].(string)
    if !ok {
        return nil, fmt.Errorf("path parameter is required")
    }
    
    // Read the file
    content, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read file: %w", err)
    }
    
    // Get file info
    info, err := os.Stat(path)
    if err != nil {
        return nil, fmt.Errorf("failed to get file info: %w", err)
    }
    
    return map[string]interface{}{
        "content": string(content),
        "size": info.Size(),
        "modified": info.ModTime().Format("2006-01-02 15:04:05"),
    }, nil
}
```

## Working with Resources

Resources provide a way to expose data that can be accessed by URI. Let's add file resources:

```go
// Add this in your main function

// Register a resource pattern for files
fileResource := mcp.NewResource(
    "file:///{path}",
    "Local Files",
    "Access to local file system",
    "text/plain",
)

server.AddResource(fileResource, mcp.ResourceHandlerFunc(fileResourceHandler))

// Add this handler function
func fileResourceHandler(ctx context.Context, uri string) ([]mcp.Content, error) {
    // Extract path from URI
    // Remove the "file:///" prefix
    path := strings.TrimPrefix(uri, "file:///")
    
    // Read the file
    content, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read file: %w", err)
    }
    
    // Determine MIME type
    mimeType := "text/plain"
    if strings.HasSuffix(path, ".json") {
        mimeType = "application/json"
    } else if strings.HasSuffix(path, ".md") {
        mimeType = "text/markdown"
    }
    
    return []mcp.Content{
        {
            Type:     "text",
            Text:     string(content),
            MimeType: mimeType,
        },
    }, nil
}
```

## Adding Prompts

Prompts are reusable templates that help guide AI interactions. Let's add some useful prompts:

```go
// Add this in your main function

// Code review prompt
codeReviewPrompt := mcp.NewPrompt(
    "review_code",
    "Review code in a file",
    mcp.PromptArguments{
        "file_path": {
            Description: "Path to the code file to review",
            Required:    true,
        },
        "language": {
            Description: "Programming language (optional)",
            Required:    false,
        },
    },
)

server.AddPrompt(codeReviewPrompt, mcp.PromptHandlerFunc(codeReviewHandler))

// Add this handler function
func codeReviewHandler(ctx context.Context, args map[string]string) (*mcp.PromptMessage, error) {
    filePath := args["file_path"]
    language := args["language"]
    
    if language == "" {
        // Try to detect language from file extension
        ext := filepath.Ext(filePath)
        switch ext {
        case ".go":
            language = "Go"
        case ".js":
            language = "JavaScript"
        case ".py":
            language = "Python"
        default:
            language = "the programming language"
        }
    }
    
    prompt := fmt.Sprintf(`Please review the %s code in the file: %s

Focus on:
1. Code quality and best practices
2. Potential bugs or issues
3. Performance considerations
4. Security vulnerabilities
5. Suggestions for improvement

Provide specific, actionable feedback.`, language, filePath)
    
    return &mcp.PromptMessage{
        Role:    "user",
        Content: prompt,
    }, nil
}
```

## Error Handling

Proper error handling is crucial for a robust MCP server. Let's improve our error handling:

```go
// Create custom error types
type FileError struct {
    Path string
    Op   string
    Err  error
}

func (e *FileError) Error() string {
    return fmt.Sprintf("file operation '%s' failed for path '%s': %v", e.Op, e.Path, e.Err)
}

// Update the readFileHandler with better error handling
func readFileHandler(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    path, ok := params["path"].(string)
    if !ok {
        return nil, mcp.NewError(
            mcp.ErrorCodeInvalidParams,
            "path parameter is required",
            map[string]interface{}{
                "param": "path",
                "type": "string",
            },
        )
    }
    
    // Validate path
    if filepath.IsAbs(path) {
        return nil, mcp.NewError(
            mcp.ErrorCodeInvalidParams,
            "absolute paths are not allowed for security reasons",
            map[string]interface{}{
                "path": path,
            },
        )
    }
    
    // Read the file
    content, err := os.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, mcp.NewError(
                mcp.ErrorCodeResourceNotFound,
                "file not found",
                map[string]interface{}{
                    "path": path,
                },
            )
        }
        if os.IsPermission(err) {
            return nil, mcp.NewError(
                mcp.ErrorCodeResourceAccessDenied,
                "permission denied",
                map[string]interface{}{
                    "path": path,
                },
            )
        }
        return nil, &FileError{
            Path: path,
            Op:   "read",
            Err:  err,
        }
    }
    
    // Check file size
    info, _ := os.Stat(path)
    if info.Size() > 10*1024*1024 { // 10MB limit
        return nil, mcp.NewError(
            mcp.ErrorCodeResourceTooLarge,
            "file is too large (max 10MB)",
            map[string]interface{}{
                "path": path,
                "size": info.Size(),
            },
        )
    }
    
    return map[string]interface{}{
        "content": string(content),
        "size": info.Size(),
        "modified": info.ModTime().Format("2006-01-02 15:04:05"),
    }, nil
}
```

## Testing Your Server

### Unit Testing

Create a test file `main_test.go`:

```go
package main

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
)

func TestListFilesHandler(t *testing.T) {
    ctx := context.Background()
    
    // Test with valid parameters
    params := map[string]interface{}{
        "path": ".",
        "pattern": "*.go",
    }
    
    result, err := listFilesHandler(ctx, params)
    assert.NoError(t, err)
    assert.NotNil(t, result)
    
    // Check result structure
    resultMap := result.(map[string]interface{})
    assert.Contains(t, resultMap, "files")
    assert.Contains(t, resultMap, "count")
    
    // Test with missing path
    params = map[string]interface{}{}
    _, err = listFilesHandler(ctx, params)
    assert.Error(t, err)
}

func TestReadFileHandler(t *testing.T) {
    // Create a test file
    testContent := "Hello, MCP!"
    err := os.WriteFile("test.txt", []byte(testContent), 0644)
    assert.NoError(t, err)
    defer os.Remove("test.txt")
    
    ctx := context.Background()
    params := map[string]interface{}{
        "path": "test.txt",
    }
    
    result, err := readFileHandler(ctx, params)
    assert.NoError(t, err)
    
    resultMap := result.(map[string]interface{})
    assert.Equal(t, testContent, resultMap["content"])
}
```

### Integration Testing with Claude

1. Build your server:
```bash
go build -o file-manager
```

2. Configure Claude Desktop to use your server by adding to `claude_desktop_config.json`:
```json
{
  "mcpServers": {
    "file-manager": {
      "command": "/path/to/your/file-manager"
    }
  }
}
```

3. Restart Claude Desktop and test your tools!

## Advanced Features

### Adding Search Functionality

Let's add a more advanced search tool:

```go
searchFilesTool := mcp.NewTool(
    "search_files",
    "Search for files containing specific text",
    mcp.ObjectSchema("Search parameters", map[string]interface{}{
        "path": mcp.StringParam("Directory to search in", true),
        "query": mcp.StringParam("Text to search for", true),
        "filePattern": mcp.StringParam("File pattern (e.g., *.txt)", false),
        "maxResults": map[string]interface{}{
            "type": "integer",
            "description": "Maximum number of results",
            "minimum": 1,
            "maximum": 100,
            "default": 20,
        },
    }, []string{"path", "query"}),
)

server.AddTool(searchFilesTool, mcp.ToolHandlerFunc(searchFilesHandler))

func searchFilesHandler(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    path := params["path"].(string)
    query := params["query"].(string)
    filePattern := "*.txt"
    if p, ok := params["filePattern"].(string); ok {
        filePattern = p
    }
    maxResults := 20
    if m, ok := params["maxResults"].(float64); ok {
        maxResults = int(m)
    }
    
    var results []map[string]interface{}
    count := 0
    
    err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
        if err != nil || info.IsDir() || count >= maxResults {
            return nil
        }
        
        matched, _ := filepath.Match(filePattern, info.Name())
        if !matched {
            return nil
        }
        
        // Read file and search for query
        content, err := os.ReadFile(filePath)
        if err != nil {
            return nil
        }
        
        if strings.Contains(string(content), query) {
            // Find line numbers
            lines := strings.Split(string(content), "\n")
            var matches []map[string]interface{}
            
            for i, line := range lines {
                if strings.Contains(line, query) {
                    matches = append(matches, map[string]interface{}{
                        "line": i + 1,
                        "text": strings.TrimSpace(line),
                    })
                }
            }
            
            results = append(results, map[string]interface{}{
                "path": filePath,
                "matches": matches,
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
        "results": results,
        "totalMatches": count,
        "query": query,
    }, nil
}
```

### Adding Middleware

Add logging and metrics to your server:

```go
// Create a logging middleware
func loggingMiddleware(next mcp.Handler) mcp.Handler {
    return mcp.HandlerFunc(func(ctx context.Context, method string, params interface{}) (interface{}, error) {
        start := time.Now()
        log.Printf("Starting %s request", method)
        
        result, err := next.Handle(ctx, method, params)
        
        duration := time.Since(start)
        if err != nil {
            log.Printf("Request %s failed after %v: %v", method, duration, err)
        } else {
            log.Printf("Request %s completed in %v", method, duration)
        }
        
        return result, err
    })
}

// In your main function
server.Use(loggingMiddleware)
```

### Adding Configuration

Make your server configurable:

```go
type Config struct {
    MaxFileSize   int64  `json:"maxFileSize"`
    AllowedPaths  []string `json:"allowedPaths"`
    FileExtensions []string `json:"fileExtensions"`
}

func loadConfig() (*Config, error) {
    configFile := os.Getenv("FILE_MANAGER_CONFIG")
    if configFile == "" {
        // Default configuration
        return &Config{
            MaxFileSize: 10 * 1024 * 1024, // 10MB
            AllowedPaths: []string{"."},
            FileExtensions: []string{"*"},
        }, nil
    }
    
    data, err := os.ReadFile(configFile)
    if err != nil {
        return nil, err
    }
    
    var config Config
    if err := json.Unmarshal(data, &config); err != nil {
        return nil, err
    }
    
    return &config, nil
}
```

## Next Steps

Congratulations! You've built a complete MCP server with:
- Multiple tools for file operations
- Resource management
- Prompt templates
- Error handling
- Testing strategies
- Advanced features

### Where to Go From Here

1. **Explore the Advanced Guide**: Learn about performance optimization, security, and production deployment
2. **Check Out Examples**: See more complex server implementations in our examples directory
3. **Add More Features**: Consider adding:
   - File watching for real-time updates
   - File modification tools
   - Directory creation/deletion
   - Archive handling (zip/tar)
   
4. **Deploy Your Server**: Package and distribute your server for others to use

### Resources

- [MCP Specification](https://modelcontextprotocol.io)
- [MCP-Go API Documentation](https://pkg.go.dev/github.com/yourusername/mcp-go)
- [Example Servers](../examples/)
- [Advanced Guide](ADVANCED.md)

### Getting Help

- Join our [Discord community](https://discord.gg/mcp-go)
- Check out [GitHub Issues](https://github.com/yourusername/mcp-go/issues)
- Read the [FAQ](FAQ.md)

Happy coding with MCP-Go! 🚀