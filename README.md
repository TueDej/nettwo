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

TLS verification is disabled — the portal certificates are unreliable. Use a trusted network.

## Build

Requires Go 1.26+.

```bash
go build -o nettwo .
go test ./...
go vet ./...
```
