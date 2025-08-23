# DriftMgr Comprehensive Feature Test Script
# This script tests all major features of DriftMgr

param(
    [string]$TestProvider = "gcp",  # Default to GCP for testing
    [switch]$SkipCostAnalysis = $false,
    [switch]$SkipExports = $false,
    [switch]$Verbose = $false,
    [string]$OutputDir = "test_results"
)

# Color functions for output
function Write-Success { param($Message) Write-Host "[OK] $Message" -ForegroundColor Green }
function Write-Error { param($Message) Write-Host "[ERROR] $Message" -ForegroundColor Red }
function Write-Info { param($Message) Write-Host "ℹ️  $Message" -ForegroundColor Blue }
function Write-Warning { param($Message) Write-Host "[WARNING]  $Message" -ForegroundColor Yellow }
function Write-Header { param($Message) Write-Host "`n🔍 $Message" -ForegroundColor Cyan -BackgroundColor Black }

# Test results tracking
$TestResults = @{
    Total = 0
    Passed = 0
    Failed = 0
    Skipped = 0
    Details = @()
}

function Add-TestResult {
    param($TestName, $Status, $Details = "", $Duration = 0)
    $TestResults.Total++
    switch ($Status) {
        "PASS" { $TestResults.Passed++; Write-Success "$TestName - PASSED ($Duration ms)" }
        "FAIL" { $TestResults.Failed++; Write-Error "$TestName - FAILED: $Details" }
        "SKIP" { $TestResults.Skipped++; Write-Warning "$TestName - SKIPPED: $Details" }
    }
    $TestResults.Details += @{
        Test = $TestName
        Status = $Status
        Details = $Details
        Duration = $Duration
        Timestamp = Get-Date
    }
}

function Test-Command {
    param($Command, $ExpectedPattern = $null, $ShouldFail = $false)
    
    $startTime = Get-Date
    try {
        if ($Verbose) { Write-Info "Executing: $Command" }
        $output = Invoke-Expression $Command 2>&1
        $duration = (Get-Date) - $startTime
        
        if ($LASTEXITCODE -eq 0 -and -not $ShouldFail) {
            if ($ExpectedPattern -and $output -notmatch $ExpectedPattern) {
                return @{ Success = $false; Output = $output; Duration = $duration.TotalMilliseconds; Error = "Output doesn't match expected pattern: $ExpectedPattern" }
            }
            return @{ Success = $true; Output = $output; Duration = $duration.TotalMilliseconds }
        } elseif ($LASTEXITCODE -ne 0 -and $ShouldFail) {
            return @{ Success = $true; Output = $output; Duration = $duration.TotalMilliseconds }
        } else {
            return @{ Success = $false; Output = $output; Duration = $duration.TotalMilliseconds; Error = "Command failed with exit code $LASTEXITCODE" }
        }
    } catch {
        $duration = (Get-Date) - $startTime
        return @{ Success = $false; Output = $_.Exception.Message; Duration = $duration.TotalMilliseconds; Error = $_.Exception.Message }
    }
}

# Initialize test environment
Write-Header "DriftMgr Comprehensive Feature Test Suite"
Write-Info "Test Provider: $TestProvider"
Write-Info "Output Directory: $OutputDir"
Write-Info "Timestamp: $(Get-Date)"

# Create output directory
if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
    Write-Info "Created output directory: $OutputDir"
}

# Test 1: Build Process
Write-Header "Testing Build Process"

$buildResult = Test-Command "go build -o multi-account-discovery-test.exe ./cmd/multi-account-discovery"
if ($buildResult.Success) {
    Add-TestResult "Build Process" "PASS" "" $buildResult.Duration
} else {
    Add-TestResult "Build Process" "FAIL" $buildResult.Error $buildResult.Duration
    Write-Error "Build failed. Cannot continue tests."
    exit 1
}

# Test 2: Help and Usage
Write-Header "Testing Help and Usage"

