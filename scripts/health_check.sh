#!/bin/bash

# Comprehensive Health Check Script for MCP Memory Server
# Monitors all services, dependencies, and system health

set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
LOG_DIR="${PROJECT_DIR}/logs"
COMPOSE_PROJECT_NAME="lerian-mcp-memory"
HEALTH_CHECK_TIMEOUT=30

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Health check results
declare -A HEALTH_RESULTS
declare -A HEALTH_DETAILS
declare -A PERFORMANCE_METRICS

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

# Show help
show_help() {
    cat << EOF
Comprehensive Health Check Script for MCP Memory Server

Usage: $0 [OPTIONS] [COMMAND]

Commands:
    check           Run complete health check (default)
    services        Check service health only
    dependencies    Check external dependencies
    performance     Check performance metrics
    network         Check network connectivity
    storage         Check storage health
    monitoring      Check monitoring stack
    continuous      Run continuous health monitoring

Options:
    -v, --verbose           Enable verbose output
    -q, --quiet             Quiet mode (errors only)
    -f, --format FORMAT     Output format: text, json, prometheus (default: text)
    -t, --timeout SECONDS   Health check timeout (default: 30)
    -w, --webhook URL       Send results to webhook
    -a, --alert             Send alerts for failures
    --nagios                Nagios-compatible output
    --prometheus-port PORT  Expose metrics on port for Prometheus
    -h, --help              Show this help message

Examples:
    $0                              # Run complete health check
    $0 services                     # Check services only
    $0 -f json                      # Output in JSON format
    $0 continuous -t 60             # Continuous monitoring with 60s timeout
    $0 --nagios                     # Nagios-compatible check

Exit Codes:
    0 - All checks passed
    1 - Warning conditions detected
    2 - Critical failures detected

EOF
}

# Default values
COMMAND="check"
VERBOSE=false
QUIET=false
OUTPUT_FORMAT="text"
WEBHOOK_URL=""
SEND_ALERTS=false
NAGIOS_OUTPUT=false
PROMETHEUS_PORT=""
CONTINUOUS_MODE=false

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -q|--quiet)
                QUIET=true
                shift
                ;;
            -f|--format)
                OUTPUT_FORMAT="$2"
                shift 2
                ;;
            -t|--timeout)
                HEALTH_CHECK_TIMEOUT="$2"
                shift 2
                ;;
            -w|--webhook)
                WEBHOOK_URL="$2"
                shift 2
                ;;
            -a|--alert)
                SEND_ALERTS=true
                shift
                ;;
            --nagios)
                NAGIOS_OUTPUT=true
                shift
                ;;
            --prometheus-port)
                PROMETHEUS_PORT="$2"
                shift 2
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            check|services|dependencies|performance|network|storage|monitoring|continuous)
                COMMAND="$1"
                if [[ "$1" == "continuous" ]]; then
                    CONTINUOUS_MODE=true
                fi
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# Utility functions
is_verbose() {
    [[ "$VERBOSE" == true ]]
}

is_quiet() {
    [[ "$QUIET" == true ]]
}

verbose_log() {
    if is_verbose; then
        log_info "$1"
    fi
}

# Set health result
set_health_result() {
    local service="$1"
    local status="$2"
    local details="${3:-}"
    
    HEALTH_RESULTS["$service"]="$status"
    HEALTH_DETAILS["$service"]="$details"
}

# Get overall health status
get_overall_status() {
    local critical_count=0
    local warning_count=0
    
    for service in "${!HEALTH_RESULTS[@]}"; do
        case "${HEALTH_RESULTS[$service]}" in
            "CRITICAL"|"DOWN"|"FAILED")
                critical_count=$((critical_count + 1))
                ;;
            "WARNING"|"DEGRADED")
                warning_count=$((warning_count + 1))
                ;;
        esac
    done
    
    if [[ $critical_count -gt 0 ]]; then
        echo "CRITICAL"
        return 2
    elif [[ $warning_count -gt 0 ]]; then
        echo "WARNING"
        return 1
    else
        echo "OK"
        return 0
    fi
}

