# MCP Memory Server v2 - Data Model Documentation

## Overview

The MCP Memory Server v2 features a clean, intuitive data model built around three core domains with clear separation of concerns. This document describes the complete data model, entity relationships, and lifecycle management.

## Core Architecture

### Domain Separation

The v2 architecture separates functionality into three distinct domains:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Memory Domain  │    │   Task Domain   │    │ System Domain   │
│                 │    │                 │    │                 │
│ • Content       │    │ • Tasks         │    │ • Health        │
│ • Search        │    │ • Workflows     │    │ • Sessions      │
│ • Relationships │    │ • Templates     │    │ • Data Export   │
│ • Intelligence  │    │ • Metrics       │    │ • Citations     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
        │                       │                       │
        └───────────────────────┼───────────────────────┘
                                │
                    ┌─────────────────┐
                    │   Coordinator   │
                    │                 │
                    │ • Cross-domain  │
                    │ • Orchestration │
                    │ • Integration   │
                    └─────────────────┘
```

### Core Identifiers

All entities in v2 use consistent, clean identifiers:

- **ProjectID**: Tenant/project isolation (`project_abc123`)
- **SessionID**: User session scoping (`session_xyz789`)  
- **ContentID**: Unique content identifier (`content_def456`)
- **TaskID**: Unique task identifier (`task_ghi789`)

## Memory Domain Entities

### Content

The primary knowledge storage entity.

```go
type Content struct {
    ID          string                 `json:"id"`
    ProjectID   ProjectID              `json:"project_id"`
    SessionID   SessionID              `json:"session_id,omitempty"`
    Content     string                 `json:"content"`
    Summary     string                 `json:"summary,omitempty"`
    ContentType string                 `json:"content_type"`
    
    // Organization
    Tags        []string               `json:"tags,omitempty"`
    Categories  []string               `json:"categories,omitempty"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
    
    // Timestamps
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
    AccessedAt  *time.Time             `json:"accessed_at,omitempty"`
    
    // Version Control
    Version     int                    `json:"version"`
    ParentID    string                 `json:"parent_id,omitempty"`
    
    // Vector Data
    Embedding   []float32              `json:"embedding,omitempty"`
    EmbeddingModel string              `json:"embedding_model,omitempty"`
}
```

#### Content Lifecycle

1. **Creation**: Content stored with automatic embedding generation
2. **Versioning**: Updates create new versions, preserving history
3. **Relationships**: Automatically detected and manually created links
4. **Analysis**: AI-powered pattern detection and quality analysis
5. **Archival**: Soft deletion with retention policies

#### Content Types

- `text/plain` - Plain text content
- `text/markdown` - Markdown formatted content
- `application/json` - Structured JSON data
- `text/code` - Source code with language detection
- `text/documentation` - Technical documentation
- `text/conversation` - Chat/conversation records

### Relationships

Connections between content entities.

```go
type Relationship struct {
    ID          string                 `json:"id"`
    ProjectID   ProjectID              `json:"project_id"`
    SourceID    string                 `json:"source_id"`
    TargetID    string                 `json:"target_id"`
    Type        RelationshipType       `json:"type"`
    Strength    float64                `json:"strength"`
    Direction   RelationshipDirection  `json:"direction"`
    
    // Context
    Context     string                 `json:"context,omitempty"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
    
    // Detection
    DetectedBy  string                 `json:"detected_by"` // "manual", "ai", "rule"
    Confidence  float64                `json:"confidence"`
    
    // Timestamps
    CreatedAt   time.Time              `json:"created_at"`
    ValidatedAt *time.Time             `json:"validated_at,omitempty"`
}
```

#### Relationship Types

- **Semantic**: Content similarity (`similar_to`, `related_to`)
- **Hierarchical**: Parent-child relationships (`contains`, `part_of`)
- **Temporal**: Time-based connections (`follows`, `precedes`)
- **Causal**: Cause-effect relationships (`causes`, `resolves`)
- **Reference**: Direct references (`references`, `cites`)

### Search and Discovery

```go
type SearchQuery struct {
    ProjectID     ProjectID    `json:"project_id"`
    SessionID     SessionID    `json:"session_id,omitempty"`
    Query         string       `json:"query"`
    QueryType     string       `json:"query_type"` // "semantic", "keyword", "hybrid"
    
    // Filtering
    Filters       *Filters     `json:"filters,omitempty"`
    ContentTypes  []string     `json:"content_types,omitempty"`
    DateRange     *DateRange   `json:"date_range,omitempty"`
    
    // Pagination
    Limit         int          `json:"limit,omitempty"`
    Offset        int          `json:"offset,omitempty"`
    
    // Relevance
    MinRelevance  float64      `json:"min_relevance,omitempty"`
    BoostRecent   bool         `json:"boost_recent,omitempty"`
}

