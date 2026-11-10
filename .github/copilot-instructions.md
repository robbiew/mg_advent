# Copilot Instructions - Mistigris Advent Calendar

## Project Overview
This is a Go-based BBS door program that displays ANSI art advent calendars. It's designed as a **BBS-first application** where remote users connect through terminal sessions, with special handling for Windows BBS socket inheritance.

## Architecture & Key Components

### Core Design Pattern
- **Modular architecture**: Clean separation between `cmd/advent/main.go` (entry point) and `internal/*` packages
- **Dual I/O handling**: Raw CP437 for BBS connections, UTF-8 conversion for local terminals
- **CLI-driven**: All configuration via command-line flags with sensible defaults
- **Cross-platform BBS integration**: Windows socket inheritance vs Unix STDIN/STDOUT

### Critical BBS Integration (GoDoors)
```go
// main converts -path's filename to its containing directory.
info, err := gd.OpenSession(directory) // no probes until explicitly requested
// Pair with gd.CloseSession(), use gd.ReadKey()/ReadKeyTimeout(), gd.Output().
```

GoDoors owns inherited Windows sockets and standard-I/O selection. There is no
application-specific socket implementation. Never write caller-facing text
straight to os.Stdout in BBS mode; use gd.Output() or DisplayEngine.

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
go run ./cmd/advent -local -debug-date=2026-12-15
```

### Debugging Flags (Essential for Development)
- `--local`: Enables UTF-8 mode for local terminals
- `--debug-disable-date`: Unlocks every installed year and day
- `--debug-date=YYYY-MM-DD`: Pretend today is this date (calendar only; session timers use the real clock)
- `--path=/path/to/door32.sys`: BBS dropfile path (required in production)

## Project-Specific Patterns

### Navigation State Management (`internal/navigation/`)
```go
type State struct {
    CurrentYear    int        // 2023, 2024, 2025
    CurrentDay     int        // 1-25
    Screen         ScreenType // Welcome, Day, Comeback, etc.
    MaxDay         int        // Days of CurrentYear open today (Navigator.MaxDay)
}
```

**Critical Logic** (`Navigator.MaxDay`): past years are open in full all year;
the current year opens one day at a time in December (paging past today shows
"COMEBACK.ANS"); future years are hidden. Never gate the whole door on the
month - off-season callers browse the archive.

### Configuration Approach
1. **Hard-coded defaults**: 5min timeout, "art/" directory, 50MB cache
2. **CLI flag overrides**: `--local`, `--socket-host`, `--path`, debug flags
3. **No config file**: Eliminates complexity, uses sensible BBS door defaults

### Error Handling Patterns
- **Missing art**: Shows `MISSING.ANS` with filename in bottom-right corner
- **BBS connection failures**: Fatal errors - door cannot function without proper I/O
- **Graceful timeouts**: 5 min idle; total capped at the caller's door32.sys time left (max 120 min)

### Session Management (`internal/session/`)
- **No timers or goroutines**: `Manager.Next` gives the input wait; the main loop reads with that timeout and exits through normal cleanup
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
- **Never** write to `os.Stdout` in BBS mode - always use GoDoors or the display engine
- **Art fallbacks**: year screens fall back to `art/common`, missing days show `MISSING.ANS`; only common MISSING/NOTYET are required
- **Cross-platform differences**: Windows uses socket inheritance, Linux uses STDIO
- **Column handling**: 80-column terminals need special care for proper display

## Current Development Focus (2025)
- Multi-year navigation improvements
- Enhanced theming system (`internal/display/theme.go`)
- Performance optimizations for art caching
- Better error recovery and user feedback