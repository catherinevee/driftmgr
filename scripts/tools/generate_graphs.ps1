# Terraform Graph Generation Script for DriftMgr (PowerShell)
# Integrates multiple visualization tools for creating infrastructure diagrams

param(
    [string]$TerraformDir,
    [string]$ExportFile,
    [string]$OutputDir = "diagrams",
    [ValidateSet("dot", "png", "svg", "html", "all")]
    [string]$Format = "dot",
    [switch]$Serve,
    [string]$Host = "localhost",
    [int]$Port = 5000
)

function Write-Status {
    param([string]$Message, [string]$Type = "Info")
    
    switch ($Type) {
        "Success" { Write-Host "✅ $Message" -ForegroundColor Green }
        "Error" { Write-Host "❌ $Message" -ForegroundColor Red }
        "Warning" { Write-Host "⚠️ $Message" -ForegroundColor Yellow }
        default { Write-Host "ℹ️ $Message" -ForegroundColor Cyan }
    }
}

function Invoke-Command {
    param([string]$Command, [string]$WorkingDirectory = $null)
    
    try {
        $processInfo = New-Object System.Diagnostics.ProcessStartInfo
        $processInfo.FileName = "cmd.exe"
        $processInfo.Arguments = "/c $Command"
        $processInfo.UseShellExecute = $false
        $processInfo.RedirectStandardOutput = $true
        $processInfo.RedirectStandardError = $true
        
        if ($WorkingDirectory) {
            $processInfo.WorkingDirectory = $WorkingDirectory
        }
        
        $process = New-Object System.Diagnostics.Process
        $process.StartInfo = $processInfo
        $process.Start() | Out-Null
        
        $stdout = $process.StandardOutput.ReadToEnd()
        $stderr = $process.StandardError.ReadToEnd()
        $process.WaitForExit()
        
        return @{
            Success = $process.ExitCode -eq 0
            Output = $stdout
            Error = $stderr
        }
    }
    catch {
        return @{
            Success = $false
            Output = ""
            Error = $_.Exception.Message
        }
    }
}

function Generate-TerraformGraph {
    param([string]$TerraformDir, [string]$OutputFile)
    
    Write-Status "Generating Terraform graph for: $TerraformDir"
    
    $result = Invoke-Command "terraform graph" -WorkingDirectory $TerraformDir
    
    if ($result.Success) {
        $result.Output | Out-File -FilePath $OutputFile -Encoding UTF8
        Write-Status "Terraform graph saved to: $OutputFile" -Type Success
        return $true
    }
    else {
        Write-Status "Failed to generate Terraform graph: $($result.Error)" -Type Error
        return $false
    }
}

function Generate-BlastRadiusGraph {
    param([string]$TerraformDir, [string]$OutputDir, [string]$Format)
    
    Write-Status "Generating Blast Radius diagram for: $TerraformDir"
    
    # Check if blast_radius.py exists
    $blastRadiusPath = Join-Path (Split-Path $PSScriptRoot -Parent) "..\blast-radius\blast_radius.py"
    if (-not (Test-Path $blastRadiusPath)) {
        Write-Status "blast_radius.py not found. Please ensure it's available." -Type Error
        return $false
    }
    
    $cmd = "python `"$blastRadiusPath`" --export `"$TerraformDir`" --format $Format --output `"$OutputDir`""
    $result = Invoke-Command $cmd
    
    if ($result.Success) {
        Write-Status "Blast Radius diagram saved to: $OutputDir" -Type Success
        return $true
    }
    else {
        Write-Status "Failed to generate Blast Radius diagram: $($result.Error)" -Type Error
        return $false
    }
}

function Start-BlastRadiusServer {
    param([string]$TerraformDir, [string]$Host, [int]$Port)
    
    Write-Status "Starting Blast Radius web server for: $TerraformDir"
    
    $blastRadiusPath = Join-Path (Split-Path $PSScriptRoot -Parent) "..\blast-radius\blast_radius.py"
    if (-not (Test-Path $blastRadiusPath)) {
        Write-Status "blast_radius.py not found. Please ensure it's available." -Type Error
        return $false
    }
    
    $cmd = "python `"$blastRadiusPath`" --serve `"$TerraformDir`" --host $Host --port $Port"
    Write-Status "Starting web server at: http://$Host`:$Port" -Type Info
    Write-Status "Press Ctrl+C to stop the server" -Type Info
    
    try {
        Invoke-Command $cmd
    }
    catch {
        Write-Status "Server stopped" -Type Info
        return $true
    }
}

