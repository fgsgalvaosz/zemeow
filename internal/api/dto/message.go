package dto

import (
	"time"
)

// === MESSAGE DTOs ===

// SendTextRequest para envio de mensagem de texto seguindo padrão do sistema de referência
type SendTextRequest struct {
	To          string       `json:"to" validate:"required"`
	Text        string       `json:"text" validate:"required,max=4096"`
	MessageID   string       `json:"message_id,omitempty"`
	ContextInfo *ContextInfo `json:"context_info,omitempty"`
}

// SendMediaRequest para envio unificado de mídia (imagem, áudio, vídeo, documento, sticker)
type SendMediaRequest struct {
	To          string       `json:"to" validate:"required"`
	Type        MediaType    `json:"type" validate:"required,oneof=image audio video document sticker"`
	Media       string       `json:"media" validate:"required"` // Base64 data URL
	Caption     string       `json:"caption,omitempty"`
	Filename    string       `json:"filename,omitempty"`
	MimeType    string       `json:"mime_type,omitempty"`
	MessageID   string       `json:"message_id,omitempty"`
	ContextInfo *ContextInfo `json:"context_info,omitempty"`
}

// SendLocationRequest para envio de localização
type SendLocationRequest struct {
	To          string       `json:"to" validate:"required"`
	Latitude    float64      `json:"latitude" validate:"required,latitude"`
	Longitude   float64      `json:"longitude" validate:"required,longitude"`
	Name        string       `json:"name,omitempty"`
	MessageID   string       `json:"message_id,omitempty"`
	ContextInfo *ContextInfo `json:"context_info,omitempty"`
}

// SendContactRequest para envio de contato
type SendContactRequest struct {
	To          string       `json:"to" validate:"required"`
	Name        string       `json:"name" validate:"required"`
	Vcard       string       `json:"vcard" validate:"required"`
	MessageID   string       `json:"message_id,omitempty"`
	ContextInfo *ContextInfo `json:"context_info,omitempty"`
}

// MediaType enumera os tipos de mídia suportados
type MediaType string

const (
	MediaTypeImage    MediaType = "image"
	MediaTypeAudio    MediaType = "audio"
	MediaTypeVideo    MediaType = "video"
	MediaTypeDocument MediaType = "document"
	MediaTypeSticker  MediaType = "sticker"
)

// ContextInfo para replies e menções (baseado no sistema de referência)
type ContextInfo struct {
	StanzaID     *string  `json:"stanza_id,omitempty"`     // Para reply
	Participant  *string  `json:"participant,omitempty"`   // Para reply em grupo
	MentionedJID []string `json:"mentioned_jid,omitempty"` // Para menções
}

// MessageSentResponse resposta padrão para mensagens enviadas (seguindo padrão do ref)
type MessageSentResponse struct {
	Details   string    `json:"details"`
	Timestamp time.Time `json:"timestamp"`
	MessageID string    `json:"message_id"`
}

// SendMessageRequest requisição genérica para envio de mensagem (mantido para compatibilidade)
type SendMessageRequest struct {
	To       string                 `json:"to" validate:"required,min=10,max=20"`
	Type     string                 `json:"type" validate:"oneof=text image audio video document location contact"`
	Text     string                 `json:"text,omitempty" validate:"required_if=Type text,max=4096"`
	Media    *MessageMedia          `json:"media,omitempty"`
	Location *MessageLocation       `json:"location,omitempty"`
	Contact  *MessageContact        `json:"contact,omitempty"`
	Options  *MessageOptions        `json:"options,omitempty"`
}

// MessageMedia estrutura para mídia da mensagem
type MessageMedia struct {
	URL      string `json:"url,omitempty" validate:"omitempty,url"`
	Data     string `json:"data,omitempty"` // Base64 encoded
	MimeType string `json:"mime_type,omitempty"`
	Caption  string `json:"caption,omitempty" validate:"max=1024"`
	Filename string `json:"filename,omitempty"`
}

