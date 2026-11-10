package display

import (
	"bytes"
	"fmt"
	"io/fs"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"unicode/utf8"

	"github.com/robbiew/advent/internal/embedded"
)

func engineTo(mode DisplayMode, artFS fs.FS, w *countingWriter) *DisplayEngine {
	de := NewDisplayEngine(DisplayConfig{Mode: mode, Width: 80, Height: 25}, artFS)
	de.SetOutput(w)
	return de
}

// countingWriter records output and how many writes delivered it.
type countingWriter struct {
	bytes.Buffer
	writes int
}

func (c *countingWriter) Write(p []byte) (int, error) {
	c.writes++
	return c.Buffer.Write(p)
}

func TestTransportPreservesArtEncoding(t *testing.T) {
	for _, mode := range []DisplayMode{ModeCP437Raw, ModeCP437} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			fs := fstest.MapFS{"art/test.ans": {Data: []byte{0xdb, 0xc4, '\r', '\n'}}}
			var out countingWriter
			if err := engineTo(mode, fs, &out).Display("art/test.ans"); err != nil {
				t.Fatal(err)
			}
			want := []byte{0xdb, 0xc4}
			if mode == ModeCP437 {
				want = []byte("█─")
			}
			if !bytes.Contains(out.Bytes(), want) {
				t.Fatalf("missing %x in output %x", want, out.Bytes())
			}
		})
	}
}

