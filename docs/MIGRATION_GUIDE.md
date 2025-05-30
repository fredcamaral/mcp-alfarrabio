# ChromaDB to Qdrant Migration Guide

This guide provides comprehensive instructions for migrating your existing ChromaDB data to the new Qdrant vector database.

## 🚀 Quick Start

```bash
# 1. Run the migration preparation script
./scripts/migrate-chromadb-to-qdrant.sh

# 2. Export ChromaDB data to JSON
python3 scripts/export_chromadb.py /path/to/chromadb-data

# 3. Import into Qdrant
go run cmd/migrate/main.go -chroma-export=chromadb_export.json -dry-run
go run cmd/migrate/main.go -chroma-export=chromadb_export.json
```

## 📋 Prerequisites

- Docker and Docker Compose installed
- Python 3.7+ with ChromaDB client (for export)
- Go 1.19+ (for migration tool)
- Existing ChromaDB data in Docker volume

## 🔍 Migration Options

### Option 1: Automated Script (Recommended)

The easiest way to get started:

```bash
# This script handles backup, analysis, and preparation
./scripts/migrate-chromadb-to-qdrant.sh
```

**What it does:**
- ✅ Backs up your ChromaDB data automatically
- ✅ Analyzes your existing data structure
- ✅ Starts Qdrant and ensures it's ready
- ✅ Creates detailed migration instructions
- ✅ Prepares all necessary tools

### Option 2: Manual Step-by-Step

For more control over the process:

#### Step 1: Backup ChromaDB Data
```bash
# Create backup of ChromaDB volume
docker run --rm \
  -v mcp_memory_chroma_vector_db_NEVER_DELETE:/data \
  -v $(pwd)/backup:/backup \
  alpine tar czf /backup/chromadb-backup-$(date +%Y%m%d).tar.gz -C /data .
```

#### Step 2: Export ChromaDB to JSON
```bash
# Install ChromaDB Python client
pip install chromadb

# Export data (adjust path to your ChromaDB data)
python3 scripts/export_chromadb.py /var/lib/docker/volumes/mcp_memory_chroma_vector_db_NEVER_DELETE/_data

# Or export from mounted directory
docker run --rm \
  -v mcp_memory_chroma_vector_db_NEVER_DELETE:/chroma-data \
  -v $(pwd):/workspace \
  -w /workspace \
  python:3.11 \
  bash -c "pip install chromadb && python scripts/export_chromadb.py /chroma-data"
```

#### Step 3: Start Qdrant
```bash
# Start Qdrant service
docker-compose up -d qdrant

# Wait for it to be ready
curl http://localhost:6333/collections
```

#### Step 4: Run Migration
```bash
# Build migration tool
go build -o migrate ./cmd/migrate

# Test with dry run
./migrate -chroma-export=chromadb_export.json -dry-run -verbose

# Run actual migration
./migrate -chroma-export=chromadb_export.json -verbose
```

## 🛠️ Migration Tool Usage

The migration tool supports several modes:

### Dry Run Mode
```bash
go run cmd/migrate/main.go \
  -chroma-export=chromadb_export.json \
  -dry-run \
  -verbose
```
- Tests the migration without writing to Qdrant
- Validates data structure and compatibility
- Shows what would be migrated

### Validation Only
```bash
go run cmd/migrate/main.go \
  -chroma-export=chromadb_export.json \
  -validate-only
```
- Only validates the exported data
- Checks for invalid chunks
- No migration performed

### Force Migration
```bash
go run cmd/migrate/main.go \
  -chroma-export=chromadb_export.json \
  -force
```
- Overwrites existing data in Qdrant
- Use when target collection already has data

### Full Options
```bash
go run cmd/migrate/main.go \
  -chroma-export=chromadb_export.json \
  -backup-dir=./migration-backup \
  -dry-run=false \
  -validate-only=false \
  -force=false \
  -verbose=true
```

## 📊 Data Mapping

The migration converts ChromaDB documents to ConversationChunk format:

| ChromaDB Field | ConversationChunk Field | Notes |
|----------------|-------------------------|-------|
| `id` | `id` | Direct mapping |
| `document` | `content` | Main text content |
| `embedding` | `embeddings` | Vector embeddings |
| `metadata.session_id` | `session_id` | Session identifier |
| `metadata.timestamp` | `timestamp` | Converted to ISO format |
| `metadata.type` | `type` | Chunk type mapping |
| `metadata.summary` | `summary` | AI-generated summary |
| `metadata.repository` | `metadata.repository` | Repository name |
| `metadata.*` | `metadata.extended_metadata` | Additional metadata |

### Default Values for Missing Fields

- `session_id`: `"migrated"`
- `type`: `"discussion"`
- `timestamp`: Current time
- `outcome`: `"success"`
- `difficulty`: `"simple"`
- `repository`: `"migrated"`

## ✅ Validation and Verification

### Automatic Validation
The migration tool performs several validation checks:

1. **Data Structure Validation**
   - Checks all chunks conform to ConversationChunk schema
   - Validates required fields are present
   - Verifies data types are correct

2. **Migration Validation**
   - Compares total chunk counts
   - Samples random chunks for content verification
   - Validates embeddings are preserved

3. **Statistics Validation**
   - Compares chunk distribution by type
   - Verifies repository groupings
   - Checks timestamp ranges

### Manual Verification
After migration, verify your data:

```bash
# Check Qdrant collection status
curl http://localhost:6333/collections/claude_memory

# Test search functionality
curl -X POST http://localhost:6333/collections/claude_memory/points/search \
  -H "Content-Type: application/json" \
  -d '{
    "vector": [0.1, 0.2, 0.3, ...],
    "limit": 5
  }'

# View statistics via your application
# Use your normal MCP memory tools to test functionality
```

