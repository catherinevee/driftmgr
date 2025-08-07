# CI/CD Pipeline Documentation

This document describes the comprehensive CI/CD pipeline for the Terraform Import Helper project.

## 🏗️ Pipeline Overview

Our CI/CD pipeline consists of several automated workflows:

### 1. **Continuous Integration (CI)**
- **Trigger**: Every push to `main` and `develop` branches, and all pull requests
- **File**: `.github/workflows/ci.yml`
- **Duration**: ~5-10 minutes

**Jobs:**
- **Test**: Runs comprehensive test suite across Go 1.23 and 1.24
- **Lint**: Code quality checks with golangci-lint
- **Security**: Security scanning with Gosec
- **Integration**: Integration tests (main branch only)

### 2. **Release Pipeline**
- **Trigger**: When a version tag is pushed (e.g., `v1.0.0`)
- **File**: `.github/workflows/release.yml`
- **Duration**: ~15-20 minutes

**Jobs:**
- **Release**: Builds multi-platform binaries and creates GitHub release
- **Docker**: Builds and pushes Docker images to registry

### 3. **Dependency Management**
- **Trigger**: Weekly schedule (Sundays 2 AM UTC) or manual
- **File**: `.github/workflows/dependencies.yml`
- **Duration**: ~5 minutes

**Jobs:**
- **Update**: Automatically updates Go dependencies
- **Security Audit**: Scans for security vulnerabilities

## 🔧 Workflow Details

### CI Workflow Jobs

#### Test Job
```yaml
strategy:
  matrix:
    go-version: ['1.23', '1.24']
```

**Steps:**
1. **Checkout**: Get latest code
2. **Setup Go**: Install specified Go version
3. **Cache**: Cache Go modules for faster builds
4. **Download**: Download dependencies
5. **Verify**: Verify dependency integrity
6. **Vet**: Run `go vet` static analysis
7. **Test**: Run tests with race detection and coverage
8. **Upload**: Send coverage to Codecov

#### Lint Job
**Steps:**
1. **Checkout**: Get latest code
2. **Setup Go**: Install Go 1.24
3. **Lint**: Run golangci-lint with custom configuration

#### Security Job
**Steps:**
1. **Checkout**: Get latest code
2. **Setup Go**: Install Go 1.24
3. **Scan**: Run Gosec security scanner
4. **Upload**: Upload SARIF results to GitHub Security

#### Integration Test Job
**Steps:**
1. **Checkout**: Get latest code
2. **Setup Go**: Install Go 1.24
3. **Test**: Run integration tests with cloud provider access

### Release Workflow Jobs

#### Release Job
**Multi-platform builds:**
- Linux: AMD64, ARM64
- macOS: AMD64, ARM64
- Windows: AMD64, ARM64

**Steps:**
1. **Checkout**: Get latest code with full history
2. **Setup Go**: Install Go 1.24
3. **Cache**: Cache Go modules
4. **Download**: Download dependencies
5. **Test**: Run tests to ensure quality
6. **Build**: Build for all platforms with optimizations
7. **Checksum**: Generate SHA256 checksums
8. **Archive**: Create tar.gz and zip archives
9. **Changelog**: Auto-generate changelog from commits
10. **Release**: Create GitHub release with artifacts

#### Docker Job
**Steps:**
1. **Checkout**: Get latest code
2. **Buildx**: Setup Docker Buildx for multi-platform
3. **Login**: Authenticate with Docker Hub
4. **Metadata**: Extract version and tags
5. **Build**: Build and push multi-platform images

## 📦 Build Artifacts

### Binary Releases
- **Linux AMD64**: `driftmgr-linux-amd64.tar.gz`
- **Linux ARM64**: `driftmgr-linux-arm64.tar.gz`
- **macOS AMD64**: `driftmgr-darwin-amd64.tar.gz`
- **macOS ARM64**: `driftmgr-darwin-arm64.tar.gz`
- **Windows AMD64**: `driftmgr-windows-amd64.zip`
- **Windows ARM64**: `driftmgr-windows-arm64.zip`
- **Checksums**: `checksums.txt`

### Docker Images
- **Registry**: Docker Hub (`catherinevee/driftmgr`)
- **Tags**: 
  - `latest` (latest release)
  - `v1.0.0` (specific version)
  - `1.0` (major.minor)
  - `1` (major only)

## 🚀 Deployment Process

### Automatic Release Process
1. **Tag Creation**: Developer creates and pushes a version tag
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **Pipeline Trigger**: Release workflow automatically starts

3. **Build & Test**: Complete test suite runs

4. **Multi-platform Build**: Binaries built for all platforms

5. **Docker Build**: Multi-platform Docker images built

6. **Release Creation**: GitHub release created with:
   - Auto-generated changelog
   - Binary downloads
   - Docker pull commands
   - Installation instructions

### Manual Deployment
```bash
# Test CI locally
make ci-local

# Test release build locally
make release-local

# Create and push tag
git tag v1.0.0
git push origin v1.0.0
```

## 🛡️ Security & Quality

### Security Measures
- **Gosec**: Static security analysis
- **Dependency Scanning**: Weekly vulnerability checks
- **SARIF Upload**: Security results to GitHub Security tab
- **Signed Releases**: SHA256 checksums for all artifacts

### Quality Gates
- **Test Coverage**: Minimum coverage requirements
- **Lint Checks**: Code quality standards
- **Race Detection**: Concurrent safety testing
- **Multi-version Testing**: Go 1.23 and 1.24 compatibility

## 📊 Monitoring & Observability

### Build Status
- **GitHub Actions**: Real-time build status
- **Codecov**: Test coverage tracking
- **Security Tab**: Vulnerability tracking

### Metrics Tracked
- **Build Duration**: Pipeline performance
- **Test Coverage**: Code quality metrics
- **Security Issues**: Vulnerability counts
- **Dependency Health**: Outdated packages

## 🔄 Maintenance

### Weekly Automation
- **Dependency Updates**: Automatic PR creation
- **Security Audits**: Vulnerability scanning
- **Performance Monitoring**: Build time tracking

### Manual Maintenance
- **Workflow Updates**: Pipeline improvements
- **Security Reviews**: Configuration audits
- **Performance Optimization**: Build speed improvements

## 🚨 Troubleshooting

### Common Issues

#### Failed Tests
```bash
# Run tests locally
make test-verbose

# Check specific package
make test-models
```

#### Build Failures
```bash
# Test build locally
make build-all

# Check for compilation issues
go build ./...
```

#### Docker Issues
```bash
# Test Docker build locally
make docker-build

# Run container
make docker-run
```

### Pipeline Debugging
1. **Check Logs**: GitHub Actions logs for detailed error messages
2. **Local Testing**: Use `make ci-local` to reproduce issues
3. **Branch Testing**: Test changes in feature branches first

## 📚 Best Practices

### Development Workflow
1. **Feature Branches**: Develop in feature branches
2. **Pull Requests**: All changes via PR with CI checks
3. **Code Review**: Require reviews before merging
4. **Testing**: Write tests for new features

### Release Management
1. **Semantic Versioning**: Use semver for tags
2. **Release Notes**: Auto-generated from commits
3. **Breaking Changes**: Major version bumps
4. **Hotfixes**: Patch releases for critical fixes

This comprehensive CI/CD pipeline ensures high-quality, secure, and reliable releases while maintaining developer productivity and automation.
