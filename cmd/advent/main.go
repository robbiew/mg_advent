package main

import (
	"flag"
	"fmt"
	"io"
	"os"
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

var (
	// Command line flags
	localMode    = flag.Bool("local", false, "run in local UTF-8 mode")
	socketHost   = flag.String("socket-host", "127.0.0.1", "BBS server IP address")
	debugDate    = flag.String("debug-date", "", "override date (YYYY-MM-DD)")
	disableDate  = flag.Bool("debug-disable-date", false, "disable date validation")
	disableArt   = flag.Bool("debug-disable-art", false, "disable art validation")
	dropfilePath = flag.String("path", "", "path to door32.sys file")
	debugMode    = flag.Bool("debug", false, "enable debug logging")
)

func main() {
	flag.Parse()

	// Setup logging - only show errors by default to keep BBS output clean
	// Set to InfoLevel or DebugLevel for troubleshooting
	if *debugMode {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.ErrorLevel)
	}
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true, // Cleaner output for BBS
	})

	// Initialize components with embedded art filesystem
	artManager := art.NewManager(embedded.ArtFS, "art")
	navigator := navigation.NewNavigator(embedded.ArtFS, "art")
	validator := validation.NewValidator(embedded.ArtFS, "art")

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

	// Create BBS connection - this is REQUIRED for Windows BBS doors
	// All output must go through the inherited socket handle, not stdout
	var bbsConn *bbs.BBSConnection
	if *dropfilePath != "" {
		var connErr error
		bbsConn, connErr = bbs.NewBBSConnection(*dropfilePath, *socketHost)
		if connErr != nil {
			logrus.WithError(connErr).Error("Failed to create BBS connection - continuing in fallback mode")
			fmt.Println("ERROR: Unable to establish BBS connection.")
			fmt.Println("This door requires proper BBS integration to function.")
			fmt.Printf("Error: %v\n", connErr)
			fmt.Println("\nPress any key to exit...")
			fmt.Scanln()
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
			fmt.Println("\nIdle timeout reached... exiting.")
			cleanup(displayEngine, inputHandler, sessionManager)
			os.Exit(0)
		},
		func() {
			fmt.Println("\nMaximum session time reached... exiting.")
			cleanup(displayEngine, inputHandler, sessionManager)
			os.Exit(0)
		})

	// Get user information
	user := getUserInfo(*localMode)

	// Detect terminal size (prefer BBS connection query over term.GetSize)
	width, height := detectTerminalSize(bbsConn)

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
		fmt.Printf("WARNING: Terminal size %dx%d may not be optimal\n", width, height)
	}

	// Validate ANSI emulation
	if err := validator.ValidateEmulation(user.Emulation); err != nil {
		logrus.WithError(err).Warn("ANSI emulation validation failed - continuing anyway")
		fmt.Printf("WARNING: Terminal emulation type %d may not be fully supported\n", user.Emulation)
	}

	// Get initial navigation state
	initialState, err := navigator.GetInitialState()
	if err != nil {
		logrus.WithError(err).Error("Failed to get initial navigation state")
		fmt.Println("ERROR: Unable to initialize door navigation.")
		fmt.Printf("Error: %v\n", err)
		fmt.Println("\nPress any key to exit...")
		fmt.Scanln()
		return
	}

	// Apply debug overrides
	if *disableDate {
		logrus.Info("Date validation disabled by debug flag")
	} else {
		if err := validator.ValidateDate(); err != nil {
			displayNotYet(displayEngine, artManager, initialState.CurrentYear, user, inputHandler, bbsConn)
			return
		}
	}

	// Validate art files
	if !*disableArt {
		if err := validator.ValidateArtFiles(initialState.CurrentYear); err != nil {
			logrus.WithError(err).Error("Art file validation failed")
			fmt.Println("ERROR: Required art files are missing.")
			fmt.Printf("Error: %v\n", err)
			fmt.Println("The door cannot function without proper art files.")
			fmt.Println("\nPress any key to exit...")
			fmt.Scanln()
			return
		}
	}

	// Apply date override if specified
	if *debugDate != "" {
		if err := applyDateOverride(&initialState, *debugDate); err != nil {
			logrus.WithError(err).Error("Failed to apply date override")
			fmt.Printf("ERROR: Invalid debug date specified: %s\n", *debugDate)
			fmt.Printf("Error: %v\n", err)
			fmt.Println("\nPress any key to exit...")
			fmt.Scanln()
			return
		}
	}

	// Start session manager
	sessionManager.Start()
	defer sessionManager.Stop()

	// Open input handler
	if err := inputHandler.Open(); err != nil {
		logrus.WithError(err).Error("Failed to open input handler")
		fmt.Println("ERROR: Unable to initialize user input system.")
		fmt.Printf("Error: %v\n", err)
		fmt.Println("\nPress any key to exit...")
		fmt.Scanln()
		return
	}
	defer inputHandler.Close()

	// Hide cursor and clear screen for both BBS and local modes
	displayEngine.HideCursor()
	displayEngine.ClearScreen()
	defer displayEngine.ShowCursor() // Ensure cursor is shown on exit

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
		door32Info, err := bbs.ParseDoor32(*dropfilePath, *socketHost)
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

