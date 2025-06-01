# MCP Memory Server - Streamlined Makefile

# Project configuration
PROJECT_NAME := mcp-memory-server
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build configuration
BINARY_NAME := mcp-memory-server
BUILD_DIR := ./bin
DOCKER_IMAGE := mcp-memory-server

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
	dev-up dev-down dev-logs setup-env deps tidy ensure-env

# Default target - show help
help: ## Show this help message
	@echo "$(BLUE)MCP Memory Server$(RESET)"
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
	@echo "  make dev-up                 # Start with hot reload"
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
	@echo "$(GREEN)Starting development server...$(RESET)"
	go run ./cmd/server -mode=stdio

test: ## Run tests
	@echo "$(GREEN)Running tests...$(RESET)"
	go test -short -v ./...

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

## Docker Commands
docker-build: ensure-env ## Build Docker image
	@echo "$(GREEN)Building Docker image $(DOCKER_IMAGE):$(VERSION)...$(RESET)"
	docker build --build-arg VERSION=$(VERSION) --build-arg BUILD_TIME=$(BUILD_TIME) --build-arg COMMIT_HASH=$(COMMIT_HASH) -t $(DOCKER_IMAGE):$(VERSION) .
	docker tag $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest

docker-up: ensure-env ## Start services with docker-compose
	@echo "$(GREEN)Starting services with docker-compose...$(RESET)"
	docker-compose up -d
	@echo "$(GREEN)✓ Services started! Check health at http://localhost:9080/health$(RESET)"

docker-down: ## Stop services with docker-compose
	@echo "$(GREEN)Stopping services with docker-compose...$(RESET)"
	docker-compose down

## Development Mode Commands (with Hot Reload)
dev-up: ensure-env ## Start development mode with hot reload
	@echo "$(GREEN)Starting development mode with hot reload...$(RESET)"
	docker-compose -f docker-compose.dev.yml up -d
	@echo "$(GREEN)✓ Development server started with hot reload!$(RESET)"
	@echo "$(GREEN)✓ MCP endpoint: http://localhost:9080/mcp$(RESET)"
	@echo "$(GREEN)✓ Health check: http://localhost:9080/health$(RESET)"
	@echo "$(YELLOW)📝 Edit any Go file and the server will automatically reload$(RESET)"

dev-down: ## Stop development mode
	@echo "$(GREEN)Stopping development mode...$(RESET)"
	docker-compose -f docker-compose.dev.yml down

dev-logs: ## View development mode logs
	docker-compose -f docker-compose.dev.yml logs -f mcp-memory-server-dev

## Utility Commands
clean: ## Clean build artifacts
	@echo "$(GREEN)Cleaning build artifacts...$(RESET)"
	rm -rf $(BUILD_DIR)
	docker rmi $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest 2>/dev/null || true

deps: ## Download and verify dependencies
	@echo "$(GREEN)Downloading dependencies...$(RESET)"
	go mod download
	go mod verify

tidy: ## Tidy go modules
	@echo "$(GREEN)Tidying go modules...$(RESET)"
	go mod tidy

## Internal targets
ensure-env: ## Ensure .env file exists (internal)
	@if [ ! -f .env ]; then \
		echo "$(YELLOW)Creating .env from .env.example...$(RESET)"; \
		cp .env.example .env; \
		echo "$(YELLOW)⚠️  Please edit .env file with your configuration (especially OPENAI_API_KEY)$(RESET)"; \
		echo "$(YELLOW)⚠️  Run 'make setup-env' to continue$(RESET)"; \
		exit 1; \
	fi