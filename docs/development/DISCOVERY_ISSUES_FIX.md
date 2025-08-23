# DriftMgr Discovery Issues and Fixes

## Problem Summary
DriftMgr wasn't displaying all resources from cloud accounts due to several issues in the discovery process.

## Root Causes Identified

### 1. Missing AWS Services in Discovery
**Issue**: Many AWS services were implemented but not being called in the main discovery function.

**Services Missing from Discovery**:
- AWS Athena
- AWS Kinesis
- AWS Data Pipeline
- AWS QuickSight
- AWS DataSync
- AWS Storage Gateway
- AWS Backup
- AWS FSx
- AWS WorkSpaces
- AWS AppStream

**Fix**: Added all missing services to the `discoverAWSEnhanced` function in `driftmgr/internal/discovery/enhanced_discovery.go`.

### 2. Overly Strict Credential Validation
**Issue**: The discovery process was skipping entire providers if credential validation failed, even for services that might work with partial permissions.

**Fix**: Modified credential validation to be less strict:
- Discovery continues even if credential validation fails
- Added detailed logging for credential issues
- Individual service failures are logged but don't stop the entire discovery process

### 3. Insufficient Timeout Configuration
**Issue**: The default 5-minute timeout was too short for complete discovery across multiple services and regions.

**Fix**: Increased timeout and added better configuration:
- Increased discovery timeout from 5m to 10m
- Added API timeout configuration (30s per call)
- Added concurrent region limit (5 regions at once)

### 4. Poor Error Handling and Logging
**Issue**: Many discovery errors were being silently ignored, making it difficult to diagnose issues.

**Fix**: Improved error handling and logging:
- Added detailed error logging for each service
- Separated credential errors from permission errors
- Added success logging for each region/account
- Changed logging level to debug for more detailed information

### 5. Missing Configuration for Complete Discovery
**Issue**: Default filters and quality thresholds were not configured for maximum resource discovery.

**Fix**: Added complete configuration:
- Empty resource type filters (discover all types)
- No age threshold filtering
- Quality thresholds for completeness and accuracy
- Debug logging enabled

## Files Modified

### 1. `driftmgr/internal/discovery/enhanced_discovery.go`
- Added missing AWS services to discovery function
- Improved error handling in EC2 discovery
- Modified credential validation to be less strict
- Added detailed logging throughout discovery process

### 2. `driftmgr/configs/config.yaml`
- Increased discovery timeout from 5m to 10m
- Added API timeout configuration
- Added concurrent region limit
- Added quality thresholds
- Added default filters (empty for maximum discovery)
- Changed logging level to debug

### 3. `driftmgr/scripts/test_discovery.sh` (Linux/Mac)
- Created diagnostic script to test individual AWS services
- Helps identify which services are working and which are failing

### 4. `driftmgr/scripts/test_discovery.ps1` (Windows)
- PowerShell version of diagnostic script
- Tests AWS CLI availability and individual services

## How to Use the Fixes

### 1. Run the Diagnostic Script
First, run the diagnostic script to identify any remaining issues:

**Windows:**
```powershell
.\driftmgr\scripts\test_discovery.ps1
```

**Linux/Mac:**
```bash
./driftmgr/scripts/test_discovery.sh
```

### 2. Check Logs
With debug logging enabled, you'll now see detailed information about:
- Which services are being discovered
- Any errors that occur during discovery
- Success counts for each region/account
- Credential and permission issues

### 3. Monitor Discovery Progress
The enhanced discovery now provides:
- Better progress tracking
- Detailed error reporting
- Success logging for each step
- Information about skipped services

## Expected Improvements

After applying these fixes, you should see:

1. **More Resources Discovered**: All implemented AWS services will now be checked
2. **Better Error Visibility**: Detailed logs will show exactly what's working and what's not
3. **Improved Reliability**: Discovery continues even if some services fail
4. **Faster Discovery**: Better timeout and concurrency settings
5. **Better Debugging**: Diagnostic scripts help identify permission issues

## Troubleshooting

If you're still not seeing all resources:

1. **Check IAM Permissions**: Use the diagnostic script to see which services are failing
2. **Review Logs**: Look for detailed error messages in the debug logs
3. **Verify Credentials**: Ensure AWS CLI is properly configured
4. **Check Regions**: Make sure you're checking the right regions for your resources
5. **Monitor Timeouts**: If discovery is timing out, increase the timeout further

## Additional Recommendations

1. **IAM Permissions**: Ensure your AWS credentials have permissions for all services you want to discover
2. **Region Coverage**: Make sure you're checking all regions where your resources exist
3. **Service-Specific Permissions**: Some services require specific IAM permissions (e.g., IAM for IAM resources)
4. **Rate Limiting**: If you have many resources, consider increasing the concurrency limits

## Future Improvements

Consider implementing:
1. **Parallel Service Discovery**: Discover multiple services simultaneously
2. **Incremental Discovery**: Only check for changes since last discovery
3. **Service-Specific Timeouts**: Different timeouts for different service types
4. **Retry Logic**: Automatic retry for failed service discoveries
5. **Resource Filtering**: Allow users to specify which services to discover
