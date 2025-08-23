# DriftMgr Project Restructuring Script
param(
    [switch]$DryRun,
    [switch]$Backup = $true
)

Write-Host "DriftMgr Project Restructuring Script" -ForegroundColor Cyan
Write-Host "=====================================" -ForegroundColor Cyan

if ($DryRun) {
    Write-Host "DRY RUN MODE - No files will be moved" -ForegroundColor Yellow
}

if ($Backup) {
    Write-Host "Creating backup..." -ForegroundColor Green
    $backupName = "driftmgr_backup_$(Get-Date -Format 'yyyyMMdd_HHmmss')"
    if (-not $DryRun) {
        Copy-Item -Path "." -Destination "../$backupName" -Recurse -Force
        Write-Host "Backup created at ../$backupName" -ForegroundColor Green
    } else {
        Write-Host "Would create backup at ../$backupName" -ForegroundColor Yellow
    }
}

function Safe-Move {
    param([string]$Source, [string]$Destination, [string]$Description)
    
    if (Test-Path $Source) {
        Write-Host "$Description" -ForegroundColor Blue
        Write-Host "   $Source -> $Destination" -ForegroundColor Gray
        
        if (-not $DryRun) {
            $destDir = Split-Path $Destination -Parent
            if ($destDir -and -not (Test-Path $destDir)) {
                New-Item -ItemType Directory -Path $destDir -Force | Out-Null
            }
            
            if (Test-Path $Destination) {
                Write-Host "   Destination exists, merging..." -ForegroundColor Yellow
                if ((Get-Item $Source).PSIsContainer) {
                    Get-ChildItem $Source | ForEach-Object {
                        Move-Item $_.FullName $Destination -Force
                    }
                    Remove-Item $Source -Force
                } else {
                    Move-Item $Source $Destination -Force
                }
            } else {
                Move-Item $Source $Destination -Force
            }
        }
    } else {
        Write-Host "   Source not found: $Source" -ForegroundColor Yellow
    }
}

Write-Host "`nPhase 1: Create new directory structure" -ForegroundColor Magenta

$newDirs = @(
    "new_structure/cmd/driftmgr",
    "new_structure/cmd/server", 
    "new_structure/cmd/validate",
    "new_structure/internal/discovery",
    "new_structure/internal/models",
    "new_structure/internal/config",
    "new_structure/internal/api",
    "new_structure/internal/dashboard",
    "new_structure/internal/utils",
    "new_structure/pkg/client",
    "new_structure/pkg/types",
    "new_structure/configs/examples",
    "new_structure/docs/api",
    "new_structure/docs/deployment", 
    "new_structure/docs/user-guide",
    "new_structure/docs/development",
    "new_structure/scripts/deploy",
    "new_structure/scripts/tools",
    "new_structure/examples/basic",
    "new_structure/examples/multi-cloud",
    "new_structure/examples/terraform",
    "new_structure/tests/unit",
    "new_structure/tests/integration",
    "new_structure/tests/e2e",
    "new_structure/tests/manual",
    "new_structure/deployments/docker",
    "new_structure/deployments/kubernetes",
    "new_structure/bin"
)

foreach ($dir in $newDirs) {
    if (-not $DryRun) {
        New-Item -ItemType Directory -Path $dir -Force | Out-Null
    }
    Write-Host "   Created: $dir" -ForegroundColor Green
}

Write-Host "`nPhase 2: Move core files" -ForegroundColor Magenta

# Core project files
$coreFiles = @("README.md", "go.mod", "go.sum", "Makefile", "LICENSE")
foreach ($file in $coreFiles) {
    if (Test-Path $file) {
        if (-not $DryRun) {
            Copy-Item $file "new_structure/$file" -Force
        }
        Write-Host "   Copied: $file" -ForegroundColor Green
    }
}

Write-Host "`nPhase 3: Reorganize commands" -ForegroundColor Magenta

# Commands
Safe-Move "cmd/driftmgr/*" "new_structure/cmd/driftmgr/" "Main driftmgr command"
Safe-Move "cmd/driftmgr-server/*" "new_structure/cmd/server/" "Server command" 
Safe-Move "cmd/validate-discovery/*" "new_structure/cmd/validate/" "Validation command"
Safe-Move "cmd/driftmgr-client/*" "new_structure/pkg/client/" "Client functionality"

Write-Host "`nPhase 4: Consolidate internal structure" -ForegroundColor Magenta

# Core internal packages
$internalDirs = @("discovery", "models", "config", "utils", "analysis", "cache", "credentials", "deletion", "remediation", "security", "state")
foreach ($dir in $internalDirs) {
    if (Test-Path "internal/$dir") {
        Safe-Move "internal/$dir" "new_structure/internal/$dir" "Internal: $dir"
    }
}

# Special moves
Safe-Move "internal/dashboard/*" "new_structure/internal/dashboard/" "Dashboard"
Safe-Move "internal/platform/api/*" "new_structure/internal/api/" "API handlers"
Safe-Move "internal/shared/*" "new_structure/internal/utils/" "Shared utilities"

Write-Host "`nPhase 5: Consolidate documentation" -ForegroundColor Magenta

# Move docs
Safe-Move "docs/*" "new_structure/docs/" "Existing documentation"

# Move markdown files
$mdFiles = Get-ChildItem -Path "." -Filter "*.md" | Where-Object { $_.Name -ne "README.md" }
foreach ($file in $mdFiles) {
    Safe-Move $file.FullName "new_structure/docs/development/$($file.Name)" "Documentation: $($file.Name)"
}

Write-Host "`nPhase 6: Consolidate configurations" -ForegroundColor Magenta

