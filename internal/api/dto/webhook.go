package dto

import "time"

// === WEBHOOK DTOs ===

// WebhookRequest requisição manual de webhook
type WebhookRequest struct {
	URL     string                 `json:"url" validate:"required,url"`
	Method  string                 `json:"method" validate:"oneof=POST PUT PATCH"`
	Headers map[string]string      `json:"headers,omitempty"`
	Payload map[string]interface{} `json:"payload" validate:"required"`
	Retry   *WebhookRetryConfig    `json:"retry,omitempty"`
}

// WebhookRetryConfig configuração de retry para webhook
type WebhookRetryConfig struct {
	MaxRetries    int   `json:"max_retries" validate:"min=0,max=10"`
	RetryInterval int   `json:"retry_interval" validate:"min=1,max=3600"` // seconds
	BackoffFactor float64 `json:"backoff_factor" validate:"min=1,max=10"`
}

// WebhookResponse resposta do webhook
type WebhookResponse struct {
	ID          string            `json:"id"`
	URL         string            `json:"url"`
	Method      string            `json:"method"`
	StatusCode  int               `json:"status_code"`
	Response    string            `json:"response,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Duration    int64             `json:"duration_ms"`
	Success     bool              `json:"success"`
	Error       string            `json:"error,omitempty"`
	Attempts    int               `json:"attempts"`
	SentAt      time.Time         `json:"sent_at"`
	CompletedAt time.Time         `json:"completed_at"`
}

// WebhookStatsResponse estatísticas de webhooks
type WebhookStatsResponse struct {
	TotalSent         int64   `json:"total_sent"`
	TotalSuccessful   int64   `json:"total_successful"`
	TotalFailed       int64   `json:"total_failed"`
	SuccessRate       float64 `json:"success_rate"`
	AverageLatency    int64   `json:"average_latency_ms"`
	Last24Hours       WebhookStats24h `json:"last_24_hours"`
	TopFailureReasons []FailureReason `json:"top_failure_reasons,omitempty"`
}

// WebhookStats24h estatísticas das últimas 24 horas
type WebhookStats24h struct {
	Sent       int64   `json:"sent"`
	Successful int64   `json:"successful"`
	Failed     int64   `json:"failed"`
	SuccessRate float64 `json:"success_rate"`
}

// FailureReason razões de falha mais comuns
type FailureReason struct {
	Reason string `json:"reason"`
	Count  int64  `json:"count"`
}

// SessionWebhookStatsResponse estatísticas de webhook por sessão
type SessionWebhookStatsResponse struct {
	SessionID    string      `json:"session_id"`
	TotalSent    int64       `json:"total_sent"`
	TotalFailed  int64       `json:"total_failed"`
	SuccessRate  float64     `json:"success_rate"`
	LastWebhook  *time.Time  `json:"last_webhook,omitempty"`
	WebhookURL   string      `json:"webhook_url,omitempty"`
	Events       []string    `json:"events,omitempty"`
}

// WebhookServiceStatusResponse status do serviço de webhook
type WebhookServiceStatusResponse struct {
	Running        bool      `json:"running"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	Uptime         int64     `json:"uptime_seconds,omitempty"`
	QueueSize      int       `json:"queue_size"`
	ProcessingRate float64   `json:"processing_rate_per_second"`
	Workers        int       `json:"active_workers"`
	LastProcessed  *time.Time `json:"last_processed,omitempty"`
}

// WebhookServiceControlRequest requisição para controlar serviço
type WebhookServiceControlRequest struct {
	Action string `json:"action" validate:"required,oneof=start stop restart"`
}

// WebhookServiceControlResponse resposta do controle do serviço
type WebhookServiceControlResponse struct {
	Action    string    `json:"action"`
	Success   bool      `json:"success"`
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// === AUTH DTOs ===

// AuthValidationRequest requisição de validação de API key
type AuthValidationRequest struct {
	APIKey string `json:"api_key" validate:"required,min=32"`
}

// AuthValidationResponse resposta da validação
type AuthValidationResponse struct {
	Valid     bool   `json:"valid"`
	Type      string `json:"type"` // global, session
	SessionID string `json:"session_id,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// AuthStatsResponse estatísticas de autenticação
type AuthStatsResponse struct {
	TotalKeys       int `json:"total_keys"`
	GlobalKeys      int `json:"global_keys"`
	SessionKeys     int `json:"session_keys"`
	CacheType       string `json:"cache_type"`
	CacheHitRate    float64 `json:"cache_hit_rate,omitempty"`
}

// === HEALTH DTOs ===

// HealthCheckResponse resposta do health check
type HealthCheckResponse struct {
	Status      string    `json:"status"`
	Version     string    `json:"version"`
	Timestamp   time.Time `json:"timestamp"`
	Uptime      int64     `json:"uptime_seconds"`
	Environment string    `json:"environment"`
	Services    map[string]ServiceHealth `json:"services"`
}

// ServiceHealth status de um serviço específico
type ServiceHealth struct {
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
	Latency   int64  `json:"latency_ms,omitempty"`
	Available bool   `json:"available"`
}