#!/bin/bash
set -e

# Install script for lmmc (Lerian MCP Memory CLI)
# Usage: curl -fsSL https://raw.githubusercontent.com/lerianstudio/lerian-mcp-memory/main/scripts/install.sh | bash

REPO="lerianstudio/lerian-mcp-memory"
BINARY_NAME="lmmc"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Detect OS and architecture
detect_platform() {
    local os arch

    # Detect OS
    case "$(uname -s)" in
        Darwin*) os="darwin" ;;
        Linux*) os="linux" ;;
        CYGWIN*|MINGW*|MSYS*) os="windows" ;;
        *) 
            log_error "Unsupported operating system: $(uname -s)"
            exit 1
            ;;
    esac

    # Detect architecture
    case "$(uname -m)" in
        x86_64|amd64) arch="amd64" ;;
        arm64|aarch64) arch="arm64" ;;
        armv7l) arch="arm" ;;
        *) 
            log_error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac

    echo "${os}-${arch}"
}

# Get latest release version from GitHub
get_latest_version() {
    local version
    version=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    
    if [ -z "$version" ]; then
        log_error "Failed to get latest version from GitHub API"
        exit 1
    fi
    
    echo "$version"
}

# Download and install binary
install_binary() {
    local version=$1
    local platform=$2
    local temp_dir=$(mktemp -d)
    local archive_name="${BINARY_NAME}-${version}-${platform}"
    
    if [[ "$platform" == *"windows"* ]]; then
        archive_name="${archive_name}.zip"
    else
        archive_name="${archive_name}.tar.gz"
    fi
    
    local download_url="https://github.com/${REPO}/releases/download/${version}/${archive_name}"
    
    log_info "Downloading ${BINARY_NAME} ${version} for ${platform}..."
    
    # Download archive
    if ! curl -fsSL -o "${temp_dir}/${archive_name}" "$download_url"; then
        log_error "Failed to download ${archive_name}"
        exit 1
    fi
    
    # Extract archive
    cd "$temp_dir"
    if [[ "$archive_name" == *.zip ]]; then
        unzip -q "$archive_name"
        binary_path="${BINARY_NAME}.exe"
    else
        tar -xzf "$archive_name"
        binary_path="$BINARY_NAME"
    fi
    
    # Verify binary exists
    if [ ! -f "$binary_path" ]; then
        log_error "Binary not found in archive"
        exit 1
    fi
    
    # Install binary
    log_info "Installing to ${INSTALL_DIR}/${BINARY_NAME}..."
    
    # Create install directory if it doesn't exist
    sudo mkdir -p "$INSTALL_DIR"
    
    # Install binary
    sudo mv "$binary_path" "${INSTALL_DIR}/${BINARY_NAME}"
    sudo chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    
    # Cleanup
    rm -rf "$temp_dir"
    
    log_success "${BINARY_NAME} ${version} installed successfully!"
}

# Verify installation
verify_installation() {
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        local installed_version
        installed_version=$("$BINARY_NAME" version 2>/dev/null | head -n1 || echo "unknown")
        log_success "Installation verified: $installed_version"
        log_info "Try running: $BINARY_NAME tui"
    else
        log_warning "Binary installed but not found in PATH. You may need to add ${INSTALL_DIR} to your PATH."
        log_info "Run: echo 'export PATH=\"${INSTALL_DIR}:\$PATH\"' >> ~/.bashrc && source ~/.bashrc"
    fi
}

# Show usage information
show_usage() {
    cat << EOF

🎉 ${BINARY_NAME} has been installed!

Quick Start:
  ${BINARY_NAME} tui              # Start interactive TUI dashboard
  ${BINARY_NAME} add "task"       # Add a new task
  ${BINARY_NAME} list             # List all tasks
  ${BINARY_NAME} help             # Show all commands

TUI Navigation:
  F1-F6: Switch views (Command, Dashboard, Analytics, Tasks, Patterns, Insights)
  Tab:   Navigate between panes
  hjkl:  Vim-style navigation
  r:     Refresh data

Configuration:
  Config file: ~/.lmmc/config.yaml
  Run: ${BINARY_NAME} config --help

For more information, visit: https://github.com/${REPO}

EOF
}

# Main installation flow
main() {
    echo -e "${BLUE}"
    cat << "EOF"
    ██      ███    ███ ███    ███  ██████ 
    ██      ████  ████ ████  ████ ██      
    ██      ██ ████ ██ ██ ████ ██ ██      
    ██      ██  ██  ██ ██  ██  ██ ██      
    ███████ ██      ██ ██      ██  ██████ 
    
    Lerian MCP Memory CLI Installer
    Intelligent Multi-Repository Task Management
EOF
    echo -e "${NC}"
    
    # Check if running as root
    if [ "$EUID" -eq 0 ]; then
        log_warning "Running as root. This is not recommended."
    fi
    
    # Detect platform
    local platform
    platform=$(detect_platform)
    log_info "Detected platform: $platform"
    
    # Get latest version
    local version
    version=$(get_latest_version)
    log_info "Latest version: $version"
    
    # Check if already installed
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        local current_version
        current_version=$("$BINARY_NAME" version 2>/dev/null | head -n1 || echo "unknown")
        log_info "Current installation: $current_version"
        
        if [[ "$current_version" == *"$version"* ]]; then
            log_success "Latest version already installed!"
            show_usage
            exit 0
        else
            log_info "Updating from $current_version to $version"
        fi
    fi
    
    # Install binary
    install_binary "$version" "$platform"
    
    # Verify installation
    verify_installation
    
    # Show usage
    show_usage
}

# Run main function
main "$@"