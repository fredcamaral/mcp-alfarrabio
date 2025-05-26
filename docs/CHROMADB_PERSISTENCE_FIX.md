# ChromaDB Persistence Configuration

## Issue
ChromaDB was not persisting data between container restarts. Data was being saved to `/data` instead of the mounted volume.

## Solution

### 1. Update docker-compose.yml

Add the `command` directive to explicitly set the data path:

```yaml
services:
  chroma:
    image: chromadb/chroma:latest
    container_name: mcp-chroma
    restart: unless-stopped
    command: ["run", "--path", "/chroma/chroma", "--host", "0.0.0.0", "--port", "8000"]
    ports:
      - "${CHROMA_HOST_PORT:-8000}:8000"
    environment:
      - CHROMA_SERVER_HOST=0.0.0.0
      - CHROMA_SERVER_HTTP_PORT=${CHROMA_HTTP_PORT:-8000}
      - IS_PERSISTENT=TRUE
    volumes:
      - chroma_data:/chroma/chroma
    networks:
      - mcp_network
```

### 2. Volume Configuration

The volume is correctly configured in docker-compose.yml:

```yaml
volumes:
  chroma_data:
    driver: local
    name: mcp_memory_chroma_vector_db_NEVER_DELETE
```

### 3. Verification

After applying the changes:

1. ChromaDB creates `chroma.sqlite3` in `/chroma/chroma`
2. The SQLite file persists after container restart
3. Volume data is stored at: `/var/lib/docker/volumes/mcp_memory_chroma_vector_db_NEVER_DELETE/_data`

### 4. Port Configuration

- Container internal port: 8000
- Host port (configurable): 9000 (default) or set via `CHROMA_HOST_PORT`
- GraphQL server connection: `http://localhost:9000`

### 5. Known Issues

- Document count may show as 0 after restart (investigating whether vector embeddings need separate persistence configuration)
- The environment variable `PERSIST_DIRECTORY` alone is not sufficient; the `--path` command argument is required

## Usage

```bash
# Start ChromaDB with persistence
docker-compose up -d chroma

# Connect GraphQL server
MCP_MEMORY_CHROMA_ENDPOINT=http://localhost:9000 ./graphql

# Verify persistence
docker exec mcp-chroma ls -la /chroma/chroma
# Should show: chroma.sqlite3
```

## Data Backup

To backup ChromaDB data:

```bash
# Create backup
docker run --rm -v mcp_memory_chroma_vector_db_NEVER_DELETE:/data -v $(pwd):/backup alpine tar czf /backup/chroma-backup.tar.gz -C /data .

# Restore backup
docker run --rm -v mcp_memory_chroma_vector_db_NEVER_DELETE:/data -v $(pwd):/backup alpine tar xzf /backup/chroma-backup.tar.gz -C /data
```