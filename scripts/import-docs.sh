#!/bin/bash

# Import MCP Memory documentation into the memory system

GRAPHQL_ENDPOINT="http://localhost:8082/graphql"
SESSION_ID="docs-import-$(date +%Y-%m-%d)"
REPOSITORY="mcp-memory"

# Function to store a memory
store_memory() {
    local content="$1"
    local tags="$2"
    
    # Escape content for JSON
    content=$(echo "$content" | sed 's/"/\\"/g' | sed ':a;N;$!ba;s/\n/\\n/g')
    
    curl -s -X POST "$GRAPHQL_ENDPOINT" \
        -H "Content-Type: application/json" \
        -d "{
            \"query\": \"mutation { storeChunk(input: { content: \\\"$content\\\", sessionId: \\\"$SESSION_ID\\\", repository: \\\"$REPOSITORY\\\", tags: [$tags] }) { id } }\"
        }" | jq -r '.data.storeChunk.id'
}

echo "🚀 Starting documentation import..."

# Store overview memories
echo "📚 Importing overview..."
store_memory "MCP Memory Server is a semantic memory system for AI assistants that provides persistent, intelligent context management. It uses ChromaDB for vector storage, supports GraphQL API, and includes features like pattern recognition, context suggestion, and memory tracing." '"overview", "architecture"'

store_memory "The project includes comprehensive documentation covering getting started guides, development setup with hot reload, production deployment instructions, monitoring with Prometheus and Grafana, IDE integrations for VS Code and IntelliJ, and marketing materials." '"documentation", "structure"'

# Store architecture memories
echo "🏗️ Importing architecture decisions..."
store_memory "Architecture Decision: Use ChromaDB as the vector database for storing embeddings. ChromaDB provides efficient similarity search, persistent storage, and easy integration with Go applications." '"architecture", "chromadb", "vector-database"'

store_memory "Architecture Decision: Implement GraphQL API for flexible querying. GraphQL allows clients to request exactly the data they need, supports real-time subscriptions, and provides strong typing." '"architecture", "graphql", "api"'

store_memory "Architecture Decision: Use Model Context Protocol (MCP) for AI integration. MCP provides a standardized way for AI assistants to interact with external tools and data sources." '"architecture", "mcp", "ai-integration"'

# Store feature memories
echo "✨ Importing features..."
store_memory "Feature: Memory Tracing - Ability to trace memories by session or find related memories using semantic similarity. Includes timeline visualization for session traces and relationship graphs for connected memories." '"feature", "tracing", "visualization"'

store_memory "Feature: Pattern Recognition - Automatically identifies recurring patterns in stored memories, helping to surface common workflows, repeated problems, and established solutions." '"feature", "pattern-recognition", "intelligence"'

store_memory "Feature: Context Suggestion - Proactively suggests relevant memories based on current context, helping AI assistants maintain continuity across conversations." '"feature", "context-suggestion", "intelligence"'

# Store deployment memories
echo "🚀 Importing deployment info..."
store_memory "Deployment: Docker Compose setup with ChromaDB, MCP Memory Server, monitoring stack (Prometheus, Grafana), and Traefik for routing. Volumes are configured for data persistence with named volumes that should never be deleted." '"deployment", "docker", "infrastructure"'

store_memory "Deployment: ChromaDB persistence requires explicit --path command in docker-compose.yml. Data is stored in /chroma/chroma directory mounted to mcp_memory_chroma_vector_db_NEVER_DELETE volume." '"deployment", "chromadb", "persistence", "configuration"'

# Store development memories
echo "💻 Importing development setup..."
store_memory "Development: Hot reload setup using Air for automatic server restarts. Configuration in .air.toml watches for changes in .go files and rebuilds the server. Excludes vendor, tmp, and other non-source directories." '"development", "hot-reload", "dx"'

store_memory "Development: GraphQL server runs on port 8082 with web UI for memory browsing, GraphiQL playground for API exploration, and REST endpoints for health checks." '"development", "graphql", "api"'

# Store roadmap memories
echo "🗺️ Importing roadmap..."
store_memory "Roadmap: Priority 1 - Bug fixes including ChromaDB v2 API migration, error handling improvements, graceful shutdown, and memory leak prevention." '"roadmap", "priority-1", "bugs"'

store_memory "Roadmap: Priority 2 - Testing infrastructure including unit tests for all services, integration tests for API endpoints, and performance benchmarks." '"roadmap", "priority-2", "testing"'

store_memory "Roadmap: Priority 3 - AI enhancements including better chunking strategies, multi-modal support, and improved pattern recognition algorithms." '"roadmap", "priority-3", "ai-features"'

echo "✅ Documentation import complete!"

# Verify import
echo ""
echo "📊 Verifying import..."
CHUNK_COUNT=$(curl -s -X POST "$GRAPHQL_ENDPOINT" \
    -H "Content-Type: application/json" \
    -d "{\"query\": \"{ listChunks(repository: \\\"$REPOSITORY\\\", limit: 100) { id } }\"}" \
    | jq '.data.listChunks | length')

echo "✨ Successfully imported $CHUNK_COUNT memories!"