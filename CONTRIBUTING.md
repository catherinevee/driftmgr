# Contributing to DriftMgr

Thank you for your interest in contributing to DriftMgr! We welcome contributions from the community and are grateful for any help you can provide.

## Table of Contents
- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [How to Contribute](#how-to-contribute)
- [Development Process](#development-process)
- [Pull Request Process](#pull-request-process)
- [Coding Standards](#coding-standards)
- [Testing Requirements](#testing-requirements)
- [Documentation](#documentation)
- [Security](#security)
- [Recognition](#recognition)

## Code of Conduct

### Our Pledge
We are committed to providing a friendly, safe, and welcoming environment for all contributors, regardless of experience level, gender identity, sexual orientation, disability, appearance, race, ethnicity, age, religion, or nationality.

### Expected Behavior
- Be respectful and inclusive
- Welcome newcomers and help them get started
- Focus on constructive criticism
- Accept feedback gracefully
- Respect differing viewpoints

### Unacceptable Behavior
- Harassment, discrimination, or offensive comments
- Personal attacks or trolling
- Publishing others' private information
- Any conduct that could reasonably be considered inappropriate

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR-USERNAME/driftmgr.git
   cd driftmgr
   ```
3. **Add upstream remote**:
   ```bash
   git remote add upstream https://github.com/catherinevee/driftmgr.git
   ```
4. **Set up development environment**:
   ```bash
   make setup
   ```
5. **Read the documentation**:
   - [DEVELOPMENT.md](docs/DEVELOPMENT.md) - Development setup guide
   - [API_REFERENCE.md](docs/API_REFERENCE.md) - API documentation
   - [README.md](README.md) - Project overview

## How to Contribute

### Reporting Bugs

Before creating a bug report, please check existing issues to avoid duplicates.

**To report a bug:**
1. Go to [Issues](https://github.com/catherinevee/driftmgr/issues)
2. Click "New Issue"
3. Select "Bug Report" template
4. Fill in all required information:
   - Clear, descriptive title
   - Steps to reproduce
   - Expected vs actual behavior
   - Environment details (OS, Go version, etc.)
   - Logs or error messages
   - Screenshots if applicable

### Suggesting Features

We love new ideas! Before suggesting a feature, search existing issues to see if it's been discussed.

**To suggest a feature:**
1. Go to [Issues](https://github.com/catherinevee/driftmgr/issues)
2. Click "New Issue"
3. Select "Feature Request" template
4. Provide:
   - Clear description of the feature
   - Use cases and benefits
   - Potential implementation approach
   - Any relevant examples or mockups

### Contributing Code

#### First-Time Contributors
Look for issues labeled:
- `good-first-issue` - Simple tasks perfect for beginners
- `help-wanted` - Issues where we need community help
- `documentation` - Documentation improvements

#### Setting Up Your Branch
```bash
# Sync with upstream
git fetch upstream
git checkout main
git merge upstream/main

# Create feature branch
git checkout -b feature/your-feature-name
# Or for bugs
git checkout -b fix/issue-description
```

#### Making Changes
1. Write clean, well-documented code
2. Follow our [coding standards](#coding-standards)
3. Add/update tests as needed
4. Update documentation if required
5. Keep commits focused and atomic

#### Commit Messages
Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Test additions or corrections
- `chore`: Maintenance tasks
- `perf`: Performance improvements

**Examples:**
```bash
git commit -m "feat(providers): add DigitalOcean support"
git commit -m "fix(drift): resolve memory leak in detector"
git commit -m "docs(api): update REST endpoints documentation"
```

## Development Process

### 1. Choose or Create an Issue
- Check [open issues](https://github.com/catherinevee/driftmgr/issues)
- Comment on the issue to claim it
- Wait for maintainer approval before starting major work

### 2. Design Discussion (for major changes)
For significant features:
1. Create a design proposal issue
2. Include:
   - Problem statement
   - Proposed solution
   - Alternative approaches considered
   - API changes (if any)
   - Migration strategy (if needed)

### 3. Implementation
- Write code following our standards
- Add comprehensive tests
- Update documentation
- Ensure backward compatibility

### 4. Testing
Run all tests before submitting:
```bash
# Format and lint
make fmt
make lint

# Run tests
make test
make test-integration
make test-coverage

# Security checks
make security
```

## Pull Request Process

### Before Submitting

**Checklist:**
- [ ] Code follows project style guidelines
- [ ] Tests pass locally (`make test`)
- [ ] Documentation updated (if needed)
- [ ] Commit messages follow convention
- [ ] Branch is up-to-date with `main`
- [ ] No unrelated changes included

### Submitting a PR

1. **Push your branch**:
   ```bash
   git push origin feature/your-feature-name
   ```

2. **Create Pull Request**:
   - Go to your fork on GitHub
   - Click "New Pull Request"
   - Select your branch
   - Fill in the PR template

3. **PR Description Should Include**:
   - Related issue number (Fixes #123)
   - Summary of changes
   - Testing performed
   - Screenshots (for UI changes)
   - Breaking changes (if any)

### PR Review Process

1. **Automated Checks**: CI/CD will run tests and checks
2. **Code Review**: Maintainers will review your code
3. **Address Feedback**: Make requested changes
4. **Approval**: Once approved, PR will be merged

### After Merge

- Delete your feature branch
- Sync your fork with upstream
- Celebrate your contribution! 🎉

## Coding Standards

### Go Code Style

#### General Rules
- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` and `goimports`
- Maximum line length: 120 characters
- Write self-documenting code

#### File Organization
```go
package mypackage

import (
    // Standard library
    "context"
    "fmt"
    
    // Third-party
    "github.com/aws/aws-sdk-go-v2/service/ec2"
    
    // Internal packages
    "github.com/catherinevee/driftmgr/internal/logger"
)
```

#### Naming Conventions
```go
// Exported types and functions: CamelCase
type DriftDetector struct {}
func NewDriftDetector() *DriftDetector {}

// Unexported: camelCase
type internalType struct {}
func helperFunction() {}

// Constants: CamelCase or ALL_CAPS
const MaxRetries = 3
const DEFAULT_TIMEOUT = 30 * time.Second

// Interfaces: end with -er
type Scanner interface {}
type Provider interface {}
```

#### Error Handling
```go
// Always check errors
result, err := someFunction()
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}

// Custom error types
type NotFoundError struct {
    Resource string
}

func (e *NotFoundError) Error() string {
    return fmt.Sprintf("resource not found: %s", e.Resource)
}
```

#### Comments and Documentation
```go
// Package driftmgr implements drift detection for cloud infrastructure.
package driftmgr

// DriftDetector analyzes infrastructure drift by comparing
// actual cloud resources with desired state definitions.
type DriftDetector struct {
    // provider is the cloud provider interface
    provider Provider
    
    // cache stores recent discovery results
    cache Cache
}

// DetectDrift identifies configuration drift in cloud resources.
// It returns a list of drifted resources or an error if detection fails.
//
// Example:
//
//	detector := NewDriftDetector(provider)
//	drifts, err := detector.DetectDrift(ctx, state)
//	if err != nil {
//	    return err
//	}
func (d *DriftDetector) DetectDrift(ctx context.Context, state State) ([]*Drift, error) {
    // Implementation
}
```

## Testing Requirements

### Test Coverage
- Minimum coverage: 70% overall
- Critical paths: 80%+ coverage
- New code: Must include tests

### Test Types

#### Unit Tests
```go
func TestDriftDetector_DetectDrift(t *testing.T) {
    tests := []struct {
        name    string
        state   State
        want    []*Drift
        wantErr bool
    }{
        {
            name:  "no drift",
            state: State{...},
            want:  nil,
            wantErr: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            detector := NewDriftDetector(mockProvider)
            got, err := detector.DetectDrift(context.Background(), tt.state)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("DetectDrift() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("DetectDrift() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

#### Integration Tests
```go
// +build integration

func TestAWSProvider_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    // Test with real or mocked services
}
```

#### Benchmark Tests
```go
func BenchmarkDriftDetection(b *testing.B) {
    detector := NewDriftDetector(provider)
    state := generateLargeState()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = detector.DetectDrift(context.Background(), state)
    }
}
```

## Documentation

### Code Documentation
- All exported types and functions must have godoc comments
- Include examples for complex functions
- Document any non-obvious behavior

### User Documentation
When adding features:
1. Update relevant markdown files in `docs/`
2. Add examples to `examples/`
3. Update README if needed
4. Add to CHANGELOG.md

### API Documentation
For API changes:
1. Update OpenAPI/Swagger specs
2. Update API_REFERENCE.md
3. Include request/response examples

## Security

### Reporting Security Issues
**DO NOT** file public issues for security vulnerabilities.

Email: security@driftmgr.io (or create private security advisory on GitHub)

Include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

### Security Requirements
- Never commit secrets or credentials
- Use secure coding practices
- Validate all inputs
- Handle errors gracefully
- Follow principle of least privilege

## Recognition

### Contributors
All contributors will be recognized in:
- [CONTRIBUTORS.md](CONTRIBUTORS.md) file
- Release notes
- Project documentation

### Types of Contributions
We value all contributions:
- 💻 Code contributions
- 📖 Documentation improvements
- 🐛 Bug reports and fixes
- 💡 Feature suggestions
- 🎨 Design improvements
- 📢 Community advocacy
- ❓ Helping others in discussions

## Getting Help

### Resources
- [Documentation](docs/)
- [GitHub Discussions](https://github.com/catherinevee/driftmgr/discussions)
- [Issue Tracker](https://github.com/catherinevee/driftmgr/issues)

### Communication Channels
- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: General questions and discussions
- **Pull Requests**: Code reviews and contributions

## License

By contributing to DriftMgr, you agree that your contributions will be licensed under the same license as the project (see [LICENSE](LICENSE) file).

---

Thank you for contributing to DriftMgr! Your efforts help make infrastructure management better for everyone. 🚀