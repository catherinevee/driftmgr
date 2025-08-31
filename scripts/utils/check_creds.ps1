Write-Host "=== Checking Cloud Provider Credentials ===" -ForegroundColor Cyan

# AWS
Write-Host "`nAWS:" -ForegroundColor Yellow
$awsConfigured = $false
if (Test-Path "$env:USERPROFILE\.aws\config") {
    Write-Host "  Config file found" -ForegroundColor Green
    $awsConfigured = $true
}
if ($env:AWS_ACCESS_KEY_ID) {
    Write-Host "  AWS_ACCESS_KEY_ID set" -ForegroundColor Green
    $awsConfigured = $true
}
if ($env:AWS_PROFILE) {
    Write-Host "  Profile: $env:AWS_PROFILE" -ForegroundColor Green
}

# Azure
Write-Host "`nAzure:" -ForegroundColor Yellow
$azureConfigured = $false
$azAccount = az account show --output json 2>$null | ConvertFrom-Json
if ($azAccount) {
    Write-Host "  Logged in: $($azAccount.name)" -ForegroundColor Green
    $azureConfigured = $true
} else {
    Write-Host "  Not logged in" -ForegroundColor Red
}

# GCP
Write-Host "`nGCP:" -ForegroundColor Yellow
$gcpConfigured = $false
if ($env:GOOGLE_APPLICATION_CREDENTIALS) {
    Write-Host "  Credentials file set" -ForegroundColor Green
    $gcpConfigured = $true
}
$gcloudAccount = gcloud config get-value account 2>$null
if ($gcloudAccount -and $gcloudAccount -ne "(unset)") {
    Write-Host "  Account: $gcloudAccount" -ForegroundColor Green
    $gcpConfigured = $true
}

# DigitalOcean
Write-Host "`nDigitalOcean:" -ForegroundColor Yellow
$doConfigured = $false
if ($env:DIGITALOCEAN_TOKEN -or $env:DO_TOKEN) {
    Write-Host "  Token set" -ForegroundColor Green
    $doConfigured = $true
} else {
    Write-Host "  No token" -ForegroundColor Red
}

Write-Host "`n=== Summary ===" -ForegroundColor Cyan
$count = 0
if ($awsConfigured) { $count++ }
if ($azureConfigured) { $count++ }
if ($gcpConfigured) { $count++ }
if ($doConfigured) { $count++ }
Write-Host "Configured: $count/4 providers" -ForegroundColor Yellow