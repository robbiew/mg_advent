package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/term"

	"github.com/robbiew/advent/internal/art"
	"github.com/robbiew/advent/internal/bbs"
	"github.com/robbiew/advent/internal/display"
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
)

func main() {
	flag.Parse()

	// Setup logging (simplified)
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetFormatter(&logrus.TextFormatter{})

	// Determine display mode
	displayMode := display.ModeCP437
	if *localMode {
		displayMode = display.ModeUTF8
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
	})

	artManager := art.NewManager("art")
	navigator := navigation.NewNavigator("art")
	validator := validation.NewValidator("art")
	// Create BBS connection - this is REQUIRED for Windows BBS doors
	// All output must go through the inherited socket handle, not stdout
	var bbsConn *bbs.BBSConnection
	if *dropfilePath != "" {
		var connErr error
		bbsConn, connErr = bbs.NewBBSConnection(*dropfilePath, *socketHost)
		if connErr != nil {
			logrus.WithError(connErr).Fatal("Failed to create BBS connection - door cannot function without it")
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

	// Detect terminal size and update display engine
	width, height := detectTerminalSize()
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
	})

	// Configure dual output (OpenDoors pattern: console + BBS connection)
	if bbsConn != nil {
		displayEngine.SetBBSConnection(bbsConn)
		displayEngine.SetUser(user) // Set user info for sysop status bar
		logrus.Info("Display engine configured for dual output (sysop console + user BBS terminal)")
	}

	// Validate terminal size
	if err := validator.ValidateTerminalSize(width, height); err != nil {
		logrus.WithError(err).Fatal("Terminal size validation failed")
	}

	// Validate ANSI emulation
	if err := validator.ValidateEmulation(user.Emulation); err != nil {
		logrus.WithError(err).Fatal("ANSI emulation validation failed")
	}

	// Get initial navigation state
	initialState, err := navigator.GetInitialState()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to get initial navigation state")
	}

	// Apply debug overrides
	if *disableDate {
		logrus.Info("Date validation disabled by debug flag")
	} else {
		if err := validator.ValidateDate(); err != nil {
			displayNotYet(displayEngine, artManager, initialState.CurrentYear)
			return
		}
	}

	// Validate art files
	if !*disableArt {
		if err := validator.ValidateArtFiles(initialState.CurrentYear); err != nil {
			logrus.WithError(err).Fatal("Art file validation failed")
		}
	}

	// Apply date override if specified
	if *debugDate != "" {
		if err := applyDateOverride(&initialState, *debugDate); err != nil {
			logrus.WithError(err).Fatal("Failed to apply date override")
		}
	}

	// Start session manager
	sessionManager.Start()
	defer sessionManager.Stop()

	// Open input handler
	if err := inputHandler.Open(); err != nil {
		logrus.WithError(err).Fatal("Failed to open input handler")
	}
	defer inputHandler.Close()

	// Hide cursor and clear screen
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

func detectTerminalSize() (width, height int) {
	// Try to get terminal size from stdin
	if width, height, err := term.GetSize(int(os.Stdin.Fd())); err == nil {
		logrus.WithFields(logrus.Fields{
			"width":  width,
			"height": height,
		}).Debug("Detected terminal size")
		return width, height
	}

	// Fallback to default 80x25
	logrus.Debug("Could not detect terminal size, using default 80x25")
	return 80, 25
}

func applyDateOverride(state *navigation.State, dateStr string) error {
	// Parse and apply debug date override
	// Implementation would update state.CurrentDay and state.MaxDay
	logrus.WithField("date", dateStr).Info("Applied date override")
	return nil
}

func displayNotYet(displayEngine *display.DisplayEngine, artManager *art.Manager, year int) {
	// Display "not yet" screen
	notYetPath := artManager.GetPath(year, 0, "notyet")
	if notYetPath != "" {
		displayEngine.Display(notYetPath, display.User{})
	}
	time.Sleep(3 * time.Second)
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

			// Check if art file exists, fallback to MISSING.ANS if not found
			finalArtPath := artPath
			overlayText := ""

			if _, err := os.Stat(artPath); os.IsNotExist(err) {
				logrus.WithField("missingPath", artPath).Warn("Art file not found, using MISSING.ANS")

				// Extract just the filename for display
				missingFileName := filepath.Base(artPath)
				overlayText = missingFileName

				finalArtPath = artManager.GetPath(currentState.CurrentYear, 0, "missing")

				// If MISSING.ANS also doesn't exist, log error
				if _, err := os.Stat(finalArtPath); os.IsNotExist(err) {
					logrus.WithField("missingPath", finalArtPath).Error("MISSING.ANS not found")
					continue
				}
			}

			if err := displayEngine.DisplayWithOverlay(finalArtPath, user, overlayText); err != nil {
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
				welcomePath := artManager.GetPath(currentState.CurrentYear, 0, "welcome")
				if welcomePath != "" {
					artPath = welcomePath
				}
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
				if newArtPath != "" {
					artPath = newArtPath // Update artPath to display new art
				}
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
	displayEngine.ClearScreen()
	fmt.Print("\033[0m") // Reset colors
}
