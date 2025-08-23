#!/usr/bin/env pwsh

# Multi-Subscription Test Runner
# This script runs the multi-subscription tests for driftmgr

Write-Host "=== DriftMgr Multi-Subscription Test Runner ===" -ForegroundColor Cyan
Write-Host ""

# Check if we're in the right directory
if (-not (Test-Path "go.mod")) {
    Write-Host "Error: go.mod not found. Please run this script from the driftmgr root directory." -ForegroundColor Red
    exit 1
}

# Build the project first
Write-Host "Building driftmgr..." -ForegroundColor Yellow
& go build -o bin/driftmgr.exe ./cmd/driftmgr
if ($LASTEXITCODE -ne 0) {
    Write-Host "Error: Failed to build driftmgr" -ForegroundColor Red
    exit 1
}

& go build -o bin/driftmgr-client.exe ./cmd/driftmgr-client
if ($LASTEXITCODE -ne 0) {
    Write-Host "Error: Failed to build driftmgr-client" -ForegroundColor Red
    exit 1
}

Write-Host "Build completed successfully!" -ForegroundColor Green
Write-Host ""

# Run the multi-subscription tests
Write-Host "Running multi-subscription tests..." -ForegroundColor Yellow
Write-Host ""

# Test 1: Multi-Subscription Discovery
Write-Host "Test 1: Multi-Subscription Discovery" -ForegroundColor Magenta
& go test -v ./tests -run TestMultiSubscriptionDiscovery -timeout 10m
$discoveryExitCode = $LASTEXITCODE

Write-Host ""
Write-Host "Test 1 completed with exit code: $discoveryExitCode" -ForegroundColor $(if ($discoveryExitCode -eq 0) { "Green" } else { "Red" })
Write-Host ""

# Test 2: Multi-Subscription Credential Management
Write-Host "Test 2: Multi-Subscription Credential Management" -ForegroundColor Magenta
& go test -v ./tests -run TestMultiSubscriptionCredentialManagement -timeout 5m
$credentialExitCode = $LASTEXITCODE

Write-Host ""
Write-Host "Test 2 completed with exit code: $credentialExitCode" -ForegroundColor $(if ($credentialExitCode -eq 0) { "Green" } else { "Red" })
Write-Host ""

# Test 3: Multi-Subscription Performance
Write-Host "Test 3: Multi-Subscription Performance" -ForegroundColor Magenta
& go test -v ./tests -run TestMultiSubscriptionPerformance -timeout 5m
$performanceExitCode = $LASTEXITCODE

Write-Host ""
Write-Host "Test 3 completed with exit code: $performanceExitCode" -ForegroundColor $(if ($performanceExitCode -eq 0) { "Green" } else { "Red" })
Write-Host ""

# Test 4: Multi-Subscription Integration
Write-Host "Test 4: Multi-Subscription Integration" -ForegroundColor Magenta
& go test -v ./tests -run TestMultiSubscriptionIntegration -timeout 2m
$integrationExitCode = $LASTEXITCODE

Write-Host ""
Write-Host "Test 4 completed with exit code: $integrationExitCode" -ForegroundColor $(if ($integrationExitCode -eq 0) { "Green" } else { "Red" })
Write-Host ""

# Summary
Write-Host "=== Test Summary ===" -ForegroundColor Cyan
Write-Host "Multi-Subscription Discovery:     $(if ($discoveryExitCode -eq 0) { "PASS" } else { "FAIL" })" -ForegroundColor $(if ($discoveryExitCode -eq 0) { "Green" } else { "Red" })
Write-Host "Credential Management:            $(if ($credentialExitCode -eq 0) { "PASS" } else { "FAIL" })" -ForegroundColor $(if ($credentialExitCode -eq 0) { "Green" } else { "Red" })
Write-Host "Performance Tests:               $(if ($performanceExitCode -eq 0) { "PASS" } else { "FAIL" })" -ForegroundColor $(if ($performanceExitCode -eq 0) { "Green" } else { "Red" })
Write-Host "Integration Tests:               $(if ($integrationExitCode -eq 0) { "PASS" } else { "FAIL" })" -ForegroundColor $(if ($integrationExitCode -eq 0) { "Green" } else { "Red" })
Write-Host ""

$totalExitCode = $discoveryExitCode + $credentialExitCode + $performanceExitCode + $integrationExitCode

if ($totalExitCode -eq 0) {
    Write-Host "All multi-subscription tests passed! ✓" -ForegroundColor Green
    Write-Host ""
    Write-Host "Multi-subscription support includes:" -ForegroundColor Cyan
    Write-Host "  • Account/Subscription discovery across all providers" -ForegroundColor White
    Write-Host "  • Account switching and credential management" -ForegroundColor White
    Write-Host "  • Cross-account resource discovery and comparison" -ForegroundColor White
    Write-Host "  • Performance testing with multiple accounts" -ForegroundColor White
    Write-Host "  • Integration with caching, concurrency, and security" -ForegroundColor White
} else {
    Write-Host "Some multi-subscription tests failed! ✗" -ForegroundColor Red
    Write-Host "Check the output above for details." -ForegroundColor Yellow
}

Write-Host ""
Write-Host "Test execution completed." -ForegroundColor Cyan
exit $totalExitCode
