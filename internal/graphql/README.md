# GraphQL API for MCP Memory

The GraphQL API provides a flexible query interface for the MCP memory system, allowing clients to search, store, and manage conversation memories.

## Starting the Server

```bash
go run cmd/graphql/main.go
```

The server will start on port 8082 by default (configurable via `GRAPHQL_PORT` environment variable).

## GraphiQL Playground

Access the interactive GraphQL playground at: http://localhost:8082/graphql

## Schema Overview

### Queries

- `search`: Search for memories using natural language queries
- `getChunk`: Get a specific memory chunk by ID
- `listChunks`: List all chunks in a repository
- `getPatterns`: Identify patterns in a repository's history
- `suggestRelated`: Get AI-powered suggestions for related context
- `findSimilar`: Find similar problems and their solutions

### Mutations

- `storeChunk`: Store a new conversation chunk
- `storeDecision`: Store an architectural decision
- `deleteChunk`: Delete a memory chunk

## Example Queries

### Search for Memories

```graphql
query SearchMemories {
  search(input: {
    query: "authentication implementation"
    repository: "my-project"
    types: ["code", "decision"]
    limit: 10
    minRelevanceScore: 0.7
    recency: "recent"
  }) {
    chunks {
      score
      chunk {
        id
        content
        summary
        timestamp
        type
        tags
      }
    }
  }
}
```

### Store a Conversation

```graphql
mutation StoreConversation {
  storeChunk(input: {
    content: "Implemented JWT authentication with refresh tokens"
    sessionId: "session-123"
    repository: "my-project"
    tags: ["auth", "security", "jwt"]
    toolsUsed: ["vscode", "postman"]
    filesModified: ["auth.go", "middleware.go"]
  }) {
    id
    summary
    timestamp
  }
}
```

### Store an Architectural Decision

```graphql
mutation StoreDecision {
  storeDecision(input: {
    decision: "Use PostgreSQL for primary data storage"
    rationale: "PostgreSQL provides ACID compliance, JSON support, and excellent performance for our use case"
    sessionId: "session-123"
    repository: "my-project"
    context: "Evaluated MongoDB, MySQL, and PostgreSQL. PostgreSQL won due to its feature set and team expertise."
  }) {
    id
    decisionOutcome
    decisionRationale
    timestamp
  }
}
```

### Find Similar Problems

```graphql
query FindSimilarProblems {
  findSimilar(
    problem: "Getting CORS errors when calling API from frontend"
    repository: "my-project"
    limit: 5
  ) {
    id
    problemDescription
    solutionApproach
    outcome
    lessonsLearned
  }
}
```

### Get Context Suggestions

```graphql
query GetSuggestions {
  suggestRelated(
    currentContext: "Working on user authentication flow"
    sessionId: "session-123"
    repository: "my-project"
    includePatterns: true
    maxSuggestions: 5
  ) {
    relevantChunks {
      score
      chunk {
        content
        summary
      }
    }
    suggestedTasks
    relatedConcepts
    potentialIssues
  }
}
```

### List Repository Chunks

```graphql
query ListChunks {
  listChunks(
    repository: "my-project"
    limit: 50
    offset: 0
  ) {
    id
    timestamp
    type
    summary
    tags
  }
}
```

### Get Patterns

```graphql
query GetPatterns {
  getPatterns(
    repository: "my-project"
    timeframe: "month"
  ) {
    name
    description
    occurrences
    confidence
    lastSeen
    examples
  }
}
```

## Global Memories

To store or search global memories (not tied to a specific repository), use `"_global"` as the repository name:

```graphql
mutation StoreGlobalMemory {
  storeChunk(input: {
    content: "Learned about GraphQL schema design patterns"
    sessionId: "learning-session"
    repository: "_global"
    tags: ["learning", "graphql", "patterns"]
  }) {
    id
    repository
  }
}
```

## Error Handling

The API returns standard GraphQL errors with meaningful messages:

```json
{
  "errors": [
    {
      "message": "Failed to generate embeddings: API rate limit exceeded",
      "path": ["search"]
    }
  ]
}
```

## Performance Considerations

1. **Pagination**: Use `limit` and `offset` for large result sets
2. **Caching**: Results are not cached by default; implement client-side caching as needed
3. **Rate Limiting**: The API respects underlying service rate limits (OpenAI, Chroma)
4. **Timeouts**: Queries have a 15-second timeout by default

## Security

1. **Authentication**: Currently no authentication (add as needed)
2. **CORS**: Configured to allow all origins (restrict in production)
3. **Input Validation**: All inputs are validated and sanitized
4. **Rate Limiting**: Implement rate limiting for production use