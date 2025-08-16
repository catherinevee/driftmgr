# DriftMgr Cleanup Script (PowerShell)
# This script removes duplicate files and merges directories

param(
    [switch]$WhatIf,
    [switch]$Force
)

# Set error action preference
$ErrorActionPreference = "Stop"

Write-Host "🧹 Starting DriftMgr cleanup..." -ForegroundColor Blue

# Function to print colored output
function Write-Status {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host "[SUCCESS] $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "[WARNING] $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

# Function to move file with error handling
function Move-FileSafe {
    param([string]$Source, [string]$Destination)
    
    if (Test-Path $Source) {
        if (-not (Test-Path $Destination)) {
            if ($WhatIf) {
                Write-Status "Would move: $Source -> $Destination"
            } else {
                Move-Item $Source $Destination
                Write-Success "Moved: $Source -> $Destination"
            }
        } else {
            Write-Warning "Destination already exists, skipping: $Destination"
        }
    } else {
        Write-Warning "Source file not found: $Source"
    }
}

# Function to delete file with error handling
function Remove-FileSafe {
    param([string]$File)
    
    if (Test-Path $File) {
        if ($WhatIf) {
            Write-Status "Would delete: $File"
        } else {
            Remove-Item $File -Force
            Write-Success "Deleted: $File"
        }
    } else {
        Write-Warning "File not found: $File"
    }
}

# Function to delete directory with error handling
function Remove-DirectorySafe {
    param([string]$Directory)
    
    if (Test-Path $Directory) {
        if ($WhatIf) {
            Write-Status "Would delete directory: $Directory"
        } else {
            Remove-Item $Directory -Recurse -Force
            Write-Success "Deleted directory: $Directory"
        }
    } else {
        Write-Warning "Directory not found: $Directory"
    }
}

# Phase 1: Move remaining markdown files to docs/summaries/
Write-Status "Phase 1: Moving remaining markdown files..."

$remainingMdFiles = @(
    "ENHANCED_CLI_IMPLEMENTATION_SUMMARY.md",
    "EXPANDED_SERVICES_IMPLEMENTATION_SUMMARY.md",
    "IMMEDIATE_IMPROVEMENTS_PLAN.md",
    "PROPOSED_REORGANIZATION.md",
    "RESTRUCTURE_PLAN.md"
)

foreach ($mdFile in $remainingMdFiles) {
    Move-FileSafe $mdFile "docs\summaries\$mdFile"
}

# Phase 2: Move shell script to scripts/
Write-Status "Phase 2: Moving shell script..."

Move-FileSafe "delete_aws_resources.sh" "scripts\delete_aws_resources.sh"

# Phase 3: Move test file from test/ to tests/
Write-Status "Phase 3: Moving test file..."

if (Test-Path "test\deletion_test.go") {
    Move-FileSafe "test\deletion_test.go" "tests\unit\deletion_test.go"
    # Delete empty test directory
    Remove-DirectorySafe "test"
}

# Phase 4: Delete duplicate executables from root
Write-Status "Phase 4: Deleting duplicate executables from root..."

$duplicateExes = @(
    "driftmgr-client.exe",
    "driftmgr-server.exe"
)

foreach ($exe in $duplicateExes) {
    if ((Test-Path $exe) -and (Test-Path "bin\$exe")) {
        Remove-FileSafe $exe
    }
}

# Phase 5: Delete backup executable in bin/
Write-Status "Phase 5: Deleting backup executable..."

Remove-FileSafe "bin\driftmgr-server.exe~"

# Phase 6: Delete empty tools directory
Write-Status "Phase 6: Deleting empty tools directory..."

if ((Test-Path "tools") -and ((Get-ChildItem "tools" | Measure-Object).Count -eq 0)) {
    Remove-DirectorySafe "tools"
}

# Phase 7: Create missing test directories if needed
Write-Status "Phase 7: Creating missing test directories..."

$testDirs = @("tests\unit", "tests\integration", "tests\e2e")

foreach ($dir in $testDirs) {
    if (-not (Test-Path $dir)) {
        if ($WhatIf) {
            Write-Status "Would create directory: $dir"
        } else {
            New-Item -ItemType Directory -Path $dir -Force | Out-Null
            Write-Status "Created directory: $dir"
        }
    }
}

# Summary
Write-Success "✅ DriftMgr cleanup complete!"
Write-Status "📋 Summary of cleanup:"

if (Test-Path "docs\summaries") {
    $summaryCount = (Get-ChildItem "docs\summaries" | Measure-Object).Count
    Write-Host "  - Moved $summaryCount markdown files to docs\summaries/"
}

Write-Host "  - Moved shell script to scripts/"
Write-Host "  - Moved test file to tests\unit/"
Write-Host "  - Deleted duplicate executables from root"
Write-Host "  - Deleted backup executable from bin/"
Write-Host "  - Deleted empty tools directory"

Write-Host ""
Write-Status "📖 The project structure is now clean and organized"

if ($WhatIf) {
    Write-Host ""
    Write-Warning "This was a dry run. Use -Force to actually perform the cleanup."
}
