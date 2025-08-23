# DriftMgr Auto-Installer

This directory contains auto-installers for DriftMgr that automatically set up the application, add it to your PATH, detect cloud credentials, and configure your environment.

## Quick Start

### Windows Users
```powershell
# Option 1: Run the universal installer (recommended)
.\install.sh

# Option 2: Run the Windows-specific installer
.\installer\windows\install.ps1

# Option 3: Double-click install.bat
```

### Linux/macOS Users
```bash
# Option 1: Run the universal installer (recommended)
./install.sh

# Option 2: Run the Linux-specific installer
./installer/linux/install.sh
```

## What the Installer Does

### [OK] Automatic Setup
- **Installs DriftMgr** to your user directory (`~/driftmgr` on Linux/macOS, `%USERPROFILE%\driftmgr` on Windows)
- **Adds to PATH** so you can run `driftmgr` from anywhere
- **Creates desktop shortcuts** for easy access
- **Copies configuration files** and assets

### Cloud Credential Detection
The installer automatically detects and reports on your cloud provider credentials:

#### AWS Credentials
- AWS CLI configuration (`~/.aws/config`)
- AWS credentials file (`~/.aws/credentials`)
- Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
- AWS CLI authentication status

#### Azure Credentials
- Azure CLI authentication status
- Environment variables (`AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, `AZURE_TENANT_ID`)

#### Google Cloud Credentials
- Google Cloud CLI authentication status
- Service account key file (`GOOGLE_APPLICATION_CREDENTIALS`)

### Optional CLI Installation
If cloud credentials aren't detected, the installer can optionally install:
- **AWS CLI** - For AWS resource management
- **Azure CLI** - For Azure resource management
- **Google Cloud CLI** - For GCP resource management

## Installation Structure

After installation, your DriftMgr installation will look like this:

```
~/driftmgr/ (or %USERPROFILE%\driftmgr\)
 driftmgr # Main CLI executable
 driftmgr-server # Web server executable
 driftmgr.yaml # Configuration file
 go.mod # Go module file
 go.sum # Go checksums
 assets/ # Static assets and data
 docs/ # Documentation
 examples/ # Example configurations
```

## Installation Options

### Universal Installer Options
```bash
./install.sh [OPTIONS]

Options:
 -p, --path PATH Installation path (default: ~/driftmgr)
 -f, --force Force installation (overwrite existing)
 -s, --skip-credentials Skip cloud credential check
 -h, --help Show help message
```

### Windows PowerShell Installer Options
```powershell
.\installer\windows\install.ps1 [OPTIONS]

Options:
 -InstallPath PATH Installation path (default: $env:USERPROFILE\driftmgr)
 -Force Force installation (overwrite existing)
 -SkipCredentialCheck Skip cloud credential check
```

### Linux/macOS Installer Options
```bash
./installer/linux/install.sh [OPTIONS]

Options:
 -p, --path PATH Installation path (default: ~/driftmgr)
 -f, --force Force installation (overwrite existing)
 -s, --skip-credentials Skip cloud credential check
 -h, --help Show help message
```

## Examples

### Basic Installation
```bash
# Install to default location
./install.sh

# Install to custom location
./install.sh --path /opt/driftmgr

# Skip credential check (for automated installations)
./install.sh --skip-credentials
```

### Windows Examples
```powershell
# Install to default location
.\installer\windows\install.ps1

# Install to custom location
.\installer\windows\install.ps1 -InstallPath "C:\Tools\driftmgr"

# Force installation (overwrite existing)
.\installer\windows\install.ps1 -Force
```

## Prerequisites

### Required
- **Go 1.21+** - The installer will download and install Go if not present
- **DriftMgr binaries** - Must be built before running installer (`make build`)

### Optional (for cloud provider integration)
- **AWS CLI** - For AWS resource management
- **Azure CLI** - For Azure resource management
- **Google Cloud CLI** - For GCP resource management

## Troubleshooting

### Common Issues

#### "DriftMgr executable not found"
**Solution**: Build the application first:
```bash
make build
# or
go build ./cmd/driftmgr
go build ./cmd/driftmgr-server
```

#### "PowerShell execution policy error" (Windows)
**Solution**: The installer uses `-ExecutionPolicy Bypass` to handle this automatically. If you still get errors, run:
```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

#### "Permission denied" (Linux/macOS)
**Solution**: Make the installer executable:
```bash
chmod +x install.sh
chmod +x installer/linux/install.sh
```

#### "Go not found"
**Solution**: The installer will automatically download and install Go. If it fails, install Go manually:
- **Windows**: Download from https://golang.org/dl/
- **Linux/macOS**: Use your package manager or download from https://golang.org/dl/

### PATH Issues

#### DriftMgr not found in PATH after installation
**Solution**:
1. **Windows**: Open a new command prompt (PATH changes require new session)
2. **Linux/macOS**: Restart terminal or run:
 ```bash
 source ~/.bashrc # or ~/.zshrc for zsh
 ```

#### Manual PATH addition
If automatic PATH addition fails, add manually:

**Windows**:
```powershell
# Add to user PATH
[Environment]::SetEnvironmentVariable("PATH", $env:PATH + ";$env:USERPROFILE\driftmgr", "User")
```

**Linux/macOS**:
```bash
# Add to ~/.bashrc or ~/.zshrc
echo 'export PATH="$PATH:$HOME/driftmgr"' >> ~/.bashrc
source ~/.bashrc
```

## Manual Installation

If you prefer to install manually or the auto-installer doesn't work for your environment:

### Windows Manual Installation
1. Copy `bin/driftmgr.exe` and `bin/driftmgr-server.exe` to a directory
2. Add the directory to your PATH
3. Copy configuration files (`driftmgr.yaml`, etc.)
4. Configure cloud credentials manually

### Linux/macOS Manual Installation
1. Copy `bin/driftmgr` and `bin/driftmgr-server` to a directory
2. Add the directory to your PATH in `~/.bashrc` or `~/.zshrc`
3. Copy configuration files (`driftmgr.yaml`, etc.)
4. Configure cloud credentials manually

## Support

If you encounter issues with the installer:

1. **Check the troubleshooting section** above
2. **Review the logs** - The installer provides detailed output
3. **Try manual installation** as a fallback
4. **Open an issue** on GitHub with:
 - Operating system and version
 - Installer command used
 - Full error output
 - Steps to reproduce

## Uninstallation

To uninstall DriftMgr:

### Windows
```powershell
# Remove from PATH
$currentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
$newPath = ($currentPath.Split(';') | Where-Object { $_ -notlike "*driftmgr*" }) -join ';'
[Environment]::SetEnvironmentVariable("PATH", $newPath, "User")

# Remove installation directory
Remove-Item "$env:USERPROFILE\driftmgr" -Recurse -Force

# Remove desktop shortcuts
Remove-Item "$env:USERPROFILE\Desktop\DriftMgr*.lnk" -Force
```

### Linux/macOS
```bash
# Remove from PATH (edit ~/.bashrc or ~/.zshrc)
# Remove the line: export PATH="$PATH:$HOME/driftmgr"

# Remove installation directory
rm -rf ~/driftmgr

# Remove desktop shortcuts
rm -f ~/.local/share/applications/driftmgr-*.desktop
```

---

**DriftMgr Auto-Installer** - Making cloud infrastructure management easier!
