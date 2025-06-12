# MCP Memory Server v2 - Tutorials

## Progressive Learning Path

Learn the MCP Memory Server v2 through hands-on tutorials that build from simple concepts to advanced workflows.

## Tutorial 1: First Steps with Memory Storage

**Duration**: 10 minutes  
**Prerequisites**: Basic understanding of JSON and APIs  
**What you'll learn**: Store, search, and retrieve your first memories

### Step 1: Store Your First Memory

Let's start by storing some important information:

```bash
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_store",
      "arguments": {
        "operation": "store_content",
        "project_id": "tutorial-project",
        "session_id": "tutorial-session",
        "content": "The quadratic formula is x = (-b ± √(b²-4ac)) / 2a. This formula solves any quadratic equation of the form ax² + bx + c = 0.",
        "tags": ["mathematics", "formula", "quadratic"],
        "content_type": "text/plain",
        "options": {
          "generate_embeddings": true
        }
      }
    },
    "id": 1
  }'
```

**Expected Result**: You'll receive a `content_id` that you can use to reference this content later.

### Step 2: Search for Your Memory

Now let's search for information about quadratic equations:

```bash
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_retrieve",
      "arguments": {
        "operation": "search_content",
        "project_id": "tutorial-project",
        "query": "solving quadratic equations",
        "options": {
          "limit": 5,
          "query_type": "semantic"
        }
      }
    },
    "id": 2
  }'
```

**What Happened**: The semantic search found your content even though you searched for "solving quadratic equations" instead of the exact text.

### Step 3: Retrieve Specific Content

Using the `content_id` from Step 1, retrieve the specific content:

```bash
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_retrieve",
      "arguments": {
        "operation": "get_content",
        "project_id": "tutorial-project",
        "content_id": "YOUR_CONTENT_ID_HERE",
        "options": {
          "include_metadata": true
        }
      }
    },
    "id": 3
  }'
```

**Key Insights**:
- Content is automatically embedded for semantic search
- You can find content using natural language queries
- Each piece of content has a unique ID for direct access

---

## Tutorial 2: Building Knowledge Relationships

**Duration**: 15 minutes  
**Prerequisites**: Tutorial 1 completed  
**What you'll learn**: Create and explore relationships between content

### Step 1: Store Related Content

Let's add more mathematical content to build a knowledge base:

```bash
# Store content about discriminant
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_store",
      "arguments": {
        "operation": "store_content",
        "project_id": "tutorial-project",
        "session_id": "tutorial-session",
        "content": "The discriminant of a quadratic equation ax² + bx + c = 0 is b² - 4ac. If the discriminant is positive, there are two real solutions; if zero, one real solution; if negative, no real solutions.",
        "tags": ["mathematics", "discriminant", "quadratic"],
        "content_type": "text/plain"
      }
    },
    "id": 4
  }'

# Store content about completing the square
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_store",
      "arguments": {
        "operation": "store_content",
        "project_id": "tutorial-project",
        "session_id": "tutorial-session",
        "content": "Completing the square is an alternative method to solve quadratic equations. For ax² + bx + c = 0, rearrange to a(x + b/2a)² = (b²-4ac)/4a².",
        "tags": ["mathematics", "completing-square", "quadratic"],
        "content_type": "text/plain"
      }
    },
    "id": 5
  }'
```

### Step 2: Create Explicit Relationships

Link these mathematical concepts together:

```bash
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_store",
      "arguments": {
        "operation": "create_relationship",
        "project_id": "tutorial-project",
        "session_id": "tutorial-session",
        "source_id": "QUADRATIC_FORMULA_CONTENT_ID",
        "target_id": "DISCRIMINANT_CONTENT_ID",
        "type": "related_to",
        "strength": 0.9,
        "context": "Both concepts are fundamental to understanding quadratic equations"
      }
    },
    "id": 6
  }'
```

### Step 3: Find Similar Content

Discover automatically detected relationships:

```bash
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_retrieve",
      "arguments": {
        "operation": "find_similar_content",
        "project_id": "tutorial-project",
        "content_id": "QUADRATIC_FORMULA_CONTENT_ID",
        "limit": 3,
        "threshold": 0.7
      }
    },
    "id": 7
  }'
```

### Step 4: Explore the Knowledge Graph

Find all content related to quadratic equations:

```bash
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_analyze",
      "arguments": {
        "operation": "find_content_relationships",
        "project_id": "tutorial-project",
        "content_id": "QUADRATIC_FORMULA_CONTENT_ID",
        "max_depth": 2,
        "options": {
          "include_strength": true,
          "include_context": true
        }
      }
    },
    "id": 8
  }'
```

**Key Insights**:
- Relationships can be explicit (manually created) or implicit (AI-detected)
- The system builds a knowledge graph of interconnected content
- You can explore related concepts through relationship traversal

