#!/bin/bash

# CI/CD Verification Script for DriftMgr
# This script verifies CI/CD pipeline after each testing phase

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "================================================"
echo "       DriftMgr CI/CD Verification Tool"
echo "================================================"

# Function to check current CI status
check_ci_status() {
    echo -e "\n${BLUE}=== Checking Current CI Status ===${NC}"

    # Get latest runs
    echo -e "${YELLOW}Latest CI runs:${NC}"
    gh run list --repo catherinevee/driftmgr --limit 5

    # Count failures
    failures=$(gh run list --repo catherinevee/driftmgr --status failure --limit 10 --json conclusion | grep -c "failure" || true)
    echo -e "\n${RED}Recent failures: $failures${NC}"
}

# Function to run pre-flight checks
preflight_checks() {
    echo -e "\n${BLUE}=== Running Pre-flight Checks ===${NC}"

    # Check if code compiles
    echo -e "${YELLOW}Checking compilation...${NC}"
    if go build ./cmd/driftmgr 2>/dev/null; then
        echo -e "${GREEN}✓ Code compiles successfully${NC}"
    else
        echo -e "${RED}✗ Compilation failed${NC}"
        return 1
    fi

    # Check for obvious test failures
    echo -e "${YELLOW}Running quick test check...${NC}"
    if go test ./internal/drift/comparator/... -timeout 10s 2>/dev/null; then
        echo -e "${GREEN}✓ Sample tests pass${NC}"
    else
        echo -e "${RED}✗ Sample tests fail${NC}"
    fi

    # Check formatting
    echo -e "${YELLOW}Checking formatting...${NC}"
    if [ -z "$(gofmt -l .)" ]; then
        echo -e "${GREEN}✓ Code is properly formatted${NC}"
    else
        echo -e "${RED}✗ Code needs formatting${NC}"
        echo "Run: gofmt -w ."
    fi
}

