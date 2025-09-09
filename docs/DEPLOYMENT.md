# DriftMgr Production Deployment Guide

Comprehensive guide for deploying DriftMgr in production environments.

## Table of Contents
- [Deployment Options](#deployment-options)
- [Prerequisites](#prerequisites)
- [Kubernetes Deployment](#kubernetes-deployment)
- [Docker Deployment](#docker-deployment)
- [VM/Bare Metal Deployment](#vmbare-metal-deployment)
- [Cloud-Specific Deployments](#cloud-specific-deployments)
- [High Availability Setup](#high-availability-setup)
- [Security Configuration](#security-configuration)
- [Monitoring Setup](#monitoring-setup)
- [Backup and Disaster Recovery](#backup-and-disaster-recovery)
- [Maintenance](#maintenance)

## Deployment Options

### Comparison Matrix

| Option | Scalability | Complexity | Cost | Best For |
|--------|------------|------------|------|----------|
| Kubernetes | High | High | Medium-High | Large enterprises, multi-team |
| Docker Swarm | Medium | Medium | Medium | Medium organizations |
| Docker Compose | Low | Low | Low | Small teams, POC |
| VM/Bare Metal | Medium | Medium | Variable | On-premise requirements |
| Serverless | High | Low | Pay-per-use | Periodic scans |
| Managed Service | High | Low | High | Zero maintenance |

## Prerequisites

### System Requirements

#### Minimum (up to 1000 resources)
- CPU: 2 cores
- RAM: 4 GB
- Storage: 20 GB SSD
- Network: 10 Mbps

#### Recommended (up to 10,000 resources)
- CPU: 4 cores
- RAM: 16 GB
- Storage: 100 GB SSD
- Network: 100 Mbps

#### Enterprise (10,000+ resources)
- CPU: 8+ cores
- RAM: 32+ GB
- Storage: 500+ GB SSD
- Network: 1 Gbps

### Software Requirements
- Container runtime: Docker 20.10+
- Kubernetes: 1.24+ (if using K8s)
- PostgreSQL: 14+
- Redis: 7+

### Network Requirements
- Outbound HTTPS (443) to cloud provider APIs
- Inbound access for web UI (configurable)
- Database connectivity (PostgreSQL: 5432, Redis: 6379)

## Kubernetes Deployment

### 1. Create Namespace and Secrets
```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: driftmgr
---
# secrets.yaml
apiVersion: v1
kind: Secret
metadata:
  name: driftmgr-secrets
  namespace: driftmgr
type: Opaque
stringData:
  aws-access-key-id: "YOUR_AWS_KEY"
  aws-secret-access-key: "YOUR_AWS_SECRET"
  azure-client-id: "YOUR_AZURE_CLIENT_ID"
  azure-client-secret: "YOUR_AZURE_SECRET"
  gcp-credentials: |
    {
      "type": "service_account",
      "project_id": "your-project"
    }
  db-password: "secure_password"
  jwt-secret: "secure_jwt_secret"
  encryption-key: "secure_encryption_key"
```

Apply:
```bash
kubectl apply -f namespace.yaml
kubectl apply -f secrets.yaml
```

### 2. Deploy PostgreSQL and Redis
```yaml
# postgresql.yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
  namespace: driftmgr
spec:
  serviceName: postgres
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:15-alpine
        env:
        - name: POSTGRES_DB
          value: driftmgr
        - name: POSTGRES_USER
          value: driftmgr
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: driftmgr-secrets
              key: db-password
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
  volumeClaimTemplates:
  - metadata:
      name: postgres-storage
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 50Gi
---
# redis.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis
  namespace: driftmgr
spec:
  replicas: 1
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
      - name: redis
        image: redis:7-alpine
        command: ["redis-server", "--appendonly", "yes"]
        volumeMounts:
        - name: redis-storage
          mountPath: /data
      volumes:
      - name: redis-storage
        persistentVolumeClaim:
          claimName: redis-pvc
```

### 3. Deploy DriftMgr
```yaml
# driftmgr-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: driftmgr
  namespace: driftmgr
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
        ports:
        - containerPort: 8080
          name: web
        - containerPort: 9090
          name: metrics
        env:
        - name: DRIFTMGR_ENV
          value: production
        - name: DATABASE_URL
          value: postgres://driftmgr:$(DB_PASSWORD)@postgres:5432/driftmgr?sslmode=require
        - name: REDIS_URL
          value: redis://redis:6379/0
        - name: AWS_ACCESS_KEY_ID
          valueFrom:
            secretKeyRef:
              name: driftmgr-secrets
              key: aws-access-key-id
        - name: AWS_SECRET_ACCESS_KEY
          valueFrom:
            secretKeyRef:
              name: driftmgr-secrets
              key: aws-secret-access-key
        envFrom:
        - secretRef:
            name: driftmgr-secrets
        volumeMounts:
        - name: config
          mountPath: /app/configs
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
          initialDelaySeconds: 15
          periodSeconds: 5
        resources:
          requests:
            cpu: 500m
            memory: 512Mi
          limits:
            cpu: 2000m
            memory: 2Gi
      volumes:
      - name: config
        configMap:
          name: driftmgr-config
---
# service.yaml
apiVersion: v1
kind: Service
metadata:
  name: driftmgr
  namespace: driftmgr
spec:
  selector:
    app: driftmgr
  ports:
  - name: web
    port: 8080
    targetPort: 8080
  - name: metrics
    port: 9090
    targetPort: 9090
  type: LoadBalancer
```

### 4. Configure Ingress
```yaml
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: driftmgr
  namespace: driftmgr
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/proxy-body-size: "50m"
spec:
  tls:
  - hosts:
    - driftmgr.company.com
    secretName: driftmgr-tls
  rules:
  - host: driftmgr.company.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: driftmgr
            port:
              number: 8080
```

### 5. Horizontal Pod Autoscaler
```yaml
# hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: driftmgr-hpa
  namespace: driftmgr
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: driftmgr
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

## Docker Deployment

### Single Host with Docker Compose

```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  driftmgr:
    image: catherinevee/driftmgr:latest
    container_name: driftmgr
    restart: always
    ports:
      - "443:8080"
    environment:
      - DRIFTMGR_ENV=production
      - SSL_ENABLED=true
      - SSL_CERT=/certs/cert.pem
      - SSL_KEY=/certs/key.pem
    volumes:
      - ./certs:/certs:ro
      - ./configs:/app/configs:ro
      - driftmgr-data:/app/data
    networks:
      - driftmgr-net
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '0.5'
          memory: 512M

  postgres:
    image: postgres:15-alpine
    restart: always
    environment:
      - POSTGRES_DB=driftmgr
      - POSTGRES_USER=driftmgr
      - POSTGRES_PASSWORD=${DB_PASSWORD}
      - POSTGRES_INITDB_ARGS=--encoding=UTF8
    volumes:
      - postgres-data:/var/lib/postgresql/data
      - ./backup:/backup
    networks:
      - driftmgr-net
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U driftmgr"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    restart: always
    command: >
      redis-server
      --appendonly yes
      --requirepass ${REDIS_PASSWORD}
      --maxmemory 512mb
      --maxmemory-policy allkeys-lru
    volumes:
      - redis-data:/data
    networks:
      - driftmgr-net
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  nginx:
    image: nginx:alpine
    restart: always
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./certs:/etc/nginx/certs:ro
      - nginx-cache:/var/cache/nginx
    networks:
      - driftmgr-net
    depends_on:
      - driftmgr

networks:
  driftmgr-net:
    driver: bridge

volumes:
  driftmgr-data:
  postgres-data:
  redis-data:
  nginx-cache:
```

### Docker Swarm Deployment

```bash
# Initialize swarm
docker swarm init

# Create overlay network
docker network create --driver overlay driftmgr-net

# Create secrets
echo "password" | docker secret create db_password -
echo "redis_password" | docker secret create redis_password -

# Deploy stack
docker stack deploy -c docker-compose.prod.yml driftmgr

# Scale service
docker service scale driftmgr_driftmgr=3

# Update service
docker service update --image catherinevee/driftmgr:v2.0 driftmgr_driftmgr
```

## VM/Bare Metal Deployment

### 1. System Preparation
```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install dependencies
sudo apt install -y \
  postgresql-14 \
  redis-server \
  nginx \
  certbot \
  python3-certbot-nginx \
  supervisor

# Create user
sudo useradd -m -s /bin/bash driftmgr
sudo usermod -aG sudo driftmgr
```

### 2. Install DriftMgr
```bash
# Download binary
sudo curl -L https://github.com/catherinevee/driftmgr/releases/latest/download/driftmgr-linux-amd64 \
  -o /usr/local/bin/driftmgr
sudo chmod +x /usr/local/bin/driftmgr

# Create directories
sudo mkdir -p /etc/driftmgr /var/lib/driftmgr /var/log/driftmgr
sudo chown -R driftmgr:driftmgr /etc/driftmgr /var/lib/driftmgr /var/log/driftmgr
```

### 3. Configure PostgreSQL
```bash
# Create database and user
sudo -u postgres psql << EOF
CREATE USER driftmgr WITH PASSWORD 'secure_password';
CREATE DATABASE driftmgr OWNER driftmgr;
GRANT ALL PRIVILEGES ON DATABASE driftmgr TO driftmgr;
EOF

# Configure PostgreSQL
sudo nano /etc/postgresql/14/main/postgresql.conf
# Set: listen_addresses = 'localhost'
# Set: max_connections = 200

sudo systemctl restart postgresql
```

### 4. Configure Systemd Service
```ini
# /etc/systemd/system/driftmgr.service
[Unit]
Description=DriftMgr Infrastructure Drift Detection
After=network.target postgresql.service redis.service
Wants=postgresql.service redis.service

[Service]
Type=simple
User=driftmgr
Group=driftmgr
WorkingDirectory=/var/lib/driftmgr
ExecStart=/usr/local/bin/driftmgr serve web --config /etc/driftmgr/config.yaml
Restart=always
RestartSec=10
StandardOutput=append:/var/log/driftmgr/output.log
StandardError=append:/var/log/driftmgr/error.log

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/driftmgr /var/log/driftmgr

# Resource limits
LimitNOFILE=65535
LimitNPROC=4096
MemoryLimit=2G
CPUQuota=200%

# Environment
Environment="DRIFTMGR_ENV=production"
Environment="DATABASE_URL=postgres://driftmgr:password@localhost/driftmgr"
Environment="REDIS_URL=redis://localhost:6379/0"

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl daemon-reload
sudo systemctl enable driftmgr
sudo systemctl start driftmgr
sudo systemctl status driftmgr
```

### 5. Configure Nginx
```nginx
# /etc/nginx/sites-available/driftmgr
upstream driftmgr {
    server 127.0.0.1:8080;
    keepalive 32;
}

server {
    listen 80;
    server_name driftmgr.company.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name driftmgr.company.com;

    ssl_certificate /etc/letsencrypt/live/driftmgr.company.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/driftmgr.company.com/privkey.pem;
    
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;

    client_max_body_size 50M;
    
    location / {
        proxy_pass http://driftmgr;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_buffering off;
        proxy_read_timeout 86400;
    }
    
    location /metrics {
        deny all;
        allow 10.0.0.0/8;
        proxy_pass http://127.0.0.1:9090;
    }
}
```

Enable site:
```bash
sudo ln -s /etc/nginx/sites-available/driftmgr /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

## Cloud-Specific Deployments

### AWS ECS Fargate

```json
{
  "family": "driftmgr",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "1024",
  "memory": "2048",
  "containerDefinitions": [
    {
      "name": "driftmgr",
      "image": "catherinevee/driftmgr:latest",
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "DRIFTMGR_ENV",
          "value": "production"
        }
      ],
      "secrets": [
        {
          "name": "DATABASE_URL",
          "valueFrom": "arn:aws:secretsmanager:region:account:secret:db-url"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/driftmgr",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "ecs"
        }
      },
      "healthCheck": {
        "command": ["CMD-SHELL", "curl -f http://localhost:8080/health || exit 1"],
        "interval": 30,
        "timeout": 10,
        "retries": 3
      }
    }
  ]
}
```

### Azure Container Instances

```yaml
# azure-deployment.yaml
apiVersion: '2019-12-01'
location: eastus
name: driftmgr
properties:
  containers:
  - name: driftmgr
    properties:
      image: catherinevee/driftmgr:latest
      resources:
        requests:
          cpu: 2
          memoryInGb: 4
      ports:
      - port: 8080
        protocol: TCP
      environmentVariables:
      - name: DRIFTMGR_ENV
        value: production
      - name: DATABASE_URL
        secureValue: postgres://user:pass@host/db
  osType: Linux
  ipAddress:
    type: Public
    ports:
    - protocol: tcp
      port: 8080
    dnsNameLabel: driftmgr
```

### Google Cloud Run

```yaml
# service.yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: driftmgr
spec:
  template:
    metadata:
      annotations:
        run.googleapis.com/execution-environment: gen2
    spec:
      containerConcurrency: 100
      timeoutSeconds: 300
      containers:
      - image: catherinevee/driftmgr:latest
        ports:
        - containerPort: 8080
        env:
        - name: DRIFTMGR_ENV
          value: production
        resources:
          limits:
            cpu: '2'
            memory: 2Gi
```

Deploy:
```bash
gcloud run deploy driftmgr \
  --image catherinevee/driftmgr:latest \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --set-env-vars="DRIFTMGR_ENV=production"
```

## High Availability Setup

### Multi-Region Architecture

```yaml
# Primary Region (us-east-1)
driftmgr-primary:
  replicas: 3
  database: 
    type: RDS Aurora PostgreSQL
    multi-az: true
    read-replicas: 2
  cache:
    type: ElastiCache Redis
    cluster-mode: enabled
  load-balancer:
    type: ALB
    cross-zone: true

# Secondary Region (us-west-2)  
driftmgr-secondary:
  replicas: 2
  database:
    type: RDS Aurora Read Replica
    promote-on-failure: true
  cache:
    type: ElastiCache Redis
    cluster-mode: enabled
  load-balancer:
    type: ALB
    cross-zone: true

# Global Load Balancer
route53:
  - record: driftmgr.company.com
    type: A
    routing-policy: geolocation
    health-check: enabled
    failover: automatic
```

### Database HA Configuration

```sql
-- PostgreSQL Streaming Replication
-- Primary server postgresql.conf
wal_level = replica
max_wal_senders = 10
wal_keep_segments = 64
synchronous_commit = on
synchronous_standby_names = 'standby1,standby2'

-- Standby server recovery.conf
standby_mode = 'on'
primary_conninfo = 'host=primary port=5432 user=replicator'
trigger_file = '/tmp/postgresql.trigger'
```

## Security Configuration

### SSL/TLS Setup
```bash
# Generate certificates with Let's Encrypt
certbot certonly --standalone -d driftmgr.company.com

# Or self-signed for testing
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes
```

### Firewall Rules
```bash
# UFW rules
sudo ufw allow 22/tcp
sudo ufw allow 443/tcp
sudo ufw allow from 10.0.0.0/8 to any port 5432
sudo ufw allow from 10.0.0.0/8 to any port 6379
sudo ufw enable
```

### Secrets Management
```bash
# HashiCorp Vault integration
vault kv put secret/driftmgr \
  db_password="secure_password" \
  jwt_secret="secure_jwt" \
  aws_access_key="KEY" \
  aws_secret_key="SECRET"

# Kubernetes secrets from Vault
kubectl create secret generic driftmgr-secrets \
  --from-literal=db-password="$(vault kv get -field=db_password secret/driftmgr)"
```

## Monitoring Setup

### Prometheus Configuration
```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'driftmgr'
    static_configs:
      - targets: ['driftmgr:9090']
    metrics_path: /metrics
    
  - job_name: 'postgres'
    static_configs:
      - targets: ['postgres-exporter:9187']
      
  - job_name: 'redis'
    static_configs:
      - targets: ['redis-exporter:9121']
```

### Grafana Dashboard
```json
{
  "dashboard": {
    "title": "DriftMgr Production",
    "panels": [
      {
        "title": "Request Rate",
        "targets": [
          {
            "expr": "rate(http_requests_total[5m])"
          }
        ]
      },
      {
        "title": "Drift Detection Rate",
        "targets": [
          {
            "expr": "rate(drift_detections_total[1h])"
          }
        ]
      }
    ]
  }
}
```

## Backup and Disaster Recovery

### Backup Strategy
```bash
#!/bin/bash
# backup.sh - Run daily via cron

# Database backup
pg_dump -h localhost -U driftmgr -d driftmgr | gzip > /backup/db-$(date +%Y%m%d).sql.gz

# Redis backup
redis-cli --rdb /backup/redis-$(date +%Y%m%d).rdb

# Upload to S3
aws s3 sync /backup s3://backup-bucket/driftmgr/

# Cleanup old backups (keep 30 days)
find /backup -mtime +30 -delete
```

### Disaster Recovery Plan

1. **RTO Target**: 1 hour
2. **RPO Target**: 15 minutes
3. **Backup Schedule**: Every 6 hours
4. **Recovery Steps**:
   ```bash
   # Restore database
   gunzip < backup.sql.gz | psql -U driftmgr -d driftmgr
   
   # Restore Redis
   cp redis-backup.rdb /var/lib/redis/dump.rdb
   systemctl restart redis
   
   # Verify services
   driftmgr health check
   ```

## Maintenance

### Rolling Updates
```bash
# Kubernetes
kubectl set image deployment/driftmgr driftmgr=catherinevee/driftmgr:v2.0 -n driftmgr
kubectl rollout status deployment/driftmgr -n driftmgr

# Docker Swarm
docker service update --image catherinevee/driftmgr:v2.0 driftmgr_driftmgr

# Systemd
sudo systemctl stop driftmgr
sudo curl -L https://github.com/catherinevee/driftmgr/releases/download/v2.0/driftmgr-linux-amd64 \
  -o /usr/local/bin/driftmgr
sudo systemctl start driftmgr
```

### Health Checks
```bash
# Check application health
curl https://driftmgr.company.com/health

# Check database
psql -U driftmgr -d driftmgr -c "SELECT 1"

# Check Redis
redis-cli ping

# Check metrics
curl http://localhost:9090/metrics
```

### Log Management
```bash
# Centralized logging with ELK
filebeat.inputs:
- type: log
  enabled: true
  paths:
    - /var/log/driftmgr/*.log
  multiline.pattern: '^\d{4}-\d{2}-\d{2}'
  multiline.negate: true
  multiline.match: after

output.elasticsearch:
  hosts: ["elasticsearch:9200"]
  index: "driftmgr-%{+yyyy.MM.dd}"
```

## Performance Tuning

### Application Tuning
```yaml
# config.yaml
performance:
  parallel_workers: 20
  batch_size: 100
  connection_pool_size: 50
  cache_ttl: 3600
  request_timeout: 30
```

### Database Tuning
```sql
-- PostgreSQL optimization
ALTER SYSTEM SET shared_buffers = '4GB';
ALTER SYSTEM SET effective_cache_size = '12GB';
ALTER SYSTEM SET maintenance_work_mem = '1GB';
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
ALTER SYSTEM SET wal_buffers = '16MB';
ALTER SYSTEM SET default_statistics_target = 100;
ALTER SYSTEM SET random_page_cost = 1.1;
```

### System Tuning
```bash
# /etc/sysctl.conf
net.core.somaxconn = 65535
net.ipv4.tcp_max_syn_backlog = 65535
net.ipv4.tcp_syncookies = 1
net.ipv4.tcp_tw_reuse = 1
net.ipv4.tcp_fin_timeout = 30
net.ipv4.ip_local_port_range = 1024 65535
fs.file-max = 65535
```

## Troubleshooting

### Common Issues

1. **High Memory Usage**
   ```bash
   # Check memory usage
   docker stats driftmgr
   
   # Analyze heap dump
   curl http://localhost:6060/debug/pprof/heap > heap.prof
   go tool pprof heap.prof
   ```

2. **Database Connection Errors**
   ```bash
   # Check connection pool
   psql -U driftmgr -c "SELECT count(*) FROM pg_stat_activity;"
   
   # Reset connections
   psql -U postgres -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='driftmgr' AND state='idle';"
   ```

3. **Slow Performance**
   ```bash
   # Enable profiling
   export DRIFTMGR_ENABLE_PROFILING=true
   
   # Collect CPU profile
   curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
   go tool pprof -http=:8080 cpu.prof
   ```

## Security Checklist

- [ ] SSL/TLS configured
- [ ] Firewall rules configured
- [ ] Secrets in secure store
- [ ] Regular security updates
- [ ] Audit logging enabled
- [ ] RBAC configured
- [ ] Network segmentation
- [ ] Encrypted backups
- [ ] Vulnerability scanning
- [ ] Penetration testing