# CLAUDE.md - AI Assistant Guide for Miryokusha

## Project Overview

**Miryokusha** is a Terminal User Interface (TUI) client for Suwayomi/Tachiyomi servers, built using Go and the Charm framework. The project provides a beautiful, keyboard-driven interface for managing and reading manga from both Suwayomi servers and local files.

### Project Metadata
- **Language**: Go (Golang)
- **License**: GNU General Public License v3 (GPLv3)
- **UI Framework**: Charm (Bubble Tea, Lip Gloss, Bubbles)
- **Backend**: Suwayomi Server API + Local File Support
- **Repository**: github.com/Justice-Caban/Miryokusha
- **Stage**: Early development / Initial setup

### Supported Sources
1. **Suwayomi Server** - Remote manga library via HTTP API
2. **Local Files** - Direct reading from filesystem:
   - CBZ (Comic Book ZIP) archives
   - CBR (Comic Book RAR) archives
   - PDF files
   - Image directories (JPG, PNG, WEBP)

## Repository Structure

### Current State
The repository is newly initialized with:
- `.gitignore` - Go-specific ignore patterns
- `LICENSE` - GPLv3 license file
- No source code yet (awaiting initial implementation)

### Expected Project Structure
```
Miryokusha/
├── cmd/
│   └── miryokusha/          # Main application entry point
│       └── main.go
├── internal/
│   ├── config/              # Configuration management
│   │   ├── config.go
│   │   └── validation.go
│   ├── suwayomi/           # Suwayomi API client
│   │   ├── client.go
│   │   ├── types.go
│   │   └── endpoints.go
│   ├── local/              # Local file handling
│   │   ├── scanner.go      # Directory scanner
│   │   ├── reader.go       # Archive reader (CBZ/CBR/PDF)
│   │   ├── types.go        # Local manga types
│   │   └── watcher.go      # File system watcher (optional)
│   ├── source/             # Unified source interface
│   │   ├── interface.go    # Source abstraction
│   │   └── manager.go      # Source manager
│   ├── tui/                # TUI components
│   │   ├── app.go          # Main application model
│   │   ├── styles.go       # Lip Gloss styles
│   │   ├── source/         # Source selection view
│   │   ├── library/        # Manga library view
│   │   │   ├── view.go     # Main library view
│   │   │   ├── filters.go  # Filtering (read, unread, downloaded)
│   │   │   └── sort.go     # Sorting options
│   │   ├── reader/         # Manga reader view
│   │   │   ├── view.go     # Reader interface
│   │   │   ├── modes.go    # Reading modes (single, double, webtoon)
│   │   │   └── controls.go # Reader controls
│   │   ├── extensions/     # Extension management view
│   │   │   ├── browser.go  # Browse available extensions
│   │   │   ├── installed.go # Installed extensions list
│   │   │   └── detail.go   # Extension details
│   │   ├── categories/     # Category management
│   │   │   ├── view.go     # Category list
│   │   │   └── editor.go   # Create/edit categories
│   │   ├── downloads/      # Download management
│   │   │   ├── queue.go    # Download queue
│   │   │   └── manager.go  # Download manager
│   │   ├── browse/         # Browse sources
│   │   │   ├── popular.go  # Popular manga
│   │   │   ├── latest.go   # Latest updates
│   │   │   └── search.go   # Global search
│   │   ├── tracking/       # Tracker integration
│   │   │   ├── list.go     # Tracker list
│   │   │   └── bind.go     # Bind manga to tracker
│   │   ├── updates/        # Library updates
│   │   │   └── view.go     # Update history
│   │   └── settings/       # Settings view
│   └── storage/            # Local data persistence
│       ├── cache.go
│       └── history.go
├── pkg/                    # Public API (if needed)
├── test/                   # Integration tests
├── docs/                   # Documentation
├── .gitignore
├── LICENSE
├── README.md
├── CLAUDE.md              # This file
├── go.mod                 # Go module definition
├── go.sum                 # Dependency checksums
└── Makefile               # Build automation
```

## Architecture & Design Patterns

### 1. Configuration-First Approach

**Server connectivity uses a config file strategy:**

**Config Location**: `~/.config/miryokusha/config.yaml`

