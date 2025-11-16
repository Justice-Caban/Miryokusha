package downloads

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Justice-Caban/Miryokusha/internal/source"
)

// Manager handles download queue and execution
type Manager struct {
	mu sync.RWMutex

	config        *DownloadConfig
	sourceManager *source.SourceManager

	queue        []*DownloadItem
	active       map[string]*DownloadItem
	completed    []*DownloadItem
	stats        *DownloadStats

	running      bool
	ctx          context.Context
	cancel       context.CancelFunc

	// Callbacks
	onProgress   func(*DownloadItem)
	onComplete   func(*DownloadItem)
	onError      func(*DownloadItem, error)
}

// NewManager creates a new download manager
func NewManager(config *DownloadConfig, sm *source.SourceManager) *Manager {
	if config == nil {
		config = DefaultDownloadConfig()
	}

	return &Manager{
		config:        config,
		sourceManager: sm,
		queue:         make([]*DownloadItem, 0),
		active:        make(map[string]*DownloadItem),
		completed:     make([]*DownloadItem, 0),
		stats:         &DownloadStats{},
	}
}

// Add adds a chapter to the download queue
func (m *Manager) Add(manga *source.Manga, chapter *source.Chapter, priority int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already in queue or downloading
	for _, item := range m.queue {
		if item.ChapterID == chapter.ID {
			return fmt.Errorf("chapter already in queue")
		}
	}

	if _, exists := m.active[chapter.ID]; exists {
		return fmt.Errorf("chapter is currently downloading")
	}

	// Create download item
	item := &DownloadItem{
		ID:          fmt.Sprintf("%s-%s-%d", manga.ID, chapter.ID, time.Now().Unix()),
		MangaID:     manga.ID,
		MangaTitle:  manga.Title,
		ChapterID:   chapter.ID,
		ChapterName: chapter.Title,
		SourceType:  manga.SourceType,
		Status:      StatusQueued,
		Priority:    priority,
	}

	m.queue = append(m.queue, item)
	m.sortQueue()

	m.stats.Update(func() {
		m.stats.TotalDownloads++
	})

	// Start processing if running
	if m.running {
		go m.processQueue()
	}

	return nil
}

// Remove removes a download from the queue
func (m *Manager) Remove(chapterID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if in queue
	for i, item := range m.queue {
		if item.ChapterID == chapterID {
			m.queue = append(m.queue[:i], m.queue[i+1:]...)
			return nil
		}
	}

	// Check if active
	if item, exists := m.active[chapterID]; exists {
		if item.Cancel != nil {
			item.Cancel()
		}
		delete(m.active, chapterID)
		return nil
	}

	return fmt.Errorf("chapter not found in queue")
}

// Start starts the download manager
func (m *Manager) Start() {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}

	m.running = true
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.mu.Unlock()

	go m.processQueue()
}

// Stop stops the download manager
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	m.running = false
	if m.cancel != nil {
		m.cancel()
	}

	// Cancel all active downloads
	for _, item := range m.active {
		if item.Cancel != nil {
			item.Cancel()
		}
	}
	m.active = make(map[string]*DownloadItem)
}

// Pause pauses all downloads
func (m *Manager) Pause() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Move active downloads back to queue
	for _, item := range m.active {
		if item.Cancel != nil {
			item.Cancel()
		}
		item.Status = StatusPaused
		m.queue = append(m.queue, item)
	}
	m.active = make(map[string]*DownloadItem)
	m.sortQueue()
}

// Resume resumes paused downloads
func (m *Manager) Resume() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, item := range m.queue {
		if item.Status == StatusPaused {
			item.Status = StatusQueued
		}
	}

	if m.running {
		go m.processQueue()
	}
}

// ClearCompleted removes all completed downloads from the list
func (m *Manager) ClearCompleted() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.completed = make([]*DownloadItem, 0)
}

// GetQueue returns a copy of the current queue
func (m *Manager) GetQueue() []*DownloadItem {
	m.mu.RLock()
	defer m.mu.RUnlock()

	queue := make([]*DownloadItem, len(m.queue))
	copy(queue, m.queue)
	return queue
}

// GetActive returns a copy of active downloads
func (m *Manager) GetActive() []*DownloadItem {
	m.mu.RLock()
	defer m.mu.RUnlock()

	active := make([]*DownloadItem, 0, len(m.active))
	for _, item := range m.active {
		active = append(active, item)
	}
	return active
}

// GetCompleted returns a copy of completed downloads
func (m *Manager) GetCompleted() []*DownloadItem {
	m.mu.RLock()
	defer m.mu.RUnlock()

	completed := make([]*DownloadItem, len(m.completed))
	copy(completed, m.completed)
	return completed
}

// GetStats returns current download statistics
func (m *Manager) GetStats() DownloadStats {
	return m.stats.GetStats()
}

