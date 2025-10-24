# 🚀 Phase 1: Simulation System Completion - Implementation Guide

## 📋 **Overview**

This guide provides detailed step-by-step instructions for completing the simulation system in DriftMgr. The simulation system is responsible for simulating various types of drift in cloud resources across AWS, Azure, and GCP.

## 🎯 **Objectives**

- Complete all missing drift types in AWS simulator
- Complete all missing drift types in Azure simulator  
- Complete all missing drift types in GCP simulator
- Implement comprehensive error handling
- Create comprehensive test coverage
- Ensure 100% functionality

## 📊 **Current Status**

### **AWS Simulator (`internal/simulation/aws_simulator.go`)**
- ❌ Missing drift types: `driftType` not implemented
- ❌ Missing tag drift simulation for all resource types
- ❌ Missing attribute change simulation
- ❌ Incomplete error handling

### **Azure Simulator (`internal/simulation/azure_simulator.go`)**
- ❌ Missing drift types: `driftType` not implemented
- ❌ Missing tag drift simulation for all resource types
- ❌ Incomplete error handling

### **GCP Simulator (`internal/simulation/gcp_simulator.go`)**
- ❌ Missing drift types: `driftType` not implemented
- ❌ Missing label drift simulation for all resource types
- ❌ Missing rollback functionality
- ❌ Incomplete error handling

## 🔧 **Implementation Steps**

### **Step 1: AWS Simulator Completion**

#### **1.1 Implement Missing Drift Types**

```go
// In internal/simulation/aws_simulator.go

func (s *AWSSimulator) SimulateDrift(ctx context.Context, resource models.Resource, driftType string) (*models.DriftRecord, error) {
    switch driftType {
    case "configuration":
        return s.simulateConfigurationDrift(ctx, resource)
    case "state":
        return s.simulateStateDrift(ctx, resource)
    case "tags":
        return s.simulateTagDrift(ctx, resource)
    case "permissions":
        return s.simulatePermissionDrift(ctx, resource)
    case "networking":
        return s.simulateNetworkingDrift(ctx, resource)
    case "security":
        return s.simulateSecurityDrift(ctx, resource)
    case "cost":
        return s.simulateCostDrift(ctx, resource)
    case "compliance":
        return s.simulateComplianceDrift(ctx, resource)
    default:
        return nil, fmt.Errorf("drift type %s not implemented for AWS", driftType)
    }
}
```

#### **1.2 Implement Tag Drift Simulation**

```go
func (s *AWSSimulator) simulateTagDrift(ctx context.Context, resource models.Resource) (*models.DriftRecord, error) {
    // Get current tags from resource
    currentTags := s.getCurrentTags(resource)
    
    // Simulate tag drift scenarios
    driftScenarios := []struct {
        name        string
        description string
        severity    string
        tags        map[string]string
    }{
        {
            name:        "missing_required_tags",
            description: "Resource missing required tags",
            severity:    "high",
            tags:        s.generateMissingRequiredTags(currentTags),
        },
        {
            name:        "incorrect_tag_values",
            description: "Resource has incorrect tag values",
            severity:    "medium",
            tags:        s.generateIncorrectTagValues(currentTags),
        },
        {
            name:        "extra_tags",
            description: "Resource has extra tags not in desired state",
            severity:    "low",
            tags:        s.generateExtraTags(currentTags),
        },
    }
    
    // Select a random drift scenario
    scenario := driftScenarios[rand.Intn(len(driftScenarios))]
    
    return &models.DriftRecord{
        ID:           generateDriftID(),
        ResourceID:   resource.ID,
        ResourceName: resource.Name,
        ResourceType: resource.Type,
        Provider:     resource.Provider,
        Region:       resource.Region,
        DriftType:    "tags",
        Severity:     scenario.severity,
        Status:       "active",
        DetectedAt:   time.Now(),
        Description:  scenario.description,
        Details: map[string]interface{}{
            "scenario": scenario.name,
            "tags":     scenario.tags,
        },
    }, nil
}
```

