@echo off
REM Test script to simulate Talisman BBS calling the Advent door
REM This simulates how Talisman would call: advent.bat [node] [socket_handle]

setlocal enabledelayedexpansion

echo Mistigris Advent Calendar - BBS Door Test
echo =========================================
echo.

REM Set test directory
set TEST_DIR=C:\talisman\doors\advent
echo Test Directory: %TEST_DIR%
echo.

REM Check if test directory exists
if not exist "%TEST_DIR%" (
    echo ERROR: Test directory does not exist!
    echo Please run the setup script first to create: %TEST_DIR%
    echo.
    pause
    exit /b 1
)

REM Check if advent.exe exists
if not exist "%TEST_DIR%\advent.exe" (
    echo ERROR: advent.exe not found in %TEST_DIR%
    echo Please build the application first.
    echo.
    pause
    exit /b 1
)

REM Check if advent.bat and advent_test.bat exist
if not exist "%TEST_DIR%\advent.bat" (
    echo Creating advent.bat in test directory...
    copy advent.bat "%TEST_DIR%\"
)
if not exist "%TEST_DIR%\advent_test.bat" (
    echo Creating advent_test.bat in test directory...
    copy advent_test.bat "%TEST_DIR%\"
)

REM Change to test directory
cd /d "%TEST_DIR%"

echo Choose test mode:
echo.
echo 1. Local test (no BBS simulation)
echo 2. BBS simulation with socket (simulates Talisman call)
echo 3. Direct door32.sys test
echo 4. Full Talisman simulation (creates temp directory structure)
echo.
set /p CHOICE="Enter choice (1-4): "

if "%CHOICE%"=="1" goto local_test
if "%CHOICE%"=="2" goto bbs_simulation
if "%CHOICE%"=="3" goto door32_test
if "%CHOICE%"=="4" goto talisman_simulation
goto invalid_choice

:local_test
echo.
echo Running local test...
echo Command: advent.exe --local
echo.
advent.exe --local
goto end

:bbs_simulation
echo.
echo Simulating BBS call...
set NODE_NUM=1
set SOCKET_HANDLE=2024
echo.
echo Simulating: advent_test.bat %NODE_NUM% %SOCKET_HANDLE%
echo (This simulates how Talisman BBS would call the door, but uses test version)
echo.
echo Note: This will create door32.sys and attempt socket connection
echo The socket connection will likely fail since we're not running a BBS
echo.
pause
call advent_test.bat %NODE_NUM% %SOCKET_HANDLE%
goto end

:door32_test
echo.
echo Creating test Talisman directory structure...
set NODE_NUM=1
set TEST_TEMP_DIR=C:\talisman\temp\%NODE_NUM%

if not exist "%TEST_TEMP_DIR%" (
    echo Creating: %TEST_TEMP_DIR%
    mkdir "%TEST_TEMP_DIR%"
)

echo Creating test door32.sys file in Talisman temp directory...
(
echo 2
echo Talisman Test BBS
echo Test
echo User  
echo TestUser
echo 255
echo 60
echo 1
echo %NODE_NUM%
echo 2024
echo SocketHost=127.0.0.1
echo SocketPort=2024
) > "%TEST_TEMP_DIR%\door32.sys"

echo.
echo door32.sys created in: %TEST_TEMP_DIR%\door32.sys
type "%TEST_TEMP_DIR%\door32.sys"
echo.
echo Running: advent.exe --path "%TEST_TEMP_DIR%\door32.sys"
echo Note: Socket connection will likely fail since we're not running a BBS
echo.
pause
advent.exe --path "%TEST_TEMP_DIR%\door32.sys"

echo.
echo Cleaning up test Talisman directory...
if exist "%TEST_TEMP_DIR%\door32.sys" del "%TEST_TEMP_DIR%\door32.sys"
if exist "%TEST_TEMP_DIR%" rmdir "%TEST_TEMP_DIR%"
if exist "C:\talisman\temp" rmdir "C:\talisman\temp" 2>nul
goto end

:talisman_simulation
echo.
echo Full Talisman BBS simulation...
set NODE_NUM=1
set SOCKET_HANDLE=2024
set TEST_TEMP_DIR=C:\talisman\temp\%NODE_NUM%

echo Creating Talisman temp directory structure: %TEST_TEMP_DIR%
if not exist "%TEST_TEMP_DIR%" (
    mkdir "%TEST_TEMP_DIR%"
)

echo Creating door32.sys as Talisman would...
(
echo 2
echo Talisman BBS
echo John
echo Doe
echo TestUser
echo 255
echo 60
echo 1
echo %NODE_NUM%
echo %SOCKET_HANDLE%
) > "%TEST_TEMP_DIR%\door32.sys"

echo.
echo Talisman door32.sys created in: %TEST_TEMP_DIR%\door32.sys
echo Contents:
type "%TEST_TEMP_DIR%\door32.sys"
echo.
echo Now calling advent.bat as Talisman would: advent.bat %NODE_NUM% %SOCKET_HANDLE%
echo This should find the door32.sys in the Talisman temp directory.
echo.
pause
call advent.bat %NODE_NUM% %SOCKET_HANDLE%

echo.
echo Cleaning up test Talisman directory...
if exist "%TEST_TEMP_DIR%\door32.sys" del "%TEST_TEMP_DIR%\door32.sys"
if exist "%TEST_TEMP_DIR%" rmdir "%TEST_TEMP_DIR%"
if exist "C:\talisman\temp" rmdir "C:\talisman\temp" 2>nul
goto end

:invalid_choice
echo Invalid choice. Please run the script again.
goto end

:end
echo.
echo Test completed.
if exist door32.sys (
    echo.
    echo door32.sys was created during test:
    type door32.sys
    echo.
    echo Cleaning up door32.sys...
    del door32.sys
)

if exist advent_door.log (
    echo.
    echo Door log entries:
    type advent_door.log
)

echo.
pause