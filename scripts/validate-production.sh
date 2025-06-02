#!/bin/bash
# Production validation script for MCP Memory Server

set -e

echo "🧹 MCP Memory Production Validation"
echo "===================================="

# Check for artifacts that shouldn't exist
echo "1. Checking for development artifacts..."

# Check for binary artifacts
if [ -d "bin/" ]; then
    echo "❌ ERROR: bin/ directory exists (should be gitignored)"
    exit 1
fi

# Check for backup files
if find . -name "*.bak" -o -name "*.backup" -o -name "*.bak2" | grep -q .; then
    echo "❌ ERROR: Backup files found"
    find . -name "*.bak" -o -name "*.backup" -o -name "*.bak2"
    exit 1
fi

# Check for audit logs
if find . -name "audit_logs" -type d | grep -q .; then
    echo "❌ ERROR: Development audit logs found"
    find . -name "audit_logs" -type d
    exit 1
fi

# Check for external docs copy
if [ -d "docs/tmp/" ]; then
    echo "❌ ERROR: External documentation copy found in docs/tmp/"
    exit 1
fi

# Check for marketing prototypes
if [ -d "lp/" ]; then
    echo "❌ ERROR: Marketing prototypes found in lp/"
    exit 1
fi

echo "✅ No development artifacts found"

# Build validation
echo "2. Testing build process..."
make clean || echo "No clean target, continuing..."

echo "3. Testing Go build..."
go build ./cmd/server
echo "✅ Server builds successfully"

echo "4. Testing Docker build..."
docker build -t mcp-memory-test . >/dev/null 2>&1
echo "✅ Docker builds successfully"

echo "5. Running tests..."
go test -short ./... >/dev/null 2>&1
echo "✅ Tests pass"

echo "6. Checking dependencies..."
go mod verify >/dev/null 2>&1
echo "✅ Dependencies verified"

echo ""
echo "🎉 Production validation complete!"
echo "✅ Codebase ready for production deployment"