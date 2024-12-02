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
	MissingFile = "MISSING.ANS"  // Missing art files detected
	NotYet      = "NOTYET.ANS"   // Come back in December
	ComeBack    = "COMEBACK.ANS" // Come back tomorrow to see new art
)

var (
	DropPath          string
	timeOut           time.Duration
	debugDisableDate  bool
	debugDisableArt   bool
	debugDateOverride string
	modalH            int // in case height is odd
	modalW            int // in case width is odd
	localDisplay      bool
	u                 User // Global User object
)

func init() {
	timeOut = 5 * time.Minute
	pathPtr := flag.String("path", "", "path to door32.sys file (required unless --local is set)")
	flag.BoolVar(&debugDisableDate, "debug-disable-date", false, "Disable validateDate check")
	flag.BoolVar(&debugDisableArt, "debug-disable-art", false, "Disable validateArtFiles check")
	flag.StringVar(&debugDateOverride, "debug-date", "", "Override date in YYYY-MM-DD format")
	flag.BoolVar(&localDisplay, "local", false, "Use local UTF-8 display instead of CP437")
	flag.Parse()

	if localDisplay {
		log.Println("INFO: Running in local mode. No DropPath required.")
		u = User{
			Alias:     "SysOp",
			TimeLeft:  120 * time.Minute,
			Emulation: 1,
			NodeNum:   1,
			H:         25,
			W:         80,
			ModalH:    25,
			ModalW:    80,
		}
	} else {
		if *pathPtr == "" {
			log.Fatal("Error: --path is required unless --local is set.")
		}
		DropPath = *pathPtr
		log.Printf("INFO: Running in DropPath mode. Path set to %s.", DropPath)
	}
}

