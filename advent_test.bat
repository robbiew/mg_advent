@echo off
REM Mistigris Advent Calendar BBS Door - TEST VERSION
REM This batch file is for TESTING ONLY and includes fallback door32.sys creation
REM For production BBS use, use advent.bat instead
REM Usage: advent_test.bat [node] [socket_handle]

setlocal enabledelayedexpansion

REM Get parameters from Talisman BBS
set NODE=%1
set SOCKET_HANDLE=%2

REM Default to node 1 if not provided
if "%NODE%"=="" set NODE=1

REM Set working directory to the door's directory
cd /d "%~dp0"

REM Log the door start for debugging
echo [%DATE% %TIME%] Starting Advent Door (TEST MODE) - Node: %NODE%, Socket: %SOCKET_HANDLE% >> advent_door.log

REM Talisman generates door32.sys in C:\talisman\temp\[node number]
set DROPFILE_PATH=C:\talisman\temp\%NODE%\door32.sys

REM Check if the Talisman-generated door32.sys exists
if exist "%DROPFILE_PATH%" (
    echo [%DATE% %TIME%] Using Talisman door32.sys: %DROPFILE_PATH% >> advent_door.log
    REM Launch the advent door with the Talisman dropfile path
    advent.exe --path "%DROPFILE_PATH%"
) else (
    REM Fallback: Create our own door32.sys for testing ONLY
    echo [%DATE% %TIME%] TEST MODE: Talisman door32.sys not found, creating test dropfile >> advent_door.log
    call :create_door32_sys
    
    if "%SOCKET_HANDLE%"=="" (
        REM No socket handle - run in local mode for testing
        echo [%DATE% %TIME%] TEST MODE: Running in local mode >> advent_door.log
        advent.exe --local --path door32.sys
    ) else (
        REM Socket mode - use the test dropfile
        echo [%DATE% %TIME%] TEST MODE: Running in BBS mode with socket %SOCKET_HANDLE% >> advent_door.log
        advent.exe --path door32.sys
    )
    
    REM Clean up our test dropfile
    if exist door32.sys del door32.sys
)

REM Log completion
echo [%DATE% %TIME%] Advent Door (TEST MODE) completed >> advent_door.log

goto :eof

REM Create door32.sys dropfile with proper format (for testing only)
:create_door32_sys
echo [%DATE% %TIME%] TEST MODE: Creating test door32.sys >> advent_door.log
REM Standard Door32.sys format for socket-based doors
(
echo 2
echo Talisman
echo User
echo Test User
echo 25
echo 1
echo %NODE%
echo %SOCKET_HANDLE%
echo 0
echo 30
echo 09:00
echo 10080
echo 1
) > door32.sys

REM Add socket connection info that our BBS connector expects
REM Note: This assumes Talisman provides socket on localhost
REM You may need to adjust this based on your Talisman setup
if not "%SOCKET_HANDLE%"=="" (
    echo SocketHost=127.0.0.1 >> door32.sys
    echo SocketPort=%SOCKET_HANDLE% >> door32.sys
)

goto :eof