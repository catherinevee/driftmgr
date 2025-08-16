# DriftMgr Installation Script for Windows
# This script installs driftmgr to make it available system-wide

param(
    [string]$InstallPath = "$env:USERPROFILE\driftmgr"
)

Write-Host "Installing DriftMgr..." -ForegroundColor Green

# Create installation directory
if (!(Test-Path $InstallPath)) {
    New-Item -ItemType Directory -Path $InstallPath -Force | Out-Null
    Write-Host "Created installation directory: $InstallPath" -ForegroundColor Yellow
}

# Copy the main executable
$SourceExe = ".\driftmgr.exe"
$DestExe = "$InstallPath\driftmgr.exe"

if (Test-Path $SourceExe) {
    Copy-Item $SourceExe $DestExe -Force
    Write-Host "Copied driftmgr.exe to: $DestExe" -ForegroundColor Yellow
} else {
    Write-Host "Error: driftmgr.exe not found in current directory" -ForegroundColor Red
    Write-Host "Please run 'make build' or 'go build -o driftmgr.exe main.go' first" -ForegroundColor Red
    exit 1
}

# Copy the bin directory
$SourceBin = ".\bin"
$DestBin = "$InstallPath\bin"

if (Test-Path $SourceBin) {
    if (Test-Path $DestBin) {
        Remove-Item $DestBin -Recurse -Force
    }
    Copy-Item $SourceBin $DestBin -Recurse -Force
    Write-Host "Copied bin directory to: $DestBin" -ForegroundColor Yellow
} else {
    Write-Host "Warning: bin directory not found" -ForegroundColor Yellow
}

# Add to PATH if not already there
$CurrentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($CurrentPath -notlike "*$InstallPath*") {
    [Environment]::SetEnvironmentVariable("PATH", "$CurrentPath;$InstallPath", "User")
    Write-Host "Added $InstallPath to PATH" -ForegroundColor Yellow
    Write-Host "Note: You may need to restart your terminal for PATH changes to take effect" -ForegroundColor Cyan
} else {
    Write-Host "Installation path already in PATH" -ForegroundColor Yellow
}

# Verify installation
Write-Host "`nVerifying installation..." -ForegroundColor Green
if (Test-Path $DestExe) {
    Write-Host "✓ Main executable installed successfully" -ForegroundColor Green
} else {
    Write-Host "✗ Main executable installation failed" -ForegroundColor Red
}

if (Test-Path $DestBin) {
    Write-Host "✓ Bin directory installed successfully" -ForegroundColor Green
} else {
    Write-Host "✗ Bin directory installation failed" -ForegroundColor Red
}

Write-Host "`nInstallation complete!" -ForegroundColor Green
Write-Host "You can now use 'driftmgr' from anywhere in your terminal" -ForegroundColor Green
Write-Host "`nUsage examples:" -ForegroundColor Cyan
Write-Host "  driftmgr                    # Start interactive shell" -ForegroundColor White
Write-Host "  driftmgr discover aws all   # Discover all AWS regions" -ForegroundColor White
Write-Host "  driftmgr discover aws us-east-1 us-west-2  # Discover specific regions" -ForegroundColor White
Write-Host "  driftmgr analyze terraform  # Analyze drift for state file" -ForegroundColor White
Write-Host "  driftmgr perspective terraform aws  # Compare state with live infrastructure" -ForegroundColor White
Write-Host "  driftmgr help              # Show help" -ForegroundColor White
Write-Host "`nNote: DriftMgr will automatically start the server when needed" -ForegroundColor Yellow
