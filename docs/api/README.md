# MCP Memory API Reference

## Overview

MCP Memory provides three API interfaces:
1. **MCP Protocol** - Native Model Context Protocol tools
2. **GraphQL API** - Flexible query interface
3. **OpenAPI REST** - RESTful HTTP endpoints

## MCP Protocol Tools

### memory_store_chunk

Store a conversation chunk in memory with automatic analysis and embedding generation.

**Parameters:**
- `content` (string, required): The conversation content to store
- `session_id` (string, required): Session identifier for grouping related chunks
- `repository` (string, optional): Repository name or '_global' for global memories
- `branch` (string, optional): Git branch name
- `tags` (array[string], optional): Additional tags for categorization
- `tools_used` (array[string], optional): List of tools that were used
- `files_modified` (array[string], optional): List of files that were modified

**Returns:**
```json
{
  "chunk_id": "uuid",
  "summary": "AI-generated summary",
  "concepts": ["extracted", "concepts"],
  "entities": ["detected", "entities"],
  "stored_at": "2024-01-01T12:00:00Z"
}
```

### memory_search

Search for similar conversation chunks based on natural language query.

**Parameters:**
- `query` (string, required): Natural language search query
- `repository` (string, optional): Filter by repository name
- `types` (array[string], optional): Filter by chunk types
- `limit` (integer, default: 10): Maximum number of results
- `min_relevance` (float, default: 0.7): Minimum relevance score (0-1)
- `recency` (enum, default: "recent"): Time filter ["recent", "last_month", "all_time"]

**Returns:**
```json
{
  "chunks": [
    {
      "chunk": { /* chunk data */ },
      "score": 0.95,
      "highlights": ["relevant", "snippets"]
    }
  ],
  "total_found": 42
}
```

### memory_get_patterns

Identify recurring patterns in project history.

**Parameters:**
- `repository` (string, required): Repository to analyze
- `timeframe` (enum, default: "month"): Time period ["week", "month", "quarter", "all"]

**Returns:**
```json
{
  "patterns": [
    {
      "name": "Authentication Implementation",
      "description": "Recurring pattern of auth-related work",
      "occurrences": 15,
      "confidence": 0.85,
      "last_seen": "2024-01-01T12:00:00Z",
      "examples": ["chunk_id_1", "chunk_id_2"]
    }
  ]
}
```

### memory_suggest_related

Get AI-powered suggestions for related context based on current work.

**Parameters:**
- `current_context` (string, required): Current work context or conversation
- `session_id` (string, required): Session identifier
- `repository` (string, optional): Repository to search for context
- `include_patterns` (boolean, default: true): Include pattern-based suggestions
- `max_suggestions` (integer, default: 5): Maximum suggestions to return

**Returns:**
```json
{
  "relevant_chunks": [/* array of scored chunks */],
  "suggested_tasks": ["Consider adding tests", "Review security implications"],
  "related_concepts": ["authentication", "JWT", "session management"],
  "potential_issues": ["Similar bug fixed in commit abc123"]
}
```

### memory_find_similar

Find similar past problems and their solutions.

**Parameters:**
- `problem` (string, required): Description of the current problem
- `repository` (string, optional): Repository context
- `limit` (integer, default: 5): Maximum number of similar problems

**Returns:**
```json
{
  "similar_problems": [
    {
      "chunk": { /* chunk data */ },
      "similarity": 0.92,
      "solution": "Applied fix by...",
      "outcome": "Resolved successfully"
    }
  ]
}
```

### memory_store_decision

Store an architectural decision with rationale.

**Parameters:**
- `decision` (string, required): The architectural decision made
- `rationale` (string, required): Reasoning behind the decision
- `context` (string, optional): Additional context and alternatives
- `session_id` (string, required): Session identifier
- `repository` (string, optional): Repository this applies to

**Returns:**
```json
{
  "chunk_id": "uuid",
  "decision": "Use PostgreSQL for primary storage",
  "stored_at": "2024-01-01T12:00:00Z"
}
```

### memory_export_project

Export all memory data for a project in various formats.

**Parameters:**
- `repository` (string, required): Repository to export
- `session_id` (string, required): Session identifier
- `format` (enum, default: "json"): Export format ["json", "markdown", "archive"]
- `include_vectors` (boolean, optional): Include vector embeddings
- `date_range` (object, optional): Filter by date range
  - `start` (string): Start date (ISO 8601)
  - `end` (string): End date (ISO 8601)

**Returns:**
```json
{
  "export_id": "uuid",
  "format": "json",
  "size_bytes": 1048576,
  "chunk_count": 250,
  "download_url": "https://...",
  "expires_at": "2024-01-02T12:00:00Z"
}
```

### memory_import_context

Import conversation context from external source.

**Parameters:**
- `source` (enum, required): Source type ["conversation", "file", "archive"]
- `data` (string, required): Data to import (text, file content, or base64)
- `repository` (string, required): Target repository
- `session_id` (string, required): Session identifier
- `chunking_strategy` (enum, default: "auto"): How to chunk ["auto", "paragraph", "fixed_size", "conversation_turns"]
- `metadata` (object, optional): Import metadata
  - `source_system` (string): Name of source system
  - `import_date` (string): Original date of content
  - `tags` (array[string]): Tags to apply

