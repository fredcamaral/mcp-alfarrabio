# MCP Memory Server v2 - API Reference

## Overview

The MCP Memory Server v2 provides a clean, intuitive JSON-RPC API through the Model Context Protocol (MCP). The API is organized into 4 logical tools with clear boundaries and consistent parameter patterns.

## API Architecture

### Transport Protocols

The server supports multiple transport protocols:

- **stdio**: Standard input/output for MCP clients (Claude Desktop, VS Code)
- **WebSocket**: Real-time bidirectional communication (`ws://localhost:9080/ws`)
- **HTTP**: Direct JSON-RPC over HTTP (`http://localhost:9080/mcp`)
- **Server-Sent Events**: SSE with HTTP fallback (`http://localhost:9080/sse`)

### JSON-RPC Format

All API calls use JSON-RPC 2.0 format:

```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "memory_store",
    "arguments": {
      "operation": "store_content",
      "project_id": "my-project",
      "content": "Hello, world!"
    }
  },
  "id": 1
}
```

### Response Format

```json
{
  "jsonrpc": "2.0",
  "result": {
    "success": true,
    "message": "Content stored successfully",
    "timestamp": "2024-12-06T15:30:45Z",
    "duration": "15ms",
    "content_id": "content_abc123"
  },
  "id": 1
}
```

## Core Parameter System

### Standard Parameters

All operations use consistent parameter patterns:

```typescript
interface StandardParams {
  project_id: string;          // Required for project isolation
  session_id?: string;         // Optional for expanded access
  user_id?: string;           // Optional for audit logging
  request_id?: string;        // Optional for request tracking
}
```

### Parameter Scoping

Operations have different parameter requirements based on scope:

- **Global Scope**: No `project_id` required (system operations)
- **Project Scope**: `project_id` required (most operations)
- **Session Scope**: `project_id` + `session_id` required (user-specific operations)

### Session Semantics

**Important**: Session semantics are logical and intuitive:

- **Without session_id**: Read-only access to project data
- **With session_id**: Full access to session data + project data
- Session provides **MORE** access, not less

## Tool 1: memory_store

Handles all data persistence operations.

### Operations

#### store_content

Stores new content in the memory system.

**Parameters:**
```typescript
interface StoreContentParams extends StandardParams {
  operation: "store_content";
  content: string;                    // Content text to store
  summary?: string;                   // Optional content summary
  content_type?: string;              // MIME type (default: "text/plain")
  tags?: string[];                    // Organizational tags
  metadata?: Record<string, any>;     // Custom metadata
  options?: {
    generate_embeddings?: boolean;    // Auto-generate embeddings
    detect_relationships?: boolean;   // Auto-detect content relationships
    extract_metadata?: boolean;       // Auto-extract metadata
  };
}
```

**Response:**
```typescript
interface StoreContentResponse {
  success: boolean;
  message: string;
  timestamp: string;
  duration: string;
  content_id: string;                 // Generated content ID
  metadata?: Record<string, any>;     // Extracted/generated metadata
}
```

**Example:**
```json
{
  "operation": "store_content",
  "project_id": "my-project",
  "session_id": "session-123",
  "content": "# Project Documentation\n\nThis document describes...",
  "content_type": "text/markdown",
  "tags": ["documentation", "project"],
  "options": {
    "generate_embeddings": true,
    "detect_relationships": true
  }
}
```

#### update_content

Updates existing content.

**Parameters:**
```typescript
interface UpdateContentParams extends StandardParams {
  operation: "update_content";
  content_id: string;                 // Content to update
  updates: {
    content?: string;                 // New content text
    summary?: string;                 // New summary
    tags?: string[];                  // New tags (replaces existing)
    metadata?: Record<string, any>;   // Metadata updates (merged)
  };
  options?: {
    regenerate_embeddings?: boolean;  // Regenerate embeddings
    update_relationships?: boolean;   // Update relationships
    preserve_previous?: boolean;      // Keep previous version
  };
}
```

