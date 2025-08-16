# DriftMgr Root File Organization Script
# This script organizes the remaining files in the root directory

param(
    [switch]$DryRun,
    [switch]$Force
)

# Set error action preference
$ErrorActionPreference = "Stop"

# Colors for output
$Red = "Red"
$Green = "Green"
$Yellow = "Yellow"
$Blue = "Blue"
$White = "White"

# Function to print colored output
function Write-Status {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor $Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host "[SUCCESS] $Message" -ForegroundColor $Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "[WARNING] $Message" -ForegroundColor $Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor $Red
}

# Function to move file with backup
function Move-FileWithBackup {
    param(
        [string]$Source,
        [string]$Destination
    )
    
    if (Test-Path $Source) {
        if (Test-Path $Destination) {
            Write-Warning "Destination file exists, creating backup: $Destination.backup"
            Move-Item $Destination "$Destination.backup" -Force
        }
        Move-Item $Source $Destination -Force
        Write-Status "Moved: $Source -> $Destination"
    } else {
        Write-Warning "Source file not found: $Source"
    }
}

Write-Status "Starting root file organization..."

if ($DryRun) {
    Write-Warning "DRY RUN MODE - No actual changes will be made"
}

# Move main.go to cmd/driftmgr/
Write-Status "Moving main.go to cmd/driftmgr/..."
if (Test-Path "main.go") {
    if (-not $DryRun) {
        Move-FileWithBackup "main.go" "cmd/driftmgr/main.go"
    } else {
        Write-Status "Would move: main.go -> cmd/driftmgr/main.go"
    }
}

# Move install.ps1 to scripts/build/
Write-Status "Moving install.ps1 to scripts/build/..."
if (Test-Path "install.ps1") {
    if (-not $DryRun) {
        Move-FileWithBackup "install.ps1" "scripts/build/install.ps1"
    } else {
        Write-Status "Would move: install.ps1 -> scripts/build/install.ps1"
    }
}

# Move documentation files
Write-Status "Moving documentation files..."

$docFiles = @(
    "FILE_STRUCTURE_IMPROVEMENT_PROPOSAL.md",
    "MIGRATION_SUMMARY.md"
)

foreach ($file in $docFiles) {
    if (Test-Path $file) {
        $destPath = "docs/development/$file"
        if (-not $DryRun) {
            Move-FileWithBackup $file $destPath
        } else {
            Write-Status "Would move: $file -> $destPath"
        }
    }
}

# Move static directory to assets
Write-Status "Moving static directory to assets..."
if (Test-Path "static") {
    if (-not $DryRun) {
        # Copy contents to assets/static
        Copy-Item "static/*" "assets/static/" -Recurse -Force -ErrorAction SilentlyContinue
        Remove-Item "static" -Recurse -Force
        Write-Status "Moved static directory contents to assets/static/"
    } else {
        Write-Status "Would move static directory contents to assets/static/"
    }
}

# Move web directory to internal/platform/web
Write-Status "Moving web directory to internal/platform/web..."
if (Test-Path "web") {
    if (-not $DryRun) {
        # Copy contents to internal/platform/web
        Copy-Item "web/*" "internal/platform/web/" -Recurse -Force -ErrorAction SilentlyContinue
        Remove-Item "web" -Recurse -Force
        Write-Status "Moved web directory contents to internal/platform/web/"
    } else {
        Write-Status "Would move web directory contents to internal/platform/web/"
    }
}

# Move config directory to configs
Write-Status "Moving config directory to configs..."
if (Test-Path "config") {
    if (-not $DryRun) {
        # Copy contents to configs
        Copy-Item "config/*" "configs/" -Recurse -Force -ErrorAction SilentlyContinue
        Remove-Item "config" -Recurse -Force
        Write-Status "Moved config directory contents to configs/"
    } else {
        Write-Status "Would move config directory contents to configs/"
    }
}

# Move outputs directory to assets
Write-Status "Moving outputs directory to assets..."
if (Test-Path "outputs") {
    if (-not $DryRun) {
        # Copy contents to assets
        Copy-Item "outputs/*" "assets/" -Recurse -Force -ErrorAction SilentlyContinue
        Remove-Item "outputs" -Recurse -Force
        Write-Status "Moved outputs directory contents to assets/"
    } else {
        Write-Status "Would move outputs directory contents to assets/"
    }
}

# Move logs directory to internal/shared/logging
Write-Status "Moving logs directory to internal/shared/logging..."
if (Test-Path "logs") {
    if (-not $DryRun) {
        # Copy contents to internal/shared/logging
        Copy-Item "logs/*" "internal/shared/logging/" -Recurse -Force -ErrorAction SilentlyContinue
        Remove-Item "logs" -Recurse -Force
        Write-Status "Moved logs directory contents to internal/shared/logging/"
    } else {
        Write-Status "Would move logs directory contents to internal/shared/logging/"
    }
}

# Move microservices directory to examples
Write-Status "Moving microservices directory to examples..."
if (Test-Path "microservices") {
    if (-not $DryRun) {
        # Copy contents to examples
        Copy-Item "microservices/*" "examples/" -Recurse -Force -ErrorAction SilentlyContinue
        Remove-Item "microservices" -Recurse -Force
        Write-Status "Moved microservices directory contents to examples/"
    } else {
        Write-Status "Would move microservices directory contents to examples/"
    }
}

# Move bin directory to deployments
Write-Status "Moving bin directory to deployments..."
if (Test-Path "bin") {
    if (-not $DryRun) {
        # Copy contents to deployments
        Copy-Item "bin/*" "deployments/" -Recurse -Force -ErrorAction SilentlyContinue
        Remove-Item "bin" -Recurse -Force
        Write-Status "Moved bin directory contents to deployments/"
    } else {
        Write-Status "Would move bin directory contents to deployments/"
    }
}

Write-Success "Root file organization completed!"

if ($DryRun) {
    Write-Status "This was a dry run. Run without -DryRun to actually move files."
} else {
    Write-Status "All root files have been organized into the new structure."
    Write-Status "The root directory should now be clean and organized."
}
