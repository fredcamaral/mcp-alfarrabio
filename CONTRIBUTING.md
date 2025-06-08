# Contributing to MCP-Memory

First off, thank you for considering contributing to MCP-Memory! 🎉 It's people like you that make MCP-Memory such a great tool.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [How Can I Contribute?](#how-can-i-contribute)
  - [Reporting Bugs](#reporting-bugs)
  - [Suggesting Enhancements](#suggesting-enhancements)
  - [Your First Code Contribution](#your-first-code-contribution)
  - [Pull Requests](#pull-requests)
- [Development Process](#development-process)
  - [Setting Up Your Environment](#setting-up-your-environment)
  - [Running Tests](#running-tests)
  - [Code Style](#code-style)
  - [Commit Messages](#commit-messages)
- [Community](#community)

## Code of Conduct

This project and everyone participating in it is governed by the [MCP-Memory Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. Please report unacceptable behavior to [security@mcp-memory.io](mailto:security@mcp-memory.io).

## Getting Started

MCP-Memory is a Go-based implementation of the Model Context Protocol (MCP) with advanced memory capabilities. Before you begin:

1. Make sure you have Go 1.21+ installed
2. Fork and clone the repository
3. Install dependencies: `go mod download`
4. Run tests to ensure everything works: `make test`

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check [existing issues](https://github.com/LerianStudio/lerian-mcp-memory/issues) as you might find out that you don't need to create one. When you are creating a bug report, please include as many details as possible:

**Bug Report Template:**
```markdown
**Describe the bug**
A clear and concise description of what the bug is.

**To Reproduce**
Steps to reproduce the behavior:
1. Go to '...'
2. Run command '....'
3. See error

**Expected behavior**
What you expected to happen.

**Environment:**
 - OS: [e.g. macOS 14.0]
 - Go version: [e.g. 1.21.5]
 - MCP-Memory version: [e.g. 0.1.0]

**Additional context**
Add any other context about the problem here.
```

### Suggesting Enhancements

Enhancement suggestions are tracked as [GitHub issues](https://github.com/LerianStudio/lerian-mcp-memory/issues). Create an issue and provide:

- **Use a clear and descriptive title**
- **Provide a step-by-step description** of the suggested enhancement
- **Provide specific examples** to demonstrate the steps
- **Describe the current behavior** and **explain which behavior you expected to see instead**
- **Explain why this enhancement would be useful**

### Your First Code Contribution

Unsure where to begin? You can start by looking through these issues:

- [Beginner issues](https://github.com/LerianStudio/lerian-mcp-memory/labels/good%20first%20issue) - issues which should only require a few lines of code
- [Help wanted issues](https://github.com/LerianStudio/lerian-mcp-memory/labels/help%20wanted) - issues which should be a bit more involved

### Pull Requests

1. Fork the repo and create your branch from `main`.
2. If you've added code that should be tested, add tests.
3. If you've changed APIs, update the documentation.
4. Ensure the test suite passes.
5. Make sure your code follows the existing style.
6. Issue that pull request!

## Development Process

### Setting Up Your Environment

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/lerian-mcp-memory.git
cd lerian-mcp-memory

# Add upstream remote
git remote add upstream https://github.com/LerianStudio/lerian-mcp-memory.git

# Install dependencies
go mod download

# Install development tools
make dev-tools

# Set up pre-commit hooks
make setup-hooks
```

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific package tests
go test ./internal/storage/...

# Run tests with race detection
go test -race ./...

# Run integration tests
make test-integration
```

### Code Style

We follow the standard Go coding conventions:

- Run `gofmt` on your code
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `golangci-lint` for additional checks: `make lint`
- Write meaningful variable and function names
- Keep functions focused and small
- Document exported functions and types

**Example:**
```go
// StoreChunk stores a conversation chunk in memory with automatic analysis
// and embedding generation. It returns the stored chunk ID or an error.
func (s *Server) StoreChunk(ctx context.Context, content string, sessionID string) (string, error) {
    // Implementation
}
```

### Commit Messages

We follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation only changes
- `style`: Formatting, missing semi-colons, etc.
- `refactor`: A code change that neither fixes a bug nor adds a feature
- `perf`: A code change that improves performance
- `test`: Adding missing tests
- `build`: Changes to the build process
- `ci`: Changes to CI configuration
- `chore`: Other changes that don't modify src or test files

**Examples:**
```
feat(storage): add support for PostgreSQL backend

fix(mcp): handle nil pointer in tool execution

docs(readme): update installation instructions

perf(embeddings): optimize batch processing
```

## Project Structure

```
lerian-mcp-memory/
├── cmd/                    # Command-line applications
│   ├── server/            # MCP server application
│   ├── migrate/           # Database migration tool
│   └── openapi/           # OpenAPI specification generator
├── internal/              # Private application code
│   ├── mcp/              # MCP protocol implementation
│   ├── storage/          # Storage backends
│   ├── embeddings/       # Embedding generation
│   ├── intelligence/     # AI/ML features
│   └── workflow/         # Workflow management
├── pkg/                   # Public libraries
│   └── mcp/              # MCP Go SDK
├── configs/              # Configuration files
├── docs/                 # Documentation
└── tests/                # Integration tests
```

## Testing Guidelines

1. **Unit Tests**: Write tests for all new functions
2. **Table-Driven Tests**: Use table-driven tests for multiple scenarios
3. **Mocking**: Use interfaces for dependencies to enable mocking
4. **Coverage**: Aim for >80% code coverage
5. **Integration Tests**: Add integration tests for cross-component features

**Example Test:**
```go
func TestStoreChunk(t *testing.T) {
    tests := []struct {
        name      string
        content   string
        sessionID string
        wantErr   bool
    }{
        {
            name:      "valid chunk",
            content:   "test content",
            sessionID: "session123",
            wantErr:   false,
        },
        {
            name:      "empty content",
            content:   "",
            sessionID: "session123",
            wantErr:   true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## Documentation

- Document all exported functions, types, and constants
- Add examples for complex features
- Update README.md for user-facing changes
- Maintain architectural decision records (ADRs) in `docs/adr/`

## Community

- **Discord**: Join our [Discord server](https://discord.gg/mcp-memory) for discussions
- **GitHub Discussions**: For longer-form discussions about features and ideas
- **Stack Overflow**: Tag questions with `mcp-memory`

## Recognition

Contributors will be recognized in:
- The CONTRIBUTORS file
- Release notes
- Special recognition for significant contributions

Thank you for contributing to MCP-Memory! 🚀