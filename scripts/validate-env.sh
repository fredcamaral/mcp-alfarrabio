#!/bin/bash

# Environment Variables Validation Script
# Validates that key environment variables are properly configured

set -euo pipefail

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if .env file exists
check_env_file() {
    if [[ -f ".env" ]]; then
        log_success ".env file found"
        return 0
    else
        log_warning ".env file not found, checking .env.example"
        if [[ -f ".env.example" ]]; then
            log_info "Using .env.example for validation"
            return 0
        else
            log_error "Neither .env nor .env.example found"
            return 1
        fi
    fi
}

# Load environment variables
load_env() {
    if [[ -f ".env" ]]; then
        set -a
        source .env
        set +a
        log_success "Loaded environment from .env"
    elif [[ -f ".env.example" ]]; then
        set -a
        source .env.example
        set +a
        log_info "Loaded environment from .env.example"
    fi
}

# Validate required variables
validate_required() {
    log_info "Validating required environment variables..."
    
    local errors=0
    
    # Core required variables
    if [[ -z "${OPENAI_API_KEY:-}" ]]; then
        log_error "OPENAI_API_KEY is required but not set"
        errors=$((errors + 1))
    else
        log_success "OPENAI_API_KEY is set"
    fi
    
    if [[ -z "${AI_PROVIDER:-}" ]]; then
        log_warning "AI_PROVIDER not set (will auto-detect)"
    else
        log_success "AI_PROVIDER set to: ${AI_PROVIDER}"
    fi
    
    return $errors
}

# Validate database configuration
validate_database() {
    log_info "Validating database configuration..."
    
    local errors=0
    
    if [[ -n "${DATABASE_URL:-}" ]]; then
        log_success "DATABASE_URL is configured"
    else
        # Check individual DB settings
        local db_vars=("DB_HOST" "DB_PORT" "DB_NAME" "DB_USER" "DB_PASSWORD")
        local missing_vars=()
        
        for var in "${db_vars[@]}"; do
            if [[ -z "${!var:-}" ]]; then
                missing_vars+=("$var")
            fi
        done
        
        if [[ ${#missing_vars[@]} -eq 0 ]]; then
            log_success "Individual database variables are configured"
        else
            log_error "Missing database variables: ${missing_vars[*]}"
            errors=$((errors + 1))
        fi
    fi
    
    return $errors
}

# Validate Qdrant configuration
validate_qdrant() {
    log_info "Validating Qdrant configuration..."
    
    local errors=0
    
    if [[ -z "${MCP_MEMORY_QDRANT_HOST:-}" ]]; then
        log_error "MCP_MEMORY_QDRANT_HOST is required but not set"
        errors=$((errors + 1))
    else
        log_success "MCP_MEMORY_QDRANT_HOST set to: ${MCP_MEMORY_QDRANT_HOST}"
    fi
    
    if [[ -z "${MCP_MEMORY_QDRANT_PORT:-}" ]]; then
        log_warning "MCP_MEMORY_QDRANT_PORT not set (will use default)"
    else
        log_success "MCP_MEMORY_QDRANT_PORT set to: ${MCP_MEMORY_QDRANT_PORT}"
    fi
    
    return $errors
}

# Validate server configuration
validate_server() {
    log_info "Validating server configuration..."
    
    if [[ -z "${MCP_MEMORY_HOST:-}" ]]; then
        log_warning "MCP_MEMORY_HOST not set (will use default)"
    else
        log_success "MCP_MEMORY_HOST set to: ${MCP_MEMORY_HOST}"
    fi
    
    if [[ -z "${MCP_MEMORY_PORT:-}" ]]; then
        log_warning "MCP_MEMORY_PORT not set (will use default)"
    else
        log_success "MCP_MEMORY_PORT set to: ${MCP_MEMORY_PORT}"
    fi
    
    if [[ -z "${MCP_MEMORY_LOG_LEVEL:-}" ]]; then
        log_warning "MCP_MEMORY_LOG_LEVEL not set (will use default)"
    else
        log_success "MCP_MEMORY_LOG_LEVEL set to: ${MCP_MEMORY_LOG_LEVEL}"
    fi
}

# Validate Docker configuration
validate_docker() {
    log_info "Validating Docker configuration..."
    
    if command -v docker &> /dev/null; then
        log_success "Docker is installed"
        
        if docker-compose config --quiet 2>/dev/null; then
            log_success "docker-compose.yml configuration is valid"
        else
            log_error "docker-compose.yml configuration has errors"
            return 1
        fi
    else
        log_warning "Docker not installed (skipping Docker validation)"
    fi
}

# Check for unused variables (variables in .env but not in codebase)
check_unused_variables() {
    log_info "Checking for potentially unused variables..."
    
    # List of known unused variables that users might have
    local unused_vars=(
        "MONITORING_ENABLED"
        "PROMETHEUS_PORT"
        "GRAFANA_PORT"
        "ALERTMANAGER_PORT"
        "MCP_MEMORY_ENABLE_TEAM_LEARNING"
        "MCP_MEMORY_MAX_REPOSITORIES"
        "WATCHTOWER_POLL_INTERVAL"
        "WATCHTOWER_LABEL_ENABLE"
        "WATCHTOWER_CLEANUP"
    )
    
    local found_unused=()
    
    for var in "${unused_vars[@]}"; do
        if [[ -n "${!var:-}" ]]; then
            found_unused+=("$var")
        fi
    done
    
    if [[ ${#found_unused[@]} -gt 0 ]]; then
        log_warning "Found unused variables (safe to remove): ${found_unused[*]}"
        log_info "These variables are declared but not implemented in the codebase"
    else
        log_success "No unused variables detected"
    fi
}

# Main validation function
main() {
    echo "========================================"
    echo "Environment Variables Validation"
    echo "========================================"
    echo
    
    local total_errors=0
    
    # Check for env file
    if ! check_env_file; then
        log_error "Cannot proceed without environment file"
        exit 1
    fi
    
    # Load environment
    load_env
    
    # Run validations
    echo
    if ! validate_required; then
        total_errors=$((total_errors + 1))
    fi
    
    echo
    if ! validate_database; then
        total_errors=$((total_errors + 1))
    fi
    
    echo
    if ! validate_qdrant; then
        total_errors=$((total_errors + 1))
    fi
    
    echo
    validate_server
    
    echo
    if ! validate_docker; then
        total_errors=$((total_errors + 1))
    fi
    
    echo
    check_unused_variables
    
    # Summary
    echo
    echo "========================================"
    if [[ $total_errors -eq 0 ]]; then
        log_success "Environment validation completed successfully!"
        echo "Your environment configuration is ready for use."
    else
        log_error "Environment validation failed with $total_errors error(s)"
        echo "Please fix the errors above before proceeding."
        exit 1
    fi
    echo "========================================"
}

# Execute main function
main "$@"