#### **1.3 Implement Attribute Change Simulation**

```go
func (s *AWSSimulator) simulateAttributeDrift(ctx context.Context, resource models.Resource) (*models.DriftRecord, error) {
    // Get current attributes from resource
    currentAttributes := resource.Attributes
    
    // Simulate attribute drift scenarios
    driftScenarios := []struct {
        name        string
        description string
        severity    string
        attributes  map[string]interface{}
    }{
        {
            name:        "configuration_change",
            description: "Resource configuration has changed",
            severity:    "medium",
            attributes:  s.generateConfigurationChange(currentAttributes),
        },
        {
            name:        "security_change",
            description: "Resource security settings have changed",
            severity:    "high",
            attributes:  s.generateSecurityChange(currentAttributes),
        },
        {
            name:        "performance_change",
            description: "Resource performance settings have changed",
            severity:    "low",
            attributes:  s.generatePerformanceChange(currentAttributes),
        },
    }
    
    // Select a random drift scenario
    scenario := driftScenarios[rand.Intn(len(driftScenarios))]
    
    return &models.DriftRecord{
        ID:           generateDriftID(),
        ResourceID:   resource.ID,
        ResourceName: resource.Name,
        ResourceType: resource.Type,
        Provider:     resource.Provider,
        Region:       resource.Region,
        DriftType:    "attributes",
        Severity:     scenario.severity,
        Status:       "active",
        DetectedAt:   time.Now(),
        Description:  scenario.description,
        Details: map[string]interface{}{
            "scenario":  scenario.name,
            "attributes": scenario.attributes,
        },
    }, nil
}
```

#### **1.4 Add Helper Methods**

```go
func (s *AWSSimulator) getCurrentTags(resource models.Resource) map[string]string {
    if tags, ok := resource.Tags.(map[string]string); ok {
        return tags
    }
    return make(map[string]string)
}

func (s *AWSSimulator) generateMissingRequiredTags(currentTags map[string]string) map[string]string {
    requiredTags := []string{"Environment", "Project", "Owner", "CostCenter"}
    missingTags := make(map[string]string)
    
    for _, tag := range requiredTags {
        if _, exists := currentTags[tag]; !exists {
            missingTags[tag] = "MISSING"
        }
    }
    
    return missingTags
}

func (s *AWSSimulator) generateIncorrectTagValues(currentTags map[string]string) map[string]string {
    incorrectTags := make(map[string]string)
    
    for key, value := range currentTags {
        if rand.Float32() < 0.3 { // 30% chance of incorrect value
            incorrectTags[key] = value + "_INCORRECT"
        }
    }
    
    return incorrectTags
}

func (s *AWSSimulator) generateExtraTags(currentTags map[string]string) map[string]string {
    extraTags := make(map[string]string)
    
    if rand.Float32() < 0.2 { // 20% chance of extra tags
        extraTags["ExtraTag1"] = "ExtraValue1"
        extraTags["ExtraTag2"] = "ExtraValue2"
    }
    
    return extraTags
}
```

### **Step 2: Azure Simulator Completion**

#### **2.1 Implement Missing Drift Types**

```go
// In internal/simulation/azure_simulator.go

func (s *AzureSimulator) SimulateDrift(ctx context.Context, resource models.Resource, driftType string) (*models.DriftRecord, error) {
    switch driftType {
    case "configuration":
        return s.simulateConfigurationDrift(ctx, resource)
    case "state":
        return s.simulateStateDrift(ctx, resource)
    case "tags":
        return s.simulateTagDrift(ctx, resource)
    case "permissions":
        return s.simulatePermissionDrift(ctx, resource)
    case "networking":
        return s.simulateNetworkingDrift(ctx, resource)
    case "security":
        return s.simulateSecurityDrift(ctx, resource)
    case "cost":
        return s.simulateCostDrift(ctx, resource)
    case "compliance":
        return s.simulateComplianceDrift(ctx, resource)
    default:
        return nil, fmt.Errorf("drift type %s not implemented for Azure", driftType)
    }
}
```

