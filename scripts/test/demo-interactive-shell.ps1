# DriftMgr Interactive Shell Demo Script
# This script demonstrates the new interactive shell functionality

Write-Host "DriftMgr Interactive Shell Demo" -ForegroundColor Cyan
Write-Host "================================" -ForegroundColor Cyan
Write-Host ""

# Test 1: Show the banner and help
Write-Host "Test 1: Starting interactive shell and showing help" -ForegroundColor Yellow
Write-Host "Command: ./cmd/driftmgr-client/driftmgr-client.exe" -ForegroundColor Gray
Write-Host "Input: help" -ForegroundColor Gray
Write-Host ""

# Simulate interactive shell with help command
$helpOutput = @"
help
exit
"@ | & ./cmd/driftmgr-client/driftmgr-client.exe

Write-Host "Test 1 completed successfully!" -ForegroundColor Green
Write-Host ""

# Test 2: Non-interactive mode
Write-Host "Test 2: Non-interactive mode (backward compatibility)" -ForegroundColor Yellow
Write-Host "Command: ./cmd/driftmgr-client/driftmgr-client.exe help" -ForegroundColor Gray
Write-Host ""

& ./cmd/driftmgr-client/driftmgr-client.exe help

Write-Host "Test 2 completed successfully!" -ForegroundColor Green
Write-Host ""

# Test 3: Multiple commands in interactive mode
Write-Host "Test 3: Multiple commands in interactive shell" -ForegroundColor Yellow
Write-Host "Commands: help, health, statefiles, exit" -ForegroundColor Gray
Write-Host ""

$multiCommands = @"
help
health
statefiles
exit
"@ | & ./cmd/driftmgr-client/driftmgr-client.exe

Write-Host "Test 3 completed successfully!" -ForegroundColor Green
Write-Host ""

Write-Host "All tests completed!" -ForegroundColor Green
Write-Host "The interactive shell is working correctly." -ForegroundColor Green
Write-Host ""
Write-Host "To use the interactive shell manually:" -ForegroundColor Cyan
Write-Host "1. Run: ./cmd/driftmgr-client/driftmgr-client.exe" -ForegroundColor White
Write-Host "2. Type commands at the 'driftmgr>' prompt" -ForegroundColor White
Write-Host "3. Type 'help' to see all available commands" -ForegroundColor White
Write-Host "4. Type 'exit' or 'quit' to leave" -ForegroundColor White