# Check Docker service health
check_docker_service() {
    local service_name="$1"
    local container_name="${COMPOSE_PROJECT_NAME}_${service_name}_1"
    
    verbose_log "Checking Docker service: $service_name"
    
    # Check if container exists and is running
    if ! docker ps --filter "name=$container_name" --filter "status=running" | grep -q "$service_name"; then
        set_health_result "$service_name" "CRITICAL" "Container not running"
        return 1
    fi
    
    # Check container health status
    local health_status=$(docker inspect --format='{{.State.Health.Status}}' "$container_name" 2>/dev/null || echo "unknown")
    
    case "$health_status" in
        "healthy")
            set_health_result "$service_name" "OK" "Container healthy"
            return 0
            ;;
        "unhealthy")
            set_health_result "$service_name" "CRITICAL" "Container unhealthy"
            return 1
            ;;
        "starting")
            set_health_result "$service_name" "WARNING" "Container starting"
            return 1
            ;;
        *)
            # Check if container is running but no health check defined
            local container_state=$(docker inspect --format='{{.State.Status}}' "$container_name" 2>/dev/null || echo "unknown")
            if [[ "$container_state" == "running" ]]; then
                set_health_result "$service_name" "OK" "Container running (no health check)"
                return 0
            else
                set_health_result "$service_name" "CRITICAL" "Container state: $container_state"
                return 1
            fi
            ;;
    esac
}

# Check HTTP endpoint health
check_http_endpoint() {
    local name="$1"
    local url="$2"
    local expected_status="${3:-200}"
    local timeout="${4:-10}"
    
    verbose_log "Checking HTTP endpoint: $name at $url"
    
    local response_code
    local response_time
    
    if response_code=$(curl -s -o /dev/null -w "%{http_code}" --max-time "$timeout" "$url" 2>/dev/null); then
        response_time=$(curl -s -o /dev/null -w "%{time_total}" --max-time "$timeout" "$url" 2>/dev/null)
        
        if [[ "$response_code" == "$expected_status" ]]; then
            set_health_result "$name" "OK" "HTTP $response_code (${response_time}s)"
            PERFORMANCE_METRICS["${name}_response_time"]="$response_time"
            return 0
        else
            set_health_result "$name" "WARNING" "HTTP $response_code (expected $expected_status)"
            return 1
        fi
    else
        set_health_result "$name" "CRITICAL" "Connection failed"
        return 1
    fi
}

# Check database connectivity
check_database() {
    local service="postgres"
    
    verbose_log "Checking PostgreSQL database connectivity"
    
    # First check if container is running
    if ! check_docker_service "$service"; then
        return 1
    fi
    
    # Check database connectivity
    local container_name="${COMPOSE_PROJECT_NAME}_postgres_1"
    
    if docker exec "$container_name" pg_isready -U "${POSTGRES_USER:-mcp_user}" -d "${POSTGRES_DB:-mcp_memory}" >/dev/null 2>&1; then
        # Get connection count
        local connections=$(docker exec "$container_name" psql -U "${POSTGRES_USER:-mcp_user}" -d "${POSTGRES_DB:-mcp_memory}" -t -c "SELECT count(*) FROM pg_stat_activity;" 2>/dev/null | xargs)
        
        set_health_result "postgres_connectivity" "OK" "Connected (${connections} active connections)"
        PERFORMANCE_METRICS["postgres_connections"]="$connections"
        return 0
    else
        set_health_result "postgres_connectivity" "CRITICAL" "Database connection failed"
        return 1
    fi
}

# Check Qdrant vector database
check_qdrant() {
    local service="qdrant"
    
    verbose_log "Checking Qdrant vector database"
    
    # Check container health
    if ! check_docker_service "$service"; then
        return 1
    fi
    
    # Check API endpoint
    local qdrant_port="${QDRANT_HOST_PORT:-6333}"
    check_http_endpoint "qdrant_api" "http://localhost:$qdrant_port/health" "200" 10
}

# Check Redis cache
check_redis() {
    local service="redis"
    
    verbose_log "Checking Redis cache"
    
    # Check container health
    if ! check_docker_service "$service"; then
        return 1
    fi
    
    # Check Redis connectivity
    local container_name="${COMPOSE_PROJECT_NAME}_redis_1"
    
    if docker exec "$container_name" redis-cli ping 2>/dev/null | grep -q "PONG"; then
        # Get memory usage
        local memory_usage=$(docker exec "$container_name" redis-cli info memory 2>/dev/null | grep "used_memory_human:" | cut -d: -f2 | tr -d '\r')
        
        set_health_result "redis_connectivity" "OK" "Connected (memory: ${memory_usage})"
        return 0
    else
        set_health_result "redis_connectivity" "CRITICAL" "Redis connection failed"
        return 1
    fi
}

