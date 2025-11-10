@echo off
REM Package script for Mistigris Advent Calendar GitHub releases
REM Creates release archives for Windows 32-bit, Linux x86_64, and Linux ARM64
REM
REM Usage: package.bat
REM Output: Creates .zip and .tar.gz archives in dist\releases\

setlocal enabledelayedexpansion

echo Packaging Mistigris Advent Calendar for release...

REM Configuration
set OUTPUT_DIR=dist
set RELEASE_DIR=dist\releases
set ART_DIR=art

REM Create release directory
if not exist "%RELEASE_DIR%" mkdir "%RELEASE_DIR%"

REM Files to include in all packages
set COMMON_FILES=LICENSE QUICKSTART.md dist\FILE_ID.ANS dist\INFOFILE.ANS dist\MEMBERS.ANS

REM Check if binaries exist
echo Checking for binaries...
set missing_binaries=0

if not exist "%OUTPUT_DIR%\advent-windows-386.exe" (
    echo   X Missing: advent-windows-386.exe
    set missing_binaries=1
) else (
    echo   + Found: advent-windows-386.exe
)

if not exist "%OUTPUT_DIR%\advent-linux-amd64" (
    echo   X Missing: advent-linux-amd64
    set missing_binaries=1
) else (
    echo   + Found: advent-linux-amd64
)

if not exist "%OUTPUT_DIR%\advent-linux-arm64" (
    echo   X Missing: advent-linux-arm64
    set missing_binaries=1
) else (
    echo   + Found: advent-linux-arm64
)

if %missing_binaries%==1 (
    echo.
    echo ERROR: Some binaries are missing. Please run the build script first:
    echo   build.bat or from Linux: ./build.sh
    exit /b 1
)

echo.

REM Package Windows 32-bit
echo Creating package for windows-386...
set PACKAGE_NAME=advent-windows-386
set TEMP_DIR=%RELEASE_DIR%\%PACKAGE_NAME%

if exist "%TEMP_DIR%" rmdir /s /q "%TEMP_DIR%"
mkdir "%TEMP_DIR%"

REM Copy binary
copy "%OUTPUT_DIR%\advent-windows-386.exe" "%TEMP_DIR%\" > nul

REM Copy Windows launcher script
if exist "scripts\advent.bat" copy "scripts\advent.bat" "%TEMP_DIR%\" > nul

REM Copy common files
for %%f in (%COMMON_FILES%) do (
    if exist "%%f" (
        copy "%%f" "%TEMP_DIR%\" > nul
    ) else (
        echo WARNING: %%f not found, skipping...
    )
)

REM Copy art directory
if exist "%ART_DIR%" (
    xcopy /E /I /Q "%ART_DIR%" "%TEMP_DIR%\art" > nul
) else (
    echo ERROR: Art directory not found at %ART_DIR%
    exit /b 1
)

REM Create ZIP archive (requires PowerShell)
powershell -Command "Compress-Archive -Path '%TEMP_DIR%' -DestinationPath '%RELEASE_DIR%\%PACKAGE_NAME%.zip' -Force" > nul
echo   + Created %PACKAGE_NAME%.zip

REM Clean up
rmdir /s /q "%TEMP_DIR%"

REM Package Linux x86_64
echo Creating package for linux-amd64...
set PACKAGE_NAME=advent-linux-amd64
set TEMP_DIR=%RELEASE_DIR%\%PACKAGE_NAME%

if exist "%TEMP_DIR%" rmdir /s /q "%TEMP_DIR%"
mkdir "%TEMP_DIR%"

copy "%OUTPUT_DIR%\advent-linux-amd64" "%TEMP_DIR%\" > nul

REM Copy Linux launcher script
if exist "scripts\advent.sh" copy "scripts\advent.sh" "%TEMP_DIR%\" > nul

for %%f in (%COMMON_FILES%) do (
    if exist "%%f" copy "%%f" "%TEMP_DIR%\" > nul
)

xcopy /E /I /Q "%ART_DIR%" "%TEMP_DIR%\art" > nul

powershell -Command "Compress-Archive -Path '%TEMP_DIR%' -DestinationPath '%RELEASE_DIR%\%PACKAGE_NAME%.zip' -Force" > nul
echo   + Created %PACKAGE_NAME%.zip

rmdir /s /q "%TEMP_DIR%"

REM Package Linux ARM64
echo Creating package for linux-arm64...
set PACKAGE_NAME=advent-linux-arm64
set TEMP_DIR=%RELEASE_DIR%\%PACKAGE_NAME%

if exist "%TEMP_DIR%" rmdir /s /q "%TEMP_DIR%"
mkdir "%TEMP_DIR%"

copy "%OUTPUT_DIR%\advent-linux-arm64" "%TEMP_DIR%\" > nul

REM Copy Linux launcher script
if exist "scripts\advent.sh" copy "scripts\advent.sh" "%TEMP_DIR%\" > nul

for %%f in (%COMMON_FILES%) do (
    if exist "%%f" copy "%%f" "%TEMP_DIR%\" > nul
)

xcopy /E /I /Q "%ART_DIR%" "%TEMP_DIR%\art" > nul

powershell -Command "Compress-Archive -Path '%TEMP_DIR%' -DestinationPath '%RELEASE_DIR%\%PACKAGE_NAME%.zip' -Force" > nul
echo   + Created %PACKAGE_NAME%.zip

rmdir /s /q "%TEMP_DIR%"

echo.
echo Packaging complete! Release archives available in %RELEASE_DIR%\
dir "%RELEASE_DIR%\*.zip"

echo.
echo To create a GitHub release:
echo   1. Create a new tag: git tag -a v1.0.0 -m "Release v1.0.0"
echo   2. Push the tag: git push origin v1.0.0
echo   3. Upload files from %RELEASE_DIR%\ to the GitHub release page
