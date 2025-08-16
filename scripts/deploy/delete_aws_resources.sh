#!/bin/bash

# DriftMgr AWS Resource Deletion Script (Bash version)
# This script demonstrates how to safely delete all resources in your AWS account using driftmgr

set -e

# Configuration
SERVER_URL="http://localhost:8080"
API_BASE_URL="$SERVER_URL/api/v1"
ACCOUNT_ID=""
REGION="us-east-1"
FORCE=false
DRY_RUN=true

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${CYAN}=== DriftMgr AWS Resource Deletion Script ===${NC}"
    echo ""
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${WHITE}$1${NC}"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --account-id)
            ACCOUNT_ID="$2"
            shift 2
            ;;
        --region)
            REGION="$2"
            shift 2
            ;;
        --force)
            FORCE=true
            shift
            ;;
        --no-dry-run)
            DRY_RUN=false
            shift
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo "Options:"
            echo "  --account-id ACCOUNT_ID    AWS Account ID (auto-detected if not provided)"
            echo "  --region REGION           AWS Region (default: us-east-1)"
            echo "  --force                   Bypass safety checks"
            echo "  --no-dry-run              Perform actual deletion (DANGEROUS!)"
            echo "  --help                    Show this help message"
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

print_status

# Check if server is running
print_info "Checking if DriftMgr server is running..."
if curl -s "$SERVER_URL/health" > /dev/null; then
    print_success "DriftMgr server is running"
else
    print_error "DriftMgr server is not running. Please start it first:"
    print_info "  ./driftmgr-server.exe"
    exit 1
fi

# Get AWS Account ID if not provided
if [ -z "$ACCOUNT_ID" ]; then
    print_info "Getting AWS Account ID..."
    if command -v aws &> /dev/null; then
        ACCOUNT_ID=$(aws sts get-caller-identity --query 'Account' --output text)
        print_success "AWS Account ID: $ACCOUNT_ID"
    else
        print_error "AWS CLI not found. Please install and configure AWS CLI."
        print_info "  Run: aws configure"
        exit 1
    fi
fi

# Step 1: Get supported providers
echo ""
print_info "Step 1: Checking supported providers..."
PROVIDERS_RESPONSE=$(curl -s "$API_BASE_URL/delete/providers")
PROVIDERS=$(echo "$PROVIDERS_RESPONSE" | jq -r '.providers | join(", ")')
print_success "Supported providers: $PROVIDERS"

# Step 2: Preview deletion (DRY RUN) - ALWAYS DO THIS FIRST
echo ""
print_info "Step 2: Previewing deletion (DRY RUN)..."
print_warning "This will show you what resources would be deleted WITHOUT actually deleting them."

PREVIEW_REQUEST=$(cat <<EOF
{
  "provider": "aws",
  "account_id": "$ACCOUNT_ID",
  "options": {
    "dry_run": true,
    "force": $FORCE,
    "resource_types": [],
    "regions": ["$REGION"],
    "exclude_resources": [],
    "include_resources": [],
    "timeout": "30m",
    "batch_size": 10
  }
}
EOF
)

PREVIEW_RESPONSE=$(curl -s -X POST "$API_BASE_URL/delete/preview" \
    -H "Content-Type: application/json" \
    -d "$PREVIEW_REQUEST")

