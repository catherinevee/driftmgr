#!/bin/bash
#
# DriftMgr GitHub Repository Setup Script
# This script automates the initial GitHub configuration for DriftMgr
#
# Usage: ./scripts/setup-github.sh
#
# Prerequisites:
#   - GitHub CLI (gh) installed and authenticated
#   - Repository already exists on GitHub
#   - Admin access to the repository
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REPO="catherinevee/driftmgr"
REPO_OWNER="catherinevee"
REPO_NAME="driftmgr"

# Function to print colored output
print_status() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to check if secret exists
secret_exists() {
    gh secret list | grep -q "^$1" 2>/dev/null
}

# Header
echo "================================================"
echo "       DriftMgr GitHub Repository Setup"
echo "================================================"
echo ""

# Check prerequisites
print_info "Checking prerequisites..."

if ! command_exists gh; then
    print_error "GitHub CLI (gh) is not installed"
    echo "Install from: https://cli.github.com"
    exit 1
fi

# Check authentication
if ! gh auth status >/dev/null 2>&1; then
    print_warning "Not authenticated with GitHub"
    echo "Running: gh auth login"
    gh auth login
fi

print_status "GitHub CLI authenticated"

# Verify repository access
if ! gh repo view "$REPO" >/dev/null 2>&1; then
    print_error "Cannot access repository: $REPO"
    echo "Please check repository name and your access permissions"
    exit 1
fi

print_status "Repository access confirmed"
echo ""

# Main setup menu
echo "What would you like to configure?"
echo "1) Essential secrets (Codecov, Docker Hub)"
echo "2) Optional integrations (Slack, Email, Snyk)"
echo "3) Repository settings"
echo "4) Branch protection rules"
echo "5) Everything (recommended for first setup)"
echo "6) Exit"
echo ""
read -p "Enter your choice (1-6): " choice

