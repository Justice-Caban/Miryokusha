package kitty

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// CellSize represents the pixel dimensions of a terminal cell
type CellSize struct {
	Width  int // Width in pixels
	Height int // Height in pixels
}

// GetCellSize queries the terminal for the actual pixel dimensions of a character cell
func GetCellSize() (CellSize, error) {
	// Try TIOCGWINSZ first (works on Unix systems)
	if size, err := getCellSizeFromWinsize(); err == nil {
		return size, nil
	}

	// Fallback: try CSI 16t escape sequence
	if size, err := getCellSizeFromEscape(); err == nil {
		return size, nil
	}

	// Final fallback: reasonable defaults for modern terminals
	// Most terminals use fonts around 8-12px width, 16-24px height
	return CellSize{Width: 10, Height: 20}, nil
}

// getCellSizeFromWinsize uses TIOCGWINSZ syscall to get pixel dimensions
func getCellSizeFromWinsize() (CellSize, error) {
	// winsize struct from asm-generic/termios.h
	type winsize struct {
		Row    uint16
		Col    uint16
		Xpixel uint16
		Ypixel uint16
	}

	ws := &winsize{}

	// Get file descriptor for stdout
	fd := uintptr(syscall.Stdout)

	// Call TIOCGWINSZ ioctl
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		fd,
		syscall.TIOCGWINSZ,
		uintptr(unsafe.Pointer(ws)),
	)

	if errno != 0 {
		return CellSize{}, fmt.Errorf("TIOCGWINSZ failed: %v", errno)
	}

	// Check if we got valid pixel dimensions
	if ws.Xpixel == 0 || ws.Ypixel == 0 || ws.Row == 0 || ws.Col == 0 {
		return CellSize{}, fmt.Errorf("terminal doesn't report pixel dimensions")
	}

	// Calculate cell size
	cellWidth := int(ws.Xpixel) / int(ws.Col)
	cellHeight := int(ws.Ypixel) / int(ws.Row)

	return CellSize{Width: cellWidth, Height: cellHeight}, nil
}

// getCellSizeFromEscape uses CSI 16t escape sequence to query cell size
func getCellSizeFromEscape() (CellSize, error) {
	// This is more complex and requires:
	// 1. Setting terminal to raw mode
	// 2. Sending "\x1b[16t"
	// 3. Reading response in format "\x1b[6;<height>;<width>t"
	// 4. Parsing the response
	//
	// For now, return error to fall back to other methods
	// TODO: Implement if TIOCGWINSZ doesn't work on all systems
	return CellSize{}, fmt.Errorf("CSI 16t not yet implemented")
}

// PrintCellSizeDiagnostic prints diagnostic information about cell size detection
func PrintCellSizeDiagnostic() {
	fmt.Println("=== Cell Size Diagnostic ===")
	fmt.Printf("TERM: %s\n", os.Getenv("TERM"))
	fmt.Printf("TERM_PROGRAM: %s\n", os.Getenv("TERM_PROGRAM"))

	cellSize, err := GetCellSize()
	if err != nil {
		fmt.Printf("Error detecting cell size: %v\n", err)
		fmt.Println("Using fallback: 10x20 pixels per cell")
	} else {
		fmt.Printf("Detected cell size: %dx%d pixels\n", cellSize.Width, cellSize.Height)
	}

	fmt.Println()
}
