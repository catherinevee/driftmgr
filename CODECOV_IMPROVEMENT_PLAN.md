# Comprehensive Testing Plan for DriftMgr Codecov Improvement

## Executive Summary
**Current Coverage: 5.7%**
**Target Coverage: 40%** (Phase 1) → **60%** (Phase 2) → **80%** (Phase 3)
**Timeline: 4-6 weeks**

## Current State Analysis

### Coverage Statistics
- **Source Files**: 140 Go files in internal/
- **Test Files**: 35 test files (25% file coverage)
- **Overall Coverage**: 5.7%
- **Lines Covered**: ~2,679 / 47,000

### Package Coverage Breakdown
| Package | Current | Target P1 | Target P2 | Target P3 |
|---------|---------|-----------|-----------|-----------|
| internal/api | 0% | 40% | 60% | 80% |
| internal/cli | 0% | 35% | 55% | 75% |
| internal/providers/aws | 52.2% | 65% | 75% | 85% |
| internal/providers/azure | 24.7% | 50% | 65% | 80% |
| internal/providers/gcp | 31.1% | 50% | 65% | 80% |
| internal/providers/digitalocean | 0% | 40% | 60% | 75% |
| internal/drift/comparator | 67.3% | 75% | 85% | 90% |
| internal/discovery | 8.0% | 30% | 50% | 70% |
| internal/state | 28.5% | 45% | 60% | 75% |
| internal/remediation | 0% | 35% | 55% | 75% |

## Phase 1: Foundation (Week 1-2)
**Goal: Achieve 40% overall coverage**

### Priority 1: Fix Build Failures (Day 1-2) ✅ COMPLETED
```go
// Files fixed:
- internal/api/handlers_test.go ✅
- internal/api/server_test.go ✅
- internal/cli/output_test.go ✅
- internal/cli/prompt.go ✅
```

**Actions Completed:**
1. ✅ Fixed undefined handler references in API tests - Created handlers package
2. ✅ Resolved format string issues in CLI tests - Added format specifiers
3. ✅ Created test utilities for API server
4. ✅ Both packages now compile successfully

**Progress Update (Date: Current):**
- ✅ API package: Builds successfully, tests run
- ✅ CLI package: Builds successfully, all tests pass
- ✅ Remediation package: Builds successfully, tests run
- All critical build failures fixed!
- Next: Create PR for CI/CD verification

### Priority 2: API Package Tests (Day 3-5)
```go
// Target files:
- internal/api/handlers.go → handlers_test.go
- internal/api/server.go → server_test.go
- internal/api/middleware/* → middleware_test.go
- internal/api/websocket/* → websocket_test.go
```

**Test Coverage Goals:**
- Health endpoint: 100%
- CRUD operations: 80%
- Error handling: 90%
- Middleware: 70%

### Priority 3: CLI Package Tests (Day 6-8)
```go
// Target files:
- internal/cli/commands.go → commands_test.go
- internal/cli/output.go → output_test.go
- internal/cli/prompt.go → prompt_test.go
- internal/cli/flags.go → flags_test.go
```

**Test Coverage Goals:**
- Command execution: 70%
- Output formatting: 80%
- User interaction: 60%
- Flag parsing: 90%

### Priority 4: Remediation Package Tests (Day 9-10)
```go
// Target files:
- internal/remediation/planner.go → planner_test.go
- internal/remediation/executor.go → executor_test.go
- internal/remediation/tfimport/* → tfimport_test.go
```

**Test Coverage Goals:**
- Plan generation: 70%
- Execution logic: 60%
- Import generation: 80%

## Phase 2: Enhancement (Week 3-4)
**Goal: Achieve 60% overall coverage**

### Priority 5: Provider Tests Enhancement
```go
// AWS Provider (52.2% → 75%)
- internal/providers/aws/s3_operations_test.go
- internal/providers/aws/ec2_operations_test.go
- internal/providers/aws/lambda_operations_test.go
- internal/providers/aws/dynamodb_operations_test.go

// Azure Provider (24.7% → 65%)
- internal/providers/azure/vm_operations_test.go
- internal/providers/azure/storage_operations_test.go
- internal/providers/azure/network_operations_test.go

// GCP Provider (31.1% → 65%)
- internal/providers/gcp/compute_operations_test.go
- internal/providers/gcp/storage_operations_test.go
- internal/providers/gcp/network_operations_test.go

// DigitalOcean Provider (0% → 60%)
- internal/providers/digitalocean/provider_test.go
- internal/providers/digitalocean/droplet_operations_test.go
```

### Priority 6: Discovery Enhancement (8% → 50%)
```go
// Target files:
- internal/discovery/scanner_test.go (fix failures)
- internal/discovery/parallel_discovery_test.go
- internal/discovery/incremental_test.go (enhance)
- internal/discovery/cache_test.go
```

### Priority 7: State Management (28.5% → 60%)
```go
// Target files:
- internal/state/backend/s3_backend_test.go
- internal/state/backend/azure_backend_test.go
- internal/state/backend/gcs_backend_test.go
- internal/state/parser_test.go (enhance)
- internal/state/validator_test.go (enhance)
```

## Phase 3: Excellence (Week 5-6)
**Goal: Achieve 80% overall coverage**

