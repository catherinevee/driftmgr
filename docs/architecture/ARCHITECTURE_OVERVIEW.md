# DriftMgr Architecture Overview

## Executive Summary

DriftMgr v2.0 implements a unified service layer architecture that provides consistent behavior across CLI and web interfaces. This document provides a comprehensive overview of the system architecture, design decisions, and implementation details.

## System Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                           User Layer                                │
├──────────────┬────────────────────────┬────────────────────────────┤
│     CLI      │      Web GUI           │         API Clients        │
└──────┬───────┴────────┬───────────────┴────────────┬───────────────┘
       │                │                            │
       ▼                ▼                            ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         API Gateway Layer                           │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  REST API  │  WebSocket  │  GraphQL (Future)  │  gRPC      │   │
│  └─────────────────────────────────────────────────────────────┘   │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Service Layer (Core)                           │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ Discovery │ State │ Drift │ Remediation │ Workflow │ Config │   │
│  └─────────────────────────────────────────────────────────────┘   │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
       ┌───────────────────────┼───────────────────────┐
       ▼                       ▼                       ▼
┌──────────────┐      ┌──────────────┐      ┌──────────────┐
│  Event Bus   │      │  Job Queue   │      │    Cache     │
└──────────────┘      └──────────────┘      └──────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Infrastructure Layer                             │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  Database  │  Storage  │  Monitoring  │  Logging  │  Audit  │   │
│  └─────────────────────────────────────────────────────────────┘   │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Provider Layer                                 │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │   AWS   │   Azure   │   GCP   │   DigitalOcean   │   K8s    │   │
│  └─────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Service Layer

The service layer is the heart of DriftMgr, providing:

#### Discovery Service
- **Purpose**: Unified resource discovery across all cloud providers
- **Capabilities**:
  - Multi-provider discovery orchestration
  - Incremental discovery with caching
  - Resource relationship mapping
  - Progress tracking and reporting
  - Parallel discovery execution

#### State Service
- **Purpose**: Terraform state file management and analysis
- **Capabilities**:
  - State file discovery (local, S3, Azure Blob, GCS)
  - State parsing and validation
  - State comparison and diffing
  - State migration support
  - Terragrunt integration

#### Drift Service
- **Purpose**: Configuration drift detection and analysis
- **Capabilities**:
  - State-based drift detection
  - Provider-based drift detection
  - Compliance scoring
  - Drift trend analysis
  - Custom drift policies

#### Remediation Service
- **Purpose**: Automated and manual drift remediation
- **Capabilities**:
  - Remediation plan generation
  - Dry-run execution
  - Approval workflows
  - Rollback support
  - Audit trail generation

### 2. Event-Driven Architecture

#### Event Bus
- **Technology**: In-memory pub/sub with persistence option
- **Event Types**:
  ```
  - Discovery: Started, Progress, Completed, Failed
  - State: Imported, Analyzed, Compared, Deleted
  - Drift: DetectionStarted, DriftFound, DetectionCompleted
  - Remediation: PlanCreated, Approved, Executed, RolledBack
  ```
- **Benefits**:
  - Real-time UI updates
  - Decoupled components
  - Audit trail generation
  - External integrations

#### Event Flow Example
```
User Action → Service → Event Published → Multiple Subscribers
                ↓              ↓                    ↓
            Database      WebSocket          Audit Logger
                          (UI Update)
```

### 3. Asynchronous Processing

#### Job Queue
- **Implementation**: Priority-based queue with worker pool
- **Job Types**:
  - Discovery jobs (high priority)
  - Drift detection jobs (medium priority)
  - Remediation jobs (low priority, requires approval)
  - Report generation jobs (low priority)
- **Features**:
  - Retry logic with exponential backoff
  - Job persistence for recovery
  - Progress tracking
  - Cancellation support

#### Worker Pool Configuration
```go
type WorkerConfig struct {
    MaxWorkers     int           // Maximum concurrent workers
    QueueSize      int           // Maximum queue size
    RetryAttempts  int           // Number of retry attempts
    RetryDelay     time.Duration // Initial retry delay
    Timeout        time.Duration // Job timeout
}
```

### 4. Caching Strategy

