package dto

import (
	"time"
)

type SendTextRequest struct {
	To          string       `json:"to" validate:"required"`
	Text        string       `json:"text" validate:"required,max=4096"`
	MessageID   string       `json:"message_id,omitempty"`
	ContextInfo *ContextInfo `json:"context_info,omitempty"`
}

type SendMediaRequest struct {
	To          string       `json:"to" validate:"required"`
	Type        MediaType    `json:"type" validate:"required,oneof=image audio video document sticker"`
	Media       string       `json:"media" validate:"required"`
	Caption     string       `json:"caption,omitempty"`
	Filename    string       `json:"filename,omitempty"`
	MimeType    string       `json:"mime_type,omitempty"`
	MessageID   string       `json:"message_id,omitempty"`
	ContextInfo *ContextInfo `json:"context_info,omitempty"`
}

type SendLocationRequest struct {
	To          string       `json:"to" validate:"required"`
	Latitude    float64      `json:"latitude" validate:"required,latitude"`
	Longitude   float64      `json:"longitude" validate:"required,longitude"`
	Name        string       `json:"name,omitempty"`
	MessageID   string       `json:"message_id,omitempty"`
	ContextInfo *ContextInfo `json:"context_info,omitempty"`
}

type SendContactRequest struct {
	To          string       `json:"to" validate:"required"`
	Name        string       `json:"name" validate:"required"`
	Vcard       string       `json:"vcard" validate:"required"`
	MessageID   string       `json:"message_id,omitempty"`
	ContextInfo *ContextInfo `json:"context_info,omitempty"`
}

type MediaType string

const (
	MediaTypeImage    MediaType = "image"
	MediaTypeAudio    MediaType = "audio"
	MediaTypeVideo    MediaType = "video"
	MediaTypeDocument MediaType = "document"
	MediaTypeSticker  MediaType = "sticker"
)

type ContextInfo struct {
	StanzaID     *string  `json:"stanza_id,omitempty"`
	Participant  *string  `json:"participant,omitempty"`
	MentionedJID []string `json:"mentioned_jid,omitempty"`
}

type MessageSentResponse struct {
	Details   string    `json:"details"`
	Timestamp time.Time `json:"timestamp"`
	MessageID string    `json:"message_id"`
}

type SendMessageRequest struct {
	To       string           `json:"to" validate:"required,min=10,max=20"`
	Type     string           `json:"type" validate:"oneof=text image audio video document location contact"`
	Text     string           `json:"text,omitempty" validate:"required_if=Type text,max=4096"`
	Media    *MessageMedia    `json:"media,omitempty"`
	Location *MessageLocation `json:"location,omitempty"`
	Contact  *MessageContact  `json:"contact,omitempty"`
	Options  *MessageOptions  `json:"options,omitempty"`
}

type MessageMedia struct {
	URL      string `json:"url,omitempty" validate:"omitempty,url"`
	Data     string `json:"data,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	Caption  string `json:"caption,omitempty" validate:"max=1024"`
	Filename string `json:"filename,omitempty"`
}

type MessageLocation struct {
	Latitude  float64 `json:"latitude" validate:"required,latitude"`
	Longitude float64 `json:"longitude" validate:"required,longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
}

type MessageContact struct {
	Name         string `json:"name" validate:"required"`
	PhoneNumber  string `json:"phone_number" validate:"required,e164"`
	Organization string `json:"organization,omitempty"`
	Email        string `json:"email,omitempty" validate:"omitempty,email"`
}

type MessageOptions struct {
	DisablePreview      bool   `json:"disable_preview,omitempty"`
	DisableNotification bool   `json:"disable_notification,omitempty"`
	ReplyToMessageID    string `json:"reply_to_message_id,omitempty"`
	Ephemeral           bool   `json:"ephemeral,omitempty"`
}

