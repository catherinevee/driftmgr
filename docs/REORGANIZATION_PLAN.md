# DriftMgr Root Directory Reorganization Plan

## Current State
35+ files in root directory causing clutter

## Proposed Structure

```
driftmgr/
├── README.md                    (KEEP in root - main documentation)
├── LICENSE                      (KEEP in root - legal requirement)
├── Makefile                     (KEEP in root - build entry point)
├── go.mod                       (KEEP in root - Go requirement)
├── go.sum                       (KEEP in root - Go requirement)
├── .gitignore                   (KEEP in root - Git requirement)
├── driftmgr.exe                 (KEEP in root - main executable)
│
├── docs/                        (EXISTING - expand usage)
│   ├── CHANGELOG.md            (MOVE from root)
│   ├── CONTRIBUTING.md         (MOVE from root)
│   ├── SECURITY.md             (MOVE from root)
│   ├── BUILD_TROUBLESHOOTING.md (MOVE from root)
│   ├── SECURITY_AUDIT_CHECKLIST.md (MOVE from root)
│   ├── SECURITY_CONFIG.md     (MOVE from root)
│   ├── testing/
│   │   ├── BETA_TESTING_PLAN.md (MOVE from root)
│   │   ├── TEST_EXECUTION_RESULTS.md (MOVE from root)
│   │   └── TESTING_SUMMARY.md (MOVE from root)
│   └── installation/
│       ├── INSTALLER_SUMMARY.md (MOVE from root)
│       └── CLEANUP_COMPLETE.md (MOVE from root)
│
├── examples/                    (EXISTING - add more examples)
│   ├── state-files/
│   │   ├── test.tfstate        (MOVE from root)
│   │   ├── complex.tfstate     (MOVE from root)
│   │   ├── cloud-state.tfstate (MOVE from root)
│   │   ├── terragrunt-prod.tfstate (MOVE from root)
│   │   ├── aws-local-discovered.tfstate (MOVE from root)
│   │   ├── drift-test.tfstate  (MOVE from root)
│   │   ├── minimal.tfstate     (MOVE from root)
│   │   ├── partial.tfstate     (MOVE from root)
│   │   ├── s3-demo.tfstate     (MOVE from root)
│   │   └── s3-from-bucket.tfstate (MOVE from root)
│   ├── reports/
│   │   ├── drift-report.md     (MOVE from root)
│   │   └── perspective-report.md (MOVE from root)
│   └── discovery/
│       ├── discovery-results.json (MOVE from root)
│       └── aws-discovery-output.json (MOVE from root)
│
├── scripts/                     (EXISTING - consolidate scripts)
│   ├── remediation/
│   │   ├── remediate-drift.bat (MOVE from root)
│   │   └── remediate-drift.sh  (MOVE from root)
│   └── test.sh                  (MOVE from root)
│
├── data/                        (NEW - for temporary data)
│   ├── lifecycle.json           (MOVE from root)
│   └── test-viz.json            (MOVE from root)
│
├── build/                       (NEW - for build artifacts)
│   └── driftmgr-client.exe     (MOVE from root)
│
└── .archive/                    (EXISTING - for old files)

```

## Benefits

1. **Cleaner Root**: Only essential files remain
2. **Logical Grouping**: Related files together
3. **Better Navigation**: Easier to find files
4. **Professional Structure**: Standard Go project layout
5. **Git-Friendly**: Less clutter in git status

## Files to Keep in Root

- README.md (project entry point)
- LICENSE (legal requirement)
- Makefile (build commands)
- go.mod/go.sum (Go modules)
- .gitignore (Git configuration)
- driftmgr.exe (main executable)
- docker-compose.yml (Docker setup)

## Migration Commands

```bash
# Create new directories
mkdir -p examples/state-files
mkdir -p examples/reports
mkdir -p examples/discovery
mkdir -p scripts/remediation
mkdir -p data
mkdir -p build
mkdir -p docs/testing
mkdir -p docs/installation

# Move state files
mv *.tfstate examples/state-files/

# Move documentation
mv CHANGELOG.md CONTRIBUTING.md SECURITY*.md docs/
mv *TESTING*.md *TEST*.md docs/testing/
mv *INSTALLER*.md *CLEANUP*.md docs/installation/

# Move reports
mv *-report.md examples/reports/

# Move discovery results
mv *discovery*.json examples/discovery/

# Move scripts
mv remediate-*.* scripts/remediation/
mv test.sh scripts/

# Move data files
mv *.json data/

# Move build artifacts
mv driftmgr-client.exe build/
```

## Post-Migration Tasks

1. Update README.md with new file locations
2. Update .gitignore if needed
3. Update any hardcoded paths in code
4. Test all functionality
5. Update CI/CD pipelines if they reference moved files