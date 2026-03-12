package client

import (
	"context"
	"fmt"

	"github.com/gotd/td/tg"
)

// GetContacts retrieves all contacts.
func (c *Client) GetContacts(ctx context.Context) (*tg.ContactsContacts, error) {
	result, err := c.api.ContactsGetContacts(ctx, 0)
	if err != nil {
		return nil, fmt.Errorf("get contacts: %w", err)
	}

	contacts, ok := result.(*tg.ContactsContacts)
	if !ok {
		return nil, fmt.Errorf("unexpected contacts type: %T", result)
	}
	return contacts, nil
}
