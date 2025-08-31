# DriftMgr Project Structure

## Overview
This document describes the organized directory structure of DriftMgr with the new unified service layer architecture (v2.0).

## Architecture Overview

DriftMgr v2.0 implements a **unified service layer** that ensures consistency between CLI and web interfaces:

```
┌─────────────┐     ┌─────────────┐
│   CLI       │     │   Web GUI   │
└──────┬──────┘     └──────┬──────┘
       │                    │
       ▼                    ▼
┌──────────────────────────────────┐
│        Service Layer             │
│  ┌────────────────────────────┐  │
│  │ Discovery │ State │ Drift  │  │
│  │ Service   │Service│Service │  │
│  └────────────────────────────┘  │
└──────────┬───────────────────────┘
           │
    ┌──────▼──────┐  ┌────────────┐
    │  Event Bus  │  │ Job Queue  │
    └─────────────┘  └────────────┘
           │
    ┌──────▼──────────────────┐
    │  Providers & Storage    │
    └─────────────────────────┘
```

## Directory Layout

```
driftmgr/
├── bin/                          # Binary executables (gitignored)
│   └── *.exe                     # Compiled binaries
│
├── cmd/                          # Application entry points
│   ├── driftmgr/                # Main CLI application
│   │   ├── main.go              # CLI entry point
│   │   ├── commands/            # CLI command implementations
│   │   │   ├── init.go          # Initialize configuration
│   │   │   ├── validate.go      # Validate configuration
│   │   │   ├── dashboard.go     # Dashboard command
│   │   │   ├── remediation.go   # Remediation commands
│   │   │   └── serve_web.go     # Web server command
│   │   ├── drift_report.go      # Drift reporting
│   │   ├── perspective_command.go # Perspective commands
│   │   └── state_commands.go    # State management commands
│   │
│   └── driftmgr-server/         # Server mode entry point
│       └── main.go              # Server main
│
├── configs/                      # Configuration files
│   ├── config.yaml              # Default configuration
│   └── driftmgr.yaml            # Main configuration template
│
├── data/                         # Data files (gitignored)
│   └── *.db                     # SQLite databases
│
├── docs/                         # Documentation
│   ├── architecture/            # Architecture documentation
│   │   ├── DISCOVERY_FIXES_SUMMARY.md
│   │   ├── ERROR_ANALYSIS.md
│   │   ├── FEATURES_ADDED.md
│   │   ├── RESOURCE_COUNT_VERIFICATION.md
│   │   └── SERVE_WEB_FIXES_SUMMARY.md
│   ├── api/                     # API documentation
│   ├── guides/                  # User guides
│   ├── IMPLEMENTATION_SUMMARY.md # Implementation details
│   ├── PROJECT_STRUCTURE.md    # This file
│   └── SECRETS_SETUP.md        # Secrets configuration
│
├── internal/                     # Private application code
│   ├── api/                    # API server implementation
│   │   ├── handlers/           # NEW: Refactored API handlers
│   │   │   ├── discovery_handler.go  # Discovery endpoints
│   │   │   ├── state_handler.go      # State management endpoints
│   │   │   ├── drift_handler.go      # Drift detection endpoints
│   │   │   └── remediation_handler.go # Remediation endpoints
│   │   ├── middleware/         # API middleware
│   │   ├── models/            # API models
│   │   ├── server.go          # Main server implementation
│   │   ├── websocket/         # WebSocket handlers
│   │   └── *.go               # Other API files
│   │
│   ├── audit/                  # Audit logging
│   ├── cache/                  # Caching implementation
│   ├── config/                 # Configuration management
│   │   └── manager.go         # Config hot-reload manager
│   │
│   ├── core/                   # Core business logic
│   │   ├── color/             # Terminal colors
│   │   ├── discovery/         # Resource discovery
│   │   ├── drift/             # Drift detection
│   │   ├── models/            # Core models
│   │   └── progress/          # Progress tracking
│   │
│   ├── credentials/            # Credential management
│   ├── database/              # Database layer
│   │   └── db.go              # SQLite implementation
│   │
│   ├── deletion/              # Resource deletion
│   ├── infrastructure/        # Infrastructure utilities
│   ├── integration/           # External integrations
│   ├── notifications/         # Notification system
│   │   └── notifier.go       # Multi-channel notifications
│   │
│   ├── services/              # NEW: Unified service layer
│   │   ├── discovery_service.go  # Discovery operations
│   │   ├── state_service.go      # State management
│   │   ├── drift_service.go      # Drift detection
│   │   ├── remediation_service.go # Remediation execution
│   │   └── manager.go            # Service coordinator
│   │
│   ├── events/                # NEW: Event bus system
│   │   └── event_bus.go      # Real-time event propagation
│   │
│   ├── jobs/                  # NEW: Job queue system
│   │   └── queue.go          # Async job processing
│   │
│   ├── cqrs/                  # NEW: CQRS implementation
│   │   ├── commands.go       # Command definitions
│   │   └── queries.go        # Query definitions
│   │
│   ├── providers/             # Cloud provider implementations
│   │   ├── aws/              # AWS provider
│   │   ├── azure/            # Azure provider
│   │   ├── digitalocean/     # DigitalOcean provider
│   │   ├── gcp/              # GCP provider
│   │   └── cloud/            # Common provider interfaces
│   │
│   ├── relationships/         # Resource relationships
│   │   └── mapper.go         # Dependency mapping
│   │
│   ├── remediation/           # Remediation logic
│   ├── resilience/            # Circuit breakers, retries
│   ├── search/                # Search functionality
│   ├── terraform/             # Terraform integration
│   │   ├── remediation/      # Terraform remediation
│   │   └── state/            # State file handling
│   │
│   ├── timeline/              # Timeline tracking
│   ├── utils/                 # Utility functions
│   └── validation/            # Input validation
│
├── logs/                        # Log files (gitignored)
│
├── scripts/                     # Utility scripts
│   ├── build/                  # Build scripts
│   ├── deploy/                 # Deployment scripts
│   ├── installer/              # Installation scripts
│   ├── remediation/            # Remediation scripts
│   ├── test/                   # Test scripts
│   │   ├── test_*.ps1         # PowerShell test scripts
│   │   ├── test_*.sh          # Shell test scripts
│   │   └── verify_*.ps1       # Verification scripts
│   ├── testing/                # Testing utilities
│   ├── tools/                  # Development tools
│   ├── utils/                  # Utility scripts
│   │   ├── check_*.ps1        # Check scripts
│   │   └── apply-enhancements.ps1
│   └── verify/                 # Verification scripts
│
├── tests/                       # Test files
│   ├── integration/            # Integration tests
│   │   └── test_quick_server.go
│   ├── samples/                # Sample test data
│   │   ├── test_*.html        # HTML test files
│   │   └── test_upload.json   # Test data
│   └── README.md              # Testing documentation
│
├── web/                         # Web frontend
│   ├── index.html              # Main application (latest version)
│   ├── login.html              # Login page
│   ├── archive/                # Archived versions (gitignored)
│   │   ├── index-*.html       # Old index versions
│   │   └── *.html             # Other archived files
│   ├── js/                     # JavaScript files
│   │   └── app.js             # Main application logic
│   └── css/                    # Stylesheets
│
├── .github/                     # GitHub configuration
│   └── workflows/              # CI/CD workflows
│
├── .gitignore                   # Git ignore patterns
├── .golangci.yml               # Go linting configuration
├── CLAUDE.md                   # AI assistant context
├── docker-compose.yml          # Docker composition
├── Dockerfile                  # Container definition
├── go.mod                      # Go module definition
├── go.sum                      # Go module checksums
├── LICENSE                     # License file
├── Makefile                    # Build automation
└── README.md                   # Main documentation
```