case $choice in
    1|5)
        echo ""
        echo "=== Configuring Essential Secrets ==="
        echo ""
        
        # Codecov Token
        if secret_exists "CODECOV_TOKEN"; then
            print_warning "CODECOV_TOKEN already exists"
            read -p "Do you want to update it? (y/n): " update_codecov
        else
            update_codecov="y"
        fi
        
        if [[ "$update_codecov" == "y" ]]; then
            echo "Setting up Codecov:"
            echo "1. Go to: https://app.codecov.io/gh/$REPO"
            echo "2. Click 'Setup repo' if first time"
            echo "3. Copy the repository upload token"
            echo ""
            read -s -p "Enter Codecov token (hidden): " codecov_token
            echo ""
            
            if [ -n "$codecov_token" ]; then
                echo "$codecov_token" | gh secret set CODECOV_TOKEN
                print_status "CODECOV_TOKEN configured"
            else
                print_warning "Skipped CODECOV_TOKEN"
            fi
        fi
        
        # Docker Hub
        if secret_exists "DOCKER_HUB_USERNAME"; then
            print_warning "Docker Hub secrets already exist"
            read -p "Do you want to update them? (y/n): " update_docker
        else
            update_docker="y"
        fi
        
        if [[ "$update_docker" == "y" ]]; then
            echo ""
            echo "Setting up Docker Hub:"
            echo "Create access token at: https://hub.docker.com/settings/security"
            echo ""
            read -p "Enter Docker Hub username [catherinevee]: " docker_user
            docker_user=${docker_user:-catherinevee}
            read -s -p "Enter Docker Hub access token (hidden): " docker_token
            echo ""
            
            if [ -n "$docker_token" ]; then
                echo "$docker_user" | gh secret set DOCKER_HUB_USERNAME
                echo "$docker_token" | gh secret set DOCKER_HUB_TOKEN
                print_status "Docker Hub configured"
            else
                print_warning "Skipped Docker Hub"
            fi
        fi
        ;;&  # Continue to next case if choice was 5
    
    2|5)
        if [[ "$choice" == "2" ]] || [[ "$choice" == "5" ]]; then
            echo ""
            echo "=== Configuring Optional Integrations ==="
            echo ""
            
            # Slack
            read -p "Configure Slack notifications? (y/n): " setup_slack
            if [[ "$setup_slack" == "y" ]]; then
                echo "Create webhook at: https://api.slack.com/apps"
                read -p "Enter Slack webhook URL: " slack_webhook
                if [ -n "$slack_webhook" ]; then
                    echo "$slack_webhook" | gh secret set SLACK_WEBHOOK_URL
                    print_status "Slack configured"
                fi
            fi
            
            # Email
            read -p "Configure email notifications? (y/n): " setup_email
            if [[ "$setup_email" == "y" ]]; then
                read -p "Enter SMTP server [smtp.gmail.com]: " smtp_server
                smtp_server=${smtp_server:-smtp.gmail.com}
                read -p "Enter SMTP username: " smtp_user
                read -s -p "Enter SMTP password/app password (hidden): " smtp_pass
                echo ""
                read -p "Enter notification recipient email: " email_to
                read -p "Enter sender email: " email_from
                
                if [ -n "$smtp_pass" ]; then
                    echo "$smtp_server" | gh secret set SMTP_SERVER
                    echo "$smtp_user" | gh secret set SMTP_USERNAME
                    echo "$smtp_pass" | gh secret set SMTP_PASSWORD
                    echo "$email_to" | gh secret set NOTIFICATION_EMAIL_TO
                    echo "$email_from" | gh secret set NOTIFICATION_EMAIL_FROM
                    print_status "Email notifications configured"
                fi
            fi
            
            # Snyk
            read -p "Configure Snyk security scanning? (y/n): " setup_snyk
            if [[ "$setup_snyk" == "y" ]]; then
                echo "Get token from: https://app.snyk.io/account"
                read -s -p "Enter Snyk API token (hidden): " snyk_token
                echo ""
                if [ -n "$snyk_token" ]; then
                    echo "$snyk_token" | gh secret set SNYK_TOKEN
                    print_status "Snyk configured"
                fi
            fi
        fi
        ;;&
    
    3|5)
        if [[ "$choice" == "3" ]] || [[ "$choice" == "5" ]]; then
            echo ""
            echo "=== Configuring Repository Settings ==="
            echo ""
            
            print_info "Updating repository settings..."
            
            # Update repository settings
            gh api -X PATCH "/repos/$REPO" \
                -f has_issues=true \
                -f has_projects=false \
                -f has_wiki=false \
                -f allow_squash_merge=true \
                -f allow_merge_commit=true \
                -f allow_rebase_merge=true \
                -f delete_branch_on_merge=true \
                -f allow_auto_merge=true \
                >/dev/null 2>&1 && print_status "Repository settings updated" || print_warning "Some settings may require manual configuration"
            
            # Enable security features
            print_info "Enabling security features..."
            
            # Enable dependency graph
            gh api -X PUT "/repos/$REPO/vulnerability-alerts" >/dev/null 2>&1 && \
                print_status "Vulnerability alerts enabled" || print_warning "Vulnerability alerts may already be enabled"
            
            # Enable Dependabot
            gh api -X PUT "/repos/$REPO/automated-security-fixes" >/dev/null 2>&1 && \
                print_status "Automated security fixes enabled" || print_warning "Automated security fixes may already be enabled"
        fi
        ;;&
    
    4|5)
        if [[ "$choice" == "4" ]] || [[ "$choice" == "5" ]]; then
            echo ""
            echo "=== Branch Protection Rules ==="
            echo ""
            
            read -p "Configure branch protection for 'main'? (y/n): " setup_protection
            if [[ "$setup_protection" == "y" ]]; then
                print_info "Setting up branch protection for 'main'..."
                
                # Create branch protection rule
                gh api -X PUT "/repos/$REPO/branches/main/protection" \
                    -f required_status_checks='{"strict":true,"contexts":["build (ubuntu-latest, 1.23)","unit-tests (ubuntu-latest, 1.23)","security-status"]}' \
                    -f enforce_admins=false \
                    -f required_pull_request_reviews='{"required_approving_review_count":1,"dismiss_stale_reviews":true}' \
                    -f restrictions=null \
                    -f allow_force_pushes=false \
                    -f allow_deletions=false \
                    >/dev/null 2>&1 && print_status "Branch protection configured" || print_warning "Branch protection requires manual configuration in Settings → Branches"
            fi
        fi
        ;;
    
    6)
        echo "Exiting..."
        exit 0
        ;;
    
    *)
        print_error "Invalid choice"
        exit 1
        ;;
esac

echo ""
echo "================================================"
echo "              Setup Summary"
echo "================================================"
echo ""

# Show configured secrets
print_info "Configured secrets:"
gh secret list | while read -r secret rest; do
    echo "  • $secret"
done

echo ""

# Show next steps
print_info "Next steps:"
echo "  1. Push code to trigger workflows"
echo "  2. Check Actions tab for workflow runs"
echo "  3. Verify badges in README are working"
echo "  4. Configure additional settings in GitHub UI if needed"
echo ""

# Verify everything
read -p "Would you like to trigger a test workflow run? (y/n): " test_run
if [[ "$test_run" == "y" ]]; then
    print_info "Triggering test workflow..."
    gh workflow run build.yml && print_status "Workflow triggered! Check: https://github.com/$REPO/actions"
fi

echo ""
print_status "Setup complete!"
echo ""
echo "Repository: https://github.com/$REPO"
echo "Actions: https://github.com/$REPO/actions"
echo "Settings: https://github.com/$REPO/settings"
echo ""