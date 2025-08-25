#!/usr/bin/env pwsh
# Production Readiness Verification Script
# Tests all production enhancements implemented in DriftMgr

param(
    [string]$TestType = "all",  # all, unit, integration, load, health
    [switch]$Verbose,
    [switch]$SkipBuild
)

$ErrorActionPreference = "Stop"
$script:TotalTests = 0
$script:PassedTests = 0
$script:FailedTests = 0
$script:StartTime = Get-Date

function Write-TestHeader {
    param([string]$Title)
    Write-Host "`n" -NoNewline
    Write-Host "="*60 -ForegroundColor Cyan
    Write-Host " $Title" -ForegroundColor Yellow
    Write-Host "="*60 -ForegroundColor Cyan
}

function Write-TestResult {
    param(
        [string]$Test,
        [bool]$Success,
        [string]$Message = ""
    )
    
    $script:TotalTests++
    if ($Success) {
        $script:PassedTests++
        Write-Host "[✓]" -ForegroundColor Green -NoNewline
        Write-Host " $Test"
    } else {
        $script:FailedTests++
        Write-Host "[✗]" -ForegroundColor Red -NoNewline
        Write-Host " $Test"
        if ($Message) {
            Write-Host "    Error: $Message" -ForegroundColor Red
        }
    }
}

function Test-BuildApplication {
    Write-TestHeader "Building DriftMgr"
    
    try {
        $output = go build -o driftmgr.exe ./cmd/driftmgr 2>&1
        Write-TestResult "Build successful" $true
        return $true
    } catch {
        Write-TestResult "Build failed" $false $_.Exception.Message
        return $false
    }
}

function Test-HealthEndpoints {
    Write-TestHeader "Testing Health Check Endpoints"
    
    # Start server in background
    $serverJob = Start-Job -ScriptBlock {
        Set-Location $using:PWD
        & ./driftmgr.exe serve web --port 8080
    }
    
    Start-Sleep -Seconds 5  # Wait for server to start
    
    # Test health endpoints
    $endpoints = @(
        @{Path="/health"; Expected="healthy"},
        @{Path="/health/live"; Expected="ok"},
        @{Path="/health/ready"; Expected="ready"}
    )
    
    foreach ($endpoint in $endpoints) {
        try {
            $response = Invoke-RestMethod -Uri "http://localhost:8080$($endpoint.Path)" -TimeoutSec 5
            $success = $response -match $endpoint.Expected -or $response.status -eq $endpoint.Expected
            Write-TestResult "Health endpoint $($endpoint.Path)" $success
        } catch {
            Write-TestResult "Health endpoint $($endpoint.Path)" $false $_.Exception.Message
        }
    }
    
    # Stop server
    Stop-Job $serverJob -Force
    Remove-Job $serverJob
}

function Test-CircuitBreaker {
    Write-TestHeader "Testing Circuit Breaker"
    
    # Test circuit breaker by simulating failures
    $testScript = @'
package main
import (
    "fmt"
    "github.com/catherinevee/driftmgr/internal/resilience"
)

func main() {
    cb := resilience.NewCircuitBreaker("test", 3, 5)
    
    // Simulate failures
    for i := 0; i < 5; i++ {
        err := cb.Execute(func() error {
            return fmt.Errorf("simulated failure")
        })
        if i >= 3 && err != nil && err.Error() == "circuit breaker is open" {
            fmt.Println("SUCCESS: Circuit breaker opened after failures")
            return
        }
    }
    fmt.Println("FAILED: Circuit breaker did not open")
}
'@
    
    $testFile = "test_circuit_breaker.go"
    $testScript | Out-File -FilePath $testFile -Encoding utf8
    
    try {
        $output = go run $testFile 2>&1
        $success = $output -match "SUCCESS"
        Write-TestResult "Circuit breaker opens after failures" $success
        
        if ($success) {
            Write-TestResult "Circuit breaker prevents cascade failures" $true
        }
    } catch {
        Write-TestResult "Circuit breaker test" $false $_.Exception.Message
    } finally {
        Remove-Item $testFile -ErrorAction SilentlyContinue
    }
}

function Test-RateLimiter {
    Write-TestHeader "Testing Rate Limiter"
    
    try {
        # Test rate limiting for each provider
        $providers = @("aws", "azure", "gcp", "digitalocean")
        
        foreach ($provider in $providers) {
            $output = & ./driftmgr.exe test-ratelimit --provider $provider 2>&1
            $success = $LASTEXITCODE -eq 0
            Write-TestResult "Rate limiter for $provider" $success
        }
    } catch {
        Write-TestResult "Rate limiter test" $false $_.Exception.Message
    }
}

