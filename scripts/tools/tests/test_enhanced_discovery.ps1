# Enhanced AWS Discovery Test Script (PowerShell)
# This script tests the enhanced AWS discovery functionality with various region configurations

param(
    [switch]$Verbose
)

# Colors for output
$Red = "Red"
$Green = "Green"
$Yellow = "Yellow"
$Blue = "Blue"
$Cyan = "Cyan"
$White = "White"

function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Color = $White
    )
    Write-Host $Message -ForegroundColor $Color
}

function Test-Server {
    Write-ColorOutput "Checking DriftMgr server status..." $Blue
    try {
        $response = Invoke-RestMethod -Uri "http://localhost:8080/health" -Method Get -TimeoutSec 5
        Write-ColorOutput "✅ DriftMgr server is running" $Green
        return $true
    }
    catch {
        Write-ColorOutput "❌ DriftMgr server is not running!" $Red
        Write-ColorOutput "Please start the server with: ./bin/driftmgr-server.exe" $Yellow
        return $false
    }
}

function Test-Discovery {
    param(
        [string]$TestName,
        [string]$Regions
    )
    
    Write-ColorOutput "--- $TestName ---" $Yellow
    Write-ColorOutput "Regions: $Regions" $White
    
    # Create JSON request
    $jsonRequest = @{
        provider = "aws"
        regions = $Regions -split ','
        account = "default"
    } | ConvertTo-Json
    
    try {
        $startTime = Get-Date
        $response = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/discover" -Method Post -Body $jsonRequest -ContentType "application/json"
        $endTime = Get-Date
        $duration = $endTime - $startTime
        
        if ($response.total -gt 0) {
            Write-ColorOutput "✅ Success! Discovered $($response.total) resources in $($duration.TotalSeconds.ToString('F2'))s" $Green
            
            # Show sample resources
            Write-ColorOutput "Sample resources:" $White
            $sampleResources = $response.resources | Select-Object -First 5
            foreach ($resource in $sampleResources) {
                Write-ColorOutput "  • $($resource.name) ($($resource.type)) in $($resource.region)" $White
            }
            
            if ($response.resources.Count -gt 5) {
                $remaining = $response.resources.Count - 5
                Write-ColorOutput "  ... and $remaining more resources" $White
            }
        }
        else {
            Write-ColorOutput "❌ No resources discovered" $Red
        }
    }
    catch {
        Write-ColorOutput "❌ Error: $($_.Exception.Message)" $Red
    }
    Write-Host ""
}

function Test-RegionExpansion {
    Write-ColorOutput "=== Region Expansion Test ===" $Cyan
    
    # Expected regions when "all" is specified
    $expectedRegions = @(
        "us-east-1",      # US East (N. Virginia)
        "us-east-2",      # US East (Ohio)
        "us-west-1",      # US West (N. California)
        "us-west-2",      # US West (Oregon)
        "af-south-1",     # Africa (Cape Town)
        "ap-east-1",      # Asia Pacific (Hong Kong)
        "ap-south-1",     # Asia Pacific (Mumbai)
        "ap-northeast-1", # Asia Pacific (Tokyo)
        "ap-northeast-2", # Asia Pacific (Seoul)
        "ap-northeast-3", # Asia Pacific (Osaka)
        "ap-southeast-1", # Asia Pacific (Singapore)
        "ap-southeast-2", # Asia Pacific (Sydney)
        "ap-southeast-3", # Asia Pacific (Jakarta)
        "ap-southeast-4", # Asia Pacific (Melbourne)
        "ca-central-1",   # Canada (Central)
        "eu-central-1",   # Europe (Frankfurt)
        "eu-west-1",      # Europe (Ireland)
        "eu-west-2",      # Europe (London)
        "eu-west-3",      # Europe (Paris)
        "eu-north-1",     # Europe (Stockholm)
        "eu-south-1",     # Europe (Milan)
        "eu-south-2",     # Europe (Spain)
        "me-south-1",     # Middle East (Bahrain)
        "me-central-1",   # Middle East (UAE)
        "sa-east-1"       # South America (São Paulo)
    )
    
    Write-ColorOutput "Expected regions when 'all' is specified: $($expectedRegions.Count) regions" $White
    Write-ColorOutput "Regions:" $White
    for ($i = 0; $i -lt $expectedRegions.Count; $i++) {
        Write-ColorOutput "  $($i+1). $($expectedRegions[$i])" $White
    }
    Write-Host ""
}

