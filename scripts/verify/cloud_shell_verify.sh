#!/bin/bash

# Cloud Shell Verification Tool - Works in AWS CloudShell, Azure Cloud Shell, GCP Cloud Shell
# Automatically detects which cloud shell environment and runs appropriate commands

echo "=== Cloud Shell Resource Verification Tool ==="
echo "Detecting cloud shell environment..."

# Detect which cloud shell we're in
CLOUD_PROVIDER=""

if [ ! -z "$AWS_EXECUTION_ENV" ] || [ -f /home/cloudshell-user/.bashrc ]; then
    CLOUD_PROVIDER="AWS"
    echo "✓ Detected AWS CloudShell"
elif [ ! -z "$AZURE_HTTP_USER_AGENT" ] || [ -f /home/*/.azure/azureProfile.json ]; then
    CLOUD_PROVIDER="Azure"
    echo "✓ Detected Azure Cloud Shell"
elif [ ! -z "$GOOGLE_CLOUD_SHELL" ] || [ -f /google/devshell/bashrc.google ]; then
    CLOUD_PROVIDER="GCP"
    echo "✓ Detected Google Cloud Shell"
elif command -v doctl &> /dev/null; then
    CLOUD_PROVIDER="DigitalOcean"
    echo "✓ Detected DigitalOcean CLI"
else
    echo "⚠ Could not detect cloud shell environment"
    echo "Please specify provider: AWS, Azure, GCP, or DigitalOcean"
    read -p "Provider: " CLOUD_PROVIDER
fi

# Function to count AWS resources
verify_aws_resources() {
    echo ""
    echo "### AWS Resource Verification ###"
    
    # EC2 Instances
    EC2_COUNT=$(aws ec2 describe-instances --query 'Reservations[*].Instances[*].[InstanceId]' --output text 2>/dev/null | wc -l)
    echo "EC2 Instances: $EC2_COUNT"
    
    # S3 Buckets
    S3_COUNT=$(aws s3api list-buckets --query 'Buckets[*].Name' --output text 2>/dev/null | wc -w)
    echo "S3 Buckets: $S3_COUNT"
    
    # RDS Instances
    RDS_COUNT=$(aws rds describe-db-instances --query 'DBInstances[*].DBInstanceIdentifier' --output text 2>/dev/null | wc -w)
    echo "RDS Instances: $RDS_COUNT"
    
    # Lambda Functions
    LAMBDA_COUNT=$(aws lambda list-functions --query 'Functions[*].FunctionName' --output text 2>/dev/null | wc -w)
    echo "Lambda Functions: $LAMBDA_COUNT"
    
    # VPCs
    VPC_COUNT=$(aws ec2 describe-vpcs --query 'Vpcs[*].VpcId' --output text 2>/dev/null | wc -w)
    echo "VPCs: $VPC_COUNT"
    
    # Security Groups
    SG_COUNT=$(aws ec2 describe-security-groups --query 'SecurityGroups[*].GroupId' --output text 2>/dev/null | wc -w)
    echo "Security Groups: $SG_COUNT"
    
    # IAM Users
    IAM_USER_COUNT=$(aws iam list-users --query 'Users[*].UserName' --output text 2>/dev/null | wc -w)
    echo "IAM Users: $IAM_USER_COUNT"
    
    # IAM Roles
    IAM_ROLE_COUNT=$(aws iam list-roles --query 'Roles[*].RoleName' --output text 2>/dev/null | wc -w)
    echo "IAM Roles: $IAM_ROLE_COUNT"
    
    # DynamoDB Tables
    DYNAMODB_COUNT=$(aws dynamodb list-tables --query 'TableNames[*]' --output text 2>/dev/null | wc -w)
    echo "DynamoDB Tables: $DYNAMODB_COUNT"
    
    # ECS Clusters
    ECS_COUNT=$(aws ecs list-clusters --query 'clusterArns[*]' --output text 2>/dev/null | wc -w)
    echo "ECS Clusters: $ECS_COUNT"
    
    # Load Balancers
    ELB_COUNT=$(aws elbv2 describe-load-balancers --query 'LoadBalancers[*].LoadBalancerArn' --output text 2>/dev/null | wc -w)
    echo "Load Balancers: $ELB_COUNT"
    
    # CloudFront Distributions
    CF_COUNT=$(aws cloudfront list-distributions --query 'DistributionList.Items[*].Id' --output text 2>/dev/null | wc -w)
    echo "CloudFront Distributions: $CF_COUNT"
    
    TOTAL_AWS=$((EC2_COUNT + S3_COUNT + RDS_COUNT + LAMBDA_COUNT + VPC_COUNT + SG_COUNT + IAM_USER_COUNT + IAM_ROLE_COUNT + DYNAMODB_COUNT + ECS_COUNT + ELB_COUNT + CF_COUNT))
    echo ""
    echo "Total AWS Resources: $TOTAL_AWS"
    
    # Export for comparison
    echo "$TOTAL_AWS" > /tmp/aws_resource_count.txt
}

# Function to count Azure resources
verify_azure_resources() {
    echo ""
    echo "### Azure Resource Verification ###"
    
    # Virtual Machines
    VM_COUNT=$(az vm list --query 'length(@)' --output tsv 2>/dev/null || echo 0)
    echo "Virtual Machines: $VM_COUNT"
    
    # Storage Accounts
    STORAGE_COUNT=$(az storage account list --query 'length(@)' --output tsv 2>/dev/null || echo 0)
    echo "Storage Accounts: $STORAGE_COUNT"
    
    # Resource Groups
    RG_COUNT=$(az group list --query 'length(@)' --output tsv 2>/dev/null || echo 0)
    echo "Resource Groups: $RG_COUNT"
    
    # Virtual Networks
    VNET_COUNT=$(az network vnet list --query 'length(@)' --output tsv 2>/dev/null || echo 0)
    echo "Virtual Networks: $VNET_COUNT"
    
    # Network Security Groups
    NSG_COUNT=$(az network nsg list --query 'length(@)' --output tsv 2>/dev/null || echo 0)
    echo "Network Security Groups: $NSG_COUNT"
    
    # SQL Servers
    SQL_COUNT=$(az sql server list --query 'length(@)' --output tsv 2>/dev/null || echo 0)
    echo "SQL Servers: $SQL_COUNT"
    
    # SQL Databases
    SQLDB_COUNT=$(az sql db list --all --query 'length(@)' --output tsv 2>/dev/null || echo 0)
    echo "SQL Databases: $SQLDB_COUNT"
    
    # CosmosDB Accounts
    COSMOS_COUNT=$(az cosmosdb list --query 'length(@)' --output tsv 2>/dev/null || echo 0)
    echo "CosmosDB Accounts: $COSMOS_COUNT"
    
    # Key Vaults
    KV_COUNT=$(az keyvault list --query 'length(@)' --output tsv 2>/dev/null || echo 0)
    echo "Key Vaults: $KV_COUNT"
    
    # Function Apps
    FUNC_COUNT=$(az functionapp list --query 'length(@)' --output tsv 2>/dev/null || echo 0)
    echo "Function Apps: $FUNC_COUNT"
    
    # Container Registries
    ACR_COUNT=$(az acr list --query 'length(@)' --output tsv 2>/dev/null || echo 0)
    echo "Container Registries: $ACR_COUNT"
    
    # AKS Clusters
    AKS_COUNT=$(az aks list --query 'length(@)' --output tsv 2>/dev/null || echo 0)
    echo "AKS Clusters: $AKS_COUNT"
    
    TOTAL_AZURE=$((VM_COUNT + STORAGE_COUNT + RG_COUNT + VNET_COUNT + NSG_COUNT + SQL_COUNT + SQLDB_COUNT + COSMOS_COUNT + KV_COUNT + FUNC_COUNT + ACR_COUNT + AKS_COUNT))
    echo ""
    echo "Total Azure Resources: $TOTAL_AZURE"
    
    # Export for comparison
    echo "$TOTAL_AZURE" > /tmp/azure_resource_count.txt
}

# Function to count GCP resources
verify_gcp_resources() {
    echo ""
    echo "### GCP Resource Verification ###"
    
    PROJECT=${1:-$(gcloud config get-value project)}
    echo "Project: $PROJECT"
    
    # Compute Instances
    INSTANCE_COUNT=$(gcloud compute instances list --project=$PROJECT --format='value(name)' 2>/dev/null | wc -l)
    echo "Compute Instances: $INSTANCE_COUNT"
    
    # Storage Buckets
    BUCKET_COUNT=$(gcloud storage buckets list --project=$PROJECT --format='value(name)' 2>/dev/null | wc -l)
    echo "Storage Buckets: $BUCKET_COUNT"
    
    # Persistent Disks
    DISK_COUNT=$(gcloud compute disks list --project=$PROJECT --format='value(name)' 2>/dev/null | wc -l)
    echo "Persistent Disks: $DISK_COUNT"
    
    # Networks
    NETWORK_COUNT=$(gcloud compute networks list --project=$PROJECT --format='value(name)' 2>/dev/null | wc -l)
    echo "Networks: $NETWORK_COUNT"
    
    # Subnets
    SUBNET_COUNT=$(gcloud compute networks subnets list --project=$PROJECT --format='value(name)' 2>/dev/null | wc -l)
    echo "Subnets: $SUBNET_COUNT"
    
    # Firewall Rules
    FIREWALL_COUNT=$(gcloud compute firewall-rules list --project=$PROJECT --format='value(name)' 2>/dev/null | wc -l)
    echo "Firewall Rules: $FIREWALL_COUNT"
    
    # SQL Instances
    SQL_COUNT=$(gcloud sql instances list --project=$PROJECT --format='value(name)' 2>/dev/null | wc -l)
    echo "Cloud SQL Instances: $SQL_COUNT"
    
    # GKE Clusters
    GKE_COUNT=$(gcloud container clusters list --project=$PROJECT --format='value(name)' 2>/dev/null | wc -l)
    echo "GKE Clusters: $GKE_COUNT"
    
    # Cloud Functions
    FUNCTION_COUNT=$(gcloud functions list --project=$PROJECT --format='value(name)' 2>/dev/null | wc -l)
    echo "Cloud Functions: $FUNCTION_COUNT"
    
    # Cloud Run Services
    RUN_COUNT=$(gcloud run services list --project=$PROJECT --format='value(name)' 2>/dev/null | wc -l)
    echo "Cloud Run Services: $RUN_COUNT"
    
    # Pub/Sub Topics
    TOPIC_COUNT=$(gcloud pubsub topics list --project=$PROJECT --format='value(name)' 2>/dev/null | wc -l)
    echo "Pub/Sub Topics: $TOPIC_COUNT"
    
    # Service Accounts
    SA_COUNT=$(gcloud iam service-accounts list --project=$PROJECT --format='value(email)' 2>/dev/null | wc -l)
    echo "Service Accounts: $SA_COUNT"
    
    TOTAL_GCP=$((INSTANCE_COUNT + BUCKET_COUNT + DISK_COUNT + NETWORK_COUNT + SUBNET_COUNT + FIREWALL_COUNT + SQL_COUNT + GKE_COUNT + FUNCTION_COUNT + RUN_COUNT + TOPIC_COUNT + SA_COUNT))
    echo ""
    echo "Total GCP Resources: $TOTAL_GCP"
    
    # Export for comparison
    echo "$TOTAL_GCP" > /tmp/gcp_resource_count.txt
}

# Function to count DigitalOcean resources
verify_digitalocean_resources() {
    echo ""
    echo "### DigitalOcean Resource Verification ###"
    
    # Droplets
    DROPLET_COUNT=$(doctl compute droplet list --format ID --no-header 2>/dev/null | wc -l)
    echo "Droplets: $DROPLET_COUNT"
    
    # Volumes
    VOLUME_COUNT=$(doctl compute volume list --format ID --no-header 2>/dev/null | wc -l)
    echo "Volumes: $VOLUME_COUNT"
    
    # Databases
    DB_COUNT=$(doctl databases list --format ID --no-header 2>/dev/null | wc -l)
    echo "Database Clusters: $DB_COUNT"
    
    # Load Balancers
    LB_COUNT=$(doctl compute load-balancer list --format ID --no-header 2>/dev/null | wc -l)
    echo "Load Balancers: $LB_COUNT"
    
    # Kubernetes Clusters
    K8S_COUNT=$(doctl kubernetes cluster list --format ID --no-header 2>/dev/null | wc -l)
    echo "Kubernetes Clusters: $K8S_COUNT"
    
    # Spaces (Object Storage)
    SPACES_COUNT=$(doctl compute spaces list --format Name --no-header 2>/dev/null | wc -l)
    echo "Spaces: $SPACES_COUNT"
    
    # Domains
    DOMAIN_COUNT=$(doctl compute domain list --format Domain --no-header 2>/dev/null | wc -l)
    echo "Domains: $DOMAIN_COUNT"
    
    # Firewalls
    FW_COUNT=$(doctl compute firewall list --format ID --no-header 2>/dev/null | wc -l)
    echo "Firewalls: $FW_COUNT"
    
    # Floating IPs
    FIP_COUNT=$(doctl compute floating-ip list --format IP --no-header 2>/dev/null | wc -l)
    echo "Floating IPs: $FIP_COUNT"
    
    # VPCs
    VPC_COUNT=$(doctl vpcs list --format ID --no-header 2>/dev/null | wc -l)
    echo "VPCs: $VPC_COUNT"
    
    # Apps
    APP_COUNT=$(doctl apps list --format ID --no-header 2>/dev/null | wc -l)
    echo "Apps: $APP_COUNT"
    
    # SSH Keys
    KEY_COUNT=$(doctl compute ssh-key list --format ID --no-header 2>/dev/null | wc -l)
    echo "SSH Keys: $KEY_COUNT"
    
    TOTAL_DO=$((DROPLET_COUNT + VOLUME_COUNT + DB_COUNT + LB_COUNT + K8S_COUNT + SPACES_COUNT + DOMAIN_COUNT + FW_COUNT + FIP_COUNT + VPC_COUNT + APP_COUNT + KEY_COUNT))
    echo ""
    echo "Total DigitalOcean Resources: $TOTAL_DO"
    
    # Export for comparison
    echo "$TOTAL_DO" > /tmp/do_resource_count.txt
}

# Run verification based on detected environment
case $CLOUD_PROVIDER in
    AWS)
        verify_aws_resources
        ;;
    Azure)
        verify_azure_resources
        ;;
    GCP)
        verify_gcp_resources
        ;;
    DigitalOcean)
        verify_digitalocean_resources
        ;;
    *)
        echo "Unsupported cloud provider: $CLOUD_PROVIDER"
        exit 1
        ;;
esac

echo ""
echo "=== Verification Complete ==="
echo "Resource counts saved to /tmp/*_resource_count.txt"
echo ""
echo "To compare with DriftMgr results:"
echo "1. Run: ./driftmgr.exe discover --provider $(echo $CLOUD_PROVIDER | tr '[:upper:]' '[:lower:]') --format json"
echo "2. Compare the resource counts"