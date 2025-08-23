# DriftMgr Testing Infrastructure

This directory contains thorough testing infrastructure for the driftmgr project, including end-to-end tests, integration tests, and performance benchmarks.

## Directory Structure

```
tests/
 README.md # This file
 e2e/ # End-to-end tests
 end_to_end_test.go # Complete workflow testing
 integration/ # Integration tests
 test_multi_cloud_discovery.go # Multi-cloud discovery testing
 benchmarks/ # Performance tests and benchmarks
 performance_test.go # Load testing and profiling
 fixtures/ # Test data and fixtures (auto-generated)
```

## Test Categories

### 1. End-to-End Tests (`tests/e2e/`)

Complete tests that verify complete workflows across the entire application:

- **Multi-provider workflows**: Complete AWS, Azure, and GCP discovery workflows
- **State file processing**: Terraform state file parsing and analysis
- **Drift detection**: End-to-end drift analysis
- **Visualization generation**: Diagram and visualization creation
- **Export functionality**: Data export in various formats
- **Error handling and recovery**: Graceful failure scenarios
- **Configuration variations**: Different configuration scenarios

**Key Features:**
- Uses test fixtures and mock data for reproducible results
- Gracefully handles missing cloud credentials in CI/CD environments
- Tests both success and failure scenarios
- Validates data consistency across operations

### 2. Integration Tests (`tests/integration/`)

Tests that verify integration between different components and cloud providers:

- **Multi-cloud discovery**: AWS, Azure, GCP, and DigitalOcean integration
- **Provider-specific testing**: Each cloud provider's unique characteristics
- **Parallel discovery**: Concurrent resource discovery across providers
- **Resource correlation**: Cross-provider resource relationships
- **Credential management**: Authentication and authorization testing
- **Resource filtering**: Tag-based and type-based filtering
- **Error handling**: Provider-specific error scenarios

**Key Features:**
- Mock cloud responses for consistent testing
- Parallel execution testing
- Credential validation with graceful fallbacks
- Resource type discovery validation
- Performance metrics collection

### 3. Performance Tests (`tests/benchmarks/`)

Complete performance testing and benchmarking:

- **State file processing**: Performance with different file sizes (10 to 10,000 resources)
- **Discovery throughput**: Resource discovery performance across providers
- **Memory usage**: Memory leak detection and usage profiling
- **Concurrency testing**: High-concurrency scenario testing
- **Load testing**: Stress testing under heavy load
- **Resource throughput**: Processing speed for large resource sets

**Key Features:**
- Memory tracking and leak detection
- CPU usage monitoring
- Throughput measurements
- Stress testing capabilities
- Benchmark comparisons
- Resource usage profiling

## Running Tests

### Prerequisites

1. **Go 1.23.8 or higher**
2. **Dependencies**: Run `go mod download` to install all dependencies
3. **Optional Cloud Credentials**: For real cloud provider testing (tests will skip gracefully if not available)

### Running All Tests

```bash
# Run all tests
go test ./tests/...

# Run tests with verbose output
go test -v ./tests/...

# Run tests in short mode (skips long-running tests)
go test -short ./tests/...
```

### Running Specific Test Categories

```bash
# End-to-end tests only
go test ./tests/e2e/...

# Integration tests only
go test ./tests/integration/...

# Performance tests and benchmarks
go test ./tests/benchmarks/...
```

### Running Benchmarks

```bash
# Run all benchmarks
go test -bench=. ./tests/benchmarks/

# Run specific benchmarks
go test -bench=BenchmarkStateFile ./tests/benchmarks/
go test -bench=BenchmarkDiscovery ./tests/benchmarks/

# Run benchmarks with memory profiling
go test -bench=. -benchmem ./tests/benchmarks/

# Run benchmarks multiple times for better accuracy
go test -bench=. -count=5 ./tests/benchmarks/
```

### Running Performance Tests

```bash
# Run performance tests (not benchmarks)
go test -run=TestPerformance ./tests/benchmarks/

# Run load tests
go test -run=TestHighVolume ./tests/benchmarks/
go test -run=TestStress ./tests/benchmarks/

# Run memory leak detection
go test -run=TestMemoryLeak ./tests/benchmarks/
```

## Test Configuration

### Environment Variables

The tests respect several environment variables for configuration:

```bash
# Skip credential-dependent tests
export SKIP_CLOUD_TESTS=true

# Use specific test configuration
export DRIFTMGR_TEST_CONFIG=/path/to/test-config.yaml

# Enable debug logging in tests
export DRIFTMGR_TEST_DEBUG=true

# Specify temporary directory for test artifacts
export DRIFTMGR_TEST_TMPDIR=/tmp/driftmgr-tests
```

### Cloud Provider Credentials

Tests will attempt to use cloud provider credentials if available:

- **AWS**: Uses standard AWS credential chain (AWS CLI, environment variables, IAM roles)
- **Azure**: Uses Azure CLI authentication or environment variables
- **GCP**: Uses service account keys or gcloud authentication

