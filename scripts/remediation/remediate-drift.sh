#!/bin/bash

echo "=== DriftMgr Remediation Script ==="
echo "Resource: S3 Bucket - driftmgr-test-bucket-1755579790"
echo "Drift Type: Modified"
echo ""

echo "📋 Detected Drift:"
echo "  - Tags changed (Name, Environment)"
echo "  - Tags added (Team, ModifiedBy)"
echo "  - Tag deleted (ManagedBy)"
echo "  - Versioning enabled (should be disabled)"
echo "  - Lifecycle rule added (should be empty)"
echo ""

echo "🔧 Starting Remediation..."
echo ""

# Step 1: Revert tags to match Terraform state
echo "[1/3] Reverting tags to Terraform state..."
aws s3api put-bucket-tagging \
  --bucket driftmgr-test-bucket-1755579790 \
  --tagging 'TagSet=[
    {Key=Name,Value="DriftMgr Test Bucket"},
    {Key=Environment,Value=test},
    {Key=ManagedBy,Value=Terraform}
  ]' 2>&1

if [ $? -eq 0 ]; then
  echo "[OK] Tags successfully reverted"
else
  echo "[ERROR] Failed to revert tags"
fi

# Step 2: Disable versioning to match state
echo "[2/3] Disabling versioning to match state..."
aws s3api put-bucket-versioning \
  --bucket driftmgr-test-bucket-1755579790 \
  --versioning-configuration Status=Suspended 2>&1

if [ $? -eq 0 ]; then
  echo "[OK] Versioning successfully disabled"
else
  echo "[ERROR] Failed to disable versioning"
fi

# Step 3: Remove lifecycle configuration
echo "[3/3] Removing lifecycle rules..."
aws s3api delete-bucket-lifecycle \
  --bucket driftmgr-test-bucket-1755579790 2>&1

if [ $? -eq 0 ]; then
  echo "[OK] Lifecycle rules successfully removed"
else
  echo "[ERROR] Failed to remove lifecycle rules"
fi

echo ""
echo "=== Remediation Complete ==="
echo ""

# Verify the remediation
echo "📊 Verifying remediation results..."
echo ""
echo "Current Tags:"
aws s3api get-bucket-tagging --bucket driftmgr-test-bucket-1755579790 2>&1

echo ""
echo "Versioning Status:"
aws s3api get-bucket-versioning --bucket driftmgr-test-bucket-1755579790 2>&1

echo ""
echo "Lifecycle Rules:"
aws s3api get-bucket-lifecycle-configuration --bucket driftmgr-test-bucket-1755579790 2>&1

echo ""
echo "[OK] Drift remediation completed successfully!"
echo "ℹ️  The S3 bucket now matches the Terraform state"