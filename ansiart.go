package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

const (
	Esc = "\u001B["
	Osc = "\u001B]"
	Bel = "\u0007"
)

// Common ANSI escapes sequences. This is not a complete list.
const (
	CursorBackward = Esc + "D"
	CursorPrevLine = Esc + "F"
	CursorLeft     = Esc + "G"
	CursorTop      = Esc + "d"
	CursorTopLeft  = Esc + "H"

	CursorBlinkEnable  = Esc + "?12h"
	CursorBlinkDisable = Esc + "?12I"

	ScrollUp   = Esc + "S"
	ScrollDown = Esc + "T"

	TextInsertChar = Esc + "@"
	TextDeleteChar = Esc + "P"
	TextEraseChar  = Esc + "X"
	TextInsertLine = Esc + "L"
	TextDeleteLine = Esc + "M"

	EraseRight  = Esc + "K"
	EraseLeft   = Esc + "1K"
	EraseLine   = Esc + "2K"
	EraseDown   = Esc + "J"
	EraseUp     = Esc + "1J"
	EraseScreen = Esc + "2J"

	Black     = Esc + "30m"
	Red       = Esc + "31m"
	Green     = Esc + "32m"
	Yellow    = Esc + "33m"
	Blue      = Esc + "34m"
	Magenta   = Esc + "35m"
	Cyan      = Esc + "36m"
	White     = Esc + "37m"
	BlackHi   = Esc + "30;1m"
	RedHi     = Esc + "31;1m"
	GreenHi   = Esc + "32;1m"
	YellowHi  = Esc + "33;1m"
	BlueHi    = Esc + "34;1m"
	MagentaHi = Esc + "35;1m"
	CyanHi    = Esc + "36;1m"
	WhiteHi   = Esc + "37;1m"

	BgBlack     = Esc + "40m"
	BgRed       = Esc + "41m"
	BgGreen     = Esc + "42m"
	BgYellow    = Esc + "43m"
	BgBlue      = Esc + "44m"
	BgMagenta   = Esc + "45m"
	BgCyan      = Esc + "46m"
	BgWhite     = Esc + "47m"
	BgBlackHi   = Esc + "40;1m"
	BgRedHi     = Esc + "41;1m"
	BgGreenHi   = Esc + "42;1m"
	BgYellowHi  = Esc + "43;1m"
	BgBlueHi    = Esc + "44;1m"
	BgMagentaHi = Esc + "45;1m"
	BgCyanHi    = Esc + "46;1m"
	BgWhiteHi   = Esc + "47;1m"

	Reset = Esc + "0m"
)

func readAnsiFile(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// displayAnsiFile displays the content of an ANSI file.
func displayAnsiFile(filePath string, user User) {
	fmt.Print(Reset) // Reset text and background colors before displaying new art
	ClearScreen()    // Clear the screen

	// Read the ANSI file content
	content, err := readAnsiFile(filePath)
	if err != nil {
		log.Printf("ERROR: Failed to read ANSI file %s: %v", filePath, err)
		fmt.Println("Error: Unable to load art. Please contact the Sysop.")
		return
	}

	if content == "" {
		log.Printf("ERROR: ANSI file %s is empty or unreadable.", filePath)
		fmt.Println("Error: The art file is empty or invalid.")
		return
	}

	// Call the appropriate rendering function based on the display mode
	if localDisplay {
		printUtf8(content, 0, user.H) // Pass height for rendering
	} else {
		printAnsi(content, 0, user.H) // Use standard ANSI printing logic
	}
}

// Print UTF-8 art
func printUtf8(artContent string, delay int, terminalHeight int) {
	noSauce := trimStringFromSauce(artContent) // Strip off the SAUCE metadata
	lines := strings.Split(noSauce, "\r\n")

	// Render the content line by line, respecting the terminal height
	for i := 0; i < terminalHeight && i < len(lines); i++ {
		line := lines[i]

		// Convert line from CP437 to UTF-8
		utf8Line, err := charmap.CodePage437.NewDecoder().String(line)
		if err != nil {
			log.Printf("Error converting to UTF-8: %v", err)
			utf8Line = line // Fallback to original line
		}

		// Print the line
		if i == terminalHeight-1 {
			// Avoid newline for the last line within terminal height
			fmt.Print(utf8Line)
		} else {
			fmt.Println(utf8Line)
		}

		// Optional delay between lines
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}
}

// Print ANSI art
func printAnsi(artContent string, delay int, terminalHeight int) {
	noSauce := trimStringFromSauce(artContent) // Strip off the SAUCE metadata
	lines := strings.Split(noSauce, "\r\n")

	// Render the content line by line, respecting the terminal height
	for i := 0; i < terminalHeight && i < len(lines); i++ {
		line := lines[i]

		// Print the line
		if i == terminalHeight-1 {
			// Avoid newline for the last line within terminal height
			fmt.Print(line)
		} else {
			fmt.Println(line)
		}

		// Optional delay between lines
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}
}

// TrimStringFromSauce trims SAUCE metadata from a string.
func trimStringFromSauce(s string) string {
	return trimMetadata(s, "COMNT", "SAUCE00")
}

// trimMetadata trims metadata based on delimiters.
func trimMetadata(s string, delimiters ...string) string {
	for _, delimiter := range delimiters {
		if idx := strings.Index(s, delimiter); idx != -1 {
			return trimLastChar(s[:idx])
		}
	}
	return s
}

// trimLastChar trims the last character from a string.
func trimLastChar(s string) string {
	if len(s) > 0 {
		_, size := utf8.DecodeLastRuneInString(s)
		return s[:len(s)-size]
	}
	return s
}
