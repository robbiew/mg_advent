// Package display renders the door's CP437 ANSI art.
//
// Art is kept as CP437 bytes throughout, so column arithmetic (one byte per
// cell) is always right. Local mode converts to UTF-8 only as bytes leave the
// engine. All output is buffered and sent once per frame: over a BBS link a
// screen is a few large writes instead of hundreds of tiny ones.
package display

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"strings"

	gd "github.com/robbiew/godoors"
	"golang.org/x/text/encoding/charmap"
)

// ANSI control sequences.
const (
	Esc         = "\x1b["
	EraseScreen = Esc + "2J"
	Home        = Esc + "1;1H"
	Reset       = Esc + "0m"
	EraseLine   = Esc + "K"
	HideCursor  = Esc + "?25l"
	ShowCursor  = Esc + "?25h"
	// BlinkMode makes SGR 5 blink instead of selecting a bright background
	// (CTerm/SyncTERM DECRST 33, the terminal default). The art uses blink.
	BlinkMode = Esc + "?33l"
)

const (
	footerPath       = "art/common/FOOTER.ANS"
	missingPath      = "art/common/MISSING.ANS"
	maxFooterRows    = 2
	outputBufferSize = 64 << 10 // larger than any single screen of art
)

// DisplayEngine renders art files to the caller's terminal.
type DisplayEngine struct {
	config  DisplayConfig
	fs      fs.FS
	cache   map[string][]string
	out     *bufio.Writer
	scroll  ScrollState
	content []string // art taller than the screen, scrolled by ScrollUp/Down
}

// NewDisplayEngine creates an engine writing to the GoDoors output stream.
func NewDisplayEngine(config DisplayConfig, artFS fs.FS) *DisplayEngine {
	de := &DisplayEngine{
		config: config,
		fs:     artFS,
		cache:  make(map[string][]string),
		scroll: ScrollState{VisibleLines: config.Height},
	}
	de.SetOutput(gd.Output())
	return de
}

// SetOutput directs output to w, such as the BBS connection. In local mode
// the CP437 art is converted to UTF-8 on the way out.
func (de *DisplayEngine) SetOutput(w io.Writer) {
	if de.config.Mode == ModeCP437 {
		w = utf8Writer{w}
	}
	de.out = bufio.NewWriterSize(w, outputBufferSize)
}

// utf8Writer converts CP437 to UTF-8. CP437 is a single-byte encoding, so any
// chunk converts on its own; escape sequences are ASCII and pass unchanged.
type utf8Writer struct{ w io.Writer }

