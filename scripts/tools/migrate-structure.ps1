# DriftMgr File Structure Migration Script (PowerShell)
# This script helps reorganize the project structure according to Go best practices

param(
    [switch]$DryRun,
    [switch]$Force
)

$ErrorActionPreference = "Stop"

Write-Host "Starting DriftMgr file structure migration..." -ForegroundColor Green

if ($DryRun) {
    Write-Host "DRY RUN MODE - No files will be moved" -ForegroundColor Yellow
}

# Create new directory structure
Write-Host "Creating new directory structure..." -ForegroundColor Cyan

$directories = @(
    "cmd/driftmgr-server",
    "cmd/driftmgr-web",
    "internal/api",
    "internal/drift",
    "internal/state",
    "internal/analysis",
    "internal/discovery",
    "internal/notification",
    "pkg/models",
    "pkg/config",
    "pkg/utils",
    "pkg/providers",
    "web/static",
    "web/templates",
    "web/assets",
    "scripts/build",
    "scripts/deploy",
    "scripts/test",
    "scripts/tools",
    "docs/api",
    "docs/user-guide",
    "docs/deployment",
    "docs/development",
    "docs/examples",
    "examples/terraform",
    "examples/workflows",
    "examples/configurations",
    "configs/default",
    "configs/development",
    "configs/production",
    "tests/unit",
    "tests/integration",
    "tests/e2e",
    "deployments/docker",
    "deployments/kubernetes",
    "deployments/terraform",
    "tools/blast-radius",
    "tools/state-analyzer",
    "tools/drift-visualizer",
    "bin",
    "dist"
)

foreach ($dir in $directories) {
    if (!(Test-Path $dir)) {
        if (!$DryRun) {
            New-Item -ItemType Directory -Path $dir -Force | Out-Null
        }
        Write-Host "  Created: $dir" -ForegroundColor Gray
    }
}

Write-Host "Directory structure created" -ForegroundColor Green

# Function to move files safely
function Move-FileSafely {
    param(
        [string]$Source,
        [string]$Destination,
        [string]$Description
    )
    
    if (Test-Path $Source) {
        if (!$DryRun) {
            Move-Item -Path $Source -Destination $Destination -Force
        }
        Write-Host "  Moved: $Source -> $Destination" -ForegroundColor Gray
    }
}

# Move executables
Write-Host "Moving executables..." -ForegroundColor Cyan
Move-FileSafely -Source "driftmgr.exe" -Destination "bin/" -Description "Main executable"
Move-FileSafely -Source "cmd/driftmgr.exe" -Destination "bin/" -Description "Cmd executable"

# Move scripts from root to scripts/
Write-Host "Moving scripts..." -ForegroundColor Cyan
$scripts = Get-ChildItem -Path "." -Filter "*.ps1" -File
$scripts += Get-ChildItem -Path "." -Filter "*.sh" -File

foreach ($script in $scripts) {
    $destination = switch -Wildcard ($script.Name) {
        "*demo*" { "scripts/test/" }
        "*test*" { "scripts/test/" }
        "*build*" { "scripts/build/" }
        "*compile*" { "scripts/build/" }
        "*deploy*" { "scripts/deploy/" }
        "*docker*" { "scripts/deploy/" }
        "*kubernetes*" { "scripts/deploy/" }
        default { "scripts/tools/" }
    }
    
    Move-FileSafely -Source $script.FullName -Destination $destination -Description "Script"
}

# Move documentation from root to docs/
Write-Host "Moving documentation..." -ForegroundColor Cyan
$docs = Get-ChildItem -Path "." -Filter "*.md" -File | Where-Object { $_.Name -ne "README.md" }

