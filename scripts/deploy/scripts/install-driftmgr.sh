#!/bin/bash

# DriftMgr CI/CD Installation Script
# This script installs DriftMgr in various CI/CD environments

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to detect CI/CD platform
detect_platform() {
    if [ -n "$GITHUB_ACTIONS" ]; then
        echo "github-actions"
    elif [ -n "$GITLAB_CI" ]; then
        echo "gitlab-ci"
    elif [ -n "$JENKINS_URL" ]; then
        echo "jenkins"
    elif [ -n "$AZURE_DEVOPS" ]; then
        echo "azure-devops"
    elif [ -n "$CIRCLECI" ]; then
        echo "circleci"
    else
        echo "unknown"
    fi
}

# Function to check if Go is installed
check_go() {
    if command -v go &> /dev/null; then
        GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
        print_status "Go version $GO_VERSION found"
        return 0
    else
        print_error "Go is not installed"
        return 1
    fi
}

# Function to install Go if needed
install_go() {
    local platform=$1
    
    print_status "Installing Go..."
    
    case $platform in
        "github-actions"|"gitlab-ci"|"circleci")
            # These platforms usually have Go pre-installed
            if ! check_go; then
                print_error "Go installation required but not supported on this platform"
                exit 1
            fi
            ;;
        "jenkins"|"azure-devops")
            # Install Go if not present
            if ! check_go; then
                print_status "Installing Go 1.21..."
                wget -q https://golang.org/dl/go1.21.linux-amd64.tar.gz
                sudo tar -C /usr/local -xzf go1.21.linux-amd64.tar.gz
                export PATH=$PATH:/usr/local/go/bin
                rm go1.21.linux-amd64.tar.gz
                print_success "Go installed successfully"
            fi
            ;;
        *)
            print_warning "Unknown platform, attempting to use existing Go installation"
            if ! check_go; then
                print_error "Go installation required"
                exit 1
            fi
            ;;
    esac
}

# Function to install DriftMgr
install_driftmgr() {
    local install_path=${1:-"/usr/local/bin"}
    local version=${2:-"latest"}
    
    print_status "Installing DriftMgr..."
    
    # Create temporary directory
    TEMP_DIR=$(mktemp -d)
    cd "$TEMP_DIR"
    
    # Clone DriftMgr repository
    print_status "Cloning DriftMgr repository..."
    git clone https://github.com/catherinevee/driftmgr.git
    cd driftmgr
    
    # Checkout specific version if provided
    if [ "$version" != "latest" ]; then
        print_status "Checking out version $version..."
        git checkout "$version"
    fi
    
    # Build DriftMgr
    print_status "Building DriftMgr..."
    make build
    
    # Install binaries
    print_status "Installing DriftMgr binaries to $install_path..."
    sudo cp bin/driftmgr "$install_path/"
    sudo cp bin/driftmgr-client "$install_path/"
    sudo cp bin/driftmgr-server "$install_path/"
    
    # Make executable
    sudo chmod +x "$install_path"/driftmgr*
    
    # Clean up
    cd /
    rm -rf "$TEMP_DIR"
    
    print_success "DriftMgr installed successfully"
}

# Function to verify installation
verify_installation() {
    print_status "Verifying DriftMgr installation..."
    
    if command -v driftmgr &> /dev/null; then
        DRIFT_VERSION=$(driftmgr --version 2>/dev/null || echo "unknown")
        print_success "DriftMgr installed successfully (version: $DRIFT_VERSION)"
        return 0
    else
        print_error "DriftMgr installation verification failed"
        return 1
    fi
}

# Function to create configuration
create_config() {
    local config_file=${1:-"driftmgr.yaml"}
    local environment=${2:-"production"}
    
    print_status "Creating DriftMgr configuration..."
    
    cat > "$config_file" << EOF
# DriftMgr Configuration for CI/CD
providers:
  aws:
    regions: [us-east-1, us-west-2]
  azure:
    regions: [eastus, westus]
  gcp:
    regions: [us-central1, us-east1]

ci_cd:
  enabled: true
  fail_on_drift: true
  auto_remediate: false
  severity_threshold: high
  notification_channels:
    - slack
    - email
  
  environments:
    development:
      fail_on_drift: false
      auto_remediate: true
      severity_threshold: medium
    staging:
      fail_on_drift: true
      auto_remediate: false
      severity_threshold: high
    production:
      fail_on_drift: true
      auto_remediate: false
      severity_threshold: high

# Output configuration
output:
  format: json
  directory: drift-reports
  retention_days: 30

# Notification configuration
notifications:
  slack:
    webhook_url: \${SLACK_WEBHOOK_URL}
  email:
    smtp_server: \${SMTP_SERVER}
    smtp_port: \${SMTP_PORT}
    username: \${SMTP_USERNAME}
    password: \${SMTP_PASSWORD}
    from_address: \${FROM_EMAIL}
    to_addresses: \${TO_EMAILS}
EOF
    
    print_success "Configuration created: $config_file"
}

