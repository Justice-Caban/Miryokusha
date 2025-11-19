package suwayomi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGraphQLClient_Query(t *testing.T) {
	tests := []struct {
		name           string
		responseBody   interface{}
		responseStatus int
		expectError    bool
		errorContains  string
	}{
		{
			name: "successful query",
			responseBody: GraphQLResponse{
				Data: json.RawMessage(`{"test":"value"}`),
			},
			responseStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name: "graphql error response",
			responseBody: GraphQLResponse{
				Errors: []GraphQLError{
					{Message: "field not found"},
				},
			},
			responseStatus: http.StatusOK,
			expectError:    true,
			errorContains:  "field not found",
		},
		{
			name:           "http error status",
			responseBody:   nil,
			responseStatus: http.StatusInternalServerError,
			expectError:    true,
			errorContains:  "unexpected status code",
		},
		{
			name:           "malformed json",
			responseBody:   "not json",
			responseStatus: http.StatusOK,
			expectError:    true,
			errorContains:  "failed to decode response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method and headers
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				assert.Equal(t, "application/json", r.Header.Get("Accept"))

				w.WriteHeader(tt.responseStatus)
				if tt.responseBody != nil {
					if str, ok := tt.responseBody.(string); ok {
						w.Write([]byte(str))
					} else {
						json.NewEncoder(w).Encode(tt.responseBody)
					}
				}
			}))
			defer server.Close()

			// Create client
			client := NewClient(server.URL)
			gc := client.GraphQL

			// Execute query
			var result map[string]interface{}
			err := gc.Query("query { test }", nil, &result)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, "value", result["test"])
			}
		})
	}
}

func TestGraphQLClient_Mutate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify it's a POST request
		assert.Equal(t, "POST", r.Method)

		// Parse request body
		var req GraphQLRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Verify mutation is present
		assert.Contains(t, req.Query, "mutation")

		// Return success response
		resp := GraphQLResponse{
			Data: json.RawMessage(`{"updateManga":{"manga":{"id":1,"inLibrary":true}}}`),
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	gc := client.GraphQL

	mutation := `mutation UpdateManga($input: UpdateMangaInput!) {
		updateManga(input: $input) {
			manga {
				id
				inLibrary
			}
		}
	}`

	variables := map[string]interface{}{
		"input": map[string]interface{}{
			"id":        1,
			"inLibrary": true,
		},
	}

	var result map[string]interface{}
	err := gc.Mutate(mutation, variables, &result)
	require.NoError(t, err)
	assert.NotNil(t, result["updateManga"])
}

