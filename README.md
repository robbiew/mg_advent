# MiSTiGRiS Advent Calendar

> **✨ UPDATED FOR 2025 ✨**

<img src="images/WELCOME.png" alt="Welcome Screen" width="100%">

An interactive BBS door program that displays ANSI Christmas art, a new ones each day in December. Browse past collections as well. Compatible with all modern BBS systems that support Door32 dropfiles -- Windows and Linux.

## Features

- **Daily Art Display**: Unique ANSI art for each day in December
- **Multi-Year Support**: Browse advent calendars from 2023, 2024, and 2025
- **BBS Compatibility**: Works with Linux/Windows BBS systems using door32.sys

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
# Local testing (no BBS required), allows you to skip the current date restrictions:
go run cmd/advent/main.go --local --debug-disable-date --debug-date=2024-12-15
```

## License

This project is released under the terms specified in the LICENSE file.


## Recent Updates (2025)

- **Windows 32 Support**: Beta testing socket inheritence in Go!
- **Refactored Art Structure**: Separated year-independent screens into `art/common/` directory
- **Art Bundled with Binary**: No more art/ dir needed, it's all compiled in at build-time
- **Multi-Year Navigation**: Added numeric key selection (1, 2) to browse previous years
- **Improved Navigation**: Q/ESC returns to Welcome screen instead of exiting
- **Missing Art Fallback**: Automatically displays MISSING.ANS when day art is not found
  - Missing filename shown in bottom-right corner for debugging
- **Year Independence**: Common screens (Welcome, Goodbye, etc.) are now shared across all years for future ease

## Future Enhancements

- Additional year archives (2026+)
- Enhanced scrolling for wider/longer art pieces
