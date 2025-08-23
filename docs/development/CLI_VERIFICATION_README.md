# DriftMgr CLI Verification Feature

## Overview

The CLI Verification feature in DriftMgr provides automatic double-checking of discovered cloud resources using official cloud provider CLI tools. This ensures data accuracy by comparing DriftMgr's discovery results with direct CLI queries.

## Features

- **Automatic Verification**: Seamlessly integrated into the discovery process
- **Multi-Provider Support**: AWS CLI, Azure CLI, GCP CLI, and DigitalOcean CLI
- **Complete Coverage**: Verifies EC2 instances, S3 buckets, VPCs, IAM users/roles, Azure VMs, Storage Accounts, and more
- **Discrepancy Detection**: Identifies differences between DriftMgr and CLI results with severity levels
- **Performance Monitoring**: Tracks verification accuracy and timing
- **Configurable**: Enable/disable, set timeouts, retry limits, and verbosity

## Supported Cloud Providers and Resources

### AWS
- **EC2 Instances**: Verifies instance state, type, launch time
- **S3 Buckets**: Verifies bucket existence and properties
- **VPCs**: Verifies VPC state and CIDR blocks
- **RDS Instances**: Verifies database instances
- **Lambda Functions**: Verifies function existence
- **IAM Users**: Verifies user accounts
- **IAM Roles**: Verifies role definitions

### Azure
- **Virtual Machines**: Verifies VM existence and properties
- **Storage Accounts**: Verifies storage account existence
- **Virtual Networks**: Verifies VNET existence
- **SQL Databases**: Verifies database existence

### GCP (Planned)
- Compute instances, storage buckets, and other GCP resources

### DigitalOcean (Planned)
- Droplets, volumes, and other DigitalOcean resources

## Configuration

### Enable CLI Verification

Add the following configuration to your `driftmgr.yaml` file:

```yaml
discovery:
 cli_verification:
 enabled: true # Enable CLI verification
 timeout_seconds: 30 # Timeout for CLI commands
 max_retries: 3 # Maximum retry attempts
 verbose: false # Enable verbose logging
```

### Environment Variables

You can also configure CLI verification using environment variables:

```bash
export DRIFT_CLI_VERIFICATION_ENABLED=true
export DRIFT_CLI_VERIFICATION_TIMEOUT_SECONDS=30
export DRIFT_CLI_VERIFICATION_MAX_RETRIES=3
export DRIFT_CLI_VERIFICATION_VERBOSE=false
```

## Prerequisites

### Required CLI Tools

1. **AWS CLI** (for AWS verification)
 ```bash
 # Install AWS CLI
 pip install awscli
 # or
 curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
 unzip awscliv2.zip
 sudo ./aws/install

 # Configure AWS credentials
 aws configure
 ```

2. **Azure CLI** (for Azure verification)
 ```bash
 # Install Azure CLI
 curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash

 # Login to Azure
 az login
 ```

3. **GCP CLI** (for GCP verification - planned)
 ```bash
 # Install Google Cloud CLI
 curl https://sdk.cloud.google.com | bash
 exec -l $SHELL
 gcloud init
 ```

4. **DigitalOcean CLI** (for DigitalOcean verification - planned)
 ```bash
 # Install doctl
 snap install doctl

 # Authenticate
 doctl auth init
 ```

### Credential Configuration

Ensure your cloud provider credentials are properly configured:

```bash
# AWS
aws sts get-caller-identity

# Azure
az account show

# GCP
gcloud auth list

# DigitalOcean
doctl account get
```

## Usage

### Basic Usage

1. **Configure CLI verification** in your `driftmgr.yaml` file
2. **Run discovery** as usual - CLI verification happens automatically
3. **Review results** in the console output and logs

### Example Configuration File

```yaml
# driftmgr_cli_verification.yaml
discovery:
 cli_verification:
 enabled: true
 timeout_seconds: 30
 max_retries: 3
 verbose: false

 regions:
 - "us-east-1"
 - "us-west-2"
 - "eu-west-1"
```

### Running with CLI Verification

```bash
# Run discovery with CLI verification enabled
go run test_cli_verification.go

# Or use the main DriftMgr binary
./driftmgr discover --config driftmgr_cli_verification.yaml
```

## Output and Results

### Console Output

```
 DriftMgr CLI Verification Test
==================================
Testing CLI verification with providers: [aws azure]
Testing regions: [us-east-1 us-west-2]
CLI verification enabled: true
CLI verification timeout: 30 seconds
CLI verification max retries: 3
CLI verification verbose: false

 Starting enhanced discovery with CLI verification...
[OK] Discovery completed in 2m15s
 Total resources discovered: 45

 CLI Verification Results:
============================
Total resources verified: 45
Resources found in CLI: 43
Resources not found in CLI: 2
Total discrepancies: 3
Accuracy rate: 95.56%

 Detailed Verification Results:
=================================

Resource: my-ec2-instance (aws_ec2_instance)
Provider: aws, Region: us-east-1
CLI Found: true
 - instance_type: DriftMgr=t3.micro, CLI=t3.small (severity: critical)
 - state: DriftMgr=running, CLI=stopped (severity: warning)

 Discrepancy Summary:
=======================
Critical discrepancies: 1
Warning discrepancies: 1
Info discrepancies: 1
```

### JSON Results File

A detailed JSON file is generated with complete verification results:

