# DriftMgr Command Examples

## Verified Working Commands

Based on actual testing with the current DriftMgr implementation, these commands are confirmed to work:

### Core Commands

#### Status (Auto-Discovery)
```bash
# Show system status and auto-discover resources
driftmgr status
```

#### Discovery
```bash
# Auto-discover all configured providers
driftmgr discover --auto

# Discover with all accounts/subscriptions
driftmgr discover --auto --all-accounts

# Discover specific provider
driftmgr discover --provider aws

# Check credential status (replaces deprecated 'credentials' command)
driftmgr discover --credentials
```

#### Drift Detection
```bash
# Detect drift with smart defaults (75-85% noise reduction)
driftmgr drift detect --provider aws

# Use environment-specific thresholds
driftmgr drift detect --environment production
driftmgr drift detect --environment staging

# Disable smart filtering to see all drift
driftmgr drift detect --no-smart-defaults

# Detect drift for all providers
driftmgr drift detect --provider all
```

#### Drift Management
```bash
# Generate drift report
driftmgr drift report --format html
driftmgr drift report --format json

# Generate remediation plan (dry-run)
driftmgr drift fix --dry-run

# Apply remediation (with confirmation)
driftmgr drift fix --apply

# Manage auto-remediation
driftmgr drift auto-remediate enable --dry-run
driftmgr drift auto-remediate disable
```

### State Management

#### State Inspection
```bash
# Inspect Terraform state file
driftmgr state inspect terraform.tfstate

# Visualize state file
driftmgr state visualize --state terraform.tfstate

# Analyze state file (alternate command)
driftmgr tfstate --file terraform.tfstate
```

#### State Scanning
```bash
# Scan directory for Terraform backends
driftmgr scan --path ./terraform

# Scan current directory
driftmgr scan --path .
```

### Resource Management

#### Export
```bash
# Export discovery results to JSON
driftmgr export --format json --output resources.json

# Export to CSV
driftmgr export --format csv --output resources.csv

# Export to HTML report
driftmgr export --format html --output report.html
```

#### Import
```bash
# Import existing resources into Terraform
driftmgr import --resource-id i-1234567890 --type aws_instance
```

#### Delete
```bash
# Delete a specific resource (with confirmation)
driftmgr delete --resource-id i-1234567890 --provider aws

# Dry-run deletion
driftmgr delete --resource-id i-1234567890 --provider aws --dry-run
```

### Account Management

```bash
# List all accessible cloud accounts
driftmgr accounts

# List AWS accounts
driftmgr accounts --provider aws

# List Azure subscriptions
driftmgr accounts --provider azure
```

### Server & Dashboard

#### Web Dashboard
```bash
# Start web dashboard (default port 8080)
driftmgr serve web

# Custom port
driftmgr serve web --port 9090

# With authentication
driftmgr serve web --auth-enabled
```

#### API Server
```bash
# Start REST API server
driftmgr serve api --port 8081

# With WebSocket support
driftmgr serve api --enable-websocket
```

### Verification & Validation

```bash
# Verify discovery accuracy
driftmgr verify --provider aws

# Verify with detailed output
driftmgr verify --provider aws --verbose

# Verify all providers
driftmgr verify --provider all
```

## Common Workflows

### Initial Setup
```bash
# 1. Check system status
driftmgr status

# 2. Verify credentials
driftmgr discover --credentials

# 3. Run first discovery
driftmgr discover --auto

# 4. Detect drift
driftmgr drift detect --provider all
```

### Daily Operations
```bash
# Morning drift check
driftmgr drift detect --environment production

# Generate report for review
driftmgr drift report --format html --output daily-drift.html

# Fix critical drift
driftmgr drift fix --dry-run
driftmgr drift fix --apply
```

### CI/CD Integration
```bash
# Automated drift detection
driftmgr drift detect --provider all --format json --output drift.json

# Fail on critical drift
driftmgr drift detect --severity critical --fail-on-drift

# Auto-remediation in staging
driftmgr drift auto-remediate enable --environment staging
```

## Flags Reference

### Global Flags
- `--auto` - Auto-discover all configured providers
- `--all-accounts` - Include all accessible accounts/subscriptions
- `--smart-defaults` - Enable smart filtering (default: true)
- `--no-smart-defaults` - Disable smart filtering
- `--environment` - Set environment (production/staging/development)
- `--verbose` - Verbose output
- `--debug` - Debug output
- `--quiet` - Suppress non-error output

### Provider Flags
- `--provider` - Specify provider (aws/azure/gcp/digitalocean/all)
- `--region` - Specific region(s) to scan
- `--account` - Specific account/subscription to use

### Output Flags
- `--format` - Output format (json/yaml/csv/html/table)
- `--output` - Output file path
- `--no-color` - Disable colored output

### Safety Flags
- `--dry-run` - Preview changes without applying
- `--force` - Skip confirmation prompts
- `--backup` - Create backup before changes

## Deprecated Commands

These commands are deprecated but still work with warnings:
- `driftmgr credentials` → Use `driftmgr discover --credentials`
- `driftmgr dashboard` → Use `driftmgr serve web`
- `driftmgr server` → Use `driftmgr serve api`

## Notes

1. **Smart Defaults**: Enabled by default, reducing noise by 75-85%
2. **Multi-Account**: Automatically discovers all accessible accounts when using `--all-accounts`
3. **Environment Thresholds**: Different filtering levels for production/staging/development
4. **Parallel Processing**: Automatic parallel discovery for better performance
5. **Rate Limiting**: Built-in rate limiting to avoid API throttling

---

*All examples verified with DriftMgr v2.0.0*