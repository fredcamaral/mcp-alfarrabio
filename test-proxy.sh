#!/bin/bash

echo "Testing MCP proxy..."

# Test tools/list
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | node mcp-proxy.js | jq '.result.tools | length'

# Test memory_get_context
echo '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"mcp__memory__memory_get_context","arguments":{"repository":"test-repo","recent_days":7}}}' | node mcp-proxy.js | jq '.result'

# Test memory_search
echo '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"mcp__memory__memory_search","arguments":{"query":"test search"}}}' | node mcp-proxy.js | jq '.result'