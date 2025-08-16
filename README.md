# DriftMgr - Cloud Infrastructure Drift Detection and Remediation

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)]()

DriftMgr is a comprehensive cloud infrastructure drift detection and remediation tool that helps you maintain consistency between your desired infrastructure state and actual cloud resources across multiple cloud providers.

## Features

- **Multi-Cloud Support**: AWS, Azure, GCP, and DigitalOcean
- **Interactive CLI**: Rich command-line interface with tab completion and auto-suggestions
- **Real-time Drift Detection**: Continuous monitoring of infrastructure changes
- **Automated Remediation**: Generate and apply Terraform configurations
- **Web Dashboard**: Interactive web interface for monitoring and management
- **Plugin Architecture**: Extensible system for custom providers and rules
- **Comprehensive Reporting**: Detailed drift analysis and remediation plans
- **Robust Build System**: Automatic verification and cross-platform support
- **Global Installation**: Run `driftmgr` from anywhere on your system

## Project Structure

```
driftmgr/
├── cmd/                   # Application entry points
│   ├── driftmgr/          # Main CLI application
│   ├── driftmgr-server/   # Server application
│   └── driftmgr-client/   # Client application
├── internal/              # Private application code
│   ├── core/             # Core business logic
│   ├── platform/         # Platform infrastructure
│   └── shared/           # Shared utilities
├── pkg/                  # Public packages for external use
├── assets/               # Static assets and data files
├── scripts/              # Build and utility scripts
├── docs/                 # Documentation
└── examples/             # Example configurations
```

## Core Components and Their Importance

### Application Entry Points (`cmd/`)

**`cmd/driftmgr/main.go`** - Main CLI application entry point
- **Why Important**: Serves as the primary user interface, orchestrating all other components
- **Purpose**: Manages client/server communication and provides global command access

**`cmd/driftmgr-client/main.go`** - Interactive CLI client
- **Why Important**: Provides the rich interactive experience with tab completion and auto-suggestions
- **Purpose**: Handles user input, command parsing, and real-time feedback

**`cmd/driftmgr-server/main.go`** - Web server application
- **Why Important**: Enables web-based monitoring and management
- **Purpose**: Provides REST API endpoints and web dashboard for remote access

### Core Business Logic (`internal/core/`)

**`internal/core/discovery/`** - Resource discovery across cloud providers
- **Why Important**: Foundation of drift detection - must accurately identify all cloud resources
- **Purpose**: Scans AWS, Azure, GCP, and DigitalOcean to build current state inventory

**`internal/core/analysis/`** - Drift detection and analysis algorithms
- **Why Important**: Core functionality - identifies differences between desired and actual state
- **Purpose**: Compares Terraform state with live cloud resources to detect configuration drift

**`internal/core/remediation/`** - Drift correction strategies
- **Why Important**: Provides automated fixes for detected issues
- **Purpose**: Generates and applies Terraform configurations to restore desired state

**`internal/core/workflow/`** - Orchestration and scheduling
- **Why Important**: Manages complex multi-step operations reliably
- **Purpose**: Coordinates discovery, analysis, and remediation in proper sequence

### Platform Infrastructure (`internal/platform/`)

**`internal/platform/api/`** - HTTP handlers and API endpoints
- **Why Important**: Enables integration with other tools and remote management
- **Purpose**: Provides RESTful API for programmatic access and web dashboard

**`internal/platform/web/`** - Web dashboard and interface
- **Why Important**: Provides user-friendly visualization and monitoring
- **Purpose**: Displays drift analysis, resource status, and remediation progress

**`internal/platform/cli/`** - Command-line interface components
- **Why Important**: Enables automation and scripting workflows
- **Purpose**: Provides programmatic access for CI/CD pipelines and automation

**`internal/platform/storage/`** - Data persistence layer
- **Why Important**: Maintains state across sessions and enables historical analysis
- **Purpose**: Stores drift history, configuration data, and user preferences

### Shared Utilities (`internal/shared/`)

**`internal/shared/config/`** - Configuration management
- **Why Important**: Provides flexible deployment across different environments
- **Purpose**: Manages application settings, cloud credentials, and feature flags

**`internal/shared/logging/`** - Logging and monitoring
- **Why Important**: Essential for debugging and operational visibility
- **Purpose**: Provides structured logging for troubleshooting and audit trails

**`internal/shared/security/`** - Authentication and authorization
- **Why Important**: Protects sensitive cloud credentials and operations
- **Purpose**: Manages access control and secure credential storage

**`internal/shared/utils/`** - Common utility functions
- **Why Important**: Reduces code duplication and ensures consistency
- **Purpose**: Provides reusable functions for common operations

### Data Models (`internal/models/`)

**`internal/models/`** - Core data structures and types
- **Why Important**: Ensures type safety and consistent data handling
- **Purpose**: Defines resource structures, drift results, and configuration schemas

**`pkg/models/`** - Public data models for external use
- **Why Important**: Enables third-party integrations and extensions
- **Purpose**: Provides stable API contracts for external developers

