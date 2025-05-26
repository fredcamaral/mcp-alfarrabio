# Discovery Feature Documentation

## Overview

The Discovery feature enables dynamic registration and discovery of MCP tools, resources, and prompts. This allows for plugin systems, hot-reloading, and runtime extensibility.

## Architecture

```
discovery/
├── registry.go      # Central registry for all items
├── discovery.go     # Discovery service
├── watcher.go       # Plugin file watcher
└── types.go         # Discovery types
```

## Basic Usage

### Creating a Discovery Service

```go
import (
    "github.com/yourusername/mcp-go/discovery"
    "github.com/yourusername/mcp-go/protocol"
)

// Create discovery service
service := discovery.NewService()

// Register tools dynamically
err := service.RegisterTool(
    protocol.Tool{
        Name:        "dynamic_tool",
        Description: "A dynamically registered tool",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "input": map[string]interface{}{"type": "string"},
            },
        },
    },
    toolHandler,
    "my-plugin",     // source
    []string{"utility", "dynamic"}, // tags
)

// Start the service
service.Start(context.Background())
```

### Plugin Watcher

Enable automatic plugin discovery:

```go
// Create service with plugin directory watching
service, err := discovery.NewServiceWithPluginPath(
    "/path/to/plugins",
    30 * time.Second, // scan interval
)

if err != nil {
    log.Fatal(err)
}

// Plugins are automatically discovered and registered
service.Start(context.Background())
```

## Plugin Manifest Format

Create a `mcp-manifest.json` in your plugin directory:

```json
{
    "name": "my-awesome-plugin",
    "version": "1.0.0",
    "description": "Adds awesome functionality",
    "author": "Your Name",
    "tags": ["productivity", "automation"],
    "tools": [
        {
            "name": "awesome_tool",
            "description": "Does something awesome",
            "inputSchema": {
                "type": "object",
                "properties": {
                    "action": {
                        "type": "string",
                        "enum": ["create", "update", "delete"]
                    },
                    "target": {
                        "type": "string",
                        "description": "Target to operate on"
                    }
                },
                "required": ["action", "target"]
            }
        }
    ],
    "resources": [
        {
            "uri": "awesome://data",
            "name": "Awesome Data",
            "description": "Access to awesome data",
            "mimeType": "application/json"
        }
    ],
    "prompts": [
        {
            "name": "awesome_prompt",
            "description": "Generate awesome content",
            "arguments": [
                {
                    "name": "style",
                    "description": "Style of content",
                    "required": true
                }
            ]
        }
    ]
}
```

## Discovery API

### Discover with Filters

```go
// Discover all available items
info := service.GetRegistry().Discover(nil)

// Discover with filters
info := service.GetRegistry().Discover(&discovery.DiscoveryFilter{
    Tags:      []string{"productivity"},
    Available: &true,
    Source:    "my-plugin",
    Search:    "awesome",
})

// Access discovered items
fmt.Printf("Found %d tools\n", len(info.Tools))
fmt.Printf("Found %d resources\n", len(info.Resources))
fmt.Printf("Found %d prompts\n", len(info.Prompts))
```

### Request Format

```json
{
    "method": "discovery/discover",
    "params": {
        "filter": {
            "tags": ["productivity", "automation"],
            "available": true,
            "source": "my-plugin",
            "search": "data"
        }
    }
}
```

### Response Format

```json
{
    "tools": [
        {
            "tool": {
                "name": "awesome_tool",
                "description": "Does something awesome",
                "inputSchema": {...}
            },
            "available": true,
            "lastSeen": "2024-01-15T10:30:00Z",
            "source": "plugin:my-awesome-plugin",
            "tags": ["productivity", "automation"]
        }
    ],
    "resources": [...],
    "prompts": [...],
    "lastUpdate": "2024-01-15T10:30:00Z",
    "version": "1.0.0"
}
```

## Registration Events

Subscribe to registration events:

```go
// Subscribe to registry events
eventChan := service.Subscribe()

go func() {
    for event := range eventChan {
        switch event.Type {
        case "register":
            log.Printf("New %s registered: %s from %s",
                event.Category, event.Name, event.Item)
        case "unregister":
            log.Printf("%s unregistered: %s",
                event.Category, event.Name)
        }
    }
}()
```

## Hot Reloading

The plugin watcher supports hot reloading:

