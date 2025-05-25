# IntelliJ IDEA Plugin for MCP Go

## Overview

The MCP Go plugin for IntelliJ IDEA and GoLand provides advanced IDE support for developing Model Context Protocol servers in Go.

## Features

### 1. Project Templates

#### New Project Wizard
- **MCP Server Project**: Complete server setup with examples
- **MCP Tool Library**: Reusable tool implementations
- **MCP Client Application**: Client-side MCP integration

#### Project Structure
```
mcp-project/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── handlers/
│   │   └── tools.go
│   └── config/
│       └── config.go
├── go.mod
├── go.sum
└── README.md
```

### 2. Code Generation

#### Live Templates

**Tool Handler** (`mcptool`):
```go
func (h *$HANDLER$) $NAME$(args json.RawMessage) (interface{}, error) {
    var params struct {
        $FIELD$ $TYPE$ `json:"$JSON_FIELD$"`
    }
    if err := json.Unmarshal(args, &params); err != nil {
        return nil, fmt.Errorf("invalid arguments: %w", err)
    }
    
    $END$
    
    return nil, nil
}
```

**Resource Handler** (`mcpresource`):
```go
func (h *$HANDLER$) $NAME$() (*protocol.Resource, error) {
    return &protocol.Resource{
        URI:      "$URI$",
        Name:     "$NAME$",
        MimeType: "$MIME_TYPE$",
        Contents: $CONTENTS$,
    }, nil
}
```

### 3. Code Inspections

#### MCP-Specific Inspections
- Missing error handling in tool handlers
- Unregistered tools in capability list
- Invalid JSON tags in parameter structs
- Missing required MCP interface methods
- Protocol version compatibility issues

#### Quick Fixes
- "Implement MCP Handler interface"
- "Register tool in server capabilities"
- "Add JSON tags to struct fields"
- "Generate error response"

### 4. Refactoring Support

#### MCP Refactorings
- **Extract Tool**: Extract method to standalone MCP tool
- **Inline Tool**: Inline simple tool implementation
- **Convert to Resource**: Transform data provider to MCP resource
- **Generate Schema**: Create JSON schema from Go struct

### 5. Debugging

#### Debug Configurations
```xml
<configuration name="Debug MCP Server" type="GoApplicationRunConfiguration">
  <module name="mcp-project" />
  <working_directory value="$PROJECT_DIR$" />
  <kind value="PACKAGE" />
  <package value="cmd/server" />
  <method v="2" />
  <envs>
    <env name="MCP_DEBUG" value="true" />
    <env name="MCP_LOG_LEVEL" value="debug" />
  </envs>
</configuration>
```

#### Protocol Debugger
- View MCP message flow in real-time
- Set breakpoints on specific message types
- Inspect message payloads
- Validate protocol compliance

### 6. Testing Integration

#### Test Generation
- Generate test cases for tool handlers
- Create protocol compliance tests
- Mock MCP client for testing

#### Test Runner Integration
```go
// Generated test template
func TestToolHandler_$NAME$(t *testing.T) {
    handler := &ToolHandler{}
    
    tests := []struct {
        name    string
        args    json.RawMessage
        want    interface{}
        wantErr bool
    }{
        {
            name: "valid input",
            args: json.RawMessage(`{"field": "value"}`),
            want: expectedResult,
            wantErr: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := handler.$NAME$(tt.args)
            if (err != nil) != tt.wantErr {
                t.Errorf("$NAME$() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("$NAME$() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### 7. Documentation

#### Quick Documentation (Ctrl+Q)
- Show MCP interface documentation
- Display protocol message formats
- View capability descriptions

#### External Documentation
- Direct links to MCP specification
- Context-aware help for MCP concepts

## Installation

### From JetBrains Marketplace

1. Open IntelliJ IDEA / GoLand
2. Go to `Settings` → `Plugins`
3. Search for "MCP Go"
4. Click `Install`
5. Restart IDE

### From Disk

1. Download the plugin from [releases](https://github.com/fredcamaral/mcp-memory/releases)
2. Go to `Settings` → `Plugins`
3. Click gear icon → `Install Plugin from Disk`
4. Select the downloaded `.zip` file
5. Restart IDE

## Configuration

### Plugin Settings

```
Settings → Tools → MCP Go
├── General
│   ├── ☑ Enable code inspections
│   ├── ☑ Enable live templates
│   └── ☑ Show protocol hints
├── Server
│   ├── Validator path: /usr/local/bin/mcp-validator
│   └── Default timeout: 30s
└── Development
    ├── ☑ Enable debug mode
    └── ☑ Show protocol messages
```

### Project Structure

```
Project Structure → Modules → MCP
├── Server Type: [stdio|http|websocket]
├── Capabilities
│   ├── ☑ Tools
│   ├── ☑ Resources
│   └── ☐ Prompts
└── Protocol Version: 1.0
```

## Usage Examples

### Creating a New MCP Server

1. `File` → `New` → `Project`
2. Select "MCP Server" from Go templates
3. Configure project settings
4. Choose capabilities and tools
5. Click `Create`

### Adding a New Tool

1. Right-click on handlers package
2. Select `New` → `MCP Tool`
3. Enter tool name and parameters
4. Implement generated method

### Running with MCP Client

1. Create Run Configuration
2. Set program arguments
3. Configure environment
4. Click `Run` or `Debug`

## Advanced Features

### 1. Protocol Analyzer

View → Tool Windows → MCP Protocol
- Real-time message inspection
- Performance metrics
- Error tracking

### 2. Schema Validation

- Automatic validation of tool schemas
- JSON Schema generation from Go types
- Compatibility checking

### 3. Performance Profiling

- CPU and memory profiling for MCP handlers
- Message processing metrics
- Bottleneck identification

## Troubleshooting

### Common Issues

1. **Plugin not loading**
   - Verify IDE version compatibility
   - Check plugin logs in `Help` → `Show Log`
   - Ensure Go plugin is installed

2. **Code generation not working**
   - Check project SDK configuration
   - Verify GOPATH/module settings
   - Restart IDE

3. **Debugging issues**
   - Update Delve debugger
   - Check firewall settings
   - Verify debug configuration

### Getting Help

- [Plugin Documentation](https://github.com/fredcamaral/mcp-memory/wiki/intellij-plugin)
- [Issue Tracker](https://github.com/fredcamaral/mcp-memory/issues)
- [Community Forum](https://discuss.mcp.dev)

## Development

### Building from Source

```bash
git clone https://github.com/fredcamaral/mcp-memory
cd mcp-memory/tools/intellij-plugin
./gradlew buildPlugin
```

### Running in Development

```bash
./gradlew runIde
```

## Contributing

We welcome contributions! See [CONTRIBUTING.md](https://github.com/fredcamaral/mcp-memory/blob/main/CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](https://github.com/fredcamaral/mcp-memory/blob/main/LICENSE) for details.