# DriftMgr Single Executable Demo Script
# This script demonstrates the single driftmgr.exe with all enhanced features

Write-Host "🎯 DriftMgr Single Executable Demo" -ForegroundColor Cyan
Write-Host "===================================" -ForegroundColor Cyan
Write-Host ""

# Check if driftmgr.exe exists
if (Test-Path ".\driftmgr.exe") {
    Write-Host "[OK] Found driftmgr.exe executable" -ForegroundColor Green
} else {
    Write-Host "[ERROR] driftmgr.exe not found!" -ForegroundColor Red
    exit 1
}

Write-Host ""

# 1. Show the enhanced banner and help
Write-Host "1. Enhanced Banner and Help Display:" -ForegroundColor Yellow
.\driftmgr.exe
Write-Host ""

# 2. Test health check with ASCII characters
Write-Host "2. Health Check with ASCII Characters:" -ForegroundColor Yellow
.\driftmgr.exe health
Write-Host ""

# 3. Test state files listing
Write-Host "3. State Files Listing:" -ForegroundColor Yellow
.\driftmgr.exe statefiles
Write-Host ""

# 4. Test error handling with enhanced messages
Write-Host "4. Enhanced Error Messages:" -ForegroundColor Yellow
Write-Host "Testing missing arguments:" -ForegroundColor Gray
.\driftmgr.exe analyze
Write-Host ""

Write-Host "Testing invalid command:" -ForegroundColor Gray
.\driftmgr.exe invalid-command
Write-Host ""

Write-Host "Testing missing provider:" -ForegroundColor Gray
.\driftmgr.exe discover
Write-Host ""

# 5. Test discover command (if services are running)
Write-Host "5. Testing Discover Command:" -ForegroundColor Yellow
Write-Host "Note: This requires microservices to be running" -ForegroundColor Gray
.\driftmgr.exe discover aws us-east-1
Write-Host ""

# 6. Test analyze command (if services are running)
Write-Host "6. Testing Analyze Command:" -ForegroundColor Yellow
Write-Host "Note: This requires microservices to be running" -ForegroundColor Gray
.\driftmgr.exe analyze terraform
Write-Host ""

# 7. Test perspective command (if services are running)
Write-Host "7. Testing Perspective Command:" -ForegroundColor Yellow
Write-Host "Note: This requires microservices to be running" -ForegroundColor Gray
.\driftmgr.exe perspective terraform aws
Write-Host ""

# 8. Test visualization commands (if services are running)
Write-Host "8. Testing Visualization Commands:" -ForegroundColor Yellow
Write-Host "Note: This requires microservices to be running" -ForegroundColor Gray
.\driftmgr.exe visualize terraform
Write-Host ""

.\driftmgr.exe diagram terraform
Write-Host ""

.\driftmgr.exe export terraform html
Write-Host ""

# Summary
Write-Host "[OK] Demo completed! The single driftmgr.exe features:" -ForegroundColor Green
Write-Host "   • ASCII characters (^, +, !, *, >, |, -) for visual separation" -ForegroundColor White
Write-Host "   • Color-coded output for better visual separation" -ForegroundColor White
Write-Host "   • Enhanced error messages with usage hints" -ForegroundColor White
Write-Host "   • Professional banner and help display" -ForegroundColor White
Write-Host "   • All microservices functionality in one executable" -ForegroundColor White
Write-Host "   • Universal compatibility across all terminals" -ForegroundColor White
Write-Host ""
Write-Host "🚀 Ready to use: .\driftmgr.exe <command>" -ForegroundColor Cyan
