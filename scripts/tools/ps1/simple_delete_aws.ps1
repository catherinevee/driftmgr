# Simple DriftMgr AWS Resource Deletion Script
param(
    [string]$AccountId = "",
    [switch]$Force,
    [switch]$DryRun = $true
)

$ServerURL = "http://localhost:8080"
$APIBaseURL = "$ServerURL/api/v1"

Write-Host "=== DriftMgr AWS Resource Deletion ===" -ForegroundColor Cyan

# Check if server is running
try {
    $response = Invoke-RestMethod -Uri "$ServerURL/health" -Method Get -TimeoutSec 5
    Write-Host "✓ Server is running" -ForegroundColor Green
} catch {
    Write-Host "✗ Server not running. Start with: .\driftmgr-server.exe" -ForegroundColor Red
    exit 1
}

# Get AWS Account ID
if (-not $AccountId) {
    try {
        $AccountId = (aws sts get-caller-identity --query 'Account' --output text).Trim()
        Write-Host "✓ Account ID: $AccountId" -ForegroundColor Green
    } catch {
        Write-Host "✗ Failed to get Account ID. Run: aws configure" -ForegroundColor Red
        exit 1
    }
}

# Preview deletion
Write-Host "`nPreviewing deletion..." -ForegroundColor Yellow

$request = @{
    provider = "aws"
    account_id = $AccountId
    options = @{
        dry_run = $true
        force = $Force
        resource_types = @()
        regions = @("us-east-1")
        timeout = "30m"
        batch_size = 10
    }
}

try {
    $body = $request | ConvertTo-Json -Depth 10
    $result = Invoke-RestMethod -Uri "$APIBaseURL/delete/preview" -Method Post -Body $body -ContentType "application/json"
    
    Write-Host "✓ Preview completed" -ForegroundColor Green
    Write-Host "  Total: $($result.total_resources)" -ForegroundColor White
    Write-Host "  Would delete: $($result.deleted_resources)" -ForegroundColor White
    Write-Host "  Would skip: $($result.skipped_resources)" -ForegroundColor White
    
} catch {
    Write-Host "✗ Preview failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Execute deletion if not dry-run
if (-not $DryRun) {
    Write-Host "`n⚠️  WARNING: This will DELETE ALL RESOURCES!" -ForegroundColor Red
    $confirm = Read-Host "Type 'YES DELETE ALL' to confirm"
    
    if ($confirm -eq "YES DELETE ALL") {
        Write-Host "Executing deletion..." -ForegroundColor Yellow
        
        $request.options.dry_run = $false
        $body = $request | ConvertTo-Json -Depth 10
        
        try {
            $result = Invoke-RestMethod -Uri "$APIBaseURL/delete/account" -Method Post -Body $body -ContentType "application/json"
            Write-Host "✓ Deletion completed" -ForegroundColor Green
            Write-Host "  Deleted: $($result.deleted_resources)" -ForegroundColor White
            Write-Host "  Failed: $($result.failed_resources)" -ForegroundColor White
        } catch {
            Write-Host "✗ Deletion failed: $($_.Exception.Message)" -ForegroundColor Red
        }
    } else {
        Write-Host "Deletion cancelled" -ForegroundColor Yellow
    }
} else {
    Write-Host "`n✓ Dry-run completed. No resources deleted." -ForegroundColor Green
    Write-Host "Run with: .\simple_delete_aws.ps1 -DryRun:`$false -Force" -ForegroundColor Yellow
}
