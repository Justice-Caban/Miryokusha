package source

import (
	"fmt"
)

// SuwayomiSource represents a Suwayomi server source
type SuwayomiSource struct {
	id      string
	name    string
	baseURL string
	// TODO: Add HTTP client and authentication when implementing API client
}

// NewSuwayomiSource creates a new Suwayomi source
func NewSuwayomiSource(id, name, baseURL string) *SuwayomiSource {
	return &SuwayomiSource{
		id:      id,
		name:    name,
		baseURL: baseURL,
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
	// TODO: Implement API call to /api/v1/manga
	return nil, fmt.Errorf("not yet implemented")
}

// GetManga retrieves manga details from the Suwayomi server
func (s *SuwayomiSource) GetManga(mangaID string) (*Manga, error) {
	// TODO: Implement API call to /api/v1/manga/{id}
	return nil, fmt.Errorf("not yet implemented")
}

// ListChapters lists all chapters for a manga
func (s *SuwayomiSource) ListChapters(mangaID string) ([]*Chapter, error) {
	// TODO: Implement API call to /api/v1/manga/{id}/chapters
	return nil, fmt.Errorf("not yet implemented")
}

// GetChapter retrieves chapter details
func (s *SuwayomiSource) GetChapter(chapterID string) (*Chapter, error) {
	// TODO: Implement API call to /api/v1/chapter/{id}
	return nil, fmt.Errorf("not yet implemented")
}

// GetPage retrieves a specific page from a chapter
func (s *SuwayomiSource) GetPage(chapterID string, pageIndex int) (*Page, error) {
	// TODO: Implement API call to /api/v1/chapter/{id}/page/{page}
	return nil, fmt.Errorf("not yet implemented")
}

// GetAllPages retrieves all pages from a chapter
func (s *SuwayomiSource) GetAllPages(chapterID string) ([]*Page, error) {
	// TODO: Implement by getting chapter details and fetching all pages
	return nil, fmt.Errorf("not yet implemented")
}

// Search searches for manga on the Suwayomi server
func (s *SuwayomiSource) Search(query string) ([]*Manga, error) {
	// TODO: Implement API search endpoint
	return nil, fmt.Errorf("not yet implemented")
}

// IsAvailable checks if the Suwayomi server is accessible
func (s *SuwayomiSource) IsAvailable() bool {
	// TODO: Implement health check to server
	return false
}
