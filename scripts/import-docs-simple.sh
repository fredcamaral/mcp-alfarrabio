#!/bin/bash

# Import MCP Memory documentation into the memory system

GRAPHQL_ENDPOINT="http://localhost:8082/graphql"
SESSION_ID="docs-import-2025-05-26"
REPOSITORY="mcp-memory"

echo "🚀 Starting documentation import..."

# Overview memories
curl -X POST "$GRAPHQL_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d '{
        "query": "mutation { storeChunk(input: { content: \"MCP Memory Server is a semantic memory system for AI assistants that provides persistent, intelligent context management. It uses ChromaDB for vector storage, supports GraphQL API, and includes features like pattern recognition, context suggestion, and memory tracing.\", sessionId: \"'"$SESSION_ID"'\", repository: \"'"$REPOSITORY"'\", tags: [\"overview\", \"architecture\"] }) { id } }"
    }' > /dev/null 2>&1

curl -X POST "$GRAPHQL_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d '{
        "query": "mutation { storeChunk(input: { content: \"The project includes comprehensive documentation covering getting started guides, development setup with hot reload, production deployment instructions, monitoring with Prometheus and Grafana, IDE integrations for VS Code and IntelliJ, and marketing materials.\", sessionId: \"'"$SESSION_ID"'\", repository: \"'"$REPOSITORY"'\", tags: [\"documentation\", \"structure\"] }) { id } }"
    }' > /dev/null 2>&1

# Architecture decisions
curl -X POST "$GRAPHQL_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d '{
        "query": "mutation { storeChunk(input: { content: \"Architecture Decision: Use ChromaDB as the vector database for storing embeddings. ChromaDB provides efficient similarity search, persistent storage, and easy integration with Go applications.\", sessionId: \"'"$SESSION_ID"'\", repository: \"'"$REPOSITORY"'\", tags: [\"architecture\", \"chromadb\", \"vector-database\"] }) { id } }"
    }' > /dev/null 2>&1

curl -X POST "$GRAPHQL_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d '{
        "query": "mutation { storeChunk(input: { content: \"Architecture Decision: Implement GraphQL API for flexible querying. GraphQL allows clients to request exactly the data they need, supports real-time subscriptions, and provides strong typing.\", sessionId: \"'"$SESSION_ID"'\", repository: \"'"$REPOSITORY"'\", tags: [\"architecture\", \"graphql\", \"api\"] }) { id } }"
    }' > /dev/null 2>&1

curl -X POST "$GRAPHQL_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d '{
        "query": "mutation { storeChunk(input: { content: \"Architecture Decision: Use Model Context Protocol (MCP) for AI integration. MCP provides a standardized way for AI assistants to interact with external tools and data sources.\", sessionId: \"'"$SESSION_ID"'\", repository: \"'"$REPOSITORY"'\", tags: [\"architecture\", \"mcp\", \"ai-integration\"] }) { id } }"
    }' > /dev/null 2>&1

# Features
curl -X POST "$GRAPHQL_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d '{
        "query": "mutation { storeChunk(input: { content: \"Feature: Memory Tracing - Ability to trace memories by session or find related memories using semantic similarity. Includes timeline visualization for session traces and relationship graphs for connected memories.\", sessionId: \"'"$SESSION_ID"'\", repository: \"'"$REPOSITORY"'\", tags: [\"feature\", \"tracing\", \"visualization\"] }) { id } }"
    }' > /dev/null 2>&1

curl -X POST "$GRAPHQL_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d '{
        "query": "mutation { storeChunk(input: { content: \"Feature: Pattern Recognition - Automatically identifies recurring patterns in stored memories, helping to surface common workflows, repeated problems, and established solutions.\", sessionId: \"'"$SESSION_ID"'\", repository: \"'"$REPOSITORY"'\", tags: [\"feature\", \"pattern-recognition\", \"intelligence\"] }) { id } }"
    }' > /dev/null 2>&1

curl -X POST "$GRAPHQL_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d '{
        "query": "mutation { storeChunk(input: { content: \"Feature: Context Suggestion - Proactively suggests relevant memories based on current context, helping AI assistants maintain continuity across conversations.\", sessionId: \"'"$SESSION_ID"'\", repository: \"'"$REPOSITORY"'\", tags: [\"feature\", \"context-suggestion\", \"intelligence\"] }) { id } }"
    }' > /dev/null 2>&1

