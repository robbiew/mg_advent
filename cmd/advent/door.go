package main

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/robbiew/advent/internal/art"
	"github.com/robbiew/advent/internal/display"
	"github.com/robbiew/advent/internal/input"
	"github.com/robbiew/advent/internal/navigation"
	"github.com/robbiew/advent/internal/session"
)

// helpLines is the '?' screen. Keep it to 75 columns.
var helpLines = []string{
	"Welcome   Enter or Right   open the calendar (today's door in December)",
	"          1-9              pick a year (1 = oldest)",
	"          [ ]  PgUp PgDn   previous / next year",
	"          I  M             info file, member list",
	"",
	"Doors     Left Right       previous / next door",
	"          Home End         first / last open door",
	"          PgUp PgDn        back / ahead five doors",
	"          1-25             type a door number to jump to it",
	"          Up Down          scroll art taller than the screen",
	"",
	"Info      Up Down PgUp PgDn Home End   scroll",
	"",
	"Anywhere  A                who made this art",
	"          S                slideshow through the year's doors",
	"          ?                this help",
	"          Q  Esc           back, or quit from the newest year",
}

// door is one caller's interactive session in the calendar.
type door struct {
	de    *display.DisplayEngine
	art   *art.Manager
	nav   *navigation.Navigator
	in    *input.InputHandler
	sess  *session.Manager
	state navigation.State

	shown     string      // art path on screen; "" forces a redraw
	view      *scrollView // the open Info/Members screen, if any
	entry     string      // digits typed on a day screen
	slideshow bool
	label     string // note to show over the next screen drawn
}

func runMainLoop(displayEngine *display.DisplayEngine, artManager *art.Manager,
	navigator *navigation.Navigator, inputHandler *input.InputHandler,
	sess *session.Manager, state navigation.State) {
	d := &door{de: displayEngine, art: artManager, nav: navigator, in: inputHandler, sess: sess, state: state}
	d.run()
}

func (d *door) run() {
	for {
		d.draw()

		var within time.Duration
		if d.slideshow {
			within = *slideEvery
		}
		ch, key, err := readKeyWithin(d.in, d.sess, within)
		var over errSessionOver
		switch {
		case errors.Is(err, errNoKey):
			d.nextSlide()
			continue
		case errors.As(err, &over):
			slog.Info("session limit reached", "reason", over.reason)
			showSessionOver(d.de, over.reason)
			return
		case err != nil:
			slog.Info("input closed; ending door session", "err", err)
			return
		}

		if d.slideshow {
			d.slideshow = false // any key stops it
			d.shown = ""        // redraw without the slideshow note
			continue
		}
		if d.handleKey(ch, key) {
			return
		}
	}
}

// artPath is the file for the current screen.
func (d *door) artPath() string {
	s := d.state
	switch s.Screen {
	case navigation.ScreenWelcome:
		return d.art.GetPath(s.CurrentYear, 0, "welcome")
	case navigation.ScreenDay:
		return d.art.GetPath(s.CurrentYear, s.CurrentDay, "day")
	case navigation.ScreenComeback:
		return d.art.GetPath(s.CurrentYear, 0, "comeback")
	case navigation.ScreenInfo:
		return d.art.GetPath(s.CurrentYear, 0, "info")
	case navigation.ScreenMembers:
		return d.art.GetPath(s.CurrentYear, 0, "members")
	}
	return ""
}

// draw shows the current screen if it changed, plus any pending note.
func (d *door) draw() {
	path := d.artPath()
	if path == "" || path == d.shown {
		return
	}
	slog.Debug("displaying art", "path", path, "screen", d.state.Screen, "day", d.state.CurrentDay)
	if d.state.Screen == navigation.ScreenInfo || d.state.Screen == navigation.ScreenMembers {
		d.view = openScrollView(d.de, path)
		d.view.render(d.de)
	} else {
		d.view = nil
		if err := d.de.Display(path); err != nil {
			slog.Error("failed to display art", "path", path, "err", err)
		}
	}
	d.shown = path

	label := d.label
	d.label = ""
	if label == "" && d.state.Screen == navigation.ScreenWelcome {
		label = d.countdown()
	}
	if label != "" {
		d.de.ShowLabel(label)
	}
}

// redraw repaints the current screen after a panel covered it.
func (d *door) redraw() {
	if d.view != nil {
		d.view.render(d.de)
		return
	}
	d.shown = ""
}

