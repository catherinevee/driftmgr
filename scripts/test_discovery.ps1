# Test script to diagnose driftmgr discovery issues
# This script tests individual AWS services to identify which ones are working

Write-Host "=== DriftMgr Discovery Diagnostic Script ===" -ForegroundColor Green
Write-Host "Testing AWS CLI and individual services..." -ForegroundColor Yellow
Write-Host ""

# Test AWS CLI availability
Write-Host "1. Testing AWS CLI availability..." -ForegroundColor Cyan
try {
    $awsVersion = aws --version 2>$null
    if ($awsVersion) {
        Write-Host "✓ AWS CLI is available" -ForegroundColor Green
        Write-Host $awsVersion
    } else {
        Write-Host "✗ AWS CLI is not available" -ForegroundColor Red
        exit 1
    }
} catch {
    Write-Host "✗ AWS CLI is not available" -ForegroundColor Red
    exit 1
}
Write-Host ""

# Test AWS credentials
Write-Host "2. Testing AWS credentials..." -ForegroundColor Cyan
try {
    $identity = aws sts get-caller-identity 2>$null
    if ($identity) {
        Write-Host "✓ AWS credentials are working" -ForegroundColor Green
        Write-Host $identity
    } else {
        Write-Host "✗ AWS credentials are not working" -ForegroundColor Red
        Write-Host "Error: $(aws sts get-caller-identity 2>&1)" -ForegroundColor Red
        exit 1
    }
} catch {
    Write-Host "✗ AWS credentials are not working" -ForegroundColor Red
    exit 1
}
Write-Host ""

# Test individual AWS services
Write-Host "3. Testing individual AWS services..." -ForegroundColor Cyan

# Get current region
$REGION = aws configure get region 2>$null
if (-not $REGION) { $REGION = "us-east-1" }
Write-Host "Using region: $REGION" -ForegroundColor Yellow
Write-Host ""

# Function to test AWS service
function Test-AWSService {
    param(
        [string]$ServiceName,
        [string]$Command,
        [string]$Query
    )
    
    Write-Host "Testing $ServiceName..." -ForegroundColor White
    try {
        $result = Invoke-Expression $Command 2>$null
        if ($result) {
            Write-Host "✓ $ServiceName discovery working" -ForegroundColor Green
        } else {
            Write-Host "✗ $ServiceName discovery failed" -ForegroundColor Red
        }
    } catch {
        Write-Host "✗ $ServiceName discovery failed" -ForegroundColor Red
    }
    Write-Host ""
}

# Test EC2
Test-AWSService "EC2" "aws ec2 describe-instances --region $REGION --query 'Reservations[*].Instances[*].[InstanceId]' --output table"

# Test RDS
Test-AWSService "RDS" "aws rds describe-db-instances --region $REGION --query 'DBInstances[*].[DBInstanceIdentifier]' --output table"

# Test S3
Test-AWSService "S3" "aws s3 ls"

# Test Lambda
Test-AWSService "Lambda" "aws lambda list-functions --region $REGION --query 'Functions[*].[FunctionName]' --output table"

# Test IAM
Test-AWSService "IAM" "aws iam list-users --query 'Users[*].[UserName]' --output table"

# Test VPC
Test-AWSService "VPC" "aws ec2 describe-vpcs --region $REGION --query 'Vpcs[*].[VpcId]' --output table"

# Test CloudFormation
Test-AWSService "CloudFormation" "aws cloudformation list-stacks --region $REGION --query 'StackSummaries[*].[StackName]' --output table"

# Test ElastiCache
Test-AWSService "ElastiCache" "aws elasticache describe-cache-clusters --region $REGION --query 'CacheClusters[*].[CacheClusterId]' --output table"

# Test ECS
Test-AWSService "ECS" "aws ecs list-clusters --region $REGION --query 'clusterArns' --output table"

# Test EKS
Test-AWSService "EKS" "aws eks list-clusters --region $REGION --query 'clusters' --output table"

# Test SQS
Test-AWSService "SQS" "aws sqs list-queues --region $REGION --query 'QueueUrls' --output table"

# Test SNS
Test-AWSService "SNS" "aws sns list-topics --region $REGION --query 'Topics[*].[TopicArn]' --output table"

# Test DynamoDB
Test-AWSService "DynamoDB" "aws dynamodb list-tables --region $REGION --query 'TableNames' --output table"

Write-Host "=== Diagnostic Complete ===" -ForegroundColor Green
Write-Host "If some services show as failed, check your IAM permissions." -ForegroundColor Yellow
Write-Host "The driftmgr discovery should work for services that show as working above." -ForegroundColor Yellow
