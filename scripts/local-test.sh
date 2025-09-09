#!/bin/bash
#
# Local testing script for DriftMgr
# Provides a complete testing environment with mocked cloud services
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ROOT=$(dirname "$(dirname "$(realpath "$0")")")
COMPOSE_FILE="docker-compose.test.yml"
TEST_TIMEOUT=300
COVERAGE_THRESHOLD=70

# Functions
print_status() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

cleanup() {
    print_info "Cleaning up test environment..."
    docker-compose -f "$COMPOSE_FILE" down -v
    rm -rf test-results coverage-reports
}

# Parse arguments
TEST_TYPE="all"
KEEP_RUNNING=false
VERBOSE=false
COVERAGE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --type)
            TEST_TYPE="$2"
            shift 2
            ;;
        --keep-running)
            KEEP_RUNNING=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --coverage)
            COVERAGE=true
            shift
            ;;
        --help)
            echo "Usage: $0 [options]"
            echo "Options:"
            echo "  --type <type>     Test type: unit, integration, e2e, all (default: all)"
            echo "  --keep-running    Keep services running after tests"
            echo "  --verbose         Verbose output"
            echo "  --coverage        Generate coverage report"
            echo "  --help           Show this help message"
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Trap cleanup on exit
if [ "$KEEP_RUNNING" = false ]; then
    trap cleanup EXIT
fi

# Main execution
echo "================================================"
echo "          DriftMgr Local Testing"
echo "================================================"
echo ""

# Check prerequisites
print_info "Checking prerequisites..."

if ! command -v docker &> /dev/null; then
    print_error "Docker is not installed"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    print_error "Docker Compose is not installed"
    exit 1
fi

if ! command -v go &> /dev/null; then
    print_error "Go is not installed"
    exit 1
fi

print_status "Prerequisites satisfied"
echo ""

# Start test services
print_info "Starting test services..."

cd "$PROJECT_ROOT"

# Start services
docker-compose -f "$COMPOSE_FILE" up -d

# Wait for services to be healthy
print_info "Waiting for services to be ready..."

services=("localstack" "azurite" "minio" "postgres-test" "redis-test")
max_attempts=30
attempt=0

while [ $attempt -lt $max_attempts ]; do
    all_healthy=true
    
    for service in "${services[@]}"; do
        if ! docker-compose -f "$COMPOSE_FILE" ps | grep "$service" | grep -q "Up"; then
            all_healthy=false
            break
        fi
    done
    
    if [ "$all_healthy" = true ]; then
        print_status "All services are ready"
        break
    fi
    
    attempt=$((attempt + 1))
    if [ $attempt -eq $max_attempts ]; then
        print_error "Services failed to start within timeout"
        exit 1
    fi
    
    sleep 2
done

echo ""

# Initialize test data
print_info "Initializing test data..."

# Create S3 buckets in LocalStack
docker exec driftmgr-test-localstack aws --endpoint-url=http://localhost:4566 \
    s3 mb s3://terraform-states --region us-east-1 2>/dev/null || true

docker exec driftmgr-test-localstack aws --endpoint-url=http://localhost:4566 \
    s3 mb s3://test-bucket --region us-east-1 2>/dev/null || true

# Create DynamoDB table for state locking
docker exec driftmgr-test-localstack aws --endpoint-url=http://localhost:4566 \
    dynamodb create-table \
    --table-name terraform-state-lock \
    --attribute-definitions AttributeName=LockID,AttributeType=S \
    --key-schema AttributeName=LockID,KeyType=HASH \
    --billing-mode PAY_PER_REQUEST \
    --region us-east-1 2>/dev/null || true

# Create test resources in LocalStack
docker exec driftmgr-test-localstack aws --endpoint-url=http://localhost:4566 \
    ec2 create-security-group \
    --group-name test-sg \
    --description "Test security group" \
    --region us-east-1 2>/dev/null || true

print_status "Test data initialized"
echo ""

# Run tests
print_info "Running tests..."

# Set environment variables
export AWS_ENDPOINT_URL=http://localhost:4566
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-east-1
export AZURE_STORAGE_ENDPOINT=http://localhost:10000
export AZURE_STORAGE_ACCOUNT=devstoreaccount1
export AZURE_STORAGE_KEY=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==
export S3_ENDPOINT=http://localhost:9000
export MINIO_ACCESS_KEY=minioadmin
export MINIO_SECRET_KEY=minioadmin
export DATABASE_URL=postgres://test:test@localhost:5433/driftmgr_test?sslmode=disable
export REDIS_URL=redis://localhost:6380/0
export INTEGRATION_TESTS=true

# Create test results directory
mkdir -p test-results coverage-reports

