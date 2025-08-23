@echo off
REM DriftMgr Windows Installation Script (Batch)
REM This script installs driftmgr.exe to make it accessible from anywhere

echo.
echo ==========================================
echo    DriftMgr Installation for Windows
echo ==========================================
echo.

REM Check if driftmgr.exe exists
if not exist "%~dp0driftmgr.exe" (
    echo ERROR: driftmgr.exe not found in current directory!
    echo Please build the application first with: go build -o driftmgr.exe ./cmd/driftmgr
    pause
    exit /b 1
)

REM Set installation directory
set "INSTALL_DIR=%LOCALAPPDATA%\DriftMgr"
set "EXE_PATH=%INSTALL_DIR%\driftmgr.exe"

REM Create installation directory
echo Creating installation directory: %INSTALL_DIR%
if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%"

REM Copy driftmgr.exe
echo Installing driftmgr.exe to: %EXE_PATH%
copy /Y "%~dp0driftmgr.exe" "%EXE_PATH%" >nul

if %ERRORLEVEL% neq 0 (
    echo ERROR: Failed to copy driftmgr.exe!
    pause
    exit /b 1
)

REM Add to PATH using setx (permanent change)
echo Adding DriftMgr to PATH...
setx PATH "%PATH%;%INSTALL_DIR%" >nul 2>&1

REM Also update current session PATH
set "PATH=%PATH%;%INSTALL_DIR%"

echo.
echo ========================================
echo    Installation Complete!
echo ========================================
echo.
echo DriftMgr has been installed to: %EXE_PATH%
echo.
echo IMPORTANT: 
echo   - Close and reopen your terminal for PATH changes to take effect
echo   - Or use 'refreshenv' if you have Chocolatey installed
echo.
echo To use DriftMgr:
echo   1. Open a new terminal (CMD/PowerShell)
echo   2. Type: driftmgr
echo.
echo Available commands:
echo   driftmgr              - Launch interactive TUI
echo   driftmgr --enhanced   - Launch enhanced TUI  
echo   driftmgr --help       - Show help
echo.
echo To uninstall: Delete %INSTALL_DIR% and remove from PATH
echo.
pause