# DriftMgr Refactoring Plan

## Overview
This document outlines the systematic refactoring of DriftMgr to consolidate duplicate code, improve organization, and maintain all existing functionality.

## Current Issues
1. **43 files in discovery/** - Many duplicates (enhanced_discovery.go, enhanced_discovery_v2.go, universal_discovery.go)
2. **Multiple visualization implementations** scattered across packages
3. **Duplicate cost analyzers** in different locations
4. **Provider code mixed** with generic implementations
5. **Dashboard code fragmentation** across multiple handlers

## New Structure

```
internal/
├── core/                    # Core business logic
│   ├── discovery/          # Unified discovery engine
│   │   ├── discovery.go    # Main discovery interface
│   │   ├── manager.go      # Discovery orchestration
│   │   ├── cache.go        # Discovery caching
│   │   └── filters.go      # Resource filtering
│   ├── drift/              # Drift detection
│   │   ├── detector.go     # Main drift detection
│   │   ├── analyzer.go     # Drift analysis
│   │   ├── predictor.go    # Drift prediction
│   │   └── terraform.go    # Terraform-specific
│   ├── remediation/        # Remediation engine
│   │   ├── engine.go       # Main remediation
│   │   ├── planner.go      # Remediation planning
│   │   ├── executor.go     # Execution logic
│   │   └── rollback.go     # Rollback support
│   └── state/              # State management
│       ├── manager.go      # State operations
│       ├── parser.go       # State parsing
│       └── store.go        # State storage
│
├── providers/              # Provider-specific implementations
│   ├── aws/
│   │   ├── client.go       # AWS client setup
│   │   ├── discovery.go    # AWS discovery
│   │   ├── resources.go    # AWS resource types
│   │   └── drift.go        # AWS-specific drift
│   ├── azure/
│   │   ├── client.go       # Azure client setup
│   │   ├── discovery.go    # Azure discovery
│   │   ├── resources.go    # Azure resource types
│   │   └── drift.go        # Azure-specific drift
│   ├── gcp/
│   │   ├── client.go       # GCP client setup
│   │   ├── discovery.go    # GCP discovery
│   │   ├── resources.go    # GCP resource types
│   │   └── drift.go        # GCP-specific drift
│   └── digitalocean/
│       ├── client.go       # DO client setup
│       ├── discovery.go    # DO discovery
│       ├── resources.go    # DO resource types
│       └── drift.go        # DO-specific drift
│
├── api/                    # API layer
│   ├── rest/              # REST API
│   │   ├── server.go      # API server
│   │   ├── routes.go      # Route definitions
│   │   ├── handlers/      # Request handlers
│   │   └── middleware/    # API middleware
│   ├── websocket/         # WebSocket support
│   │   ├── server.go      # WS server
│   │   └── handlers.go    # WS handlers
│   └── grpc/              # gRPC (if needed)
│
├── analytics/             # Analytics and reporting
│   ├── cost/             # Cost analysis
│   │   ├── analyzer.go   # Unified cost analyzer
│   │   └── optimizer.go  # Cost optimization
│   ├── metrics/          # Metrics collection
│   └── reports/          # Report generation
│
├── storage/              # Data persistence
│   ├── datastore.go     # Main data store
│   ├── cache.go         # Caching layer
│   └── database.go      # Database operations
│
├── visualization/        # Visualization
│   ├── diagram.go       # Unified diagram generation
│   ├── graph.go         # Graph visualization
│   └── export.go        # Export functionality
│
├── security/            # Security features
│   ├── auth.go         # Authentication
│   ├── rbac.go         # Role-based access
│   └── encryption.go   # Encryption utilities
│
├── utils/              # Utilities
│   ├── logger.go       # Logging
│   ├── errors.go       # Error handling
│   └── helpers.go      # Helper functions
│
└── models/             # Data models
    ├── resource.go     # Resource model
    ├── drift.go        # Drift model
    └── config.go       # Configuration model
```

## Consolidation Mapping

### Discovery Consolidation
- **Keep**: `discovery.go` as base
- **Merge into `manager.go`**:
  - enhanced_discovery.go
  - enhanced_discovery_v2.go
  - universal_discovery.go
  - multi_account_discovery.go
  - parallel_discovery.go
- **Move to providers/**:
  - aws_comprehensive.go → providers/aws/discovery.go
  - azure_enhanced_discovery.go → providers/azure/discovery.go
  - gcp_enhanced_discovery.go → providers/gcp/discovery.go
  - digitalocean_enhanced_discovery.go → providers/digitalocean/discovery.go

### Dashboard Consolidation
- **Merge all handlers** into organized API structure:
  - discovery_handlers.go → api/rest/handlers/discovery.go
  - drift_handlers.go → api/rest/handlers/drift.go
  - remediation_handlers.go → api/rest/handlers/remediation.go
  - resource_handlers.go → api/rest/handlers/resources.go
  - credential_handlers.go → api/rest/handlers/credentials.go
  - analytics_handlers.go → api/rest/handlers/analytics.go
  - websocket_handlers.go → api/websocket/handlers.go

### Cost Analysis Consolidation
- **Merge into single location**:
  - internal/cost/cost_analyzer.go
  - internal/analysis/cost_analyzer.go
  - internal/drift/cost_calculator.go
  → analytics/cost/analyzer.go

### Visualization Consolidation
- **Merge all visualization**:
  - internal/visualization/*.go
  - internal/discovery/visualization.go
  → visualization/ (single package)

## Migration Steps

### Phase 1: Create New Structure
1. Create new directory structure
2. Create interfaces for core components
3. Set up provider abstraction layer

### Phase 2: Consolidate Core Components
1. Merge discovery implementations
2. Consolidate drift detection
3. Unify remediation engine
4. Merge state management

### Phase 3: Organize Provider Code
1. Extract AWS-specific code
2. Extract Azure-specific code
3. Extract GCP-specific code
4. Extract DigitalOcean-specific code

### Phase 4: Consolidate API Layer
1. Merge dashboard handlers
2. Organize REST endpoints
3. Consolidate WebSocket code

### Phase 5: Clean Up
1. Remove duplicate files
2. Update all imports
3. Fix tests
4. Update documentation

## Benefits
- **Reduce code by ~40%** through deduplication
- **Clearer separation** of concerns
- **Easier maintenance** and testing
- **Better performance** through unified caching
- **Simpler onboarding** for new developers

## Preserved Functionality
- All discovery capabilities
- All drift detection features
- Complete remediation workflow
- All provider support
- Full API compatibility
- All visualization features