package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// SessionStatus representa os possíveis status de uma sessão
type SessionStatus string

const (
	SessionStatusDisconnected SessionStatus = "disconnected"
	SessionStatusConnecting   SessionStatus = "connecting"
	SessionStatusConnected    SessionStatus = "connected"
	SessionStatusAuthenticated SessionStatus = "authenticated"
	SessionStatusError        SessionStatus = "error"
)

// Session representa uma sessão WhatsApp no banco de dados
type Session struct {
	ID              uuid.UUID      `json:"id" db:"id"`
	SessionID       string         `json:"session_id" db:"session_id"`
	Name            string         `json:"name" db:"name"`
	Token           string         `json:"token" db:"token"`
	JID             *string        `json:"jid,omitempty" db:"jid"`
	Status          SessionStatus  `json:"status" db:"status"`
	ProxyEnabled    bool           `json:"proxy_enabled" db:"proxy_enabled"`
	ProxyHost       *string        `json:"proxy_host,omitempty" db:"proxy_host"`
	ProxyPort       *int           `json:"proxy_port,omitempty" db:"proxy_port"`
	ProxyUsername   *string        `json:"proxy_username,omitempty" db:"proxy_username"`
	ProxyPassword   *string        `json:"proxy_password,omitempty" db:"proxy_password"`
	WebhookURL      *string        `json:"webhook_url,omitempty" db:"webhook_url"`
	WebhookEvents   pq.StringArray `json:"webhook_events" db:"webhook_events"`
	CreatedAt       time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at" db:"updated_at"`
	LastConnectedAt *time.Time     `json:"last_connected_at,omitempty" db:"last_connected_at"`
	Metadata        Metadata       `json:"metadata" db:"metadata"`
}

// SessionConfig representa a configuração de uma sessão
type SessionConfig struct {
	SessionID     string        `json:"session_id"`
	Name          string        `json:"name"`
	Proxy         *ProxyConfig  `json:"proxy,omitempty"`
	Webhook       *WebhookConfig `json:"webhook,omitempty"`
	AutoReconnect bool          `json:"auto_reconnect"`
	Timeout       time.Duration `json:"timeout"`
}

// ProxyConfig representa a configuração de proxy para uma sessão
type ProxyConfig struct {
	Enabled  bool   `json:"enabled"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Type     string `json:"type"` // http, socks5
}

// WebhookConfig representa a configuração de webhook para uma sessão
type WebhookConfig struct {
	URL    string   `json:"url"`
	Events []string `json:"events"`
	Secret string   `json:"secret,omitempty"`
}

// Metadata representa dados adicionais em formato JSON
type Metadata map[string]interface{}

// Value implementa driver.Valuer para Metadata
func (m Metadata) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

// Scan implementa sql.Scanner para Metadata
func (m *Metadata) Scan(value interface{}) error {
	if value == nil {
		*m = make(Metadata)
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, m)
	case string:
		return json.Unmarshal([]byte(v), m)
	default:
		return fmt.Errorf("cannot scan %T into Metadata", value)
	}
}

// CreateSessionRequest representa uma requisição para criar uma nova sessão
type CreateSessionRequest struct {
	SessionID string         `json:"session_id" validate:"required,min=3,max=255"`
	Name      string         `json:"name" validate:"required,min=1,max=255"`
	Proxy     *ProxyConfig   `json:"proxy,omitempty"`
	Webhook   *WebhookConfig `json:"webhook,omitempty"`
}

// UpdateSessionRequest representa uma requisição para atualizar uma sessão
type UpdateSessionRequest struct {
	Name      *string        `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Proxy     *ProxyConfig   `json:"proxy,omitempty"`
	Webhook   *WebhookConfig `json:"webhook,omitempty"`
	Metadata  *Metadata      `json:"metadata,omitempty"`
}

// SessionListResponse representa a resposta da listagem de sessões
type SessionListResponse struct {
	Sessions   []Session `json:"sessions"`
	Total      int       `json:"total"`
	Page       int       `json:"page"`
	PerPage    int       `json:"per_page"`
	TotalPages int       `json:"total_pages"`
}

// SessionInfoResponse representa informações detalhadas de uma sessão
type SessionInfoResponse struct {
	*Session
	IsConnected    bool               `json:"is_connected"`
	ConnectionInfo *ConnectionInfo    `json:"connection_info,omitempty"`
	Statistics     *SessionStatistics `json:"statistics,omitempty"`
}

