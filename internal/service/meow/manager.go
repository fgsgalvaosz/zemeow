package meow

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"

	"github.com/felipe/zemeow/internal/config"
	"github.com/felipe/zemeow/internal/db/models"
	"github.com/felipe/zemeow/internal/db/repositories"
	"github.com/felipe/zemeow/internal/logger"
)

// WhatsAppManager gerencia múltiplas sessões WhatsApp de forma thread-safe
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

// WebhookEvent representa um evento para webhook
type WebhookEvent struct {
	SessionID string      `json:"session_id"`
	Event     string      `json:"event"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// QRCodeData representa os dados do QR Code
type QRCodeData struct {
	Code      string    `json:"code"`
	Timeout   int       `json:"timeout"`
	Timestamp time.Time `json:"timestamp"`
}

// ConnectionInfo representa informações de conexão
type ConnectionInfo struct {
	JID          string    `json:"jid"`
	PushName     string    `json:"push_name"`
	BusinessName string    `json:"business_name,omitempty"`
	ConnectedAt  time.Time `json:"connected_at"`
	LastSeen     time.Time `json:"last_seen"`
	BatteryLevel int       `json:"battery_level,omitempty"`
	Plugged      bool      `json:"plugged,omitempty"`
}

// NewWhatsAppManager cria uma nova instância do gerenciador
func NewWhatsAppManager(db *sql.DB, repository repositories.SessionRepository, config *config.Config) *WhatsAppManager {
	ctx, cancel := context.WithCancel(context.Background())

	// Criar logger para whatsmeow
	whatsmeowLogger := logger.GetWhatsAppLogger("store")

	// Criar container do sqlstore
	container := sqlstore.NewWithDB(db, "postgres", whatsmeowLogger)

	return &WhatsAppManager{
		clients:     make(map[string]*MyClient),
		sessions:    make(map[string]*models.Session),
		container:   container,
		repository:  repository,
		config:      config,
		logger:      logger.GetWithSession("whatsapp_manager"),
		ctx:         ctx,
		cancel:      cancel,
		webhookChan: make(chan WebhookEvent, 1000),
	}
}

// Start inicia o gerenciador e carrega sessões ativas
func (m *WhatsAppManager) Start() error {
	m.logger.Info().Msg("Starting WhatsApp Manager")

	// Fazer upgrade das tabelas do whatsmeow se necessário
	if err := m.container.Upgrade(); err != nil {
		m.logger.Error().Err(err).Msg("Failed to upgrade whatsmeow store")
		return fmt.Errorf("failed to upgrade whatsmeow store: %w", err)
	}

	// Carregar sessões ativas do banco
	activeSessions, err := m.repository.GetActiveConnections()
	if err != nil {
		m.logger.Error().Err(err).Msg("Failed to load active sessions")
		return fmt.Errorf("failed to load active sessions: %w", err)
	}

	// Inicializar sessões ativas
	for _, session := range activeSessions {
		if err := m.initializeSession(session); err != nil {
			m.logger.Error().Err(err).Str("session_id", session.SessionID).Msg("Failed to initialize session")
			// Atualizar status para disconnected se falhar
			m.repository.UpdateStatus(session.SessionID, models.SessionStatusDisconnected)
		}
	}

	// Iniciar processamento de webhooks
	go m.processWebhooks()

	m.logger.Info().Int("active_sessions", len(activeSessions)).Msg("WhatsApp Manager started")
	return nil
}

// Stop para o gerenciador e desconecta todas as sessões
func (m *WhatsAppManager) Stop() {
	m.logger.Info().Msg("Stopping WhatsApp Manager")

	m.cancel()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Desconectar todos os clientes
	for sessionID, client := range m.clients {
		client.Disconnect()
		m.logger.Info().Str("session_id", sessionID).Msg("Session disconnected")
	}

	// Limpar maps
	m.clients = make(map[string]*MyClient)
	m.sessions = make(map[string]*models.Session)

	close(m.webhookChan)
	m.logger.Info().Msg("WhatsApp Manager stopped")
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
		Token:     uuid.New().String(),
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

// IsSessionActive verifica se uma sessão está ativa
func (m *WhatsAppManager) IsSessionActive(sessionID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.clients[sessionID]
	return exists && client.IsConnected()
}

// GetSessionQRCode obtém o QR Code de uma sessão
func (m *WhatsAppManager) GetSessionQRCode(sessionID string) (*QRCodeData, error) {
	m.mu.RLock()
	client, exists := m.clients[sessionID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	if client.IsLoggedIn() {
		return nil, fmt.Errorf("session is already logged in")
	}

	// Iniciar processo de QR Code (whatsmeow gerará evento QR)
	if !client.IsConnected() {
		if err := client.Connect(); err != nil {
			return nil, fmt.Errorf("failed to connect for QR code: %w", err)
		}
	}

	// O QR Code será enviado via webhook quando gerado
	return &QRCodeData{
		Code:      "qr_generation_initiated",
		Timeout:   60,
		Timestamp: time.Now(),
	}, nil
}



// initializeSession inicializa uma sessão WhatsApp
func (m *WhatsAppManager) initializeSession(session *models.Session) error {
	// Obter device store
	deviceStore, err := m.container.GetFirstDevice()
	if err != nil {
		// Criar novo device se não existir
		deviceStore = m.container.NewDevice()
	}

	// Criar MyClient personalizado
	myClient := NewMyClient(session.SessionID, deviceStore, m.webhookChan)

	// Armazenar na memória
	m.clients[session.SessionID] = myClient
	m.sessions[session.SessionID] = session

	m.logger.Info().Str("session_id", session.SessionID).Msg("Session initialized")
	return nil
}

// sendWebhook envia um evento para o canal de webhooks
func (m *WhatsAppManager) sendWebhook(sessionID, event string, data interface{}) {
	select {
	case m.webhookChan <- WebhookEvent{
		SessionID: sessionID,
		Event:     event,
		Data:      data,
		Timestamp: time.Now(),
	}:
	default:
		m.logger.Warn().Str("session_id", sessionID).Str("event", event).Msg("Webhook channel full, dropping event")
	}
}

// processWebhooks processa eventos de webhook em background
func (m *WhatsAppManager) processWebhooks() {
	m.logger.Info().Msg("Starting webhook processor")

	for {
		select {
		case event, ok := <-m.webhookChan:
			if !ok {
				m.logger.Info().Msg("Webhook processor stopped")
				return
			}

			// TODO: Implementar envio real de webhook
			m.logger.Debug().Str("session_id", event.SessionID).Str("event", event.Event).Msg("Processing webhook event")

		case <-m.ctx.Done():
			m.logger.Info().Msg("Webhook processor stopped by context")
			return
		}
	}
}

// GetWebhookChannel retorna o canal de webhooks para processamento externo
func (m *WhatsAppManager) GetWebhookChannel() <-chan WebhookEvent {
	return m.webhookChan
}
