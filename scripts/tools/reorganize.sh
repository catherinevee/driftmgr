#!/bin/bash

# DriftMgr Directory Reorganization Script
# This script helps reorganize the project structure for better maintainability

set -e

echo "=========================================="
echo "DriftMgr Directory Reorganization"
echo "=========================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to create directory if it doesn't exist
create_dir() {
    if [ ! -d "$1" ]; then
        echo -e "${BLUE}Creating directory:${NC} $1"
        mkdir -p "$1"
    fi
}

# Function to move file with backup
move_file() {
    local src="$1"
    local dest="$2"
    
    if [ -f "$src" ]; then
        echo -e "${BLUE}Moving:${NC} $src -> $dest"
        # Create backup
        cp "$src" "${src}.backup"
        # Move file
        mv "$src" "$dest"
    else
        echo -e "${YELLOW}Warning:${NC} Source file not found: $src"
    fi
}

# Function to move directory with backup
move_dir() {
    local src="$1"
    local dest="$2"
    
    if [ -d "$src" ]; then
        echo -e "${BLUE}Moving directory:${NC} $src -> $dest"
        # Create backup
        cp -r "$src" "${src}.backup"
        # Move directory
        mv "$src" "$dest"
    else
        echo -e "${YELLOW}Warning:${NC} Source directory not found: $src"
    fi
}

echo -e "${GREEN}Phase 1: Creating new directory structure${NC}"

# Create main directories
create_dir "web/static/css"
create_dir "web/static/js"
create_dir "web/static/images"
create_dir "web/templates"

create_dir "configs/environments"

create_dir "scripts/build"
create_dir "scripts/install"
create_dir "scripts/deploy"
create_dir "scripts/tools"

create_dir "docs/api"
create_dir "docs/cli"
create_dir "docs/web"
create_dir "docs/deployment"
create_dir "docs/development"

create_dir "examples/basic"
create_dir "examples/advanced"
create_dir "examples/demos"

create_dir "tests/unit"
create_dir "tests/integration"
create_dir "tests/e2e"
create_dir "tests/fixtures/state-files"
create_dir "tests/fixtures/configs"

create_dir "deployments/docker"
create_dir "deployments/kubernetes"
create_dir "deployments/terraform"

create_dir "tools/codegen"
create_dir "tools/migrations"
create_dir "tools/benchmarks"

create_dir "assets/images"
create_dir "assets/logos"
create_dir "assets/samples/state-files"
create_dir "assets/samples/configs"

echo -e "${GREEN}Phase 2: Moving and organizing files${NC}"

# Move documentation
echo -e "${BLUE}Organizing documentation...${NC}"
if [ -f "docs/cli/enhanced-features-guide.md" ]; then
    move_file "docs/cli/enhanced-features-guide.md" "docs/cli/features.md"
fi

if [ -f "docs/cli/README.md" ]; then
    move_file "docs/cli/README.md" "docs/cli/README.md"
fi

# Move scripts
echo -e "${BLUE}Organizing scripts...${NC}"
if [ -f "scripts/build-enhanced-cli.sh" ]; then
    move_file "scripts/build-enhanced-cli.sh" "scripts/build/build-client.sh"
fi

if [ -f "scripts/build-enhanced-cli.bat" ]; then
    move_file "scripts/build-enhanced-cli.bat" "scripts/build/build-client.bat"
fi

if [ -f "scripts/demo-enhanced-cli.sh" ]; then
    move_file "scripts/demo-enhanced-cli.sh" "scripts/tools/demo-cli.sh"
fi

if [ -f "scripts/README.md" ]; then
    move_file "scripts/README.md" "scripts/README.md"
fi

# Move installation scripts
if [ -f "install.sh" ]; then
    move_file "install.sh" "scripts/install/install.sh"
fi

if [ -f "install.ps1" ]; then
    move_file "install.ps1" "scripts/install/install.ps1"
fi

