# MiSTiGRiS Advent Calendar

<img src="images/WELCOME.png" alt="Welcome Screen" width="100%">

An interactive BBS door program that displays ANSI Christmas art, a new one each day in December. Browse past collections as well. Compatible with all modern BBS systems that support Door32 dropfiles -- Windows and Linux.

## Quick Setup

### Windows

```batch
@echo off
set NODE=%1
if "%NODE%"=="" set NODE=1

cd /d "C:\bbs\doors\advent"
set DROPFILE_PATH=C:\bbs\temp\%NODE%\door32.sys

advent.exe --path "%DROPFILE_PATH%"
```

### Linux

```bash
#!/bin/bash
NODE=${1:-1}
cd /opt/bbs/doors/advent
DROPFILE_PATH="/opt/bbs/temp/${NODE}/door32.sys"

./advent --path "$DROPFILE_PATH"
```

Make executable: `chmod +x advent.sh advent-linux-amd64`

## Command Line Options

```
--path string           Path to door32.sys file
--local                 Run in local UTF-8 mode (not BBS mode)
--socket-host string    BBS server IP address (default "127.0.0.1")
--debug                 Enable debug logging
--debug-date string     Override date (YYYY-MM-DD)
--debug-disable-date    Disable date validation
--debug-disable-art     Disable art validation
```

## Building from Source

Prerequisites: Go 1.24 or later, Git

```bash
# Clone repository
git clone https://github.com/robbiew/mg_advent.git
cd mg_advent

# Linux/Mac build
./build.sh

# Windows build  
build.bat
```

## Testing

```bash
# Local testing (no BBS required)
# Skip date restrictions to view any day's art
./advent --local --debug-disable-date --debug-date=2024-12-15
```

## Usage

- **Arrow Keys**: Navigate between days
- **1, 2, 3**: Jump to different years (2023, 2024, 2025)
- **Q or ESC**: Return to welcome screen / exit
- **I**: View info file
- **M**: View members list

## Features

- Daily ANSI art for each day in December
- Multi-year support (2023, 2024, 2025)
- BBS compatibility with Door32.sys dropfiles
- Art bundled with binary (no separate art directory needed)

## License

This project is released under the terms specified in the LICENSE file.
