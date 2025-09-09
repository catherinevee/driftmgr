#!/bin/bash

# DriftMgr Comprehensive Testing Script
# Tests all driftmgr functions for errors and validates functionality
# This script performs both unit and integration tests

# Default values
VERBOSE=false
STOP_ON_ERROR=false
TEST_CATEGORY="all"
DRIFTMGR_PATH="./driftmgr"

# Test results
PASSED=0
FAILED=0
SKIPPED=0
ERRORS=()

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -s|--stop-on-error)
            STOP_ON_ERROR=true
            shift
            ;;
        -c|--category)
            TEST_CATEGORY="$2"
            shift 2
            ;;
        -p|--path)
            DRIFTMGR_PATH="$2"
            shift 2
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo "Options:"
            echo "  -v, --verbose         Show verbose output"
            echo "  -s, --stop-on-error   Stop on first error"
            echo "  -c, --category        Test category (all, basic, discovery, etc.)"
            echo "  -p, --path            Path to driftmgr executable"
            echo "  -h, --help            Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Test functions
write_test_header() {
    echo -e "\n${CYAN}========================================${NC}"
    echo -e "${CYAN}$1${NC}"
    echo -e "${CYAN}========================================${NC}"
}

write_test_section() {
    echo -e "\n${YELLOW}--- $1 ---${NC}"
}

write_test_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((PASSED++))
}

write_test_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    echo -e "${RED}       Error: $2${NC}"
    ((FAILED++))
    ERRORS+=("$1: $2")
    
    if [ "$STOP_ON_ERROR" = true ]; then
        echo -e "\n${YELLOW}Stopping due to error (stop-on-error flag is set)${NC}"
        show_test_summary
        exit 1
    fi
}

write_test_skip() {
    echo -e "${YELLOW}[SKIP]${NC} $1 - $2"
    ((SKIPPED++))
}

write_test_info() {
    if [ "$VERBOSE" = true ]; then
        echo "       $1"
    fi
}

# Test command execution
test_command() {
    local test_name="$1"
    local command="$2"
    local expected_output="$3"
    local not_expected_output="$4"
    local expected_exit_code="${5:-0}"
    local allow_non_zero="${6:-false}"
    
    echo -n "  Testing: $test_name"
    
    # Execute command
    output=$($command 2>&1)
    exit_code=$?
    
    write_test_info "Exit Code: $exit_code"
    if [ "$VERBOSE" = true ] && [ -n "$output" ]; then
        write_test_info "Output: ${output:0:100}..."
    fi
    
    # Check exit code
    if [ "$allow_non_zero" != "true" ] && [ $exit_code -ne $expected_exit_code ]; then
        write_test_fail "$test_name" "Expected exit code $expected_exit_code, got $exit_code"
        return 1
    fi
    
    # Check expected output
    if [ -n "$expected_output" ]; then
        if ! echo "$output" | grep -q "$expected_output"; then
            write_test_fail "$test_name" "Expected output not found: '$expected_output'"
            return 1
        fi
    fi
    
    # Check not expected output
    if [ -n "$not_expected_output" ]; then
        if echo "$output" | grep -q "$not_expected_output"; then
            write_test_fail "$test_name" "Unexpected output found: '$not_expected_output'"
            return 1
        fi
    fi
    
    write_test_pass "$test_name"
    return 0
}

# Test file operations
test_file_operation() {
    local test_name="$1"
    local file_path="$2"
    local should_exist="$3"
    local expected_content="$4"
    
    echo -n "  Testing: $test_name"
    
    if [ "$should_exist" = "true" ]; then
        if [ ! -f "$file_path" ]; then
            write_test_fail "$test_name" "File should exist but doesn't: $file_path"
            return 1
        fi
    elif [ "$should_exist" = "false" ]; then
        if [ -f "$file_path" ]; then
            write_test_fail "$test_name" "File should not exist but does: $file_path"
            return 1
        fi
    fi
    
    if [ -n "$expected_content" ] && [ -f "$file_path" ]; then
        if ! grep -q "$expected_content" "$file_path"; then
            write_test_fail "$test_name" "Expected content not found in file"
            return 1
        fi
    fi
    
    write_test_pass "$test_name"
    return 0
}

# Test Categories

test_basic_commands() {
    write_test_section "Basic Commands"
    
    # Test help command
    test_command "Help Command" \
        "$DRIFTMGR_PATH --help" \
        "Usage: driftmgr"
    
    test_command "Help Contains Core Commands" \
        "$DRIFTMGR_PATH --help" \
        "Core Commands"
    
    # Test status
    test_command "Status Command" \
        "$DRIFTMGR_PATH status" \
        "DriftMgr System Status"
    
    # Test unknown command
    test_command "Unknown Command Error" \
        "$DRIFTMGR_PATH unknowncommand" \
        "Unknown command" \
        "" \
        0 \
        true
}

