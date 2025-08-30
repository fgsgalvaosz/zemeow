package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
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
	workers    int
}


type WebhookPayload struct {
	SessionID string                 `json:"session_id"`
	Event     string                 `json:"event"`
	Data      interface{}            `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Retries   int                    `json:"-"`
	URL       string                 `json:"-"`
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
		workers:    5, // Número de workers para processar webhooks
	}
}


func (s *WebhookService) Start() {
	s.logger.Info().Int("workers", s.workers).Msg("Starting webhook service")
	

	for i := 0; i < s.workers; i++ {
		go s.worker(i)
	}
	
	s.logger.Info().Msg("Webhook service started")
}


func (s *WebhookService) Stop() {
	s.logger.Info().Msg("Stopping webhook service")
	
	s.cancel()
	close(s.queue)
	
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


func (s *WebhookService) SendWebhook(payload WebhookPayload) error {
	select {
	case s.queue <- payload:
		return nil
	default:
		return fmt.Errorf("webhook queue is full")
	}
}


func (s *WebhookService) GetStats() WebhookStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return WebhookStats{

		QueueSize:     len(s.queue),
		ActiveWorkers: s.workers,
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
			

			if err := s.sendHTTPWebhook(payload); err != nil {
				s.logger.Error().Err(err).Int("worker_id", id).Str("session_id", payload.SessionID).Msg("Failed to send webhook")
				

				if payload.Retries < s.config.Webhook.RetryCount {
					payload.Retries++
					

					delay := time.Duration(payload.Retries*payload.Retries) * time.Second
					time.Sleep(delay)
					

					select {
					case s.queue <- payload:
					default:
						s.logger.Warn().Str("session_id", payload.SessionID).Msg("Failed to requeue webhook, queue full")
					}
				}
			}
			
		case <-s.ctx.Done():
			s.logger.Debug().Int("worker_id", id).Msg("Webhook worker stopped by context")
			return
		}
	}
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