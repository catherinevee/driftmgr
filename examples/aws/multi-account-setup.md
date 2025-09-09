# AWS Multi-Account Drift Detection Setup

This guide demonstrates how to configure DriftMgr for detecting drift across multiple AWS accounts using AssumeRole.

## Prerequisites

- DriftMgr installed (`driftmgr` command available)
- AWS CLI configured with credentials
- Cross-account IAM roles configured

## Architecture

```
┌─────────────────────┐
│   Management        │
│   Account           │
│  (123456789012)     │
│                     │
│ ┌─────────────────┐ │
│ │   DriftMgr      │ │
│ │   Execution     │ │
│ └────────┬────────┘ │
└──────────┼──────────┘
           │
    AssumeRole
           │
    ┌──────┴──────┬──────────┬──────────┐
    │             │          │          │
┌───▼────┐  ┌────▼───┐  ┌───▼────┐ ┌───▼────┐
│ Dev    │  │ Stage  │  │ Prod   │ │ Audit  │
│ Account│  │ Account│  │ Account│ │ Account│
└────────┘  └────────┘  └────────┘ └────────┘
```

## Step 1: IAM Role Setup

### Create Cross-Account Role (in each target account)

Create `DriftMgrRole` in each target account:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::123456789012:root"
      },
      "Action": "sts:AssumeRole",
      "Condition": {
        "StringEquals": {
          "sts:ExternalId": "DriftMgr-UniqueExternalId"
        }
      }
    }
  ]
}
```

### Attach ReadOnly Policy

```bash
aws iam attach-role-policy \
  --role-name DriftMgrRole \
  --policy-arn arn:aws:iam::aws:policy/ReadOnlyAccess
```

## Step 2: DriftMgr Configuration

Create `driftmgr-config.yaml`:

```yaml
# Multi-Account Configuration for DriftMgr
version: "1.0"

# Default settings
defaults:
  provider: aws
  output_format: json
  parallel_workers: 4
  timeout: 30m

# Account configurations
accounts:
  - name: development
    account_id: "111111111111"
    role_arn: "arn:aws:iam::111111111111:role/DriftMgrRole"
    external_id: "DriftMgr-UniqueExternalId"
    regions:
      - us-east-1
      - us-west-2
    state_backend:
      type: s3
      bucket: dev-terraform-state
      key: infrastructure/terraform.tfstate
      region: us-east-1

  - name: staging
    account_id: "222222222222"
    role_arn: "arn:aws:iam::222222222222:role/DriftMgrRole"
    external_id: "DriftMgr-UniqueExternalId"
    regions:
      - us-east-1
      - eu-west-1
    state_backend:
      type: s3
      bucket: stage-terraform-state
      key: infrastructure/terraform.tfstate
      region: us-east-1

  - name: production
    account_id: "333333333333"
    role_arn: "arn:aws:iam::333333333333:role/DriftMgrRole"
    external_id: "DriftMgr-UniqueExternalId"
    regions:
      - us-east-1
      - us-west-2
      - eu-west-1
      - ap-southeast-1
    state_backend:
      type: s3
      bucket: prod-terraform-state
      key: infrastructure/terraform.tfstate
      region: us-east-1
      role_arn: "arn:aws:iam::333333333333:role/TerraformStateRole"

  - name: audit
    account_id: "444444444444"
    role_arn: "arn:aws:iam::444444444444:role/DriftMgrRole"
    external_id: "DriftMgr-UniqueExternalId"
    regions:
      - us-east-1
    state_backend:
      type: s3
      bucket: audit-terraform-state
      key: compliance/terraform.tfstate
      region: us-east-1

# Notification settings
notifications:
  slack:
    webhook_url: ${SLACK_WEBHOOK_URL}
    channel: "#infrastructure-drift"
    username: "DriftMgr"
    icon_emoji: ":rotating_light:"
    
  email:
    smtp_host: smtp.gmail.com
    smtp_port: 587
    from: drift-alerts@example.com
    to:
      - devops-team@example.com
      - security-team@example.com
    
  pagerduty:
    integration_key: ${PAGERDUTY_KEY}
    severity_threshold: critical

# Monitoring settings
monitoring:
  enabled: true
  interval: 30m
  concurrent_accounts: 2
  
  # Define what constitutes critical drift
  thresholds:
    critical:
      drifted_resources: 10
      missing_resources: 5
      security_groups_changed: 1
      iam_changes: 1
    
    warning:
      drifted_resources: 5
      missing_resources: 2
      unmanaged_resources: 20

# Remediation settings
remediation:
  auto_generate_import: true
  require_approval: true
  approval_webhook: ${APPROVAL_WEBHOOK_URL}
  
  # Automatic remediation for specific resource types
  auto_remediate:
    - resource_type: aws_s3_bucket_public_access_block
      action: enforce_terraform
    - resource_type: aws_security_group_rule
      action: alert_only
      