type SearchResult struct {
    Content     *Content   `json:"content"`
    Relevance   float64    `json:"relevance"`
    Highlights  []string   `json:"highlights,omitempty"`
    Context     string     `json:"context,omitempty"`
    Explanation string     `json:"explanation,omitempty"`
}
```

## Task Domain Entities

### Task

Core task management entity with workflow support.

```go
type Task struct {
    ID          string                 `json:"id"`
    ProjectID   ProjectID              `json:"project_id"`
    SessionID   SessionID              `json:"session_id,omitempty"`
    
    // Core Fields
    Title       string                 `json:"title"`
    Description string                 `json:"description,omitempty"`
    Status      TaskStatus             `json:"status"`
    Priority    TaskPriority           `json:"priority"`
    Type        TaskType               `json:"type,omitempty"`
    
    // Assignment
    AssigneeID  string                 `json:"assignee_id,omitempty"`
    CreatedBy   string                 `json:"created_by,omitempty"`
    ReviewerID  string                 `json:"reviewer_id,omitempty"`
    
    // Hierarchy
    ParentID    string                 `json:"parent_id,omitempty"`
    SubtaskIDs  []string               `json:"subtask_ids,omitempty"`
    
    // Timing
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
    DueDate     *time.Time             `json:"due_date,omitempty"`
    StartDate   *time.Time             `json:"start_date,omitempty"`
    CompletedAt *time.Time             `json:"completed_at,omitempty"`
    
    // Effort Tracking
    EstimatedMins int                  `json:"estimated_mins,omitempty"`
    ActualMins    int                  `json:"actual_mins,omitempty"`
    
    // Organization
    Tags        []string               `json:"tags,omitempty"`
    Labels      []string               `json:"labels,omitempty"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
    
    // Content References (Clean Domain Separation)
    LinkedContentIDs []string          `json:"linked_content_ids,omitempty"`
    
    // Version Control
    Version     int                    `json:"version"`
    Workflow    string                 `json:"workflow,omitempty"`
}
```

#### Task Status Flow

```
backlog → todo → in_progress → in_review → completed
   ↓        ↓         ↓           ↓
blocked ← blocked ← blocked ← blocked
   ↓        ↓         ↓           ↓
cancelled ← cancelled ← cancelled ← cancelled
```

#### Task Priorities

- **Critical**: Urgent, blocking issues requiring immediate attention
- **High**: Important tasks that should be completed soon
- **Medium**: Standard priority tasks in normal workflow
- **Low**: Nice-to-have tasks that can be deferred

### Task Dependencies

```go
type TaskDependency struct {
    ID           string          `json:"id"`
    ProjectID    ProjectID       `json:"project_id"`
    TaskID       string          `json:"task_id"`
    DependsOnID  string          `json:"depends_on_id"`
    Type         DependencyType  `json:"type"`
    CreatedAt    time.Time       `json:"created_at"`
    Metadata     map[string]interface{} `json:"metadata,omitempty"`
}
```

#### Dependency Types

- **blocked_by**: Task cannot start until dependency completes
- **subtask_of**: Task is a component of another task
- **related_to**: Tasks are related but not blocking
- **duplicate_of**: Task duplicates another task

### Task Templates

```go
type TaskTemplate struct {
    ID          string                 `json:"id"`
    ProjectID   ProjectID              `json:"project_id"`
    Name        string                 `json:"name"`
    Description string                 `json:"description,omitempty"`
    Category    string                 `json:"category,omitempty"`
    Template    map[string]interface{} `json:"template"`
    Variables   []TemplateVariable     `json:"variables,omitempty"`
    CreatedAt   time.Time              `json:"created_at"`
    CreatedBy   string                 `json:"created_by"`
}
```

## System Domain Entities

### Session

User session management with access control.

```go
type Session struct {
    ID          SessionID              `json:"id"`
    ProjectID   ProjectID              `json:"project_id"`
    UserID      string                 `json:"user_id,omitempty"`
    
    // Access Control
    AccessLevel string                 `json:"access_level"`
    Permissions []string               `json:"permissions,omitempty"`
    
    // Session Data
    CreatedAt   time.Time              `json:"created_at"`
    LastActive  time.Time              `json:"last_active"`
    ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
    
    // Context
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
    UserAgent   string                 `json:"user_agent,omitempty"`
    IPAddress   string                 `json:"ip_address,omitempty"`
    
    // State
    Active      bool                   `json:"active"`
}
```

#### Access Levels

- **read_only**: Can view project data without session_id
- **session**: Full access to session data and project data with session_id
- **project**: Access to all project data regardless of session
- **admin**: Full administrative access

### Health Status

```go
type HealthStatus struct {
    Status      string                 `json:"status"` // "healthy", "degraded", "unhealthy"
    Timestamp   time.Time              `json:"timestamp"`
    Version     string                 `json:"version"`
    Uptime      time.Duration          `json:"uptime"`
    
    // Component Health
    Components  map[string]ComponentHealth `json:"components"`
    
    // Resource Usage
    Memory      *MemoryUsage           `json:"memory,omitempty"`
    Storage     *StorageUsage          `json:"storage,omitempty"`
    
    // Performance
    ResponseTime time.Duration         `json:"response_time"`
    Throughput   float64               `json:"throughput"`
    
    // Error Rates
    ErrorRate    float64               `json:"error_rate"`
    Details      map[string]interface{} `json:"details,omitempty"`
}
```

## Cross-Domain Integration

### Clean Domain Separation

The v2 architecture maintains strict domain boundaries while enabling cross-domain operations:

#### Memory ↔ Task Integration

Tasks can reference memory content through clean interfaces:

```go
// Task references content without mixing domains
type Task struct {
    // ... other fields
    LinkedContentIDs []string `json:"linked_content_ids,omitempty"`
}

// Coordinator manages cross-domain operations
coordinator.LinkTaskToContent(taskID, contentID, linkType)
coordinator.GenerateTasksFromContent(contentID, options)
coordinator.CreateContentFromTask(taskID, contentType)
```

#### Benefits of Clean Separation

1. **Maintainability**: Each domain can evolve independently
2. **Testability**: Domains can be tested in isolation
3. **Scalability**: Domains can be scaled independently
4. **Clarity**: Clear responsibilities and boundaries

### Cross-Domain Operations

The Domain Coordinator manages operations that span multiple domains:

#### Task-Content Linking

```go
type LinkTaskToContentRequest struct {
    TaskID    string `json:"task_id"`
    ContentID string `json:"content_id"`
    LinkType  string `json:"link_type"` // "references", "created_from", "depends_on"
}
```

#### Content-Based Task Generation

```go
type GenerateTasksFromContentRequest struct {
    ContentID string                 `json:"content_id"`
    Options   map[string]interface{} `json:"options,omitempty"`
}
```

## Migration from v1

### Key Changes

1. **Parameter Simplification**: `repository` → `project_id`
2. **Session Logic**: Fixed backwards session semantics
3. **Tool Consolidation**: 9 fragmented tools → 4 clean tools
4. **Domain Separation**: Mixed responsibilities → clean boundaries

### Migration Guide

#### Parameter Updates

```diff
// v1 (confusing)
- "repository": "my-repo"
+ "project_id": "my-repo"

// v1 (backwards logic)
- session_id provides LESS access
+ session_id provides MORE access
```

#### Tool Mapping

```diff
// v1 → v2 Tool Mapping
- memory_tasks → memory_store (task operations)
- memory_create → memory_store (content creation)
- memory_read → memory_retrieve (content access)
- memory_update → memory_store (content updates)
- memory_delete → memory_store (content deletion)
- memory_analyze → memory_analyze (analysis operations)
- memory_intelligence → memory_analyze (AI operations)
- memory_system → memory_system (admin operations)
- memory_transfer → memory_system (data operations)
```

#### Data Preservation

All existing data is preserved through database migration:

1. **Column Rename**: `repository` → `project_id` across all tables
2. **Index Updates**: Update indexes for new column names
3. **Constraint Updates**: Update foreign keys and constraints
4. **Data Validation**: Ensure data integrity during migration

## Best Practices

### Content Organization

1. **Use Clear Tags**: Tag content with descriptive, consistent tags
2. **Meaningful Summaries**: Write concise, informative summaries
3. **Rich Metadata**: Include relevant metadata for better searchability
4. **Consistent Types**: Use standard content types for better organization

### Task Management

1. **Clear Titles**: Write descriptive, actionable task titles
2. **Proper Hierarchy**: Use subtasks for complex work breakdown
3. **Realistic Estimates**: Provide accurate time estimates
4. **Link Related Content**: Reference relevant memory content

### Cross-Domain Integration

1. **Minimal Coupling**: Keep domain interactions minimal and well-defined
2. **Clear References**: Use explicit linking rather than implicit dependencies
3. **Coordinator Usage**: Use the coordinator for cross-domain operations
4. **Domain Boundaries**: Respect domain boundaries in all operations

## Performance Considerations

### Indexing Strategy

1. **Primary Keys**: All entities have optimized primary key indexes
2. **Project Isolation**: Efficient `project_id` indexing for tenant isolation
3. **Search Optimization**: Vector indexes for semantic search
4. **Relationship Indexing**: Optimized relationship traversal

### Caching Strategy

1. **Content Caching**: Frequently accessed content cached in memory
2. **Search Caching**: Search results cached with TTL
3. **Relationship Caching**: Relationship graphs cached for traversal
4. **Session Caching**: Active sessions cached for quick access

### Scalability

1. **Horizontal Scaling**: Domains can be scaled independently
2. **Data Partitioning**: Project-based data partitioning
3. **Read Replicas**: Read-heavy operations use dedicated replicas
4. **Vector Store Scaling**: Qdrant clustering for vector operations

This data model provides a clean, scalable foundation for the MCP Memory Server v2, with clear domain separation, intuitive relationships, and comprehensive lifecycle management.