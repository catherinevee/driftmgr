# Claude AI Assistant Guide for DriftMgr

This document provides context and guidelines for AI assistants (particularly Claude) when working with the DriftMgr codebase.

## Project Overview

DriftMgr is a multi-cloud infrastructure drift detection and remediation tool that:
- Detects configuration drift across AWS, Azure, GCP, and DigitalOcean
- Compares actual cloud state with Terraform state files
- Provides automated remediation capabilities
- Offers both CLI and server modes

## Core Business Logic & Architecture (v3.0 Target)

### Primary Purpose
DriftMgr focuses on Terraform/Terragrunt state file remediation, autodiscovery, and analysis through:
1. **Backend Discovery**: Auto-discover remote backends (S3, Azure Storage, GCS)
2. **State Analysis**: Parse and validate tfstate files from remote backends
3. **Drift Detection**: Compare actual cloud resources with desired state
4. **Automated Remediation**: Generate and execute remediation plans
5. **Terragrunt Support**: Handle complex Terragrunt configurations and dependencies

### Architectural Components

#### 1. Backend Discovery & Management
- **Auto-discovery**: Scan repositories for terraform backend configurations
- **Multi-backend support**: S3, Azure Storage, GCS, Terraform Cloud
- **Authentication**: Handle cross-account access, service principals, workload identity
- **Connection pooling**: Efficient connection management for remote backends
- **State locking**: DynamoDB, Azure blob leases, GCS generation numbers

#### 2. State File Operations
- **State retrieval**: Pull states from remote backends with caching
- **State parsing**: Handle multiple Terraform versions (0.11-1.x)
- **State validation**: Verify integrity and schema compliance
- **State manipulation**: Move, remove, import resources
- **State history**: Track changes and enable rollback

#### 3. Resource Analysis Engine
- **Dependency graphs**: Build and analyze resource relationships
- **Orphan detection**: Identify resources without providers
- **Impact analysis**: Calculate blast radius of changes
- **Health checks**: Validate resource configurations
- **Cost analysis**: Integrate with cloud pricing APIs

#### 4. Drift Detection Engine
- **Cloud discovery**: Query actual resources via provider APIs
- **State comparison**: Deep diff between actual and desired
- **Drift classification**: Missing, unmanaged, configuration drift
- **Severity scoring**: Prioritize based on impact and risk
- **Continuous monitoring**: Real-time drift detection

#### 5. Remediation System
- **Import generation**: Create terraform import commands
- **State refresh**: Update state to match reality
- **Automated fixes**: Apply remediation with approval workflows
- **Rollback capability**: Restore previous state versions
- **Dry-run mode**: Preview changes before applying

#### 6. Terragrunt Integration
- **HCL parsing**: Handle terragrunt.hcl configurations
- **Dependency resolution**: Process module dependencies
- **Remote state handling**: Manage remote_state blocks
- **Run-all support**: Coordinate multi-module operations

#### 7. Multi-Cloud Providers
- **AWS**: AssumeRole, cross-account, Organizations support
- **Azure**: Service principals, managed identity, management groups
- **GCP**: Service accounts, workload identity, asset inventory
- **DigitalOcean**: API tokens, project management

#### 8. Safety & Compliance
- **Backup system**: Automatic state backups before modifications
- **Audit logging**: Complete trail of all operations
- **Policy engine**: OPA integration for governance
- **Compliance reporting**: SOC2, HIPAA, PCI-DSS templates
- **Encryption**: At-rest and in-transit protection

## Key Architecture Decisions

### Language and Framework
- **Language**: Go 1.23
- **Primary Dependencies**: AWS SDK v2, Azure SDK, Google Cloud SDK
- **Architecture**: Unified Service Layer v2.0 with event-driven design
- **Testing**: Standard Go testing with benchmarks

