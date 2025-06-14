# Lerian MCP Memory CLI (lmmc)

A powerful task management CLI that integrates with the Lerian MCP Memory Server for AI-powered features.

## Features

- **Local-First Design**: Works offline with local storage, syncs when connected
- **Repository-Aware**: Automatically detects and organizes tasks by Git repository
- **Flexible Output**: Supports table, JSON, and plain text formats
- **Smart Filtering**: Filter tasks by status, priority, tags, and more
- **MCP Integration**: Optional sync with Lerian MCP Memory Server
- **Fast & Lightweight**: Written in Go for optimal performance

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/lerianstudio/lerian-mcp-memory.git
cd lerian-mcp-memory/cli

# Build and install
make install
```

### Binary Releases

Download the latest release for your platform from the [releases page](https://github.com/lerianstudio/lerian-mcp-memory/releases).

## Quick Start

```bash
# Add a task
lmmc add "Fix authentication bug" --priority=high --tag=bug

# List all tasks
lmmc list

# Start working on a task
lmmc start <task-id>

# Mark task as done
lmmc done <task-id>

# Search tasks
lmmc search "authentication"

# Get statistics
lmmc stats
```

## Configuration

The CLI stores configuration in `~/.lmmc/config.yaml`. Environment variables with `LMMC_` prefix override file settings. You can manage settings using the config command:

```bash
# Set output format
lmmc config set cli.output_format json

# Get a configuration value
lmmc config get server.url

# Reset to defaults
lmmc config reset
```

### Environment Variables

All configuration options can be overridden with environment variables:

- `LMMC_SERVER_URL`: MCP server URL
- `LMMC_SERVER_TIMEOUT`: Server timeout in seconds
- `LMMC_CLI_OUTPUT_FORMAT`: Default output format (table, json, plain)
- `LMMC_LOGGING_LEVEL`: Log level (debug, info, warn, error)

### AI Provider Configuration

The CLI supports multiple AI providers for intelligent task suggestions and analysis. Configure using environment variables:

#### OpenAI
```bash
export AI_PROVIDER=openai
export OPENAI_API_KEY=your_openai_api_key_here
export OPENAI_MODEL=gpt-4o  # Optional: defaults to gpt-4o
```

#### Claude (Anthropic)
```bash
export AI_PROVIDER=claude
export CLAUDE_API_KEY=your_claude_api_key_here
export CLAUDE_MODEL=claude-sonnet-4  # Optional: defaults to claude-sonnet-4
```

#### Perplexity
```bash
export AI_PROVIDER=perplexity
export PERPLEXITY_API_KEY=your_perplexity_api_key_here
export PERPLEXITY_MODEL=sonar-pro  # Optional: defaults to sonar-pro
```

If no AI provider is configured, the CLI will fall back to mock mode for testing.

## Command Reference

### Task Management

- `add <content>`: Create a new task
- `list`: List tasks with optional filters
- `start <id>`: Mark task as in progress
- `done <id>`: Mark task as completed
- `cancel <id>`: Cancel a task
- `edit <id>`: Edit task details
- `priority <id> <level>`: Update task priority
- `delete <id>`: Delete a task
- `search <query>`: Search tasks by content

### Filtering Options

```bash
# Filter by status
lmmc list --status=pending

# Filter by priority
lmmc list --priority=high

# Filter by tags
lmmc list --tag=bug --tag=urgent

# Combine filters
lmmc list --status=in_progress --priority=high
```

### Output Formats

```bash
# Table format (default)
lmmc list

# JSON format
lmmc list --output=json

# Plain text format
lmmc list --output=plain
```

## MCP Integration

When configured with an MCP server, tasks are automatically synchronized:

```bash
# Configure MCP server
lmmc config set server.url http://localhost:9080

# Tasks will sync automatically
lmmc add "Task synced to MCP"
```

The CLI works seamlessly offline and syncs when the server is available.

## Development

### Building

```bash
# Build binary
make build

# Run tests
make test

# Run all tests including integration
make test-all

# Format code
make fmt

# Run linter
make lint
```

### Testing

```bash
# Unit tests
make test

# Integration tests
make test-integration

# End-to-end tests
make test-e2e

# Performance tests
make test-perf

# Test with race detector
make test-race
```

## Architecture

The CLI follows clean architecture principles:

```
cli/
├── cmd/                    # Application entry points
├── internal/
│   ├── domain/            # Business logic and entities
│   │   ├── entities/      # Core domain models
│   │   ├── ports/         # Interface definitions
│   │   └── services/      # Business logic
│   ├── adapters/
│   │   ├── primary/       # CLI interface
│   │   └── secondary/     # Storage, MCP, Config
│   └── di/                # Dependency injection
└── tests/                 # Integration and E2E tests
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.