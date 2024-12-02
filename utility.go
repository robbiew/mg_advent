package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/eiannone/keyboard"
)

func DropFileData(path string) (string, int, int, int) {
	// path needs to include trailing slash!
	var dropAlias string
	var dropTimeLeft string
	var dropEmulation string
	var nodeNum string

	file, err := os.Open(strings.ToLower(path + "door32.sys"))
	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	var text []string

	for scanner.Scan() {
		text = append(text, scanner.Text())
	}

	file.Close()

	count := 0
	for _, line := range text {
		if count == 6 {
			dropAlias = line
		}
		if count == 8 {
			dropTimeLeft = line
		}
		if count == 9 {
			dropEmulation = line
		}
		if count == 10 {
			nodeNum = line
		}
		if count == 11 {
			break
		}
		count++
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	timeInt, err := strconv.Atoi(dropTimeLeft) // return as int
	if err != nil {
		log.Fatal(err)
	}

	emuInt, err := strconv.Atoi(dropEmulation) // return as int
	if err != nil {
		log.Fatal(err)
	}
	nodeInt, err := strconv.Atoi(nodeNum) // return as int
	if err != nil {
		log.Fatal(err)
	}

	return dropAlias, timeInt, emuInt, nodeInt
}

/*
Get the terminal size
- Send a cursor position that we know is way too large
- Terminal sends back the largest row + col size
- Read in the result
*/
func getTermSize() (int, int) {
	// Set the terminal to raw mode so we aren't waiting for CLRF rom user (to be undone with `-raw`)
	rawMode := exec.Command("/bin/stty", "raw")
	rawMode.Stdin = os.Stdin
	_ = rawMode.Run()

	reader := bufio.NewReader(os.Stdin)
	fmt.Fprintf(os.Stdout, "\033[999;999f") // larger than any known term size
	fmt.Fprintf(os.Stdout, "\033[6n")       // ansi escape code for reporting cursor location
	text, _ := reader.ReadString('R')

	// Set the terminal back from raw mode to 'cooked'
	rawModeOff := exec.Command("/bin/stty", "-raw")
	rawModeOff.Stdin = os.Stdin
	_ = rawModeOff.Run()
	rawModeOff.Wait()

	// check for the desired output
	if strings.Contains(string(text), ";") {
		re := regexp.MustCompile(`\d+;\d+`)
		line := re.FindString(string(text))

		s := strings.Split(line, ";")
		sh, sw := s[0], s[1]

		ih, err := strconv.Atoi(sh)
		if err != nil {
			// handle error
			fmt.Println(err)
			os.Exit(2)
		}

		iw, err := strconv.Atoi(sw)
		if err != nil {
			// handle error
			fmt.Println(err)
			os.Exit(2)
		}
		h := ih
		w := iw

		ClearScreen()

		return h, w

	} else {
		// couldn't detect, so let's just set 80 x 25 to be safe
		h := 80
		w := 25

		return h, w
	}

}

func InitializeLocal() User {
	if localDisplay {
		return u // Use pre-set user for local mode
	}
	return Initialize(DropPath) // Default path-based initialization
}

// Initialize fetches terminal dimensions and creates a User object
func Initialize(path string) User {
	alias, timeLeft, emulation, nodeNum := DropFileData(path)
	h, w := getTermSize()

	if h%2 == 0 {
		modalH = h
	} else {
		modalH = h - 1
	}

	if w%2 == 0 {
		modalW = w
	} else {
		modalW = w - 1
	}

	timeLeftDuration := time.Duration(timeLeft) * time.Minute

	u := User{
		Alias:     alias,
		TimeLeft:  timeLeftDuration,
		Emulation: emulation,
		NodeNum:   nodeNum,
		H:         h,
		W:         w,
		ModalH:    modalH,
		ModalW:    modalW,
	}
	return u
}

func logDebug(format string, v ...interface{}) {
	if !localDisplay { // Suppress logs in localDisplay mode
		log.Printf(format, v...)
	}
}

// Show the cursor.
func CursorShow() {
	fmt.Print(Esc + "?25h")
}

// Hide the cursor.
func CursorHide() {
	fmt.Print(Esc + "?25l")
}

// Erase the screen
func ClearScreen() {
	fmt.Println(EraseScreen)
	MoveCursor(0, 0)
}

// Move cursor to X, Y location
func MoveCursor(x int, y int) {
	fmt.Printf(Esc+"%d;%df", y, x)
}

// getCurrentYearArtDir generates the subdirectory path for the current year.
func getCurrentYearArtDir() string {
	return filepath.Join(BaseArtDir, strconv.Itoa(time.Now().Year()))
}

func validateArtFiles(artDir string, u User) {
	requiredFiles := []string{
		filepath.Join(artDir, WelcomeFile),
		filepath.Join(artDir, ExitFile),
	}

	// Add daily art files for December 1–25 only
	currentYear := strconv.Itoa(time.Now().Year())
	for day := 1; day <= 25; day++ { // Fix: Only check December 1–25
		fileName := fmt.Sprintf("%d_DEC%s.ANS", day, currentYear[2:]) // e.g., "1_DEC24.ANS"
		requiredFiles = append(requiredFiles, filepath.Join(artDir, fileName))
	}

	// Collect missing files
	missingFiles := []string{}
	for _, file := range requiredFiles {
		if _, err := os.Stat(file); err != nil {
			if os.IsNotExist(err) {
				missingFiles = append(missingFiles, file)
			} else {
				log.Printf("Error accessing file %s: %v", file, err)
			}
		}
	}

	// Handle missing files
	if len(missingFiles) > 0 {
		ClearScreen()

		// Display the missing art message
		missingArtFile := filepath.Join(artDir, MissingFile)
		if _, err := os.Stat(missingArtFile); err == nil {
			displayAnsiFile(missingArtFile, u)
		} else {
			fmt.Println("\nMissing art files detected! Let the Sysop know.")
		}

		// Display missing files on the screen
		fmt.Println("\nThe following art files are missing:")
		for _, file := range missingFiles {
			fmt.Printf(" - %s\n", file)
		}
		pauseForKey()

		// Reset colors and cursor
		fmt.Print("\033[0m") // ANSI escape code to reset text and background
		CursorShow()         // Show the cursor if it was hidden
		os.Exit(1)
	}
}

// validateDate ensures the current date is valid for the advent calendar and displays a "not yet" message if not.
func validateDate(artDir string, u User) {
	now := time.Now()
	if now.Month() != time.December || now.Day() < 1 {
		// Display "not yet" message
		displayAnsiFile(filepath.Join(artDir, NotYet), u)
		pauseForKey()
		// Reset colors and cursor
		fmt.Print("\033[0m") // ANSI escape code to reset text and background
		CursorShow()         // Show the cursor if it was hidden
		os.Exit(1)
	}
}

// parseDebugDate parses and validates the debug date override.
func parseDebugDate(dateStr string) time.Time {
	if dateStr == "" {
		log.Fatalf("debug-date requires a valid date in YYYY-MM-DD format")
	}
	parsedDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		log.Fatalf("Invalid debug-date format: %v", err)
	}
	return parsedDate
}

func pauseForKey() {
	const message = ""
	const row = 24
	const col = 30 // Center the message approximately

	fmt.Printf("\033[%d;%dH%s", row, col, message) // Display the message
	if err := keyboard.Open(); err == nil {
		_, _, _ = keyboard.GetKey() // Wait for a single key press
		keyboard.Close()
	} else {
		logDebug("DEBUG: Keyboard open failed for pause: %v", err)
	}
	fmt.Print("\033[0m") // Reset text and background color
}
