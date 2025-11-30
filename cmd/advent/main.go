package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/term"

	"github.com/robbiew/advent/internal/art"
	"github.com/robbiew/advent/internal/bbs"
	"github.com/robbiew/advent/internal/display"
	"github.com/robbiew/advent/internal/embedded"
	"github.com/robbiew/advent/internal/input"
	"github.com/robbiew/advent/internal/navigation"
	"github.com/robbiew/advent/internal/session"
	"github.com/robbiew/advent/internal/validation"
)

// Version information - can be set at build time with ldflags
// Example: go build -ldflags "-X main.version=v1.0.0 -X main.buildDate=2024-12-01"
var (
	version   = "v1.0.0"
	buildDate = "unknown"
)

var (
	// Command line flags
	localMode    = flag.Bool("local", false, "run in local UTF-8 mode")
	debugDate    = flag.String("debug-date", "", "override date (YYYY-MM-DD)")
	disableDate  = flag.Bool("debug-disable-date", false, "disable date validation")
	dropfilePath = flag.String("path", "", "path to door32.sys file")
	logonMode    = flag.Bool("logon", false, "logon mode: show current day's door, then COMEBACK.ANS and exit")
	showVersion  = flag.Bool("version", false, "show version information")
)

func main() {
	startTime := time.Now()

	// CRITICAL Windows 7 FIX: Write to stderr FIRST to force console initialization
	// Windows 7 console buffers stdout for 20 seconds unless console is "active"
	// Stderr write forces console activation
	if runtime.GOOS == "windows" {
		fmt.Fprintf(os.Stderr, "") // Write to stderr forces console init
		os.Stderr.Sync()
	}

	flag.Parse()

	// Handle version flag
	if *showVersion {
		fmt.Printf("MiSTiGRiS Advent Calendar BBS Door %s\n", version)
		fmt.Printf("Build Date: %s\n", buildDate)
		fmt.Println("Programmed by J0hnny A1pha")
		os.Exit(0)
	}

	// Set log level - Default to ErrorLevel to hide info/debug messages from sysop console
	logrus.SetLevel(logrus.ErrorLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "15:04:05.000",
		FullTimestamp:   true,
	})

	logrus.WithField("elapsed", time.Since(startTime)).Info("STARTUP: Flags parsed")

	// Initialize components with embedded art filesystem
	logrus.WithField("elapsed", time.Since(startTime)).Info("STARTUP: Creating art manager")
	artManager := art.NewManager(embedded.ArtFS, "art")
	logrus.WithField("elapsed", time.Since(startTime)).Info("STARTUP: Creating navigator")
	navigator := navigation.NewNavigator(embedded.ArtFS, "art")
	logrus.WithField("elapsed", time.Since(startTime)).Info("STARTUP: Creating validator")
	validator := validation.NewValidator(embedded.ArtFS, "art")
	logrus.WithField("elapsed", time.Since(startTime)).Info("STARTUP: Components created")

	// Determine display mode
	// BBS mode: raw CP437 bytes (no conversion)
	// Local mode: CP437 to UTF-8 conversion for terminal display
	displayMode := display.ModeCP437Raw
	if *localMode {
		displayMode = display.ModeCP437
	}

	// Initialize components with hard-coded sensible defaults
	displayEngine := display.NewDisplayEngine(display.DisplayConfig{
		Mode:   displayMode,
		Width:  80, // Default, will be detected
		Height: 25, // Default, will be detected
		Theme:  "classic",
		Scrolling: display.ScrollingConfig{
			Enabled:           true,
			Indicators:        false,
			KeyboardShortcuts: true,
		},
		Columns: display.ColumnConfig{
			Handle80ColumnIssue: true,
			AutoDetectWidth:     true,
		},
		Performance: display.PerformanceConfig{
			CacheEnabled: true,
			CacheSizeMB:  50,
			PreloadLines: 100,
		},
	}, embedded.ArtFS)

	// Initialize BBS connection from door32.sys if provided
	var bbsConn *bbs.BBSConnection
	if *dropfilePath != "" {
		logrus.WithField("elapsed", time.Since(startTime)).Info("STARTUP: Creating BBS connection from door32.sys")
		var connErr error
		bbsConn, connErr = bbs.NewBBSConnection(*dropfilePath)
		logrus.WithField("elapsed", time.Since(startTime)).Info("STARTUP: BBS connection created")
		if connErr != nil {
			logrus.WithError(connErr).Error("Failed to create BBS connection - continuing in fallback mode")
			return
		}
		logrus.Info("BBS connection established - all I/O will go through inherited socket")
	}

	inputHandler := input.NewInputHandler()
	if bbsConn != nil {
		inputHandler.SetBBSConnection(bbsConn)
	}

	// Store BBS connection for later use by display components
	if bbsConn != nil {
		logrus.Info("BBS connection available - display output will be handled by modified display engine")
	} // Initialize session manager
	idleTimeout := 5 * time.Minute  // Hard-coded 5 minute idle timeout
	maxTimeout := 120 * time.Minute // Hard-coded 2 hour max timeout

	var sessionManager *session.Manager
	sessionManager = session.NewManager(idleTimeout, maxTimeout,
		func() {
			cleanup(displayEngine, inputHandler, sessionManager)
			os.Exit(0)
		},
		func() {
			cleanup(displayEngine, inputHandler, sessionManager)
			os.Exit(0)
		})

	// Get user information
	logrus.WithField("elapsed", time.Since(startTime)).Info("STARTUP: Getting user info")
	user := getUserInfo(*localMode)
	logrus.WithField("elapsed", time.Since(startTime)).Info("STARTUP: Got user info")

	// Detect terminal size (prefer BBS connection query over term.GetSize)
	logrus.WithField("elapsed", time.Since(startTime)).Info("STARTUP: Detecting terminal size")
	width, height := detectTerminalSize(bbsConn)
	logrus.WithField("elapsed", time.Since(startTime)).Info("STARTUP: Terminal size detected")

	// Update user struct with detected terminal size
	user.W = width
	user.H = height
	user.ModalW = width
	user.ModalH = height

	logrus.WithFields(logrus.Fields{
		"width":  width,
		"height": height,
	}).Info("Terminal size applied to user session")

	displayEngine = display.NewDisplayEngine(display.DisplayConfig{
		Mode:   displayMode,
		Width:  width,
		Height: height,
		Theme:  "classic",
		Scrolling: display.ScrollingConfig{
			Enabled:           true,
			Indicators:        false,
			KeyboardShortcuts: true,
		},
		Columns: display.ColumnConfig{
			Handle80ColumnIssue: true,
			AutoDetectWidth:     true,
		},
		Performance: display.PerformanceConfig{
			CacheEnabled: true,
			CacheSizeMB:  50,
			PreloadLines: 100,
		},
	}, embedded.ArtFS)

	// Configure BBS output (different behavior on Windows vs Linux)
	if bbsConn != nil {
		displayEngine.SetBBSConnection(bbsConn)
		logrus.Info("Display engine configured for BBS output")
	}

	// Validate terminal size
	if err := validator.ValidateTerminalSize(width, height); err != nil {
		logrus.WithError(err).Warn("Terminal size validation failed - continuing anyway")
	}

	// Validate ANSI emulation
	if err := validator.ValidateEmulation(user.Emulation); err != nil {
		logrus.WithError(err).Warn("ANSI emulation validation failed - continuing anyway")
	}

	// Get initial navigation state
	logrus.WithField("elapsed", time.Since(startTime)).Info("STARTUP: Getting initial state")
	initialState, err := navigator.GetInitialState()
	logrus.WithField("elapsed", time.Since(startTime)).Info("STARTUP: Got initial state")
	if err != nil {
		logrus.WithError(err).Error("Failed to get initial navigation state")
		return
	}

	// Apply debug overrides
	if *disableDate || *debugDate != "" {
		if *disableDate {
			logrus.Info("Date validation disabled by debug flag")
		}
		if *debugDate != "" {
			logrus.Info("Date validation skipped due to debug-date override")
		}
	} else {
		if err := validator.ValidateDate(); err != nil {
			displayNotYet(displayEngine, artManager, initialState.CurrentYear, user, inputHandler)
			return
		}
	}

	// Validate art files
	if err := validator.ValidateArtFiles(initialState.CurrentYear); err != nil {
		logrus.WithError(err).Error("Art file validation failed")
		return
	}

	// Apply date override if specified
	if *debugDate != "" {
		if err := applyDateOverride(&initialState, *debugDate); err != nil {
			logrus.WithError(err).Error("Failed to apply date override")
			return
		}
	}

	// Handle logon mode - skip welcome screen and go directly to current day's door
	if *logonMode {
		runLogonMode(displayEngine, artManager, inputHandler, sessionManager, initialState, user, validator)
		return
	}

	// Start session manager
	sessionManager.Start()
	defer sessionManager.Stop()

	// Open input handler
	logrus.WithField("elapsed", time.Since(startTime)).Info("STARTUP: About to open input handler")
	if err := inputHandler.Open(); err != nil {
		logrus.WithError(err).Error("Failed to open input handler")
		return
	}
	logrus.WithField("elapsed", time.Since(startTime)).Info("STARTUP: Input handler opened")
	defer inputHandler.Close()

	// Hide cursor, enable blink mode, and clear screen for both BBS and local modes
	logrus.WithField("elapsed", time.Since(startTime)).Info("STARTUP: Hiding cursor, enabling blink mode, and clearing screen")
	displayEngine.HideCursor()
	displayEngine.EnableBlinkMode() // Disable ICE mode to enable ANSI blink
	displayEngine.ClearScreen()
	logrus.WithField("elapsed", time.Since(startTime)).Info("STARTUP: Screen ready, entering main loop")
	defer func() {
		displayEngine.DisableBlinkMode() // Re-enable ICE mode on exit
		displayEngine.ShowCursor()       // Ensure cursor is shown on exit
	}()

	// Main application loop
	runMainLoop(displayEngine, artManager, navigator, inputHandler, sessionManager, initialState, user)
}

