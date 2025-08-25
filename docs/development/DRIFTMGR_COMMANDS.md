# DriftMgr Complete Command Reference

## Core Commands

### `driftmgr status`
Show system status and auto-discover resources across all configured cloud providers.
```bash
driftmgr status
```

### `driftmgr discover`
Discover cloud resources across providers.
```bash
# Auto-discover all configured providers
driftmgr discover --auto

# Discover specific provider
driftmgr discover --provider aws

# Show credential status
driftmgr discover --credentials

# Include all accessible accounts
driftmgr discover --all-accounts

# Filter by resource type
driftmgr discover --type EC2Instance

# Filter by tags
driftmgr discover --tag Environment=Production
```

## Drift Management Commands

### `driftmgr drift detect`
Detect infrastructure drift between Terraform state and actual cloud resources.
```bash
# Basic drift detection
driftmgr drift detect --provider aws

# With smart defaults (85% noise reduction)
driftmgr drift detect --smart-defaults

# Environment-specific thresholds
driftmgr drift detect --environment production

# Disable smart filtering
driftmgr drift detect --no-smart-defaults

# Specific state file
driftmgr drift detect --state terraform.tfstate

# Cost impact analysis
driftmgr drift detect --cost-impact

# Security-focused detection
driftmgr drift detect --security-only
```

### `driftmgr drift report`
Generate drift analysis reports in various formats.
```bash
# Generate HTML report
driftmgr drift report --format html

# Generate JSON report
driftmgr drift report --format json

# Generate CSV for spreadsheets
driftmgr drift report --format csv

# Include remediation suggestions
driftmgr drift report --include-remediation

# Output to file
driftmgr drift report --output drift-report.html
```

### `driftmgr drift fix`
Generate or apply remediation plans for detected drift.
```bash
# Generate remediation plan (dry run)
driftmgr drift fix --dry-run

# Apply remediation
driftmgr drift fix --apply

# Fix only critical issues
driftmgr drift fix --critical-only

# Fix specific drift ID
driftmgr drift fix --id drift-123

# Cost optimization mode
driftmgr drift fix --cost-optimize

# Generate Terraform code
driftmgr drift fix --terraform-output
```

### `driftmgr drift auto-remediate`
Manage automatic drift remediation.
```bash
# Enable auto-remediation
driftmgr drift auto-remediate enable --dry-run

# Disable auto-remediation
driftmgr drift auto-remediate disable

# Check status
driftmgr drift auto-remediate status

# Test with simulated drift
driftmgr drift auto-remediate test --resource test-123
```

## State Management Commands

### `driftmgr state inspect`
Display and analyze Terraform state file contents.
```bash
# Inspect state file
driftmgr state inspect terraform.tfstate

# Filter by resource type
driftmgr state inspect --type aws_instance

# Show only resource names
driftmgr state inspect --names-only

# Output as JSON
driftmgr state inspect --format json
```

### `driftmgr state visualize`
Generate visual diagrams from Terraform state.
```bash
# Generate HTML visualization
driftmgr state visualize --state terraform.tfstate

# Generate SVG diagram
driftmgr state visualize --format svg

# Generate ASCII art
driftmgr state visualize --format ascii

# Generate Mermaid diagram
driftmgr state visualize --format mermaid

# Generate DOT for Graphviz
driftmgr state visualize --format dot

# With Terravision integration
driftmgr state visualize --terravision
```

### `driftmgr state scan`
Scan directories for Terraform backend configurations.
```bash
# Scan current directory
driftmgr state scan --path .

# Scan specific directory
driftmgr state scan --path ./infrastructure

# Include all workspaces
driftmgr state scan --all-workspaces

# Output as JSON
driftmgr state scan --format json
```

### `driftmgr state list`
List and analyze Terraform state files (formerly tfstate).
```bash
# List state files
driftmgr state list

# Analyze specific state file
driftmgr state list --file terraform.tfstate

# Show resource counts
driftmgr state list --count

# Show state metadata
driftmgr state list --metadata
```

## Server Commands

### `driftmgr serve web`
Start the web dashboard for visual monitoring.
```bash
# Start on default port 8080
driftmgr serve web

# Start on custom port
driftmgr serve web --port 9090

# With authentication
driftmgr serve web --auth enabled

# With HTTPS
driftmgr serve web --tls --cert cert.pem --key key.pem
```

### `driftmgr serve api`
Start the REST API server.
```bash
# Start API server
driftmgr serve api --port 8081

# With authentication
driftmgr serve api --auth jwt

# With rate limiting
driftmgr serve api --rate-limit 100

# With CORS enabled
driftmgr serve api --cors "*"
```

## Resource Management Commands

### `driftmgr delete`
Delete cloud resources (with safety checks).
```bash
# Delete specific resource
driftmgr delete --resource-id i-1234567890

# Dry run mode
driftmgr delete --dry-run --resource-id sg-abc123

# Force delete (skip confirmations)
driftmgr delete --force --resource-id vol-xyz789

# Delete multiple resources
driftmgr delete --from-file resources.txt
```

