# MCP Memory Server - Streamlined Makefile

# Project configuration
PROJECT_NAME := mcp-memory-server
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GO_VERSION := $(shell go version | awk '{print $$3}')
BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")

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
TEST_TIMEOUT ?= 300s
COVERAGE_THRESHOLD ?= 70
BENCH_TIME ?= 10s
INTEGRATION_TAG := integration

# Colors for output
GREEN := \033[32m
YELLOW := \033[33m
BLUE := \033[34m
RESET := \033[0m

.PHONY: help build clean test lint fmt dev docker-compose-up docker-compose-down setup-env \
	test-integration test-e2e benchmark security-scan test-race check-mod \
	install uninstall proto generate release changelog

# Default target - show help
help: ## Show this help message
	@echo "$(BLUE)MCP Memory Server$(RESET)"
	@echo ""
	@echo "$(BLUE)Usage:$(RESET)"
	@echo "  make [target]"
	@echo ""
	@echo "$(BLUE)Main Targets:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(RESET) %s\n", $$1, $$2}'
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
	$(BUILD_DIR)/$(BINARY_NAME) -mode=http -addr=:${MCP_HOST_PORT:-9080}

test: ## Run tests
	@echo "$(GREEN)Running tests...$(RESET)"
	go test $(GOFLAGS) -timeout $(TEST_TIMEOUT) -v ./...

test-coverage: ## Run tests with coverage
	@echo "$(GREEN)Running tests with coverage...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	go test $(GOFLAGS) -timeout $(TEST_TIMEOUT) -coverprofile=$(BUILD_DIR)/coverage.out -covermode=atomic -v ./...
	go tool cover -html=$(BUILD_DIR)/coverage.out -o $(BUILD_DIR)/coverage.html
	@echo "$(GREEN)✓ Coverage report: $(BUILD_DIR)/coverage.html$(RESET)"
	@coverage=$$(go tool cover -func=$(BUILD_DIR)/coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ "$$coverage" -lt "$(COVERAGE_THRESHOLD)" ]; then \
		echo "$(YELLOW)⚠️  Coverage $$coverage% is below threshold $(COVERAGE_THRESHOLD)%$(RESET)"; \
	else \
		echo "$(GREEN)✓ Coverage $$coverage% meets threshold $(COVERAGE_THRESHOLD)%$(RESET)"; \
	fi

test-integration: ## Run integration tests
	@echo "$(GREEN)Running integration tests...$(RESET)"
	go test $(GOFLAGS) -timeout $(TEST_TIMEOUT) -tags=$(INTEGRATION_TAG) -v ./...

test-e2e: ## Run end-to-end tests
	@echo "$(GREEN)Running end-to-end tests...$(RESET)"
	go test $(GOFLAGS) -timeout $(TEST_TIMEOUT) -tags=e2e -v ./tests/e2e/...

test-race: ## Run tests with race detector
	@echo "$(GREEN)Running tests with race detector...$(RESET)"
	go test $(GOFLAGS) -timeout $(TEST_TIMEOUT) -race -v ./...

benchmark: ## Run benchmarks
	@echo "$(GREEN)Running benchmarks...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	go test $(GOFLAGS) -bench=. -benchtime=$(BENCH_TIME) -benchmem -cpuprofile=$(BUILD_DIR)/cpu.prof -memprofile=$(BUILD_DIR)/mem.prof ./... | tee $(BUILD_DIR)/bench.txt
	@echo "$(GREEN)✓ Benchmark results saved to $(BUILD_DIR)/bench.txt$(RESET)"

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

vet: ## Run go vet
	@echo "$(GREEN)Running go vet...$(RESET)"
	go vet ./...

security-scan: ## Run security scanning tools
	@echo "$(GREEN)Running security scans...$(RESET)"
	@if command -v gosec >/dev/null 2>&1; then \
		gosec -fmt=text ./...; \
	else \
		echo "$(YELLOW)gosec not found, installing...$(RESET)"; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
		gosec -fmt=text ./...; \
	fi
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "$(YELLOW)govulncheck not found, installing...$(RESET)"; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
		govulncheck ./...; \
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

check-mod: ## Check for outdated dependencies
	@echo "$(GREEN)Checking for outdated dependencies...$(RESET)"
	go list -u -m all

tidy: ## Tidy go modules
	@echo "$(GREEN)Tidying go modules...$(RESET)"
	go mod tidy

vendor: ## Create vendor directory
	@echo "$(GREEN)Creating vendor directory...$(RESET)"
	go mod vendor

clean: ## Clean build artifacts
	@echo "$(GREEN)Cleaning build artifacts...$(RESET)"
	rm -rf $(BUILD_DIR)
	docker rmi $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest 2>/dev/null || true

health-check: ## Check application health
	@echo "$(GREEN)Checking application health...$(RESET)"
	@curl -f http://localhost:${MCP_HOST_PORT:-9080}/health || echo "$(YELLOW)Health check failed - make sure server is running$(RESET)"

install: build ## Install binary to GOPATH/bin
	@echo "$(GREEN)Installing $(BINARY_NAME)...$(RESET)"
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
	@echo "$(GREEN)✓ Installed to $(GOPATH)/bin/$(BINARY_NAME)$(RESET)"

uninstall: ## Uninstall binary from GOPATH/bin
	@echo "$(GREEN)Uninstalling $(BINARY_NAME)...$(RESET)"
	rm -f $(GOPATH)/bin/$(BINARY_NAME)
	@echo "$(GREEN)✓ Uninstalled$(RESET)"

generate: ## Run go generate
	@echo "$(GREEN)Running go generate...$(RESET)"
	go generate ./...

release: ## Create a new release (requires VERSION parameter)
	@if [ -z "$(VERSION)" ]; then \
		echo "$(YELLOW)Please specify VERSION (e.g., make release VERSION=v1.0.0)$(RESET)"; \
		exit 1; \
	fi
	@echo "$(GREEN)Creating release $(VERSION)...$(RESET)"
	git tag -a $(VERSION) -m "Release $(VERSION)"
	git push origin $(VERSION)
	@echo "$(GREEN)✓ Release $(VERSION) created$(RESET)"

changelog: ## Generate changelog
	@echo "$(GREEN)Generating changelog...$(RESET)"
	@if command -v git-chglog >/dev/null 2>&1; then \
		git-chglog -o CHANGELOG.md; \
	else \
		echo "$(YELLOW)git-chglog not found, using basic git log...$(RESET)"; \
		git log --pretty=format:"- %s" --reverse > CHANGELOG.md; \
	fi

all: deps fmt vet lint test build ## Run all checks and build

ci: deps fmt vet lint test-race test-coverage security-scan build ## Run CI pipeline locally

## Internal targets
ensure-env: ## Ensure .env file exists (internal)
	@if [ ! -f .env ]; then \
		echo "$(YELLOW)Creating .env from .env.example...$(RESET)"; \
		cp .env.example .env; \
		echo "$(YELLOW)⚠️  Please edit .env file with your configuration (especially OPENAI_API_KEY)$(RESET)"; \
		echo "$(YELLOW)⚠️  Run 'make setup-env' to continue$(RESET)"; \
		exit 1; \
	fi