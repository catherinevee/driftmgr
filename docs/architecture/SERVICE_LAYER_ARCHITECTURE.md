# DriftMgr Service Layer Architecture

## Overview

DriftMgr v2.0 introduces a unified service layer architecture that ensures consistency between CLI and web interfaces. This document describes the architecture, its components, and the benefits it provides.

## Architecture Diagram

```
┌──────────────────────────────────────────────────────────────────┐
│                         User Interfaces                          │
├────────────────────────┬─────────────────────────────────────────┤
│       CLI              │              Web GUI                    │
│  ┌─────────────┐      │    ┌────────────────────────────┐      │
│  │ Commands    │      │    │  React/Alpine.js Frontend  │      │
│  │ - discover  │      │    │  - Dashboard               │      │
│  │ - drift     │      │    │  - State Explorer          │      │
│  │ - remediate │      │    │  - Drift Viewer            │      │
│  │ - workflow  │      │    │  - Resource Manager        │      │
│  └──────┬──────┘      │    └────────────┬───────────────┘      │
│         │              │                 │                       │
└─────────┼──────────────┴─────────────────┼──────────────────────┘
          │                                 │
          ▼                                 ▼
┌──────────────────────────────────────────────────────────────────┐
│                      API Layer (REST + WebSocket)                │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  Handlers: Discovery | State | Drift | Remediation         │ │
│  └────────────────────────────────────────────────────────────┘ │
└───────────────────────────┬──────────────────────────────────────┘
                            │
                            ▼
┌──────────────────────────────────────────────────────────────────┐
│                         Service Layer                            │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  DiscoveryService  │  StateService  │  DriftService        │ │
│  │  RemediationService │  Service Manager                      │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                  │
│  Key Features:                                                   │
│  • Single source of truth for business logic                    │
│  • Consistent behavior across interfaces                        │
│  • Centralized validation and error handling                    │
│  • Unified caching strategy                                     │
└───────────────────────┬──────────────────────────────────────────┘
                        │
          ┌─────────────┼─────────────┐
          ▼             ▼             ▼
┌──────────────┐ ┌──────────┐ ┌──────────────┐
│  Event Bus   │ │Job Queue │ │    Cache     │
│              │ │          │ │              │
│ • Real-time  │ │ • Async  │ │ • Unified    │
│   updates    │ │   jobs   │ │   caching    │
│ • WebSocket  │ │ • Retry  │ │ • TTL mgmt   │
│   sync       │ │   logic  │ │              │
└──────────────┘ └──────────┘ └──────────────┘
                        │
                        ▼
┌──────────────────────────────────────────────────────────────────┐
│                    Provider & Storage Layer                      │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  AWS  │  Azure  │  GCP  │  DigitalOcean  │  Terraform     │ │
│  └────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────┘
```

## Component Details

### 1. Service Layer (`internal/services/`)

The service layer is the heart of the architecture, providing:

#### DiscoveryService
- **Purpose**: Manages resource discovery across all providers
- **Key Methods**:
  - `StartDiscovery()` - Initiates discovery (sync or async)
  - `GetDiscoveryStatus()` - Returns job status
  - `GetCachedResources()` - Returns cached resources
- **Features**:
  - Provider abstraction
  - Progress tracking
  - Result caching

#### StateService
- **Purpose**: Handles Terraform state file operations
- **Key Methods**:
  - `DiscoverStateFiles()` - Finds state files
  - `ImportStateFile()` - Imports a state file
  - `AnalyzeStateFiles()` - Analyzes state content
  - `CompareStateFiles()` - Compares two states
- **Features**:
  - Multi-backend support
  - Terragrunt compatibility
  - State visualization data

#### DriftService
- **Purpose**: Detects configuration drift
- **Key Methods**:
  - `StartDriftDetection()` - Initiates drift detection
  - `GetDriftReport()` - Returns drift analysis
- **Features**:
  - State-based drift detection
  - Provider-based drift detection
  - Compliance scoring

#### RemediationService
- **Purpose**: Executes remediation actions
- **Key Methods**:
  - `StartRemediation()` - Begins remediation
  - `CreateRemediationPlan()` - Plans remediation
  - `ApproveRemediation()` - Approves a plan
- **Features**:
  - Dry-run mode
  - Approval workflows
  - Rollback support

### 2. Event Bus (`internal/events/`)

The event bus enables real-time communication:

- **Event Types**:
  - Discovery events (started, progress, completed)
  - State events (imported, analyzed, deleted)
  - Drift events (detection started/completed)
  - Remediation events (started, completed, failed)

