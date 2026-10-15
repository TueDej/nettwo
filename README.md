# nettwo

A terminal client for authenticating with the Sharif University network portal (`net2.sharif.edu`) and activating the internet gateway.

## Features

- Interactive TUI dashboard (default mode)
- Login, gateway activation, and cancellation from the TUI
- Encrypted account profiles using [age](https://age-encryption.org/)
- Scriptable login and account management commands
- Cross-platform builds for Linux and Windows

## Quick start

```bash
# Launch the TUI (default)
./nettwo

# Traditional CLI login flow
./nettwo login
```

## CLI commands

```bash
./nettwo login                        # Authenticate
./nettwo login --auto                 # Auto-connect to last used account
./nettwo login --dry-run              # Authenticate without activating gateway
./nettwo login --user USER --pass PASS
./nettwo account list                 # Manage saved accounts
./nettwo account add USER PASS
./nettwo account delete USER
./nettwo version                      # Build info
./nettwo --quiet login
./nettwo --verbose login
```

## Configuration

Stored in `~/.config/nettwo/` (Linux):

- `config.yaml` — connection settings
- `credentials.age` — encrypted account data
- `identity.age` — local age encryption identity

Environment variables:

| Variable | Default | Description |
| --- | --- | --- |
| `NETTWO_BASE_URL` | `https://net2.sharif.edu` | Portal base URL |
| `NETTWO_CONNECT_TIMEOUT` | `10` | TLS connection timeout (seconds) |
| `NETTWO_REQUEST_TIMEOUT` | `30` | Per-request timeout (seconds) |
| `NETTWO_MAX_RETRIES` | `3` | Max retries for failed requests |

TLS verification is enabled by default. Only set `insecure_skip_verify: true` in `config.yaml` if the portal certificates fail on your network — and prefer a trusted network when you do.

## Windows

Double-clicking `nettwo.exe` in File Explorer opens a terminal window with the TUI
running in it — the program re-opens itself in its own console, because the console
Explorer hands to a GUI launch cannot host a full-screen TUI. Launching it from
cmd, PowerShell, Windows Terminal or WSL keeps that terminal, as usual.

If the program ever fails to start, the error stays on screen (it does not vanish
with the window) and is written to `%APPDATA%\nettwo\launch.log` — send that file
along when reporting a launch problem.

## Build

Requires Go 1.26+.

```bash
go build -o nettwo .
go test ./...
go vet ./...
```
