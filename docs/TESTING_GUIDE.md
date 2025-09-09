# DriftMgr Testing Guide

## Overview
Complete testing guide for DriftMgr, covering unit tests, integration tests, end-to-end tests, and testing best practices.

## Test Coverage
- **Current Coverage**: 87%
- **Target Coverage**: 95% for critical paths
- **Coverage Report**: `go test -cover ./...`

## Testing Strategy

### Test Pyramid
```
        /\
       /E2E\      <- End-to-End Tests (10%)
      /------\
     /  Integ  \   <- Integration Tests (30%)
    /------------\
   /     Unit     \ <- Unit Tests (60%)
  /----------------\
```

### Test Categories

#### Unit Tests (60%)
- Test individual functions and methods
- Mock external dependencies
- Fast execution (<1ms per test)
- Run on every commit

#### Integration Tests (30%)
- Test component interactions
- Use test databases and services
- Medium execution (1-10s per test)
- Run on pull requests

#### End-to-End Tests (10%)
- Test complete workflows
- Use real cloud resources (test accounts)
- Slow execution (>10s per test)
- Run before releases

## Running Tests

### Quick Test Commands
```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific package
go test ./internal/discovery/...

# Run with verbose output
go test -v ./...

# Run with race detection
go test -race ./...

# Run benchmarks
go test -bench=. ./...
```

### Test Scripts
```bash
# Windows
.\scripts\test.ps1
.\scripts\test-all-providers.ps1

# Linux/Mac
./scripts/test.sh
./scripts/test-all-providers.sh
```

## Writing Tests

### Unit Test Example
```go
// internal/discovery/discovery_test.go
package discovery

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestDiscoverResources(t *testing.T) {
    // Arrange
    mockProvider := new(MockProvider)
    mockProvider.On("Discover", mock.Anything).Return([]Resource{
        {ID: "1", Type: "ec2"},
    }, nil)
    
    discovery := NewDiscovery(mockProvider)
    
    // Act
    resources, err := discovery.DiscoverResources()
    
    // Assert
    assert.NoError(t, err)
    assert.Len(t, resources, 1)
    assert.Equal(t, "ec2", resources[0].Type)
    mockProvider.AssertExpectations(t)
}
```

### Integration Test Example
```go
// tests/integration/multi_cloud_discovery_test.go
package integration

import (
    "testing"
    "context"
    "time"
)

func TestMultiCloudDiscovery(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    // Setup test environment
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Create real clients with test credentials
    discovery := setupTestDiscovery(t)
    
    // Execute discovery
    results, err := discovery.DiscoverAll(ctx)
    
    // Verify results
    require.NoError(t, err)
    assert.NotEmpty(t, results)
}
```

### End-to-End Test Example
```go
// tests/e2e/drift_detection_e2e_test.go
package e2e

func TestDriftDetectionWorkflow(t *testing.T) {
    if !*e2eFlag {
        t.Skip("E2E tests not enabled")
    }
    
    // 1. Deploy test infrastructure
    infraID := deployTestInfrastructure(t)
    defer cleanupInfrastructure(infraID)
    
    // 2. Run drift detection
    output := runCommand(t, "driftmgr", "drift", "detect", "--provider", "aws")
    
    // 3. Verify results
    assert.Contains(t, output, "Drift detection complete")
    assert.Contains(t, output, "Resources scanned")
}
```

## Test Data Management

### Fixtures
```go
// tests/fixtures/fixtures.go
var (
    SampleEC2Instance = Resource{
        ID:       "i-1234567890",
        Type:     "aws_instance",
        Provider: "aws",
        Region:   "us-east-1",
    }
    
    SampleTerraformState = []byte(`{
        "version": 4,
        "terraform_version": "1.0.0",
        "resources": []
    }`)
)
```

### Test Helpers
```go
// tests/helpers/helpers.go
func SetupTestDB(t *testing.T) *sql.DB {
    db, err := sql.Open("sqlite3", ":memory:")
    require.NoError(t, err)
    
    // Run migrations
    err = migrate.Up(db)
    require.NoError(t, err)
    
    return db
}

func MockAWSProvider(t *testing.T) *MockProvider {
    mock := new(MockProvider)
    mock.On("Name").Return("aws")
    return mock
}
```