#### delete_content

Removes content from the system.

**Parameters:**
```typescript
interface DeleteContentParams extends StandardParams {
  operation: "delete_content";
  content_id: string;                 // Content to delete
  options?: {
    hard?: boolean;                   // Permanent deletion
    delete_relationships?: boolean;   // Also delete relationships
    preserve_references?: boolean;    // Keep references but mark deleted
  };
}
```

#### store_decision

Stores important decisions for future reference.

**Parameters:**
```typescript
interface StoreDecisionParams extends StandardParams {
  operation: "store_decision";
  title: string;                      // Decision title
  context: string;                    // Decision context/background
  decision: string;                   // The decision made
  rationale: string;                  // Why this decision was made
  alternatives?: string[];            // Alternatives considered
  impact?: string;                    // Expected impact
  stakeholders?: string[];            // People affected
  tags?: string[];                    // Decision tags
  metadata?: Record<string, any>;     // Additional metadata
}
```

#### create_relationship

Creates relationships between content items.

**Parameters:**
```typescript
interface CreateRelationshipParams extends StandardParams {
  operation: "create_relationship";
  source_id: string;                  // Source content ID
  target_id: string;                  // Target content ID
  type: RelationshipType;             // Relationship type
  strength?: number;                  // Relationship strength (0-1)
  context?: string;                   // Relationship context
  metadata?: Record<string, any>;     // Relationship metadata
}

type RelationshipType = 
  | "similar_to" | "related_to" | "references" | "cites"
  | "contains" | "part_of" | "follows" | "precedes"
  | "causes" | "resolves" | "implements" | "describes";
```

## Tool 2: memory_retrieve

Handles all data retrieval operations.

### Operations

#### search_content

Performs semantic search across content.

**Parameters:**
```typescript
interface SearchContentParams extends StandardParams {
  operation: "search_content";
  query: string;                      // Search query
  query_type?: "semantic" | "keyword" | "hybrid"; // Search type
  filters?: {
    content_types?: string[];         // Filter by content type
    tags?: string[];                  // Filter by tags
    date_range?: {
      start: string;                  // ISO date string
      end: string;                    // ISO date string
    };
    metadata?: Record<string, any>;   // Metadata filters
  };
  options?: {
    limit?: number;                   // Results limit (default: 10)
    offset?: number;                  // Results offset (default: 0)
    min_relevance?: number;           // Minimum relevance score
    include_context?: boolean;        // Include surrounding context
    include_highlights?: boolean;     // Include search highlights
    sort_by?: "relevance" | "date" | "title";
    sort_order?: "asc" | "desc";
  };
}
```

**Response:**
```typescript
interface SearchContentResponse {
  success: boolean;
  timestamp: string;
  duration: string;
  results: SearchResult[];
  total: number;                      // Total matching results
  page: number;                       // Current page
  per_page: number;                   // Results per page
  facets?: Record<string, any>;       // Search facets
}

interface SearchResult {
  content: Content;                   // Full content object
  relevance: number;                  // Relevance score (0-1)
  highlights?: string[];              // Search term highlights
  context?: string;                   // Surrounding context
  explanation?: string;               // Why this result matched
}
```

#### get_content

Retrieves specific content by ID.

**Parameters:**
```typescript
interface GetContentParams extends StandardParams {
  operation: "get_content";
  content_id: string;                 // Content ID to retrieve
  include_refs?: boolean;             // Include references
  include_history?: boolean;          // Include version history
  options?: {
    include_embeddings?: boolean;     // Include vector embeddings
    include_relationships?: boolean;  // Include relationships
    include_metadata?: boolean;       // Include all metadata
    format?: "full" | "summary" | "minimal"; // Response format
  };
}
```

#### find_similar_content

Finds semantically similar content.

