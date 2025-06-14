#!/bin/bash

# LMMC CLI Configuration Installation Script
# This script sets up the LMMC configuration directory and files

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
CLI_NAME="lmmc"
CONFIG_DIR="$HOME/.lmmc"
CONFIG_FILE="$CONFIG_DIR/config.yaml"
ROOT_ENV_EXAMPLE="$(dirname "$0")/../../.env.example"

# Helper functions
print_header() {
    echo ""
    echo -e "${BLUE}================================================${NC}"
    echo -e "${BLUE}  LMMC CLI Configuration Setup${NC}"
    echo -e "${BLUE}================================================${NC}"
    echo ""
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${CYAN}ℹ $1${NC}"
}

ask_yes_no() {
    local question="$1"
    local default="${2:-n}"
    local prompt
    
    # Check if running in non-interactive mode
    if [[ ! -t 0 ]]; then
        # Non-interactive mode - use default
        if [[ "$default" == "y" ]]; then
            return 0
        else
            return 1
        fi
    fi
    
    if [[ "$default" == "y" ]]; then
        prompt="[Y/n]"
    else
        prompt="[y/N]"
    fi
    
    while true; do
        echo -e -n "${CYAN}$question $prompt: ${NC}"
        read -r response
        
        # Use default if empty response
        if [[ -z "$response" ]]; then
            response="$default"
        fi
        
        case "$response" in
            [Yy]|[Yy][Ee][Ss]) return 0 ;;
            [Nn]|[Nn][Oo]) return 1 ;;
            *) echo -e "${YELLOW}Please answer yes or no.${NC}" ;;
        esac
    done
}

ask_input() {
    local question="$1"
    local default="$2"
    local response
    
    # Check if running in non-interactive mode
    if [[ ! -t 0 ]]; then
        # Non-interactive mode - use default
        echo "$default"
        return
    fi
    
    if [[ -n "$default" ]]; then
        echo -e -n "${CYAN}$question [$default]: ${NC}"
    else
        echo -e -n "${CYAN}$question: ${NC}"
    fi
    
    read -r response
    
    # Use default if empty response
    if [[ -z "$response" && -n "$default" ]]; then
        response="$default"
    fi
    
    echo "$response"
}

check_dependencies() {
    # Check if root .env.example exists
    if [[ ! -f "$ROOT_ENV_EXAMPLE" ]]; then
        print_error "Root environment template not found: $ROOT_ENV_EXAMPLE"
        print_error "Make sure you're running this script from the CLI directory"
        exit 1
    fi
    
    # Check if GOPATH/bin is in PATH
    if ! command -v "$CLI_NAME" >/dev/null 2>&1; then
        print_warning "CLI binary '$CLI_NAME' not found in PATH"
        print_info "Make sure \$GOPATH/bin is in your PATH environment variable"
        print_info "Add this to your shell profile: export PATH=\$PATH:\$GOPATH/bin"
    fi
}

setup_config_directory() {
    echo -e "${BLUE}Setting up configuration directory...${NC}"
    
    # Create config directory
    if [[ ! -d "$CONFIG_DIR" ]]; then
        mkdir -p "$CONFIG_DIR"
        print_success "Created configuration directory: $CONFIG_DIR"
    else
        print_info "Configuration directory already exists: $CONFIG_DIR"
    fi
}

setup_configuration() {
    echo ""
    echo -e "${BLUE}CLI Configuration Setup${NC}"
    echo -e "${BLUE}=======================${NC}"
    
    # Check if config already exists
    if [[ -f "$CONFIG_FILE" ]]; then
        echo ""
        print_warning "CLI configuration already exists: $CONFIG_FILE"
        
        if ask_yes_no "Do you want to overwrite the existing configuration?" "n"; then
            print_info "Backing up existing configuration..."
            cp "$CONFIG_FILE" "$CONFIG_FILE.backup.$(date +%Y%m%d_%H%M%S)"
            print_success "Backup created: $CONFIG_FILE.backup.*"
        else
            print_info "Keeping existing configuration file"
            return 0
        fi
    fi
    
    echo ""
    if ask_yes_no "Do you want to create a CLI configuration with default settings?" "y"; then
        # Generate CLI config.yaml from defaults
        generate_cli_config
        print_success "CLI configuration created: $CONFIG_FILE"
        
        # Offer to customize basic settings
        echo ""
        if ask_yes_no "Do you want to customize basic settings now?" "y"; then
            customize_basic_settings
        fi
        
        # Show AI configuration information
        show_ai_configuration_info
        
    else
        print_info "Skipping CLI configuration creation"
        print_info "CLI will use environment variables with LMMC_ prefix"
        print_info "See root .env.example for available options: $ROOT_ENV_EXAMPLE"
    fi
}

