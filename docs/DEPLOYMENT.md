# DriftMgr Deployment Guide

This guide covers various deployment scenarios for DriftMgr, from simple single-server deployments to complex multi-node production environments.

## 📋 Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Single Server Deployment](#single-server-deployment)
- [Docker Deployment](#docker-deployment)
- [Kubernetes Deployment](#kubernetes-deployment)
- [Production Considerations](#production-considerations)
- [Monitoring & Logging](#monitoring--logging)
- [Backup & Recovery](#backup--recovery)
- [Security](#security)
- [Troubleshooting](#troubleshooting)

## 🌟 Overview

DriftMgr can be deployed in various configurations depending on your requirements:

- **Development**: Single server with local database
- **Staging**: Docker containers with external database
- **Production**: Kubernetes cluster with high availability
- **Enterprise**: Multi-region deployment with load balancing

## 📋 Prerequisites

### System Requirements

#### Minimum Requirements
- **CPU**: 2 cores
- **RAM**: 4 GB
- **Storage**: 20 GB
- **OS**: Linux (Ubuntu 20.04+, CentOS 8+, RHEL 8+)

#### Recommended Requirements
- **CPU**: 4+ cores
- **RAM**: 8+ GB
- **Storage**: 100+ GB SSD
- **OS**: Linux (Ubuntu 22.04 LTS, RHEL 9+)

### Software Dependencies

- **Go**: 1.21 or later
- **PostgreSQL**: 13 or later
- **Docker**: 20.10+ (for containerized deployment)
- **Kubernetes**: 1.24+ (for K8s deployment)
- **Nginx**: 1.18+ (for reverse proxy)

### Network Requirements

- **Ports**: 8080 (HTTP), 443 (HTTPS), 5432 (PostgreSQL)
- **Firewall**: Configure appropriate rules
- **SSL/TLS**: Certificates for HTTPS

## 🖥️ Single Server Deployment

### Step 1: Prepare the Server

```bash
# Update system packages
sudo apt update && sudo apt upgrade -y

# Install required packages
sudo apt install -y postgresql postgresql-contrib nginx certbot python3-certbot-nginx

# Create driftmgr user
sudo useradd -r -s /bin/false driftmgr
sudo mkdir -p /opt/driftmgr
sudo chown driftmgr:driftmgr /opt/driftmgr
```

### Step 2: Database Setup

```bash
# Switch to postgres user
sudo -u postgres psql

# Create database and user
CREATE DATABASE driftmgr;
CREATE USER driftmgr WITH PASSWORD 'secure-password';
GRANT ALL PRIVILEGES ON DATABASE driftmgr TO driftmgr;
ALTER USER driftmgr CREATEDB;
\q
```

### Step 3: Install DriftMgr

```bash
# Download and extract DriftMgr
cd /opt/driftmgr
sudo wget https://github.com/catherinevee/driftmgr/releases/latest/download/driftmgr-linux-amd64.tar.gz
sudo tar -xzf driftmgr-linux-amd64.tar.gz
sudo chown -R driftmgr:driftmgr /opt/driftmgr
```

### Step 4: Configuration

```bash
# Create configuration file
sudo tee /opt/driftmgr/config.yaml > /dev/null <<EOF
server:
  host: "0.0.0.0"
  port: 8080
  auth_enabled: true
  cors_enabled: true
  rate_limit_enabled: true
  rate_limit_rps: 100

database:
  host: "localhost"
  port: 5432
  name: "driftmgr"
  user: "driftmgr"
  password: "secure-password"
  ssl_mode: "require"

auth:
  jwt_secret: "$(openssl rand -base64 32)"
  jwt_issuer: "driftmgr"
  jwt_audience: "driftmgr-api"
  access_token_expiry: "15m"
  refresh_token_expiry: "7d"

logging:
  level: "info"
  format: "json"
  output: "/var/log/driftmgr/driftmgr.log"
EOF

# Create log directory
sudo mkdir -p /var/log/driftmgr
sudo chown driftmgr:driftmgr /var/log/driftmgr
```

### Step 5: Systemd Service

```bash
# Create systemd service file
sudo tee /etc/systemd/system/driftmgr.service > /dev/null <<EOF
[Unit]
Description=DriftMgr Server
After=network.target postgresql.service
Requires=postgresql.service

[Service]
Type=simple
User=driftmgr
Group=driftmgr
WorkingDirectory=/opt/driftmgr
ExecStart=/opt/driftmgr/driftmgr-server --config=/opt/driftmgr/config.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=driftmgr

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/driftmgr

[Install]
WantedBy=multi-user.target
EOF

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable driftmgr
sudo systemctl start driftmgr
sudo systemctl status driftmgr
```

### Step 6: Nginx Configuration

```bash
# Create Nginx configuration
sudo tee /etc/nginx/sites-available/driftmgr > /dev/null <<EOF
server {
    listen 80;
    server_name your-domain.com;
    
    # Redirect HTTP to HTTPS
    return 301 https://\$server_name\$request_uri;
}

server {
    listen 443 ssl http2;
    server_name your-domain.com;
    
    # SSL configuration
    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512:ECDHE-RSA-AES256-GCM-SHA384:DHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;
    
    # Security headers
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;
    add_header X-XSS-Protection "1; mode=block";
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    
    # Rate limiting
    limit_req_zone \$binary_remote_addr zone=api:10m rate=10r/s;
    limit_req_zone \$binary_remote_addr zone=login:10m rate=1r/s;
    
    # Main application
    location / {
        limit_req zone=api burst=20 nodelay;
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header X-Forwarded-Host \$host;
        proxy_set_header X-Forwarded-Port \$server_port;
    }
    
    # WebSocket support
    location /ws {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_read_timeout 86400;
    }
    
    # Login rate limiting
    location /api/v1/auth/login {
        limit_req zone=login burst=5 nodelay;
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
    
    # Static files caching
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg)$ {
        proxy_pass http://127.0.0.1:8080;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
EOF

# Enable site
sudo ln -s /etc/nginx/sites-available/driftmgr /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

### Step 7: SSL Certificate

```bash
# Obtain SSL certificate
sudo certbot --nginx -d your-domain.com

# Test certificate renewal
sudo certbot renew --dry-run
```

### Step 8: Firewall Configuration

```bash
# Configure UFW
sudo ufw allow ssh
sudo ufw allow 'Nginx Full'
sudo ufw --force enable
```

## 🐳 Docker Deployment

### Docker Compose Setup

```yaml
# docker-compose.yml
version: '3.8'

services:
  driftmgr:
    image: driftmgr/driftmgr:latest
    container_name: driftmgr
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      - DRIFTMGR_DB_HOST=postgres
      - DRIFTMGR_DB_PASSWORD=secure-password
      - DRIFTMGR_JWT_SECRET=your-jwt-secret
      - DRIFTMGR_LOG_LEVEL=info
    depends_on:
      postgres:
        condition: service_healthy
    volumes:
      - driftmgr_logs:/var/log/driftmgr
    networks:
      - driftmgr-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  postgres:
    image: postgres:15-alpine
    container_name: driftmgr-postgres
    restart: unless-stopped
    environment:
      - POSTGRES_DB=driftmgr
      - POSTGRES_USER=driftmgr
      - POSTGRES_PASSWORD=secure-password
      - POSTGRES_INITDB_ARGS=--auth-host=scram-sha-256
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./init.sql:/docker-entrypoint-initdb.d/init.sql:ro
    networks:
      - driftmgr-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U driftmgr -d driftmgr"]
      interval: 10s
      timeout: 5s
      retries: 5

  nginx:
    image: nginx:alpine
    container_name: driftmgr-nginx
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/nginx/ssl:ro
    depends_on:
      - driftmgr
    networks:
      - driftmgr-network

volumes:
  postgres_data:
    driver: local
  driftmgr_logs:
    driver: local

networks:
  driftmgr-network:
    driver: bridge
```

### Nginx Configuration for Docker

```nginx
# nginx.conf
events {
    worker_connections 1024;
}

http {
    upstream driftmgr {
        server driftmgr:8080;
    }
    
    server {
        listen 80;
        server_name your-domain.com;
        return 301 https://$server_name$request_uri;
    }
    
    server {
        listen 443 ssl http2;
        server_name your-domain.com;
        
        ssl_certificate /etc/nginx/ssl/cert.pem;
        ssl_certificate_key /etc/nginx/ssl/key.pem;
        
        location / {
            proxy_pass http://driftmgr;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
        
        location /ws {
            proxy_pass http://driftmgr;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_set_header Host $host;
        }
    }
}
```

### Database Initialization

```sql
-- init.sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Create additional indexes for performance
CREATE INDEX IF NOT EXISTS idx_resources_provider ON resources(provider);
CREATE INDEX IF NOT EXISTS idx_resources_type ON resources(type);
CREATE INDEX IF NOT EXISTS idx_resources_region ON resources(region);
CREATE INDEX IF NOT EXISTS idx_drift_results_status ON drift_results(status);
CREATE INDEX IF NOT EXISTS idx_drift_results_created_at ON drift_results(created_at);
```

### Deployment Commands

```bash
# Start services
docker-compose up -d

# View logs
docker-compose logs -f driftmgr

# Scale DriftMgr instances
docker-compose up -d --scale driftmgr=3

# Update to latest version
docker-compose pull driftmgr
docker-compose up -d driftmgr

# Backup database
docker-compose exec postgres pg_dump -U driftmgr driftmgr > backup.sql
```

## ☸️ Kubernetes Deployment

### Namespace and ConfigMap

```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: driftmgr
---
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: driftmgr-config
  namespace: driftmgr
data:
  config.yaml: |
    server:
      host: "0.0.0.0"
      port: 8080
      auth_enabled: true
      cors_enabled: true
      rate_limit_enabled: true
      rate_limit_rps: 100
    
    database:
      host: "postgres-service"
      port: 5432
      name: "driftmgr"
      user: "driftmgr"
      ssl_mode: "require"
    
    auth:
      jwt_issuer: "driftmgr"
      jwt_audience: "driftmgr-api"
      access_token_expiry: "15m"
      refresh_token_expiry: "7d"
    
    logging:
      level: "info"
      format: "json"
```

### Secrets

```yaml
# secrets.yaml
apiVersion: v1
kind: Secret
metadata:
  name: driftmgr-secrets
  namespace: driftmgr
type: Opaque
data:
  db-password: <base64-encoded-password>
  jwt-secret: <base64-encoded-jwt-secret>
```

### PostgreSQL Deployment

```yaml
# postgres.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: postgres
  namespace: driftmgr
spec:
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
          value: "driftmgr"
        - name: POSTGRES_USER
          value: "driftmgr"
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: driftmgr-secrets
              key: db-password
        ports:
        - containerPort: 5432
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
        - name: init-script
          mountPath: /docker-entrypoint-initdb.d
        resources:
          requests:
            memory: "512Mi"
            cpu: "250m"
          limits:
            memory: "1Gi"
            cpu: "500m"
        livenessProbe:
          exec:
            command:
            - pg_isready
            - -U
            - driftmgr
            - -d
            - driftmgr
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          exec:
            command:
            - pg_isready
            - -U
            - driftmgr
            - -d
            - driftmgr
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: postgres-storage
        persistentVolumeClaim:
          claimName: postgres-pvc
      - name: init-script
        configMap:
          name: postgres-init
---
apiVersion: v1
kind: Service
metadata:
  name: postgres-service
  namespace: driftmgr
spec:
  selector:
    app: postgres
  ports:
  - port: 5432
    targetPort: 5432
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: postgres-pvc
  namespace: driftmgr
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 20Gi
```

### DriftMgr Deployment

```yaml
# driftmgr.yaml
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
        image: driftmgr/driftmgr:latest
        ports:
        - containerPort: 8080
        env:
        - name: DRIFTMGR_DB_HOST
          value: "postgres-service"
        - name: DRIFTMGR_DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: driftmgr-secrets
              key: db-password
        - name: DRIFTMGR_JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: driftmgr-secrets
              key: jwt-secret
        - name: DRIFTMGR_LOG_LEVEL
          value: "info"
        volumeMounts:
        - name: config
          mountPath: /app/config.yaml
          subPath: config.yaml
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: config
        configMap:
          name: driftmgr-config
---
apiVersion: v1
kind: Service
metadata:
  name: driftmgr-service
  namespace: driftmgr
spec:
  selector:
    app: driftmgr
  ports:
  - port: 80
    targetPort: 8080
  type: ClusterIP
---
apiVersion: v1
kind: Service
metadata:
  name: driftmgr-headless
  namespace: driftmgr
spec:
  selector:
    app: driftmgr
  ports:
  - port: 8080
    targetPort: 8080
  clusterIP: None
```

### Ingress Configuration

```yaml
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: driftmgr-ingress
  namespace: driftmgr
  annotations:
    kubernetes.io/ingress.class: "nginx"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
    nginx.ingress.kubernetes.io/proxy-body-size: "10m"
    nginx.ingress.kubernetes.io/websocket-services: "driftmgr-service"
spec:
  tls:
  - hosts:
    - your-domain.com
    secretName: driftmgr-tls
  rules:
  - host: your-domain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: driftmgr-service
            port:
              number: 80
```

### Horizontal Pod Autoscaler

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
  minReplicas: 3
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

### Deployment Commands

```bash
# Apply all configurations
kubectl apply -f namespace.yaml
kubectl apply -f configmap.yaml
kubectl apply -f secrets.yaml
kubectl apply -f postgres.yaml
kubectl apply -f driftmgr.yaml
kubectl apply -f ingress.yaml
kubectl apply -f hpa.yaml

# Check deployment status
kubectl get pods -n driftmgr
kubectl get services -n driftmgr
kubectl get ingress -n driftmgr

# View logs
kubectl logs -f deployment/driftmgr -n driftmgr

# Scale deployment
kubectl scale deployment driftmgr --replicas=5 -n driftmgr
```

## 🏭 Production Considerations

### High Availability

#### Database Clustering
```bash
# PostgreSQL streaming replication
# Primary server
sudo -u postgres pg_basebackup -h primary-server -D /var/lib/postgresql/replica -U replicator -v -P -W

# Configure replica
echo "standby_mode = 'on'" >> /var/lib/postgresql/replica/recovery.conf
echo "primary_conninfo = 'host=primary-server port=5432 user=replicator'" >> /var/lib/postgresql/replica/recovery.conf
```

#### Load Balancing
```nginx
# Nginx upstream configuration
upstream driftmgr_backend {
    least_conn;
    server driftmgr-1:8080 max_fails=3 fail_timeout=30s;
    server driftmgr-2:8080 max_fails=3 fail_timeout=30s;
    server driftmgr-3:8080 max_fails=3 fail_timeout=30s;
    keepalive 32;
}
```

### Performance Optimization

#### Database Tuning
```sql
-- PostgreSQL configuration optimizations
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET effective_cache_size = '1GB';
ALTER SYSTEM SET maintenance_work_mem = '64MB';
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
ALTER SYSTEM SET wal_buffers = '16MB';
ALTER SYSTEM SET default_statistics_target = 100;
SELECT pg_reload_conf();
```

#### Application Tuning
```yaml
# config.yaml optimizations
server:
  read_timeout: "30s"
  write_timeout: "30s"
  idle_timeout: "120s"
  max_header_bytes: 1048576  # 1MB

database:
  max_open_connections: 25
  max_idle_connections: 5
  connection_max_lifetime: "1h"
```

### Security Hardening

#### Network Security
```bash
# Firewall rules
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow ssh
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow from 10.0.0.0/8 to any port 5432  # Database access
```

#### Application Security
```yaml
# Security headers in Nginx
add_header X-Frame-Options DENY;
add_header X-Content-Type-Options nosniff;
add_header X-XSS-Protection "1; mode=block";
add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
add_header Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'";
```

## 📊 Monitoring & Logging

### Prometheus Metrics

```yaml
# prometheus-config.yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'driftmgr'
    static_configs:
      - targets: ['driftmgr:8080']
    metrics_path: '/metrics'
    scrape_interval: 5s
```

### Grafana Dashboard

```json
{
  "dashboard": {
    "title": "DriftMgr Monitoring",
    "panels": [
      {
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total[5m])",
            "legendFormat": "{{method}} {{endpoint}}"
          }
        ]
      },
      {
        "title": "Response Time",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "95th percentile"
          }
        ]
      }
    ]
  }
}
```

### Log Aggregation

```yaml
# fluentd-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: fluentd-config
data:
  fluent.conf: |
    <source>
      @type tail
      path /var/log/driftmgr/*.log
      pos_file /var/log/fluentd/driftmgr.log.pos
      tag driftmgr.*
      format json
    </source>
    
    <match driftmgr.**>
      @type elasticsearch
      host elasticsearch.logging.svc.cluster.local
      port 9200
      index_name driftmgr
    </match>
```

## 💾 Backup & Recovery

### Database Backup

```bash
#!/bin/bash
# backup.sh

BACKUP_DIR="/opt/backups"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="driftmgr_backup_${DATE}.sql"

# Create backup
pg_dump -h localhost -U driftmgr -d driftmgr > "${BACKUP_DIR}/${BACKUP_FILE}"

# Compress backup
gzip "${BACKUP_DIR}/${BACKUP_FILE}"

# Upload to S3 (optional)
aws s3 cp "${BACKUP_DIR}/${BACKUP_FILE}.gz" s3://your-backup-bucket/

# Cleanup old backups (keep 30 days)
find "${BACKUP_DIR}" -name "driftmgr_backup_*.sql.gz" -mtime +30 -delete
```

### Application Backup

```bash
#!/bin/bash
# app-backup.sh

BACKUP_DIR="/opt/backups"
DATE=$(date +%Y%m%d_%H%M%S)

# Backup configuration
tar -czf "${BACKUP_DIR}/driftmgr_config_${DATE}.tar.gz" /opt/driftmgr/config.yaml

# Backup logs
tar -czf "${BACKUP_DIR}/driftmgr_logs_${DATE}.tar.gz" /var/log/driftmgr/
```

### Recovery Procedures

```bash
# Database recovery
gunzip driftmgr_backup_20250924_100000.sql.gz
psql -h localhost -U driftmgr -d driftmgr < driftmgr_backup_20250924_100000.sql

# Application recovery
tar -xzf driftmgr_config_20250924_100000.tar.gz -C /
systemctl restart driftmgr
```

## 🔒 Security

### SSL/TLS Configuration

```nginx
# Strong SSL configuration
ssl_protocols TLSv1.2 TLSv1.3;
ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512:ECDHE-RSA-AES256-GCM-SHA384:DHE-RSA-AES256-GCM-SHA384;
ssl_prefer_server_ciphers off;
ssl_session_cache shared:SSL:10m;
ssl_session_timeout 10m;
ssl_stapling on;
ssl_stapling_verify on;
```

### Authentication Security

```yaml
# Strong password policy
auth:
  password_min_length: 12
  password_require_uppercase: true
  password_require_lowercase: true
  password_require_numbers: true
  password_require_symbols: true
  password_history: 5
  account_lockout_threshold: 5
  account_lockout_duration: "15m"
```

### Network Security

```bash
# Fail2ban configuration
[driftmgr]
enabled = true
port = 8080
filter = driftmgr
logpath = /var/log/driftmgr/driftmgr.log
maxretry = 5
bantime = 3600
```

## 🔧 Troubleshooting

### Common Issues

#### Service Won't Start
```bash
# Check service status
sudo systemctl status driftmgr

# Check logs
sudo journalctl -u driftmgr -f

# Check configuration
sudo driftmgr-server --config=/opt/driftmgr/config.yaml --validate
```

#### Database Connection Issues
```bash
# Test database connection
psql -h localhost -U driftmgr -d driftmgr -c "SELECT 1;"

# Check PostgreSQL status
sudo systemctl status postgresql

# Check PostgreSQL logs
sudo tail -f /var/log/postgresql/postgresql-15-main.log
```

#### Performance Issues
```bash
# Check system resources
htop
iostat -x 1
free -h

# Check database performance
sudo -u postgres psql -c "SELECT * FROM pg_stat_activity;"
sudo -u postgres psql -c "SELECT * FROM pg_stat_database;"
```

### Health Checks

```bash
# Application health
curl -f http://localhost:8080/health

# Database health
curl -f http://localhost:8080/api/v1/health

# WebSocket health
curl -f http://localhost:8080/api/v1/ws/stats
```

### Log Analysis

```bash
# Search for errors
grep -i error /var/log/driftmgr/driftmgr.log

# Monitor real-time logs
tail -f /var/log/driftmgr/driftmgr.log | grep -i "error\|warn"

# Analyze access patterns
awk '{print $1}' /var/log/nginx/access.log | sort | uniq -c | sort -nr
```

---

This deployment guide provides comprehensive instructions for deploying DriftMgr in various environments. Choose the deployment method that best fits your requirements and infrastructure setup.
