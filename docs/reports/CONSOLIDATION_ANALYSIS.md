# DriftMgr Consolidation Plan

## Current State Analysis
- **Total Go files**: 397
- **Duplicate directories**: 890MB of duplication (driftmgr-clean: 588MB, driftmgr-restructured: 302MB)
- **Multiple command directories**: 10 main.go files across cmd/
- **Internal packages**: 31 separate packages with overlapping functionality

## Critical Issues Identified

### 1. Massive Directory Duplication (890MB waste)
- `driftmgr-clean/` (588MB) - Appears to be a previous refactoring attempt
- `driftmgr-restructured/` (302MB) - Another refactoring attempt
- Both contain complete copies of internal/, cmd/, configs/, etc.

### 2. Command Structure Fragmentation
```
cmd/
├── dashboard/           # Standalone dashboard
├── driftmgr/           # Main CLI (active)
├── driftmgr-client/    # Client-specific commands
├── driftmgr-server/    # Server mode
├── multi-account-discovery/  # Single-purpose discovery
├── server/             # Duplicate server?
├── services/           # Microservice commands
└── validate/           # Validation commands
```

### 3. Internal Package Overlap
- `internal/analysis/` + `internal/cost/` (cost analysis duplication)
- `internal/discovery/` (46 discovery-related files scattered)
- `internal/drift/` + multiple drift calculators
- `internal/web/` + `internal/dashboard/` + `web/`

### 4. Binary Proliferation
- 8 different driftmgr executables in root directory
- Each ~135MB (over 1GB total)

## Consolidation Strategy

### Phase 1: Remove Duplicate Directories (Immediate 890MB savings)
```bash
# SAFE TO DELETE (confirmed duplicates):
rm -rf driftmgr-clean/
rm -rf driftmgr-restructured/
rm driftmgr-*.exe  # Keep only driftmgr.exe
```

### Phase 2: Consolidate Commands
**Current**: 10 separate commands
**Target**: 3 main commands

```
cmd/
├── driftmgr/          # Main CLI (keep)
│   ├── main.go
│   ├── commands/      # Move subcommands here
│   │   ├── discover.go    # From cloud_discover.go
│   │   ├── state.go       # From state_*.go
│   │   ├── dashboard.go   # From cmd/dashboard/
│   │   ├── server.go      # From cmd/server/
│   │   └── validate.go    # From cmd/validate/
├── driftmgr-client/   # Keep for client-server mode
└── services/          # Keep for microservices
```

### Phase 3: Internal Package Consolidation
**Current**: 31 packages
**Target**: ~15 packages

#### Merge Candidates:
1. **Analysis Consolidation**:
   ```
   internal/analysis/     # Main analysis
   internal/cost/         # → Move to internal/analysis/cost/
   internal/drift/        # → Move to internal/analysis/drift/
   ```

2. **Discovery Consolidation**:
   ```
   internal/discovery/    # Keep main
   internal/cloud/        # → Move to internal/discovery/cloud/
   internal/multicloud/   # → Move to internal/discovery/multicloud/
   ```

3. **Infrastructure Consolidation**:
   ```
   internal/terraform/    # Keep
   internal/terragrunt/   # → Move to internal/terraform/terragrunt/
   internal/state/        # → Move to internal/terraform/state/
   ```

4. **UI Consolidation**:
   ```
   internal/dashboard/    # Main dashboard logic
   internal/web/          # → Move to internal/dashboard/web/
   internal/ui/           # → Move to internal/dashboard/ui/
   web/                   # → Move to internal/dashboard/static/
   ```

5. **Utilities Consolidation**:
   ```
   internal/utils/        # Keep
   internal/errors/       # → Move to internal/utils/errors/
   internal/monitoring/   # → Move to internal/utils/monitoring/
   ```

## Expected Results
- **File reduction**: 397 → ~250 Go files (-37%)
- **Package reduction**: 31 → ~15 packages (-52%)
- **Disk space savings**: 890MB+ immediate savings
- **Simplified architecture**: Cleaner dependency graph
- **Easier maintenance**: Fewer places to update code
- **Faster builds**: Fewer packages to compile

## Priority Order
1. **HIGH**: Remove duplicate directories (immediate space savings)
2. **MEDIUM**: Consolidate commands (developer experience)
3. **MEDIUM**: Internal package cleanup (maintainability)
4. **LOW**: Configuration cleanup (organization)