// MessageLocation estrutura para localização
type MessageLocation struct {
	Latitude  float64 `json:"latitude" validate:"required,latitude"`
	Longitude float64 `json:"longitude" validate:"required,longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
}

// MessageContact estrutura para contato
type MessageContact struct {
	Name         string `json:"name" validate:"required"`
	PhoneNumber  string `json:"phone_number" validate:"required,e164"`
	Organization string `json:"organization,omitempty"`
	Email        string `json:"email,omitempty" validate:"omitempty,email"`
}

// MessageOptions opções adicionais da mensagem
type MessageOptions struct {
	DisablePreview   bool   `json:"disable_preview,omitempty"`
	DisableNotification bool `json:"disable_notification,omitempty"`
	ReplyToMessageID string `json:"reply_to_message_id,omitempty"`
	Ephemeral        bool   `json:"ephemeral,omitempty"`
}

// SendMessageResponse resposta do envio de mensagem
type SendMessageResponse struct {
	MessageID   string    `json:"message_id"`
	SessionID   string    `json:"session_id"`
	To          string    `json:"to"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	SentAt      time.Time `json:"sent_at"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
	ReadAt      *time.Time `json:"read_at,omitempty"`
}

// BulkMessageRequest requisição para envio em lote
type BulkMessageRequest struct {
	Messages []SendMessageRequest `json:"messages" validate:"required,min=1,max=100,dive"`
	Options  *BulkOptions         `json:"options,omitempty"`
}

// BulkOptions opções para envio em lote
type BulkOptions struct {
	DelayBetweenMessages int  `json:"delay_between_messages,omitempty" validate:"min=0,max=60"`
	StopOnError          bool `json:"stop_on_error,omitempty"`
	MaxRetries           int  `json:"max_retries,omitempty" validate:"min=0,max=5"`
}

// BulkMessageResponse resposta do envio em lote
type BulkMessageResponse struct {
	BatchID      string                 `json:"batch_id"`
	SessionID    string                 `json:"session_id"`
	TotalCount   int                    `json:"total_count"`
	SuccessCount int                    `json:"success_count"`
	FailedCount  int                    `json:"failed_count"`
	Results      []SendMessageResponse  `json:"results"`
	Errors       []BulkMessageError     `json:"errors,omitempty"`
	StartedAt    time.Time              `json:"started_at"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
}

