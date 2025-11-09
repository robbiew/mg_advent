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

1. **Download Files**
   - Download `advent-windows-amd64.exe` from releases
   - Rename to `advent.exe`
   - Create your door directory (e.g., `C:\bbs\doors\advent\`)

2. **Directory Structure**
   ```
   C:\bbs\doors\advent\
   ├── advent.exe
   ├── advent.bat           # BBS launcher script
   ├── config\
   │   └── config.yaml
   └── art\
       ├── common\          # Year-independent screens
       ├── 2023\           # 2023 advent calendar
       ├── 2024\           # 2024 advent calendar
       └── 2025\           # 2025 advent calendar
   ```

3. **Create Launch Script**
   - Copy `scripts/advent.bat` template to your door directory
   - Edit the paths to match your BBS setup
   
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

4. **Configure Your BBS**
   - Add door menu entry pointing to `advent.bat`
   - Ensure your BBS creates `door32.sys` files in the temp directory
   - Set ANSI emulation requirement

### Linux BBS Systems

1. **Download Files**
   - Download `advent-linux-amd64` from releases
   - Rename to `advent` and make executable: `chmod +x advent`
   - Create your door directory (e.g., `/opt/bbs/doors/advent/`)

2. **Directory Structure**
   ```
   /opt/bbs/doors/advent/
   ├── advent               # Executable
   ├── advent.sh           # BBS launcher script
   ├── config/
   │   └── config.yaml
   └── art/
       ├── common/         # Year-independent screens
       ├── 2023/          # 2023 advent calendar
       ├── 2024/          # 2024 advent calendar
       └── 2025/          # 2025 advent calendar
   ```

3. **Create Launch Script**
   - Copy `scripts/advent.sh` template to your door directory
   - Edit the paths to match your BBS setup
   - Make it executable: `chmod +x advent.sh`
   
   Example `advent.sh`:
   ```bash
   #!/bin/bash
   # BBS Door Launcher - Replace paths with your BBS directories
   
   NODE=${1:-1}
   cd /opt/bbs/doors/advent
   DROPFILE_PATH="/opt/bbs/temp/${NODE}/door32.sys"
   
   ./advent --path "$DROPFILE_PATH"
   ```

4. **Configure Your BBS**
   - Add door menu entry pointing to `advent.sh`
   - Ensure your BBS creates `door32.sys` files in the temp directory
   - Set ANSI emulation requirement

## Configuration

### config.yaml
```yaml
app:
  timeout_idle: "5m"      # User idle timeout
  timeout_max: "120m"     # Maximum session time

display:
  mode: "cp437"           # cp437 for BBS, utf8 for local testing
  
bbs:
  socket_host: "127.0.0.1"  # Change to your BBS server IP if needed
  emulation_required: 1     # Requires ANSI support

art:
  base_dir: "art"         # Relative to door executable
```

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
- Check `socket_host` in config.yaml matches your BBS server
- Verify firewall allows connections
- Check BBS logs for socket errors

### Debug Mode
Add debug flags to your launch script for troubleshooting:
```
advent.exe --path "%DROPFILE_PATH%" --debug-disable-date --debug-date=2024-12-15
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