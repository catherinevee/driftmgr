@echo off
REM DriftMgr CLI Build Script for Windows
REM This script builds the CLI with all features

echo ==========================================
echo Building DriftMgr CLI
echo ==========================================

REM Check if Go is installed
go version >nul 2>&1
if errorlevel 1 (
    echo Error: Go is not installed or not in PATH
    exit /b 1
)

echo Go version:
go version
echo.

REM Set build variables
set CLI_DIR=cmd\driftmgr-client
set OUTPUT_NAME=driftmgr-client.exe

REM Check if all required files exist
echo Checking required files...
if exist "%CLI_DIR%\main.go" (
    echo ✓ %CLI_DIR%\main.go
) else (
    echo ✗ %CLI_DIR%\main.go (missing)
    exit /b 1
)

if exist "%CLI_DIR%\completion.go" (
    echo ✓ %CLI_DIR%\completion.go
) else (
    echo ✗ %CLI_DIR%\completion.go (missing)
    exit /b 1
)

if exist "%CLI_DIR%\enhanced_analyze.go" (
    echo ✓ %CLI_DIR%\enhanced_analyze.go
) else (
    echo ✗ %CLI_DIR%\enhanced_analyze.go (missing)
    exit /b 1
)

if exist "%CLI_DIR%\remediate.go" (
    echo ✓ %CLI_DIR%\remediate.go
) else (
    echo ✗ %CLI_DIR%\remediate.go (missing)
    exit /b 1
)

if exist "%CLI_DIR%\credentials.go" (
    echo ✓ %CLI_DIR%\credentials.go
) else (
    echo ✗ %CLI_DIR%\credentials.go (missing)
    exit /b 1
)
echo.

REM Build the CLI
echo Building CLI...
go build -o %OUTPUT_NAME% %CLI_DIR%\main.go %CLI_DIR%\completion.go %CLI_DIR%\enhanced_analyze.go %CLI_DIR%\remediate.go %CLI_DIR%\credentials.go

if errorlevel 1 (
    echo ✗ Build failed!
    exit /b 1
)

echo ✓ CLI built successfully!
echo Output: %OUTPUT_NAME%

REM Show file size
if exist "%OUTPUT_NAME%" (
    echo File size:
    dir %OUTPUT_NAME% | findstr "%OUTPUT_NAME%"
)

echo.
echo ==========================================
echo Build Complete!
echo ==========================================
echo.
echo To test the CLI:
echo   %OUTPUT_NAME%
echo.
echo For documentation:
echo   docs\cli\enhanced-features-guide.md
echo.
echo Features:
echo   ✓ Tab completion
echo   ✓ Auto-suggestions
echo   ✓ Fuzzy search
echo   ✓ Arrow key navigation
echo   ✓ Context-aware completion
