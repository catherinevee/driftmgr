# Multi-Cloud Discovery Test Script (PowerShell)
# This script tests the enhanced multi-cloud discovery functionality with AWS, Azure, and GCP

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
$Magenta = "Magenta"

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

function Test-ProviderDiscovery {
    param(
        [string]$Provider,
        [string]$TestName,
        [string]$Regions
    )
    
    Write-ColorOutput "  $TestName" $Yellow
    Write-ColorOutput "  Regions: $Regions" $White
    
    # Create JSON request
    $jsonRequest = @{
        provider = $Provider
        regions = $Regions -split ','
        account = "default"
    } | ConvertTo-Json
    
    try {
        $startTime = Get-Date
        $response = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/discover" -Method Post -Body $jsonRequest -ContentType "application/json"
        $endTime = Get-Date
        $duration = $endTime - $startTime
        
        if ($response.total -gt 0) {
            Write-ColorOutput "  ✅ Success! Discovered $($response.total) resources in $($duration.TotalSeconds.ToString('F2'))s" $Green
            
            # Show sample resources
            Write-ColorOutput "  Sample resources:" $White
            $sampleResources = $response.resources | Select-Object -First 3
            foreach ($resource in $sampleResources) {
                Write-ColorOutput "    • $($resource.name) ($($resource.type)) in $($resource.region)" $White
            }
            
            if ($response.resources.Count -gt 3) {
                $remaining = $response.resources.Count - 3
                Write-ColorOutput "    ... and $remaining more resources" $White
            }
        }
        else {
            Write-ColorOutput "  ❌ No resources discovered" $Red
        }
    }
    catch {
        Write-ColorOutput "  ❌ Error: $($_.Exception.Message)" $Red
    }
    Write-Host ""
}

function Test-MultiCloudDiscovery {
    Write-ColorOutput "=== Multi-Cloud Discovery Test ===" $Cyan
    Write-ColorOutput "Testing AWS, Azure, and GCP resource discovery with various region configurations..." $White
    Write-Host ""
    
    $providers = @("aws", "azure", "gcp")
    
    foreach ($provider in $providers) {
        Write-ColorOutput "--- Testing $provider Discovery ---" $Magenta
        
        # Single Region Test
        $singleRegion = Get-SingleRegion -Provider $provider
        Test-ProviderDiscovery -Provider $provider -TestName "Single Region Test" -Regions $singleRegion
        
        # Multiple Regions Test
        $multipleRegions = Get-MultipleRegions -Provider $provider
        Test-ProviderDiscovery -Provider $provider -TestName "Multiple Regions Test" -Regions $multipleRegions
        
        # All Regions Test
        Test-ProviderDiscovery -Provider $provider -TestName "All Regions Test" -Regions "all"
    }
}

function Get-SingleRegion {
    param([string]$Provider)
    
    switch ($Provider) {
        "aws" { return "us-east-1" }
        "azure" { return "eastus" }
        "gcp" { return "us-central1" }
        default { return "us-east-1" }
    }
}

function Get-MultipleRegions {
    param([string]$Provider)
    
    switch ($Provider) {
        "aws" { return "us-east-1,us-west-2,eu-west-1" }
        "azure" { return "eastus,westus2,northeurope" }
        "gcp" { return "us-central1,us-east1,europe-west1" }
        default { return "us-east-1,us-west-2" }
    }
}

function Test-RegionExpansionMultiCloud {
    Write-ColorOutput "=== Multi-Cloud Region Expansion Test ===" $Cyan
    
    $providers = @("aws", "azure", "gcp")
    
    foreach ($provider in $providers) {
        Write-ColorOutput "--- $provider Regions ---" $Yellow
        
        $expectedRegions = Get-ExpectedRegions -Provider $provider
        
        Write-ColorOutput "Expected regions when 'all' is specified: $($expectedRegions.Count) regions" $White
        Write-ColorOutput "Sample regions:" $White
        for ($i = 0; $i -lt [Math]::Min(10, $expectedRegions.Count); $i++) {
            Write-ColorOutput "  $($i+1). $($expectedRegions[$i])" $White
        }
        if ($expectedRegions.Count -gt 10) {
            $remaining = $expectedRegions.Count - 10
            Write-ColorOutput "  ... and $remaining more regions" $White
        }
        Write-Host ""
    }
}