**Expected Config Structure**:
```yaml
servers:
  - name: "Home Server"
    url: "http://localhost:4567"
    default: true
  - name: "Remote Server"
    url: "https://manga.example.com"
    auth:
      type: basic
      username: user
      password: encrypted_pass

local_sources:
  - name: "Manga Collection"
    path: "~/Manga"
    recursive: true
    watch: false  # File system watching for new files
  - name: "Downloads"
    path: "~/Downloads/Manga"
    recursive: false
    file_types: [".cbz", ".cbr", ".pdf"]

preferences:
  theme: dark
  default_source: "server:0"  # or "local:0"
  cache_size_mb: 500
  auto_mark_read: true
  image_fit: "width"  # width, height, both
  double_page_mode: false

library:
  default_category: "Reading"
  update_interval: 12h  # Check for new chapters every 12 hours
  update_only_completed: false
  update_only_started: true
  auto_download_new_chapters: false
  delete_read_chapters: false
  sort_mode: "alphabetical"  # alphabetical, last_read, last_updated, unread_count
  filter_downloaded: false
  filter_unread: false
  filter_bookmarked: false

categories:
  default: ["Reading", "Completed", "Plan to Read", "On Hold"]

downloads:
  location: "~/.local/share/miryokusha/downloads"
  simultaneous_downloads: 3
  only_on_wifi: false  # For future mobile version
  delete_after_read: false

tracking:
  auto_track: true
  auto_update_status: true
  services:
    - name: "MyAnimeList"
      enabled: false
      username: ""
      token: ""
    - name: "AniList"
      enabled: false
      username: ""
      token: ""
    - name: "Kitsu"
      enabled: false
      username: ""
      token: ""

reader:
  reading_mode: "single_page"  # single_page, double_page, webtoon
  reading_direction: "ltr"     # ltr, rtl, vertical
  background_color: "#000000"
  preload_pages: 3
  show_page_number: true
  keep_screen_on: true
  volume_key_navigation: true

extensions:
  auto_update_check: true
  update_check_interval: 24h
  show_nsfw: false  # Hide NSFW extensions by default
  language_filter: ["en", "ja"]  # Only show extensions for these languages
```

**Implementation Guidelines**:
- Use `viper` or `koanf` for config management
- Support environment variable overrides
- Validate server URLs on load
- Encrypt sensitive data (passwords, tokens)
- Graceful fallback if config is missing/corrupt

### 2. TUI Architecture (Bubble Tea Pattern)

**The Elm Architecture** (Model-Update-View):

```go
type Model struct {
    currentView      string
    sourceList       SourceListModel
    library          LibraryModel
    reader           ReaderModel
    fileBrowser      FileBrowserModel
    extensionBrowser ExtensionBrowserModel
    extensionList    InstalledExtensionsModel
    config           *config.Config
    currentSource    source.Source  // Can be Suwayomi or Local
    sourceManager    *source.Manager
}

func (m Model) Init() tea.Cmd
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (m Model) View() string
```

**View Routing**:
- **Library** - Main view with manga organized by categories
- **Updates** - Recent chapter updates across all manga
- **Browse** - Discover manga from sources (popular, latest, search)
- **Downloads** - Manage downloaded chapters and queue
- **Reader** - Reading view with multiple modes (single, double, webtoon)
- **Extensions** - Browse and install extensions from Suwayomi
- **Tracking** - Sync with MyAnimeList, AniList, Kitsu
- **Categories** - Organize manga into collections
- **Settings** - App preferences and configuration
- **Search** - Global search across all sources or current source
- **File Browser** - Ad-hoc local file selection (local mode)

### 3. Suwayomi API Client

**HTTP Client Pattern**:
- Use `net/http` with connection pooling
- Implement retry logic with exponential backoff
- Handle authentication (if server requires it)
- Support both HTTP and HTTPS
- Graceful degradation on network errors

**Key API Endpoints** (to be implemented):

**Library & Manga**:
```
GET  /api/v1/source/list                # Get available sources
GET  /api/v1/manga/list                 # Get manga library
GET  /api/v1/manga/{id}                 # Get manga details
GET  /api/v1/manga/{id}/chapters        # Get chapters
GET  /api/v1/manga/{id}/category        # Get manga category
POST /api/v1/manga/{id}/category        # Set manga category
GET  /api/v1/chapter/{id}/page/{page}   # Get page image
PATCH /api/v1/chapter/{id}              # Update chapter (mark as read, bookmark)
POST /api/v1/chapter/batch              # Batch update chapters
```

**Categories**:
```
GET  /api/v1/category/list              # Get all categories
POST /api/v1/category                   # Create category
PUT  /api/v1/category/{id}              # Update category
DELETE /api/v1/category/{id}            # Delete category
GET  /api/v1/category/{id}/manga        # Get manga in category
```

**Downloads**:
```
GET  /api/v1/download/queue             # Get download queue
POST /api/v1/download/chapter/{id}      # Add chapter to download queue
DELETE /api/v1/download/chapter/{id}    # Remove from queue
POST /api/v1/download/start             # Start downloads
POST /api/v1/download/stop              # Stop downloads
POST /api/v1/download/clear             # Clear completed downloads
GET  /api/v1/download/status            # Get download status
```

**Updates**:
```
POST /api/v1/update/library             # Trigger library update
POST /api/v1/update/category/{id}       # Update specific category
GET  /api/v1/update/status              # Get update status
GET  /api/v1/update/summary             # Get recent updates
```

**Tracking**:
```
GET  /api/v1/tracker/list               # Get available trackers
POST /api/v1/tracker/{id}/login         # Login to tracker
GET  /api/v1/manga/{id}/tracker         # Get tracker status for manga
POST /api/v1/manga/{id}/tracker         # Bind manga to tracker
PUT  /api/v1/manga/{id}/tracker         # Update tracker status
DELETE /api/v1/manga/{id}/tracker       # Unbind manga from tracker
```

