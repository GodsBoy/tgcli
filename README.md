# tgcli

A Telegram CLI built in Go using [gotd/td](https://github.com/gotd/td) (MTProto).

tgcli authenticates as a real user account (not Bot API), syncs messages to a local SQLite database with full-text search (FTS5), and provides commands for searching, listing, and sending messages.

## Prerequisites

- Go 1.21+
- GCC / build-essential (CGO required for SQLite)
- A Telegram account
- App credentials from [my.telegram.org](https://my.telegram.org)

## Install

```bash
# From source
git clone https://github.com/GodsBoy/tgcli.git
cd tgcli
make install

# Or directly
CGO_ENABLED=1 go install -tags sqlite_fts5 ./cmd/tgcli@latest
```

## Quick Start

### 1. Get API Credentials

Go to [my.telegram.org](https://my.telegram.org), log in, and create an application to get your `api_id` and `api_hash`.

### 2. Configure

Set environment variables:

```bash
export TGCLI_APP_ID=12345
export TGCLI_APP_HASH=abcdef1234567890
export TGCLI_PHONE=+1234567890
```

Or create a config file at `~/.tgcli/config.json`:

```json
{
  "app_id": 12345,
  "app_hash": "abcdef1234567890",
  "phone": "+1234567890"
}
```

### 3. Authenticate

```bash
tgcli auth
# Enter OTP code when prompted
# Enter 2FA password if enabled
```

### 4. Sync Messages

```bash
# One-time sync
tgcli sync

# Continuous sync (stays running)
tgcli sync --follow
```

### 5. Search & Browse

```bash
# Search messages
tgcli messages search "hello world"

# List recent messages in a chat
tgcli messages list --chat 12345

# Show a specific message
tgcli messages show --chat 12345 --id 42
```

## Commands

### Authentication

```bash
tgcli auth                    # Authenticate (phone + OTP + optional 2FA)
tgcli auth status             # Check authentication status
tgcli auth logout             # Invalidate session
```

### Sync

```bash
tgcli sync                    # Sync message history to SQLite
tgcli sync --follow           # Continuous sync (Ctrl+C to stop)
```

### Messages

```bash
tgcli messages list --chat <id> [--limit N] [--after TIME] [--before TIME]
tgcli messages search "query" [--chat <id>] [--limit N]
tgcli messages show --chat <id> --id <msg_id>
```

### Send

```bash
tgcli send text --to <id> --message "hello"
tgcli send file --to <id> --file ./photo.jpg [--caption "hi"]
```

### Chats, Contacts, Groups

```bash
tgcli chats list [--limit N]
tgcli contacts list [--limit N]
tgcli groups list [--limit N]
tgcli groups info --chat <id>
```

### Diagnostics

```bash
tgcli doctor                  # Check configuration and connectivity
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--store DIR` | Storage directory (default: `~/.tgcli`) |
| `--json` | Output as JSON (structured envelope) |
| `--timeout DURATION` | Operation timeout (default: 5m) |

## JSON Output

When `--json` is passed, all output is wrapped in an envelope:

```json
{
  "success": true,
  "data": { ... }
}
```

On error:

```json
{
  "success": false,
  "error": "error message"
}
```

## Storage Layout

```
~/.tgcli/
├── config.json     # App credentials + settings
├── session.json    # MTProto session data
├── tgcli.db        # SQLite database (messages, chats, FTS5)
└── LOCK            # Instance lock file
```

## Architecture

- `cmd/tgcli/` — Cobra CLI commands
- `internal/auth/` — MTProto phone + OTP + 2FA auth flow
- `internal/client/` — gotd/td wrapper
- `internal/sync/` — Message sync engine (bootstrap + follow)
- `internal/store/` — SQLite storage layer with FTS5
- `internal/format/` — Output formatting (JSON, plain text)
- `internal/lock/` — Single-instance safety (flock)
- `internal/config/` — Configuration loading

## Development

```bash
make build          # Build binary to dist/
make test           # Run tests with race detector
make vet            # Run go vet
make lint           # Run golangci-lint
make install        # Install to $GOPATH/bin
```

## License

MIT
