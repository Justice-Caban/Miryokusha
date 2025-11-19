package suwayomi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// GraphQLClient represents a GraphQL client for Suwayomi
type GraphQLClient struct {
	client   *Client
	endpoint string
}

// NewGraphQLClient creates a new GraphQL client
func NewGraphQLClient(client *Client) *GraphQLClient {
	return &GraphQLClient{
		client:   client,
		endpoint: client.BaseURL + "/api/graphql",
	}
}

// GraphQLRequest represents a GraphQL request
type GraphQLRequest struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName,omitempty"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
}

// GraphQLResponse represents a GraphQL response
type GraphQLResponse struct {
	Data   json.RawMessage   `json:"data"`
	Errors []GraphQLError    `json:"errors,omitempty"`
}

// GraphQLError represents a GraphQL error
type GraphQLError struct {
	Message    string                 `json:"message"`
	Locations  []GraphQLErrorLocation `json:"locations,omitempty"`
	Path       []interface{}          `json:"path,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// GraphQLErrorLocation represents the location of a GraphQL error
type GraphQLErrorLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// Query executes a GraphQL query
func (gc *GraphQLClient) Query(query string, variables map[string]interface{}, result interface{}) error {
	req := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	return gc.execute(req, result)
}

// Mutate executes a GraphQL mutation
func (gc *GraphQLClient) Mutate(mutation string, variables map[string]interface{}, result interface{}) error {
	req := GraphQLRequest{
		Query:     mutation,
		Variables: variables,
	}

	return gc.execute(req, result)
}

// execute performs the GraphQL request
func (gc *GraphQLClient) execute(req GraphQLRequest, result interface{}) error {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", gc.endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := gc.client.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var gqlResp GraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if len(gqlResp.Errors) > 0 {
		return fmt.Errorf("GraphQL error: %s", gqlResp.Errors[0].Message)
	}

	if result != nil {
		if err := json.Unmarshal(gqlResp.Data, result); err != nil {
			return fmt.Errorf("failed to unmarshal data: %w", err)
		}
	}

	return nil
}

// MangaListResponse represents the response from the manga list query
type MangaListResponse struct {
	Mangas struct {
		Nodes []MangaNode `json:"nodes"`
		TotalCount int     `json:"totalCount"`
	} `json:"mangas"`
}

// MangaNode represents a manga in the GraphQL response
type MangaNode struct {
	ID             int      `json:"id"`
	Title          string   `json:"title"`
	ThumbnailURL   string   `json:"thumbnailUrl"`
	InLibrary      bool     `json:"inLibrary"`
	UnreadCount    int      `json:"unreadCount"`
	DownloadCount  int      `json:"downloadCount"`
	Chapters       struct {
		TotalCount int `json:"totalCount"`
	} `json:"chapters"`
	LatestUploadedChapter *ChapterNode `json:"latestUploadedChapter"`
	Source         *SourceNode `json:"source"`
}

// GetChapterCount returns the total chapter count for convenience
func (m *MangaNode) GetChapterCount() int {
	return m.Chapters.TotalCount
}

// ChapterNode represents a chapter in the GraphQL response
type ChapterNode struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	ChapterNumber float64 `json:"chapterNumber"`
	UploadDate    int64   `json:"uploadDate"`
	IsRead        bool    `json:"isRead"`
	IsBookmarked  bool    `json:"isBookmarked"`
	IsDownloaded  bool    `json:"isDownloaded"`
	PageCount     int     `json:"pageCount"`
}

// SourceNode represents a source in the GraphQL response
type SourceNode struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Lang        string `json:"lang"`
	IconURL     string `json:"iconUrl"`
	IsNsfw      bool   `json:"isNsfw"`
}

// ExtensionNode represents an extension in the GraphQL response
type ExtensionNode struct {
	PkgName      string `json:"pkgName"`
	Name         string `json:"name"`
	VersionName  string `json:"versionName"`
	Lang         string `json:"lang"`
	IsInstalled  bool   `json:"isInstalled"`
	HasUpdate    bool   `json:"hasUpdate"`
	IsObsolete   bool   `json:"isObsolete"`
	IsNsfw       bool   `json:"isNsfw"`
	IconURL      string `json:"iconUrl"`
}

// GetMangaList retrieves the manga library using GraphQL
func (gc *GraphQLClient) GetMangaList(inLibrary bool, limit int, offset int) (*MangaListResponse, error) {
	query := `
		query GetMangaList($inLibrary: Boolean, $first: Int, $offset: Int) {
			mangas(condition: {inLibrary: $inLibrary}, first: $first, offset: $offset) {
				totalCount
				nodes {
					id
					title
					thumbnailUrl
					inLibrary
					unreadCount
					downloadCount
					chapters {
						totalCount
					}
					latestUploadedChapter {
						id
						name
						chapterNumber
						uploadDate
					}
					source {
						id
						name
						lang
						iconUrl
						isNsfw
					}
				}
			}
		}
	`

	variables := map[string]interface{}{
		"inLibrary": inLibrary,
		"first":     limit,
		"offset":    offset,
	}

	var result MangaListResponse
	if err := gc.Query(query, variables, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetMangaDetails retrieves detailed information about a manga
func (gc *GraphQLClient) GetMangaDetails(mangaID int) (*MangaNode, error) {
	query := `
		query GetManga($id: Int!) {
			manga(id: $id) {
				id
				title
				thumbnailUrl
				inLibrary
				unreadCount
				downloadCount
				chapters {
					totalCount
				}
				source {
					id
					name
					lang
					iconUrl
					isNsfw
				}
			}
		}
	`

	variables := map[string]interface{}{
		"id": mangaID,
	}

	var result struct {
		Manga MangaNode `json:"manga"`
	}

	if err := gc.Query(query, variables, &result); err != nil {
		return nil, err
	}

	return &result.Manga, nil
}

// GetChapterList retrieves the chapter list for a manga
func (gc *GraphQLClient) GetChapterList(mangaID int) ([]ChapterNode, error) {
	query := `
		query GetChapters($mangaId: Int!) {
			chapters(condition: {mangaId: $mangaId}, orderBy: CHAPTER_NUMBER_DESC) {
				nodes {
					id
					name
					chapterNumber
					uploadDate
					isRead
					isBookmarked
					isDownloaded
					pageCount
				}
			}
		}
	`

	variables := map[string]interface{}{
		"mangaId": mangaID,
	}

	var result struct {
		Chapters struct {
			Nodes []ChapterNode `json:"nodes"`
		} `json:"chapters"`
	}

	if err := gc.Query(query, variables, &result); err != nil {
		return nil, err
	}

	return result.Chapters.Nodes, nil
}

// UpdateChapter updates a chapter's read/bookmark status
func (gc *GraphQLClient) UpdateChapter(chapterID int, isRead *bool, isBookmarked *bool) error {
	mutation := `
		mutation UpdateChapter($input: UpdateChapterInput!) {
			updateChapter(input: $input) {
				chapter {
					id
					isRead
					isBookmarked
				}
			}
		}
	`

	input := map[string]interface{}{
		"id": chapterID,
	}

	if isRead != nil {
		input["isRead"] = *isRead
	}

	if isBookmarked != nil {
		input["isBookmarked"] = *isBookmarked
	}

	variables := map[string]interface{}{
		"input": input,
	}

	return gc.Mutate(mutation, variables, nil)
}

// AddMangaToLibrary adds a manga to the library
func (gc *GraphQLClient) AddMangaToLibrary(mangaID int) error {
	mutation := `
		mutation UpdateManga($input: UpdateMangaInput!) {
			updateManga(input: $input) {
				manga {
					id
					inLibrary
				}
			}
		}
	`

	variables := map[string]interface{}{
		"input": map[string]interface{}{
			"id":        mangaID,
			"inLibrary": true,
		},
	}

	return gc.Mutate(mutation, variables, nil)
}

// RemoveMangaFromLibrary removes a manga from the library
func (gc *GraphQLClient) RemoveMangaFromLibrary(mangaID int) error {
	mutation := `
		mutation UpdateManga($input: UpdateMangaInput!) {
			updateManga(input: $input) {
				manga {
					id
					inLibrary
				}
			}
		}
	`

	variables := map[string]interface{}{
		"input": map[string]interface{}{
			"id":        mangaID,
			"inLibrary": false,
		},
	}

	return gc.Mutate(mutation, variables, nil)
}

// GetExtensionList retrieves the list of available extensions
func (gc *GraphQLClient) GetExtensionList() ([]ExtensionNode, error) {
	query := `
		query GetExtensions {
			extensions {
				nodes {
					pkgName
					name
					versionName
					lang
					isInstalled
					hasUpdate
					isObsolete
					isNsfw
					iconUrl
				}
			}
		}
	`

	var result struct {
		Extensions struct {
			Nodes []ExtensionNode `json:"nodes"`
		} `json:"extensions"`
	}

	if err := gc.Query(query, nil, &result); err != nil {
		return nil, err
	}

	return result.Extensions.Nodes, nil
}

// InstallExtension installs an extension
func (gc *GraphQLClient) InstallExtension(pkgName string) error {
	mutation := `
		mutation InstallExtension($input: InstallExternalExtensionInput!) {
			installExternalExtension(input: $input) {
				extension {
					pkgName
					isInstalled
				}
			}
		}
	`

	variables := map[string]interface{}{
		"input": map[string]interface{}{
			"pkgName": pkgName,
		},
	}

	return gc.Mutate(mutation, variables, nil)
}

// UninstallExtension uninstalls an extension
func (gc *GraphQLClient) UninstallExtension(pkgName string) error {
	mutation := `
		mutation UninstallExtension($input: UninstallExtensionInput!) {
			uninstallExtension(input: $input) {
				extension {
					pkgName
					isInstalled
				}
			}
		}
	`

	variables := map[string]interface{}{
		"input": map[string]interface{}{
			"pkgName": pkgName,
		},
	}

	return gc.Mutate(mutation, variables, nil)
}

// UpdateExtension updates an extension
func (gc *GraphQLClient) UpdateExtension(pkgName string) error {
	mutation := `
		mutation UpdateExtension($input: UpdateExtensionInput!) {
			updateExtension(input: $input) {
				extension {
					pkgName
					hasUpdate
				}
			}
		}
	`

	variables := map[string]interface{}{
		"input": map[string]interface{}{
			"pkgName": pkgName,
		},
	}

	return gc.Mutate(mutation, variables, nil)
}
