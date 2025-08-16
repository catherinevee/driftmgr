# DriftMgr Comprehensive Test Runner (PowerShell)
# This script runs all types of tests including unit, integration, e2e, and benchmarks

param(
    [Parameter(Position=0)]
    [ValidateSet("unit", "integration", "e2e", "benchmarks", "security", "coverage", "clean", "help")]
    [string]$Command = ""
)

# Configuration
$TestTimeout = "10m"
$CoverageThreshold = 80
$BenchmarkIterations = 100
$ParallelTests = 4

# Directories
$UnitTestDir = "tests/unit"
$IntegrationTestDir = "tests"
$E2eTestDir = "tests/e2e"
$BenchmarkTestDir = "tests/benchmarks"
$CoverageDir = "coverage"

# Functions
function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host "[SUCCESS] $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "[WARNING] $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

function Test-Prerequisites {
    Write-Info "Checking prerequisites..."
    
    # Check if Go is installed
    try {
        $goVersion = go version
        if (-not $goVersion) {
            throw "Go not found"
        }
        Write-Info "Go version: $goVersion"
    }
    catch {
        Write-Error "Go is not installed or not in PATH"
        exit 1
    }
    
    # Check Go version
    $goVersionOutput = go version
    $goVersionMatch = [regex]::Match($goVersionOutput, 'go(\d+\.\d+)')
    if ($goVersionMatch.Success) {
        $goVersion = $goVersionMatch.Groups[1].Value
        $requiredVersion = "1.21"
        
        if ([version]$goVersion -lt [version]$requiredVersion) {
            Write-Error "Go version $goVersion is less than required version $requiredVersion"
            exit 1
        }
    }
    
    # Check if testify is installed
    try {
        $testifyPath = go list -f '{{.Dir}}' github.com/stretchr/testify 2>$null
        if (-not $testifyPath) {
            Write-Info "Installing testify..."
            go get github.com/stretchr/testify
        }
    }
    catch {
        Write-Info "Installing testify..."
        go get github.com/stretchr/testify
    }
    
    Write-Success "Prerequisites check passed"
}

function Setup-TestEnvironment {
    Write-Info "Setting up test environment..."
    
    # Create coverage directory
    if (-not (Test-Path $CoverageDir)) {
        New-Item -ItemType Directory -Path $CoverageDir -Force | Out-Null
    }
    
    # Set test environment variables
    $env:DRIFT_TEST_MODE = "true"
    $env:DRIFT_LOG_LEVEL = "debug"
    $env:DRIFT_TEST_TIMEOUT = $TestTimeout
    
    # Create test configuration
    $testConfig = @"
test:
  enabled: true
  timeout: $TestTimeout
  parallel: $ParallelTests
  coverage:
    enabled: true
    threshold: $CoverageThreshold
    output: $CoverageDir
  benchmarks:
    iterations: $BenchmarkIterations
    timeout: 5m
"@
    
    $testConfig | Out-File -FilePath "driftmgr.test.yaml" -Encoding UTF8
    
    Write-Success "Test environment setup complete"
}

function Invoke-UnitTests {
    Write-Info "Running unit tests..."
    
    $startTime = Get-Date
    $testPackages = Get-ChildItem -Path $UnitTestDir -Filter "*_test.go" -Recurse | ForEach-Object { $_.Directory.FullName } | Sort-Object -Unique
    
    if (-not $testPackages) {
        Write-Warning "No unit test packages found"
        return $true
    }
    
    $failedTests = 0
    
    foreach ($package in $testPackages) {
        Write-Info "Testing package: $package"
        
        $packageName = Split-Path $package -Leaf
        $coverageFile = Join-Path $CoverageDir "unit_$packageName.out"
        
        try {
            go test -v -timeout=$TestTimeout -coverprofile=$coverageFile $package
            if ($LASTEXITCODE -ne 0) {
                Write-Error "Unit tests failed in package: $package"
                $failedTests++
            }
        }
        catch {
            Write-Error "Unit tests failed in package: $package"
            $failedTests++
        }
    }
    
    $endTime = Get-Date
    $duration = ($endTime - $startTime).TotalSeconds
    
    if ($failedTests -eq 0) {
        Write-Success "Unit tests completed in ${duration}s"
        return $true
    }
    else {
        Write-Error "Unit tests failed: $failedTests packages"
        return $false
    }
}

