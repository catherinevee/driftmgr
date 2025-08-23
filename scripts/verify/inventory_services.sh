#!/bin/bash

# Cloud Provider Inventory Services Configuration
# Integrates with native inventory and compliance services for verification

echo "=== Cloud Provider Inventory Services Setup ==="
echo ""

# Function to setup AWS Config for inventory
setup_aws_config() {
    echo "### Setting up AWS Config ###"
    
    # Check if AWS Config is enabled
    CONFIG_STATUS=$(aws configservice describe-configuration-recorders --query 'ConfigurationRecorders[0].name' --output text 2>/dev/null)
    
    if [ -z "$CONFIG_STATUS" ]; then
        echo "AWS Config not configured. Setting up..."
        
        # Create S3 bucket for Config
        BUCKET_NAME="aws-config-bucket-$(date +%s)"
        aws s3api create-bucket --bucket $BUCKET_NAME --region us-east-1
        
        # Create IAM role for Config
        cat > config-role-trust.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "config.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
        
        aws iam create-role \
            --role-name ConfigRole \
            --assume-role-policy-document file://config-role-trust.json
        
        aws iam attach-role-policy \
            --role-name ConfigRole \
            --policy-arn arn:aws:iam::aws:policy/service-role/ConfigRole
        
        # Create Config recorder
        aws configservice put-configuration-recorder \
            --configuration-recorder name=default,roleArn=arn:aws:iam::$(aws sts get-caller-identity --query Account --output text):role/ConfigRole \
            --recording-group allSupported=true
        
        # Create delivery channel
        aws configservice put-delivery-channel \
            --delivery-channel name=default,s3BucketName=$BUCKET_NAME
        
        # Start recorder
        aws configservice start-configuration-recorder --configuration-recorder-name default
        
        echo "✓ AWS Config setup complete"
    else
        echo "✓ AWS Config already configured: $CONFIG_STATUS"
    fi
    
    # Query AWS Config inventory
    echo ""
    echo "Querying AWS Config inventory..."
    
    # Get resource counts by type
    aws configservice select-aggregate-resource-config \
        --expression "SELECT resourceType, COUNT(*) WHERE resourceType LIKE 'AWS::%' GROUP BY resourceType" \
        --configuration-aggregator-name default \
        --output table 2>/dev/null || \
    aws configservice select-resource-config \
        --expression "SELECT resourceType, COUNT(*) WHERE resourceType LIKE 'AWS::%' GROUP BY resourceType" \
        --output table
    
    # Export inventory to JSON
    aws configservice select-resource-config \
        --expression "SELECT resourceId, resourceType, resourceName, awsRegion FROM resources" \
        --output json > aws_config_inventory.json
    
    echo "✓ AWS Config inventory exported to aws_config_inventory.json"
}

# Function to setup AWS Systems Manager Inventory
setup_aws_ssm_inventory() {
    echo ""
    echo "### Setting up AWS Systems Manager Inventory ###"
    
    # Create inventory association for EC2 instances
    aws ssm create-association \
        --name "AWS-GatherSoftwareInventory" \
        --targets "Key=InstanceIds,Values=*" \
        --schedule-expression "rate(30 minutes)" \
        --parameters '{"applications":["Enabled"],"networkConfig":["Enabled"],"instanceDetailedInformation":["Enabled"]}' \
        2>/dev/null || echo "SSM Inventory association already exists"
    
    # Query SSM inventory
    aws ssm get-inventory-schema --output json > ssm_inventory_schema.json
    aws ssm describe-instance-information --output json > ssm_instances.json
    
    echo "✓ SSM Inventory data exported"
}

# Function to setup Azure Resource Graph
setup_azure_resource_graph() {
    echo ""
    echo "### Azure Resource Graph Queries ###"
    
    # Check if logged in
    if ! az account show &>/dev/null; then
        echo "Not logged into Azure. Please run: az login"
        return 1
    fi
    
    # Query all resources
    echo "Querying Azure Resource Graph..."
    
    # Count by resource type
    az graph query -q "Resources | summarize count() by type | order by count_ desc" \
        --output table
    
    # Export full inventory
    az graph query -q "Resources | project id, name, type, location, resourceGroup, subscriptionId, tags | limit 5000" \
        --output json > azure_graph_inventory.json
    
    # Query resource changes (if available)
    az graph query -q "ResourceChanges | extend changeTime=todatetime(properties.changeAttributes.timestamp) | project changeTime, resourceId, changeType, properties | order by changeTime desc | limit 100" \
        --output json > azure_resource_changes.json 2>/dev/null || echo "Resource changes not available"
    
    # Azure Policy compliance
    az policy state summarize --output json > azure_policy_compliance.json
    
    echo "✓ Azure Resource Graph inventory exported"
}

