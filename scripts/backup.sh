#!/bin/bash

# Comprehensive Backup Script for MCP Memory Server
# Handles database backups, data backups, and disaster recovery

set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
BACKUP_DIR="${PROJECT_DIR}/backups"
LOG_DIR="${PROJECT_DIR}/logs"
COMPOSE_PROJECT_NAME="lerian-mcp-memory"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
    echo "$(date '+%Y-%m-%d %H:%M:%S') [INFO] $1" >> "${LOG_DIR}/backup.log"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
    echo "$(date '+%Y-%m-%d %H:%M:%S') [SUCCESS] $1" >> "${LOG_DIR}/backup.log"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
    echo "$(date '+%Y-%m-%d %H:%M:%S') [WARNING] $1" >> "${LOG_DIR}/backup.log"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
    echo "$(date '+%Y-%m-%d %H:%M:%S') [ERROR] $1" >> "${LOG_DIR}/backup.log"
}

# Show help
show_help() {
    cat << EOF
Comprehensive Backup Script for MCP Memory Server

Usage: $0 [OPTIONS] COMMAND

Commands:
    create          Create a full backup
    restore         Restore from backup
    list            List available backups
    cleanup         Clean up old backups
    verify          Verify backup integrity
    schedule        Setup automated backup schedule
    monitor         Monitor backup health

Options:
    -t, --type TYPE       Backup type: full, incremental, database, config (default: full)
    -f, --file FILE       Backup file for restore operations
    -d, --destination     Backup destination (local, s3, gcs)
    -r, --retention DAYS  Retention period in days (default: 30)
    -c, --compress        Enable compression (default: true)
    -e, --encrypt         Enable encryption
    -v, --verify          Verify backup after creation
    --exclude PATTERN     Exclude files matching pattern
    --include-logs        Include log files in backup
    --s3-bucket BUCKET    S3 bucket for remote backups
    --s3-prefix PREFIX    S3 prefix for backup files
    -h, --help            Show this help message

Examples:
    $0 create                              # Create full backup
    $0 create -t database                  # Database only backup
    $0 restore -f backup_20240101_120000   # Restore specific backup
    $0 cleanup -r 7                       # Keep only 7 days of backups
    $0 create -d s3 --s3-bucket my-backup # Backup to S3

EOF
}

# Default values
BACKUP_TYPE="full"
DESTINATION="local"
RETENTION_DAYS=30
ENABLE_COMPRESSION=true
ENABLE_ENCRYPTION=false
VERIFY_BACKUP=false
INCLUDE_LOGS=false
EXCLUDE_PATTERNS=()
S3_BUCKET=""
S3_PREFIX="mcp-memory-backups"
BACKUP_FILE=""

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -t|--type)
                BACKUP_TYPE="$2"
                shift 2
                ;;
            -f|--file)
                BACKUP_FILE="$2"
                shift 2
                ;;
            -d|--destination)
                DESTINATION="$2"
                shift 2
                ;;
            -r|--retention)
                RETENTION_DAYS="$2"
                shift 2
                ;;
            -c|--compress)
                ENABLE_COMPRESSION=true
                shift
                ;;
            -e|--encrypt)
                ENABLE_ENCRYPTION=true
                shift
                ;;
            -v|--verify)
                VERIFY_BACKUP=true
                shift
                ;;
            --exclude)
                EXCLUDE_PATTERNS+=("$2")
                shift 2
                ;;
            --include-logs)
                INCLUDE_LOGS=true
                shift
                ;;
            --s3-bucket)
                S3_BUCKET="$2"
                shift 2
                ;;
            --s3-prefix)
                S3_PREFIX="$2"
                shift 2
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            create|restore|list|cleanup|verify|schedule|monitor)
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
    log_info "Checking backup prerequisites..."
    
    # Create backup directory
    mkdir -p "$BACKUP_DIR" "$LOG_DIR"
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi
    
    # Check for AWS CLI if using S3
    if [[ "$DESTINATION" == "s3" ]]; then
        if ! command -v aws &> /dev/null; then
            log_error "AWS CLI is not installed but S3 destination specified"
            exit 1
        fi
        
        if [[ -z "$S3_BUCKET" ]]; then
            log_error "S3 bucket must be specified for S3 destination"
            exit 1
        fi
    fi
    
    # Check encryption tools if needed
    if [[ "$ENABLE_ENCRYPTION" == true ]]; then
        if ! command -v gpg &> /dev/null; then
            log_error "GPG is not installed but encryption is enabled"
            exit 1
        fi
    fi
    
    log_success "Prerequisites check passed"
}

