#!/bin/bash

# Update Go imports after refactoring
# This script updates all import paths to use the new consolidated structure

echo "Updating Go imports after refactoring..."

# Define the base module path
MODULE="github.com/catherinevee/driftmgr"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to update imports in a file
update_file_imports() {
    local file=$1
    echo -e "${YELLOW}Processing: $file${NC}"
    
    # Backup the original file
    cp "$file" "$file.bak"
    
    # Update discovery imports
    sed -i "s|${MODULE}/internal/discovery/enhanced_discovery|${MODULE}/internal/core/discovery|g" "$file"
    sed -i "s|${MODULE}/internal/discovery/universal_discovery|${MODULE}/internal/core/discovery|g" "$file"
    sed -i "s|${MODULE}/internal/discovery/multi_account_discovery|${MODULE}/internal/core/discovery|g" "$file"
    sed -i "s|${MODULE}/internal/discovery\"|${MODULE}/internal/core/discovery\"|g" "$file"
    
    # Update drift imports
    sed -i "s|${MODULE}/internal/drift|${MODULE}/internal/core/drift|g" "$file"
    
    # Update remediation imports
    sed -i "s|${MODULE}/internal/remediation|${MODULE}/internal/core/remediation|g" "$file"
    
    # Update visualization imports
    sed -i "s|${MODULE}/internal/visualization|${MODULE}/internal/core/visualization|g" "$file"
    
    # Update dashboard imports to API
    sed -i "s|${MODULE}/internal/dashboard|${MODULE}/internal/api/rest|g" "$file"
    
    # Update provider-specific imports
    sed -i "s|${MODULE}/internal/discovery/aws_|${MODULE}/internal/providers/aws/|g" "$file"
    sed -i "s|${MODULE}/internal/discovery/azure_|${MODULE}/internal/providers/azure/|g" "$file"
    sed -i "s|${MODULE}/internal/discovery/gcp_|${MODULE}/internal/providers/gcp/|g" "$file"
    sed -i "s|${MODULE}/internal/discovery/digitalocean_|${MODULE}/internal/providers/digitalocean/|g" "$file"
    
    # Check if file was modified
    if diff -q "$file" "$file.bak" > /dev/null; then
        rm "$file.bak"
    else
        echo -e "${GREEN}  Updated imports in $file${NC}"
    fi
}

# Find all Go files and update imports
echo "Finding all Go files..."
GO_FILES=$(find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*")

# Count total files
TOTAL=$(echo "$GO_FILES" | wc -l)
CURRENT=0

echo -e "${GREEN}Found $TOTAL Go files to process${NC}\n"

# Process each file
for file in $GO_FILES; do
    CURRENT=$((CURRENT + 1))
    echo -e "[$CURRENT/$TOTAL] Processing $file"
    update_file_imports "$file"
done

echo -e "\n${GREEN}Import updates completed!${NC}"

# Run go mod tidy to clean up dependencies
echo -e "\n${YELLOW}Running 'go mod tidy' to clean up dependencies...${NC}"
go mod tidy

# Check for any compilation errors
echo -e "\n${YELLOW}Checking for compilation errors...${NC}"
if go build ./... 2>&1 | grep -q "error"; then
    echo -e "${RED}Compilation errors found! Please review and fix manually.${NC}"
    go build ./...
else
    echo -e "${GREEN}No compilation errors found!${NC}"
fi

# Run go fmt to format all files
echo -e "\n${YELLOW}Running 'go fmt' to format code...${NC}"
go fmt ./...

echo -e "\n${GREEN}Refactoring import updates complete!${NC}"
echo -e "${YELLOW}Please review the changes and run tests to ensure everything works correctly.${NC}"