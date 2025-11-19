package source

import (
	"fmt"
	"strconv"
	"time"

	"github.com/Justice-Caban/Miryokusha/internal/suwayomi"
)

// SuwayomiSource represents a Suwayomi server source
type SuwayomiSource struct {
	id      string
	name    string
	baseURL string
	client  *suwayomi.Client
}

// NewSuwayomiSource creates a new Suwayomi source
func NewSuwayomiSource(id, name, baseURL string) *SuwayomiSource {
	return &SuwayomiSource{
		id:      id,
		name:    name,
		baseURL: baseURL,
		client:  suwayomi.NewClient(baseURL),
	}
}

// GetType returns the source type
func (s *SuwayomiSource) GetType() SourceType {
	return SourceTypeSuwayomi
}

// GetID returns the source ID
func (s *SuwayomiSource) GetID() string {
	return s.id
}

// GetName returns the source name
func (s *SuwayomiSource) GetName() string {
	return s.name
}

// ListManga lists all manga from the Suwayomi server
func (s *SuwayomiSource) ListManga() ([]*Manga, error) {
	// Use GraphQL to fetch manga list (in library)
	resp, err := s.client.GraphQL.GetMangaList(true, 1000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manga list: %w", err)
	}

	// Convert GraphQL response to source.Manga
	result := make([]*Manga, 0, len(resp.Mangas.Nodes))
	for _, node := range resp.Mangas.Nodes {
		manga := s.convertMangaNode(&node)
		result = append(result, manga)
	}

	return result, nil
}

// GetManga retrieves manga details from the Suwayomi server
func (s *SuwayomiSource) GetManga(mangaID string) (*Manga, error) {
	// Convert string ID to int
	id, err := strconv.Atoi(mangaID)
	if err != nil {
		return nil, fmt.Errorf("invalid manga ID: %w", err)
	}

	// Use GraphQL to fetch manga details
	node, err := s.client.GraphQL.GetMangaDetails(id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manga: %w", err)
	}

	return s.convertMangaNode(node), nil
}

// ListChapters lists all chapters for a manga
func (s *SuwayomiSource) ListChapters(mangaID string) ([]*Chapter, error) {
	// Convert string ID to int
	id, err := strconv.Atoi(mangaID)
	if err != nil {
		return nil, fmt.Errorf("invalid manga ID: %w", err)
	}

	// Use GraphQL to fetch chapters
	nodes, err := s.client.GraphQL.GetChapterList(id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch chapters: %w", err)
	}

	// Convert GraphQL response to source.Chapter
	result := make([]*Chapter, 0, len(nodes))
	for _, node := range nodes {
		chapter := s.convertChapterNode(&node, mangaID)
		result = append(result, chapter)
	}

	return result, nil
}

// GetChapter retrieves chapter details
func (s *SuwayomiSource) GetChapter(chapterID string) (*Chapter, error) {
	// GraphQL doesn't have a single chapter query, so we need to get it from the chapter list
	// We'll need to extract the manga ID from somewhere or iterate through all manga
	// For now, return an error indicating this needs the manga ID
	return nil, fmt.Errorf("GetChapter requires manga context - use ListChapters instead")
}

// GetPage retrieves a specific page from a chapter
func (s *SuwayomiSource) GetPage(chapter *Chapter, pageIndex int) (*Page, error) {
	// Use REST API for page image retrieval
	// Correct Suwayomi API endpoint includes manga ID
	url := fmt.Sprintf("%s/api/v1/manga/%s/chapter/%s/page/%d",
		s.client.BaseURL, chapter.MangaID, chapter.ID, pageIndex)

	// Fetch the image
	resp, err := s.client.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch page: status %d", resp.StatusCode)
	}

	// Read image data
	var imageData []byte
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			imageData = append(imageData, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	// Determine image type from content-type header
	imageType := "image/jpeg" // default
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		imageType = ct
	}

	return &Page{
		Index:     pageIndex,
		URL:       url,
		ImageData: imageData,
		ImageType: imageType,
	}, nil
}

