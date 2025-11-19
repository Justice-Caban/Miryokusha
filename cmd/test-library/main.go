package main

import (
	"fmt"
	"os"

	"github.com/Justice-Caban/Miryokusha/internal/source"
	"github.com/Justice-Caban/Miryokusha/internal/suwayomi"
)

func main() {
	// Test direct API call
	serverURL := "http://localhost:4567"
	if len(os.Args) > 1 {
		serverURL = os.Args[1]
	}

	fmt.Printf("Testing connection to: %s\n\n", serverURL)

	// Create client
	client := suwayomi.NewClient(serverURL)

	// Test health check
	fmt.Println("1. Testing health check...")
	info, err := client.HealthCheck()
	if err != nil {
		fmt.Printf("   ❌ Health check failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   ✓ Server version: %s\n", info.Version)
	fmt.Printf("   ✓ Build type: %s\n", info.BuildType)
	fmt.Printf("   ✓ Extension count: %d\n", info.ExtensionCount)
	fmt.Printf("   ✓ Manga count: %d\n\n", info.MangaCount)

	// Test GetMangaList via GraphQL
	fmt.Println("2. Testing GetMangaList (GraphQL)...")
	result, err := client.GraphQL.GetMangaList(true, 10, 0)
	if err != nil {
		fmt.Printf("   ❌ GetMangaList failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   ✓ Total count: %d\n", result.Mangas.TotalCount)
	fmt.Printf("   ✓ Fetched: %d manga\n", len(result.Mangas.Nodes))

	if len(result.Mangas.Nodes) > 0 {
		fmt.Println("\n   First manga:")
		manga := result.Mangas.Nodes[0]
		fmt.Printf("     ID: %d\n", manga.ID)
		fmt.Printf("     Title: '%s'\n", manga.Title)
		fmt.Printf("     ThumbnailURL: %s\n", manga.ThumbnailURL)
		fmt.Printf("     InLibrary: %v\n", manga.InLibrary)
		fmt.Printf("     UnreadCount: %d\n", manga.UnreadCount)
		fmt.Printf("     ChapterCount: %d\n", manga.GetChapterCount())
		if manga.Source != nil {
			fmt.Printf("     Source: %s (%s)\n", manga.Source.Name, manga.Source.Lang)
		}
	}

	// Test SuwayomiSource wrapper
	fmt.Println("\n3. Testing SuwayomiSource wrapper...")
	src := source.NewSuwayomiSource("test-source", "Test Server", serverURL)

	mangaList, err := src.ListManga()
	if err != nil {
		fmt.Printf("   ❌ ListManga failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   ✓ Fetched: %d manga\n", len(mangaList))

	if len(mangaList) > 0 {
		fmt.Println("\n   First manga:")
		manga := mangaList[0]
		fmt.Printf("     ID: %s\n", manga.ID)
		fmt.Printf("     Title: '%s'\n", manga.Title)
		fmt.Printf("     CoverURL: %s\n", manga.CoverURL)
		fmt.Printf("     InLibrary: %v\n", manga.InLibrary)
		fmt.Printf("     UnreadCount: %d\n", manga.UnreadCount)
		fmt.Printf("     ChapterCount: %d\n", manga.ChapterCount)
		fmt.Printf("     SourceType: %s\n", manga.SourceType)
		fmt.Printf("     SourceName: %s\n", manga.SourceName)
	}

	fmt.Println("\n✅ All tests passed!")
}
