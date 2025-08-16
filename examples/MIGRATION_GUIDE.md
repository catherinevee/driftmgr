# DriftMgr Microservices Migration Guide

## Overview

This guide provides step-by-step instructions for migrating from the current monolithic DriftMgr architecture to the new microservices-based architecture.

## Prerequisites

### System Requirements
- Docker and Docker Compose
- Kubernetes cluster (for production deployment)
- PostgreSQL 15+
- Redis 7+
- Go 1.23+

### Development Environment
- Git
- Make
- kubectl (for Kubernetes deployment)
- helm (optional, for Helm charts)

## Migration Strategy

### Phase 1: Preparation (Week 1)

#### 1.1 Environment Setup
```bash
# Clone the repository
git clone https://github.com/catherinevee/driftmgr.git
cd driftmgr

# Create microservices directory structure
mkdir -p microservices/{discovery-service,analysis-service,remediation-service,workflow-service,notification-service,web-service,cli-service,database-service,cache-service,ml-service,monitoring-service,security-service,api-gateway}

# Set up development environment
cp .env.example .env
# Edit .env with your configuration
```

#### 1.2 Database Migration
```sql
-- Create new database schema for microservices
CREATE DATABASE driftmgr_microservices;

-- Run migration scripts
\i migrations/001_create_discovery_tables.sql
\i migrations/002_create_analysis_tables.sql
\i migrations/003_create_remediation_tables.sql
\i migrations/004_create_workflow_tables.sql
\i migrations/005_create_notification_tables.sql
```

#### 1.3 Shared Libraries Extraction
```bash
# Extract shared libraries to pkg directory
mkdir -p pkg/{models,utils,config,monitoring}

# Move common code
mv internal/models/* pkg/models/
mv internal/utils/* pkg/utils/
mv internal/config/* pkg/config/
mv internal/monitoring/* pkg/monitoring/
```

### Phase 2: Service Extraction (Weeks 2-4)

#### 2.1 Discovery Service (Week 2)
```bash
# Create discovery service
cd microservices/discovery-service

# Copy relevant code from monolithic server
cp ../../cmd/driftmgr-server/main.go .
# Extract discovery-related handlers and functions

# Create Dockerfile
cat > Dockerfile << 'EOF'
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o discovery-service .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/discovery-service .
EXPOSE 8081
CMD ["./discovery-service"]
EOF

# Build and test
docker build -t driftmgr/discovery-service:latest .
docker run -p 8081:8081 driftmgr/discovery-service:latest
```

#### 2.2 Analysis Service (Week 2-3)
```bash
# Create analysis service
cd ../analysis-service

# Extract analysis-related code
# Copy drift detection, pattern recognition, and risk assessment logic

# Create service-specific models
mkdir -p models
# Define analysis-specific data structures

# Build and test
docker build -t driftmgr/analysis-service:latest .
```

#### 2.3 Remediation Service (Week 3)
```bash
# Create remediation service
cd ../remediation-service

# Extract remediation logic
# Include strategy generation, command execution, and rollback functionality

# Create approval workflow system
mkdir -p workflows
# Define approval processes and workflows

# Build and test
docker build -t driftmgr/remediation-service:latest .
```

#### 2.4 API Gateway (Week 3-4)
```bash
# Create API Gateway
cd ../api-gateway

# Implement routing logic
# Add authentication, rate limiting, and CORS middleware

# Configure service discovery
# Set up health checks and load balancing

# Build and test
docker build -t driftmgr/api-gateway:latest .
```

### Phase 3: Infrastructure Setup (Week 4-5)

#### 3.1 Docker Compose Setup
```bash
# Create docker-compose.yml for local development
cd microservices
docker-compose up -d

# Verify all services are running
docker-compose ps
docker-compose logs -f
```

#### 3.2 Kubernetes Deployment
```bash
# Create Kubernetes manifests
cd k8s

# Apply namespace
kubectl apply -f namespace.yaml

# Apply secrets
kubectl apply -f secrets.yaml

# Deploy services
kubectl apply -f discovery-service.yaml
kubectl apply -f analysis-service.yaml
kubectl apply -f remediation-service.yaml
kubectl apply -f api-gateway.yaml

# Verify deployment
kubectl get pods -n driftmgr
kubectl get services -n driftmgr
```

#### 3.3 Monitoring Setup
```bash
# Deploy monitoring stack
kubectl apply -f monitoring/

# Access monitoring tools
# Prometheus: http://localhost:9090
# Grafana: http://localhost:3000 (admin/admin)
# Jaeger: http://localhost:16686
```

### Phase 4: Testing and Validation (Week 5-6)

#### 4.1 Unit Testing
```bash
# Run unit tests for each service
cd discovery-service
go test ./...

cd ../analysis-service
go test ./...

cd ../remediation-service
go test ./...
```

#### 4.2 Integration Testing
```bash
# Create integration test suite
mkdir -p tests/integration

# Test service communication
go test ./tests/integration/...

# Test end-to-end workflows
go test ./tests/e2e/...
```

#### 4.3 Performance Testing
```bash
# Run load tests
cd tests/performance
go run load_test.go

# Monitor performance metrics
# Check response times, throughput, and resource usage
```

