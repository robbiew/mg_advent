package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

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

// TimerManager manages timers for idle and max timeout
type TimerManager struct {
	idleTimer    *time.Timer
	maxTimer     *time.Timer
	idleDuration time.Duration
	maxDuration  time.Duration
	lock         sync.Mutex
}

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

const (
	BaseArtDir  = "art"
	WelcomeFile = "WELCOME.ANS"
	ExitFile    = "GOODBYE.ANS"
	MissingFile = "MISSING.ANS"  // Missing art files detected
	NotYet      = "NOTYET.ANS"   // Come back in December
	ComeBack    = "COMEBACK.ANS" // Come back tomorrow to see new art
)

var (
	DropPath          string
	timeOut           time.Duration
	localDisplay      bool
	debugDisableDate  bool
	debugDisableArt   bool
	debugDateOverride string
	modalH            int // in case height is odd
	modalW            int // in case width is odd
)

func init() {
	timeOut = 5 * time.Minute
	pathPtr := flag.String("path", "", "path to door32.sys file (optional if --local is set)")
	// localDisplayPtr := flag.Bool("local", false, "use local UTF-8 display instead of CP437")
	flag.BoolVar(&debugDisableDate, "debug-disable-date", false, "Disable validateDate check")
	flag.BoolVar(&debugDisableArt, "debug-disable-art", false, "Disable validateArtFiles check")
	flag.StringVar(&debugDateOverride, "debug-date", "", "Override date in YYYY-MM-DD format")
	flag.Parse() // Ensure this is executed before any flag is used

	// localDisplay = *localDisplayPtr

	if !localDisplay && *pathPtr == "" {
		fmt.Fprintln(os.Stderr, "missing required -path argument")
		os.Exit(2)
	}
	DropPath = *pathPtr

}

// NewTimerManager creates a new TimerManager with specified durations
func NewTimerManager(idleDuration, maxDuration time.Duration) *TimerManager {
	return &TimerManager{
		idleDuration: idleDuration,
		maxDuration:  maxDuration,
	}
}

// StartIdleTimer starts or resets the idle timer
func (tm *TimerManager) StartIdleTimer() {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	if tm.idleTimer != nil {
		tm.idleTimer.Stop()
	}

	tm.idleTimer = time.AfterFunc(tm.idleDuration, func() {
		fmt.Println("\nYou've been idle for too long... exiting!")
		time.Sleep(2 * time.Second)
		os.Exit(0)
	})
}

// StopIdleTimer stops the idle timer
func (tm *TimerManager) StopIdleTimer() {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	if tm.idleTimer != nil {
		tm.idleTimer.Stop()
	}
}

// StartMaxTimer starts the max timeout timer
func (tm *TimerManager) StartMaxTimer() {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	if tm.maxTimer != nil {
		tm.maxTimer.Stop()
	}

	tm.maxTimer = time.AfterFunc(tm.maxDuration, func() {
		fmt.Println("\nMax time exceeded... exiting!")
		time.Sleep(2 * time.Second)
		os.Exit(0)
	})

}

// StopMaxTimer stops the max timeout timer
func (tm *TimerManager) StopMaxTimer() {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	if tm.maxTimer != nil {
		tm.maxTimer.Stop()
	}
}

// ResetTimers resets both idle and max timers
func (tm *TimerManager) ResetTimers() {
	tm.StartIdleTimer()
	tm.StartMaxTimer()
}

// ResetIdleTimer resets idle timers
func (tm *TimerManager) ResetIdleTimer() {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	if tm.idleTimer != nil {
		tm.idleTimer.Stop()
	}

	tm.idleTimer = time.AfterFunc(tm.idleDuration, func() {
		fmt.Println("\nYou've been idle for too long... exiting!")
		time.Sleep(2 * time.Second)
		os.Exit(0)
	})
}

// ResetMaxTimer resets max timers
func (tm *TimerManager) ResetMaxTimer() {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	if tm.maxTimer != nil {
		tm.maxTimer.Stop()
	}

	tm.maxTimer = time.AfterFunc(tm.maxDuration, func() {
		fmt.Println("\nMax time exceeded... exiting!")
		time.Sleep(2 * time.Second)
		os.Exit(0)
	})
}

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

	// Print the ANSI content, respecting terminal height and width
	printAnsi(content, 0, user.H)
}

