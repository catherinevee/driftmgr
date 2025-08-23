#!/bin/bash

# DriftMgr Installation Script
# This script installs DriftMgr and its dependencies on Linux/macOS systems
# 
# Usage: curl -sSL https://raw.githubusercontent.com/your-org/driftmgr/main/ci-cd/scripts/install-driftmgr.sh | bash
# Or: wget -qO- https://raw.githubusercontent.com/your-org/driftmgr/main/ci-cd/scripts/install-driftmgr.sh | bash

set -euo pipefail

# Configuration
GITHUB_REPO="your-org/driftmgr"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="$HOME/.driftmgr"
TEMP_DIR="/tmp/driftmgr-install"
VERSION=""
FORCE_INSTALL=false
INSTALL_DEPS=true
VERIFY_CHECKSUM=true

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Help function
show_help() {
    cat << EOF
DriftMgr Installation Script

USAGE:
    install-driftmgr.sh [OPTIONS]

OPTIONS:
    -v, --version VERSION    Install specific version (default: latest)
    -d, --install-dir DIR    Installation directory (default: /usr/local/bin)
    -f, --force             Force installation even if already installed
    --no-deps               Skip dependency installation
    --no-verify             Skip checksum verification
    -h, --help              Show this help message

EXAMPLES:
    # Install latest version
    ./install-driftmgr.sh

    # Install specific version
    ./install-driftmgr.sh --version v1.2.3

    # Install to custom directory
    ./install-driftmgr.sh --install-dir /opt/driftmgr/bin

    # Force reinstall with no dependency check
    ./install-driftmgr.sh --force --no-deps

ENVIRONMENT VARIABLES:
    DRIFTMGR_VERSION        Version to install (overridden by --version)
    DRIFTMGR_INSTALL_DIR    Installation directory (overridden by --install-dir)
    GITHUB_TOKEN            GitHub token for API access (for rate limiting)

EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -v|--version)
                VERSION="$2"
                shift 2
                ;;
            -d|--install-dir)
                INSTALL_DIR="$2"
                shift 2
                ;;
            -f|--force)
                FORCE_INSTALL=true
                shift
                ;;
            --no-deps)
                INSTALL_DEPS=false
                shift
                ;;
            --no-verify)
                VERIFY_CHECKSUM=false
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done

    # Override with environment variables if not set
    VERSION=${VERSION:-${DRIFTMGR_VERSION:-}}
    INSTALL_DIR=${INSTALL_DIR:-${DRIFTMGR_INSTALL_DIR:-/usr/local/bin}}
}

# Detect operating system and architecture
detect_platform() {
    local os arch

    # Detect OS
    case "$(uname -s)" in
        Linux*)   os="linux" ;;
        Darwin*)  os="darwin" ;;
        CYGWIN*)  os="windows" ;;
        MINGW*)   os="windows" ;;
        *)        
            log_error "Unsupported operating system: $(uname -s)"
            exit 1
            ;;
    esac

    # Detect architecture
    case "$(uname -m)" in
        x86_64)   arch="amd64" ;;
        amd64)    arch="amd64" ;;
        arm64)    arch="arm64" ;;
        aarch64)  arch="arm64" ;;
        armv7l)   arch="arm" ;;
        *)        
            log_error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac

    echo "${os}-${arch}"
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check system requirements
check_requirements() {
    log_info "Checking system requirements..."

    # Check for required commands
    local required_commands=("curl" "tar")
    
    if [[ "$VERIFY_CHECKSUM" == "true" ]]; then
        required_commands+=("sha256sum")
    fi

    for cmd in "${required_commands[@]}"; do
        if ! command_exists "$cmd"; then
            log_error "Required command not found: $cmd"
            log_info "Please install $cmd and try again"
            exit 1
        fi
    done

    # Check if installation directory exists and is writable
    if [[ ! -d "$INSTALL_DIR" ]]; then
        log_info "Creating installation directory: $INSTALL_DIR"
        if ! sudo mkdir -p "$INSTALL_DIR" 2>/dev/null; then
            log_error "Cannot create installation directory: $INSTALL_DIR"
            exit 1
        fi
    fi

    if [[ ! -w "$INSTALL_DIR" ]]; then
        log_warning "Installation directory $INSTALL_DIR is not writable by current user"
        log_info "Will attempt to use sudo for installation"
    fi
}

