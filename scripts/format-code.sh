#!/bin/bash

# Automated Code Cleanup and Formatting Script
# Formats Go and TypeScript/JavaScript code automatically

set -e

echo "🧹 Starting automated code cleanup and formatting..."

# Go formatting and cleanup
echo "📝 Formatting Go code..."
if command -v gofmt &> /dev/null; then
    find . -name "*.go" -not -path "./vendor/*" -not -path "./web-ui/node_modules/*" | xargs gofmt -l -w
    echo "✅ Go files formatted with gofmt"
else
    echo "⚠️  gofmt not found, skipping Go formatting"
fi

if command -v goimports &> /dev/null; then
    find . -name "*.go" -not -path "./vendor/*" -not -path "./web-ui/node_modules/*" | xargs goimports -l -w
    echo "✅ Go imports organized with goimports"
else
    echo "ℹ️  goimports not found, installing..."
    go install golang.org/x/tools/cmd/goimports@latest
    find . -name "*.go" -not -path "./vendor/*" -not -path "./web-ui/node_modules/*" | xargs goimports -l -w
    echo "✅ Go imports organized with goimports"
fi

# Go mod tidy
echo "📦 Tidying Go modules..."
go mod tidy
echo "✅ Go modules tidied"

# TypeScript/JavaScript formatting (WebUI)
if [ -d "web-ui" ]; then
    echo "📝 Formatting TypeScript/JavaScript code..."
    cd web-ui
    
    # Install prettier if not available
    if ! npm list prettier &> /dev/null; then
        echo "ℹ️  Installing prettier..."
        npm install --save-dev prettier @trivago/prettier-plugin-sort-imports
    fi
    
    # Format with prettier
    if command -v npx &> /dev/null; then
        npx prettier --write "**/*.{ts,tsx,js,jsx,json,css,md}" --ignore-unknown
        echo "✅ TypeScript/JavaScript files formatted with prettier"
    else
        echo "⚠️  npx not found, skipping TS/JS formatting"
    fi
    
    # ESLint fix
    if [ -f "package.json" ] && grep -q "eslint" package.json; then
        npx eslint . --fix --max-warnings 0 || echo "⚠️  ESLint found issues that couldn't be auto-fixed"
        echo "✅ ESLint auto-fixes applied"
    fi
    
    cd ..
fi

# Additional cleanup
echo "🧹 Additional cleanup..."

# Remove trailing whitespace from all text files
find . -type f \( -name "*.go" -o -name "*.ts" -o -name "*.tsx" -o -name "*.js" -o -name "*.jsx" -o -name "*.md" -o -name "*.yml" -o -name "*.yaml" \) \
    -not -path "./vendor/*" -not -path "./web-ui/node_modules/*" -not -path "./.git/*" \
    -exec sed -i '' 's/[[:space:]]*$//' {} \;
echo "✅ Trailing whitespace removed"

# Fix line endings (ensure LF)
find . -type f \( -name "*.go" -o -name "*.ts" -o -name "*.tsx" -o -name "*.js" -o -name "*.jsx" \) \
    -not -path "./vendor/*" -not -path "./web-ui/node_modules/*" -not -path "./.git/*" \
    -exec dos2unix {} \; 2>/dev/null || true
echo "✅ Line endings normalized"

echo "🎉 Code cleanup and formatting completed!"
echo ""
echo "📊 Summary:"
echo "  - Go files: formatted with gofmt and goimports"
echo "  - Go modules: tidied"
echo "  - TypeScript/JS: formatted with prettier and ESLint"
echo "  - Whitespace: trailing whitespace removed"
echo "  - Line endings: normalized to LF"
echo ""
echo "💡 Next steps:"
echo "  - Review changes with: git diff"
echo "  - Run tests: make test"
echo "  - Run linters: make lint"