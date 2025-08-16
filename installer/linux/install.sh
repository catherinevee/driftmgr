#!/bin/bash

# DriftMgr Linux Installer
# This script installs DriftMgr and configures the environment

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default installation path
INSTALL_PATH="${HOME}/driftmgr"
FORCE=false
SKIP_CREDENTIAL_CHECK=false

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

# Function to detect OS and package manager
detect_os() {
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        OS=$NAME
        VER=$VERSION_ID
    else
        OS=$(uname -s)
        VER=$(uname -r)
    fi
    
    # Detect package manager
    if command -v apt-get &> /dev/null; then
        PKG_MANAGER="apt"
    elif command -v yum &> /dev/null; then
        PKG_MANAGER="yum"
    elif command -v dnf &> /dev/null; then
        PKG_MANAGER="dnf"
    elif command -v zypper &> /dev/null; then
        PKG_MANAGER="zypper"
    elif command -v pacman &> /dev/null; then
        PKG_MANAGER="pacman"
    else
        PKG_MANAGER="unknown"
    fi
}

# Function to check if Go is installed
check_go() {
    if command -v go &> /dev/null; then
        GO_VERSION=$(go version | awk '{print $3}')
        print_success "Go is installed: $GO_VERSION"
        return 0
    else
        print_error "Go is not installed or not in PATH"
        return 1
    fi
}

# Function to install Go
install_go() {
    print_status "Installing Go..."
    
    # Detect architecture
    ARCH=$(uname -m)
    case $ARCH in
        x86_64)
            GO_ARCH="amd64"
            ;;
        aarch64|arm64)
            GO_ARCH="arm64"
            ;;
        *)
            print_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac
    
    # Download and install Go
    GO_VERSION="1.21.0"
    GO_TAR="go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    GO_URL="https://golang.org/dl/${GO_TAR}"
    TEMP_DIR=$(mktemp -d)
    
    cd "$TEMP_DIR"
    
    print_status "Downloading Go..."
    if command -v curl &> /dev/null; then
        curl -L -O "$GO_URL"
    elif command -v wget &> /dev/null; then
        wget "$GO_URL"
    else
        print_error "Neither curl nor wget is available"
        exit 1
    fi
    
    print_status "Installing Go..."
    sudo tar -C /usr/local -xzf "$GO_TAR"
    
    # Add Go to PATH
    if ! grep -q "/usr/local/go/bin" ~/.bashrc; then
        echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
        export PATH=$PATH:/usr/local/go/bin
    fi
    
    # Cleanup
    cd - > /dev/null
    rm -rf "$TEMP_DIR"
    
    print_success "Go installed successfully"
}

# Function to install DriftMgr
install_driftmgr() {
    print_status "Installing DriftMgr..."
    
    # Create installation directory
    mkdir -p "$INSTALL_PATH"
    
    # Get the directory where this script is located
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"
    
    # Copy executable to installation directory
    SOURCE_EXE="$PROJECT_ROOT/bin/driftmgr"
    SOURCE_SERVER="$PROJECT_ROOT/bin/driftmgr-server"
    TARGET_EXE="$INSTALL_PATH/driftmgr"
    TARGET_SERVER="$INSTALL_PATH/driftmgr-server"
    
    if [[ -f "$SOURCE_EXE" ]]; then
        cp "$SOURCE_EXE" "$TARGET_EXE"
        chmod +x "$TARGET_EXE"
        print_success "DriftMgr CLI installed"
    else
        print_error "DriftMgr executable not found at $SOURCE_EXE"
        exit 1
    fi
    
    if [[ -f "$SOURCE_SERVER" ]]; then
        cp "$SOURCE_SERVER" "$TARGET_SERVER"
        chmod +x "$TARGET_SERVER"
        print_success "DriftMgr Server installed"
    fi
    
    # Copy configuration files
    CONFIG_FILES=("driftmgr.yaml" "go.mod" "go.sum")
    for file in "${CONFIG_FILES[@]}"; do
        SOURCE_FILE="$PROJECT_ROOT/$file"
        TARGET_FILE="$INSTALL_PATH/$file"
        if [[ -f "$SOURCE_FILE" ]]; then
            cp "$SOURCE_FILE" "$TARGET_FILE"
        fi
    done
    
    # Copy assets directory
    SOURCE_ASSETS="$PROJECT_ROOT/assets"
    TARGET_ASSETS="$INSTALL_PATH/assets"
    if [[ -d "$SOURCE_ASSETS" ]]; then
        cp -r "$SOURCE_ASSETS" "$TARGET_ASSETS"
    fi
    
    # Copy docs directory
    SOURCE_DOCS="$PROJECT_ROOT/docs"
    TARGET_DOCS="$INSTALL_PATH/docs"
    if [[ -d "$SOURCE_DOCS" ]]; then
        cp -r "$SOURCE_DOCS" "$TARGET_DOCS"
    fi
    
    # Copy examples directory
    SOURCE_EXAMPLES="$PROJECT_ROOT/examples"
    TARGET_EXAMPLES="$INSTALL_PATH/examples"
    if [[ -d "$SOURCE_EXAMPLES" ]]; then
        cp -r "$SOURCE_EXAMPLES" "$TARGET_EXAMPLES"
    fi
}

