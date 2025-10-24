#!/bin/bash

# Comprehensive Test Runner for DriftMgr
# This script runs all types of tests: unit, integration, benchmarks, and e2e

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to run tests with coverage
run_tests_with_coverage() {
    local test_type=$1
    local test_path=$2
    local coverage_file=$3
    
    print_status "Running $test_type tests with coverage..."
    
    if [ -d "$test_path" ]; then
        go test -v -race -coverprofile="$coverage_file" -covermode=atomic "$test_path/..."
        if [ $? -eq 0 ]; then
            print_success "$test_type tests passed"
        else
            print_error "$test_type tests failed"
            return 1
        fi
    else
        print_warning "$test_type test directory not found: $test_path"
    fi
}

# Function to run benchmarks
run_benchmarks() {
    local benchmark_path=$1
    
    print_status "Running benchmarks..."
    
    if [ -d "$benchmark_path" ]; then
        go test -bench=. -benchmem -run=^$ "$benchmark_path/..."
        if [ $? -eq 0 ]; then
            print_success "Benchmarks completed"
        else
            print_error "Benchmarks failed"
            return 1
        fi
    else
        print_warning "Benchmark directory not found: $benchmark_path"
    fi
}

# Function to generate coverage report
generate_coverage_report() {
    local coverage_file=$1
    local output_file=$2
    
    if [ -f "$coverage_file" ]; then
        print_status "Generating coverage report..."
        go tool cover -html="$coverage_file" -o "$output_file"
        print_success "Coverage report generated: $output_file"
    else
        print_warning "Coverage file not found: $coverage_file"
    fi
}

# Function to run linting
run_linting() {
    print_status "Running linters..."
    
    # Check if golangci-lint is installed
    if command_exists golangci-lint; then
        golangci-lint run
        if [ $? -eq 0 ]; then
            print_success "Linting passed"
        else
            print_error "Linting failed"
            return 1
        fi
    else
        print_warning "golangci-lint not installed, skipping linting"
    fi
}

# Function to run security scanning
run_security_scan() {
    print_status "Running security scan..."
    
    # Check if gosec is installed
    if command_exists gosec; then
        gosec -fmt json -out security-report.json ./...
        if [ $? -eq 0 ]; then
            print_success "Security scan completed"
        else
            print_warning "Security scan found issues (check security-report.json)"
        fi
    else
        print_warning "gosec not installed, skipping security scan"
    fi
}

# Function to run dependency check
run_dependency_check() {
    print_status "Running dependency check..."
    
    # Check if nancy is installed
    if command_exists nancy; then
        go list -json -deps ./... | nancy sleuth
        if [ $? -eq 0 ]; then
            print_success "Dependency check passed"
        else
            print_error "Dependency check failed"
            return 1
        fi
    else
        print_warning "nancy not installed, skipping dependency check"
    fi
}

# Function to run e2e tests
run_e2e_tests() {
    local e2e_path=$1
    
    print_status "Running end-to-end tests..."
    
    if [ -d "$e2e_path" ]; then
        go test -v -tags=e2e "$e2e_path/..."
        if [ $? -eq 0 ]; then
            print_success "E2E tests passed"
        else
            print_error "E2E tests failed"
            return 1
        fi
    else
        print_warning "E2E test directory not found: $e2e_path"
    fi
}

# Main execution
main() {
    print_status "Starting comprehensive test suite for DriftMgr..."
    
    # Create reports directory
    mkdir -p reports
    
    # Initialize variables
    UNIT_COVERAGE="reports/unit-coverage.out"
    INTEGRATION_COVERAGE="reports/integration-coverage.out"
    COMBINED_COVERAGE="reports/combined-coverage.out"
    COVERAGE_HTML="reports/coverage.html"
    
    # Track overall success
    OVERALL_SUCCESS=true
    
    # Run linting first
    if ! run_linting; then
        OVERALL_SUCCESS=false
    fi
    
    # Run security scan
    run_security_scan
    
    # Run dependency check
    if ! run_dependency_check; then
        OVERALL_SUCCESS=false
    fi
    
    # Run unit tests
    if ! run_tests_with_coverage "Unit" "tests/unit" "$UNIT_COVERAGE"; then
        OVERALL_SUCCESS=false
    fi
    
    # Run integration tests
    if ! run_tests_with_coverage "Integration" "tests/integration" "$INTEGRATION_COVERAGE"; then
        OVERALL_SUCCESS=false
    fi
    
    # Run benchmarks
    if ! run_benchmarks "tests/benchmarks"; then
        OVERALL_SUCCESS=false
    fi
    
    # Run e2e tests
    if ! run_e2e_tests "tests/e2e"; then
        OVERALL_SUCCESS=false
    fi
    
    # Combine coverage reports
    if [ -f "$UNIT_COVERAGE" ] && [ -f "$INTEGRATION_COVERAGE" ]; then
        print_status "Combining coverage reports..."
        echo "mode: atomic" > "$COMBINED_COVERAGE"
        tail -n +2 "$UNIT_COVERAGE" >> "$COMBINED_COVERAGE"
        tail -n +2 "$INTEGRATION_COVERAGE" >> "$COMBINED_COVERAGE"
        
        # Generate combined coverage report
        generate_coverage_report "$COMBINED_COVERAGE" "$COVERAGE_HTML"
    fi
    
    # Generate individual coverage reports
    generate_coverage_report "$UNIT_COVERAGE" "reports/unit-coverage.html"
    generate_coverage_report "$INTEGRATION_COVERAGE" "reports/integration-coverage.html"
    
    # Print summary
    echo ""
    print_status "Test Summary:"
    echo "=============="
    
    if [ -f "$COMBINED_COVERAGE" ]; then
        COVERAGE_PERCENT=$(go tool cover -func="$COMBINED_COVERAGE" | grep total | awk '{print $3}')
        print_status "Overall Coverage: $COVERAGE_PERCENT"
    fi
    
    if [ "$OVERALL_SUCCESS" = true ]; then
        print_success "All tests passed! ✅"
        echo ""
        print_status "Reports generated:"
        echo "  - Coverage HTML: $COVERAGE_HTML"
        echo "  - Security Report: reports/security-report.json"
        echo "  - Unit Coverage: reports/unit-coverage.html"
        echo "  - Integration Coverage: reports/integration-coverage.html"
        exit 0
    else
        print_error "Some tests failed! ❌"
        echo ""
        print_status "Check the output above for details."
        exit 1
    fi
}

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    print_error "go.mod not found. Please run this script from the project root."
    exit 1
fi

# Check if Go is installed
if ! command_exists go; then
    print_error "Go is not installed or not in PATH"
    exit 1
fi

# Run main function
main "$@"