// Print ANSI art with a delay between lines and terminal size constraints
func printAnsi(artContent string, delay int, terminalHeight int) {
	noSauce := trimStringFromSauce(artContent) // Strip off the SAUCE metadata
	lines := strings.Split(noSauce, "\r\n")

	// Limit the number of lines printed to the terminal height
	for i := 0; i < len(lines) && i < terminalHeight; i++ {
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

func pauseForKey() {
	const message = ""
	const row = 24
	const col = 30 // Center the message approximately

	fmt.Printf("\033[%d;%dH%s", row, col, message) // Display the message
	if err := keyboard.Open(); err == nil {
		_, _, _ = keyboard.GetKey() // Wait for a single key press
		keyboard.Close()
	} else {
		log.Printf("DEBUG: Keyboard open failed for pause: %v", err)
	}
	fmt.Print("\033[0m") // Reset text and background color
}

func main() {

	u := Initialize(DropPath)
	artDir := getCurrentYearArtDir()

	// Validate date unless disabled by debug flag
	var displayDate time.Time
	if debugDisableDate {
		log.Println("DEBUG: Date validation skipped due to --debug-disable-date flag.")
		if debugDateOverride != "" {
			displayDate = parseDebugDate(debugDateOverride)
			log.Printf("DEBUG: Using debug date override: %s", displayDate.Format("2006-01-02"))
		} else {
			displayDate = time.Date(time.Now().Year(), time.December, 1, 0, 0, 0, 0, time.Local)
			log.Printf("DEBUG: No debug date provided; defaulting to December 1: %s", displayDate.Format("2006-01-02"))
		}
	} else {
		log.Println("DEBUG: Running validateDate.")
		validateDate(artDir, u)
		displayDate = time.Now()
	}

	// Validate art files unless disabled by debug flag
	if !debugDisableArt {
		log.Println("DEBUG: Running validateArtFiles.")
		validateArtFiles(artDir, u)
	}

	// Check for ANSI emulation
	if u.Emulation != 1 {
		log.Println("DEBUG: ANSI emulation required. Exiting.")
		fmt.Println("Sorry, ANSI is required to use this...")
		fmt.Print("\033[0m") // Reset text and background
		CursorShow()         // Show the cursor if hidden
		os.Exit(1)
	}

	// Open keyboard input
	if err := keyboard.Open(); err != nil {
		log.Printf("DEBUG: Keyboard open failed: %v", err)
		os.Exit(1)
	}
	defer keyboard.Close()

	// Start timers
	timerManager := NewTimerManager(timeOut, u.TimeLeft)
	timerManager.StartIdleTimer()
	timerManager.StartMaxTimer()

	CursorHide()
	ClearScreen()
	defer CursorShow()

	// Calculate `maxDay`
	maxDay := displayDate.Day()
	if debugDisableDate && debugDateOverride != "" {
		overrideDate := parseDebugDate(debugDateOverride)
		if overrideDate.Month() == time.December {
			maxDay = overrideDate.Day()
		}
	} else if maxDay > 25 {
		maxDay = 25
	}

	// Initialize starting day based on displayDate
	day := displayDate.Day()
	if day > maxDay {
		day = maxDay
	}

	// Initial state
	welcomeDisplayed := true
	currentDayDisplayed := false
	comebackDisplayed := false

	// Display the welcome screen
	welcomeFilePath := filepath.Join(artDir, WelcomeFile)
	if _, err := os.Stat(welcomeFilePath); err == nil {
		log.Println("DEBUG: Displaying Welcome screen.")
		displayAnsiFile(welcomeFilePath, u)

		todayDate := displayDate.Format("January 2, 2006") // Format the date as "Month Day, Year"
		centeredText := todayDate
		screenWidth := 82 // Assume a standard 80-character wide screen
		x := (screenWidth - len(centeredText)) / 2
		y := 22

		// Move the cursor to the specified X, Y position and print the text
		fmt.Printf("\033[%d;%dH%s", y, x, centeredText) // ANSI escape sequence for cursor positioning
	} else {
		log.Printf("DEBUG: Welcome screen file not found: %s", welcomeFilePath)
	}

	// Start navigation loop
	for {
		char, key, err := keyboard.GetKey()
		if err != nil {
			log.Printf("DEBUG: Keyboard error: %v", err)
			panic(err)
		}

		// Handle Quit (Q or ESC)
		if string(char) == "q" || key == keyboard.KeyEsc {
			log.Println("DEBUG: Exiting on user command.")
			displayAnsiFile(filepath.Join(artDir, ExitFile), u)
			pauseForKey()
			fmt.Print(Reset) // Reset text and background
			CursorShow()     // Show the cursor
			os.Exit(1)
		}

		// Handle Right Arrow Navigation
		if key == keyboard.KeyArrowRight {
			if welcomeDisplayed {
				// Transition from Welcome screen to the user's first day art
				welcomeDisplayed = false
				currentDayDisplayed = true
				log.Printf("DEBUG: Transitioning from Welcome screen to first day (%d).", day)
				displayAnsiFile(filepath.Join(artDir, fmt.Sprintf("%d_DEC%s.ANS", day, strconv.Itoa(displayDate.Year())[2:])), u)
			} else if currentDayDisplayed && day < maxDay {
				day++
				log.Printf("DEBUG: Navigating to day %d.", day)
				displayAnsiFile(filepath.Join(artDir, fmt.Sprintf("%d_DEC%s.ANS", day, strconv.Itoa(displayDate.Year())[2:])), u)
			} else if currentDayDisplayed && day == maxDay && maxDay != 25 {
				currentDayDisplayed = false
				comebackDisplayed = true
				log.Printf("DEBUG: Showing COMEBACK screen for max day (%d).", maxDay)
				// Display the COMEBACK screen
				comebackFilePath := filepath.Join(artDir, ComeBack)
				if _, err := os.Stat(comebackFilePath); err == nil {
					log.Println("DEBUG: Displaying COMEBACK screen.")
					displayAnsiFile(comebackFilePath, u)

					// Add centered text
					var centeredText string
					if maxDay >= 25 {
						centeredText = "See you next year!"
					} else {
						MoveCursor(34, 21)
						fmt.Print("Tomorrow's art")
						tomorrowDate := displayDate.Add(24 * time.Hour).Format("January 2, 2006") // Add one day
						centeredText = tomorrowDate
					}

					screenWidth := 82 // Assume a standard 80-character wide screen
					x := (screenWidth - len(centeredText)) / 2
					y := 22 // Example: Place the text near the bottom of the screen (row 24)

					// Move the cursor to the specified X, Y position and print the text
					fmt.Printf("\033[%d;%dH%s", y, x, centeredText) // ANSI escape sequence for cursor positioning
				} else {
					log.Printf("DEBUG: COMEBACK screen file not found: %s", comebackFilePath)
				}

			} else if comebackDisplayed {
				log.Printf("DEBUG: Right arrow pressed on COMEBACK screen; no action taken.")
			}
		}

		// Handle Left Arrow Navigation
		if key == keyboard.KeyArrowLeft {
			if welcomeDisplayed {
				log.Printf("DEBUG: Left arrow pressed on Welcome screen; no action taken.")
			} else if comebackDisplayed {
				comebackDisplayed = false
				currentDayDisplayed = true
				log.Printf("DEBUG: Navigating back to current day (%d) from COMEBACK screen.", maxDay)
				displayAnsiFile(filepath.Join(artDir, fmt.Sprintf("%d_DEC%s.ANS", maxDay, strconv.Itoa(displayDate.Year())[2:])), u)
			} else if currentDayDisplayed && day > 1 {
				day--
				log.Printf("DEBUG: Navigating to day %d.", day)
				displayAnsiFile(filepath.Join(artDir, fmt.Sprintf("%d_DEC%s.ANS", day, strconv.Itoa(displayDate.Year())[2:])), u)
			} else if currentDayDisplayed && day == 1 {
				currentDayDisplayed = false
				welcomeDisplayed = true
				log.Printf("DEBUG: Navigating to Welcome screen from day %d.", day)
				displayAnsiFile(filepath.Join(artDir, WelcomeFile), u)
			}
		}
	}
}