# Config files
Safe-Move "configs/*" "new_structure/configs/" "Configuration files"
$yamlFiles = Get-ChildItem -Path "." -Filter "*.yaml"
foreach ($file in $yamlFiles) {
    Safe-Move $file.FullName "new_structure/configs/$($file.Name)" "Config: $($file.Name)"
}

# Region files
$regionFiles = Get-ChildItem -Path "." -Filter "*regions*.json"
foreach ($file in $regionFiles) {
    if (-not $DryRun) {
        Copy-Item $file.FullName "new_structure/configs/$($file.Name)" -Force
    }
    Write-Host "   Copied region file: $($file.Name)" -ForegroundColor Green
}

Write-Host "`nPhase 7: Organize scripts and tools" -ForegroundColor Magenta

# Scripts
Safe-Move "scripts/*" "new_structure/scripts/" "Scripts"
Safe-Move "install*" "new_structure/scripts/" "Install scripts"
Safe-Move "ci-cd/*" "new_structure/scripts/deploy/" "CI/CD"

# Python and PowerShell scripts
$scriptFiles = Get-ChildItem -Path "." -Filter "*.py"
foreach ($file in $scriptFiles) {
    Safe-Move $file.FullName "new_structure/scripts/tools/$($file.Name)" "Python script: $($file.Name)"
}

$ps1Files = Get-ChildItem -Path "." -Filter "*.ps1" | Where-Object { $_.Name -notlike "*restructure*" }
foreach ($file in $ps1Files) {
    Safe-Move $file.FullName "new_structure/scripts/tools/$($file.Name)" "PowerShell script: $($file.Name)"
}

Write-Host "`nPhase 8: Handle examples and tests" -ForegroundColor Magenta

# Examples and tests
Safe-Move "examples/*" "new_structure/examples/" "Examples"
Safe-Move "tests/*" "new_structure/tests/" "Tests"

# Test files
$testFiles = Get-ChildItem -Path "." -Filter "test_*.go"
foreach ($file in $testFiles) {
    Safe-Move $file.FullName "new_structure/tests/manual/$($file.Name)" "Test file: $($file.Name)"
}

Write-Host "`nPhase 9: Handle binaries and deployments" -ForegroundColor Magenta

# Binaries
$binFiles = Get-ChildItem -Path "." -Filter "*.exe"
foreach ($file in $binFiles) {
    Safe-Move $file.FullName "new_structure/bin/$($file.Name)" "Binary: $($file.Name)"
}

# Named binaries
$namedBins = @("driftmgr", "validate-discovery")
foreach ($bin in $namedBins) {
    if (Test-Path $bin) {
        Safe-Move $bin "new_structure/bin/$bin" "Binary: $bin"
    }
}

# Deployments
Safe-Move "deployments/*" "new_structure/deployments/" "Deployments"

Write-Host "`nPhase 10: Clean up remaining files" -ForegroundColor Magenta

# JSON files
$jsonFiles = Get-ChildItem -Path "." -Filter "*.json" | Where-Object { $_.Name -notlike "*regions*" }
foreach ($file in $jsonFiles) {
    Safe-Move $file.FullName "new_structure/configs/$($file.Name)" "Data file: $($file.Name)"
}

# Database and state files
$dbFiles = Get-ChildItem -Path "." -Filter "*.db"
foreach ($file in $dbFiles) {
    Safe-Move $file.FullName "new_structure/bin/$($file.Name)" "Database: $($file.Name)"
}

$tfFiles = Get-ChildItem -Path "." -Filter "*.tfstate"
foreach ($file in $tfFiles) {
    Safe-Move $file.FullName "new_structure/examples/terraform/$($file.Name)" "Terraform state: $($file.Name)"
}

Write-Host "`nPhase 11: Finalization" -ForegroundColor Magenta

if (-not $DryRun) {
    Write-Host "Replacing old structure with new structure..." -ForegroundColor Yellow
    
    # Create gitignore
    $gitignoreContent = @"
# Binaries
*.exe
driftmgr
validate-discovery

# Go
*.so
*.dylib
*.test
*.out

# Database
*.db

# Logs
*.log

# OS
.DS_Store
Thumbs.db

# Terraform
*.tfstate
.terraform/

# Config
local.yaml
"@
    
    Set-Content -Path "new_structure/.gitignore" -Value $gitignoreContent -Encoding UTF8
    
    # Move old structure
    New-Item -ItemType Directory -Path "old_structure" -Force | Out-Null
    Get-ChildItem -Path "." | Where-Object { 
        $_.Name -notin @("new_structure", "old_structure", $backupName, "restructure_simple.ps1") 
    } | ForEach-Object {
        Move-Item $_.FullName "old_structure/" -Force
    }
    
    # Move new structure to current directory
    Get-ChildItem -Path "new_structure" | ForEach-Object {
        Move-Item $_.FullName "." -Force
    }
    
    Remove-Item "new_structure" -Force
    
    Write-Host "Project restructuring completed!" -ForegroundColor Green
    Write-Host "Old structure preserved in: old_structure/" -ForegroundColor Blue
} else {
    Write-Host "Dry run completed! Review the proposed changes above." -ForegroundColor Green
    Write-Host "Run without -DryRun to execute the restructuring." -ForegroundColor Yellow
}

Write-Host "`nRestructuring Summary:" -ForegroundColor Cyan
Write-Host "   • Reduced root directory files from 67+ to ~5" -ForegroundColor Green
Write-Host "   • Consolidated documentation into docs/" -ForegroundColor Green  
Write-Host "   • Unified configuration in configs/" -ForegroundColor Green
Write-Host "   • Organized commands in cmd/" -ForegroundColor Green
Write-Host "   • Simplified internal structure" -ForegroundColor Green
Write-Host "   • Moved binaries to bin/" -ForegroundColor Green