# Function to setup GCP Cloud Asset Inventory
setup_gcp_asset_inventory() {
    echo ""
    echo "### GCP Cloud Asset Inventory ###"
    
    PROJECT_ID=$(gcloud config get-value project)
    if [ -z "$PROJECT_ID" ]; then
        echo "No GCP project set. Please run: gcloud config set project PROJECT_ID"
        return 1
    fi
    
    echo "Project: $PROJECT_ID"
    
    # Enable Cloud Asset API if not enabled
    gcloud services enable cloudasset.googleapis.com --project=$PROJECT_ID 2>/dev/null
    
    # Create temporary bucket for export
    TEMP_BUCKET="gs://asset-inventory-temp-$(date +%s)"
    gsutil mb $TEMP_BUCKET 2>/dev/null || true
    
    # Export asset inventory
    echo "Exporting GCP asset inventory..."
    gcloud asset export \
        --project=$PROJECT_ID \
        --output-path=$TEMP_BUCKET/inventory.json \
        --content-type=RESOURCE \
        --snapshot-time=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    
    # Download inventory
    gsutil cp $TEMP_BUCKET/inventory.json gcp_asset_inventory.json
    
    # Clean up temp bucket
    gsutil rm -r $TEMP_BUCKET
    
    # Query specific resource types
    echo "Querying resource counts..."
    gcloud asset search-all-resources \
        --project=$PROJECT_ID \
        --format="csv(assetType)" | \
        tail -n +2 | \
        sort | uniq -c | \
        sort -rn > gcp_resource_counts.txt
    
    echo "✓ GCP Asset Inventory exported"
}

# Function to setup DigitalOcean inventory
setup_digitalocean_inventory() {
    echo ""
    echo "### DigitalOcean Inventory ###"
    
    if ! command -v doctl &> /dev/null; then
        echo "doctl not installed. Please install it first."
        return 1
    fi
    
    echo "Collecting DigitalOcean inventory..."
    
    # Export all resources to JSON
    {
        echo "{"
        echo '  "droplets": '
        doctl compute droplet list --output json
        echo ','
        echo '  "volumes": '
        doctl compute volume list --output json
        echo ','
        echo '  "databases": '
        doctl databases list --output json
        echo ','
        echo '  "kubernetes_clusters": '
        doctl kubernetes cluster list --output json
        echo ','
        echo '  "load_balancers": '
        doctl compute load-balancer list --output json
        echo ','
        echo '  "domains": '
        doctl compute domain list --output json
        echo ','
        echo '  "vpcs": '
        doctl vpcs list --output json
        echo ','
        echo '  "apps": '
        doctl apps list --output json
        echo ','
        echo '  "projects": '
        doctl projects list --output json
        echo "}"
    } > digitalocean_inventory.json 2>/dev/null
    
    # Count resources
    echo "Resource counts:"
    echo "Droplets: $(doctl compute droplet list --format ID --no-header | wc -l)"
    echo "Volumes: $(doctl compute volume list --format ID --no-header | wc -l)"
    echo "Databases: $(doctl databases list --format ID --no-header | wc -l)"
    echo "Kubernetes: $(doctl kubernetes cluster list --format ID --no-header | wc -l)"
    
    echo "✓ DigitalOcean inventory exported"
}

