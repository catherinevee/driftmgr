# DriftMgr Windows Installer
# This script installs DriftMgr and configures the environment

param(
    [string]$InstallPath = "$env:USERPROFILE\driftmgr",
    [switch]$Force,
    [switch]$SkipCredentialCheck,
    [switch]$UpdateCredentials
)

# Set error action preference
$ErrorActionPreference = "Stop"

# Colors for output
$Red = "Red"
$Green = "Green"
$Yellow = "Yellow"
$Blue = "Blue"

function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Color = "White"
    )
    Write-Host $Message -ForegroundColor $Color
}

function Update-DriftMgrCredentials {
    Write-ColorOutput "Auto-detecting DriftMgr credentials..." $Blue
    
    # Get the directory where this script is located
    $ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
    $ProjectRoot = Split-Path -Parent (Split-Path -Parent $ScriptDir)
    
    # Check if driftmgr binary exists
    $DriftMgrExe = Join-Path $ProjectRoot "bin\driftmgr.exe"
    if (Test-Path $DriftMgrExe) {
        Write-ColorOutput "Running credential auto-detection..." $Blue
        try {
            & $DriftMgrExe credentials auto-detect
            Write-ColorOutput "Credentials auto-detection completed!" $Green
        }
        catch {
            Write-ColorOutput "Failed to auto-detect credentials: $($_.Exception.Message)" $Red
            throw
        }
    }
    else {
        Write-ColorOutput "DriftMgr binary not found at $DriftMgrExe" $Red
        Write-ColorOutput "Please install DriftMgr first." $Yellow
        exit 1
    }
}

function Test-Administrator {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Test-GoInstalled {
    try {
        $goVersion = go version 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-ColorOutput "Go is installed: $goVersion" $Green
            return $true
        }
    }
    catch {
        Write-ColorOutput "Go is not installed or not in PATH" $Red
        return $false
    }
    return $false
}

