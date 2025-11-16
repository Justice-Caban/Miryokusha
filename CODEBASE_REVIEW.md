# Codebase Review - Miryokusha

**Date**: 2025-11-16
**Reviewer**: Claude (AI Assistant)
**Branch**: claude/claude-md-mi0n3god6hmzm6eq-013oeKK9iLu9VL9X7gAYUrxt

## Summary

Comprehensive code review scanning for mistakes, missing features, and redundancy.

**Build Status**: ✅ Compiles successfully
**Go Vet Status**: ✅ Passes (after mutex fix)
**Test Coverage**: ❌ No tests exist

---

## Critical Issues

### 1. ✅ FIXED: Mutex Copying Bug (HIGH PRIORITY)

**Location**: `internal/tui/downloads/downloads.go:60`

**Issue**: The `DownloadStats` struct contains a `sync.RWMutex` and was being passed by value in the Model, causing illegal mutex copying when Bubble Tea methods (Init, Update, View) pass Model by value.

**Impact**: Runtime panics, race conditions, undefined behavior

**Fix Applied**:
- Changed `stats downloads.DownloadStats` to `stats *downloads.DownloadStats`
- Updated all references to use pointer
- Added nil checks for safety

**Commit**: cc95f5d - "fix: resolve mutex copying bug in downloads.Model"

---

## Missing Features & Stubs

### 1. Browse View Not Implemented (MEDIUM PRIORITY)

**Location**: `internal/tui/app.go:368`

**Status**: Shows placeholder only
```go
case ViewBrowse:
    content = m.renderPlaceholderView("Browse", "🔍 Browse manga sources here")
```

**Impact**: Users cannot browse/discover new manga from sources

**Recommended Action**:
- Create `internal/tui/browse/` package
- Implement popular, latest, and search views
- Add source filtering and pagination
- Reference: CLAUDE.md sections 8 & 9

### 2. Reader Image Rendering (MEDIUM PRIORITY)

**Location**: `internal/tui/reader/reader.go:390-410`

**Status**: Shows placeholder text instead of actual images
```go
placeholder := fmt.Sprintf(
    "[Page %d]\n\n"+
    "Image: %s\n\n"+
    "(Image rendering not yet implemented)\n\n"+
    "Use ← → or h l to navigate\n"+
    "Press 'c' to hide controls",
    m.currentPage+1,
    page.URL,
)
```

**Impact**: Core reading functionality incomplete

**Recommended Action**:
- Terminal image rendering is complex and has limitations
- Options:
  1. Use sixel/kitty protocol for terminals that support it
  2. ASCII art representation (low fidelity)
  3. Open external image viewer
  4. Use terminal graphics libraries (tcell/termimg)
- Document limitation in README that TUI is primarily for library management
- Consider web-based reader as primary reading interface

### 3. GetExtensionSources Stub (LOW PRIORITY)

**Location**: `internal/suwayomi/extensions.go:128-132`

**Status**: Returns empty list
```go
func (c *Client) GetExtensionSources(packageName string) ([]*ExtensionSource, error) {
    // For now, we could use GraphQL to query sources
    // This would require a custom GraphQL query
    // For now, return empty list
    return []*ExtensionSource{}, nil
}
```

**Impact**: Extension details panel cannot show available sources

**Recommended Action**:
- Implement GraphQL query for sources
- Add to extension details view
- Allow browsing manga from specific extension

### 4. Local File GetAllPages Incomplete (LOW PRIORITY)

**Location**: `internal/source/local.go:379-395`

**Status**: Returns Page objects without ImageData
```go
func (ls *LocalSource) GetAllPages(chapterID string) ([]*Page, error) {
    // ...
    // For now, return placeholder - full implementation would extract images
    pages := make([]*Page, chapter.PageCount)
    for i := 0; i < chapter.PageCount; i++ {
        pages[i] = &Page{
            Index: i,
            URL:   "", // Local files don't have URLs
        }
    }
    return pages, nil
}
```

**Impact**: Local file reading won't work for bulk operations

**Recommended Action**:
- Extract actual images from CBZ/CBR/PDF files
- Populate ImageData field
- Handle different archive formats properly
- This is related to issue #2 (image rendering)

### 5. PDF Page Count Estimation (LOW PRIORITY)

**Location**: `internal/source/local.go:266-281`

**Status**: Uses crude file size heuristic
```go
// For now, estimate 20 pages as placeholder
pageCount := 20 // Placeholder

// Estimate ~50KB per page as a rough heuristic
estimatedPages := int(stat.Size() / 50000)
```

**Impact**: Inaccurate page counts for PDF files

**Recommended Action**:
- Use proper PDF library (e.g., `github.com/pdfcpu/pdfcpu` or `github.com/unidoc/unipdf`)
- Extract actual page count from PDF metadata
- Handle encrypted/malformed PDFs gracefully

---

## Code Quality Issues

### 1. Style Redundancy (MEDIUM PRIORITY)

**Analysis**: Multiple view files define identical style variables

**Duplicates Found**:
- `titleStyle`: Defined 7 times (library, downloads, extensions, settings, history, categories, reader)
- `helpStyle`: Defined 7 times
- `mutedStyle`: Defined 4 times
- `sectionStyle`: Defined 3 times
- `successStyle`: Defined 3 times
- `errorStyle`: Defined 3 times