# Get latest version from GitHub API
get_latest_version() {
    local api_url="https://api.github.com/repos/$GITHUB_REPO/releases/latest"
    local auth_header=""

    if [[ -n "${GITHUB_TOKEN:-}" ]]; then
        auth_header="Authorization: token $GITHUB_TOKEN"
    fi

    log_info "Fetching latest version from GitHub..."

    if command_exists "curl"; then
        local response
        if [[ -n "$auth_header" ]]; then
            response=$(curl -sSL -H "$auth_header" "$api_url")
        else
            response=$(curl -sSL "$api_url")
        fi
        
        echo "$response" | grep '"tag_name":' | sed -E 's/.*"tag_name": "([^"]+)".*/\1/'
    else
        log_error "curl is required to fetch latest version"
        exit 1
    fi
}

# Download and verify DriftMgr binary
download_driftmgr() {
    local version="$1"
    local platform="$2"
    local archive_name="driftmgr-${platform}.tar.gz"
    local download_url="https://github.com/$GITHUB_REPO/releases/download/$version/$archive_name"
    local checksum_url="https://github.com/$GITHUB_REPO/releases/download/$version/checksums.txt"

    log_info "Downloading DriftMgr $version for $platform..."

    # Create temporary directory
    mkdir -p "$TEMP_DIR"
    cd "$TEMP_DIR"

    # Download binary archive
    if ! curl -sSL -o "$archive_name" "$download_url"; then
        log_error "Failed to download DriftMgr from $download_url"
        exit 1
    fi

    # Download and verify checksum if enabled
    if [[ "$VERIFY_CHECKSUM" == "true" ]]; then
        log_info "Verifying checksum..."
        
        if ! curl -sSL -o "checksums.txt" "$checksum_url"; then
            log_warning "Failed to download checksums, skipping verification"
        else
            local expected_checksum
            expected_checksum=$(grep "$archive_name" checksums.txt | awk '{print $1}')
            
            if [[ -n "$expected_checksum" ]]; then
                local actual_checksum
                actual_checksum=$(sha256sum "$archive_name" | awk '{print $1}')
                
                if [[ "$actual_checksum" != "$expected_checksum" ]]; then
                    log_error "Checksum verification failed!"
                    log_error "Expected: $expected_checksum"
                    log_error "Actual:   $actual_checksum"
                    exit 1
                fi
                
                log_success "Checksum verification passed"
            else
                log_warning "Could not find checksum for $archive_name, skipping verification"
            fi
        fi
    fi

    # Extract archive
    log_info "Extracting archive..."
    if ! tar -xzf "$archive_name"; then
        log_error "Failed to extract archive"
        exit 1
    fi

    # Find the binary (handle different archive structures)
    local binary_path=""
    if [[ -f "driftmgr" ]]; then
        binary_path="driftmgr"
    elif [[ -f "driftmgr-${platform}" ]]; then
        binary_path="driftmgr-${platform}"
    else
        # Look for any executable file named driftmgr*
        binary_path=$(find . -name "driftmgr*" -type f -executable | head -n1)
    fi

    if [[ -z "$binary_path" ]]; then
        log_error "Could not find DriftMgr binary in archive"
        exit 1
    fi

    echo "$TEMP_DIR/$binary_path"
}

# Install DriftMgr binary
install_binary() {
    local binary_path="$1"
    local target_path="$INSTALL_DIR/driftmgr"

    log_info "Installing DriftMgr to $target_path..."

    # Check if binary already exists
    if [[ -f "$target_path" ]] && [[ "$FORCE_INSTALL" != "true" ]]; then
        local existing_version
        existing_version=$("$target_path" --version 2>/dev/null | head -n1 || echo "unknown")
        
        log_warning "DriftMgr is already installed: $existing_version"
        read -p "Do you want to overwrite it? [y/N] " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Installation cancelled by user"
            exit 0
        fi
    fi

    # Install binary
    if [[ -w "$INSTALL_DIR" ]]; then
        cp "$binary_path" "$target_path"
        chmod +x "$target_path"
    else
        sudo cp "$binary_path" "$target_path"
        sudo chmod +x "$target_path"
    fi

    # Verify installation
    if [[ -x "$target_path" ]]; then
        local installed_version
        installed_version=$("$target_path" --version 2>/dev/null | head -n1 || echo "unknown")
        log_success "DriftMgr installed successfully: $installed_version"
    else
        log_error "Installation failed - binary is not executable"
        exit 1
    fi
}