test_credential_detection() {
    write_test_section "Credential Detection"
    
    # Test credential status
    test_command "Show Credentials" \
        "$DRIFTMGR_PATH discover --credentials" \
        "Checking credential status"
    
    # Test status shows credentials
    test_command "Status Shows Credentials" \
        "$DRIFTMGR_PATH status" \
        "Cloud Credentials"
    
    # Check for providers
    output=$($DRIFTMGR_PATH status 2>&1)
    has_provider=false
    
    for provider in AWS Azure GCP DigitalOcean; do
        if echo "$output" | grep -q "$provider"; then
            has_provider=true
            write_test_info "Found provider: $provider"
        fi
    done
    
    if [ "$has_provider" = true ]; then
        write_test_pass "At least one provider detected"
    else
        write_test_skip "No cloud providers configured" "Configure at least one provider for full testing"
    fi
}

test_discovery_commands() {
    write_test_section "Discovery Commands"
    
    # Test basic discovery
    test_command "Discovery Help" \
        "$DRIFTMGR_PATH discover --help" \
        "discover cloud resources"
    
    # Test discovery with invalid provider
    test_command "Invalid Provider Error" \
        "$DRIFTMGR_PATH discover --provider invalid" \
        "" \
        "" \
        0 \
        true
    
    # Test auto discovery
    test_command "Auto Discovery Flag" \
        "$DRIFTMGR_PATH discover --auto --format json" \
        "Auto-discovering"
    
    # Test discovery output formats
    for format in json summary table; do
        test_command "Discovery Format: $format" \
            "$DRIFTMGR_PATH discover --provider aws --format $format" \
            "" \
            "" \
            0 \
            true
    done
}

test_drift_detection() {
    write_test_section "Drift Detection"
    
    # Test drift command structure
    test_command "Drift Help" \
        "$DRIFTMGR_PATH drift --help" \
        "drift"
    
    test_command "Drift Detect Help" \
        "$DRIFTMGR_PATH drift detect --help" \
        "detect"
    
    # Test drift detection without state file
    test_command "Drift Detect Without State" \
        "$DRIFTMGR_PATH drift detect --provider aws" \
        "state" \
        "" \
        0 \
        true
    
    # Test smart defaults
    test_command "Smart Defaults Flag" \
        "$DRIFTMGR_PATH drift detect --smart-defaults --help" \
        "Smart Defaults"
}

test_state_management() {
    write_test_section "State Management"
    
    # Test state commands
    test_command "State Help" \
        "$DRIFTMGR_PATH state --help" \
        "state"
    
    test_command "State Discover" \
        "$DRIFTMGR_PATH state discover" \
        "" \
        "" \
        0 \
        true
    
    # Test state visualization
    test_command "State Visualize Help" \
        "$DRIFTMGR_PATH state visualize --help" \
        "visualize"
}

test_account_management() {
    write_test_section "Account Management"
    
    # Test account listing
    test_command "List Accounts" \
        "$DRIFTMGR_PATH accounts" \
        "" \
        "" \
        0 \
        true
    
    # Test use command
    test_command "Use Command Help" \
        "$DRIFTMGR_PATH use --help" \
        "Select"
    
    # Test use with all flag
    test_command "Use All Flag" \
        "$DRIFTMGR_PATH use --all" \
        "Available"
}

test_export_import() {
    write_test_section "Export/Import Commands"
    
    test_export_file="test_export_$$.json"
    
    # Test export
    test_command "Export Help" \
        "$DRIFTMGR_PATH export --help" \
        "Export"
    
    # Test export to file
    test_command "Export to JSON" \
        "$DRIFTMGR_PATH export --format json --output $test_export_file" \
        "" \
        "" \
        0 \
        true
    
    # Clean up test file
    if [ -f "$test_export_file" ]; then
        rm -f "$test_export_file"
        write_test_info "Cleaned up test export file"
    fi
    
    # Test import
    test_command "Import Help" \
        "$DRIFTMGR_PATH import --help" \
        "Import"
}

test_verify_command() {
    write_test_section "Verify Command"
    
    test_command "Verify Help" \
        "$DRIFTMGR_PATH verify --help" \
        "Verify"
    
    test_command "Verify Execution" \
        "$DRIFTMGR_PATH verify" \
        "" \
        "" \
        0 \
        true
}

