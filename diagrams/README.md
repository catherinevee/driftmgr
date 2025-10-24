# DriftMgr Architecture Diagrams

This directory contains Go-based architecture diagrams for DriftMgr using the [go-diagrams](https://github.com/blushft/go-diagrams) library.

## Prerequisites

1. **Go 1.23+** - Required to run the diagram generation scripts
2. **Graphviz** - Required for rendering diagrams
   ```bash
   # Ubuntu/Debian
   sudo apt-get install graphviz
   
   # macOS
   brew install graphviz
   
   # Windows (using chocolatey)
   choco install graphviz
   ```

## Available Diagrams

### 1. Production Architecture (`architecture_diagram.go`)
- **Output**: `driftmgr_production_architecture.png`
- **Description**: Complete production system architecture with real-time capabilities
- **Components**:
  - User Interface Layer (Web Dashboard, REST API, WebSocket API)
  - Authentication & Security (JWT, RBAC, API Keys)
  - Core Business Logic (Drift Detection, State Manager, Remediation, Discovery, WebSocket Service)
  - Analytics & Intelligence (ML, Automation, Alerting, Monitoring)
  - Data Layer (PostgreSQL, Redis)
  - Cloud Providers (AWS, Azure, GCP, DigitalOcean)
  - Terraform Backend Storage (S3, Azure Blob, GCS, Local)

### 2. Real-time Architecture (`realtime_architecture.go`)
- **Output**: `driftmgr_realtime_architecture.png`
- **Description**: Real-time communication architecture with WebSocket flows
- **Components**:
  - Client Layer (Web Dashboard, API Clients)
  - WebSocket Layer (Hub, Connection Management, Broadcasting)
  - Real-time Services (Drift Detection, Alerting, Monitoring, Authentication)
  - Message Types (Drift Events, Alerts, Status Updates, Heartbeats)
  - Data Sources (Cloud APIs, State Files, Database)

### 3. API Architecture (`api_architecture.go`)
- **Output**: `driftmgr_api_architecture.png`
- **Description**: Complete API architecture showing all 25+ endpoints
- **Endpoint Groups**:
  - Authentication Endpoints (Login, Register, Refresh, Profile, API Keys)
  - Backend Management (List, Discover, Details, Update, Test)
  - State Management (List, Details, Import, Remove, Move, Lock)
  - Resource Management (List, Details, Search, Tags, Cost, Compliance)
  - Drift Detection (Detect, Results, Details, Delete, History, Summary)
  - WebSocket Endpoints (Connection, API WebSocket, Stats)

### 4. Drift Detection Flow (`drift_detection_flow.go`)
- **Output**: `drift_detection_flow.png`
- **Description**: Step-by-step workflow of the drift detection process
- **Flow**:
  1. Backend Discovery
  2. State File Retrieval
  3. State Parsing & Validation
  4. Parallel Cloud Resource Discovery
  5. Resource Comparison
  6. Drift Classification
  7. Severity Scoring
  8. Report Generation

### 5. Remediation Workflow (`remediation_workflow.go`)
- **Output**: `remediation_workflow.png`
- **Description**: Remediation process from drift detection to resolution
- **Strategies**:
  - Code-as-Truth (Apply Terraform)
  - Cloud-as-Truth (Update Code)
  - Manual Review (Generate Plan)
- **Safety**: Backup creation, approval workflow, rollback capability

## Generating Diagrams

### Method 1: Using Scripts (Recommended)

**Linux/macOS:**
```bash
chmod +x generate.sh
./generate.sh
```

**Windows:**
```powershell
.\generate.ps1
```

### Method 2: Manual Generation

```bash
# Initialize Go modules
go mod tidy

# Generate individual diagrams
go run architecture_diagram.go
go run realtime_architecture.go
go run api_architecture.go
go run drift_detection_flow.go
go run remediation_workflow.go
```

## Output

All generated diagrams will be placed in the `output/` directory:
- `driftmgr_production_architecture.png` - Production system architecture
- `driftmgr_realtime_architecture.png` - Real-time communication architecture
- `driftmgr_api_architecture.png` - Complete API architecture (25+ endpoints)
- `drift_detection_flow.png` - Drift detection workflow
- `remediation_workflow.png` - Remediation process flow

Additional formats may be generated (SVG, DOT) depending on the go-diagrams configuration.

## Customization

To modify the diagrams:

1. Edit the respective `.go` files
2. Adjust node types, labels, and connections as needed
3. Available node types can be found in the [go-diagrams documentation](https://github.com/blushft/go-diagrams)
4. Re-run the generation scripts

## Integration with Documentation

These diagrams are designed to be included in:
- README.md (architecture overview)
- Documentation websites
- Presentations and proposals
- Technical specifications

## Troubleshooting

### "Graphviz not found" Error
Install Graphviz using the package manager for your operating system (see Prerequisites).

### "Module not found" Error
Run `go mod tidy` in the diagrams directory to download dependencies.

### Empty or Corrupted Output
Ensure you have the latest version of go-diagrams and that Graphviz is properly installed and in your PATH.

## Dependencies

- `github.com/blushft/go-diagrams` - Main diagram generation library
- `github.com/awalterschulze/gographviz` - Graphviz integration
- `github.com/emicklei/dot` - DOT format support
- `github.com/lucasb-eyer/go-colorful` - Color handling

## Contributing

When adding new diagrams:
1. Create a new `.go` file following the existing naming convention
2. Update this README with the new diagram description
3. Add the new diagram to the generation scripts
4. Test generation on multiple platforms

---

*Generated diagrams help visualize DriftMgr's architecture and workflows for better understanding and documentation.*