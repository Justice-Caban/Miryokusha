// Package kitty provides image rendering support using the Kitty graphics protocol.
//
// The Kitty graphics protocol is a terminal graphics protocol that allows displaying
// images directly in the terminal. This package provides utilities for:
//   - Fetching images from URLs
//   - Resizing images to fit terminal dimensions
//   - Encoding images using the Kitty protocol
//   - Caching images to avoid redundant downloads
//
// Usage:
//   renderer := kitty.NewImageRenderer()
//   opts := kitty.DefaultImageOptions()
//   imageStr, err := renderer.RenderImageFromURL("https://example.com/image.jpg", opts)
//   if err != nil {
//       // Handle error or use placeholder
//       imageStr = kitty.CreatePlaceholder(opts.Width, opts.Height, "[IMG]")
//   }
//   fmt.Print(imageStr)
//
// The Kitty protocol uses escape sequences in the format:
//   ESC _G <key=value,...> ; <base64 data> ESC \
//
// For more information on the Kitty graphics protocol, see:
// https://sw.kovidgoyal.net/kitty/graphics-protocol/
package kitty

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// Protocol control characters
const (
	ESC = "\x1b"
	APC = "\x1b_"
	ST  = "\x1b\\"
)

// ImageRenderer handles Kitty protocol image rendering
type ImageRenderer struct {
	// Cache to avoid re-downloading images
	cache map[string][]byte
	// HTTP client with timeout
	httpClient *http.Client
}

// NewImageRenderer creates a new image renderer
func NewImageRenderer() *ImageRenderer {
	return &ImageRenderer{
		cache: make(map[string][]byte),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ImageOptions contains options for image rendering
type ImageOptions struct {
	Width      int    // Width in cells (terminal columns)
	Height     int    // Height in cells (terminal rows)
	PreserveAspectRatio bool
	ImageID    uint32 // Unique ID for this image
}

// DefaultImageOptions returns sensible defaults for image rendering
func DefaultImageOptions() ImageOptions {
	return ImageOptions{
		Width:      10,
		Height:     10,
		PreserveAspectRatio: true,
		ImageID:    1,
	}
}

// FetchImage downloads an image from a URL
func (ir *ImageRenderer) FetchImage(url string) ([]byte, error) {
	// Check cache first
	if data, ok := ir.cache[url]; ok {
		return data, nil
	}

	// Download image
	resp, err := ir.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch image: status %d", resp.StatusCode)
	}

	// Read image data
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	// Diagnostic: log content type and size
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "unknown"
	}
	_ = contentType // Available for debugging: shows image/jpeg, image/png, etc.

	// Cache the image
	ir.cache[url] = data

	return data, nil
}

// ResizeImage resizes an image to fit within the specified dimensions
func ResizeImage(imgData []byte, maxWidth, maxHeight int) ([]byte, error) {
	// Decode image
	img, format, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		// Provide diagnostic information
		header := ""
		if len(imgData) > 16 {
			header = fmt.Sprintf("%x", imgData[:16])
		} else if len(imgData) > 0 {
			header = fmt.Sprintf("%x", imgData)
		}
		return nil, fmt.Errorf("failed to decode image (first bytes: %s, data len: %d): %w", header, len(imgData), err)
	}

	// Log successful decode for debugging
	_ = format // format is detected (jpeg, png, gif, webp, etc.)

	// Calculate new dimensions while preserving aspect ratio
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Calculate scaling factor
	scaleX := float64(maxWidth) / float64(width)
	scaleY := float64(maxHeight) / float64(height)
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	newWidth := int(float64(width) * scale)
	newHeight := int(float64(height) * scale)

	// Resize image
	resized := imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)

	// Encode back to PNG
	var buf bytes.Buffer
	if err := imaging.Encode(&buf, resized, imaging.PNG); err != nil {
		return nil, fmt.Errorf("failed to encode resized image: %w", err)
	}

	return buf.Bytes(), nil
}

