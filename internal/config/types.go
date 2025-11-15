package config

// Config represents the application configuration
type Config struct {
	Servers     []ServerConfig  `mapstructure:"servers"`
	Preferences PreferencesConfig `mapstructure:"preferences"`
	Paths       PathsConfig     `mapstructure:"paths"`
}

// ServerConfig represents a Suwayomi server configuration
type ServerConfig struct {
	Name    string     `mapstructure:"name"`
	URL     string     `mapstructure:"url"`
	Default bool       `mapstructure:"default"`
	Auth    *AuthConfig `mapstructure:"auth,omitempty"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Type     string `mapstructure:"type"` // "basic", "token", "none"
	Username string `mapstructure:"username,omitempty"`
	Password string `mapstructure:"password,omitempty"`
	Token    string `mapstructure:"token,omitempty"`
}

// PreferencesConfig represents user preferences
type PreferencesConfig struct {
	Theme          string   `mapstructure:"theme"`           // "dark", "light"
	DefaultServer  int      `mapstructure:"default_server"`  // Index of default server
	CacheSizeMB    int      `mapstructure:"cache_size_mb"`
	AutoMarkRead   bool     `mapstructure:"auto_mark_read"`
	ReadingMode    string   `mapstructure:"reading_mode"`    // "single", "double", "webtoon"
	LocalScanDirs  []string `mapstructure:"local_scan_dirs"`
	ShowThumbnails bool     `mapstructure:"show_thumbnails"`
}

// PathsConfig represents path configurations
type PathsConfig struct {
	Database  string `mapstructure:"database"`
	Cache     string `mapstructure:"cache"`
	Downloads string `mapstructure:"downloads"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Servers: []ServerConfig{},
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
