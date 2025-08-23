# DriftMgr Data Verification Script
# This script verifies that DriftMgr is gathering and displaying the correct data
# by comparing its discovery results with direct AWS and Azure CLI queries.

param(
    [switch]$AWS,
    [switch]$Azure,
    [switch]$All,
    [switch]$Verbose,
    [switch]$Save,
    [string]$DriftMgrPath = "./driftmgr.exe"
)

# Set error action preference
$ErrorActionPreference = "Continue"

# Initialize results array
$Results = @()

function Write-Log {
    param([string]$Message)
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    Write-Host "[$timestamp] $Message"
}

function Test-Command {
    param([string]$Command, [string[]]$Arguments, [int]$TimeoutSeconds = 30)
    
    try {
        $process = Start-Process -FilePath $Command -ArgumentList $Arguments -Wait -PassThru -NoNewWindow -RedirectStandardOutput "temp_stdout.txt" -RedirectStandardError "temp_stderr.txt"
        
        $stdout = Get-Content "temp_stdout.txt" -Raw -ErrorAction SilentlyContinue
        $stderr = Get-Content "temp_stderr.txt" -Raw -ErrorAction SilentlyContinue
        
        # Clean up temp files
        Remove-Item "temp_stdout.txt" -ErrorAction SilentlyContinue
        Remove-Item "temp_stderr.txt" -ErrorAction SilentlyContinue
        
        return @{
            Success = $process.ExitCode -eq 0
            Stdout = $stdout
            Stderr = $stderr
            ExitCode = $process.ExitCode
        }
    }
    catch {
        return @{
            Success = $false
            Stdout = ""
            Stderr = $_.Exception.Message
            ExitCode = -1
        }
    }
}

function Test-AWSCredentials {
    Write-Log "🔐 Verifying AWS credentials..."
    
    $result = Test-Command -Command "aws" -Arguments @("sts", "get-caller-identity")
    
    if ($result.Success) {
        try {
            $identity = $result.Stdout | ConvertFrom-Json
            Write-Log "[OK] AWS credentials valid - Account: $($identity.Account)"
            return $true
        }
        catch {
            Write-Log "[OK] AWS credentials valid (non-JSON output)"
            return $true
        }
    }
    else {
        Write-Log "[ERROR] AWS credentials invalid: $($result.Stderr)"
        return $false
    }
}

function Test-AzureCredentials {
    Write-Log "🔐 Verifying Azure credentials..."
    
    $result = Test-Command -Command "az" -Arguments @("account", "show")
    
    if ($result.Success) {
        try {
            $account = $result.Stdout | ConvertFrom-Json
            Write-Log "[OK] Azure credentials valid - Subscription: $($account.name)"
            return $true
        }
        catch {
            Write-Log "[OK] Azure credentials valid (non-JSON output)"
            return $true
        }
    }
    else {
        Write-Log "[ERROR] Azure credentials invalid: $($result.Stderr)"
        return $false
    }
}

function Get-AWSRegions {
    Write-Log "🌍 Getting AWS regions..."
    
    $result = Test-Command -Command "aws" -Arguments @("ec2", "describe-regions", "--query", "Regions[*].RegionName", "--output", "json")
    
    if ($result.Success) {
        try {
            $regions = $result.Stdout | ConvertFrom-Json
            Write-Log "[OK] Found $($regions.Count) AWS regions"
            return $regions[0..2]  # Return first 3 regions
        }
        catch {
            Write-Log "[WARNING]  Using fallback AWS regions"
            return @("us-east-1", "us-west-2", "eu-west-1")
        }
    }
    else {
        Write-Log "[WARNING]  Using fallback AWS regions"
        return @("us-east-1", "us-west-2", "eu-west-1")
    }
}

