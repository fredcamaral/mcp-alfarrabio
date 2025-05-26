#!/bin/bash

echo "🚀 MCP Feature Testing Script"
echo "============================"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Base URL
BASE_URL="http://localhost:3000/rpc"

# Helper function to send JSON-RPC request
send_request() {
    local method=$1
    local params=$2
    local id=$(date +%s%N)
    
    if [ -z "$params" ]; then
        params="null"
    fi
    
    local request="{\"jsonrpc\":\"2.0\",\"id\":$id,\"method\":\"$method\",\"params\":$params}"
    
    echo -e "${BLUE}→ Sending: $method${NC}"
    
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$request" \
        "$BASE_URL")
    
    echo "$response" | jq '.' 2>/dev/null || echo "$response"
    echo
}

# Test 1: Initialize as different clients
echo -e "${GREEN}1. Testing Initialize with Different Clients${NC}"
echo "---------------------------------------------"

# Claude Desktop
echo "Testing as Claude Desktop:"
send_request "initialize" '{
    "protocolVersion": "2024-11-05",
    "clientInfo": {
        "name": "claude-desktop",
        "version": "1.0.0"
    },
    "capabilities": {}
}'

# VS Code Copilot
echo "Testing as VS Code Copilot:"
send_request "initialize" '{
    "protocolVersion": "2024-11-05",
    "clientInfo": {
        "name": "vscode-copilot",
        "version": "1.0.0"
    },
    "capabilities": {}
}'

# Test 2: Core Features
echo -e "${GREEN}2. Testing Core Features${NC}"
echo "------------------------"

# List tools
send_request "tools/list"

# Call echo tool
send_request "tools/call" '{
    "name": "echo",
    "arguments": {
        "message": "Hello from MCP test!"
    }
}'

# List resources
send_request "resources/list"

# Read resource
send_request "resources/read" '{
    "uri": "demo://test.txt"
}'

# List prompts
send_request "prompts/list"

# Get prompt
send_request "prompts/get" '{
    "name": "greeting",
    "arguments": {
        "name": "MCP Tester",
        "style": "casual"
    }
}'

# Test 3: New Features
echo -e "${GREEN}3. Testing New Features${NC}"
echo "-----------------------"

# List roots
send_request "roots/list"

# Sampling
send_request "sampling/createMessage" '{
    "messages": [
        {
            "role": "user",
            "content": {
                "type": "text",
                "text": "What is the Model Context Protocol?"
            }
        }
    ],
    "maxTokens": 150,
    "temperature": 0.7
}'

# Discovery
send_request "discovery/discover" '{
    "filter": {
        "available": true
    }
}'

# Subscribe to tools
send_request "tools/subscribe" '{}'

# Test 4: Error Handling
echo -e "${GREEN}4. Testing Error Handling${NC}"
echo "-------------------------"

# Invalid method
send_request "invalid/method"

# Invalid tool call
send_request "tools/call" '{
    "name": "nonexistent",
    "arguments": {}
}'

# Missing required parameters
send_request "tools/call" '{
    "name": "echo"
}'

echo -e "${GREEN}✅ Testing Complete!${NC}"