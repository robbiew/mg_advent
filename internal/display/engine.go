package display

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

// SetScrollState allows external code to set the scroll state for custom scrollable screens
func (de *DisplayEngine) SetScrollState(currentLine, totalLines int) {
	// Determine footer height by loading the footer file
	footerHeight := 1 // Default to 1 row
	footerLines, err := de.loadAndProcess("art/common/FOOTER.ANS")
	if err == nil && len(footerLines) > 0 {
		footerHeight = len(footerLines)
		if footerHeight > 2 {
			// Cap at 2 lines to avoid taking too much screen space
			footerHeight = 2
		}
	}

	de.scrollState.CurrentLine = currentLine
	de.scrollState.TotalLines = totalLines
	de.scrollState.VisibleLines = de.config.Height - footerHeight
	de.updateScrollState()
}

// LoadAnsiLines loads and processes an ANSI file into lines (CP437/UTF-8 aware)
func (de *DisplayEngine) LoadAnsiLines(filePath string) ([]string, error) {
	return de.loadAndProcess(filePath)
}

// RenderScrollable renders a window of lines at the given scroll position
func (de *DisplayEngine) RenderScrollable(lines []string, scrollPos int) error {
	if len(lines) == 0 {
		return nil
	}
	if scrollPos < 0 {
		scrollPos = 0
	}

	// Determine footer height by loading the footer file
	footerHeight := 1 // Default to 1 row
	footerLines, err := de.loadAndProcess("art/common/FOOTER.ANS")
	if err == nil && len(footerLines) > 0 {
		footerHeight = len(footerLines)
		if footerHeight > 2 {
			// Cap at 2 lines to avoid taking too much screen space
			footerHeight = 2
		}
	}

	// Reserve space for footer
	usableHeight := de.config.Height - footerHeight

	maxStart := len(lines) - usableHeight
	if scrollPos > maxStart {
		scrollPos = maxStart
		if scrollPos < 0 {
			scrollPos = 0
		}
	}
	de.ClearScreen()
	end := scrollPos + usableHeight
	if end > len(lines) {
		end = len(lines)
	}

	// Apply 80-column handling for each visible line
	visibleLines := lines[scrollPos:end]
	if de.config.Columns.Handle80ColumnIssue && de.config.Width == 80 {
		visibleLines = de.handle80ColumnIssue(visibleLines)
	}

	for i := 0; i < len(visibleLines); i++ {
		isLast := i == len(visibleLines)-1
		de.printLine(visibleLines[i], isLast)
	}
	// Draw menu bar at bottom
	de.renderMenuBar()
	de.flushOutput()
	return nil
}

// RenderScrollableContentOnly renders only the content area without clearing screen or redrawing footer
// This is used for efficient scrolling where the footer remains static
func (de *DisplayEngine) RenderScrollableContentOnly(lines []string, scrollPos int) error {
	if len(lines) == 0 {
		return nil
	}
	if scrollPos < 0 {
		scrollPos = 0
	}

	// Determine footer height by loading the footer file
	footerHeight := 1 // Default to 1 row
	footerLines, err := de.loadAndProcess("art/common/FOOTER.ANS")
	if err == nil && len(footerLines) > 0 {
		footerHeight = len(footerLines)
		if footerHeight > 2 {
			// Cap at 2 lines to avoid taking too much screen space
			footerHeight = 2
		}
	}

	// Reserve space for footer
	usableHeight := de.config.Height - footerHeight

	maxStart := len(lines) - usableHeight
	if scrollPos > maxStart {
		scrollPos = maxStart
		if scrollPos < 0 {
			scrollPos = 0
		}
	}

	end := scrollPos + usableHeight
	if end > len(lines) {
		end = len(lines)
	}

	// Apply 80-column handling for each visible line
	visibleLines := lines[scrollPos:end]
	if de.config.Columns.Handle80ColumnIssue && de.config.Width == 80 {
		visibleLines = de.handle80ColumnIssue(visibleLines)
	}

	// Move cursor to top-left and render each line at its specific position
	// This avoids clearing the screen and preserves the footer
	for i := 0; i < usableHeight; i++ {
		// Position cursor at the start of line i+1 (1-indexed)
		de.output.Write([]byte(fmt.Sprintf("\033[%d;1H", i+1)))

		if i < len(visibleLines) {
			// Print the line content
			de.output.Write([]byte(visibleLines[i]))
		}

		// Clear to end of line to remove any leftover content
		de.output.Write([]byte("\033[K"))
	}

	de.flushOutput()
	return nil
}

