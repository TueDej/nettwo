# nettwo

A terminal client for the Sharif University network portal (`net2.sharif.edu`): log in, activate the internet gateway, and see the remaining volume across your accounts.

## Install

Grab a prebuilt binary from [Releases](https://github.com/TueDej/nettwo/releases) — Linux (amd64, arm64, 386) and Windows (amd64, arm64) — or build it from source (below).

## Usage

```bash
nettwo                 # TUI dashboard — the default; every key is shown on screen
nettwo login           # CLI login flow
nettwo -k              # auto-connect with the last used account
nettwo account list    # scriptable account management
nettwo version
```

The TUI lists its keybindings at the bottom of the window, and `nettwo --help` documents the CLI commands and flags.

On Windows, double-clicking `nettwo.exe` in File Explorer opens a terminal window and runs the TUI inside it; launching it from cmd, PowerShell or Windows Terminal keeps that terminal instead.

## Configuration

Settings live in `~/.config/nettwo/` on Linux and `%APPDATA%\nettwo\` on Windows:

- `config.yaml` — connection settings
- `credentials.age` — accounts, encrypted with [age](https://age-encryption.org/)
- `identity.age` — the local age identity used to decrypt them

| Variable | Default | Description |
| --- | --- | --- |
| `NETTWO_BASE_URL` | `https://net2.sharif.edu` | Portal base URL |
| `NETTWO_CONNECT_TIMEOUT` | `10` | TLS connection timeout (seconds) |
| `NETTWO_REQUEST_TIMEOUT` | `30` | Per-request timeout (seconds) |
| `NETTWO_MAX_RETRIES` | `3` | Retries for failed requests |

TLS verification is enabled by default. Set `insecure_skip_verify: true` in `config.yaml` only if the portal certificates fail on your network — and prefer a trusted network when you do.

## Build

Requires Go 1.25+.

```bash
go build -o nettwo .
go test ./...
```
