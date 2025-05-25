# VS Code Extension for MCP Go

## Overview

The MCP Go VS Code extension provides comprehensive support for developing Model Context Protocol servers in Go.

## Features

### 1. Language Support

- **Syntax Highlighting**: Custom syntax highlighting for MCP-specific patterns
- **Code Snippets**: Pre-built snippets for common MCP patterns
- **IntelliSense**: Auto-completion for MCP types and methods
- **Go to Definition**: Navigate to MCP interface implementations

### 2. MCP Server Development

#### Quick Start Templates
```json
{
  "MCP Echo Server": {
    "prefix": "mcp-echo",
    "body": [
      "package main",
      "",
      "import (",
      "\t\"github.com/fredcamaral/mcp-memory/pkg/mcp\"",
      "\t\"github.com/fredcamaral/mcp-memory/pkg/mcp/server\"",
      "\t\"github.com/fredcamaral/mcp-memory/pkg/mcp/transport\"",
      ")",
      "",
      "func main() {",
      "\thandler := &EchoHandler{}",
      "\tsrv := server.NewServer(\"echo-server\", \"1.0.0\", handler)",
      "\ttransport.ServeStdio(srv)",
      "}"
    ]
  }
}
```

#### Tool Implementation Snippets
```json
{
  "MCP Tool Handler": {
    "prefix": "mcp-tool",
    "body": [
      "func (h *${1:Handler}) ${2:ToolName}(args json.RawMessage) (interface{}, error) {",
      "\tvar params struct {",
      "\t\t${3:Field} ${4:Type} \`json:\"${5:field}\"\`",
      "\t}",
      "\tif err := json.Unmarshal(args, &params); err != nil {",
      "\t\treturn nil, fmt.Errorf(\"invalid arguments: %w\", err)",
      "\t}",
      "\t",
      "\t${6:// Implementation}",
      "\t",
      "\treturn ${7:result}, nil",
      "}"
    ]
  }
}
```

### 3. Debugging Support

#### Launch Configuration
```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug MCP Server",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/server",
      "env": {
        "MCP_DEBUG": "true",
        "MCP_LOG_LEVEL": "debug"
      },
      "args": []
    },
    {
      "name": "Attach to MCP Server",
      "type": "go",
      "request": "attach",
      "mode": "remote",
      "remotePath": "${workspaceFolder}",
      "port": 2345,
      "host": "127.0.0.1"
    }
  ]
}
```

### 4. Testing Integration

#### Test Runner
- Run MCP protocol tests directly from VS Code
- Visualize test results in the Test Explorer
- Debug individual test cases

#### Test Configuration
```json
{
  "go.testFlags": [
    "-v",
    "-race",
    "-coverprofile=coverage.out"
  ],
  "go.testTags": "integration mcp",
  "go.testTimeout": "30s"
}
```

### 5. MCP Protocol Validation

- Real-time validation of MCP messages
- Schema validation for tool definitions
- Protocol compliance checking

### 6. Code Actions

#### Quick Fixes
- "Implement missing MCP methods"
- "Add tool to capability list"
- "Generate tool documentation"
- "Convert to MCP error format"

#### Refactoring
- "Extract tool to separate handler"
- "Convert function to MCP tool"
- "Generate tool schema from struct"

### 7. Integrated Terminal Commands

```bash
# Validate MCP server
mcp-validator ./cmd/server

# Run benchmarks
mcp-benchmark --server ./cmd/server --duration 60s

# Generate tool documentation
mcp-tools-doc ./cmd/server > tools.md
```

## Installation

### From VS Code Marketplace

1. Open VS Code
2. Press `Ctrl+P` / `Cmd+P`
3. Type `ext install mcp-go`
4. Click Install

### From VSIX

```bash
# Download the latest release
curl -L https://github.com/fredcamaral/mcp-memory/releases/latest/download/mcp-go.vsix -o mcp-go.vsix

# Install
code --install-extension mcp-go.vsix
```

## Configuration

### Extension Settings

```json
{
  "mcp-go.serverPath": "/usr/local/bin/mcp-server",
  "mcp-go.validatorPath": "/usr/local/bin/mcp-validator",
  "mcp-go.enableAutoComplete": true,
  "mcp-go.enableDiagnostics": true,
  "mcp-go.enableCodeLens": true,
  "mcp-go.trace.server": "verbose"
}
```

### Workspace Settings

```json
{
  "mcp-go.project": {
    "serverType": "stdio",
    "capabilities": {
      "tools": true,
      "resources": true,
      "prompts": false
    },
    "testMode": "integration"
  }
}
```

## Commands

| Command | Description | Keybinding |
|---------|-------------|------------|
| `MCP: Create New Server` | Generate new MCP server boilerplate | `Ctrl+Shift+M N` |
| `MCP: Add Tool` | Add a new tool to current server | `Ctrl+Shift+M T` |
| `MCP: Validate Server` | Run MCP validator on current file | `Ctrl+Shift+M V` |
| `MCP: Run Tests` | Execute MCP protocol tests | `Ctrl+Shift+M R` |
| `MCP: Show Protocol Docs` | Open MCP documentation | `Ctrl+Shift+M D` |

## Development

### Building the Extension

```bash
# Clone the repository
git clone https://github.com/fredcamaral/mcp-memory
cd mcp-memory/tools/vscode-extension

# Install dependencies
npm install

# Compile
npm run compile

# Package
vsce package
```

### Testing

```bash
# Run tests
npm test

# Run with coverage
npm run test:coverage
```

## Troubleshooting

### Common Issues

1. **Extension not activating**
   - Ensure you have a Go file open
   - Check VS Code developer console for errors
   - Verify Go extension is installed

2. **IntelliSense not working**
   - Run `Go: Restart Language Server`
   - Check `gopls` is installed and up to date
   - Verify `go.mod` includes MCP dependency

3. **Debugger not attaching**
   - Ensure server is built with debug symbols
   - Check firewall settings for debug port
   - Verify Delve debugger is installed

## Contributing

Contributions are welcome! Please see the [contribution guidelines](https://github.com/fredcamaral/mcp-memory/blob/main/CONTRIBUTING.md).

## License

MIT License - see [LICENSE](https://github.com/fredcamaral/mcp-memory/blob/main/LICENSE) for details.