# Install cloud CLI dependencies
install_dependencies() {
    if [[ "$INSTALL_DEPS" != "true" ]]; then
        log_info "Skipping dependency installation"
        return
    fi

    log_info "Installing cloud CLI dependencies..."

    # Detect package manager
    local package_manager=""
    if command_exists "apt-get"; then
        package_manager="apt"
    elif command_exists "yum"; then
        package_manager="yum"
    elif command_exists "dnf"; then
        package_manager="dnf"
    elif command_exists "brew"; then
        package_manager="brew"
    elif command_exists "pacman"; then
        package_manager="pacman"
    fi

    # Install AWS CLI
    if ! command_exists "aws"; then
        log_info "Installing AWS CLI..."
        case "$package_manager" in
            apt)
                curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
                unzip -q awscliv2.zip
                sudo ./aws/install
                rm -rf aws awscliv2.zip
                ;;
            yum|dnf)
                sudo $package_manager install -y awscli
                ;;
            brew)
                brew install awscli
                ;;
            *)
                log_warning "Cannot auto-install AWS CLI for this package manager"
                log_info "Please install AWS CLI manually: https://aws.amazon.com/cli/"
                ;;
        esac
    else
        log_info "AWS CLI already installed: $(aws --version 2>&1 | head -n1)"
    fi

    # Install Azure CLI
    if ! command_exists "az"; then
        log_info "Installing Azure CLI..."
        case "$package_manager" in
            apt)
                curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash
                ;;
            yum)
                sudo rpm --import https://packages.microsoft.com/keys/microsoft.asc
                sudo sh -c 'echo -e "[azure-cli]\nname=Azure CLI\nbaseurl=https://packages.microsoft.com/yumrepos/azure-cli\nenabled=1\ngpgcheck=1\ngpgkey=https://packages.microsoft.com/keys/microsoft.asc" > /etc/yum.repos.d/azure-cli.repo'
                sudo yum install -y azure-cli
                ;;
            dnf)
                sudo rpm --import https://packages.microsoft.com/keys/microsoft.asc
                sudo sh -c 'echo -e "[azure-cli]\nname=Azure CLI\nbaseurl=https://packages.microsoft.com/yumrepos/azure-cli\nenabled=1\ngpgcheck=1\ngpgkey=https://packages.microsoft.com/keys/microsoft.asc" > /etc/yum.repos.d/azure-cli.repo'
                sudo dnf install -y azure-cli
                ;;
            brew)
                brew install azure-cli
                ;;
            *)
                log_warning "Cannot auto-install Azure CLI for this package manager"
                log_info "Please install Azure CLI manually: https://docs.microsoft.com/en-us/cli/azure/install-azure-cli"
                ;;
        esac
    else
        log_info "Azure CLI already installed: $(az version --query '\"azure-cli\"' -o tsv 2>/dev/null || echo 'unknown')"
    fi

    # Install Google Cloud CLI
    if ! command_exists "gcloud"; then
        log_info "Installing Google Cloud CLI..."
        case "$package_manager" in
            apt)
                echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" | sudo tee -a /etc/apt/sources.list.d/google-cloud-sdk.list
                curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key --keyring /usr/share/keyrings/cloud.google.gpg add -
                sudo apt-get update && sudo apt-get install -y google-cloud-cli
                ;;
            yum)
                sudo tee -a /etc/yum.repos.d/google-cloud-sdk.repo << EOM
[google-cloud-cli]
name=Google Cloud CLI
baseurl=https://packages.cloud.google.com/yum/repos/cloud-sdk-el7-x86_64
enabled=1
gpgcheck=1
repo_gpgcheck=0
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg
       https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
EOM
                sudo yum install -y google-cloud-cli
                ;;
            brew)
                brew install google-cloud-sdk
                ;;
            *)
                log_warning "Cannot auto-install Google Cloud CLI for this package manager"
                log_info "Please install Google Cloud CLI manually: https://cloud.google.com/sdk/docs/install"
                ;;
        esac
    else
        log_info "Google Cloud CLI already installed: $(gcloud --version 2>/dev/null | head -n1 || echo 'unknown')"
    fi

    # Install doctl (DigitalOcean CLI)
    if ! command_exists "doctl"; then
        log_info "Installing doctl (DigitalOcean CLI)..."
        case "$package_manager" in
            brew)
                brew install doctl
                ;;
            *)
                # Install from GitHub releases
                local doctl_version="1.94.0"
                local doctl_url="https://github.com/digitalocean/doctl/releases/download/v${doctl_version}/doctl-${doctl_version}-linux-amd64.tar.gz"
                
                curl -sSL "$doctl_url" | tar -xz
                sudo mv doctl "$INSTALL_DIR/"
                ;;
        esac
    else
        log_info "doctl already installed: $(doctl version 2>/dev/null || echo 'unknown')"
    fi
}

