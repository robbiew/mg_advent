package navigation

import (
	"errors"
	"slices"
	"testing"
	"testing/fstest"
	"time"
)

// artFS installs year directories 2023-2026 plus a non-year directory.
func artFS() fstest.MapFS {
	fs := fstest.MapFS{"art/common/MISSING.ANS": {}}
	for _, y := range []string{"2023", "2024", "2025", "2026"} {
		fs["art/"+y+"/WELCOME.ANS"] = &fstest.MapFile{}
	}
	return fs
}

func navAt(t *testing.T, date string) *Navigator {
	t.Helper()
	day, err := time.Parse("2006-01-02", date)
	if err != nil {
		t.Fatal(err)
	}
	n := NewNavigator(artFS(), "art")
	n.SetClock(func() time.Time { return day })
	return n
}

func TestMaxDay(t *testing.T) {
	tests := []struct {
		date string
		year int
		want int
	}{
		{"2026-11-10", 2025, 25}, // past season: fully open off-season
		{"2026-11-10", 2023, 25},
		{"2026-11-10", 2026, 0},  // current year before December: locked
		{"2026-12-05", 2026, 5},  // live season: one day at a time
		{"2026-12-05", 2024, 25}, // past seasons stay fully open in December
		{"2026-12-31", 2026, 25}, // after Christmas: capped at 25
		{"2026-12-05", 2027, 0},  // future year: locked
		{"2027-01-02", 2026, 25}, // last season becomes an archive in January
	}
	for _, tc := range tests {
		if got := navAt(t, tc.date).MaxDay(tc.year); got != tc.want {
			t.Errorf("on %s MaxDay(%d) = %d, want %d", tc.date, tc.year, got, tc.want)
		}
	}
}

func TestOffSeasonOpensArchive(t *testing.T) {
	n := navAt(t, "2026-11-10")
	state, err := n.GetInitialState()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(state.AvailableYears, []int{2023, 2024, 2025}) {
		t.Fatalf("visible years = %v; 2026 must stay hidden until December", state.AvailableYears)
	}
	if state.CurrentYear != 2025 || state.MaxDay != 25 {
		t.Fatalf("initial state = %+v, want newest past season fully open", state)
	}
	if _, live := n.LiveYear(); live {
		t.Fatal("no season should be live in November")
	}
}

func TestLiveSeason(t *testing.T) {
	n := navAt(t, "2026-12-03")
	state, err := n.GetInitialState()
	if err != nil {
		t.Fatal(err)
	}
	if state.CurrentYear != 2026 || state.MaxDay != 3 {
		t.Fatalf("initial state = %+v, want 2026 open to day 3", state)
	}
	if year, live := n.LiveYear(); !live || year != 2026 {
		t.Fatalf("LiveYear = %d, %v", year, live)
	}
}

func TestDecemberWithoutNewArtIsArchiveOnly(t *testing.T) {
	fs := artFS()
	delete(fs, "art/2026/WELCOME.ANS")
	n := NewNavigator(fs, "art")
	n.SetClock(func() time.Time { return time.Date(2026, 12, 3, 0, 0, 0, 0, time.Local) })
	if _, live := n.LiveYear(); live {
		t.Fatal("no season is live until that year's art is installed")
	}
	state, err := n.GetInitialState()
	if err != nil || state.CurrentYear != 2025 || state.MaxDay != 25 {
		t.Fatalf("state = %+v, err = %v; want 2025 fully open", state, err)
	}
}

func TestNothingOpenYet(t *testing.T) {
	n := NewNavigator(fstest.MapFS{"art/2026/WELCOME.ANS": {}}, "art")
	n.SetClock(func() time.Time { return time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local) })
	if _, err := n.GetInitialState(); !errors.Is(err, ErrNoYearsAvailable) {
		t.Fatalf("err = %v, want ErrNoYearsAvailable", err)
	}
}

func TestUnlockAll(t *testing.T) {
	n := navAt(t, "2026-06-01")
	n.SetDisableDateCheck(true)
	years, _ := n.VisibleYears()
	if !slices.Equal(years, []int{2023, 2024, 2025, 2026}) || n.MaxDay(2026) != 25 {
		t.Fatalf("unlock all: years %v, MaxDay(2026) %d", years, n.MaxDay(2026))
	}
}

func TestSelectYearRecomputesDays(t *testing.T) {
	n := navAt(t, "2026-12-03")
	state, err := n.GetInitialState()
	if err != nil {
		t.Fatal(err)
	}
	state, err = n.SelectYearByIndex(1, state) // 2023
	if err != nil || state.CurrentYear != 2023 || state.MaxDay != 25 {
		t.Fatalf("select 2023: %+v, %v", state, err)
	}
	state.CurrentDay, state.Screen = 20, ScreenDay
	state, err = n.SelectYear(2026, state)
	if err != nil || state.MaxDay != 3 || state.CurrentDay != 1 || state.Screen != ScreenWelcome {
		t.Fatalf("back to live year: %+v, %v", state, err)
	}
	if _, err := n.SelectYear(2031, state); err == nil {
		t.Fatal("selecting an unavailable year should fail")
	}
}

func TestComebackOnlyDuringLiveSeason(t *testing.T) {
	n := navAt(t, "2026-12-03")
	live := State{CurrentYear: 2026, CurrentDay: 3, MaxDay: 3, Screen: ScreenDay}
	if s := n.Navigate(DirRight, live); s.Screen != ScreenComeback {
		t.Fatalf("past today's door should show COMEBACK, got %v", s.Screen)
	}
	archive := State{CurrentYear: 2024, CurrentDay: 25, MaxDay: 25, Screen: ScreenDay}
	if s := n.Navigate(DirRight, archive); s.Screen != ScreenDay || s.CurrentDay != 25 {
		t.Fatalf("archive day 25 should stay put, got %+v", s)
	}
}

func TestIgnoresNonYearDirectories(t *testing.T) {
	fs := artFS()
	fs["art/2024/2025/WELCOME.ANS"] = &fstest.MapFile{}
	fs["art/12345/x"] = &fstest.MapFile{}
	years, err := NewNavigator(fs, "art").GetAvailableYears()
	if err != nil || !slices.Equal(years, []int{2023, 2024, 2025, 2026}) {
		t.Fatalf("years = %v, %v", years, err)
	}
}

func TestStepYear(t *testing.T) {
	n := navAt(t, "2026-12-03")
	state, err := n.GetInitialState() // 2026, the live season
	if err != nil {
		t.Fatal(err)
	}
	if s := n.StepYear(1, state); s.CurrentYear != 2026 {
		t.Fatalf("stepping past the newest year moved to %d", s.CurrentYear)
	}
	state = n.StepYear(-1, state)
	if state.CurrentYear != 2025 || state.MaxDay != 25 || state.Screen != ScreenWelcome {
		t.Fatalf("previous year: %+v", state)
	}
	for i := 0; i < 5; i++ {
		state = n.StepYear(-1, state)
	}
	if state.CurrentYear != 2023 {
		t.Fatalf("should stop at the oldest year, got %d", state.CurrentYear)
	}
}
