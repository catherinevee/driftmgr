@echo off
REM Codecov Upload Test Script for DriftMgr (Windows)
REM This script validates the Codecov upload process locally

echo === Codecov Upload Test Script ===
echo This script tests the Codecov integration for DriftMgr

REM Check if we're in the right directory
if not exist "go.mod" (
    echo ERROR: Not in DriftMgr root directory. Please run from project root.
    exit /b 1
)
if not exist "internal" (
    echo ERROR: Not in DriftMgr root directory. Please run from project root.
    exit /b 1
)

echo INFO: Starting Codecov upload test...

REM Step 1: Check environment
echo INFO: Checking environment...

REM Check Go version
go version >nul 2>&1
if %errorlevel% neq 0 (
    echo ERROR: Go not found. Please install Go 1.23+
    exit /b 1
) else (
    for /f "tokens=3" %%i in ('go version') do set GO_VERSION=%%i
    echo SUCCESS: Go found: %GO_VERSION%
)

REM Check Git repository
git rev-parse --git-dir >nul 2>&1
if %errorlevel% neq 0 (
    echo ERROR: Not a Git repository
    exit /b 1
) else (
    echo SUCCESS: Git repository detected
    for /f %%i in ('git rev-parse --abbrev-ref HEAD') do set BRANCH=%%i
    for /f %%i in ('git rev-parse HEAD') do set COMMIT=%%i
    echo INFO: Branch: %BRANCH%
    echo INFO: Commit: %COMMIT:~0,8%
)

REM Step 2: Run tests and generate coverage
echo INFO: Running tests and generating coverage...

REM Clean up previous coverage files
if exist "coverage*.out" del /q coverage*.out
if exist "combined_coverage.out" del /q combined_coverage.out

REM Run a smaller subset of tests
echo INFO: Testing package: ./internal/state/backend
go test -v -race -coverprofile=backend_coverage.out -covermode=atomic ./internal/state/backend -timeout 15s >nul 2>&1
if %errorlevel% equ 0 (
    echo SUCCESS: Tests passed for backend package
) else (
    echo WARNING: Tests failed for backend package (continuing anyway)
)

echo INFO: Testing package: ./internal/providers/factory
go test -v -race -coverprofile=factory_coverage.out -covermode=atomic ./internal/providers/factory -timeout 15s >nul 2>&1
if %errorlevel% equ 0 (
    echo SUCCESS: Tests passed for factory package
) else (
    echo WARNING: Tests failed for factory package (continuing anyway)
)

REM Merge coverage files
echo INFO: Merging coverage files...
echo mode: atomic > combined_coverage.out

for %%f in (*_coverage.out) do (
    if exist "%%f" (
        more +1 "%%f" >> combined_coverage.out 2>nul
    )
)

REM Check if we have coverage data
if exist "combined_coverage.out" (
    for /f %%i in ('find /c /v "" ^< combined_coverage.out') do set COVERAGE_LINES=%%i
    echo SUCCESS: Coverage file generated with %COVERAGE_LINES% lines

    REM Generate coverage report
    go tool cover -func=combined_coverage.out > coverage_report.txt 2>nul
    if %errorlevel% equ 0 (
        for /f "tokens=3" %%i in ('type coverage_report.txt ^| find "total"') do set TOTAL_COVERAGE=%%i
        echo SUCCESS: Total coverage: %TOTAL_COVERAGE%
    ) else (
        echo WARNING: Could not generate coverage report
    )
) else (
    echo ERROR: No coverage data generated
    exit /b 1
)

REM Step 3: Check Codecov configuration
echo INFO: Checking Codecov configuration...

if exist "codecov.yml" (
    echo SUCCESS: codecov.yml found
) else (
    echo ERROR: codecov.yml not found
)

REM Step 4: Test Codecov upload
echo INFO: Testing Codecov upload process...

REM Check if CODECOV_TOKEN is set
if defined CODECOV_TOKEN (
    echo SUCCESS: CODECOV_TOKEN is set
    set TOKEN_FLAG=-t %CODECOV_TOKEN%
) else (
    echo WARNING: CODECOV_TOKEN not set (required for private repos)
    set TOKEN_FLAG=
)

REM Try to download codecov uploader if not present
if not exist "codecov.exe" (
    echo INFO: Downloading Codecov uploader...
    curl -Os https://cli.codecov.io/latest/windows/codecov.exe >nul 2>&1
    if %errorlevel% equ 0 (
        echo SUCCESS: Codecov uploader downloaded
    ) else (
        echo WARNING: Could not download Codecov uploader
        goto skip_upload_test
    )
)

REM Test upload (dry run)
echo INFO: Testing Codecov upload (dry run)...
codecov.exe --dry-run --file combined_coverage.out --flags unittests --name codecov-test-%time:~0,8% --verbose %TOKEN_FLAG% >nul 2>&1
if %errorlevel% equ 0 (
    echo SUCCESS: Codecov dry run completed
) else (
    echo WARNING: Codecov dry run had issues
)

:skip_upload_test

REM Step 5: GitHub Actions workflow validation
echo INFO: Checking GitHub Actions workflow...

if exist ".github\workflows\test-coverage.yml" (
    echo SUCCESS: test-coverage.yml workflow found

    findstr /c:"codecov/codecov-action@v4" ".github\workflows\test-coverage.yml" >nul 2>&1
    if %errorlevel% equ 0 (
        echo SUCCESS: Uses latest Codecov GitHub Action (v4)
    ) else (
        echo WARNING: May not be using latest Codecov GitHub Action
    )

    findstr /c:"CODECOV_TOKEN" ".github\workflows\test-coverage.yml" >nul 2>&1
    if %errorlevel% equ 0 (
        echo SUCCESS: CODECOV_TOKEN configured in workflow
    ) else (
        echo WARNING: CODECOV_TOKEN not found in workflow
    )
) else (
    echo ERROR: GitHub Actions workflow not found
)

REM Step 6: Generate summary
echo.
echo INFO: Test Summary:
echo.
echo Files Generated:
if exist "combined_coverage.out" echo   - combined_coverage.out (coverage data)
if exist "coverage_report.txt" echo   - coverage_report.txt (coverage report)
echo.

REM Cleanup option
set /p cleanup="Clean up generated files? (y/N): "
if /i "%cleanup%"=="y" (
    if exist "*_coverage.out" del /q *_coverage.out
    if exist "combined_coverage.out" del /q combined_coverage.out
    if exist "coverage_report.txt" del /q coverage_report.txt
    if exist "codecov.exe" del /q codecov.exe
    echo INFO: Cleanup completed
)

echo SUCCESS: Codecov test completed!
echo.
echo Next steps:
echo 1. Set CODECOV_TOKEN secret in GitHub repository settings
echo 2. Ensure your repository is connected to Codecov.io
echo 3. Run the test-coverage.yml GitHub Actions workflow
echo 4. Check Codecov dashboard for reports