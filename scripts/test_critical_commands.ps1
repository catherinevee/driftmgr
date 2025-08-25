# Test Critical DriftMgr Commands
Write-Host '=== Testing Critical DriftMgr Commands ===' -ForegroundColor Cyan

$totalTests = 0
$passedTests = 0

# Test 1: Help
Write-Host 'Test 1: Help Command' -ForegroundColor Yellow
$totalTests++
$help = ./driftmgr.exe --help 2>&1 | Out-String
if ($help -match 'Core Commands') { 
    Write-Host '  ✓ PASS' -ForegroundColor Green
    $passedTests++
} else { 
    Write-Host '  ✗ FAIL' -ForegroundColor Red 
}

# Test 2: Status (with timeout)
Write-Host 'Test 2: Status Command (30s timeout)' -ForegroundColor Yellow
$totalTests++
$job = Start-Job { Set-Location $using:PWD; ./driftmgr.exe status 2>&1 }
$result = Wait-Job $job -Timeout 35
if ($result) { 
    $output = Receive-Job $job | Out-String
    if ($output -match 'System Status') { 
        Write-Host '  ✓ PASS' -ForegroundColor Green
        $passedTests++
    } else { 
        Write-Host '  ✗ FAIL' -ForegroundColor Red 
    }
} else { 
    Stop-Job $job
    Write-Host '  ✗ FAIL (Timeout)' -ForegroundColor Red 
}
Remove-Job $job -Force | Out-Null

# Test 3: Discover
Write-Host 'Test 3: Discover Command' -ForegroundColor Yellow
$totalTests++
$discover = ./driftmgr.exe discover --help 2>&1 | Out-String
if ($discover -match 'Discover cloud resources') { 
    Write-Host '  ✓ PASS' -ForegroundColor Green
    $passedTests++
} else { 
    Write-Host '  ✗ FAIL' -ForegroundColor Red 
}

# Test 4: Drift
Write-Host 'Test 4: Drift Command' -ForegroundColor Yellow
$totalTests++
$drift = ./driftmgr.exe drift --help 2>&1 | Out-String
if ($drift -match 'detect') { 
    Write-Host '  ✓ PASS' -ForegroundColor Green
    $passedTests++
} else { 
    Write-Host '  ✗ FAIL' -ForegroundColor Red 
}

# Test 5: State
Write-Host 'Test 5: State Command' -ForegroundColor Yellow
$totalTests++
$state = ./driftmgr.exe state --help 2>&1 | Out-String
if ($state -match 'inspect') { 
    Write-Host '  ✓ PASS' -ForegroundColor Green
    $passedTests++
} else { 
    Write-Host '  ✗ FAIL' -ForegroundColor Red 
}

# Test 6: Export
Write-Host 'Test 6: Export Command' -ForegroundColor Yellow
$totalTests++
$export = ./driftmgr.exe export --help 2>&1 | Out-String
if ($export -match 'Export discovery results') { 
    Write-Host '  ✓ PASS' -ForegroundColor Green
    $passedTests++
} else { 
    Write-Host '  ✗ FAIL' -ForegroundColor Red 
}

# Test 7: Import
Write-Host 'Test 7: Import Command' -ForegroundColor Yellow
$totalTests++
$import = ./driftmgr.exe import --help 2>&1 | Out-String
if ($import -match 'Import existing') { 
    Write-Host '  ✓ PASS' -ForegroundColor Green
    $passedTests++
} else { 
    Write-Host '  ✗ FAIL' -ForegroundColor Red 
}

# Test 8: Use
Write-Host 'Test 8: Use Command' -ForegroundColor Yellow
$totalTests++
$use = ./driftmgr.exe use --help 2>&1 | Out-String
if ($use -match 'Select which cloud') { 
    Write-Host '  ✓ PASS' -ForegroundColor Green
    $passedTests++
} else { 
    Write-Host '  ✗ FAIL' -ForegroundColor Red 
}

