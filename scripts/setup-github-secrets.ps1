# PowerShell script to set up GitHub secrets for Docker workflows
# Requires GitHub CLI (gh) to be installed and authenticated

$ErrorActionPreference = "Stop"

# Colors for output
function Write-ColorOutput {
    param([string]$Message, [string]$Color = "White")
    Write-Host $Message -ForegroundColor $Color
}

# Check if gh CLI is installed
try {
    $null = gh --version
} catch {
    Write-ColorOutput "Error: GitHub CLI (gh) is not installed" "Red"
    Write-ColorOutput "Install it from: https://cli.github.com/" "Yellow"
    exit 1
}

# Check if authenticated
try {
    $null = gh auth status 2>$null
} catch {
    Write-ColorOutput "GitHub CLI is not authenticated. Running 'gh auth login'..." "Yellow"
    gh auth login
}

# Get repository name
try {
    $REPO = gh repo view --json nameWithOwner -q .nameWithOwner 2>$null
} catch {
    $REPO = ""
}

if ([string]::IsNullOrEmpty($REPO)) {
    Write-ColorOutput "Could not detect repository. Please enter repository name (e.g., catherinevee/driftmgr):" "Yellow"
    $REPO = Read-Host
}

Write-ColorOutput "Setting up secrets for repository: $REPO" "Green"

# Function to set a secret
function Set-GitHubSecret {
    param(
        [string]$SecretName,
        [string]$SecretValue,
        [bool]$Required = $false,
        [bool]$IsPassword = $true
    )
    
    if ([string]::IsNullOrEmpty($SecretValue)) {
        if ($Required) {
            Write-ColorOutput "Enter value for $SecretName (required):" "Yellow"
        } else {
            Write-ColorOutput "$SecretName is optional. Press Enter to skip or enter value:" "Yellow"
        }
        
        if ($IsPassword) {
            $SecureString = Read-Host -AsSecureString
            $BSTR = [System.Runtime.InteropServices.Marshal]::SecureStringToBSTR($SecureString)
            $SecretValue = [System.Runtime.InteropServices.Marshal]::PtrToStringAuto($BSTR)
        } else {
            $SecretValue = Read-Host
        }
    }
    
    if (![string]::IsNullOrEmpty($SecretValue)) {
        $SecretValue | gh secret set $SecretName --repo="$REPO"
        Write-ColorOutput "✓ Set $SecretName" "Green"
    } else {
        Write-ColorOutput "⊘ Skipped $SecretName" "Yellow"
    }
}

# Docker Hub credentials
Write-ColorOutput "`n=== Docker Hub Configuration ===" "Green"
Write-Host "Docker Hub credentials are required to push images to Docker Hub"
Write-Host "Get an access token from: https://hub.docker.com/settings/security"

Write-ColorOutput "Enter Docker Hub username:" "Yellow"
$DOCKER_USERNAME = Read-Host

Write-ColorOutput "Enter Docker Hub access token (not password):" "Yellow"
$SecureToken = Read-Host -AsSecureString
$BSTR = [System.Runtime.InteropServices.Marshal]::SecureStringToBSTR($SecureToken)
$DOCKER_TOKEN = [System.Runtime.InteropServices.Marshal]::PtrToStringAuto($BSTR)

if (![string]::IsNullOrEmpty($DOCKER_USERNAME) -and ![string]::IsNullOrEmpty($DOCKER_TOKEN)) {
    Set-GitHubSecret -SecretName "DOCKER_HUB_USERNAME" -SecretValue $DOCKER_USERNAME -IsPassword $false
    Set-GitHubSecret -SecretName "DOCKER_HUB_TOKEN" -SecretValue $DOCKER_TOKEN -IsPassword $false
}

# AWS credentials (optional)
Write-ColorOutput "`n=== AWS Configuration (Optional) ===" "Green"
Write-Host "AWS credentials are needed if DriftMgr will scan AWS resources from CI/CD"
Set-GitHubSecret -SecretName "AWS_ACCESS_KEY_ID" -SecretValue "" -Required $false -IsPassword $false
Set-GitHubSecret -SecretName "AWS_SECRET_ACCESS_KEY" -SecretValue "" -Required $false
Set-GitHubSecret -SecretName "AWS_DEFAULT_REGION" -SecretValue "" -Required $false -IsPassword $false

# Azure credentials (optional)
Write-ColorOutput "`n=== Azure Configuration (Optional) ===" "Green"
Write-Host "Azure credentials are needed if DriftMgr will scan Azure resources from CI/CD"
Set-GitHubSecret -SecretName "AZURE_CLIENT_ID" -SecretValue "" -Required $false -IsPassword $false
Set-GitHubSecret -SecretName "AZURE_CLIENT_SECRET" -SecretValue "" -Required $false
Set-GitHubSecret -SecretName "AZURE_TENANT_ID" -SecretValue "" -Required $false -IsPassword $false
Set-GitHubSecret -SecretName "AZURE_SUBSCRIPTION_ID" -SecretValue "" -Required $false -IsPassword $false

# GCP credentials (optional)
Write-ColorOutput "`n=== GCP Configuration (Optional) ===" "Green"
Write-Host "GCP service account JSON is needed if DriftMgr will scan GCP resources from CI/CD"
Write-Host "Enter path to service account JSON file (or press Enter to skip):"
$GCP_PATH = Read-Host
if (![string]::IsNullOrEmpty($GCP_PATH) -and (Test-Path $GCP_PATH)) {
    $GCP_CREDS = Get-Content $GCP_PATH -Raw
    Set-GitHubSecret -SecretName "GCP_SERVICE_ACCOUNT_JSON" -SecretValue $GCP_CREDS -IsPassword $false
}

# Slack webhook (optional)
Write-ColorOutput "`n=== Slack Notifications (Optional) ===" "Green"
Write-Host "Slack webhook URL for CI/CD notifications"
Set-GitHubSecret -SecretName "SLACK_WEBHOOK_URL" -SecretValue "" -Required $false -IsPassword $false

# List all secrets
Write-ColorOutput "`n=== Configured Secrets ===" "Green"
gh secret list --repo="$REPO"

Write-ColorOutput "`n✓ GitHub secrets setup complete!" "Green"
Write-ColorOutput "You can manage secrets at: https://github.com/$REPO/settings/secrets/actions" "Cyan"