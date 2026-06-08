# SpendSense CLI

A Go command-line client for the SpendSense personal finance tracker. Wraps the full REST API so you can manage expenses, incomes, wallets, and categories from any terminal.

> Full project context: [../README.md](../README.md)  
> Backend API reference: [../backend/README.md](../backend/README.md)

---

## Table of Contents

- [Architecture](#architecture)
- [Project Structure](#project-structure)
- [Developer Setup](#developer-setup)
- [Command Reference](#command-reference)
  - [auth](#auth)
  - [expense](#expense)
  - [income](#income)
  - [wallet](#wallet)
  - [category](#category)
  - [config](#config)
- [Configuration](#configuration)
- [Installing a Release Binary](#installing-a-release-binary)
- [Building a Release](#building-a-release)

---

## Architecture

The CLI is a thin HTTP client. Every subcommand translates user flags into a JSON request, calls the SpendSense REST API, and pretty-prints the response.

```
User
  └─► Cobra command (cmd/)
        └─► resources.go / client.go  (typed API calls)
              └─► net/http  (Authorization: Bearer <token>)
                    └─► SpendSense Go API  :8080
```

Authentication tokens are stored locally in `~/.expenserc` (managed by Viper). The token is loaded once at startup and injected into every request header.

---

## Project Structure

```
cli/
├── main.go            # Entry point — calls cmd.Execute()
├── go.mod
├── go.sum
├── .goreleaser.yml    # GoReleaser config for multi-platform release builds
└── cmd/
    ├── root.go        # Root cobra command, global flags (--api-url), config init
    ├── auth.go        # auth subcommands: register, login, me, refresh, logout, logout-all
    ├── resources.go   # expense, income, wallet, category subcommands + flag definitions
    └── client.go      # HTTP helpers: doRequest, parseResponse, token management
```

### Key design points

- **Cobra + Viper** — command structure and config file management.
- **Config file** — `~/.expenserc` stores `api_url`, `access_token`, and `refresh_token`.
- **Global `--api-url` flag** — overrides the config value for a single invocation.
- **`SPENDSENSE_API_URL` env var** — also accepted by Viper; env > config file > default.
- **Single HTTP client** — `cmd/client.go` provides `doRequest` which handles JSON encoding, auth header injection, and error unmarshalling.

---

## Developer Setup

### Prerequisites

- Go 1.21+
- A running SpendSense backend (see [../backend/README.md](../backend/README.md))

### Run directly

```bash
cd cli
go run . --help
```

### Build locally

```bash
# From repository root
make cli-build
# Binary: ./bin/spendsense

# Or directly
cd cli
go build -o ../bin/spendsense .
```

### Run built binary

```bash
./bin/spendsense auth login
./bin/spendsense expense list
```

### Point at a non-default API

```bash
./bin/spendsense --api-url http://localhost:9090 expense list
# or persistently:
./bin/spendsense config set api_url http://localhost:9090
```

---

## Command Reference

All commands follow the pattern:

```
spendsense <resource> <action> [flags]
```

Protected commands require a valid session (run `spendsense auth login` first).

---

### auth

| Command | Description |
|---------|-------------|
| `auth register` | Create a new account (prompts for email & password) |
| `auth login` | Login and store tokens locally |
| `auth me` | Print the current user's profile |
| `auth refresh` | Exchange the stored refresh token for a new access token |
| `auth logout` | Revoke the current session's refresh token |
| `auth logout-all` | Revoke **all** sessions for the current user |

**Examples:**

```bash
spendsense auth register
spendsense auth login
spendsense auth me
spendsense auth logout
```

---

### expense

| Command | Description |
|---------|-------------|
| `expense add` | Create a new expense |
| `expense list` | List expenses (paginated, supports date filters) |
| `expense delete <id>` | Soft-delete an expense by ID |

**`expense add` flags:**

| Flag | Required | Description |
|------|----------|-------------|
| `--amount` | Yes | Amount (e.g. `12.50`) |
| `--currency` | No | Currency code (default: `USD`) |
| `--category` | Yes | Category name or ID |
| `--wallet` | No | Wallet name or ID (defaults to first wallet) |
| `--date` | No | Date in `YYYY-MM-DD` or `today` (default: today) |
| `--merchant` | No | Merchant / description |
| `--notes` | No | Additional notes |

**`expense list` flags:**

| Flag | Description |
|------|-------------|
| `--from` | Start date filter (`YYYY-MM-DD`) |
| `--to` | End date filter (`YYYY-MM-DD`) |
| `--limit` | Max results (default: 20) |

**Examples:**

```bash
spendsense expense add \
  --amount 50 \
  --category Food \
  --date today \
  --merchant "Corner Cafe"

spendsense expense list
spendsense expense list --from 2026-06-01 --to 2026-06-30
spendsense expense delete 3f2a1b4c-...
```

---

### income

| Command | Description |
|---------|-------------|
| `income add` | Record a new income entry |
| `income list` | List income entries |
| `income delete <id>` | Soft-delete an income entry by ID |

**`income add` flags:**

| Flag | Required | Description |
|------|----------|-------------|
| `--amount` | Yes | Amount |
| `--currency` | No | Currency code (default: `USD`) |
| `--source` | Yes | Source name (e.g. `Salary Paycheck`) |
| `--category` | No | Category name or ID |
| `--wallet` | No | Wallet name or ID |
| `--date` | No | Date (`YYYY-MM-DD` or `today`) |
| `--notes` | No | Additional notes |

**Examples:**

```bash
spendsense income add \
  --amount 1500 \
  --wallet "Cash Wallet" \
  --category Salary \
  --source "May Paycheck"

spendsense income list
spendsense income delete 7a9c3d1e-...
```

---

### wallet

| Command | Description |
|---------|-------------|
| `wallet list` | List all wallets with balances |

**Examples:**

```bash
spendsense wallet list
```

---

### category

| Command | Description |
|---------|-------------|
| `category list` | List all categories (default + user-defined) |

**Flags:**

| Flag | Description |
|------|-------------|
| `--kind` | Filter by `EXPENSE` or `INCOME` (default: `EXPENSE`) |

**Examples:**

```bash
spendsense category list
spendsense category list --kind INCOME
```

---

### config

| Command | Description |
|---------|-------------|
| `config view` | Print the full contents of `~/.expenserc` |
| `config set <key> <value>` | Set a config value |
| `config get <key>` | Get a single config value |

**Common keys:**

| Key | Description |
|-----|-------------|
| `api_url` | SpendSense API base URL |

**Examples:**

```bash
spendsense config view
spendsense config set api_url http://localhost:8080
spendsense config get api_url
```

---

## Configuration

The CLI stores its state in `~/.expenserc` (YAML, managed by Viper).

```yaml
api_url: http://localhost:8080
access_token: <jwt>
refresh_token: <opaque-token>
```

**Precedence (highest → lowest):**

1. `--api-url` command-line flag
2. `SPENDSENSE_API_URL` environment variable
3. `api_url` in `~/.expenserc`
4. Default: `http://localhost:8080`

> ⚠️ `~/.expenserc` contains your auth tokens. Do not share or commit this file.

---

## Installing a Release Binary

Pre-built binaries are published via [GoReleaser](https://goreleaser.com) to GitHub Releases.

### Download

| Platform | Architecture | File |
|----------|-------------|------|
| macOS (Intel) | amd64 | `spendsense-cli_<ver>_darwin_amd64.tar.gz` |
| macOS (Apple Silicon) | arm64 | `spendsense-cli_<ver>_darwin_arm64.tar.gz` |
| Linux (x86_64) | amd64 | `spendsense-cli_<ver>_linux_amd64.tar.gz` |
| Linux (ARM64) | arm64 | `spendsense-cli_<ver>_linux_arm64.tar.gz` |
| Windows (64-bit) | amd64 | `spendsense-cli_<ver>_windows_amd64.zip` |

### Linux

```bash
tar -xzf spendsense-cli_*_linux_amd64.tar.gz
chmod +x spendsense
sudo mv spendsense /usr/local/bin/
spendsense --help
```

### macOS

```bash
tar -xzf spendsense-cli_*_darwin_arm64.tar.gz   # or _amd64 for Intel
chmod +x spendsense
sudo mv spendsense /usr/local/bin/

# If macOS blocks the unsigned binary:
xattr -d com.apple.quarantine spendsense
```

### Windows

Extract `spendsense-cli_*_windows_amd64.zip`, move `spendsense.exe` to a directory on your `PATH`, then verify:

```powershell
spendsense --help
```

---

## Building a Release

Releases are built with [GoReleaser](https://goreleaser.com) using the config in `.goreleaser.yml`.

```bash
# Dry-run (no publish)
goreleaser release --snapshot --clean

# Full release (requires GITHUB_TOKEN)
goreleaser release --clean
```