function Get-AWSResourceCounts {
    param([string[]]$Regions)
    
    $counts = @{}
    
    # Get S3 buckets (global)
    Write-Log "   Getting S3 bucket count..."
    $result = Test-Command -Command "aws" -Arguments @("s3api", "list-buckets")
    if ($result.Success) {
        try {
            $data = $result.Stdout | ConvertFrom-Json
            $counts["s3"] = $data.Buckets.Count
        }
        catch {
            $counts["s3"] = 0
        }
    }
    else {
        $counts["s3"] = 0
    }
    
    # Get EC2 instances per region
    foreach ($region in $Regions) {
        Write-Log "   Getting EC2 instances in $region..."
        $result = Test-Command -Command "aws" -Arguments @("ec2", "describe-instances", "--region", $region, "--query", "Reservations[*].Instances[*]")
        if ($result.Success) {
            try {
                $data = $result.Stdout | ConvertFrom-Json
                # Flatten nested structure
                $instances = @()
                foreach ($reservation in $data) {
                    $instances += $reservation
                }
                $counts["ec2_$region"] = $instances.Count
            }
            catch {
                $counts["ec2_$region"] = 0
            }
        }
        else {
            $counts["ec2_$region"] = 0
        }
    }
    
    return $counts
}

function Get-AzureResourceCounts {
    param([string[]]$Regions)
    
    $counts = @{}
    
    foreach ($region in $Regions) {
        # Get VMs
        Write-Log "   Getting VMs in $region..."
        $result = Test-Command -Command "az" -Arguments @("vm", "list", "--resource-group", "*", "--query", "[?location=='$region']")
        if ($result.Success) {
            try {
                $data = $result.Stdout | ConvertFrom-Json
                $counts["vm_$region"] = $data.Count
            }
            catch {
                $counts["vm_$region"] = 0
            }
        }
        else {
            $counts["vm_$region"] = 0
        }
        
        # Get Storage Accounts
        Write-Log "   Getting Storage Accounts in $region..."
        $result = Test-Command -Command "az" -Arguments @("storage", "account", "list", "--query", "[?location=='$region']")
        if ($result.Success) {
            try {
                $data = $result.Stdout | ConvertFrom-Json
                $counts["storage_$region"] = $data.Count
            }
            catch {
                $counts["storage_$region"] = 0
            }
        }
        else {
            $counts["storage_$region"] = 0
        }
    }
    
    return $counts
}

function Test-DriftMgrExecutable {
    Write-Log "[LIGHTNING] Testing DriftMgr executable..."
    
    if (-not (Test-Path $DriftMgrPath)) {
        Write-Log "[ERROR] DriftMgr executable not found at $DriftMgrPath"
        return $false
    }
    
    $result = Test-Command -Command $DriftMgrPath -Arguments @("--help")
    if ($result.Success) {
        Write-Log "[OK] DriftMgr executable is working"
        return $true
    }
    else {
        Write-Log "[ERROR] DriftMgr executable failed: $($result.Stderr)"
        return $false
    }
}

function Run-DriftMgrDiscovery {
    param([string]$Provider, [string[]]$Regions = $null)
    
    Write-Log "🚀 Running DriftMgr discovery for $Provider..."
    
    $arguments = @("discover", "--provider", $Provider)
    if ($Regions) {
        $arguments += "--regions"
        $arguments += $Regions
    }
    if ($Verbose) {
        $arguments += "--verbose"
    }
    
    Write-Log "   Command: $DriftMgrPath $($arguments -join ' ')"
    
    $result = Test-Command -Command $DriftMgrPath -Arguments $arguments -TimeoutSeconds 300
    
    if ($result.Success) {
        Write-Log "[OK] DriftMgr discovery completed for $Provider"
        return $result
    }
    else {
        Write-Log "[ERROR] DriftMgr discovery failed for $Provider`: $($result.Stderr)"
        return $result
    }
}

function Parse-DriftMgrOutput {
    param([string]$Output)
    
    $counts = @{}
    
    # Look for patterns like "Found X resources" or "Discovered X Y"
    $patterns = @(
        "Found (\d+) (\w+)",
        "Discovered (\d+) (\w+)",
        "(\d+) (\w+) found",
        "(\d+) (\w+) discovered"
    )
    
    $lines = $Output -split "`n"
    foreach ($line in $lines) {
        foreach ($pattern in $patterns) {
            if ($line -match $pattern) {
                $count = [int]$matches[1]
                $resourceType = $matches[2].ToLower()
                $counts[$resourceType] = $count
                break
            }
        }
    }
    
    return $counts
}

