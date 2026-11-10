package art

import (
	"testing"
	"testing/fstest"
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
