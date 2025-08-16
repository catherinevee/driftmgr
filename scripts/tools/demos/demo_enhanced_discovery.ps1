# Enhanced AWS Discovery Demo Script
# This script demonstrates the enhanced AWS discovery functionality

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
║                    Enhanced AWS Discovery Demo               ║
║                                                              ║
║  Discover • Analyze • Monitor • Remediate                   ║
║                                                              ║
║  Features:                                                   ║
║  • Single Region Discovery                                   ║
║  • Multiple Regions Discovery                                ║
║  • All Regions Discovery (26+ regions)                      ║
║  • Parallel Processing                                       ║
║  • Comprehensive Resource Discovery                         ║
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

function Show-RegionInfo {
    Write-ColorOutput "🌍 AWS Regions Supported:" $Cyan
    Write-ColorOutput ""
    
    $regions = @{
        "North America" = @("us-east-1", "us-east-2", "us-west-1", "us-west-2", "ca-central-1")
        "Europe" = @("eu-central-1", "eu-west-1", "eu-west-2", "eu-west-3", "eu-north-1", "eu-south-1", "eu-south-2")
        "Asia Pacific" = @("ap-east-1", "ap-south-1", "ap-northeast-1", "ap-northeast-2", "ap-northeast-3", "ap-southeast-1", "ap-southeast-2", "ap-southeast-3", "ap-southeast-4")
        "Other Regions" = @("af-south-1", "me-south-1", "me-central-1", "sa-east-1")
    }
    
    foreach ($continent in $regions.Keys) {
        Write-ColorOutput "  $continent:" $Yellow
        foreach ($region in $regions[$continent]) {
            Write-ColorOutput "    • $region" $White
        }
        Write-ColorOutput ""
    }
}

function Show-ResourceTypes {
    Write-ColorOutput "🔍 AWS Resource Types Discovered:" $Cyan
    Write-ColorOutput ""
    
    $resourceCategories = @{
        "Compute & Networking" = @("EC2 Instances", "VPCs", "Security Groups", "Auto Scaling Groups")
        "Storage & Databases" = @("S3 Buckets", "RDS Instances", "DynamoDB Tables", "ElastiCache Clusters")
        "Serverless & Containers" = @("Lambda Functions", "ECS Clusters", "EKS Clusters")
        "Management & Monitoring" = @("CloudFormation Stacks", "IAM Users", "Route53 Zones", "SQS Queues", "SNS Topics")
    }
    
    foreach ($category in $resourceCategories.Keys) {
        Write-ColorOutput "  $category:" $Yellow
        foreach ($resource in $resourceCategories[$category]) {
            Write-ColorOutput "    • $resource" $White
        }
        Write-ColorOutput ""
    }
}