**Browse**:
```
GET  /api/v1/source/{id}/popular        # Get popular manga from source
GET  /api/v1/source/{id}/latest         # Get latest manga from source
GET  /api/v1/source/{id}/search         # Search manga in source
POST /api/v1/search/global              # Global search across all sources
```

**Extension Management**:
```
GET  /api/v1/extension/list                    # Get installed extensions
GET  /api/v1/extension/available              # Get available extensions (from repo)
POST /api/v1/extension/install/{pkgName}      # Install extension
POST /api/v1/extension/uninstall/{pkgName}    # Uninstall extension
POST /api/v1/extension/update/{pkgName}       # Update extension
GET  /api/v1/extension/{pkgName}              # Get extension details
GET  /api/v1/extension/icon/{pkgName}         # Get extension icon
```

**Extension Repositories**:
```
GET  /api/v1/repository/list                   # Get configured repositories
POST /api/v1/repository/add                    # Add custom repository
DELETE /api/v1/repository/{id}                 # Remove repository
```

### 4. Category Management

**Categories help organize manga library similar to Mihon/Tachiyomi:**

**Features**:
- Create custom categories (Reading, Completed, Plan to Read, etc.)
- Assign manga to multiple categories
- Category-specific update settings
- Default category for new manga
- Reorder categories
- Category-based filtering in library

**Implementation Pattern**:
```go
// internal/tui/categories/view.go
type CategoryModel struct {
    categories    []suwayomi.Category
    mangaCount    map[int]int  // Category ID -> manga count
    selectedIndex int
    editMode      bool
}

// Operations
func (m CategoryModel) CreateCategory(name string) tea.Cmd
func (m CategoryModel) DeleteCategory(id int) tea.Cmd
func (m CategoryModel) ReorderCategories(from, to int) tea.Cmd
func (m CategoryModel) AssignManga(mangaID int, categoryID int) tea.Cmd
```

**UI Layout**:
```
┌─ Categories ──────────────────────────────────────────────────────┐
│                                                                    │
│ ▸ All Manga                                              (1,247)  │
│ ▸ Reading                                                  (342)  │
│ ▸ Completed                                                (128)  │
│ ▸ Plan to Read                                             (89)   │
│ ▸ On Hold                                                  (23)   │
│                                                                    │
│ [n]ew  [e]dit  [d]elete  [↑↓]reorder  [Enter]view                │
└────────────────────────────────────────────────────────────────────┘
```

### 5. Download Management

**Offline reading support is essential for manga readers:**

**Features**:
- Download chapters for offline reading
- Download queue management (pause, resume, cancel)
- Automatic download of new chapters
- Storage management (view disk usage, delete old downloads)
- Download only on WiFi (future mobile version)
- Auto-delete read chapters
- Download priority (next chapter, all chapters, unread chapters)

**Implementation Pattern**:
```go
// internal/downloads/manager.go
type DownloadManager struct {
    queue        []DownloadItem
    active       map[string]*DownloadJob
    maxConcurrent int
    storage      string
    client       *suwayomi.Client
}

type DownloadItem struct {
    MangaID    string
    ChapterID  string
    Priority   int
    Status     DownloadStatus  // queued, downloading, completed, failed
    Progress   int             // 0-100
    Error      error
}

func (dm *DownloadManager) AddToQueue(chapterID string) error
func (dm *DownloadManager) Start() error
func (dm *DownloadManager) Pause() error
func (dm *DownloadManager) Cancel(chapterID string) error
func (dm *DownloadManager) GetStorageUsage() (int64, error)
```

**UI Layout**:
```
┌─ Downloads ───────────────────────────────────────────────────────┐
│                                                                    │
│ Active Downloads (2/3)                                             │
│ ▼ One Piece - Chapter 1089                         [████░░] 67%   │
│ ▼ Naruto - Chapter 700                             [██░░░░] 40%   │
│                                                                    │
│ Queue (5)                                                          │
│ ○ Bleach - Chapter 686                                             │
│ ○ Attack on Titan - Chapter 139                                   │
│ ○ My Hero Academia - Chapter 400                                  │
│                                                                    │
│ Storage: 2.3 GB / 50 GB                                            │
│ [p]ause  [r]esume  [c]ancel  [d]elete  [C]lear completed          │
└────────────────────────────────────────────────────────────────────┘
```

### 6. Tracking Integration

**Sync reading progress with external services like Mihon:**

**Supported Trackers**:
- MyAnimeList (MAL)
- AniList
- Kitsu
- MangaUpdates
- Shikimori
- Bangumi

**Features**:
- OAuth login for trackers
- Bind manga to tracker entry
- Auto-update reading progress
- Sync status (Reading, Completed, On Hold, Dropped, Plan to Read)
- Sync score/rating
- Manual sync or auto-sync
- Track multiple manga across different trackers

