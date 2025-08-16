# DriftMgr - Email Notification and Visualization Test Script
# Tests the new email notification provider and diagram generation features

param(
    [string]$StateFileID = "terraform",
    [string]$EmailRecipient = "test@example.com",
    [switch]$SkipEmail = $false
)

# Colors for output
$ColorRed = "Red"
$ColorGreen = "Green"
$ColorYellow = "Yellow"
$ColorCyan = "Cyan"
$ColorBlue = "Blue"

function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Color = "White"
    )
    Write-Host $Message -ForegroundColor $Color
}

function Test-ServerHealth {
    Write-ColorOutput "🔍 Testing server health..." $ColorCyan
    
    try {
        $response = Invoke-RestMethod -Uri "http://localhost:8080/health" -Method Get -TimeoutSec 5
        Write-ColorOutput "✅ Server is healthy: $($response.status)" $ColorGreen
        return $true
    }
    catch {
        Write-ColorOutput "❌ Server health check failed: $($_.Exception.Message)" $ColorRed
        return $false
    }
}

function Test-EmailNotification {
    Write-ColorOutput "📧 Testing email notification..." $ColorCyan
    
    $notificationRequest = @{
        type = "email"
        recipients = @($EmailRecipient)
        subject = "DriftMgr Test Notification"
        message = "This is a test notification from DriftMgr. Infrastructure drift has been detected."
        priority = "normal"
    }
    
    try {
        $jsonBody = $notificationRequest | ConvertTo-Json -Depth 3
        $response = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/notify" -Method Post -Body $jsonBody -ContentType "application/json" -TimeoutSec 30
        
        Write-ColorOutput "✅ Email notification sent successfully!" $ColorGreen
        Write-ColorOutput "   Message ID: $($response.message_id)" $ColorBlue
        Write-ColorOutput "   Sent at: $($response.sent_at)" $ColorBlue
        
        return $true
    }
    catch {
        Write-ColorOutput "❌ Email notification failed: $($_.Exception.Message)" $ColorRed
        Write-ColorOutput "   Note: Make sure SMTP settings are configured via environment variables:" $ColorYellow
        Write-ColorOutput "   - DRIFT_SMTP_HOST" $ColorYellow
        Write-ColorOutput "   - DRIFT_SMTP_PORT" $ColorYellow
        Write-ColorOutput "   - DRIFT_SMTP_USERNAME" $ColorYellow
        Write-ColorOutput "   - DRIFT_SMTP_PASSWORD" $ColorYellow
        Write-ColorOutput "   - DRIFT_FROM_EMAIL" $ColorYellow
        Write-ColorOutput "   - DRIFT_FROM_NAME" $ColorYellow
        return $false
    }
}

function Test-Visualization {
    Write-ColorOutput "📊 Testing infrastructure visualization..." $ColorCyan
    
    $visualizationRequest = @{
        state_file_id = $StateFileID
        terraform_path = "./terraform"
    }
    
    try {
        $jsonBody = $visualizationRequest | ConvertTo-Json -Depth 3
        $response = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/visualize" -Method Post -Body $jsonBody -ContentType "application/json" -TimeoutSec 60
        
        Write-ColorOutput "✅ Visualization generated successfully!" $ColorGreen
        Write-ColorOutput "   Duration: $($response.duration)" $ColorBlue
        Write-ColorOutput "   Total Resources: $($response.summary.total_resources)" $ColorBlue
        Write-ColorOutput "   Total Dependencies: $($response.summary.total_dependencies)" $ColorBlue
        Write-ColorOutput "   Complexity Score: $($response.summary.complexity_score)" $ColorBlue
        Write-ColorOutput "   Risk Level: $($response.summary.risk_level)" $ColorBlue
        
        Write-ColorOutput "📁 Generated outputs:" $ColorCyan
        foreach ($output in $response.outputs) {
            Write-ColorOutput "   - $($output.format): $($output.url)" $ColorBlue
        }
        
        return $true
    }
    catch {
        Write-ColorOutput "❌ Visualization failed: $($_.Exception.Message)" $ColorRed
        Write-ColorOutput "   Note: Make sure Graphviz is installed and state file exists" $ColorYellow
        return $false
    }
}