**Impact**:
- Code duplication (~50-70 lines)
- Inconsistent styling if one is modified
- Maintenance burden

**Recommended Action**:
Create `internal/tui/theme/styles.go`:

```go
package theme

import "github.com/charmbracelet/lipgloss"

// Common styles used across all views
var (
    TitleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(ColorPrimary).
        MarginBottom(1)

    SectionStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(ColorSecondary).
        MarginTop(1)

    HelpStyle = lipgloss.NewStyle().
        Foreground(ColorMuted).
        MarginTop(1)

    MutedStyle = lipgloss.NewStyle().
        Foreground(ColorMuted)

    SuccessStyle = lipgloss.NewStyle().
        Foreground(ColorSuccess)

    ErrorStyle = lipgloss.NewStyle().
        Foreground(ColorError)

    WarningStyle = lipgloss.NewStyle().
        Foreground(ColorWarning)

    // Box/Container styles
    BoxStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(ColorBorder).
        Padding(1, 2)

    HighlightStyle = lipgloss.NewStyle().
        Background(ColorPrimary).
        Foreground(lipgloss.Color("#000000")).
        Bold(true)
)
```

Then update all view files to use `theme.TitleStyle`, etc.

**Estimated Effort**: 2-3 hours
**Files to Update**: 7 view files in `internal/tui/*/`

### 2. No Tests (HIGH PRIORITY)

**Status**: Project has zero test files

**Impact**:
- No confidence in refactoring
- Regressions not caught
- Hard to verify bug fixes
- Poor documentation of expected behavior

**Recommended Action**:
Start with critical path tests:

1. **Storage Layer** (`internal/storage/*_test.go`):
   - Database migrations
   - CRUD operations
   - Transaction rollback
   - Schema versioning

2. **Source Layer** (`internal/source/*_test.go`):
   - Local file parsing (CBZ, CBR, PDF)
   - Suwayomi API client
   - GraphQL queries

3. **Updates Layer** (`internal/updates/*_test.go`):
   - Smart update algorithm
   - Filter logic (started, completed manga)
   - Running average calculations

4. **Downloads Layer** (`internal/downloads/*_test.go`):
   - Queue management
   - Retry logic
   - Stats tracking

**Example Test Structure**:
```go
// internal/storage/progress_test.go
func TestProgressManager_UpdateProgress(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    pm := NewProgressManager(db)

    err := pm.UpdateProgress("manga1", "Test Manga", "ch1", 5, 10)
    require.NoError(t, err)

    entry, err := pm.GetProgress("manga1", "ch1")
    require.NoError(t, err)
    assert.Equal(t, 5, entry.CurrentPage)
    assert.Equal(t, 10, entry.TotalPages)
    assert.False(t, entry.IsCompleted)
}
```

**Estimated Effort**: 1-2 weeks for basic coverage
**Priority**: Start with storage and source layers

### 3. Theme Package Incomplete (LOW PRIORITY)

**Current**: Only `colors.go` exists
**Missing**: Shared style definitions (see Issue #1)

**Recommended Action**: Add `styles.go` as described above

---

## Positive Findings

### ✅ What's Working Well

1. **No SQL Injection**: All queries use parameterized statements or literal strings
2. **Proper Error Handling**: Consistent `if err != nil` patterns, wrapped errors with context
3. **Resource Cleanup**: All `defer Close()` patterns are correct
4. **No Legacy Syntax**: No `interface{}` usage (uses `map[string]interface{}` correctly)
5. **Build Quality**: Compiles cleanly with no warnings after mutex fix
6. **Schema Versioning**: Well-implemented migration system (v1→v2→v3)
7. **Concurrency Safety**: Proper mutex usage in DownloadStats (after fix)
8. **GraphQL Client**: Well-structured with proper error handling
9. **Transaction Support**: Good helper functions with automatic rollback
10. **No Dead Code**: All functions are used, no obvious redundancy

---

## Recommendations by Priority

### Immediate (This Week)
1. ✅ **DONE**: Fix mutex copying bug
2. Create shared `theme/styles.go` to eliminate duplication
3. Add basic tests for storage layer

### Short Term (Next 2 Weeks)
4. Add tests for source and updates layers
5. Document image rendering limitations in README
6. Implement GetExtensionSources if needed for UI

### Medium Term (Next Month)
7. Implement browse view for manga discovery
8. Improve PDF page count detection
9. Complete local file GetAllPages implementation

### Long Term (Future)
10. Investigate terminal image rendering solutions
11. Expand test coverage to 70%+
12. Performance profiling and optimization

---

## Metrics

**Total Go Files**: 45
**Total Lines of Code**: ~10,800
**Largest Files**:
- settings.go: 733 lines (10 render functions - reasonable)
- extensions.go: 663 lines
- history.go: 644 lines

**Code Duplication**:
- ~95 inline style definitions (could reduce by 60-70%)
- Minimal logic duplication

**Test Coverage**: 0%
**Technical Debt**: Low-Medium (mostly missing features, not bad code)

---

## Conclusion

The codebase is in **good shape** overall with solid architecture and clean implementation. The main gaps are:

1. Missing test coverage (biggest risk)
2. Style duplication (easy fix)
3. Placeholder features (expected in early development)

No critical bugs or security issues found beyond the mutex copying bug (now fixed).

**Recommendation**: Focus on tests before adding new features. The architecture is sound, but lack of tests makes refactoring risky.
