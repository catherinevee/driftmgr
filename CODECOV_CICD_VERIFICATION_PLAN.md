# Codecov CI/CD Verification Plan

## Overview
Each phase of test implementation will be verified through the CI/CD pipeline to ensure:
- Tests pass in GitHub Actions environment
- Coverage metrics are accurately reported to Codecov
- No regression in existing tests
- Build remains stable across all platforms

## Phase-by-Phase CI/CD Verification Strategy

### 🔧 Phase 0: Pre-Implementation Setup
**Goal**: Ensure CI/CD pipeline is working correctly

#### Verification Steps:
```bash
# 1. Check current CI status
gh run list --repo catherinevee/driftmgr --limit 5

# 2. Verify Codecov integration
gh workflow run test-coverage.yml --repo catherinevee/driftmgr

# 3. Monitor Codecov dashboard
# https://app.codecov.io/gh/catherinevee/driftmgr
```

#### Success Criteria:
- [ ] GitHub Actions workflows run without infrastructure errors
- [ ] Codecov receives coverage reports
- [ ] Base coverage metric established (5.7%)

---

### 🚨 Phase 1: Build Failure Fixes (Day 1-2)
**Goal**: All packages compile and basic tests pass

#### Implementation:
1. Fix API package build failures
2. Fix CLI format string issues
3. Fix remediation strategy builds

#### CI/CD Verification:
```bash
# Create branch for fixes
git checkout -b fix/build-failures

# After fixes, push and create PR
git add .
git commit -m "fix: resolve build failures in API, CLI, and remediation packages"
git push origin fix/build-failures

# Create PR with gh CLI
gh pr create --title "Fix build failures for test coverage improvement" \
  --body "Fixes build failures to enable test coverage collection" \
  --repo catherinevee/driftmgr

# Monitor CI checks
gh pr checks --watch

# After CI passes, merge
gh pr merge --auto --squash
```

#### Success Criteria:
- [ ] All packages compile in CI
- [ ] No build failures in test workflow
- [ ] Coverage report generated (even if low)
- [ ] Codecov comment appears on PR

---

### 📊 Phase 2: API Package Tests (Day 3-5)
**Goal**: API package reaches 40% coverage

#### Implementation:
1. Create handler tests
2. Add middleware tests
3. Implement websocket tests

#### CI/CD Verification:
```bash
# Create feature branch
git checkout -b test/api-coverage

# Run tests locally first
go test ./internal/api/... -cover

# Push changes
git add internal/api/*_test.go
git commit -m "test: add comprehensive API package tests (0% -> 40%)"
git push origin test/api-coverage

# Create PR
gh pr create --title "Add API package tests - Phase 2" \
  --body "Implements comprehensive API tests to achieve 40% coverage" \
  --repo catherinevee/driftmgr

# Monitor specific test job
gh run watch --repo catherinevee/driftmgr

# Check coverage change
gh pr comment --body "Awaiting Codecov report for coverage verification"
```

#### Success Criteria:
- [ ] API package shows 40%+ coverage in Codecov
- [ ] All API tests pass in CI
- [ ] No timeout issues in CI
- [ ] Codecov shows coverage increase

---

### 💻 Phase 3: CLI & Remediation Tests (Day 6-10)
**Goal**: CLI reaches 35%, Remediation reaches 35%

#### Implementation:
1. CLI command tests
2. Output formatting tests
3. Remediation planner tests
4. Executor tests

#### CI/CD Verification:
```bash
# Create branch
git checkout -b test/cli-remediation

# Test locally with coverage
go test ./internal/cli/... ./internal/remediation/... -cover

# Commit and push
git add .
git commit -m "test: add CLI and remediation tests"
git push origin test/cli-remediation

# Create PR with detailed description
gh pr create --title "Phase 3: CLI and Remediation tests" \
  --body "$(cat <<EOF
## Coverage Targets
- CLI: 0% -> 35%
- Remediation: 0% -> 35%

## Tests Added
- Command execution tests
- Output formatting tests
- Planner logic tests
- Executor framework tests

## CI/CD Verification
- All tests pass locally
- Ready for CI validation
EOF
)"

# Wait for and verify CI
gh pr checks --watch
```

#### Success Criteria:
- [ ] CLI package shows 35%+ coverage
- [ ] Remediation package shows 35%+ coverage
- [ ] Total project coverage reaches 15%+
- [ ] CI completes within 10 minutes

---

### ☁️ Phase 4: Provider Enhancement (Week 3)
**Goal**: Improve all provider coverage

#### CI/CD Verification:
```bash
# Create branch for provider tests
git checkout -b test/provider-enhancement

# Test each provider individually
go test ./internal/providers/aws/... -cover
go test ./internal/providers/azure/... -cover
go test ./internal/providers/gcp/... -cover
go test ./internal/providers/digitalocean/... -cover

# Push incremental updates
git add internal/providers/
git commit -m "test: enhance provider test coverage"
git push origin test/provider-enhancement

# Create PR
gh pr create --title "Phase 4: Provider test enhancement" \
  --body "Enhances test coverage for all cloud providers"

# Monitor long-running tests
gh run view --log --repo catherinevee/driftmgr
```

#### Success Criteria:
- [ ] AWS: 65%+ coverage
- [ ] Azure: 50%+ coverage
- [ ] GCP: 50%+ coverage
- [ ] DigitalOcean: 40%+ coverage
- [ ] No provider tests timeout

---

### 🔄 Phase 5: Integration Tests (Week 5)
**Goal**: Add end-to-end test coverage

