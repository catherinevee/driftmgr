# DriftMgr Makefile
# Comprehensive build and development automation

.PHONY: help build test lint clean docker docs deploy

# Default target
.DEFAULT_GOAL := help

# Variables
BINARY_NAME := driftmgr
VERSION := $(shell git describe --tags --always --dirty)
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GO_VERSION := $(shell go version | awk '{print $$3}')
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -s -w"

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

## Help
help: ## Show this help message
	@echo "$(BLUE)DriftMgr - Infrastructure Drift Management$(NC)"
	@echo "$(BLUE)==========================================$(NC)"
	@echo ""
	@echo "$(GREEN)Available targets:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(YELLOW)%-20s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(GREEN)Phase-specific targets:$(NC)"
	@echo "  $(YELLOW)phase1$(NC)              Build and test Phase 1 (Drift Results)"
	@echo "  $(YELLOW)phase2$(NC)              Build and test Phase 2 (Remediation)"
	@echo "  $(YELLOW)phase3$(NC)              Build and test Phase 3 (State Management)"
	@echo "  $(YELLOW)phase4$(NC)              Build and test Phase 4 (Discovery)"
	@echo "  $(YELLOW)phase5$(NC)              Build and test Phase 5 (Configuration)"
	@echo "  $(YELLOW)phase6$(NC)              Build and test Phase 6 (Monitoring)"
	@echo ""
	@echo "$(GREEN)Current version:$(NC) $(VERSION)"
	@echo "$(GREEN)Go version:$(NC) $(GO_VERSION)"

## Development
dev: ## Start development environment
	@echo "$(BLUE)Starting development environment...$(NC)"
	docker-compose -f deployments/docker-compose.dev.yml up -d
	@echo "$(GREEN)Development environment started!$(NC)"
	@echo "$(YELLOW)API Server: http://localhost:8080$(NC)"
	@echo "$(YELLOW)Web Dashboard: http://localhost:3000$(NC)"
	@echo "$(YELLOW)Database: localhost:5432$(NC)"
	@echo "$(YELLOW)Redis: localhost:6379$(NC)"

dev-stop: ## Stop development environment
	@echo "$(BLUE)Stopping development environment...$(NC)"
	docker-compose -f deployments/docker-compose.dev.yml down
	@echo "$(GREEN)Development environment stopped!$(NC)"

## Building
build: ## Build the application
	@echo "$(BLUE)Building $(BINARY_NAME)...$(NC)"
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/server
	@echo "$(GREEN)Build complete!$(NC)"

build-all: ## Build for all platforms
	@echo "$(BLUE)Building for all platforms...$(NC)"
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/server
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 ./cmd/server
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe ./cmd/server
	GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-arm64.exe ./cmd/server
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/server
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 ./cmd/server
	@echo "$(GREEN)Multi-platform build complete!$(NC)"

## Testing
test: ## Run all tests
	@echo "$(BLUE)Running all tests...$(NC)"
	go test -v -race -coverprofile=coverage.out ./...
	@echo "$(GREEN)All tests completed!$(NC)"

test-unit: ## Run unit tests only
	@echo "$(BLUE)Running unit tests...$(NC)"
	go test -v -race -short ./internal/...
	@echo "$(GREEN)Unit tests completed!$(NC)"

test-integration: ## Run integration tests
	@echo "$(BLUE)Running integration tests...$(NC)"
	go test -v -race ./tests/integration/...
	@echo "$(GREEN)Integration tests completed!$(NC)"

test-api: ## Run API tests
	@echo "$(BLUE)Running API tests...$(NC)"
	go test -v -race ./tests/api/...
	@echo "$(GREEN)API tests completed!$(NC)"

test-performance: ## Run performance tests
	@echo "$(BLUE)Running performance tests...$(NC)"
	go test -v -race -timeout=10m ./tests/performance/...
	@echo "$(GREEN)Performance tests completed!$(NC)"

test-coverage: ## Generate test coverage report
	@echo "$(BLUE)Generating coverage report...$(NC)"
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

## Phase-specific testing
phase1: ## Build and test Phase 1 (Drift Results)
	@echo "$(BLUE)Testing Phase 1: Drift Results & History Management$(NC)"
	go test -v -race -coverprofile=coverage-phase1.out ./internal/api/drift/... ./internal/business/drift/... ./internal/storage/drift/...
	go test -v -race ./tests/api/drift/... ./tests/integration/drift/...
	@echo "$(GREEN)Phase 1 tests completed!$(NC)"

phase2: ## Build and test Phase 2 (Remediation)
	@echo "$(BLUE)Testing Phase 2: Remediation Engine$(NC)"
	go test -v -race -coverprofile=coverage-phase2.out ./internal/api/remediation/... ./internal/business/remediation/... ./internal/storage/remediation/... ./internal/jobs/...
	go test -v -race ./tests/api/remediation/... ./tests/integration/remediation/...
	@echo "$(GREEN)Phase 2 tests completed!$(NC)"

