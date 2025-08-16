#!/bin/bash

# DriftMgr File Structure Migration Script
# This script helps migrate from the current structure to the improved structure

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to create directory if it doesn't exist
create_dir() {
    if [ ! -d "$1" ]; then
        mkdir -p "$1"
        print_status "Created directory: $1"
    fi
}

# Function to move file with backup
move_file() {
    local src="$1"
    local dst="$2"
    
    if [ -f "$src" ]; then
        if [ -f "$dst" ]; then
            print_warning "Destination file exists, creating backup: $dst.backup"
            mv "$dst" "$dst.backup"
        fi
        mv "$src" "$dst"
        print_status "Moved: $src -> $dst"
    else
        print_warning "Source file not found: $src"
    fi
}

# Function to copy file if it doesn't exist
copy_file() {
    local src="$1"
    local dst="$2"
    
    if [ -f "$src" ] && [ ! -f "$dst" ]; then
        cp "$src" "$dst"
        print_status "Copied: $src -> $dst"
    elif [ ! -f "$src" ]; then
        print_warning "Source file not found: $src"
    fi
}

print_status "Starting DriftMgr file structure migration..."

# Create new directory structure
print_status "Creating new directory structure..."

# API layer
create_dir "api/v1"
create_dir "api/docs"

# Assets
create_dir "assets/regions"
create_dir "assets/configs"
create_dir "assets/templates/terraform"
create_dir "assets/templates/reports"
create_dir "assets/static/css"
create_dir "assets/static/js"
create_dir "assets/static/images"

# Commands
create_dir "cmd/driftmgr/commands"
create_dir "cmd/driftmgr-server/server"
create_dir "cmd/driftmgr-agent/agent"

# Configurations
create_dir "configs"

# Deployments
create_dir "deployments/docker"
create_dir "deployments/kubernetes"
create_dir "deployments/terraform"

# Documentation
create_dir "docs/api"
create_dir "docs/deployment"
create_dir "docs/development"
create_dir "docs/user-guide"
create_dir "docs/architecture"

# Examples
create_dir "examples/basic"
create_dir "examples/advanced"
create_dir "examples/multi-cloud"
create_dir "examples/custom-plugins"

# Internal structure
create_dir "internal/core/discovery/providers"
create_dir "internal/core/analysis/detectors"
create_dir "internal/core/remediation/strategies"
create_dir "internal/core/workflow"

create_dir "internal/platform/api/handlers"
create_dir "internal/platform/api/middleware"
create_dir "internal/platform/web/templates"
create_dir "internal/platform/web/static"
create_dir "internal/platform/cli"
create_dir "internal/platform/storage"

create_dir "internal/shared/config"
create_dir "internal/shared/logging"
create_dir "internal/shared/metrics"
create_dir "internal/shared/security"
create_dir "internal/shared/utils"
create_dir "internal/shared/errors"

# Public packages
create_dir "pkg/models"
create_dir "pkg/client"
create_dir "pkg/plugins"
create_dir "pkg/sdk"

# Scripts
create_dir "scripts/build"
create_dir "scripts/test"
create_dir "scripts/deploy"
create_dir "scripts/tools"

# Tests
create_dir "tests/unit"
create_dir "tests/integration"
create_dir "tests/e2e"
create_dir "tests/benchmarks"
create_dir "tests/fixtures"

# Tools
create_dir "tools/codegen"
create_dir "tools/lint"
create_dir "tools/docs"

print_status "Moving and reorganizing files..."

# Move region files to assets
print_status "Moving region files to assets/regions/"
move_file "aws_regions.json" "assets/regions/aws_regions.json"
move_file "azure_regions.json" "assets/regions/azure_regions.json"
move_file "gcp_regions.json" "assets/regions/gcp_regions.json"
move_file "digitalocean_regions.json" "assets/regions/digitalocean_regions.json"
move_file "all_regions.json" "assets/regions/all_regions.json"

