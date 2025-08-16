#!/bin/bash
# DriftMgr Development Tools Script
# This script provides various development utilities for DriftMgr

set -e

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_NAME="DriftMgr"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Logging functions
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

# Error handling
handle_error() {
    log_error "Operation failed at line $1"
    exit 1
}

trap 'handle_error $LINENO' ERR

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    # Check if we're in the right directory
    if [ ! -f "go.mod" ]; then
        log_error "go.mod not found. Please run this script from the project root."
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Lint code
lint_code() {
    log_info "Running code linting..."
    
    if command -v golangci-lint &> /dev/null; then
        golangci-lint run
        log_success "Linting completed"
    else
        log_warning "golangci-lint not found, installing..."
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2
        golangci-lint run
        log_success "Linting completed"
    fi
}

# Format code
format_code() {
    log_info "Formatting code..."
    go fmt ./...
    log_success "Code formatting completed"
}

# Run tests
run_tests() {
    local coverage=false
    local verbose=false
    local race=false
    
    # Parse test options
    while [[ $# -gt 0 ]]; do
        case $1 in
            --coverage)
                coverage=true
                shift
                ;;
            --verbose)
                verbose=true
                shift
                ;;
            --race)
                race=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    log_info "Running tests..."
    
    local test_flags=""
    if [ "$verbose" = true ]; then
        test_flags="$test_flags -v"
    fi
    if [ "$race" = true ]; then
        test_flags="$test_flags -race"
    fi
    if [ "$coverage" = true ]; then
        test_flags="$test_flags -coverprofile=coverage.out"
    fi
    
    go test $test_flags ./...
    
    if [ "$coverage" = true ]; then
        log_info "Generating coverage report..."
        go tool cover -html=coverage.out -o coverage.html
        log_success "Coverage report generated: coverage.html"
    fi
    
    log_success "Tests completed"
}

# Run benchmarks
run_benchmarks() {
    log_info "Running benchmarks..."
    go test -bench=. -benchmem ./...
    log_success "Benchmarks completed"
}

# Generate documentation
generate_docs() {
    log_info "Generating documentation..."
    
    # Generate Go documentation
    if command -v godoc &> /dev/null; then
        log_info "Starting godoc server on http://localhost:6060"
        godoc -http=:6060 &
        local godoc_pid=$!
        log_success "Go documentation server started (PID: $godoc_pid)"
        log_info "Visit http://localhost:6060/pkg/ to view documentation"
        log_info "Press Ctrl+C to stop the server"
        wait $godoc_pid
    else
        log_warning "godoc not found, installing..."
        go install golang.org/x/tools/cmd/godoc@latest
        generate_docs
    fi
}

# Clean build artifacts
clean_artifacts() {
    log_info "Cleaning build artifacts..."
    
    # Remove binaries
    rm -f driftmgr-server driftmgr-client
    rm -f driftmgr-server.exe driftmgr-client.exe
    rm -f driftmgr-server-* driftmgr-client-*
    
    # Remove test artifacts
    rm -f coverage.out coverage.html
    rm -f *.prof
    
    # Remove backup files
    rm -f *.backup
    
    # Remove temporary files
    rm -rf tmp/ temp/
    
    log_success "Clean completed"
}

# Check for security issues
security_scan() {
    log_info "Running security scan..."
    
    if command -v gosec &> /dev/null; then
        gosec ./...
        log_success "Security scan completed"
    else
        log_warning "gosec not found, installing..."
        go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
        gosec ./...
        log_success "Security scan completed"
    fi
}

# Update dependencies
update_dependencies() {
    log_info "Updating dependencies..."
    
    # Update Go modules
    go get -u ./...
    go mod tidy
    
    log_success "Dependencies updated"
}

