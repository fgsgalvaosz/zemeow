package session

import (
	"context"
	"errors"
	"time"

	"github.com/felipe/zemeow/internal/logger"
	"go.mau.fi/whatsmeow"
)

// Status representa o status de uma sessão
type Status string

const (
	StatusDisconnected  Status = "disconnected"
	StatusConnecting    Status = "connecting"
	StatusConnected     Status = "connected"
	StatusAuthenticated Status = "authenticated"
	StatusError         Status = "error"
)

// Erros comuns
var (
	ErrSessionExists       = errors.New("session already exists")
	ErrSessionNotFound     = errors.New("session not found")
	ErrSessionConnected    = errors.New("session is already connected")
	ErrSessionNotConnected = errors.New("session is not connected")
	ErrInvalidConfig       = errors.New("invalid session configuration")
)

// Session representa uma sessão WhatsApp
type Session struct {
	ID          string
	Config      *Config
	Status      Status
	Client      *whatsmeow.Client
	CreatedAt   time.Time
	ConnectedAt *time.Time
	logger      logger.Logger
}

// Config representa a configuração de uma sessão
type Config struct {
	SessionID     string         `json:"session_id,omitempty"`
	Name          string         `json:"name"`
	APIKey        string         `json:"api_key,omitempty"`
	Proxy         *ProxyConfig   `json:"proxy,omitempty"`
	Webhook       *WebhookConfig `json:"webhook,omitempty"`
	AutoReconnect bool           `json:"auto_reconnect"`
	LogLevel      string         `json:"log_level"`
}

// ProxyConfig representa a configuração de proxy
type ProxyConfig struct {
	Enabled  bool   `json:"enabled"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Type     string `json:"type"` // http, socks5
}

// WebhookConfig representa a configuração de webhook
type WebhookConfig struct {
	URL    string   `json:"url"`
	Events []string `json:"events"`
	Secret string   `json:"secret,omitempty"`
}

// Connect conecta a sessão ao WhatsApp
func (s *Session) Connect(ctx context.Context) error {
	if s.Status == StatusConnected || s.Status == StatusAuthenticated {
		return ErrSessionConnected
	}

	s.Status = StatusConnecting
	s.logger.Info().Msg("Connecting session to WhatsApp")

	// TODO: Implementar lógica de conexão com whatsmeow
	// Isso será implementado na próxima fase

	return nil
}

// Disconnect desconecta a sessão do WhatsApp
func (s *Session) Disconnect() error {
	if s.Status == StatusDisconnected {
		return ErrSessionNotConnected
	}

	if s.Client != nil {
		s.Client.Disconnect()
	}

	s.Status = StatusDisconnected
	s.logger.Info().Msg("Session disconnected")

	return nil
}

// GetStatus retorna o status atual da sessão
func (s *Session) GetStatus() Status {
	return s.Status
}

// IsConnected verifica se a sessão está conectada
func (s *Session) IsConnected() bool {
	return s.Status == StatusConnected || s.Status == StatusAuthenticated
}

// GetInfo retorna informações da sessão
func (s *Session) GetInfo() *SessionInfo {
	info := &SessionInfo{
		ID:        s.ID,
		Name:      s.Config.Name,
		Status:    s.Status,
		CreatedAt: s.CreatedAt,
	}

	if s.ConnectedAt != nil {
		info.ConnectedAt = s.ConnectedAt
	}

	if s.Client != nil && s.Client.Store != nil && s.Client.Store.ID != nil {
		jid := s.Client.Store.ID.String()
		info.JID = &jid
	}

	return info
}

// SessionInfo representa informações básicas de uma sessão
type SessionInfo struct {
	ID                string     `json:"id"`
	Name              string     `json:"name"`
	APIKey            string     `json:"api_key,omitempty"`
	Status            Status     `json:"status"`
	JID               *string    `json:"jid,omitempty"`
	IsConnected       bool       `json:"is_connected"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	ConnectedAt       *time.Time `json:"connected_at,omitempty"`
	LastConnectedAt   *time.Time `json:"last_connected_at,omitempty"`
}

// QRCodeInfo representa informações do QR Code
type QRCodeInfo struct {
	Code      string    `json:"code"`
	Timeout   int       `json:"timeout"`
	Timestamp time.Time `json:"timestamp"`
}

// PairPhoneRequest representa uma solicitação de pareamento por telefone
type PairPhoneRequest struct {
	PhoneNumber string `json:"phone_number"`
}

// PairPhoneResponse representa a resposta do pareamento por telefone
type PairPhoneResponse struct {
	Success bool   `json:"success"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}
