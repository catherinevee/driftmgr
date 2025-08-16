# DriftMgr Windows Installer
# This script installs DriftMgr and configures the environment

param(
    [string]$InstallPath = "$env:USERPROFILE\driftmgr",
    [switch]$Force,
    [switch]$SkipCredentialCheck
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
    }
    
    # Copy configuration files
    $configFiles = @("driftmgr.yaml", "go.mod", "go.sum")
    foreach ($file in $configFiles) {
        $sourceFile = Join-Path $PSScriptRoot "..\..\$file"
        $targetFile = Join-Path $InstallPath $file
        if (Test-Path $sourceFile) {
            Copy-Item $sourceFile $targetFile -Force
        }
    }
    
    # Copy assets directory
    $sourceAssets = Join-Path $PSScriptRoot "..\..\assets"
    $targetAssets = Join-Path $InstallPath "assets"
    if (Test-Path $sourceAssets) {
        Copy-Item $sourceAssets $targetAssets -Recurse -Force
    }
}

function Add-ToPath {
    Write-ColorOutput "Adding DriftMgr to PATH..." $Blue
    
    $currentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    
    if ($currentPath -notlike "*$InstallPath*") {
        $newPath = "$currentPath;$InstallPath"
        [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
        Write-ColorOutput "DriftMgr added to PATH" $Green
        
        # Update current session PATH
        $env:PATH = "$env:PATH;$InstallPath"
    } else {
        Write-ColorOutput "DriftMgr already in PATH" $Green
    }
}

function Test-AWSCredentials {
    Write-ColorOutput "Checking AWS credentials..." $Blue
    
    $awsProfiles = @()
    
    # Check AWS CLI configuration
    $awsConfigPath = "$env:USERPROFILE\.aws\config"
    if (Test-Path $awsConfigPath) {
        $awsProfiles += "AWS CLI config found"
    }
    
    # Check environment variables
    if ($env:AWS_ACCESS_KEY_ID -and $env:AWS_SECRET_ACCESS_KEY) {
        $awsProfiles += "AWS environment variables found"
    }
    
    # Check AWS credentials file
    $awsCredsPath = "$env:USERPROFILE\.aws\credentials"
    if (Test-Path $awsCredsPath) {
        $awsProfiles += "AWS credentials file found"
    }
    
    if ($awsProfiles.Count -gt 0) {
        Write-ColorOutput "AWS credentials detected:" $Green
        foreach ($profile in $awsProfiles) {
            Write-ColorOutput "  - $profile" $Green
        }
        return $true
    } else {
        Write-ColorOutput "No AWS credentials found" $Yellow
        return $false
    }
}

function Test-AzureCredentials {
    Write-ColorOutput "Checking Azure credentials..." $Blue
    
    $azureProfiles = @()
    
    # Check Azure CLI
    try {
        $azAccount = az account show 2>$null
        if ($LASTEXITCODE -eq 0) {
            $azureProfiles += "Azure CLI authenticated"
        }
    }
    catch {
        # Azure CLI not installed or not authenticated
    }
    
    # Check environment variables
    if ($env:AZURE_CLIENT_ID -and $env:AZURE_CLIENT_SECRET -and $env:AZURE_TENANT_ID) {
        $azureProfiles += "Azure environment variables found"
    }
    
    if ($azureProfiles.Count -gt 0) {
        Write-ColorOutput "Azure credentials detected:" $Green
        foreach ($profile in $azureProfiles) {
            Write-ColorOutput "  - $profile" $Green
        }
        return $true
    } else {
        Write-ColorOutput "No Azure credentials found" $Yellow
        return $false
    }
}

function Test-GCPCredentials {
    Write-ColorOutput "Checking GCP credentials..." $Blue
    
    $gcpProfiles = @()
    
    # Check gcloud CLI
    try {
        $gcloudAuth = gcloud auth list --filter=status:ACTIVE --format="value(account)" 2>$null
        if ($LASTEXITCODE -eq 0 -and $gcloudAuth) {
            $gcpProfiles += "GCP CLI authenticated: $gcloudAuth"
        }
    }
    catch {
        # gcloud CLI not installed or not authenticated
    }
    
    # Check service account key file
    if ($env:GOOGLE_APPLICATION_CREDENTIALS -and (Test-Path $env:GOOGLE_APPLICATION_CREDENTIALS)) {
        $gcpProfiles += "GCP service account key found"
    }
    
    if ($gcpProfiles.Count -gt 0) {
        Write-ColorOutput "GCP credentials detected:" $Green
        foreach ($profile in $gcpProfiles) {
            Write-ColorOutput "  - $profile" $Green
        }
        return $true
    } else {
        Write-ColorOutput "No GCP credentials found" $Yellow
        return $false
    }
}

function Install-AWS-CLI {
    Write-ColorOutput "Installing AWS CLI..." $Blue
    
    try {
        # Download AWS CLI MSI installer
        $awsCliUrl = "https://awscli.amazonaws.com/AWSCLIV2.msi"
        $tempPath = "$env:TEMP\AWSCLIV2.msi"
        
        Write-ColorOutput "Downloading AWS CLI..." $Blue
        Invoke-WebRequest -Uri $awsCliUrl -OutFile $tempPath
        
        Write-ColorOutput "Installing AWS CLI..." $Blue
        Start-Process msiexec.exe -Wait -ArgumentList "/i $tempPath /quiet"
        
        Write-ColorOutput "AWS CLI installed successfully" $Green
    }
    catch {
        Write-ColorOutput "Failed to install AWS CLI: $($_.Exception.Message)" $Red
    }
    finally {
        if (Test-Path $tempPath) {
            Remove-Item $tempPath -Force
        }
    }
}

function Install-Azure-CLI {
    Write-ColorOutput "Installing Azure CLI..." $Blue
    
    try {
        # Download Azure CLI installer
        $azureCliUrl = "https://aka.ms/installazurecliwindows"
        $tempPath = "$env:TEMP\azure-cli-installer.msi"
        
        Write-ColorOutput "Downloading Azure CLI..." $Blue
        Invoke-WebRequest -Uri $azureCliUrl -OutFile $tempPath
        
        Write-ColorOutput "Installing Azure CLI..." $Blue
        Start-Process msiexec.exe -Wait -ArgumentList "/i $tempPath /quiet"
        
        Write-ColorOutput "Azure CLI installed successfully" $Green
    }
    catch {
        Write-ColorOutput "Failed to install Azure CLI: $($_.Exception.Message)" $Red
    }
    finally {
        if (Test-Path $tempPath) {
            Remove-Item $tempPath -Force
        }
    }
}

function Install-GCP-CLI {
    Write-ColorOutput "Installing Google Cloud CLI..." $Blue
    
    try {
        # Download Google Cloud SDK installer
        $gcpCliUrl = "https://dl.google.com/dl/cloudsdk/channels/rapid/GoogleCloudSDKInstaller.exe"
        $tempPath = "$env:TEMP\GoogleCloudSDKInstaller.exe"
        
        Write-ColorOutput "Downloading Google Cloud SDK..." $Blue
        Invoke-WebRequest -Uri $gcpCliUrl -OutFile $tempPath
        
        Write-ColorOutput "Installing Google Cloud SDK..." $Blue
        Start-Process $tempPath -Wait -ArgumentList "/S"
        
        Write-ColorOutput "Google Cloud SDK installed successfully" $Green
    }
    catch {
        Write-ColorOutput "Failed to install Google Cloud SDK: $($_.Exception.Message)" $Red
    }
    finally {
        if (Test-Path $tempPath) {
            Remove-Item $tempPath -Force
        }
    }
}

function Create-DesktopShortcut {
    Write-ColorOutput "Creating desktop shortcuts..." $Blue
    
    $desktopPath = [Environment]::GetFolderPath("Desktop")
    $wshShell = New-Object -ComObject WScript.Shell
    
    # Create DriftMgr CLI shortcut
    $shortcut = $wshShell.CreateShortcut("$desktopPath\DriftMgr CLI.lnk")
    $shortcut.TargetPath = Join-Path $InstallPath "driftmgr.exe"
    $shortcut.WorkingDirectory = $InstallPath
    $shortcut.Description = "DriftMgr Command Line Interface"
    $shortcut.Save()
    
    # Create DriftMgr Server shortcut
    $serverShortcut = $wshShell.CreateShortcut("$desktopPath\DriftMgr Server.lnk")
    $serverShortcut.TargetPath = Join-Path $InstallPath "driftmgr-server.exe"
    $serverShortcut.WorkingDirectory = $InstallPath
    $serverShortcut.Description = "DriftMgr Web Server"
    $serverShortcut.Save()
    
    Write-ColorOutput "Desktop shortcuts created" $Green
}

function Show-InstallationSummary {
    Write-ColorOutput "`nDriftMgr Installation Complete!" $Green
    Write-ColorOutput "=====================================" $Blue
    
    Write-ColorOutput "Installation Path: $InstallPath" $Blue
    Write-ColorOutput "Executable: driftmgr.exe" $Blue
    Write-ColorOutput "Server: driftmgr-server.exe" $Blue
    
    Write-ColorOutput "`nNext Steps:" $Yellow
    Write-ColorOutput "1. Open a new terminal window (to refresh PATH)" $Blue
    Write-ColorOutput "2. Run 'driftmgr --help' to see available commands" $Blue
    Write-ColorOutput "3. Run 'driftmgr-server' to start the web dashboard" $Blue
    Write-ColorOutput "4. Configure your cloud credentials if not detected" $Blue
    
    Write-ColorOutput "`nDocumentation:" $Yellow
    Write-ColorOutput "- User Guide: $InstallPath\docs\user-guide\" $Blue
    Write-ColorOutput "- Examples: $InstallPath\examples\" $Blue
    
    Write-ColorOutput "`nQuick Start:" $Yellow
    Write-ColorOutput "driftmgr discover --provider aws --region us-east-1" $Blue
    Write-ColorOutput "driftmgr-server" $Blue
}

# Main installation script
try {
    Write-ColorOutput "DriftMgr Windows Installer" $Blue
    Write-ColorOutput "=============================" $Blue
    
    # Check if running as administrator (optional, but recommended)
    if (!(Test-Administrator)) {
        Write-ColorOutput "Running without administrator privileges. Some features may be limited." $Yellow
    }
    
    # Check if Go is installed
    if (!(Test-GoInstalled)) {
        Write-ColorOutput "Go is required for DriftMgr. Installing..." $Yellow
        Install-Go
    }
    
    # Install DriftMgr
    Install-DriftMgr
    
    # Add to PATH
    Add-ToPath
    
    # Create desktop shortcuts
    Create-DesktopShortcut
    
    # Check cloud credentials (unless skipped)
    if (!$SkipCredentialCheck) {
        Write-ColorOutput "`nChecking Cloud Provider Credentials..." $Blue
        
        $hasCredentials = $false
        
        if (Test-AWSCredentials) {
            $hasCredentials = $true
        } else {
            Write-ColorOutput "Would you like to install AWS CLI? (y/n)" $Yellow
            $response = Read-Host
            if ($response -eq "y" -or $response -eq "Y") {
                Install-AWS-CLI
            }
        }
        
        if (Test-AzureCredentials) {
            $hasCredentials = $true
        } else {
            Write-ColorOutput "Would you like to install Azure CLI? (y/n)" $Yellow
            $response = Read-Host
            if ($response -eq "y" -or $response -eq "Y") {
                Install-Azure-CLI
            }
        }
        
        if (Test-GCPCredentials) {
            $hasCredentials = $true
        } else {
            Write-ColorOutput "Would you like to install Google Cloud CLI? (y/n)" $Yellow
            $response = Read-Host
            if ($response -eq "y" -or $response -eq "Y") {
                Install-GCP-CLI
            }
        }
        
        if (!$hasCredentials) {
            Write-ColorOutput "`nNo cloud credentials detected. You'll need to configure them manually." $Yellow
            Write-ColorOutput "See the documentation for setup instructions." $Blue
        }
    }
    
    # Show installation summary
    Show-InstallationSummary
    
} catch {
    Write-ColorOutput "`nInstallation failed: $($_.Exception.Message)" $Red
    Write-ColorOutput "Please check the error and try again." $Red
    exit 1
}
