package display

import (
	"strings"
	"testing"

	"github.com/robbiew/advent/internal/embedded"
)

func TestSGRStateTracksColours(t *testing.T) {
	var s sgrState
	s.apply(Esc + "1;31m" + "red" + Esc + "44mon blue")
	if got := s.sequence(); got != Esc+"0;1;31;44m" {
		t.Fatalf("state = %q", got)
	}
	s.apply("plain text keeps it")
	s.apply(Esc + "22;39mnormal fg" + Esc + "5m")
	if got := s.sequence(); got != Esc+"0;5;44m" {
		t.Fatalf("after 22;39;5: %q", got)
	}
	s.apply(Esc + "m")
	if got := s.sequence(); got != "" {
		t.Fatalf("ESC[m should reset, got %q", got)
	}
	s.apply(Esc + "38;5;200;1m")
	if got := s.sequence(); got != Esc+"0;1m" {
		t.Fatalf("extended colour arguments misread: %q", got)
	}
}

// On an 80x24 terminal 25-row days scroll; repainted rows must keep the
// colours set on the rows above the window.
func TestScrollKeepsColoursFromAbove(t *testing.T) {
	const day = "art/2024/13_DEC24.ANS" // relies on colours carried between lines
	de := NewDisplayEngine(DisplayConfig{Mode: ModeCP437Raw, Width: 80, Height: 24}, embedded.ArtFS)
	var out countingWriter
	de.SetOutput(&out)
	if err := de.Display(day); err != nil {
		t.Fatal(err)
	}
	if !de.GetScrollState().CanScrollDown {
		t.Fatal("25-row art should scroll on a 24-row terminal")
	}
	lines, _ := de.LoadAnsiLines(day)
	var want sgrState
	want.apply(lines[0])
	if want.sequence() == "" {
		t.Fatal("test art no longer carries colour past line 1; pick another day")
	}
	out.Reset()
	de.ScrollDown()
	row1 := Esc + "1;1H" + Reset + EraseLine + want.sequence() + lines[1]
	if !strings.Contains(out.String(), row1) {
		t.Fatal("row 1 repainted without the colours set by the line above it")
	}
}
