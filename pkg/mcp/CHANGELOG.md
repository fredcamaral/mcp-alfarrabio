# Changelog

All notable changes to MCP-Go will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial public release of MCP-Go library
- Complete MCP protocol implementation
- Core server functionality with tool, resource, and prompt support
- STDIO transport implementation
- HTTP/HTTPS transport with WebSocket support
- Server-Sent Events (SSE) transport
- Comprehensive middleware system
  - Authentication middleware (JWT and API key)
  - Rate limiting middleware
  - Logging middleware
  - Recovery middleware
  - Metrics middleware
- Plugin architecture for extensibility
- Performance optimizations
  - Memory pooling
  - Optimized JSON encoding/decoding
  - Concurrent request handling
- Monitoring and observability
  - Prometheus metrics integration
  - OpenTelemetry tracing support
  - Structured logging
  - Health check endpoints
- Comprehensive test suite (>90% coverage)
- Documentation and examples
  - API documentation
  - Tutorial guide
  - Advanced usage guide
  - Example applications

### Security
- Input validation for all user inputs
- JWT token validation
- API key authentication
- Rate limiting to prevent abuse
- TLS support for HTTP transport

## [0.2.0] - 2025-01-24

### Added
- API versioning and stability guarantees
- Breaking changes policy
- Performance benchmarks
- Memory pool implementations
- Optimized JSON parser
- Concurrent request handling

### Changed
- Improved error handling throughout the library
- Enhanced documentation with more examples

### Fixed
- JSONRPCError now implements the error interface
- Fixed race conditions in concurrent scenarios

## [0.1.0] - 2025-01-01

### Added
- Initial internal release
- Basic MCP protocol support
- STDIO transport
- Server implementation
- Tool, resource, and prompt handlers

---

## Version Guidelines

### Major Changes (x.0.0)
- Breaking API changes
- Major architectural changes
- Removal of deprecated features

### Minor Changes (0.x.0)
- New features
- Non-breaking API additions
- Performance improvements

### Patch Changes (0.0.x)
- Bug fixes
- Documentation updates
- Security patches

[Unreleased]: https://github.com/yourusername/mcp-go/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/yourusername/mcp-go/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/yourusername/mcp-go/releases/tag/v0.1.0