function Invoke-IntegrationTests {
    Write-Info "Running integration tests..."
    
    $startTime = Get-Date
    
    try {
        $coverageFile = Join-Path $CoverageDir "integration.out"
        go test -v -timeout=$TestTimeout -coverprofile=$coverageFile ./tests
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Integration tests failed"
            return $false
        }
    }
    catch {
        Write-Error "Integration tests failed"
        return $false
    }
    
    $endTime = Get-Date
    $duration = ($endTime - $startTime).TotalSeconds
    
    Write-Success "Integration tests completed in ${duration}s"
    return $true
}

function Invoke-E2eTests {
    Write-Info "Running end-to-end tests..."
    
    $startTime = Get-Date
    
    # Check if e2e test directory exists
    if (-not (Test-Path $E2eTestDir)) {
        Write-Warning "E2E test directory not found: $E2eTestDir"
        return $true
    }
    
    $e2ePackages = Get-ChildItem -Path $E2eTestDir -Filter "*_test.go" -Recurse | ForEach-Object { $_.Directory.FullName } | Sort-Object -Unique
    
    if (-not $e2ePackages) {
        Write-Warning "No E2E test packages found"
        return $true
    }
    
    $failedTests = 0
    
    foreach ($package in $e2ePackages) {
        Write-Info "Running E2E tests in package: $package"
        
        $packageName = Split-Path $package -Leaf
        $coverageFile = Join-Path $CoverageDir "e2e_$packageName.out"
        
        try {
            go test -v -timeout=$TestTimeout -coverprofile=$coverageFile $package
            if ($LASTEXITCODE -ne 0) {
                Write-Error "E2E tests failed in package: $package"
                $failedTests++
            }
        }
        catch {
            Write-Error "E2E tests failed in package: $package"
            $failedTests++
        }
    }
    
    $endTime = Get-Date
    $duration = ($endTime - $startTime).TotalSeconds
    
    if ($failedTests -eq 0) {
        Write-Success "E2E tests completed in ${duration}s"
        return $true
    }
    else {
        Write-Error "E2E tests failed: $failedTests packages"
        return $false
    }
}

function Invoke-Benchmarks {
    Write-Info "Running benchmarks..."
    
    $startTime = Get-Date
    
    # Check if benchmark test directory exists
    if (-not (Test-Path $BenchmarkTestDir)) {
        Write-Warning "Benchmark test directory not found: $BenchmarkTestDir"
        return $true
    }
    
    $benchmarkPackages = Get-ChildItem -Path $BenchmarkTestDir -Filter "*_test.go" -Recurse | ForEach-Object { $_.Directory.FullName } | Sort-Object -Unique
    
    if (-not $benchmarkPackages) {
        Write-Warning "No benchmark test packages found"
        return $true
    }
    
    $failedBenchmarks = 0
    
    foreach ($package in $benchmarkPackages) {
        Write-Info "Running benchmarks in package: $package"
        
        $packageName = Split-Path $package -Leaf
        $coverageFile = Join-Path $CoverageDir "benchmark_$packageName.out"
        
        try {
            go test -v -bench=. -benchmem -timeout=$TestTimeout -coverprofile=$coverageFile $package
            if ($LASTEXITCODE -ne 0) {
                Write-Error "Benchmarks failed in package: $package"
                $failedBenchmarks++
            }
        }
        catch {
            Write-Error "Benchmarks failed in package: $package"
            $failedBenchmarks++
        }
    }
    
    $endTime = Get-Date
    $duration = ($endTime - $startTime).TotalSeconds
    
    if ($failedBenchmarks -eq 0) {
        Write-Success "Benchmarks completed in ${duration}s"
        return $true
    }
    else {
        Write-Error "Benchmarks failed: $failedBenchmarks packages"
        return $false
    }
}

