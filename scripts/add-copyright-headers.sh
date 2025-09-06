#!/bin/bash

# Script to add copyright headers to all Go source files
# Usage: ./scripts/add-copyright-headers.sh

COPYRIGHT_HEADER="// Copyright (c) 2025 Catherine Vee and DriftMgr Contributors
// SPDX-License-Identifier: MIT
//
// This file is part of DriftMgr, an enterprise-grade infrastructure drift
// detection and remediation tool. For more information, see:
// https://github.com/catherinevee/driftmgr
"

# Function to add header to a file
add_header() {
    local file="$1"
    
    # Check if file already has copyright header
    if head -n 1 "$file" | grep -q "Copyright"; then
        echo "Skipping $file - already has copyright header"
        return
    fi
    
    # Check if file starts with package declaration
    if head -n 1 "$file" | grep -q "^package "; then
        # Add header before package declaration
        echo "Adding header to $file"
        {
            echo "$COPYRIGHT_HEADER"
            cat "$file"
        } > "$file.tmp" && mv "$file.tmp" "$file"
    elif head -n 1 "$file" | grep -q "^//go:build"; then
        # Handle build tags
        build_tag=$(head -n 1 "$file")
        rest=$(tail -n +2 "$file")
        echo "Adding header to $file (with build tags)"
        {
            echo "$build_tag"
            echo ""
            echo "$COPYRIGHT_HEADER"
            echo "$rest"
        } > "$file.tmp" && mv "$file.tmp" "$file"
    fi
}

# Find all Go files in the project
echo "Adding copyright headers to Go source files..."

# Process cmd directory
find cmd -name "*.go" -type f | while read -r file; do
    add_header "$file"
done

# Process internal directory
find internal -name "*.go" -type f | while read -r file; do
    add_header "$file"
done

# Process pkg directory
find pkg -name "*.go" -type f | while read -r file; do
    add_header "$file"
done

# Process test directories
find tests -name "*.go" -type f 2>/dev/null | while read -r file; do
    add_header "$file"
done

# Process quality directory
find quality -name "*.go" -type f 2>/dev/null | while read -r file; do
    add_header "$file"
done

echo "Copyright headers added successfully!"
echo ""
echo "To verify headers were added correctly:"
echo "  grep -l 'Copyright' \$(find . -name '*.go' -type f) | wc -l"
echo ""
echo "To find files without headers:"
echo "  find . -name '*.go' -type f -exec sh -c 'head -n 10 {} | grep -q Copyright || echo {}' \;"