#### Multi-Level Cache
```
┌─────────────────────────────────────┐
│         L1: In-Memory Cache         │  ← Hot data (< 5 min)
├─────────────────────────────────────┤
│         L2: Local Disk Cache        │  ← Warm data (< 1 hour)
├─────────────────────────────────────┤
│         L3: Database Cache          │  ← Cold data (persistent)
└─────────────────────────────────────┘
```

#### Cache Invalidation
- **TTL-based**: Automatic expiration
- **Event-based**: Invalidate on state changes
- **Manual**: API endpoint for cache clearing

## Design Patterns

### 1. Command Query Responsibility Segregation (CQRS)

#### Commands (Write Operations)
```go
type Command interface {
    Execute(ctx context.Context) error
    Validate() error
    Rollback(ctx context.Context) error
}
```

#### Queries (Read Operations)
```go
type Query interface {
    Execute(ctx context.Context) (interface{}, error)
    Cache() bool
    CacheDuration() time.Duration
}
```

### 2. Repository Pattern

#### Provider Abstraction
```go
type CloudProvider interface {
    DiscoverResources(ctx context.Context, config Config) ([]Resource, error)
    GetResource(ctx context.Context, id string) (*Resource, error)
    CreateResource(ctx context.Context, spec ResourceSpec) (*Resource, error)
    UpdateResource(ctx context.Context, id string, spec ResourceSpec) error
    DeleteResource(ctx context.Context, id string) error
}
```

### 3. Circuit Breaker Pattern

#### Implementation
```go
type CircuitBreaker struct {
    MaxFailures      int
    ResetTimeout     time.Duration
    OnStateChange    func(from, to State)
}
```

#### States
- **Closed**: Normal operation
- **Open**: Failing, reject requests
- **Half-Open**: Testing recovery

### 4. Observer Pattern

#### Event Subscription
```go
eventBus.Subscribe("discovery.*", func(event Event) {
    // Handle all discovery events
})

eventBus.SubscribeOnce("remediation.completed", func(event Event) {
    // Handle single remediation completion
})
```

## Data Flow

### 1. Discovery Flow

```
1. User Request
   ↓
2. API Handler validates request
   ↓
3. DiscoveryService creates job
   ↓
4. Job Queue schedules execution
   ↓
5. Worker executes discovery
   ↓
6. Provider APIs called in parallel
   ↓
7. Results aggregated and cached
   ↓
8. Event published
   ↓
9. UI updated via WebSocket
```

### 2. Drift Detection Flow

```
1. Scheduled or manual trigger
   ↓
2. DriftService loads state files
   ↓
3. Current state discovered
   ↓
4. State comparison executed
   ↓
5. Drift items identified
   ↓
6. Compliance score calculated
   ↓
7. Report generated
   ↓
8. Notifications sent
```

### 3. Remediation Flow

```
1. Drift report reviewed
   ↓
2. Remediation plan created
   ↓
3. Safety checks performed
   ↓
4. Approval requested (if needed)
   ↓
5. Snapshot created for rollback
   ↓
6. Actions executed in order
   ↓
7. Validation performed
   ↓
8. Audit trail updated
```

## Security Architecture

### 1. Authentication & Authorization

#### RBAC Model
```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│    Users     │────▶│    Roles     │────▶│ Permissions  │
└──────────────┘     └──────────────┘     └──────────────┘
```

#### Predefined Roles
- **Admin**: Full system access
- **Operator**: Execute operations
- **Viewer**: Read-only access
- **Approver**: Approve remediation plans

### 2. Secrets Management

#### Integration Points
- HashiCorp Vault for dynamic secrets
- AWS Secrets Manager
- Azure Key Vault
- Environment variables (development)

#### Secret Rotation
- Automatic rotation support
- Zero-downtime credential updates
- Audit trail for access

### 3. Audit Logging

#### Compliance Modes
- **SOC2**: Security and availability logging
- **HIPAA**: PHI access tracking
- **PCI-DSS**: Payment card data protection
- **Custom**: User-defined policies

#### Audit Event Structure
```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "user": "admin@example.com",
  "action": "remediation.execute",
  "resource": "aws.ec2.instance.i-12345",
  "result": "success",
  "metadata": {
    "ip": "192.168.1.1",
    "session": "sess-12345",
    "changes": ["tag.updated", "security_group.modified"]
  }
}
```

