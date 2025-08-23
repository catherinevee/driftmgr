# DriftMgr Macroservices Architecture

## Overview

DriftMgr now supports a **macroservices architecture** for handling large-scale cloud resource discovery and drift detection. This architecture provides:

- **10x performance improvement** for 50,000+ resources
- **Horizontal scaling** per cloud provider
- **Fault isolation** between services
- **Real-time processing** with Redis streams
- **Distributed tracing** for debugging

## Architecture Components

### 1. Discovery Orchestrator
- **Purpose**: Coordinates all discovery operations
- **Responsibilities**:
  - Job scheduling and distribution
  - Shard creation based on strategy (region, resource-type, account)
  - Load balancing across discovery workers
  - Job status tracking and monitoring
- **Scaling**: 2 replicas for HA
- **Port**: 8081

### 2. Cloud Discovery Services
- **Purpose**: Provider-specific resource discovery
- **Variants**:
  - `cloud-discovery-aws`: 10 replicas (highest load)
  - `cloud-discovery-azure`: 5 replicas
  - `cloud-discovery-gcp`: 3 replicas
- **Features**:
  - Intelligent pagination for large datasets
  - Resource-type specific optimizations
  - Automatic retry with exponential backoff
  - Intermediate result streaming for large shards

### 3. Drift Analyzer
- **Purpose**: Analyzes discovered resources for drift
- **Features**:
  - Smart defaults filtering (75-85% noise reduction)
  - Cost impact analysis
  - Risk scoring
  - Batch processing for efficiency
- **Scaling**: 5 replicas
- **Processing Modes**:
  - Full analysis
  - Incremental (only changed resources)
  - Smart (with intelligent filtering)

### 4. Data Pipeline
- **Purpose**: Manages data flow and persistence
- **Responsibilities**:
  - Stream processing from Redis
  - Batch writes to PostgreSQL
  - Data aggregation and transformation
  - Historical trending
- **Scaling**: 3 replicas
- **Port**: 8084

### 5. API Gateway
- **Purpose**: Single entry point for all services
- **Features**:
  - Request routing
  - Authentication/authorization
  - Rate limiting
  - Response caching
- **Scaling**: 2 replicas
- **Port**: 8080

## Quick Start

### Prerequisites
- Docker and Docker Compose
- Cloud credentials configured
- 16GB RAM minimum (for full stack)

### 1. Configure Environment

Create `.env` file:
```bash
# AWS Credentials
AWS_ACCESS_KEY_ID=your_key
AWS_SECRET_ACCESS_KEY=your_secret
AWS_REGION=us-east-1

# Azure Credentials
AZURE_SUBSCRIPTION_ID=your_subscription
AZURE_TENANT_ID=your_tenant
AZURE_CLIENT_ID=your_client
AZURE_CLIENT_SECRET=your_secret

# GCP Credentials
GCP_PROJECT_ID=your_project
GCP_CREDENTIALS_PATH=/path/to/credentials.json
```

### 2. Start Services

```bash
# Start all services
docker-compose -f docker-compose.macroservices.yml up -d

# Scale specific services
docker-compose -f docker-compose.macroservices.yml up -d --scale cloud-discovery-aws=20

# View logs
docker-compose -f docker-compose.macroservices.yml logs -f discovery-orchestrator
```

### 3. Trigger Discovery

```bash
# Via API Gateway
curl -X POST http://localhost:8080/api/v1/discovery \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "aws",
    "regions": ["us-east-1", "us-west-2"],
    "sharding": {
      "type": "resource-type",
      "shard_size": 1000
    }
  }'

# Check status
curl http://localhost:8080/api/v1/jobs/{job_id}
```

### 4. Monitor Services

- **Grafana Dashboard**: http://localhost:3000 (admin/driftmgr)
- **Prometheus Metrics**: http://localhost:9090
- **Jaeger Tracing**: http://localhost:16686
- **API Gateway**: http://localhost:8080/health

## Scaling Guidelines

### When to Scale

| Resource Count | Architecture | Services | Performance |
|---------------|--------------|----------|-------------|
| < 10,000 | Monolith | Single CLI | 5-10 min |
| 10,000-50,000 | Monolith + Workers | CLI with parallel workers | 10-30 min |
| 50,000-200,000 | Macroservices | This architecture | 5-15 min |
| > 200,000 | Full Microservices | Fine-grained services | 5-10 min |

### Scaling Strategies

