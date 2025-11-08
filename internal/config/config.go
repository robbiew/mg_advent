package config

import (
	"fmt"
	"os"
	"time"

	"github.com/robbiew/advent/internal/display"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	App     AppConfig     `mapstructure:"app"`
	Display DisplayConfig `mapstructure:"display"`
	BBS     BBSConfig     `mapstructure:"bbs"`
	Art     ArtConfig     `mapstructure:"art"`
	Logging LoggingConfig `mapstructure:"logging"`
}

type AppConfig struct {
	Name        string `mapstructure:"name"`
	Version     string `mapstructure:"version"`
	TimeoutIdle string `mapstructure:"timeout_idle"`
	TimeoutMax  string `mapstructure:"timeout_max"`
}

type DisplayConfig struct {
	Mode        string            `mapstructure:"mode"`
	Theme       string            `mapstructure:"theme"`
	Scrolling   ScrollingConfig   `mapstructure:"scrolling"`
	Columns     ColumnConfig      `mapstructure:"columns"`
	Performance PerformanceConfig `mapstructure:"performance"`
}

// GetDisplayMode returns the display mode as DisplayMode enum
func (dc *DisplayConfig) GetDisplayMode() display.DisplayMode {
	switch dc.Mode {
	case "cp437_local":
		return display.ModeCP437
	case "utf8", "utf8_raw":
		return display.ModeUTF8
	case "cp437_raw":
		return display.ModeCP437Raw
	case "cp437":
		return display.ModeCP437Raw
	default:
		return display.ModeCP437Raw
	}
}

type ScrollingConfig struct {
	Enabled           bool `mapstructure:"enabled"`
	Indicators        bool `mapstructure:"indicators"`
	KeyboardShortcuts bool `mapstructure:"keyboard_shortcuts"`
}

type ColumnConfig struct {
	Handle80ColumnIssue bool `mapstructure:"handle_80_column_issue"`
	AutoDetectWidth     bool `mapstructure:"auto_detect_width"`
}

type PerformanceConfig struct {
	CacheEnabled bool `mapstructure:"cache_enabled"`
	CacheSizeMB  int  `mapstructure:"cache_size_mb"`
	PreloadLines int  `mapstructure:"preload_lines"`
}

type BBSConfig struct {
	DropfilePath      string `mapstructure:"dropfile_path"`
	EmulationRequired int    `mapstructure:"emulation_required"`
}

type ArtConfig struct {
	BaseDir     string `mapstructure:"base_dir"`
	CacheSize   string `mapstructure:"cache_size"`
	PreloadDays int    `mapstructure:"preload_days"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

// Load loads configuration from various sources
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Read from config file if provided
	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			logrus.WithError(err).Warn("Failed to read config file, using defaults")
		}
	} else {
		// Try to find config in default locations
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./config")
		v.AddConfigPath(".")

		if err := v.ReadInConfig(); err != nil {
			logrus.WithError(err).Debug("No config file found, using defaults")
		}
	}

	// Environment variables override
	v.SetEnvPrefix("ADVENT")
	v.AutomaticEnv()

	// Unmarshal into config struct
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	v.SetDefault("app.name", "Mistigris Advent Calendar")
	v.SetDefault("app.version", "2.0.0")
	v.SetDefault("app.timeout_idle", "5m")
	v.SetDefault("app.timeout_max", "120m")

	v.SetDefault("display.mode", "cp437")
	v.SetDefault("display.theme", "classic")
	v.SetDefault("display.scrolling.enabled", true)
	v.SetDefault("display.scrolling.indicators", true)
	v.SetDefault("display.scrolling.keyboard_shortcuts", true)
	v.SetDefault("display.columns.handle_80_column_issue", true)
	v.SetDefault("display.columns.auto_detect_width", true)
	v.SetDefault("display.performance.cache_enabled", true)
	v.SetDefault("display.performance.cache_size_mb", 50)
	v.SetDefault("display.performance.preload_lines", 100)

	v.SetDefault("bbs.emulation_required", 1)

	v.SetDefault("art.base_dir", "art")
	v.SetDefault("art.cache_size", "100MB")
	v.SetDefault("art.preload_days", 7)

	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "text")
	v.SetDefault("logging.output", "stderr")
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	// Validate display mode
	validModes := map[string]struct{}{
		"cp437":       {},
		"cp437_raw":   {},
		"cp437_local": {},
		"utf8":        {},
		"utf8_raw":    {},
	}
	if _, ok := validModes[config.Display.Mode]; !ok {
		return fmt.Errorf("invalid display mode: %s (must be one of cp437, cp437_raw, cp437_local, utf8, utf8_raw)", config.Display.Mode)
	}

	// Validate cache size
	if config.Display.Performance.CacheSizeMB < 0 {
		return fmt.Errorf("cache size must be non-negative: %d", config.Display.Performance.CacheSizeMB)
	}

	// Validate timeouts
	if _, err := time.ParseDuration(config.App.TimeoutIdle); err != nil {
		return fmt.Errorf("invalid idle timeout: %s", config.App.TimeoutIdle)
	}
	if _, err := time.ParseDuration(config.App.TimeoutMax); err != nil {
		return fmt.Errorf("invalid max timeout: %s", config.App.TimeoutMax)
	}

	return nil
}

// GetConfigPath returns the default config path
func GetConfigPath() string {
	// Check for config file in current directory first
	if _, err := os.Stat("config.yaml"); err == nil {
		return "config.yaml"
	}
	if _, err := os.Stat("config/config.yaml"); err == nil {
		return "config/config.yaml"
	}
	return ""
}