# Check for outdated dependencies
check_outdated_deps() {
    log_info "Checking for outdated dependencies..."
    
    if command -v go-mod-outdated &> /dev/null; then
        go list -u -m all | go-mod-outdated -update -direct
    else
        log_warning "go-mod-outdated not found, installing..."
        go install github.com/psampaz/go-mod-outdated@latest
        go list -u -m all | go-mod-outdated -update -direct
    fi
    
    log_success "Dependency check completed"
}

# Run development server with hot reload
dev_server() {
    log_info "Starting development server with hot reload..."
    
    if command -v air &> /dev/null; then
        # Create air configuration if it doesn't exist
        if [ ! -f ".air.toml" ]; then
            cat > .air.toml << EOF
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
  args_bin = []
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ./cmd/driftmgr-server"
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "testdata"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html"]
  include_file = []
  kill_delay = "0s"
  log = "build-errors.log"
  poll = false
  poll_interval = 0
  rerun = false
  rerun_delay = 500
  send_interrupt = false
  stop_on_root = false

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  main_only = false
  time = false

[misc]
  clean_on_exit = false
EOF
        fi
        
        air
    else
        log_warning "air not found, installing..."
        go install github.com/cosmtrek/air@latest
        dev_server
    fi
}

# Run development client
dev_client() {
    log_info "Starting development client..."
    go run ./cmd/driftmgr-client
}

# Show project status
show_status() {
    log_info "Project Status:"
    echo "  Project: $PROJECT_NAME"
    echo "  Version: $VERSION"
    echo "  Go version: $(go version)"
    echo "  Git branch: $(git branch --show-current 2>/dev/null || echo 'unknown')"
    echo "  Git commit: $(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
    echo "  Build artifacts:"
    
    local artifacts_found=false
    for binary in driftmgr-server driftmgr-client driftmgr-server.exe driftmgr-client.exe; do
        if [ -f "$binary" ]; then
            echo "    ✓ $binary"
            artifacts_found=true
        fi
    done
    
    if [ "$artifacts_found" = false ]; then
        echo "    No build artifacts found"
    fi
}

# Show usage
show_usage() {
    echo "Usage: $0 [COMMAND] [OPTIONS]"
    echo ""
    echo "Commands:"
    echo "  lint                    Run code linting"
    echo "  format                  Format code"
    echo "  test [OPTIONS]          Run tests"
    echo "  bench                   Run benchmarks"
    echo "  docs                    Generate documentation"
    echo "  clean                   Clean build artifacts"
    echo "  security                Run security scan"
    echo "  deps                    Update dependencies"
    echo "  deps-check              Check for outdated dependencies"
    echo "  dev-server              Start development server with hot reload"
    echo "  dev-client              Start development client"
    echo "  status                  Show project status"
    echo "  help                    Show this help message"
    echo ""
    echo "Test Options:"
    echo "  --coverage              Generate coverage report"
    echo "  --verbose               Verbose output"
    echo "  --race                  Run with race detection"
    echo ""
    echo "Examples:"
    echo "  $0 lint                 # Run linting"
    echo "  $0 test --coverage      # Run tests with coverage"
    echo "  $0 dev-server           # Start development server"
    echo "  $0 clean                # Clean artifacts"
}

# Main function
main() {
    local command="$1"
    shift
    
    # Check prerequisites
    check_prerequisites
    
    case $command in
        "lint")
            lint_code
            ;;
        "format")
            format_code
            ;;
        "test")
            run_tests "$@"
            ;;
        "bench")
            run_benchmarks
            ;;
        "docs")
            generate_docs
            ;;
        "clean")
            clean_artifacts
            ;;
        "security")
            security_scan
            ;;
        "deps")
            update_dependencies
            ;;
        "deps-check")
            check_outdated_deps
            ;;
        "dev-server")
            dev_server
            ;;
        "dev-client")
            dev_client
            ;;
        "status")
            show_status
            ;;
        "help"|"--help"|"-h")
            show_usage
            ;;
        "")
            show_usage
            exit 1
            ;;
        *)
            log_error "Unknown command: $command"
            show_usage
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"
