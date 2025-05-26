# CI Cache Management

## Clearing GitHub Actions Cache

If you encounter tar restoration warnings or cache-related issues in CI, you can clear the GitHub Actions cache using the following commands:

### List all caches
```bash
gh api /repos/fredcamaral/mcp-memory/actions/caches --jq '.actions_caches[] | {key: .key, size_in_bytes: .size_in_bytes, created_at: .created_at}'
```

### Delete all caches
```bash
gh api /repos/fredcamaral/mcp-memory/actions/caches --jq '.actions_caches[].id' | while read cache_id; do
  echo "Deleting cache ID: $cache_id"
  gh api -X DELETE /repos/fredcamaral/mcp-memory/actions/caches/$cache_id
done
```

### Delete specific cache by key pattern
```bash
# Example: Delete all Go module caches
gh api /repos/fredcamaral/mcp-memory/actions/caches --jq '.actions_caches[] | select(.key | contains("go-")) | .id' | while read cache_id; do
  gh api -X DELETE /repos/fredcamaral/mcp-memory/actions/caches/$cache_id
done
```

## Common Cache Issues

1. **Tar restoration warnings**: Usually caused by corrupted or incomplete cache entries. Solution: Clear all caches and let them rebuild.

2. **Cache size limits**: GitHub Actions has a 10GB total cache size limit per repository. Old caches are automatically evicted when this limit is reached.

3. **Stale dependencies**: Sometimes cached dependencies become outdated. Clear module-specific caches to force a fresh download.

## CodeQL Configuration Conflicts

If you see the error "CodeQL analyses from advanced configurations cannot be processed when the default setup is enabled", this indicates a conflict between:
- CodeQL default setup (configured in repository settings)
- CodeQL advanced setup (configured in workflow files)

To resolve:
1. Disable CodeQL default setup in repository settings under Security > Code scanning
2. Or remove the CodeQL job from the Security workflow and rely on the default setup