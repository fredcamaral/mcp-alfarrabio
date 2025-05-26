# Full-Featured MCP Server Example

This example demonstrates all MCP protocol features including:
- Tools, Resources, and Prompts (standard features)
- Sampling (LLM integration)
- Roots (file system access points)
- Discovery (dynamic registration)
- Subscriptions & Notifications
- Client Compatibility

## Running the Server

### Basic HTTP Server

```bash
# Run with HTTP transport on port 3000
go run main.go

# Or specify a custom port
MCP_PORT=8080 go run main.go
```

### With stdio Transport

```bash
# Run with stdio transport (for desktop clients)
MCP_TRANSPORT=stdio go run main.go
```

### With Plugin Discovery

```bash
# Enable plugin discovery
MCP_PLUGIN_PATH=/path/to/plugins go run main.go
```

## Testing Features

### Run Automated Tests

```bash
# Run the simple demo (tests all features directly)
go run simple_demo.go

# Run the test script (requires server running)
./test_features.sh
```

### Manual Testing with curl

```bash
# Initialize
curl -X POST http://localhost:3000/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {
      "protocolVersion": "2024-11-05",
      "clientInfo": {
        "name": "test-client",
        "version": "1.0.0"
      },
      "capabilities": {}
    }
  }'

# List roots
curl -X POST http://localhost:3000/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "roots/list",
    "params": {}
  }'

# Test sampling
curl -X POST http://localhost:3000/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 3,
    "method": "sampling/createMessage",
    "params": {
      "messages": [{
        "role": "user",
        "content": {
          "type": "text",
          "text": "Hello, MCP!"
        }
      }],
      "maxTokens": 100
    }
  }'
```

## Features Demonstrated

### 1. Standard Features
- **Tools**: Echo tool that reflects input
- **Resources**: Demo resource with dynamic content
- **Prompts**: Greeting prompt with style options

### 2. Sampling (AI Integration)
- Mock LLM response handler
- Ready for integration with OpenAI, Anthropic, etc.
- Supports message history and model preferences

### 3. Roots (File System)
- Automatic discovery of home, working, and temp directories
- Custom workspace root
- Secure path validation

### 4. Discovery (Coming Soon)
- Plugin manifest support
- Hot-reloading capability
- Dynamic tool registration

### 5. Client Compatibility
- Adapts capabilities based on client
- Graceful degradation for unsupported features
- Helpful error messages

## Output Examples

### Initialize Response
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2024-11-05",
    "capabilities": {
      "tools": {},
      "resources": {},
      "prompts": {},
      "sampling": {},
      "roots": {}
    },
    "serverInfo": {
      "name": "MCP Full Featured Demo",
      "version": "1.0.0"
    }
  }
}
```

### Roots Response
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "roots": [
      {
        "uri": "file:///home/user",
        "name": "Home",
        "description": "User home directory"
      },
      {
        "uri": "file:///workspace/project",
        "name": "Working Directory",
        "description": "Current working directory"
      },
      {
        "uri": "demo://workspace",
        "name": "Demo Workspace",
        "description": "Virtual workspace for demo purposes"
      }
    ]
  }
}
```

### Sampling Response
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "role": "assistant",
    "content": {
      "type": "text",
      "text": "This is a mock response. Implement actual LLM integration here."
    },
    "model": "mock-model",
    "stopReason": "stop_sequence"
  }
}
```

## Extending the Example

### Add Real LLM Integration

```go
// Replace the mock handler with real implementation
srv.SetSamplingHandler(&OpenAISamplingHandler{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "gpt-4",
})
```

### Add More Tools

```go
srv.AddTool(
    protocol.Tool{
        Name:        "file_read",
        Description: "Read a file from allowed roots",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "path": map[string]interface{}{
                    "type":        "string",
                    "description": "File path to read",
                },
            },
            "required": []string{"path"},
        },
    },
    &FileReadHandler{roots: srv.GetRoots()},
)
```

### Enable Subscriptions

```go
// In full-featured server (not yet in example)
srv.EnableSubscriptions()

// Client can subscribe
// POST /rpc
{
  "method": "resources/subscribe",
  "params": {"uri": "demo://test.txt"}
}
```

## Troubleshooting

### Server won't start
- Check if port is already in use
- Ensure Go 1.21+ is installed
- Check for compilation errors

### Features not working
- Verify client supports the feature
- Check server logs for errors
- Use simple_demo.go to test directly

### Client compatibility issues
- Some clients don't support all features
- Check the compatibility matrix in main docs
- Use appropriate transport for your client