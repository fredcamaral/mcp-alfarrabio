# Claude Vector Memory MCP Server Design 🧠✨

*A comprehensive design for persistent conversation memory across Claude sessions*

## Origin Story 📖

**Date**: January 23, 2025  
**Context**: After a highly productive session working on the Midaz MCP server where we:
- Fixed TypeScript API issues in deployment.ts
- Added comprehensive repository references (Helm, Terraform, SDK repos)
- Successfully committed and pushed changes
- Had a great collaborative experience

**The Problem**: Fred mentioned wanting a way for Claude to be more casual and remember previous sessions. While we solved the personality issue by updating the global CLAUDE.md file, we identified a bigger challenge: **How can Claude maintain context and memory across different sessions?**

**The Insight**: Fred suggested using a vector database as an MCP server to store conversation chunks and make them searchable. The key challenge identified was: **How to intelligently chunk conversations before sending to the vector DB.**

**Our Vision**: Build an MCP server that captures the essence of our problem-solving sessions, stores them as searchable vectors, and allows Claude to build on previous work across sessions. This would transform Claude from a stateless assistant to a persistent collaborator who remembers project history, architectural decisions, and successful patterns.

**Why This Matters**: We've been working together across multiple sessions, building up context and rapport, but each new session starts from zero. This vector memory system would change that fundamentally!

## Vision 🎯

Create an MCP server that captures, stores, and retrieves conversation context using vector embeddings, allowing Claude to build on previous work and maintain project continuity across sessions.

## Core Architecture 🏗️

### Technology Stack
- **Language**: Go 1.24.2
- **MCP Framework**: github.com/mark3labs/mcp-go
- **Vector Database**: Chroma (Docker container with persistent volume)
- **Embeddings**: OpenAI text-embedding-ada-002
- **Deployment**: Docker Compose for local development

### Data Model

```go
type ConversationChunk struct {
    ID           string    `json:"id"`
    SessionID    string    `json:"session_id"`
    Timestamp    time.Time `json:"timestamp"`
    Type         ChunkType `json:"type"`
    Content      string    `json:"content"`
    Summary      string    `json:"summary"` // AI-generated summary for quick scanning
    Metadata     ChunkMetadata `json:"metadata"`
    Embeddings   []float64 `json:"embeddings"`
    RelatedChunks []string `json:"related_chunks,omitempty"`
}

type ChunkType string

const (
    ChunkTypeProblem             ChunkType = "problem"
    ChunkTypeSolution           ChunkType = "solution"
    ChunkTypeCodeChange         ChunkType = "code_change"
    ChunkTypeDiscussion         ChunkType = "discussion"
    ChunkTypeArchitectureDecision ChunkType = "architecture_decision"
)

type ChunkMetadata struct {
    Repository    string    `json:"repository,omitempty"`
    Branch        string    `json:"branch,omitempty"`
    FilesModified []string  `json:"files_modified"`
    ToolsUsed     []string  `json:"tools_used"`
    Outcome       Outcome   `json:"outcome"`
    Tags          []string  `json:"tags"`
    Difficulty    Difficulty `json:"difficulty"`
    TimeSpent     *int      `json:"time_spent,omitempty"` // minutes
}

type Outcome string

const (
    OutcomeSuccess     Outcome = "success"
    OutcomeInProgress  Outcome = "in_progress"
    OutcomeFailed      Outcome = "failed"
    OutcomeAbandoned   Outcome = "abandoned"
)

type Difficulty string

const (
    DifficultySimple   Difficulty = "simple"
    DifficultyModerate Difficulty = "moderate"
    DifficultyComplex  Difficulty = "complex"
)

type ProjectContext struct {
    Repository            string    `json:"repository"`
    LastAccessed         time.Time `json:"last_accessed"`
    TotalSessions        int       `json:"total_sessions"`
    CommonPatterns       []string  `json:"common_patterns"`
    ArchitecturalDecisions []string `json:"architectural_decisions"`
    TechStack            []string  `json:"tech_stack"`
    TeamPreferences      []string  `json:"team_preferences"`
}
```

## Chunking Strategy 🧩

