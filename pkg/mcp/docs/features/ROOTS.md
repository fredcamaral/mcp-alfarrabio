# Roots Feature Documentation

## Overview

The Roots feature provides MCP servers with the ability to expose file system access points. This is essential for clients like VS Code Copilot that need to understand the workspace structure and access files within defined boundaries.

## Implementation

### Basic Usage

```go
import (
    "github.com/yourusername/mcp-go/roots"
    "github.com/yourusername/mcp-go/server"
)

// Create server with roots support
srv := server.NewExtendedServer("File Server", "1.0.0")

// Default roots are automatically added:
// - Home directory
// - Current working directory  
// - System temp directory

// Add custom roots
srv.AddRoot(roots.Root{
    URI:         "file:///workspace/project",
    Name:        "Project Root",
    Description: "Main project directory",
})

srv.AddRoot(roots.Root{
    URI:         "file:///data/shared",
    Name:        "Shared Data",
    Description: "Shared data directory",
})
```

### Custom Roots Handler

```go
type CustomRootsHandler struct {
    roots []roots.Root
}

func NewCustomRootsHandler() *CustomRootsHandler {
    return &CustomRootsHandler{
        roots: []roots.Root{
            {
                URI:         "s3://my-bucket/data",
                Name:        "S3 Data",
                Description: "Cloud storage root",
            },
            {
                URI:         "git://github.com/user/repo",
                Name:        "Git Repository",
                Description: "Remote git repository",
            },
        },
    }
}

func (h *CustomRootsHandler) ListRoots(ctx context.Context, params json.RawMessage) (interface{}, error) {
    // Could dynamically determine roots based on context
    return roots.ListRootsResponse{
        Roots: h.roots,
    }, nil
}

// Set custom handler
srv.SetRootsHandler(NewCustomRootsHandler())
```

## Request Format

```json
{
    "method": "roots/list",
    "params": {} // No parameters required
}
```

## Response Format

```json
{
    "roots": [
        {
            "uri": "file:///Users/alice",
            "name": "Home",
            "description": "User home directory"
        },
        {
            "uri": "file:///workspace/project",
            "name": "Project Root",
            "description": "Current project directory"
        },
        {
            "uri": "file:///tmp",
            "name": "Temporary",
            "description": "System temporary directory"
        }
    ]
}
```

## URI Schemes

Roots support various URI schemes:

- **file://**: Local file system paths
- **workspace://**: Workspace-relative paths
- **git://**: Git repositories
- **s3://**: S3 buckets
- **http(s)://**: Remote resources
- **custom://**: Custom schemes for your application

## Integration with Resources

Roots work seamlessly with the Resources feature:

```go
// When a root is defined
srv.AddRoot(roots.Root{
    URI:  "file:///workspace",
    Name: "Workspace",
})

// Resources within that root can be accessed
srv.AddResource(
    protocol.Resource{
        URI:      "file:///workspace/config.json",
        Name:     "Configuration",
        MimeType: "application/json",
    },
    resourceHandler,
)
```

## Dynamic Roots

Implement dynamic root discovery:

```go
func (h *DynamicRootsHandler) ListRoots(ctx context.Context, params json.RawMessage) (interface{}, error) {
    var roots []roots.Root
    
    // Discover project directories
    projects, _ := findProjectDirectories()
    for _, proj := range projects {
        roots = append(roots, roots.Root{
            URI:         "file://" + proj.Path,
            Name:        proj.Name,
            Description: fmt.Sprintf("Project: %s", proj.Type),
        })
    }
    
    // Add user-specific roots
    userRoots, _ := getUserSpecificRoots(ctx)
    roots = append(roots, userRoots...)
    
    return roots.ListRootsResponse{
        Roots: roots,
    }, nil
}
```

## Security Considerations

### Path Validation

```go
func validateRootAccess(uri string) error {
    parsed, err := url.Parse(uri)
    if err != nil {
        return err
    }
    
    if parsed.Scheme == "file" {
        path := parsed.Path
        
        // Ensure path is absolute
        if !filepath.IsAbs(path) {
            return fmt.Errorf("root paths must be absolute")
        }
        
        // Check if path exists and is accessible
        info, err := os.Stat(path)
        if err != nil {
            return fmt.Errorf("root path not accessible: %w", err)
        }
        
        // Ensure it's a directory
        if !info.IsDir() {
            return fmt.Errorf("root must be a directory")
        }
        
        // Check permissions
        if !isReadable(path) {
            return fmt.Errorf("root directory not readable")
        }
    }
    
    return nil
}
```

### Access Control

```go
type SecureRootsHandler struct {
    allowedPaths []string
    userRoles    map[string][]string
}

func (h *SecureRootsHandler) ListRoots(ctx context.Context, params json.RawMessage) (interface{}, error) {
    // Get user from context
    user := getUserFromContext(ctx)
    
    var roots []roots.Root
    for _, path := range h.allowedPaths {
        // Check if user has access to this path
        if h.userHasAccess(user, path) {
            roots = append(roots, roots.Root{
                URI:  "file://" + path,
                Name: filepath.Base(path),
            })
        }
    }
    
    return roots.ListRootsResponse{Roots: roots}, nil
}
```

## Client Support

| Client | Roots Required | Usage |
|--------|---------------|-------|
| VS Code Copilot | ✅ Yes | Workspace navigation |
| fast-agent | ✅ Yes | File system access |
| Claude Desktop | ❌ No | Not supported |
| Cursor | ❌ No | Uses own file access |

## Best Practices

1. **Descriptive Names**: Use clear, descriptive names for roots
2. **Hierarchical Organization**: Organize roots hierarchically when possible
3. **Access Validation**: Always validate root access permissions
4. **Path Normalization**: Normalize paths to avoid duplicates
5. **Change Detection**: Monitor roots for availability changes

## Examples

### Multi-Repository Setup

```go
roots := []roots.Root{
    {
        URI:         "file:///repos/frontend",
        Name:        "Frontend",
        Description: "React application",
    },
    {
        URI:         "file:///repos/backend",
        Name:        "Backend",
        Description: "Go API server",
    },
    {
        URI:         "file:///repos/shared",
        Name:        "Shared Libraries",
        Description: "Common utilities",
    },
}
```

### Cloud Storage Integration

```go
roots := []roots.Root{
    {
        URI:         "s3://data-bucket/raw",
        Name:        "Raw Data",
        Description: "Unprocessed data files",
    },
    {
        URI:         "s3://data-bucket/processed",
        Name:        "Processed Data",
        Description: "Analysis results",
    },
}
```

### Development Environment

```go
roots := []roots.Root{
    {
        URI:         "file://" + os.Getenv("GOPATH"),
        Name:        "Go Workspace",
        Description: "Go development workspace",
    },
    {
        URI:         "file://" + os.Getenv("HOME") + "/.config",
        Name:        "Configuration",
        Description: "User configuration files",
    },
}