#### CI/CD Verification:
```bash
# Create integration test branch
git checkout -b test/integration

# Run integration tests with extended timeout
go test ./tests/integration/... -timeout 30m -cover

# Push changes
git add tests/integration/
git commit -m "test: add comprehensive integration tests"
git push origin test/integration

# Create PR with special CI considerations
gh pr create --title "Phase 5: Integration tests" \
  --body "$(cat <<EOF
## Integration Tests Added
- Multi-provider workflows
- Complete discovery flows
- Remediation scenarios

## CI Considerations
- Tests may take longer to run
- Requires mock cloud responses
- May need increased timeout
EOF
)"

# Monitor CI with focus on timeout issues
gh run watch --exit-status
```

#### Success Criteria:
- [ ] Integration tests complete in CI
- [ ] Overall coverage reaches 60%+
- [ ] No flaky test failures
- [ ] Codecov shows all packages improving

---

## CI/CD Monitoring Commands

### Real-time Monitoring
```bash
# Watch current workflow run
gh run watch --repo catherinevee/driftmgr

# View specific job logs
gh run view --log --job <job-id>

# Check PR status
gh pr checks --watch

# Get coverage from latest run
gh run download --name coverage-report
```

### Codecov Verification
```bash
# Check Codecov status via API
curl -X GET https://api.codecov.io/api/v2/github/catherinevee/repos/driftmgr \
  -H "Authorization: Bearer ${CODECOV_TOKEN}"

# View coverage trend
gh api repos/catherinevee/driftmgr/commits/HEAD/check-runs \
  --jq '.check_runs[] | select(.name | contains("codecov")) | .output'
```

### Troubleshooting CI Failures

#### Common Issues and Solutions:

1. **Test Timeouts**
```yaml
# Increase timeout in workflow
- name: Run tests
  run: go test ./... -timeout 30m -cover
```

2. **Coverage Upload Failures**
```yaml
# Retry codecov upload
- name: Upload coverage
  uses: codecov/codecov-action@v3
  with:
    file: ./coverage.out
    fail_ci_if_error: false
    verbose: true
    max_attempts: 3
```

3. **Flaky Tests**
```go
// Add retry logic for flaky tests
func TestWithRetry(t *testing.T) {
    maxRetries := 3
    for i := 0; i < maxRetries; i++ {
        if err := actualTest(); err == nil {
            return
        }
        if i < maxRetries-1 {
            time.Sleep(time.Second * 2)
        }
    }
    t.Fatal("Test failed after retries")
}
```

## GitHub Actions Workflow Updates

### Enhanced Test Coverage Workflow
```yaml
name: Test Coverage with Verification
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go-version: ['1.23']

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: ${{ matrix.go-version }}

      - name: Cache Go modules
        uses: actions/cache@v3
        with:
          path: ~/go/pkg/mod
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
          restore-keys: |
            ${{ runner.os }}-go-

      - name: Install dependencies
        run: go mod download

      - name: Run tests with coverage
        run: |
          go test -race -coverprofile=coverage.out -covermode=atomic ./...
          go tool cover -func=coverage.out

      - name: Check coverage threshold
        run: |
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          echo "Coverage: ${COVERAGE}%"
          if (( $(echo "$COVERAGE < 10" | bc -l) )); then
            echo "Coverage is below 10% threshold"
            exit 1
          fi

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out
          flags: unittests
          fail_ci_if_error: true
          verbose: true

      - name: Comment PR with coverage
        if: github.event_name == 'pull_request'
        uses: actions/github-script@v6
        with:
          script: |
            const coverage = // extract from coverage.out
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: `Coverage: ${coverage}%`
            })
```

## Success Metrics Dashboard

### Phase Completion Checklist

| Phase | Target Coverage | CI Status | Codecov Updated | PR Merged |
|-------|----------------|-----------|-----------------|-----------|
| Phase 1: Build Fixes | Compiles | ⬜ | ⬜ | ⬜ |
| Phase 2: API Tests | 40% | ⬜ | ⬜ | ⬜ |
| Phase 3: CLI/Remediation | 35% | ⬜ | ⬜ | ⬜ |
| Phase 4: Providers | 50%+ | ⬜ | ⬜ | ⬜ |
| Phase 5: Integration | 60%+ | ⬜ | ⬜ | ⬜ |
| Phase 6: Final Push | 80% | ⬜ | ⬜ | ⬜ |

### Daily CI/CD Verification
```bash
#!/bin/bash
# Daily verification script

echo "=== Daily CI/CD Verification ==="
echo "Date: $(date)"

# Check latest CI runs
echo -e "\n📊 Latest CI Runs:"
gh run list --repo catherinevee/driftmgr --limit 3

# Check current coverage
echo -e "\n📈 Current Coverage:"
curl -s https://codecov.io/api/gh/catherinevee/driftmgr | jq '.commit.totals.c'

# Check open PRs
echo -e "\n🔄 Open PRs:"
gh pr list --repo catherinevee/driftmgr

# Check failing tests
echo -e "\n❌ Any Failing Tests:"
gh run list --repo catherinevee/driftmgr --status failure --limit 1

echo -e "\n✅ Verification Complete"
```

## Conclusion

This CI/CD verification plan ensures that each phase of test implementation is properly validated through the GitHub Actions pipeline and Codecov integration. By verifying after each phase, we can:

1. **Catch issues early** before they compound
2. **Ensure accurate coverage reporting** to Codecov
3. **Maintain build stability** throughout the improvement process
4. **Track progress** with concrete metrics
5. **Prevent regression** in existing functionality

The plan emphasizes incremental validation, allowing for quick feedback and adjustment as needed.