# DriftMgr Troubleshooting Guide

This guide helps you diagnose and resolve common issues with DriftMgr deployments.

## 📋 Table of Contents

- [Quick Diagnostics](#quick-diagnostics)
- [Common Issues](#common-issues)
- [Server Issues](#server-issues)
- [Database Issues](#database-issues)
- [Authentication Issues](#authentication-issues)
- [WebSocket Issues](#websocket-issues)
- [Performance Issues](#performance-issues)
- [Network Issues](#network-issues)
- [Log Analysis](#log-analysis)
- [Debugging Tools](#debugging-tools)

## 🔍 Quick Diagnostics

### Health Check Commands

```bash
# Basic health check
curl -f http://localhost:8080/health

# Detailed health information
curl -f http://localhost:8080/api/v1/health

# WebSocket connection test
curl -i -N -H "Connection: Upgrade" \
     -H "Upgrade: websocket" \
     -H "Sec-WebSocket-Key: test" \
     -H "Sec-WebSocket-Version: 13" \
     http://localhost:8080/ws

# Check server status
systemctl status driftmgr

# Check database connectivity
psql -h localhost -U driftmgr -d driftmgr -c "SELECT 1;"
```

### System Resource Check

```bash
# Check system resources
htop
free -h
df -h
iostat -x 1

# Check network connectivity
netstat -tlnp | grep :8080
ss -tlnp | grep :8080

# Check process information
ps aux | grep driftmgr
lsof -i :8080
```

## 🚨 Common Issues

### Issue: Server Won't Start

#### Symptoms
- Server fails to start with error messages
- Port binding errors
- Configuration validation failures

#### Diagnosis
```bash
# Check if port is already in use
sudo netstat -tlnp | grep :8080
sudo lsof -i :8080

# Check configuration file
driftmgr-server --config=config.yaml --validate

# Check logs
journalctl -u driftmgr -f
tail -f /var/log/driftmgr/driftmgr.log
```

#### Solutions

**Port Already in Use:**
```bash
# Find process using port 8080
sudo lsof -i :8080

# Kill the process
sudo kill -9 <PID>

# Or use a different port
driftmgr-server --port 8081
```

**Configuration Issues:**
```bash
# Validate configuration
driftmgr-server --config=config.yaml --validate

# Check file permissions
ls -la config.yaml
chmod 644 config.yaml
```

**Permission Issues:**
```bash
# Check file ownership
ls -la /opt/driftmgr/
sudo chown -R driftmgr:driftmgr /opt/driftmgr/

# Check log directory permissions
sudo mkdir -p /var/log/driftmgr
sudo chown driftmgr:driftmgr /var/log/driftmgr
```

### Issue: Database Connection Failed

#### Symptoms
- "database connection failed" errors
- Authentication failures
- Connection timeout errors

#### Diagnosis
```bash
# Test database connectivity
psql -h localhost -U driftmgr -d driftmgr -c "SELECT 1;"

# Check PostgreSQL status
systemctl status postgresql

# Check PostgreSQL logs
tail -f /var/log/postgresql/postgresql-15-main.log

# Check network connectivity
telnet localhost 5432
```

#### Solutions

**PostgreSQL Not Running:**
```bash
# Start PostgreSQL
sudo systemctl start postgresql
sudo systemctl enable postgresql

# Check PostgreSQL configuration
sudo -u postgres psql -c "SHOW config_file;"
```

**Authentication Issues:**
```bash
# Check user exists
sudo -u postgres psql -c "SELECT usename FROM pg_user WHERE usename = 'driftmgr';"

# Create user if missing
sudo -u postgres psql -c "CREATE USER driftmgr WITH PASSWORD 'secure-password';"

# Grant permissions
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE driftmgr TO driftmgr;"
```

**Database Doesn't Exist:**
```bash
# Create database
sudo -u postgres psql -c "CREATE DATABASE driftmgr OWNER driftmgr;"

# Run migrations
driftmgr-server --migrate
```

**Connection Pool Exhausted:**
```yaml
# config.yaml
database:
  max_open_connections: 10  # Reduce from default 25
  max_idle_connections: 2   # Reduce from default 5
  connection_max_lifetime: "30m"  # Reduce from 1h
```

### Issue: Authentication Failures

#### Symptoms
- "invalid credentials" errors
- JWT token validation failures
- User registration failures

#### Diagnosis
```bash
# Check JWT secret configuration
echo $DRIFTMGR_JWT_SECRET

# Test user registration
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"test","email":"test@example.com","password":"Test123!","first_name":"Test","last_name":"User"}'

# Test user login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"Test123!"}'
```

#### Solutions

**JWT Secret Not Set:**
```bash
# Generate JWT secret
openssl rand -base64 32

# Set environment variable
export DRIFTMGR_JWT_SECRET="your-generated-secret"

# Or update config file
# config.yaml
auth:
  jwt_secret: "your-generated-secret"
```

**Password Validation Issues:**
```bash
# Check password requirements
# Password must contain:
# - At least 8 characters
# - At least one uppercase letter
# - At least one lowercase letter
# - At least one number
# - At least one special character

# Example valid password: "SecurePass123!"
```

**User Already Exists:**
```bash
# Check if user exists
psql -h localhost -U driftmgr -d driftmgr -c "SELECT username FROM users WHERE username = 'testuser';"

# Delete user if needed
psql -h localhost -U driftmgr -d driftmgr -c "DELETE FROM users WHERE username = 'testuser';"
```

## 🖥️ Server Issues

### Issue: High Memory Usage

#### Symptoms
- Server consuming excessive memory
- Out of memory errors
- Slow response times

#### Diagnosis
```bash
# Check memory usage
free -h
ps aux --sort=-%mem | head -10

# Check for memory leaks
go tool pprof http://localhost:6060/debug/pprof/heap

# Monitor memory over time
watch -n 1 'ps aux | grep driftmgr'
```

#### Solutions

**Memory Leak Investigation:**
```bash
# Enable memory profiling
# Add to main.go
import _ "net/http/pprof"

go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()

# Generate heap profile
go tool pprof http://localhost:6060/debug/pprof/heap
```

**Reduce Memory Usage:**
```yaml
# config.yaml
server:
  max_header_bytes: 1048576  # 1MB instead of default
  read_timeout: "30s"
  write_timeout: "30s"
  idle_timeout: "60s"  # Reduce from 120s

database:
  max_open_connections: 10  # Reduce from 25
  max_idle_connections: 2   # Reduce from 5
```

### Issue: High CPU Usage

#### Symptoms
- Server consuming high CPU
- Slow response times
- System becoming unresponsive

#### Diagnosis
```bash
# Check CPU usage
top -p $(pgrep driftmgr)
htop

# Profile CPU usage
go tool pprof http://localhost:6060/debug/pprof/profile

# Check for infinite loops
strace -p $(pgrep driftmgr)
```

#### Solutions

**CPU Profiling:**
```bash
# Generate CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile

# Analyze profile
(pprof) top10
(pprof) list functionName
(pprof) web
```

**Optimize Database Queries:**
```sql
-- Check slow queries
SELECT query, mean_time, calls 
FROM pg_stat_statements 
ORDER BY mean_time DESC 
LIMIT 10;

-- Add indexes for frequently queried columns
CREATE INDEX CONCURRENTLY idx_users_username ON users(username);
CREATE INDEX CONCURRENTLY idx_resources_provider ON resources(provider);
```

### Issue: Server Crashes

#### Symptoms
- Server process terminates unexpectedly
- Core dumps generated
- Service restarting frequently

#### Diagnosis
```bash
# Check system logs
journalctl -u driftmgr --since "1 hour ago"

# Check for core dumps
ls -la /var/crash/
ls -la core.*

# Check system resources
dmesg | tail -20
```

#### Solutions

**Panic Recovery:**
```go
// Add panic recovery to main function
defer func() {
    if r := recover(); r != nil {
        log.Printf("Panic recovered: %v", r)
        // Log stack trace
        debug.PrintStack()
    }
}()
```

**Resource Limits:**
```bash
# Set systemd resource limits
# /etc/systemd/system/driftmgr.service
[Service]
LimitNOFILE=65536
LimitNPROC=32768
MemoryLimit=2G
```

## 🗄️ Database Issues

### Issue: Slow Database Queries

#### Symptoms
- Slow API responses
- Database connection timeouts
- High database CPU usage

#### Diagnosis
```sql
-- Check active queries
SELECT pid, now() - pg_stat_activity.query_start AS duration, query 
FROM pg_stat_activity 
WHERE (now() - pg_stat_activity.query_start) > interval '5 minutes';

-- Check slow queries
SELECT query, mean_time, calls, total_time
FROM pg_stat_statements 
ORDER BY mean_time DESC 
LIMIT 10;

-- Check database size
SELECT pg_size_pretty(pg_database_size('driftmgr'));

-- Check table sizes
SELECT schemaname,tablename,pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables 
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

#### Solutions

**Add Indexes:**
```sql
-- Add indexes for frequently queried columns
CREATE INDEX CONCURRENTLY idx_users_username ON users(username);
CREATE INDEX CONCURRENTLY idx_users_email ON users(email);
CREATE INDEX CONCURRENTLY idx_resources_provider ON resources(provider);
CREATE INDEX CONCURRENTLY idx_resources_type ON resources(type);
CREATE INDEX CONCURRENTLY idx_drift_results_status ON drift_results(status);
CREATE INDEX CONCURRENTLY idx_drift_results_created_at ON drift_results(created_at);

-- Composite indexes for common query patterns
CREATE INDEX CONCURRENTLY idx_resources_provider_type ON resources(provider, type);
CREATE INDEX CONCURRENTLY idx_drift_results_resource_status ON drift_results(resource_id, status);
```

**Optimize Queries:**
```sql
-- Use EXPLAIN ANALYZE to understand query plans
EXPLAIN ANALYZE SELECT * FROM users WHERE username = 'testuser';

-- Use prepared statements for repeated queries
PREPARE get_user_by_username(text) AS 
SELECT * FROM users WHERE username = $1;
```

**Database Configuration:**
```sql
-- Optimize PostgreSQL configuration
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET effective_cache_size = '1GB';
ALTER SYSTEM SET maintenance_work_mem = '64MB';
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
ALTER SYSTEM SET wal_buffers = '16MB';
ALTER SYSTEM SET default_statistics_target = 100;
SELECT pg_reload_conf();
```

### Issue: Database Connection Pool Exhausted

#### Symptoms
- "too many connections" errors
- Connection timeout errors
- Database becoming unresponsive

#### Diagnosis
```sql
-- Check current connections
SELECT count(*) FROM pg_stat_activity;

-- Check connection limits
SHOW max_connections;

-- Check connection by database
SELECT datname, count(*) 
FROM pg_stat_activity 
GROUP BY datname;
```

#### Solutions

**Reduce Connection Pool Size:**
```yaml
# config.yaml
database:
  max_open_connections: 10  # Reduce from 25
  max_idle_connections: 2   # Reduce from 5
  connection_max_lifetime: "30m"  # Reduce from 1h
```

**Increase PostgreSQL Connection Limit:**
```sql
-- Increase max_connections
ALTER SYSTEM SET max_connections = 200;

-- Restart PostgreSQL
sudo systemctl restart postgresql
```

**Connection Pooling:**
```bash
# Use PgBouncer for connection pooling
sudo apt install pgbouncer

# Configure PgBouncer
# /etc/pgbouncer/pgbouncer.ini
[databases]
driftmgr = host=localhost port=5432 dbname=driftmgr

[pgbouncer]
listen_port = 6432
listen_addr = 127.0.0.1
auth_type = md5
auth_file = /etc/pgbouncer/userlist.txt
pool_mode = transaction
max_client_conn = 100
default_pool_size = 20
```

## 🔐 Authentication Issues

### Issue: JWT Token Validation Failures

#### Symptoms
- "invalid token" errors
- Token expiration issues
- Authentication middleware failures

#### Diagnosis
```bash
# Check JWT secret consistency
echo $DRIFTMGR_JWT_SECRET

# Decode JWT token (for debugging)
echo "your-jwt-token" | cut -d. -f2 | base64 -d

# Check token expiration
curl -H "Authorization: Bearer <token>" http://localhost:8080/api/v1/auth/profile
```

#### Solutions

**JWT Secret Mismatch:**
```bash
# Ensure JWT secret is consistent across all instances
export DRIFTMGR_JWT_SECRET="your-secret-key"

# Or use the same secret in config file
# config.yaml
auth:
  jwt_secret: "your-secret-key"
```

**Token Expiration:**
```yaml
# config.yaml
auth:
  access_token_expiry: "15m"   # Increase if needed
  refresh_token_expiry: "7d"   # Increase if needed
```

**Clock Skew Issues:**
```bash
# Synchronize system time
sudo ntpdate -s time.nist.gov

# Or use chrony
sudo chrony sources -v
```

### Issue: Password Hashing Problems

#### Symptoms
- Password validation failures
- Argon2 errors
- Authentication panics

#### Diagnosis
```bash
# Check password service logs
grep -i "password\|argon" /var/log/driftmgr/driftmgr.log

# Test password hashing manually
go run -c 'package main; import "fmt"; import "github.com/catherinevee/driftmgr/internal/auth"; func main() { s := auth.NewPasswordService(); h, err := s.HashPassword("test123"); fmt.Println(h, err) }'
```

#### Solutions

**Argon2 Configuration:**
```go
// Reduce Argon2 parameters for better compatibility
func NewPasswordService() *PasswordService {
    return &PasswordService{
        memory:      32 * 1024, // Reduce from 64MB
        iterations:  2,         // Reduce from 3
        parallelism: 1,         // Reduce from 2
        saltLength:  16,
        keyLength:   32,
    }
}
```

**Password Validation:**
```go
// Add password validation before hashing
func (p *PasswordService) HashPassword(password string) (string, error) {
    if len(password) < 8 {
        return "", fmt.Errorf("password too short")
    }
    
    // Rest of implementation
}
```

## 🔌 WebSocket Issues

### Issue: WebSocket Connection Failures

#### Symptoms
- WebSocket connections dropping
- Connection upgrade failures
- Real-time updates not working

#### Diagnosis
```bash
# Test WebSocket connection
curl -i -N -H "Connection: Upgrade" \
     -H "Upgrade: websocket" \
     -H "Sec-WebSocket-Key: test" \
     -H "Sec-WebSocket-Version: 13" \
     http://localhost:8080/ws

# Check WebSocket stats
curl http://localhost:8080/api/v1/ws/stats

# Check firewall rules
sudo ufw status
sudo iptables -L
```

#### Solutions

**Firewall Configuration:**
```bash
# Allow WebSocket connections
sudo ufw allow 8080/tcp

# Check for proxy interference
# Ensure proxy supports WebSocket upgrades
```

**Nginx Configuration:**
```nginx
# Ensure proper WebSocket configuration
location /ws {
    proxy_pass http://127.0.0.1:8080;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_read_timeout 86400;
}
```

**WebSocket Timeout:**
```go
// Increase WebSocket timeout
const (
    writeWait = 10 * time.Second
    pongWait = 60 * time.Second
    pingPeriod = (pongWait * 9) / 10
    maxMessageSize = 512
)
```

### Issue: WebSocket Memory Leaks

#### Symptoms
- Increasing memory usage over time
- WebSocket connections not being cleaned up
- Server becoming unresponsive

#### Diagnosis
```bash
# Check WebSocket connection count
curl http://localhost:8080/api/v1/ws/stats

# Monitor memory usage
watch -n 1 'ps aux | grep driftmgr'

# Check for goroutine leaks
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

#### Solutions

**Connection Cleanup:**
```go
// Ensure proper connection cleanup
func (c *Client) readPump() {
    defer func() {
        c.hub.unregister <- c
        c.conn.Close()
    }()
    
    // Rest of implementation
}

func (c *Client) writePump() {
    defer func() {
        c.conn.Close()
    }()
    
    // Rest of implementation
}
```

**Connection Limits:**
```go
// Add connection limits
const MaxConnections = 1000

func (h *Hub) registerClient(client *Client) {
    if len(h.clients) >= MaxConnections {
        client.conn.Close()
        return
    }
    
    h.clients[client] = true
}
```

## ⚡ Performance Issues

### Issue: Slow API Responses

#### Symptoms
- High response times
- Timeout errors
- Poor user experience

#### Diagnosis
```bash
# Test API response times
time curl http://localhost:8080/api/v1/health

# Monitor response times
curl -w "@curl-format.txt" -o /dev/null -s http://localhost:8080/api/v1/resources

# Check server metrics
curl http://localhost:8080/metrics
```

#### Solutions

**Database Query Optimization:**
```sql
-- Add indexes for slow queries
CREATE INDEX CONCURRENTLY idx_resources_provider_type ON resources(provider, type);
CREATE INDEX CONCURRENTLY idx_users_username ON users(username);

-- Use prepared statements
PREPARE get_resources(text, text) AS 
SELECT * FROM resources WHERE provider = $1 AND type = $2;
```

**Response Caching:**
```go
// Add response caching
import "github.com/patrickmn/go-cache"

var c = cache.New(5*time.Minute, 10*time.Minute)

func (h *Handler) ListResources(w http.ResponseWriter, r *http.Request) {
    // Check cache first
    if cached, found := c.Get("resources"); found {
        json.NewEncoder(w).Encode(cached)
        return
    }
    
    // Fetch from database
    resources, err := h.service.ListResources()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Cache the result
    c.Set("resources", resources, cache.DefaultExpiration)
    
    json.NewEncoder(w).Encode(resources)
}
```

**Connection Pooling:**
```yaml
# config.yaml
database:
  max_open_connections: 25
  max_idle_connections: 5
  connection_max_lifetime: "1h"
```

### Issue: High Memory Usage

#### Symptoms
- Memory usage growing over time
- Out of memory errors
- System becoming unresponsive

#### Diagnosis
```bash
# Check memory usage
free -h
ps aux --sort=-%mem | head -10

# Generate memory profile
go tool pprof http://localhost:6060/debug/pprof/heap

# Check for memory leaks
go tool pprof -alloc_space http://localhost:6060/debug/pprof/heap
```

#### Solutions

**Memory Profiling:**
```bash
# Generate heap profile
go tool pprof http://localhost:6060/debug/pprof/heap

# Analyze memory usage
(pprof) top10
(pprof) list functionName
(pprof) web
```

**Reduce Memory Usage:**
```go
// Use object pooling
var userPool = sync.Pool{
    New: func() interface{} {
        return &User{}
    },
}

func GetUser() *User {
    return userPool.Get().(*User)
}

func PutUser(user *User) {
    // Reset fields
    *user = User{}
    userPool.Put(user)
}
```

**Garbage Collection Tuning:**
```bash
# Set GC target percentage
export GOGC=100

# Or set in code
debug.SetGCPercent(100)
```

## 🌐 Network Issues

### Issue: Connection Timeouts

#### Symptoms
- Request timeouts
- Connection refused errors
- Intermittent connectivity issues

#### Diagnosis
```bash
# Test network connectivity
ping localhost
telnet localhost 8080

# Check network statistics
netstat -i
ss -s

# Check for network errors
dmesg | grep -i network
```

#### Solutions

**Increase Timeouts:**
```yaml
# config.yaml
server:
  read_timeout: "60s"    # Increase from 30s
  write_timeout: "60s"   # Increase from 30s
  idle_timeout: "300s"   # Increase from 120s
```

**Network Configuration:**
```bash
# Increase network buffer sizes
echo 'net.core.rmem_max = 16777216' >> /etc/sysctl.conf
echo 'net.core.wmem_max = 16777216' >> /etc/sysctl.conf
echo 'net.ipv4.tcp_rmem = 4096 87380 16777216' >> /etc/sysctl.conf
echo 'net.ipv4.tcp_wmem = 4096 65536 16777216' >> /etc/sysctl.conf
sysctl -p
```

### Issue: SSL/TLS Problems

#### Symptoms
- SSL handshake failures
- Certificate validation errors
- HTTPS connection issues

#### Diagnosis
```bash
# Test SSL connection
openssl s_client -connect your-domain.com:443

# Check certificate validity
openssl x509 -in /path/to/cert.pem -text -noout

# Test with curl
curl -v https://your-domain.com/health
```

#### Solutions

**Certificate Issues:**
```bash
# Renew Let's Encrypt certificate
sudo certbot renew

# Check certificate expiration
openssl x509 -in /etc/letsencrypt/live/your-domain.com/cert.pem -noout -dates
```

**SSL Configuration:**
```nginx
# Strong SSL configuration
ssl_protocols TLSv1.2 TLSv1.3;
ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512;
ssl_prefer_server_ciphers off;
ssl_session_cache shared:SSL:10m;
ssl_session_timeout 10m;
```

## 📊 Log Analysis

### Log Locations

```bash
# Application logs
/var/log/driftmgr/driftmgr.log

# System logs
journalctl -u driftmgr

# Nginx logs
/var/log/nginx/access.log
/var/log/nginx/error.log

# PostgreSQL logs
/var/log/postgresql/postgresql-15-main.log
```

### Log Analysis Commands

```bash
# Search for errors
grep -i error /var/log/driftmgr/driftmgr.log

# Monitor real-time logs
tail -f /var/log/driftmgr/driftmgr.log | grep -i "error\|warn"

# Count error types
grep -i error /var/log/driftmgr/driftmgr.log | awk '{print $4}' | sort | uniq -c

# Analyze access patterns
awk '{print $1}' /var/log/nginx/access.log | sort | uniq -c | sort -nr

# Check response codes
awk '{print $9}' /var/log/nginx/access.log | sort | uniq -c
```

### Log Rotation

```bash
# Configure logrotate
sudo tee /etc/logrotate.d/driftmgr > /dev/null <<EOF
/var/log/driftmgr/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 644 driftmgr driftmgr
    postrotate
        systemctl reload driftmgr
    endscript
}
EOF
```

## 🛠️ Debugging Tools

### Profiling Tools

```bash
# Install profiling tools
go install github.com/google/pprof@latest

# CPU profiling
go tool pprof http://localhost:6060/debug/pprof/profile

# Memory profiling
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutine profiling
go tool pprof http://localhost:6060/debug/pprof/goroutine

# Block profiling
go tool pprof http://localhost:6060/debug/pprof/block
```

### Monitoring Tools

```bash
# Install monitoring tools
go install github.com/prometheus/prometheus@latest
go install github.com/grafana/grafana@latest

# Start Prometheus
prometheus --config.file=prometheus.yml

# Start Grafana
grafana-server
```

### Database Tools

```bash
# Install database tools
sudo apt install postgresql-client-common postgresql-client

# Connect to database
psql -h localhost -U driftmgr -d driftmgr

# Database monitoring
sudo -u postgres psql -c "SELECT * FROM pg_stat_activity;"
sudo -u postgres psql -c "SELECT * FROM pg_stat_database;"
```

### Network Tools

```bash
# Install network tools
sudo apt install net-tools tcpdump wireshark

# Monitor network traffic
sudo tcpdump -i any port 8080

# Check network connections
netstat -tlnp | grep :8080
ss -tlnp | grep :8080
```

---

This troubleshooting guide covers the most common issues you might encounter with DriftMgr. For additional support, check the main documentation or create an issue on GitHub.
