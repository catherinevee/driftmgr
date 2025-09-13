# DriftMgr Deployment Guide: Phased Approach for Production

This guide provides a comprehensive, phased approach to deploying the DriftMgr Terraform drift detection tool securely and reliably in production environments. Each phase builds upon the previous one, ensuring that this critical infrastructure management tool is properly secured and monitored.

## Phase 1: Foundation & Planning

### Architecture Design for DriftMgr
Before deploying DriftMgr, establish your drift detection architecture considering its unique requirements:

**Security Requirements:**
- DriftMgr requires read access to all cloud resources for discovery
- Write access needed for remediation features
- State files contain sensitive infrastructure information
- API keys and cloud credentials must be protected
- Audit logging required for compliance tracking

**Deployment Models:**
- **Centralized**: Single DriftMgr instance monitoring multiple accounts
- **Distributed**: DriftMgr instances per environment/account
- **Hybrid**: Central UI with distributed agents

### Project Structure Setup
Organize your DriftMgr deployment repository:

```
driftmgr-deployment/
├── cmd/
│   ├── driftmgr/          # Main CLI application
│   ├── web/               # Web UI server
│   └── api/               # API server
├── internal/
│   ├── drift/             # Drift detection logic
│   ├── remediate/         # Remediation strategies
│   ├── providers/         # Cloud provider interfaces
│   └── state/             # State management
├── configs/
│   ├── driftmgr.yaml      # Base configuration
│   ├── providers/         # Provider-specific configs
│   └── policies/          # Compliance policies
├── deployments/
│   ├── docker/            # Container configurations
│   ├── kubernetes/        # K8s manifests
│   └── terraform/         # Infrastructure as Code
└── scripts/
    ├── install.sh         # Installation automation
    └── security-scan.sh   # Security checks
```

### Multi-Cloud Provider Planning
Map out your cloud provider requirements:

```yaml
# configs/providers/multi-cloud.yaml
providers:
  aws:
    accounts:
      - id: "123456789012"
        role: "arn:aws:iam::123456789012:role/DriftMgrReader"
        regions: [us-east-1, us-west-2, eu-west-1]
      - id: "987654321098"
        role: "arn:aws:iam::987654321098:role/DriftMgrReader"
        regions: [us-east-1]
  
  azure:
    subscriptions:
      - id: "${AZURE_SUBSCRIPTION_PROD}"
        tenant_id: "${AZURE_TENANT_ID}"
        service_principal: "${AZURE_CLIENT_ID}"
  
  gcp:
    projects:
      - id: "production-project"
        service_account: "/secrets/gcp-sa.json"
      - id: "staging-project"
        service_account: "/secrets/gcp-sa-staging.json"
```

## Phase 2: Secure Development

### Core Security Implementation for DriftMgr

**Authentication & Authorization Layer:**
Implement role-based access control for DriftMgr operations:

```go
// internal/auth/rbac.go
type Role string

const (
    RoleViewer    Role = "viewer"    // Read-only drift detection
    RoleOperator  Role = "operator"  // Can trigger remediations
    RoleAdmin     Role = "admin"     // Full access including config
)

type Permission struct {
    Resource string
    Action   string
}

var RolePermissions = map[Role][]Permission{
    RoleViewer: {
        {"drift", "read"},
        {"resources", "read"},
        {"reports", "read"},
    },
    RoleOperator: {
        {"drift", "read"},
        {"resources", "read"},
        {"reports", "read"},
        {"remediation", "execute"},
        {"import", "execute"},
    },
    RoleAdmin: {
        {"*", "*"},
    },
}
```

**Secure Credential Management:**
Never hardcode credentials. Implement a credential provider interface:

