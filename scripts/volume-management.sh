#!/bin/bash

# MCP Memory Volume Management Script
# CRITICAL: This script helps manage persistent data volumes
# NEVER DELETE the volumes marked as NEVER_DELETE!

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Volume names
CHROMA_VOLUME="mcp_memory_chroma_vector_db_NEVER_DELETE"
DATA_VOLUME="mcp_memory_app_data_NEVER_DELETE"
BACKUP_VOLUME="mcp_memory_backups_NEVER_DELETE"
LOGS_VOLUME="mcp_memory_logs"

echo -e "${BOLD}🗄️  MCP Memory Volume Management${NC}"
echo "========================================"

# Function to show volume status
show_status() {
    echo -e "\n${BLUE}📊 Volume Status:${NC}"
    echo "-------------------"
    
    for volume in "$CHROMA_VOLUME" "$DATA_VOLUME" "$BACKUP_VOLUME" "$LOGS_VOLUME"; do
        if docker volume inspect "$volume" >/dev/null 2>&1; then
            size=$(docker run --rm -v "$volume":/data alpine sh -c "du -sh /data" 2>/dev/null | cut -f1 || echo "Unknown")
            echo -e "✅ ${GREEN}$volume${NC} - Size: $size"
        else
            echo -e "❌ ${RED}$volume${NC} - Not found"
        fi
    done
}

# Function to backup volumes
backup_volumes() {
    echo -e "\n${YELLOW}💾 Creating Volume Backups...${NC}"
    
    timestamp=$(date +"%Y%m%d_%H%M%S")
    backup_dir="./volume_backups_$timestamp"
    mkdir -p "$backup_dir"
    
    echo "📁 Backup directory: $backup_dir"
    
    # Backup critical volumes
    for volume in "$CHROMA_VOLUME" "$DATA_VOLUME" "$BACKUP_VOLUME"; do
        if docker volume inspect "$volume" >/dev/null 2>&1; then
            echo "🔄 Backing up $volume..."
            docker run --rm -v "$volume":/source -v "$(pwd)/$backup_dir":/backup alpine \
                tar czf "/backup/${volume}_${timestamp}.tar.gz" -C /source .
            echo -e "✅ ${GREEN}Backup created: ${backup_dir}/${volume}_${timestamp}.tar.gz${NC}"
        else
            echo -e "⚠️  ${YELLOW}Volume $volume not found, skipping${NC}"
        fi
    done
    
    echo -e "\n🎉 ${GREEN}Backup completed!${NC}"
    echo -e "📁 Backups saved to: ${BOLD}$backup_dir${NC}"
}

# Function to restore volumes (DANGEROUS!)
restore_volumes() {
    echo -e "\n${RED}⚠️  DANGER: Volume Restoration${NC}"
    echo -e "${RED}This will OVERWRITE existing data!${NC}"
    echo -e "Only proceed if you want to restore from backup."
    echo ""
    read -p "Are you absolutely sure? Type 'YES I UNDERSTAND' to continue: " confirm
    
    if [ "$confirm" != "YES I UNDERSTAND" ]; then
        echo "❌ Restoration cancelled"
        return 1
    fi
    
    echo "📁 Available backup directories:"
    ls -la | grep "volume_backups_" || echo "No backup directories found"
    echo ""
    read -p "Enter backup directory name (e.g., volume_backups_20250524_234500): " backup_dir
    
    if [ ! -d "$backup_dir" ]; then
        echo -e "❌ ${RED}Backup directory not found!${NC}"
        return 1
    fi
    
    # Stop containers first
    echo "🛑 Stopping containers..."
    docker-compose down
    
    # Restore each volume
    for volume in "$CHROMA_VOLUME" "$DATA_VOLUME" "$BACKUP_VOLUME"; do
        backup_file="${backup_dir}/${volume}_*.tar.gz"
        if ls $backup_file 1> /dev/null 2>&1; then
            echo "🔄 Restoring $volume..."
            docker volume rm "$volume" 2>/dev/null || true
            docker volume create "$volume"
            docker run --rm -v "$volume":/target -v "$(pwd)/$backup_dir":/backup alpine \
                tar xzf "/backup/$(basename $backup_file)" -C /target
            echo -e "✅ ${GREEN}Restored $volume${NC}"
        else
            echo -e "⚠️  ${YELLOW}No backup found for $volume${NC}"
        fi
    done
    
    echo -e "\n🎉 ${GREEN}Restoration completed!${NC}"
    echo "🚀 Starting containers..."
    docker-compose up -d
}

# Function to list volume contents
list_contents() {
    echo -e "\n${BLUE}📋 Volume Contents:${NC}"
    
    for volume in "$CHROMA_VOLUME" "$DATA_VOLUME" "$BACKUP_VOLUME"; do
        if docker volume inspect "$volume" >/dev/null 2>&1; then
            echo -e "\n${BOLD}📁 $volume:${NC}"
            docker run --rm -v "$volume":/data alpine find /data -type f | head -20
        fi
    done
}

# Function to check container status
check_containers() {
    echo -e "\n${BLUE}🐳 Container Status:${NC}"
    echo "-------------------"
    
    containers=("mcp-memory-server" "mcp-chroma")
    for container in "${containers[@]}"; do
        if docker ps -q -f name="$container" | grep -q .; then
            echo -e "✅ ${GREEN}$container${NC} - Running"
        elif docker ps -aq -f name="$container" | grep -q .; then
            echo -e "⚠️  ${YELLOW}$container${NC} - Stopped"
        else
            echo -e "❌ ${RED}$container${NC} - Not found"
        fi
    done
}

# Main menu
show_menu() {
    echo -e "\n${BOLD}📋 Available Commands:${NC}"
    echo "1) show-status    - Show volume status and sizes"
    echo "2) backup         - Create backup of all volumes"
    echo "3) restore        - Restore volumes from backup (DANGEROUS!)"
    echo "4) list-contents  - List volume contents"
    echo "5) check-containers - Check container status"
    echo "6) help           - Show this menu"
    echo ""
    echo -e "${RED}CRITICAL VOLUMES (NEVER DELETE):${NC}"
    echo "- $CHROMA_VOLUME"
    echo "- $DATA_VOLUME"
    echo "- $BACKUP_VOLUME"
}

# Handle command line arguments
case "${1:-help}" in
    "show-status"|"status")
        show_status
        check_containers
        ;;
    "backup")
        backup_volumes
        ;;
    "restore")
        restore_volumes
        ;;
    "list-contents"|"list")
        list_contents
        ;;
    "check-containers"|"containers")
        check_containers
        ;;
    "help"|"--help"|"-h")
        show_menu
        ;;
    *)
        echo -e "${RED}❌ Unknown command: $1${NC}"
        show_menu
        exit 1
        ;;
esac

echo -e "\n${GREEN}✅ Command completed!${NC}"