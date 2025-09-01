package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Message struct {
	ID                uuid.UUID `json:"id" db:"id"`
	SessionID         uuid.UUID `json:"session_id" db:"session_id"`
	MessageID         string    `json:"message_id" db:"message_id"`
	WhatsAppMessageID *string   `json:"whatsapp_message_id" db:"whatsapp_message_id"`

	ChatJID      string  `json:"chat_jid" db:"chat_jid"`
	SenderJID    string  `json:"sender_jid" db:"from_jid"`
	RecipientJID *string `json:"recipient_jid" db:"to_jid"`

	MessageType string          `json:"message_type" db:"message_type"`
	Content     *string         `json:"content" db:"content"`
	RawMessage  json.RawMessage `json:"raw_message" db:"raw_message"`

	MediaURL      *string `json:"media_url" db:"media_url"`
	MediaType     *string `json:"media_type" db:"media_type"`
	MediaSize     *int64  `json:"media_size" db:"media_size"`
	MediaFilename *string `json:"media_filename" db:"media_filename"`
	MediaSHA256   *string `json:"media_sha256" db:"media_sha256"`
	MediaKey      []byte  `json:"media_key" db:"media_key"`

	MinIOMediaID *string `json:"minio_media_id" db:"minio_media_id"`
	MinIOPath    *string `json:"minio_path" db:"minio_path"`
	MinIOURL     *string `json:"minio_url" db:"minio_url"`
	MinIOBucket  *string `json:"minio_bucket" db:"minio_bucket"`

	Caption          *string `json:"caption" db:"caption"`
	QuotedMessageID  *string `json:"quoted_message_id" db:"quoted_message_id"`
	QuotedContent    *string `json:"quoted_content" db:"quoted_content"`
	ReplyToMessageID *string `json:"reply_to_message_id" db:"reply_to_message_id"`
	ContextInfo      JSONB   `json:"context_info" db:"context_info"`

	Direction   string `json:"direction" db:"direction"`
	Status      string `json:"status" db:"status"`
	IsFromMe    bool   `json:"is_from_me" db:"is_from_me"`
	IsEphemeral bool   `json:"is_ephemeral" db:"is_ephemeral"`
	IsViewOnce  bool   `json:"is_view_once" db:"is_view_once"`
	IsForwarded bool   `json:"is_forwarded" db:"is_forwarded"`
	IsEdit      bool   `json:"is_edit" db:"is_edit"`
	EditVersion int    `json:"edit_version" db:"edit_version"`

	Mentions          pq.StringArray `json:"mentions" db:"mentions"`
	ReactionEmoji     *string        `json:"reaction_emoji" db:"reaction_emoji"`
	ReactionTimestamp *time.Time     `json:"reaction_timestamp" db:"reaction_timestamp"`

	LocationLatitude  *float64 `json:"location_latitude" db:"location_latitude"`
	LocationLongitude *float64 `json:"location_longitude" db:"location_longitude"`
	LocationName      *string  `json:"location_name" db:"location_name"`
	LocationAddress   *string  `json:"location_address" db:"location_address"`

	ContactName  *string `json:"contact_name" db:"contact_name"`
	ContactPhone *string `json:"contact_phone" db:"contact_phone"`
	ContactVCard *string `json:"contact_vcard" db:"contact_vcard"`

	StickerPackID   *string `json:"sticker_pack_id" db:"sticker_pack_id"`
	StickerPackName *string `json:"sticker_pack_name" db:"sticker_pack_name"`

	GroupInviteCode       *string    `json:"group_invite_code" db:"group_invite_code"`
	GroupInviteExpiration *time.Time `json:"group_invite_expiration" db:"group_invite_expiration"`

	PollName            *string `json:"poll_name" db:"poll_name"`
	PollOptions         JSONB   `json:"poll_options" db:"poll_options"`
	PollSelectableCount *int    `json:"poll_selectable_count" db:"poll_selectable_count"`

	ErrorMessage *string `json:"error_message" db:"error_message"`
	RetryCount   int     `json:"retry_count" db:"retry_count"`

	Timestamp time.Time `json:"timestamp" db:"timestamp"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	Session *Session `json:"session,omitempty" db:"-"`
}

type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, j)
}

type MessageType string

const (
	MessageTypeText        MessageType = "text"
	MessageTypeImage       MessageType = "image"
	MessageTypeAudio       MessageType = "audio"
	MessageTypeVideo       MessageType = "video"
	MessageTypeDocument    MessageType = "document"
	MessageTypeSticker     MessageType = "sticker"
	MessageTypeLocation    MessageType = "location"
	MessageTypeContact     MessageType = "contact"
	MessageTypeGroupInvite MessageType = "group_invite"
	MessageTypePoll        MessageType = "poll"
	MessageTypeReaction    MessageType = "reaction"
	MessageTypeSystem      MessageType = "system"
	MessageTypeCall        MessageType = "call"
	MessageTypeUnknown     MessageType = "unknown"
)

type MessageDirection string

const (
	MessageDirectionIncoming MessageDirection = "incoming"
	MessageDirectionOutgoing MessageDirection = "outgoing"
)

type MessageStatus string

const (
	MessageStatusSent          MessageStatus = "sent"
	MessageStatusDelivered     MessageStatus = "delivered"
	MessageStatusRead          MessageStatus = "read"
	MessageStatusFailed        MessageStatus = "failed"
	MessageStatusReceived      MessageStatus = "received"
	MessageStatusPending       MessageStatus = "pending"
	MessageStatusServerAck     MessageStatus = "server_ack"
	MessageStatusRetry         MessageStatus = "retry"
	MessageStatusUndecryptable MessageStatus = "undecryptable"
)

func (m *Message) IsMediaMessage() bool {
	return m.MessageType == string(MessageTypeImage) ||
		m.MessageType == string(MessageTypeAudio) ||
		m.MessageType == string(MessageTypeVideo) ||
		m.MessageType == string(MessageTypeDocument) ||
		m.MessageType == string(MessageTypeSticker)
}

func (m *Message) HasLocation() bool {
	return m.LocationLatitude != nil && m.LocationLongitude != nil
}

func (m *Message) HasContact() bool {
	return m.ContactName != nil || m.ContactPhone != nil
}

func (m *Message) IsReply() bool {
	return m.ReplyToMessageID != nil && *m.ReplyToMessageID != ""
}

func (m *Message) HasMentions() bool {
	return len(m.Mentions) > 0
}

func (m *Message) IsReaction() bool {
	return m.ReactionEmoji != nil && *m.ReactionEmoji != ""
}

func (m *Message) GetDisplayContent() string {
	if m.Content != nil && *m.Content != "" {
		return *m.Content
	}
	if m.Caption != nil && *m.Caption != "" {
		return *m.Caption
	}
	if m.IsMediaMessage() {
		return "[" + m.MessageType + "]"
	}
	if m.HasLocation() {
		return "[Location]"
	}
	if m.HasContact() {
		return "[Contact]"
	}
	return "[" + m.MessageType + "]"
}

type MessageFilter struct {
	SessionID   *uuid.UUID        `json:"session_id"`
	ChatJID     *string           `json:"chat_jid"`
	SenderJID   *string           `json:"sender_jid"`
	MessageType *MessageType      `json:"message_type"`
	Direction   *MessageDirection `json:"direction"`
	Status      *MessageStatus    `json:"status"`
	IsFromMe    *bool             `json:"is_from_me"`
	HasMedia    *bool             `json:"has_media"`
	IsEphemeral *bool             `json:"is_ephemeral"`
	DateFrom    *time.Time        `json:"date_from"`
	DateTo      *time.Time        `json:"date_to"`
	SearchText  *string           `json:"search_text"`
	Limit       int               `json:"limit"`
	Offset      int               `json:"offset"`
}

type MessageStatistics struct {
	TotalMessages    int64            `json:"total_messages"`
	MessagesByType   map[string]int64 `json:"messages_by_type"`
	MessagesByStatus map[string]int64 `json:"messages_by_status"`
	MediaMessages    int64            `json:"media_messages"`
	IncomingMessages int64            `json:"incoming_messages"`
	OutgoingMessages int64            `json:"outgoing_messages"`
	UnreadMessages   int64            `json:"unread_messages"`
	FailedMessages   int64            `json:"failed_messages"`
	LastMessageTime  *time.Time       `json:"last_message_time"`
}
