package suwayomi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		expectedURL string
	}{
		{
			name:        "url with http prefix",
			baseURL:     "http://localhost:4567",
			expectedURL: "http://localhost:4567",
		},
		{
			name:        "url with https prefix",
			baseURL:     "https://example.com:4567",
			expectedURL: "https://example.com:4567",
		},
		{
			name:        "url without prefix",
			baseURL:     "localhost:4567",
			expectedURL: "http://localhost:4567",
		},
		{
			name:        "url with trailing slash",
			baseURL:     "http://localhost:4567/",
			expectedURL: "http://localhost:4567",
		},
		{
			name:        "url without prefix and with trailing slash",
			baseURL:     "localhost:4567/",
			expectedURL: "http://localhost:4567",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.baseURL)

			assert.NotNil(t, client)
			assert.Equal(t, tt.expectedURL, client.BaseURL)
			assert.NotNil(t, client.HTTPClient)
			assert.NotNil(t, client.GraphQL)
			assert.Equal(t, 10*time.Second, client.HTTPClient.Timeout)
		})
	}
}

func TestClient_ListAvailableExtensions(t *testing.T) {
	t.Run("successful fetch", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify it's a GraphQL request
			assert.Equal(t, "/api/graphql", r.URL.Path)

			mockResponse := GraphQLResponse{
				Data: json.RawMessage(`{
					"extensions": {
						"nodes": [
							{
								"pkgName": "eu.kanade.tachiyomi.extension.en.mangadex",
								"name": "MangaDex",
								"versionName": "1.4.2",
								"lang": "en",
								"isInstalled": true,
								"hasUpdate": false,
								"isObsolete": false,
								"isNsfw": false,
								"iconUrl": "http://example.com/icon.png"
							},
							{
								"pkgName": "eu.kanade.tachiyomi.extension.ja.mangaplus",
								"name": "Manga Plus",
								"versionName": "1.2.0",
								"lang": "ja",
								"isInstalled": false,
								"hasUpdate": false,
								"isObsolete": false,
								"isNsfw": false,
								"iconUrl": "http://example.com/icon2.png"
							},
							{
								"pkgName": "eu.kanade.tachiyomi.extension.en.nhentai",
								"name": "nHentai",
								"versionName": "2.0.1",
								"lang": "en",
								"isInstalled": true,
								"hasUpdate": true,
								"isObsolete": false,
								"isNsfw": true,
								"iconUrl": "http://example.com/icon3.png"
							}
						]
					}
				}`),
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		extensions, err := client.ListAvailableExtensions()

		require.NoError(t, err)
		assert.Len(t, extensions, 3)

		// Verify first extension
		ext1 := extensions[0]
		assert.Equal(t, "eu.kanade.tachiyomi.extension.en.mangadex", ext1.PkgName)
		assert.Equal(t, "MangaDex", ext1.Name)
		assert.Equal(t, "1.4.2", ext1.VersionName)
		assert.Equal(t, "en", ext1.Language)
		assert.True(t, ext1.IsInstalled)
		assert.False(t, ext1.HasUpdate)
		assert.False(t, ext1.IsObsolete)
		assert.False(t, ext1.IsNSFW)
		assert.Equal(t, "http://example.com/icon.png", ext1.IconURL)

		// Verify second extension (not installed)
		ext2 := extensions[1]
		assert.Equal(t, "Manga Plus", ext2.Name)
		assert.Equal(t, "ja", ext2.Language)
		assert.False(t, ext2.IsInstalled)

		// Verify third extension (NSFW with update)
		ext3 := extensions[2]
		assert.Equal(t, "nHentai", ext3.Name)
		assert.True(t, ext3.IsNSFW)
		assert.True(t, ext3.HasUpdate)
		assert.True(t, ext3.IsInstalled)
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		extensions, err := client.ListAvailableExtensions()

		require.Error(t, err)
		assert.Nil(t, extensions)
		assert.Contains(t, err.Error(), "failed to fetch extensions")
	})

	t.Run("graphql error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mockResponse := GraphQLResponse{
				Errors: []GraphQLError{
					{Message: "extensions query failed"},
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		extensions, err := client.ListAvailableExtensions()

		require.Error(t, err)
		assert.Nil(t, extensions)
		assert.Contains(t, err.Error(), "extensions query failed")
	})
}

