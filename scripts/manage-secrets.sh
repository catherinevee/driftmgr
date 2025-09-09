#!/bin/bash

# Comprehensive GitHub secrets management script
# Supports multiple methods of setting secrets

set -e

# Configuration
REPO="${GITHUB_REPOSITORY:-catherinevee/driftmgr}"
SECRETS_FILE=".env.secrets"
ENCRYPTED_FILE=".env.secrets.gpg"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Print usage
usage() {
    cat << EOF
Usage: $0 [COMMAND] [OPTIONS]

Commands:
    setup       - Interactive setup of GitHub secrets
    import      - Import secrets from .env file
    export      - Export secret names (not values) to template
    validate    - Check which secrets are configured
    encrypt     - Encrypt secrets file with GPG
    decrypt     - Decrypt secrets file with GPG
    clean       - Remove local secrets files

Options:
    -r, --repo REPO     Repository name (default: $REPO)
    -f, --file FILE     Secrets file (default: $SECRETS_FILE)
    -h, --help          Show this help message

Examples:
    $0 setup                    # Interactive setup
    $0 import -f .env.prod      # Import from file
    $0 validate                 # Check configured secrets
    $0 encrypt -f .env.secrets  # Encrypt secrets file

EOF
    exit 0
}

# Check prerequisites
check_prerequisites() {
    if ! command -v gh &> /dev/null; then
        echo -e "${RED}Error: GitHub CLI (gh) is not installed${NC}"
        echo "Install from: https://cli.github.com/"
        exit 1
    fi
    
    if ! gh auth status &> /dev/null; then
        echo -e "${YELLOW}Authenticating with GitHub...${NC}"
        gh auth login
    fi
}

# Setup secrets interactively
setup_secrets() {
    check_prerequisites
    
    echo -e "${GREEN}=== GitHub Secrets Setup for $REPO ===${NC}\n"
    
    # Docker Hub
    echo -e "${BLUE}Docker Hub Configuration:${NC}"
    read -p "Docker Hub username: " docker_user
    read -sp "Docker Hub token: " docker_token
    echo
    
    if [[ -n "$docker_user" ]] && [[ -n "$docker_token" ]]; then
        echo "$docker_user" | gh secret set DOCKER_HUB_USERNAME --repo="$REPO"
        echo "$docker_token" | gh secret set DOCKER_HUB_TOKEN --repo="$REPO"
        echo -e "${GREEN}✓ Docker Hub secrets configured${NC}"
    fi
    
    # AWS (optional)
    echo -e "\n${BLUE}AWS Configuration (optional, press Enter to skip):${NC}"
    read -p "AWS Access Key ID: " aws_key
    if [[ -n "$aws_key" ]]; then
        read -sp "AWS Secret Access Key: " aws_secret
        echo
        read -p "AWS Region (default: us-east-1): " aws_region
        aws_region=${aws_region:-us-east-1}
        
        echo "$aws_key" | gh secret set AWS_ACCESS_KEY_ID --repo="$REPO"
        echo "$aws_secret" | gh secret set AWS_SECRET_ACCESS_KEY --repo="$REPO"
        echo "$aws_region" | gh secret set AWS_DEFAULT_REGION --repo="$REPO"
        echo -e "${GREEN}✓ AWS secrets configured${NC}"
    fi
    
    # Azure (optional)
    echo -e "\n${BLUE}Azure Configuration (optional, press Enter to skip):${NC}"
    read -p "Azure Client ID: " azure_client
    if [[ -n "$azure_client" ]]; then
        read -sp "Azure Client Secret: " azure_secret
        echo
        read -p "Azure Tenant ID: " azure_tenant
        read -p "Azure Subscription ID: " azure_subscription
        
        echo "$azure_client" | gh secret set AZURE_CLIENT_ID --repo="$REPO"
        echo "$azure_secret" | gh secret set AZURE_CLIENT_SECRET --repo="$REPO"
        echo "$azure_tenant" | gh secret set AZURE_TENANT_ID --repo="$REPO"
        echo "$azure_subscription" | gh secret set AZURE_SUBSCRIPTION_ID --repo="$REPO"
        echo -e "${GREEN}✓ Azure secrets configured${NC}"
    fi
    
    echo -e "\n${GREEN}✓ Setup complete!${NC}"
    validate_secrets
}

