# MCP-Go Integration Guide

This guide covers how to integrate MCP-Go with ANY MCP-compatible client and various scenarios.

## Universal Compatibility

MCP-Go is designed to work with ANY client that implements the Model Context Protocol. The server automatically detects client capabilities and adapts its behavior accordingly. Whether you're using Claude, VS Code, Cursor, or building your own MCP client, MCP-Go will provide the best possible experience.

## Table of Contents
1. [Client Integration](#client-integration)
2. [LLM Integration](#llm-integration)
3. [File System Integration](#file-system-integration)
4. [Plugin Development](#plugin-development)
5. [Production Deployment](#production-deployment)

## Client Integration

### Universal Server Setup

For any MCP client, start with the FullFeaturedServer which includes all capabilities:

```go
func main() {
    // This server works with ANY MCP client
    srv := server.NewFullFeaturedServer("Universal MCP Server", "1.0.0")
    
    // The server will automatically adapt based on client capabilities
    // Add your tools, resources, and prompts
    setupTools(srv)
    setupResources(srv)
    setupPrompts(srv)
    
    // Choose transport based on client needs
    if os.Getenv("MCP_TRANSPORT") == "stdio" {
        srv.SetTransport(transport.NewStdioTransport())
    } else {
        srv.SetTransport(transport.NewHTTPTransport(&transport.HTTPConfig{
            Address: ":8080",
            EnableCORS: true,
        }))
    }
    
    srv.Start(context.Background())
}
```

### Specific Client Examples

### Claude Desktop

Claude Desktop supports tools, resources, and prompts but not advanced features.

**Configuration** (`claude_desktop_config.json`):
```json
{
  "mcpServers": {
    "my-server": {
      "command": "path/to/your/mcp-server",
      "args": [],
      "env": {
        "MCP_TRANSPORT": "stdio"
      }
    }
  }
}
```

**Server Setup**:
```go
func main() {
    srv := server.NewServer("My Server", "1.0.0")
    
    // Add tools, resources, prompts
    setupTools(srv)
    setupResources(srv)
    setupPrompts(srv)
    
    // Must use stdio transport for Claude Desktop
    srv.SetTransport(transport.NewStdioTransport())
    srv.Start(context.Background())
}
```

### VS Code GitHub Copilot

VS Code Copilot requires roots and supports discovery.

**Extension Configuration**:
```json
{
  "github.copilot.mcp.servers": [
    {
      "name": "workspace-server",
      "command": "mcp-server",
      "args": ["--port", "8080"],
      "roots": true,
      "discovery": true
    }
  ]
}
```

**Server Setup**:
```go
func main() {
    srv := server.NewExtendedServer("VS Code Server", "1.0.0")
    
    // Add workspace roots (REQUIRED for VS Code)
    srv.AddRoot(roots.Root{
        URI:  "file://" + os.Getenv("WORKSPACE_FOLDER"),
        Name: "Workspace",
    })
    
    // Enable discovery for dynamic tools
    if discovery, err := discovery.NewServiceWithPluginPath(
        filepath.Join(os.Getenv("WORKSPACE_FOLDER"), ".mcp/plugins"),
        30 * time.Second,
    ); err == nil {
        srv.SetDiscoveryService(discovery)
    }
    
    // HTTP transport for VS Code
    srv.SetTransport(transport.NewHTTPTransport(&transport.HTTPConfig{
        Address: ":8080",
    }))
    
    srv.Start(context.Background())
}
```

### Continue

Continue supports basic features with no discovery.

**Configuration** (`.continuerc.json`):
```json
{
  "models": [...],
  "mcpServers": [
    {
      "name": "my-tools",
      "url": "http://localhost:3000/mcp",
      "apiKey": "optional-api-key"
    }
  ]
}
```

### Cursor

Cursor only supports tools, requiring workarounds for other features.

**Server Adaptation**:
```go
func adaptForCursor(srv *server.Server) {
    // Convert resources to tools
    for _, resource := range srv.GetResources() {
        srv.AddTool(
            protocol.Tool{
                Name:        "read_" + sanitizeName(resource.Name),
                Description: "Read " + resource.Description,
                InputSchema: map[string]interface{}{
                    "type": "object",
                    "properties": map[string]interface{}{},
                },
            },
            &ResourceAsToolHandler{Resource: resource},
        )
    }
    
    // Convert prompts to tools
    for _, prompt := range srv.GetPrompts() {
        srv.AddTool(promptToTool(prompt))
    }
}
```

## LLM Integration

### OpenAI Integration

```go
package sampling

import (
    "github.com/sashabaranov/go-openai"
)

type OpenAIHandler struct {
    client *openai.Client
}

func NewOpenAIHandler(apiKey string) *OpenAIHandler {
    return &OpenAIHandler{
        client: openai.NewClient(apiKey),
    }
}

func (h *OpenAIHandler) CreateMessage(ctx context.Context, params json.RawMessage) (interface{}, error) {
    var req CreateMessageRequest
    json.Unmarshal(params, &req)
    
    // Convert messages
    messages := make([]openai.ChatCompletionMessage, len(req.Messages))
    for i, msg := range req.Messages {
        messages[i] = openai.ChatCompletionMessage{
            Role:    msg.Role,
            Content: msg.Content.Text,
        }
    }
    
    // Add system prompt if provided
    if req.SystemPrompt != nil {
        messages = append([]openai.ChatCompletionMessage{{
            Role:    openai.ChatMessageRoleSystem,
            Content: *req.SystemPrompt,
        }}, messages...)
    }
    
    // Call OpenAI
    resp, err := h.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
        Model:       h.selectModel(req.ModelPreferences),
        Messages:    messages,
        MaxTokens:   req.MaxTokens,
        Temperature: float32(getOrDefault(req.Temperature, 0.7)),
        Stop:        req.StopSequences,
    })
    
    if err != nil {
        return nil, err
    }
    
    choice := resp.Choices[0]
    return CreateMessageResponse{
        Role: "assistant",
        Content: SamplingMessageContent{
            Type: "text",
            Text: choice.Message.Content,
        },
        Model:      resp.Model,
        StopReason: string(choice.FinishReason),
    }, nil
}

func (h *OpenAIHandler) selectModel(prefs *ModelPreferences) string {
    if prefs == nil {
        return openai.GPT4
    }
    
    // Balance intelligence vs speed vs cost
    score := prefs.IntelligencePriority*3 + prefs.SpeedPriority*-1 + prefs.CostPriority*-2
    
    if score > 2 {
        return openai.GPT4 // Most intelligent
    } else if score < -1 {
        return openai.GPT3Dot5Turbo // Fastest/cheapest
    }
    return openai.GPT4Turbo // Balanced
}
```

### Anthropic Integration

```go
type AnthropicHandler struct {
    apiKey string
}

func (h *AnthropicHandler) CreateMessage(ctx context.Context, params json.RawMessage) (interface{}, error) {
    // Implementation similar to OpenAI
    // Use Anthropic's SDK or HTTP API
}
```

### Local LLM Integration (Ollama)

```go
type OllamaHandler struct {
    baseURL string
    model   string
}

func NewOllamaHandler(model string) *OllamaHandler {
    return &OllamaHandler{
        baseURL: "http://localhost:11434",
        model:   model,
    }
}

func (h *OllamaHandler) CreateMessage(ctx context.Context, params json.RawMessage) (interface{}, error) {
    var req CreateMessageRequest
    json.Unmarshal(params, &req)
    
    // Call Ollama API
    resp, err := h.callOllama(ctx, map[string]interface{}{
        "model":  h.model,
        "prompt": req.Messages[len(req.Messages)-1].Content.Text,
        "system": req.SystemPrompt,
        "options": map[string]interface{}{
            "num_predict": req.MaxTokens,
            "temperature": req.Temperature,
        },
    })
    
    if err != nil {
        return nil, err
    }
    
    return CreateMessageResponse{
        Role: "assistant",
        Content: SamplingMessageContent{
            Type: "text",
            Text: resp.Response,
        },
        Model: h.model,
    }, nil
}
```

## File System Integration

### Secure File Access

```go
type SecureFileHandler struct {
    roots []roots.Root
}

func (h *SecureFileHandler) Handle(ctx context.Context, uri string) ([]protocol.Content, error) {
    // Validate URI is within allowed roots
    if !h.isWithinRoots(uri) {
        return nil, fmt.Errorf("access denied: URI outside allowed roots")
    }
    
    // Parse URI
    parsed, _ := url.Parse(uri)
    if parsed.Scheme != "file" {
        return nil, fmt.Errorf("only file:// URIs supported")
    }
    
    // Read file with size limit
    content, err := h.readFileWithLimit(parsed.Path, 10*1024*1024) // 10MB limit
    if err != nil {
        return nil, err
    }
    
    return []protocol.Content{{
        Type: "text",
        Text: string(content),
    }}, nil
}

func (h *SecureFileHandler) isWithinRoots(uri string) bool {
    parsed, _ := url.Parse(uri)
    path := parsed.Path
    
    for _, root := range h.roots {
        rootParsed, _ := url.Parse(root.URI)
        if strings.HasPrefix(path, rootParsed.Path) {
            return true
        }
    }
    return false
}
```

### Virtual File Systems

```go
type VirtualFileSystem struct {
    providers map[string]FileProvider
}

type FileProvider interface {
    Read(path string) ([]byte, error)
    List(path string) ([]FileInfo, error)
}

// S3 Provider
type S3Provider struct {
    bucket string
    client *s3.Client
}

func (p *S3Provider) Read(path string) ([]byte, error) {
    result, err := p.client.GetObject(context.Background(), &s3.GetObjectInput{
        Bucket: &p.bucket,
        Key:    &path,
    })
    if err != nil {
        return nil, err
    }
    return io.ReadAll(result.Body)
}

// Git Provider
type GitProvider struct {
    repo *git.Repository
}

func (p *GitProvider) Read(path string) ([]byte, error) {
    // Read from git repository
}
```

## Plugin Development

### Basic Plugin Structure

```
my-plugin/
├── mcp-manifest.json
├── plugin.go
├── handlers/
│   ├── tools.go
│   └── resources.go
└── README.md
```

### Plugin Manifest

```json
{
    "name": "code-analyzer",
    "version": "1.0.0",
    "description": "Static code analysis tools",
    "author": "Your Name",
    "homepage": "https://github.com/you/code-analyzer",
    "tags": ["development", "analysis", "quality"],
    "tools": [
        {
            "name": "analyze_code",
            "description": "Analyze code for issues",
            "inputSchema": {
                "type": "object",
                "properties": {
                    "language": {
                        "type": "string",
                        "enum": ["go", "python", "javascript"]
                    },
                    "code": {
                        "type": "string",
                        "description": "Code to analyze"
                    },
                    "rules": {
                        "type": "array",
                        "items": {"type": "string"},
                        "description": "Analysis rules to apply"
                    }
                },
                "required": ["language", "code"]
            }
        }
    ],
    "config": {
        "defaultRules": ["security", "performance", "style"],
        "maxCodeSize": 1048576
    }
}
```

### Plugin Implementation

```go
// plugin.go
package main

import (
    "github.com/yourusername/mcp-go/plugin"
)

// Exported function for plugin system
func GetHandlers() map[string]plugin.Handler {
    return map[string]plugin.Handler{
        "analyze_code": &CodeAnalyzer{},
    }
}

type CodeAnalyzer struct{}

func (a *CodeAnalyzer) Handle(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    language := params["language"].(string)
    code := params["code"].(string)
    rules := getStringArray(params["rules"])
    
    results := analyzeCode(language, code, rules)
    
    return map[string]interface{}{
        "issues":    results.Issues,
        "metrics":   results.Metrics,
        "summary":   results.Summary,
    }, nil
}
```

## Production Deployment

### Docker Deployment

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o mcp-server ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/mcp-server .
COPY --from=builder /app/configs ./configs

EXPOSE 8080
CMD ["./mcp-server"]
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mcp-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: mcp-server
  template:
    metadata:
      labels:
        app: mcp-server
    spec:
      containers:
      - name: mcp-server
        image: your-registry/mcp-server:latest
        ports:
        - containerPort: 8080
        env:
        - name: MCP_PORT
          value: "8080"
        - name: MCP_TRANSPORT
          value: "http"
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: mcp-server
spec:
  selector:
    app: mcp-server
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

### Monitoring Setup

```go
// Enable Prometheus metrics
srv.EnableMetrics()

// Custom metrics
var (
    toolCallDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "mcp_tool_call_duration_seconds",
            Help: "Tool call duration in seconds",
        },
        []string{"tool"},
    )
)

// Track metrics
func trackToolCall(tool string, duration time.Duration) {
    toolCallDuration.WithLabelValues(tool).Observe(duration.Seconds())
}
```

### Security Best Practices

1. **Authentication**:
```go
srv.Use(middleware.Auth(func(token string) (User, error) {
    // Validate JWT or API key
    return validateToken(token)
}))
```

2. **Rate Limiting**:
```go
srv.Use(middleware.RateLimit(
    100,              // requests
    time.Minute,      // per minute
    middleware.ByIP,  // rate limit by IP
))
```

3. **Input Validation**:
```go
srv.Use(middleware.ValidateInput(schemas))
```

4. **Audit Logging**:
```go
srv.Use(middleware.AuditLog(auditLogger))
```