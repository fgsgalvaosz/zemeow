package session

import (
	"context"
	"sync"
	"time"

	"github.com/felipe/zemeow/internal/db/models"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/rs/zerolog"
)

type LifecycleEvent struct {
	SessionID string
	Event     LifecycleEventType
	Data      interface{}
	Timestamp time.Time
}

type LifecycleEventType string

const (
	EventSessionCreated      LifecycleEventType = "session_created"
	EventSessionStarting     LifecycleEventType = "session_starting"
	EventSessionConnected    LifecycleEventType = "session_connected"
	EventSessionDisconnected LifecycleEventType = "session_disconnected"
	EventSessionError        LifecycleEventType = "session_error"
	EventSessionDeleted      LifecycleEventType = "session_deleted"
	EventSessionCleanup      LifecycleEventType = "session_cleanup"
)

type LifecycleManager struct {
	mu            sync.RWMutex
	eventChan     chan LifecycleEvent
	stopChan      chan struct{}
	handlers      map[LifecycleEventType][]LifecycleHandler
	sessionStates map[string]*SessionState
	logger        zerolog.Logger
	ctx           context.Context
	cancel        context.CancelFunc
}

type LifecycleHandler func(event LifecycleEvent) error

type SessionState struct {
	SessionID    string
	Status       models.SessionStatus
	StartedAt    time.Time
	LastActivity time.Time
	ErrorCount   int
	Metadata     map[string]interface{}
}

func NewLifecycleManager() *LifecycleManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &LifecycleManager{
		eventChan:     make(chan LifecycleEvent, 1000),
		stopChan:      make(chan struct{}),
		handlers:      make(map[LifecycleEventType][]LifecycleHandler),
		sessionStates: make(map[string]*SessionState),
		logger:        logger.Get().With().Str("component", "lifecycle_manager").Logger(),
		ctx:           ctx,
		cancel:        cancel,
	}
}

func (lm *LifecycleManager) Start() error {
	lm.logger.Info().Msg("Starting lifecycle manager")

	go lm.processEvents()

	go lm.periodicCleanup()

	lm.logger.Info().Msg("Lifecycle manager started")
	return nil
}

func (lm *LifecycleManager) Stop() {
	lm.logger.Info().Msg("Stopping lifecycle manager")

	lm.cancel()
	close(lm.stopChan)
	close(lm.eventChan)

	lm.logger.Info().Msg("Lifecycle manager stopped")
}

func (lm *LifecycleManager) EmitEvent(sessionID string, eventType LifecycleEventType, data interface{}) {
	event := LifecycleEvent{
		SessionID: sessionID,
		Event:     eventType,
		Data:      data,
		Timestamp: time.Now(),
	}

	select {
	case lm.eventChan <- event:
		lm.logger.Debug().Str("session_id", sessionID).Str("event", string(eventType)).Msg("Event emitted")
	default:
		lm.logger.Warn().Str("session_id", sessionID).Str("event", string(eventType)).Msg("Event channel full, dropping event")
	}
}

func (lm *LifecycleManager) RegisterHandler(eventType LifecycleEventType, handler LifecycleHandler) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if lm.handlers[eventType] == nil {
		lm.handlers[eventType] = make([]LifecycleHandler, 0)
	}

	lm.handlers[eventType] = append(lm.handlers[eventType], handler)
	lm.logger.Debug().Str("event_type", string(eventType)).Msg("Handler registered")
}

func (lm *LifecycleManager) GetSessionState(sessionID string) (*SessionState, bool) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	state, exists := lm.sessionStates[sessionID]
	return state, exists
}

func (lm *LifecycleManager) UpdateSessionState(sessionID string, status models.SessionStatus) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	state, exists := lm.sessionStates[sessionID]
	if !exists {
		state = &SessionState{
			SessionID: sessionID,
			StartedAt: time.Now(),
			Metadata:  make(map[string]interface{}),
		}
		lm.sessionStates[sessionID] = state
	}

	state.Status = status
	state.LastActivity = time.Now()

	lm.logger.Debug().Str("session_id", sessionID).Str("status", string(status)).Msg("Session state updated")
}

