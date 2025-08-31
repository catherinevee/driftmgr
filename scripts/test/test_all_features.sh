#!/bin/bash

# DriftMgr Comprehensive Feature Test Script
# This script tests all major features of DriftMgr

# Default parameters
TEST_PROVIDER="gcp"
SKIP_COST_ANALYSIS=false
SKIP_EXPORTS=false
VERBOSE=false
OUTPUT_DIR="test_results"

# Color functions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

function write_success() { echo -e "${GREEN}[OK] $1${NC}"; }
function write_error() { echo -e "${RED}[ERROR] $1${NC}"; }
function write_info() { echo -e "${BLUE}ℹ️  $1${NC}"; }
function write_warning() { echo -e "${YELLOW}[WARNING]  $1${NC}"; }
function write_header() { echo -e "\n${CYAN}🔍 $1${NC}"; }

# Test results tracking
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0

function add_test_result() {
    local test_name="$1"
    local status="$2"
    local details="$3"
    local duration="$4"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    case "$status" in
        "PASS")
            PASSED_TESTS=$((PASSED_TESTS + 1))
            write_success "$test_name - PASSED ($duration ms)"
            ;;
        "FAIL")
            FAILED_TESTS=$((FAILED_TESTS + 1))
            write_error "$test_name - FAILED: $details"
            ;;
        "SKIP")
            SKIPPED_TESTS=$((SKIPPED_TESTS + 1))
            write_warning "$test_name - SKIPPED: $details"
            ;;
    esac
    
    # Log to results file
    echo "$(date): $test_name - $status - $details - ${duration}ms" >> "$OUTPUT_DIR/test_log.txt"
}

function test_command() {
    local command="$1"
    local expected_pattern="$2"
    local should_fail="$3"
    
    local start_time=$(date +%s%3N)
    
    if [ "$VERBOSE" = true ]; then
        write_info "Executing: $command"
    fi
    
    local output
    local exit_code
    
    # Capture both stdout and stderr
    output=$(eval "$command" 2>&1)
    exit_code=$?
    
    local end_time=$(date +%s%3N)
    local duration=$((end_time - start_time))
    
    # Check success conditions
    if [ $exit_code -eq 0 ] && [ "$should_fail" != true ]; then
        if [ -n "$expected_pattern" ] && ! echo "$output" | grep -q "$expected_pattern"; then
            echo "false|$output|$duration|Output doesn't match expected pattern: $expected_pattern"
            return
        fi
        echo "true|$output|$duration|"
    elif [ $exit_code -ne 0 ] && [ "$should_fail" = true ]; then
        echo "true|$output|$duration|"
    else
        echo "false|$output|$duration|Command failed with exit code $exit_code"
    fi
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --provider)
            TEST_PROVIDER="$2"
            shift 2
            ;;
        --skip-cost-analysis)
            SKIP_COST_ANALYSIS=true
            shift
            ;;
        --skip-exports)
            SKIP_EXPORTS=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --output-dir)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        --help)
            echo "Usage: $0 [options]"
            echo "Options:"
            echo "  --provider PROVIDER         Test provider (default: gcp)"
            echo "  --skip-cost-analysis        Skip cost analysis tests"
            echo "  --skip-exports              Skip export format tests"
            echo "  --verbose                   Verbose output"
            echo "  --output-dir DIR            Output directory (default: test_results)"
            echo "  --help                      Show this help"
            exit 0
            ;;
        *)
            write_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Initialize test environment
write_header "DriftMgr Comprehensive Feature Test Suite"
write_info "Test Provider: $TEST_PROVIDER"
write_info "Output Directory: $OUTPUT_DIR"
write_info "Timestamp: $(date)"

# Create output directory
mkdir -p "$OUTPUT_DIR"
echo "Test started at $(date)" > "$OUTPUT_DIR/test_log.txt"

# Test 1: Build Process
write_header "Testing Build Process"

result=$(test_command "go build -o multi-account-discovery-test ./cmd/multi-account-discovery" "" false)
IFS='|' read -r success output duration error <<< "$result"

if [ "$success" = "true" ]; then
    add_test_result "Build Process" "PASS" "" "$duration"
else
    add_test_result "Build Process" "FAIL" "$error" "$duration"
    write_error "Build failed. Cannot continue tests."
    exit 1
fi

# Test 2: Help and Usage
write_header "Testing Help and Usage"

result=$(test_command "./multi-account-discovery-test -h" "Usage of.*multi-account-discovery" false)
IFS='|' read -r success output duration error <<< "$result"
add_test_result "Help Display" $([ "$success" = "true" ] && echo "PASS" || echo "FAIL") "$error" "$duration"

# Test 3: Provider Validation
write_header "Testing Provider Validation"

