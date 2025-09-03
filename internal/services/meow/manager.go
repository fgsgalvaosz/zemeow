package meow

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image/png"
	"os"
	"sync"
	"time"

	"github.com/felipe/zemeow/internal/config"
	"github.com/felipe/zemeow/internal/models"
	"github.com/felipe/zemeow/internal/repositories"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/felipe/zemeow/internal/services/media"
	"github.com/felipe/zemeow/internal/services/message"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/mdp/qrterminal/v3"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
)

type WhatsAppManager struct {
	mu                 sync.RWMutex
	clients            map[string]*MyClient
	sessions           map[string]*models.Session
	container          *sqlstore.Container
	repository         repositories.SessionRepository
	messageRepo        repositories.MessageRepository
	messagePersistence *message.PersistenceService
	config             *config.Config
	logger             logger.Logger
	ctx                context.Context
	cancel             context.CancelFunc
	webhookChan        chan WebhookEvent
}

func NewWhatsAppManager(container *sqlstore.Container, repository repositories.SessionRepository, messageRepo repositories.MessageRepository, config *config.Config) *WhatsAppManager {
	ctx, cancel := context.WithCancel(context.Background())

	var mediaService *media.MediaService
	if config.MinIO.Endpoint != "" {
		var err error
		mediaService, err = media.NewMediaServiceFromConfig(&config.MinIO)
		if err != nil {
			logger.Get().Warn().Err(err).Msg("Failed to initialize MediaService, media storage will be disabled")
			mediaService = nil
		} else {
			logger.Get().Info().Msg("MediaService initialized successfully")
		}
	} else {
		logger.Get().Warn().Msg("MinIO not configured, media storage will be disabled")
	}

	manager := &WhatsAppManager{
		clients:     make(map[string]*MyClient),
		sessions:    make(map[string]*models.Session),
		container:   container,
		repository:  repository,
		messageRepo: messageRepo,
		config:      config,
		logger:      logger.GetWithSession("whatsapp_manager"),
		ctx:         ctx,
		cancel:      cancel,
		webhookChan: make(chan WebhookEvent, 1000),
	}

	messagePersistence := message.NewPersistenceService(messageRepo, mediaService, manager)

	manager.messagePersistence = messagePersistence
	return manager
}

func (m *WhatsAppManager) GetClient(sessionID string) *whatsmeow.Client {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if myClient, exists := m.clients[sessionID]; exists {
		return myClient.client
	}
	return nil
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

	go m.reconnectOnStartup()

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
	sessionID := session.GetSessionID()

	var deviceStore *store.Device

	if session.JID != nil && *session.JID != "" {

		jid, err := types.ParseJID(*session.JID)
		if err == nil {
			deviceStore, err = m.container.GetDevice(context.Background(), jid)
			if err != nil {
				m.logger.Warn().Err(err).Str("session_id", sessionID).Str("jid", *session.JID).Msg("Failed to get existing device, creating new one")
				deviceStore = m.container.NewDevice()
			}
		} else {
			m.logger.Warn().Err(err).Str("session_id", sessionID).Str("jid", *session.JID).Msg("Invalid JID format, creating new device")
			deviceStore = m.container.NewDevice()
		}
	} else {

		deviceStore = m.container.NewDevice()
		m.logger.Info().Str("session_id", sessionID).Msg("Creating new device for session without JID")
	}

	client := NewMyClient(sessionID, session.ID, deviceStore, m.webhookChan, m.messagePersistence)
	client.SetOnPairSuccess(m.onPairSuccess)
	m.registerClientHandlers(client, sessionID)

	m.sessions[sessionID] = session
	m.clients[sessionID] = client

	return nil
}

func (m *WhatsAppManager) InitializeSession(session *models.Session) error {
	return m.initializeSession(session)
}