# Move configuration files
print_status "Moving configuration files..."
if [ -f "config/discovery-services.json" ]; then
    move_file "config/discovery-services.json" "assets/configs/discovery-services.json"
fi

# Move documentation files
print_status "Moving documentation files..."
move_file "COMPONENT_INTERACTIONS_ANALYSIS.md" "docs/architecture/component-interactions.md"
move_file "SECURITY_AND_FUNCTIONALITY_TEST_PLAN.md" "docs/development/security-test-plan.md"
move_file "TEST_EXECUTION_GUIDE.md" "docs/development/test-execution-guide.md"
move_file "HOW_TO_RUN.md" "docs/user-guide/how-to-run.md"
move_file "CLEANUP_SUMMARY.md" "docs/development/cleanup-summary.md"
move_file "RESTRUCTURE_COMPLETE.md" "docs/development/restructure-complete.md"

# Move static files
print_status "Moving static files..."
if [ -d "static" ]; then
    cp -r static/* assets/static/ 2>/dev/null || true
    print_status "Copied static files to assets/static/"
fi

# Move web assets
print_status "Moving web assets..."
if [ -d "web/static" ]; then
    cp -r web/static/* assets/static/ 2>/dev/null || true
    print_status "Copied web static files to assets/static/"
fi

# Move deployment files
print_status "Moving deployment files..."
if [ -f "Dockerfile" ]; then
    move_file "Dockerfile" "deployments/docker/Dockerfile"
fi

# Move scripts
print_status "Moving scripts..."
if [ -d "scripts" ]; then
    # Move existing scripts to appropriate locations
    for script in scripts/*.sh; do
        if [ -f "$script" ]; then
            filename=$(basename "$script")
            case "$filename" in
                *build*) move_file "$script" "scripts/build/$filename" ;;
                *test*) move_file "$script" "scripts/test/$filename" ;;
                *deploy*) move_file "$script" "scripts/deploy/$filename" ;;
                *) move_file "$script" "scripts/tools/$filename" ;;
            esac
        fi
    done
fi

# Move PowerShell scripts
print_status "Moving PowerShell scripts..."
if [ -f "install.ps1" ]; then
    move_file "install.ps1" "scripts/build/install.ps1"
fi

# Create configuration files
print_status "Creating configuration files..."

# Create main configuration file
cat > driftmgr.yaml << 'EOF'
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
EOF

# Create .golangci.yml
cat > .golangci.yml << 'EOF'
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
EOF

# Create CONTRIBUTING.md
cat > CONTRIBUTING.md << 'EOF'
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
EOF

# Create CHANGELOG.md
cat > CHANGELOG.md << 'EOF'
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
EOF

print_status "Updating .gitignore..."

# Update .gitignore
cat >> .gitignore << 'EOF'

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
EOF

print_status "Creating build scripts..."

# Create build script
cat > scripts/build/build.sh << 'EOF'
#!/bin/bash

set -e

VERSION=${VERSION:-$(git describe --tags --always --dirty)}
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
EOF

chmod +x scripts/build/build.sh

# Create test script
cat > scripts/test/test.sh << 'EOF'
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
EOF

chmod +x scripts/test/test.sh

# Create deployment script
cat > scripts/deploy/deploy.sh << 'EOF'
#!/bin/bash

set -e

ENVIRONMENT=${1:-development}

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
EOF

chmod +x scripts/deploy/deploy.sh

print_status "Creating Makefile..."

# Create Makefile
cat > Makefile << 'EOF'
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
EOF

print_success "File structure migration completed successfully!"

print_status "Next steps:"
echo "1. Update import paths in Go files"
echo "2. Run 'make setup' to install dependencies"
echo "3. Run 'make test' to verify everything works"
echo "4. Update documentation references"
echo "5. Commit changes with appropriate message"

print_warning "Note: You may need to manually update some import paths and fix any build issues that arise."

print_status "Migration script completed!"
