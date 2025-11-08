package display

import (
	"fmt"
	"strings"
)

// Theme represents a visual theme for the display
type Theme struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Colors      map[string]string `yaml:"colors"`
	Styles      map[string]string `yaml:"styles"`
}

// DefaultTheme returns the default classic theme
func DefaultTheme() *Theme {
	return &Theme{
		Name:        "classic",
		Description: "Classic ANSI art theme",
		Colors: map[string]string{
			"primary":   "\033[37m", // White
			"secondary": "\033[36m", // Cyan
			"accent":    "\033[33m", // Yellow
			"error":     "\033[31m", // Red
			"success":   "\033[32m", // Green
			"warning":   "\033[33m", // Yellow
			"info":      "\033[34m", // Blue
		},
		Styles: map[string]string{
			"title":     "bold",
			"subtitle":  "normal",
			"highlight": "bold",
			"normal":    "normal",
		},
	}
}

// ChristmasTheme returns a festive Christmas theme
func ChristmasTheme() *Theme {
	return &Theme{
		Name:        "christmas",
		Description: "Festive Christmas theme",
		Colors: map[string]string{
			"primary":   "\033[31m", // Red
			"secondary": "\033[32m", // Green
			"accent":    "\033[33m", // Gold/Yellow
			"error":     "\033[35m", // Magenta
			"success":   "\033[32m", // Green
			"warning":   "\033[33m", // Yellow
			"info":      "\033[36m", // Cyan
		},
		Styles: map[string]string{
			"title":     "bold",
			"subtitle":  "bold",
			"highlight": "bold",
			"normal":    "normal",
		},
	}
}

// WinterTheme returns a cool winter theme
func WinterTheme() *Theme {
	return &Theme{
		Name:        "winter",
		Description: "Cool winter theme",
		Colors: map[string]string{
			"primary":   "\033[37m", // White
			"secondary": "\033[34m", // Blue
			"accent":    "\033[36m", // Cyan
			"error":     "\033[31m", // Red
			"success":   "\033[32m", // Green
			"warning":   "\033[33m", // Yellow
			"info":      "\033[35m", // Magenta
		},
		Styles: map[string]string{
			"title":     "bold",
			"subtitle":  "normal",
			"highlight": "bold",
			"normal":    "normal",
		},
	}
}

// GetColor returns the ANSI color code for a theme color
func (t *Theme) GetColor(name string) string {
	if color, exists := t.Colors[name]; exists {
		return color
	}
	// Fallback to default color
	if defaultColor, exists := DefaultTheme().Colors[name]; exists {
		return defaultColor
	}
	return "\033[37m" // Default white
}

// GetStyle returns the ANSI style code for a theme style
func (t *Theme) GetStyle(name string) string {
	if style, exists := t.Styles[name]; exists {
		switch style {
		case "bold":
			return "\033[1m"
		case "italic":
			return "\033[3m"
		case "underline":
			return "\033[4m"
		case "normal":
			return "\033[0m"
		}
	}
	return "\033[0m" // Normal style
}

// ApplyColor applies a theme color to text
func (t *Theme) ApplyColor(colorName, text string) string {
	color := t.GetColor(colorName)
	reset := "\033[0m"
	return color + text + reset
}

// ApplyStyle applies a theme style to text
func (t *Theme) ApplyStyle(styleName, text string) string {
	style := t.GetStyle(styleName)
	reset := "\033[0m"
	return style + text + reset
}

// ApplyBoth applies both color and style to text
func (t *Theme) ApplyBoth(colorName, styleName, text string) string {
	color := t.GetColor(colorName)
	style := t.GetStyle(styleName)
	reset := "\033[0m"
	return color + style + text + reset
}

// ThemeManager manages available themes
type ThemeManager struct {
	themes map[string]*Theme
}

// NewThemeManager creates a new theme manager with built-in themes
func NewThemeManager() *ThemeManager {
	tm := &ThemeManager{
		themes: make(map[string]*Theme),
	}

	// Register built-in themes
	tm.RegisterTheme(DefaultTheme())
	tm.RegisterTheme(ChristmasTheme())
	tm.RegisterTheme(WinterTheme())

	return tm
}

// RegisterTheme adds a custom theme
func (tm *ThemeManager) RegisterTheme(theme *Theme) {
	tm.themes[theme.Name] = theme
}

// GetTheme returns a theme by name
func (tm *ThemeManager) GetTheme(name string) (*Theme, error) {
	if theme, exists := tm.themes[name]; exists {
		return theme, nil
	}
	return nil, fmt.Errorf("theme '%s' not found", name)
}

// ListThemes returns all available theme names
func (tm *ThemeManager) ListThemes() []string {
	var names []string
	for name := range tm.themes {
		names = append(names, name)
	}
	return names
}

// ValidateTheme checks if a theme is valid
func (tm *ThemeManager) ValidateTheme(theme *Theme) error {
	if strings.TrimSpace(theme.Name) == "" {
		return fmt.Errorf("theme name cannot be empty")
	}

	// Check required colors
	requiredColors := []string{"primary", "secondary", "accent"}
	for _, color := range requiredColors {
		if _, exists := theme.Colors[color]; !exists {
			return fmt.Errorf("theme missing required color: %s", color)
		}
	}

	return nil
}

// LoadThemeFromConfig creates a theme from configuration
func (tm *ThemeManager) LoadThemeFromConfig(config map[string]interface{}) (*Theme, error) {
	theme := &Theme{
		Colors: make(map[string]string),
		Styles: make(map[string]string),
	}

	if name, ok := config["name"].(string); ok {
		theme.Name = name
	}

	if desc, ok := config["description"].(string); ok {
		theme.Description = desc
	}

	if colors, ok := config["colors"].(map[string]interface{}); ok {
		for k, v := range colors {
			if colorStr, ok := v.(string); ok {
				theme.Colors[k] = colorStr
			}
		}
	}

	if styles, ok := config["styles"].(map[string]interface{}); ok {
		for k, v := range styles {
			if styleStr, ok := v.(string); ok {
				theme.Styles[k] = styleStr
			}
		}
	}

	if err := tm.ValidateTheme(theme); err != nil {
		return nil, err
	}

	return theme, nil
}