# Check MCP Memory Server application
check_mcp_server() {
    local service="mcp-memory-server"
    
    verbose_log "Checking MCP Memory Server application"
    
    # Check container health
    if ! check_docker_service "$service"; then
        return 1
    fi
    
    # Check health endpoint
    local mcp_port="${MCP_HOST_PORT:-9080}"
    check_http_endpoint "mcp_health" "http://localhost:$mcp_port/health" "200" 15
    
    # Check API endpoint
    check_http_endpoint "mcp_api" "http://localhost:$mcp_port/docs" "200" 10
}

# Check Nginx reverse proxy
check_nginx() {
    local service="nginx"
    
    verbose_log "Checking Nginx reverse proxy"
    
    # Check container health
    if ! check_docker_service "$service"; then
        return 1
    fi
    
    # Check HTTP endpoint
    local nginx_port="${NGINX_HTTP_PORT:-80}"
    check_http_endpoint "nginx_proxy" "http://localhost:$nginx_port/health" "200" 10
}

# Check system resources
check_system_resources() {
    verbose_log "Checking system resources"
    
    # Check disk usage
    local disk_usage=$(df "${PROJECT_DIR}" | awk 'NR==2 {print $5}' | sed 's/%//')
    if [[ $disk_usage -gt 90 ]]; then
        set_health_result "disk_usage" "CRITICAL" "${disk_usage}% used"
    elif [[ $disk_usage -gt 80 ]]; then
        set_health_result "disk_usage" "WARNING" "${disk_usage}% used"
    else
        set_health_result "disk_usage" "OK" "${disk_usage}% used"
    fi
    PERFORMANCE_METRICS["disk_usage_percent"]="$disk_usage"
    
    # Check memory usage
    local memory_usage=$(free | awk 'NR==2{printf "%.1f", $3*100/$2}')
    if (( $(echo "$memory_usage > 90" | bc -l) )); then
        set_health_result "memory_usage" "CRITICAL" "${memory_usage}% used"
    elif (( $(echo "$memory_usage > 80" | bc -l) )); then
        set_health_result "memory_usage" "WARNING" "${memory_usage}% used"
    else
        set_health_result "memory_usage" "OK" "${memory_usage}% used"
    fi
    PERFORMANCE_METRICS["memory_usage_percent"]="$memory_usage"
    
    # Check load average
    local load_avg=$(uptime | awk -F'load average:' '{print $2}' | awk '{print $1}' | sed 's/,//')
    local cpu_count=$(nproc)
    local load_percent=$(echo "scale=1; $load_avg * 100 / $cpu_count" | bc)
    
    if (( $(echo "$load_percent > 90" | bc -l) )); then
        set_health_result "cpu_load" "CRITICAL" "${load_avg} (${load_percent}%)"
    elif (( $(echo "$load_percent > 70" | bc -l) )); then
        set_health_result "cpu_load" "WARNING" "${load_avg} (${load_percent}%)"
    else
        set_health_result "cpu_load" "OK" "${load_avg} (${load_percent}%)"
    fi
    PERFORMANCE_METRICS["cpu_load_percent"]="$load_percent"
}

# Check external dependencies
check_external_dependencies() {
    verbose_log "Checking external dependencies"
    
    # Check OpenAI API
    if [[ -n "${OPENAI_API_KEY:-}" ]]; then
        if curl -s --max-time 10 -H "Authorization: Bearer $OPENAI_API_KEY" \
           "https://api.openai.com/v1/models" > /dev/null 2>&1; then
            set_health_result "openai_api" "OK" "API accessible"
        else
            set_health_result "openai_api" "WARNING" "API not accessible"
        fi
    else
        set_health_result "openai_api" "WARNING" "API key not configured"
    fi
    
    # Check DNS resolution
    if nslookup google.com > /dev/null 2>&1; then
        set_health_result "dns_resolution" "OK" "DNS working"
    else
        set_health_result "dns_resolution" "CRITICAL" "DNS resolution failed"
    fi
    
    # Check internet connectivity
    if curl -s --max-time 10 http://www.google.com > /dev/null 2>&1; then
        set_health_result "internet_connectivity" "OK" "Internet accessible"
    else
        set_health_result "internet_connectivity" "WARNING" "Internet not accessible"
    fi
}

