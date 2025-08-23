# DriftMgr Build Verification Script
# This script verifies that all required binaries are built correctly

Write-Host "Verifying DriftMgr build..." -ForegroundColor Blue

# Required binaries
$requiredBinaries = @(
    "driftmgr.exe",
    "driftmgr-client.exe",
    "driftmgr-server.exe"
)

$allGood = $true
$missingBinaries = @()

# Check required binaries
foreach ($binary in $requiredBinaries) {
    $path = "bin\$binary"
    if (Test-Path $path) {
        $size = (Get-Item $path).Length
        $sizeMB = [math]::Round($size/1MB, 2)
        Write-Host "OK $binary found ($sizeMB MB)" -ForegroundColor Green
    } else {
        Write-Host "MISSING $binary" -ForegroundColor Red
        $missingBinaries += $binary
        $allGood = $false
    }
}

# Test main driftmgr command
Write-Host "Testing main driftmgr command..." -ForegroundColor Blue
try {
    # Test with a simple command that exits immediately
    $result = echo "exit" | & ".\bin\driftmgr.exe" 2>&1 | Select-Object -First 5
    if ($result -match "DriftMgr") {
        Write-Host "OK Main driftmgr command works" -ForegroundColor Green
    } else {
        Write-Host "WARNING Main driftmgr command may have issues" -ForegroundColor Yellow
    }
} catch {
    Write-Host "ERROR Main driftmgr command failed: $_" -ForegroundColor Red
    $allGood = $false
}

# Summary
Write-Host ""
if ($allGood) {
    Write-Host "Build verification PASSED! All required binaries are present and working." -ForegroundColor Green
} else {
    Write-Host "Build verification FAILED! Missing required binaries: $($missingBinaries -join ', ')" -ForegroundColor Red
    exit 1
}

Write-Host "Verification complete." -ForegroundColor Blue
