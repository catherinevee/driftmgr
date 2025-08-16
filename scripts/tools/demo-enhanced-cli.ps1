# DriftMgr Enhanced CLI Demo Script (PowerShell)
# This script demonstrates the enhanced CLI with special characters and colors

Write-Host "🎯 DriftMgr Enhanced CLI Demo" -ForegroundColor Cyan
Write-Host "==============================" -ForegroundColor Cyan
Write-Host ""

# Check if services are running
Write-Host "1. Checking service health..." -ForegroundColor Yellow
& ".\bin\driftmgr-client.exe" health
Write-Host ""

# List available state files
Write-Host "2. Listing available state files..." -ForegroundColor Yellow
& ".\bin\driftmgr-client.exe" statefiles
Write-Host ""

# Show help (enhanced formatting)
Write-Host "3. Enhanced help display..." -ForegroundColor Yellow
& ".\bin\driftmgr-client.exe"
Write-Host ""

# Demonstrate error handling
Write-Host "4. Enhanced error messages..." -ForegroundColor Yellow
Write-Host "Testing invalid command:" -ForegroundColor Gray
& ".\bin\driftmgr-client.exe" invalid-command
Write-Host ""

Write-Host "Testing missing arguments:" -ForegroundColor Gray
& ".\bin\driftmgr-client.exe" analyze
Write-Host ""

Write-Host "Testing missing arguments for discover:" -ForegroundColor Gray
& ".\bin\driftmgr-client.exe" discover
Write-Host ""

Write-Host "✅ Demo completed! The enhanced CLI now features:" -ForegroundColor Green
Write-Host "   • ASCII characters (^, +, !, *, >, |, -) for visual separation" -ForegroundColor White
Write-Host "   • Color-coded output for better visual separation" -ForegroundColor White
Write-Host "   • Enhanced error messages with usage hints" -ForegroundColor White
Write-Host "   • Improved command structure and readability" -ForegroundColor White
Write-Host "   • Better visual hierarchy with standard characters" -ForegroundColor White
