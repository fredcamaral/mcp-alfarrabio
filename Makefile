# Lerian MCP Memory Server - Makefile

# Project configuration
PROJECT_NAME := lerian-mcp-memory-server
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build configuration
BINARY_NAME := lerian-mcp-memory-server
BUILD_DIR := ./bin
DOCKER_IMAGE := lerian-mcp-memory-server

# Go configuration
GO_MODULE := $(shell go list -m)
GOFLAGS := -mod=readonly
LDFLAGS := -w -s \
	-X '$(GO_MODULE)/internal/version.Version=$(VERSION)' \
	-X '$(GO_MODULE)/internal/version.BuildTime=$(BUILD_TIME)' \
	-X '$(GO_MODULE)/internal/version.CommitHash=$(COMMIT_HASH)'

# Colors for output
GREEN := \033[32m
YELLOW := \033[33m
BLUE := \033[34m
RESET := \033[0m

.PHONY: help build clean test lint fmt vet dev docker-build docker-up docker-down \
	setup-env deps tidy ensure-env test-coverage test-integration test-race benchmark ci

# Default target - show help
help: ## Show this help message
	@echo "$(BLUE)Lerian MCP Memory Server$(RESET)"
	@echo ""
	@echo "$(BLUE)Usage:$(RESET)"
	@echo "  make [target]"
	@echo ""
	@echo "$(BLUE)Essential Targets:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BLUE)Quick Start:$(RESET)"
	@echo "  make setup-env              # Setup environment and run server"
	@echo "  make build                  # Build the binary"
	@echo "  make test                   # Run tests"
	@echo "  make docker-up              # Start with Docker"
	@echo ""

## Environment Setup
setup-env: ## Setup .env file and run development server
	@echo "$(GREEN)Setting up environment...$(RESET)"
	@if [ ! -f .env ]; then \
		echo "$(YELLOW)Copying .env.example to .env...$(RESET)"; \
		cp .env.example .env; \
		echo "$(YELLOW)⚠️  Please edit .env file with your configuration (especially OPENAI_API_KEY)$(RESET)"; \
	else \
		echo "$(GREEN)✓ .env file already exists$(RESET)"; \
	fi
	@$(MAKE) dev

## Core Development Commands
build: ## Build the binary
	@echo "$(GREEN)Building $(BINARY_NAME) $(VERSION)...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server
	@echo "$(GREEN)✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)$(RESET)"

dev: ensure-env ## Run in development mode (stdio)
	@echo "$(GREEN)Starting development server (stdio mode)...$(RESET)"
	go run ./cmd/server -mode=stdio

dev-http: ensure-env ## Run in development mode (HTTP)
	@echo "$(GREEN)Starting development server (HTTP mode)...$(RESET)"
	go run ./cmd/server -mode=http -addr=:9080

## Testing
test: ## Run tests
	@echo "$(GREEN)Running tests...$(RESET)"
	go test -short -v ./...

test-coverage: ## Run tests with coverage (70% threshold)
	@echo "$(GREEN)Running tests with coverage...$(RESET)"
	go test -short -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report: coverage.html$(RESET)"
	go tool cover -func=coverage.out | tail -1

test-integration: ## Run integration tests
	@echo "$(GREEN)Running integration tests...$(RESET)"
	go test -tags=integration -v ./...

test-race: ## Run tests with race detector
	@echo "$(GREEN)Running tests with race detector...$(RESET)"
	go test -race -short ./...

benchmark: ## Run benchmarks
	@echo "$(GREEN)Running benchmarks...$(RESET)"
	go test -bench=. -benchmem ./...

## Code Quality
lint: ## Run linters
	@echo "$(GREEN)Running linters...$(RESET)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --config .golangci.yml ./...; \
	else \
		echo "$(YELLOW)golangci-lint not found, installing...$(RESET)"; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run --config .golangci.yml ./...; \
	fi

fmt: ## Format code
	@echo "$(GREEN)Formatting code...$(RESET)"
	go fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	else \
		echo "$(YELLOW)goimports not found, installing...$(RESET)"; \
		go install golang.org/x/tools/cmd/goimports@latest; \
		goimports -w .; \
	fi

vet: ## Run go vet
	@echo "$(GREEN)Running go vet...$(RESET)"
	go vet ./...

security-scan: ## Run security scan (gosec + govulncheck)
	@echo "$(GREEN)Running security scan...$(RESET)"
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "$(YELLOW)gosec not found, installing...$(RESET)"; \
		go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
		gosec ./...; \
	fi
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "$(YELLOW)govulncheck not found, installing...$(RESET)"; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
		govulncheck ./...; \
	fi

ci: ## Run complete CI pipeline
	@echo "$(GREEN)Running complete CI pipeline...$(RESET)"
	$(MAKE) fmt
	$(MAKE) vet
	$(MAKE) lint
	$(MAKE) security-scan
	$(MAKE) test-coverage
	$(MAKE) build

## Docker Commands
docker-build: ensure-env ## Build Docker image
	@echo "$(GREEN)Building Docker image $(DOCKER_IMAGE):$(VERSION)...$(RESET)"
	docker build --build-arg VERSION=$(VERSION) --build-arg BUILD_TIME=$(BUILD_TIME) --build-arg COMMIT_HASH=$(COMMIT_HASH) -t $(DOCKER_IMAGE):$(VERSION) .
	docker tag $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest

docker-up: ensure-env ## Start services with docker-compose
	@echo "$(GREEN)Starting Lerian MCP Memory Server...$(RESET)"
	docker-compose up -d
	@echo "$(GREEN)✅ Services started!$(RESET)"
	@echo "  - MCP API: http://localhost:9080"
	@echo "  - Health: http://localhost:8081/health"
	@echo "  - Qdrant: http://localhost:6333"
	@echo ""
	@echo "$(BLUE)Test the server:$(RESET)"
	@echo "  curl http://localhost:8081/health"
	@echo "  curl -X POST http://localhost:9080/mcp -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"method\":\"tools/list\",\"id\":1}'"

docker-down: ## Stop services with docker-compose
	@echo "$(GREEN)Stopping services...$(RESET)"
	docker-compose down

docker-logs: ## View Docker logs
	docker-compose logs -f

docker-restart: ## Restart Docker services
	@echo "$(GREEN)Restarting services...$(RESET)"
	docker-compose restart

docker-clean: ## Clean up Docker resources
	@echo "$(YELLOW)Cleaning up Docker resources...$(RESET)"
	docker-compose down -v
	docker system prune -f

docker-rebuild: ## Rebuild Docker images
	@echo "$(GREEN)Rebuilding Docker images...$(RESET)"
	docker-compose build --no-cache

## Utility Commands
clean: ## Clean build artifacts
	@echo "$(GREEN)Cleaning build artifacts...$(RESET)"
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	docker rmi $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest 2>/dev/null || true

deps: ## Download and verify dependencies
	@echo "$(GREEN)Downloading dependencies...$(RESET)"
	go mod download
	go mod verify

tidy: ## Tidy go modules
	@echo "$(GREEN)Tidying go modules...$(RESET)"
	go mod tidy

## Demo and Testing Commands
demo: ## Run demo client
	@echo "$(GREEN)Running demo client...$(RESET)"
	go run ./cmd/demo

test-mcp: ## Run MCP protocol test
	@echo "$(GREEN)Testing MCP protocol...$(RESET)"
	go run ./cmd/test-mcp

graphql: ## Start GraphQL server
	@echo "$(GREEN)Starting GraphQL server...$(RESET)"
	go run ./cmd/graphql

## Production Commands
prod-deploy: ci docker-build ## Deploy to production (CI + Docker build)
	@echo "$(GREEN)Production deployment ready$(RESET)"
	@echo "$(YELLOW)Push image: docker push $(DOCKER_IMAGE):$(VERSION)$(RESET)"

health-check: ## Check server health
	@echo "$(GREEN)Checking server health...$(RESET)"
	@curl -f http://localhost:8081/health || echo "$(YELLOW)Server not responding$(RESET)"

## Internal targets
ensure-env: ## Ensure .env file exists (internal)
	@if [ ! -f .env ]; then \
		echo "$(YELLOW)Creating .env from .env.example...$(RESET)"; \
		cp .env.example .env; \
		echo "$(YELLOW)⚠️  Please edit .env file with your configuration (especially OPENAI_API_KEY)$(RESET)"; \
		echo "$(YELLOW)⚠️  Run 'make setup-env' to continue$(RESET)"; \
		exit 1; \
	fi