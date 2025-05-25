# Claude Context - Hot Reload Development Setup

## Current Status
- ✅ Hot reload development environment is set up and working
- ✅ Server runs in HTTP mode on port 9080 with mcp-proxy.js as stdio-to-HTTP proxy
- ✅ Air watches for changes and auto-reloads (~2-3 seconds)
- ✅ Using your global OPENAI_API_KEY (not the .env placeholder)
- ✅ Container name fixed: Changed from `mcp-memory-server-dev` to `mcp-memory-server` to match Claude CLI expectations

## Bug Fixes Applied
1. **Fixed panic in `internal/storage/chroma.go:185`** by adding bounds checking:
```go
// Ensure all result groups have data
if len(metadatas) == 0 || len(distances) == 0 || len(ids) == 0 {
    return results
}
```

2. **Fixed MCP connection error** - Container name mismatch resolved:
   - Claude CLI expects: `docker exec -i mcp-memory-server node /app/mcp-proxy.js`
   - Dev container was named `mcp-memory-server-dev` causing connection failures
   - Solution: Updated `docker-compose.dev.yml` to use `mcp-memory-server` as container name

## Development Commands
- `make dev-up` - Start development mode with hot reload
- `make dev-logs` - View real-time logs
- `make dev-down` - Stop development mode

## Quick Test
After reconnecting, test the fixed tools:
```bash
# Should work now without socket hang up
mcp__memory__memory_search("test query")
mcp__memory__memory_suggest_related("current context")
```

## Architecture Reminder
- Container runs Air → Go server (HTTP mode on :9080)
- mcp-proxy.js handles stdio↔HTTP translation
- Claude connects via stdio to the proxy
- Hot reload means no disconnections when code changes!

The fix is already applied and the server is running. Just reconnect and test! 🚀