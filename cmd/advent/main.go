package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	gd "github.com/robbiew/godoors"

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
	noIce        = flag.Bool("noice", false, "don't send the blink/iCE colour mode sequence")
	noDetect     = flag.Bool("nodetect", false, "disable terminal size detection (use default 80x25)")
	logFile      = flag.String("log", "", "append a detailed log to this file (default: errors only, to stderr)")
	slideEvery   = flag.Duration("slide", 8*time.Second, "slideshow: how long each door is shown")
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

	closeLog, err := setupLogging(*logFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "advent:", err)
		os.Exit(1)
	}
	code := run()
	closeLog()
	os.Exit(code)
}

// setupLogging logs errors to stderr by default. Under some BBSes stderr is
// the caller's terminal, so -log sends a detailed log to a file instead.
func setupLogging(path string) (func(), error) {
	var w io.Writer = os.Stderr
	level := slog.LevelError
	closeLog := func() {}
	if path != "" {
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, fmt.Errorf("opening log file: %w", err)
		}
		w, level = f, slog.LevelInfo
		closeLog = func() { f.Close() }
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: level})))
	return closeLog, nil
}

// run is the door's body. It returns the process exit code; deferred cleanup
// (terminal restore, closing the BBS session) runs before main exits.
func run() int {
	artManager := art.NewManager(embedded.ArtFS, "art")
	navigator := navigation.NewNavigator(embedded.ArtFS, "art")
	validator := validation.NewValidator(embedded.ArtFS, "art")
	if err := configureCalendar(navigator, *debugDate, *disableDate); err != nil {
		slog.Error("invalid date options", "err", err)
		return 1
	}

	// Logon mode only has something to show while a season is running. Bail
	// out before touching the terminal so off-season logons stay silent.
	if *logonMode {
		if _, live := navigator.LiveYear(); !live {
			slog.Info("logon mode: no advent season running today; nothing to show")
			return 0
		}
	}

	// Select the transport before any terminal probe or input read.
	var doorInfo *gd.DropFile
	if *dropfilePath != "" {
		info, err := openDoor(*dropfilePath)
		if err != nil {
			slog.Error("failed to open BBS connection", "err", err)
			return 1
		}
		doorInfo = &info
	}
	defer gd.CloseSession()

	inputHandler := input.NewInputHandler()
	if err := inputHandler.Open(); err != nil {
		slog.Error("failed to open input", "err", err)
		return 1
	}
	defer inputHandler.Close()

	user := getUserInfo(*localMode, doorInfo)
	width, height := detectTerminalSize(doorInfo != nil, *noDetect)
	slog.Info("session starting", "user", user.Alias, "node", user.NodeNum, "width", width, "height", height)

	displayEngine := newDisplayEngine(width, height)
	if err := validator.ValidateTerminalSize(width, height); err != nil {
		slog.Warn("terminal size validation failed; continuing", "err", err)
	}
	if err := validator.ValidateEmulation(user.Emulation); err != nil {
		slog.Warn("ANSI emulation validation failed; continuing", "err", err)
	}

	sess := session.NewManager(idleTimeout, sessionLimit(user))
	if wait, reason := sess.Next(); wait <= 0 {
		// The caller has no BBS time left: send them back before showing art.
		slog.Info("no session time remaining", "reason", reason)
		showSessionOver(displayEngine, reason)
		return 0
	}

	if *logonMode {
		runLogonMode(displayEngine, artManager, navigator, inputHandler, sess)
		return 0
	}

	initialState, err := navigator.GetInitialState()
	if errors.Is(err, navigation.ErrNoYearsAvailable) {
		// Only possible when every installed year is still in the future.
		displayNotYet(displayEngine, artManager, inputHandler)
		return 0
	}
	if err != nil {
		slog.Error("failed to get initial navigation state", "err", err)
		return 1
	}
	if err := validator.ValidateArtFiles(initialState.CurrentYear); err != nil {
		slog.Error("art file validation failed", "err", err)
		return 1
	}

	prepareTerminal(displayEngine)
	defer restoreTerminal(displayEngine)

	runMainLoop(displayEngine, artManager, navigator, inputHandler, sess, initialState)
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
	}, embedded.ArtFS)
}

