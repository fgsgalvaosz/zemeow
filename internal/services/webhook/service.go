package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/felipe/zemeow/internal/config"
	"github.com/felipe/zemeow/internal/models"
	"github.com/felipe/zemeow/internal/repositories"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/felipe/zemeow/internal/services/meow"
)

type WebhookService struct {
	mu         sync.RWMutex
	client     *http.Client
	repository repositories.SessionRepository
	config     *config.Config
	logger     logger.Logger
	ctx        context.Context
	cancel     context.CancelFunc
	queue      chan WebhookPayload
	retryQueue chan WebhookPayload
	workers    int
	stats      WebhookServiceStats
}

type WebhookPayload struct {
	SessionID   string                 `json:"session_id"`
	Event       string                 `json:"event"`
	Data        interface{}            `json:"data,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Retries     int                    `json:"-"`
	URL         string                 `json:"-"`
	NextRetryAt time.Time              `json:"-"`
	LastError   string                 `json:"-"`
	CreatedAt   time.Time              `json:"-"`
}

// RawWebhookPayload estrutura para payloads brutos da whatsmeow
type RawWebhookPayload struct {
	SessionID    string          `json:"session_id"`
	EventType    string          `json:"event_type"`
	RawData      json.RawMessage `json:"raw_data"`
	EventMeta    EventMetadata   `json:"event_meta"`
	Timestamp    time.Time       `json:"timestamp"`
	PayloadType  string          `json:"payload_type"` // "raw" | "processed"
	Retries      int             `json:"-"`
	URL          string          `json:"-"`
	NextRetryAt  time.Time       `json:"-"`
	LastError    string          `json:"-"`
	CreatedAt    time.Time       `json:"-"`
}

// EventMetadata metadados do evento para contexto adicional
type EventMetadata struct {
	WhatsmeowVersion string `json:"whatsmeow_version,omitempty"`
	ProtocolVersion  string `json:"protocol_version,omitempty"`
	SessionJID       string `json:"session_jid,omitempty"`
	DeviceInfo       string `json:"device_info,omitempty"`
	GoVersion        string `json:"go_version,omitempty"`
}

type WebhookResponse struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type WebhookStats struct {
	TotalSent     int64 `json:"total_sent"`
	TotalSuccess  int64 `json:"total_success"`
	TotalFailed   int64 `json:"total_failed"`
	TotalRetries  int64 `json:"total_retries"`
	QueueSize     int   `json:"queue_size"`
	ActiveWorkers int   `json:"active_workers"`
}

type WebhookServiceStats struct {
	TotalSent    int64
	TotalSuccess int64
	TotalFailed  int64
	TotalRetries int64
}

type RetryStrategy struct {
	MaxRetries    int
	BaseDelay     time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
	JitterEnabled bool
}

func NewWebhookService(repository repositories.SessionRepository, config *config.Config) *WebhookService {
	ctx, cancel := context.WithCancel(context.Background())

	return &WebhookService{
		client: &http.Client{
			Timeout: config.Webhook.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     30 * time.Second,
			},
		},
		repository: repository,
		config:     config,
		logger:     logger.GetWithSession("webhook_service"),
		ctx:        ctx,
		cancel:     cancel,
		queue:      make(chan WebhookPayload, 10000),
		retryQueue: make(chan WebhookPayload, 5000),
		workers:    5,
		stats:      WebhookServiceStats{},
	}
}

func (s *WebhookService) Start() {
	s.logger.Info().Int("workers", s.workers).Msg("Starting webhook service")

	for i := 0; i < s.workers; i++ {
		go s.worker(i)
	}

	go s.retryWorker()

	s.logger.Info().Msg("Webhook service started")
}

func (s *WebhookService) Stop() {
	s.logger.Info().Msg("Stopping webhook service")

	s.cancel()
	close(s.queue)
	close(s.retryQueue)

	s.logger.Info().Msg("Webhook service stopped")
}

func (s *WebhookService) ProcessEvents(eventChan <-chan meow.WebhookEvent) {
	s.logger.Info().Msg("Starting event processor")

	go func() {
		for {
			select {
			case event, ok := <-eventChan:
				if !ok {
					s.logger.Info().Msg("Event channel closed")
					return
				}

				if err := s.processEvent(event); err != nil {
					s.logger.Error().Err(err).Str("session_id", event.SessionID).Str("event", event.Event).Msg("Failed to process event")
				}

			case <-s.ctx.Done():
				s.logger.Info().Msg("Event processor stopped")
				return
			}
		}
	}()
}

func (s *WebhookService) GetStats() WebhookStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return WebhookStats{
		TotalSent:     atomic.LoadInt64(&s.stats.TotalSent),
		TotalSuccess:  atomic.LoadInt64(&s.stats.TotalSuccess),
		TotalFailed:   atomic.LoadInt64(&s.stats.TotalFailed),
		TotalRetries:  atomic.LoadInt64(&s.stats.TotalRetries),
		QueueSize:     len(s.queue),
		ActiveWorkers: s.workers,
	}
}

func (s *WebhookService) SendWebhook(payload WebhookPayload) error {
	select {
	case s.queue <- payload:
		s.logger.Debug().
			Str("session_id", payload.SessionID).
			Str("event", payload.Event).
			Msg("Webhook queued manually")
		return nil
	default:
		return fmt.Errorf("webhook queue is full")
	}
}

func (s *WebhookService) processEvent(event meow.WebhookEvent) error {
	session, err := s.repository.GetByIdentifier(event.SessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if session.WebhookURL == nil || *session.WebhookURL == "" {
		s.logger.Debug().Str("session_id", event.SessionID).Msg("No webhook URL configured")
		return nil
	}

	if !s.isEventEnabled(event.Event, session.WebhookEvents) {
		// Also check with the raw event type if available
		if event.EventType != "" && !s.isEventEnabled(event.EventType, session.WebhookEvents) {
			s.logger.Debug().Str("session_id", event.SessionID).Str("event", event.Event).Str("event_type", event.EventType).Msg("Event not enabled for webhook")
			return nil
		}
	}

	// Determinar modo de payload (padrão: processed)
	payloadMode := "processed"

	// Processar conforme o modo configurado
	switch payloadMode {
	case "processed":
		return s.processEventProcessed(event, session)
	case "raw":
		return s.processEventRaw(event, session)
	case "both":
		// Enviar ambos os formatos
		err1 := s.processEventProcessed(event, session)
		err2 := s.processEventRaw(event, session)
		if err1 != nil {
			return err1
		}
		return err2
	default:
		// Fallback para modo processado
		return s.processEventProcessed(event, session)
	}
}

func (s *WebhookService) worker(id int) {
	s.logger.Debug().Int("worker_id", id).Msg("Starting webhook worker")

	for {
		select {
		case payload, ok := <-s.queue:
			if !ok {
				s.logger.Debug().Int("worker_id", id).Msg("Webhook worker stopped")
				return
			}

			atomic.AddInt64(&s.stats.TotalSent, 1)

			if err := s.sendHTTPWebhook(payload); err != nil {
				atomic.AddInt64(&s.stats.TotalFailed, 1)
				s.logger.Error().Err(err).Int("worker_id", id).Str("session_id", payload.SessionID).Msg("Failed to send webhook")

				if payload.Retries < s.config.Webhook.RetryCount {
					payload.Retries++
					payload.LastError = err.Error()
					payload.NextRetryAt = s.calculateNextRetry(payload.Retries)
					atomic.AddInt64(&s.stats.TotalRetries, 1)

					select {
					case s.retryQueue <- payload:
						s.logger.Debug().
							Str("session_id", payload.SessionID).
							Int("retry", payload.Retries).
							Time("next_retry", payload.NextRetryAt).
							Msg("Webhook queued for retry")
					default:
						s.logger.Warn().Str("session_id", payload.SessionID).Msg("Failed to queue webhook for retry, retry queue full")
					}
				} else {
					s.logger.Error().
						Str("session_id", payload.SessionID).
						Str("event", payload.Event).
						Int("retries", payload.Retries).
						Msg("Webhook failed permanently after all retries")
				}
			} else {
				atomic.AddInt64(&s.stats.TotalSuccess, 1)
			}

		case <-s.ctx.Done():
			s.logger.Debug().Int("worker_id", id).Msg("Webhook worker stopped by context")
			return
		}
	}
}

func (s *WebhookService) retryWorker() {
	s.logger.Debug().Msg("Starting retry worker")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var pendingRetries []WebhookPayload

	for {
		select {
		case payload := <-s.retryQueue:

			pendingRetries = append(pendingRetries, payload)
			s.logger.Debug().
				Str("session_id", payload.SessionID).
				Int("pending_retries", len(pendingRetries)).
				Msg("Added webhook to retry queue")

		case <-ticker.C:

			now := time.Now()
			var stillPending []WebhookPayload

			for _, payload := range pendingRetries {
				if now.After(payload.NextRetryAt) {

					select {
					case s.queue <- payload:
						s.logger.Debug().
							Str("session_id", payload.SessionID).
							Int("retry", payload.Retries).
							Msg("Webhook moved from retry queue to main queue")
					default:

						stillPending = append(stillPending, payload)
					}
				} else {

					stillPending = append(stillPending, payload)
				}
			}

			pendingRetries = stillPending

		case <-s.ctx.Done():
			s.logger.Debug().Msg("Retry worker stopped by context")
			return
		}
	}
}

func (s *WebhookService) calculateNextRetry(retryCount int) time.Time {

	baseDelay := 2 * time.Second
	maxDelay := 60 * time.Second

	delay := time.Duration(math.Pow(2, float64(retryCount))) * baseDelay
	if delay > maxDelay {
		delay = maxDelay
	}

	jitter := time.Duration(float64(delay) * 0.25 * (2*rand.Float64() - 1))
	delay += jitter

	return time.Now().Add(delay)
}

func (s *WebhookService) sendHTTPWebhook(payload WebhookPayload) error {

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(s.ctx, "POST", payload.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ZeMeow-Webhook/1.0")
	req.Header.Set("X-Webhook-Event", payload.Event)
	req.Header.Set("X-Session-ID", payload.SessionID)

	// Add webhook secret if configured
	if s.config.Webhook.Secret != "" {
		req.Header.Set("X-Webhook-Secret", s.config.Webhook.Secret)
	}

	start := time.Now()
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	duration := time.Since(start)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	s.logger.Info().
		Str("session_id", payload.SessionID).
		Str("event", payload.Event).
		Str("url", payload.URL).
		Int("status", resp.StatusCode).
		Dur("duration", duration).
		Int("retries", payload.Retries).
		Msg("Webhook sent successfully")

	return nil
}

func (s *WebhookService) isEventEnabled(event string, enabledEvents []string) bool {
	if len(enabledEvents) == 0 {

		return true
	}

	for _, enabledEvent := range enabledEvents {
		if enabledEvent == "*" || enabledEvent == event {
			return true
		}
	}

	return false
}

func (s *WebhookService) TestWebhook(url string, sessionID string) error {
	testPayload := WebhookPayload{
		SessionID: sessionID,
		Event:     "test",
		Data: map[string]interface{}{
			"message": "This is a test webhook from ZeMeow",
		},
		Timestamp: time.Now(),
		URL:       url,
		Metadata: map[string]interface{}{
			"test": true,
		},
	}

	return s.sendHTTPWebhook(testPayload)
}

// processEventProcessed processa evento no formato processado (atual)
func (s *WebhookService) processEventProcessed(event meow.WebhookEvent, session *models.Session) error {
	payload := WebhookPayload{
		SessionID: event.SessionID,
		Event:     event.Event,
		Data:      event.Data,
		Timestamp: event.Timestamp,
		URL:       *session.WebhookURL,
		Metadata: map[string]interface{}{
			"session_name": session.Name,
			"jid":          session.JID,
			"payload_type": "processed",
		},
	}

	return s.SendWebhook(payload)
}

// processEventRaw processa evento no formato bruto da whatsmeow
func (s *WebhookService) processEventRaw(event meow.WebhookEvent, session *models.Session) error {
	// Verificar se há dados brutos disponíveis
	if event.RawEventData == nil {
		s.logger.Warn().Str("session_id", event.SessionID).Str("event", event.Event).Msg("No raw event data available")
		return nil
	}

	// Serializar dados brutos
	rawBytes, err := json.Marshal(event.RawEventData)
	if err != nil {
		return fmt.Errorf("failed to marshal raw event data: %w", err)
	}

	// Criar metadados do evento
	eventMeta := s.createEventMetadata(session)

	// Criar payload bruto
	rawPayload := RawWebhookPayload{
		SessionID:   event.SessionID,
		EventType:   event.EventType,
		RawData:     json.RawMessage(rawBytes),
		EventMeta:   eventMeta,
		Timestamp:   event.Timestamp,
		PayloadType: "raw",
		URL:         *session.WebhookURL,
	}

	return s.SendRawWebhook(rawPayload)
}

// createEventMetadata cria metadados do evento
func (s *WebhookService) createEventMetadata(session *models.Session) EventMetadata {
	var sessionJID string
	if session.JID != nil {
		sessionJID = *session.JID
	}

	return EventMetadata{
		WhatsmeowVersion: "v0.0.0-20250611130243", // Versão da whatsmeow
		ProtocolVersion:  "2.24.6",                 // Versão do protocolo
		SessionJID:       sessionJID,
		DeviceInfo:       "ZeMeow/1.0",
		GoVersion:        runtime.Version(),
	}
}

// SendRawWebhook envia webhook com payload bruto
func (s *WebhookService) SendRawWebhook(payload RawWebhookPayload) error {
	// Converter para interface comum para usar a fila existente
	genericPayload := WebhookPayload{
		SessionID: payload.SessionID,
		Event:     payload.EventType,
		Data:      payload, // O payload completo como data
		Timestamp: payload.Timestamp,
		URL:       payload.URL,
		Metadata: map[string]interface{}{
			"payload_type": "raw",
			"event_type":   payload.EventType,
		},
	}

	select {
	case s.queue <- genericPayload:
		s.logger.Debug().
			Str("session_id", payload.SessionID).
			Str("event_type", payload.EventType).
			Msg("Raw webhook queued")
		return nil
	default:
		return fmt.Errorf("webhook queue is full")
	}
}
