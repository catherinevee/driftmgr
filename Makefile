# DriftMgr Makefile
# Build automation for DriftMgr project

# Variables
BINARY_NAME := driftmgr
DOCKER_IMAGE := catherinevee/driftmgr
VERSION := $(shell git describe --tags --always --dirty 2>nul || echo "dev")
BUILD_TIME := $(shell date /t)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>nul || echo "unknown")

# Go build flags
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -s -w"

# Colors for output (Windows compatible)
RED := [31m
GREEN := [32m
YELLOW := [33m
BLUE := [34m
NC := [0m

.PHONY: help build build-all test test-all test-unit test-integration test-e2e test-benchmarks test-performance test-coverage clean setup lint docker-build docker-run install dev

# Default target
help:
	@echo "$(BLUE)DriftMgr Development Makefile$(NC)"
	@echo "$(YELLOW)Usage:$(NC) make [target]"
	@echo ""
	@echo "$(GREEN)Build targets:$(NC)"
	@echo "  build            - Build the application for current OS"
	@echo "  build-all        - Build for all platforms"
	@echo "  install          - Install binary to GOPATH/bin"
	@echo ""
	@echo "$(GREEN)Test targets:$(NC)"
	@echo "  test             - Run unit tests"
	@echo "  test-all         - Run all test suites"
	@echo "  test-integration - Run integration tests"
	@echo "  test-e2e         - Run end-to-end tests"
	@echo "  test-coverage    - Generate coverage report"
	@echo "  benchmark        - Run performance benchmarks"
	@echo ""
	@echo "$(GREEN)Development:$(NC)"
	@echo "  dev              - Start development environment"
	@echo "  serve            - Start web server"
	@echo "  watch            - Watch and rebuild on changes"
	@echo "  fmt              - Format code"
	@echo "  vet              - Run go vet"
	@echo "  lint             - Run linters"
	@echo "  security         - Run security checks"
	@echo ""
	@echo "$(GREEN)Docker:$(NC)"
	@echo "  docker-build     - Build Docker image"
	@echo "  docker-run       - Run Docker container"
	@echo "  docker-push      - Push to Docker Hub"
	@echo ""
	@echo "$(GREEN)Utilities:$(NC)"
	@echo "  clean            - Clean build artifacts"
	@echo "  setup            - Setup development environment"
	@echo "  tools            - Install development tools"
	@echo "  mod              - Tidy and verify modules"
	@echo "  info             - Display build information"

# Build all binaries
build:
	@echo "$(BLUE)Building DriftMgr...$(NC)"
	@if not exist build mkdir build
	go build $(LDFLAGS) -o build/$(BINARY_NAME).exe ./cmd/driftmgr
	go build $(LDFLAGS) -o build/driftmgr-server.exe ./cmd/driftmgr-server
	@echo "$(GREEN)Build complete!$(NC)"

# Build for all platforms
build-all: build-windows build-linux build-darwin

build-windows:
	@echo "$(BLUE)Building for Windows...$(NC)"
	@if not exist dist mkdir dist
	@set GOOS=windows&& set GOARCH=amd64&& go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe ./cmd/driftmgr

build-linux:
	@echo "$(BLUE)Building for Linux...$(NC)"
	@if not exist dist mkdir dist
	@set GOOS=linux&& set GOARCH=amd64&& go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 ./cmd/driftmgr
	@set GOOS=linux&& set GOARCH=arm64&& go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 ./cmd/driftmgr

build-darwin:
	@echo "$(BLUE)Building for macOS...$(NC)"
	@if not exist dist mkdir dist
	@set GOOS=darwin&& set GOARCH=amd64&& go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 ./cmd/driftmgr
	@set GOOS=darwin&& set GOARCH=arm64&& go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 ./cmd/driftmgr

# Install binary
install: build
	@echo "$(BLUE)Installing $(BINARY_NAME)...$(NC)"
	go install $(LDFLAGS) ./cmd/driftmgr
	@echo "$(GREEN)Installed to GOPATH/bin$(NC)"

# Run unit tests
test:
	@echo "$(BLUE)Running unit tests...$(NC)"
	go test -v -race -timeout 30s ./...
	@echo "$(GREEN)Tests passed!$(NC)"

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
setup: tools mod
	@echo "$(BLUE)Setting up development environment...$(NC)"
	@if not exist .env copy .env.example .env 2>nul
	@if not exist dist mkdir dist
	@if not exist build mkdir build
	@echo "$(GREEN)Setup complete!$(NC)"

# Install development tools
tools:
	@echo "$(BLUE)Installing development tools...$(NC)"
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/cosmtrek/air@latest
	@echo "$(GREEN)Tools installed!$(NC)"

# Manage Go modules
mod:
	@echo "$(BLUE)Managing Go modules...$(NC)"
	go mod download
	go mod tidy
	go mod verify
	@echo "$(GREEN)Modules updated!$(NC)"

# Run linters
lint:
	@echo "Running linters..."
	@golangci-lint run

# Build Docker image
docker-build:
	@echo "$(BLUE)Building Docker image...$(NC)"
	docker build -t $(DOCKER_IMAGE):$(VERSION) -t $(DOCKER_IMAGE):latest .
	@echo "$(GREEN)Docker image built!$(NC)"

# Push Docker image
docker-push: docker-build
	@echo "$(BLUE)Pushing Docker image...$(NC)"
	docker push $(DOCKER_IMAGE):$(VERSION)
	docker push $(DOCKER_IMAGE):latest
	@echo "$(GREEN)Docker image pushed!$(NC)"

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
benchmark:
	@echo "$(BLUE)Running benchmarks...$(NC)"
	go test -bench=. -benchmem -run=^$$ ./... > benchmark.txt
	@echo "$(GREEN)Benchmark complete: benchmark.txt$(NC)"

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
	@echo "$(BLUE)Running security checks...$(NC)"
	@gosec -fmt json -out gosec-report.json ./... 2>nul || echo "$(YELLOW)gosec not installed$(NC)"
	@go list -json -deps ./... 2>nul | nancy sleuth 2>nul || echo "$(YELLOW)nancy not installed$(NC)"
	@echo "$(GREEN)Security checks complete!$(NC)"

# Start development server
serve: build
	@echo "$(BLUE)Starting web server...$(NC)"
	.\build\$(BINARY_NAME).exe serve web --port 8080

# Watch for changes
watch:
	@echo "$(BLUE)Watching for changes...$(NC)"
	@air 2>nul || echo "$(YELLOW)air not installed. Run: make tools$(NC)"

# Start development environment
dev:
	@echo "$(BLUE)Starting development environment...$(NC)"
	docker-compose up -d
	@echo "$(GREEN)Development environment ready!$(NC)"
	@echo "  Web UI: http://localhost:8080"
	@echo "  API: http://localhost:8080/api"

# Display build info
info:
	@echo "$(BLUE)Build Information:$(NC)"
	@echo "  Version:    $(VERSION)"
	@echo "  Git Commit: $(GIT_COMMIT)"
	@echo "  Build Time: $(BUILD_TIME)"
	@echo "  Binary:     $(BINARY_NAME)"

# Run all checks
check: fmt vet lint test

# CI pipeline
ci: clean check test-coverage build-all

# Quick build and run
run: build
	.\build\$(BINARY_NAME).exe
