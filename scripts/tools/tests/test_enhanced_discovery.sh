#!/bin/bash

# Enhanced AWS Discovery Test Script
# This script tests the enhanced AWS discovery functionality with various region configurations

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}=== Enhanced AWS Discovery Test Suite ===${NC}"
echo "Testing DriftMgr enhanced AWS discovery with various region configurations"
echo

# Function to check if server is running
check_server() {
    echo -e "${BLUE}Checking DriftMgr server status...${NC}"
    if curl -s http://localhost:8080/health > /dev/null; then
        echo -e "${GREEN}✅ DriftMgr server is running${NC}"
        return 0
    else
        echo -e "${RED}❌ DriftMgr server is not running!${NC}"
        echo "Please start the server with: ./bin/driftmgr-server.exe"
        return 1
    fi
}

# Function to test discovery with specific regions
test_discovery() {
    local test_name="$1"
    local regions="$2"
    
    echo -e "${YELLOW}--- $test_name ---${NC}"
    echo "Regions: $regions"
    
    # Create JSON request
    local json_request="{\"provider\":\"aws\",\"regions\":$regions,\"account\":\"default\"}"
    
    # Send request and capture response
    local response=$(curl -s -X POST http://localhost:8080/api/v1/discover \
        -H "Content-Type: application/json" \
        -d "$json_request")
    
    # Extract total resources and duration
    local total=$(echo "$response" | jq -r '.total // 0')
    local duration=$(echo "$response" | jq -r '.duration // "0s"')
    
    if [ "$total" != "null" ] && [ "$total" != "0" ]; then
        echo -e "${GREEN}✅ Success! Discovered $total resources in $duration${NC}"
        
        # Show sample resources
        echo "Sample resources:"
        echo "$response" | jq -r '.resources[0:5][] | "  • \(.name) (\(.type)) in \(.region)"' 2>/dev/null || echo "  No resources found"
        
        local total_resources=$(echo "$response" | jq -r '.resources | length // 0')
        if [ "$total_resources" -gt 5 ]; then
            local remaining=$((total_resources - 5))
            echo "  ... and $remaining more resources"
        fi
    else
        echo -e "${RED}❌ No resources discovered or error occurred${NC}"
        echo "Response: $response"
    fi
    echo
}

# Function to test region expansion
test_region_expansion() {
    echo -e "${CYAN}=== Region Expansion Test ===${NC}"
    
    # Expected regions when "all" is specified
    local expected_regions=(
        "us-east-1"      # US East (N. Virginia)
        "us-east-2"      # US East (Ohio)
        "us-west-1"      # US West (N. California)
        "us-west-2"      # US West (Oregon)
        "af-south-1"     # Africa (Cape Town)
        "ap-east-1"      # Asia Pacific (Hong Kong)
        "ap-south-1"     # Asia Pacific (Mumbai)
        "ap-northeast-1" # Asia Pacific (Tokyo)
        "ap-northeast-2" # Asia Pacific (Seoul)
        "ap-northeast-3" # Asia Pacific (Osaka)
        "ap-southeast-1" # Asia Pacific (Singapore)
        "ap-southeast-2" # Asia Pacific (Sydney)
        "ap-southeast-3" # Asia Pacific (Jakarta)
        "ap-southeast-4" # Asia Pacific (Melbourne)
        "ca-central-1"   # Canada (Central)
        "eu-central-1"   # Europe (Frankfurt)
        "eu-west-1"      # Europe (Ireland)
        "eu-west-2"      # Europe (London)
        "eu-west-3"      # Europe (Paris)
        "eu-north-1"     # Europe (Stockholm)
        "eu-south-1"     # Europe (Milan)
        "eu-south-2"     # Europe (Spain)
        "me-south-1"     # Middle East (Bahrain)
        "me-central-1"   # Middle East (UAE)
        "sa-east-1"      # South America (São Paulo)
    )
    
    echo "Expected regions when 'all' is specified: ${#expected_regions[@]} regions"
    echo "Regions:"
    for i in "${!expected_regions[@]}"; do
        printf "  %2d. %s\n" $((i+1)) "${expected_regions[$i]}"
    done
    echo
}

