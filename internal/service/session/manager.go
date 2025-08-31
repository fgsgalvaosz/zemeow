package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/felipe/zemeow/internal/config"
	"github.com/felipe/zemeow/internal/db/models"
	"github.com/felipe/zemeow/internal/db/repositories"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/felipe/zemeow/internal/service/meow"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"go.mau.fi/whatsmeow/store/sqlstore"
)

type Manager struct {
	mu          sync.RWMutex
	whatsappMgr *meow.WhatsAppManager
	repository  repositories.SessionRepository
	cache       *SessionCache
	lifecycle   *LifecycleManager
	config      *config.Config
	logger      zerolog.Logger
	ctx         context.Context
	cancel      context.CancelFunc
}

func NewManager(container *sqlstore.Container, repository repositories.SessionRepository, config *config.Config) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	whatsappMgr := meow.NewWhatsAppManager(container, repository, config)

	cache := NewSessionCache(1*time.Hour, 15*time.Minute)

	lifecycle := NewLifecycleManager()

	return &Manager{
		whatsappMgr: whatsappMgr,
		repository:  repository,
		cache:       cache,
		lifecycle:   lifecycle,
		config:      config,
		logger:      logger.Get().With().Str("component", "session-manager").Logger(),
		ctx:         ctx,
		cancel:      cancel,
	}
}

func (m *Manager) Start() error {
	m.logger.Info().Msg("Starting session manager")

	if err := m.lifecycle.Start(); err != nil {
		return fmt.Errorf("failed to start lifecycle manager: %w", err)
	}

	if err := m.whatsappMgr.Start(); err != nil {
		return fmt.Errorf("failed to start WhatsApp manager: %w", err)
	}

	m.registerLifecycleHandlers()

	m.logger.Info().Msg("Session manager started successfully")
	return nil
}

