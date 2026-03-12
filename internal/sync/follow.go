package sync

import (
	"context"
	"log"
	"time"

	"github.com/gotd/td/tg"

	"github.com/GodsBoy/tgcli/internal/store"
)

// Follow listens for new messages via updates and persists them.
// Blocks until ctx is cancelled.
func (e *Engine) Follow(ctx context.Context) error {
	log.Println("following for new messages (Ctrl+C to stop)...")

	// Use getState to get the initial update state, then poll for differences.
	state, err := e.api.UpdatesGetState(ctx)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			diff, err := e.api.UpdatesGetDifference(ctx, &tg.UpdatesGetDifferenceRequest{
				Pts:  state.Pts,
				Date: state.Date,
				Qts:  state.Qts,
			})
			if err != nil {
				log.Printf("get difference: %v", err)
				continue
			}

			switch d := diff.(type) {
			case *tg.UpdatesDifference:
				e.processNewMessages(d.NewMessages)
				if d.State.Pts > state.Pts {
					state.Pts = d.State.Pts
				}
				if d.State.Date > state.Date {
					state.Date = d.State.Date
				}
				if d.State.Qts > state.Qts {
					state.Qts = d.State.Qts
				}
			case *tg.UpdatesDifferenceSlice:
				e.processNewMessages(d.NewMessages)
				if d.IntermediateState.Pts > state.Pts {
					state.Pts = d.IntermediateState.Pts
				}
				if d.IntermediateState.Date > state.Date {
					state.Date = d.IntermediateState.Date
				}
				if d.IntermediateState.Qts > state.Qts {
					state.Qts = d.IntermediateState.Qts
				}
			case *tg.UpdatesDifferenceEmpty:
				// No updates, keep polling.
			case *tg.UpdatesDifferenceTooLong:
				state.Pts = d.Pts
			}
		}
	}
}

func (e *Engine) processNewMessages(messages []tg.MessageClass) {
	for _, m := range messages {
		msg, ok := m.(*tg.Message)
		if !ok {
			continue
		}

		chatID := extractChatID(msg)
		if chatID == 0 {
			continue
		}

		senderID := extractSenderID(msg)
		mediaType, mediaCaption := extractMedia(msg)
		replyTo := 0
		if msg.ReplyTo != nil {
			if rh, ok := msg.ReplyTo.(*tg.MessageReplyHeader); ok {
				replyTo = rh.ReplyToMsgID
			}
		}

		if err := e.db.UpsertMessage(store.UpsertMessageParams{
			ChatID:       chatID,
			MsgID:        msg.ID,
			SenderID:     senderID,
			Timestamp:    time.Unix(int64(msg.Date), 0),
			FromMe:       msg.Out,
			Text:         msg.Message,
			MediaType:    mediaType,
			MediaCaption: mediaCaption,
			ReplyToMsgID: replyTo,
		}); err != nil {
			log.Printf("upsert new message %d: %v", msg.ID, err)
			continue
		}

		_ = e.db.UpsertChat(store.Chat{
			ChatID:        chatID,
			LastMessageTS: time.Unix(int64(msg.Date), 0),
		})

		log.Printf("new message in chat %d: msg_id=%d", chatID, msg.ID)
	}
}

func extractChatID(msg *tg.Message) int64 {
	if msg.PeerID == nil {
		return 0
	}
	switch p := msg.PeerID.(type) {
	case *tg.PeerUser:
		return p.UserID
	case *tg.PeerChat:
		return p.ChatID
	case *tg.PeerChannel:
		return p.ChannelID
	}
	return 0
}