function Test-DiagramGeneration {
    Write-ColorOutput "🎨 Testing diagram generation..." $ColorCyan
    
    $diagramRequest = @{
        state_file_id = $StateFileID
        terraform_path = "./terraform"
    }
    
    try {
        $jsonBody = $diagramRequest | ConvertTo-Json -Depth 3
        $response = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/diagram" -Method Post -Body $jsonBody -ContentType "application/json" -TimeoutSec 60
        
        Write-ColorOutput "✅ Diagram generated successfully!" $ColorGreen
        Write-ColorOutput "   Status: $($response.status)" $ColorBlue
        Write-ColorOutput "   Duration: $($response.duration)" $ColorBlue
        Write-ColorOutput "   Resources: $($response.diagram_data.resources.Count)" $ColorBlue
        Write-ColorOutput "   Data Sources: $($response.diagram_data.data_sources.Count)" $ColorBlue
        Write-ColorOutput "   Dependencies: $($response.diagram_data.dependencies.Count)" $ColorBlue
        Write-ColorOutput "   Modules: $($response.diagram_data.modules.Count)" $ColorBlue
        
        return $true
    }
    catch {
        Write-ColorOutput "❌ Diagram generation failed: $($_.Exception.Message)" $ColorRed
        return $false
    }
}

function Test-ExportFunctionality {
    Write-ColorOutput "📤 Testing export functionality..." $ColorCyan
    
    $exportRequest = @{
        format = "png"
    }
    
    try {
        $jsonBody = $exportRequest | ConvertTo-Json -Depth 3
        $response = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/export" -Method Post -Body $jsonBody -ContentType "application/json" -TimeoutSec 30
        
        Write-ColorOutput "✅ Export completed successfully!" $ColorGreen
        Write-ColorOutput "   Format: $($response.format)" $ColorBlue
        Write-ColorOutput "   Output Path: $($response.output_path)" $ColorBlue
        Write-ColorOutput "   URL: $($response.url)" $ColorBlue
        
        return $true
    }
    catch {
        Write-ColorOutput "❌ Export failed: $($_.Exception.Message)" $ColorRed
        return $false
    }
}

function Show-ConfigurationHelp {
    Write-ColorOutput "`n📋 Configuration Help" $ColorCyan
    Write-ColorOutput "==================" $ColorCyan
    
    Write-ColorOutput "`n📧 Email Configuration (Environment Variables):" $ColorYellow
    Write-ColorOutput "   DRIFT_SMTP_HOST=smtp.gmail.com" $ColorBlue
    Write-ColorOutput "   DRIFT_SMTP_PORT=587" $ColorBlue
    Write-ColorOutput "   DRIFT_SMTP_USERNAME=your-email@gmail.com" $ColorBlue
    Write-ColorOutput "   DRIFT_SMTP_PASSWORD=your-app-password" $ColorBlue
    Write-ColorOutput "   DRIFT_FROM_EMAIL=driftmgr@yourdomain.com" $ColorBlue
    Write-ColorOutput "   DRIFT_FROM_NAME=DriftMgr" $ColorBlue
    Write-ColorOutput "   DRIFT_SMTP_TLS=true" $ColorBlue
    Write-ColorOutput "   DRIFT_SMTP_SSL=false" $ColorBlue
    
    Write-ColorOutput "`n🎨 Visualization Requirements:" $ColorYellow
    Write-ColorOutput "   - Graphviz must be installed on the system" $ColorBlue
    Write-ColorOutput "   - Terraform state file must exist" $ColorBlue
    Write-ColorOutput "   - Output directory must be writable" $ColorBlue
    
    Write-ColorOutput "`n🔧 Installation Commands:" $ColorYellow
    Write-ColorOutput "   # Install Graphviz (Windows with Chocolatey)" $ColorBlue
    Write-ColorOutput "   choco install graphviz" $ColorBlue
    Write-ColorOutput "   " $ColorBlue
    Write-ColorOutput "   # Install Graphviz (macOS with Homebrew)" $ColorBlue
    Write-ColorOutput "   brew install graphviz" $ColorBlue
    Write-ColorOutput "   " $ColorBlue
    Write-ColorOutput "   # Install Graphviz (Ubuntu/Debian)" $ColorBlue
    Write-ColorOutput "   sudo apt-get install graphviz" $ColorBlue
}

