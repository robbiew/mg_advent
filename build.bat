@echo off
REM Build script for Mistigris Advent Calendar (Windows)
REM This script builds the application for multiple platforms
REM
REM Usage: build.bat
REM Output: Creates binaries in dist\ directory

setlocal enabledelayedexpansion

echo Building Mistigris Advent Calendar...

REM Default values
set OUTPUT_DIR=dist

REM Create output directory
if not exist "%OUTPUT_DIR%" mkdir "%OUTPUT_DIR%"

echo Building for Windows (386/32-bit)...
set GOOS=windows
set GOARCH=386
go build -ldflags="-s -w" -o "%OUTPUT_DIR%\advent-windows-386.exe" ./cmd/advent
if errorlevel 1 (
    echo Build failed for Windows 386
    exit /b 1
)

REM Create checksum (requires certutil on Windows)
certutil -hashfile "%OUTPUT_DIR%\advent-windows-386.exe" SHA256 > "%OUTPUT_DIR%\advent-windows-386.exe.sha256"

echo.
echo Build complete! Binaries available in %OUTPUT_DIR%\
dir "%OUTPUT_DIR%"