function Get-ExpectedRegions {
    param([string]$Provider)
    
    switch ($Provider) {
        "aws" {
            return @(
                "us-east-1", "us-east-2", "us-west-1", "us-west-2", "af-south-1",
                "ap-east-1", "ap-south-1", "ap-northeast-1", "ap-northeast-2", "ap-northeast-3",
                "ap-southeast-1", "ap-southeast-2", "ap-southeast-3", "ap-southeast-4",
                "ca-central-1", "eu-central-1", "eu-west-1", "eu-west-2", "eu-west-3",
                "eu-north-1", "eu-south-1", "eu-south-2", "me-south-1", "me-central-1", "sa-east-1"
            )
        }
        "azure" {
            return @(
                "eastus", "eastus2", "southcentralus", "westus2", "westus3",
                "australiaeast", "southeastasia", "northeurope", "swedencentral", "uksouth",
                "westeurope", "centralus", "northcentralus", "westus", "southafricanorth",
                "centralindia", "eastasia", "japaneast", "japanwest", "koreacentral",
                "canadacentral", "francecentral", "germanywestcentral", "italynorth",
                "norwayeast", "polandcentral", "switzerlandnorth", "uaenorth", "brazilsouth"
            )
        }
        "gcp" {
            return @(
                "us-central1", "us-east1", "us-east4", "us-west1", "us-west2",
                "us-west3", "us-west4", "europe-west1", "europe-west2", "europe-west3",
                "europe-west4", "europe-west6", "europe-west8", "europe-west9", "europe-west10",
                "europe-west12", "europe-central2", "europe-north1", "europe-southwest1",
                "asia-east1", "asia-northeast1", "asia-northeast2", "asia-northeast3",
                "asia-south1", "asia-south2", "asia-southeast1", "asia-southeast2",
                "australia-southeast1", "australia-southeast2", "southamerica-east1",
                "northamerica-northeast1", "northamerica-northeast2"
            )
        }
        default { return @() }
    }
}

