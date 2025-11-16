package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const (
	configDirName  = "miryokusha"
	configFileName = "config"
	configFileType = "yaml"
)

var (
	configDir  string
	configPath string
)

func init() {
	// Get user config directory
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		// Fallback to home directory
		homeDir, _ := os.UserHomeDir()
		userConfigDir = filepath.Join(homeDir, ".config")
	}

	configDir = filepath.Join(userConfigDir, configDirName)
	configPath = filepath.Join(configDir, configFileName+"."+configFileType)
}

// Load loads the configuration from the config file
func Load() (*Config, error) {
	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Set up Viper
	viper.SetConfigName(configFileName)
	viper.SetConfigType(configFileType)
	viper.AddConfigPath(configDir)

	// Set environment variable prefix
	viper.SetEnvPrefix("MIRYOKUSHA")
	viper.AutomaticEnv()

	// Try to read the config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found, create default config
			return createDefaultConfig()
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	// Unmarshal config
	config := &Config{}
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Set default paths if not specified
	if err := setDefaultPaths(config); err != nil {
		return nil, fmt.Errorf("failed to set default paths: %w", err)
	}

	// Validate configuration
	if err := Validate(config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

// Save saves the configuration to the config file
func Save(config *Config) error {
	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Validate before saving
	if err := Validate(config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// Marshal config to Viper
	viper.Set("servers", config.Servers)
	viper.Set("preferences", config.Preferences)
	viper.Set("paths", config.Paths)
	viper.Set("updates", config.Updates)
	viper.Set("server_management", config.ServerManagement)

	// Write config file
	if err := viper.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// createDefaultConfig creates and saves a default configuration
func createDefaultConfig() (*Config, error) {
	config := DefaultConfig()

	// Set default paths
	if err := setDefaultPaths(config); err != nil {
		return nil, fmt.Errorf("failed to set default paths: %w", err)
	}

	// Save the default config
	if err := Save(config); err != nil {
		return nil, fmt.Errorf("failed to save default config: %w", err)
	}

	return config, nil
}

// setDefaultPaths sets default paths if not already set
func setDefaultPaths(config *Config) error {
	// Get data directory
	dataDir, err := getDataDir()
	if err != nil {
		return err
	}

	// Set default database path
	if config.Paths.Database == "" {
		config.Paths.Database = filepath.Join(dataDir, "miryokusha.db")
	}

	// Set default cache path
	if config.Paths.Cache == "" {
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			cacheDir = filepath.Join(dataDir, "cache")
		} else {
			cacheDir = filepath.Join(cacheDir, configDirName)
		}
		config.Paths.Cache = cacheDir
	}

	// Set default downloads path
	if config.Paths.Downloads == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			config.Paths.Downloads = filepath.Join(dataDir, "downloads")
		} else {
			config.Paths.Downloads = filepath.Join(homeDir, "Downloads", "Miryokusha")
		}
	}

	// Create directories if they don't exist
	dirs := []string{
		filepath.Dir(config.Paths.Database),
		config.Paths.Cache,
		config.Paths.Downloads,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// getDataDir returns the data directory for the application
func getDataDir() (string, error) {
	// On Linux, use XDG_DATA_HOME or ~/.local/share
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		dataHome = filepath.Join(homeDir, ".local", "share")
	}

	return filepath.Join(dataHome, configDirName), nil
}

// GetConfigPath returns the path to the config file
func GetConfigPath() string {
	return configPath
}

// GetConfigDir returns the config directory
func GetConfigDir() string {
	return configDir
}

// Exists checks if the config file exists
func Exists() bool {
	_, err := os.Stat(configPath)
	return err == nil
}
