package input

import (
	"time"

	gd "github.com/robbiew/godoors"
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

// InputHandler adapts GoDoors key events to the application's existing keys.
type InputHandler struct{}

func NewInputHandler() *InputHandler  { return &InputHandler{} }
func (ih *InputHandler) Open() error  { return gd.OpenInput() }
func (ih *InputHandler) Close() error { gd.CloseInput(); return nil }

func (ih *InputHandler) ReadKey() (rune, Key, error) {
	ch, key, err := gd.ReadKey()
	if err != nil {
		return 0, KeyUnknown, err
	}
	return translateKey(ch, key)
}

func (ih *InputHandler) ReadKeyTimeout(timeout time.Duration) (rune, Key, error) {
	ch, key, err := gd.ReadKeyTimeout(timeout)
	if err != nil {
		return 0, KeyUnknown, err
	}
	return translateKey(ch, key)
}

func translateKey(ch rune, key gd.Key) (rune, Key, error) {
	switch key {
	case gd.KeyEsc:
		return 0, KeyEsc, nil
	case gd.KeyEnter:
		return 0, KeyEnter, nil
	case gd.KeyUp:
		return 0, KeyArrowUp, nil
	case gd.KeyDown:
		return 0, KeyArrowDown, nil
	case gd.KeyLeft:
		return 0, KeyArrowLeft, nil
	case gd.KeyRight:
		return 0, KeyArrowRight, nil
	case gd.KeyPageUp:
		return 0, KeyPageUp, nil
	case gd.KeyPageDown:
		return 0, KeyPageDown, nil
	case gd.KeyHome:
		return 0, KeyHome, nil
	case gd.KeyEnd:
		return 0, KeyEnd, nil
	case gd.KeyInsert:
		return 0, KeyInsert, nil
	case gd.KeyDelete:
		return 0, KeyDelete, nil
	}
	if key >= gd.KeyF1 && key <= gd.KeyF12 {
		return 0, KeyF1 + Key(key-gd.KeyF1), nil
	}
	switch ch {
	case ' ':
		return 0, KeySpace, nil
	case '\b', 0x7f:
		return 0, KeyBackspace, nil
	case '\t':
		return 0, KeyTab, nil
	}
	if IsPrintable(ch) {
		return ch, KeyUnknown, nil
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
