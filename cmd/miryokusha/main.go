package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/Justice-Caban/Miryokusha/internal/config"
	"github.com/Justice-Caban/Miryokusha/internal/reader"
	"github.com/Justice-Caban/Miryokusha/internal/source"
	"github.com/Justice-Caban/Miryokusha/internal/storage"
	"github.com/Justice-Caban/Miryokusha/internal/tui"
)

func main() {
	// Check if we're being invoked in reader mode
	if len(os.Args) > 1 && os.Args[1] == "reader" {
		runReaderMode()
		return
	}

	// Normal TUI mode
	runTUIMode()
}

// runTUIMode runs the main TUI application with alt-screen
func runTUIMode() {
	// Create the application model
	m := tui.NewAppModel()

	// Run the TUI program with alt-screen
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running Miryokusha: %v\n", err)
		os.Exit(1)
	}
}

// runReaderMode runs the standalone reader without alt-screen
func runReaderMode() {
	// DEBUG: Log to file in current directory (guaranteed writable)
	logPath := "./miryokusha-reader.log"
	logFile, logErr := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if logFile != nil {
		defer logFile.Close()
		fmt.Fprintf(logFile, "\n=== runReaderMode called ===\n")
		fmt.Fprintf(logFile, "Args: %v\n", os.Args)
		fmt.Fprintf(logFile, "Log path: %s\n", logPath)
	} else if logErr != nil {
		// If we can't even create the log file, write to stderr
		fmt.Fprintf(os.Stderr, "WARNING: Could not create log file: %v\n", logErr)
	}

	// Parse reader flags - now accepting IDs instead of full JSON data
	readerFlags := flag.NewFlagSet("reader", flag.ExitOnError)
	mangaID := readerFlags.String("manga-id", "", "Manga ID")
	chapterID := readerFlags.String("chapter-id", "", "Chapter ID")

	if err := readerFlags.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing reader flags: %v\n", err)
		if logFile != nil {
			fmt.Fprintf(logFile, "ERROR parsing flags: %v\n", err)
		}
		os.Exit(1)
	}

	if logFile != nil {
		fmt.Fprintf(logFile, "Manga ID: %s\n", *mangaID)
		fmt.Fprintf(logFile, "Chapter ID: %s\n", *chapterID)
	}

	if *mangaID == "" || *chapterID == "" {
		errMsg := "Error: manga-id and chapter-id are required"
		fmt.Fprintf(os.Stderr, "%s\n", errMsg)
		if logFile != nil {
			fmt.Fprintf(logFile, "ERROR: %s\n", errMsg)
		}
		os.Exit(1)
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		if logFile != nil {
			fmt.Fprintf(logFile, "Config load failed, using defaults: %v\n", err)
		}
		cfg = config.DefaultConfig()
	} else if logFile != nil {
		fmt.Fprintf(logFile, "Config loaded successfully\n")
	}

	// Initialize storage
	st, err := storage.NewStorage(cfg.Paths.Database)
	if err != nil {
		if logFile != nil {
			fmt.Fprintf(logFile, "Storage init failed, continuing without: %v\n", err)
		}
		// Continue without storage
		st = nil
	} else if logFile != nil {
		fmt.Fprintf(logFile, "Storage initialized\n")
	}

	// Initialize source manager
	sourceManager := source.NewSourceManager()

	// Add Suwayomi source if configured
	if len(cfg.Servers) > 0 {
		for i, serverCfg := range cfg.Servers {
			suwayomiSource := source.NewSuwayomiSource(
				fmt.Sprintf("server-%d", i),
				serverCfg.Name,
				serverCfg.URL,
			)
			sourceManager.AddSource(suwayomiSource)
			if logFile != nil {
				fmt.Fprintf(logFile, "Added Suwayomi source: %s (%s)\n", serverCfg.Name, serverCfg.URL)
			}
		}
	}

	// Fetch manga data from source
	if logFile != nil {
		fmt.Fprintf(logFile, "Fetching manga data for ID: %s\n", *mangaID)
	}

	var manga *source.Manga
	var fetchErr error
	for _, src := range sourceManager.GetSources() {
		manga, fetchErr = src.GetManga(*mangaID)
		if fetchErr == nil {
			if logFile != nil {
				fmt.Fprintf(logFile, "Successfully fetched manga: %s\n", manga.Title)
			}
			break
		}
	}
	if manga == nil {
		errMsg := fmt.Sprintf("Error: could not fetch manga with ID %s: %v", *mangaID, fetchErr)
		fmt.Fprintf(os.Stderr, "%s\n", errMsg)
		if logFile != nil {
			fmt.Fprintf(logFile, "ERROR: %s\n", errMsg)
		}
		os.Exit(1)
	}

	// Fetch all chapters for navigation
	if logFile != nil {
		fmt.Fprintf(logFile, "Fetching all chapters for manga: %s\n", manga.Title)
	}

	var chapters []*source.Chapter
	for _, src := range sourceManager.GetSources() {
		chapters, fetchErr = src.ListChapters(*mangaID)
		if fetchErr == nil {
			if logFile != nil {
				fmt.Fprintf(logFile, "Successfully fetched %d chapters\n", len(chapters))
			}
			break
		}
	}
	if chapters == nil || len(chapters) == 0 {
		errMsg := fmt.Sprintf("Error: could not fetch chapters for manga %s: %v", *mangaID, fetchErr)
		fmt.Fprintf(os.Stderr, "%s\n", errMsg)
		if logFile != nil {
			fmt.Fprintf(logFile, "ERROR: %s\n", errMsg)
		}
		os.Exit(1)
	}

	// Find the requested chapter
	var chapter *source.Chapter
	for _, ch := range chapters {
		if ch.ID == *chapterID {
			chapter = ch
			if logFile != nil {
				fmt.Fprintf(logFile, "Found requested chapter: %.1f - %s\n", ch.ChapterNumber, ch.Title)
			}
			break
		}
	}
	if chapter == nil {
		errMsg := fmt.Sprintf("Error: could not find chapter with ID %s", *chapterID)
		fmt.Fprintf(os.Stderr, "%s\n", errMsg)
		if logFile != nil {
			fmt.Fprintf(logFile, "ERROR: %s\n", errMsg)
		}
		os.Exit(1)
	}

	if logFile != nil {
		fmt.Fprintf(logFile, "Creating standalone reader...\n")
	}

	// Create standalone reader
	r := reader.NewStandaloneReader(manga, chapter, chapters, sourceManager, st)

	if logFile != nil {
		fmt.Fprintf(logFile, "Starting reader...\n")
	}

	// Run reader
	if err := r.Run(); err != nil {
		errMsg := fmt.Sprintf("Error running reader: %v", err)
		fmt.Fprintf(os.Stderr, "%s\n", errMsg)
		if logFile != nil {
			fmt.Fprintf(logFile, "ERROR: %s\n", errMsg)
		}
		os.Exit(1)
	}

	if logFile != nil {
		fmt.Fprintf(logFile, "Reader exited normally\n")
	}

	// Save progress before exiting
	r.SaveProgress()
}