func applyDateOverride(_ *navigation.State, _ string) error {
	// Parse and apply debug date override
	// Implementation would update state.CurrentDay and state.MaxDay
	// Currently a stub - not implemented
	return nil
}

func displayNotYet(displayEngine *display.DisplayEngine, artManager *art.Manager, year int, user display.User, inputHandler *input.InputHandler, bbsConn *bbs.BBSConnection) {
	// Display "not yet" screen
	notYetPath := artManager.GetPath(year, 0, "notyet")
	if notYetPath != "" {
		displayEngine.Display(notYetPath, user)
	}

	// Add "[Press a Key]" prompt at bottom of screen
	var writer io.Writer
	if bbsConn != nil {
		writer = bbsConn
	} else {
		writer = os.Stdout
	}

	// Position cursor at bottom of screen and display prompt
	fmt.Fprintf(writer, "\033[%d;1H\033[2K\033[37;1m[Press a Key]\033[0m", user.H) // Flush output if it's a buffered writer
	if flusher, ok := writer.(interface{ Flush() error }); ok {
		flusher.Flush()
	}

	// Wait for any key press
	if inputHandler != nil {
		// Open input handler temporarily if not already open
		wasOpen := true
		if err := inputHandler.Open(); err != nil {
			// If we can't open input handler, fall back to simple wait
			time.Sleep(3 * time.Second)
			return
		}
		defer func() {
			if !wasOpen {
				inputHandler.Close()
			}
		}()

		// Wait for any key press
		inputHandler.ReadKey()
	} else {
		// Fallback: simple pause
		time.Sleep(3 * time.Second)
	}
}

func runMainLoop(displayEngine *display.DisplayEngine, artManager *art.Manager,
	navigator *navigation.Navigator, inputHandler *input.InputHandler,
	sessionManager *session.Manager, state navigation.State, user display.User) {

	currentState := state
	var currentArtPath string

	for {
		// Display current screen only if the art path changed
		var artPath string
		switch currentState.Screen {
		case navigation.ScreenWelcome:
			artPath = artManager.GetPath(currentState.CurrentYear, 0, "welcome")
		case navigation.ScreenDay:
			artPath = artManager.GetPath(currentState.CurrentYear, currentState.CurrentDay, "day")
		case navigation.ScreenComeback:
			artPath = artManager.GetPath(currentState.CurrentYear, 0, "comeback")
		}

		// Only display if art path changed
		if artPath != "" && artPath != currentArtPath {
			logrus.WithFields(logrus.Fields{
				"artPath":        artPath,
				"currentArtPath": currentArtPath,
				"screen":         currentState.Screen,
				"day":            currentState.CurrentDay,
			}).Debug("Displaying art")

			if err := displayEngine.Display(artPath, user); err != nil {
				logrus.WithError(err).Error("Failed to display art")
			}
			currentArtPath = artPath
		}

		// Get user input
		char, key, err := inputHandler.ReadKey()
		if err != nil {
			logrus.WithError(err).Error("Failed to read input")
			continue
		}

		// Reset idle timer
		sessionManager.ResetIdleTimer()

		// Handle quit/back navigation
		if char == 'q' || char == 'Q' || key == input.KeyEsc {
			if currentState.Screen == navigation.ScreenWelcome {
				// Already on welcome screen, exit application
				logrus.Info("User requested exit from welcome screen")
				exitPath := artManager.GetPath(currentState.CurrentYear, 0, "goodbye")
				if exitPath != "" {
					displayEngine.Display(exitPath, user)
					time.Sleep(2 * time.Second)
				}
				break
			} else {
				// Go back to welcome screen
				logrus.Info("User requested return to welcome screen")
				currentState.Screen = navigation.ScreenWelcome
				// Art will be displayed in next iteration
				continue
			}
		}

		// Handle year selection from welcome screen with numeric keys
		if currentState.Screen == navigation.ScreenWelcome && char >= '1' && char <= '9' {
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
				// Art will be displayed in next iteration
			}
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

		// Handle RETURN key on WELCOME screen to navigate to current day (same as arrow right)
		if currentState.Screen == navigation.ScreenWelcome && (char == '\r' || char == '\n') {
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

func cleanup(displayEngine *display.DisplayEngine, inputHandler *input.InputHandler, sessionManager *session.Manager) {
	sessionManager.Stop()
	inputHandler.Close()
	displayEngine.ShowCursor()
	displayEngine.ClearScreen() // ClearScreen already flushes
	fmt.Print("\033[0m")        // Reset colors
}
