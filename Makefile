# DriftMgr Makefile - AI-Optimized Development

.PHONY: help build test test-coverage test-security lint security-scan clean install

# Default target
help:
	@echo "DriftMgr AI-Optimized Development Commands"
	@echo "=========================================="
	@echo ""
	@echo "Development:"
	@echo "  build          Build the driftmgr binary"
	@echo "  install        Install driftmgr to GOPATH/bin"
	@echo "  clean          Clean build artifacts"
	@echo ""
	@echo "Testing & Quality:"
	@echo "  test           Run all tests"
	@echo "  test-coverage  Run tests with coverage (80% requirement)"
	@echo "  test-security  Run security-focused tests"
	@echo "  lint           Run linters (golangci-lint)"
	@echo "  security-scan  Run security scanning tools"
	@echo ""
	@echo "AI Optimization:"
	@echo "  ai-review      Run AI-assisted code review"
	@echo "  quality-gates  Run all quality gates"
	@echo "  metrics        Generate development metrics"

# Build configuration
BINARY_NAME=driftmgr
BUILD_DIR=dist
COVERAGE_DIR=coverage
VERSION=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/driftmgr
	@echo "✅ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Install to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME)..."
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
	@echo "✅ Installed to $(GOPATH)/bin/$(BINARY_NAME)"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -rf $(COVERAGE_DIR)
	rm -f coverage.out coverage.html
	rm -f *.log
	@echo "✅ Clean complete"

# Run all tests
test:
	@echo "Running tests..."
	go test -race -v ./...
	@echo "✅ Tests completed"

# Run tests with coverage (80% requirement)
test-coverage:
	@echo "Running tests with coverage analysis..."
	@mkdir -p $(COVERAGE_DIR)
	
	# Run tests with coverage
	go test -race -coverprofile=coverage.out ./...
	
	# Generate HTML coverage report
	go tool cover -html=coverage.out -o $(COVERAGE_DIR)/coverage.html
	
	# Check coverage threshold
	@coverage=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	echo "Coverage: $$coverage%"; \
	if [ $$coverage -lt 80 ]; then \
		echo "❌ Coverage $$coverage% is below 80% requirement"; \
		echo "Coverage report: $(COVERAGE_DIR)/coverage.html"; \
		exit 1; \
	else \
		echo "✅ Coverage $$coverage% meets 80% requirement"; \
	fi

# Run security-focused tests
test-security:
	@echo "Running security-focused tests..."
	go test -race -v -tags=security ./internal/security/...
	go test -race -v -tags=security ./internal/api/...
	@echo "✅ Security tests completed"

# Run linters
lint:
	@echo "Running linters..."
	
	# Install golangci-lint if not present
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.54.2; \
	fi
	
	# Run golangci-lint
	golangci-lint run --timeout=5m --config=.golangci.yml
	
	# Run go vet
	go vet ./...
	
	# Run staticcheck
	@if ! command -v staticcheck >/dev/null 2>&1; then \
		echo "Installing staticcheck..."; \
		go install honnef.co/go/tools/cmd/staticcheck@latest; \
	fi
	staticcheck ./...
	
	@echo "✅ Linting completed"

# Run security scanning tools
security-scan:
	@echo "Running security scans..."
	
	# Install security tools if not present
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "Installing gosec..."; \
		go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
	fi
	
	@if ! command -v govulncheck >/dev/null 2>&1; then \
		echo "Installing govulncheck..."; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
	fi
	
	# Run gosec SAST scan
	@echo "Running gosec SAST scan..."
	gosec -fmt json -out gosec-report.json ./...
	gosec -fmt sarif -out gosec-report.sarif ./...
	
	# Check for high/critical issues
	@if grep -q '"severity":"HIGH"' gosec-report.json || grep -q '"severity":"CRITICAL"' gosec-report.json; then \
		echo "❌ High or critical security issues found!"; \
		cat gosec-report.json; \
		exit 1; \
	fi
	
	# Run vulnerability check
	@echo "Running vulnerability check..."
	govulncheck ./...
	
	@echo "✅ Security scans completed"

