package input

import (
	"fmt"
	"os"
	"runtime"

	"github.com/robbiew/advent/internal/bbs"
	"golang.org/x/term"
)

// Key represents a keyboard input
type Key int

const (
	KeyUnknown Key = iota
	KeyEsc
	KeyEnter
	KeySpace
	KeyBackspace
	KeyTab
	KeyArrowUp
	KeyArrowDown
	KeyArrowLeft
	KeyArrowRight
	KeyPageUp
	KeyPageDown
	KeyHome
	KeyEnd
	KeyInsert
	KeyDelete
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
)

// InputHandler manages keyboard input
type InputHandler struct {
	oldState  *term.State
	bbsConn   *bbs.BBSConnection
	isWindows bool
}

// NewInputHandler creates a new input handler
func NewInputHandler() *InputHandler {
	return &InputHandler{
		isWindows: runtime.GOOS == "windows",
	}
}

// SetBBSConnection sets the BBS connection for Windows socket I/O
func (ih *InputHandler) SetBBSConnection(conn *bbs.BBSConnection) {
	ih.bbsConn = conn
}

// Open initializes the terminal for raw input
func (ih *InputHandler) Open() error {
	if ih.isWindows && ih.bbsConn != nil {
		// Windows with socket connection - no terminal setup needed
		return nil
	}

	// Unix/Linux: Setup raw terminal mode
	// Get the current terminal state
	oldState, err := term.GetState(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to get terminal state: %w", err)
	}
	ih.oldState = oldState

	// Make stdin raw
	_, err = term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to make terminal raw: %w", err)
	}

	return nil
}

// Close restores the terminal to its original state
func (ih *InputHandler) Close() error {
	if ih.oldState != nil {
		return term.Restore(int(os.Stdin.Fd()), ih.oldState)
	}
	return nil
}

// ReadKey reads a single key press
func (ih *InputHandler) ReadKey() (rune, Key, error) {
	var buf [256]byte
	var n int
	var err error

	if ih.isWindows && ih.bbsConn != nil {
		// Windows: Read from BBS socket connection
		n, err = ih.bbsConn.Read(buf[:])
	} else {
		// Unix/Linux: Read from stdin
		n, err = os.Stdin.Read(buf[:])
	}

	if err != nil {
		return 0, KeyUnknown, err
	}

	if n == 0 {
		return 0, KeyUnknown, nil
	}

	// Handle escape sequences
	if buf[0] == '\x1b' {
		if n == 1 {
			return 0, KeyEsc, nil
		}

		// Check for escape sequences
		seq := string(buf[:n])
		switch seq {
		case "\x1b[A", "\x1b[1A": // Up arrow
			return 0, KeyArrowUp, nil
		case "\x1b[B", "\x1b[1B": // Down arrow
			return 0, KeyArrowDown, nil
		case "\x1b[C", "\x1b[1C": // Right arrow
			return 0, KeyArrowRight, nil
		case "\x1b[D", "\x1b[1D": // Left arrow
			return 0, KeyArrowLeft, nil
		case "\x1b[5~": // Page Up
			return 0, KeyPageUp, nil
		case "\x1b[6~": // Page Down
			return 0, KeyPageDown, nil
		case "\x1b[1~", "\x1b[H": // Home
			return 0, KeyHome, nil
		case "\x1b[4~", "\x1b[F": // End
			return 0, KeyEnd, nil
		case "\x1b[2~": // Insert
			return 0, KeyInsert, nil
		case "\x1b[3~": // Delete
			return 0, KeyDelete, nil
		case "\x1b[11~": // F1
			return 0, KeyF1, nil
		case "\x1b[12~": // F2
			return 0, KeyF2, nil
		case "\x1b[13~": // F3
			return 0, KeyF3, nil
		case "\x1b[14~": // F4
			return 0, KeyF4, nil
		case "\x1b[15~": // F5
			return 0, KeyF5, nil
		case "\x1b[16~": // F6
			return 0, KeyF6, nil
		case "\x1b[17~": // F7
			return 0, KeyF7, nil
		case "\x1b[18~": // F8
			return 0, KeyF8, nil
		case "\x1b[19~": // F9
			return 0, KeyF9, nil
		case "\x1b[20~": // F10
			return 0, KeyF10, nil
		case "\x1b[21~": // F11
			return 0, KeyF11, nil
		case "\x1b[22~": // F12
			return 0, KeyF12, nil
		default:
			// Unknown escape sequence, treat as regular character
			if n > 1 && buf[1] != '[' {
				return rune(buf[1]), KeyUnknown, nil
			}
		}
	}

	// Handle special keys
	switch buf[0] {
	case '\r', '\n':
		return 0, KeyEnter, nil
	case ' ':
		return 0, KeySpace, nil
	case '\b', '\x7f':
		return 0, KeyBackspace, nil
	case '\t':
		return 0, KeyTab, nil
	default:
		// Regular character
		if buf[0] >= 32 && buf[0] <= 126 {
			return rune(buf[0]), KeyUnknown, nil
		}
	}

	return 0, KeyUnknown, nil
}

// IsPrintable checks if a rune is printable
func IsPrintable(r rune) bool {
	return r >= 32 && r <= 126
}

// KeyToString converts a Key to its string representation
func KeyToString(key Key) string {
	switch key {
	case KeyEsc:
		return "Esc"
	case KeyEnter:
		return "Enter"
	case KeySpace:
		return "Space"
	case KeyBackspace:
		return "Backspace"
	case KeyTab:
		return "Tab"
	case KeyArrowUp:
		return "↑"
	case KeyArrowDown:
		return "↓"
	case KeyArrowLeft:
		return "←"
	case KeyArrowRight:
		return "→"
	case KeyPageUp:
		return "PageUp"
	case KeyPageDown:
		return "PageDown"
	case KeyHome:
		return "Home"
	case KeyEnd:
		return "End"
	case KeyInsert:
		return "Insert"
	case KeyDelete:
		return "Delete"
	case KeyF1:
		return "F1"
	case KeyF2:
		return "F2"
	case KeyF3:
		return "F3"
	case KeyF4:
		return "F4"
	case KeyF5:
		return "F5"
	case KeyF6:
		return "F6"
	case KeyF7:
		return "F7"
	case KeyF8:
		return "F8"
	case KeyF9:
		return "F9"
	case KeyF10:
		return "F10"
	case KeyF11:
		return "F11"
	case KeyF12:
		return "F12"
	default:
		return "Unknown"
	}
}
