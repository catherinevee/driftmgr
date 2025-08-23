# DriftMgr CI/CD Integration

This directory contains CI/CD pipeline integrations for DriftMgr, enabling automated drift detection, validation, and remediation across various platforms.

## Overview

DriftMgr CI/CD integration provides:
- **Pre-deployment validation** - Catch drift before deployment
- **Post-deployment verification** - Ensure successful deployments
- **Automated remediation** - Fix drift issues automatically
- **Reporting and alerting** - Generate reports and send notifications
- **Multi-platform support** - GitHub Actions, GitLab CI, Jenkins, Azure DevOps, CircleCI

## Quick Start

### 1. Choose Your Platform
Select the appropriate integration for your CI/CD platform:
- [GitHub Actions](github-actions/)
- [GitLab CI](gitlab-ci/)
- [Jenkins](jenkins/)
- [Azure DevOps](azure-devops/)
- [CircleCI](circleci/)
- [Terraform Cloud](terraform-cloud/)

### 2. Configure Environment
Set up required environment variables:
```bash
# Cloud Provider Credentials
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AZURE_CLIENT_ID="your-client-id"
export AZURE_CLIENT_SECRET="your-client-secret"
export GOOGLE_APPLICATION_CREDENTIALS="path/to/credentials.json"

# DriftMgr Configuration
export DRIFT_CONFIG_FILE="driftmgr.yaml"
export DRIFT_OUTPUT_FORMAT="json"
export DRIFT_FAIL_ON_DRIFT="true"
```

### 3. Add to Pipeline
Copy the appropriate pipeline file to your repository and customize as needed.

## Integration Patterns

### Pre-Deployment Validation
```yaml
# Fail pipeline if drift detected
- name: Check Infrastructure Drift
  run: |
    driftmgr discover aws us-east-1
    driftmgr analyze terraform.tfstate
    # Pipeline fails if drift found
```

### Post-Deployment Verification
```yaml
# Verify deployment was successful
- name: Verify Deployment
  run: |
    driftmgr discover aws us-east-1
    driftmgr perspective terraform.tfstate aws
    # Alert if post-deployment drift found
```

### Automated Remediation
```yaml
# Auto-fix drift issues
- name: Remediate Drift
  run: |
    driftmgr remediate-batch terraform --auto
    # Only fail if remediation fails
```

### Scheduled Monitoring
```yaml
# Run on schedule, not tied to deployments
- name: Scheduled Drift Check
  run: |
    driftmgr discover aws us-east-1
    driftmgr analyze terraform.tfstate
    driftmgr notify slack "Drift Alert" "Drift detected in production"
```

## Configuration Options

### Environment Variables
| Variable | Description | Default |
|----------|-------------|---------|
| `DRIFT_CONFIG_FILE` | DriftMgr configuration file | `driftmgr.yaml` |
| `DRIFT_OUTPUT_FORMAT` | Output format (json, yaml, text) | `json` |
| `DRIFT_FAIL_ON_DRIFT` | Fail pipeline on drift detection | `true` |
| `DRIFT_AUTO_REMEDIATE` | Automatically remediate drift | `false` |
| `DRIFT_NOTIFICATION_CHANNEL` | Notification channel (slack, email, webhook) | `slack` |
| `DRIFT_SEVERITY_THRESHOLD` | Minimum severity to fail pipeline | `high` |

### DriftMgr Configuration
```yaml
# driftmgr.yaml
ci_cd:
  enabled: true
  fail_on_drift: true
  auto_remediate: false
  notification_channels:
    - slack
    - email
  severity_threshold: high
  environments:
    development:
      fail_on_drift: false
      auto_remediate: true
    staging:
      fail_on_drift: true
      auto_remediate: false
    production:
      fail_on_drift: true
      auto_remediate: false
```

## Best Practices

### 1. Environment-Specific Configurations
- Use different drift thresholds per environment
- Enable auto-remediation in development
- Require manual approval in production

### 2. Performance Optimization
- Cache discovery results between runs
- Use incremental discovery for large infrastructures
- Run discovery in parallel when possible

### 3. Security Considerations
- Use secure credential management
- Limit auto-remediation permissions
- Audit all drift detection and remediation actions

### 4. Monitoring and Alerting
- Set up complete notifications
- Monitor pipeline performance
- Track drift trends over time

## Troubleshooting

### Common Issues

#### Pipeline Hangs
- Check for interactive prompts in DriftMgr
- Ensure all required inputs are provided
- Verify network connectivity to cloud providers

#### Authentication Failures
- Verify credential environment variables
- Check credential permissions
- Ensure credentials haven't expired

#### False Positives
- Review drift detection sensitivity
- Check for temporary resource states
- Verify Terraform state file accuracy

### Debug Mode
Enable debug mode for troubleshooting:
```bash
export DRIFT_DEBUG="true"
export DRIFT_VERBOSE="true"
```

## Support

For issues with CI/CD integration:
- Check the [DriftMgr documentation](../README.md)
- Review platform-specific guides in subdirectories
- Open an issue on GitHub with CI/CD tag

## Contributing

To add support for new CI/CD platforms:
1. Create a new subdirectory for the platform
2. Add pipeline configuration files
3. Include documentation and examples
4. Update this README with platform information
