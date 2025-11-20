package reader

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/Justice-Caban/Miryokusha/internal/source"
	"github.com/Justice-Caban/Miryokusha/internal/storage"
	"github.com/Justice-Caban/Miryokusha/internal/tui/kitty"
	"golang.org/x/term"
)

// StandaloneReader runs outside of Bubble Tea for full terminal control
// This bypasses Bubble Tea's alt-screen limitations with Kitty protocol
type StandaloneReader struct {
	manga         *source.Manga
	chapter       *source.Chapter
	chapters      []*source.Chapter
	currentPage   int
	pages         []*source.Page
	chapterIndex  int
	width         int
	height        int
	sourceManager *source.SourceManager
	storage       *storage.Storage
	imageRenderer *kitty.ImageRenderer
	cellSize      kitty.CellSize
	tty           *os.File

	// Reading session tracking
	sessionStart time.Time
	pagesRead    int
}

// NewStandaloneReader creates a new standalone reader
func NewStandaloneReader(
	manga *source.Manga,
	chapter *source.Chapter,
	chapters []*source.Chapter,
	sourceManager *source.SourceManager,
	storage *storage.Storage,
) *StandaloneReader {
	// Find chapter index
	chapterIndex := 0
	for i, ch := range chapters {
		if ch.ID == chapter.ID {
			chapterIndex = i
			break
		}
	}

	// Get cell size
	cellSize, err := kitty.GetCellSize()
	if err != nil {
		cellSize = kitty.CellSize{Width: 10, Height: 20}
	}

	return &StandaloneReader{
		manga:         manga,
		chapter:       chapter,
		chapters:      chapters,
		chapterIndex:  chapterIndex,
		sourceManager: sourceManager,
		storage:       storage,
		imageRenderer: kitty.NewImageRenderer(),
		cellSize:      cellSize,
		sessionStart:  time.Now(),
	}
}

// Run starts the standalone reader
func (r *StandaloneReader) Run() error {
	// DEBUG: Write to a log file in current directory
	logPath := "./miryokusha-reader.log"
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err == nil {
		defer logFile.Close()
		fmt.Fprintf(logFile, "\n=== Reader.Run() started at %v ===\n", time.Now())
		fmt.Fprintf(logFile, "Manga: %s\n", r.manga.Title)
		fmt.Fprintf(logFile, "Chapter: %.1f - %s\n", r.chapter.ChapterNumber, r.chapter.Title)
	}

	// Open /dev/tty explicitly to ensure we have the controlling terminal
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		if logFile != nil {
			fmt.Fprintf(logFile, "ERROR: Failed to open /dev/tty: %v\n", err)
		}
		return fmt.Errorf("failed to open /dev/tty: %w", err)
	}
	defer tty.Close()

	if logFile != nil {
		fmt.Fprintf(logFile, "Successfully opened /dev/tty\n")
	}

	// Get terminal size from TTY
	width, height, err := term.GetSize(int(tty.Fd()))
	if err != nil {
		if logFile != nil {
			fmt.Fprintf(logFile, "ERROR: Failed to get terminal size: %v\n", err)
		}
		return fmt.Errorf("failed to get terminal size: %w", err)
	}
	r.width = width
	r.height = height

	if logFile != nil {
		fmt.Fprintf(logFile, "Terminal size: %dx%d\n", width, height)
	}

	// Load initial chapter
	if logFile != nil {
		fmt.Fprintf(logFile, "Loading chapter pages...\n")
	}

	if err := r.loadChapter(); err != nil {
		if logFile != nil {
			fmt.Fprintf(logFile, "ERROR: Failed to load chapter: %v\n", err)
		}
		// Show error to user before exiting
		fmt.Fprintf(tty, "\n\nError loading chapter: %v\n", err)
		fmt.Fprintf(tty, "Press Enter to exit...")
		tty.Read(make([]byte, 1))
		return err
	}

	if logFile != nil {
		fmt.Fprintf(logFile, "Loaded %d pages\n", len(r.pages))
	}

	// Set terminal to raw mode for key handling
	oldState, err := term.MakeRaw(int(tty.Fd()))
	if err != nil {
		if logFile != nil {
			fmt.Fprintf(logFile, "ERROR: Failed to set raw mode: %v\n", err)
		}
		return fmt.Errorf("failed to set raw mode: %w", err)
	}
	defer term.Restore(int(tty.Fd()), oldState)

	// IMPORTANT: Flush the TTY input buffer
	// The Bubble Tea TUI may have left escape sequences in the buffer
	// We need to drain them before we start reading keys
	if logFile != nil {
		fmt.Fprintf(logFile, "Flushing TTY input buffer...\n")
	}

	// Set non-blocking mode temporarily
	if err := syscall.SetNonblock(int(tty.Fd()), true); err != nil {
		if logFile != nil {
			fmt.Fprintf(logFile, "WARNING: Failed to set non-blocking mode: %v\n", err)
		}
	} else {
		// Drain all pending input
		drainBuf := make([]byte, 1024)
		totalDrained := 0
		maxIterations := 100 // Safety limit to prevent infinite loops

		for iteration := 0; iteration < maxIterations; iteration++ {
			n, err := tty.Read(drainBuf)

			if logFile != nil && iteration < 5 {
				fmt.Fprintf(logFile, "Drain iteration %d: n=%d, err=%v\n", iteration, n, err)
			}

			// In non-blocking mode, EAGAIN/EWOULDBLOCK means no more data
			if err != nil {
				if logFile != nil {
					fmt.Fprintf(logFile, "Drain stopped with error: %v (this is normal)\n", err)
				}
				break
			}

			if n == 0 {
				if logFile != nil {
					fmt.Fprintf(logFile, "Drain stopped: read 0 bytes\n")
				}
				break
			}

			totalDrained += n
			if logFile != nil && totalDrained < 100 {
				// Log first few bytes for debugging
				for i := 0; i < n && totalDrained <= 100; i++ {
					fmt.Fprintf(logFile, "Drained byte: %d\n", drainBuf[i])
				}
			}
		}

		if logFile != nil {
			fmt.Fprintf(logFile, "Drained %d bytes from TTY buffer\n", totalDrained)
		}

		// Set back to blocking mode
		if err := syscall.SetNonblock(int(tty.Fd()), false); err != nil {
			if logFile != nil {
				fmt.Fprintf(logFile, "WARNING: Failed to restore blocking mode: %v\n", err)
			}
		}
	}

	// Hide cursor
	fmt.Fprint(tty, "\x1b[?25l")
	defer fmt.Fprint(tty, "\x1b[?25h")

	if logFile != nil {
		fmt.Fprintf(logFile, "Entering event loop\n")
	}

	// Store tty for use in event loop
	r.tty = tty

	// Main loop
	return r.eventLoop()
}

