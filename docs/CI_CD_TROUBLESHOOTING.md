# CI/CD Troubleshooting Guide for DriftMgr

This guide helps diagnose and fix common issues with the GitHub Actions CI/CD pipeline.

## Table of Contents
- [Quick Diagnostics](#quick-diagnostics)
- [Common Build Issues](#common-build-issues)
- [Test Failures](#test-failures)
- [Security Scan Issues](#security-scan-issues)
- [Release Problems](#release-problems)
- [Docker Issues](#docker-issues)
- [Platform-Specific Issues](#platform-specific-issues)
- [Performance Problems](#performance-problems)
- [Emergency Fixes](#emergency-fixes)

## Quick Diagnostics

### Check Pipeline Status
```bash
# View all workflow runs
gh run list --limit 10

# View specific workflow status
gh workflow view build.yml
gh workflow view test.yml
gh workflow view security.yml

# Get details of failed run
gh run view --log-failed

# Download artifacts from failed run
gh run download [RUN_ID]
```

### Validate Workflow Files
```bash
# Check YAML syntax
yamllint .github/workflows/*.yml

# Validate with act (local testing)
act --list
act --job build --dryrun

# Check for common issues
grep -r "uses:" .github/workflows/ | grep -v "@v"  # Find unversioned actions
```

## Common Build Issues

### Issue: Go Module Download Failures

**Symptoms:**
```
Error: failed to download module: timeout downloading module
```

**Solutions:**
```yaml
# 1. Increase timeout in workflow
- name: Install dependencies
  run: go mod download
  timeout-minutes: 10  # Increase from default 6

# 2. Add retry logic
- name: Install dependencies with retry
  uses: nick-invision/retry@v2
  with:
    timeout_minutes: 10
    max_attempts: 3
    command: go mod download

# 3. Use module proxy
- name: Install dependencies
  env:
    GOPROXY: https://proxy.golang.org,direct
  run: go mod download
```

### Issue: Build Fails on Specific Platform

**Windows Build Failure:**
```yaml
# Common fix for Windows-specific issues
- name: Build on Windows
  if: matrix.os == 'windows-latest'
  run: |
    go build -tags windows -o driftmgr.exe ./cmd/driftmgr
  env:
    CGO_ENABLED: 0  # Disable CGO for Windows

# Alternative: Skip problematic tests on Windows
- name: Run tests
  run: |
    if [ "$RUNNER_OS" == "Windows" ]; then
      go test -short ./...
    else
      go test -race ./...
    fi
  shell: bash
```

**macOS Build Failure:**
```yaml
# Fix for macOS code signing issues
- name: Build on macOS
  if: matrix.os == 'macos-latest'
  run: |
    go build -ldflags="-s -w" -o driftmgr ./cmd/driftmgr
    # Skip code signing in CI
    codesign --remove-signature driftmgr || true
```

### Issue: Out of Memory During Build

```yaml
# Limit parallel builds
- name: Build with limited parallelism
  run: |
    export GOMAXPROCS=2
    go build -p 2 ./...

# Or increase runner size (requires GitHub Teams/Enterprise)
jobs:
  build:
    runs-on: ubuntu-latest-8-cores  # Larger runner
```

## Test Failures

### Issue: Tests Pass Locally but Fail in CI

**Check timezone differences:**
```go
// Fix: Use UTC in tests
func TestTimeFunction(t *testing.T) {
    // Bad
    now := time.Now()
    
    // Good
    now := time.Now().UTC()
}
```

**Check file path separators:**
```go
// Fix: Use filepath package
import "path/filepath"

// Bad
path := "configs/test.yaml"

// Good
path := filepath.Join("configs", "test.yaml")
```

### Issue: Integration Tests Timeout

```yaml
# Increase timeout for integration tests
- name: Run integration tests
  run: go test -v -tags=integration -timeout 20m ./...
  timeout-minutes: 25

# Or skip in CI if not critical
- name: Run integration tests
  if: github.event_name == 'push' && github.ref == 'refs/heads/main'
  run: go test -v -tags=integration ./...
```

### Issue: Race Condition Detected

```bash
# Reproduce locally
go test -race -count=10 ./...

# Common fixes:
# 1. Add mutex locks
# 2. Use sync.Once for initialization
# 3. Use channels for communication
```

**Example Fix:**
```go
// Before (race condition)
var cache map[string]string

func GetCache(key string) string {
    if cache == nil {
        cache = make(map[string]string)
    }
    return cache[key]
}

// After (fixed)
var (
    cache map[string]string
    mu    sync.RWMutex
)

func GetCache(key string) string {
    mu.RLock()
    defer mu.RUnlock()
    
    if cache == nil {
        mu.RUnlock()
        mu.Lock()
        if cache == nil {
            cache = make(map[string]string)
        }
        mu.Unlock()
        mu.RLock()
    }
    return cache[key]
}
```

## Security Scan Issues

### Issue: Gosec False Positives

```yaml
# Suppress specific warnings
- name: Run Gosec
  run: |
    gosec -fmt sarif -out gosec.sarif \
      -exclude=G104,G304 \  # Exclude specific rules
      -exclude-dir=test \    # Exclude test directory
      ./...
```

**In code:**
```go
// Suppress in source
func readFile(path string) {
    // #nosec G304 - path is validated above
    data, _ := ioutil.ReadFile(path)
}
```

### Issue: Vulnerability in Dependencies

```bash
# Update specific dependency
go get -u github.com/vulnerable/package@latest
go mod tidy

# Or temporarily ignore (add to .github/workflows/security.yml)
- name: Run Nancy
  continue-on-error: true  # Temporary until fix available
  run: |
    go list -json -deps ./... | nancy sleuth
```

### Issue: CodeQL Analysis Timeout

```yaml
# Optimize CodeQL performance
- name: Initialize CodeQL
  uses: github/codeql-action/init@v3
  with:
    languages: go
    queries: security-and-quality  # Use predefined query suite
    
- name: Build for CodeQL
  run: |
    # Build only necessary code
    go build -mod=readonly ./cmd/...
```

## Release Problems

### Issue: Release Workflow Not Triggered

```bash
# Check tag format
git tag -l  # List tags
git tag -d v1.0.0  # Delete local tag
git push --delete origin v1.0.0  # Delete remote tag

# Create proper tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# Or trigger manually
gh workflow run release.yml -f version=v1.0.0
```

### Issue: Asset Upload Fails

```yaml
# Fix file path issues
- name: Upload Release Asset
  uses: actions/upload-release-asset@v1
  with:
    upload_url: ${{ steps.create_release.outputs.upload_url }}
    asset_path: ./dist/driftmgr-linux-amd64.tar.gz  # Use ./ prefix
    asset_name: driftmgr-linux-amd64.tar.gz
    asset_content_type: application/gzip
```

### Issue: Changelog Generation Fails

```bash
# Ensure conventional commits
git log --oneline | head -10

# Fix: Use conventional commit format
git commit -m "feat: add new feature"
git commit -m "fix: resolve bug"
git commit -m "docs: update README"
```

## Docker Issues

### Issue: Docker Build Fails

```dockerfile
# Common fixes in Dockerfile

# 1. Fix certificate issues
FROM golang:1.23-alpine AS builder
RUN apk add --no-cache ca-certificates git

# 2. Fix timezone data
RUN apk add --no-cache tzdata

# 3. Handle private dependencies
ARG GITHUB_TOKEN
RUN git config --global url."https://${GITHUB_TOKEN}:@github.com/".insteadOf "https://github.com/"
```

### Issue: Docker Hub Rate Limits

```yaml
# Use GitHub Container Registry instead
- name: Build and push Docker image
  uses: docker/build-push-action@v5
  with:
    push: true
    tags: |
      ghcr.io/${{ github.repository }}:latest
      ghcr.io/${{ github.repository }}:${{ github.sha }}
```

### Issue: Multi-platform Build Fails

```yaml
# Fix QEMU issues
- name: Set up QEMU
  uses: docker/setup-qemu-action@v3
  with:
    platforms: linux/amd64,linux/arm64

# Limit platforms if needed
- name: Build Docker image
  uses: docker/build-push-action@v5
  with:
    platforms: linux/amd64  # Single platform for testing
```

## Platform-Specific Issues

### Windows-Specific

```yaml
# Handle path differences
- name: Fix Windows paths
  if: runner.os == 'Windows'
  run: |
    $env:PATH = "$env:PATH;$pwd"
    go build -o driftmgr.exe ./cmd/driftmgr
  shell: powershell

# Handle line endings
- name: Configure Git
  if: runner.os == 'Windows'
  run: |
    git config --global core.autocrlf false
    git config --global core.eol lf
```

### Linux-Specific

```yaml
# Install required system packages
- name: Install Linux dependencies
  if: runner.os == 'Linux'
  run: |
    sudo apt-get update
    sudo apt-get install -y build-essential
```

### macOS-Specific

```yaml
# Handle Homebrew dependencies
- name: Install macOS dependencies
  if: runner.os == 'macOS'
  run: |
    brew install pkg-config
    brew link --force pkg-config
```

## Performance Problems

### Issue: Workflows Running Slowly

```yaml
# 1. Use dependency caching
- uses: actions/setup-go@v5
  with:
    go-version: '1.23'
    cache: true
    cache-dependency-path: go.sum

# 2. Run jobs in parallel
strategy:
  matrix:
    test-suite: [unit, integration, e2e]

# 3. Skip unnecessary steps
- name: Skip on docs changes
  if: |
    !contains(github.event.head_commit.message, '[skip ci]') &&
    !contains(github.event.pull_request.labels.*.name, 'documentation')
```

### Issue: Cache Not Working

```bash
# Check cache key
- uses: actions/cache@v3
  with:
    path: |
      ~/.cache/go-build
      ~/go/pkg/mod
    key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
    restore-keys: |
      ${{ runner.os }}-go-
      
# Debug cache
- name: Check cache
  run: |
    echo "Cache key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}"
    ls -la ~/.cache/go-build || echo "No build cache"
    du -sh ~/go/pkg/mod || echo "No module cache"
```

## Emergency Fixes

### Disable Failing Workflow Temporarily

```yaml
# Add to top of problematic workflow
on:
  workflow_dispatch:  # Manual trigger only
  # push:  # Temporarily disabled
  #   branches: [ main ]
```

### Skip Specific Jobs

```yaml
jobs:
  build:
    if: ${{ !contains(github.event.head_commit.message, '[skip build]') }}
    runs-on: ubuntu-latest
```

### Quick Rollback

```bash
# Revert last commit
git revert HEAD
git push origin main

# Or reset workflow files
git checkout origin/main -- .github/workflows/
git commit -m "fix: revert workflow changes"
git push
```

### Force Re-run Workflows

```bash
# Re-run failed workflow
gh run rerun [RUN_ID]

# Re-run specific failed jobs
gh run rerun [RUN_ID] --failed

# Cancel stuck workflow
gh run cancel [RUN_ID]
```

## Debug Techniques

### Enable Debug Logging

```yaml
# In workflow file
env:
  ACTIONS_RUNNER_DEBUG: true
  ACTIONS_STEP_DEBUG: true

# Or as repository secret
ACTIONS_RUNNER_DEBUG: true
ACTIONS_STEP_DEBUG: true
```

### Add Debug Steps

```yaml
- name: Debug Environment
  run: |
    echo "Event: ${{ github.event_name }}"
    echo "Ref: ${{ github.ref }}"
    echo "SHA: ${{ github.sha }}"
    echo "Runner: ${{ runner.os }}"
    echo "Go version: $(go version)"
    go env
    
- name: Debug Secrets
  run: |
    echo "Secret exists: ${{ secrets.CODECOV_TOKEN != '' }}"
    echo "Secret length: ${#CODECOV_TOKEN}"
  env:
    CODECOV_TOKEN: ${{ secrets.CODECOV_TOKEN }}
```

### Local Testing with Act

```bash
# Install act
brew install act  # macOS
# or
curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash

# Test workflows locally
act push  # Simulate push event
act pull_request  # Simulate PR
act -j build  # Run specific job

# With secrets
act --secret-file .env.secrets

# Verbose output
act -v --job build
```

## Getting Help

### Resources

1. **GitHub Actions Documentation:**
   - https://docs.github.com/en/actions

2. **DriftMgr Issues:**
   - https://github.com/catherinevee/driftmgr/issues

3. **Community Forum:**
   - https://github.com/orgs/community/discussions

### Logs and Artifacts

```bash
# Download all logs
gh run download [RUN_ID] --dir logs/

# Get specific job logs
gh run view [RUN_ID] --log --job [JOB_ID]

# Search logs
gh run view [RUN_ID] --log | grep -i error
```

### Reporting Issues

When reporting CI/CD issues, include:

1. **Workflow run URL**
2. **Error messages** (full output)
3. **Recent changes** (last 3 commits)
4. **Environment details** (OS, Go version)
5. **Steps to reproduce**

Template:
```markdown
## CI/CD Issue Report

**Workflow:** [build/test/security/release]
**Run URL:** https://github.com/catherinevee/driftmgr/actions/runs/XXX
**Status:** [Failed/Timeout/Cancelled]

**Error:**
```
[paste error message]
```

**Last Working Run:** [URL or "never worked"]
**Recent Changes:** [what changed]

**Attempted Fixes:**
- [ ] Checked workflow syntax
- [ ] Reviewed recent commits
- [ ] Tested locally with act
- [ ] Checked secrets configuration
```

## Prevention Checklist

Before pushing changes:

- [ ] Run tests locally: `go test ./...`
- [ ] Check linting: `golangci-lint run`
- [ ] Validate workflow files: `yamllint .github/workflows/*.yml`
- [ ] Test with act: `act --job build --dryrun`
- [ ] Review changed files: `git diff --staged`
- [ ] Check commit message format
- [ ] Ensure no secrets in code
- [ ] Update documentation if needed