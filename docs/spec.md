# tgcli Design Specification

## Overview

tgcli is a command-line interface for Telegram that authenticates as a real user account via the MTProto protocol (using [gotd/td](https://github.com/gotd/td)), syncs messages to a local SQLite database with full-text search (FTS5), and provides commands for searching, listing, and sending messages.

## Goals

1. **Offline-first**: All data is stored locally in SQLite. Search and browsing work without network.
2. **Fast search**: FTS5 provides instant full-text search across all synced messages.
3. **Scriptable**: JSON output mode (`--json`) enables integration with other tools.
4. **Single-instance safety**: flock-based locking prevents concurrent database corruption.
5. **Idempotent sync**: Upsert semantics ensure re-syncing is safe and replay-proof.

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌──────────────┐
│  cmd/tgcli  │────▶│  internal/*  │────▶│   gotd/td    │
│  (Cobra CLI)│     │  (business   │     │  (MTProto)   │
│             │     │   logic)     │     │              │
└─────────────┘     └──────┬───────┘     └──────────────┘
                           │
                    ┌──────▼───────┐
                    │  SQLite DB   │
                    │  (FTS5)      │
                    └──────────────┘
```

### Package Responsibilities

| Package | Responsibility |
|---------|---------------|
| `cmd/tgcli` | CLI commands, flag parsing, output formatting |
| `internal/auth` | Phone + OTP + 2FA authentication flow |
| `internal/client` | gotd/td wrapper (connect, send, fetch) |
| `internal/sync` | Sync engine (bootstrap + follow modes) |
| `internal/store` | SQLite storage layer with FTS5 |
| `internal/format` | JSON/text output helpers |
| `internal/lock` | Single-instance flock |
| `internal/config` | Config file loading |

## Data Model

### SQLite Schema

```sql
-- Chats (DMs, groups, channels, supergroups)
CREATE TABLE chats (
    chat_id         INTEGER PRIMARY KEY,
    kind            TEXT NOT NULL DEFAULT 'dm',
    name            TEXT NOT NULL DEFAULT '',
    last_message_ts TEXT NOT NULL DEFAULT ''
);

-- Contacts
CREATE TABLE contacts (
    user_id    INTEGER PRIMARY KEY,
    first_name TEXT NOT NULL DEFAULT '',
    last_name  TEXT NOT NULL DEFAULT '',
    username   TEXT NOT NULL DEFAULT '',
    phone      TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL DEFAULT ''
);

-- Groups
CREATE TABLE groups (
    chat_id      INTEGER PRIMARY KEY,
    title        TEXT NOT NULL DEFAULT '',
    creator_id   INTEGER NOT NULL DEFAULT 0,
    created_ts   TEXT NOT NULL DEFAULT '',
    member_count INTEGER NOT NULL DEFAULT 0,
    updated_at   TEXT NOT NULL DEFAULT ''
);

-- Group participants
CREATE TABLE group_participants (
    group_chat_id INTEGER NOT NULL REFERENCES groups(chat_id),
    user_id       INTEGER NOT NULL,
    role          TEXT NOT NULL DEFAULT 'member',
    updated_at    TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (group_chat_id, user_id)
);

-- Messages
CREATE TABLE messages (
    rowid           INTEGER PRIMARY KEY AUTOINCREMENT,
    chat_id         INTEGER NOT NULL REFERENCES chats(chat_id),
    msg_id          INTEGER NOT NULL,
    sender_id       INTEGER NOT NULL DEFAULT 0,
    ts              TEXT NOT NULL DEFAULT '',
    from_me         INTEGER NOT NULL DEFAULT 0,
    text            TEXT NOT NULL DEFAULT '',
    media_type      TEXT NOT NULL DEFAULT '',
    media_caption   TEXT NOT NULL DEFAULT '',
    reply_to_msg_id INTEGER NOT NULL DEFAULT 0,
    UNIQUE(chat_id, msg_id)
);

-- Full-text search (FTS5)
CREATE VIRTUAL TABLE messages_fts USING fts5(
    text, media_caption,
    content=messages, content_rowid=rowid
);
```

### Idempotency

All inserts use `INSERT ... ON CONFLICT DO UPDATE` (upsert). This ensures:
- Re-syncing the same messages is safe
- No duplicate entries
- Latest data wins on conflict

### FTS5 Synchronization

Three triggers keep the FTS5 index in sync:
- `messages_ai`: After INSERT → add to FTS
- `messages_ad`: After DELETE → remove from FTS
- `messages_au`: After UPDATE → remove old, add new

## Authentication

1. User provides APP_ID + APP_HASH (from my.telegram.org)
2. Phone number provided via env var, config, or interactive prompt
3. OTP code entered interactively (from Telegram app or SMS)
4. Optional 2FA password entered interactively (masked input)
5. Session stored as JSON file for future connections

## Sync Modes

### Bootstrap (via `tgcli auth`)
- Authenticates and performs initial sync
- Fetches all dialogs and recent history
- Exits when idle

### One-shot (via `tgcli sync`)
- Requires prior authentication
- Fetches dialogs and recent history
- Exits when complete

### Follow (via `tgcli sync --follow`)
- Continuous sync mode
- Polls for new updates every 3 seconds
- Persists new messages as they arrive
- Runs until Ctrl+C

## Output Format

### Human (default)
- Tables via `tabwriter` for aligned columns
- Human output to stdout, progress/logs to stderr
- Truncated text for readability

### JSON (`--json`)
- All output wrapped in envelope: `{success, data, error}`
- Full data without truncation
- Errors include structured error message

## Security

- Session file stored with 0600 permissions
- Config file stored with 0600 permissions
- Store directory created with 0700 permissions
- No secrets in code, logs, or output
- 2FA password input masked via `golang.org/x/term`

## Configuration

### Environment Variables
| Variable | Description |
|----------|-------------|
| `TGCLI_APP_ID` | Telegram app ID |
| `TGCLI_APP_HASH` | Telegram app hash |
| `TGCLI_PHONE` | Phone number for auth |
| `TGCLI_STORE_DIR` | Override store directory |

### Config File (`~/.tgcli/config.json`)
```json
{
  "app_id": 12345,
  "app_hash": "abcdef1234567890",
  "phone": "+1234567890"
}
```

Priority: env vars > config file > defaults.

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/gotd/td` | Telegram MTProto client |
| `github.com/gotd/contrib` | Middleware (floodwait, ratelimit) |
| `github.com/mattn/go-sqlite3` | SQLite driver with FTS5 (CGO) |
| `github.com/spf13/cobra` | CLI framework |
| `golang.org/x/term` | Terminal input (password masking) |

## Build Requirements

- Go 1.21+
- CGO enabled (for SQLite)
- Build tag: `sqlite_fts5`
- GCC / build-essential