# Exclusions
exclusions:
  # Resources to ignore during drift detection
  resource_types:
    - aws_autoscaling_group  # Dynamic capacity
    - aws_ecs_service        # Task counts change
    
  resource_patterns:
    - "aws_instance.bastion-*"  # Temporary bastion hosts
    - "aws_lambda_function.*-dev"  # Dev lambdas
    
  tags:
    - Key: Environment
      Value: Development
    - Key: ManagedBy
      Value: AutoScaling
```

## Step 3: Running Multi-Account Drift Detection

### Detect Drift Across All Accounts

```bash
#!/bin/bash
# multi-account-drift.sh

# Load configuration
CONFIG_FILE="driftmgr-config.yaml"

# Function to check single account
check_account() {
    local account_name=$1
    local account_id=$2
    local role_arn=$3
    local regions=$4
    
    echo "========================================="
    echo "Checking account: $account_name ($account_id)"
    echo "========================================="
    
    for region in $regions; do
        echo "  Region: $region"
        
        driftmgr drift detect \
            --config "$CONFIG_FILE" \
            --account "$account_name" \
            --region "$region" \
            --assume-role "$role_arn" \
            --output json \
            --save-report "reports/drift-${account_name}-${region}-$(date +%Y%m%d).json"
    done
    
    echo ""
}

# Create reports directory
mkdir -p reports

# Parse accounts from config and check each
accounts=$(yq eval '.accounts[].name' "$CONFIG_FILE")

for account in $accounts; do
    account_id=$(yq eval ".accounts[] | select(.name == \"$account\") | .account_id" "$CONFIG_FILE")
    role_arn=$(yq eval ".accounts[] | select(.name == \"$account\") | .role_arn" "$CONFIG_FILE")
    regions=$(yq eval ".accounts[] | select(.name == \"$account\") | .regions[]" "$CONFIG_FILE" | tr '\n' ' ')
    
    check_account "$account" "$account_id" "$role_arn" "$regions"
done

# Generate consolidated report
echo "Generating consolidated report..."
driftmgr report consolidate \
    --input-dir reports \
    --output consolidated-drift-report.html \
    --format html

echo "Multi-account drift detection complete!"
echo "  Reports available in: ./reports/"
echo "  Consolidated report: ./consolidated-drift-report.html"
```

## Step 4: Continuous Monitoring

### Kubernetes CronJob for Continuous Monitoring

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: driftmgr-multi-account
  namespace: infrastructure
spec:
  schedule: "0 */6 * * *"  # Every 6 hours
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: driftmgr
          containers:
          - name: driftmgr
            image: catherinevee/driftmgr:latest
            env:
            - name: AWS_REGION
              value: us-east-1
            - name: SLACK_WEBHOOK_URL
              valueFrom:
                secretKeyRef:
                  name: driftmgr-secrets
                  key: slack-webhook
            volumeMounts:
            - name: config
              mountPath: /config
            - name: reports
              mountPath: /reports
            command:
            - /bin/sh
            - -c
            - |
              # Run multi-account drift detection
              driftmgr drift detect-multi \
                --config /config/driftmgr-config.yaml \
                --output-dir /reports \
                --notify-slack
                
              # Upload reports to S3
              aws s3 sync /reports s3://drift-reports/$(date +%Y/%m/%d)/
          
          volumes:
          - name: config
            configMap:
              name: driftmgr-config
          - name: reports
            emptyDir: {}
          
          restartPolicy: OnFailure
```

## Step 5: Lambda Function for Event-Driven Detection

```python
# lambda_drift_detector.py
import boto3
import json
import os
import subprocess
from datetime import datetime

def lambda_handler(event, context):
    """
    Triggered by CloudTrail events to detect drift after changes
    """
    
    # Parse CloudTrail event
    account_id = event['account']
    region = event['region']
    resource_type = event['detail']['resourceType']
    
    # Skip if resource type is excluded
    excluded_types = os.environ.get('EXCLUDED_TYPES', '').split(',')
    if resource_type in excluded_types:
        return {
            'statusCode': 200,
            'body': json.dumps('Resource type excluded from drift detection')
        }
    
    # Run DriftMgr
    result = subprocess.run([
        '/opt/driftmgr',
        'drift', 'detect',
        '--account-id', account_id,
        '--region', region,
        '--resource-type', resource_type,
        '--output', 'json'
    ], capture_output=True, text=True)
    
    drift_data = json.loads(result.stdout)
    
    # Send notification if drift detected
    if drift_data.get('drift_detected', False):
        send_notification(account_id, region, drift_data)
    
    # Store results in DynamoDB
    store_results(account_id, region, drift_data)
    
    return {
        'statusCode': 200,
        'body': json.dumps(drift_data)
    }

def send_notification(account_id, region, drift_data):
    """Send drift notification to SNS"""
    sns = boto3.client('sns')
    
    message = {
        'account_id': account_id,
        'region': region,
        'timestamp': datetime.now().isoformat(),
        'drift_summary': drift_data.get('summary', {}),
        'affected_resources': drift_data.get('drifted_resources', [])
    }
    
    sns.publish(
        TopicArn=os.environ['SNS_TOPIC_ARN'],
        Subject=f'Drift Detected in {account_id}/{region}',
        Message=json.dumps(message, indent=2)
    )

def store_results(account_id, region, drift_data):
    """Store drift results in DynamoDB"""
    dynamodb = boto3.resource('dynamodb')
    table = dynamodb.Table('drift-detection-results')
    
    table.put_item(
        Item={
            'account_region': f'{account_id}#{region}',
            'timestamp': datetime.now().isoformat(),
            'drift_data': drift_data
        }
    )
```

