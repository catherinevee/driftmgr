# DriftMgr Build Script (PowerShell)
# This script builds the DriftMgr application

param(
    [string]$Version = "",
    [switch]$Clean
)

# Set error action preference
$ErrorActionPreference = "Stop"

# Colors for output
$Green = "Green"
$Blue = "Blue"
$Red = "Red"

# Function to print colored output
function Write-Status {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor $Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host "[SUCCESS] $Message" -ForegroundColor $Green
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor $Red
}

Write-Status "Starting DriftMgr build..."

# Get version from git if not provided
if (-not $Version) {
    try {
        $Version = git describe --tags --always --dirty 2>$null
        if (-not $Version) {
            $Version = "dev"
        }
    } catch {
        $Version = "dev"
    }
}

$BuildTime = Get-Date -Format "yyyy-MM-dd_HH:mm:ss"
$LDFLAGS = "-X main.Version=$Version -X main.BuildTime=$BuildTime"

Write-Status "Building DriftMgr version $Version..."

# Create bin directory if it doesn't exist
if (-not (Test-Path "bin")) {
    New-Item -ItemType Directory -Path "bin" -Force | Out-Null
}

# Build main CLI
Write-Status "Building main CLI..."
try {
    go build -ldflags $LDFLAGS -o bin/driftmgr.exe ./cmd/driftmgr
    Write-Success "Main CLI built successfully"
} catch {
    Write-Error "Failed to build main CLI: $_"
    exit 1
}

# Build server (if main.go exists)
if (Test-Path "cmd/driftmgr-server/main.go") {
    Write-Status "Building server..."
    try {
        go build -ldflags $LDFLAGS -o bin/driftmgr-server.exe ./cmd/driftmgr-server
        Write-Success "Server built successfully"
    } catch {
        Write-Error "Failed to build server: $_"
        exit 1
    }
}

# Build client (if main.go exists)
if (Test-Path "cmd/driftmgr-client/main.go") {
    Write-Status "Building client..."
    try {
        go build -ldflags $LDFLAGS -o bin/driftmgr-client.exe ./cmd/driftmgr-client
        Write-Success "Client built successfully"
    } catch {
        Write-Error "Failed to build client: $_"
        exit 1
    }
}

# Build agent (if main.go exists)
if (Test-Path "cmd/driftmgr-agent/main.go") {
    Write-Status "Building agent..."
    try {
        go build -ldflags $LDFLAGS -o bin/driftmgr-agent.exe ./cmd/driftmgr-agent
        Write-Success "Agent built successfully"
    } catch {
        Write-Error "Failed to build agent: $_"
        exit 1
    }
}

Write-Success "Build complete! Binaries are in the bin/ directory."
Write-Status "Built version: $Version"
Write-Status "Build time: $BuildTime"
