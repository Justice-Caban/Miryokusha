package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"github.com/Justice-Caban/Miryokusha/internal/tui/kitty"
)

func main() {
	fmt.Println("=== Ghostty Terminal Diagnostic ===\n")

	// Environment variables
	fmt.Printf("TERM: %s\n", os.Getenv("TERM"))
	fmt.Printf("TERM_PROGRAM: %s\n", os.Getenv("TERM_PROGRAM"))
	fmt.Printf("TERM_PROGRAM_VERSION: %s\n\n", os.Getenv("TERM_PROGRAM_VERSION"))

	// TIOCGWINSZ syscall
	type winsize struct {
		Row    uint16
		Col    uint16
		Xpixel uint16
		Ypixel uint16
	}

	ws := &winsize{}
	fd := uintptr(syscall.Stdout)
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		fd,
		syscall.TIOCGWINSZ,
		uintptr(unsafe.Pointer(ws)),
	)

	if errno != 0 {
		fmt.Printf("ERROR: TIOCGWINSZ failed: %v\n", errno)
	} else {
		fmt.Println("TIOCGWINSZ Results:")
		fmt.Printf("  Rows (cells): %d\n", ws.Row)
		fmt.Printf("  Cols (cells): %d\n", ws.Col)
		fmt.Printf("  Xpixel (total width in pixels): %d\n", ws.Xpixel)
		fmt.Printf("  Ypixel (total height in pixels): %d\n", ws.Ypixel)

		if ws.Xpixel > 0 && ws.Ypixel > 0 && ws.Row > 0 && ws.Col > 0 {
			cellW := int(ws.Xpixel) / int(ws.Col)
			cellH := int(ws.Ypixel) / int(ws.Row)
			fmt.Printf("\n  Calculated cell size: %dx%d pixels\n", cellW, cellH)
		} else {
			fmt.Println("\n  WARNING: Pixel dimensions are zero!")
			fmt.Println("  This terminal doesn't report pixel dimensions via TIOCGWINSZ")
		}
	}

	fmt.Println()

	// Our cell size detection
	kitty.PrintCellSizeDiagnostic()

	// Show what dimensions would be used for reader
	cols := int(ws.Col)
	rows := int(ws.Row)
	if cols > 0 && rows > 0 {
		availableHeight := rows - 8
		cellWidth := cols - 4
		cellHeight := availableHeight - 2

		fmt.Printf("Reader would use:\n")
		fmt.Printf("  Image size: %d cols × %d rows\n", cellWidth, cellHeight)

		cellSize, _ := kitty.GetCellSize()
		targetPx := cellWidth * cellSize.Width
		targetPy := cellHeight * cellSize.Height
		fmt.Printf("  Pixel dimensions: %d × %d pixels\n", targetPx, targetPy)
	}
}