**Parameters:**
```typescript
interface FindSimilarParams extends StandardParams {
  operation: "find_similar_content";
  content?: string;                   // Content text to match
  content_id?: string;                // OR content ID to match
  limit?: number;                     // Results limit (default: 10)
  threshold?: number;                 // Similarity threshold (0-1)
  options?: {
    include_self?: boolean;           // Include the source content
    content_types?: string[];         // Filter by content types
    exclude_ids?: string[];           // Exclude specific content IDs
    similarity_method?: "cosine" | "euclidean" | "manhattan";
  };
}
```

#### get_content_history

Retrieves version history for content.

**Parameters:**
```typescript
interface GetContentHistoryParams extends StandardParams {
  operation: "get_content_history";
  content_id: string;                 // Content ID
  limit?: number;                     // Version limit
  include_diffs?: boolean;            // Include change diffs
}
```

## Tool 3: memory_analyze

Handles all analysis and intelligence operations.

### Operations

#### detect_patterns

Identifies patterns in content and behavior.

**Parameters:**
```typescript
interface DetectPatternsParams extends StandardParams {
  operation: "detect_patterns";
  scope?: "project" | "session" | "content";
  content_id?: string;                // Analyze specific content
  pattern_types?: PatternType[];      // Types of patterns to detect
  options?: {
    min_confidence?: number;          // Minimum confidence threshold
    include_explanations?: boolean;   // Include pattern explanations
    time_range?: {
      start: string;
      end: string;
    };
  };
}

type PatternType = 
  | "topic_clusters" | "usage_patterns" | "quality_issues"
  | "relationship_patterns" | "temporal_patterns" | "user_patterns";
```

#### analyze_quality

Analyzes content quality and provides improvement suggestions.

**Parameters:**
```typescript
interface AnalyzeQualityParams extends StandardParams {
  operation: "analyze_quality";
  content_id: string;                 // Content to analyze
  quality_dimensions?: QualityDimension[];
  options?: {
    include_suggestions?: boolean;    // Include improvement suggestions
    include_metrics?: boolean;        // Include quality metrics
    benchmark_against?: string[];     // Compare to other content
  };
}

type QualityDimension = 
  | "clarity" | "completeness" | "accuracy" | "relevance"
  | "structure" | "consistency" | "readability" | "actionability";
```

#### find_content_relationships

Discovers relationships between content items.

**Parameters:**
```typescript
interface FindRelationshipsParams extends StandardParams {
  operation: "find_content_relationships";
  content_id: string;                 // Source content
  relationship_types?: RelationshipType[];
  max_depth?: number;                 // Relationship traversal depth
  limit?: number;                     // Results limit
  options?: {
    include_strength?: boolean;       // Include relationship strength
    include_context?: boolean;        // Include relationship context
    min_confidence?: number;          // Minimum confidence threshold
  };
}
```

#### detect_conflicts

Identifies conflicting information across content.

**Parameters:**
```typescript
interface DetectConflictsParams extends StandardParams {
  operation: "detect_conflicts";
  scope?: "project" | "content_set";
  content_ids?: string[];             // Specific content to check
  conflict_types?: ConflictType[];
  options?: {
    sensitivity?: "high" | "medium" | "low";
    include_resolution_suggestions?: boolean;
  };
}

type ConflictType = 
  | "factual_conflicts" | "recommendation_conflicts" 
  | "temporal_conflicts" | "logical_conflicts";
```

#### generate_insights

Generates intelligent insights from content analysis.

**Parameters:**
```typescript
interface GenerateInsightsParams extends StandardParams {
  operation: "generate_insights";
  scope?: "project" | "session" | "content_set";
  content_ids?: string[];             // Specific content to analyze
  insight_types?: InsightType[];
  options?: {
    depth?: "surface" | "deep" | "comprehensive";
    include_recommendations?: boolean;
    time_horizon?: "short" | "medium" | "long";
  };
}

type InsightType = 
  | "content_gaps" | "usage_insights" | "trend_analysis"
  | "efficiency_opportunities" | "knowledge_synthesis";
```

## Tool 4: memory_system

Handles all system administration and configuration operations.