### Phase 5: Data Migration (Week 6)

#### 5.1 Data Export
```bash
# Export data from monolithic database
pg_dump driftmgr > driftmgr_backup.sql

# Transform data for microservices
python scripts/transform_data.py driftmgr_backup.sql
```

#### 5.2 Data Import
```bash
# Import data to microservices databases
psql driftmgr_discovery < data/discovery_data.sql
psql driftmgr_analysis < data/analysis_data.sql
psql driftmgr_remediation < data/remediation_data.sql
```

#### 5.3 Data Validation
```bash
# Verify data integrity
python scripts/validate_data.py

# Run consistency checks
go run cmd/validate/main.go
```

### Phase 6: Production Deployment (Week 7)

#### 6.1 Production Environment Setup
```bash
# Set up production Kubernetes cluster
# Configure ingress controllers
# Set up SSL certificates
# Configure monitoring and alerting

# Deploy to production
kubectl apply -f k8s/production/
```

#### 6.2 Blue-Green Deployment
```bash
# Deploy new microservices alongside existing monolithic app
# Gradually shift traffic to microservices
# Monitor for issues and rollback if necessary

# Update DNS and load balancer configuration
# Complete traffic migration
```

#### 6.3 Post-Deployment Validation
```bash
# Run smoke tests
./scripts/smoke_tests.sh

# Monitor application metrics
# Check error rates and performance
# Validate all functionality works correctly
```

## Configuration Management

### Environment Variables
```bash
# Create environment-specific configs
cp .env.example .env.development
cp .env.example .env.staging
cp .env.example .env.production

# Configure service URLs
DISCOVERY_SERVICE_URL=http://discovery-service:8081
ANALYSIS_SERVICE_URL=http://analysis-service:8082
REMEDIATION_SERVICE_URL=http://remediation-service:8083
# ... other services
```

### Kubernetes Secrets
```yaml
# Create secrets for sensitive data
apiVersion: v1
kind: Secret
metadata:
  name: driftmgr-secrets
  namespace: driftmgr
type: Opaque
data:
  database-url: <base64-encoded-url>
  redis-url: <base64-encoded-url>
  aws-access-key-id: <base64-encoded-key>
  aws-secret-access-key: <base64-encoded-secret>
  # ... other secrets
```

## Monitoring and Observability

### Metrics Collection
```yaml
# Prometheus configuration
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'driftmgr-services'
    static_configs:
      - targets: ['discovery-service:8081', 'analysis-service:8082', 'remediation-service:8083']
```

### Logging
```yaml
# ELK stack configuration
# Configure log aggregation and search
# Set up log retention policies
# Create dashboards for different services
```

### Alerting
```yaml
# AlertManager configuration
# Define alert rules for:
# - Service health
# - Performance degradation
# - Error rates
# - Resource usage
```

## Troubleshooting

### Common Issues

#### 1. Service Communication Failures
```bash
# Check service discovery
kubectl get endpoints -n driftmgr

# Verify network policies
kubectl get networkpolicies -n driftmgr

# Test connectivity
kubectl exec -it <pod-name> -- curl http://service-name:port/health
```

#### 2. Database Connection Issues
```bash
# Check database connectivity
kubectl exec -it <pod-name> -- nc -zv postgres 5432

# Verify database credentials
kubectl get secret driftmgr-secrets -o yaml
```

#### 3. Performance Issues
```bash
# Monitor resource usage
kubectl top pods -n driftmgr

# Check logs for errors
kubectl logs -f <pod-name> -n driftmgr

# Analyze metrics in Grafana
```

### Rollback Procedures
```bash
# Rollback to previous version
kubectl rollout undo deployment/discovery-service -n driftmgr

# Rollback all services
kubectl rollout undo deployment --all -n driftmgr

# Restore from backup
kubectl delete deployment --all -n driftmgr
kubectl apply -f k8s/backup/
```

## Best Practices

### Development
1. **Service Independence**: Each service should be independently deployable
2. **API Versioning**: Use semantic versioning for APIs
3. **Error Handling**: Implement proper error handling and retry mechanisms
4. **Testing**: Maintain high test coverage for each service

### Deployment
1. **Health Checks**: Implement proper health checks for all services
2. **Resource Limits**: Set appropriate resource limits and requests
3. **Security**: Use security contexts and network policies
4. **Monitoring**: Implement comprehensive monitoring and alerting

### Operations
1. **Backup Strategy**: Regular backups of databases and configurations
2. **Disaster Recovery**: Plan for disaster recovery scenarios
3. **Scaling**: Implement horizontal scaling for high-traffic services
4. **Documentation**: Maintain up-to-date documentation

## Success Metrics

### Performance Metrics
- Response time < 200ms for 95% of requests
- Throughput > 1000 requests/second
- Availability > 99.9%

### Business Metrics
- Reduced deployment time
- Improved developer productivity
- Better resource utilization
- Enhanced system reliability

## Conclusion

This migration guide provides a comprehensive approach to transitioning DriftMgr from a monolithic architecture to microservices. Follow the phases carefully, test thoroughly at each stage, and maintain proper monitoring throughout the process.

For additional support or questions, refer to the project documentation or create an issue in the repository.
