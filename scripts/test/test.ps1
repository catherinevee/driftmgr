# PowerShell test runner script for comprehensive test suite

Write-Host "🧪 Running Terraform Import Helper Test Suite" -ForegroundColor Cyan
Write-Host "===============================================" -ForegroundColor Cyan

# Change to project directory
Set-Location $PSScriptRoot

Write-Host "📍 Current directory: $(Get-Location)" -ForegroundColor Yellow
Write-Host ""

# Run go mod tidy first
Write-Host "🔧 Running go mod tidy..." -ForegroundColor Green
go mod tidy
Write-Host ""

# Test individual packages
Write-Host "🧮 Testing Models package..." -ForegroundColor Green
go test -v ./internal/models
Write-Host ""

Write-Host "🔍 Testing Discovery package..." -ForegroundColor Green
go test -v ./internal/discovery
Write-Host ""

Write-Host "🎨 Testing TUI package..." -ForegroundColor Green
go test -v ./internal/tui
Write-Host ""

Write-Host "📥 Testing Importer package..." -ForegroundColor Green
go test -v ./internal/importer
Write-Host ""

# Run all tests
Write-Host "🚀 Running all tests..." -ForegroundColor Green
go test -v ./...
Write-Host ""

Write-Host "[OK] Test suite completed!" -ForegroundColor Cyan
