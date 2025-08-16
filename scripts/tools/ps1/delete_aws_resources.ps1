# DriftMgr AWS Resource Deletion Script
# This script demonstrates how to safely delete all resources in your AWS account using driftmgr

param(
    [string]$AccountId = "",
    [string]$Region = "us-east-1",
    [switch]$Force,
    [switch]$DryRun = $true
)

# Configuration
$ServerURL = "http://localhost:8080"
$APIBaseURL = "$ServerURL/api/v1"

Write-Host "=== DriftMgr AWS Resource Deletion Script ===" -ForegroundColor Cyan
Write-Host ""

# Check if server is running
Write-Host "Checking if DriftMgr server is running..." -ForegroundColor Yellow
try {
    $response = Invoke-RestMethod -Uri "$ServerURL/health" -Method Get -TimeoutSec 5
    Write-Host "✓ DriftMgr server is running" -ForegroundColor Green
} catch {
    Write-Host "✗ DriftMgr server is not running. Please start it first:" -ForegroundColor Red
    Write-Host "  .\driftmgr-server.exe" -ForegroundColor Yellow
    exit 1
}

# Get AWS Account ID if not provided
if (-not $AccountId) {
    Write-Host "Getting AWS Account ID..." -ForegroundColor Yellow
    try {
        $accountInfo = aws sts get-caller-identity --query 'Account' --output text
        $AccountId = $accountInfo.Trim()
        Write-Host "✓ AWS Account ID: $AccountId" -ForegroundColor Green
    } catch {
        Write-Host "✗ Failed to get AWS Account ID. Please ensure AWS CLI is configured." -ForegroundColor Red
        Write-Host "  Run: aws configure" -ForegroundColor Yellow
        exit 1
    }
}

# Step 1: Get supported providers
Write-Host ""
Write-Host "Step 1: Checking supported providers..." -ForegroundColor Cyan
try {
    $providers = Invoke-RestMethod -Uri "$APIBaseURL/delete/providers" -Method Get
    Write-Host "✓ Supported providers: $($providers.providers -join ', ')" -ForegroundColor Green
} catch {
    Write-Host "✗ Failed to get supported providers: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Step 2: Preview deletion (DRY RUN) - ALWAYS DO THIS FIRST
Write-Host ""
Write-Host "Step 2: Previewing deletion (DRY RUN)..." -ForegroundColor Cyan
Write-Host "This will show you what resources would be deleted WITHOUT actually deleting them." -ForegroundColor Yellow

$previewRequest = @{
    provider = "aws"
    account_id = $AccountId
    options = @{
        dry_run = $true
        force = $Force
        resource_types = @()
        regions = @($Region)
        exclude_resources = @()
        include_resources = @()
        timeout = "30m"
        batch_size = 10
    }
}

try {
    $previewBody = $previewRequest | ConvertTo-Json -Depth 10
    $previewResult = Invoke-RestMethod -Uri "$APIBaseURL/delete/preview" -Method Post -Body $previewBody -ContentType "application/json"
    
    Write-Host "✓ Preview completed successfully" -ForegroundColor Green
    Write-Host "  Total resources found: $($previewResult.total_resources)" -ForegroundColor White
    Write-Host "  Resources that would be deleted: $($previewResult.deleted_resources)" -ForegroundColor White
    Write-Host "  Resources that would be skipped: $($previewResult.skipped_resources)" -ForegroundColor White
    
    if ($previewResult.errors -and $previewResult.errors.Count -gt 0) {
        Write-Host "  Errors during preview:" -ForegroundColor Yellow
        foreach ($error in $previewResult.errors) {
            Write-Host "    - $($error.resource_id) ($($error.resource_type)): $($error.error)" -ForegroundColor Yellow
        }
    }
    
    if ($previewResult.warnings -and $previewResult.warnings.Count -gt 0) {
        Write-Host "  Warnings:" -ForegroundColor Yellow
        foreach ($warning in $previewResult.warnings) {
            Write-Host "    - $warning" -ForegroundColor Yellow
        }
    }
    
} catch {
    Write-Host "✗ Preview failed: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.Exception.Response) {
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        $responseBody = $reader.ReadToEnd()
        Write-Host "Response: $responseBody" -ForegroundColor Red
    }
    exit 1
}

# Step 3: Ask for confirmation if not in dry-run mode
if (-not $DryRun) {
    Write-Host ""
    Write-Host "⚠️  WARNING: You are about to DELETE ALL RESOURCES in AWS Account $AccountId" -ForegroundColor Red
    Write-Host "This action is IRREVERSIBLE and will result in data loss!" -ForegroundColor Red
    Write-Host ""
    
    $confirmation = Read-Host "Are you absolutely sure you want to proceed? Type 'YES DELETE ALL' to confirm"
    
    if ($confirmation -ne "YES DELETE ALL") {
        Write-Host "Deletion cancelled by user." -ForegroundColor Yellow
        exit 0
    }
    
    # Step 4: Execute actual deletion
    Write-Host ""
    Write-Host "Step 4: Executing actual deletion..." -ForegroundColor Cyan
    Write-Host "This will actually delete the resources. Progress will be shown below." -ForegroundColor Yellow
    
    $deletionRequest = @{
        provider = "aws"
        account_id = $AccountId
        options = @{
            dry_run = $false
            force = $Force
            resource_types = @()
            regions = @($Region)
            exclude_resources = @()
            include_resources = @()
            timeout = "30m"
            batch_size = 10
        }
    }
    
    try {
        $deletionBody = $deletionRequest | ConvertTo-Json -Depth 10
        $deletionResult = Invoke-RestMethod -Uri "$APIBaseURL/delete/account" -Method Post -Body $deletionBody -ContentType "application/json"
        
        Write-Host "✓ Deletion completed successfully" -ForegroundColor Green
        Write-Host "  Total resources processed: $($deletionResult.total_resources)" -ForegroundColor White
        Write-Host "  Resources deleted: $($deletionResult.deleted_resources)" -ForegroundColor White
        Write-Host "  Resources failed: $($deletionResult.failed_resources)" -ForegroundColor White
        Write-Host "  Resources skipped: $($deletionResult.skipped_resources)" -ForegroundColor White
        Write-Host "  Duration: $($deletionResult.duration)" -ForegroundColor White
        
        if ($deletionResult.errors -and $deletionResult.errors.Count -gt 0) {
            Write-Host "  Errors during deletion:" -ForegroundColor Yellow
            foreach ($error in $deletionResult.errors) {
                Write-Host "    - $($error.resource_id) ($($error.resource_type)): $($error.error)" -ForegroundColor Yellow
            }
        }
        
    } catch {
        Write-Host "✗ Deletion failed: $($_.Exception.Message)" -ForegroundColor Red
        if ($_.Exception.Response) {
            $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
            $responseBody = $reader.ReadToEnd()
            Write-Host "Response: $responseBody" -ForegroundColor Red
        }
        exit 1
    }
} else {
    Write-Host ""
    Write-Host "✓ Dry-run mode completed. No resources were actually deleted." -ForegroundColor Green
    Write-Host ""
    Write-Host "To perform actual deletion, run this script with:" -ForegroundColor Yellow
    Write-Host "  .\delete_aws_resources.ps1 -DryRun:`$false -Force" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "⚠️  WARNING: Actual deletion will permanently delete all resources!" -ForegroundColor Red
}

Write-Host ""
Write-Host "=== Script completed ===" -ForegroundColor Cyan