func TestClient_ListInstalledExtensions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockResponse := GraphQLResponse{
			Data: json.RawMessage(`{
				"extensions": {
					"nodes": [
						{
							"pkgName": "ext1",
							"name": "Extension 1",
							"versionName": "1.0.0",
							"lang": "en",
							"isInstalled": true,
							"hasUpdate": false,
							"isObsolete": false,
							"isNsfw": false,
							"iconUrl": ""
						},
						{
							"pkgName": "ext2",
							"name": "Extension 2",
							"versionName": "2.0.0",
							"lang": "en",
							"isInstalled": false,
							"hasUpdate": false,
							"isObsolete": false,
							"isNsfw": false,
							"iconUrl": ""
						},
						{
							"pkgName": "ext3",
							"name": "Extension 3",
							"versionName": "3.0.0",
							"lang": "ja",
							"isInstalled": true,
							"hasUpdate": true,
							"isObsolete": false,
							"isNsfw": false,
							"iconUrl": ""
						}
					]
				}
			}`),
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	installed, err := client.ListInstalledExtensions()

	require.NoError(t, err)
	assert.Len(t, installed, 2) // Only ext1 and ext3 are installed

	// Verify only installed extensions are returned
	assert.Equal(t, "ext1", installed[0].PkgName)
	assert.Equal(t, "ext3", installed[1].PkgName)
	assert.True(t, installed[0].IsInstalled)
	assert.True(t, installed[1].IsInstalled)
}

