# Multi-Cloud Discovery Demo Script
# This script demonstrates the enhanced multi-cloud discovery functionality

param(
    [switch]$Demo,
    [switch]$Interactive
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

function Show-Banner {
    Write-ColorOutput @"
╔══════════════════════════════════════════════════════════════╗
║                    Multi-Cloud Discovery Demo                ║
║                                                              ║
║  AWS • Azure • GCP                                          ║
║                                                              ║
║  Features:                                                   ║
║  • Multi-Provider Support (AWS, Azure, GCP)                ║
║  • Single Region Discovery                                   ║
║  • Multiple Regions Discovery                                ║
║  • All Regions Discovery (26+ AWS, 29+ Azure, 33+ GCP)     ║
║  • Parallel Processing                                       ║
║  • Cross-Cloud Comparison                                    ║
╚══════════════════════════════════════════════════════════════╝
"@ $Cyan
}

function Test-ServerConnection {
    Write-ColorOutput "🔍 Checking DriftMgr server connection..." $Blue
    try {
        $response = Invoke-RestMethod -Uri "http://localhost:8080/health" -Method Get -TimeoutSec 5
        Write-ColorOutput "✅ Server is running and healthy!" $Green
        Write-ColorOutput "   Version: $($response.version)" $White
        Write-ColorOutput "   Service: $($response.service)" $White
        return $true
    }
    catch {
        Write-ColorOutput "❌ Server connection failed!" $Red
        Write-ColorOutput "   Please start the server with: ./bin/driftmgr-server.exe" $Yellow
        return $false
    }
}

function Show-ProviderInfo {
    Write-ColorOutput "☁️  Cloud Providers Supported:" $Cyan
    Write-ColorOutput ""
    
    $providers = @{
        "AWS" = @{
            "Regions" = 26
            "Description" = "Amazon Web Services"
            "SampleRegions" = @("us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1")
        }
        "Azure" = @{
            "Regions" = 29
            "Description" = "Microsoft Azure"
            "SampleRegions" = @("eastus", "westus2", "northeurope", "southeastasia")
        }
        "GCP" = @{
            "Regions" = 33
            "Description" = "Google Cloud Platform"
            "SampleRegions" = @("us-central1", "us-east1", "europe-west1", "asia-east1")
        }
    }
    
    foreach ($provider in $providers.Keys) {
        $info = $providers[$provider]
        Write-ColorOutput "  $provider ($($info.Description)):" $Yellow
        Write-ColorOutput "    • $($info.Regions) regions available" $White
        Write-ColorOutput "    • Sample regions: $($info.SampleRegions -join ', ')" $White
        Write-ColorOutput ""
    }
}

function Show-ResourceTypes {
    Write-ColorOutput "🔍 Resource Types Discovered:" $Cyan
    Write-ColorOutput ""
    
    $resourceCategories = @{
        "Compute" = @("Virtual Machines", "Instance Groups", "Auto Scaling")
        "Storage" = @("Object Storage", "Block Storage", "File Storage")
        "Databases" = @("Relational DBs", "NoSQL DBs", "Caching")
        "Networking" = @("VPCs/VNets", "Security Groups", "Load Balancers")
        "Serverless" = @("Functions", "Containers", "Kubernetes")
        "Management" = @("IAM/Policies", "Monitoring", "Security")
    }
    
    foreach ($category in $resourceCategories.Keys) {
        Write-ColorOutput "  $category:" $Yellow
        foreach ($resource in $resourceCategories[$category]) {
            Write-ColorOutput "    • $resource" $White
        }
        Write-ColorOutput ""
    }
}

function Invoke-ProviderDiscoveryDemo {
    param(
        [string]$Provider,
        [string]$DemoName,
        [string]$Regions,
        [string]$Description
    )
    
    Write-ColorOutput "🚀 Demo: $DemoName" $Magenta
    Write-ColorOutput "   Provider: $Provider" $White
    Write-ColorOutput "   $Description" $White
    Write-ColorOutput "   Regions: $Regions" $Yellow
    Write-ColorOutput ""
    
    # Create JSON request
    $jsonRequest = @{
        provider = $Provider.ToLower()
        regions = $Regions -split ','
        account = "default"
    } | ConvertTo-Json
    
    try {
        $startTime = Get-Date
        Write-ColorOutput "   🔍 Discovering $Provider resources..." $Blue
        
        $response = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/discover" -Method Post -Body $jsonRequest -ContentType "application/json"
        
        $endTime = Get-Date
        $duration = $endTime - $startTime
        
        if ($response.total -gt 0) {
            Write-ColorOutput "   ✅ Success! Discovered $($response.total) resources in $($duration.TotalSeconds.ToString('F2'))s" $Green
            
            # Show resource breakdown by type
            $resourceTypes = $response.resources | Group-Object type | Sort-Object Count -Descending
            Write-ColorOutput "   📊 Resource breakdown:" $Blue
            foreach ($type in $resourceTypes | Select-Object -First 5) {
                Write-ColorOutput "      • $($type.Name): $($type.Count)" $White
            }
            
            # Show sample resources
            Write-ColorOutput "   📋 Sample resources:" $Blue
            $sampleResources = $response.resources | Select-Object -First 3
            foreach ($resource in $sampleResources) {
                Write-ColorOutput "      • $($resource.name) ($($resource.type)) in $($resource.region)" $White
            }
        }
        else {
            Write-ColorOutput "   ⚠️  No resources discovered in specified regions" $Yellow
        }
    }
    catch {
        Write-ColorOutput "   ❌ Error: $($_.Exception.Message)" $Red
    }
    
    Write-ColorOutput ""
    Start-Sleep -Seconds 2
}

function Show-InteractiveDemo {
    Write-ColorOutput "🎮 Interactive Demo Mode" $Magenta
    Write-ColorOutput "Choose a demo to run:" $White
    Write-ColorOutput ""
    Write-ColorOutput "1. AWS Single Region Discovery (us-east-1)" $Yellow
    Write-ColorOutput "2. Azure Single Region Discovery (eastus)" $Yellow
    Write-ColorOutput "3. GCP Single Region Discovery (us-central1)" $Yellow
    Write-ColorOutput "4. AWS Multiple Regions Discovery" $Yellow
    Write-ColorOutput "5. Azure Multiple Regions Discovery" $Yellow
    Write-ColorOutput "6. GCP Multiple Regions Discovery" $Yellow
    Write-ColorOutput "7. AWS All Regions Discovery" $Yellow
    Write-ColorOutput "8. Azure All Regions Discovery" $Yellow
    Write-ColorOutput "9. GCP All Regions Discovery" $Yellow
    Write-ColorOutput "10. Cross-Cloud Comparison" $Yellow
    Write-ColorOutput "11. Exit" $Yellow
    Write-ColorOutput ""
    
    do {
        $choice = Read-Host "Enter your choice (1-11)"
        
        switch ($choice) {
            "1" { 
                Invoke-ProviderDiscoveryDemo -Provider "AWS" -DemoName "AWS Single Region" -Regions "us-east-1" -Description "Discover AWS resources in US East (N. Virginia)"
            }
            "2" { 
                Invoke-ProviderDiscoveryDemo -Provider "Azure" -DemoName "Azure Single Region" -Regions "eastus" -Description "Discover Azure resources in East US"
            }
            "3" { 
                Invoke-ProviderDiscoveryDemo -Provider "GCP" -DemoName "GCP Single Region" -Regions "us-central1" -Description "Discover GCP resources in US Central (Iowa)"
            }
            "4" { 
                Invoke-ProviderDiscoveryDemo -Provider "AWS" -DemoName "AWS Multiple Regions" -Regions "us-east-1,us-west-2,eu-west-1" -Description "Discover AWS resources across multiple regions"
            }
            "5" { 
                Invoke-ProviderDiscoveryDemo -Provider "Azure" -DemoName "Azure Multiple Regions" -Regions "eastus,westus2,northeurope" -Description "Discover Azure resources across multiple regions"
            }
            "6" { 
                Invoke-ProviderDiscoveryDemo -Provider "GCP" -DemoName "GCP Multiple Regions" -Regions "us-central1,us-east1,europe-west1" -Description "Discover GCP resources across multiple regions"
            }
            "7" { 
                Invoke-ProviderDiscoveryDemo -Provider "AWS" -DemoName "AWS All Regions" -Regions "all" -Description "Discover AWS resources across all 26+ regions"
            }
            "8" { 
                Invoke-ProviderDiscoveryDemo -Provider "Azure" -DemoName "Azure All Regions" -Regions "all" -Description "Discover Azure resources across all 29+ regions"
            }
            "9" { 
                Invoke-ProviderDiscoveryDemo -Provider "GCP" -DemoName "GCP All Regions" -Regions "all" -Description "Discover GCP resources across all 33+ regions"
            }
            "10" { 
                Show-CrossCloudComparison
            }
            "11" { 
                Write-ColorOutput "👋 Goodbye!" $Green
                return
            }
            default { 
                Write-ColorOutput "❌ Invalid choice. Please enter 1-11." $Red
            }
        }
        
        if ($choice -ne "11") {
            Write-ColorOutput "Press any key to continue..." $Yellow
            $null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
            Clear-Host
            Show-InteractiveDemo
        }
    } while ($choice -ne "11")
}

function Show-CrossCloudComparison {
    Write-ColorOutput "📊 Cross-Cloud Comparison Demo" $Magenta
    Write-ColorOutput ""
    
    $providers = @("aws", "azure", "gcp")
    $results = @{}
    
    foreach ($provider in $providers) {
        Write-ColorOutput "Testing $($provider.ToUpper())..." $Blue
        
        $singleRegion = switch ($provider) {
            "aws" { "us-east-1" }
            "azure" { "eastus" }
            "gcp" { "us-central1" }
        }
        
        try {
            $startTime = Get-Date
            $response = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/discover" -Method Post -Body "{\"provider\":\"$provider\",\"regions\":[\"$singleRegion\"],\"account\":\"default\"}" -ContentType "application/json"
            $endTime = Get-Date
            $duration = $endTime - $startTime
            
            $results[$provider] = @{
                "Total" = $response.total
                "Duration" = $duration
                "Success" = $true
            }
            
            Write-ColorOutput "  ✅ $($provider.ToUpper()): $($response.total) resources in $($duration.TotalSeconds.ToString('F2'))s" $Green
        }
        catch {
            Write-ColorOutput "  ❌ $($provider.ToUpper()) test failed: $($_.Exception.Message)" $Red
            $results[$provider] = @{
                "Total" = 0
                "Duration" = [TimeSpan]::Zero
                "Success" = $false
            }
        }
    }
    
    # Compare results
    Write-ColorOutput ""
    Write-ColorOutput "📈 Cross-Cloud Comparison Results:" $Cyan
    
    $successfulResults = $results | Where-Object { $_.Value.Success }
    if ($successfulResults.Count -gt 1) {
        $maxResources = 0
        $fastestProvider = ""
        $fastestTime = [TimeSpan]::MaxValue
        
        foreach ($provider in $successfulResults.Keys) {
            $result = $successfulResults[$provider]
            if ($result.Total -gt $maxResources) {
                $maxResources = $result.Total
            }
            if ($result.Duration -lt $fastestTime) {
                $fastestTime = $result.Duration
                $fastestProvider = $provider
            }
        }
        
        Write-ColorOutput "   Most resources: $maxResources" $White
        Write-ColorOutput "   Fastest discovery: $($fastestProvider.ToUpper()) ($($fastestTime.TotalSeconds.ToString('F2'))s)" $White
        
        # Show resource distribution
        Write-ColorOutput "   Resource distribution:" $White
        foreach ($provider in $successfulResults.Keys) {
            $result = $successfulResults[$provider]
            $percentage = if ($maxResources -gt 0) { [math]::Round(($result.Total / $maxResources) * 100, 1) } else { 0 }
            Write-ColorOutput "     • $($provider.ToUpper()): $($result.Total) resources ($percentage%)" $White
        }
    }
    else {
        Write-ColorOutput "   Not enough successful results for comparison" $Yellow
    }
    
    Write-ColorOutput ""
}

function Show-AutomatedDemo {
    Write-ColorOutput "🤖 Automated Demo Mode" $Magenta
    Write-ColorOutput "Running multi-cloud discovery scenarios..." $White
    Write-ColorOutput ""
    
    # Demo 1: AWS Single Region
    Invoke-ProviderDiscoveryDemo -Provider "AWS" -DemoName "AWS Single Region Discovery" -Regions "us-east-1" -Description "Discover AWS resources in US East (N. Virginia)"
    
    # Demo 2: Azure Single Region
    Invoke-ProviderDiscoveryDemo -Provider "Azure" -DemoName "Azure Single Region Discovery" -Regions "eastus" -Description "Discover Azure resources in East US"
    
    # Demo 3: GCP Single Region
    Invoke-ProviderDiscoveryDemo -Provider "GCP" -DemoName "GCP Single Region Discovery" -Regions "us-central1" -Description "Discover GCP resources in US Central (Iowa)"
    
    # Demo 4: AWS Multiple Regions
    Invoke-ProviderDiscoveryDemo -Provider "AWS" -DemoName "AWS Multiple Regions Discovery" -Regions "us-east-1,us-west-2,eu-west-1" -Description "Discover AWS resources across multiple regions"
    
    # Demo 5: Cross-Cloud Comparison
    Show-CrossCloudComparison
    
    Write-ColorOutput "🎉 Automated demo completed!" $Green
}

# Main execution
Clear-Host
Show-Banner

# Check server connection
if (-not (Test-ServerConnection)) {
    exit 1
}

Write-ColorOutput ""
Show-ProviderInfo
Show-ResourceTypes

# Determine demo mode
if ($Interactive) {
    Show-InteractiveDemo
}
elseif ($Demo) {
    Show-AutomatedDemo
}
else {
    Write-ColorOutput "Choose demo mode:" $Cyan
    Write-ColorOutput "  -Demo        : Run automated demo" $White
    Write-ColorOutput "  -Interactive : Run interactive demo" $White
    Write-ColorOutput ""
    Write-ColorOutput "Example: .\demo_multi_cloud_discovery.ps1 -Demo" $Yellow
}

Write-ColorOutput ""
Write-ColorOutput "🎯 Multi-Cloud Discovery Demo Complete!" $Green
Write-ColorOutput "For more information, see: MULTI_CLOUD_DISCOVERY_FEATURE.md" $Blue
