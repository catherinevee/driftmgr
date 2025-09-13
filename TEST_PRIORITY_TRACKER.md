# Test Priority Tracker - DriftMgr Codecov Improvement

## 🎯 Current Status
- **Current Coverage**: 5.7%
- **Week 1 Target**: 15%
- **Week 2 Target**: 30%
- **Final Target**: 80%

## 📊 Progress Dashboard

### Overall Progress: [████░░░░░░░░░░░░░░░░] 5.7% / 80%

## 🚨 Critical Path (Must Fix First)

### Day 1-2: Build Failures
- [ ] Fix `internal/api/handlers_test.go` - undefined handlers
- [ ] Fix `internal/api/server_test.go` - undefined NewAPIServer
- [ ] Fix `internal/cli/output_test.go` - format string issues
- [ ] Fix `internal/remediation/strategies/*_test.go` - build failures

### Day 3-5: API Package (0% → 40%)
- [ ] Create `handlers_base_test.go` - Test infrastructure
- [ ] Test HealthHandler - 100% coverage
- [ ] Test DiscoverHandler - 80% coverage
- [ ] Test DriftHandler - 80% coverage
- [ ] Test StateHandler - 80% coverage
- [ ] Test RemediationHandler - 70% coverage
- [ ] Test ResourcesHandler - 70% coverage
- [ ] Test error handling - 90% coverage

### Day 6-8: CLI Package (0% → 35%)
- [ ] Fix format string in Warning/Info calls
- [ ] Test command execution framework
- [ ] Test output formatting
- [ ] Test user prompts
- [ ] Test flag parsing
- [ ] Test help generation

### Day 9-10: Remediation Package (0% → 35%)
- [ ] Test planner logic
- [ ] Test executor framework
- [ ] Test terraform import generation
- [ ] Test rollback mechanisms
- [ ] Test dry-run mode

## 📈 Package Coverage Targets

| Package | Current | Day 5 | Day 10 | Week 3 | Week 4 | Final |
|---------|---------|-------|--------|--------|--------|-------|
| **api** | 0% | 40% | 40% | 50% | 60% | 80% |
| **cli** | 0% | 0% | 35% | 45% | 55% | 75% |
| **providers/aws** | 52% | 52% | 55% | 65% | 70% | 85% |
| **providers/azure** | 25% | 25% | 30% | 50% | 60% | 80% |
| **providers/gcp** | 31% | 31% | 35% | 50% | 60% | 80% |
| **providers/digitalocean** | 0% | 0% | 20% | 40% | 50% | 75% |
| **drift/comparator** | 67% | 70% | 72% | 75% | 80% | 90% |
| **discovery** | 8% | 15% | 25% | 40% | 50% | 70% |
| **state** | 28% | 30% | 35% | 45% | 55% | 75% |
| **remediation** | 0% | 0% | 35% | 45% | 55% | 75% |

## 🔧 Implementation Checklist

### Week 1 (Foundation)
#### High Priority
- [ ] Setup mock provider factory
- [ ] Create test data fixtures directory
- [ ] Implement base test helpers
- [ ] Fix all build failures
- [ ] API: handlers_test.go (new)
- [ ] API: server_test.go (fix)
- [ ] API: middleware_test.go (new)

#### Medium Priority
- [ ] CLI: commands_test.go (new)
- [ ] CLI: output_test.go (fix)
- [ ] Discovery: scanner_test.go (fix)

### Week 2 (Core Features)
#### High Priority
- [ ] Remediation: planner_test.go (new)
- [ ] Remediation: executor_test.go (new)
- [ ] State: backend_test.go (enhance)
- [ ] Providers: mock implementations

#### Medium Priority
- [ ] Discovery: parallel_test.go (new)
- [ ] Drift: detector_test.go (new)
- [ ] State: parser_test.go (enhance)

### Week 3 (Provider Coverage)
#### High Priority
- [ ] AWS: ec2_test.go (enhance)
- [ ] AWS: s3_test.go (enhance)
- [ ] Azure: vm_test.go (new)
- [ ] GCP: compute_test.go (enhance)

#### Medium Priority
- [ ] DigitalOcean: provider_test.go (new)
- [ ] AWS: lambda_test.go (new)
- [ ] Azure: storage_test.go (new)

### Week 4 (Integration)
#### High Priority
- [ ] Integration: discovery_flow_test.go
- [ ] Integration: drift_detection_test.go
- [ ] Integration: remediation_flow_test.go
- [ ] E2E: multi_provider_test.go

#### Medium Priority
- [ ] Performance: benchmark_test.go
- [ ] Stress: concurrent_test.go
- [ ] Edge cases: error_paths_test.go

## 📝 Test Template Library

### Basic Unit Test
```go
func TestFunctionName(t *testing.T) {
    // Arrange
    expected := "expected"

    // Act
    result := FunctionName()

    // Assert
    assert.Equal(t, expected, result)
}
```

### Table-Driven Test
```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "test", "TEST", false},
        {"empty input", "", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Function(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            assert.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

### Mock Provider Test
```go
func TestProviderOperation(t *testing.T) {
    mockProvider := &MockProvider{}
    mockProvider.On("ListResources", mock.Anything).Return([]Resource{
        {ID: "1", Name: "test"},
    }, nil)

    result, err := mockProvider.ListResources(context.Background())

    assert.NoError(t, err)
    assert.Len(t, result, 1)
    mockProvider.AssertExpectations(t)
}
```

## 🏆 Success Criteria

### Week 1 Milestones
- ✅ All packages compile without errors
- ✅ API package has 40% coverage
- ✅ Test infrastructure established
- ✅ Mock providers created
- ✅ CI/CD uploads to Codecov

### Week 2 Milestones
- ⬜ Overall coverage reaches 30%
- ⬜ CLI package has 35% coverage
- ⬜ Remediation package has 35% coverage
- ⬜ 600+ unit tests created

### Final Milestones
- ⬜ 80% overall coverage achieved
- ⬜ All critical paths have 90% coverage
- ⬜ Integration tests cover all workflows
- ⬜ Performance benchmarks established
- ⬜ Codecov badge shows green

## 🚀 Quick Commands

```bash
# Check current coverage
go test ./... -cover

# Generate HTML report
go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out

# Test specific package
go test -v -cover ./internal/api/...

# Run with race detection
go test -race ./...

# Upload to Codecov
bash <(curl -s https://codecov.io/bash)

# Run test improvement script
./scripts/test_improvement.sh
```

## 📅 Daily Standup Template

### Date: _______
- **Yesterday**: Completed _______ tests, increased coverage by ____%
- **Today**: Working on _______ package, target _____ tests
- **Blockers**: _______
- **Coverage**: Current ___%, Target ____%

## 🔗 Resources
- [Codecov Dashboard](https://app.codecov.io/gh/catherinevee/driftmgr)
- [Go Testing Guide](https://golang.org/pkg/testing/)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Mock Generation](https://github.com/golang/mock)

---
*Last Updated: [Date]*
*Next Review: [Date + 1 week]*