# Function to add DriftMgr to PATH
add_to_path() {
    print_status "Adding DriftMgr to PATH..."
    
    # Determine shell configuration file
    if [[ -n "$ZSH_VERSION" ]]; then
        SHELL_RC="$HOME/.zshrc"
    else
        SHELL_RC="$HOME/.bashrc"
    fi
    
    # Add to PATH if not already present
    if ! grep -q "$INSTALL_PATH" "$SHELL_RC"; then
        echo "export PATH=\"\$PATH:$INSTALL_PATH\"" >> "$SHELL_RC"
        print_success "DriftMgr added to PATH in $SHELL_RC"
        
        # Update current session PATH
        export PATH="$PATH:$INSTALL_PATH"
    else
        print_success "DriftMgr already in PATH"
    fi
}

# Function to check AWS credentials
check_aws_credentials() {
    print_status "Checking AWS credentials..."
    
    AWS_PROFILES=()
    
    # Check AWS CLI configuration
    if [[ -f "$HOME/.aws/config" ]]; then
        AWS_PROFILES+=("AWS CLI config found")
    fi
    
    # Check environment variables
    if [[ -n "$AWS_ACCESS_KEY_ID" && -n "$AWS_SECRET_ACCESS_KEY" ]]; then
        AWS_PROFILES+=("AWS environment variables found")
    fi
    
    # Check AWS credentials file
    if [[ -f "$HOME/.aws/credentials" ]]; then
        AWS_PROFILES+=("AWS credentials file found")
    fi
    
    # Check AWS CLI
    if command -v aws &> /dev/null; then
        if aws sts get-caller-identity &> /dev/null; then
            AWS_PROFILES+=("AWS CLI authenticated")
        fi
    fi
    
    if [[ ${#AWS_PROFILES[@]} -gt 0 ]]; then
        print_success "AWS credentials detected:"
        for profile in "${AWS_PROFILES[@]}"; do
            echo "  - $profile"
        done
        return 0
    else
        print_warning "No AWS credentials found"
        return 1
    fi
}

# Function to check Azure credentials
check_azure_credentials() {
    print_status "Checking Azure credentials..."
    
    AZURE_PROFILES=()
    
    # Check Azure CLI
    if command -v az &> /dev/null; then
        if az account show &> /dev/null; then
            AZURE_PROFILES+=("Azure CLI authenticated")
        fi
    fi
    
    # Check environment variables
    if [[ -n "$AZURE_CLIENT_ID" && -n "$AZURE_CLIENT_SECRET" && -n "$AZURE_TENANT_ID" ]]; then
        AZURE_PROFILES+=("Azure environment variables found")
    fi
    
    if [[ ${#AZURE_PROFILES[@]} -gt 0 ]]; then
        print_success "Azure credentials detected:"
        for profile in "${AZURE_PROFILES[@]}"; do
            echo "  - $profile"
        done
        return 0
    else
        print_warning "No Azure credentials found"
        return 1
    fi
}

# Function to check GCP credentials
check_gcp_credentials() {
    print_status "Checking GCP credentials..."
    
    GCP_PROFILES=()
    
    # Check gcloud CLI
    if command -v gcloud &> /dev/null; then
        GCLOUD_AUTH=$(gcloud auth list --filter=status:ACTIVE --format="value(account)" 2>/dev/null)
        if [[ -n "$GCLOUD_AUTH" ]]; then
            GCP_PROFILES+=("GCP CLI authenticated: $GCLOUD_AUTH")
        fi
    fi
    
    # Check service account key file
    if [[ -n "$GOOGLE_APPLICATION_CREDENTIALS" && -f "$GOOGLE_APPLICATION_CREDENTIALS" ]]; then
        GCP_PROFILES+=("GCP service account key found")
    fi
    
    if [[ ${#GCP_PROFILES[@]} -gt 0 ]]; then
        print_success "GCP credentials detected:"
        for profile in "${GCP_PROFILES[@]}"; do
            echo "  - $profile"
        done
        return 0
    else
        print_warning "No GCP credentials found"
        return 1
    fi
}

# Function to install AWS CLI
install_aws_cli() {
    print_status "Installing AWS CLI..."
    
    case $PKG_MANAGER in
        apt)
            sudo apt-get update
            sudo apt-get install -y awscli
            ;;
        yum|dnf)
            sudo $PKG_MANAGER install -y awscli
            ;;
        pacman)
            sudo pacman -S --noconfirm aws-cli
            ;;
        *)
            # Install using pip if package manager not available
            if command -v pip3 &> /dev/null; then
                pip3 install --user awscli
            elif command -v pip &> /dev/null; then
                pip install --user awscli
            else
                print_error "No suitable package manager found for AWS CLI installation"
                return 1
            fi
            ;;
    esac
    
    print_success "AWS CLI installed successfully"
}

# Function to install Azure CLI
install_azure_cli() {
    print_status "Installing Azure CLI..."
    
    case $PKG_MANAGER in
        apt)
            # Add Microsoft signing key
            curl -sL https://packages.microsoft.com/keys/microsoft.asc | gpg --dearmor | sudo tee /etc/apt/trusted.gpg.d/microsoft.gpg > /dev/null
            echo "deb [arch=amd64] https://packages.microsoft.com/repos/azure-cli/ $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/azure-cli.list
            sudo apt-get update
            sudo apt-get install -y azure-cli
            ;;
        yum|dnf)
            sudo rpm --import https://packages.microsoft.com/keys/microsoft.asc
            echo -e "[azure-cli]\nname=Azure CLI\nbaseurl=https://packages.microsoft.com/yumrepos/azure-cli\nenabled=1\ngpgcheck=1\ngpgkey=https://packages.microsoft.com/keys/microsoft.asc" | sudo tee /etc/yum.repos.d/azure-cli.repo
            sudo $PKG_MANAGER install -y azure-cli
            ;;
        pacman)
            sudo pacman -S --noconfirm azure-cli
            ;;
        *)
            # Install using script
            curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash
            ;;
    esac
    
    print_success "Azure CLI installed successfully"
}

# Function to install Google Cloud CLI
install_gcp_cli() {
    print_status "Installing Google Cloud CLI..."
    
    # Download and install Google Cloud SDK
    TEMP_DIR=$(mktemp -d)
    cd "$TEMP_DIR"
    
    # Detect architecture
    ARCH=$(uname -m)
    case $ARCH in
        x86_64)
            GCP_ARCH="x86_64"
            ;;
        aarch64|arm64)
            GCP_ARCH="arm"
            ;;
        *)
            print_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac
    
    # Download Google Cloud SDK
    GCP_VERSION="latest"
    GCP_TAR="google-cloud-cli-${GCP_VERSION}-linux-${GCP_ARCH}.tar.gz"
    GCP_URL="https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/${GCP_TAR}"
    
    if command -v curl &> /dev/null; then
        curl -L -O "$GCP_URL"
    elif command -v wget &> /dev/null; then
        wget "$GCP_URL"
    else
        print_error "Neither curl nor wget is available"
        exit 1
    fi
    
    # Extract and install
    tar -xzf "$GCP_TAR"
    ./google-cloud-sdk/install.sh --quiet --usage-reporting=false --rc-path="$HOME/.bashrc"
    
    # Cleanup
    cd - > /dev/null
    rm -rf "$TEMP_DIR"
    
    print_success "Google Cloud CLI installed successfully"
}

