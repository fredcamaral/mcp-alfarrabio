# MCP Memory Server v2 - Error Handling Guide

## Overview

The MCP Memory Server v2 provides comprehensive, user-friendly error handling with clear messages, recovery suggestions, and detailed troubleshooting guidance. This guide covers all error scenarios and how to handle them effectively.

## Error Response Format

All errors follow a consistent JSON-RPC 2.0 error response format:

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
      "documentation_url": "https://docs.lerian.dev/mcp-memory/parameters",
      "error_id": "err_20241206_153045_abc123"
    }
  },
  "id": 1
}
```

## Error Categories

### Client Errors (4xx)

These errors indicate issues with the request that need to be fixed by the client.

#### 400 - Bad Request

**When it occurs**: Malformed JSON-RPC request or invalid request structure

**Example**:
```json
{
  "jsonrpc": "2.0", 
  "error": {
    "code": 400,
    "message": "Invalid JSON-RPC request",
    "data": {
      "error_type": "malformed_request",
      "details": {
        "issue": "Missing required field 'method'",
        "received_fields": ["jsonrpc", "params", "id"]
      },
      "suggestions": [
        "Include the 'method' field in your request",
        "Ensure your request follows JSON-RPC 2.0 format",
        "Check the API documentation for correct request structure"
      ],
      "documentation_url": "https://docs.lerian.dev/mcp-memory/json-rpc"
    }
  },
  "id": null
}
```

**How to fix**:
1. Validate your JSON syntax
2. Ensure all required JSON-RPC fields are present
3. Check parameter formatting

#### 401 - Unauthorized

**When it occurs**: Missing or invalid authentication credentials

**Example**:
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": 401, 
    "message": "Authentication required",
    "data": {
      "error_type": "authentication_error",
      "details": {
        "issue": "Missing Authorization header",
        "auth_methods": ["bearer_token", "session_id"]
      },
      "suggestions": [
        "Include a valid Bearer token in the Authorization header",
        "Provide a valid session_id parameter",
        "Check if your API key has expired"
      ],
      "documentation_url": "https://docs.lerian.dev/mcp-memory/authentication"
    }
  },
  "id": 1
}
```

**How to fix**:
1. Add valid authentication credentials
2. Check if your API key is active
3. Verify session_id is valid

#### 403 - Forbidden

**When it occurs**: Insufficient permissions for the requested operation

**Example**:
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": 403,
    "message": "Insufficient permissions",
    "data": {
      "error_type": "permission_error", 
      "details": {
        "operation": "delete_content",
        "resource": "content_abc123",
        "required_permission": "content:delete",
        "user_permissions": ["content:read", "content:write"]
      },
      "suggestions": [
        "Request delete permissions from your administrator",
        "Use an account with sufficient permissions",
        "Try a read or update operation instead"
      ],
      "documentation_url": "https://docs.lerian.dev/mcp-memory/permissions"
    }
  },
  "id": 1
}
```

**How to fix**:
1. Request additional permissions
2. Use an account with appropriate access
3. Contact your administrator

#### 404 - Not Found

**When it occurs**: Requested resource does not exist

**Example**:
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
        "project_id": "my-project",
        "search_scope": "project"
      },
      "suggestions": [
        "Verify the content ID is correct: content_xyz789",
        "Check if the content was deleted recently",
        "Ensure you're searching in the right project: my-project",
        "Use the search_content operation to find available content"
      ],
      "documentation_url": "https://docs.lerian.dev/mcp-memory/resources"
    }
  },
  "id": 1
}
```

**How to fix**:
1. Verify the resource ID
2. Check if the resource was deleted
3. Ensure you're in the correct project
4. Use search to find available resources

#### 409 - Conflict

**When it occurs**: Resource conflict, such as duplicate IDs or concurrent modifications

