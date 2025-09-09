# DriftMgr Scripts

This directory contains scripts for building, testing, and deploying DriftMgr.

## Directory Structure

```
scripts/
├── build/         # Build scripts for different platforms
├── deploy/        # CI/CD configuration examples
├── installer/     # Platform-specific installers
├── remediation/   # Drift remediation scripts
├── test/          # Test automation scripts
├── tools/         # Development and utility tools
└── verify/        # Verification and validation scripts
```

## Available Scripts

### Installation & Setup
- `install.sh` - Linux/macOS installation
- `install.ps1` - Windows PowerShell installation
- `install.bat` - Windows batch installation
- `setup-github-secrets.sh` - Configure GitHub Actions secrets
- `setup-github-secrets.ps1` - Configure GitHub Actions secrets (Windows)
- `manage-secrets.sh` - Comprehensive secrets management tool

### Building
- `build/build.sh` - Build DriftMgr for current platform
- `build/build-all.sh` - Build for all supported platforms
- `build/build-all.bat` - Build for all platforms (Windows)
- `build/build-enhanced-cli.sh` - Build with enhanced CLI features
- `build/install.sh` - Build and install locally

### Testing
- `run-tests.sh` - Run all tests
- `run-tests.ps1` - Run all tests (Windows)
- `test.sh` - Quick test runner
- `test-all-providers.sh` - Test all cloud provider integrations
- `test-discovery.sh` - Test resource discovery
- `test-credential-update.sh` - Test credential management
- `test_all_features.sh` - Comprehensive feature testing

### Test Suites
- `test/run-functionality-tests.sh` - Functional test suite
- `test/run-security-tests.sh` - Security test suite
- `test/test-github-actions.sh` - GitHub Actions workflow tests
- `test/test-security-fixes.sh` - Security fix validation

### Verification
- `verify/cloud_shell_verify.sh` - Verify cloud shell compatibility
- `verify/generate_verification_commands.sh` - Generate verification commands
- `verify/inventory_services.sh` - Inventory cloud services
- `verify/tag_based_verification.sh` - Verify resources by tags

### Remediation
- `remediation/remediate-drift.sh` - Automated drift remediation
- `remediation/remediate-drift.bat` - Automated drift remediation (Windows)

### Development Tools
- `tools/update-imports.sh` - Update Go imports after refactoring
- `tools/user_simulation.py` - Simulate user interactions
- `tools/verify_driftmgr.ps1` - Verify DriftMgr installation
- `tools/verify_driftmgr_cli.py` - CLI verification tool

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

### Build from Source
```bash
# Build for current platform
./scripts/build/build.sh

# Build for all platforms
./scripts/build/build-all.sh

# Windows
.\scripts\build\build.ps1
```

### Run Tests
```bash
# Run all tests
./scripts/run-tests.sh

# Test specific provider
./scripts/test-all-providers.sh

# Windows
.\scripts\run-tests.ps1
```

### Setup GitHub Secrets
```bash
# Interactive setup
./scripts/setup-github-secrets.sh

# Advanced management
./scripts/manage-secrets.sh setup
./scripts/manage-secrets.sh validate
./scripts/manage-secrets.sh export
```

### Verify Installation
```bash
# Linux/macOS
./scripts/verify/cloud_shell_verify.sh

# Windows
.\scripts\tools\verify_driftmgr.ps1
```

## CI/CD Integration

The scripts in `deploy/` provide examples for various CI/CD platforms:

- **GitHub Actions**: See `.github/workflows/` directory
- **GitLab CI**: `deploy/gitlab-ci/.gitlab-ci.yml`
- **Jenkins**: `deploy/jenkins/Jenkinsfile`
- **CircleCI**: `deploy/circleci/.circleci/config.yml`
- **Azure DevOps**: `deploy/azure-devops/azure-pipelines.yml`

## Environment Variables

Many scripts support environment variables for configuration:

- `DRIFTMGR_CONFIG` - Path to configuration file
- `DRIFTMGR_LOG_LEVEL` - Logging level (debug, info, warn, error)
- `DRIFTMGR_PROVIDER` - Default cloud provider
- `DRIFTMGR_REGION` - Default region
- `DRIFTMGR_OUTPUT` - Output format (json, table, summary)

## Contributing

When adding new scripts:
1. Place in appropriate subdirectory
2. Add execution permissions: `chmod +x script.sh`
3. Include help text with `--help` flag support
4. Update this README with the new script
5. Test on both Linux/macOS and Windows if applicable