func (lm *LifecycleManager) RemoveSessionState(sessionID string) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	delete(lm.sessionStates, sessionID)
	lm.logger.Debug().Str("session_id", sessionID).Msg("Session state removed")
}

func (lm *LifecycleManager) processEvents() {
	lm.logger.Info().Msg("Starting event processor")

	for {
		select {
		case event, ok := <-lm.eventChan:
			if !ok {
				lm.logger.Info().Msg("Event channel closed, stopping processor")
				return
			}

			lm.handleEvent(event)

		case <-lm.ctx.Done():
			lm.logger.Info().Msg("Context cancelled, stopping event processor")
			return
		}
	}
}

func (lm *LifecycleManager) handleEvent(event LifecycleEvent) {
	lm.logger.Debug().
		Str("session_id", event.SessionID).
		Str("event", string(event.Event)).
		Msg("Processing lifecycle event")

	lm.updateStateFromEvent(event)

	lm.mu.RLock()
	handlers := lm.handlers[event.Event]
	lm.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(event); err != nil {
			lm.logger.Error().
				Err(err).
				Str("session_id", event.SessionID).
				Str("event", string(event.Event)).
				Msg("Handler failed to process event")
		}
	}
}

func (lm *LifecycleManager) updateStateFromEvent(event LifecycleEvent) {
	switch event.Event {
	case EventSessionCreated:
		lm.UpdateSessionState(event.SessionID, models.SessionStatusDisconnected)

	case EventSessionStarting:
		lm.UpdateSessionState(event.SessionID, models.SessionStatusConnecting)

	case EventSessionConnected:
		lm.UpdateSessionState(event.SessionID, models.SessionStatusConnected)

	case EventSessionDisconnected:
		lm.UpdateSessionState(event.SessionID, models.SessionStatusDisconnected)

	case EventSessionError:
		lm.UpdateSessionState(event.SessionID, models.SessionStatusError)
		lm.incrementErrorCount(event.SessionID)

	case EventSessionDeleted:
		lm.RemoveSessionState(event.SessionID)
	}
}

func (lm *LifecycleManager) incrementErrorCount(sessionID string) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if state, exists := lm.sessionStates[sessionID]; exists {
		state.ErrorCount++
	}
}

func (lm *LifecycleManager) periodicCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	lm.logger.Info().Msg("Starting periodic cleanup")

	for {
		select {
		case <-ticker.C:
			lm.cleanup()

		case <-lm.ctx.Done():
			lm.logger.Info().Msg("Context cancelled, stopping periodic cleanup")
			return
		}
	}
}

func (lm *LifecycleManager) cleanup() {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	now := time.Now()
	threshold := 1 * time.Hour

	toRemove := make([]string, 0)

	for sessionID, state := range lm.sessionStates {
		if now.Sub(state.LastActivity) > threshold {
			toRemove = append(toRemove, sessionID)
		}
	}

	for _, sessionID := range toRemove {
		delete(lm.sessionStates, sessionID)
		lm.logger.Debug().Str("session_id", sessionID).Msg("Cleaned up inactive session state")
	}

	if len(toRemove) > 0 {
		lm.logger.Info().Int("cleaned_count", len(toRemove)).Msg("Periodic cleanup completed")
	}
}

func (lm *LifecycleManager) GetStats() LifecycleStats {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	stats := LifecycleStats{
		TotalSessions: len(lm.sessionStates),
	}

	for _, state := range lm.sessionStates {
		switch state.Status {
		case models.SessionStatusConnected, models.SessionStatusAuthenticated:
			stats.ConnectedSessions++
		case models.SessionStatusDisconnected:
			stats.DisconnectedSessions++
		case models.SessionStatusConnecting:
			stats.ConnectingSessions++
		case models.SessionStatusError:
			stats.ErrorSessions++
		}

		if state.ErrorCount > 0 {
			stats.SessionsWithErrors++
		}
	}

	return stats
}

type LifecycleStats struct {
	TotalSessions        int `json:"total_sessions"`
	ConnectedSessions    int `json:"connected_sessions"`
	DisconnectedSessions int `json:"disconnected_sessions"`
	ConnectingSessions   int `json:"connecting_sessions"`
	ErrorSessions        int `json:"error_sessions"`
	SessionsWithErrors   int `json:"sessions_with_errors"`
}