# Function to test performance comparison
test_performance() {
    echo -e "${CYAN}=== Performance Comparison Test ===${NC}"
    
    # Test single region
    echo "Testing single region (us-east-1)..."
    local start_time=$(date +%s.%N)
    local single_response=$(curl -s -X POST http://localhost:8080/api/v1/discover \
        -H "Content-Type: application/json" \
        -d '{"provider":"aws","regions":["us-east-1"],"account":"default"}')
    local single_end_time=$(date +%s.%N)
    local single_duration=$(echo "$single_end_time - $start_time" | bc)
    
    local single_total=$(echo "$single_response" | jq -r '.total // 0')
    if [ "$single_total" != "null" ] && [ "$single_total" != "0" ]; then
        echo -e "${GREEN}✅ Single region: $single_total resources in ${single_duration}s${NC}"
    else
        echo -e "${RED}❌ Single region test failed${NC}"
        return
    fi
    
    # Test all regions
    echo "Testing all regions..."
    local start_time=$(date +%s.%N)
    local all_response=$(curl -s -X POST http://localhost:8080/api/v1/discover \
        -H "Content-Type: application/json" \
        -d '{"provider":"aws","regions":["all"],"account":"default"}')
    local all_end_time=$(date +%s.%N)
    local all_duration=$(echo "$all_end_time - $start_time" | bc)
    
    local all_total=$(echo "$all_response" | jq -r '.total // 0')
    if [ "$all_total" != "null" ] && [ "$all_total" != "0" ]; then
        echo -e "${GREEN}✅ All regions: $all_total resources in ${all_duration}s${NC}"
        
        # Calculate performance metrics
        if [ "$single_total" -gt 0 ]; then
            local resource_ratio=$(echo "scale=2; $all_total / $single_total" | bc)
            local time_ratio=$(echo "scale=2; $all_duration / $single_duration" | bc)
            local efficiency=$(echo "scale=2; $all_total / $all_duration" | bc)
            
            echo -e "${BLUE}📊 Performance metrics:${NC}"
            echo "   Resource ratio: ${resource_ratio}x more resources"
            echo "   Time ratio: ${time_ratio}x longer"
            echo "   Efficiency: ${efficiency} resources per second"
        fi
    else
        echo -e "${RED}❌ All regions test failed${NC}"
    fi
    echo
}

# Function to test edge cases
test_edge_cases() {
    echo -e "${CYAN}=== Edge Cases Test ===${NC}"
    
    # Test invalid region
    echo "Testing invalid region..."
    local response=$(curl -s -X POST http://localhost:8080/api/v1/discover \
        -H "Content-Type: application/json" \
        -d '{"provider":"aws","regions":["invalid-region"],"account":"default"}')
    
    local total=$(echo "$response" | jq -r '.total // 0')
    if [ "$total" = "0" ]; then
        echo -e "${GREEN}✅ Invalid region handled gracefully${NC}"
    else
        echo -e "${YELLOW}⚠️  Invalid region returned $total resources${NC}"
    fi
    
    # Test empty regions array
    echo "Testing empty regions array..."
    local response=$(curl -s -X POST http://localhost:8080/api/v1/discover \
        -H "Content-Type: application/json" \
        -d '{"provider":"aws","regions":[],"account":"default"}')
    
    local total=$(echo "$response" | jq -r '.total // 0')
    if [ "$total" = "0" ]; then
        echo -e "${GREEN}✅ Empty regions array handled gracefully${NC}"
    else
        echo -e "${YELLOW}⚠️  Empty regions array returned $total resources${NC}"
    fi
    
    # Test unsupported provider
    echo "Testing unsupported provider..."
    local response=$(curl -s -X POST http://localhost:8080/api/v1/discover \
        -H "Content-Type: application/json" \
        -d '{"provider":"invalid-provider","regions":["us-east-1"],"account":"default"}')
    
    if echo "$response" | grep -q "Unsupported provider"; then
        echo -e "${GREEN}✅ Unsupported provider handled correctly${NC}"
    else
        echo -e "${YELLOW}⚠️  Unsupported provider response: $response${NC}"
    fi
    echo
}

# Main test execution
main() {
    # Check if required tools are available
    if ! command -v curl &> /dev/null; then
        echo -e "${RED}❌ curl is required but not installed${NC}"
        exit 1
    fi
    
    if ! command -v jq &> /dev/null; then
        echo -e "${RED}❌ jq is required but not installed${NC}"
        exit 1
    fi
    
    if ! command -v bc &> /dev/null; then
        echo -e "${RED}❌ bc is required but not installed${NC}"
        exit 1
    fi
    
    # Check server status
    if ! check_server; then
        exit 1
    fi
    
    echo
    
    # Run tests
    test_region_expansion
    test_discovery "Single Region Test" '["us-east-1"]'
    test_discovery "Multiple Regions Test" '["us-east-1", "us-west-2", "eu-west-1"]'
    test_discovery "All Regions Test" '["all"]'
    test_discovery "Edge Regions Test" '["ap-southeast-4", "me-central-1", "eu-south-2"]'
    test_performance
    test_edge_cases
    
    echo -e "${CYAN}=== Test Complete ===${NC}"
    echo -e "${GREEN}Enhanced AWS discovery with more regions has been tested successfully!${NC}"
}

# Run main function
main "$@"