**Returns:**
```json
{
  "imported_chunks": 42,
  "total_tokens": 15000,
  "processing_time_ms": 3500
}
```

### memory_get_context

Get project context and recent activity for session initialization.

**Parameters:**
- `repository` (string, required): Repository name to get context for
- `recent_days` (integer, default: 7): Number of recent days to include

**Returns:**
```json
{
  "repository": "my-project",
  "total_chunks": 1250,
  "recent_activity": [/* recent chunks */],
  "common_patterns": [/* detected patterns */],
  "active_topics": ["authentication", "database", "API design"],
  "suggested_context": "Recent work focused on auth implementation..."
}
```

### memory_health

Check the health status of the memory system.

**Parameters:** None

**Returns:**
```json
{
  "status": "healthy",
  "vector_store": {
    "status": "connected",
    "chunk_count": 50000,
    "response_time_ms": 15
  },
  "embedding_service": {
    "status": "connected",
    "model": "text-embedding-3-small",
    "rate_limit_remaining": 9500
  },
  "storage": {
    "used_bytes": 524288000,
    "free_bytes": 10737418240
  }
}
```

## GraphQL API

### Endpoint
```
POST http://localhost:8082/graphql
```

### Authentication
```http
Authorization: Bearer <token>
```

### Example Query
```graphql
query SearchMemories($query: MemoryQueryInput!) {
  search(input: $query) {
    chunks {
      score
      chunk {
        id
        content
        summary
        timestamp
        repository
        tags
      }
    }
  }
}
```

### Example Variables
```json
{
  "query": {
    "query": "authentication implementation",
    "repository": "my-project",
    "limit": 10,
    "minRelevanceScore": 0.8
  }
}
```

## OpenAPI REST

### Base URL
```
http://localhost:8081/api/v1
```

### Authentication
```http
Authorization: Bearer <token>
```

### Endpoints

#### POST /chunks
Store a new conversation chunk.

**Request Body:**
```json
{
  "content": "Implemented JWT authentication",
  "session_id": "session-123",
  "repository": "my-project",
  "tags": ["auth", "security"]
}
```

**Response:**
```json
{
  "chunk_id": "uuid",
  "summary": "JWT authentication implementation",
  "stored_at": "2024-01-01T12:00:00Z"
}
```

#### GET /chunks/search
Search for conversation chunks.

**Query Parameters:**
- `q` - Search query
- `repository` - Filter by repository
- `type` - Filter by type
- `limit` - Maximum results (default: 10)
- `offset` - Pagination offset

**Response:**
```json
{
  "results": [/* array of chunks */],
  "total": 42,
  "limit": 10,
  "offset": 0
}
```

#### GET /chunks/{id}
Get a specific chunk by ID.

**Response:**
```json
{
  "id": "uuid",
  "content": "...",
  "summary": "...",
  "metadata": { /* chunk metadata */ }
}
```

#### DELETE /chunks/{id}
Delete a chunk.

**Response:**
```json
{
  "deleted": true
}
```

#### GET /patterns
Get detected patterns.

**Query Parameters:**
- `repository` - Repository to analyze
- `timeframe` - Time period (week|month|quarter|all)

**Response:**
```json
{
  "patterns": [/* array of patterns */]
}
```

#### POST /decisions
Store an architectural decision.

**Request Body:**
```json
{
  "decision": "Use PostgreSQL",
  "rationale": "ACID compliance and JSON support",
  "context": "Evaluated multiple options...",
  "session_id": "session-123",
  "repository": "my-project"
}
```

#### GET /health
Health check endpoint.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-01T12:00:00Z",
  "services": { /* service statuses */ }
}
```

## Error Handling

### Error Response Format
```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "Human-readable error message",
    "details": {
      "field": "repository",
      "reason": "Repository not found"
    }
  }
}
```

### Common Error Codes
- `INVALID_REQUEST` - Malformed request
- `NOT_FOUND` - Resource not found
- `RATE_LIMITED` - Rate limit exceeded
- `UNAUTHORIZED` - Authentication required
- `FORBIDDEN` - Insufficient permissions
- `INTERNAL_ERROR` - Server error

### Rate Limiting
```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1609459200
```

## Webhooks (Planned)

### Event Types
- `chunk.created` - New chunk stored
- `pattern.detected` - New pattern identified
- `decision.stored` - Architectural decision recorded

### Webhook Payload
```json
{
  "event": "chunk.created",
  "timestamp": "2024-01-01T12:00:00Z",
  "data": { /* event-specific data */ }
}
```

## SDK Examples

### Python
```python
from mcp_memory import MemoryClient

client = MemoryClient(api_key="...")
results = client.search(
    query="authentication", 
    repository="my-project"
)
```

### JavaScript/TypeScript
```typescript
import { MemoryClient } from '@mcp/memory-client';

const client = new MemoryClient({ apiKey: '...' });
const results = await client.search({
  query: 'authentication',
  repository: 'my-project'
});
```

### Go
```go
client := memory.NewClient("api-key")
results, err := client.Search(ctx, &memory.SearchQuery{
    Query:      "authentication",
    Repository: "my-project",
})
```