# DriftMgr CLI Documentation

This directory contains detailed documentation for the DriftMgr CLI interface.

## Documentation Index

### User Guides
- **[Features Guide](enhanced-features-guide.md)** - Complete guide to the CLI features including tab completion, auto-suggestions, and fuzzy search

### Quick Reference
- **[Main README](../../README.md)** - Project overview and quick start guide
- **[Installation Guide](../../README.md#installation)** - Installation instructions

## CLI Features Overview

### Core Features
- **Interactive Shell** - User-friendly command-line interface
- **Context-Sensitive Help** - Type `?` or `command ?` for help
- **Command History** - Up to 100 commands with navigation
- **Security Hardened** - Input validation and injection prevention

### Advanced Features
- **Tab Completion** - Auto-complete commands and arguments
- **Auto-Suggestions** - Smart suggestions based on history and context
- **Fuzzy Search** - Find commands with partial input
- **Arrow Key Navigation** - Navigate history and move cursor
- **Context-Aware Completion** - Dynamic completion based on discovered resources

## Getting Started

### Installation
```bash
# Build the enhanced CLI
go build -o driftmgr-client.exe cmd/driftmgr-client/main.go cmd/driftmgr-client/completion.go cmd/driftmgr-client/enhanced_analyze.go cmd/driftmgr-client/remediate.go cmd/driftmgr-client/credentials.go
```

### Basic Usage
```bash
# Start interactive shell
./driftmgr-client.exe

# Run commands directly
./driftmgr-client.exe discover aws us-east-1
./driftmgr-client.exe analyze terraform
./driftmgr-client.exe help
```

### Features Demo
```bash
# Run the demo script
./scripts/demo-enhanced-cli.sh
```

## Command Reference

### Core Commands
- `discover <provider> [regions...]` - Discover cloud resources with progress tracking
- `analyze <statefile_id>` - Analyze drift for a state file with configurable sensitivity
- `perspective <statefile_id> [provider]` - Compare state with live infrastructure
- `visualize <statefile_id> [path]` - Generate infrastructure visualization
- `diagram <statefile_id>` - Generate infrastructure diagram
- `export <statefile_id> <format>` - Export diagram in specified format
- `statefiles` - List available state files



### Remediation Commands
- `remediate <drift_id> [options]` - Remediate drift with automated commands
- `remediate-batch <statefile_id> [options]` - Batch remediation for multiple drifts
- `remediate-history` - Show remediation history
- `remediate-rollback <snapshot_id>` - Rollback to previous state

### Utility Commands
- `credentials <command>` - Manage cloud provider credentials
- `health` - Check service health
- `notify <type> <subject> <message>` - Send notifications
- `terragrunt <subcommand>` - Manage Terragrunt configurations

### Shell Commands
- `help` - Show help message
- `history` - Show command history
- `clear` - Clear the screen
- `exit`, `quit` - Exit the shell

## Advanced Usage

### Tab Completion Examples
```bash
driftmgr> disc<TAB>          # Completes to "discover"
driftmgr> discover a<TAB>    # Completes to "discover aws"
driftmgr> discover aws us<TAB> # Shows available US regions
```

### Auto-Suggestion Examples
```bash
driftmgr> d                  # Shows suggestions starting with 'd'
driftmgr> disc               # Shows "discover" and "enhanced-discover"
```

### Arrow Key Navigation
- **Up Arrow**: Navigate to previous command in history
- **Down Arrow**: Navigate to next command in history
- **Left/Right Arrows**: Move cursor within current line

## Troubleshooting

### Common Issues
1. **Tab completion not working**: Ensure you're using the enhanced CLI version
2. **Arrow keys not working**: May be terminal-specific; try different terminal emulator
3. **Auto-suggestions not showing**: Check command history and input length (1-10 characters)

### Performance Notes
- Tab completion is instant for small datasets
- Auto-suggestions are limited to 5 results for performance
- History is limited to 100 commands
- Completion data is updated dynamically as you use the CLI

## Development

### Building from Source
```bash
# Build all CLI components
go build -o driftmgr-client.exe cmd/driftmgr-client/*.go

# Build specific components
go build -o driftmgr-client.exe cmd/driftmgr-client/main.go cmd/driftmgr-client/completion.go
```

### Key Files
- `cmd/driftmgr-client/main.go` - Main CLI application
- `cmd/driftmgr-client/completion.go` - Advanced input handling
- `cmd/driftmgr-client/enhanced_analyze.go` - Analysis features
- `cmd/driftmgr-client/remediate.go` - Remediation features
- `cmd/driftmgr-client/credentials.go` - Credential management

## Contributing

When adding new CLI features:
1. Update the completion data in `completion.go`
2. Add help text in `main.go`
3. Update this documentation
4. Test with the demo script

## Support

For issues and questions:
1. Check the troubleshooting section
2. Review the enhanced features guide
3. Test with the demo script
4. Check the main project README