### Current Project Structure (v3.0 Reorganized)
```
driftmgr/
├── cmd/                           # Entry points
│   ├── driftmgr/                 # Main CLI application
│   │   ├── main.go              # Single unified main
│   │   └── commands/            # All CLI commands
│   └── driftmgr-server/          # Server mode
│       └── main.go              # Server entry point
├── internal/                      # Core business logic (63 Go files, down from 447)
│   ├── providers/                # All cloud providers (consolidated)
│   │   ├── aws/                 # AWS provider with SDK
│   │   ├── azure/               # Azure provider
│   │   ├── gcp/                 # GCP provider
│   │   ├── digitalocean/        # DO provider
│   │   ├── interface.go         # CloudProvider interface
│   │   └── factory.go           # Provider factory
│   ├── discovery/                # All discovery logic (consolidated)
│   │   ├── backend/             # Backend discovery
│   │   ├── resource/            # Resource discovery
│   │   └── parallel/            # Parallel discovery
│   ├── state/                    # All state management (unified)
│   │   ├── parser/              # State file parsing
│   │   ├── manager/             # State CRUD operations
│   │   ├── cache/               # State caching
│   │   ├── backup/              # Backup management
│   │   └── validator/           # State validation
│   ├── drift/                    # Drift detection
│   │   ├── detector/            # Detection engine
│   │   ├── comparator/          # Comparison logic
│   │   └── types/               # Shared types
│   ├── remediation/              # Remediation system
│   │   ├── planner/             # Remediation planning
│   │   ├── executor/            # Execution engine
│   │   └── deletion/            # Resource deletion
│   ├── analysis/                 # Analysis modules
│   │   ├── cost/                # Cost analysis with impact
│   │   ├── graph/               # Dependency graphs
│   │   └── health/              # Health checks
│   ├── api/                      # API server
│   │   ├── server.go            # Main server
│   │   ├── handlers/            # Request handlers
│   │   └── websocket/           # WebSocket support
│   ├── terragrunt/               # Terragrunt support
│   │   ├── parser/              # HCL parsing
│   │   └── executor/            # Command execution
│   ├── safety/                   # Safety & compliance
│   │   ├── backup/              # Backup system
│   │   ├── audit/               # Audit logging
│   │   └── compliance/          # Compliance reporting
│   └── shared/                   # Shared utilities
│       ├── logger/              # Logging
│       ├── config/              # Configuration
│       └── metrics/             # Metrics
├── configs/                       # Configuration files
├── scripts/                       # Build and utility scripts
├── docs/                         # Documentation
│   └── architecture/            # Architecture documentation
└── web/                          # Web UI assets
```

**Key Statistics:**
- **Go Files**: 63 (86% reduction from 447)
- **Directories**: 43 (77% reduction from 186)
- **Duplicate Implementations**: 0 (eliminated 5+ duplicates per feature)
- **Main Files**: 1 (consolidated from 4)

## Development Guidelines

### Code Style
- Follow standard Go conventions
- Use meaningful variable names
- Keep functions small and focused
- NO EMOJI in code or comments unless explicitly requested
- Minimal comments - code should be self-documenting

### Testing Requirements
Before marking any feature as complete:
1. Run tests: `go test ./...`
2. Check linting: `golangci-lint run`
3. Verify build: `go build ./cmd/driftmgr`

### Common Commands
```bash
# Build main CLI
go build -o driftmgr.exe ./cmd/driftmgr

# Build server mode
go build -o driftmgr-server.exe ./cmd/driftmgr-server
# Alternative server location
go build -o server.exe ./cmd/server

# Build validation tool
go build -o validate.exe ./cmd/validate

# Test
go test ./... -v

# Run
./driftmgr.exe discover --provider aws --region us-east-1

# Docker build (using root Dockerfile)
docker build -t catherinevee/driftmgr:latest .

# Run in Docker
docker run --rm -v ~/.aws:/root/.aws catherinevee/driftmgr discover --provider aws
```

## Important Context

### Platform-Specific Considerations
- **Windows Development**: Project is primarily developed on Windows
- **Cross-platform**: Must work on Windows, Linux, and macOS
- **File Paths**: Always use filepath.Join() for cross-platform compatibility
- **Build Tags**: Use for platform-specific code (see terminal_windows.go example)

### Authentication Methods
- **AWS**: Environment variables, IAM roles, or credentials file
- **Azure**: Service principal or Azure CLI
- **GCP**: Service account JSON or application default credentials
- **DigitalOcean**: API token

### Recently Implemented Features (v3.0)
- **Unified Service Layer**: Complete service layer architecture ensuring consistency between CLI and web interfaces
- **Event Bus System**: Real-time event propagation for UI updates and audit trails
- **Job Queue System**: Asynchronous job processing with priority, retry logic, and progress tracking
- **State Management Commands**: Push/pull with remote backend support (S3, Azure, GCS)
- **Policy Engine**: OPA integration for governance
- **Compliance Reporting**: SOC2, HIPAA, PCI-DSS report generation
- **Continuous Monitoring**: Real-time monitoring with webhooks
- **Backup System**: Automated cleanup with quarantine system
- **Error Recovery**: Comprehensive error taxonomy with recovery strategies
- **Incremental Discovery**: Bloom filters for efficient change detection