```go
// internal/providers/credentials.go
type CredentialProvider interface {
    GetAWSCredentials(accountID string) (*aws.Credentials, error)
    GetAzureCredentials(subscriptionID string) (*azure.Credentials, error)
    GetGCPCredentials(projectID string) (*gcp.Credentials, error)
    RotateCredentials() error
}

// Vault implementation
type VaultCredentialProvider struct {
    client *vault.Client
    path   string
}

func (v *VaultCredentialProvider) GetAWSCredentials(accountID string) (*aws.Credentials, error) {
    secret, err := v.client.Logical().Read(fmt.Sprintf("%s/aws/%s", v.path, accountID))
    if err != nil {
        return nil, fmt.Errorf("failed to fetch AWS credentials: %w", err)
    }
    // Validate and return credentials
    return parseAWSCredentials(secret.Data), nil
}
```

### API Security Middleware
Protect the DriftMgr API and web interfaces:

```go
// internal/middleware/security.go
func SecurityHeaders() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("X-Content-Type-Options", "nosniff")
        c.Header("X-Frame-Options", "DENY")
        c.Header("X-XSS-Protection", "1; mode=block")
        c.Header("Content-Security-Policy", "default-src 'self'")
        c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        c.Next()
    }
}

func RateLimiter(rps int) gin.HandlerFunc {
    limiter := rate.NewLimiter(rate.Limit(rps), rps*2)
    return func(c *gin.Context) {
        if !limiter.Allow() {
            c.JSON(429, gin.H{"error": "rate limit exceeded"})
            c.Abort()
            return
        }
        c.Next()
    }
}

func JWTAuth(secret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        claims, err := validateJWT(token, secret)
        if err != nil {
            c.JSON(401, gin.H{"error": "unauthorized"})
            c.Abort()
            return
        }
        c.Set("user", claims.User)
        c.Set("role", claims.Role)
        c.Next()
    }
}
```

### Drift Detection Security
Ensure drift detection operations are secure:

```go
// internal/drift/detector.go
type SecureDetector struct {
    detector *Detector
    auditor  *AuditLogger
    filter   *SensitiveDataFilter
}

func (s *SecureDetector) Detect(ctx context.Context, state *terraform.State) (*DriftReport, error) {
    // Audit the detection request
    s.auditor.Log(AuditEntry{
        Action:    "drift.detect",
        User:      ctx.Value("user").(string),
        Timestamp: time.Now(),
        Resources: len(state.Resources),
    })
    
    // Perform detection
    report, err := s.detector.Detect(ctx, state)
    if err != nil {
        return nil, fmt.Errorf("detection failed: %w", err)
    }
    
    // Filter sensitive data from report
    report = s.filter.FilterReport(report)
    
    return report, nil
}
```

## Phase 3: Testing & Validation

### Unit Testing for DriftMgr Components

**Table-driven tests for drift detection:**

```go
// internal/drift/detector_test.go
func TestDriftDetection(t *testing.T) {
    tests := []struct {
        name          string
        stateFile     string
        cloudState    map[string]interface{}
        expectedDrift []DriftItem
        expectError   bool
    }{
        {
            name:      "no drift in security group",
            stateFile: "testdata/sg_clean.tfstate",
            cloudState: map[string]interface{}{
                "aws_security_group.web": map[string]interface{}{
                    "ingress": []map[string]interface{}{
                        {"from_port": 443, "to_port": 443, "protocol": "tcp"},
                    },
                },
            },
            expectedDrift: []DriftItem{},
        },
        {
            name:      "detect unauthorized port opening",
            stateFile: "testdata/sg_clean.tfstate",
            cloudState: map[string]interface{}{
                "aws_security_group.web": map[string]interface{}{
                    "ingress": []map[string]interface{}{
                        {"from_port": 443, "to_port": 443, "protocol": "tcp"},
                        {"from_port": 22, "to_port": 22, "protocol": "tcp"}, // Drift!
                    },
                },
            },
            expectedDrift: []DriftItem{
                {
                    ResourceType: "aws_security_group",
                    ResourceID:   "web",
                    DriftType:    "MODIFIED",
                    Severity:     "CRITICAL",
                },
            },
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            detector := NewDetector()
            result, err := detector.CompareStates(tt.stateFile, tt.cloudState)
            
            if tt.expectError {
                assert.Error(t, err)
                return
            }
            assert.NoError(t, err)
            assert.Equal(t, tt.expectedDrift, result.Drift)
        })
    }
}
```