# Run tests based on type
case $TEST_TYPE in
    unit)
        print_info "Running unit tests..."
        if [ "$VERBOSE" = true ]; then
            go test -v -race -timeout "$TEST_TIMEOUT"s ./... | tee test-results/unit.txt
        else
            go test -race -timeout "$TEST_TIMEOUT"s ./...
        fi
        ;;
    
    integration)
        print_info "Running integration tests..."
        if [ "$VERBOSE" = true ]; then
            go test -v -tags=integration -timeout "$TEST_TIMEOUT"s ./tests/integration/... | tee test-results/integration.txt
        else
            go test -tags=integration -timeout "$TEST_TIMEOUT"s ./tests/integration/...
        fi
        ;;
    
    e2e)
        print_info "Running end-to-end tests..."
        if [ "$VERBOSE" = true ]; then
            go test -v -tags=e2e -timeout "$TEST_TIMEOUT"s ./tests/e2e/... | tee test-results/e2e.txt
        else
            go test -tags=e2e -timeout "$TEST_TIMEOUT"s ./tests/e2e/...
        fi
        ;;
    
    all)
        print_info "Running all tests..."
        
        # Unit tests
        echo "Running unit tests..."
        go test -race -timeout "$TEST_TIMEOUT"s ./... > test-results/unit.txt 2>&1 || true
        
        # Integration tests
        echo "Running integration tests..."
        go test -tags=integration -timeout "$TEST_TIMEOUT"s ./tests/integration/... > test-results/integration.txt 2>&1 || true
        
        # E2E tests
        echo "Running E2E tests..."
        go test -tags=e2e -timeout "$TEST_TIMEOUT"s ./tests/e2e/... > test-results/e2e.txt 2>&1 || true
        ;;
    
    *)
        print_error "Unknown test type: $TEST_TYPE"
        exit 1
        ;;
esac

print_status "Tests completed"
echo ""

# Generate coverage report if requested
if [ "$COVERAGE" = true ]; then
    print_info "Generating coverage report..."
    
    go test -coverprofile=coverage-reports/coverage.out -covermode=atomic ./...
    go tool cover -html=coverage-reports/coverage.out -o coverage-reports/coverage.html
    
    # Calculate coverage percentage
    COVERAGE_PCT=$(go tool cover -func=coverage-reports/coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    
    print_status "Coverage: ${COVERAGE_PCT}%"
    
    # Check threshold
    if (( $(echo "$COVERAGE_PCT < $COVERAGE_THRESHOLD" | bc -l) )); then
        print_warning "Coverage ${COVERAGE_PCT}% is below threshold ${COVERAGE_THRESHOLD}%"
    else
        print_status "Coverage meets threshold"
    fi
    
    echo "Coverage report: coverage-reports/coverage.html"
    echo ""
fi

# Run benchmarks
if [ "$TEST_TYPE" = "all" ]; then
    print_info "Running benchmarks..."
    go test -bench=. -benchmem -run=^$$ ./... > test-results/benchmarks.txt 2>&1 || true
    print_status "Benchmarks completed"
    echo ""
fi

# Display test results
print_info "Test Results Summary:"
echo "------------------------"

if [ -f test-results/unit.txt ]; then
    UNIT_PASS=$(grep -c "PASS" test-results/unit.txt || true)
    UNIT_FAIL=$(grep -c "FAIL" test-results/unit.txt || true)
    echo "Unit Tests: $UNIT_PASS passed, $UNIT_FAIL failed"
fi

if [ -f test-results/integration.txt ]; then
    INT_PASS=$(grep -c "PASS" test-results/integration.txt || true)
    INT_FAIL=$(grep -c "FAIL" test-results/integration.txt || true)
    echo "Integration Tests: $INT_PASS passed, $INT_FAIL failed"
fi

if [ -f test-results/e2e.txt ]; then
    E2E_PASS=$(grep -c "PASS" test-results/e2e.txt || true)
    E2E_FAIL=$(grep -c "FAIL" test-results/e2e.txt || true)
    echo "E2E Tests: $E2E_PASS passed, $E2E_FAIL failed"
fi

echo ""

# Service URLs
if [ "$KEEP_RUNNING" = true ]; then
    echo "================================================"
    echo "Test services are still running:"
    echo "------------------------------------------------"
    echo "LocalStack (AWS):    http://localhost:4566"
    echo "Azurite (Azure):     http://localhost:10000"
    echo "MinIO (S3):          http://localhost:9001"
    echo "PostgreSQL:          localhost:5433"
    echo "Redis:               localhost:6380"
    echo "MailHog:             http://localhost:8025"
    echo "Webhook Receiver:    http://localhost:8091"
    echo ""
    echo "To stop services: docker-compose -f $COMPOSE_FILE down"
    echo "================================================"
fi

print_status "Testing complete!"