function Show-TestResults {
    param(
        [hashtable]$Results
    )
    
    Write-ColorOutput "`n📊 Test Results Summary" $ColorCyan
    Write-ColorOutput "=====================" $ColorCyan
    
    $totalTests = $Results.Count
    $passedTests = ($Results.Values | Where-Object { $_ -eq $true }).Count
    $failedTests = $totalTests - $passedTests
    
    Write-ColorOutput "Total Tests: $totalTests" $ColorBlue
    Write-ColorOutput "Passed: $passedTests" $ColorGreen
    Write-ColorOutput "Failed: $failedTests" $(if ($failedTests -gt 0) { $ColorRed } else { $ColorGreen })
    
    Write-ColorOutput "`nDetailed Results:" $ColorCyan
    foreach ($test in $Results.Keys) {
        $status = if ($Results[$test]) { "✅ PASS" } else { "❌ FAIL" }
        $color = if ($Results[$test]) { $ColorGreen } else { $ColorRed }
        Write-ColorOutput "   $test`: $status" $color
    }
    
    if ($failedTests -gt 0) {
        Write-ColorOutput "`n💡 Troubleshooting Tips:" $ColorYellow
        Write-ColorOutput "   - Check server logs for detailed error messages" $ColorBlue
        Write-ColorOutput "   - Verify all required dependencies are installed" $ColorBlue
        Write-ColorOutput "   - Ensure environment variables are properly set" $ColorBlue
        Write-ColorOutput "   - Check file permissions for output directory" $ColorBlue
    }
}

# Main execution
Write-ColorOutput "🚀 DriftMgr - Email Notification and Visualization Test" $ColorCyan
Write-ColorOutput "=====================================================" $ColorCyan
Write-ColorOutput "State File ID: $StateFileID" $ColorBlue
Write-ColorOutput "Email Recipient: $EmailRecipient" $ColorBlue
Write-ColorOutput "Skip Email: $SkipEmail" $ColorBlue
Write-ColorOutput ""

# Test results tracking
$testResults = @{}

# Test 1: Server Health
$testResults["Server Health"] = Test-ServerHealth

if (-not $testResults["Server Health"]) {
    Write-ColorOutput "❌ Server is not running. Please start the DriftMgr server first." $ColorRed
    Show-ConfigurationHelp
    exit 1
}

# Test 2: Email Notification (if not skipped)
if (-not $SkipEmail) {
    $testResults["Email Notification"] = Test-EmailNotification
} else {
    Write-ColorOutput "⏭️  Skipping email notification test" $ColorYellow
    $testResults["Email Notification"] = $true
}

# Test 3: Visualization
$testResults["Visualization"] = Test-Visualization

# Test 4: Diagram Generation
$testResults["Diagram Generation"] = Test-DiagramGeneration

# Test 5: Export Functionality
$testResults["Export Functionality"] = Test-ExportFunctionality

# Show results
Show-TestResults -Results $testResults

# Show configuration help if any tests failed
$failedTests = ($testResults.Values | Where-Object { $_ -eq $false }).Count
if ($failedTests -gt 0) {
    Show-ConfigurationHelp
}

Write-ColorOutput "`n✨ Test completed!" $ColorCyan