## Key Changes from Cleanup

### Files Moved
- **Test files**: `test_*.html`, `test_*.ps1` → `tests/samples/` and `scripts/test/`
- **Documentation**: Scattered `.md` files → `docs/architecture/`
- **Binaries**: `*.exe` files → `bin/`
- **Logs**: `*.log` files → `logs/`
- **Database**: `*.db` files → `data/`
- **Scripts**: Root level scripts → `scripts/utils/`

### Files Archived
- **Web files**: Old index versions → `web/archive/`
- **Test data**: Sample files → `tests/samples/`

### Files Organized
- **API handlers**: Loose handler files → `internal/api/handlers/` subdirectories
- **Scripts**: Categorized into `build/`, `test/`, `utils/` subdirectories

### Files Removed
- `nul` file (accidental creation)
- Duplicate test scripts
- Redundant credential check scripts

## Benefits of New Structure

1. **Clear Separation**: Source code, tests, scripts, and documentation are clearly separated
2. **Better Organization**: Related files are grouped together
3. **Cleaner Root**: Root directory only contains essential files
4. **Improved Navigation**: Logical structure makes finding files easier
5. **CI/CD Ready**: Test and build scripts are properly organized
6. **Gitignore Compliance**: Generated files are properly excluded

## Development Workflow

### Building
```bash
go build -o bin/driftmgr ./cmd/driftmgr
```

### Testing
```bash
# Run Go tests
go test ./...

# Run test scripts
./scripts/test/test_simple.ps1
```

### Running
```bash
# From binary
./bin/driftmgr serve web

# During development
go run ./cmd/driftmgr serve web
```

## Maintenance

### Adding New Features
- Source code → `internal/`
- Tests → `tests/`
- Documentation → `docs/`
- Scripts → `scripts/`

### Generated Files
All generated files (binaries, logs, databases) are gitignored and stored in:
- `bin/` - Executables
- `logs/` - Log files
- `data/` - Databases

This structure maintains all recent feature integrations while providing a clean, professional organization suitable for enterprise deployment.