// countdown is the off-season note on welcome screens.
func (d *door) countdown() string {
	year, days, ok := d.nav.NextOpening()
	switch {
	case !ok:
		return ""
	case days == 1:
		return fmt.Sprintf("The %d calendar opens tomorrow!", year)
	default:
		return fmt.Sprintf("The %d calendar opens in %d days", year, days)
	}
}

// handleKey acts on one key and reports whether the caller is leaving.
func (d *door) handleKey(ch rune, key input.Key) bool {
	if d.handleDayEntry(ch, key) {
		return false
	}

	switch ch {
	case '?':
		d.showPanel("Advent Calendar keys", helpLines)
		return false
	case 'a', 'A':
		d.showCredits()
		return false
	case 's', 'S':
		if d.state.Screen != navigation.ScreenInfo && d.state.Screen != navigation.ScreenMembers {
			d.startSlideshow()
			return false
		}
	}

	if d.view != nil && d.view.handleKey(d.de, key) {
		return false
	}

	onYearMenu := d.state.Screen == navigation.ScreenWelcome || d.state.Screen == navigation.ScreenComeback

	if ch == 'q' || ch == 'Q' || key == input.KeyEsc {
		latestYear := d.state.AvailableYears[len(d.state.AvailableYears)-1]
		switch {
		case onYearMenu && d.state.CurrentYear == latestYear:
			slog.Info("user requested exit")
			d.de.Display(d.art.GetPath(d.state.CurrentYear, 0, "goodbye"))
			waitForKey(d.in, exitScreenWait)
			return true
		case d.view != nil:
			d.state.Screen = navigation.ScreenWelcome
		case d.state.CurrentYear != latestYear:
			if s, err := d.nav.SelectYear(latestYear, d.state); err == nil {
				d.state = s
			}
		default:
			d.state.Screen = navigation.ScreenWelcome
		}
		return false
	}

	if onYearMenu {
		switch {
		case ch >= '1' && ch <= '9':
			if s, err := d.nav.SelectYearByIndex(int(ch-'0'), d.state); err == nil {
				slog.Info("year selected", "year", s.CurrentYear)
				d.state = s
			}
			return false
		case ch == 'i' || ch == 'I':
			d.state.Screen = navigation.ScreenInfo
			return false
		case ch == 'm' || ch == 'M':
			d.state.Screen = navigation.ScreenMembers
			return false
		case ch == '[' || key == input.KeyPageUp:
			d.state = d.nav.StepYear(-1, d.state)
			return false
		case ch == ']' || key == input.KeyPageDown:
			d.state = d.nav.StepYear(1, d.state)
			return false
		}
	}

	if d.state.Screen == navigation.ScreenDay {
		scroll := d.de.GetScrollState()
		if key == input.KeyArrowUp && scroll.CanScrollUp {
			d.de.ScrollUp()
			return false
		}
		if key == input.KeyArrowDown && scroll.CanScrollDown {
			d.de.ScrollDown()
			return false
		}
	}

	var direction navigation.Direction
	switch key {
	case input.KeyEnter:
		if onYearMenu {
			direction = navigation.DirRight // RETURN opens the calendar like arrow right
		}
	case input.KeyArrowRight:
		direction = navigation.DirRight
	case input.KeyArrowLeft:
		direction = navigation.DirLeft
	case input.KeyPageUp:
		direction = navigation.DirPageUp
	case input.KeyPageDown:
		direction = navigation.DirPageDown
	case input.KeyHome:
		direction = navigation.DirHome
	case input.KeyEnd:
		direction = navigation.DirEnd
	}
	if direction != navigation.DirNone {
		d.state = d.nav.Navigate(direction, d.state)
	}
	return false
}

// handleDayEntry collects a door number typed on a day screen and reports
// whether it used the key. One digit 3-9, or any two digits, jump at once;
// 1, 2 or 0 wait for a second digit or Enter. Esc cancels; any other key
// cancels and is then handled as usual.
func (d *door) handleDayEntry(ch rune, key input.Key) bool {
	if d.state.Screen != navigation.ScreenDay {
		d.entry = ""
		return false
	}
	switch {
	case ch >= '0' && ch <= '9':
		d.entry += string(ch)
		if len(d.entry) == 2 || ch >= '3' && len(d.entry) == 1 {
			d.commitDayEntry()
		} else {
			d.de.ShowLabel("Go to door: " + d.entry + "_")
		}
		return true
	case d.entry == "":
		return false
	case key == input.KeyEnter:
		d.commitDayEntry()
		return true
	case key == input.KeyBackspace:
		d.entry = d.entry[:len(d.entry)-1]
		if d.entry == "" {
			d.shown = "" // repaint over the prompt
		} else {
			d.de.ShowLabel("Go to door: " + d.entry + "_")
		}
		return true
	default:
		d.entry = ""
		d.shown = "" // repaint over the prompt
		return key == input.KeyEsc
	}
}

