# MCP Memory User Guide

## Getting Started

MCP Memory helps AI assistants remember and learn from past conversations. This guide will help you understand how to use it effectively.

## Table of Contents

1. [Installation](#installation)
2. [Basic Usage](#basic-usage)
3. [Memory Organization](#memory-organization)
4. [Search and Retrieval](#search-and-retrieval)
5. [Best Practices](#best-practices)
6. [Advanced Features](#advanced-features)
7. [Troubleshooting](#troubleshooting)

## Installation

### Using Docker (Recommended)

```bash
# Clone the repository
git clone https://github.com/your-org/mcp-memory.git
cd mcp-memory

# Start with Docker Compose
docker-compose up -d
```

### Manual Installation

```bash
# Install Go 1.21+
# Install ChromaDB
pip install chromadb

# Clone and build
git clone https://github.com/your-org/mcp-memory.git
cd mcp-memory
go build ./cmd/server

# Run
./server
```

### Configuration

Create a `.env` file:
```bash
# Required
OPENAI_API_KEY=sk-...

# Optional
CHROMA_ENDPOINT=http://localhost:8000
CHROMA_COLLECTION=memories
MCP_PORT=3000
GRAPHQL_PORT=8082
OPENAPI_PORT=8081

# Performance
CHROMA_USE_POOLING=true
USE_CIRCUIT_BREAKER=true

# Storage
MCP_MEMORY_BACKUP_DIRECTORY=./backups
```

## Basic Usage

### Storing Memories

Memories are automatically stored when you interact with the AI. Each conversation chunk includes:
- The actual content
- AI-generated summary
- Extracted concepts and entities
- Metadata (timestamp, tags, tools used)

Example conversation that gets stored:
```
You: I'm implementing JWT authentication for our API. Should I use RS256 or HS256?

AI: For your API authentication, I recommend RS256 (RSA signatures) because...
[This entire exchange is automatically stored and indexed]
```

### Memory Types

1. **Conversations** - General discussions and Q&A
2. **Decisions** - Architectural and design decisions
3. **Problems** - Issues and their solutions
4. **Code** - Code snippets and implementations
5. **Documentation** - Docs and explanations
6. **Learning** - New concepts learned

### Global vs Repository Memories

- **Repository Memories**: Specific to a project
  ```
  Repository: my-project
  Content: "Fixed authentication bug in login.js"
  ```

- **Global Memories**: Shared across all projects
  ```
  Repository: _global
  Content: "Learned about new React 18 features"
  ```

## Memory Organization

### Session Management

Sessions group related conversations:
```
Session: feature-authentication-2024-01-15
├── Planning discussion
├── Implementation details
├── Bug fixes
└── Final review
```

### Tagging System

Use consistent tags for better organization:
- `#bug` - Bug fixes
- `#feature` - New features
- `#refactor` - Code refactoring
- `#learning` - Learning moments
- `#decision` - Architectural decisions

### File Association

Memories automatically track associated files:
```
Files Modified: [
  "src/auth/jwt.js",
  "src/middleware/auth.js",
  "tests/auth.test.js"
]
```

## Search and Retrieval

### Natural Language Search

Simply describe what you're looking for:
```
"How did we implement authentication?"
"What was that PostgreSQL optimization?"
"Show me all decisions about the API design"
```

### Filtered Search

Use filters for precise results:
```
Repository: my-project
Type: decision
Tags: [architecture, database]
Time: last_month
```

### Pattern Recognition

The system automatically identifies patterns:
- Recurring problems
- Common workflows
- Technology preferences
- Team practices

## Best Practices

### 1. Descriptive Conversations

Instead of:
```
"Fix the bug"
```

Use:
```
"Fix the authentication bug where JWT tokens expire too early in the login flow"
```

### 2. Document Decisions

When making decisions, include:
- The decision made
- Why it was made
- Alternatives considered
- Trade-offs accepted

Example:
```
Decision: Use PostgreSQL instead of MongoDB
Rationale: Need ACID compliance and complex queries
Alternatives: MongoDB (too eventual), MySQL (less features)
Trade-offs: Slightly more complex setup for better reliability
```

### 3. Link Related Concepts

Reference previous discussions:
```
"Building on our authentication discussion from last week..."
"Similar to the caching problem we solved in the user service..."
```

### 4. Regular Reviews

Periodically review stored memories:
- Identify outdated information
- Extract reusable patterns
- Build team knowledge base

## Advanced Features

### Context Suggestions

The AI proactively suggests relevant context:
```
Current: "Working on user authentication"
Suggestions:
- Previous auth implementation in project-x
- Security best practices discussed last month
- Similar OAuth setup from project-y
```

### Problem Matching

When describing a problem, find similar past issues:
```
Problem: "Getting CORS errors in production"
Similar Issues:
1. "CORS configuration for API Gateway" (95% match)
2. "Production CORS with load balancer" (87% match)
```

### Workflow Learning

The system learns your development patterns:
```
Detected Workflow: Feature Implementation
1. Create design document
2. Write tests
3. Implement feature
4. Code review
5. Deploy to staging
```

### Memory Chains

Related memories are automatically linked:
```
Memory Chain: Authentication Implementation
├── Initial planning meeting
├── Security review
├── Implementation phase
├── Bug fixes
├── Performance optimization
└── Documentation
```

## Troubleshooting

### Common Issues

#### 1. Memories Not Being Stored
- Check OpenAI API key is valid
- Verify ChromaDB is running
- Look for errors in logs

#### 2. Search Not Finding Expected Results
- Memories may need time to index
- Try broader search terms
- Check repository filter

#### 3. Slow Performance
- Enable connection pooling
- Increase resource limits
- Check ChromaDB performance

### Debug Mode

Enable debug logging:
```bash
export LOG_LEVEL=debug
export DEBUG_MCP=true
```

### Health Checks

Check system status:
```bash
curl http://localhost:8081/health
```

### Backup and Restore

Backup memories:
```bash
# Automatic backups
export MCP_MEMORY_BACKUP_SCHEDULE="0 2 * * *"  # 2 AM daily

# Manual backup
curl -X POST http://localhost:8081/api/v1/backup
```

Restore from backup:
```bash
./restore.sh backup-2024-01-15.tar.gz
```

## Privacy and Security

### Data Privacy

- Memories are stored locally by default
- No data sent to external services except OpenAI for embeddings
- PII detection and optional masking

### Access Control

- Repository-based isolation
- Session-based grouping
- API key authentication

### Data Retention

Configure retention policies:
```yaml
retention:
  default: 365  # days
  repositories:
    temp-project: 30
    archive-project: 0  # keep forever
```

## Integration Examples

### With Claude Desktop

```json
{
  "memory_config": {
    "enabled": true,
    "repository": "my-project",
    "auto_store": true,
    "search_before_response": true
  }
}
```

### With VS Code

Install the MCP Memory extension:
1. Search for "MCP Memory" in extensions
2. Configure API endpoint
3. Use Command Palette: "MCP: Search Memory"

### With CLI Tools

```bash
# Search memories
mcp-memory search "authentication bug"

# Store a decision
mcp-memory decide "Use Redis for caching" \
  --rationale "Need fast key-value store" \
  --repo my-project

# Export memories
mcp-memory export --repo my-project --format markdown
```

## Tips for Effective Use

1. **Be Specific**: Include details that will help future you
2. **Tag Consistently**: Develop a tagging taxonomy
3. **Review Regularly**: Memories are most valuable when reviewed
4. **Share Knowledge**: Export and share useful patterns with your team
5. **Prune Outdated**: Remove or update obsolete information
6. **Link Contexts**: Reference related discussions and decisions

## Keyboard Shortcuts (VS Code Extension)

- `Cmd/Ctrl + Shift + M` - Search memories
- `Cmd/Ctrl + Shift + D` - Store decision
- `Cmd/Ctrl + Shift + R` - Show related context
- `Cmd/Ctrl + Shift + P` - Find similar problems

## Getting Help

- **Documentation**: https://docs.mcp-memory.dev
- **GitHub Issues**: https://github.com/your-org/mcp-memory/issues
- **Discord Community**: https://discord.gg/mcp-memory
- **Email Support**: support@mcp-memory.dev