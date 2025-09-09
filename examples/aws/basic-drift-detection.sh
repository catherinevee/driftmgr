#!/bin/bash

# Basic AWS Drift Detection Example
# This script demonstrates basic drift detection for AWS resources

set -e

echo "========================================="
echo "DriftMgr - AWS Basic Drift Detection Demo"
echo "========================================="
echo ""

# Check if DriftMgr is installed
if ! command -v driftmgr &> /dev/null; then
    echo "Error: driftmgr is not installed or not in PATH"
    echo "Please install from: https://github.com/catherinevee/driftmgr/releases"
    exit 1
fi

# Check AWS credentials
if [ -z "$AWS_ACCESS_KEY_ID" ] && [ ! -f ~/.aws/credentials ]; then
    echo "Error: AWS credentials not configured"
    echo "Please set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables"
    echo "Or configure credentials using: aws configure"
    exit 1
fi

# Configuration
REGION=${AWS_REGION:-us-east-1}
STATE_FILE=${1:-terraform.tfstate}
OUTPUT_FORMAT=${2:-table}

echo "Configuration:"
echo "  Region: $REGION"
echo "  State File: $STATE_FILE"
echo "  Output Format: $OUTPUT_FORMAT"
echo ""

# Step 1: Discover AWS Resources
echo "Step 1: Discovering AWS resources in $REGION..."
echo "----------------------------------------"
driftmgr discover \
    --provider aws \
    --region "$REGION" \
    --output "$OUTPUT_FORMAT" \
    --save-to discovered-resources.json

echo ""
echo "Discovery complete. Results saved to discovered-resources.json"
echo ""

# Step 2: Check if state file exists
if [ ! -f "$STATE_FILE" ]; then
    echo "Warning: State file '$STATE_FILE' not found"
    echo "Skipping drift detection. To detect drift, provide a valid Terraform state file."
    echo ""
    echo "You can:"
    echo "  1. Copy your terraform.tfstate to this directory"
    echo "  2. Specify a different path as the first argument"
    echo "  3. Use remote state with --backend flag"
    exit 0
fi

# Step 3: Detect Drift
echo "Step 2: Detecting drift between state and actual resources..."
echo "----------------------------------------"
driftmgr drift detect \
    --state "$STATE_FILE" \
    --provider aws \
    --region "$REGION" \
    --output "$OUTPUT_FORMAT" \
    --save-report drift-report.json

echo ""
echo "Drift detection complete. Report saved to drift-report.json"
echo ""

# Step 4: Analyze Results
echo "Step 3: Analyzing drift results..."
echo "----------------------------------------"

# Parse JSON report for statistics (requires jq)
if command -v jq &> /dev/null; then
    TOTAL_RESOURCES=$(jq '.summary.total_resources // 0' drift-report.json)
    DRIFTED_RESOURCES=$(jq '.summary.drifted_resources // 0' drift-report.json)
    MISSING_RESOURCES=$(jq '.summary.missing_resources // 0' drift-report.json)
    UNMANAGED_RESOURCES=$(jq '.summary.unmanaged_resources // 0' drift-report.json)
    
    echo "Summary:"
    echo "  Total Resources: $TOTAL_RESOURCES"
    echo "  Drifted Resources: $DRIFTED_RESOURCES"
    echo "  Missing Resources: $MISSING_RESOURCES"
    echo "  Unmanaged Resources: $UNMANAGED_RESOURCES"
    echo ""
    
    if [ "$DRIFTED_RESOURCES" -gt 0 ]; then
        echo "WARNING: Drift detected! Review drift-report.json for details."
        echo ""
        echo "Drifted resources:"
        jq -r '.drifted_resources[]?.resource_id // empty' drift-report.json | head -10
        echo ""
    else
        echo "No drift detected. Infrastructure matches state file."
        echo ""
    fi
else
    echo "Install 'jq' for detailed analysis of results"
    echo "Results are available in drift-report.json"
    echo ""
fi

# Step 5: Generate Remediation (Optional)
if [ "$DRIFTED_RESOURCES" -gt 0 ] || [ "$UNMANAGED_RESOURCES" -gt 0 ]; then
    echo "Step 4: Generating remediation plan..."
    echo "----------------------------------------"
    
    driftmgr remediate \
        --drift-report drift-report.json \
        --output-format terraform \
        --save-to remediation-plan.tf
    
    echo "Remediation plan saved to remediation-plan.tf"
    echo ""
    echo "Review the plan and apply with:"
    echo "  terraform plan -out=tfplan"
    echo "  terraform apply tfplan"
    echo ""
fi

echo "========================================="
echo "Demo Complete!"
echo "========================================="
echo ""
echo "Generated files:"
echo "  - discovered-resources.json : All discovered AWS resources"
echo "  - drift-report.json : Detailed drift analysis"
[ -f remediation-plan.tf ] && echo "  - remediation-plan.tf : Terraform import commands"
echo ""
echo "Next steps:"
echo "  1. Review the drift report for detailed findings"
echo "  2. Apply remediation if needed"
echo "  3. Set up continuous monitoring with 'driftmgr monitor'"
echo ""
echo "For more examples, visit:"
echo "  https://github.com/catherinevee/driftmgr/tree/main/examples"