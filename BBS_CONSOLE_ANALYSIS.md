# BBS Door Console Communication Analysis

## Problem Summary
The door opens in a new console window instead of communicating through the user's BBS terminal session.

## Root Cause Analysis

### What We Discovered
1. **Go creates its own console** - When launched, Go applications automatically allocate a console window
2. **Classic DOS doors don't manage sockets** - They rely on the BBS to handle stdin/stdout redirection
3. **Talisman expects stdio communication** - The BBS should redirect the door's I/O to the user session

### Tested Solutions

#### 1. Socket Handle Inheritance ❌
- **Attempted**: Direct socket management with Windows APIs
- **Result**: Socket handles from Talisman are invalid outside BBS context
- **Issue**: Complex and not how classic doors work

#### 2. TCP Connection Fallback ❌  
- **Attempted**: Create new TCP connections when socket inheritance fails
- **Result**: Connection refused (no listening socket)
- **Issue**: Talisman doesn't provide listening sockets for doors

#### 3. Console Application with stdio ❌
- **Attempted**: Force stdio mode, let BBS handle redirection
- **Result**: Still creates console window, output not redirected to user
- **Issue**: Go runtime allocates console regardless

#### 4. GUI Application Build ⚠️
- **Attempted**: Build with `-ldflags "-H windowsgui"`
- **Result**: No console window, but no output visible anywhere
- **Issue**: GUI apps don't inherit console handles properly

## The Real Issue: BBS Door Architecture

### Classic DOS Door Behavior
```
BBS Process (Talisman)
├── Manages user socket connection
├── Launches door with redirected handles  
├── door.exe reads from stdin (user input)
├── door.exe writes to stdout (user display)
└── BBS forwards I/O between user and door
```

### Current Go Door Behavior  
```
BBS Process (Talisman)
├── Launches advent.bat
├── Batch file launches advent.exe
└── advent.exe creates new console (problem!)
    ├── Reads from its own stdin
    ├── Writes to its own stdout  
    └── User sees nothing
```

## Recommended Solutions

### Option 1: Hybrid Approach (Recommended)
- **Sysop Console**: Keep existing console for monitoring/debugging
- **User Output**: Create secondary thread that writes to BBS handles
- **Implementation**: Detect BBS mode and duplicate output streams

### Option 2: Pure stdio Door  
- **Build**: Regular console application
- **Launch**: Use Windows handle redirection in batch file
- **BBS Config**: Configure Talisman for proper stdio door support

### Option 3: Native Windows Door API
- **Research**: How other Windows BBS doors solve this problem
- **Implement**: Use Windows door development frameworks
- **Reference**: Study other successful Windows BBS doors

## Immediate Action Plan

1. **Verify Talisman Configuration**: Ensure Talisman is configured to support stdio doors
2. **Test Handle Redirection**: Create batch file that properly redirects handles
3. **Implement Dual Output**: Modify door to write to both console and stdio
4. **Contact Talisman Developer**: Ask about proper door implementation methods

## Code Examples for Testing

### Test Handle Redirection in Batch
```batch
REM Force handle inheritance
set HANDLE_INHERIT=1
advent.exe --path "%DROPFILE_PATH%" 0<&0 1>&1 2>&2
```

### Test Direct stdio Communication
```batch
REM Pipe door through stdout
advent.exe --path "%DROPFILE_PATH%" | more
```

The core issue is architectural - we need to understand how Talisman expects doors to communicate with users.