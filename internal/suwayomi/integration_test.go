// +build integration

package suwayomi

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration tests that run against a real Suwayomi server
// Run with: go test -tags=integration -v ./internal/suwayomi/...
//
// Set environment variable SUWAYOMI_URL before running:
// export SUWAYOMI_URL="http://localhost:4567"

func getTestClient(t *testing.T) *Client {
	url := os.Getenv("SUWAYOMI_URL")
	if url == "" {
		t.Skip("Skipping integration test: SUWAYOMI_URL not set")
	}
	return NewClient(url)
}

func TestIntegration_ServerPing(t *testing.T) {
	client := getTestClient(t)

	result := client.Ping()
	assert.True(t, result, "Server should be reachable")
}

func TestIntegration_HealthCheck(t *testing.T) {
	client := getTestClient(t)

	info, err := client.HealthCheck()
	require.NoError(t, err)
	require.NotNil(t, info)

	assert.True(t, info.IsHealthy)
	assert.NotEmpty(t, info.Version)
	t.Logf("Server Version: %s", info.Version)
	t.Logf("Build Type: %s", info.BuildType)
	t.Logf("Revision: %s", info.Revision)
	t.Logf("Extension Count: %d", info.ExtensionCount)
	t.Logf("Manga Count: %d", info.MangaCount)
}

func TestIntegration_ExtensionList(t *testing.T) {
	client := getTestClient(t)

	extensions, err := client.ListAvailableExtensions()
	require.NoError(t, err)
	require.NotNil(t, extensions)

	t.Logf("Found %d extensions", len(extensions))

	if len(extensions) > 0 {
		ext := extensions[0]
		t.Logf("Sample extension:")
		t.Logf("  PkgName: %s", ext.PkgName)
		t.Logf("  Name: %s", ext.Name)
		t.Logf("  Version: %s", ext.VersionName)
		t.Logf("  Language: %s", ext.Language)
		t.Logf("  Installed: %t", ext.IsInstalled)
		t.Logf("  NSFW: %t", ext.IsNSFW)

		// Verify expected fields are populated
		assert.NotEmpty(t, ext.PkgName)
		assert.NotEmpty(t, ext.Name)
	}
}

func TestIntegration_InstalledExtensions(t *testing.T) {
	client := getTestClient(t)

	installed, err := client.ListInstalledExtensions()
	require.NoError(t, err)
	require.NotNil(t, installed)

	t.Logf("Found %d installed extensions", len(installed))

	// All returned extensions should be marked as installed
	for _, ext := range installed {
		assert.True(t, ext.IsInstalled, "Extension %s should be marked as installed", ext.Name)
	}
}

func TestIntegration_MangaList(t *testing.T) {
	client := getTestClient(t)

	// Query first 10 manga in library
	result, err := client.GraphQL.GetMangaList(true, 10, 0)
	require.NoError(t, err)
	require.NotNil(t, result)

	t.Logf("Total manga in library: %d", result.Mangas.TotalCount)
	t.Logf("Fetched: %d manga", len(result.Mangas.Nodes))

	if len(result.Mangas.Nodes) > 0 {
		manga := result.Mangas.Nodes[0]
		t.Logf("Sample manga:")
		t.Logf("  ID: %d", manga.ID)
		t.Logf("  Title: %s", manga.Title)
		t.Logf("  Unread Count: %d", manga.UnreadCount)
		t.Logf("  Chapter Count: %d", manga.GetChapterCount())

		// Verify expected fields
		assert.NotZero(t, manga.ID)
		assert.NotEmpty(t, manga.Title)
	}
}

func TestIntegration_ChapterList(t *testing.T) {
	client := getTestClient(t)

	// First get a manga from library
	mangaResult, err := client.GraphQL.GetMangaList(true, 1, 0)
	require.NoError(t, err)

	if len(mangaResult.Mangas.Nodes) == 0 {
		t.Skip("No manga in library to test chapters")
	}

	mangaID := mangaResult.Mangas.Nodes[0].ID
	t.Logf("Testing chapters for manga ID: %d", mangaID)

	// Get chapters for this manga
	chapters, err := client.GraphQL.GetChapterList(mangaID)
	require.NoError(t, err)
	require.NotNil(t, chapters)

	t.Logf("Found %d chapters", len(chapters))

	if len(chapters) > 0 {
		chapter := chapters[0]
		t.Logf("Sample chapter:")
		t.Logf("  ID: %d", chapter.ID)
		t.Logf("  Name: %s", chapter.Name)
		t.Logf("  Chapter Number: %.1f", chapter.ChapterNumber)
		t.Logf("  Is Read: %t", chapter.IsRead)
		t.Logf("  Page Count: %d", chapter.PageCount)

		// Verify expected fields
		assert.NotZero(t, chapter.ID)
		assert.NotEmpty(t, chapter.Name)
	}
}

func TestIntegration_MangaDetails(t *testing.T) {
	client := getTestClient(t)

	// First get a manga from library
	mangaResult, err := client.GraphQL.GetMangaList(true, 1, 0)
	require.NoError(t, err)

	if len(mangaResult.Mangas.Nodes) == 0 {
		t.Skip("No manga in library to test details")
	}

	mangaID := mangaResult.Mangas.Nodes[0].ID
	t.Logf("Testing details for manga ID: %d", mangaID)

	// Get detailed manga info
	manga, err := client.GraphQL.GetMangaDetails(mangaID)
	require.NoError(t, err)
	require.NotNil(t, manga)

	t.Logf("Manga details:")
	t.Logf("  ID: %d", manga.ID)
	t.Logf("  Title: %s", manga.Title)
	t.Logf("  In Library: %t", manga.InLibrary)
	t.Logf("  Unread Count: %d", manga.UnreadCount)
	t.Logf("  Chapter Count: %d", manga.GetChapterCount())

	if manga.Source != nil {
		t.Logf("  Source: %s (%s)", manga.Source.Name, manga.Source.Lang)
	}

	// Verify expected fields
	assert.NotZero(t, manga.ID)
	assert.NotEmpty(t, manga.Title)
	assert.True(t, manga.InLibrary)
}

// Note: Mutation tests (UpdateChapter, AddMangaToLibrary, etc.) are commented out
// because they modify server state. Uncomment and run manually if needed.

/*
func TestIntegration_UpdateChapter(t *testing.T) {
	client := getTestClient(t)

	// WARNING: This modifies server state!
	// Get a chapter to test with
	mangaResult, err := client.GraphQL.GetMangaList(true, 1, 0)
	require.NoError(t, err)
	if len(mangaResult.Mangas.Nodes) == 0 {
		t.Skip("No manga in library")
	}

	mangaID := mangaResult.Mangas.Nodes[0].ID
	chapters, err := client.GraphQL.GetChapterList(mangaID)
	require.NoError(t, err)
	if len(chapters) == 0 {
		t.Skip("No chapters available")
	}

	chapterID := chapters[0].ID
	originalState := chapters[0].IsRead

	// Toggle read state
	newState := !originalState
	err = client.GraphQL.UpdateChapter(chapterID, &newState, nil)
	require.NoError(t, err)

	// Restore original state
	err = client.GraphQL.UpdateChapter(chapterID, &originalState, nil)
	require.NoError(t, err)

	t.Log("Successfully toggled chapter read state")
}
*/
