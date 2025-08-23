#!/bin/bash
# DriftMgr Universal Build Script
# This script builds both server and client applications for Unix/Linux/macOS

set -e

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_NAME="DriftMgr"
SERVER_BINARY="driftmgr-server"
CLIENT_BINARY="driftmgr-client"
BUILD_DIR="."
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Error handling
handle_error() {
    log_error "Build failed at line $1"
    exit 1
}

trap 'handle_error $LINENO' ERR

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    # Check Go version
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    log_info "Go version: $GO_VERSION"
    
    # Check if we're in the right directory
    if [ ! -f "go.mod" ]; then
        log_error "go.mod not found. Please run this script from the project root."
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Clean previous builds
clean_builds() {
    log_info "Cleaning previous builds..."
    rm -f "$SERVER_BINARY" "$CLIENT_BINARY"
    rm -f "$SERVER_BINARY.exe" "$CLIENT_BINARY.exe"
    log_success "Clean complete"
}

# Install dependencies
install_dependencies() {
    log_info "Installing dependencies..."
    go mod download
    go mod tidy
    log_success "Dependencies installed"
}

# Build server
build_server() {
    log_info "Building server application..."
    
    # Build for current platform
    go build -ldflags "-X main.version=$VERSION" -o "$SERVER_BINARY" ./cmd/driftmgr-server
    
    if [ $? -eq 0 ]; then
        log_success "Server built successfully: $SERVER_BINARY"
    else
        log_error "Server build failed"
        exit 1
    fi
}

# Build client
build_client() {
    log_info "Building client application..."
    
    # Build for current platform
    go build -ldflags "-X main.version=$VERSION" -o "$CLIENT_BINARY" ./cmd/driftmgr-client
    
    if [ $? -eq 0 ]; then
        log_success "Client built successfully: $CLIENT_BINARY"
    else
        log_error "Client build failed"
        exit 1
    fi
}

# Build for specific platform
build_for_platform() {
    local platform=$1
    local arch=${2:-amd64}
    
    log_info "Building for $platform/$arch..."
    
    case $platform in
        "windows")
            GOOS=windows GOARCH=$arch go build -ldflags "-X main.version=$VERSION" -o "$SERVER_BINARY-windows-$arch.exe" ./cmd/driftmgr-server
            GOOS=windows GOARCH=$arch go build -ldflags "-X main.version=$VERSION" -o "$CLIENT_BINARY-windows-$arch.exe" ./cmd/driftmgr-client
            ;;
        "linux")
            GOOS=linux GOARCH=$arch go build -ldflags "-X main.version=$VERSION" -o "$SERVER_BINARY-linux-$arch" ./cmd/driftmgr-server
            GOOS=linux GOARCH=$arch go build -ldflags "-X main.version=$VERSION" -o "$CLIENT_BINARY-linux-$arch" ./cmd/driftmgr-client
            ;;
        "darwin")
            GOOS=darwin GOARCH=$arch go build -ldflags "-X main.version=$VERSION" -o "$SERVER_BINARY-darwin-$arch" ./cmd/driftmgr-server
            GOOS=darwin GOARCH=$arch go build -ldflags "-X main.version=$VERSION" -o "$CLIENT_BINARY-darwin-$arch" ./cmd/driftmgr-client
            ;;
        *)
            log_error "Unsupported platform: $platform"
            exit 1
            ;;
    esac
    
    log_success "Build for $platform/$arch completed"
}

# Run tests
run_tests() {
    log_info "Running tests..."
    go test -v ./...
    log_success "Tests completed"
}

# Run linting
run_lint() {
    log_info "Running linter..."
    if command -v golangci-lint &> /dev/null; then
        golangci-lint run
        log_success "Linting completed"
    else
        log_warning "golangci-lint not found, skipping linting"
    fi
}

# Show build info
show_build_info() {
    log_info "Build Information:"
    echo "  Project: $PROJECT_NAME"
    echo "  Version: $VERSION"
    echo "  Platform: $(uname -s) $(uname -m)"
    echo "  Go version: $(go version)"
    echo "  Build time: $(date)"
}

# Show usage
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --clean              Clean previous builds before building"
    echo "  --deps               Install dependencies"
    echo "  --server             Build server only"
    echo "  --client             Build client only"
    echo "  --test               Run tests"
    echo "  --lint               Run linter"
    echo "  --platform PLATFORM  Build for specific platform (windows, linux, darwin)"
    echo "  --arch ARCH          Architecture (amd64, arm64, 386) [default: amd64]"
    echo "  --all-platforms      Build for all platforms"
    echo "  --help               Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                    # Build for current platform"
    echo "  $0 --clean --test     # Clean, build, and test"
    echo "  $0 --platform windows # Build for Windows"
    echo "  $0 --all-platforms    # Build for all platforms"
}

# Main build function
main_build() {
    local clean=false
    local deps=false
    local server_only=false
    local client_only=false
    local run_tests_flag=false
    local run_lint_flag=false
    local platform=""
    local arch="amd64"
    local all_platforms=false
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --clean)
                clean=true
                shift
                ;;
            --deps)
                deps=true
                shift
                ;;
            --server)
                server_only=true
                shift
                ;;
            --client)
                client_only=true
                shift
                ;;
            --test)
                run_tests_flag=true
                shift
                ;;
            --lint)
                run_lint_flag=true
                shift
                ;;
            --platform)
                platform="$2"
                shift 2
                ;;
            --arch)
                arch="$2"
                shift 2
                ;;
            --all-platforms)
                all_platforms=true
                shift
                ;;
            --help)
                show_usage
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done
    
    # Show build information
    show_build_info
    echo ""
    
    # Check prerequisites
    check_prerequisites
    
    # Install dependencies if requested
    if [ "$deps" = true ]; then
        install_dependencies
    fi
    
    # Clean if requested
    if [ "$clean" = true ]; then
        clean_builds
    fi
    
    # Build for specific platform
    if [ -n "$platform" ]; then
        build_for_platform "$platform" "$arch"
        exit 0
    fi
    
    # Build for all platforms
    if [ "$all_platforms" = true ]; then
        log_info "Building for all platforms..."
        build_for_platform "windows" "amd64"
        build_for_platform "linux" "amd64"
        build_for_platform "darwin" "amd64"
        build_for_platform "darwin" "arm64"
        log_success "All platform builds completed"
        exit 0
    fi
    
    # Build applications
    if [ "$server_only" = true ]; then
        build_server
    elif [ "$client_only" = true ]; then
        build_client
    else
        build_server
        build_client
    fi
    
    # Run tests if requested
    if [ "$run_tests_flag" = true ]; then
        run_tests
    fi
    
    # Run linting if requested
    if [ "$run_lint_flag" = true ]; then
        run_lint
    fi
    
    # Show results
    echo ""
    log_success "Build completed successfully!"
    echo ""
    log_info "Built binaries:"
    if [ -f "$SERVER_BINARY" ]; then
        echo "  Server: $SERVER_BINARY"
    fi
    if [ -f "$CLIENT_BINARY" ]; then
        echo "  Client: $CLIENT_BINARY"
    fi
    echo ""
    log_info "To run the applications:"
    echo "  Server: ./$SERVER_BINARY"
    echo "  Client: ./$CLIENT_BINARY"
}

# Run main function with all arguments
main_build "$@"