function Convert-DotToImage {
    param([string]$DotFile, [string]$OutputFormat)
    
    Write-Status "Converting $DotFile to $OutputFormat"
    
    # Check if dot command is available
    $result = Invoke-Command "dot -V"
    if (-not $result.Success) {
        Write-Status "GraphViz 'dot' command not found. Please install GraphViz." -Type Error
        Write-Status "Download from: https://graphviz.org/download/" -Type Info
        return $false
    }
    
    $outputFile = $DotFile -replace '\.dot$', ".$OutputFormat"
    $cmd = "dot -T$OutputFormat `"$DotFile`" -o `"$outputFile`""
    
    $result = Invoke-Command $cmd
    if ($result.Success) {
        Write-Status "Image saved to: $outputFile" -Type Success
        return $true
    }
    else {
        Write-Status "Failed to convert DOT file: $($result.Error)" -Type Error
        return $false
    }
}

function Generate-DriftMgrExportGraph {
    param([string]$ExportFile, [string]$OutputDir)
    
    Write-Status "Generating graph from driftmgr export: $ExportFile"
    
    if (-not (Test-Path $ExportFile)) {
        Write-Status "Export file not found: $ExportFile" -Type Error
        return $false
    }
    
    try {
        $data = Get-Content $ExportFile | ConvertFrom-Json
        
        # Create a simple DOT graph from the export data
        $dotContent = @"
digraph DriftMgrExport {
  rankdir=TB;
  node [shape=box, style=filled, fillcolor=lightblue];
"@
        
        foreach ($resource in $data) {
            $resourceId = if ($resource.id) { $resource.id } elseif ($resource.name) { $resource.name } else { "unknown" }
            $resourceType = if ($resource.type) { $resource.type } else { "unknown" }
            $dotContent += "`n  `"$resourceId`" [label=`"$resourceType`n$resourceId`"];"
        }
        
        $dotContent += "`n}"
        
        $outputFile = Join-Path $OutputDir "driftmgr-export.dot"
        New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
        
        $dotContent | Out-File -FilePath $outputFile -Encoding UTF8
        
        Write-Status "DriftMgr export graph saved to: $outputFile" -Type Success
        return $true
    }
    catch {
        Write-Status "Failed to generate export graph: $($_.Exception.Message)" -Type Error
        return $false
    }
}

# Main execution
if (-not $TerraformDir -and -not $ExportFile) {
    Write-Status "Please specify either -TerraformDir or -ExportFile" -Type Error
    Write-Host @"

Usage:
  .\generate_graphs.ps1 -TerraformDir "path/to/terraform" -Format "png"
  .\generate_graphs.ps1 -ExportFile "resources.json" -OutputDir "diagrams"
  .\generate_graphs.ps1 -TerraformDir "path/to/terraform" -Serve -Port 8080

Parameters:
  -TerraformDir: Path to Terraform directory
  -ExportFile: Path to driftmgr export file (JSON)
  -OutputDir: Output directory for diagrams (default: diagrams)
  -Format: Output format - dot, png, svg, html, all (default: dot)
  -Serve: Start web server for interactive visualization
  -Host: Web server host (default: localhost)
  -Port: Web server port (default: 5000)
"@
    exit 1
}

# Create output directory
if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
}

if ($TerraformDir) {
    if ($Serve) {
        Start-BlastRadiusServer -TerraformDir $TerraformDir -Host $Host -Port $Port
    }
    else {
        # Generate Terraform graph
        $dotFile = Join-Path $OutputDir "terraform-graph.dot"
        if (Generate-TerraformGraph -TerraformDir $TerraformDir -OutputFile $dotFile) {
            if ($Format -in @("png", "svg")) {
                Convert-DotToImage -DotFile $dotFile -OutputFormat $Format
            }
            elseif ($Format -eq "all") {
                Convert-DotToImage -DotFile $dotFile -OutputFormat "png"
                Convert-DotToImage -DotFile $dotFile -OutputFormat "svg"
            }
        }
        
        # Generate Blast Radius diagram
        if ($Format -in @("html", "all")) {
            Generate-BlastRadiusGraph -TerraformDir $TerraformDir -OutputDir $OutputDir -Format "html"
        }
    }
}

if ($ExportFile) {
    Generate-DriftMgrExportGraph -ExportFile $ExportFile -OutputDir $OutputDir
}