func getUserInfo(localMode bool) display.User {
	if localMode {
		logrus.Info("Running in local mode")
		return display.User{
			Alias:     "SysOp",
			TimeLeft:  120 * time.Minute,
			Emulation: 1,
			NodeNum:   1,
			H:         25,
			W:         80,
			ModalH:    25,
			ModalW:    80,
		}
	}

	// BBS mode - parse door32.sys if available
	logrus.Info("Running in BBS mode")

	if *dropfilePath != "" {
		door32Info, err := bbs.ParseDoor32(*dropfilePath)
		if err != nil {
			logrus.WithError(err).Warn("Failed to parse door32.sys, using defaults")
		} else {
			logrus.WithFields(logrus.Fields{
				"alias":     door32Info.Alias,
				"timeLeft":  door32Info.TimeLeft,
				"emulation": door32Info.Emulation,
				"node":      door32Info.NodeNumber,
			}).Info("Parsed user info from door32.sys")

			return display.User{
				Alias:     door32Info.Alias,
				TimeLeft:  time.Duration(door32Info.TimeLeft) * time.Minute,
				Emulation: door32Info.Emulation,
				NodeNum:   door32Info.NodeNumber,
				H:         25,
				W:         80,
				ModalH:    25,
				ModalW:    80,
			}
		}
	}

	// Fallback to default values
	return display.User{
		Alias:     "BBSUser",
		TimeLeft:  30 * time.Minute,
		Emulation: 1,
		NodeNum:   1,
		H:         25,
		W:         80,
		ModalH:    25,
		ModalW:    80,
	}
}

