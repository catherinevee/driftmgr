# DriftMgr Testing Guide

This document provides comprehensive information about the testing infrastructure for DriftMgr, including how to run tests, understand test results, and troubleshoot common issues.

## Test Overview

DriftMgr includes a comprehensive testing suite with **95%+ test success rate** across multiple test categories:

- **Unit Tests**: Individual component testing
- **Integration Tests**: Component interaction testing  
- **Benchmark Tests**: Performance validation
- **End-to-End Tests**: Complete workflow validation

## Test Categories

### Unit Tests (`tests/unit/`)

**Status**: 9/9 tests passing (1 skipped due to CGO requirement)

Unit tests focus on individual components and functions:

#### Security Tests (`security_test.go`)
- ✅ **TokenManager**: JWT token generation, validation, and revocation
- ✅ **TokenExpiration**: Proper token expiration handling
- ✅ **RateLimiter**: Rate limiting functionality with window-based limits
- ✅ **PasswordValidator**: Password policy validation
- ✅ **PasswordHashing**: Secure password hashing and comparison
- ✅ **PasswordGeneration**: Secure password generation with strength validation
- ✅ **SecurityMiddleware**: Middleware creation and basic functionality
- ⏭️ **AuthManager**: Skipped due to SQLite CGO requirement

#### Deletion Tests (`deletion_test.go`)
- ✅ **Provider Registration**: AWS, Azure, GCP provider registration
- ✅ **Deletion Options**: Configuration and validation
- ✅ **Safety Checks**: Critical resource type detection
- ✅ **Progress Tracking**: Deletion progress monitoring
- ✅ **Deletion Safety**: Critical resource and tag protection
- ✅ **Deletion Order**: Proper resource deletion sequencing

### Integration Tests (`tests/integration_test.go`)

**Status**: 6/6 tests passing

Integration tests verify component interactions:

#### Cache Integration
- ✅ **Basic Operations**: Set/Get functionality with different data types
- ✅ **Expiration**: TTL-based cache expiration
- ✅ **Type Safety**: Proper type assertion and validation
- ✅ **Concurrent Access**: Thread-safe operations with multiple goroutines
- ✅ **Performance**: Fast operations (completed in ~1ms for 1000 operations)

#### Worker Pool Integration
- ✅ **Task Processing**: All 10 tasks completed successfully
- ✅ **Concurrency**: Proper parallel task execution
- ✅ **Shutdown**: Graceful pool shutdown with timeout handling

#### Security Integration
- ✅ **Token Management**: JWT token generation, validation, and revocation
- ✅ **Rate Limiting**: IP-based request rate limiting
- ✅ **Password Operations**: Hashing, validation, and policy enforcement

#### Semaphore Integration
- ✅ **Acquire/Release**: Basic semaphore functionality
- ✅ **Capacity Limits**: Proper capacity enforcement
- ✅ **Timeout Handling**: Timeout-based acquisition

#### Performance Tests
- ✅ **Concurrent Cache Access**: 1000 operations across 10 goroutines
- ✅ **Cache Performance**: 1000 operations completed in ~1ms

### Benchmark Tests (`tests/benchmarks/`)

**Status**: Infrastructure implemented, ready for execution

Performance benchmarks for core components:

- **CachePerformance**: Set/Get operations
- **SecurityOperations**: Token generation and validation
- **RateLimiter**: Request rate limiting
- **PasswordOperations**: Validation, hashing, and generation
- **ConcurrentOperations**: Parallel cache access
- **MemoryUsage**: Large dataset handling

### End-to-End Tests (`tests/e2e/`)

**Status**: Infrastructure implemented, needs cloud credentials

Complete workflow validation:

- **Complete Workflow**: Full drift detection and remediation cycle
- **Multi-Cloud Workflow**: Cross-provider testing
- **Concurrent Operations**: System behavior under load
- **Error Handling**: Graceful failure handling
- **Performance Under Load**: Sustained operation testing
- **Security Features**: Authentication and authorization
- **Data Integrity**: Consistency validation

## Running Tests

### Quick Start

```bash
# Run all tests
make test

# Run comprehensive test suite
powershell -ExecutionPolicy Bypass -File scripts/test/run_comprehensive_tests.ps1
```

### Specific Test Categories

```bash
# Unit tests
make test-unit
# or
go test ./tests/unit/... -v

# Integration tests
make test-integration
# or
go test ./tests/integration_test.go -v

# Benchmark tests
make test-benchmark
# or
go test -bench=. ./tests/benchmarks/...

# End-to-end tests
go test ./tests/e2e/... -v
```

### Test with Coverage

```bash
# Run tests with coverage reporting
go test ./tests/... -cover

# Generate detailed coverage report
go test ./tests/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Verbose Output

```bash
# Run with verbose output for debugging
go test ./tests/... -v

# Run with race detection
go test ./tests/... -race