### Integration Testing with Mock Providers

```go
// internal/providers/aws_mock_test.go
type MockAWSProvider struct {
    mock.Mock
}

func (m *MockAWSProvider) DiscoverResources(ctx context.Context, region string) ([]Resource, error) {
    args := m.Called(ctx, region)
    return args.Get(0).([]Resource), args.Error(1)
}

func TestAWSDiscovery(t *testing.T) {
    mockProvider := new(MockAWSProvider)
    mockProvider.On("DiscoverResources", mock.Anything, "us-east-1").Return(
        []Resource{
            {Type: "aws_instance", ID: "i-1234567890"},
            {Type: "aws_s3_bucket", ID: "my-bucket"},
        }, nil,
    )
    
    discoverer := NewDiscoverer(mockProvider)
    resources, err := discoverer.Discover(context.Background(), "us-east-1")
    
    assert.NoError(t, err)
    assert.Len(t, resources, 2)
    mockProvider.AssertExpectations(t)
}
```

### Security Testing
Run security-specific tests:

```bash
#!/bin/bash
# scripts/security-test.sh

echo "Running security tests for DriftMgr..."

# Static analysis
echo "→ Running gosec..."
gosec -fmt json -out gosec-report.json ./...

# Check for vulnerabilities
echo "→ Checking dependencies..."
govulncheck ./...

# Test credential handling
echo "→ Testing credential security..."
go test -v ./internal/providers/... -run TestCredentialSecurity

# Verify no secrets in code
echo "→ Scanning for secrets..."
gitleaks detect --source . --verbose

# Test rate limiting
echo "→ Testing rate limits..."
go test -v ./internal/middleware/... -run TestRateLimiter
```

## Phase 4: Build & Package

### Production Build for DriftMgr

```dockerfile
# deployments/docker/Dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

# Install security updates
RUN apk update && apk upgrade && apk add --no-cache ca-certificates git

WORKDIR /build

# Copy dependencies first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build with security flags
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -extldflags '-static'" \
    -tags netgo \
    -trimpath \
    -o driftmgr \
    ./cmd/driftmgr

# Security scan the binary
FROM aquasec/trivy AS scanner
COPY --from=builder /build/driftmgr /driftmgr
RUN trivy fs --exit-code 1 --no-progress /driftmgr

# Runtime stage
FROM scratch

# Import certificates for TLS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Non-root user
COPY --from=builder /etc/passwd /etc/passwd
USER nobody

# Copy binary
COPY --from=builder /build/driftmgr /driftmgr

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/driftmgr", "health"]

ENTRYPOINT ["/driftmgr"]
```

### Configuration Templates

```yaml
# configs/driftmgr-production.yaml
# Production configuration with security defaults

server:
  port: 8080
  tls:
    enabled: true
    cert_file: /secrets/tls/cert.pem
    key_file: /secrets/tls/key.pem
  
  # Security settings
  security:
    jwt_secret: ${JWT_SECRET}
    cors:
      enabled: false
    rate_limit:
      enabled: true
      requests_per_second: 10
    
# Drift detection settings
detection:
  mode: smart
  workers: 20
  timeout: 5m
  
  # Priority configuration for critical resources
  priority_rules:
    critical:
      - aws_security_group
      - aws_iam_role
      - aws_kms_key
      - azure_key_vault
      - gcp_iam_policy
    high:
      - aws_rds_instance
      - aws_elasticache_cluster
      - azure_sql_database
    
# State management
state:
  backends:
    s3:
      bucket: ${STATE_BUCKET}
      encrypt: true
      kms_key_id: ${KMS_KEY_ID}
    
# Monitoring
monitoring:
  metrics:
    enabled: true
    port: 9090
  tracing:
    enabled: true
    endpoint: ${OTEL_EXPORTER_OTLP_ENDPOINT}
    
# Audit logging
audit:
  enabled: true
  destinations:
    - type: cloudwatch
      log_group: /aws/driftmgr/audit
    - type: file
      path: /var/log/driftmgr/audit.json
      rotate: daily
      retain: 30
```

## Phase 5: CI/CD Pipeline

