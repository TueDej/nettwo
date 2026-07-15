# nettwo

CLI tool for authenticating to the Sharif University network portal (net2.sharif.edu).

## Features

- Login and activate internet gateway
- Store multiple accounts with encrypted credentials
- Display remaining volume for each account before login
- Skip TLS verification for unreliable certificates

## Dependencies

- Go 1.26+
- [age](https://github.com/FiloSottile/age) (credential encryption)
- [cobra](https://github.com/spf13/cobra) (CLI framework)
- [koanf](https://github.com/knadh/koanf) (config management)

## Build

```bash
go build -o nettwo .
```

## Usage

```bash
# Login (shows account selection with volumes)
./nettwo

# Same as above
./nettwo login

# Manage accounts
./nettwo account list
./nettwo account add <username> <password>
./nettwo account delete <username>

# Verbose mode
./nettwo login -v
```

## Config

Config is stored in `~/.config/nettwo/`. Credentials are encrypted using age.

Environment variables (prefixed with `NETTWO_`):
- `NETTWO_BASE_URL` - Portal URL (default: https://net2.sharif.edu)
- `NETTWO_CONNECT_TIMEOUT` - Connection timeout in seconds (default: 10)
- `NETTWO_REQUEST_TIMEOUT` - Request timeout in seconds (default: 30)
- `NETTWO_MAX_RETRIES` - Max retries for failed requests (default: 3)