### Operations

#### check_system_health

Retrieves system health status and metrics.

**Parameters:**
```typescript
interface SystemHealthParams extends StandardParams {
  operation: "check_system_health";
  detailed?: boolean;                 // Include detailed diagnostics
  components?: string[];              // Specific components to check
}
```

**Response:**
```typescript
interface SystemHealthResponse {
  success: boolean;
  timestamp: string;
  health: {
    status: "healthy" | "degraded" | "unhealthy";
    uptime: string;                   // Uptime duration
    version: string;                  // Server version
    components: {
      database: ComponentHealth;
      vector_store: ComponentHealth;
      ai_service: ComponentHealth;
      cache: ComponentHealth;
    };
    performance: {
      response_time: string;
      throughput: number;
      error_rate: number;
    };
    resources: {
      memory_usage: string;
      storage_usage: string;
      connection_count: number;
    };
  };
}

interface ComponentHealth {
  status: "healthy" | "degraded" | "unhealthy";
  message?: string;
  last_check: string;
  metrics?: Record<string, any>;
}
```

#### export_project_data

Exports project data in various formats.

**Parameters:**
```typescript
interface ExportProjectParams extends StandardParams {
  operation: "export_project_data";
  format: "json" | "csv" | "markdown" | "zip";
  include?: {
    content?: boolean;                // Include content data
    relationships?: boolean;          // Include relationships
    metadata?: boolean;               // Include metadata
    history?: boolean;                // Include version history
  };
  filters?: {
    date_range?: DateRange;
    content_types?: string[];
    tags?: string[];
  };
  options?: {
    compress?: boolean;               // Compress export
    include_embeddings?: boolean;     // Include vector data
    anonymize?: boolean;              // Remove personal data
  };
}
```

#### import_project_data

Imports project data from external sources.

**Parameters:**
```typescript
interface ImportProjectParams extends StandardParams {
  operation: "import_project_data";
  source: string;                     // Data source
  format: "json" | "csv" | "markdown";
  data?: string;                      // Inline data
  options?: {
    merge_strategy?: "replace" | "merge" | "skip_existing";
    validate_data?: boolean;          // Validate before import
    generate_embeddings?: boolean;    // Generate embeddings
    detect_relationships?: boolean;   // Auto-detect relationships
    preserve_ids?: boolean;           // Preserve original IDs
  };
}
```

#### validate_data_integrity

Validates data integrity across the system.

**Parameters:**
```typescript
interface ValidateIntegrityParams extends StandardParams {
  operation: "validate_data_integrity";
  scope?: "project" | "session" | "system";
  checks?: IntegrityCheck[];
  options?: {
    fix_issues?: boolean;             // Auto-fix detected issues
    detailed_report?: boolean;        // Include detailed report
  };
}

type IntegrityCheck = 
  | "orphaned_relationships" | "missing_embeddings" 
  | "data_consistency" | "reference_integrity"
  | "schema_compliance" | "performance_issues";
```

#### generate_citation

Generates academic citations for content.

**Parameters:**
```typescript
interface GenerateCitationParams extends StandardParams {
  operation: "generate_citation";
  content_id: string;                 // Content to cite
  style?: "apa" | "mla" | "chicago" | "ieee" | "harvard";
  options?: {
    include_permalink?: boolean;      // Include permanent link
    include_access_date?: boolean;    // Include access date
  };
}
```

## Error Handling

### Error Response Format

```typescript
interface ErrorResponse {
  jsonrpc: "2.0";
  error: {
    code: number;                     // Error code
    message: string;                  // Human-readable message
    data?: {
      error_type: string;             // Categorized error type
      details: Record<string, any>;   // Error details
      suggestions?: string[];         // Recovery suggestions
      documentation_url?: string;    // Help documentation
    };
  };
  id: any;                           // Request ID
}
```

### Error Categories

#### Client Errors (4xx codes)

