# DriftMgr Directory Refactoring Plan

## Goal
Consolidate duplicate code and create a clean, maintainable directory structure while preserving ALL current functionality.

## Current Issues
1. **Azure implementations scattered across 3 locations**
   - internal/discovery/azure*.go (3 files)
   - internal/providers/azure/*.go (5 files)
   - Duplicate azure_windows_cli.go in both locations

2. **Multiple discovery implementations**
   - internal/discovery/ (25 files)
   - internal/core/discovery/ (4 files)  
   - internal/providers/ (provider-specific discovery)

3. **Version sprawl**
   - enhanced_discovery.go
   - enhanced_discovery_v2.go
   - enhanced_engine.go

4. **Cache duplication**
   - internal/cache/
   - internal/utils/cache/

## Refactoring Strategy

### Phase 1: Consolidate Providers
**Move all provider-specific code to internal/providers/**

```
internal/providers/
├── aws/
│   ├── discovery.go (from internal/discovery/aws.go)
│   ├── comprehensive.go (existing)
│   └── role_assumption.go (existing)
├── azure/
│   ├── discovery.go (merge all azure discovery files)
│   ├── comprehensive.go (existing)
│   ├── expanded_resources.go (existing)
│   └── windows_cli.go (deduplicate)
├── gcp/
│   ├── discovery.go (from internal/discovery/gcp.go)
│   ├── comprehensive.go (existing)
│   └── expanded_resources.go (existing)
└── digitalocean/
    ├── discovery.go (merge DO files)
    └── comprehensive.go (existing)
```

### Phase 2: Consolidate Discovery Engine
**Keep only one discovery implementation in internal/core/discovery/**

```
internal/core/discovery/
├── engine.go (merge engine.go + enhanced_engine.go)
├── discovery.go (orchestration logic)
├── filters.go (from advanced_filtering.go)
├── cache.go (caching logic)
├── parallel.go (from parallel_discovery.go)
└── validation.go (existing)
```

### Phase 3: Clean Core Structure
**Organize core business logic**

```
internal/core/
├── discovery/     # Discovery orchestration
├── drift/         # Drift detection logic
├── remediation/   # Remediation logic
├── visualization/ # Visualization logic
└── models.go      # Shared models
```

### Phase 4: Consolidate Utilities
**Merge duplicate utility packages**

```
internal/utils/
├── cache/         # Single cache implementation
├── retry/         # Retry logic
├── osutil/        # OS utilities
└── performance/   # Performance utilities
```

### Phase 5: Clean Up Scripts and Docs
**Organize auxiliary files**

```
scripts/
├── build/         # Build scripts
├── deploy/        # Deployment scripts
├── install/       # Installation scripts
└── test/          # Test scripts

docs/
├── api/           # API documentation
├── deployment/    # Deployment guides
├── development/   # Development guides
└── user-guide/    # User documentation
```

## File Mappings

### Consolidations:
1. **Azure Discovery**: Merge these into `internal/providers/azure/discovery.go`:
   - internal/discovery/azure.go
   - internal/discovery/azure_enhanced_discovery.go
   - internal/providers/azure/azure_enhanced_discovery.go

2. **Enhanced Discovery**: Merge into `internal/core/discovery/engine.go`:
   - internal/discovery/enhanced_discovery.go
   - internal/discovery/enhanced_discovery_v2.go
   - internal/discovery/enhanced_engine.go

3. **Cache**: Keep only `internal/utils/cache/cache.go`:
   - Remove internal/cache/cache.go
   - Remove internal/core/discovery/cache.go

4. **DigitalOcean**: Merge into `internal/providers/digitalocean/discovery.go`:
   - internal/discovery/digitalocean.go
   - internal/discovery/digitalocean_provider.go

## Import Updates Required
All files importing from old locations will need updates:
- `"github.com/catherinevee/driftmgr/internal/discovery"` → provider-specific imports
- Cache imports → `"github.com/catherinevee/driftmgr/internal/utils/cache"`
- Discovery engine imports → `"github.com/catherinevee/driftmgr/internal/core/discovery"`

## Verification Steps
1. Build test after each phase
2. Run existing tests
3. Verify all CLI commands still work
4. Check web dashboard functionality
5. Ensure Docker build succeeds

## Rollback Plan
- Git commit before starting
- Test at each phase
- Revert if any functionality breaks