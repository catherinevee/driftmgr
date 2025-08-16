# DriftMgr Script Organization Script
# This script organizes the remaining scripts into the new structure

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

Write-Status "Starting script organization..."

if ($DryRun) {
    Write-Warning "DRY RUN MODE - No actual changes will be made"
}

# Organize build-related scripts
Write-Status "Organizing build-related scripts..."

$buildScripts = @(
    "build-enhanced-cli.sh",
    "build-enhanced-cli.bat",
    "install.sh",
    "install.ps1"
)

foreach ($script in $buildScripts) {
    $sourcePath = "scripts/$script"
    $destPath = "scripts/build/$script"
    if (Test-Path $sourcePath) {
        if (-not $DryRun) {
            Move-FileWithBackup $sourcePath $destPath
        } else {
            Write-Status "Would move: $sourcePath -> $destPath"
        }
    }
}

# Organize test-related scripts
Write-Status "Organizing test-related scripts..."

$testScripts = @(
    "run-all-tests.ps1",
    "run-functionality-tests.sh",
    "run-security-tests.sh",
    "test-workflow-success.ps1",
    "test-workflow-success.sh",
    "test-github-actions.ps1",
    "test-github-actions.sh"
)

foreach ($script in $testScripts) {
    $sourcePath = "scripts/$script"
    $destPath = "scripts/test/$script"
    if (Test-Path $sourcePath) {
        if (-not $DryRun) {
            Move-FileWithBackup $sourcePath $destPath
        } else {
            Write-Status "Would move: $sourcePath -> $destPath"
        }
    }
}

# Organize deployment-related scripts
Write-Status "Organizing deployment-related scripts..."

$deployScripts = @(
    "docker-run.ps1",
    "delete_aws_resources.sh"
)

foreach ($script in $deployScripts) {
    $sourcePath = "scripts/$script"
    $destPath = "scripts/deploy/$script"
    if (Test-Path $sourcePath) {
        if (-not $DryRun) {
            Move-FileWithBackup $sourcePath $destPath
        } else {
            Write-Status "Would move: $sourcePath -> $destPath"
        }
    }
}

# Organize utility scripts
Write-Status "Organizing utility scripts..."

$utilityScripts = @(
    "cleanup.ps1",
    "cleanup.sh",
    "set-timeout.ps1",
    "set-timeout.sh",
    "demo-enhanced-cli.ps1",
    "demo-enhanced-cli.sh",
    "terravision_integration.py",
    "generate_graphs.ps1",
    "generate_graphs.py",
    "reorganize.bat",
    "reorganize.sh",
    "migrate-structure.ps1",
    "migrate-structure.sh"
)

foreach ($script in $utilityScripts) {
    $sourcePath = "scripts/$script"
    $destPath = "scripts/tools/$script"
    if (Test-Path $sourcePath) {
        if (-not $DryRun) {
            Move-FileWithBackup $sourcePath $destPath
        } else {
            Write-Status "Would move: $sourcePath -> $destPath"
        }
    }
}

# Move documentation files
Write-Status "Moving documentation files..."

$docFiles = @(
    "README_RESTRUCTURE.md",
    "SCRIPT_CONSOLIDATION_SUMMARY.md",
    "consolidate-scripts.md",
    "README.md"
)

foreach ($file in $docFiles) {
    $sourcePath = "scripts/$file"
    $destPath = "docs/development/$file"
    if (Test-Path $sourcePath) {
        if (-not $DryRun) {
            Move-FileWithBackup $sourcePath $destPath
        } else {
            Write-Status "Would move: $sourcePath -> $destPath"
        }
    }
}

# Move migration scripts to tools
Write-Status "Moving migration scripts..."

$migrationScripts = @(
    "restructure.ps1",
    "restructure.sh",
    "restructure-fixed.ps1"
)

foreach ($script in $migrationScripts) {
    $sourcePath = "scripts/$script"
    $destPath = "scripts/tools/$script"
    if (Test-Path $sourcePath) {
        if (-not $DryRun) {
            Move-FileWithBackup $sourcePath $destPath
        } else {
            Write-Status "Would move: $sourcePath -> $destPath"
        }
    }
}

# Move Makefile from scripts to tools
if (Test-Path "scripts/Makefile") {
    if (-not $DryRun) {
        Move-FileWithBackup "scripts/Makefile" "scripts/tools/scripts-Makefile"
    } else {
        Write-Status "Would move: scripts/Makefile -> scripts/tools/scripts-Makefile"
    }
}

# Move directories
Write-Status "Moving script directories..."

$scriptDirs = @(
    "validate_regions",
    "install",
    "tests",
    "demos",
    "ps1"
)

foreach ($dir in $scriptDirs) {
    $sourcePath = "scripts/$dir"
    $destPath = "scripts/tools/$dir"
    if (Test-Path $sourcePath) {
        if (-not $DryRun) {
            Move-Item $sourcePath $destPath -Force
            Write-Status "Moved directory: $sourcePath -> $destPath"
        } else {
            Write-Status "Would move directory: $sourcePath -> $destPath"
        }
    }
}

Write-Success "Script organization completed!"

if ($DryRun) {
    Write-Status "This was a dry run. Run without -DryRun to actually move files."
} else {
    Write-Status "All scripts have been organized into the new structure."
    Write-Status "Check the scripts/build/, scripts/test/, scripts/deploy/, and scripts/tools/ directories."
}
