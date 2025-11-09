# Talisman BBS Integration Guide

## Overview
This guide explains how to integrate the Mistigris Advent Calendar as a BBS door with Talisman BBS.

## Files Required

### 1. advent.exe
The main application binary built from the Go source code.

### 2. advent.bat
The production batch file that Talisman BBS calls to launch the door. This file:
- Receives node number and socket handle parameters from Talisman
- Locates the door32.sys file in the correct Talisman temp directory (C:\talisman\temp\[node]\door32.sys)
- Launches the advent.exe with appropriate parameters
- Provides logging for debugging
- **NEVER creates its own door32.sys** - always expects the BBS to provide it
- Exits with error if door32.sys is not found (as it should be on a proper BBS)

### 2b. advent_test.bat (Testing Only)
A separate test-only batch file that includes fallback door32.sys creation for testing scenarios when no actual BBS is running. This file should NOT be used in production.

### 3. Supporting Files
- config/ directory with configuration files
- art/ directory with all ANSI art files
- Other documentation files (README.md, etc.)

## Talisman BBS Configuration

In your Talisman BBS doors.toml file, add:

```toml
[[menuitem]]
command = "RUNDOOR"
data = "C:\talisman\doors\advent\advent.bat"
hotkey = "A"
```

## How It Works

1. **User selects the door**: User presses 'A' in the doors menu
2. **Talisman calls the door**: Talisman executes: `advent.bat [node] [socket_handle]`
3. **Door32.sys location**: Talisman creates door32.sys in `C:\talisman\temp\[node]\door32.sys`
4. **Batch file processing**: advent.bat:
   - Receives node number and socket handle as parameters
   - Looks for door32.sys in `C:\talisman\temp\[node]\door32.sys`
   - Launches advent.exe with the path to the door32.sys file
   - Logs all activity to advent_door.log for debugging
5. **Door execution**: advent.exe:
   - Parses door32.sys for user information and socket details
   - Connects to the BBS via TCP socket using the provided socket handle
   - Displays the advent calendar interface
   - Handles user input and navigation
   - Exits cleanly when user quits

## Door32.sys Format

The door expects a standard door32.sys file with this format:
```
Line 1: Connection type (2 = socket)
Line 2: BBS name
Line 3: User first name
Line 4: User last name  
Line 5: User alias/handle
Line 6: User security level
Line 7: Time left (minutes)
Line 8: Emulation type (0=TTY, 1=ANSI, 2=Avatar, 3=RIP, 4=Max Graphics)
Line 9: Node number
Line 10: Socket handle
```

## Important: Production vs Testing

### Production Use (Real BBS)
- Use `advent.bat` - this is the production version
- The door will ONLY work if Talisman provides a valid door32.sys file
- The door will exit with an error if door32.sys is missing (as it should)
- NEVER creates its own door32.sys file

### Testing/Development
- Use `advent_test.bat` for testing scenarios
- This version can create fallback door32.sys files for testing
- Use the provided `test_bbs_door.bat` script for comprehensive testing

## Installation Steps

### Quick Production Setup

1. **Copy the production build**:
   The door is pre-configured and ready to use. Simply copy to:
   ```
   C:\talisman\doors\advent\
   ```

2. **Configure your BBS IP** (if needed):
   Edit `C:\talisman\doors\advent\config\config.yaml`:
   ```yaml
   bbs:
     socket_host: "127.0.0.1"  # Use 127.0.0.1 for local BBS
   ```

3. **Add to Talisman doors menu**:
   Edit your doors.toml file to include the menu item shown above.

4. **Test the door**:
   ```bash
   cd C:\talisman\doors\advent
   test_production.bat
   ```

### Files Included
- `advent.exe` - Main door application
- `advent.bat` - Simple BBS launcher (production ready)
- `config/` - Configuration files
- `art/` - All ANSI art assets
- `test_production.bat` - Production testing script

## Testing

### Local Testing
```bash
cd C:\talisman\doors\advent
advent.exe --local
```

### BBS Simulation Testing
Run the test_bbs_door.bat script and choose option 4 for full Talisman simulation.

## Logging and Debugging

The advent.bat file creates advent_door.log with timestamps for:
- Door start events
- Parameter values received from Talisman
- Door32.sys file location and status
- Door completion events

Check this log file if the door doesn't work as expected.

## Socket Connection

The door connects to Talisman via TCP socket using:
- Host: 127.0.0.1 (localhost)
- Port: The socket handle provided by Talisman

If socket connection fails, check:
1. Talisman is properly configured for socket doors
2. Windows firewall settings
3. The socket handle is being passed correctly

## Troubleshooting

### Door doesn't start
- Check that advent.exe exists and is executable
- Verify the path in doors.toml is correct
- Check advent_door.log for error messages

### Socket connection fails
- Verify Talisman socket configuration
- Check Windows firewall
- Ensure port isn't blocked or in use

### User information not displayed correctly
- Check door32.sys file format
- Verify Talisman is creating door32.sys correctly
- Check the door32.sys parsing in the logs

### Art files not displaying
- Ensure art/ directory structure is correct
- Check file permissions
- Verify ANSI emulation settings

## Technical Notes

- The door is built with Go and handles both Windows socket and Linux stdio connections
- ANSI art is displayed with proper CP437 encoding support
- The door respects user time limits from door32.sys
- Session timeouts are configurable in config.yaml
- The door handles 80-column terminal width issues automatically

## Socket Handle Inheritance Implementation ✅

**Implemented**: The door now properly implements Windows socket handle inheritance according to the Door32 specification and OpenDoors reference implementation.

- **Primary approach**: Door inherits the socket handle from the parent BBS process (line 2 of door32.sys)
- **Fallback approach**: If handle inheritance fails, door attempts a new TCP connection
- **Compliance**: Full Door32 specification compliance with proper Windows socket APIs

**Technical Details**:

- Uses Windows WinSock APIs to validate and inherit socket handles
- Implements proper `net.Conn` interface for Go compatibility  
- Graceful fallback ensures compatibility with various BBS configurations
- Comprehensive logging for troubleshooting connection issues

This implementation should work reliably with all BBS systems that properly implement the Door32 specification.