func detectTerminalSize(bbsConn *bbs.BBSConnection) (width, height int) {
	// Wrap in recovery to handle panics in terminal detection
	defer func() {
		if r := recover(); r != nil {
			logrus.WithField("panic", r).Error("Terminal size detection panicked - using fallback")
			width, height = 80, 25 // Standard fallback
		}
	}()

	logrus.Debug("Starting terminal size detection")

	// Try to detect actual terminal size for BBS connections
	if bbsConn != nil {
		logrus.Debug("BBS connection available, attempting terminal size detection")
		w, h, err := bbsConn.DetectTerminalSize()
		logrus.WithFields(logrus.Fields{
			"width":  w,
			"height": h,
			"error":  err,
		}).Debug("Terminal detection result")

		if err == nil && w > 0 && h > 0 {
			logrus.WithFields(logrus.Fields{
				"width":  w,
				"height": h,
				"method": "BBS terminal size detection",
			}).Info("Detected actual terminal size")
			return w, h
		} else {
			logrus.WithFields(logrus.Fields{
				"error":  err,
				"width":  w,
				"height": h,
			}).Warn("BBS terminal size detection failed, using standard 80x25")
			return 80, 25 // Fallback to standard BBS dimensions
		}
	}

	// Fallback: Try to get terminal size from stdin (works for local mode)
	logrus.Debug("Attempting fallback terminal size detection using term.GetSize")
	width, height, err := term.GetSize(int(os.Stdin.Fd()))
	logrus.WithFields(logrus.Fields{
		"width":  width,
		"height": height,
		"error":  err,
	}).Debug("term.GetSize result")

	if err == nil && width > 0 && height > 0 {
		logrus.WithFields(logrus.Fields{
			"width":  width,
			"height": height,
			"method": "term.GetSize (local)",
		}).Info("Detected terminal size")
		return width, height
	}

	// Final fallback to default 80x25
	logrus.WithError(err).Info("Could not detect terminal size, using default 80x25")
	return 80, 25
}

