# MT-006 Sub-Tasks: Server-Side AI Integration and HTTP API Foundation

**Based on**: MT-006 main task specification  
**Duration**: 3-4 weeks (6 sub-tasks × 2-4 hours each)  
**Phase**: Server Foundation

## ST-006-01: HTTP API Layer Foundation with Chi Router

### Overview
Create the foundational HTTP API layer for the MCP Memory Server using Chi router, establishing the base infrastructure for CLI communication.

### Scope
- Implement Chi router setup with middleware stack
- Create basic HTTP server configuration and startup
- Setup API versioning (v1) and route structure
- Implement health check and status endpoints
- Add basic request/response logging
- Configure CORS for development and production

### Technical Requirements
- Chi router v5.0+ with middleware support
- HTTP server with graceful shutdown
- API versioning structure `/api/v1/*`
- Health check endpoint at `/api/v1/health`
- Request ID generation and tracing
- Error handling middleware with structured responses

### Acceptance Criteria
- [x] HTTP server starts successfully on configurable port (default 9080)
- [x] Chi router handles requests with proper middleware chain
- [x] Health check endpoint returns server status and Qdrant connectivity
- [x] API versioning works with consistent URL structure
- [x] Request logging captures all essential information
- [x] CORS configuration allows CLI client connections
- [x] Graceful shutdown works without data loss
- [x] Error responses follow consistent JSON format

**Status**: ✅ **COMPLETED** - Implemented Chi router foundation with middleware stack, health checks, and proper API structure

### Testing Requirements
- Unit tests for middleware functions
- Integration tests for health endpoint
- Router configuration validation
- Error handling scenarios

### Files to Create/Modify
- `internal/api/router.go` - Chi router setup and configuration
- `internal/api/server.go` - HTTP server with graceful shutdown
- `internal/api/middleware/cors.go` - CORS configuration
- `internal/api/middleware/logging.go` - Request logging
- `internal/api/handlers/health.go` - Health check handler
- `cmd/server/main.go` - Add HTTP server startup

### Dependencies
- Chi router library installation
- Existing MCP server structure

### Estimated Time
4 hours

---

## ST-006-02: Multi-Model AI Service Layer Implementation

### Overview
Implement the server-side AI service layer with support for multiple AI models (Claude Sonnet 4, Perplexity Sonar Pro, OpenAI GPT-4o) including fallback and routing logic.

### Scope
- Create AI service abstraction layer
- Implement Claude Sonnet 4 API client
- Implement Perplexity Sonar Pro API client
- Implement OpenAI GPT-4o API client
- Create model router with fallback logic
- Add rate limiting per model
- Implement response caching system

### Technical Requirements
- Unified AI service interface for all models
- HTTP clients for each AI provider
- Circuit breaker pattern for reliability
- Model routing with priority and fallback
- Request/response caching (Redis or in-memory)
- Rate limiting per model with different thresholds
- Error handling with specific model error types

### Acceptance Criteria
- [x] All three AI models work independently
- [x] Model router selects appropriate model based on availability
- [x] Fallback works when primary model fails or rate limits
- [x] Response caching reduces redundant API calls by >70%
- [x] Rate limiting prevents exceeding provider limits
- [x] Circuit breaker prevents cascading failures
- [x] AI service handles timeouts and retries properly
- [x] Error messages provide actionable information

**Status**: ✅ **COMPLETED** - Implemented multi-model AI service layer with Claude Sonnet 4, Perplexity Sonar Pro, and OpenAI GPT-4o with fallback and caching

### Testing Requirements
- Unit tests for each AI client
- Integration tests with mock AI APIs
- Fallback scenario testing
- Cache effectiveness validation
- Rate limiting boundary testing

### Files to Create/Modify
- `internal/ai/service.go` - AI service interface and router
- `internal/ai/claude_client.go` - Claude Sonnet 4 client
- `internal/ai/perplexity_client.go` - Perplexity Sonar Pro client
- `internal/ai/openai_client.go` - OpenAI GPT-4o client
- `internal/ai/router.go` - Model routing and fallback
- `internal/ai/cache.go` - Response caching system
- `internal/ai/circuit_breaker.go` - Circuit breaker implementation

### Dependencies
- AI provider API credentials in server environment
- HTTP client libraries
- Circuit breaker library

### Estimated Time
4 hours

---

## ST-006-03: Task Management HTTP Endpoints

### Overview
Implement HTTP API endpoints for task CRUD operations, providing the interface for CLI task management operations.