// renderMenuBar draws the menu bar at the last row of the terminal
func (de *DisplayEngine) renderMenuBar() {
	// Load the FOOTER.ANS file
	footerLines, err := de.loadAndProcess("art/common/FOOTER.ANS")
	if err != nil {
		// Fallback if FOOTER.ANS doesn't exist
		return
	}

	if len(footerLines) == 0 {
		return
	}

	// Get the actual footer height (number of lines in the footer)
	footerHeight := len(footerLines)
	if footerHeight > 2 {
		// Cap at 2 lines to avoid taking too much screen space
		footerHeight = 2
	}

	// Move cursor to appropriate row based on footer height and reset colors
	startRow := de.config.Height - footerHeight + 1
	de.output.Write([]byte(fmt.Sprintf("\033[%d;1H\033[0m", startRow)))

	// Print the footer (up to 2 lines)
	for i := 0; i < footerHeight && i < len(footerLines); i++ {
		de.output.Write([]byte(footerLines[i]))
		if i < footerHeight-1 {
			// Add newline between footer lines, but not after the last one
			de.output.Write([]byte("\r\n"))
		}
	}
}

type TeeWriter struct {
	writers []io.Writer
}

// NewTeeWriter creates a new TeeWriter
func NewTeeWriter(writers ...io.Writer) *TeeWriter {
	return &TeeWriter{writers: writers}
}

// Write implements io.Writer interface
func (tw *TeeWriter) Write(p []byte) (n int, err error) {
	for _, w := range tw.writers {
		if n, err = w.Write(p); err != nil {
			return
		}
	}
	return len(p), nil
}

type DualModeWriter struct {
	consoleWriter io.Writer // Sysop console (needs CP437->UTF8 conversion)
	bbsWriter     io.Writer // BBS connection (raw CP437)
}

// NewDualModeWriter creates a writer for dual output with proper encoding
func NewDualModeWriter(consoleWriter, bbsWriter io.Writer) *DualModeWriter {
	return &DualModeWriter{
		consoleWriter: consoleWriter,
		bbsWriter:     bbsWriter,
	}
} // Write implements io.Writer interface with encoding conversion
func (dmw *DualModeWriter) Write(p []byte) (n int, err error) {
	// Send raw CP437 to BBS connection (user terminal)
	if dmw.bbsWriter != nil {
		if _, err = dmw.bbsWriter.Write(p); err != nil {
			return
		}
	}

	// Convert CP437 to UTF-8 for sysop console
	if dmw.consoleWriter != nil {
		// Convert CP437 bytes to UTF-8 for proper display on Windows console
		converted := convertCP437ToUTF8(p)
		if _, err = dmw.consoleWriter.Write(converted); err != nil {
			return
		}
	}

	return len(p), nil
}

// convertCP437ToUTF8 converts CP437 encoded bytes to UTF-8
func convertCP437ToUTF8(data []byte) []byte {
	// Use the same charmap.CodePage437 decoder as in processCP437
	decoder := charmap.CodePage437.NewDecoder()
	utf8Data, err := decoder.Bytes(data)
	if err != nil {
		// If conversion fails, return original data
		return data
	}
	return utf8Data
}

// DisplayEngine implements the Displayer interface
type DisplayEngine struct {
	config         DisplayConfig
	themeManager   *ThemeManager
	scrollState    ScrollState
	cache          map[string][]string
	currentContent []string      // Store current content for scrolling re-renders
	output         io.Writer     // Output destination (console, BBS, or both)
	fs             fs.FS         // Embedded filesystem for art files
	stdoutBuf      *bufio.Writer // Buffered writer for Windows console
}

