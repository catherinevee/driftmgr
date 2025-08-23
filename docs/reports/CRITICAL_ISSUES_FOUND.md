# Critical Issues Found in DriftMgr

## 1. 🔴 CRITICAL: Limited Resource Type Support
**Location:** `internal/discovery/universal_discovery.go:454-467`

### Issue:
DriftMgr only supports a tiny fraction of available resource types for each provider:
- **AWS**: Only 10 types (ec2, rds, lambda, s3, iam, vpc, elb, cloudformation, cloudtrail, kms)
- **Azure**: Only 6 types (virtual_machine, storage_account, resource_group, managed_identity, network_watcher, key_vault)
- **GCP**: Only 5 types (compute_instance, storage_bucket, container_cluster, sql_instance, vpc_network)
- **DigitalOcean**: Only 7 types (droplet, load_balancer, database, kubernetes_cluster, volume, vpc, firewall)

### Impact:
**Missing 80-90% of resource types!** Examples of missing critical resources:
- AWS: Missing EKS clusters, ECS services, DynamoDB tables, SQS queues, SNS topics, Route53, CloudFront, API Gateway, etc.
- Azure: Missing AKS clusters, App Services, Functions, Cosmos DB, Service Bus, Event Hubs, etc.
- GCP: Missing GKE clusters, Cloud Functions, Pub/Sub, BigQuery, Dataflow, etc.

### Fix Required:
```go
// Should discover ALL resource types, not just hardcoded ones
func (gd *GenericDiscoverer) GetSupportedResourceTypes() []string {
    // Should dynamically discover all available types
    // OR at minimum include 100+ resource types per provider
}
```

## 2. 🔴 CRITICAL: Hardcoded TODO/Unimplemented Functions
**Location:** Multiple files

### Issues Found:
- `context.TODO()` used instead of proper context: 20+ instances
- Unimplemented Cloud Debugger: `cmd/server/main.go:3186`
- Missing Terraform state parsing: `cmd/server/main.go:3308 // TODO: Implement proper Terraform state file parsing`

### Impact:
- No proper context cancellation/timeout handling
- Features advertised but not implemented
- State file comparison incomplete

## 3. 🟡 HIGH: No Multi-Account Role Assumption for AWS
**Location:** `internal/discovery/multi_account_discovery.go:520`

### Issue:
```go
// For organization accounts, we might need to assume a role
// BUT IT'S NOT IMPLEMENTED!
```

### Impact:
Cannot discover resources in AWS Organizations with cross-account roles, limiting enterprise usage.

## 4. 🟡 HIGH: Extremely Limited API Pagination
**Location:** `internal/backup/backup_manager.go:279`

### Issue:
```go
Limit: 100  // Hardcoded limit
```

### Impact:
- Will miss resources if there are more than 100
- No pagination handling for large environments
- AWS/Azure/GCP all support thousands of resources per account

## 5. 🟡 HIGH: Fatal Errors Instead of Graceful Handling
**Location:** Multiple files

### Issues:
- `log.Fatal()` used in 20+ places
- Server crashes on errors instead of returning error responses
- No retry logic for transient failures

### Examples:
```go
log.Fatalf("Failed to initialize authentication manager: %v", err)
log.Fatal(http.ListenAndServe(":"+port, mux))
```

## 6. 🟡 HIGH: Insufficient Timeouts
**Location:** Various

### Issues:
- Read/Write timeout: Only 15 seconds (`cmd/driftmgr-server/main.go:380`)
- Discovery timeout: 30 minutes for ALL resources
- No per-region or per-resource-type timeouts

### Impact:
- Timeouts in large environments
- Cannot handle slow API responses
- All-or-nothing discovery (no partial results)

## 7. 🟡 HIGH: Rate Limiting Too Aggressive
**Location:** `internal/web/server.go:759`

### Issue:
```go
RateLimitPerMinute: 100  // Only 100 requests per minute!
```

### Impact:
- Discovery will be throttled in large environments
- 100 req/min = 1.67 req/sec (AWS allows 10-20 req/sec per service)
- Will take hours to discover large infrastructures

## 8. 🟡 HIGH: No Resource Caching Strategy
**Location:** Throughout codebase

### Issue:
- No caching of discovered resources
- Every discovery call hits the API
- No incremental discovery

### Impact:
- Slow performance
- High API costs
- Unnecessary API calls

## 9. 🟠 MEDIUM: Debug Logging in Production
**Location:** `cmd/server/main.go:1230-1233`

### Issue:
```go
fmt.Printf("DEBUG: Discovered %d real resources...")
```

### Impact:
- Exposes sensitive information
- Performance overhead
- Not using proper logging levels

## 10. 🟠 MEDIUM: Missing Resource Types in Type Conversion
**Location:** `internal/discovery/universal_discovery.go:487-504`

### Issue:
Only 6 Azure types mapped, 5 GCP types mapped. Everything else becomes "azure_unknown" or "gcp_unknown"

### Impact:
- Resources not properly categorized
- Drift detection won't work for unmapped types
- Reporting will be inaccurate

## Summary of Critical Issues

| Priority | Issue | Impact | Effort to Fix |
|----------|-------|--------|--------------|
| 🔴 CRITICAL | Limited resource types | Missing 80-90% of resources | High |
| 🔴 CRITICAL | Unimplemented features | Broken functionality | Medium |
| 🟡 HIGH | No AWS role assumption | Can't scan enterprises | Medium |
| 🟡 HIGH | No pagination | Misses resources >100 | Medium |
| 🟡 HIGH | Fatal error handling | Server crashes | Low |
| 🟡 HIGH | Insufficient timeouts | Fails in large envs | Low |
| 🟡 HIGH | Aggressive rate limiting | Very slow discovery | Low |
| 🟡 HIGH | No caching | Poor performance | High |
| 🟠 MEDIUM | Debug logging | Security risk | Low |
| 🟠 MEDIUM | Missing type mappings | Incorrect categorization | Medium |

## Recommended Immediate Fixes

1. **Expand resource type support** - Add ALL AWS/Azure/GCP/DO resource types
2. **Implement pagination** - Handle large result sets properly
3. **Fix error handling** - Return errors instead of crashing
4. **Adjust timeouts and rate limits** - Make configurable and reasonable
5. **Implement caching** - Cache discoveries for performance
6. **Complete TODOs** - Implement missing functionality
7. **Add retry logic** - Handle transient failures gracefully

These issues significantly impact DriftMgr's ability to function in production environments and must be addressed for enterprise readiness.