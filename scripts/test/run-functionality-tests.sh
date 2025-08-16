#!/bin/bash

# DriftMgr Functionality Test Runner
# This script runs functionality tests to ensure DriftMgr works correctly

set -e

echo "🔧 Starting DriftMgr Functionality Tests"
echo "======================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counter
PASSED=0
FAILED=0
SKIPPED=0

# Function to run a test and report results
run_test() {
    local test_name="$1"
    local test_command="$2"
    local expected_output="$3"
    
    echo -e "\n${BLUE}Testing: ${test_name}${NC}"
    echo "Command: $test_command"
    
    if [ -n "$expected_output" ]; then
        echo "Expected: $expected_output"
    fi
    
    if eval "$test_command" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ PASS: ${test_name}${NC}"
        ((PASSED++))
    else
        echo -e "${RED}✗ FAIL: ${test_name}${NC}"
        ((FAILED++))
    fi
}

# Function to run a test that should produce specific output
run_output_test() {
    local test_name="$1"
    local test_command="$2"
    local expected_pattern="$3"
    
    echo -e "\n${BLUE}Output Test: ${test_name}${NC}"
    echo "Command: $test_command"
    echo "Expected Pattern: $expected_pattern"
    
    output=$(eval "$test_command" 2>&1)
    if echo "$output" | grep -q "$expected_pattern"; then
        echo -e "${GREEN}✓ PASS: ${test_name} - Expected output found${NC}"
        ((PASSED++))
    else
        echo -e "${RED}✗ FAIL: ${test_name} - Expected output not found${NC}"
        echo "Actual output: $output"
        ((FAILED++))
    fi
}

# Function to skip a test
skip_test() {
    local test_name="$1"
    local reason="$2"
    
    echo -e "\n${YELLOW}⏭ SKIP: ${test_name} - ${reason}${NC}"
    ((SKIPPED++))
}

echo -e "\n${YELLOW}1. Core Discovery Engine Tests${NC}"

# AWS Discovery Tests
run_test "AWS Discovery - All Regions" "./bin/driftmgr.exe discover aws all"
run_test "AWS Discovery - US East 1" "./bin/driftmgr.exe discover aws us-east-1"
run_test "AWS Discovery - EU West 1" "./bin/driftmgr.exe discover aws eu-west-1"
run_test "AWS Discovery - AP Southeast 1" "./bin/driftmgr.exe discover aws ap-southeast-1"

# Test resource type filtering (if supported)
run_test "AWS Discovery - Resource Filtering" "./bin/driftmgr.exe discover aws us-east-1 --resource-types ec2_instance,s3_bucket"
run_test "AWS Discovery - Exclude Types" "./bin/driftmgr.exe discover aws us-east-1 --exclude-types iam_user,iam_role"

# Azure Discovery Tests
run_test "Azure Discovery - All Regions" "./bin/driftmgr.exe discover azure all"
run_test "Azure Discovery - East US" "./bin/driftmgr.exe discover azure eastus"
run_test "Azure Discovery - West Europe" "./bin/driftmgr.exe discover azure westeurope"

# GCP Discovery Tests (skip if API not configured)
if [ -n "$GCP_PROJECT_ID" ]; then
    run_test "GCP Discovery - All Regions" "./bin/driftmgr.exe discover gcp all"
    run_test "GCP Discovery - US Central 1" "./bin/driftmgr.exe discover gcp us-central1"
    run_test "GCP Discovery - Europe West 1" "./bin/driftmgr.exe discover gcp europe-west1"
else
    skip_test "GCP Discovery Tests" "GCP_PROJECT_ID not set"
fi

echo -e "\n${YELLOW}2. Drift Analysis Tests${NC}"

# Basic Drift Analysis
run_test "Basic Terraform Analysis" "./bin/driftmgr.exe analyze terraform"
run_test "Terraform Analysis - JSON Output" "./bin/driftmgr.exe analyze terraform --output-format=json"

# Test with specific state file if available
if [ -f "./examples/statefiles/terraform.tfstate" ]; then
    run_test "Terraform Analysis - Custom State File" "./bin/driftmgr.exe analyze terraform --state-file=./examples/statefiles/terraform.tfstate"
