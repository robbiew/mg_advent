package display

import "time"

// DisplayMode selects how art bytes reach the terminal.
type DisplayMode int

const (
	// ModeCP437 converts the CP437 art to UTF-8, for a local terminal.
	ModeCP437 DisplayMode = iota
	// ModeCP437Raw sends CP437 bytes unchanged, for a BBS caller.
	ModeCP437Raw
)

// ScrollState describes the window onto art taller than the screen.
type ScrollState struct {
	CurrentLine   int
	TotalLines    int
	VisibleLines  int
	CanScrollUp   bool
	CanScrollDown bool
}

// DisplayConfig holds the terminal the engine draws for.
type DisplayConfig struct {
	Mode   DisplayMode
	Width  int
	Height int
	NoIce  bool // don't send the blink/iCE mode sequence
}

// User is the caller, as described by the BBS drop file.
type User struct {
	Alias     string
	TimeLeft  time.Duration
	Emulation int
	NodeNum   int
}