## Performance Considerations

### 1. Scalability

#### Horizontal Scaling
- Stateless service design
- Load balancer ready
- Shared cache (Redis)
- Distributed job queue

#### Vertical Scaling
- Worker pool sizing
- Memory management
- Connection pooling

### 2. Optimization Techniques

#### Resource Discovery
- Parallel API calls
- Incremental discovery
- Smart caching
- Pagination support

#### State Processing
- Streaming parser for large files
- Selective field loading
- Compression support

### 3. Monitoring & Metrics

#### Key Metrics
```
- Discovery duration by provider
- Cache hit ratio
- Job queue depth
- API response times
- Error rates by operation
- Resource count by type
```

#### Health Checks
```
GET /health
{
  "status": "healthy",
  "checks": {
    "database": "ok",
    "cache": "ok",
    "providers": {
      "aws": "ok",
      "azure": "degraded",
      "gcp": "ok"
    }
  }
}
```

## Deployment Architecture

### 1. Container Deployment

#### Docker Compose
```yaml
version: '3.8'
services:
  driftmgr:
    image: catherinevee/driftmgr:latest
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://db:5432/driftmgr
      - CACHE_URL=redis://cache:6379
    depends_on:
      - database
      - cache
```

### 2. Kubernetes Deployment

#### Components
- **Deployment**: Main application pods
- **Service**: Load balancing
- **ConfigMap**: Configuration
- **Secret**: Credentials
- **HPA**: Auto-scaling
- **PDB**: Pod disruption budget

### 3. Cloud-Native Deployment

#### AWS Architecture
```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│     ALB      │────▶│   ECS/EKS    │────▶│     RDS      │
└──────────────┘     └──────────────┘     └──────────────┘
                            │
                            ▼
                     ┌──────────────┐
                     │  ElastiCache │
                     └──────────────┘
```

## Integration Points

### 1. CI/CD Integration

#### Pipeline Stages
1. **Build**: Compile and test
2. **Scan**: Security and compliance
3. **Deploy**: Staging environment
4. **Test**: Integration tests
5. **Promote**: Production deployment

### 2. Monitoring Integration

#### Supported Platforms
- Prometheus + Grafana
- Datadog
- New Relic
- CloudWatch
- Azure Monitor

### 3. Notification Channels

#### Supported Integrations
- Slack
- Microsoft Teams
- Email (SMTP)
- PagerDuty
- Webhooks

## Future Roadmap

### Phase 1: Enhanced Discovery (Q1 2024)
- Kubernetes resource discovery
- Custom resource definitions
- Cross-region aggregation

### Phase 2: Advanced Remediation (Q2 2024)
- ML-based remediation suggestions
- Policy-driven auto-remediation
- Change impact analysis

### Phase 3: Enterprise Features (Q3 2024)
- Multi-tenancy support
- Cost optimization recommendations
- Compliance reporting

### Phase 4: Platform Expansion (Q4 2024)
- Oracle Cloud support
- IBM Cloud support
- On-premises infrastructure

## Best Practices

### 1. Development

#### Code Organization
- Single responsibility principle
- Dependency injection
- Interface-based design
- Comprehensive testing

#### Testing Strategy
- Unit tests: > 80% coverage
- Integration tests: Critical paths
- E2E tests: User workflows
- Performance tests: Load testing

### 2. Operations

#### Monitoring
- Set up alerts for critical metrics
- Regular health checks
- Performance baselines
- Capacity planning

#### Maintenance
- Regular dependency updates
- Security patches
- Database maintenance
- Cache optimization

### 3. Security

#### Best Practices
- Principle of least privilege
- Defense in depth
- Regular security audits
- Incident response plan

## Conclusion

DriftMgr v2.0's architecture provides a robust, scalable, and maintainable foundation for multi-cloud infrastructure management. The unified service layer ensures consistency across all interfaces while the event-driven architecture enables real-time updates and extensibility.

For implementation details, refer to:
- [Service Layer Architecture](./SERVICE_LAYER_ARCHITECTURE.md)
- [API Documentation](../api/README.md)
- [Deployment Guide](../guides/DEPLOYMENT.md)