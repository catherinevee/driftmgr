#!/bin/bash

# DriftMgr Universal Installer
# This script detects the operating system and runs the appropriate installer

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}$1${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# Function to detect operating system
detect_os() {
    case "$(uname -s)" in
        Linux*)
            echo "linux"
            ;;
        Darwin*)
            echo "macos"
            ;;
        CYGWIN*|MINGW*|MSYS*)
            echo "windows"
            ;;
        *)
            echo "unknown"
            ;;
    esac
}

# Function to show help
show_help() {
    echo "DriftMgr Universal Installer"
    echo
    echo "Usage: $0 [OPTIONS]"
    echo
    echo "Options:"
    echo "  -p, --path PATH        Installation path"
    echo "  -f, --force           Force installation (overwrite existing)"
    echo "  -s, --skip-credentials Skip cloud credential check"
    echo "  -h, --help            Show this help message"
    echo
    echo "Supported Operating Systems:"
    echo "  - Windows (PowerShell)"
    echo "  - Linux (Bash)"
    echo "  - macOS (Bash)"
    echo
}

# Function to run Windows installer
run_windows_installer() {
    print_status "Detected Windows. Running PowerShell installer..."
    
    # Get the directory where this script is located
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    WINDOWS_INSTALLER="$SCRIPT_DIR/installer/windows/install.ps1"
    
    if [[ ! -f "$WINDOWS_INSTALLER" ]]; then
        print_error "Windows installer not found at $WINDOWS_INSTALLER"
        exit 1
    fi
    
    # Build PowerShell command with arguments
    PS_ARGS=""
    if [[ -n "$INSTALL_PATH" ]]; then
        PS_ARGS="$PS_ARGS -InstallPath '$INSTALL_PATH'"
    fi
    if [[ "$FORCE" == "true" ]]; then
        PS_ARGS="$PS_ARGS -Force"
    fi
    if [[ "$SKIP_CREDENTIAL_CHECK" == "true" ]]; then
        PS_ARGS="$PS_ARGS -SkipCredentialCheck"
    fi
    
    # Run PowerShell installer
    if command -v powershell &> /dev/null; then
        powershell -ExecutionPolicy Bypass -File "$WINDOWS_INSTALLER" $PS_ARGS
    elif command -v pwsh &> /dev/null; then
        pwsh -ExecutionPolicy Bypass -File "$WINDOWS_INSTALLER" $PS_ARGS
    else
        print_error "PowerShell not found. Please install PowerShell Core or Windows PowerShell."
        exit 1
    fi
}

# Function to run Linux/macOS installer
run_linux_installer() {
    print_status "Detected Linux/macOS. Running Bash installer..."
    
    # Get the directory where this script is located
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    LINUX_INSTALLER="$SCRIPT_DIR/installer/linux/install.sh"
    
    if [[ ! -f "$LINUX_INSTALLER" ]]; then
        print_error "Linux installer not found at $LINUX_INSTALLER"
        exit 1
    fi
    
    # Make installer executable
    chmod +x "$LINUX_INSTALLER"
    
    # Build bash command with arguments
    BASH_ARGS=""
    if [[ -n "$INSTALL_PATH" ]]; then
        BASH_ARGS="$BASH_ARGS --path '$INSTALL_PATH'"
    fi
    if [[ "$FORCE" == "true" ]]; then
        BASH_ARGS="$BASH_ARGS --force"
    fi
    if [[ "$SKIP_CREDENTIAL_CHECK" == "true" ]]; then
        BASH_ARGS="$BASH_ARGS --skip-credentials"
    fi
    
    # Run bash installer
    bash "$LINUX_INSTALLER" $BASH_ARGS
}

# Function to check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    # Check if we're in the right directory
    if [[ ! -f "driftmgr.yaml" ]]; then
        print_warning "driftmgr.yaml not found in current directory."
        print_warning "Make sure you're running this script from the DriftMgr project root."
    fi
    
    # Check if binaries exist
    if [[ ! -d "bin" ]]; then
        print_error "bin directory not found. Please build DriftMgr first."
        print_error "Run 'make build' or 'go build' to build the application."
        exit 1
    fi
    
    # Check for required binaries
    OS=$(detect_os)
    if [[ "$OS" == "windows" ]]; then
        if [[ ! -f "bin/driftmgr.exe" ]]; then
            print_error "driftmgr.exe not found in bin directory."
            exit 1
        fi
    else
        if [[ ! -f "bin/driftmgr" ]]; then
            print_error "driftmgr binary not found in bin directory."
            exit 1
        fi
    fi
    
    print_success "Prerequisites check passed"
}

# Parse command line arguments
INSTALL_PATH=""
FORCE=false
SKIP_CREDENTIAL_CHECK=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -p|--path)
            INSTALL_PATH="$2"
            shift 2
            ;;
        -f|--force)
            FORCE=true
            shift
            ;;
        -s|--skip-credentials)
            SKIP_CREDENTIAL_CHECK=true
            shift
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Main installation script
main() {
    echo "🚀 DriftMgr Universal Installer"
    echo "==============================="
    echo
    
    # Detect operating system
    OS=$(detect_os)
    print_status "Detected OS: $OS"
    
    # Check prerequisites
    check_prerequisites
    
    # Run appropriate installer based on OS
    case "$OS" in
        windows)
            run_windows_installer
            ;;
        linux|macos)
            run_linux_installer
            ;;
        unknown)
            print_error "Unsupported operating system: $(uname -s)"
            print_error "Please run the appropriate installer manually:"
            echo "  - Windows: installer/windows/install.ps1"
            echo "  - Linux/macOS: installer/linux/install.sh"
            exit 1
            ;;
    esac
    
    print_success "Installation completed successfully!"
}

# Run main function
main "$@"