# Function to create verification PR
create_verification_pr() {
    phase=$1
    coverage_target=$2

    echo -e "\n${BLUE}=== Creating Verification PR for $phase ===${NC}"

    # Create branch
    branch_name="verify/$phase-$(date +%Y%m%d-%H%M%S)"
    git checkout -b "$branch_name"

    # Create PR
    pr_body="## CI/CD Verification for $phase

### Coverage Target: $coverage_target

### Checklist:
- [ ] All tests pass locally
- [ ] CI pipeline completes successfully
- [ ] Coverage meets target
- [ ] Codecov report received
- [ ] No regression in existing tests

### Test Command:
\`\`\`bash
go test ./... -cover -race
\`\`\`

### Verification:
This PR is for CI/CD verification only."

    gh pr create --title "CI/CD Verify: $phase" \
        --body "$pr_body" \
        --repo catherinevee/driftmgr \
        --draft

    echo -e "${GREEN}✓ Draft PR created for verification${NC}"
}

# Function to monitor PR checks
monitor_pr_checks() {
    pr_number=$1

    echo -e "\n${BLUE}=== Monitoring PR #$pr_number ===${NC}"

    # Watch checks
    gh pr checks $pr_number --watch --repo catherinevee/driftmgr

    # Get check status
    status=$(gh pr checks $pr_number --repo catherinevee/driftmgr --json state -q '.[].state' | head -1)

    if [ "$status" = "success" ]; then
        echo -e "${GREEN}✓ All checks passed!${NC}"
        return 0
    else
        echo -e "${RED}✗ Some checks failed${NC}"
        return 1
    fi
}

# Function to verify codecov update
verify_codecov() {
    echo -e "\n${BLUE}=== Verifying Codecov Update ===${NC}"

    # Check for codecov comment on latest PR
    pr_number=$(gh pr list --repo catherinevee/driftmgr --limit 1 --json number -q '.[0].number')

    if [ -n "$pr_number" ]; then
        comments=$(gh pr view $pr_number --repo catherinevee/driftmgr --json comments -q '.comments[].body' | grep -i codecov || true)

        if [ -n "$comments" ]; then
            echo -e "${GREEN}✓ Codecov commented on PR #$pr_number${NC}"
            echo "$comments" | head -5
        else
            echo -e "${YELLOW}⚠ No Codecov comment found on PR #$pr_number${NC}"
        fi
    fi

    # Open Codecov dashboard
    echo -e "\n${YELLOW}Opening Codecov dashboard...${NC}"
    echo "URL: https://app.codecov.io/gh/catherinevee/driftmgr"
}

# Function to run phase verification
run_phase_verification() {
    phase=$1

    echo -e "\n${BLUE}=== Running Phase Verification: $phase ===${NC}"

    case $phase in
        "phase1")
            echo "Verifying Phase 1: Build Fixes"
            packages=("internal/api" "internal/cli" "internal/remediation")
            ;;
        "phase2")
            echo "Verifying Phase 2: API Tests (40% target)"
            packages=("internal/api")
            ;;
        "phase3")
            echo "Verifying Phase 3: CLI & Remediation (35% target)"
            packages=("internal/cli" "internal/remediation")
            ;;
        "phase4")
            echo "Verifying Phase 4: Provider Enhancement"
            packages=("internal/providers/aws" "internal/providers/azure" "internal/providers/gcp")
            ;;
        *)
            echo "Unknown phase: $phase"
            return 1
            ;;
    esac

    # Test each package
    for pkg in "${packages[@]}"; do
        echo -e "\n${YELLOW}Testing $pkg...${NC}"
        if go test ./$pkg/... -cover -timeout 30s; then
            echo -e "${GREEN}✓ $pkg tests pass${NC}"
        else
            echo -e "${RED}✗ $pkg tests fail${NC}"
            return 1
        fi
    done

    echo -e "\n${GREEN}✓ Phase $phase verification complete${NC}"
}

# Function to generate coverage report
generate_coverage_report() {
    echo -e "\n${BLUE}=== Generating Coverage Report ===${NC}"

    # Run tests with coverage
    echo -e "${YELLOW}Running tests with coverage...${NC}"
    go test ./... -coverprofile=coverage_verify.out 2>/dev/null || true

    # Get total coverage
    total=$(go tool cover -func=coverage_verify.out | grep total | awk '{print $3}')
    echo -e "\n${GREEN}Total Coverage: $total${NC}"

    # Show top covered packages
    echo -e "\n${YELLOW}Top covered packages:${NC}"
    go tool cover -func=coverage_verify.out | sort -k3 -rn | head -10

    # Generate HTML report
    go tool cover -html=coverage_verify.out -o coverage_verify.html
    echo -e "\n${GREEN}✓ HTML report saved to coverage_verify.html${NC}"
}

# Main menu
show_menu() {
    echo -e "\n${BLUE}Choose verification option:${NC}"
    echo "1. Check current CI status"
    echo "2. Run pre-flight checks"
    echo "3. Verify Phase 1 (Build Fixes)"
    echo "4. Verify Phase 2 (API Tests)"
    echo "5. Verify Phase 3 (CLI & Remediation)"
    echo "6. Verify Phase 4 (Providers)"
    echo "7. Create verification PR"
    echo "8. Monitor PR checks"
    echo "9. Verify Codecov update"
    echo "10. Generate coverage report"
    echo "11. Run full verification"
    echo "0. Exit"
}

# Full verification
run_full_verification() {
    echo -e "\n${BLUE}=== Running Full CI/CD Verification ===${NC}"

    # Step 1: Pre-flight
    preflight_checks || return 1

    # Step 2: Generate coverage
    generate_coverage_report

    # Step 3: Check CI status
    check_ci_status

    # Step 4: Verify Codecov
    verify_codecov

    echo -e "\n${GREEN}✓ Full verification complete${NC}"
}

# Main loop
while true; do
    show_menu
    read -p "Enter choice: " choice

    case $choice in
        1) check_ci_status ;;
        2) preflight_checks ;;
        3) run_phase_verification "phase1" ;;
        4) run_phase_verification "phase2" ;;
        5) run_phase_verification "phase3" ;;
        6) run_phase_verification "phase4" ;;
        7)
            read -p "Enter phase name: " phase
            read -p "Enter coverage target: " target
            create_verification_pr "$phase" "$target"
            ;;
        8)
            read -p "Enter PR number: " pr_num
            monitor_pr_checks "$pr_num"
            ;;
        9) verify_codecov ;;
        10) generate_coverage_report ;;
        11) run_full_verification ;;
        0)
            echo "Exiting..."
            exit 0
            ;;
        *)
            echo -e "${RED}Invalid option${NC}"
            ;;
    esac
done