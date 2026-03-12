package client

import (
	"context"
	"fmt"

	"github.com/gotd/td/tg"
)

// Dialog represents a dialog from the API.
type Dialog struct {
	Peer        tg.PeerClass
	TopMessage  int
	UnreadCount int
}

// GetDialogs retrieves all dialogs (chats).
func (c *Client) GetDialogs(ctx context.Context) ([]Dialog, *tg.MessagesDialogsClass, error) {
	result, err := c.api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		OffsetPeer: &tg.InputPeerEmpty{},
		Limit:      500,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("get dialogs: %w", err)
	}
	return nil, &result, nil
}