- **400 - Bad Request**: Invalid parameters or malformed request
- **401 - Unauthorized**: Authentication required
- **403 - Forbidden**: Insufficient permissions
- **404 - Not Found**: Resource not found
- **409 - Conflict**: Resource conflict (e.g., duplicate ID)
- **422 - Validation Error**: Parameter validation failed

#### Server Errors (5xx codes)

- **500 - Internal Error**: Unexpected server error
- **502 - Service Unavailable**: External service unavailable
- **503 - Rate Limited**: Too many requests
- **504 - Timeout**: Operation timeout

### Error Examples

#### Validation Error
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": 422,
    "message": "Parameter validation failed",
    "data": {
      "error_type": "validation_error",
      "details": {
        "field": "project_id",
        "violation": "required",
        "provided": null
      },
      "suggestions": [
        "Provide a valid project_id parameter",
        "Use an existing project ID from your account"
      ],
      "documentation_url": "https://docs.example.com/api/parameters"
    }
  },
  "id": 1
}
```

#### Not Found Error
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": 404,
    "message": "Content not found",
    "data": {
      "error_type": "resource_not_found",
      "details": {
        "resource_type": "content",
        "resource_id": "content_xyz789",
        "project_id": "my-project"
      },
      "suggestions": [
        "Verify the content ID is correct",
        "Check if the content was deleted",
        "Ensure you have access to this project"
      ]
    }
  },
  "id": 1
}
```

## Rate Limits

### Default Limits

- **Global**: 1000 requests per minute
- **Per Session**: 100 requests per minute  
- **Heavy Operations**: 10 requests per minute (analysis, export)

### Rate Limit Headers

Response headers indicate current rate limit status:

```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 945
X-RateLimit-Reset: 1638360000
X-RateLimit-Type: global
```

## Authentication

### Session-Based Authentication

Most operations require a valid session:

```json
{
  "project_id": "my-project",
  "session_id": "session-abc123"
}
```

### API Key Authentication (Optional)

For programmatic access:

```json
{
  "headers": {
    "Authorization": "Bearer api-key-xyz789"
  }
}
```

## SDK Examples

### JavaScript/TypeScript

```typescript
import { MCPMemoryClient } from '@lerian/mcp-memory-client';

const client = new MCPMemoryClient({
  endpoint: 'ws://localhost:9080/ws'
});

// Store content
const result = await client.memoryStore({
  operation: 'store_content',
  project_id: 'my-project',
  content: 'Important information to remember',
  tags: ['important', 'memory']
});

// Search content
const searchResults = await client.memoryRetrieve({
  operation: 'search_content',
  project_id: 'my-project',
  query: 'important information',
  options: { limit: 5 }
});
```

### Python

```python
from lerian_mcp_memory import MCPMemoryClient

client = MCPMemoryClient(endpoint='ws://localhost:9080/ws')

# Store content
result = client.memory_store(
    operation='store_content',
    project_id='my-project',
    content='Important information to remember',
    tags=['important', 'memory']
)

# Search content
search_results = client.memory_retrieve(
    operation='search_content',
    project_id='my-project',
    query='important information',
    options={'limit': 5}
)
```

### Go

```go
package main

import (
    "github.com/lerian/mcp-memory-client-go"
)

func main() {
    client := mcpmemory.NewClient("ws://localhost:9080/ws")
    
    // Store content
    result, err := client.MemoryStore(context.Background(), &mcpmemory.StoreContentRequest{
        Operation: "store_content",
        ProjectID: "my-project",
        Content:   "Important information to remember",
        Tags:      []string{"important", "memory"},
    })
    
    // Search content
    searchResults, err := client.MemoryRetrieve(context.Background(), &mcpmemory.SearchContentRequest{
        Operation: "search_content",
        ProjectID: "my-project",
        Query:     "important information",
        Options:   &mcpmemory.SearchOptions{Limit: 5},
    })
}
```

This API reference provides comprehensive documentation for all available operations, parameters, and response formats in the MCP Memory Server v2.