test_server_commands() {
    write_test_section "Server Commands"
    
    test_command "Serve Help" \
        "$DRIFTMGR_PATH serve --help" \
        "Start"
    
    test_command "Serve Web Help" \
        "$DRIFTMGR_PATH serve web --help" \
        "web"
}

test_delete_command() {
    write_test_section "Delete Command"
    
    test_command "Delete Help" \
        "$DRIFTMGR_PATH delete --help" \
        "Delete"
    
    # Test delete with dry-run (safe)
    test_command "Delete Dry Run" \
        "$DRIFTMGR_PATH delete --resource-id test-123 --dry-run" \
        "" \
        "" \
        0 \
        true
}

test_error_handling() {
    write_test_section "Error Handling"
    
    # Test invalid flags
    test_command "Invalid Flag Error" \
        "$DRIFTMGR_PATH --invalidflag" \
        "Unknown" \
        "" \
        0 \
        true
    
    # Test missing required arguments
    test_command "Missing Required Args" \
        "$DRIFTMGR_PATH discover --provider" \
        "" \
        "" \
        0 \
        true
    
    # Test invalid file paths
    test_command "Invalid File Path" \
        "$DRIFTMGR_PATH export --output /invalid:/path/file.json" \
        "" \
        "" \
        0 \
        true
}

test_color_and_progress() {
    write_test_section "Color and Progress Features"
    
    # Test with NO_COLOR
    NO_COLOR=1 test_command "NO_COLOR Environment" \
        "$DRIFTMGR_PATH status" \
        "" \
        "\033\[31m"  # Should not contain ANSI color codes
    
    # Test with FORCE_COLOR
    FORCE_COLOR=1 test_command "FORCE_COLOR Environment" \
        "$DRIFTMGR_PATH status"
}

test_configuration_files() {
    write_test_section "Configuration Files"
    
    # Check for config files
    for config_file in \
        "configs/config.yaml" \
        "configs/smart-defaults.yaml" \
        "configs/driftmgr.yaml"; do
        test_file_operation "Config File: $config_file" \
            "$config_file" \
            "true"
    done
}

test_build_artifacts() {
    write_test_section "Build Artifacts"
    
    # Check main executable exists
    test_file_operation "Main Executable" \
        "$DRIFTMGR_PATH" \
        "true"
    
    # Test executable runs
    test_command "Executable Runs" \
        "$DRIFTMGR_PATH" \
        "driftmgr"
}

test_edge_cases() {
    write_test_section "Edge Cases"
    
    # Test with very long arguments
    long_arg=$(printf 'a%.0s' {1..1000})
    test_command "Very Long Argument" \
        "$DRIFTMGR_PATH discover --provider $long_arg" \
        "" \
        "" \
        0 \
        true
    
    # Test special characters in arguments
    test_command "Special Characters" \
        "$DRIFTMGR_PATH export --output \"test file with spaces.json\"" \
        "" \
        "" \
        0 \
        true
    
    # Test Unicode in arguments
    test_command "Unicode Characters" \
        "$DRIFTMGR_PATH export --output test_😀.json" \
        "" \
        "" \
        0 \
        true
}

test_integration() {
    write_test_section "Integration Tests"
    
    # Test command chaining
    echo -n "  Testing: Command Chaining"
    
    status_output=$($DRIFTMGR_PATH status 2>&1)
    if echo "$status_output" | grep -q "configured"; then
        discover_output=$($DRIFTMGR_PATH discover --auto 2>&1)
        if [ -n "$discover_output" ]; then
            write_test_pass "Command Chaining"
        else
            write_test_fail "Command Chaining" "Discovery after status failed"
        fi
    else
        write_test_skip "Command Chaining" "No providers configured"
    fi
    
    # Test multiple format outputs
    all_formats_work=true
    for format in json summary table; do
        output=$($DRIFTMGR_PATH discover --format $format 2>&1)
        if [ $? -ne 0 ] && ! echo "$output" | grep -q "No credentials"; then
            all_formats_work=false
            write_test_info "Format $format failed"
        fi
    done
    
    if [ "$all_formats_work" = true ]; then
        write_test_pass "Multiple Output Formats"
    else
        write_test_fail "Multiple Output Formats" "Some formats failed"
    fi
}

