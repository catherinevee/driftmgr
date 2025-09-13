# DriftMgr Test Coverage Improvement Plan
## Target: 80% Code Coverage

### Overview
This plan outlines a phased approach to improve test coverage from ~30% to 80%, with verification through CodeCov after each phase.

### CodeCov Integration
- **Repository**: https://app.codecov.io/gh/catherinevee/driftmgr
- **Current Coverage**: ~30%
- **Target Coverage**: 80%
- **Verification**: After each phase, push to GitHub and check CodeCov report

---

## Phase 1: Fix Failing Tests & Establish Baseline
**Target Coverage: 35%** | **Duration: 1-2 days**

### Tasks
1. Fix failing tests in `internal/drift/detector/`
   - [ ] Fix `TestEnhancedDetector_ErrorHandling`
   - [ ] Fix `TestNewEnhancedDetector`
   - [ ] Ensure all existing tests pass

2. Set up coverage reporting
   - [ ] Add coverage to GitHub Actions workflow
   - [ ] Configure codecov.yml for proper reporting
   - [ ] Create baseline coverage report

### Files to Fix
- `internal/drift/detector/enhanced_detector_test.go`
- `internal/drift/detector/enhanced_detector_error_test.go`

### Verification Commands
```bash
go test ./... -v -race -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
git add . && git commit -m "Phase 1: Fix failing tests"
git push origin main
# Check CodeCov: https://app.codecov.io/gh/catherinevee/driftmgr
```

---

## Phase 2: Core State Management Tests
**Target Coverage: 50%** | **Duration: 2-3 days**

### Priority Files (High Impact)
1. **State Parser** (`internal/state/parser.go` - 12KB)
   - [ ] Test Terraform state parsing (v0.11-1.x)
   - [ ] Test resource extraction
   - [ ] Test error handling for malformed states
   - [ ] Use golden files for test data

2. **State Manager** (`internal/state/manager.go` - 13KB)
   - [ ] Test CRUD operations
   - [ ] Test state locking mechanisms
   - [ ] Test remote backend operations
   - [ ] Test state migration

3. **State Validator** (`internal/state/validator.go` - 10KB)
   - [ ] Test validation rules
   - [ ] Test resource address validation
   - [ ] Test JSON validation
   - [ ] Test custom rule addition/removal

4. **Backup Manager** (`internal/state/backup.go` - 8KB)
   - [ ] Test backup creation/restoration
   - [ ] Test compression/encryption
   - [ ] Test cleanup of old backups
   - [ ] Test metadata management

### Test Files to Create
```
internal/state/parser_test.go
internal/state/manager_test.go
internal/state/validator_test.go
internal/state/backup_test.go
```

### Verification
```bash
go test ./internal/state/... -v -coverprofile=phase2.out
go tool cover -func=phase2.out | grep total
git add . && git commit -m "Phase 2: Add state management tests"
git push origin main
# Check CodeCov for 50% target
```

---

## Phase 3: Discovery Engine Tests
**Target Coverage: 60%** | **Duration: 3-4 days**

### Priority Files (Highest LOC Impact)
1. **Enhanced Discovery** (`internal/discovery/enhanced_discovery.go` - 211KB!)
   - [ ] Mock cloud provider APIs
   - [ ] Test resource discovery per provider
   - [ ] Test pagination handling
   - [ ] Test error recovery
   - [ ] Test filtering and query options

2. **Incremental Discovery** (`internal/discovery/incremental.go` - 13KB)
   - [ ] Test bloom filter implementation
   - [ ] Test change detection
   - [ ] Test incremental updates

3. **Parallel Discovery** (`internal/discovery/parallel_discovery.go` - 7KB)
   - [ ] Test concurrent discovery
   - [ ] Test rate limiting
   - [ ] Test worker pool management

4. **SDK Integration** (`internal/discovery/sdk_integration.go` - 12KB)
   - [ ] Test SDK initialization
   - [ ] Test credential handling
   - [ ] Test retry logic

### Test Files to Create
```
internal/discovery/enhanced_discovery_test.go
internal/discovery/incremental_test.go
internal/discovery/parallel_discovery_test.go
internal/discovery/sdk_integration_test.go
```

### Verification
```bash
go test ./internal/discovery/... -v -coverprofile=phase3.out
go tool cover -func=phase3.out | grep total
git add . && git commit -m "Phase 3: Add discovery engine tests"
git push origin main
# Check CodeCov for 60% target
```

---

## Phase 4: Cloud Provider Tests
**Target Coverage: 70%** | **Duration: 2-3 days**

### Provider Tests
1. **Azure Provider** (`internal/providers/azure/provider.go`)
   - [ ] Test authentication methods
   - [ ] Test resource discovery
   - [ ] Test error handling
   - [ ] Mock Azure SDK calls

2. **GCP Provider** (`internal/providers/gcp/provider.go`)
   - [ ] Test service account auth
   - [ ] Test resource listing
   - [ ] Test project iteration
   - [ ] Mock GCP SDK calls

3. **DigitalOcean Provider** (`internal/providers/digitalocean/provider.go`)
   - [ ] Test API token auth
   - [ ] Test droplet/resource discovery
   - [ ] Mock DO API calls

4. **AWS Provider Enhancement** (`internal/providers/aws/provider.go`)
   - [ ] Increase existing coverage
   - [ ] Test cross-account access
   - [ ] Test all resource types

