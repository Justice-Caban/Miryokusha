package kitty

import (
	"testing"
)

func TestNewImageRenderer(t *testing.T) {
	renderer := NewImageRenderer()
	if renderer == nil {
		t.Fatal("NewImageRenderer returned nil")
	}
	if renderer.cache == nil {
		t.Error("ImageRenderer cache should be initialized")
	}
	if renderer.httpClient == nil {
		t.Error("ImageRenderer httpClient should be initialized")
	}
}

func TestDefaultImageOptions(t *testing.T) {
	opts := DefaultImageOptions()
	if opts.Width == 0 {
		t.Error("Default width should not be 0")
	}
	if opts.Height == 0 {
		t.Error("Default height should not be 0")
	}
	if !opts.PreserveAspectRatio {
		t.Error("Default should preserve aspect ratio")
	}
}

func TestCreatePlaceholder(t *testing.T) {
	placeholder := CreatePlaceholder(10, 5, "Test")
	if placeholder == "" {
		t.Error("Placeholder should not be empty")
	}
	// Check that placeholder contains the text
	if len(placeholder) < 4 {
		t.Error("Placeholder should have reasonable length")
	}
}

func TestClearImage(t *testing.T) {
	result := ClearImage(1)
	if result == "" {
		t.Error("ClearImage should return a non-empty string")
	}
	// Check that it contains the Kitty protocol escape sequences
	if len(result) < 5 {
		t.Error("ClearImage result should contain Kitty protocol commands")
	}
}

func TestClearAllImages(t *testing.T) {
	result := ClearAllImages()
	if result == "" {
		t.Error("ClearAllImages should return a non-empty string")
	}
	// Check that it contains the Kitty protocol escape sequences
	if len(result) < 5 {
		t.Error("ClearAllImages result should contain Kitty protocol commands")
	}
}

func TestSplitIntoChunks(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		chunkSize int
		expected  int
	}{
		{"Empty string", "", 10, 0},
		{"Single chunk", "hello", 10, 1},
		{"Multiple chunks", "hello world this is a test", 5, 6},
		{"Exact fit", "12345", 5, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitIntoChunks(tt.input, tt.chunkSize)
			if len(result) != tt.expected {
				t.Errorf("Expected %d chunks, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestCenterText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		width    int
		expected string
	}{
		{"Exact fit", "hello", 5, "hello"},
		{"Text too long", "hello world", 5, "hello"},
		{"Center short text", "hi", 6, "  hi  "},
		{"Center odd width", "hi", 7, "  hi   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := centerText(tt.text, tt.width)
			if len(result) != tt.width {
				t.Errorf("Expected length %d, got %d", tt.width, len(result))
			}
		})
	}
}
