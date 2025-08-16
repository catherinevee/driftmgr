#!/bin/bash

set -e

echo "Running tests..."

# Run unit tests
go test ./internal/... -v

# Run integration tests
go test ./tests/integration/... -v

# Run benchmarks
go test ./tests/benchmarks/... -bench=.

echo "All tests passed!"
