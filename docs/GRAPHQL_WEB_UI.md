# GraphQL API & Web UI Guide

This guide covers the GraphQL API server and modern web interface for the MCP Memory Server.

## Overview

The GraphQL server provides a modern, flexible API for interacting with the memory system, along with a web-based user interface for browsing and managing memories.

## Features

### Web UI
- **Memory Browser**: Browse all stored memories with search and filtering
- **Memory Details**: View full content, metadata, and relationships
- **Session Tracing**: Follow the timeline of memories within a session
- **Relationship Graphs**: Visualize connections between related memories
- **Real-time Updates**: See new memories as they're added

### GraphQL API
- **Flexible Queries**: Request exactly the data you need
- **Type Safety**: Strongly typed schema with validation
- **GraphiQL Playground**: Interactive API explorer
- **Batch Operations**: Efficient bulk queries and mutations

## Quick Start

### Starting the GraphQL Server

1. **Ensure ChromaDB is running**:
   ```bash
   docker run -p 9000:8000 chromadb/chroma:latest run --path /data --host 0.0.0.0
   ```

2. **Set environment variable for ChromaDB**:
   ```bash
   export MCP_MEMORY_CHROMA_ENDPOINT=http://localhost:9000
   ```

3. **Run the GraphQL server**:
   ```bash
   go run cmd/graphql/main.go
   # Or if you've built the binary:
   ./graphql
   ```

4. **Access the interfaces**:
   - Web UI: http://localhost:8082/
   - GraphQL Playground: http://localhost:8082/graphql

## Web UI Usage

### Memory Browser

The main interface shows a list of all memories with:
- **Search Bar**: Full-text search across memory content
- **Filters**: 
  - Repository selector
  - Time period (Recent, Last Month, All Time)
  - Memory type filter
- **Memory List**: Shows summary, repository, timestamp, and relevance score

### Memory Details

Click on any memory to view:
- Full content
- Metadata (session ID, repository, timestamp)
- Tags and classifications
- Related tools and files

### Memory Tracing

When viewing a memory, use the trace buttons:

#### 🔍 Trace Session
- Shows all memories from the same session
- Displays chronological timeline visualization
- Useful for understanding conversation flow

#### 🔗 Find Related
- Discovers semantically similar memories
- Shows relationship graph visualization
- Helps find connected concepts across sessions

### Visualization Panel

The right panel shows interactive visualizations:
- **Timeline View**: For session traces, shows chronological progression
- **Relationship Graph**: For related memories, shows connections in a circular layout
- **Color Coding**: Different memory types have distinct colors
- **Interactive**: Click nodes to navigate between memories

## GraphQL API Reference

### Schema Overview

```graphql
type Query {
  # Search for memories
  search(input: MemoryQueryInput!): SearchResults!
  
  # Get a specific memory by ID
  getChunk(id: String!): ConversationChunk
  
  # List memories by repository
  listChunks(repository: String!, limit: Int, offset: Int): [ConversationChunk!]!
  
  # Get patterns for a repository
  getPatterns(repository: String!, timeframe: String): [Pattern!]!
  
  # Trace all memories in a session
  traceSession(sessionId: String!): [ConversationChunk!]!
  
  # Find related memories
  traceRelated(chunkId: String!, depth: Int): [ConversationChunk!]!
  
  # Get context suggestions
  suggestRelated(
    currentContext: String!
    sessionId: String!
    repository: String
    includePatterns: Boolean
    maxSuggestions: Int
  ): ContextSuggestion!
  
  # Find similar problems
  findSimilar(problem: String!, repository: String, limit: Int): [ConversationChunk!]!
}

type Mutation {
  # Store a new memory
  storeChunk(input: StoreChunkInput!): ConversationChunk!
  
  # Store an architectural decision
  storeDecision(input: StoreDecisionInput!): ConversationChunk!
  
  # Delete a memory
  deleteChunk(id: String!): Boolean!
}
```

### Common Queries

#### Search for Memories
```graphql
query SearchMemories {
  search(input: {
    query: "authentication implementation"
    repository: "my-project"
    limit: 10
    recency: "recent"
  }) {
    chunks {
      chunk {
        id
        content
        summary
        type
        timestamp
        repository
        sessionId
        tags
      }
      score
    }
  }
}
```

#### Trace a Session
```graphql
query TraceSession {
  traceSession(sessionId: "session-123") {
    id
    content
    type
    timestamp
    summary
  }
}
```

#### Find Related Memories
```graphql
query FindRelated {
  traceRelated(chunkId: "memory-id", depth: 2) {
    id
    content
    type
    timestamp
    sessionId
  }
}
```

### Common Mutations

