# MCP Memory Tools Documentation

## Overview

The MCP Memory system provides intelligent memory management for Claude, allowing it to store, search, and learn from past conversations and decisions. This document explains each tool, its purpose, and when to use it.

## Core Concepts

- **Repository**: A project or context boundary. Use `_global` for cross-project memories.
- **Session ID**: Groups related conversation chunks within a session.
- **Embeddings**: Automatically generated for semantic search capability.
- **Chunk Types**: Automatically detected (problem, solution, decision, etc.).

## Available Tools

### 1. `mcp__memory__memory_store_chunk`

**Purpose**: Store conversation segments with automatic analysis and categorization.

**When to use**:
- After solving a problem or bug
- When making important decisions
- After implementing a feature
- When learning something new
- To preserve important context for future sessions

**Parameters**:
- `content` (required): The conversation content to store
- `session_id` (required): Current session identifier
- `repository` (optional): Project name or `_global`
- `branch` (optional): Git branch name
- `files_modified` (optional): List of modified files
- `tools_used` (optional): List of tools used
- `tags` (optional): Custom tags for categorization

**Example scenarios**:
```json
// After fixing a bug
{
  "content": "Fixed the memory leak in the connection pool by properly closing idle connections after 30 minutes",
  "session_id": "session-123",
  "repository": "my-project",
  "files_modified": ["internal/pool/connection.go"],
  "tools_used": ["Read", "Edit"],
  "tags": ["bug-fix", "memory-leak", "connection-pool"]
}

// After an architecture decision
{
  "content": "Decided to use ChromaDB for vector storage due to its simplicity and Go client support",
  "session_id": "session-123",
  "repository": "_global",
  "tags": ["architecture-decision", "vector-database"]
}
```

### 2. `mcp__memory__memory_search`

**Purpose**: Search past memories using natural language queries with semantic understanding.

**When to use**:
- Looking for similar problems you've solved before
- Finding past decisions and their rationale
- Searching for code patterns or implementations
- Retrieving context from previous sessions

**Parameters**:
- `query` (required): Natural language search query
- `repository` (optional): Filter by project or `_global`
- `recency` (optional): Time filter - "recent" (7 days), "last_month", "all_time"
- `types` (optional): Filter by chunk types ["problem", "solution", "decision", etc.]
- `limit` (optional): Max results (1-50, default: 10)
- `min_relevance` (optional): Minimum similarity score (0-1, default: 0.7)

**Example scenarios**:
```json
// Find similar bugs
{
  "query": "connection pool memory leak timeout issues",
  "repository": "my-project",
  "types": ["problem", "solution"],
  "recency": "all_time"
}

// Find architecture decisions
{
  "query": "why did we choose this database vector storage approach",
  "repository": "_global",
  "types": ["architecture_decision"],
  "limit": 5
}
```

### 3. `mcp__memory__memory_get_context`

**Purpose**: Get project overview and recent activity when starting a new session.

**When to use**:
- At the beginning of a new conversation
- When switching between projects
- To understand recent changes and decisions
- To get familiar with a project's patterns

**Parameters**:
- `repository` (required): Project name or `_global`
- `recent_days` (optional): Days of history (1-90, default: 7)

**Returns**: Recent chunks, patterns, common issues, and project insights.

### 4. `mcp__memory__memory_find_similar`

**Purpose**: Find similar problems and their solutions from past experiences.

**When to use**:
- When encountering an error or bug
- Before implementing a complex feature
- When facing a technical challenge
- To learn from past solutions

**Parameters**:
- `problem` (required): Description of the current problem
- `repository` (optional): Project context or `_global`
- `limit` (optional): Max results (1-20, default: 5)

**Example**:
```json
{
  "problem": "Getting 'connection refused' errors when trying to connect to ChromaDB in Docker",
  "repository": "my-project",
  "limit": 3
}
```

### 5. `mcp__memory__memory_store_decision`

**Purpose**: Explicitly store architectural or design decisions with full context.

**When to use**:
- After making significant architectural choices
- When choosing between alternatives
- After technical discussions or trade-off analysis
- To document why certain approaches were taken

**Parameters**:
- `decision` (required): The decision made
- `rationale` (required): Why this decision was made
- `context` (optional): Alternatives considered, constraints, etc.
- `repository` (optional): Project or `_global`
- `session_id` (required): Current session ID

**Example**:
```json
{
  "decision": "Use connection pooling for ChromaDB with max 10 connections",
  "rationale": "Prevents connection exhaustion under load while maintaining reasonable resource usage",
  "context": "Considered single connection (too slow) and unlimited connections (resource exhaustion). Benchmarks showed 10 connections optimal for our workload.",
  "repository": "mcp-memory",
  "session_id": "session-123"
}
```

### 6. `mcp__memory__memory_get_patterns`

**Purpose**: Identify recurring patterns, common issues, and trends in project history.

**When to use**:
- During retrospectives or reviews
- To identify areas needing refactoring
- To understand common challenges
- For learning and improvement

**Parameters**:
- `repository` (required): Project or `_global`
- `timeframe` (optional): Analysis period - "week", "month", "quarter", "all"

**Returns**: Common patterns, frequent issues, popular tools, and trends.

### 7. `mcp__memory__memory_health`

**Purpose**: Check the health and status of the memory system.

**When to use**:
- Troubleshooting memory system issues
- Monitoring system status
- Before important operations

**Parameters**: None

**Returns**: System status, storage stats, and diagnostics.

## Advanced Features

### Automatic Intelligence

The system automatically:
- Generates embeddings for semantic search
- Detects chunk types (problem, solution, decision, etc.)
- Identifies importance scores
- Extracts entities and concepts
- Links related memories
- Detects patterns over time

### Memory Decay

- Older, less accessed memories naturally decay in importance
- Frequently accessed memories are boosted
- System automatically summarizes and consolidates old memories

### Multi-Repository Support

- Store project-specific memories with repository name
- Store global memories with `_global` repository
- Cross-repository pattern detection and learning

## Best Practices

1. **Store Important Moments**: After solving bugs, making decisions, or learning something new
2. **Use Descriptive Content**: Include context, approach, and outcomes
3. **Tag Appropriately**: Use consistent tags for better retrieval
4. **Search Before Solving**: Check if similar problems were solved before
5. **Document Decisions**: Store architectural decisions with full rationale
6. **Regular Context Checks**: Use `get_context` at session start

## Integration Tips

### For Bug Fixes
```
1. Search for similar issues first
2. Store the problem description
3. Store the solution with full context
4. Tag with "bug-fix" and specific area
```

### For Feature Development
```
1. Get context for the repository
2. Search for related implementations
3. Store design decisions
4. Store implementation insights
```

### For Learning
```
1. Store new concepts learned
2. Store examples and patterns
3. Link to documentation or resources
4. Tag with technology/framework
```

## Performance Considerations

- Searches use vector similarity (fast)
- Results are relevance-ranked
- Recent memories are prioritized
- System handles thousands of memories efficiently
- Automatic cleanup of old, unused memories

## Privacy and Security

- All memories are stored locally
- No external API calls for memory storage
- Repository isolation for project separation
- Sensitive data should not be stored