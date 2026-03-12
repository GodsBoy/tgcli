package store

import "time"

// Chat represents a Telegram chat (DM, group, channel, or supergroup).
type Chat struct {
	ChatID        int64     `json:"chat_id"`
	Kind          string    `json:"kind"` // dm, group, channel, supergroup
	Name          string    `json:"name"`
	LastMessageTS time.Time `json:"last_message_ts"`
}

// Contact represents a Telegram contact.
type Contact struct {
	UserID    int64     `json:"user_id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Username  string    `json:"username"`
	Phone     string    `json:"phone"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Group represents a Telegram group or supergroup.
type Group struct {
	ChatID      int64     `json:"chat_id"`
	Title       string    `json:"title"`
	CreatorID   int64     `json:"creator_id"`
	CreatedTS   time.Time `json:"created_ts"`
	MemberCount int       `json:"member_count"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// GroupParticipant represents a member of a group.
type GroupParticipant struct {
	GroupChatID int64     `json:"group_chat_id"`
	UserID      int64     `json:"user_id"`
	Role        string    `json:"role"` // member, admin, creator
	UpdatedAt   time.Time `json:"updated_at"`
}

// Message represents a stored Telegram message.
type Message struct {
	ChatID       int64     `json:"chat_id"`
	ChatName     string    `json:"chat_name,omitempty"`
	MsgID        int       `json:"msg_id"`
	SenderID     int64     `json:"sender_id"`
	Timestamp    time.Time `json:"timestamp"`
	FromMe       bool      `json:"from_me"`
	Text         string    `json:"text"`
	MediaType    string    `json:"media_type,omitempty"`
	MediaCaption string    `json:"media_caption,omitempty"`
	ReplyToMsgID int       `json:"reply_to_msg_id,omitempty"`
	Snippet      string    `json:"snippet,omitempty"` // FTS5 search snippet
}

// UpsertMessageParams holds parameters for upserting a message.
type UpsertMessageParams struct {
	ChatID       int64
	MsgID        int
	SenderID     int64
	Timestamp    time.Time
	FromMe       bool
	Text         string
	MediaType    string
	MediaCaption string
	ReplyToMsgID int
}

// ListMessagesParams holds parameters for listing messages.
type ListMessagesParams struct {
	ChatID int64
	Limit  int
	Before time.Time
	After  time.Time
}

// SearchMessagesParams holds parameters for searching messages.
type SearchMessagesParams struct {
	Query  string
	ChatID int64
	Limit  int
}