### Shared Test Suite
Create a shared test interface for all providers:
```go
// internal/providers/provider_test_suite.go
type ProviderTestSuite interface {
    TestAuthentication()
    TestDiscovery()
    TestErrorHandling()
    TestPagination()
}
```

### Test Files to Create
```
internal/providers/azure/provider_test.go
internal/providers/gcp/provider_test.go
internal/providers/digitalocean/provider_test.go
internal/providers/provider_test_suite.go
```

### Verification
```bash
go test ./internal/providers/... -v -coverprofile=phase4.out
go tool cover -func=phase4.out | grep total
git add . && git commit -m "Phase 4: Add cloud provider tests"
git push origin main
# Check CodeCov for 70% target
```

---

## Phase 5: API, CLI & Integration Tests
**Target Coverage: 80%** | **Duration: 2-3 days**

### API Server Tests
1. **Server** (`internal/api/server.go`)
   - [ ] Test server initialization
   - [ ] Test middleware
   - [ ] Test WebSocket connections

2. **Handlers** (`internal/api/handlers.go`)
   - [ ] Test all HTTP endpoints
   - [ ] Test request validation
   - [ ] Test error responses
   - [ ] Use httptest package

3. **Router** (`internal/api/router.go`)
   - [ ] Test route registration
   - [ ] Test path matching

### CLI Command Tests
1. **Main Commands** (`cmd/driftmgr/commands/`)
   - [ ] Test command execution
   - [ ] Test flag parsing
   - [ ] Test output formatting

### Remediation Tests
1. **Remediation Engine** (`internal/remediation/`)
   - [ ] Test remediation planning
   - [ ] Test execution logic
   - [ ] Test rollback capabilities

### Test Files to Create
```
internal/api/server_test.go
internal/api/handlers_test.go
internal/api/router_test.go
cmd/driftmgr/commands/discover_test.go
cmd/driftmgr/commands/remediate_test.go
internal/remediation/planner_test.go
```

### Verification
```bash
go test ./... -v -race -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
go tool cover -func=coverage.out | grep total
git add . && git commit -m "Phase 5: Add API and CLI tests - 80% coverage achieved"
git push origin main
# Check CodeCov for 80% target
```

---

## Testing Best Practices

### 1. Use Table-Driven Tests
```go
func TestStateParser(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    *State
        wantErr bool
    }{
        // Test cases
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test logic
        })
    }
}
```

### 2. Mock External Dependencies
```go
type mockCloudProvider struct {
    mock.Mock
}

func (m *mockCloudProvider) DiscoverResources(ctx context.Context) ([]Resource, error) {
    args := m.Called(ctx)
    return args.Get(0).([]Resource), args.Error(1)
}
```

### 3. Use Golden Files for Large Test Data
```go
func TestParseState(t *testing.T) {
    golden := filepath.Join("testdata", "terraform.tfstate.golden")
    // Compare output with golden file
}
```

### 4. Parallel Tests Where Possible
```go
func TestSomething(t *testing.T) {
    t.Parallel()
    // Test logic
}
```

---

## Continuous Integration Setup

### GitHub Actions Workflow Addition
```yaml
# .github/workflows/test-coverage.yml
name: Test Coverage

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.23'

    - name: Run tests with coverage
      run: go test -race -coverprofile=coverage.out -covermode=atomic ./...

    - name: Upload coverage to Codecov
      uses: codecov/codecov-action@v4
      with:
        token: ${{ secrets.CODECOV_TOKEN }}
        file: ./coverage.out
        flags: unittests
        name: codecov-umbrella
```

---

## Monitoring Progress

### After Each Phase
1. Run local coverage check:
   ```bash
   go test ./... -coverprofile=coverage.out
   go tool cover -func=coverage.out | grep total
   ```

2. Push to GitHub:
   ```bash
   git push origin main
   ```

3. Check CodeCov dashboard:
   - Visit: https://app.codecov.io/gh/catherinevee/driftmgr
   - Review coverage percentage
   - Check coverage trends
   - Identify remaining uncovered lines

### Coverage Badges
Add to README.md:
```markdown
[![codecov](https://codecov.io/gh/catherinevee/driftmgr/branch/main/graph/badge.svg)](https://codecov.io/gh/catherinevee/driftmgr)
```

---

## Success Metrics

### Per Phase
- **Phase 1**: All tests passing, baseline established
- **Phase 2**: 50% coverage, state management fully tested
- **Phase 3**: 60% coverage, discovery engine tested
- **Phase 4**: 70% coverage, all providers tested
- **Phase 5**: 80% coverage target achieved

### Final Deliverables
- [ ] 80% overall code coverage
- [ ] All critical paths tested
- [ ] No failing tests
- [ ] CodeCov integration working
- [ ] Coverage badge in README
- [ ] Automated coverage checks in CI/CD

---

## Timeline Summary

**Total Duration: 10-15 days**

- Phase 1: Days 1-2 (Baseline)
- Phase 2: Days 3-5 (State Management)
- Phase 3: Days 6-9 (Discovery Engine)
- Phase 4: Days 10-12 (Providers)
- Phase 5: Days 13-15 (API/CLI/Integration)

---

## Next Steps

1. Start with Phase 1 immediately
2. Fix failing tests first
3. Set up CodeCov GitHub Action
4. Begin systematic test creation
5. Monitor progress on CodeCov dashboard after each phase