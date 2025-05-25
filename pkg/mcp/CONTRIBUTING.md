# Contributing to MCP-Go

We love your input! We want to make contributing to MCP-Go as easy and transparent as possible, whether it's:

- Reporting a bug
- Discussing the current state of the code
- Submitting a fix
- Proposing new features
- Becoming a maintainer

## Development Process

We use GitHub to host code, to track issues and feature requests, as well as accept pull requests.

1. Fork the repo and create your branch from `main`.
2. If you've added code that should be tested, add tests.
3. If you've changed APIs, update the documentation.
4. Ensure the test suite passes.
5. Make sure your code lints.
6. Issue that pull request!

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Docker (for integration tests)
- Make

### Setting Up Your Development Environment

```bash
# Clone your fork
git clone https://github.com/yourusername/mcp-go.git
cd mcp-go

# Add upstream remote
git remote add upstream https://github.com/originalowner/mcp-go.git

# Install dependencies
go mod download

# Run tests
make test

# Run linters
make lint
```

## Code Style

We follow standard Go conventions:

- Run `gofmt -s` on your code
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use meaningful variable names
- Add comments for exported functions
- Keep functions focused and small

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run with race detector
make test-race

# Run integration tests
make test-integration

# Run benchmarks
make benchmark

# Generate coverage report
make test-coverage
```

### Writing Tests

- Write unit tests for all new functionality
- Include both positive and negative test cases
- Use table-driven tests where appropriate
- Mock external dependencies
- Aim for >80% code coverage

Example test structure:

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:  "valid input",
            input: "test",
            want:  "TEST",
        },
        {
            name:    "empty input",
            input:   "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := MyFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("MyFunction() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("MyFunction() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Pull Request Process

1. **Before submitting:**
   - Run the full test suite
   - Run linters
   - Update documentation
   - Add/update tests

2. **PR Title:** Use conventional commits format
   - `feat:` New feature
   - `fix:` Bug fix
   - `docs:` Documentation changes
   - `test:` Test changes
   - `refactor:` Code refactoring
   - `perf:` Performance improvements
   - `chore:` Maintenance tasks

3. **PR Description:**
   - Describe the changes
   - Link related issues
   - Include test results
   - Note breaking changes

4. **Review Process:**
   - At least one maintainer approval required
   - All CI checks must pass
   - Address review feedback promptly

## Reporting Bugs

Use GitHub Issues to report bugs. When filing an issue, please include:

- MCP-Go version
- Go version
- Operating system
- Steps to reproduce
- Expected behavior
- Actual behavior
- Relevant logs/errors

## Feature Requests

We use GitHub Issues for feature requests. Please:

- Check existing issues first
- Describe the feature clearly
- Explain the use case
- Consider implementation approach

## Documentation

- Update README.md for user-facing changes
- Update API documentation for code changes
- Add examples for new features
- Keep documentation concise and clear

## Versioning

We use [Semantic Versioning](http://semver.org/):

- MAJOR version for incompatible API changes
- MINOR version for backwards-compatible functionality
- PATCH version for backwards-compatible bug fixes

## License

By contributing, you agree that your contributions will be licensed under the Apache 2.0 License.

## Recognition

Contributors are recognized in:
- CONTRIBUTORS.md file
- Release notes
- Project documentation

## Questions?

- Open a GitHub Issue
- Join our Discord community
- Email: maintainers@example.com

Thank you for contributing to MCP-Go! 🎉