func main() {
	if localDisplay {
		u = InitializeLocal()
	} else {
		// Ensure DropPath-based initialization only happens when not in local mode
		u = Initialize(DropPath)
	}
	artDir := getCurrentYearArtDir()

	// Setup callbacks for timeouts
	onIdle := func() {
		fmt.Println("\nIdle timeout reached... exiting.")
		os.Exit(0)
	}

	onMax := func() {
		fmt.Println("\nMaximum session time reached... exiting.")
		os.Exit(0)
	}

	// Initialize and start timers
	timerManager := NewTimerManager(timeOut, u.TimeLeft, onIdle, onMax)
	timerManager.StartTimers()

	// Validate date unless disabled by debug flag
	var displayDate time.Time
	if debugDisableDate {
		logDebug("DEBUG: Date validation skipped due to --debug-disable-date flag.")
		if debugDateOverride != "" {
			displayDate = parseDebugDate(debugDateOverride)
			logDebug("DEBUG: Using debug date override: %s", displayDate.Format("2006-01-02"))
		} else {
			displayDate = time.Date(time.Now().Year(), time.December, 1, 0, 0, 0, 0, time.Local)
			logDebug("DEBUG: No debug date provided; defaulting to December 1: %s", displayDate.Format("2006-01-02"))
		}
	} else {
		logDebug("DEBUG: Running validateDate.")
		validateDate(artDir, u)
		displayDate = time.Now()
	}

	// Validate art files unless disabled by debug flag
	if !debugDisableArt {
		logDebug("DEBUG: Running validateArtFiles.")
		validateArtFiles(artDir, u)
	}

	// Check for ANSI emulation
	if u.Emulation != 1 {
		logDebug("DEBUG: ANSI emulation required. Exiting.")
		fmt.Println("Sorry, ANSI is required to use this...")
		fmt.Print("\033[0m") // Reset text and background
		CursorShow()         // Show the cursor if hidden
		os.Exit(1)
	}

	// Open keyboard input
	if err := keyboard.Open(); err != nil {
		logDebug("DEBUG: Keyboard open failed: %v", err)
		os.Exit(1)
	}
	defer keyboard.Close()

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
		logDebug("DEBUG: Displaying Welcome screen.")
		displayAnsiFile(welcomeFilePath, u)

		todayDate := displayDate.Format("January 02, 2006") // Format the date as "Month Day, Year"
		centeredText := todayDate
		screenWidth := 82 // Assume a standard 80-character wide screen
		x := (screenWidth - len(centeredText)) / 2
		y := 20

		// Move the cursor to the specified X, Y position and print the text
		fmt.Printf("\033[%d;%dH%s", y, x, centeredText) // ANSI escape sequence for cursor positioning
	} else {
		logDebug("DEBUG: Welcome screen file not found: %s", welcomeFilePath)
	}

	// Start navigation loop
	for {
		char, key, err := keyboard.GetKey()
		if err != nil {
			logDebug("DEBUG: Keyboard error: %v", err)
			panic(err)
		}

		// Reset idle timer on any key press
		timerManager.ResetIdleTimer()

		// Handle Quit (Q or ESC)
		if string(char) == "q" || key == keyboard.KeyEsc {
			logDebug("DEBUG: Exiting on user command.")
			displayAnsiFile(filepath.Join(artDir, ExitFile), u)
			pauseForKey()
			fmt.Print(Reset) // Reset text and background
			CursorShow()     // Show the cursor
			os.Exit(1)
		}

		// Handle menu selection for 2023 Calendar
		if welcomeDisplayed && string(char) == "1" {
			// Set up for the 2023 calendar
			artDir = filepath.Join(BaseArtDir, "2023")
			maxDay = 25 // No date restrictions for 2023
			day = 1
			currentDayDisplayed = true
			welcomeDisplayed = false // Exit the welcome menu
			comebackDisplayed = false
			logDebug("DEBUG: Switching to 2023 Calendar.")
			displayAnsiFile(filepath.Join(artDir, fmt.Sprintf("%d_DEC23.ANS", day)), u)
		}

		// Navigation logic for the 2024 calendar (default)
		if artDir == filepath.Join(BaseArtDir, "2024") {
			// Handle Right Arrow Navigation
			if key == keyboard.KeyArrowRight {
				if welcomeDisplayed {
					// Transition from Welcome screen to the user's first day art
					welcomeDisplayed = false
					currentDayDisplayed = true
					logDebug("DEBUG: Transitioning from Welcome screen to first day (%d).", day)
					displayAnsiFile(filepath.Join(artDir, fmt.Sprintf("%d_DEC%s.ANS", day, strconv.Itoa(displayDate.Year())[2:])), u)
				} else if currentDayDisplayed && day < maxDay {
					day++
					logDebug("DEBUG: Navigating to day %d.", day)
					displayAnsiFile(filepath.Join(artDir, fmt.Sprintf("%d_DEC%s.ANS", day, strconv.Itoa(displayDate.Year())[2:])), u)
				} else if currentDayDisplayed && day == maxDay && maxDay != 25 {
					currentDayDisplayed = false
					comebackDisplayed = true
					logDebug("DEBUG: Showing COMEBACK screen for max day (%d).", maxDay)
					// Display the COMEBACK screen
					comebackFilePath := filepath.Join(artDir, ComeBack)
					if _, err := os.Stat(comebackFilePath); err == nil {
						logDebug("DEBUG: Displaying COMEBACK screen.")
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
						logDebug("DEBUG: COMEBACK screen file not found: %s", comebackFilePath)
					}

				} else if comebackDisplayed {
					logDebug("DEBUG: Right arrow pressed on COMEBACK screen; no action taken.")
				}
			}

			// Handle Left Arrow Navigation
			if key == keyboard.KeyArrowLeft {
				if welcomeDisplayed {
					logDebug("DEBUG: Left arrow pressed on Welcome screen; no action taken.")
				} else if comebackDisplayed {
					comebackDisplayed = false
					currentDayDisplayed = true
					logDebug("DEBUG: Navigating back to current day (%d) from COMEBACK screen.", maxDay)
					displayAnsiFile(filepath.Join(artDir, fmt.Sprintf("%d_DEC%s.ANS", maxDay, strconv.Itoa(displayDate.Year())[2:])), u)
				} else if currentDayDisplayed && day > 1 {
					day--
					logDebug("DEBUG: Navigating to day %d.", day)
					displayAnsiFile(filepath.Join(artDir, fmt.Sprintf("%d_DEC%s.ANS", day, strconv.Itoa(displayDate.Year())[2:])), u)
				} else if currentDayDisplayed && day == 1 {
					currentDayDisplayed = false
					welcomeDisplayed = true
					logDebug("DEBUG: Navigating to Welcome screen from day %d.", day)
					displayAnsiFile(filepath.Join(artDir, WelcomeFile), u)
					todayDate := displayDate.Format("January 02, 2006") // Format the date as "Month Day, Year"
					centeredText := todayDate
					screenWidth := 82 // Assume a standard 80-character wide screen
					x := (screenWidth - len(centeredText)) / 2
					y := 20

					// Move the cursor to the specified X, Y position and print the text
					fmt.Printf("\033[%d;%dH%s", y, x, centeredText) // ANSI escape sequence for cursor positioning
				}
			}
		}

		// Navigation logic for the 2023 calendar
		if artDir == filepath.Join(BaseArtDir, "2023") {
			if string(char) == "1" && comebackDisplayed && artDir == filepath.Join(BaseArtDir, "2023") {
				// Transition back to the 2024 WELCOME screen
				artDir = filepath.Join(BaseArtDir, "2024")
				welcomeDisplayed = true
				currentDayDisplayed = false
				comebackDisplayed = false

				// Recalculate today's date logic for the 2024 calendar
				displayDate := time.Now()
				if debugDisableDate && debugDateOverride != "" {
					overrideDate := parseDebugDate(debugDateOverride)
					if overrideDate.Year() == 2024 && overrideDate.Month() == time.December {
						displayDate = overrideDate
					}
				}

				maxDay = displayDate.Day()
				if displayDate.Month() != time.December || maxDay > 25 {
					maxDay = 25 // Restrict to December 25
				}

				// Reset the `day` variable to the current day
				day = displayDate.Day()
				if day > maxDay {
					day = maxDay
				}

				logDebug("DEBUG: User returned to 2024 WELCOME screen from 2023 COMEBACK menu.")
				displayAnsiFile(filepath.Join(artDir, WelcomeFile), u)
				todayDate := displayDate.Format("January 02, 2006") // Format the date as "Month Day, Year"
				centeredText := todayDate
				screenWidth := 82 // Assume a standard 80-character wide screen
				x := (screenWidth - len(centeredText)) / 2
				y := 20

				// Move the cursor to the specified X, Y position and print the text
				fmt.Printf("\033[%d;%dH%s", y, x, centeredText) // ANSI escape sequence for cursor positioning
				continue
			}
			if key == keyboard.KeyArrowRight {
				if comebackDisplayed {
					// Right arrow is disabled on the COMEBACK screen
					logDebug("DEBUG: Right arrow key pressed on COMEBACK screen for 2023; no action taken.")
					continue
				} else if currentDayDisplayed && day < maxDay {
					day++
					logDebug("DEBUG: Navigating to day %d in 2023 Calendar.", day)
					displayAnsiFile(filepath.Join(artDir, fmt.Sprintf("%d_DEC23.ANS", day)), u)
				} else if currentDayDisplayed && day == maxDay {
					currentDayDisplayed = false
					comebackDisplayed = true
					logDebug("DEBUG: Showing COMEBACK screen for 2023 Calendar.")
					// Display the COMEBACK screen for 2023
					comebackFilePath := filepath.Join(artDir, ComeBack)
					if _, err := os.Stat(comebackFilePath); err == nil {
						log.Println("DEBUG: Displaying COMEBACK screen for 2023.")
						displayAnsiFile(comebackFilePath, u)
					} else {
						logDebug("DEBUG: COMEBACK screen file not found for 2023: %s", comebackFilePath)
					}
				}
			} else if key == keyboard.KeyArrowLeft {
				if comebackDisplayed {
					// Transition back to the last day from COMEBACK screen
					comebackDisplayed = false
					currentDayDisplayed = true
					logDebug("DEBUG: Navigating back to day %d from COMEBACK screen in 2023 Calendar.", maxDay)
					displayAnsiFile(filepath.Join(artDir, fmt.Sprintf("%d_DEC23.ANS", maxDay)), u)
				} else if day > 1 {
					day--
					logDebug("DEBUG: Navigating to day %d in 2023 Calendar.", day)
					displayAnsiFile(filepath.Join(artDir, fmt.Sprintf("%d_DEC23.ANS", day)), u)
				} else if day == 1 {
					// Transition back to the 2024 WELCOME screen
					artDir = filepath.Join(BaseArtDir, "2024")
					welcomeDisplayed = true
					currentDayDisplayed = false

					// Recalculate today's date logic for the 2024 calendar
					displayDate := time.Now()
					if debugDisableDate && debugDateOverride != "" {
						overrideDate := parseDebugDate(debugDateOverride)
						if overrideDate.Year() == 2024 && overrideDate.Month() == time.December {
							displayDate = overrideDate
						}
					}

					maxDay = displayDate.Day()
					if displayDate.Month() != time.December || maxDay > 25 {
						maxDay = 25 // Restrict to December 25
					}

					// Reset the `day` variable to the current day
					day = displayDate.Day()
					if day > maxDay {
						day = maxDay
					}

					logDebug("DEBUG: Navigating back to 2024 WELCOME screen. Today's day: %d, Max day: %d.", day, maxDay)
					displayAnsiFile(filepath.Join(artDir, WelcomeFile), u)

				}
			}
		}
	}
}
