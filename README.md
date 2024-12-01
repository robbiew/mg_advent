# Mistigris Advent Calendar

This is an interactive ANSI art viewer designed to display an advent calendar with unique art files for each day in December. It supports navigation, custom date overrides, and enhanced user feedback through centered text messages.

## Features

### General Functionality

- **Daily Art Display**: Displays unique ANSI art files for each day of December.
- **Welcome Screen**: On launch, displays a **WELCOME.ANS** file with today's date centered on the screen.
- **Navigation**:
  - Use the **Right Arrow** to navigate forward through days.
  - Use the **Left Arrow** to navigate backward.
- **Quit Option**: Press **Q** or **Esc** to exit the program gracefully, displaying the **GOODBYE.ANS** screen.

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

- **Missing Art Files**: Detects and lists missing art files in a readable format. Displays **MISSING.ANS** if available.
- **Idle Timeout**: Exits the program if the user is idle for too long.
- **Max Time Exceeded**: Exits the program if the session duration exceeds the allowed time.

## File Requirements

Ensure the following files are present in the appropriate directory (`art/2024` for the current year):

- **WELCOME.ANS**: Welcome screen art.
- **GOODBYE.ANS**: Exit screen art.
- **COMEBACK.ANS**: "Come Back Tomorrow" screen art.
- Daily art files (`1_DEC24.ANS` to `25_DEC24.ANS`).

## Usage

Run the program with the desired flags:

```bash
./advent --path /path/to/dropfile --debug-disable-date --debug-date=2024-12-12
```

### Command-Line Options

| Option                    | Description                                                                 |
|---------------------------|-----------------------------------------------------------------------------|
| `--path`                  | Path to the `door32.sys` file.                                             |
| `--local`                 | Enables local UTF-8 display instead of CP437 encoding.                     |
| `--debug-disable-date`    | Disables date validation for debugging.                                    |
| `--debug-disable-art`     | Skips art file validation for debugging.                                   |
| `--debug-date=YYYY-MM-DD` | Overrides the current date (useful for testing future days).               |

## Dependencies

- **Golang Modules**:
  - `github.com/eiannone/keyboard`: For capturing user input.

## Example Workflow

1. **Launch**: Displays the Welcome screen with today's date centered.
2. **Navigate**:
   - Use the **Right Arrow** to proceed to the first available day or the next day.
   - Use the **Left Arrow** to navigate backward.
3. **Quit**: Press **Q** or **Esc** to exit and display the Goodbye screen.

## Future Enhancements

- Support for scrolling beyond 80x24
- Browse previous years art
- UTF-8 (local) support
