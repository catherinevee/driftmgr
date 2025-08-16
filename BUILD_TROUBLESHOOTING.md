# DriftMgr Build Troubleshooting Guide

## Problem: "driftmgr" command doesn't work or shows no output

### Root Cause
The main `driftmgr.exe` executable depends on `driftmgr-client.exe` to function properly. If the client binary is missing, the main executable will appear to do nothing.

### Solution
1. **Rebuild the project:**
   ```cmd
   make build
   ```

2. **Verify all binaries are present:**
   ```cmd
   powershell -ExecutionPolicy Bypass -File scripts/build/verify-build.ps1
   ```

3. **Check that these files exist in the `bin/` directory:**
   - `driftmgr.exe` (main executable)
   - `driftmgr-client.exe` (required for CLI functionality)
   - `driftmgr-server.exe` (optional, for web interface)

### Prevention
The build scripts have been updated to automatically build all required components:
- **Windows**: `scripts/build/build.ps1` now includes client build
- **Linux/Mac**: `scripts/build/build.sh` now includes client build
- **Makefile**: `make build` now includes verification step

### Manual Fix
If you need to build just the client:
```cmd
go build -o bin/driftmgr-client.exe ./cmd/driftmgr-client
```

## Testing the Installation
After building, test that `driftmgr` works from any directory:
```cmd
cd C:\Users\yourname\Desktop
driftmgr
```

You should see the DriftMgr banner and interactive shell prompt.

## Common Issues

### Issue: "driftmgr-client.exe not found"
**Solution**: Run `make build` to rebuild all components

### Issue: Command hangs or shows no output
**Solution**: Check that `driftmgr-client.exe` exists in the `bin/` directory

### Issue: Permission denied
**Solution**: Run PowerShell as Administrator or check file permissions

### Issue: PATH not set correctly
**Solution**: Ensure the `bin/` directory is in your system PATH
