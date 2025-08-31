package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/felipe/zemeow/internal/config"
	"github.com/felipe/zemeow/internal/db/repositories"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/felipe/zemeow/internal/service/meow"
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
	SessionID    string                 `json:"session_id"`
	Event        string                 `json:"event"`
	Data         interface{}            `json:"data,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Retries      int                    `json:"-"`
	URL          string                 `json:"-"`
	NextRetryAt  time.Time              `json:"-"`
	LastError    string                 `json:"-"`
	CreatedAt    time.Time              `json:"-"`
}


type WebhookResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
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

// WebhookServiceStats mantém estatísticas thread-safe do serviço
type WebhookServiceStats struct {
	TotalSent    int64
	TotalSuccess int64
	TotalFailed  int64
	TotalRetries int64
}

// RetryStrategy define a estratégia de retry
type RetryStrategy struct {
	MaxRetries      int
	BaseDelay       time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	JitterEnabled   bool
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
		queue:      make(chan WebhookPayload, 10000), // Buffer grande para evitar bloqueios
		retryQueue: make(chan WebhookPayload, 5000),  // Buffer para retries
		workers:    5, // Número de workers para processar webhooks
		stats:      WebhookServiceStats{},
	}
}


func (s *WebhookService) Start() {
	s.logger.Info().Int("workers", s.workers).Msg("Starting webhook service")


	for i := 0; i < s.workers; i++ {
		go s.worker(i)
	}

	// Iniciar worker dedicado para retry
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

// SendWebhook envia um webhook manualmente
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
		s.logger.Debug().Str("session_id", event.SessionID).Str("event", event.Event).Msg("Event not enabled for webhook")
		return nil
	}
	

	payload := WebhookPayload{
		SessionID: event.SessionID,
		Event:     event.Event,
		Data:      event.Data,
		Timestamp: event.Timestamp,
		URL:       *session.WebhookURL,
		Metadata: map[string]interface{}{
			"session_name": session.Name,
			"jid":          session.JID,
		},
	}
	

	return s.SendWebhook(payload)
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

				// Implementar retry com backoff exponencial
				if payload.Retries < s.config.Webhook.RetryCount {
					payload.Retries++
					payload.LastError = err.Error()
					payload.NextRetryAt = s.calculateNextRetry(payload.Retries)
					atomic.AddInt64(&s.stats.TotalRetries, 1)

					// Enviar para fila de retry
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

// retryWorker processa webhooks que falharam e precisam ser reenviados
func (s *WebhookService) retryWorker() {
	s.logger.Debug().Msg("Starting retry worker")

	ticker := time.NewTicker(1 * time.Second) // Verificar retries a cada segundo
	defer ticker.Stop()

	var pendingRetries []WebhookPayload

	for {
		select {
		case payload := <-s.retryQueue:
			// Adicionar à lista de retries pendentes
			pendingRetries = append(pendingRetries, payload)
			s.logger.Debug().
				Str("session_id", payload.SessionID).
				Int("pending_retries", len(pendingRetries)).
				Msg("Added webhook to retry queue")

		case <-ticker.C:
			// Processar retries que estão prontos
			now := time.Now()
			var stillPending []WebhookPayload

			for _, payload := range pendingRetries {
				if now.After(payload.NextRetryAt) {
					// Hora de tentar novamente
					select {
					case s.queue <- payload:
						s.logger.Debug().
							Str("session_id", payload.SessionID).
							Int("retry", payload.Retries).
							Msg("Webhook moved from retry queue to main queue")
					default:
						// Se a fila principal está cheia, manter na fila de retry
						stillPending = append(stillPending, payload)
					}
				} else {
					// Ainda não é hora de tentar novamente
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

// calculateNextRetry calcula o próximo momento para retry com backoff exponencial
func (s *WebhookService) calculateNextRetry(retryCount int) time.Time {
	// Backoff exponencial: 2^retry * base_delay
	baseDelay := 2 * time.Second
	maxDelay := 60 * time.Second

	delay := time.Duration(math.Pow(2, float64(retryCount))) * baseDelay
	if delay > maxDelay {
		delay = maxDelay
	}

	// Adicionar jitter (±25% de variação)
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