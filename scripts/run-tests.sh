#!/bin/bash

# DriftMgr Test Runner Script
# This script runs all tests for the DriftMgr application

set -e

echo "🧪 DriftMgr Test Suite Runner"
echo "=============================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
TEST_TIMEOUT="10m"
COVERAGE_THRESHOLD=80
VERBOSE=false
RUN_BENCHMARKS=false
RUN_E2E=false
RUN_PERFORMANCE=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -b|--benchmarks)
            RUN_BENCHMARKS=true
            shift
            ;;
        -e|--e2e)
            RUN_E2E=true
            shift
            ;;
        -p|--performance)
            RUN_PERFORMANCE=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo "Options:"
            echo "  -v, --verbose      Run tests in verbose mode"
            echo "  -b, --benchmarks   Run benchmark tests"
            echo "  -e, --e2e          Run end-to-end tests"
            echo "  -p, --performance  Run performance tests"
            echo "  -h, --help         Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Function to print colored output
print_status() {
    local status=$1
    local message=$2
    case $status in
        "INFO")
            echo -e "${BLUE}ℹ️  $message${NC}"
            ;;
        "SUCCESS")
            echo -e "${GREEN}✅ $message${NC}"
            ;;
        "WARNING")
            echo -e "${YELLOW}⚠️  $message${NC}"
            ;;
        "ERROR")
            echo -e "${RED}❌ $message${NC}"
            ;;
    esac
}

# Function to run tests with timeout
run_tests() {
    local test_path=$1
    local test_name=$2
    local timeout=$3
    
    print_status "INFO" "Running $test_name..."
    
    local start_time=$(date +%s)
    
    if [ "$VERBOSE" = true ]; then
        timeout $timeout go test -v $test_path
    else
        timeout $timeout go test $test_path
    fi
    
    local exit_code=$?
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    if [ $exit_code -eq 0 ]; then
        print_status "SUCCESS" "$test_name completed in ${duration}s"
    else
        print_status "ERROR" "$test_name failed after ${duration}s"
        return $exit_code
    fi
}

# Function to run benchmarks
run_benchmarks() {
    local test_path=$1
    local test_name=$2
    
    print_status "INFO" "Running $test_name benchmarks..."
    
    local start_time=$(date +%s)
    
    go test -bench=. -benchmem $test_path
    
    local exit_code=$?
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    if [ $exit_code -eq 0 ]; then
        print_status "SUCCESS" "$test_name benchmarks completed in ${duration}s"
    else
        print_status "ERROR" "$test_name benchmarks failed after ${duration}s"
        return $exit_code
    fi
}

# Function to check test coverage
check_coverage() {
    local test_path=$1
    local test_name=$2
    
    print_status "INFO" "Checking coverage for $test_name..."
    
    go test -coverprofile=coverage.out $test_path
    local coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    
    if (( $(echo "$coverage >= $COVERAGE_THRESHOLD" | bc -l) )); then
        print_status "SUCCESS" "$test_name coverage: ${coverage}% (threshold: ${COVERAGE_THRESHOLD}%)"
    else
        print_status "WARNING" "$test_name coverage: ${coverage}% (below threshold: ${COVERAGE_THRESHOLD}%)"
    fi
    
    # Generate HTML coverage report
    go tool cover -html=coverage.out -o coverage.html
    print_status "INFO" "Coverage report generated: coverage.html"
}

# Main test execution
main() {
    print_status "INFO" "Starting DriftMgr test suite..."
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        print_status "ERROR" "Go is not installed or not in PATH"
        exit 1
    fi
    
    # Check if bc is installed (for coverage comparison)
    if ! command -v bc &> /dev/null; then
        print_status "WARNING" "bc is not installed, coverage threshold check will be skipped"
    fi
    
    # Set Go test timeout
    export GOTEST_TIMEOUT=$TEST_TIMEOUT
    
    local overall_exit_code=0
    
    # Run unit tests
    print_status "INFO" "Running unit tests..."
    
    # Test individual packages
    packages=(
        "./internal/websocket"
        "./internal/auth"
        "./internal/api"
    )
    
    for package in "${packages[@]}"; do
        if ! run_tests "$package" "Unit tests for $package" "5m"; then
            overall_exit_code=1
        fi
        
        if [ "$VERBOSE" = true ]; then
            check_coverage "$package" "$package"
        fi
    done
    
    # Run integration tests
    print_status "INFO" "Running integration tests..."
    
    if ! run_tests "./internal/api" "API integration tests" "5m"; then
        overall_exit_code=1
    fi
    
    if ! run_tests "./internal/auth" "Auth integration tests" "5m"; then
        overall_exit_code=1
    fi
    
    # Run end-to-end tests
    if [ "$RUN_E2E" = true ]; then
        print_status "INFO" "Running end-to-end tests..."
        
        if ! run_tests "./tests" "End-to-end tests" "10m"; then
            overall_exit_code=1
        fi
    fi
    
    # Run performance tests
    if [ "$RUN_PERFORMANCE" = true ]; then
        print_status "INFO" "Running performance tests..."
        
        if ! run_tests "./tests" "Performance tests" "15m"; then
            overall_exit_code=1
        fi
    fi
    
    # Run benchmarks
    if [ "$RUN_BENCHMARKS" = true ]; then
        print_status "INFO" "Running benchmark tests..."
        
        if ! run_benchmarks "./internal/websocket" "WebSocket"; then
            overall_exit_code=1
        fi
        
        if ! run_benchmarks "./internal/auth" "Auth"; then
            overall_exit_code=1
        fi
        
        if ! run_benchmarks "./internal/api" "API"; then
            overall_exit_code=1
        fi
        
        if ! run_benchmarks "./tests" "Performance"; then
            overall_exit_code=1
        fi
    fi
    
    # Generate overall coverage report
    print_status "INFO" "Generating overall coverage report..."
    
    go test -coverprofile=coverage.out ./...
    local overall_coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    
    if (( $(echo "$overall_coverage >= $COVERAGE_THRESHOLD" | bc -l) )); then
        print_status "SUCCESS" "Overall coverage: ${overall_coverage}% (threshold: ${COVERAGE_THRESHOLD}%)"
    else
        print_status "WARNING" "Overall coverage: ${overall_coverage}% (below threshold: ${COVERAGE_THRESHOLD}%)"
    fi
    
    # Generate HTML coverage report
    go tool cover -html=coverage.out -o coverage.html
    print_status "INFO" "Overall coverage report generated: coverage.html"
    
    # Clean up
    rm -f coverage.out
    
    # Final status
    if [ $overall_exit_code -eq 0 ]; then
        print_status "SUCCESS" "All tests passed! 🎉"
    else
        print_status "ERROR" "Some tests failed! ❌"
    fi
    
    exit $overall_exit_code
}

# Run main function
main "$@"
