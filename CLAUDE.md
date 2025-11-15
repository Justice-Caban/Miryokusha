# CLAUDE.md - AI Assistant Guide for Miryokusha

## Project Overview

**Miryokusha** is a Terminal User Interface (TUI) client for Suwayomi/Tachiyomi servers, built using Go and the Charm framework. The project provides a beautiful, keyboard-driven interface for managing and reading manga from Suwayomi servers.

### Project Metadata
- **Language**: Go (Golang)
- **License**: GNU General Public License v3 (GPLv3)
- **UI Framework**: Charm (Bubble Tea, Lip Gloss, Bubbles)
- **Backend**: Suwayomi Server API
- **Repository**: github.com/Justice-Caban/Miryokusha
- **Stage**: Early development / Initial setup

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
│   ├── tui/                # TUI components
│   │   ├── app.go          # Main application model
│   │   ├── styles.go       # Lip Gloss styles
│   │   ├── server/         # Server selection view
│   │   ├── library/        # Manga library view
│   │   ├── reader/         # Manga reader view
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

preferences:
  theme: dark
  default_server: 0
  cache_size_mb: 500
  auto_mark_read: true
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
    currentView string
    serverList  ServerListModel
    library     LibraryModel
    reader      ReaderModel
    config      *config.Config
    suwayomi    *suwayomi.Client
}

func (m Model) Init() tea.Cmd
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (m Model) View() string
```

**View Routing**:
- Server selection (on first run or server switch)
- Library browser (main view)
- Manga reader (reading view)
- Settings/preferences
- Search interface

### 3. Suwayomi API Client

**HTTP Client Pattern**:
- Use `net/http` with connection pooling
- Implement retry logic with exponential backoff
- Handle authentication (if server requires it)
- Support both HTTP and HTTPS
- Graceful degradation on network errors

**Key API Endpoints** (to be implemented):
```
GET  /api/v1/source/list          # Get available sources
GET  /api/v1/manga/list           # Get manga library
GET  /api/v1/manga/{id}           # Get manga details
GET  /api/v1/manga/{id}/chapters  # Get chapters
GET  /api/v1/chapter/{id}/page/{page}  # Get page image
```

### 4. Error Handling Strategy

**Graceful Degradation**:
- Network errors → Show connection status indicator
- Missing config → Launch first-run setup wizard
- API errors → Display user-friendly error messages
- Cache miss → Fall back to network fetch

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

# Create basic structure
mkdir -p cmd/miryokusha internal/{config,suwayomi,tui,storage}

# Build and run
go build -o bin/miryokusha ./cmd/miryokusha
./bin/miryokusha
```

### Development Workflow

```bash
# Run in development mode
go run ./cmd/miryokusha

# Run with specific server
go run ./cmd/miryokusha --server http://localhost:4567

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
   - `q` - Quit application
   - `?` - Help/shortcuts
   - `↑↓` - Navigate lists
   - `Enter` - Select/confirm
   - `Esc` - Go back
   - `/` - Search
   - `Tab` - Switch panels

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
- **Logging**: `github.com/rs/zerolog` or `log/slog` (Go 1.21+)
- **Testing**: `github.com/stretchr/testify`

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

3. **API Client Changes**:
   - Document Suwayomi API endpoints in code comments
   - Add integration tests for new endpoints
   - Handle API versioning gracefully
   - Cache responses when appropriate

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

**Adding a New API Endpoint**:
1. Define method in `internal/suwayomi/client.go`
2. Add response types in `internal/suwayomi/types.go`
3. Document endpoint and parameters
4. Add error handling
5. Add integration test
6. Update client cache if needed

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
- Multi-server switching behavior?
- Keyboard shortcut preferences?
- Theme customization level?

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
**Next Steps**: Initialize Go modules, set up basic TUI framework, implement server configuration
