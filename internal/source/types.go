package source

import (
	"fmt"
	"time"
)

// SourceType represents the type of manga source
type SourceType string

const (
	SourceTypeSuwayomi SourceType = "suwayomi"
	SourceTypeLocal    SourceType = "local"
)

// Manga represents manga metadata
type Manga struct {
	ID             string
	Title          string
	Author         string
	Artist         string
	Description    string
	Status         string
	CoverURL       string
	Genres         []string
	SourceType     SourceType
	SourceID       string
	SourceName     string
	URL            string
	InLibrary      bool
	UnreadCount    int
	DownloadCount  int
	ChapterCount   int
	LastReadAt     *time.Time
}

// Chapter represents a manga chapter
type Chapter struct {
	ID             string
	MangaID        string
	Title          string
	ChapterNumber  float64
	VolumeNumber   float64
	PageCount      int
	ScanlatorGroup string
	UploadDate     time.Time
	SourceType     SourceType
	SourceID       string
	IsRead         bool
	IsBookmarked   bool
	IsDownloaded   bool
}

// Page represents a manga page
type Page struct {
	Index     int
	URL       string
	ImageData []byte // For local files or cached data
	ImageType string // MIME type (e.g., "image/jpeg", "image/png")
}

// Source is the interface that all manga sources must implement
type Source interface {
	// GetType returns the type of this source
	GetType() SourceType

	// GetID returns a unique identifier for this source instance
	GetID() string

	// GetName returns a human-readable name for this source
	GetName() string

	// ListManga lists all available manga from this source
	ListManga() ([]*Manga, error)

	// GetManga retrieves detailed metadata for a specific manga
	GetManga(mangaID string) (*Manga, error)

	// ListChapters lists all chapters for a specific manga
	ListChapters(mangaID string) ([]*Chapter, error)

	// GetChapter retrieves detailed metadata for a specific chapter
	GetChapter(chapterID string) (*Chapter, error)

	// GetPage retrieves a specific page from a chapter
	GetPage(chapter *Chapter, pageIndex int) (*Page, error)

	// GetAllPages retrieves all pages from a chapter
	GetAllPages(chapter *Chapter) ([]*Page, error)

	// Search searches for manga by query (optional, may return nil for unsupported sources)
	Search(query string) ([]*Manga, error)

	// IsAvailable checks if the source is currently accessible
	IsAvailable() bool
}

// SourceManager manages multiple manga sources
type SourceManager struct {
	sources []Source
}

// NewSourceManager creates a new source manager
func NewSourceManager() *SourceManager {
	return &SourceManager{
		sources: make([]Source, 0),
	}
}

// AddSource adds a source to the manager
func (sm *SourceManager) AddSource(source Source) {
	sm.sources = append(sm.sources, source)
}

// GetSources returns all registered sources
func (sm *SourceManager) GetSources() []Source {
	return sm.sources
}

// GetSource retrieves a source by ID
func (sm *SourceManager) GetSource(id string) Source {
	for _, source := range sm.sources {
		if source.GetID() == id {
			return source
		}
	}
	return nil
}

// GetSourcesByType retrieves all sources of a specific type
func (sm *SourceManager) GetSourcesByType(sourceType SourceType) []Source {
	var result []Source
	for _, source := range sm.sources {
		if source.GetType() == sourceType {
			result = append(result, source)
		}
	}
	return result
}

// ListAllManga lists manga from all available sources
func (sm *SourceManager) ListAllManga() ([]*Manga, error) {
	var allManga []*Manga
	var lastErr error
	errorCount := 0

	for _, source := range sm.sources {
		if !source.IsAvailable() {
			continue
		}
		manga, err := source.ListManga()
		if err != nil {
			// Track the error
			lastErr = err
			errorCount++
			// Continue with other sources
			continue
		}
		allManga = append(allManga, manga...)
	}

	// If we have no manga and encountered errors, return the last error
	if len(allManga) == 0 && lastErr != nil {
		return nil, lastErr
	}

	return allManga, nil
}

// SearchAllSources searches for manga across all sources
func (sm *SourceManager) SearchAllSources(query string) ([]*Manga, error) {
	var results []*Manga
	for _, source := range sm.sources {
		if !source.IsAvailable() {
			continue
		}
		manga, err := source.Search(query)
		if err != nil {
			// Log error but continue with other sources
			continue
		}
		results = append(results, manga...)
	}
	return results, nil
}

// GetAllPages retrieves all pages for a chapter from its source
func (sm *SourceManager) GetAllPages(chapter *Chapter) ([]*Page, error) {
	// Find the source that matches this chapter's source ID
	for _, source := range sm.sources {
		if source.GetID() != chapter.SourceID {
			continue
		}
		if !source.IsAvailable() {
			continue
		}
		return source.GetAllPages(chapter)
	}
	return nil, fmt.Errorf("source not found for chapter %s (source ID: %s)", chapter.ID, chapter.SourceID)
}

// GetPage retrieves a specific page from a chapter
func (sm *SourceManager) GetPage(chapter *Chapter, pageIndex int) ([]byte, error) {
	// Find the source that matches this chapter's source ID
	for _, source := range sm.sources {
		if source.GetID() != chapter.SourceID {
			continue
		}
		if !source.IsAvailable() {
			continue
		}
		page, err := source.GetPage(chapter, pageIndex)
		if err == nil && page != nil {
			// If ImageData is already loaded, return it
			if len(page.ImageData) > 0 {
				return page.ImageData, nil
			}
			// Otherwise, would need to fetch from URL
			// For now, return empty if only URL is available
			return page.ImageData, nil
		}
	}
	return nil, fmt.Errorf("source not found for chapter %s (source ID: %s)", chapter.ID, chapter.SourceID)
}
