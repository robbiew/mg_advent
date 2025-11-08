package display

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

// DisplayEngine implements the Displayer interface
type DisplayEngine struct {
	config       DisplayConfig
	themeManager *ThemeManager
	scrollState  ScrollState
	cache        map[string][]string
}

// NewDisplayEngine creates a new display engine
func NewDisplayEngine(config DisplayConfig) *DisplayEngine {
	return &DisplayEngine{
		config:       config,
		themeManager: NewThemeManager(),
		cache:        make(map[string][]string),
		scrollState: ScrollState{
			CurrentLine:  0,
			TotalLines:   0,
			VisibleLines: config.Height,
		},
	}
}

// Display displays the content of an ANSI file
func (de *DisplayEngine) Display(filePath string, user User) error {
	fmt.Print(Reset) // Reset text and background colors
	de.ClearScreen()

	// Load and process content
	content, err := de.loadAndProcess(filePath)
	if err != nil {
		log.Printf("ERROR: Failed to load file %s: %v", filePath, err)
		fmt.Println("Error: Unable to load art. Please contact the Sysop.")
		return err
	}

	if len(content) == 0 {
		log.Printf("ERROR: File %s is empty", filePath)
		fmt.Println("Error: The art file is empty or invalid.")
		return fmt.Errorf("empty file")
	}

	// Handle scrolling if needed
	if len(content) > de.config.Height && de.config.Scrolling.Enabled {
		return de.renderWithScrolling(content)
	}

	// Normal rendering
	return de.renderNormal(content)
}

// loadAndProcess loads and processes the file content
func (de *DisplayEngine) loadAndProcess(filePath string) ([]string, error) {
	// Check cache first
	if cached, exists := de.cache[filePath]; exists {
		return cached, nil
	}

	// Load file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Process content based on mode
	var lines []string
	switch de.config.Mode {
	case ModeUTF8:
		lines = de.processUTF8(content)
	case ModeCP437:
		lines = de.processCP437(content)
	case ModeCP437Raw:
		lines = de.processCP437Raw(content)
	default:
		lines = de.processUTF8(content) // Default fallback
	}

	// Handle 80-column issue if enabled
	if de.config.Columns.Handle80ColumnIssue {
		lines = de.handle80ColumnIssue(lines)
	}

	// Cache the result
	if de.config.Performance.CacheEnabled {
		de.cache[filePath] = lines
	}

	return lines, nil
}

// processUTF8 processes UTF-8 content
func (de *DisplayEngine) processUTF8(content []byte) []string {
	noSauce := trimStringFromSauce(string(content))
	return strings.Split(noSauce, "\r\n")
}

// processCP437 processes CP437 content with UTF-8 conversion (for local mode)
func (de *DisplayEngine) processCP437(content []byte) []string {
	noSauce := trimStringFromSauce(string(content))
	lines := strings.Split(noSauce, "\r\n")

	// Convert each line from CP437 to UTF-8
	converted := make([]string, len(lines))
	for i, line := range lines {
		// Convert line from CP437 to UTF-8 for local display
		utf8Line, err := charmap.CodePage437.NewDecoder().String(line)
		if err != nil {
			log.Printf("Error converting to UTF-8: %v", err)
			utf8Line = line // Fallback to original
		}

		// Apply terminal-specific fixes if needed
		utf8Line = de.applyTerminalFixes(utf8Line)
		converted[i] = utf8Line
	}

	return converted
}

// applyTerminalFixes applies fixes for terminal compatibility issues
func (de *DisplayEngine) applyTerminalFixes(line string) string {
	// For now, return as-is. This can be extended to detect terminal
	// capabilities and apply appropriate transformations
	return line
}

// processCP437WithEnhancedConversion provides enhanced CP437 to UTF-8 conversion
// with special handling for macOS Terminal compatibility
func (de *DisplayEngine) processCP437WithEnhancedConversion(content []byte) []string {
	noSauce := trimStringFromSauce(string(content))
	lines := strings.Split(noSauce, "\r\n")

	converted := make([]string, len(lines))
	for i, line := range lines {
		// Convert from CP437 to UTF-8
		utf8Line, err := charmap.CodePage437.NewDecoder().String(line)
		if err != nil {
			log.Printf("Error converting line to UTF-8: %v", err)
			utf8Line = line // Fallback
		}

		// Additional processing for macOS Terminal compatibility
		// Some terminals may need special character handling
		converted[i] = utf8Line
	}

	return converted
}

// processCP437Raw processes CP437 content without conversion (for BBS mode)
func (de *DisplayEngine) processCP437Raw(content []byte) []string {
	noSauce := trimStringFromSauce(string(content))
	return strings.Split(noSauce, "\r\n")
}

// handle80ColumnIssue handles the 80-column line break issue
func (de *DisplayEngine) handle80ColumnIssue(lines []string) []string {
	if de.config.Width != 80 {
		return lines
	}

	result := make([]string, len(lines))
	for i, line := range lines {
		if len(line) == 80 {
			// Truncate to 79 to prevent unwanted line wrapping
			result[i] = line[:79]
		} else {
			result[i] = line
		}
	}
	return result
}

