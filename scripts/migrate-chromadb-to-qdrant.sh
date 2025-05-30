#!/bin/bash
set -euo pipefail

# ChromaDB to Qdrant Migration Script
# This script helps migrate data from ChromaDB to Qdrant vector database

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BACKUP_DIR="$PROJECT_ROOT/migration-backup"
LOG_FILE="$BACKUP_DIR/migration.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
CHROMA_VOLUME="mcp_memory_chroma_vector_db_NEVER_DELETE"
QDRANT_VOLUME="mcp_memory_qdrant_vector_db_NEVER_DELETE"
DOCKER_COMPOSE_FILE="$PROJECT_ROOT/docker-compose.yml"

print_header() {
    echo -e "${BLUE}"
    echo "=============================================="
    echo "  ChromaDB to Qdrant Migration Tool"
    echo "=============================================="
    echo -e "${NC}"
}

print_step() {
    echo -e "${GREEN}[STEP]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

check_prerequisites() {
    print_step "Checking prerequisites..."
    
    # Check if Docker is running
    if ! docker info > /dev/null 2>&1; then
        print_error "Docker is not running. Please start Docker and try again."
        exit 1
    fi
    
    # Check if ChromaDB volume exists
    if ! docker volume inspect "$CHROMA_VOLUME" > /dev/null 2>&1; then
        print_error "ChromaDB volume '$CHROMA_VOLUME' not found."
        print_info "This means you don't have existing ChromaDB data to migrate."
        exit 1
    fi
    
    # Check if docker-compose file exists
    if [[ ! -f "$DOCKER_COMPOSE_FILE" ]]; then
        print_error "docker-compose.yml not found at $DOCKER_COMPOSE_FILE"
        exit 1
    fi
    
    print_info "✅ Prerequisites check passed"
}

backup_chromadb_volume() {
    print_step "Creating backup of ChromaDB data..."
    
    mkdir -p "$BACKUP_DIR"
    
    local backup_file="$BACKUP_DIR/chromadb-backup-$(date +%Y%m%d-%H%M%S).tar.gz"
    
    print_info "Backing up ChromaDB volume to: $backup_file"
    
    docker run --rm \
        -v "$CHROMA_VOLUME:/data" \
        -v "$BACKUP_DIR:/backup" \
        alpine:latest \
        tar czf "/backup/$(basename "$backup_file")" -C /data .
    
    if [[ -f "$backup_file" ]]; then
        local size=$(du -h "$backup_file" | cut -f1)
        print_info "✅ ChromaDB backup created: $backup_file ($size)"
        echo "$backup_file" > "$BACKUP_DIR/chromadb-backup-latest.txt"
    else
        print_error "Failed to create ChromaDB backup"
        exit 1
    fi
}

analyze_chromadb_data() {
    print_step "Analyzing ChromaDB data..."
    
    # Mount ChromaDB volume and analyze its contents
    docker run --rm \
        -v "$CHROMA_VOLUME:/chroma-data" \
        alpine:latest \
        sh -c "
            echo 'ChromaDB Data Analysis:'
            echo '======================'
            echo 'Directory structure:'
            find /chroma-data -type f -name '*.sqlite*' -o -name '*.db' | head -10
            echo ''
            echo 'Total files:'
            find /chroma-data -type f | wc -l
            echo ''
            echo 'Directory size:'
            du -sh /chroma-data
            echo ''
            echo 'SQLite databases found:'
            find /chroma-data -name '*.sqlite*' -exec ls -lh {} \;
        " | tee -a "$LOG_FILE"
}

start_qdrant() {
    print_step "Starting Qdrant service..."
    
    cd "$PROJECT_ROOT"
    
    # Start only Qdrant service
    docker-compose up -d qdrant
    
    # Wait for Qdrant to be ready
    print_info "Waiting for Qdrant to be ready..."
    local max_attempts=30
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        if curl -s "http://localhost:6333/collections" > /dev/null 2>&1; then
            print_info "✅ Qdrant is ready"
            break
        fi
        
        if [[ $attempt -eq $max_attempts ]]; then
            print_error "Qdrant failed to start within ${max_attempts} seconds"
            exit 1
        fi
        
        echo -n "."
        sleep 1
        ((attempt++))
    done
}

run_migration_dry_run() {
    print_step "Running migration dry run..."
    
    cd "$PROJECT_ROOT"
    
    # Build migration tool
    go build -o migrate ./cmd/migrate
    
    # Run dry run
    ./migrate \
        -chroma-path="/tmp/chroma-mounted" \
        -config="configs/dev/config.yaml" \
        -backup-dir="$BACKUP_DIR" \
        -dry-run \
        -verbose | tee -a "$LOG_FILE"
}

extract_chromadb_for_analysis() {
    print_step "Extracting ChromaDB data for analysis..."
    
    local extract_dir="$BACKUP_DIR/chromadb-extracted"
    mkdir -p "$extract_dir"
    
    # Extract ChromaDB volume to temporary directory
    docker run --rm \
        -v "$CHROMA_VOLUME:/chroma-data" \
        -v "$extract_dir:/extract" \
        alpine:latest \
        cp -r /chroma-data/. /extract/
    
    print_info "ChromaDB data extracted to: $extract_dir"
    
    # List SQLite files
    print_info "SQLite databases found:"
    find "$extract_dir" -name "*.sqlite*" -exec ls -lh {} \;
}

show_migration_options() {
    print_step "Migration Options"
    echo ""
    echo "The migration tool template has been created, but you need to complete it:"
    echo ""
    echo "1. 📋 OPTION 1: Manual Export/Import (Recommended for small datasets)"
    echo "   - Use ChromaDB's export functionality to export data as JSON"
    echo "   - Transform the JSON to match ConversationChunk format"
    echo "   - Import into Qdrant using our batch import tools"
    echo ""
    echo "2. 🔧 OPTION 2: Complete the Migration Tool (Recommended for large datasets)"
    echo "   - Add ChromaDB client dependency to go.mod"
    echo "   - Implement readChromaDBData() function in cmd/migrate/main.go"
    echo "   - Run automated migration with validation"
    echo ""
    echo "3. 🐍 OPTION 3: Python Script (If you have ChromaDB Python client)"
    echo "   - Write a Python script to read from ChromaDB"
    echo "   - Export as JSON intermediate format"
    echo "   - Import using Go migration tool"
    echo ""
}

create_migration_instructions() {
    print_step "Creating detailed migration instructions..."
    
    local instructions_file="$BACKUP_DIR/MIGRATION_INSTRUCTIONS.md"
    
    cat > "$instructions_file" << 'EOF'
# ChromaDB to Qdrant Migration Instructions

## Overview
This document provides step-by-step instructions for migrating your data from ChromaDB to Qdrant.

## What We Found
- ChromaDB volume: `mcp_memory_chroma_vector_db_NEVER_DELETE`
- Backup created in: `migration-backup/`
- Qdrant is configured and ready

## Migration Options

### Option 1: Manual JSON Export/Import (Recommended)

This is the safest and most controllable approach:

#### Step 1: Export from ChromaDB
```python
# Create this Python script: export_chromadb.py
import chromadb
import json
from datetime import datetime

def export_chromadb_to_json():
    # Connect to your ChromaDB instance
    client = chromadb.PersistentClient(path="/path/to/your/chromadb/data")
    
    # Get all collections
    collections = client.list_collections()
    
    exported_data = {
        "export_timestamp": datetime.now().isoformat(),
        "collections": {}
    }
    
    for collection in collections:
        coll = client.get_collection(collection.name)
        
        # Get all documents
        results = coll.get(include=["documents", "metadatas", "embeddings"])
        
        exported_data["collections"][collection.name] = {
            "documents": results["documents"],
            "metadatas": results["metadatas"], 
            "embeddings": results["embeddings"],
            "ids": results["ids"]
        }
    
    # Save to JSON file
    with open("chromadb_export.json", "w") as f:
        json.dump(exported_data, f, indent=2)
    
    print(f"Exported {len(collections)} collections")

if __name__ == "__main__":
    export_chromadb_to_json()
```

#### Step 2: Run the Export
```bash
# Install ChromaDB Python client if needed
pip install chromadb

# Run the export script
python export_chromadb.py
```

#### Step 3: Transform and Import
```bash
# Use our migration tool to import the JSON
go run cmd/migrate/main.go \
    -chroma-export="chromadb_export.json" \
    -config="configs/dev/config.yaml" \
    -backup-dir="migration-backup"
```

### Option 2: Complete the Migration Tool

If you want to implement the full automated migration:

#### Step 1: Add ChromaDB Dependency
```bash
go get github.com/chromadb/chromadb-go
```

#### Step 2: Implement readChromaDBData()
Edit `cmd/migrate/main.go` and replace the placeholder `readChromaDBData()` function with:

```go
func (mt *MigrationTool) readChromaDBData(ctx context.Context) ([]types.ConversationChunk, error) {
    // Connect to ChromaDB
    client := chromadb.NewClient(chromadb.WithDatabase(mt.chromaDBPath))
    
    // Get collections
    collections, err := client.ListCollections(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to list collections: %w", err)
    }
    
    var allChunks []types.ConversationChunk
    
    for _, collName := range collections {
        coll, err := client.GetCollection(ctx, collName)
        if err != nil {
            continue
        }
        
        // Query all documents
        results, err := coll.Query(ctx, chromadb.QueryRequest{
            NResults: 10000, // Adjust based on your data size
        })
        if err != nil {
            continue
        }
        
        // Transform to ConversationChunk format
        for i, doc := range results.Documents {
            chunk := types.ConversationChunk{
                ID:        results.IDs[i],
                Content:   doc,
                Embeddings: results.Embeddings[i],
                // Map other fields from metadata
                // ... implement field mapping based on your ChromaDB schema
            }
            
            allChunks = append(allChunks, chunk)
        }
    }
    
    return allChunks, nil
}
```

#### Step 3: Run Migration
```bash
go run cmd/migrate/main.go \
    -chroma-path="/var/lib/docker/volumes/mcp_memory_chroma_vector_db_NEVER_DELETE/_data" \
    -config="configs/dev/config.yaml" \
    -backup-dir="migration-backup" \
    -verbose
```

### Option 3: Python Bridge Script

Create a Python script that exports ChromaDB data in our ConversationChunk JSON format:

```python
# chromadb_to_qdrant_bridge.py
import chromadb
import json
import uuid
from datetime import datetime

def convert_to_conversation_chunk(doc_id, document, metadata, embedding):
    """Convert ChromaDB document to ConversationChunk format"""
    return {
        "id": doc_id,
        "session_id": metadata.get("session_id", "migrated"),
        "timestamp": metadata.get("timestamp", datetime.now().isoformat()),
        "type": metadata.get("type", "discussion"),
        "content": document,
        "summary": metadata.get("summary", ""),
        "metadata": {
            "repository": metadata.get("repository", "migrated"),
            "branch": metadata.get("branch", "main"),
            "files_modified": metadata.get("files_modified", []),
            "tools_used": metadata.get("tools_used", []),
            "outcome": metadata.get("outcome", "success"),
            "tags": metadata.get("tags", []),
            "difficulty": metadata.get("difficulty", "simple"),
            "extended_metadata": metadata
        },
        "embeddings": embedding,
        "related_chunks": []
    }

# Use this to export and then import with our Go tools
```

## Validation Steps

After migration, validate your data:

1. **Check total count**: Compare total chunks in ChromaDB vs Qdrant
2. **Sample validation**: Verify a few random chunks have correct content
3. **Search testing**: Test vector search functionality
4. **Metadata verification**: Ensure all metadata fields migrated correctly

## Rollback Plan

If migration fails:
1. Stop Qdrant service
2. Clear Qdrant volume: `docker volume rm mcp_memory_qdrant_vector_db_NEVER_DELETE`
3. Restore ChromaDB from backup if needed
4. Restart with ChromaDB until issues are resolved

## Post-Migration

1. **Test thoroughly** with your typical use cases
2. **Monitor performance** comparing ChromaDB vs Qdrant
3. **Update configuration** to use Qdrant permanently
4. **Archive ChromaDB backup** in secure location
5. **Update documentation** and team knowledge

## Support

If you encounter issues:
1. Check the migration logs in `migration-backup/migration.log`
2. Review the backup files for data integrity
3. Test with a small subset first using dry-run mode
4. Consider staged migration for large datasets

EOF

    print_info "✅ Detailed instructions created: $instructions_file"
}

main() {
    print_header
    
    # Create backup directory and log file
    mkdir -p "$BACKUP_DIR"
    echo "Migration started at $(date)" > "$LOG_FILE"
    
    check_prerequisites
    backup_chromadb_volume
    analyze_chromadb_data
    extract_chromadb_for_analysis
    start_qdrant
    
    # Note: We can't run the actual migration yet because we need ChromaDB client
    # run_migration_dry_run
    
    show_migration_options
    create_migration_instructions
    
    print_step "Migration preparation complete!"
    echo ""
    print_info "📁 Backup directory: $BACKUP_DIR"
    print_info "📋 Instructions: $BACKUP_DIR/MIGRATION_INSTRUCTIONS.md"
    print_info "📊 Log file: $LOG_FILE"
    echo ""
    print_warning "Next steps:"
    echo "1. Review the detailed instructions in $BACKUP_DIR/MIGRATION_INSTRUCTIONS.md"
    echo "2. Choose your preferred migration method"
    echo "3. Test with a small dataset first"
    echo "4. Run full migration when ready"
    echo ""
    print_info "✅ Qdrant is running and ready to receive data"
}

# Run main function
main "$@"