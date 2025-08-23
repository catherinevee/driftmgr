# PowerShell script to migrate fmt.Print to structured logging
param(
    [string]$RootPath = "C:\Users\cathe\OneDrive\Desktop\github\driftmgr"
)

Write-Host "Migrating fmt.Print statements to structured logging..." -ForegroundColor Green

# Get all Go files
$goFiles = Get-ChildItem -Path $RootPath -Filter "*.go" -Recurse | Where-Object { $_.FullName -notlike "*\vendor\*" }

$totalFiles = 0
$modifiedFiles = 0

foreach ($file in $goFiles) {
    $totalFiles++
    $content = Get-Content $file.FullName -Raw
    $originalContent = $content
    
    # Check if file has fmt.Print statements
    if ($content -match 'fmt\.Print') {
        Write-Host "Processing: $($file.FullName)" -ForegroundColor Yellow
        
        # Add logger import if not present
        if ($content -notmatch 'github\.com/catherinevee/driftmgr/internal/logger') {
            # Add import after package declaration
            $content = $content -replace '(package [^\n]+\n)', "`$1`nimport (`n`t`"github.com/catherinevee/driftmgr/internal/logger`"`n)`n"
        }
        
        # Replace fmt.Printf with logger.Printf
        $content = $content -replace 'fmt\.Printf\((.*?)\)', 'logger.Printf($1)'
        
        # Replace fmt.Println with logger.Println
        $content = $content -replace 'fmt\.Println\((.*?)\)', 'logger.Println($1)'
        
        # Replace fmt.Print with logger equivalent
        $content = $content -replace 'fmt\.Print\((.*?)\)', 'logger.Println($1)'
        
        # Save if modified
        if ($content -ne $originalContent) {
            Set-Content -Path $file.FullName -Value $content -NoNewline
            $modifiedFiles++
            Write-Host "  Modified: $($file.Name)" -ForegroundColor Green
        }
    }
}

Write-Host "`nMigration complete!" -ForegroundColor Green
Write-Host "Total files scanned: $totalFiles"
Write-Host "Files modified: $modifiedFiles"