# Load environment variables
load_environment() {
    local env_file="${PROJECT_DIR}/.env.production"
    
    if [[ -f "$env_file" ]]; then
        log_info "Loading environment from $env_file"
        set -a
        source "$env_file"
        set +a
    else
        log_warning "Environment file not found: $env_file"
    fi
}

# Create database backup
backup_database() {
    local backup_path="$1"
    local timestamp="$2"
    
    log_info "Creating database backup..."
    
    # PostgreSQL backup
    if docker ps --filter "name=${COMPOSE_PROJECT_NAME}_postgres" --filter "status=running" | grep -q postgres; then
        log_info "Backing up PostgreSQL database..."
        
        local pg_container="${COMPOSE_PROJECT_NAME}_postgres_1"
        local db_backup_file="${backup_path}/postgres_${timestamp}.sql"
        
        # Create database dump
        docker exec "$pg_container" pg_dump \
            -U "${POSTGRES_USER:-mcp_user}" \
            -d "${POSTGRES_DB:-mcp_memory}" \
            --verbose \
            --format=custom \
            --no-owner \
            --no-privileges > "${db_backup_file}.dump"
        
        # Also create plain SQL for easier restore
        docker exec "$pg_container" pg_dump \
            -U "${POSTGRES_USER:-mcp_user}" \
            -d "${POSTGRES_DB:-mcp_memory}" \
            --verbose \
            --no-owner \
            --no-privileges > "$db_backup_file"
        
        # Create schema-only backup
        docker exec "$pg_container" pg_dump \
            -U "${POSTGRES_USER:-mcp_user}" \
            -d "${POSTGRES_DB:-mcp_memory}" \
            --schema-only \
            --no-owner \
            --no-privileges > "${backup_path}/postgres_schema_${timestamp}.sql"
        
        log_success "PostgreSQL backup completed"
    else
        log_warning "PostgreSQL container not running, skipping database backup"
    fi
}

# Create data backup
backup_data() {
    local backup_path="$1"
    local timestamp="$2"
    
    log_info "Creating data backup..."
    
    local data_dir="${PROJECT_DIR}/data"
    
    if [[ ! -d "$data_dir" ]]; then
        log_warning "Data directory not found: $data_dir"
        return 0
    fi
    
    # Build rsync exclude options
    local rsync_excludes=()
    rsync_excludes+=(--exclude='*.log')
    rsync_excludes+=(--exclude='*.log.*')
    rsync_excludes+=(--exclude='tmp/')
    rsync_excludes+=(--exclude='temp/')
    
    for pattern in "${EXCLUDE_PATTERNS[@]}"; do
        rsync_excludes+=(--exclude="$pattern")
    done
    
    if [[ "$INCLUDE_LOGS" != true ]]; then
        rsync_excludes+=(--exclude='logs/')
        rsync_excludes+=(--exclude='*.log')
    fi
    
    # Create data backup
    log_info "Backing up data directory..."
    rsync -av "${rsync_excludes[@]}" "$data_dir/" "${backup_path}/data/"
    
    # Create file manifest
    log_info "Creating file manifest..."
    find "${backup_path}/data" -type f -exec sha256sum {} \; > "${backup_path}/data_manifest_${timestamp}.txt"
    
    log_success "Data backup completed"
}

