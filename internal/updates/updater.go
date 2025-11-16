package updates

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Justice-Caban/Miryokusha/internal/source"
	"github.com/Justice-Caban/Miryokusha/internal/storage"
)

// Updater manages library updates
type Updater struct {
	mu sync.RWMutex

	config        *UpdateConfig
	sourceManager *source.SourceManager
	storage       *storage.Storage

	// Update state
	running         bool
	currentSummary  *UpdateSummary
	updateHistory   []*UpdateSummary
	notifications   []*Notification

	// Scheduling
	ticker *time.Ticker
	ctx    context.Context
	cancel context.CancelFunc

	// Callbacks
	onProgress     func(*UpdateTask)
	onComplete     func(*UpdateSummary)
	onNotification func(*Notification)
}

// NewUpdater creates a new library updater
func NewUpdater(config *UpdateConfig, sm *source.SourceManager, st *storage.Storage) *Updater {
	if config == nil {
		config = DefaultUpdateConfig()
	}

	return &Updater{
		config:         config,
		sourceManager:  sm,
		storage:        st,
		updateHistory:  make([]*UpdateSummary, 0),
		notifications:  make([]*Notification, 0),
	}
}

// Start starts the automatic update scheduler
func (u *Updater) Start() {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.running || u.config.Interval == 0 {
		return
	}

	u.running = true
	u.ctx, u.cancel = context.WithCancel(context.Background())
	u.ticker = time.NewTicker(u.config.Interval)

	go u.scheduleUpdates()
}

// Stop stops the automatic update scheduler
func (u *Updater) Stop() {
	u.mu.Lock()
	defer u.mu.Unlock()

	if !u.running {
		return
	}

	u.running = false
	if u.cancel != nil {
		u.cancel()
	}
	if u.ticker != nil {
		u.ticker.Stop()
	}
}

// UpdateLibrary performs a full library update
func (u *Updater) UpdateLibrary() (*UpdateSummary, error) {
	// Get all manga from sources
	allManga, err := u.sourceManager.ListAllManga()
	if err != nil {
		return nil, err
	}

	// Filter manga based on config
	manga := u.filterManga(allManga)

	return u.updateMangaList(manga)
}

// UpdateManga updates a specific manga
func (u *Updater) UpdateManga(mangaID string) (*UpdateTask, error) {
	// Find manga in sources
	var manga *source.Manga
	for _, src := range u.sourceManager.GetSources() {
		m, err := src.GetManga(mangaID)
		if err == nil && m != nil {
			manga = m
			break
		}
	}

	if manga == nil {
		return nil, fmt.Errorf("manga not found: %s", mangaID)
	}

	return u.updateSingleManga(manga)
}

// GetCurrentSummary returns the current update summary
func (u *Updater) GetCurrentSummary() *UpdateSummary {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.currentSummary
}

// GetUpdateHistory returns the update history
func (u *Updater) GetUpdateHistory() []*UpdateSummary {
	u.mu.RLock()
	defer u.mu.RUnlock()

	history := make([]*UpdateSummary, len(u.updateHistory))
	copy(history, u.updateHistory)
	return history
}

// GetNotifications returns all notifications
func (u *Updater) GetNotifications() []*Notification {
	u.mu.RLock()
	defer u.mu.RUnlock()

	notifications := make([]*Notification, len(u.notifications))
	copy(notifications, u.notifications)
	return notifications
}

// ClearNotifications clears all notifications
func (u *Updater) ClearNotifications() {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.notifications = make([]*Notification, 0)
}

// MarkNotificationRead marks a notification as read
func (u *Updater) MarkNotificationRead(id string) {
	u.mu.Lock()
	defer u.mu.Unlock()

	for _, notif := range u.notifications {
		if notif.ID == id {
			notif.Read = true
			break
		}
	}
}

// SetCallbacks sets callback functions for update events
func (u *Updater) SetCallbacks(onProgress func(*UpdateTask), onComplete func(*UpdateSummary), onNotification func(*Notification)) {
	u.onProgress = onProgress
	u.onComplete = onComplete
	u.onNotification = onNotification
}

// scheduleUpdates runs the automatic update scheduler
func (u *Updater) scheduleUpdates() {
	for {
		select {
		case <-u.ctx.Done():
			return
		case <-u.ticker.C:
			// Perform library update
			summary, err := u.UpdateLibrary()
			if err != nil {
				// Log error but continue
				continue
			}

			// Notify if configured
			if u.config.NotifyNewChapters && summary.NewChapters > 0 {
				u.addNotification(&Notification{
					ID:        fmt.Sprintf("update-%d", time.Now().Unix()),
					Type:      NotifyUpdateComplete,
					Title:     "Library Updated",
					Message:   fmt.Sprintf("Found %d new chapters across %d manga", summary.NewChapters, summary.UpdatedManga),
					CreatedAt: time.Now(),
				})
			}
		}
	}
}

