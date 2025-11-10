# Mistigris Advent Calendar - Quick Start Guide

## What's Included

- Advent calendar BBS door executable
- ANSI art files for display
- FILE_ID.ANS, INFOFILE.ANS, MEMBERS.ANS - BBS file descriptions

## Installation

### 1. Extract Files
Extract all files to your BBS door directory (e.g., `C:\bbs\doors\advent` or `/opt/bbs/doors/advent`)

### 2. Create Launch Script

**Windows (advent.bat):**
```batch
@echo off
set NODE=%1
if "%NODE%"=="" set NODE=1

cd /d "C:\bbs\doors\advent"
set DROPFILE_PATH=C:\bbs\temp\%NODE%\door32.sys

advent-windows-386.exe --path "%DROPFILE_PATH%"
```

**Linux (advent.sh):**
```bash
#!/bin/bash
NODE=${1:-1}
cd /opt/bbs/doors/advent
DROPFILE_PATH="/opt/bbs/temp/${NODE}/door32.sys"

./advent-linux-amd64 --path "$DROPFILE_PATH"
```

Make executable: `chmod +x advent.sh advent-linux-amd64`

### 3. Configure in BBS

Add the door to your BBS menu system:
- **Command**: Path to your launch script
- **Required**: Door32.sys dropfile support
- **Terminal**: ANSI/CP437 emulation

## Testing

Test locally without BBS:
```bash
# Windows
advent-windows-386.exe --local --debug-disable-date --debug-date=2024-12-15

# Linux
./advent-linux-amd64 --local --debug-disable-date --debug-date=2024-12-15
```

## Usage

- **Arrow Keys**: Navigate between days
- **1, 2, 3**: Jump to different years (2023, 2024, 2025)
- **Q or ESC**: Quit door

## Requirements

- BBS system with Door32.sys support
- ANSI/CP437 terminal emulation
- The door only displays art for days that have passed (up to Dec 25)

## Troubleshooting

**Art not displaying?**
- Verify art files are in `art/common/` and `art/2023/`, `art/2024/`, `art/2025/` directories
- Check ANSI emulation is enabled

**Door won't start?**
- Verify door32.sys path is correct
- Check file permissions (Linux)
- Use `--socket-host` flag if BBS is not on localhost

## Support

Full documentation: https://github.com/robbiew/mg_advent
