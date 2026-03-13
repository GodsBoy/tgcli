# 📡 tgcli - Telegram CLI: sync, search, send.

Telegram CLI built on top of [gotd/td](https://github.com/gotd/td) (MTProto), focused on:

- **Local message sync** with continuous capture
- **Fast offline search** via SQLite FTS5
- **Sending messages** and files
- **Contact + group management**

This is a third-party tool that uses the Telegram MTProto protocol via `gotd/td` and is not affiliated with Telegram.

## Status

Core implementation is in place. See `docs/spec.md` for the full design notes.

## Install / Build

### Prerequisites

- Go 1.21+
- GCC / build-essential (CGO required for SQLite)
- App credentials from [my.telegram.org](https://my.telegram.org)

### Build locally

```bash
git clone https://github.com/GodsBoy/tgcli.git
cd tgcli
make build
```

Run:

```bash
./dist/tgcli --help
```

### Install to $GOPATH/bin

```bash
make install
```

## Quick start

Default store directory is `~/.tgcli` (override with `--store DIR`).

```bash
# 1) Get API credentials from https://my.telegram.org
#    Create config:
cat > ~/.tgcli/config.json << EOF
{
  "app_id": 12345,
  "app_hash": "your_api_hash_here",
  "phone": "+1234567890"
}
EOF

# 2) Authenticate (phone + OTP + optional 2FA)
tgcli auth

# 3) Sync message history
tgcli sync

# 4) Keep syncing new messages (Ctrl+C to stop)
tgcli sync --follow

# 5) Diagnostics
tgcli doctor

# Search messages (FTS5 full-text search)
tgcli messages search "meeting"

# List recent messages in a chat
tgcli messages list --chat 12345 --limit 20

# Send a message
tgcli send text --to 12345 --message "hello"

# Send a file
tgcli send file --to 12345 --file ./photo.jpg --caption "check this out"

# List chats, contacts, groups
tgcli chats list
tgcli contacts list
tgcli groups list
tgcli groups info --chat 12345
```

## Configuration

Set environment variables:

```bash
export TGCLI_APP_ID=12345
export TGCLI_APP_HASH=abcdef1234567890
export TGCLI_PHONE=+1234567890
```

Or create `~/.tgcli/config.json`:

```json
{
  "app_id": 12345,
  "app_hash": "abcdef1234567890",
  "phone": "+1234567890"
}
```

Config file values are used as defaults; environment variables take precedence.

## Commands

### Authentication

```bash
tgcli auth                    # Authenticate (phone + OTP + optional 2FA)
tgcli auth status             # Show authentication status
tgcli auth logout             # Invalidate session
```

### Sync

```bash
tgcli sync                    # Sync message history to SQLite
tgcli sync --follow           # Continuous sync (Ctrl+C to stop)
```

`tgcli sync` never prompts for authentication - it errors if not logged in. Use `tgcli auth` first.

### Messages

```bash
tgcli messages list --chat <id> [--limit N] [--after TIME] [--before TIME]
tgcli messages search "query" [--chat <id>] [--limit N]
tgcli messages show --chat <id> --id <msg_id>
```

Search uses SQLite FTS5 for fast full-text matching with result highlighting.

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
tgcli doctor                  # Check configuration, session, database, FTS5
```

## Global Flags

```
--store DIR          Storage directory (default: ~/.tgcli)
--json               Output as JSON (structured envelope)
--timeout DURATION   Operation timeout (default: 5m)
```

## JSON Output

All commands support `--json` for machine-readable output:

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

## Storage

Defaults to `~/.tgcli` (override with `--store DIR`).

```
~/.tgcli/
├── config.json     # App credentials + settings
├── session.json    # MTProto session data
├── tgcli.db        # SQLite database (messages, chats, FTS5)
└── LOCK            # Instance lock file
```

Store permissions are set to `0700` (directory) and `0600` (files) for security.

## High-level UX

- `tgcli auth`: interactive login (phone + OTP + 2FA), then ready to sync.
- `tgcli sync`: non-interactive sync (never prompts for auth; errors if not authenticated).
- Output is human-readable by default; pass `--json` for scripting.
- Progress is written to stderr; primary output goes to stdout.
- Single-instance safety: store locking prevents concurrent access.

## Architecture

```
cmd/tgcli/           Cobra CLI commands
internal/auth/       MTProto phone + OTP + 2FA auth flow
internal/client/     gotd/td wrapper (connect, auth, API calls)
internal/sync/       Message sync engine (bootstrap + follow)
internal/store/      SQLite storage layer with FTS5 full-text search
internal/format/     Output formatting (JSON, plain text)
internal/lock/       Single-instance safety (flock)
internal/config/     Configuration loading (file + env)
```

## Development

```bash
make build          # Build binary to dist/
make test           # Run tests with race detector
make vet            # Run go vet
make lint           # Run golangci-lint (requires golangci-lint)
make clean          # Remove build artifacts
```

## Prior Art / Credit

This project is inspired by the excellent [wacli](https://github.com/steipete/wacli) by Peter Steinberger.

## License

MIT - see [LICENSE](LICENSE).