### Priority 8: Integration Tests
```go
// End-to-end test files:
- tests/integration/discovery_flow_test.go
- tests/integration/drift_detection_test.go
- tests/integration/remediation_flow_test.go
- tests/integration/multi_provider_test.go
```

### Priority 9: Edge Cases & Error Paths
```go
// Focus areas:
- Network failures
- Authentication errors
- Rate limiting
- Concurrent operations
- Large resource sets
- Malformed state files
```

### Priority 10: Performance Tests
```go
// Benchmark files:
- internal/discovery/benchmark_test.go
- internal/drift/benchmark_test.go
- internal/providers/benchmark_test.go
```

## Implementation Strategy

### Test Development Guidelines

#### 1. Test Structure Template
```go
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name    string
        input   interface{}
        want    interface{}
        wantErr bool
    }{
        // Test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

#### 2. Mock Strategy
```go
// Use interfaces for dependencies
type CloudProvider interface {
    Connect(ctx context.Context) error
    ListResources(ctx context.Context) ([]Resource, error)
}

// Create mock implementations
type mockProvider struct {
    mock.Mock
}
```

#### 3. Test Data Management
```go
// Use testdata directories
- testdata/
  - valid_state.json
  - invalid_state.json
  - mock_responses/
    - aws_ec2_response.json
    - azure_vm_response.json
```

### Execution Plan

#### Week 1: Foundation Setup
- [ ] Fix all build failures
- [ ] Setup test infrastructure
- [ ] Create mock providers
- [ ] Implement API tests (0% → 40%)

#### Week 2: Core Functionality
- [ ] Complete CLI tests (0% → 35%)
- [ ] Implement remediation tests (0% → 35%)
- [ ] Enhance discovery tests (8% → 30%)

#### Week 3: Provider Coverage
- [ ] AWS provider tests (52% → 65%)
- [ ] Azure provider tests (25% → 50%)
- [ ] GCP provider tests (31% → 50%)
- [ ] DigitalOcean provider tests (0% → 40%)

#### Week 4: State & Backend
- [ ] State management tests (28% → 45%)
- [ ] Backend tests for S3, Azure, GCS
- [ ] Drift comparator enhancement (67% → 75%)

#### Week 5: Integration & E2E
- [ ] Multi-provider workflows
- [ ] Complete discovery flows
- [ ] Remediation scenarios
- [ ] Error recovery paths

#### Week 6: Polish & Optimization
- [ ] Performance benchmarks
- [ ] Edge case coverage
- [ ] Documentation tests
- [ ] Final coverage push

## CI/CD Integration

### GitHub Actions Workflow
```yaml
name: Test Coverage
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.23'
      - name: Run tests
        run: go test -race -coverprofile=coverage.out ./...
      - name: Upload to Codecov
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out
          flags: unittests
```

### Pre-commit Hooks
```yaml
repos:
  - repo: local
    hooks:
      - id: go-test
        name: Go Tests
        entry: go test ./...
        language: system
        pass_filenames: false
```

## Success Metrics

### Coverage Targets
| Milestone | Overall | Critical Path | Unit Tests | Integration |
|-----------|---------|---------------|------------|-------------|
| Week 1 | 15% | 25% | 300 | 5 |
| Week 2 | 30% | 45% | 600 | 10 |
| Week 3 | 45% | 60% | 900 | 15 |
| Week 4 | 60% | 75% | 1200 | 20 |
| Week 5 | 70% | 85% | 1400 | 30 |
| Week 6 | 80% | 90% | 1600 | 40 |

### Quality Metrics
- Test execution time: < 5 minutes
- Test flakiness: < 1%
- Mock coverage: > 90%
- Assertion density: > 2 per test

## Risk Mitigation

### Potential Blockers
1. **Complex cloud provider mocking**
   - Solution: Use recorded responses (VCR pattern)

2. **Test environment setup**
   - Solution: Docker-based test environments

3. **Flaky integration tests**
   - Solution: Retry mechanisms, proper cleanup

4. **Long test execution time**
   - Solution: Parallel test execution, test categories

## Tooling & Resources

### Required Tools
- **Testing**: testify, mock, gomock
- **Coverage**: go test -cover, codecov
- **Mocking**: mockery, go-vcr
- **Benchmarking**: go test -bench

### Documentation
- Test writing guide
- Mock creation patterns
- Coverage improvement tips
- CI/CD configuration

## Next Steps

### Immediate Actions (Today)
1. Fix build failures in API and CLI packages
2. Create base mock implementations
3. Setup test data fixtures
4. Configure codecov.yml properly

### This Week
1. Implement Phase 1 Priority 1-2
2. Achieve 15% overall coverage
3. Establish testing patterns
4. Document test guidelines

### Tracking Progress
- Daily coverage reports
- Weekly milestone reviews
- Codecov dashboard monitoring
- GitHub Actions status checks

## Conclusion

This comprehensive plan provides a structured approach to improving DriftMgr's test coverage from 5.7% to 80% over 6 weeks. The phased approach ensures:

1. **Quick wins** through fixing build failures and testing high-impact areas
2. **Sustainable progress** by establishing patterns and infrastructure
3. **Quality focus** through proper mocking and test design
4. **Measurable outcomes** via Codecov integration

By following this plan, DriftMgr will achieve enterprise-grade test coverage, ensuring reliability, maintainability, and confidence in the codebase.