// BulkMessageError erro em envio em lote
type BulkMessageError struct {
	Index   int    `json:"index"`
	To      string `json:"to"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

// MessageStatusResponse resposta do status da mensagem
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

// MessageListRequest requisição para listar mensagens
type MessageListRequest struct {
	Limit       int    `query:"limit" validate:"min=1,max=100"`
	Offset      int    `query:"offset" validate:"min=0"`
	Status      string `query:"status" validate:"omitempty,oneof=sent delivered read failed"`
	Type        string `query:"type" validate:"omitempty,oneof=text image audio video document location contact"`
	From        string `query:"from" validate:"omitempty,min=10,max=20"`
	To          string `query:"to" validate:"omitempty,min=10,max=20"`
	DateFrom    string `query:"date_from" validate:"omitempty,datetime=2006-01-02"`
	DateTo      string `query:"date_to" validate:"omitempty,datetime=2006-01-02"`
	SearchText  string `query:"search" validate:"omitempty,max=100"`
}

// MessageListResponse resposta da lista de mensagens
type MessageListResponse struct {
	SessionID string            `json:"session_id"`
	Messages  []MessageResponse `json:"messages"`
	Total     int               `json:"total"`
}

// MessageResponse estrutura de resposta de mensagem
type MessageResponse struct {
	ID          string     `json:"id"`
	MessageID   string     `json:"message_id"`
	SessionID   string     `json:"session_id"`
	From        string     `json:"from"`
	To          string     `json:"to"`
	Type        string     `json:"type"`
	Text        string     `json:"text,omitempty"`
	Media       *MessageMedia `json:"media,omitempty"`
	Location    *MessageLocation `json:"location,omitempty"`
	Contact     *MessageContact `json:"contact,omitempty"`
	Status      string     `json:"status"`
	Direction   string     `json:"direction"` // incoming, outgoing
	IsFromMe    bool       `json:"is_from_me"`
	Timestamp   time.Time  `json:"timestamp"`
	SentAt      *time.Time `json:"sent_at,omitempty"`
}

// === NOVOS DTOs PARA ENDPOINTS FALTANTES ===

// SendImageRequest para envio de imagem
type SendImageRequest struct {
	To          string       `json:"to" validate:"required"`
	Image       string       `json:"image" validate:"required"` // Base64 ou URL
	Caption     string       `json:"caption,omitempty" validate:"max=1024"`
	MessageID   string       `json:"message_id,omitempty"`
	ContextInfo *ContextInfo `json:"context_info,omitempty"`
}

// SendAudioRequest para envio de áudio
type SendAudioRequest struct {
	To          string       `json:"to" validate:"required"`
	Audio       string       `json:"audio" validate:"required"` // Base64 ou URL
	MessageID   string       `json:"message_id,omitempty"`
	ContextInfo *ContextInfo `json:"context_info,omitempty"`
}

// SendDocumentRequest para envio de documento
type SendDocumentRequest struct {
	To          string       `json:"to" validate:"required"`
	Document    string       `json:"document" validate:"required"` // Base64 ou URL
	Filename    string       `json:"filename" validate:"required"`
	Caption     string       `json:"caption,omitempty" validate:"max=1024"`
	MessageID   string       `json:"message_id,omitempty"`
	ContextInfo *ContextInfo `json:"context_info,omitempty"`
}

// SendVideoRequest para envio de vídeo
type SendVideoRequest struct {
	To          string       `json:"to" validate:"required"`
	Video       string       `json:"video" validate:"required"` // Base64 ou URL
	Caption     string       `json:"caption,omitempty" validate:"max=1024"`
	MessageID   string       `json:"message_id,omitempty"`
	ContextInfo *ContextInfo `json:"context_info,omitempty"`
}

// SendStickerRequest para envio de sticker
type SendStickerRequest struct {
	To          string       `json:"to" validate:"required,min=10,max=20"` // Validação de telefone
	Sticker     string       `json:"sticker" validate:"required,min=10"`   // Base64 ou URL
	MessageID   string       `json:"message_id,omitempty" validate:"omitempty,max=100"`
	ContextInfo *ContextInfo `json:"context_info,omitempty"`
}

// ReactRequest para reagir a mensagem
type ReactRequest struct {
	To        string `json:"to" validate:"required"`
	MessageID string `json:"message_id" validate:"required"`
	Emoji     string `json:"emoji" validate:"required,max=10"`
}

// DeleteMessageRequest para deletar mensagem
type DeleteMessageRequest struct {
	To        string `json:"to" validate:"required"`
	MessageID string `json:"message_id" validate:"required"`
	ForAll    bool   `json:"for_all,omitempty"`
}

// === DTOs PARA OPERAÇÕES DE CHAT ===

// ChatPresenceRequest para definir presença no chat
type ChatPresenceRequest struct {
	To       string `json:"to" validate:"required"`
	Presence string `json:"presence" validate:"required,oneof=available unavailable composing recording paused"`
}

// MarkReadRequest para marcar mensagens como lidas
type MarkReadRequest struct {
	To        string   `json:"to" validate:"required"`
	MessageID []string `json:"message_id,omitempty"`
}

// DownloadMediaRequest para download de mídia
type DownloadMediaRequest struct {
	MessageID string `json:"message_id" validate:"required"`
	Type      string `json:"type" validate:"required,oneof=image video audio document"`
}

// === DTOs PARA OPERAÇÕES DE SESSÃO (WHATSAPP) ===

// SessionPresenceRequest para definir presença da sessão
type SessionPresenceRequest struct {
	Presence string `json:"presence" validate:"required,oneof=available unavailable"`
}

// ContactInfoRequest para obter informações de contato
type ContactInfoRequest struct {
	Phone string `json:"phone" validate:"required,min=10,max=20"` // Validação de telefone
}

// CheckContactRequest para verificar se contatos existem
type CheckContactRequest struct {
	Phone []string `json:"phone" validate:"required,min=1,max=50,dive,min=10,max=20"` // Máximo 50 números
}

// ContactAvatarRequest para obter avatar de contato
type ContactAvatarRequest struct {
	Phone string `json:"phone" validate:"required,min=10,max=20"` // Validação de telefone
}

