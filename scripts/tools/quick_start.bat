@echo off
echo.
echo ================================
echo    DriftMgr Quick Start Guide
echo ================================
echo.
echo DriftMgr is now available with both interactive and command-line interfaces:
echo.
echo   driftmgr.exe - Interactive TUI mode (run without arguments)
echo   driftmgr.exe [OPTIONS] - Command-line mode (with arguments)
echo.
echo ================================
echo      Interactive TUI Mode
echo ================================
echo.
echo # Launch interactive interface
echo driftmgr.exe
echo.
echo Navigate through menus to:
echo - Discover cloud resources
echo - Analyze costs
echo - Export results
echo - Configure settings
echo.
echo ================================
echo      Command-Line Mode
echo ================================
echo.
echo # Basic resource discovery
echo driftmgr.exe --provider gcp --format summary
echo.
echo # Discovery with cost analysis
echo driftmgr.exe --provider gcp --cost-analysis --format summary
echo.
echo # Export to CSV
echo driftmgr.exe --provider gcp --cost-analysis --export csv
echo.
echo # Export to HTML report
echo driftmgr.exe --provider gcp --cost-analysis --export html
echo.
echo # Test different providers (requires credentials)
echo driftmgr.exe --provider aws --format summary
echo driftmgr.exe --provider azure --format summary
echo driftmgr.exe --provider digitalocean --format summary
echo.
echo ================================
echo         Key Features
echo ================================
echo.
echo [OK] Multi-cloud discovery (AWS, Azure, GCP, DigitalOcean)
echo [OK] Cost analysis with realistic pricing
echo [OK] Export formats (CSV, HTML, JSON, Excel)
echo [OK] Account auto-discovery
echo [OK] Resource type detection
echo [OK] Professional reports
echo.
echo ================================
echo      Performance Results
echo ================================
echo.
echo GCP Discovery: 21 resources in ~50 seconds
echo Cost Analysis: $1.46/month estimation
echo Export Speed: 2-15ms depending on format
echo.
echo ================================
echo     Using the Tools
echo ================================
echo.
echo To use DriftMgr, simply run:
echo.
echo   driftmgr.exe --provider [PROVIDER] [OPTIONS]
echo.
echo Available providers: aws, azure, gcp, digitalocean
echo Available formats: summary, json
echo Export formats: csv, html, json, excel
echo.
echo For help: driftmgr.exe --help
echo.
pause