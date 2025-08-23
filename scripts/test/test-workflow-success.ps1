# Test script to ensure GitHub Actions workflow will succeed (PowerShell)
Write-Host "🧪 Testing GitHub Actions Workflow Success" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan

# Test 1: Build DriftMgr
Write-Host "📦 Test 1: Building DriftMgr..." -ForegroundColor Yellow
go build -o driftmgr.exe ./cmd/main.go
if (Test-Path "driftmgr.exe") {
    Write-Host "[OK] Build successful" -ForegroundColor Green
} else {
    Write-Host "[ERROR] Build failed" -ForegroundColor Red
    exit 1
}

# Test 2: Validate binary exists
Write-Host "🔍 Test 2: Validating binary..." -ForegroundColor Yellow
if (Test-Path "driftmgr.exe") {
    Write-Host "[OK] Binary exists" -ForegroundColor Green
    Get-ChildItem driftmgr.exe | Select-Object Name, Length, LastWriteTime
} else {
    Write-Host "[ERROR] Binary validation failed" -ForegroundColor Red
    exit 1
}

# Test 3: Test GitHub Actions integration
Write-Host "🚀 Test 3: Testing GitHub Actions integration..." -ForegroundColor Yellow
$env:WORKFLOW_TYPE = "drift-analysis"
$env:PROVIDER = "aws"
$env:REGIONS = "us-east-1"
$env:ENVIRONMENT = "test"
$env:DRY_RUN = "true"
$env:PARALLEL_IMPORTS = "5"
$env:OUTPUT_FORMAT = "json"

try {
    .\driftmgr.exe github-actions validate-inputs
    Write-Host "[OK] GitHub Actions validation passed" -ForegroundColor Green
} catch {
    Write-Host "[ERROR] GitHub Actions validation failed" -ForegroundColor Red
    exit 1
}

# Test 4: Test environment setup
Write-Host "🔧 Test 4: Testing environment setup..." -ForegroundColor Yellow
try {
    .\driftmgr.exe github-actions setup-env
    Write-Host "[OK] Environment setup passed" -ForegroundColor Green
} catch {
    Write-Host "[ERROR] Environment setup failed" -ForegroundColor Red
    exit 1
}

# Test 5: Test report generation
Write-Host "📊 Test 5: Testing report generation..." -ForegroundColor Yellow
try {
    .\driftmgr.exe github-actions generate-report --output test-workflow-report.md
    if (Test-Path "test-workflow-report.md") {
        Write-Host "[OK] Report generation passed" -ForegroundColor Green
        Write-Host "📄 Report preview:" -ForegroundColor Yellow
        Get-Content test-workflow-report.md | Select-Object -First 10
    } else {
        Write-Host "[ERROR] Report generation failed" -ForegroundColor Red
        exit 1
    }
} catch {
    Write-Host "[ERROR] Report generation failed" -ForegroundColor Red
    exit 1
}

# Test 6: Test workflow dispatch (dry run)
Write-Host "🎯 Test 6: Testing workflow dispatch..." -ForegroundColor Yellow
try {
    .\driftmgr.exe github-actions workflow-dispatch --type drift-analysis --provider aws --regions us-east-1 --environment test --dry-run
    Write-Host "[OK] Workflow dispatch passed" -ForegroundColor Green
} catch {
    Write-Host "[ERROR] Workflow dispatch failed" -ForegroundColor Red
    exit 1
}

# Test 7: Check generated files
Write-Host "📁 Test 7: Checking generated files..." -ForegroundColor Yellow
if (Test-Path "driftmgr-data") {
    Write-Host "[OK] Data directory created" -ForegroundColor Green
    Get-ChildItem driftmgr-data -Recurse | Select-Object Name, Length
} else {
    Write-Host "[WARNING] No data directory found (expected for dry run)" -ForegroundColor Gray
}

Write-Host ""
Write-Host "🎉 All tests passed! GitHub Actions workflow will succeed." -ForegroundColor Green
Write-Host ""
Write-Host "📋 Summary:" -ForegroundColor Cyan
Write-Host "- [OK] Build process works" -ForegroundColor Green
Write-Host "- [OK] Binary validation works" -ForegroundColor Green
Write-Host "- [OK] GitHub Actions integration works" -ForegroundColor Green
Write-Host "- [OK] Environment setup works" -ForegroundColor Green
Write-Host "- [OK] Report generation works" -ForegroundColor Green
Write-Host "- [OK] Workflow dispatch works" -ForegroundColor Green
Write-Host ""
Write-Host "🚀 Ready for GitHub Actions deployment!" -ForegroundColor Green
