package dto

import "time"

type ProxyConfig struct {
	Enabled  bool   `json:"enabled" example:"true"`
	Host     string `json:"host" example:"proxy.example.com"`
	Port     int    `json:"port" example:"8080"`
	Username string `json:"username,omitempty" example:"usuario_proxy"`
	Password string `json:"password,omitempty" example:"senha_proxy"`
	Type     string `json:"type" example:"http"`
}

type WebhookConfig struct {
	URL    string   `json:"url" example:"https://example.com/webhook"`
	Events []string `json:"events" example:"message,status"`
	Secret string   `json:"secret,omitempty" example:"meu_webhook_secret_123"`
}

type CreateSessionRequest struct {
	Name      string         `json:"name" validate:"required,min=1,max=100" example:"Minha Sessão WhatsApp"`
	SessionID string         `json:"session_id,omitempty" validate:"omitempty,alphanum,min=3,max=50" example:"sessao123"`
	APIKey    string         `json:"api_key,omitempty" validate:"omitempty,min=32" example:"zmw_1234567890abcdef1234567890abcdef"`
	Webhook   *WebhookConfig `json:"webhook,omitempty"`
	Proxy     *ProxyConfig   `json:"proxy,omitempty"`
}

type UpdateSessionRequest struct {
	Name    string `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Webhook string `json:"webhook,omitempty" validate:"omitempty,url"`
	Proxy   string `json:"proxy,omitempty" validate:"omitempty,url"`
	Events  string `json:"events,omitempty"`
}

type SessionResponse struct {
	ID        string     `json:"id"`
	SessionID string     `json:"session_id"`
	Name      string     `json:"name"`
	APIKey    string     `json:"api_key"`
	Status    string     `json:"status"`
	JID       string     `json:"jid,omitempty"`
	Webhook   string     `json:"webhook,omitempty"`
	Proxy     string     `json:"proxy,omitempty"`
	Events    string     `json:"events,omitempty"`
	QRCode    string     `json:"qr_code,omitempty"`
	Connected bool       `json:"connected"`
	LastSeen  *time.Time `json:"last_seen,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type SessionListResponse struct {
	Sessions []SessionResponse `json:"sessions"`
	Total    int               `json:"total"`
}

type SessionStatusResponse struct {
	SessionID    string     `json:"session_id"`
	Status       string     `json:"status"`
	Connected    bool       `json:"connected"`
	JID          string     `json:"jid,omitempty"`
	LastSeen     *time.Time `json:"last_seen,omitempty"`
	ConnectionAt *time.Time `json:"connection_at,omitempty"`
	BatteryLevel int        `json:"battery_level,omitempty"`
	IsCharging   bool       `json:"is_charging,omitempty"`
}

type SessionStatsResponse struct {
	SessionID        string     `json:"session_id"`
	MessagesSent     int64      `json:"messages_sent"`
	MessagesReceived int64      `json:"messages_received"`
	MessagesFailed   int64      `json:"messages_failed"`
	Uptime           int64      `json:"uptime_seconds"`
	LastActivity     *time.Time `json:"last_activity,omitempty"`
}

type QRCodeResponse struct {
	SessionID string `json:"session_id"`
	QRCode    string `json:"qr_code"`
	QRData    string `json:"qr_data"`
	ExpiresAt int64  `json:"expires_at"`
	Status    string `json:"status"`
}

type ProxyRequest struct {
	Enabled  bool   `json:"enabled"`
	Type     string `json:"type" validate:"omitempty,oneof=http https socks5"`
	Host     string `json:"host" validate:"required_if=Enabled true,omitempty,hostname|ip"`
	Port     int    `json:"port" validate:"required_if=Enabled true,omitempty,min=1,max=65535"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type ProxyResponse struct {
	SessionID string `json:"session_id"`
	Enabled   bool   `json:"enabled"`
	Type      string `json:"type,omitempty"`
	Host      string `json:"host,omitempty"`
	Port      int    `json:"port,omitempty"`
	Username  string `json:"username,omitempty"`
	Status    string `json:"status"`
}

type ProxyTestResponse struct {
	SessionID    string `json:"session_id"`
	Success      bool   `json:"success"`
	ResponseTime string `json:"response_time,omitempty"`
	IPAddress    string `json:"ip_address,omitempty"`
	Error        string `json:"error,omitempty"`
}
