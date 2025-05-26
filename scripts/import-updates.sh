#!/bin/bash

# Import memories about the mcp-memory updates we implemented

GRAPHQL_ENDPOINT="http://localhost:8082/graphql"
SESSION_ID="mcp-updates-$(date +%Y-%m-%d)"
REPOSITORY="mcp-memory"

echo "🚀 Importing memories about mcp-memory updates..."

# GraphQL Implementation
curl -X POST "$GRAPHQL_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d '{
        "query": "mutation { storeChunk(input: { content: \"Implemented GraphQL API server for mcp-memory. Created comprehensive GraphQL schema with queries (search, getChunk, listChunks, traceSession, traceRelated) and mutations (storeChunk, storeDecision, deleteChunk). Server runs on port 8082 with GraphiQL playground.\", sessionId: \"'"$SESSION_ID"'\", repository: \"'"$REPOSITORY"'\", tags: [\"implementation\", \"graphql\", \"api\"] }) { id } }"
    }' > /dev/null 2>&1

# Web UI Implementation
curl -X POST "$GRAPHQL_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d '{
        "query": "mutation { storeChunk(input: { content: \"Created modern web UI for memory browsing. Features include: memory list with search and filters, detailed memory view with metadata, session tracing with timeline visualization, relationship discovery with graph visualization. Built with vanilla JavaScript and Canvas API for visualizations.\", sessionId: \"'"$SESSION_ID"'\", repository: \"'"$REPOSITORY"'\", tags: [\"implementation\", \"web-ui\", \"visualization\"] }) { id } }"
    }' > /dev/null 2>&1

# Memory Tracing Feature
curl -X POST "$GRAPHQL_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d '{
        "query": "mutation { storeChunk(input: { content: \"Implemented memory tracing functionality. Added traceSession query to get all memories from a session chronologically. Added traceRelated query using BFS algorithm to find connected memories up to specified depth. Includes timeline visualization for sessions and circular graph layout for relationships.\", sessionId: \"'"$SESSION_ID"'\", repository: \"'"$REPOSITORY"'\", tags: [\"feature\", \"tracing\", \"visualization\", \"algorithm\"] }) { id } }"
    }' > /dev/null 2>&1

# ChromaDB Persistence Fix
curl -X POST "$GRAPHQL_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d '{
        "query": "mutation { storeChunk(input: { content: \"Fixed ChromaDB persistence issue. ChromaDB was saving to /data instead of mounted volume. Solution: Added command parameter in docker-compose.yml with --path /chroma/chroma. Now data persists in named volume mcp_memory_chroma_vector_db_NEVER_DELETE. SQLite file grows from 160K to 360K+ with data.\", sessionId: \"'"$SESSION_ID"'\", repository: \"'"$REPOSITORY"'\", tags: [\"bug-fix\", \"chromadb\", \"persistence\", \"docker\"] }) { id } }"
    }' > /dev/null 2>&1

# Storage Layer Updates
curl -X POST "$GRAPHQL_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d '{
        "query": "mutation { storeChunk(input: { content: \"Extended VectorStore interface with ListBySession method. Implemented in ChromaStore, PooledChromaStore, RetryWrapper, and CircuitBreakerWrapper. Fixed validation by adding default Outcome and Difficulty values in storeChunkResolver. Added embeddings generation before storing chunks.\", sessionId: \"'"$SESSION_ID"'\", repository: \"'"$REPOSITORY"'\", tags: [\"implementation\", \"storage\", \"interface\"] }) { id } }"
    }' > /dev/null 2>&1

# Documentation Updates
curl -X POST "$GRAPHQL_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d '{
        "query": "mutation { storeChunk(input: { content: \"Comprehensive documentation update. Created GRAPHQL_WEB_UI.md with full API reference. Updated README.md with GraphQL section and marked MCP tools as legacy. Updated DEPLOYMENT.md with GraphQL server and ChromaDB persistence sections. Rewrote api-reference.md with GraphQL focus and migration guide.\", sessionId: \"'"$SESSION_ID"'\", repository: \"'"$REPOSITORY"'\", tags: [\"documentation\", \"graphql\", \"deployment\"] }) { id } }"
    }' > /dev/null 2>&1

