# File Browser MCP Server Example

A secure file browser MCP server that demonstrates advanced MCP-Go features including security, resources, and coordinated tools.

## Features

This example includes the following tools:

- **list_files** - List files in a directory with pattern matching
- **read_file** - Read file contents with security checks
- **search_files** - Search for text within files
- **file_info** - Get detailed file/directory information

And a resource:
- **file:///{path}** - Direct file access via URI

## What This Example Demonstrates

1. **Security Best Practices**:
   - Path traversal prevention
   - File size limits
   - Extension whitelisting
   - Forbidden path checking

2. **Resource Management**:
   - URI-based file access
   - MIME type detection
   - Content type handling

3. **Advanced Tool Features**:
   - Complex parameter schemas
   - Optional parameters with defaults
   - Recursive operations
   - Pattern matching

4. **Error Handling**:
   - Detailed error messages
   - Graceful degradation
   - Security-aware errors

## Running the Example

### Build and Run

```bash
go build -o file-browser
./file-browser
```

### Configuration

The server can be configured by modifying the `Config` struct in `main.go`:

```go
config := &Config{
    BasePath: ".",  // Base directory for browsing
    AllowedExts: []string{".txt", ".md", ".json", ...},
    ForbiddenPaths: []string{".git", "node_modules", ...},
}
```

### Test with Claude Desktop

1. Add to your `claude_desktop_config.json`:
```json
{
  "mcpServers": {
    "file-browser": {
      "command": "/path/to/file-browser"
    }
  }
}
```

2. Example interactions:
   - "List all markdown files in the current directory"
   - "Search for 'TODO' in all .go files"
   - "Read the contents of README.md"
   - "Get info about the src directory"

## Security Features

### Path Validation

All paths are validated to prevent directory traversal:

```go
func validatePath(config *Config, path string) (string, error) {
    // Clean and resolve path
    // Ensure within base directory
    // Check forbidden paths
}
```

### File Type Restrictions

Only whitelisted file extensions can be read:

```go
AllowedExts: []string{
    ".txt", ".md", ".json", ".yaml",
    ".go", ".js", ".py", // Source files
    ".html", ".css",     // Web files
}
```

### Size Limits

Files larger than 10MB cannot be read to prevent memory issues:

```go
const maxFileSize = 10 * 1024 * 1024 // 10MB
```

## Tool Examples

### List Files

```json
{
  "tool": "list_files",
  "parameters": {
    "path": "src",
    "pattern": "*.go",
    "recursive": true
  }
}
```

### Search Files

```json
{
  "tool": "search_files",
  "parameters": {
    "query": "TODO",
    "path": ".",
    "filePattern": "*.go",
    "caseSensitive": false,
    "maxResults": 50
  }
}
```

### File Resource

Access files directly via URI:
- `file:///README.md`
- `file:///src/main.go`
- `file:///docs/guide.md`

## Code Organization

```
file-browser/
├── README.md       # This file
├── main.go         # Server implementation
├── go.mod          # Module definition
└── test_files/     # Optional test directory
    ├── sample.txt
    └── data.json
```

## Extending the Example

Ideas for enhancement:

1. **Write Operations**: Add file creation/modification (with safety checks)
2. **Archive Support**: Handle zip/tar files
3. **Metadata**: Extract and display file metadata
4. **Watching**: Monitor files for changes
5. **Permissions**: More granular access control

## Best Practices Demonstrated

1. **Input Validation**: Every input is validated before use
2. **Error Context**: Errors include helpful context
3. **Resource Cleanup**: Proper handling of file operations
4. **Defensive Programming**: Assume all input is potentially malicious
5. **Clear Documentation**: Well-documented security decisions

## Troubleshooting

### Access Denied Errors

- Check that the path is within the base directory
- Ensure the path doesn't contain forbidden directories
- Verify file extension is allowed

### File Not Found

- Use relative paths from the base directory
- Check file exists with `file_info` tool first
- Ensure proper path separators for your OS

### Search Not Working

- Check file size limits (files > 10MB are skipped)
- Verify file extension is in allowed list
- Try case-insensitive search

## Learning Next

After this example, explore:

- [Database Query](../database/) - External service integration
- [API Gateway](../api-gateway/) - HTTP and authentication
- [AI Assistant](../ai-assistant/) - Complex multi-tool coordination

This example provides a solid foundation for building secure, file-based MCP tools!