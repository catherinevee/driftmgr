# DriftMgr Component Interactions Analysis

## Overview

DriftMgr is a complete cloud infrastructure drift detection and remediation tool with a modular architecture designed for extensibility and maintainability. The system consists of multiple interconnected components that work together to provide end-to-end drift management capabilities.

## Architecture Overview

```

 Main Entry Web Dashboard CLI Client
 (main.go) (web/) (cmd/client/)

 API Layer
 (HTTP/WebSocket)

 Core Components

 Discovery Analysis Remediation
 (internal/ (internal/ (internal/
 discovery/) analysis/) remediation/)

 Support Components

 Models Cache Config
 (internal/ (internal/ (internal/
 models/) cache/) config/)

```

## Core Component Interactions

### 1. Main Entry Point (`main.go`)

The main entry point orchestrates the overall application flow:

**Key Responsibilities:**
- Server lifecycle management (start/stop)
- Client-server communication coordination
- Resource deletion command handling
- Interactive resource selection

**Component Interactions:**
```go
// Server management
if !isServerRunning() {
 startServer(exeDir)
}

// Client execution
cmd := exec.Command(clientPath, os.Args[1:]...)
cmd.Run()

// Resource deletion with dependency management
handleResourceDeletion(args)
```

**Key Features:**
- Automatic server startup if not running
- Cross-platform executable detection
- Interactive resource discovery and selection
- Dependency-aware resource deletion

### 2. Discovery Component (`internal/discovery/`)

The discovery component is responsible for finding and cataloging cloud resources across multiple providers.

**Key Components:**

#### Enhanced Discoverer (`enhanced_discovery.go`)
```go
type EnhancedDiscoverer struct {
 config *config.Config
 cache *cache.DiscoveryCache
 plugins map[string]*DiscoveryPlugin
 hierarchy *ResourceHierarchy
 filters *DiscoveryFilter
 progressTracker *ProgressTracker
 visualizer *DiscoveryVisualizer
 errorHandler *ErrorHandler
 errorReporting *EnhancedErrorReporting
 advancedQuery *AdvancedQuery
 realTimeMonitor *RealTimeMonitor
 sdkIntegration *SDKIntegration
}
```

**Provider Support:**
- **AWS**: EC2, RDS, Lambda, EKS, ECS, S3, IAM, CloudFormation, ElastiCache, SQS, SNS, DynamoDB, AutoScaling, WAF, Shield, Config, GuardDuty, CloudFront, API Gateway, Glue, Redshift, Elasticsearch, CloudWatch, Systems Manager, Step Functions
- **Azure**: VMs, Storage Accounts, SQL Databases, Web Apps, Virtual Networks, Load Balancers, Key Vaults, Resource Groups, Functions, Logic Apps, Event Hubs, Service Bus, CosmosDB, Data Factory, Synapse Analytics, Application Insights, Policy, Bastion
- **GCP**: Compute Instances, Storage Buckets, GKE Clusters, Cloud SQL, VPC Networks, Cloud Functions, Cloud Run, Cloud Build, Cloud Pub/Sub, BigQuery, Cloud Spanner, Cloud Firestore, Cloud Armor, Cloud Monitoring, Cloud Logging

**Key Interactions:**
```go
// Multi-provider discovery
func (ed *EnhancedDiscoverer) DiscoverAllResourcesEnhanced(ctx context.Context, providers []string, regions []string) ([]models.Resource, error) {
 // Check cache first
 if cached, found := ed.cache.Get(cacheKey); found {
 return cached.([]models.Resource), nil
 }

 // Discover by provider and region
 for _, provider := range providers {
 for _, region := range regions {
 resources, err := ed.discoverProviderRegionEnhanced(ctx, provider, region)
 // ...
 }
 }

 // Apply filters and build hierarchy
 filteredResources := ed.applyFilters(allResources)
 ed.buildResourceHierarchy(filteredResources)

 // Cache results
 ed.cache.Set(cacheKey, filteredResources, ed.config.Discovery.CacheTTL)
}
```

### 3. Analysis Component (`internal/analysis/`)

The analysis component performs drift detection and analysis between expected and actual resource states.

**Key Components:**

#### Drift Analyzer (`analysis.go`)
```go
type Analyzer struct {
 config map[string]interface{}
}

type DriftAnalysis struct {
 ResourceID string
 ResourceType string
 Provider string
 Region string
 DriftDetected bool
 DriftType string
 Changes []DriftChange
 Severity string
 Timestamp time.Time
 Metadata map[string]interface{}
}
```

**Analysis Process:**
```go
func (a *Analyzer) AnalyzeResource(ctx context.Context, resource models.Resource, expectedState map[string]interface{}) (*DriftAnalysis, error) {
 // Compare actual vs expected state
 changes := a.detectChanges(resource, expectedState)

 if len(changes) > 0 {
 analysis.DriftDetected = true
 analysis.Changes = changes
 analysis.DriftType = a.determineDriftType(changes)
 analysis.Severity = a.calculateSeverity(changes)
 }
}
```

