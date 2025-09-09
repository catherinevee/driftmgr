# Enterprise Features

DriftMgr includes enterprise-grade features for production deployments.

## Audit Logging

### Overview
Comprehensive audit logging system that tracks all operations, user actions, and system events.

### Features
- File-based audit logging with automatic rotation
- Compliance modes (SOC2, HIPAA, PCI-DSS)
- Configurable retention policies
- Export to multiple formats (JSON, CSV, SIEM/CEF)

### Usage

```go
import "github.com/catherinevee/driftmgr/internal/audit"

// Create audit logger
logger, err := audit.NewFileLogger("/var/log/driftmgr/audit")

// For compliance
complianceLogger, err := audit.NewComplianceLogger("/var/log/driftmgr/audit", "SOC2")

// Log an event
event := &audit.AuditEvent{
    EventType: audit.EventTypeDiscovery,
    Severity:  audit.SeverityInfo,
    User:      "admin@example.com",
    Service:   "discovery",
    Action:    "ListResources",
    Resource:  "aws:ec2:instances",
    Provider:  "aws",
    Region:    "us-east-1",
    Result:    "success",
}
logger.Log(ctx, event)

// Query audit logs
filter := audit.QueryFilter{
    StartTime:  time.Now().AddDate(0, -1, 0),
    EndTime:    time.Now(),
    EventTypes: []audit.EventType{audit.EventTypeRemediation},
    Severities: []audit.Severity{audit.SeverityCritical},
}
events, err := logger.Query(ctx, filter)

// Export audit logs
file, _ := os.Create("audit-export.csv")
logger.Export(ctx, audit.ExportFormatCSV, file)
```

### Compliance Modes

| Mode | Retention | Encryption | Features |
|------|-----------|------------|----------|
| SOC2 | 3 years | Required | Type 2 compliance, access controls |
| HIPAA | 6 years | Required | PHI tracking, access logs |
| PCI-DSS | 2 years | Required | Payment data tracking |

## Role-Based Access Control (RBAC)

### Overview
Fine-grained access control system with predefined roles and custom policies.

### Predefined Roles

| Role | Description | Permissions |
|------|-------------|-------------|
| Admin | Full system access | All permissions |
| Operator | Manage drift and remediation | Read/write for discovery, drift, remediation, state |
| Viewer | Read-only access | Read permissions only |
| Approver | Approve remediation | Read + approve remediation |

### Usage

```go
import "github.com/catherinevee/driftmgr/internal/rbac"

// Create RBAC manager
store, _ := rbac.NewFileStore("/etc/driftmgr/rbac")
manager, _ := rbac.NewManager(store)

// Create user
user := &rbac.User{
    ID:       "user123",
    Username: "john.doe",
    Email:    "john@example.com",
}
manager.CreateUser(user)

// Assign role
manager.AssignRole("user123", "operator")

// Check permission
hasPermission, _ := manager.CheckPermission(ctx, "user123", rbac.PermissionRemediationExecute)

// Get user permissions
permissions, _ := manager.GetUserPermissions("user123")
```

### Custom Roles

```go
// Create custom role
role := &rbac.Role{
    ID:          "custom-reviewer",
    Name:        "Custom Reviewer",
    Description: "Review drift and approve simple remediation",
    Permissions: []rbac.Permission{
        rbac.PermissionDriftRead,
        rbac.PermissionRemediationRead,
        rbac.PermissionRemediationApprove,
    },
}
manager.CreateRole(role)
```

### Access Policies

```go
// Create policy
policy := &rbac.Policy{
    ID:     "prod-readonly",
    Name:   "Production Read-Only",
    Effect: rbac.EffectAllow,
    Resources: []string{
        "aws:*:prod-*",
        "azure:*:prod-*",
    },
    Actions: []string{
        "read",
        "list",
    },
    Priority: 100,
}
```

## HashiCorp Vault Integration

### Overview
Secure secrets management using HashiCorp Vault.

### Features
- Encrypted credential storage
- Dynamic secrets
- Automatic credential rotation
- Audit logging of secret access

### Configuration

```yaml
vault:
  address: https://vault.example.com:8200
  token: ${VAULT_TOKEN}
  mount_path: secret/driftmgr
  cache_ttl: 5m
  auto_renew: true
```

### Usage

```go
import "github.com/catherinevee/driftmgr/internal/vault"

// Initialize vault client
config := &vault.Config{
    Address:   "https://vault.example.com:8200",
    Token:     os.Getenv("VAULT_TOKEN"),
    MountPath: "secret/driftmgr",
}
client, _ := vault.NewClient(config)

// Store credential
client.StoreCredential(ctx, "aws", map[string]interface{}{
    "access_key_id":     "AKIA...",
    "secret_access_key": "...",
})

// Retrieve credential
creds, _ := client.GetCredential(ctx, "aws")
```

