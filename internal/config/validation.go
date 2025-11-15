package config

import (
	"fmt"
	"net/url"
	"strings"
)

// Validate validates the configuration
func Validate(config *Config) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	// Validate servers
	for i, server := range config.Servers {
		if err := validateServer(&server); err != nil {
			return fmt.Errorf("invalid server at index %d: %w", i, err)
		}
	}

	// Validate preferences
	if err := validatePreferences(&config.Preferences); err != nil {
		return fmt.Errorf("invalid preferences: %w", err)
	}

	// Validate paths
	if err := validatePaths(&config.Paths); err != nil {
		return fmt.Errorf("invalid paths: %w", err)
	}

	return nil
}

// validateServer validates a server configuration
func validateServer(server *ServerConfig) error {
	if server == nil {
		return fmt.Errorf("server is nil")
	}

	// Validate name
	if strings.TrimSpace(server.Name) == "" {
		return fmt.Errorf("server name cannot be empty")
	}

	// Validate URL
	if strings.TrimSpace(server.URL) == "" {
		return fmt.Errorf("server URL cannot be empty")
	}

	// Parse and validate URL format
	parsedURL, err := url.Parse(server.URL)
	if err != nil {
		return fmt.Errorf("invalid server URL: %w", err)
	}

	// Ensure scheme is http or https
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("server URL must use http or https scheme, got: %s", parsedURL.Scheme)
	}

	// Ensure host is not empty
	if parsedURL.Host == "" {
		return fmt.Errorf("server URL must have a host")
	}

	// Validate authentication if present
	if server.Auth != nil {
		if err := validateAuth(server.Auth); err != nil {
			return fmt.Errorf("invalid auth config: %w", err)
		}
	}

	return nil
}

// validateAuth validates authentication configuration
func validateAuth(auth *AuthConfig) error {
	if auth == nil {
		return fmt.Errorf("auth is nil")
	}

	// Validate auth type
	validTypes := map[string]bool{
		"basic": true,
		"token": true,
		"none":  true,
		"":      true,
	}

	if !validTypes[auth.Type] {
		return fmt.Errorf("invalid auth type: %s (must be 'basic', 'token', or 'none')", auth.Type)
	}

	// Validate auth fields based on type
	switch auth.Type {
	case "basic":
		if strings.TrimSpace(auth.Username) == "" {
			return fmt.Errorf("username is required for basic auth")
		}
		if strings.TrimSpace(auth.Password) == "" {
			return fmt.Errorf("password is required for basic auth")
		}

	case "token":
		if strings.TrimSpace(auth.Token) == "" {
			return fmt.Errorf("token is required for token auth")
		}
	}

	return nil
}

// validatePreferences validates preferences configuration
func validatePreferences(prefs *PreferencesConfig) error {
	if prefs == nil {
		return fmt.Errorf("preferences is nil")
	}

	// Validate theme
	validThemes := map[string]bool{
		"dark":  true,
		"light": true,
	}

	if !validThemes[prefs.Theme] {
		return fmt.Errorf("invalid theme: %s (must be 'dark' or 'light')", prefs.Theme)
	}

	// Validate reading mode
	validModes := map[string]bool{
		"single":  true,
		"double":  true,
		"webtoon": true,
	}

	if !validModes[prefs.ReadingMode] {
		return fmt.Errorf("invalid reading mode: %s (must be 'single', 'double', or 'webtoon')", prefs.ReadingMode)
	}

	// Validate cache size
	if prefs.CacheSizeMB < 0 {
		return fmt.Errorf("cache size must be non-negative, got: %d", prefs.CacheSizeMB)
	}

	// Validate default server index (will be checked against actual servers later)
	if prefs.DefaultServer < 0 {
		return fmt.Errorf("default server index must be non-negative, got: %d", prefs.DefaultServer)
	}

	return nil
}

// validatePaths validates path configuration
func validatePaths(paths *PathsConfig) error {
	if paths == nil {
		return fmt.Errorf("paths is nil")
	}

	// Paths can be empty (will be set to defaults)
	// Just ensure they are valid if set
	if paths.Database != "" {
		if !isValidPath(paths.Database) {
			return fmt.Errorf("invalid database path: %s", paths.Database)
		}
	}

	if paths.Cache != "" {
		if !isValidPath(paths.Cache) {
			return fmt.Errorf("invalid cache path: %s", paths.Cache)
		}
	}

	if paths.Downloads != "" {
		if !isValidPath(paths.Downloads) {
			return fmt.Errorf("invalid downloads path: %s", paths.Downloads)
		}
	}

	return nil
}

// isValidPath checks if a path string is valid
func isValidPath(path string) bool {
	// Basic validation - just check it's not empty and doesn't contain null bytes
	if strings.TrimSpace(path) == "" {
		return false
	}

	if strings.Contains(path, "\x00") {
		return false
	}

	return true
}

// ValidateServerURL validates a server URL without other server fields
func ValidateServerURL(urlStr string) error {
	if strings.TrimSpace(urlStr) == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL must use http or https scheme")
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("URL must have a host")
	}

	return nil
}
