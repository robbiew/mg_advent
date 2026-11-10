package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	gd "github.com/robbiew/godoors"
	"github.com/sirupsen/logrus"

	"github.com/robbiew/advent/internal/art"
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
	debugDate    = flag.String("debug-date", "", "pretend today is this date (YYYY-MM-DD)")
	disableDate  = flag.Bool("debug-disable-date", false, "unlock every installed year and day")
	dropfilePath = flag.String("path", "", "door32.sys file, or the directory containing it")
	logonMode    = flag.Bool("logon", false, "logon mode: show today's door during the December season, then exit")
	showVersion  = flag.Bool("version", false, "show version information")
	noIce        = flag.Bool("noice", false, "disable ICE mode control codes (for terminals that don't support them)")
	noDetect     = flag.Bool("nodetect", false, "disable terminal size detection (use default 80x25)")
)

const (
	idleTimeout    = 5 * time.Minute   // no key for this long ends the session
	maxSessionTime = 120 * time.Minute // upper bound, lowered by the caller's BBS time left
	logonWait      = 30 * time.Second  // logon mode: how long today's door stays up without a key
	exitScreenWait = 10 * time.Second  // goodbye / not-yet screens
)

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("MiSTiGRiS Advent Calendar BBS Door %s\n", version)
		fmt.Printf("Build Date: %s\n", buildDate)
		fmt.Println("Programmed by J0hnny A1pha")
		os.Exit(0)
	}

	// Default to ErrorLevel to hide info/debug messages from the sysop console.
	logrus.SetLevel(logrus.ErrorLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "15:04:05.000",
		FullTimestamp:   true,
	})

	os.Exit(run())
}

// run is the door's body. It returns the process exit code; deferred cleanup
// (terminal restore, closing the BBS session) runs before main exits.
func run() int {
	artManager := art.NewManager(embedded.ArtFS, "art")
	navigator := navigation.NewNavigator(embedded.ArtFS, "art")
	validator := validation.NewValidator(embedded.ArtFS, "art")
	if err := configureCalendar(navigator, *debugDate, *disableDate); err != nil {
		logrus.WithError(err).Error("Invalid date options")
		return 1
	}

	// Logon mode only has something to show while a season is running. Bail
	// out before touching the terminal so off-season logons stay silent.
	if *logonMode {
		if _, live := navigator.LiveYear(); !live {
			logrus.Info("Logon mode: no advent season running today; nothing to show")
			return 0
		}
	}

	// Select the transport before any terminal probe or input read.
	var doorInfo *gd.DropFile
	if *dropfilePath != "" {
		info, err := openDoor(*dropfilePath)
		if err != nil {
			logrus.WithError(err).Error("Failed to open BBS connection")
			return 1
		}
		doorInfo = &info
	}
	defer gd.CloseSession()

	inputHandler := input.NewInputHandler()
	if err := inputHandler.Open(); err != nil {
		logrus.WithError(err).Error("Failed to open input handler")
		return 1
	}
	defer inputHandler.Close()

	user := getUserInfo(*localMode, doorInfo)
	width, height := detectTerminalSize(doorInfo != nil, *noDetect)
	user.W, user.H = width, height
	user.ModalW, user.ModalH = width, height

	displayEngine := newDisplayEngine(width, height)
	if doorInfo != nil {
		displayEngine.SetBBSConnection(gd.Output())
	}

	if err := validator.ValidateTerminalSize(width, height); err != nil {
		logrus.WithError(err).Warn("Terminal size validation failed - continuing anyway")
	}
	if err := validator.ValidateEmulation(user.Emulation); err != nil {
		logrus.WithError(err).Warn("ANSI emulation validation failed - continuing anyway")
	}

	sess := session.NewManager(idleTimeout, sessionLimit(user))

	if *logonMode {
		runLogonMode(displayEngine, artManager, navigator, inputHandler, sess, user)
		return 0
	}

	initialState, err := navigator.GetInitialState()
	if errors.Is(err, navigation.ErrNoYearsAvailable) {
		// Only possible when every installed year is still in the future.
		displayNotYet(displayEngine, artManager, user, inputHandler)
		return 0
	}
	if err != nil {
		logrus.WithError(err).Error("Failed to get initial navigation state")
		return 1
	}
	if err := validator.ValidateArtFiles(initialState.CurrentYear); err != nil {
		logrus.WithError(err).Error("Art file validation failed")
		return 1
	}

	prepareTerminal(displayEngine)
	defer restoreTerminal(displayEngine)

	runMainLoop(displayEngine, artManager, navigator, inputHandler, sess, initialState, user)
	return 0
}

