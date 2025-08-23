#!/bin/bash

# DriftMgr Test Runner Script
# Provides easy execution of different test suites

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Default values
TEST_TYPE="all"
VERBOSE=false
SHORT=false
COVERAGE=false
BENCHMARK_COUNT=1
OUTPUT_DIR="./test-results"

# Color output functions
print_success() { echo -e "${GREEN}$1${NC}"; }
print_error() { echo -e "${RED}$1${NC}"; }
print_warning() { echo -e "${YELLOW}$1${NC}"; }
print_info() { echo -e "${CYAN}$1${NC}"; }

# Help function
show_help() {
    cat << EOF
DriftMgr Test Runner

Usage: $0 [options]

Test Types:
  -t, --type TYPE        Test type: all, e2e, integration, benchmarks, performance, unit, coverage (default: all)

Options:
  -v, --verbose          Enable verbose test output
  -s, --short            Run tests in short mode (skip long-running tests)
  -c, --coverage         Generate coverage report
  -b, --benchmark-count N Run benchmarks N times (default: 1)
  -o, --output-dir PATH  Specify output directory for results (default: ./test-results)
  -h, --help             Show this help message

Examples:
  # Run all tests
  $0

  # Run integration tests with verbose output
  $0 --type integration --verbose

  # Run benchmarks 5 times
  $0 --type benchmarks --benchmark-count 5

  # Generate coverage report
  $0 --type coverage

  # Run short tests only
  $0 --short

Environment Variables:
  SKIP_CLOUD_TESTS=true     Skip tests requiring cloud credentials
  DRIFTMGR_TEST_DEBUG=true  Enable debug logging in tests
  DRIFTMGR_TEST_TMPDIR      Custom temporary directory for test data

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -t|--type)
            TEST_TYPE="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -s|--short)
            SHORT=true
            shift
            ;;
        -c|--coverage)
            COVERAGE=true
            shift
            ;;
        -b|--benchmark-count)
            BENCHMARK_COUNT="$2"
            shift 2
            ;;
        -o|--output-dir)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Validate test type
case $TEST_TYPE in
    all|e2e|integration|benchmarks|performance|unit|coverage)
        ;;
    *)
        print_error "Invalid test type: $TEST_TYPE"
        print_error "Valid types: all, e2e, integration, benchmarks, performance, unit, coverage"
        exit 1
        ;;
esac

# Ensure we're in the project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Build test flags
TEST_FLAGS=()
if [[ "$VERBOSE" == "true" ]]; then
    TEST_FLAGS+=("-v")
fi
if [[ "$SHORT" == "true" ]]; then
    TEST_FLAGS+=("-short")
fi
if [[ "$COVERAGE" == "true" ]]; then
    TEST_FLAGS+=("-coverprofile=$OUTPUT_DIR/coverage.out")
fi

print_info "DriftMgr Test Runner"
print_info "===================="
print_info "Test Type: $TEST_TYPE"
print_info "Output Directory: $OUTPUT_DIR"
print_info "Flags: ${TEST_FLAGS[*]}"
print_info ""

# Function to run tests with error handling
run_test_command() {
    local command="$1"
    local description="$2"
    
    print_info "Running: $description"
    echo "Command: $command"
    
    local start_time=$(date +%s)
    
    if eval "$command"; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        print_success "✓ $description completed successfully (Duration: ${duration}s)"
        return 0
    else
        print_error "✗ $description failed"
        return 1
    fi
}

# Function to check Go installation
check_go_installation() {
    if command -v go &> /dev/null; then
        local go_version=$(go version)
        print_info "Go installation: $go_version"
        return 0
    else
        print_error "Go is not installed or not in PATH"
        return 1
    fi
}

# Function to install dependencies
install_dependencies() {
    print_info "Installing/updating dependencies..."
    run_test_command "go mod download" "Download dependencies" || return 1
    run_test_command "go mod tidy" "Clean up dependencies" || return 1
    return 0
}

