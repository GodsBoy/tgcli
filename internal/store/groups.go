package store

import "time"

// UpsertGroup inserts or updates a group record.
func (d *DB) UpsertGroup(g Group) error {
	_, err := d.sql.Exec(`
		INSERT INTO groups (chat_id, title, creator_id, created_ts, member_count, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(chat_id) DO UPDATE SET
			title        = excluded.title,
			creator_id   = excluded.creator_id,
			created_ts   = excluded.created_ts,
			member_count = excluded.member_count,
			updated_at   = excluded.updated_at
	`, g.ChatID, g.Title, g.CreatorID,
		g.CreatedTS.UTC().Format(time.RFC3339),
		g.MemberCount,
		time.Now().UTC().Format(time.RFC3339))
	return err
}

// ListGroups returns all groups ordered by title.
func (d *DB) ListGroups(limit int) ([]Group, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := d.sql.Query(`
		SELECT chat_id, title, creator_id, created_ts, member_count, updated_at
		FROM groups
		ORDER BY title
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []Group
	for rows.Next() {
		var g Group
		var createdTS, updatedAt string
		if err := rows.Scan(&g.ChatID, &g.Title, &g.CreatorID, &createdTS, &g.MemberCount, &updatedAt); err != nil {
			return nil, err
		}
		g.CreatedTS, _ = time.Parse(time.RFC3339, createdTS)
		g.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

// GetGroup returns a single group by chat ID.
func (d *DB) GetGroup(chatID int64) (*Group, error) {
	var g Group
	var createdTS, updatedAt string
	err := d.sql.QueryRow(`
		SELECT chat_id, title, creator_id, created_ts, member_count, updated_at
		FROM groups WHERE chat_id = ?
	`, chatID).Scan(&g.ChatID, &g.Title, &g.CreatorID, &createdTS, &g.MemberCount, &updatedAt)
	if err != nil {
		return nil, err
	}
	g.CreatedTS, _ = time.Parse(time.RFC3339, createdTS)
	g.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &g, nil
}

// ReplaceGroupParticipants replaces all participants for a group.
func (d *DB) ReplaceGroupParticipants(chatID int64, participants []GroupParticipant) error {
	tx, err := d.sql.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM group_participants WHERE group_chat_id = ?`, chatID); err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
		INSERT INTO group_participants (group_chat_id, user_id, role, updated_at)
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	for _, p := range participants {
		if _, err := stmt.Exec(chatID, p.UserID, p.Role, now); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetGroupParticipants returns all participants for a group.
func (d *DB) GetGroupParticipants(chatID int64) ([]GroupParticipant, error) {
	rows, err := d.sql.Query(`
		SELECT group_chat_id, user_id, role, updated_at
		FROM group_participants
		WHERE group_chat_id = ?
		ORDER BY role, user_id
	`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var parts []GroupParticipant
	for rows.Next() {
		var p GroupParticipant
		var updatedAt string
		if err := rows.Scan(&p.GroupChatID, &p.UserID, &p.Role, &updatedAt); err != nil {
			return nil, err
		}
		p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		parts = append(parts, p)
	}
	return parts, rows.Err()
}
