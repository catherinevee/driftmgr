# DriftMgr File Structure Migration Script (PowerShell)
# This script helps migrate from the current structure to the improved structure

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

# Function to create directory if it doesn't exist
function New-DirectoryIfNotExists {
    param([string]$Path)
    
    if (-not (Test-Path $Path)) {
        New-Item -ItemType Directory -Path $Path -Force | Out-Null
        Write-Status "Created directory: $Path"
    }
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

# Function to copy file if it doesn't exist
function Copy-FileIfNotExists {
    param(
        [string]$Source,
        [string]$Destination
    )
    
    if ((Test-Path $Source) -and (-not (Test-Path $Destination))) {
        Copy-Item $Source $Destination -Force
        Write-Status "Copied: $Source -> $Destination"
    } elseif (-not (Test-Path $Source)) {
        Write-Warning "Source file not found: $Source"
    }
}

Write-Status "Starting DriftMgr file structure migration..."

if ($DryRun) {
    Write-Warning "DRY RUN MODE - No actual changes will be made"
}

# Create new directory structure
Write-Status "Creating new directory structure..."

# API layer
New-DirectoryIfNotExists "api/v1"
New-DirectoryIfNotExists "api/docs"

# Assets
New-DirectoryIfNotExists "assets/regions"
New-DirectoryIfNotExists "assets/configs"
New-DirectoryIfNotExists "assets/templates/terraform"
New-DirectoryIfNotExists "assets/templates/reports"
New-DirectoryIfNotExists "assets/static/css"
New-DirectoryIfNotExists "assets/static/js"
New-DirectoryIfNotExists "assets/static/images"

# Commands
New-DirectoryIfNotExists "cmd/driftmgr/commands"
New-DirectoryIfNotExists "cmd/driftmgr-server/server"
New-DirectoryIfNotExists "cmd/driftmgr-agent/agent"

# Configurations
New-DirectoryIfNotExists "configs"

# Deployments
New-DirectoryIfNotExists "deployments/docker"
New-DirectoryIfNotExists "deployments/kubernetes"
New-DirectoryIfNotExists "deployments/terraform"

# Documentation
New-DirectoryIfNotExists "docs/api"
New-DirectoryIfNotExists "docs/deployment"
New-DirectoryIfNotExists "docs/development"
New-DirectoryIfNotExists "docs/user-guide"
New-DirectoryIfNotExists "docs/architecture"

# Examples
New-DirectoryIfNotExists "examples/basic"
New-DirectoryIfNotExists "examples/advanced"
New-DirectoryIfNotExists "examples/multi-cloud"
New-DirectoryIfNotExists "examples/custom-plugins"

# Internal structure
New-DirectoryIfNotExists "internal/core/discovery/providers"
New-DirectoryIfNotExists "internal/core/analysis/detectors"
New-DirectoryIfNotExists "internal/core/remediation/strategies"
New-DirectoryIfNotExists "internal/core/workflow"

New-DirectoryIfNotExists "internal/platform/api/handlers"
New-DirectoryIfNotExists "internal/platform/api/middleware"
New-DirectoryIfNotExists "internal/platform/web/templates"
New-DirectoryIfNotExists "internal/platform/web/static"
New-DirectoryIfNotExists "internal/platform/cli"
New-DirectoryIfNotExists "internal/platform/storage"

New-DirectoryIfNotExists "internal/shared/config"
New-DirectoryIfNotExists "internal/shared/logging"
New-DirectoryIfNotExists "internal/shared/metrics"
New-DirectoryIfNotExists "internal/shared/security"
New-DirectoryIfNotExists "internal/shared/utils"
New-DirectoryIfNotExists "internal/shared/errors"

# Public packages
New-DirectoryIfNotExists "pkg/models"
New-DirectoryIfNotExists "pkg/client"
New-DirectoryIfNotExists "pkg/plugins"
New-DirectoryIfNotExists "pkg/sdk"

# Scripts
New-DirectoryIfNotExists "scripts/build"
New-DirectoryIfNotExists "scripts/test"
New-DirectoryIfNotExists "scripts/deploy"
New-DirectoryIfNotExists "scripts/tools"

# Tests
New-DirectoryIfNotExists "tests/unit"
New-DirectoryIfNotExists "tests/integration"
New-DirectoryIfNotExists "tests/e2e"
New-DirectoryIfNotExists "tests/benchmarks"
New-DirectoryIfNotExists "tests/fixtures"

# Tools
New-DirectoryIfNotExists "tools/codegen"
New-DirectoryIfNotExists "tools/lint"
New-DirectoryIfNotExists "tools/docs"

if (-not $DryRun) {
    Write-Status "Moving and reorganizing files..."

    # Move region files to assets
    Write-Status "Moving region files to assets/regions/"
    Move-FileWithBackup "aws_regions.json" "assets/regions/aws_regions.json"
    Move-FileWithBackup "azure_regions.json" "assets/regions/azure_regions.json"
    Move-FileWithBackup "gcp_regions.json" "assets/regions/gcp_regions.json"
    Move-FileWithBackup "digitalocean_regions.json" "assets/regions/digitalocean_regions.json"
    Move-FileWithBackup "all_regions.json" "assets/regions/all_regions.json"

    # Move configuration files
    Write-Status "Moving configuration files..."
    if (Test-Path "config/discovery-services.json") {
        Move-FileWithBackup "config/discovery-services.json" "assets/configs/discovery-services.json"
    }

    # Move documentation files
    Write-Status "Moving documentation files..."
    Move-FileWithBackup "COMPONENT_INTERACTIONS_ANALYSIS.md" "docs/architecture/component-interactions.md"
    Move-FileWithBackup "SECURITY_AND_FUNCTIONALITY_TEST_PLAN.md" "docs/development/security-test-plan.md"
    Move-FileWithBackup "TEST_EXECUTION_GUIDE.md" "docs/development/test-execution-guide.md"
    Move-FileWithBackup "HOW_TO_RUN.md" "docs/user-guide/how-to-run.md"
    Move-FileWithBackup "CLEANUP_SUMMARY.md" "docs/development/cleanup-summary.md"
    Move-FileWithBackup "RESTRUCTURE_COMPLETE.md" "docs/development/restructure-complete.md"

    # Move static files
    Write-Status "Moving static files..."
    if (Test-Path "static") {
        Copy-Item "static/*" "assets/static/" -Recurse -Force -ErrorAction SilentlyContinue
        Write-Status "Copied static files to assets/static/"
    }

    # Move web assets
    Write-Status "Moving web assets..."
    if (Test-Path "web/static") {
        Copy-Item "web/static/*" "assets/static/" -Recurse -Force -ErrorAction SilentlyContinue
        Write-Status "Copied web static files to assets/static/"
    }

    # Move deployment files
    Write-Status "Moving deployment files..."
    if (Test-Path "Dockerfile") {
        Move-FileWithBackup "Dockerfile" "deployments/docker/Dockerfile"
    }

    # Move scripts
    Write-Status "Moving scripts..."
    if (Test-Path "scripts") {
        Get-ChildItem "scripts/*.sh" -ErrorAction SilentlyContinue | ForEach-Object {
            $filename = $_.Name
            switch -Wildcard ($filename) {
                "*build*" { Move-FileWithBackup $_.FullName "scripts/build/$filename" }
                "*test*" { Move-FileWithBackup $_.FullName "scripts/test/$filename" }
                "*deploy*" { Move-FileWithBackup $_.FullName "scripts/deploy/$filename" }
                default { Move-FileWithBackup $_.FullName "scripts/tools/$filename" }
            }
        }
    }

    # Move PowerShell scripts
    Write-Status "Moving PowerShell scripts..."
    if (Test-Path "install.ps1") {
        Move-FileWithBackup "install.ps1" "scripts/build/install.ps1"
    }

    # Create configuration files
    Write-Status "Creating configuration files..."

    # Create main configuration file
    @"
# DriftMgr Main Configuration
version: "1.0"

# Server configuration
server:
  port: 8080
  host: "0.0.0.0"
  timeout: 30s

# Discovery configuration
discovery:
  cache_ttl: 300s
  cache_max_size: 1000
  parallel_workers: 10
  regions:
    - "us-east-1"
    - "us-west-2"
    - "eu-west-1"

# Analysis configuration
analysis:
  drift_threshold: 0.1
  severity_levels:
    - "critical"
    - "high"
    - "medium"
    - "low"

# Remediation configuration
remediation:
  auto_approve: false
  dry_run: true
  max_concurrent: 5

# Logging configuration
logging:
  level: "info"
  format: "json"
  output: "stdout"

# Security configuration
security:
  enable_auth: true
  jwt_secret: ""
  cors_origins: ["*"]
"@ | Out-File -FilePath "driftmgr.yaml" -Encoding UTF8

    # Create .golangci.yml
    @"
run:
  timeout: 5m
  modules-download-mode: readonly

linters:
  enable:
    - gofmt
    - golint
    - govet
    - errcheck
    - staticcheck
    - gosimple
    - ineffassign
    - unused
    - misspell
    - gosec

linters-settings:
  govet:
    check-shadowing: true
  gosec:
    excludes:
      - G404 # Use of weak random number generator

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - errcheck
        - gosec
"@ | Out-File -FilePath ".golangci.yml" -Encoding UTF8

    # Create CONTRIBUTING.md
    @"
# Contributing to DriftMgr

## Development Setup

1. Clone the repository
2. Install Go 1.21 or later
3. Run `make setup` to install dependencies
4. Run `make test` to verify everything works

## Code Style

- Follow Go formatting standards (`gofmt`)
- Use meaningful variable and function names
- Add comments for exported functions
- Keep functions small and focused

## Testing

- Write unit tests for new functionality
- Run `make test` before submitting PRs
- Ensure test coverage doesn't decrease

## Pull Request Process

1. Create a feature branch
2. Make your changes
3. Add tests
4. Update documentation
5. Submit a pull request

## Commit Messages

Use conventional commit format:
- `feat:` for new features
- `fix:` for bug fixes
- `docs:` for documentation
- `test:` for tests
- `refactor:` for refactoring
"@ | Out-File -FilePath "CONTRIBUTING.md" -Encoding UTF8

    # Create CHANGELOG.md
    @"
# Changelog

All notable changes to DriftMgr will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- New file structure for better organization
- Improved documentation
- Enhanced configuration management

### Changed
- Reorganized internal packages
- Updated import paths
- Improved build process

### Fixed
- Various minor issues

## [0.1.0] - 2024-01-01

### Added
- Initial release
- Basic drift detection
- Multi-cloud support
- Web dashboard
- CLI interface
"@ | Out-File -FilePath "CHANGELOG.md" -Encoding UTF8

    Write-Status "Updating .gitignore..."

    # Update .gitignore
    @"

# Build artifacts
bin/
dist/
build/

# IDE files
.vscode/
.idea/
*.swp
*.swo

# OS files
.DS_Store
Thumbs.db

# Logs
*.log
logs/

# Temporary files
*.tmp
*.temp

# Test coverage
coverage.out
coverage.html

# Environment files
.env
.env.local
.env.*.local

# Backup files
*.backup
*.bak

# Generated files
*.gen.go
"@ | Add-Content -Path ".gitignore"

    Write-Status "Creating build scripts..."

    # Create build script
    @"
#!/bin/bash

set -e

VERSION=`${VERSION:-$(git describe --tags --always --dirty)}`
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS="-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME"

echo "Building DriftMgr version $VERSION..."

# Build main CLI
go build -ldflags "$LDFLAGS" -o bin/driftmgr cmd/driftmgr/main.go

# Build server
go build -ldflags "$LDFLAGS" -o bin/driftmgr-server cmd/driftmgr-server/main.go

# Build agent
go build -ldflags "$LDFLAGS" -o bin/driftmgr-agent cmd/driftmgr-agent/main.go

echo "Build complete! Binaries are in the bin/ directory."
"@ | Out-File -FilePath "scripts/build/build.sh" -Encoding UTF8

    # Create test script
    @"
#!/bin/bash

set -e

echo "Running tests..."

# Run unit tests
go test ./internal/... -v

# Run integration tests
go test ./tests/integration/... -v

# Run benchmarks
go test ./tests/benchmarks/... -bench=.

echo "All tests passed!"
"@ | Out-File -FilePath "scripts/test/test.sh" -Encoding UTF8

    # Create deployment script
    @"
#!/bin/bash

set -e

ENVIRONMENT=`${1:-development}`

echo "Deploying DriftMgr to $ENVIRONMENT..."

# Build the application
./scripts/build/build.sh

# Deploy based on environment
case $ENVIRONMENT in
    development)
        echo "Starting development server..."
        ./bin/driftmgr-server
        ;;
    production)
        echo "Deploying to production..."
        docker-compose -f deployments/docker/docker-compose.prod.yml up -d
        ;;
    *)
        echo "Unknown environment: $ENVIRONMENT"
        exit 1
        ;;