func applyDateOverride(state *navigation.State, dateStr string) error {
	// Parse date string in YYYY-MM-DD format
	parsedDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return fmt.Errorf("invalid date format (expected YYYY-MM-DD): %w", err)
	}

	// Extract the day from the date
	day := parsedDate.Day()

	// Validate day is within advent calendar range (1-25)
	if day < 1 || day > 25 {
		return fmt.Errorf("day %d is out of advent calendar range (1-25)", day)
	}

	// Update state to simulate it being this date
	// CurrentDay starts at 1, but MaxDay is set to the debug day
	// This allows navigation from day 1 up to (and including) the debug day
	state.CurrentDay = 1
	state.MaxDay = day

	logrus.WithFields(logrus.Fields{
		"debugDate":  dateStr,
		"currentDay": state.CurrentDay,
		"maxDay":     state.MaxDay,
	}).Info("Applied debug date override - simulating current date")

	return nil
}

func displayNotYet(displayEngine *display.DisplayEngine, artManager *art.Manager, year int, user display.User, inputHandler *input.InputHandler) {
	// Display "not yet" screen
	notYetPath := artManager.GetPath(year, 0, "notyet")
	if notYetPath != "" {
		displayEngine.Display(notYetPath, user)
	}

	// Wait for key press or 10 seconds, whichever comes first (no visible prompt)
	if inputHandler != nil {
		// Open input handler temporarily if not already open
		wasOpen := true
		if err := inputHandler.Open(); err != nil {
			// If we can't open input handler, fall back to 10 second wait
			time.Sleep(10 * time.Second)
			return
		}
		defer func() {
			if !wasOpen {
				inputHandler.Close()
			}
		}()

		// Create a channel for key press
		keyPressed := make(chan bool, 1)

		// Start goroutine to wait for key press
		go func() {
			inputHandler.ReadKey()
			keyPressed <- true
		}()

		// Wait for either key press or timeout
		select {
		case <-keyPressed:
			logrus.Info("NOTYET: key pressed, exiting")
		case <-time.After(10 * time.Second):
			logrus.Info("NOTYET: timeout reached, exiting")
		}
	} else {
		// Fallback: 10 second pause
		time.Sleep(10 * time.Second)
	}
}

