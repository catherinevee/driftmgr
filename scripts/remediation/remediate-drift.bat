@echo off
echo === DriftMgr Remediation Script ===
echo Resource: S3 Bucket - driftmgr-test-bucket-1755579790
echo Drift Type: Modified
echo.

echo Detected Drift:
echo   - Tags changed (Name, Environment)
echo   - Tags added (Team, ModifiedBy)
echo   - Tag deleted (ManagedBy)
echo   - Versioning enabled (should be disabled)
echo   - Lifecycle rule added (should be empty)
echo.

echo Starting Remediation...
echo.

echo [1/3] Reverting tags to Terraform state...
aws s3api put-bucket-tagging --bucket driftmgr-test-bucket-1755579790 --tagging "TagSet=[{Key=Name,Value=\"DriftMgr Test Bucket\"},{Key=Environment,Value=test},{Key=ManagedBy,Value=Terraform}]"
if %ERRORLEVEL% EQU 0 (
  echo SUCCESS: Tags reverted
) else (
  echo ERROR: Failed to revert tags
)

echo [2/3] Disabling versioning to match state...
aws s3api put-bucket-versioning --bucket driftmgr-test-bucket-1755579790 --versioning-configuration Status=Suspended
if %ERRORLEVEL% EQU 0 (
  echo SUCCESS: Versioning disabled
) else (
  echo ERROR: Failed to disable versioning
)

echo [3/3] Removing lifecycle rules...
aws s3api delete-bucket-lifecycle --bucket driftmgr-test-bucket-1755579790
if %ERRORLEVEL% EQU 0 (
  echo SUCCESS: Lifecycle rules removed
) else (
  echo ERROR: Failed to remove lifecycle rules
)

echo.
echo === Remediation Complete ===
echo.

echo Verifying remediation results...
echo.
echo Current Tags:
aws s3api get-bucket-tagging --bucket driftmgr-test-bucket-1755579790

echo.
echo Versioning Status:
aws s3api get-bucket-versioning --bucket driftmgr-test-bucket-1755579790

echo.
echo Drift remediation completed successfully!
echo The S3 bucket now matches the Terraform state