func (m *WhatsAppManager) CreateSession(sessionIdentifier string, config *models.SessionConfig) (*models.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sessionID := uuid.New()
	session := &models.Session{
		ID:        sessionID,
		Name:      config.Name,
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

	// Initialize the session immediately after creation to ensure proper setup
	if err := m.initializeSession(session); err != nil {
		m.repository.Delete(session.ID)
		return nil, fmt.Errorf("failed to initialize session: %w", err)
	}

	m.logger.Info().Str("session_id", session.GetSessionID()).Str("name", session.Name).Msg("Session created and initialized successfully")
	return session, nil
}

func (m *WhatsAppManager) InitializeNewSession(session *models.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sessionID := session.GetSessionID()

	if _, exists := m.clients[sessionID]; exists {
		return fmt.Errorf("session already initialized")
	}

	if err := m.initializeSession(session); err != nil {
		return fmt.Errorf("failed to initialize session: %w", err)
	}

	m.logger.Info().Str("session_id", sessionID).Str("name", session.Name).Msg("New session initialized in manager")
	return nil
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

func (m *WhatsAppManager) GetMyClient(identifier string) (*MyClient, error) {

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

	return nil, fmt.Errorf("client not found for identifier: %s", identifier)
}

func (m *WhatsAppManager) ConnectSession(sessionID string) (*QRCodeData, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info().Str("session_id", sessionID).Msg("Connecting session to WhatsApp")

	client, err := m.GetMyClient(sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	if client.IsLoggedIn() && client.IsConnected() {
		return nil, fmt.Errorf("session already connected")
	}

	if client.client.IsConnected() {
		m.logger.Info().Str("session_id", sessionID).Msg("Cleaning up existing connection")
		client.Disconnect()
		time.Sleep(500 * time.Millisecond)
	}

	if err := m.repository.UpdateStatus(sessionID, models.SessionStatusConnecting); err != nil {
		return nil, fmt.Errorf("failed to update status: %w", err)
	}

	if client.client.Store.ID == nil {
		return m.handleQRCodeFlow(sessionID, client)
	} else {
		return m.handleDirectConnection(sessionID, client)
	}
}

func (m *WhatsAppManager) handleQRCodeFlow(sessionID string, client *MyClient) (*QRCodeData, error) {
	m.logger.Info().Str("session_id", sessionID).Msg("Starting QR code flow")

	go func() {
		defer func() {
			if r := recover(); r != nil {
				m.logger.Error().Interface("panic", r).Str("session_id", sessionID).Msg("Panic in QR connection")
			}
		}()

		qrChan, err := client.client.GetQRChannel(context.Background())
		if err != nil {
			m.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to get QR channel")
			m.repository.UpdateStatus(sessionID, models.SessionStatusDisconnected)
			return
		}

		err = client.Connect()
		if err != nil {
			m.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to connect client")
			m.repository.UpdateStatus(sessionID, models.SessionStatusDisconnected)
			return
		}

		for evt := range qrChan {
			switch evt.Event {
			case "code":
				m.logger.Info().Str("session_id", sessionID).Msg("QR code generated")

				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				fmt.Printf("QR code for session %s:\n%s\n", sessionID, evt.Code)

				qrBase64, err := m.generateQRCodeBase64(evt.Code)
				if err != nil {
					m.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to generate QR code image")

					qrBase64 = evt.Code
				}

				if err := m.repository.UpdateQRCode(sessionID, qrBase64); err != nil {
					m.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to store QR code")
				}

			case "timeout":
				m.logger.Warn().Str("session_id", sessionID).Msg("QR code timeout")
				m.repository.UpdateStatus(sessionID, models.SessionStatusDisconnected)
				return

			case "success":
				m.logger.Info().Str("session_id", sessionID).Msg("QR pairing successful!")
				return

			default:
				m.logger.Info().Str("session_id", sessionID).Str("event", evt.Event).Msg("QR event received")
			}
		}
	}()

	return &QRCodeData{
		Code:      "connecting",
		Timeout:   int(m.config.WhatsApp.QRCodeTimeout.Seconds()),
		Timestamp: time.Now(),
	}, nil
}

func (m *WhatsAppManager) handleDirectConnection(sessionID string, client *MyClient) (*QRCodeData, error) {
	m.logger.Info().Str("session_id", sessionID).Msg("Already logged in, connecting directly")

	err := client.Connect()
	if err != nil {
		m.repository.UpdateStatus(sessionID, models.SessionStatusDisconnected)
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	time.Sleep(1 * time.Second)

	var jid *string
	if client.client.Store.ID != nil {
		jidStr := client.client.Store.ID.String()
		jid = &jidStr
		m.logger.Info().Str("session_id", sessionID).Str("jid", jidStr).Msg("Device JID obtained from direct connection")

		if err := m.repository.UpdateStatusAndJID(sessionID, models.SessionStatusConnected, jid); err != nil {
			m.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to update session status and JID")
		}
	} else {
		m.logger.Warn().Str("session_id", sessionID).Msg("Store ID is nil after connection - JID will be updated via Connected event")

		m.repository.UpdateStatus(sessionID, models.SessionStatusConnected)
	}

	return nil, fmt.Errorf("already logged in, no QR code needed")
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

func (m *WhatsAppManager) GetQRCode(sessionID string) (*QRCodeData, error) {

	session, err := m.repository.GetByIdentifier(sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	if session.QRCode != nil && *session.QRCode != "" && *session.QRCode != "connecting" {
		m.logger.Info().Str("session_id", sessionID).Msg("Returning existing QR code from database")
		return &QRCodeData{
			Code:      *session.QRCode,
			Timeout:   int(m.config.WhatsApp.QRCodeTimeout.Seconds()),
			Timestamp: session.UpdatedAt,
		}, nil
	}

	m.mu.RLock()
	client, exists := m.clients[sessionID]
	m.mu.RUnlock()

	if exists && client != nil {

		if session.QRCode != nil && *session.QRCode == "connecting" {
			return &QRCodeData{
				Code:      "connecting",
				Timeout:   int(m.config.WhatsApp.QRCodeTimeout.Seconds()),
				Timestamp: session.UpdatedAt,
			}, nil
		}
	}

	return nil, fmt.Errorf("no QR code available - session needs to be connected first. Use POST /sessions/%s/connect", sessionID)
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
		ConnectedAt: time.Now(),
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

func (m *WhatsAppManager) GetWebhookChannel() <-chan WebhookEvent {
	return m.webhookChan
}


func (m *WhatsAppManager) generateQRCodeBase64(qrText string) (string, error) {

	qrCode, err := qrcode.New(qrText, qrcode.Medium)
	if err != nil {
		return "", fmt.Errorf("failed to create QR code: %w", err)
	}

	qrCode.DisableBorder = false

	img := qrCode.Image(300)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", fmt.Errorf("failed to encode PNG: %w", err)
	}

	base64Str := base64.StdEncoding.EncodeToString(buf.Bytes())

	return fmt.Sprintf("data:image/png;base64,%s", base64Str), nil
}

func (m *WhatsAppManager) onPairSuccess(sessionName, jid string) {
	m.logger.Info().Str("session_name", sessionName).Str("jid", jid).Msg("QR pairing successful! Updating database...")

	// Update the session with the real JID and status
	if err := m.repository.UpdateStatusAndJID(sessionName, models.SessionStatusConnected, &jid); err != nil {
		m.logger.Error().Err(err).Str("session_name", sessionName).Str("jid", jid).Msg("Failed to update status and JID on pair success")
		
		// Try to update only the JID if status update failed
		if err2 := m.repository.UpdateJID(sessionName, &jid); err2 != nil {
			m.logger.Error().Err(err2).Str("session_name", sessionName).Str("jid", jid).Msg("Failed to update JID on pair success")
		} else {
			// Also update status separately
			if err3 := m.repository.UpdateStatus(sessionName, models.SessionStatusConnected); err3 != nil {
				m.logger.Error().Err(err3).Str("session_name", sessionName).Msg("Failed to update status on pair success")
			} else {
				m.logger.Info().Str("session_name", sessionName).Str("jid", jid).Msg("Session JID and status updated successfully in separate operations")
			}
		}
	} else {
		m.logger.Info().Str("session_name", sessionName).Str("jid", jid).Msg("Session connected and JID updated successfully")
	}

	if err := m.repository.ClearQRCode(sessionName); err != nil {
		m.logger.Error().Err(err).Str("session_name", sessionName).Msg("Failed to clear QR code")
	}
}

func (m *WhatsAppManager) reconnectOnStartup() {
	m.logger.Info().Msg("Starting automatic reconnection of previously connected sessions")

	sessions, err := m.repository.GetActiveConnections()
	if err != nil {
		m.logger.Error().Err(err).Msg("Failed to get active connections for reconnection")
		return
	}

	if len(sessions) == 0 {
		m.logger.Info().Msg("No previously connected sessions found")
		return
	}

	m.logger.Info().Int("session_count", len(sessions)).Msg("Found previously connected sessions, attempting reconnection")

	for _, session := range sessions {
		sessionID := session.GetSessionID()

		if _, exists := m.clients[sessionID]; !exists {
			m.logger.Warn().Str("session_id", sessionID).Str("name", session.Name).Msg("Session not initialized, skipping reconnection")
			continue
		}

		m.logger.Info().Str("session_id", sessionID).Str("name", session.Name).Str("jid", func() string {
			if session.JID != nil {
				return *session.JID
			}
			return "nil"
		}()).Msg("Attempting to reconnect session")

		go func(sess *models.Session) {
			sessionID := sess.GetSessionID()

			time.Sleep(2 * time.Second)

			client, exists := m.clients[sessionID]
			if !exists {
				m.logger.Error().Str("session_id", sessionID).Msg("Client not found for reconnection")
				return
			}

			if sess.JID != nil && *sess.JID != "" {
				m.logger.Info().Str("session_id", sessionID).Str("jid", *sess.JID).Msg("Session has JID, attempting direct connection")

				if err := m.repository.UpdateStatus(sessionID, models.SessionStatusConnecting); err != nil {
					m.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to update status to connecting")
				}

				if err := client.Connect(); err != nil {
					m.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to reconnect session")

					if err := m.repository.UpdateStatus(sessionID, models.SessionStatusDisconnected); err != nil {
						m.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to update status to disconnected after reconnection failure")
					}
				} else {
					m.logger.Info().Str("session_id", sessionID).Msg("Session reconnected successfully")
				}
			} else {
				m.logger.Info().Str("session_id", sessionID).Msg("Session has no JID, marking as disconnected")

				if err := m.repository.UpdateStatus(sessionID, models.SessionStatusDisconnected); err != nil {
					m.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to update status to disconnected")
				}
			}
		}(session)
	}
}
