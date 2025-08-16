# DriftMgr Test Script (PowerShell)
# This script runs tests for the DriftMgr application

param(
    [switch]$Unit,
    [switch]$Integration,
    [switch]$Benchmark,
    [switch]$All
)

# Set error action preference
$ErrorActionPreference = "Stop"

# Colors for output
$Green = "Green"
$Blue = "Blue"
$Red = "Red"
$Yellow = "Yellow"

# Function to print colored output
function Write-Status {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor $Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host "[SUCCESS] $Message" -ForegroundColor $Green
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor $Red
}

function Write-Warning {
    param([string]$Message)
    Write-Host "[WARNING] $Message" -ForegroundColor $Yellow
}

Write-Status "Starting DriftMgr tests..."

# If no specific test type is specified, run all
if (-not $Unit -and -not $Integration -and -not $Benchmark) {
    $All = $true
}

$TestResults = @()

# Run unit tests
if ($Unit -or $All) {
    Write-Status "Running unit tests..."
    try {
        $UnitResult = go test ./internal/... -v
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Unit tests passed"
            $TestResults += "Unit tests: PASSED"
        } else {
            Write-Error "Unit tests failed"
            $TestResults += "Unit tests: FAILED"
        }
    } catch {
        Write-Error "Failed to run unit tests: $_"
        $TestResults += "Unit tests: ERROR"
    }
}

# Run integration tests
if ($Integration -or $All) {
    Write-Status "Running integration tests..."
    try {
        if (Test-Path "tests/integration") {
            $IntegrationResult = go test ./tests/integration/... -v
            if ($LASTEXITCODE -eq 0) {
                Write-Success "Integration tests passed"
                $TestResults += "Integration tests: PASSED"
            } else {
                Write-Error "Integration tests failed"
                $TestResults += "Integration tests: FAILED"
            }
        } else {
            Write-Warning "Integration tests directory not found, skipping"
            $TestResults += "Integration tests: SKIPPED"
        }
    } catch {
        Write-Error "Failed to run integration tests: $_"
        $TestResults += "Integration tests: ERROR"
    }
}

# Run benchmarks
if ($Benchmark -or $All) {
    Write-Status "Running benchmarks..."
    try {
        if (Test-Path "tests/benchmarks") {
            $BenchmarkResult = go test ./tests/benchmarks/... -bench=.
            if ($LASTEXITCODE -eq 0) {
                Write-Success "Benchmarks completed"
                $TestResults += "Benchmarks: PASSED"
            } else {
                Write-Error "Benchmarks failed"
                $TestResults += "Benchmarks: FAILED"
            }
        } else {
            Write-Warning "Benchmarks directory not found, skipping"
            $TestResults += "Benchmarks: SKIPPED"
        }
    } catch {
        Write-Error "Failed to run benchmarks: $_"
        $TestResults += "Benchmarks: ERROR"
    }
}

# Summary
Write-Status "Test Summary:"
foreach ($result in $TestResults) {
    Write-Host "  $result"
}

# Check if all tests passed
$FailedTests = $TestResults | Where-Object { $_ -like "*FAILED*" -or $_ -like "*ERROR*" }
if ($FailedTests) {
    Write-Error "Some tests failed!"
    exit 1
} else {
    Write-Success "All tests passed!"
}
