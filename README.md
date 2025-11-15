# Miryokusha

A beautiful Terminal User Interface (TUI) client for Suwayomi/Tachiyomi servers, built with Go and the Charm framework.

## Features (Planned)

- 🖥️  **Beautiful TUI** - Powered by Charm's Bubble Tea framework
- 📚 **Suwayomi Integration** - Full support for Suwayomi/Tachiyomi servers
- 📂 **Local File Support** - Read manga from CBZ, CBR, PDF files
- 📖 **Multiple Reading Modes** - Single page, double page, and webtoon modes
- 📊 **Reading History** - Track your reading progress locally
- 🔖 **Bookmarks** - Save your favorite pages
- 📥 **Downloads** - Offline reading support
- 🏷️  **Categories** - Organize your manga library
- 🔄 **Tracking** - Sync with MyAnimeList, AniList, Kitsu
- 🧩 **Extensions** - Browse and install Suwayomi extensions

## Status

🚧 **Early Development** - This project is in active development. Core features are being implemented.

## Requirements

- Go 1.21 or higher
- Terminal with ANSI color support
- Suwayomi server (optional, for server features)

## Building

```bash
# Clone the repository
git clone https://github.com/Justice-Caban/Miryokusha.git
cd Miryokusha

# Build the application
go build -o bin/miryokusha ./cmd/miryokusha

# Run the application
./bin/miryokusha
```

## Development

```bash
# Run in development mode
go run ./cmd/miryokusha

# Run tests
go test ./...

# Build with optimizations
go build -ldflags="-s -w" -o bin/miryokusha ./cmd/miryokusha
```

## Configuration

Configuration file location: `~/.config/miryokusha/config.yaml`

See [CLAUDE.md](CLAUDE.md) for detailed configuration options and development guidelines.

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Suwayomi](https://github.com/Suwayomi/Suwayomi-Server) - The server backend
- [Charm](https://charm.sh/) - Beautiful TUI framework
- [Mihon](https://mihon.app/) - Inspiration for features

## Contributing

Contributions are welcome! Please read [CLAUDE.md](CLAUDE.md) for development guidelines.

---

**Note**: This project is not affiliated with or endorsed by Suwayomi, Tachiyomi, or Mihon.
