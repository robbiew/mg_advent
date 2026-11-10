package art

import (
	"testing"
	"testing/fstest"

	"github.com/robbiew/advent/internal/embedded"
)

func TestYearScreensFallBackToCommon(t *testing.T) {
	fs := fstest.MapFS{
		"art/common/GOODBYE.ANS": {},
		"art/2026/WELCOME.ANS":   {},
		"art/2026/01_DEC26.ANS":  {},
		"art/2024/1_DEC24.ANS":   {},
	}
	m := NewManager(fs, "art")
	tests := []struct {
		year, day int
		screen    string
		want      string
	}{
		{2026, 0, "welcome", "art/2026/WELCOME.ANS"},
		{2026, 0, "goodbye", "art/common/GOODBYE.ANS"}, // falls back to common
		{2026, 0, "comeback", "art/2026/COMEBACK.ANS"}, // neither: MISSING.ANS shows this name
		{2026, 1, "day", "art/2026/01_DEC26.ANS"},
		{2024, 1, "day", "art/2024/1_DEC24.ANS"},
		{2026, 2, "day", "art/2026/02_DEC26.ANS"},
	}
	for _, tc := range tests {
		if got := m.GetPath(tc.year, tc.day, tc.screen); got != tc.want {
			t.Errorf("GetPath(%d, %d, %q) = %q, want %q", tc.year, tc.day, tc.screen, got, tc.want)
		}
	}
}

func TestReadCredits(t *testing.T) {
	rec := make([]byte, 128)
	for i := range rec {
		rec[i] = ' '
	}
	copy(rec, "SAUCE00")
	copy(rec[7:], "Door 13")
	copy(rec[42:], "Artist")
	copy(rec[62:], "MiSTiGRiS")
	copy(rec[82:], "20251213")
	c, ok := ReadCredits(append([]byte("art\x1a"), rec...))
	if !ok || c.Title != "Door 13" || c.Artist != "Artist" || c.Group != "MiSTiGRiS" || c.Date.Day() != 13 {
		t.Fatalf("credits = %+v, %v", c, ok)
	}
	if _, ok := ReadCredits([]byte("no sauce here")); ok {
		t.Fatal("file without SAUCE reported credits")
	}
}

func TestBundledArtHasCredits(t *testing.T) {
	m := NewManager(embedded.ArtFS, "art")
	for _, p := range []string{"art/2023/1_DEC23.ANS", "art/2024/13_DEC24.ANS", "art/2025/01_DEC25.ANS"} {
		if c, ok := m.Credits(p); !ok || (c.Title == "" && c.Artist == "") {
			t.Errorf("%s: no credits (%+v)", p, c)
		}
	}
}