// RenderImage renders an image using the Kitty graphics protocol
func (ir *ImageRenderer) RenderImage(imgData []byte, opts ImageOptions) (string, error) {
	// DON'T pre-resize - let Kitty's protocol handle scaling
	// This preserves maximum quality and lets Kitty scale based on actual cell dimensions
	// The c and r parameters tell Kitty how many cells to fill, and it will scale accordingly

	// Just re-encode as PNG to ensure consistent format
	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		// Try to use original data if decode fails
		return ir.renderImageDirect(imgData, opts)
	}

	// Re-encode as PNG
	var buf bytes.Buffer
	if err := imaging.Encode(&buf, img, imaging.PNG); err != nil {
		// Fallback to original data
		return ir.renderImageDirect(imgData, opts)
	}

	return ir.renderImageDirect(buf.Bytes(), opts)
}

// renderImageDirect sends image data directly to Kitty without resizing
func (ir *ImageRenderer) renderImageDirect(imgData []byte, opts ImageOptions) (string, error) {
	// Encode image data in base64
	encoded := base64.StdEncoding.EncodeToString(imgData)

	// Build Kitty graphics protocol command
	// Format: ESC _G <key=value,...> ; <base64 data> ESC \
	var sb strings.Builder

	// Chunked transmission for large images
	const chunkSize = 4096
	chunks := splitIntoChunks(encoded, chunkSize)

	for i, chunk := range chunks {
		sb.WriteString(APC)
		sb.WriteString("G")

		// Parameters
		params := []string{
			fmt.Sprintf("i=%d", opts.ImageID), // Image ID
			"f=100",                             // Format: PNG
		}

		if i == 0 {
			// First chunk
			params = append(params, "a=T") // Action: transmit and display
			if opts.Width > 0 {
				params = append(params, fmt.Sprintf("c=%d", opts.Width))
			}
			if opts.Height > 0 {
				params = append(params, fmt.Sprintf("r=%d", opts.Height))
			}
		}

		// More chunks to come?
		if i < len(chunks)-1 {
			params = append(params, "m=1") // More data coming
		} else {
			params = append(params, "m=0") // Last chunk
		}

		sb.WriteString(strings.Join(params, ","))
		sb.WriteString(";")
		sb.WriteString(chunk)
		sb.WriteString(ST)
	}

	return sb.String(), nil
}

// RenderImageFromURL fetches an image from a URL and renders it
func (ir *ImageRenderer) RenderImageFromURL(url string, opts ImageOptions) (string, error) {
	// Fetch image
	imgData, err := ir.FetchImage(url)
	if err != nil {
		return "", err
	}

	// Render image
	return ir.RenderImage(imgData, opts)
}

// ClearImage clears an image with the given ID
func ClearImage(imageID uint32) string {
	return fmt.Sprintf("%sG%s%s", APC, fmt.Sprintf("a=d,d=I,i=%d", imageID), ST)
}

// ClearAllImages clears all images from the screen
func ClearAllImages() string {
	return fmt.Sprintf("%sG%s%s", APC, "a=d,d=A", ST)
}

// CreatePlaceholder creates a text placeholder for images that fail to load
func CreatePlaceholder(width, height int, text string) string {
	var sb strings.Builder

	// Create a simple box with text
	topBottom := "+" + strings.Repeat("-", width-2) + "+"
	sb.WriteString(topBottom + "\n")

	// Calculate vertical centering
	textLine := "|" + centerText(text, width-2) + "|"
	emptyLine := "|" + strings.Repeat(" ", width-2) + "|"

	topPadding := (height - 3) / 2
	bottomPadding := height - 3 - topPadding

	for i := 0; i < topPadding; i++ {
		sb.WriteString(emptyLine + "\n")
	}
	sb.WriteString(textLine + "\n")
	for i := 0; i < bottomPadding; i++ {
		sb.WriteString(emptyLine + "\n")
	}

	sb.WriteString(topBottom)

	return sb.String()
}

// Helper functions

func splitIntoChunks(s string, chunkSize int) []string {
	var chunks []string
	for i := 0; i < len(s); i += chunkSize {
		end := i + chunkSize
		if end > len(s) {
			end = len(s)
		}
		chunks = append(chunks, s[i:end])
	}
	return chunks
}

func centerText(text string, width int) string {
	if len(text) >= width {
		return text[:width]
	}
	padding := (width - len(text)) / 2
	return strings.Repeat(" ", padding) + text + strings.Repeat(" ", width-len(text)-padding)
}
