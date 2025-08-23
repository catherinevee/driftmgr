# Simple DriftMgr Verification Test

Write-Host "DriftMgr Basic Verification Test"
Write-Host "=================================="

# Test AWS CLI
Write-Host "`nTesting AWS CLI..."
try {
    $awsResult = aws --version
    Write-Host "AWS CLI is available: $awsResult"
    
    # Test AWS credentials
    $awsIdentity = aws sts get-caller-identity 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "AWS credentials are configured"
        $identity = $awsIdentity | ConvertFrom-Json
        Write-Host "Account: $($identity.Account)"
    }
    else {
        Write-Host "AWS credentials not configured"
    }
}
catch {
    Write-Host "AWS CLI not available"
}

# Test Azure CLI
Write-Host "`nTesting Azure CLI..."
try {
    $azResult = az --version
    Write-Host "Azure CLI is available"
    
    # Test Azure credentials
    $azAccount = az account show 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Azure credentials are configured"
        $account = $azAccount | ConvertFrom-Json
        Write-Host "Subscription: $($account.name)"
    }
    else {
        Write-Host "Azure credentials not configured"
    }
}
catch {
    Write-Host "Azure CLI not available"
}

# Test DriftMgr executable
Write-Host "`nTesting DriftMgr executable..."
if (Test-Path "./driftmgr.exe") {
    Write-Host "DriftMgr executable found"
    
    try {
        $driftmgrHelp = ./driftmgr.exe --help 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Host "DriftMgr executable is working"
        }
        else {
            Write-Host "DriftMgr executable failed"
        }
    }
    catch {
        Write-Host "DriftMgr executable error"
    }
}
else {
    Write-Host "DriftMgr executable not found"
}

Write-Host "`nTest completed!"
