package source

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nwaples/rardecode/v2"
)

// LocalSource represents a local file source for manga
type LocalSource struct {
	id        string
	name      string
	baseDir   string
	scanDirs  []string
	manga     map[string]*Manga
	chapters  map[string][]*Chapter
}

// NewLocalSource creates a new local file source
func NewLocalSource(id, name, baseDir string) *LocalSource {
	return &LocalSource{
		id:       id,
		name:     name,
		baseDir:  baseDir,
		scanDirs: make([]string, 0),
		manga:    make(map[string]*Manga),
		chapters: make(map[string][]*Chapter),
	}
}

// AddScanDirectory adds a directory to scan for manga files
func (ls *LocalSource) AddScanDirectory(dir string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if directory exists
	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", absDir)
	}

	ls.scanDirs = append(ls.scanDirs, absDir)
	return nil
}

// AddFile adds a specific manga file (CBZ, CBR, PDF)
func (ls *LocalSource) AddFile(filePath string) error {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", absPath)
	}

	// Parse the file and add to manga/chapters
	return ls.parseFile(absPath)
}

// Scan scans all configured directories for manga files
func (ls *LocalSource) Scan() error {
	for _, dir := range ls.scanDirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			// Check if it's a supported file type
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".cbz" || ext == ".cbr" || ext == ".pdf" {
				// Parse and add the file
				if err := ls.parseFile(path); err != nil {
					// Log error but continue scanning
					fmt.Fprintf(os.Stderr, "Failed to parse %s: %v\n", path, err)
				}
			}

			return nil
		})

		if err != nil {
			return fmt.Errorf("failed to scan directory %s: %w", dir, err)
		}
	}

	return nil
}

// parseFile parses a manga file and extracts metadata
func (ls *LocalSource) parseFile(filePath string) error {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".cbz":
		return ls.parseCBZ(filePath)
	case ".cbr":
		return ls.parseCBR(filePath)
	case ".pdf":
		return ls.parsePDF(filePath)
	default:
		return fmt.Errorf("unsupported file type: %s", ext)
	}
}

// parseCBZ parses a CBZ (Comic Book ZIP) file
func (ls *LocalSource) parseCBZ(filePath string) error {
	// Extract metadata from filename
	baseName := filepath.Base(filePath)
	title := strings.TrimSuffix(baseName, filepath.Ext(baseName))

	// Create manga ID from file path
	mangaID := fmt.Sprintf("local-%s", sanitizeID(filePath))

	// Check if manga already exists
	manga, exists := ls.manga[mangaID]
	if !exists {
		manga = &Manga{
			ID:         mangaID,
			Title:      title,
			SourceType: SourceTypeLocal,
			SourceID:   ls.id,
		}
		ls.manga[mangaID] = manga
	}

	// Open ZIP file to count pages
	zipReader, err := zip.OpenReader(filePath)
	if err != nil {
		return fmt.Errorf("failed to open CBZ: %w", err)
	}
	defer zipReader.Close()

	// Count image files
	pageCount := 0
	for _, file := range zipReader.File {
		if isImageFile(file.Name) {
			pageCount++
		}
	}

	// Create chapter
	chapterID := fmt.Sprintf("local-chapter-%s", sanitizeID(filePath))
	chapter := &Chapter{
		ID:            chapterID,
		MangaID:       mangaID,
		Title:         title,
		ChapterNumber: 1.0, // Single file = single chapter
		PageCount:     pageCount,
		SourceType:    SourceTypeLocal,
		SourceID:      ls.id,
	}

	ls.chapters[mangaID] = []*Chapter{chapter}

	return nil
}

// parseCBR parses a CBR (Comic Book RAR) file
func (ls *LocalSource) parseCBR(filePath string) error {
	// Extract metadata from filename
	baseName := filepath.Base(filePath)
	title := strings.TrimSuffix(baseName, filepath.Ext(baseName))

	// Create manga ID from file path
	mangaID := fmt.Sprintf("local-%s", sanitizeID(filePath))

	// Check if manga already exists
	manga, exists := ls.manga[mangaID]
	if !exists {
		manga = &Manga{
			ID:         mangaID,
			Title:      title,
			SourceType: SourceTypeLocal,
			SourceID:   ls.id,
		}
		ls.manga[mangaID] = manga
	}

	// Open RAR file to count pages
	rarFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open CBR: %w", err)
	}
	defer rarFile.Close()

	rarReader, err := rardecode.NewReader(rarFile)
	if err != nil {
		return fmt.Errorf("failed to create RAR reader: %w", err)
	}

	// Count image files
	pageCount := 0
	for {
		header, err := rarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read RAR entry: %w", err)
		}

		if !header.IsDir && isImageFile(header.Name) {
			pageCount++
		}
	}

	// Create chapter
	chapterID := fmt.Sprintf("local-chapter-%s", sanitizeID(filePath))
	chapter := &Chapter{
		ID:            chapterID,
		MangaID:       mangaID,
		Title:         title,
		ChapterNumber: 1.0, // Single file = single chapter
		PageCount:     pageCount,
		SourceType:    SourceTypeLocal,
		SourceID:      ls.id,
	}

	ls.chapters[mangaID] = []*Chapter{chapter}

	return nil
}