#### **2.2 Implement Tag Drift Simulation**

```go
func (s *AzureSimulator) simulateTagDrift(ctx context.Context, resource models.Resource) (*models.DriftRecord, error) {
    // Get current tags from resource
    currentTags := s.getCurrentTags(resource)
    
    // Simulate tag drift scenarios
    driftScenarios := []struct {
        name        string
        description string
        severity    string
        tags        map[string]string
    }{
        {
            name:        "missing_required_tags",
            description: "Resource missing required tags",
            severity:    "high",
            tags:        s.generateMissingRequiredTags(currentTags),
        },
        {
            name:        "incorrect_tag_values",
            description: "Resource has incorrect tag values",
            severity:    "medium",
            tags:        s.generateIncorrectTagValues(currentTags),
        },
        {
            name:        "extra_tags",
            description: "Resource has extra tags not in desired state",
            severity:    "low",
            tags:        s.generateExtraTags(currentTags),
        },
    }
    
    // Select a random drift scenario
    scenario := driftScenarios[rand.Intn(len(driftScenarios))]
    
    return &models.DriftRecord{
        ID:           generateDriftID(),
        ResourceID:   resource.ID,
        ResourceName: resource.Name,
        ResourceType: resource.Type,
        Provider:     resource.Provider,
        Region:       resource.Region,
        DriftType:    "tags",
        Severity:     scenario.severity,
        Status:       "active",
        DetectedAt:   time.Now(),
        Description:  scenario.description,
        Details: map[string]interface{}{
            "scenario": scenario.name,
            "tags":     scenario.tags,
        },
    }, nil
}
```

### **Step 3: GCP Simulator Completion**

#### **3.1 Implement Missing Drift Types**

```go
// In internal/simulation/gcp_simulator.go

func (s *GCPSimulator) SimulateDrift(ctx context.Context, resource models.Resource, driftType string) (*models.DriftRecord, error) {
    switch driftType {
    case "configuration":
        return s.simulateConfigurationDrift(ctx, resource)
    case "state":
        return s.simulateStateDrift(ctx, resource)
    case "labels":
        return s.simulateLabelDrift(ctx, resource)
    case "permissions":
        return s.simulatePermissionDrift(ctx, resource)
    case "networking":
        return s.simulateNetworkingDrift(ctx, resource)
    case "security":
        return s.simulateSecurityDrift(ctx, resource)
    case "cost":
        return s.simulateCostDrift(ctx, resource)
    case "compliance":
        return s.simulateComplianceDrift(ctx, resource)
    default:
        return nil, fmt.Errorf("drift type %s not implemented for GCP", driftType)
    }
}
```

#### **3.2 Implement Label Drift Simulation**

```go
func (s *GCPSimulator) simulateLabelDrift(ctx context.Context, resource models.Resource) (*models.DriftRecord, error) {
    // Get current labels from resource
    currentLabels := s.getCurrentLabels(resource)
    
    // Simulate label drift scenarios
    driftScenarios := []struct {
        name        string
        description string
        severity    string
        labels      map[string]string
    }{
        {
            name:        "missing_required_labels",
            description: "Resource missing required labels",
            severity:    "high",
            labels:      s.generateMissingRequiredLabels(currentLabels),
        },
        {
            name:        "incorrect_label_values",
            description: "Resource has incorrect label values",
            severity:    "medium",
            labels:      s.generateIncorrectLabelValues(currentLabels),
        },
        {
            name:        "extra_labels",
            description: "Resource has extra labels not in desired state",
            severity:    "low",
            labels:      s.generateExtraLabels(currentLabels),
        },
    }
    
    // Select a random drift scenario
    scenario := driftScenarios[rand.Intn(len(driftScenarios))]
    
    return &models.DriftRecord{
        ID:           generateDriftID(),
        ResourceID:   resource.ID,
        ResourceName: resource.Name,
        ResourceType: resource.Type,
        Provider:     resource.Provider,
        Region:       resource.Region,
        DriftType:    "labels",
        Severity:     scenario.severity,
        Status:       "active",
        DetectedAt:   time.Now(),
        Description:  scenario.description,
        Details: map[string]interface{}{
            "scenario": scenario.name,
            "labels":   scenario.labels,
        },
    }, nil
}
```

