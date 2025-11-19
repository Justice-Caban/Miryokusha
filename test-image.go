package main

import (
	"fmt"
	"image"
	"image/color"
	"os"

	"github.com/BourgeoisBear/rasterm"
)

func main() {
	// Create a simple test image - red square
	width, height := 100, 100
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with red
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}

	fmt.Println("Testing Kitty Graphics Protocol with Ghostty")
	fmt.Println("============================================")
	fmt.Printf("Terminal: %s\n", os.Getenv("TERM"))
	fmt.Printf("TERM_PROGRAM: %s\n\n", os.Getenv("TERM_PROGRAM"))

	// Test 1: Small size (10x10 cells)
	fmt.Println("Test 1: 10x10 cells")
	opts1 := rasterm.KittyImgOpts{
		DstCols: 10,
		DstRows: 10,
		ImageId: 1,
	}
	rasterm.KittyWriteImage(os.Stdout, img, opts1)
	fmt.Println("\n")

	// Test 2: Medium size (30x15 cells)
	fmt.Println("Test 2: 30x15 cells")
	opts2 := rasterm.KittyImgOpts{
		DstCols: 30,
		DstRows: 15,
		ImageId: 2,
	}
	rasterm.KittyWriteImage(os.Stdout, img, opts2)
	fmt.Println("\n")

	// Test 3: Large size (60x30 cells)
	fmt.Println("Test 3: 60x30 cells")
	opts3 := rasterm.KittyImgOpts{
		DstCols: 60,
		DstRows: 30,
		ImageId: 3,
	}
	rasterm.KittyWriteImage(os.Stdout, img, opts3)
	fmt.Println("\n")

	fmt.Println("If you see three red squares of increasing size, the protocol is working!")
	fmt.Println("If all squares are tiny or the same size, there's a Ghostty compatibility issue.")
}
