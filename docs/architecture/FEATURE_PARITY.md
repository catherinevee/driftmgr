# CLI vs Web UI Feature Parity Analysis

## Feature Comparison Matrix

| Feature Category | CLI Command | Web UI | Status | Notes |
|-----------------|-------------|---------|--------|-------|
| **Discovery** |
| Discover resources | `discover --provider aws` | Discovery page | Full parity | Both support provider selection and regions |
| Auto-discovery | `discover --auto` | Missing | Partial | Web needs auto-discovery option |
| All accounts | `discover --all-accounts` | Missing | Partial | Web needs multi-account support |
| Credential check | `discover --credentials` | API endpoint | Full parity | Available via provider status |
| **Drift Detection** |
| Detect drift | `drift detect` | API endpoint | Partial | Web has API but needs UI |
| Smart defaults | `drift detect --smart-defaults` | Missing | Partial | Not exposed in Web UI |
| Environment-based | `drift detect --environment prod` | Missing | Partial | Not in Web UI |
| Drift report | `drift report` | Dashboard view | Full parity | Dashboard shows drift summary |
| Auto-remediation | `drift auto-remediate` | API endpoint | Partial | API exists, needs UI |
| **State Management** |
| State inspect | `state inspect` | API endpoint | Partial | Upload exists, needs viewer |
| State visualize | `state visualize` | API endpoint | Partial | API ready, needs UI |
| State scan | `scan --path` | Missing | Not implemented | Not in Web |
| **Resource Management** |
| List resources | N/A | Resources page | Web only | CLI uses discover |
| Delete resource | `delete --resource-id` | Missing | Not implemented | Not in Web |
| Export resources | `export --format json` | Export endpoint | Full parity | Both support JSON/CSV |
| Import resources | `import --csv file.csv` | Missing | Not implemented | Not in Web |
| **Account Management** |
| List accounts | `accounts list` | Missing | Not implemented | Not in Web |
| Switch account | `use --account` | Missing | Not implemented | Not in Web |
| **Verification** |
| Verify discovery | `verify` | Missing | Not implemented | Not in Web |
| Enhanced verify | `verify --enhanced` | Missing | Not implemented | Not in Web |
| **Server/UI** |
| Web dashboard | `serve web` | Full UI | Full parity | Web is primary interface |
| API server | `serve api` | REST API | Full parity | Same backend |
| **System** |
| Status | `status` | Dashboard | Full parity | Dashboard shows status |
| Version | `--version` | API /health | Full parity | Version in health endpoint |
| Help | `--help` | Missing | N/A | Not needed in Web |
| **Enterprise Features** |
| Audit logs | N/A | Audit page | Web only | Web has better audit view |
| RBAC | N/A | Missing | Not implemented | Needs user management UI |
| Metrics | N/A | Dashboard | Web only | Web shows metrics |

## Summary

### Features with Full Parity
1. Basic discovery operations
2. Drift detection (API level)
3. Export functionality
4. System status
5. Health/version information

### CLI-Only Features 🖥️
1. `scan --path` - Scan directories for Terraform files
2. `delete --resource-id` - Delete specific resources
3. `import --csv` - Import resources from CSV
4. `accounts` - Account management commands
5. `use` - Switch between accounts
6. `verify` - Verification commands
7. Smart defaults flags
8. Environment-specific detection

### Web-Only Features 🌐
1. Real-time WebSocket updates
2. Visual charts and graphs
3. Interactive resource browser
4. Audit log viewer
5. Progress bars for long operations
6. Notifications system

### Missing in Web UI ❌
1. **Drift Detection UI** - API exists but no UI page
2. **State File Viewer** - Upload works but no visualization
3. **Multi-Account Support** - No account switching
4. **Smart Defaults Toggle** - Not exposed in UI
5. **Environment Selection** - No env-based filtering
6. **Resource Deletion** - No delete functionality
7. **Import from CSV** - No import feature
8. **Verification Tools** - No verify functionality
9. **Directory Scanning** - No Terraform file discovery
10. **RBAC Management** - No user/role management UI

## Recommendations for Feature Parity

### High Priority
1. Add Drift Detection page with:
   - Smart defaults toggle
   - Environment selection
   - Remediation actions

2. Add State Management page with:
   - State file upload and viewer
   - Visual dependency graph
   - Resource browser

3. Add Account Management:
   - Account switcher in navbar
   - Multi-account discovery
   - Credential management

### Medium Priority
1. Resource Management enhancements:
   - Delete resource functionality
   - Bulk operations
   - Import from CSV

2. Verification tools:
   - Discovery verification
   - Resource count validation
   - Cost impact analysis

### Low Priority
1. Directory scanning for Terraform files
2. RBAC management interface
3. Advanced filtering options

## Implementation Status

- **Total CLI Features**: 25
- **Total Web Features**: 18
- **Shared Features**: 12
- **Feature Parity**: 48%

The Web UI currently has about half the features of the CLI, with significant gaps in:
- Drift detection UI
- State management visualization
- Multi-account support
- Resource operations (delete, import)
- Verification tools