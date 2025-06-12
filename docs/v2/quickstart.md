# MCP Memory Server v2 - Quick Start Guide

## 5-Minute Setup

Get started with the MCP Memory Server v2 in just 5 minutes. This guide will walk you through installation, basic configuration, and your first memory operations.

## Prerequisites

- Go 1.23 or later
- Docker and Docker Compose
- OpenAI API key (for embeddings)

## Installation

### Option 1: Docker (Recommended)

The fastest way to get started:

```bash
# Clone the repository
git clone https://github.com/lerian/mcp-memory-server.git
cd mcp-memory-server

# Copy environment template
cp .env.example .env

# Edit .env and add your OpenAI API key
echo "OPENAI_API_KEY=your-api-key-here" >> .env

# Start the server
make docker-up
```

### Option 2: Local Development

For development or customization:

```bash
# Clone and setup
git clone https://github.com/lerian/mcp-memory-server.git
cd mcp-memory-server
make setup-env

# Add your OpenAI API key to .env
echo "OPENAI_API_KEY=your-api-key-here" >> .env

# Start development mode
make dev
```

## Environment Configuration

Edit your `.env` file with these essential settings:

```bash
# Required
OPENAI_API_KEY=your-api-key-here

# Server Configuration
MCP_HOST_PORT=9080
QDRANT_HOST_PORT=6333

# Memory Settings
MCP_MEMORY_LOG_LEVEL=info
MCP_MEMORY_VECTOR_DIM=1536
```

## Verify Installation

Check that the server is running:

```bash
# Health check
curl http://localhost:9080/health

# Expected response:
{
  "status": "healthy",
  "version": "v2.0.0",
  "timestamp": "2024-12-06T15:30:45Z"
}
```

## Your First Memory Operations

### 1. Store Your First Content

```bash
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_store",
      "arguments": {
        "operation": "store_content",
        "project_id": "quickstart-project",
        "session_id": "my-session",
        "content": "The MCP Memory Server v2 provides intelligent memory capabilities for AI assistants. It supports semantic search, relationship detection, and cross-domain operations.",
        "tags": ["documentation", "memory", "ai"],
        "options": {
          "generate_embeddings": true
        }
      }
    },
    "id": 1
  }'
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "success": true,
    "message": "Content stored successfully",
    "content_id": "content_abc123",
    "timestamp": "2024-12-06T15:30:45Z"
  },
  "id": 1
}
```

### 2. Search Your Content

```bash
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_retrieve",
      "arguments": {
        "operation": "search_content",
        "project_id": "quickstart-project",
        "query": "semantic search capabilities",
        "options": {
          "limit": 5
        }
      }
    },
    "id": 2
  }'
```

### 3. Store a Decision

```bash
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_store",
      "arguments": {
        "operation": "store_decision",
        "project_id": "quickstart-project",
        "session_id": "my-session",
        "title": "Architecture: Choose Vector Database",
        "context": "We need a vector database for semantic search capabilities in our memory server.",
        "decision": "Use Qdrant as the vector database",
        "rationale": "Qdrant provides excellent performance, supports clustering, and has a clean API that integrates well with our Go stack.",
        "alternatives": ["Pinecone", "Weaviate", "Chroma"],
        "tags": ["architecture", "database", "vectors"]
      }
    },
    "id": 3
  }'
```

### 4. Analyze Content Quality

```bash
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_analyze",
      "arguments": {
        "operation": "analyze_quality",
        "project_id": "quickstart-project",
        "content_id": "content_abc123",
        "options": {
          "include_suggestions": true
        }
      }
    },
    "id": 4
  }'
```

### 5. Check System Health

```bash
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_system",
      "arguments": {
        "operation": "check_system_health",
        "detailed": true
      }
    },
    "id": 5
  }'
```

## Understanding the 4-Tool Architecture

The MCP Memory Server v2 organizes all functionality into 4 logical tools:

### 1. memory_store
**Purpose**: All data persistence operations
- `store_content` - Store new content
- `update_content` - Modify existing content  
- `delete_content` - Remove content
- `store_decision` - Store important decisions
- `create_relationship` - Link content items

### 2. memory_retrieve
**Purpose**: All data retrieval operations
- `search_content` - Semantic and keyword search
- `get_content` - Retrieve specific content
- `find_similar_content` - Find related content
- `get_content_history` - Version history

### 3. memory_analyze
**Purpose**: All analysis and intelligence operations
- `detect_patterns` - Identify content patterns
- `analyze_quality` - Content quality analysis
- `find_content_relationships` - Discover relationships
- `detect_conflicts` - Find conflicting information
- `generate_insights` - AI-powered insights

### 4. memory_system
**Purpose**: All system administration operations
- `check_system_health` - System status
- `export_project_data` - Data export
- `import_project_data` - Data import
- `validate_data_integrity` - Data validation
- `generate_citation` - Academic citations

## Key Concepts

### Project Isolation

All data is organized by `project_id` for complete tenant isolation:

```json
{
  "project_id": "my-project",  // Required for data isolation
  "session_id": "session-123"  // Optional for expanded access
}
```

### Session Semantics

Sessions provide **MORE** access, not less:

- **Without session_id**: Read-only access to project data
- **With session_id**: Full access to session data + project data

### Clean Domain Separation

The v2 architecture separates concerns into distinct domains:

- **Memory Domain**: Content storage and knowledge management
- **Task Domain**: Task management and workflows  
- **System Domain**: Administration and system operations
- **Cross-Domain**: Coordinated operations between domains

## Common Workflows

### Content Management Workflow

```bash
# 1. Store content
store_content → content_id

# 2. Search and discover
search_content → find related content

# 3. Analyze and improve
analyze_quality → get improvement suggestions

# 4. Update based on analysis
update_content → improved content
```

### Decision Documentation Workflow

```bash
# 1. Store decision
store_decision → decision_id

# 2. Link to supporting content
create_relationship → decision ↔ content

# 3. Track decision impact
detect_patterns → decision impact analysis

# 4. Generate insights
generate_insights → decision effectiveness
```

### Knowledge Discovery Workflow

```bash
# 1. Search for information
search_content → relevant content

# 2. Find similar content
find_similar_content → related information

# 3. Detect relationships
find_content_relationships → knowledge graph

# 4. Generate insights
generate_insights → knowledge synthesis
```

## Next Steps

Now that you have the basics working, explore these areas:

### 1. Advanced Search
Learn about semantic search, filters, and relevance tuning:
```bash
# Semantic search with filters
{
  "operation": "search_content",
  "query": "machine learning algorithms",
  "filters": {
    "tags": ["ai", "ml"],
    "date_range": {
      "start": "2024-01-01",
      "end": "2024-12-31"
    }
  },
  "options": {
    "query_type": "semantic",
    "min_relevance": 0.7
  }
}
```

### 2. Relationship Management
Build knowledge graphs through content relationships:
```bash
# Create semantic relationship
{
  "operation": "create_relationship",
  "source_id": "content_123",
  "target_id": "content_456", 
  "type": "implements",
  "strength": 0.9
}
```

### 3. Quality Analysis
Improve content quality with AI-powered analysis:
```bash
# Comprehensive quality analysis
{
  "operation": "analyze_quality",
  "content_id": "content_123",
  "quality_dimensions": ["clarity", "completeness", "accuracy"],
  "options": {
    "include_suggestions": true,
    "benchmark_against": ["content_456", "content_789"]
  }
}
```

### 4. System Integration
Monitor and maintain your memory system:
```bash
# Comprehensive health check
{
  "operation": "check_system_health",
  "detailed": true,
  "components": ["database", "vector_store", "ai_service"]
}
```

## CLI Usage

For command-line interaction, use the integrated CLI:

```bash
# Install CLI (if not using Docker)
make build
./lmmc --help

# Store content via CLI
./lmmc store --project quickstart-project --content "Hello, world!"

# Search via CLI  
./lmmc search --project quickstart-project --query "hello"

# Interactive mode
./lmmc interactive --project quickstart-project
```

## Integration Examples

### Claude Desktop Integration

Add to your Claude Desktop MCP configuration:

```json
{
  "mcp": {
    "servers": {
      "memory": {
        "command": "docker",
        "args": ["exec", "-i", "mcp-memory-server", "./mcp-proxy.js"],
        "env": {
          "MCP_MEMORY_PROJECT_ID": "claude-workspace"
        }
      }
    }
  }
}
```

### VS Code Extension

Install the MCP Memory extension for VS Code:

```json
{
  "mcp.memory.endpoint": "http://localhost:9080/mcp",
  "mcp.memory.defaultProject": "vscode-workspace",
  "mcp.memory.autoStore": true
}
```

### Custom Application

Integrate with your application using the SDK:

```typescript
import { MCPMemoryClient } from '@lerian/mcp-memory-client';

const memory = new MCPMemoryClient({
  endpoint: 'ws://localhost:9080/ws',
  project_id: 'my-app'
});

// Store user input
await memory.store({
  content: userInput,
  tags: ['user-input', 'conversation']
});

// Search for context
const context = await memory.search({
  query: userQuery,
  limit: 5
});
```

## Troubleshooting

### Common Issues

**Server won't start:**
```bash
# Check logs
docker logs mcp-memory-server

# Common fix: reset environment
make docker-down && make docker-up
```

**Search returns no results:**
```bash
# Verify content is stored
curl -X POST http://localhost:9080/mcp -d '{
  "jsonrpc": "2.0",
  "method": "tools/call", 
  "params": {
    "name": "memory_retrieve",
    "arguments": {
      "operation": "get_content",
      "project_id": "your-project",
      "content_id": "your-content-id"
    }
  },
  "id": 1
}'
```

**Missing OpenAI API key:**
```bash
# Add to .env file
echo "OPENAI_API_KEY=your-key-here" >> .env
make docker-restart
```

### Getting Help

- **Documentation**: Read the complete [API Reference](./api-reference.md)
- **Data Model**: Understand the [Data Model](./data-model.md)  
- **Issues**: Report issues on [GitHub](https://github.com/lerian/mcp-memory-server/issues)
- **Discussions**: Join [GitHub Discussions](https://github.com/lerian/mcp-memory-server/discussions)

## What's Next?

You now have a working MCP Memory Server v2 with:
- ✅ Content storage with semantic search
- ✅ Decision documentation
- ✅ Quality analysis
- ✅ System monitoring

Explore advanced features:
- **Tutorials**: Step-by-step guides for complex workflows
- **Integration Guides**: Connect with popular tools and platforms
- **Best Practices**: Optimize performance and organization
- **API Reference**: Complete documentation of all operations

Happy memory building! 🧠✨