**Implementation Pattern**:
```go
// internal/tracking/tracker.go
type Tracker interface {
    Name() string
    Login(credentials map[string]string) error
    Search(query string) ([]TrackEntry, error)
    Bind(mangaID string, trackID string) error
    UpdateProgress(mangaID string, chaptersRead int) error
    UpdateStatus(mangaID string, status ReadingStatus) error
    UpdateScore(mangaID string, score float64) error
}

type ReadingStatus string

const (
    StatusReading    ReadingStatus = "reading"
    StatusCompleted  ReadingStatus = "completed"
    StatusOnHold     ReadingStatus = "on_hold"
    StatusDropped    ReadingStatus = "dropped"
    StatusPlanToRead ReadingStatus = "plan_to_read"
)
```

**UI Layout**:
```
┌─ Tracking: One Piece ─────────────────────────────────────────────┐
│                                                                    │
│ MyAnimeList              Status: Reading      Score: 9/10         │
│ ├─ Progress: 1089/1100+  Last Updated: 2025-11-14                │
│ └─ [Sync Now]  [Update Status]  [Unbind]                         │
│                                                                    │
│ AniList                  Status: Reading      Score: 95/100       │
│ ├─ Progress: 1089/1100+  Last Updated: 2025-11-14                │
│ └─ [Sync Now]  [Update Status]  [Unbind]                         │
│                                                                    │
│ + Add Tracker                                                      │
│                                                                    │
│ Auto-update: [✓] On chapter read   [✓] On status change           │
└────────────────────────────────────────────────────────────────────┘
```

### 7. Library Updates & Notifications

**Automatic checking for new chapters:**

**Features**:
- Global library update (check all manga)
- Category-specific updates
- Update scheduling (every X hours)
- Smart update (only update likely-to-update manga)
- Update only started manga
- Update only completed manga
- Update notifications
- Update history view

**Implementation Pattern**:
```go
// internal/updates/updater.go
type LibraryUpdater struct {
    client        *suwayomi.Client
    updateQueue   []UpdateTask
    lastUpdate    time.Time
    interval      time.Duration
    notifier      *Notifier
}

type UpdateTask struct {
    MangaID       string
    LastChecked   time.Time
    NewChapters   []Chapter
    Status        UpdateStatus
}

func (lu *LibraryUpdater) UpdateLibrary() error
func (lu *LibraryUpdater) UpdateCategory(categoryID int) error
func (lu *LibraryUpdater) ScheduleUpdates(interval time.Duration) error
```

### 8. Browse & Discover

**Discover new manga from sources:**

**Features**:
- Browse popular manga from each source
- Browse latest updates from each source
- Global search across all installed sources
- Source-specific filters (genre, status, etc.)
- Add manga to library from browse
- Preview manga before adding

**UI Layout**:
```
┌─ Browse: MangaDex ────────────────────────────────────────────────┐
│                                                                    │
│ [Popular] [Latest] [Search]                    Sort: [Popularity▼]│
│                                                                    │
│ ┌──────────┐  One Piece                                    ★ 9.2  │
│ │  [IMG]   │  Adventure, Action • Ongoing                         │
│ │          │  Latest: Ch. 1089 • 2 hours ago                      │
│ └──────────┘  [+] Add to Library   [Read]                         │
│                                                                    │
│ ┌──────────┐  Attack on Titan                               ★ 9.0  │
│ │  [IMG]   │  Action, Drama • Completed                           │
│ │          │  Latest: Ch. 139 • Final                             │
│ └──────────┘  [✓] In Library   [Read]                             │
│                                                                    │
│ [g]lobal search  [f]ilters  [s]ource  [↑↓]navigate  [Enter]view  │
└────────────────────────────────────────────────────────────────────┘
```

### 9. Advanced Library Features

**Library filtering, sorting, and bulk operations:**

**Filtering**:
- Filter by read status (all, read, unread)
- Filter by download status (all, downloaded, not downloaded)
- Filter by tracker status
- Filter by bookmark
- Filter by category
- Combine multiple filters

**Sorting**:
- Alphabetical (A-Z, Z-A)
- Last read
- Last updated
- Unread chapter count
- Total chapter count
- Date added

**Bulk Operations**:
- Select multiple manga
- Mark all as read/unread
- Download all chapters
- Delete downloads
- Change category
- Update tracker status
- Remove from library

### 10. Advanced Reader Features

**Enhanced reading experience:**

**Reading Modes**:
- Single page (one page at a time)
- Double page (two pages side-by-side)
- Webtoon mode (continuous vertical scroll)

**Reading Direction**:
- Left-to-right (Western comics)
- Right-to-left (Japanese manga)
- Vertical (Webtoons)

**Reader Features**:
- Page preloading (load next N pages)
- Bookmarks (save position in chapter)
- Page number display
- Background color customization
- Brightness control (future)
- Color filters (future)
- Crop borders
- Volume key navigation
- Tap zones for navigation
- Fullscreen mode
- Rotation lock

### 11. Backup & Restore

**Protect library and settings:**

**Features**:
- Export library backup (JSON format)
- Export reading history
- Export categories and settings
- Restore from backup
- Scheduled auto-backup
- Cloud storage sync (optional, via external tools)

### 12. Migration Tools

**Move manga between sources:**

**Features**:
- Migrate manga from one source to another
- Preserve reading history
- Preserve categories
- Batch migration
- Duplicate detection

### 13. Extension Management System

