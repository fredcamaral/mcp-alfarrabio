# MCP Memory Server - Streamlined Makefile

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
	dev-up dev-down dev-logs setup-env deps tidy ensure-env \
	backend-up backend-down backend-logs frontend-up frontend-down frontend-logs \
	dev-backend-up dev-backend-down dev-backend-logs full-up full-down full-logs

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
	docker-compose -f docker-compose.dev.yml logs -f lerian-mcp-memory-server-dev

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

## Docker Compose - Segregated Services
# Backend only (fastest for backend development)
backend-up: ## Start backend services only (Qdrant + MCP Server)
	@echo "$(GREEN)Starting backend services...$(RESET)"
	docker-compose -f docker-compose.backend.yml up -d
	@echo "$(GREEN)✅ Backend services running:$(RESET)"
	@echo "  - MCP API: http://localhost:9080"
	@echo "  - Health: http://localhost:9081/health"
	@echo "  - Qdrant: http://localhost:6333"

backend-down: ## Stop backend services
	@echo "$(YELLOW)Stopping backend services...$(RESET)"
	docker-compose -f docker-compose.backend.yml down

backend-logs: ## Show backend logs
	docker-compose -f docker-compose.backend.yml logs -f

# Frontend only (requires backend running)
frontend-up: ## Start frontend service only (requires backend running)
	@echo "$(GREEN)Starting frontend service...$(RESET)"
	@echo "$(YELLOW)⚠️  Make sure backend is running (make backend-up)$(RESET)"
	@if ! docker network ls | grep -q "lerian-mcp-memory_lerian_mcp_network"; then \
		echo "$(YELLOW)Creating network (backend not running?)...$(RESET)"; \
		docker network create lerian-mcp-memory_lerian_mcp_network || true; \
	fi
	docker-compose -f docker-compose.frontend.yml up -d
	@echo "$(GREEN)✅ Frontend service running:$(RESET)"
	@echo "  - WebUI: http://localhost:2001"

frontend-down: ## Stop frontend service
	@echo "$(YELLOW)Stopping frontend service...$(RESET)"
	docker-compose -f docker-compose.frontend.yml down

frontend-logs: ## Show frontend logs
	docker-compose -f docker-compose.frontend.yml logs -f

# Development setup (backend in Docker, frontend local)
dev-backend-up: ## Start development backend (optimized for frontend development)
	@echo "$(GREEN)Starting development backend...$(RESET)"
	docker-compose -f docker-compose.dev.yml up -d
	@echo "$(GREEN)✅ Development backend running:$(RESET)"
	@echo "  - MCP API: http://localhost:9080"
	@echo "  - Qdrant: http://localhost:6333"
	@echo "$(BLUE)💡 Now run frontend locally:$(RESET)"
	@echo "  cd web-ui && npm run dev"

dev-backend-down: ## Stop development backend
	@echo "$(YELLOW)Stopping development backend...$(RESET)"
	docker-compose -f docker-compose.dev.yml down

dev-backend-logs: ## Show development backend logs
	docker-compose -f docker-compose.dev.yml logs -f

# Full stack (original setup)
full-up: ## Start full stack (backend + frontend) - original setup
	@echo "$(GREEN)Starting full stack...$(RESET)"
	docker-compose up -d
	@echo "$(GREEN)✅ Full stack running:$(RESET)"
	@echo "  - MCP API: http://localhost:9080"
	@echo "  - WebUI: http://localhost:2001"
	@echo "  - Qdrant: http://localhost:6333"

full-down: ## Stop full stack
	@echo "$(YELLOW)Stopping full stack...$(RESET)"
	docker-compose down

full-logs: ## Show full stack logs
	docker-compose logs -f

# Quick development workflows
dev-quick: ## Quick development setup (backend in Docker, instructions for frontend)
	@echo "$(BLUE)🚀 Quick Development Setup$(RESET)"
	@echo "$(GREEN)1. Starting backend services...$(RESET)"
	$(MAKE) dev-backend-up
	@echo ""
	@echo "$(BLUE)2. Start frontend locally:$(RESET)"
	@echo "  cd web-ui"
	@echo "  npm install  # if first time"
	@echo "  npm run dev"
	@echo ""
	@echo "$(BLUE)3. Access your application:$(RESET)"
	@echo "  - Frontend: http://localhost:2002 (with hot reload)"
	@echo "  - Backend API: http://localhost:9080"
	@echo ""
	@echo "$(YELLOW)When done, run: make dev-backend-down$(RESET)"

# Utility commands
docker-clean: ## Clean up Docker resources
	@echo "$(YELLOW)Cleaning up Docker resources...$(RESET)"
	docker-compose -f docker-compose.yml down -v
	docker-compose -f docker-compose.backend.yml down -v
	docker-compose -f docker-compose.frontend.yml down -v
	docker-compose -f docker-compose.dev.yml down -v
	docker system prune -f

docker-rebuild: ## Rebuild all Docker images
	@echo "$(GREEN)Rebuilding Docker images...$(RESET)"
	docker-compose -f docker-compose.yml build --no-cache
	docker-compose -f docker-compose.backend.yml build --no-cache
	docker-compose -f docker-compose.frontend.yml build --no-cache