### Scope
- Create task list endpoint with filtering and pagination
- Implement task creation endpoint with validation
- Add task update endpoint (PATCH) with partial updates
- Create task deletion endpoint (soft delete)
- Implement task search with repository filtering
- Add bulk operations for multiple tasks
- Setup proper error handling and validation

### Technical Requirements
- RESTful endpoint design following OpenAPI patterns
- Request validation with structured error responses
- Pagination with cursor-based navigation
- Repository-based filtering and authorization
- Proper HTTP status codes and headers
- JSON request/response formatting
- Input sanitization and validation

### Acceptance Criteria
- [x] GET /api/v1/tasks returns paginated task list with filtering
- [x] POST /api/v1/tasks creates tasks with proper validation
- [x] PATCH /api/v1/tasks/{id} updates tasks with conflict detection
- [x] DELETE /api/v1/tasks/{id} performs soft delete
- [x] Repository filtering works correctly
- [x] Pagination handles large datasets efficiently
- [x] Validation errors provide clear field-specific messages
- [x] All endpoints return consistent JSON format

**Status**: ✅ **COMPLETED** - Implemented comprehensive task management HTTP endpoints with CRUD operations, search, and batch processing

### Testing Requirements
- Unit tests for each endpoint handler
- Integration tests with real database operations
- Validation boundary testing
- Error scenario testing
- Pagination edge case testing

### Files to Create/Modify
- `internal/api/handlers/task_handler.go` - Task CRUD handlers
- `internal/api/middleware/validation.go` - Request validation
- `pkg/types/api_types.go` - API request/response types
- `internal/storage/task_repository.go` - Enhanced task repository
- `internal/api/handlers/task_handler_test.go` - Handler tests

### Dependencies
- Enhanced task storage layer
- Validation library
- Existing Qdrant integration

### Estimated Time
4 hours

---

## ST-006-04: PRD Import and AI-Powered Task Generation

### Overview
Implement PRD import endpoint that processes document content and generates tasks using AI analysis and parsing.

### Scope
- Create PRD import endpoint (POST /api/v1/prd/import)
- Implement PRD content parsing and analysis
- Add AI-powered task generation from PRD content
- Create complexity analysis for generated tasks
- Implement task relationship detection
- Add batch task creation from PRD analysis
- Setup progress tracking for long-running operations

### Technical Requirements
- PRD document parsing (Markdown, text formats)
- AI integration for content analysis
- Task generation algorithms with complexity scoring
- Relationship detection between generated tasks
- Progress tracking with status updates
- Atomic operations for batch task creation
- Error handling for parsing failures

### Acceptance Criteria
- [x] PRD import endpoint accepts various document formats
- [x] AI analysis generates relevant tasks from PRD content
- [x] Complexity scoring accurately reflects task difficulty
- [x] Task relationships are properly detected and stored
- [x] Batch task creation is atomic (all or nothing)
- [x] Progress tracking works for long-running imports
- [x] Error handling provides specific failure reasons
- [x] Generated tasks have proper metadata linking to PRD

**Status**: ✅ **COMPLETED** - Implemented comprehensive PRD import and AI-powered task generation system

### Testing Requirements
- Unit tests for PRD parsing logic
- Integration tests with AI services
- Test with various PRD document formats
- Complexity scoring accuracy validation
- Error handling for malformed documents

### Files to Create/Modify
- `internal/api/handlers/prd_handler.go` - PRD import handler
- `internal/prd/parser.go` - PRD document parsing
- `internal/prd/analyzer.go` - AI-powered PRD analysis
- `internal/prd/task_generator.go` - Task generation from PRD
- `internal/prd/complexity.go` - Complexity analysis
- `pkg/types/prd_types.go` - PRD-related types

### Dependencies
- AI service layer (ST-006-02)
- Task management endpoints (ST-006-03)
- Document parsing libraries

### Estimated Time
4 hours

---

## ST-006-05: Rate Limiting and Version Compatibility

### Overview
Implement comprehensive rate limiting system and version compatibility checking to ensure server stability and CLI compatibility.

### Scope
- Create rate limiting middleware with configurable rules
- Implement per-endpoint rate limiting as specified in TRD
- Add version compatibility checking middleware
- Create CLI version validation system
- Implement rate limit headers and responses
- Add monitoring for rate limiting effectiveness
- Setup version compatibility matrix

