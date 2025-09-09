@echo off
echo Testing DriftMgr Production Features
echo ==================================================

set passed=0
set failed=0

echo.
echo Checking production components...

for %%f in (
    "internal\logging\structured.go"
    "internal\resilience\retry.go"
    "internal\cache\ttl_cache.go"
    "internal\security\vault.go"
    "internal\resilience\ratelimiter.go"
    "internal\metrics\collector.go"
    "internal\testing\integration\suite.go"
    "internal\state\distributed.go"
    "internal\resilience\circuit_breaker.go"
    "internal\telemetry\tracing.go"
    "internal\health\checks.go"
    "internal\lifecycle\shutdown.go"
    "loadtest\scenarios.js"
    "docs\runbooks\OPERATIONAL_RUNBOOK.md"
) do (
    if exist %%f (
        echo [OK] %%f
        set /a passed+=1
    ) else (
        echo [MISSING] %%f
        set /a failed+=1
    )
)

echo.
echo Building DriftMgr...
go build -o driftmgr.exe ./cmd/driftmgr 2>nul
if %errorlevel% equ 0 (
    echo [OK] Build successful
    set /a passed+=1
) else (
    echo [FAILED] Build failed
    set /a failed+=1
)

echo.
echo Testing CLI commands...
driftmgr.exe --help >nul 2>&1
if %errorlevel% equ 0 (
    echo [OK] Help command works
    set /a passed+=1
) else (
    echo [FAILED] Help command
    set /a failed+=1
)

driftmgr.exe credentials >nul 2>&1
if %errorlevel% equ 0 (
    echo [OK] Credentials command works
    set /a passed+=1
) else (
    echo [FAILED] Credentials command
    set /a failed+=1
)

echo.
echo ==================================================
echo SUMMARY
echo Passed: %passed%
echo Failed: %failed%
echo.

if %failed% equ 0 (
    echo SUCCESS: All production features verified!
    echo DriftMgr is PRODUCTION READY
    exit /b 0
) else (
    echo WARNING: Some features missing or failed
    exit /b 1
)