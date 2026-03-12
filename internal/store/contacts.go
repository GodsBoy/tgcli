package store

import "time"

// UpsertContact inserts or updates a contact record.
func (d *DB) UpsertContact(c Contact) error {
	_, err := d.sql.Exec(`
		INSERT INTO contacts (user_id, first_name, last_name, username, phone, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			first_name = excluded.first_name,
			last_name  = excluded.last_name,
			username   = excluded.username,
			phone      = excluded.phone,
			updated_at = excluded.updated_at
	`, c.UserID, c.FirstName, c.LastName, c.Username, c.Phone,
		time.Now().UTC().Format(time.RFC3339))
	return err
}

// ListContacts returns all contacts ordered by first name.
func (d *DB) ListContacts(limit int) ([]Contact, error) {
	if limit <= 0 {
		limit = 500
	}
	rows, err := d.sql.Query(`
		SELECT user_id, first_name, last_name, username, phone, updated_at
		FROM contacts
		ORDER BY first_name, last_name
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []Contact
	for rows.Next() {
		var c Contact
		var ts string
		if err := rows.Scan(&c.UserID, &c.FirstName, &c.LastName, &c.Username, &c.Phone, &ts); err != nil {
			return nil, err
		}
		c.UpdatedAt, _ = time.Parse(time.RFC3339, ts)
		contacts = append(contacts, c)
	}
	return contacts, rows.Err()
}

// GetContact returns a single contact by user ID.
func (d *DB) GetContact(userID int64) (*Contact, error) {
	var c Contact
	var ts string
	err := d.sql.QueryRow(`
		SELECT user_id, first_name, last_name, username, phone, updated_at
		FROM contacts WHERE user_id = ?
	`, userID).Scan(&c.UserID, &c.FirstName, &c.LastName, &c.Username, &c.Phone, &ts)
	if err != nil {
		return nil, err
	}
	c.UpdatedAt, _ = time.Parse(time.RFC3339, ts)
	return &c, nil
}