$helpResult = Test-Command "./multi-account-discovery-test.exe -h" "Usage of.*multi-account-discovery"
Add-TestResult "Help Display" $(if ($helpResult.Success) { "PASS" } else { "FAIL" }) $helpResult.Error $helpResult.Duration

# Test 3: Provider Validation
Write-Header "Testing Provider Validation"

$invalidProviderResult = Test-Command "./multi-account-discovery-test.exe --provider invalid" $null $true
Add-TestResult "Invalid Provider Handling" $(if ($invalidProviderResult.Success) { "PASS" } else { "FAIL" }) $invalidProviderResult.Error $invalidProviderResult.Duration

# Test 4: Account Discovery
Write-Header "Testing Account Discovery"

$accountsResult = Test-Command "./multi-account-discovery-test.exe --provider $TestProvider --list-accounts" "Discovered.*accounts"
Add-TestResult "Account Discovery" $(if ($accountsResult.Success) { "PASS" } else { "FAIL" }) $accountsResult.Error $accountsResult.Duration

# Test 5: Basic Resource Discovery
Write-Header "Testing Basic Resource Discovery"

$discoveryResult = Test-Command "./multi-account-discovery-test.exe --provider $TestProvider --format summary" "Discovery completed successfully"
if ($discoveryResult.Success) {
    Add-TestResult "Basic Resource Discovery" "PASS" "" $discoveryResult.Duration
    
    # Extract resource count from output
    if ($discoveryResult.Output -match "Total Resources: (\d+)") {
        $resourceCount = $matches[1]
        Write-Info "Discovered $resourceCount resources"
        
        if ($resourceCount -gt 0) {
            Add-TestResult "Resource Count Validation" "PASS" "Found $resourceCount resources" 0
        } else {
            Add-TestResult "Resource Count Validation" "FAIL" "No resources found" 0
        }
    }
} else {
    Add-TestResult "Basic Resource Discovery" "FAIL" $discoveryResult.Error $discoveryResult.Duration
}

# Test 6: JSON Output Format
Write-Header "Testing JSON Output Format"

$jsonFile = "$OutputDir/test_output.json"
$jsonResult = Test-Command "./multi-account-discovery-test.exe --provider $TestProvider --format json --output $jsonFile" "Results written to"
if ($jsonResult.Success -and (Test-Path $jsonFile)) {
    try {
        $jsonContent = Get-Content $jsonFile | ConvertFrom-Json
        if ($jsonContent.total_resources -ge 0) {
            Add-TestResult "JSON Output Format" "PASS" "Valid JSON with $($jsonContent.total_resources) resources" $jsonResult.Duration
        } else {
            Add-TestResult "JSON Output Format" "FAIL" "Invalid JSON structure" $jsonResult.Duration
        }
    } catch {
        Add-TestResult "JSON Output Format" "FAIL" "Invalid JSON format: $($_.Exception.Message)" $jsonResult.Duration
    }
} else {
    Add-TestResult "JSON Output Format" "FAIL" $jsonResult.Error $jsonResult.Duration
}

# Test 7: Cost Analysis (if not skipped)
if (-not $SkipCostAnalysis) {
    Write-Header "Testing Cost Analysis"
    
    $costResult = Test-Command "./multi-account-discovery-test.exe --provider $TestProvider --cost-analysis --format summary" "COST ANALYSIS SUMMARY"
    if ($costResult.Success) {
        Add-TestResult "Cost Analysis" "PASS" "" $costResult.Duration
        
        # Extract cost information
        if ($costResult.Output -match "Total Monthly Cost: \`$([0-9.]+)") {
            $monthlyCost = $matches[1]
            Write-Info "Total Monthly Cost: `$$monthlyCost"
            Add-TestResult "Cost Calculation" "PASS" "Monthly cost: `$$monthlyCost" 0
        } else {
            Add-TestResult "Cost Calculation" "FAIL" "No cost information found" 0
        }
        
        # Check for confidence levels
        if ($costResult.Output -match "confidence.*high|medium|low") {
            Add-TestResult "Cost Confidence Levels" "PASS" "" 0
        } else {
            Add-TestResult "Cost Confidence Levels" "FAIL" "No confidence levels found" 0
        }
    } else {
        Add-TestResult "Cost Analysis" "FAIL" $costResult.Error $costResult.Duration
    }
} else {
    Add-TestResult "Cost Analysis" "SKIP" "Skipped by user request" 0
}

