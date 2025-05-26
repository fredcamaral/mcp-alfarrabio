# MCP SDK Migration Notice

## Important Update

The MCP Go implementation has been moved to a standalone open-source SDK:

🚀 **New Location**: [github.com/fredcamaral/gomcp-sdk](https://github.com/fredcamaral/gomcp-sdk)

## Why the Change?

1. **Community Benefit**: The MCP SDK is now available for anyone to build MCP-compatible applications
2. **Independent Development**: The SDK can evolve separately from the MCP Memory Server
3. **Better Modularity**: Cleaner separation of concerns
4. **Open Source**: Easier for the community to contribute and improve

## Migration Guide

If you were using the MCP implementation from this repository:

### 1. Update your imports

```go
// Old
import "mcp-memory/pkg/mcp/server"

// New
import "github.com/fredcamaral/gomcp-sdk/server"
```

### 2. Update go.mod

```bash
go get github.com/fredcamaral/gomcp-sdk
go mod tidy
```

### 3. No API Changes

The API remains exactly the same - only the import paths have changed.

## Features

The standalone SDK includes all features:
- ✅ Full MCP protocol support
- ✅ Universal client compatibility
- ✅ Sampling (LLM integration)
- ✅ Roots (file system access)
- ✅ Discovery (plugins)
- ✅ Subscriptions & Notifications
- ✅ Production-ready features

## Contributing

The SDK is open source and welcomes contributions! Visit the [GoMCP SDK repository](https://github.com/fredcamaral/gomcp-sdk) to:
- Report issues
- Submit pull requests
- Read documentation
- See examples

## MCP Memory Server

This project (mcp-memory) will be updated to use the standalone SDK as a dependency in a future release.