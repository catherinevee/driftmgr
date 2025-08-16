# Test DriftMgr deletion
Write-Host "Testing DriftMgr deletion..." -ForegroundColor Cyan

# Check server
try {
    $response = Invoke-RestMethod -Uri "http://localhost:8080/health" -Method Get
    Write-Host "Server is running" -ForegroundColor Green
} catch {
    Write-Host "Server not running" -ForegroundColor Red
    exit 1
}

# Get account ID
try {
    $accountId = (aws sts get-caller-identity --query 'Account' --output text).Trim()
    Write-Host "Account ID: $accountId" -ForegroundColor Green
} catch {
    Write-Host "Failed to get account ID" -ForegroundColor Red
    exit 1
}

# Preview deletion
$request = @{
    provider = "aws"
    account_id = $accountId
    options = @{
        dry_run = $true
        force = $false
        resource_types = @()
        regions = @("us-east-1")
        timeout = "30m"
        batch_size = 10
    }
}

try {
    $body = $request | ConvertTo-Json -Depth 10
    $result = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/delete/preview" -Method Post -Body $body -ContentType "application/json"
    Write-Host "Preview successful" -ForegroundColor Green
    Write-Host "Total resources: $($result.total_resources)" -ForegroundColor White
} catch {
    Write-Host "Preview failed: $($_.Exception.Message)" -ForegroundColor Red
}
