Write-Host "=== Checking Cloud Provider Credentials ===" -ForegroundColor Cyan
Write-Host ""

# Check AWS
Write-Host "AWS:" -ForegroundColor Yellow
if ($env:AWS_ACCESS_KEY_ID) {
    Write-Host "  ✓ AWS_ACCESS_KEY_ID set" -ForegroundColor Green
}
if ($env:AWS_SECRET_ACCESS_KEY) {
    Write-Host "  ✓ AWS_SECRET_ACCESS_KEY set" -ForegroundColor Green
}
if ($env:AWS_PROFILE) {
    Write-Host "  ✓ AWS_PROFILE: $($env:AWS_PROFILE)" -ForegroundColor Green
}
$awsConfigPath = "$env:USERPROFILE\.aws\config"
if (Test-Path $awsConfigPath) {
    Write-Host "  ✓ AWS config file exists" -ForegroundColor Green
    $profiles = Get-Content $awsConfigPath | Select-String "\[.*\]" | ForEach-Object { $_.ToString().Trim('[]') }
    if ($profiles) {
        Write-Host "  ✓ Profiles found: $($profiles -join ', ')" -ForegroundColor Green
    }
}
$awsCredsPath = "$env:USERPROFILE\.aws\credentials"
if (Test-Path $awsCredsPath) {
    Write-Host "  ✓ AWS credentials file exists" -ForegroundColor Green
}

# Check Azure
Write-Host "`nAzure:" -ForegroundColor Yellow
try {
    $azAccount = az account show 2>$null | ConvertFrom-Json
    if ($azAccount) {
        Write-Host "  ✓ Logged in as: $($azAccount.user.name)" -ForegroundColor Green
        Write-Host "  ✓ Subscription: $($azAccount.name)" -ForegroundColor Green
    }
} catch {
    Write-Host "  ✗ Not logged in to Azure CLI" -ForegroundColor Red
}

# Check GCP
Write-Host "`nGCP:" -ForegroundColor Yellow
if ($env:GOOGLE_APPLICATION_CREDENTIALS) {
    Write-Host "  ✓ GOOGLE_APPLICATION_CREDENTIALS: $($env:GOOGLE_APPLICATION_CREDENTIALS)" -ForegroundColor Green
}
if ($env:GOOGLE_CLOUD_PROJECT -or $env:GCP_PROJECT) {
    $project = if ($env:GOOGLE_CLOUD_PROJECT) { $env:GOOGLE_CLOUD_PROJECT } else { $env:GCP_PROJECT }
    Write-Host "  ✓ Project: $project" -ForegroundColor Green
}
try {
    $gcloudAccount = gcloud config get-value account 2>$null
    if ($gcloudAccount -and $gcloudAccount -ne "(unset)") {
        Write-Host "  ✓ Gcloud account: $gcloudAccount" -ForegroundColor Green
    }
    $gcloudProject = gcloud config get-value project 2>$null
    if ($gcloudProject -and $gcloudProject -ne "(unset)") {
        Write-Host "  ✓ Gcloud project: $gcloudProject" -ForegroundColor Green
    }
} catch {
    Write-Host "  ✗ Gcloud not configured" -ForegroundColor Red
}

# Check DigitalOcean
Write-Host "`nDigitalOcean:" -ForegroundColor Yellow
$doTokenFound = $false
@("DIGITALOCEAN_TOKEN", "DIGITALOCEAN_ACCESS_TOKEN", "DO_TOKEN", "DO_ACCESS_TOKEN") | ForEach-Object {
    if ((Get-Item -Path "Env:$_" -ErrorAction SilentlyContinue).Value) {
        Write-Host "  ✓ $_ is set" -ForegroundColor Green
        $doTokenFound = $true
    }
}
if (-not $doTokenFound) {
    Write-Host "  ✗ No DigitalOcean token found" -ForegroundColor Red
}

# Check doctl
try {
    $doctlAuth = doctl auth list 2>$null
    if ($doctlAuth -match "default") {
        Write-Host "  ✓ doctl authenticated" -ForegroundColor Green
    }
} catch {
    # doctl might not be installed
}

Write-Host "`n=== Summary ===" -ForegroundColor Cyan
$configured = @()
if ((Test-Path $awsConfigPath) -or $env:AWS_ACCESS_KEY_ID) { $configured += "AWS" }
if ($azAccount) { $configured += "Azure" }
if ($env:GOOGLE_APPLICATION_CREDENTIALS -or $gcloudAccount) { $configured += "GCP" }
if ($doTokenFound) { $configured += "DigitalOcean" }

Write-Host "Configured providers: $($configured -join ', ')" -ForegroundColor Green
Write-Host "Total: $($configured.Count) providers" -ForegroundColor Green