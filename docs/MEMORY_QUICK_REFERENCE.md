# MCP Memory Quick Reference Card

## 🎯 When to Use Each Tool

### Starting a Session
```bash
mcp__memory__memory_get_context
├── repository: "project-name"  # or "_global"
└── recent_days: 7              # last week's activity
```
**Returns**: Recent memories, patterns, common issues

### Before Solving a Problem
```bash
mcp__memory__memory_search
├── query: "specific error or problem description"
├── repository: "project-name"
├── types: ["problem", "solution"]
└── recency: "all_time"

# OR for similar problems:
mcp__memory__memory_find_similar
├── problem: "detailed error description"
└── repository: "project-name"
```

### After Solving Something
```bash
mcp__memory__memory_store_chunk
├── content: "Problem: X happened because Y. Solution: Did Z by..."
├── session_id: "current-session-id"
├── repository: "project-name"
├── files_modified: ["file1.go", "file2.go"]
├── tools_used: ["Read", "Edit", "Bash"]
└── tags: ["bug-fix", "timeout", "performance"]
```

### After Making a Decision
```bash
mcp__memory__memory_store_decision
├── decision: "Use Redis for session storage"
├── rationale: "Need distributed sessions for horizontal scaling"
├── context: "Considered: in-memory (not scalable), DB (too slow)"
├── repository: "project-name"
└── session_id: "current-session-id"
```

### During Review/Retrospective
```bash
mcp__memory__memory_get_patterns
├── repository: "project-name"
└── timeframe: "month"  # or "week", "quarter", "all"
```

## 📊 Tool Decision Tree

```
Need memory help?
│
├── 🔍 Looking for something?
│   ├── General search → memory_search
│   └── Similar problems → memory_find_similar
│
├── 💾 Want to save something?
│   ├── General conversation → memory_store_chunk
│   └── Specific decision → memory_store_decision
│
├── 📋 Need context?
│   ├── Project overview → memory_get_context
│   └── Patterns/trends → memory_get_patterns
│
└── 🏥 System check → memory_health
```

## 🏷️ Tag Suggestions

**Problem Types**:
- `bug-fix`, `error`, `crash`, `performance`, `memory-leak`, `timeout`

**Feature Areas**:
- `api`, `database`, `frontend`, `backend`, `infrastructure`, `security`

**Decision Types**:
- `architecture`, `design`, `technology-choice`, `trade-off`

**Learning**:
- `til` (today I learned), `pattern`, `best-practice`, `gotcha`

## 💡 Pro Tips

1. **Search First**: Always search before implementing
2. **Be Specific**: More context = better future matches
3. **Tag Consistently**: Use standard tags for better retrieval
4. **Store Failures**: Failed attempts are valuable learning
5. **Global vs Local**: Use `_global` for cross-project knowledge

## 🔧 Common Patterns

### Bug Fix Flow
```python
# 1. Search for similar
similar = memory_find_similar(problem="connection timeout error")

# 2. After fixing
memory_store_chunk(
    content="Fixed timeout by increasing limit and adding retry",
    tags=["bug-fix", "timeout", "connection"]
)
```

### Decision Flow
```python
# 1. Search past decisions
past = memory_search(query="database choice architecture", types=["decision"])

# 2. Store new decision
memory_store_decision(
    decision="Switch from PostgreSQL to CockroachDB",
    rationale="Need global distribution and automatic sharding"
)
```

### Learning Flow
```python
# 1. Store what you learned
memory_store_chunk(
    content="Learned that Go interfaces are satisfied implicitly...",
    tags=["til", "golang", "interfaces"]
)

# 2. Later, search for it
memory_search(query="golang interfaces implicit", repository="_global")
```