func (d *door) commitDayEntry() {
	var day int
	fmt.Sscanf(d.entry, "%d", &day)
	d.entry = ""
	if s, ok := d.nav.JumpToDay(day, d.state); ok {
		d.state = s
		d.shown = "" // repaint even when jumping to the door already shown
		return
	}
	d.shown = ""
	d.label = fmt.Sprintf("Door %d is not open", day)
}

func (d *door) showPanel(title string, lines []string) {
	d.de.ShowPanel(title, lines)
	if _, _, err := readKey(d.in, d.sess); err != nil {
		return // the main loop sees the closed input or expiry next
	}
	d.redraw()
}

// showCredits shows the SAUCE title, artist and group of the art on screen.
func (d *door) showCredits() {
	file := strings.TrimPrefix(d.shown, "art/")
	c, ok := d.art.Credits(d.shown)
	if !ok {
		d.showPanel("About this art", []string{file, "", "No credits are recorded in this file."})
		return
	}
	lines := []string{}
	add := func(label, value string) {
		if value != "" {
			lines = append(lines, label+value)
		}
	}
	add("Title:   ", c.Title)
	add("Artist:  ", c.Artist)
	add("Group:   ", c.Group)
	if !c.Date.IsZero() {
		add("Date:    ", c.Date.Format("2 January 2006"))
	}
	add("File:    ", file)
	d.showPanel("About this art", lines)
}

// startSlideshow shows the year's open doors one after another.
func (d *door) startSlideshow() {
	if d.state.MaxDay < 1 {
		return
	}
	d.slideshow = true
	note := fmt.Sprintf("Slideshow: a new door every %s - any key stops", *slideEvery)
	if d.state.Screen != navigation.ScreenDay {
		d.state.Screen = navigation.ScreenDay
		d.state.CurrentDay = 1
		d.label = note
		return
	}
	d.de.ShowLabel(note)
}

func (d *door) nextSlide() {
	if d.state.CurrentDay >= d.state.MaxDay {
		d.state.CurrentDay = 1
	} else {
		d.state.CurrentDay++
	}
	d.state.Screen = navigation.ScreenDay
	if *localMode {
		d.sess.Touch() // a sysop's console slideshow doesn't idle out
	}
}

// scrollView is a screen with a footer (Info, Members) whose art scrolls
// under the footer one line or one page at a time.
type scrollView struct {
	lines []string
	pos   int
}

func openScrollView(displayEngine *display.DisplayEngine, artPath string) *scrollView {
	lines, err := displayEngine.LoadAnsiLines(artPath)
	if err != nil {
		lines = []string{fmt.Sprintf("[Unable to load %s]", artPath)}
	}
	return &scrollView{lines: lines}
}

func (v *scrollView) render(displayEngine *display.DisplayEngine) {
	displayEngine.SetScrollState(v.pos, len(v.lines))
	displayEngine.RenderScrollable(v.lines, v.pos)
}

// handleKey scrolls for arrow/page keys and reports whether it used the key.
func (v *scrollView) handleKey(displayEngine *display.DisplayEngine, key input.Key) bool {
	rows := displayEngine.GetScrollState().VisibleLines
	last := max(len(v.lines)-rows, 0)
	pos := v.pos
	switch key {
	case input.KeyArrowUp:
		pos--
	case input.KeyArrowDown:
		pos++
	case input.KeyPageUp:
		pos -= rows
	case input.KeyPageDown:
		pos += rows
	case input.KeyHome:
		pos = 0
	case input.KeyEnd:
		pos = last
	default:
		return false
	}
	pos = max(0, min(pos, last))
	if pos == v.pos {
		return key != input.KeyArrowUp && key != input.KeyArrowDown
	}
	v.pos = pos
	displayEngine.RenderScrollableContentOnly(v.lines, v.pos)
	return true
}