// NewDisplayEngine creates a new display engine
func NewDisplayEngine(config DisplayConfig, embeddedFS fs.FS) *DisplayEngine {
	// Use buffered writer for stdout to ensure proper flushing on Windows
	writer := bufio.NewWriter(os.Stdout)

	return &DisplayEngine{
		config:       config,
		themeManager: NewThemeManager(),
		cache:        make(map[string][]string),
		scrollState: ScrollState{
			CurrentLine:  0,
			TotalLines:   0,
			VisibleLines: config.Height,
		},
		output:    writer,
		stdoutBuf: writer,
		fs:        embeddedFS,
	}
}

// SetBBSConnection configures output to BBS connection only (no sysop console)
func (de *DisplayEngine) SetBBSConnection(bbsConn io.Writer) {
	if bbsConn != nil {
		// Output only to BBS connection (user terminal)
		de.output = bbsConn
		de.stdoutBuf = nil // BBS connection doesn't use stdout buffer
	} else {
		// Fall back to console only with buffered writer
		de.stdoutBuf = bufio.NewWriter(os.Stdout)
		de.output = de.stdoutBuf
	}
}

// Display displays the content of an ANSI file
func (de *DisplayEngine) Display(filePath string, user User) error {
	err := de.DisplayWithOverlay(filePath, user, "")
	// Flush output immediately - critical for Windows 7 console
	de.flushOutput()
	return err
}

// flushOutput flushes buffered output if the writer supports it
func (de *DisplayEngine) flushOutput() {
	// Flush bufio.Writer if present
	if de.stdoutBuf != nil {
		de.stdoutBuf.Flush()
	}

	if flusher, ok := de.output.(interface{ Flush() error }); ok {
		flusher.Flush()
	}

	// Force Windows console to flush
	os.Stdout.Sync()
	os.Stderr.Sync()
}

// DisplayWithOverlay displays the content of an ANSI file with optional overlay text
func (de *DisplayEngine) DisplayWithOverlay(filePath string, user User, overlayText string) error {
	de.output.Write([]byte(Reset)) // Reset text and background colors
	de.flushOutput()               // Ensure reset is sent
	de.ClearScreen()

	// Load and process content
	content, err := de.loadAndProcess(filePath)
	if err != nil {
		// Silently fallback to MISSING.ANS when art file is not found
		missingPath := "art/common/MISSING.ANS"
		var fallbackErr error
		content, fallbackErr = de.loadAndProcess(missingPath)
		if fallbackErr != nil {
			// Only show error if MISSING.ANS itself is missing
			de.output.Write([]byte("Error: Unable to load art. Please contact the Sysop.\r\n"))
			return err
		}
		// Add filename to overlay to show which file was missing
		if overlayText == "" {
			// Strip "art/" prefix for cleaner display
			overlayText = strings.TrimPrefix(filePath, "art/")
		}
	}

	if len(content) == 0 {
		de.output.Write([]byte("Error: The art file is empty or invalid.\r\n"))
		return fmt.Errorf("empty file")
	}

	// Handle scrolling if needed
	if len(content) > de.config.Height && de.config.Scrolling.Enabled {
		de.currentContent = content // Store for scroll re-renders
		return de.renderWithScrolling(content)
	}

	// Normal rendering (content fits on screen or scrolling disabled)
	de.currentContent = nil // Clear stored content
	err = de.renderNormal(content)

	// Add overlay text if provided (bottom right corner)
	if overlayText != "" {
		de.renderOverlayText(overlayText)
	}

	return err
}

