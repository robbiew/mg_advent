package art

import (
	"fmt"
	"io/fs"
	"path" // Use path instead of filepath for embedded FS (always forward slashes)
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

// Manager handles art file management and caching
type Manager struct {
	baseDir string
	cache   map[string][]string
	fs      fs.FS // Embedded filesystem
}

// NewManager creates a new art manager using embedded filesystem
func NewManager(embeddedFS fs.FS, baseDir string) *Manager {
	return &Manager{
		baseDir: baseDir,
		cache:   make(map[string][]string),
		fs:      embeddedFS,
	}
}

// Validate checks if art files exist for the given year
func (m *Manager) Validate(year int) error {
	yearDir := path.Join(m.baseDir, strconv.Itoa(year))

	// Check if year directory exists
	if _, err := fs.Stat(m.fs, yearDir); err != nil {
		return fmt.Errorf("art directory for year %d does not exist", year)
	}

	// Check common directory exists
	commonDir := path.Join(m.baseDir, "common")
	if _, err := fs.Stat(m.fs, commonDir); err != nil {
		return fmt.Errorf("art/common directory does not exist")
	}

	// Required common files (year-independent)
	requiredCommonFiles := []string{
		path.Join(commonDir, "MISSING.ANS"),
		path.Join(commonDir, "NOTYET.ANS"),
	}

	// Check required common files
	for _, file := range requiredCommonFiles {
		if _, err := fs.Stat(m.fs, file); err != nil {
			return fmt.Errorf("required common art file missing: %s", path.Base(file))
		}
	}

	// Check daily art files (1-25)
	for day := 1; day <= 25; day++ {
		fileName := fmt.Sprintf("%d_DEC%s.ANS", day, strconv.Itoa(year)[2:])
		filePath := path.Join(yearDir, fileName)
		if _, err := fs.Stat(m.fs, filePath); err != nil {
			logrus.WithField("file", fileName).Warn("Daily art file missing")
			// Don't fail validation for missing daily files, just warn
		}
	}

	return nil
}

// GetPath returns the path to an art file
func (m *Manager) GetPath(year int, day int, screenType string) string {
	yearDir := path.Join(m.baseDir, strconv.Itoa(year))
	commonDir := path.Join(m.baseDir, "common")

	switch screenType {
	case "welcome":
		// Always use year-specific WELCOME.ANS
		return path.Join(yearDir, "WELCOME.ANS")
	case "info":
		// First check if year-specific INFOFILE.ANS exists
		yearSpecificInfo := path.Join(yearDir, "INFOFILE.ANS")
		if _, err := fs.Stat(m.fs, yearSpecificInfo); err == nil {
			return yearSpecificInfo
		}
		// Fall back to root INFOFILE.ANS
		return path.Join(m.baseDir, "INFOFILE.ANS")
	case "members":
		// First check if year-specific MEMBERS.ANS exists
		yearSpecificMembers := path.Join(yearDir, "MEMBERS.ANS")
		if _, err := fs.Stat(m.fs, yearSpecificMembers); err == nil {
			return yearSpecificMembers
		}
		// Fall back to root MEMBERS.ANS
		return path.Join(m.baseDir, "MEMBERS.ANS")
	case "goodbye", "exit":
		// Always use year-specific GOODBYE.ANS (like WELCOME.ANS)
		return path.Join(yearDir, "GOODBYE.ANS")
	case "comeback":
		// Always use year-specific COMEBACK.ANS (like WELCOME.ANS and GOODBYE.ANS)
		return path.Join(yearDir, "COMEBACK.ANS")
	case "day":
		// Try both zero-padded (01_DEC25.ANS) and single-digit (1_DEC25.ANS) formats
		yearSuffix := strconv.Itoa(year)[2:]

		// First try the format that matches the year's convention
		var primaryFileName, fallbackFileName string

		// For 2025 and later, try zero-padded format first
		if year >= 2025 {
			// Zero-padded format (e.g., 01_DEC25.ANS)
			primaryFileName = fmt.Sprintf("%02d_DEC%s.ANS", day, yearSuffix)
			// Single-digit format as fallback (e.g., 1_DEC25.ANS)
			fallbackFileName = fmt.Sprintf("%d_DEC%s.ANS", day, yearSuffix)
		} else {
			// For 2024 and earlier, try single-digit format first
			primaryFileName = fmt.Sprintf("%d_DEC%s.ANS", day, yearSuffix)
			// Zero-padded format as fallback
			fallbackFileName = fmt.Sprintf("%02d_DEC%s.ANS", day, yearSuffix)
		}

		// Try primary format first
		primaryPath := path.Join(yearDir, primaryFileName)
		if _, err := fs.Stat(m.fs, primaryPath); err == nil {
			return primaryPath
		}

		// Try fallback format if primary format doesn't exist
		fallbackPath := path.Join(yearDir, fallbackFileName)
		if _, err := fs.Stat(m.fs, fallbackPath); err == nil {
			return fallbackPath
		}

		// If neither exists, return the primary format path
		// This will eventually fall back to MISSING.ANS in the display engine
		return primaryPath
	case "missing":
		return path.Join(commonDir, "MISSING.ANS")
	case "notyet":
		return path.Join(commonDir, "NOTYET.ANS")
	default:
		return ""
	}
}

// ListYears returns all available years
func (m *Manager) ListYears() ([]int, error) {
	var years []int

	entries, err := fs.ReadDir(m.fs, m.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read art directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			if year, err := strconv.Atoi(entry.Name()); err == nil {
				// Validate it's a reasonable year
				if year >= 2020 && year <= 2030 {
					years = append(years, year)
				}
			}
		}
	}

	return years, nil
}

