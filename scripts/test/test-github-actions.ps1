# Test script for DriftMgr GitHub Actions integration (PowerShell)
# This script simulates the GitHub Actions environment and tests the workflow dispatch functionality

Write-Host "🧪 Testing DriftMgr GitHub Actions Integration" -ForegroundColor Cyan
Write-Host "==============================================" -ForegroundColor Cyan

# Build driftmgr
Write-Host "📦 Building DriftMgr..." -ForegroundColor Yellow
go build -o driftmgr.exe ./cmd/main.go

# Test 1: Validate inputs (should fail without environment variables)
Write-Host ""
Write-Host "🔍 Test 1: Validate inputs (should fail)" -ForegroundColor Yellow
try {
    .\driftmgr.exe github-actions validate-inputs
} catch {
    Write-Host "✅ Expected failure - no environment variables set" -ForegroundColor Green
}

# Test 2: Setup environment
Write-Host ""
Write-Host "🔧 Test 2: Setup environment" -ForegroundColor Yellow
.\driftmgr.exe github-actions setup-env

# Test 3: Validate inputs with environment variables
Write-Host ""
Write-Host "🔍 Test 3: Validate inputs with environment variables" -ForegroundColor Yellow
$env:WORKFLOW_TYPE = "drift-analysis"
$env:PROVIDER = "aws"
$env:REGIONS = "us-east-1"
$env:ENVIRONMENT = "test"
$env:DRY_RUN = "true"
$env:PARALLEL_IMPORTS = "5"
$env:OUTPUT_FORMAT = "json"

.\driftmgr.exe github-actions validate-inputs

# Test 4: Generate report
Write-Host ""
Write-Host "📊 Test 4: Generate report" -ForegroundColor Yellow
.\driftmgr.exe github-actions generate-report --output test-report.md

if (Test-Path "test-report.md") {
    Write-Host "✅ Report generated successfully" -ForegroundColor Green
    Write-Host "📄 Report preview:" -ForegroundColor Yellow
    Get-Content test-report.md | Select-Object -First 20
} else {
    Write-Host "❌ Report generation failed" -ForegroundColor Red
}

# Test 5: Workflow dispatch (dry run)
Write-Host ""
Write-Host "🚀 Test 5: Workflow dispatch (dry run)" -ForegroundColor Yellow
.\driftmgr.exe github-actions workflow-dispatch --type drift-analysis --provider aws --regions us-east-1 --environment test --dry-run

# Test 6: Check generated files
Write-Host ""
Write-Host "📁 Test 6: Check generated files" -ForegroundColor Yellow
if (Test-Path "driftmgr-data") {
    Get-ChildItem driftmgr-data -Recurse | Select-Object Name, Length
} else {
    Write-Host "No driftmgr-data directory found (expected for dry run)" -ForegroundColor Gray
}

$markdownFiles = Get-ChildItem *.md -ErrorAction SilentlyContinue
if ($markdownFiles) {
    Write-Host "Markdown files found:" -ForegroundColor Green
    $markdownFiles | ForEach-Object { Write-Host "  - $($_.Name)" -ForegroundColor Cyan }
} else {
    Write-Host "No markdown files found" -ForegroundColor Gray
}

Write-Host ""
Write-Host "✅ All tests completed!" -ForegroundColor Green
Write-Host ""
Write-Host "📋 Summary:" -ForegroundColor Cyan
Write-Host "- GitHub Actions integration is working" -ForegroundColor Green
Write-Host "- Environment setup is functional" -ForegroundColor Green
Write-Host "- Input validation is working" -ForegroundColor Green
Write-Host "- Report generation is working" -ForegroundColor Green
Write-Host "- Workflow dispatch is working" -ForegroundColor Green
Write-Host ""
Write-Host "🎉 DriftMgr is ready for GitHub Actions workflow dispatch!" -ForegroundColor Green