// Every bundled art file must reach a local terminal as valid UTF-8. The
// 80-column fixes used to cut multi-byte characters in half.
func TestLocalModeOutputIsValidUTF8(t *testing.T) {
	err := fs.WalkDir(embedded.ArtFS, "art", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".ANS") {
			return err
		}
		var out countingWriter
		if err := engineTo(ModeCP437, embedded.ArtFS, &out).Display(p); err != nil {
			t.Errorf("%s: %v", p, err)
		}
		if !utf8.Valid(out.Bytes()) {
			t.Errorf("%s: local output is not valid UTF-8", p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestScreenIsSentInOneWrite(t *testing.T) {
	var out countingWriter
	if err := engineTo(ModeCP437Raw, embedded.ArtFS, &out).Display("art/2025/WELCOME.ANS"); err != nil {
		t.Fatal(err)
	}
	if out.writes != 1 {
		t.Fatalf("screen took %d writes, want 1", out.writes)
	}
}

// Eleven 2024 days are 25 rows plus a trailing CRLF; they must not be treated
// as 26-row art that scrolls.
func TestTrailingCRLFDoesNotScroll(t *testing.T) {
	var out countingWriter
	de := engineTo(ModeCP437Raw, embedded.ArtFS, &out)
	if err := de.Display("art/2024/10_DEC24.ANS"); err != nil {
		t.Fatal(err)
	}
	if s := de.GetScrollState(); s.TotalLines != 25 || s.CanScrollDown {
		t.Fatalf("scroll state %+v, want 25 rows that fit", s)
	}
}

func TestScrollRepaintsInPlace(t *testing.T) {
	var rows []string
	for i := 1; i <= 40; i++ {
		rows = append(rows, fmt.Sprintf("row %02d", i))
	}
	fs := fstest.MapFS{"art/tall.ans": {Data: []byte(strings.Join(rows, "\r\n"))}}
	var out countingWriter
	de := engineTo(ModeCP437Raw, fs, &out)
	if err := de.Display("art/tall.ans"); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := de.ScrollDown(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if strings.Contains(got, EraseScreen) {
		t.Fatal("scrolling cleared the screen")
	}
	if !strings.Contains(got, "\x1b[1;1H"+Reset+EraseLine+"row 02") || !strings.Contains(got, "row 26") {
		t.Fatalf("scroll did not repaint rows 2-26 in place: %q", got)
	}
}

func TestSplitArtTrimsOnlySauce(t *testing.T) {
	sauce := "SAUCE00" + strings.Repeat(" ", 128-7)
	art := "art mentions COMNT here\r\nLAST"
	if got := splitArt(art + "\x1a" + sauce); strings.Join(got, "\r\n") != art {
		t.Fatalf("got %q, want the art untouched", got)
	}
	if got := splitArt(art + sauce); strings.Join(got, "\r\n") != art {
		t.Fatalf("without an EOF marker the last character was cut: %q", got)
	}
}

func TestToRows(t *testing.T) {
	red := Esc + "31m"
	cases := []struct {
		name  string
		lines []string
		want  []int // cells per row
	}{
		{"short lines stay one row each", []string{"abc", ""}, []int{3, 0}},
		{"exactly 80 cells is one row", []string{strings.Repeat("x", 80)}, []int{80}},
		{"160 cells wrap to two rows", []string{red + strings.Repeat("y", 160) + Reset}, []int{80, 80}},
		{"a 2000-cell stream fills 25 rows", []string{strings.Repeat("z", 2000)}, nil},
		{"cursor-forward counts as cells", []string{"ab" + Esc + "100C" + "cd"}, []int{80, 24}},
	}
	for _, c := range cases {
		rows := toRows(c.lines, 80)
		if c.want == nil {
			if len(rows) != 25 {
				t.Errorf("%s: %d rows", c.name, len(rows))
			}
			continue
		}
		var got []int
		for _, r := range rows {
			got = append(got, cellsIn(r))
		}
		if fmt.Sprint(got) != fmt.Sprint(c.want) {
			t.Errorf("%s: cells per row %v, want %v", c.name, got, c.want)
		}
	}
	// Colour codes stay in the row where they occur.
	rows := toRows([]string{red + strings.Repeat("y", 100)}, 80)
	if !strings.HasPrefix(rows[0], red) || strings.Contains(rows[1], red) {
		t.Errorf("escape codes moved between rows: %q", rows)
	}
}

func TestCursorPositionedArtIsDrawnAsAStream(t *testing.T) {
	art := Esc + "1;1Hfirst" + Esc + "25;80Hx"
	fs := fstest.MapFS{"art/pos.ans": {Data: []byte(art)}}
	var out countingWriter
	de := engineTo(ModeCP437Raw, fs, &out)
	de.Display("art/pos.ans")
	if !strings.Contains(out.String(), Esc+"25;79H") {
		t.Fatalf("bottom-right cursor position not moved to column 79: %q", out.String())
	}
	if de.GetScrollState().CanScrollDown {
		t.Fatal("positioned art should not scroll")
	}
}

// Splitting into rows must keep every cell of every bundled file.
func TestBundledArtKeepsItsContent(t *testing.T) {
	err := fs.WalkDir(embedded.ArtFS, "art", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".ANS") {
			return err
		}
		raw, _ := fs.ReadFile(embedded.ArtFS, p)
		lines := splitArt(string(raw))
		if hasPositioning(lines) {
			t.Errorf("%s uses cursor positioning; check it still draws correctly", p)
			return nil
		}
		before, after := 0, 0
		for _, l := range lines {
			before += cellsIn(l)
		}
		for _, r := range toRows(lines, 80) {
			if n := cellsIn(r); n > 80 {
				t.Errorf("%s: a row has %d cells", p, n)
			}
			after += cellsIn(r)
		}
		if before != after {
			t.Errorf("%s: %d cells before splitting, %d after", p, before, after)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Art drawn for autowrap (four lines filling 25 rows) must not overflow an
// 80x24 terminal: it scrolls by rows instead (Codex review on #8).
func TestWrappedArtScrollsOn80x24(t *testing.T) {
	const door = "art/2025/01_DEC25.ANS"
	de := NewDisplayEngine(DisplayConfig{Mode: ModeCP437Raw, Width: 80, Height: 24}, embedded.ArtFS)
	var out countingWriter
	de.SetOutput(&out)
	if err := de.Display(door); err != nil {
		t.Fatal(err)
	}
	s := de.GetScrollState()
	if s.TotalLines != 25 || !s.CanScrollDown {
		t.Fatalf("scroll state %+v, want 25 rows that scroll", s)
	}
	pos := regexp.MustCompile(`\x1b\[(\d+);1H`)
	for _, m := range pos.FindAllStringSubmatch(out.String(), -1) {
		if row, _ := strconv.Atoi(m[1]); row > 24 {
			t.Fatalf("drew on row %d of a 24-row screen", row)
		}
	}
	out.Reset()
	de.ScrollDown()
	if strings.Contains(out.String(), EraseScreen) {
		t.Fatal("scrolling wrapped art should repaint in place")
	}
}

func TestBottomRightCellStaysEmpty(t *testing.T) {
	fs := fstest.MapFS{"art/full.ans": {Data: []byte(strings.Repeat("#", 80*25))}}
	var out countingWriter
	de := engineTo(ModeCP437Raw, fs, &out)
	de.Display("art/full.ans")
	last := out.String()[strings.LastIndex(out.String(), Esc+"25;1H"):]
	if n := strings.Count(last, "#"); n != 79 {
		t.Fatalf("bottom row drew %d cells, want 79", n)
	}
}

// Panels must stay inside 79 columns, whatever they're given.
func TestPanelFitsTheScreen(t *testing.T) {
	var out countingWriter
	de := engineTo(ModeCP437Raw, fstest.MapFS{}, &out)
	long := strings.Repeat("x", 200)
	de.ShowPanel(long, []string{"short", long})
	rows := regexp.MustCompile(`\x1b\[(\d+);(\d+)H`)
	parts := rows.Split(out.String(), -1)
	cols := rows.FindAllStringSubmatch(out.String(), -1)
	if len(cols) < 4 {
		t.Fatalf("panel drew %d rows", len(cols))
	}
	for i, m := range cols {
		col, _ := strconv.Atoi(m[2])
		if end := col - 1 + cellsIn(parts[i+1]); end > 79 {
			t.Errorf("panel row %s reaches column %d", m[1], end)
		}
	}
}
