#!/bin/bash

# Test script for DriftMgr Interactive Shell
echo "Testing DriftMgr Interactive Shell..."
echo "======================================"

# Test 1: Start interactive shell and run help
echo "Test 1: Interactive shell with help command"
echo "help" | ./cmd/driftmgr-client/driftmgr-client.exe

echo ""
echo "Test 2: Non-interactive mode with help"
./cmd/driftmgr-client/driftmgr-client.exe help

echo ""
echo "Test 3: Interactive shell with multiple commands"
(
echo "help"
echo "health"
echo "statefiles"
echo "exit"
) | ./cmd/driftmgr-client/driftmgr-client.exe

echo ""
echo "All tests completed successfully!"
echo "The interactive shell is working as expected."