// ConnectionInfo representa informações da conexão WhatsApp
type ConnectionInfo struct {
	JID           string    `json:"jid"`
	PushName      string    `json:"push_name"`
	BusinessName  string    `json:"business_name,omitempty"`
	ConnectedAt   time.Time `json:"connected_at"`
	LastSeen      time.Time `json:"last_seen"`
	BatteryLevel  int       `json:"battery_level,omitempty"`
	Plugged       bool      `json:"plugged"`
	Platform      string    `json:"platform"`
}

// SessionStatistics representa estatísticas de uma sessão
type SessionStatistics struct {
	MessagesReceived int `json:"messages_received"`
	MessagesSent     int `json:"messages_sent"`
	Uptime           int `json:"uptime_seconds"`
	Reconnections    int `json:"reconnections"`
	LastActivity     time.Time `json:"last_activity"`
}

// QRCodeResponse representa a resposta do QR Code
type QRCodeResponse struct {
	QRCode    string    `json:"qr_code"`
	Timeout   int       `json:"timeout_seconds"`
	ExpiresAt time.Time `json:"expires_at"`
}

// PairPhoneRequest representa uma requisição para pair por telefone
type PairPhoneRequest struct {
	PhoneNumber string `json:"phone_number" validate:"required,e164"`
}

// PairPhoneResponse representa a resposta do pair por telefone
type PairPhoneResponse struct {
	Code      string    `json:"code"`
	Timeout   int       `json:"timeout_seconds"`
	ExpiresAt time.Time `json:"expires_at"`
}

// SessionFilter representa filtros para listagem de sessões
type SessionFilter struct {
	Status    *SessionStatus `json:"status,omitempty"`
	Name      *string        `json:"name,omitempty"`
	JID       *string        `json:"jid,omitempty"`
	CreatedAt *time.Time     `json:"created_at,omitempty"`
	Page      int            `json:"page"`
	PerPage   int            `json:"per_page"`
	OrderBy   string         `json:"order_by"`
	OrderDir  string         `json:"order_dir"`
}

// Validate valida os dados da sessão
func (s *Session) Validate() error {
	if s.SessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	if s.Name == "" {
		return fmt.Errorf("name is required")
	}
	if s.Token == "" {
		return fmt.Errorf("token is required")
	}
	return nil
}

// IsConnected verifica se a sessão está conectada
func (s *Session) IsConnected() bool {
	return s.Status == SessionStatusConnected || s.Status == SessionStatusAuthenticated
}

// GetProxyConfig retorna a configuração de proxy
func (s *Session) GetProxyConfig() *ProxyConfig {
	if !s.ProxyEnabled || s.ProxyHost == nil || s.ProxyPort == nil {
		return nil
	}

	config := &ProxyConfig{
		Enabled: s.ProxyEnabled,
		Host:    *s.ProxyHost,
		Port:    *s.ProxyPort,
		Type:    "http", // default
	}

	if s.ProxyUsername != nil {
		config.Username = *s.ProxyUsername
	}
	if s.ProxyPassword != nil {
		config.Password = *s.ProxyPassword
	}

	return config
}

// GetWebhookConfig retorna a configuração de webhook
func (s *Session) GetWebhookConfig() *WebhookConfig {
	if s.WebhookURL == nil {
		return nil
	}

	return &WebhookConfig{
		URL:    *s.WebhookURL,
		Events: []string(s.WebhookEvents),
	}
}

// SetProxyConfig define a configuração de proxy
func (s *Session) SetProxyConfig(config *ProxyConfig) {
	if config == nil || !config.Enabled {
		s.ProxyEnabled = false
		s.ProxyHost = nil
		s.ProxyPort = nil
		s.ProxyUsername = nil
		s.ProxyPassword = nil
		return
	}

	s.ProxyEnabled = true
	s.ProxyHost = &config.Host
	s.ProxyPort = &config.Port

	if config.Username != "" {
		s.ProxyUsername = &config.Username
	}
	if config.Password != "" {
		s.ProxyPassword = &config.Password
	}
}

// SetWebhookConfig define a configuração de webhook
func (s *Session) SetWebhookConfig(config *WebhookConfig) {
	if config == nil || config.URL == "" {
		s.WebhookURL = nil
		s.WebhookEvents = nil
		return
	}

	s.WebhookURL = &config.URL
	s.WebhookEvents = pq.StringArray(config.Events)
}

// UpdateStatus atualiza o status da sessão
func (s *Session) UpdateStatus(status SessionStatus) {
	s.Status = status
	s.UpdatedAt = time.Now()

	if status == SessionStatusConnected || status == SessionStatusAuthenticated {
		now := time.Now()
		s.LastConnectedAt = &now
	}
}