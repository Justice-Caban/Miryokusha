package main

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"strings"

	"github.com/BourgeoisBear/rasterm"
	"github.com/Justice-Caban/Miryokusha/internal/tui/kitty"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	imageStr string
	cellSize kitty.CellSize
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Render image when we know the size
		if m.imageStr == "" {
			// Create gradient test image
			img := image.NewRGBA(image.Rect(0, 0, 400, 400))
			for y := 0; y < 400; y++ {
				for x := 0; x < 400; x++ {
					img.Set(x, y, color.RGBA{
						uint8((x * 255) / 400),
						uint8((y * 255) / 400),
						255,
						255,
					})
				}
			}

			// Get cell size
			cellSize, _ := kitty.GetCellSize()
			m.cellSize = cellSize

			// Render at large size (60x30 cells)
			var buf strings.Builder
			opts := rasterm.KittyImgOpts{
				DstCols: 60,
				DstRows: 30,
				ImageId: 1,
			}
			rasterm.KittyWriteImage(&buf, img, opts)
			m.imageStr = buf.String()
		}
	case tea.KeyMsg:
		if msg.String() == "q" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.imageStr == "" {
		return "Waiting for window size..."
	}

	var b strings.Builder
	b.WriteString("=== Testing WITHOUT Alt Screen ===\n\n")
	b.WriteString(fmt.Sprintf("Cell Size: %dx%d pixels\n", m.cellSize.Width, m.cellSize.Height))
	b.WriteString(fmt.Sprintf("Image: 60 cols × 30 rows\n"))
	b.WriteString(fmt.Sprintf("Expected: %d × %d pixels\n\n", 60*m.cellSize.Width, 30*m.cellSize.Height))

	// Include the Kitty protocol escape sequences in the View string
	b.WriteString(m.imageStr)
	b.WriteString("\n\nPress 'q' to quit")

	return b.String()
}

func main() {
	// NO tea.WithAltScreen() - this is the key test
	p := tea.NewProgram(model{})
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