generate_cli_config() {
    cat > "$CONFIG_FILE" << 'EOF'
# LMMC CLI Configuration
# This file is generated from the installation script
# Environment variables with LMMC_ prefix take precedence over these settings
# 
# For the complete server configuration, see the root .env file

# ================================================================
# SERVER CONNECTION
# ================================================================
server:
  # MCP Memory Server URL
  url: "http://localhost:9080"
  
  # API version to use
  version: "v1"
  
  # Connection timeout in seconds
  timeout: 30

# ================================================================
# CLI BEHAVIOR
# ================================================================
cli:
  # Default repository when none specified
  # Options: "" (auto-detect), "current", or specific path
  default_repository: ""
  
  # Output format for commands
  # Options: table, json, yaml
  output_format: "table"
  
  # Enable tab completion and command suggestions
  auto_complete: true
  
  # Color scheme for output
  # Options: auto (detect terminal), always, never
  color_scheme: "auto"
  
  # Number of items to show per page in paginated output
  page_size: 20
  
  # Default text editor for interactive commands
  # If empty, uses $EDITOR environment variable or system default
  editor: ""

# ================================================================
# STORAGE & CACHING
# ================================================================
storage:
  # Enable local caching of server responses
  cache_enabled: true
  
  # Cache time-to-live in seconds (300 = 5 minutes)
  cache_ttl: 300
  
  # Number of backup files to keep
  backup_count: 3

# ================================================================
# LOGGING
# ================================================================
logging:
  # Log level for CLI operations
  # Options: debug, info, warn, error
  level: "info"
  
  # Log output format
  # Options: text, json
  format: "text"
  
  # Log file path (optional)
  # If empty, logs only to stderr
  file: ""

# ================================================================
# ENVIRONMENT VARIABLE OVERRIDES
# ================================================================
# The following environment variables can override settings above:
#
# Server Connection:
# LMMC_SERVER_URL       - Override server.url
# LMMC_SERVER_VERSION   - Override server.version  
# LMMC_SERVER_TIMEOUT   - Override server.timeout
#
# CLI Behavior:
# LMMC_CLI_DEFAULT_REPOSITORY - Override cli.default_repository
# LMMC_CLI_OUTPUT_FORMAT      - Override cli.output_format
# LMMC_CLI_AUTO_COMPLETE      - Override cli.auto_complete
# LMMC_CLI_COLOR_SCHEME       - Override cli.color_scheme
# LMMC_CLI_PAGE_SIZE          - Override cli.page_size
# LMMC_CLI_EDITOR             - Override cli.editor
#
# Storage & Caching:
# LMMC_STORAGE_CACHE_ENABLED  - Override storage.cache_enabled
# LMMC_STORAGE_CACHE_TTL      - Override storage.cache_ttl
# LMMC_STORAGE_BACKUP_COUNT   - Override storage.backup_count
#
# Logging:
# LMMC_LOGGING_LEVEL    - Override logging.level
# LMMC_LOGGING_FORMAT   - Override logging.format
# LMMC_LOGGING_FILE     - Override logging.file
#
# AI Configuration (environment variables only):
# See the root .env file for complete AI configuration options
# including AI_PROVIDER, CLAUDE_API_KEY, OPENAI_API_KEY, etc.
EOF
}

