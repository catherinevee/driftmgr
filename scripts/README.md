# DriftMgr Scripts

This directory contains scripts for building, testing, and deploying DriftMgr.

## Directory Structure

```
scripts/
├── build/          # Build scripts for different platforms
├── deploy/         # Deployment and CI/CD scripts
├── install/        # Installation scripts
├── test/           # Test automation scripts
└── tools/          # Development and utility tools
```

## Main Scripts

### Installation
- `install.sh` - Linux/macOS installation
- `install.ps1` - Windows PowerShell installation
- `install.bat` - Windows batch installation

### Testing
- `run-tests.sh` - Run all tests (Linux/macOS)
- `run-tests.ps1` - Run all tests (Windows)

### Development Tools
- `tools/dev-setup.sh` - Set up development environment
- `tools/update-imports.sh` - Update Go imports after refactoring
- `tools/clean-build.sh` - Clean build artifacts

## Usage Examples

### Install DriftMgr
```bash
# Linux/macOS
./scripts/install.sh

# Windows PowerShell
.\scripts\install.ps1

# Windows Command Prompt
scripts\install.bat
```

### Run Tests
```bash
# Linux/macOS
./scripts/run-tests.sh

# Windows
.\scripts\run-tests.ps1
```

### Deploy to Production
```bash
# Using deployment script
./scripts/deploy/deploy.sh production

# Using Docker
docker-compose -f deploy/docker-compose.yml up -d
```

## CI/CD Integration

The scripts in `deploy/` are designed to work with various CI/CD platforms:
- GitHub Actions: See `.github/workflows/`
- GitLab CI: Use `deploy/gitlab-ci/`
- Jenkins: Use `deploy/jenkins/`
- CircleCI: Use `deploy/circleci/`

## Development Scripts

For development tasks:
```bash
# Update all Go imports after refactoring
./scripts/tools/update-imports.sh

# Clean and rebuild
./scripts/build/clean-build.sh

# Generate documentation
./scripts/tools/generate-docs.sh
```