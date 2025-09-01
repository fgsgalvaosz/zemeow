package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type SessionStatus string

const (
	SessionStatusDisconnected  SessionStatus = "disconnected"
	SessionStatusConnecting    SessionStatus = "connecting"
	SessionStatusConnected     SessionStatus = "connected"
	SessionStatusAuthenticated SessionStatus = "authenticated"
	SessionStatusError         SessionStatus = "error"
)

type Session struct {
	ID               uuid.UUID      `json:"id" db:"id"`
	Name             string         `json:"name" db:"name"`
	APIKey           string         `json:"api_key" db:"api_key"`
	JID              *string        `json:"jid,omitempty" db:"jid"`
	Status           SessionStatus  `json:"status" db:"status"`
	ProxyEnabled     bool           `json:"proxy_enabled" db:"proxy_enabled"`
	ProxyHost        *string        `json:"proxy_host,omitempty" db:"proxy_host"`
	ProxyPort        *int           `json:"proxy_port,omitempty" db:"proxy_port"`
	ProxyUsername    *string        `json:"proxy_username,omitempty" db:"proxy_username"`
	ProxyPassword    *string        `json:"proxy_password,omitempty" db:"proxy_password"`
	WebhookURL       *string        `json:"webhook_url,omitempty" db:"webhook_url"`
	WebhookEvents    pq.StringArray `json:"webhook_events" db:"webhook_events"`
	WebhookPayloadMode *string      `json:"webhook_payload_mode,omitempty" db:"webhook_payload_mode"` // "processed" | "raw" | "both"`
	CreatedAt        time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at" db:"updated_at"`
	LastConnectedAt  *time.Time     `json:"last_connected_at,omitempty" db:"last_connected_at"`
	Metadata         Metadata       `json:"metadata" db:"metadata"`
	MessagesReceived int            `json:"messages_received" db:"messages_received"`
	MessagesSent     int            `json:"messages_sent" db:"messages_sent"`
	Reconnections    int            `json:"reconnections" db:"reconnections"`
	LastActivity     *time.Time     `json:"last_activity,omitempty" db:"last_activity"`
	QRCode           *string        `json:"qr_code,omitempty" db:"qrcode"`
}

func (s *Session) GetSessionID() string {
	return s.ID.String()
}

func (s *Session) GetIdentifier() string {
	return s.ID.String()
}

func (s *Session) IsValidName() bool {
	if len(s.Name) < 3 || len(s.Name) > 50 {
		return false
	}

	for _, char := range s.Name {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_' || char == '-') {
			return false
		}
	}
	return true
}

type SessionConfig struct {
	SessionID     string         `json:"session_id"`
	Name          string         `json:"name"`
	Proxy         *ProxyConfig   `json:"proxy,omitempty"`
	Webhook       *WebhookConfig `json:"webhook,omitempty"`
	AutoReconnect bool           `json:"auto_reconnect"`
	Timeout       time.Duration  `json:"timeout"`
}

type ProxyConfig struct {
	Enabled  bool   `json:"enabled"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Type     string `json:"type"`
}

type WebhookConfig struct {
	URL         string   `json:"url"`
	Events      []string `json:"events"`
	Secret      string   `json:"secret,omitempty"`
	PayloadMode string   `json:"payload_mode,omitempty"` // "processed" | "raw" | "both"
}

type Metadata map[string]interface{}

func (m Metadata) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

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

type CreateSessionRequest struct {
	SessionID string         `json:"session_id" validate:"omitempty,min=3,max=255"`
	Name      string         `json:"name" validate:"required,min=1,max=255"`
	APIKey    string         `json:"api_key,omitempty" validate:"omitempty,min=10,max=255"`
	Proxy     *ProxyConfig   `json:"proxy,omitempty"`
	Webhook   *WebhookConfig `json:"webhook,omitempty"`
}

type UpdateSessionRequest struct {
	Name     *string        `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Proxy    *ProxyConfig   `json:"proxy,omitempty"`
	Webhook  *WebhookConfig `json:"webhook,omitempty"`
	Metadata *Metadata      `json:"metadata,omitempty"`
}

type SessionListResponse struct {
	Sessions   []Session `json:"sessions"`
	Total      int       `json:"total"`
	Page       int       `json:"page"`
	PerPage    int       `json:"per_page"`
	TotalPages int       `json:"total_pages"`
}

type SessionInfoResponse struct {
	*Session
	IsConnected    bool               `json:"is_connected"`
	ConnectionInfo *ConnectionInfo    `json:"connection_info,omitempty"`
	Statistics     *SessionStatistics `json:"statistics,omitempty"`
}

type ConnectionInfo struct {
	JID          string    `json:"jid"`
	PushName     string    `json:"push_name"`
	BusinessName string    `json:"business_name,omitempty"`
	ConnectedAt  time.Time `json:"connected_at"`
	LastSeen     time.Time `json:"last_seen"`
	BatteryLevel int       `json:"battery_level,omitempty"`
	Plugged      bool      `json:"plugged"`
	Platform     string    `json:"platform"`
}

type SessionStatistics struct {
	MessagesReceived int       `json:"messages_received"`
	MessagesSent     int       `json:"messages_sent"`
	Uptime           int       `json:"uptime_seconds"`
	Reconnections    int       `json:"reconnections"`
	LastActivity     time.Time `json:"last_activity"`
}

type QRCodeResponse struct {
	QRCode    string    `json:"qr_code"`
	Timeout   int       `json:"timeout_seconds"`
	ExpiresAt time.Time `json:"expires_at"`
}

type PairPhoneRequest struct {
	PhoneNumber string `json:"phone_number" validate:"required,e164"`
}

type PairPhoneResponse struct {
	Code      string    `json:"code"`
	Timeout   int       `json:"timeout_seconds"`
	ExpiresAt time.Time `json:"expires_at"`
}

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

func (s *Session) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("name is required")
	}
	if !s.IsValidName() {
		return fmt.Errorf("name must be 3-50 characters and contain only letters, numbers, underscore and hyphen")
	}
	if s.APIKey == "" {
		return fmt.Errorf("api_key is required")
	}
	return nil
}

func (s *Session) IsConnected() bool {
	return s.Status == SessionStatusConnected || s.Status == SessionStatusAuthenticated
}

func (s *Session) GetProxyConfig() *ProxyConfig {
	if !s.ProxyEnabled || s.ProxyHost == nil || s.ProxyPort == nil {
		return nil
	}

	config := &ProxyConfig{
		Enabled: s.ProxyEnabled,
		Host:    *s.ProxyHost,
		Port:    *s.ProxyPort,
		Type:    "http",
	}

	if s.ProxyUsername != nil {
		config.Username = *s.ProxyUsername
	}
	if s.ProxyPassword != nil {
		config.Password = *s.ProxyPassword
	}

	return config
}

func (s *Session) GetWebhookConfig() *WebhookConfig {
	if s.WebhookURL == nil {
		return nil
	}

	return &WebhookConfig{
		URL:    *s.WebhookURL,
		Events: []string(s.WebhookEvents),
	}
}

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

func (s *Session) SetWebhookConfig(config *WebhookConfig) {
	if config == nil || config.URL == "" {
		s.WebhookURL = nil
		s.WebhookEvents = nil
		return
	}

	s.WebhookURL = &config.URL
	s.WebhookEvents = pq.StringArray(config.Events)
}

func (s *Session) UpdateStatus(status SessionStatus) {
	s.Status = status
	s.UpdatedAt = time.Now()

	if status == SessionStatusConnected || status == SessionStatusAuthenticated {
		now := time.Now()
		s.LastConnectedAt = &now
	}
}