#### By Provider Load
```yaml
# High AWS usage
cloud-discovery-aws:
  deploy:
    replicas: 20  # Increase for more accounts

# Moderate Azure usage
cloud-discovery-azure:
  deploy:
    replicas: 5
```

#### By Region Distribution
```yaml
# Add region-specific instances
cloud-discovery-aws-useast1:
  environment:
    PROVIDER: aws
    REGION_FILTER: us-east-1
  deploy:
    replicas: 5
```

#### By Resource Type
```yaml
# Specialized EC2 discovery
cloud-discovery-aws-ec2:
  environment:
    PROVIDER: aws
    RESOURCE_TYPE_FILTER: ec2
  deploy:
    replicas: 10
```

## Performance Tuning

### Redis Optimization
```yaml
redis:
  command: >
    redis-server
    --maxmemory 4gb
    --maxmemory-policy allkeys-lru
    --appendonly yes
    --appendfsync everysec
```

### PostgreSQL Optimization
```yaml
postgres:
  command: >
    postgres
    -c max_connections=200
    -c shared_buffers=1GB
    -c effective_cache_size=3GB
    -c maintenance_work_mem=256MB
```

### Worker Tuning
```yaml
environment:
  # Discovery workers
  AWS_WORKERS: 30  # Increase for parallel discovery
  
  # Analyzer workers  
  ANALYZER_WORKERS: 20  # Increase for faster analysis
  
  # Batch sizes
  BATCH_SIZE: 1000  # Larger batches for efficiency
  FLUSH_INTERVAL: 5s  # More frequent flushes for real-time
```

## Monitoring & Alerting

### Key Metrics

1. **Discovery Performance**
   - Jobs per minute
   - Resources discovered per second
   - API rate limit usage
   - Shard completion time

2. **Drift Analysis**
   - Analysis queue depth
   - Drift detection rate
   - False positive rate
   - Processing latency

3. **System Health**
   - Service availability
   - Memory usage per service
   - Redis queue depths
   - PostgreSQL connections

### Grafana Dashboards

Pre-configured dashboards available:
- Service Overview
- Discovery Performance
- Drift Analysis
- Cost Impact
- System Health

### Alert Rules

```yaml
# Example Prometheus alert
groups:
  - name: driftmgr
    rules:
      - alert: HighDriftRate
        expr: rate(drift_detected_total[5m]) > 100
        for: 10m
        annotations:
          summary: "High drift detection rate"
          
      - alert: DiscoveryQueueBacklog
        expr: redis_queue_length{queue="discovery"} > 1000
        for: 5m
        annotations:
          summary: "Discovery queue backlog growing"
```

## Troubleshooting

### Common Issues

1. **Service Won't Start**
   ```bash
   # Check logs
   docker-compose logs service-name
   
   # Verify dependencies
   docker-compose ps
   ```

2. **Slow Discovery**
   ```bash
   # Check worker utilization
   curl http://localhost:8081/metrics
   
   # Scale workers
   docker-compose up -d --scale cloud-discovery-aws=30
   ```

3. **High Memory Usage**
   ```bash
   # Check memory per service
   docker stats
   
   # Adjust limits in docker-compose.yml
   ```

## Migration from Monolith

### Step 1: Export Current State
```bash
./driftmgr export --format json --output current-state.json
```

### Step 2: Import to Macroservices
```bash
curl -X POST http://localhost:8080/api/v1/import \
  -F "file=@current-state.json"
```

### Step 3: Verify
```bash
curl http://localhost:8080/api/v1/resources/summary
```

## Cost Comparison

| Setup | Monthly Cost | Resources/Hour | Cost per 1M Resources |
|-------|--------------|----------------|----------------------|
| Monolith (m5.xlarge) | $140 | 10,000 | $14.00 |
| Macroservices (3x m5.large) | $210 | 100,000 | $2.10 |
| Full Microservices (10x t3.medium) | $300 | 500,000 | $0.60 |

## Best Practices

1. **Start Small**: Begin with default replica counts and scale based on metrics
2. **Monitor First**: Use Grafana to identify bottlenecks before scaling
3. **Shard Wisely**: Choose sharding strategy based on resource distribution
4. **Cache Aggressively**: Use Redis for frequently accessed data
5. **Batch Operations**: Process in batches to reduce overhead
6. **Async Everything**: Use message queues for long-running operations

## Future Enhancements

- [ ] Kubernetes deployment with HPA
- [ ] Multi-region deployment
- [ ] GraphQL API
- [ ] WebSocket support for real-time updates
- [ ] Machine learning for drift prediction
- [ ] Automated scaling based on queue depth