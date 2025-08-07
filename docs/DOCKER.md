# 🐳 Docker Usage Guide

This guide shows you how to run the Terraform Import Helper using Docker containers.

## 🚀 Quick Start

### Option 1: Using Pre-built Image (Recommended)
```bash
# Pull the latest image
docker pull catherinevee/driftmgr:latest

# Run with help
docker run --rm catherinevee/driftmgr:latest

# Run interactive mode
docker run --rm -it catherinevee/driftmgr:latest interactive
```

### Option 2: Build Locally
```bash
# Clone the repository
git clone https://github.com/catherinevee/driftmgr.git
cd driftmgr

# Build the Docker image
docker build -t driftmgr:local .

# Run the container
docker run --rm driftmgr:local
```

## 📋 Available Docker Images

### Production Image (`Dockerfile`)
- **Base**: `scratch` (minimal, ~10MB)
- **Purpose**: Production deployments
- **Security**: Non-root user (65534)
- **Features**: Optimized binary, CA certificates, timezone data

### Development Image (`Dockerfile.dev`)
- **Base**: `golang:1.24-alpine`
- **Purpose**: Development and testing
- **Features**: Hot reloading with Air, development tools

## 🛠️ Usage Examples

### 1. Basic Commands
```bash
# Show help
docker run --rm catherinevee/driftmgr:latest --help

# Show version
docker run --rm catherinevee/driftmgr:latest version

# Discover AWS resources
docker run --rm \
  -e AWS_ACCESS_KEY_ID=your_key \
  -e AWS_SECRET_ACCESS_KEY=your_secret \
  -e AWS_REGION=us-east-1 \
  catherinevee/driftmgr:latest discover --provider aws --region us-east-1
```

### 2. With Configuration Files
```bash
# Create a local config directory
mkdir -p ./config
echo "provider: aws" > ./config/driftmgr.yaml

# Run with mounted config
docker run --rm \
  -v $(pwd)/config:/config:ro \
  -e AWS_PROFILE=default \
  -v ~/.aws:/root/.aws:ro \
  catherinevee/driftmgr:latest discover
```

### 3. Import Resources
```bash
# Create output directory
mkdir -p ./output

# Run import with file mapping
docker run --rm \
  -v $(pwd)/resources.csv:/resources.csv:ro \
  -v $(pwd)/output:/output \
  -e AWS_ACCESS_KEY_ID=your_key \
  -e AWS_SECRET_ACCESS_KEY=your_secret \
  catherinevee/driftmgr:latest import --file /resources.csv
```

### 4. Interactive Mode
```bash
# Run in interactive mode with TTY
docker run --rm -it \
  -v $(pwd)/config:/config:ro \
  -v $(pwd)/output:/output \
  -e AWS_PROFILE=default \
  -v ~/.aws:/root/.aws:ro \
  catherinevee/driftmgr:latest interactive
```

## 🐙 Docker Compose Usage

### Development Environment
```bash
# Start development environment with hot reloading
docker-compose up driftmgr-dev

# Run tests in container
docker-compose up driftmgr-test

# Production mode
docker-compose up driftmgr
```

### Custom Docker Compose
```yaml
version: "3.8"
services:
  driftmgr:
    image: catherinevee/driftmgr:latest
    environment:
      - AWS_REGION=us-east-1
      - AZURE_LOCATION=eastus
      - GCP_REGION=us-central1
    volumes:
      - ./config:/config:ro
      - ./output:/output
      - ~/.aws:/root/.aws:ro
    command: ["interactive"]
    stdin_open: true
    tty: true
```

## 🔐 Cloud Provider Authentication

### AWS Authentication
```bash
# Option 1: Environment variables
docker run --rm \
  -e AWS_ACCESS_KEY_ID=your_key \
  -e AWS_SECRET_ACCESS_KEY=your_secret \
  -e AWS_REGION=us-east-1 \
  catherinevee/driftmgr:latest discover

# Option 2: AWS credentials file
docker run --rm \
  -v ~/.aws:/root/.aws:ro \
  -e AWS_PROFILE=default \
  catherinevee/driftmgr:latest discover

# Option 3: IAM Role (ECS/EKS)
docker run --rm \
  -e AWS_REGION=us-east-1 \
  catherinevee/driftmgr:latest discover
```

### Azure Authentication
```bash
# Option 1: Service Principal
docker run --rm \
  -e AZURE_CLIENT_ID=your_client_id \
  -e AZURE_CLIENT_SECRET=your_secret \
  -e AZURE_TENANT_ID=your_tenant \
  -e AZURE_SUBSCRIPTION_ID=your_subscription \
  catherinevee/driftmgr:latest discover --provider azure

# Option 2: Azure CLI (if available)
docker run --rm \
  -v ~/.azure:/root/.azure:ro \
  catherinevee/driftmgr:latest discover --provider azure
```