phase3: ## Build and test Phase 3 (State Management)
	@echo "$(BLUE)Testing Phase 3: Enhanced State Management$(NC)"
	go test -v -race -coverprofile=coverage-phase3.out ./internal/api/state/... ./internal/business/state/... ./internal/storage/state/...
	go test -v -race ./tests/api/state/... ./tests/integration/state/...
	@echo "$(GREEN)Phase 3 tests completed!$(NC)"

phase4: ## Build and test Phase 4 (Discovery)
	@echo "$(BLUE)Testing Phase 4: Advanced Discovery & Scanning$(NC)"
	go test -v -race -coverprofile=coverage-phase4.out ./internal/api/discovery/... ./internal/business/discovery/... ./internal/storage/discovery/... ./internal/providers/...
	go test -v -race ./tests/api/discovery/... ./tests/integration/discovery/...
	@echo "$(GREEN)Phase 4 tests completed!$(NC)"

phase5: ## Build and test Phase 5 (Configuration)
	@echo "$(BLUE)Testing Phase 5: Configuration & Provider Management$(NC)"
	go test -v -race -coverprofile=coverage-phase5.out ./internal/api/config/... ./internal/business/config/... ./internal/storage/config/...
	go test -v -race ./tests/api/config/... ./tests/integration/config/...
	@echo "$(GREEN)Phase 5 tests completed!$(NC)"

phase6: ## Build and test Phase 6 (Monitoring)
	@echo "$(BLUE)Testing Phase 6: Monitoring & Observability$(NC)"
	go test -v -race -coverprofile=coverage-phase6.out ./internal/api/monitoring/... ./internal/business/monitoring/... ./internal/storage/monitoring/...
	go test -v -race ./tests/api/monitoring/... ./tests/integration/monitoring/...
	@echo "$(GREEN)Phase 6 tests completed!$(NC)"

## Code Quality
lint: ## Run linters
	@echo "$(BLUE)Running linters...$(NC)"
	golangci-lint run --timeout=5m
	@echo "$(GREEN)Linting completed!$(NC)"

lint-fix: ## Run linters with auto-fix
	@echo "$(BLUE)Running linters with auto-fix...$(NC)"
	golangci-lint run --fix --timeout=5m
	@echo "$(GREEN)Linting with auto-fix completed!$(NC)"

fmt: ## Format code
	@echo "$(BLUE)Formatting code...$(NC)"
	go fmt ./...
	@echo "$(GREEN)Code formatting completed!$(NC)"

vet: ## Run go vet
	@echo "$(BLUE)Running go vet...$(NC)"
	go vet ./...
	@echo "$(GREEN)Go vet completed!$(NC)"

## Security
security: ## Run security checks
	@echo "$(BLUE)Running security checks...$(NC)"
	gosec ./...
	@echo "$(GREEN)Security checks completed!$(NC)"

security-audit: ## Run security audit
	@echo "$(BLUE)Running security audit...$(NC)"
	go list -json -deps ./... | nancy sleuth
	@echo "$(GREEN)Security audit completed!$(NC)"

## Documentation
docs: ## Generate documentation
	@echo "$(BLUE)Generating documentation...$(NC)"
	go run ./scripts/generate-api-docs.go --all-phases
	go run ./scripts/generate-implementation-report.go
	@echo "$(GREEN)Documentation generated!$(NC)"

docs-check: ## Check API documentation
	@echo "$(BLUE)Checking API documentation...$(NC)"
	go run ./scripts/check-api-docs.go --phase=all
	@echo "$(GREEN)API documentation check completed!$(NC)"

## Database
db-migrate: ## Run database migrations
	@echo "$(BLUE)Running database migrations...$(NC)"
	go run ./cmd/migrate/main.go up
	@echo "$(GREEN)Database migrations completed!$(NC)"

db-rollback: ## Rollback database migrations
	@echo "$(BLUE)Rolling back database migrations...$(NC)"
	go run ./cmd/migrate/main.go down
	@echo "$(GREEN)Database rollback completed!$(NC)"

db-reset: ## Reset database
	@echo "$(BLUE)Resetting database...$(NC)"
	go run ./cmd/migrate/main.go reset
	@echo "$(GREEN)Database reset completed!$(NC)"

## Docker
docker-build: ## Build Docker image
	@echo "$(BLUE)Building Docker image...$(NC)"
	docker build -t $(BINARY_NAME):$(VERSION) -f deployments/Dockerfile .
	docker tag $(BINARY_NAME):$(VERSION) $(BINARY_NAME):latest
	@echo "$(GREEN)Docker image built!$(NC)"

docker-run: ## Run Docker container
	@echo "$(BLUE)Running Docker container...$(NC)"
	docker run -p 8080:8080 -p 3000:3000 $(BINARY_NAME):latest
	@echo "$(GREEN)Docker container running!$(NC)"

docker-push: ## Push Docker image to registry
	@echo "$(BLUE)Pushing Docker image...$(NC)"
	docker push $(BINARY_NAME):$(VERSION)
	docker push $(BINARY_NAME):latest
	@echo "$(GREEN)Docker image pushed!$(NC)"