# Import Scripts
curl -X POST "$GRAPHQL_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d '{
        "query": "mutation { storeChunk(input: { content: \"Created import scripts for bulk memory creation. Scripts use GraphQL mutations to store memories with proper tags and metadata. Successfully imported 16 memories about mcp-memory documentation. Database size increased from 160K to 360K confirming data storage.\", sessionId: \"'"$SESSION_ID"'\", repository: \"'"$REPOSITORY"'\", tags: [\"tooling\", \"import\", \"automation\"] }) { id } }"
    }' > /dev/null 2>&1

# Architecture Decisions
curl -X POST "$GRAPHQL_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d '{
        "query": "mutation { storeChunk(input: { content: \"Architecture Decision: Use GraphQL instead of REST for flexibility. GraphQL provides type safety, single endpoint, flexible queries, and built-in documentation through introspection. Chose vanilla JavaScript for web UI to minimize dependencies. Used Canvas API for custom visualizations.\", sessionId: \"'"$SESSION_ID"'\", repository: \"'"$REPOSITORY"'\", tags: [\"architecture\", \"graphql\", \"web-ui\", \"decision\"] }) { id } }"
    }' > /dev/null 2>&1

# Port Configuration
curl -X POST "$GRAPHQL_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d '{
        "query": "mutation { storeChunk(input: { content: \"Updated port configuration: 8080 for MCP API, 8081 for health checks, 8082 for GraphQL/Web UI (changed from metrics), 9000 for ChromaDB host port (maps to container 8000), 9090 for Prometheus metrics. Updated all documentation to reflect new port assignments.\", sessionId: \"'"$SESSION_ID"'\", repository: \"'"$REPOSITORY"'\", tags: [\"configuration\", \"ports\", \"networking\"] }) { id } }"
    }' > /dev/null 2>&1

# Testing and Verification
curl -X POST "$GRAPHQL_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d '{
        "query": "mutation { storeChunk(input: { content: \"Verified complete system functionality: ChromaDB persistence works across restarts, GraphQL queries return correct data, Web UI displays memories with search and filters, Memory tracing shows proper timeline and relationships, Import scripts successfully populate database. All features tested and working.\", sessionId: \"'"$SESSION_ID"'\", repository: \"'"$REPOSITORY"'\", tags: [\"testing\", \"verification\", \"qa\"] }) { id } }"
    }' > /dev/null 2>&1

# Lessons Learned
curl -X POST "$GRAPHQL_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d '{
        "query": "mutation { storeChunk(input: { content: \"Lessons learned: ChromaDB uses SQLite internally for metadata storage. The PERSIST_DIRECTORY env var alone is insufficient - must use --path command. GraphQL provides excellent developer experience with playground. Canvas API works well for simple graph visualizations. Proper validation (Outcome, Difficulty) is critical for chunk storage.\", sessionId: \"'"$SESSION_ID"'\", repository: \"'"$REPOSITORY"'\", tags: [\"lessons-learned\", \"chromadb\", \"graphql\", \"debugging\"] }) { id } }"
    }' > /dev/null 2>&1

# Future Enhancements
curl -X POST "$GRAPHQL_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d '{
        "query": "mutation { storeChunk(input: { content: \"Potential future enhancements: Real-time GraphQL subscriptions for live updates, Advanced graph layouts (force-directed, hierarchical), Memory editing capabilities in UI, Bulk import/export with progress tracking, Authentication and multi-user support, Memory versioning and history tracking.\", sessionId: \"'"$SESSION_ID"'\", repository: \"'"$REPOSITORY"'\", tags: [\"roadmap\", \"future\", \"enhancements\"] }) { id } }"
    }' > /dev/null 2>&1

echo "✅ Update memories import complete!"

# Verify import
CHUNK_COUNT=$(curl -s -X POST "$GRAPHQL_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d "{\"query\": \"{ listChunks(repository: \\\"$REPOSITORY\\\", limit: 100) { id } }\"}" \
    | jq '.data.listChunks | length')

echo "📊 Total memories in repository: $CHUNK_COUNT"

# Show recent memories from this session
echo ""
echo "📝 Recent memories from this update session:"
curl -s -X POST "$GRAPHQL_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d "{\"query\": \"{ traceSession(sessionId: \\\"$SESSION_ID\\\") { content type timestamp } }\"}" \
    | jq -r '.data.traceSession[:3][] | "[\(.type)] \(.content | split(" ")[0:10] | join(" "))..."'