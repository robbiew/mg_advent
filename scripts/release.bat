@echo off
REM GitHub Release Script for Mistigris Advent Calendar
REM Creates a GitHub release and uploads all packaged archives
REM
REM Prerequisites:
REM   - GitHub CLI (gh) installed and authenticated
REM   - Archives built using package.bat
REM
REM Usage: release.bat <version> [release-notes]
REM Example: release.bat v1.0.0 "Initial release with 2023-2025 calendars"

setlocal enabledelayedexpansion

REM Configuration
set RELEASE_DIR=dist\releases
set REPO=robbiew/mg_advent

REM Check arguments
if "%~1"=="" (
    echo Usage: %~nx0 ^<version^> [release-notes]
    echo.
    echo Examples:
    echo   %~nx0 v1.0.0
    echo   %~nx0 v1.0.0 "Initial release with 2023-2025 calendars"
    echo   %~nx0 v1.1.0 "Added new features and bug fixes"
    echo.
    echo Note: Version should start with 'v' ^(e.g., v1.0.0^)
    exit /b 1
)

set VERSION=%~1
set NOTES=%~2
if "%NOTES%"=="" set NOTES=Release %VERSION%

REM Check if gh is installed
where gh >nul 2>nul
if errorlevel 1 (
    echo ERROR: GitHub CLI ^(gh^) is not installed.
    echo.
    echo Install from: https://cli.github.com/manual/installation
    echo After installing, authenticate with: gh auth login
    exit /b 1
)

REM Check if authenticated
gh auth status >nul 2>nul
if errorlevel 1 (
    echo ERROR: GitHub CLI is not authenticated.
    echo Please run: gh auth login
    exit /b 1
)

REM Check if release directory exists
if not exist "%RELEASE_DIR%" (
    echo ERROR: Release directory not found: %RELEASE_DIR%
    echo Please run package.bat first to create release archives.
    exit /b 1
)

REM Find all archives
set ARCHIVE_COUNT=0
for %%f in ("%RELEASE_DIR%\*.zip") do (
    set /a ARCHIVE_COUNT+=1
)

if %ARCHIVE_COUNT%==0 (
    echo ERROR: No archives found in %RELEASE_DIR%
    echo Please run package.bat first to create release archives.
    exit /b 1
)

echo ================================================
echo GitHub Release Creator
echo ================================================
echo Repository: %REPO%
echo Version:    %VERSION%
echo Notes:      %NOTES%
echo.
echo Archives to upload:
for %%f in ("%RELEASE_DIR%\*.zip") do (
    echo   - %%~nxf
)
echo ================================================
echo.

REM Confirm release
set /p CONFIRM="Create this release? (y/N) "
if /i not "%CONFIRM%"=="y" (
    echo Release cancelled.
    exit /b 0
)

REM Check if tag exists
git rev-parse %VERSION% >nul 2>nul
if not errorlevel 1 (
    echo Tag %VERSION% already exists.
    set /p RETAG="Delete and recreate tag? (y/N) "
    if /i "!RETAG!"=="y" (
        echo Deleting local tag...
        git tag -d %VERSION%
        echo Deleting remote tag...
        git push origin :refs/tags/%VERSION% 2>nul
    ) else (
        echo Using existing tag.
    )
) else (
    REM Create and push tag
    echo Creating tag %VERSION%...
    git tag -a %VERSION% -m "%NOTES%"
    echo Pushing tag to GitHub...
    git push origin %VERSION%
)

REM Check if release exists
gh release view %VERSION% --repo %REPO% >nul 2>nul
if not errorlevel 1 (
    echo.
    echo Release %VERSION% already exists.
    set /p RECREATE="Delete and recreate release? (y/N) "
    if /i "!RECREATE!"=="y" (
        echo Deleting existing release...
        gh release delete %VERSION% --repo %REPO% --yes
    ) else (
        echo Cancelled. Use a different version or delete the existing release first.
        exit /b 1
    )
)

REM Build file list for upload
set FILES=
for %%f in ("%RELEASE_DIR%\*.zip") do (
    set FILES=!FILES! "%%f"
)

REM Create release
echo.
echo Creating GitHub release...
gh release create %VERSION% --repo %REPO% --title "%VERSION%" --notes "%NOTES%" %FILES%

if errorlevel 1 (
    echo.
    echo ERROR: Failed to create release.
    exit /b 1
)

echo.
echo ================================================
echo + Release %VERSION% created successfully!
echo ================================================
echo.
echo View release at:
echo   https://github.com/%REPO%/releases/tag/%VERSION%
echo.
echo Uploaded archives:
for %%f in ("%RELEASE_DIR%\*.zip") do (
    echo   + %%~nxf
)
