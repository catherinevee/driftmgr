# DriftMgr

A comprehensive infrastructure drift detection and auto-remediation platform for multi-cloud environments. DriftMgr provides continuous monitoring, intelligent remediation, and cost optimization across AWS, Azure, GCP, and DigitalOcean.

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Usage](#usage)
- [Web Interface](#web-interface)
- [API Reference](#api-reference)
- [CI/CD Integration](#cicd-integration)
- [Monitoring](#monitoring)
- [Contributing](#contributing)
- [License](#license)

## Features

### Core Capabilities

- **Multi-Cloud Support**: Comprehensive coverage for AWS, Azure, GCP, and DigitalOcean
- **Drift Detection**: Real-time identification of configuration drift between desired and actual states
- **Auto-Remediation**: Intelligent, rule-based automatic drift correction with safety controls
- **Cost Optimization**: Multi-cloud cost analysis and optimization recommendations
- **Compliance Monitoring**: Framework-based compliance tracking and violation detection
- **Web Interface**: Modern dashboard with real-time visualizations and monitoring
- **Enhanced TUI**: Full-featured terminal interface with drift detection, remediation, and real-time updates

### Advanced Features

- **Approval Chains**: Multi-level approval workflows for critical resource changes
- **Auto Backups**: Automatic state backup before any remediation action
- **Drift Simulation**: Test remediation plans in isolated environments
- **Change Attribution**: Audit log integration to identify who made changes
- **Resource Dependency Mapping**: Visual dependency graphs with impact analysis
- **SLO Monitoring**: Service Level Objective tracking with error budget management

### Service Coverage

- **AWS**: 75 services including EC2, S3, RDS, Lambda, EKS, and more
- **Azure**: 66 services including VMs, Storage, AKS, SQL Database, and more
- **GCP**: 47 services including Compute, Storage, GKE, BigQuery, and more
- **DigitalOcean**: 10 services including Droplets, Kubernetes, Databases, and more
- **Total Coverage**: 198 cloud services across all providers

## Architecture

DriftMgr uses a modular architecture with the following components:

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Web UI        │────▶│   REST API      │────▶│  Core Engine    │
│  (React/TS)     │     │   (Gin)         │     │   (Go)          │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                               │                         │
                               ▼                         ▼
                        ┌─────────────────┐     ┌─────────────────┐
                        │   WebSocket     │     │  Cloud APIs     │
                        │   Real-time     │     │  AWS/Azure/GCP  │
                        └─────────────────┘     └─────────────────┘
                               │                         │
                               ▼                         ▼
                        ┌─────────────────┐     ┌─────────────────┐
                        │  Monitoring     │     │   Database      │
                        │  Prometheus     │     │  PostgreSQL     │
                        │  Grafana        │     │   Redis         │
                        └─────────────────┘     └─────────────────┘
```

## Installation

### Prerequisites

- Go 1.21 or higher
- Node.js 18 or higher (for web interface)
- Docker and Docker Compose (for monitoring stack)
- Cloud provider credentials configured

### Build from Source

```bash
# Clone the repository
git clone https://github.com/catherinevee/driftmgr.git
cd driftmgr

# Build the binary
go build -o driftmgr ./cmd/driftmgr

# Make it executable (Linux/Mac)
chmod +x driftmgr

# Move to PATH (optional)
sudo mv driftmgr /usr/local/bin/
```

### Install with Script

```bash
# Linux/Mac
curl -sSL https://raw.githubusercontent.com/catherinevee/driftmgr/main/install.sh | bash

# Windows
powershell -ExecutionPolicy Bypass -File install.ps1
```

## Quick Start

### 1. Basic Drift Detection

```bash
# Detect drift across all providers
driftmgr drift detect --provider all

# Detect drift for specific provider
driftmgr drift detect --provider aws --region us-east-1

# Generate drift report
driftmgr drift report --format json --output drift-report.json
```

### 2. Enable Auto-Remediation

```bash
# Enable auto-remediation (dry-run by default)
driftmgr auto-remediation enable --dry-run

# View auto-remediation status
driftmgr auto-remediation status

# Test remediation without applying
driftmgr auto-remediation test --resource ec2-instance-123
```

### 3. Start Web Interface

```bash
# Start all services
./start-web.sh  # Linux/Mac
.\start-web.ps1  # Windows

# Access at http://localhost:5173
```

## Configuration

### Main Configuration (configs/config.yaml)

```yaml
providers:
  aws:
    enabled: true
    regions:
      - us-east-1
      - us-west-2
    credentials:
      access_key_id: ${AWS_ACCESS_KEY_ID}
      secret_access_key: ${AWS_SECRET_ACCESS_KEY}
  
  azure:
    enabled: true
    subscriptions:
      - ${AZURE_SUBSCRIPTION_ID}
    credentials:
      client_id: ${AZURE_CLIENT_ID}
      client_secret: ${AZURE_CLIENT_SECRET}
      tenant_id: ${AZURE_TENANT_ID}

  gcp:
    enabled: true
    projects:
      - ${GCP_PROJECT_ID}
    credentials:
      key_file: ${GCP_KEY_FILE}

  digitalocean:
    enabled: true
    token: ${DIGITALOCEAN_TOKEN}

drift_detection:
  scan_interval: 15m
  parallel_workers: 10
  state_storage: local  # or s3, azure_blob, gcs

monitoring:
  metrics_enabled: true
  tracing_enabled: true
  logging_level: info
```

### Auto-Remediation Configuration (configs/auto-remediation.yaml)

```yaml
enabled: false  # Set to true to enable
dry_run: true   # Set to false for actual remediation
scan_interval: 15m
max_concurrent: 5

rules:
  - name: auto-fix-tags
    description: Automatically fix missing or incorrect tags
    enabled: true
    drift_types: [modified]
    max_risk_level: low
    action:
      type: auto_fix
      strategy: terraform
    requires_approval: false

  - name: recreate-missing-resources
    description: Recreate missing resources with approval
    enabled: true
    drift_types: [missing]
    max_risk_level: medium
    action:
      type: auto_fix
      strategy: terraform
    requires_approval: true
    approval_timeout: 30m

safety:
  max_remediations_per_hour: 20
  max_cost_impact: 1000.0
  require_backup: true
  enable_rollback: true
  critical_resource_protection: true
```

## Usage

### Command Line Interface

```bash
# Launch interactive TUI (basic mode)
driftmgr

# Launch enhanced TUI with full features (requires web server running)
driftmgr --enhanced
# or
driftmgr -e

# Drift Detection
driftmgr drift detect --provider aws --severity high
driftmgr drift report --format html --output report.html

# Resource Discovery
driftmgr discover --provider all --export json
driftmgr discover --filter "type=ec2_instance"

# Cost Analysis
driftmgr cost analyze --provider all
driftmgr cost optimize --min-savings 100

# Compliance Scanning
driftmgr compliance scan --framework pci-dss
driftmgr compliance violations --severity critical

# Backup and Restore
driftmgr backup create --resource vpc-123
driftmgr backup restore --backup-id backup_abc123

# Change Attribution
driftmgr audit who-changed --resource sg-123456
driftmgr audit history --days 30
```

### Environment Variables

```bash
# Cloud Credentials
export AWS_ACCESS_KEY_ID=your-key
export AWS_SECRET_ACCESS_KEY=your-secret
export AZURE_CLIENT_ID=your-client-id
export AZURE_CLIENT_SECRET=your-secret
export AZURE_TENANT_ID=your-tenant
export GCP_PROJECT_ID=your-project
export DIGITALOCEAN_TOKEN=your-token

# DriftMgr Settings
export DRIFTMGR_CONFIG=/path/to/config.yaml
export DRIFTMGR_LOG_LEVEL=info
export DRIFTMGR_PORT=8080
```

## Terminal User Interface (TUI)

DriftMgr offers two TUI modes:

### Basic TUI
The default lightweight interface for resource discovery and basic operations:
```bash
driftmgr
```

Features:
- Resource discovery across cloud providers
- Account management
- Export functionality
- Configuration viewing

### Enhanced TUI
Full-featured terminal interface with complete drift management capabilities:
```bash
# Start the web server first (required for API access)
go run ./internal/web/server.go &

# Launch enhanced TUI
driftmgr --enhanced
```

Enhanced TUI Features:
- **Drift Detection**: View and analyze detected drifts
- **Auto-Remediation**: Manage remediation workflows with approval chains
- **Compliance Monitoring**: Track compliance violations and scores
- **SLO Monitoring**: View service level objectives and error budgets
- **Resource Dependencies**: Text-based dependency visualization
- **Cost Optimization**: View cost analysis and recommendations
- **Real-time Updates**: WebSocket integration for live data
- **Multi-Cloud Support**: Parallel scanning across providers

## Web Interface

The web interface provides comprehensive visualization and management capabilities:

### Starting the Web Interface

```bash
# Start monitoring stack (optional but recommended)
docker-compose -f docker-compose.monitoring.yml up -d

# Start backend server
go run ./internal/web/server.go

# Start frontend (in another terminal)
cd web
npm install
npm run dev

# Access at http://localhost:5173
```

### Features

- **Dashboard**: Real-time drift overview with key metrics
- **Drift Detection**: Visual drift analysis with severity mapping
- **Remediation Center**: Manage and track remediation actions
- **Resource Map**: Interactive dependency visualization
- **Cost Analysis**: Cost impact and optimization opportunities
- **Compliance**: Framework-based compliance tracking
- **SLO Monitoring**: Service level objective tracking
- **Settings**: Configure rules, thresholds, and integrations

### API Endpoints

```
GET  /api/v1/dashboard          - Dashboard data
GET  /api/v1/drift/detect       - Trigger drift detection
GET  /api/v1/drift/report/:id   - Get drift report
POST /api/v1/remediation/execute - Execute remediation
GET  /api/v1/resources          - List resources
GET  /api/v1/cost/analysis     - Cost analysis
GET  /api/v1/compliance/status  - Compliance status
GET  /api/v1/slo/metrics       - SLO metrics
WS   /ws                        - WebSocket for real-time updates
```

## CI/CD Integration

### GitHub Actions

```yaml
# .github/workflows/drift-check.yml
name: Drift Check
on:
  schedule:
    - cron: '0 */6 * * *'
  pull_request:
    branches: [main]

jobs:
  drift-detection:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run DriftMgr
        run: |
          driftmgr drift detect --provider all --format json
```

### GitLab CI

```yaml
# .gitlab-ci.yml
drift:check:
  stage: test
  script:
    - driftmgr drift detect --provider all
  only:
    - merge_requests
```

### Jenkins

```groovy
// Jenkinsfile
pipeline {
    agent any
    stages {
        stage('Drift Detection') {
            steps {
                sh 'driftmgr drift detect --provider all'
            }
        }
    }
}
```

## Monitoring

DriftMgr integrates with the LGTM stack (Loki, Grafana, Tempo, Mimir/Prometheus) for comprehensive observability:

### Metrics

- `driftmgr_drift_detected_total` - Total drifts detected
- `driftmgr_remediation_success_total` - Successful remediations
- `driftmgr_scan_duration_seconds` - Scan performance
- `driftmgr_api_latency_seconds` - API latency
- `driftmgr_false_positive_rate` - Detection accuracy

### Accessing Monitoring

```bash
# Start monitoring stack
docker-compose -f docker-compose.monitoring.yml up -d

# Access dashboards
# Grafana: http://localhost:3000 (admin/driftmgr)
# Prometheus: http://localhost:9090
# Jaeger: http://localhost:16686
```

### SLO Targets

- **Availability**: 99.9% uptime
- **Detection Latency**: P95 < 10 minutes
- **Remediation Success**: > 95%
- **False Positive Rate**: < 5%

## API Reference

Detailed API documentation is available at:
- Swagger UI: http://localhost:8080/swagger
- OpenAPI Spec: http://localhost:8080/openapi.json

## Troubleshooting

### Common Issues

**Drift detection not working:**
```bash
# Check cloud credentials
driftmgr validate credentials --provider aws

# Increase log verbosity
export DRIFTMGR_LOG_LEVEL=debug
driftmgr drift detect --provider aws
```

**Web interface connection issues:**
```bash
# Check services are running
curl http://localhost:8080/health
curl http://localhost:5173

# Check logs
docker-compose logs -f
```

**High memory usage:**
```bash
# Reduce parallel workers
driftmgr drift detect --workers 5

# Enable memory profiling
export DRIFTMGR_PPROF=true
```

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Security

For security issues, please email security@driftmgr.io instead of using the issue tracker.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- Documentation: [https://docs.driftmgr.io](https://docs.driftmgr.io)
- Issues: [GitHub Issues](https://github.com/catherinevee/driftmgr/issues)
- Discussions: [GitHub Discussions](https://github.com/catherinevee/driftmgr/discussions)
- Email: support@driftmgr.io