func TestClient_InstallExtension(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req GraphQLRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Verify correct mutation
		assert.Contains(t, req.Query, "InstallExtension")

		// Verify package name
		input := req.Variables["input"].(map[string]interface{})
		assert.Equal(t, "test.extension.pkg", input["pkgName"])

		mockResponse := GraphQLResponse{
			Data: json.RawMessage(`{
				"installExternalExtension": {
					"extension": {
						"pkgName": "test.extension.pkg",
						"isInstalled": true
					}
				}
			}`),
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.InstallExtension("test.extension.pkg")

	require.NoError(t, err)
}

func TestClient_UninstallExtension(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req GraphQLRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Verify correct mutation
		assert.Contains(t, req.Query, "UninstallExtension")

		// Verify package name
		input := req.Variables["input"].(map[string]interface{})
		assert.Equal(t, "test.extension.pkg", input["pkgName"])

		mockResponse := GraphQLResponse{
			Data: json.RawMessage(`{
				"uninstallExtension": {
					"extension": {
						"pkgName": "test.extension.pkg",
						"isInstalled": false
					}
				}
			}`),
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.UninstallExtension("test.extension.pkg")

	require.NoError(t, err)
}

func TestClient_UpdateExtension(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req GraphQLRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Verify correct mutation
		assert.Contains(t, req.Query, "UpdateExtension")

		// Verify package name
		input := req.Variables["input"].(map[string]interface{})
		assert.Equal(t, "test.extension.pkg", input["pkgName"])

		mockResponse := GraphQLResponse{
			Data: json.RawMessage(`{
				"updateExtension": {
					"extension": {
						"pkgName": "test.extension.pkg",
						"hasUpdate": false
					}
				}
			}`),
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.UpdateExtension("test.extension.pkg")

	require.NoError(t, err)
}

func TestClient_GetExtensionSources(t *testing.T) {
	client := NewClient("http://localhost:4567")

	// Currently returns empty list (not implemented)
	sources, err := client.GetExtensionSources("test.extension")

	require.NoError(t, err)
	assert.NotNil(t, sources)
	assert.Len(t, sources, 0)
}

func TestClient_HealthCheck(t *testing.T) {
	t.Run("successful health check", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/v1/settings/about" {
				about := AboutResponse{
					Name:      "Suwayomi-Server",
					Version:   "v0.7.0",
					Revision:  "abc123",
					BuildType: "Stable",
					BuildTime: 1700000000000,
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(about)
			} else if r.URL.Path == "/api/graphql" {
				// Mock extension list request
				var req GraphQLRequest
				json.NewDecoder(r.Body).Decode(&req)

				if req.Query != "" && len(req.Query) > 0 {
					if contains(req.Query, "GetExtensions") {
						// Extensions list
						mockResponse := GraphQLResponse{
							Data: json.RawMessage(`{
								"extensions": {
									"nodes": [
										{"pkgName": "ext1", "name": "Ext1", "versionName": "1.0", "lang": "en", "isInstalled": true, "hasUpdate": false, "isObsolete": false, "isNsfw": false, "iconUrl": ""}
									]
								}
							}`),
						}
						w.WriteHeader(http.StatusOK)
						json.NewEncoder(w).Encode(mockResponse)
					} else if contains(req.Query, "GetMangaList") {
						// Manga list
						mockResponse := GraphQLResponse{
							Data: json.RawMessage(`{
								"mangas": {
									"totalCount": 42,
									"nodes": []
								}
							}`),
						}
						w.WriteHeader(http.StatusOK)
						json.NewEncoder(w).Encode(mockResponse)
					}
				}
			}
		}))
		defer server.Close()

		client := NewClient(server.URL)
		info, err := client.HealthCheck()

		require.NoError(t, err)
		require.NotNil(t, info)

		assert.True(t, info.IsHealthy)
		assert.Equal(t, "v0.7.0", info.Version)
		assert.Equal(t, "Stable", info.BuildType)
		assert.Equal(t, "abc123", info.Revision)
		assert.NotEmpty(t, info.BuildTime)
		assert.Equal(t, 1, info.ExtensionCount)
		assert.Equal(t, 42, info.MangaCount)
	})

	t.Run("empty base URL", func(t *testing.T) {
		client := &Client{
			BaseURL:    "",
			HTTPClient: &http.Client{},
		}

		info, err := client.HealthCheck()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no server URL configured")
		assert.False(t, info.IsHealthy)
	})

	t.Run("server unreachable", func(t *testing.T) {
		client := NewClient("http://localhost:19999") // Non-existent port

		info, err := client.HealthCheck()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to connect")
		assert.False(t, info.IsHealthy)
	})

	t.Run("server returns error status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		info, err := client.HealthCheck()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "server returned status")
		assert.False(t, info.IsHealthy)
	})

	t.Run("invalid json response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("not json"))
		}))
		defer server.Close()

		client := NewClient(server.URL)
		info, err := client.HealthCheck()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse")
		assert.False(t, info.IsHealthy)
	})

	t.Run("health check with extension fetch error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/v1/settings/about" {
				about := AboutResponse{
					Name:      "Suwayomi-Server",
					Version:   "v0.7.0",
					Revision:  "abc123",
					BuildType: "Stable",
					BuildTime: 1700000000000,
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(about)
			} else if r.URL.Path == "/api/graphql" {
				// Return error for GraphQL requests
				w.WriteHeader(http.StatusInternalServerError)
			}
		}))
		defer server.Close()

		client := NewClient(server.URL)
		info, err := client.HealthCheck()

		// Should still succeed even if extension/manga count fails
		require.NoError(t, err)
		assert.True(t, info.IsHealthy)
		assert.Equal(t, "v0.7.0", info.Version)
		// Extension and manga counts should be 0 (default)
		assert.Equal(t, 0, info.ExtensionCount)
		assert.Equal(t, 0, info.MangaCount)
	})
}

func TestClient_Ping(t *testing.T) {
	t.Run("successful ping", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v1/settings/about", r.URL.Path)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(AboutResponse{
				Name:    "Suwayomi-Server",
				Version: "v0.7.0",
			})
		}))
		defer server.Close()

		client := NewClient(server.URL)
		result := client.Ping()

		assert.True(t, result)
	})

	t.Run("empty base URL", func(t *testing.T) {
		client := &Client{
			BaseURL:    "",
			HTTPClient: &http.Client{},
		}

		result := client.Ping()

		assert.False(t, result)
	})

	t.Run("server unreachable", func(t *testing.T) {
		client := NewClient("http://localhost:19999") // Non-existent port

		result := client.Ping()

		assert.False(t, result)
	})

	t.Run("server returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		result := client.Ping()

		assert.False(t, result)
	})
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
