# Cleanup script to remove redundant directories after refactoring

Write-Host "Removing empty and redundant directories..." -ForegroundColor Green

# Remove empty directories
$emptyDirs = @(
    "internal/analytics/cost",
    "internal/analytics/metrics", 
    "internal/analytics/reports",
    "internal/api/handlers",
    "internal/api/middleware",
    "internal/core/state",
    "internal/utils/config",
    "internal/utils/errors",
    "internal/utils/logging",
    "internal/utils/metrics",
    "internal/utils/security",
    "internal/utils/utils"
)

foreach ($dir in $emptyDirs) {
    if (Test-Path $dir) {
        Remove-Item -Path $dir -Force -Recurse
        Write-Host "Removed empty directory: $dir" -ForegroundColor Yellow
    }
}

# Remove backup directory
if (Test-Path "internal/backup") {
    Remove-Item -Path "internal/backup" -Force -Recurse
    Write-Host "Removed backup directory: internal/backup" -ForegroundColor Yellow
}

Write-Host "`nDirectories cleaned up successfully!" -ForegroundColor Green
Write-Host "`nConsider reviewing these directories for removal:" -ForegroundColor Cyan
Write-Host "  - internal/discovery (consolidated into core/discovery)"
Write-Host "  - internal/drift (consolidated into core/drift)" 
Write-Host "  - internal/remediation (consolidated into core/remediation)"
Write-Host "  - internal/visualization (consolidated into core/visualization)"
Write-Host "  - internal/workspace"
Write-Host "  - internal/perspective"
Write-Host "  - internal/plugin"
Write-Host "  - internal/approval"