// loadChapter loads pages for the current chapter
func (r *StandaloneReader) loadChapter() error {
	r.currentPage = 0
	r.sessionStart = time.Now()
	r.pagesRead = 0

	// Load pages from source
	pages, err := r.sourceManager.GetAllPages(r.chapter)
	if err != nil {
		return fmt.Errorf("failed to load pages: %w", err)
	}

	r.pages = pages
	return nil
}

// eventLoop handles keyboard input
func (r *StandaloneReader) eventLoop() error {
	// DEBUG: Open log file
	logFile, _ := os.OpenFile("./miryokusha-reader.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if logFile != nil {
		defer logFile.Close()
	}

	buf := make([]byte, 3)
	iterationCount := 0

	for {
		iterationCount++
		if logFile != nil && iterationCount <= 5 {
			fmt.Fprintf(logFile, "Event loop iteration %d\n", iterationCount)
		}

		// Render current page
		if logFile != nil && iterationCount <= 5 {
			fmt.Fprintf(logFile, "Calling render()...\n")
		}

		if err := r.render(); err != nil {
			if logFile != nil {
				fmt.Fprintf(logFile, "ERROR: render() failed: %v\n", err)
			}
			return err
		}

		if logFile != nil && iterationCount <= 5 {
			fmt.Fprintf(logFile, "render() completed successfully\n")
			fmt.Fprintf(logFile, "About to read from tty...\n")
		}

		// Read key from TTY
		n, err := r.tty.Read(buf)

		if logFile != nil && iterationCount <= 5 {
			fmt.Fprintf(logFile, "tty.Read returned: n=%d, err=%v, buf[0]=%d\n", n, err, buf[0])
		}

		if err != nil {
			if logFile != nil {
				fmt.Fprintf(logFile, "ERROR: tty.Read failed: %v\n", err)
			}
			return err
		}

		if n == 0 {
			if logFile != nil && iterationCount <= 5 {
				fmt.Fprintf(logFile, "Read 0 bytes, continuing...\n")
			}
			continue
		}

		if logFile != nil {
			fmt.Fprintf(logFile, "Processing key: buf[0]=%d (%c)\n", buf[0], buf[0])
		}

		// Handle key
		switch {
		case buf[0] == 'q' || buf[0] == 27: // q or Esc
			if logFile != nil {
				fmt.Fprintf(logFile, "User pressed q or Esc, exiting\n")
			}
			return nil

		case buf[0] == ' ', buf[0] == 'l', buf[0] == '\r': // Space, l, Enter - next page
			if r.currentPage < len(r.pages)-1 {
				r.currentPage++
				r.pagesRead++
			} else {
				// At end of chapter, go to next
				if r.chapterIndex < len(r.chapters)-1 {
					r.chapterIndex++
					r.chapter = r.chapters[r.chapterIndex]
					if err := r.loadChapter(); err != nil {
						return err
					}
				}
			}

		case buf[0] == 'h', buf[0] == 8: // h or Backspace - previous page
			if r.currentPage > 0 {
				r.currentPage--
			} else {
				// At beginning of chapter, go to previous
				if r.chapterIndex > 0 {
					r.chapterIndex--
					r.chapter = r.chapters[r.chapterIndex]
					if err := r.loadChapter(); err != nil {
						return err
					}
					r.currentPage = len(r.pages) - 1
				}
			}

		case buf[0] == 'n': // n - next chapter
			if r.chapterIndex < len(r.chapters)-1 {
				r.chapterIndex++
				r.chapter = r.chapters[r.chapterIndex]
				if err := r.loadChapter(); err != nil {
					return err
				}
			}

		case buf[0] == 'p': // p - previous chapter
			if r.chapterIndex > 0 {
				r.chapterIndex--
				r.chapter = r.chapters[r.chapterIndex]
				if err := r.loadChapter(); err != nil {
					return err
				}
			}
		}
	}
}

