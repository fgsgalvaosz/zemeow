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

// Manager gerencia múltiplas sessões WhatsApp de alto nível
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

// NewManager cria um novo gerenciador de sessões
func NewManager(container *sqlstore.Container, repository repositories.SessionRepository, config *config.Config) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	whatsappMgr := meow.NewWhatsAppManager(container, repository, config)

	// Criar cache com TTL de 1 hora e limpeza a cada 15 minutos
	cache := NewSessionCache(1*time.Hour, 15*time.Minute)

	// Criar lifecycle manager
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

// Start inicia o gerenciador de sessões
func (m *Manager) Start() error {
	m.logger.Info().Msg("Starting session manager")

	// Iniciar o lifecycle manager
	if err := m.lifecycle.Start(); err != nil {
		return fmt.Errorf("failed to start lifecycle manager: %w", err)
	}

	// Iniciar o WhatsApp manager
	if err := m.whatsappMgr.Start(); err != nil {
		return fmt.Errorf("failed to start WhatsApp manager: %w", err)
	}

	// Registrar handlers de lifecycle
	m.registerLifecycleHandlers()

	m.logger.Info().Msg("Session manager started successfully")
	return nil
}

// CreateSession cria uma nova sessão
func (m *Manager) CreateSession(sessionID string, name string, token string) (*models.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Verificar se já existe
	exists, err := m.repository.Exists(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to check if session exists: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("session %s already exists", sessionID)
	}

	// Criar modelo da sessão
	session := &models.Session{
		ID:        uuid.New(),
		SessionID: sessionID,
		Name:      name,
		Token:     token,
		Status:    models.SessionStatusDisconnected,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  make(models.Metadata),
	}

	// Salvar no banco
	if err := m.repository.Create(session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Adicionar ao cache
	m.cache.Set(sessionID, session)

	// Emitir evento de lifecycle
	m.lifecycle.EmitEvent(sessionID, EventSessionCreated, session)

	m.logger.Info().Str("session_id", sessionID).Msg("Session created successfully")
	return session, nil
}

// GetSession retorna uma sessão pelo ID
func (m *Manager) GetSession(sessionID string) (*models.Session, error) {
	// Tentar buscar no cache primeiro
	if session, found := m.cache.Get(sessionID); found {
		return session, nil
	}

	// Se não estiver no cache, buscar no banco
	session, err := m.repository.GetBySessionID(sessionID)
	if err != nil {
		return nil, err
	}

	// Adicionar ao cache
	m.cache.Set(sessionID, session)

	return session, nil
}

// ListSessions retorna todas as sessões
func (m *Manager) ListSessions(filter *models.SessionFilter) (*models.SessionListResponse, error) {
	return m.repository.GetAll(filter)
}

// DeleteSession remove uma sessão
func (m *Manager) DeleteSession(sessionID string) error {
	// Primeiro desconectar do WhatsApp se estiver conectado
	if err := m.whatsappMgr.DisconnectSession(sessionID); err != nil {
		m.logger.Warn().Err(err).Str("session_id", sessionID).Msg("Failed to disconnect session before deletion")
	}

	// Remover do banco de dados
	if err := m.repository.DeleteBySessionID(sessionID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	// Remover do cache
	m.cache.Delete(sessionID)

	m.logger.Info().Str("session_id", sessionID).Msg("Session deleted successfully")
	return nil
}

// ConnectSession conecta uma sessão ao WhatsApp
func (m *Manager) ConnectSession(ctx context.Context, sessionID string) error {
	_, err := m.whatsappMgr.ConnectSession(sessionID)
	if err != nil {
		return err
	}

	// Atualizar cache
	if session, err := m.repository.GetBySessionID(sessionID); err == nil {
		session.Status = models.SessionStatusConnecting
		m.cache.Set(sessionID, session)
	}

	return nil
}

// DisconnectSession desconecta uma sessão do WhatsApp
func (m *Manager) DisconnectSession(sessionID string) error {
	return m.whatsappMgr.DisconnectSession(sessionID)
}

// GetSessionStatus retorna o status de uma sessão
func (m *Manager) GetSessionStatus(sessionID string) (models.SessionStatus, error) {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return models.SessionStatusDisconnected, err
	}

	return session.Status, nil
}

// GetQRCode obtém o QR Code para uma sessão
func (m *Manager) GetQRCode(sessionID string) (*meow.QRCodeData, error) {
	// O QR Code é obtido através do ConnectSession
	return m.whatsappMgr.ConnectSession(sessionID)
}

// PairPhone realiza pareamento por telefone
func (m *Manager) PairPhone(sessionID string, phoneNumber string) error {
	// Por enquanto, retornar não implementado
	// TODO: Implementar pareamento por telefone quando o whatsmeow suportar
	return fmt.Errorf("phone pairing not implemented yet")
}

// SetProxy configura proxy para uma sessão
func (m *Manager) SetProxy(sessionID string, proxy *models.ProxyConfig) error {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.SetProxyConfig(proxy)
	return m.repository.Update(session)
}

// SetWebhook configura webhook para uma sessão
func (m *Manager) SetWebhook(sessionID string, webhook *models.WebhookConfig) error {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.SetWebhookConfig(webhook)
	return m.repository.Update(session)
}

// Shutdown desconecta todas as sessões
func (m *Manager) Shutdown(ctx context.Context) error {
	m.logger.Info().Msg("Shutting down session manager")

	// Parar o WhatsApp manager
	m.whatsappMgr.Stop()

	// Parar o lifecycle manager
	m.lifecycle.Stop()

	// Parar o cache
	m.cache.Stop()

	// Cancelar contexto
	m.cancel()

	m.logger.Info().Msg("Session manager shutdown completed")
	return nil
}

// GetCacheStats retorna estatísticas do cache
func (m *Manager) GetCacheStats() CacheStats {
	return m.cache.GetStats()
}

// RefreshSessionCache atualiza o TTL de uma sessão no cache
func (m *Manager) RefreshSessionCache(sessionID string) bool {
	return m.cache.Refresh(sessionID)
}

// ClearCache limpa todo o cache
func (m *Manager) ClearCache() {
	m.cache.Clear()
}

// GetLifecycleStats retorna estatísticas do lifecycle manager
func (m *Manager) GetLifecycleStats() LifecycleStats {
	return m.lifecycle.GetStats()
}

// EmitLifecycleEvent emite um evento de lifecycle
func (m *Manager) EmitLifecycleEvent(sessionID string, eventType LifecycleEventType, data interface{}) {
	m.lifecycle.EmitEvent(sessionID, eventType, data)
}

// registerLifecycleHandlers registra handlers para eventos de lifecycle
func (m *Manager) registerLifecycleHandlers() {
	// Handler para sessão criada
	m.lifecycle.RegisterHandler(EventSessionCreated, func(event LifecycleEvent) error {
		m.logger.Info().Str("session_id", event.SessionID).Msg("Session created lifecycle event")
		return nil
	})

	// Handler para sessão conectada
	m.lifecycle.RegisterHandler(EventSessionConnected, func(event LifecycleEvent) error {
		m.logger.Info().Str("session_id", event.SessionID).Msg("Session connected lifecycle event")

		// Atualizar cache
		m.cache.UpdateStatus(event.SessionID, models.SessionStatusConnected)

		return nil
	})

	// Handler para sessão desconectada
	m.lifecycle.RegisterHandler(EventSessionDisconnected, func(event LifecycleEvent) error {
		m.logger.Info().Str("session_id", event.SessionID).Msg("Session disconnected lifecycle event")

		// Atualizar cache
		m.cache.UpdateStatus(event.SessionID, models.SessionStatusDisconnected)

		return nil
	})

	// Handler para erro de sessão
	m.lifecycle.RegisterHandler(EventSessionError, func(event LifecycleEvent) error {
		m.logger.Error().Str("session_id", event.SessionID).Interface("data", event.Data).Msg("Session error lifecycle event")

		// Atualizar cache
		m.cache.UpdateStatus(event.SessionID, models.SessionStatusError)

		return nil
	})

	// Handler para sessão deletada
	m.lifecycle.RegisterHandler(EventSessionDeleted, func(event LifecycleEvent) error {
		m.logger.Info().Str("session_id", event.SessionID).Msg("Session deleted lifecycle event")

		// Remover do cache
		m.cache.Delete(event.SessionID)

		return nil
	})
}
