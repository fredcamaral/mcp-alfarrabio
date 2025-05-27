# TODO List

This file contains all TODO items found in the codebase as of 2025-05-27.

## Core Application TODOs

### Memory System Enhancement (Priority: High)
- **Purpose**: Improve memory storage with better contextual keys for enhanced retrieval and search
- **Current Limitations**:
  - [ ] Limited context about where code/work is happening (working directory, relative paths)
  - [ ] No client type awareness (CLI vs ChatGPT vs other)
  - [ ] Missing hierarchical relationships between memories
  - [ ] No automatic language/framework detection
  
- **Proposed Enhancements**:
  - [ ] **Location Context**:
    - Working directory (relative path from repo root)
    - Active files being edited/viewed
    - Current git branch and commit hash
    - Project type detection (Go, Python, Node.js, etc.)
  
  - [ ] **Client Context**:
    - Client type (claude-cli, chatgpt, vscode, etc.)
    - Client version
    - Platform (macOS, Linux, Windows)
    - Session environment variables
  
  - [ ] **Enhanced Metadata**:
    - Language/framework versions detected from files
    - Package dependencies from go.mod, package.json, etc.
    - Error signatures and stack traces
    - Command execution results and exit codes
    - File operation types (create, edit, delete, rename)
  
  - [ ] **Relationship Tracking**:
    - Parent/child memory relationships
    - Memory chains for multi-step solutions
    - Cross-reference to related memories
    - Superseded/obsolete memory links
  
  - [ ] **Search Improvements**:
    - Auto-tag with detected technologies
    - Problem domain classification (frontend, backend, CI/CD, etc.)
    - Semantic categorization
    - Confidence scoring based on outcomes
  
  - [ ] **Usage Analytics**:
    - Track which memories are accessed
    - Success rate of retrieved memories
    - Memory effectiveness scoring
    - Auto-archive stale memories

- **Implementation Ideas**:
  - [ ] Add `ExtendedMetadata` JSON field to ConversationChunk
  - [ ] Create memory templates for common scenarios
  - [ ] Implement automatic context detection
  - [ ] Add client type parameter to all memory operations
  - [ ] Build memory relationship graph
  - [ ] Create memory quality scoring system

### GraphQL Integration
- **File**: `internal/graphql/resolvers.go`
  - [ ] Fix SuggestContext API
  - [ ] Implement ProcessConversation for new chunking service

### MCP Server
- **File**: `internal/mcp/server.go`
  - [ ] Fix interface compatibility and re-enable pattern-based suggestions

### Security
- **File**: `internal/security/access_control.go`
  - [ ] Write to persistent audit log storage

## Chroma-Go Library TODOs

### Core Functionality
- **File**: `internal/storage/chroma-go/chroma.go`
  - [ ] Evaluate collection deletion strategy when other collections use the same EF
  - [ ] Add validation for collection operations (2 instances)

### API Development
- **File**: `internal/storage/chroma-go/pkg/api/v2/base.go`
  - [ ] Ensure compatibility with v1 API (2 instances)

- **File**: `internal/storage/chroma-go/pkg/api/v2/client.go`
  - [ ] Add support for collection configuration

- **File**: `internal/storage/chroma-go/pkg/api/v2/client_http.go`
  - [ ] Optimize database setting logic

- **File**: `internal/storage/chroma-go/pkg/api/v2/collection.go`
  - [ ] Add documentation links for ID requirements (2 instances)

- **File**: `internal/storage/chroma-go/pkg/api/v2/collection_http.go`
  - [ ] Improve name validation
  - [ ] Add utility methods for metadata lookups

### Embeddings
- **File**: `internal/storage/chroma-go/pkg/embeddings/cohere/option.go`
  - [ ] Add support for returning multiple embedding types from EmbeddingFunction

- **File**: `internal/storage/chroma-go/pkg/embeddings/embedding.go`
  - [ ] Optimize data copying in FromFloat32 conversion

- **File**: `internal/storage/chroma-go/pkg/embeddings/jina/jina.go`
  - [ ] Support other embedding types beyond float32

- **File**: `internal/storage/chroma-go/pkg/embeddings/mistral/mistral.go`
  - [ ] Support integer embeddings based on encoding format

### Default Embedding Function
- **File**: `internal/storage/chroma-go/pkg/embeddings/default_ef/download_utils.go`
  - [ ] Add integrity check for downloaded files (3 instances)

### Re-ranking
- **File**: `internal/storage/chroma-go/pkg/rerankings/hf/huggingface.go`
  - [ ] Serialize body in error messages

- **File**: `internal/storage/chroma-go/pkg/rerankings/hf/huggingface_test.go`
  - [ ] Extract reranking tests into separate commons file (2 instances)

- **File**: `internal/storage/chroma-go/pkg/rerankings/jina/jina.go`
  - [ ] Serialize body in error messages
  - [ ] Clarify Documents field structure (objects vs strings)

- **File**: `internal/storage/chroma-go/pkg/rerankings/jina/jina_test.go`
  - [ ] Extract reranking tests into separate commons file (2 instances)

### Types and Records
- **File**: `internal/storage/chroma-go/types/record.go`
  - [ ] Add optional error logging

- **File**: `internal/storage/chroma-go/types/types.go`
  - [ ] Validate where conditions (2 instances)

### Documentation
- **File**: `internal/storage/chroma-go/docs/docs/filtering.md`
  - [ ] Add builder example (2 instances)
  - [ ] Describe all available operations (2 instances)

### Testing
- **File**: `internal/storage/chroma-go/pkg/api/v2/client_http_integration_test.go`
  - [ ] Document odd behavior in test

- **File**: `internal/storage/chroma-go/pkg/api/v2/client_http_test.go`
  - [ ] Add tests with tenant, database, and EF

### Commons
- **File**: `internal/storage/chroma-go/pkg/commons/cohere/cohere_commons.go`
  - [ ] Rename GetRequest to GetHTTPRequest for clarity

## Notes

- Many TODOs use `context.TODO()` which should eventually be replaced with proper context handling
- The chroma-go library has the most TODOs and may need focused attention
- Some TODOs indicate missing documentation and examples
- Several TODOs relate to API compatibility and validation improvements