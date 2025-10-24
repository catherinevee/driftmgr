# Generate DriftMgr Architecture Diagrams (Windows PowerShell)
Write-Host "Generating DriftMgr architecture diagrams..." -ForegroundColor Green

# Create output directory
New-Item -ItemType Directory -Force -Path "output" | Out-Null

# Generate DOT files
Write-Host "Generating production architecture diagram..." -ForegroundColor Yellow
go run architecture_diagram.go

Write-Host "Generating real-time architecture diagram..." -ForegroundColor Yellow
go run realtime_architecture.go

Write-Host "Generating API architecture diagram..." -ForegroundColor Yellow
go run api_architecture.go

Write-Host "Generating drift detection flow diagram..." -ForegroundColor Yellow
go run drift_detection_flow.go

Write-Host "Generating remediation workflow diagram..." -ForegroundColor Yellow
go run remediation_workflow.go

# Convert DOT files to PNG (if Graphviz is available)
Write-Host "Converting DOT files to PNG..." -ForegroundColor Yellow
$dotFiles = @("driftmgr_production_architecture.dot", "driftmgr_realtime_architecture.dot", "driftmgr_api_architecture.dot", "drift_detection_flow.dot", "remediation_workflow.dot")

foreach ($dotFile in $dotFiles) {
    if (Test-Path $dotFile) {
        $pngFile = $dotFile -replace "\.dot$", ".png"
        try {
            dot -Tpng $dotFile -o $pngFile
            Write-Host "Generated $pngFile" -ForegroundColor Green
        } catch {
            Write-Host "Warning: Could not generate $pngFile (Graphviz not available?)" -ForegroundColor Yellow
        }
    }
}

# Move generated files to output directory
Get-ChildItem -Path "." -Include "*.png", "*.svg", "*.dot" | Move-Item -Destination "output" -Force

Write-Host "Diagrams generated successfully in output/ directory!" -ForegroundColor Green
Write-Host "Generated files:" -ForegroundColor Cyan
Get-ChildItem -Path "output" | Format-Table Name, Length, LastWriteTime

Write-Host "`nNote: To generate PNG files from DOT files, install Graphviz:" -ForegroundColor Yellow
Write-Host "  Windows: choco install graphviz" -ForegroundColor Gray
Write-Host "  macOS:   brew install graphviz" -ForegroundColor Gray
Write-Host "  Ubuntu:  sudo apt-get install graphviz" -ForegroundColor Gray