function Invoke-DiscoveryDemo {
    param(
        [string]$DemoName,
        [string]$Regions,
        [string]$Description
    )
    
    Write-ColorOutput "🚀 Demo: $DemoName" $Magenta
    Write-ColorOutput "   $Description" $White
    Write-ColorOutput "   Regions: $Regions" $Yellow
    Write-ColorOutput ""
    
    # Create JSON request
    $jsonRequest = @{
        provider = "aws"
        regions = $Regions -split ','
        account = "default"
    } | ConvertTo-Json
    
    try {
        $startTime = Get-Date
        Write-ColorOutput "   🔍 Discovering resources..." $Blue
        
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
    Write-ColorOutput "1. Single Region Discovery (us-east-1)" $Yellow
    Write-ColorOutput "2. Multiple Regions Discovery (us-east-1, us-west-2, eu-west-1)" $Yellow
    Write-ColorOutput "3. All Regions Discovery (26+ regions)" $Yellow
    Write-ColorOutput "4. Edge Regions Discovery (ap-southeast-4, me-central-1)" $Yellow
    Write-ColorOutput "5. Performance Comparison" $Yellow
    Write-ColorOutput "6. Exit" $Yellow
    Write-ColorOutput ""
    
    do {
        $choice = Read-Host "Enter your choice (1-6)"
        
        switch ($choice) {
            "1" { 
                Invoke-DiscoveryDemo -DemoName "Single Region" -Regions "us-east-1" -Description "Discover resources in US East (N. Virginia)"
            }
            "2" { 
                Invoke-DiscoveryDemo -DemoName "Multiple Regions" -Regions "us-east-1,us-west-2,eu-west-1" -Description "Discover resources across multiple regions"
            }
            "3" { 
                Invoke-DiscoveryDemo -DemoName "All Regions" -Regions "all" -Description "Discover resources across all AWS regions"
            }
            "4" { 
                Invoke-DiscoveryDemo -DemoName "Edge Regions" -Regions "ap-southeast-4,me-central-1" -Description "Discover resources in edge regions"
            }
            "5" { 
                Show-PerformanceComparison
            }
            "6" { 
                Write-ColorOutput "👋 Goodbye!" $Green
                return
            }
            default { 
                Write-ColorOutput "❌ Invalid choice. Please enter 1-6." $Red
            }
        }
        
        if ($choice -ne "6") {
            Write-ColorOutput "Press any key to continue..." $Yellow
            $null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
            Clear-Host
            Show-InteractiveDemo
        }
    } while ($choice -ne "6")
}

function Show-PerformanceComparison {
    Write-ColorOutput "📊 Performance Comparison Demo" $Magenta
    Write-ColorOutput ""
    
    # Test single region
    Write-ColorOutput "Testing single region (us-east-1)..." $Blue
    $startTime = Get-Date
    try {
        $singleResponse = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/discover" -Method Post -Body '{"provider":"aws","regions":["us-east-1"],"account":"default"}' -ContentType "application/json"
        $singleEndTime = Get-Date
        $singleDuration = $singleEndTime - $startTime
        
        Write-ColorOutput "   ✅ Single region: $($singleResponse.total) resources in $($singleDuration.TotalSeconds.ToString('F2'))s" $Green
    }
    catch {
        Write-ColorOutput "   ❌ Single region test failed" $Red
        return
    }
    
    # Test all regions
    Write-ColorOutput "Testing all regions..." $Blue
    $startTime = Get-Date
    try {
        $allResponse = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/discover" -Method Post -Body '{"provider":"aws","regions":["all"],"account":"default"}' -ContentType "application/json"
        $allEndTime = Get-Date
        $allDuration = $allEndTime - $startTime
        
        Write-ColorOutput "   ✅ All regions: $($allResponse.total) resources in $($allDuration.TotalSeconds.ToString('F2'))s" $Green
        
        # Calculate metrics
        if ($singleResponse.total -gt 0) {
            $resourceRatio = [math]::Round($allResponse.total / $singleResponse.total, 2)
            $timeRatio = [math]::Round($allDuration.TotalSeconds / $singleDuration.TotalSeconds, 2)
            $efficiency = [math]::Round($allResponse.total / $allDuration.TotalSeconds, 2)
            
            Write-ColorOutput ""
            Write-ColorOutput "📈 Performance Metrics:" $Cyan
            Write-ColorOutput "   • Resource ratio: ${resourceRatio}x more resources" $White
            Write-ColorOutput "   • Time ratio: ${timeRatio}x longer" $White
            Write-ColorOutput "   • Efficiency: ${efficiency} resources per second" $White
        }
    }
    catch {
        Write-ColorOutput "   ❌ All regions test failed" $Red
    }
    
    Write-ColorOutput ""
}

function Show-AutomatedDemo {
    Write-ColorOutput "🤖 Automated Demo Mode" $Magenta
    Write-ColorOutput "Running all discovery scenarios..." $White
    Write-ColorOutput ""
    
    # Demo 1: Single Region
    Invoke-DiscoveryDemo -DemoName "Single Region Discovery" -Regions "us-east-1" -Description "Discover resources in US East (N. Virginia)"
    
    # Demo 2: Multiple Regions
    Invoke-DiscoveryDemo -DemoName "Multiple Regions Discovery" -Regions "us-east-1,us-west-2,eu-west-1" -Description "Discover resources across multiple regions"
    
    # Demo 3: All Regions
    Invoke-DiscoveryDemo -DemoName "All Regions Discovery" -Regions "all" -Description "Discover resources across all AWS regions"
    
    # Demo 4: Edge Regions
    Invoke-DiscoveryDemo -DemoName "Edge Regions Discovery" -Regions "ap-southeast-4,me-central-1" -Description "Discover resources in edge regions"
    
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
Show-RegionInfo
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
    Write-ColorOutput "Example: .\demo_enhanced_discovery.ps1 -Demo" $Yellow
}

Write-ColorOutput ""
Write-ColorOutput "🎯 Enhanced AWS Discovery Demo Complete!" $Green
Write-ColorOutput "For more information, see: AWS_DISCOVERY_FEATURE.md" $Blue
