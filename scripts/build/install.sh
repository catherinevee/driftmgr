#!/bin/bash

# DriftMgr Unix/Linux Installation Script
# This script installs DriftMgr and adds it to the system PATH

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="$SCRIPT_DIR/bin"
DRIFTMGR_EXE="$BIN_DIR/driftmgr"

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# Function to detect shell and shell config file
detect_shell_config() {
    local shell_name=$(basename "$SHELL")
    local config_file=""
    
    case "$shell_name" in
        "bash")
            if [[ -f "$HOME/.bashrc" ]]; then
                config_file="$HOME/.bashrc"
            elif [[ -f "$HOME/.bash_profile" ]]; then
                config_file="$HOME/.bash_profile"
            fi
            ;;
        "zsh")
            if [[ -f "$HOME/.zshrc" ]]; then
                config_file="$HOME/.zshrc"
            fi
            ;;
        "fish")
            if [[ -d "$HOME/.config/fish" ]]; then
                config_file="$HOME/.config/fish/config.fish"
            fi
            ;;
        *)
            # Default to bash
            if [[ -f "$HOME/.bashrc" ]]; then
                config_file="$HOME/.bashrc"
            elif [[ -f "$HOME/.bash_profile" ]]; then
                config_file="$HOME/.bash_profile"
            fi
            ;;
    esac
    
    echo "$config_file"
}

# Function to add to PATH
add_to_path() {
    local path_to_add="$1"
    local config_file="$2"
    
    if [[ -z "$config_file" ]]; then
        print_error "Could not determine shell configuration file"
        return 1
    fi
    
    # Check if already in PATH
    if grep -q "$path_to_add" "$config_file" 2>/dev/null; then
        print_warning "DriftMgr is already in your PATH configuration"
        return 0
    fi
    
    # Add to PATH
    echo "" >> "$config_file"
    echo "# DriftMgr PATH configuration" >> "$config_file"
    echo "export PATH=\"$path_to_add:\$PATH\"" >> "$config_file"
    
    print_success "Added DriftMgr to PATH in $config_file"
    return 0
}

# Function to create desktop entry (Linux)
create_desktop_entry() {
    local target_path="$1"
    
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        local desktop_dir="$HOME/.local/share/applications"
        local desktop_file="$desktop_dir/driftmgr.desktop"
        
        # Create directory if it doesn't exist
        mkdir -p "$desktop_dir"
        
        # Create desktop entry
        cat > "$desktop_file" << EOF
[Desktop Entry]
Version=1.0
Type=Application
Name=DriftMgr
Comment=Terraform Drift Detection & Remediation Tool
Exec=$target_path
Icon=terminal
Terminal=true
Categories=Development;System;
EOF
        
        # Make it executable
        chmod +x "$desktop_file"
        print_success "Created desktop entry: $desktop_file"
    fi
}

# Function to uninstall
uninstall() {
    print_status "Uninstalling DriftMgr..."
    
    local config_file=$(detect_shell_config)
    
    if [[ -n "$config_file" ]]; then
        # Remove PATH configuration
        if grep -q "DriftMgr PATH configuration" "$config_file" 2>/dev/null; then
            # Remove the lines we added
            sed -i '/# DriftMgr PATH configuration/,+1d' "$config_file"
            print_success "Removed DriftMgr from PATH configuration"
        fi
    fi
    
    # Remove desktop entry
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        local desktop_file="$HOME/.local/share/applications/driftmgr.desktop"
        if [[ -f "$desktop_file" ]]; then
            rm -f "$desktop_file"
            print_success "Removed desktop entry"
        fi
    fi
    
    print_success "DriftMgr has been uninstalled successfully"
    print_warning "Note: The executable files in the bin directory were not removed"
    print_warning "You can delete the entire driftmgr directory if you want to remove everything"
    exit 0
}

# Function to check if running as root
check_root() {
    if [[ $EUID -eq 0 ]]; then
        print_warning "Running as root is not recommended for user installations"
        read -p "Do you want to continue? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
}

# Main installation logic
main() {
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --uninstall)
                uninstall
                ;;
            --help|-h)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --uninstall    Uninstall DriftMgr"
                echo "  --help, -h     Show this help message"
                echo ""
                echo "This script installs DriftMgr and adds it to your system PATH."
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                echo "Use --help for usage information"
                exit 1
                ;;
        esac
        shift
    done
    
    print_status "Installing DriftMgr..."
    
    # Check if executables exist
    if [[ ! -f "$DRIFTMGR_EXE" ]]; then
        print_error "DriftMgr executable not found at: $DRIFTMGR_EXE"
        print_error "Please run 'make build' first to build the executables"
        exit 1
    fi
    
    # Make executable
    chmod +x "$DRIFTMGR_EXE"
    
    # Check for root
    check_root
    
    # Detect shell configuration
    local config_file=$(detect_shell_config)
    
    if [[ -z "$config_file" ]]; then
        print_warning "Could not detect shell configuration file"
        print_warning "You may need to manually add $BIN_DIR to your PATH"
    else
        # Add to PATH
        print_status "Adding DriftMgr to PATH..."
        if add_to_path "$BIN_DIR" "$config_file"; then
            print_success "PATH updated successfully"
        else
            print_error "Failed to update PATH"
            exit 1
        fi
    fi
    
    # Create desktop entry
    print_status "Creating desktop entry..."
    create_desktop_entry "$DRIFTMGR_EXE"
    
    # Display installation summary
    echo ""
    echo -e "${CYAN}==========================================${NC}"
    echo -e "${CYAN}DriftMgr Installation Complete!${NC}"
    echo -e "${CYAN}==========================================${NC}"
    echo ""
    echo -e "${BLUE}Installation Summary:${NC}"
    echo -e "  ✓ Added to PATH: $BIN_DIR"
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        echo -e "  ✓ Created desktop entry"
    fi
    echo ""
    echo -e "${BLUE}Usage:${NC}"
    echo -e "  • Open a new terminal or restart your current terminal"
    echo -e "  • Run 'driftmgr' to start the interactive shell"
    echo -e "  • Or run 'driftmgr discover aws all' for direct commands"
    echo ""
    echo -e "${BLUE}Timeout Configuration:${NC}"
    echo -e "  • For large infrastructure, configure timeouts:"
    echo -e "    ./scripts/set-timeout.sh -s large"
    echo -e "  • Or set environment variables:"
    echo -e "    export DRIFT_DISCOVERY_TIMEOUT=10m"
    echo ""
    echo -e "${YELLOW}Note: You may need to restart your terminal for PATH changes to take effect.${NC}"
    echo ""
    echo -e "${BLUE}To uninstall, run: $0 --uninstall${NC}"
}

# Run main function with all arguments
main "$@"