function Test-AWSDiscovery {
    Write-Log "🔍 Starting AWS discovery verification..."
    
    if (-not (Test-AWSCredentials)) {
        Write-Log "[ERROR] AWS credentials not available, skipping AWS verification"
        return
    }
    
    $regions = Get-AWSRegions
    Write-Log "[OK] Testing $($regions.Count) AWS regions: $($regions -join ', ')"
    
    # Run DriftMgr discovery
    $driftmgrResult = Run-DriftMgrDiscovery -Provider "aws" -Regions $regions
    
    if (-not $driftmgrResult.Success) {
        Write-Log "[ERROR] DriftMgr AWS discovery failed"
        return
    }
    
    # Parse DriftMgr output
    $driftmgrCounts = Parse-DriftMgrOutput -Output $driftmgrResult.Stdout
    Write-Log "   DriftMgr found: $($driftmgrCounts | ConvertTo-Json -Compress)"
    
    # Get CLI counts
    $cliCounts = Get-AWSResourceCounts -Regions $regions
    Write-Log "   AWS CLI found: $($cliCounts | ConvertTo-Json -Compress)"
    
    # Compare results
    Compare-ResourceCounts -Provider "aws" -DriftMgrCounts $driftmgrCounts -CLICounts $cliCounts
}

function Test-AzureDiscovery {
    Write-Log "🔍 Starting Azure discovery verification..."
    
    if (-not (Test-AzureCredentials)) {
        Write-Log "[ERROR] Azure credentials not available, skipping Azure verification"
        return
    }
    
    $regions = @("eastus", "westus2", "centralus")
    Write-Log "[OK] Testing $($regions.Count) Azure regions: $($regions -join ', ')"
    
    # Run DriftMgr discovery
    $driftmgrResult = Run-DriftMgrDiscovery -Provider "azure" -Regions $regions
    
    if (-not $driftmgrResult.Success) {
        Write-Log "[ERROR] DriftMgr Azure discovery failed"
        return
    }
    
    # Parse DriftMgr output
    $driftmgrCounts = Parse-DriftMgrOutput -Output $driftmgrResult.Stdout
    Write-Log "   DriftMgr found: $($driftmgrCounts | ConvertTo-Json -Compress)"
    
    # Get CLI counts
    $cliCounts = Get-AzureResourceCounts -Regions $regions
    Write-Log "   Azure CLI found: $($cliCounts | ConvertTo-Json -Compress)"
    
    # Compare results
    Compare-ResourceCounts -Provider "azure" -DriftMgrCounts $driftmgrCounts -CLICounts $cliCounts
}

function Compare-ResourceCounts {
    param([string]$Provider, [hashtable]$DriftMgrCounts, [hashtable]$CLICounts)
    
    Write-Log "📊 Comparing $Provider resource counts..."
    
    $allKeys = $DriftMgrCounts.Keys + $CLICounts.Keys | Sort-Object -Unique
    
    foreach ($key in $allKeys) {
        $driftmgrCount = if ($DriftMgrCounts.ContainsKey($key)) { $DriftMgrCounts[$key] } else { 0 }
        $cliCount = if ($CLICounts.ContainsKey($key)) { $CLICounts[$key] } else { 0 }
        $match = $driftmgrCount -eq $cliCount
        
        $status = if ($match) { "[OK]" } else { "[ERROR]" }
        Write-Host "$status $key`: DriftMgr=$driftmgrCount, CLI=$cliCount"
        
        # Store result
        $result = [PSCustomObject]@{
            Provider = $Provider
            Service = $key
            Region = "multiple"
            CLICount = $cliCount
            DriftMgrCount = $driftmgrCount
            Match = $match
            Error = $null
        }
        $script:Results += $result
    }
}