### GitHub Actions Pipeline for DriftMgr

```yaml
# .github/workflows/driftmgr-deploy.yml
name: DriftMgr Deployment Pipeline

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

env:
  GO_VERSION: '1.21'
  DOCKER_REGISTRY: 'your-registry.io'

jobs:
  security-scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Run Gosec Security Scanner
        uses: securego/gosec@master
        with:
          args: '-fmt sarif -out gosec-results.sarif ./...'
      
      - name: Upload Security Results
        uses: github/codeql-action/upload-sarif@v2
        with:
          sarif_file: gosec-results.sarif
      
      - name: Dependency Vulnerability Check
        run: |
          go install golang.org/x/vuln/cmd/govulncheck@latest
          govulncheck ./...
      
      - name: Container Security Scan
        uses: aquasecurity/trivy-action@master
        with:
          image-ref: ${{ env.DOCKER_REGISTRY }}/driftmgr:${{ github.sha }}
          exit-code: '1'
          severity: 'CRITICAL,HIGH'

  test:
    runs-on: ubuntu-latest
    needs: security-scan
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: ${{ env.GO_VERSION }}
      
      - name: Run Tests
        run: |
          go test -v -race -coverprofile=coverage.out ./...
          go tool cover -html=coverage.out -o coverage.html
      
      - name: Upload Coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out

  build:
    runs-on: ubuntu-latest
    needs: test
    steps:
      - uses: actions/checkout@v3
      
      - name: Build Production Binary
        run: |
          VERSION=$(git describe --tags --always)
          COMMIT=$(git rev-parse HEAD)
          
          CGO_ENABLED=0 go build \
            -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
            -trimpath \
            -o driftmgr \
            ./cmd/driftmgr
      
      - name: Build Docker Image
        run: |
          docker build \
            --build-arg VERSION=${VERSION} \
            --build-arg COMMIT=${COMMIT} \
            -t ${{ env.DOCKER_REGISTRY }}/driftmgr:${{ github.sha }} \
            -t ${{ env.DOCKER_REGISTRY }}/driftmgr:latest \
            -f deployments/docker/Dockerfile .
      
      - name: Push to Registry
        if: github.ref == 'refs/heads/main'
        run: |
          echo ${{ secrets.DOCKER_PASSWORD }} | docker login -u ${{ secrets.DOCKER_USERNAME }} --password-stdin
          docker push ${{ env.DOCKER_REGISTRY }}/driftmgr:${{ github.sha }}
          docker push ${{ env.DOCKER_REGISTRY }}/driftmgr:latest

  deploy:
    runs-on: ubuntu-latest
    needs: build
    if: github.ref == 'refs/heads/main'
    steps:
      - name: Deploy to Kubernetes
        run: |
          kubectl set image deployment/driftmgr \
            driftmgr=${{ env.DOCKER_REGISTRY }}/driftmgr:${{ github.sha }} \
            -n driftmgr-system
          
          kubectl rollout status deployment/driftmgr -n driftmgr-system
```

### Terraform for DriftMgr Infrastructure

```hcl
# deployments/terraform/main.tf
module "driftmgr" {
  source = "./modules/driftmgr"
  
  environment = "production"
  
  # IAM roles for cross-account access
  monitored_accounts = [
    "123456789012",
    "987654321098"
  ]
  
  # KMS for encryption
  kms_key_arn = aws_kms_key.driftmgr.arn
  
  # VPC configuration
  vpc_id     = data.aws_vpc.main.id
  subnet_ids = data.aws_subnets.private.ids
  
  # Security groups
  allowed_cidrs = ["10.0.0.0/8"]
  
  # Container configuration
  container_image = "${var.docker_registry}/driftmgr:${var.version}"
  container_cpu   = 2048
  container_memory = 4096
  
  # Autoscaling
  min_instances = 2
  max_instances = 10
  
  # Monitoring
  enable_monitoring = true
  log_retention_days = 30
}
```

## Phase 6: Deployment & Operations

### Kubernetes Deployment for DriftMgr

