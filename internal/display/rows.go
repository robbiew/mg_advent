package display

import (
	"fmt"
	"strconv"
	"strings"
)

// artWidth is the width ANSI art is drawn for. Art often relies on the
// terminal wrapping at this column: one line of 160 cells fills two rows.
const artWidth = 80

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

// cursorForward returns n for a cursor-forward sequence ESC[nC, which ANSI
// editors use to skip over blank cells.
func cursorForward(seq string) (int, bool) {
	if len(seq) < 3 || seq[1] != '[' || seq[len(seq)-1] != 'C' {
		return 0, false
	}
	params := seq[2 : len(seq)-1]
	if params == "" {
		return 1, true
	}
	n, err := strconv.Atoi(params)
	if err != nil || n < 1 {
		return 1, true
	}
	return n, true
}

// hasPositioning reports whether art moves the cursor anywhere other than
// forward (absolute positions, up/down/back, save/restore). Such art draws
// itself and can't be split into rows by counting cells.
func hasPositioning(lines []string) bool {
	for _, line := range lines {
		for i := 0; i < len(line); i++ {
			if line[i] != '\x1b' {
				continue
			}
			n := escapeLen(line, i)
			seq := line[i : i+n]
			i += n - 1
			if len(seq) == 2 && (seq[1] == '7' || seq[1] == '8') {
				return true
			}
			if len(seq) >= 3 && seq[1] == '[' && strings.IndexByte("HfABDGdsu", seq[len(seq)-1]) >= 0 {
				return true
			}
		}
	}
	return false
}

// cellsIn counts the screen cells s covers: each printable byte (art is
// CP437, one byte per cell) and each cell skipped by ESC[nC.
func cellsIn(s string) int {
	cells := 0
	for i := 0; i < len(s); {
		if s[i] == '\x1b' {
			n := escapeLen(s, i)
			if fwd, ok := cursorForward(s[i : i+n]); ok {
				cells += fwd
			}
			i += n
			continue
		}
		cells++
		i++
	}
	return cells
}

// truncateCells keeps the first limit cells of s, plus every escape sequence
// (so colours still take effect); a cursor-forward that crosses the limit is
// shortened.
func truncateCells(s string, limit int) string {
	if cellsIn(s) <= limit {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	cells := 0
	for i := 0; i < len(s); {
		if s[i] == '\x1b' {
			n := escapeLen(s, i)
			seq := s[i : i+n]
			i += n
			if fwd, ok := cursorForward(seq); ok {
				if keep := min(fwd, limit-cells); keep > 0 {
					fmt.Fprintf(&b, Esc+"%dC", keep)
				}
				cells += fwd
				continue
			}
			b.WriteString(seq)
			continue
		}
		if cells < limit {
			b.WriteByte(s[i])
		}
		cells++
		i++
	}
	return b.String()
}

// toRows splits art lines into the screen rows they fill at the given width,
// the way the terminal's wrap would: a line of 160 cells becomes two rows,
// and a line of exactly width cells stays one row. Cursor-forward runs are
// split across the row boundary. Escape sequences stay with the row they
// occur in.
func toRows(lines []string, width int) []string {
	var rows []string
	for _, line := range lines {
		var b strings.Builder
		cells := 0
		endRow := func() {
			rows = append(rows, b.String())
			b.Reset()
			cells = 0
		}
		for i := 0; i < len(line); {
			if line[i] == '\x1b' {
				n := escapeLen(line, i)
				seq := line[i : i+n]
				i += n
				if fwd, ok := cursorForward(seq); ok {
					for fwd > 0 {
						if cells == width {
							endRow()
						}
						step := min(fwd, width-cells)
						fmt.Fprintf(&b, Esc+"%dC", step)
						cells += step
						fwd -= step
					}
					continue
				}
				b.WriteString(seq)
				continue
			}
			if cells == width {
				endRow()
			}
			b.WriteByte(line[i])
			cells++
			i++
		}
		endRow()
	}
	return rows
}

// legacyFit applies the door's original 80-column fixes to cursor-positioned
// art, which is drawn as a stream: a line reaching 80 cells loses its last
// cell, and single-stream art never draws in the bottom-right cell.
func legacyFit(lines []string, width, height int) []string {
	if width != artWidth {
		return lines
	}
	out := make([]string, len(lines))
	for i, line := range lines {
		if n := cellsIn(line); n >= artWidth {
			line = truncateCells(line, n-1)
		}
		if len(lines) == 1 {
			line = fix80ColumnCursorPositioning(line, height)
		}
		out[i] = line
	}
	return out
}
