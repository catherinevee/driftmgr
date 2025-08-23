#!/bin/bash

# Test script for DriftMgr credential update functionality
# This script demonstrates how to use the new credential update feature

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}$1${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

echo "🧪 DriftMgr Credential Update Test"
echo "=================================="
echo

# Test 1: Update credentials using install script
print_status "Test 1: Testing credential update via install script"
echo "Running: ./install.sh --update-credentials"
echo

if [[ -f "./install.sh" ]]; then
    if ./install.sh --update-credentials; then
        print_success "Install script credential update test passed"
    else
        print_warning "Install script credential update test failed (expected if DriftMgr not installed)"
    fi
else
    print_warning "Install script not found, skipping test"
fi

echo

# Test 2: Update credentials using CLI directly
print_status "Test 2: Testing credential update via CLI"
echo "Running: ./bin/driftmgr credentials update"
echo

if [[ -f "./bin/driftmgr" ]]; then
    if ./bin/driftmgr credentials update; then
        print_success "CLI credential update test passed"
    else
        print_warning "CLI credential update test failed (expected if no credentials to update)"
    fi
elif [[ -f "./bin/driftmgr.exe" ]]; then
    if ./bin/driftmgr.exe credentials update; then
        print_success "CLI credential update test passed"
    else
        print_warning "CLI credential update test failed (expected if no credentials to update)"
    fi
else
    print_warning "DriftMgr binary not found, skipping test"
fi

echo

# Test 3: List current credentials
print_status "Test 3: Testing credential listing"
echo "Running: ./bin/driftmgr credentials list"
echo

if [[ -f "./bin/driftmgr" ]]; then
    if ./bin/driftmgr credentials list; then
        print_success "Credential listing test passed"
    else
        print_warning "Credential listing test failed"
    fi
elif [[ -f "./bin/driftmgr.exe" ]]; then
    if ./bin/driftmgr.exe credentials list; then
        print_success "Credential listing test passed"
    else
        print_warning "Credential listing test failed"
    fi
else
    print_warning "DriftMgr binary not found, skipping test"
fi

echo

# Test 4: Show help for credentials command
print_status "Test 4: Testing credentials help"
echo "Running: ./bin/driftmgr credentials help"
echo

if [[ -f "./bin/driftmgr" ]]; then
    if ./bin/driftmgr credentials help; then
        print_success "Credentials help test passed"
    else
        print_warning "Credentials help test failed"
    fi
elif [[ -f "./bin/driftmgr.exe" ]]; then
    if ./bin/driftmgr.exe credentials help; then
        print_success "Credentials help test passed"
    else
        print_warning "Credentials help test failed"
    fi
else
    print_warning "DriftMgr binary not found, skipping test"
fi

echo

print_success "Credential update functionality test completed!"
echo
echo "Usage Examples:"
echo "==============="
echo "1. Update credentials via install script:"
echo "   ./install.sh --update-credentials"
echo
echo "2. Update credentials via CLI:"
echo "   driftmgr credentials update"
echo
echo "3. List current credentials:"
echo "   driftmgr credentials list"
echo
echo "4. Setup new credentials:"
echo "   driftmgr credentials setup"
echo
echo "5. Validate specific provider:"
echo "   driftmgr credentials validate aws"
echo
echo "For more information, run: driftmgr credentials help"
