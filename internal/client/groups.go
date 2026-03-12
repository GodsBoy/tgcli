package client

import (
	"context"
	"fmt"

	"github.com/gotd/td/tg"
)

// GetFullChat retrieves full info for a basic group.
func (c *Client) GetFullChat(ctx context.Context, chatID int64) (*tg.MessagesChatFull, error) {
	result, err := c.api.MessagesGetFullChat(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("get full chat: %w", err)
	}
	return result, nil
}

// GetFullChannel retrieves full info for a channel or supergroup.
func (c *Client) GetFullChannel(ctx context.Context, channel *tg.InputChannel) (*tg.MessagesChatFull, error) {
	result, err := c.api.ChannelsGetFullChannel(ctx, channel)
	if err != nil {
		return nil, fmt.Errorf("get full channel: %w", err)
	}
	return result, nil
}

// GetChannelParticipants retrieves participants for a channel/supergroup.
func (c *Client) GetChannelParticipants(ctx context.Context, channel *tg.InputChannel, limit int) (*tg.ChannelsChannelParticipants, error) {
	result, err := c.api.ChannelsGetParticipants(ctx, &tg.ChannelsGetParticipantsRequest{
		Channel: channel,
		Filter:  &tg.ChannelParticipantsRecent{},
		Limit:   limit,
	})
	if err != nil {
		return nil, fmt.Errorf("get channel participants: %w", err)
	}
	participants, ok := result.(*tg.ChannelsChannelParticipants)
	if !ok {
		return nil, fmt.Errorf("unexpected participants type: %T", result)
	}
	return participants, nil
}
