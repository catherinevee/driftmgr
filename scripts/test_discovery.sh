#!/bin/bash

# Test script to diagnose driftmgr discovery issues
# This script tests individual AWS services to identify which ones are working

set -e

echo "=== DriftMgr Discovery Diagnostic Script ==="
echo "Testing AWS CLI and individual services..."
echo

# Test AWS CLI availability
echo "1. Testing AWS CLI availability..."
if command -v aws &> /dev/null; then
    echo "✓ AWS CLI is available"
    aws --version
else
    echo "✗ AWS CLI is not available"
    exit 1
fi
echo

# Test AWS credentials
echo "2. Testing AWS credentials..."
if aws sts get-caller-identity &> /dev/null; then
    echo "✓ AWS credentials are working"
    aws sts get-caller-identity
else
    echo "✗ AWS credentials are not working"
    echo "Error: $(aws sts get-caller-identity 2>&1)"
    exit 1
fi
echo

# Test individual AWS services
echo "3. Testing individual AWS services..."

# Get current region
REGION=$(aws configure get region || echo "us-east-1")
echo "Using region: $REGION"
echo

# Test EC2
echo "Testing EC2..."
if aws ec2 describe-instances --region $REGION --query 'Reservations[*].Instances[*].[InstanceId]' --output table 2>/dev/null; then
    echo "✓ EC2 discovery working"
else
    echo "✗ EC2 discovery failed"
fi
echo

# Test RDS
echo "Testing RDS..."
if aws rds describe-db-instances --region $REGION --query 'DBInstances[*].[DBInstanceIdentifier]' --output table 2>/dev/null; then
    echo "✓ RDS discovery working"
else
    echo "✗ RDS discovery failed"
fi
echo

# Test S3
echo "Testing S3..."
if aws s3 ls 2>/dev/null; then
    echo "✓ S3 discovery working"
else
    echo "✗ S3 discovery failed"
fi
echo

# Test Lambda
echo "Testing Lambda..."
if aws lambda list-functions --region $REGION --query 'Functions[*].[FunctionName]' --output table 2>/dev/null; then
    echo "✓ Lambda discovery working"
else
    echo "✗ Lambda discovery failed"
fi
echo

# Test IAM
echo "Testing IAM..."
if aws iam list-users --query 'Users[*].[UserName]' --output table 2>/dev/null; then
    echo "✓ IAM discovery working"
else
    echo "✗ IAM discovery failed"
fi
echo

# Test VPC
echo "Testing VPC..."
if aws ec2 describe-vpcs --region $REGION --query 'Vpcs[*].[VpcId]' --output table 2>/dev/null; then
    echo "✓ VPC discovery working"
else
    echo "✗ VPC discovery failed"
fi
echo

# Test CloudFormation
echo "Testing CloudFormation..."
if aws cloudformation list-stacks --region $REGION --query 'StackSummaries[*].[StackName]' --output table 2>/dev/null; then
    echo "✓ CloudFormation discovery working"
else
    echo "✗ CloudFormation discovery failed"
fi
echo

# Test ElastiCache
echo "Testing ElastiCache..."
if aws elasticache describe-cache-clusters --region $REGION --query 'CacheClusters[*].[CacheClusterId]' --output table 2>/dev/null; then
    echo "✓ ElastiCache discovery working"
else
    echo "✗ ElastiCache discovery failed"
fi
echo

# Test ECS
echo "Testing ECS..."
if aws ecs list-clusters --region $REGION --query 'clusterArns' --output table 2>/dev/null; then
    echo "✓ ECS discovery working"
else
    echo "✗ ECS discovery failed"
fi
echo

# Test EKS
echo "Testing EKS..."
if aws eks list-clusters --region $REGION --query 'clusters' --output table 2>/dev/null; then
    echo "✓ EKS discovery working"
else
    echo "✗ EKS discovery failed"
fi
echo

# Test SQS
echo "Testing SQS..."
if aws sqs list-queues --region $REGION --query 'QueueUrls' --output table 2>/dev/null; then
    echo "✓ SQS discovery working"
else
    echo "✗ SQS discovery failed"
fi
echo

# Test SNS
echo "Testing SNS..."
if aws sns list-topics --region $REGION --query 'Topics[*].[TopicArn]' --output table 2>/dev/null; then
    echo "✓ SNS discovery working"
else
    echo "✗ SNS discovery failed"
fi
echo

# Test DynamoDB
echo "Testing DynamoDB..."
if aws dynamodb list-tables --region $REGION --query 'TableNames' --output table 2>/dev/null; then
    echo "✓ DynamoDB discovery working"
else
    echo "✗ DynamoDB discovery failed"
fi
echo

echo "=== Diagnostic Complete ==="
echo "If some services show as failed, check your IAM permissions."
echo "The driftmgr discovery should work for services that show as working above."
