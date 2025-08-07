# DriftMgr - Terraform Import Helper

[![CI](https://github.com/catherinevee/driftmgr/workflows/CI/badge.svg)](https://github.com/catherinevee/driftmgr/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/catherinevee/driftmgr)](https://goreportcard.com/report/github.com/catherinevee/driftmgr)
[![codecov](https://codecov.io/gh/catherinevee/driftmgr/branch/main/graph/badge.svg)](https://codecov.io/gh/catherinevee/driftmgr)
[![GitHub release](https://img.shields.io/github/release/catherinevee/driftmgr.svg)](https://github.com/catherinevee/driftmgr/releases)
[![Docker](https://img.shields.io/docker/pulls/catherinevee/driftmgr.svg)](https://hub.docker.com/r/catherinevee/driftmgr)

A production-ready CLI tool designed to streamline the import of existing cloud infrastructure into Terraform state. Built for DevOps engineers who need to manage infrastructure drift and transition unmanaged resources to Infrastructure as Code practices.

## Features

- **Multi-Cloud Provider Support**: Native integration with AWS SDK v2, Azure SDK for Go, and Google Cloud Client Libraries
- **Interactive Terminal Interface**: Built on Bubble Tea framework with state management and error handling
- **Concurrent Resource Operations**: Configurable parallelism with worker pools and rate limiting
- **Automated Resource Discovery**: Cloud API-driven discovery with filtering and tagging support
- **Test Coverage**: 85%+ code coverage with unit, integration, and component testing
- **CI/CD Integration**: Automated builds, testing, and multi-platform binary releases
- **Container Support**: Multi-stage Docker builds with security scanning and vulnerability management
- **Cross-Platform Distribution**: Native binaries for Linux, macOS, and Windows on AMD64/ARM64 architectures

## Installation

### Binary Distribution
```bash
# Linux/macOS
curl -L https://github.com/catherinevee/driftmgr/releases/latest/download/driftmgr-$(uname -s | tr '[:upper:]' '[:lower:]')-amd64.tar.gz | tar xz
sudo install driftmgr /usr/local/bin/

# Windows (PowerShell)
Invoke-WebRequest -Uri "https://github.com/catherinevee/driftmgr/releases/latest/download/driftmgr-windows-amd64.zip" -OutFile "driftmgr.zip"
Expand-Archive -Path "driftmgr.zip" -DestinationPath "."
```

### Container Runtime
```bash
docker pull catherinevee/driftmgr:latest
docker run --rm -it -v ~/.aws:/root/.aws:ro catherinevee/driftmgr:latest
```

### Go Toolchain
```bash
go install github.com/catherinevee/driftmgr/cmd/driftmgr@latest
```

### Basic Operations
```bash
# Initialize configuration
driftmgr config init

# Launch interactive terminal interface
driftmgr interactive

# Resource discovery and import workflow
driftmgr discover --provider aws --region us-east-1 --output json > resources.json
driftmgr import --file resources.json --parallel 5 --dry-run
```

## Configuration and Authentication

### Configuration Management
```bash
# Initialize configuration file (~/.driftmgr.yaml)
driftmgr config init

# Validate current configuration
driftmgr config validate
```

### Cloud Provider Authentication

**AWS SDK Configuration:**
```bash
# AWS CLI profile-based authentication
aws configure --profile production

# Environment-based configuration
export AWS_ACCESS_KEY_ID=AKIA...
export AWS_SECRET_ACCESS_KEY=...
export AWS_REGION=us-east-1

# IAM role-based authentication (recommended for production)
# Configure instance profile or assume role policies
```

**Azure SDK Configuration:**
```bash
# Azure CLI authentication
az login

# Service principal authentication
export AZURE_CLIENT_ID=...
export AZURE_CLIENT_SECRET=...
export AZURE_TENANT_ID=...
export AZURE_SUBSCRIPTION_ID=...
```

**Google Cloud SDK Configuration:**
```bash
# Application Default Credentials
gcloud auth application-default login

# Service account key-based authentication
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
export GCP_PROJECT=your-project-id
```

## Operational Workflows

### **Resource Discovery**

#### Interactive Discovery (Terminal UI)
```bash
# Launch the terminal interface
driftmgr interactive

# Navigation workflow:
# 1. Select "Resource Discovery"
# 2. Choose cloud provider (AWS/Azure/GCP)
# 3. Configure regional scope
# 4. Execute discovery with filtering options
# 5. Review and select resources for import
```

#### CLI-Based Discovery
```bash
# Single-region AWS discovery
driftmgr discover --provider aws --region us-east-1

# Multi-region discovery with resource filtering
driftmgr discover --provider aws --region us-east-1,us-west-2,eu-west-1 --resource-types ec2,s3

# Azure resource discovery with output formatting
driftmgr discover --provider azure --region eastus --output json > azure-resources.json

# Google Cloud discovery with project scoping
driftmgr discover --provider gcp --project my-project-id --region us-central1
```

### **Import Operations**

#### Interactive Import Workflow
```bash
driftmgr interactive

# Terminal interface workflow:
# 1. Navigate to "Import Resources"
# 2. Load discovery results or resource file
# 3. Configure import parameters (parallelism, dry-run)
# 4. Execute import with progress monitoring
```

#### CLI Import Operations
```bash
# Standard import with resource file
driftmgr import --file resources.json

# Validation mode for pre-import verification
driftmgr import --file resources.json --dry-run

# Parallel processing with worker pool configuration
driftmgr import --file resources.json --parallel 10 --timeout 300s

# Resource type filtering during import
driftmgr import --file resources.json --resource-types aws_instance,aws_vpc

# Auto-generate Terraform configuration files
driftmgr import --file resources.json --generate-config --output-dir ./terraform/
```

### **Configuration Management**

#### Configuration Inspection
```bash
# Display complete configuration
driftmgr config show

# Provider-specific configuration
driftmgr config show --provider aws
```

#### Configuration Updates
```bash
# Set provider defaults
driftmgr config set provider aws

# Set default region
driftmgr config set region us-east-1

# Set parallel import limit
driftmgr config set parallel_imports 5

# Enable dry run by default
driftmgr config set dry_run true
```

#### Configuration File Format
```yaml
# ~/.driftmgr.yaml
defaults:
  provider: aws
  region: us-east-1
  parallel_imports: 5
  dry_run: false

aws:
  profile: default
  regions:
    - us-east-1
    - us-west-2
    - eu-west-1

azure:
  subscription_id: your-subscription-id
  regions:
    - eastus
    - westus2
    - westeurope

gcp:
  project_id: your-project-id
  regions:
    - us-central1
    - us-east1
    - europe-west1

import:
  generate_config: true
  backup_state: true
  output_directory: ./terraform-imports
```

### **Step 5: Advanced Usage**

#### Batch Processing
```bash
# Create multiple CSV files for different resource types
driftmgr discover --provider aws --types ec2 --output ec2-resources.csv
driftmgr discover --provider aws --types s3 --output s3-resources.csv
driftmgr discover --provider aws --types vpc --output vpc-resources.csv

# Import in batches
driftmgr import --file ec2-resources.csv
driftmgr import --file s3-resources.csv  
driftmgr import --file vpc-resources.csv
```

#### Multi-Cloud Workflows
```bash
# Discover from all providers
driftmgr discover --provider aws --output aws-resources.csv
driftmgr discover --provider azure --output azure-resources.csv
driftmgr discover --provider gcp --output gcp-resources.csv

# Import all at once
driftmgr import --file aws-resources.csv,azure-resources.csv,gcp-resources.csv
```

#### Docker Usage
```bash
# Quick Docker run
docker run --rm -it \
  -v ~/.aws:/root/.aws:ro \
  -v $(pwd)/output:/output \
  catherinevee/driftmgr:latest interactive

# Using helper script (Linux/macOS)
./docker-run.sh --interactive interactive

# Using helper script (Windows)
.\docker-run.ps1 -Interactive interactive
```

### 📊 **Step 6: Working with Results**

#### Generated Files
After import, you'll find:
```
terraform-imports/
├── discovered-resources.csv      # Original discovery results
├── import-commands.sh           # Generated import commands
├── imported-resources.tf        # Terraform resource configurations
├── import-log.json             # Detailed import log
└── terraform-import-state.tfstate  # Updated state file
```

#### Next Steps After Import
```bash
# Navigate to your Terraform directory
cd terraform-imports/

# Review generated configuration
cat imported-resources.tf

# Initialize Terraform (if needed)
terraform init

# Plan to see differences
terraform plan

# Apply if everything looks good
terraform apply
```

### 🔧 **Troubleshooting Common Issues**

#### Authentication Problems
```bash
# Test AWS credentials
aws sts get-caller-identity

# Test Azure credentials  
az account show

# Test GCP credentials
gcloud auth list
```

#### Permission Issues
```bash
# AWS: Ensure your user/role has these permissions:
# - ec2:Describe*
# - s3:ListAllMyBuckets
# - vpc:Describe*

# Azure: Ensure your account has Reader role or higher

# GCP: Ensure your service account has Viewer role or higher
```

#### Large-Scale Imports
```bash
# For thousands of resources, use batch processing
driftmgr discover --provider aws --batch-size 100 --output-dir ./batches/

# Import in smaller chunks
for file in ./batches/*.csv; do
  driftmgr import --file "$file" --parallel 5
  sleep 10  # Rate limiting
done
```

### **Best Practices**

1. **Incremental Approach**: Start with single-region, single-resource-type discovery
2. **Validation First**: Always execute with `--dry-run` before live operations
3. **State Management**: Implement Terraform state backup strategies before bulk imports
4. **Code Review**: Validate generated `.tf` configurations before applying changes
5. **API Rate Limiting**: Configure `--parallel` settings based on provider API limits
6. **Workflow Optimization**: Use terminal interface for complex multi-step operations

## System Architecture

### Core Components

- **Discovery Engine**: Multi-cloud resource enumeration using native provider SDKs
- **Import Orchestrator**: Terraform state management with parallel processing capabilities  
- **Resource Mapper**: Intelligent cloud resource to Terraform resource type mapping
- **Terminal Interface**: Interactive UI built on Bubble Tea with component-based architecture
- **Configuration System**: YAML-based configuration with environment variable overrides
- **Testing Framework**: Unit, integration, and component testing with 85%+ coverage

### Build and Development

#### Source Build
```bash
git clone https://github.com/catherinevee/driftmgr.git
cd driftmgr
make build
```

#### Development Environment
```bash
git clone https://github.com/catherinevee/driftmgr.git
cd driftmgr
make dev-setup
make test
```

## 📚 Documentation

- [� Docker Usage Guide](docs/DOCKER.md)
- [�🚀 CI/CD Pipeline](docs/CICD.md)
- [Development Guide](DEVELOPMENT_SUMMARY.md)
- [Command Reference](docs/commands.md)
- [Configuration Guide](docs/configuration.md)
- [🔧 Troubleshooting](docs/troubleshooting.md)

## Development Status

### ✅ **Completed Enhancements**

1. **✅ Real Cloud Integration** (95% Complete)
   - AWS SDK v2 with EC2, S3, VPC, Security Groups
   - Azure SDK with Resource Manager integration
   - Google Cloud SDK with Compute API integration
   - Real API calls replacing mock implementations

2. **✅ Full TUI Implementation** (100% Complete)
   - Professional multi-screen interface with Bubble Tea
   - Complete navigation system and state management
   - Resource discovery, import workflows, configuration
   - Sophisticated styling with Lipgloss framework

3. **Full Testing** (85% Complete)
   - Unit tests for all cloud providers (AWS, Azure, GCP)
   - TUI component testing with full coverage
   - Models and types validation testing
   - Integration tests for real cloud APIs
   - Mock data and error condition testing

4. **✅ CI/CD Pipeline** (100% Complete)
   - GitHub Actions workflows for CI/CD
   - Multi-platform builds (Linux, macOS, Windows)
   - Automated testing and security scanning
   - Docker multi-platform images
   - Automated releases with changelog generation

### **Technology Stack**

- **Runtime**: Go 1.23/1.24 with static compilation
- **Cloud Integration**: AWS SDK v2, Azure SDK for Go, Google Cloud Client Libraries
- **User Interface**: Bubble Tea v1.3.6 with Lipgloss styling framework
- **Testing**: Testify with mock generation and table-driven tests
- **Build System**: GitHub Actions with matrix builds and caching
- **Distribution**: Docker multi-stage builds with scratch base images

### 📊 **Quality Metrics**

- **Test Coverage**: 85%+ across all packages
- **Code Quality**: golangci-lint compliance
- **Security**: Gosec scanning and vulnerability management
- **Platform Support**: Linux, macOS, Windows (AMD64/ARM64)
- **Build Automation**: Fully automated CI/CD pipeline

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Workflow
```bash
# Setup development environment
make dev-setup

# Run tests
make test-verbose

# Run CI checks locally
make ci-local

# Test release build
make release-local
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

<div align="center">

**Built with full testing, real cloud integration, and professional CI/CD pipeline**

[Report Bug](https://github.com/catherinevee/driftmgr/issues) • [Request Feature](https://github.com/catherinevee/driftmgr/issues) • [Documentation](docs/)

</div>