**Example**:
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": 409,
    "message": "Resource conflict", 
    "data": {
      "error_type": "resource_conflict",
      "details": {
        "conflict_type": "version_mismatch",
        "resource_id": "content_abc123",
        "expected_version": 3,
        "current_version": 5,
        "last_modified": "2024-12-06T15:25:30Z"
      },
      "suggestions": [
        "Refresh the content to get the latest version",
        "Use the current version number (5) in your update",
        "Review conflicting changes before proceeding",
        "Consider using force update if appropriate"
      ],
      "documentation_url": "https://docs.lerian.dev/mcp-memory/versioning"
    }
  },
  "id": 1
}
```

**How to fix**:
1. Refresh the resource to get current state
2. Use the correct version number
3. Resolve conflicts manually if needed
4. Consider force updates carefully

#### 422 - Validation Error

**When it occurs**: Request parameters fail validation rules

**Example**:
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": 422,
    "message": "Parameter validation failed",
    "data": {
      "error_type": "validation_error",
      "details": {
        "field": "content",
        "violation": "too_long",
        "provided_length": 15000,
        "max_length": 10000,
        "location": "arguments.content"
      },
      "suggestions": [
        "Reduce content length to 10,000 characters or less",
        "Split large content into multiple smaller pieces",
        "Use the update_content operation to append to existing content",
        "Consider storing large content as external files with references"
      ],
      "documentation_url": "https://docs.lerian.dev/mcp-memory/limits"
    }
  },
  "id": 1
}
```

**Common validation errors**:

**Missing required field**:
```json
{
  "field": "project_id",
  "violation": "required",
  "provided": null
}
```

**Invalid format**:
```json
{
  "field": "session_id", 
  "violation": "invalid_format",
  "provided": "invalid-session-123!",
  "expected_format": "alphanumeric with hyphens and underscores"
}
```

**Value out of range**:
```json
{
  "field": "limit",
  "violation": "out_of_range", 
  "provided": 500,
  "min_value": 1,
  "max_value": 100
}
```

### Server Errors (5xx)

These errors indicate issues on the server side that are not caused by the client request.

#### 500 - Internal Server Error

**When it occurs**: Unexpected server errors

**Example**:
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": 500,
    "message": "Internal server error",
    "data": {
      "error_type": "internal_error",
      "details": {
        "error_id": "err_20241206_153045_xyz789",
        "component": "vector_store",
        "operation": "similarity_search"
      },
      "suggestions": [
        "Try the request again in a few moments",
        "If the error persists, contact support with error ID: err_20241206_153045_xyz789",
        "Check system status at https://status.lerian.dev"
      ],
      "documentation_url": "https://docs.lerian.dev/mcp-memory/troubleshooting"
    }
  },
  "id": 1
}
```

**How to handle**:
1. Retry the request after a brief delay
2. Check system status
3. Contact support with the error ID

#### 502 - Service Unavailable

**When it occurs**: External service dependencies are unavailable

**Example**:
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": 502,
    "message": "External service unavailable",
    "data": {
      "error_type": "service_unavailable",
      "details": {
        "service": "openai_api",
        "operation": "text_embedding",
        "last_successful": "2024-12-06T15:20:00Z",
        "retry_after": 30
      },
      "suggestions": [
        "Wait 30 seconds before retrying",
        "Operations not requiring embeddings will still work",
        "Check OpenAI API status at https://status.openai.com",
        "Consider using cached embeddings if available"
      ],
      "documentation_url": "https://docs.lerian.dev/mcp-memory/dependencies"
    }
  },
  "id": 1
}
```

**How to handle**:
1. Wait for the specified retry time
2. Use operations that don't require the unavailable service
3. Check external service status

#### 503 - Rate Limited

**When it occurs**: Too many requests in a short time period