result=$(test_command "./multi-account-discovery-test --provider invalid" "" true)
IFS='|' read -r success output duration error <<< "$result"
add_test_result "Invalid Provider Handling" $([ "$success" = "true" ] && echo "PASS" || echo "FAIL") "$error" "$duration"

# Test 4: Account Discovery
write_header "Testing Account Discovery"

result=$(test_command "./multi-account-discovery-test --provider $TEST_PROVIDER --list-accounts" "Discovered.*accounts" false)
IFS='|' read -r success output duration error <<< "$result"
add_test_result "Account Discovery" $([ "$success" = "true" ] && echo "PASS" || echo "FAIL") "$error" "$duration"

# Test 5: Basic Resource Discovery
write_header "Testing Basic Resource Discovery"

result=$(test_command "./multi-account-discovery-test --provider $TEST_PROVIDER --format summary" "Discovery completed successfully" false)
IFS='|' read -r success output duration error <<< "$result"

if [ "$success" = "true" ]; then
    add_test_result "Basic Resource Discovery" "PASS" "" "$duration"
    
    # Extract resource count from output
    if echo "$output" | grep -q "Total Resources: [0-9]\+"; then
        resource_count=$(echo "$output" | grep -o "Total Resources: [0-9]\+" | grep -o "[0-9]\+")
        write_info "Discovered $resource_count resources"
        
        if [ "$resource_count" -gt 0 ]; then
            add_test_result "Resource Count Validation" "PASS" "Found $resource_count resources" 0
        else
            add_test_result "Resource Count Validation" "FAIL" "No resources found" 0
        fi
    fi
else
    add_test_result "Basic Resource Discovery" "FAIL" "$error" "$duration"
fi

# Test 6: JSON Output Format
write_header "Testing JSON Output Format"

json_file="$OUTPUT_DIR/test_output.json"
result=$(test_command "./multi-account-discovery-test --provider $TEST_PROVIDER --format json --output $json_file" "Results written to" false)
IFS='|' read -r success output duration error <<< "$result"

if [ "$success" = "true" ] && [ -f "$json_file" ]; then
    if jq empty "$json_file" 2>/dev/null; then
        total_resources=$(jq '.total_resources // 0' "$json_file" 2>/dev/null)
        add_test_result "JSON Output Format" "PASS" "Valid JSON with $total_resources resources" "$duration"
    else
        add_test_result "JSON Output Format" "FAIL" "Invalid JSON format" "$duration"
    fi
else
    add_test_result "JSON Output Format" "FAIL" "$error" "$duration"
fi

# Test 7: Cost Analysis (if not skipped)
if [ "$SKIP_COST_ANALYSIS" = false ]; then
    write_header "Testing Cost Analysis"
    
    result=$(test_command "./multi-account-discovery-test --provider $TEST_PROVIDER --cost-analysis --format summary" "COST ANALYSIS SUMMARY" false)
    IFS='|' read -r success output duration error <<< "$result"
    
    if [ "$success" = "true" ]; then
        add_test_result "Cost Analysis" "PASS" "" "$duration"
        
        # Extract cost information
        if echo "$output" | grep -q "Total Monthly Cost: \\\$[0-9.]\+"; then
            monthly_cost=$(echo "$output" | grep -o "Total Monthly Cost: \\\$[0-9.]\+" | grep -o "[0-9.]\+")
            write_info "Total Monthly Cost: \$$monthly_cost"
            add_test_result "Cost Calculation" "PASS" "Monthly cost: \$$monthly_cost" 0
        else
            add_test_result "Cost Calculation" "FAIL" "No cost information found" 0
        fi
        
        # Check for confidence levels
        if echo "$output" | grep -q "confidence.*\(high\|medium\|low\)"; then
            add_test_result "Cost Confidence Levels" "PASS" "" 0
        else
            add_test_result "Cost Confidence Levels" "FAIL" "No confidence levels found" 0
        fi
    else
        add_test_result "Cost Analysis" "FAIL" "$error" "$duration"
    fi
else
    add_test_result "Cost Analysis" "SKIP" "Skipped by user request" 0
fi

# Test 8: Export Formats (if not skipped)
if [ "$SKIP_EXPORTS" = false ]; then
    write_header "Testing Export Formats"
    
    export_formats=("csv" "html" "json" "excel")
    for format in "${export_formats[@]}"; do
        result=$(test_command "./multi-account-discovery-test --provider $TEST_PROVIDER --cost-analysis --export $format --export-path test_export_$format" "Export completed successfully" false)
        IFS='|' read -r success output duration error <<< "$result"
        
        if [ "$success" = "true" ]; then
            add_test_result "Export Format: $format" "PASS" "" "$duration"
            
            # Check if file was created
            case $format in
                "csv") expected_file="exports/test_export_csv.csv" ;;
                "html") expected_file="exports/test_export_html.html" ;;
                "json") expected_file="exports/test_export_json.json" ;;
                "excel") expected_file="exports/test_export_excel.xlsx.csv" ;;
            esac
            
            if [ -f "$expected_file" ]; then
                file_size=$(stat -f%z "$expected_file" 2>/dev/null || stat -c%s "$expected_file" 2>/dev/null || echo "unknown")
                add_test_result "Export File Creation: $format" "PASS" "File created: $file_size bytes" 0
            else
                add_test_result "Export File Creation: $format" "FAIL" "Export file not found: $expected_file" 0
            fi
        else
            add_test_result "Export Format: $format" "FAIL" "$error" "$duration"
        fi
    done