### Current State (v3.0 Complete)
- **Clean Architecture**: Codebase reorganized following v3.0 target structure
- **No Mock Data**: All implementations are production-ready with real cloud provider integrations
- **Feature Complete**: All v3.0 commands implemented
- **63 Go Files**: Down from 447 (86% reduction)
- **Zero Duplicates**: Eliminated all duplicate implementations

## Working with This Project

### When Adding Features
1. Check `internal/` for existing implementation (only one place now!)
2. Follow existing patterns in the reorganized codebase
3. Add tests for new functionality
4. Update relevant documentation

### When Fixing Bugs
1. Reproduce the issue first
2. Add a test that fails with the bug
3. Fix the bug
4. Ensure test passes

### When Refactoring
1. Ensure all tests pass before starting
2. Make incremental changes
3. Run tests after each change
4. Update documentation if interfaces change

### Important File Locations
- **Main CLI**: `cmd/driftmgr/main.go`
- **AWS Provider**: `internal/providers/aws/provider.go`
- **Azure Provider**: `internal/providers/azure/provider.go`
- **GCP Provider**: `internal/providers/gcp/provider.go`
- **Drift Detection**: `internal/drift/detector/detector.go`
- **State Manager**: `internal/state/manager/manager.go`
- **Remediation**: `internal/remediation/planner/planner.go`

## Response Guidelines for AI Assistants

### Be Concise
- Give direct answers
- Avoid unnecessary explanations
- Show code rather than describing it

### Be Accurate
- Test commands before suggesting them
- Verify file paths exist
- Check for existing implementations

### Be Helpful
- Suggest the simplest solution first
- Provide working examples
- Include error handling

### Avoid
- Creating unnecessary files
- Adding emoji unless requested
- Implementing features that already exist
- Making assumptions about user intent

## Common Issues and Solutions

### Docker Build Issues
- **Problem**: Windows-specific syscalls in Linux builds
- **Solution**: Use build tags to separate platform code

### Dependency Conflicts
- **Problem**: Go module version conflicts
- **Solution**: Remove toolchain directive, use Go 1.23

### Credential Detection
- **Problem**: Can't find cloud credentials
- **Solution**: Check environment variables and credential files

## GitHub Actions Workflows

### Available Commands for Workflows
- **Main CLI**: `./cmd/driftmgr` - Primary drift management tool
- **Server Mode**: `./cmd/driftmgr-server` or `./cmd/server` - API server
- **Validation Tool**: `./cmd/validate` - State and configuration validation
- **Quality Tools**: Located in `quality/cmd/` for code quality analysis
- **UAT Tools**: Located in `tests/uat/` for user acceptance testing

### Dockerfile Location
The project uses the root `./Dockerfile` for all Docker builds. The alternative location at `./deployments/docker/Dockerfile` is deprecated.

## CI/CD Secrets Required

For GitHub Actions workflows:
- `DOCKER_HUB_USERNAME`: Docker Hub username
- `DOCKER_HUB_TOKEN`: Docker Hub access token
- `AWS_ACCESS_KEY_ID`: (Optional) AWS credentials
- `AZURE_CLIENT_ID`: (Optional) Azure service principal
- `GCP_SERVICE_ACCOUNT_JSON`: (Optional) GCP service account

## Testing Checklist

When making changes, ensure:
- [ ] Code compiles: `go build ./cmd/driftmgr`
- [ ] Tests pass: `go test ./...`
- [ ] Docker builds: `docker build -t driftmgr:test .`
- [ ] No linting errors: `golangci-lint run`
- [ ] Documentation updated if needed

## Questions to Ask

When unclear about requirements:
1. "Should this work on all platforms or just Windows?"
2. "Is this replacing existing functionality or adding new?"
3. "Should this be added to the CLI commands?"
4. "Does this need tests?"
5. "Should errors be logged or returned?"

## Project Maintainer Notes

- Primary repository: `github.com/catherinevee/driftmgr`
- Docker images: `catherinevee/driftmgr` on Docker Hub
- Issues: Report at GitHub Issues
- Main branch: `main` (default for PRs)

---

*This file helps AI assistants understand the project context and provide better assistance. Last major update: v3.0 reorganization completed.*