## 🚨 Troubleshooting

### Common Issues

#### 1. ChromaDB Connection Errors
```
Error: Failed to connect to ChromaDB
```
**Solution:**
- Check ChromaDB data path is correct
- Try using the JSON export approach instead
- Verify ChromaDB volume exists: `docker volume ls`

#### 2. JSON Export Errors
```
Error: Failed to decode JSON export
```
**Solution:**
- Check JSON file is valid: `jq . chromadb_export.json`
- Re-export with verbose mode: `python scripts/export_chromadb.py -v`
- Check file permissions and size

#### 3. Qdrant Connection Errors
```
Error: Failed to initialize Qdrant
```
**Solution:**
- Ensure Qdrant is running: `docker-compose ps qdrant`
- Check port availability: `netstat -ln | grep 6334`
- Verify configuration: check environment variables

#### 4. Memory/Performance Issues
```
Error: Out of memory during migration
```
**Solution:**
- Reduce batch size in migration tool
- Use progressive migration for large datasets
- Increase Docker memory limits

#### 5. Data Validation Failures
```
Error: Invalid chunk found
```
**Solution:**
- Review validation errors in verbose mode
- Fix data in export before migration
- Use force mode to skip validation (not recommended)

### Recovery Procedures

#### Rollback Migration
If migration fails or produces incorrect results:

```bash
# 1. Stop all services
docker-compose down

# 2. Remove Qdrant volume
docker volume rm mcp_memory_qdrant_vector_db_NEVER_DELETE

# 3. Restore ChromaDB if needed
docker run --rm \
  -v mcp_memory_chroma_vector_db_NEVER_DELETE:/data \
  -v $(pwd)/backup:/backup \
  alpine tar xzf /backup/chromadb-backup-YYYYMMDD.tar.gz -C /data

# 4. Restart with ChromaDB configuration
# (Temporarily update docker-compose.yml to use ChromaDB)
```

#### Partial Migration Recovery
If some chunks failed to migrate:

```bash
# 1. Check migration logs for failed chunk IDs
grep "Failed" migration-backup/migration.log

# 2. Export only failed chunks
# (Modify export script to filter specific IDs)

# 3. Re-run migration for failed chunks only
go run cmd/migrate/main.go -chroma-export=failed_chunks.json
```

## 📈 Performance Optimization

### Large Dataset Migration

For datasets with >100K chunks:

1. **Use Batch Processing**
   ```bash
   # Increase batch size (default: 100)
   # Edit cmd/migrate/main.go, change batchSize constant
   ```

2. **Progressive Migration**
   ```bash
   # Split export by date ranges or repositories
   python scripts/export_chromadb.py --filter-repo=repo1
   python scripts/export_chromadb.py --filter-repo=repo2
   ```

3. **Resource Allocation**
   ```yaml
   # docker-compose.yml - increase Qdrant resources
   services:
     qdrant:
       deploy:
         resources:
           limits:
             memory: 4G
             cpus: '2'
   ```

### Migration Speed Optimization

- **SSD Storage**: Use SSD for Docker volumes
- **Memory**: Increase available RAM for Docker
- **Parallel Processing**: Run multiple migration processes for different repositories
- **Network**: Ensure low latency between migration tool and Qdrant

## 🔐 Security Considerations

### Data Protection
- ✅ Backups are created automatically
- ✅ Dry-run mode tests before real migration
- ✅ Validation ensures data integrity
- ✅ Rollback procedures available

### Access Control
- Ensure proper file permissions on export files
- Use secure channels for data transfer
- Review exported data for sensitive information
- Clean up temporary files after migration

## 📝 Post-Migration Tasks

### 1. Update Configuration
```bash
# Update environment variables to use Qdrant
export MCP_MEMORY_STORAGE_PROVIDER=qdrant
export MCP_MEMORY_QDRANT_HOST=localhost
export MCP_MEMORY_QDRANT_PORT=6334
```

### 2. Test Application Functionality
- Test vector search operations
- Verify memory retrieval works correctly
- Check performance compared to ChromaDB
- Validate all MCP tools function properly

### 3. Monitor Performance
- Monitor Qdrant resource usage
- Compare search response times
- Track memory usage patterns
- Set up health monitoring

### 4. Cleanup
```bash
# Archive ChromaDB backup
mv migration-backup/chromadb-backup-*.tar.gz /secure/archive/

# Remove temporary files
rm chromadb_export.json
rm -rf migration-backup/chromadb-extracted/

# Keep migration logs for reference
mv migration-backup/migration.log /logs/archive/
```

## 🎯 Success Criteria

Migration is considered successful when:

- ✅ All chunks migrated without data loss
- ✅ Vector search returns expected results  
- ✅ Application functionality unchanged
- ✅ Performance meets or exceeds ChromaDB
- ✅ Data validation passes all checks
- ✅ Backup and rollback procedures tested

## 📞 Support

If you encounter issues during migration:

1. **Check the logs**: Review migration logs in `migration-backup/`
2. **Validate data**: Use dry-run mode to test without changes
3. **Incremental approach**: Try migrating a small subset first
4. **Documentation**: Review this guide and tool documentation
5. **Community**: Check project issues and discussions

## 🔄 Migration Checklist

- [ ] Backup ChromaDB data
- [ ] Install Python dependencies
- [ ] Export ChromaDB to JSON
- [ ] Validate JSON export
- [ ] Start Qdrant service
- [ ] Run migration dry-run
- [ ] Review dry-run results
- [ ] Run actual migration
- [ ] Validate migration results
- [ ] Test application functionality
- [ ] Update configuration
- [ ] Monitor performance
- [ ] Archive backups
- [ ] Document any issues
- [ ] Clean up temporary files