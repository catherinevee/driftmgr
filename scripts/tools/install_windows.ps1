# DriftMgr Windows Installation Script
# This script installs driftmgr.exe to make it accessible from anywhere

Write-Host "DriftMgr Installation Script for Windows" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check if running as administrator
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")

# Define installation directory
$installDir = "$env:LOCALAPPDATA\DriftMgr"
$exePath = Join-Path $installDir "driftmgr.exe"
$sourcePath = Join-Path $PSScriptRoot "driftmgr.exe"

# Check if source file exists
if (-not (Test-Path $sourcePath)) {
    Write-Host "Error: driftmgr.exe not found in current directory!" -ForegroundColor Red
    Write-Host "Please build the application first with: go build -o driftmgr.exe ./cmd/driftmgr" -ForegroundColor Yellow
    exit 1
}

# Create installation directory if it doesn't exist
Write-Host "Creating installation directory: $installDir" -ForegroundColor Green
if (-not (Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir -Force | Out-Null
}

# Copy driftmgr.exe to installation directory
Write-Host "Installing driftmgr.exe to: $exePath" -ForegroundColor Green
Copy-Item -Path $sourcePath -Destination $exePath -Force

# Check if installation directory is in PATH
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
$paths = $userPath -split ';'

if ($paths -notcontains $installDir) {
    Write-Host "Adding $installDir to user PATH..." -ForegroundColor Green
    
    # Add to user PATH
    $newPath = $userPath
    if ($newPath -and !$newPath.EndsWith(';')) {
        $newPath += ';'
    }
    $newPath += $installDir
    
    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    
    Write-Host "✓ Added to PATH successfully!" -ForegroundColor Green
    Write-Host ""
    Write-Host "IMPORTANT: You need to restart your terminal for PATH changes to take effect!" -ForegroundColor Yellow
} else {
    Write-Host "✓ Installation directory already in PATH" -ForegroundColor Green
}

# Verify installation
if (Test-Path $exePath) {
    $version = & $exePath --version 2>&1
    Write-Host ""
    Write-Host "✓ DriftMgr installed successfully!" -ForegroundColor Green
    Write-Host "  Location: $exePath" -ForegroundColor Gray
    Write-Host ""
    Write-Host "To use DriftMgr:" -ForegroundColor Cyan
    Write-Host "  1. Close and reopen your terminal (PowerShell/CMD)" -ForegroundColor White
    Write-Host "  2. Run: driftmgr" -ForegroundColor White
    Write-Host ""
    Write-Host "Available commands:" -ForegroundColor Cyan
    Write-Host "  driftmgr              - Launch interactive TUI" -ForegroundColor White
    Write-Host "  driftmgr --enhanced   - Launch enhanced TUI" -ForegroundColor White
    Write-Host "  driftmgr --help       - Show help" -ForegroundColor White
} else {
    Write-Host "Error: Installation failed!" -ForegroundColor Red
    exit 1
}

# Create uninstall script
$uninstallScript = @"
# DriftMgr Uninstall Script
Write-Host 'Uninstalling DriftMgr...' -ForegroundColor Yellow

`$installDir = "`$env:LOCALAPPDATA\DriftMgr"

# Remove from PATH
`$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
`$paths = `$userPath -split ';' | Where-Object { `$_ -ne `$installDir }
`$newPath = `$paths -join ';'
[Environment]::SetEnvironmentVariable('Path', `$newPath, 'User')

# Remove installation directory
if (Test-Path `$installDir) {
    Remove-Item -Path `$installDir -Recurse -Force
    Write-Host '✓ DriftMgr uninstalled successfully!' -ForegroundColor Green
} else {
    Write-Host 'DriftMgr is not installed.' -ForegroundColor Yellow
}
"@

$uninstallPath = Join-Path $installDir "uninstall.ps1"
$uninstallScript | Out-File -FilePath $uninstallPath -Encoding UTF8

Write-Host "Uninstaller created at: $uninstallPath" -ForegroundColor Gray