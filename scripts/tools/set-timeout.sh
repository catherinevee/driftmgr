#!/bin/bash

# DriftMgr Timeout Configuration Script
# This script helps set appropriate timeout values for different scenarios

# Default scenario
SCENARIO="large"
CLIENT_TIMEOUT=""
DISCOVERY_TIMEOUT=""

# Function to show usage
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -s, --scenario SCENARIO    Set scenario (dev|small|large|multi-cloud|custom)"
    echo "  -c, --client-timeout TIME  Set client timeout (e.g., 5m, 10m)"
    echo "  -d, --discovery-timeout TIME Set discovery timeout (e.g., 10m, 15m)"
    echo "  -h, --help                 Show this help message"
    echo ""
    echo "Scenarios:"
    echo "  dev         - Development and testing with small infrastructure"
    echo "  small       - Production with small infrastructure (< 100 resources)"
    echo "  large       - Production with large infrastructure (> 100 resources)"
    echo "  multi-cloud - Multi-cloud or all regions discovery"
    echo "  custom      - Use custom timeout values"
    echo ""
    echo "Examples:"
    echo "  $0 -s large"
    echo "  $0 -s custom -c 5m -d 10m"
    echo "  $0 --scenario multi-cloud"
}

# Function to set timeout variables
set_timeout_variables() {
    local client_timeout="$1"
    local discovery_timeout="$2"
    
    if [ -n "$client_timeout" ]; then
        export DRIFT_CLIENT_TIMEOUT="$client_timeout"
        echo -e "\033[32mSet DRIFT_CLIENT_TIMEOUT = $client_timeout\033[0m"
    fi
    
    if [ -n "$discovery_timeout" ]; then
        export DRIFT_DISCOVERY_TIMEOUT="$discovery_timeout"
        echo -e "\033[32mSet DRIFT_DISCOVERY_TIMEOUT = $discovery_timeout\033[0m"
    fi
}

# Function to show current settings
show_current_settings() {
    echo -e "\033[33mCurrent timeout settings:\033[0m"
    echo -e "  DRIFT_CLIENT_TIMEOUT: ${DRIFT_CLIENT_TIMEOUT:-'not set'}"
    echo -e "  DRIFT_DISCOVERY_TIMEOUT: ${DRIFT_DISCOVERY_TIMEOUT:-'not set'}"
    echo ""
}

# Function to show recommended settings
show_recommended_settings() {
    local scenario="$1"
    
    echo -e "\033[33mRecommended settings for '$scenario' scenario:\033[0m"
    
    case "$scenario" in
        "dev")
            echo -e "  DRIFT_CLIENT_TIMEOUT: 1m"
            echo -e "  DRIFT_DISCOVERY_TIMEOUT: 2m"
            echo -e "  \033[90mUse case: Development and testing with small infrastructure\033[0m"
            ;;
        "small")
            echo -e "  DRIFT_CLIENT_TIMEOUT: 2m"
            echo -e "  DRIFT_DISCOVERY_TIMEOUT: 5m"
            echo -e "  \033[90mUse case: Production with small infrastructure (< 100 resources)\033[0m"
            ;;
        "large")
            echo -e "  DRIFT_CLIENT_TIMEOUT: 5m"
            echo -e "  DRIFT_DISCOVERY_TIMEOUT: 10m"
            echo -e "  \033[90mUse case: Production with large infrastructure (> 100 resources)\033[0m"
            ;;
        "multi-cloud")
            echo -e "  DRIFT_CLIENT_TIMEOUT: 5m"
            echo -e "  DRIFT_DISCOVERY_TIMEOUT: 15m"
            echo -e "  \033[90mUse case: Multi-cloud or all regions discovery\033[0m"
            ;;
        "custom")
            echo -e "  Use -c and -d parameters to set custom values"
            ;;
    esac
    echo ""
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -s|--scenario)
            SCENARIO="$2"
            shift 2
            ;;
        -c|--client-timeout)
            CLIENT_TIMEOUT="$2"
            shift 2
            ;;
        -d|--discovery-timeout)
            DISCOVERY_TIMEOUT="$2"
            shift 2
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Validate scenario
case "$SCENARIO" in
    dev|small|large|multi-cloud|custom)
        ;;
    *)
        echo "Error: Invalid scenario '$SCENARIO'"
        show_usage
        exit 1
        ;;
esac

echo -e "\033[36mDriftMgr Timeout Configuration Script\033[0m"
echo -e "\033[36m=====================================\033[0m"
echo ""

# Show current settings
show_current_settings

# Show recommended settings for the scenario
show_recommended_settings "$SCENARIO"

# Apply settings based on scenario
case "$SCENARIO" in
    "dev")
        set_timeout_variables "1m" "2m"
        ;;
    "small")
        set_timeout_variables "2m" "5m"
        ;;
    "large")
        set_timeout_variables "5m" "10m"
        ;;
    "multi-cloud")
        set_timeout_variables "5m" "15m"
        ;;
    "custom")
        if [ -n "$CLIENT_TIMEOUT" ] || [ -n "$DISCOVERY_TIMEOUT" ]; then
            set_timeout_variables "$CLIENT_TIMEOUT" "$DISCOVERY_TIMEOUT"
        else
            echo -e "\033[31mFor custom timeouts, use -c and -d parameters\033[0m"
            echo -e "\033[90mExample: $0 -s custom -c 3m -d 8m\033[0m"
            exit 1
        fi
        ;;
esac

echo -e "\033[32mTimeout configuration completed!\033[0m"
echo ""
echo -e "\033[36mYou can now run the driftmgr client:\033[0m"
echo -e "  ./bin/driftmgr-client"
echo ""
echo -e "\033[33mNote: These settings are only valid for the current shell session.\033[0m"
echo -e "\033[33mTo make them permanent, add them to your shell profile or use system environment variables.\033[0m"
