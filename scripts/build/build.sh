#!/bin/bash

set -e

VERSION=${VERSION:-$(git describe --tags --always --dirty)}
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS="-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME"

echo "Building DriftMgr version $VERSION..."

# Build main CLI
go build -ldflags "$LDFLAGS" -o bin/driftmgr cmd/driftmgr/main.go

# Build server
go build -ldflags "$LDFLAGS" -o bin/driftmgr-server cmd/driftmgr-server/main.go

# Build client
go build -ldflags "$LDFLAGS" -o bin/driftmgr-client cmd/driftmgr-client/main.go

# Build agent
go build -ldflags "$LDFLAGS" -o bin/driftmgr-agent cmd/driftmgr-agent/main.go

echo "Build complete! Binaries are in the bin/ directory."