func (u utf8Writer) Write(p []byte) (int, error) {
	converted, err := charmap.CodePage437.NewDecoder().Bytes(p)
	if err != nil {
		converted = p
	}
	if _, err := u.w.Write(converted); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (de *DisplayEngine) flush() { _ = de.out.Flush() }

// Display clears the screen and shows an art file. A missing file shows
// MISSING.ANS labelled with the name that was not found.
func (de *DisplayEngine) Display(filePath string) error {
	defer de.flush()
	de.out.WriteString(Reset)
	de.clear()

	content, err := de.loadAndProcess(filePath)
	label := ""
	if err != nil {
		var fallbackErr error
		if content, fallbackErr = de.loadAndProcess(missingPath); fallbackErr != nil {
			de.out.WriteString("Error: Unable to load art. Please contact the Sysop.\r\n")
			return err
		}
		label = strings.TrimPrefix(filePath, "art/")
	}
	if len(content) == 0 {
		de.out.WriteString("Error: The art file is empty or invalid.\r\n")
		return fmt.Errorf("empty art file: %s", filePath)
	}

	rows := min(len(content), de.config.Height)
	de.scroll = ScrollState{TotalLines: len(content), VisibleLines: de.config.Height}
	de.content = nil
	if len(content) > de.config.Height {
		de.content = content
	}
	de.updateScrollState()
	de.drawRows(content, 0, rows, false)

	if label != "" {
		de.renderLabel(label)
	}
	return nil
}

// renderLabel prints text in the bottom-right corner.
func (de *DisplayEngine) renderLabel(text string) {
	col := max(de.config.Width-len(text), 1)
	fmt.Fprintf(de.out, "\x1b7"+Esc+"%d;%dH"+Esc+"97;40m%s"+Reset+"\x1b8", de.config.Height, col, text)
}

// drawRows draws rows lines starting at lines[start] from the top of the
// screen. A fresh draw (screen just cleared) lets the lines flow with CRLF. A
// repaint positions and erases each row first, so scrolling updates in place
// without blanking the screen. Each row that doesn't follow on from the line
// above it starts with the colours the art above has left in effect.
func (de *DisplayEngine) drawRows(lines []string, start, rows int, repaint bool) {
	start = max(0, min(start, len(lines)))
	end := min(start+rows, len(lines))
	var colours sgrState
	for _, line := range lines[:start] {
		colours.apply(line)
	}
	for i, line := range lines[start:end] {
		switch {
		case repaint:
			fmt.Fprintf(de.out, Esc+"%d;1H"+Reset+EraseLine+"%s", i+1, colours.sequence())
		case i > 0:
			de.out.WriteString("\r\n")
		default:
			de.out.WriteString(colours.sequence())
		}
		de.out.WriteString(line)
		colours.apply(line)
	}
	if repaint {
		for i := end - start; i < rows; i++ {
			fmt.Fprintf(de.out, Esc+"%d;1H"+Reset+EraseLine, i+1)
		}
	}
}

// LoadAnsiLines loads an art file as display-ready lines.
func (de *DisplayEngine) LoadAnsiLines(filePath string) ([]string, error) {
	return de.loadAndProcess(filePath)
}

// loadAndProcess reads an art file, strips SAUCE metadata and trailing blank
// lines, and applies the 80-column fixes. Results are cached.
func (de *DisplayEngine) loadAndProcess(filePath string) ([]string, error) {
	if cached, ok := de.cache[filePath]; ok {
		return cached, nil
	}
	raw, err := fs.ReadFile(de.fs, filePath)
	if err != nil {
		return nil, err
	}
	lines := splitArt(string(raw))
	if de.config.Width == 80 {
		lines = fit80Columns(lines, de.config.Height)
	}
	de.cache[filePath] = lines
	return lines, nil
}

// splitArt strips SAUCE metadata and splits art into lines, dropping the
// empty line(s) a trailing CRLF would otherwise add.
func splitArt(s string) []string {
	lines := strings.Split(gd.TrimStringFromSauce(s), "\r\n")
	for len(lines) > 1 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// fit80Columns keeps art from pushing the screen around on an 80-column
// terminal. Much art is drawn for the terminal's own wrap at column 80: one
// line of 160 characters fills two rows. So lines are never cut to fit.
// Instead, as the door always has, a line reaching 80 characters loses its
// last character (an 80-character line followed by CRLF would otherwise skip
// a row on terminals without a last-column flag), and on a screen filled
// exactly, the bottom row stops at column 79 so the screen can't scroll.
func fit80Columns(lines []string, height int) []string {
	out := make([]string, len(lines))
	if len(lines) == 1 {
		// Cursor-positioned art: one stream with no line breaks.
		line := lines[0]
		if n := countVisibleChars(line); n >= 80 {
			line = truncateVisible(line, n-1)
		}
		line = fix80ColumnCursorPositioning(line, height)
		if height == 25 {
			line = truncateVisible(line, 80*25-1)
		}
		out[0] = line
		return out
	}
	for i, line := range lines {
		if n := countVisibleChars(line); n >= 80 {
			line = truncateVisible(line, n-1)
		}
		if len(lines) == height && i == height-1 {
			line = truncateVisible(line, 79)
		}
		out[i] = line
	}
	return out
}

// wraps reports whether any of lines fills more than one screen row.
func wraps(lines []string) bool {
	for _, l := range lines {
		if countVisibleChars(l) >= 80 {
			return true
		}
	}
	return false
}

// escapeLen returns the length of the escape sequence starting at s[i]
// (s[i] == ESC): a CSI sequence through its final byte, or ESC plus one byte.
func escapeLen(s string, i int) int {
	j := i + 1
	if j < len(s) && s[j] == '[' {
		for j++; j < len(s); j++ {
			if s[j] >= 0x40 && s[j] <= 0x7E {
				return j + 1 - i
			}
		}
		return len(s) - i
	}
	return min(j+1, len(s)) - i
}

// countVisibleChars counts the characters in s that occupy a cell, skipping
// escape sequences. Art is CP437, so each byte is one cell.
func countVisibleChars(s string) int {
	count := 0
	for i := 0; i < len(s); {
		if s[i] == '\x1b' {
			i += escapeLen(s, i)
			continue
		}
		count++
		i++
	}
	return count
}

// truncateVisible drops visible characters after the first limit, keeping
// every escape sequence so colours and modes still take effect.
func truncateVisible(s string, limit int) string {
	if countVisibleChars(s) <= limit {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	visible := 0
	for i := 0; i < len(s); {
		if s[i] == '\x1b' {
			n := escapeLen(s, i)
			b.WriteString(s[i : i+n])
			i += n
			continue
		}
		if visible < limit {
			b.WriteByte(s[i])
		}
		visible++
		i++
	}
	return b.String()
}

// fix80ColumnCursorPositioning moves any cursor positioning to row height,
// column 80 (the bottom-right cell) to column 79, so drawing there cannot
// scroll the screen.
func fix80ColumnCursorPositioning(content string, height int) string {
	var result strings.Builder
	result.Grow(len(content))
	for i := 0; i < len(content); {
		if content[i] != '\x1b' || i+1 >= len(content) || content[i+1] != '[' {
			result.WriteByte(content[i])
			i++
			continue
		}
		seqStart := i
		i += 2
		paramStart := i
		for i < len(content) && (content[i] >= '0' && content[i] <= '9' || content[i] == ';') {
			i++
		}
		if i < len(content) && (content[i] == 'H' || content[i] == 'f') {
			var row, col int
			if n, _ := fmt.Sscanf(content[paramStart:i], "%d;%d", &row, &col); n == 2 && row == height && col == 80 {
				fmt.Fprintf(&result, Esc+"%d;79%c", row, content[i])
				i++
				continue
			}
			i++
		}
		result.WriteString(content[seqStart:i])
	}
	return result.String()
}

// footer returns the menu bar shown under scrollable screens (at most two rows).
func (de *DisplayEngine) footer() []string {
	lines, err := de.loadAndProcess(footerPath)
	if err != nil || len(lines) == 0 {
		return nil
	}
	return lines[:min(len(lines), maxFooterRows)]
}

// contentRows is the number of rows above the footer.
func (de *DisplayEngine) contentRows() int {
	return de.config.Height - max(len(de.footer()), 1)
}

// SetScrollState records the position of a footer screen (Info, Members).
func (de *DisplayEngine) SetScrollState(currentLine, totalLines int) {
	de.scroll = ScrollState{CurrentLine: currentLine, TotalLines: totalLines, VisibleLines: de.contentRows()}
	de.updateScrollState()
}

// clampScroll limits pos to the positions that keep the window full.
func clampScroll(pos, total, rows int) int {
	return max(0, min(pos, total-rows))
}

// RenderScrollable clears the screen and draws a window of lines above the footer.
func (de *DisplayEngine) RenderScrollable(lines []string, scrollPos int) error {
	if len(lines) == 0 {
		return nil
	}
	defer de.flush()
	rows := de.contentRows()
	de.clear()
	de.drawRows(lines, clampScroll(scrollPos, len(lines), rows), rows, false)
	if footer := de.footer(); len(footer) > 0 {
		fmt.Fprintf(de.out, Esc+"%d;1H"+Reset+"%s", de.config.Height-len(footer)+1, strings.Join(footer, "\r\n"))
	}
	return nil
}

// RenderScrollableContentOnly repaints the window above the footer in place.
func (de *DisplayEngine) RenderScrollableContentOnly(lines []string, scrollPos int) error {
	if len(lines) == 0 {
		return nil
	}
	rows := de.contentRows()
	pos := clampScroll(scrollPos, len(lines), rows)
	if wraps(lines[pos:min(pos+rows, len(lines))]) {
		return de.RenderScrollable(lines, pos) // see scrollBy
	}
	defer de.flush()
	de.drawRows(lines, pos, rows, true)
	return nil
}

// ScrollUp scrolls tall art up one line.
func (de *DisplayEngine) ScrollUp() error { return de.scrollBy(-1) }

// ScrollDown scrolls tall art down one line.
func (de *DisplayEngine) ScrollDown() error { return de.scrollBy(1) }

func (de *DisplayEngine) scrollBy(delta int) error {
	if de.content == nil {
		return nil
	}
	pos := clampScroll(de.scroll.CurrentLine+delta, len(de.content), de.config.Height)
	if pos == de.scroll.CurrentLine {
		return nil
	}
	de.scroll.CurrentLine = pos
	de.updateScrollState()
	defer de.flush()
	window := de.content[pos:min(pos+de.config.Height, len(de.content))]
	if wraps(window) {
		// Rows can't be repainted one per line when a line wraps onto the
		// next row; redraw the window from the top instead.
		de.clear()
		de.drawRows(de.content, pos, de.config.Height, false)
		return nil
	}
	de.drawRows(de.content, pos, de.config.Height, true)
	return nil
}

// GetScrollState returns the current scroll state.
func (de *DisplayEngine) GetScrollState() ScrollState {
	return de.scroll
}

func (de *DisplayEngine) updateScrollState() {
	de.scroll.CanScrollUp = de.scroll.CurrentLine > 0
	de.scroll.CanScrollDown = de.scroll.CurrentLine < de.scroll.TotalLines-de.scroll.VisibleLines
}

func (de *DisplayEngine) clear() {
	de.out.WriteString(EraseScreen + Home)
}

// ClearScreen clears the screen and homes the cursor.
func (de *DisplayEngine) ClearScreen() {
	de.clear()
	de.flush()
}

// HideCursor hides the terminal cursor.
func (de *DisplayEngine) HideCursor() {
	de.out.WriteString(HideCursor)
	de.flush()
}

// ShowCursor shows the terminal cursor.
func (de *DisplayEngine) ShowCursor() {
	de.out.WriteString(ShowCursor)
	de.flush()
}

// EnableBlinkMode makes blinking art blink rather than show bright
// backgrounds. That is the terminal default, so nothing needs restoring at
// exit. Skipped with NoIce for terminals that mishandle the sequence.
func (de *DisplayEngine) EnableBlinkMode() {
	if !de.config.NoIce {
		de.out.WriteString(BlinkMode)
		de.flush()
	}
}

// ResetAttributes resets colours, e.g. before returning to the BBS.
func (de *DisplayEngine) ResetAttributes() {
	de.out.WriteString(Reset)
	de.flush()
}

// ShowLabel prints a short note in the bottom-right corner over the current
// screen, e.g. a countdown or the digits of a door number being typed.
func (de *DisplayEngine) ShowLabel(text string) {
	defer de.flush()
	de.renderLabel(text)
}

// ShowPanel clears the screen and draws title and lines in a centred box,
// with a "press any key" hint. Lines are CP437 text without escapes; long
// ones are cut to fit the screen.
func (de *DisplayEngine) ShowPanel(title string, lines []string) {
	defer de.flush()
	de.out.WriteString(Reset)
	de.clear()

	// CP437 double-line box: ╔ ═ ╗ ║ ╚ ╝, with the title between ╡ and ╞.
	const (
		tl, tr, bl, br, h, v = "\xc9", "\xbb", "\xc8", "\xbc", "\xcd", "\xba"
		titleL, titleR       = "\xb5 ", " \xc6"
		hint                 = " press any key "
	)
	maxInner := min(de.config.Width, 80) - 3 // the box never reaches column 80
	title = title[:min(len(title), maxInner-len(titleL)-len(titleR)-2)]
	inner := max(len(title)+len(titleL)+len(titleR)+2, len(hint)+2)
	for _, l := range lines {
		inner = max(inner, len(l)+2)
	}
	inner = min(inner, maxInner)

	rows := len(lines) + 2 // a blank row above and below the text
	top := max((de.config.Height-rows-2)/2, 0) + 1
	left := max((de.config.Width-inner-2)/2, 0) + 1
	border := Esc + "0;1;36m" // bright cyan
	text := Esc + "0;1;37m"   // bright white
	row := func(n int, s string) { fmt.Fprintf(de.out, Esc+"%d;%dH%s", top+n, left, s) }
	body := func(s string) string {
		s = " " + s[:min(len(s), inner-2)]
		return border + v + text + s + strings.Repeat(" ", inner-len(s)) + border + v
	}

	heading := titleL + title + titleR
	row(0, border+tl+h+heading+strings.Repeat(h, inner-1-len(heading))+tr)
	row(1, body(""))
	for i, l := range lines {
		row(2+i, body(l))
	}
	row(rows, body(""))
	row(rows+1, border+bl+strings.Repeat(h, inner-len(hint)-1)+hint+h+br+Reset)
}

// ShowMessage clears the screen and prints a one-line notice in its centre.
func (de *DisplayEngine) ShowMessage(text string) {
	defer de.flush()
	de.clear()
	row := max(de.config.Height/2, 1)
	col := max((de.config.Width-len(text))/2+1, 1)
	fmt.Fprintf(de.out, Reset+Esc+"%d;%dH%s", row, col, text)
}
