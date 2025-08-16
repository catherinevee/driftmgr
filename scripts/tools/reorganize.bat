@echo off
REM DriftMgr Directory Reorganization Script for Windows
REM This script helps reorganize the project structure for better maintainability

echo ==========================================
echo DriftMgr Directory Reorganization
echo ==========================================

REM Function to create directory if it doesn't exist
:create_dir
if not exist "%~1" (
    echo Creating directory: %~1
    mkdir "%~1"
)
goto :eof

REM Function to move file with backup
:move_file
if exist "%~1" (
    echo Moving: %~1 -^> %~2
    REM Create backup
    copy "%~1" "%~1.backup"
    REM Move file
    move "%~1" "%~2"
) else (
    echo Warning: Source file not found: %~1
)
goto :eof

echo Phase 1: Creating new directory structure

REM Create main directories
call :create_dir "web\static\css"
call :create_dir "web\static\js"
call :create_dir "web\static\images"
call :create_dir "web\templates"

call :create_dir "configs\environments"

call :create_dir "scripts\build"
call :create_dir "scripts\install"
call :create_dir "scripts\deploy"
call :create_dir "scripts\tools"

call :create_dir "docs\api"
call :create_dir "docs\cli"
call :create_dir "docs\web"
call :create_dir "docs\deployment"
call :create_dir "docs\development"

call :create_dir "examples\basic"
call :create_dir "examples\advanced"
call :create_dir "examples\demos"

call :create_dir "tests\unit"
call :create_dir "tests\integration"
call :create_dir "tests\e2e"
call :create_dir "tests\fixtures\state-files"
call :create_dir "tests\fixtures\configs"

call :create_dir "deployments\docker"
call :create_dir "deployments\kubernetes"
call :create_dir "deployments\terraform"

call :create_dir "tools\codegen"
call :create_dir "tools\migrations"
call :create_dir "tools\benchmarks"

call :create_dir "assets\images"
call :create_dir "assets\logos"
call :create_dir "assets\samples\state-files"
call :create_dir "assets\samples\configs"

echo Phase 2: Moving and organizing files

REM Move documentation
echo Organizing documentation...
if exist "docs\cli\enhanced-features-guide.md" (
    call :move_file "docs\cli\enhanced-features-guide.md" "docs\cli\features.md"
)

REM Move scripts
echo Organizing scripts...
if exist "scripts\build-enhanced-cli.sh" (
    call :move_file "scripts\build-enhanced-cli.sh" "scripts\build\build-client.sh"
)

if exist "scripts\build-enhanced-cli.bat" (
    call :move_file "scripts\build-enhanced-cli.bat" "scripts\build\build-client.bat"
)

if exist "scripts\demo-enhanced-cli.sh" (
    call :move_file "scripts\demo-enhanced-cli.sh" "scripts\tools\demo-cli.sh"
)

REM Move installation scripts
if exist "install.sh" (
    call :move_file "install.sh" "scripts\install\install.sh"
)

if exist "install.ps1" (
    call :move_file "install.ps1" "scripts\install\install.ps1"
)

REM Move configuration files
echo Organizing configuration files...
if exist "driftmgr.yaml" (
    call :move_file "driftmgr.yaml" "configs\driftmgr.yaml"
)

if exist "driftmgr.yaml.example" (
    call :move_file "driftmgr.yaml.example" "configs\driftmgr.yaml.example"
)

REM Move deployment files
echo Organizing deployment files...
if exist "Dockerfile" (
    call :move_file "Dockerfile" "deployments\docker\Dockerfile"
)

if exist "docker-compose.yml" (
    call :move_file "docker-compose.yml" "deployments\docker\docker-compose.yml"
)

if exist ".dockerignore" (
    call :move_file ".dockerignore" "deployments\docker\.dockerignore"
)

REM Move assets
echo Organizing assets...
if exist "test_state_file.json" (
    call :move_file "test_state_file.json" "assets\samples\state-files\test_state_file.json"
)

REM Move implementation summary
if exist "ENHANCED_CLI_IMPLEMENTATION_SUMMARY.md" (
    call :move_file "ENHANCED_CLI_IMPLEMENTATION_SUMMARY.md" "docs\development\cli-implementation-summary.md"
)

echo Phase 3: Creating new documentation structure

REM Create main documentation index
echo # DriftMgr Documentation > docs\README.md
echo. >> docs\README.md
echo This directory contains comprehensive documentation for the DriftMgr project. >> docs\README.md
echo. >> docs\README.md
echo ## Documentation Index >> docs\README.md
echo. >> docs\README.md
echo ### User Documentation >> docs\README.md
echo - **[CLI Documentation](cli/)** - Command-line interface guide >> docs\README.md
echo - **[Web Interface](web/)** - Web dashboard documentation >> docs\README.md
echo - **[API Reference](api/)** - REST API documentation >> docs\README.md

echo Phase 4: Creating build scripts