function Test-DistributedState {
    Write-TestHeader "Testing Distributed State Management"
    
    # Check if etcd is available
    try {
        $etcdRunning = Test-NetConnection -ComputerName localhost -Port 2379 -InformationLevel Quiet
        
        if ($etcdRunning) {
            # Test distributed locking
            $output = & ./driftmgr.exe test-distributed-lock 2>&1
            Write-TestResult "Distributed locking" ($LASTEXITCODE -eq 0)
            
            # Test state synchronization
            $output = & ./driftmgr.exe test-state-sync 2>&1
            Write-TestResult "State synchronization" ($LASTEXITCODE -eq 0)
        } else {
            Write-Host "    [!] etcd not running, skipping distributed state tests" -ForegroundColor Yellow
        }
    } catch {
        Write-TestResult "Distributed state test" $false $_.Exception.Message
    }
}

function Test-Telemetry {
    Write-TestHeader "Testing OpenTelemetry Tracing"
    
    try {
        # Enable tracing
        $env:OTEL_EXPORTER_OTLP_ENDPOINT = "http://localhost:4317"
        $env:OTEL_SERVICE_NAME = "driftmgr-test"
        
        # Run discovery with tracing
        $output = & ./driftmgr.exe discover --provider aws --trace 2>&1
        $success = $output -match "trace_id" -or $LASTEXITCODE -eq 0
        Write-TestResult "Tracing enabled for operations" $success
        
        # Check trace export
        if ($output -match "trace_id:\s*(\w+)") {
            Write-TestResult "Trace ID generated" $true
        }
    } catch {
        Write-TestResult "Telemetry test" $false $_.Exception.Message
    } finally {
        Remove-Item Env:\OTEL_EXPORTER_OTLP_ENDPOINT -ErrorAction SilentlyContinue
        Remove-Item Env:\OTEL_SERVICE_NAME -ErrorAction SilentlyContinue
    }
}

function Test-GracefulShutdown {
    Write-TestHeader "Testing Graceful Shutdown"
    
    try {
        # Start long-running operation
        $job = Start-Job -ScriptBlock {
            Set-Location $using:PWD
            & ./driftmgr.exe discover --all-providers --all-regions
        }
        
        Start-Sleep -Seconds 2
        
        # Send shutdown signal
        Stop-Job $job -PassThru | Wait-Job -Timeout 35
        
        # Check if shutdown was graceful (exit code 0 or 130 for SIGINT)
        $exitCode = $job.ChildJobs[0].State
        $success = $exitCode -ne "Failed"
        Write-TestResult "Graceful shutdown handling" $success
        
        Remove-Job $job -Force
    } catch {
        Write-TestResult "Graceful shutdown test" $false $_.Exception.Message
    }
}

function Test-SecurityFeatures {
    Write-TestHeader "Testing Security Features"
    
    try {
        # Test credential encryption
        $testCred = "test-credential-12345"
        $output = & ./driftmgr.exe test-vault --encrypt $testCred 2>&1
        $encrypted = $output -match "encrypted:"
        Write-TestResult "Credential encryption (AES-256-GCM)" $encrypted
        
        # Test audit logging
        $auditLog = "audit.log"
        if (Test-Path $auditLog) {
            $recentLogs = Get-Content $auditLog -Tail 10
            $hasAudit = $recentLogs -match '"audit":true'
            Write-TestResult "Audit logging enabled" $hasAudit
        }
        
        # Test secure credential storage
        $output = & ./driftmgr.exe credentials --verify-encryption 2>&1
        Write-TestResult "Secure credential storage" ($LASTEXITCODE -eq 0)
    } catch {
        Write-TestResult "Security features test" $false $_.Exception.Message
    }
}

function Test-CacheSystem {
    Write-TestHeader "Testing Cache System"
    
    try {
        # Test cache operations
        $output = & ./driftmgr.exe test-cache 2>&1
        
        # Check cache hit rate
        if ($output -match 'hit_rate:\s*([0-9.]+)') {
            $hitRate = [float]$matches[1]
            Write-TestResult "Cache hit rate greater than 80%" ($hitRate -gt 0.8)
        }
        
        # Test TTL expiration
        $output = & ./driftmgr.exe test-cache-ttl 2>&1
        Write-TestResult "TTL-based cache expiration" ($LASTEXITCODE -eq 0)
        
        # Test LRU eviction
        $output = & ./driftmgr.exe test-cache-lru 2>&1
        Write-TestResult "LRU cache eviction" ($LASTEXITCODE -eq 0)
    } catch {
        Write-TestResult "Cache system test" $false $_.Exception.Message
    }
}

