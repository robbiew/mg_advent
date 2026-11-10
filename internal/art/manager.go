// Package art maps screens to art files in the embedded art tree:
// art/<year>/ holds a season's screens and days, art/common/ the fallbacks.
package art

import (
	"fmt"
	"io/fs"
	"path" // embedded FS paths always use forward slashes
	"strconv"
)

// Manager resolves art paths.
type Manager struct {
	baseDir string
	fs      fs.FS
}

// NewManager creates a manager over the art tree rooted at baseDir in artFS.
func NewManager(artFS fs.FS, baseDir string) *Manager {
	return &Manager{baseDir: baseDir, fs: artFS}
}

// GetPath returns the art file for a screen of year ("welcome", "info",
// "members", "goodbye", "comeback", "day", "missing" or "notyet"; day is
// only used for "day"). Year screens prefer the year's own art and fall back
// to art/common, so a new year works while its pack is still arriving. A
// path that does not exist is shown as MISSING.ANS by the display engine.
func (m *Manager) GetPath(year, day int, screenType string) string {
	yearDir := path.Join(m.baseDir, strconv.Itoa(year))
	commonDir := path.Join(m.baseDir, "common")

	switch screenType {
	case "welcome":
		return m.yearOrCommon(yearDir, commonDir, "WELCOME.ANS")
	case "info":
		return m.yearOrCommon(yearDir, commonDir, "INFOFILE.ANS")
	case "members":
		return m.yearOrCommon(yearDir, commonDir, "MEMBERS.ANS")
	case "goodbye", "exit":
		return m.yearOrCommon(yearDir, commonDir, "GOODBYE.ANS")
	case "comeback":
		return m.yearOrCommon(yearDir, commonDir, "COMEBACK.ANS")
	case "day":
		return m.dayPath(yearDir, year, day)
	case "missing":
		return path.Join(commonDir, "MISSING.ANS")
	case "notyet":
		return path.Join(commonDir, "NOTYET.ANS")
	default:
		return ""
	}
}

// DayFileNames returns the two accepted names for a day's art, the year's own
// convention first: 2025 onward pad the day (01_DEC25.ANS), earlier years do
// not (1_DEC24.ANS).
func DayFileNames(year, day int) [2]string {
	suffix := fmt.Sprintf("%02d", year%100)
	padded := fmt.Sprintf("%02d_DEC%s.ANS", day, suffix)
	plain := fmt.Sprintf("%d_DEC%s.ANS", day, suffix)
	if year >= 2025 {
		return [2]string{padded, plain}
	}
	return [2]string{plain, padded}
}

func (m *Manager) dayPath(yearDir string, year, day int) string {
	names := DayFileNames(year, day)
	for _, name := range names {
		p := path.Join(yearDir, name)
		if _, err := fs.Stat(m.fs, p); err == nil {
			return p
		}
	}
	return path.Join(yearDir, names[0])
}

// yearOrCommon returns the year's copy of name, else the common copy, else
// the year path (so MISSING.ANS is labelled with the year's file name).
func (m *Manager) yearOrCommon(yearDir, commonDir, name string) string {
	yearPath := path.Join(yearDir, name)
	for _, p := range []string{yearPath, path.Join(commonDir, name)} {
		if _, err := fs.Stat(m.fs, p); err == nil {
			return p
		}
	}
	return yearPath
}
