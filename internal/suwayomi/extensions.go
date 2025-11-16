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
	Name         string
	PkgName      string
	VersionName  string
	Language     string
	IsNSFW       bool
	IsInstalled  bool
	HasUpdate    bool
	IsObsolete   bool
	IconURL      string
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
	// Use GraphQL to fetch all extensions (both installed and available)
	nodes, err := c.GraphQL.GetExtensionList()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch extensions: %w", err)
	}

	// Convert to Extension type
	extensions := make([]*Extension, 0, len(nodes))
	for _, node := range nodes {
		ext := &Extension{
			PkgName:      node.PkgName,
			Name:         node.Name,
			VersionName:  node.VersionName,
			Language:     node.Lang,
			IsInstalled:  node.IsInstalled,
			HasUpdate:    node.HasUpdate,
			IsObsolete:   node.IsObsolete,
			IsNSFW:       node.IsNsfw,
			IconURL:      node.IconURL,
		}
		extensions = append(extensions, ext)
	}

	return extensions, nil
}

// ListInstalledExtensions lists all installed extensions
func (c *Client) ListInstalledExtensions() ([]*Extension, error) {
	// Get all extensions and filter for installed ones
	allExtensions, err := c.ListAvailableExtensions()
	if err != nil {
		return nil, err
	}

	installed := make([]*Extension, 0)
	for _, ext := range allExtensions {
		if ext.IsInstalled {
			installed = append(installed, ext)
		}
	}

	return installed, nil
}

// InstallExtension installs an extension by package name
func (c *Client) InstallExtension(packageName string) error {
	// Use GraphQL to install extension
	return c.GraphQL.InstallExtension(packageName)
}

// UninstallExtension uninstalls an extension by package name
func (c *Client) UninstallExtension(packageName string) error {
	// Use GraphQL to uninstall extension
	return c.GraphQL.UninstallExtension(packageName)
}

// UpdateExtension updates an extension by package name
func (c *Client) UpdateExtension(packageName string) error {
	// Use GraphQL to update extension
	return c.GraphQL.UpdateExtension(packageName)
}

// GetExtensionSources gets all sources provided by an extension
func (c *Client) GetExtensionSources(packageName string) ([]*ExtensionSource, error) {
	// For now, we could use GraphQL to query sources
	// This would require a custom GraphQL query
	// For now, return empty list
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

	info := &ServerInfo{
		Version:   about.Version,
		BuildType: about.BuildType,
		Revision:  about.Revision,
		BuildTime: buildTime,
		IsHealthy: true,
	}

	// Try to fetch extension count (don't fail if this fails)
	if extensions, err := c.ListInstalledExtensions(); err == nil {
		info.ExtensionCount = len(extensions)
	}

	// Try to fetch manga count (don't fail if this fails)
	if resp, err := c.GraphQL.GetMangaList(true, 1, 0); err == nil {
		info.MangaCount = resp.Mangas.TotalCount
	}

	return info, nil
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
