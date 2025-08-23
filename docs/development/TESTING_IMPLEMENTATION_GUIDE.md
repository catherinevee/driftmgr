# DriftMgr Testing Implementation Guide

This guide provides practical steps for effectively testing each function within DriftMgr to ensure they work correctly.

## How to Test Each DriftMgr Function

### **Step 1: Identify the Function to Test**

First, identify which DriftMgr function you want to test:

```bash
# List all available functions by category
find internal/ -name "*.go" -exec grep -l "func " {} \;
```

### **Step 2: Understand the Function's Purpose**

For each function, understand:
- **Input**: What parameters does it accept?
- **Output**: What does it return?
- **Side Effects**: Does it modify external state?
- **Dependencies**: What other components does it rely on?

### **Step 3: Create Test Categories**

For each function, create these test types:

#### **A. Unit Tests** (Test function in isolation)
```go
func TestFunctionName_BasicFunctionality(t *testing.T) {
    // Test normal operation
}

func TestFunctionName_ErrorHandling(t *testing.T) {
    // Test error scenarios
}

func TestFunctionName_EdgeCases(t *testing.T) {
    // Test boundary conditions
}
```

#### **B. Integration Tests** (Test with dependencies)
```go
func TestFunctionName_Integration(t *testing.T) {
    // Test with real or mock dependencies
}
```

#### **C. Performance Tests** (Test speed and efficiency)
```go
func BenchmarkFunctionName(b *testing.B) {
    // Measure performance
}
```

### **Step 4: Implement Mock Dependencies**

Create mocks for external dependencies:

```go
type MockCloudProvider struct {
    mock.Mock
}

func (m *MockCloudProvider) ListResources(region string) ([]Resource, error) {
    args := m.Called(region)
    return args.Get(0).([]Resource), args.Error(1)
}
```

### **Step 5: Write Comprehensive Test Cases**

For each function, test these scenarios:

#### **Basic Functionality**
- Normal operation with valid inputs
- Expected outputs for given inputs
- Function completes successfully

#### **Error Handling**
- Invalid inputs
- Missing dependencies
- Network failures
- Timeout scenarios

#### **Edge Cases**
- Empty inputs
- Maximum values
- Null/zero values
- Boundary conditions

#### **Performance**
- Large datasets
- Concurrent operations
- Memory usage
- Response times

## Testing Examples by Function Type

### **1. Resource Discovery Functions**

**Function**: `DiscoverResources()`
**Purpose**: Find cloud resources across providers

**Test Cases**:
```go
// Test basic discovery
func TestDiscoverResources_Basic(t *testing.T) {
    mockProvider := new(MockCloudProvider)
    regions := []string{"us-east-1"}
    expected := []Resource{{ID: "i-123", Type: "ec2"}}
    
    mockProvider.On("ListResources", "us-east-1").Return(expected, nil)
    
    result, err := DiscoverResources(mockProvider, regions, nil)
    
    assert.NoError(t, err)
    assert.Equal(t, expected, result)
}

// Test error handling
func TestDiscoverResources_Error(t *testing.T) {
    mockProvider := new(MockCloudProvider)
    regions := []string{"us-east-1"}
    
    mockProvider.On("ListResources", "us-east-1").Return(nil, errors.New("API error"))
    
    result, err := DiscoverResources(mockProvider, regions, nil)
    
    assert.Error(t, err)
    assert.Nil(t, result)
}

// Test filtering
func TestDiscoverResources_Filtering(t *testing.T) {
    mockProvider := new(MockCloudProvider)
    regions := []string{"us-east-1"}
    resources := []Resource{
        {ID: "i-123", Type: "ec2", State: "running"},
        {ID: "i-456", Type: "ec2", State: "stopped"},
    }
    filters := map[string]string{"state": "running"}
    
    mockProvider.On("ListResources", "us-east-1").Return(resources, nil)
    
    result, err := DiscoverResources(mockProvider, regions, filters)
    
    assert.NoError(t, err)
    assert.Len(t, result, 1)
    assert.Equal(t, "running", result[0].State)
}
```

### **2. Drift Analysis Functions**

**Function**: `AnalyzeDrift()`
**Purpose**: Compare desired vs actual state

**Test Cases**:
```go
// Test no drift scenario
func TestAnalyzeDrift_NoDrift(t *testing.T) {
    desired := []Resource{{ID: "i-123", Type: "ec2"}}
    actual := []Resource{{ID: "i-123", Type: "ec2"}}
    
    result := AnalyzeDrift(desired, actual)
    
    assert.Empty(t, result.DriftItems)
    assert.Equal(t, 0, result.DriftScore)
}

// Test drift detected
func TestAnalyzeDrift_DriftDetected(t *testing.T) {
    desired := []Resource{{ID: "i-123", Type: "ec2", State: "running"}}
    actual := []Resource{{ID: "i-123", Type: "ec2", State: "stopped"}}
    
    result := AnalyzeDrift(desired, actual)
    
    assert.Len(t, result.DriftItems, 1)
    assert.Greater(t, result.DriftScore, 0)
}

// Test missing resources
func TestAnalyzeDrift_MissingResources(t *testing.T) {
    desired := []Resource{{ID: "i-123", Type: "ec2"}}
    actual := []Resource{}
    
    result := AnalyzeDrift(desired, actual)
    
    assert.Len(t, result.DriftItems, 1)
    assert.Equal(t, "missing", result.DriftItems[0].Type)
}
```

### **3. Remediation Functions**

