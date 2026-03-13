package store

import "time"

// UpsertChat inserts or updates a chat record.
// Empty kind/name fields are preserved (not overwritten) on conflict.
func (d *DB) UpsertChat(c Chat) error {
	_, err := d.sql.Exec(`
		INSERT INTO chats (chat_id, kind, name, last_message_ts)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(chat_id) DO UPDATE SET
			kind            = CASE WHEN excluded.kind = '' THEN chats.kind ELSE excluded.kind END,
			name            = CASE WHEN excluded.name = '' THEN chats.name ELSE excluded.name END,
			last_message_ts = CASE
				WHEN excluded.last_message_ts > chats.last_message_ts THEN excluded.last_message_ts
				ELSE chats.last_message_ts
			END
	`, c.ChatID, c.Kind, c.Name, c.LastMessageTS.UTC().Format(time.RFC3339))
	return err
}

// ListChats returns all chats ordered by last message timestamp descending.
func (d *DB) ListChats(limit int) ([]Chat, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := d.sql.Query(`
		SELECT chat_id, kind, name, last_message_ts
		FROM chats
		ORDER BY last_message_ts DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []Chat
	for rows.Next() {
		var c Chat
		var ts string
		if err := rows.Scan(&c.ChatID, &c.Kind, &c.Name, &ts); err != nil {
			return nil, err
		}
		c.LastMessageTS, _ = time.Parse(time.RFC3339, ts)
		chats = append(chats, c)
	}
	return chats, rows.Err()
}

// GetChat returns a single chat by ID.
func (d *DB) GetChat(chatID int64) (*Chat, error) {
	var c Chat
	var ts string
	err := d.sql.QueryRow(`
		SELECT chat_id, kind, name, last_message_ts
		FROM chats WHERE chat_id = ?
	`, chatID).Scan(&c.ChatID, &c.Kind, &c.Name, &ts)
	if err != nil {
		return nil, err
	}
	c.LastMessageTS, _ = time.Parse(time.RFC3339, ts)
	return &c, nil
}
