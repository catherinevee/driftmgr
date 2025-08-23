# DriftMgr Credential Display Demo

## Main Menu View

When DriftMgr starts, it automatically detects credentials for all supported cloud providers and displays them prominently:

```
╭─────────────────────────────────────────────────────────────╮
│ Cloud Credentials: AWS: ✓ | Azure: ✓ | GCP: ✗ | DigitalOcean: ✗ │
│ Press 'c' for configuration help                             │
╰─────────────────────────────────────────────────────────────╯
```

### Icons Explained:
- ✓ (Green) = Configured and valid credentials
- ✗ (Gray) = Not configured
- ⚠ (Yellow) = Configured but has errors (expired, invalid, etc.)

## Status Bar

The bottom status bar shows a quick summary:
```
DriftMgr v1.0.0 | ✓ AWS, Azure | ✗ GCP, DigitalOcean     Press ? for help | q to quit
```

## Configuration View (Press 'c')

Detailed credential information with setup instructions:

```
Configuration & Settings
════════════════════════════════════════════════════════════════

Auto-Detected Cloud Credentials:

✓ AWS
  Source: file
  Account: 0250****4478
  Region: us-west-2
  
✓ Azure  
  Source: cli
  Subscription: 4842****cfe6
  
✗ GCP
  No GCP credentials found
  → Run: gcloud auth login
  
✗ DigitalOcean
  No DigitalOcean credentials found
  → Run: doctl auth init

════════════════════════════════════════════════════════════════
```

## Benefits of This Display

1. **Immediate Visibility**: Users know instantly which cloud providers they can access
2. **All Providers Shown**: Even unconfigured providers are displayed with setup help
3. **Security**: Account IDs are partially masked for security
4. **Actionable**: Provides exact commands to configure missing providers
5. **Multiple Locations**: Credentials shown in:
   - Main menu (detailed box)
   - Status bar (summary)
   - Configuration view (full details with help)

## Credential Sources Detected

DriftMgr checks multiple sources for each provider:

### AWS
- Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)
- ~/.aws/credentials file
- ~/.aws/config file
- IAM instance roles (when running on EC2)

### Azure
- Azure CLI (az login)
- Environment variables (AZURE_CLIENT_ID, AZURE_CLIENT_SECRET)
- Managed Service Identity

### GCP
- gcloud CLI authentication
- GOOGLE_APPLICATION_CREDENTIALS environment variable
- ~/.config/gcloud/application_default_credentials.json
- Service account JSON files

### DigitalOcean
- doctl CLI authentication
- DIGITALOCEAN_TOKEN environment variable
- ~/.config/doctl/config.yaml

## Usage Flow

1. **Start DriftMgr**: Credentials auto-detected immediately
2. **Check Status**: See which providers are available at a glance
3. **Configure Missing**: Press 'c' to see setup instructions
4. **Run Commands**: Copy and run the provided setup commands
5. **Refresh**: Press 'r' in config view to re-detect credentials
6. **Discover Resources**: Only configured providers will work for discovery

This comprehensive credential display ensures users always know:
- Which cloud providers they can access
- How to configure missing providers
- Whether their credentials are valid
- What account/project they're connected to