# Create configuration backup
backup_configs() {
    local backup_path="$1"
    local timestamp="$2"
    
    log_info "Creating configuration backup..."
    
    # Backup configuration files
    local config_dirs=(
        "configs"
        "docker-compose.production.yml"
        "docker-compose.monitoring.yml"
        ".env.production"
        ".env.example"
    )
    
    mkdir -p "${backup_path}/configs"
    
    for item in "${config_dirs[@]}"; do
        local source_path="${PROJECT_DIR}/$item"
        if [[ -e "$source_path" ]]; then
            if [[ -d "$source_path" ]]; then
                cp -r "$source_path" "${backup_path}/configs/"
            else
                cp "$source_path" "${backup_path}/configs/"
            fi
            log_info "Backed up: $item"
        else
            log_warning "Configuration not found: $item"
        fi
    done
    
    log_success "Configuration backup completed"
}

# Create application backup
backup_application() {
    local backup_path="$1"
    local timestamp="$2"
    
    log_info "Creating application backup..."
    
    # Export Docker images
    log_info "Exporting Docker images..."
    
    local images=()
    images+=($(docker images --filter "reference=ghcr.io/lerianstudio/lerian-mcp-memory" --format "{{.Repository}}:{{.Tag}}"))
    
    if [[ ${#images[@]} -gt 0 ]]; then
        mkdir -p "${backup_path}/images"
        
        for image in "${images[@]}"; do
            local image_file="${backup_path}/images/$(echo "$image" | sed 's/[\/:]/_/g').tar"
            log_info "Exporting image: $image"
            docker save "$image" > "$image_file"
        done
        
        log_success "Docker images exported"
    else
        log_warning "No application images found to backup"
    fi
}

# Create full backup
create_backup() {
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_name="backup_${timestamp}"
    local backup_path="${BACKUP_DIR}/${backup_name}"
    
    log_info "Creating $BACKUP_TYPE backup: $backup_name"
    
    # Create backup directory
    mkdir -p "$backup_path"
    
    # Create metadata
    cat > "${backup_path}/metadata.json" << EOF
{
    "backup_id": "$backup_name",
    "backup_type": "$BACKUP_TYPE",
    "timestamp": "$(date -Iseconds)",
    "hostname": "$(hostname)",
    "project_dir": "$PROJECT_DIR",
    "compression": $ENABLE_COMPRESSION,
    "encryption": $ENABLE_ENCRYPTION,
    "include_logs": $INCLUDE_LOGS,
    "retention_days": $RETENTION_DAYS
}
EOF
    
    # Perform backup based on type
    case "$BACKUP_TYPE" in
        full)
            backup_database "$backup_path" "$timestamp"
            backup_data "$backup_path" "$timestamp"
            backup_configs "$backup_path" "$timestamp"
            backup_application "$backup_path" "$timestamp"
            ;;
        database)
            backup_database "$backup_path" "$timestamp"
            ;;
        config)
            backup_configs "$backup_path" "$timestamp"
            ;;
        incremental)
            # For incremental, compare with last backup
            backup_incremental "$backup_path" "$timestamp"
            ;;
        *)
            log_error "Unknown backup type: $BACKUP_TYPE"
            exit 1
            ;;
    esac
    
    # Compress backup if enabled
    if [[ "$ENABLE_COMPRESSION" == true ]]; then
        log_info "Compressing backup..."
        tar -czf "${backup_path}.tar.gz" -C "$BACKUP_DIR" "$backup_name"
        
        # Remove uncompressed backup
        rm -rf "$backup_path"
        backup_path="${backup_path}.tar.gz"
    fi
    
    # Encrypt backup if enabled
    if [[ "$ENABLE_ENCRYPTION" == true ]]; then
        log_info "Encrypting backup..."
        gpg --symmetric --cipher-algo AES256 --compress-algo 1 --s2k-mode 3 \
            --s2k-digest-algo SHA512 --s2k-count 65011712 \
            --output "${backup_path}.gpg" "$backup_path"
        
        # Remove unencrypted backup
        rm -f "$backup_path"
        backup_path="${backup_path}.gpg"
    fi
    
    # Upload to remote destination if specified
    if [[ "$DESTINATION" != "local" ]]; then
        upload_backup "$backup_path"
    fi
    
    # Verify backup if requested
    if [[ "$VERIFY_BACKUP" == true ]]; then
        verify_backup_integrity "$backup_path"
    fi
    
    # Update latest backup reference
    echo "$(basename "$backup_path")" > "${BACKUP_DIR}/latest_backup.txt"
    
    local backup_size=$(du -h "$backup_path" | cut -f1)
    log_success "Backup completed: $(basename "$backup_path") (Size: $backup_size)"
}

