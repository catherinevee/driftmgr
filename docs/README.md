# DriftMgr Documentation

Welcome to the DriftMgr documentation! This guide covers installation, configuration, and usage of DriftMgr for cloud infrastructure drift detection and management.

## Quick Links

- [Installation Guide](./INSTALLATION_GUIDE.md) - Install DriftMgr on your system
- [Command Reference](./COMMAND_REFERENCE.md) - Complete command reference
- [Enterprise Features](./ENTERPRISE_FEATURES.md) - Enterprise-grade capabilities
- [Secrets Setup](./SECRETS_SETUP.md) - Configure GitHub Actions secrets

## Available Documentation

### Getting Started
- [Installation Guide](./INSTALLATION_GUIDE.md) - Build and install DriftMgr
- [Command Reference](./COMMAND_REFERENCE.md) - All available commands
- [Testing Guide](./TESTING_GUIDE.md) - Run tests and validate installation

### User Guides
- [CLI Reference](./user-guide/cli-reference.md) - CLI usage examples
- [Drift Detection Guide](./user-guide/drift-detection-guide.md) - Detect infrastructure drift
- [CLI Help Feature](./user-guide/cli-help-feature.md) - Interactive help system
- [Terraform Examples](./user-guide/terraform-examples.md) - Terraform integration

### Enterprise Features
- [Enterprise Features](./ENTERPRISE_FEATURES.md) - Audit, RBAC, Vault integration
- [AWS Multi-Account](./AWS_MULTI_ACCOUNT_SUPPORT.md) - Multi-account AWS support
- [Cost Calculation](./COST_CALCULATION.md) - Cost impact analysis

### Development
- [Contributing](./CONTRIBUTING.md) - How to contribute to DriftMgr
- [DriftMgr Commands](./development/DRIFTMGR_COMMANDS.md) - Command implementation details
- [TUI Loading Bar Guide](./development/TUI_LOADING_BAR_GUIDE.md) - Terminal UI components

### Demos
- [Drift Detection Demo](./demos/DRIFT_DETECTION_DEMO.md) - Live drift detection example
- [Credential Display Demo](./demos/demo_credential_display.md) - Credential management
- [TUI Demo](./demos/demo_tui.md) - Terminal UI demonstration

### CI/CD & Deployment
- [GitHub Actions Workflows](./../.github/workflows/) - Pre-configured workflows
- [CI/CD Examples](./ci-cd-examples/README.md) - Examples for various platforms
- [Docker Setup](./../.github/workflows/docker.yml) - Docker build and push

### Architecture
- [Component Interactions](./architecture/component-interactions.md) - System architecture
- [Future Proofing Strategy](./architecture/FUTURE_PROOFING_STRATEGY.md) - Extensibility design

### Features
- [Progress Indicators](./PROGRESS_INDICATORS.md) - Progress bars and animations
- [Color Support](./COLOR_SUPPORT.md) - Terminal color output
- [Testing](./TESTING.md) - Test framework and strategies

### Operations
- [Operational Runbook](./runbooks/OPERATIONAL_RUNBOOK.md) - Production operations guide

## Core Features

### Discovery & Analysis
- **Multi-Cloud Discovery** - Discover resources across AWS, Azure, GCP, DigitalOcean
- **Drift Detection** - Compare actual state with desired state
- **Smart Defaults** - Intelligent filtering to reduce noise by 75-85%
- **Cost Analysis** - Calculate financial impact of drift

### Remediation
- **Automated Remediation** - Generate and execute remediation plans
- **Dry-Run Mode** - Preview changes before applying
- **Safety Checks** - Built-in safety mechanisms
- **Rollback Support** - Undo changes if needed

### State Management
- **Terraform Integration** - Analyze and visualize .tfstate files
- **State Inspection** - Deep dive into state structure
- **Backend Detection** - Auto-discover Terraform backends

### Enterprise Features
- **Audit Logging** - Complete audit trail with compliance modes
- **RBAC** - Role-based access control
- **HashiCorp Vault** - Secure secrets management
- **Circuit Breakers** - Prevent cascading failures
- **Rate Limiting** - Control API usage

## Configuration

### Environment Variables
```bash
DRIFTMGR_CONFIG=/etc/driftmgr/config.yaml
DRIFTMGR_LOG_LEVEL=debug
DRIFTMGR_PROVIDER=aws
DRIFTMGR_REGION=us-east-1
DRIFTMGR_OUTPUT=json
```

### Configuration File
Default location: `~/.driftmgr/config.yaml`

```yaml
providers:
  aws:
    regions: ["us-east-1", "us-west-2"]
    profile: default
  azure:
    subscription_id: "xxx"
  gcp:
    project_id: "xxx"

drift:
  smart_defaults: true
  environment: production
  
audit:
  enabled: true
  path: /var/log/driftmgr/audit
  compliance_mode: SOC2
```

## Quick Start Examples

### Basic Discovery
```bash
# Auto-discover all configured providers
driftmgr discover --auto

# Discover specific provider
driftmgr discover --provider aws --region us-east-1
```

### Drift Detection
```bash
# Detect drift with smart defaults
driftmgr drift detect --provider aws

# Detect all drift (no filtering)
driftmgr drift detect --no-smart-defaults
```

### State Management
```bash
# Inspect Terraform state
driftmgr state inspect terraform.tfstate

# Scan for Terraform backends
driftmgr scan --path ./terraform
```

### Export Results
```bash
# Export to JSON
driftmgr export --format json --output resources.json

# Export to HTML report
driftmgr export --format html --output report.html
```

## Support

- **GitHub Issues**: [Report issues](https://github.com/catherinevee/driftmgr/issues)
- **Documentation**: This directory contains all documentation
- **Examples**: See [examples/](../examples/) directory for usage examples

## License

DriftMgr is open source software. See [LICENSE](../LICENSE) for details.