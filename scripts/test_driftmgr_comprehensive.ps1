# DriftMgr Comprehensive Testing Script
# Tests all driftmgr functions for errors and validates functionality
# This script performs both unit and integration tests

param(
    [switch]$Verbose = $false,
    [switch]$StopOnError = $false,
    [string]$TestCategory = "all"
)

# Test configuration
$script:TestResults = @{
    Passed = 0
    Failed = 0
    Skipped = 0
    Errors = @()
}

$script:DriftMgrPath = ".\driftmgr.exe"

# Color output functions
function Write-TestHeader {
    param([string]$Message)
    Write-Host "`n========================================" -ForegroundColor Cyan
    Write-Host $Message -ForegroundColor Cyan
    Write-Host "========================================" -ForegroundColor Cyan
}

function Write-TestSection {
    param([string]$Section)
    Write-Host "`n--- $Section ---" -ForegroundColor Yellow
}

function Write-TestPass {
    param([string]$Test)
    Write-Host "[PASS] $Test" -ForegroundColor Green
    $script:TestResults.Passed++
}

function Write-TestFail {
    param([string]$Test, [string]$Error)
    Write-Host "[FAIL] $Test" -ForegroundColor Red
    Write-Host "       Error: $Error" -ForegroundColor Red
    $script:TestResults.Failed++
    $script:TestResults.Errors += "$Test : $Error"
    
    if ($StopOnError) {
        Write-Host "`nStopping due to error (StopOnError flag is set)" -ForegroundColor Yellow
        Show-TestSummary
        exit 1
    }
}

function Write-TestSkip {
    param([string]$Test, [string]$Reason)
    Write-Host "[SKIP] $Test - $Reason" -ForegroundColor Yellow
    $script:TestResults.Skipped++
}

function Write-TestInfo {
    param([string]$Message)
    if ($Verbose) {
        Write-Host "       $Message" -ForegroundColor Gray
    }
}

# Test helper functions
function Test-Command {
    param(
        [string]$TestName,
        [string]$Command,
        [string]$ExpectedOutput = $null,
        [string]$NotExpectedOutput = $null,
        [int]$ExpectedExitCode = 0,
        [switch]$AllowNonZeroExit = $false
    )
    
    Write-Host "  Testing: $TestName" -NoNewline
    
    try {
        # Execute command
        $output = & cmd /c "$Command 2>&1"
        $exitCode = $LASTEXITCODE
        
        Write-TestInfo "Exit Code: $exitCode"
        if ($Verbose -and $output) {
            Write-TestInfo "Output: $($output -join ' ')"
        }
        
        # Check exit code
        if (-not $AllowNonZeroExit -and $exitCode -ne $ExpectedExitCode) {
            Write-TestFail $TestName "Expected exit code $ExpectedExitCode, got $exitCode"
            return $false
        }
        
        # Check expected output
        if ($ExpectedOutput) {
            $outputString = $output -join "`n"
            if ($outputString -notmatch [regex]::Escape($ExpectedOutput) -and 
                $outputString -notlike "*$ExpectedOutput*") {
                Write-TestFail $TestName "Expected output not found: '$ExpectedOutput'"
                return $false
            }
        }
        
        # Check not expected output
        if ($NotExpectedOutput) {
            $outputString = $output -join "`n"
            if ($outputString -match [regex]::Escape($NotExpectedOutput) -or 
                $outputString -like "*$NotExpectedOutput*") {
                Write-TestFail $TestName "Unexpected output found: '$NotExpectedOutput'"
                return $false
            }
        }
        
        Write-TestPass $TestName
        return $true
    }
    catch {
        Write-TestFail $TestName $_.Exception.Message
        return $false
    }
}

