#!/bin/bash

# Production Deployment Script for MCP Memory Server
# This script handles complete production deployment with rollback capabilities

set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
COMPOSE_PROJECT_NAME="lerian-mcp-memory"
BACKUP_DIR="${PROJECT_DIR}/backups"
LOG_DIR="${PROJECT_DIR}/logs"
DATA_DIR="${PROJECT_DIR}/data"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# Help function
show_help() {
    cat << EOF
Production Deployment Script for MCP Memory Server

Usage: $0 [OPTIONS] COMMAND

Commands:
    deploy          Deploy the complete stack
    update          Update services with zero-downtime
    rollback        Rollback to previous version
    status          Show deployment status
    logs            Show service logs
    backup          Create backup before deployment
    restore         Restore from backup
    cleanup         Clean up old deployments
    health          Check system health

Options:
    -e, --env FILE        Environment file (default: .env.production)
    -c, --config DIR      Configuration directory (default: configs/production)
    -v, --version TAG     Docker image tag to deploy (default: latest)
    -f, --force           Force deployment without confirmations
    -d, --dry-run         Show what would be done without executing
    -b, --backup          Create backup before deployment
    -m, --monitoring      Include monitoring stack
    -h, --help            Show this help message

Examples:
    $0 deploy                          # Deploy with defaults
    $0 deploy -v v1.2.3 -b            # Deploy specific version with backup
    $0 update --env .env.staging      # Update staging environment
    $0 rollback                       # Rollback to previous version
    $0 status                         # Show current deployment status

EOF
}

# Default values
ENV_FILE=".env.production"
CONFIG_DIR="configs/production"
IMAGE_TAG="latest"
FORCE=false
DRY_RUN=false
CREATE_BACKUP=false
INCLUDE_MONITORING=false
DEPLOYMENT_ID="$(date +%Y%m%d_%H%M%S)"

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -e|--env)
                ENV_FILE="$2"
                shift 2
                ;;
            -c|--config)
                CONFIG_DIR="$2"
                shift 2
                ;;
            -v|--version)
                IMAGE_TAG="$2"
                shift 2
                ;;
            -f|--force)
                FORCE=true
                shift
                ;;
            -d|--dry-run)
                DRY_RUN=true
                shift
                ;;
            -b|--backup)
                CREATE_BACKUP=true
                shift
                ;;
            -m|--monitoring)
                INCLUDE_MONITORING=true
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            deploy|update|rollback|status|logs|backup|restore|cleanup|health)
                COMMAND="$1"
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done

    if [[ -z "${COMMAND:-}" ]]; then
        log_error "No command specified"
        show_help
        exit 1
    fi
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if Docker is installed and running
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running"
        exit 1
    fi
    
    # Check if Docker Compose is available
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        log_error "Docker Compose is not installed"
        exit 1
    fi
    
    # Check environment file
    if [[ ! -f "${PROJECT_DIR}/${ENV_FILE}" ]]; then
        log_error "Environment file not found: ${ENV_FILE}"
        exit 1
    fi
    
    # Check configuration directory
    if [[ ! -d "${PROJECT_DIR}/${CONFIG_DIR}" ]]; then
        log_error "Configuration directory not found: ${CONFIG_DIR}"
        exit 1
    fi
    
    # Create necessary directories
    mkdir -p "${DATA_DIR}" "${LOG_DIR}" "${BACKUP_DIR}"
    
    log_success "Prerequisites check passed"
}

# Load environment variables
load_environment() {
    log_info "Loading environment from ${ENV_FILE}..."
    
    if [[ -f "${PROJECT_DIR}/${ENV_FILE}" ]]; then
        set -a
        source "${PROJECT_DIR}/${ENV_FILE}"
        set +a
        log_success "Environment loaded successfully"
    else
        log_error "Environment file not found: ${ENV_FILE}"
        exit 1
    fi
}

# Create data directories
setup_data_directories() {
    log_info "Setting up data directories..."
    
    local data_dirs=(
        "${DATA_DIR}/mcp-memory"
        "${DATA_DIR}/postgres"
        "${DATA_DIR}/qdrant"
        "${DATA_DIR}/redis"
        "${DATA_DIR}/prometheus"
        "${DATA_DIR}/grafana"
        "${DATA_DIR}/alertmanager"
        "${DATA_DIR}/loki"
    )
    
    for dir in "${data_dirs[@]}"; do
        if [[ ! -d "$dir" ]]; then
            log_info "Creating directory: $dir"
            mkdir -p "$dir"
            # Set appropriate permissions
            chmod 755 "$dir"
        fi
    done
    
    log_success "Data directories setup completed"
}

