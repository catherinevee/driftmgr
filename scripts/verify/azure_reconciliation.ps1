#!/usr/bin/env pwsh
# Azure Resource Reconciliation Script
# Compares DriftMgr findings with actual Azure resources

Write-Host "=== Azure Resource Reconciliation ===" -ForegroundColor Cyan
Write-Host ""

# Get actual Azure resources
Write-Host "Discovering actual Azure resources..." -ForegroundColor Yellow

# Get all subscriptions
$subscriptions = az account list --output json | ConvertFrom-Json

$totalResources = 0
$resourceDetails = @()

foreach ($sub in $subscriptions) {
    Write-Host "Checking subscription: $($sub.name) ($($sub.id))"
    
    # Set subscription
    az account set --subscription $sub.id
    
    # Get resources in this subscription
    $resources = az resource list --output json | ConvertFrom-Json
    $subCount = $resources.Count
    $totalResources += $subCount
    
    Write-Host "  Found $subCount resources" -ForegroundColor Green
    
    foreach ($resource in $resources) {
        $resourceDetails += @{
            Subscription = $sub.name
            Name = $resource.name
            Type = $resource.type
            Location = $resource.location
            ResourceGroup = $resource.resourceGroup
        }
    }
}

Write-Host ""
Write-Host "Total Azure resources across all subscriptions: $totalResources" -ForegroundColor Cyan

# Run DriftMgr discovery
Write-Host ""
Write-Host "Running DriftMgr discovery..." -ForegroundColor Yellow
$driftmgrOutput = ./driftmgr.exe discover --provider azure --format json 2>$null | ConvertFrom-Json
$driftmgrCount = if ($driftmgrOutput.azure.resource_count) { $driftmgrOutput.azure.resource_count } else { 0 }

Write-Host "DriftMgr found: $driftmgrCount resources" -ForegroundColor Yellow

# Analysis
Write-Host ""
Write-Host "=== Reconciliation Report ===" -ForegroundColor Cyan
Write-Host "Actual Azure Resources: $totalResources" -ForegroundColor Green
Write-Host "DriftMgr Detected: $driftmgrCount" -ForegroundColor Yellow
Write-Host "Discrepancy: $($totalResources - $driftmgrCount) resources" -ForegroundColor Red

Write-Host ""
Write-Host "=== Resource Breakdown by Location ===" -ForegroundColor Cyan
$resourcesByLocation = $resourceDetails | Group-Object -Property Location
foreach ($location in $resourcesByLocation) {
    Write-Host "$($location.Name): $($location.Count) resources"
}

Write-Host ""
Write-Host "=== Issue Analysis ===" -ForegroundColor Cyan
Write-Host "DriftMgr is configured to scan only these Azure regions:" -ForegroundColor Yellow
Write-Host "  - eastus"
Write-Host "  - westus"
Write-Host "  - centralus"

Write-Host ""
Write-Host "Your resources are actually in:" -ForegroundColor Green
$actualLocations = $resourceDetails | Select-Object -ExpandProperty Location -Unique
foreach ($loc in $actualLocations) {
    $inDefaultRegions = @("eastus", "westus", "centralus") -contains $loc
    if ($inDefaultRegions) {
        Write-Host "  - $loc (scanned)" -ForegroundColor Green
    } else {
        Write-Host "  - $loc (NOT scanned)" -ForegroundColor Red
    }
}

Write-Host ""
Write-Host "=== Recommendations ===" -ForegroundColor Cyan
Write-Host "1. DriftMgr needs to be updated to scan all Azure regions dynamically"
Write-Host "2. The --regions flag should override default regions for Azure"
Write-Host "3. Consider using 'az account list-locations' to get all available regions"
Write-Host "4. For now, resources in polandcentral and mexicocentral won't be detected"

# Generate detailed JSON report
$report = @{
    timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    actual_resources = @{
        total = $totalResources
        by_subscription = @()
        by_location = @()
        details = $resourceDetails
    }
    driftmgr_results = @{
        detected = $driftmgrCount
        regions_scanned = @("eastus", "westus", "centralus")
    }
    discrepancy = @{
        missing_count = $totalResources - $driftmgrCount
        unscanned_regions = $actualLocations | Where-Object { @("eastus", "westus", "centralus") -notcontains $_ }
    }
}

# Add subscription breakdown
foreach ($sub in $subscriptions) {
    az account set --subscription $sub.id
    $subResources = az resource list --output json | ConvertFrom-Json
    $report.actual_resources.by_subscription += @{
        name = $sub.name
        id = $sub.id
        count = $subResources.Count
    }
}

# Add location breakdown
foreach ($location in $resourcesByLocation) {
    $report.actual_resources.by_location += @{
        location = $location.Name
        count = $location.Count
    }
}

# Save report
$report | ConvertTo-Json -Depth 10 | Out-File "azure_reconciliation_report.json"
Write-Host ""
Write-Host "Detailed report saved to: azure_reconciliation_report.json" -ForegroundColor Green