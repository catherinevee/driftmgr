# How to Run DriftMgr 🚀

## Overview

DriftMgr is a multi-component application with several ways to run it depending on your needs. The main entry point automatically manages the server and client components.

## Quick Start

### 1. **Main Executable (Recommended)**

The easiest way to run DriftMgr is using the main executable:

```bash
# From the project root directory
./bin/driftmgr.exe

# Or if you're on Linux/macOS
./bin/driftmgr
```

**What this does:**
- ✅ Automatically starts the server if it's not running
- ✅ Launches the interactive CLI client
- ✅ Manages all components seamlessly
- ✅ Handles server/client communication

### 2. **Direct Client Execution**

If you want to run just the client (server must be running):

```bash
# Run the client directly
./bin/driftmgr-client.exe

# Or on Linux/macOS
./bin/driftmgr-client
```

### 3. **Server-Only Mode**

To run just the server component:

```bash
# Run the server
./bin/driftmgr-server.exe

# Or on Linux/macOS
./bin/driftmgr-server
```

## Available Executables

### 📁 **bin/ Directory Contents**

| Executable | Purpose | Size | Description |
|------------|---------|------|-------------|
| `driftmgr.exe` | **Main Entry Point** | 8.9MB | **Recommended** - Auto-manages server & client |
| `driftmgr-client.exe` | CLI Client | 12.7MB | Interactive command-line interface |
| `driftmgr-server.exe` | API Server | 26.8MB | Backend API server |
| `driftmgr` | Main (Linux/macOS) | 26.9MB | Main executable for Unix systems |
| `driftmgr-server` | Server (Linux/macOS) | 28.7MB | Server for Unix systems |
| `examples.exe` | Examples | 65.8MB | Example applications and demos |
| `delete_all_resources.exe` | Resource Deletion | 65.9MB | Bulk resource deletion tool |
| `integration.exe` | Integration Tests | 5.0MB | Integration testing tool |
| `main.exe` | Alternative Main | 8.5MB | Alternative main executable |

## Running Modes

### 🎯 **Interactive Shell Mode (Recommended)**

```bash
# Start interactive shell
./bin/driftmgr.exe

# You'll see:
# Welcome to DriftMgr - Terraform Drift Detection & Remediation Tool
# driftmgr>
# 
# Type '?' to see all available commands
# Type 'help' for detailed help
# Type 'exit' to quit
```

**Interactive Features:**
- ✅ **Tab Completion** - Auto-complete commands and arguments
- ✅ **Command History** - Navigate with arrow keys
- ✅ **Context-Sensitive Help** - Type `?` for help on any command
- ✅ **Auto-Suggestions** - Smart command suggestions
- ✅ **Fuzzy Search** - Find commands with partial input

### 🔧 **Direct Command Mode**

```bash
# Run commands directly without entering interactive mode
./bin/driftmgr.exe discover aws all
./bin/driftmgr.exe perspective terraform aws
./bin/driftmgr.exe analyze terraform
./bin/driftmgr.exe enhanced-analyze terraform
./bin/driftmgr.exe remediate example --generate
```

### 🌐 **Server-Client Architecture**

```bash
# Start server in background
./bin/driftmgr-server.exe &

# Then run client
./bin/driftmgr-client.exe
```

## Common Commands

### 🔍 **Discovery Commands**

```bash
# Discover AWS resources across all regions
driftmgr> discover aws all

# Discover specific AWS regions
driftmgr> discover aws us-east-1 us-west-2

# Discover Azure resources
driftmgr> discover azure westeurope northeurope

# Discover GCP resources
driftmgr> discover gcp us-central1 europe-west1
```

### 📊 **Analysis Commands**

```bash
# Analyze Terraform state
driftmgr> analyze terraform

# Enhanced analysis with risk reasoning
driftmgr> enhanced-analyze terraform

# Compare state with live infrastructure
driftmgr> perspective terraform aws
```

### 🛠️ **Remediation Commands**

