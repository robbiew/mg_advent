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
	cache   map[string]artFile
	out     *bufio.Writer
	scroll  ScrollState
	content []string // art taller than the screen, scrolled by ScrollUp/Down
}

// NewDisplayEngine creates an engine writing to the GoDoors output stream.
func NewDisplayEngine(config DisplayConfig, artFS fs.FS) *DisplayEngine {
	de := &DisplayEngine{
		config: config,
		fs:     artFS,
		cache:  make(map[string]artFile),
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

// artFile is a loaded art file.
type artFile struct {
	// rows has one string per screen row. For cursor-positioned art it holds
	// the file's lines instead, drawn as a stream.
	rows       []string
	positioned bool
}

// Display clears the screen and shows an art file. A missing file shows
// MISSING.ANS labelled with the name that was not found.
func (de *DisplayEngine) Display(filePath string) error {
	defer de.flush()
	de.out.WriteString(Reset)
	de.clear()

	a, err := de.load(filePath)
	label := ""
	if err != nil {
		var fallbackErr error
		if a, fallbackErr = de.load(missingPath); fallbackErr != nil {
			de.out.WriteString("Error: Unable to load art. Please contact the Sysop.\r\n")
			return err
		}
		label = strings.TrimPrefix(filePath, "art/")
	}
	if len(a.rows) == 0 {
		de.out.WriteString("Error: The art file is empty or invalid.\r\n")
		return fmt.Errorf("empty art file: %s", filePath)
	}

	de.content = nil
	if a.positioned {
		// Art that positions the cursor draws itself; it can't scroll by rows.
		de.scroll = ScrollState{TotalLines: de.config.Height, VisibleLines: de.config.Height}
		de.out.WriteString(strings.Join(a.rows, "\r\n"))
	} else {
		de.scroll = ScrollState{TotalLines: len(a.rows), VisibleLines: de.config.Height}
		if len(a.rows) > de.config.Height {
			de.content = a.rows
		}
		de.drawRows(a.rows, 0, de.config.Height, false)
	}
	de.updateScrollState()

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

// drawRows draws up to count screen rows starting at rows[start], each moved
// to its own row from the top, so a full 80-cell row never depends on how
// the terminal wraps. A repaint erases each row first, for scrolling in
// place; a fresh draw follows a clear. Rows start with the colours the rows
// above left in effect. A row reaching the bottom-right cell stops one cell
// short, so the screen can't scroll.
func (de *DisplayEngine) drawRows(rows []string, start, count int, repaint bool) {
	start = max(0, min(start, len(rows)))
	end := min(start+count, len(rows))
	var colours sgrState
	for _, row := range rows[:start] {
		colours.apply(row)
	}
	for i, row := range rows[start:end] {
		fmt.Fprintf(de.out, Esc+"%d;1H", i+1)
		if repaint {
			de.out.WriteString(Reset + EraseLine + colours.sequence())
		} else if i == 0 {
			de.out.WriteString(colours.sequence())
		}
		if i+1 == de.config.Height && cellsIn(row) >= de.config.Width {
			row = truncateCells(row, de.config.Width-1)
		}
		de.out.WriteString(row)
		colours.apply(row)
	}
	if repaint {
		for i := end - start; i < count; i++ {
			fmt.Fprintf(de.out, Esc+"%d;1H"+Reset+EraseLine, i+1)
		}
	}
}

// LoadAnsiLines returns an art file's screen rows.
func (de *DisplayEngine) LoadAnsiLines(filePath string) ([]string, error) {
	a, err := de.load(filePath)
	return a.rows, err
}

// load reads an art file, strips SAUCE metadata and trailing blank lines,
// and splits it into the screen rows it fills. Results are cached.
func (de *DisplayEngine) load(filePath string) (artFile, error) {
	if cached, ok := de.cache[filePath]; ok {
		return cached, nil
	}
	raw, err := fs.ReadFile(de.fs, filePath)
	if err != nil {
		return artFile{}, err
	}
	lines := splitArt(string(raw))
	a := artFile{rows: toRows(lines, artWidth)}
	if hasPositioning(lines) {
		a = artFile{rows: legacyFit(lines, de.config.Width, de.config.Height), positioned: true}
	}
	de.cache[filePath] = a
	return a, nil
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
	a, err := de.load(footerPath)
	if err != nil || len(a.rows) == 0 {
		return nil
	}
	return a.rows[:min(len(a.rows), maxFooterRows)]
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
	defer de.flush()
	rows := de.contentRows()
	de.drawRows(lines, clampScroll(scrollPos, len(lines), rows), rows, true)
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