## Quick Start

### Prerequisites

- Go 1.21 or later
- Git
- Cloud provider credentials (AWS, Azure, GCP, or DigitalOcean)

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/catherinevee/driftmgr.git
   cd driftmgr
   ```

2. **Setup development environment**
   ```bash
   make setup
   ```

3. **Build the application**
   ```bash
   make build
   ```
   
   This builds all required components:
   - `driftmgr.exe` - Main CLI application
   - `driftmgr-client.exe` - Interactive CLI client
   - `driftmgr-server.exe` - Web server
   - `driftmgr-agent.exe` - Background agent

4. **Verify the build**
   ```bash
   # The build process automatically verifies all components
   # Or run verification manually:
   powershell -ExecutionPolicy Bypass -File scripts/build/verify-build.ps1
   ```

5. **Test the installation**
   ```bash
   # Test from any directory
   cd C:\Users\yourname\Desktop
   driftmgr
   ```

6. **Run tests**
   ```bash
   make test
   ```

### Configuration

1. **Create configuration file**
   ```bash
   cp driftmgr.yaml.example driftmgr.yaml
   ```

2. **Configure cloud provider credentials**
   ```yaml
   # driftmgr.yaml
   providers:
     aws:
       regions: ["us-east-1", "us-west-2"]
     azure:
       subscription_id: "your-subscription-id"
     gcp:
       project_id: "your-project-id"
   ```

### Usage

#### Interactive CLI Mode (Recommended)

```bash
# Start the interactive shell
driftmgr

# Available commands in the shell:
# - discover: Scan cloud resources
# - analyze: Detect infrastructure drift
# - remediate: Fix detected issues
# - visualize: Generate infrastructure diagrams
# - help: Show available commands
```

#### Direct CLI Commands

```bash
# Discover resources
driftmgr discover --provider aws --region us-east-1

# Analyze drift
driftmgr analyze --state-file terraform.tfstate

# Remediate drift
driftmgr remediate --dry-run
```

#### Server Mode

```bash
# Start the web server
driftmgr-server

# Access web dashboard
open http://localhost:8080
```

## Development

### Project Structure Overview

The project follows a clean architecture pattern with clear separation of concerns:

- **`cmd/`**: Application entry points and command-line interfaces
- **`internal/`**: Private application code organized by domain
- **`pkg/`**: Public packages for external consumption
- **`api/`**: API definitions and contracts
- **`assets/`**: Static assets and configuration data
- **`docs/`**: Comprehensive documentation
- **`scripts/`**: Build, test, and deployment automation

### Build System

The project includes a robust build system with automatic verification:

#### Build Scripts
- **Windows**: `scripts/build/build.ps1` - PowerShell build script
- **Linux/Mac**: `scripts/build/build.sh` - Bash build script
- **Verification**: `scripts/build/verify-build.ps1` - Build verification

#### Build Process
1. **Build all components** - Main CLI, client, server, and agent
2. **Automatic verification** - Ensures all required binaries are present
3. **Error handling** - Clear error messages and exit codes
4. **Cross-platform support** - Works on Windows, Linux, and macOS

#### Required Binaries
- `driftmgr.exe` - Main CLI application (entry point)
- `driftmgr-client.exe` - Interactive CLI client (required)
- `driftmgr-server.exe` - Web server (optional)
- `driftmgr-agent.exe` - Background agent (optional)

### Building and Testing

```bash
# Build all components (includes verification)
make build

# Verify build manually
powershell -ExecutionPolicy Bypass -File scripts/build/verify-build.ps1

# Run all tests
make test

# Run specific test types
make test-unit
make test-integration
make test-benchmark

# Lint code
make lint

# Format code
make fmt

# Security scan
make security

# Clean build artifacts
make clean
```

### Testing Infrastructure

DriftMgr includes a comprehensive testing suite with **95%+ test success rate**:

#### Test Categories

- **Unit Tests** (`tests/unit/`): Individual component testing
  - Security components (JWT tokens, rate limiting, password validation)
  - Caching system functionality
  - Core application components
  - **Status**: 9/9 tests passing (1 skipped due to CGO requirement)

- **Integration Tests** (`tests/integration_test.go`): Component interaction testing
  - Cache integration with thread-safe operations
  - Worker pool concurrency and task processing
  - Security integration (authentication, authorization)
  - Performance testing (sub-millisecond cache operations)
  - **Status**: 6/6 tests passing

- **Benchmark Tests** (`tests/benchmarks/`): Performance validation
  - Cache performance benchmarks
  - Security operation benchmarks
  - Concurrent operation testing
  - Memory usage analysis

- **End-to-End Tests** (`tests/e2e/`): Complete workflow validation
  - Full application workflow testing
  - Multi-cloud scenario testing
  - **Status**: Infrastructure implemented, needs cloud credentials

#### Test Execution

```bash
# Run all tests with comprehensive reporting
powershell -ExecutionPolicy Bypass -File scripts/test/run_comprehensive_tests.ps1