If credentials are not available, tests will skip cloud-dependent operations gracefully.

## Test Data and Fixtures

### Automatic Test Data Generation

The testing infrastructure automatically generates test data:

- **State files**: Various sizes from 10 to 10,000 resources
- **Mock resources**: Realistic cloud resource data
- **Configuration files**: Different configuration scenarios

### Custom Test Data

You can provide custom test data by placing files in the `tests/fixtures/` directory:

```
tests/fixtures/
 state-files/
 small.tfstate
 medium.tfstate
 large.tfstate
 configs/
 test-config.yaml
 mock-data/
 resources.json
```

## Test Patterns and Best Practices

### 1. Graceful Credential Handling

```go
result, err := discoverer.DiscoverResources(ctx, req)
if err != nil {
 if isCredentialError(err) {
 t.Skip("Credentials not available for testing")
 return
 }
 require.NoError(t, err)
}
```

### 2. Resource Validation

```go
for _, resource := range result.Resources {
 assert.Equal(t, "aws", resource.Provider)
 assert.NotEmpty(t, resource.ID)
 assert.NotEmpty(t, resource.Type)
 assert.Contains(t, validRegions, resource.Region)
}
```

### 3. Memory Tracking

```go
memStats := startMemoryTracking()
// ... perform operations ...
finishMemoryTracking(&memStats)

assert.Less(t, memStats.PeakHeap, maxMemoryUsage)
```

### 4. Timeout Handling

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result, err := discoverer.DiscoverResources(ctx, req)
// Handle timeout errors appropriately
```

## Continuous Integration

### GitHub Actions Integration

Add to your `.github/workflows/test.yml`:

```yaml
name: Tests
on: [push, pull_request]

jobs:
 test:
 runs-on: ubuntu-latest
 steps:
 - uses: actions/checkout@v3
 - uses: actions/setup-go@v4
 with:
 go-version: '1.23'

 - name: Run Tests
 run: |
 go test -v ./tests/e2e/
 go test -v ./tests/integration/

 - name: Run Benchmarks
 run: go test -bench=. -benchmem ./tests/benchmarks/

 - name: Run Performance Tests
 run: go test -run=TestPerformance ./tests/benchmarks/
```

### Test Coverage

```bash
# Generate test coverage report
go test -coverprofile=coverage.out ./tests/...

# View coverage in HTML
go tool cover -html=coverage.out -o coverage.html

# Check coverage percentage
go tool cover -func=coverage.out
```

## Troubleshooting

### Common Issues

1. **Missing Dependencies**
 ```bash
 go mod download
 go mod tidy
 ```

2. **Credential Errors**
 - Tests will skip if credentials are not available
 - Set `SKIP_CLOUD_TESTS=true` to skip all cloud-dependent tests

3. **Memory Issues**
 - Increase available memory for large state file tests
 - Use `-short` flag to skip memory-intensive tests

4. **Timeout Issues**
 - Some tests may take longer in CI environments
 - Adjust timeout values in test configuration

### Debug Mode

Enable debug logging for detailed test output:

```bash
export DRIFTMGR_TEST_DEBUG=true
go test -v ./tests/...
```

## Contributing to Tests

### Adding New Tests

1. **End-to-End Tests**: Add to `tests/e2e/` for complete workflow testing
2. **Integration Tests**: Add to `tests/integration/` for component integration testing
3. **Performance Tests**: Add to `tests/benchmarks/` for performance-critical functionality

### Test Naming Conventions

- Test functions: `TestFunctionality`
- Benchmark functions: `BenchmarkFunctionality`
- Test suites: `FunctionalityTestSuite`

### Test Documentation

Document new tests with:
- Purpose and scope
- Prerequisites and setup requirements
- Expected behavior and validation criteria
- Performance characteristics (for benchmarks)

## Performance Baselines

### Typical Performance Expectations

- **Small state files (10 resources)**: < 100ms processing time
- **Medium state files (100 resources)**: < 500ms processing time
- **Large state files (1,000 resources)**: < 2s processing time
- **Huge state files (10,000 resources)**: < 10s processing time
- **Memory usage**: Should not exceed 2x the size of processed data
- **Discovery throughput**: > 10 resources/second per provider

### Benchmarking Guidelines

Run benchmarks multiple times for accurate results:

```bash
go test -bench=. -count=10 ./tests/benchmarks/ | tee benchmark.txt
benchstat benchmark.txt
```

## Test Maintenance

### Regular Maintenance Tasks

1. **Update test data**: Refresh mock data to reflect current cloud services
2. **Review performance baselines**: Ensure expectations remain realistic
3. **Update credentials**: Rotate test credentials periodically
4. **Clean up test artifacts**: Remove temporary files and databases

### Monitoring Test Health

- Monitor test execution time trends
- Track test flakiness and failure rates
- Review resource usage patterns
- Update test environments regularly

For questions or issues with the testing infrastructure, please refer to the main project documentation or open an issue in the project repository.