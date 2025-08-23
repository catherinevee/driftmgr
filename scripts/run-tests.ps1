# DriftMgr Test Runner Script
# Provides easy execution of different test suites

param(
    [Parameter(Mandatory=$false)]
    [ValidateSet("all", "e2e", "integration", "benchmarks", "performance", "unit", "coverage")]
    [string]$TestType = "all",
    
    [Parameter(Mandatory=$false)]
    [switch]$Verbose,
    
    [Parameter(Mandatory=$false)]
    [switch]$Short,
    
    [Parameter(Mandatory=$false)]
    [switch]$Coverage,
    
    [Parameter(Mandatory=$false)]
    [int]$BenchmarkCount = 1,
    
    [Parameter(Mandatory=$false)]
    [string]$OutputDir = "./test-results"
)

# Color output functions
function Write-Success { param($Message) Write-Host $Message -ForegroundColor Green }
function Write-Error { param($Message) Write-Host $Message -ForegroundColor Red }
function Write-Warning { param($Message) Write-Host $Message -ForegroundColor Yellow }
function Write-Info { param($Message) Write-Host $Message -ForegroundColor Cyan }

# Ensure we're in the project root
$projectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $projectRoot

# Create output directory
if (!(Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
}

# Build test flags
$testFlags = @()
if ($Verbose) { $testFlags += "-v" }
if ($Short) { $testFlags += "-short" }
if ($Coverage) { $testFlags += "-coverprofile=$OutputDir/coverage.out" }

Write-Info "DriftMgr Test Runner"
Write-Info "===================="
Write-Info "Test Type: $TestType"
Write-Info "Output Directory: $OutputDir"
Write-Info "Flags: $($testFlags -join ' ')"
Write-Info ""

# Function to run tests with error handling
function Invoke-TestCommand {
    param(
        [string]$Command,
        [string]$Description
    )
    
    Write-Info "Running: $Description"
    Write-Host "Command: $Command" -ForegroundColor Gray
    
    $startTime = Get-Date
    
    try {
        Invoke-Expression $Command
        $exitCode = $LASTEXITCODE
        
        $endTime = Get-Date
        $duration = $endTime - $startTime
        
        if ($exitCode -eq 0) {
            Write-Success "✓ $Description completed successfully (Duration: $($duration.ToString('mm\:ss')))"
        } else {
            Write-Error "✗ $Description failed with exit code $exitCode"
            return $false
        }
    }
    catch {
        Write-Error "✗ $Description failed with exception: $($_.Exception.Message)"
        return $false
    }
    
    return $true
}

# Function to check Go installation
function Test-GoInstallation {
    try {
        $goVersion = go version
        Write-Info "Go installation: $goVersion"
        return $true
    }
    catch {
        Write-Error "Go is not installed or not in PATH"
        return $false
    }
}

# Function to download dependencies
function Install-Dependencies {
    Write-Info "Installing/updating dependencies..."
    $success = Invoke-TestCommand "go mod download" "Download dependencies"
    if ($success) {
        $success = Invoke-TestCommand "go mod tidy" "Clean up dependencies"
    }
    return $success
}

# Main test execution logic
function Start-Tests {
    # Check prerequisites
    if (!(Test-GoInstallation)) {
        exit 1
    }
    
    # Install dependencies
    if (!(Install-Dependencies)) {
        Write-Error "Failed to install dependencies"
        exit 1
    }
    
    $allSuccessful = $true
    
    switch ($TestType) {
        "all" {
            Write-Info "Running all test suites..."
            
            # Run unit tests first (if they exist)
            if (Test-Path "./internal/*_test.go") {
                $success = Invoke-TestCommand "go test $($testFlags -join ' ') ./internal/..." "Unit tests"
                $allSuccessful = $allSuccessful -and $success
            }
            
            # Run integration tests
            $success = Invoke-TestCommand "go test $($testFlags -join ' ') ./tests/integration/..." "Integration tests"
            $allSuccessful = $allSuccessful -and $success
            
            # Run end-to-end tests
            $success = Invoke-TestCommand "go test $($testFlags -join ' ') ./tests/e2e/..." "End-to-end tests"
            $allSuccessful = $allSuccessful -and $success
            
            # Run performance tests (non-benchmark)
            $success = Invoke-TestCommand "go test $($testFlags -join ' ') -run=TestPerformance ./tests/benchmarks/..." "Performance tests"
            $allSuccessful = $allSuccessful -and $success
        }
        
        "e2e" {
            $success = Invoke-TestCommand "go test $($testFlags -join ' ') ./tests/e2e/..." "End-to-end tests"
            $allSuccessful = $allSuccessful -and $success
        }
        
        "integration" {
            $success = Invoke-TestCommand "go test $($testFlags -join ' ') ./tests/integration/..." "Integration tests"
            $allSuccessful = $allSuccessful -and $success
        }
        
        "benchmarks" {
            $benchFlags = "-bench=. -benchmem"
            if ($BenchmarkCount -gt 1) {
                $benchFlags += " -count=$BenchmarkCount"
            }
            
            $command = "go test $benchFlags ./tests/benchmarks/... | Tee-Object $OutputDir/benchmark-results.txt"
            $success = Invoke-TestCommand $command "Benchmark tests"
            $allSuccessful = $allSuccessful -and $success
            
            if ($success -and (Test-Path "$OutputDir/benchmark-results.txt")) {
                Write-Info "Benchmark results saved to: $OutputDir/benchmark-results.txt"
            }
        }
        
        "performance" {
            $success = Invoke-TestCommand "go test $($testFlags -join ' ') -run=TestPerformance ./tests/benchmarks/..." "Performance tests"
            $allSuccessful = $allSuccessful -and $success
            
            # Also run stress tests
            $success = Invoke-TestCommand "go test $($testFlags -join ' ') -run=TestStress ./tests/benchmarks/..." "Stress tests"
            $allSuccessful = $allSuccessful -and $success
        }
        
        "unit" {
            if (Test-Path "./internal/*_test.go") {
                $success = Invoke-TestCommand "go test $($testFlags -join ' ') ./internal/..." "Unit tests"
                $allSuccessful = $allSuccessful -and $success
            } else {
                Write-Warning "No unit tests found in ./internal/"
            }
        }
        
        "coverage" {
            Write-Info "Running tests with coverage analysis..."
            
            # Run tests with coverage
            $coverageFlags = $testFlags + @("-coverprofile=$OutputDir/coverage.out")
            $success = Invoke-TestCommand "go test $($coverageFlags -join ' ') ./..." "Tests with coverage"
            
            if ($success -and (Test-Path "$OutputDir/coverage.out")) {
                # Generate HTML coverage report
                $success = Invoke-TestCommand "go tool cover -html=$OutputDir/coverage.out -o $OutputDir/coverage.html" "Generate HTML coverage report"
                
                if ($success) {
                    Write-Success "Coverage report generated: $OutputDir/coverage.html"
                    
                    # Show coverage summary
                    Write-Info "Coverage summary:"
                    go tool cover -func="$OutputDir/coverage.out" | Select-Object -Last 1
                }
            }
            
            $allSuccessful = $allSuccessful -and $success
        }
    }
    
    # Final summary
    Write-Info ""
    Write-Info "Test Execution Summary"
    Write-Info "======================"
    
    if ($allSuccessful) {
        Write-Success "✓ All tests completed successfully!"
        
        # Show additional information
        if (Test-Path "$OutputDir/coverage.out") {
            Write-Info "Coverage report available at: $OutputDir/coverage.html"
        }
        
        if (Test-Path "$OutputDir/benchmark-results.txt") {
            Write-Info "Benchmark results available at: $OutputDir/benchmark-results.txt"
        }
        
        # Show test artifacts
        $artifacts = Get-ChildItem $OutputDir -ErrorAction SilentlyContinue
        if ($artifacts) {
            Write-Info "Test artifacts:"
            $artifacts | ForEach-Object { Write-Host "  - $($_.Name)" -ForegroundColor Gray }
        }
        
        exit 0
    } else {
        Write-Error "✗ Some tests failed. Check the output above for details."
        exit 1
    }
}

# Additional helper functions
function Show-TestHelp {
    Write-Host @"
DriftMgr Test Runner

Usage: ./scripts/run-tests.ps1 [options]

Test Types:
  -TestType all          Run all test suites (default)
  -TestType e2e          Run end-to-end tests only
  -TestType integration  Run integration tests only
  -TestType benchmarks   Run benchmark tests only
  -TestType performance  Run performance tests only
  -TestType unit         Run unit tests only
  -TestType coverage     Run tests with coverage analysis

Options:
  -Verbose               Enable verbose test output
  -Short                 Run tests in short mode (skip long-running tests)
  -Coverage              Generate coverage report
  -BenchmarkCount N      Run benchmarks N times (default: 1)
  -OutputDir PATH        Specify output directory for results (default: ./test-results)

Examples:
  # Run all tests
  ./scripts/run-tests.ps1

  # Run integration tests with verbose output
  ./scripts/run-tests.ps1 -TestType integration -Verbose

  # Run benchmarks 5 times
  ./scripts/run-tests.ps1 -TestType benchmarks -BenchmarkCount 5

  # Generate coverage report
  ./scripts/run-tests.ps1 -TestType coverage

  # Run short tests only
  ./scripts/run-tests.ps1 -Short

Environment Variables:
  SKIP_CLOUD_TESTS=true     Skip tests requiring cloud credentials
  DRIFTMGR_TEST_DEBUG=true  Enable debug logging in tests
  DRIFTMGR_TEST_TMPDIR      Custom temporary directory for test data

"@ -ForegroundColor Yellow
}

# Handle help requests
if ($args -contains "-h" -or $args -contains "--help" -or $args -contains "help") {
    Show-TestHelp
    exit 0
}

# Start the test execution
try {
    Start-Tests
}
catch {
    Write-Error "Test execution failed with error: $($_.Exception.Message)"
    Write-Error $_.ScriptStackTrace
    exit 1
}