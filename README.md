# MiSTiGRiS Advent Calendar

<img src="images/WELCOME.png" alt="Welcome Screen" width="100%">

An interactive BBS door program that displays ANSI Christmas art, a new one each day in December. Browse past collections as well. Compatible with all modern BBS systems that support Door32 dropfiles -- Windows and Linux.

## Unified Repository Structure (2025+)

This repository now contains all versions of the Advent Calendar:

- **Go/Modern (Linux, Windows, Mac):** Main Go codebase in root (`cmd/`, `internal/`, etc.)
- **DOS (Legacy):** All Turbo Pascal and DOS-specific files are in `dos/` (NOT 100% functional!! see below)

### Directory Layout

```text
mg_advent/
├── cmd/           # Go entry point
├── internal/      # Go core modules
├── art/           # Modern art assets (Go version)
├── dos/           # All DOS/Turbo Pascal code and assets
│   ├── ADVENT.PAS
│   ├── *.DAT
│   ├── build_utils/
│   └── art/
│       ├── 2023/
│       ├── 2024/
│       ├── 2025/
│       └── common/
└── ...
```

## Quick Setup

### Windows

```batch
@echo off
set NODE=%1
if "%NODE%"=="" set NODE=1

cd /d "C:\bbs\doors\advent"
set DROPFILE_PATH=C:\bbs\temp\%NODE%\door32.sys

mg-advent.exe -path "%DROPFILE_PATH%"
```

### Linux

```bash
#!/bin/bash
NODE=${1:-1}
cd /opt/bbs/doors/advent
DROPFILE_PATH="/opt/bbs/temp/${NODE}/door32.sys"

./advent -path "$DROPFILE_PATH"
```

Make executable: `chmod +x advent.sh advent-linux-amd64`

### DOS (Legacy)

All DOS code and assets are in `dos/`. You need Turbo Pascal 7 or compatible.

**To build/run in DOSBox or real DOS:**

1. Enter the `dos/` directory:

 ```text
 cd dos
 ```

1. Open `ADVENT.PAS` in Turbo Pascal 7 and compile, or use batch/scripts in `build_utils/` if available.
2. Run the resulting executable in DOS or DOSBox.

Art assets for DOS are in `dos/art/` (identical structure to modern art/).

## Command Line Options

```text
-path string           Path to door32.sys file
-logon                 Logon mode: show current day's door, then COMEBACK.ANS and exit
-debug-date string     Override date (YYYY-MM-DD)
-debug-disable-date    Disable date validation
-debug-disable-art     Disable art validation
```

## Building from Source

### For Modern Systems (Windows 10+, Linux, Mac)

Prerequisites: Go 1.24 or later, Git

```bash
# Clone repository
git clone https://github.com/robbiew/mg_advent.git
cd mg_advent

# Linux/Mac build (builds Linux amd64, arm64, and Windows 386)
./scripts/build.sh

# Windows build
scripts\build.bat
```

### For Windows 7 32-bit Compatibility

Windows 7 requires Go 1.20 (last version with Windows 7 support).

**One-time setup:**

```bash
# Install Go 1.20.14 toolchain
go install golang.org/dl/go1.20.14@latest
go1.20.14 download
```

**Building:**

```bash
# Linux/Mac (cross-compile)
./scripts/build.sh  # Automatically uses go1.20.14 for Windows builds

# Windows
scripts\build.bat   # Checks for go1.20.14, shows install instructions if needed

# Manual build
GOOS=windows GOARCH=386 CGO_ENABLED=0 ~/go/bin/go1.20.14 build -ldflags="-s -w" -o dist/mg-advent.exe ./cmd/advent
```

**Note:** The build scripts automatically:

- Detect and use Go 1.20.14 for Windows builds (Windows 7 compatibility)
- Embed a Windows application manifest to prevent the 15-20 second delay when renaming executables on Windows 7

**Windows 7 Rename Delay Fix:** The build scripts will automatically embed a manifest if `windres` is available. To enable this:

**Linux/Mac:** `sudo apt-get install mingw-w64` or `brew install mingw-w64`
**Windows:** Install MinGW-w64 from https://www.mingw-w64.org/ or `choco install mingw`

If `windres` is not available, the build will still succeed but renamed executables may experience a 15-20 second startup delay on Windows 7. See [`WINDOWS7-FIX.md`](WINDOWS7-FIX.md) for details.

## Testing

```bash
# Local testing (no BBS required)
# Skip date restrictions to view any day's art
./advent -local -debug-disable-date -debug-date=2024-12-15
```

## Usage

- **Arrow Keys**: Navigate between days
- **1, 2, 3**: Jump to different years (2023, 2024, 2025)
- **Q or ESC**: Return to welcome screen / exit
- **I**: View info file
- **M**: View members list

## License

This project is released under the terms specified in the LICENSE file.

---

### Migration Note (2025)

The `dos/` directory is a work in progress and is not fully functional. The focus is on maintaining the modern Go version. Contributions to the DOS version are welcome but may require significant effort to bring it up to date.