type SendMessageResponse struct {
	MessageID   string     `json:"message_id"`
	SessionID   string     `json:"session_id"`
	To          string     `json:"to"`
	Type        string     `json:"type"`
	Status      string     `json:"status"`
	SentAt      time.Time  `json:"sent_at"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
	ReadAt      *time.Time `json:"read_at,omitempty"`
}

type BulkMessageRequest struct {
	Messages []SendMessageRequest `json:"messages" validate:"required,min=1,max=100,dive"`
	Options  *BulkOptions         `json:"options,omitempty"`
}

type BulkOptions struct {
	DelayBetweenMessages int  `json:"delay_between_messages,omitempty" validate:"min=0,max=60"`
	StopOnError          bool `json:"stop_on_error,omitempty"`
	MaxRetries           int  `json:"max_retries,omitempty" validate:"min=0,max=5"`
}

type BulkMessageResponse struct {
	BatchID      string                `json:"batch_id"`
	SessionID    string                `json:"session_id"`
	TotalCount   int                   `json:"total_count"`
	SuccessCount int                   `json:"success_count"`
	FailedCount  int                   `json:"failed_count"`
	Results      []SendMessageResponse `json:"results"`
	Errors       []BulkMessageError    `json:"errors,omitempty"`
	StartedAt    time.Time             `json:"started_at"`
	CompletedAt  *time.Time            `json:"completed_at,omitempty"`
}

type BulkMessageError struct {
	Index   int    `json:"index"`
	To      string `json:"to"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

type MessageStatusResponse struct {
	MessageID   string     `json:"message_id"`
	SessionID   string     `json:"session_id"`
	To          string     `json:"to"`
	Status      string     `json:"status"`
	SentAt      time.Time  `json:"sent_at"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
	ReadAt      *time.Time `json:"read_at,omitempty"`
	FailedAt    *time.Time `json:"failed_at,omitempty"`
	Error       string     `json:"error,omitempty"`
}

type MessageListRequest struct {
	Limit      int    `query:"limit" validate:"min=1,max=100"`
	Offset     int    `query:"offset" validate:"min=0"`
	Status     string `query:"status" validate:"omitempty,oneof=sent delivered read failed"`
	Type       string `query:"type" validate:"omitempty,oneof=text image audio video document location contact"`
	From       string `query:"from" validate:"omitempty,min=10,max=20"`
	To         string `query:"to" validate:"omitempty,min=10,max=20"`
	DateFrom   string `query:"date_from" validate:"omitempty,datetime=2006-01-02"`
	DateTo     string `query:"date_to" validate:"omitempty,datetime=2006-01-02"`
	SearchText string `query:"search" validate:"omitempty,max=100"`
}

type MessageListResponse struct {
	SessionID string            `json:"session_id"`
	Messages  []MessageResponse `json:"messages"`
	Total     int               `json:"total"`
}

type MessageResponse struct {
	ID        string           `json:"id"`
	MessageID string           `json:"message_id"`
	SessionID string           `json:"session_id"`
	From      string           `json:"from"`
	To        string           `json:"to"`
	Type      string           `json:"type"`
	Text      string           `json:"text,omitempty"`
	Media     *MessageMedia    `json:"media,omitempty"`
	Location  *MessageLocation `json:"location,omitempty"`
	Contact   *MessageContact  `json:"contact,omitempty"`
	Status    string           `json:"status"`
	Direction string           `json:"direction"`
	IsFromMe  bool             `json:"is_from_me"`
	Timestamp time.Time        `json:"timestamp"`
	SentAt    *time.Time       `json:"sent_at,omitempty"`
}

type SendImageRequest struct {
	To          string       `json:"to" validate:"required"`
	Image       string       `json:"image" validate:"required"`
	Caption     string       `json:"caption,omitempty" validate:"max=1024"`
	MessageID   string       `json:"message_id,omitempty"`
	ContextInfo *ContextInfo `json:"context_info,omitempty"`
}

type SendAudioRequest struct {
	To          string       `json:"to" validate:"required"`
	Audio       string       `json:"audio" validate:"required"`
	MessageID   string       `json:"message_id,omitempty"`
	ContextInfo *ContextInfo `json:"context_info,omitempty"`
}

type SendDocumentRequest struct {
	To          string       `json:"to" validate:"required"`
	Document    string       `json:"document" validate:"required"`
	Filename    string       `json:"filename" validate:"required"`
	Caption     string       `json:"caption,omitempty" validate:"max=1024"`
	MessageID   string       `json:"message_id,omitempty"`
	ContextInfo *ContextInfo `json:"context_info,omitempty"`
}

type SendVideoRequest struct {
	To          string       `json:"to" validate:"required"`
	Video       string       `json:"video" validate:"required"`
	Caption     string       `json:"caption,omitempty" validate:"max=1024"`
	MessageID   string       `json:"message_id,omitempty"`
	ContextInfo *ContextInfo `json:"context_info,omitempty"`
}

type SendStickerRequest struct {
	To          string       `json:"to" validate:"required,min=10,max=20"`
	Sticker     string       `json:"sticker" validate:"required,min=10"`
	MessageID   string       `json:"message_id,omitempty" validate:"omitempty,max=100"`
	ContextInfo *ContextInfo `json:"context_info,omitempty"`
}

type ReactRequest struct {
	To        string `json:"to" validate:"required"`
	MessageID string `json:"message_id" validate:"required"`
	Emoji     string `json:"emoji" validate:"required,max=10"`
}

type DeleteMessageRequest struct {
	To        string `json:"to" validate:"required"`
	MessageID string `json:"message_id" validate:"required"`
	ForAll    bool   `json:"for_all,omitempty"`
}

type ChatPresenceRequest struct {
	To       string `json:"to" validate:"required"`
	Presence string `json:"presence" validate:"required,oneof=available unavailable composing recording paused"`
}

type MarkReadRequest struct {
	To        string   `json:"to" validate:"required"`
	MessageID []string `json:"message_id,omitempty"`
}

type DownloadMediaRequest struct {
	MessageID string `json:"message_id" validate:"required"`
	Type      string `json:"type" validate:"required,oneof=image video audio document"`
}

type SessionPresenceRequest struct {
	Presence string `json:"presence" validate:"required,oneof=available unavailable"`
}

type ContactInfoRequest struct {
	Phone string `json:"phone" validate:"required,min=10,max=20"`
}

type CheckContactRequest struct {
	Phone []string `json:"phone" validate:"required,min=1,max=50,dive,min=10,max=20"`
}

type ContactAvatarRequest struct {
	Phone string `json:"phone" validate:"required,min=10,max=20"`
}

type SendButtonsRequest struct {
	To          string       `json:"to" validate:"required,min=10,max=20"`
	Text        string       `json:"text" validate:"required,min=1,max=4096"`
	Footer      string       `json:"footer,omitempty" validate:"max=60"`
	Buttons     []Button     `json:"buttons" validate:"required,min=1,max=3,dive"`
	MessageID   string       `json:"message_id,omitempty" validate:"omitempty,max=100"`
	ContextInfo *ContextInfo `json:"context_info,omitempty"`
}

type Button struct {
	ID    string `json:"id" validate:"required,min=1,max=256"`
	Text  string `json:"text" validate:"required,min=1,max=20"`
	Type  string `json:"type,omitempty" validate:"omitempty,oneof=reply url call"`
	URL   string `json:"url,omitempty" validate:"omitempty,url"`
	Phone string `json:"phone,omitempty" validate:"omitempty,min=10,max=20"`
}

type SendListRequest struct {
	To          string       `json:"to" validate:"required,min=10,max=20"`
	Text        string       `json:"text" validate:"required,min=1,max=1024"`
	Footer      string       `json:"footer,omitempty" validate:"max=60"`
	Title       string       `json:"title" validate:"required,min=1,max=60"`
	ButtonText  string       `json:"button_text" validate:"required,min=1,max=20"`
	Sections    []Section    `json:"sections" validate:"required,min=1,max=10,dive"`
	MessageID   string       `json:"message_id,omitempty" validate:"omitempty,max=100"`
	ContextInfo *ContextInfo `json:"context_info,omitempty"`
}

type Section struct {
	Title string    `json:"title" validate:"required,min=1,max=24"`
	Rows  []ListRow `json:"rows" validate:"required,min=1,max=10,dive"`
}

type ListRow struct {
	ID          string `json:"id" validate:"required,min=1,max=200"`
	Title       string `json:"title" validate:"required,min=1,max=24"`
	Description string `json:"description,omitempty" validate:"max=72"`
}

type SendPollRequest struct {
	To          string       `json:"to" validate:"required,min=10,max=20"`
	Name        string       `json:"name" validate:"required,min=1,max=140"`
	Options     []string     `json:"options" validate:"required,min=2,max=12,dive,min=1,max=100"`
	Selectable  int          `json:"selectable,omitempty" validate:"omitempty,min=1,max=12"`
	MessageID   string       `json:"message_id,omitempty" validate:"omitempty,max=100"`
	ContextInfo *ContextInfo `json:"context_info,omitempty"`
}

type EditMessageRequest struct {
	To        string `json:"to" validate:"required,min=10,max=20"`
	MessageID string `json:"message_id" validate:"required"`
	Text      string `json:"text" validate:"required,min=1,max=4096"`
}