// GetAllPages retrieves all pages from a chapter
func (s *SuwayomiSource) GetAllPages(chapter *Chapter) ([]*Page, error) {
	// Use the PageCount from chapter metadata if available
	maxPages := chapter.PageCount
	if maxPages == 0 {
		// Fallback to trying up to 500 pages
		maxPages = 500
	}

	var pages []*Page

	// Fetch all pages based on PageCount
	for pageIndex := 0; pageIndex < maxPages; pageIndex++ {
		page, err := s.GetPage(chapter, pageIndex)
		if err != nil {
			// If we can't get the page, assume we've reached the end
			break
		}

		pages = append(pages, page)
	}

	if len(pages) == 0 {
		return nil, fmt.Errorf("no pages found for chapter %s (manga %s)", chapter.ID, chapter.MangaID)
	}

	return pages, nil
}

// Search searches for manga on the Suwayomi server
func (s *SuwayomiSource) Search(query string) ([]*Manga, error) {
	// GraphQL search would require a custom query
	// For now, we'll fetch all manga and filter client-side
	allManga, err := s.ListManga()
	if err != nil {
		return nil, err
	}

	// Simple case-insensitive title search
	var results []*Manga
	queryLower := toLower(query)
	for _, manga := range allManga {
		if contains(toLower(manga.Title), queryLower) {
			results = append(results, manga)
		}
	}

	return results, nil
}

// IsAvailable checks if the Suwayomi server is accessible
func (s *SuwayomiSource) IsAvailable() bool {
	info, err := s.client.HealthCheck()
	return err == nil && info.IsHealthy
}

// Helper functions

// convertMangaNode converts a GraphQL MangaNode to source.Manga
func (s *SuwayomiSource) convertMangaNode(node *suwayomi.MangaNode) *Manga {
	manga := &Manga{
		ID:            strconv.Itoa(node.ID),
		Title:         node.Title,
		Author:        "",   // Not in basic query
		Artist:        "",   // Not in basic query
		Description:   "",   // Not in basic query
		Genres:        nil,  // Not in basic query
		Status:        "",   // Not in basic query
		CoverURL:      node.ThumbnailURL,
		SourceType:    SourceTypeSuwayomi,
		SourceID:      s.id,
		URL:           fmt.Sprintf("%s/manga/%d", s.baseURL, node.ID),
		InLibrary:     node.InLibrary,
		UnreadCount:   node.UnreadCount,
		DownloadCount: node.DownloadCount,
		ChapterCount:  node.GetChapterCount(),
		// LastReadAt tracked locally, not provided by Suwayomi schema
	}

	// Add source info if available
	if node.Source != nil {
		manga.SourceName = node.Source.Name
	}

	return manga
}

// convertChapterNode converts a GraphQL ChapterNode to source.Chapter
func (s *SuwayomiSource) convertChapterNode(node *suwayomi.ChapterNode, mangaID string) *Chapter {
	// Parse uploadDate - it's a Unix timestamp in milliseconds as a string
	uploadDateMs, err := strconv.ParseInt(node.UploadDate, 10, 64)
	var uploadDate time.Time
	if err == nil {
		uploadDate = time.Unix(uploadDateMs/1000, 0)
	}

	return &Chapter{
		ID:             strconv.Itoa(node.ID),
		MangaID:        mangaID,
		Title:          node.Name,
		ChapterNumber:  node.ChapterNumber,
		VolumeNumber:   0,  // Not in basic query
		ScanlatorGroup: "", // Not in basic query
		UploadDate:     uploadDate,
		IsRead:         node.IsRead,
		IsBookmarked:   node.IsBookmarked,
		IsDownloaded:   node.IsDownloaded,
		PageCount:      node.PageCount,
	}
}

// Helper string functions
func toLower(s string) string {
	result := make([]rune, len(s))
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			result[i] = r + 32
		} else {
			result[i] = r
		}
	}
	return string(result)
}

func contains(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
