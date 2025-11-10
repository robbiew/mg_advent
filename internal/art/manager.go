package art

import (
	"fmt"
	"io/fs"
	"path/filepath"
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
	yearDir := filepath.Join(m.baseDir, strconv.Itoa(year))

	// Check if year directory exists
	if _, err := fs.Stat(m.fs, yearDir); err != nil {
		return fmt.Errorf("art directory for year %d does not exist", year)
	}

	// Check common directory exists
	commonDir := filepath.Join(m.baseDir, "common")
	if _, err := fs.Stat(m.fs, commonDir); err != nil {
		return fmt.Errorf("art/common directory does not exist")
	}

	// Required common files (year-independent)
	requiredCommonFiles := []string{
		filepath.Join(commonDir, "WELCOME.ANS"),
		filepath.Join(commonDir, "GOODBYE.ANS"),
		filepath.Join(commonDir, "COMEBACK.ANS"),
		filepath.Join(commonDir, "MISSING.ANS"),
		filepath.Join(commonDir, "NOTYET.ANS"),
	}

	// Check required common files
	for _, file := range requiredCommonFiles {
		if _, err := fs.Stat(m.fs, file); err != nil {
			return fmt.Errorf("required common art file missing: %s", filepath.Base(file))
		}
	}

	// Check daily art files (1-25)
	for day := 1; day <= 25; day++ {
		fileName := fmt.Sprintf("%d_DEC%s.ANS", day, strconv.Itoa(year)[2:])
		filePath := filepath.Join(yearDir, fileName)
		if _, err := fs.Stat(m.fs, filePath); err != nil {
			logrus.WithField("file", fileName).Warn("Daily art file missing")
			// Don't fail validation for missing daily files, just warn
		}
	}

	return nil
}

// GetPath returns the path to an art file
func (m *Manager) GetPath(year int, day int, screenType string) string {
	yearDir := filepath.Join(m.baseDir, strconv.Itoa(year))
	commonDir := filepath.Join(m.baseDir, "common")

	switch screenType {
	case "welcome":
		return filepath.Join(commonDir, "WELCOME.ANS")
	case "goodbye", "exit":
		return filepath.Join(commonDir, "GOODBYE.ANS")
	case "comeback":
		return filepath.Join(commonDir, "COMEBACK.ANS")
	case "day":
		fileName := fmt.Sprintf("%d_DEC%s.ANS", day, strconv.Itoa(year)[2:])
		return filepath.Join(yearDir, fileName)
	case "missing":
		return filepath.Join(commonDir, "MISSING.ANS")
	case "notyet":
		return filepath.Join(commonDir, "NOTYET.ANS")
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

	// Preload common screens
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
