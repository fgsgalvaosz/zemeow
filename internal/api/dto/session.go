package dto

import "time"

// === SESSION DTOs ===

// CreateSessionRequest requisição para criar uma nova sessão
type CreateSessionRequest struct {
	Name      string `json:"name" validate:"required,min=1,max=100"`
	SessionID string `json:"session_id,omitempty" validate:"omitempty,alphanum,min=3,max=50"`
	APIKey    string `json:"api_key,omitempty" validate:"omitempty,min=32"`
	Webhook   string `json:"webhook,omitempty" validate:"omitempty,url"`
	Proxy     string `json:"proxy,omitempty" validate:"omitempty,url"`
	Events    string `json:"events,omitempty"`
}

// UpdateSessionRequest requisição para atualizar uma sessão
type UpdateSessionRequest struct {
	Name    string `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Webhook string `json:"webhook,omitempty" validate:"omitempty,url"`
	Proxy   string `json:"proxy,omitempty" validate:"omitempty,url"`
	Events  string `json:"events,omitempty"`
}

// SessionResponse resposta com dados da sessão
type SessionResponse struct {
	ID          string    `json:"id"`
	SessionID   string    `json:"session_id"`
	Name        string    `json:"name"`
	APIKey      string    `json:"api_key"`
	Status      string    `json:"status"`
	JID         string    `json:"jid,omitempty"`
	Webhook     string    `json:"webhook,omitempty"`
	Proxy       string    `json:"proxy,omitempty"`
	Events      string    `json:"events,omitempty"`
	QRCode      string    `json:"qr_code,omitempty"`
	Connected   bool      `json:"connected"`
	LastSeen    *time.Time `json:"last_seen,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SessionListResponse resposta com lista de sessões
type SessionListResponse struct {
	Sessions []SessionResponse `json:"sessions"`
	Total    int               `json:"total"`
}

// SessionStatusResponse resposta com status da sessão
type SessionStatusResponse struct {
	SessionID     string    `json:"session_id"`
	Status        string    `json:"status"`
	Connected     bool      `json:"connected"`
	JID           string    `json:"jid,omitempty"`
	LastSeen      *time.Time `json:"last_seen,omitempty"`
	ConnectionAt  *time.Time `json:"connection_at,omitempty"`
	BatteryLevel  int       `json:"battery_level,omitempty"`
	IsCharging    bool      `json:"is_charging,omitempty"`
}

// SessionStatsResponse resposta com estatísticas da sessão
type SessionStatsResponse struct {
	SessionID        string `json:"session_id"`
	MessagesSent     int64  `json:"messages_sent"`
	MessagesReceived int64  `json:"messages_received"`
	MessagesFailed   int64  `json:"messages_failed"`
	Uptime           int64  `json:"uptime_seconds"`
	LastActivity     *time.Time `json:"last_activity,omitempty"`
}

// QRCodeResponse resposta com QR code
type QRCodeResponse struct {
	SessionID string `json:"session_id"`
	QRCode    string `json:"qr_code"`
	QRData    string `json:"qr_data"`
	ExpiresAt int64  `json:"expires_at"`
	Status    string `json:"status"`
}

// PairPhoneRequest requisição para pareamento por telefone
type PairPhoneRequest struct {
	PhoneNumber string `json:"phone_number" validate:"required,e164"`
}

// PairPhoneResponse resposta do pareamento por telefone
type PairPhoneResponse struct {
	SessionID    string    `json:"session_id"`
	PhoneNumber  string    `json:"phone_number"`
	PairingCode  string    `json:"pairing_code"`
	ExpiresAt    int64     `json:"expires_at"`
	InitiatedAt  time.Time `json:"initiated_at"`
}

// ProxyRequest requisição para configurar proxy
type ProxyRequest struct {
	Enabled  bool   `json:"enabled"`
	Type     string `json:"type" validate:"omitempty,oneof=http https socks5"`
	Host     string `json:"host" validate:"required_if=Enabled true,omitempty,hostname|ip"`
	Port     int    `json:"port" validate:"required_if=Enabled true,omitempty,min=1,max=65535"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// ProxyResponse resposta da configuração de proxy
type ProxyResponse struct {
	SessionID string `json:"session_id"`
	Enabled   bool   `json:"enabled"`
	Type      string `json:"type,omitempty"`
	Host      string `json:"host,omitempty"`
	Port      int    `json:"port,omitempty"`
	Username  string `json:"username,omitempty"`
	Status    string `json:"status"`
}

// ProxyTestResponse resposta do teste de proxy
type ProxyTestResponse struct {
	SessionID    string `json:"session_id"`
	Success      bool   `json:"success"`
	ResponseTime string `json:"response_time,omitempty"`
	IPAddress    string `json:"ip_address,omitempty"`
	Error        string `json:"error,omitempty"`
}