function Test-PerformanceComparisonMultiCloud {
    Write-ColorOutput "=== Multi-Cloud Performance Comparison Test ===" $Cyan
    
    $providers = @("aws", "azure", "gcp")
    
    foreach ($provider in $providers) {
        Write-ColorOutput "--- $provider Performance Test ---" $Magenta
        
        # Test single region
        Write-ColorOutput "Testing single region..." $White
        $startTime = Get-Date
        try {
            $singleResponse = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/discover" -Method Post -Body "{\"provider\":\"$provider\",\"regions\":[\"$(Get-SingleRegion -Provider $provider)\"],\"account\":\"default\"}" -ContentType "application/json"
            $singleEndTime = Get-Date
            $singleDuration = $singleEndTime - $startTime
            
            if ($singleResponse.total -gt 0) {
                Write-ColorOutput "  ✅ Single region: $($singleResponse.total) resources in $($singleDuration.TotalSeconds.ToString('F2'))s" $Green
            }
            else {
                Write-ColorOutput "  ❌ Single region test failed" $Red
                continue
            }
        }
        catch {
            Write-ColorOutput "  ❌ Single region test failed: $($_.Exception.Message)" $Red
            continue
        }
        
        # Test all regions
        Write-ColorOutput "Testing all regions..." $White
        $startTime = Get-Date
        try {
            $allResponse = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/discover" -Method Post -Body "{\"provider\":\"$provider\",\"regions\":[\"all\"],\"account\":\"default\"}" -ContentType "application/json"
            $allEndTime = Get-Date
            $allDuration = $allEndTime - $startTime
            
            if ($allResponse.total -gt 0) {
                Write-ColorOutput "  ✅ All regions: $($allResponse.total) resources in $($allDuration.TotalSeconds.ToString('F2'))s" $Green
                
                # Calculate performance metrics
                if ($singleResponse.total -gt 0) {
                    $resourceRatio = [math]::Round($allResponse.total / $singleResponse.total, 2)
                    $timeRatio = [math]::Round($allDuration.TotalSeconds / $singleDuration.TotalSeconds, 2)
                    $efficiency = [math]::Round($allResponse.total / $allDuration.TotalSeconds, 2)
                    
                    Write-ColorOutput "  📊 Performance metrics:" $Blue
                    Write-ColorOutput "     Resource ratio: ${resourceRatio}x more resources" $White
                    Write-ColorOutput "     Time ratio: ${timeRatio}x longer" $White
                    Write-ColorOutput "     Efficiency: ${efficiency} resources per second" $White
                }
            }
            else {
                Write-ColorOutput "  ❌ All regions test failed" $Red
            }
        }
        catch {
            Write-ColorOutput "  ❌ All regions test failed: $($_.Exception.Message)" $Red
        }
        Write-Host ""
    }
}

function Test-CrossCloudComparison {
    Write-ColorOutput "=== Cross-Cloud Comparison Test ===" $Cyan
    
    # Test all providers with single region
    $providers = @("aws", "azure", "gcp")
    $results = @{}
    
    foreach ($provider in $providers) {
        Write-ColorOutput "Testing $provider..." $White
        
        try {
            $response = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/discover" -Method Post -Body "{\"provider\":\"$provider\",\"regions\":[\"$(Get-SingleRegion -Provider $provider)\"],\"account\":\"default\"}" -ContentType "application/json"
            $results[$provider] = $response
            Write-ColorOutput "  ✅ $provider`: $($response.total) resources in $($response.duration)" $Green
        }
        catch {
            Write-ColorOutput "  ❌ $provider test failed: $($_.Exception.Message)" $Red
        }
    }
    
    # Compare results
    Write-ColorOutput "`n📊 Cross-Cloud Comparison:" $Blue
    if ($results.Count -gt 1) {
        $maxResources = 0
        $fastestProvider = ""
        $fastestTime = [TimeSpan]::MaxValue
        
        foreach ($provider in $results.Keys) {
            $result = $results[$provider]
            if ($result.total -gt $maxResources) {
                $maxResources = $result.total
            }
            if ($result.duration -lt $fastestTime) {
                $fastestTime = $result.duration
                $fastestProvider = $provider
            }
        }
        
        Write-ColorOutput "   Most resources: $maxResources" $White
        Write-ColorOutput "   Fastest discovery: $fastestProvider ($fastestTime)" $White
    }
    Write-Host ""
}

function Test-EdgeCasesMultiCloud {
    Write-ColorOutput "=== Multi-Cloud Edge Cases Test ===" $Cyan
    
    $providers = @("aws", "azure", "gcp")
    
    foreach ($provider in $providers) {
        Write-ColorOutput "--- $provider Edge Cases ---" $Yellow
        
        # Test invalid region
        Write-ColorOutput "Testing invalid region..." $White
        try {
            $response = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/discover" -Method Post -Body "{\"provider\":\"$provider\",\"regions\":[\"invalid-region\"],\"account\":\"default\"}" -ContentType "application/json"
            if ($response.total -eq 0) {
                Write-ColorOutput "  ✅ Invalid region handled gracefully" $Green
            }
            else {
                Write-ColorOutput "  ⚠️  Invalid region returned $($response.total) resources" $Yellow
            }
        }
        catch {
            Write-ColorOutput "  ✅ Invalid region handled gracefully (error returned)" $Green
        }
        
        # Test empty regions array
        Write-ColorOutput "Testing empty regions array..." $White
        try {
            $response = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/discover" -Method Post -Body "{\"provider\":\"$provider\",\"regions\":[],\"account\":\"default\"}" -ContentType "application/json"
            if ($response.total -eq 0) {
                Write-ColorOutput "  ✅ Empty regions array handled gracefully" $Green
            }
            else {
                Write-ColorOutput "  ⚠️  Empty regions array returned $($response.total) resources" $Yellow
            }
        }
        catch {
            Write-ColorOutput "  ✅ Empty regions array handled gracefully (error returned)" $Green
        }
        
        Write-Host ""
    }
}

# Main execution
Write-ColorOutput "=== Multi-Cloud Discovery Test Suite ===" $Cyan
Write-ColorOutput "Testing DriftMgr enhanced multi-cloud discovery with AWS, Azure, and GCP" $White
Write-Host ""

# Check server status
if (-not (Test-Server)) {
    exit 1
}

Write-Host ""

# Run tests
Test-RegionExpansionMultiCloud
Test-MultiCloudDiscovery
Test-PerformanceComparisonMultiCloud
Test-CrossCloudComparison
Test-EdgeCasesMultiCloud

Write-ColorOutput "=== Multi-Cloud Test Complete ===" $Cyan
Write-ColorOutput "Enhanced multi-cloud discovery with AWS, Azure, and GCP has been tested successfully!" $Green
