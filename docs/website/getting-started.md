# Getting Started with MCP Memory

This guide will help you install, configure, and start using MCP Memory with your AI assistant.

## Prerequisites

- Go 1.21 or later
- Docker and Docker Compose (for ChromaDB)
- An AI assistant that supports MCP (Claude Desktop, VS Code with Continue, etc.)

## Installation

### Option 1: Using Pre-built Binaries

Download the latest release for your platform:

```bash
# macOS
curl -L https://github.com/your-org/mcp-memory/releases/latest/download/mcp-memory-darwin-amd64 -o mcp-memory
chmod +x mcp-memory

# Linux
curl -L https://github.com/your-org/mcp-memory/releases/latest/download/mcp-memory-linux-amd64 -o mcp-memory
chmod +x mcp-memory

# Windows
# Download mcp-memory-windows-amd64.exe from releases page
```

### Option 2: Building from Source

```bash
git clone https://github.com/your-org/mcp-memory.git
cd mcp-memory
make build
```

## Quick Start

### 1. Start the Services

```bash
# Start ChromaDB with persistence
docker run -d -p 9000:8000 \
  -v chroma_data:/chroma/chroma \
  chromadb/chroma:latest \
  run --path /chroma/chroma --host 0.0.0.0 --port 8000

# Start GraphQL server and Web UI
export MCP_MEMORY_CHROMA_ENDPOINT=http://localhost:9000
./graphql

# Or with Docker Compose (recommended)
docker-compose up -d
```

### 2. Access the Web UI

Open your browser and navigate to:
- **Web UI**: http://localhost:8082/
- **GraphQL Playground**: http://localhost:8082/graphql

You can now browse memories, search, and explore relationships visually.

### 2. Configure Your AI Assistant

#### For Claude Desktop

Add to your Claude Desktop configuration (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "memory": {
      "command": "/path/to/mcp-memory",
      "args": ["serve", "--stdio"],
      "env": {
        "OPENAI_API_KEY": "your-api-key"
      }
    }
  }
}
```

#### For VS Code with Continue

Add to your Continue configuration:

```json
{
  "models": [
    {
      "title": "Claude with Memory",
      "provider": "anthropic",
      "model": "claude-3-opus-20240229",
      "mcpServers": {
        "memory": {
          "command": "/path/to/mcp-memory",
          "args": ["serve", "--stdio"]
        }
      }
    }
  ]
}
```

### 3. Verify Installation

Once configured, your AI assistant should have access to memory tools. Try these commands:

```
- "Check memory system health"
- "Store this conversation about setting up the project"
- "What have we discussed about authentication?"
```

## Configuration

### Basic Configuration

Create a configuration file at `~/.mcp-memory/config.yaml`:

```yaml
server:
  mode: stdio  # or http
  host: localhost
  port: 8080

storage:
  type: chroma
  chroma:
    url: http://localhost:8000
    collection: mcp_memory

embeddings:
  provider: openai
  model: text-embedding-3-small
  dimension: 1536

security:
  encryption:
    enabled: false  # Enable for production
    key_file: ~/.mcp-memory/encryption.key
```

### Environment Variables

You can override configuration with environment variables:

```bash
export MCP_MEMORY_STORAGE_TYPE=chroma
export MCP_MEMORY_CHROMA_URL=http://localhost:8000
export OPENAI_API_KEY=your-api-key
```

### Advanced Configuration

For production deployments, see our [deployment guide](../DEPLOYMENT.md) for:

- High availability setup
- Security hardening
- Performance tuning
- Monitoring and observability

## Basic Usage

### Storing Context

When working on a feature or solving a problem, store the context:

```
"Store this conversation about implementing user authentication with JWT tokens"
```

### Searching Past Context

Find relevant past discussions:

```
"Search for previous discussions about authentication"
"Find similar errors to 'connection refused'"
```

### Getting Suggestions

Get AI-powered suggestions based on your history:

```
"Suggest related context for the current database migration"
"What patterns have we used for API error handling?"
```

## Best Practices

1. **Regular Storage**: Store important conversations and decisions as you work
2. **Descriptive Context**: Include relevant details when storing memories
3. **Tag Appropriately**: Use tags to categorize different types of knowledge
4. **Review Patterns**: Periodically check identified patterns to improve your workflow

## Troubleshooting

### Common Issues

**ChromaDB Connection Failed**
```bash
# Check if ChromaDB is running
docker ps | grep chroma

# Restart ChromaDB
docker-compose restart chroma
```

**OpenAI API Key Missing**
```bash
# Set the API key
export OPENAI_API_KEY=your-api-key

# Or add to your configuration file
```

**Memory Search Returns No Results**
- Ensure conversations have been stored
- Check if embeddings are being generated correctly
- Verify ChromaDB is accessible

### Getting Help

- Check our [FAQ](faq.md) for common questions
- Join our [Discord server](https://discord.gg/mcp-memory) for community support
- Report issues on [GitHub](https://github.com/your-org/mcp-memory/issues)

## Next Steps

- Explore [API Reference](api-reference.md) for all available tools
- Check out [Examples](examples.md) for real-world use cases
- Learn about [advanced features](api-reference.md#advanced-features) like pattern recognition

---

Ready to enhance your development workflow? Start using MCP Memory today! 🚀