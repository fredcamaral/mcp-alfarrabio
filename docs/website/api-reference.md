# API Reference

Complete reference for the MCP Memory GraphQL API and MCP tools.

## GraphQL API

The primary interface for interacting with the memory system is the GraphQL API available at `http://localhost:8082/graphql`.

### Queries

#### search
Search for memories using semantic similarity.

```graphql
query SearchMemories($input: MemoryQueryInput!) {
  search(input: $input) {
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

**Variables:**
```json
{
  "input": {
    "query": "authentication implementation",
    "repository": "my-project",
    "limit": 10,
    "recency": "recent",
    "minRelevanceScore": 0.7
  }
}
```

#### getChunk
Retrieve a specific memory by ID.

```graphql
query GetMemory($id: String!) {
  getChunk(id: $id) {
    id
    content
    summary
    type
    timestamp
    repository
    sessionId
    tags
    metadata {
      repository
      branch
      filesModified
      toolsUsed
      outcome
      difficulty
    }
  }
}
```

#### listChunks
List memories from a specific repository.

```graphql
query ListMemories($repository: String!, $limit: Int, $offset: Int) {
  listChunks(repository: $repository, limit: $limit, offset: $offset) {
    id
    content
    type
    timestamp
    summary
  }
}
```

#### traceSession
Get all memories from a specific session in chronological order.

```graphql
query TraceSession($sessionId: String!) {
  traceSession(sessionId: $sessionId) {
    id
    content
    type
    timestamp
    summary
    repository
  }
}
```

#### traceRelated
Find memories related to a specific memory using similarity search.

```graphql
query TraceRelated($chunkId: String!, $depth: Int) {
  traceRelated(chunkId: $chunkId, depth: $depth) {
    id
    content
    type
    timestamp
    sessionId
  }
}
```

#### getPatterns
Analyze patterns in a repository's memories.

```graphql
query GetPatterns($repository: String!, $timeframe: String) {
  getPatterns(repository: $repository, timeframe: $timeframe) {
    name
    description
    occurrences
    confidence
    lastSeen
    examples
  }
}
```

#### suggestRelated
Get AI-powered suggestions based on current context.

```graphql
query SuggestRelated(
  $currentContext: String!
  $sessionId: String!
  $repository: String
  $includePatterns: Boolean
  $maxSuggestions: Int
) {
  suggestRelated(
    currentContext: $currentContext
    sessionId: $sessionId
    repository: $repository
    includePatterns: $includePatterns
    maxSuggestions: $maxSuggestions
  ) {
    relevantChunks {
      id
      content
      relevance
    }
    suggestedTasks
    relatedConcepts
    potentialIssues
  }
}
```

#### findSimilar
Find similar problems and their solutions.

```graphql
query FindSimilar($problem: String!, $repository: String, $limit: Int) {
  findSimilar(problem: $problem, repository: $repository, limit: $limit) {
    id
    content
    outcome
    solutionApproach
    timestamp
  }
}
```

### Mutations

#### storeChunk
Store a new memory in the system.

```graphql
mutation StoreMemory($input: StoreChunkInput!) {
  storeChunk(input: $input) {
    id
    summary
    timestamp
  }
}
```

**Variables:**
```json
{
  "input": {
    "content": "Implemented user authentication with JWT tokens",
    "sessionId": "dev-session-2024",
    "repository": "my-project",
    "branch": "feature/auth",
    "tags": ["authentication", "security", "jwt"],
    "toolsUsed": ["Edit", "Write"],
    "filesModified": ["auth/jwt.go", "auth/middleware.go"]
  }
}
```

#### storeDecision
Store an architectural decision with rationale.

```graphql
mutation StoreDecision($input: StoreDecisionInput!) {
  storeDecision(input: $input) {
    id
    summary
  }
}
```

**Variables:**
```json
{
  "input": {
    "decision": "Use Redis for session storage",
    "rationale": "Need distributed session management for horizontal scaling",
    "sessionId": "architecture-review",
    "repository": "my-project",
    "context": "Evaluating session storage options for multi-server deployment"
  }
}
```

#### deleteChunk
Delete a memory by ID.

```graphql
mutation DeleteMemory($id: String!) {
  deleteChunk(id: $id)
}
```

### Types

#### MemoryQueryInput
```graphql
input MemoryQueryInput {
  query: String              # Search text
  repository: String         # Filter by repository
  types: [String!]          # Filter by memory types
  tags: [String!]           # Filter by tags
  limit: Int                # Max results (default: 10)
  minRelevanceScore: Float  # Min similarity score (default: 0.7)
  recency: String           # "recent", "last_month", "all_time"
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
  toolsUsed: [String!]     # Tools used
  filesModified: [String!] # Modified files
}
```

#### StoreDecisionInput
```graphql
input StoreDecisionInput {
  decision: String!         # The decision made (required)
  rationale: String!        # Why this decision (required)
  sessionId: String!        # Session ID (required)
  repository: String        # Repository
  context: String          # Additional context
}
```

#### ConversationChunk
```graphql
type ConversationChunk {
  id: String!
  sessionId: String!
  repository: String
  timestamp: DateTime!
  type: String!
  content: String!
  summary: String
  tags: [String!]
  
  # Extended metadata
  branch: String
  toolsUsed: [String!]
  filePaths: [String!]
  entities: [String!]
  concepts: [String!]
  
  # Analysis results
  outcome: String
  difficulty: String
  problemDescription: String
  solutionApproach: String
  lessonsLearned: String
  nextSteps: String
  decisionRationale: String
  decisionOutcome: String
}
```

## MCP Tools (Legacy)

The following MCP tools are available for backward compatibility. New integrations should use the GraphQL API.

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

### memory_get_patterns

Identifies recurring patterns in project history.

**Parameters:**
- `repository` (string, required): Repository to analyze
- `timeframe` (string, optional): "week", "month", "quarter", "all"

### memory_suggest_related

Get AI-powered suggestions for related context.

**Parameters:**
- `current_context` (string, required): Current work context
- `session_id` (string, required): Session identifier
- `repository` (string, optional): Repository context
- `include_patterns` (boolean, optional): Include pattern suggestions
- `max_suggestions` (number, optional): Maximum suggestions (default: 5)

### memory_find_similar

Find similar past problems and their solutions.

**Parameters:**
- `problem` (string, required): Description of the current problem
- `repository` (string, optional): Repository context
- `limit` (number, optional): Maximum results (default: 5)

### memory_store_decision

Store an architectural decision with rationale.

**Parameters:**
- `decision` (string, required): The architectural decision
- `rationale` (string, required): Reasoning behind the decision
- `session_id` (string, required): Session identifier
- `repository` (string, optional): Repository this applies to
- `context` (string, optional): Additional context

### memory_export_project

Export all memory data for a project.

**Parameters:**
- `repository` (string, required): Repository to export
- `session_id` (string, required): Session identifier
- `format` (string, optional): "json", "markdown", "archive"
- `include_vectors` (boolean, optional): Include embeddings
- `date_range` (object, optional): Start and end dates

### memory_import_context

Import conversation context from external source.

**Parameters:**
- `source` (string, required): "conversation", "file", "archive"
- `data` (string, required): Data to import
- `repository` (string, required): Target repository
- `session_id` (string, required): Session identifier
- `chunking_strategy` (string, optional): How to chunk the data
- `metadata` (object, optional): Import metadata

## Error Codes

### GraphQL Errors
- `INVALID_INPUT`: Invalid input parameters
- `NOT_FOUND`: Resource not found
- `EMBEDDING_ERROR`: Failed to generate embeddings
- `STORAGE_ERROR`: Database operation failed
- `VALIDATION_ERROR`: Input validation failed

### MCP Tool Errors
- `tool.memory.invalid_content`: Empty or invalid content
- `tool.memory.embedding_failed`: Embedding generation failed
- `tool.memory.storage_error`: Storage operation failed
- `tool.memory.not_found`: Requested resource not found

## Rate Limits

- **Search queries**: 60/minute per user
- **Store operations**: 30/minute per user
- **Export operations**: 5/minute per user
- **Batch operations**: 10/minute per user

## Best Practices

1. **Session Management**
   - Use consistent session IDs for related work
   - Include timestamp in session IDs for clarity
   - Group related memories in the same session

2. **Tagging Strategy**
   - Use consistent tag naming conventions
   - Include technology tags (e.g., "golang", "react")
   - Add workflow tags (e.g., "bug-fix", "feature")

3. **Repository Organization**
   - Use repository names that match your project structure
   - Consider using "_global" for cross-project memories
   - Separate concerns by repository

4. **Search Optimization**
   - Use specific search queries for better results
   - Leverage filters to narrow results
   - Set appropriate relevance thresholds

5. **Performance Tips**
   - Batch related operations when possible
   - Use pagination for large result sets
   - Cache frequently accessed memories

## Migration Guide

### From MCP Tools to GraphQL

1. **Replace tool calls with GraphQL mutations**:
   ```javascript
   // Old (MCP Tool)
   await mcp.callTool('memory_store_chunk', {
     content: 'Implementation details',
     session_id: 'session-123'
   });
   
   // New (GraphQL)
   await graphql.mutate({
     mutation: STORE_CHUNK,
     variables: {
       input: {
         content: 'Implementation details',
         sessionId: 'session-123'
       }
     }
   });
   ```

2. **Update search queries**:
   ```javascript
   // Old (MCP Tool)
   const results = await mcp.callTool('memory_search', {
     query: 'authentication',
     limit: 10
   });
   
   // New (GraphQL)
   const { data } = await graphql.query({
     query: SEARCH_MEMORIES,
     variables: {
       input: {
         query: 'authentication',
         limit: 10
       }
     }
   });
   ```

3. **Use tracing features**:
   - Replace manual session filtering with `traceSession`
   - Use `traceRelated` for discovering connections
   - Leverage the Web UI for visual exploration