function Test-FileOperation {
    param(
        [string]$TestName,
        [string]$FilePath,
        [switch]$ShouldExist,
        [switch]$ShouldNotExist,
        [string]$ExpectedContent = $null
    )
    
    Write-Host "  Testing: $TestName" -NoNewline
    
    try {
        $exists = Test-Path $FilePath
        
        if ($ShouldExist -and -not $exists) {
            Write-TestFail $TestName "File should exist but doesn't: $FilePath"
            return $false
        }
        
        if ($ShouldNotExist -and $exists) {
            Write-TestFail $TestName "File should not exist but does: $FilePath"
            return $false
        }
        
        if ($ExpectedContent -and $exists) {
            $content = Get-Content $FilePath -Raw
            if ($content -notlike "*$ExpectedContent*") {
                Write-TestFail $TestName "Expected content not found in file"
                return $false
            }
        }
        
        Write-TestPass $TestName
        return $true
    }
    catch {
        Write-TestFail $TestName $_.Exception.Message
        return $false
    }
}

# Test Categories

function Test-BasicCommands {
    Write-TestSection "Basic Commands"
    
    # Test help command
    Test-Command -TestName "Help Command" `
        -Command "$script:DriftMgrPath --help" `
        -ExpectedOutput "Usage: driftmgr"
    
    Test-Command -TestName "Help Contains Core Commands" `
        -Command "$script:DriftMgrPath --help" `
        -ExpectedOutput "Core Commands"
    
    # Test version/status
    Test-Command -TestName "Status Command" `
        -Command "$script:DriftMgrPath status" `
        -ExpectedOutput "DriftMgr System Status"
    
    # Test unknown command
    Test-Command -TestName "Unknown Command Error" `
        -Command "$script:DriftMgrPath unknowncommand" `
        -AllowNonZeroExit `
        -ExpectedOutput "Unknown command"
}

function Test-CredentialDetection {
    Write-TestSection "Credential Detection"
    
    # Test credential status
    Test-Command -TestName "Show Credentials" `
        -Command "$script:DriftMgrPath discover --credentials" `
        -ExpectedOutput "Checking credential status"
    
    # Test status shows credentials
    Test-Command -TestName "Status Shows Credentials" `
        -Command "$script:DriftMgrPath status" `
        -ExpectedOutput "Cloud Credentials"
    
    # Check for at least one provider
    $output = & cmd /c "$script:DriftMgrPath status 2>&1"
    $hasProvider = $false
    
    @("AWS", "Azure", "GCP", "DigitalOcean") | ForEach-Object {
        if ($output -like "*$_*") {
            $hasProvider = $true
            Write-TestInfo "Found provider: $_"
        }
    }
    
    if ($hasProvider) {
        Write-TestPass "At least one provider detected"
    } else {
        Write-TestSkip "No cloud providers configured" "Configure at least one provider for full testing"
    }
}

function Test-DiscoveryCommands {
    Write-TestSection "Discovery Commands"
    
    # Test basic discovery
    Test-Command -TestName "Discovery Help" `
        -Command "$script:DriftMgrPath discover --help" `
        -ExpectedOutput "discover cloud resources"
    
    # Test discovery with invalid provider
    Test-Command -TestName "Invalid Provider Error" `
        -Command "$script:DriftMgrPath discover --provider invalid" `
        -AllowNonZeroExit
    
    # Test auto discovery
    Test-Command -TestName "Auto Discovery Flag" `
        -Command "$script:DriftMgrPath discover --auto --format json" `
        -ExpectedOutput "Auto-discovering"
    
    # Test discovery output formats
    @("json", "summary", "table") | ForEach-Object {
        Test-Command -TestName "Discovery Format: $_" `
            -Command "$script:DriftMgrPath discover --provider aws --format $_" `
            -AllowNonZeroExit
    }
}

function Test-DriftDetection {
    Write-TestSection "Drift Detection"
    
    # Test drift command structure
    Test-Command -TestName "Drift Help" `
        -Command "$script:DriftMgrPath drift --help" `
        -ExpectedOutput "drift"
    
    Test-Command -TestName "Drift Detect Help" `
        -Command "$script:DriftMgrPath drift detect --help" `
        -ExpectedOutput "detect"
    
    # Test drift detection without state file
    Test-Command -TestName "Drift Detect Without State" `
        -Command "$script:DriftMgrPath drift detect --provider aws" `
        -AllowNonZeroExit `
        -ExpectedOutput "state"
    
    # Test smart defaults
    Test-Command -TestName "Smart Defaults Flag" `
        -Command "$script:DriftMgrPath drift detect --smart-defaults --help" `
        -ExpectedOutput "Smart Defaults"
}

