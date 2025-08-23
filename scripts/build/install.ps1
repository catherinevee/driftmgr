# DriftMgr Windows Installation Script
param([switch]$Uninstall)

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$BinDir = Join-Path $ScriptDir "bin"
$DriftMgrExe = Join-Path $BinDir "driftmgr.exe"

function Test-Administrator {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Add-ToPath {
    param([string]$PathToAdd)
    try {
        $currentPath = [Environment]::GetEnvironmentVariable("PATH", "Machine")
        if ($currentPath -split ';' -contains $PathToAdd) {
            Write-Host "DriftMgr is already in the system PATH." -ForegroundColor Yellow
            return $true
        }
        $newPath = "$currentPath;$PathToAdd"
        [Environment]::SetEnvironmentVariable("PATH", $newPath, "Machine")
        Write-Host "Successfully added DriftMgr to system PATH." -ForegroundColor Green
        return $true
    }
    catch {
        Write-Error "Failed to add to PATH: $($_.Exception.Message)"
        return $false
    }
}

function Remove-FromPath {
    param([string]$PathToRemove)
    try {
        $currentPath = [Environment]::GetEnvironmentVariable("PATH", "Machine")
        $pathArray = $currentPath -split ';' | Where-Object { $_ -ne $PathToRemove }
        $newPath = $pathArray -join ';'
        [Environment]::SetEnvironmentVariable("PATH", $newPath, "Machine")
        Write-Host "Successfully removed DriftMgr from system PATH." -ForegroundColor Green
        return $true
    }
    catch {
        Write-Error "Failed to remove from PATH: $($_.Exception.Message)"
        return $false
    }
}

if ($Uninstall) {
    Write-Host "Uninstalling DriftMgr..." -ForegroundColor Yellow
    if (!(Test-Administrator)) {
        Write-Error "Administrator privileges required for uninstallation."
        exit 1
    }
    Remove-FromPath $BinDir
    Write-Host "DriftMgr has been uninstalled successfully." -ForegroundColor Green
    exit 0
}

Write-Host "Installing DriftMgr..." -ForegroundColor Green

if (!(Test-Path $DriftMgrExe)) {
    Write-Error "DriftMgr executable not found at: $DriftMgrExe"
    Write-Host "Please run 'make build' first to build the executables." -ForegroundColor Red
    exit 1
}

if (!(Test-Administrator)) {
    Write-Warning "Administrator privileges are required to add DriftMgr to the system PATH."
    Write-Host "Attempting to elevate privileges..." -ForegroundColor Yellow
    $arguments = "& '$($MyInvocation.MyCommand.Path)'"
    Start-Process powershell -Verb RunAs -ArgumentList "-Command $arguments"
    exit 0
}

Write-Host "Adding DriftMgr to system PATH..." -ForegroundColor Yellow
if (Add-ToPath $BinDir) {
    Write-Host "PATH updated successfully." -ForegroundColor Green
} else {
    Write-Error "Failed to update PATH. Installation incomplete."
    exit 1
}

Write-Host ""
Write-Host "==========================================" -ForegroundColor Green
Write-Host "DriftMgr Installation Complete!" -ForegroundColor Green
Write-Host "==========================================" -ForegroundColor Green
Write-Host ""
Write-Host "Installation Summary:" -ForegroundColor Cyan
Write-Host "  Added to system PATH: $BinDir" -ForegroundColor Green
Write-Host ""
Write-Host "Usage:" -ForegroundColor Cyan
Write-Host "  Open a new command prompt or PowerShell window" -ForegroundColor White
Write-Host "  Run 'driftmgr' to start the interactive shell" -ForegroundColor White
Write-Host "  Or run 'driftmgr discover aws all' for direct commands" -ForegroundColor White
Write-Host ""
Write-Host "Timeout Configuration:" -ForegroundColor Cyan
Write-Host "  For large infrastructure, configure timeouts:" -ForegroundColor White
Write-Host "  .\scripts\set-timeout.ps1 -Scenario large" -ForegroundColor White
Write-Host "  Or set environment variables:" -ForegroundColor White
Write-Host "  \$env:DRIFT_DISCOVERY_TIMEOUT = '10m'" -ForegroundColor White
Write-Host ""
Write-Host "Note: You may need to restart your terminal for PATH changes to take effect." -ForegroundColor Yellow
Write-Host ""
Write-Host "To uninstall, run: .\install.ps1 -Uninstall" -ForegroundColor Gray
