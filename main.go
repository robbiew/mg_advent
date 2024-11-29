package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/eiannone/keyboard"
)

type User struct {
	Alias     string
	TimeLeft  time.Duration
	Emulation int
	NodeNum   int
	H         int
	W         int
	ModalH    int
	ModalW    int
}

const (
	BaseArtDir  = "art"
	WelcomeFile = "WELCOME.ANS"
	ExitFile    = "GOODBYE.ANS"
	DateFormat  = "_DEC23.ANS"
	MissingFile = "MISSING.ANS" // Missing art files detected
	NotYet      = "NOTYET.ANS"  // Come back in December
)

var (
	DropPath     string
	timeOut      time.Duration
	localDisplay bool
)

func init() {
	timeOut = 1 * time.Minute
	pathPtr := flag.String("path", "", "path to door32.sys file (optional if --local is set)")
	localDisplayPtr := flag.Bool("local", false, "use local UTF-8 display instead of CP437")
	flag.Parse()

	localDisplay = *localDisplayPtr

	if !localDisplay && *pathPtr == "" {
		fmt.Fprintln(os.Stderr, "missing required -path argument")
		os.Exit(2)
	}
	DropPath = *pathPtr
}

// getCurrentYearArtDir generates the subdirectory path for the current year.
func getCurrentYearArtDir() string {
	return filepath.Join(BaseArtDir, strconv.Itoa(time.Now().Year()))
}

// validateArtFiles verifies the presence of all required art files and displays missing art if any are absent.
func validateArtFiles(artDir string) {
	requiredFiles := []string{
		filepath.Join(artDir, WelcomeFile),
		filepath.Join(artDir, ExitFile),
	}

	// Add daily art files for December 1–25
	for day := 1; day <= 25; day++ {
		requiredFiles = append(requiredFiles, filepath.Join(artDir, fmt.Sprintf("%d%s", day, DateFormat)))
	}

	// Check for missing files
	for _, file := range requiredFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			// Display missing art message
			displayAnsiFile(filepath.Join(artDir, MissingFile))
			pauseAndExit()
		}
	}
}

// validateDate ensures the current date is valid for the advent calendar and displays a "not yet" message if not.
func validateDate(artDir string) {
	now := time.Now()
	if now.Month() != time.December || now.Day() < 1 {
		// Display "not yet" message
		displayAnsiFile(filepath.Join(artDir, NotYet))
		pauseAndExit()
	}
}

// pauseAndExit pauses for a key press and then exits.
func pauseAndExit() {
	const message = "Press any key to exit..."
	const row = 21
	const terminalWidth = 80

	// Calculate the starting column to center the message
	startCol := (terminalWidth - len(message)) / 2

	// Clear screen and position cursor at the specified row and column
	fmt.Printf("\033[%d;%dH%s", row, startCol, message)

	// Wait for a key press
	if err := keyboard.Open(); err == nil {
		_, _, _ = keyboard.GetKey()
		keyboard.Close()
	}

	// Exit the program
	os.Exit(1)
}

func ReadAnsiFile(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// displayAnsiFile displays the content of an ANSI file.
func displayAnsiFile(filePath string) {
	content, err := ReadAnsiFile(filePath)
	if err != nil {
		log.Fatalf("Error reading file %s: %v", filePath, err)
	}
	ClearScreen()
	PrintAnsi(content, 0, localDisplay)
}

// main is the program's entry point.
func main() {
	u := Initialize(DropPath)
	artDir := getCurrentYearArtDir()

	validateDate(artDir)
	validateArtFiles(artDir)

	if u.Emulation != 1 {
		fmt.Println("Sorry, ANSI is required to use this...")
		pauseAndExit()
	}

	if err := keyboard.Open(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer keyboard.Close()

	// start idle and max timers
	timerManager := NewTimerManager(timeOut, u.TimeLeft)
	timerManager.StartIdleTimer()
	timerManager.StartMaxTimer()

	CursorHide()
	ClearScreen()
	defer CursorShow()

	displayAnsiFile(filepath.Join(artDir, WelcomeFile))

	for {
		char, key, err := keyboard.GetKey()
		if err != nil {
			panic(err)
		}

		if string(char) == "q" || key == keyboard.KeyEsc {
			displayAnsiFile(filepath.Join(artDir, ExitFile))
			pauseAndExit()
		}
	}
}
