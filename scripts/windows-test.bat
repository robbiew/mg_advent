@echo off
REM Setup script for local testing of Mistigris Advent Calendar
REM Copies all necessary files to C:\talisman\doors\advent for testing
REM This script is for local development testing only

setlocal enabledelayedexpansion

set TEST_DIR=C:\talisman\doors\advent
echo Setting up local test environment in: %TEST_DIR%

REM Clean up any existing test directory
if exist "%TEST_DIR%" (
    echo Removing existing test directory...
    rmdir /s /q "%TEST_DIR%"
)

REM Create test directory
echo Creating test directory...
mkdir "%TEST_DIR%"

REM Copy the built binary
if exist "advent.exe" (
    echo Copying application binary...
    copy advent.exe "%TEST_DIR%\"
) else if exist "dist\advent-windows-amd64.exe" (
    echo Using pre-built Windows binary...
    copy "dist\advent-windows-amd64.exe" "%TEST_DIR%\advent.exe"
) else (
    echo Building application first...
    go build -o advent.exe ./cmd/advent
    if errorlevel 1 (
        echo Build failed from ./cmd/advent directory.
        pause
        exit /b 1
    )
    copy advent.exe "%TEST_DIR%\"
)

REM Copy configuration
echo Copying configuration...
xcopy /s /e /i config "%TEST_DIR%\config"

REM Copy art directory
echo Copying art assets...
xcopy /s /e /i art "%TEST_DIR%\art"

REM Copy documentation
echo Copying documentation...
copy README.md "%TEST_DIR%\"
copy LICENSE "%TEST_DIR%\"

REM Copy BBS info files
echo Copying BBS info files...
copy FILE_ID.ANS "%TEST_DIR%\"
copy INFOFILE.ANS "%TEST_DIR%\"
copy MEMBERS.ANS "%TEST_DIR%\"

REM Create a sample config for local testing
echo Creating sample config.yaml for local testing...
(
echo app:
echo   name: "Mistigris Advent Calendar - Local Test"
echo   version: "2.0.0"
echo   timeout_idle: "5m"
echo   timeout_max: "120m"
echo.
echo display:
echo   mode: "utf8"
echo   theme: "classic"
echo   scrolling:
echo     enabled: true
echo     indicators: true
echo     keyboard_shortcuts: true
echo   columns:
echo     handle_80_column_issue: true
echo     auto_detect_width: true
echo   performance:
echo     cache_enabled: true
echo     cache_size_mb: 50
echo     preload_lines: 100
echo.
echo logging:
echo   level: "info"
echo   format: "text"
echo.
echo art:
echo   base_dir: "art"
echo.
echo bbs:
echo   dropfile_path: "door32.sys"
) > "%TEST_DIR%\config.yaml"

echo.
echo ✅ Local test environment setup complete!
echo.
echo Test Directory: %TEST_DIR%
echo Contents:
dir "%TEST_DIR%"
echo.
echo To test locally:
echo   cd /d "%TEST_DIR%"
echo   advent.exe --local
echo.
echo To test with BBS simulation:
echo   cd /d "%TEST_DIR%"
echo   advent.exe --path door32.sys
echo.
echo Note: This directory is gitignored and only for your local testing.

pause