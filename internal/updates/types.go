package updates

import (
	"time"

	"github.com/Justice-Caban/Miryokusha/internal/source"
)

// UpdateStatus represents the current state of an update task
type UpdateStatus string

const (
	StatusPending    UpdateStatus = "pending"
	StatusChecking   UpdateStatus = "checking"
	StatusCompleted  UpdateStatus = "completed"
	StatusFailed     UpdateStatus = "failed"
)

// UpdateTask represents a manga update check task
type UpdateTask struct {
	MangaID       string
	MangaTitle    string
	SourceType    source.SourceType
	Status        UpdateStatus
	Error         error

	// Update results
	OldChapterCount int
	NewChapterCount int
	NewChapters     []*source.Chapter

	// Timing
	StartedAt   time.Time
	CompletedAt time.Time
}

// HasNewChapters returns true if new chapters were found
func (ut *UpdateTask) HasNewChapters() bool {
	return ut.NewChapterCount > ut.OldChapterCount
}

// GetNewChapters returns the count of new chapters found
func (ut *UpdateTask) GetNewChapters() int {
	if ut.NewChapterCount > ut.OldChapterCount {
		return ut.NewChapterCount - ut.OldChapterCount
	}
	return 0
}

// UpdateSummary summarizes an update session
type UpdateSummary struct {
	StartedAt   time.Time
	CompletedAt time.Time
	TotalManga  int
	UpdatedManga int // Manga with new chapters
	FailedManga int
	NewChapters int
	Tasks       []*UpdateTask
}

// Duration returns how long the update took
func (us *UpdateSummary) Duration() time.Duration {
	if us.CompletedAt.IsZero() {
		return time.Since(us.StartedAt)
	}
	return us.CompletedAt.Sub(us.StartedAt)
}

// UpdateConfig holds configuration for library updates
type UpdateConfig struct {
	// Update interval (0 = manual only)
	Interval time.Duration

	// Filters
	UpdateOnlyStarted   bool // Only update manga that have been read
	UpdateOnlyCompleted bool // Only update completed manga

	// Concurrency
	MaxConcurrent int // Max concurrent update checks

	// Smart update (skip manga unlikely to have updates)
	SmartUpdate        bool
	SmartUpdateMinDays int // Min days since last update to check again

	// Notifications
	NotifyNewChapters bool
	NotifyFailures    bool
}

// DefaultUpdateConfig returns default update configuration
func DefaultUpdateConfig() *UpdateConfig {
	return &UpdateConfig{
		Interval:           12 * time.Hour, // Check every 12 hours
		UpdateOnlyStarted:  true,
		UpdateOnlyCompleted: false,
		MaxConcurrent:      5,
		SmartUpdate:        false,
		SmartUpdateMinDays: 1,
		NotifyNewChapters:  true,
		NotifyFailures:     false,
	}
}

// Notification represents an update notification
type Notification struct {
	ID        string
	Type      NotificationType
	Title     string
	Message   string
	CreatedAt time.Time
	Read      bool

	// Related data
	MangaID   string
	ChapterID string
}

// NotificationType represents the type of notification
type NotificationType string

const (
	NotifyNewChapter    NotificationType = "new_chapter"
	NotifyUpdateComplete NotificationType = "update_complete"
	NotifyUpdateFailed   NotificationType = "update_failed"
)
