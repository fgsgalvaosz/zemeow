package meow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/felipe/zemeow/internal/config"
	"github.com/felipe/zemeow/internal/db/models"
	"github.com/felipe/zemeow/internal/db/repositories"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
)

// WhatsAppManager gerencia múltiplas sessões WhatsApp de baixo nível
type WhatsAppManager struct {
	mu          sync.RWMutex
	clients     map[string]*MyClient
	sessions    map[string]*models.Session
	container   *sqlstore.Container
	repository  repositories.SessionRepository
	config      *config.Config
	logger      logger.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	webhookChan chan WebhookEvent
}

// NewWhatsAppManager cria um novo gerenciador de sessões WhatsApp
func NewWhatsAppManager(container *sqlstore.Container, repository repositories.SessionRepository, config *config.Config) *WhatsAppManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &WhatsAppManager{
		clients:     make(map[string]*MyClient),
		sessions:    make(map[string]*models.Session),
		container:   container,
		repository:  repository,
		config:      config,
		logger:      logger.GetWithSession("whatsapp_manager"),
		ctx:         ctx,
		cancel:      cancel,
		webhookChan: make(chan WebhookEvent, 100),
	}
}

// Start inicia o gerenciador de sessões WhatsApp
func (m *WhatsAppManager) Start() error {
	m.logger.Info().Msg("Starting WhatsApp manager")

	// Carregar sessões existentes do banco
	sessions, err := m.repository.GetAll(nil)
	if err != nil {
		return fmt.Errorf("failed to load sessions: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Inicializar sessões
	for _, session := range sessions.Sessions {
		if err := m.initializeSession(&session); err != nil {
			m.logger.Warn().Err(err).Str("session_id", session.SessionID).Msg("Failed to initialize session")
			continue
		}
		m.logger.Info().Str("session_id", session.SessionID).Msg("Session initialized")
	}

	m.logger.Info().Int("session_count", len(m.sessions)).Msg("WhatsApp manager started successfully")
	return nil
}

// Stop para o gerenciador de sessões WhatsApp
func (m *WhatsAppManager) Stop() {
	m.logger.Info().Msg("Stopping WhatsApp manager")

	// Cancelar contexto
	m.cancel()

	// Desconectar todos os clientes
	m.mu.Lock()
	defer m.mu.Unlock()

	for sessionID, client := range m.clients {
		m.logger.Info().Str("session_id", sessionID).Msg("Disconnecting client")
		client.Disconnect()
	}

	m.logger.Info().Msg("WhatsApp manager stopped")
}

// initializeSession inicializa uma sessão a partir do banco de dados
func (m *WhatsAppManager) initializeSession(session *models.Session) error {
	// Obter device store
	deviceStore, err := m.container.GetFirstDevice()
	if err != nil {
		// Criar novo device se não existir
		deviceStore = m.container.NewDevice()
	}

	// Criar cliente WhatsApp
	client := NewMyClient(session.SessionID, deviceStore, m.webhookChan)

	// Registrar handlers
	m.registerClientHandlers(client, session.SessionID)

	// Armazenar na memória
	m.sessions[session.SessionID] = session
	m.clients[session.SessionID] = client

	return nil
}

// CreateSession cria uma nova sessão WhatsApp
func (m *WhatsAppManager) CreateSession(sessionID string, config *models.SessionConfig) (*models.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Verificar se a sessão já existe
	if _, exists := m.clients[sessionID]; exists {
		return nil, fmt.Errorf("session already exists")
	}

	// Criar nova sessão no banco
	session := &models.Session{
		ID:        uuid.New(),
		SessionID: sessionID,
		Name:      config.Name,
		// Token:     uuid.New().String(),  // Removendo a referência ao token
		Status:    models.SessionStatusDisconnected,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Configurar proxy se fornecido
	if config.Proxy != nil {
		session.ProxyEnabled = true
		session.ProxyHost = &config.Proxy.Host
		session.ProxyPort = &config.Proxy.Port
		session.ProxyUsername = &config.Proxy.Username
		session.ProxyPassword = &config.Proxy.Password
	}

	// Configurar webhook se fornecido
	if config.Webhook != nil {
		session.WebhookURL = &config.Webhook.URL
		session.WebhookEvents = pq.StringArray(config.Webhook.Events)
	}

	// Salvar no banco
	if err := m.repository.Create(session); err != nil {
		return nil, fmt.Errorf("failed to create session in database: %w", err)
	}

	// Inicializar sessão
	if err := m.initializeSession(session); err != nil {
		m.repository.Delete(session.ID) // Limpar se falhar
		return nil, fmt.Errorf("failed to initialize session: %w", err)
	}

	m.logger.Info().Str("session_id", sessionID).Msg("Session created successfully")
	return session, nil
}

// GetSession retorna uma sessão pelo ID
func (m *WhatsAppManager) GetSession(sessionID string) (*models.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	return session, nil
}

// GetClient retorna o cliente WhatsApp de uma sessão
func (m *WhatsAppManager) GetClient(sessionID string) (*MyClient, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.clients[sessionID]
	if !exists {
		return nil, fmt.Errorf("client not found")
	}

	return client, nil
}

// ConnectSession conecta uma sessão específica
func (m *WhatsAppManager) ConnectSession(sessionID string) (*QRCodeData, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, exists := m.clients[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	// Verificar se já está conectado
	if client.IsConnected() {
		return nil, fmt.Errorf("session already connected")
	}

	// Atualizar status para connecting
	m.repository.UpdateStatus(sessionID, models.SessionStatusConnecting)

	// Canal para receber QR code
	qrChan := make(chan string, 1)
	var qrEventID uint32

	// Registrar handler para QR code
	qrEventID = client.client.AddEventHandler(func(evt interface{}) {
		if qr, ok := evt.(*events.QR); ok {
			select {
			case qrChan <- qr.Codes[0]:
			default:
			}
		}
	})

	// Conectar
	err := client.Connect()
	if err != nil {
		client.client.RemoveEventHandler(qrEventID)
		m.repository.UpdateStatus(sessionID, models.SessionStatusDisconnected)
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	// Aguardar QR code ou timeout
	select {
	case qrCode := <-qrChan:
		client.RemoveEventHandler(qrEventID)
		return &QRCodeData{
			Code:      qrCode,
			Timeout:   int(m.config.WhatsApp.QRCodeTimeout.Seconds()),
			Timestamp: time.Now(),
		}, nil
	case <-time.After(m.config.WhatsApp.QRCodeTimeout):
		client.RemoveEventHandler(qrEventID)
		client.Disconnect()
		m.repository.UpdateStatus(sessionID, models.SessionStatusDisconnected)
		return nil, fmt.Errorf("QR code timeout")
	}
}

// DisconnectSession desconecta uma sessão específica
func (m *WhatsAppManager) DisconnectSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, exists := m.clients[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}

	client.Disconnect()
	m.repository.UpdateStatus(sessionID, models.SessionStatusDisconnected)

	m.logger.Info().Str("session_id", sessionID).Msg("Session disconnected")
	return nil
}

// DeleteSession remove uma sessão completamente
func (m *WhatsAppManager) DeleteSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Desconectar se estiver conectado
	if client, exists := m.clients[sessionID]; exists {
		client.Disconnect()
		delete(m.clients, sessionID)
	}

	// Remover da memória
	delete(m.sessions, sessionID)

	// Remover do banco
	err := m.repository.DeleteBySessionID(sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session from database: %w", err)
	}

	m.logger.Info().Str("session_id", sessionID).Msg("Session deleted")
	return nil
}

// GetConnectionInfo retorna informações de conexão de uma sessão
func (m *WhatsAppManager) GetConnectionInfo(sessionID string) (*ConnectionInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.clients[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	if !client.IsConnected() {
		return nil, fmt.Errorf("session not connected")
	}

	store := client.client.Store
	if store.ID == nil {
		return nil, fmt.Errorf("session not authenticated")
	}

	return &ConnectionInfo{
		JID:         store.ID.String(),
		PushName:    store.PushName,
		ConnectedAt: time.Now(), // TODO: armazenar tempo real de conexão
		LastSeen:    time.Now(),
	}, nil
}

// ListSessions retorna todas as sessões ativas
func (m *WhatsAppManager) ListSessions() map[string]*models.Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Criar cópia para evitar race conditions
	result := make(map[string]*models.Session)
	for k, v := range m.sessions {
		result[k] = v
	}
	return result
}

// registerClientHandlers registra handlers para eventos do cliente
func (m *WhatsAppManager) registerClientHandlers(client *MyClient, sessionID string) {
	// Os handlers são registrados no próprio MyClient através do setupDefaultEventHandlers
	// Nenhuma ação adicional necessária aqui
}