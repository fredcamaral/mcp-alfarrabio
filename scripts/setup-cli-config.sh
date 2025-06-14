#!/bin/bash
# CLI configuration setup for lmmc - reads from root .env file

set -e

# Find the script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

CONFIG_DIR="$HOME/.lmmc"
CONFIG_FILE="$CONFIG_DIR/config.yaml"
ENV_FILE="${ENV_FILE:-$PROJECT_ROOT/.env}"

# Create config directory
mkdir -p "$CONFIG_DIR"

# Function to read value from .env file
read_env_var() {
    local var_name=$1
    local default_value=$2
    
    if [ -f "$ENV_FILE" ]; then
        # Extract value from .env, handling comments and spaces
        local value=$(grep "^${var_name}=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2- | sed 's/^[[:space:]]*//;s/[[:space:]]*$//' | sed 's/^"//;s/"$//')
        if [ -n "$value" ]; then
            echo "$value"
        else
            echo "$default_value"
        fi
    else
        echo "$default_value"
    fi
}

# Read configuration values from .env
SERVER_URL=$(read_env_var "LMMC_SERVER_URL" "")
SERVER_TIMEOUT=$(read_env_var "LMMC_SERVER_TIMEOUT" "30")
LOG_LEVEL=$(read_env_var "LMMC_LOG_LEVEL" "warn")
LOG_FORMAT=$(read_env_var "LMMC_LOG_FORMAT" "text")
OUTPUT_FORMAT=$(read_env_var "LMMC_OUTPUT_FORMAT" "table")
COLOR_SCHEME=$(read_env_var "LMMC_COLOR_SCHEME" "auto")
PAGE_SIZE=$(read_env_var "LMMC_PAGE_SIZE" "20")
EDITOR=$(read_env_var "LMMC_EDITOR" "")

# Use server URL directly if provided, otherwise default to empty (offline mode)
FULL_SERVER_URL="$SERVER_URL"

# Create config file if it doesn't exist
if [ ! -f "$CONFIG_FILE" ]; then
    cat > "$CONFIG_FILE" << EOF
# LMMC CLI Configuration
# Generated from $ENV_FILE

# Server connection (optional - CLI works standalone) 
server:
  url: "$FULL_SERVER_URL"  # Empty for offline mode
  version: "v1"
  timeout: $SERVER_TIMEOUT

# CLI behavior
cli:
  default_repository: ""
  output_format: "$OUTPUT_FORMAT"
  auto_complete: true
  color_scheme: "$COLOR_SCHEME" 
  page_size: $PAGE_SIZE
  editor: "$EDITOR"

# Storage & caching
storage:
  cache_enabled: true
  cache_ttl: 300
  backup_count: 3

# Logging
logging:
  level: "$LOG_LEVEL"
  format: "$LOG_FORMAT"
  file: ""
EOF
    echo "✓ Created CLI configuration: $CONFIG_FILE"
    echo "  Server URL: ${FULL_SERVER_URL:-offline mode}"
    echo "  Log level: $LOG_LEVEL"
else
    echo "✓ CLI configuration already exists: $CONFIG_FILE"
    echo "  To regenerate, delete $CONFIG_FILE and run this script again"
fi