@echo off
setlocal ENABLEDELAYEDEXPANSION

REM Simple build script for mg_advent DOS (go32v2)

set ROOT=%~dp0..
set SRC=%ROOT%\src
set OUT=%ROOT%\bin

if not exist "%OUT%" mkdir "%OUT%"

echo Building mg_advent (DOS / go32v2) ...
fpc -Tgo32v2 -O2 -Sm -Sd ^
  -Fu"%SRC%" ^
  -Fi"%SRC%" ^
  -FE"%OUT%" ^
  "%SRC%\advent.pas"

echo Done. Output: %OUT%\ADVENT.EXE
endlocal