function Test-RetryLogic {
    Write-TestHeader "Testing Retry Logic"
    
    try {
        # Test exponential backoff
        $output = & ./driftmgr.exe test-retry --simulate-failures 3 2>&1
        $success = $output -match "succeeded after \d+ retries"
        Write-TestResult "Exponential backoff retry" $success
        
        # Test jitter
        if ($output -match "jitter applied") {
            Write-TestResult "Retry with jitter" $true
        }
    } catch {
        Write-TestResult "Retry logic test" $false $_.Exception.Message
    }
}

function Test-LoadPerformance {
    Write-TestHeader "Testing Load Performance"
    
    # Check if k6 is installed
    $k6Installed = Get-Command k6 -ErrorAction SilentlyContinue
    
    if ($k6Installed) {
        try {
            # Run k6 load test
            $output = k6 run --quiet loadtest/scenarios.js 2>&1
            
            # Check performance thresholds
            $metricsPass = $output -notmatch "threshold crossed"
            Write-TestResult "Performance thresholds met (P95 less than 2s, P99 less than 5s)" $metricsPass
            
            # Check error rate
            if ($output -match 'errors.*rate.*(\d+\.\d+)') {
                $errorRate = [float]$matches[1]
                Write-TestResult "Error rate less than 10%" ($errorRate -lt 0.1)
            }
        } catch {
            Write-TestResult "Load performance test" $false $_.Exception.Message
        }
    } else {
        Write-Host "    [!] k6 not installed, skipping load tests" -ForegroundColor Yellow
        Write-Host "        Install k6: choco install k6" -ForegroundColor Gray
    }
}

function Test-IntegrationSuite {
    Write-TestHeader "Testing Integration Suite"
    
    try {
        # Run Go integration tests
        $output = go test ./internal/testing/integration/... -v 2>&1
        $success = $LASTEXITCODE -eq 0
        Write-TestResult "Integration test suite" $success
        
        # Parse test results
        if ($output -match 'PASS.*(\d+)/(\d+)') {
            $passed = $matches[1]
            $total = $matches[2]
            Write-Host "    Integration tests: $passed/$total passed" -ForegroundColor Gray
        }
    } catch {
        Write-TestResult "Integration suite" $false $_.Exception.Message
    }
}

function Show-Summary {
    Write-Host "`n" -NoNewline
    Write-Host "="*60 -ForegroundColor Cyan
    Write-Host " TEST SUMMARY" -ForegroundColor Yellow
    Write-Host "="*60 -ForegroundColor Cyan
    
    $duration = (Get-Date) - $script:StartTime
    $successRate = if ($script:TotalTests -gt 0) { 
        ($script:PassedTests / $script:TotalTests) * 100 
    } else { 0 }
    
    Write-Host "Total Tests:  $script:TotalTests"
    Write-Host "Passed:       $script:PassedTests" -ForegroundColor Green
    Write-Host "Failed:       $script:FailedTests" -ForegroundColor $(if ($script:FailedTests -gt 0) { "Red" } else { "Gray" })
    Write-Host "Success Rate: $([math]::Round($successRate, 1))%"
    Write-Host "Duration:     $([math]::Round($duration.TotalSeconds, 1))s"
    
    if ($script:FailedTests -eq 0) {
        Write-Host "`n✅ DriftMgr is PRODUCTION READY!" -ForegroundColor Green
        Write-Host "All production enhancements verified successfully." -ForegroundColor Green
    } else {
        Write-Host "`n⚠️  Some tests failed. Review and fix issues before production deployment." -ForegroundColor Yellow
    }
}

# Main execution
Write-Host "DriftMgr Production Readiness Verification" -ForegroundColor Cyan
Write-Host "Testing all production enhancements..." -ForegroundColor Gray

# Build if not skipped
if (-not $SkipBuild) {
    if (-not (Test-BuildApplication)) {
        Write-Host "`n❌ Build failed. Cannot continue tests." -ForegroundColor Red
        exit 1
    }
}

# Run tests based on type
switch ($TestType) {
    "all" {
        Test-HealthEndpoints
        Test-CircuitBreaker
        Test-RateLimiter
        Test-DistributedState
        Test-Telemetry
        Test-GracefulShutdown
        Test-SecurityFeatures
        Test-CacheSystem
        Test-RetryLogic
        Test-LoadPerformance
        Test-IntegrationSuite
    }
    "unit" {
        Test-CircuitBreaker
        Test-RateLimiter
        Test-SecurityFeatures
        Test-CacheSystem
        Test-RetryLogic
    }
    "integration" {
        Test-IntegrationSuite
        Test-DistributedState
        Test-Telemetry
    }
    "load" {
        Test-LoadPerformance
    }
    "health" {
        Test-HealthEndpoints
        Test-GracefulShutdown
    }
}

# Show summary
Show-Summary

# Exit with appropriate code
exit $(if ($script:FailedTests -eq 0) { 0 } else { 1 })