// SetCallbacks sets callback functions for download events
func (m *Manager) SetCallbacks(onProgress, onComplete func(*DownloadItem), onError func(*DownloadItem, error)) {
	m.onProgress = onProgress
	m.onComplete = onComplete
	m.onError = onError
}

// processQueue processes the download queue
func (m *Manager) processQueue() {
	for {
		m.mu.Lock()
		if !m.running {
			m.mu.Unlock()
			return
		}

		// Check if we can start more downloads
		if len(m.active) >= m.config.MaxConcurrent {
			m.mu.Unlock()
			time.Sleep(time.Second)
			continue
		}

		// Get next item from queue
		var nextItem *DownloadItem
		for i, item := range m.queue {
			if item.Status == StatusQueued {
				nextItem = item
				m.queue = append(m.queue[:i], m.queue[i+1:]...)
				break
			}
		}

		if nextItem == nil {
			m.mu.Unlock()
			time.Sleep(time.Second)
			continue
		}

		// Start download
		m.active[nextItem.ChapterID] = nextItem
		m.stats.Update(func() {
			m.stats.ActiveDownloads++
		})
		m.mu.Unlock()

		go m.downloadChapter(nextItem)
	}
}

// downloadChapter downloads a single chapter
func (m *Manager) downloadChapter(item *DownloadItem) {
	defer func() {
		m.mu.Lock()
		delete(m.active, item.ChapterID)
		m.stats.Update(func() {
			m.stats.ActiveDownloads--
		})
		m.mu.Unlock()
	}()

	// Create context with cancel
	ctx, cancel := context.WithCancel(m.ctx)
	item.Cancel = cancel
	defer cancel()

	item.Status = StatusDownloading
	item.StartedAt = time.Now()

	// Get all pages for the chapter
	pages, err := m.sourceManager.GetAllPages(item.ChapterID)
	if err != nil {
		m.handleError(item, err)
		return
	}

	item.TotalPages = len(pages)

	// Create chapter directory
	chapterDir := filepath.Join(
		m.config.DownloadPath,
		sanitizeFilename(item.MangaTitle),
		sanitizeFilename(item.ChapterName),
	)

	if err := os.MkdirAll(chapterDir, 0755); err != nil {
		m.handleError(item, err)
		return
	}

	// Download each page
	for i, page := range pages {
		select {
		case <-ctx.Done():
			// Download cancelled
			item.Status = StatusFailed
			item.Error = fmt.Errorf("download cancelled")
			return
		default:
		}

		// Download page with timeout
		pageCtx, pageCancel := context.WithTimeout(ctx, m.config.PageTimeout)
		err := m.downloadPage(pageCtx, page, chapterDir, i, item.ChapterID)
		pageCancel()

		if err != nil {
			m.handleError(item, err)
			return
		}

		item.CurrentPage = i + 1

		// Call progress callback
		if m.onProgress != nil {
			m.onProgress(item)
		}
	}

	// Mark as completed
	item.Status = StatusCompleted
	item.CompletedAt = time.Now()

	m.mu.Lock()
	m.completed = append(m.completed, item)
	m.mu.Unlock()

	m.stats.Update(func() {
		m.stats.CompletedDownloads++
	})

	if m.onComplete != nil {
		m.onComplete(item)
	}
}

// downloadPage downloads a single page
func (m *Manager) downloadPage(ctx context.Context, page *source.Page, chapterDir string, index int, chapterID string) error {
	// Get page data - try using the ImageData if it's already loaded
	var data []byte
	if len(page.ImageData) > 0 {
		data = page.ImageData
	} else {
		// Otherwise fetch from source
		var err error
		data, err = m.sourceManager.GetPage(chapterID, index)
		if err != nil {
			return err
		}
	}

	// Determine file extension from URL or default to .jpg
	ext := ".jpg"
	if len(page.URL) > 0 {
		ext = filepath.Ext(page.URL)
		if ext == "" {
			ext = ".jpg"
		}
	}

	// Save to file
	filename := fmt.Sprintf("%04d%s", index+1, ext)
	filePath := filepath.Join(chapterDir, filename)

	return os.WriteFile(filePath, data, 0644)
}

// handleError handles download errors with retry logic
func (m *Manager) handleError(item *DownloadItem, err error) {
	item.Error = err
	item.Status = StatusFailed

	m.stats.Update(func() {
		m.stats.FailedDownloads++
	})

	if m.onError != nil {
		m.onError(item, err)
	}

	// TODO: Implement retry logic
}

// sortQueue sorts the queue by priority (lower number = higher priority)
func (m *Manager) sortQueue() {
	sort.Slice(m.queue, func(i, j int) bool {
		return m.queue[i].Priority < m.queue[j].Priority
	})
}

// sanitizeFilename removes invalid characters from filenames
func sanitizeFilename(name string) string {
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := name
	for _, invalidChar := range invalid {
		// Replace invalid characters with underscores
		result = strings.ReplaceAll(result, invalidChar, "_")
	}
	return result
}
