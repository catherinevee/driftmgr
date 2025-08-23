#!/bin/bash

# Tag-Based Resource Verification
# Tags all discovered resources and verifies them against DriftMgr findings

echo "=== Tag-Based Resource Verification System ==="
echo ""

# Configuration
TAG_KEY="DriftMgrVerified"
TAG_VALUE="$(date +%Y%m%d-%H%M%S)"
DISCOVERY_TAG="DiscoveredBy=DriftMgr"
TIMESTAMP_TAG="VerificationTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# Function to tag AWS resources
tag_aws_resources() {
    echo "### Tagging AWS Resources ###"
    
    # Get all resources that support tagging
    echo "Discovering taggable AWS resources..."
    
    # EC2 Instances
    INSTANCE_ARNS=$(aws ec2 describe-instances \
        --query 'Reservations[*].Instances[*].[InstanceId]' \
        --output text | while read id; do
        echo "arn:aws:ec2:$AWS_REGION:$AWS_ACCOUNT:instance/$id"
    done)
    
    # S3 Buckets (tags are per bucket, not ARN-based)
    BUCKETS=$(aws s3api list-buckets --query 'Buckets[*].Name' --output text)
    
    # RDS Instances
    RDS_ARNS=$(aws rds describe-db-instances \
        --query 'DBInstances[*].DBInstanceArn' \
        --output text)
    
    # Lambda Functions
    LAMBDA_ARNS=$(aws lambda list-functions \
        --query 'Functions[*].FunctionArn' \
        --output text)
    
    # Tag EC2 resources
    if [ ! -z "$INSTANCE_ARNS" ]; then
        echo "Tagging EC2 instances..."
        echo "$INSTANCE_ARNS" | while read arn; do
            if [ ! -z "$arn" ]; then
                instance_id=$(echo $arn | rev | cut -d'/' -f1 | rev)
                aws ec2 create-tags \
                    --resources $instance_id \
                    --tags Key=$TAG_KEY,Value=$TAG_VALUE \
                           Key=DiscoveredBy,Value=DriftMgr \
                           Key=VerificationTime,Value=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
                    2>/dev/null && echo "✓ Tagged instance: $instance_id"
            fi
        done
    fi
    
    # Tag S3 buckets
    if [ ! -z "$BUCKETS" ]; then
        echo "Tagging S3 buckets..."
        echo "$BUCKETS" | while read bucket; do
            if [ ! -z "$bucket" ]; then
                aws s3api put-bucket-tagging \
                    --bucket $bucket \
                    --tagging "{\"TagSet\": [
                        {\"Key\": \"$TAG_KEY\", \"Value\": \"$TAG_VALUE\"},
                        {\"Key\": \"DiscoveredBy\", \"Value\": \"DriftMgr\"},
                        {\"Key\": \"VerificationTime\", \"Value\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}
                    ]}" 2>/dev/null && echo "✓ Tagged bucket: $bucket"
            fi
        done
    fi
    
    # Tag RDS instances
    if [ ! -z "$RDS_ARNS" ]; then
        echo "Tagging RDS instances..."
        echo "$RDS_ARNS" | while read arn; do
            if [ ! -z "$arn" ]; then
                aws rds add-tags-to-resource \
                    --resource-name $arn \
                    --tags Key=$TAG_KEY,Value=$TAG_VALUE \
                           Key=DiscoveredBy,Value=DriftMgr \
                    2>/dev/null && echo "✓ Tagged RDS: $(basename $arn)"
            fi
        done
    fi
    
    # Tag Lambda functions
    if [ ! -z "$LAMBDA_ARNS" ]; then
        echo "Tagging Lambda functions..."
        echo "$LAMBDA_ARNS" | while read arn; do
            if [ ! -z "$arn" ]; then
                aws lambda tag-resource \
                    --resource $arn \
                    --tags $TAG_KEY=$TAG_VALUE,DiscoveredBy=DriftMgr \
                    2>/dev/null && echo "✓ Tagged Lambda: $(basename $arn)"
            fi
        done
    fi
    
    echo "✓ AWS resource tagging complete"
}

# Function to tag Azure resources
tag_azure_resources() {
    echo ""
    echo "### Tagging Azure Resources ###"
    
    # Get all resources
    echo "Discovering Azure resources..."
    
    # Tag all resources in subscription
    az resource list --query "[].id" --output tsv | while read resource_id; do
        if [ ! -z "$resource_id" ]; then
            echo "Tagging: $resource_id"
            az tag create --resource-id "$resource_id" \
                --tags $TAG_KEY=$TAG_VALUE \
                       DiscoveredBy=DriftMgr \
                       VerificationTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
                2>/dev/null && echo "✓ Tagged: $(basename $resource_id)"
        fi
    done
    
    # Alternative: Tag by resource group
    az group list --query "[].name" --output tsv | while read rg; do
        if [ ! -z "$rg" ]; then
            az group update --name $rg \
                --tags $TAG_KEY=$TAG_VALUE DiscoveredBy=DriftMgr \
                2>/dev/null && echo "✓ Tagged resource group: $rg"
        fi
    done
    
    echo "✓ Azure resource tagging complete"
}

# Function to tag GCP resources
tag_gcp_resources() {
    echo ""
    echo "### Tagging GCP Resources ###"
    
    PROJECT_ID=$(gcloud config get-value project)
    echo "Project: $PROJECT_ID"
    
    # Tag Compute instances
    echo "Tagging Compute instances..."
    gcloud compute instances list --format="value(name,zone)" | while read name zone; do
        if [ ! -z "$name" ]; then
            gcloud compute instances add-labels $name \
                --zone=$zone \
                --labels=${TAG_KEY,,}=${TAG_VALUE,,},discovered_by=driftmgr,verification_time=$(date +%Y%m%d) \
                2>/dev/null && echo "✓ Tagged instance: $name"
        fi
    done
    
    # Tag Storage buckets
    echo "Tagging Storage buckets..."
    gsutil ls -p $PROJECT_ID | while read bucket; do
        if [ ! -z "$bucket" ]; then
            gsutil label ch -l ${TAG_KEY,,}:${TAG_VALUE,,} $bucket 2>/dev/null
            gsutil label ch -l discovered_by:driftmgr $bucket 2>/dev/null
            echo "✓ Tagged bucket: $bucket"
        fi
    done
    
    # Tag Cloud SQL instances
    echo "Tagging Cloud SQL instances..."
    gcloud sql instances list --format="value(name)" | while read instance; do
        if [ ! -z "$instance" ]; then
            gcloud sql instances patch $instance \
                --update-labels=${TAG_KEY,,}=${TAG_VALUE,,},discovered_by=driftmgr \
                2>/dev/null && echo "✓ Tagged SQL instance: $instance"
        fi
    done
    
    echo "✓ GCP resource tagging complete"
}

# Function to tag DigitalOcean resources
tag_digitalocean_resources() {
    echo ""
    echo "### Tagging DigitalOcean Resources ###"
    
    # Create tag if it doesn't exist
    TAG_NAME="driftmgr-verified-$TAG_VALUE"
    doctl compute tag create $TAG_NAME 2>/dev/null
    
    # Tag Droplets
    echo "Tagging Droplets..."
    doctl compute droplet list --format ID --no-header | while read id; do
        if [ ! -z "$id" ]; then
            doctl compute droplet tag $id --tag-name $TAG_NAME 2>/dev/null && \
                echo "✓ Tagged droplet: $id"
        fi
    done
    
    # Tag Volumes
    echo "Tagging Volumes..."
    doctl compute volume list --format ID --no-header | while read id; do
        if [ ! -z "$id" ]; then
            doctl compute volume tag $id --tag-name $TAG_NAME 2>/dev/null && \
                echo "✓ Tagged volume: $id"
        fi
    done
    
    # Tag Load Balancers
    echo "Tagging Load Balancers..."
    doctl compute load-balancer list --format ID --no-header | while read id; do
        if [ ! -z "$id" ]; then
            doctl compute load-balancer add-tag $id --tag-name $TAG_NAME 2>/dev/null && \
                echo "✓ Tagged load balancer: $id"
        fi
    done
    
    echo "✓ DigitalOcean resource tagging complete"
}

# Function to verify tagged resources
verify_tagged_resources() {
    echo ""
    echo "### Verifying Tagged Resources ###"
    
    # Run DriftMgr discovery
    echo "Running DriftMgr discovery..."
    ./driftmgr.exe discover --auto --format json --output driftmgr_results.json
    
    # Count DriftMgr discovered resources
    DRIFT_TOTAL=$(jq '[.aws.resources, .azure.resources, .gcp.resources, .digitalocean.resources] | add | length' driftmgr_results.json)
    echo "DriftMgr discovered: $DRIFT_TOTAL resources"
    
    # Count tagged resources by provider
    echo ""
    echo "Counting tagged resources..."
    
    # AWS tagged resources
    AWS_TAGGED=$(aws resourcegroupstaggingapi get-resources \
        --tag-filters Key=$TAG_KEY,Values=$TAG_VALUE \
        --query 'ResourceTagMappingList | length' \
        --output text 2>/dev/null || echo 0)
    echo "AWS tagged: $AWS_TAGGED"
    
    # Azure tagged resources
    AZURE_TAGGED=$(az resource list \
        --tag $TAG_KEY=$TAG_VALUE \
        --query 'length(@)' \
        --output tsv 2>/dev/null || echo 0)
    echo "Azure tagged: $AZURE_TAGGED"
    
    # GCP tagged resources (using labels)
    GCP_TAGGED=$(gcloud compute instances list \
        --filter="labels.${TAG_KEY,,}=${TAG_VALUE,,}" \
        --format="value(name)" | wc -l)
    echo "GCP tagged: $GCP_TAGGED"
    
    # DigitalOcean tagged resources
    DO_TAG="driftmgr-verified-$TAG_VALUE"
    DO_TAGGED=$(doctl compute tag get $DO_TAG --format ResourceCount --no-header 2>/dev/null || echo 0)
    echo "DigitalOcean tagged: $DO_TAGGED"
    
    TOTAL_TAGGED=$((AWS_TAGGED + AZURE_TAGGED + GCP_TAGGED + DO_TAGGED))
    echo ""
    echo "Total tagged resources: $TOTAL_TAGGED"
    echo "DriftMgr discovered: $DRIFT_TOTAL"
    
    # Compare results
    if [ "$TOTAL_TAGGED" -eq "$DRIFT_TOTAL" ]; then
        echo "[OK] Verification PASSED: All discovered resources are tagged"
    else
        DIFF=$((DRIFT_TOTAL - TOTAL_TAGGED))
        echo "[WARNING]  Verification MISMATCH: $DIFF resources not tagged"
        
        # Find untagged resources
        echo ""
        echo "Finding untagged resources..."
        find_untagged_resources
    fi
}

# Function to find untagged resources
find_untagged_resources() {
    echo ""
    echo "### Untagged Resources Report ###"
    
    # AWS untagged
    echo "AWS untagged resources:"
    aws resourcegroupstaggingapi get-resources \
        --query "ResourceTagMappingList[?!Tags[?Key=='$TAG_KEY']].ResourceARN" \
        --output json > aws_untagged.json
    jq -r '.[]' aws_untagged.json 2>/dev/null | head -10
    
    # Azure untagged
    echo ""
    echo "Azure untagged resources:"
    az resource list \
        --query "[?tags.$TAG_KEY == null].id" \
        --output json > azure_untagged.json
    jq -r '.[]' azure_untagged.json 2>/dev/null | head -10
    
    # Create untagged report
    cat > untagged_resources_report.md <<EOF
# Untagged Resources Report

**Verification Date:** $(date -u +%Y-%m-%dT%H:%M:%SZ)
**Tag Key:** $TAG_KEY
**Tag Value:** $TAG_VALUE

## Summary
- Total DriftMgr Discovered: $DRIFT_TOTAL
- Total Tagged: $TOTAL_TAGGED
- Untagged Resources: $((DRIFT_TOTAL - TOTAL_TAGGED))

## Untagged Resources by Provider

### AWS
$(jq -r '.[] | "- " + .' aws_untagged.json 2>/dev/null | head -20)

### Azure
$(jq -r '.[] | "- " + .' azure_untagged.json 2>/dev/null | head -20)

## Recommendations
1. Review untagged resources for compliance
2. Update tagging automation to cover all resource types
3. Consider implementing tag policies to enforce tagging
EOF
    
    echo ""
    echo "✓ Untagged resources report saved to untagged_resources_report.md"
}

# Function to cleanup verification tags
cleanup_tags() {
    echo ""
    echo "### Cleaning Up Verification Tags ###"
    
    read -p "Remove verification tags? (y/n): " confirm
    if [ "$confirm" != "y" ]; then
        echo "Cleanup cancelled"
        return
    fi
    
    # Remove AWS tags
    echo "Removing AWS tags..."
    aws resourcegroupstaggingapi get-resources \
        --tag-filters Key=$TAG_KEY,Values=$TAG_VALUE \
        --query 'ResourceTagMappingList[*].ResourceARN' \
        --output text | while read arn; do
        if [ ! -z "$arn" ]; then
            aws resourcegroupstaggingapi untag-resources \
                --resource-arn-list $arn \
                --tag-keys $TAG_KEY 2>/dev/null
        fi
    done
    
    # Remove Azure tags
    echo "Removing Azure tags..."
    az resource list --tag $TAG_KEY=$TAG_VALUE --query "[].id" --output tsv | while read id; do
        if [ ! -z "$id" ]; then
            az tag delete --resource-id "$id" --name $TAG_KEY 2>/dev/null
        fi
    done
    
    # Remove DigitalOcean tags
    echo "Removing DigitalOcean tags..."
    DO_TAG="driftmgr-verified-$TAG_VALUE"
    doctl compute tag delete $DO_TAG --force 2>/dev/null
    
    echo "✓ Cleanup complete"
}

# Main menu
echo "Tag-Based Verification Options:"
echo "1. Tag all resources (all providers)"
echo "2. Tag AWS resources only"
echo "3. Tag Azure resources only"
echo "4. Tag GCP resources only"
echo "5. Tag DigitalOcean resources only"
echo "6. Verify tagged resources"
echo "7. Find untagged resources"
echo "8. Cleanup verification tags"
echo ""
read -p "Choice (1-8): " choice

case $choice in
    1)
        tag_aws_resources
        tag_azure_resources
        tag_gcp_resources
        tag_digitalocean_resources
        verify_tagged_resources
        ;;
    2)
        tag_aws_resources
        ;;
    3)
        tag_azure_resources
        ;;
    4)
        tag_gcp_resources
        ;;
    5)
        tag_digitalocean_resources
        ;;
    6)
        verify_tagged_resources
        ;;
    7)
        find_untagged_resources
        ;;
    8)
        cleanup_tags
        ;;
    *)
        echo "Invalid choice"
        exit 1
        ;;
esac

echo ""
echo "=== Tag-Based Verification Complete ==="