# DriftMgr Credential Auto-Detection Feature

This document describes the enhanced credential management functionality in DriftMgr, which now includes automatic detection of cloud provider credentials from standard locations and environment variables.

## Overview

DriftMgr now automatically detects cloud provider credentials from standard locations, eliminating the need for manual configuration in most cases. This is particularly useful when:

- You already have cloud provider CLI tools installed (AWS CLI, Azure CLI, gcloud, doctl)
- Credentials are stored in standard locations
- Environment variables are set for cloud access
- You want to quickly get started without manual setup

## Features

### 1. Automatic Credential Detection

DriftMgr automatically detects credentials from:

#### **AWS (Amazon Web Services)**
- Environment variables: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`
- AWS CLI credentials file: `~/.aws/credentials`
- AWS CLI config file: `~/.aws/config`
- AWS SSO configuration: `~/.aws/sso/cache`

#### **Azure (Microsoft Azure)**
- Environment variables: `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, `AZURE_TENANT_ID`
- Azure CLI credentials: `~/.azure/credentials.json`
- Azure CLI access tokens: `~/.azure/accessTokens.json`
- Azure CLI profiles: `~/.azure/azureProfile.json`

#### **GCP (Google Cloud Platform)**
- Environment variable: `GOOGLE_APPLICATION_CREDENTIALS` (service account key file)
- gcloud application default credentials: `~/.config/gcloud/application_default_credentials.json`
- gcloud configuration: `~/.config/gcloud/configurations/config_default`
- gcloud active account: `~/.config/gcloud/active_config`

#### **DigitalOcean**
- Environment variable: `DIGITALOCEAN_TOKEN`
- DigitalOcean CLI credentials: `~/.digitalocean/credentials`
- DigitalOcean CLI config: `~/.digitalocean/config`

### 2. Install Script Integration

The main install script now uses auto-detection by default:

```bash
# Auto-detect credentials via install script
./install.sh --update-credentials
```

### 3. CLI Command

Direct CLI access to auto-detection:

```bash
# Auto-detect credentials via CLI
driftmgr credentials auto-detect
```

**Available Commands:**
- `credentials auto-detect`: Auto-detect credentials from standard locations
- `credentials setup`: Interactive setup for new credentials
- `credentials update`: Update existing credentials
- `credentials list`: List configured cloud providers
- `credentials validate <provider>`: Validate credentials for a specific provider
- `credentials help`: Show help information

### 4. Cross-Platform Support

The feature works across all supported platforms:

- **Windows**: PowerShell installer with auto-detection
- **Linux/macOS**: Bash installer with auto-detection
- **Universal**: Main install script detects OS and delegates appropriately

## How It Works

### 1. Auto-Detection Process

When auto-detecting credentials, the system:

1. **Checks Environment Variables**: Looks for standard cloud provider environment variables
2. **Scans Standard Locations**: Checks common credential file locations
3. **Validates CLI Tools**: Detects existing CLI tool configurations
4. **Provides Summary**: Shows which providers were detected and which were not
5. **Offers Guidance**: Suggests next steps for missing credentials

### 2. Detection Priority

The system checks credentials in the following order:

1. **Environment Variables** (highest priority)
2. **CLI Tool Configurations**
3. **Standard Credential Files**
4. **SSO/Token Configurations**

### 3. Security Features

- **Read-Only Detection**: Only reads existing credentials, never modifies them
- **No Credential Storage**: Does not store credentials in DriftMgr-specific locations
- **Environment Variable Support**: Respects existing environment variable configurations
- **CLI Tool Integration**: Works seamlessly with existing CLI tool setups

## Usage Examples

### Basic Usage

```bash
# Auto-detect credentials via install script
./install.sh --update-credentials

# Auto-detect credentials via CLI
driftmgr credentials auto-detect

# List current credentials
driftmgr credentials list

# Validate specific provider
driftmgr credentials validate aws
```

### Advanced Usage

```bash
# Force installation with credential auto-detection
./install.sh --force --update-credentials

# Auto-detect credentials for specific installation path (Windows)
./installer/windows/install.ps1 -UpdateCredentials -InstallPath "C:\CustomPath"

# Auto-detect credentials for specific installation path (Linux)
./installer/linux/install.sh --update-credentials --path "/opt/driftmgr"
```

### Auto-Detection Output

When running `driftmgr credentials auto-detect`, you'll see:

```
Auto-detecting cloud provider credentials...
=============================================

Checking AWS credentials... ✓ Found in AWS CLI credentials file
Checking Azure credentials... ✓ Found Azure CLI profile
Checking GCP credentials... ✗ Not found
Checking DigitalOcean credentials... ✗ Not found

Auto-detection Summary:
=======================
✓ Successfully detected 2 provider(s): AWS, Azure

Detected credentials are now available for use with DriftMgr.
You can verify the configuration by running: driftmgr credentials list

⚠ No cloud provider credentials were auto-detected.

Common credential locations checked:
  • AWS: ~/.aws/credentials, AWS_ACCESS_KEY_ID env var
  • Azure: ~/.azure/credentials.json, az CLI, env vars
  • GCP: ~/.config/gcloud/, GOOGLE_APPLICATION_CREDENTIALS env var
  • DigitalOcean: ~/.digitalocean/credentials, DIGITALOCEAN_TOKEN env var

To manually configure credentials, run: driftmgr credentials setup
```

## Implementation Details

### Files Modified

1. **Credential Manager** (`internal/credentials/manager.go`)
   - Added `AutoDetectCredentials()` method
   - Added individual auto-detection methods for each provider
   - Enhanced credential detection logic

2. **CLI Handler** (`cmd/driftmgr-client/credentials.go`)
   - Added `auto-detect` command to credentials handler
   - Added `handleCredentialsAutoDetect()` function
   - Updated help text and command structure

3. **Install Scripts**
   - **Main Install Script** (`install.sh`): Updated to use auto-detection
   - **Windows Installer** (`installer/windows/install.ps1`): Updated to use auto-detection
   - **Linux Installer** (`installer/linux/install.sh`): Updated to use auto-detection

### Test Scripts

- **Bash Test Script** (`scripts/test-credential-update.sh`)
- **PowerShell Test Script** (`scripts/test-credential-update.ps1`)

Both scripts test the functionality and provide usage examples.

## Benefits

### 1. User Experience
- **Zero Configuration**: Works out of the box with existing cloud setups
- **Fast Setup**: No manual credential entry required
- **Seamless Integration**: Works with existing CLI tools and configurations
- **Cross-Platform**: Consistent experience across all operating systems

### 2. Security
- **No Credential Storage**: Uses existing secure credential storage
- **Environment Variable Support**: Respects existing security practices
- **CLI Tool Integration**: Leverages existing secure CLI tool configurations
- **Read-Only Access**: Only reads credentials, never modifies them

### 3. Maintenance
- **Automatic Updates**: Detects credential changes automatically
- **Multiple Sources**: Supports various credential storage methods
- **Backward Compatibility**: Existing manual setup still works
- **Flexible Configuration**: Supports both auto-detection and manual setup

## Troubleshooting

### Common Issues

1. **No Credentials Detected**
   ```bash
   # Check if credentials exist in standard locations
   ls ~/.aws/credentials
   ls ~/.azure/credentials.json
   ls ~/.config/gcloud/
   ls ~/.digitalocean/credentials
   ```

2. **Environment Variables Not Set**
   ```bash
   # Check environment variables
   echo $AWS_ACCESS_KEY_ID
   echo $AZURE_CLIENT_ID
   echo $GOOGLE_APPLICATION_CREDENTIALS
   echo $DIGITALOCEAN_TOKEN
   ```

3. **CLI Tools Not Installed**
   ```bash
   # Install CLI tools for auto-detection
   # AWS CLI
   curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
   unzip awscliv2.zip
   sudo ./aws/install
   
   # Azure CLI
   curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash
   
   # Google Cloud CLI
   curl https://sdk.cloud.google.com | bash
   exec -l $SHELL
   
   # DigitalOcean CLI
   snap install doctl
   ```

### Getting Help

```bash
# Show install script help
./install.sh --help

# Show credentials help
driftmgr credentials help

# Run test script
./scripts/test-credential-update.sh
# or
./scripts/test-credential-update.ps1
```

## Future Enhancements

Potential improvements for future versions:

1. **Credential Validation**: Automatically test detected credentials
2. **Credential Rotation**: Detect and handle credential expiration
3. **Multi-Account Support**: Detect multiple accounts per provider
4. **Credential Backup**: Backup existing credentials before changes
5. **Audit Trail**: Log credential detection activities
6. **Integration**: Integration with cloud provider credential managers

## Conclusion

The auto-detection feature significantly improves the user experience for DriftMgr credential management. It provides a seamless, secure way to use existing cloud credentials without requiring manual configuration or credential re-entry.

For more information, see the main [README.md](README.md) and [INSTALLER_SUMMARY.md](INSTALLER_SUMMARY.md) files.
