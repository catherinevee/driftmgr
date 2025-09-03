# DriftMgr Development Guide

This guide helps you set up a local development environment for contributing to DriftMgr.

## Table of Contents
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Development Setup](#development-setup)
- [Project Structure](#project-structure)
- [Development Workflow](#development-workflow)
- [Testing](#testing)
- [Debugging](#debugging)
- [Code Style](#code-style)
- [Common Tasks](#common-tasks)
- [Troubleshooting](#troubleshooting)

## Prerequisites

### Required Software
- **Go 1.23+** - [Download](https://go.dev/dl/)
- **Git** - [Download](https://git-scm.com/downloads)
- **Docker Desktop** - [Download](https://www.docker.com/products/docker-desktop)
- **Make** - Windows: Use Git Bash or WSL2

### Recommended Tools
- **VS Code** with Go extension
- **GoLand** IDE
- **GitHub CLI** - `gh` command
- **Air** - Live reload for Go apps
- **golangci-lint** - Linting tool

### Platform-Specific Setup

#### Windows
```powershell
# Install Chocolatey (if not installed)
Set-ExecutionPolicy Bypass -Scope Process -Force
[System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))

# Install tools
choco install golang git docker-desktop make

# Install Go tools
go install github.com/cosmtrek/air@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

#### macOS
```bash
# Install Homebrew (if not installed)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install tools
brew install go git docker make

# Install Go tools
go install github.com/cosmtrek/air@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

#### Linux
```bash
# Ubuntu/Debian
sudo apt update
sudo apt install -y golang git docker.io docker-compose make

# Install Go tools
go install github.com/cosmtrek/air@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Add user to docker group
sudo usermod -aG docker $USER
newgrp docker
```

## Quick Start

### 1. Clone the Repository
```bash
git clone https://github.com/catherinevee/driftmgr.git
cd driftmgr
```

### 2. Set Up Environment
```bash
# Copy environment template
cp .env.example .env

# Edit .env with your cloud credentials (optional for testing)
# AWS, Azure, GCP credentials can be configured
```

### 3. Install Dependencies
```bash
# Install Go dependencies and tools
make setup

# Verify installation
make info
```

### 4. Run Tests
```bash
# Run unit tests
make test

# Run all tests with coverage
make test-coverage
```

### 5. Build and Run
```bash
# Build the application
make build

# Run the application
make run

# Or start the web server
make serve
```

### 6. Start Development Environment
```bash
# Start all services (PostgreSQL, Redis, LocalStack, etc.)
make dev

# Access services:
# - Web UI: http://localhost:8080
# - API: http://localhost:8080/api
# - PostgreSQL: localhost:5432
# - Redis: localhost:6379
# - LocalStack: http://localhost:4566
# - MinIO: http://localhost:9001
# - Grafana: http://localhost:3000 (admin/admin)
# - Jaeger: http://localhost:16686
```

## Development Setup

### IDE Configuration

#### VS Code
Create `.vscode/settings.json`:
```json
{
  "go.useLanguageServer": true,
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "package",
  "go.formatTool": "goimports",
  "go.testFlags": ["-v", "-race"],
  "go.testTimeout": "60s",
  "go.buildTags": "",
  "editor.formatOnSave": true,
  "[go]": {
    "editor.codeActionsOnSave": {
      "source.organizeImports": true
    }
  }
}
```

Create `.vscode/launch.json`:
```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch DriftMgr",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/cmd/driftmgr",
      "args": ["serve", "web", "--port", "8080"],
      "env": {
        "DRIFTMGR_ENV": "development",
        "DRIFTMGR_LOG_LEVEL": "debug"
      }
    },
    {
      "name": "Debug Tests",
      "type": "go",
      "request": "launch",
      "mode": "test",
      "program": "${workspaceFolder}/...",
      "args": ["-test.v"]
    }
  ]
}
```

#### GoLand
1. Open project in GoLand
2. Configure Go SDK: File → Settings → Go → GOROOT
3. Enable Go Modules: File → Settings → Go → Go Modules
4. Configure formatting: File → Settings → Editor → Code Style → Go
5. Set up Run Configuration:
   - Run → Edit Configurations
   - Add Go Build
   - Package: `github.com/catherinevee/driftmgr/cmd/driftmgr`
   - Working Directory: Project root

### Git Hooks
```bash
# Install pre-commit hooks
make setup-hooks

# Or manually create .git/hooks/pre-commit
cat > .git/hooks/pre-commit << 'EOF'
#!/bin/sh
make check
EOF
chmod +x .git/hooks/pre-commit
```

## Project Structure

```
driftmgr/
├── cmd/                      # Application entry points
│   ├── driftmgr/            # Main CLI application
│   └── driftmgr-server/     # API server
├── internal/                 # Private application code
│   ├── api/                 # REST API implementation
│   ├── providers/           # Cloud provider implementations
│   ├── drift/               # Drift detection logic
│   ├── state/               # State management
│   ├── remediation/         # Remediation engine
│   └── monitoring/          # Monitoring and metrics
├── pkg/                     # Public libraries
├── web/                     # Web UI assets
├── configs/                 # Configuration files
├── scripts/                 # Utility scripts
├── tests/                   # Test files
│   ├── unit/               # Unit tests
│   ├── integration/        # Integration tests
│   └── e2e/                # End-to-end tests
├── docs/                    # Documentation
└── examples/                # Example configurations
```

## Development Workflow

### 1. Create a Feature Branch
```bash
# Create and switch to a new branch
git checkout -b feature/your-feature-name

# Or for bug fixes
git checkout -b fix/issue-description
```

### 2. Make Changes
```bash
# Watch for changes and auto-rebuild
make watch

# Or manually rebuild after changes
make build
```

### 3. Test Your Changes
```bash
# Run tests for your package
go test -v ./internal/your-package/...

# Run all tests
make test-all

# Run specific test
go test -v -run TestYourFunction ./internal/your-package
```

### 4. Format and Lint
```bash
# Format code
make fmt

# Run linters
make lint

# Run security checks
make security

# Run all checks
make check
```

### 5. Commit Changes
```bash
# Stage changes
git add .

# Commit with conventional commit message
git commit -m "feat: add new feature"
# or
git commit -m "fix: resolve issue with X"
# or
git commit -m "docs: update README"
```

### 6. Push and Create PR
```bash
# Push to your fork
git push origin feature/your-feature-name

# Create pull request
gh pr create --title "feat: your feature" --body "Description of changes"
```

## Testing

### Unit Tests
```bash
# Run all unit tests
make test

# Run with verbose output
go test -v ./...

# Run with race detection
go test -race ./...

# Run specific package tests
go test -v ./internal/providers/aws/...
```

### Integration Tests
```bash
# Start test services
docker-compose -f docker-compose.test.yml up -d

# Run integration tests
make test-integration

# Or with environment variables
INTEGRATION_TESTS=true go test -tags=integration ./tests/integration/...
```

### End-to-End Tests
```bash
# Run E2E tests
make test-e2e

# Run specific E2E test
go test -v -tags=e2e -run TestE2EWorkflow ./tests/e2e/
```

### Test Coverage
```bash
# Generate coverage report
make test-coverage

# View coverage in browser
open coverage.html  # macOS
start coverage.html  # Windows
xdg-open coverage.html  # Linux

# Check coverage for specific package
go test -cover ./internal/drift/...
```

### Benchmark Tests
```bash
# Run benchmarks
make benchmark

# Run specific benchmark
go test -bench=BenchmarkDriftDetection -benchmem ./internal/drift/

# Compare benchmarks
go test -bench=. -count=10 ./internal/drift/ > old.txt
# Make changes
go test -bench=. -count=10 ./internal/drift/ > new.txt
benchstat old.txt new.txt
```

## Debugging

### Debug with Delve
```bash
# Install Delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug the application
dlv debug ./cmd/driftmgr -- serve web --port 8080

# Debug a test
dlv test ./internal/drift -- -test.run TestDetectDrift

# Common Delve commands:
# break main.main - Set breakpoint
# continue - Continue execution
# next - Step over
# step - Step into
# print varName - Print variable
# locals - Show local variables
```

### Debug with VS Code
1. Set breakpoints by clicking left of line numbers
2. Press F5 or Run → Start Debugging
3. Use Debug Console for interactive debugging

### Logging
```go
// Use structured logging
import "github.com/catherinevee/driftmgr/internal/logger"

logger.Info("Processing resource", 
    "resource_id", resourceID,
    "resource_type", resourceType)

logger.Debug("Detailed info",
    "state", state,
    "config", config)

logger.Error("Operation failed",
    "error", err,
    "context", ctx)
```

### Environment Variables for Debugging
```bash
# Enable debug logging
export DRIFTMGR_LOG_LEVEL=debug

# Enable Go race detector
export GORACE="log_path=/tmp/race"

# Enable CPU profiling
export DRIFTMGR_CPU_PROFILE=/tmp/cpu.prof

# Enable memory profiling
export DRIFTMGR_MEM_PROFILE=/tmp/mem.prof

# Run with debugging
./driftmgr serve web --debug
```

## Code Style

### Go Conventions
- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` and `goimports`
- Write idiomatic Go code
- Keep functions small and focused
- Use meaningful variable names

### Naming Conventions
```go
// Packages: lowercase, no underscores
package driftdetector

// Interfaces: end with -er
type Scanner interface {
    Scan(ctx context.Context) error
}

// Constants: CamelCase or CAPS
const MaxRetries = 3
const DEFAULT_TIMEOUT = 30

// Private functions: camelCase
func processResource(r Resource) error

// Public functions: CamelCase
func DetectDrift(state State) error

// Structs: CamelCase
type DriftDetector struct {
    provider Provider
}
```

### Error Handling
```go
// Always check errors
result, err := someFunction()
if err != nil {
    return fmt.Errorf("failed to process: %w", err)
}

// Use error wrapping
if err := validateInput(input); err != nil {
    return fmt.Errorf("validation failed: %w", err)
}

// Custom errors
var ErrNotFound = errors.New("resource not found")
```

### Comments
```go
// Package driftmgr provides drift detection capabilities.
package driftmgr

// DriftDetector analyzes infrastructure drift.
// It compares actual cloud resources with desired state.
type DriftDetector struct {
    // provider is the cloud provider interface
    provider Provider
}

// DetectDrift identifies configuration drift.
// It returns a list of drifted resources or an error.
func (d *DriftDetector) DetectDrift(ctx context.Context) ([]*Drift, error) {
    // Implementation
}
```

## Common Tasks

### Adding a New Provider
1. Create provider package: `internal/providers/newprovider/`
2. Implement `CloudProvider` interface
3. Add provider factory in `internal/providers/factory.go`
4. Write tests in `*_test.go`
5. Update documentation

### Adding a New Command
1. Create command file: `cmd/driftmgr/commands/newcmd.go`
2. Implement command logic
3. Register in `cmd/driftmgr/main.go`
4. Add tests
5. Update help documentation

### Updating Dependencies
```bash
# Update all dependencies
go get -u ./...
go mod tidy

# Update specific dependency
go get -u github.com/aws/aws-sdk-go-v2

# Check for vulnerabilities
go list -json -deps ./... | nancy sleuth
```

## Troubleshooting

### Build Issues
```bash
# Clean build cache
go clean -cache
make clean

# Rebuild with verbose output
go build -v ./cmd/driftmgr

# Check for missing dependencies
go mod download
go mod verify
```

### Test Failures
```bash
# Run tests with more output
go test -v -count=1 ./...

# Disable test caching
go test -count=1 ./...

# Run tests sequentially
go test -p 1 ./...
```

### Docker Issues
```bash
# Reset Docker environment
docker-compose down -v
docker system prune -a

# Rebuild without cache
docker-compose build --no-cache

# Check logs
docker-compose logs -f service-name
```

### Performance Issues
```bash
# Profile CPU usage
go test -cpuprofile cpu.prof -bench .
go tool pprof cpu.prof

# Profile memory usage
go test -memprofile mem.prof -bench .
go tool pprof mem.prof

# Trace execution
go test -trace trace.out ./...
go tool trace trace.out
```

## Getting Help

- **Documentation**: [docs/](../docs/)
- **Examples**: [examples/](../examples/)
- **Issues**: [GitHub Issues](https://github.com/catherinevee/driftmgr/issues)
- **Discussions**: [GitHub Discussions](https://github.com/catherinevee/driftmgr/discussions)
- **Slack**: Join #driftmgr channel (if available)

## Next Steps

- Read [CONTRIBUTING.md](../CONTRIBUTING.md) for contribution guidelines
- Review [API_REFERENCE.md](./API_REFERENCE.md) for API documentation
- Check [examples/](../examples/) for usage examples
- Join the community discussions