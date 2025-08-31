# Apply web GUI enhancements
Write-Host "Applying DriftMgr enhancements..." -ForegroundColor Green

# Backup original files
if (Test-Path "web/index.html") {
    Copy-Item "web/index.html" "web/index-original.html" -Force
    Write-Host "  Backed up original index.html" -ForegroundColor Yellow
}
if (Test-Path "web/js/app.js") {
    Copy-Item "web/js/app.js" "web/js/app-original.js" -Force
    Write-Host "  Backed up original app.js" -ForegroundColor Yellow
}

# Apply enhanced files
Copy-Item "web/index-enhanced.html" "web/index.html" -Force
Write-Host "  ✓ Applied enhanced HTML interface" -ForegroundColor Green

Copy-Item "web/js/app-enhanced.js" "web/js/app.js" -Force
Write-Host "  ✓ Applied enhanced JavaScript functionality" -ForegroundColor Green

Write-Host "`nEnhancements applied successfully!" -ForegroundColor Green
Write-Host "`nNew features added:" -ForegroundColor Cyan
Write-Host "  • Environment selector (Production/Staging/Development)"
Write-Host "  • Account switcher for multiple cloud accounts"
Write-Host "  • Resource deletion (single and bulk)"
Write-Host "  • Resource export/import functionality"
Write-Host "  • Audit logs with filtering and export"
Write-Host "  • Auto-remediation toggle in discovery"
Write-Host "  • Configured providers display"
Write-Host "  • WebSocket real-time updates"
Write-Host "  • Enhanced charts with provider status"

Write-Host "`nTo start the enhanced dashboard:" -ForegroundColor Yellow
Write-Host "  ./driftmgr.exe serve web" -ForegroundColor White
Write-Host "  or" -ForegroundColor Gray
Write-Host "  ./driftmgr.exe dashboard" -ForegroundColor White