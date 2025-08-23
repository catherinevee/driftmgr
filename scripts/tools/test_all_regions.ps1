# Test script for "all" regions functionality
Write-Host "Testing 'all' regions functionality..."

# Test different variations
$testCommands = @(
    "discover aws all",
    "discover aws ALL", 
    "discover aws all",
    "discover aws all",
    "exit"
)

$testCommands | Out-File -FilePath "test_commands.txt" -Encoding ASCII

Write-Host "Commands to execute:"
Get-Content "test_commands.txt" | ForEach-Object { Write-Host "  $_" }

# Run driftmgr with the commands
Write-Host "Executing driftmgr..."
Get-Content "test_commands.txt" | ./bin/driftmgr.exe

# Clean up
Remove-Item "test_commands.txt" -ErrorAction SilentlyContinue

Write-Host "Test completed."