# Function to create desktop shortcuts
create_desktop_shortcuts() {
    print_status "Creating desktop shortcuts..."
    
    # Create desktop entry for DriftMgr CLI
    cat > "$HOME/.local/share/applications/driftmgr-cli.desktop" << EOF
[Desktop Entry]
Version=1.0
Type=Application
Name=DriftMgr CLI
Comment=Cloud Infrastructure Drift Detection and Remediation
Exec=$INSTALL_PATH/driftmgr
Icon=terminal
Terminal=true
Categories=System;Utility;
EOF
    
    # Create desktop entry for DriftMgr Server
    cat > "$HOME/.local/share/applications/driftmgr-server.desktop" << EOF
[Desktop Entry]
Version=1.0
Type=Application
Name=DriftMgr Server
Comment=DriftMgr Web Dashboard
Exec=$INSTALL_PATH/driftmgr-server
Icon=web-browser
Terminal=false
Categories=Network;WebBrowser;
EOF
    
    # Make desktop entries executable
    chmod +x "$HOME/.local/share/applications/driftmgr-cli.desktop"
    chmod +x "$HOME/.local/share/applications/driftmgr-server.desktop"
    
    print_success "Desktop shortcuts created"
}

# Function to show installation summary
show_installation_summary() {
    echo
    print_success "🎉 DriftMgr Installation Complete!"
    echo "====================================="
    echo
    echo "Installation Path: $INSTALL_PATH"
    echo "Executable: driftmgr"
    echo "Server: driftmgr-server"
    echo
    print_warning "📋 Next Steps:"
    echo "1. Restart your terminal or run: source ~/.bashrc (or ~/.zshrc)"
    echo "2. Run 'driftmgr --help' to see available commands"
    echo "3. Run 'driftmgr-server' to start the web dashboard"
    echo "4. Configure your cloud credentials if not detected"
    echo
    print_warning "📚 Documentation:"
    echo "- User Guide: $INSTALL_PATH/docs/user-guide/"
    echo "- Examples: $INSTALL_PATH/examples/"
    echo
    print_warning "🔗 Quick Start:"
    echo "driftmgr discover --provider aws --region us-east-1"
    echo "driftmgr-server"
}

