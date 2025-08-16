# DriftMgr Documentation

Welcome to the DriftMgr documentation. This guide provides comprehensive information about using DriftMgr for Terraform drift detection and remediation.

## Quick Start

- [Installation Guide](installation.md) - How to install DriftMgr
- [Getting Started](getting-started.md) - Your first steps with DriftMgr
- [Basic Usage](basic-usage.md) - Core commands and workflows

## Core Features

### Discovery & Analysis
- [Resource Discovery](discovery.md) - Discover cloud resources across providers
- [Drift Analysis](analysis.md) - Analyze infrastructure drift
- [Perspective Analysis](perspective.md) - Compare state with live infrastructure

### Remediation
- [Automated Remediation](remediation.md) - Generate and execute remediation commands
- [Batch Remediation](batch-remediation.md) - Handle multiple drifts at once
- [Rollback Operations](rollback.md) - Rollback to previous states

### Visualization
- [Infrastructure Diagrams](diagrams.md) - Generate infrastructure visualizations
- [Export Options](export.md) - Export diagrams in various formats

## Configuration

### Timeout Configuration
- [Timeout Configuration Guide](../TIMEOUT_CONFIGURATION.md) - Configure timeouts for discovery operations
- [Timeout Fix Summary](../TIMEOUT_FIX_SUMMARY.md) - Complete fix documentation
- [Timeout Verification](../verify_timeout_fix.md) - Verification report

### Environment Setup
- [Environment Variables](environment.md) - All available environment variables
- [Provider Configuration](providers.md) - Configure cloud providers
- [Security Configuration](security.md) - Security settings and best practices

## Advanced Topics

### Multi-Cloud Support
- [AWS Integration](aws-integration.md) - AWS-specific features and configuration
- [Azure Integration](azure-integration.md) - Azure-specific features and configuration
- [GCP Integration](gcp-integration.md) - GCP-specific features and configuration
- [DigitalOcean Integration](digitalocean-integration.md) - DigitalOcean-specific features

### Development & Customization
- [API Reference](api-reference.md) - REST API documentation
- [Plugin Development](plugins.md) - Creating custom plugins
- [Contributing](contributing.md) - How to contribute to DriftMgr

## Troubleshooting

### Common Issues
- [Timeout Issues](troubleshooting/timeouts.md) - Resolving timeout problems
- [Connection Issues](troubleshooting/connections.md) - Network and connectivity problems
- [Authentication Issues](troubleshooting/authentication.md) - Provider authentication problems

### Performance
- [Performance Tuning](performance.md) - Optimize DriftMgr performance
- [Large Infrastructure](large-infrastructure.md) - Handling large-scale deployments

## Reference

- [Command Reference](commands.md) - Complete command reference
- [Configuration Reference](config-reference.md) - All configuration options
- [Error Codes](error-codes.md) - Error code reference
- [Changelog](../CHANGELOG.md) - Version history and changes

## Examples

- [Basic Examples](examples/basic.md) - Simple usage examples
- [Advanced Examples](examples/advanced.md) - Complex scenarios
- [Real-World Workflows](examples/workflows.md) - Production workflows

## Support

- [FAQ](faq.md) - Frequently asked questions
- [Community](community.md) - Get help from the community
- [Reporting Issues](reporting-issues.md) - How to report bugs and request features

---

## Quick Reference

### Essential Commands
```bash
# Discover resources
driftmgr discover aws all

# Analyze drift
driftmgr analyze terraform

# Generate remediation
driftmgr remediate drift_123 --generate

# Configure timeouts for large infrastructure
export DRIFT_DISCOVERY_TIMEOUT=10m
export DRIFT_CLIENT_TIMEOUT=5m
```

### Configuration Scripts
```bash
# Windows PowerShell
.\scripts\set-timeout.ps1 -Scenario large

# Linux/macOS
./scripts/set-timeout.sh -s large
```

### Environment Variables
- `DRIFT_CLIENT_TIMEOUT` - General client timeout (default: 30s)
- `DRIFT_DISCOVERY_TIMEOUT` - Discovery-specific timeout (default: 2m/5m)
- `AWS_ACCESS_KEY_ID` - AWS access key
- `AZURE_SUBSCRIPTION_ID` - Azure subscription ID
- `GOOGLE_APPLICATION_CREDENTIALS` - GCP credentials path
