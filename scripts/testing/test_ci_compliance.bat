@echo off
echo ========================================
echo Testing GitHub Actions CI Compliance
echo ========================================
echo.

set PASS=0
set FAIL=0

echo [1/7] Checking Go Module Integrity...
go mod download 2>nul
if %errorlevel% equ 0 (
    echo [PASS] Go modules downloaded
    set /a PASS+=1
) else (
    echo [FAIL] Go module download failed
    set /a FAIL+=1
)

echo.
echo [2/7] Running go vet...
go vet ./... 2>vet_errors.txt
if %errorlevel% equ 0 (
    echo [PASS] go vet passed
    set /a PASS+=1
) else (
    echo [FAIL] go vet found issues:
    type vet_errors.txt | findstr /N "^" | findstr "^[1-5]:"
    set /a FAIL+=1
)

echo.
echo [3/7] Checking code formatting...
gofmt -l . > fmt_issues.txt
for %%A in (fmt_issues.txt) do set size=%%~zA
if %size% equ 0 (
    echo [PASS] Code is properly formatted
    set /a PASS+=1
) else (
    echo [FAIL] Code formatting issues in:
    type fmt_issues.txt | findstr /N "^" | findstr "^[1-5]:"
    set /a FAIL+=1
)

echo.
echo [4/7] Running tests...
go test ./... -v -short 2>test_errors.txt >test_output.txt
if %errorlevel% equ 0 (
    echo [PASS] All tests passed
    set /a PASS+=1
) else (
    echo [FAIL] Tests failed
    type test_errors.txt | findstr /N "^" | findstr "^[1-5]:"
    set /a FAIL+=1
)

echo.
echo [5/7] Building main binary...
go build -o driftmgr.exe ./cmd/driftmgr 2>build_errors.txt
if %errorlevel% equ 0 (
    echo [PASS] Main binary built successfully
    set /a PASS+=1
) else (
    echo [FAIL] Build failed
    type build_errors.txt | findstr /N "^" | findstr "^[1-5]:"
    set /a FAIL+=1
)

echo.
echo [6/7] Building server binary...
go build -o driftmgr-server.exe ./cmd/server 2>server_errors.txt
if exist ./cmd/server (
    if %errorlevel% equ 0 (
        echo [PASS] Server binary built
        set /a PASS+=1
    ) else (
        echo [FAIL] Server build failed
        set /a FAIL+=1
    )
) else (
    echo [SKIP] No server command found
)

echo.
echo [7/7] Checking for common issues...
findstr /S /C:"fmt.Println" *.go >debug_prints.txt 2>nul
for %%A in (debug_prints.txt) do set size=%%~zA
if %size% gtr 100 (
    echo [WARN] Debug print statements found
) else (
    echo [PASS] No debug prints
    set /a PASS+=1
)

echo.
echo ========================================
echo CI COMPLIANCE SUMMARY
echo ========================================
echo Passed: %PASS%
echo Failed: %FAIL%
echo.

if %FAIL% equ 0 (
    echo RESULT: READY FOR CI/CD
    echo DriftMgr would PASS GitHub Actions workflow
    del vet_errors.txt fmt_issues.txt test_errors.txt test_output.txt build_errors.txt server_errors.txt debug_prints.txt 2>nul
    exit /b 0
) else (
    echo RESULT: NOT CI COMPLIANT
    echo DriftMgr would FAIL GitHub Actions workflow
    echo.
    echo Fix the above issues before pushing to GitHub
    exit /b 1
)