# Claude AI Assistant Guide for DriftMgr

This document provides context and guidelines for AI assistants (particularly Claude) when working with the DriftMgr codebase.

## Project Overview

DriftMgr is a multi-cloud infrastructure drift detection and remediation tool that:
- Detects configuration drift across AWS, Azure, GCP, and DigitalOcean
- Compares actual cloud state with Terraform state files
- Provides automated remediation capabilities
- Offers both CLI and server modes

## Key Architecture Decisions

### Language and Framework
- **Language**: Go 1.23
- **Primary Dependencies**: AWS SDK v2, Azure SDK, Google Cloud SDK
- **Architecture**: Modular with provider interfaces
- **Testing**: Standard Go testing with benchmarks

### Project Structure
```
driftmgr/
├── cmd/                    # Entry points
│   ├── driftmgr/          # Main CLI application
│   └── driftmgr-server/   # Server mode
├── internal/              # Core business logic
│   ├── providers/         # Cloud provider implementations
│   ├── core/              # Core functionality
│   ├── discovery/         # Resource discovery
│   ├── drift/             # Drift detection
│   └── remediation/       # Auto-remediation
├── configs/               # Configuration files
├── scripts/               # Build and utility scripts
└── docs/                  # Documentation
```

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
# Build
go build -o driftmgr.exe ./cmd/driftmgr

# Test
go test ./... -v

# Run
./driftmgr.exe discover --provider aws --region us-east-1

# Docker build
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

### Recently Implemented Features
- **Enterprise Audit Logging**: Complete audit trail with compliance modes (SOC2, HIPAA, PCI-DSS)
- **RBAC System**: Role-based access control with predefined roles (Admin, Operator, Viewer, Approver)
- **HashiCorp Vault Integration**: Secure secrets management via vault client
- **Circuit Breaker Pattern**: Already implemented in internal/resilience
- **Rate Limiting**: Already implemented in internal/resilience

### Current Limitations
- State management commands (state push/pull) are NOT implemented (by design - focusing on drift detection)

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

## CI/CD Secrets Required

For GitHub Actions workflows:
- `DOCKER_HUB_USERNAME`: Docker Hub username
- `DOCKER_HUB_TOKEN`: Docker Hub access token
- `AWS_ACCESS_KEY_ID`: (Optional) AWS credentials
- `AZURE_CLIENT_ID`: (Optional) Azure service principal
- `GCP_SERVICE_ACCOUNT_JSON`: (Optional) GCP service account

## Working with This Project

### When Adding Features
1. Check if similar functionality exists
2. Follow existing patterns in the codebase
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

## Useful Resources

- **README.md**: Main project documentation
- **docs/SECRETS_SETUP.md**: GitHub secrets configuration
- **configs/config.yaml**: Default configuration
- **.github/workflows/**: CI/CD pipeline definitions

## Project Maintainer Notes

- Primary repository: `github.com/catherinevee/driftmgr`
- Docker images: `catherinevee/driftmgr` on Docker Hub
- Issues: Report at GitHub Issues
- Main branch: `main` (default for PRs)

## Recent Changes

- Fixed Docker build issues with platform-specific code
- Added comprehensive GitHub Actions workflows
- Improved documentation accuracy
- Added workflow examples to README

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

---

*This file helps AI assistants understand the project context and provide better assistance. Keep it updated as the project evolves.*