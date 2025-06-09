# Integration Tests Makefile for MT-003 functionality

.PHONY: all test test-sync test-network test-websocket test-conflict test-quick test-full clean setup help

# Default target
all: test

# Setup dependencies for integration tests
setup:
	@echo "Setting up integration test dependencies..."
	go mod tidy
	go get github.com/testcontainers/testcontainers-go
	go get github.com/testcontainers/testcontainers-go/wait
	go get github.com/stretchr/testify/suite
	go get github.com/gorilla/websocket
	@echo "Dependencies installed"

# Run all integration tests
test: setup
	@echo "Running all MT-003 integration tests..."
	go test -v -timeout=30m ./...

# Run specific test suites
test-sync:
	@echo "Running bidirectional sync integration tests..."
	go test -v -timeout=15m -run "TestSyncIntegration" ./...

test-network:
	@echo "Running network resilience tests..."
	go test -v -timeout=20m -run "TestNetworkResilience" ./...

test-websocket:
	@echo "Running WebSocket resilience tests..."
	go test -v -timeout=15m -run "TestWebSocketResilience" ./...

test-conflict:
	@echo "Running conflict resolution tests..."
	go test -v -timeout=15m -run "TestConflictResolution" ./...

# Quick test run (shorter timeouts for CI)
test-quick:
	@echo "Running quick integration tests..."
	go test -short -v -timeout=10m ./...

# Full test run with coverage
test-full: setup
	@echo "Running full integration test suite with coverage..."
	go test -v -timeout=45m -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Test with race detection
test-race:
	@echo "Running integration tests with race detection..."
	go test -race -v -timeout=30m ./...

# Run tests in parallel
test-parallel:
	@echo "Running integration tests in parallel..."
	go test -parallel=4 -v -timeout=30m ./...

# Performance/stress testing
test-performance:
	@echo "Running performance-focused integration tests..."
	go test -v -timeout=45m -run "Performance|Stress|Load" ./...

# Test network conditions
test-network-conditions:
	@echo "Running network condition simulation tests..."
	go test -v -timeout=25m -run "Latency|Timeout|Bandwidth|Drop" ./...

# Clean up test artifacts and containers
clean:
	@echo "Cleaning up test artifacts..."
	-rm -f coverage.out coverage.html
	-docker container prune -f
	-docker image prune -f
	@echo "Cleanup completed"

# Check Docker environment
check-docker:
	@echo "Checking Docker environment..."
	@docker version >/dev/null 2>&1 || (echo "Docker is not available" && exit 1)
	@docker info >/dev/null 2>&1 || (echo "Docker daemon is not running" && exit 1)
	@echo "Docker environment is ready"

# Lint test files
lint:
	@echo "Linting integration test files..."
	golangci-lint run ./...

# Format test files
fmt:
	@echo "Formatting integration test files..."
	go fmt ./...

# Run security scan on test files
security:
	@echo "Running security scan on integration tests..."
	gosec ./...

# Pre-commit checks
pre-commit: fmt lint security test-quick

# CI pipeline
ci: check-docker setup test-full

# Development workflow
dev: fmt lint test-quick

# Generate test documentation
docs:
	@echo "Generating test documentation..."
	@echo "# MT-003 Integration Tests" > TEST_GUIDE.md
	@echo "" >> TEST_GUIDE.md
	@echo "## Test Suites" >> TEST_GUIDE.md
	@echo "" >> TEST_GUIDE.md
	@echo "### Bidirectional Sync Tests" >> TEST_GUIDE.md
	@echo "- Tests complete sync flow between CLI and server" >> TEST_GUIDE.md
	@echo "- Validates WebSocket real-time notifications" >> TEST_GUIDE.md
	@echo "- Tests batch synchronization" >> TEST_GUIDE.md
	@echo "" >> TEST_GUIDE.md
	@echo "### Network Resilience Tests" >> TEST_GUIDE.md
	@echo "- Tests behavior under network latency" >> TEST_GUIDE.md
	@echo "- Tests timeout and retry scenarios" >> TEST_GUIDE.md
	@echo "- Tests bandwidth limitations" >> TEST_GUIDE.md
	@echo "- Tests connection drops and recovery" >> TEST_GUIDE.md
	@echo "" >> TEST_GUIDE.md
	@echo "### WebSocket Resilience Tests" >> TEST_GUIDE.md
	@echo "- Tests WebSocket connection establishment" >> TEST_GUIDE.md
	@echo "- Tests automatic reconnection" >> TEST_GUIDE.md
	@echo "- Tests exponential backoff" >> TEST_GUIDE.md
	@echo "- Tests concurrent connections" >> TEST_GUIDE.md
	@echo "- Tests long-running connections" >> TEST_GUIDE.md
	@echo "- Tests ping-pong keep-alive" >> TEST_GUIDE.md
	@echo "" >> TEST_GUIDE.md
	@echo "### Conflict Resolution Tests" >> TEST_GUIDE.md
	@echo "- Tests content conflicts" >> TEST_GUIDE.md
	@echo "- Tests priority conflicts" >> TEST_GUIDE.md
	@echo "- Tests status conflicts" >> TEST_GUIDE.md
	@echo "- Tests Qdrant-based resolution" >> TEST_GUIDE.md
	@echo "- Tests batch conflict resolution" >> TEST_GUIDE.md
	@echo "" >> TEST_GUIDE.md
	@echo "## Running Tests" >> TEST_GUIDE.md
	@echo "" >> TEST_GUIDE.md
	@echo '```bash' >> TEST_GUIDE.md
	@echo '# Run all tests' >> TEST_GUIDE.md
	@echo 'make test' >> TEST_GUIDE.md
	@echo '' >> TEST_GUIDE.md
	@echo '# Run specific test suite' >> TEST_GUIDE.md
	@echo 'make test-sync' >> TEST_GUIDE.md
	@echo 'make test-network' >> TEST_GUIDE.md
	@echo 'make test-websocket' >> TEST_GUIDE.md
	@echo 'make test-conflict' >> TEST_GUIDE.md
	@echo '```' >> TEST_GUIDE.md
	@echo "Test documentation generated: TEST_GUIDE.md"

# Show help
help:
	@echo "MT-003 Integration Test Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  setup              - Install test dependencies"
	@echo "  test               - Run all integration tests"
	@echo "  test-sync          - Run bidirectional sync tests"
	@echo "  test-network       - Run network resilience tests"
	@echo "  test-websocket     - Run WebSocket resilience tests"
	@echo "  test-conflict      - Run conflict resolution tests"
	@echo "  test-quick         - Run quick tests (for development)"
	@echo "  test-full          - Run full tests with coverage"
	@echo "  test-race          - Run tests with race detection"
	@echo "  test-parallel      - Run tests in parallel"
	@echo "  test-performance   - Run performance tests"
	@echo "  clean              - Clean up test artifacts"
	@echo "  check-docker       - Check Docker environment"
	@echo "  lint               - Lint test files"
	@echo "  fmt                - Format test files"
	@echo "  security           - Run security scan"
	@echo "  pre-commit         - Run pre-commit checks"
	@echo "  ci                 - Run CI pipeline"
	@echo "  dev                - Development workflow"
	@echo "  docs               - Generate test documentation"
	@echo "  help               - Show this help"
	@echo ""
	@echo "Examples:"
	@echo "  make test                    # Run all tests"
	@echo "  make test-quick              # Quick test run"
	@echo "  make test-sync               # Test sync functionality"
	@echo "  make clean                   # Clean up"