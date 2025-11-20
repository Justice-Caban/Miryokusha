package main

import (
	"fmt"
	"image"
	"image/color"
	"os"

	"github.com/BourgeoisBear/rasterm"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	imageRendered bool
}

type imageRenderedMsg struct{}

func (m model) Init() tea.Cmd {
	// Render image as an initialization command
	return func() tea.Msg {
		// Create test image
		img := image.NewRGBA(image.Rect(0, 0, 200, 200))
		for y := 0; y < 200; y++ {
			for x := 0; x < 200; x++ {
				img.Set(x, y, color.RGBA{255, 0, 0, 255})
			}
		}

		// Write directly to stdout (bypassing Bubble Tea)
		opts := rasterm.KittyImgOpts{
			DstCols: 60,
			DstRows: 30,
			ImageId: 1,
		}
		rasterm.KittyWriteImage(os.Stdout, img, opts)
		fmt.Println("\n\nImage rendered directly to stdout")

		return imageRenderedMsg{}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case imageRenderedMsg:
		m.imageRendered = true
	case tea.KeyMsg:
		return m, tea.Quit
	}
	return m, nil
}

func (m model) View() string {
	if !m.imageRendered {
		return "Rendering image directly to terminal (bypassing Bubble Tea)...\n\nPress any key to exit"
	}
	return "Image rendered!\n\nPress any key to exit"
}

func main() {
	fmt.Println("Testing direct terminal output with Bubble Tea + Alt Screen")
	p := tea.NewProgram(model{}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
