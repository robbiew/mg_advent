package display

import (
	"time"
)

// DisplayMode represents the display rendering mode
type DisplayMode int

const (
	ModeCP437 DisplayMode = iota
	ModeUTF8
	ModeCP437Raw
)

// ScrollState represents the current scrolling state
type ScrollState struct {
	CurrentLine   int
	TotalLines    int
	VisibleLines  int
	CanScrollUp   bool
	CanScrollDown bool
}

// DisplayConfig holds display-related configuration
type DisplayConfig struct {
	Mode        DisplayMode
	Width       int
	Height      int
	Theme       string
	Scrolling   ScrollingConfig
	Columns     ColumnConfig
	Performance PerformanceConfig
	NoIce       bool // Disable ICE mode control codes
}

type ScrollingConfig struct {
	Enabled           bool
	Indicators        bool
	KeyboardShortcuts bool
}

type ColumnConfig struct {
	Handle80ColumnIssue bool
	AutoDetectWidth     bool
}

type PerformanceConfig struct {
	CacheEnabled bool
	CacheSizeMB  int
	PreloadLines int
}

// Displayer interface for display operations
type Displayer interface {
	Display(filePath string) error
	ClearScreen() error
	MoveCursor(x, y int) error
	GetDimensions() (width, height int)
	SetTheme(theme string) error
	ScrollUp() error
	ScrollDown() error
	GetScrollState() ScrollState
}

// User represents BBS user information
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