- **Subscription Model**:
  ```go
  eventBus.SubscribeToType(events.DiscoveryCompleted, handler)
  ```

- **Benefits**:
  - Real-time UI updates
  - Audit trail generation
  - Decoupled components

### 3. Job Queue (`internal/jobs/`)

Manages long-running operations:

- **Job Types**:
  - Discovery jobs
  - Drift detection jobs
  - Remediation jobs
  - State analysis jobs

- **Features**:
  - Priority-based execution
  - Retry logic with exponential backoff
  - Job persistence
  - Progress tracking

### 4. CQRS Pattern (`internal/cqrs/`)

Separates commands from queries:

#### Commands (Write Operations)
- `StartDiscoveryCommand`
- `ImportStateCommand`
- `DetectDriftCommand`
- `StartRemediationCommand`

#### Queries (Read Operations)
- `GetDiscoveryStatusQuery`
- `GetStateFileQuery`
- `GetDriftReportQuery`
- `GetRemediationPlanQuery`

### 5. Unified API Handlers (`internal/api/handlers/`)

Thin handlers that delegate to services:

```go
// Example: Discovery Handler
func (h *DiscoveryHandler) StartDiscovery(w http.ResponseWriter, r *http.Request) {
    var req services.DiscoveryRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    // Delegate to service
    response, err := h.service.StartDiscovery(r.Context(), req)
    
    json.NewEncoder(w).Encode(response)
}
```

## Data Flow Examples

### Example 1: Resource Discovery

```
1. User initiates discovery (CLI or Web)
   ↓
2. Request reaches appropriate handler
   ↓
3. Handler calls DiscoveryService.StartDiscovery()
   ↓
4. Service creates job and publishes DiscoveryStarted event
   ↓
5. Job queue processes discovery asynchronously
   ↓
6. Progress events update all connected clients
   ↓
7. Completion event triggers UI updates and caching
```

### Example 2: Drift Detection with Remediation

```
1. User requests drift detection
   ↓
2. DriftService analyzes state vs reality
   ↓
3. Drift report generated with remediation options
   ↓
4. User approves remediation plan
   ↓
5. RemediationService executes approved actions
   ↓
6. Events update UI in real-time
   ↓
7. Rollback snapshot created for safety
```

## Benefits of the Architecture

### 1. Consistency
- Same business logic for CLI and Web
- Unified validation rules
- Consistent error handling

### 2. Scalability
- Async job processing
- Horizontal scaling capability
- Efficient caching strategy

### 3. Maintainability
- Clear separation of concerns
- Testable components
- Single source of truth

### 4. Real-time Updates
- WebSocket integration
- Event-driven updates
- Live progress tracking

### 5. Reliability
- Circuit breakers for external calls
- Retry logic with backoff
- Graceful degradation

## Migration from v1.0

### What Changed

| Component | v1.0 | v2.0 |
|-----------|------|------|
| Business Logic | Duplicated in CLI and API | Unified in Service Layer |
| API Handlers | Contains business logic | Thin delegation layer |
| CLI Commands | Direct provider calls | Service layer calls |
| Caching | Multiple implementations | Unified cache service |
| Events | Ad-hoc notifications | Centralized event bus |
| Jobs | Synchronous execution | Async job queue |

### Migration Steps

1. **Update imports** - Use service layer instead of direct provider calls
2. **Refactor handlers** - Remove business logic, use services
3. **Add event subscriptions** - Subscribe to relevant events
4. **Implement job handlers** - For long-running operations
5. **Update tests** - Test services instead of handlers

## Best Practices

### 1. Service Design
- Keep services focused on a single domain
- Use dependency injection
- Return consistent response types

### 2. Event Usage
- Publish events for significant state changes
- Keep event payloads small
- Use event types consistently

### 3. Job Queue
- Use for operations > 5 seconds
- Implement proper retry logic
- Track progress granularly

### 4. Error Handling
- Use typed errors
- Provide actionable error messages
- Log errors at appropriate levels

## Future Enhancements

### Planned Features
- GraphQL API layer
- Plugin architecture for custom providers
- Distributed job processing
- Advanced caching with Redis
- Metrics and observability integration

### Extensibility Points
- Custom event handlers
- Provider plugins
- Workflow definitions
- Policy engines

## Conclusion

The unified service layer architecture in DriftMgr v2.0 provides a robust foundation for infrastructure management. By centralizing business logic and providing consistent interfaces, it ensures reliable and maintainable operations across all user interfaces.