```yaml
# deployments/kubernetes/driftmgr-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: driftmgr
  namespace: driftmgr-system
  labels:
    app: driftmgr
    component: server
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  selector:
    matchLabels:
      app: driftmgr
  template:
    metadata:
      labels:
        app: driftmgr
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9090"
    spec:
      serviceAccountName: driftmgr
      securityContext:
        runAsNonRoot: true
        runAsUser: 65534
        fsGroup: 65534
      
      containers:
      - name: driftmgr
        image: your-registry.io/driftmgr:latest
        imagePullPolicy: Always
        
        ports:
        - name: http
          containerPort: 8080
          protocol: TCP
        - name: metrics
          containerPort: 9090
          protocol: TCP
        
        env:
        - name: DRIFTMGR_CONFIG
          value: /config/driftmgr.yaml
        - name: AWS_REGION
          value: us-east-1
        - name: LOG_LEVEL
          value: info
        
        envFrom:
        - secretRef:
            name: driftmgr-secrets
        
        volumeMounts:
        - name: config
          mountPath: /config
          readOnly: true
        - name: aws-credentials
          mountPath: /root/.aws
          readOnly: true
        
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "2000m"
        
        livenessProbe:
          httpGet:
            path: /health
            port: http
          initialDelaySeconds: 10
          periodSeconds: 10
        
        readinessProbe:
          httpGet:
            path: /ready
            port: http
          initialDelaySeconds: 5
          periodSeconds: 5
        
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          capabilities:
            drop:
            - ALL
      
      volumes:
      - name: config
        configMap:
          name: driftmgr-config
      - name: aws-credentials
        secret:
          secretName: aws-credentials
```

### Monitoring Configuration

```yaml
# deployments/kubernetes/monitoring.yaml
apiVersion: v1
kind: Service
metadata:
  name: driftmgr-metrics
  namespace: driftmgr-system
  labels:
    app: driftmgr
spec:
  ports:
  - name: metrics
    port: 9090
    targetPort: 9090
  selector:
    app: driftmgr

---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: driftmgr
  namespace: driftmgr-system
spec:
  selector:
    matchLabels:
      app: driftmgr
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics

---
apiVersion: v1
kind: ConfigMap
metadata:
  name: driftmgr-dashboard
  namespace: driftmgr-system
data:
  dashboard.json: |
    {
      "dashboard": {
        "title": "DriftMgr Operations",
        "panels": [
          {
            "title": "Drift Detection Rate",
            "targets": [
              {
                "expr": "rate(driftmgr_detections_total[5m])"
              }
            ]
          },
          {
            "title": "Resources Scanned",
            "targets": [
              {
                "expr": "driftmgr_resources_scanned_total"
              }
            ]
          },
          {
            "title": "Critical Drift Detected",
            "targets": [
              {
                "expr": "driftmgr_drift_critical_total"
              }
            ]
          },
          {
            "title": "API Response Time",
            "targets": [
              {
                "expr": "histogram_quantile(0.95, driftmgr_http_duration_seconds_bucket)"
              }
            ]
          }
        ]
      }
    }
```

## Phase 7: Production Hardening

### Performance Optimization for Large-Scale Scanning

```go
// internal/drift/optimizer.go
type OptimizedDetector struct {
    pool      *WorkerPool
    cache     *ResourceCache
    rateLimit *rate.Limiter
}

func (o *OptimizedDetector) DetectLargeScale(ctx context.Context, states []string) (*DriftReport, error) {
    // Use worker pool for parallel processing
    results := make(chan *DriftItem, 1000)
    errors := make(chan error, len(states))
    
    for _, state := range states {
        o.pool.Submit(func() {
            // Check cache first
            if cached := o.cache.Get(state); cached != nil && !cached.Expired() {
                results <- cached.DriftItem
                return
            }
            
            // Rate limit API calls
            o.rateLimit.Wait(ctx)
            
            // Perform detection
            drift, err := o.detectSingle(ctx, state)
            if err != nil {
                errors <- err
                return
            }
            
            // Update cache
            o.cache.Set(state, drift, 5*time.Minute)
            results <- drift
        })
    }
    
    // Aggregate results
    report := &DriftReport{
        Timestamp: time.Now(),
        Items:     make([]DriftItem, 0, len(states)),
    }
    
    for i := 0; i < len(states); i++ {
        select {
        case item := <-results:
            report.Items = append(report.Items, *item)
        case err := <-errors:
            log.Errorf("Detection error: %v", err)
        case <-ctx.Done():
            return nil, ctx.Err()
        }
    }
    
    return report, nil
}
```

