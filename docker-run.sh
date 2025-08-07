#!/bin/bash
#
# Docker runner script for Terraform Import Helper
# Makes it easy to run driftmgr in Docker with proper volume mounts and environment setup
#

set -e

# Default values
IMAGE="catherinevee/driftmgr:latest"
CONFIG_DIR="./config"
OUTPUT_DIR="./output"
INPUT_FILE=""
AWS_PROFILE="default"
INTERACTIVE=false

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

show_help() {
    cat << EOF
🐳 Docker Runner for Terraform Import Helper

Usage: $0 [OPTIONS] COMMAND [ARGS...]

OPTIONS:
    -i, --image IMAGE       Docker image to use (default: catherinevee/driftmgr:latest)
    -c, --config DIR        Config directory to mount (default: ./config)
    -o, --output DIR        Output directory to mount (default: ./output)
    -f, --file FILE         Input file to mount
    -p, --profile PROFILE   AWS profile to use (default: default)
    --interactive           Run in interactive mode with TTY
    --build                 Build image locally before running
    -h, --help              Show this help

COMMANDS:
    discover                Discover cloud resources
    import                  Import resources to Terraform
    interactive             Launch interactive TUI
    config                  Manage configuration
    help                    Show driftmgr help
    version                 Show version

EXAMPLES:
    # Basic help
    $0 help

    # Discover AWS resources
    $0 discover --provider aws --region us-east-1

    # Import with file
    $0 -f resources.csv import --file /input/resources.csv

    # Interactive mode
    $0 --interactive interactive

    # Build and run locally
    $0 --build version

    # Custom config directory
    $0 -c /path/to/config discover

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -i|--image)
            IMAGE="$2"
            shift 2
            ;;
        -c|--config)
            CONFIG_DIR="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        -f|--file)
            INPUT_FILE="$2"
            shift 2
            ;;
        -p|--profile)
            AWS_PROFILE="$2"
            shift 2
            ;;
        --interactive)
            INTERACTIVE=true
            shift
            ;;
        --build)
            BUILD_LOCAL=true
            shift
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            # Remaining arguments are the command
            break
            ;;
    esac
done

# Build locally if requested
if [[ "$BUILD_LOCAL" == "true" ]]; then
    log_info "Building Docker image locally..."
    docker build -t driftmgr:local .
    IMAGE="driftmgr:local"
    log_success "Image built successfully"
fi

# Create directories if they don't exist
mkdir -p "$CONFIG_DIR" "$OUTPUT_DIR"

# Build Docker run command
DOCKER_ARGS=(
    "run"
    "--rm"
)

# Add interactive flags if needed
if [[ "$INTERACTIVE" == "true" ]] || [[ "$1" == "interactive" ]]; then
    DOCKER_ARGS+=("-it")
fi

# Add volume mounts
DOCKER_ARGS+=(
    "-v" "$(pwd)/$CONFIG_DIR:/config:ro"
    "-v" "$(pwd)/$OUTPUT_DIR:/output"
)

# Add input file mount if specified
if [[ -n "$INPUT_FILE" ]]; then
    if [[ ! -f "$INPUT_FILE" ]]; then
        log_error "Input file not found: $INPUT_FILE"
        exit 1
    fi
    DOCKER_ARGS+=("-v" "$(pwd)/$INPUT_FILE:/input/$(basename "$INPUT_FILE"):ro")
fi

# Add AWS credentials if available
if [[ -d "$HOME/.aws" ]]; then
    DOCKER_ARGS+=("-v" "$HOME/.aws:/root/.aws:ro")
    DOCKER_ARGS+=("-e" "AWS_PROFILE=$AWS_PROFILE")
fi

# Add Azure credentials if available
if [[ -d "$HOME/.azure" ]]; then
    DOCKER_ARGS+=("-v" "$HOME/.azure:/root/.azure:ro")
fi

# Add GCP credentials if available
if [[ -d "$HOME/.config/gcloud" ]]; then
    DOCKER_ARGS+=("-v" "$HOME/.config/gcloud:/root/.config/gcloud:ro")
fi

# Add common environment variables
DOCKER_ARGS+=(
    "-e" "AWS_REGION=${AWS_REGION:-us-east-1}"
    "-e" "AZURE_LOCATION=${AZURE_LOCATION:-eastus}"
    "-e" "GCP_REGION=${GCP_REGION:-us-central1}"
)

# Add image and command
DOCKER_ARGS+=("$IMAGE")

# Add remaining arguments as the command
if [[ $# -gt 0 ]]; then
    DOCKER_ARGS+=("$@")
fi

# Show what we're about to run
log_info "Running: docker ${DOCKER_ARGS[*]}"
echo

# Execute the command
exec docker "${DOCKER_ARGS[@]}"
