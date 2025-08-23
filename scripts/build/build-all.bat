@echo off
REM DriftMgr Universal Build Script for Windows
REM This script builds both server and client applications for Windows

setlocal enabledelayedexpansion

REM Configuration
set PROJECT_NAME=DriftMgr
set SERVER_BINARY=driftmgr-server.exe
set CLIENT_BINARY=driftmgr-client.exe
set BUILD_DIR=.
set VERSION=dev

REM Color definitions (Windows 10+)
set RED=[91m
set GREEN=[92m
set YELLOW=[93m
set BLUE=[94m
set NC=[0m

REM Logging functions
:log_info
echo %BLUE%[INFO]%NC% %~1
goto :eof

:log_success
echo %GREEN%[SUCCESS]%NC% %~1
goto :eof

:log_warning
echo %YELLOW%[WARNING]%NC% %~1
goto :eof

:log_error
echo %RED%[ERROR]%NC% %~1
goto :eof

REM Check prerequisites
:check_prerequisites
call :log_info "Checking prerequisites..."

REM Check if Go is installed
go version >nul 2>&1
if errorlevel 1 (
    call :log_error "Go is not installed or not in PATH"
    exit /b 1
)

REM Get Go version
for /f "tokens=3" %%i in ('go version') do set GO_VERSION=%%i
call :log_info "Go version: %GO_VERSION%"

REM Check if we're in the right directory
if not exist "go.mod" (
    call :log_error "go.mod not found. Please run this script from the project root."
    exit /b 1
)

call :log_success "Prerequisites check passed"
goto :eof

REM Clean previous builds
:clean_builds
call :log_info "Cleaning previous builds..."
if exist "%SERVER_BINARY%" del "%SERVER_BINARY%"
if exist "%CLIENT_BINARY%" del "%CLIENT_BINARY%"
if exist "driftmgr-server" del "driftmgr-server"
if exist "driftmgr-client" del "driftmgr-client"
call :log_success "Clean complete"
goto :eof

REM Install dependencies
:install_dependencies
call :log_info "Installing dependencies..."
go mod download
go mod tidy
call :log_success "Dependencies installed"
goto :eof

REM Build server
:build_server
call :log_info "Building server application..."
go build -ldflags "-X main.version=%VERSION%" -o "%SERVER_BINARY%" ./cmd/driftmgr-server
if errorlevel 1 (
    call :log_error "Server build failed"
    exit /b 1
)
call :log_success "Server built successfully: %SERVER_BINARY%"
goto :eof

REM Build client
:build_client
call :log_info "Building client application..."
go build -ldflags "-X main.version=%VERSION%" -o "%CLIENT_BINARY%" ./cmd/driftmgr-client
if errorlevel 1 (
    call :log_error "Client build failed"
    exit /b 1
)
call :log_success "Client built successfully: %CLIENT_BINARY%"
goto :eof

REM Build for specific platform
:build_for_platform
set platform=%~1
set arch=%~2
if "%arch%"=="" set arch=amd64

call :log_info "Building for %platform%/%arch%..."

if "%platform%"=="windows" (
    set GOOS=windows
    set GOARCH=%arch%
    go build -ldflags "-X main.version=%VERSION%" -o "driftmgr-server-windows-%arch%.exe" ./cmd/driftmgr-server
    go build -ldflags "-X main.version=%VERSION%" -o "driftmgr-client-windows-%arch%.exe" ./cmd/driftmgr-client
) else if "%platform%"=="linux" (
    set GOOS=linux
    set GOARCH=%arch%
    go build -ldflags "-X main.version=%VERSION%" -o "driftmgr-server-linux-%arch%" ./cmd/driftmgr-server
    go build -ldflags "-X main.version=%VERSION%" -o "driftmgr-client-linux-%arch%" ./cmd/driftmgr-client
) else if "%platform%"=="darwin" (
    set GOOS=darwin
    set GOARCH=%arch%
    go build -ldflags "-X main.version=%VERSION%" -o "driftmgr-server-darwin-%arch%" ./cmd/driftmgr-server
    go build -ldflags "-X main.version=%VERSION%" -o "driftmgr-client-darwin-%arch%" ./cmd/driftmgr-client
) else (
    call :log_error "Unsupported platform: %platform%"
    exit /b 1
)

call :log_success "Build for %platform%/%arch% completed"
goto :eof

REM Run tests
:run_tests
call :log_info "Running tests..."
go test -v ./...
call :log_success "Tests completed"
goto :eof

REM Run linting
:run_lint
call :log_info "Running linter..."
golangci-lint run >nul 2>&1
if errorlevel 1 (
    call :log_warning "golangci-lint not found, skipping linting"
) else (
    call :log_success "Linting completed"
)
goto :eof

REM Show build info
:show_build_info
call :log_info "Build Information:"
echo   Project: %PROJECT_NAME%
echo   Version: %VERSION%
echo   Platform: Windows
echo   Go version: %GO_VERSION%
echo   Build time: %date% %time%
goto :eof

REM Show usage
:show_usage
echo Usage: %~nx0 [OPTIONS]
echo.
echo Options:
echo   --clean              Clean previous builds before building
echo   --deps               Install dependencies
echo   --server             Build server only
echo   --client             Build client only
echo   --test               Run tests
echo   --lint               Run linter
echo   --platform PLATFORM  Build for specific platform (windows, linux, darwin)
echo   --arch ARCH          Architecture (amd64, arm64, 386) [default: amd64]
echo   --all-platforms      Build for all platforms
echo   --help               Show this help message
echo.
echo Examples:
echo   %~nx0                    # Build for current platform
echo   %~nx0 --clean --test     # Clean, build, and test
echo   %~nx0 --platform linux   # Build for Linux
echo   %~nx0 --all-platforms    # Build for all platforms
goto :eof

REM Main build function
:main_build
set clean=false
set deps=false
set server_only=false
set client_only=false
set run_tests_flag=false
set run_lint_flag=false
set platform=
set arch=amd64
set all_platforms=false

REM Parse command line arguments
:parse_args
if "%~1"=="" goto :build_start
if "%~1"=="--clean" (
    set clean=true
    shift
    goto :parse_args
)
if "%~1"=="--deps" (
    set deps=true
    shift
    goto :parse_args
)
if "%~1"=="--server" (
    set server_only=true
    shift
    goto :parse_args
)
if "%~1"=="--client" (
    set client_only=true
    shift
    goto :parse_args
)
if "%~1"=="--test" (
    set run_tests_flag=true
    shift
    goto :parse_args
)
if "%~1"=="--lint" (
    set run_lint_flag=true
    shift
    goto :parse_args
)
if "%~1"=="--platform" (
    set platform=%~2
    shift
    shift
    goto :parse_args
)
if "%~1"=="--arch" (
    set arch=%~2
    shift
    shift
    goto :parse_args
)
if "%~1"=="--all-platforms" (
    set all_platforms=true
    shift
    goto :parse_args
)
if "%~1"=="--help" (
    call :show_usage
    exit /b 0
)
call :log_error "Unknown option: %~1"
call :show_usage
exit /b 1

:build_start
REM Show build information
call :show_build_info
echo.

REM Check prerequisites
call :check_prerequisites

REM Install dependencies if requested
if "%deps%"=="true" (
    call :install_dependencies
)

REM Clean if requested
if "%clean%"=="true" (
    call :clean_builds
)

REM Build for specific platform
if not "%platform%"=="" (
    call :build_for_platform "%platform%" "%arch%"
    exit /b 0
)

REM Build for all platforms
if "%all_platforms%"=="true" (
    call :log_info "Building for all platforms..."
    call :build_for_platform "windows" "amd64"
    call :build_for_platform "linux" "amd64"
    call :build_for_platform "darwin" "amd64"
    call :build_for_platform "darwin" "arm64"
    call :log_success "All platform builds completed"
    exit /b 0
)

REM Build applications
if "%server_only%"=="true" (
    call :build_server
) else if "%client_only%"=="true" (
    call :build_client
) else (
    call :build_server
    call :build_client
)

REM Run tests if requested
if "%run_tests_flag%"=="true" (
    call :run_tests
)

REM Run linting if requested
if "%run_lint_flag%"=="true" (
    call :run_lint
)

REM Show results
echo.
call :log_success "Build completed successfully!"
echo.
call :log_info "Built binaries:"
if exist "%SERVER_BINARY%" (
    echo   Server: %SERVER_BINARY%
)
if exist "%CLIENT_BINARY%" (
    echo   Client: %CLIENT_BINARY%
)
echo.
call :log_info "To run the applications:"
echo   Server: %SERVER_BINARY%
echo   Client: %CLIENT_BINARY%

exit /b 0

REM Run main function with all arguments
call :main_build %*
