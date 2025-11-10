# Copilot Instructions - Mistigris Advent Calendar

## Project Overview
This is a Go-based BBS door program that displays ANSI art advent calendars. It's designed as a **BBS-first application** where remote users connect through terminal sessions, with special handling for Windows BBS socket inheritance.

## Architecture & Key Components

### Core Design Pattern
- **Modular architecture**: Clean separation between `cmd/advent/main.go` (entry point) and `internal/*` packages
- **Dual I/O handling**: Raw CP437 for BBS connections, UTF-8 conversion for local terminals
- **CLI-driven**: All configuration via command-line flags with sensible defaults
- **Cross-platform BBS integration**: Windows socket inheritance vs Unix STDIN/STDOUT

### Critical BBS Integration (`internal/bbs/`)
```go
// Windows: Inherits socket handle from parent BBS process
door32Info, err := ParseDoor32(dropfilePath, socketHost)
socketConn, err := CreateSocketFromHandle(door32Info.SocketHandle)

// Linux: Uses STDIN/STDOUT directly
conn.connType = ConnectionStdio
```

**Key Pattern**: All I/O must go through `BBSConnection` - never write directly to `os.Stdout` in BBS mode.

### Display Engine (`internal/display/`)
- **Encoding handling**: `DualModeWriter` sends raw CP437 to BBS, UTF-8 to console
- **Performance**: Caching system for art files (`CacheSizeMB: 50`)
- **Column handling**: Special logic for 80-column terminal issues (`Handle80ColumnIssue: true`)

### Art Management (`internal/art/`)
```
art/
├── common/              # Year-independent screens (WELCOME.ANS, GOODBYE.ANS, etc.)
├── 2023/, 2024/, 2025/  # Year-specific daily art (1_DEC25.ANS format)
```

**Naming Convention**: Daily files use `{day}_DEC{YY}.ANS` (e.g., `1_DEC25.ANS`, `25_DEC25.ANS`)

## Development Workflows

### Building & Testing
```bash
# Multi-platform build with checksums
./scripts/build.sh
# Creates dist/advent-{os}-{arch}[.exe] with .sha256 files

# Testing with coverage
./scripts/test.sh
# Generates coverage.out and coverage.html

# Local testing (bypasses BBS dropfile requirement)
go run cmd/advent/main.go --local --debug-disable-date --debug-date=2024-12-15
```

### Debugging Flags (Essential for Development)
- `--local`: Enables UTF-8 mode for local terminals
- `--debug-disable-date`: Bypasses December-only restriction
- `--debug-date=YYYY-MM-DD`: Override current date for testing
- `--debug-disable-art`: Skip art file validation
- `--path=/path/to/door32.sys`: BBS dropfile path (required in production)

## Project-Specific Patterns

### Navigation State Management (`internal/navigation/`)
```go
type State struct {
    CurrentYear    int        // 2023, 2024, 2025
    CurrentDay     int        // 1-25
    Screen         ScreenType // Welcome, Day, Comeback, etc.
    MaxDay         int        // User's allowed day (today's date)
}
```

**Critical Logic**: Users can only view art up to current date - future dates show "COMEBACK.ANS"

### Configuration Approach
1. **Hard-coded defaults**: 5min timeout, "art/" directory, 50MB cache
2. **CLI flag overrides**: `--local`, `--socket-host`, `--path`, debug flags
3. **No config file**: Eliminates complexity, uses sensible BBS door defaults

### Error Handling Patterns
- **Missing art**: Shows `MISSING.ANS` with filename in bottom-right corner
- **BBS connection failures**: Fatal errors - door cannot function without proper I/O
- **Graceful timeouts**: `timeout_idle: "5m"`, `timeout_max: "120m"`

### Session Management (`internal/session/`)
- **Dual timers**: Idle timeout resets on input, max timeout is absolute
- **BBS compatibility**: Must handle user disconnections gracefully
- **State persistence**: No database - all state is session-based

## Integration Points

### External Dependencies
- **BBS Systems**: Compatible with all modern BBS systems using `door32.sys` dropfile format
- **Terminal Emulation**: Requires ANSI/CP437 support (`emulation_required: 1`)
- **Socket Inheritance**: Windows-specific handle inheritance from parent BBS process

### Memory Bank Documentation
- `memory-bank/`: Contains architectural decisions and modernization roadmap
- Reference for understanding evolution from monolithic to modular design
- Documents the "why" behind current structure decisions

## Common Pitfalls
- **Never** write to `os.Stdout` in BBS mode - always use `BBSConnection`
- **Art validation** is required - missing files break user experience
- **Date validation** is enforced by default - use debug flags for testing
- **Cross-platform differences**: Windows uses socket inheritance, Linux uses STDIO
- **Column handling**: 80-column terminals need special care for proper display

## Current Development Focus (2025)
- Multi-year navigation improvements
- Enhanced theming system (`internal/display/theme.go`)
- Performance optimizations for art caching
- Better error recovery and user feedback