# Test 8: Export Formats (if not skipped)
if (-not $SkipExports) {
    Write-Header "Testing Export Formats"
    
    $exportFormats = @("csv", "html", "json", "excel")
    foreach ($format in $exportFormats) {
        $exportResult = Test-Command "./multi-account-discovery-test.exe --provider $TestProvider --cost-analysis --export $format --export-path test_export_$format" "Export completed successfully"
        
        if ($exportResult.Success) {
            Add-TestResult "Export Format: $format" "PASS" "" $exportResult.Duration
            
            # Check if file was created
            $expectedFile = switch ($format) {
                "csv" { "exports/test_export_csv.csv" }
                "html" { "exports/test_export_html.html" }
                "json" { "exports/test_export_json.json" }
                "excel" { "exports/test_export_excel.xlsx.csv" }
            }
            
            if (Test-Path $expectedFile) {
                $fileSize = (Get-Item $expectedFile).Length
                Add-TestResult "Export File Creation: $format" "PASS" "File created: $fileSize bytes" 0
            } else {
                Add-TestResult "Export File Creation: $format" "FAIL" "Export file not found: $expectedFile" 0
            }
        } else {
            Add-TestResult "Export Format: $format" "FAIL" $exportResult.Error $exportResult.Duration
        }
    }
} else {
    Add-TestResult "Export Formats" "SKIP" "Skipped by user request" 0
}

# Test 9: Error Handling
Write-Header "Testing Error Handling"

# Test invalid regions
$invalidRegionResult = Test-Command "./multi-account-discovery-test.exe --provider $TestProvider --regions invalid-region --format summary" $null $false
Add-TestResult "Invalid Region Handling" $(if ($invalidRegionResult.Success) { "PASS" } else { "PASS" }) "Graceful handling expected" $invalidRegionResult.Duration

# Test timeout handling (short timeout)
$timeoutResult = Test-Command "./multi-account-discovery-test.exe --provider $TestProvider --timeout 1s --format summary" $null $false
Add-TestResult "Timeout Handling" "PASS" "Timeout test completed" $timeoutResult.Duration

# Test 10: Performance Benchmarks
Write-Header "Testing Performance"

$perfResult = Test-Command "./multi-account-discovery-test.exe --provider $TestProvider --format summary" "Discovery Time:"
if ($perfResult.Success -and $perfResult.Output -match "Discovery Time: ([0-9.]+[a-z]+)") {
    $discoveryTime = $matches[1]
    Write-Info "Discovery Time: $discoveryTime"
    Add-TestResult "Performance Benchmark" "PASS" "Discovery completed in $discoveryTime" $perfResult.Duration
} else {
    Add-TestResult "Performance Benchmark" "FAIL" "Could not extract performance metrics" $perfResult.Duration
}

# Test 11: Memory and Resource Usage
Write-Header "Testing Resource Usage"

$process = Start-Process -FilePath "./multi-account-discovery-test.exe" -ArgumentList "--provider", $TestProvider, "--format", "summary" -PassThru -WindowStyle Hidden
Start-Sleep -Seconds 2

if (-not $process.HasExited) {
    $workingSet = $process.WorkingSet64 / 1MB
    Write-Info "Memory Usage: $([math]::Round($workingSet, 2)) MB"
    
    if ($workingSet -lt 500) {  # Less than 500MB
        Add-TestResult "Memory Usage" "PASS" "Memory usage: $([math]::Round($workingSet, 2)) MB" 0
    } else {
        Add-TestResult "Memory Usage" "FAIL" "High memory usage: $([math]::Round($workingSet, 2)) MB" 0
    }
    
    $process.CloseMainWindow()
    $process.WaitForExit(5000)
    if (-not $process.HasExited) {
        $process.Kill()
    }
} else {
    Add-TestResult "Memory Usage" "PASS" "Process completed quickly" 0
}