**Extension management is a core feature for Suwayomi integration:**

#### Extension Browser View

**Features**:
- List all available extensions from Suwayomi repository
- Filter by language (English, Japanese, etc.)
- Filter by category (manga, anime, etc.)
- Search extensions by name
- Show NSFW/SFW status
- Display extension icon, name, version, and description
- Show installation status (installed, available, update available)

**Implementation Pattern**:
```go
// internal/tui/extensions/browser.go
type BrowserModel struct {
    extensions      []suwayomi.Extension
    filteredList    []suwayomi.Extension
    selectedIndex   int
    searchQuery     string
    languageFilter  []string
    showNSFW        bool
    loading         bool
    installQueue    map[string]bool  // Track ongoing installations
}

func (m BrowserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "i", "enter":
            // Install selected extension
            return m, m.installExtension(m.selectedExtension())
        case "/":
            // Enter search mode
            return m.enableSearch(), nil
        case "f":
            // Toggle filters
            return m.toggleFilters(), nil
        }
    case extensionInstalledMsg:
        // Update extension list after installation
        return m.refreshList(), nil
    }
    return m, nil
}
```

#### Installed Extensions View

**Features**:
- List all installed extensions
- Show extension status (enabled, disabled, obsolete, updating)
- Display version and update availability
- Enable/disable extensions
- Uninstall extensions
- Update individual or all extensions
- View extension sources (manga sources provided by extension)

**UI Layout**:
```
┌─ Installed Extensions ────────────────────────────────────────────┐
│                                                                    │
│ ✓ MangaDex (en)                                      v1.4.2  [UP] │
│   • MangaDex - High quality manga source                          │
│   • 342 manga in library                                          │
│                                                                    │
│ ✓ MangaSee (en)                                      v2.1.0       │
│   • MangaSee - Popular manga source                               │
│   • 128 manga in library                                          │
│                                                                    │
│ ✗ NHentai (en) [NSFW]                                v1.2.3       │
│   • Disabled - Press 'e' to enable                                │
│                                                                    │
│ [i]nfo  [e]nable/disable  [u]ninstall  [U]pdate all  [r]efresh   │
└────────────────────────────────────────────────────────────────────┘
```

#### Extension Details Panel

**Shows**:
- Full extension name and description
- Version information
- Developer/maintainer
- Languages supported
- Installation size
- Last update date
- Number of manga sources provided
- List of sources (clickable to browse that source)

**Actions Available**:
- Install/Uninstall
- Enable/Disable
- Update (if available)
- View source code/repository (if available)
- Browse manga from this extension

### 14. Local File Support

**Two Primary Strategies for Local File Reading:**

#### Strategy 1: CLI File Arguments
Users can pass files directly via command-line arguments:

```bash
# Read a single file
miryokusha read ~/Manga/OnePiece.cbz

# Read multiple files
miryokusha read ~/Manga/*.cbz

# Read from specific chapter
miryokusha read ~/Manga/OnePiece.cbz --chapter 5

# Read from specific page
miryokusha read ~/Manga/OnePiece.cbz --page 15
```

**Implementation Pattern**:
```go
// cmd/miryokusha/main.go
func main() {
    if len(os.Args) > 2 && os.Args[1] == "read" {
        filePath := os.Args[2]
        reader, err := local.NewReader(filePath)
        if err != nil {
            log.Fatal(err)
        }
        // Launch TUI with this specific file
        runReaderMode(reader)
    } else {
        // Launch normal TUI (library browser)
        runNormalMode()
    }
}
```

#### Strategy 2: Directory Scanning
Scan configured directories for manga files:

```go
// internal/local/scanner.go
type Scanner struct {
    basePath   string
    recursive  bool
    fileTypes  []string
}

func (s *Scanner) Scan() ([]*LocalManga, error) {
    var manga []*LocalManga

    err := filepath.Walk(s.basePath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        // Skip directories unless recursive
        if info.IsDir() && !s.recursive {
            return filepath.SkipDir
        }

        // Check file extension
        ext := filepath.Ext(path)
        if s.isSupported(ext) {
            m, err := s.parseFile(path, info)
            if err == nil {
                manga = append(manga, m)
            }
        }

        return nil
    })

    return manga, err
}
```

**Supported Archive Formats**:
- **CBZ**: ZIP archive containing images (use `archive/zip`)
- **CBR**: RAR archive containing images (use `github.com/nwaples/rardecode`)
- **PDF**: PDF files with images (use `github.com/gen2brain/go-fitz`)
- **Directories**: Folders containing image files directly

**File Organization Detection**:
```
~/Manga/
├── OnePiece/
│   ├── Vol 1/
│   │   ├── 001.jpg
│   │   ├── 002.jpg
│   │   └── ...
│   └── Vol 2/
│       └── ...
├── Naruto.cbz
└── Bleach.pdf
```

**Metadata Extraction**:
- Parse filenames for manga title, volume, chapter
- Support ComicInfo.xml (standard comic metadata)
- Extract cover images for thumbnails
- Remember reading position

### 15. Unified Source Interface

**Abstract sources for flexibility**:

```go
// internal/source/interface.go
type Source interface {
    Type() SourceType  // "suwayomi" or "local"
    Name() string
    GetLibrary() ([]*Manga, error)
    GetManga(id string) (*Manga, error)
    GetChapters(mangaID string) ([]*Chapter, error)
    GetPage(chapterID string, page int) ([]byte, error)
}

type SourceType string

const (
    SourceTypeSuwayomi SourceType = "suwayomi"
    SourceTypeLocal    SourceType = "local"
)

// Implementations
type SuwayomiSource struct { /* ... */ }
type LocalSource struct { /* ... */ }
```

**Benefits**:
- Seamless switching between sources in TUI
- Unified interface for library browsing
- Consistent reading experience
- Easy to add new source types (Komga, Kavita, etc.)

### 16. Error Handling Strategy

**Graceful Degradation**:
- Network errors → Show connection status indicator
- Missing config → Launch first-run setup wizard
- API errors → Display user-friendly error messages
- Cache miss → Fall back to network fetch
- Corrupted archive → Show error, skip to next file
- Unsupported format → Warn user, suggest conversion
- Missing permissions → Clear error about file access
- Invalid path → Suggest path correction
- Extension install failure → Show clear error, suggest troubleshooting
- Extension repo unavailable → Cache last known state, retry later
- Extension conflicts → Warn user, offer resolution options

**User Experience**:
- Never crash on network issues
- Show loading indicators for async operations
- Provide actionable error messages
- Allow retry without restarting app

## Development Workflows

### Initial Setup (First Time)

```bash
# Initialize Go module
go mod init github.com/Justice-Caban/Miryokusha

# Install Charm dependencies
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/lipgloss
go get github.com/charmbracelet/bubbles

# Install other dependencies
go get github.com/spf13/viper  # Config management
go get github.com/go-resty/resty/v2  # HTTP client (optional)
go get github.com/nwaples/rardecode  # RAR archive support
go get github.com/gen2brain/go-fitz  # PDF support

# Create basic structure
mkdir -p cmd/miryokusha internal/{config,suwayomi,local,source,tui,storage}

# Build and run
go build -o bin/miryokusha ./cmd/miryokusha
./bin/miryokusha
```

### Development Workflow

```bash
# Run in development mode (TUI library browser)
go run ./cmd/miryokusha

# Run with specific server
go run ./cmd/miryokusha --server http://localhost:4567

# Read local file directly
go run ./cmd/miryokusha read ~/Manga/OnePiece.cbz

# Read from local directory
go run ./cmd/miryokusha --local ~/Manga

# Scan and list local manga
go run ./cmd/miryokusha scan ~/Manga --recursive

# Build optimized binary
go build -ldflags="-s -w" -o bin/miryokusha ./cmd/miryokusha

# Run tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Format code
go fmt ./...

# Lint (requires golangci-lint)
golangci-lint run
```

### Git Workflow

**Branch Strategy**:
- `main` - Stable releases
- `develop` - Active development
- `feature/*` - Feature branches
- `claude/*` - AI-assisted development branches

**Commit Conventions**:
- `feat:` - New features
- `fix:` - Bug fixes
- `refactor:` - Code restructuring
- `docs:` - Documentation updates
- `test:` - Test additions/updates
- `chore:` - Maintenance tasks

**Example**:
```bash
git checkout -b feature/server-config
# Make changes
git add .
git commit -m "feat: add server configuration management with viper"
git push -u origin feature/server-config
```

## Code Conventions

### Go Style Guidelines