# Create incremental backup
backup_incremental() {
    local backup_path="$1"
    local timestamp="$2"
    
    log_info "Creating incremental backup..."
    
    # Find last full backup
    local last_backup=""
    if [[ -f "${BACKUP_DIR}/latest_backup.txt" ]]; then
        last_backup=$(cat "${BACKUP_DIR}/latest_backup.txt")
    fi
    
    if [[ -z "$last_backup" ]]; then
        log_warning "No previous backup found, creating full backup instead"
        backup_database "$backup_path" "$timestamp"
        backup_data "$backup_path" "$timestamp"
        backup_configs "$backup_path" "$timestamp"
        return 0
    fi
    
    log_info "Creating incremental backup since: $last_backup"
    
    # Create incremental data backup using rsync
    local data_dir="${PROJECT_DIR}/data"
    if [[ -d "$data_dir" ]]; then
        # Use link-dest for incremental backup
        local last_backup_path="${BACKUP_DIR}/${last_backup%.*}"
        if [[ -d "${last_backup_path}/data" ]]; then
            rsync -av --link-dest="${last_backup_path}/data" \
                --exclude='*.log' \
                "$data_dir/" "${backup_path}/data/"
        else
            # Fallback to full data backup
            backup_data "$backup_path" "$timestamp"
        fi
    fi
    
    # Always backup database and configs for incremental
    backup_database "$backup_path" "$timestamp"
    backup_configs "$backup_path" "$timestamp"
}

# Upload backup to remote destination
upload_backup() {
    local backup_file="$1"
    
    case "$DESTINATION" in
        s3)
            log_info "Uploading backup to S3..."
            local s3_key="${S3_PREFIX}/$(basename "$backup_file")"
            
            if aws s3 cp "$backup_file" "s3://${S3_BUCKET}/${s3_key}"; then
                log_success "Backup uploaded to S3: s3://${S3_BUCKET}/${s3_key}"
                
                # Store S3 location in metadata
                echo "s3://${S3_BUCKET}/${s3_key}" > "${backup_file}.s3_location"
            else
                log_error "Failed to upload backup to S3"
                exit 1
            fi
            ;;
        gcs)
            log_info "Uploading backup to Google Cloud Storage..."
            # Implementation for GCS
            log_warning "GCS upload not implemented yet"
            ;;
        *)
            log_error "Unknown destination: $DESTINATION"
            exit 1
            ;;
    esac
}

# Verify backup integrity
verify_backup_integrity() {
    local backup_file="$1"
    
    log_info "Verifying backup integrity..."
    
    if [[ "$backup_file" == *.gpg ]]; then
        log_info "Verifying encrypted backup..."
        if gpg --batch --quiet --decrypt "$backup_file" > /dev/null 2>&1; then
            log_success "Backup encryption verification passed"
        else
            log_error "Backup encryption verification failed"
            exit 1
        fi
    elif [[ "$backup_file" == *.tar.gz ]]; then
        log_info "Verifying compressed backup..."
        if tar -tzf "$backup_file" > /dev/null 2>&1; then
            log_success "Backup compression verification passed"
        else
            log_error "Backup compression verification failed"
            exit 1
        fi
    else
        log_info "Verifying backup directory..."
        if [[ -d "$backup_file" ]]; then
            log_success "Backup directory verification passed"
        else
            log_error "Backup directory verification failed"
            exit 1
        fi
    fi
}

# List available backups
list_backups() {
    log_info "Available backups:"
    echo
    
    # Local backups
    echo "Local Backups:"
    echo "=============="
    if [[ -d "$BACKUP_DIR" ]]; then
        ls -la "$BACKUP_DIR"/backup_* 2>/dev/null | while read -r line; do
            echo "  $line"
        done
    else
        echo "  No local backups found"
    fi
    echo
    
    # Remote backups
    if [[ "$DESTINATION" == "s3" && -n "$S3_BUCKET" ]]; then
        echo "S3 Backups:"
        echo "==========="
        if command -v aws &> /dev/null; then
            aws s3 ls "s3://${S3_BUCKET}/${S3_PREFIX}/" --recursive | while read -r line; do
                echo "  $line"
            done
        else
            echo "  AWS CLI not available"
        fi
    fi
}