foreach ($doc in $docs) {
    $destination = switch -Wildcard ($doc.Name) {
        "*API*" { "docs/api/" }
        "*api*" { "docs/api/" }
        "*SECURITY*" { "docs/development/" }
        "*security*" { "docs/development/" }
        "*CISCO*" { "docs/user-guide/" }
        "*CONTEXT*" { "docs/user-guide/" }
        default { "docs/" }
    }
    
    Move-FileSafely -Source $doc.FullName -Destination $destination -Description "Documentation"
}

# Move web files
Write-Host "Moving web files..." -ForegroundColor Cyan
Move-FileSafely -Source "web/main.go" -Destination "cmd/driftmgr-web/" -Description "Web main"
Move-FileSafely -Source "web/server.go" -Destination "cmd/driftmgr-web/" -Description "Web server"

# Move shared models
Write-Host "Moving shared models..." -ForegroundColor Cyan
if (Test-Path "shared/models") {
    $sharedModels = Get-ChildItem -Path "shared/models" -File
    foreach ($file in $sharedModels) {
        Move-FileSafely -Source $file.FullName -Destination "pkg/models/" -Description "Shared model"
    }
    if (!$DryRun) {
        Remove-Item -Path "shared/models" -Force -ErrorAction SilentlyContinue
    }
}

if (Test-Path "shared/config") {
    $sharedConfig = Get-ChildItem -Path "shared/config" -File
    foreach ($file in $sharedConfig) {
        Move-FileSafely -Source $file.FullName -Destination "pkg/config/" -Description "Shared config"
    }
    if (!$DryRun) {
        Remove-Item -Path "shared/config" -Force -ErrorAction SilentlyContinue
    }
}

# Move tools
Write-Host "Moving tools..." -ForegroundColor Cyan
if (Test-Path "tools") {
    $tools = Get-ChildItem -Path "tools" -Filter "*.py" -File
    foreach ($tool in $tools) {
        $destination = switch -Wildcard ($tool.Name) {
            "*blast*" { "tools/blast-radius/" }
            "*tfstate*" { "tools/state-analyzer/" }
            default { "tools/drift-visualizer/" }
        }
        
        Move-FileSafely -Source $tool.FullName -Destination $destination -Description "Tool"
    }
}

# Move services to internal
Write-Host "Moving services..." -ForegroundColor Cyan
if (Test-Path "services") {
    $services = Get-ChildItem -Path "services" -Directory
    foreach ($service in $services) {
        $destination = switch -Wildcard ($service.Name) {
            "*api*" { "internal/api/" }
            "*gateway*" { "internal/api/" }
            "*analysis*" { "internal/analysis/" }
            "*discovery*" { "internal/discovery/" }
            "*notification*" { "internal/notification/" }
            "*state*" { "internal/state/" }
            "*visualization*" { "internal/drift/" }
            default { "internal/" }
        }
        
        $serviceFiles = Get-ChildItem -Path $service.FullName -Recurse -File
        foreach ($file in $serviceFiles) {
            $relativePath = $file.FullName.Substring($service.FullName.Length + 1)
            $newPath = Join-Path $destination $relativePath
            $newDir = Split-Path $newPath -Parent
            
            if (!$DryRun) {
                if (!(Test-Path $newDir)) {
                    New-Item -ItemType Directory -Path $newDir -Force | Out-Null
                }
                Move-Item -Path $file.FullName -Destination $newPath -Force
            }
            Write-Host "  Moved: $($file.FullName) -> $newPath" -ForegroundColor Gray
        }
        
        if (!$DryRun) {
            Remove-Item -Path $service.FullName -Recurse -Force
        }
    }
    
    if (!$DryRun) {
        Remove-Item -Path "services" -Force -ErrorAction SilentlyContinue
    }
}

# Create .gitignore updates
Write-Host "Updating .gitignore..." -ForegroundColor Cyan
$gitignoreLines = @(
    "",
    "# Binaries and distributions",
    "bin/",
    "dist/",
    "",
    "# IDE files",
    ".vscode/",
    ".idea/",
    "*.swp",
    "*.swo",
    "",
    "# OS files",
    ".DS_Store",
    "Thumbs.db",
    "",
    "# Logs",
    "*.log",
    "",
    "# Environment files",
    ".env",
    ".env.local",
    "",
    "# Temporary files",
    "tmp/",
    "temp/"
)