// renderOverlayText renders text at the bottom right corner of the screen
func (de *DisplayEngine) renderOverlayText(text string) {
	// Save cursor position
	de.output.Write([]byte("\0337")) // Save cursor position (ESC 7)

	// Position cursor at bottom right
	// Account for text length to position correctly
	row := de.config.Height
	col := de.config.Width - len(text)

	if col < 1 {
		col = 1
	}

	// Move cursor and print text with bright white on black background
	de.output.Write([]byte(fmt.Sprintf("\033[%d;%dH", row, col)))
	de.output.Write([]byte(fmt.Sprintf("\033[97;40m%s\033[0m", text))) // Bright white text on black background

	// Restore cursor position
	de.output.Write([]byte("\0338")) // Restore cursor position (ESC 8)
} // loadAndProcess loads and processes the file content
func (de *DisplayEngine) loadAndProcess(filePath string) ([]string, error) {
	// Check cache first
	if cached, exists := de.cache[filePath]; exists {
		return cached, nil
	}

	// Load file from embedded filesystem
	content, err := fs.ReadFile(de.fs, filePath)
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
	// Reset scroll state for non-scrolling content
	de.scrollState.TotalLines = len(lines)
	de.scrollState.CurrentLine = 0
	de.scrollState.CanScrollUp = false
	de.scrollState.CanScrollDown = false

	// Calculate how many lines to actually display
	linesToDisplay := de.config.Height
	if len(lines) < linesToDisplay {
		linesToDisplay = len(lines)
	}

	for i := 0; i < linesToDisplay; i++ {
		line := lines[i]
		// Last line is the last one we're displaying in the viewport
		isLastLine := i == linesToDisplay-1
		de.printLine(line, isLastLine)
	}

	// Force output flush after rendering all lines
	de.flushOutput()
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

	// Calculate number of lines we'll actually render
	numLines := endLine - startLine
	lineIndex := 0

	for i := startLine; i < endLine; i++ {
		line := lines[i]
		// Last line is the last one in this viewport render
		isLastLine := lineIndex == numLines-1
		de.printLine(line, isLastLine)
		lineIndex++
	}

	// Don't show scroll indicators - removed to save screen space
	de.flushOutput() // Ensure visible lines are sent to terminal

	return nil
}

// ScrollUp scrolls up one line
func (de *DisplayEngine) ScrollUp() error {
	if de.scrollState.CurrentLine > 0 {
		de.scrollState.CurrentLine--
		de.updateScrollState()
		// Re-render with new scroll position
		if de.currentContent != nil {
			return de.renderVisibleLines(de.currentContent)
		}
	}
	return nil
}

// ScrollDown scrolls down one line
func (de *DisplayEngine) ScrollDown() error {
	if de.scrollState.CurrentLine < de.scrollState.TotalLines-de.config.Height {
		de.scrollState.CurrentLine++
		de.updateScrollState()
		// Re-render with new scroll position
		if de.currentContent != nil {
			return de.renderVisibleLines(de.currentContent)
		}
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

// ClearScreen clears the screen
func (de *DisplayEngine) ClearScreen() error {
	de.output.Write([]byte(EraseScreen))
	de.MoveCursor(0, 0)
	de.flushOutput() // Ensure clear screen is sent immediately
	return nil
}

// MoveCursor moves the cursor to the specified position
func (de *DisplayEngine) MoveCursor(x, y int) error {
	de.output.Write([]byte(fmt.Sprintf(Esc+"%d;%df", y, x)))
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

// HideCursor hides the terminal cursor
func (de *DisplayEngine) HideCursor() {
	de.output.Write([]byte(HideCursor))
}

// ShowCursor shows the terminal cursor
func (de *DisplayEngine) ShowCursor() {
	de.output.Write([]byte(ShowCursor))
}

// ANSI escape sequences (extracted from original ansiart.go)
const (
	Esc         = "\u001B["
	EraseScreen = Esc + "2J"
	Reset       = Esc + "0m"
	HideCursor  = Esc + "?25l"
	ShowCursor  = Esc + "?25h"
)

// printLine handles newline behavior per mode
func (de *DisplayEngine) printLine(line string, isLastLine bool) {
	if de.config.Mode == ModeCP437Raw {
		de.output.Write([]byte(line))
		// Only add line break if not the last line to avoid trailing breaks
		if !isLastLine {
			de.output.Write([]byte("\r\n"))
		}
		return
	}

	if isLastLine {
		de.output.Write([]byte(line))
	} else {
		de.output.Write([]byte(line + "\r\n"))
	}
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

// trimLastChar trims the last character from a string
func trimLastChar(s string) string {
	if len(s) > 0 {
		_, size := utf8.DecodeLastRuneInString(s)
		return s[:len(s)-size]
	}
	return s
}
