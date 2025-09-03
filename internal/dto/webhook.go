package dto

import "time"

type WebhookRequest struct {
	URL     string                 `json:"url" validate:"required,url"`
	Method  string                 `json:"method" validate:"oneof=POST PUT PATCH"`
	Headers map[string]string      `json:"headers,omitempty"`
	Payload map[string]interface{} `json:"payload" validate:"required"`
	Retry   *WebhookRetryConfig    `json:"retry,omitempty"`
}

type WebhookConfigRequest struct {
	URL         string   `json:"url" validate:"required,url" example:"https://example.com/webhook"`
	Events      []string `json:"events" validate:"required,min=1" example:"message,receipt,presence"`
	Active      bool     `json:"active" example:"true"`

}

type WebhookRetryConfig struct {
	MaxRetries    int     `json:"max_retries" validate:"min=0,max=10"`
	RetryInterval int     `json:"retry_interval" validate:"min=1,max=3600"`
	BackoffFactor float64 `json:"backoff_factor" validate:"min=1,max=10"`
}

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

type WebhookStatsResponse struct {
	TotalSent         int64           `json:"total_sent"`
	TotalSuccessful   int64           `json:"total_successful"`
	TotalFailed       int64           `json:"total_failed"`
	SuccessRate       float64         `json:"success_rate"`
	AverageLatency    int64           `json:"average_latency_ms"`
	Last24Hours       WebhookStats24h `json:"last_24_hours"`
	TopFailureReasons []FailureReason `json:"top_failure_reasons,omitempty"`
}

type WebhookStats24h struct {
	Sent        int64   `json:"sent"`
	Successful  int64   `json:"successful"`
	Failed      int64   `json:"failed"`
	SuccessRate float64 `json:"success_rate"`
}

type FailureReason struct {
	Reason string `json:"reason"`
	Count  int64  `json:"count"`
}

type SessionWebhookStatsResponse struct {
	SessionID   string     `json:"session_id"`
	TotalSent   int64      `json:"total_sent"`
	TotalFailed int64      `json:"total_failed"`
	SuccessRate float64    `json:"success_rate"`
	LastWebhook *time.Time `json:"last_webhook,omitempty"`
	WebhookURL  string     `json:"webhook_url,omitempty"`
	Events      []string   `json:"events,omitempty"`
}

type WebhookServiceStatusResponse struct {
	Running        bool       `json:"running"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	Uptime         int64      `json:"uptime_seconds,omitempty"`
	QueueSize      int        `json:"queue_size"`
	ProcessingRate float64    `json:"processing_rate_per_second"`
	Workers        int        `json:"active_workers"`
	LastProcessed  *time.Time `json:"last_processed,omitempty"`
}

type WebhookServiceControlRequest struct {
	Action string `json:"action" validate:"required,oneof=start stop restart"`
}

type WebhookServiceControlResponse struct {
	Action    string    `json:"action"`
	Success   bool      `json:"success"`
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type AuthValidationRequest struct {
	APIKey string `json:"api_key" validate:"required,min=32"`
}

type AuthValidationResponse struct {
	Valid     bool       `json:"valid"`
	Type      string     `json:"type"`
	SessionID string     `json:"session_id,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type AuthStatsResponse struct {
	TotalKeys    int     `json:"total_keys"`
	GlobalKeys   int     `json:"global_keys"`
	SessionKeys  int     `json:"session_keys"`
	CacheType    string  `json:"cache_type"`
	CacheHitRate float64 `json:"cache_hit_rate,omitempty"`
}

type HealthCheckResponse struct {
	Status      string                   `json:"status"`
	Version     string                   `json:"version"`
	Timestamp   time.Time                `json:"timestamp"`
	Uptime      int64                    `json:"uptime_seconds"`
	Environment string                   `json:"environment"`
	Services    map[string]ServiceHealth `json:"services"`
}

type ServiceHealth struct {
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
	Latency   int64  `json:"latency_ms,omitempty"`
	Available bool   `json:"available"`
}
