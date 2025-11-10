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

// ValidateDate checks if the current date is valid for advent calendar
func (v *Validator) ValidateDate() error {
	now := time.Now()
	if now.Month() != time.December || now.Day() < 1 {
		return fmt.Errorf("advent calendar only available in December")
	}
	return nil
}

// ValidateArtFiles checks if required art files exist for a year
func (v *Validator) ValidateArtFiles(year int) error {
	yearDir := path.Join(v.baseArtDir, strconv.Itoa(year))

	// Check if year directory exists
	if _, err := fs.Stat(v.fs, yearDir); err != nil {
		return fmt.Errorf("art directory for year %d does not exist", year)
	}

	// Check common directory exists
	commonDir := path.Join(v.baseArtDir, "common")
	if _, err := fs.Stat(v.fs, commonDir); err != nil {
		return fmt.Errorf("art/common directory does not exist")
	}

	// Required common files (year-independent)
	requiredCommonFiles := []string{
		path.Join(commonDir, "WELCOME.ANS"),
		path.Join(commonDir, "GOODBYE.ANS"),
		path.Join(commonDir, "COMEBACK.ANS"),
		path.Join(commonDir, "MISSING.ANS"),
		path.Join(commonDir, "NOTYET.ANS"),
	}

	// Check required common files
	missingFiles := []string{}
	for _, file := range requiredCommonFiles {
		if _, err := fs.Stat(v.fs, file); err != nil {
			missingFiles = append(missingFiles, path.Base(file))
		}
	}

	if len(missingFiles) > 0 {
		return fmt.Errorf("missing required art files: %v", missingFiles)
	}

	// Check daily art files (warn but don't fail)
	currentYear := time.Now().Year()
	maxDay := 25
	if year == currentYear && time.Now().Month() == time.December {
		maxDay = time.Now().Day()
		if maxDay > 25 {
			maxDay = 25
		}
	}

	missingDays := []int{}
	for day := 1; day <= maxDay; day++ {
		fileName := fmt.Sprintf("%d_DEC%s.ANS", day, strconv.Itoa(year)[2:])
		filePath := path.Join(yearDir, fileName)
		if _, err := fs.Stat(v.fs, filePath); err != nil {
			missingDays = append(missingDays, day)
		}
	}

	if len(missingDays) > 0 {
		logrus.WithFields(logrus.Fields{
			"year":         year,
			"missing_days": missingDays,
		}).Warn("Some daily art files are missing")
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

	// Check date validity
	if err := v.ValidateDate(); err != nil {
		report.Issues = append(report.Issues, ValidationIssue{
			Type:     "date",
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
