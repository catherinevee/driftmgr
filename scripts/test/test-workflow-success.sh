#!/bin/bash

# Test script to ensure GitHub Actions workflow will succeed
set -e

echo "🧪 Testing GitHub Actions Workflow Success"
echo "=========================================="

# Test 1: Build DriftMgr
echo "📦 Test 1: Building DriftMgr..."
go build -o driftmgr ./cmd/main.go
chmod +x driftmgr
echo "[OK] Build successful"

# Test 2: Validate binary exists
echo "🔍 Test 2: Validating binary..."
if [ -f "driftmgr" ] && [ -x "driftmgr" ]; then
    echo "[OK] Binary exists and is executable"
    ls -la driftmgr
else
    echo "[ERROR] Binary validation failed"
    exit 1
fi

# Test 3: Test GitHub Actions integration
echo "🚀 Test 3: Testing GitHub Actions integration..."
export WORKFLOW_TYPE="drift-analysis"
export PROVIDER="aws"
export REGIONS="us-east-1"
export ENVIRONMENT="test"
export DRY_RUN="true"
export PARALLEL_IMPORTS="5"
export OUTPUT_FORMAT="json"

./driftmgr github-actions validate-inputs
echo "[OK] GitHub Actions validation passed"

# Test 4: Test environment setup
echo "🔧 Test 4: Testing environment setup..."
./driftmgr github-actions setup-env
echo "[OK] Environment setup passed"

# Test 5: Test report generation
echo "📊 Test 5: Testing report generation..."
./driftmgr github-actions generate-report --output test-workflow-report.md
if [ -f "test-workflow-report.md" ]; then
    echo "[OK] Report generation passed"
    echo "📄 Report preview:"
    head -10 test-workflow-report.md
else
    echo "[ERROR] Report generation failed"
    exit 1
fi

# Test 6: Test workflow dispatch (dry run)
echo "🎯 Test 6: Testing workflow dispatch..."
./driftmgr github-actions workflow-dispatch --type drift-analysis --provider aws --regions us-east-1 --environment test --dry-run
echo "[OK] Workflow dispatch passed"

# Test 7: Check generated files
echo "📁 Test 7: Checking generated files..."
if [ -d "driftmgr-data" ]; then
    echo "[OK] Data directory created"
    ls -la driftmgr-data/
else
    echo "[WARNING] No data directory found (expected for dry run)"
fi

echo ""
echo "🎉 All tests passed! GitHub Actions workflow will succeed."
echo ""
echo "📋 Summary:"
echo "- [OK] Build process works"
echo "- [OK] Binary validation works"
echo "- [OK] GitHub Actions integration works"
echo "- [OK] Environment setup works"
echo "- [OK] Report generation works"
echo "- [OK] Workflow dispatch works"
echo ""
echo "🚀 Ready for GitHub Actions deployment!"
