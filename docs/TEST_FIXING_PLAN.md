# Comprehensive Test Fixing Plan for DriftMgr v3.0

## Overview
The test suite is currently failing due to the v3.0 architecture reorganization that reduced the codebase from 447 to 63 Go files. This document provides detailed plans to fix each category of test issues.

## 1. Import Path Updates Plan

### Current Issues
- Tests import non-existent packages from pre-v3.0 structure
- Package reorganization broke all import statements
- Missing packages that were consolidated or removed

### Import Mapping Table

| Old Import Path | New Import Path | Action Required |
|----------------|-----------------|-----------------|
| `internal/config` | Configuration merged into various packages | Use specific config structs from relevant packages |
| `internal/cloud/aws` | `internal/providers/aws` | Simple rename |
| `internal/cloud/azure` | `internal/providers/azure` | Simple rename |
| `internal/cloud/gcp` | `internal/providers/gcp` | Simple rename |
| `internal/credentials` | Merged into provider packages | Use provider-specific credential handling |
| `internal/visualization` | Removed - functionality in `internal/api` | Rewrite to use API responses |
| `internal/shared/errors` | Use standard Go errors or package-specific | Remove dependency |

### Implementation Steps

#### Step 1: Automated Import Fix Script
```go
// scripts/fix_test_imports.go
package main

import (
    "io/ioutil"
    "path/filepath"
    "strings"
)

var importMappings = map[string]string{
    "github.com/catherinevee/driftmgr/internal/cloud/aws": "github.com/catherinevee/driftmgr/internal/providers/aws",
    "github.com/catherinevee/driftmgr/internal/cloud/azure": "github.com/catherinevee/driftmgr/internal/providers/azure",
    "github.com/catherinevee/driftmgr/internal/cloud/gcp": "github.com/catherinevee/driftmgr/internal/providers/gcp",
    "github.com/catherinevee/driftmgr/internal/cloud/digitalocean": "github.com/catherinevee/driftmgr/internal/providers/digitalocean",
}

func fixImports(filePath string) error {
    content, err := ioutil.ReadFile(filePath)
    if err != nil {
        return err
    }
    
    fileContent := string(content)
    for oldImport, newImport := range importMappings {
        fileContent = strings.ReplaceAll(fileContent, oldImport, newImport)
    }
    
    return ioutil.WriteFile(filePath, []byte(fileContent), 0644)
}
```

#### Step 2: Manual Import Resolution
For packages that were removed or significantly changed:

1. **Config Package Tests**
   ```go
   // Before
   import "github.com/catherinevee/driftmgr/internal/config"
   cfg := config.New()
   
   // After - use specific configs
   import "github.com/catherinevee/driftmgr/internal/api"
   cfg := &api.Config{
       Host: "localhost",
       Port: 8080,
   }
   ```

2. **Credentials Package Tests**
   ```go
   // Before
   import "github.com/catherinevee/driftmgr/internal/credentials"
   creds := credentials.GetAWSCredentials()
   
   // After - use provider-specific
   import "github.com/catherinevee/driftmgr/internal/providers/aws"
   provider := aws.NewAWSProvider()
   // Credentials handled internally by provider
   ```

#### Step 3: Remove Dead Imports
```bash
# Find and list all test files with import errors
go test ./... 2>&1 | grep "no required module provides package" | cut -d: -f1 | sort -u

# For each file, either:
# 1. Update the import to new path
# 2. Remove the import if functionality no longer exists
# 3. Rewrite the test to use new architecture
```

### Timeline: 2-3 days
- Day 1: Run automated script, fix simple renames
- Day 2: Manual resolution of complex imports
- Day 3: Verify all imports resolve

---

## 2. Mock Rewriting Plan

### Current Issues
- Mock interfaces don't match new method signatures
- Return types have changed
- Method parameters have been modified

### Mock Update Strategy

#### Step 1: Generate New Interface Mocks
```bash
# Install mockery if not present
go install github.com/vektra/mockery/v2@latest

# Generate mocks for all interfaces
mockery --all --dir internal/providers --output tests/mocks
mockery --all --dir internal/drift/detector --output tests/mocks
mockery --all --dir internal/remediation --output tests/mocks
```