#### Enhanced Drift Detector (`internal/drift/enhanced_detector.go`)
```go
type AttributeDriftDetector struct {
 SensitiveFields map[string]bool
 IgnoreFields map[string]bool
 Thresholds DriftThresholds
 CustomComparators map[string]AttributeComparator
 SeverityRules []SeverityRule
}
```

**Drift Detection Process:**
```go
func (d *AttributeDriftDetector) DetectDrift(stateResources, liveResources []models.Resource) models.AnalysisResult {
 // Detect missing resources (in state but not in live)
 for id, stateResource := range stateMap {
 if _, exists := liveMap[id]; !exists {
 driftResult := d.createDriftResult(stateResource, "missing", "high", "Resource exists in Terraform state but not in live infrastructure")
 driftResults = append(driftResults, driftResult)
 }
 }

 // Detect extra resources (in live but not in state)
 for id, liveResource := range liveMap {
 if _, exists := stateMap[id]; !exists {
 driftResult := d.createDriftResult(liveResource, "extra", "medium", "Resource exists in live infrastructure but not in Terraform state")
 driftResults = append(driftResults, driftResult)
 }
 }

 // Detect modified resources (attribute-level drift)
 for id, stateResource := range stateMap {
 if liveResource, exists := liveMap[id]; exists {
 attributeDrifts := d.detectAttributeDrift(stateResource, liveResource)
 if len(attributeDrifts) > 0 {
 driftResult := d.createAttributeDriftResult(stateResource, attributeDrifts)
 driftResults = append(driftResults, driftResult)
 }
 }
 }
}
```

### 4. Remediation Component (`internal/remediation/`)

The remediation component handles the correction of detected drifts through various strategies.

**Key Components:**

#### Terraform Remediation Engine (`terraform_remediation.go`)
```go
type TerraformRemediationEngine struct {
 workingDir string
 terraformPath string
}

type TerraformRemediationPlan struct {
 ID string
 Description string
 Resources []TerraformRemediationResource
 PlanOutput string
 PlanFile string
 CreatedAt time.Time
 Status string
}
```

**Remediation Process:**
```go
func (tre *TerraformRemediationEngine) GenerateTerraformConfiguration(drifts []models.DriftResult) (*TerraformRemediationPlan, error) {
 // Group drifts by action type
 createResources := []models.DriftResult{}
 updateResources := []models.DriftResult{}
 deleteResources := []models.DriftResult{}
 importResources := []models.DriftResult{}

 for _, drift := range drifts {
 switch drift.DriftType {
 case "missing":
 createResources = append(createResources, drift)
 case "modified":
 updateResources = append(updateResources, drift)
 case "extra":
 deleteResources = append(deleteResources, drift)
 case "unmanaged":
 importResources = append(importResources, drift)
 }
 }

 // Generate resources in dependency order
 allResources := []TerraformRemediationResource{}

 // Handle imports first
 for _, drift := range importResources {
 resource := tre.createImportResource(drift)
 allResources = append(allResources, resource)
 }

 // Handle creates (dependencies first)
 createResources = tre.sortByDependencies(createResources)
 for _, drift := range createResources {
 resource := tre.createResource(drift)
 allResources = append(allResources, resource)
 }

 // Handle updates and deletes
 // ...
}
```

### 5. Web Dashboard (`web/`)

The web dashboard provides a real-time interface for monitoring and managing drift.

**Key Components:**

#### Dashboard (`dashboard.go`)
```go
type Dashboard struct {
 router *mux.Router
 discoveryEngine *discovery.EnhancedDiscoveryEngine
 remediationEngine *remediation.AdvancedRemediationEngine
 upgrader websocket.Upgrader
 clients map[*websocket.Conn]bool
 clientsMutex sync.RWMutex
 config *DashboardConfig
}
```

**API Endpoints:**
```go
func (d *Dashboard) setupRoutes() {
 // API routes
 api := d.router.PathPrefix("/api/v1").Subrouter()
 api.HandleFunc("/resources", d.getResources).Methods("GET")
 api.HandleFunc("/drift", d.getDrift).Methods("GET")
 api.HandleFunc("/remediate", d.remediateDrift).Methods("POST")
 api.HandleFunc("/costs", d.getCosts).Methods("GET")
 api.HandleFunc("/security", d.getSecurity).Methods("GET")
 api.HandleFunc("/compliance", d.getCompliance).Methods("GET")
 api.HandleFunc("/metrics", d.getMetrics).Methods("GET")

 // WebSocket for real-time updates
 d.router.HandleFunc("/ws", d.handleWebSocket)
}
```

**Real-time Updates:**
```go
func (d *Dashboard) startRealTimeUpdates() {
 ticker := time.NewTicker(d.config.RefreshInterval)
 defer ticker.Stop()

 for range ticker.C {
 d.broadcastUpdate()
 }
}

func (d *Dashboard) broadcastUpdate() {
 update := DashboardUpdate{
 Timestamp: time.Now(),
 Type: "update",
 Data: d.getDashboardData(),
 }

 updateJSON, err := json.Marshal(update)
 if err != nil {
 return
 }

 for client := range d.clients {
 err := client.WriteMessage(websocket.TextMessage, updateJSON)
 if err != nil {
 client.Close()
 delete(d.clients, client)
 }
 }
}
```

