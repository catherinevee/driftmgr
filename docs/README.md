# DriftMgr Documentation

Welcome to the DriftMgr documentation! This comprehensive guide covers everything you need to know about DriftMgr, from installation and configuration to advanced usage and deployment.

## 📚 Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Installation](#installation)
- [Configuration](#configuration)
- [API Reference](#api-reference)
- [Web Dashboard](#web-dashboard)
- [Authentication](#authentication)
- [WebSocket API](#websocket-api)
- [Deployment](#deployment)
- [Development](#development)
- [Testing](#testing)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)

## 🌟 Overview

DriftMgr is a comprehensive cloud resource drift detection and remediation platform designed to help organizations maintain consistency between their Terraform-managed infrastructure and actual cloud resources.

### Key Features

- **🔍 Drift Detection**: Automatically detect discrepancies between Terraform state and actual cloud resources
- **🔧 Remediation**: Automated and manual remediation strategies for detected drift
- **🌐 Multi-Cloud Support**: Support for AWS, Azure, GCP, and DigitalOcean
- **📊 Real-time Dashboard**: Web-based dashboard with live updates via WebSocket
- **🔐 Authentication & Authorization**: JWT-based authentication with role-based access control
- **📈 Analytics**: Comprehensive analytics and reporting capabilities
- **🤖 Automation**: Intelligent automation engine with ML-powered insights
- **🔔 Alerting**: Advanced alerting and notification system

### Architecture

DriftMgr follows a microservices architecture with the following components:

- **API Server**: RESTful API with WebSocket support
- **Web Dashboard**: Modern web interface with real-time updates
- **Authentication Service**: JWT-based authentication and authorization
- **WebSocket Service**: Real-time communication and notifications
- **Analytics Engine**: Data processing and insights generation
- **Automation Engine**: Intelligent remediation and orchestration

## 🚀 Quick Start

### Prerequisites

- Go 1.21 or later
- Node.js 18 or later (for web dashboard development)
- PostgreSQL 13 or later (for production)
- Docker (optional, for containerized deployment)

### Installation

1. **Clone the repository**:
   ```bash
   git clone https://github.com/catherinevee/driftmgr.git
   cd driftmgr
   ```

2. **Install dependencies**:
   ```bash
   go mod download
   ```

3. **Build the application**:
   ```bash
   go build -o bin/driftmgr-server ./cmd/server
   ```

4. **Run the server**:
   ```bash
   ./bin/driftmgr-server --port 8080 --host 0.0.0.0
   ```

5. **Access the dashboard**:
   Open your browser and navigate to `http://localhost:8080/dashboard`

### First Steps

1. **Health Check**: Verify the server is running by visiting `http://localhost:8080/health`
2. **API Documentation**: Explore the API at `http://localhost:8080/api/v1/version`
3. **WebSocket Connection**: Test real-time features at `ws://localhost:8080/ws`

## 📦 Installation

### Binary Installation

Download the latest release from the [releases page](https://github.com/catherinevee/driftmgr/releases) and extract it to your desired location.

### Docker Installation

```bash
# Pull the latest image
docker pull driftmgr/driftmgr:latest

# Run the container
docker run -d \
  --name driftmgr \
  -p 8080:8080 \
  -e DRIFTMGR_DB_HOST=your-db-host \
  -e DRIFTMGR_DB_PASSWORD=your-db-password \
  driftmgr/driftmgr:latest
```

### Source Installation

1. **Clone and build**:
   ```bash
   git clone https://github.com/catherinevee/driftmgr.git
   cd driftmgr
   go build -o bin/driftmgr-server ./cmd/server
   ```

2. **Install systemd service** (Linux):
   ```bash
   sudo cp scripts/driftmgr.service /etc/systemd/system/
   sudo systemctl enable driftmgr
   sudo systemctl start driftmgr
   ```

## ⚙️ Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `DRIFTMGR_HOST` | Server host | `0.0.0.0` | No |
| `DRIFTMGR_PORT` | Server port | `8080` | No |
| `DRIFTMGR_DB_HOST` | Database host | `localhost` | Yes |
| `DRIFTMGR_DB_PORT` | Database port | `5432` | No |
| `DRIFTMGR_DB_NAME` | Database name | `driftmgr` | Yes |
| `DRIFTMGR_DB_USER` | Database user | `driftmgr` | Yes |
| `DRIFTMGR_DB_PASSWORD` | Database password | - | Yes |
| `DRIFTMGR_JWT_SECRET` | JWT secret key | - | Yes |
| `DRIFTMGR_LOG_LEVEL` | Log level | `info` | No |

### Configuration File

Create a `config.yaml` file in your working directory:

```yaml
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
  password: "your-password"
  ssl_mode: "require"

auth:
  jwt_secret: "your-jwt-secret-key"
  jwt_issuer: "driftmgr"
  jwt_audience: "driftmgr-api"
  access_token_expiry: "15m"
  refresh_token_expiry: "7d"

logging:
  level: "info"
  format: "json"
  output: "stdout"
```

### Command Line Options

```bash
./bin/driftmgr-server --help

Usage of driftmgr-server:
  -auth
        Enable authentication (default false)
  -config string
        Path to configuration file
  -host string
        Server host (default "0.0.0.0")
  -port string
        Server port (default "8080")
```

## 🔌 API Reference

### Base URL

All API endpoints are prefixed with `/api/v1/`

### Authentication

Most endpoints require authentication. Include the JWT token in the Authorization header:

```bash
curl -H "Authorization: Bearer <your-jwt-token>" \
     http://localhost:8080/api/v1/resources
```

### Response Format

All API responses follow a consistent format:

```json
{
  "success": true,
  "data": {
    // Response data
  },
  "error": null
}
```

### Error Format

```json
{
  "success": false,
  "data": null,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable error message",
    "details": "Additional error details"
  }
}
```

### Core Endpoints

#### Health Check
```http
GET /health
GET /api/v1/health
```

#### Version Information
```http
GET /api/v1/version
```

#### Authentication
```http
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/refresh
POST /api/v1/auth/logout
GET  /api/v1/auth/profile
PUT  /api/v1/auth/profile
```

#### Backend Management
```http
GET    /api/v1/backends/list
POST   /api/v1/backends/discover
GET    /api/v1/backends/{id}
PUT    /api/v1/backends/{id}
DELETE /api/v1/backends/{id}
POST   /api/v1/backends/{id}/test
```

#### State Management
```http
GET    /api/v1/state/list
GET    /api/v1/state/details
POST   /api/v1/state/import
DELETE /api/v1/state/resources/{id}
POST   /api/v1/state/move
POST   /api/v1/state/lock
POST   /api/v1/state/unlock
```

#### Resource Management
```http
GET  /api/v1/resources
GET  /api/v1/resources/{id}
GET  /api/v1/resources/search
PUT  /api/v1/resources/{id}/tags
GET  /api/v1/resources/{id}/cost
GET  /api/v1/resources/{id}/compliance
```

#### Drift Detection
```http
POST   /api/v1/drift/detect
GET    /api/v1/drift/results
GET    /api/v1/drift/results/{id}
DELETE /api/v1/drift/results/{id}
GET    /api/v1/drift/history
GET    /api/v1/drift/summary
```

#### WebSocket
```http
GET /ws
GET /api/v1/ws
GET /api/v1/ws/stats
```

## 🖥️ Web Dashboard

The DriftMgr web dashboard provides a modern, responsive interface for managing your cloud infrastructure.

### Features

- **📊 Real-time Dashboard**: Live updates via WebSocket
- **🔍 Resource Explorer**: Browse and search cloud resources
- **📈 Analytics**: Comprehensive charts and visualizations
- **🔧 Remediation Tools**: Manage drift detection and remediation
- **⚙️ Configuration**: Backend and state management
- **👥 User Management**: Authentication and authorization

### Accessing the Dashboard

1. **Login Page**: `http://localhost:8080/login`
2. **Main Dashboard**: `http://localhost:8080/dashboard`

### Dashboard Pages

#### Overview
- System health metrics
- Resource distribution charts
- Cost analysis
- Drift detection summary

#### Backend Management
- Terraform backend configuration
- Backend discovery and testing
- Connection management

#### State Management
- Terraform state file management
- Resource import/export
- State locking and unlocking

#### Resources
- Cloud resource inventory
- Resource search and filtering
- Tag management
- Cost and compliance information

#### Drift Detection
- Drift detection jobs
- Results analysis
- Remediation strategies
- Historical drift data

#### Remediation
- Remediation job management
- Strategy configuration
- Progress monitoring
- Approval workflows

## 🔐 Authentication

DriftMgr uses JWT-based authentication with role-based access control (RBAC).

### User Registration

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "email": "admin@example.com",
    "password": "SecurePassword123!",
    "first_name": "Admin",
    "last_name": "User"
  }'
```

### User Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "SecurePassword123!"
  }'
```

### Roles and Permissions

#### Admin Role
- Full system access
- User management
- System configuration
- All resource operations

#### User Role
- Resource viewing
- Drift detection
- Limited remediation

#### Viewer Role
- Read-only access
- Dashboard viewing
- Report generation

### API Key Authentication

For programmatic access, you can create API keys:

```bash
curl -X POST http://localhost:8080/api/v1/auth/api-keys \
  -H "Authorization: Bearer <your-jwt-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "automation-key",
    "permissions": ["read", "write"]
  }'
```

## 🔌 WebSocket API

DriftMgr provides real-time updates via WebSocket connections.

### Connection

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onopen = function(event) {
  console.log('Connected to DriftMgr WebSocket');
};

ws.onmessage = function(event) {
  const message = JSON.parse(event.data);
  console.log('Received:', message);
};
```

### Message Types

#### Connection Established
```json
{
  "type": "connection_established",
  "data": {
    "message": "Connected to DriftMgr WebSocket",
    "user_id": "user-id",
    "roles": ["user"]
  },
  "timestamp": "2025-09-24T10:00:00Z"
}
```

#### Drift Detection Updates
```json
{
  "type": "drift_detection",
  "data": {
    "job_id": "job-id",
    "status": "completed",
    "results": {...}
  },
  "timestamp": "2025-09-24T10:00:00Z"
}
```

#### System Alerts
```json
{
  "type": "system_alert",
  "data": {
    "severity": "warning",
    "message": "High drift detection rate detected",
    "details": {...}
  },
  "timestamp": "2025-09-24T10:00:00Z"
}
```

#### Heartbeat
```json
{
  "type": "heartbeat",
  "data": {
    "timestamp": 1695556800,
    "server_time": "2025-09-24T10:00:00Z"
  },
  "timestamp": "2025-09-24T10:00:00Z"
}
```

## 🚀 Deployment

### Production Deployment

#### Prerequisites

- PostgreSQL database
- SSL certificates (for HTTPS)
- Reverse proxy (nginx/Apache)
- Monitoring solution

#### Database Setup

1. **Create database**:
   ```sql
   CREATE DATABASE driftmgr;
   CREATE USER driftmgr WITH PASSWORD 'secure-password';
   GRANT ALL PRIVILEGES ON DATABASE driftmgr TO driftmgr;
   ```

2. **Run migrations**:
   ```bash
   ./bin/driftmgr-server --migrate
   ```

#### Environment Configuration

```bash
export DRIFTMGR_DB_HOST=your-db-host
export DRIFTMGR_DB_PASSWORD=secure-password
export DRIFTMGR_JWT_SECRET=your-very-secure-jwt-secret
export DRIFTMGR_LOG_LEVEL=info
```

#### Nginx Configuration

```nginx
server {
    listen 443 ssl;
    server_name your-domain.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
    
    location /ws {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
    }
}
```

### Docker Deployment

#### Docker Compose

```yaml
version: '3.8'

services:
  driftmgr:
    image: driftmgr/driftmgr:latest
    ports:
      - "8080:8080"
    environment:
      - DRIFTMGR_DB_HOST=postgres
      - DRIFTMGR_DB_PASSWORD=secure-password
      - DRIFTMGR_JWT_SECRET=your-jwt-secret
    depends_on:
      - postgres
    restart: unless-stopped

  postgres:
    image: postgres:15
    environment:
      - POSTGRES_DB=driftmgr
      - POSTGRES_USER=driftmgr
      - POSTGRES_PASSWORD=secure-password
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: unless-stopped

volumes:
  postgres_data:
```

#### Kubernetes Deployment

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
---
apiVersion: v1
kind: Service
metadata:
  name: driftmgr-service
spec:
  selector:
    app: driftmgr
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

## 🛠️ Development

### Development Setup

1. **Clone repository**:
   ```bash
   git clone https://github.com/catherinevee/driftmgr.git
   cd driftmgr
   ```

2. **Install dependencies**:
   ```bash
   go mod download
   npm install  # For web dashboard development
   ```

3. **Run development server**:
   ```bash
   go run ./cmd/server --port 8080 --host 0.0.0.0
   ```

4. **Run tests**:
   ```bash
   go test ./...
   ```

### Project Structure

```
driftmgr/
├── cmd/                    # Application entry points
│   └── server/            # Main server application
├── internal/              # Private application code
│   ├── api/              # API handlers and routes
│   ├── auth/             # Authentication service
│   ├── websocket/        # WebSocket service
│   ├── models/           # Data models
│   └── ...
├── web/                  # Web dashboard
│   ├── dashboard/        # Main dashboard
│   └── login/           # Login page
├── docs/                # Documentation
├── scripts/             # Build and deployment scripts
├── tests/               # Test files
└── ...
```

### Code Style

- Follow Go standard formatting (`gofmt`)
- Use meaningful variable and function names
- Add comments for exported functions and types
- Write tests for new functionality
- Follow the existing project structure

### Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

## 🧪 Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific test package
go test ./internal/websocket

# Run tests with verbose output
go test -v ./...

# Run benchmarks
go test -bench=. ./...
```

### Test Categories

- **Unit Tests**: Individual component testing
- **Integration Tests**: Component interaction testing
- **End-to-End Tests**: Full application flow testing
- **Performance Tests**: Load and benchmark testing

### Test Automation

Use the provided test runner script:

```bash
# Run all tests
./scripts/run-tests.sh

# Run with specific options
./scripts/run-tests.sh --verbose --benchmarks --e2e
```

## 🔧 Troubleshooting

### Common Issues

#### Server Won't Start

**Issue**: Server fails to start with port binding error
**Solution**: Check if port 8080 is already in use and change the port:
```bash
./bin/driftmgr-server --port 8081
```

#### Database Connection Issues

**Issue**: Cannot connect to database
**Solution**: Verify database configuration and connectivity:
```bash
# Test database connection
psql -h localhost -U driftmgr -d driftmgr -c "SELECT 1;"
```

#### WebSocket Connection Issues

**Issue**: WebSocket connections fail
**Solution**: Check firewall settings and proxy configuration:
```bash
# Test WebSocket connection
curl -i -N -H "Connection: Upgrade" \
     -H "Upgrade: websocket" \
     -H "Sec-WebSocket-Key: test" \
     -H "Sec-WebSocket-Version: 13" \
     http://localhost:8080/ws
```

#### Authentication Issues

**Issue**: JWT token validation fails
**Solution**: Verify JWT secret configuration and token format:
```bash
# Check JWT secret is set
echo $DRIFTMGR_JWT_SECRET
```

### Logging

Enable debug logging for troubleshooting:

```bash
export DRIFTMGR_LOG_LEVEL=debug
./bin/driftmgr-server
```

### Health Checks

Monitor application health:

```bash
# Basic health check
curl http://localhost:8080/health

# Detailed health information
curl http://localhost:8080/api/v1/health
```

### Performance Monitoring

Monitor application performance:

```bash
# WebSocket connection stats
curl http://localhost:8080/api/v1/ws/stats

# System metrics (if enabled)
curl http://localhost:8080/api/v1/metrics
```

## 📞 Support

### Getting Help

- **Documentation**: Check this documentation first
- **Issues**: Report bugs and feature requests on GitHub
- **Discussions**: Join community discussions
- **Email**: Contact the maintainers

### Reporting Issues

When reporting issues, please include:

1. DriftMgr version
2. Operating system and version
3. Steps to reproduce the issue
4. Expected vs actual behavior
5. Relevant log output
6. Configuration details (without sensitive information)

### Feature Requests

For feature requests, please include:

1. Description of the feature
2. Use case and benefits
3. Proposed implementation (if applicable)
4. Any relevant examples or mockups

---

## 📄 License

DriftMgr is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Terraform community for inspiration
- Go community for excellent tooling
- All contributors and users

---

**DriftMgr** - Keeping your cloud infrastructure in sync! 🚀
