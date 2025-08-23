# Credential Auto-Detection Feature

## Overview

DriftMgr now includes complete auto-detection capabilities for all four supported cloud providers: **AWS**, **Azure**, **GCP**, and **DigitalOcean**. This feature automatically discovers and configures credentials from standard locations and environment variables, significantly reducing manual setup time.

## Supported Providers

### 1. AWS (Amazon Web Services)
**Detection Sources:**
- Environment variables: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`
- AWS CLI credentials file: `~/.aws/credentials`
- AWS CLI config file: `~/.aws/config`
- AWS SSO configuration: `~/.aws/sso/cache`
- AWS CLI v2 cache: `~/.aws/cli/cache`
- AWS SAM CLI: `~/.aws-sam`

### 2. Azure (Microsoft Azure)
**Detection Sources:**
- Environment variables: `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, `AZURE_TENANT_ID`
- Azure CLI credentials: `~/.azure/credentials.json`
- Azure CLI access tokens: `~/.azure/accessTokens.json`
- Azure CLI profiles: `~/.azure/azureProfile.json`
- Azure CLI config: `~/.azure/config`
- Azure CLI extensions: `~/.azure/cliextensions`

### 3. GCP (Google Cloud Platform)
**Detection Sources:**
- Environment variable: `GOOGLE_APPLICATION_CREDENTIALS`
- gcloud application default credentials: `~/.config/gcloud/application_default_credentials.json`
- gcloud configuration: `~/.config/gcloud/configurations/config_default`
- gcloud active account: `~/.config/gcloud/active_config`
- gcloud credentials directory: `~/.config/gcloud/credentials`
- gcloud properties: `~/.config/gcloud/properties`
- gcloud logs: `~/.config/gcloud/logs`
- gcloud cache: `~/.config/gcloud/cache`
- gcloud auth directory: `~/.config/gcloud/auth`

### 4. DigitalOcean
**Detection Sources:**
- Environment variable: `DIGITALOCEAN_TOKEN`
- DigitalOcean CLI credentials: `~/.digitalocean/credentials`
- DigitalOcean CLI config: `~/.digitalocean/config`
- doctl configuration: `~/.config/doctl/config.yaml`
- DigitalOcean CLI cache: `~/.digitalocean/cache`
- DigitalOcean CLI logs: `~/.digitalocean/logs`

## Usage

### Command Line Interface

#### Auto-Detect All Providers
```bash
driftmgr credentials auto-detect
```

**Output Example:**
```
Auto-detecting cloud provider credentials...
=============================================

Checking AWS credentials... Found in AWS CLI credentials file
Checking Azure credentials... Found Azure CLI profile
Checking GCP credentials... Found gcloud application default credentials
Checking DigitalOcean credentials... Found in DigitalOcean CLI credentials file

Auto-detection Summary:
=======================
 Successfully detected 4 provider(s): AWS, Azure, GCP, DigitalOcean

Detected credentials are now available for use with DriftMgr.
You can verify the configuration by running: driftmgr credentials list
```

#### List Detected Providers
```bash
driftmgr credentials list
```

#### Get Help
```bash
driftmgr credentials help
```

### TUI Integration

When DriftMgr starts, the TUI automatically displays detected credentials below the ASCII art banner:

```
================================================================================
DriftMgr - Cloud Infrastructure Drift Detection and Remediation
================================================================================
Version 1.6.4 - Enhanced Multi-Cloud Architecture
Discover • Analyze • Monitor • Remediate

Author: Catherine Vee
GitHub: https://github.com/catherinevee/driftmgr
License: MIT

Cloud Provider Credentials:
==========================
 Detected 4 provider(s): AWS, Azure, GCP, DigitalOcean

Welcome to DriftMgr Interactive Shell!
...
```

### Installation Scripts

#### Linux/macOS Installer
```bash
./install.sh --update-credentials
```

#### Windows Installer
```powershell
.\install.ps1 -UpdateCredentials
```

## Benefits

1. **Zero Configuration**: Automatically detects existing credentials without manual setup
2. **Multi-Provider Support**: Handles all four major cloud providers seamlessly
3. **Complete Detection**: Checks multiple credential sources for each provider
4. **User-Friendly**: Clear feedback on what credentials were found and where
5. **TUI Integration**: Shows credential status immediately upon startup
6. **Cross-Platform**: Works on Windows, Linux, and macOS

## Detection Priority

The auto-detection follows this priority order for each provider:

1. **Environment Variables** (highest priority)
2. **CLI Tool Configurations**
3. **Standard Credential Files**
4. **Cache and Log Directories**
5. **SSO and Token Configurations**

## Error Handling

- **No Credentials Found**: Provides helpful guidance on how to set up credentials
- **Partial Detection**: Shows which providers were detected and which were not
- **Invalid Credentials**: Validates credentials and reports issues
- **Permission Errors**: Gracefully handles file access issues

## Testing

Complete test scripts are available to verify auto-detection functionality:

- **Linux/macOS**: `scripts/test-all-providers.sh`
- **Windows**: `scripts/test-all-providers.ps1`

These scripts test:
- All four providers detection
- TUI credential display
- Individual provider detection
- Credential listing
- Help command functionality

## Troubleshooting

### Common Issues

1. **No Credentials Detected**
 - Ensure you have installed and configured the respective cloud CLI tools
 - Check that credential files exist in standard locations
 - Verify environment variables are set correctly

2. **Partial Detection**
 - Some providers may not be configured on your system
 - Use `driftmgr credentials setup` to configure missing providers

3. **Permission Errors**
 - Ensure DriftMgr has read access to credential directories
 - Check file permissions on credential files

### Manual Setup

If auto-detection doesn't find your credentials, you can manually set them up:

```bash
driftmgr credentials setup
```

This provides an interactive setup process for each provider.

## Security Considerations

- Credentials are read from standard locations only
- No credentials are stored or transmitted by DriftMgr
- Environment variables are checked but not modified
- File permissions are respected during detection

## Future Enhancements

- Support for additional cloud providers
- Enhanced credential validation
- Integration with cloud provider SDKs
- Support for enterprise SSO configurations
- Credential rotation detection

For more information, see the main [README.md](README.md) and [INSTALLER_SUMMARY.md](INSTALLER_SUMMARY.md) files.
