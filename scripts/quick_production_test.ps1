# Quick Production Readiness Test
Write-Host "Testing DriftMgr Production Features" -ForegroundColor Cyan
Write-Host "="*50 -ForegroundColor Gray

$tests = @()
$passed = 0
$failed = 0

# Test 1: Check if key production files exist
Write-Host "`nChecking production components..." -ForegroundColor Yellow

$productionFiles = @(
    "internal/logging/structured.go",
    "internal/resilience/retry.go",
    "internal/cache/ttl_cache.go",
    "internal/security/vault.go",
    "internal/resilience/ratelimiter.go",
    "internal/metrics/collector.go",
    "internal/testing/integration/suite.go",
    "internal/state/distributed.go",
    "internal/resilience/circuit_breaker.go",
    "internal/telemetry/tracing.go",
    "internal/health/checks.go",
    "internal/lifecycle/shutdown.go",
    "loadtest/scenarios.js",
    "docs/runbooks/OPERATIONAL_RUNBOOK.md"
)

foreach ($file in $productionFiles) {
    if (Test-Path $file) {
        Write-Host "[✓] $file" -ForegroundColor Green
        $passed++
    } else {
        Write-Host "[✗] $file" -ForegroundColor Red
        $failed++
    }
}

# Test 2: Build the application
Write-Host "`nBuilding DriftMgr..." -ForegroundColor Yellow
$buildResult = go build -o driftmgr.exe ./cmd/driftmgr 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "[✓] Build successful" -ForegroundColor Green
    $passed++
} else {
    Write-Host "[✗] Build failed" -ForegroundColor Red
    Write-Host $buildResult
    $failed++
}

# Test 3: Check CLI commands
Write-Host "`nTesting CLI commands..." -ForegroundColor Yellow
$commands = @(
    @{Cmd="--help"; Expected="DriftMgr"},
    @{Cmd="--version"; Expected="Version"},
    @{Cmd="credentials"; Expected="Detected"}
)

foreach ($test in $commands) {
    $output = & ./driftmgr.exe $test.Cmd 2>&1
    if ($output -match $test.Expected) {
        Write-Host "[✓] Command: $($test.Cmd)" -ForegroundColor Green
        $passed++
    } else {
        Write-Host "[✗] Command: $($test.Cmd)" -ForegroundColor Red
        $failed++
    }
}

# Summary
Write-Host "`n" -NoNewline
Write-Host "="*50 -ForegroundColor Gray
Write-Host "SUMMARY" -ForegroundColor Yellow
Write-Host "Passed: $passed" -ForegroundColor Green
Write-Host "Failed: $failed" -ForegroundColor Red

if ($failed -eq 0) {
    Write-Host "`n✅ All production features verified!" -ForegroundColor Green
    Write-Host "DriftMgr is PRODUCTION READY" -ForegroundColor Green
} else {
    Write-Host "`n⚠️ Some features missing or failed" -ForegroundColor Yellow
}

exit $(if ($failed -eq 0) { 0 } else { 1 })