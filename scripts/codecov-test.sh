#!/bin/bash

# Codecov Upload Test Script for DriftMgr
# This script validates the Codecov upload process locally

set -e

echo "=== Codecov Upload Test Script ==="
echo "This script tests the Codecov integration for DriftMgr"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    case $1 in
        "SUCCESS") echo -e "${GREEN}✓ $2${NC}" ;;
        "ERROR") echo -e "${RED}✗ $2${NC}" ;;
        "WARNING") echo -e "${YELLOW}⚠ $2${NC}" ;;
        "INFO") echo -e "${BLUE}ℹ $2${NC}" ;;
    esac
}

# Check if we're in the right directory
if [ ! -f "go.mod" ] || [ ! -d "internal" ]; then
    print_status "ERROR" "Not in DriftMgr root directory. Please run from project root."
    exit 1
fi

print_status "INFO" "Starting Codecov upload test..."

# Step 1: Check environment
print_status "INFO" "Checking environment..."

# Check Go version
if command -v go &> /dev/null; then
    GO_VERSION=$(go version | awk '{print $3}')
    print_status "SUCCESS" "Go found: $GO_VERSION"
else
    print_status "ERROR" "Go not found. Please install Go 1.23+"
    exit 1
fi

# Check Git repository
if git rev-parse --git-dir > /dev/null 2>&1; then
    BRANCH=$(git rev-parse --abbrev-ref HEAD)
    COMMIT=$(git rev-parse HEAD)
    print_status "SUCCESS" "Git repository detected"
    print_status "INFO" "Branch: $BRANCH"
    print_status "INFO" "Commit: ${COMMIT:0:8}"
else
    print_status "ERROR" "Not a Git repository"
    exit 1
fi

# Step 2: Run tests and generate coverage
print_status "INFO" "Running tests and generating coverage..."

# Clean up previous coverage files
rm -f coverage*.out combined_coverage.out

# Run a smaller subset of tests first (to avoid timeout)
TEST_PACKAGES=(
    "./internal/state/backend"
    "./internal/providers/factory"
    "./internal/api/handlers"
)

for pkg in "${TEST_PACKAGES[@]}"; do
    if [ -d "${pkg#./}" ]; then
        print_status "INFO" "Testing package: $pkg"

        # Run test with timeout and capture output
        if go test -v -race -coverprofile="${pkg//\//_}_coverage.out" -covermode=atomic "$pkg" -timeout 15s 2>/dev/null; then
            print_status "SUCCESS" "Tests passed for $pkg"
        else
            print_status "WARNING" "Tests failed for $pkg (continuing anyway)"
        fi
    fi
done

# Merge coverage files
print_status "INFO" "Merging coverage files..."
echo "mode: atomic" > combined_coverage.out

for coverage_file in *_coverage.out; do
    if [ -f "$coverage_file" ] && [ "$coverage_file" != "combined_coverage.out" ]; then
        tail -n +2 "$coverage_file" >> combined_coverage.out 2>/dev/null || true
    fi
done

# Check if we have coverage data
if [ -s "combined_coverage.out" ]; then
    COVERAGE_LINES=$(wc -l < combined_coverage.out)
    print_status "SUCCESS" "Coverage file generated with $COVERAGE_LINES lines"

    # Generate coverage report
    if go tool cover -func=combined_coverage.out > coverage_report.txt 2>/dev/null; then
        TOTAL_COVERAGE=$(tail -1 coverage_report.txt | awk '{print $3}')
        print_status "SUCCESS" "Total coverage: $TOTAL_COVERAGE"
    else
        print_status "WARNING" "Could not generate coverage report"
    fi
else
    print_status "ERROR" "No coverage data generated"
    exit 1
fi

# Step 3: Check Codecov configuration
print_status "INFO" "Checking Codecov configuration..."

if [ -f "codecov.yml" ]; then
    print_status "SUCCESS" "codecov.yml found"

    # Basic YAML validation (check if it parses)
    if command -v python3 &> /dev/null; then
        if python3 -c "import yaml; yaml.safe_load(open('codecov.yml'))" 2>/dev/null; then
            print_status "SUCCESS" "codecov.yml is valid YAML"
        else
            print_status "WARNING" "codecov.yml may have syntax issues"
        fi
    fi
