@echo off
REM Build script for Mistigris Advent Calendar (Windows)
REM This script builds Windows 7 compatible 32-bit binaries
REM
REM Requirements:
REM   - Go 1.20.14 for Windows 7 compatibility
REM   - Install: go install golang.org/dl/go1.20.14@latest
REM   - Download: go1.20.14 download
REM
REM Usage: build.bat
REM Output: Creates binaries in dist\ directory

setlocal enabledelayedexpansion

echo Building Mistigris Advent Calendar for Windows 7...

REM Check for Go 1.20.14
set GO120=%USERPROFILE%\go\bin\go1.20.14.exe
if not exist "%GO120%" (
    echo ERROR: Go 1.20.14 not found at %GO120%
    echo Windows 7 compatibility requires Go 1.20.14
    echo.
    echo Install with:
    echo   go install golang.org/dl/go1.20.14@latest
    echo   go1.20.14 download
    echo.
    set GO120=go
    echo Falling back to default go compiler (may not work on Windows 7^)
    pause
)

REM Default values
set OUTPUT_DIR=dist

REM Create output directory
if not exist "%OUTPUT_DIR%" mkdir "%OUTPUT_DIR%"

echo Building for Windows 7 (386/32-bit) with Go 1.20.14...
set GOOS=windows
set GOARCH=386
set CGO_ENABLED=0
"%GO120%" build -ldflags="-s -w" -o "%OUTPUT_DIR%\advent-windows-386.exe" ./cmd/advent
if errorlevel 1 (
    echo Build failed for Windows 386
    exit /b 1
)

REM Create checksum (requires certutil on Windows)
certutil -hashfile "%OUTPUT_DIR%\advent-windows-386.exe" SHA256 > "%OUTPUT_DIR%\advent-windows-386.exe.sha256"

echo.
echo Build complete! Binary available in %OUTPUT_DIR%\
echo.
echo Binary Details:
dir "%OUTPUT_DIR%\advent-windows-386.exe"
echo.
echo Verify Go version:
"%GO120%" version "%OUTPUT_DIR%\advent-windows-386.exe"