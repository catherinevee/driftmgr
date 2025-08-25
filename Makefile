# DriftMgr Makefile
# Build automation for DriftMgr project

.PHONY: help build test test-all test-unit test-integration test-e2e test-benchmarks test-performance test-coverage clean setup lint docker-build docker-run

# Default target
help:
	@echo "Available targets:"
	@echo "  build            - Build the application"
	@echo "  test             - Run basic tests"
	@echo "  test-all         - Run comprehensive test suite"
	@echo "  test-unit        - Run unit tests"
	@echo "  test-integration - Run integration tests"
	@echo "  test-e2e         - Run end-to-end tests"
	@echo "  test-benchmarks  - Run benchmark tests"
	@echo "  test-performance - Run performance tests"
	@echo "  test-coverage    - Generate test coverage report"
	@echo "  clean            - Clean build artifacts"
	@echo "  setup            - Setup development environment"
	@echo "  lint             - Run linters"
	@echo "  docker-build     - Build Docker image"
	@echo "  docker-run       - Run Docker container"

# Build all binaries
build:
	@echo "Building DriftMgr..."
	go build -o build/driftmgr ./cmd/driftmgr
	go build -o build/driftmgr-server ./cmd/server
	go build -o build/validate-discovery ./cmd/validate

# Run basic tests
test:
	@echo "Running basic tests..."
	go test ./internal/...

# Run comprehensive test suite
test-all:
	@echo "Running comprehensive test suite..."
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File scripts/run-tests.ps1 -TestType all
else
	@bash scripts/run-tests.sh --type all
endif

# Run unit tests
test-unit:
	@echo "Running unit tests..."
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File scripts/run-tests.ps1 -TestType unit
else
	@bash scripts/run-tests.sh --type unit
endif

# Run integration tests
test-integration:
	@echo "Running integration tests..."
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File scripts/run-tests.ps1 -TestType integration
else
	@bash scripts/run-tests.sh --type integration
endif

# Run end-to-end tests
test-e2e:
	@echo "Running end-to-end tests..."
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File scripts/run-tests.ps1 -TestType e2e
else
	@bash scripts/run-tests.sh --type e2e
endif

# Run benchmark tests
test-benchmarks:
	@echo "Running benchmark tests..."
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File scripts/run-tests.ps1 -TestType benchmarks
else
	@bash scripts/run-tests.sh --type benchmarks
endif

# Run performance tests
test-performance:
	@echo "Running performance tests..."
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File scripts/run-tests.ps1 -TestType performance
else
	@bash scripts/run-tests.sh --type performance
endif

# Generate test coverage report
test-coverage:
	@echo "Generating test coverage report..."
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File scripts/run-tests.ps1 -TestType coverage
else
	@bash scripts/run-tests.sh --type coverage
endif

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@if exist build rmdir /s /q build
	@if exist bin rmdir /s /q bin
	@if exist dist rmdir /s /q dist
	@if exist temp rmdir /s /q temp
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

# Alias for benchmark tests
bench: test-benchmarks

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
