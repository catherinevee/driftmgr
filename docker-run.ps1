# PowerShell Docker runner script for Terraform Import Helper
# Makes it easy to run driftmgr in Docker with proper volume mounts and environment setup

param(
    [string]$Image = "catherinevee/driftmgr:latest",
    [string]$ConfigDir = "./config",
    [string]$OutputDir = "./output", 
    [string]$InputFile = "",
    [string]$AwsProfile = "default",
    [switch]$Interactive,
    [switch]$Build,
    [switch]$Help,
    [Parameter(ValueFromRemainingArguments)]
    [string[]]$RemainingArgs
)

# Helper functions
function Write-Info {
    param([string]$Message)
    Write-Host "ℹ️  $Message" -ForegroundColor Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host "✅ $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "⚠️  $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "❌ $Message" -ForegroundColor Red
}

function Show-Help {
    @"
🐳 Docker Runner for Terraform Import Helper

Usage: .\docker-run.ps1 [OPTIONS] COMMAND [ARGS...]

OPTIONS:
    -Image IMAGE           Docker image to use (default: catherinevee/driftmgr:latest)
    -ConfigDir DIR         Config directory to mount (default: ./config)
    -OutputDir DIR         Output directory to mount (default: ./output)
    -InputFile FILE        Input file to mount
    -AwsProfile PROFILE    AWS profile to use (default: default)
    -Interactive           Run in interactive mode with TTY
    -Build                 Build image locally before running
    -Help                  Show this help

COMMANDS:
    discover               Discover cloud resources
    import                 Import resources to Terraform
    interactive            Launch interactive TUI
    config                 Manage configuration
    help                   Show driftmgr help
    version                Show version

EXAMPLES:
    # Basic help
    .\docker-run.ps1 help

    # Discover AWS resources
    .\docker-run.ps1 discover --provider aws --region us-east-1

    # Import with file
    .\docker-run.ps1 -InputFile resources.csv import --file /input/resources.csv

    # Interactive mode
    .\docker-run.ps1 -Interactive interactive

    # Build and run locally
    .\docker-run.ps1 -Build version

    # Custom config directory
    .\docker-run.ps1 -ConfigDir C:\path\to\config discover

"@
}

# Show help if requested
if ($Help) {
    Show-Help
    exit 0
}

# Build locally if requested
if ($Build) {
    Write-Info "Building Docker image locally..."
    docker build -t driftmgr:local .
    if ($LASTEXITCODE -eq 0) {
        $Image = "driftmgr:local"
        Write-Success "Image built successfully"
    } else {
        Write-Error "Failed to build image"
        exit 1
    }
}

# Create directories if they don't exist
if (!(Test-Path $ConfigDir)) {
    New-Item -ItemType Directory -Path $ConfigDir -Force | Out-Null
}
if (!(Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
}

# Build Docker run command
$DockerArgs = @("run", "--rm")

# Add interactive flags if needed
if ($Interactive -or ($RemainingArgs -contains "interactive")) {
    $DockerArgs += "-it"
}

# Add volume mounts (convert Windows paths)
$ConfigPath = (Resolve-Path $ConfigDir).Path -replace '\\', '/'
$OutputPath = (Resolve-Path $OutputDir).Path -replace '\\', '/'

$DockerArgs += "-v", "$ConfigPath:/config:ro"
$DockerArgs += "-v", "$OutputPath:/output"

# Add input file mount if specified
if ($InputFile) {
    if (!(Test-Path $InputFile)) {
        Write-Error "Input file not found: $InputFile"
        exit 1
    }
    $InputPath = (Resolve-Path $InputFile).Path -replace '\\', '/'
    $InputName = Split-Path $InputFile -Leaf
    $DockerArgs += "-v", "$InputPath:/input/$InputName:ro"
}

# Add AWS credentials if available
$AwsPath = "$env:USERPROFILE\.aws"
if (Test-Path $AwsPath) {
    $AwsPath = $AwsPath -replace '\\', '/'
    $DockerArgs += "-v", "$AwsPath:/root/.aws:ro"
    $DockerArgs += "-e", "AWS_PROFILE=$AwsProfile"
}

# Add Azure credentials if available
$AzurePath = "$env:USERPROFILE\.azure"
if (Test-Path $AzurePath) {
    $AzurePath = $AzurePath -replace '\\', '/'
    $DockerArgs += "-v", "$AzurePath:/root/.azure:ro"
}

# Add GCP credentials if available
$GcpPath = "$env:APPDATA\gcloud"
if (Test-Path $GcpPath) {
    $GcpPath = $GcpPath -replace '\\', '/'
    $DockerArgs += "-v", "$GcpPath:/root/.config/gcloud:ro"
}

# Add common environment variables
$DockerArgs += "-e", "AWS_REGION=$($env:AWS_REGION ?? 'us-east-1')"
$DockerArgs += "-e", "AZURE_LOCATION=$($env:AZURE_LOCATION ?? 'eastus')"
$DockerArgs += "-e", "GCP_REGION=$($env:GCP_REGION ?? 'us-central1')"

# Add image
$DockerArgs += $Image

# Add remaining arguments as the command
if ($RemainingArgs) {
    $DockerArgs += $RemainingArgs
}

# Show what we're about to run
Write-Info "Running: docker $($DockerArgs -join ' ')"
Write-Host ""

# Execute the command
& docker @DockerArgs
