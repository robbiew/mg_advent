package navigation

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

// ScreenType represents different screens in the application
type ScreenType int

const (
	ScreenWelcome ScreenType = iota
	ScreenDay
	ScreenComeback
	ScreenYearSelect
	ScreenExit
)

// Direction represents navigation direction
type Direction int

const (
	DirNone Direction = iota // 0 = no direction
	DirLeft
	DirRight
	DirUp
	DirDown
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

// Navigator handles navigation logic
type Navigator struct {
	baseArtDir string
}

// NewNavigator creates a new navigator
func NewNavigator(baseArtDir string) *Navigator {
	return &Navigator{
		baseArtDir: baseArtDir,
	}
}

// GetAvailableYears returns list of available years with art
func (n *Navigator) GetAvailableYears() ([]int, error) {
	var years []int

	entries, err := os.ReadDir(n.baseArtDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read art directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			year, err := strconv.Atoi(entry.Name())
			if err == nil && year >= 2023 && year <= 2030 { // Reasonable year range
				years = append(years, year)
			}
		}
	}

	// Sort years descending (newest first)
	for i := 0; i < len(years)-1; i++ {
		for j := i + 1; j < len(years); j++ {
			if years[i] < years[j] {
				years[i], years[j] = years[j], years[i]
			}
		}
	}

	return years, nil
}

// Navigate handles navigation based on current state and direction
func (n *Navigator) Navigate(direction Direction, currentState State) (newState State, artPath string, err error) {
	newState = currentState

	switch currentState.Screen {
	case ScreenWelcome:
		return n.navigateFromWelcome(direction, currentState)
	case ScreenDay:
		return n.navigateFromDay(direction, currentState)
	case ScreenComeback:
		return n.navigateFromComeback(direction, currentState)
	case ScreenYearSelect:
		return n.navigateFromYearSelect(direction, currentState)
	default:
		return currentState, "", fmt.Errorf("unknown screen type: %d", currentState.Screen)
	}
}

// navigateFromWelcome handles navigation from welcome screen
func (n *Navigator) navigateFromWelcome(direction Direction, state State) (State, string, error) {
	switch direction {
	case DirRight:
		// Move to current day's art
		state.Screen = ScreenDay
		artPath := n.getDayArtPath(state.CurrentYear, state.CurrentDay)
		return state, artPath, nil
	case DirLeft:
		// Stay on welcome screen
		return state, "", nil
	default:
		// For now, only right arrow from welcome
		return state, "", nil
	}
}

// navigateFromDay handles navigation from day screen
func (n *Navigator) navigateFromDay(direction Direction, state State) (State, string, error) {
	logrus.WithFields(logrus.Fields{
		"direction":  direction,
		"currentDay": state.CurrentDay,
		"maxDay":     state.MaxDay,
	}).Debug("navigateFromDay called")

	switch direction {
	case DirRight:
		if state.CurrentDay < state.MaxDay {
			state.CurrentDay++
			artPath := n.getDayArtPath(state.CurrentYear, state.CurrentDay)
			logrus.WithField("newDay", state.CurrentDay).Debug("Moving to next day")
			return state, artPath, nil
		} else if state.MaxDay < 25 {
			// Move to comeback screen
			state.Screen = ScreenComeback
			artPath := n.getComebackArtPath(state.CurrentYear)
			logrus.Debug("Moving to comeback screen")
			return state, artPath, nil
		} else {
			// Stay on last day
			logrus.Debug("Already at last day, staying")
			return state, "", nil
		}
	case DirLeft:
		if state.CurrentDay > 1 {
			state.CurrentDay--
			artPath := n.getDayArtPath(state.CurrentYear, state.CurrentDay)
			logrus.WithField("newDay", state.CurrentDay).Debug("Moving to previous day")
			return state, artPath, nil
		} else {
			// Move to welcome screen
			state.Screen = ScreenWelcome
			artPath := n.getWelcomeArtPath(state.CurrentYear)
			logrus.Debug("Moving to welcome screen")
			return state, artPath, nil
		}
	default:
		logrus.WithField("direction", direction).Debug("Unhandled direction from day screen")
		return state, "", nil
	}
}

// navigateFromComeback handles navigation from comeback screen
func (n *Navigator) navigateFromComeback(direction Direction, state State) (State, string, error) {
	switch direction {
	case DirLeft:
		// Move back to last available day
		state.Screen = ScreenDay
		state.CurrentDay = state.MaxDay
		artPath := n.getDayArtPath(state.CurrentYear, state.CurrentDay)
		return state, artPath, nil
	case DirRight:
		// Stay on comeback screen
		return state, "", nil
	default:
		return state, "", nil
	}
}