### Security Hardening for Production

```yaml
# deployments/kubernetes/network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: driftmgr-network-policy
  namespace: driftmgr-system
spec:
  podSelector:
    matchLabels:
      app: driftmgr
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 8080
  - from:
    - namespaceSelector:
        matchLabels:
          name: prometheus
    ports:
    - protocol: TCP
      port: 9090
  egress:
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 443  # HTTPS for cloud APIs
  - to:
    - namespaceSelector:
        matchLabels:
          name: kube-system
    ports:
    - protocol: TCP
      port: 53  # DNS
    - protocol: UDP
      port: 53
```

### Automated Remediation Safeguards

```go
// internal/remediate/safeguards.go
type SafeRemediator struct {
    remediator *Remediator
    approver   ApprovalService
    validator  *ChangeValidator
}

func (s *SafeRemediator) Remediate(ctx context.Context, drift *DriftItem) error {
    // Validate the change is safe
    validation := s.validator.Validate(drift)
    if !validation.Safe {
        return fmt.Errorf("unsafe remediation: %s", validation.Reason)
    }
    
    // Check if approval needed for critical resources
    if drift.Severity == "CRITICAL" {
        approval, err := s.approver.RequestApproval(ctx, drift)
        if err != nil {
            return fmt.Errorf("approval request failed: %w", err)
        }
        
        if !approval.Approved {
            return fmt.Errorf("remediation not approved: %s", approval.Reason)
        }
    }
    
    // Create backup before remediation
    backup, err := s.createBackup(drift.ResourceID)
    if err != nil {
        return fmt.Errorf("backup failed: %w", err)
    }
    
    // Perform remediation with rollback capability
    if err := s.remediator.Apply(ctx, drift); err != nil {
        // Attempt rollback
        if rollbackErr := s.restoreBackup(backup); rollbackErr != nil {
            log.Errorf("Rollback failed: %v", rollbackErr)
        }
        return fmt.Errorf("remediation failed: %w", err)
    }
    
    return nil
}
```

## Phase 8: Maintenance & Evolution

### Continuous Monitoring Setup

```yaml
# configs/monitoring-rules.yaml
groups:
- name: driftmgr_alerts
  interval: 30s
  rules:
  
  - alert: CriticalDriftDetected
    expr: driftmgr_drift_critical_total > 0
    for: 1m
    labels:
      severity: critical
      team: platform
    annotations:
      summary: "Critical infrastructure drift detected"
      description: "{{ $value }} critical drift items detected in {{ $labels.provider }}"
  
  - alert: DriftDetectionFailing
    expr: rate(driftmgr_detection_errors_total[5m]) > 0.1
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Drift detection error rate high"
      description: "Error rate: {{ $value }} errors per second"
  
  - alert: UnauthorizedRemediationAttempt
    expr: driftmgr_unauthorized_remediation_total > 0
    for: 1m
    labels:
      severity: critical
      team: security
    annotations:
      summary: "Unauthorized remediation attempt detected"
      description: "User {{ $labels.user }} attempted unauthorized remediation"
  
  - alert: StateFileAccessError
    expr: driftmgr_state_access_errors_total > 5
    for: 2m
    labels:
      severity: high
    annotations:
      summary: "Cannot access Terraform state files"
      description: "{{ $value }} state file access errors in backend {{ $labels.backend }}"
```

### Compliance Reporting

