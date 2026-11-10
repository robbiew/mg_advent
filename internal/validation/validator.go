package validation

import (
	"fmt"
	"io/fs"
	"path" // Use path instead of filepath for embedded FS (always forward slashes)
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

// Validator handles various validation checks
type Validator struct {
	baseArtDir string
	fs         fs.FS
}

// NewValidator creates a new validator
func NewValidator(embeddedFS fs.FS, baseArtDir string) *Validator {
	return &Validator{
		baseArtDir: baseArtDir,
		fs:         embeddedFS,
	}
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
		if _, err := fs.Stat(v.fs, path.Join(commonDir, name)); err != nil {
			missingCommon = append(missingCommon, name)
		}
	}
	if len(missingCommon) > 0 {
		return fmt.Errorf("missing required common art files: %v", missingCommon)
	}

	var missingScreens []string
	for _, name := range []string{"WELCOME.ANS", "COMEBACK.ANS", "GOODBYE.ANS"} {
		if _, err := fs.Stat(v.fs, path.Join(yearDir, name)); err != nil {
			missingScreens = append(missingScreens, name)
		}
	}

	suffix := strconv.Itoa(year)[2:]
	var missingDays []int
	for day := 1; day <= 25; day++ {
		padded := path.Join(yearDir, fmt.Sprintf("%02d_DEC%s.ANS", day, suffix))
		plain := path.Join(yearDir, fmt.Sprintf("%d_DEC%s.ANS", day, suffix))
		_, errPadded := fs.Stat(v.fs, padded)
		_, errPlain := fs.Stat(v.fs, plain)
		if errPadded != nil && errPlain != nil {
			missingDays = append(missingDays, day)
		}
	}

	if len(missingScreens) > 0 || len(missingDays) > 0 {
		logrus.WithFields(logrus.Fields{
			"year":            year,
			"missing_screens": missingScreens,
			"missing_days":    missingDays,
		}).Warn("Some art files are missing; fallback art will be shown")
	}
	return nil
}

// ValidateYear checks if a year is valid and has art
func (v *Validator) ValidateYear(year int) error {
	// Check reasonable year range
	if year < 2020 || year > 2030 {
		return fmt.Errorf("year %d is out of valid range (2020-2030)", year)
	}

	// Check if year directory exists
	yearDir := path.Join(v.baseArtDir, strconv.Itoa(year))
	if _, err := fs.Stat(v.fs, yearDir); err != nil {
		return fmt.Errorf("no art available for year %d", year)
	}

	return nil
}

// ValidateEmulation checks if ANSI emulation is supported
func (v *Validator) ValidateEmulation(emulation int) error {
	if emulation != 1 {
		return fmt.Errorf("ANSI emulation required (got %d, need 1)", emulation)
	}
	return nil
}

// ValidateTerminalSize checks if terminal size is adequate
func (v *Validator) ValidateTerminalSize(width, height int) error {
	if width < 80 {
		return fmt.Errorf("terminal width too small (got %d, need at least 80)", width)
	}
	if height < 24 {
		return fmt.Errorf("terminal height too small (got %d, need at least 24)", height)
	}
	return nil
}

// Override time.Now for testing
var timeNow = time.Now

// RequireKey checks if a key is required to access a year
func (v *Validator) RequireKey(year int) bool {
	// Get current date
	now := timeNow()

	// Current year is always accessible without key
	currentYear := now.Year()
	if year == currentYear {
		return false
	}

	// Only 2025 requires keys to access older years
	if currentYear == 2025 && year < currentYear {
		return true
	}

	// Non-current years don't require keys
	return false
}

// ValidateKey checks if a key is valid for a year
func (v *Validator) ValidateKey(year int, key string) bool {
	// For now, any non-empty key is valid
	return key != ""
}

// GetValidationReport generates a comprehensive validation report
func (v *Validator) GetValidationReport(year int) *ValidationReport {
	report := &ValidationReport{
		Year:     year,
		Issues:   []ValidationIssue{},
		Warnings: []ValidationIssue{},
	}

	// Check year validity
	if err := v.ValidateYear(year); err != nil {
		report.Issues = append(report.Issues, ValidationIssue{
			Type:     "year",
			Message:  err.Error(),
			Severity: "error",
		})
	}

	// Check art files
	if err := v.ValidateArtFiles(year); err != nil {
		report.Issues = append(report.Issues, ValidationIssue{
			Type:     "art_files",
			Message:  err.Error(),
			Severity: "error",
		})
	}

	report.Valid = len(report.Issues) == 0
	return report
}

// ValidationReport contains validation results
type ValidationReport struct {
	Year     int               `json:"year"`
	Valid    bool              `json:"valid"`
	Issues   []ValidationIssue `json:"issues"`
	Warnings []ValidationIssue `json:"warnings"`
}

// ValidationIssue represents a validation problem
type ValidationIssue struct {
	Type     string `json:"type"`
	Message  string `json:"message"`
	Severity string `json:"severity"` // "error" or "warning"
	Field    string `json:"field,omitempty"`
}

// HasErrors returns true if the report has any errors
func (vr *ValidationReport) HasErrors() bool {
	for _, issue := range vr.Issues {
		if issue.Severity == "error" {
			return true
		}
	}
	return false
}

// GetErrorMessages returns all error messages
func (vr *ValidationReport) GetErrorMessages() []string {
	var messages []string
	for _, issue := range vr.Issues {
		messages = append(messages, issue.Message)
	}
	return messages
}

// GetWarningMessages returns all warning messages
func (vr *ValidationReport) GetWarningMessages() []string {
	var messages []string
	for _, issue := range vr.Warnings {
		messages = append(messages, issue.Message)
	}
	return messages
}