function Generate-Report {
    Write-Log "📊 Generating verification report..."
    
    $totalChecks = $Results.Count
    $successfulMatches = ($Results | Where-Object { $_.Match }).Count
    $failedMatches = $totalChecks - $successfulMatches
    
    Write-Host ""
    Write-Host "=" * 80
    Write-Host "🔍 DRIFTMGR VERIFICATION REPORT"
    Write-Host "=" * 80
    Write-Host "Generated: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')"
    Write-Host "Total Checks: $totalChecks"
    Write-Host "Successful Matches: $successfulMatches"
    Write-Host "Failed Matches: $failedMatches"
    if ($totalChecks -gt 0) {
        $successRate = [math]::Round(($successfulMatches / $totalChecks) * 100, 1)
        Write-Host "Success Rate: $successRate%"
    }
    else {
        Write-Host "Success Rate: N/A"
    }
    
    Write-Host ""
    Write-Host "-" * 80
    Write-Host "DETAILED RESULTS"
    Write-Host "-" * 80
    
    foreach ($result in $Results) {
        $status = if ($result.Match) { "[OK]" } else { "[ERROR]" }
        Write-Host "$status $($result.Provider.ToUpper()) - $($result.Service)"
        Write-Host "   DriftMgr Count: $($result.DriftMgrCount)"
        Write-Host "   CLI Count: $($result.CLICount)"
        if ($result.Error) {
            Write-Host "   Error: $($result.Error)"
        }
        Write-Host ""
    }
    
    # Group by provider
    $awsResults = $Results | Where-Object { $_.Provider -eq "aws" }
    $azureResults = $Results | Where-Object { $_.Provider -eq "azure" }
    
    Write-Host "-" * 80
    Write-Host "PROVIDER SUMMARY"
    Write-Host "-" * 80
    
    if ($awsResults) {
        $awsMatches = ($awsResults | Where-Object { $_.Match }).Count
        Write-Host "AWS: $awsMatches/$($awsResults.Count) checks passed"
    }
    
    if ($azureResults) {
        $azureMatches = ($azureResults | Where-Object { $_.Match }).Count
        Write-Host "Azure: $azureMatches/$($azureResults.Count) checks passed"
    }
    
    Write-Host ""
    Write-Host "-" * 80
    Write-Host "RECOMMENDATIONS"
    Write-Host "-" * 80
    
    if ($failedMatches -gt 0) {
        Write-Host "[ERROR] Issues detected:"
        Write-Host "   • Some resource counts don't match between DriftMgr and CLI"
        Write-Host "   • Review DriftMgr discovery logic for affected services"
        Write-Host "   • Check for permission issues or API rate limits"
        Write-Host "   • Verify DriftMgr configuration and credentials"
    }
    else {
        Write-Host "[OK] All verifications passed!"
        Write-Host "   • DriftMgr is correctly discovering resources"
        Write-Host "   • Data accuracy is confirmed"
    }
    
    Write-Host ""
    Write-Host "=" * 80
}

function Save-Results {
    param([string]$Filename = "driftmgr_verification_results.json")
    
    $data = @{
        timestamp = Get-Date -Format "yyyy-MM-ddTHH:mm:ss"
        summary = @{
            total_checks = $Results.Count
            successful_matches = ($Results | Where-Object { $_.Match }).Count
            failed_matches = ($Results | Where-Object { -not $_.Match }).Count
        }
        results = $Results | ForEach-Object {
            @{
                provider = $_.Provider
                service = $_.Service
                region = $_.Region
                cli_count = $_.CLICount
                driftmgr_count = $_.DriftMgrCount
                match = $_.Match
                error = $_.Error
            }
        }
    }
    
    $data | ConvertTo-Json -Depth 10 | Out-File -FilePath $Filename -Encoding UTF8
    Write-Log "💾 Results saved to $Filename"
}

# Main execution
Write-Host "🔍 DriftMgr Data Verification Tool"
Write-Host "=" * 50

try {
    # Default to all if no specific provider specified
    if (-not ($AWS -or $Azure)) {
        $All = $true
    }
    
    # Test DriftMgr executable first
    if (-not (Test-DriftMgrExecutable)) {
        Write-Host "[ERROR] DriftMgr is not working properly. Please check the installation."
        exit 1
    }
    
    if ($All -or $AWS) {
        Test-AWSDiscovery
    }
    
    if ($All -or $Azure) {
        Test-AzureDiscovery
    }
    
    Generate-Report
    
    if ($Save) {
        Save-Results
    }
}
catch {
    Write-Host "[ERROR] Verification failed: $($_.Exception.Message)"
    exit 1
}
