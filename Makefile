# DriftMgr Makefile
# Build automation for DriftMgr project

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
	@powershell -ExecutionPolicy Bypass -File scripts/build/build.ps1
	@echo "Verifying build..."
	@powershell -ExecutionPolicy Bypass -File scripts/build/verify-build.ps1

# Run tests
test:
	@echo "Running comprehensive tests..."
	@powershell -ExecutionPolicy Bypass -File scripts/test/run_comprehensive_tests.ps1

# Run specific test types
test-unit:
	@echo "Running unit tests..."
	@powershell -ExecutionPolicy Bypass -File scripts/test/run_comprehensive_tests.ps1 unit

test-integration:
	@echo "Running integration tests..."
	@powershell -ExecutionPolicy Bypass -File scripts/test/run_comprehensive_tests.ps1 integration

test-e2e:
	@echo "Running end-to-end tests..."
	@powershell -ExecutionPolicy Bypass -File scripts/test/run_comprehensive_tests.ps1 e2e

test-benchmark:
	@echo "Running benchmarks..."
	@powershell -ExecutionPolicy Bypass -File scripts/test/run_comprehensive_tests.ps1 benchmarks

test-security:
	@echo "Running security tests..."
	@powershell -ExecutionPolicy Bypass -File scripts/test/run_comprehensive_tests.ps1 security

test-coverage:
	@echo "Generating coverage report..."
	@powershell -ExecutionPolicy Bypass -File scripts/test/run_comprehensive_tests.ps1 coverage

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@if exist bin rmdir /s /q bin
	@if exist dist rmdir /s /q dist
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
	@if not exist docs\api mkdir docs\api
	@swag init -g cmd/driftmgr-server/main.go -o docs/api

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	@powershell -ExecutionPolicy Bypass -File scripts/test/run_comprehensive_tests.ps1 benchmarks

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
