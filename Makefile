# Lerian MCP Memory - Unified Makefile

# Project configuration
PROJECT_NAME := lerian-mcp-memory
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build configuration
BUILD_DIR := ./bin
DOCKER_IMAGE := lerian-mcp-memory-server

# Go configuration
GO_MODULE := $(shell go list -m)
GOFLAGS := -mod=readonly
SERVER_LDFLAGS := -w -s \
	-X '$(GO_MODULE)/internal/version.Version=$(VERSION)' \
	-X '$(GO_MODULE)/internal/version.BuildTime=$(BUILD_TIME)' \
	-X '$(GO_MODULE)/internal/version.CommitHash=$(COMMIT_HASH)'
CLI_LDFLAGS := -X 'main.BuildVersion=$(VERSION)' -X 'main.BuildCommit=$(COMMIT_HASH)' -X 'main.BuildDate=$(BUILD_TIME)'

# Colors for output
GREEN := \033[32m
YELLOW := \033[33m
BLUE := \033[34m
RESET := \033[0m

.PHONY: help setup build build-server build-cli install test clean \
	dev dev-server dev-cli docker-up docker-down \
	fmt lint vet security-scan ci \
	deps tidy health-check

# Default target - show help
help: ## Show this help message
	@echo "$(BLUE)Lerian MCP Memory - AI-Powered Task Management$(RESET)"
	@echo ""
	@echo "$(BLUE)Quick Start:$(RESET)"
	@echo "  make setup                  # Setup environment"
	@echo "  make build                  # Build server and CLI"
	@echo "  make install                # Install CLI to PATH"
	@echo "  make dev                    # Start development server"
	@echo "  make docker-up              # Start with Docker"
	@echo ""
	@echo "$(BLUE)Available Commands:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(RESET) %s\n", $$1, $$2}'

## Essential Commands

setup: ## Setup development environment
	@echo "$(GREEN)Setting up development environment...$(RESET)"
	@if [ ! -f .env ]; then \
		echo "$(YELLOW)Creating .env from template...$(RESET)"; \
		cp .env.example .env; \
		echo "$(YELLOW)⚠️  Edit .env file with your configuration (especially OPENAI_API_KEY)$(RESET)"; \
	else \
		echo "$(GREEN)✓ Environment already configured$(RESET)"; \
	fi
	@$(MAKE) deps
	@echo "$(GREEN)✓ Setup complete$(RESET)"

build: build-server build-cli ## Build both server and CLI

build-server: ## Build the server binary
	@echo "$(GREEN)Building server...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 go build $(GOFLAGS) -ldflags="$(SERVER_LDFLAGS)" -o $(BUILD_DIR)/lerian-mcp-memory-server ./cmd/server
	@echo "$(GREEN)✓ Server built: $(BUILD_DIR)/lerian-mcp-memory-server$(RESET)"

build-cli: ## Build the CLI binary
	@echo "$(GREEN)Building CLI...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	@cd cli && CGO_ENABLED=0 go build $(GOFLAGS) -ldflags="$(CLI_LDFLAGS)" -o ../$(BUILD_DIR)/lmmc ./cmd/lerian-mcp-memory-cli
	@echo "$(GREEN)✓ CLI built: $(BUILD_DIR)/lmmc$(RESET)"

install: build-cli ## Install CLI to PATH
	@echo "$(GREEN)Installing CLI to PATH...$(RESET)"
	@cp -f $(BUILD_DIR)/lmmc $(GOPATH)/bin/
	@echo "$(GREEN)✓ CLI installed: $(GOPATH)/bin/lmmc$(RESET)"
	@echo "$(GREEN)Setting up CLI configuration...$(RESET)"
	@./scripts/setup-cli-config.sh
	@echo "$(BLUE)Try: lmmc --help$(RESET)"

## Development

dev: dev-server ## Start development server (alias)

dev-server: ensure-env ## Start development server (stdio mode)
	@echo "$(GREEN)Starting development server...$(RESET)"
	@go run ./cmd/server -mode=stdio

dev-cli: build-cli ## Run CLI interactively
	@echo "$(GREEN)Starting CLI...$(RESET)"
	@./$(BUILD_DIR)/lmmc

## Testing

test: ## Run all tests (server + CLI)
	@echo "$(GREEN)Running server tests...$(RESET)"
	@go test -short -v ./...
	@echo "$(GREEN)Running CLI tests...$(RESET)"
	@cd cli && go test -short -v ./...
	@echo "$(GREEN)✓ All tests completed$(RESET)"

test-coverage: ## Run tests with coverage report
	@echo "$(GREEN)Running tests with coverage...$(RESET)"
	@go test -short -coverprofile=server-coverage.out ./...
	@cd cli && go test -short -coverprofile=../cli-coverage.out ./...
	@go tool cover -html=server-coverage.out -o server-coverage.html
	@go tool cover -html=cli-coverage.out -o cli-coverage.html
	@echo "$(GREEN)Coverage reports: server-coverage.html, cli-coverage.html$(RESET)"

## Code Quality

fmt: ## Format code (server + CLI)
	@echo "$(GREEN)Formatting code...$(RESET)"
	@go fmt ./...
	@cd cli && go fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
		cd cli && goimports -w .; \
	fi
	@echo "$(GREEN)✓ Code formatted$(RESET)"

lint: ## Run linters (server + CLI)
	@echo "$(GREEN)Running linters...$(RESET)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --config .golangci.yml ./...; \
		cd cli && golangci-lint run ./...; \
	else \
		echo "$(YELLOW)Installing golangci-lint...$(RESET)"; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run --config .golangci.yml ./...; \
		cd cli && golangci-lint run ./...; \
	fi
	@echo "$(GREEN)✓ Linting complete$(RESET)"

vet: ## Run go vet (server + CLI)
	@echo "$(GREEN)Running go vet...$(RESET)"
	@go vet ./...
	@cd cli && go vet ./...
	@echo "$(GREEN)✓ Vet complete$(RESET)"

security-scan: ## Run security scans (server + CLI)
	@echo "$(GREEN)Running security scans...$(RESET)"
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
		cd cli && gosec ./...; \
	else \
		echo "$(YELLOW)Installing gosec...$(RESET)"; \
		go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
		gosec ./...; \
		cd cli && gosec ./...; \
	fi
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
		cd cli && govulncheck ./...; \
	else \
		echo "$(YELLOW)Installing govulncheck...$(RESET)"; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
		govulncheck ./...; \
		cd cli && govulncheck ./...; \
	fi
	@echo "$(GREEN)✓ Security scan complete$(RESET)"

ci: ## Run complete CI pipeline
	@echo "$(GREEN)Running CI pipeline...$(RESET)"
	@$(MAKE) fmt
	@$(MAKE) vet
	@$(MAKE) lint
	@$(MAKE) security-scan
	@$(MAKE) test-coverage
	@$(MAKE) build
	@echo "$(GREEN)✓ CI pipeline complete$(RESET)"

## Docker

docker-build: ensure-env ## Build Docker image
	@echo "$(GREEN)Building Docker image...$(RESET)"
	@docker build --build-arg VERSION=$(VERSION) --build-arg BUILD_TIME=$(BUILD_TIME) --build-arg COMMIT_HASH=$(COMMIT_HASH) -t $(DOCKER_IMAGE):$(VERSION) .
	@docker tag $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest
	@echo "$(GREEN)✓ Docker image built$(RESET)"

docker-up: ensure-env ## Start production services
	@echo "$(GREEN)Starting production services...$(RESET)"
	@docker compose --profile prod up -d
	@echo "$(GREEN)✅ Services started!$(RESET)"
	@echo "  • MCP API: http://localhost:9080"
	@echo "  • Health: http://localhost:8081/health"
	@echo "  • Qdrant: http://localhost:6333"
	@echo ""
	@echo "$(BLUE)Test: curl http://localhost:8081/health$(RESET)"

docker-down: ## Stop all services
	@echo "$(GREEN)Stopping services...$(RESET)"
	@docker compose --profile prod --profile dev --profile monitoring down

docker-dev: ensure-env ## Start development services with hot reload
	@echo "$(GREEN)Starting development environment...$(RESET)"
	@docker compose --profile dev up -d
	@echo "$(GREEN)✅ Development services started!$(RESET)"
	@echo "$(YELLOW)Hot reload enabled$(RESET)"

docker-logs: ## View service logs
	@docker compose --profile prod logs -f

## Utilities

clean: ## Clean build artifacts
	@echo "$(GREEN)Cleaning build artifacts...$(RESET)"
	@rm -rf $(BUILD_DIR)
	@rm -f *-coverage.out *-coverage.html
	@docker rmi $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest 2>/dev/null || true
	@echo "$(GREEN)✓ Clean complete$(RESET)"

deps: ## Download and verify dependencies
	@echo "$(GREEN)Managing dependencies...$(RESET)"
	@go mod download && go mod verify
	@cd cli && go mod download && go mod verify
	@echo "$(GREEN)✓ Dependencies ready$(RESET)"

tidy: ## Tidy go modules
	@echo "$(GREEN)Tidying modules...$(RESET)"
	@go mod tidy
	@cd cli && go mod tidy
	@echo "$(GREEN)✓ Modules tidied$(RESET)"

health-check: ## Check server health
	@echo "$(GREEN)Checking server health...$(RESET)"
	@curl -f http://localhost:8081/health || echo "$(YELLOW)Server not responding$(RESET)"

## Advanced (for developers)

migrate-up: ## Run database migrations
	@echo "$(GREEN)Running migrations...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/migrate ./cmd/migrate/main.go
	@$(BUILD_DIR)/migrate -command=migrate -verbose

benchmark: ## Run performance benchmarks
	@echo "$(GREEN)Running benchmarks...$(RESET)"
	@go test -bench=. -benchmem ./...
	@cd cli && go test -bench=. -benchmem ./...

integration-test: ## Run integration tests
	@echo "$(GREEN)Running integration tests...$(RESET)"
	@./scripts/run-integration-tests.sh all

## Internal targets
ensure-env: ## Ensure .env file exists (internal)
	@if [ ! -f .env ]; then \
		echo "$(YELLOW)Creating .env from template...$(RESET)"; \
		cp .env.example .env; \
		echo "$(YELLOW)⚠️  Edit .env file with your configuration$(RESET)"; \
		exit 1; \
	fi