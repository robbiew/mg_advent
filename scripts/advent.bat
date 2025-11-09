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

REM BBS door32.sys location
set DROPFILE_PATH=c:\talisman\temp\%NODE%\door32.sys

REM Simple logging
echo [%DATE% %TIME%] Node %NODE% - Starting >> advent_door.log

REM Check for door32.sys and launch
if exist "%DROPFILE_PATH%" (
    advent.exe --path "%DROPFILE_PATH%"
    echo [%DATE% %TIME%] Node %NODE% - Completed >> advent_door.log
) else (
    echo [%DATE% %TIME%] Node %NODE% - ERROR: door32.sys not found at %DROPFILE_PATH% >> advent_door.log
    echo ERROR: door32.sys not found
    pause
    exit /b 1
)