function Invoke-SecurityTests {
    Write-Info "Running security tests..."
    
    $startTime = Get-Date
    
    # Run security-specific tests
    $securityTestPackages = Get-ChildItem -Path . -Filter "*security*_test.go" -Recurse | ForEach-Object { $_.Directory.FullName } | Sort-Object -Unique
    
    if (-not $securityTestPackages) {
        Write-Warning "No security test packages found"
        return $true
    }
    
    $failedTests = 0
    
    foreach ($package in $securityTestPackages) {
        Write-Info "Running security tests in package: $package"
        
        $packageName = Split-Path $package -Leaf
        $coverageFile = Join-Path $CoverageDir "security_$packageName.out"
        
        try {
            go test -v -timeout=$TestTimeout -coverprofile=$coverageFile $package
            if ($LASTEXITCODE -ne 0) {
                Write-Error "Security tests failed in package: $package"
                $failedTests++
            }
        }
        catch {
            Write-Error "Security tests failed in package: $package"
            $failedTests++
        }
    }
    
    $endTime = Get-Date
    $duration = ($endTime - $startTime).TotalSeconds
    
    if ($failedTests -eq 0) {
        Write-Success "Security tests completed in ${duration}s"
        return $true
    }
    else {
        Write-Error "Security tests failed: $failedTests packages"
        return $false
    }
}

function Invoke-StaticAnalysis {
    Write-Info "Running static analysis..."
    
    # Run golangci-lint if available
    try {
        $golangciLint = Get-Command golangci-lint -ErrorAction SilentlyContinue
        if ($golangciLint) {
            golangci-lint run
            if ($LASTEXITCODE -eq 0) {
                Write-Success "Static analysis passed"
            }
            else {
                Write-Error "Static analysis failed"
                return $false
            }
        }
        else {
            Write-Warning "golangci-lint not found, skipping static analysis"
        }
    }
    catch {
        Write-Warning "golangci-lint not found, skipping static analysis"
    }
    
    # Run go vet
    try {
        go vet ./...
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Go vet passed"
        }
        else {
            Write-Error "Go vet failed"
            return $false
        }
    }
    catch {
        Write-Error "Go vet failed"
        return $false
    }
    
    return $true
}

function Invoke-SecurityScan {
    Write-Info "Running security scan..."
    
    # Run gosec if available
    try {
        $gosec = Get-Command gosec -ErrorAction SilentlyContinue
        if ($gosec) {
            gosec ./...
            if ($LASTEXITCODE -eq 0) {
                Write-Success "Security scan passed"
            }
            else {
                Write-Warning "Security scan found issues"
            }
        }
        else {
            Write-Warning "gosec not found, skipping security scan"
        }
    }
    catch {
        Write-Warning "gosec not found, skipping security scan"
    }
}

function Generate-CoverageReport {
    Write-Info "Generating coverage report..."
    
    # Combine all coverage files
    if (Test-Path $CoverageDir) {
        $coverageFiles = Get-ChildItem -Path $CoverageDir -Filter "*.out" -File
        
        if ($coverageFiles) {
            # Create combined coverage file
            "mode: set" | Out-File -FilePath (Join-Path $CoverageDir "coverage.out") -Encoding UTF8
            
            foreach ($file in $coverageFiles) {
                if (Test-Path $file.FullName) {
                    Get-Content $file.FullName | Select-Object -Skip 1 | Add-Content -Path (Join-Path $CoverageDir "coverage.out") -ErrorAction SilentlyContinue
                }
            }
            
            # Generate HTML coverage report
            $coverageOutFile = Join-Path $CoverageDir "coverage.out"
            $coverageHtmlFile = Join-Path $CoverageDir "coverage.html"
            
            if (Test-Path $coverageOutFile) {
                go tool cover -html=$coverageOutFile -o=$coverageHtmlFile
                
                # Calculate coverage percentage
                $coverageOutput = go tool cover -func=$coverageOutFile
                $lastLine = $coverageOutput | Select-Object -Last 1
                $coverageMatch = [regex]::Match($lastLine, 'total:\s+\(statements\)\s+(\d+\.\d+)%')
                
                if ($coverageMatch.Success) {
                    $coveragePercent = [double]$coverageMatch.Groups[1].Value
                    Write-Info "Coverage: ${coveragePercent}%"
                    
                    if ($coveragePercent -ge $CoverageThreshold) {
                        Write-Success "Coverage threshold met: ${coveragePercent}% >= ${CoverageThreshold}%"
                    }
                    else {
                        Write-Warning "Coverage below threshold: ${coveragePercent}% < ${CoverageThreshold}%"
                    }
                }
            }
        }
        else {
            Write-Warning "No coverage files found"
        }
    }
    else {
        Write-Warning "Coverage directory not found"
    }
}

