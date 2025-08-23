# DriftMgr Windows Installer
# PowerShell script for Windows installation

param(
    [string]$InstallPath = "",
    [switch]$Force,
    [switch]$SkipCredentialCheck,
    [switch]$UpdateCredentials
)

# Colors for output
$Red = "Red"
$Green = "Green"
$Yellow = "Yellow"
$Blue = "Blue"

# Function to print colored output
function Write-Status {
    param([string]$Message)
    Write-Host $Message -ForegroundColor $Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host " $Message" -ForegroundColor $Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host " $Message" -ForegroundColor $Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host " $Message" -ForegroundColor $Red
}

# Function to check prerequisites
function Test-Prerequisites {
    Write-Status "Checking prerequisites..."
    
    # Check if we are in the right directory
    if (-not (Test-Path "driftmgr.yaml")) {
        Write-Warning "driftmgr.yaml not found in current directory."
        Write-Warning "Make sure you are running this script from the DriftMgr project root."
    }
    
    # Check if binaries exist
    if (-not (Test-Path "bin")) {
        Write-Error "bin directory not found. Please build DriftMgr first."
        Write-Error "Run 'make build' or 'go build' to build the application."
        exit 1
    }
    
    # Check for required binaries
    if (-not (Test-Path "bin\driftmgr.exe")) {
        Write-Error "driftmgr.exe not found in bin directory."
        exit 1
    }
    
    if (-not (Test-Path "bin\driftmgr-server.exe")) {
        Write-Error "driftmgr-server.exe not found in bin directory."
        exit 1
    }
    
    Write-Success "Prerequisites check passed"
}

# Function to install DriftMgr
function Install-DriftMgr {
    Write-Status "Installing DriftMgr..."
    
    # Get current directory
    $ScriptDir = Get-Location
    $BinDir = "$ScriptDir\bin"
    
    # Default installation path
    if (-not $InstallPath) {
        $InstallPath = "$env:USERPROFILE\driftmgr"
    }
    
    # Create installation directory
    if (-not (Test-Path $InstallPath)) {
        New-Item -ItemType Directory -Path $InstallPath -Force | Out-Null
        Write-Success "Created installation directory: $InstallPath"
    }
    
    # Copy binaries
    Write-Status "Copying binaries..."
    Copy-Item "$BinDir\driftmgr.exe" "$InstallPath\" -Force
    Copy-Item "$BinDir\driftmgr-server.exe" "$InstallPath\" -Force
    Write-Success "Binaries copied successfully"
    
    # Copy configuration files
    if (Test-Path "driftmgr.yaml") {
        Copy-Item "driftmgr.yaml" "$InstallPath\" -Force
        Write-Success "Configuration copied successfully"
    }
    
    # Add to PATH if not already there
    $CurrentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($CurrentPath -notlike "*$InstallPath*") {
        [Environment]::SetEnvironmentVariable("PATH", "$CurrentPath;$InstallPath", "User")
        Write-Success "Added DriftMgr to PATH"
        Write-Warning "Please restart your terminal or run 'refreshenv' to update PATH"
    } else {
        Write-Success "DriftMgr is already in PATH"
    }
    
    Write-Success "Installation completed successfully!"
    Write-Host ""
    Write-Host "You can now run:"
    Write-Host "  driftmgr --help"
    Write-Host "  driftmgr-server --help"
    Write-Host ""
}

# Main installation script
function Main {
    Write-Host " DriftMgr Windows Installer"
    Write-Host "============================="
    Write-Host ""
    
    # Check prerequisites
    Test-Prerequisites
    
    # Install DriftMgr
    Install-DriftMgr
    
    Write-Success "Installation completed successfully!"
}

# Run main function
Main @args
