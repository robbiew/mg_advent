@echo off
REM BBS Door Launcher Script Template for Windows
REM
REM TEMPLATE FILE: Copy this to your BBS door directory and customize paths
REM Called by BBS: advent.bat [node] [socket_handle]
REM Replace paths below with your actual BBS directories

setlocal enabledelayedexpansion

REM Get node number from BBS (default to 1)
set NODE=%1
if "%NODE%"=="" set NODE=1

REM Set working directory to door location  
cd /d "%~dp0"

REM BBS door32.sys location (adjust path for your BBS)
set DROPFILE_PATH=c:\talisman\temp\%NODE%\door32.sys

REM BBS server IP (change if BBS runs on different server)
set BBS_HOST=127.0.0.1

REM Simple logging
echo [%DATE% %TIME%] Node %NODE% - Starting >> advent_door.log

REM Check for door32.sys and launch
if exist "%DROPFILE_PATH%" (
    echo [%DATE% %TIME%] Node %NODE% - Launching door with dropfile: %DROPFILE_PATH% >> advent_door.log
    advent.exe --path "%DROPFILE_PATH%" --socket-host "%BBS_HOST%" 2>&1 >> advent_door.log
    set DOOR_EXIT_CODE=%ERRORLEVEL%
    echo [%DATE% %TIME%] Node %NODE% - Door exited with code: %DOOR_EXIT_CODE% >> advent_door.log
    if !DOOR_EXIT_CODE! NEQ 0 (
        echo [%DATE% %TIME%] Node %NODE% - ERROR: Door failed with exit code !DOOR_EXIT_CODE! >> advent_door.log
    )
) else (
    echo [%DATE% %TIME%] Node %NODE% - ERROR: door32.sys not found at %DROPFILE_PATH% >> advent_door.log
    echo ERROR: door32.sys not found
    pause
    exit /b 1
)