else
    skip_test "Terraform Analysis - Custom State File" "State file not found"
fi

# Enhanced Drift Analysis
run_test "Enhanced Analysis - Sensitive Fields" "./bin/driftmgr.exe analyze terraform --sensitive-fields tags.environment,tags.owner"
run_test "Enhanced Analysis - Ignore Fields" "./bin/driftmgr.exe analyze terraform --ignore-fields tags.last-updated,metadata.timestamp"
run_test "Enhanced Analysis - Severity Rules" "./bin/driftmgr.exe analyze terraform --severity-rules production=critical"

# Cross-Provider Analysis
run_test "Multi-Provider Analysis" "./bin/driftmgr.exe analyze terraform --providers aws,azure"
run_test "Provider Comparison" "./bin/driftmgr.exe analyze terraform --compare-providers"

echo -e "\n${YELLOW}3. Remediation Engine Tests${NC}"

# Remediation Command Generation
run_test "Remediation Generation - Example" "./bin/driftmgr.exe remediate example --generate"
run_test "Remediation Generation - Terraform" "./bin/driftmgr.exe remediate terraform --generate --dry-run"
run_test "Remediation Generation - AWS" "./bin/driftmgr.exe remediate aws --generate --resource-types ec2_instance"

# Remediation Execution Tests (with safety checks)
run_test "Remediation Execution - Approval Required" "./bin/driftmgr.exe remediate terraform --execute --approval-required"
run_test "Remediation Execution - Batch Size" "./bin/driftmgr.exe remediate aws --execute --batch-size=5"
run_test "Remediation Execution - Rollback Enabled" "./bin/driftmgr.exe remediate critical --execute --rollback-enabled"

# Remediation History and Rollback
run_test "Remediation History - List" "./bin/driftmgr.exe remediate-history list"
run_test "Remediation History - Show" "./bin/driftmgr.exe remediate-history show test-session"
run_test "Remediation Rollback" "./bin/driftmgr.exe remediate-rollback test-session"

echo -e "\n${YELLOW}4. Visualization and Export Tests${NC}"

# Infrastructure Visualization (fix JSON parsing issues first)
run_test "Diagram Generation - PNG" "./bin/driftmgr.exe diagram aws --format=png"
run_test "Diagram Generation - SVG" "./bin/driftmgr.exe diagram aws --format=svg"
run_test "Diagram Generation - Resource Filter" "./bin/driftmgr.exe diagram aws --include-resources ec2_instance,vpc"

# Data Export Tests
run_test "Data Export - JSON" "./bin/driftmgr.exe export aws --format=json"
run_test "Data Export - CSV" "./bin/driftmgr.exe export aws --format=csv"
run_test "Data Export - YAML" "./bin/driftmgr.exe export aws --format=yaml"
run_test "Data Export - Terraform" "./bin/driftmgr.exe export aws --format=terraform"

echo -e "\n${YELLOW}5. Integration Tests${NC}"

# Terragrunt Integration
run_test "Terragrunt Plan" "./bin/driftmgr.exe terragrunt plan"
run_test "Terragrunt Apply" "./bin/driftmgr.exe terragrunt apply"
run_test "Terragrunt Destroy" "./bin/driftmgr.exe terragrunt destroy"
run_test "Terragrunt Run All" "./bin/driftmgr.exe terragrunt run-all plan"

# CI/CD Pipeline Integration
run_test "CI/CD - Discovery Output" "./bin/driftmgr.exe discover aws all --output-file=discovery.json"
run_test "CI/CD - Analysis Output" "./bin/driftmgr.exe analyze terraform --output-file=drift-report.json"
run_test "CI/CD - Remediation Output" "./bin/driftmgr.exe remediate terraform --auto-approve --output-file=remediation.log"

echo -e "\n${YELLOW}6. Performance and Scalability Tests${NC}"