#### Step 2: Update Existing Mock Implementations

##### Provider Mock Updates
```go
// OLD Mock
type MockProvider struct {
    mock.Mock
}

func (m *MockProvider) DiscoverResources(ctx context.Context, config map[string]interface{}) ([]providers.CloudResource, error) {
    args := m.Called(ctx, config)
    return args.Get(0).([]providers.CloudResource), args.Error(1)
}

// NEW Mock
type MockProvider struct {
    mock.Mock
}

func (m *MockProvider) Name() string {
    args := m.Called()
    return args.String(0)
}

func (m *MockProvider) DiscoverResources(ctx context.Context, region string) ([]models.Resource, error) {
    args := m.Called(ctx, region)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).([]models.Resource), args.Error(1)
}

func (m *MockProvider) GetResource(ctx context.Context, resourceID string) (*models.Resource, error) {
    args := m.Called(ctx, resourceID)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*models.Resource), args.Error(1)
}

func (m *MockProvider) ValidateCredentials(ctx context.Context) error {
    args := m.Called(ctx)
    return args.Error(0)
}

func (m *MockProvider) ListRegions(ctx context.Context) ([]string, error) {
    args := m.Called(ctx)
    return args.Get(0).([]string), args.Error(1)
}

func (m *MockProvider) SupportedResourceTypes() []string {
    args := m.Called()
    return args.Get(0).([]string)
}
```

##### EventBus Mock Updates
```go
// Create comprehensive mock for all event bus interfaces
type MockEventBus struct {
    mock.Mock
}

func (m *MockEventBus) PublishEvent(eventType string, data interface{}) error {
    args := m.Called(eventType, data)
    return args.Error(0)
}

func (m *MockEventBus) PublishRemediationEvent(event remediation.RemediationEvent) error {
    args := m.Called(event)
    return args.Error(0)
}

func (m *MockEventBus) PublishComplianceEvent(event security.ComplianceEvent) error {
    args := m.Called(event)
    return args.Error(0)
}

func (m *MockEventBus) PublishTenantEvent(event tenant.TenantEvent) error {
    args := m.Called(event)
    return args.Error(0)
}
```

#### Step 3: Create Mock Factory
```go
// tests/mocks/factory.go
package mocks

import (
    "github.com/catherinevee/driftmgr/pkg/models"
    "github.com/stretchr/testify/mock"
)

// MockFactory creates configured mocks for testing
type MockFactory struct{}

func NewMockFactory() *MockFactory {
    return &MockFactory{}
}

func (f *MockFactory) CreateMockProvider(name string) *MockProvider {
    m := &MockProvider{}
    m.On("Name").Return(name)
    m.On("ValidateCredentials", mock.Anything).Return(nil)
    m.On("ListRegions", mock.Anything).Return([]string{"us-east-1", "us-west-2"}, nil)
    m.On("SupportedResourceTypes").Return([]string{"instance", "bucket", "database"})
    return m
}

func (f *MockFactory) CreateMockResource(id, resourceType string) *models.Resource {
    return &models.Resource{
        ID:       id,
        Type:     resourceType,
        Provider: "mock",
        Region:   "us-east-1",
        Properties: map[string]interface{}{
            "name": "mock-resource",
        },
    }
}
```

### Timeline: 3-4 days
- Day 1: Generate new interface mocks with mockery
- Day 2-3: Update existing custom mocks
- Day 4: Create mock factory and helpers

---

## 3. Test Assertion Updates Plan

### Current Issues
- Field names have changed in structs
- Return types are different
- Status enums have new values

### Assertion Update Mapping

#### ActionResult Assertions
```go
// OLD Assertions
assert.True(t, result.Success)
assert.Equal(t, "message", result.Message)
assert.Empty(t, result.Errors)

// NEW Assertions
assert.Equal(t, remediation.StatusSuccess, result.Status)
assert.Equal(t, "message", result.Output)
assert.Empty(t, result.Error)
```

