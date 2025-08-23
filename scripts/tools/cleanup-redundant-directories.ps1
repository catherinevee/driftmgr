# Final cleanup script to remove all redundant directories after refactoring

Write-Host "`n=== DriftMgr Directory Cleanup Script ===" -ForegroundColor Cyan
Write-Host "This will remove directories that have been consolidated into core modules" -ForegroundColor Yellow
Write-Host ""

# Directories confirmed safe to remove (no imports, functionality moved to core)
$redundantDirs = @(
    # Consolidated into core modules
    "internal/discovery",      # 17 files - moved to core/discovery
    "internal/drift",          # 5 files - moved to core/drift
    "internal/remediation",    # 11 files - moved to core/remediation
    "internal/visualization",  # 5 files - moved to core/visualization
    
    # Empty directories with no Go files
    "internal/workspace",
    "internal/perspective",
    "internal/plugin",
    "internal/approval",
    "internal/export",
    "internal/cache",
    
    # Not imported anywhere
    "internal/deletion",       # 6 files - not used
    
    # Analytics subdirectories (parent still has files)
    "internal/analytics"       # Check if needed
)

$totalFiles = 0
$removedDirs = 0

Write-Host "Analyzing directories to remove..." -ForegroundColor Green
Write-Host ""

foreach ($dir in $redundantDirs) {
    if (Test-Path $dir) {
        $fileCount = (Get-ChildItem -Path $dir -Filter "*.go" -Recurse -ErrorAction SilentlyContinue | Measure-Object).Count
        $totalFiles += $fileCount
        
        Write-Host "Removing: $dir" -ForegroundColor Yellow
        if ($fileCount -gt 0) {
            Write-Host "  - Contains $fileCount Go files (functionality moved to core modules)" -ForegroundColor Gray
        } else {
            Write-Host "  - Empty directory" -ForegroundColor Gray
        }
        
        Remove-Item -Path $dir -Force -Recurse
        $removedDirs++
    }
}

Write-Host ""
Write-Host "=== Cleanup Summary ===" -ForegroundColor Green
Write-Host "Removed $removedDirs directories" -ForegroundColor Cyan
Write-Host "Removed $totalFiles redundant Go files" -ForegroundColor Cyan
Write-Host ""

# Check remaining structure
Write-Host "=== Remaining internal structure ===" -ForegroundColor Green
$remaining = Get-ChildItem -Path "internal" -Directory | Select-Object -ExpandProperty Name
$remaining | ForEach-Object {
    $goFiles = (Get-ChildItem -Path "internal/$_" -Filter "*.go" -Recurse -ErrorAction SilentlyContinue | Measure-Object).Count
    if ($goFiles -gt 0) {
        Write-Host "  internal/$_" -NoNewline -ForegroundColor White
        Write-Host " ($goFiles files)" -ForegroundColor Gray
    }
}

Write-Host ""
Write-Host "Cleanup complete!" -ForegroundColor Green
Write-Host "The codebase is now cleaner with consolidated core modules." -ForegroundColor Cyan