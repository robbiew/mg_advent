# Mistigris Advent Calendar BBS Door

`mg_advent` is an advent calendar BBS Door designed for Linux-based systems. This door displays ANSI art for each day of December, acting as a fun and festive addition to your BBS. It supports both BBS and local modes and includes robust error handling for missing art files or invalid dates.

## Features
- **Dynamic Advent Calendar:** Displays daily ANSI art for December 1–25.
- **Error Handling:** Displays ANSI art for missing files or when accessed before December.
- **BBS and Local Modes:** CP437 encoding for BBS mode and UTF-8 for local terminals.
- **Cross-Platform Capabilities:** Includes a DOS executable for use in DOSBox.
- **Timeout:** Automatically exits after 1 minute of inactivity.

## Contents
- **Go Source Code:**
  - `main.go`: Main entry point for the program.
  - `godoor.go`: Utilities and helpers for the Go door.
- **Example Launch Script:**
  - A shell script for launching with Door32.sys.
- **DOS Executable and Source Code:**
  - `madvent.exe` (DOS executable) and `madvent.pas` (Turbo Pascal source code) for running in DOSBox.
- **ANSI Art Files:**
  - Placeholder ANSI files (`WELCOME.ANS`, `GOODBYE.ANS`, `MISSING.ANS`, and `NOTYET.ANS`) in the `/art` directory.

## Usage
### Build and Run
1. **Build the Go Program:**
   ```bash
   go build -o advent
   ```

2. **Run the Program:**
   - **BBS Mode:**
     Provide the path to the `door32.sys` file:
     ```bash
     ./advent --path /path/to/dropfile/
     ```
   - **Local Mode:**
     Run with the `--local` flag for UTF-8 encoding:
     ```bash
     ./advent --local
     ```

### Directory Structure
- All ANSI files should be placed in the `/art/<year>` directory, where `<year>` is the current year.
  - Example: `art/2023/WELCOME.ANS`

### Required ANSI Files
- `WELCOME.ANS`: Welcome screen.
- `GOODBYE.ANS`: Exit screen.
- `MISSING.ANS`: Error screen for missing art files.
- `NOTYET.ANS`: Error screen for accessing the calendar before December.
- Daily art files: `1_DEC23.ANS` to `25_DEC23.ANS`.

## Notes
- **ANSI Art Requirements:**
  - Files must be 80x25 with the 80th column empty. Ideally, tell users they need to hide the status bar in their Terminal.
- **Timeout:**
  - The user is logged out after 1 minute of inactivity.
- **Encoding:**
  - CP437 encoding is used in BBS mode; UTF-8 is used in local mode.
- **DOS Version:**
  - The DOS version (`madvent.exe`) uses a single `.DAT` file for calendar art.

## Example Shell Script
Here’s an example script to launch the door with Door32.sys:
```bash
#!/bin/bash
/path/to/advent --path /path/to/dropfile/
```

## To-Do
- Allow scrolling for art files larger than 25 rows.
- Add enhanced UTF-8 support for local mode.