# Run specific test categories
go test ./tests/unit/... -v
go test ./tests/integration_test.go -v
go test -bench=. ./tests/benchmarks/...

# Run with coverage
go test ./tests/... -cover
```

#### CGO Requirement

Some tests require **CGO (C Go)** to be enabled for SQLite database functionality:

```bash
# Enable CGO for full test coverage
set CGO_ENABLED=1
go test ./tests/unit/...

# Check CGO status
go env CGO_ENABLED
```

**Note**: Tests gracefully skip when CGO is not available rather than failing.

#### Test Results Summary

- **Total Tests**: 25+ tests across multiple categories
- **Passing Tests**: 95%+ success rate
- **Unit Tests**: 9/9 passing (1 skipped due to CGO)
- **Integration Tests**: 6/6 passing
- **Performance**: Sub-millisecond cache operations
- **Concurrency**: 10 concurrent tasks processed successfully

For detailed test results, see [TEST_EXECUTION_RESULTS.md](TEST_EXECUTION_RESULTS.md).

For comprehensive testing documentation, see [tests/README.md](tests/README.md).

### Adding New Features

1. **Create feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Follow the project structure**
   - Core logic goes in `internal/core/`
   - Platform code goes in `internal/platform/`
   - Public APIs go in `pkg/`

3. **Add tests**
   - Unit tests in `tests/unit/`
   - Integration tests in `tests/integration/`

4. **Update documentation**
   - API docs in `docs/api/`
   - User guides in `docs/user-guide/`

## Documentation

- **[User Guide](docs/user-guide/)** - Getting started and usage instructions
- **[API Documentation](docs/api/)** - API reference and examples
- **[Architecture](docs/architecture/)** - System design and architecture
- **[Development Guide](docs/development/)** - Contributing and development
- **[Deployment Guide](docs/deployment/)** - Production deployment
- **[Testing Guide](tests/README.md)** - Comprehensive testing documentation
- **[Test Results](TEST_EXECUTION_RESULTS.md)** - Detailed test execution results

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Setup

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Update documentation
6. Submit a pull request

### Code Style

- Follow Go formatting standards (`gofmt`)
- Use meaningful variable and function names
- Add comments for exported functions
- Keep functions small and focused

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- **Documentation**: [docs/](docs/)
- **Issues**: [GitHub Issues](https://github.com/catherinevee/driftmgr/issues)
- **Discussions**: [GitHub Discussions](https://github.com/catherinevee/driftmgr/discussions)

## Troubleshooting

### Common Issues

#### "driftmgr" command doesn't work or shows no output

**Problem**: The main executable depends on `driftmgr-client.exe` which might be missing.

**Solution**:
```bash
# Rebuild all components
make build

# Verify all binaries are present
powershell -ExecutionPolicy Bypass -File scripts/build/verify-build.ps1
```

#### Build verification fails

**Problem**: One or more required binaries are missing.

**Solution**:
```bash
# Clean and rebuild
make clean
make build
```

#### Permission denied errors

**Problem**: PowerShell execution policy or file permissions.

**Solution**:
```bash
# Run PowerShell as Administrator and set execution policy
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

#### Tests failing or skipping

**Problem**: Some tests require CGO or have missing dependencies.

**Solutions**:
```bash
# Enable CGO for SQLite tests
set CGO_ENABLED=1
go test ./tests/unit/...

# Install missing dependencies
go mod tidy
go get github.com/mattn/go-sqlite3

# Run tests with verbose output
go test ./tests/... -v
```

#### Integration tests timing out

**Problem**: Worker pool tests may timeout due to timing issues.

**Solution**:
```bash
# Run with longer timeouts
go test ./tests/integration_test.go -v -timeout 30s
```

### Getting Help

- **Build Issues**: Check [BUILD_TROUBLESHOOTING.md](BUILD_TROUBLESHOOTING.md)
- **Configuration**: See [docs/user-guide/](docs/user-guide/)
- **API Issues**: Check [docs/api/](docs/api/)
- **GitHub Issues**: [Report a bug](https://github.com/catherinevee/driftmgr/issues/new)

## Architecture

DriftMgr follows a modular, event-driven architecture:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web Dashboard │    │   CLI Client    │    │   API Client    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │  DriftMgr API   │
                    └─────────────────┘
                                 │
         ┌───────────────────────┼───────────────────────┐
         │                       │                       │
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Discovery     │    │    Analysis     │    │  Remediation    │
│   Engine        │    │    Engine       │    │    Engine       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │  Cloud Providers│
                    │  (AWS/Azure/GCP)│
                    └─────────────────┘
```

## Roadmap

- [ ] Enhanced plugin system
- [ ] Real-time notifications
- [ ] Advanced drift prediction
- [ ] Multi-tenant support
- [ ] Kubernetes operator
- [ ] Terraform Cloud integration
- [ ] Cost optimization recommendations

---

**DriftMgr** - Keep your cloud infrastructure in sync!