#!/bin/bash
# Monitoring Stack Management Script for Lerian MCP Memory Server
# This script provides easy commands to manage the monitoring infrastructure

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
COMPOSE_FILE="$PROJECT_ROOT/docker-compose.yml"

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

# Check if Docker and Docker Compose are available
check_dependencies() {
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed or not in PATH"
        exit 1
    fi

    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        log_error "Docker Compose is not installed or not in PATH"
        exit 1
    fi
}

# Get Docker Compose command (handle both docker-compose and docker compose)
get_compose_cmd() {
    if command -v docker-compose &> /dev/null; then
        echo "docker-compose"
    else
        echo "docker compose"
    fi
}

# Check if .env file exists
check_env_file() {
    if [[ ! -f "$PROJECT_ROOT/.env" ]]; then
        log_warning ".env file not found. Creating from .env.example..."
        cp "$PROJECT_ROOT/.env.example" "$PROJECT_ROOT/.env"
        log_info "Please edit .env file with your configuration before proceeding"
        log_info "At minimum, set your OPENAI_API_KEY"
        return 1
    fi
}

# Start monitoring stack
start_monitoring() {
    log_info "Starting monitoring stack..."
    
    local compose_cmd=$(get_compose_cmd)
    
    # Start core services first
    log_info "Starting core services (Qdrant + MCP Memory Server)..."
    $compose_cmd --profile dev up -d qdrant lerian-mcp-memory-server
    
    # Wait a bit for core services to be ready
    log_info "Waiting for core services to be ready..."
    sleep 10
    
    # Start monitoring services
    log_info "Starting monitoring services (Prometheus + Grafana + Alertmanager)..."
    $compose_cmd --profile monitoring up -d
    
    log_success "Monitoring stack started successfully!"
    print_access_info
}

# Stop monitoring stack
stop_monitoring() {
    log_info "Stopping monitoring stack..."
    
    local compose_cmd=$(get_compose_cmd)
    $compose_cmd --profile monitoring down
    
    log_success "Monitoring stack stopped successfully!"
}

# Restart monitoring stack
restart_monitoring() {
    log_info "Restarting monitoring stack..."
    stop_monitoring
    sleep 2
    start_monitoring
}

# Show status of monitoring services
show_status() {
    log_info "Checking monitoring stack status..."
    
    local compose_cmd=$(get_compose_cmd)
    $compose_cmd --profile monitoring ps
    
    echo ""
    log_info "Service health checks:"
    
    # Check each service
    check_service_health "MCP Memory Server" "http://localhost:9081/health"
    check_service_health "Prometheus" "http://localhost:9090/-/healthy"
    check_service_health "Grafana" "http://localhost:3000/api/health"
    check_service_health "Alertmanager" "http://localhost:9093/-/healthy"
    check_service_health "Qdrant" "http://localhost:6333/health"
}

# Check health of individual service
check_service_health() {
    local service_name="$1"
    local health_url="$2"
    
    if curl -s -f "$health_url" > /dev/null 2>&1; then
        log_success "$service_name is healthy"
    else
        log_error "$service_name is not responding"
    fi
}

# Print access information
print_access_info() {
    echo ""
    log_info "Monitoring stack is now running. Access URLs:"
    echo "  🔧 MCP Memory Server:  http://localhost:9080"
    echo "  ❤️  Health Check:       http://localhost:9081/health"
    echo "  📊 Metrics:            http://localhost:8082/metrics"
    echo "  📈 Prometheus:         http://localhost:9090"
    echo "  📊 Grafana:            http://localhost:3000 (admin/admin)"
    echo "  🚨 Alertmanager:       http://localhost:9093"
    echo "  🗄️  Qdrant:             http://localhost:6333"
    echo ""
    log_info "Default Grafana credentials: admin/admin (change after first login)"
}

# View logs for monitoring services
view_logs() {
    local service="${1:-}"
    local compose_cmd=$(get_compose_cmd)
    
    if [[ -z "$service" ]]; then
        log_info "Showing logs for all monitoring services..."
        $compose_cmd --profile monitoring logs -f
    else
        log_info "Showing logs for $service..."
        $compose_cmd logs -f "$service"
    fi
}

