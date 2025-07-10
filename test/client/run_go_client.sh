#!/bin/bash

# MCP Streamable HTTP Transport - Go Client Runner
# Usage: ./run_go_client.sh [server_url]

set -e

# Default server URL
DEFAULT_SERVER_URL="http://localhost:8080"
SERVER_URL="${1:-$DEFAULT_SERVER_URL}"

echo "🚀 Starting MCP Streamable HTTP Transport Go Client"
echo "🌐 Server URL: $SERVER_URL"
echo "📁 Working directory: $(pwd)"
echo ""

# Check if we're in the right directory
if [[ ! -f "go_client.go" ]]; then
    echo "❌ Error: go_client.go not found in current directory"
    echo "   Please run this script from the test/client directory"
    exit 1
fi

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Error: Go is not installed or not in PATH"
    echo "   Please install Go: https://golang.org/doc/install"
    exit 1
fi

# Check if we can access the go.mod file (need to be in project root context)
if [[ ! -f "../../go.mod" ]]; then
    echo "❌ Error: go.mod not found in project root"
    echo "   Please ensure you're running from the correct project structure"
    exit 1
fi

echo "✅ Go installation found: $(go version)"
echo "✅ Project structure validated"
echo ""

echo "🔧 Building and running Go client..."
echo "   Press Ctrl+C to stop the client"
echo ""

# Run the Go client with the specified server URL
cd ../.. # Go to project root for module context
go run test/client/go_client.go "$SERVER_URL"