# Check monitoring stack
check_monitoring_stack() {
    verbose_log "Checking monitoring stack"
    
    # Check Prometheus
    local prometheus_port="${PROMETHEUS_PORT:-9090}"
    if curl -s --max-time 10 "http://localhost:$prometheus_port/-/healthy" > /dev/null 2>&1; then
        set_health_result "prometheus" "OK" "Prometheus healthy"
    else
        set_health_result "prometheus" "WARNING" "Prometheus not accessible"
    fi
    
    # Check Grafana
    local grafana_port="${GRAFANA_PORT:-3000}"
    if curl -s --max-time 10 "http://localhost:$grafana_port/api/health" > /dev/null 2>&1; then
        set_health_result "grafana" "OK" "Grafana healthy"
    else
        set_health_result "grafana" "WARNING" "Grafana not accessible"
    fi
}

# Run service health checks
check_services() {
    log_info "Checking service health..."
    
    # Core services
    check_mcp_server
    check_database
    check_qdrant
    check_redis
    check_nginx
    
    verbose_log "Service health checks completed"
}

# Run dependency checks
check_dependencies() {
    log_info "Checking external dependencies..."
    
    check_external_dependencies
    
    verbose_log "Dependency checks completed"
}

# Run performance checks
check_performance() {
    log_info "Checking performance metrics..."
    
    check_system_resources
    
    verbose_log "Performance checks completed"
}

# Output results in text format
output_text_format() {
    echo
    echo "============================================"
    echo "MCP Memory Server Health Check Report"
    echo "============================================"
    echo "Timestamp: $(date)"
    echo "Overall Status: $(get_overall_status)"
    echo
    
    echo "Service Health:"
    echo "---------------"
    for service in "${!HEALTH_RESULTS[@]}"; do
        local status="${HEALTH_RESULTS[$service]}"
        local details="${HEALTH_DETAILS[$service]}"
        
        case "$status" in
            "OK") echo -e "  ${GREEN}✓${NC} $service: $status - $details" ;;
            "WARNING"|"DEGRADED") echo -e "  ${YELLOW}⚠${NC} $service: $status - $details" ;;
            "CRITICAL"|"DOWN"|"FAILED") echo -e "  ${RED}✗${NC} $service: $status - $details" ;;
            *) echo "  ? $service: $status - $details" ;;
        esac
    done
    
    if [[ ${#PERFORMANCE_METRICS[@]} -gt 0 ]]; then
        echo
        echo "Performance Metrics:"
        echo "-------------------"
        for metric in "${!PERFORMANCE_METRICS[@]}"; do
            echo "  $metric: ${PERFORMANCE_METRICS[$metric]}"
        done
    fi
    
    echo
}

# Output results in JSON format
output_json_format() {
    local overall_status
    overall_status=$(get_overall_status)
    
    echo "{"
    echo "  \"timestamp\": \"$(date -Iseconds)\","
    echo "  \"overall_status\": \"$overall_status\","
    echo "  \"services\": {"
    
    local first=true
    for service in "${!HEALTH_RESULTS[@]}"; do
        if [[ "$first" == true ]]; then
            first=false
        else
            echo ","
        fi
        
        echo -n "    \"$service\": {"
        echo -n "\"status\": \"${HEALTH_RESULTS[$service]}\", "
        echo -n "\"details\": \"${HEALTH_DETAILS[$service]}\""
        echo -n "}"
    done
    
    echo
    echo "  },"
    echo "  \"metrics\": {"
    
    first=true
    for metric in "${!PERFORMANCE_METRICS[@]}"; do
        if [[ "$first" == true ]]; then
            first=false
        else
            echo ","
        fi
        echo -n "    \"$metric\": ${PERFORMANCE_METRICS[$metric]}"
    done
    
    echo
    echo "  }"
    echo "}"
}

# Output results in Prometheus format
output_prometheus_format() {
    echo "# HELP mcp_memory_service_health Health status of MCP Memory Server services"
    echo "# TYPE mcp_memory_service_health gauge"
    
    for service in "${!HEALTH_RESULTS[@]}"; do
        local status="${HEALTH_RESULTS[$service]}"
        local value
        case "$status" in
            "OK") value=1 ;;
            "WARNING"|"DEGRADED") value=0.5 ;;
            *) value=0 ;;
        esac
        echo "mcp_memory_service_health{service=\"$service\",status=\"$status\"} $value"
    done
    
    echo
    echo "# HELP mcp_memory_performance_metric Performance metrics for MCP Memory Server"
    echo "# TYPE mcp_memory_performance_metric gauge"
    
    for metric in "${!PERFORMANCE_METRICS[@]}"; do
        echo "mcp_memory_performance_metric{metric=\"$metric\"} ${PERFORMANCE_METRICS[$metric]}"
    done
}

