#!/bin/bash
# DriftMgr Installation Script
# This script installs DriftMgr and its dependencies for Unix/Linux/macOS

set -e

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_NAME="DriftMgr"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="$HOME/.driftmgr"
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
    log_error "Installation failed at line $1"
    exit 1
}

trap 'handle_error $LINENO' ERR

# Check if running as root
check_root() {
    if [ "$EUID" -eq 0 ]; then
        log_warning "Running as root. This is not recommended for security reasons."
        read -p "Continue anyway? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
}

# Detect operating system
detect_os() {
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        OS="linux"
        if [ -f /etc/debian_version ]; then
            DISTRO="debian"
        elif [ -f /etc/redhat-release ]; then
            DISTRO="redhat"
        elif [ -f /etc/arch-release ]; then
            DISTRO="arch"
        else
            DISTRO="unknown"
        fi
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        OS="macos"
        DISTRO="macos"
    else
        OS="unknown"
        DISTRO="unknown"
    fi
    
    log_info "Detected OS: $OS ($DISTRO)"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed. Please install Go 1.21 or later."
        log_info "Visit: https://golang.org/doc/install"
        exit 1
    fi
    
    # Check Go version
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    log_info "Go version: $GO_VERSION"
    
    # Check if git is installed
    if ! command -v git &> /dev/null; then
        log_error "Git is not installed. Please install Git."
        exit 1
    fi
    
    # Check if make is installed
    if ! command -v make &> /dev/null; then
        log_warning "Make is not installed. Some build features may not work."
    fi
    
    log_success "Prerequisites check passed"
}

# Install system dependencies
install_system_deps() {
    log_info "Installing system dependencies..."
    
    case $DISTRO in
        "debian"|"ubuntu")
            sudo apt-get update
            sudo apt-get install -y build-essential git curl wget
            ;;
        "redhat"|"centos"|"fedora")
            sudo yum groupinstall -y "Development Tools"
            sudo yum install -y git curl wget
            ;;
        "arch")
            sudo pacman -S --needed base-devel git curl wget
            ;;
        "macos")
            if ! command -v brew &> /dev/null; then
                log_info "Installing Homebrew..."
                /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
            fi
            brew install git curl wget
            ;;
        *)
            log_warning "Unknown distribution. Please install build tools manually."
            ;;
    esac
    
    log_success "System dependencies installed"
}

# Install Go tools
install_go_tools() {
    log_info "Installing Go tools..."
    
    # Install golangci-lint
    if ! command -v golangci-lint &> /dev/null; then
        log_info "Installing golangci-lint..."
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2
    fi
    
    # Install air for hot reloading
    if ! command -v air &> /dev/null; then
        log_info "Installing air..."
        go install github.com/cosmtrek/air@latest
    fi
    
    # Install godoc
    if ! command -v godoc &> /dev/null; then
        log_info "Installing godoc..."
        go install golang.org/x/tools/cmd/godoc@latest
    fi
    
    log_success "Go tools installed"
}

# Build DriftMgr
build_driftmgr() {
    log_info "Building DriftMgr..."
    
    # Install dependencies
    go mod download
    go mod tidy
    
    # Build applications
    go build -ldflags "-X main.version=$VERSION" -o driftmgr-server ./cmd/driftmgr-server
    go build -ldflags "-X main.version=$VERSION" -o driftmgr-client ./cmd/driftmgr-client
    
    log_success "DriftMgr built successfully"
}

# Install binaries
install_binaries() {
    log_info "Installing binaries to $INSTALL_DIR..."
    
    # Create install directory if it doesn't exist
    sudo mkdir -p "$INSTALL_DIR"
    
    # Install binaries
    sudo cp driftmgr-server "$INSTALL_DIR/"
    sudo cp driftmgr-client "$INSTALL_DIR/"
    
    # Make executable
    sudo chmod +x "$INSTALL_DIR/driftmgr-server"
    sudo chmod +x "$INSTALL_DIR/driftmgr-client"
    
    log_success "Binaries installed to $INSTALL_DIR"
}

# Create configuration directory
create_config() {
    log_info "Creating configuration directory..."
    
    mkdir -p "$CONFIG_DIR"
    
    # Create default configuration
    cat > "$CONFIG_DIR/config.yaml" << EOF
# DriftMgr Configuration
version: "1.0"

# Server configuration
server:
  host: "localhost"
  port: 8080
  debug: false

# Client configuration
client:
  server_url: "http://localhost:8080"
  timeout: 30s

# Logging configuration
logging:
  level: "info"
  format: "json"
  output: "stdout"

# Cloud provider configuration
providers:
  aws:
    enabled: true
    regions: ["us-east-1", "us-west-2"]
  azure:
    enabled: false
    subscription_id: ""
  gcp:
    enabled: false
    project_id: ""
EOF
    
    log_success "Configuration created at $CONFIG_DIR/config.yaml"
}