## Mocking

### Interface Mocking
```go
//go:generate mockery --name=Provider --output=mocks
type Provider interface {
    Discover(ctx context.Context) ([]Resource, error)
    ValidateCredentials() error
}
```

### HTTP Mocking
```go
func TestAPIClient(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"status": "ok"}`))
    }))
    defer server.Close()
    
    client := NewClient(server.URL)
    resp, err := client.GetStatus()
    
    assert.NoError(t, err)
    assert.Equal(t, "ok", resp.Status)
}
```

## Test Organization

### Directory Structure
```
tests/
├── unit/           # Unit tests
├── integration/    # Integration tests
├── e2e/           # End-to-end tests
├── benchmarks/    # Performance tests
├── fixtures/      # Test data
├── helpers/       # Test utilities
└── mocks/         # Generated mocks
```

### Naming Conventions
- Test files: `*_test.go`
- Test functions: `TestFunctionName`
- Benchmarks: `BenchmarkFunctionName`
- Examples: `ExampleFunctionName`

## Continuous Integration

### GitHub Actions
```yaml
# .github/workflows/test.yml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
      - run: make test-coverage
      - uses: codecov/codecov-action@v2
```

### Pre-commit Hooks
```yaml
# .pre-commit-config.yaml
repos:
  - repo: local
    hooks:
      - id: go-test
        name: go test
        entry: go test ./...
        language: system
        pass_filenames: false
```

## Performance Testing

### Benchmarks
```go
func BenchmarkDiscovery(b *testing.B) {
    discovery := setupDiscovery()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        discovery.DiscoverResources()
    }
}
```

### Load Testing
```bash
# Using vegeta for API load testing
echo "GET http://localhost:8080/api/health" | vegeta attack -duration=30s | vegeta report
```

## Test Environment

### Environment Variables
```bash
# Test configuration
export TEST_AWS_REGION=us-east-1
export TEST_AZURE_SUBSCRIPTION=test-sub
export TEST_TIMEOUT=30s
export TEST_PARALLEL=4
```

### Docker Test Environment
```yaml
# docker-compose.test.yml
services:
  test:
    build: .
    command: go test -v ./...
    environment:
      - CI=true
      - TEST_DB=postgres://test:test@db:5432/test
    depends_on:
      - db
      - redis
```

## Debugging Tests

### Verbose Output
```bash
go test -v ./...
```

### Debug Specific Test
```bash
go test -run TestSpecificFunction -v ./package
```

### Use Delve Debugger
```bash
dlv test ./package -- -test.run TestFunction
```

## Test Quality Metrics

### Coverage Goals
- Critical paths: 95%
- Core business logic: 90%
- API endpoints: 85%
- Utilities: 70%

### Test Performance
- Unit tests: <100ms each
- Integration tests: <5s each
- E2E tests: <30s each
- Total suite: <5 minutes

### Test Maintenance
- No flaky tests allowed
- Tests must be deterministic
- Clear test names and descriptions
- Regular test review and cleanup

## Common Testing Patterns

### Table-Driven Tests
```go
func TestCalculate(t *testing.T) {
    tests := []struct {
        name     string
        input    int
        expected int
    }{
        {"zero", 0, 0},
        {"positive", 5, 25},
        {"negative", -3, 9},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Calculate(tt.input)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Parallel Tests
```go
func TestParallel(t *testing.T) {
    t.Parallel()
    // Test implementation
}
```

### Cleanup
```go
func TestWithCleanup(t *testing.T) {
    resource := createResource()
    t.Cleanup(func() {
        deleteResource(resource)
    })
    // Test using resource
}
```

## Troubleshooting

### Common Issues

#### Tests Hanging
- Check for deadlocks
- Add timeouts to tests
- Use context with cancellation

#### Flaky Tests
- Remove time dependencies
- Mock external services
- Use deterministic data

#### Slow Tests
- Run in parallel
- Use test short flag
- Cache test data

---

*This document consolidates all testing documentation for DriftMgr.*