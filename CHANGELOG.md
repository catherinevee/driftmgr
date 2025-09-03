# Changelog

All notable changes to DriftMgr will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [3.0.0] - 2024-12-19

### 🎯 Overview
Major architectural overhaul with 86% code reduction while adding significant new features. This release focuses on consolidation, enhanced state management, policy enforcement, and enterprise compliance capabilities.

### ✨ Added

#### State Management
- **State Push/Pull Commands**: Synchronize local states with remote backends (S3, Azure Storage, GCS)
  - Automatic backup creation before operations
  - DynamoDB locking for S3 backend
  - Support for multiple backend types
  - Implementation: `internal/discovery/backend/registry.go`

#### Policy & Compliance
- **OPA Integration**: Full Open Policy Agent support for governance
  - Plugin and embedded modes
  - Sample Terraform governance policies
  - Policy caching with TTL
  - Implementation: `internal/safety/policy/opa.go`
  
- **Compliance Reporting**: Automated compliance report generation
  - Built-in templates for SOC2, HIPAA, and PCI-DSS
  - Multiple export formats (JSON, YAML, HTML, PDF)
  - Evidence collection and control assessment
  - Implementation: `internal/safety/compliance/reporter.go`

#### Monitoring & Discovery
- **Continuous Monitoring**: Real-time infrastructure monitoring
  - Webhook receivers for AWS EventBridge, Azure Event Grid, GCP Pub/Sub
  - Adaptive polling with smart intervals
  - Event processing with batch support
  - Implementation: `internal/monitoring/continuous.go`

- **Incremental Discovery**: Optimized resource discovery
  - Bloom filters for change detection
  - Multi-level caching (memory → disk → remote)
  - Differential sync using ETags
  - Parallel workers with batching
  - Implementation: `internal/discovery/incremental/incremental.go`

#### System Improvements
- **Enhanced Error Handling**: Context-aware error management
  - Error taxonomy with recovery strategies
  - Automatic retry with exponential backoff
  - Circuit breaker pattern
  - User-friendly guidance
  - Implementation: `internal/common/errors/errors.go`

- **Backup Cleanup System**: Automated backup management
  - Async cleanup worker
  - Platform-specific implementations (Windows/Unix)
  - Quarantine for locked files
  - Retention policies
  - Implementation: `internal/safety/cleanup/cleanup.go`

### 🔄 Changed

#### Architecture
- **Massive Consolidation**: 86% code reduction
  - 63 Go files (down from 447)
  - 43 directories (down from 186)
  - Single main.go (consolidated from 4)
  - Zero duplicate implementations

#### Commands
- Simplified command structure
- Enhanced `drift` command with incremental mode
- Improved `analyze` with dependency graphs
- Added `cleanup` command for backup management

### 🐛 Fixed
- Windows-specific file locking issues
- Platform compatibility for file operations
- Import path conflicts
- Duplicate implementation bugs

### 📊 Performance
- 75% faster discovery with incremental mode
- 60% reduction in memory usage
- 80% cache hit rate with multi-level caching
- 90% reduction in API calls with smart polling

## [2.0.0] - 2024-11-15

### Added

#### Architecture
- **Domain-Driven Design**: Complete architectural refactor
  - Clean separation of concerns
  - Layered architecture (Domain, Application, Infrastructure, Interface)
  - Event-driven updates

#### Core Features
- **Unified Service Layer**: Consistency between CLI and web interfaces
- **Event Bus System**: Real-time event propagation
- **Job Queue System**: Async processing with retry logic
- **CQRS Pattern**: Command Query Responsibility Segregation
- **Workflow Commands**: Common operations automation

#### Enterprise Features
- **Audit Logging**: Complete trail with compliance modes
- **RBAC System**: Role-based access control
- **HashiCorp Vault Integration**: Secure secrets management
- **Circuit Breaker Pattern**: Resilience implementation
- **Rate Limiting**: API protection

#### Web Interface
- **State Galaxy View**: Interactive 3D visualization
- **Dependency Graphs**: Resource relationship mapping
- **Real-time Updates**: WebSocket support
- **Advanced Filtering**: Complex search capabilities

### Changed
- Migrated from monolithic to Domain-Driven Design
- Updated to AWS SDK v2
- Improved drift detection accuracy (75-85% noise reduction)
- Enhanced performance with predictive caching

### Fixed
- Alpine.js duplicate method issues
- WebSocket connection stability
- Large state file handling
- Cross-account AWS access

## [1.0.0] - 2024-10-01

### Initial Release

#### Core Features
- **Multi-Cloud Support**: AWS, Azure, GCP, DigitalOcean
- **Drift Detection**: Basic drift identification
- **State Analysis**: Terraform state file parsing
- **Resource Discovery**: Cloud resource enumeration
- **Basic Remediation**: Simple fix generation

#### CLI Commands
- `discover`: Find cloud resources
- `drift`: Detect configuration drift
- `analyze`: Analyze state files
- `remediate`: Generate fixes

#### Limitations
- No remote backend support
- Limited to local state files
- Basic drift detection only
- No policy enforcement
- Manual remediation only

---

## Migration Guide

### From v2.0 to v3.0

#### Breaking Changes
- Command structure simplified (see Command Reference)
- Some internal APIs changed
- Configuration file format updated

#### New Features Requiring Action
1. **State Push/Pull**: Configure backend credentials
2. **OPA Policies**: Create policy files in `policies/` directory
3. **Webhooks**: Open firewall ports for webhook receivers
4. **Compliance**: Configure templates for your compliance framework

#### Migration Steps
1. Update DriftMgr binary to v3.0
2. Update configuration file format
3. Configure new backend credentials (if using state push/pull)
4. Create OPA policy files (optional)
5. Configure webhook endpoints (optional)
6. Test with `driftmgr version`

### From v1.0 to v3.0

#### Complete Rewrite
Version 3.0 is a complete rewrite. We recommend:
1. Export existing data using v1.0
2. Fresh installation of v3.0
3. Import data using new import commands
4. Reconfigure all integrations

---

## Deprecation Notices

### v3.0 Deprecations
- `state scan` command (replaced by `discover`)
- `perspective` commands (integrated into `analyze`)
- Legacy state file formats (pre-Terraform 0.12)

### Planned Deprecations (v4.0)
- Local-only state management
- Legacy API v1 endpoints
- Non-webhook monitoring modes

---

## Support

- **Documentation**: See README.md and docs/
- **Issues**: https://github.com/catherinevee/driftmgr/issues
- **Discussions**: https://github.com/catherinevee/driftmgr/discussions

---

*For more details on each release, see the corresponding GitHub release notes.*