# Restore from backup
restore_backup() {
    if [[ -z "$BACKUP_FILE" ]]; then
        log_error "Backup file must be specified for restore"
        exit 1
    fi
    
    log_info "Restoring from backup: $BACKUP_FILE"
    
    local restore_path="${BACKUP_DIR}/restore_$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$restore_path"
    
    # Download from remote if needed
    local local_backup_file=""
    if [[ "$BACKUP_FILE" == s3://* ]]; then
        log_info "Downloading backup from S3..."
        local_backup_file="${restore_path}/$(basename "$BACKUP_FILE")"
        aws s3 cp "$BACKUP_FILE" "$local_backup_file"
    else
        local_backup_file="${BACKUP_DIR}/${BACKUP_FILE}"
    fi
    
    if [[ ! -f "$local_backup_file" ]]; then
        log_error "Backup file not found: $local_backup_file"
        exit 1
    fi
    
    # Decrypt if needed
    if [[ "$local_backup_file" == *.gpg ]]; then
        log_info "Decrypting backup..."
        local decrypted_file="${local_backup_file%.gpg}"
        gpg --batch --quiet --decrypt "$local_backup_file" > "$decrypted_file"
        local_backup_file="$decrypted_file"
    fi
    
    # Extract if compressed
    if [[ "$local_backup_file" == *.tar.gz ]]; then
        log_info "Extracting backup..."
        tar -xzf "$local_backup_file" -C "$restore_path"
        local_backup_file="${restore_path}/$(tar -tzf "$local_backup_file" | head -1 | cut -d/ -f1)"
    fi
    
    # Perform restore
    log_info "Restoring data..."
    
    # Stop services
    log_info "Stopping services for restore..."
    docker-compose -f "${PROJECT_DIR}/docker-compose.production.yml" down
    
    # Restore database
    if [[ -f "${local_backup_file}/postgres_"*.sql ]]; then
        log_info "Restoring PostgreSQL database..."
        # Start only postgres for restore
        docker-compose -f "${PROJECT_DIR}/docker-compose.production.yml" up -d postgres
        sleep 10
        
        # Restore database
        local sql_file=$(ls "${local_backup_file}"/postgres_*.sql | head -1)
        docker exec -i "${COMPOSE_PROJECT_NAME}_postgres_1" psql \
            -U "${POSTGRES_USER:-mcp_user}" \
            -d "${POSTGRES_DB:-mcp_memory}" < "$sql_file"
        
        log_success "Database restored"
    fi
    
    # Restore data
    if [[ -d "${local_backup_file}/data" ]]; then
        log_info "Restoring data directory..."
        rsync -av "${local_backup_file}/data/" "${PROJECT_DIR}/data/"
        log_success "Data directory restored"
    fi
    
    # Restore configurations
    if [[ -d "${local_backup_file}/configs" ]]; then
        log_info "Restoring configurations..."
        rsync -av "${local_backup_file}/configs/" "${PROJECT_DIR}/"
        log_success "Configurations restored"
    fi
    
    # Restart all services
    log_info "Restarting services..."
    docker-compose -f "${PROJECT_DIR}/docker-compose.production.yml" up -d
    
    # Cleanup
    rm -rf "$restore_path"
    
    log_success "Restore completed successfully"
}

# Clean up old backups
cleanup_backups() {
    log_info "Cleaning up backups older than $RETENTION_DAYS days..."
    
    # Local cleanup
    find "$BACKUP_DIR" -name "backup_*" -type f -mtime +$RETENTION_DAYS -delete
    
    # S3 cleanup
    if [[ "$DESTINATION" == "s3" && -n "$S3_BUCKET" ]]; then
        local cutoff_date=$(date -d "$RETENTION_DAYS days ago" +%Y-%m-%d)
        log_info "Cleaning up S3 backups older than $cutoff_date..."
        
        aws s3api list-objects-v2 \
            --bucket "$S3_BUCKET" \
            --prefix "$S3_PREFIX/" \
            --query "Contents[?LastModified<='$cutoff_date'].Key" \
            --output text | while read -r key; do
                if [[ -n "$key" ]]; then
                    aws s3 rm "s3://${S3_BUCKET}/${key}"
                    log_info "Deleted: s3://${S3_BUCKET}/${key}"
                fi
            done
    fi
    
    log_success "Cleanup completed"
}

# Setup backup schedule
setup_schedule() {
    log_info "Setting up backup schedule..."
    
    # Create cron job for automated backups
    local cron_schedule="0 2 * * *"  # Daily at 2 AM
    local script_path="$SCRIPT_DIR/backup.sh"
    local cron_command="$script_path create -t full -c -v >> $LOG_DIR/backup_cron.log 2>&1"
    
    # Check if cron job already exists
    if crontab -l 2>/dev/null | grep -q "$script_path"; then
        log_info "Backup cron job already exists"
    else
        # Add cron job
        (crontab -l 2>/dev/null; echo "$cron_schedule $cron_command") | crontab -
        log_success "Backup cron job added: $cron_schedule"
    fi
    
    # Create logrotate configuration
    cat > /etc/logrotate.d/mcp-memory-backup << EOF
$LOG_DIR/backup*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 644 root root
}
EOF
    
    log_success "Backup schedule setup completed"
}

# Monitor backup health
monitor_backups() {
    log_info "Monitoring backup health..."
    
    local issues=0
    
    # Check recent backups
    local recent_backup=$(find "$BACKUP_DIR" -name "backup_*" -type f -mtime -1 | wc -l)
    if [[ $recent_backup -eq 0 ]]; then
        log_warning "No recent backups found (last 24 hours)"
        issues=$((issues + 1))
    else
        log_success "Recent backup found"
    fi
    
    # Check backup integrity
    local latest_backup_file=""
    if [[ -f "${BACKUP_DIR}/latest_backup.txt" ]]; then
        latest_backup_file=$(cat "${BACKUP_DIR}/latest_backup.txt")
        local latest_backup_path="${BACKUP_DIR}/${latest_backup_file}"
        
        if [[ -f "$latest_backup_path" ]]; then
            verify_backup_integrity "$latest_backup_path"
            log_success "Latest backup integrity verified"
        else
            log_warning "Latest backup file not found: $latest_backup_path"
            issues=$((issues + 1))
        fi
    else
        log_warning "No latest backup reference found"
        issues=$((issues + 1))
    fi
    
    # Check disk space
    local backup_dir_usage=$(df "$BACKUP_DIR" | awk 'NR==2 {print $5}' | sed 's/%//')
    if [[ $backup_dir_usage -gt 80 ]]; then
        log_warning "Backup directory disk usage is high: ${backup_dir_usage}%"
        issues=$((issues + 1))
    else
        log_success "Backup directory disk usage is normal: ${backup_dir_usage}%"
    fi
    
    # Summary
    if [[ $issues -eq 0 ]]; then
        log_success "Backup monitoring: All checks passed"
        exit 0
    else
        log_error "Backup monitoring: $issues issues found"
        exit 1
    fi
}

# Main execution
main() {
    parse_args "$@"
    
    log_info "Starting backup script with command: $COMMAND"
    
    check_prerequisites
    load_environment
    
    case "$COMMAND" in
        create)
            create_backup
            ;;
        restore)
            restore_backup
            ;;
        list)
            list_backups
            ;;
        cleanup)
            cleanup_backups
            ;;
        verify)
            if [[ -n "$BACKUP_FILE" ]]; then
                verify_backup_integrity "${BACKUP_DIR}/${BACKUP_FILE}"
            else
                log_error "Backup file must be specified for verification"
                exit 1
            fi
            ;;
        schedule)
            setup_schedule
            ;;
        monitor)
            monitor_backups
            ;;
        *)
            log_error "Unknown command: $COMMAND"
            show_help
            exit 1
            ;;
    esac
    
    log_success "Backup script completed successfully"
}

# Execute main function with all arguments
main "$@"