# Import secrets from file
import_secrets() {
    check_prerequisites
    
    if [[ ! -f "$SECRETS_FILE" ]]; then
        echo -e "${RED}Error: File $SECRETS_FILE not found${NC}"
        exit 1
    fi
    
    echo -e "${YELLOW}Importing secrets from $SECRETS_FILE...${NC}"
    
    # Read and set each secret
    while IFS='=' read -r key value; do
        # Skip comments and empty lines
        [[ "$key" =~ ^#.*$ ]] && continue
        [[ -z "$key" ]] && continue
        
        # Remove quotes and whitespace
        key=$(echo "$key" | xargs)
        value=$(echo "$value" | xargs | sed 's/^["'"'"']//;s/["'"'"']$//')
        
        if [[ -n "$value" ]] && [[ "$value" != "<"* ]]; then
            echo "$value" | gh secret set "$key" --repo="$REPO"
            echo -e "${GREEN}✓ Set $key${NC}"
        fi
    done < "$SECRETS_FILE"
    
    echo -e "\n${GREEN}✓ Import complete!${NC}"
}

# Export secret template
export_template() {
    cat > "$SECRETS_FILE.template" << 'EOF'
# GitHub Secrets Template for DriftMgr
# Fill in your values and import with: ./manage-secrets.sh import

# Docker Hub (Required for Docker workflows)
DOCKER_HUB_USERNAME=<your-docker-username>
DOCKER_HUB_TOKEN=<your-docker-token>

# AWS Credentials (Optional)
AWS_ACCESS_KEY_ID=<your-aws-key>
AWS_SECRET_ACCESS_KEY=<your-aws-secret>
AWS_DEFAULT_REGION=us-east-1

# Azure Credentials (Optional)
AZURE_CLIENT_ID=<your-azure-client-id>
AZURE_CLIENT_SECRET=<your-azure-secret>
AZURE_TENANT_ID=<your-azure-tenant>
AZURE_SUBSCRIPTION_ID=<your-azure-subscription>

# GCP Credentials (Optional)
GCP_SERVICE_ACCOUNT_JSON=<paste-entire-json>

# DigitalOcean (Optional)
DIGITALOCEAN_TOKEN=<your-do-token>

# Slack Notifications (Optional)
SLACK_WEBHOOK_URL=<your-slack-webhook>

# Terraform Cloud (Optional)
TF_API_TOKEN=<your-terraform-token>
EOF
    
    echo -e "${GREEN}✓ Template exported to $SECRETS_FILE.template${NC}"
    echo -e "${YELLOW}Edit the file and import with: $0 import -f $SECRETS_FILE.template${NC}"
}

# Validate configured secrets
validate_secrets() {
    check_prerequisites
    
    echo -e "${BLUE}=== Configured Secrets for $REPO ===${NC}\n"
    
    # Get list of secrets
    secrets=$(gh secret list --repo="$REPO" 2>/dev/null || echo "")
    
    if [[ -z "$secrets" ]]; then
        echo -e "${YELLOW}No secrets configured${NC}"
        return
    fi
    
    echo "$secrets"
    
    # Check specific secrets
    echo -e "\n${BLUE}Checking required secrets:${NC}"
    
    required_secrets=(
        "DOCKER_HUB_USERNAME"
        "DOCKER_HUB_TOKEN"
    )
    
    optional_secrets=(
        "AWS_ACCESS_KEY_ID"
        "AWS_SECRET_ACCESS_KEY"
        "AZURE_CLIENT_ID"
        "GCP_SERVICE_ACCOUNT_JSON"
        "DIGITALOCEAN_TOKEN"
        "SLACK_WEBHOOK_URL"
    )
    
    for secret in "${required_secrets[@]}"; do
        if echo "$secrets" | grep -q "$secret"; then
            echo -e "${GREEN}✓ $secret (required)${NC}"
        else
            echo -e "${RED}✗ $secret (required - missing!)${NC}"
        fi
    done
    
    for secret in "${optional_secrets[@]}"; do
        if echo "$secrets" | grep -q "$secret"; then
            echo -e "${GREEN}✓ $secret (optional)${NC}"
        else
            echo -e "${YELLOW}○ $secret (optional)${NC}"
        fi
    done
}

# Encrypt secrets file
encrypt_secrets() {
    if [[ ! -f "$SECRETS_FILE" ]]; then
        echo -e "${RED}Error: File $SECRETS_FILE not found${NC}"
        exit 1
    fi
    
    if ! command -v gpg &> /dev/null; then
        echo -e "${RED}Error: GPG is not installed${NC}"
        exit 1
    fi
    
    echo -e "${YELLOW}Encrypting $SECRETS_FILE...${NC}"
    gpg --symmetric --cipher-algo AES256 --output "$ENCRYPTED_FILE" "$SECRETS_FILE"
    
    echo -e "${GREEN}✓ Encrypted to $ENCRYPTED_FILE${NC}"
    echo -e "${YELLOW}Original file kept. Run 'clean' to remove it.${NC}"
}

# Decrypt secrets file
decrypt_secrets() {
    if [[ ! -f "$ENCRYPTED_FILE" ]]; then
        echo -e "${RED}Error: File $ENCRYPTED_FILE not found${NC}"
        exit 1
    fi
    
    echo -e "${YELLOW}Decrypting $ENCRYPTED_FILE...${NC}"
    gpg --decrypt --output "$SECRETS_FILE" "$ENCRYPTED_FILE"
    
    echo -e "${GREEN}✓ Decrypted to $SECRETS_FILE${NC}"
    echo -e "${YELLOW}Remember to clean up after use!${NC}"
}

# Clean up secrets files
clean_secrets() {
    echo -e "${YELLOW}Cleaning up secrets files...${NC}"
    
    files_removed=0
    for file in "$SECRETS_FILE" "$SECRETS_FILE.template" ".env.secrets" ".env.prod" ".env.local"; do
        if [[ -f "$file" ]]; then
            rm "$file"
            echo -e "${GREEN}✓ Removed $file${NC}"
            ((files_removed++))
        fi
    done
    
    if [[ $files_removed -eq 0 ]]; then
        echo -e "${YELLOW}No secrets files found to clean${NC}"
    else
        echo -e "${GREEN}✓ Cleaned $files_removed file(s)${NC}"
    fi
}

# Main script
main() {
    case "${1:-}" in
        setup)
            setup_secrets
            ;;
        import)
            import_secrets
            ;;
        export)
            export_template
            ;;
        validate)
            validate_secrets
            ;;
        encrypt)
            encrypt_secrets
            ;;
        decrypt)
            decrypt_secrets
            ;;
        clean)
            clean_secrets
            ;;
        -h|--help|help)
            usage
            ;;
        *)
            echo -e "${RED}Invalid command: ${1:-}${NC}"
            usage
            ;;
    esac
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -r|--repo)
            REPO="$2"
            shift 2
            ;;
        -f|--file)
            SECRETS_FILE="$2"
            shift 2
            ;;
        *)
            break
            ;;
    esac
done

main "$@"