#### Resource Assertions
```go
// OLD
assert.Equal(t, "resource-id", resource.ResourceID)
assert.Equal(t, "aws", resource.CloudProvider)

// NEW
assert.Equal(t, "resource-id", resource.ID)
assert.Equal(t, "aws", resource.Provider)
```

#### Status Assertions
```go
// OLD
if result.Success {
    // success logic
}

// NEW
if result.Status == remediation.StatusSuccess {
    // success logic
}
```

### Implementation Strategy

#### Step 1: Create Assertion Helper Functions
```go
// tests/helpers/assertions.go
package helpers

import (
    "testing"
    "github.com/catherinevee/driftmgr/internal/remediation"
    "github.com/stretchr/testify/assert"
)

func AssertActionSuccess(t *testing.T, result *remediation.ActionResult) {
    assert.Equal(t, remediation.StatusSuccess, result.Status)
    assert.Empty(t, result.Error)
}

func AssertActionFailed(t *testing.T, result *remediation.ActionResult, expectedError string) {
    assert.Equal(t, remediation.StatusFailed, result.Status)
    assert.Contains(t, result.Error, expectedError)
}

func AssertResourceEqual(t *testing.T, expected, actual *models.Resource) {
    assert.Equal(t, expected.ID, actual.ID)
    assert.Equal(t, expected.Type, actual.Type)
    assert.Equal(t, expected.Provider, actual.Provider)
    assert.Equal(t, expected.Region, actual.Region)
}
```

#### Step 2: Bulk Update Script
```go
// scripts/update_assertions.go
package main

import (
    "regexp"
    "strings"
)

var assertionMappings = []struct{
    pattern *regexp.Regexp
    replacement string
}{
    {
        pattern: regexp.MustCompile(`result\.Success`),
        replacement: `result.Status == remediation.StatusSuccess`,
    },
    {
        pattern: regexp.MustCompile(`resource\.ResourceID`),
        replacement: `resource.ID`,
    },
    {
        pattern: regexp.MustCompile(`resource\.CloudProvider`),
        replacement: `resource.Provider`,
    },
    {
        pattern: regexp.MustCompile(`result\.Errors`),
        replacement: `result.Error`,
    },
}

func updateAssertions(content string) string {
    for _, mapping := range assertionMappings {
        content = mapping.pattern.ReplaceAllString(content, mapping.replacement)
    }
    return content
}
```

### Timeline: 2-3 days
- Day 1: Create helper functions
- Day 2: Run bulk update script
- Day 3: Manual verification and fixes

---

## 4. Obsolete Test Removal Plan

### Current Issues
- Tests for deleted functionality
- Tests for consolidated features
- Tests for removed packages

### Identification Strategy

#### Step 1: Mark Obsolete Tests
```go
// Add build tags to isolate obsolete tests
// +build obsolete

package test

// Tests that reference removed functionality
```

#### Step 2: Create Obsolete Test List
```yaml
# tests/obsolete.yaml
obsolete_tests:
  - path: tests/unit/config_test.go
    reason: Config package removed, functionality distributed
    replacement: Use specific package tests
    
  - path: tests/unit/credentials_test.go
    reason: Credentials handling moved to providers
    replacement: Test within provider packages
    
  - path: tests/integration/visualization_test.go
    reason: Visualization package removed
    replacement: Test API endpoints instead
    
  - path: tests/benchmarks/parallel_discovery_test.go
    reason: Parallel discovery is now default behavior
    replacement: Covered by standard discovery tests
```

#### Step 3: Gradual Removal Process
```bash
#!/bin/bash
# scripts/remove_obsolete_tests.sh

# Step 1: Move to obsolete directory
mkdir -p tests/obsolete
mv tests/unit/config_test.go tests/obsolete/
mv tests/unit/credentials_test.go tests/obsolete/

# Step 2: Verify tests still pass without obsolete tests
go test ./...

# Step 3: After verification, remove obsolete directory
# rm -rf tests/obsolete
```

### Decision Matrix for Test Removal