#### Store a Memory
```graphql
mutation StoreMemory {
  storeChunk(input: {
    content: "Implemented user authentication with JWT"
    sessionId: "dev-session-2024"
    repository: "my-project"
    tags: ["auth", "security", "jwt"]
  }) {
    id
    summary
  }
}
```

#### Store a Decision
```graphql
mutation StoreDecision {
  storeDecision(input: {
    decision: "Use Redis for session storage"
    rationale: "Need distributed session management for horizontal scaling"
    sessionId: "architecture-review"
    repository: "my-project"
    context: "Evaluating session storage options for multi-server deployment"
  }) {
    id
    summary
  }
}
```

### Input Types

#### MemoryQueryInput
```graphql
input MemoryQueryInput {
  query: String              # Search text
  repository: String         # Filter by repository
  types: [String!]          # Filter by memory types
  tags: [String!]           # Filter by tags
  limit: Int                # Max results (default: 10)
  minRelevanceScore: Float  # Min similarity score (default: 0.7)
  recency: String           # Time filter: "recent", "last_month", "all_time"
}
```

#### StoreChunkInput
```graphql
input StoreChunkInput {
  content: String!          # Memory content (required)
  sessionId: String!        # Session identifier (required)
  repository: String        # Repository name
  branch: String           # Git branch
  tags: [String!]          # Tags for categorization
  toolsUsed: [String!]     # Tools used in this memory
  filesModified: [String!] # Files that were modified
}
```

## Configuration

### Environment Variables

```bash
# GraphQL Server Port
MCP_MEMORY_GRAPHQL_PORT=8082

# ChromaDB Connection
MCP_MEMORY_CHROMA_ENDPOINT=http://localhost:9000

# OpenAI API (for embeddings)
OPENAI_API_KEY=your-api-key

# Logging
LOG_LEVEL=info
```

### Docker Compose

The GraphQL server is included in the Docker Compose setup:

```yaml
services:
  graphql:
    build:
      context: .
      target: graphql
    ports:
      - "8082:8082"
    environment:
      - MCP_MEMORY_CHROMA_ENDPOINT=http://chroma:8000
      - OPENAI_API_KEY=${OPENAI_API_KEY}
    depends_on:
      - chroma
```

## Import Scripts

### Importing Documentation

Use the provided import script to bulk import documentation:

```bash
# Make script executable
chmod +x scripts/import-docs-simple.sh

# Run import
./scripts/import-docs-simple.sh
```

The script will:
1. Read documentation files
2. Create appropriate memories
3. Tag them for easy retrieval
4. Report import statistics

### Custom Import

Create your own import scripts using curl:

```bash
curl -X POST http://localhost:8082/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation { storeChunk(input: { content: \"Your content here\", sessionId: \"import-session\", repository: \"your-repo\", tags: [\"imported\"] }) { id } }"
  }'
```

## Troubleshooting

### Common Issues

1. **"Loading..." forever in Web UI**
   - Check if GraphQL server is running
   - Verify ChromaDB is accessible
   - Check browser console for errors

2. **Empty memory list**
   - Ensure ChromaDB persistence is configured correctly
   - Check if memories have been imported
   - Verify repository filter isn't excluding results

3. **Connection refused errors**
   - ChromaDB default port is 9000 (not 8000)
   - Use IPv4 (127.0.0.1) instead of IPv6 (::1)
   - Check Docker container is running

### Debug Mode

Enable debug logging:
```bash
LOG_LEVEL=debug ./graphql
```

## Advanced Features

### Memory Tracing Algorithm

The relationship discovery uses:
1. **Semantic Similarity**: Finds memories with similar embeddings
2. **Breadth-First Search**: Explores connections up to specified depth
3. **Score Threshold**: Filters by minimum similarity score (0.7 default)

### Visualization Components

The UI uses:
- **Canvas API**: For custom graph rendering
- **D3.js concepts**: Force-directed layouts for relationships
- **Timeline.js patterns**: For chronological displays

## API Limits

- **Search Results**: Default 10, max 100
- **Trace Depth**: Default 2, max 5
- **Session Trace**: No limit (returns all)
- **Bulk Operations**: Not currently supported

## Future Enhancements

Planned features include:
- Real-time subscriptions for live updates
- Bulk import/export operations
- Advanced filtering with date ranges
- Custom visualization layouts
- Memory editing capabilities
- Collaborative annotations

## Related Documentation

- [Architecture Overview](ARCHITECTURE.md) - System design details
- [API Reference](website/api-reference.md) - Complete API documentation
- [Development Guide](DEV-HOT-RELOAD.md) - Development setup
- [ChromaDB Persistence](CHROMADB_PERSISTENCE_FIX.md) - Storage configuration