# Main test execution function
run_tests() {
    local all_successful=true
    
    # Check prerequisites
    if ! check_go_installation; then
        exit 1
    fi
    
    # Install dependencies
    if ! install_dependencies; then
        print_error "Failed to install dependencies"
        exit 1
    fi
    
    case $TEST_TYPE in
        "all")
            print_info "Running all test suites..."
            
            # Run unit tests first (if they exist)
            if ls ./internal/*_test.go 1> /dev/null 2>&1; then
                if ! run_test_command "go test ${TEST_FLAGS[*]} ./internal/..." "Unit tests"; then
                    all_successful=false
                fi
            fi
            
            # Run integration tests
            if ! run_test_command "go test ${TEST_FLAGS[*]} ./tests/integration/..." "Integration tests"; then
                all_successful=false
            fi
            
            # Run end-to-end tests
            if ! run_test_command "go test ${TEST_FLAGS[*]} ./tests/e2e/..." "End-to-end tests"; then
                all_successful=false
            fi
            
            # Run performance tests (non-benchmark)
            if ! run_test_command "go test ${TEST_FLAGS[*]} -run=TestPerformance ./tests/benchmarks/..." "Performance tests"; then
                all_successful=false
            fi
            ;;
            
        "e2e")
            if ! run_test_command "go test ${TEST_FLAGS[*]} ./tests/e2e/..." "End-to-end tests"; then
                all_successful=false
            fi
            ;;
            
        "integration")
            if ! run_test_command "go test ${TEST_FLAGS[*]} ./tests/integration/..." "Integration tests"; then
                all_successful=false
            fi
            ;;
            
        "benchmarks")
            local bench_flags="-bench=. -benchmem"
            if [[ $BENCHMARK_COUNT -gt 1 ]]; then
                bench_flags="$bench_flags -count=$BENCHMARK_COUNT"
            fi
            
            local command="go test $bench_flags ./tests/benchmarks/... | tee $OUTPUT_DIR/benchmark-results.txt"
            if run_test_command "$command" "Benchmark tests"; then
                if [[ -f "$OUTPUT_DIR/benchmark-results.txt" ]]; then
                    print_info "Benchmark results saved to: $OUTPUT_DIR/benchmark-results.txt"
                fi
            else
                all_successful=false
            fi
            ;;
            
        "performance")
            if ! run_test_command "go test ${TEST_FLAGS[*]} -run=TestPerformance ./tests/benchmarks/..." "Performance tests"; then
                all_successful=false
            fi
            
            # Also run stress tests
            if ! run_test_command "go test ${TEST_FLAGS[*]} -run=TestStress ./tests/benchmarks/..." "Stress tests"; then
                all_successful=false
            fi
            ;;
            
        "unit")
            if ls ./internal/*_test.go 1> /dev/null 2>&1; then
                if ! run_test_command "go test ${TEST_FLAGS[*]} ./internal/..." "Unit tests"; then
                    all_successful=false
                fi
            else
                print_warning "No unit tests found in ./internal/"
            fi
            ;;
            
        "coverage")
            print_info "Running tests with coverage analysis..."
            
            # Run tests with coverage
            local coverage_flags="${TEST_FLAGS[*]} -coverprofile=$OUTPUT_DIR/coverage.out"
            if run_test_command "go test $coverage_flags ./..." "Tests with coverage"; then
                if [[ -f "$OUTPUT_DIR/coverage.out" ]]; then
                    # Generate HTML coverage report
                    if run_test_command "go tool cover -html=$OUTPUT_DIR/coverage.out -o $OUTPUT_DIR/coverage.html" "Generate HTML coverage report"; then
                        print_success "Coverage report generated: $OUTPUT_DIR/coverage.html"
                        
                        # Show coverage summary
                        print_info "Coverage summary:"
                        go tool cover -func="$OUTPUT_DIR/coverage.out" | tail -1
                    fi
                fi
            else
                all_successful=false
            fi
            ;;
    esac
    
    # Final summary
    print_info ""
    print_info "Test Execution Summary"
    print_info "======================"
    
    if [[ "$all_successful" == "true" ]]; then
        print_success "✓ All tests completed successfully!"
        
        # Show additional information
        if [[ -f "$OUTPUT_DIR/coverage.html" ]]; then
            print_info "Coverage report available at: $OUTPUT_DIR/coverage.html"
        fi
        
        if [[ -f "$OUTPUT_DIR/benchmark-results.txt" ]]; then
            print_info "Benchmark results available at: $OUTPUT_DIR/benchmark-results.txt"
        fi
        
        # Show test artifacts
        if [[ -d "$OUTPUT_DIR" ]] && [[ -n "$(ls -A "$OUTPUT_DIR" 2>/dev/null)" ]]; then
            print_info "Test artifacts:"
            ls -la "$OUTPUT_DIR" | tail -n +2 | while read -r line; do
                echo "  - $(echo "$line" | awk '{print $9}')"
            done
        fi
        
        exit 0
    else
        print_error "✗ Some tests failed. Check the output above for details."
        exit 1
    fi
}

# Trap to clean up on exit
cleanup() {
    if [[ -n "$TEMP_FILES" ]]; then
        rm -f $TEMP_FILES
    fi
}
trap cleanup EXIT

# Start the test execution
run_tests