// updateMangaList updates a list of manga
func (u *Updater) updateMangaList(mangaList []*source.Manga) (*UpdateSummary, error) {
	summary := &UpdateSummary{
		StartedAt:  time.Now(),
		TotalManga: len(mangaList),
		Tasks:      make([]*UpdateTask, 0),
	}

	u.mu.Lock()
	u.currentSummary = summary
	u.mu.Unlock()

	// Use semaphore to limit concurrency
	sem := make(chan struct{}, u.config.MaxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, manga := range mangaList {
		wg.Add(1)
		go func(m *source.Manga) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Update single manga
			task, err := u.updateSingleManga(m)

			mu.Lock()
			if err != nil {
				summary.FailedManga++
				if task != nil {
					task.Status = StatusFailed
					task.Error = err
				}
			} else if task.HasNewChapters() {
				summary.UpdatedManga++
				summary.NewChapters += task.GetNewChapters()

				// Send notification for new chapters
				if u.config.NotifyNewChapters {
					u.addNotification(&Notification{
						ID:        fmt.Sprintf("manga-%s-%d", m.ID, time.Now().Unix()),
						Type:      NotifyNewChapter,
						Title:     m.Title,
						Message:   fmt.Sprintf("%d new chapter(s) available", task.GetNewChapters()),
						CreatedAt: time.Now(),
						MangaID:   m.ID,
					})
				}
			}

			if task != nil {
				summary.Tasks = append(summary.Tasks, task)
			}
			mu.Unlock()

			// Call progress callback
			if u.onProgress != nil && task != nil {
				u.onProgress(task)
			}
		}(manga)
	}

	wg.Wait()

	summary.CompletedAt = time.Now()

	// Add to history
	u.mu.Lock()
	u.updateHistory = append(u.updateHistory, summary)
	// Keep only last 50 updates
	if len(u.updateHistory) > 50 {
		u.updateHistory = u.updateHistory[len(u.updateHistory)-50:]
	}
	u.mu.Unlock()

	// Call complete callback
	if u.onComplete != nil {
		u.onComplete(summary)
	}

	return summary, nil
}

// updateSingleManga updates a single manga
func (u *Updater) updateSingleManga(manga *source.Manga) (*UpdateTask, error) {
	task := &UpdateTask{
		MangaID:    manga.ID,
		MangaTitle: manga.Title,
		SourceType: manga.SourceType,
		Status:     StatusChecking,
		StartedAt:  time.Now(),
	}

	// Get current chapter count from storage (if available)
	if u.storage != nil && u.storage.UpdateTracking != nil {
		tracking, err := u.storage.UpdateTracking.GetTracking(manga.ID)
		if err == nil && tracking != nil {
			task.OldChapterCount = tracking.ChapterCount
		}
	}

	// Fetch latest chapters from source
	src := u.sourceManager.GetSource(manga.ID)
	if src == nil {
		// Try to find source by type
		sources := u.sourceManager.GetSourcesByType(manga.SourceType)
		if len(sources) > 0 {
			src = sources[0]
		}
	}

	if src == nil {
		task.Status = StatusFailed
		task.Error = fmt.Errorf("source not found")
		task.CompletedAt = time.Now()
		return task, task.Error
	}

	chapters, err := src.ListChapters(manga.ID)
	if err != nil {
		task.Status = StatusFailed
		task.Error = err
		task.CompletedAt = time.Now()
		return task, err
	}

	task.NewChapterCount = len(chapters)
	task.NewChapters = chapters
	task.Status = StatusCompleted
	task.CompletedAt = time.Now()

	return task, nil
}

// filterManga filters manga based on update config
func (u *Updater) filterManga(allManga []*source.Manga) []*source.Manga {
	if !u.config.UpdateOnlyStarted && !u.config.UpdateOnlyCompleted {
		return allManga
	}

	filtered := make([]*source.Manga, 0)

	for _, manga := range allManga {
		// Check if manga has been read (if UpdateOnlyStarted is true)
		if u.config.UpdateOnlyStarted && u.storage != nil {
			history, err := u.storage.History.GetMangaHistory(manga.ID)
			if err != nil || len(history) == 0 {
				continue // Skip manga that hasn't been started
			}
		}

		// TODO: Implement completed filter when manga status is available

		filtered = append(filtered, manga)
	}

	return filtered
}

// addNotification adds a notification (internal)
func (u *Updater) addNotification(notif *Notification) {
	u.mu.Lock()
	u.notifications = append(u.notifications, notif)
	// Keep only last 100 notifications
	if len(u.notifications) > 100 {
		u.notifications = u.notifications[len(u.notifications)-100:]
	}
	u.mu.Unlock()

	// Call notification callback
	if u.onNotification != nil {
		u.onNotification(notif)
	}
}
