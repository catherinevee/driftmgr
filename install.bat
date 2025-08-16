@echo off
REM DriftMgr Windows Installer Launcher
REM This batch file launches the PowerShell installer

setlocal enabledelayedexpansion

echo.
echo ========================================
echo    DriftMgr Windows Installer
echo ========================================
echo.

REM Check if PowerShell is available
where powershell >nul 2>&1
if %errorlevel% neq 0 (
    where pwsh >nul 2>&1
    if %errorlevel% neq 0 (
        echo ERROR: PowerShell not found.
        echo Please install PowerShell Core or Windows PowerShell.
        echo Download from: https://docs.microsoft.com/en-us/powershell/scripting/install/installing-powershell
        pause
        exit /b 1
    ) else (
        set "PS_EXEC=pwsh"
    )
) else (
    set "PS_EXEC=powershell"
)

REM Get the directory where this batch file is located
set "SCRIPT_DIR=%~dp0"
set "INSTALLER_PATH=%SCRIPT_DIR%installer\windows\install.ps1"

REM Check if installer exists
if not exist "%INSTALLER_PATH%" (
    echo ERROR: Installer not found at %INSTALLER_PATH%
    echo Make sure you're running this from the DriftMgr project root.
    pause
    exit /b 1
)

REM Check if binaries exist
if not exist "%SCRIPT_DIR%bin\driftmgr.exe" (
    echo ERROR: DriftMgr executable not found.
    echo Please build the application first by running 'make build' or 'go build'
    pause
    exit /b 1
)

echo Starting DriftMgr installation...
echo.

REM Run the PowerShell installer
%PS_EXEC% -ExecutionPolicy Bypass -File "%INSTALLER_PATH%" %*

if %errorlevel% neq 0 (
    echo.
    echo ERROR: Installation failed with error code %errorlevel%
    pause
    exit /b %errorlevel%
)

echo.
echo Installation completed successfully!
echo.
echo Next steps:
echo 1. Open a new command prompt to refresh PATH
echo 2. Run 'driftmgr --help' to see available commands
echo 3. Run 'driftmgr-server' to start the web dashboard
echo.
pause