# Function to show help
show_help() {
    echo "DriftMgr Linux Installer"
    echo
    echo "Usage: $0 [OPTIONS]"
    echo
    echo "Options:"
    echo "  -p, --path PATH        Installation path (default: $INSTALL_PATH)"
    echo "  -f, --force           Force installation (overwrite existing)"
    echo "  -s, --skip-credentials Skip cloud credential check"
    echo "  -h, --help            Show this help message"
    echo
}

# Parse command line arguments
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
    echo "🚀 DriftMgr Linux Installer"
    echo "==========================="
    echo
    
    # Detect OS and package manager
    detect_os
    print_status "Detected OS: $OS"
    print_status "Package Manager: $PKG_MANAGER"
    
    # Check if Go is installed
    if ! check_go; then
        print_warning "Go is required for DriftMgr. Installing..."
        install_go
    fi
    
    # Install DriftMgr
    install_driftmgr
    
    # Add to PATH
    add_to_path
    
    # Create desktop shortcuts
    create_desktop_shortcuts
    
    # Check cloud credentials (unless skipped)
    if [[ "$SKIP_CREDENTIAL_CHECK" == "false" ]]; then
        echo
        print_status "🔍 Checking Cloud Provider Credentials..."
        
        HAS_CREDENTIALS=false
        
        if check_aws_credentials; then
            HAS_CREDENTIALS=true
        else
            echo -n "Would you like to install AWS CLI? (y/n): "
            read -r response
            if [[ "$response" =~ ^[Yy]$ ]]; then
                install_aws_cli
            fi
        fi
        
        if check_azure_credentials; then
            HAS_CREDENTIALS=true
        else
            echo -n "Would you like to install Azure CLI? (y/n): "
            read -r response
            if [[ "$response" =~ ^[Yy]$ ]]; then
                install_azure_cli
            fi
        fi
        
        if check_gcp_credentials; then
            HAS_CREDENTIALS=true
        else
            echo -n "Would you like to install Google Cloud CLI? (y/n): "
            read -r response
            if [[ "$response" =~ ^[Yy]$ ]]; then
                install_gcp_cli
            fi
        fi
        
        if [[ "$HAS_CREDENTIALS" == "false" ]]; then
            echo
            print_warning "No cloud credentials detected. You'll need to configure them manually."
            echo "See the documentation for setup instructions."
        fi
    fi
    
    # Show installation summary
    show_installation_summary
}

# Run main function
main "$@"