## Step 6: Terraform Module for DriftMgr Setup

```hcl
# modules/driftmgr/main.tf

variable "accounts" {
  description = "List of AWS accounts to monitor"
  type = list(object({
    name       = string
    account_id = string
    regions    = list(string)
  }))
}

variable "external_id" {
  description = "External ID for role assumption"
  type        = string
  default     = "DriftMgr-UniqueExternalId"
}

# Create IAM role in each account
resource "aws_iam_role" "driftmgr" {
  for_each = { for acc in var.accounts : acc.name => acc }
  
  name = "DriftMgrRole"
  
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          AWS = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"
        }
        Action = "sts:AssumeRole"
        Condition = {
          StringEquals = {
            "sts:ExternalId" = var.external_id
          }
        }
      }
    ]
  })
}

# Attach ReadOnly policy
resource "aws_iam_role_policy_attachment" "driftmgr_readonly" {
  for_each = aws_iam_role.driftmgr
  
  role       = each.value.name
  policy_arn = "arn:aws:iam::aws:policy/ReadOnlyAccess"
}

# S3 bucket for drift reports
resource "aws_s3_bucket" "drift_reports" {
  bucket = "drift-reports-${data.aws_caller_identity.current.account_id}"
}

resource "aws_s3_bucket_versioning" "drift_reports" {
  bucket = aws_s3_bucket.drift_reports.id
  
  versioning_configuration {
    status = "Enabled"
  }
}

# DynamoDB table for drift history
resource "aws_dynamodb_table" "drift_history" {
  name           = "drift-detection-results"
  billing_mode   = "PAY_PER_REQUEST"
  hash_key       = "account_region"
  range_key      = "timestamp"
  
  attribute {
    name = "account_region"
    type = "S"
  }
  
  attribute {
    name = "timestamp"
    type = "S"
  }
  
  ttl {
    enabled        = true
    attribute_name = "ttl"
  }
}

# SNS topic for notifications
resource "aws_sns_topic" "drift_alerts" {
  name = "driftmgr-alerts"
}

resource "aws_sns_topic_subscription" "drift_alerts_email" {
  topic_arn = aws_sns_topic.drift_alerts.arn
  protocol  = "email"
  endpoint  = "devops-team@example.com"
}

# CloudWatch Dashboard
resource "aws_cloudwatch_dashboard" "drift" {
  dashboard_name = "DriftMgr"
  
  dashboard_body = jsonencode({
    widgets = [
      {
        type = "metric"
        properties = {
          metrics = [
            ["DriftMgr", "DriftedResources", { stat = "Sum" }],
            [".", "MissingResources", { stat = "Sum" }],
            [".", "UnmanagedResources", { stat = "Sum" }]
          ]
          period = 300
          stat   = "Average"
          region = "us-east-1"
          title  = "Drift Detection Metrics"
        }
      }
    ]
  })
}

output "role_arns" {
  value = { for k, v in aws_iam_role.driftmgr : k => v.arn }
}

output "reports_bucket" {
  value = aws_s3_bucket.drift_reports.id
}

output "sns_topic_arn" {
  value = aws_sns_topic.drift_alerts.arn
}
```

## Best Practices

1. **Use External IDs**: Always use external IDs for role assumption to prevent confused deputy attacks
2. **Least Privilege**: Grant only read permissions to DriftMgr roles
3. **Separate State Access**: Use different roles for state file access if needed
4. **Rate Limiting**: Be aware of AWS API rate limits when scanning multiple accounts
5. **Cost Optimization**: Use filters to scan only relevant resources
6. **Audit Logging**: Enable CloudTrail for all DriftMgr operations

## Troubleshooting

### Common Issues

1. **AccessDenied when assuming role**
   - Check trust relationship in target account
   - Verify external ID matches
   - Ensure source account has permission to assume

2. **State file access issues**
   - Verify S3 bucket permissions
   - Check if role has access to KMS key (if encrypted)
   - Ensure correct region specified

3. **Timeout errors**
   - Increase timeout in configuration
   - Reduce number of parallel workers
   - Filter resources to reduce scope

## Next Steps

- Set up automated remediation workflows
- Integrate with your ticketing system
- Create custom OPA policies for compliance
- Build dashboards for drift trends

For more information, see the [full documentation](https://github.com/catherinevee/driftmgr/docs).