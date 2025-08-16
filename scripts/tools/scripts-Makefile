# Makefile for Custom Blast Radius

.PHONY: help install build run docker-build docker-run docker-stop clean test examples

# Default target
help:
	@echo "Custom Blast Radius - Available Commands:"
	@echo ""
	@echo "Installation:"
	@echo "  install      - Install Python dependencies"
	@echo "  build        - Build Docker image"
	@echo ""
	@echo "Running:"
	@echo "  run          - Run with AWS VPC example"
	@echo "  run-multi    - Run with multi-tier app example"
	@echo "  run-k8s      - Run with Kubernetes example"
	@echo "  run-serverless - Run with serverless example"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run all examples with Docker Compose"
	@echo "  docker-stop  - Stop all Docker containers"
	@echo ""
	@echo "Export:"
	@echo "  export-vpc   - Export AWS VPC diagram"
	@echo "  export-multi - Export multi-tier app diagram"
	@echo "  export-all   - Export all examples"
	@echo ""
	@echo "Utilities:"
	@echo "  clean        - Clean generated files"
	@echo "  test         - Run tests"
	@echo "  examples     - Initialize Terraform examples"

# Install dependencies
install:
	@echo "📦 Installing Python dependencies..."
	pip install -r requirements.txt
	@echo "✅ Dependencies installed"

# Build Docker image
build:
	@echo "🐳 Building Docker image..."
	docker build -t custom-blast-radius .
	@echo "✅ Docker image built"

# Run with AWS VPC example
run:
	@echo "🚀 Starting Blast Radius with AWS VPC example..."
	python blast_radius.py --serve examples/aws-vpc

# Run with multi-tier app example
run-multi:
	@echo "🚀 Starting Blast Radius with multi-tier app example..."
	python blast_radius.py --serve examples/multi-tier-app

# Run with Kubernetes example
run-k8s:
	@echo "🚀 Starting Blast Radius with Kubernetes example..."
	python blast_radius.py --serve examples/kubernetes

# Run with serverless example
run-serverless:
	@echo "🚀 Starting Blast Radius with serverless example..."
	python blast_radius.py --serve examples/serverless

# Docker commands
docker-build:
	@echo "🐳 Building Docker image..."
	docker build -t custom-blast-radius .
	@echo "✅ Docker image built"

docker-run:
	@echo "🐳 Starting all examples with Docker Compose..."
	docker-compose up -d
	@echo "✅ All services started:"
	@echo "  - AWS VPC: http://localhost:5000"
	@echo "  - Multi-tier App: http://localhost:5001"
	@echo "  - Kubernetes: http://localhost:5002"
	@echo "  - Serverless: http://localhost:5003"

docker-stop:
	@echo "🛑 Stopping Docker containers..."
	docker-compose down
	@echo "✅ Docker containers stopped"

# Export commands
export-vpc:
	@echo "📊 Exporting AWS VPC diagram..."
	python blast_radius.py --export examples/aws-vpc --format all --output output/vpc
	@echo "✅ VPC diagram exported to output/vpc/"

export-multi:
	@echo "📊 Exporting multi-tier app diagram..."
	python blast_radius.py --export examples/multi-tier-app --format all --output output/multi-tier
	@echo "✅ Multi-tier app diagram exported to output/multi-tier/"

export-all: export-vpc export-multi
	@echo "📊 Exporting all examples..."
	@if [ -d "examples/kubernetes" ]; then \
		python blast_radius.py --export examples/kubernetes --format all --output output/kubernetes; \
		echo "✅ Kubernetes diagram exported to output/kubernetes/"; \
	fi
	@if [ -d "examples/serverless" ]; then \
		python blast_radius.py --export examples/serverless --format all --output output/serverless; \
		echo "✅ Serverless diagram exported to output/serverless/"; \
	fi
	@echo "✅ All diagrams exported"

# Clean generated files
clean:
	@echo "🧹 Cleaning generated files..."
	rm -rf output/
	rm -rf __pycache__/
	rm -rf .pytest_cache/
	@echo "✅ Cleaned"

# Run tests
test:
	@echo "🧪 Running tests..."
	python -m pytest tests/ -v
	@echo "✅ Tests completed"

# Initialize Terraform examples
examples:
	@echo "📝 Initializing Terraform examples..."
	@for example in examples/*/; do \
		if [ -f "$$example/main.tf" ]; then \
			echo "Initializing $$example..."; \
			cd "$$example" && terraform init -backend=false && cd ../..; \
		fi; \
	done
	@echo "✅ Terraform examples initialized"

# Development commands
dev-install:
	@echo "🔧 Installing development dependencies..."
	pip install -r requirements.txt
	pip install black flake8 pytest pytest-cov
	@echo "✅ Development dependencies installed"

format:
	@echo "🎨 Formatting code..."
	black blast_radius.py
	@echo "✅ Code formatted"

lint:
	@echo "🔍 Linting code..."
	flake8 blast_radius.py
	@echo "✅ Code linted"

# Quick start
quick-start: install examples
	@echo "🚀 Quick start completed!"
	@echo "Run 'make run' to start the application"
	@echo "Or run 'make docker-run' to start with Docker" 