// sessionLimit caps the session at the caller's remaining BBS time. TimeLeft
// is always known (door32.sys, or the local/fallback defaults), so zero or less
// means the caller has no time left, not that the time is unknown.
func sessionLimit(user display.User) time.Duration {
	return min(max(user.TimeLeft, 0), maxSessionTime)
}

func getUserInfo(localMode bool, info *gd.DropFile) display.User {
	switch {
	case localMode:
		return display.User{Alias: "SysOp", TimeLeft: 120 * time.Minute, Emulation: 1, NodeNum: 1}
	case info != nil:
		return display.User{
			Alias:     info.Alias,
			TimeLeft:  time.Duration(info.TimeLeft) * time.Minute,
			Emulation: info.Emulation,
			NodeNum:   info.Node,
		}
	default:
		return display.User{Alias: "BBSUser", TimeLeft: 30 * time.Minute, Emulation: 1, NodeNum: 1}
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
		slog.Warn("implausible terminal size reported; using 80x25", "width", width, "height", height)
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

// errNoKey reports that readKeyWithin's own wait ended without a key.
var errNoKey = errors.New("no key pressed")

// readKey waits for the caller's next key within the session's limits.
func readKey(in *input.InputHandler, sess *session.Manager) (rune, input.Key, error) {
	return readKeyWithin(in, sess, 0)
}

// readKeyWithin is readKey that also gives up after within (if > 0) with
// errNoKey, e.g. to advance a slideshow.
func readKeyWithin(in *input.InputHandler, sess *session.Manager, within time.Duration) (rune, input.Key, error) {
	var giveUp time.Time
	if within > 0 {
		giveUp = time.Now().Add(within)
	}
	for {
		wait, reason := sess.Next()
		if wait <= 0 {
			return 0, input.KeyUnknown, errSessionOver{reason}
		}
		if !giveUp.IsZero() {
			left := time.Until(giveUp)
			if left <= 0 {
				return 0, input.KeyUnknown, errNoKey
			}
			wait = min(wait, left)
		}
		ch, key, err := in.ReadKeyTimeout(wait)
		if errors.Is(err, input.ErrTimeout) {
			continue // re-check which limit has passed
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
	displayEngine.EnableBlinkMode()
	displayEngine.ClearScreen()
}

func restoreTerminal(displayEngine *display.DisplayEngine) {
	displayEngine.ShowCursor()
	displayEngine.ClearScreen()
	displayEngine.ResetAttributes()
}

func displayNotYet(displayEngine *display.DisplayEngine, artManager *art.Manager, inputHandler *input.InputHandler) {
	displayEngine.Display(artManager.GetPath(0, 0, "notyet"))
	waitForKey(inputHandler, exitScreenWait)
}

func waitForKey(handler *input.InputHandler, timeout time.Duration) {
	_, _, _ = handler.ReadKeyTimeout(timeout)
}

// runLogonMode shows today's door once at logon, then returns to the BBS. The
// caller has already checked that a season is running.
func runLogonMode(displayEngine *display.DisplayEngine, artManager *art.Manager,
	navigator *navigation.Navigator, inputHandler *input.InputHandler, sess *session.Manager) {

	year, live := navigator.LiveYear()
	if !live {
		return
	}
	day := navigator.MaxDay(year)

	prepareTerminal(displayEngine)
	defer restoreTerminal(displayEngine)

	if err := displayEngine.Display(artManager.GetPath(year, day, "day")); err != nil {
		slog.Error("failed to display day art in logon mode", "err", err)
	}

	// Discard type-ahead (e.g. the LF after the caller's last Enter at the BBS)
	// so it doesn't dismiss the art the instant it appears.
	inputHandler.Drain(150*time.Millisecond, 64)

	wait := logonWait
	if left, _ := sess.Next(); left < wait {
		wait = left
	}
	if _, _, err := inputHandler.ReadKeyTimeout(wait); err != nil && !errors.Is(err, input.ErrTimeout) {
		slog.Info("input closed in logon mode", "err", err)
	}
}