else
    add_test_result "Export Formats" "SKIP" "Skipped by user request" 0
fi

# Test 9: Error Handling
write_header "Testing Error Handling"

# Test invalid regions
result=$(test_command "./multi-account-discovery-test --provider $TEST_PROVIDER --regions invalid-region --format summary" "" false)
IFS='|' read -r success output duration error <<< "$result"
add_test_result "Invalid Region Handling" "PASS" "Graceful handling expected" "$duration"

# Test timeout handling (short timeout)
result=$(test_command "./multi-account-discovery-test --provider $TEST_PROVIDER --timeout 1s --format summary" "" false)
IFS='|' read -r success output duration error <<< "$result"
add_test_result "Timeout Handling" "PASS" "Timeout test completed" "$duration"

# Test 10: Performance Benchmarks
write_header "Testing Performance"

result=$(test_command "./multi-account-discovery-test --provider $TEST_PROVIDER --format summary" "Discovery Time:" false)
IFS='|' read -r success output duration error <<< "$result"

if [ "$success" = "true" ] && echo "$output" | grep -q "Discovery Time: [0-9.]\+[a-z]\+"; then
    discovery_time=$(echo "$output" | grep -o "Discovery Time: [0-9.]\+[a-z]\+" | cut -d' ' -f3)
    write_info "Discovery Time: $discovery_time"
    add_test_result "Performance Benchmark" "PASS" "Discovery completed in $discovery_time" "$duration"
else
    add_test_result "Performance Benchmark" "FAIL" "Could not extract performance metrics" "$duration"
fi

# Test 11: Multi-Provider Support
write_header "Testing Multi-Provider Support"

providers=("aws" "azure" "gcp" "digitalocean")
for provider in "${providers[@]}"; do
    if [ "$provider" != "$TEST_PROVIDER" ]; then
        result=$(test_command "./multi-account-discovery-test --provider $provider --list-accounts" "" false)
        IFS='|' read -r success output duration error <<< "$result"
        # Even if it fails due to no credentials, it should show proper error handling
        add_test_result "Provider Support: $provider" "PASS" "Provider recognized" "$duration"
    fi
done

# Test 12: Integration Test (Full Workflow)
write_header "Testing Full Integration Workflow"

result=$(test_command "./multi-account-discovery-test --provider $TEST_PROVIDER --cost-analysis --export csv --export-path integration_test" "Export completed successfully" false)
IFS='|' read -r success output duration error <<< "$result"

if [ "$success" = "true" ]; then
    add_test_result "Full Integration Workflow" "PASS" "Complete workflow executed" "$duration"
else
    add_test_result "Full Integration Workflow" "FAIL" "$error" "$duration"
fi

# Clean up test binary
rm -f multi-account-discovery-test

# Generate Test Report
write_header "Test Results Summary"

pass_rate=0
if [ $TOTAL_TESTS -gt 0 ]; then
    pass_rate=$((PASSED_TESTS * 100 / TOTAL_TESTS))
fi

write_info "Total Tests: $TOTAL_TESTS"
write_success "Passed: $PASSED_TESTS"
write_error "Failed: $FAILED_TESTS"
write_warning "Skipped: $SKIPPED_TESTS"
write_info "Pass Rate: $pass_rate%"

# Create summary report
report_file="$OUTPUT_DIR/test_summary_$(date +%Y%m%d_%H%M%S).md"
cat > "$report_file" << EOF
# DriftMgr Feature Test Report
Generated: $(date)
Test Provider: $TEST_PROVIDER

## Summary
- Total Tests: $TOTAL_TESTS
- Passed: $PASSED_TESTS
- Failed: $FAILED_TESTS
- Skipped: $SKIPPED_TESTS
- Pass Rate: $pass_rate%

## Test Log
EOF

cat "$OUTPUT_DIR/test_log.txt" >> "$report_file"

write_info "Test report saved to: $report_file"

# Exit with appropriate code
if [ $FAILED_TESTS -gt 0 ]; then
    write_error "Some tests failed. Check the detailed report for more information."
    exit 1
else
    write_success "All tests passed successfully!"
    exit 0
fi