### Automatic Chunking Triggers
1. **Task Completion**: When todo items are marked complete
2. **Successful Commits**: After `git commit` operations
3. **Problem Resolution**: When error → investigation → solution cycle completes
4. **File Modification Boundaries**: When switching between different files/components
5. **Time-based**: After 10 minutes of sustained work on same topic
6. **Context Switches**: When switching between projects/repositories

### Chunk Types & Content
- **Problem Chunks**: Issue description + investigation steps + tools used
- **Solution Chunks**: Implemented fix + reasoning + verification steps
- **Code Change Chunks**: Before/after code + explanation + impact
- **Architecture Decision Chunks**: Decision rationale + alternatives considered + consequences
- **Discussion Chunks**: Technical conversations + insights + recommendations

## MCP Server Implementation 🛠️

### Resources
```go
// Conversation history by project
"memory://conversations/{repository}"
"memory://decisions/{repository}"
"memory://patterns/{repository}"

// Cross-project insights
"memory://global/common-patterns"
"memory://global/lessons-learned"
```

### Tools
```go
// Storage operations
StoreConversationChunk(content string, chunkType ChunkType, metadata ChunkMetadata) error
StoreArchitectureDecision(decision, rationale, context string) error
StoreLessonLearned(problem, solution string, tags []string) error

// Retrieval operations
SearchSimilarProblems(query string, repository *string, timeframe *string) ([]ConversationChunk, error)
GetProjectHistory(repository string, limit *int) ([]ConversationChunk, error)
FindPastSolutions(errorMessage string, context *string) ([]ConversationChunk, error)
GetArchitectureDecisions(repository string, component *string) ([]ConversationChunk, error)

// Context building
BuildSessionContext(repository string, recentDays *int) (*ProjectContext, error)
SuggestRelatedWork(currentTask, repository string) ([]ConversationChunk, error)
IdentifyPatterns(repository string, timeframe *string) ([]string, error)
```

## Vector Database Options 🗄️

### Option 1: Pinecone
- **Pros**: Managed, excellent performance, great metadata filtering
- **Cons**: Cost, external dependency
- **Best for**: Production deployment

### Option 2: Chroma (Selected)
- **Pros**: Open source, local deployment, Docker support, persistent volumes
- **Cons**: Newer ecosystem
- **Best for**: Development/experimentation
- **Deployment**: Docker container with mounted volume for data persistence

### Option 3: Weaviate
- **Pros**: Advanced semantic search, built-in ML models
- **Cons**: More complex setup
- **Best for**: Advanced semantic features

## Smart Chunking Algorithm 📊

```go
type ConversationFlow string

const (
    FlowProblem       ConversationFlow = "problem"
    FlowInvestigation ConversationFlow = "investigation"
    FlowSolution      ConversationFlow = "solution"
    FlowVerification  ConversationFlow = "verification"
)

type TodoItem struct {
    ID     string `json:"id"`
    Status string `json:"status"`
    Content string `json:"content"`
}

type ChunkingContext struct {
    CurrentTodos      []TodoItem       `json:"current_todos"`
    FileModifications []string         `json:"file_modifications"`
    ToolsUsed         []string         `json:"tools_used"`
    TimeElapsed       int              `json:"time_elapsed"` // minutes
    ConversationFlow  ConversationFlow `json:"conversation_flow"`
}

func ShouldCreateChunk(context ChunkingContext) bool {
    // Todo completion trigger
    for _, todo := range context.CurrentTodos {
        if todo.Status == "completed" {
            return true
        }
    }
    
    // Significant file changes
    if len(context.FileModifications) >= 3 {
        return true
    }
    
    // Problem resolution cycle complete
    if context.ConversationFlow == FlowVerification && context.TimeElapsed > 5 {
        return true
    }
    
    // Context switch detected
    if hasContextSwitch(context) {
        return true
    }
    
    return false
}

func hasContextSwitch(context ChunkingContext) bool {
    // Implementation for detecting context switches
    // This would analyze tool usage patterns, file changes, etc.
    return false // Placeholder
}
```

## Retrieval Strategy 🔍