customize_basic_settings() {
    echo ""
    echo -e "${BLUE}Basic Configuration Customization${NC}"
    echo -e "${BLUE}==================================${NC}"
    
    # Server URL
    echo ""
    echo -e "${CYAN}Server Configuration:${NC}"
    local server_url
    server_url=$(ask_input "MCP Memory Server URL" "http://localhost:9080")
    
    if [[ "$server_url" != "http://localhost:9080" ]]; then
        # Update the server URL in YAML file
        if command -v sed >/dev/null 2>&1; then
            sed -i.bak "s|url: \"http://localhost:9080\"|url: \"$server_url\"|g" "$CONFIG_FILE"
            rm -f "$CONFIG_FILE.bak"
            print_success "Updated server URL to: $server_url"
        else
            print_warning "Could not automatically update server URL. Please edit manually."
        fi
    fi
    
    # Output format
    echo ""
    echo -e "${CYAN}CLI Behavior:${NC}"
    echo "Available output formats: table, json, yaml"
    local output_format
    output_format=$(ask_input "Preferred output format" "table")
    
    if [[ "$output_format" != "table" && "$output_format" =~ ^(json|yaml)$ ]]; then
        if command -v sed >/dev/null 2>&1; then
            sed -i.bak "s|output_format: \"table\"|output_format: \"$output_format\"|g" "$CONFIG_FILE"
            rm -f "$CONFIG_FILE.bak"
            print_success "Updated output format to: $output_format"
        else
            print_warning "Could not automatically update output format. Please edit manually."
        fi
    fi
    
    # Log level
    echo ""
    echo -e "${CYAN}Logging:${NC}"
    echo "Available log levels: debug, info, warn, error"
    local log_level
    log_level=$(ask_input "Log level" "info")
    
    if [[ "$log_level" != "info" && "$log_level" =~ ^(debug|warn|error)$ ]]; then
        if command -v sed >/dev/null 2>&1; then
            sed -i.bak "s|level: \"info\"|level: \"$log_level\"|g" "$CONFIG_FILE"
            rm -f "$CONFIG_FILE.bak"
            print_success "Updated log level to: $log_level"
        else
            print_warning "Could not automatically update log level. Please edit manually."
        fi
    fi
}

show_ai_configuration_info() {
    echo ""
    echo -e "${BLUE}AI Configuration Information${NC}"
    echo -e "${BLUE}============================${NC}"
    echo ""
    print_info "AI features require environment variables to be set."
    print_info "The following providers are supported:"
    echo ""
    echo -e "${CYAN}• Claude/Anthropic:${NC}"
    echo "  export CLAUDE_API_KEY=your_api_key_here"
    echo ""
    echo -e "${CYAN}• OpenAI:${NC}"
    echo "  export OPENAI_API_KEY=your_api_key_here"
    echo ""
    echo -e "${CYAN}• Perplexity:${NC}"
    echo "  export PERPLEXITY_API_KEY=your_api_key_here"
    echo ""
    print_info "Provider auto-detection: If you set multiple API keys, the CLI will"
    print_info "automatically choose in this priority: Claude > OpenAI > Perplexity"
    echo ""
    print_info "To force a specific provider, set environment variable:"
    print_info "export AI_PROVIDER=claude|openai|perplexity"
    echo ""
    print_info "For complete server and AI configuration, see: $ROOT_ENV_EXAMPLE"
    print_info "CLI configuration: $CONFIG_FILE"
}

show_completion_info() {
    echo ""
    echo -e "${BLUE}Installation Complete!${NC}"
    echo -e "${BLUE}======================${NC}"
    echo ""
    print_success "LMMC CLI has been installed successfully"
    echo ""
    echo -e "${CYAN}Next steps:${NC}"
    echo "1. Ensure \$GOPATH/bin is in your PATH"
    echo "2. Set up AI provider environment variables (optional)"
    echo "3. Start the MCP Memory Server if not already running"
    echo "4. Test the CLI: ${CYAN}lmmc --help${NC}"
    echo ""
    echo -e "${CYAN}Configuration files:${NC}"
    echo "• CLI config: $CONFIG_FILE"
    echo "• Server/AI config: $ROOT_ENV_EXAMPLE"
    echo ""
    echo -e "${CYAN}Useful commands:${NC}"
    echo "• ${CYAN}lmmc config show${NC}     - Show current configuration"
    echo "• ${CYAN}lmmc config edit${NC}     - Edit configuration file"
    echo "• ${CYAN}lmmc --help${NC}          - Show all available commands"
    echo "• ${CYAN}lmmc status${NC}          - Check connection to server"
    echo ""
}

# Main installation flow
main() {
    print_header
    
    echo "This script will set up the LMMC CLI configuration."
    echo ""
    
    # Check dependencies
    check_dependencies
    
    # Setup configuration directory
    setup_config_directory
    
    # Setup configuration file
    setup_configuration
    
    # Show completion information
    show_completion_info
}

# Run main function
main "$@"