**Function**: `ApplyRemediation()`
**Purpose**: Fix detected drift

**Test Cases**:
```go
// Test safe remediation
func TestApplyRemediation_Safe(t *testing.T) {
    drift := DriftItem{Type: "configuration", ResourceID: "i-123"}
    
    result := ApplyRemediation(drift, false) // dry-run
    
    assert.True(t, result.Success)
    assert.Equal(t, "dry-run", result.Mode)
}

// Test critical resource protection
func TestApplyRemediation_CriticalProtection(t *testing.T) {
    drift := DriftItem{Type: "deletion", ResourceID: "i-123", Critical: true}
    
    result := ApplyRemediation(drift, true)
    
    assert.False(t, result.Success)
    assert.Contains(t, result.Error, "critical resource")
}

// Test approval workflow
func TestApplyRemediation_ApprovalRequired(t *testing.T) {
    drift := DriftItem{Type: "deletion", ResourceID: "i-123", RequiresApproval: true}
    
    result := ApplyRemediation(drift, true)
    
    assert.False(t, result.Success)
    assert.Equal(t, "approval_required", result.Status)
}
```

### **4. Security Functions**

**Function**: `AuthenticateUser()`
**Purpose**: Validate user credentials

**Test Cases**:
```go
// Test valid authentication
func TestAuthenticateUser_Valid(t *testing.T) {
    credentials := Credentials{Username: "user", Password: "password"}
    
    result, err := AuthenticateUser(credentials)
    
    assert.NoError(t, err)
    assert.NotNil(t, result.Token)
    assert.True(t, result.Authenticated)
}

// Test invalid credentials
func TestAuthenticateUser_Invalid(t *testing.T) {
    credentials := Credentials{Username: "user", Password: "wrong"}
    
    result, err := AuthenticateUser(credentials)
    
    assert.Error(t, err)
    assert.False(t, result.Authenticated)
}

// Test rate limiting
func TestAuthenticateUser_RateLimit(t *testing.T) {
    // Make multiple rapid requests
    for i := 0; i < 10; i++ {
        credentials := Credentials{Username: "user", Password: "password"}
        AuthenticateUser(credentials)
    }
    
    // 11th request should be rate limited
    credentials := Credentials{Username: "user", Password: "password"}
    result, err := AuthenticateUser(credentials)
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "rate limit")
}
```

## Testing Tools and Commands

### **Run Tests**
```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./internal/discovery/...

# Run with verbose output
go test ./... -v

# Run with coverage
go test ./... -cover

# Run benchmarks
go test -bench=. ./...

# Run race detection
go test -race ./...
```

### **Generate Coverage Report**
```bash
# Generate coverage file
go test ./... -coverprofile=coverage.out

# View coverage in browser
go tool cover -html=coverage.out -o coverage.html
```

### **Run Specific Test Functions**
```bash
# Run specific test
go test -run TestDiscoverResources_Basic

# Run tests matching pattern
go test -run TestDiscoverResources.*
```

## Test Quality Metrics

### **Coverage Requirements**
- **Unit Tests**: 90%+ line coverage
- **Integration Tests**: 80%+ workflow coverage
- **Critical Functions**: 100% coverage

### **Performance Requirements**
- **Response Time**: < 100ms for most operations
- **Memory Usage**: < 100MB for large datasets
- **Concurrency**: Support 10+ concurrent operations

### **Reliability Requirements**
- **Test Pass Rate**: 95%+ consistently
- **False Positives**: < 1% in drift detection
- **False Negatives**: < 5% in drift detection

## Best Practices

### **1. Test Structure**
```go
func TestFunctionName_Scenario(t *testing.T) {
    // Arrange - Set up test data and mocks
    input := createTestInput()
    expected := createExpectedOutput()
    
    // Act - Execute the function
    result, err := functionToTest(input)
    
    // Assert - Verify results
    assert.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

### **2. Mock Management**
```go
// Set up mock expectations
mockProvider.On("ListResources", "us-east-1").Return(resources, nil)

// Execute function
result := function(mockProvider)

// Verify mock was called correctly
mockProvider.AssertExpectations(t)
```

### **3. Test Data Management**
```go
// Use test fixtures
func loadTestData() *TestData {
    return &TestData{
        Resources: []Resource{...},
        Config:    Config{...},
    }
}

// Clean up after tests
func TestMain(m *testing.M) {
    setup()
    code := m.Run()
    cleanup()
    os.Exit(code)
}
```

### **4. Error Testing**
```go
// Test specific error conditions
func TestFunction_NetworkError(t *testing.T) {
    mockProvider.On("ListResources").Return(nil, networkError)
    
    result, err := function(mockProvider)
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "network")
}
```

## Testing Checklist

### **Before Writing Tests**
- [ ] Understand the function's purpose and behavior
- [ ] Identify all input parameters and output values
- [ ] List all dependencies and external interactions
- [ ] Define success criteria and failure scenarios

### **While Writing Tests**
- [ ] Test normal operation with valid inputs
- [ ] Test error handling with invalid inputs
- [ ] Test edge cases and boundary conditions
- [ ] Test performance with large datasets
- [ ] Test concurrent access if applicable

### **After Writing Tests**
- [ ] Run all tests and verify they pass
- [ ] Check test coverage meets requirements
- [ ] Run performance benchmarks
- [ ] Review test readability and maintainability
- [ ] Update documentation if needed

This comprehensive testing approach ensures that each DriftMgr function is thoroughly validated and ready for production use.