if [ $? -eq 0 ]; then
    TOTAL_RESOURCES=$(echo "$PREVIEW_RESPONSE" | jq -r '.total_resources')
    DELETED_RESOURCES=$(echo "$PREVIEW_RESPONSE" | jq -r '.deleted_resources')
    SKIPPED_RESOURCES=$(echo "$PREVIEW_RESPONSE" | jq -r '.skipped_resources')
    
    print_success "Preview completed successfully"
    print_info "  Total resources found: $TOTAL_RESOURCES"
    print_info "  Resources that would be deleted: $DELETED_RESOURCES"
    print_info "  Resources that would be skipped: $SKIPPED_RESOURCES"
    
    # Show errors if any
    ERROR_COUNT=$(echo "$PREVIEW_RESPONSE" | jq -r '.errors | length')
    if [ "$ERROR_COUNT" -gt 0 ]; then
        print_warning "  Errors during preview:"
        echo "$PREVIEW_RESPONSE" | jq -r '.errors[] | "    - \(.resource_id) (\(.resource_type)): \(.error)"'
    fi
    
    # Show warnings if any
    WARNING_COUNT=$(echo "$PREVIEW_RESPONSE" | jq -r '.warnings | length')
    if [ "$WARNING_COUNT" -gt 0 ]; then
        print_warning "  Warnings:"
        echo "$PREVIEW_RESPONSE" | jq -r '.warnings[] | "    - \(.)"'
    fi
else
    print_error "Preview failed"
    exit 1
fi

# Step 3: Ask for confirmation if not in dry-run mode
if [ "$DRY_RUN" = false ]; then
    echo ""
    print_error "WARNING: You are about to DELETE ALL RESOURCES in AWS Account $ACCOUNT_ID"
    print_error "This action is IRREVERSIBLE and will result in data loss!"
    echo ""
    
    read -p "Are you absolutely sure you want to proceed? Type 'YES DELETE ALL' to confirm: " confirmation
    
    if [ "$confirmation" != "YES DELETE ALL" ]; then
        print_warning "Deletion cancelled by user."
        exit 0
    fi
    
    # Step 4: Execute actual deletion
    echo ""
    print_info "Step 4: Executing actual deletion..."
    print_warning "This will actually delete the resources. Progress will be shown below."
    
    DELETION_REQUEST=$(cat <<EOF
{
  "provider": "aws",
  "account_id": "$ACCOUNT_ID",
  "options": {
    "dry_run": false,
    "force": $FORCE,
    "resource_types": [],
    "regions": ["$REGION"],
    "exclude_resources": [],
    "include_resources": [],
    "timeout": "30m",
    "batch_size": 10
  }
}
EOF
)
    
    DELETION_RESPONSE=$(curl -s -X POST "$API_BASE_URL/delete/account" \
        -H "Content-Type: application/json" \
        -d "$DELETION_REQUEST")
    
    if [ $? -eq 0 ]; then
        TOTAL_PROCESSED=$(echo "$DELETION_RESPONSE" | jq -r '.total_resources')
        DELETED=$(echo "$DELETION_RESPONSE" | jq -r '.deleted_resources')
        FAILED=$(echo "$DELETION_RESPONSE" | jq -r '.failed_resources')
        SKIPPED=$(echo "$DELETION_RESPONSE" | jq -r '.skipped_resources')
        DURATION=$(echo "$DELETION_RESPONSE" | jq -r '.duration')
        
        print_success "Deletion completed successfully"
        print_info "  Total resources processed: $TOTAL_PROCESSED"
        print_info "  Resources deleted: $DELETED"
        print_info "  Resources failed: $FAILED"
        print_info "  Resources skipped: $SKIPPED"
        print_info "  Duration: $DURATION"
        
        # Show errors if any
        ERROR_COUNT=$(echo "$DELETION_RESPONSE" | jq -r '.errors | length')
        if [ "$ERROR_COUNT" -gt 0 ]; then
            print_warning "  Errors during deletion:"
            echo "$DELETION_RESPONSE" | jq -r '.errors[] | "    - \(.resource_id) (\(.resource_type)): \(.error)"'
        fi
    else
        print_error "Deletion failed"
        exit 1
    fi
else
    echo ""
    print_success "Dry-run mode completed. No resources were actually deleted."
    echo ""
    print_info "To perform actual deletion, run this script with:"
    print_info "  $0 --no-dry-run --force"
    echo ""
    print_error "WARNING: Actual deletion will permanently delete all resources!"
fi

echo ""
print_info "=== Script completed ==="
