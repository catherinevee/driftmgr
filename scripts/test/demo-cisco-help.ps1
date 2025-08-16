# Demo Script for DriftMgr Context-Sensitive "?" Help Functionality
Write-Host "DriftMgr Context-Sensitive Help Demo" -ForegroundColor Cyan
Write-Host "====================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "This demo shows the context-sensitive '?' help functionality:" -ForegroundColor Yellow
Write-Host "1. Type '?' to see all available commands" -ForegroundColor White
Write-Host "2. Type 'command ?' to see detailed help for a specific command" -ForegroundColor White
Write-Host "3. Type 'partial ?' to see suggestions for partial commands" -ForegroundColor White
Write-Host "4. Type 'invalid ?' to see error handling" -ForegroundColor White
Write-Host ""

Write-Host "Example 1: Basic '?' help" -ForegroundColor Green
Write-Host "Command: ?" -ForegroundColor Gray
echo "?" | ./cmd/driftmgr-client/driftmgr-client.exe
Write-Host ""

Write-Host "Example 2: Command-specific help" -ForegroundColor Green
Write-Host "Command: discover ?" -ForegroundColor Gray
echo "discover ?" | ./cmd/driftmgr-client/driftmgr-client.exe
Write-Host ""

Write-Host "Example 3: Another command help" -ForegroundColor Green
Write-Host "Command: analyze ?" -ForegroundColor Gray
echo "analyze ?" | ./cmd/driftmgr-client/driftmgr-client.exe
Write-Host ""

Write-Host "Example 4: Partial command matching" -ForegroundColor Green
Write-Host "Command: disc ?" -ForegroundColor Gray
echo "disc ?" | ./cmd/driftmgr-client/driftmgr-client.exe
Write-Host ""

Write-Host "Example 5: Invalid command handling" -ForegroundColor Green
Write-Host "Command: invalid ?" -ForegroundColor Gray
echo "invalid ?" | ./cmd/driftmgr-client/driftmgr-client.exe
Write-Host ""

Write-Host "Context-sensitive '?' help functionality successfully implemented!" -ForegroundColor Green
Write-Host "This provides interactive help for better user experience." -ForegroundColor Yellow
