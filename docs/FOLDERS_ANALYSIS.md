# DriftMgr Folder Analysis - What Can Be Removed

## Folders Safe to Remove

### 1. **.archive/** [OK] SAFE TO REMOVE
- **Contains**: Old backup files (driftmgr.exe~, nul)
- **Size**: ~140MB (mostly old exe backup)
- **Used by**: Nothing
- **Impact**: None - just old backups

### 2. **ci-cd/** [OK] SAFE TO REMOVE
- **Contains**: CI/CD templates for various platforms
- **Why remove**: These are example configurations, not used by driftmgr runtime
- **Alternative**: Move to docs/ci-cd-examples if you want to keep as reference
- **Impact**: None on functionality

### 3. **.claude/** [OK] SAFE TO REMOVE
- **Contains**: Claude AI assistant files
- **Used by**: Only by Claude AI assistant, not driftmgr
- **Impact**: None

### 4. **visualizations/** [WARNING] KEEP BUT CLEAN
- **Contains**: Generated visualization outputs
- **Used by**: Code references this directory for output
- **Recommendation**: Keep directory but delete old generated files inside
- **Impact**: Would break visualization output if removed

## Folders to Keep

### Essential for Functionality:
- **cmd/** - Command implementations (REQUIRED)
- **internal/** - Core logic (REQUIRED)
- **configs/** - Configuration files (REQUIRED)
- **assets/** - Static files for dashboard (REQUIRED for web UI)

### Important for Development:
- **tests/** - Test suite (Keep for testing)
- **examples/** - Example files (Keep for documentation)
- **docs/** - Documentation (Keep for users)
- **scripts/** - Utility scripts (Keep for automation)

### Generated/Runtime:
- **build/** - Build outputs (Keep, but can clean contents)
- **data/** - Runtime data (Keep, but can clean contents)
- **.git/** - Version control (NEVER REMOVE)
- **.github/** - GitHub workflows (Keep for GitHub integration)

## Recommended Actions

```bash
# 1. Remove .archive folder (saves 140MB)
rm -rf .archive/

# 2. Remove ci-cd folder (or move to docs)
rm -rf ci-cd/
# OR
mv ci-cd/ docs/ci-cd-examples/

# 3. Remove .claude folder
rm -rf .claude/

# 4. Clean visualizations folder (keep directory)
rm -rf visualizations/*

# 5. Clean data folder contents (keep directory)
rm data/*.json data/*.db 2>/dev/null

# 6. Clean build folder contents (keep directory)
rm build/*.exe 2>/dev/null
```

## Space Savings

| Folder | Size | Action | Space Saved |
|--------|------|--------|-------------|
| .archive | ~140MB | Remove | 140MB |
| ci-cd | ~500KB | Remove/Move | 500KB |
| .claude | ~100KB | Remove | 100KB |
| visualizations/* | ~2MB | Clean contents | 2MB |
| **TOTAL** | | | **~142MB** |

## Folders That Look Removable But Aren't

- **assets/** - Used by dashboard server for web UI
- **configs/** - Required for smart defaults and auto-remediation
- **Makefile** - Needed for building
- **Jenkinsfile** - Keep if using Jenkins CI
- **docker-compose.yml** - Keep if using Docker

## Summary

You can safely remove 3 folders completely and clean 2 others, saving ~142MB and reducing clutter without impacting functionality.