// navigateFromYearSelect handles navigation from year selection screen
func (n *Navigator) navigateFromYearSelect(direction Direction, state State) (State, string, error) {
	// Year selection navigation would be implemented here
	// For now, return to welcome
	state.Screen = ScreenWelcome
	artPath := n.getWelcomeArtPath(state.CurrentYear)
	return state, artPath, nil
}

// SetYear changes the current year
func (n *Navigator) SetYear(year int) error {
	// Validate year exists
	years, err := n.GetAvailableYears()
	if err != nil {
		return err
	}

	for _, y := range years {
		if y == year {
			return nil
		}
	}

	return fmt.Errorf("year %d not available", year)
}

// GetInitialState returns the initial application state
func (n *Navigator) GetInitialState() (State, error) {
	years, err := n.GetAvailableYears()
	if err != nil {
		return State{}, err
	}

	if len(years) == 0 {
		return State{}, fmt.Errorf("no art years available")
	}

	// Use current year if available, otherwise latest
	currentYear := time.Now().Year()
	var selectedYear int
	for _, year := range years {
		if year == currentYear {
			selectedYear = year
			break
		}
	}
	if selectedYear == 0 {
		selectedYear = years[0] // Use latest available
	}

	// Calculate max day for the year
	maxDay := n.calculateMaxDay(selectedYear)

	// Current day is today, but capped at maxDay
	currentDay := time.Now().YearDay()
	if selectedYear != time.Now().Year() {
		currentDay = 1 // Default to first day for past/future years
	}
	if currentDay > maxDay {
		currentDay = maxDay
	}
	if currentDay > 25 {
		currentDay = 25
	}

	state := State{
		CurrentYear:    selectedYear,
		CurrentDay:     currentDay,
		Screen:         ScreenWelcome,
		MaxDay:         maxDay,
		AvailableYears: years,
	}

	return state, nil
}

// calculateMaxDay calculates the maximum available day for a year
func (n *Navigator) calculateMaxDay(year int) int {
	now := time.Now()
	if year < now.Year() {
		return 25 // Past years have all days
	} else if year > now.Year() {
		return 1 // Future years only have day 1
	} else {
		// Current year
		maxDay := now.YearDay()
		if maxDay > 25 {
			maxDay = 25
		}
		return maxDay
	}
}

// getDayArtPath returns the path to a day's art file
func (n *Navigator) getDayArtPath(year, day int) string {
	yearDir := filepath.Join(n.baseArtDir, strconv.Itoa(year))
	fileName := fmt.Sprintf("%d_DEC%s.ANS", day, strconv.Itoa(year)[2:])
	return filepath.Join(yearDir, fileName)
}

// getWelcomeArtPath returns the path to the welcome art file
func (n *Navigator) getWelcomeArtPath(year int) string {
	yearDir := filepath.Join(n.baseArtDir, strconv.Itoa(year))
	return filepath.Join(yearDir, "WELCOME.ANS")
}

// getComebackArtPath returns the path to the comeback art file
func (n *Navigator) getComebackArtPath(year int) string {
	yearDir := filepath.Join(n.baseArtDir, strconv.Itoa(year))
	return filepath.Join(yearDir, "COMEBACK.ANS")
}

// getExitArtPath returns the path to the exit art file
func (n *Navigator) getExitArtPath(year int) string {
	yearDir := filepath.Join(n.baseArtDir, strconv.Itoa(year))
	return filepath.Join(yearDir, "GOODBYE.ANS")
}

// ValidateState validates that the current state is consistent
func (n *Navigator) ValidateState(state State) error {
	// Check year is available
	years, err := n.GetAvailableYears()
	if err != nil {
		return err
	}

	yearValid := false
	for _, y := range years {
		if y == state.CurrentYear {
			yearValid = true
			break
		}
	}
	if !yearValid {
		return fmt.Errorf("current year %d is not available", state.CurrentYear)
	}

	// Check day is valid
	if state.CurrentDay < 1 || state.CurrentDay > 25 {
		return fmt.Errorf("current day %d is out of range", state.CurrentDay)
	}

	// Check max day is reasonable
	if state.MaxDay < 1 || state.MaxDay > 25 {
		return fmt.Errorf("max day %d is out of range", state.MaxDay)
	}

	return nil
}

// LogState logs the current navigation state
func (n *Navigator) LogState(state State) {
	logrus.WithFields(logrus.Fields{
		"year":   state.CurrentYear,
		"day":    state.CurrentDay,
		"screen": state.Screen,
		"maxDay": state.MaxDay,
	}).Debug("Navigation state")
}