### Technical Requirements
- Rate limiting using token bucket or sliding window algorithm
- Per-endpoint rate limit configuration
- Version compatibility matrix with semantic versioning
- HTTP headers for rate limit status
- Monitoring metrics for rate limiting
- Database or Redis backend for distributed rate limiting
- Clear error messages for rate limit violations

### Acceptance Criteria
- [x] Rate limiting enforces TRD-specified limits per endpoint
- [x] Version checking blocks incompatible CLI versions
- [x] Rate limit headers inform clients of current status
- [x] Rate limiting scales with multiple server instances
- [x] Version compatibility matrix handles semantic versioning
- [x] Monitoring captures rate limiting metrics
- [x] Error messages provide clear upgrade instructions
- [x] Rate limits can be configured without code changes

**Status**: ✅ **COMPLETED** - Implemented comprehensive rate limiting with sliding window algorithm and version compatibility checking

### Testing Requirements
- Unit tests for rate limiting algorithms
- Integration tests for version compatibility
- Load testing to validate rate limits
- Edge case testing for version matching
- Monitoring metric validation

### Files to Create/Modify
- `internal/api/middleware/rate_limit.go` - Rate limiting middleware
- `internal/api/middleware/version_check.go` - Version validation
- `internal/config/rate_limits.go` - Rate limit configuration
- `internal/version/compatibility.go` - Version compatibility logic
- `internal/monitoring/rate_limit_metrics.go` - Rate limiting metrics

### Dependencies
- Rate limiting library (golang.org/x/time/rate)
- Version parsing library
- Monitoring infrastructure

### Estimated Time
3 hours

---

## ST-006-06: OpenAPI Specification and Response Optimization

### Overview
Generate comprehensive OpenAPI 3.0 specification and implement response optimization including caching and compression.

### Scope
- Generate OpenAPI 3.0 specification from code annotations
- Create interactive API documentation (Swagger UI)
- Implement response caching for AI operations
- Add response compression (gzip)
- Create API documentation endpoint
- Setup automatic documentation updates
- Optimize response formats and sizes

### Technical Requirements
- OpenAPI 3.0 compliant specification
- Interactive documentation interface
- Response caching with TTL configuration
- HTTP compression middleware
- Documentation served at /api/v1/docs
- Automatic spec generation from code
- Performance optimized response handling

### Acceptance Criteria
- [x] OpenAPI specification accurately reflects all endpoints
- [x] Interactive documentation is accessible and functional
- [x] Response caching improves API performance significantly
- [x] Compression reduces response sizes appropriately
- [x] Documentation automatically updates with code changes
- [x] API specification validates successfully
- [x] Documentation provides clear usage examples
- [x] Response optimization doesn't break functionality

**Status**: ✅ **COMPLETED** - Implemented comprehensive OpenAPI 3.0 specification with interactive documentation and response optimization

### Testing Requirements
- OpenAPI specification validation
- Documentation accessibility testing
- Cache effectiveness measurement
- Compression ratio validation
- Performance impact testing

### Files to Create/Modify
- `internal/api/docs/openapi.go` - OpenAPI specification generation
- `internal/api/middleware/compression.go` - Response compression
- `internal/api/middleware/caching.go` - Response caching
- `internal/api/handlers/docs_handler.go` - Documentation endpoint
- `api/openapi.yaml` - Generated OpenAPI specification
- `docs/api/` - API documentation assets

### Dependencies
- OpenAPI generation library
- Swagger UI assets
- Compression middleware
- Caching library

### Estimated Time
3 hours

---

## Summary

Total estimated time: 22 hours (3-4 weeks at 6-8 hours per week)

### Sub-task Dependencies
1. ST-006-01 (HTTP API Foundation) → All other sub-tasks
2. ST-006-02 (AI Service Layer) → ST-006-04 (PRD Import)
3. ST-006-03 (Task Endpoints) → ST-006-04 (PRD Import)
4. ST-006-05 (Rate Limiting) can run in parallel with others
5. ST-006-06 (Documentation) depends on all API endpoints

### Integration Points
- ST-006-03 and ST-006-04 integrate with MT-003 (Server Integration)
- ST-006-02 provides foundation for AI features in MT-002
- ST-006-05 security measures support MT-007 requirements
- ST-006-06 documentation supports MT-008 infrastructure

### Testing Strategy
- Unit tests for each component
- Integration tests for API endpoints
- Performance tests for AI operations
- Load tests for rate limiting
- End-to-end tests with CLI integration