# Large-Scale Discovery Tests
run_test "Large Scale - High Concurrency" "./bin/driftmgr.exe discover aws all --concurrency=20"
run_test "Large Scale - Extended Timeout" "./bin/driftmgr.exe discover aws all --timeout=30m"
run_test "Large Scale - Large Batch Size" "./bin/driftmgr.exe discover aws all --batch-size=100"

# Memory and Resource Usage Tests
run_test "Resource Limits - Memory" "./bin/driftmgr.exe discover aws all --memory-limit=2GB"
run_test "Resource Limits - CPU" "./bin/driftmgr.exe analyze terraform --cpu-limit=4"

# Concurrent Operation Tests
run_test "Concurrent Operations" "timeout 30s bash -c './bin/driftmgr.exe discover aws us-east-1 & ./bin/driftmgr.exe discover azure eastus & ./bin/driftmgr.exe analyze terraform & wait'"

echo -e "\n${YELLOW}7. Error Handling and Recovery Tests${NC}"

# Network Failure Recovery
run_test "Network Recovery - Retry Attempts" "./bin/driftmgr.exe discover aws all --retry-attempts=5"
run_test "Network Recovery - Retry Delay" "./bin/driftmgr.exe discover aws all --retry-delay=10s"

# Partial Failure Recovery
run_test "Partial Failure - Continue on Error" "./bin/driftmgr.exe discover aws all --continue-on-error"
run_test "Partial Failure - Remediation Continue" "./bin/driftmgr.exe remediate terraform --continue-on-error"

# State Recovery Tests
run_test "State Recovery - Backup State" "./bin/driftmgr.exe analyze terraform --backup-state"
run_test "State Recovery - Validate State" "./bin/driftmgr.exe analyze terraform --validate-state"

echo -e "\n${YELLOW}8. Configuration and Environment Tests${NC}"

# Configuration File Tests
run_test "Config Loading - Default" "./bin/driftmgr.exe --config=./config/config.yaml"
run_test "Config Loading - Custom" "./bin/driftmgr.exe --config=./config/custom-config.yaml"

# Environment Variable Tests
run_test "Environment Variables - Timeout" "DRIFT_CLIENT_TIMEOUT=60s ./bin/driftmgr.exe discover aws us-east-1"
run_test "Environment Variables - Log Level" "DRIFT_LOG_LEVEL=debug ./bin/driftmgr.exe discover aws us-east-1"
run_test "Environment Variables - Cache TTL" "DRIFT_CACHE_TTL=30m ./bin/driftmgr.exe discover aws us-east-1"

# Cross-Platform Compatibility Tests
run_test "Cross-Platform - Windows" "./bin/driftmgr.exe discover aws us-east-1"
run_test "Cross-Platform - Linux" "./bin/driftmgr.exe analyze terraform"

echo -e "\n${YELLOW}9. Specific Issue Resolution Tests${NC}"

# Fix Integration Test Issues
run_test "Cache Integration Fix" "go test ./tests/integration/... -v -run TestCacheIntegration"
run_test "Worker Pool Integration Fix" "go test ./tests/integration/... -v -run TestWorkerPoolIntegration"

# Fix Visualization Issues
run_test "Diagram Generation Fix" "./bin/driftmgr.exe diagram aws --debug"
run_test "Visualization Fix" "./bin/driftmgr.exe visualize aws --debug"

# Fix Command Completion Issues
run_test "Perspective Command Fix" "./bin/driftmgr.exe perspective terraform aws --verbose"
run_test "Remediate Command Fix" "./bin/driftmgr.exe remediate example --generate --verbose"
run_test "Notify Command Fix" "./bin/driftmgr.exe notify test --verbose"
run_test "Export Command Fix" "./bin/driftmgr.exe export aws --verbose"

echo -e "\n${YELLOW}10. Compliance and Governance Tests${NC}"

# Regulatory Compliance
run_test "GDPR Compliance" "./bin/driftmgr.exe discover aws all --gdpr-compliant"
run_test "Data Anonymization" "./bin/driftmgr.exe export aws --anonymize-data"
run_test "SOX Compliance" "./bin/driftmgr.exe analyze terraform --sox-compliant"
run_test "Audit Trail" "./bin/driftmgr.exe remediate terraform --audit-trail"

