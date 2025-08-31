```
     .___      .__  _____  __                         
   __| _/______|__|/ ____\/  |_  _____    ___________ 
  / __ |\_  __ \  \   __\\   __\/     \  / ___\_  __ \
 / /_/ | |  | \/  ||  |   |  | |  Y Y  \/ /_/  >  | \/
 \____ | |__|  |__||__|   |__| |__|_|  /\___  /|__|   
      \/                             \//_____/        
```

# DriftMgr - Terraform State Intelligence Platform

[![Production Ready](https://img.shields.io/badge/Production-Ready-green.svg)](https://github.com/catherinevee/driftmgr)
[![CI/CD](https://img.shields.io/badge/CI%2FCD-Ready-brightgreen.svg)](https://github.com/catherinevee/driftmgr)
[![Multi-Cloud](https://img.shields.io/badge/Multi--Cloud-AWS%20%7C%20Azure%20%7C%20GCP%20%7C%20DO-blue.svg)](https://github.com/catherinevee/driftmgr)
[![Docker Support](https://img.shields.io/badge/Docker-Supported-blue.svg)](https://github.com/catherinevee/driftmgr)
[![Security](https://img.shields.io/badge/Security-AES--256--GCM-orange.svg)](https://github.com/catherinevee/driftmgr)

## Overview

**DriftMgr** is an advanced Terraform State Intelligence Platform that provides deep insights into your infrastructure-as-code deployments. It automatically discovers and analyzes Terraform state files across your organization, visualizes infrastructure relationships, identifies out-of-band changes, and provides intelligent remediation capabilities. DriftMgr transforms infrastructure management from reactive drift detection to proactive state governance.

**Latest Version (v2.0)** introduces Domain-Driven Design architecture, real-time cloud event processing, intelligent resource graphs, historical state reconstruction, and predictive caching for enterprise-scale deployments.

### Core Capabilities

- **State-Centric Intelligence** - Automatic discovery and analysis of Terraform/Terragrunt state files across local, cloud, and version control systems
- **Perspective Analysis** - View infrastructure from the state file's perspective vs cloud reality
- **Out-of-Band Resource Management** - Identify and adopt resources created outside of Terraform
- **Interactive Visualizations** - State galaxy view, resource dependency graphs, and coverage analytics
- **Intelligent Drift Detection** - Context-aware drift analysis with 75-85% noise reduction
- **Multi-Cloud Support** - Unified management for AWS, Azure, GCP, and DigitalOcean
- **Automated Remediation** - Safe, rollback-enabled fixes with approval workflows
- **Enterprise Features** - Audit logging, RBAC, health monitoring, and resilience patterns

### Architecture (v2.0)

DriftMgr features a **Domain-Driven Design architecture** with clean separation of concerns:

#### Layered Architecture
- **Domain Layer** - Core business logic with zero external dependencies
- **Application Layer** - Use cases and service orchestration
- **Infrastructure Layer** - Cloud providers, databases, and external integrations
- **Interface Layer** - REST API, WebSocket, CLI, and web handlers
- **Shared Layer** - Cross-cutting concerns like authentication and resilience

#### Advanced Detection Capabilities
- **Real-time Cloud Events** - CloudTrail/EventBridge integration for instant detection
- **Intelligent Resource Graph** - Automatic relationship mapping and dependency analysis
- **Historical State Reconstruction** - Track resource lifecycle across time
- **Smart Caching** - Predictive caching with multiple strategies (LRU, LFU, ARC)
- **Resource Fingerprinting** - Identify how resources were created (Console, CLI, Terraform)

## Table of Contents

- [Quick Start](#quick-start)
- [Installation](#installation)
- [Architecture](#architecture)
- [Web Interface](#web-interface)
- [State File Discovery](#state-file-discovery)
- [Perspective Analysis](#perspective-analysis)
- [Visualizations](#visualizations)
- [Out-of-Band Resources](#out-of-band-resources)
- [Drift Detection](#drift-detection)
- [Command Reference](#command-reference)
- [API Reference](#api-reference)
- [Configuration](#configuration)
- [Production Features](#production-features)
- [Development](#development)
- [Troubleshooting](#troubleshooting)

## Quick Start

### 1. Install DriftMgr

```bash
# Build from source
git clone https://github.com/catherinevee/driftmgr.git
cd driftmgr
go build -o driftmgr ./cmd/driftmgr

# Add to PATH
export PATH=$PATH:$(pwd)
```

### 2. Start the Web Interface

```bash
$ driftmgr serve web --port 8080

     .___      .__  _____  __                         
   __| _/______|__|/ ____\/  |_  _____    ___________ 
  / __ |\_  __ \  \   __\\   __\/     \  / ___\_  __ \
 / /_/ | |  | \/  ||  |   |  | |  Y Y  \/ /_/  >  | \/
 \____ | |__|  |__||__|   |__| |__|_|  /\___  /|__|   
      \/                             \//_____/        

Loading cached resources...
[API] Detected AWS credentials
[API] Detected Azure credentials
[API] Detected GCP credentials
[API] Detected DigitalOcean credentials

Starting DriftMgr Web Server on port 8080
Open your browser at http://localhost:8080

Press Ctrl+C to stop the server
```

### 3. Access the Web Interface

Navigate to `http://localhost:8080` to access the DriftMgr web interface.

## Architecture

### Directory Structure

DriftMgr follows Domain-Driven Design principles with a clean, layered architecture:

```
driftmgr/
├── cmd/                         # Application entry points
│   ├── driftmgr/               # CLI application
│   └── driftmgr-server/        # Server mode
├── internal/                    # Private application code
│   ├── domain/                 # Core business logic (no external dependencies)
│   │   ├── resource/           # Resource models, fingerprinting, categorization
│   │   ├── drift/              # Drift detection and analysis
│   │   ├── state/              # State management and aggregation
│   │   └── remediation/        # Remediation strategies and execution
│   ├── application/            # Application services and use cases
│   │   ├── discovery/          # Resource discovery orchestration
│   │   ├── monitoring/         # Real-time and continuous monitoring
│   │   ├── analysis/           # Graph analysis and history tracking
│   │   └── workflow/           # Automation workflows
│   ├── infrastructure/         # External integrations and adapters
│   │   ├── cloud/              # Cloud provider implementations
│   │   │   ├── aws/            # AWS-specific implementations
│   │   │   ├── azure/          # Azure-specific implementations
│   │   │   ├── gcp/            # GCP-specific implementations
│   │   │   └── digitalocean/   # DigitalOcean implementations
│   │   ├── persistence/        # Storage and caching layers
│   │   ├── terraform/          # Terraform state handling
│   │   └── notifications/      # External notification systems
│   ├── interfaces/             # API and UI interfaces
│   │   ├── api/                # REST, WebSocket, gRPC endpoints
│   │   └── cli/                # CLI formatters and handlers
│   └── shared/                 # Shared utilities and cross-cutting concerns
│       ├── config/             # Configuration management
│       ├── credentials/        # Credential handling
│       └── resilience/         # Circuit breakers and retry logic
├── pkg/                        # Public packages for external use
├── web/                        # Web UI assets
└── docs/                       # Documentation

```

### Key Components

#### Domain Layer
- **Resource Management**: Models, fingerprinting, and categorization of cloud resources
- **Drift Detection**: Core algorithms for identifying configuration drift
- **State Analysis**: Terraform state parsing and aggregation
- **Remediation Engine**: Safe, automated drift correction

#### Application Layer
- **Discovery Service**: Orchestrates multi-cloud resource discovery
- **Monitoring Service**: Real-time event processing and continuous monitoring
- **Analysis Service**: Resource graphs, historical tracking, and predictions
- **Workflow Engine**: Automated workflows for common operations

#### Infrastructure Layer
- **Cloud Adapters**: Provider-specific implementations for AWS, Azure, GCP, DigitalOcean
- **Smart Caching**: Multi-strategy caching (LRU, LFU, ARC, Predictive)
- **Event Integration**: CloudTrail, EventBridge, and other event sources
- **Persistence**: Database, file system, and distributed cache support

## Data Flow Architecture

### High-Level Data Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│                           USER INTERACTIONS                         │
│  CLI Commands │ Web UI │ REST API │ WebSocket │ Terraform Plugin   │
└────────┬──────────┬────────┬──────────┬────────────┬───────────────┘
         │          │        │          │            │
         ▼          ▼        ▼          ▼            ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         INTERFACE LAYER                             │
│   Command Handlers │ HTTP Handlers │ WebSocket Hub │ API Gateway    │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       APPLICATION LAYER                             │
│                                                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌─────────────┐              │
│  │  Discovery   │  │   Analysis   │  │ Remediation │              │
│  │   Service    │◄─►│   Service    │◄─►│   Service   │              │
│  └──────┬───────┘  └──────┬───────┘  └──────┬──────┘              │
│         │                  │                  │                     │
│  ┌──────▼───────┐  ┌──────▼───────┐  ┌──────▼──────┐              │
│  │  Monitoring  │  │    Graph     │  │   Workflow   │              │
│  │   Service    │◄─►│   Builder    │◄─►│   Engine    │              │
│  └──────────────┘  └──────────────┘  └─────────────┘              │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         DOMAIN LAYER                                │
│                                                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌─────────────┐              │
│  │   Resource   │  │    Drift     │  │    State     │              │
│  │    Models    │  │   Detector   │  │   Analyzer   │              │
│  └──────────────┘  └──────────────┘  └─────────────┘              │
│                                                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌─────────────┐              │
│  │ Fingerprint  │  │  Validation  │  │   Business   │              │
│  │    Engine    │  │    Rules     │  │    Rules     │              │
│  └──────────────┘  └──────────────┘  └─────────────┘              │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      INFRASTRUCTURE LAYER                           │
│                                                                      │
│  ┌─────────────────────────────────────────────────┐               │
│  │              CLOUD PROVIDERS                     │               │
│  │   AWS SDK │ Azure SDK │ GCP SDK │ DO SDK        │               │
│  └─────────────────────────────────────────────────┘               │
│                                                                      │
│  ┌─────────────────────────────────────────────────┐               │
│  │              PERSISTENCE & CACHING               │               │
│  │   SQLite │ File Cache │ Redis │ Smart Cache     │               │
│  └─────────────────────────────────────────────────┘               │
│                                                                      │
│  ┌─────────────────────────────────────────────────┐               │
│  │              EXTERNAL INTEGRATIONS               │               │
│  │   Git │ Terraform │ Notifications │ Monitoring  │               │
│  └─────────────────────────────────────────────────┘               │
└──────────────────────────────────────────────────────────────────────┘
```

### Detailed Data Flow Sequences

#### 1. Resource Discovery Flow

```
User Request → Discovery Service → Provider Selection
                                        │
                                        ▼
                              ┌─────────────────────┐
                              │ Credential Manager  │
                              │ (Auto-detection)    │
                              └─────────┬───────────┘
                                        │
                    ┌───────────────────┼───────────────────┐
                    ▼                   ▼                   ▼
              AWS Provider        Azure Provider      GCP Provider
                    │                   │                   │
                    ▼                   ▼                   ▼
              API Calls           API Calls          API Calls
                    │                   │                   │
                    └───────────────────┴───────────────────┘
                                        │
                                        ▼
                              Resource Aggregation
                                        │
                                        ▼
                    ┌───────────────────┴───────────────────┐
                    │                                       │
                    ▼                                       ▼
              Smart Cache                           Resource Store
              (Predictive)                         (Persistence)
                    │                                       │
                    └───────────────────┬───────────────────┘
                                        │
                                        ▼
                                  Return Results
```

#### 2. Drift Detection Flow

```
Scheduled/Manual Trigger → Drift Detector
                                │
                    ┌───────────┴───────────┐
                    ▼                       ▼
            Load State Files        Discover Cloud Resources
                    │                       │
                    ▼                       ▼
            Parse & Normalize        Normalize & Index
                    │                       │
                    └───────────┬───────────┘
                                │
                                ▼
                        Resource Comparison
                                │
                    ┌───────────┴───────────┐
                    │                       │
                    ▼                       ▼
            Smart Filtering          Generate Drift Report
            (75-85% noise            │
             reduction)              ▼
                    │          Categorize by Severity
                    │                       │
                    └───────────┬───────────┘
                                │
                                ▼
                    ┌───────────────────────┐
                    │   Drift Analysis      │
                    │   - Security Impact   │
                    │   - Cost Impact       │
                    │   - Compliance Risk   │
                    └───────────┬───────────┘
                                │
                                ▼
                        Store & Notify
```

#### 3. State File Discovery Flow

```
Auto-Discovery Trigger → State File Manager
                                │
                ┌───────────────┼───────────────────┐
                ▼               ▼                   ▼
        Local Filesystem   Git Repository    Cloud Backends
                │               │                   │
                ▼               ▼                   ▼
        Scan .tfstate     Mine Git History    Query S3/Azure/GCS
                │               │                   │
                └───────────────┴───────────────────┘
                                │
                                ▼
                        State File Analysis
                                │
                    ┌───────────┴───────────┐
                    ▼                       ▼
            Extract Resources        Detect Backend Type
                    │                       │
                    ▼                       ▼
            Build Resource Map      Store Metadata
                    │                       │
                    └───────────┬───────────┘
                                │
                                ▼
                        Update State Registry
```

#### 4. Real-Time Event Processing Flow

```
Cloud Events (CloudTrail/EventBridge) → Event Listener
                                            │
                                            ▼
                                    Event Normalization
                                            │
                                ┌───────────┴───────────┐
                                ▼                       ▼
                        Resource Creation       Configuration Change
                                │                       │
                                ▼                       ▼
                        Fingerprint Analysis    Compare with State
                                │                       │
                                ▼                       ▼
                        Detect Creation Method  Detect Drift
                        (Console/CLI/TF)              │
                                │                       │
                                └───────────┬───────────┘
                                            │
                                            ▼
                                    Update Cache & Notify
                                            │
                                ┌───────────┴───────────┐
                                ▼                       ▼
                        WebSocket Broadcast    Persist to Database
```

### Configuration Management Flow

```
┌──────────────────────────────────────────────────────────┐
│                  Configuration Sources                    │
│                  (Priority: High → Low)                   │
├──────────────────────────────────────────────────────────┤
│  1. Command-line flags (--auto-discover, --port, etc.)   │
│  2. Environment variables (DRIFTMGR_*)                    │
│  3. Configuration file (configs/config.yaml)              │
│  4. Default values (hardcoded)                            │
└────────────────────────┬─────────────────────────────────┘
                         │
                         ▼
              ┌──────────────────────┐
              │  Config Manager      │
              │  - Load & Parse      │
              │  - Apply Overrides   │
              │  - Validate          │
              │  - Hot Reload        │
              └──────────┬───────────┘
                         │
        ┌────────────────┼────────────────┐
        ▼                ▼                ▼
    Application     Server Config    Provider Config
    Settings        (Port, Auth)     (Credentials)
```

### Caching Strategy

```
┌─────────────────────────────────────────────────────────┐
│                     Smart Cache System                   │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │  LRU Cache   │  │  LFU Cache   │  │  ARC Cache   │ │
│  │  (Recent)    │  │  (Frequent)  │  │  (Adaptive)  │ │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘ │
│         └──────────────────┴──────────────────┘        │
│                            │                            │
│                            ▼                            │
│                  ┌──────────────────┐                  │
│                  │ Cache Warmer     │                  │
│                  │ (Predictive)     │                  │
│                  └──────────────────┘                  │
│                            │                            │
│                            ▼                            │
│                  ┌──────────────────┐                  │
│                  │ TTL Management   │                  │
│                  │ (24h default)    │                  │
│                  └──────────────────┘                  │
└─────────────────────────────────────────────────────────┘
```

### Advanced Features

#### Out-of-Band Detection Engine
DriftMgr employs multiple strategies to detect resources created outside Terraform:

1. **State File Discovery**
   - Deep recursive filesystem scanning
   - Git history mining for historical states
   - CI/CD artifact discovery
   - Remote backend exploration (S3, Azure Storage, GCS)
   - Terragrunt cache analysis

2. **Real-time Detection**
   - CloudTrail event processing
   - EventBridge integration
   - Resource creation pattern analysis
   - API call fingerprinting

3. **Intelligent Analysis**
   - Resource relationship graphing
   - Dependency mapping
   - Creation method detection (Console vs CLI vs Terraform)
   - Historical state reconstruction
   - Drift prediction based on patterns

4. **Resource Categorization**
   - MANAGED: In Terraform state
   - MANAGEABLE: Should be in Terraform
   - SHADOW_IT: Created outside approved processes
   - ORPHANED: Previously managed, now abandoned
   - TEMPORARY: Short-lived resources

## Web Interface

The DriftMgr web interface is a state-centric control panel for managing your infrastructure-as-code deployments.

### Navigation Structure

The interface is organized around state file operations:

1. **State Discovery** (Default View) - Auto-detect and catalog state files
2. **State Perspective** - Analyze infrastructure from state file viewpoint
3. **State vs Reality** - Compare state expectations with cloud resources
4. **State Overview** - Dashboard with health metrics and statistics
5. **Resources** - Browse all discovered cloud resources
6. **Drift Detection** - Identify configuration drift
7. **Remediation** - Fix drift with safety checks

### Key Features

#### State File Dashboard
- Real-time discovery status
- State file health indicators (fresh/recent/stale/abandoned)
- Resource count badges
- Backend type grouping (S3, Azure Storage, GCS, local)
- Terragrunt detection and module hierarchy

#### Interactive Visualizations
- **State Galaxy View** - Force-directed graph showing state files as solar systems
- **Tree Map** - Hierarchical view sized by resource count
- **Sankey Diagram** - Resource flow from state to providers to types
- **Dependency Graph** - Interactive resource relationship visualization
- **Timeline View** - Historical state file modifications

#### Split-Screen Perspective View
- **Left Panel**: "Through State's Eyes" - What Terraform knows
- **Right Panel**: "Reality Check" - What actually exists in cloud
- Color-coded resource status (managed, out-of-band, conflicted)
- One-click import command generation

## State File Discovery

DriftMgr automatically discovers Terraform state files across your infrastructure.

### Automatic Discovery

```bash
# Start auto-discovery via API
curl -X POST http://localhost:8080/api/v1/state/discovery/start \
  -H "Content-Type: application/json" \
  -d '{
    "paths": ["/terraform", "~/infrastructure"],
    "cloud_backends": [
      {
        "type": "s3",
        "config": {
          "bucket": "terraform-states",
          "region": "us-west-2"
        }
      }
    ],
    "auto_scan": true,
    "scan_interval_minutes": 5
  }'
```

### Discovery Sources

DriftMgr scans multiple sources for state files:

1. **Local Filesystem**
   - Recursive directory scanning
   - `.terraform` directory detection
   - Backup file recognition

2. **Cloud Backends**
   - S3 buckets with versioning support
   - Azure Storage containers
   - Google Cloud Storage buckets
   - Terraform Cloud/Enterprise workspaces

3. **Version Control**
   - Git repository scanning
   - Branch and tag analysis
   - History exploration

4. **Terragrunt Projects**
   - `terragrunt.hcl` detection
   - Dependency graph construction
   - Parent configuration inheritance

### State File Health Scoring

Each discovered state file receives a health assessment:

```
Health Score Calculation:
- Age: Fresh (<24h), Recent (<7d), Stale (<30d), Abandoned (>30d)
- Size: Warnings at 10MB, Critical at 50MB
- Resource Count: Warnings at 500, Critical at 1000
- Backend: Cloud backends score higher than local
- Versioning: Points for backend versioning enabled
```

## Perspective Analysis

The perspective feature provides a unique view of your infrastructure from each state file's point of view.

### Generating a Perspective

```bash
# Via CLI
driftmgr perspective generate --state-file terraform.tfstate

# Via API
curl -X POST http://localhost:8080/api/v1/perspective/{state-file-id}/generate
```

### Perspective Components

1. **Managed Resources** - Resources known to the state file
2. **Out-of-Band Resources** - Cloud resources not in state
3. **Conflicts** - Mismatches between state and reality
4. **Dependencies** - Resource relationship mapping
5. **Statistics** - Coverage percentages and adoption opportunities

### Coverage Analytics

The perspective analysis calculates infrastructure coverage:

```
Coverage = (Managed Resources / Total Cloud Resources) × 100

Categories:
- Excellent: >90% coverage
- Good: 70-90% coverage  
- Fair: 50-70% coverage
- Poor: <50% coverage
```

## Visualizations

DriftMgr provides multiple visualization types for understanding your infrastructure.

### State Galaxy View

An interactive 3D force-directed graph where:
- State files are solar systems
- Resources orbit their state files
- Out-of-band resources are rogue planets
- Links show dependencies
- Colors indicate health status

### Tree Map

A hierarchical visualization where:
- Rectangle size represents resource count
- Color represents health (green=healthy, yellow=warning, red=critical)
- Click to zoom into specific state files
- Hover for detailed statistics

### Resource Dependency Graph

An interactive network diagram showing:
- Resources as nodes
- Dependencies as directed edges
- Color coding by resource status
- Drag-and-drop repositioning
- Zoom and pan capabilities

### Timeline Visualization

A chronological view displaying:
- State file modifications
- Terraform apply events
- Drift detection events
- Import operations
- Configuration changes

## Out-of-Band Resources

DriftMgr excels at identifying resources created outside of Terraform through multiple advanced detection methods.

### Detection Methods

#### 1. Comprehensive State Discovery
- Searches entire filesystem for `.tfstate` files
- Mines Git history for historical states
- Scans CI/CD artifacts and build outputs
- Discovers Terragrunt caches
- Explores remote backends (S3, Azure Storage, GCS)

#### 2. Real-time Cloud Events
- Monitors CloudTrail for AWS resource creation
- Integrates with Azure Activity Log
- Processes GCP Cloud Audit Logs
- Detects resources within seconds of creation

#### 3. Resource Fingerprinting
DriftMgr analyzes resource characteristics to determine creation method:
- **Console-created**: Default names, missing tags, manual configurations
- **CLI-created**: Specific parameter patterns, automation indicators
- **Terraform-created**: Consistent naming, proper tags, state references
- **CloudFormation/ARM**: Stack references, template metadata

### Resource Categories

**MANAGED** - Currently in a Terraform state file
**MANAGEABLE** - Should be managed by Terraform (high-value resources)
**SHADOW_IT** - Created outside approved processes
**ORPHANED** - Previously managed but removed from state
**TEMPORARY** - Short-lived resources (Lambda invocations, temp storage)
**UNMANAGEABLE** - System/default resources that shouldn't be managed

### Import Scoring

Each out-of-band resource receives an import score (0-100):
```
Score Factors:
- Resource importance (security groups: +30, tags: +5)
- Naming convention compliance (+10)
- Tag compliance (+10)
- Dependencies on managed resources (+20)
- Age of resource (+10 if recent)
- Environment (production: +10)
```

Resources with scores > 70 are recommended for immediate import.

### Adoption Workflow

1. **Review** - Examine out-of-band resources in the perspective view
2. **Generate Import Commands** - Automatic `terraform import` generation
3. **Preview** - Dry-run to see impact
4. **Execute** - Run imports with safety checks
5. **Verify** - Confirm resources are now managed

```bash
# Generate import commands
curl http://localhost:8080/api/v1/perspective/{id}/import-commands

# Preview adoption
curl -X POST http://localhost:8080/api/v1/perspective/{id}/adopt/preview

# Execute adoption
curl -X POST http://localhost:8080/api/v1/perspective/{id}/adopt \
  -d '{"resource_ids": ["resource-1", "resource-2"], "dry_run": false}'
```

## Drift Detection

While DriftMgr focuses on state intelligence, it retains powerful drift detection capabilities.

### Intelligent Filtering

DriftMgr reduces noise by 75-85% through:
- Ignoring cosmetic changes (tags, descriptions)
- Prioritizing security-critical drift
- Grouping related changes
- Suppressing expected modifications

### Detection Modes

```bash
# Quick drift check
driftmgr check

# Analyze state for drift
driftmgr state analyze --file terraform.tfstate

# Compare state with reality
driftmgr state compare --file terraform.tfstate

# Check workspace drift
driftmgr workspace --path .
```

### Drift Categories

**Critical**
- Security group exposures
- Encryption disabled
- Public access enabled
- Authentication weakened

**Important**
- Backup configuration changes
- Capacity modifications
- Network settings altered
- Version downgrades

**Informational**
- Tag updates
- Description changes
- Metadata modifications

## Command Reference

### Core Commands

| Command | Description |
|---------|-------------|
| `serve web` | Start the web interface and API server |
| `discover` | Discover cloud resources across providers |
| `state analyze` | Analyze Terraform state files |
| `state inspect` | Display state file contents |
| `state compare` | Compare multiple state files |
| `perspective generate` | Generate state file perspective |
| `perspective out-of-band` | Identify unmanaged resources |
| `check` | Quick drift check - is it safe to apply? |
| `workspace` | Compare drift across Terraform workspaces |

### State Management Commands

```bash
# Scan for Terraform backend configurations
driftmgr state scan

# List and analyze state files
driftmgr state list

# Inspect state file contents
driftmgr state inspect --file terraform.tfstate

# Analyze state file for drift
driftmgr state analyze --file terraform.tfstate

# Compare multiple state files
driftmgr state compare --files terraform.tfstate,terraform.tfstate.backup

# Generate perspective
driftmgr perspective generate --state-file terraform.tfstate

# List out-of-band resources
driftmgr perspective out-of-band --state-file terraform.tfstate
```

### Web Server Commands

```bash
# Start with default settings (port 8080)
driftmgr serve web

# Start on custom port
driftmgr serve web --port 9090

# Start web server with specific port
driftmgr serve web -p 8081
```

## API Reference

The DriftMgr API provides programmatic access to all features.

### Base URL
```
http://localhost:8080/api/v1
```

### Key Endpoints

#### State Discovery
- `POST /state/discovery/start` - Start state file discovery
- `GET /state/discovery/status` - Get discovery status
- `GET /state/discovery/results` - Get discovered state files
- `POST /state/discovery/auto` - Configure auto-discovery

#### State Management
- `GET /state/files` - List all state files
- `GET /state/files/{id}` - Get state file details
- `POST /state/files/{id}/refresh` - Refresh state file analysis
- `POST /state/files/{id}/analyze` - Deep analysis of state file

#### Perspective Operations
- `GET /perspective/{id}` - Get cached perspective
- `POST /perspective/{id}/generate` - Generate new perspective
- `GET /perspective/{id}/out-of-band` - Get out-of-band resources
- `GET /perspective/{id}/conflicts` - Get conflicts
- `GET /perspective/{id}/graph` - Get dependency graph
- `POST /perspective/compare` - Compare two perspectives

#### Adoption Operations
- `POST /perspective/{id}/adopt` - Adopt out-of-band resources
- `POST /perspective/{id}/adopt/preview` - Preview adoption
- `GET /perspective/{id}/import-commands` - Get import commands

#### Search and Filter
- `POST /state/search` - Search state files
- `POST /state/filter` - Filter with complex criteria

### WebSocket Support

Real-time updates via WebSocket:
```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  switch(data.type) {
    case 'discovery_progress':
      updateProgressBar(data.progress);
      break;
    case 'state_updated':
      refreshStateView(data.state_id);
      break;
    case 'drift_detected':
      showDriftNotification(data.drift);
      break;
  }
};
```

## Configuration

### Configuration File

```yaml
# driftmgr.yaml
app:
  name: driftmgr
  environment: production
  log_level: info

# State Discovery Settings
state_discovery:
  auto_scan: true
  scan_interval: 5m
  scan_paths:
    - /terraform
    - ~/infrastructure
    - /git/repos
  
  backends:
    s3:
      enabled: true
      buckets:
        - terraform-states
        - infrastructure-states
      regions:
        - us-west-2
        - us-east-1
    
    azurerm:
      enabled: true
      storage_accounts:
        - tfstatestorage
      containers:
        - tfstate
    
    gcs:
      enabled: true
      buckets:
        - terraform-state-bucket

# Perspective Settings
perspective:
  cache_ttl: 30m
  auto_refresh: true
  adoption_recommendations: true
  conflict_detection: aggressive

# Visualization Settings
visualizations:
  enable_3d: true
  max_nodes: 1000
  auto_layout: force-directed
  color_scheme: health-based

# Provider Settings
providers:
  aws:
    enabled: true
    regions:
      - us-west-2
      - us-east-1
    rate_limit: 20
    
  azure:
    enabled: true
    subscriptions:
      - production
      - staging
    rate_limit: 15
    
  gcp:
    enabled: true
    projects:
      - my-project-123
    rate_limit: 20

# Drift Detection Settings
drift:
  sensitivity: medium
  smart_filter: true
  ignore_tags:
    - LastModified
    - CreatedBy

# Performance Settings
performance:
  cache_ttl: 5m
  max_connections: 100
  discovery_timeout: 30s
  workers: 10
```

### Environment Variables

```bash
# State Discovery
export DRIFTMGR_STATE_SCAN_PATHS="/terraform,~/infrastructure"
export DRIFTMGR_STATE_SCAN_INTERVAL=5m
export DRIFTMGR_AUTO_DISCOVER=true

# Backends
export DRIFTMGR_S3_BUCKETS="terraform-states,infrastructure-states"
export DRIFTMGR_AZURE_STORAGE_ACCOUNTS="tfstatestorage"
export DRIFTMGR_GCS_BUCKETS="terraform-state-bucket"

# Performance
export DRIFTMGR_WORKERS=10
export DRIFTMGR_CACHE_TTL=10m

# Logging
export DRIFTMGR_LOG_LEVEL=debug
```

## Production Features

### Enterprise Capabilities

#### State Governance
- **Automatic Discovery** - Continuous scanning for new state files
- **Health Monitoring** - Proactive alerts for stale or oversized states
- **Version Tracking** - State file version history and rollback
- **Backup Management** - Automatic state file backups

#### Security & Compliance
- **Encrypted Storage** - AES-256-GCM encryption for credentials
- **Audit Logging** - Complete audit trail with retention policies
- **RBAC Support** - Role-based access control (Admin, Operator, Viewer, Approver)
- **Compliance Modes** - SOC2, HIPAA, and PCI-DSS templates

#### Resilience
- **Circuit Breakers** - Prevent cascade failures
- **Health Endpoints** - Kubernetes-ready probes
- **Rate Limiting** - Provider-specific throttling
- **Graceful Degradation** - Continue operating with partial failures

#### Observability
- **Metrics Export** - Prometheus-compatible metrics
- **Distributed Tracing** - OpenTelemetry support
- **Structured Logging** - JSON-formatted logs
- **Performance Profiling** - Built-in pprof endpoints

### High Availability

```yaml
# Kubernetes Deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: driftmgr
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: driftmgr
        image: driftmgr:latest
        ports:
        - containerPort: 8080
        env:
        - name: DRIFTMGR_HA_MODE
          value: "true"
        - name: DRIFTMGR_CACHE_BACKEND
          value: "redis"
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8080
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
```

### Monitoring

```bash
# Health check endpoints
curl http://localhost:8080/health
curl http://localhost:8080/health/live
curl http://localhost:8080/health/ready

# Metrics endpoint
curl http://localhost:8080/metrics

# Cache statistics
curl http://localhost:8080/api/v1/cache/stats
```

## Troubleshooting

### Common Issues

#### State Files Not Discovered
```bash
# Check scan paths
driftmgr state discovery status

# Manually trigger scan
curl -X POST http://localhost:8080/api/v1/state/discovery/start

# Check permissions
ls -la /terraform
```

#### Perspective Generation Fails
```bash
# Check state file accessibility
driftmgr state analyze --file terraform.tfstate

# Verify cloud credentials
driftmgr status

# Check cache
curl http://localhost:8080/api/v1/cache/stats
```

#### Out-of-Band Resources Missing
```bash
# Ensure discovery is complete
driftmgr discover --all

# Force perspective regeneration
curl -X POST http://localhost:8080/api/v1/perspective/{id}/generate

# Check provider regions
driftmgr discover --provider aws --regions all
```

#### WebSocket Connection Issues
```bash
# Check WebSocket endpoint
wscat -c ws://localhost:8080/ws

# Verify server is running
netstat -an | grep 8080

# Check browser console for errors
# Open browser developer tools
```

### Debug Mode

```bash
# Enable debug logging
export DRIFTMGR_LOG_LEVEL=debug

# Start with verbose output
driftmgr serve web --debug

# Enable trace logging for specific components
export DRIFTMGR_TRACE=discovery,perspective,visualization
```

## Docker Deployment

```bash
# Build image
docker build -t driftmgr:latest .

# Run container
docker run -it --rm \
  -p 8080:8080 \
  -e AWS_PROFILE=default \
  -e AZURE_SUBSCRIPTION_ID=xxx \
  -v ~/.aws:/root/.aws:ro \
  -v ~/terraform:/terraform:ro \
  driftmgr:latest \
  serve web

# Docker Compose
docker-compose up -d
```

## Performance Optimization

### Smart Caching System

DriftMgr features an advanced multi-strategy caching system:

#### Cache Strategies
- **LRU (Least Recently Used)** - Default strategy for general caching
- **LFU (Least Frequently Used)** - For frequently accessed resources
- **ARC (Adaptive Replacement Cache)** - Self-tuning cache algorithm
- **Predictive** - ML-based prefetching for anticipated access patterns

#### Cache Layers
1. **Memory Cache** - Sharded in-memory cache with microsecond access
2. **Redis Cache** - Distributed cache for multi-instance deployments
3. **State File Cache** - 30-minute TTL for parsed state files
4. **Perspective Cache** - 30-minute TTL with dependency tracking
5. **Resource Cache** - 5-minute TTL with real-time invalidation
6. **Graph Cache** - Pre-computed relationship graphs

#### Performance Features
- **Predictive Preloading** - Anticipates and preloads frequently accessed data
- **Compression** - Automatic compression for large cache entries
- **Sharding** - CPU-count based sharding for concurrent access
- **Memory Management** - Automatic eviction under memory pressure

### Scaling Considerations

| Component | Scaling Limit | Optimization |
|-----------|--------------|--------------|
| State Files | 10,000+ | Pagination and filtering |
| Resources per State | 5,000+ | Incremental loading |
| Visualization Nodes | 1,000 | Clustering and aggregation |
| Concurrent Users | 100+ | WebSocket connection pooling |
| API Requests | 1,000/sec | Rate limiting and queuing |

## Security

### Security Features

- **Zero-Trust Architecture** - All operations require authentication
- **Encrypted Communication** - TLS 1.3 for all connections
- **Secrets Management** - Integration with HashiCorp Vault
- **Vulnerability Scanning** - Automatic security assessment
- **Compliance Reporting** - Generate compliance reports

### Best Practices

```bash
# Use read-only credentials for discovery
export AWS_PROFILE=readonly

# Enable audit logging
driftmgr serve web --audit-log /var/log/driftmgr/

# Restrict network access
driftmgr serve web --bind 127.0.0.1 --port 8080

# Use TLS certificates
driftmgr serve web --tls-cert cert.pem --tls-key key.pem
```

## Development

### Development Setup

```bash
# Clone repository
git clone https://github.com/catherinevee/driftmgr.git
cd driftmgr

# Install dependencies
go mod download

# Run tests
go test ./...

# Build the CLI
go build -o driftmgr ./cmd/driftmgr

# Build the server (optional)
go build -o driftmgr-server ./cmd/driftmgr-server

# Run locally
./driftmgr serve web
```

### Code Organization

The codebase follows Domain-Driven Design principles:

- **Domain logic** (`internal/domain/`) - Pure business logic with no external dependencies
- **Application services** (`internal/application/`) - Use case orchestration
- **Infrastructure** (`internal/infrastructure/`) - External integrations
- **Interfaces** (`internal/interfaces/`) - API and CLI handlers
- **Shared utilities** (`internal/shared/`) - Cross-cutting concerns

### Key Packages

```go
// Core domain models
import "github.com/catherinevee/driftmgr/internal/domain/resource"
import "github.com/catherinevee/driftmgr/internal/domain/drift"
import "github.com/catherinevee/driftmgr/internal/domain/state"

// Application services
import "github.com/catherinevee/driftmgr/internal/application/discovery"
import "github.com/catherinevee/driftmgr/internal/application/monitoring"
import "github.com/catherinevee/driftmgr/internal/application/analysis"

// Cloud providers
import "github.com/catherinevee/driftmgr/internal/infrastructure/cloud/aws"
import "github.com/catherinevee/driftmgr/internal/infrastructure/cloud/azure"
import "github.com/catherinevee/driftmgr/internal/infrastructure/cloud/gcp"
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/domain/resource

# Run integration tests
go test ./tests/integration -tags=integration

# Run with race detection
go test -race ./...
```

### Building for Production

```bash
# Build with optimizations
go build -ldflags="-s -w" -o driftmgr ./cmd/driftmgr

# Cross-compile for different platforms
GOOS=linux GOARCH=amd64 go build -o driftmgr-linux ./cmd/driftmgr
GOOS=darwin GOARCH=amd64 go build -o driftmgr-mac ./cmd/driftmgr
GOOS=windows GOARCH=amd64 go build -o driftmgr.exe ./cmd/driftmgr

# Build Docker image
docker build -t driftmgr:latest .
```

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## License

DriftMgr is licensed under the MIT License. See [LICENSE](LICENSE) for details.

## Support

- **Documentation**: [GitHub Wiki](https://github.com/catherinevee/driftmgr/wiki)
- **Issues**: [GitHub Issues](https://github.com/catherinevee/driftmgr/issues)
- **Discussions**: [GitHub Discussions](https://github.com/catherinevee/driftmgr/discussions)

## Acknowledgments

DriftMgr leverages:
- D3.js for interactive visualizations
- Alpine.js for reactive UI components
- Terraform state parsing libraries
- Cloud provider SDKs

---

**Built for Infrastructure Engineers by Infrastructure Engineers**

*Transforming Terraform state management from reactive to proactive*