### GCP Authentication
```bash
# Option 1: Service Account Key
docker run --rm \
  -v /path/to/service-account.json:/sa.json:ro \
  -e GOOGLE_APPLICATION_CREDENTIALS=/sa.json \
  -e GCP_PROJECT=your_project \
  catherinevee/driftmgr:latest discover --provider gcp

# Option 2: Application Default Credentials
docker run --rm \
  -v ~/.config/gcloud:/root/.config/gcloud:ro \
  -e GCP_PROJECT=your_project \
  catherinevee/driftmgr:latest discover --provider gcp
```

## 📁 Volume Mounts

### Common Volume Patterns
```bash
# Configuration directory
-v $(pwd)/config:/config:ro

# Output directory for generated files
-v $(pwd)/output:/output

# Input files (CSV, JSON)
-v $(pwd)/input:/input:ro

# Cloud provider credentials
-v ~/.aws:/root/.aws:ro
-v ~/.azure:/root/.azure:ro
-v ~/.config/gcloud:/root/.config/gcloud:ro

# Terraform state (if needed)
-v $(pwd)/terraform:/terraform
```

## 🔧 Build Options

### Multi-stage Production Build
```bash
# Build optimized production image
docker build -t driftmgr:prod .

# Check image size
docker images driftmgr:prod
```

### Development Build
```bash
# Build development image with tools
docker build -f Dockerfile.dev -t driftmgr:dev .

# Run with hot reloading
docker run --rm -it \
  -v $(pwd):/app \
  -p 8080:8080 \
  driftmgr:dev
```

### Multi-platform Build
```bash
# Build for multiple architectures
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t driftmgr:multiarch .
```

## 🚀 Deployment Examples

### Kubernetes Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: driftmgr
spec:
  replicas: 1
  selector:
    matchLabels:
      app: driftmgr
  template:
    metadata:
      labels:
        app: driftmgr
    spec:
      containers:
      - name: driftmgr
        image: catherinevee/driftmgr:latest
        command: ["driftmgr", "interactive"]
        env:
        - name: AWS_REGION
          value: "us-east-1"
        volumeMounts:
        - name: config
          mountPath: /config
          readOnly: true
        - name: output
          mountPath: /output
      volumes:
      - name: config
        configMap:
          name: driftmgr-config
      - name: output
        persistentVolumeClaim:
          claimName: driftmgr-output
```

### AWS ECS Task Definition
```json
{
  "family": "driftmgr",
  "taskRoleArn": "arn:aws:iam::123456789012:role/driftmgr-task-role",
  "containerDefinitions": [
    {
      "name": "driftmgr",
      "image": "catherinevee/driftmgr:latest",
      "command": ["driftmgr", "discover", "--provider", "aws"],
      "environment": [
        {"name": "AWS_REGION", "value": "us-east-1"}
      ],
      "mountPoints": [
        {
          "sourceVolume": "output",
          "containerPath": "/output"
        }
      ]
    }
  ],
  "volumes": [
    {
      "name": "output",
      "efsVolumeConfiguration": {
        "fileSystemId": "fs-12345678"
      }
    }
  ]
}
```

## 🐛 Troubleshooting

### Common Issues

#### Permission Denied
```bash
# If you get permission errors, ensure proper user mapping
docker run --rm -it \
  --user $(id -u):$(id -g) \
  -v $(pwd)/output:/output \
  catherinevee/driftmgr:latest
```

#### Cloud Credentials Not Found
```bash
# Verify credential mounting
docker run --rm \
  -v ~/.aws:/root/.aws:ro \
  catherinevee/driftmgr:latest \
  sh -c "ls -la /root/.aws/"
```

#### Network Issues
```bash
# Test with host networking
docker run --rm --network host \
  catherinevee/driftmgr:latest discover
```

### Debug Container
```bash
# Enter container for debugging
docker run --rm -it \
  --entrypoint sh \
  catherinevee/driftmgr:latest

# Or use development image
docker run --rm -it \
  driftmgr:dev bash
```

## 📊 Performance Tips

1. **Use Volume Caches**: Mount Go module cache for faster builds
   ```bash
   -v go-mod-cache:/go/pkg/mod
   ```

2. **Multi-stage Builds**: Use the provided Dockerfile for optimal size

3. **Layer Caching**: Order COPY commands to maximize Docker layer cache

4. **Resource Limits**: Set appropriate memory/CPU limits
   ```bash
   --memory 512m --cpus 1.0
   ```

This Docker setup provides a complete containerized solution for running the Terraform Import Helper in any environment!