1. **Add Plugin**: Drop manifest in plugin directory → Automatically registered
2. **Update Plugin**: Modify manifest → Old version unregistered, new version registered
3. **Remove Plugin**: Delete manifest → Automatically unregistered

## Advanced Plugin Example

### Plugin with Dependencies

```json
{
    "name": "database-tools",
    "version": "2.0.0",
    "description": "Database management tools",
    "tags": ["database", "sql", "management"],
    "requirements": [
        "postgresql >= 13.0",
        "redis >= 6.0"
    ],
    "tools": [
        {
            "name": "db_query",
            "description": "Execute database queries",
            "inputSchema": {
                "type": "object",
                "properties": {
                    "database": {
                        "type": "string",
                        "enum": ["postgres", "mysql", "sqlite"]
                    },
                    "query": {
                        "type": "string",
                        "description": "SQL query to execute"
                    },
                    "params": {
                        "type": "array",
                        "items": {"type": "string"},
                        "description": "Query parameters"
                    }
                },
                "required": ["database", "query"]
            },
            "examples": [
                {
                    "description": "Select all users",
                    "arguments": {
                        "database": "postgres",
                        "query": "SELECT * FROM users WHERE active = $1",
                        "params": ["true"]
                    }
                }
            ]
        }
    ]
}
```

### Dynamic Plugin Loading

```go
type PluginLoader struct {
    registry *discovery.Registry
    plugins  map[string]*Plugin
}

func (l *PluginLoader) LoadPlugin(manifestPath string) error {
    // Read manifest
    data, err := ioutil.ReadFile(manifestPath)
    if err != nil {
        return err
    }
    
    var manifest PluginManifest
    if err := json.Unmarshal(data, &manifest); err != nil {
        return err
    }
    
    // Load plugin binary (if applicable)
    pluginPath := filepath.Join(filepath.Dir(manifestPath), manifest.Name)
    plugin, err := plugin.Open(pluginPath)
    if err != nil {
        return err
    }
    
    // Register tools with actual handlers
    for _, tool := range manifest.Tools {
        handlerSym, err := plugin.Lookup(tool.Name + "Handler")
        if err != nil {
            continue
        }
        
        handler, ok := handlerSym.(protocol.ToolHandler)
        if !ok {
            continue
        }
        
        l.registry.RegisterTool(tool, handler, manifest.Name, manifest.Tags)
    }
    
    return nil
}
```

## Security Considerations

### Plugin Validation

```go
func validatePlugin(manifest *PluginManifest) error {
    // Validate version
    if !semver.IsValid("v" + manifest.Version) {
        return fmt.Errorf("invalid version: %s", manifest.Version)
    }
    
    // Validate tool names
    for _, tool := range manifest.Tools {
        if !isValidIdentifier(tool.Name) {
            return fmt.Errorf("invalid tool name: %s", tool.Name)
        }
        
        // Validate input schema
        if err := validateJSONSchema(tool.InputSchema); err != nil {
            return fmt.Errorf("invalid schema for %s: %w", tool.Name, err)
        }
    }
    
    // Check for dangerous permissions
    if containsDangerousPermissions(manifest) {
        return fmt.Errorf("plugin requests dangerous permissions")
    }
    
    return nil
}
```

### Sandboxing

```go
// Run plugins in isolated environment
type SandboxedPlugin struct {
    plugin   *Plugin
    sandbox  *Sandbox
    limits   ResourceLimits
}

func (s *SandboxedPlugin) ExecuteTool(name string, params map[string]interface{}) (interface{}, error) {
    // Apply resource limits
    ctx := context.WithValue(context.Background(), "limits", s.limits)
    
    // Execute in sandbox
    return s.sandbox.Execute(ctx, func() (interface{}, error) {
        return s.plugin.Tools[name].Handle(ctx, params)
    })
}
```

## Best Practices

1. **Version Management**: Use semantic versioning for plugins
2. **Dependency Declaration**: Clearly declare all dependencies
3. **Error Handling**: Gracefully handle plugin failures
4. **Resource Limits**: Set limits on plugin resource usage
5. **Monitoring**: Track plugin performance and errors

## Client Support

| Client | Discovery Support | Notes |
|--------|------------------|-------|
| VS Code Copilot | ✅ Full | Dynamic tool discovery |
| Windsurf | ✅ Full | Tool discovery only |
| Apify MCP | ✅ Full | Remote server discovery |
| Claude Desktop | ❌ | Static registration only |