| Criteria | Action | Example |
|----------|--------|---------|
| Package no longer exists | Remove test | `config_test.go` |
| Functionality merged into another package | Move and update test | Provider-specific tests |
| Feature deprecated but still present | Keep but mark deprecated | Legacy API tests |
| Duplicate test after consolidation | Remove duplicate, keep best version | Multiple discovery tests |

### Timeline: 2 days
- Day 1: Identify and mark obsolete tests
- Day 2: Remove or migrate tests

---

## 5. New v3.0 Test Creation Plan

### Current Gaps
- No tests for new unified service layer
- Missing event bus tests
- No tests for new API endpoints
- Missing integration tests for v3.0 features

### Test Coverage Requirements

#### Priority 1: Core Functionality Tests
```go
// tests/unit/providers/provider_test.go
func TestProviderFactory(t *testing.T) {
    tests := []struct {
        name     string
        provider string
        wantErr  bool
    }{
        {"AWS Provider", "aws", false},
        {"Azure Provider", "azure", false},
        {"GCP Provider", "gcp", false},
        {"Invalid Provider", "invalid", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            provider, err := providers.GetProvider(tt.provider)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, provider)
                assert.Equal(t, tt.provider, provider.Name())
            }
        })
    }
}
```

#### Priority 2: Service Layer Tests
```go
// tests/unit/api/services_test.go
func TestAnalyticsService(t *testing.T) {
    service := analytics.NewAnalyticsService()
    
    t.Run("GetPredictiveEngine", func(t *testing.T) {
        engine := service.GetPredictiveEngine()
        assert.NotNil(t, engine)
    })
    
    t.Run("GetTrendAnalyzer", func(t *testing.T) {
        analyzer := service.GetTrendAnalyzer()
        assert.NotNil(t, analyzer)
    })
}

func TestRemediationService(t *testing.T) {
    service := remediation.NewIntelligentRemediationService(nil)
    
    t.Run("GeneratePlan", func(t *testing.T) {
        driftResult := &detector.DriftResult{
            Resource:     "test-resource",
            ResourceType: "instance",
            Provider:     "aws",
            DriftType:    detector.ConfigurationDrift,
        }
        
        plan, err := service.GeneratePlan(context.Background(), driftResult)
        assert.NoError(t, err)
        assert.NotNil(t, plan)
        assert.Len(t, plan.Actions, 1)
    })
}
```

#### Priority 3: Integration Tests
```go
// tests/integration/e2e_test.go
func TestEndToEndWorkflow(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    // Step 1: Create API server
    config := &api.Config{
        Host: "localhost",
        Port: 0, // Random port
    }
    services := &api.Services{}
    server := api.NewServer(config, services)
    
    // Step 2: Start server
    ctx := context.Background()
    err := server.Start(ctx)
    assert.NoError(t, err)
    defer server.Stop(ctx)
    
    // Step 3: Test discovery endpoint
    // Step 4: Test drift detection
    // Step 5: Test remediation
}
```

#### Priority 4: Benchmark Tests
```go
// tests/benchmarks/v3_performance_test.go
func BenchmarkProviderDiscovery(b *testing.B) {
    provider := createMockProvider()
    ctx := context.Background()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = provider.DiscoverResources(ctx, "us-east-1")
    }
}

func BenchmarkRemediationPlanGeneration(b *testing.B) {
    service := remediation.NewIntelligentRemediationService(nil)
    driftResult := createMockDriftResult()
    ctx := context.Background()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = service.GeneratePlan(ctx, driftResult)
    }
}
```

### Test Structure for v3.0
```
tests/
├── unit/
│   ├── providers/
│   │   ├── aws_test.go
│   │   ├── azure_test.go
│   │   ├── gcp_test.go
│   │   └── factory_test.go
│   ├── drift/
│   │   ├── detector_test.go
│   │   └── comparator_test.go
│   ├── remediation/
│   │   ├── planner_test.go
│   │   └── executor_test.go
│   ├── api/
│   │   ├── server_test.go
│   │   ├── handlers_test.go
│   │   └── services_test.go
│   └── state/
│       ├── manager_test.go
│       └── backend_test.go
├── integration/
│   ├── e2e_test.go
│   ├── multi_provider_test.go
│   └── api_test.go
├── benchmarks/
│   ├── discovery_bench_test.go
│   ├── drift_bench_test.go
│   └── remediation_bench_test.go
├── mocks/
│   ├── provider_mock.go
│   ├── eventbus_mock.go
│   └── factory.go
└── helpers/
    ├── assertions.go
    ├── fixtures.go
    └── test_utils.go
```

