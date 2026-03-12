package client

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"

	"github.com/gotd/td/tg"
)

// SendText sends a text message to a peer.
func (c *Client) SendText(ctx context.Context, peer tg.InputPeerClass, text string) (tg.UpdatesClass, error) {
	return c.api.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
		Peer:     peer,
		Message:  text,
		RandomID: randomID(),
	})
}

// SendMedia sends a media message (file/photo) to a peer.
func (c *Client) SendMedia(ctx context.Context, peer tg.InputPeerClass, media tg.InputMediaClass, caption string) (tg.UpdatesClass, error) {
	return c.api.MessagesSendMedia(ctx, &tg.MessagesSendMediaRequest{
		Peer:     peer,
		Media:    media,
		Message:  caption,
		RandomID: randomID(),
	})
}

// GetHistory retrieves message history for a peer.
func (c *Client) GetHistory(ctx context.Context, peer tg.InputPeerClass, limit int, offsetID int) (tg.MessagesMessagesClass, error) {
	result, err := c.api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:     peer,
		Limit:    limit,
		OffsetID: offsetID,
	})
	if err != nil {
		return nil, fmt.Errorf("get history: %w", err)
	}
	return result, nil
}

func randomID() int64 {
	var buf [8]byte
	_, _ = rand.Read(buf[:])
	return int64(binary.LittleEndian.Uint64(buf[:]))
}