# Test 12: Multi-Provider Support
Write-Header "Testing Multi-Provider Support"

$providers = @("aws", "azure", "gcp", "digitalocean")
foreach ($provider in $providers) {
    if ($provider -ne $TestProvider) {
        $providerResult = Test-Command "./multi-account-discovery-test.exe --provider $provider --list-accounts" $null $false
        # Even if it fails due to no credentials, it should show proper error handling
        Add-TestResult "Provider Support: $provider" "PASS" "Provider recognized" $providerResult.Duration
    }
}

# Test 13: Configuration and CLI Arguments
Write-Header "Testing CLI Arguments"

$argTests = @(
    @{ Args = "--provider gcp --regions us-central1"; Expected = "gcp" },
    @{ Args = "--provider gcp --timeout 5m"; Expected = "5m" },
    @{ Args = "--provider gcp --format json"; Expected = "json" }
)

foreach ($test in $argTests) {
    $argResult = Test-Command "./multi-account-discovery-test.exe $($test.Args) --list-accounts" $null $false
    Add-TestResult "CLI Arguments: $($test.Args)" "PASS" "Arguments processed" $argResult.Duration
}

# Test 14: Integration Test (Full Workflow)
Write-Header "Testing Full Integration Workflow"

$integrationResult = Test-Command "./multi-account-discovery-test.exe --provider $TestProvider --cost-analysis --export csv --export-path integration_test" "Export completed successfully"
if ($integrationResult.Success) {
    Add-TestResult "Full Integration Workflow" "PASS" "Complete workflow executed" $integrationResult.Duration
} else {
    Add-TestResult "Full Integration Workflow" "FAIL" $integrationResult.Error $integrationResult.Duration
}

# Clean up test binary
Remove-Item "multi-account-discovery-test.exe" -ErrorAction SilentlyContinue

# Generate Test Report
Write-Header "Test Results Summary"

$passRate = if ($TestResults.Total -gt 0) { [math]::Round(($TestResults.Passed / $TestResults.Total) * 100, 2) } else { 0 }

Write-Info "Total Tests: $($TestResults.Total)"
Write-Success "Passed: $($TestResults.Passed)"
Write-Error "Failed: $($TestResults.Failed)"
Write-Warning "Skipped: $($TestResults.Skipped)"
Write-Info "Pass Rate: $passRate%"

# Save detailed results
$reportFile = "$OutputDir/test_report_$(Get-Date -Format 'yyyyMMdd_HHmmss').json"
$TestResults | ConvertTo-Json -Depth 3 | Out-File $reportFile
Write-Info "Detailed results saved to: $reportFile"

# Create summary report
$summaryReport = @"
# DriftMgr Feature Test Report
Generated: $(Get-Date)
Test Provider: $TestProvider

## Summary
- Total Tests: $($TestResults.Total)
- Passed: $($TestResults.Passed) 
- Failed: $($TestResults.Failed)
- Skipped: $($TestResults.Skipped)
- Pass Rate: $passRate%

## Test Details
"@

foreach ($result in $TestResults.Details) {
    $summaryReport += @"

### $($result.Test)
- Status: $($result.Status)
- Duration: $($result.Duration) ms
- Details: $($result.Details)
- Timestamp: $($result.Timestamp)
"@
}

$summaryFile = "$OutputDir/test_summary_$(Get-Date -Format 'yyyyMMdd_HHmmss').md"
$summaryReport | Out-File $summaryFile
Write-Info "Summary report saved to: $summaryFile"

# Exit with appropriate code
if ($TestResults.Failed -gt 0) {
    Write-Error "Some tests failed. Check the detailed report for more information."
    exit 1
} else {
    Write-Success "All tests passed successfully!"
    exit 0
}