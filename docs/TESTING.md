# DriftMgr Testing Documentation

Comprehensive testing suite for validating all DriftMgr functionality.

## Overview

DriftMgr includes multiple testing layers to ensure reliability:
1. **Unit Tests** - Test individual functions and modules
2. **Integration Tests** - Test component interactions  
3. **Functional Tests** - Test complete command workflows
4. **CLI Tests** - Test command-line interface behavior
5. **Performance Tests** - Validate performance requirements

## Quick Start

### Run All Tests
```powershell
# Windows
.\scripts\run_all_tests.ps1

# Linux/macOS
./scripts/test_driftmgr_comprehensive.sh
```

### Run Specific Test Categories
```powershell
# Run only basic tests (fast)
.\scripts\test_driftmgr_comprehensive.ps1 -TestCategory basic

# Run discovery tests
.\scripts\test_driftmgr_comprehensive.ps1 -TestCategory discovery

# Run with verbose output
.\scripts\test_driftmgr_comprehensive.ps1 -Verbose
```

## Test Categories

### Basic Commands
- Help command display
- Status command execution
- Unknown command handling
- Invalid flag detection

### Credential Detection
- Provider credential detection
- Multiple profile handling
- AWS multi-account support
- Account switching functionality

### Discovery Commands
- Resource discovery across providers
- Output format validation (JSON, table, summary)
- Auto-discovery mode
- All-accounts flag functionality

### Drift Detection
- State file loading
- Drift detection logic
- Smart defaults application
- Severity classification

### State Management
- State file discovery
- Terraform backend detection
- State visualization
- Remote state handling

### Account Management
- Account listing
- Account selection
- Profile switching
- Multi-subscription support

### Export/Import
- Export to various formats
- Import validation
- File handling
- Data integrity

### Error Handling
- Invalid argument handling
- Missing required parameters
- File path validation
- Special character handling

### Color and Progress
- Color output validation
- NO_COLOR environment support
- FORCE_COLOR environment support
- Progress indicator display

### Performance
- Help command < 1 second
- Status command < 5 seconds
- Discovery performance
- Memory usage validation

## Go Tests

### Running Unit Tests
```bash
# Run all unit tests
go test ./...

# Run with coverage
go test -cover ./...

# Run with verbose output
go test -v ./...

# Run specific package
go test ./internal/credentials

# Run with race detection
go test -race ./...
```

### Running Functional Tests
```bash
cd tests/functional
go test -v

# Run specific test
go test -v -run TestBasicCommands

# Run benchmarks
go test -bench=.
```

## Test Scripts

### Comprehensive Test Script (PowerShell)
Location: `scripts/test_driftmgr_comprehensive.ps1`

Features:
- Tests all commands and flags
- Validates error handling
- Checks performance requirements
- Tests edge cases
- Provides detailed reporting

Options:
- `-Verbose`: Show detailed output
- `-StopOnError`: Stop on first failure
- `-TestCategory`: Run specific category

### Comprehensive Test Script (Bash)
Location: `scripts/test_driftmgr_comprehensive.sh`

Features:
- Cross-platform testing
- POSIX compliant
- Color-coded output
- Performance validation

Options:
- `-v, --verbose`: Verbose output
- `-s, --stop-on-error`: Stop on failure
- `-c, --category`: Test category
- `-p, --path`: DriftMgr path

## Test Data

### Mock Credentials
Tests can use mock credentials for testing:
```bash
export AWS_PROFILE=test
export AZURE_SUBSCRIPTION_ID=test
export GOOGLE_CLOUD_PROJECT=test
```

### Test State Files
Sample Terraform state files are in `examples/statefiles/`

### Test Configurations
Test configuration files are in `configs/`

## Continuous Integration

### GitHub Actions
Tests run automatically on:
- Pull requests
- Commits to main
- Release tags

### Local Pre-commit
```bash
# Install pre-commit hook
cp scripts/pre-commit .git/hooks/

# Run tests before commit
git commit  # Tests run automatically
```

## Coverage Reports

### Generate Coverage
```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View in terminal
go tool cover -func=coverage.out

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html
```

### Coverage Goals
- Overall: > 70%
- Core packages: > 80%
- Critical paths: > 90%

## Writing New Tests

### Go Test Example
```go
func TestNewFeature(t *testing.T) {
    // Arrange
    input := "test-input"
    expected := "expected-output"
    
    // Act
    result := NewFeature(input)
    
    // Assert
    if result != expected {
        t.Errorf("Expected %s, got %s", expected, result)
    }
}
```

### CLI Test Example
```powershell
Test-Command -TestName "New Feature Test" `
    -Command "$script:DriftMgrPath new-feature --flag value" `
    -ExpectedOutput "Success" `
    -ExpectedExitCode 0
```

## Troubleshooting

### Common Issues

1. **Tests fail with "executable not found"**
   - Build the project first: `go build -o driftmgr.exe ./cmd/driftmgr`

2. **Credential tests fail**
   - Configure at least one cloud provider
   - Or set mock credentials for testing

3. **Performance tests fail**
   - Ensure system isn't under heavy load
   - Check network connectivity for API calls

4. **Color tests fail on Windows**
   - Ensure Windows Terminal or compatible terminal
   - Check ANSI color support

### Debug Mode

Enable debug output:
```bash
# Set debug environment variable
export DRIFTMGR_DEBUG=1

# Run tests with debug output
./scripts/test_driftmgr_comprehensive.sh -v
```

## Test Matrix

| Test Type | Coverage | Frequency | Duration |
|-----------|----------|-----------|----------|
| Unit | Functions & methods | Every commit | < 1 min |
| Integration | Component interactions | Every PR | < 5 min |
| Functional | Complete workflows | Every PR | < 10 min |
| CLI | Command interface | Every release | < 5 min |
| Performance | Speed & resources | Weekly | < 15 min |
| Security | Vulnerabilities | Monthly | < 30 min |

## Best Practices

1. **Write tests first** - TDD approach for new features
2. **Test edge cases** - Empty inputs, large data, special characters
3. **Mock external dependencies** - Don't rely on external services
4. **Keep tests fast** - Use `-short` flag for slow tests
5. **Clean up after tests** - Remove temporary files and data
6. **Use descriptive names** - Test names should explain what they test
7. **Avoid flaky tests** - Tests should be deterministic
8. **Test error paths** - Validate error handling

## Validation Checklist

Before marking a feature complete:
- [ ] Unit tests written and passing
- [ ] Integration tests updated
- [ ] CLI commands tested
- [ ] Error cases handled
- [ ] Performance validated
- [ ] Documentation updated
- [ ] Coverage maintained/improved