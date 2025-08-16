# DriftMgr Auto-Installer System

## 🎯 Overview

I've created a comprehensive auto-installer system for DriftMgr that provides a seamless installation experience across Windows, Linux, and macOS. The system automatically handles PATH setup, cloud credential detection, and environment configuration.

## 📁 Files Created

### Core Installer Files
- **`install.sh`** - Universal installer that detects OS and runs appropriate installer
- **`install.bat`** - Windows batch file for easy double-click installation
- **`installer/windows/install.ps1`** - Windows PowerShell installer
- **`installer/linux/install.sh`** - Linux/macOS bash installer
- **`installer/README.md`** - Comprehensive documentation

## 🚀 Key Features

### ✅ Automatic Setup
- **OS Detection** - Automatically detects Windows, Linux, or macOS
- **PATH Integration** - Adds DriftMgr to system PATH for global access
- **Desktop Shortcuts** - Creates desktop shortcuts for easy access
- **Configuration Copying** - Copies all necessary config files and assets

### 🔍 Cloud Credential Detection
The installer automatically detects and reports on:

#### AWS Credentials
- AWS CLI configuration files (`~/.aws/config`, `~/.aws/credentials`)
- Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
- AWS CLI authentication status

#### Azure Credentials
- Azure CLI authentication status
- Environment variables (`AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, `AZURE_TENANT_ID`)

#### Google Cloud Credentials
- Google Cloud CLI authentication status
- Service account key files (`GOOGLE_APPLICATION_CREDENTIALS`)

### 🛠️ Optional CLI Installation
If cloud credentials aren't detected, the installer can optionally install:
- **AWS CLI** - Downloads and installs AWS CLI
- **Azure CLI** - Downloads and installs Azure CLI  
- **Google Cloud CLI** - Downloads and installs Google Cloud SDK

### 🔧 Smart Dependencies
- **Go Installation** - Automatically downloads and installs Go if not present
- **Architecture Detection** - Supports x86_64 and ARM64 architectures
- **Package Manager Detection** - Uses appropriate package managers (apt, yum, dnf, pacman)

## 🎯 Usage Examples

### Windows Users
```powershell
# Option 1: Universal installer (recommended)
.\install.sh

# Option 2: Windows-specific installer
.\installer\windows\install.ps1

# Option 3: Double-click install.bat
# Just double-click the install.bat file
```

### Linux/macOS Users
```bash
# Option 1: Universal installer (recommended)
./install.sh

# Option 2: Linux-specific installer
./installer/linux/install.sh
```

### Advanced Options
```bash
# Install to custom location
./install.sh --path /opt/driftmgr

# Skip credential check (for automated installations)
./install.sh --skip-credentials

# Force installation (overwrite existing)
./install.sh --force
```

## 📋 Installation Process

### 1. Prerequisites Check
- Verifies DriftMgr binaries exist
- Checks for Go installation
- Validates configuration files

### 2. OS Detection & Setup
- Detects operating system and architecture
- Identifies appropriate package manager
- Sets up installation paths

### 3. Application Installation
- Copies executables to installation directory
- Copies configuration files and assets
- Sets proper file permissions

### 4. Environment Configuration
- Adds DriftMgr to system PATH
- Creates desktop shortcuts
- Updates shell configuration files

### 5. Cloud Credential Analysis
- Scans for existing cloud credentials
- Reports detected configurations
- Offers to install missing CLI tools

### 6. Installation Summary
- Shows installation path and next steps
- Provides quick start commands
- Links to documentation

## 🔧 Technical Details

### Windows Implementation
- **PowerShell Script** - Uses PowerShell for robust Windows integration
- **Execution Policy** - Handles PowerShell execution policy automatically
- **Registry Integration** - Updates user PATH in Windows registry
- **Desktop Shortcuts** - Creates Windows shortcut files (.lnk)

### Linux/macOS Implementation
- **Bash Script** - Uses bash for cross-platform compatibility
- **Package Manager Support** - Supports apt, yum, dnf, zypper, pacman
- **Shell Integration** - Updates ~/.bashrc or ~/.zshrc
- **Desktop Entries** - Creates .desktop files for Linux desktop environments

### Universal Features
- **Error Handling** - Comprehensive error checking and reporting
- **Colored Output** - User-friendly colored terminal output
- **Progress Indicators** - Shows installation progress
- **Rollback Support** - Handles installation failures gracefully

## 🚨 Safety Features

### Pre-Installation Checks
- Validates prerequisites before installation
- Checks for existing installations
- Verifies file permissions and access

### Safe Installation
- Uses user directories (no system-wide changes)
- Preserves existing configurations
- Provides dry-run options

### Error Recovery
- Detailed error messages and troubleshooting
- Graceful failure handling
- Manual installation fallback options

## 📊 Benefits for Users

### 🎯 For New Users
- **One-Command Installation** - Single command to get started
- **Automatic Setup** - No manual PATH configuration needed
- **Cloud Integration** - Automatic credential detection and setup
- **Desktop Access** - Easy access via desktop shortcuts

### 🎯 For Experienced Users
- **Custom Installation Paths** - Install to any directory
- **Automated Deployments** - Skip credential checks for CI/CD
- **Advanced Options** - Force installation, custom configurations
- **Manual Override** - Full control over installation process

### 🎯 For Organizations
- **Standardized Installation** - Consistent setup across teams
- **Automated Onboarding** - Reduce setup time for new team members
- **Cloud Provider Support** - Works with AWS, Azure, and GCP
- **Cross-Platform** - Same experience on Windows, Linux, and macOS

## 🔄 Maintenance

### Updates
- Installer can be updated independently of DriftMgr
- Supports force installation to update existing installations
- Preserves user configurations during updates

### Uninstallation
- Provides uninstallation instructions
- Removes from PATH automatically
- Cleans up desktop shortcuts

## 📈 Future Enhancements

### Planned Features
- **Silent Installation** - Non-interactive installation for automation
- **Configuration Import** - Import existing DriftMgr configurations
- **Plugin Installation** - Install additional DriftMgr plugins
- **Update Notifications** - Check for DriftMgr updates

### Potential Improvements
- **Docker Support** - Container-based installation option
- **Package Manager Integration** - Native package manager support
- **Enterprise Features** - Corporate deployment options
- **Multi-User Support** - System-wide installation options

---

## 🎉 Summary

The DriftMgr auto-installer system provides a professional, user-friendly installation experience that:

1. **Reduces Friction** - One command to get started
2. **Handles Complexity** - Automatic PATH setup and credential detection
3. **Works Everywhere** - Cross-platform support for Windows, Linux, and macOS
4. **Provides Safety** - Comprehensive error handling and validation
5. **Offers Flexibility** - Custom installation options for advanced users

This installer significantly improves the user experience for DriftMgr, making it much easier for new users to get started while providing the flexibility that experienced users need.