# Function to compare inventories with DriftMgr
compare_with_driftmgr() {
    echo ""
    echo "### Comparing with DriftMgr Results ###"
    
    # Run DriftMgr discovery
    if [ -f "./driftmgr.exe" ]; then
        ./driftmgr.exe discover --auto --format json --output driftmgr_inventory.json
    elif [ -f "./driftmgr" ]; then
        ./driftmgr discover --auto --format json --output driftmgr_inventory.json
    else
        echo "DriftMgr not found. Please build it first."
        return 1
    fi
    
    # Create comparison script
    cat > compare_inventories.py <<'EOF'
import json
import sys

def load_json(filename):
    try:
        with open(filename, 'r') as f:
            return json.load(f)
    except:
        return {}

# Load inventories
driftmgr = load_json('driftmgr_inventory.json')
aws_config = load_json('aws_config_inventory.json')
azure_graph = load_json('azure_graph_inventory.json')
gcp_asset = load_json('gcp_asset_inventory.json')
do_inventory = load_json('digitalocean_inventory.json')

# Count resources
def count_resources(data):
    if isinstance(data, list):
        return len(data)
    elif isinstance(data, dict):
        total = 0
        for key, value in data.items():
            if isinstance(value, list):
                total += len(value)
            elif isinstance(value, dict) and 'resources' in value:
                total += len(value['resources'])
        return total
    return 0

print("\n=== Inventory Comparison ===\n")
print("Source                  | Resource Count")
print("------------------------|---------------")
print(f"DriftMgr                | {count_resources(driftmgr)}")
print(f"AWS Config              | {len(aws_config.get('Results', []))}")
print(f"Azure Resource Graph    | {len(azure_graph.get('data', []))}")
print(f"GCP Asset Inventory     | {count_resources(gcp_asset)}")
print(f"DigitalOcean Inventory  | {count_resources(do_inventory)}")

# Detailed comparison
print("\n=== Detailed Comparison ===\n")

# AWS comparison
if 'aws' in driftmgr:
    drift_aws = len(driftmgr['aws'].get('resources', []))
    config_aws = len(aws_config.get('Results', []))
    diff = drift_aws - config_aws
    status = "✓" if diff == 0 else "✗"
    print(f"AWS: DriftMgr={drift_aws}, Config={config_aws}, Diff={diff} {status}")

# Azure comparison
if 'azure' in driftmgr:
    drift_azure = len(driftmgr['azure'].get('resources', []))
    graph_azure = len(azure_graph.get('data', []))
    diff = drift_azure - graph_azure
    status = "✓" if diff == 0 else "✗"
    print(f"Azure: DriftMgr={drift_azure}, Graph={graph_azure}, Diff={diff} {status}")

# GCP comparison
if 'gcp' in driftmgr:
    drift_gcp = len(driftmgr['gcp'].get('resources', []))
    asset_gcp = count_resources(gcp_asset)
    diff = drift_gcp - asset_gcp
    status = "✓" if abs(diff) < 5 else "✗"  # Allow small variance
    print(f"GCP: DriftMgr={drift_gcp}, Assets={asset_gcp}, Diff={diff} {status}")

# DigitalOcean comparison
if 'digitalocean' in driftmgr:
    drift_do = len(driftmgr['digitalocean'].get('resources', []))
    inv_do = count_resources(do_inventory)
    diff = drift_do - inv_do
    status = "✓" if diff == 0 else "✗"
    print(f"DO: DriftMgr={drift_do}, Inventory={inv_do}, Diff={diff} {status}")
EOF
    
    # Run comparison
    python3 compare_inventories.py
    
    echo ""
    echo "✓ Comparison complete"
}

# Main menu
echo "Select inventory service to configure:"
echo "1. AWS Config & Systems Manager"
echo "2. Azure Resource Graph"
echo "3. GCP Cloud Asset Inventory"
echo "4. DigitalOcean Inventory"
echo "5. All Providers"
echo "6. Compare with DriftMgr"
echo ""
read -p "Choice (1-6): " choice

case $choice in
    1)
        setup_aws_config
        setup_aws_ssm_inventory
        ;;
    2)
        setup_azure_resource_graph
        ;;
    3)
        setup_gcp_asset_inventory
        ;;
    4)
        setup_digitalocean_inventory
        ;;
    5)
        setup_aws_config
        setup_aws_ssm_inventory
        setup_azure_resource_graph
        setup_gcp_asset_inventory
        setup_digitalocean_inventory
        ;;
    6)
        compare_with_driftmgr
        ;;
    *)
        echo "Invalid choice"
        exit 1
        ;;
esac

echo ""
echo "=== Inventory Service Configuration Complete ==="
echo "Files created:"
echo "  - aws_config_inventory.json"
echo "  - azure_graph_inventory.json"
echo "  - gcp_asset_inventory.json"
echo "  - digitalocean_inventory.json"
echo ""
echo "Run option 6 to compare with DriftMgr results"