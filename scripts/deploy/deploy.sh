#!/bin/bash

set -e

ENVIRONMENT=${1:-development}

echo "Deploying DriftMgr to $ENVIRONMENT..."

# Build the application
./scripts/build/build.sh

# Deploy based on environment
case $ENVIRONMENT in
    development)
        echo "Starting development server..."
        ./bin/driftmgr-server
        ;;
    production)
        echo "Deploying to production..."
        docker-compose -f deployments/docker/docker-compose.prod.yml up -d
        ;;
    *)
        echo "Unknown environment: $ENVIRONMENT"
        exit 1
        ;;
esac

echo "Deployment complete!"
