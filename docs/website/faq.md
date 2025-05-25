# Frequently Asked Questions

## General Questions

### What is MCP Memory?

MCP Memory is a Model Context Protocol (MCP) server that provides intelligent memory and context management for AI-powered development workflows. It helps AI assistants remember past conversations, learn from your development history, and provide more relevant suggestions.

### How does MCP Memory differ from chat history?

While chat history is a simple chronological record, MCP Memory:
- Uses semantic search to find relevant information
- Identifies patterns across conversations
- Builds knowledge graphs of your codebase
- Learns from successes and failures
- Provides intelligent suggestions based on context

### Is my data secure?

Yes! MCP Memory is designed with privacy first:
- All data is stored locally on your machine
- No data is sent to external services (except embedding generation)
- Optional encryption for sensitive data
- You control what gets stored and when
- Easy data export and deletion

### What AI assistants are supported?

MCP Memory works with any AI assistant that supports the Model Context Protocol:
- Claude Desktop
- VS Code with Continue extension
- Any MCP-compatible client

## Installation & Setup

### Do I need Docker?

Docker is required for ChromaDB (the vector database). If you prefer not to use Docker, you can:
- Use the in-memory storage option (data won't persist)
- Deploy ChromaDB separately
- Use an alternative vector database (requires code changes)

### What about the OpenAI API key?

The OpenAI API key is used for generating embeddings (converting text to vectors for semantic search). You can:
- Use OpenAI's embedding models (recommended)
- Use alternative providers (Cohere, HuggingFace)
- Use local embedding models (requires more setup)

### Can I use MCP Memory offline?

Partially. You can:
- Store and retrieve exact matches offline
- Use cached embeddings
- But semantic search requires embedding generation (needs API access)

### How much storage does it need?

Storage requirements depend on usage:
- Typical conversation chunk: ~1-2 KB
- Embedding per chunk: ~6 KB
- 1000 conversations ≈ 8 MB
- 10,000 conversations ≈ 80 MB

## Usage Questions

### When should I store conversations?

Store conversations when:
- Making important decisions
- Solving complex problems
- Learning something new
- Implementing key features
- Debugging tricky issues

### How specific should my searches be?

Start broad, then narrow:
```
# Too broad
"error"

# Good starting point
"database connection error"

# Very specific
"PostgreSQL connection pool timeout in order service"
```

### Can I edit or delete stored memories?

Yes! You can:
- Export all data for a project
- Delete specific conversations (coming soon)
- Clear all data for a repository
- Modify exported data and re-import

### How do tags work?

Tags help categorize memories:
- Use consistent naming (`api`, not `API`, `apis`, `api-v2`)
- Create hierarchical tags (`bug:performance:database`)
- Common tags: `architecture`, `bug`, `feature`, `optimization`
- Tags improve search accuracy

## Technical Questions

### What's the difference between stdio and HTTP mode?

**stdio mode**:
- Direct communication with AI assistant
- Lower latency
- Single user
- Ideal for personal use

**HTTP mode**:
- Network accessible
- Multiple clients
- REST API
- Better for team use

### How are embeddings generated?

1. Text is sent to embedding API (OpenAI by default)
2. API returns a vector representation
3. Vector is stored in ChromaDB
4. Searches compare vector similarity

### Can I use a different vector database?

Currently, ChromaDB is the primary option. To use alternatives:
- Implement the `storage.VectorStore` interface
- Update configuration to use your implementation
- Popular alternatives: Pinecone, Weaviate, Qdrant

### What happens if ChromaDB is down?

MCP Memory will:
- Return an error for search operations
- Store operations will fail
- Exact match retrieval still works
- Health check will indicate the issue

## Performance Questions

### How fast are searches?

Typical search performance:
- Semantic search: 50-200ms
- Exact match: 10-50ms
- Pattern detection: 100-500ms
- Depends on data size and hardware

### Is there a limit on stored conversations?

No hard limit, but consider:
- ChromaDB performance may degrade with millions of items
- Search becomes slower with very large datasets
- Regular cleanup recommended for optimal performance

### How can I improve performance?

1. **Regular maintenance**: Archive old conversations
2. **Optimize searches**: Use specific queries
3. **Hardware**: SSD storage, adequate RAM
4. **Configuration**: Tune ChromaDB settings

## Troubleshooting

### "Connection refused" errors

Check if services are running:
```bash
# Check ChromaDB
docker ps | grep chroma

# Check MCP Memory
ps aux | grep mcp-memory

# Restart services
docker-compose restart
```

### "API key invalid" errors

```bash
# Check if key is set
echo $OPENAI_API_KEY

# Verify key works
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer $OPENAI_API_KEY"
```

### Search returns no results

1. Verify conversations were stored successfully
2. Check if embeddings were generated
3. Try broader search terms
4. Check relevance threshold (default: 0.7)

### High memory usage

- ChromaDB caches data in memory
- Restart ChromaDB to clear cache
- Consider archiving old data
- Adjust ChromaDB memory limits

## Advanced Questions

### Can I customize the embedding model?

Yes! In your configuration:
```yaml
embeddings:
  provider: openai
  model: text-embedding-3-large  # or text-embedding-3-small
  dimension: 3072  # Match model dimension
```

### How do patterns get identified?

Pattern recognition uses:
- Frequency analysis
- Temporal clustering
- Semantic similarity
- Statistical analysis
- Machine learning models

### Can I share memories between team members?

Yes, several options:
1. Use HTTP mode with shared server
2. Export/import memory archives
3. Sync ChromaDB data
4. Use centralized deployment

### How do I backup my data?

```bash
# Backup ChromaDB volumes
docker run --rm -v mcp-memory_chroma_data:/data \
  -v $(pwd):/backup alpine \
  tar -czf /backup/chroma-backup.tar.gz /data

# Export memories
mcp-memory export --repository my-project \
  --format archive --output backup.json
```

## Feature Requests

### What features are planned?

Check our [roadmap](https://github.com/your-org/mcp-memory/blob/main/ROADMAP.md):
- Web UI for memory management
- Team collaboration features
- Advanced analytics
- More embedding providers
- Plugin system

### Can I contribute?

Absolutely! See our [contributing guide](https://github.com/your-org/mcp-memory/blob/main/CONTRIBUTING.md):
- Report bugs
- Suggest features
- Submit pull requests
- Improve documentation
- Share use cases

---

Still have questions? Join our [Discord community](https://discord.gg/mcp-memory) or open a [GitHub issue](https://github.com/your-org/mcp-memory/issues)!