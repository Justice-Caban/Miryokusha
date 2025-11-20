package main

import (
	"encoding/json"
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
	// Parse reader flags
	readerFlags := flag.NewFlagSet("reader", flag.ExitOnError)
	mangaJSON := readerFlags.String("manga", "", "Manga data (JSON)")
	chapterJSON := readerFlags.String("chapter", "", "Chapter data (JSON)")
	chaptersJSON := readerFlags.String("chapters", "", "All chapters data (JSON)")

	if err := readerFlags.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing reader flags: %v\n", err)
		os.Exit(1)
	}

	// Decode manga
	var manga source.Manga
	if err := json.Unmarshal([]byte(*mangaJSON), &manga); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding manga: %v\n", err)
		os.Exit(1)
	}

	// Decode chapter
	var chapter source.Chapter
	if err := json.Unmarshal([]byte(*chapterJSON), &chapter); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding chapter: %v\n", err)
		os.Exit(1)
	}

	// Decode chapters
	var chapters []*source.Chapter
	if err := json.Unmarshal([]byte(*chaptersJSON), &chapters); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding chapters: %v\n", err)
		os.Exit(1)
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Initialize storage
	st, err := storage.NewStorage(cfg.Paths.Database)
	if err != nil {
		// Continue without storage
		st = nil
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
		}
	}

	// Create standalone reader
	r := reader.NewStandaloneReader(&manga, &chapter, chapters, sourceManager, st)

	// Run reader
	if err := r.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running reader: %v\n", err)
		os.Exit(1)
	}

	// Save progress before exiting
	r.SaveProgress()
}