# Run with timeout
go test ./tests/... -timeout 30s
```

## CGO Requirement

Some tests require **CGO (C Go)** to be enabled for SQLite database functionality:

### Checking CGO Status

```bash
# Check if CGO is enabled
go env CGO_ENABLED
```

### Enabling CGO

**Windows:**
```bash
set CGO_ENABLED=1
go test ./tests/unit/...
```

**Linux/macOS:**
```bash
export CGO_ENABLED=1
go test ./tests/unit/...
```

### Why CGO is Required

The `TestAuthManager` test requires CGO because:

1. **SQLite Dependency**: Uses `github.com/mattn/go-sqlite3` driver
2. **C Library**: SQLite is a C-based database library
3. **Go Wrapper**: The driver is a Go wrapper around the C library
4. **CGO Bridge**: CGO enables Go to call C libraries

### Alternative Solutions

**Option 1: Use Pure Go SQLite**
Replace `github.com/mattn/go-sqlite3` with `modernc.org/sqlite`

**Option 2: Mock Database**
Use in-memory mocks for testing

**Option 3: Skip Test (Current)**
Gracefully skip when CGO is not available

## Troubleshooting

### Common Test Issues

#### Tests Skipping Due to CGO

**Problem**: Tests show "SKIP" status due to CGO requirement.

**Solution**:
```bash
# Enable CGO
set CGO_ENABLED=1
go test ./tests/unit/...
```

#### Missing Dependencies

**Problem**: Tests fail with import errors.

**Solution**:
```bash
# Update dependencies
go mod tidy
go get github.com/mattn/go-sqlite3
```

#### Integration Test Timeouts

**Problem**: Worker pool tests timeout.

**Solution**:
```bash
# Run with longer timeout
go test ./tests/integration_test.go -v -timeout 30s
```

#### Cache Test Failures

**Problem**: Cache tests fail due to timing issues.

**Solution**:
```bash
# Run with race detection
go test ./tests/integration_test.go -race
```

### Performance Issues

#### Slow Test Execution

**Problem**: Tests take too long to run.

**Solutions**:
```bash
# Run tests in parallel
go test ./tests/... -parallel 4

# Skip slow tests
go test ./tests/... -short
```

#### Memory Issues

**Problem**: Tests consume too much memory.

**Solutions**:
```bash
# Run with memory profiling
go test ./tests/... -memprofile=mem.out

# Limit memory usage
go test ./tests/... -benchmem
```

## Test Infrastructure

### Automated Test Runners

- **PowerShell Script**: `scripts/test/run_comprehensive_tests.ps1`
- **Bash Script**: `scripts/test/run_comprehensive_tests.sh`
- **Makefile Integration**: `make test-unit`, `make test-integration`, etc.

### Test Environment

- **Go Version**: 1.21+
- **Platform**: Windows, Linux, macOS
- **Dependencies**: See `go.mod` for required packages
- **CGO**: Required for SQLite tests

### Test Data

- **Fixtures**: `tests/fixtures/` - Test data and configurations
- **Mocks**: In-memory mocks for external dependencies
- **Samples**: Example configurations and scenarios

## Continuous Integration

### GitHub Actions

Tests are automatically run on:
- Pull requests
- Push to main branch
- Scheduled runs

### Local Development

```bash
# Pre-commit testing
make test

# Full test suite
powershell -ExecutionPolicy Bypass -File scripts/test/run_comprehensive_tests.ps1
```

## Test Results

### Current Status

- **Total Tests**: 25+ tests across multiple categories
- **Passing Tests**: 95%+ success rate
- **Unit Tests**: 9/9 passing (1 skipped due to CGO)
- **Integration Tests**: 6/6 passing
- **Performance**: Sub-millisecond cache operations
- **Concurrency**: 10 concurrent tasks processed successfully

### Detailed Results

For comprehensive test execution results, see [TEST_EXECUTION_RESULTS.md](../TEST_EXECUTION_RESULTS.md).

## Contributing to Tests

### Adding New Tests

1. **Unit Tests**: Add to appropriate package in `tests/unit/`
2. **Integration Tests**: Add to `tests/integration_test.go`
3. **Benchmark Tests**: Add to `tests/benchmarks/`
4. **E2E Tests**: Add to `tests/e2e/`

### Test Guidelines

- **Naming**: Use descriptive test names
- **Coverage**: Aim for high test coverage
- **Isolation**: Tests should be independent
- **Performance**: Keep tests fast
- **Documentation**: Add comments for complex tests

### Test Best Practices

- Use table-driven tests for multiple scenarios
- Mock external dependencies
- Test both success and failure cases
- Include edge cases and error conditions
- Use appropriate assertions and matchers

## Support

For test-related issues:

- **Documentation**: This README and [TEST_EXECUTION_RESULTS.md](../TEST_EXECUTION_RESULTS.md)
- **Issues**: [GitHub Issues](https://github.com/catherinevee/driftmgr/issues)
- **Discussions**: [GitHub Discussions](https://github.com/catherinevee/driftmgr/discussions)

---

**Last Updated**: December 2024
**Test Status**: ✅ 95%+ Success Rate
