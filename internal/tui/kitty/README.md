# Kitty Graphics Protocol Support

This package implements image rendering using the [Kitty graphics protocol](https://sw.kovidgoyal.net/kitty/graphics-protocol/), which allows displaying images directly in supported terminals.

## Supported Terminals

The Kitty graphics protocol is supported by:

- **Kitty** - Full support (the original implementation)
- **WezTerm** - Full support
- **Konsole** - Partial support (KDE terminal)
- **Terminology** - Partial support (Enlightenment terminal)

Other terminals may have varying levels of support or may not support the protocol at all.

## Features

- **Image Fetching**: Downloads images from URLs with built-in HTTP client
- **Caching**: Caches downloaded images to avoid redundant downloads
- **Resizing**: Automatically resizes images to fit terminal dimensions
- **Chunked Transmission**: Handles large images by splitting them into chunks
- **Fallback Support**: Provides text placeholders when images fail to load

## Usage

### Basic Example

```go
import "github.com/Justice-Caban/Miryokusha/internal/tui/kitty"

// Create a renderer
renderer := kitty.NewImageRenderer()

// Configure image options
opts := kitty.ImageOptions{
    Width:               10,  // Width in terminal columns
    Height:              10,  // Height in terminal rows
    PreserveAspectRatio: true,
    ImageID:             1,   // Unique ID for this image
}

// Render an image from URL
imageStr, err := renderer.RenderImageFromURL("https://example.com/cover.jpg", opts)
if err != nil {
    // Use placeholder on error
    imageStr = kitty.CreatePlaceholder(opts.Width, opts.Height, "[IMG]")
}

// Print to terminal (will display the image)
fmt.Print(imageStr)
```

### In the Library View

The library view in Miryokusha uses this package to display manga cover images:

- Press `i` to toggle image display on/off
- Configure in `config.yaml` with `show_thumbnails: true/false`
- Images are automatically cached for performance

### Clearing Images

```go
// Clear a specific image
fmt.Print(kitty.ClearImage(1))

// Clear all images
fmt.Print(kitty.ClearAllImages())
```

## Configuration

In your `config.yaml`:

```yaml
preferences:
  show_thumbnails: true  # Enable image display in library
```

## Protocol Details

The Kitty graphics protocol uses escape sequences to transmit images:

```
ESC _G <parameters> ; <base64 data> ESC \
```

Key parameters:
- `a=T` - Action: transmit and display
- `f=100` - Format: PNG
- `i=<id>` - Image ID (unique identifier)
- `c=<cols>` - Width in columns
- `r=<rows>` - Height in rows
- `m=1/0` - More data coming (1) or last chunk (0)

## Performance Considerations

- Images are cached in memory after first download
- Large images are automatically resized to terminal dimensions
- Images are transmitted in 4KB chunks to avoid terminal buffer limits
- Use `ClearImage()` to free memory when images are no longer needed

## Troubleshooting

**Images not displaying?**
- Ensure you're using a supported terminal (Kitty, WezTerm, etc.)
- Check that `show_thumbnails` is enabled in config
- Verify the terminal supports the graphics protocol: `echo -e "\e_Ga=q,i=1\e\\"`

**Images appear corrupted?**
- Try a different terminal emulator
- Check terminal color depth and graphics settings
- Ensure images are valid (PNG, JPEG, GIF)

**Performance issues?**
- Disable images with `i` key in library view
- Reduce cache size in configuration
- Consider using text-only mode for slow connections

## References

- [Kitty Graphics Protocol Documentation](https://sw.kovidgoyal.net/kitty/graphics-protocol/)
- [Kitty Terminal](https://sw.kovidgoyal.net/kitty/)
- [WezTerm](https://wezfurlong.org/wezterm/)
