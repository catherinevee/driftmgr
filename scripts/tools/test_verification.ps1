# Simple DriftMgr Verification Test
# This script tests basic AWS and Azure CLI functionality

Write-Host "🔍 DriftMgr Basic Verification Test"
Write-Host "=" * 50

# Test AWS CLI
Write-Host "`n🔐 Testing AWS CLI..."
try {
    $awsResult = aws --version
    Write-Host "[OK] AWS CLI is available: $awsResult"
    
    # Test AWS credentials
    $awsIdentity = aws sts get-caller-identity 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "[OK] AWS credentials are configured"
        $identity = $awsIdentity | ConvertFrom-Json
        Write-Host "   Account: $($identity.Account)"
        Write-Host "   User: $($identity.Arn)"
    }
    else {
        Write-Host "[ERROR] AWS credentials not configured: $awsIdentity"
    }
}
catch {
    Write-Host "[ERROR] AWS CLI not available: $($_.Exception.Message)"
}

# Test Azure CLI
Write-Host "`n🔐 Testing Azure CLI..."
try {
    $azResult = az --version
    Write-Host "[OK] Azure CLI is available"
    
    # Test Azure credentials
    $azAccount = az account show 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "[OK] Azure credentials are configured"
        $account = $azAccount | ConvertFrom-Json
        Write-Host "   Subscription: $($account.name)"
        Write-Host "   ID: $($account.id)"
    }
    else {
        Write-Host "[ERROR] Azure credentials not configured: $azAccount"
    }
}
catch {
    Write-Host "[ERROR] Azure CLI not available: $($_.Exception.Message)"
}

# Test DriftMgr executable
Write-Host "`n[LIGHTNING] Testing DriftMgr executable..."
if (Test-Path "./driftmgr.exe") {
    Write-Host "[OK] DriftMgr executable found"
    
    try {
        $driftmgrHelp = ./driftmgr.exe --help 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Host "[OK] DriftMgr executable is working"
        }
        else {
            Write-Host "[ERROR] DriftMgr executable failed: $driftmgrHelp"
        }
    }
    catch {
        Write-Host "[ERROR] DriftMgr executable error: $($_.Exception.Message)"
    }
}
else {
    Write-Host "[ERROR] DriftMgr executable not found at ./driftmgr.exe"
}

# Test basic AWS resource discovery
Write-Host "`n🔍 Testing AWS resource discovery..."
try {
    # Get S3 buckets
    $s3Buckets = aws s3api list-buckets --query "Buckets[*].Name" --output json 2>&1
    if ($LASTEXITCODE -eq 0) {
        $buckets = $s3Buckets | ConvertFrom-Json
        Write-Host "[OK] Found $($buckets.Count) S3 buckets"
    }
    else {
        Write-Host "[ERROR] Failed to list S3 buckets: $s3Buckets"
    }
    
    # Get EC2 instances in us-east-1
    $ec2Instances = aws ec2 describe-instances --region us-east-1 --query "Reservations[*].Instances[*]" --output json 2>&1
    if ($LASTEXITCODE -eq 0) {
        $instances = $ec2Instances | ConvertFrom-Json
        $totalInstances = 0
        foreach ($reservation in $instances) {
            $totalInstances += $reservation.Count
        }
        Write-Host "[OK] Found $totalInstances EC2 instances in us-east-1"
    }
    else {
        Write-Host "[ERROR] Failed to list EC2 instances: $ec2Instances"
    }
}
catch {
    Write-Host "[ERROR] AWS resource discovery failed: $($_.Exception.Message)"
}

# Test basic Azure resource discovery
Write-Host "`n🔍 Testing Azure resource discovery..."
try {
    # Get VMs
    $vms = az vm list --query "[].name" --output json 2>&1
    if ($LASTEXITCODE -eq 0) {
        $vmList = $vms | ConvertFrom-Json
        Write-Host "[OK] Found $($vmList.Count) Azure VMs"
    }
    else {
        Write-Host "[ERROR] Failed to list Azure VMs: $vms"
    }
    
    # Get Storage Accounts
    $storageAccounts = az storage account list --query "[].name" --output json 2>&1
    if ($LASTEXITCODE -eq 0) {
        $saList = $storageAccounts | ConvertFrom-Json
        Write-Host "[OK] Found $($saList.Count) Azure Storage Accounts"
    }
    else {
        Write-Host "[ERROR] Failed to list Azure Storage Accounts: $storageAccounts"
    }
}
catch {
    Write-Host "[ERROR] Azure resource discovery failed: $($_.Exception.Message)"
}

Write-Host "`n" + ("=" * 50)
Write-Host "[OK] Basic verification test completed!"
Write-Host ("=" * 50)
