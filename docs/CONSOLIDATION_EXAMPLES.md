# MCP Tool Usage Examples

**Quick Start Guide** | 9 Consolidated Tools | 78% Tool Reduction

## Common Patterns

### 1. Bug Fix Workflow
```json
// Store bug fix
{"name": "memory_create", "arguments": {"operation": "store_chunk", "scope": "single", "options": {"content": "Fixed JWT expiration bug", "session_id": "bugfix_123", "repository": "github.com/org/api", "tags": ["bug-fix", "auth"]}}}

// Search similar issues
{"name": "memory_read", "arguments": {"operation": "search", "scope": "single", "options": {"query": "JWT authentication bug", "repository": "github.com/org/api", "limit": 5}}}

// Get project status
{"name": "memory_system", "arguments": {"operation": "status", "scope": "repository", "options": {"repository": "github.com/org/api"}}}
```

### 2. Architecture Decision
```json
// Store decision
{"name": "memory_create", "arguments": {"operation": "store_decision", "scope": "single", "options": {"decision": "Migrate to GraphQL", "rationale": "Better performance, 40% data reduction", "session_id": "arch_123", "repository": "github.com/org/api"}}}

// Find similar decisions
{"name": "memory_read", "arguments": {"operation": "find_similar", "scope": "single", "options": {"problem": "API technology selection", "repository": "github.com/org/api"}}}
```

### 3. Task Management
```json
// Create session
{"name": "memory_tasks", "arguments": {"operation": "session_create", "scope": "session", "options": {"session_id": "task_123", "repository": "github.com/org/api"}}}

// Write todos
{"name": "memory_tasks", "arguments": {"operation": "todo_write", "scope": "session", "options": {"session_id": "task_123", "todos": [{"id": "1", "content": "Fix auth bug", "status": "in_progress", "priority": "high"}]}}}

// Track progress
{"name": "memory_tasks", "arguments": {"operation": "todo_update", "scope": "session", "options": {"session_id": "task_123", "tool_name": "Edit", "tool_context": {"file_path": "auth.go"}}}}

// End session
{"name": "memory_tasks", "arguments": {"operation": "session_end", "scope": "session", "options": {"session_id": "task_123", "outcome": "success"}}}
```

### 4. Health Monitoring
```json
// System health
{"name": "memory_system", "arguments": {"operation": "health", "scope": "system", "options": {}}}

// Detailed dashboard (⚠️ both required)
{"name": "memory_analyze", "arguments": {"operation": "health_dashboard", "scope": "single", "options": {"repository": "github.com/org/api", "session_id": "health_123"}}}

// Check freshness
{"name": "memory_analyze", "arguments": {"operation": "check_freshness", "scope": "single", "options": {"repository": "github.com/org/api"}}}
```

### 5. Cross-Repo Analysis
```json
// Multi-repo search
{"name": "memory_read", "arguments": {"operation": "search_multi_repo", "scope": "cross_repo", "options": {"query": "auth patterns", "session_id": "analysis_123", "repositories": ["github.com/org/api", "github.com/org/web"]}}}

// Pattern analysis
{"name": "memory_analyze", "arguments": {"operation": "cross_repo_patterns", "scope": "cross_repo", "options": {"session_id": "analysis_123", "pattern_types": ["architectural"]}}}
```

### 6. Data Management
```json
// Bulk import
{"name": "memory_create", "arguments": {"operation": "bulk_import", "scope": "bulk", "options": {"data": "base64_data", "format": "archive", "repository": "github.com/org/api"}}}

// Export project (⚠️ repository AND session_id required)
{"name": "memory_transfer", "arguments": {"operation": "export_project", "scope": "project", "options": {"repository": "github.com/org/api", "session_id": "export_123"}}}

// Import context (⚠️ data, repository AND session_id required)
{"name": "memory_transfer", "arguments": {"operation": "import_context", "scope": "single", "options": {"data": "context_data", "repository": "github.com/org/api", "session_id": "import_123"}}}

// Continuity check
{"name": "memory_transfer", "arguments": {"operation": "continuity", "scope": "single", "options": {"repository": "github.com/org/api", "session_id": "continuity_123"}}}
```

### 7. Intelligence Features
```json
// Get suggestions (⚠️ current_context AND session_id required)
{"name": "memory_intelligence", "arguments": {"operation": "suggest_related", "scope": "single", "options": {"current_context": "working on auth", "session_id": "work_123", "repository": "github.com/org/api"}}}

// Auto insights (⚠️ repository AND session_id required)
{"name": "memory_intelligence", "arguments": {"operation": "auto_insights", "scope": "single", "options": {"repository": "github.com/org/api", "session_id": "insights_123", "timeframe": "30d"}}}

// Pattern prediction (⚠️ context, repository AND session_id required)
{"name": "memory_intelligence", "arguments": {"operation": "pattern_prediction", "scope": "single", "options": {"context": "authentication patterns", "repository": "github.com/org/api", "session_id": "predict_123"}}}

// Get documentation
{"name": "memory_system", "arguments": {"operation": "get_documentation", "scope": "system", "options": {"doc_type": "mappings"}}}
```

## Best Practices

### Parameter Structure
```json
{
  "operation": "operation_name",    // Required: what to do
  "scope": "single|bulk|cross_repo", // Required: operation scope  
  "options": {                      // Required: operation parameters
    "required_param": "value",      // * = required
    "optional_param": "value"       // no * = optional
  }
}
```

### Scope Guidelines
- `"single"` - Single item operations (search, store_chunk)
- `"bulk"` - Multiple items (bulk_import, bulk_delete) 
- `"cross_repo"` - Multi-repository (search_multi_repo, patterns)
- `"system"` - System-wide (health, documentation)
- `"session"` - Task/session-specific (todo operations)

### Common Errors to Avoid
- ❌ `chunk_ids` for mark_refreshed → ✅ `chunk_id` (singular)
- ❌ Missing `session_id` for suggest_related → ✅ Both `current_context` AND `session_id` required
- ❌ Missing `repository` for health_dashboard → ✅ Both `repository` AND `session_id` required
- ❌ Using `confidence` for create_relationship → ✅ Use `strength` parameter instead
- ❌ Complex `target` object for create_alias → ✅ Use simple `target` string parameter
- ❌ Missing `name` and `description` for create_thread → ✅ Both required along with `chunk_ids`
- ❌ Invalid `chunk_ids` for create_thread → ✅ Must reference existing valid chunks
- ❌ Missing required fields for intelligence operations → ✅ Check repository/session_id requirements

## Migration Guide

### Legacy Tool → Consolidated Tool
```bash
# Legacy (41 tools)
"mcp__memory__memory_store_chunk" → "memory_create" + "store_chunk"
"mcp__memory__memory_search" → "memory_read" + "search"  
"mcp__memory__memory_health" → "memory_system" + "health"

# Benefits: 78% reduction (41→9), better client compatibility
```

### Testing Migration
```bash
# Test consolidated mode
export MCP_MEMORY_USE_CONSOLIDATED_TOOLS=true
curl -X POST http://localhost:9080/mcp -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'

# Result: 9 tools instead of 41
```

## Quick Reference

**Memory Operations**: create → read → update → delete → analyze → intelligence → transfer → system → tasks

**Token Efficiency**: ~600 tokens (vs ~1500 tokens original)