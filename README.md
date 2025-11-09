# Mistigris Advent Calendar

This is an interactive ANSI art viewer designed to display an advent calendar with unique art files for each day in December. It supports navigation, year selection, custom date overrides, and enhanced user feedback through centered text messages.

## Features

### General Functionality

- **Daily Art Display**: Displays unique ANSI art files for each day of December.
- **Multi-Year Support**: Browse advent calendars from 2023, 2024, and 2025.
- **Welcome Screen**: On launch, displays a **WELCOME.ANS** file with today's date centered on the screen.
- **Navigation**:
  - Use the **Right Arrow** to navigate forward through days.
  - Use the **Left Arrow** to navigate backward.
  - Press **1** to view 2023 art, **2** for 2024 art (from welcome screen).
  - Arrow keys navigate the current/default year (2025).
- **Quit/Back Navigation**: 
  - Press **Q** or **Esc** while viewing art to return to the Welcome screen.
  - Press **Q** or **Esc** on the Welcome screen to exit and display **GOODBYE.ANS**.
- **BBS-First Output**: Default rendering streams raw CP437 bytes for remote callers; use `--local` to view a Unicode-converted version locally.

### Navigation Highlights

- **"Come Back Tomorrow" Screen**:
  - If navigating beyond the user's current day (but before December 25), the **COMEBACK.ANS** file is displayed with a centered message:
    - `"Tomorrow's art: [date]"` for dates before December 25.
    - `"See you next year!"` for December 25 and beyond.
- **Welcome Screen Navigation**:
  - Pressing the **Right Arrow** transitions to the user's first available day.
  - The **Left Arrow** does nothing when on the Welcome screen.

### Customization and Debugging

- **Command-Line Arguments**:
  - `--path`: Path to the `door32.sys` file (required unless `--local` is used).
  - `--debug-disable-date`: Disables date validation (useful for debugging).
  - `--debug-disable-art`: Skips art file validation.
  - `--debug-date=[YYYY-MM-DD]`: Overrides the current date for testing.
- **Centered Messages**:
  - On the Welcome screen: `"Today's Art: [date]"` (centered on the screen).
  - On the Comeback screen: `"Tomorrow's art: [date]"` or `"See you next year!"`.

### Error Handling

- **Missing Art Files**: Automatically displays **MISSING.ANS** (from `art/common/`) if a day's art file is not found.
  - The missing filename is displayed in the bottom-right corner of the MISSING screen for reference.
- **Idle Timeout**: Exits the program if the user is idle for too long.
- **Max Time Exceeded**: Exits the program if the session duration exceeds the allowed time.

## File Structure

The application uses the following directory structure:

```
art/
├── common/              # Year-independent screen files
│   ├── WELCOME.ANS      # Welcome screen
│   ├── GOODBYE.ANS      # Exit screen
│   ├── COMEBACK.ANS     # "Come Back Tomorrow" screen
│   ├── MISSING.ANS      # Displayed when day art is missing
│   └── NOTYET.ANS       # Future date warning
├── 2023/                # 2023 advent calendar
│   ├── 1_DEC23.ANS - 25_DEC23.ANS
│   ├── INFOFILE.ANS     # Year-specific info
│   └── MEMBERS.ANS      # Year-specific credits
├── 2024/                # 2024 advent calendar
│   ├── 1_DEC24.ANS - 25_DEC24.ANS
│   ├── INFOFILE.ANS
│   └── MEMBERS.ANS
└── 2025/                # 2025 advent calendar
    ├── 1_DEC25.ANS - 25_DEC25.ANS
    ├── INFOFILE.ANS
    └── MEMBERS.ANS
```

### Required Files

- **Common directory** (`art/common/`):
  - `WELCOME.ANS`, `GOODBYE.ANS`, `COMEBACK.ANS`, `MISSING.ANS`, `NOTYET.ANS`
- **Year directories** (e.g., `art/2025/`):
  - Daily art files (`1_DEC25.ANS` to `25_DEC25.ANS`)
  - Year-specific info files: `INFOFILE.ANS`, `MEMBERS.ANS`

## Usage

Run the program with the desired flags:

```bash
./advent --path /path/to/dropfile --debug-disable-date --debug-date=2024-12-12
```

### Command-Line Options

| Option                    | Description                                                                 |
|---------------------------|-----------------------------------------------------------------------------|
| `--path`                  | Path to the `door32.sys` file.                                             |
| `--local`                 | Converts CP437 art to UTF-8 for local terminals (default output is raw CP437). |
| `--debug-disable-date`    | Disables date validation for debugging.                                    |
| `--debug-disable-art`     | Skips art file validation for debugging.                                   |
| `--debug-date=YYYY-MM-DD` | Overrides the current date (useful for testing future days).               |

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
