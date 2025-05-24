.PHONY: build run test clean docker-up docker-down install-deps lint vet

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=mcp-memory
BINARY_UNIX=$(BINARY_NAME)_unix

# Build the project
build:
	$(GOBUILD) -o $(BINARY_NAME) -v ./cmd/server

# Run the project
run:
	$(GOBUILD) -o $(BINARY_NAME) -v ./cmd/server
	./$(BINARY_NAME)

# Run with hot reload (requires air: go install github.com/cosmtrek/air@latest)
dev:
	air

# Test all packages
test:
	$(GOTEST) -v ./...

# Test with coverage (excluding cmd/server)
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./internal/... ./pkg/...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Test with coverage percentage (excluding cmd/server)
test-cover:
	$(GOTEST) -v -coverprofile=coverage.out ./internal/... ./pkg/...
	$(GOCMD) tool cover -func=coverage.out

# Test coverage with target percentage (excluding cmd/server)
test-cover-check:
	$(GOTEST) -v -coverprofile=coverage.out ./internal/... ./pkg/...
	@COVERAGE=$$($(GOCMD) tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	echo "Total coverage: $$COVERAGE%"; \
	if [ $$(echo "$$COVERAGE >= 70.0" | bc -l) -eq 1 ]; then \
		echo "✅ Coverage target achieved (≥70%)"; \
	else \
		echo "❌ Coverage target not met (<70%)"; \
		exit 1; \
	fi

# Run tests with race detection
test-race:
	$(GOTEST) -v -race ./...

# Test specific package
test-pkg:
	@read -p "Enter package (e.g., ./pkg/types): " pkg; \
	$(GOTEST) -v $$pkg

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -f *.out
	rm -f coverage.html
	rm -rf bin/
	rm -rf dist/
	@echo "✅ All artifacts cleaned!"

# Install dependencies
install-deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Lint code
lint:
	golangci-lint run

# Lint with fixes
lint-fix:
	golangci-lint run --fix

# Install linting tools
install-lint:
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.55.2; \
	else \
		echo "golangci-lint already installed"; \
	fi

# Format code
fmt:
	$(GOCMD) fmt ./...

# Vet code
vet:
	$(GOCMD) vet ./...

# Build for Linux
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v ./cmd/server

# Docker operations
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

# Quality check (run all checks)
quality: fmt vet lint test-cover-check
	@echo "✅ All quality checks passed!"

# Development workflow
setup: install-deps docker-up
	@echo "✅ Development environment ready!"

# Help
help:
	@echo "Available commands:"
	@echo "  build         - Build the binary"
	@echo "  run           - Build and run the server"
	@echo "  dev           - Run with hot reload (requires air)"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  clean         - Clean build artifacts"
	@echo "  install-deps  - Install Go dependencies"
	@echo "  lint          - Run linter"
	@echo "  fmt           - Format code"
	@echo "  docker-up     - Start Docker containers"
	@echo "  docker-down   - Stop Docker containers"
	@echo "  setup         - Setup development environment"
	@echo "  help          - Show this help"