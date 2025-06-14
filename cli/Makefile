# Makefile for Lerian MCP Memory CLI

# Variables
BINARY_NAME := lmmc
GO := go
GOFLAGS := -v
BUILD_DIR := bin
SOURCE_DIR := cmd/lerian-mcp-memory-cli

# Version information
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date +%Y-%m-%d)

# Build flags
LDFLAGS := -X 'main.BuildVersion=$(VERSION)' -X 'main.BuildCommit=$(COMMIT)' -X 'main.BuildDate=$(BUILD_DATE)'

# Targets
.PHONY: all build clean test test-coverage test-race install uninstall run help

all: clean build test ## Build and test everything

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./$(SOURCE_DIR)
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@$(GO) clean
	@echo "Clean complete"

test: ## Run tests
	@echo "Running tests..."
	$(GO) test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

test-race: ## Run tests with race detector
	@echo "Running tests with race detector..."
	$(GO) test -v -race ./...

test-integration: build ## Run integration tests
	@echo "Running integration tests..."
	$(GO) test -v -tags=integration ./tests/integration/...

test-e2e: build ## Run end-to-end tests
	@echo "Running end-to-end tests..."
	RUN_E2E_TESTS=true $(GO) test -v -tags=e2e ./tests/e2e/...

test-perf: build ## Run performance tests
	@echo "Running performance tests..."
	RUN_PERF_TESTS=true RUN_E2E_TESTS=true $(GO) test -v -tags=e2e ./tests/e2e/... -run TestCLIPerformance

test-all: test test-integration test-e2e ## Run all tests

install: build ## Install the binary and setup configuration
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
	@echo "✓ Binary installed to $(GOPATH)/bin/$(BINARY_NAME)"
	@echo ""
	@echo "Setting up configuration..."
	@./scripts/install-config.sh

uninstall: ## Uninstall the binary from GOPATH/bin
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f $(GOPATH)/bin/$(BINARY_NAME)
	@echo "Uninstalled"

run: build ## Build and run the binary
	@echo "Running $(BINARY_NAME)..."
	@./$(BUILD_DIR)/$(BINARY_NAME)

fmt: ## Format code
	@echo "Formatting code..."
	$(GO) fmt ./...
	@echo "Formatting complete"

lint: ## Run linter
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install it from https://golangci-lint.run/"; \
	fi

vet: ## Run go vet
	@echo "Running go vet..."
	$(GO) vet ./...

mod-tidy: ## Tidy go modules
	@echo "Tidying modules..."
	$(GO) mod tidy

mod-download: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GO) mod download

# Development helpers
dev-add: build ## Build and run 'add' command (for testing)
	@./$(BUILD_DIR)/$(BINARY_NAME) add "Test task from Makefile"

dev-list: build ## Build and run 'list' command (for testing)
	@./$(BUILD_DIR)/$(BINARY_NAME) list

help: ## Display this help
	@echo "Lerian MCP Memory CLI - Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help