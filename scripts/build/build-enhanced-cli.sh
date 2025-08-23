#!/bin/bash

# DriftMgr CLI Build Script
# This script builds the CLI with all features

set -e

echo "=========================================="
echo "Building DriftMgr CLI"
echo "=========================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed or not in PATH${NC}"
    exit 1
fi

echo -e "${BLUE}Go version:${NC}"
go version
echo

# Set build variables
CLI_DIR="cmd/driftmgr-client"
OUTPUT_NAME="driftmgr-client"
BUILD_FILES=(
    "main.go"
    "completion.go"
    "enhanced_analyze.go"
    "remediate.go"
    "credentials.go"
)

# Check if all required files exist
echo -e "${BLUE}Checking required files...${NC}"
for file in "${BUILD_FILES[@]}"; do
    if [ -f "$CLI_DIR/$file" ]; then
        echo -e "${GREEN}✓${NC} $CLI_DIR/$file"
    else
        echo -e "${RED}✗${NC} $CLI_DIR/$file (missing)"
        exit 1
    fi
done
echo

# Build the CLI
echo -e "${BLUE}Building CLI...${NC}"
BUILD_CMD="go build -o $OUTPUT_NAME"

# Add all source files to build command
for file in "${BUILD_FILES[@]}"; do
    BUILD_CMD="$BUILD_CMD $CLI_DIR/$file"
done

echo "Build command: $BUILD_CMD"
echo

# Execute build
if eval $BUILD_CMD; then
	echo -e "${GREEN}✓ CLI built successfully!${NC}"
    echo -e "${BLUE}Output:${NC} $OUTPUT_NAME"
    
    # Show file size
    if command -v ls &> /dev/null; then
        echo -e "${BLUE}File size:${NC}"
        ls -lh $OUTPUT_NAME
    fi
else
    echo -e "${RED}✗ Build failed!${NC}"
    exit 1
fi

echo
echo "=========================================="
echo -e "${GREEN}Build Complete!${NC}"
echo "=========================================="
echo
echo "To test the CLI:"
echo "  ./$OUTPUT_NAME"
echo
echo "To run the demo:"
echo "  ./scripts/demo-enhanced-cli.sh"
echo
echo "For documentation:"
echo "  docs/cli/enhanced-features-guide.md"
echo
echo -e "${YELLOW}Features:${NC}"
echo "  ✓ Tab completion"
echo "  ✓ Auto-suggestions"
echo "  ✓ Fuzzy search"
echo "  ✓ Arrow key navigation"
echo "  ✓ Context-aware completion"