function Test-StateManagement {
    Write-TestSection "State Management"
    
    # Test state commands
    Test-Command -TestName "State Help" `
        -Command "$script:DriftMgrPath state --help" `
        -ExpectedOutput "state"
    
    Test-Command -TestName "State Discover" `
        -Command "$script:DriftMgrPath state discover" `
        -AllowNonZeroExit
    
    # Test state visualization
    Test-Command -TestName "State Visualize Help" `
        -Command "$script:DriftMgrPath state visualize --help" `
        -ExpectedOutput "visualize"
}

function Test-AccountManagement {
    Write-TestSection "Account Management"
    
    # Test account listing
    Test-Command -TestName "List Accounts" `
        -Command "$script:DriftMgrPath accounts" `
        -AllowNonZeroExit
    
    # Test use command
    Test-Command -TestName "Use Command Help" `
        -Command "$script:DriftMgrPath use --help" `
        -ExpectedOutput "Select"
    
    # Test use with all flag
    Test-Command -TestName "Use All Flag" `
        -Command "$script:DriftMgrPath use --all" `
        -ExpectedOutput "Available"
}

function Test-ExportImport {
    Write-TestSection "Export/Import Commands"
    
    $testExportFile = "test_export_$(Get-Random).json"
    
    # Test export
    Test-Command -TestName "Export Help" `
        -Command "$script:DriftMgrPath export --help" `
        -ExpectedOutput "Export"
    
    # Test export to file
    Test-Command -TestName "Export to JSON" `
        -Command "$script:DriftMgrPath export --format json --output $testExportFile" `
        -AllowNonZeroExit
    
    # Clean up test file
    if (Test-Path $testExportFile) {
        Remove-Item $testExportFile -Force
        Write-TestInfo "Cleaned up test export file"
    }
    
    # Test import
    Test-Command -TestName "Import Help" `
        -Command "$script:DriftMgrPath import --help" `
        -ExpectedOutput "Import"
}

function Test-VerifyCommand {
    Write-TestSection "Verify Command"
    
    Test-Command -TestName "Verify Help" `
        -Command "$script:DriftMgrPath verify --help" `
        -ExpectedOutput "Verify"
    
    Test-Command -TestName "Verify Execution" `
        -Command "$script:DriftMgrPath verify" `
        -AllowNonZeroExit
}

function Test-ServerCommands {
    Write-TestSection "Server Commands"
    
    Test-Command -TestName "Serve Help" `
        -Command "$script:DriftMgrPath serve --help" `
        -ExpectedOutput "Start"
    
    Test-Command -TestName "Serve Web Help" `
        -Command "$script:DriftMgrPath serve web --help" `
        -ExpectedOutput "web"
}

function Test-DeleteCommand {
    Write-TestSection "Delete Command"
    
    Test-Command -TestName "Delete Help" `
        -Command "$script:DriftMgrPath delete --help" `
        -ExpectedOutput "Delete"
    
    # Test delete with dry-run (safe)
    Test-Command -TestName "Delete Dry Run" `
        -Command "$script:DriftMgrPath delete --resource-id test-123 --dry-run" `
        -AllowNonZeroExit
}