// renderNormal renders content without scrolling
func (de *DisplayEngine) renderNormal(lines []string) error {
	for i := 0; i < de.config.Height && i < len(lines); i++ {
		line := lines[i]
		if i == de.config.Height-1 {
			fmt.Print(line)
		} else {
			fmt.Println(line)
		}

		// Optional delay between lines
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}

// renderWithScrolling renders content with scrolling support
func (de *DisplayEngine) renderWithScrolling(lines []string) error {
	de.scrollState.TotalLines = len(lines)
	de.scrollState.CurrentLine = 0
	de.updateScrollState()

	return de.renderVisibleLines(lines)
}

// renderVisibleLines renders the currently visible lines
func (de *DisplayEngine) renderVisibleLines(lines []string) error {
	de.ClearScreen()

	startLine := de.scrollState.CurrentLine
	endLine := startLine + de.config.Height
	if endLine > len(lines) {
		endLine = len(lines)
	}

	for i := startLine; i < endLine && i < len(lines); i++ {
		line := lines[i]
		if i == endLine-1 {
			fmt.Print(line)
		} else {
			fmt.Println(line)
		}
	}

	// Show scroll indicators if enabled
	if de.config.Scrolling.Indicators {
		de.showScrollIndicators()
	}

	return nil
}

// ScrollUp scrolls up one line
func (de *DisplayEngine) ScrollUp() error {
	if de.scrollState.CurrentLine > 0 {
		de.scrollState.CurrentLine--
		de.updateScrollState()
		// Note: Would need content reference to re-render
	}
	return nil
}

// ScrollDown scrolls down one line
func (de *DisplayEngine) ScrollDown() error {
	if de.scrollState.CurrentLine < de.scrollState.TotalLines-de.config.Height {
		de.scrollState.CurrentLine++
		de.updateScrollState()
		// Note: Would need content reference to re-render
	}
	return nil
}

// GetScrollState returns the current scroll state
func (de *DisplayEngine) GetScrollState() ScrollState {
	return de.scrollState
}

// updateScrollState updates the scroll state flags
func (de *DisplayEngine) updateScrollState() {
	de.scrollState.CanScrollUp = de.scrollState.CurrentLine > 0
	de.scrollState.CanScrollDown = de.scrollState.CurrentLine < de.scrollState.TotalLines-de.config.Height
}

// showScrollIndicators shows scroll position indicators
func (de *DisplayEngine) showScrollIndicators() {
	if de.scrollState.TotalLines <= de.config.Height {
		return
	}

	// Position indicator at bottom right
	percentage := float64(de.scrollState.CurrentLine) / float64(de.scrollState.TotalLines-de.config.Height) * 100
	fmt.Printf("\033[%d;%dH[%d%%]", de.config.Height, de.config.Width-5, int(percentage))
}

// ClearScreen clears the screen
func (de *DisplayEngine) ClearScreen() error {
	fmt.Print(EraseScreen)
	MoveCursor(0, 0)
	return nil
}

// MoveCursor moves the cursor to the specified position
func (de *DisplayEngine) MoveCursor(x, y int) error {
	fmt.Printf(Esc+"%d;%df", y, x)
	return nil
}

// GetDimensions returns the display dimensions
func (de *DisplayEngine) GetDimensions() (width, height int) {
	return de.config.Width, de.config.Height
}

// SetTheme sets the display theme (placeholder for future implementation)
func (de *DisplayEngine) SetTheme(theme string) error {
	de.config.Theme = theme
	return nil
}

// ANSI escape sequences (extracted from original ansiart.go)
const (
	Esc         = "\u001B["
	EraseScreen = Esc + "2J"
	Reset       = Esc + "0m"
)

// MoveCursor moves cursor to X, Y location
func MoveCursor(x int, y int) {
	fmt.Printf(Esc+"%d;%df", y, x)
}

// trimStringFromSauce trims SAUCE metadata from a string
func trimStringFromSauce(s string) string {
	if idx := strings.Index(s, "COMNT"); idx != -1 {
		leftOfDelimiter := strings.Split(s, "COMNT")[0]
		return trimLastChar(leftOfDelimiter)
	}
	if idx := strings.Index(s, "SAUCE00"); idx != -1 {
		leftOfDelimiter := strings.Split(s, "SAUCE00")[0]
		return trimLastChar(leftOfDelimiter)
	}
	return s
}

// trimMetadata trims metadata based on delimiters
func trimMetadata(s string, delimiters ...string) string {
	for _, delimiter := range delimiters {
		if idx := strings.Index(s, delimiter); idx != -1 {
			return trimLastChar(s[:idx])
		}
	}
	return s
}

// trimLastChar trims the last character from a string
func trimLastChar(s string) string {
	if len(s) > 0 {
		_, size := utf8.DecodeLastRuneInString(s)
		return s[:len(s)-size]
	}
	return s
}
