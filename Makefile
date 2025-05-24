# Claude Vector Memory MCP Server - Streamlined Makefile

# Project configuration
PROJECT_NAME := mcp-memory-server
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GO_VERSION := $(shell go version | awk '{print $$3}')

# Build configuration
BINARY_NAME := mcp-memory-server
BUILD_DIR := ./bin
DOCKER_IMAGE := mcp-memory-server
DOCKER_TAG ?= $(VERSION)

# Go configuration
GO_MODULE := $(shell go list -m)
GOFLAGS := -mod=readonly
LDFLAGS := -w -s \
	-X '$(GO_MODULE)/internal/version.Version=$(VERSION)' \
	-X '$(GO_MODULE)/internal/version.BuildTime=$(BUILD_TIME)' \
	-X '$(GO_MODULE)/internal/version.CommitHash=$(COMMIT_HASH)' \
	-X '$(GO_MODULE)/internal/version.GoVersion=$(GO_VERSION)'

# Test configuration
TEST_TIMEOUT := 300s

# Colors for output
GREEN := \033[32m
YELLOW := \033[33m
BLUE := \033[34m
RESET := \033[0m

.PHONY: help build clean test lint fmt dev docker-compose-up docker-compose-down setup-env

# Default target - show help
help: ## Show this help message
	@echo "$(BLUE)Claude Vector Memory MCP Server$(RESET)"
	@echo ""
	@echo "$(BLUE)Usage:$(RESET)"
	@echo "  make [target]"
	@echo ""
	@echo "$(BLUE)Main Targets:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(RESET) %s\\n", $$1, $$2}'
	@echo ""
	@echo "$(BLUE)Examples:$(RESET)"
	@echo "  make setup-env              # Setup environment and run server"
	@echo "  make build                  # Build the binary"
	@echo "  make test                   # Run tests"
	@echo "  make docker-compose-up      # Start with Docker Compose"
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

dev: ensure-env ## Run in development mode
	@echo "$(GREEN)Starting development server...$(RESET)"
	go run ./cmd/server -mode=stdio

run-http: ensure-env build ## Build and run HTTP server
	@echo "$(GREEN)Running $(BINARY_NAME) in HTTP mode...$(RESET)"
	$(BUILD_DIR)/$(BINARY_NAME) -mode=http -addr=:9080

test: ## Run tests
	@echo "$(GREEN)Running tests...$(RESET)"
	go test $(GOFLAGS) -timeout $(TEST_TIMEOUT) -v ./...

test-coverage: ## Run tests with coverage
	@echo "$(GREEN)Running tests with coverage...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	go test $(GOFLAGS) -timeout $(TEST_TIMEOUT) -coverprofile=$(BUILD_DIR)/coverage.out -v ./...
	go tool cover -html=$(BUILD_DIR)/coverage.out -o $(BUILD_DIR)/coverage.html
	@echo "$(GREEN)✓ Coverage report: $(BUILD_DIR)/coverage.html$(RESET)"

## Code Quality
lint: ## Run linters (install if needed)
	@echo "$(GREEN)Running linters...$(RESET)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --issues-exit-code=0 || echo "$(YELLOW)Linting issues found but continuing...$(RESET)"; \
	else \
		echo "$(YELLOW)golangci-lint not found, installing...$(RESET)"; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run --issues-exit-code=0 || echo "$(YELLOW)Linting issues found but continuing...$(RESET)"; \
	fi

fmt: ## Format code
	@echo "$(GREEN)Formatting code...$(RESET)"
	go fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	fi

## Docker Commands
docker-build: ensure-env ## Build Docker image
	@echo "$(GREEN)Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)...$(RESET)"
	docker build --build-arg VERSION=$(VERSION) --build-arg BUILD_TIME=$(BUILD_TIME) --build-arg COMMIT_HASH=$(COMMIT_HASH) -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest

docker-compose-up: ensure-env ## Start services with docker-compose
	@echo "$(GREEN)Starting services with docker-compose...$(RESET)"
	docker-compose up -d
	@echo "$(GREEN)✓ Services started! Check health at http://localhost:9080/health$(RESET)"

docker-compose-down: ## Stop services with docker-compose
	@echo "$(GREEN)Stopping services with docker-compose...$(RESET)"
	docker-compose down

docker-logs: ## View docker-compose logs
	docker-compose logs -f

## Utility Commands
deps: ## Download and verify dependencies
	@echo "$(GREEN)Downloading dependencies...$(RESET)"
	go mod download
	go mod verify

clean: ## Clean build artifacts
	@echo "$(GREEN)Cleaning build artifacts...$(RESET)"
	rm -rf $(BUILD_DIR)
	docker rmi $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest 2>/dev/null || true

health-check: ## Check application health
	@echo "$(GREEN)Checking application health...$(RESET)"
	@curl -f http://localhost:9080/health || echo "$(YELLOW)Health check failed - make sure server is running$(RESET)"

## Internal targets
ensure-env: ## Ensure .env file exists (internal)
	@if [ ! -f .env ]; then \
		echo "$(YELLOW)Creating .env from .env.example...$(RESET)"; \
		cp .env.example .env; \
		echo "$(YELLOW)⚠️  Please edit .env file with your configuration (especially OPENAI_API_KEY)$(RESET)"; \
		echo "$(YELLOW)⚠️  Run 'make setup-env' to continue$(RESET)"; \
		exit 1; \
	fi