# Contributing to DriftMgr

Thank you for your interest in contributing to DriftMgr! This document provides comprehensive guidelines for contributing to the project, from setting up your development environment to submitting high-quality pull requests.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Project Structure](#project-structure)
- [Development Workflow](#development-workflow)
- [Code Style and Conventions](#code-style-and-conventions)
- [Testing Guidelines](#testing-guidelines)
- [Documentation Guidelines](#documentation-guidelines)
- [Submitting Changes](#submitting-changes)
- [Review Process](#review-process)
- [Community and Communication](#community-and-communication)

## Code of Conduct

By participating in this project, you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md). Please read it before contributing.

### Our Pledge

We pledge to make participation in our project a harassment-free experience for everyone, regardless of age, body size, disability, ethnicity, gender identity and expression, level of experience, nationality, personal appearance, race, religion, or sexual identity and orientation.

## Getting Started

### Prerequisites

Before you begin, ensure you have the following installed:

- **Go 1.21 or higher**
- **Node.js 18 or higher** (for web interface development)
- **Docker and Docker Compose** (for integration testing)
- **Git**
- **Make** (for build automation)

### Development Tools (Recommended)

- **VS Code** with Go extension
- **Postman** or **curl** for API testing
- **jq** for JSON processing
- **govulncheck** for security scanning
- **golangci-lint** for code linting

### First Contribution Workflow

1. **Fork the repository**
   ```bash
   git clone https://github.com/your-username/driftmgr.git
   cd driftmgr
   ```

2. **Set up development environment**
   ```bash
   make setup-dev
   ```

3. **Find an issue to work on**
   - Check issues labeled `good first issue` or `help wanted`
   - Comment on the issue to indicate you're working on it

4. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

5. **Make your changes**
   - Follow the coding standards
   - Add tests for new functionality
   - Update documentation as needed

6. **Test your changes**
   ```bash
   make test
   make lint
   ```

7. **Submit a pull request**
   - Include a clear description of changes
   - Reference the issue number

## Development Setup

### Quick Setup

```bash
# Clone the repository
git clone https://github.com/catherinevee/driftmgr.git
cd driftmgr

# Run setup script
make setup-dev

# Build the project
make build

# Run tests
make test
```

### Manual Setup

```bash
# Install Go dependencies
go mod download

# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
go install github.com/swaggo/swag/cmd/swag@latest

# Install Node.js dependencies (for web interface)
cd web
npm install
cd ..

# Install pre-commit hooks
pip install pre-commit
pre-commit install

# Set up test environment
cp configs/config.yaml.example configs/config.yaml
# Edit configs/config.yaml with your test credentials
```

### Environment Configuration

Create a `.env` file for development:

```bash
# .env
# Development settings
DRIFTMGR_ENV=development
DRIFTMGR_LOG_LEVEL=debug
DRIFTMGR_PORT=8080

# Test cloud credentials (optional)
AWS_ACCESS_KEY_ID=your-test-key
AWS_SECRET_ACCESS_KEY=your-test-secret
AZURE_CLIENT_ID=your-client-id
AZURE_CLIENT_SECRET=your-client-secret
AZURE_TENANT_ID=your-tenant-id

# Database (for integration tests)
POSTGRES_URL=postgres://user:pass@localhost/driftmgr_test
REDIS_URL=redis://localhost:6379/0
```

### Docker Development Environment

```bash
# Start development services
docker-compose -f docker-compose.dev.yml up -d

# This starts:
# - PostgreSQL (for state storage)
# - Redis (for caching)
# - Prometheus (for metrics)
# - Grafana (for visualization)
```

## Project Structure

```
driftmgr/
├── cmd/                          # Application entry points
│   ├── driftmgr/                # Main CLI application
│   ├── driftmgr-tui/           # Terminal UI application
│   ├── server/                  # Web server
│   └── validate/               # Validation utilities
├── configs/                     # Configuration files
│   ├── config.yaml             # Main configuration
│   ├── auto-remediation.yaml   # Auto-remediation rules
│   └── providers/              # Provider-specific configs
├── internal/                    # Private application code
│   ├── analysis/               # Drift analysis logic
│   ├── api/                    # REST API handlers
│   ├── config/                 # Configuration management
│   ├── discovery/              # Resource discovery
│   ├── drift/                  # Drift detection
│   ├── models/                 # Data models
│   ├── remediation/            # Auto-remediation engine
│   ├── security/               # Security components
│   ├── state/                  # State management
│   └── visualization/          # Data visualization
├── web/                         # Frontend React application
│   ├── src/                    # React source code
│   ├── public/                 # Static assets
│   └── package.json            # NPM dependencies
├── docs/                        # Documentation
│   ├── api/                    # API documentation
│   ├── user-guide/             # User guides
│   └── development/            # Development docs
├── scripts/                     # Build and deployment scripts
├── tests/                       # Test files
│   ├── unit/                   # Unit tests
│   ├── integration/            # Integration tests
│   └── e2e/                    # End-to-end tests
├── Makefile                     # Build automation
├── go.mod                       # Go module definition
├── docker-compose.yml           # Production Docker setup
└── docker-compose.dev.yml       # Development Docker setup
```

### Key Packages

- **`internal/discovery/`**: Multi-cloud resource discovery
- **`internal/drift/`**: Drift detection algorithms
- **`internal/remediation/`**: Auto-remediation engine
- **`internal/api/`**: REST API implementation
- **`internal/models/`**: Core data structures
- **`internal/security/`**: Authentication and authorization

## Development Workflow

### Branch Naming

Use descriptive branch names with prefixes:

- `feature/` - New features
- `bugfix/` - Bug fixes
- `hotfix/` - Critical fixes
- `docs/` - Documentation updates
- `refactor/` - Code refactoring
- `test/` - Test improvements

Examples:
- `feature/azure-auto-remediation`
- `bugfix/aws-credential-detection`
- `docs/api-authentication-guide`

### Commit Message Format

Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

Types:
- `feat`: New features
- `fix`: Bug fixes
- `docs`: Documentation changes
- `style`: Code style changes
- `refactor`: Code refactoring
- `test`: Adding or fixing tests
- `chore`: Maintenance tasks

Examples:
```
feat(discovery): add DigitalOcean Kubernetes support

Add support for discovering DigitalOcean Kubernetes clusters
including node pools and associated resources.

Closes #123
```

```
fix(auth): resolve AWS credential chain issue

Fix credential precedence order to properly handle
IAM roles when multiple credential sources are available.

Fixes #456
```

### Development Commands

```bash
# Build the application
make build

# Run all tests
make test

# Run specific test suites
make test-unit
make test-integration
make test-e2e

# Lint code
make lint

# Format code
make fmt

# Generate API documentation
make docs-api

# Start development server
make dev-server

# Run security scan
make security-scan

# Clean build artifacts
make clean
```

## Code Style and Conventions

### Go Code Style

We follow standard Go conventions with additional project-specific rules:

#### General Guidelines

1. **Use `gofmt`** for formatting
2. **Follow `golint`** recommendations
3. **Use meaningful variable names**
4. **Write self-documenting code**
5. **Handle all errors explicitly**

#### Package Structure

```go
// Package declaration with description
package discovery

import (
    // Standard library imports first
    "context"
    "fmt"
    "time"
    
    // Third-party imports
    "github.com/aws/aws-sdk-go-v2/service/ec2"
    
    // Local imports last
    "github.com/catherinevee/driftmgr/internal/models"
)
```

#### Function Documentation

```go
// DiscoverResources discovers cloud resources for the specified provider.
// It returns a slice of discovered resources and any error encountered.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - provider: Cloud provider name (aws, azure, gcp, digitalocean)
//   - config: Provider-specific configuration
//
// Returns:
//   - []models.Resource: Discovered resources
//   - error: Any error that occurred during discovery
func DiscoverResources(ctx context.Context, provider string, config ProviderConfig) ([]models.Resource, error) {
    // Implementation
}
```

#### Error Handling

```go
// Good: Wrap errors with context
func processResource(resource *models.Resource) error {
    if err := validateResource(resource); err != nil {
        return fmt.Errorf("failed to validate resource %s: %w", resource.ID, err)
    }
    
    if err := saveResource(resource); err != nil {
        return fmt.Errorf("failed to save resource %s: %w", resource.ID, err)
    }
    
    return nil
}

// Good: Handle errors at appropriate levels
func main() {
    if err := run(); err != nil {
        log.Fatalf("Application failed: %v", err)
    }
}
```

#### Interface Design

```go
// Keep interfaces small and focused
type ResourceDiscoverer interface {
    DiscoverResources(ctx context.Context) ([]models.Resource, error)
}

type ResourceValidator interface {
    ValidateResource(resource *models.Resource) error
}

// Compose interfaces when needed
type ResourceProcessor interface {
    ResourceDiscoverer
    ResourceValidator
}
```

#### Testing Conventions

```go
func TestDiscoverAWSResources(t *testing.T) {
    tests := []struct {
        name           string
        config         AWSConfig
        expectedCount  int
        expectedError  string
    }{
        {
            name: "successful discovery",
            config: AWSConfig{
                Region: "us-east-1",
                AccessKey: "test-key",
            },
            expectedCount: 5,
        },
        {
            name: "invalid credentials",
            config: AWSConfig{
                Region: "us-east-1",
                AccessKey: "invalid",
            },
            expectedError: "authentication failed",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            discoverer := NewAWSDiscoverer(tt.config)
            resources, err := discoverer.DiscoverResources(context.Background())
            
            if tt.expectedError != "" {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.expectedError)
                return
            }
            
            assert.NoError(t, err)
            assert.Len(t, resources, tt.expectedCount)
        })
    }
}
```

### JavaScript/TypeScript Style (Web Interface)

#### General Guidelines

1. **Use TypeScript** for type safety
2. **Follow ESLint rules** configured in the project
3. **Use Prettier** for code formatting
4. **Prefer functional components** with hooks
5. **Use meaningful component and variable names**

#### Component Structure

```typescript
// components/DriftDetection/DriftTable.tsx
import React, { useState, useEffect } from 'react';
import { DriftResult, Provider } from '@/types/drift';
import { useDriftDetection } from '@/hooks/useDriftDetection';

interface DriftTableProps {
  provider: Provider;
  onSelectDrift: (drift: DriftResult) => void;
}

export const DriftTable: React.FC<DriftTableProps> = ({
  provider,
  onSelectDrift
}) => {
  const { drifts, loading, error } = useDriftDetection(provider);
  
  if (loading) return <LoadingSpinner />;
  if (error) return <ErrorMessage error={error} />;
  
  return (
    <div className="drift-table">
      {/* Component implementation */}
    </div>
  );
};
```

#### API Client Structure

```typescript
// api/driftClient.ts
export class DriftClient {
  private baseURL: string;
  
  constructor(baseURL: string) {
    this.baseURL = baseURL;
  }
  
  async detectDrift(provider: Provider): Promise<DriftResult[]> {
    const response = await fetch(`${this.baseURL}/api/v1/drift/detect`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ provider }),
    });
    
    if (!response.ok) {
      throw new Error(`Drift detection failed: ${response.statusText}`);
    }
    
    return response.json();
  }
}
```

## Testing Guidelines

### Test Organization

```
tests/
├── unit/                    # Unit tests (fast, isolated)
│   ├── discovery_test.go
│   ├── drift_test.go
│   └── models_test.go
├── integration/             # Integration tests (slower, real services)
│   ├── aws_integration_test.go
│   ├── api_integration_test.go
│   └── database_test.go
└── e2e/                     # End-to-end tests (slowest, full scenarios)
    ├── drift_detection_e2e_test.go
    └── auto_remediation_e2e_test.go
```

### Unit Tests

- **Fast execution** (< 100ms per test)
- **No external dependencies**
- **High code coverage** (aim for 80%+)
- **Test edge cases and error conditions**

```go
func TestValidateAWSCredentials(t *testing.T) {
    tests := []struct {
        name     string
        creds    AWSCredentials
        expected bool
    }{
        {
            name: "valid credentials",
            creds: AWSCredentials{
                AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
                SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
            },
            expected: true,
        },
        {
            name: "empty access key",
            creds: AWSCredentials{
                AccessKeyID:     "",
                SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
            },
            expected: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := ValidateAWSCredentials(tt.creds)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Integration Tests

- **Test component interactions**
- **Use test databases/services**
- **Clean up resources after tests**

```go
func TestAWSDiscoveryIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    // Setup test environment
    config := getTestAWSConfig(t)
    discoverer := NewAWSDiscoverer(config)
    
    // Cleanup after test
    t.Cleanup(func() {
        cleanupTestResources(t, config)
    })
    
    // Run test
    ctx := context.Background()
    resources, err := discoverer.DiscoverResources(ctx)
    
    require.NoError(t, err)
    assert.NotEmpty(t, resources)
}
```

### End-to-End Tests

- **Test complete user workflows**
- **Use production-like environment**
- **Focus on critical paths**

```go
func TestDriftDetectionWorkflow(t *testing.T) {
    // Start test server
    server := startTestServer(t)
    defer server.Close()
    
    client := NewAPIClient(server.URL)
    
    // Test complete workflow
    t.Run("detect drift", func(t *testing.T) {
        result, err := client.DetectDrift("aws", "us-east-1")
        require.NoError(t, err)
        assert.NotEmpty(t, result.Drifts)
    })
    
    t.Run("remediate drift", func(t *testing.T) {
        // Implementation
    })
}
```

### Test Data and Fixtures

```go
// tests/fixtures/aws.go
func GetTestAWSConfig() AWSConfig {
    return AWSConfig{
        Region:          getEnvOrDefault("TEST_AWS_REGION", "us-east-1"),
        AccessKeyID:     getEnvOrDefault("TEST_AWS_ACCESS_KEY_ID", ""),
        SecretAccessKey: getEnvOrDefault("TEST_AWS_SECRET_ACCESS_KEY", ""),
    }
}

func CreateTestEC2Instance(t *testing.T) *models.Resource {
    return &models.Resource{
        ID:       "i-1234567890abcdef0",
        Type:     "ec2_instance",
        Provider: "aws",
        Region:   "us-east-1",
        State:    models.ResourceStateRunning,
        Tags: map[string]string{
            "Environment": "test",
            "CreatedBy":   "driftmgr-test",
        },
    }
}
```

### Running Tests

```bash
# Run all tests
make test

# Run only unit tests
make test-unit

# Run tests with coverage
make test-coverage

# Run tests with race detection
make test-race

# Run specific test
go test -v ./internal/discovery -run TestDiscoverAWSResources

# Run integration tests (requires credentials)
make test-integration

# Run e2e tests (requires full environment)
make test-e2e
```

## Documentation Guidelines

### Code Documentation

1. **Public APIs must be documented**
2. **Include examples for complex functions**
3. **Document behavior, not implementation**
4. **Keep documentation up-to-date with code changes**

### API Documentation

Use OpenAPI/Swagger annotations:

```go
// @Summary      Detect infrastructure drift
// @Description  Scan cloud infrastructure for configuration drift
// @Tags         drift
// @Accept       json
// @Produce      json
// @Param        request body DriftDetectionRequest true "Detection parameters"
// @Success      200 {object} DriftDetectionResponse
// @Failure      400 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /api/v1/drift/detect [post]
func (h *DriftHandler) DetectDrift(c *gin.Context) {
    // Implementation
}
```

### User Documentation

1. **Write for the target audience**
2. **Include working examples**
3. **Provide troubleshooting guides**
4. **Keep documentation current**

Example structure for new features:

```markdown
# Feature Name

## Overview
Brief description of what the feature does.

## Getting Started
Quick example to get users started.

## Configuration
Detailed configuration options.

## Examples
Real-world usage examples.

## Troubleshooting
Common issues and solutions.

## API Reference
Detailed API documentation.
```

## Submitting Changes

### Pull Request Process

1. **Create a feature branch** from `main`
2. **Make your changes** following the guidelines
3. **Add tests** for new functionality
4. **Update documentation** as needed
5. **Ensure all tests pass**
6. **Submit pull request** with clear description

### Pull Request Template

When creating a pull request, use this template:

```markdown
## Description
Brief description of the changes.

## Type of Change
- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update

## Related Issues
Closes #(issue number)

## Testing
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual testing completed

## Checklist
- [ ] Code follows the project's style guidelines
- [ ] Self-review of code completed
- [ ] Code is commented, particularly in hard-to-understand areas
- [ ] Corresponding changes to documentation made
- [ ] No new warnings introduced
- [ ] Tests added for new functionality
```

### Commit Requirements

Before submitting:

```bash
# Ensure code is formatted
make fmt

# Run linter
make lint

# Run all tests
make test

# Check for security issues
make security-scan

# Verify build
make build
```

## Review Process

### Code Review Guidelines

#### For Authors

1. **Keep pull requests focused** and reasonably sized
2. **Provide clear descriptions** of changes
3. **Respond promptly** to review feedback
4. **Test thoroughly** before requesting review
5. **Update documentation** alongside code changes

#### For Reviewers

1. **Be constructive** and helpful in feedback
2. **Focus on code quality**, not personal preferences
3. **Suggest improvements** rather than just pointing out problems
4. **Approve promptly** when changes look good
5. **Test changes locally** for complex features

### Review Criteria

Code reviews should check for:

- **Correctness**: Does the code do what it's supposed to do?
- **Performance**: Are there any performance implications?
- **Security**: Are there any security vulnerabilities?
- **Maintainability**: Is the code easy to understand and modify?
- **Test Coverage**: Are there adequate tests?
- **Documentation**: Is new functionality documented?

### Merge Requirements

Pull requests must meet these criteria before merging:

- [ ] **All CI checks pass**
- [ ] **At least one approval** from a maintainer
- [ ] **No merge conflicts**
- [ ] **Up-to-date with main branch**
- [ ] **Tests added/updated** for changes
- [ ] **Documentation updated** if needed

## Community and Communication

### Getting Help

- **GitHub Discussions**: For questions and general discussion
- **GitHub Issues**: For bug reports and feature requests
- **Discord**: Real-time chat with the community
- **Stack Overflow**: Tag questions with `driftmgr`

### Communication Channels

- **GitHub**: Primary platform for development
- **Discord**: https://discord.gg/driftmgr
- **Twitter**: @driftmgr for announcements
- **Email**: dev@driftmgr.io for private communication

### Community Guidelines

1. **Be respectful** and inclusive
2. **Help others** when you can
3. **Search existing issues** before creating new ones
4. **Provide detailed information** when reporting bugs
5. **Follow up** on your own issues and PRs

### Recognition

We recognize contributors in several ways:

- **Contributors file**: Listed in CONTRIBUTORS.md
- **Release notes**: Credited for significant contributions
- **Hall of fame**: Featured on project website
- **Swag**: DriftMgr swag for regular contributors

## Development Resources

### Useful Links

- **Project Website**: https://driftmgr.io
- **Documentation**: https://docs.driftmgr.io
- **API Reference**: https://api.driftmgr.io
- **Examples**: https://github.com/catherinevee/driftmgr/tree/main/examples

### Learning Resources

- **Go Documentation**: https://golang.org/doc/
- **Cloud Provider APIs**:
  - AWS SDK: https://aws.github.io/aws-sdk-go-v2/
  - Azure SDK: https://docs.microsoft.com/en-us/azure/developer/go/
  - GCP SDK: https://cloud.google.com/go/docs
- **Infrastructure as Code**:
  - Terraform: https://www.terraform.io/docs
  - Pulumi: https://www.pulumi.com/docs

### Development Tools

Recommended extensions for VS Code:

```json
{
  "recommendations": [
    "golang.go",
    "ms-vscode.vscode-typescript-next",
    "bradlc.vscode-tailwindcss",
    "esbenp.prettier-vscode",
    "ms-vscode.vscode-json"
  ]
}
```

## License

By contributing to DriftMgr, you agree that your contributions will be licensed under the MIT License.

---

**Thank you for contributing to DriftMgr!** 

Your contributions help make infrastructure management better for everyone. If you have questions about contributing, feel free to reach out to us through any of our communication channels.

**Last Updated**: 2024-12-19  
**Version**: 2.0  
**Maintainers**: DriftMgr Development Team