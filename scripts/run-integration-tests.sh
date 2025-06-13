#!/bin/bash

# Integration Test Runner for Lerian MCP Memory Server v2
# This script sets up the environment and runs comprehensive integration tests

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🧪 Lerian MCP Memory Server v2 - Integration Test Runner${NC}"
echo ""

# Function to print colored output
print_status() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

# Check if we're in the right directory
if [ ! -f "go.mod" ] || [ ! -d "internal/testing" ]; then
    print_error "Please run this script from the project root directory"
    exit 1
fi

# Load environment variables if .env exists
if [ -f ".env" ]; then
    print_status "Loading environment from .env file"
    export $(grep -v '^#' .env | xargs)
else
    print_warning "No .env file found - using default settings"
fi

# Set test environment variables
export RUN_INTEGRATION_TESTS=true
export CLEANUP_AFTER_TESTS=${CLEANUP_AFTER_TESTS:-true}
export VERBOSE_TEST_LOGGING=${VERBOSE_TEST_LOGGING:-false}
export TEST_TIMEOUT=${TEST_TIMEOUT:-5m}

# Check for required dependencies
print_info "Checking system dependencies..."

# Check if Docker is available for test storage
if command -v docker &> /dev/null && docker ps &> /dev/null; then
    export HAS_DOCKER=true
    print_status "Docker is available for test storage"
else
    export HAS_DOCKER=false
    print_warning "Docker not available - tests will use mock storage"
fi

# Check if we have OpenAI API key for AI tests
if [ -n "$OPENAI_API_KEY" ] || [ -n "$TEST_OPENAI_API_KEY" ]; then
    export HAS_OPENAI=true
    print_status "OpenAI API key available for AI integration tests"
else
    export HAS_OPENAI=false
    export SKIP_REAL_AI=true
    print_warning "No OpenAI API key - AI tests will be mocked"
fi

# Function to setup test infrastructure
setup_test_infrastructure() {
    print_info "Setting up test infrastructure..."
    
    if [ "$HAS_DOCKER" = true ]; then
        print_info "Starting test storage with Docker..."
        
        # Start test Qdrant instance
        if ! docker ps | grep -q "qdrant-test"; then
            print_info "Starting test Qdrant instance..."
            docker run -d --name qdrant-test \
                -p 6334:6333 \
                -e QDRANT__SERVICE__HTTP_PORT=6333 \
                qdrant/qdrant:latest >/dev/null 2>&1 || true
            
            # Wait for Qdrant to be ready
            sleep 5
            print_status "Test Qdrant instance started on port 6334"
        else
            print_status "Test Qdrant instance already running"
        fi
        
        export TEST_QDRANT_URL="http://localhost:6334"
    fi
}

# Function to cleanup test infrastructure
cleanup_test_infrastructure() {
    if [ "$CLEANUP_AFTER_TESTS" = true ] && [ "$HAS_DOCKER" = true ]; then
        print_info "Cleaning up test infrastructure..."
        
        # Stop and remove test containers
        docker stop qdrant-test >/dev/null 2>&1 || true
        docker rm qdrant-test >/dev/null 2>&1 || true
        
        print_status "Test infrastructure cleaned up"
    fi
}

# Function to run specific test categories
run_test_category() {
    local category=$1
    local description=$2
    
    echo ""
    print_info "Running $description..."
    
    case $category in
        "unit")
            go test -v -short ./internal/testing/... -run "TestUnit"
            ;;
        "integration")
            go test -v -timeout=$TEST_TIMEOUT ./internal/testing/... -run "TestIntegration"
            ;;
        "storage")
            go test -v -timeout=$TEST_TIMEOUT ./internal/testing/... -run "TestMemoryStore|TestMemoryRetrieve"
            ;;
        "ai")
            if [ "$HAS_OPENAI" = true ]; then
                go test -v -timeout=$TEST_TIMEOUT ./internal/testing/... -run "TestMemoryAnalyze|TestAI"
            else
                print_warning "Skipping AI tests - no OpenAI API key available"
            fi
            ;;
        "templates")
            go test -v -timeout=$TEST_TIMEOUT ./internal/testing/... -run "TestTemplate"
            ;;
        "cross-tool")
            go test -v -timeout=$TEST_TIMEOUT ./internal/testing/... -run "TestCrossTool"
            ;;
        "all")
            go test -v -timeout=$TEST_TIMEOUT ./internal/testing/...
            ;;
    esac
}

# Function to show test environment status
show_test_environment() {
    echo ""
    print_info "Test Environment Status:"
    echo "  Docker Available: ${HAS_DOCKER}"
    echo "  OpenAI Available: ${HAS_OPENAI}"
    echo "  Real Storage: ${HAS_DOCKER}"
    echo "  Skip AI Tests: ${SKIP_REAL_AI:-false}"
    echo "  Test Timeout: ${TEST_TIMEOUT}"
    echo "  Cleanup After: ${CLEANUP_AFTER_TESTS}"
    echo "  Verbose Logging: ${VERBOSE_TEST_LOGGING}"
    echo ""
}

# Parse command line arguments
TEST_CATEGORY=${1:-"all"}
VALID_CATEGORIES=("unit" "integration" "storage" "ai" "templates" "cross-tool" "all")

if [[ ! " ${VALID_CATEGORIES[@]} " =~ " ${TEST_CATEGORY} " ]]; then
    print_error "Invalid test category: $TEST_CATEGORY"
    echo "Valid categories: ${VALID_CATEGORIES[*]}"
    exit 1
fi

# Main execution
echo "Test Category: $TEST_CATEGORY"
show_test_environment

# Setup signal handlers for cleanup
trap cleanup_test_infrastructure EXIT INT TERM

# Build the project first
print_info "Building project..."
go build -o bin/lerian-mcp-memory-server ./cmd/server
print_status "Project built successfully"

# Setup test infrastructure
setup_test_infrastructure

# Run the tests
case $TEST_CATEGORY in
    "unit")
        run_test_category "unit" "Unit Tests"
        ;;
    "integration")
        run_test_category "integration" "Integration Tests"
        ;;
    "storage")
        run_test_category "storage" "Storage Integration Tests"
        ;;
    "ai")
        run_test_category "ai" "AI Integration Tests"
        ;;
    "templates")
        run_test_category "templates" "Template System Tests"
        ;;
    "cross-tool")
        run_test_category "cross-tool" "Cross-Tool Workflow Tests"
        ;;
    "all")
        print_info "Running comprehensive integration test suite..."
        run_test_category "all" "All Integration Tests"
        ;;
esac

# Check test results
if [ $? -eq 0 ]; then
    echo ""
    print_status "Integration tests completed successfully!"
    echo ""
    print_info "Test Summary:"
    echo "  Category: $TEST_CATEGORY"
    echo "  Environment: Docker=$HAS_DOCKER, OpenAI=$HAS_OPENAI"
    echo "  Duration: Complete"
    echo ""
    print_status "All tests passed - MCP Memory Server v2 integration verified!"
else
    echo ""
    print_error "Integration tests failed!"
    echo ""
    print_info "Troubleshooting:"
    echo "  1. Check that all required services are running"
    echo "  2. Verify environment configuration in .env file"
    echo "  3. Run individual test categories to isolate issues"
    echo "  4. Check logs with VERBOSE_TEST_LOGGING=true"
    echo ""
    exit 1
fi