# Create configuration directory and sample config
setup_configuration() {
    log_info "Setting up configuration..."

    # Create config directory
    mkdir -p "$CONFIG_DIR"

    # Create sample configuration if it doesn't exist
    local config_file="$CONFIG_DIR/config.yaml"
    if [[ ! -f "$config_file" ]]; then
        cat > "$config_file" << EOF
# DriftMgr Configuration File
# For more information, see: https://github.com/$GITHUB_REPO/docs/configuration

# Discovery settings
discovery:
  providers:
    - name: "aws"
      enabled: true
      regions: []  # Discovered dynamically
      services: ["ec2", "s3", "rds", "lambda"]
    - name: "azure"
      enabled: false
      regions: []  # Discovered dynamically
      services: ["compute", "storage", "network"]
    - name: "gcp"
      enabled: false
      regions: []  # Discovered dynamically
      services: ["compute", "storage"]
    - name: "digitalocean"
      enabled: false
      regions: ["nyc1", "sfo3"]
      services: ["droplets", "volumes"]

# Drift detection settings
drift_detection:
  enabled: true
  threshold: 10
  ignore_patterns:
    - "*.tfstate.backup"
    - "*/terraform.tfstate.d/*"

# Notification settings
notifications:
  slack:
    enabled: false
    webhook_url: ""
    channel: "#infrastructure-alerts"
  email:
    enabled: false
    smtp_server: ""
    from: ""
    to: ""

# Export settings
export:
  formats: ["json", "html"]
  output_dir: "./reports"

# Logging settings
logging:
  level: "info"
  file: ""
EOF
        
        log_success "Created sample configuration: $config_file"
        log_info "Edit this file to configure DriftMgr for your environment"
    else
        log_info "Configuration file already exists: $config_file"
    fi
}

# Add to PATH if not already there
update_path() {
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        log_info "Adding $INSTALL_DIR to PATH..."
        
        # Add to shell profiles
        local shell_profiles=("$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.profile")
        local path_line="export PATH=\"$INSTALL_DIR:\$PATH\""
        
        for profile in "${shell_profiles[@]}"; do
            if [[ -f "$profile" ]] && ! grep -q "$INSTALL_DIR" "$profile"; then
                echo "$path_line" >> "$profile"
                log_info "Added to $profile"
            fi
        done
        
        log_warning "Please run 'source ~/.bashrc' or restart your shell to update PATH"
    fi
}

# Cleanup temporary files
cleanup() {
    if [[ -d "$TEMP_DIR" ]]; then
        rm -rf "$TEMP_DIR"
    fi
}

# Main installation function
main() {
    # Banner
    cat << 'EOF'
  ____       _  __ _   __  __            
 |  _ \ _ __(_)/ _| |_|  \/  | __ _ _ __  
 | | | | '__| | |_| __| |\/| |/ _` | '__| 
 | |_| | |  | |  _| |_| |  | | (_| | |    
 |____/|_|  |_|_|  \__|_|  |_|\__, |_|    
                              |___/       
    Infrastructure Drift Detection Tool
    
EOF

    log_info "Starting DriftMgr installation..."

    # Parse command line arguments
    parse_args "$@"

    # Check system requirements
    check_requirements

    # Get version to install
    if [[ -z "$VERSION" ]]; then
        VERSION=$(get_latest_version)
        if [[ -z "$VERSION" ]]; then
            log_error "Could not determine latest version"
            exit 1
        fi
    fi

    log_info "Installing DriftMgr version: $VERSION"

    # Detect platform
    local platform
    platform=$(detect_platform)
    log_info "Detected platform: $platform"

    # Download and install
    local binary_path
    binary_path=$(download_driftmgr "$VERSION" "$platform")
    install_binary "$binary_path"

    # Install dependencies
    install_dependencies

    # Setup configuration
    setup_configuration

    # Update PATH
    update_path

    # Cleanup
    cleanup

    # Success message
    log_success "DriftMgr installation completed successfully!"
    echo
    echo "Next steps:"
    echo "1. Configure your cloud provider credentials"
    echo "2. Edit the configuration file: $CONFIG_DIR/config.yaml"
    echo "3. Run 'driftmgr --help' to see available commands"
    echo "4. Start with 'driftmgr discover' to scan your infrastructure"
    echo
    echo "Documentation: https://github.com/$GITHUB_REPO/docs"
    echo "Support: https://github.com/$GITHUB_REPO/issues"
}

# Trap to ensure cleanup on exit
trap cleanup EXIT

# Run main function with all arguments
main "$@"