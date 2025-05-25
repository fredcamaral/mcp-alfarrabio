# Hot Reload Development Setup 🔥

This setup allows you to develop with automatic hot-reloading - the server automatically restarts when you change any Go files, maintaining your MCP connection!

## Quick Start

```bash
# Start development mode with hot reload
make dev-up

# View logs (you'll see Air's reload messages)
make dev-logs

# Stop development mode
make dev-down
```

## How It Works

1. **Air** - Go live-reload tool watches for file changes
2. **Volume Mounts** - Your local code is mounted into the container
3. **Same Port** - Server always runs on port 9080, so MCP connection stays stable
4. **Automatic Rebuilds** - Air rebuilds and restarts the server on file changes

## What Gets Watched

- `cmd/` - Command files
- `internal/` - Internal packages
- `pkg/` - Public packages
- `*.go` files

## Development Workflow

1. Start development mode: `make dev-up`
2. Connect Claude to the MCP server (it will stay connected!)
3. Edit any Go file
4. Air automatically detects changes and rebuilds
5. Server restarts with your changes
6. MCP connection remains active! 🎉

## Useful Commands

```bash
# View real-time logs
make dev-logs

# Open shell in container
make dev-shell

# Manual restart (if needed)
make dev-restart

# Stop everything
make dev-down
```

## Notes

- First build might take a bit longer as Go downloads dependencies
- Air waits 1 second after detecting changes before rebuilding (configurable in `.air.toml`)
- The container runs as non-root user for security
- Data persists in Docker volumes between restarts