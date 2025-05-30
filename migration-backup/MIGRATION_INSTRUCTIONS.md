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