# Pull latest images
pull_images() {
    log_info "Pulling Docker images..."
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would pull images with tag: $IMAGE_TAG"
        return 0
    fi
    
    # Main application image
    log_info "Pulling MCP Memory Server image: $IMAGE_TAG"
    docker pull "ghcr.io/lerianstudio/lerian-mcp-memory:${IMAGE_TAG}"
    
    # Pull all other images defined in compose files
    docker-compose -f docker-compose.production.yml pull
    
    if [[ "$INCLUDE_MONITORING" == true ]]; then
        docker-compose -f docker-compose.monitoring.yml pull
    fi
    
    log_success "Images pulled successfully"
}

# Create backup
create_backup() {
    log_info "Creating backup..."
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would create backup with ID: backup_${DEPLOYMENT_ID}"
        return 0
    fi
    
    local backup_name="backup_${DEPLOYMENT_ID}"
    local backup_path="${BACKUP_DIR}/${backup_name}"
    
    mkdir -p "$backup_path"
    
    # Backup databases
    if docker ps --filter "name=${COMPOSE_PROJECT_NAME}_postgres" --filter "status=running" | grep -q postgres; then
        log_info "Creating PostgreSQL backup..."
        docker exec "${COMPOSE_PROJECT_NAME}_postgres_1" pg_dump -U "${POSTGRES_USER}" "${POSTGRES_DB}" > "${backup_path}/postgres_dump.sql"
    fi
    
    # Backup configurations
    log_info "Backing up configurations..."
    cp -r "${PROJECT_DIR}/${CONFIG_DIR}" "${backup_path}/configs"
    
    # Backup data directories (excluding large files)
    log_info "Backing up critical data..."
    rsync -av --exclude='*.log' --exclude='wal' "${DATA_DIR}/" "${backup_path}/data/"
    
    # Create metadata file
    cat > "${backup_path}/metadata.json" << EOF
{
    "backup_id": "${backup_name}",
    "timestamp": "$(date -Iseconds)",
    "deployment_id": "${DEPLOYMENT_ID}",
    "image_tag": "${IMAGE_TAG}",
    "environment": "${ENV_FILE}",
    "config_dir": "${CONFIG_DIR}"
}
EOF
    
    # Compress backup
    log_info "Compressing backup..."
    tar -czf "${backup_path}.tar.gz" -C "${BACKUP_DIR}" "${backup_name}"
    rm -rf "$backup_path"
    
    log_success "Backup created: ${backup_path}.tar.gz"
    echo "$backup_name" > "${BACKUP_DIR}/latest_backup.txt"
}

# Deploy services
deploy_services() {
    log_info "Deploying services..."
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would deploy services with configuration:"
        log_info "  - Environment: $ENV_FILE"
        log_info "  - Config Dir: $CONFIG_DIR"
        log_info "  - Image Tag: $IMAGE_TAG"
        log_info "  - Monitoring: $INCLUDE_MONITORING"
        return 0
    fi
    
    # Set image tag in environment
    export MCP_MEMORY_IMAGE_TAG="$IMAGE_TAG"
    
    # Deploy main services
    log_info "Starting main services..."
    docker-compose -f docker-compose.production.yml up -d --remove-orphans
    
    # Deploy monitoring if requested
    if [[ "$INCLUDE_MONITORING" == true ]]; then
        log_info "Starting monitoring services..."
        docker-compose -f docker-compose.monitoring.yml up -d --remove-orphans
    fi
    
    log_success "Services deployment completed"
}

# Wait for services to be healthy
wait_for_services() {
    log_info "Waiting for services to become healthy..."
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would wait for services to become healthy"
        return 0
    fi
    
    local max_attempts=60
    local attempt=0
    
    while [[ $attempt -lt $max_attempts ]]; do
        if check_health_status; then
            log_success "All services are healthy"
            return 0
        fi
        
        attempt=$((attempt + 1))
        log_info "Waiting for services... (attempt $attempt/$max_attempts)"
        sleep 10
    done
    
    log_error "Services did not become healthy within expected time"
    return 1
}