REM Create main build script
echo @echo off > scripts\build\build-all.bat
echo REM DriftMgr Build All Script for Windows >> scripts\build\build-all.bat
echo REM Builds both server and client applications >> scripts\build\build-all.bat
echo. >> scripts\build\build-all.bat
echo echo ========================================== >> scripts\build\build-all.bat
echo echo Building DriftMgr Applications >> scripts\build\build-all.bat
echo echo ========================================== >> scripts\build\build-all.bat
echo. >> scripts\build\build-all.bat
echo REM Check if Go is installed >> scripts\build\build-all.bat
echo go version ^>nul 2^>^&1 >> scripts\build\build-all.bat
echo if errorlevel 1 ( >> scripts\build\build-all.bat
echo     echo Error: Go is not installed or not in PATH >> scripts\build\build-all.bat
echo     exit /b 1 >> scripts\build\build-all.bat
echo ) >> scripts\build\build-all.bat
echo. >> scripts\build\build-all.bat
echo echo Go version: >> scripts\build\build-all.bat
echo go version >> scripts\build\build-all.bat
echo echo. >> scripts\build\build-all.bat
echo. >> scripts\build\build-all.bat
echo REM Build server >> scripts\build\build-all.bat
echo echo Building server... >> scripts\build\build-all.bat
echo go build -o driftmgr-server.exe cmd\driftmgr-server\*.go >> scripts\build\build-all.bat
echo if errorlevel 1 ( >> scripts\build\build-all.bat
echo     echo ✗ Server build failed! >> scripts\build\build-all.bat
echo     exit /b 1 >> scripts\build\build-all.bat
echo ) >> scripts\build\build-all.bat
echo echo ✓ Server built successfully! >> scripts\build\build-all.bat
echo. >> scripts\build\build-all.bat
echo REM Build client >> scripts\build\build-all.bat
echo echo Building client... >> scripts\build\build-all.bat
echo go build -o driftmgr-client.exe cmd\driftmgr-client\*.go >> scripts\build\build-all.bat
echo if errorlevel 1 ( >> scripts\build\build-all.bat
echo     echo ✗ Client build failed! >> scripts\build\build-all.bat
echo     exit /b 1 >> scripts\build\build-all.bat
echo ) >> scripts\build\build-all.bat
echo echo ✓ Client built successfully! >> scripts\build\build-all.bat

echo Phase 5: Creating Makefile

REM Create Makefile
echo # DriftMgr Makefile > Makefile
echo # Build automation for DriftMgr project >> Makefile
echo. >> Makefile
echo .PHONY: help build build-server build-client clean test install >> Makefile
echo. >> Makefile
echo # Default target >> Makefile
echo help: >> Makefile
echo 	@echo "DriftMgr Build Targets:" >> Makefile
echo 	@echo "  build        - Build both server and client" >> Makefile
echo 	@echo "  build-server - Build server application only" >> Makefile
echo 	@echo "  build-client - Build client application only" >> Makefile
echo 	@echo "  clean        - Clean build artifacts" >> Makefile
echo 	@echo "  test         - Run tests" >> Makefile
echo 	@echo "  install      - Install dependencies" >> Makefile
echo 	@echo "  help         - Show this help message" >> Makefile
echo. >> Makefile
echo # Build both applications >> Makefile
echo build: build-server build-client >> Makefile
echo. >> Makefile
echo # Build server application >> Makefile
echo build-server: >> Makefile
echo 	@echo "Building server application..." >> Makefile
echo 	go build -o driftmgr-server.exe cmd/driftmgr-server/*.go >> Makefile
echo 	@echo "✓ Server built successfully!" >> Makefile
echo. >> Makefile
echo # Build client application >> Makefile
echo build-client: >> Makefile
echo 	@echo "Building client application..." >> Makefile
echo 	go build -o driftmgr-client.exe cmd/driftmgr-client/*.go >> Makefile
echo 	@echo "✓ Client built successfully!" >> Makefile

echo Phase 6: Updating .gitignore

REM Update .gitignore
echo. >> .gitignore
echo # Build artifacts >> .gitignore
echo *.exe >> .gitignore
echo *.dll >> .gitignore
echo *.so >> .gitignore
echo *.dylib >> .gitignore
echo. >> .gitignore
echo # Backup files >> .gitignore
echo *.backup >> .gitignore
echo. >> .gitignore
echo # IDE files >> .gitignore
echo .vscode/ >> .gitignore
echo .idea/ >> .gitignore
echo *.swp >> .gitignore
echo *.swo >> .gitignore
echo. >> .gitignore
echo # OS files >> .gitignore
echo .DS_Store >> .gitignore
echo Thumbs.db >> .gitignore
echo. >> .gitignore
echo # Log files >> .gitignore
echo *.log >> .gitignore
echo. >> .gitignore
echo # Environment files >> .gitignore
echo .env >> .gitignore
echo .env.local >> .gitignore
echo. >> .gitignore
echo # Test coverage >> .gitignore
echo coverage.out >> .gitignore
echo coverage.html >> .gitignore
echo. >> .gitignore
echo # Temporary files >> .gitignore
echo tmp/ >> .gitignore
echo temp/ >> .gitignore

echo.
echo ==========================================
echo Reorganization Complete!
echo ==========================================
echo.
echo Next Steps:
echo 1. Review the new directory structure
echo 2. Update import statements in Go files
echo 3. Update documentation links
echo 4. Test all functionality
echo 5. Update CI/CD configurations
echo.
echo Backup files have been created with .backup extension
echo You can safely delete them after verifying everything works
echo.
echo New structure created:
echo   ✓ Organized documentation in docs/
echo   ✓ Organized scripts in scripts/
echo   ✓ Organized examples in examples/
echo   ✓ Organized tests in tests/
echo   ✓ Organized configs in configs/
echo   ✓ Organized deployments in deployments/
echo   ✓ Created build automation with Makefile
echo   ✓ Updated .gitignore for better artifact management
