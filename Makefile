# Makefile for driftmgr

.PHONY: build test clean install run fmt vet tidy

# Build the application
build:
	go build -o bin/driftmgr cmd/main.go

# Build for multiple platforms
build-all:
	GOOS=windows GOARCH=amd64 go build -o bin/driftmgr-windows-amd64.exe cmd/main.go
	GOOS=linux GOARCH=amd64 go build -o bin/driftmgr-linux-amd64 cmd/main.go
	GOOS=darwin GOARCH=amd64 go build -o bin/driftmgr-darwin-amd64 cmd/main.go
	GOOS=darwin GOARCH=arm64 go build -o bin/driftmgr-darwin-arm64 cmd/main.go

# Install the application
install:
	go install cmd/main.go

# Run the application
run:
	go run cmd/main.go

# Run with interactive mode
interactive:
	go run cmd/main.go interactive

# Test the application
test:
	go test ./...

# Test with verbose output
test-verbose:
	go test -v ./...

# Test individual packages
test-models:
	go test -v ./internal/models

test-discovery:
	go test -v ./internal/discovery

test-tui:
	go test -v ./internal/tui

test-importer:
	go test -v ./internal/importer

# Test with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Format the code
fmt:
	go fmt ./...

# Vet the code
vet:
	go vet ./...

# Tidy up dependencies
tidy:
	go mod tidy

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Run linters
lint:
	golangci-lint run

# Initialize go modules if not present
init:
	go mod init github.com/catherinevee/driftmgr || true
	go mod tidy

# Download dependencies
deps:
	go mod download

# Generate mocks (if using mockgen)
mocks:
	go generate ./...

# Run all checks
check: fmt vet test

# Development setup
dev-setup:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Example commands
example-discover:
	go run cmd/main.go discover --provider aws --region us-east-1

example-import:
	go run cmd/main.go import --file examples/sample-resources.csv --dry-run

example-config:
	go run cmd/main.go config init

# Help
help:
	@echo "Available targets:"
	@echo "  build          - Build the application"
	@echo "  build-all      - Build for multiple platforms"
	@echo "  install        - Install the application"
	@echo "  run            - Run the application"
	@echo "  interactive    - Run in interactive mode"
	@echo "  test           - Run tests"
	@echo "  test-verbose   - Run tests with verbose output"
	@echo "  test-models    - Test models package only"
	@echo "  test-discovery - Test discovery package only"
	@echo "  test-tui       - Test TUI package only"
	@echo "  test-importer  - Test importer package only"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  fmt            - Format code"
	@echo "  vet            - Vet code"
	@echo "  tidy           - Tidy dependencies"
	@echo "  clean          - Clean build artifacts"
	@echo "  lint           - Run linters"
	@echo "  check          - Run all checks"
	@echo "  dev-setup      - Setup development environment"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run Docker container"
	@echo "  docker-dev     - Start development environment"
	@echo "  ci-local       - Run CI checks locally"
	@echo "  release-local  - Test release build locally"
	@echo "  help           - Show this help"

# Docker targets
docker-build:
	docker build -t driftmgr:latest .

docker-run:
	docker run --rm -it driftmgr:latest

docker-dev:
	docker-compose up driftmgr-dev

docker-test:
	docker-compose up driftmgr-test

# CI/CD targets
ci-local: fmt vet lint test
	@echo "✅ All CI checks passed locally"

release-local:
	@echo "🚀 Testing release build locally..."
	rm -rf dist/
	mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/driftmgr-linux-amd64 ./cmd/driftmgr
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o dist/driftmgr-darwin-amd64 ./cmd/driftmgr
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/driftmgr-windows-amd64.exe ./cmd/driftmgr
	@echo "✅ Release build completed successfully"

# Security targets
security-scan:
	@echo "🔍 Running security scan..."
	go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...
