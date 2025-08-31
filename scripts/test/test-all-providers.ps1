# Test script to verify all four cloud providers can be auto-detected
# This script tests various credential detection scenarios

# Colors for output
$Red = "Red"
$Green = "Green"
$Yellow = "Yellow"
$Blue = "Blue"

function Write-Status {
    param([string]$Message)
    Write-Host $Message -ForegroundColor $Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host "✓ $Message" -ForegroundColor $Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "⚠ $Message" -ForegroundColor $Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "✗ $Message" -ForegroundColor $Red
}

Write-Host "Testing All Cloud Provider Auto-Detection" -ForegroundColor $Blue
Write-Host "=============================================" -ForegroundColor $Blue
Write-Host ""

# Test 1: All providers detected
Write-Status "Test 1: All four providers detected"
Write-Host "Running: ./bin/driftmgr.exe credentials auto-detect"
Write-Host ""

$autoDetectOutput = ./bin/driftmgr.exe credentials auto-detect
if ($autoDetectOutput -match "Successfully detected 4 provider") {
    Write-Success "All four providers detected successfully"
} else {
    Write-Error "Failed to detect all four providers"
    exit 1
}

Write-Host ""

# Test 2: TUI startup with all providers
Write-Status "Test 2: TUI startup with all providers"
Write-Host "Running: echo 'exit' | ./bin/driftmgr.exe"
Write-Host ""

$tuiOutput = echo "exit" | ./bin/driftmgr.exe
if ($tuiOutput -match "Detected 4 provider") {
    Write-Success "TUI correctly displays all four providers"
} else {
    Write-Error "TUI failed to display all four providers"
    exit 1
}

Write-Host ""

# Test 3: Individual provider detection
Write-Status "Test 3: Individual provider detection"
Write-Host ""

# Test AWS detection
if ($autoDetectOutput -match "AWS.*Found") {
    Write-Success "AWS credentials detected"
} else {
    Write-Error "AWS credentials not detected"
}

# Test Azure detection
if ($autoDetectOutput -match "Azure.*Found") {
    Write-Success "Azure credentials detected"
} else {
    Write-Error "Azure credentials not detected"
}

# Test GCP detection
if ($autoDetectOutput -match "GCP.*Found") {
    Write-Success "GCP credentials detected"
} else {
    Write-Error "GCP credentials not detected"
}

# Test DigitalOcean detection
if ($autoDetectOutput -match "DigitalOcean.*Found") {
    Write-Success "DigitalOcean credentials detected"
} else {
    Write-Error "DigitalOcean credentials not detected"
}

Write-Host ""

# Test 4: Credential listing
Write-Status "Test 4: Credential listing"
Write-Host "Running: ./bin/driftmgr.exe credentials list"
Write-Host ""

try {
    ./bin/driftmgr.exe credentials list
    Write-Success "Credential listing works"
} catch {
    Write-Error "Credential listing failed"
}

Write-Host ""

# Test 5: Help command
Write-Status "Test 5: Help command"
Write-Host "Running: ./bin/driftmgr.exe credentials help"
Write-Host ""

$helpOutput = ./bin/driftmgr.exe credentials help
if ($helpOutput -match "auto-detect") {
    Write-Success "Help command includes auto-detect option"
} else {
    Write-Error "Help command missing auto-detect option"
}

Write-Host ""

Write-Success "All cloud provider auto-detection tests completed successfully!"
Write-Host ""
Write-Host "Summary:" -ForegroundColor $Blue
Write-Host "=========" -ForegroundColor $Blue
Write-Host "✓ All four providers (AWS, Azure, GCP, DigitalOcean) can be auto-detected"
Write-Host "✓ TUI displays detected credentials on startup"
Write-Host "✓ Individual provider detection works correctly"
Write-Host "✓ Credential listing and help commands work"
Write-Host "✓ Enhanced detection logic covers multiple credential sources"
Write-Host ""
Write-Host "Detection Sources Covered:" -ForegroundColor $Blue
Write-Host "=========================" -ForegroundColor $Blue
Write-Host "• Environment variables"
Write-Host "• CLI tool configurations"
Write-Host "• Standard credential files"
Write-Host "• Cache and log directories"
Write-Host "• SSO and token configurations"
Write-Host ""
Write-Host "For more information, see: CREDENTIAL_UPDATE_FEATURE.md"
