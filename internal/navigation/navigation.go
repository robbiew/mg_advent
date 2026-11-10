package navigation

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"slices"
	"strconv"
	"time"
)

// ErrNoYearsAvailable means no art year can be opened today: every installed
// year is still in the future.
var ErrNoYearsAvailable = errors.New("no art years available yet")

// ScreenType represents different screens in the application
type ScreenType int

const (
	ScreenWelcome ScreenType = iota
	ScreenDay
	ScreenComeback
	ScreenInfo
	ScreenMembers
)

// Direction represents navigation direction
type Direction int

const (
	DirNone Direction = iota // 0 = no direction
	DirLeft
	DirRight
	DirPageUp
	DirPageDown
	DirHome
	DirEnd
)

// State represents the current navigation state
type State struct {
	CurrentYear    int
	CurrentDay     int
	Screen         ScreenType
	MaxDay         int
	AvailableYears []int
}

// Navigator handles navigation logic and decides which years and days are
// open. Past seasons are open in full all year round; the current year opens
// one day at a time during December; years still to come stay hidden.
type Navigator struct {
	baseArtDir string
	fs         fs.FS
	now        func() time.Time
	unlockAll  bool
}

// NewNavigator creates a new navigator
func NewNavigator(embeddedFS fs.FS, baseArtDir string) *Navigator {
	return &Navigator{
		baseArtDir: baseArtDir,
		fs:         embeddedFS,
		now:        time.Now,
	}
}

// SetClock replaces the calendar clock, e.g. to simulate a date with
// -debug-date. Session timers are unaffected.
func (n *Navigator) SetClock(now func() time.Time) {
	n.now = now
}

// SetDisableDateCheck unlocks every installed year and day when disable is true.
func (n *Navigator) SetDisableDateCheck(disable bool) {
	n.unlockAll = disable
}

// GetAvailableYears returns every installed art year, oldest first, whether or
// not it can be opened today. Index-based selection relies on this order:
// 1 = oldest, 2 = next, and so on.
func (n *Navigator) GetAvailableYears() ([]int, error) {
	entries, err := fs.ReadDir(n.fs, n.baseArtDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read art directory: %w", err)
	}

	var years []int
	for _, entry := range entries {
		if !entry.IsDir() || len(entry.Name()) != 4 {
			continue
		}
		year, err := strconv.Atoi(entry.Name())
		if err == nil && year >= 2000 {
			years = append(years, year)
		}
	}
	slices.Sort(years)
	return years, nil
}

// MaxDay reports how many days of year can be opened today: 25 for a past
// season, today's date (capped at 25) for the current year in December, and
// 0 for a year that is not open yet.
func (n *Navigator) MaxDay(year int) int {
	if n.unlockAll {
		return 25
	}
	now := n.now()
	switch {
	case year < now.Year():
		return 25
	case year == now.Year() && now.Month() == time.December:
		return min(now.Day(), 25)
	default:
		return 0
	}
}

// VisibleYears returns the installed years that can be opened today, oldest
// first.
func (n *Navigator) VisibleYears() ([]int, error) {
	years, err := n.GetAvailableYears()
	if err != nil {
		return nil, err
	}
	var visible []int
	for _, year := range years {
		if n.MaxDay(year) > 0 {
			visible = append(visible, year)
		}
	}
	return visible, nil
}

// LiveYear returns the year whose calendar is running today: it is December
// and that year's art is installed. With the date check disabled the newest
// installed year counts as live.
func (n *Navigator) LiveYear() (int, bool) {
	years, err := n.VisibleYears()
	if err != nil || len(years) == 0 {
		return 0, false
	}
	latest := years[len(years)-1]
	if n.unlockAll {
		return latest, true
	}
	now := n.now()
	if now.Month() == time.December && latest == now.Year() {
		return latest, true
	}
	return 0, false
}

// Navigate moves from the current screen in direction. Keys that don't
// apply to the screen leave the state unchanged.
func (n *Navigator) Navigate(direction Direction, state State) State {
	switch state.Screen {
	case ScreenWelcome:
		if direction == DirRight {
			state.Screen = ScreenDay
			state.CurrentDay = max(1, min(state.CurrentDay, state.MaxDay))
		}
	case ScreenDay:
		switch direction {
		case DirRight:
			if state.CurrentDay < state.MaxDay {
				state.CurrentDay++
			} else if state.MaxDay < 25 {
				// Today's door is the last one open: invite the caller back tomorrow.
				state.Screen = ScreenComeback
			}
		case DirLeft:
			if state.CurrentDay > 1 {
				state.CurrentDay--
			} else {
				state.Screen = ScreenWelcome
			}
		}
	case ScreenComeback:
		if direction == DirLeft {
			state.Screen = ScreenDay
			state.CurrentDay = state.MaxDay
		}
	}
	return state
}

// SelectYear switches to year's welcome screen, recomputing which of its days
// are open. year must be one of state.AvailableYears.
func (n *Navigator) SelectYear(year int, state State) (State, error) {
	if !slices.Contains(state.AvailableYears, year) {
		return state, fmt.Errorf("year %d not available", year)
	}
	state.CurrentYear = year
	state.MaxDay = n.MaxDay(year)
	state.CurrentDay = 1
	state.Screen = ScreenWelcome
	return state, nil
}

// SelectYearByIndex selects a year by its index (1-based) from available years
// Years are sorted ascending (oldest first), so index 1 = oldest year, index 2 = next, etc.
func (n *Navigator) SelectYearByIndex(index int, currentState State) (State, error) {
	if index < 1 || index > len(currentState.AvailableYears) {
		return currentState, fmt.Errorf("invalid year index: %d", index)
	}
	return n.SelectYear(currentState.AvailableYears[index-1], currentState)
}

// GetInitialState opens the newest year that can be viewed today: the live
// season in December, otherwise the most recent past season. It returns
// ErrNoYearsAvailable when nothing is open yet.
func (n *Navigator) GetInitialState() (State, error) {
	years, err := n.VisibleYears()
	if err != nil {
		return State{}, err
	}
	if len(years) == 0 {
		return State{}, ErrNoYearsAvailable
	}

	selectedYear := years[len(years)-1]
	slog.Info("selected initial year", "visible_years", years, "year", selectedYear)

	return State{
		CurrentYear:    selectedYear,
		CurrentDay:     1,
		Screen:         ScreenWelcome,
		MaxDay:         n.MaxDay(selectedYear),
		AvailableYears: years,
	}, nil
}