# Output results in Nagios format
output_nagios_format() {
    local overall_status
    overall_status=$(get_overall_status)
    local exit_code=$?
    
    local status_text
    case "$overall_status" in
        "OK") status_text="OK" ;;
        "WARNING") status_text="WARNING" ;;
        "CRITICAL") status_text="CRITICAL" ;;
        *) status_text="UNKNOWN"; exit_code=3 ;;
    esac
    
    local failed_services=()
    local warning_services=()
    
    for service in "${!HEALTH_RESULTS[@]}"; do
        case "${HEALTH_RESULTS[$service]}" in
            "CRITICAL"|"DOWN"|"FAILED")
                failed_services+=("$service")
                ;;
            "WARNING"|"DEGRADED")
                warning_services+=("$service")
                ;;
        esac
    done
    
    local message="MCP Memory Server $status_text"
    
    if [[ ${#failed_services[@]} -gt 0 ]]; then
        message="$message - Failed: ${failed_services[*]}"
    fi
    
    if [[ ${#warning_services[@]} -gt 0 ]]; then
        message="$message - Warning: ${warning_services[*]}"
    fi
    
    echo "$message"
    exit $exit_code
}

# Send webhook notification
send_webhook() {
    if [[ -n "$WEBHOOK_URL" ]]; then
        local payload
        payload=$(output_json_format)
        
        curl -s -X POST -H "Content-Type: application/json" \
             -d "$payload" "$WEBHOOK_URL" > /dev/null 2>&1 || true
    fi
}

# Run complete health check
run_complete_check() {
    case "$COMMAND" in
        services)
            check_services
            ;;
        dependencies)
            check_dependencies
            ;;
        performance)
            check_performance
            ;;
        network)
            check_dependencies  # Network checks are part of dependencies
            ;;
        storage)
            check_system_resources  # Storage checks are part of system resources
            ;;
        monitoring)
            check_monitoring_stack
            ;;
        check|*)
            check_services
            check_dependencies
            check_performance
            check_monitoring_stack
            ;;
    esac
}

# Continuous monitoring mode
run_continuous_monitoring() {
    log_info "Starting continuous health monitoring (interval: ${HEALTH_CHECK_TIMEOUT}s)"
    
    while true; do
        # Clear previous results
        HEALTH_RESULTS=()
        HEALTH_DETAILS=()
        PERFORMANCE_METRICS=()
        
        # Run health checks
        run_complete_check
        
        # Output results
        if [[ "$NAGIOS_OUTPUT" == true ]]; then
            output_nagios_format
        elif [[ "$OUTPUT_FORMAT" == "json" ]]; then
            output_json_format
        elif [[ "$OUTPUT_FORMAT" == "prometheus" ]]; then
            output_prometheus_format
        else
            output_text_format
        fi
        
        # Send webhook if configured
        send_webhook
        
        # Wait for next check
        sleep "$HEALTH_CHECK_TIMEOUT"
    done
}

# Main execution
main() {
    parse_args "$@"
    
    # Create log directory
    mkdir -p "$LOG_DIR"
    
    if is_verbose; then
        log_info "Starting health check with command: $COMMAND"
    fi
    
    # Run health checks
    if [[ "$CONTINUOUS_MODE" == true ]]; then
        run_continuous_monitoring
    else
        run_complete_check
        
        # Output results
        if [[ "$NAGIOS_OUTPUT" == true ]]; then
            output_nagios_format
        elif [[ "$OUTPUT_FORMAT" == "json" ]]; then
            output_json_format
        elif [[ "$OUTPUT_FORMAT" == "prometheus" ]]; then
            output_prometheus_format
        else
            if ! is_quiet; then
                output_text_format
            fi
        fi
        
        # Send webhook if configured
        send_webhook
        
        # Exit with appropriate code
        get_overall_status > /dev/null
        exit $?
    fi
}

# Execute main function with all arguments
main "$@"