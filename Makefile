# Claude Vector Memory MCP Server - Makefile
# Inspired by HashiCorp's build system best practices

# Project configuration
PROJECT_NAME := mcp-memory-server
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GO_VERSION := $(shell go version | awk '{print $$3}')

# Build configuration
BINARY_NAME := mcp-memory-server
BUILD_DIR := ./bin
DIST_DIR := ./dist
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

# Platform targets
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

# Docker configuration
DOCKER_BUILD_ARGS := --build-arg VERSION=$(VERSION) \
	--build-arg BUILD_TIME=$(BUILD_TIME) \
	--build-arg COMMIT_HASH=$(COMMIT_HASH)

# Test configuration
TEST_TIMEOUT := 300s
TEST_COVERAGE_THRESHOLD := 80

# Colors for output
RED := \033[31m
GREEN := \033[32m
YELLOW := \033[33m
BLUE := \033[34m
RESET := \033[0m

.PHONY: all build clean test lint fmt vet tidy deps dev docker docker-build docker-run help
.PHONY: cross-compile package release install uninstall security-scan
.PHONY: test-unit test-integration test-e2e test-coverage benchmark
.PHONY: docker-compose-up docker-compose-down docker-logs
.PHONY: migrate backup restore health-check

# Default target
all: clean deps lint test build

# Quick build target (skip tests)
quick: clean deps lint build ## Quick build without tests

## Build commands
build: ## Build the binary
	@echo "$(GREEN)Building $(BINARY_NAME) $(VERSION)...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server
	@echo "$(GREEN)✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)$(RESET)"

build-race: ## Build with race detection
	@echo "$(GREEN)Building $(BINARY_NAME) with race detection...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 go build $(GOFLAGS) -race -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-race ./cmd/server

cross-compile: ## Build for multiple platforms
	@echo "$(GREEN)Cross-compiling for multiple platforms...$(RESET)"
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d'/' -f1); \
		arch=$$(echo $$platform | cut -d'/' -f2); \
		output_name=$(BINARY_NAME)-$$os-$$arch; \
		if [ $$os = "windows" ]; then output_name=$$output_name.exe; fi; \
		echo "Building for $$os/$$arch..."; \
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build $(GOFLAGS) \
			-ldflags="$(LDFLAGS)" \
			-o $(DIST_DIR)/$$output_name ./cmd/server; \
	done
	@echo "$(GREEN)✓ Cross-compilation complete$(RESET)"

## Development commands
dev: ## Run in development mode with hot reload
	@echo "$(GREEN)Starting development server...$(RESET)"
	go run ./cmd/server --config ./configs/dev/config.yaml --log-level debug

run: build ## Build and run the server
	@echo "$(GREEN)Running $(BINARY_NAME)...$(RESET)"
	$(BUILD_DIR)/$(BINARY_NAME) --config ./configs/dev/config.yaml

install: build ## Install the binary to GOPATH/bin
	@echo "$(GREEN)Installing $(BINARY_NAME)...$(RESET)"
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/

uninstall: ## Remove the binary from GOPATH/bin
	@echo "$(GREEN)Uninstalling $(BINARY_NAME)...$(RESET)"
	rm -f $(GOPATH)/bin/$(BINARY_NAME)

## Testing commands
test: ## Run all tests
	@echo "$(GREEN)Running tests...$(RESET)"
	go test $(GOFLAGS) -timeout $(TEST_TIMEOUT) -v ./...

test-unit: ## Run unit tests only
	@echo "$(GREEN)Running unit tests...$(RESET)"
	go test $(GOFLAGS) -timeout $(TEST_TIMEOUT) -short -v ./...

test-integration: ## Run integration tests only
	@echo "$(GREEN)Running integration tests...$(RESET)"
	go test $(GOFLAGS) -timeout $(TEST_TIMEOUT) -run "TestIntegration" -v ./...

test-e2e: ## Run end-to-end tests
	@echo "$(GREEN)Running e2e tests...$(RESET)"
	go test $(GOFLAGS) -timeout $(TEST_TIMEOUT) -run "TestE2E" -v ./...

test-coverage: ## Run tests with coverage
	@echo "$(GREEN)Running tests with coverage...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	go test $(GOFLAGS) -timeout $(TEST_TIMEOUT) -coverprofile=$(BUILD_DIR)/coverage.out -v ./...
	go tool cover -html=$(BUILD_DIR)/coverage.out -o $(BUILD_DIR)/coverage.html
	@coverage=$$(go tool cover -func=$(BUILD_DIR)/coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	echo "Coverage: $$coverage%"; \
	if [ $$(echo "$$coverage < $(TEST_COVERAGE_THRESHOLD)" | bc -l) -eq 1 ]; then \
		echo "$(RED)✗ Coverage $$coverage% is below threshold $(TEST_COVERAGE_THRESHOLD)%$(RESET)"; \
		exit 1; \
	else \
		echo "$(GREEN)✓ Coverage $$coverage% meets threshold$(RESET)"; \
	fi

benchmark: ## Run benchmarks
	@echo "$(GREEN)Running benchmarks...$(RESET)"
	go test $(GOFLAGS) -bench=. -benchmem -v ./...

## Code quality commands
lint: ## Run linters (warnings only)
	@echo "$(GREEN)Running linters...$(RESET)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --issues-exit-code=0 || echo "$(YELLOW)Linting issues found but continuing...$(RESET)"; \
	else \
		echo "$(YELLOW)golangci-lint not found, installing...$(RESET)"; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run --issues-exit-code=0 || echo "$(YELLOW)Linting issues found but continuing...$(RESET)"; \
	fi

lint-strict: ## Run linters (fail on issues)
	@echo "$(GREEN)Running linters (strict mode)...$(RESET)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "$(YELLOW)golangci-lint not found, installing...$(RESET)"; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run; \
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

tidy: ## Tidy go modules
	@echo "$(GREEN)Tidying go modules...$(RESET)"
	go mod tidy
	go mod verify

security-scan: ## Run security scanning
	@echo "$(GREEN)Running security scan...$(RESET)"
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "$(YELLOW)gosec not found, installing...$(RESET)"; \
		go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
		gosec ./...; \
	fi

## Dependency management
deps: ## Download dependencies
	@echo "$(GREEN)Downloading dependencies...$(RESET)"
	go mod download
	go mod verify

deps-update: ## Update dependencies
	@echo "$(GREEN)Updating dependencies...$(RESET)"
	go get -u ./...
	go mod tidy

deps-vendor: ## Vendor dependencies
	@echo "$(GREEN)Vendoring dependencies...$(RESET)"
	go mod vendor

## Docker commands
docker-build: ## Build Docker image
	@echo "$(GREEN)Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)...$(RESET)"
	docker build $(DOCKER_BUILD_ARGS) -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest

docker-run: docker-build ## Build and run Docker container
	@echo "$(GREEN)Running Docker container...$(RESET)"
	docker run --rm -p 8080:8080 -p 8081:8081 -p 8082:8082 $(DOCKER_IMAGE):$(DOCKER_TAG)

docker-compose-up: ## Start services with docker-compose
	@echo "$(GREEN)Starting services with docker-compose...$(RESET)"
	docker-compose up -d

docker-compose-down: ## Stop services with docker-compose
	@echo "$(GREEN)Stopping services with docker-compose...$(RESET)"
	docker-compose down

docker-logs: ## View docker-compose logs
	docker-compose logs -f

## Database operations
migrate: ## Run database migrations
	@echo "$(GREEN)Running database migrations...$(RESET)"
	$(BUILD_DIR)/$(BINARY_NAME) migrate --config ./configs/dev/config.yaml

migrate-create: ## Create new migration
	@if [ -z "$(NAME)" ]; then \
		echo "$(RED)Usage: make migrate-create NAME=migration_name$(RESET)"; \
		exit 1; \
	fi
	@echo "$(GREEN)Creating migration $(NAME)...$(RESET)"
	$(BUILD_DIR)/$(BINARY_NAME) migrate create $(NAME) --config ./configs/dev/config.yaml

backup: ## Create database backup
	@echo "$(GREEN)Creating database backup...$(RESET)"
	$(BUILD_DIR)/$(BINARY_NAME) backup --config ./configs/dev/config.yaml

restore: ## Restore database from backup
	@if [ -z "$(FILE)" ]; then \
		echo "$(RED)Usage: make restore FILE=backup_file$(RESET)"; \
		exit 1; \
	fi
	@echo "$(GREEN)Restoring database from $(FILE)...$(RESET)"
	$(BUILD_DIR)/$(BINARY_NAME) restore $(FILE) --config ./configs/dev/config.yaml

## Health and monitoring
health-check: ## Check application health
	@echo "$(GREEN)Checking application health...$(RESET)"
	@curl -f http://localhost:8081/health || echo "$(RED)Health check failed$(RESET)"

metrics: ## View application metrics
	@echo "$(GREEN)Fetching application metrics...$(RESET)"
	@curl -s http://localhost:8082/metrics

## Release commands
package: cross-compile ## Package binaries for release
	@echo "$(GREEN)Packaging binaries for release...$(RESET)"
	@mkdir -p $(DIST_DIR)/packages
	@for file in $(DIST_DIR)/$(BINARY_NAME)-*; do \
		if [ -f "$$file" ]; then \
			basename=$$(basename $$file); \
			tar -czf $(DIST_DIR)/packages/$$basename.tar.gz -C $(DIST_DIR) $$basename; \
		fi; \
	done
	@echo "$(GREEN)✓ Packaging complete$(RESET)"

release: clean deps lint test cross-compile package ## Create a full release
	@echo "$(GREEN)✓ Release $(VERSION) ready in $(DIST_DIR)/packages/$(RESET)"

## Cleanup commands
clean: ## Clean build artifacts
	@echo "$(GREEN)Cleaning build artifacts...$(RESET)"
	rm -rf $(BUILD_DIR) $(DIST_DIR)
	docker rmi $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest 2>/dev/null || true

clean-all: clean ## Clean everything including caches
	@echo "$(GREEN)Cleaning everything...$(RESET)"
	go clean -cache -testcache -modcache
	docker system prune -f

## Help
help: ## Show this help message
	@echo "$(BLUE)Claude Vector Memory MCP Server - Build System$(RESET)"
	@echo ""
	@echo "$(BLUE)Usage:$(RESET)"
	@echo "  make [target]"
	@echo ""
	@echo "$(BLUE)Targets:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BLUE)Examples:$(RESET)"
	@echo "  make build                   # Build the binary"
	@echo "  make test                    # Run all tests"
	@echo "  make docker-build            # Build Docker image"
	@echo "  make release                 # Create full release"
	@echo ""
	@echo "$(BLUE)Environment Variables:$(RESET)"
	@echo "  VERSION      # Version tag (default: git describe)"
	@echo "  DOCKER_TAG   # Docker image tag (default: VERSION)"
	@echo ""