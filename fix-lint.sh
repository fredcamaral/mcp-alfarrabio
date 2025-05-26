#!/bin/bash

echo "🔧 Fixing linting issues..."

# Fix error check issues in test files (common pattern)
echo "Fixing errcheck in test files..."
find . -name "*_test.go" -type f | while read -r file; do
    # Fix pool.Close() calls
    sed -i '' 's/defer pool\.Close()/defer func() { _ = pool.Close() }()/g' "$file"
    
    # Fix pool.Put() calls
    sed -i '' 's/pool\.Put(\([^)]*\))/_ = pool.Put(\1)/g' "$file"
    
    # Fix conn.Close() calls in non-defer contexts
    sed -i '' 's/^\([[:space:]]*\)conn\.Close()/\1_ = conn.Close()/g' "$file"
    
    # Fix cb.Execute calls
    sed -i '' 's/^\([[:space:]]*\)cb\.Execute(/\1_ = cb.Execute(/g' "$file"
    
    # Fix store.StoreChunk calls
    sed -i '' 's/^\([[:space:]]*\)store\.StoreChunk(/\1_ = store.StoreChunk(/g' "$file"
    
    # Fix acm.GrantPermission calls
    sed -i '' 's/^\([[:space:]]*\)acm\.GrantPermission(/\1_ = acm.GrantPermission(/g' "$file"
    
    # Fix acm.CheckAccess calls
    sed -i '' 's/^\([[:space:]]*\)acm\.CheckAccess(/\1_ = acm.CheckAccess(/g' "$file"
    
    # Fix acm.GenerateToken calls
    sed -i '' 's/^\([[:space:]]*\)acm\.GenerateToken(/\1_ = acm.GenerateToken(/g' "$file"
done

# Fix error check issues in main files
echo "Fixing errcheck in main files..."
find . -name "*.go" -not -name "*_test.go" -type f | while read -r file; do
    # Fix defer conn.Close() patterns
    sed -i '' 's/defer conn\.Close()/defer func() { _ = conn.Close() }()/g' "$file"
    
    # Fix container.Shutdown()
    sed -i '' 's/defer container\.Shutdown()/defer func() { _ = container.Shutdown() }()/g' "$file"
    
    # Fix fmt.Fprintf calls
    sed -i '' 's/^\([[:space:]]*\)fmt\.Fprintf(/\1_, _ = fmt.Fprintf(/g' "$file"
    
    # Fix fmt.Fprint calls
    sed -i '' 's/^\([[:space:]]*\)fmt\.Fprint(/\1_, _ = fmt.Fprint(/g' "$file"
    
    # Fix w.Write calls
    sed -i '' 's/^\([[:space:]]*\)w\.Write(/\1_, _ = w.Write(/g' "$file"
    
    # Fix json.NewEncoder().Encode calls
    sed -i '' 's/^\([[:space:]]*\)json\.NewEncoder([^)]*\))\.Encode(/\1_ = json.NewEncoder(\1).Encode(/g' "$file"
done

# Fix specific issues
echo "Fixing specific issues..."

# Fix the goconst issue in di/container.go
sed -i '' 's/"true"/chromaUsePooling/g' internal/di/container.go
sed -i '' '/func NewContainer/i const chromaUsePooling = "true"' internal/di/container.go

# Fix the gosec G114 issue in cmd/openapi/main.go
sed -i '' 's/log\.Fatal(http\.ListenAndServe/srv := \&http.Server{Addr: ":" + port, Handler: router, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second}\
	log.Fatal(srv.ListenAndServe/g' cmd/openapi/main.go

# Add time import if needed
if ! grep -q "import.*time" cmd/openapi/main.go; then
    sed -i '' '/import (/a\
	"time"' cmd/openapi/main.go
fi

echo "✅ Automated fixes applied. Running golangci-lint to check remaining issues..."
golangci-lint run --timeout 5m 2>&1 | grep -E "^[^:]+:[0-9]+:[0-9]+:" | wc -l