test_performance() {
    write_test_section "Performance Tests"
    
    # Test help command performance
    echo -n "  Testing: Help Command Performance"
    start_time=$(date +%s%N)
    $DRIFTMGR_PATH --help > /dev/null 2>&1
    end_time=$(date +%s%N)
    elapsed=$((($end_time - $start_time) / 1000000))
    
    if [ $elapsed -lt 1000 ]; then
        write_test_pass "Help Command Performance (<1s)"
        write_test_info "Completed in ${elapsed}ms"
    else
        write_test_fail "Help Command Performance" "Took ${elapsed}ms (>1s)"
    fi
    
    # Test status command performance
    echo -n "  Testing: Status Command Performance"
    start_time=$(date +%s%N)
    $DRIFTMGR_PATH status > /dev/null 2>&1
    end_time=$(date +%s%N)
    elapsed=$((($end_time - $start_time) / 1000000))
    
    if [ $elapsed -lt 5000 ]; then
        write_test_pass "Status Command Performance (<5s)"
        write_test_info "Completed in ${elapsed}ms"
    else
        write_test_fail "Status Command Performance" "Took ${elapsed}ms (>5s)"
    fi
}

show_test_summary() {
    write_test_header "Test Summary"
    
    total=$((PASSED + FAILED + SKIPPED))
    if [ $total -gt 0 ]; then
        pass_rate=$((PASSED * 100 / total))
    else
        pass_rate=0
    fi
    
    echo "Total Tests: $total"
    echo -e "${GREEN}Passed: $PASSED${NC}"
    if [ $FAILED -eq 0 ]; then
        echo -e "${GREEN}Failed: $FAILED${NC}"
    else
        echo -e "${RED}Failed: $FAILED${NC}"
    fi
    echo -e "${YELLOW}Skipped: $SKIPPED${NC}"
    
    if [ $pass_rate -ge 80 ]; then
        echo -e "${GREEN}Pass Rate: $pass_rate%${NC}"
    elif [ $pass_rate -ge 60 ]; then
        echo -e "${YELLOW}Pass Rate: $pass_rate%${NC}"
    else
        echo -e "${RED}Pass Rate: $pass_rate%${NC}"
    fi
    
    if [ $FAILED -gt 0 ]; then
        echo -e "\n${RED}Failed Tests:${NC}"
        for error in "${ERRORS[@]}"; do
            echo -e "${RED}  - $error${NC}"
        done
    fi
    
    # Exit with appropriate code
    if [ $FAILED -gt 0 ]; then
        exit 1
    fi
}

# Main execution
run_all_tests() {
    write_test_header "DriftMgr Comprehensive Test Suite"
    echo "Testing: $DRIFTMGR_PATH"
    echo "Verbose: $VERBOSE"
    echo "StopOnError: $STOP_ON_ERROR"
    echo "Category: $TEST_CATEGORY"
    
    # Check if driftmgr exists
    if [ ! -f "$DRIFTMGR_PATH" ]; then
        echo -e "\n${RED}ERROR: DriftMgr executable not found at $DRIFTMGR_PATH${NC}"
        echo -e "${YELLOW}Please build the project first: go build -o driftmgr ./cmd/driftmgr${NC}"
        exit 1
    fi
    
    # Make executable if needed
    chmod +x "$DRIFTMGR_PATH" 2>/dev/null
    
    # Run test categories
    case $TEST_CATEGORY in
        all)
            test_basic_commands
            test_credential_detection
            test_discovery_commands
            test_drift_detection
            test_state_management
            test_account_management
            test_export_import
            test_verify_command
            test_server_commands
            test_delete_command
            test_error_handling
            test_color_and_progress
            test_configuration_files
            test_build_artifacts
            test_edge_cases
            test_integration
            test_performance
            ;;
        basic)
            test_basic_commands
            ;;
        credentials)
            test_credential_detection
            ;;
        discovery)
            test_discovery_commands
            ;;
        drift)
            test_drift_detection
            ;;
        state)
            test_state_management
            ;;
        accounts)
            test_account_management
            ;;
        export)
            test_export_import
            ;;
        verify)
            test_verify_command
            ;;
        server)
            test_server_commands
            ;;
        delete)
            test_delete_command
            ;;
        errors)
            test_error_handling
            ;;
        color)
            test_color_and_progress
            ;;
        config)
            test_configuration_files
            ;;
        build)
            test_build_artifacts
            ;;
        edge)
            test_edge_cases
            ;;
        integration)
            test_integration
            ;;
        performance)
            test_performance
            ;;
        *)
            echo -e "${RED}Invalid test category: $TEST_CATEGORY${NC}"
            echo "Available categories: all, basic, credentials, discovery, drift, state, accounts, export, verify, server, delete, errors, color, config, build, edge, integration, performance"
            exit 1
            ;;
    esac
    
    show_test_summary
}

# Run the tests
run_all_tests