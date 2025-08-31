#!/bin/bash

# Test runner script for comprehensive test suite

echo "🧪 Running Terraform Import Helper Test Suite"
echo "=============================================="

# Change to project directory
cd "$(dirname "$0")"

echo "📍 Current directory: $(pwd)"
echo ""

# Run go mod tidy first
echo "🔧 Running go mod tidy..."
go mod tidy
echo ""

# Test individual packages
echo "🧮 Testing Models package..."
go test -v ./internal/models
echo ""

echo "🔍 Testing Discovery package..."
go test -v ./internal/discovery
echo ""

echo "🎨 Testing TUI package..."
go test -v ./internal/tui
echo ""

echo "📥 Testing Importer package..."
go test -v ./internal/importer
echo ""

# Run all tests
echo "🚀 Running all tests..."
go test -v ./...
echo ""

echo "[OK] Test suite completed!"
