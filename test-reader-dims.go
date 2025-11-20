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
	width  int
	height int
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "q" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Detecting terminal size..."
	}

	// Create a bright test image
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			// Colorful gradient
			img.Set(x, y, color.RGBA{
				uint8((x * 255) / 200),
				uint8((y * 255) / 200),
				255,
				255,
			})
		}
	}

	// Calculate what Miryokusha reader would use
	availableHeight := m.height - 8 // Header/footer space
	cellWidth := m.width - 4
	cellHeight := availableHeight - 2

	result := fmt.Sprintf("Terminal Size: %d cols × %d rows\n", m.width, m.height)
	result += fmt.Sprintf("TERM: %s\n", os.Getenv("TERM"))
	result += fmt.Sprintf("TERM_PROGRAM: %s\n\n", os.Getenv("TERM_PROGRAM"))
	result += fmt.Sprintf("Image will render at: %d cols × %d rows\n\n", cellWidth, cellHeight)

	// Render the test image using same dimensions as reader
	opts := rasterm.KittyImgOpts{
		DstCols: uint32(cellWidth),
		DstRows: uint32(cellHeight),
		ImageId: 1,
	}

	var buf []byte
	writer := &bufferWriter{buf: &buf}
	if err := rasterm.KittyWriteImage(writer, img, opts); err != nil {
		result += fmt.Sprintf("Error: %v\n", err)
	} else {
		result += string(buf)
	}

	result += "\n\nPress 'q' to quit"
	return result
}

type bufferWriter struct {
	buf *[]byte
}

func (bw *bufferWriter) Write(p []byte) (n int, err error) {
	*bw.buf = append(*bw.buf, p...)
	return len(p), nil
}

func main() {
	p := tea.NewProgram(model{}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
