#!/bin/bash

# DriftMgr File Structure Migration Script
# This script helps reorganize the project structure according to Go best practices

set -e

echo "🚀 Starting DriftMgr file structure migration..."

# Create new directory structure
echo "📁 Creating new directory structure..."

# Core Go directories
mkdir -p cmd/driftmgr-server
mkdir -p cmd/driftmgr-web
mkdir -p internal/api
mkdir -p internal/drift
mkdir -p internal/state
mkdir -p internal/analysis
mkdir -p internal/discovery
mkdir -p internal/notification
mkdir -p pkg/models
mkdir -p pkg/config
mkdir -p pkg/utils
mkdir -p pkg/providers

# Web assets
mkdir -p web/static
mkdir -p web/templates
mkdir -p web/assets

# Scripts organization
mkdir -p scripts/build
mkdir -p scripts/deploy
mkdir -p scripts/test
mkdir -p scripts/tools

# Documentation organization
mkdir -p docs/api
mkdir -p docs/user-guide
mkdir -p docs/deployment
mkdir -p docs/development
mkdir -p docs/examples

# Examples organization
mkdir -p examples/terraform
mkdir -p examples/workflows
mkdir -p examples/configurations

# Configuration organization
mkdir -p configs/default
mkdir -p configs/development
mkdir -p configs/production

# Test organization
mkdir -p tests/unit
mkdir -p tests/integration
mkdir -p tests/e2e

# Deployment organization
mkdir -p deployments/docker
mkdir -p deployments/kubernetes
mkdir -p deployments/terraform

# Tools organization
mkdir -p tools/blast-radius
mkdir -p tools/state-analyzer
mkdir -p tools/drift-visualizer

# Binary and distribution directories
mkdir -p bin
mkdir -p dist

echo "✅ Directory structure created"

# Move executables
echo "📦 Moving executables..."
if [ -f "driftmgr.exe" ]; then
    mv driftmgr.exe bin/
fi
if [ -f "cmd/driftmgr.exe" ]; then
    mv cmd/driftmgr.exe bin/
fi

# Move scripts from root to scripts/
echo "📜 Moving scripts..."
for script in *.ps1 *.sh; do
    if [ -f "$script" ]; then
        case "$script" in
            *demo*|*test*)
                mv "$script" scripts/test/
                ;;
            *build*|*compile*)
                mv "$script" scripts/build/
                ;;
            *deploy*|*docker*|*kubernetes*)
                mv "$script" scripts/deploy/
                ;;
            *)
                mv "$script" scripts/tools/
                ;;
        esac
    fi
done

# Move documentation from root to docs/
echo "📚 Moving documentation..."
for doc in *.md; do
    if [ -f "$doc" ] && [ "$doc" != "README.md" ]; then
        case "$doc" in
            *API*|*api*)
                mv "$doc" docs/api/
                ;;
            *SECURITY*|*security*)
                mv "$doc" docs/development/
                ;;
            *CISCO*|*CONTEXT*)
                mv "$doc" docs/user-guide/
                ;;
            *)
                mv "$doc" docs/
                ;;
        esac
    fi
done

# Move web files
echo "🌐 Moving web files..."
if [ -f "web/main.go" ]; then
    mv web/main.go cmd/driftmgr-web/
fi
if [ -f "web/server.go" ]; then
    mv web/server.go cmd/driftmgr-web/
fi

# Move shared models
echo "📋 Moving shared models..."
if [ -d "shared/models" ]; then
    mv shared/models/* pkg/models/
    rmdir shared/models
fi
if [ -d "shared/config" ]; then
    mv shared/config/* pkg/config/
    rmdir shared/config
fi

# Move tools
echo "🔧 Moving tools..."
if [ -d "tools" ]; then
    for tool in tools/*.py; do
        if [ -f "$tool" ]; then
            case "$tool" in
                *blast*)
                    mv "$tool" tools/blast-radius/
                    ;;
                *tfstate*)
                    mv "$tool" tools/state-analyzer/
                    ;;
                *)
                    mv "$tool" tools/drift-visualizer/
                    ;;
            esac
        fi
    done
fi

# Move services to internal
echo "⚙️ Moving services..."
if [ -d "services" ]; then
    for service in services/*; do
        if [ -d "$service" ]; then
            service_name=$(basename "$service")
            case "$service_name" in
                *api*|*gateway*)
                    mv "$service"/* internal/api/
                    ;;
                *analysis*)
                    mv "$service"/* internal/analysis/
                    ;;
                *discovery*)
                    mv "$service"/* internal/discovery/
                    ;;
                *notification*)
                    mv "$service"/* internal/notification/
                    ;;
                *state*)
                    mv "$service"/* internal/state/
                    ;;
                *visualization*)
                    mv "$service"/* internal/drift/
                    ;;
            esac
            rmdir "$service"
        fi
    done
    rmdir services
fi

# Create .gitignore updates
echo "🚫 Updating .gitignore..."
cat >> .gitignore << EOF

# Binaries and distributions
bin/
dist/

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

# Environment files
.env
.env.local

# Temporary files
tmp/
temp/
EOF

# Create Makefile if it doesn't exist
if [ ! -f "Makefile" ]; then
    echo "🔨 Creating Makefile..."
    cat > Makefile << 'EOF'
.PHONY: build clean test lint format help

# Build targets
build:
	go build -o bin/driftmgr-client ./cmd/driftmgr-client
	go build -o bin/driftmgr-server ./cmd/driftmgr-server
	go build -o bin/driftmgr-web ./cmd/driftmgr-web

build-client:
	go build -o bin/driftmgr-client ./cmd/driftmgr-client

build-server:
	go build -o bin/driftmgr-server ./cmd/driftmgr-server

build-web:
	go build -o bin/driftmgr-web ./cmd/driftmgr-web

# Clean
clean:
	rm -rf bin/ dist/

# Test
test:
	go test ./...

test-unit:
	go test ./tests/unit/...

test-integration:
	go test ./tests/integration/...

test-e2e:
	go test ./tests/e2e/...

# Lint and format
lint:
	golangci-lint run

format:
	go fmt ./...
	gofmt -s -w .

# Development
dev:
	go run ./cmd/driftmgr-server

dev-client:
	go run ./cmd/driftmgr-client

dev-web:
	go run ./cmd/driftmgr-web

# Docker
docker-build:
	docker build -t driftmgr .

docker-run:
	docker run -p 8080:8080 driftmgr

# Help
help:
	@echo "Available targets:"
	@echo "  build        - Build all binaries"
	@echo "  build-client - Build client only"
	@echo "  build-server - Build server only"
	@echo "  build-web    - Build web interface only"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run all tests"
	@echo "  test-unit    - Run unit tests"
	@echo "  test-integration - Run integration tests"
	@echo "  test-e2e     - Run end-to-end tests"
	@echo "  lint         - Run linter"
	@echo "  format       - Format code"
	@echo "  dev          - Run server in development mode"
	@echo "  dev-client   - Run client in development mode"
	@echo "  dev-web      - Run web interface in development mode"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"
EOF
fi

echo "✅ Migration completed successfully!"
echo ""
echo "📋 Next steps:"
echo "1. Review the new structure"
echo "2. Update import paths in Go files"
echo "3. Update documentation references"
echo "4. Test all functionality"
echo "5. Update CI/CD pipelines if needed"
echo ""
echo "📖 See STRUCTURE_IMPROVEMENTS.md for detailed information"
