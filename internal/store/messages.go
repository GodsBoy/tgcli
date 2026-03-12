package store

import (
	"fmt"
	"time"
)

// UpsertMessage inserts or updates a message. Idempotent via UNIQUE(chat_id, msg_id).
func (d *DB) UpsertMessage(p UpsertMessageParams) error {
	_, err := d.sql.Exec(`
		INSERT INTO messages (chat_id, msg_id, sender_id, ts, from_me, text, media_type, media_caption, reply_to_msg_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(chat_id, msg_id) DO UPDATE SET
			sender_id      = excluded.sender_id,
			ts             = excluded.ts,
			from_me        = excluded.from_me,
			text           = excluded.text,
			media_type     = excluded.media_type,
			media_caption  = excluded.media_caption,
			reply_to_msg_id = excluded.reply_to_msg_id
	`,
		p.ChatID, p.MsgID, p.SenderID,
		p.Timestamp.UTC().Format(time.RFC3339),
		boolToInt(p.FromMe),
		p.Text, p.MediaType, p.MediaCaption, p.ReplyToMsgID,
	)
	return err
}

// GetMessage retrieves a single message by chat_id and msg_id.
func (d *DB) GetMessage(chatID int64, msgID int) (*Message, error) {
	row := d.sql.QueryRow(`
		SELECT m.chat_id, m.msg_id, m.sender_id, m.ts, m.from_me, m.text,
		       m.media_type, m.media_caption, m.reply_to_msg_id,
		       COALESCE(c.name, '') AS chat_name
		FROM messages m
		LEFT JOIN chats c ON c.chat_id = m.chat_id
		WHERE m.chat_id = ? AND m.msg_id = ?
	`, chatID, msgID)
	return scanMessage(row)
}

// ListMessages returns messages for a given chat, ordered by timestamp descending.
func (d *DB) ListMessages(p ListMessagesParams) ([]Message, error) {
	query := `
		SELECT m.chat_id, m.msg_id, m.sender_id, m.ts, m.from_me, m.text,
		       m.media_type, m.media_caption, m.reply_to_msg_id,
		       COALESCE(c.name, '') AS chat_name
		FROM messages m
		LEFT JOIN chats c ON c.chat_id = m.chat_id
		WHERE 1=1`
	args := []interface{}{}

	if p.ChatID != 0 {
		query += ` AND m.chat_id = ?`
		args = append(args, p.ChatID)
	}
	if !p.After.IsZero() {
		query += ` AND m.ts >= ?`
		args = append(args, p.After.UTC().Format(time.RFC3339))
	}
	if !p.Before.IsZero() {
		query += ` AND m.ts <= ?`
		args = append(args, p.Before.UTC().Format(time.RFC3339))
	}

	query += ` ORDER BY m.ts DESC`

	limit := p.Limit
	if limit <= 0 {
		limit = 50
	}
	query += fmt.Sprintf(` LIMIT %d`, limit)

	rows, err := d.sql.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanMessages(rows)
}

// SearchMessages performs full-text search if FTS5 is available, otherwise falls back to LIKE.
func (d *DB) SearchMessages(p SearchMessagesParams) ([]Message, error) {
	if d.ftsEnabled {
		return d.searchFTS(p)
	}
	return d.searchLIKE(p)
}

func (d *DB) searchFTS(p SearchMessagesParams) ([]Message, error) {
	query := `
		SELECT m.chat_id, m.msg_id, m.sender_id, m.ts, m.from_me, m.text,
		       m.media_type, m.media_caption, m.reply_to_msg_id,
		       COALESCE(c.name, '') AS chat_name,
		       snippet(messages_fts, 0, '>>>', '<<<', '...', 32) AS snippet
		FROM messages_fts f
		JOIN messages m ON m.rowid = f.rowid
		LEFT JOIN chats c ON c.chat_id = m.chat_id
		WHERE messages_fts MATCH ?`
	args := []interface{}{p.Query}

	if p.ChatID != 0 {
		query += ` AND m.chat_id = ?`
		args = append(args, p.ChatID)
	}

	query += ` ORDER BY bm25(messages_fts)`

	limit := p.Limit
	if limit <= 0 {
		limit = 50
	}
	query += fmt.Sprintf(` LIMIT %d`, limit)

	rows, err := d.sql.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var m Message
		var ts string
		var fromMe int
		if err := rows.Scan(
			&m.ChatID, &m.MsgID, &m.SenderID, &ts, &fromMe,
			&m.Text, &m.MediaType, &m.MediaCaption, &m.ReplyToMsgID,
			&m.ChatName, &m.Snippet,
		); err != nil {
			return nil, err
		}
		m.FromMe = fromMe != 0
		m.Timestamp, _ = time.Parse(time.RFC3339, ts)
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (d *DB) searchLIKE(p SearchMessagesParams) ([]Message, error) {
	query := `
		SELECT m.chat_id, m.msg_id, m.sender_id, m.ts, m.from_me, m.text,
		       m.media_type, m.media_caption, m.reply_to_msg_id,
		       COALESCE(c.name, '') AS chat_name
		FROM messages m
		LEFT JOIN chats c ON c.chat_id = m.chat_id
		WHERE (m.text LIKE ? OR m.media_caption LIKE ?)`
	pattern := "%" + p.Query + "%"
	args := []interface{}{pattern, pattern}

	if p.ChatID != 0 {
		query += ` AND m.chat_id = ?`
		args = append(args, p.ChatID)
	}

	query += ` ORDER BY m.ts DESC`

	limit := p.Limit
	if limit <= 0 {
		limit = 50
	}
	query += fmt.Sprintf(` LIMIT %d`, limit)

	rows, err := d.sql.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanMessages(rows)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanMessage(row scanner) (*Message, error) {
	var m Message
	var ts string
	var fromMe int
	if err := row.Scan(
		&m.ChatID, &m.MsgID, &m.SenderID, &ts, &fromMe,
		&m.Text, &m.MediaType, &m.MediaCaption, &m.ReplyToMsgID,
		&m.ChatName,
	); err != nil {
		return nil, err
	}
	m.FromMe = fromMe != 0
	m.Timestamp, _ = time.Parse(time.RFC3339, ts)
	return &m, nil
}

func scanMessages(rows interface {
	Next() bool
	Scan(...interface{}) error
	Err() error
}) ([]Message, error) {
	var msgs []Message
	for rows.Next() {
		var m Message
		var ts string
		var fromMe int
		if err := rows.Scan(
			&m.ChatID, &m.MsgID, &m.SenderID, &ts, &fromMe,
			&m.Text, &m.MediaType, &m.MediaCaption, &m.ReplyToMsgID,
			&m.ChatName,
		); err != nil {
			return nil, err
		}
		m.FromMe = fromMe != 0
		m.Timestamp, _ = time.Parse(time.RFC3339, ts)
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}
