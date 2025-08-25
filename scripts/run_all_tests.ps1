# DriftMgr Test Runner Script
# Runs all available tests for DriftMgr

param(
    [switch]$Quick = $false,     # Run only quick tests
    [switch]$Full = $false,       # Run all tests including slow ones
    [switch]$Coverage = $false,   # Generate coverage report
    [switch]$Verbose = $false     # Verbose output
)

$ErrorActionPreference = "Continue"

Write-Host "`n=====================================" -ForegroundColor Cyan
Write-Host "     DriftMgr Test Runner" -ForegroundColor Cyan
Write-Host "=====================================" -ForegroundColor Cyan

# Check if Go is installed
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "`nError: Go is not installed or not in PATH" -ForegroundColor Red
    exit 1
}

# Build the project first
Write-Host "`n1. Building DriftMgr..." -ForegroundColor Yellow
$buildResult = go build -o driftmgr.exe ./cmd/driftmgr 2>&1

if ($LASTEXITCODE -ne 0) {
    Write-Host "Build failed!" -ForegroundColor Red
    Write-Host $buildResult
    exit 1
} else {
    Write-Host "Build successful!" -ForegroundColor Green
}

# Run Go tests
Write-Host "`n2. Running Go Unit Tests..." -ForegroundColor Yellow

$testCommand = "go test"
if ($Coverage) {
    $testCommand += " -cover -coverprofile=coverage.out"
}
if ($Verbose) {
    $testCommand += " -v"
}
if ($Quick) {
    $testCommand += " -short"
}
$testCommand += " ./..."

Write-Host "Command: $testCommand" -ForegroundColor Gray
Invoke-Expression $testCommand

if ($LASTEXITCODE -ne 0) {
    Write-Host "Some Go tests failed!" -ForegroundColor Red
} else {
    Write-Host "All Go tests passed!" -ForegroundColor Green
}

# Generate coverage report if requested
if ($Coverage -and (Test-Path coverage.out)) {
    Write-Host "`nGenerating coverage report..." -ForegroundColor Yellow
    go tool cover -html=coverage.out -o coverage.html
    Write-Host "Coverage report saved to coverage.html" -ForegroundColor Green
}

# Run functional tests
Write-Host "`n3. Running Functional Tests..." -ForegroundColor Yellow

$functionalTestPath = "tests/functional"
if (Test-Path $functionalTestPath) {
    Push-Location $functionalTestPath
    
    $testCommand = "go test"
    if ($Verbose) {
        $testCommand += " -v"
    }
    if ($Quick) {
        $testCommand += " -short"
    }
    $testCommand += " ."
    
    Write-Host "Command: $testCommand" -ForegroundColor Gray
    Invoke-Expression $testCommand
    
    Pop-Location
    
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Some functional tests failed!" -ForegroundColor Red
    } else {
        Write-Host "All functional tests passed!" -ForegroundColor Green
    }
} else {
    Write-Host "Functional test directory not found, skipping..." -ForegroundColor Yellow
}

# Run comprehensive CLI tests
Write-Host "`n4. Running Comprehensive CLI Tests..." -ForegroundColor Yellow

$cliTestScript = ".\scripts\test_driftmgr_comprehensive.ps1"
if (Test-Path $cliTestScript) {
    $params = @{}
    if ($Verbose) {
        $params['Verbose'] = $true
    }
    if ($Quick) {
        $params['TestCategory'] = 'basic'
    }
    
    & $cliTestScript @params
    
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Some CLI tests failed!" -ForegroundColor Red
    } else {
        Write-Host "All CLI tests passed!" -ForegroundColor Green
    }
} else {
    Write-Host "CLI test script not found, skipping..." -ForegroundColor Yellow
}

# Run specific feature tests if Full mode
if ($Full) {
    Write-Host "`n5. Running Full Test Suite..." -ForegroundColor Yellow
    
    # Test progress indicators
    Write-Host "  Testing progress indicators..." -ForegroundColor Gray
    if (Test-Path "examples/progress_demo.go") {
        go run examples/progress_demo.go
    }
    
    # Test color support
    Write-Host "  Testing color support..." -ForegroundColor Gray
    if (Test-Path "examples/color_demo.go") {
        go run examples/color_demo.go
    }
    
    # Run benchmarks
    Write-Host "  Running benchmarks..." -ForegroundColor Gray
    go test -bench=. -benchtime=10s ./tests/functional
}

# Summary
Write-Host "`n=====================================" -ForegroundColor Cyan
Write-Host "         Test Summary" -ForegroundColor Cyan
Write-Host "=====================================" -ForegroundColor Cyan

# Check for test failures
$testsFailed = $false

# Check Go test results
$goTestResults = go test -json ./... 2>$null | ConvertFrom-Json -ErrorAction SilentlyContinue
if ($goTestResults) {
    $failedTests = $goTestResults | Where-Object { $_.Action -eq "fail" }
    if ($failedTests) {
        Write-Host "Go Tests: FAILED" -ForegroundColor Red
        $testsFailed = $true
    } else {
        Write-Host "Go Tests: PASSED" -ForegroundColor Green
    }
} else {
    Write-Host "Go Tests: UNKNOWN" -ForegroundColor Yellow
}

# Check if executable works
Write-Host "`nSmoke Test:" -ForegroundColor White
.\driftmgr.exe --help | Out-Null
if ($LASTEXITCODE -eq 0) {
    Write-Host "  DriftMgr executable: OK" -ForegroundColor Green
} else {
    Write-Host "  DriftMgr executable: FAILED" -ForegroundColor Red
    $testsFailed = $true
}

# Final result
Write-Host "`n=====================================" -ForegroundColor Cyan
if ($testsFailed) {
    Write-Host "     TESTS FAILED" -ForegroundColor Red
    Write-Host "=====================================" -ForegroundColor Cyan
    exit 1
} else {
    Write-Host "   ALL TESTS PASSED!" -ForegroundColor Green
    Write-Host "=====================================" -ForegroundColor Cyan
    exit 0
}