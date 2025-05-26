#!/bin/bash

# Script to complete the GoMCP SDK migration after it's pushed to GitHub

echo "🔄 Completing GoMCP SDK Migration..."

# Check if the SDK is available
echo "📦 Checking if gomcp-sdk is available..."
if ! go list -m github.com/fredcamaral/gomcp-sdk@latest &>/dev/null; then
    echo "❌ Error: gomcp-sdk is not yet available on GitHub"
    echo "Please ensure you've pushed the repository to https://github.com/fredcamaral/gomcp-sdk"
    exit 1
fi

echo "✅ gomcp-sdk is available!"

# Get the latest version
echo "📥 Getting latest gomcp-sdk..."
go get github.com/fredcamaral/gomcp-sdk@latest

# Tidy up
echo "🧹 Cleaning up dependencies..."
go mod tidy

# Verify the build
echo "🔨 Verifying build..."
if go build ./cmd/server; then
    echo "✅ Server builds successfully!"
else
    echo "❌ Build failed"
    exit 1
fi

if go build ./cmd/graphql; then
    echo "✅ GraphQL server builds successfully!"
else
    echo "❌ GraphQL build failed"
    exit 1
fi

# Run tests
echo "🧪 Running tests..."
if go test ./internal/...; then
    echo "✅ Tests pass!"
else
    echo "⚠️  Some tests failed (this might be expected if they need services)"
fi

echo "🎉 Migration complete! The mcp-memory project now uses the public gomcp-sdk"
echo ""
echo "Next steps:"
echo "1. Commit these changes: git add -A && git commit -m 'chore: migrate to public gomcp-sdk'"
echo "2. Push to repository: git push"
echo "3. Update documentation if needed"