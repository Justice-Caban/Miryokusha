package suwayomi

// Extension represents a Suwayomi extension
type Extension struct {
	Name        string
	PackageName string
	Version     string
	Language    string
	IsNSFW      bool
	IsInstalled bool
	HasUpdate   bool
	IsObsolete  bool
	IconURL     string
}

// ExtensionSource represents a manga source provided by an extension
type ExtensionSource struct {
	ID          string
	Name        string
	Language    string
	IsNSFW      bool
	DisplayName string
}

// Client represents a Suwayomi server client
type Client struct {
	BaseURL string
	// TODO: Add HTTP client and authentication
}

// NewClient creates a new Suwayomi client
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
	}
}

// ListAvailableExtensions lists all available extensions from the repository
func (c *Client) ListAvailableExtensions() ([]*Extension, error) {
	// TODO: Implement API call to /api/v1/extension/list
	// For now, return empty list
	return []*Extension{}, nil
}

// ListInstalledExtensions lists all installed extensions
func (c *Client) ListInstalledExtensions() ([]*Extension, error) {
	// TODO: Implement API call to /api/v1/extension/list with filter
	// For now, return empty list
	return []*Extension{}, nil
}

// InstallExtension installs an extension by package name
func (c *Client) InstallExtension(packageName string) error {
	// TODO: Implement API call to /api/v1/extension/install/{pkgName}
	return nil
}

// UninstallExtension uninstalls an extension by package name
func (c *Client) UninstallExtension(packageName string) error {
	// TODO: Implement API call to /api/v1/extension/uninstall/{pkgName}
	return nil
}

// UpdateExtension updates an extension by package name
func (c *Client) UpdateExtension(packageName string) error {
	// TODO: Implement API call to /api/v1/extension/update/{pkgName}
	return nil
}

// GetExtensionSources gets all sources provided by an extension
func (c *Client) GetExtensionSources(packageName string) ([]*ExtensionSource, error) {
	// TODO: Implement getting sources from extension
	return []*ExtensionSource{}, nil
}

// ServerInfo represents information about the Suwayomi server
type ServerInfo struct {
	Version       string
	BuildType     string
	Revision      string
	BuildTime     string
	IsHealthy     bool
	ExtensionCount int
	MangaCount    int
}

// HealthCheck performs a health check on the Suwayomi server
func (c *Client) HealthCheck() (*ServerInfo, error) {
	// TODO: Implement API call to /api/v1/settings/about or similar endpoint
	// For now, return mock data indicating server is available
	if c.BaseURL == "" {
		return &ServerInfo{
			IsHealthy: false,
		}, nil
	}

	// TODO: Actually ping the server
	return &ServerInfo{
		Version:       "1.0.0",
		BuildType:     "Stable",
		Revision:      "abc123",
		BuildTime:     "2024-01-01",
		IsHealthy:     true,
		ExtensionCount: 0,
		MangaCount:    0,
	}, nil
}

// Ping checks if the server is reachable
func (c *Client) Ping() bool {
	if c.BaseURL == "" {
		return false
	}
	// TODO: Implement actual HTTP ping
	// For now, return true if BaseURL is set
	return true
}
