# Configuration Guide

## Configuration File Location

Miryokusha looks for its configuration file at:

- **Linux/macOS**: `~/.config/miryokusha/config.yaml`
- **Windows**: `%APPDATA%\miryokusha\config.yaml`

The application will automatically create this directory if it doesn't exist.

## Setting Up Configuration

### 1. Copy the Example Configuration

```bash
# Linux/macOS
mkdir -p ~/.config/miryokusha
cp config.example.yaml ~/.config/miryokusha/config.yaml

# Windows (PowerShell)
New-Item -ItemType Directory -Force -Path "$env:APPDATA\miryokusha"
Copy-Item config.example.yaml "$env:APPDATA\miryokusha\config.yaml"
```

### 2. Edit the Configuration

Open `~/.config/miryokusha/config.yaml` in your favorite text editor and customize:

```yaml
servers:
  - name: "My Server"
    url: "http://your-server:4567"
    default: true
```

## Configuration Sections

### Servers

Define one or more Suwayomi servers:

```yaml
servers:
  - name: "Local Server"
    url: "http://localhost:4567"
    default: true
    auth:  # Optional
      type: "basic"
      username: "admin"
      password: "password"
```

**Auth types**: `basic`, `token`, or `none`

### Server Management

Let Miryokusha start/stop the Suwayomi server:

```yaml
server_management:
  enabled: true
  executable_path: "/path/to/Suwayomi-Server.jar"
  args: ["--server.port=4567"]
  work_dir: "/path/to/suwayomi"
  auto_start: true
```

### Preferences

User interface and behavior settings:

```yaml
preferences:
  theme: "dark"  # or "light"
  reading_mode: "single"  # "single", "double", or "webtoon"
  cache_size_mb: 500
  auto_mark_read: true
  show_thumbnails: true
```

### Paths

Override default data storage locations:

```yaml
paths:
  database: ""  # Empty = ~/.local/share/miryokusha/miryokusha.db
  cache: ""     # Empty = ~/.cache/miryokusha
  downloads: "" # Empty = ~/Downloads/Miryokusha
```

### Updates

Configure library update behavior:

```yaml
updates:
  smart_update: true  # Mihon-style smart updates
  min_interval_hours: 12
  update_only_ongoing: true
  auto_update_enabled: false
  auto_update_interval_hrs: 24
```

## Environment Variables

Override configuration with environment variables (prefix: `MIRYOKUSHA_`):

```bash
export MIRYOKUSHA_SERVERS_0_URL="http://localhost:4567"
export MIRYOKUSHA_PREFERENCES_THEME="light"
```

## Validation

The application validates your configuration on startup and will report errors like:

- Invalid URLs
- Missing required fields
- Invalid enum values (theme, reading_mode, etc.)

## Security

**Important**: The config file is automatically set to `0600` permissions (owner read/write only) because it can contain sensitive authentication data (passwords, tokens).

If you manually create or edit the config file, ensure it has restrictive permissions:

```bash
chmod 600 ~/.config/miryokusha/config.yaml
```

## Troubleshooting

### "Config file not found"

The application will create a default config automatically. If you want to use your own:

1. Ensure it's in the correct location (see above)
2. Check file permissions
3. Verify the filename is exactly `config.yaml`

### "Config validation failed"

Common issues:

- **Missing sections**: Ensure all sections (servers, preferences, paths, updates, server_management) are present
- **Invalid values**: Check enum values like `theme`, `reading_mode`
- **Invalid URLs**: Server URLs must start with `http://` or `https://`

The application will set defaults for missing fields as of recent versions, but it's best to use a complete config.

### "Cannot read config file"

- Check file permissions (`ls -l ~/.config/miryokusha/config.yaml`)
- Ensure the directory exists
- Verify YAML syntax (use a YAML validator)

## Default Values

If a field is empty or missing, these defaults are used:

| Field | Default |
|-------|---------|
| `theme` | `"dark"` |
| `reading_mode` | `"single"` |
| `cache_size_mb` | `500` |
| `auto_mark_read` | `true` |
| `show_thumbnails` | `true` |
| `smart_update` | `true` |
| `min_interval_hours` | `12` |
| `update_only_ongoing` | `true` |
| `auto_update_enabled` | `false` |
| `auto_update_interval_hrs` | `24` |

## Example: Complete Configuration

See `config.example.yaml` in the repository root for a complete, documented example configuration with all available options.
