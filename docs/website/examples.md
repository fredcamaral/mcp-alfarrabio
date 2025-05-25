# MCP Memory Examples

Real-world examples of using MCP Memory to enhance your development workflow.

## Basic Usage Examples

### Storing Development Context

```python
# When implementing a new feature
"Store this conversation about implementing OAuth2 authentication with Google provider"

# After solving a complex bug
"Store our debugging session for the race condition in the order processing system"

# Recording architectural decisions
"Store this discussion about choosing PostgreSQL over MongoDB for our analytics data"
```

### Searching Past Knowledge

```python
# Finding similar problems
"Search for previous issues with database connection pooling"

# Looking up past decisions
"What did we decide about API versioning strategy?"

# Finding implementation examples
"Show me how we've implemented caching in other services"
```

## Development Workflow Examples

### 1. Debugging Production Issues

**Scenario**: Production API is returning 500 errors intermittently.

```python
# First, check if we've seen this before
"Find similar problems with intermittent 500 errors in the API"

# After investigation
"Store this debugging session: intermittent 500 errors caused by connection pool exhaustion during peak traffic. Fixed by increasing pool size from 10 to 50 and implementing connection timeout of 30s"

# Tag for future reference
tags: ["production-issue", "database", "connection-pool"]
```

### 2. Code Review Patterns

**Scenario**: Reviewing PRs and maintaining coding standards.

```python
# During code review
"Store code review feedback: always use context.Context for API calls, implement proper error wrapping, and add request ID to all log entries"

# Later, when reviewing similar code
"What are our standards for API error handling?"

# Building review checklist
"Show me common code review feedback for API endpoints"
```

### 3. Onboarding New Team Members

**Scenario**: New developer joining the team.

```python
# Get project context
"Get project context for the payment-service repository"

# Show architectural decisions
"What are the key architectural decisions for our microservices?"

# Common gotchas
"What are common issues new developers face in this project?"
```

## Advanced Usage Examples

### 1. Pattern Recognition for Performance

```python
# After multiple performance optimizations
"Get patterns for performance-related changes in the last month"

# MCP Memory identifies:
# - Most performance issues involve N+1 queries
# - Caching added to 5 endpoints reduced load by 60%
# - Database indexes were missing on foreign keys
```

### 2. Cross-Repository Learning

```python
# Working on authentication in a new service
"How have we implemented JWT authentication in other services?"

# MCP Memory returns:
# - auth-service: Custom JWT with refresh tokens
# - api-gateway: JWT validation middleware
# - user-service: JWT generation with claims
```

### 3. Architecture Evolution Tracking

```python
# Tracking architecture changes over time
"Show me how our authentication architecture has evolved"

# Timeline:
# - Jan 2024: Started with session-based auth
# - Mar 2024: Migrated to JWT for stateless auth
# - Jun 2024: Added OAuth2 providers
# - Sep 2024: Implemented SSO with SAML
```

## Integration Examples

### With CI/CD Pipelines

```yaml
# .github/workflows/memory-store.yml
name: Store Deployment Context
on:
  deployment:
    types: [completed]

jobs:
  store-context:
    runs-on: ubuntu-latest
    steps:
      - name: Store deployment info
        run: |
          mcp-memory store \
            --content "Deployed ${{ github.sha }} to ${{ github.event.deployment.environment }}" \
            --tags "deployment,${{ github.event.deployment.environment }}" \
            --repository ${{ github.repository }}
```

### With Development Tools

```bash
# Git hook to store commit context
# .git/hooks/post-commit
#!/bin/bash

COMMIT_MSG=$(git log -1 --pretty=%B)
CHANGED_FILES=$(git diff-tree --no-commit-id --name-only -r HEAD)

mcp-memory store \
  --content "Commit: $COMMIT_MSG\nFiles: $CHANGED_FILES" \
  --session-id "git-commits-$(date +%Y%m)" \
  --repository $(basename $(pwd))
```

### With IDE Extensions

```javascript
// VS Code extension snippet
vscode.commands.registerCommand('mcp-memory.storeSelection', async () => {
  const editor = vscode.window.activeTextEditor;
  const selection = editor.document.getText(editor.selection);
  
  await mcpMemory.store({
    content: selection,
    session_id: `vscode-${Date.now()}`,
    files_modified: [editor.document.fileName],
    tags: ['code-snippet', editor.document.languageId]
  });
});
```

## Workflow Automation Examples

### 1. Daily Standup Helper

```python
# At the start of each day
"What did I work on yesterday in the payment-service?"

# Get's yesterday's context:
# - Fixed race condition in payment processing
# - Added retry logic for failed transactions
# - Updated API documentation
```

### 2. Sprint Retrospective

```python
# At the end of sprint
"Get patterns for the last two weeks in all repositories"

# Identifies:
# - 15 authentication-related changes
# - 8 performance optimizations
# - 5 bug fixes related to data validation
# - Suggestion: Consider dedicated sprint for auth improvements
```

### 3. Knowledge Base Building

```python
# Periodically extract learned patterns
"Export all solutions for database-related problems as markdown"

# Creates a knowledge base with:
# - Common PostgreSQL issues and fixes
# - Query optimization techniques used
# - Migration best practices discovered
```

## Tips and Tricks

### 1. Effective Tagging

```python
# Use hierarchical tags
tags: ["bug", "bug:performance", "bug:performance:database"]

# Use consistent naming
tags: ["api", "api-v2", "rest-api"]  # Bad
tags: ["api", "api", "api"]          # Good (same tag for all API-related)
```

### 2. Session Management

```python
# Group related work
session_id: "feature-user-auth-2024-01"
session_id: "bugfix-payment-race-2024-01"
session_id: "refactor-api-structure-2024-02"
```

### 3. Search Optimization

```python
# Be specific in queries
"database connection error"           # Too generic
"PostgreSQL connection pool timeout error in order service"  # Better

# Use technical terms
"slow API"                           # Vague
"API endpoint latency above 500ms"   # Specific
```

## Common Patterns

### Error Resolution Pattern

1. Error occurs
2. Search for similar errors
3. Apply suggested solutions
4. Store successful resolution
5. Tag with error type and solution

### Feature Development Pattern

1. Get context for similar features
2. Review past architectural decisions
3. Implement feature
4. Store implementation details
5. Document decision rationale

### Performance Optimization Pattern

1. Identify performance issue
2. Search for similar optimizations
3. Apply relevant techniques
4. Measure improvements
5. Store results with metrics

---

Ready to try these examples? Check our [Getting Started Guide](getting-started.md) to set up MCP Memory!