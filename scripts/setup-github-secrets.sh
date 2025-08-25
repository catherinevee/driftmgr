#!/bin/bash

# Script to set up GitHub secrets for Docker workflows
# Requires GitHub CLI (gh) to be installed and authenticated

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if gh CLI is installed
if ! command -v gh &> /dev/null; then
    echo -e "${RED}Error: GitHub CLI (gh) is not installed${NC}"
    echo "Install it from: https://cli.github.com/"
    exit 1
fi

# Check if authenticated
if ! gh auth status &> /dev/null; then
    echo -e "${YELLOW}GitHub CLI is not authenticated. Running 'gh auth login'...${NC}"
    gh auth login
fi

# Get repository name
REPO=$(gh repo view --json nameWithOwner -q .nameWithOwner 2>/dev/null || echo "")
if [ -z "$REPO" ]; then
    echo -e "${YELLOW}Could not detect repository. Please enter repository name (e.g., catherinevee/driftmgr):${NC}"
    read -r REPO
fi

echo -e "${GREEN}Setting up secrets for repository: $REPO${NC}"

# Function to set a secret
set_secret() {
    local secret_name=$1
    local secret_value=$2
    local required=$3
    
    if [ -z "$secret_value" ] && [ "$required" = "true" ]; then
        echo -e "${YELLOW}Enter value for $secret_name (required):${NC}"
        read -rs secret_value
        echo
    elif [ -z "$secret_value" ] && [ "$required" = "false" ]; then
        echo -e "${YELLOW}$secret_name is optional. Press Enter to skip or enter value:${NC}"
        read -rs secret_value
        echo
    fi
    
    if [ -n "$secret_value" ]; then
        echo "$secret_value" | gh secret set "$secret_name" --repo="$REPO"
        echo -e "${GREEN}✓ Set $secret_name${NC}"
    else
        echo -e "${YELLOW}⊘ Skipped $secret_name${NC}"
    fi
}

# Docker Hub credentials
echo -e "\n${GREEN}=== Docker Hub Configuration ===${NC}"
echo "Docker Hub credentials are required to push images to Docker Hub"
echo "Get an access token from: https://hub.docker.com/settings/security"

DOCKER_USERNAME=""
DOCKER_TOKEN=""

echo -e "${YELLOW}Enter Docker Hub username:${NC}"
read -r DOCKER_USERNAME

echo -e "${YELLOW}Enter Docker Hub access token (not password):${NC}"
read -rs DOCKER_TOKEN
echo

if [ -n "$DOCKER_USERNAME" ] && [ -n "$DOCKER_TOKEN" ]; then
    set_secret "DOCKER_HUB_USERNAME" "$DOCKER_USERNAME" false
    set_secret "DOCKER_HUB_TOKEN" "$DOCKER_TOKEN" false
fi

# AWS credentials (optional)
echo -e "\n${GREEN}=== AWS Configuration (Optional) ===${NC}"
echo "AWS credentials are needed if DriftMgr will scan AWS resources from CI/CD"
set_secret "AWS_ACCESS_KEY_ID" "" false
set_secret "AWS_SECRET_ACCESS_KEY" "" false
set_secret "AWS_DEFAULT_REGION" "" false

# Azure credentials (optional)
echo -e "\n${GREEN}=== Azure Configuration (Optional) ===${NC}"
echo "Azure credentials are needed if DriftMgr will scan Azure resources from CI/CD"
set_secret "AZURE_CLIENT_ID" "" false
set_secret "AZURE_CLIENT_SECRET" "" false
set_secret "AZURE_TENANT_ID" "" false
set_secret "AZURE_SUBSCRIPTION_ID" "" false

# GCP credentials (optional)
echo -e "\n${GREEN}=== GCP Configuration (Optional) ===${NC}"
echo "GCP service account JSON is needed if DriftMgr will scan GCP resources from CI/CD"
echo "Paste the entire service account JSON (press Ctrl+D when done):"
GCP_CREDS=$(cat)
if [ -n "$GCP_CREDS" ]; then
    set_secret "GCP_SERVICE_ACCOUNT_JSON" "$GCP_CREDS" false
fi

# Slack webhook (optional)
echo -e "\n${GREEN}=== Slack Notifications (Optional) ===${NC}"
echo "Slack webhook URL for CI/CD notifications"
set_secret "SLACK_WEBHOOK_URL" "" false

# List all secrets
echo -e "\n${GREEN}=== Configured Secrets ===${NC}"
gh secret list --repo="$REPO"

echo -e "\n${GREEN}✓ GitHub secrets setup complete!${NC}"
echo -e "You can manage secrets at: https://github.com/$REPO/settings/secrets/actions"