// LoadArt loads and processes an art file
func (m *Manager) LoadArt(filePath string) ([]string, error) {
	// Check cache first
	if cached, exists := m.cache[filePath]; exists {
		return cached, nil
	}

	// Load file
	content, err := fs.ReadFile(m.fs, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read art file: %w", err)
	}

	// Process content
	lines := m.processContent(content)

	// Cache the result
	m.cache[filePath] = lines

	return lines, nil
}

// processContent processes raw file content into displayable lines
func (m *Manager) processContent(content []byte) []string {
	// Convert to string and trim SAUCE metadata
	contentStr := string(content)
	contentStr = m.trimSAUCE(contentStr)

	// Split into lines
	lines := strings.Split(contentStr, "\r\n")

	// Trim empty lines at end
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	return lines
}

// trimSAUCE removes SAUCE metadata from content
func (m *Manager) trimSAUCE(content string) string {
	// Look for SAUCE marker
	sauceIndex := strings.Index(content, "SAUCE00")
	if sauceIndex == -1 {
		return content
	}

	// Find the start of SAUCE metadata
	comntIndex := strings.LastIndex(content[:sauceIndex], "COMNT")
	if comntIndex == -1 {
		comntIndex = sauceIndex
	}

	// Return content before SAUCE metadata
	return content[:comntIndex]
}

// IsCached checks if a file is in cache
func (m *Manager) IsCached(filePath string) bool {
	_, exists := m.cache[filePath]
	return exists
}

// ClearCache clears the art cache
func (m *Manager) ClearCache() {
	m.cache = make(map[string][]string)
}

// GetCacheSize returns the number of cached files
func (m *Manager) GetCacheSize() int {
	return len(m.cache)
}

// PreloadArt preloads art for a year into cache
func (m *Manager) PreloadArt(year int, maxDay int) error {
	logrus.WithField("year", year).Info("Preloading art files")

	// Preload common and year-specific screens
	screens := []string{"welcome", "goodbye", "comeback"}
	for _, screen := range screens {
		filePath := m.GetPath(year, 0, screen)
		if filePath != "" {
			if _, err := m.LoadArt(filePath); err != nil {
				logrus.WithError(err).WithField("file", filePath).Warn("Failed to preload art file")
			}
		}
	}

	// Preload daily art up to maxDay
	for day := 1; day <= maxDay && day <= 25; day++ {
		filePath := m.GetPath(year, day, "day")
		if filePath != "" {
			if _, err := m.LoadArt(filePath); err != nil {
				logrus.WithError(err).WithField("file", filePath).Warn("Failed to preload daily art")
			}
		}
	}

	logrus.WithFields(logrus.Fields{
		"year":   year,
		"cached": m.GetCacheSize(),
	}).Info("Art preloading complete")

	return nil
}

// GetArtInfo returns information about an art file
func (m *Manager) GetArtInfo(filePath string) (lines int, width int, height int, err error) {
	content, err := m.LoadArt(filePath)
	if err != nil {
		return 0, 0, 0, err
	}

	height = len(content)
	width = 0

	for _, line := range content {
		if len(line) > width {
			width = len(line)
		}
	}

	return len(content), width, height, nil
}

// ValidateFile validates a single art file
func (m *Manager) ValidateFile(filePath string) error {
	if _, err := fs.Stat(m.fs, filePath); err != nil {
		return fmt.Errorf("file does not exist: %s", filePath)
	}

	content, err := m.LoadArt(filePath)
	if err != nil {
		return fmt.Errorf("failed to load file: %w", err)
	}

	if len(content) == 0 {
		return fmt.Errorf("file is empty: %s", filePath)
	}

	return nil
}
