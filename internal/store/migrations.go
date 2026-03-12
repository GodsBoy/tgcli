package store

import (
	"database/sql"
	"fmt"
)

func (d *DB) ensureSchema() error {
	if _, err := d.sql.Exec(`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`); err != nil {
		return fmt.Errorf("create schema_version table: %w", err)
	}

	var ver int
	err := d.sql.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_version`).Scan(&ver)
	if err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	migrations := []func(*sql.DB) error{
		migrateV1,
		migrateV2FTS,
	}

	for i := ver; i < len(migrations); i++ {
		if err := migrations[i](d.sql); err != nil {
			return fmt.Errorf("migration v%d: %w", i+1, err)
		}
		if _, err := d.sql.Exec(`INSERT INTO schema_version (version) VALUES (?)`, i+1); err != nil {
			return fmt.Errorf("record migration v%d: %w", i+1, err)
		}
	}

	// Check if FTS5 is available by verifying the table exists.
	d.ftsEnabled = ftsTableExists(d.sql)

	return nil
}

func migrateV1(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS chats (
			chat_id    INTEGER PRIMARY KEY,
			kind       TEXT NOT NULL DEFAULT 'dm',
			name       TEXT NOT NULL DEFAULT '',
			last_message_ts TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS contacts (
			user_id    INTEGER PRIMARY KEY,
			first_name TEXT NOT NULL DEFAULT '',
			last_name  TEXT NOT NULL DEFAULT '',
			username   TEXT NOT NULL DEFAULT '',
			phone      TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS groups (
			chat_id      INTEGER PRIMARY KEY,
			title        TEXT NOT NULL DEFAULT '',
			creator_id   INTEGER NOT NULL DEFAULT 0,
			created_ts   TEXT NOT NULL DEFAULT '',
			member_count INTEGER NOT NULL DEFAULT 0,
			updated_at   TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS group_participants (
			group_chat_id INTEGER NOT NULL REFERENCES groups(chat_id),
			user_id       INTEGER NOT NULL,
			role          TEXT NOT NULL DEFAULT 'member',
			updated_at    TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (group_chat_id, user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS messages (
			rowid          INTEGER PRIMARY KEY AUTOINCREMENT,
			chat_id        INTEGER NOT NULL REFERENCES chats(chat_id),
			msg_id         INTEGER NOT NULL,
			sender_id      INTEGER NOT NULL DEFAULT 0,
			ts             TEXT NOT NULL DEFAULT '',
			from_me        INTEGER NOT NULL DEFAULT 0,
			text           TEXT NOT NULL DEFAULT '',
			media_type     TEXT NOT NULL DEFAULT '',
			media_caption  TEXT NOT NULL DEFAULT '',
			reply_to_msg_id INTEGER NOT NULL DEFAULT 0,
			UNIQUE(chat_id, msg_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_chat_ts ON messages(chat_id, ts)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_ts ON messages(ts)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("exec %q: %w", s[:40], err)
		}
	}
	return nil
}

func migrateV2FTS(db *sql.DB) error {
	// Create FTS5 virtual table. This will fail gracefully if FTS5 is not compiled in.
	ftsStmts := []string{
		`CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
			text,
			media_caption,
			content=messages,
			content_rowid=rowid
		)`,
		// Trigger: auto-update FTS on INSERT
		`CREATE TRIGGER IF NOT EXISTS messages_ai AFTER INSERT ON messages BEGIN
			INSERT INTO messages_fts(rowid, text, media_caption)
			VALUES (new.rowid, new.text, new.media_caption);
		END`,
		// Trigger: auto-update FTS on DELETE
		`CREATE TRIGGER IF NOT EXISTS messages_ad AFTER DELETE ON messages BEGIN
			INSERT INTO messages_fts(messages_fts, rowid, text, media_caption)
			VALUES ('delete', old.rowid, old.text, old.media_caption);
		END`,
		// Trigger: auto-update FTS on UPDATE
		`CREATE TRIGGER IF NOT EXISTS messages_au AFTER UPDATE ON messages BEGIN
			INSERT INTO messages_fts(messages_fts, rowid, text, media_caption)
			VALUES ('delete', old.rowid, old.text, old.media_caption);
			INSERT INTO messages_fts(rowid, text, media_caption)
			VALUES (new.rowid, new.text, new.media_caption);
		END`,
		// Populate FTS from existing messages.
		`INSERT OR IGNORE INTO messages_fts(rowid, text, media_caption)
			SELECT rowid, text, media_caption FROM messages`,
	}
	for _, s := range ftsStmts {
		if _, err := db.Exec(s); err != nil {
			// FTS5 not available — skip silently, search will fall back to LIKE.
			return nil
		}
	}
	return nil
}

func ftsTableExists(db *sql.DB) bool {
	var name string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='messages_fts'`).Scan(&name)
	return err == nil && name == "messages_fts"
}