#### **3.3 Implement Rollback Functionality**

```go
func (s *GCPSimulator) RollbackDrift(ctx context.Context, data models.DriftRecord) error {
    switch data.ResourceType {
    case "gcp_compute_instance":
        return s.rollbackComputeInstance(ctx, data)
    case "gcp_storage_bucket":
        return s.rollbackStorageBucket(ctx, data)
    case "gcp_network":
        return s.rollbackNetwork(ctx, data)
    case "gcp_firewall_rule":
        return s.rollbackFirewallRule(ctx, data)
    default:
        return fmt.Errorf("rollback not implemented for resource type %s", data.ResourceType)
    }
}

func (s *GCPSimulator) rollbackComputeInstance(ctx context.Context, data models.DriftRecord) error {
    // Implement compute instance rollback logic
    // This would involve:
    // 1. Getting the current state of the instance
    // 2. Comparing with the desired state
    // 3. Making necessary changes to restore desired state
    
    return nil
}

func (s *GCPSimulator) rollbackStorageBucket(ctx context.Context, data models.DriftRecord) error {
    // Implement storage bucket rollback logic
    return nil
}

func (s *GCPSimulator) rollbackNetwork(ctx context.Context, data models.DriftRecord) error {
    // Implement network rollback logic
    return nil
}

func (s *GCPSimulator) rollbackFirewallRule(ctx context.Context, data models.DriftRecord) error {
    // Implement firewall rule rollback logic
    return nil
}
```

### **Step 4: Update Main Simulation Orchestrator**

#### **4.1 Update Drift Simulator**

```go
// In internal/simulation/drift_simulator.go

func (s *DriftSimulator) SimulateDrift(ctx context.Context, resource models.Resource, driftType string) (*models.DriftRecord, error) {
    switch resource.Provider {
    case "aws":
        return s.awsSimulator.SimulateDrift(ctx, resource, driftType)
    case "azure":
        return s.azureSimulator.SimulateDrift(ctx, resource, driftType)
    case "gcp":
        return s.gcpSimulator.SimulateDrift(ctx, resource, driftType)
    default:
        return nil, fmt.Errorf("provider %s not implemented", s.provider)
    }
}
```

### **Step 5: Create Comprehensive Tests**

#### **5.1 Unit Tests for AWS Simulator**

```go
// In tests/unit/simulation/aws_simulator_test.go

func TestAWSSimulator_SimulateDrift(t *testing.T) {
    simulator := NewAWSSimulator()
    
    testCases := []struct {
        name       string
        resource   models.Resource
        driftType  string
        expectErr  bool
    }{
        {
            name: "configuration drift",
            resource: models.Resource{
                ID:       "test-resource",
                Type:     "aws_s3_bucket",
                Provider: "aws",
                Region:   "us-east-1",
            },
            driftType: "configuration",
            expectErr: false,
        },
        {
            name: "tag drift",
            resource: models.Resource{
                ID:       "test-resource",
                Type:     "aws_s3_bucket",
                Provider: "aws",
                Region:   "us-east-1",
            },
            driftType: "tags",
            expectErr: false,
        },
        {
            name: "unsupported drift type",
            resource: models.Resource{
                ID:       "test-resource",
                Type:     "aws_s3_bucket",
                Provider: "aws",
                Region:   "us-east-1",
            },
            driftType: "unsupported",
            expectErr: true,
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            result, err := simulator.SimulateDrift(context.Background(), tc.resource, tc.driftType)
            
            if tc.expectErr {
                assert.Error(t, err)
                assert.Nil(t, result)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, result)
                assert.Equal(t, tc.resource.ID, result.ResourceID)
                assert.Equal(t, tc.driftType, result.DriftType)
            }
        })
    }
}
```

