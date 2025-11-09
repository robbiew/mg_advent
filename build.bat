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

echo Building for Windows (amd64)...
set GOOS=windows
set GOARCH=amd64
go build -ldflags="-s -w" -o "%OUTPUT_DIR%\advent-windows-amd64.exe" ./cmd/advent
if errorlevel 1 (
    echo Build failed for Windows amd64
    exit /b 1
)

REM Create checksum (requires certutil on Windows)
certutil -hashfile "%OUTPUT_DIR%\advent-windows-amd64.exe" SHA256 > "%OUTPUT_DIR%\advent-windows-amd64.exe.sha256"

echo Building for Linux (amd64)...
set GOOS=linux
set GOARCH=amd64
go build -ldflags="-s -w" -o "%OUTPUT_DIR%\advent-linux-amd64" ./cmd/advent
if errorlevel 1 (
    echo Build failed for Linux amd64
    exit /b 1
)

REM Create checksum for Linux binary
certutil -hashfile "%OUTPUT_DIR%\advent-linux-amd64" SHA256 > "%OUTPUT_DIR%\advent-linux-amd64.sha256"

echo.
echo Build complete! Binaries available in %OUTPUT_DIR%\
dir "%OUTPUT_DIR%"