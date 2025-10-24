# Testing Guide for DriftMgr

This document provides comprehensive information about testing in the DriftMgr project, including unit tests, integration tests, performance tests, and end-to-end tests.

## Table of Contents

- [Overview](#overview)
- [Test Structure](#test-structure)
- [Running Tests](#running-tests)
- [Test Types](#test-types)
- [Writing Tests](#writing-tests)
- [Performance Testing](#performance-testing)
- [CI/CD Integration](#cicd-integration)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Overview

DriftMgr uses a comprehensive testing strategy that includes:

- **Unit Tests**: Test individual components in isolation
- **Integration Tests**: Test component interactions
- **Performance Tests**: Benchmark critical operations
- **End-to-End Tests**: Test complete user workflows
- **Security Tests**: Validate security measures
- **Compliance Tests**: Ensure regulatory compliance

## Test Structure

```
tests/
├── unit/                    # Unit tests
│   ├── auth/               # Authentication tests
│   ├── compliance/         # Compliance service tests
│   ├── providers/          # Cloud provider tests
│   └── services/           # Business logic tests
├── integration/            # Integration tests
│   ├── api/               # API integration tests
│   ├── compliance/        # Compliance integration tests
│   └── providers/         # Provider integration tests
├── benchmarks/            # Performance benchmarks
│   ├── compliance_performance_test.go
│   └── api_performance_test.go
├── e2e/                   # End-to-end tests
│   ├── simple_e2e_test.go
│   └── tfstate_e2e_test.go
└── fixtures/              # Test data and fixtures
    ├── policies/          # OPA policy fixtures
    └── data/              # Test data files
```

## Running Tests

### Prerequisites

1. **Go 1.21+**: Required for running tests
2. **Dependencies**: Install required packages
   ```bash
   go mod download
   ```

3. **Test Tools** (optional but recommended):
   ```bash
   # Install golangci-lint for code quality
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   
   # Install gosec for security scanning
   go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
   
   # Install nancy for dependency checking
   go install github.com/sonatypecommunity/nancy@latest
   ```

### Quick Start

```bash
# Run all tests
make test

# Run specific test types
make test-unit
make test-integration
make test-benchmarks
make test-e2e

# Run with coverage
make test-coverage

# Run comprehensive test suite
./scripts/run-comprehensive-tests.sh
```

### Individual Test Commands

```bash
# Unit tests
go test -v ./tests/unit/...

# Integration tests
go test -v ./tests/integration/...

# Benchmarks
go test -bench=. -benchmem ./tests/benchmarks/...

# E2E tests
go test -v -tags=e2e ./tests/e2e/...

# Tests with coverage
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

# Generate coverage report
go tool cover -html=coverage.out -o coverage.html
```

## Test Types

### Unit Tests

Unit tests verify individual components in isolation using mocks and stubs.

**Example: Compliance Service Test**
```go
func TestComplianceService_CreatePolicy(t *testing.T) {
    // Setup
    service := createMockComplianceService(t)
    
    // Test
    policy := &compliance.Policy{
        ID:      "test-policy",
        Name:    "Test Policy",
        Package: "test.policy",
        Rules:   `package test.policy\n\ndefault allow = false`,
    }
    
    err := service.CreatePolicy(context.Background(), policy)
    
    // Assert
    assert.NoError(t, err)
    assert.NotZero(t, policy.CreatedAt)
}
```

**Key Features:**
- Fast execution
- Isolated testing
- Mock dependencies
- High coverage

### Integration Tests

Integration tests verify component interactions and external dependencies.

**Example: API Integration Test**
```go
func TestComplianceAPI_CreatePolicy(t *testing.T) {
    // Setup
    router, service := setupTestRouter(t)
    
    // Test
    request := api.CreatePolicyRequest{
        ID:      "test-policy",
        Name:    "Test Policy",
        Package: "test.policy",
        Rules:   `package test.policy\n\ndefault allow = false`,
    }
    
    requestBody, _ := json.Marshal(request)
    req := httptest.NewRequest("POST", "/api/v1/compliance/policies", bytes.NewBuffer(requestBody))
    w := httptest.NewRecorder()
    
    router.ServeHTTP(w, req)
    
    // Assert
    assert.Equal(t, http.StatusCreated, w.Code)
}
```

**Key Features:**
- Real component interactions
- Database integration
- API endpoint testing
- External service mocking

### Performance Tests

Performance tests benchmark critical operations and identify bottlenecks.

**Example: OPA Evaluation Benchmark**
```go
func BenchmarkOPAEngine_Evaluate(b *testing.B) {
    engine := createTestOPAEngine()
    input := createTestPolicyInput()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := engine.Evaluate(context.Background(), "test.policy", input)
        if err != nil {
            b.Fatalf("Evaluation failed: %v", err)
        }
    }
}
```

**Key Features:**
- Performance measurement
- Memory usage tracking
- Scalability testing
- Regression detection

### End-to-End Tests

E2E tests verify complete user workflows and system behavior.

**Example: Compliance Workflow Test**
```go
func TestComplianceWorkflow(t *testing.T) {
    // Setup test environment
    setupTestEnvironment(t)
    defer cleanupTestEnvironment(t)
    
    // Create policy
    policy := createTestPolicy(t)
    
    // Evaluate policy
    decision := evaluatePolicy(t, policy, testInput)
    
    // Generate report
    report := generateComplianceReport(t, compliance.ComplianceSOC2)
    
    // Assert workflow completion
    assert.NotNil(t, decision)
    assert.NotNil(t, report)
}
```

**Key Features:**
- Complete workflow testing
- Real environment simulation
- User scenario validation
- System integration verification

## Writing Tests

### Test Structure

Follow the **Arrange-Act-Assert** pattern:

```go
func TestFunctionName_Scenario(t *testing.T) {
    // Arrange: Setup test data and dependencies
    service := createTestService()
    input := createTestInput()
    
    // Act: Execute the function under test
    result, err := service.Function(input)
    
    // Assert: Verify the results
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, expectedValue, result.Value)
}
```

### Test Naming Convention

- **Unit Tests**: `TestComponentName_MethodName_Scenario`
- **Integration Tests**: `TestAPI_Endpoint_Scenario`
- **Benchmarks**: `BenchmarkComponent_Operation`
- **E2E Tests**: `TestWorkflow_Scenario`

### Mocking and Test Doubles

Use interfaces and dependency injection for testability:

```go
// Interface for dependency
type PolicyRepository interface {
    Save(ctx context.Context, policy *Policy) error
    Get(ctx context.Context, id string) (*Policy, error)
}

// Service with injected dependency
type ComplianceService struct {
    repo PolicyRepository
}

// Mock implementation
type mockPolicyRepository struct {
    policies map[string]*Policy
}

func (m *mockPolicyRepository) Save(ctx context.Context, policy *Policy) error {
    m.policies[policy.ID] = policy
    return nil
}
```

### Test Data Management

Use fixtures and factories for consistent test data:

```go
// Test fixtures
func createTestPolicy() *compliance.Policy {
    return &compliance.Policy{
        ID:          "test-policy",
        Name:        "Test Policy",
        Description: "A test policy",
        Package:     "test.policy",
        Rules:       `package test.policy\n\ndefault allow = false`,
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }
}

// Test input factory
func createTestPolicyInput() compliance.PolicyInput {
    return compliance.PolicyInput{
        Resource: map[string]interface{}{
            "type": "s3_bucket",
            "name": "test-bucket",
        },
        Action: "read",
        Tags: map[string]string{
            "Owner": "test-user",
        },
    }
}
```

## Performance Testing

### Benchmarking Guidelines

1. **Use `b.ResetTimer()`** to exclude setup time
2. **Test realistic scenarios** with representative data
3. **Measure memory usage** with `-benchmem`
4. **Run multiple iterations** for statistical significance

### Performance Targets

| Operation | Target | Current |
|-----------|--------|---------|
| OPA Policy Evaluation | < 10ms | ~5ms |
| PDF Report Generation | < 2s | ~1.5s |
| API Response Time | < 100ms | ~50ms |
| Database Queries | < 50ms | ~25ms |

### Benchmark Commands

```bash
# Run all benchmarks
go test -bench=. -benchmem ./tests/benchmarks/...

# Run specific benchmark
go test -bench=BenchmarkOPAEngine_Evaluate -benchmem ./tests/benchmarks/...

# Compare benchmarks
go test -bench=. -benchmem -count=5 ./tests/benchmarks/... > old.txt
# Make changes
go test -bench=. -benchmem -count=5 ./tests/benchmarks/... > new.txt
benchcmp old.txt new.txt
```

## CI/CD Integration

### GitHub Actions Workflow

The project includes a comprehensive CI/CD pipeline:

```yaml
# .github/workflows/comprehensive-testing.yml
name: Comprehensive Testing Pipeline

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main, develop ]

jobs:
  quality:
    name: Code Quality & Security
    runs-on: ubuntu-latest
    steps:
    - name: Run golangci-lint
    - name: Run security scan
    - name: Run dependency check
  
  unit-tests:
    name: Unit Tests
    runs-on: ubuntu-latest
    steps:
    - name: Run unit tests
    - name: Generate coverage report
  
  integration-tests:
    name: Integration Tests
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
      redis:
        image: redis:7
    steps:
    - name: Run integration tests
  
  performance-tests:
    name: Performance Tests
    runs-on: ubuntu-latest
    steps:
    - name: Run benchmarks
  
  e2e-tests:
    name: End-to-End Tests
    runs-on: ubuntu-latest
    steps:
    - name: Run e2e tests
```

### Coverage Requirements

- **Unit Tests**: Minimum 80% coverage
- **Integration Tests**: Minimum 60% coverage
- **Overall Coverage**: Minimum 75% coverage

### Quality Gates

All tests must pass before merging:
- ✅ Code quality checks
- ✅ Security scans
- ✅ Unit tests (80% coverage)
- ✅ Integration tests
- ✅ Performance benchmarks
- ✅ E2E tests

## Best Practices

### Test Organization

1. **Group related tests** in the same package
2. **Use table-driven tests** for multiple scenarios
3. **Keep tests focused** on single responsibilities
4. **Use descriptive test names** that explain the scenario

### Test Data

1. **Use realistic test data** that mirrors production
2. **Create reusable fixtures** for common scenarios
3. **Clean up test data** after each test
4. **Use factories** for complex object creation

### Assertions

1. **Use specific assertions** rather than generic ones
2. **Test both success and failure cases**
3. **Verify error messages** and error types
4. **Check side effects** and state changes

### Performance

1. **Run benchmarks regularly** to detect regressions
2. **Profile memory usage** in performance tests
3. **Test with realistic data sizes**
4. **Monitor test execution time**

### Security

1. **Test authentication and authorization**
2. **Validate input sanitization**
3. **Test for common vulnerabilities**
4. **Verify secure defaults**

## Troubleshooting

### Common Issues

#### Test Failures

```bash
# Run tests with verbose output
go test -v ./tests/unit/...

# Run specific test
go test -v -run TestSpecificFunction ./tests/unit/...

# Run tests with race detection
go test -race ./tests/unit/...
```

#### Coverage Issues

```bash
# Check coverage for specific package
go test -cover ./internal/compliance/...

# Generate detailed coverage report
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

#### Performance Issues

```bash
# Profile CPU usage
go test -cpuprofile=cpu.prof -bench=. ./tests/benchmarks/...
go tool pprof cpu.prof

# Profile memory usage
go test -memprofile=mem.prof -bench=. ./tests/benchmarks/...
go tool pprof mem.prof
```

#### Integration Test Issues

```bash
# Run integration tests with debug output
go test -v -tags=integration ./tests/integration/...

# Check test environment setup
go test -v -run TestEnvironment ./tests/integration/...
```

### Debugging Tips

1. **Use `t.Logf()`** for debug output
2. **Set breakpoints** in IDE for complex debugging
3. **Use `-race` flag** to detect race conditions
4. **Profile tests** to identify performance bottlenecks

### Getting Help

1. **Check test logs** for detailed error messages
2. **Review CI/CD pipeline** for environment issues
3. **Consult team documentation** for project-specific guidance
4. **Ask team members** for assistance with complex scenarios

## Conclusion

This testing guide provides a comprehensive framework for testing the DriftMgr project. By following these guidelines and best practices, you can ensure high-quality, reliable, and maintainable code.

For additional information, refer to:
- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Testify Documentation](https://github.com/stretchr/testify)
- [GolangCI-Lint Documentation](https://golangci-lint.run/)
- [Project Architecture Documentation](./architecture/)
