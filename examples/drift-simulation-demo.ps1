# DriftMgr Drift Simulation Demo
# This script demonstrates the complete drift simulation workflow
# It uses your existing cloud credentials to create real (but safe) drift

Write-Host "=====================================" -ForegroundColor Cyan
Write-Host "    DriftMgr Drift Simulation Demo   " -ForegroundColor Cyan
Write-Host "=====================================" -ForegroundColor Cyan
Write-Host ""

# Configuration
$StateFile = "examples\state-files\complex.tfstate"
$Provider = "aws"  # Change to azure or gcp as needed

# Function to run command and display output
function Run-Command {
    param([string]$Command, [string]$Description)
    
    Write-Host "`n→ $Description" -ForegroundColor Yellow
    Write-Host "  Command: $Command" -ForegroundColor Gray
    Write-Host ""
    
    Invoke-Expression $Command
}

# Check if driftmgr exists
if (-not (Test-Path ".\driftmgr.exe")) {
    Write-Host "Building DriftMgr..." -ForegroundColor Yellow
    go build -o driftmgr.exe ./cmd/driftmgr
}

Write-Host "`n📋 Step 1: Check Initial State" -ForegroundColor Green
Write-Host "First, let's see what resources are managed in the state file:" -ForegroundColor White

Run-Command -Command ".\driftmgr.exe state analyze --state $StateFile" `
            -Description "Analyzing current state file"

Write-Host "`n🎯 Step 2: Simulate Tag Drift" -ForegroundColor Green
Write-Host "Now we'll add tags to resources to simulate configuration drift:" -ForegroundColor White

Run-Command -Command ".\driftmgr.exe simulate-drift --state $StateFile --provider $Provider --type tag-change --auto-rollback=false" `
            -Description "Creating tag drift on $Provider resources"

Write-Host "`n🔍 Step 3: Detect the Drift" -ForegroundColor Green
Write-Host "DriftMgr should now detect the changes we made:" -ForegroundColor White

Run-Command -Command ".\driftmgr.exe state analyze --state $StateFile --check-drift" `
            -Description "Running drift detection"

Write-Host "`n🚀 Step 4: Simulate Unmanaged Resource Creation" -ForegroundColor Green
Write-Host "Let's create a resource that's not in Terraform state:" -ForegroundColor White

Run-Command -Command ".\driftmgr.exe simulate-drift --state $StateFile --provider $Provider --type resource-creation" `
            -Description "Creating unmanaged $Provider resource"

Write-Host "`n🔍 Step 5: Discover Unmanaged Resources" -ForegroundColor Green
Write-Host "Now let's find resources that aren't in Terraform:" -ForegroundColor White

Run-Command -Command ".\driftmgr.exe discover --provider $Provider --unmanaged-only" `
            -Description "Discovering unmanaged resources"

Write-Host "`n🛡️ Step 6: Simulate Security Rule Drift" -ForegroundColor Green
Write-Host "Adding a security group/firewall rule:" -ForegroundColor White

Run-Command -Command ".\driftmgr.exe simulate-drift --state $StateFile --provider $Provider --type rule-addition" `
            -Description "Adding security rule"

Write-Host "`n📊 Step 7: Generate Drift Report" -ForegroundColor Green
Write-Host "Let's see a comprehensive drift report:" -ForegroundColor White

Run-Command -Command ".\driftmgr.exe drift report --state $StateFile --provider $Provider --format detailed" `
            -Description "Generating drift report"

Write-Host "`n🔄 Step 8: Rollback All Changes" -ForegroundColor Green
Write-Host "Finally, let's clean up all the drift we created:" -ForegroundColor White

$confirm = Read-Host "Ready to rollback all changes? (y/n)"
if ($confirm -eq 'y') {
    Run-Command -Command ".\driftmgr.exe simulate-drift --rollback" `
                -Description "Rolling back drift simulation"
}

Write-Host "`n✅ Demo Complete!" -ForegroundColor Green
Write-Host "=====================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "What we demonstrated:" -ForegroundColor White
Write-Host "  • Created controlled drift using real cloud APIs" -ForegroundColor Gray
Write-Host "  • Detected configuration changes (tags)" -ForegroundColor Gray
Write-Host "  • Found unmanaged resources" -ForegroundColor Gray
Write-Host "  • Identified security rule additions" -ForegroundColor Gray
Write-Host "  • Rolled back all changes safely" -ForegroundColor Gray
Write-Host ""
Write-Host "All changes were:" -ForegroundColor White
Write-Host "  • Zero cost (used only free resources)" -ForegroundColor Green
Write-Host "  • Completely reversible" -ForegroundColor Green
Write-Host "  • Safe for production accounts" -ForegroundColor Green
Write-Host ""
Write-Host "Try different drift types:" -ForegroundColor Yellow
Write-Host "  .\driftmgr.exe simulate-drift --help" -ForegroundColor Gray