---

## Tutorial 3: Decision Documentation Workflow

**Duration**: 20 minutes  
**Prerequisites**: Tutorials 1-2 completed  
**What you'll learn**: Document important decisions with context and rationale

### Step 1: Document an Architecture Decision

Let's document a technology choice decision:

```bash
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_store",
      "arguments": {
        "operation": "store_decision",
        "project_id": "tutorial-project",
        "session_id": "tutorial-session",
        "title": "Database Choice: PostgreSQL vs MongoDB",
        "context": "Our application needs to store user data, preferences, and activity logs. We need to choose between relational and document databases.",
        "decision": "Choose PostgreSQL as the primary database",
        "rationale": "PostgreSQL provides ACID compliance, excellent performance for complex queries, strong consistency guarantees, and supports both relational and JSON data types. Our data has clear relationships that benefit from a relational model.",
        "alternatives": [
          "MongoDB - Better for unstructured data but lacks strong consistency",
          "MySQL - Good performance but limited JSON support",
          "SQLite - Too limited for production scale"
        ],
        "impact": "Strong data integrity, easier complex queries, team familiarity with SQL",
        "stakeholders": ["engineering-team", "data-team", "product-team"],
        "tags": ["architecture", "database", "postgresql", "decision"],
        "metadata": {
          "decision_date": "2024-12-06",
          "review_date": "2025-06-06",
          "confidence_level": "high",
          "reversibility": "medium"
        }
      }
    },
    "id": 9
  }'
```

### Step 2: Store Supporting Documentation

Add technical analysis that supports the decision:

```bash
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_store",
      "arguments": {
        "operation": "store_content",
        "project_id": "tutorial-project",
        "session_id": "tutorial-session",
        "content": "Performance Benchmark Results:\\n\\nPostgreSQL:\\n- Read queries: 1,200 QPS\\n- Write queries: 800 QPS\\n- Complex joins: 150 QPS\\n- Storage efficiency: 85%\\n\\nMongoDB:\\n- Read queries: 1,800 QPS\\n- Write queries: 1,200 QPS\\n- Aggregation: 200 QPS\\n- Storage efficiency: 70%\\n\\nConclusion: PostgreSQL provides better consistency and join performance, which aligns with our relational data model.",
        "tags": ["benchmark", "performance", "postgresql", "mongodb"],
        "content_type": "text/plain",
        "metadata": {
          "document_type": "benchmark_report",
          "related_decision": "database-choice-postgresql"
        }
      }
    },
    "id": 10
  }'
```

### Step 3: Link Decision to Supporting Evidence

Create a relationship between the decision and the benchmark:

```bash
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_store",
      "arguments": {
        "operation": "create_relationship",
        "project_id": "tutorial-project",
        "session_id": "tutorial-session",
        "source_id": "DECISION_CONTENT_ID",
        "target_id": "BENCHMARK_CONTENT_ID",
        "type": "cites",
        "strength": 0.95,
        "context": "Decision is supported by performance benchmark data"
      }
    },
    "id": 11
  }'
```

### Step 4: Search for Related Decisions

Find other architecture decisions:

```bash
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_retrieve",
      "arguments": {
        "operation": "search_content",
        "project_id": "tutorial-project",
        "query": "architecture decisions database technology",
        "filters": {
          "tags": ["architecture", "decision"],
          "content_types": ["text/plain"]
        },
        "options": {
          "limit": 10,
          "include_highlights": true
        }
      }
    },
    "id": 12
  }'
```

### Step 5: Generate Decision Insights

Analyze decision patterns:

```bash
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_analyze",
      "arguments": {
        "operation": "detect_patterns",
        "project_id": "tutorial-project",
        "pattern_types": ["topic_clusters", "relationship_patterns"],
        "options": {
          "min_confidence": 0.7,
          "include_explanations": true
        }
      }
    },
    "id": 13
  }'
```

**Key Insights**:
- Decisions can be structured with context, rationale, and alternatives
- Supporting evidence can be linked to decisions
- Pattern analysis helps identify decision-making trends
- Metadata enables decision review and lifecycle management

---

## Tutorial 4: Quality Analysis and Improvement

**Duration**: 25 minutes  
**Prerequisites**: Tutorials 1-3 completed  
**What you'll learn**: Analyze content quality and get AI-powered improvement suggestions

### Step 1: Store Content for Analysis

Let's create some content that could be improved:

```bash
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_store",
      "arguments": {
        "operation": "store_content",
        "project_id": "tutorial-project",
        "session_id": "tutorial-session",
        "content": "API docs. GET /users returns users. POST /users creates user. PUT /users/:id updates user. DELETE /users/:id deletes user. All endpoints need auth token in header.",
        "tags": ["documentation", "api", "users"],
        "content_type": "text/plain",
        "metadata": {
          "document_type": "api_documentation",
          "status": "draft"
        }
      }
    },
    "id": 14
  }'
```