# Create desktop shortcuts (Linux)
create_desktop_shortcuts() {
    if [ "$OS" = "linux" ]; then
        log_info "Creating desktop shortcuts..."
        
        # Create desktop entry for server
        cat > "$HOME/.local/share/applications/driftmgr-server.desktop" << EOF
[Desktop Entry]
Name=DriftMgr Server
Comment=DriftMgr Infrastructure Management Server
Exec=$INSTALL_DIR/driftmgr-server
Icon=terminal
Terminal=true
Type=Application
Categories=Development;System;
EOF
        
        # Create desktop entry for client
        cat > "$HOME/.local/share/applications/driftmgr-client.desktop" << EOF
[Desktop Entry]
Name=DriftMgr Client
Comment=DriftMgr Infrastructure Management Client
Exec=$INSTALL_DIR/driftmgr-client
Icon=terminal
Terminal=true
Type=Application
Categories=Development;System;
EOF
        
        log_success "Desktop shortcuts created"
    fi
}

# Run tests
run_tests() {
    log_info "Running tests..."
    go test -v ./...
    log_success "Tests passed"
}

# Show installation info
show_installation_info() {
    log_info "Installation Information:"
    echo "  Project: $PROJECT_NAME"
    echo "  Version: $VERSION"
    echo "  OS: $OS ($DISTRO)"
    echo "  Install directory: $INSTALL_DIR"
    echo "  Config directory: $CONFIG_DIR"
    echo "  Go version: $(go version)"
    echo "  Install time: $(date)"
}

# Show usage
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --local              Install to local directory (not system-wide)"
    echo "  --config-only        Only create configuration files"
    echo "  --build-only         Only build, don't install"
    echo "  --test               Run tests after installation"
    echo "  --skip-deps          Skip system dependency installation"
    echo "  --skip-go-tools      Skip Go tools installation"
    echo "  --help               Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                    # Full installation"
    echo "  $0 --local            # Local installation"
    echo "  $0 --build-only       # Build only"
    echo "  $0 --test             # Install and test"
}

# Main installation function
main_install() {
    local local_install=false
    local config_only=false
    local build_only=false
    local run_tests_flag=false
    local skip_deps=false
    local skip_go_tools=false
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --local)
                local_install=true
                INSTALL_DIR="$HOME/.local/bin"
                shift
                ;;
            --config-only)
                config_only=true
                shift
                ;;
            --build-only)
                build_only=true
                shift
                ;;
            --test)
                run_tests_flag=true
                shift
                ;;
            --skip-deps)
                skip_deps=true
                shift
                ;;
            --skip-go-tools)
                skip_go_tools=true
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
    
    # Show installation information
    show_installation_info
    echo ""
    
    # Check if running as root
    check_root
    
    # Detect operating system
    detect_os
    
    # Check prerequisites
    check_prerequisites
    
    # Install system dependencies
    if [ "$skip_deps" = false ]; then
        install_system_deps
    fi
    
    # Install Go tools
    if [ "$skip_go_tools" = false ]; then
        install_go_tools
    fi
    
    # Build DriftMgr
    if [ "$config_only" = false ]; then
        build_driftmgr
    fi
    
    # Install binaries
    if [ "$build_only" = false ] && [ "$config_only" = false ]; then
        install_binaries
    fi
    
    # Create configuration
    create_config
    
    # Create desktop shortcuts
    create_desktop_shortcuts
    
    # Run tests if requested
    if [ "$run_tests_flag" = true ]; then
        run_tests
    fi
    
    # Show results
    echo ""
    log_success "Installation completed successfully!"
    echo ""
    log_info "Installed binaries:"
    echo "  Server: $INSTALL_DIR/driftmgr-server"
    echo "  Client: $INSTALL_DIR/driftmgr-client"
    echo ""
    log_info "Configuration:"
    echo "  Config file: $CONFIG_DIR/config.yaml"
    echo ""
    log_info "To get started:"
    echo "  Server: driftmgr-server"
    echo "  Client: driftmgr-client"
    echo "  Web interface: http://localhost:8080"
    echo ""
    log_info "Documentation:"
    echo "  See docs/ directory for detailed documentation"
}

# Run main function with all arguments
main_install "$@"
