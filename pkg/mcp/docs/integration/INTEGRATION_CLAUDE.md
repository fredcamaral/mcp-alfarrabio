# Claude Integration Guide

This guide provides step-by-step instructions for integrating the MCP Memory Server with Claude Desktop and Claude CLI.

## Table of Contents
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Configuration](#configuration)
- [Usage Examples](#usage-examples)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)
- [Performance Tuning](#performance-tuning)

## Prerequisites

- Go 1.21 or higher
- Docker (optional, for containerized deployment)
- Claude Desktop or Claude CLI
- Chroma vector database running

## Installation

### Option 1: Build from Source

```bash
# Clone the repository
git clone https://github.com/yourusername/mcp-memory.git
cd mcp-memory

# Build the server
make build

# Run the server
./bin/server
```

### Option 2: Docker

```bash
# Pull the Docker image
docker pull your-registry/mcp-memory:latest

# Run with Docker Compose
docker-compose up -d
```

## Configuration

### 1. Claude Desktop Configuration

Add the MCP Memory Server to your Claude Desktop configuration:

```json
{
  "mcpServers": {
    "memory": {
      "command": "/path/to/mcp-memory/bin/server",
      "args": ["--config", "/path/to/config.yaml"],
      "env": {
        "OPENAI_API_KEY": "your-api-key",
        "MCP_MEMORY_ENV": "production"
      }
    }
  }
}
```

### 2. Environment Variables

Create a `.env` file or set these environment variables:

```bash
# Required
export OPENAI_API_KEY="your-openai-api-key"
export CHROMA_URL="http://localhost:8000"

# Optional
export MCP_MEMORY_ENV="production"
export MCP_MEMORY_LOG_LEVEL="info"
export MCP_MEMORY_PORT="8080"
```

### 3. Configuration File

Create a `config.yaml` file:

```yaml
environment: production
service:
  name: mcp-memory
  version: 1.0.0

server:
  mode: stdio  # For Claude Desktop
  port: 8080   # For HTTP mode
  timeout: 30s

chroma:
  url: "http://localhost:8000"
  collection: "memory_production"

openai:
  model: "text-embedding-3-small"
  max_retries: 3
  timeout: 30s

features:
  auto_analysis: true
  pattern_detection: true
  context_suggestions: true
  multi_repository: true

logging:
  level: info
  format: json
```

## Usage Examples

### Basic Memory Storage

```typescript
// Store a conversation chunk
await memory.store_chunk({
  content: "Implemented user authentication with JWT tokens",
  session_id: "dev-session-123",
  repository: "my-app",
  tags: ["auth", "security"],
  files_modified: ["auth/jwt.go", "middleware/auth.go"]
});
```

### Searching Memory

```typescript
// Search for similar past conversations
const results = await memory.search({
  query: "authentication implementation",
  repository: "my-app",
  limit: 5,
  min_relevance: 0.7
});
```

### Finding Similar Problems

```typescript
// Find solutions to similar past problems
const solutions = await memory.find_similar({
  problem: "JWT token validation failing",
  repository: "my-app",
  limit: 3
});
```

### Architectural Decisions

```typescript
// Store an architectural decision
await memory.store_decision({
  decision: "Use JWT for authentication",
  rationale: "Stateless, scalable, and industry standard",
  context: "Considered sessions but JWT better for microservices",
  repository: "my-app",
  session_id: "arch-review-001"
});
```

### Pattern Analysis

```typescript
// Get recurring patterns in development
const patterns = await memory.get_patterns({
  repository: "my-app",
  timeframe: "month"
});
```

### Context Suggestions

```typescript
// Get AI-powered suggestions for related context
const suggestions = await memory.suggest_related({
  current_context: "Working on API rate limiting",
  repository: "my-app",
  session_id: "current-session",
  include_patterns: true
});
```

## Best Practices

### 1. Session Management

- Always use consistent session IDs for related work
- Create new sessions for distinct tasks or features
- Include meaningful session prefixes (e.g., `feature-`, `bugfix-`, `refactor-`)

### 2. Content Organization

- Store chunks at logical boundaries (function implementations, bug fixes)
- Include relevant metadata (files, tools, tags)
- Keep chunk content focused and specific

### 3. Repository Naming

- Use consistent repository names across sessions
- Match repository names to actual project names
- Consider using hierarchical names for monorepos (e.g., `company/service`)

### 4. Tag Strategy

- Develop a consistent tagging taxonomy
- Use both technical tags (e.g., `database`, `api`) and business tags (e.g., `billing`, `user-management`)
- Include language/framework tags when relevant

### 5. Memory Hygiene

- Regularly export important project memories
- Archive old or irrelevant memories
- Use the pattern analysis to identify and clean up noise

## Troubleshooting

### Common Issues

#### 1. Connection to Chroma Failed

**Symptom**: "Failed to connect to Chroma" error

**Solution**:
```bash
# Check if Chroma is running
docker ps | grep chroma

# Restart Chroma if needed
docker-compose restart chroma

# Verify connectivity
curl http://localhost:8000/api/v1/heartbeat
```

#### 2. OpenAI API Errors

**Symptom**: "Failed to generate embeddings" error

**Solution**:
- Verify API key is correct
- Check API rate limits
- Ensure network connectivity
- Try using a different embedding model

#### 3. Memory Search Returns No Results

**Symptom**: Searches return empty results despite stored content

**Solution**:
- Check minimum relevance threshold (lower it if needed)
- Verify repository filter matches stored chunks
- Ensure embeddings were generated successfully
- Check Chroma logs for indexing issues

#### 4. High Memory Usage

**Symptom**: Server consuming excessive memory

**Solution**:
- Implement chunk size limits
- Configure connection pooling
- Enable memory profiling
- Set appropriate garbage collection parameters

### Debug Mode

Enable debug logging for detailed troubleshooting:

```yaml
logging:
  level: debug
  format: text
  output: stdout
```

Or via environment variable:
```bash
export MCP_MEMORY_LOG_LEVEL=debug
```

## Performance Tuning

### 1. Embedding Generation

```yaml
openai:
  model: "text-embedding-3-small"  # Faster, lower cost
  batch_size: 100                  # Process in batches
  max_concurrent: 5                # Parallel requests
```

### 2. Search Optimization

```yaml
search:
  max_results: 50          # Limit initial retrieval
  rerank_enabled: true     # Enable result reranking
  cache_ttl: 300s         # Cache frequent searches
```

### 3. Storage Optimization

```yaml
storage:
  compression: true        # Enable chunk compression
  deduplication: true      # Remove duplicate chunks
  retention_days: 90       # Auto-cleanup old data
```

### 4. Connection Pooling

```yaml
chroma:
  max_connections: 20
  connection_timeout: 5s
  idle_timeout: 300s
```

### 5. Resource Limits

When running in Docker:

```yaml
services:
  mcp-memory:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '1'
          memory: 1G
```

### 6. Monitoring

Enable Prometheus metrics:

```yaml
monitoring:
  enabled: true
  port: 9090
  path: /metrics
```

Key metrics to monitor:
- `mcp_memory_chunk_storage_duration` - Time to store chunks
- `mcp_memory_search_duration` - Search latency
- `mcp_memory_embedding_generation_duration` - Embedding generation time
- `mcp_memory_active_sessions` - Number of active sessions
- `mcp_memory_storage_size_bytes` - Total storage usage

## Advanced Features

### 1. Multi-Repository Support

```typescript
// Import context from multiple repositories
await memory.import_context({
  source: "archive",
  data: base64ArchiveData,
  repository: "legacy-project",
  session_id: "import-001",
  metadata: {
    source_system: "old-system",
    import_date: "2024-01-01"
  }
});
```

### 2. Custom Pattern Detection

Configure custom patterns in `config.yaml`:

```yaml
patterns:
  custom:
    - name: "error_handling"
      regex: "(?i)(error|exception|fail)"
      category: "reliability"
    - name: "performance"
      regex: "(?i)(optimize|slow|performance)"
      category: "optimization"
```

### 3. Workflow Integration

```yaml
workflows:
  auto_suggest:
    enabled: true
    threshold: 0.8
    max_suggestions: 5
  
  pattern_alerts:
    enabled: true
    min_occurrences: 3
```

## Security Considerations

1. **API Key Management**: Never commit API keys to version control
2. **Data Encryption**: Enable encryption for sensitive repositories
3. **Access Control**: Implement repository-level access controls
4. **Audit Logging**: Enable audit logs for compliance
5. **Network Security**: Use TLS for all external connections

## Conclusion

The MCP Memory Server provides powerful memory and context management for Claude. By following these guidelines, you can effectively integrate it into your development workflow and leverage its full potential for enhanced productivity and knowledge retention.

For more information, see:
- [API Reference](./README.md)
- [Architecture Overview](./ROADMAP.md)
- [Contributing Guide](../../CONTRIBUTING.md)