### Step 2: Analyze Content Quality

Get a comprehensive quality analysis:

```bash
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_analyze",
      "arguments": {
        "operation": "analyze_quality",
        "project_id": "tutorial-project",
        "content_id": "API_DOCS_CONTENT_ID",
        "quality_dimensions": [
          "clarity",
          "completeness",
          "structure",
          "actionability"
        ],
        "options": {
          "include_suggestions": true,
          "include_metrics": true
        }
      }
    },
    "id": 15
  }'
```

**Expected Analysis Results**:
- **Clarity**: Low - Too abbreviated, lacks examples
- **Completeness**: Medium - Missing error codes, request/response formats
- **Structure**: Low - No proper formatting or sections
- **Actionability**: Medium - Covers basic operations but lacks detail

### Step 3: Store Improved Content

Based on the suggestions, create improved documentation:

```bash
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_store",
      "arguments": {
        "operation": "store_content",
        "project_id": "tutorial-project",
        "session_id": "tutorial-session",
        "content": "# User Management API\\n\\n## Overview\\nThe User Management API provides endpoints for creating, reading, updating, and deleting user accounts.\\n\\n## Authentication\\nAll endpoints require a Bearer token in the Authorization header:\\n```\\nAuthorization: Bearer your-token-here\\n```\\n\\n## Endpoints\\n\\n### GET /users\\n**Description**: Retrieve all users\\n**Response**: 200 OK\\n```json\\n{\\n  \\\"users\\\": [\\n    {\\n      \\\"id\\\": 1,\\n      \\\"name\\\": \\\"John Doe\\\",\\n      \\\"email\\\": \\\"john@example.com\\\"\\n    }\\n  ]\\n}\\n```\\n\\n### POST /users\\n**Description**: Create a new user\\n**Request Body**:\\n```json\\n{\\n  \\\"name\\\": \\\"Jane Doe\\\",\\n  \\\"email\\\": \\\"jane@example.com\\\"\\n}\\n```\\n**Response**: 201 Created\\n\\n### PUT /users/:id\\n**Description**: Update an existing user\\n**Parameters**: id (integer) - User ID\\n**Request Body**: Same as POST\\n**Response**: 200 OK\\n\\n### DELETE /users/:id\\n**Description**: Delete a user\\n**Parameters**: id (integer) - User ID\\n**Response**: 204 No Content\\n\\n## Error Handling\\n- 401 Unauthorized: Invalid or missing token\\n- 404 Not Found: User ID does not exist\\n- 422 Unprocessable Entity: Invalid request data",
        "tags": ["documentation", "api", "users", "improved"],
        "content_type": "text/markdown",
        "metadata": {
          "document_type": "api_documentation",
          "status": "reviewed",
          "previous_version": "API_DOCS_CONTENT_ID"
        }
      }
    },
    "id": 16
  }'
```

### Step 4: Compare Quality Improvements

Analyze the improved content:

```bash
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_analyze",
      "arguments": {
        "operation": "analyze_quality",
        "project_id": "tutorial-project",
        "content_id": "IMPROVED_DOCS_CONTENT_ID",
        "quality_dimensions": [
          "clarity",
          "completeness",
          "structure",
          "actionability"
        ],
        "options": {
          "include_suggestions": true,
          "include_metrics": true,
          "benchmark_against": ["API_DOCS_CONTENT_ID"]
        }
      }
    },
    "id": 17
  }'
```

### Step 5: Create Relationship Between Versions

Link the improved version to the original:

```bash
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_store",
      "arguments": {
        "operation": "create_relationship",
        "project_id": "tutorial-project",
        "session_id": "tutorial-session",
        "source_id": "IMPROVED_DOCS_CONTENT_ID",
        "target_id": "API_DOCS_CONTENT_ID",
        "type": "improves",
        "strength": 1.0,
        "context": "Improved version with better structure, examples, and completeness"
      }
    },
    "id": 18
  }'
```

**Key Insights**:
- Quality analysis identifies specific improvement areas
- AI suggestions provide actionable recommendations
- Comparative analysis shows improvement metrics
- Version relationships track content evolution

---

## Tutorial 5: Advanced Cross-Domain Operations

**Duration**: 30 minutes  
**Prerequisites**: All previous tutorials completed  
**What you'll learn**: Coordinate operations across Memory, Task, and System domains

### Step 1: Create Task Templates

First, let's create a template for documentation improvement tasks:

```bash
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_store",
      "arguments": {
        "operation": "create_task_template",
        "project_id": "tutorial-project",
        "session_id": "tutorial-session",
        "name": "Documentation Improvement",
        "description": "Template for improving documentation quality",
        "category": "quality",
        "template": {
          "title": "Improve documentation: {{content_title}}",
          "description": "Based on quality analysis, improve the documentation for {{content_id}}",
          "priority": "medium",
          "estimated_mins": 60,
          "tags": ["documentation", "quality", "improvement"]
        },
        "variables": [
          {
            "name": "content_title",
            "type": "string",
            "required": true,
            "description": "Title of the content to improve"
          },
          {
            "name": "content_id",
            "type": "string", 
            "required": true,
            "description": "ID of the content to improve"
          }
        ]
      }
    },
    "id": 19
  }'