function Install-Go {
    Write-ColorOutput "Installing Go..." $Blue
    
    # Download Go installer
    $goVersion = "1.21.0"
    $goInstaller = "go$goVersion.windows-amd64.msi"
    $goUrl = "https://golang.org/dl/$goInstaller"
    $tempPath = "$env:TEMP\$goInstaller"
    
    try {
        Write-ColorOutput "Downloading Go installer..." $Blue
        Invoke-WebRequest -Uri $goUrl -OutFile $tempPath
        
        Write-ColorOutput "Installing Go..." $Blue
        Start-Process msiexec.exe -Wait -ArgumentList "/i $tempPath /quiet"
        
        # Refresh environment variables
        $env:PATH = [System.Environment]::GetEnvironmentVariable("PATH","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("PATH","User")
        
        Write-ColorOutput "Go installed successfully" $Green
    }
    catch {
        Write-ColorOutput "Failed to install Go: $($_.Exception.Message)" $Red
        throw
    }
    finally {
        if (Test-Path $tempPath) {
            Remove-Item $tempPath -Force
        }
    }
}

function Install-DriftMgr {
    Write-ColorOutput "Installing DriftMgr..." $Blue
    
    # Create installation directory
    if (!(Test-Path $InstallPath)) {
        New-Item -ItemType Directory -Path $InstallPath -Force | Out-Null
    }
    
    # Copy executable to installation directory
    $sourceExe = Join-Path $PSScriptRoot "..\..\bin\driftmgr.exe"
    $sourceServerExe = Join-Path $PSScriptRoot "..\..\bin\driftmgr-server.exe"
    $targetExe = Join-Path $InstallPath "driftmgr.exe"
    $targetServerExe = Join-Path $InstallPath "driftmgr-server.exe"
    
    if (Test-Path $sourceExe) {
        Copy-Item $sourceExe $targetExe -Force
        Write-ColorOutput "DriftMgr CLI installed" $Green
    } else {
        Write-ColorOutput "DriftMgr executable not found at $sourceExe" $Red
        throw "DriftMgr executable not found"
    }
    
    if (Test-Path $sourceServerExe) {
        Copy-Item $sourceServerExe $targetServerExe -Force
        Write-ColorOutput "DriftMgr Server installed" $Green
    } else {
        Write-ColorOutput "DriftMgr Server executable not found at $sourceServerExe" $Yellow
    }
    
    # Create bin directory and copy executables there for global access
    $binPath = Join-Path $InstallPath "bin"
    if (!(Test-Path $binPath)) {
        New-Item -ItemType Directory -Path $binPath -Force | Out-Null
    }
    
    Copy-Item $sourceExe (Join-Path $binPath "driftmgr.exe") -Force
    if (Test-Path $sourceServerExe) {
        Copy-Item $sourceServerExe (Join-Path $binPath "driftmgr-server.exe") -Force
    }
}

function Add-ToPath {
    Write-ColorOutput "Adding DriftMgr to PATH..." $Blue
    
    $currentPath = [System.Environment]::GetEnvironmentVariable("PATH", "User")
    
    if ($currentPath -notlike "*$InstallPath*") {
        $newPath = "$currentPath;$InstallPath"
        [System.Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
        Write-ColorOutput "DriftMgr added to PATH" $Green
    } else {
        Write-ColorOutput "DriftMgr already in PATH" $Yellow
    }
    
    # Also add bin directory to PATH
    $binPath = Join-Path $InstallPath "bin"
    if ($currentPath -notlike "*$binPath*") {
        $newPath = "$currentPath;$binPath"
        [System.Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
        Write-ColorOutput "DriftMgr bin directory added to PATH" $Green
    }
}

function Test-CloudCredentials {
    Write-ColorOutput "Checking cloud credentials..." $Blue
    
    $hasCredentials = $false
    
    # Check AWS credentials
    $awsCredentials = Join-Path $env:USERPROFILE ".aws\credentials"
    if (Test-Path $awsCredentials) {
        Write-ColorOutput "AWS credentials found" $Green
        $hasCredentials = $true
    } else {
        Write-ColorOutput "AWS credentials not found" $Yellow
    }
    
    # Check Azure credentials
    $azureCredentials = Join-Path $env:USERPROFILE ".azure\credentials.json"
    if (Test-Path $azureCredentials) {
        Write-ColorOutput "Azure credentials found" $Green
        $hasCredentials = $true
    } else {
        Write-ColorOutput "Azure credentials not found" $Yellow
    }
    
    # Check GCP credentials
    $gcpCredentials = $env:GOOGLE_APPLICATION_CREDENTIALS
    if ($gcpCredentials -and (Test-Path $gcpCredentials)) {
        Write-ColorOutput "GCP credentials found" $Green
        $hasCredentials = $true
    } else {
        Write-ColorOutput "GCP credentials not found" $Yellow
    }
    
    # Check DigitalOcean credentials
    $doCredentials = Join-Path $env:USERPROFILE ".digitalocean\credentials"
    if (Test-Path $doCredentials) {
        Write-ColorOutput "DigitalOcean credentials found" $Green
        $hasCredentials = $true
    } else {
        Write-ColorOutput "DigitalOcean credentials not found" $Yellow
    }
    
    if (-not $hasCredentials) {
        Write-ColorOutput "No cloud credentials found. You may need to configure them later." $Yellow
        Write-ColorOutput "Run 'driftmgr credentials setup' to configure cloud credentials." $Blue
    }
    
    return $hasCredentials
}

function Setup-CloudCredentials {
    Write-ColorOutput "Setting up cloud credentials..." $Blue
    
    $driftmgrExe = Join-Path $InstallPath "driftmgr.exe"
    if (Test-Path $driftmgrExe) {
        try {
            & $driftmgrExe credentials setup
            Write-ColorOutput "Cloud credentials setup completed" $Green
        }
        catch {
            Write-ColorOutput "Failed to setup cloud credentials: $($_.Exception.Message)" $Red
            Write-ColorOutput "You can run 'driftmgr credentials setup' manually later." $Yellow
        }
    } else {
        Write-ColorOutput "DriftMgr executable not found. Cannot setup credentials automatically." $Red
    }
}

function Test-Installation {
    Write-ColorOutput "Testing installation..." $Blue
    
    $driftmgrExe = Join-Path $InstallPath "driftmgr.exe"
    if (Test-Path $driftmgrExe) {
        try {
            $version = & $driftmgrExe version
            Write-ColorOutput "DriftMgr installation verified: $version" $Green
            return $true
        }
        catch {
            Write-ColorOutput "Failed to verify DriftMgr installation: $($_.Exception.Message)" $Red
            return $false
        }
    } else {
        Write-ColorOutput "DriftMgr executable not found" $Red
        return $false
    }
}

function Show-InstallationSummary {
    Write-ColorOutput "Installation Summary" $Blue
    Write-ColorOutput "===================" $Blue
    Write-ColorOutput "Installation Path: $InstallPath" $White
    Write-ColorOutput "Executable: $InstallPath\driftmgr.exe" $White
    
    if (Test-Path (Join-Path $InstallPath "driftmgr-server.exe")) {
        Write-ColorOutput "Server: $InstallPath\driftmgr-server.exe" $White
    }
    
    Write-ColorOutput "" $White
    Write-ColorOutput "Next Steps:" $Blue
    Write-ColorOutput "1. Open a new PowerShell window to refresh PATH" $White
    Write-ColorOutput "2. Run 'driftmgr --help' to see available commands" $White
    Write-ColorOutput "3. Run 'driftmgr credentials setup' to configure cloud credentials" $White
    Write-ColorOutput "4. Run 'driftmgr discover' to start discovering resources" $White
    Write-ColorOutput "" $White
    Write-ColorOutput "Documentation: https://github.com/catherinevee/driftmgr" $White
}

# Main installation function
function Install-DriftMgrComplete {
    Write-ColorOutput "🚀 DriftMgr Windows Installer" $Blue
    Write-ColorOutput "=============================" $Blue
    Write-ColorOutput "" $White
    
    # Check if Go is installed
    if (-not (Test-GoInstalled)) {
        Write-ColorOutput "Go is required but not installed. Installing Go..." $Yellow
        Install-Go
    }
    
    # Install DriftMgr
    Install-DriftMgr
    
    # Add to PATH
    Add-ToPath
    
    # Check cloud credentials
    if (-not $SkipCredentialCheck) {
        $hasCredentials = Test-CloudCredentials
        
        if (-not $hasCredentials) {
            $setupCredentials = Read-Host "No cloud credentials found. Would you like to setup credentials now? (y/n)"
            if ($setupCredentials -eq "y" -or $setupCredentials -eq "Y") {
                Setup-CloudCredentials
            }
        }
    }
    
    # Test installation
    if (Test-Installation) {
        Write-ColorOutput "" $White
        Show-InstallationSummary
        Write-ColorOutput "" $White
        Write-ColorOutput "Installation completed successfully! 🎉" $Green
    } else {
        Write-ColorOutput "Installation completed but verification failed." $Yellow
        Write-ColorOutput "Please check the installation manually." $Yellow
    }
}

# Main execution
try {
    # Handle credential update only
    if ($UpdateCredentials) {
        Update-DriftMgrCredentials
        exit 0
    }
    
    # Check if installation already exists
    if ((Test-Path (Join-Path $InstallPath "driftmgr.exe")) -and -not $Force) {
        Write-ColorOutput "DriftMgr is already installed at $InstallPath" $Yellow
        $overwrite = Read-Host "Would you like to overwrite the existing installation? (y/n)"
        if ($overwrite -ne "y" -and $overwrite -ne "Y") {
            Write-ColorOutput "Installation cancelled." $Yellow
            exit 0
        }
    }
    
    Install-DriftMgrComplete
}
catch {
    Write-ColorOutput "Installation failed: $($_.Exception.Message)" $Red
    exit 1
}

