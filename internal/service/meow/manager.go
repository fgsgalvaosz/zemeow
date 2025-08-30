package meow

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/felipe/zemeow/internal/config"
	"github.com/felipe/zemeow/internal/db/models"
	"github.com/felipe/zemeow/internal/db/repositories"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
)


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


func (m *WhatsAppManager) Start() error {
	m.logger.Info().Msg("Starting WhatsApp manager")


	sessions, err := m.repository.GetAll(nil)
	if err != nil {
		return fmt.Errorf("failed to load sessions: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()


	for _, session := range sessions.Sessions {
		if err := m.initializeSession(&session); err != nil {
			m.logger.Warn().Err(err).Str("session_id", session.GetSessionID()).Str("name", session.Name).Msg("Failed to initialize session")
			continue
		}
		m.logger.Info().Str("session_id", session.GetSessionID()).Str("name", session.Name).Msg("Session initialized")
	}

	m.logger.Info().Int("session_count", len(m.sessions)).Msg("WhatsApp manager started successfully")
	return nil
}


func (m *WhatsAppManager) Stop() {
	m.logger.Info().Msg("Stopping WhatsApp manager")


	m.cancel()


	m.mu.Lock()
	defer m.mu.Unlock()

	for sessionName, client := range m.clients {
		m.logger.Info().Str("session_name", sessionName).Msg("Disconnecting client")
		client.Disconnect()
	}

	m.logger.Info().Msg("WhatsApp manager stopped")
}


func (m *WhatsAppManager) initializeSession(session *models.Session) error {

	deviceStore, err := m.container.GetFirstDevice(context.Background())
	if err != nil {

		deviceStore = m.container.NewDevice()
	}


	sessionID := session.GetSessionID()
	client := NewMyClient(sessionID, deviceStore, m.webhookChan)


	client.SetOnPairSuccess(m.onPairSuccess)


	m.registerClientHandlers(client, sessionID)


	m.sessions[sessionID] = session
	m.clients[sessionID] = client

	return nil
}


func (m *WhatsAppManager) CreateSession(sessionIdentifier string, config *models.SessionConfig) (*models.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()


	sessionID := uuid.New()
	session := &models.Session{
		ID:        sessionID,
		Name:      config.Name, // nome fornecido na config
		Status:    models.SessionStatusDisconnected,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}


	if _, exists := m.clients[sessionID.String()]; exists {
		return nil, fmt.Errorf("session already exists")
	}


	if config.Proxy != nil {
		session.ProxyEnabled = true
		session.ProxyHost = &config.Proxy.Host
		session.ProxyPort = &config.Proxy.Port
		session.ProxyUsername = &config.Proxy.Username
		session.ProxyPassword = &config.Proxy.Password
	}


	if config.Webhook != nil {
		session.WebhookURL = &config.Webhook.URL
		session.WebhookEvents = pq.StringArray(config.Webhook.Events)
	}


	if err := m.repository.Create(session); err != nil {
		return nil, fmt.Errorf("failed to create session in database: %w", err)
	}


	if err := m.initializeSession(session); err != nil {
		m.repository.Delete(session.ID) // Limpar se falhar
		return nil, fmt.Errorf("failed to initialize session: %w", err)
	}

	m.logger.Info().Str("session_id", session.GetSessionID()).Str("name", session.Name).Msg("Session created successfully")
	return session, nil
}


func (m *WhatsAppManager) GetSession(identifier string) (*models.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()


	if session, exists := m.sessions[identifier]; exists {
		return session, nil
	}


	for _, session := range m.sessions {
		if session.Name == identifier {
			return session, nil
		}
	}

	return nil, fmt.Errorf("session not found")
}


func (m *WhatsAppManager) GetClient(identifier string) (*MyClient, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()


	if client, exists := m.clients[identifier]; exists {
		return client, nil
	}


	for sessionID, session := range m.sessions {
		if session.Name == identifier {
			if client, exists := m.clients[sessionID]; exists {
				return client, nil
			}
		}
	}

	return nil, fmt.Errorf("client not found")
}


func (m *WhatsAppManager) ConnectSession(sessionName string) (*QRCodeData, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info().Str("session_name", sessionName).Msg("Connecting session to WhatsApp")

	client, exists := m.clients[sessionName]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}


	if client.IsLoggedIn() && client.IsConnected() {
		return nil, fmt.Errorf("session already connected")
	}


	if client.client.IsConnected() {
		m.logger.Info().Str("session_name", sessionName).Msg("Cleaning up existing connection")
		client.Disconnect()
		time.Sleep(500 * time.Millisecond) // Breve pausa para limpeza
	}


	if err := m.repository.UpdateStatus(sessionName, models.SessionStatusConnecting); err != nil {
		return nil, fmt.Errorf("failed to update status: %w", err)
	}


	if client.client.Store.ID == nil {

		return m.handleQRCodeFlow(sessionName, client)
	} else {

		return m.handleDirectConnection(sessionName, client)
	}
}


func (m *WhatsAppManager) handleQRCodeFlow(sessionName string, client *MyClient) (*QRCodeData, error) {
	m.logger.Info().Str("session_name", sessionName).Msg("Starting QR code flow")


	qrChan, err := client.client.GetQRChannel(context.Background())
	if err != nil {
		m.logger.Error().Err(err).Str("session_name", sessionName).Msg("Failed to get QR channel")
		m.repository.UpdateStatus(sessionName, models.SessionStatusDisconnected)
		return nil, fmt.Errorf("failed to get QR channel: %w", err)
	}


	go m.handleQREvents(sessionName, qrChan, client)


	err = client.Connect()
	if err != nil {
		m.logger.Error().Err(err).Str("session_name", sessionName).Msg("Failed to connect client")
		m.repository.UpdateStatus(sessionName, models.SessionStatusDisconnected)
		return nil, fmt.Errorf("failed to connect: %w", err)
	}


	return m.waitForQRCode(sessionName, qrChan, client)
}


func (m *WhatsAppManager) handleDirectConnection(sessionName string, client *MyClient) (*QRCodeData, error) {
	m.logger.Info().Str("session_name", sessionName).Msg("Already logged in, connecting directly")

	err := client.Connect()
	if err != nil {
		m.repository.UpdateStatus(sessionName, models.SessionStatusDisconnected)
		return nil, fmt.Errorf("failed to connect: %w", err)
	}


	m.repository.UpdateStatus(sessionName, models.SessionStatusConnected)
	return nil, fmt.Errorf("already logged in, no QR code needed")
}


func (m *WhatsAppManager) waitForQRCode(sessionName string, qrChan <-chan whatsmeow.QRChannelItem, client *MyClient) (*QRCodeData, error) {
	select {
	case evt := <-qrChan:
		if evt.Event == "code" {
			return &QRCodeData{
				Code:      evt.Code,
				Timeout:   int(m.config.WhatsApp.QRCodeTimeout.Seconds()),
				Timestamp: time.Now(),
			}, nil
		} else {
			m.logger.Warn().Str("session_name", sessionName).Str("event", evt.Event).Msg("Unexpected QR event")
			return nil, fmt.Errorf("unexpected QR event: %s", evt.Event)
		}
	case <-time.After(m.config.WhatsApp.QRCodeTimeout):
		client.Disconnect()
		m.repository.UpdateStatus(sessionName, models.SessionStatusDisconnected)
		return nil, fmt.Errorf("QR code timeout")
	}
}


func (m *WhatsAppManager) DisconnectSession(sessionName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, exists := m.clients[sessionName]
	if !exists {
		return fmt.Errorf("session not found")
	}

	client.Disconnect()
	m.repository.UpdateStatus(sessionName, models.SessionStatusDisconnected)

	m.logger.Info().Str("session_name", sessionName).Msg("Session disconnected")
	return nil
}


func (m *WhatsAppManager) DeleteSession(sessionName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()


	if client, exists := m.clients[sessionName]; exists {
		client.Disconnect()
		delete(m.clients, sessionName)
	}


	delete(m.sessions, sessionName)


	err := m.repository.DeleteByIdentifier(sessionName)
	if err != nil {
		return fmt.Errorf("failed to delete session from database: %w", err)
	}

	m.logger.Info().Str("session_name", sessionName).Msg("Session deleted")
	return nil
}


func (m *WhatsAppManager) GetConnectionInfo(sessionName string) (*ConnectionInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.clients[sessionName]
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


func (m *WhatsAppManager) ListSessions() map[string]*models.Session {
	m.mu.RLock()
	defer m.mu.RUnlock()


	result := make(map[string]*models.Session)
	for k, v := range m.sessions {
		result[k] = v
	}
	return result
}


func (m *WhatsAppManager) registerClientHandlers(client *MyClient, sessionName string) {


}


func (m *WhatsAppManager) handleQREvents(sessionName string, qrChan <-chan whatsmeow.QRChannelItem, client *MyClient) {
	defer func() {
		if r := recover(); r != nil {
			m.logger.Error().Interface("panic", r).Str("session_name", sessionName).Msg("Panic in QR event handler")
		}
	}()

	for evt := range qrChan {
		m.logger.Info().Str("session_name", sessionName).Str("event", evt.Event).Msg("QR event received")

		switch evt.Event {
		case "code":
			m.handleQRCodeEvent(sessionName, evt.Code)

		case "timeout":
			m.handleQRTimeoutEvent(sessionName, client)

		case "success":
			m.logger.Info().Str("session_name", sessionName).Msg("QR pairing successful! (handled by PairSuccess event)")

		default:
			m.logger.Info().Str("session_name", sessionName).Str("event", evt.Event).Msg("Unknown QR event")
		}
	}
}


func (m *WhatsAppManager) handleQRCodeEvent(sessionName, qrCode string) {
	m.logger.Info().Str("session_name", sessionName).Msg("QR Code generated! Scan with WhatsApp:")
	m.logger.Info().Str("session_name", sessionName).Str("qr_code", qrCode).Msg("QR Code data")


	if err := m.displayQRInTerminal(qrCode, sessionName); err != nil {
		m.logger.Error().Err(err).Str("session_name", sessionName).Msg("Failed to display QR in terminal")
	}
}


func (m *WhatsAppManager) handleQRTimeoutEvent(sessionName string, client *MyClient) {
	m.logger.Warn().Str("session_name", sessionName).Msg("QR code timeout")


	if err := m.repository.UpdateStatus(sessionName, models.SessionStatusDisconnected); err != nil {
		m.logger.Error().Err(err).Str("session_name", sessionName).Msg("Failed to update status on timeout")
	}


	if err := m.repository.ClearQRCode(sessionName); err != nil {
		m.logger.Error().Err(err).Str("session_name", sessionName).Msg("Failed to clear QR code on timeout")
	}

	client.Disconnect()
}


func (m *WhatsAppManager) displayQRInTerminal(qrCode, sessionName string) error {

	fmt.Printf("\n=== QR CODE FOR SESSION %s ===\n", sessionName)
	fmt.Println("Scan this QR code with your WhatsApp app:")
	fmt.Println("1. Open WhatsApp on your phone")
	fmt.Println("2. Go to Settings > Linked Devices")
	fmt.Println("3. Tap 'Link a Device'")
	fmt.Println("4. Scan the QR code below")
	fmt.Println("=============================")


	qrterminal.GenerateHalfBlock(qrCode, qrterminal.L, os.Stdout)
	fmt.Printf("QR code: %s\n", qrCode)
	fmt.Println("===============================")


	m.logger.Info().Str("session_name", sessionName).Str("qr_data", qrCode).Msg("QR Code displayed in terminal")

	return nil
}


func (m *WhatsAppManager) onPairSuccess(sessionName, jid string) {
	m.logger.Info().Str("session_name", sessionName).Str("jid", jid).Msg("QR pairing successful! Updating database...")


	if err := m.repository.UpdateStatusAndJID(sessionName, models.SessionStatusConnected, &jid); err != nil {
		m.logger.Error().Err(err).Str("session_name", sessionName).Str("jid", jid).Msg("Failed to update status and JID on pair success")
	} else {
		m.logger.Info().Str("session_name", sessionName).Str("jid", jid).Msg("Session connected and JID updated successfully")
	}


	if err := m.repository.ClearQRCode(sessionName); err != nil {
		m.logger.Error().Err(err).Str("session_name", sessionName).Msg("Failed to clear QR code")
	}
}

