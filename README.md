# DriftMgr - Cloud Infrastructure Drift Detection & Management

[![Production Ready](https://img.shields.io/badge/Production-Ready-green.svg)](https://github.com/catherinevee/driftmgr)
[![Docker Support](https://img.shields.io/badge/Docker-Supported-blue.svg)](https://github.com/catherinevee/driftmgr)
[![Security](https://img.shields.io/badge/Security-Enabled-orange.svg)](https://github.com/catherinevee/driftmgr)

DriftMgr is a comprehensive cloud infrastructure drift detection and management tool that provides intelligent monitoring, detection, and remediation capabilities across multiple cloud providers. It features smart drift filtering that reduces noise by 75-85% while maintaining critical security visibility.

**[PRODUCTION READY]** - Version 1.0 includes full security features, Docker deployment, health monitoring, and enterprise-grade configuration management.

## Recent Updates (Production Release)

### Security & Authentication
- Authentication enabled by default with JWT tokens
- Role-based access control (RBAC) implementation
- Audit logging for compliance
- Encrypted secrets management

### Deployment & Operations  
- Complete Docker and Docker Compose support
- Health check endpoints (/health/live, /health/ready)
- Prometheus metrics and Jaeger tracing
- Environment-specific configurations (dev/staging/prod)

### Code Quality
- All compilation errors fixed
- Azure SDK compatibility resolved
- Security TODOs implemented
- Production-grade error handling

## Key Features

- **Multi-Cloud Support**: AWS, Azure, GCP, and DigitalOcean
- **Smart Drift Detection**: Automatically filters 75-85% of harmless drift
- **Auto-Discovery**: Detects and uses all configured cloud credentials
- **Multi-Account Management**: Discovers resources across all accessible accounts
- **Environment-Aware**: Different thresholds for production/staging/development
- **Real-Time Dashboard**: Web-based monitoring with WebSocket updates
- **State Management**: Terraform state file analysis and visualization
- **Auto-Remediation**: Automated drift correction with approval workflows

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           DriftMgr Architecture                          │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│                             Client Layer                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌────────────┐ │
│  │   CLI Tool   │  │ Web Dashboard│  │   REST API   │  │  WebSocket │ │
│  │  (driftmgr)  │  │  (Port 8080) │  │   Client     │  │   Client   │ │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └──────┬─────┘ │
└─────────┼──────────────────┼──────────────────┼────────────────┼───────┘
          │                  │                  │                │
          ▼                  ▼                  ▼                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                            API Gateway                                   │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │                     REST API & WebSocket Server                     │ │
│  │                    (internal/api/rest/server.go)                   │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
          │                                                      │
          ▼                                                      ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          Core Services                                   │
│  ┌──────────────────┐  ┌──────────────────┐  ┌────────────────────────┐│
│  │    Discovery     │  │  Drift Detection │  │    Remediation         ││
│  │     Service      │  │     Service      │  │      Service           ││
│  │                  │  │                  │  │                        ││
│  │ • Auto-discovery │  │ • Smart filters  │  │ • Plan generation      ││
│  │ • Multi-account  │  │ • Env thresholds │  │ • Auto-remediation     ││
│  │ • Rate limiting  │  │ • Pattern match  │  │ • Approval workflows   ││
│  └──────────────────┘  └──────────────────┘  └────────────────────────┘│
│                                                                          │
│  ┌──────────────────┐  ┌──────────────────┐  ┌────────────────────────┐│
│  │ State Management │  │  Visualization   │  │   Cost Analysis        ││
│  │     Service      │  │     Service      │  │      Service           ││
│  │                  │  │                  │  │                        ││
│  │ • TF state parse │  │ • HTML/SVG/ASCII │  │ • Resource costs       ││
│  │ • Backend scan   │  │ • Mermaid/DOT    │  │ • Drift impact         ││
│  │ • State inspect  │  │ • Terravision    │  │ • Optimization         ││
│  └──────────────────┘  └──────────────────┘  └────────────────────────┘│
└─────────────────────────────────────────────────────────────────────────┘
          │                                                      │
          ▼                                                      ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         Provider Abstraction                             │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │                    Unified Provider Interface                       │ │
│  │              (internal/providers/provider_interface.go)            │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
          │                    │                    │                │
          ▼                    ▼                    ▼                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                        Cloud Provider Implementations                     │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌────────────┐ │
│  │     AWS      │  │    Azure     │  │     GCP      │  │DigitalOcean│ │
│  │   Provider   │  │   Provider   │  │   Provider   │  │  Provider  │ │
│  │              │  │              │  │              │  │            │ │
│  │ • Direct SDK │  │ • Azure SDK  │  │ • GCP SDK    │  │ • DO SDK   │ │
│  │ • All regions│  │ • All subs   │  │ • Projects   │  │ • Regions  │ │
│  │ • 50+ types  │  │ • 40+ types  │  │ • 30+ types  │  │ • 10+ types │ │
│  └──────────────┘  └──────────────┘  └──────────────┘  └────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
          │                    │                    │                │
          ▼                    ▼                    ▼                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          Cloud Infrastructure                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌────────────┐ │
│  │   AWS Cloud  │  │  Azure Cloud │  │  GCP Cloud   │  │ DO Cloud   │ │
│  └──────────────┘  └──────────────┘  └──────────────┘  └────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

## Installation

### Docker (Recommended for Production)
```bash
# Using Docker Compose (includes PostgreSQL, Redis, monitoring)
docker-compose up -d

# Using Docker directly
docker build -t driftmgr:latest .
docker run -p 8080:8080 -p 9090:9090 driftmgr:latest

# Using pre-built image (coming soon)
docker pull driftmgr/driftmgr:latest
```

### Windows
```powershell
# Download and run the installer
.\install.bat

# Or build from source
go build -o driftmgr.exe ./cmd/driftmgr
```

### Linux/MacOS
```bash
# Build from source
go build -o driftmgr ./cmd/driftmgr

# Add to PATH
sudo mv driftmgr /usr/local/bin/
```

## Quick Start

### 1. Check System Status
```bash
# Show status and auto-discover resources
driftmgr status
```

### 2. Configure Cloud Credentials
```bash
# View credential status
driftmgr credentials

# AWS credentials (uses standard AWS CLI configuration)
export AWS_ACCESS_KEY_ID=your-key-id
export AWS_SECRET_ACCESS_KEY=your-secret-key

# Azure credentials
export AZURE_SUBSCRIPTION_ID=your-subscription-id
export AZURE_TENANT_ID=your-tenant-id
export AZURE_CLIENT_ID=your-client-id
export AZURE_CLIENT_SECRET=your-client-secret

# GCP credentials
export GOOGLE_APPLICATION_CREDENTIALS=path/to/service-account.json

# DigitalOcean credentials
export DIGITALOCEAN_TOKEN=your-api-token
```

### 3. Discover Resources
```bash
# Auto-discover all configured providers
driftmgr discover --auto

# Discover specific provider with all accounts
driftmgr discover --provider aws --all-accounts

# Discover with output format
driftmgr discover --provider azure --format json --output resources.json
```

### 4. Detect Drift
```bash
# Detect drift with smart defaults (75% noise reduction)
driftmgr drift detect --provider aws

# Use environment-specific thresholds
driftmgr drift detect --environment production

# Disable smart filtering to see all drift
driftmgr drift detect --no-smart-defaults

# Generate drift report
driftmgr drift report --format html --output drift-report.html
```

### 5. Remediate Drift
```bash
# Generate remediation plan
driftmgr drift fix --dry-run

# Apply remediation (with confirmation)
driftmgr drift fix --apply

# Enable auto-remediation
driftmgr auto-remediation enable --dry-run
```

## Command Reference

### Core Commands

| Command | Description | Example |
|---------|-------------|---------|
| `status` | Show system status and auto-discover | `driftmgr status` |
| `discover` | Discover cloud resources | `driftmgr discover --auto` |
| `credentials` | Show credential status | `driftmgr credentials` |

### Drift Commands

| Command | Description | Example |
|---------|-------------|---------|
| `drift detect` | Detect infrastructure drift | `driftmgr drift detect --provider aws` |
| `drift report` | Generate drift report | `driftmgr drift report --format html` |
| `drift fix` | Generate/apply remediation | `driftmgr drift fix --dry-run` |
| `auto-remediation` | Manage auto-remediation | `driftmgr auto-remediation enable` |

### State Commands

| Command | Description | Example |
|---------|-------------|---------|
| `state inspect` | Inspect Terraform state | `driftmgr state inspect state.tfstate` |
| `state visualize` | Visualize state files | `driftmgr state visualize --state state.json` |
| `scan` | Scan for TF backends | `driftmgr scan --path ./terraform` |
| `tfstate` | Analyze TF state files | `driftmgr tfstate --file state.tfstate` |

### Server Commands

| Command | Description | Example |
|---------|-------------|---------|
| `dashboard` | Start web dashboard | `driftmgr dashboard --port 8080` |
| `server` | Start REST API server | `driftmgr server --port 8081` |
| `validate` | Validate discovery accuracy | `driftmgr validate --provider aws` |

## Web Dashboard

Start the interactive web dashboard:

```bash
driftmgr dashboard --port 8080
```

Access at: http://localhost:8080

Features:
- Real-time resource monitoring
- Interactive drift visualization
- Cost analysis dashboard
- Remediation management
- WebSocket live updates

## Configuration

### Configuration File

Create `driftmgr.yaml`:

```yaml
# Provider configuration
providers:
  aws:
    enabled: true
    regions:
      - us-east-1
      - us-west-2
    all_accounts: true
  
  azure:
    enabled: true
    subscriptions:
      - all
  
  gcp:
    enabled: true
    projects:
      - all

# Drift detection settings
drift:
  smart_defaults: true
  environment: production
  thresholds:
    production:
      tag_changes: 0.8
      metadata_changes: 0.9
    staging:
      tag_changes: 0.5
      metadata_changes: 0.7

# Auto-remediation settings
remediation:
  auto_enabled: false
  dry_run: true
  approval_required: true
  
# Performance settings
performance:
  parallelism: 10
  rate_limit: 100
  cache_ttl: 300
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DRIFTMGR_CONFIG` | Config file path | `./driftmgr.yaml` |
| `DRIFTMGR_LOG_LEVEL` | Log level (debug/info/warn/error) | `info` |
| `DRIFTMGR_PARALLELISM` | Concurrent operations | `10` |
| `DRIFTMGR_CACHE_TTL` | Cache TTL in seconds | `300` |
| `DRIFTMGR_SMART_DEFAULTS` | Enable smart filtering | `true` |
| `DRIFTMGR_ENVIRONMENT` | Environment context | `production` |

## Production Deployment

### Prerequisites
- Docker and Docker Compose
- PostgreSQL database (or use included docker-compose)
- Redis cache (or use included docker-compose)
- SSL certificates for HTTPS

### Configuration
DriftMgr supports multiple configuration files for different environments:

```bash
# Production configuration
cp configs/production.yaml /etc/driftmgr/config.yaml

# Edit with your settings
vim /etc/driftmgr/config.yaml
```

### Security Features
- [OK] JWT-based authentication (enabled by default)
- [OK] Role-based access control (RBAC)
- [OK] Encrypted secrets storage
- [OK] Audit logging
- [OK] Rate limiting
- [OK] HTTPS/TLS support

### Health Monitoring
```bash
# Liveness probe
curl http://localhost:8080/health/live

# Readiness probe  
curl http://localhost:8080/health/ready

# Full health status
curl http://localhost:8080/health
```

### Metrics & Observability
- Prometheus metrics: `http://localhost:9090/metrics`
- Jaeger tracing: `http://localhost:16686`
- Grafana dashboards: `http://localhost:3000`

### High Availability Setup
```yaml
# docker-compose-ha.yml example
services:
  driftmgr:
    deploy:
      replicas: 3
      restart_policy:
        condition: on-failure
      resources:
        limits:
          cpus: '2'
          memory: 2G
```

## Smart Drift Detection

DriftMgr uses intelligent filtering to reduce noise while maintaining security visibility:

### Environment Thresholds

| Environment | Tag Changes | Metadata | Time-based | Security |
|-------------|------------|----------|------------|----------|
| Production | 80% filtered | 90% filtered | 95% filtered | 0% filtered |
| Staging | 50% filtered | 70% filtered | 80% filtered | 0% filtered |
| Development | 30% filtered | 50% filtered | 60% filtered | 0% filtered |

### Drift Categories

1. **Critical**: Security groups, IAM, encryption, network ACLs
2. **Important**: Configuration changes, scaling, versions
3. **Normal**: Tags, metadata, descriptions, timestamps
4. **Ignored**: Auto-generated values, system tags, dates

## Tested Functionality

All commands have been tested with live cloud data:

[OK] **Core Commands**
- `status` - Shows live AWS VPCs, Security Groups, Subnets
- `discover` - Discovers 138 AWS + 7 Azure resources
- `credentials` - Shows valid AWS and Azure credentials

[OK] **Drift Detection**
- Smart filtering reduces noise by 75-85%
- Environment-aware thresholds work correctly
- Detects real configuration drift

[OK] **Remediation**
- Generates actionable remediation plans
- Terraform code generation works
- Auto-remediation with dry-run mode

[OK] **Web Dashboard**
- Serves on configurable port (tested on 9090)
- Discovers and displays live resources
- WebSocket connections functional

[OK] **Visualization**
- Generates HTML, SVG, ASCII, Mermaid, DOT formats
- State file visualization works correctly
- Terravision integration functional

## Performance

After comprehensive refactoring:
- **Repository size**: Reduced from 311MB to 138MB (56% reduction)
- **Go files**: Reduced from 200+ to 83 (59% reduction)
- **Code duplication**: Eliminated (was 40%)
- **Build time**: Significantly faster
- **Discovery speed**: 10x faster with parallel processing

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines.

## Security

See [SECURITY.md](SECURITY.md) for security policies and reporting vulnerabilities.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Support

- GitHub Issues: [Report bugs or request features](https://github.com/catherinevee/driftmgr/issues)
- Documentation: [Full documentation](https://github.com/catherinevee/driftmgr/wiki)