```bash
# Generate remediation plan
driftmgr> remediate example --generate

# Execute remediation
driftmgr> remediate example --execute

# Interactive remediation
driftmgr> remediate example --interactive
```

## Building from Source

### 🔨 **Build All Components**

```bash
# Build everything
make build

# Or build specific components
make build-server    # Build server only
make build-client    # Build client only
```

### 🧪 **Development Mode**

```bash
# Run server in development mode
make dev-server

# Run client in development mode
make dev-client
```

## Installation Options

### 📦 **System Installation**

```bash
# Windows (requires Administrator)
./install.ps1

# Linux/macOS
./install.sh

# This adds driftmgr to your PATH
# Now you can run: driftmgr
```

### 🐳 **Docker Installation**

```bash
# Build Docker image
make docker-build

# Run Docker container
make docker-run
```

## Configuration

### ⚙️ **Environment Setup**

```bash
# Set AWS credentials
export AWS_ACCESS_KEY_ID=your_access_key
export AWS_SECRET_ACCESS_KEY=your_secret_key
export AWS_DEFAULT_REGION=us-east-1

# Set Azure credentials
export AZURE_CLIENT_ID=your_client_id
export AZURE_CLIENT_SECRET=your_client_secret
export AZURE_TENANT_ID=your_tenant_id

# Set GCP credentials
export GOOGLE_APPLICATION_CREDENTIALS=path/to/service-account.json
```

### 📁 **Region Configuration Fix**

If you see region file warnings, copy the region files to the root directory:

```bash
# Windows
copy config\regions\aws_regions.json aws_regions.json
copy config\regions\azure_regions.json azure_regions.json
copy config\regions\gcp_regions.json gcp_regions.json
copy config\regions\all_regions.json all_regions.json

# Linux/macOS
cp config/regions/aws_regions.json aws_regions.json
cp config/regions/azure_regions.json azure_regions.json
cp config/regions/gcp_regions.json gcp_regions.json
cp config/regions/all_regions.json all_regions.json
```

### 🕐 **Timeout Configuration**

```bash
# Configure timeouts for different operations
./scripts/set-timeout.ps1 --discovery 300
./scripts/set-timeout.ps1 --analysis 120
./scripts/set-timeout.ps1 --remediation 600
```

## Troubleshooting

### 🔧 **Common Issues**

1. **Server not starting:**
   ```bash
   # Check if port 8080 is available
   netstat -an | findstr :8080
   
   # Kill existing process if needed
   taskkill /F /IM driftmgr-server.exe
   ```

2. **Permission denied:**
   ```bash
   # Make executable on Linux/macOS
   chmod +x bin/driftmgr
   chmod +x bin/driftmgr-client
   chmod +x bin/driftmgr-server
   ```

3. **Missing dependencies:**
   ```bash
   # Install Go dependencies
   go mod download
   
   # Build from source
   make build
   ```

### 📋 **System Requirements**

- **Go**: 1.19 or higher
- **Memory**: 512MB RAM minimum
- **Disk**: 100MB free space
- **Network**: Internet access for cloud provider APIs
- **OS**: Windows, Linux, or macOS

## Examples

### 🎯 **Quick Examples**

```bash
# 1. Start DriftMgr
./bin/driftmgr.exe

# 2. Discover AWS resources
driftmgr> discover aws all

# 3. Analyze Terraform state
driftmgr> analyze terraform

# 4. Generate remediation plan
driftmgr> remediate example --generate

# 5. Exit
driftmgr> exit
```

### 🔄 **Workflow Example**

```bash
# Complete workflow
./bin/driftmgr.exe

driftmgr> discover aws us-east-1 us-west-2
driftmgr> perspective terraform aws
driftmgr> enhanced-analyze terraform
driftmgr> remediate drift-resources --generate
driftmgr> remediate drift-resources --execute
driftmgr> exit
```

## Support

- 📖 **Documentation**: See `docs/` directory
- 🐛 **Issues**: Check GitHub issues
- 💬 **Help**: Use `help` command in interactive mode
- ❓ **Context Help**: Type `?` after any command

---

**Happy Drift Detection! 🎉**
