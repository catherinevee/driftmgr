#!/bin/bash

# DriftMgr Comprehensive Test Runner
# This script runs all types of tests including unit, integration, e2e, and benchmarks

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
TEST_TIMEOUT="10m"
COVERAGE_THRESHOLD=80
BENCHMARK_ITERATIONS=100
PARALLEL_TESTS=4

# Directories
UNIT_TEST_DIR="tests/unit"
INTEGRATION_TEST_DIR="tests"
E2E_TEST_DIR="tests/e2e"
BENCHMARK_TEST_DIR="tests/benchmarks"
COVERAGE_DIR="coverage"

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    # Check Go version
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    REQUIRED_VERSION="1.21"
    
    if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
        log_error "Go version $GO_VERSION is less than required version $REQUIRED_VERSION"
        exit 1
    fi
    
    # Check if testify is installed
    if ! go list -f '{{.Dir}}' github.com/stretchr/testify &> /dev/null; then
        log_info "Installing testify..."
        go get github.com/stretchr/testify
    fi
    
    log_success "Prerequisites check passed"
}

setup_test_environment() {
    log_info "Setting up test environment..."
    
    # Create coverage directory
    mkdir -p "$COVERAGE_DIR"
    
    # Set test environment variables
    export DRIFT_TEST_MODE=true
    export DRIFT_LOG_LEVEL=debug
    export DRIFT_TEST_TIMEOUT=$TEST_TIMEOUT
    
    # Create test configuration
    cat > driftmgr.test.yaml << EOF
test:
  enabled: true
  timeout: $TEST_TIMEOUT
  parallel: $PARALLEL_TESTS
  coverage:
    enabled: true
    threshold: $COVERAGE_THRESHOLD
    output: $COVERAGE_DIR
  benchmarks:
    iterations: $BENCHMARK_ITERATIONS
    timeout: 5m
EOF
    
    log_success "Test environment setup complete"
}

run_unit_tests() {
    log_info "Running unit tests..."
    
    local start_time=$(date +%s)
    local test_packages=$(find "$UNIT_TEST_DIR" -name "*_test.go" -exec dirname {} \; | sort -u)
    
    if [ -z "$test_packages" ]; then
        log_warning "No unit test packages found"
        return 0
    fi
    
    local failed_tests=0
    
    for package in $test_packages; do
        log_info "Testing package: $package"
        
        if ! go test -v -timeout="$TEST_TIMEOUT" -coverprofile="$COVERAGE_DIR/unit_$(basename $package).out" "$package"; then
            log_error "Unit tests failed in package: $package"
            failed_tests=$((failed_tests + 1))
        fi
    done
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    if [ $failed_tests -eq 0 ]; then
        log_success "Unit tests completed in ${duration}s"
    else
        log_error "Unit tests failed: $failed_tests packages"
        return 1
    fi
}

run_integration_tests() {
    log_info "Running integration tests..."
    
    local start_time=$(date +%s)
    
    if ! go test -v -timeout="$TEST_TIMEOUT" -coverprofile="$COVERAGE_DIR/integration.out" ./tests; then
        log_error "Integration tests failed"
        return 1
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    log_success "Integration tests completed in ${duration}s"
}

run_e2e_tests() {
    log_info "Running end-to-end tests..."
    
    local start_time=$(date +%s)
    
    # Check if e2e test directory exists
    if [ ! -d "$E2E_TEST_DIR" ]; then
        log_warning "E2E test directory not found: $E2E_TEST_DIR"
        return 0
    fi
    
    local e2e_packages=$(find "$E2E_TEST_DIR" -name "*_test.go" -exec dirname {} \; | sort -u)
    
    if [ -z "$e2e_packages" ]; then
        log_warning "No E2E test packages found"
        return 0
    fi
    
    local failed_tests=0
    
    for package in $e2e_packages; do
        log_info "Running E2E tests in package: $package"
        
        if ! go test -v -timeout="$TEST_TIMEOUT" -coverprofile="$COVERAGE_DIR/e2e_$(basename $package).out" "$package"; then
            log_error "E2E tests failed in package: $package"
            failed_tests=$((failed_tests + 1))
        fi
    done
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    if [ $failed_tests -eq 0 ]; then
        log_success "E2E tests completed in ${duration}s"
    else
        log_error "E2E tests failed: $failed_tests packages"
        return 1
    fi
}

