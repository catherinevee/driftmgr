# DriftMgr Timeout Configuration Script
# This script helps set appropriate timeout values for different scenarios

param(
    [Parameter(Mandatory=$false)]
    [ValidateSet("dev", "small", "large", "multi-cloud", "custom")]
    [string]$Scenario = "large",
    
    [Parameter(Mandatory=$false)]
    [string]$ClientTimeout,
    
    [Parameter(Mandatory=$false)]
    [string]$DiscoveryTimeout
)

Write-Host "DriftMgr Timeout Configuration Script" -ForegroundColor Cyan
Write-Host "=====================================" -ForegroundColor Cyan
Write-Host ""

# Function to set environment variables
function Set-TimeoutVariables {
    param(
        [string]$ClientTimeout,
        [string]$DiscoveryTimeout
    )
    
    if ($ClientTimeout) {
        $env:DRIFT_CLIENT_TIMEOUT = $ClientTimeout
        Write-Host "Set DRIFT_CLIENT_TIMEOUT = $ClientTimeout" -ForegroundColor Green
    }
    
    if ($DiscoveryTimeout) {
        $env:DRIFT_DISCOVERY_TIMEOUT = $DiscoveryTimeout
        Write-Host "Set DRIFT_DISCOVERY_TIMEOUT = $DiscoveryTimeout" -ForegroundColor Green
    }
}

# Function to show current settings
function Show-CurrentSettings {
    Write-Host "Current timeout settings:" -ForegroundColor Yellow
    Write-Host "  DRIFT_CLIENT_TIMEOUT: $($env:DRIFT_CLIENT_TIMEOUT)" -ForegroundColor White
    Write-Host "  DRIFT_DISCOVERY_TIMEOUT: $($env:DRIFT_DISCOVERY_TIMEOUT)" -ForegroundColor White
    Write-Host ""
}

# Function to show recommended settings
function Show-RecommendedSettings {
    param([string]$Scenario)
    
    Write-Host "Recommended settings for '$Scenario' scenario:" -ForegroundColor Yellow
    
    switch ($Scenario) {
        "dev" {
            Write-Host "  DRIFT_CLIENT_TIMEOUT: 1m" -ForegroundColor White
            Write-Host "  DRIFT_DISCOVERY_TIMEOUT: 2m" -ForegroundColor White
            Write-Host "  Use case: Development and testing with small infrastructure" -ForegroundColor Gray
        }
        "small" {
            Write-Host "  DRIFT_CLIENT_TIMEOUT: 2m" -ForegroundColor White
            Write-Host "  DRIFT_DISCOVERY_TIMEOUT: 5m" -ForegroundColor White
            Write-Host "  Use case: Production with small infrastructure (< 100 resources)" -ForegroundColor Gray
        }
        "large" {
            Write-Host "  DRIFT_CLIENT_TIMEOUT: 5m" -ForegroundColor White
            Write-Host "  DRIFT_DISCOVERY_TIMEOUT: 10m" -ForegroundColor White
            Write-Host "  Use case: Production with large infrastructure (> 100 resources)" -ForegroundColor Gray
        }
        "multi-cloud" {
            Write-Host "  DRIFT_CLIENT_TIMEOUT: 5m" -ForegroundColor White
            Write-Host "  DRIFT_DISCOVERY_TIMEOUT: 15m" -ForegroundColor White
            Write-Host "  Use case: Multi-cloud or all regions discovery" -ForegroundColor Gray
        }
        "custom" {
            Write-Host "  Use -ClientTimeout and -DiscoveryTimeout parameters to set custom values" -ForegroundColor White
        }
    }
    Write-Host ""
}

# Show current settings
Show-CurrentSettings

# Show recommended settings for the scenario
Show-RecommendedSettings -Scenario $Scenario

# Apply settings based on scenario
switch ($Scenario) {
    "dev" {
        Set-TimeoutVariables -ClientTimeout "1m" -DiscoveryTimeout "2m"
    }
    "small" {
        Set-TimeoutVariables -ClientTimeout "2m" -DiscoveryTimeout "5m"
    }
    "large" {
        Set-TimeoutVariables -ClientTimeout "5m" -DiscoveryTimeout "10m"
    }
    "multi-cloud" {
        Set-TimeoutVariables -ClientTimeout "5m" -DiscoveryTimeout "15m"
    }
    "custom" {
        if ($ClientTimeout -or $DiscoveryTimeout) {
            Set-TimeoutVariables -ClientTimeout $ClientTimeout -DiscoveryTimeout $DiscoveryTimeout
        } else {
            Write-Host "For custom timeouts, use -ClientTimeout and -DiscoveryTimeout parameters" -ForegroundColor Red
            Write-Host "Example: .\set-timeout.ps1 -Scenario custom -ClientTimeout '3m' -DiscoveryTimeout '8m'" -ForegroundColor Gray
            exit 1
        }
    }
}

Write-Host "Timeout configuration completed!" -ForegroundColor Green
Write-Host ""
Write-Host "You can now run the driftmgr client:" -ForegroundColor Cyan
Write-Host "  .\bin\driftmgr-client.exe" -ForegroundColor White
Write-Host ""
Write-Host "Note: These settings are only valid for the current PowerShell session." -ForegroundColor Yellow
Write-Host "To make them permanent, add them to your PowerShell profile or use system environment variables." -ForegroundColor Yellow