esac

echo "Deployment complete!"
"@ | Out-File -FilePath "scripts/deploy/deploy.sh" -Encoding UTF8

    Write-Status "Creating Makefile..."

    # Create Makefile
    @"
.PHONY: help build test clean setup lint docker-build docker-run

# Default target
help:
	@echo "Available targets:"
	@echo "  build        - Build the application"
	@echo "  test         - Run tests"
	@echo "  clean        - Clean build artifacts"
	@echo "  setup        - Setup development environment"
	@echo "  lint         - Run linters"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"

# Build the application
build:
	@echo "Building DriftMgr..."
	@./scripts/build/build.sh

# Run tests
test:
	@echo "Running tests..."
	@./scripts/test/test.sh

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -rf dist/
	@go clean -cache

# Setup development environment
setup:
	@echo "Setting up development environment..."
	@go mod download
	@go mod tidy
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linters
lint:
	@echo "Running linters..."
	@golangci-lint run

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	@docker build -f deployments/docker/Dockerfile -t driftmgr:latest .

# Run Docker container
docker-run:
	@echo "Running Docker container..."
	@docker run -p 8080:8080 driftmgr:latest

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Generate documentation
docs:
	@echo "Generating documentation..."
	@mkdir -p docs/api
	@swag init -g cmd/driftmgr-server/main.go -o docs/api

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	@go test ./tests/benchmarks/... -bench=. -benchmem

# Format code
fmt:
	@echo "Formatting code..."
	@gofmt -s -w .
	@goimports -w .

# Vet code
vet:
	@echo "Vetting code..."
	@go vet ./...

# Check for security issues
security:
	@echo "Checking for security issues..."
	@gosec ./...
"@ | Out-File -FilePath "Makefile" -Encoding UTF8
}

Write-Success "File structure migration completed successfully!"

Write-Status "Next steps:"
Write-Host "1. Update import paths in Go files" -ForegroundColor $White
Write-Host "2. Run 'make setup' to install dependencies" -ForegroundColor $White
Write-Host "3. Run 'make test' to verify everything works" -ForegroundColor $White
Write-Host "4. Update documentation references" -ForegroundColor $White
Write-Host "5. Commit changes with appropriate message" -ForegroundColor $White

Write-Warning "Note: You may need to manually update some import paths and fix any build issues that arise."

Write-Status "Migration script completed!"