#### **5.2 Integration Tests**

```go
// In tests/integration/simulation/simulation_test.go

func TestSimulationSystem_Integration(t *testing.T) {
    // Test the complete simulation system
    simulator := NewDriftSimulator()
    
    resources := []models.Resource{
        {
            ID:       "aws-resource",
            Type:     "aws_s3_bucket",
            Provider: "aws",
            Region:   "us-east-1",
        },
        {
            ID:       "azure-resource",
            Type:     "azurerm_storage_account",
            Provider: "azure",
            Region:   "eastus",
        },
        {
            ID:       "gcp-resource",
            Type:     "gcp_compute_instance",
            Provider: "gcp",
            Region:   "us-central1",
        },
    }
    
    driftTypes := []string{"configuration", "tags", "security", "compliance"}
    
    for _, resource := range resources {
        for _, driftType := range driftTypes {
            t.Run(fmt.Sprintf("%s_%s_%s", resource.Provider, resource.Type, driftType), func(t *testing.T) {
                result, err := simulator.SimulateDrift(context.Background(), resource, driftType)
                
                assert.NoError(t, err)
                assert.NotNil(t, result)
                assert.Equal(t, resource.ID, result.ResourceID)
                assert.Equal(t, driftType, result.DriftType)
            })
        }
    }
}
```

## 🧪 **Testing Strategy**

### **Unit Tests**
- Test each drift type individually
- Test error handling scenarios
- Test edge cases and boundary conditions
- Achieve 100% code coverage

### **Integration Tests**
- Test complete simulation workflows
- Test cross-provider functionality
- Test error propagation
- Test performance under load

### **End-to-End Tests**
- Test simulation with real cloud resources
- Test simulation result validation
- Test simulation orchestration
- Test simulation monitoring

## 📊 **Success Criteria**

### **Functional Requirements**
- [ ] All drift types implemented for AWS, Azure, and GCP
- [ ] Tag/label drift simulation working for all resource types
- [ ] Attribute change simulation working for all resource types
- [ ] Rollback functionality working for GCP
- [ ] Comprehensive error handling implemented

### **Quality Requirements**
- [ ] 100% test coverage for simulation system
- [ ] All unit tests passing
- [ ] All integration tests passing
- [ ] All end-to-end tests passing
- [ ] GitHub Actions passing

### **Performance Requirements**
- [ ] Simulation operations complete within acceptable time limits
- [ ] Memory usage within acceptable limits
- [ ] No memory leaks detected
- [ ] Concurrent simulation operations working correctly

## 🚀 **Implementation Timeline**

- **Day 1**: AWS Simulator completion
- **Day 2**: Azure Simulator completion
- **Day 3**: GCP Simulator completion and integration
- **Day 4**: Testing and validation
- **Day 5**: Documentation and final verification

## 📝 **Next Steps**

1. **Start with AWS Simulator**: Implement missing drift types
2. **Move to Azure Simulator**: Implement missing drift types
3. **Complete GCP Simulator**: Implement missing drift types and rollback
4. **Update Integration**: Update main simulation orchestrator
5. **Create Tests**: Implement comprehensive test suite
6. **Validate**: Run all tests and verify GitHub Actions pass

## 🎯 **Ready to Begin Phase 1 Implementation!**

This guide provides all the necessary information to complete the simulation system. Follow the steps carefully and ensure thorough testing at each stage.