# Deployment
curl -X POST "$GRAPHQL_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d '{
        "query": "mutation { storeChunk(input: { content: \"Deployment: Docker Compose setup with ChromaDB, MCP Memory Server, monitoring stack (Prometheus, Grafana), and Traefik for routing. Volumes are configured for data persistence with named volumes that should never be deleted.\", sessionId: \"'"$SESSION_ID"'\", repository: \"'"$REPOSITORY"'\", tags: [\"deployment\", \"docker\", \"infrastructure\"] }) { id } }"
    }' > /dev/null 2>&1

curl -X POST "$GRAPHQL_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d '{
        "query": "mutation { storeChunk(input: { content: \"Deployment: ChromaDB persistence requires explicit --path command in docker-compose.yml. Data is stored in /chroma/chroma directory mounted to mcp_memory_chroma_vector_db_NEVER_DELETE volume.\", sessionId: \"'"$SESSION_ID"'\", repository: \"'"$REPOSITORY"'\", tags: [\"deployment\", \"chromadb\", \"persistence\", \"configuration\"] }) { id } }"
    }' > /dev/null 2>&1

# Development
curl -X POST "$GRAPHQL_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d '{
        "query": "mutation { storeChunk(input: { content: \"Development: Hot reload setup using Air for automatic server restarts. Configuration in .air.toml watches for changes in .go files and rebuilds the server. Excludes vendor, tmp, and other non-source directories.\", sessionId: \"'"$SESSION_ID"'\", repository: \"'"$REPOSITORY"'\", tags: [\"development\", \"hot-reload\", \"dx\"] }) { id } }"
    }' > /dev/null 2>&1

curl -X POST "$GRAPHQL_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d '{
        "query": "mutation { storeChunk(input: { content: \"Development: GraphQL server runs on port 8082 with web UI for memory browsing, GraphiQL playground for API exploration, and REST endpoints for health checks.\", sessionId: \"'"$SESSION_ID"'\", repository: \"'"$REPOSITORY"'\", tags: [\"development\", \"graphql\", \"api\"] }) { id } }"
    }' > /dev/null 2>&1

# Roadmap
curl -X POST "$GRAPHQL_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d '{
        "query": "mutation { storeChunk(input: { content: \"Roadmap: Priority 1 - Bug fixes including ChromaDB v2 API migration, error handling improvements, graceful shutdown, and memory leak prevention.\", sessionId: \"'"$SESSION_ID"'\", repository: \"'"$REPOSITORY"'\", tags: [\"roadmap\", \"priority-1\", \"bugs\"] }) { id } }"
    }' > /dev/null 2>&1

curl -X POST "$GRAPHQL_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d '{
        "query": "mutation { storeChunk(input: { content: \"Roadmap: Priority 2 - Testing infrastructure including unit tests for all services, integration tests for API endpoints, and performance benchmarks.\", sessionId: \"'"$SESSION_ID"'\", repository: \"'"$REPOSITORY"'\", tags: [\"roadmap\", \"priority-2\", \"testing\"] }) { id } }"
    }' > /dev/null 2>&1

curl -X POST "$GRAPHQL_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d '{
        "query": "mutation { storeChunk(input: { content: \"Roadmap: Priority 3 - AI enhancements including better chunking strategies, multi-modal support, and improved pattern recognition algorithms.\", sessionId: \"'"$SESSION_ID"'\", repository: \"'"$REPOSITORY"'\", tags: [\"roadmap\", \"priority-3\", \"ai-features\"] }) { id } }"
    }' > /dev/null 2>&1

echo "✅ Documentation import complete!"

# Verify import
CHUNK_COUNT=$(curl -s -X POST "$GRAPHQL_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d "{\"query\": \"{ listChunks(repository: \\\"$REPOSITORY\\\", limit: 100) { id } }\"}" \
    | jq '.data.listChunks | length')

echo "✨ Successfully imported $CHUNK_COUNT memories!"

# Check ChromaDB persistence
echo ""
echo "📊 Checking ChromaDB persistence..."
docker exec mcp-chroma du -h /chroma/chroma/chroma.sqlite3