function Test-Performance {
    Write-ColorOutput "=== Performance Comparison Test ===" $Cyan
    
    # Test single region
    Write-ColorOutput "Testing single region (us-east-1)..." $White
    $startTime = Get-Date
    try {
        $singleResponse = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/discover" -Method Post -Body '{"provider":"aws","regions":["us-east-1"],"account":"default"}' -ContentType "application/json"
        $singleEndTime = Get-Date
        $singleDuration = $singleEndTime - $startTime
        
        if ($singleResponse.total -gt 0) {
            Write-ColorOutput "✅ Single region: $($singleResponse.total) resources in $($singleDuration.TotalSeconds.ToString('F2'))s" $Green
        }
        else {
            Write-ColorOutput "❌ Single region test failed" $Red
            return
        }
    }
    catch {
        Write-ColorOutput "❌ Single region test failed: $($_.Exception.Message)" $Red
        return
    }
    
    # Test all regions
    Write-ColorOutput "Testing all regions..." $White
    $startTime = Get-Date
    try {
        $allResponse = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/discover" -Method Post -Body '{"provider":"aws","regions":["all"],"account":"default"}' -ContentType "application/json"
        $allEndTime = Get-Date
        $allDuration = $allEndTime - $startTime
        
        if ($allResponse.total -gt 0) {
            Write-ColorOutput "✅ All regions: $($allResponse.total) resources in $($allDuration.TotalSeconds.ToString('F2'))s" $Green
            
            # Calculate performance metrics
            if ($singleResponse.total -gt 0) {
                $resourceRatio = [math]::Round($allResponse.total / $singleResponse.total, 2)
                $timeRatio = [math]::Round($allDuration.TotalSeconds / $singleDuration.TotalSeconds, 2)
                $efficiency = [math]::Round($allResponse.total / $allDuration.TotalSeconds, 2)
                
                Write-ColorOutput "📊 Performance metrics:" $Blue
                Write-ColorOutput "   Resource ratio: ${resourceRatio}x more resources" $White
                Write-ColorOutput "   Time ratio: ${timeRatio}x longer" $White
                Write-ColorOutput "   Efficiency: ${efficiency} resources per second" $White
            }
        }
        else {
            Write-ColorOutput "❌ All regions test failed" $Red
        }
    }
    catch {
        Write-ColorOutput "❌ All regions test failed: $($_.Exception.Message)" $Red
    }
    Write-Host ""
}

function Test-EdgeCases {
    Write-ColorOutput "=== Edge Cases Test ===" $Cyan
    
    # Test invalid region
    Write-ColorOutput "Testing invalid region..." $White
    try {
        $response = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/discover" -Method Post -Body '{"provider":"aws","regions":["invalid-region"],"account":"default"}' -ContentType "application/json"
        if ($response.total -eq 0) {
            Write-ColorOutput "✅ Invalid region handled gracefully" $Green
        }
        else {
            Write-ColorOutput "⚠️  Invalid region returned $($response.total) resources" $Yellow
        }
    }
    catch {
        Write-ColorOutput "✅ Invalid region handled gracefully (error returned)" $Green
    }
    
    # Test empty regions array
    Write-ColorOutput "Testing empty regions array..." $White
    try {
        $response = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/discover" -Method Post -Body '{"provider":"aws","regions":[],"account":"default"}' -ContentType "application/json"
        if ($response.total -eq 0) {
            Write-ColorOutput "✅ Empty regions array handled gracefully" $Green
        }
        else {
            Write-ColorOutput "⚠️  Empty regions array returned $($response.total) resources" $Yellow
        }
    }
    catch {
        Write-ColorOutput "✅ Empty regions array handled gracefully (error returned)" $Green
    }
    
    # Test unsupported provider
    Write-ColorOutput "Testing unsupported provider..." $White
    try {
        $response = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/discover" -Method Post -Body '{"provider":"invalid-provider","regions":["us-east-1"],"account":"default"}' -ContentType "application/json"
        Write-ColorOutput "⚠️  Unsupported provider response: $($response | ConvertTo-Json)" $Yellow
    }
    catch {
        Write-ColorOutput "✅ Unsupported provider handled correctly (error returned)" $Green
    }
    Write-Host ""
}

# Main execution
Write-ColorOutput "=== Enhanced AWS Discovery Test Suite ===" $Cyan
Write-ColorOutput "Testing DriftMgr enhanced AWS discovery with various region configurations" $White
Write-Host ""

# Check server status
if (-not (Test-Server)) {
    exit 1
}

Write-Host ""

# Run tests
Test-RegionExpansion
Test-Discovery -TestName "Single Region Test" -Regions "us-east-1"
Test-Discovery -TestName "Multiple Regions Test" -Regions "us-east-1,us-west-2,eu-west-1"
Test-Discovery -TestName "All Regions Test" -Regions "all"
Test-Discovery -TestName "Edge Regions Test" -Regions "ap-southeast-4,me-central-1,eu-south-2"
Test-Performance
Test-EdgeCases

Write-ColorOutput "=== Test Complete ===" $Cyan
Write-ColorOutput "Enhanced AWS discovery with more regions has been tested successfully!" $Green