```json
{
 "timestamp": "2025-01-17T10:30:00Z",
 "summary": {
 "total_resources": 45,
 "cli_found": 43,
 "cli_not_found": 2,
 "discrepancies": 3,
 "accuracy_rate": 95.56
 },
 "results": [
 {
 "resource_id": "i-1234567890abcdef0",
 "resource_name": "my-ec2-instance",
 "resource_type": "aws_ec2_instance",
 "provider": "aws",
 "region": "us-east-1",
 "cli_found": true,
 "cli_resource": {
 "InstanceId": "i-1234567890abcdef0",
 "InstanceType": "t3.small",
 "State": {"Name": "stopped"}
 },
 "discrepancies": [
 {
 "field": "instance_type",
 "driftmgr_value": "t3.micro",
 "cli_value": "t3.small",
 "severity": "critical"
 }
 ],
 "verification_time": "2025-01-17T10:30:00Z"
 }
 ],
 "discovery_duration": "2m15s",
 "total_resources_discovered": 45
}
```

## Discrepancy Severity Levels

### Critical
- Resource type mismatches
- Instance type differences
- CIDR block changes
- Fundamental resource properties

### Warning
- State differences (running vs stopped)
- Tag differences
- Non-critical property changes

### Info
- Timestamp differences
- Metadata variations
- Minor attribute changes

## Performance Considerations

### Time Impact
- **CLI verification adds 10-30%** to discovery time
- **Batch processing** minimizes CLI command overhead
- **Parallel execution** for multiple resource types
- **Caching** reduces redundant CLI calls

### Resource Usage
- **Memory**: Minimal additional memory usage
- **CPU**: Low impact, mainly for JSON parsing
- **Network**: Additional API calls to cloud providers

### Optimization Tips
1. **Limit regions** for faster verification
2. **Use specific resource types** instead of all resources
3. **Adjust timeouts** based on your environment
4. **Enable caching** to reduce CLI calls

## Troubleshooting

### Common Issues

1. **CLI tools not found**
 ```
 Error: AWS CLI command failed: exec: "aws": executable file not found in $PATH
 ```
 **Solution**: Install and configure the required CLI tools

2. **Authentication errors**
 ```
 Error: AWS CLI command failed: Unable to locate credentials
 ```
 **Solution**: Configure cloud provider credentials

3. **Timeout errors**
 ```
 Error: AWS CLI command failed: context deadline exceeded
 ```
 **Solution**: Increase timeout_seconds in configuration

4. **Permission errors**
 ```
 Error: AWS CLI command failed: AccessDenied
 ```
 **Solution**: Ensure CLI credentials have sufficient permissions

### Debug Mode

Enable verbose logging for detailed troubleshooting:

```yaml
discovery:
 cli_verification:
 enabled: true
 verbose: true
```

### Log Analysis

Check logs for verification details:

```bash
# Look for CLI verification logs
grep "CLI verification" driftmgr.log

# Check for discrepancies
grep "discrepancy" driftmgr.log

# Monitor performance
grep "verification.*accuracy" driftmgr.log
```

## Best Practices

### Configuration
1. **Start with small regions** for testing
2. **Use appropriate timeouts** for your environment
3. **Enable verbose logging** during initial setup
4. **Monitor performance** and adjust settings

### Security
1. **Use least-privilege credentials** for CLI tools
2. **Rotate credentials regularly**
3. **Monitor CLI command execution**
4. **Review verification logs** for sensitive data

### Performance
1. **Batch verification** by resource type
2. **Cache results** when possible
3. **Parallel execution** for multiple regions
4. **Optimize timeouts** based on network conditions

## Integration with Existing Workflows

### CI/CD Integration

Add CLI verification to your CI/CD pipeline:

```yaml
# .github/workflows/driftmgr-verification.yml
name: DriftMgr CLI Verification
on: [push, pull_request]

jobs:
 verify:
 runs-on: ubuntu-latest
 steps:
 - uses: actions/checkout@v2
 - name: Install AWS CLI
 run: |
 curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
 unzip awscliv2.zip
 sudo ./aws/install
 - name: Configure AWS credentials
 uses: aws-actions/configure-aws-credentials@v1
 with:
 aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
 aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
 aws-region: us-east-1
 - name: Run DriftMgr with CLI verification
 run: |
 go run test_cli_verification.go
 - name: Upload verification results
 uses: actions/upload-artifact@v2
 with:
 name: cli-verification-results
 path: cli_verification_results.json
```

### Monitoring and Alerting

Set up monitoring for verification accuracy:

```bash
# Check verification accuracy
accuracy=$(jq -r '.summary.accuracy_rate' cli_verification_results.json)

if (( $(echo "$accuracy < 95" | bc -l) )); then
 echo "Warning: CLI verification accuracy below 95%: ${accuracy}%"
 # Send alert
fi
```

## Future Enhancements

### Planned Features
1. **GCP CLI verification** support
2. **DigitalOcean CLI verification** support
3. **Real-time verification** during discovery
4. **Custom verification rules** configuration
5. **Verification result caching**
6. **Integration with drift detection**

### Contributing

To contribute to CLI verification:

1. **Add new resource types** in `cli_verification.go`
2. **Implement provider-specific verification** methods
3. **Add tests** for new verification logic
4. **Update documentation** for new features

## Support

For issues and questions:

1. **Check troubleshooting section** above
2. **Review logs** for detailed error messages
3. **Verify CLI tool installation** and configuration
4. **Test with minimal configuration** first
5. **Open an issue** on the DriftMgr repository

---

**Note**: CLI verification is designed to enhance data accuracy but should not replace proper monitoring and alerting systems. Always verify critical infrastructure changes through multiple methods.