run_benchmarks() {
    log_info "Running benchmarks..."
    
    local start_time=$(date +%s)
    
    # Check if benchmark test directory exists
    if [ ! -d "$BENCHMARK_TEST_DIR" ]; then
        log_warning "Benchmark test directory not found: $BENCHMARK_TEST_DIR"
        return 0
    fi
    
    local benchmark_packages=$(find "$BENCHMARK_TEST_DIR" -name "*_test.go" -exec dirname {} \; | sort -u)
    
    if [ -z "$benchmark_packages" ]; then
        log_warning "No benchmark test packages found"
        return 0
    fi
    
    local failed_benchmarks=0
    
    for package in $benchmark_packages; do
        log_info "Running benchmarks in package: $package"
        
        if ! go test -v -bench=. -benchmem -timeout="$TEST_TIMEOUT" -coverprofile="$COVERAGE_DIR/benchmark_$(basename $package).out" "$package"; then
            log_error "Benchmarks failed in package: $package"
            failed_benchmarks=$((failed_benchmarks + 1))
        fi
    done
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    if [ $failed_benchmarks -eq 0 ]; then
        log_success "Benchmarks completed in ${duration}s"
    else
        log_error "Benchmarks failed: $failed_benchmarks packages"
        return 1
    fi
}

run_security_tests() {
    log_info "Running security tests..."
    
    local start_time=$(date +%s)
    
    # Run security-specific tests
    local security_test_packages=$(find . -name "*security*_test.go" -exec dirname {} \; | sort -u)
    
    if [ -z "$security_test_packages" ]; then
        log_warning "No security test packages found"
        return 0
    fi
    
    local failed_tests=0
    
    for package in $security_test_packages; do
        log_info "Running security tests in package: $package"
        
        if ! go test -v -timeout="$TEST_TIMEOUT" -coverprofile="$COVERAGE_DIR/security_$(basename $package).out" "$package"; then
            log_error "Security tests failed in package: $package"
            failed_tests=$((failed_tests + 1))
        fi
    done
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    if [ $failed_tests -eq 0 ]; then
        log_success "Security tests completed in ${duration}s"
    else
        log_error "Security tests failed: $failed_tests packages"
        return 1
    fi
}

generate_coverage_report() {
    log_info "Generating coverage report..."
    
    # Combine all coverage files
    if [ -d "$COVERAGE_DIR" ]; then
        local coverage_files=$(find "$COVERAGE_DIR" -name "*.out" -type f)
        
        if [ -n "$coverage_files" ]; then
            # Create combined coverage file
            echo "mode: set" > "$COVERAGE_DIR/coverage.out"
            
            for file in $coverage_files; do
                if [ -f "$file" ]; then
                    tail -n +2 "$file" >> "$COVERAGE_DIR/coverage.out" 2>/dev/null || true
                fi
            done
            
            # Generate HTML coverage report
            go tool cover -html="$COVERAGE_DIR/coverage.out" -o "$COVERAGE_DIR/coverage.html"
            
            # Calculate coverage percentage
            local coverage_percent=$(go tool cover -func="$COVERAGE_DIR/coverage.out" | tail -1 | awk '{print $3}' | sed 's/%//')
            
            log_info "Coverage: ${coverage_percent}%"
            
            if (( $(echo "$coverage_percent >= $COVERAGE_THRESHOLD" | bc -l) )); then
                log_success "Coverage threshold met: ${coverage_percent}% >= ${COVERAGE_THRESHOLD}%"
            else
                log_warning "Coverage below threshold: ${coverage_percent}% < ${COVERAGE_THRESHOLD}%"
            fi
        else
            log_warning "No coverage files found"
        fi
    else
        log_warning "Coverage directory not found"
    fi
}