## Deployment
deploy-staging: ## Deploy to staging
	@echo "$(BLUE)Deploying to staging...$(NC)"
	kubectl apply -f deployments/kubernetes/staging/
	@echo "$(GREEN)Staging deployment completed!$(NC)"

deploy-production: ## Deploy to production
	@echo "$(BLUE)Deploying to production...$(NC)"
	kubectl apply -f deployments/kubernetes/production/
	@echo "$(GREEN)Production deployment completed!$(NC)"

## Cleanup
clean: ## Clean build artifacts
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	rm -rf bin/
	rm -f coverage*.out coverage*.html
	rm -rf dist/
	@echo "$(GREEN)Cleanup completed!$(NC)"

clean-docker: ## Clean Docker images and containers
	@echo "$(BLUE)Cleaning Docker artifacts...$(NC)"
	docker system prune -f
	docker image prune -f
	@echo "$(GREEN)Docker cleanup completed!$(NC)"

## CI/CD
ci-test: ## Run CI test suite
	@echo "$(BLUE)Running CI test suite...$(NC)"
	$(MAKE) lint
	$(MAKE) test
	$(MAKE) security
	$(MAKE) docs-check
	@echo "$(GREEN)CI test suite completed!$(NC)"

ci-build: ## Run CI build
	@echo "$(BLUE)Running CI build...$(NC)"
	$(MAKE) build-all
	$(MAKE) docker-build
	@echo "$(GREEN)CI build completed!$(NC)"

## Release
release: ## Create a new release
	@echo "$(BLUE)Creating release...$(NC)"
	@if [ -z "$(VERSION)" ]; then echo "$(RED)Error: VERSION is required$(NC)"; exit 1; fi
	git tag -a $(VERSION) -m "Release $(VERSION)"
	git push origin $(VERSION)
	@echo "$(GREEN)Release $(VERSION) created!$(NC)"

## Monitoring
logs: ## View application logs
	@echo "$(BLUE)Viewing application logs...$(NC)"
	docker-compose -f deployments/docker-compose.yml logs -f

status: ## Check application status
	@echo "$(BLUE)Checking application status...$(NC)"
	curl -s http://localhost:8080/health | jq .

metrics: ## View application metrics
	@echo "$(BLUE)Viewing application metrics...$(NC)"
	curl -s http://localhost:8080/api/v1/metrics | jq .

## Development Tools
install-tools: ## Install development tools
	@echo "$(BLUE)Installing development tools...$(NC)"
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	go install github.com/sonatypecommunity/nancy@latest
	@echo "$(GREEN)Development tools installed!$(NC)"

update-deps: ## Update dependencies
	@echo "$(BLUE)Updating dependencies...$(NC)"
	go get -u ./...
	go mod tidy
	@echo "$(GREEN)Dependencies updated!$(NC)"

## Validation
validate-all: ## Validate all phases
	@echo "$(BLUE)Validating all phases...$(NC)"
	$(MAKE) phase1
	$(MAKE) phase2
	$(MAKE) phase3
	$(MAKE) phase4
	$(MAKE) phase5
	$(MAKE) phase6
	@echo "$(GREEN)All phases validated!$(NC)"

validate-phase: ## Validate specific phase (usage: make validate-phase PHASE=1)
	@echo "$(BLUE)Validating Phase $(PHASE)...$(NC)"
	$(MAKE) phase$(PHASE)
	@echo "$(GREEN)Phase $(PHASE) validated!$(NC)"

## Quick Commands
quick-test: ## Quick test (unit tests only)
	@echo "$(BLUE)Running quick tests...$(NC)"
	go test -v -short ./internal/...
	@echo "$(GREEN)Quick tests completed!$(NC)"

quick-build: ## Quick build (current platform only)
	@echo "$(BLUE)Quick building...$(NC)"
	go build -o bin/$(BINARY_NAME) ./cmd/server
	@echo "$(GREEN)Quick build completed!$(NC)"

## Information
version: ## Show version information
	@echo "$(BLUE)Version Information$(NC)"
	@echo "$(GREEN)Application:$(NC) $(BINARY_NAME)"
	@echo "$(GREEN)Version:$(NC) $(VERSION)"
	@echo "$(GREEN)Build Time:$(NC) $(BUILD_TIME)"
	@echo "$(GREEN)Go Version:$(NC) $(GO_VERSION)"

info: ## Show project information
	@echo "$(BLUE)Project Information$(NC)"
	@echo "$(GREEN)Name:$(NC) DriftMgr"
	@echo "$(GREEN)Description:$(NC) Infrastructure Drift Management Platform"
	@echo "$(GREEN)Repository:$(NC) https://github.com/catherinevee/driftmgr"
	@echo "$(GREEN)Documentation:$(NC) https://docs.driftmgr.com"
	@echo "$(GREEN)Status Page:$(NC) https://status.driftmgr.com"