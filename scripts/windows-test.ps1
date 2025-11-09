# Setup script for local testing of Mistigris Advent Calendar
# Copies all necessary files to C:\talisman\doors\advent for testing
# This script is for local development testing only

param(
    [string]$TestDir = "C:\talisman\doors\advent"
)

$ErrorActionPreference = "Stop"

Write-Host "Setting up local test environment in: $TestDir" -ForegroundColor Green

# Clean up any existing test directory
if (Test-Path $TestDir) {
    Write-Host "Removing existing test directory..." -ForegroundColor Yellow
    Remove-Item -Path $TestDir -Recurse -Force
}

# Create test directory
Write-Host "Creating test directory..." -ForegroundColor Cyan
New-Item -ItemType Directory -Path $TestDir -Force | Out-Null

# Copy the built binary
if (Test-Path "advent.exe") {
    Write-Host "Copying application binary..." -ForegroundColor Cyan
    Copy-Item "advent.exe" -Destination $TestDir
} elseif (Test-Path "dist\advent-windows-amd64.exe") {
    Write-Host "Using pre-built Windows binary..." -ForegroundColor Cyan
    Copy-Item "dist\advent-windows-amd64.exe" -Destination "$TestDir\advent.exe"
} else {
    Write-Host "Building application first..." -ForegroundColor Yellow
    # Try building from cmd/advent directory
    & go build -o advent.exe ./cmd/advent
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to build application from ./cmd/advent"
        exit 1
    }
    Copy-Item "advent.exe" -Destination $TestDir
}

# Copy configuration
if (Test-Path "config") {
    Write-Host "Copying configuration..." -ForegroundColor Cyan
    Copy-Item -Path "config" -Destination $TestDir -Recurse
}

# Copy art directory
if (Test-Path "art") {
    Write-Host "Copying art assets..." -ForegroundColor Cyan
    Copy-Item -Path "art" -Destination $TestDir -Recurse
}

# Copy documentation
Write-Host "Copying documentation..." -ForegroundColor Cyan
$docFiles = @("README.md", "LICENSE")
foreach ($file in $docFiles) {
    if (Test-Path $file) {
        Copy-Item $file -Destination $TestDir
    }
}

# Copy BBS info files
Write-Host "Copying BBS info files..." -ForegroundColor Cyan
$bbsFiles = @("FILE_ID.ANS", "INFOFILE.ANS", "MEMBERS.ANS")
foreach ($file in $bbsFiles) {
    if (Test-Path $file) {
        Copy-Item $file -Destination $TestDir
    }
}

# Create a sample config for local testing
Write-Host "Creating sample config.yaml for local testing..." -ForegroundColor Cyan
$configContent = @"
app:
  name: "Mistigris Advent Calendar - Local Test"
  version: "2.0.0"
  timeout_idle: "5m"
  timeout_max: "120m"

display:
  mode: "utf8"
  theme: "classic"
  scrolling:
    enabled: true
    indicators: true
    keyboard_shortcuts: true
  columns:
    handle_80_column_issue: true
    auto_detect_width: true
  performance:
    cache_enabled: true
    cache_size_mb: 50
    preload_lines: 100

logging:
  level: "info"
  format: "text"

art:
  base_dir: "art"

bbs:
  dropfile_path: "door32.sys"
"@

Set-Content -Path "$TestDir\config.yaml" -Value $configContent

Write-Host ""
Write-Host "✅ Local test environment setup complete!" -ForegroundColor Green
Write-Host ""
Write-Host "Test Directory: $TestDir" -ForegroundColor White
Write-Host "Contents:" -ForegroundColor White
Get-ChildItem $TestDir | Format-Table Name, Length, LastWriteTime
Write-Host ""
Write-Host "To test locally:" -ForegroundColor Yellow
Write-Host "  Set-Location '$TestDir'" -ForegroundColor Gray
Write-Host "  .\advent.exe --local" -ForegroundColor Gray
Write-Host ""
Write-Host "To test with BBS simulation:" -ForegroundColor Yellow
Write-Host "  Set-Location '$TestDir'" -ForegroundColor Gray
Write-Host "  .\advent.exe --path door32.sys" -ForegroundColor Gray
Write-Host ""
Write-Host "Note: This directory is gitignored and only for your local testing." -ForegroundColor DarkGray

Write-Host ""
Write-Host "Press any key to continue..."
$null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")