### `driftmgr export`
Export discovery results in various formats.
```bash
# Export as JSON
driftmgr export --format json

# Export as CSV
driftmgr export --format csv

# Export as Terraform
driftmgr export --format terraform

# Export specific provider
driftmgr export --provider aws --format json

# Export to file
driftmgr export --output resources.json
```

### `driftmgr import`
Import existing cloud resources into Terraform.
```bash
# Import from discovery
driftmgr import --from-discovery

# Import specific resource
driftmgr import --resource-type aws_instance --resource-id i-123456

# Generate import commands
driftmgr import --generate-only

# Import from CSV
driftmgr import --from-csv resources.csv
```

### `driftmgr accounts`
List all accessible cloud accounts/subscriptions.
```bash
# List all accounts
driftmgr accounts

# List AWS accounts
driftmgr accounts --provider aws

# List Azure subscriptions
driftmgr accounts --provider azure

# Include metadata
driftmgr accounts --detailed
```

## Verification Commands

### `driftmgr verify`
Verify discovery accuracy and resource counts.
```bash
# Basic verification
driftmgr verify --provider aws

# Enhanced verification with ML
driftmgr verify --enhanced

# Validate against cloud APIs
driftmgr verify --validate

# Specific resource type
driftmgr verify --type EC2Instance

# Output detailed report
driftmgr verify --detailed
```

## Compliance Commands

### `driftmgr compliance report`
Generate compliance reports for audits.
```bash
# SOC2 compliance report
driftmgr compliance report --standard SOC2

# HIPAA compliance report
driftmgr compliance report --standard HIPAA

# PCI-DSS compliance report
driftmgr compliance report --standard PCI-DSS

# Custom compliance rules
driftmgr compliance report --rules custom-rules.yaml

# Export as PDF
driftmgr compliance report --format pdf --output audit-report.pdf
```

## Global Flags

These flags can be used with most commands:

| Flag | Description | Default |
|------|-------------|---------|
| `--help, -h` | Show help for command | - |
| `--version, -v` | Show version information | - |
| `--config` | Config file path | `./driftmgr.yaml` |
| `--log-level` | Log level (debug/info/warn/error) | `info` |
| `--format` | Output format (json/yaml/table/summary) | `summary` |
| `--output, -o` | Output to file instead of stdout | - |
| `--no-color` | Disable colored output | `false` |
| `--quiet, -q` | Suppress non-essential output | `false` |
| `--verbose, -v` | Show detailed output | `false` |

## Environment-Specific Flags

| Flag | Description | Options |
|------|-------------|---------|
| `--environment` | Environment context | `production`, `staging`, `development` |
| `--smart-defaults` | Enable smart filtering | `true/false` |
| `--all-accounts` | Include all accessible accounts | `true/false` |
| `--parallelism` | Number of concurrent operations | `1-100` |
| `--timeout` | Operation timeout | `30s`, `5m`, `1h` |

## Provider-Specific Flags

| Flag | Description | Providers |
|------|-------------|-----------|
| `--provider` | Cloud provider | `aws`, `azure`, `gcp`, `digitalocean` |
| `--region` | Cloud region | Provider-specific |
| `--profile` | AWS profile name | AWS only |
| `--subscription` | Azure subscription ID | Azure only |
| `--project` | GCP project ID | GCP only |

## Deprecated Commands

These commands still work but show deprecation warnings:

| Old Command | New Command |
|-------------|-------------|
| `driftmgr scan` | `driftmgr state scan` |
| `driftmgr tfstate` | `driftmgr state list` |
| `driftmgr credentials` | `driftmgr discover --credentials` |
| `driftmgr dashboard` | `driftmgr serve web` |
| `driftmgr server` | `driftmgr serve api` |
| `driftmgr validate` | `driftmgr verify --validate` |
| `driftmgr auto-remediation` | `driftmgr drift auto-remediate` |
| `driftmgr delete-resource` | `driftmgr delete` |

## Quick Examples

### Daily Operations
```bash
# Morning status check
driftmgr status

# Discover all resources
driftmgr discover --auto --all-accounts

# Detect drift with smart defaults
driftmgr drift detect --smart-defaults

# Generate remediation plan
driftmgr drift fix --dry-run
```

### Security Audit
```bash
# Security-focused drift detection
driftmgr drift detect --security-only

# Generate compliance report
driftmgr compliance report --standard SOC2

# Export for auditor
driftmgr export audit-report --format pdf
```

### Cost Optimization
```bash
# Detect cost drift
driftmgr drift detect --cost-impact

# Fix cost issues
driftmgr drift fix --cost-optimize

# Generate cost report
driftmgr drift report --format csv --cost-details
```

### CI/CD Integration
```bash
# Check for drift in CI
driftmgr drift detect --provider aws --format json --output drift.json

# Fail if critical drift found
driftmgr drift detect --critical-only --fail-on-drift

# Generate remediation for review
driftmgr drift fix --dry-run --terraform-output > remediation.tf
```