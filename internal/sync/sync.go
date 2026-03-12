package sync

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gotd/td/tg"

	"github.com/GodsBoy/tgcli/internal/store"
)

// Options configures sync behavior.
type Options struct {
	Follow bool
}

// Result contains sync statistics.
type Result struct {
	MessagesStored int
	ChatsStored    int
}

// Engine handles syncing Telegram messages to the local store.
type Engine struct {
	api *tg.Client
	db  *store.DB
}

// New creates a new sync engine.
func New(api *tg.Client, db *store.DB) *Engine {
	return &Engine{api: api, db: db}
}

// Run performs the sync: fetches dialogs, then fetches history for each dialog.
func (e *Engine) Run(ctx context.Context, opts Options) (*Result, error) {
	result := &Result{}

	// Fetch all dialogs.
	dialogs, err := e.fetchDialogs(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch dialogs: %w", err)
	}

	// Process each dialog.
	for _, d := range dialogs {
		chatID, kind, name := extractDialogInfo(d)
		if chatID == 0 {
			continue
		}

		if err := e.db.UpsertChat(store.Chat{
			ChatID: chatID,
			Kind:   kind,
			Name:   name,
		}); err != nil {
			log.Printf("upsert chat %d: %v", chatID, err)
			continue
		}
		result.ChatsStored++

		// Fetch recent messages for this dialog.
		count, err := e.syncChatHistory(ctx, d.peer, chatID)
		if err != nil {
			log.Printf("sync history for %d: %v", chatID, err)
			continue
		}
		result.MessagesStored += count
	}

	return result, nil
}

type dialogInfo struct {
	peer    tg.InputPeerClass
	chatID  int64
	kind    string
	name    string
}

func (e *Engine) fetchDialogs(ctx context.Context) ([]dialogInfo, error) {
	result, err := e.api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		OffsetPeer: &tg.InputPeerEmpty{},
		Limit:      500,
	})
	if err != nil {
		return nil, err
	}

	var infos []dialogInfo

	switch r := result.(type) {
	case *tg.MessagesDialogs:
		infos = extractFromDialogs(r.Dialogs, r.Chats, r.Users)
	case *tg.MessagesDialogsSlice:
		infos = extractFromDialogs(r.Dialogs, r.Chats, r.Users)
	default:
		return nil, fmt.Errorf("unexpected dialogs type: %T", result)
	}

	return infos, nil
}

func extractFromDialogs(dialogs []tg.DialogClass, chats []tg.ChatClass, users []tg.UserClass) []dialogInfo {
	chatMap := make(map[int64]tg.ChatClass)
	for _, c := range chats {
		switch chat := c.(type) {
		case *tg.Chat:
			chatMap[chat.ID] = chat
		case *tg.Channel:
			chatMap[chat.ID] = chat
		}
	}
	userMap := make(map[int64]*tg.User)
	for _, u := range users {
		if user, ok := u.(*tg.User); ok {
			userMap[user.ID] = user
		}
	}

	var infos []dialogInfo
	for _, d := range dialogs {
		dialog, ok := d.(*tg.Dialog)
		if !ok {
			continue
		}

		var info dialogInfo
		switch p := dialog.Peer.(type) {
		case *tg.PeerUser:
			info.chatID = p.UserID
			info.kind = "dm"
			info.peer = &tg.InputPeerUser{UserID: p.UserID}
			if u, ok := userMap[p.UserID]; ok {
				info.name = userName(u)
				info.peer = &tg.InputPeerUser{UserID: p.UserID, AccessHash: u.AccessHash}
			}
		case *tg.PeerChat:
			info.chatID = p.ChatID
			info.kind = "group"
			info.peer = &tg.InputPeerChat{ChatID: p.ChatID}
			if c, ok := chatMap[p.ChatID]; ok {
				if chat, ok := c.(*tg.Chat); ok {
					info.name = chat.Title
				}
			}
		case *tg.PeerChannel:
			info.chatID = p.ChannelID
			info.peer = &tg.InputPeerChannel{ChannelID: p.ChannelID}
			if c, ok := chatMap[p.ChannelID]; ok {
				if ch, ok := c.(*tg.Channel); ok {
					info.name = ch.Title
					info.peer = &tg.InputPeerChannel{ChannelID: p.ChannelID, AccessHash: ch.AccessHash}
					if ch.Broadcast {
						info.kind = "channel"
					} else {
						info.kind = "supergroup"
					}
				}
			}
			if info.kind == "" {
				info.kind = "channel"
			}
		}

		if info.chatID != 0 {
			infos = append(infos, info)
		}
	}
	return infos
}

func (e *Engine) syncChatHistory(ctx context.Context, peer tg.InputPeerClass, chatID int64) (int, error) {
	result, err := e.api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:  peer,
		Limit: 100,
	})
	if err != nil {
		return 0, err
	}

	var messages []tg.MessageClass
	switch r := result.(type) {
	case *tg.MessagesMessages:
		messages = r.Messages
	case *tg.MessagesMessagesSlice:
		messages = r.Messages
	case *tg.MessagesChannelMessages:
		messages = r.Messages
	default:
		return 0, fmt.Errorf("unexpected messages type: %T", result)
	}

	count := 0
	for _, m := range messages {
		msg, ok := m.(*tg.Message)
		if !ok {
			continue
		}

		senderID := extractSenderID(msg)
		fromMe := false
		if senderID == 0 {
			fromMe = msg.Out
		}

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
			FromMe:       fromMe,
			Text:         msg.Message,
			MediaType:    mediaType,
			MediaCaption: mediaCaption,
			ReplyToMsgID: replyTo,
		}); err != nil {
			log.Printf("upsert message %d in chat %d: %v", msg.ID, chatID, err)
			continue
		}

		// Update chat last_message_ts.
		_ = e.db.UpsertChat(store.Chat{
			ChatID:        chatID,
			LastMessageTS: time.Unix(int64(msg.Date), 0),
		})
		count++
	}

	return count, nil
}

func extractDialogInfo(d dialogInfo) (int64, string, string) {
	return d.chatID, d.kind, d.name
}

func extractSenderID(msg *tg.Message) int64 {
	if msg.FromID == nil {
		return 0
	}
	switch p := msg.FromID.(type) {
	case *tg.PeerUser:
		return p.UserID
	case *tg.PeerChat:
		return p.ChatID
	case *tg.PeerChannel:
		return p.ChannelID
	}
	return 0
}

func extractMedia(msg *tg.Message) (mediaType, caption string) {
	if msg.Media == nil {
		return "", ""
	}
	switch msg.Media.(type) {
	case *tg.MessageMediaPhoto:
		return "photo", msg.Message
	case *tg.MessageMediaDocument:
		return "document", msg.Message
	case *tg.MessageMediaGeo:
		return "geo", ""
	case *tg.MessageMediaContact:
		return "contact", ""
	case *tg.MessageMediaWebPage:
		return "webpage", ""
	case *tg.MessageMediaVenue:
		return "venue", ""
	case *tg.MessageMediaPoll:
		return "poll", ""
	default:
		return "other", ""
	}
}

func userName(u *tg.User) string {
	name := u.FirstName
	if u.LastName != "" {
		name += " " + u.LastName
	}
	if name == "" {
		name = u.Username
	}
	return name
}