// configureCalendar applies the debug date options to the navigator.
func configureCalendar(navigator *navigation.Navigator, date string, unlockAll bool) error {
	navigator.SetDisableDateCheck(unlockAll)
	if date == "" {
		return nil
	}
	day, err := time.ParseInLocation("2006-01-02", date, time.Local)
	if err != nil {
		return fmt.Errorf("invalid -debug-date (expected YYYY-MM-DD): %w", err)
	}
	navigator.SetClock(func() time.Time { return day })
	return nil
}

func newDisplayEngine(width, height int) *display.DisplayEngine {
	// BBS mode: raw CP437 bytes. Local mode: CP437 converted to UTF-8.
	mode := display.ModeCP437Raw
	if *localMode {
		mode = display.ModeCP437
	}
	return display.NewDisplayEngine(display.DisplayConfig{
		Mode:   mode,
		Width:  width,
		Height: height,
		NoIce:  *noIce,
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
}

// sessionLimit caps the session at the caller's remaining BBS time.
func sessionLimit(user display.User) time.Duration {
	if user.TimeLeft > 0 && user.TimeLeft < maxSessionTime {
		return user.TimeLeft
	}
	return maxSessionTime
}

func getUserInfo(localMode bool, info *gd.DropFile) display.User {
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

	logrus.Info("Running in BBS mode")
	if info != nil {
		return display.User{
			Alias: info.Alias, TimeLeft: time.Duration(info.TimeLeft) * time.Minute,
			Emulation: info.Emulation, NodeNum: info.Node, H: 25, W: 80, ModalH: 25, ModalW: 80,
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

// openDoor accepts either the door32.sys file or the directory holding it, so
// a BBS can pass its node directory or drop file path straight through.
func openDoor(path string) (gd.DropFile, error) {
	directory := path
	info, err := os.Stat(path)
	if err != nil {
		return gd.DropFile{}, err
	}
	if !info.IsDir() {
		if !strings.EqualFold(filepath.Base(path), "door32.sys") {
			return gd.DropFile{}, fmt.Errorf("drop file must be named door32.sys")
		}
		directory = filepath.Dir(path)
	}
	return gd.OpenSession(directory)
}

func detectTerminalSize(bbsMode, noDetect bool) (width, height int) {
	if noDetect {
		return 80, 25
	}
	if bbsMode {
		fmt.Fprint(gd.Output(), "\033[2J\033[H\033[12;25HDetecting Your Terminal Size...")
	}
	height, width = gd.GetTermSize()
	if bbsMode {
		gd.ClearScreen()
	}
	if !plausibleTermSize(width, height) {
		logrus.WithFields(logrus.Fields{"width": width, "height": height}).
			Warn("Implausible terminal size reported; using 80x25")
		return 80, 25
	}
	return width, height
}

// plausibleTermSize rejects probe answers the art cannot be drawn on. Some
// terminals answer the size probe wrongly (e.g. half the real height), which
// would otherwise truncate every screen.
func plausibleTermSize(width, height int) bool {
	return width >= 80 && height >= 24 && width < 999 && height < 999
}

// errSessionOver reports that the caller ran out of idle or total time.
type errSessionOver struct{ reason session.Reason }

func (e errSessionOver) Error() string { return "session ended: " + e.reason.String() }

// readKey waits for the caller's next key within the session's limits.
func readKey(in *input.InputHandler, sess *session.Manager) (rune, input.Key, error) {
	for {
		wait, reason := sess.Next()
		if wait <= 0 {
			return 0, input.KeyUnknown, errSessionOver{reason}
		}
		ch, key, err := in.ReadKeyTimeout(wait)
		if errors.Is(err, input.ErrTimeout) {
			continue // re-check: the limit has now passed
		}
		if err != nil {
			return 0, input.KeyUnknown, err
		}
		sess.Touch()
		return ch, key, nil
	}
}

func showSessionOver(displayEngine *display.DisplayEngine, reason session.Reason) {
	msg := "Your time is up - returning you to the BBS..."
	if reason == session.ReasonIdle {
		msg = "No key pressed for a while - returning you to the BBS..."
	}
	displayEngine.ShowMessage(msg)
	time.Sleep(2 * time.Second)
}

func prepareTerminal(displayEngine *display.DisplayEngine) {
	displayEngine.HideCursor()
	displayEngine.EnableBlinkMode() // Disable ICE mode to enable ANSI blink
	displayEngine.ClearScreen()
}

func restoreTerminal(displayEngine *display.DisplayEngine) {
	displayEngine.DisableBlinkMode() // Re-enable ICE mode
	displayEngine.ShowCursor()
	displayEngine.ClearScreen()            // ClearScreen flushes
	fmt.Fprint(gd.Output(), display.Reset) // Reset colors
}

func displayNotYet(displayEngine *display.DisplayEngine, artManager *art.Manager, user display.User, inputHandler *input.InputHandler) {
	if notYetPath := artManager.GetPath(0, 0, "notyet"); notYetPath != "" {
		displayEngine.Display(notYetPath, user)
	}
	waitForKey(inputHandler, exitScreenWait)
}

func waitForKey(handler *input.InputHandler, timeout time.Duration) {
	_, _, _ = handler.ReadKeyTimeout(timeout)
}

func runMainLoop(displayEngine *display.DisplayEngine, artManager *art.Manager,
	navigator *navigation.Navigator, inputHandler *input.InputHandler,
	sess *session.Manager, state navigation.State, user display.User) {

	currentState := state
	var currentArtPath string

	// For scrolling screens
	var infoLoaded, membersLoaded bool
	var infoLines, membersLines []string
	var infoScrollPos, membersScrollPos int

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
		case navigation.ScreenInfo:
			artPath = artManager.GetPath(currentState.CurrentYear, 0, "info")
		case navigation.ScreenMembers:
			artPath = artManager.GetPath(currentState.CurrentYear, 0, "members")
		}

		if artPath != "" && artPath != currentArtPath {
			logrus.WithFields(logrus.Fields{
				"artPath": artPath,
				"screen":  currentState.Screen,
				"day":     currentState.CurrentDay,
			}).Debug("Displaying art")

			switch currentState.Screen {
			case navigation.ScreenInfo:
				if !infoLoaded {
					lines, err := displayEngine.LoadAnsiLines(artPath)
					if err != nil {
						lines = []string{fmt.Sprintf("[Unable to load %s]", artPath)}
					}
					infoLines, infoLoaded, infoScrollPos = lines, true, 0
				}
				displayEngine.SetScrollState(infoScrollPos, len(infoLines))
				displayEngine.RenderScrollable(infoLines, infoScrollPos)
			case navigation.ScreenMembers:
				if !membersLoaded {
					lines, err := displayEngine.LoadAnsiLines(artPath)
					if err != nil {
						lines = []string{fmt.Sprintf("[Unable to load %s]", artPath)}
					}
					membersLines, membersLoaded, membersScrollPos = lines, true, 0
				}
				displayEngine.SetScrollState(membersScrollPos, len(membersLines))
				displayEngine.RenderScrollable(membersLines, membersScrollPos)
			default:
				if err := displayEngine.Display(artPath, user); err != nil {
					logrus.WithError(err).Error("Failed to display art")
				}
			}
			currentArtPath = artPath
		}

		char, key, err := readKey(inputHandler, sess)
		if err != nil {
			var over errSessionOver
			if errors.As(err, &over) {
				logrus.WithField("reason", over.reason).Info("Session limit reached")
				showSessionOver(displayEngine, over.reason)
			} else {
				logrus.WithError(err).Info("Input closed; ending door session")
			}
			return
		}

		// Handle scrolling for Info/Members screens
		if currentState.Screen == navigation.ScreenInfo {
			visibleLines := displayEngine.GetScrollState().VisibleLines
			if key == input.KeyArrowUp && infoScrollPos > 0 {
				infoScrollPos--
				displayEngine.RenderScrollableContentOnly(infoLines, infoScrollPos)
				continue
			} else if key == input.KeyArrowDown && infoScrollPos < len(infoLines)-visibleLines {
				infoScrollPos++
				displayEngine.RenderScrollableContentOnly(infoLines, infoScrollPos)
				continue
			}
		} else if currentState.Screen == navigation.ScreenMembers {
			visibleLines := displayEngine.GetScrollState().VisibleLines
			if key == input.KeyArrowUp && membersScrollPos > 0 {
				membersScrollPos--
				displayEngine.RenderScrollableContentOnly(membersLines, membersScrollPos)
				continue
			} else if key == input.KeyArrowDown && membersScrollPos < len(membersLines)-visibleLines {
				membersScrollPos++
				displayEngine.RenderScrollableContentOnly(membersLines, membersScrollPos)
				continue
			}
		}

		onYearMenu := currentState.Screen == navigation.ScreenWelcome || currentState.Screen == navigation.ScreenComeback

		// Handle quit/back navigation
		if char == 'q' || char == 'Q' || key == input.KeyEsc {
			latestYear := currentState.AvailableYears[len(currentState.AvailableYears)-1]

			switch {
			case onYearMenu && currentState.CurrentYear == latestYear:
				// Exit from the newest year's welcome/comeback screen
				logrus.Info("User requested exit")
				if exitPath := artManager.GetPath(currentState.CurrentYear, 0, "goodbye"); exitPath != "" {
					displayEngine.Display(exitPath, user)
					waitForKey(inputHandler, exitScreenWait)
				}
				return
			case currentState.Screen == navigation.ScreenInfo || currentState.Screen == navigation.ScreenMembers:
				// Return to welcome screen from Info/Members
				currentState.Screen = navigation.ScreenWelcome
				infoScrollPos, membersScrollPos = 0, 0
				infoLoaded, membersLoaded = false, false
			case currentState.CurrentYear != latestYear:
				// Back to the newest year's welcome screen, recomputing its open days
				if newState, _, err := navigator.SelectYear(latestYear, currentState); err == nil {
					currentState = newState
				}
			default:
				currentState.Screen = navigation.ScreenWelcome
			}
			continue
		}

		// Handle year selection from welcome/comeback screen with numeric keys
		if onYearMenu && char >= '1' && char <= '9' {
			yearIndex := int(char - '0')
			if newState, _, err := navigator.SelectYearByIndex(yearIndex, currentState); err == nil {
				logrus.WithField("selectedYear", newState.CurrentYear).Info("Year selected")
				currentState = newState
			}
			continue
		}

		// Handle Info/Members menu keys from welcome/comeback screen
		if onYearMenu && (char == 'i' || char == 'I') {
			currentState.Screen = navigation.ScreenInfo
			infoScrollPos, infoLoaded = 0, false
			continue
		}
		if onYearMenu && (char == 'm' || char == 'M') {
			currentState.Screen = navigation.ScreenMembers
			membersScrollPos, membersLoaded = 0, false
			continue
		}

		// Handle scrolling of tall day art
		if currentState.Screen == navigation.ScreenDay {
			scrollState := displayEngine.GetScrollState()
			if key == input.KeyArrowUp && scrollState.CanScrollUp {
				displayEngine.ScrollUp()
				continue
			}
			if key == input.KeyArrowDown && scrollState.CanScrollDown {
				displayEngine.ScrollDown()
				continue
			}
		}

		// Handle navigation
		var direction navigation.Direction

		// RETURN on WELCOME/COMEBACK behaves like arrow right
		if onYearMenu && key == input.KeyEnter {
			direction = navigation.DirRight
		}

		switch key {
		case input.KeyArrowRight:
			direction = navigation.DirRight
		case input.KeyArrowLeft:
			direction = navigation.DirLeft
		case input.KeyPageUp:
			direction = navigation.DirPageUp
		case input.KeyPageDown:
			direction = navigation.DirPageDown
		case input.KeyHome:
			direction = navigation.DirHome
		case input.KeyEnd:
			direction = navigation.DirEnd
		}

		if direction != navigation.DirNone {
			newState, _, err := navigator.Navigate(direction, currentState)
			if err != nil {
				logrus.WithError(err).Error("Navigation error")
				continue
			}
			currentState = newState
		}
	}
}

// runLogonMode shows today's door once at logon, then returns to the BBS. The
// caller has already checked that a season is running.
func runLogonMode(displayEngine *display.DisplayEngine, artManager *art.Manager,
	navigator *navigation.Navigator, inputHandler *input.InputHandler,
	sess *session.Manager, user display.User) {

	year, live := navigator.LiveYear()
	if !live {
		return
	}
	day := navigator.MaxDay(year)

	prepareTerminal(displayEngine)
	defer restoreTerminal(displayEngine)

	if err := displayEngine.Display(artManager.GetPath(year, day, "day"), user); err != nil {
		logrus.WithError(err).Error("Failed to display day art in logon mode")
	}

	// Discard type-ahead (e.g. the LF after the caller's last Enter at the BBS)
	// so it doesn't dismiss the art the instant it appears.
	inputHandler.Drain(150*time.Millisecond, 64)

	wait := logonWait
	if left, _ := sess.Next(); left < wait {
		wait = left
	}
	if _, _, err := inputHandler.ReadKeyTimeout(wait); err != nil && !errors.Is(err, input.ErrTimeout) {
		logrus.WithError(err).Info("Input closed in logon mode")
	}
}
