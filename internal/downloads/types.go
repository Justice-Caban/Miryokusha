package downloads

import (
	"sync"
	"time"

	"github.com/Justice-Caban/Miryokusha/internal/source"
)

// DownloadStatus represents the current state of a download
type DownloadStatus string

const (
	StatusQueued      DownloadStatus = "queued"
	StatusDownloading DownloadStatus = "downloading"
	StatusCompleted   DownloadStatus = "completed"
	StatusFailed      DownloadStatus = "failed"
	StatusPaused      DownloadStatus = "paused"
)

// DownloadItem represents a single chapter download
type DownloadItem struct {
	ID          string // Unique ID for this download
	MangaID     string
	MangaTitle  string
	ChapterID   string
	ChapterName string
	SourceType  source.SourceType

	Status       DownloadStatus
	Priority     int // Lower number = higher priority
	CurrentPage  int
	TotalPages   int
	Error        error
	StartedAt    time.Time
	CompletedAt  time.Time

	// Progress tracking
	BytesDownloaded int64
	TotalBytes      int64

	// Retry tracking
	RetryCount int // Current number of retry attempts

	// Cancel function
	Cancel func()
}

// Progress returns the download progress as a percentage (0-100)
func (di *DownloadItem) Progress() float64 {
	if di.TotalPages == 0 {
		return 0
	}
	return (float64(di.CurrentPage) / float64(di.TotalPages)) * 100
}

// IsActive returns true if the download is currently active
func (di *DownloadItem) IsActive() bool {
	return di.Status == StatusDownloading
}

// IsComplete returns true if the download has finished successfully
func (di *DownloadItem) IsComplete() bool {
	return di.Status == StatusCompleted
}

// IsFailed returns true if the download has failed
func (di *DownloadItem) IsFailed() bool {
	return di.Status == StatusFailed
}

// DownloadConfig holds configuration for the download manager
type DownloadConfig struct {
	DownloadPath       string // Where to save downloads
	MaxConcurrent      int    // Max simultaneous downloads
	RetryAttempts      int    // Number of retry attempts for failed downloads
	RetryDelay         time.Duration
	PageTimeout        time.Duration // Timeout for downloading a single page
	AutoDeleteRead     bool          // Auto-delete chapters after reading
	OnlyOnWiFi         bool          // Only download on WiFi (future mobile)
}

// DefaultDownloadConfig returns default download configuration
func DefaultDownloadConfig() *DownloadConfig {
	return &DownloadConfig{
		DownloadPath:    "~/.local/share/miryokusha/downloads",
		MaxConcurrent:   3,
		RetryAttempts:   3,
		RetryDelay:      time.Second * 5,
		PageTimeout:     time.Second * 30,
		AutoDeleteRead:  false,
		OnlyOnWiFi:      false,
	}
}

// DownloadStats holds statistics about downloads
type DownloadStats struct {
	mu sync.RWMutex

	TotalDownloads    int
	CompletedDownloads int
	FailedDownloads   int
	TotalBytesDownloaded int64
	ActiveDownloads   int
}

// Update updates the stats (thread-safe)
func (ds *DownloadStats) Update(fn func()) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	fn()
}

// GetStats returns a copy of the current stats (thread-safe)
func (ds *DownloadStats) GetStats() DownloadStats {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return DownloadStats{
		TotalDownloads:       ds.TotalDownloads,
		CompletedDownloads:   ds.CompletedDownloads,
		FailedDownloads:      ds.FailedDownloads,
		TotalBytesDownloaded: ds.TotalBytesDownloaded,
		ActiveDownloads:      ds.ActiveDownloads,
	}
}