run_static_analysis() {
    log_info "Running static analysis..."
    
    # Run golangci-lint if available
    if command -v golangci-lint &> /dev/null; then
        if golangci-lint run; then
            log_success "Static analysis passed"
        else
            log_error "Static analysis failed"
            return 1
        fi
    else
        log_warning "golangci-lint not found, skipping static analysis"
    fi
    
    # Run go vet
    if go vet ./...; then
        log_success "Go vet passed"
    else
        log_error "Go vet failed"
        return 1
    fi
}

run_security_scan() {
    log_info "Running security scan..."
    
    # Run gosec if available
    if command -v gosec &> /dev/null; then
        if gosec ./...; then
            log_success "Security scan passed"
        else
            log_warning "Security scan found issues"
        fi
    else
        log_warning "gosec not found, skipping security scan"
    fi
}

cleanup() {
    log_info "Cleaning up test artifacts..."
    
    # Remove test configuration
    rm -f driftmgr.test.yaml
    
    # Clean test cache
    go clean -testcache
    
    log_success "Cleanup complete"
}

print_summary() {
    local total_start_time=$1
    local total_end_time=$(date +%s)
    local total_duration=$((total_end_time - total_start_time))
    
    echo
    echo "=========================================="
    echo "           TEST SUMMARY"
    echo "=========================================="
    echo "Total Duration: ${total_duration}s"
    echo "Coverage Report: $COVERAGE_DIR/coverage.html"
    echo "Test Configuration: driftmgr.test.yaml"
    echo "=========================================="
    echo
}

# Main execution
main() {
    local total_start_time=$(date +%s)
    
    echo "=========================================="
    echo "    DriftMgr Comprehensive Test Suite"
    echo "=========================================="
    echo
    
    # Check prerequisites
    check_prerequisites
    
    # Setup test environment
    setup_test_environment
    
    # Run static analysis
    if ! run_static_analysis; then
        log_error "Static analysis failed"
        cleanup
        exit 1
    fi
    
    # Run security scan
    run_security_scan
    
    # Run unit tests
    if ! run_unit_tests; then
        log_error "Unit tests failed"
        cleanup
        exit 1
    fi
    
    # Run integration tests
    if ! run_integration_tests; then
        log_error "Integration tests failed"
        cleanup
        exit 1
    fi
    
    # Run E2E tests
    if ! run_e2e_tests; then
        log_error "E2E tests failed"
        cleanup
        exit 1
    fi
    
    # Run security tests
    if ! run_security_tests; then
        log_error "Security tests failed"
        cleanup
        exit 1
    fi
    
    # Run benchmarks
    if ! run_benchmarks; then
        log_error "Benchmarks failed"
        cleanup
        exit 1
    fi
    
    # Generate coverage report
    generate_coverage_report
    
    # Cleanup
    cleanup
    
    # Print summary
    print_summary $total_start_time
    
    log_success "All tests completed successfully!"
}

# Handle script arguments
case "${1:-}" in
    "unit")
        check_prerequisites
        setup_test_environment
        run_unit_tests
        generate_coverage_report
        cleanup
        ;;
    "integration")
        check_prerequisites
        setup_test_environment
        run_integration_tests
        generate_coverage_report
        cleanup
        ;;
    "e2e")
        check_prerequisites
        setup_test_environment
        run_e2e_tests
        generate_coverage_report
        cleanup
        ;;
    "benchmarks")
        check_prerequisites
        setup_test_environment
        run_benchmarks
        cleanup
        ;;
    "security")
        check_prerequisites
        setup_test_environment
        run_security_tests
        run_security_scan
        cleanup
        ;;
    "coverage")
        generate_coverage_report
        ;;
    "clean")
        cleanup
        ;;
    "help"|"-h"|"--help")
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  unit        Run unit tests only"
        echo "  integration Run integration tests only"
        echo "  e2e         Run end-to-end tests only"
        echo "  benchmarks  Run benchmarks only"
        echo "  security    Run security tests only"
        echo "  coverage    Generate coverage report only"
        echo "  clean       Clean up test artifacts"
        echo "  help        Show this help message"
        echo ""
        echo "If no command is specified, all tests will be run."
        ;;
    *)
        main
        ;;
esac