# Move examples
echo -e "${BLUE}Organizing examples...${NC}"
if [ -d "examples" ]; then
    # Move existing examples to appropriate subdirectories
    for example_dir in examples/*/; do
        if [ -d "$example_dir" ]; then
            dir_name=$(basename "$example_dir")
            if [[ "$dir_name" == *"demo"* ]]; then
                move_dir "$example_dir" "examples/demos/$dir_name"
            elif [[ "$dir_name" == *"basic"* || "$dir_name" == *"simple"* ]]; then
                move_dir "$example_dir" "examples/basic/$dir_name"
            else
                move_dir "$example_dir" "examples/advanced/$dir_name"
            fi
        fi
    done
fi

# Move test files
echo -e "${BLUE}Organizing test files...${NC}"
if [ -d "tests" ]; then
    # Move existing test files to appropriate subdirectories
    for test_file in tests/*_test.go; do
        if [ -f "$test_file" ]; then
            move_file "$test_file" "tests/unit/$(basename "$test_file")"
        fi
    done
fi

# Move configuration files
echo -e "${BLUE}Organizing configuration files...${NC}"
if [ -f "driftmgr.yaml" ]; then
    move_file "driftmgr.yaml" "configs/driftmgr.yaml"
fi

if [ -f "driftmgr.yaml.example" ]; then
    move_file "driftmgr.yaml.example" "configs/driftmgr.yaml.example"
fi

# Move deployment files
echo -e "${BLUE}Organizing deployment files...${NC}"
if [ -f "Dockerfile" ]; then
    move_file "Dockerfile" "deployments/docker/Dockerfile"
fi

if [ -f "docker-compose.yml" ]; then
    move_file "docker-compose.yml" "deployments/docker/docker-compose.yml"
fi

if [ -f ".dockerignore" ]; then
    move_file ".dockerignore" "deployments/docker/.dockerignore"
fi

# Move assets
echo -e "${BLUE}Organizing assets...${NC}"
if [ -f "test_state_file.json" ]; then
    move_file "test_state_file.json" "assets/samples/state-files/test_state_file.json"
fi

# Move implementation summary
if [ -f "ENHANCED_CLI_IMPLEMENTATION_SUMMARY.md" ]; then
    move_file "ENHANCED_CLI_IMPLEMENTATION_SUMMARY.md" "docs/development/cli-implementation-summary.md"
fi

echo -e "${GREEN}Phase 3: Creating new documentation structure${NC}"

# Create main documentation index
cat > docs/README.md << 'EOF'
# DriftMgr Documentation

This directory contains comprehensive documentation for the DriftMgr project.

## Documentation Index

### User Documentation
- **[CLI Documentation](cli/)** - Command-line interface guide
- **[Web Interface](web/)** - Web dashboard documentation
- **[API Reference](api/)** - REST API documentation

### Developer Documentation
- **[Development Setup](development/)** - Development environment setup
- **[Architecture](development/architecture.md)** - System architecture overview
- **[Contributing](development/contributing.md)** - Contribution guidelines

### Deployment Documentation
- **[Docker Deployment](deployment/docker.md)** - Docker-based deployment
- **[Kubernetes Deployment](deployment/kubernetes.md)** - Kubernetes deployment
- **[Cloud Deployment](deployment/cloud.md)** - Cloud platform deployment

## Quick Start

1. **Installation**: See [CLI Documentation](cli/) for installation instructions
2. **Configuration**: See [Development Setup](development/) for configuration
3. **Usage**: See [CLI Documentation](cli/) for usage examples

## Support

For issues and questions:
1. Check the relevant documentation section
2. Review troubleshooting guides
3. Check the main project README
EOF

# Create CLI documentation index
cat > docs/cli/README.md << 'EOF'
# DriftMgr CLI Documentation

This directory contains documentation for the DriftMgr command-line interface.

## Documentation Index

### User Guides
- **[Commands Reference](commands.md)** - Complete command reference
- **[Features Guide](features.md)** - Advanced CLI features
- **[Troubleshooting](troubleshooting.md)** - Common issues and solutions

### Quick Reference
- **[Main README](../../README.md)** - Project overview
- **[Installation Guide](../../README.md#installation)** - Installation instructions

## CLI Features

### Core Features
- **Interactive Shell** - User-friendly command-line interface
- **Context-Sensitive Help** - Type `?` or `command ?` for help
- **Command History** - Up to 100 commands with navigation
- **Security Hardened** - Input validation and injection prevention

### Advanced Features
- **Tab Completion** - Auto-complete commands and arguments
- **Auto-Suggestions** - Smart suggestions based on history and context
- **Fuzzy Search** - Find commands with partial input
- **Arrow Key Navigation** - Navigate history and move cursor
- **Context-Aware Completion** - Dynamic completion based on discovered resources

## Getting Started

### Installation
```bash
# Build the CLI
go build -o driftmgr-client.exe cmd/driftmgr-client/*.go

# Or use the build script
./scripts/build/build-client.sh
```

### Basic Usage
```bash
# Start interactive shell
./driftmgr-client.exe

# Run commands directly
./driftmgr-client.exe discover aws us-east-1
./driftmgr-client.exe analyze terraform
./driftmgr-client.exe help
```
EOF

echo -e "${GREEN}Phase 4: Creating build scripts${NC}"

# Create main build script
cat > scripts/build/build-all.sh << 'EOF'
#!/bin/bash

# DriftMgr Build All Script
# Builds both server and client applications

set -e

echo "=========================================="
echo "Building DriftMgr Applications"
echo "=========================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed or not in PATH${NC}"
    exit 1
fi

echo -e "${BLUE}Go version:${NC}"
go version
echo

# Build server
echo -e "${BLUE}Building server...${NC}"
go build -o driftmgr-server.exe cmd/driftmgr-server/*.go
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Server built successfully!${NC}"
else
    echo -e "${RED}✗ Server build failed!${NC}"
    exit 1
fi

# Build client
echo -e "${BLUE}Building client...${NC}"
go build -o driftmgr-client.exe cmd/driftmgr-client/*.go
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Client built successfully!${NC}"
else
    echo -e "${RED}✗ Client build failed!${NC}"
    exit 1
fi

echo
echo "=========================================="
echo -e "${GREEN}Build Complete!${NC}"
echo "=========================================="
echo
echo "Built applications:"
echo "  - driftmgr-server.exe (Server application)"
echo "  - driftmgr-client.exe (CLI application)"
echo
echo "To run the server:"
echo "  ./driftmgr-server.exe"
echo
echo "To run the client:"
echo "  ./driftmgr-client.exe"
EOF

chmod +x scripts/build/build-all.sh

# Create Windows build script
cat > scripts/build/build-all.bat << 'EOF'
@echo off
REM DriftMgr Build All Script for Windows
REM Builds both server and client applications

echo ==========================================
echo Building DriftMgr Applications
echo ==========================================

REM Check if Go is installed
go version >nul 2>&1
if errorlevel 1 (
    echo Error: Go is not installed or not in PATH
    exit /b 1
)

echo Go version:
go version
echo.

REM Build server
echo Building server...
go build -o driftmgr-server.exe cmd\driftmgr-server\*.go
if errorlevel 1 (
    echo ✗ Server build failed!
    exit /b 1
)
echo ✓ Server built successfully!

REM Build client
echo Building client...
go build -o driftmgr-client.exe cmd\driftmgr-client\*.go
if errorlevel 1 (
    echo ✗ Client build failed!
    exit /b 1
)
echo ✓ Client built successfully!

echo.
echo ==========================================
echo Build Complete!
echo ==========================================
echo.
echo Built applications:
echo   - driftmgr-server.exe (Server application)
echo   - driftmgr-client.exe (CLI application)
echo.
echo To run the server:
echo   driftmgr-server.exe
echo.
echo To run the client:
echo   driftmgr-client.exe
EOF

echo -e "${GREEN}Phase 5: Creating Makefile${NC}"

# Create Makefile
cat > Makefile << 'EOF'
# DriftMgr Makefile
# Build automation for DriftMgr project

.PHONY: help build build-server build-client clean test install

# Default target
help:
	@echo "DriftMgr Build Targets:"
	@echo "  build        - Build both server and client"
	@echo "  build-server - Build server application only"
	@echo "  build-client - Build client application only"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run tests"
	@echo "  install      - Install dependencies"
	@echo "  help         - Show this help message"

# Build both applications
build: build-server build-client

# Build server application
build-server:
	@echo "Building server application..."
	go build -o driftmgr-server.exe cmd/driftmgr-server/*.go
	@echo "✓ Server built successfully!"

# Build client application
build-client:
	@echo "Building client application..."
	go build -o driftmgr-client.exe cmd/driftmgr-client/*.go
	@echo "✓ Client built successfully!"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f driftmgr-server.exe driftmgr-client.exe
	rm -f *.backup
	@echo "✓ Clean complete!"

# Run tests
test:
	@echo "Running tests..."
	go test ./...
	@echo "✓ Tests complete!"

# Install dependencies
install:
	@echo "Installing dependencies..."
	go mod download
	@echo "✓ Dependencies installed!"

# Development targets
dev-server:
	@echo "Starting development server..."
	go run cmd/driftmgr-server/*.go

dev-client:
	@echo "Starting development client..."
	go run cmd/driftmgr-client/*.go

# Documentation targets
docs-serve:
	@echo "Serving documentation..."
	cd docs && python3 -m http.server 8000

# Docker targets
docker-build:
	@echo "Building Docker image..."
	docker build -t driftmgr:latest deployments/docker/

docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 driftmgr:latest
EOF

echo -e "${GREEN}Phase 6: Updating .gitignore${NC}"

# Update .gitignore
cat >> .gitignore << 'EOF'

# Build artifacts
*.exe
*.dll
*.so
*.dylib

# Backup files
*.backup

# IDE files
.vscode/
.idea/
*.swp
*.swo

# OS files
.DS_Store
Thumbs.db

# Log files
*.log

# Environment files
.env
.env.local

# Test coverage
coverage.out
coverage.html

# Temporary files
tmp/
temp/
EOF

echo
echo "=========================================="
echo -e "${GREEN}Reorganization Complete!${NC}"
echo "=========================================="
echo
echo -e "${YELLOW}Next Steps:${NC}"
echo "1. Review the new directory structure"
echo "2. Update import statements in Go files"
echo "3. Update documentation links"
echo "4. Test all functionality"
echo "5. Update CI/CD configurations"
echo
echo -e "${BLUE}Backup files have been created with .backup extension${NC}"
echo -e "${BLUE}You can safely delete them after verifying everything works${NC}"
echo
echo -e "${GREEN}New structure created:${NC}"
echo "  ✓ Organized documentation in docs/"
echo "  ✓ Organized scripts in scripts/"
echo "  ✓ Organized examples in examples/"
echo "  ✓ Organized tests in tests/"
echo "  ✓ Organized configs in configs/"
echo "  ✓ Organized deployments in deployments/"
echo "  ✓ Created build automation with Makefile"
echo "  ✓ Updated .gitignore for better artifact management"