function Invoke-Cleanup {
    Write-Info "Cleaning up test artifacts..."
    
    # Remove test configuration
    if (Test-Path "driftmgr.test.yaml") {
        Remove-Item "driftmgr.test.yaml" -Force
    }
    
    # Clean test cache
    go clean -testcache
    
    Write-Success "Cleanup complete"
}

function Show-Summary {
    param([DateTime]$TotalStartTime)
    
    $totalEndTime = Get-Date
    $totalDuration = ($totalEndTime - $TotalStartTime).TotalSeconds
    
    Write-Host ""
    Write-Host "=========================================="
    Write-Host "           TEST SUMMARY"
    Write-Host "=========================================="
    Write-Host "Total Duration: ${totalDuration}s"
    Write-Host "Coverage Report: $CoverageDir/coverage.html"
    Write-Host "Test Configuration: driftmgr.test.yaml"
    Write-Host "=========================================="
    Write-Host ""
}

function Show-Help {
    Write-Host "Usage: .\run_comprehensive_tests.ps1 [command]"
    Write-Host ""
    Write-Host "Commands:"
    Write-Host "  unit        Run unit tests only"
    Write-Host "  integration Run integration tests only"
    Write-Host "  e2e         Run end-to-end tests only"
    Write-Host "  benchmarks  Run benchmarks only"
    Write-Host "  security    Run security tests only"
    Write-Host "  coverage    Generate coverage report only"
    Write-Host "  clean       Clean up test artifacts"
    Write-Host "  help        Show this help message"
    Write-Host ""
    Write-Host "If no command is specified, all tests will be run."
}

# Main execution
function Main {
    $totalStartTime = Get-Date
    
    Write-Host "=========================================="
    Write-Host "    DriftMgr Comprehensive Test Suite"
    Write-Host "=========================================="
    Write-Host ""
    
    # Check prerequisites
    Test-Prerequisites
    
    # Setup test environment
    Setup-TestEnvironment
    
    # Run static analysis
    if (-not (Invoke-StaticAnalysis)) {
        Write-Error "Static analysis failed"
        Invoke-Cleanup
        exit 1
    }
    
    # Run security scan
    Invoke-SecurityScan
    
    # Run unit tests
    if (-not (Invoke-UnitTests)) {
        Write-Error "Unit tests failed"
        Invoke-Cleanup
        exit 1
    }
    
    # Run integration tests
    if (-not (Invoke-IntegrationTests)) {
        Write-Error "Integration tests failed"
        Invoke-Cleanup
        exit 1
    }
    
    # Run E2E tests
    if (-not (Invoke-E2eTests)) {
        Write-Error "E2E tests failed"
        Invoke-Cleanup
        exit 1
    }
    
    # Run security tests
    if (-not (Invoke-SecurityTests)) {
        Write-Error "Security tests failed"
        Invoke-Cleanup
        exit 1
    }
    
    # Run benchmarks
    if (-not (Invoke-Benchmarks)) {
        Write-Error "Benchmarks failed"
        Invoke-Cleanup
        exit 1
    }
    
    # Generate coverage report
    Generate-CoverageReport
    
    # Cleanup
    Invoke-Cleanup
    
    # Print summary
    Show-Summary $totalStartTime
    
    Write-Success "All tests completed successfully!"
}

# Handle script arguments
switch ($Command) {
    "unit" {
        Test-Prerequisites
        Setup-TestEnvironment
        Invoke-UnitTests
        Generate-CoverageReport
        Invoke-Cleanup
    }
    "integration" {
        Test-Prerequisites
        Setup-TestEnvironment
        Invoke-IntegrationTests
        Generate-CoverageReport
        Invoke-Cleanup
    }
    "e2e" {
        Test-Prerequisites
        Setup-TestEnvironment
        Invoke-E2eTests
        Generate-CoverageReport
        Invoke-Cleanup
    }
    "benchmarks" {
        Test-Prerequisites
        Setup-TestEnvironment
        Invoke-Benchmarks
        Invoke-Cleanup
    }
    "security" {
        Test-Prerequisites
        Setup-TestEnvironment
        Invoke-SecurityTests
        Invoke-SecurityScan
        Invoke-Cleanup
    }
    "coverage" {
        Generate-CoverageReport
    }
    "clean" {
        Invoke-Cleanup
    }
    "help" {
        Show-Help
    }
    default {
        Main
    }
}