**Example**:
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": 503,
    "message": "Rate limit exceeded",
    "data": {
      "error_type": "rate_limit_exceeded",
      "details": {
        "limit_type": "per_session",
        "limit": 100,
        "window": "60s",
        "current_usage": 105,
        "reset_time": "2024-12-06T15:31:00Z"
      },
      "suggestions": [
        "Wait until 15:31:00 UTC before making more requests",
        "Reduce request frequency to stay under 100 requests per minute",
        "Consider batching multiple operations into single requests",
        "Use caching to reduce redundant requests"
      ],
      "documentation_url": "https://docs.lerian.dev/mcp-memory/rate-limits"
    }
  },
  "id": 1
}
```

**How to handle**:
1. Wait until the reset time
2. Implement exponential backoff
3. Batch requests when possible
4. Cache responses to reduce calls

#### 504 - Timeout

**When it occurs**: Operation takes longer than the configured timeout

**Example**:
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": 504,
    "message": "Operation timeout",
    "data": {
      "error_type": "operation_timeout",
      "details": {
        "operation": "analyze_quality",
        "timeout": "30s",
        "elapsed_time": "31.5s",
        "content_size": "25KB",
        "complexity": "high"
      },
      "suggestions": [
        "Try breaking the content into smaller pieces",
        "Reduce the analysis complexity if possible",
        "Retry with a simpler operation first",
        "Contact support if timeouts persist"
      ],
      "documentation_url": "https://docs.lerian.dev/mcp-memory/timeouts"
    }
  },
  "id": 1
}
```

**How to handle**:
1. Reduce operation complexity
2. Break large operations into smaller chunks
3. Retry with simplified parameters
4. Check for system performance issues

## Error Recovery Strategies

### Automatic Retry Logic

Implement intelligent retry logic based on error type:

```typescript
async function retryOperation(operation: () => Promise<any>, maxRetries = 3) {
  for (let attempt = 1; attempt <= maxRetries; attempt++) {
    try {
      return await operation();
    } catch (error) {
      const shouldRetry = isRetryableError(error);
      const delay = calculateDelay(attempt, error);
      
      if (!shouldRetry || attempt === maxRetries) {
        throw error;
      }
      
      await new Promise(resolve => setTimeout(resolve, delay));
    }
  }
}

function isRetryableError(error: any): boolean {
  const retryableCodes = [500, 502, 503, 504];
  return retryableCodes.includes(error.code);
}

function calculateDelay(attempt: number, error: any): number {
  // Rate limit: use provided retry_after
  if (error.code === 503 && error.data?.details?.retry_after) {
    return error.data.details.retry_after * 1000;
  }
  
  // Exponential backoff for other errors
  return Math.min(1000 * Math.pow(2, attempt - 1), 30000);
}
```

### Graceful Degradation

Handle service unavailability gracefully:

```typescript
async function searchContent(query: string, options: SearchOptions = {}) {
  try {
    // Try semantic search first
    return await memoryRetrieve({
      operation: 'search_content',
      query,
      options: { ...options, query_type: 'semantic' }
    });
  } catch (error) {
    if (error.code === 502 && error.data?.details?.service === 'openai_api') {
      // Fall back to keyword search if embeddings unavailable
      console.warn('Semantic search unavailable, falling back to keyword search');
      return await memoryRetrieve({
        operation: 'search_content',
        query,
        options: { ...options, query_type: 'keyword' }
      });
    }
    throw error;
  }
}
```

### User-Friendly Error Messages

Transform technical errors into user-friendly messages:

```typescript
function getUserFriendlyMessage(error: any): string {
  switch (error.code) {
    case 404:
      return "The content you're looking for doesn't exist. It may have been deleted or you might have the wrong ID.";
    
    case 422:
      if (error.data?.details?.field === 'content' && error.data?.details?.violation === 'too_long') {
        return "Your content is too long. Please split it into smaller pieces or trim it down.";
      }
      return "There's an issue with your request. Please check your input and try again.";
    
    case 503:
      return "The server is currently busy. Please wait a moment and try again.";
    
    case 500:
      return "Something went wrong on our end. Please try again, and contact support if the problem continues.";
    
    default:
      return error.message || "An unexpected error occurred. Please try again.";
  }
}
```

## Debugging Tools

### Error ID Tracking

Every error includes a unique error ID for tracking:

```bash
# Search logs using error ID
grep "err_20241206_153045_abc123" /var/log/mcp-memory-server.log

# Get detailed error context
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_system",
      "arguments": {
        "operation": "get_error_details",
        "error_id": "err_20241206_153045_abc123"
      }
    },
    "id": 1
  }'
```

### Health Check Diagnostics

Use comprehensive health checks to identify issues:

```bash
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_system",
      "arguments": {
        "operation": "check_system_health",
        "detailed": true,
        "components": ["database", "vector_store", "ai_service", "cache"]
      }
    },
    "id": 1
  }'
```

### Request Validation

Validate requests before sending:

```typescript
function validateRequest(request: any): string[] {
  const errors: string[] = [];
  
  // Check required fields
  if (!request.params?.arguments?.project_id) {
    errors.push("project_id is required");
  }
  
  // Check field formats
  if (request.params?.arguments?.project_id && 
      !/^[a-zA-Z0-9_-]+$/.test(request.params.arguments.project_id)) {
    errors.push("project_id must contain only letters, numbers, hyphens, and underscores");
  }
  
  // Check value ranges
  if (request.params?.arguments?.limit && 
      (request.params.arguments.limit < 1 || request.params.arguments.limit > 100)) {
    errors.push("limit must be between 1 and 100");
  }
  
  return errors;
}
```

## Common Troubleshooting Scenarios

### Scenario 1: Content Not Found After Storage

**Problem**: Content was just stored but search/get operations return 404

**Likely causes**:
1. Eventual consistency delay
2. Wrong project_id in retrieval
3. Content stored in different session

**Solution**:
```bash
# 1. Check if content exists with exact ID
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_retrieve",
      "arguments": {
        "operation": "get_content",
        "project_id": "EXACT_PROJECT_ID_FROM_STORAGE",
        "content_id": "CONTENT_ID_FROM_STORAGE_RESPONSE"
      }
    },
    "id": 1
  }'

# 2. Search for recently stored content
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_retrieve",
      "arguments": {
        "operation": "search_content",
        "project_id": "YOUR_PROJECT_ID",
        "query": "CONTENT_KEYWORDS",
        "options": {
          "sort_by": "date",
          "sort_order": "desc"
        }
      }
    },
    "id": 2
  }'
```

### Scenario 2: Slow Search Performance

**Problem**: Search operations are taking too long

**Diagnostics**:
```bash
# Check system performance
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_system",
      "arguments": {
        "operation": "check_system_health",
        "detailed": true
      }
    },
    "id": 1
  }'
```

**Solutions**:
1. Add more specific filters
2. Reduce search limit
3. Use keyword search instead of semantic
4. Check vector store performance

### Scenario 3: Embedding Generation Failures

**Problem**: Content storage fails with embedding errors

**Diagnosis**:
```bash
# Check AI service status
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "memory_system",
      "arguments": {
        "operation": "check_system_health",
        "components": ["ai_service"]
      }
    },
    "id": 1
  }'
```

**Solutions**:
1. Check OpenAI API key
2. Verify API quotas
3. Store without embeddings temporarily
4. Use cached embeddings if available

## Best Practices

### Error Handling in Applications

1. **Always check error codes** before parsing responses
2. **Log errors with context** for debugging
3. **Implement retry logic** for transient errors
4. **Provide user feedback** for all error conditions
5. **Monitor error rates** and patterns

### Prevention Strategies

1. **Validate inputs** before sending requests
2. **Use connection pooling** to avoid connection errors
3. **Implement caching** to reduce API calls
4. **Monitor quotas** and usage patterns
5. **Test error scenarios** during development

### Monitoring and Alerting

1. **Track error rates** by type and endpoint
2. **Set up alerts** for critical error patterns
3. **Monitor external dependencies** (OpenAI, Qdrant)
4. **Log performance metrics** for optimization
5. **Review error logs** regularly for improvement opportunities

This comprehensive error handling guide ensures you can effectively handle, recover from, and prevent errors when using the MCP Memory Server v2.