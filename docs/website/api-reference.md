# API Reference

Complete reference for all MCP Memory tools and their parameters.

## Core Memory Tools

### memory_store_chunk

Stores a conversation chunk with automatic analysis and embedding generation.

**Parameters:**
- `content` (string, required): The conversation content to store
- `session_id` (string, required): Session identifier for grouping related chunks
- `repository` (string, optional): Repository name
- `branch` (string, optional): Git branch name
- `tools_used` (array, optional): List of tools that were used
- `files_modified` (array, optional): List of modified files
- `tags` (array, optional): Additional tags for categorization

**Example:**
```json
{
  "content": "Implemented JWT authentication with refresh tokens",
  "session_id": "auth-implementation-2024",
  "repository": "my-app",
  "tools_used": ["Edit", "Write"],
  "files_modified": ["auth/jwt.go", "auth/middleware.go"],
  "tags": ["authentication", "security"]
}
```

### memory_search

Searches for similar conversation chunks using semantic search.

**Parameters:**
- `query` (string, required): Natural language search query
- `limit` (number, optional): Maximum results (default: 10, max: 50)
- `min_relevance` (number, optional): Minimum relevance score 0-1 (default: 0.7)
- `repository` (string, optional): Filter by repository
- `recency` (string, optional): Time filter - "recent", "last_month", "all_time"
- `types` (array, optional): Filter by chunk types

**Example:**
```json
{
  "query": "how did we implement rate limiting",
  "limit": 5,
  "repository": "api-gateway",
  "recency": "last_month"
}
```

### memory_find_similar

Finds similar past problems and their solutions.

**Parameters:**
- `problem` (string, required): Description of the current problem
- `repository` (string, optional): Repository context
- `limit` (number, optional): Maximum results (default: 5, max: 20)

**Example:**
```json
{
  "problem": "PostgreSQL connection pool exhausted under load",
  "repository": "backend-api",
  "limit": 3
}
```

## Context Management Tools

### memory_get_context

Gets project context and recent activity for session initialization.

**Parameters:**
- `repository` (string, required): Repository name
- `recent_days` (number, optional): Days of history (default: 7, max: 90)

**Returns:**
- Project overview and goals
- Recent conversations and decisions
- Active patterns and workflows
- Common issues and solutions

### memory_suggest_related

Gets AI-powered suggestions for related context.

**Parameters:**
- `current_context` (string, required): Current work context
- `session_id` (string, required): Session identifier
- `repository` (string, optional): Repository to search
- `max_suggestions` (number, optional): Maximum suggestions (default: 5)
- `include_patterns` (boolean, optional): Include pattern-based suggestions

**Example:**
```json
{
  "current_context": "Working on database migration scripts",
  "session_id": "migration-2024",
  "repository": "backend",
  "include_patterns": true
}
```

## Knowledge Management Tools

### memory_store_decision

Stores architectural decisions with rationale.

**Parameters:**
- `decision` (string, required): The architectural decision
- `rationale` (string, required): Reasoning behind the decision
- `session_id` (string, required): Session identifier
- `repository` (string, optional): Repository this applies to
- `context` (string, optional): Additional context and alternatives

**Example:**
```json
{
  "decision": "Use event sourcing for order management",
  "rationale": "Provides complete audit trail and enables event replay for debugging",
  "context": "Considered CRUD but lacks history tracking",
  "session_id": "architecture-review-2024",
  "repository": "order-service"
}
```

### memory_get_patterns

Identifies recurring patterns in project history.

**Parameters:**
- `repository` (string, required): Repository to analyze
- `timeframe` (string, optional): Analysis period - "week", "month", "quarter", "all"

**Returns:**
- Common error patterns and fixes
- Frequently modified code areas
- Recurring architectural decisions
- Team workflow patterns

## Data Management Tools

### memory_import_context

Imports conversation context from external sources.

**Parameters:**
- `source` (string, required): Source type - "conversation", "file", "archive"
- `data` (string, required): Data to import (text, file content, or base64)
- `repository` (string, required): Target repository
- `session_id` (string, required): Session identifier
- `chunking_strategy` (string, optional): How to chunk - "auto", "paragraph", "fixed_size"
- `metadata` (object, optional): Import metadata

**Example:**
```json
{
  "source": "file",
  "data": "Previous conversation export...",
  "repository": "my-project",
  "session_id": "import-2024",
  "metadata": {
    "source_system": "slack",
    "import_date": "2024-01-15"
  }
}
```

### memory_export_project

Exports all memory data for a project.

**Parameters:**
- `repository` (string, required): Repository to export
- `session_id` (string, required): Session identifier
- `format` (string, optional): Export format - "json", "markdown", "archive"
- `date_range` (object, optional): Date filter with start/end
- `include_vectors` (boolean, optional): Include embeddings

**Example:**
```json
{
  "repository": "my-project",
  "session_id": "export-2024",
  "format": "markdown",
  "date_range": {
    "start": "2024-01-01",
    "end": "2024-12-31"
  }
}
```

## System Tools

### memory_health

Checks the health status of the memory system.

**Parameters:** None

**Returns:**
- Storage backend status
- Embedding service status
- Recent activity metrics
- System diagnostics

## Advanced Features

### Pattern Recognition

MCP Memory automatically identifies patterns:

- **Error Patterns**: Common errors and their solutions
- **Code Patterns**: Frequently used code structures
- **Workflow Patterns**: Common development workflows
- **Architecture Patterns**: Recurring design decisions

### Knowledge Graphs

Automatically builds relationships between:

- Files and their purposes
- Components and dependencies
- Problems and solutions
- Decisions and outcomes

### Multi-Repository Support

- Maintains separate contexts per repository
- Cross-repository pattern identification
- Shared knowledge across projects
- Repository-specific configurations

## Best Practices

1. **Consistent Tagging**: Use consistent tags for better organization
2. **Descriptive Content**: Include enough context in stored chunks
3. **Regular Storage**: Store important conversations as they happen
4. **Session Management**: Use meaningful session IDs for grouping

## Error Handling

All tools return structured errors:

```json
{
  "error": {
    "code": "STORAGE_ERROR",
    "message": "Failed to connect to ChromaDB",
    "details": "Connection refused at localhost:8000"
  }
}
```

Common error codes:
- `STORAGE_ERROR`: Storage backend issues
- `EMBEDDING_ERROR`: Embedding generation failed
- `VALIDATION_ERROR`: Invalid parameters
- `NOT_FOUND`: Resource not found

## Rate Limits

- Search operations: 100 requests/minute
- Storage operations: 50 requests/minute
- Export operations: 10 requests/hour

---

Need help? Check our [FAQ](faq.md) or join our [Discord community](https://discord.gg/mcp-memory).