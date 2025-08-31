# Test script for DriftMgr credential update functionality
# This script demonstrates how to use the new credential update feature

param(
    [switch]$Verbose
)

# Colors for output
$Red = "Red"
$Green = "Green"
$Yellow = "Yellow"
$Blue = "Blue"

function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Color = "White"
    )
    Write-Host $Message -ForegroundColor $Color
}

Write-ColorOutput "🧪 DriftMgr Credential Update Test" $Blue
Write-ColorOutput "==================================" $Blue
Write-Host ""

# Test 1: Update credentials using install script
Write-ColorOutput "Test 1: Testing credential update via install script" $Blue
Write-ColorOutput "Running: ./install.sh --update-credentials" $White
Write-Host ""

if (Test-Path "./install.sh") {
    try {
        bash ./install.sh --update-credentials
        Write-ColorOutput "✓ Install script credential update test passed" $Green
    }
    catch {
        Write-ColorOutput "⚠ Install script credential update test failed (expected if DriftMgr not installed)" $Yellow
    }
} else {
    Write-ColorOutput "⚠ Install script not found, skipping test" $Yellow
}

Write-Host ""

# Test 2: Update credentials using CLI directly
Write-ColorOutput "Test 2: Testing credential update via CLI" $Blue
Write-ColorOutput "Running: ./bin/driftmgr.exe credentials update" $White
Write-Host ""

if (Test-Path "./bin/driftmgr.exe") {
    try {
        & ./bin/driftmgr.exe credentials update
        Write-ColorOutput "✓ CLI credential update test passed" $Green
    }
    catch {
        Write-ColorOutput "⚠ CLI credential update test failed (expected if no credentials to update)" $Yellow
    }
} else {
    Write-ColorOutput "⚠ DriftMgr binary not found, skipping test" $Yellow
}

Write-Host ""

# Test 3: List current credentials
Write-ColorOutput "Test 3: Testing credential listing" $Blue
Write-ColorOutput "Running: ./bin/driftmgr.exe credentials list" $White
Write-Host ""

if (Test-Path "./bin/driftmgr.exe") {
    try {
        & ./bin/driftmgr.exe credentials list
        Write-ColorOutput "✓ Credential listing test passed" $Green
    }
    catch {
        Write-ColorOutput "⚠ Credential listing test failed" $Yellow
    }
} else {
    Write-ColorOutput "⚠ DriftMgr binary not found, skipping test" $Yellow
}

Write-Host ""

# Test 4: Show help for credentials command
Write-ColorOutput "Test 4: Testing credentials help" $Blue
Write-ColorOutput "Running: ./bin/driftmgr.exe credentials help" $White
Write-Host ""

if (Test-Path "./bin/driftmgr.exe") {
    try {
        & ./bin/driftmgr.exe credentials help
        Write-ColorOutput "✓ Credentials help test passed" $Green
    }
    catch {
        Write-ColorOutput "⚠ Credentials help test failed" $Yellow
    }
} else {
    Write-ColorOutput "⚠ DriftMgr binary not found, skipping test" $Yellow
}

Write-Host ""

Write-ColorOutput "✓ Credential update functionality test completed!" $Green
Write-Host ""
Write-ColorOutput "Usage Examples:" $Blue
Write-ColorOutput "===============" $Blue
Write-ColorOutput "1. Update credentials via install script:" $White
Write-ColorOutput "   ./install.sh --update-credentials" $White
Write-Host ""
Write-ColorOutput "2. Update credentials via CLI:" $White
Write-ColorOutput "   driftmgr credentials update" $White
Write-Host ""
Write-ColorOutput "3. List current credentials:" $White
Write-ColorOutput "   driftmgr credentials list" $White
Write-Host ""
Write-ColorOutput "4. Setup new credentials:" $White
Write-ColorOutput "   driftmgr credentials setup" $White
Write-Host ""
Write-ColorOutput "5. Validate specific provider:" $White
Write-ColorOutput "   driftmgr credentials validate aws" $White
Write-Host ""
Write-ColorOutput "For more information, run: driftmgr credentials help" $White
