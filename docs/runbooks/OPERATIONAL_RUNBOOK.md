# DriftMgr Operational Runbook

## Table of Contents
1. [System Overview](#system-overview)
2. [Deployment Procedures](#deployment-procedures)
3. [Monitoring & Alerting](#monitoring--alerting)
4. [Incident Response](#incident-response)
5. [Performance Tuning](#performance-tuning)
6. [Troubleshooting Guide](#troubleshooting-guide)
7. [Disaster Recovery](#disaster-recovery)
8. [Maintenance Procedures](#maintenance-procedures)

---

## System Overview

### Architecture
```
┌─────────────┐     ┌──────────────┐     ┌────────────┐
│   Clients   │────▶│  API Gateway │────▶│  DriftMgr  │
└─────────────┘     └──────────────┘     └────────────┘
                                               │
                    ┌──────────────────────────┼──────────────────────────┐
                    │                          │                          │
              ┌─────▼─────┐           ┌───────▼──────┐           ┌───────▼──────┐
              │   etcd    │           │    Cache     │           │   Database   │
              └───────────┘           └──────────────┘           └──────────────┘
                                               │
                    ┌──────────────────────────┼──────────────────────────┐
                    │                          │                          │
              ┌─────▼─────┐           ┌───────▼──────┐           ┌───────▼──────┐
              │    AWS    │           │    Azure     │           │     GCP      │
              └───────────┘           └──────────────┘           └──────────────┘
```

### Key Components
- **API Gateway**: Handles routing, authentication, rate limiting
- **DriftMgr Core**: Main application logic
- **etcd**: Distributed state management
- **Cache**: Redis/in-memory caching layer
- **Database**: PostgreSQL for persistent storage
- **Cloud Providers**: AWS, Azure, GCP, DigitalOcean APIs

### Dependencies
- Go 1.21+
- etcd 3.5+
- PostgreSQL 14+ (optional)
- Redis 7+ (optional)

---

## Deployment Procedures

### Pre-Deployment Checklist
- [ ] All tests passing (`make test`)
- [ ] Security scan completed (`make security-scan`)
- [ ] Configuration validated (`driftmgr validate-config`)
- [ ] Database migrations ready
- [ ] Rollback plan prepared
- [ ] Monitoring alerts configured
- [ ] Team notified of deployment window

### Deployment Steps

#### 1. Blue-Green Deployment
```bash
# 1. Deploy to green environment
kubectl apply -f deployments/k8s/green/

# 2. Run smoke tests
./scripts/smoke-test.sh green

# 3. Switch traffic to green
kubectl patch service driftmgr -p '{"spec":{"selector":{"version":"green"}}}'

# 4. Monitor for 10 minutes
watch -n 5 'kubectl get pods -l app=driftmgr'

# 5. If successful, scale down blue
kubectl scale deployment driftmgr-blue --replicas=0
```

#### 2. Canary Deployment
```bash
# 1. Deploy canary version (10% traffic)
kubectl apply -f deployments/k8s/canary/

# 2. Monitor metrics
./scripts/monitor-canary.sh

# 3. Gradually increase traffic
for percent in 25 50 75 100; do
    kubectl set env deployment/driftmgr CANARY_WEIGHT=$percent
    sleep 300  # Wait 5 minutes
    ./scripts/check-metrics.sh || exit 1
done
```

### Rollback Procedure
```bash
# Immediate rollback
kubectl rollout undo deployment/driftmgr

# Rollback to specific version
kubectl rollout undo deployment/driftmgr --to-revision=3

# Verify rollback
kubectl rollout status deployment/driftmgr
```

---

## Monitoring & Alerting

### Key Metrics to Monitor

#### Application Metrics
| Metric | Threshold | Alert Severity |
|--------|-----------|----------------|
| Response Time P95 | > 2s | Warning |
| Response Time P99 | > 5s | Critical |
| Error Rate | > 1% | Warning |
| Error Rate | > 5% | Critical |
| Discovery Duration | > 30s | Warning |
| Drift Detection Duration | > 10s | Warning |

#### System Metrics
| Metric | Threshold | Alert Severity |
|--------|-----------|----------------|
| CPU Usage | > 80% | Warning |
| Memory Usage | > 85% | Critical |
| Disk Usage | > 90% | Critical |
| Open File Descriptors | > 80% limit | Warning |

#### Circuit Breaker Metrics
| Metric | Threshold | Alert Severity |
|--------|-----------|----------------|
| Circuit Open Count | > 0 | Warning |
| Circuit Open Count | > 2 | Critical |
| Failure Rate | > 50% | Critical |

### Monitoring Commands

```bash
# Check application health
curl -s http://localhost:8080/health | jq '.'

# View real-time metrics
curl -s http://localhost:8080/metrics | grep driftmgr

# Check circuit breaker status
curl -s http://localhost:8080/api/v1/circuit-breakers | jq '.'

# View cache statistics
curl -s http://localhost:8080/api/v1/cache/stats | jq '.'

# Check rate limiter status
curl -s http://localhost:8080/api/v1/rate-limiters | jq '.'
```

### Alert Response

#### High Error Rate Alert
1. Check recent deployments
2. Review error logs: `kubectl logs -l app=driftmgr --tail=100`
3. Check circuit breaker status
4. Verify cloud provider APIs are accessible
5. Scale up if load-related
6. Consider rolling back if deployment-related

#### High Latency Alert
1. Check current load: `kubectl top pods`
2. Review slow query logs
3. Check cache hit rates
4. Verify network connectivity to cloud providers
5. Scale horizontally if needed

---

## Incident Response

### Incident Severity Levels

| Level | Description | Response Time | Examples |
|-------|-------------|---------------|----------|
| P1 | Critical - Service Down | 15 minutes | Complete outage, data loss |
| P2 | Major - Degraded Service | 30 minutes | Partial outage, high error rate |
| P3 | Minor - Feature Impact | 2 hours | Single feature broken |
| P4 | Low - Cosmetic Issue | Next business day | UI glitch, typo |

### Incident Response Playbooks

#### Playbook: Complete Service Outage (P1)

**Symptoms:**
- All health checks failing
- No API responses
- Multiple alerts firing

**Immediate Actions:**
1. **Notify** incident commander and stakeholders
2. **Check** infrastructure status:
   ```bash
   kubectl get pods -l app=driftmgr
   kubectl describe pods -l app=driftmgr
   kubectl get events --sort-by='.lastTimestamp'
   ```

3. **Restart** if necessary:
   ```bash
   kubectl rollout restart deployment/driftmgr
   ```

4. **Scale** to handle load:
   ```bash
   kubectl scale deployment/driftmgr --replicas=10
   ```

5. **Enable** emergency mode (read-only):
   ```bash
   kubectl set env deployment/driftmgr EMERGENCY_MODE=true
   ```

**Root Cause Analysis:**
- Collect logs from last hour
- Review recent changes
- Check dependency services
- Analyze metrics leading to incident

#### Playbook: High Memory Usage (P2)

**Symptoms:**
- Memory usage > 85%
- Increasing GC pause times
- OOM kills

**Actions:**
1. **Capture** heap profile:
   ```bash
   curl http://localhost:8080/debug/pprof/heap > heap.prof
   go tool pprof heap.prof
   ```

2. **Identify** memory leaks:
   ```bash
   # Compare heap profiles
   go tool pprof -base=heap1.prof heap2.prof
   ```

3. **Temporary mitigation:**
   ```bash
   # Increase memory limits
   kubectl set resources deployment/driftmgr -c=driftmgr --limits=memory=4Gi
   
   # Force garbage collection
   curl -X POST http://localhost:8080/admin/gc
   ```

4. **Long-term fix:**
   - Identify and fix memory leak
   - Optimize caching strategy
   - Implement memory circuit breaker

---

## Performance Tuning

### Configuration Optimization

```yaml
# config/production.yaml
performance:
  # Connection pooling
  database:
    max_connections: 100
    max_idle: 10
    connection_timeout: 5s
  
  # Caching
  cache:
    ttl: 5m
    max_size: 1000
    eviction_policy: lru
  
  # Rate limiting
  rate_limits:
    aws: 20  # requests per second
    azure: 15
    gcp: 20
    digitalocean: 10
  
  # Circuit breakers
  circuit_breakers:
    max_failures: 5
    reset_timeout: 30s
    half_open_requests: 3
  
  # Concurrency
  workers:
    discovery: 10
    drift_detection: 5
    remediation: 3
```

### Database Optimization

```sql
-- Add indexes for common queries
CREATE INDEX idx_resources_provider ON resources(provider);
CREATE INDEX idx_resources_created_at ON resources(created_at);
CREATE INDEX idx_drift_results_timestamp ON drift_results(timestamp);

-- Analyze query performance
EXPLAIN ANALYZE SELECT * FROM resources WHERE provider = 'aws';

-- Vacuum and analyze tables
VACUUM ANALYZE resources;
VACUUM ANALYZE drift_results;
```

### Cache Warming

```bash
# Warm cache on startup
./scripts/warm-cache.sh

# Schedule periodic cache warming
*/15 * * * * /usr/local/bin/driftmgr cache-warm --providers=aws,azure
```

---

## Troubleshooting Guide

### Common Issues and Solutions

#### Issue: Discovery Timeout
**Symptoms:** Discovery operations timing out

**Diagnosis:**
```bash
# Check provider connectivity
driftmgr test-connection --provider=aws

# Review discovery logs
grep "discovery" /var/log/driftmgr/app.log | tail -100

# Check rate limits
curl http://localhost:8080/api/v1/rate-limiters
```

**Solutions:**
1. Increase timeout: `DISCOVERY_TIMEOUT=60s`
2. Reduce parallel workers: `DISCOVERY_WORKERS=5`
3. Enable retry with backoff
4. Check cloud provider quotas

#### Issue: High Drift False Positives
**Symptoms:** Drift detected when infrastructure unchanged

**Diagnosis:**
```bash
# Compare state with reality
driftmgr drift detect --verbose --provider=aws

# Check ignore rules
cat configs/drift-ignore.yaml
```

**Solutions:**
1. Update ignore patterns
2. Adjust drift sensitivity thresholds
3. Exclude volatile attributes
4. Update provider SDK versions

#### Issue: Circuit Breaker Open
**Symptoms:** Requests failing with "circuit breaker open"

**Diagnosis:**
```bash
# Check circuit breaker status
curl http://localhost:8080/api/v1/circuit-breakers

# Review error logs
grep "circuit" /var/log/driftmgr/app.log | tail -50
```

**Solutions:**
1. Identify root cause of failures
2. Manually reset if false trigger:
   ```bash
   curl -X POST http://localhost:8080/api/v1/circuit-breakers/reset
   ```
3. Adjust circuit breaker thresholds
4. Implement fallback mechanisms

### Debug Commands

```bash
# Enable debug logging
export DRIFTMGR_LOG_LEVEL=DEBUG

# Trace specific operation
driftmgr discover --trace --provider=aws

# Profile CPU usage
curl http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof

# Dump goroutines
curl http://localhost:8080/debug/pprof/goroutine?debug=2

# Check file descriptors
lsof -p $(pgrep driftmgr) | wc -l
```

---

## Disaster Recovery

### Backup Procedures

#### Daily Backups
```bash
#!/bin/bash
# backup.sh - Run daily at 2 AM

# Backup database
pg_dump driftmgr > /backup/db/driftmgr-$(date +%Y%m%d).sql

# Backup etcd
etcdctl snapshot save /backup/etcd/snapshot-$(date +%Y%m%d).db

# Backup configuration
tar -czf /backup/config/config-$(date +%Y%m%d).tar.gz /etc/driftmgr/

# Upload to S3
aws s3 sync /backup/ s3://driftmgr-backups/$(date +%Y%m%d)/

# Cleanup old backups (keep 30 days)
find /backup -type f -mtime +30 -delete
```

### Recovery Procedures

#### Database Recovery
```bash
# Restore from backup
pg_restore -d driftmgr /backup/db/driftmgr-20240315.sql

# Verify integrity
psql -d driftmgr -c "SELECT COUNT(*) FROM resources;"
```

#### etcd Recovery
```bash
# Restore snapshot
etcdctl snapshot restore snapshot.db \
  --data-dir=/var/lib/etcd-recovery \
  --initial-cluster=etcd-0=http://etcd-0:2380

# Start etcd with recovered data
etcd --data-dir=/var/lib/etcd-recovery
```

### Business Continuity Plan

#### RTO (Recovery Time Objective): 1 hour
#### RPO (Recovery Point Objective): 24 hours

**Failover Steps:**
1. Activate DR site
2. Update DNS to point to DR
3. Restore latest backups
4. Verify system functionality
5. Notify stakeholders

---

## Maintenance Procedures

### Regular Maintenance Tasks

#### Daily
- [ ] Review error logs
- [ ] Check backup completion
- [ ] Monitor resource usage trends

#### Weekly
- [ ] Review and acknowledge alerts
- [ ] Update dependencies if security patches available
- [ ] Clean up old logs and temporary files
- [ ] Review circuit breaker and rate limiter statistics

#### Monthly
- [ ] Performance review meeting
- [ ] Capacity planning review
- [ ] Security audit
- [ ] Disaster recovery drill
- [ ] Update documentation

### Maintenance Mode

```bash
# Enable maintenance mode
kubectl set env deployment/driftmgr MAINTENANCE_MODE=true

# Display maintenance message
kubectl set env deployment/driftmgr MAINTENANCE_MESSAGE="Scheduled maintenance until 3 PM UTC"

# Disable after maintenance
kubectl set env deployment/driftmgr MAINTENANCE_MODE=false
```

### Log Rotation

```yaml
# /etc/logrotate.d/driftmgr
/var/log/driftmgr/*.log {
    daily
    rotate 30
    compress
    delaycompress
    notifempty
    create 0640 driftmgr driftmgr
    sharedscripts
    postrotate
        kill -USR1 $(cat /var/run/driftmgr.pid)
    endscript
}
```

---

## Contact Information

### Escalation Path
1. **L1 Support**: support@driftmgr.io
2. **L2 Engineering**: eng-oncall@driftmgr.io
3. **L3 Architecture**: architects@driftmgr.io
4. **Management**: cto@driftmgr.io

### On-Call Rotation
- Primary: Check PagerDuty
- Secondary: Check PagerDuty
- Escalation: Engineering Manager

### External Dependencies
- **AWS Support**: Premium support plan
- **Azure Support**: Professional Direct
- **GCP Support**: Production SLA
- **Database Vendor**: 24/7 support contract

---

## Appendix

### Useful Links
- [Architecture Documentation](../architecture/README.md)
- [API Documentation](../api/README.md)
- [Security Procedures](../security/README.md)
- [Development Guide](../development/README.md)

### Emergency Contacts
- Infrastructure Team: +1-555-INFRA-911
- Security Team: security@driftmgr.io
- Legal: legal@driftmgr.io

### Change Log
- 2024-03-15: Initial runbook creation
- 2024-03-16: Added disaster recovery procedures
- 2024-03-17: Updated monitoring thresholds
- 2024-03-18: Added performance tuning section