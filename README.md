# nettwo

`nettwo` is a terminal client for authenticating with the Sharif University network portal (`net2.sharif.edu`) and activating the internet gateway.

## Features

- Interactive TUI dashboard is the default mode.
- Login, gateway activation, and cancellation without leaving the TUI.
- Multiple encrypted account profiles using [age](https://age-encryption.org/).
- Remaining-volume display for saved accounts.
- Volume refresh with a visible status spinner.
- Scriptable login and account-management commands.
- Cross-platform release binaries for Linux and Windows.

## Quick start

```bash
# Launch the TUI (the default mode)
./nettwo

# Explicitly launch the TUI
./nettwo tui

# Authenticate through the traditional CLI flow
./nettwo login
```

The first TUI launch lets you add an account. Saved credentials are encrypted locally and reused for later logins.

## TUI controls

| Key | Action |
| --- | --- |
| `j` / `k` or arrow keys | Select an account |
| `Enter` | Authenticate and activate the selected account |
| `a` | Add an account |
| `d` | Delete the selected account |
| `r` | Refresh remaining volumes |
| `Esc` | Cancel login or return to the dashboard |
| `q` / `Ctrl+C` | Quit |

## CLI commands

```bash
# Authenticate interactively or use a saved account
./nettwo login

# Auto-connect to the last used account
./nettwo --auto

# Same via the login subcommand
./nettwo login --auto

# Authenticate without activating the gateway
./nettwo login --dry-run

# Authenticate non-interactively
./nettwo login --username USERNAME --password PASSWORD

# Manage encrypted account profiles
./nettwo account list
./nettwo account add USERNAME PASSWORD
./nettwo account delete USERNAME

# Show build information
./nettwo version
```

Global output flags are available on CLI commands:

```bash
./nettwo --quiet login
./nettwo --verbose login
```

The `--auto` (`-k`) flag reuses the last successfully connected account and
authenticates without any prompts. The last used username is stored in the
platform's user cache directory (e.g. `~/.cache/nettwo/last_username`).

## Configuration

Configuration and encrypted credentials are stored in the platform's user config directory, usually `~/.config/nettwo/` on Linux:

- `config.yaml` — connection settings
- `credentials.age` — encrypted account data
- `identity.age` — local age encryption identity
- `cookies.json` — saved portal cookies when available

Environment variables use the `NETTWO_` prefix:

| Variable | Default | Description |
| --- | --- | --- |
| `NETTWO_BASE_URL` | `https://net2.sharif.edu` | Portal base URL |
| `NETTWO_CONNECT_TIMEOUT` | `10` | TLS connection timeout in seconds |
| `NETTWO_REQUEST_TIMEOUT` | `30` | Per-request timeout in seconds |
| `NETTWO_MAX_RETRIES` | `3` | Maximum retries for failed requests |

Example:

```bash
NETTWO_REQUEST_TIMEOUT=15 ./nettwo
```

The client currently skips TLS certificate verification because the portal may expose unreliable certificates. Use a trusted network and understand the security implications before changing the portal URL.

## Build from source

Requirements:

- Go 1.26 or newer

```bash
go build -o nettwo .
go test ./...
go vet ./...
```

For a static Linux build suitable for minimal distributions such as Alpine:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath -ldflags="-s -w" -o nettwo-linux-amd64 .
```

## Releases

Pushing a version tag such as `v0.2.0` triggers [GitHub Actions](.github/workflows/release.yml), which builds static release binaries for:

- Linux amd64
- Linux arm64
- Windows amd64
- Windows arm64

The binaries are attached automatically to the generated GitHub Release and embed the release tag in `nettwo version`.

## Project structure

```text
cmd/                  Cobra commands and entry points
internal/auth/        Portal authentication and volume requests
internal/config/      Configuration and encrypted credential storage
internal/httpclient/  Cookie-aware HTTP client with retries
internal/tui/         Bubble Tea model, views, components, and styles
internal/ui/          Non-TUI terminal output and prompts
```