// render displays the current page
func (r *StandaloneReader) render() error {
	// Clear screen
	fmt.Fprint(r.tty, "\x1b[2J\x1b[H")

	if len(r.pages) == 0 {
		fmt.Fprintln(r.tty, "No pages available")
		fmt.Fprintln(r.tty, "Press 'q' to exit")
		return nil
	}

	// Calculate image dimensions
	cellWidth := r.width - 4
	cellHeight := r.height - 6 // Leave room for header/footer

	// Render header
	fmt.Fprintf(r.tty, "%s - Chapter %.1f", r.manga.Title, r.chapter.ChapterNumber)
	if r.chapter.Title != "" {
		fmt.Fprintf(r.tty, ": %s", r.chapter.Title)
	}
	fmt.Fprintf(r.tty, " [%d/%d]\n\n", r.currentPage+1, len(r.pages))

	// Render image
	page := r.pages[r.currentPage]
	imageOpts := kitty.ImageOptions{
		Width:               cellWidth,
		Height:              cellHeight,
		PreserveAspectRatio: true,
		ImageID:             uint32(r.currentPage + 1),
	}

	var imageStr string
	var err error

	if len(page.ImageData) > 0 {
		imageStr, err = r.imageRenderer.RenderImage(page.ImageData, imageOpts)
	} else if page.URL != "" {
		imageStr, err = r.imageRenderer.RenderImageFromURL(page.URL, imageOpts)
	} else {
		err = fmt.Errorf("no image data or URL available")
	}

	if err != nil {
		fmt.Fprintf(r.tty, "Failed to load image: %v\n", err)
	} else {
		fmt.Fprint(r.tty, imageStr)
	}

	// Render footer
	fmt.Fprintf(r.tty, "\n\n[Space/l] Next  [h/Backspace] Prev  [n] Next Chapter  [p] Prev Chapter  [q] Exit")

	return nil
}

// SaveProgress saves reading progress before exiting
func (r *StandaloneReader) SaveProgress() {
	if r.storage == nil {
		return
	}

	// Save progress using the actual storage API
	err := r.storage.Progress.UpdateProgress(
		r.manga.ID,
		r.manga.Title,
		r.chapter.ID,
		r.currentPage,
		len(r.pages),
	)
	if err != nil {
		// Silently fail - don't interrupt exit
		return
	}

	// TODO: Add reading session tracking when Sessions manager is implemented
}