else
    print_status "ERROR" "codecov.yml not found"
fi

# Step 4: Test Codecov upload (dry run)
print_status "INFO" "Testing Codecov upload process..."

# Check if CODECOV_TOKEN is set
if [ -n "$CODECOV_TOKEN" ]; then
    print_status "SUCCESS" "CODECOV_TOKEN is set"
    TOKEN_FLAG="-t $CODECOV_TOKEN"
else
    print_status "WARNING" "CODECOV_TOKEN not set (required for private repos)"
    TOKEN_FLAG=""
fi

# Try to download codecov uploader if not present
if [ ! -f "./codecov.exe" ] && [ ! -f "./codecov" ]; then
    print_status "INFO" "Downloading Codecov uploader..."

    if command -v curl &> /dev/null; then
        if [[ "$OSTYPE" == "msys" || "$OSTYPE" == "win32" ]]; then
            curl -Os https://cli.codecov.io/latest/windows/codecov.exe
            CODECOV_CMD="./codecov.exe"
        else
            curl -Os https://cli.codecov.io/latest/linux/codecov
            chmod +x codecov
            CODECOV_CMD="./codecov"
        fi
        print_status "SUCCESS" "Codecov uploader downloaded"
    else
        print_status "WARNING" "Could not download Codecov uploader (curl not found)"
        CODECOV_CMD=""
    fi
else
    if [ -f "./codecov.exe" ]; then
        CODECOV_CMD="./codecov.exe"
    else
        CODECOV_CMD="./codecov"
    fi
    print_status "SUCCESS" "Codecov uploader found"
fi

# Test upload (dry run)
if [ -n "$CODECOV_CMD" ]; then
    print_status "INFO" "Testing Codecov upload (dry run)..."

    if $CODECOV_CMD --dry-run \
        --file combined_coverage.out \
        --flags unittests \
        --name "codecov-test-$(date +%s)" \
        --verbose \
        $TOKEN_FLAG 2>&1 | head -20; then
        print_status "SUCCESS" "Codecov dry run completed"
    else
        print_status "WARNING" "Codecov dry run had issues (check output above)"
    fi
else
    print_status "WARNING" "Skipping Codecov upload test (uploader not available)"
fi

# Step 5: GitHub Actions workflow validation
print_status "INFO" "Checking GitHub Actions workflow..."

if [ -f ".github/workflows/test-coverage.yml" ]; then
    print_status "SUCCESS" "test-coverage.yml workflow found"

    # Check for required components
    if grep -q "codecov/codecov-action@v4" ".github/workflows/test-coverage.yml"; then
        print_status "SUCCESS" "Uses latest Codecov GitHub Action (v4)"
    else
        print_status "WARNING" "May not be using latest Codecov GitHub Action"
    fi

    if grep -q "CODECOV_TOKEN" ".github/workflows/test-coverage.yml"; then
        print_status "SUCCESS" "CODECOV_TOKEN configured in workflow"
    else
        print_status "WARNING" "CODECOV_TOKEN not found in workflow"
    fi
else
    print_status "ERROR" "GitHub Actions workflow not found"
fi

# Step 6: Generate summary
print_status "INFO" "Test Summary:"
echo ""
echo "Files Generated:"
[ -f "combined_coverage.out" ] && echo "  - combined_coverage.out (coverage data)"
[ -f "coverage_report.txt" ] && echo "  - coverage_report.txt (coverage report)"
echo ""

# Cleanup option
read -p "Clean up generated files? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm -f *_coverage.out combined_coverage.out coverage_report.txt codecov.exe codecov
    print_status "INFO" "Cleanup completed"
fi

print_status "SUCCESS" "Codecov test completed!"
echo ""
echo "Next steps:"
echo "1. Set CODECOV_TOKEN secret in GitHub repository settings"
echo "2. Ensure your repository is connected to Codecov.io"
echo "3. Run the test-coverage.yml GitHub Actions workflow"
echo "4. Check Codecov dashboard for reports"