func runMainLoop(displayEngine *display.DisplayEngine, artManager *art.Manager,
	navigator *navigation.Navigator, inputHandler *input.InputHandler,
	sessionManager *session.Manager, state navigation.State, user display.User) {

	loopStart := time.Now()
	currentState := state
	var currentArtPath string

	// For scrolling screens
	var infoLoaded, membersLoaded bool
	var infoLines, membersLines []string
	var infoScrollPos, membersScrollPos int

	logrus.WithField("elapsed", time.Since(loopStart)).Info("MAINLOOP: Starting first iteration")

	for {
		// Display current screen only if the art path changed
		var artPath string
		switch currentState.Screen {
		case navigation.ScreenWelcome:
			logrus.WithField("elapsed", time.Since(loopStart)).Info("MAINLOOP: Getting welcome art path")
			artPath = artManager.GetPath(currentState.CurrentYear, 0, "welcome")
			logrus.WithField("elapsed", time.Since(loopStart)).WithField("path", artPath).Info("MAINLOOP: Got welcome art path")
		case navigation.ScreenDay:
			artPath = artManager.GetPath(currentState.CurrentYear, currentState.CurrentDay, "day")
		case navigation.ScreenComeback:
			artPath = artManager.GetPath(currentState.CurrentYear, 0, "comeback")
		case navigation.ScreenInfo:
			artPath = artManager.GetPath(currentState.CurrentYear, 0, "info")
		case navigation.ScreenMembers:
			artPath = artManager.GetPath(currentState.CurrentYear, 0, "members")
		}

		// Only display if art path changed
		if artPath != "" && artPath != currentArtPath {
			logrus.WithField("elapsed", time.Since(loopStart)).WithField("path", artPath).Info("MAINLOOP: About to display art")
			logrus.WithFields(logrus.Fields{
				"artPath":        artPath,
				"currentArtPath": currentArtPath,
				"screen":         currentState.Screen,
				"day":            currentState.CurrentDay,
			}).Debug("Displaying art")

			switch currentState.Screen {
			case navigation.ScreenInfo:
				if !infoLoaded {
					lines, err := displayEngine.LoadAnsiLines(artPath)
					if err == nil {
						infoLines = lines
						infoLoaded = true
						infoScrollPos = 0 // Always start at top
					} else {
						infoLines = []string{fmt.Sprintf("[Unable to load %s]", artPath)}
						infoLoaded = true
						infoScrollPos = 0
					}
				}
				// Always use scrolling logic for INFOFILE.ANS
				displayEngine.SetScrollState(infoScrollPos, len(infoLines))
				displayEngine.RenderScrollable(infoLines, infoScrollPos)
				currentArtPath = artPath
			case navigation.ScreenMembers:
				if !membersLoaded {
					lines, err := displayEngine.LoadAnsiLines(artPath)
					if err == nil {
						membersLines = lines
						membersLoaded = true
						membersScrollPos = 0 // Always start at top
					} else {
						membersLines = []string{fmt.Sprintf("[Unable to load %s]", artPath)}
						membersLoaded = true
						membersScrollPos = 0
					}
				}
				// Always use scrolling logic for MEMBERS.ANS
				displayEngine.SetScrollState(membersScrollPos, len(membersLines))
				displayEngine.RenderScrollable(membersLines, membersScrollPos)
				currentArtPath = artPath
			default:
				logrus.WithField("elapsed", time.Since(loopStart)).Info("MAINLOOP: Calling displayEngine.Display()")
				if err := displayEngine.Display(artPath, user); err != nil {
					logrus.WithError(err).Error("Failed to display art")
				}
				logrus.WithField("elapsed", time.Since(loopStart)).Info("MAINLOOP: displayEngine.Display() returned")
				currentArtPath = artPath
			}
		}

		// Get user input
		// CRITICAL: Give Mystic BBS time to transmit buffered output to user's terminal before blocking on input
		// Windows 7 + Mystic BBS needs significant time to flush output buffers
		logrus.WithField("elapsed", time.Since(loopStart)).Info("MAINLOOP: Waiting 500ms before reading input (BBS output stabilization)")
		time.Sleep(500 * time.Millisecond)

		logrus.WithField("elapsed", time.Since(loopStart)).Info("MAINLOOP: Now reading user input")
		char, key, err := inputHandler.ReadKey()
		logrus.WithField("elapsed", time.Since(loopStart)).Info("MAINLOOP: Got user input")
		if err != nil {
			logrus.WithError(err).Error("Failed to read input")
			continue
		}

		// Handle scrolling for Info/Members screens
		// Reserve last row for menu bar (user.H - 1 is usable height)
		if currentState.Screen == navigation.ScreenInfo {
			// Get the scroll state to determine visible lines
			scrollState := displayEngine.GetScrollState()
			visibleLines := scrollState.VisibleLines

			if key == input.KeyArrowUp && infoScrollPos > 0 {
				infoScrollPos--
				logrus.WithField("infoScrollPos", infoScrollPos).Debug("Scrolling info up")
				displayEngine.RenderScrollableContentOnly(infoLines, infoScrollPos)
				continue
			} else if key == input.KeyArrowDown && infoScrollPos < len(infoLines)-visibleLines {
				infoScrollPos++
				logrus.WithField("infoScrollPos", infoScrollPos).Debug("Scrolling info down")
				displayEngine.RenderScrollableContentOnly(infoLines, infoScrollPos)
				continue
			} else if key == input.KeyArrowUp || key == input.KeyArrowDown {
				logrus.WithFields(logrus.Fields{
					"key":          key,
					"scrollPos":    infoScrollPos,
					"totalLines":   len(infoLines),
					"visibleLines": visibleLines,
					"maxScrollPos": len(infoLines) - visibleLines,
				}).Debug("Arrow key pressed but bounds check failed")
			}
		} else if currentState.Screen == navigation.ScreenMembers {
			// Get the scroll state to determine visible lines
			scrollState := displayEngine.GetScrollState()
			visibleLines := scrollState.VisibleLines

			if key == input.KeyArrowUp && membersScrollPos > 0 {
				membersScrollPos--
				logrus.WithField("membersScrollPos", membersScrollPos).Debug("Scrolling members up")
				displayEngine.RenderScrollableContentOnly(membersLines, membersScrollPos)
				continue
			} else if key == input.KeyArrowDown && membersScrollPos < len(membersLines)-visibleLines {
				membersScrollPos++
				logrus.WithField("membersScrollPos", membersScrollPos).Debug("Scrolling members down")
				displayEngine.RenderScrollableContentOnly(membersLines, membersScrollPos)
				continue
			} else if key == input.KeyArrowUp || key == input.KeyArrowDown {
				logrus.WithFields(logrus.Fields{
					"key":          key,
					"scrollPos":    membersScrollPos,
					"totalLines":   len(membersLines),
					"visibleLines": visibleLines,
					"maxScrollPos": len(membersLines) - visibleLines,
				}).Debug("Arrow key pressed but bounds check failed")
			}
		}

		// Reset idle timer
		sessionManager.ResetIdleTimer()

		// Handle quit/back navigation
		if char == 'q' || char == 'Q' || key == input.KeyEsc {
			// Get the latest year from available years
			latestYear := currentState.AvailableYears[len(currentState.AvailableYears)-1]

			// Exit on Q/ESC from WELCOME or COMEBACK in the latest year
			if (currentState.Screen == navigation.ScreenWelcome || currentState.Screen == navigation.ScreenComeback) && currentState.CurrentYear == latestYear {
				// Only exit if we're on the welcome screen of the latest year (2025)
				logrus.Info("User requested exit from latest year's welcome screen")
				exitPath := artManager.GetPath(currentState.CurrentYear, 0, "goodbye")
				if exitPath != "" {
					displayEngine.Display(exitPath, user)

					// Wait for key press or 10 seconds, whichever comes first (no visible prompt)
					keyPressed := make(chan bool, 1)

					// Start goroutine to wait for key press
					go func() {
						inputHandler.ReadKey()
						keyPressed <- true
					}()

					// Wait for either key press or timeout
					select {
					case <-keyPressed:
						logrus.Info("GOODBYE: key pressed, exiting")
					case <-time.After(10 * time.Second):
						logrus.Info("GOODBYE: timeout reached, exiting")
					}
				}
				break
			} else if currentState.Screen == navigation.ScreenInfo || currentState.Screen == navigation.ScreenMembers {
				// Return to welcome screen from Info/Members
				currentState.Screen = navigation.ScreenWelcome
				infoScrollPos = 0
				membersScrollPos = 0
				// Reset loaded flags to ensure proper reloading next time
				infoLoaded = false
				membersLoaded = false
				continue
			} else {
				// Go back to welcome screen of the latest year
				logrus.Info("User requested return to welcome screen")

				// Get the latest year from available years
				latestYear := currentState.AvailableYears[len(currentState.AvailableYears)-1]

				// Only change the year if we're not already in the latest year
				if currentState.CurrentYear != latestYear {
					logrus.WithFields(logrus.Fields{
						"previousYear": currentState.CurrentYear,
						"latestYear":   latestYear,
					}).Info("Returning to latest year's welcome screen")
					currentState.CurrentYear = latestYear
				}

				currentState.Screen = navigation.ScreenWelcome
				continue
			}
		}

		// Handle year selection from welcome/comeback screen with numeric keys
		if (currentState.Screen == navigation.ScreenWelcome || currentState.Screen == navigation.ScreenComeback) && char >= '1' && char <= '9' {
			yearIndex := int(char - '0')
			newState, newArtPath, err := navigator.SelectYearByIndex(yearIndex, currentState)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"index": yearIndex,
					"char":  string(char),
				}).Debug("Invalid year selection")
				// Invalid selection, just ignore
			} else {
				logrus.WithFields(logrus.Fields{
					"index":        yearIndex,
					"selectedYear": newState.CurrentYear,
					"artPath":      newArtPath,
				}).Info("Year selected")
				currentState = newState
			}
			continue
		}

		// Handle Info/Members menu keys from welcome/comeback screen
		if (currentState.Screen == navigation.ScreenWelcome || currentState.Screen == navigation.ScreenComeback) && (char == 'i' || char == 'I') {
			currentState.Screen = navigation.ScreenInfo
			infoScrollPos = 0
			// Force reload of INFOFILE.ANS to ensure proper handling
			infoLoaded = false
			continue
		}
		if (currentState.Screen == navigation.ScreenWelcome || currentState.Screen == navigation.ScreenComeback) && (char == 'm' || char == 'M') {
			currentState.Screen = navigation.ScreenMembers
			membersScrollPos = 0
			// Force reload of MEMBERS.ANS to ensure proper handling
			membersLoaded = false
			continue
		}

		// Handle scrolling keys first (if content is scrollable)
		scrollHandled := false
		if currentState.Screen == navigation.ScreenDay {
			scrollState := displayEngine.GetScrollState()
			switch key {
			case input.KeyArrowUp:
				if scrollState.CanScrollUp {
					displayEngine.ScrollUp()
					scrollHandled = true
				}
			case input.KeyArrowDown:
				if scrollState.CanScrollDown {
					displayEngine.ScrollDown()
					scrollHandled = true
				}
			}
		}

		// If scroll was handled, skip navigation
		if scrollHandled {
			continue
		}

		// Handle navigation
		var direction navigation.Direction

		// Handle RETURN key on WELCOME/COMEBACK screen to navigate to current day (same as arrow right)
		if (currentState.Screen == navigation.ScreenWelcome || currentState.Screen == navigation.ScreenComeback) && (char == '\r' || char == '\n') {
			direction = navigation.DirRight
			logrus.Info("RETURN pressed on WELCOME - navigating to current day")
		}

		switch key {
		case input.KeyArrowRight:
			direction = navigation.DirRight
			logrus.WithField("direction", "right").Debug("Right arrow pressed")
		case input.KeyArrowLeft:
			direction = navigation.DirLeft
			logrus.WithField("direction", "left").Debug("Left arrow pressed")
		case input.KeyPageUp:
			direction = navigation.DirPageUp
		case input.KeyPageDown:
			direction = navigation.DirPageDown
		case input.KeyHome:
			direction = navigation.DirHome
		case input.KeyEnd:
			direction = navigation.DirEnd
		}

		if direction != 0 {
			logrus.WithFields(logrus.Fields{
				"direction":     direction,
				"currentDay":    currentState.CurrentDay,
				"currentScreen": currentState.Screen,
			}).Debug("Attempting navigation")

			newState, newArtPath, err := navigator.Navigate(direction, currentState)
			if err != nil {
				logrus.WithError(err).Error("Navigation error")
				continue
			}

			logrus.WithFields(logrus.Fields{
				"newDay":    newState.CurrentDay,
				"newScreen": newState.Screen,
				"artPath":   newArtPath,
			}).Debug("Navigation result")

			currentState = newState
			if newArtPath != "" && newArtPath != artPath {
				// Art path changed, will be displayed in next iteration
			}
		}
	}

	cleanup(displayEngine, inputHandler, sessionManager)
}

