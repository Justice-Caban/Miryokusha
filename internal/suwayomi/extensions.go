package suwayomi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

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
	BaseURL    string
	HTTPClient *http.Client
	GraphQL    *GraphQLClient
}

// NewClient creates a new Suwayomi client
func NewClient(baseURL string) *Client {
	// Ensure baseURL has http:// or https:// prefix
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}

	// Remove trailing slash
	baseURL = strings.TrimSuffix(baseURL, "/")

	client := &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	// Initialize GraphQL client
	client.GraphQL = NewGraphQLClient(client)

	return client
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
	Version        string
	BuildType      string
	Revision       string
	BuildTime      string
	IsHealthy      bool
	ExtensionCount int
	MangaCount     int
}

// AboutResponse represents the /api/v1/settings/about endpoint response
type AboutResponse struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Revision string `json:"revision"`
	BuildType string `json:"buildType"`
	BuildTime int64  `json:"buildTime"`
}

// HealthCheck performs a health check on the Suwayomi server
func (c *Client) HealthCheck() (*ServerInfo, error) {
	if c.BaseURL == "" {
		return &ServerInfo{
			IsHealthy: false,
		}, fmt.Errorf("no server URL configured")
	}

	// Try to fetch server info from /api/v1/settings/about
	url := c.BaseURL + "/api/v1/settings/about"

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return &ServerInfo{
			IsHealthy: false,
		}, fmt.Errorf("failed to connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &ServerInfo{
			IsHealthy: false,
		}, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var about AboutResponse
	if err := json.NewDecoder(resp.Body).Decode(&about); err != nil {
		return &ServerInfo{
			IsHealthy: false,
		}, fmt.Errorf("failed to parse server response: %w", err)
	}

	// Convert build time from milliseconds to readable format
	buildTime := time.UnixMilli(about.BuildTime).Format("2006-01-02 15:04:05")

	return &ServerInfo{
		Version:        about.Version,
		BuildType:      about.BuildType,
		Revision:       about.Revision,
		BuildTime:      buildTime,
		IsHealthy:      true,
		ExtensionCount: 0, // TODO: Fetch from /api/v1/extension/list
		MangaCount:     0, // TODO: Fetch from /api/v1/manga/list
	}, nil
}

// Ping checks if the server is reachable
func (c *Client) Ping() bool {
	if c.BaseURL == "" {
		return false
	}

	// Try to ping the about endpoint
	url := c.BaseURL + "/api/v1/settings/about"

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
