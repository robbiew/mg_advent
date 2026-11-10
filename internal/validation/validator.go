// Package validation checks the art tree and the caller's terminal before a
// session starts.
package validation

import (
	"fmt"
	"io/fs"
	"log/slog"
	"path" // embedded FS paths always use forward slashes
	"strconv"

	"github.com/robbiew/advent/internal/art"
)

// Validator checks the art tree rooted at baseArtDir.
type Validator struct {
	baseArtDir string
	fs         fs.FS
}

// NewValidator creates a validator over the art tree in artFS.
func NewValidator(artFS fs.FS, baseArtDir string) *Validator {
	return &Validator{baseArtDir: baseArtDir, fs: artFS}
}

// ValidateArtFiles checks the art needed to open year. Only the common
// fallback screens are required; a year's own WELCOME/COMEBACK/GOODBYE and
// daily files are optional (common art or MISSING.ANS stands in), so a new
// year can be installed while its pack is still arriving. Gaps are logged.
func (v *Validator) ValidateArtFiles(year int) error {
	yearDir := path.Join(v.baseArtDir, strconv.Itoa(year))
	if _, err := fs.Stat(v.fs, yearDir); err != nil {
		return fmt.Errorf("art directory for year %d does not exist", year)
	}

	commonDir := path.Join(v.baseArtDir, "common")
	var missingCommon []string
	for _, name := range []string{"MISSING.ANS", "NOTYET.ANS"} {
		if !v.exists(path.Join(commonDir, name)) {
			missingCommon = append(missingCommon, name)
		}
	}
	if len(missingCommon) > 0 {
		return fmt.Errorf("missing required common art files: %v", missingCommon)
	}

	var missingScreens []string
	for _, name := range []string{"WELCOME.ANS", "COMEBACK.ANS", "GOODBYE.ANS"} {
		if !v.exists(path.Join(yearDir, name)) {
			missingScreens = append(missingScreens, name)
		}
	}
	var missingDays []int
	for day := 1; day <= 25; day++ {
		names := art.DayFileNames(year, day)
		if !v.exists(path.Join(yearDir, names[0])) && !v.exists(path.Join(yearDir, names[1])) {
			missingDays = append(missingDays, day)
		}
	}
	if len(missingScreens) > 0 || len(missingDays) > 0 {
		slog.Warn("some art files are missing; fallback art will be shown",
			"year", year, "missing_screens", missingScreens, "missing_days", missingDays)
	}
	if placeholders := v.Placeholders(year); len(placeholders) > 0 {
		slog.Warn("year still has placeholder art", "year", year, "count", len(placeholders), "files", placeholders)
	}
	return nil
}

// Placeholders lists year's art files that are still generated placeholders.
func (v *Validator) Placeholders(year int) []string {
	yearDir := path.Join(v.baseArtDir, strconv.Itoa(year))
	entries, err := fs.ReadDir(v.fs, yearDir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if data, err := fs.ReadFile(v.fs, path.Join(yearDir, e.Name())); err == nil && art.IsPlaceholder(data) {
			names = append(names, e.Name())
		}
	}
	return names
}

func (v *Validator) exists(p string) bool {
	_, err := fs.Stat(v.fs, p)
	return err == nil
}

// ValidateEmulation checks that the caller has ANSI emulation.
func (v *Validator) ValidateEmulation(emulation int) error {
	if emulation != 1 {
		return fmt.Errorf("ANSI emulation required (got %d, need 1)", emulation)
	}
	return nil
}

// ValidateTerminalSize checks that the terminal can show 80x24 art.
func (v *Validator) ValidateTerminalSize(width, height int) error {
	if width < 80 {
		return fmt.Errorf("terminal width too small (got %d, need at least 80)", width)
	}
	if height < 24 {
		return fmt.Errorf("terminal height too small (got %d, need at least 24)", height)
	}
	return nil
}