func TestGraphQLClient_GetMangaList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req GraphQLRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Verify query contains correct fields
		assert.Contains(t, req.Query, "GetMangaList")
		assert.Contains(t, req.Query, "mangas")
		assert.Contains(t, req.Query, "inLibrary")

		// Verify variables
		assert.Equal(t, true, req.Variables["inLibrary"])
		assert.Equal(t, float64(10), req.Variables["first"]) // JSON numbers are float64
		assert.Equal(t, float64(0), req.Variables["offset"])

		// Return mock manga list
		mockResponse := GraphQLResponse{
			Data: json.RawMessage(`{
				"mangas": {
					"totalCount": 2,
					"nodes": [
						{
							"id": 1,
							"title": "Test Manga 1",
							"thumbnailUrl": "http://example.com/thumb1.jpg",
							"inLibrary": true,
							"unreadCount": 5,
							"downloadCount": 10,
							"chapters": {
								"totalCount": 15
							},
							"source": {
								"id": "source1",
								"name": "Test Source",
								"lang": "en",
								"iconUrl": "http://example.com/icon.jpg",
								"isNsfw": false
							}
						},
						{
							"id": 2,
							"title": "Test Manga 2",
							"thumbnailUrl": "http://example.com/thumb2.jpg",
							"inLibrary": true,
							"unreadCount": 3,
							"downloadCount": 8,
							"chapters": {
								"totalCount": 12
							}
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
	gc := client.GraphQL

	result, err := gc.GetMangaList(true, 10, 0)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 2, result.Mangas.TotalCount)
	assert.Len(t, result.Mangas.Nodes, 2)

	// Verify first manga
	manga1 := result.Mangas.Nodes[0]
	assert.Equal(t, 1, manga1.ID)
	assert.Equal(t, "Test Manga 1", manga1.Title)
	assert.Equal(t, "http://example.com/thumb1.jpg", manga1.ThumbnailURL)
	assert.True(t, manga1.InLibrary)
	assert.Equal(t, 5, manga1.UnreadCount)
	assert.Equal(t, 10, manga1.DownloadCount)
	assert.Equal(t, 15, manga1.GetChapterCount())
	require.NotNil(t, manga1.Source)
	assert.Equal(t, "source1", manga1.Source.ID)
	assert.Equal(t, "Test Source", manga1.Source.Name)

	// Verify second manga
	manga2 := result.Mangas.Nodes[1]
	assert.Equal(t, 2, manga2.ID)
	assert.Equal(t, "Test Manga 2", manga2.Title)
}

func TestGraphQLClient_GetMangaDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req GraphQLRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Verify query and variables
		assert.Contains(t, req.Query, "GetManga")
		assert.Equal(t, float64(42), req.Variables["id"])

		mockResponse := GraphQLResponse{
			Data: json.RawMessage(`{
				"manga": {
					"id": 42,
					"title": "Detailed Manga",
					"thumbnailUrl": "http://example.com/detailed.jpg",
					"inLibrary": true,
					"unreadCount": 7,
					"downloadCount": 12,
					"chapters": {
						"totalCount": 20
					},
					"source": {
						"id": "detailed-source",
						"name": "Detailed Source",
						"lang": "ja",
						"iconUrl": "http://example.com/icon-detailed.jpg",
						"isNsfw": false
					}
				}
			}`),
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	gc := client.GraphQL

	manga, err := gc.GetMangaDetails(42)
	require.NoError(t, err)
	require.NotNil(t, manga)

	assert.Equal(t, 42, manga.ID)
	assert.Equal(t, "Detailed Manga", manga.Title)
	assert.Equal(t, "http://example.com/detailed.jpg", manga.ThumbnailURL)
	assert.True(t, manga.InLibrary)
	assert.Equal(t, 7, manga.UnreadCount)
	assert.Equal(t, 12, manga.DownloadCount)
	assert.Equal(t, 20, manga.GetChapterCount())
	require.NotNil(t, manga.Source)
	assert.Equal(t, "detailed-source", manga.Source.ID)
	assert.Equal(t, "ja", manga.Source.Lang)
}

func TestGraphQLClient_GetChapterList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req GraphQLRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Verify query
		assert.Contains(t, req.Query, "GetChapters")
		assert.Contains(t, req.Query, "CHAPTER_NUMBER_DESC")
		assert.Equal(t, float64(1), req.Variables["mangaId"])

		mockResponse := GraphQLResponse{
			Data: json.RawMessage(`{
				"chapters": {
					"nodes": [
						{
							"id": 101,
							"name": "Chapter 10",
							"chapterNumber": 10.0,
							"uploadDate": "1700000000000",
							"isRead": false,
							"isBookmarked": true,
							"isDownloaded": false,
							"pageCount": 25
						},
						{
							"id": 102,
							"name": "Chapter 9",
							"chapterNumber": 9.0,
							"uploadDate": "1699000000000",
							"isRead": true,
							"isBookmarked": false,
							"isDownloaded": true,
							"pageCount": 30
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
	gc := client.GraphQL

	chapters, err := gc.GetChapterList(1)
	require.NoError(t, err)
	assert.Len(t, chapters, 2)

	// Verify first chapter (should be sorted descending)
	ch1 := chapters[0]
	assert.Equal(t, 101, ch1.ID)
	assert.Equal(t, "Chapter 10", ch1.Name)
	assert.Equal(t, 10.0, ch1.ChapterNumber)
	assert.False(t, ch1.IsRead)
	assert.True(t, ch1.IsBookmarked)
	assert.False(t, ch1.IsDownloaded)
	assert.Equal(t, 25, ch1.PageCount)

	// Verify second chapter
	ch2 := chapters[1]
	assert.Equal(t, 102, ch2.ID)
	assert.Equal(t, "Chapter 9", ch2.Name)
	assert.Equal(t, 9.0, ch2.ChapterNumber)
	assert.True(t, ch2.IsRead)
	assert.False(t, ch2.IsBookmarked)
	assert.True(t, ch2.IsDownloaded)
	assert.Equal(t, 30, ch2.PageCount)
}

func TestGraphQLClient_UpdateChapter(t *testing.T) {
	tests := []struct {
		name         string
		isRead       *bool
		isBookmarked *bool
	}{
		{
			name:         "mark as read",
			isRead:       boolPtr(true),
			isBookmarked: nil,
		},
		{
			name:         "mark as unread and bookmarked",
			isRead:       boolPtr(false),
			isBookmarked: boolPtr(true),
		},
		{
			name:         "bookmark only",
			isRead:       nil,
			isBookmarked: boolPtr(true),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req GraphQLRequest
				json.NewDecoder(r.Body).Decode(&req)

				// Verify mutation
				assert.Contains(t, req.Query, "UpdateChapter")

				// Verify input
				input := req.Variables["input"].(map[string]interface{})
				assert.Equal(t, float64(123), input["id"])

				if tt.isRead != nil {
					assert.Equal(t, *tt.isRead, input["isRead"])
				} else {
					assert.NotContains(t, input, "isRead")
				}

				if tt.isBookmarked != nil {
					assert.Equal(t, *tt.isBookmarked, input["isBookmarked"])
				} else {
					assert.NotContains(t, input, "isBookmarked")
				}

				// Return success
				mockResponse := GraphQLResponse{
					Data: json.RawMessage(`{"updateChapter":{"chapter":{"id":123}}}`),
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(mockResponse)
			}))
			defer server.Close()

			client := NewClient(server.URL)
			gc := client.GraphQL

			err := gc.UpdateChapter(123, tt.isRead, tt.isBookmarked)
			require.NoError(t, err)
		})
	}
}

func TestGraphQLClient_AddMangaToLibrary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req GraphQLRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Verify mutation
		assert.Contains(t, req.Query, "UpdateManga")

		// Verify variables
		input := req.Variables["input"].(map[string]interface{})
		assert.Equal(t, float64(456), input["id"])
		assert.Equal(t, true, input["inLibrary"])

		mockResponse := GraphQLResponse{
			Data: json.RawMessage(`{"updateManga":{"manga":{"id":456,"inLibrary":true}}}`),
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	gc := client.GraphQL

	err := gc.AddMangaToLibrary(456)
	require.NoError(t, err)
}

func TestGraphQLClient_RemoveMangaFromLibrary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req GraphQLRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Verify mutation
		assert.Contains(t, req.Query, "UpdateManga")

		// Verify variables
		input := req.Variables["input"].(map[string]interface{})
		assert.Equal(t, float64(789), input["id"])
		assert.Equal(t, false, input["inLibrary"])

		mockResponse := GraphQLResponse{
			Data: json.RawMessage(`{"updateManga":{"manga":{"id":789,"inLibrary":false}}}`),
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	gc := client.GraphQL

	err := gc.RemoveMangaFromLibrary(789)
	require.NoError(t, err)
}

func TestGraphQLClient_GetExtensionList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req GraphQLRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Verify query
		assert.Contains(t, req.Query, "GetExtensions")

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
							"iconUrl": "http://example.com/icon-mangadex.png"
						},
						{
							"pkgName": "eu.kanade.tachiyomi.extension.en.mangasee",
							"name": "MangaSee",
							"versionName": "2.1.0",
							"lang": "en",
							"isInstalled": false,
							"hasUpdate": false,
							"isObsolete": false,
							"isNsfw": false,
							"iconUrl": "http://example.com/icon-mangasee.png"
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
	gc := client.GraphQL

	extensions, err := gc.GetExtensionList()
	require.NoError(t, err)
	assert.Len(t, extensions, 2)

	// Verify first extension (installed)
	ext1 := extensions[0]
	assert.Equal(t, "eu.kanade.tachiyomi.extension.en.mangadex", ext1.PkgName)
	assert.Equal(t, "MangaDex", ext1.Name)
	assert.Equal(t, "1.4.2", ext1.VersionName)
	assert.Equal(t, "en", ext1.Lang)
	assert.True(t, ext1.IsInstalled)
	assert.False(t, ext1.HasUpdate)
	assert.False(t, ext1.IsObsolete)
	assert.False(t, ext1.IsNsfw)

	// Verify second extension (not installed)
	ext2 := extensions[1]
	assert.Equal(t, "eu.kanade.tachiyomi.extension.en.mangasee", ext2.PkgName)
	assert.False(t, ext2.IsInstalled)
}

func TestGraphQLClient_ExtensionManagement(t *testing.T) {
	t.Run("install extension", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req GraphQLRequest
			json.NewDecoder(r.Body).Decode(&req)

			assert.Contains(t, req.Query, "InstallExtension")
			input := req.Variables["input"].(map[string]interface{})
			assert.Equal(t, "test.extension.pkg", input["pkgName"])

			mockResponse := GraphQLResponse{
				Data: json.RawMessage(`{"installExternalExtension":{"extension":{"pkgName":"test.extension.pkg","isInstalled":true}}}`),
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		gc := client.GraphQL

		err := gc.InstallExtension("test.extension.pkg")
		require.NoError(t, err)
	})

	t.Run("uninstall extension", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req GraphQLRequest
			json.NewDecoder(r.Body).Decode(&req)

			assert.Contains(t, req.Query, "UninstallExtension")
			input := req.Variables["input"].(map[string]interface{})
			assert.Equal(t, "test.extension.pkg", input["pkgName"])

			mockResponse := GraphQLResponse{
				Data: json.RawMessage(`{"uninstallExtension":{"extension":{"pkgName":"test.extension.pkg","isInstalled":false}}}`),
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		gc := client.GraphQL

		err := gc.UninstallExtension("test.extension.pkg")
		require.NoError(t, err)
	})

	t.Run("update extension", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req GraphQLRequest
			json.NewDecoder(r.Body).Decode(&req)

			assert.Contains(t, req.Query, "UpdateExtension")
			input := req.Variables["input"].(map[string]interface{})
			assert.Equal(t, "test.extension.pkg", input["pkgName"])

			mockResponse := GraphQLResponse{
				Data: json.RawMessage(`{"updateExtension":{"extension":{"pkgName":"test.extension.pkg","hasUpdate":false}}}`),
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		gc := client.GraphQL

		err := gc.UpdateExtension("test.extension.pkg")
		require.NoError(t, err)
	})
}

// Helper function to create bool pointer
func boolPtr(b bool) *bool {
	return &b
}