function Test-ErrorHandling {
    Write-TestSection "Error Handling"
    
    # Test invalid flags
    Test-Command -TestName "Invalid Flag Error" `
        -Command "$script:DriftMgrPath --invalidflag" `
        -AllowNonZeroExit `
        -ExpectedOutput "Unknown"
    
    # Test missing required arguments
    Test-Command -TestName "Missing Required Args" `
        -Command "$script:DriftMgrPath discover --provider" `
        -AllowNonZeroExit
    
    # Test invalid file paths
    Test-Command -TestName "Invalid File Path" `
        -Command "$script:DriftMgrPath export --output /invalid:/path/file.json" `
        -AllowNonZeroExit
}

function Test-ColorAndProgress {
    Write-TestSection "Color and Progress Features"
    
    # Test with NO_COLOR
    $env:NO_COLOR = "1"
    Test-Command -TestName "NO_COLOR Environment" `
        -Command "$script:DriftMgrPath status" `
        -NotExpectedOutput "[31m"  # Should not contain ANSI color codes
    $env:NO_COLOR = ""
    
    # Test with FORCE_COLOR
    $env:FORCE_COLOR = "1"
    Test-Command -TestName "FORCE_COLOR Environment" `
        -Command "$script:DriftMgrPath status"
    $env:FORCE_COLOR = ""
}

function Test-ConfigurationFiles {
    Write-TestSection "Configuration Files"
    
    # Check for config files
    @(
        "configs\config.yaml",
        "configs\smart-defaults.yaml",
        "configs\driftmgr.yaml"
    ) | ForEach-Object {
        Test-FileOperation -TestName "Config File: $_" `
            -FilePath $_ `
            -ShouldExist
    }
}

function Test-BuildArtifacts {
    Write-TestSection "Build Artifacts"
    
    # Check main executable exists
    Test-FileOperation -TestName "Main Executable" `
        -FilePath $script:DriftMgrPath `
        -ShouldExist
    
    # Test executable runs
    Test-Command -TestName "Executable Runs" `
        -Command "$script:DriftMgrPath" `
        -ExpectedOutput "driftmgr"
}

function Test-EdgeCases {
    Write-TestSection "Edge Cases"
    
    # Test with very long arguments
    $longArg = "a" * 1000
    Test-Command -TestName "Very Long Argument" `
        -Command "$script:DriftMgrPath discover --provider $longArg" `
        -AllowNonZeroExit
    
    # Test special characters in arguments
    Test-Command -TestName "Special Characters" `
        -Command "$script:DriftMgrPath export --output `"test file with spaces.json`"" `
        -AllowNonZeroExit
    
    # Test Unicode in arguments
    Test-Command -TestName "Unicode Characters" `
        -Command "$script:DriftMgrPath export --output test_😀.json" `
        -AllowNonZeroExit
}

function Test-Integration {
    Write-TestSection "Integration Tests"
    
    # Test command chaining
    Write-Host "  Testing: Command Chaining" -NoNewline
    
    $statusOutput = & cmd /c "$script:DriftMgrPath status 2>&1"
    if ($statusOutput -like "*configured*") {
        $discoverOutput = & cmd /c "$script:DriftMgrPath discover --auto 2>&1"
        if ($discoverOutput) {
            Write-TestPass "Command Chaining"
        } else {
            Write-TestFail "Command Chaining" "Discovery after status failed"
        }
    } else {
        Write-TestSkip "Command Chaining" "No providers configured"
    }
    
    # Test multiple format outputs
    $formats = @("json", "summary", "table")
    $allFormatsWork = $true
    
    foreach ($format in $formats) {
        $output = & cmd /c "$script:DriftMgrPath discover --format $format 2>&1"
        if ($LASTEXITCODE -ne 0 -and $output -notlike "*No credentials*") {
            $allFormatsWork = $false
            Write-TestInfo "Format $format failed"
        }
    }
    
    if ($allFormatsWork) {
        Write-TestPass "Multiple Output Formats"
    } else {
        Write-TestFail "Multiple Output Formats" "Some formats failed"
    }
}

