#!/bin/bash

# DriftMgr Cleanup Script
# This script removes duplicate files and merges directories

set -e  # Exit on any error

echo "🧹 Starting DriftMgr cleanup..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to move file with error handling
move_file() {
    local src="$1"
    local dest="$2"
    
    if [ -f "$src" ]; then
        if [ ! -f "$dest" ]; then
            mv "$src" "$dest"
            print_success "Moved: $src -> $dest"
        else
            print_warning "Destination already exists, skipping: $dest"
        fi
    else
        print_warning "Source file not found: $src"
    fi
}

# Function to delete file with error handling
delete_file() {
    local file="$1"
    
    if [ -f "$file" ]; then
        rm "$file"
        print_success "Deleted: $file"
    else
        print_warning "File not found: $file"
    fi
}

# Function to delete directory with error handling
delete_dir() {
    local dir="$1"
    
    if [ -d "$dir" ]; then
        rm -rf "$dir"
        print_success "Deleted directory: $dir"
    else
        print_warning "Directory not found: $dir"
    fi
}

# Phase 1: Move remaining markdown files to docs/summaries/
print_status "Phase 1: Moving remaining markdown files..."

remaining_md_files=(
    "ENHANCED_CLI_IMPLEMENTATION_SUMMARY.md"
    "EXPANDED_SERVICES_IMPLEMENTATION_SUMMARY.md"
    "IMMEDIATE_IMPROVEMENTS_PLAN.md"
    "PROPOSED_REORGANIZATION.md"
    "RESTRUCTURE_PLAN.md"
)

for md_file in "${remaining_md_files[@]}"; do
    move_file "$md_file" "docs/summaries/$md_file"
done

# Phase 2: Move shell script to scripts/
print_status "Phase 2: Moving shell script..."

move_file "delete_aws_resources.sh" "scripts/delete_aws_resources.sh"

# Phase 3: Move test file from test/ to tests/
print_status "Phase 3: Moving test file..."

if [ -f "test/deletion_test.go" ]; then
    move_file "test/deletion_test.go" "tests/unit/deletion_test.go"
    # Delete empty test directory
    delete_dir "test"
fi

# Phase 4: Delete duplicate executables from root
print_status "Phase 4: Deleting duplicate executables from root..."

duplicate_exes=(
    "driftmgr-client.exe"
    "driftmgr-server.exe"
)

for exe in "${duplicate_exes[@]}"; do
    if [ -f "$exe" ] && [ -f "bin/$exe" ]; then
        delete_file "$exe"
    fi
done

# Phase 5: Delete backup executable in bin/
print_status "Phase 5: Deleting backup executable..."

delete_file "bin/driftmgr-server.exe~"

# Phase 6: Delete empty tools directory
print_status "Phase 6: Deleting empty tools directory..."

if [ -d "tools" ] && [ -z "$(ls -A tools)" ]; then
    delete_dir "tools"
fi

# Phase 7: Create missing test directories if needed
print_status "Phase 7: Creating missing test directories..."

mkdir -p tests/unit
mkdir -p tests/integration
mkdir -p tests/e2e

print_success "✅ DriftMgr cleanup complete!"
print_status "📋 Summary of cleanup:"
echo "  - Moved $(ls docs/summaries/ | wc -l) markdown files to docs/summaries/"
echo "  - Moved shell script to scripts/"
echo "  - Moved test file to tests/unit/"
echo "  - Deleted duplicate executables from root"
echo "  - Deleted backup executable from bin/"
echo "  - Deleted empty tools directory"
echo ""
print_status "📖 The project structure is now clean and organized"
