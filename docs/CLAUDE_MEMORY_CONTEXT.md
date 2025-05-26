# Claude Memory System Context

You have access to an intelligent memory system through MCP tools. This system helps you remember and learn from past conversations, solutions, and decisions.

## Quick Reference

### 🧠 Core Memory Tools

1. **Store memories** → `mcp__memory__memory_store_chunk`
   - Use after: solving bugs, making decisions, learning something new
   - Automatically: categorizes, generates embeddings, detects patterns

2. **Search memories** → `mcp__memory__memory_search`
   - Use for: finding similar problems, past solutions, decisions
   - Supports: natural language queries, semantic search

3. **Get context** → `mcp__memory__memory_get_context`
   - Use at: session start, project switches
   - Returns: recent activity, patterns, common issues

4. **Find similar problems** → `mcp__memory__memory_find_similar`
   - Use when: facing errors, complex problems
   - Returns: similar issues and their solutions

5. **Store decisions** → `mcp__memory__memory_store_decision`
   - Use for: architectural choices, design decisions
   - Includes: rationale and alternatives considered

## When to Use Memory

### 🔍 SEARCH before solving
```
Before fixing a bug or implementing a feature:
1. Search for similar past issues
2. Check for related decisions
3. Look for existing patterns
```

### 💾 STORE after accomplishing
```
After solving a problem or making progress:
1. Store the problem and solution
2. Include context and approach
3. Tag appropriately
```

### 📋 CONTEXT at start
```
When beginning a session:
1. Get repository context
2. Review recent activities
3. Check for ongoing work
```

## Memory Intelligence

The system automatically:
- **Categorizes** chunks as: problem, solution, decision, analysis, etc.
- **Links** related memories together
- **Detects** patterns and recurring issues
- **Prioritizes** recent and frequently accessed memories
- **Learns** from your problem-solving patterns

## Best Practices

### Good Memory Storage
```json
✅ GOOD: Detailed context with problem and solution
{
  "content": "Fixed connection timeout issue in ChromaDB by increasing timeout from 5s to 30s. The issue occurred under heavy load when vector operations took longer than expected. Added retry logic with exponential backoff.",
  "tags": ["bug-fix", "chromadb", "timeout", "performance"]
}

❌ POOR: Vague or minimal context
{
  "content": "Fixed the bug",
  "tags": ["fix"]
}
```

### Effective Searching
```json
✅ GOOD: Specific, contextual queries
{
  "query": "ChromaDB connection timeout under heavy load vector operations",
  "types": ["problem", "solution"]
}

❌ POOR: Too general
{
  "query": "error",
  "types": []
}
```

## Repository Strategy

- **Project-specific**: Use actual repository name for project-related memories
- **Global knowledge**: Use `_global` for cross-project learnings, patterns, and decisions
- **Automatic detection**: System learns patterns across repositories

## Memory Lifecycle

1. **Active** (0-7 days): Full detail, high priority
2. **Recent** (7-30 days): Normal access, standard priority  
3. **Historical** (30+ days): Lower priority, may be summarized
4. **Archived** (90+ days): Consolidated, pattern extraction

## Example Workflow

```python
# 1. Start of session
context = get_context(repository="my-project")
# Review recent work and patterns

# 2. Encountering a problem
similar = find_similar(problem="Getting CORS errors in API calls")
# Check past solutions

# 3. After solving
store_chunk(
    content="Solved CORS by adding proper headers...",
    tags=["cors", "api", "bug-fix"]
)

# 4. Making a decision
store_decision(
    decision="Use Redis for session storage",
    rationale="Need distributed sessions for scaling",
    context="Considered: local memory, database, Redis"
)
```

## Tips for Maximum Value

1. **Be descriptive**: Include the why, not just the what
2. **Tag consistently**: Develop a tagging taxonomy
3. **Store failures too**: Failed approaches are valuable learning
4. **Link issues**: Reference related problems or PRs
5. **Review patterns**: Periodically check recurring issues

Remember: The more context you provide, the more intelligent the memory system becomes at helping you in the future!