# Test 9: Verify
Write-Host 'Test 9: Verify Command' -ForegroundColor Yellow
$totalTests++
$verify = ./driftmgr.exe verify --help 2>&1 | Out-String
if ($verify -match 'Verify discovery') { 
    Write-Host '  ✓ PASS' -ForegroundColor Green
    $passedTests++
} else { 
    Write-Host '  ✗ FAIL' -ForegroundColor Red 
}

# Test 10: Unknown Command
Write-Host 'Test 10: Unknown Command Error' -ForegroundColor Yellow
$totalTests++
$unknown = ./driftmgr.exe unknowncommand 2>&1 | Out-String
if ($unknown -match 'Unknown command') { 
    Write-Host '  ✓ PASS' -ForegroundColor Green
    $passedTests++
} else { 
    Write-Host '  ✗ FAIL' -ForegroundColor Red 
}

# Test 11: Invalid Flag
Write-Host 'Test 11: Invalid Flag Error' -ForegroundColor Yellow
$totalTests++
$invalid = ./driftmgr.exe --invalidflag 2>&1 | Out-String
if ($invalid -match 'Unknown flag') { 
    Write-Host '  ✓ PASS' -ForegroundColor Green
    $passedTests++
} else { 
    Write-Host '  ✗ FAIL' -ForegroundColor Red 
}

# Test 12: Credentials
Write-Host 'Test 12: Credentials Command' -ForegroundColor Yellow
$totalTests++
$creds = ./driftmgr.exe credentials 2>&1 | Out-String
if ($creds -match 'deprecated' -or $creds -match 'Configured') { 
    Write-Host '  ✓ PASS' -ForegroundColor Green
    $passedTests++
} else { 
    Write-Host '  ✗ FAIL' -ForegroundColor Red 
}

# Test 13: Accounts
Write-Host 'Test 13: Accounts Command' -ForegroundColor Yellow
$totalTests++
$accounts = ./driftmgr.exe accounts 2>&1 | Out-String
if ($accounts -match 'Accessible Cloud Accounts' -or $accounts -match 'AWS' -or $accounts -match 'Azure') { 
    Write-Host '  ✓ PASS' -ForegroundColor Green
    $passedTests++
} else { 
    Write-Host '  ✗ FAIL' -ForegroundColor Red 
}

# Test 14: Delete Help
Write-Host 'Test 14: Delete Command' -ForegroundColor Yellow
$totalTests++
$delete = ./driftmgr.exe delete --help 2>&1 | Out-String
if ($delete -match 'Delete' -or $delete -match 'remove') { 
    Write-Host '  ✓ PASS' -ForegroundColor Green
    $passedTests++
} else { 
    Write-Host '  ✗ FAIL' -ForegroundColor Red 
}

# Test 15: Serve Help
Write-Host 'Test 15: Serve Command' -ForegroundColor Yellow
$totalTests++
$serve = ./driftmgr.exe serve --help 2>&1 | Out-String
if ($serve -match 'server' -or $serve -match 'dashboard') { 
    Write-Host '  ✓ PASS' -ForegroundColor Green
    $passedTests++
} else { 
    Write-Host '  ✗ FAIL' -ForegroundColor Red 
}

Write-Host "`n=== Test Summary ===" -ForegroundColor Cyan
Write-Host "Total Tests: $totalTests" -ForegroundColor White
Write-Host "Passed: $passedTests" -ForegroundColor Green
Write-Host "Failed: $($totalTests - $passedTests)" -ForegroundColor Red
$passRate = [math]::Round(($passedTests / $totalTests) * 100, 2)
Write-Host "Pass Rate: $passRate%" -ForegroundColor $(if ($passRate -eq 100) { 'Green' } elseif ($passRate -ge 80) { 'Yellow' } else { 'Red' })

if ($passRate -eq 100) {
    Write-Host "`n✅ All critical commands are working correctly!" -ForegroundColor Green
} elseif ($passRate -ge 80) {
    Write-Host "`n⚠️ Most commands working, but some issues detected." -ForegroundColor Yellow
} else {
    Write-Host "`n❌ Multiple command failures detected." -ForegroundColor Red
}