# Function to setup environment variables
setup_environment() {
    local platform=$1
    
    print_status "Setting up environment variables..."
    
    # Export common environment variables
    export DRIFT_CONFIG_FILE=${DRIFT_CONFIG_FILE:-"driftmgr.yaml"}
    export DRIFT_OUTPUT_FORMAT=${DRIFT_OUTPUT_FORMAT:-"json"}
    export DRIFT_FAIL_ON_DRIFT=${DRIFT_FAIL_ON_DRIFT:-"true"}
    export DRIFT_AUTO_REMEDIATE=${DRIFT_AUTO_REMEDIATE:-"false"}
    export DRIFT_SEVERITY_THRESHOLD=${DRIFT_SEVERITY_THRESHOLD:-"high"}
    
    # Platform-specific environment setup
    case $platform in
        "github-actions")
            # GitHub Actions environment variables are already set
            print_status "Using GitHub Actions environment variables"
            ;;
        "gitlab-ci")
            # GitLab CI environment variables are already set
            print_status "Using GitLab CI environment variables"
            ;;
        "jenkins")
            # Jenkins environment variables
            print_status "Using Jenkins environment variables"
            ;;
        "azure-devops")
            # Azure DevOps environment variables
            print_status "Using Azure DevOps environment variables"
            ;;
        "circleci")
            # CircleCI environment variables
            print_status "Using CircleCI environment variables"
            ;;
    esac
    
    print_success "Environment variables configured"
}

# Function to run basic tests
run_tests() {
    print_status "Running basic DriftMgr tests..."
    
    # Test help command
    if driftmgr --help &> /dev/null; then
        print_success "Help command works"
    else
        print_error "Help command failed"
        return 1
    fi
    
    # Test version command
    if driftmgr --version &> /dev/null; then
        print_success "Version command works"
    else
        print_warning "Version command not available"
    fi
    
    print_success "Basic tests completed"
}

# Main installation function
main() {
    local install_path=${1:-"/usr/local/bin"}
    local version=${2:-"latest"}
    local config_file=${3:-"driftmgr.yaml"}
    local environment=${4:-"production"}
    
    print_status "Starting DriftMgr CI/CD installation..."
    
    # Detect platform
    PLATFORM=$(detect_platform)
    print_status "Detected platform: $PLATFORM"
    
    # Install Go if needed
    install_go "$PLATFORM"
    
    # Install DriftMgr
    install_driftmgr "$install_path" "$version"
    
    # Verify installation
    if ! verify_installation; then
        print_error "Installation verification failed"
        exit 1
    fi
    
    # Create configuration
    create_config "$config_file" "$environment"
    
    # Setup environment
    setup_environment "$PLATFORM"
    
    # Run basic tests
    run_tests
    
    print_success "DriftMgr CI/CD installation completed successfully!"
    
    # Print usage information
    echo ""
    print_status "Usage examples:"
    echo "  driftmgr discover aws us-east-1"
    echo "  driftmgr analyze terraform.tfstate"
    echo "  driftmgr remediate-batch terraform --auto"
    echo "  driftmgr notify slack 'Drift Alert' 'Drift detected'"
    echo ""
    print_status "Configuration file: $config_file"
    print_status "Installation path: $install_path"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --install-path)
            INSTALL_PATH="$2"
            shift 2
            ;;
        --version)
            VERSION="$2"
            shift 2
            ;;
        --config-file)
            CONFIG_FILE="$2"
            shift 2
            ;;
        --environment)
            ENVIRONMENT="$2"
            shift 2
            ;;
        --help)
            echo "DriftMgr CI/CD Installation Script"
            echo ""
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --install-path PATH    Installation path (default: /usr/local/bin)"
            echo "  --version VERSION      DriftMgr version to install (default: latest)"
            echo "  --config-file FILE     Configuration file path (default: driftmgr.yaml)"
            echo "  --environment ENV      Environment name (default: production)"
            echo "  --help                 Show this help message"
            echo ""
            echo "Environment Variables:"
            echo "  DRIFT_CONFIG_FILE      DriftMgr configuration file"
            echo "  DRIFT_OUTPUT_FORMAT    Output format (json, yaml, text)"
            echo "  DRIFT_FAIL_ON_DRIFT    Fail pipeline on drift detection"
            echo "  DRIFT_AUTO_REMEDIATE   Automatically remediate drift"
            echo "  DRIFT_SEVERITY_THRESHOLD Minimum severity to fail pipeline"
            echo ""
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Run main installation
main "${INSTALL_PATH:-/usr/local/bin}" "${VERSION:-latest}" "${CONFIG_FILE:-driftmgr.yaml}" "${ENVIRONMENT:-production}"