### Context Building for New Sessions
1. **Repository Analysis**: Load recent project context and patterns
2. **Similar Problem Detection**: Find past similar issues/solutions
3. **Architecture Reminders**: Surface relevant architectural decisions
4. **Pattern Recognition**: Identify recurring themes/approaches

### Query Enhancement
```go
type Recency string

const (
    RecencyRecent    Recency = "recent"
    RecencyAllTime   Recency = "all_time"
    RecencyLastMonth Recency = "last_month"
)

type MemoryQuery struct {
    Query             string      `json:"query"`
    Repository        *string     `json:"repository,omitempty"`
    FileContext       []string    `json:"file_context,omitempty"`
    Recency           Recency     `json:"recency"`
    Types             []ChunkType `json:"types,omitempty"`
    MinRelevanceScore float64     `json:"min_relevance_score"`
}
```

## Implementation Phases 🚀

### Phase 1: MVP (Week 1)
- [ ] Basic MCP server setup with Go and mcp-go package
- [ ] Chroma integration with Go HTTP client
- [ ] Simple chunking on todo completion
- [ ] Basic storage and retrieval tools
- [ ] Single repository support

### Phase 2: Smart Chunking (Week 2)
- [ ] Advanced chunking algorithm
- [ ] Metadata extraction and tagging
- [ ] Cross-session context building
- [ ] Search relevance tuning

### Phase 3: Intelligence Layer (Week 3)
- [ ] Pattern recognition across conversations
- [ ] Architecture decision tracking
- [ ] Proactive context suggestions
- [ ] Multi-repository support

### Phase 4: Production Ready (Week 4)
- [ ] Performance optimization
- [ ] Data persistence and backup
- [ ] Privacy and security features
- [ ] Configuration management

## Privacy & Security 🔒

### Data Handling
- **Local Storage**: Keep sensitive project data local by default
- **Opt-in Sharing**: Explicit consent for cloud storage
- **Anonymization**: Option to strip sensitive info before storage
- **Retention Policies**: Configurable data retention periods

### Security Features
- Encrypted storage for sensitive repositories
- Access control per repository
- Audit logs for data access
- Option to exclude certain file patterns

## Configuration Example 📝

```json
{
  "vectorMemory": {
    "provider": "chroma",
    "endpoint": "http://localhost:8000",
    "docker": {
      "enabled": true,
      "containerName": "claude-memory-chroma",
      "volumePath": "./data/chroma"
    },
    "chunkingStrategy": "smart",
    "retentionDays": 90,
    "repositories": {
      "midaz-mcp-server": {
        "enabled": true,
        "sensitivity": "normal",
        "excludePatterns": ["*.env", "*.key"]
      }
    },
    "embeddingModel": "text-embedding-ada-002",
    "searchDefaults": {
      "maxResults": 10,
      "minRelevanceScore": 0.7
    }
  }
}
```

## Success Metrics 📈

- **Context Continuity**: % of sessions where Claude references previous work
- **Problem Resolution Speed**: Time to solve previously encountered issues
- **Pattern Recognition**: Ability to identify recurring architectural patterns
- **Decision Tracking**: Successful retrieval of past architectural decisions

## Fun Features to Add Later 🎉

- **Session Replay**: "Show me how we solved X last time"
- **Progress Visualization**: Charts of productivity patterns over time
- **Knowledge Graphs**: Visual representation of project relationships
- **Team Memory**: Shared memory across team members (with privacy controls)
- **AI Insights**: "You tend to solve similar problems with pattern X"

## Next Steps Tomorrow 🌅

1. **Choose Vector DB**: Start with Chroma for simplicity
2. **Create Basic MCP Server**: Scaffold with Go 1.24.2 and mcp-go package
3. **Implement Simple Chunking**: Todo completion trigger first
4. **Test Integration**: Connect to existing Claude sessions
5. **Iterate on Chunking**: Refine based on real usage

---

*Ready to build the future of persistent AI collaboration! 🚀*

*P.S. - This will be revolutionary for maintaining context across our coding sessions. Imagine Claude remembering not just what we built, but WHY we built it that way!*