```

### Step 2: Generate Tasks from Content Analysis

Use the coordinator to generate improvement tasks based on quality analysis:

```bash
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_system",
      "arguments": {
        "operation": "coordinate_cross_domain",
        "project_id": "tutorial-project",
        "session_id": "tutorial-session",
        "coordination_type": "generate_tasks_from_content",
        "content_id": "API_DOCS_CONTENT_ID",
        "options": {
          "task_type": "quality_improvement",
          "include_quality_analysis": true,
          "auto_assign": false
        }
      }
    },
    "id": 20
  }'
```

### Step 3: Link Tasks to Content

Create explicit links between tasks and the content they reference:

```bash
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_system",
      "arguments": {
        "operation": "coordinate_cross_domain", 
        "project_id": "tutorial-project",
        "session_id": "tutorial-session",
        "coordination_type": "link_task_to_content",
        "task_id": "GENERATED_TASK_ID",
        "content_id": "API_DOCS_CONTENT_ID",
        "link_type": "improves"
      }
    },
    "id": 21
  }'
```

### Step 4: Complete Task and Generate Documentation

When the task is completed, generate final documentation:

```bash
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_system",
      "arguments": {
        "operation": "coordinate_cross_domain",
        "project_id": "tutorial-project", 
        "session_id": "tutorial-session",
        "coordination_type": "create_content_from_task",
        "task_id": "GENERATED_TASK_ID",
        "content_type": "solution",
        "options": {
          "include_task_context": true,
          "link_to_original": true
        }
      }
    },
    "id": 22
  }'
```

### Step 5: Analyze Cross-Domain Patterns

Get insights across all domains:

```bash
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_analyze",
      "arguments": {
        "operation": "analyze_cross_domain_patterns",
        "project_id": "tutorial-project",
        "pattern_types": [
          "task_content_relationships",
          "quality_improvement_cycles",
          "decision_implementation_patterns"
        ],
        "options": {
          "time_range": {
            "start": "2024-12-01",
            "end": "2024-12-31"
          },
          "include_recommendations": true
        }
      }
    },
    "id": 23
  }'
```

### Step 6: Export Complete Project

Export everything for backup or analysis:

```bash
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_system",
      "arguments": {
        "operation": "export_project_data",
        "project_id": "tutorial-project",
        "format": "json",
        "include": {
          "content": true,
          "relationships": true,
          "metadata": true,
          "history": true
        },
        "options": {
          "compress": true,
          "include_embeddings": false,
          "anonymize": false
        }
      }
    },
    "id": 24
  }'
```

**Key Insights**:
- Cross-domain operations maintain clean separation while enabling coordination
- Tasks can be automatically generated from content analysis
- The system tracks relationships between tasks and content
- Pattern analysis reveals workflow effectiveness
- Complete project export enables backup and external analysis

---

## Next Steps

Congratulations! You've completed all five tutorials and learned:

1. **Basic Memory Operations**: Store, search, and retrieve content
2. **Knowledge Relationships**: Build and explore content connections
3. **Decision Documentation**: Structure decision-making with context
4. **Quality Analysis**: Improve content with AI-powered suggestions
5. **Cross-Domain Coordination**: Orchestrate operations across domains

### Advanced Topics to Explore

- **Performance Optimization**: Tuning embeddings and search parameters
- **Custom Metadata Schemas**: Structuring domain-specific metadata
- **Workflow Automation**: Building automated quality improvement cycles
- **Integration Patterns**: Connecting with external tools and systems
- **Multi-Project Management**: Organizing multiple project workspaces

### Best Practices Summary

1. **Use descriptive tags** for better content organization
2. **Include rich metadata** to enhance searchability
3. **Create explicit relationships** for important connections
4. **Document decisions with full context** and rationale
5. **Regular quality analysis** to maintain content standards
6. **Cross-domain coordination** for complex workflows
7. **Regular exports** for backup and analysis

### Community and Support

- **Documentation**: [Complete API Reference](./api-reference.md)
- **GitHub**: [Source code and issues](https://github.com/lerian/mcp-memory-server)
- **Discussions**: [Community support](https://github.com/lerian/mcp-memory-server/discussions)
- **Examples**: [Real-world use cases](./examples/)

Happy building with MCP Memory Server v2! 🧠✨