func (m *Manager) CreateSession(sessionID string, name string, token string) (*models.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	exists, err := m.repository.Exists(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to check if session exists: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("session %s already exists", sessionID)
	}

	sessionUUID := uuid.New()
	session := &models.Session{
		ID:        sessionUUID,
		Name:      name, // nome fornecido pelo usuário
		Status:    models.SessionStatusDisconnected,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  make(models.Metadata),
	}

	if err := m.repository.Create(session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	m.cache.Set(sessionID, session)

	m.lifecycle.EmitEvent(sessionID, EventSessionCreated, session)

	m.logger.Info().Str("session_id", sessionID).Msg("Session created successfully")
	return session, nil
}

func (m *Manager) GetSession(sessionID string) (*models.Session, error) {

	if session, found := m.cache.Get(sessionID); found {
		return session, nil
	}

	session, err := m.repository.GetByIdentifier(sessionID)
	if err != nil {
		return nil, err
	}

	m.cache.Set(sessionID, session)

	return session, nil
}

func (m *Manager) ListSessions(filter *models.SessionFilter) (*models.SessionListResponse, error) {
	return m.repository.GetAll(filter)
}

func (m *Manager) DeleteSession(sessionID string) error {

	if err := m.whatsappMgr.DisconnectSession(sessionID); err != nil {
		m.logger.Warn().Err(err).Str("session_id", sessionID).Msg("Failed to disconnect session before deletion")
	}

	if err := m.repository.DeleteByIdentifier(sessionID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	m.cache.Delete(sessionID)

	m.logger.Info().Str("session_id", sessionID).Msg("Session deleted successfully")
	return nil
}

func (m *Manager) ConnectSession(ctx context.Context, sessionID string) error {
	_, err := m.whatsappMgr.ConnectSession(sessionID)
	if err != nil {
		return err
	}

	if session, err := m.repository.GetByIdentifier(sessionID); err == nil {
		session.Status = models.SessionStatusConnecting
		m.cache.Set(sessionID, session)
	}

	return nil
}

func (m *Manager) DisconnectSession(sessionID string) error {
	return m.whatsappMgr.DisconnectSession(sessionID)
}

func (m *Manager) GetSessionStatus(sessionID string) (models.SessionStatus, error) {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return models.SessionStatusDisconnected, err
	}

	return session.Status, nil
}

func (m *Manager) GetQRCode(sessionID string) (interface{}, error) {
	// Multi-device architecture: Connect session which will:
	// 1. Check if device exists in container (GetFirstDevice)
	// 2. If exists: reconnect directly (no QR needed)
	// 3. If not exists: create new device (NewDevice) and generate QR
	qrData, err := m.whatsappMgr.ConnectSession(sessionID)
	if err != nil {
		return nil, err
	}

	return &QRCodeInfo{
		Code:      qrData.Code,
		Timeout:   qrData.Timeout,
		Timestamp: qrData.Timestamp,
	}, nil
}

func (m *Manager) GetClient(sessionID string) (interface{}, error) {
	return m.whatsappMgr.GetClient(sessionID)
}

func (m *Manager) PairPhone(sessionID string, phoneNumber string) error {

	return fmt.Errorf("phone pairing not implemented yet")
}

func (m *Manager) SetProxy(sessionID string, proxy *models.ProxyConfig) error {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.SetProxyConfig(proxy)
	return m.repository.Update(session)
}

func (m *Manager) SetWebhook(sessionID string, webhook *models.WebhookConfig) error {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.SetWebhookConfig(webhook)
	return m.repository.Update(session)
}

func (m *Manager) Shutdown(ctx context.Context) error {
	m.logger.Info().Msg("Shutting down session manager")

	m.whatsappMgr.Stop()

	m.lifecycle.Stop()

	m.cache.Stop()

	m.cancel()

	m.logger.Info().Msg("Session manager shutdown completed")
	return nil
}

func (m *Manager) GetCacheStats() CacheStats {
	return m.cache.GetStats()
}

func (m *Manager) RefreshSessionCache(sessionID string) bool {
	return m.cache.Refresh(sessionID)
}

func (m *Manager) ClearCache() {
	m.cache.Clear()
}


func (m *Manager) InitializeNewSession(session *models.Session) error {
	return m.whatsappMgr.InitializeSession(session)
}

func (m *Manager) GetLifecycleStats() LifecycleStats {
	return m.lifecycle.GetStats()
}

func (m *Manager) EmitLifecycleEvent(sessionID string, eventType LifecycleEventType, data interface{}) {
	m.lifecycle.EmitEvent(sessionID, eventType, data)
}

func (m *Manager) registerLifecycleHandlers() {

	m.lifecycle.RegisterHandler(EventSessionCreated, func(event LifecycleEvent) error {
		m.logger.Info().Str("session_id", event.SessionID).Msg("Session created lifecycle event")
		return nil
	})

	m.lifecycle.RegisterHandler(EventSessionConnected, func(event LifecycleEvent) error {
		m.logger.Info().Str("session_id", event.SessionID).Msg("Session connected lifecycle event")

		m.cache.UpdateStatus(event.SessionID, models.SessionStatusConnected)

		return nil
	})

	m.lifecycle.RegisterHandler(EventSessionDisconnected, func(event LifecycleEvent) error {
		m.logger.Info().Str("session_id", event.SessionID).Msg("Session disconnected lifecycle event")

		m.cache.UpdateStatus(event.SessionID, models.SessionStatusDisconnected)

		return nil
	})

	m.lifecycle.RegisterHandler(EventSessionError, func(event LifecycleEvent) error {
		m.logger.Error().Str("session_id", event.SessionID).Interface("data", event.Data).Msg("Session error lifecycle event")

		m.cache.UpdateStatus(event.SessionID, models.SessionStatusError)

		return nil
	})

	m.lifecycle.RegisterHandler(EventSessionDeleted, func(event LifecycleEvent) error {
		m.logger.Info().Str("session_id", event.SessionID).Msg("Session deleted lifecycle event")

		m.cache.Delete(event.SessionID)

		return nil
	})
}
