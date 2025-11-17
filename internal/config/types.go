package config

// Config represents the application configuration
type Config struct {
	Servers          []ServerConfig         `mapstructure:"servers" yaml:"servers"`
	ServerManagement ServerManagementConfig `mapstructure:"server_management" yaml:"server_management"`
	Preferences      PreferencesConfig      `mapstructure:"preferences" yaml:"preferences"`
	Paths            PathsConfig            `mapstructure:"paths" yaml:"paths"`
	Updates          UpdateConfig           `mapstructure:"updates" yaml:"updates"`
}

// ServerConfig represents a Suwayomi server configuration
type ServerConfig struct {
	Name    string      `mapstructure:"name" yaml:"name"`
	URL     string      `mapstructure:"url" yaml:"url"`
	Default bool        `mapstructure:"default" yaml:"default"`
	Auth    *AuthConfig `mapstructure:"auth,omitempty" yaml:"auth,omitempty"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Type     string `mapstructure:"type" yaml:"type"`                         // "basic", "token", "none"
	Username string `mapstructure:"username,omitempty" yaml:"username,omitempty"`
	Password string `mapstructure:"password,omitempty" yaml:"password,omitempty"`
	Token    string `mapstructure:"token,omitempty" yaml:"token,omitempty"`
}

// ServerManagementConfig represents server process management configuration
type ServerManagementConfig struct {
	Enabled        bool     `mapstructure:"enabled" yaml:"enabled"`                  // Enable server management
	ExecutablePath string   `mapstructure:"executable_path" yaml:"executable_path"`  // Path to Suwayomi JAR/binary
	Args           []string `mapstructure:"args" yaml:"args"`                        // Additional arguments
	WorkDir        string   `mapstructure:"work_dir" yaml:"work_dir"`                // Working directory
	AutoStart      bool     `mapstructure:"auto_start" yaml:"auto_start"`            // Start server on app launch
}

// PreferencesConfig represents user preferences
type PreferencesConfig struct {
	Theme          string   `mapstructure:"theme" yaml:"theme"`                      // "dark", "light"
	DefaultServer  int      `mapstructure:"default_server" yaml:"default_server"`    // Index of default server
	CacheSizeMB    int      `mapstructure:"cache_size_mb" yaml:"cache_size_mb"`
	AutoMarkRead   bool     `mapstructure:"auto_mark_read" yaml:"auto_mark_read"`
	ReadingMode    string   `mapstructure:"reading_mode" yaml:"reading_mode"`        // "single", "double", "webtoon"
	LocalScanDirs  []string `mapstructure:"local_scan_dirs" yaml:"local_scan_dirs"`
	ShowThumbnails bool     `mapstructure:"show_thumbnails" yaml:"show_thumbnails"`
}

// PathsConfig represents path configurations
type PathsConfig struct {
	Database  string `mapstructure:"database" yaml:"database"`
	Cache     string `mapstructure:"cache" yaml:"cache"`
	Downloads string `mapstructure:"downloads" yaml:"downloads"`
}

// UpdateConfig represents library update configuration
type UpdateConfig struct {
	SmartUpdate            bool    `mapstructure:"smart_update" yaml:"smart_update"`                                  // Use smart update algorithm (Mihon-style)
	MinIntervalHours       int     `mapstructure:"min_interval_hours" yaml:"min_interval_hours"`                      // Minimum hours between checks (default: 12)
	UpdateOnlyOngoing      bool    `mapstructure:"update_only_ongoing" yaml:"update_only_ongoing"`                    // Only update ongoing series
	UpdateOnlyStarted      bool    `mapstructure:"update_only_started" yaml:"update_only_started"`                    // Only update series that have been read
	UpdateOnlyCompleted    bool    `mapstructure:"update_only_completed" yaml:"update_only_completed"`                // Only update completed series
	MaxConsecutiveFailures int     `mapstructure:"max_consecutive_failures" yaml:"max_consecutive_failures"`          // Skip after this many failures
	IntervalMultiplier     float64 `mapstructure:"interval_multiplier" yaml:"interval_multiplier"`                    // Multiply expected interval by this
	AutoUpdateEnabled      bool    `mapstructure:"auto_update_enabled" yaml:"auto_update_enabled"`                    // Enable automatic updates
	AutoUpdateIntervalHrs  int     `mapstructure:"auto_update_interval_hrs" yaml:"auto_update_interval_hrs"`          // Hours between automatic updates
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Servers: []ServerConfig{},
		ServerManagement: ServerManagementConfig{
			Enabled:        false,
			ExecutablePath: "",
			Args:           []string{},
			WorkDir:        "",
			AutoStart:      false,
		},
		Preferences: PreferencesConfig{
			Theme:          "dark",
			DefaultServer:  0,
			CacheSizeMB:    500,
			AutoMarkRead:   true,
			ReadingMode:    "single",
			LocalScanDirs:  []string{},
			ShowThumbnails: true,
		},
		Paths: PathsConfig{
			Database:  "", // Will be set to default location
			Cache:     "", // Will be set to default location
			Downloads: "", // Will be set to default location
		},
		Updates: UpdateConfig{
			SmartUpdate:            true,  // Enable smart updates by default
			MinIntervalHours:       12,    // Check at most every 12 hours
			UpdateOnlyOngoing:      true,  // Skip completed series
			UpdateOnlyStarted:      false, // Update all, not just started
			MaxConsecutiveFailures: 10,    // Skip after 10 consecutive failures
			IntervalMultiplier:     1.5,   // 1.5x safety margin on expected interval
			AutoUpdateEnabled:      false, // Disable auto-updates by default
			AutoUpdateIntervalHrs:  24,    // Auto-update once per day if enabled
		},
	}
}

// GetDefaultServer returns the default server configuration
func (c *Config) GetDefaultServer() *ServerConfig {
	if len(c.Servers) == 0 {
		return nil
	}

	// First, look for a server marked as default
	for i := range c.Servers {
		if c.Servers[i].Default {
			return &c.Servers[i]
		}
	}

	// If no default is marked, use the server at default_server index
	if c.Preferences.DefaultServer >= 0 && c.Preferences.DefaultServer < len(c.Servers) {
		return &c.Servers[c.Preferences.DefaultServer]
	}

	// Otherwise, return the first server
	return &c.Servers[0]
}

// AddServer adds a server to the configuration
func (c *Config) AddServer(server ServerConfig) {
	// If this is the first server, mark it as default
	if len(c.Servers) == 0 {
		server.Default = true
	}

	c.Servers = append(c.Servers, server)
}

// RemoveServer removes a server by index
func (c *Config) RemoveServer(index int) bool {
	if index < 0 || index >= len(c.Servers) {
		return false
	}

	// Remove the server
	c.Servers = append(c.Servers[:index], c.Servers[index+1:]...)

	// If we removed the default server and there are still servers left,
	// mark the first one as default
	if len(c.Servers) > 0 {
		hasDefault := false
		for i := range c.Servers {
			if c.Servers[i].Default {
				hasDefault = true
				break
			}
		}
		if !hasDefault {
			c.Servers[0].Default = true
		}
	}

	return true
}

// SetDefaultServer sets the default server by index
func (c *Config) SetDefaultServer(index int) bool {
	if index < 0 || index >= len(c.Servers) {
		return false
	}

	// Unmark all servers as default
	for i := range c.Servers {
		c.Servers[i].Default = false
	}

	// Mark the selected server as default
	c.Servers[index].Default = true
	c.Preferences.DefaultServer = index

	return true
}