### 6. CLI Client (`cmd/driftmgr-client/`)

The CLI client provides command-line interface for interacting with the driftmgr system.

**Key Components:**

#### DriftMgr Client (`main.go`)
```go
type DriftMgrClient struct {
 httpClient *http.Client
 baseURL string
}

type InteractiveShell struct {
 client *DriftMgrClient
 reader *bufio.Reader
 history []string
 historyIndex int
 historyMutex sync.RWMutex
 enhancedReader *EnhancedInputReader
 regionManager *regions.RegionManager
}
```

**Key Features:**
- Interactive shell with command history
- Tab completion for commands and resources
- Color-coded output for better readability
- Real-time progress tracking
- Batch operations support

## Data Flow and Component Interactions

### 1. Resource Discovery Flow

```
User Request → CLI Client → Web Dashboard → Discovery Engine → Cloud Providers
 ↓ ↓ ↓ ↓ ↓
Cache Check ← API Response ← Discovery Results ← Resource Data ← Cloud APIs
```

**Detailed Flow:**
1. User initiates discovery via CLI or web interface
2. Request routed to discovery engine
3. Cache checked for existing results
4. If cache miss, discovery engine queries cloud providers
5. Resources filtered and processed
6. Results cached and returned to user
7. Real-time updates broadcast via WebSocket

### 2. Drift Analysis Flow

```
State Files → Analysis Engine → Live Resources → Drift Detection → Results
 ↓ ↓ ↓ ↓ ↓
Terraform ← State Parser ← Cloud APIs ← Resource Comparison ← Drift Report
```

**Detailed Flow:**
1. Terraform state files parsed and loaded
2. Live resources discovered from cloud providers
3. State and live resources compared using configurable rules
4. Drifts categorized by type and severity
5. Analysis results cached and reported
6. Remediation plans generated if requested

### 3. Remediation Flow

```
Drift Results → Remediation Engine → Terraform Plans → Execution → Verification
 ↓ ↓ ↓ ↓ ↓
User Approval ← Safety Checks ← Plan Generation ← Apply Changes ← State Update
```

**Detailed Flow:**
1. Drift results analyzed for remediation strategies
2. Terraform configuration generated
3. Safety checks performed (dry-run, validation)
4. User approval requested if required
5. Changes applied to infrastructure
6. Results verified and reported

## Key Integration Points

### 1. Cache Integration
All components use a shared caching layer for performance optimization:
```go
type DiscoveryCache struct {
 cache map[string]interface{}
 ttl time.Duration
 maxSize int
 mutex sync.RWMutex
}
```

### 2. Configuration Management
Centralized configuration management across all components:
```go
type Config struct {
 Discovery DiscoveryConfig
 Analysis AnalysisConfig
 Remediation RemediationConfig
 Cache CacheConfig
 Security SecurityConfig
}
```

### 3. Error Handling
Complete error handling and reporting across all components:
```go
type ErrorHandler struct {
 retryConfig *RetryConfig
 errorLogger *ErrorLogger
 fallback *FallbackHandler
}
```

### 4. Monitoring and Metrics
Real-time monitoring and metrics collection:
```go
type MetricsCollector struct {
 discoveryMetrics *DiscoveryMetrics
 analysisMetrics *AnalysisMetrics
 remediationMetrics *RemediationMetrics
 performanceMetrics *PerformanceMetrics
}
```

## Performance Considerations

### 1. Parallel Processing
- Discovery operations parallelized across providers and regions
- Analysis operations batched for efficiency
- Remediation operations queued and processed in parallel

### 2. Caching Strategy
- Multi-level caching (memory, disk, distributed)
- TTL-based cache invalidation
- Cache warming for frequently accessed data

### 3. Resource Management
- Connection pooling for cloud API calls
- Memory-efficient resource representation
- Garbage collection optimization

## Security Features

### 1. Credential Management
- Secure credential storage and rotation
- Multi-provider authentication support
- Least privilege access principles

### 2. Data Protection
- Sensitive data encryption at rest and in transit
- Audit logging for all operations
- Compliance with security standards

### 3. Access Control
- Role-based access control (RBAC)
- API authentication and authorization
- Secure communication channels

## Extensibility Points

### 1. Plugin System
- Discovery plugins for new cloud providers
- Analysis plugins for custom drift detection rules
- Remediation plugins for custom correction strategies

### 2. Custom Comparators
- Configurable comparison logic for complex attributes
- Custom severity rules for drift classification
- Extensible filtering and sorting capabilities

### 3. Integration APIs
- RESTful API for external integrations
- WebSocket API for real-time updates
- Event-driven architecture for notifications

## Conclusion

The DriftMgr system demonstrates a well-architected, modular design with clear separation of concerns and strong component interactions. The system provides complete drift detection and remediation capabilities while maintaining extensibility, performance, and security. The component interactions are designed to be loosely coupled yet highly cohesive, enabling easy maintenance and future enhancements.