```go
// internal/compliance/reporter.go
type ComplianceReporter struct {
    detector *drift.Detector
    policies []Policy
}

func (c *ComplianceReporter) GenerateReport(ctx context.Context) (*ComplianceReport, error) {
    report := &ComplianceReport{
        Timestamp: time.Now(),
        Standards: []string{"CIS", "PCI-DSS", "HIPAA"},
    }
    
    // Check each compliance policy
    for _, policy := range c.policies {
        result := c.checkPolicy(ctx, policy)
        report.Results = append(report.Results, result)
    }
    
    // Generate executive summary
    report.Summary = c.generateSummary(report.Results)
    
    // Export to multiple formats
    if err := c.exportJSON(report, "/reports/compliance.json"); err != nil {
        return nil, err
    }
    
    if err := c.exportHTML(report, "/reports/compliance.html"); err != nil {
        return nil, err
    }
    
    return report, nil
}
```

### Documentation and Runbooks

```markdown
# DriftMgr Operational Runbooks

## Critical Drift Response

### Scenario: Critical Security Group Modification Detected

**Alert**: `CriticalDriftDetected - aws_security_group modified`

**Response Steps**:

1. **Immediate Assessment** (< 5 minutes)
   ```bash
   # View drift details
   driftmgr drift show --resource aws_security_group.web
   
   # Check who made the change
   driftmgr audit log --resource aws_security_group.web --last 1h
   ```

2. **Impact Analysis**
   - Identify affected applications
   - Check if change was authorized
   - Assess security implications

3. **Remediation Decision**
   - If unauthorized: Immediate revert
   - If authorized: Update Terraform code
   
   ```bash
   # Option 1: Revert to Terraform state
   driftmgr remediate --resource aws_security_group.web --strategy code-as-truth
   
   # Option 2: Update Terraform to match cloud
   driftmgr remediate --resource aws_security_group.web --strategy cloud-as-truth
   ```

4. **Post-Incident**
   - Update documentation
   - Review access controls
   - Consider adding preventive controls

## Performance Tuning

### Large-Scale Scanning Optimization

For environments with >1000 resources:

```yaml
# configs/performance-tuning.yaml
detection:
  mode: smart
  workers: 50  # Increase parallel workers
  
  # Batch processing
  batch:
    enabled: true
    size: 100
    delay: 1s  # Delay between batches
  
  # Caching
  cache:
    enabled: true
    ttl: 5m
    size: 10000
  
  # Prioritization
  priority:
    enabled: true
    scan_critical_first: true
    defer_low_priority: true
```
```

## Getting Started with DriftMgr Deployment

### Quick Start for Development

1. **Clone and setup**:
   ```bash
   git clone https://github.com/catherinevee/driftmgr.git
   cd driftmgr
   cp configs/driftmgr.example.yaml configs/driftmgr.yaml
   ```

2. **Configure providers**:
   ```bash
   # Set up AWS credentials
   aws configure --profile driftmgr-dev
   
   # Export environment variables
   export AWS_PROFILE=driftmgr-dev
   export DRIFTMGR_CONFIG=configs/driftmgr.yaml
   ```

3. **Run locally**:
   ```bash
   go run ./cmd/driftmgr serve web --dev
   ```

### Production Deployment Checklist

- [ ] **Phase 1**: Architecture planned, accounts mapped
- [ ] **Phase 2**: Security controls implemented
- [ ] **Phase 3**: Tests passing with >80% coverage
- [ ] **Phase 4**: Container images scanned and secured
- [ ] **Phase 5**: CI/CD pipeline operational
- [ ] **Phase 6**: Monitoring and alerting configured
- [ ] **Phase 7**: Performance tuned for scale
- [ ] **Phase 8**: Runbooks and documentation complete

### Key Security Considerations for DriftMgr

1. **Least Privilege Access**: DriftMgr only needs read access for detection, write only for remediation
2. **Audit Everything**: Every detection and remediation must be logged
3. **Secure State Files**: Terraform state contains sensitive data - encrypt at rest and in transit
4. **Rate Limiting**: Prevent DriftMgr from overwhelming cloud APIs
5. **Change Validation**: Never auto-remediate critical resources without approval

This phased approach ensures DriftMgr is deployed with security and reliability built-in from the start, providing a robust foundation for managing infrastructure drift across your multi-cloud environment.