if (!$DryRun) {
    Add-Content -Path ".gitignore" -Value $gitignoreLines
}
Write-Host "  Updated .gitignore" -ForegroundColor Gray

# Create Makefile if it doesn't exist
if (!(Test-Path "Makefile")) {
    Write-Host "Creating Makefile..." -ForegroundColor Cyan
    if (!$DryRun) {
        $makefileLines = @(
            ".PHONY: build clean test lint format help",
            "",
            "# Build targets",
            "build:",
            "	go build -o bin/driftmgr-client ./cmd/driftmgr-client",
            "	go build -o bin/driftmgr-server ./cmd/driftmgr-server",
            "	go build -o bin/driftmgr-web ./cmd/driftmgr-web",
            "",
            "build-client:",
            "	go build -o bin/driftmgr-client ./cmd/driftmgr-client",
            "",
            "build-server:",
            "	go build -o bin/driftmgr-server ./cmd/driftmgr-server",
            "",
            "build-web:",
            "	go build -o bin/driftmgr-web ./cmd/driftmgr-web",
            "",
            "# Clean",
            "clean:",
            "	rm -rf bin/ dist/",
            "",
            "# Test",
            "test:",
            "	go test ./...",
            "",
            "test-unit:",
            "	go test ./tests/unit/...",
            "",
            "test-integration:",
            "	go test ./tests/integration/...",
            "",
            "test-e2e:",
            "	go test ./tests/e2e/...",
            "",
            "# Lint and format",
            "lint:",
            "	golangci-lint run",
            "",
            "format:",
            "	go fmt ./...",
            "	gofmt -s -w .",
            "",
            "# Development",
            "dev:",
            "	go run ./cmd/driftmgr-server",
            "",
            "dev-client:",
            "	go run ./cmd/driftmgr-client",
            "",
            "dev-web:",
            "	go run ./cmd/driftmgr-web",
            "",
            "# Docker",
            "docker-build:",
            "	docker build -t driftmgr .",
            "",
            "docker-run:",
            "	docker run -p 8080:8080 driftmgr",
            "",
            "# Help",
            "help:",
            "	@echo Available targets:",
            "	@echo   build        - Build all binaries",
            "	@echo   build-client - Build client only",
            "	@echo   build-server - Build server only",
            "	@echo   build-web    - Build web interface only",
            "	@echo   clean        - Clean build artifacts",
            "	@echo   test         - Run all tests",
            "	@echo   test-unit    - Run unit tests",
            "	@echo   test-integration - Run integration tests",
            "	@echo   test-e2e     - Run end-to-end tests",
            "	@echo   lint         - Run linter",
            "	@echo   format       - Format code",
            "	@echo   dev          - Run server in development mode",
            "	@echo   dev-client   - Run client in development mode",
            "	@echo   dev-web      - Run web interface in development mode",
            "	@echo   docker-build - Build Docker image",
            "	@echo   docker-run   - Run Docker container"
        )
        Set-Content -Path "Makefile" -Value $makefileLines
    }
    Write-Host "  Created Makefile" -ForegroundColor Gray
}

Write-Host "Migration completed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "1. Review the new structure"
Write-Host "2. Update import paths in Go files"
Write-Host "3. Update documentation references"
Write-Host "4. Test all functionality"
Write-Host "5. Update CI/CD pipelines if needed"
Write-Host ""
Write-Host "See STRUCTURE_IMPROVEMENTS.md for detailed information" -ForegroundColor Cyan

if ($DryRun) {
    Write-Host ""
    Write-Host "This was a dry run. Run without -DryRun to actually perform the migration." -ForegroundColor Yellow
}
