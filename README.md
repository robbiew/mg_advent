# Mistigris Advent Calendar

An interactive BBS door program that displays beautiful ANSI art advent calendars. Compatible with all modern BBS systems that support Door32 dropfiles.

## Features

- **Daily Art Display**: Unique ANSI art for each day in December
- **Multi-Year Support**: Browse advent calendars from 2023, 2024, and 2025
- **Universal BBS Compatibility**: Works with any BBS system using door32.sys
- **Interactive Navigation**: Arrow keys to browse days, year selection
- **Session Management**: Configurable timeouts and graceful disconnection handling
- **Cross-Platform**: Runs on Windows and Linux BBS systems

## User Experience

- **Navigation**: Use arrow keys to browse days, press Q to quit
- **Smart Restrictions**: Only shows art up to current date, future dates show "come back tomorrow"
- **Year Selection**: Press 1, 2, or 3 to switch between available years
- **Error Handling**: Graceful fallbacks for missing art files
- **Session Timeouts**: Configurable idle and maximum session limits

## Installation

See [INSTALLATION.md](INSTALLATION.md) for complete setup instructions for Windows and Linux BBS systems.

## Building from Source

### Prerequisites
- Go 1.24 or later
- Git

### Build Steps
```bash
# Clone repository
git clone https://github.com/robbiew/mg_advent.git
cd mg_advent

# Linux/Mac build
./build.sh

# Windows build  
build.bat
```

This creates binaries in the `dist/` directory for multiple platforms.

### Development & Testing
```bash
# Local testing (no BBS required)
go run cmd/advent/main.go --local --debug-disable-date --debug-date=2024-12-15
```

## Contributing

This project welcomes contributions! Please see the development documentation in the `memory-bank/` directory for architecture details and development guidelines.

## License

This project is released under the terms specified in the LICENSE file.

## Dependencies

- **Golang Modules**:
  - `github.com/eiannone/keyboard`: For capturing user input.

## Example Workflow

1. **Launch**: Displays the Welcome screen with today's date centered.
2. **Year Selection**: 
   - Press **1** to jump to 2023 advent calendar.
   - Press **2** to jump to 2024 advent calendar.
   - Press **Right Arrow** to enter current year (2025) calendar.
3. **Navigate**:
   - Use the **Right Arrow** to proceed to the next day.
   - Use the **Left Arrow** to navigate backward.
   - Press **Q** or **Esc** to return to Welcome screen.
4. **Quit**: Press **Q** or **Esc** on Welcome screen to exit and display the Goodbye screen.

## Recent Updates (2025)

- **Refactored Art Structure**: Separated year-independent screens into `art/common/` directory
- **Multi-Year Navigation**: Added numeric key selection (1, 2) to browse previous years
- **Improved Navigation**: Q/ESC returns to Welcome screen instead of exiting
- **Missing Art Fallback**: Automatically displays MISSING.ANS when day art is not found
  - Missing filename shown in bottom-right corner for debugging
- **Year Independence**: Common screens (Welcome, Goodbye, etc.) are now shared across all years

## Future Enhancements

- Additional year archives (2026+)
- Enhanced scrolling for longer art pieces
- Additional navigation shortcuts
