# Recommended .gitignore Additions for DriftMgr

## Current Issues Found
Based on repository analysis, the following files/patterns should be added to .gitignore:

### Files Currently Tracked That Should Be Ignored:
1. `driftmgr-client.exe` - Binary file
2. `driftmgr-server.exe` - Binary file
3. `driftmgr.exe` - Binary file (already has pattern but specific names exist)
4. `coverage.out` - Test coverage file
5. `driftmgr.db` - Database file (already has *.db pattern)
6. `internal/logging/logger_old.go.bak` - Backup file

## Recommended Additions

### 1. Go-specific
```gitignore
# Go binaries and build artifacts
/driftmgr
/driftmgr-client
/driftmgr-server
/driftmgr-client.exe
/driftmgr-server.exe
/driftmgr.exe
*.exe
*.dll
*.so
*.dylib
*.test
*.out

# Go coverage and profiling
coverage.out
coverage.html
coverage.txt
*.coverprofile
*.prof
*.pprof
cpu.prof
mem.prof

# Go workspace
go.work
go.work.sum

# Vendor directory (if using vendoring)
vendor/
```

### 2. IDE and Editor Files
```gitignore
# Visual Studio Code
.vscode/
*.code-workspace
.history/

# IntelliJ IDEA / GoLand
.idea/
*.iml
*.iws
*.ipr
out/

# Vim
*.swp
*.swo
*.swn
.vim/
*~

# Emacs
*~
\#*\#
.\#*

# Sublime Text
*.sublime-workspace
*.sublime-project

# Visual Studio
*.suo
*.user
*.userosscache
*.sln.docstates
.vs/
```

### 3. Environment and Configuration
```gitignore
# Environment files
.env
.env.local
.env.*.local
*.env

# Local configuration
local.yaml
local.yml
*-local.yaml
*-local.yml
configs/local/
.driftmgr.local.yaml

# Secrets and credentials
*.pem
*.key
*.crt
*.p12
*.pfx
credentials.json
service-account.json
*-credentials.json
*-service-account.json

# Claude settings (already exists but should be consistent)
.claude/
```

### 4. Testing and Development
```gitignore
# Test artifacts
*.test
*.bench
testdata/tmp/
test-results/
test-reports/
junit.xml
test.json

# Temporary files
*.tmp
*.temp
*.bak
*.backup
*.old
*.orig
tmp/
temp/

# Debug files
debug
debug.test
*.debug
dlv
__debug_bin
```

### 5. Build and Dependencies
```gitignore
# Build directories
build/
dist/
out/
bin/
release/

# Dependency directories
vendor/
node_modules/
.terraform/
.terragrunt-cache/

# Package files
*.tar.gz
*.zip
*.tar
*.tgz
```

### 6. Documentation and Reports
```gitignore
# Generated documentation
site/
_site/
docs/_build/
*.pdf

# Reports (keep samples, ignore generated)
reports/*.html
reports/*.json
reports/*.csv
!reports/examples/
```

### 7. Docker and Containers
```gitignore
# Docker volumes (local development)
postgres-data/
redis-data/
grafana-data/
prometheus-data/
driftmgr-data/
driftmgr-logs/

# Docker override files
docker-compose.override.yml
docker-compose.*.yml
!docker-compose.test.yml
!docker-compose.prod.yml
```

### 8. Cloud Provider Files
```gitignore
# AWS
.aws/
aws-exports.js

# Azure
.azure/
azure-pipelines.yml.local

# GCP
.gcloud/
gcloud-key.json

# Terraform
*.tfstate
*.tfstate.*
*.tfplan
.terraform/
.terraform.lock.hcl
terraform.tfvars
override.tf
override.tf.json
*_override.tf
*_override.tf.json
```

### 9. Logs and Databases
```gitignore
# Logs
*.log
logs/
*.log.*
npm-debug.log*
yarn-debug.log*
yarn-error.log*

# Databases
*.db
*.sqlite
*.sqlite3
*.db-shm
*.db-wal
data/
```

### 10. OS-specific
```gitignore
# macOS
.DS_Store
.AppleDouble
.LSOverride
Icon
._*
.Spotlight-V100
.Trashes

# Windows
Thumbs.db
ehthumbs.db
Desktop.ini
$RECYCLE.BIN/
*.lnk

# Linux
.directory
.Trash-*
.nfs*
```

### 11. Python (for scripts)
```gitignore
# Python
__pycache__/
*.py[cod]
*$py.class
*.pyc
.Python
*.egg-info/
.pytest_cache/
.mypy_cache/
.coverage
htmlcov/
```

### 12. Miscellaneous
```gitignore
# Archives
*.tar.gz
*.rar
*.7z
*.dmg
*.iso
*.jar

# Certificates and keys (be careful not to commit)
*.pem
*.crt
*.key
*.p12
ca-bundle.crt
cert.pem
key.pem

# Crash logs
crash.log
lerna-debug.log*

# Cache
.cache/
.parcel-cache/
```

## Recommended Complete .gitignore

Here's a complete .gitignore combining existing and new patterns:

```gitignore
# === Go ===
# Binaries
/driftmgr
/driftmgr-client
/driftmgr-server
*.exe
*.dll
*.so
*.dylib
*.test
*.out

# Coverage and profiling
coverage.*
*.coverprofile
*.prof
*.pprof

# Dependencies
vendor/
go.work
go.work.sum

# === Development ===
# IDE/Editors
.vscode/
.idea/
*.iml
*.swp
*.swo
*~
.vim/

# Environment
.env
.env.*
*.env
local.yaml
*-local.yaml
configs/local/

# Claude
.claude/

# === Testing ===
testdata/tmp/
test-results/
*.test
*.bench
junit.xml

# === Build ===
build/
dist/
bin/
release/

# === Cloud/Infrastructure ===
# Terraform
*.tfstate
*.tfstate.*
*.tfplan
.terraform/
.terraform.lock.hcl
terraform.tfvars
.terragrunt-cache/

# Docker volumes
postgres-data/
redis-data/
*-data/
*-logs/
docker-compose.override.yml

# === Security ===
# Credentials
*.pem
*.key
*.crt
*.p12
credentials.json
*-credentials.json
service-account*.json

# === Data ===
# Databases
*.db
*.sqlite*
data/

# Logs
*.log
logs/
*.log.*

# === OS ===
.DS_Store
Thumbs.db
Desktop.ini
$RECYCLE.BIN/
.Trash-*

# === Languages ===
# Python
__pycache__/
*.py[cod]
.pytest_cache/

# Node
node_modules/
npm-debug.log*

# === Temporary ===
*.tmp
*.temp
*.bak
*.backup
*.old
*.orig
tmp/
temp/

# === Archives ===
*.tar.gz
*.zip
*.rar
*.7z
```

## Action Items

1. **Remove tracked files that should be ignored:**
   ```bash
   git rm --cached driftmgr-client.exe driftmgr-server.exe driftmgr.exe
   git rm --cached coverage.out
   git rm --cached internal/logging/logger_old.go.bak
   ```

2. **Update .gitignore with recommended patterns**

3. **Commit the changes:**
   ```bash
   git add .gitignore
   git commit -m "Update .gitignore with complete patterns"
   ```

## Priority Additions

If you want to keep it minimal, these are the most important additions:

```gitignore
# Specific binaries (currently tracked)
/driftmgr-client.exe
/driftmgr-server.exe

# Coverage files
coverage.*

# Backup files
*.bak
*.old
*.orig

# Environment files
.env
.env.*

# Credentials
*-credentials.json
service-account*.json

# IDE
.vscode/
.idea/

# Temporary
*.tmp
tmp/
```