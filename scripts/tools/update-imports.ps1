# Update Go imports after refactoring
# PowerShell version for Windows

Write-Host "Updating Go imports after refactoring..." -ForegroundColor Green

# Define the base module path
$MODULE = "github.com/catherinevee/driftmgr"

# Function to update imports in a file
function Update-FileImports {
    param (
        [string]$FilePath
    )
    
    Write-Host "Processing: $FilePath" -ForegroundColor Yellow
    
    # Read the file content
    $content = Get-Content $FilePath -Raw
    $originalContent = $content
    
    # Update discovery imports
    $content = $content -replace "$MODULE/internal/discovery/enhanced_discovery", "$MODULE/internal/core/discovery"
    $content = $content -replace "$MODULE/internal/discovery/universal_discovery", "$MODULE/internal/core/discovery"
    $content = $content -replace "$MODULE/internal/discovery/multi_account_discovery", "$MODULE/internal/core/discovery"
    $content = $content -replace "$MODULE/internal/discovery/enhanced_discovery_v2", "$MODULE/internal/core/discovery"
    $content = $content -replace "$MODULE/internal/discovery`"", "$MODULE/internal/core/discovery`""
    
    # Update drift imports
    $content = $content -replace "$MODULE/internal/drift", "$MODULE/internal/core/drift"
    
    # Update remediation imports
    $content = $content -replace "$MODULE/internal/remediation", "$MODULE/internal/core/remediation"
    
    # Update visualization imports
    $content = $content -replace "$MODULE/internal/visualization", "$MODULE/internal/core/visualization"
    
    # Update dashboard imports to API
    $content = $content -replace "$MODULE/internal/dashboard", "$MODULE/internal/api/rest"
    
    # Update provider-specific imports
    $content = $content -replace "$MODULE/internal/discovery/aws_", "$MODULE/internal/providers/aws/"
    $content = $content -replace "$MODULE/internal/discovery/azure_", "$MODULE/internal/providers/azure/"
    $content = $content -replace "$MODULE/internal/discovery/gcp_", "$MODULE/internal/providers/gcp/"
    $content = $content -replace "$MODULE/internal/discovery/digitalocean_", "$MODULE/internal/providers/digitalocean/"
    
    # Update cost analyzer imports
    $content = $content -replace "$MODULE/internal/cost/cost_analyzer", "$MODULE/internal/analytics/cost/analyzer"
    $content = $content -replace "$MODULE/internal/analysis/cost_analyzer", "$MODULE/internal/analytics/cost/analyzer"
    
    # Write back if changed
    if ($content -ne $originalContent) {
        Set-Content -Path $FilePath -Value $content -NoNewline
        Write-Host "  Updated imports in $FilePath" -ForegroundColor Green
        return $true
    }
    return $false
}

# Find all Go files
Write-Host "Finding all Go files..." -ForegroundColor Yellow
$goFiles = Get-ChildItem -Path . -Filter "*.go" -Recurse | 
    Where-Object { $_.FullName -notmatch "\\vendor\\" -and $_.FullName -notmatch "\\.git\\" }

$total = $goFiles.Count
$current = 0
$updated = 0

Write-Host "Found $total Go files to process" -ForegroundColor Green
Write-Host ""

# Process each file
foreach ($file in $goFiles) {
    $current++
    Write-Host "[$current/$total] Processing $($file.FullName)"
    if (Update-FileImports -FilePath $file.FullName) {
        $updated++
    }
}

Write-Host ""
Write-Host "Updated $updated files" -ForegroundColor Green

# Run go mod tidy
Write-Host ""
Write-Host "Running 'go mod tidy' to clean up dependencies..." -ForegroundColor Yellow
go mod tidy

# Check for compilation errors
Write-Host ""
Write-Host "Checking for compilation errors..." -ForegroundColor Yellow
$buildOutput = go build ./... 2>&1

if ($buildOutput -match "error") {
    Write-Host "Compilation errors found! Please review and fix manually." -ForegroundColor Red
    Write-Host $buildOutput
} else {
    Write-Host "No compilation errors found!" -ForegroundColor Green
}

# Run go fmt
Write-Host ""
Write-Host "Running 'go fmt' to format code..." -ForegroundColor Yellow
go fmt ./...

Write-Host ""
Write-Host "Refactoring import updates complete!" -ForegroundColor Green
Write-Host "Please review the changes and run tests to ensure everything works correctly." -ForegroundColor Yellow