# Security Policy Enforcement
run_test "Security Scan" "./bin/driftmgr.exe discover aws all --security-scan"
run_test "Policy Check" "./bin/driftmgr.exe analyze terraform --policy-check"
run_test "Policy Enforcement" "./bin/driftmgr.exe remediate terraform --policy-enforcement"

echo -e "\n${YELLOW}11. Monitoring and Alerting Tests${NC}"

# Health Check Tests
run_test "Health Monitoring - Detailed" "./bin/driftmgr.exe health --detailed"
run_test "Health Monitoring - Metrics" "./bin/driftmgr.exe health --metrics"
run_test "API Health Check" "curl -X GET http://localhost:8080/api/v1/health"

# Alerting Tests
run_test "Slack Notifications" "./bin/driftmgr.exe notify drift-detected --channel=slack"
run_test "Email Notifications" "./bin/driftmgr.exe notify critical-drift --channel=email"
run_test "Webhook Notifications" "./bin/driftmgr.exe notify remediation-complete --channel=webhook"

echo -e "\n${YELLOW}12. Documentation and Usability Tests${NC}"

# Help and Documentation
run_test "Main Help" "./bin/driftmgr.exe --help"
run_test "Discover Help" "./bin/driftmgr.exe discover --help"
run_test "Analyze Help" "./bin/driftmgr.exe analyze --help"
run_test "Remediate Help" "./bin/driftmgr.exe remediate --help"

# Interactive Mode Tests
run_test "Interactive Shell" "echo 'exit' | ./bin/driftmgr.exe"
run_test "Command History" "echo 'discover aws us-east-1' | ./bin/driftmgr.exe"

echo -e "\n${YELLOW}13. Automated Test Suite${NC}"

# Unit Test Coverage
run_test "Unit Test Coverage" "go test ./tests/unit/... -v -cover"
run_test "Race Condition Tests" "go test ./tests/unit/... -v -race"
run_test "Benchmark Tests" "go test ./tests/unit/... -v -bench=."

# Integration Test Coverage
run_test "Integration Tests" "go test ./tests/integration/... -v"
run_test "Integration Tests - Timeout" "go test ./tests/integration/... -v -timeout=10m"

# End-to-End Tests
if [ -f "./scripts/test-e2e.sh" ]; then
    run_test "End-to-End Tests" "./scripts/test-e2e.sh"
else
    skip_test "End-to-End Tests" "Script not found"
fi

echo -e "\n${YELLOW}14. Final Functionality Validation${NC}"

# Final validation tests
run_test "Complete Workflow" "./bin/driftmgr.exe discover aws us-east-1 && ./bin/driftmgr.exe analyze terraform && ./bin/driftmgr.exe remediate example --generate"
run_test "Server Functionality" "curl -X GET http://localhost:8080/api/v1/health"
run_test "Client Functionality" "./bin/driftmgr.exe health"

echo -e "\n${YELLOW}========================================${NC}"
echo -e "${YELLOW}Functionality Test Results Summary${NC}"
echo -e "${YELLOW}========================================${NC}"
echo -e "${GREEN}Tests Passed: ${PASSED}${NC}"
echo -e "${RED}Tests Failed: ${FAILED}${NC}"
echo -e "${YELLOW}Tests Skipped: ${SKIPPED}${NC}"
echo -e "${BLUE}Total Tests: $((PASSED + FAILED + SKIPPED))${NC}"

# Calculate success rate
total_run=$((PASSED + FAILED))
if [ $total_run -gt 0 ]; then
    success_rate=$((PASSED * 100 / total_run))
    echo -e "${BLUE}Success Rate: ${success_rate}%${NC}"
fi

if [ $FAILED -eq 0 ]; then
    echo -e "\n${GREEN}🎉 All functionality tests passed! DriftMgr is working correctly.${NC}"
    exit 0
else
    echo -e "\n${RED}⚠️  ${FAILED} functionality test(s) failed. Please review and fix the issues.${NC}"
    exit 1
fi
