#!/bin/bash
set -e

# Release script for lmmc (Lerian MCP Memory CLI)
# This script creates releases with cross-platform binaries

VERSION=${1:-"v0.1.0"}
REPO_ROOT=$(git rev-parse --show-toplevel)
CLI_DIR="$REPO_ROOT/cli"
DIST_DIR="$REPO_ROOT/dist"

echo "🚀 Creating release $VERSION"

# Clean and create dist directory
rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

cd "$CLI_DIR"

# Build flags
LDFLAGS="-s -w -X main.version=$VERSION -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# Cross-compile for multiple platforms
declare -a platforms=(
    "darwin/amd64"
    "darwin/arm64" 
    "linux/amd64"
    "linux/arm64"
    "windows/amd64"
)

echo "📦 Building binaries..."

for platform in "${platforms[@]}"; do
    IFS='/' read -r GOOS GOARCH <<< "$platform"
    
    output_name="lmmc"
    if [ "$GOOS" = "windows" ]; then
        output_name="lmmc.exe"
    fi
    
    archive_name="lmmc-$VERSION-$GOOS-$GOARCH"
    if [ "$GOOS" = "windows" ]; then
        archive_name="$archive_name.zip"
    else
        archive_name="$archive_name.tar.gz"
    fi
    
    echo "  Building $GOOS/$GOARCH..."
    
    env GOOS="$GOOS" GOARCH="$GOARCH" CGO_ENABLED=0 \
        go build -ldflags="$LDFLAGS" -o "$DIST_DIR/$output_name" ./cmd/lmmc
    
    # Create archive
    cd "$DIST_DIR"
    if [ "$GOOS" = "windows" ]; then
        zip -q "$archive_name" "$output_name"
    else
        tar -czf "$archive_name" "$output_name"
    fi
    rm "$output_name"
    cd "$CLI_DIR"
    
    # Calculate sha256
    sha256sum "$DIST_DIR/$archive_name" >> "$DIST_DIR/checksums.txt"
done

echo "📋 Generating release notes..."

# Generate release notes
cat > "$DIST_DIR/release-notes.md" << EOF
# lmmc $VERSION

## 🎉 Features

### Terminal User Interface (TUI)
- **Multi-View Dashboard**: Interactive terminal interface with 6 comprehensive views
- **Real-Time Analytics**: Cross-repository performance monitoring and insights
- **Pattern Visualization**: Workflow analysis with ASCII charts and success metrics
- **Multi-Repository Intelligence**: Centralized dashboard for all projects

### Core Functionality
- **AI-Powered Task Management**: Intelligent task suggestions and automation
- **Cross-Repository Analysis**: Pattern detection and similarity scoring
- **Document Generation**: Interactive PRD/TRD creation with AI assistance
- **Real-Time Sync**: Bidirectional synchronization with conflict resolution

### Intelligence Features
- **Pattern Detection**: ML-powered workflow pattern recognition
- **Cross-Repository Insights**: Statistical analysis and optimization recommendations
- **Template System**: Comprehensive template repository with AI-powered matching
- **Performance Analytics**: Detailed metrics with outlier detection

## 📦 Installation

### Homebrew (macOS/Linux)
\`\`\`bash
brew tap lerianstudio/tap
brew install lmmc
\`\`\`

### Manual Installation
Download the appropriate binary for your platform from the assets below.

## 🚀 Quick Start

\`\`\`bash
# Start the interactive TUI dashboard
lmmc tui

# Traditional CLI usage
lmmc add "Implement new feature"
lmmc list
lmmc tui --mode dashboard
\`\`\`

## 📊 TUI Navigation
- **F1-F6**: Switch between Command, Dashboard, Analytics, Tasks, Patterns, Insights
- **Tab/hjkl**: Navigate within views
- **r**: Refresh data
- **1-4**: Quick chart switching (Analytics view)
- **d/w/m**: Time range selection

## 🔧 Configuration
See [Configuration Guide](https://github.com/lerianstudio/lerian-mcp-memory/blob/main/docs/configuration.md) for detailed setup instructions.

## Full Changelog
* Added comprehensive TUI with 6 interactive views
* Implemented cross-repository analytics and insights
* Added ASCII chart visualization for metrics
* Enhanced pattern detection and workflow analysis
* Improved real-time synchronization with conflict resolution
* Added comprehensive template system with AI matching
* Implemented performance monitoring with outlier detection
EOF

echo "🔑 Updating Homebrew formula..."

# Update Homebrew formula with correct sha256
if [ -f "$DIST_DIR/lmmc-$VERSION-darwin-amd64.tar.gz" ]; then
    DARWIN_SHA256=$(sha256sum "$DIST_DIR/lmmc-$VERSION-darwin-amd64.tar.gz" | cut -d' ' -f1)
    sed -i.bak "s/PLACEHOLDER_SHA256/$DARWIN_SHA256/" "$REPO_ROOT/homebrew-formula/lmmc.rb"
    rm "$REPO_ROOT/homebrew-formula/lmmc.rb.bak"
fi

echo "✅ Release $VERSION created successfully!"
echo ""
echo "📁 Files created in $DIST_DIR:"
ls -la "$DIST_DIR"
echo ""
echo "Next steps:"
echo "1. Create GitHub release with tag $VERSION"
echo "2. Upload binaries from $DIST_DIR as release assets"
echo "3. Update Homebrew tap repository with updated formula"
echo "4. Announce release to users"