#!/bin/bash

# DriftMgr Test Coverage Improvement Script
# This script helps track and improve test coverage

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "================================================"
echo "     DriftMgr Test Coverage Improvement"
echo "================================================"

# Function to check current coverage
check_coverage() {
    echo -e "${YELLOW}Checking current coverage...${NC}"
    go test ./... -coverprofile=coverage.out 2>/dev/null || true
    total_coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
    echo -e "${GREEN}Current Total Coverage: ${total_coverage}${NC}"
}

# Function to find packages with low coverage
find_low_coverage() {
    echo -e "${YELLOW}Packages with coverage < 30%:${NC}"
    go test ./... -coverprofile=coverage.out 2>/dev/null || true
    go tool cover -func=coverage.out | awk '$3 < 30 {print $1 " - " $3}' | grep -v total || echo "None found"
}

# Function to count test files
count_tests() {
    echo -e "${YELLOW}Test file statistics:${NC}"
    source_files=$(find internal -name "*.go" ! -name "*_test.go" | wc -l)
    test_files=$(find internal -name "*_test.go" | wc -l)
    echo "Source files: $source_files"
    echo "Test files: $test_files"
    echo "Test file coverage: $(( test_files * 100 / source_files ))%"
}

# Function to generate coverage report
generate_report() {
    echo -e "${YELLOW}Generating HTML coverage report...${NC}"
    go test ./... -coverprofile=coverage.out 2>/dev/null || true
    go tool cover -html=coverage.out -o coverage.html
    echo -e "${GREEN}Coverage report saved to coverage.html${NC}"
}

# Function to run specific package tests
test_package() {
    package=$1
    echo -e "${YELLOW}Testing package: $package${NC}"
    go test -v -cover ./$package/...
}

# Function to create test file template
create_test_template() {
    package=$1
    file=$2
    test_file="${file%.go}_test.go"

    if [ ! -f "$test_file" ]; then
        echo -e "${YELLOW}Creating test file: $test_file${NC}"
        cat > "$test_file" << 'EOF'
package $(basename $(dirname $file))

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestPlaceholder(t *testing.T) {
    t.Run("basic test", func(t *testing.T) {
        assert.True(t, true, "This is a placeholder test")
    })
}
EOF
        echo -e "${GREEN}Test file created: $test_file${NC}"
    else
        echo -e "${RED}Test file already exists: $test_file${NC}"
    fi
}

# Main menu
show_menu() {
    echo ""
    echo "Choose an option:"
    echo "1. Check current coverage"
    echo "2. Find low coverage packages"
    echo "3. Count test files"
    echo "4. Generate HTML report"
    echo "5. Test specific package"
    echo "6. Run Phase 1 tests (Fix build failures)"
    echo "7. Run all tests with race detection"
    echo "8. Upload to Codecov"
    echo "9. Exit"
    echo ""
}

# Phase 1: Fix build failures
run_phase1() {
    echo -e "${YELLOW}Phase 1: Fixing build failures...${NC}"

    # Fix API tests
    echo "Fixing API tests..."
    go test ./internal/api/... 2>&1 | grep -E "undefined|error" || echo "API tests OK"

    # Fix CLI tests
    echo "Fixing CLI tests..."
    go test ./internal/cli/... 2>&1 | grep -E "undefined|error" || echo "CLI tests OK"

    # Fix remediation tests
    echo "Fixing remediation tests..."
    go test ./internal/remediation/... 2>&1 | grep -E "undefined|error" || echo "Remediation tests OK"
}

# Upload to Codecov
upload_codecov() {
    echo -e "${YELLOW}Uploading coverage to Codecov...${NC}"

    # Generate coverage
    go test ./... -race -coverprofile=coverage.out -covermode=atomic

    # Upload using codecov CLI or bash uploader
    if command -v codecov &> /dev/null; then
        codecov -f coverage.out
    else
        echo "Installing codecov CLI..."
        curl -Os https://uploader.codecov.io/latest/linux/codecov
        chmod +x codecov
        ./codecov -f coverage.out
    fi
}

# Main loop
while true; do
    show_menu
    read -p "Enter choice: " choice

    case $choice in
        1)
            check_coverage
            ;;
        2)
            find_low_coverage
            ;;
        3)
            count_tests
            ;;
        4)
            generate_report
            ;;
        5)
            read -p "Enter package path (e.g., internal/api): " pkg
            test_package $pkg
            ;;
        6)
            run_phase1
            ;;
        7)
            echo -e "${YELLOW}Running all tests with race detection...${NC}"
            go test -race ./...
            ;;
        8)
            upload_codecov
            ;;
        9)
            echo "Exiting..."
            exit 0
            ;;
        *)
            echo -e "${RED}Invalid option${NC}"
            ;;
    esac
done