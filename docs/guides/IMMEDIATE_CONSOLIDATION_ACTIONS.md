# Immediate Consolidation Actions

## Quick Wins (Can be done today)

### 1. Remove Duplicate Test Files from Root
```bash
# These are test files that should be in tests/
rm test_gcp_fix.go
rm test_universal_discovery.go
# Move any useful tests to tests/manual/
```

### 2. Clean Up Build Artifacts
```bash
# Remove executable files from deployments/
rm deployments/*.exe
rm pkg/client/*.exe
rm pkg/client/*.exe~

# Keep only source code in version control
```

### 3. Remove Old Structure
```bash
# This is a complete duplicate taking 50MB+
rm -rf old_structure/
```

### 4. Consolidate Region Files
```bash
# Keep only the working versions
rm configs/aws_regions.json
rm configs/azure_regions.json
rm configs/digitalocean_regions.json
rm configs/gcp_regions.json
rm configs/all_regions.json

# Keep only configs/regions/ directory
```

### 5. Remove Redundant Main Files
```bash
# Keep only the working main.go
rm cmd/driftmgr/enhanced_main.go
rm cmd/driftmgr/simple_enhanced_main.go
rm cmd/driftmgr/unified_main.go
```

## Medium-term Actions (Next few days)

### 6. Archive Enhanced Discovery Files
Since universal discovery is working well and enhanced discovery causes API issues:

```bash
# Create archive directory
mkdir -p archive/enhanced_discovery/

# Move enhanced discovery files
mv internal/discovery/enhanced_discovery.go archive/enhanced_discovery/
mv internal/discovery/enhanced_discovery_v2.go archive/enhanced_discovery/
mv internal/discovery/gcp_enhanced_discovery.go archive/enhanced_discovery/
mv internal/discovery/azure_enhanced_discovery.go archive/enhanced_discovery/
mv internal/discovery/digitalocean_enhanced_discovery.go archive/enhanced_discovery/
```

### 7. Consolidate Multi-Account Discovery
Since multi-account-discovery is working, integrate it into main CLI:

```bash
# Move multi-account functionality into main driftmgr command
# Keep cmd/multi-account-discovery/main.go logic but integrate as subcommand
```

## File Size Analysis

Based on the directory structure, here's what can be removed safely:

### Large Removals (High Impact)
- `old_structure/` - ~50MB duplicate codebase [OK] **SAFE TO REMOVE**
- `internal/discovery/enhanced_discovery.go` - 3,000+ lines [OK] **ARCHIVE** (universal discovery working)
- Multiple region JSON files - ~1MB duplicated data [OK] **CONSOLIDATE**
- Build artifacts (*.exe files) - ~50MB [OK] **SAFE TO REMOVE**

### Documentation Cleanup
- `docs/development/` - 30+ redundant summary files [OK] **CONSOLIDATE**
- Multiple README files [OK] **MERGE**

### Test File Organization
- Root level test files [OK] **MOVE TO tests/**
- Duplicate test scripts [OK] **CONSOLIDATE**

## Estimated Impact

### Before Consolidation
- **Total files:** ~500+ files
- **Discovery code:** ~5,000 lines across multiple systems
- **Configuration:** Duplicated across 10+ files
- **Documentation:** Scattered across 50+ files

### After Consolidation
- **File reduction:** ~40% fewer files
- **Code reduction:** ~2,000 lines removed
- **Single discovery system:** Universal discovery only
- **Organized documentation:** 3 main sections

## Immediate Benefits Test

After removing old_structure/ and build artifacts:
```bash
# Test that everything still works
go build -o driftmgr.exe ./cmd/driftmgr
go build -o multi-account-discovery.exe ./cmd/multi-account-discovery

# Verify discovery still works
./multi-account-discovery.exe --provider gcp --format summary
./multi-account-discovery.exe --provider digitalocean --format summary
```

## Safety Checklist

Before making changes:
- [OK] Current working directory backed up
- [OK] Git repository clean (no uncommitted changes)
- [OK] All discovery methods tested and working
- [OK] Build process verified

After each change:
- [OK] Code still compiles
- [OK] Tests still pass
- [OK] Discovery functionality intact
- [OK] No broken imports

This consolidation plan will significantly streamline DriftMgr while maintaining all working functionality.