# Check health status
check_health_status() {
    local unhealthy_services=()
    
    # Check main services
    for service in mcp-memory-server postgres qdrant redis nginx; do
        if ! docker ps --filter "name=${COMPOSE_PROJECT_NAME}_${service}" --filter "health=healthy" | grep -q "$service"; then
            unhealthy_services+=("$service")
        fi
    done
    
    if [[ ${#unhealthy_services[@]} -eq 0 ]]; then
        return 0
    else
        log_warning "Unhealthy services: ${unhealthy_services[*]}"
        return 1
    fi
}

# Show deployment status
show_status() {
    log_info "Deployment Status:"
    echo
    
    # Show running containers
    echo "Running Containers:"
    docker ps --filter "label=com.docker.compose.project=${COMPOSE_PROJECT_NAME}" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
    echo
    
    # Show service health
    echo "Service Health:"
    for service in mcp-memory-server postgres qdrant redis nginx; do
        local container_name="${COMPOSE_PROJECT_NAME}_${service}_1"
        if docker ps --filter "name=$container_name" | grep -q "$service"; then
            local health=$(docker inspect --format='{{.State.Health.Status}}' "$container_name" 2>/dev/null || echo "unknown")
            echo "  $service: $health"
        else
            echo "  $service: not running"
        fi
    done
    echo
    
    # Show resource usage
    echo "Resource Usage:"
    docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}"
}

# Show logs
show_logs() {
    local service="${1:-}"
    
    if [[ -n "$service" ]]; then
        log_info "Showing logs for service: $service"
        docker-compose -f docker-compose.production.yml logs -f "$service"
    else
        log_info "Showing logs for all services"
        docker-compose -f docker-compose.production.yml logs -f
    fi
}

# Rollback deployment
rollback_deployment() {
    log_info "Rolling back deployment..."
    
    if [[ ! -f "${BACKUP_DIR}/latest_backup.txt" ]]; then
        log_error "No backup found for rollback"
        exit 1
    fi
    
    local backup_name=$(cat "${BACKUP_DIR}/latest_backup.txt")
    local backup_file="${BACKUP_DIR}/${backup_name}.tar.gz"
    
    if [[ ! -f "$backup_file" ]]; then
        log_error "Backup file not found: $backup_file"
        exit 1
    fi
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would rollback using backup: $backup_name"
        return 0
    fi
    
    # Stop current services
    log_info "Stopping current services..."
    docker-compose -f docker-compose.production.yml down
    
    # Restore backup
    log_info "Restoring from backup: $backup_name"
    tar -xzf "$backup_file" -C "${BACKUP_DIR}"
    
    # Restore configurations
    cp -r "${BACKUP_DIR}/${backup_name}/configs" "${PROJECT_DIR}/"
    
    # Restore critical data
    rsync -av "${BACKUP_DIR}/${backup_name}/data/" "${DATA_DIR}/"
    
    # Restart services
    log_info "Restarting services..."
    docker-compose -f docker-compose.production.yml up -d
    
    # Clean up
    rm -rf "${BACKUP_DIR}/${backup_name}"
    
    log_success "Rollback completed successfully"
}

# Cleanup old deployments
cleanup_old_deployments() {
    log_info "Cleaning up old deployments..."
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would clean up old Docker images and backups"
        return 0
    fi
    
    # Remove unused Docker images
    log_info "Removing unused Docker images..."
    docker image prune -f
    
    # Remove old backups (keep last 5)
    log_info "Cleaning up old backups..."
    cd "$BACKUP_DIR"
    ls -t backup_*.tar.gz | tail -n +6 | xargs -r rm -f
    
    log_success "Cleanup completed"
}

# Main execution
main() {
    parse_args "$@"
    
    log_info "Starting deployment script with command: $COMMAND"
    log_info "Configuration: env=$ENV_FILE, config=$CONFIG_DIR, tag=$IMAGE_TAG"
    
    case "$COMMAND" in
        deploy)
            check_prerequisites
            load_environment
            setup_data_directories
            
            if [[ "$CREATE_BACKUP" == true ]]; then
                create_backup
            fi
            
            pull_images
            deploy_services
            wait_for_services
            show_status
            ;;
        update)
            check_prerequisites
            load_environment
            create_backup
            pull_images
            deploy_services
            wait_for_services
            show_status
            ;;
        rollback)
            check_prerequisites
            load_environment
            rollback_deployment
            wait_for_services
            show_status
            ;;
        status)
            show_status
            ;;
        logs)
            show_logs "${1:-}"
            ;;
        backup)
            check_prerequisites
            load_environment
            create_backup
            ;;
        cleanup)
            cleanup_old_deployments
            ;;
        health)
            if check_health_status; then
                log_success "All services are healthy"
                exit 0
            else
                log_error "Some services are unhealthy"
                exit 1
            fi
            ;;
        *)
            log_error "Unknown command: $COMMAND"
            show_help
            exit 1
            ;;
    esac
    
    log_success "Deployment script completed successfully"
}

# Execute main function with all arguments
main "$@"