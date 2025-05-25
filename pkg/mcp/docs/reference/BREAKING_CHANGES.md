# Breaking Changes Policy

## Overview

This document outlines our commitment to API stability and how we handle breaking changes in the MCP-Go library.

## Version Guarantees

We follow [Semantic Versioning 2.0.0](https://semver.org/):

- **Major version (X.0.0)**: May contain breaking changes
- **Minor version (0.X.0)**: New features, no breaking changes to stable APIs
- **Patch version (0.0.X)**: Bug fixes only, no API changes

## API Stability Levels

### Stable
- No breaking changes within a major version
- Deprecation notices given at least one minor version before removal
- Used for core functionality that is battle-tested

### Beta
- May have breaking changes in minor versions
- Changes will be documented in release notes
- Used for newer features that may need refinement

### Experimental
- May change or be removed at any time
- Not recommended for production use
- Used for testing new concepts

### Deprecated
- Scheduled for removal in a future version
- Replacement API will be provided
- Minimum deprecation period: 6 months or 2 minor versions

## Breaking Change Definition

A breaking change is any modification that requires users to update their code, including:

1. Removing exported types, functions, methods, or fields
2. Changing function signatures (parameters or return types)
3. Changing behavior of existing functionality
4. Renaming exported identifiers
5. Changing struct fields from exported to unexported

## Migration Support

For all breaking changes, we will:

1. Provide detailed migration guides
2. Offer automated migration tools where possible
3. Maintain the old API as deprecated for at least one minor version
4. Include clear deprecation warnings in documentation and code

## Version Compatibility

- **Forward Compatibility**: Newer library versions will work with older protocol versions
- **Backward Compatibility**: Older library versions may not work with newer protocol versions
- **API Version Check**: Use `IsCompatibleAPI()` to verify compatibility

## Deprecation Process

1. Mark API as deprecated with clear documentation
2. Log warnings when deprecated APIs are used (can be disabled)
3. Provide migration path in deprecation notice
4. Remove in next major version (minimum 6 months after deprecation)

## Examples

### Adding Deprecation

```go
// Deprecated: Use NewServer instead. Will be removed in v1.0.0.
func CreateServer(name string) *Server {
    // Log deprecation warning
    logDeprecation("CreateServer", "NewServer", "v1.0.0")
    return NewServer(name, "1.0.0")
}
```

### Checking API Compatibility

```go
import "github.com/yourusername/mcp-go"

func init() {
    if !mcp.IsCompatibleAPI(2) {
        panic("This application requires MCP-Go API v2")
    }
}
```

## Change Log

All breaking changes will be documented in:
- CHANGELOG.md with migration instructions
- GitHub release notes
- API documentation

## Getting Help

If you need help with migration:
1. Check the migration guide for your version
2. Open a GitHub issue with the "migration-help" label
3. Join our community Discord for real-time support