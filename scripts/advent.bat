@echo off
REM Mistigris Advent Calendar launcher for Windows
REM This script launches the advent calendar with the correct path to the door32.sys file

REM Use the provided dropfile path
advent-windows-386.exe -path "%~1"

REM Exit with the same error code as the advent program
exit /b %ERRORLEVEL%