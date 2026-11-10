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

func TestFit80Columns(t *testing.T) {
	// A line of exactly 80 cells loses its last cell (and keeps its codes).
	line80 := Esc + "31m" + strings.Repeat("x", 80) + Reset
	got := fit80Columns([]string{line80, "short"}, 25)
	if countVisibleChars(got[0]) != 79 || !strings.HasSuffix(got[0], Reset) {
		t.Fatalf("80-cell line: %q", got[0])
	}
	if got[1] != "short" {
		t.Fatalf("short line changed: %q", got[1])
	}
	// Art drawn for autowrap keeps its wrapped rows.
	wrapped := strings.Repeat("y", 160)
	if got := fit80Columns([]string{wrapped, "next"}, 25); countVisibleChars(got[0]) != 159 {
		t.Fatalf("wrapping line cut to %d cells, want 159", countVisibleChars(got[0]))
	}
	// On a screen filled exactly, the bottom row stops at column 79.
	full := make([]string, 25)
	for i := range full {
		full[i] = "row"
	}
	full[24] = strings.Repeat("z", 200)
	if got := fit80Columns(full, 25); countVisibleChars(got[24]) != 79 {
		t.Fatalf("bottom row has %d cells, want 79", countVisibleChars(got[24]))
	}
}

// No bundled art may lose more than the one cell the door has always
// dropped from a line reaching column 80 (bottom row of a full screen aside).
func TestBundledArtKeepsItsContent(t *testing.T) {
	err := fs.WalkDir(embedded.ArtFS, "art", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".ANS") {
			return err
		}
		raw, _ := fs.ReadFile(embedded.ArtFS, p)
		before := splitArt(string(raw))
		after := fit80Columns(before, 25)
		for i := range before {
			if len(before) == 25 && i == 24 {
				continue
			}
			if lost := countVisibleChars(before[i]) - countVisibleChars(after[i]); lost > 1 {
				t.Errorf("%s line %d lost %d characters", p, i+1, lost)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestScrollRedrawsWrappedArt(t *testing.T) {
	rows := []string{strings.Repeat("w", 120)}
	for i := 2; i <= 40; i++ {
		rows = append(rows, fmt.Sprintf("row %02d", i))
	}
	fs := fstest.MapFS{"art/wrap.ans": {Data: []byte(strings.Join(rows, "\r\n"))}}
	var out countingWriter
	de := engineTo(ModeCP437Raw, fs, &out)
	de.Display("art/wrap.ans")
	out.Reset()
	de.ScrollUp() // at the top: nothing happens
	de.ScrollDown()
	if strings.Contains(out.String(), EraseScreen) {
		t.Fatal("window without wrapping lines should repaint in place")
	}
	de.ScrollUp() // back to a window whose first line wraps
	if !strings.Contains(out.String(), EraseScreen) {
		t.Fatal("window with a wrapping line should be redrawn, not repainted per row")
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
		if end := col - 1 + countVisibleChars(parts[i+1]); end > 79 {
			t.Errorf("panel row %s reaches column %d", m[1], end)
		}
	}
}