# Run AI-assisted code review
ai-review:
	@echo "Running AI-assisted code review..."
	
	# Install security reviewer if not present
	@if [ ! -f "./internal/security/reviewer.go" ]; then \
		echo "❌ Security reviewer not found. Run 'make setup' first."; \
		exit 1; \
	fi
	
	# Run security review on all Go files
	go run ./cmd/security-review ./...
	
	@echo "✅ AI code review completed"

# Run all quality gates
quality-gates: lint test-coverage security-scan
	@echo "Running all quality gates..."
	@echo "✅ All quality gates passed"

# Generate development metrics
metrics:
	@echo "Generating development metrics..."
	@mkdir -p $(COVERAGE_DIR)
	
	# Generate test coverage metrics
	go test -coverprofile=coverage.out ./...
	@coverage=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	echo "Test Coverage: $$coverage%" > $(COVERAGE_DIR)/metrics.txt
	
	# Generate code complexity metrics
	@if command -v gocyclo >/dev/null 2>&1; then \
		gocyclo -over 10 . >> $(COVERAGE_DIR)/metrics.txt; \
	fi
	
	# Generate security metrics
	@if [ -f "gosec-report.json" ]; then \
		echo "Security Issues: $$(jq '.Stats.Total' gosec-report.json)" >> $(COVERAGE_DIR)/metrics.txt; \
	fi
	
	@echo "✅ Metrics generated: $(COVERAGE_DIR)/metrics.txt"

# Setup development environment
setup:
	@echo "Setting up development environment..."
	
	# Install required tools
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	
	# Create necessary directories
	mkdir -p $(BUILD_DIR) $(COVERAGE_DIR)
	
	@echo "✅ Development environment setup complete"

# CI/CD pipeline target
ci: quality-gates
	@echo "✅ CI/CD pipeline completed successfully"

# Development workflow
dev: clean build test-coverage
	@echo "✅ Development workflow completed"

# Release preparation
release: clean build test-coverage security-scan
	@echo "Preparing release..."
	@echo "Version: $(VERSION)"
	@echo "✅ Release preparation complete"

# Docker build
docker:
	@echo "Building Docker image..."
	docker build -t driftmgr:$(VERSION) .
	docker build -t driftmgr:latest .
	@echo "✅ Docker images built: driftmgr:$(VERSION), driftmgr:latest"

# Performance benchmarks
benchmark:
	@echo "Running performance benchmarks..."
	go test -bench=. -benchmem ./internal/...
	@echo "✅ Benchmarks completed"

# Memory profiling
profile:
	@echo "Running memory profiling..."
	go test -memprofile=mem.prof -cpuprofile=cpu.prof ./internal/...
	go tool pprof -http=:8080 mem.prof
	@echo "✅ Profiling completed (view at http://localhost:8080)"

# Generate mocks for testing
mocks:
	@echo "Generating mocks..."
	@if ! command -v mockgen >/dev/null 2>&1; then \
		echo "Installing mockgen..."; \
		go install github.com/golang/mock/mockgen@latest; \
	fi
	
	# Generate mocks for interfaces
	mockgen -source=internal/providers/types.go -destination=internal/providers/mocks/mock_providers.go
	mockgen -source=internal/state/manager.go -destination=internal/state/mocks/mock_state.go
	
	@echo "✅ Mocks generated"

# Update dependencies
deps:
	@echo "Updating dependencies..."
	go mod tidy
	go mod verify
	@echo "✅ Dependencies updated"

# Security audit
audit:
	@echo "Running security audit..."
	go list -json -deps ./... | nancy sleuth
	govulncheck ./...
	@echo "✅ Security audit completed"

# Code generation
generate:
	@echo "Generating code..."
	go generate ./...
	@echo "✅ Code generation completed"

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "✅ Code formatted"

# Show help for specific target
help-%:
	@echo "Help for target: $*"
	@make -n $* | head -20