## Circuit Breaker Pattern

### Overview
Prevents cascading failures by temporarily disabling operations that are likely to fail.

### Features
- Automatic failure detection
- Configurable thresholds
- Half-open state for testing recovery
- Per-provider circuit breakers

### Configuration

```yaml
circuit_breaker:
  failure_threshold: 5
  success_threshold: 2
  timeout: 30s
  half_open_requests: 3
```

### Usage

```go
import "github.com/catherinevee/driftmgr/internal/resilience"

// Create circuit breaker
cb := resilience.NewCircuitBreaker("aws-discovery", resilience.Config{
    FailureThreshold: 5,
    SuccessThreshold: 2,
    Timeout:         30 * time.Second,
})

// Execute with circuit breaker
result, err := cb.Execute(func() (interface{}, error) {
    return discoverAWSResources()
})
```

## Rate Limiting

### Overview
Controls the rate of API calls to prevent throttling and ensure fair resource usage.

### Features
- Token bucket algorithm
- Per-provider rate limits
- Automatic backoff
- Burst capacity

### Configuration

```yaml
rate_limits:
  aws:
    requests_per_second: 10
    burst: 20
  azure:
    requests_per_second: 5
    burst: 10
```

### Usage

```go
import "github.com/catherinevee/driftmgr/internal/resilience"

// Create rate limiter
limiter := resilience.NewRateLimiter(10, 20) // 10 req/s, burst of 20

// Use rate limiter
if err := limiter.Wait(ctx); err != nil {
    return err
}
// Proceed with API call
```

## Health Checks

### Overview
Comprehensive health monitoring for all components.

### Endpoints

| Endpoint | Description | Response |
|----------|-------------|----------|
| `/health/live` | Liveness probe | 200 if alive |
| `/health/ready` | Readiness probe | 200 if ready |
| `/metrics` | Prometheus metrics | Metrics in Prometheus format |

### Health Check Response

```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "checks": {
    "database": "healthy",
    "vault": "healthy",
    "aws": "healthy",
    "azure": "degraded",
    "gcp": "healthy"
  },
  "version": "1.0.0",
  "uptime": "2h30m15s"
}
```

## Distributed Tracing

### Overview
Track requests across multiple services and components.

### Features
- OpenTelemetry support
- Correlation IDs
- Performance metrics
- Error tracking

### Configuration

```yaml
tracing:
  enabled: true
  exporter: jaeger
  endpoint: http://jaeger:14268/api/traces
  sample_rate: 0.1
```

## Performance Optimizations

### Caching
- TTL-based cache with automatic invalidation
- Distributed caching with Redis support
- Incremental discovery for large environments

### Parallel Processing
- Concurrent resource discovery
- Batched API calls
- Worker pool management

### Configuration

```yaml
performance:
  cache:
    enabled: true
    ttl: 5m
    max_size: 1000
  parallel:
    workers: 10
    batch_size: 100
    timeout: 30s
```

## Security Features

### Encryption
- AES-256-GCM for data at rest
- TLS 1.3 for data in transit
- Encrypted audit logs

### Authentication
- OAuth 2.0 / OIDC support
- API key authentication
- mTLS for service-to-service

### Configuration

```yaml
security:
  encryption:
    algorithm: AES-256-GCM
    key_rotation: 30d
  authentication:
    type: oauth2
    provider: auth0
    client_id: ${AUTH0_CLIENT_ID}
    client_secret: ${AUTH0_CLIENT_SECRET}
```

## Deployment Patterns

### High Availability
- Multi-region deployment
- Active-passive failover
- Load balancing

### Scalability
- Horizontal scaling
- Auto-scaling based on load
- Distributed state management

### Example Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: driftmgr
spec:
  replicas: 3
  selector:
    matchLabels:
      app: driftmgr
  template:
    metadata:
      labels:
        app: driftmgr
    spec:
      containers:
      - name: driftmgr
        image: catherinevee/driftmgr:latest
        env:
        - name: DRIFTMGR_MODE
          value: "server"
        - name: DRIFTMGR_AUDIT_ENABLED
          value: "true"
        - name: DRIFTMGR_RBAC_ENABLED
          value: "true"
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8080
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
```

## Monitoring & Observability

### Metrics
- Prometheus metrics endpoint
- Custom business metrics
- SLI/SLO tracking

### Logging
- Structured logging (JSON)
- Log aggregation support
- Debug mode

### Alerts
- Configurable alert rules
- Multiple notification channels
- Alert suppression

## Support

For enterprise support and custom features:
- Email: enterprise@driftmgr.io
- Documentation: https://docs.driftmgr.io/enterprise
- GitHub Issues: https://github.com/catherinevee/driftmgr/issues