### Timeline: 5-7 days
- Day 1-2: Core functionality tests
- Day 3-4: Service layer tests
- Day 5-6: Integration tests
- Day 7: Benchmark tests

---

## 6. Implementation Roadmap

### Phase 1: Foundation (Week 1)
1. **Day 1-2**: Update all import paths
2. **Day 3-4**: Rewrite core mocks
3. **Day 5**: Remove obsolete tests

### Phase 2: Core Tests (Week 2)
1. **Day 1-2**: Update test assertions
2. **Day 3-4**: Fix provider tests
3. **Day 5**: Fix drift detection tests

### Phase 3: New Tests (Week 3)
1. **Day 1-2**: Add service layer tests
2. **Day 3-4**: Add integration tests
3. **Day 5**: Add benchmark tests

### Phase 4: Validation (Week 4)
1. **Day 1**: Run full test suite
2. **Day 2**: Fix remaining issues
3. **Day 3**: Update CI/CD pipeline
4. **Day 4**: Documentation
5. **Day 5**: Final validation

## 7. CI/CD Pipeline Updates

### GitHub Actions Workflow Modifications
```yaml
name: Test Suite

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go-version: ['1.22', '1.23']
    
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: ${{ matrix.go-version }}
    
    - name: Install dependencies
      run: |
        go mod download
        go install github.com/vektra/mockery/v2@latest
    
    - name: Generate mocks
      run: |
        mockery --all --dir internal --output tests/mocks
    
    - name: Run unit tests
      run: |
        go test ./tests/unit/... -v -race -coverprofile=coverage.out
    
    - name: Run integration tests
      run: |
        go test ./tests/integration/... -v -short
    
    - name: Run benchmarks
      run: |
        go test ./tests/benchmarks/... -bench=. -benchmem -run=^$
    
    - name: Upload coverage
      uses: codecov/codecov-action@v4
      with:
        file: ./coverage.out
```

## 8. Success Metrics

### Test Coverage Goals
- Unit test coverage: >= 80%
- Integration test coverage: >= 60%
- Critical path coverage: 100%

### Performance Benchmarks
- Provider discovery: < 100ms per resource
- Drift detection: < 500ms for 100 resources
- Remediation plan generation: < 200ms

### Quality Metrics
- Zero flaky tests
- All tests pass in CI/CD
- Test execution time < 5 minutes

## 9. Risk Mitigation

### Potential Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Breaking production code while fixing tests | High | Run tests in isolated branch, gradual migration |
| Time overrun | Medium | Prioritize critical path tests first |
| Missing test coverage for new features | Medium | Create test coverage report, identify gaps |
| Flaky tests in CI/CD | Low | Use test retry logic, fix root causes |

## 10. Maintenance Plan

### Ongoing Test Maintenance
1. **Weekly**: Review test failures in CI/CD
2. **Monthly**: Update test coverage report
3. **Quarterly**: Review and update benchmark baselines
4. **Per Release**: Add tests for new features

### Test Documentation
- Maintain test README with running instructions
- Document mock usage patterns
- Keep assertion helpers updated
- Update this plan as architecture evolves

---

## Conclusion

This comprehensive plan addresses all aspects of fixing the test suite for DriftMgr v3.0. The total estimated timeline is 4 weeks for complete implementation, but core functionality tests can be restored within the first week, allowing CI/CD to function while remaining tests are updated.

The key to success is:
1. Systematic approach - fix imports first, then mocks, then assertions
2. Prioritization - focus on critical path tests
3. Automation - use scripts to handle bulk updates
4. Documentation - keep track of changes for future maintenance

By following this plan, the test suite will be fully compatible with the v3.0 architecture and provide comprehensive coverage for all functionality.