1. **Follow Standard Go Style**:
   - Use `gofmt` for formatting
   - Follow [Effective Go](https://go.dev/doc/effective_go)
   - Use [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

2. **Naming Conventions**:
   - Packages: lowercase, single word (e.g., `config`, `suwayomi`)
   - Interfaces: noun or adjective (e.g., `Reader`, `Configurable`)
   - Getters: no "Get" prefix (e.g., `Config()` not `GetConfig()`)

3. **Error Handling**:
   ```go
   // Wrap errors with context
   if err != nil {
       return fmt.Errorf("failed to fetch manga %d: %w", id, err)
   }

   // Handle errors immediately
   result, err := fetchManga(id)
   if err != nil {
       return err
   }
   ```

4. **Concurrency**:
   ```go
   // Use contexts for cancellation
   ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
   defer cancel()

   // Always handle channel closure
   select {
   case result := <-resultChan:
       return result
   case <-ctx.Done():
       return ctx.Err()
   }
   ```

### TUI-Specific Conventions

1. **Component Organization**:
   - Each view is a separate package under `internal/tui/`
   - Each component implements Bubble Tea's Model interface
   - Share styles via `internal/tui/styles.go`

2. **Keyboard Shortcuts**:

   **Global**:
   - `q` - Quit application
   - `?` - Help/shortcuts
   - `Esc` - Go back
   - `Tab` - Switch panels

   **Navigation**:
   - `↑↓` or `jk` - Navigate lists
   - `Enter` - Select/confirm
   - `PgUp/PgDn` - Page up/down
   - `Home/End` - Jump to first/last

   **Library & Sources**:
   - `/` - Search (current source or global)
   - `s` - Switch source (server/local)
   - `o` - Open local file
   - `r` - Refresh/rescan current source
   - `c` - Manage categories
   - `F` - Filter library
   - `S` - Sort library
   - `u` - Check for updates

   **Manga Actions**:
   - `m` - Mark as read/unread
   - `b` - Bookmark chapter
   - `D` - Download chapter(s)
   - `t` - Track manga (bind to MAL/AniList)
   - `C` - Change category
   - `x` - Select multiple (bulk actions)

   **Extensions** (in extension views):
   - `e` - Browse available extensions
   - `i` - Install selected extension
   - `u` - Uninstall selected extension
   - `U` - Update all extensions
   - `Space` - Enable/disable extension
   - `f` - Toggle filters (language, NSFW)
   - `d` - View extension details

3. **Visual Design**:
   - Use Lip Gloss for all styling
   - Support both light and dark themes
   - Maintain consistent spacing and borders
   - Show loading indicators for async operations

## Dependencies Management

### Core Dependencies
```go
require (
    github.com/charmbracelet/bubbletea v0.25.0
    github.com/charmbracelet/lipgloss v0.9.1
    github.com/charmbracelet/bubbles v0.18.0
    github.com/spf13/viper v1.18.0
)
```

### Recommended Additional Dependencies
- **HTTP Client**: `github.com/go-resty/resty/v2` or standard `net/http`
- **Image Processing**: `github.com/disintegration/imaging` (for manga pages)
- **Archive Support**:
  - `archive/zip` (built-in) - CBZ files
  - `github.com/nwaples/rardecode` - CBR files
  - `github.com/gen2brain/go-fitz` - PDF files
- **File Watching**: `github.com/fsnotify/fsnotify` (monitor directories)
- **XML Parsing**: `encoding/xml` (built-in) - ComicInfo.xml metadata
- **Logging**: `github.com/rs/zerolog` or `log/slog` (Go 1.21+)
- **Testing**: `github.com/stretchr/testify`
- **CLI Framework**: `github.com/spf13/cobra` (for command-line arguments)

### Dependency Update Strategy
```bash
# Update all dependencies
go get -u ./...
go mod tidy

# Update specific dependency
go get -u github.com/charmbracelet/bubbletea@latest

# Verify no breaking changes
go test ./...
```

## Testing Strategy

### Unit Tests
```go
// internal/config/config_test.go
func TestLoadConfig(t *testing.T) {
    cfg, err := Load("testdata/valid_config.yaml")
    require.NoError(t, err)
    assert.Equal(t, "Home Server", cfg.Servers[0].Name)
}
```

### Integration Tests
```go
// test/suwayomi_test.go
func TestSuwayomiClient(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    // Test against real or mock Suwayomi server
}
```

### TUI Testing
```go
// Use Bubble Tea's testing utilities
func TestAppModel(t *testing.T) {
    m := NewModel()
    m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
    // Assert state changes
}
```

## AI Assistant Guidelines

### When Working on This Project

1. **Always Check Current State**:
   - Run `go mod tidy` after adding dependencies
   - Check `go.mod` for existing dependencies before adding new ones
   - Review existing code patterns before implementing new features

2. **Configuration Changes**:
   - Update config structs in `internal/config/config.go`
   - Update example config in documentation
   - Add validation for new config fields
   - Handle backward compatibility

3. **Source Implementation Changes**:
   - Document Suwayomi API endpoints in code comments
   - Add integration tests for new endpoints/sources
   - Handle API versioning gracefully
   - Cache responses when appropriate
   - Test local file reading with various archive formats
   - Ensure consistent behavior across source types

4. **TUI Development**:
   - Test TUI changes in terminal (may need user verification)
   - Ensure keyboard navigation works smoothly
   - Maintain consistent styling across views
   - Handle terminal resize events

5. **Error Messages**:
   - Make errors user-friendly for TUI display
   - Include actionable suggestions (e.g., "Check server URL in config")
   - Log detailed errors for debugging
   - Don't expose internal errors to users

6. **Performance Considerations**:
   - Load manga images lazily
   - Implement pagination for large libraries
   - Cache frequently accessed data
   - Use goroutines for network operations (with proper error handling)

### Common Tasks for AI Assistants

**Adding a New View**:
1. Create package under `internal/tui/viewname/`
2. Implement Model with Init/Update/View methods
3. Add routing in main app model
4. Define keyboard shortcuts
5. Add styles to `internal/tui/styles.go`
6. Update help text

**Adding a New Source Type**:
1. Implement `source.Source` interface in new package
2. Add source types in `internal/source/interface.go`
3. Document source capabilities and limitations
4. Add error handling for source-specific errors
5. Add integration test with sample data
6. Update source manager to support new type
7. Update TUI to handle source-specific features

**Adding Local File Format Support**:
1. Add format detection in `internal/local/reader.go`
2. Implement reader for the format (e.g., EPUB, CBT)
3. Add format to config `file_types` list
4. Add tests with sample files
5. Update documentation with supported formats
6. Handle format-specific edge cases

**Implementing Extension Management Features**:
1. Add extension API methods in `internal/suwayomi/client.go`
2. Create extension types in `internal/suwayomi/types.go`
3. Implement extension browser view in `internal/tui/extensions/browser.go`
4. Implement installed extensions view in `internal/tui/extensions/installed.go`
5. Add extension caching to avoid repeated API calls
6. Handle async operations (install/update/uninstall)
7. Show progress indicators for extension operations
8. Add keyboard shortcuts for extension management
9. Test with various extension states (installing, updating, failed)
10. Handle extension repository updates

**Updating Configuration**:
1. Update struct in `internal/config/config.go`
2. Add validation logic
3. Update default values
4. Update example config in docs
5. Handle migration from old config format

### Questions to Ask User

Before implementing major features, clarify:
- Server authentication requirements?
- Offline reading support needed?
- Manga download/caching strategy?
- Multi-server/source switching behavior?
- Keyboard shortcut preferences?
- Theme customization level?
- Local file organization expectations?
- Archive format priority (CBZ vs CBR vs PDF)?
- File system watching (auto-detect new files)?
- Metadata handling strategy (ComicInfo.xml, filename parsing)?
- Extension auto-update preferences?
- NSFW content display preferences?
- Default language filters for extensions?
- Extension installation confirmation required?

## Build and Release

### Building for Distribution

```bash
# Build for current platform
make build

# Cross-compile for multiple platforms
GOOS=linux GOARCH=amd64 go build -o bin/miryokusha-linux-amd64 ./cmd/miryokusha
GOOS=darwin GOARCH=arm64 go build -o bin/miryokusha-darwin-arm64 ./cmd/miryokusha
GOOS=windows GOARCH=amd64 go build -o bin/miryokusha-windows-amd64.exe ./cmd/miryokusha
```

### Release Checklist
- [ ] Update version in code
- [ ] Update CHANGELOG.md
- [ ] Run full test suite
- [ ] Build binaries for all platforms
- [ ] Test binaries on target platforms
- [ ] Create GitHub release
- [ ] Update README with new features

## Resources

### Official Documentation
- [Suwayomi API Docs](https://github.com/Suwayomi/Suwayomi-Server/wiki/API)
- [Bubble Tea Tutorial](https://github.com/charmbracelet/bubbletea/tree/master/tutorials)
- [Lip Gloss Examples](https://github.com/charmbracelet/lipgloss/tree/master/examples)
- [Go Project Layout](https://github.com/golang-standards/project-layout)

### Similar Projects
- [komga-tui](https://github.com/faldez/komga-tui) - Komga TUI client
- [glow](https://github.com/charmbracelet/glow) - Markdown reader TUI
- [lazygit](https://github.com/jesseduffield/lazygit) - Git TUI

## Troubleshooting

### Common Issues

**Config file not found**:
```go
// Ensure config directory exists
configDir := filepath.Join(os.UserConfigDir(), "miryokusha")
os.MkdirAll(configDir, 0755)
```

**Server connection fails**:
- Verify server URL in config
- Check network connectivity
- Ensure server is running
- Check firewall rules

**Local file reading fails**:
- Check file permissions (readable by user)
- Verify file is not corrupted (try unzipping manually)
- Ensure path exists and is correct
- Check supported format (.cbz, .cbr, .pdf)
- For CBR: ensure rardecode is installed
- For PDF: ensure go-fitz is installed

**Directory scanning issues**:
- Verify directory path exists
- Check recursive flag if files in subdirectories
- Ensure read permissions on directory
- Check file type filters in config
- Look for hidden files (start with .)

**Extension installation fails**:
- Check Suwayomi server version compatibility
- Verify extension repository is accessible
- Check available disk space on server
- Ensure extension package name is correct
- Check server logs for detailed error messages
- Verify no conflicting extensions are installed

**Extension not appearing in list**:
- Check language filters in config
- Check NSFW filter settings
- Verify extension repository is up to date
- Force refresh extension list
- Check server extension repository configuration

**TUI rendering issues**:
- Ensure terminal supports ANSI colors
- Check terminal size (minimum 80x24)
- Update Bubble Tea to latest version

**Build errors**:
```bash
# Clean module cache
go clean -modcache
go mod download
go mod tidy
```

## Contributing

For contributors and AI assistants:
1. Read this CLAUDE.md thoroughly
2. Follow Go and project conventions
3. Write tests for new features
4. Update documentation
5. Test in actual terminal environment
6. Keep commits focused and well-described

---

**Last Updated**: 2025-11-15
**Project Status**: Initial setup phase

**Priority Next Steps**:
1. Initialize Go modules
2. Set up basic TUI framework
3. Implement source abstraction layer
4. Implement server configuration
5. Implement basic library view with categories
6. Implement manga reader with multiple modes
7. Implement extension browser and management
8. Implement download management
9. Implement library updates
10. Implement tracking integration

**Secondary Features** (implement after core functionality):
11. Local file reading (CBZ/CBR/PDF)
12. Directory scanning
13. Browse/discover views
14. Backup & restore
15. Migration tools
16. Advanced filters and bulk operations
