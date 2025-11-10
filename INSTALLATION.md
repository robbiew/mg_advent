# BBS Door Installation Guide

## Overview
The Mistigris Advent Calendar is a BBS door program that displays interactive ANSI art advent calendars. It works with all modern BBS systems that support the Door32 dropfile specification.

## System Requirements
- Windows or Linux BBS system
- BBS software that creates `door32.sys` dropfiles
- ANSI/CP437 terminal emulation support
- Go 1.24+ (for building from source, or use pre-built binaries)

## Installation

### Windows BBS Systems

Example `advent.bat`:
   ```batch
   @echo off
   REM BBS Door Launcher - Replace paths with your BBS directories
   
   set NODE=%1
   if "%NODE%"=="" set NODE=1
   
   cd /d "C:\bbs\doors\advent"
   set DROPFILE_PATH=C:\bbs\temp\%NODE%\door32.sys
   
   advent.exe --path "%DROPFILE_PATH%"
   ```

### Linux BBS Systems
   
Example `advent.sh`:
   ```bash
   #!/bin/bash
   # BBS Door Launcher - Replace paths with your BBS directories
   
   NODE=${1:-1}
   cd /opt/bbs/doors/advent
   DROPFILE_PATH="/opt/bbs/temp/${NODE}/door32.sys"
   
   ./advent --path "$DROPFILE_PATH"
   ```

## Configuration

The door uses sensible defaults and doesn't require a configuration file. All settings can be controlled via command-line flags:

- **Timeout**: Hard-coded to 5 minutes idle, 2 hours maximum
- **Art directory**: Hard-coded to `art/` subdirectory  
- **Socket host**: Use `--socket-host` flag (default: 127.0.0.1)
- **Display mode**: Use `--local` flag for UTF-8 mode (BBS mode is default)

## Testing

### Local Testing (Without BBS)
```bash
# Windows
advent.exe --local --debug-disable-date --debug-date=2024-12-15

# Linux  
./advent --local --debug-disable-date --debug-date=2024-12-15
```

### BBS Integration Testing
- Test with actual user connections
- Verify art displays correctly
- Check timeout functionality
- Test navigation (arrow keys, Q to quit)

## Troubleshooting

### Common Issues

**Door doesn't start from BBS:**
- Verify `door32.sys` file exists in temp directory
- Check paths in launch script
- Ensure executable permissions (Linux)

**Art not displaying:**
- Verify art files exist in correct directories
- Check file naming: `1_DEC25.ANS`, `2_DEC25.ANS`, etc.
- Ensure ANSI emulation is enabled in BBS

**Connection issues:**
- Use `--socket-host` flag to specify your BBS server IP if not localhost
- Verify firewall allows connections
- Check BBS logs for socket errors

### Debug Mode
Add debug flags to your launch script for troubleshooting:
```
advent.exe --path "%DROPFILE_PATH%" --socket-host "192.168.1.100" --debug-disable-date --debug-date=2024-12-15
```

## Art File Requirements

The door expects specific art files:

### Common Files (Required)
- `art/common/WELCOME.ANS` - Welcome screen
- `art/common/GOODBYE.ANS` - Exit screen
- `art/common/COMEBACK.ANS` - "Come back tomorrow" screen
- `art/common/MISSING.ANS` - Missing art fallback
- `art/common/NOTYET.ANS` - Future date warning

### Daily Files
- `art/2025/1_DEC25.ANS` through `art/2025/25_DEC25.ANS`
- Similar for other years (2023, 2024)

Files must be in CP437 encoding with ANSI escape sequences.

## Support

For technical support or to report issues, visit the project repository or contact the development team. Include your BBS software type, operating system, and any error messages when reporting problems.