func runLogonMode(displayEngine *display.DisplayEngine, artManager *art.Manager, inputHandler *input.InputHandler, sessionManager *session.Manager, state navigation.State, user display.User, validator *validation.Validator) {

	// Check if it's December (unless date validation is disabled)
	if !*disableDate {
		if err := validator.ValidateDate(); err != nil {
			// Not December - show NOTYET.ANS with 10-second timeout
			notYetPath := artManager.GetPath(state.CurrentYear, 0, "notyet")
			if notYetPath != "" {
				displayEngine.Display(notYetPath, user)
			}

			// Wait for key press or 10 seconds, whichever comes first (no visible prompt)
			if inputHandler != nil {
				// Open input handler temporarily if not already open
				if err := inputHandler.Open(); err == nil {
					defer inputHandler.Close()

					// Create a channel for key press
					keyPressed := make(chan bool, 1)

					// Start goroutine to wait for key press
					go func() {
						inputHandler.ReadKey()
						keyPressed <- true
					}()

					// Wait for either key press or timeout
					select {
					case <-keyPressed:
						logrus.Info("NOTYET: key pressed, exiting")
					case <-time.After(10 * time.Second):
						logrus.Info("NOTYET: timeout reached, exiting")
					}
				}
			}
			return
		}
	}

	// It's December - get current day
	now := time.Now()
	currentDay := now.Day()
	if currentDay > 25 {
		currentDay = 25
	}

	// Start session manager
	sessionManager.Start()
	defer sessionManager.Stop()

	// Open input handler
	if err := inputHandler.Open(); err != nil {
		logrus.WithError(err).Error("Failed to open input handler in logon mode")
		return
	}
	defer inputHandler.Close()

	// Hide cursor, enable blink mode, and clear screen
	displayEngine.HideCursor()
	displayEngine.EnableBlinkMode() // Disable ICE mode to enable ANSI blink
	displayEngine.ClearScreen()
	defer func() {
		displayEngine.DisableBlinkMode() // Re-enable ICE mode on exit
		displayEngine.ShowCursor()
	}()

	// Display current day's door art
	dayArtPath := artManager.GetPath(state.CurrentYear, currentDay, "day")
	if dayArtPath != "" {
		if err := displayEngine.Display(dayArtPath, user); err != nil {
			logrus.WithError(err).Error("Failed to display day art in logon mode")
		}
	}

	// Wait for any key press
	logrus.Info("Logon mode: waiting for key press on day art")
	_, _, err := inputHandler.ReadKey()
	if err != nil {
		logrus.WithError(err).Error("Failed to read key in logon mode")
	}

	// Clear screen
	displayEngine.ClearScreen()

	// Display COMEBACK.ANS
	comebackPath := artManager.GetPath(state.CurrentYear, 0, "comeback")
	if comebackPath != "" {
		if err := displayEngine.Display(comebackPath, user); err != nil {
			logrus.WithError(err).Error("Failed to display comeback art in logon mode")
		}
	}

	// Wait for key press or 10 seconds, whichever comes first
	logrus.Info("Logon mode: waiting for key press or 10 seconds on COMEBACK.ANS")

	// Create a channel for key press
	keyPressed := make(chan bool, 1)

	// Start goroutine to wait for key press
	go func() {
		inputHandler.ReadKey()
		keyPressed <- true
	}()

	// Wait for either key press or timeout
	select {
	case <-keyPressed:
		logrus.Info("Logon mode: key pressed, exiting")
	case <-time.After(10 * time.Second):
		logrus.Info("Logon mode: timeout reached, exiting")
	}

	// Clean up
	displayEngine.DisableBlinkMode() // Re-enable ICE mode
	displayEngine.ShowCursor()
	displayEngine.ClearScreen()
	fmt.Print("\033[0m") // Reset colors
}

func cleanup(displayEngine *display.DisplayEngine, inputHandler *input.InputHandler, sessionManager *session.Manager) {
	sessionManager.Stop()
	inputHandler.Close()
	displayEngine.DisableBlinkMode() // Re-enable ICE mode
	displayEngine.ShowCursor()
	displayEngine.ClearScreen() // ClearScreen already flushes
	fmt.Print("\033[0m")        // Reset colors
}