# Update monitoring configuration
update_config() {
    log_info "Updating monitoring configuration..."
    
    local compose_cmd=$(get_compose_cmd)
    
    # Restart Prometheus to reload config
    log_info "Reloading Prometheus configuration..."
    $compose_cmd restart prometheus
    
    # Restart Alertmanager to reload config
    log_info "Reloading Alertmanager configuration..."
    $compose_cmd restart alertmanager
    
    log_success "Configuration updated successfully!"
}

# Backup monitoring data
backup_data() {
    local backup_dir="$PROJECT_ROOT/backups/monitoring-$(date +%Y%m%d-%H%M%S)"
    
    log_info "Creating monitoring data backup..."
    mkdir -p "$backup_dir"
    
    # Backup Prometheus data
    docker run --rm -v mcp_memory_prometheus_data:/data -v "$backup_dir":/backup alpine tar czf /backup/prometheus-data.tar.gz -C /data .
    
    # Backup Grafana data
    docker run --rm -v mcp_memory_grafana_data:/data -v "$backup_dir":/backup alpine tar czf /backup/grafana-data.tar.gz -C /data .
    
    # Backup Alertmanager data
    docker run --rm -v mcp_memory_alertmanager_data:/data -v "$backup_dir":/backup alpine tar czf /backup/alertmanager-data.tar.gz -C /data .
    
    log_success "Monitoring data backed up to: $backup_dir"
}

# Clean up old monitoring data
cleanup_data() {
    read -p "Are you sure you want to clean up monitoring data? This will delete all metrics history. (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log_info "Cleaning up monitoring data..."
        
        local compose_cmd=$(get_compose_cmd)
        $compose_cmd --profile monitoring down -v
        
        log_success "Monitoring data cleaned up successfully!"
        log_info "You can restart the monitoring stack with: $0 start"
    else
        log_info "Cleanup cancelled"
    fi
}

# Show help
show_help() {
    echo "Lerian MCP Memory Server - Monitoring Stack Management"
    echo ""
    echo "USAGE:"
    echo "  $0 <command> [options]"
    echo ""
    echo "COMMANDS:"
    echo "  start           Start the monitoring stack"
    echo "  stop            Stop the monitoring stack"
    echo "  restart         Restart the monitoring stack"
    echo "  status          Show status of monitoring services"
    echo "  logs [service]  Show logs (all services or specific service)"
    echo "  update-config   Reload monitoring configuration"
    echo "  backup          Backup monitoring data"
    echo "  cleanup         Clean up monitoring data (WARNING: deletes all data)"
    echo "  help            Show this help message"
    echo ""
    echo "EXAMPLES:"
    echo "  $0 start                    # Start monitoring stack"
    echo "  $0 logs                     # Show logs for all services"
    echo "  $0 logs prometheus          # Show logs for Prometheus only"
    echo "  $0 status                   # Check service status"
    echo ""
    echo "MONITORING URLS:"
    echo "  Grafana:      http://localhost:3000"
    echo "  Prometheus:   http://localhost:9090"
    echo "  Alertmanager: http://localhost:9093"
}

# Main script logic
main() {
    check_dependencies
    
    local command="${1:-help}"
    
    case "$command" in
        "start")
            if ! check_env_file; then
                exit 1
            fi
            start_monitoring
            ;;
        "stop")
            stop_monitoring
            ;;
        "restart")
            if ! check_env_file; then
                exit 1
            fi
            restart_monitoring
            ;;
        "status")
            show_status
            ;;
        "logs")
            view_logs "${2:-}"
            ;;
        "update-config")
            update_config
            ;;
        "backup")
            backup_data
            ;;
        "cleanup")
            cleanup_data
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            log_error "Unknown command: $command"
            echo ""
            show_help
            exit 1
            ;;
    esac
}

# Run main function
main "$@"