function Test-Performance {
    Write-TestSection "Performance Tests"
    
    # Test help command performance
    Write-Host "  Testing: Help Command Performance" -NoNewline
    $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
    & cmd /c "$script:DriftMgrPath --help 2>&1" | Out-Null
    $stopwatch.Stop()
    
    if ($stopwatch.ElapsedMilliseconds -lt 1000) {
        Write-TestPass "Help Command Performance (<1s)"
        Write-TestInfo "Completed in $($stopwatch.ElapsedMilliseconds)ms"
    } else {
        Write-TestFail "Help Command Performance" "Took $($stopwatch.ElapsedMilliseconds)ms (>1s)"
    }
    
    # Test status command performance
    Write-Host "  Testing: Status Command Performance" -NoNewline
    $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
    & cmd /c "$script:DriftMgrPath status 2>&1" | Out-Null
    $stopwatch.Stop()
    
    if ($stopwatch.ElapsedMilliseconds -lt 5000) {
        Write-TestPass "Status Command Performance (<5s)"
        Write-TestInfo "Completed in $($stopwatch.ElapsedMilliseconds)ms"
    } else {
        Write-TestFail "Status Command Performance" "Took $($stopwatch.ElapsedMilliseconds)ms (>5s)"
    }
}

function Show-TestSummary {
    Write-TestHeader "Test Summary"
    
    $total = $script:TestResults.Passed + $script:TestResults.Failed + $script:TestResults.Skipped
    $passRate = if ($total -gt 0) { [math]::Round(($script:TestResults.Passed / $total) * 100, 2) } else { 0 }
    
    Write-Host "Total Tests: $total" -ForegroundColor White
    Write-Host "Passed: $($script:TestResults.Passed)" -ForegroundColor Green
    Write-Host "Failed: $($script:TestResults.Failed)" -ForegroundColor $(if ($script:TestResults.Failed -eq 0) { "Green" } else { "Red" })
    Write-Host "Skipped: $($script:TestResults.Skipped)" -ForegroundColor Yellow
    Write-Host "Pass Rate: $passRate%" -ForegroundColor $(if ($passRate -ge 80) { "Green" } elseif ($passRate -ge 60) { "Yellow" } else { "Red" })
    
    if ($script:TestResults.Failed -gt 0) {
        Write-Host "`nFailed Tests:" -ForegroundColor Red
        $script:TestResults.Errors | ForEach-Object {
            Write-Host "  - $_" -ForegroundColor Red
        }
    }
    
    # Exit with appropriate code
    if ($script:TestResults.Failed -gt 0) {
        exit 1
    }
}

# Main execution
function Run-AllTests {
    Write-TestHeader "DriftMgr Comprehensive Test Suite"
    Write-Host "Testing: $script:DriftMgrPath"
    Write-Host "Verbose: $Verbose"
    Write-Host "StopOnError: $StopOnError"
    Write-Host "Category: $TestCategory"
    
    # Check if driftmgr exists
    if (-not (Test-Path $script:DriftMgrPath)) {
        Write-Host "`nERROR: DriftMgr executable not found at $script:DriftMgrPath" -ForegroundColor Red
        Write-Host "Please build the project first: go build -o driftmgr.exe ./cmd/driftmgr" -ForegroundColor Yellow
        exit 1
    }
    
    # Run test categories
    $testCategories = @{
        "basic" = { Test-BasicCommands }
        "credentials" = { Test-CredentialDetection }
        "discovery" = { Test-DiscoveryCommands }
        "drift" = { Test-DriftDetection }
        "state" = { Test-StateManagement }
        "accounts" = { Test-AccountManagement }
        "export" = { Test-ExportImport }
        "verify" = { Test-VerifyCommand }
        "server" = { Test-ServerCommands }
        "delete" = { Test-DeleteCommand }
        "errors" = { Test-ErrorHandling }
        "color" = { Test-ColorAndProgress }
        "config" = { Test-ConfigurationFiles }
        "build" = { Test-BuildArtifacts }
        "edge" = { Test-EdgeCases }
        "integration" = { Test-Integration }
        "performance" = { Test-Performance }
    }
    
    if ($TestCategory -eq "all") {
        foreach ($category in $testCategories.Keys) {
            & $testCategories[$category]
        }
    } elseif ($testCategories.ContainsKey($TestCategory)) {
        & $testCategories[$TestCategory]
    } else {
        Write-Host "Invalid test category: $TestCategory" -ForegroundColor Red
        Write-Host "Available categories: $($testCategories.Keys -join ', ')" -ForegroundColor Yellow
        exit 1
    }
    
    Show-TestSummary
}

# Run the tests
Run-AllTests