// parsePDF parses a PDF file
func (ls *LocalSource) parsePDF(filePath string) error {
	// Extract metadata from filename
	baseName := filepath.Base(filePath)
	title := strings.TrimSuffix(baseName, filepath.Ext(baseName))

	// Create manga ID from file path
	mangaID := fmt.Sprintf("local-%s", sanitizeID(filePath))

	// Check if manga already exists
	manga, exists := ls.manga[mangaID]
	if !exists {
		manga = &Manga{
			ID:          mangaID,
			Title:       title,
			SourceType:  SourceTypeLocal,
			SourceID:    ls.id,
			Description: "PDF manga file",
		}
		ls.manga[mangaID] = manga
	}

	// Open PDF file to count pages
	pdfFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open PDF: %w", err)
	}
	defer pdfFile.Close()

	// Get PDF page count by reading the file
	// This is a simplified version - a full implementation would use a PDF library
	// For now, estimate 20 pages as placeholder
	pageCount := 20 // Placeholder

	// Try to read PDF page count from file structure
	// PDF files have a /Count entry in the page tree
	// This is a basic heuristic and won't work for all PDFs
	stat, err := pdfFile.Stat()
	if err == nil {
		// Estimate ~50KB per page as a rough heuristic
		estimatedPages := int(stat.Size() / 50000)
		if estimatedPages > 0 {
			pageCount = estimatedPages
		}
	}

	// Create chapter
	chapterID := fmt.Sprintf("local-chapter-%s", sanitizeID(filePath))
	chapter := &Chapter{
		ID:             chapterID,
		MangaID:        mangaID,
		Title:          title,
		ChapterNumber:  1.0, // Single file = single chapter
		PageCount:      pageCount,
		SourceType:     SourceTypeLocal,
		SourceID:       ls.id,
		ScanlatorGroup: "PDF",
	}

	ls.chapters[mangaID] = []*Chapter{chapter}

	return nil
}

// GetType returns the source type
func (ls *LocalSource) GetType() SourceType {
	return SourceTypeLocal
}

// GetID returns the source ID
func (ls *LocalSource) GetID() string {
	return ls.id
}

// GetName returns the source name
func (ls *LocalSource) GetName() string {
	return ls.name
}

// ListManga lists all manga from local files
func (ls *LocalSource) ListManga() ([]*Manga, error) {
	result := make([]*Manga, 0, len(ls.manga))
	for _, manga := range ls.manga {
		result = append(result, manga)
	}

	// Sort by title
	sort.Slice(result, func(i, j int) bool {
		return result[i].Title < result[j].Title
	})

	return result, nil
}

// GetManga retrieves manga details
func (ls *LocalSource) GetManga(mangaID string) (*Manga, error) {
	manga, exists := ls.manga[mangaID]
	if !exists {
		return nil, fmt.Errorf("manga not found: %s", mangaID)
	}
	return manga, nil
}

// ListChapters lists all chapters for a manga
func (ls *LocalSource) ListChapters(mangaID string) ([]*Chapter, error) {
	chapters, exists := ls.chapters[mangaID]
	if !exists {
		return []*Chapter{}, nil
	}
	return chapters, nil
}

// GetChapter retrieves chapter details
func (ls *LocalSource) GetChapter(chapterID string) (*Chapter, error) {
	// Search through all chapters
	for _, chapterList := range ls.chapters {
		for _, chapter := range chapterList {
			if chapter.ID == chapterID {
				return chapter, nil
			}
		}
	}
	return nil, fmt.Errorf("chapter not found: %s", chapterID)
}

// GetPage retrieves a specific page from a chapter
func (ls *LocalSource) GetPage(chapterID string, pageIndex int) (*Page, error) {
	// Extract page from the source file
	// This is a simplified version - full implementation would cache pages
	pages, err := ls.GetAllPages(chapterID)
	if err != nil {
		return nil, err
	}

	if pageIndex < 0 || pageIndex >= len(pages) {
		return nil, fmt.Errorf("page index out of range: %d", pageIndex)
	}

	return pages[pageIndex], nil
}

// GetAllPages retrieves all pages from a chapter
func (ls *LocalSource) GetAllPages(chapterID string) ([]*Page, error) {
	chapter, err := ls.GetChapter(chapterID)
	if err != nil {
		return nil, err
	}

	// For now, return placeholder - full implementation would extract images
	// This would need the original file path stored in the chapter
	pages := make([]*Page, chapter.PageCount)
	for i := 0; i < chapter.PageCount; i++ {
		pages[i] = &Page{
			Index: i,
			URL:   "", // Local files don't have URLs
		}
	}

	return pages, nil
}

// Search searches for manga by title
func (ls *LocalSource) Search(query string) ([]*Manga, error) {
	query = strings.ToLower(query)
	var results []*Manga

	for _, manga := range ls.manga {
		if strings.Contains(strings.ToLower(manga.Title), query) {
			results = append(results, manga)
		}
	}

	return results, nil
}

// IsAvailable checks if the local source is accessible
func (ls *LocalSource) IsAvailable() bool {
	// Local source is always available if directories are accessible
	for _, dir := range ls.scanDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// Helper functions

func sanitizeID(s string) string {
	// Replace special characters with underscores
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, " ", "_")
	return s
}

func isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp"
}
