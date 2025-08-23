@echo off
echo.
echo ================================
echo      DriftMgr Installation
echo ================================
echo.
echo Installing DriftMgr to C:\Users\%USERNAME%\
echo.

REM Copy the main binary
copy /Y "driftmgr.exe" "C:\Users\%USERNAME%\driftmgr.exe" >nul
if %errorlevel% neq 0 (
    echo [ERROR] Failed to copy driftmgr.exe
    pause
    exit /b 1
)

echo [OK] DriftMgr installed successfully!
echo.
echo ================================
echo        Quick Test
echo ================================
echo.
echo Testing installation...
C:\Users\%USERNAME%\driftmgr.exe --help >nul 2>&1
if %errorlevel% equ 0 (
    echo [OK] DriftMgr is working correctly!
) else (
    echo [ERROR] DriftMgr test failed
)

echo.
echo ================================
echo       Usage Instructions
echo ================================
echo.
echo You can now run DriftMgr from anywhere using:
echo.
echo   driftmgr --provider gcp --format summary
echo   driftmgr --provider gcp --cost-analysis --export html
echo.
echo Make sure C:\Users\%USERNAME% is in your PATH, or run:
echo   C:\Users\%USERNAME%\driftmgr.exe [options]
echo.
echo For the quick start guide, run: quick_start.bat
echo.
pause