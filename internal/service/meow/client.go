package meow

import (
	"fmt"
	"sync"
	"time"

	"github.com/felipe/zemeow/internal/db/models"
	"github.com/felipe/zemeow/internal/logger"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

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

// MyClient representa um cliente WhatsApp personalizado para uma sessão
type MyClient struct {
	mu            sync.RWMutex
	sessionID     string
	client        *whatsmeow.Client
	deviceStore   *store.Device
	eventHandlers map[uint32]func(interface{})
	isConnected   bool
	logger        logger.Logger
	webhookChan   chan<- WebhookEvent

	// Estatísticas
	messagesReceived int64
	messagesSent     int64
	reconnections    int64
	lastActivity     time.Time
}

// NewMyClient cria um novo cliente personalizado
func NewMyClient(sessionID string, deviceStore *store.Device, webhookChan chan<- WebhookEvent) *MyClient {
	clientLogger := logger.GetWhatsAppLogger(sessionID)

	client := whatsmeow.NewClient(deviceStore, clientLogger)

	myClient := &MyClient{
		sessionID:     sessionID,
		client:        client,
		deviceStore:   deviceStore,
		eventHandlers: make(map[uint32]func(interface{})),
		logger:        logger.GetWithSession(sessionID),
		webhookChan:   webhookChan,
		lastActivity:  time.Now(),
	}

	// Configurar event handlers padrão
	myClient.setupDefaultEventHandlers()

	return myClient
}

// setupDefaultEventHandlers configura os event handlers padrão
func (c *MyClient) setupDefaultEventHandlers() {
	// Handler único com type switch para todos os eventos
	c.client.AddEventHandler(c.handleEvent)
}

// handleEvent processa todos os tipos de eventos
func (c *MyClient) handleEvent(evt interface{}) {
	switch v := evt.(type) {
	case *events.Connected:
		c.mu.Lock()
		c.isConnected = true
		c.lastActivity = time.Now()
		c.mu.Unlock()

		c.logger.Info().Msg("Connected to WhatsApp")
		c.sendWebhookEvent("connected", map[string]interface{}{
			"session_id": c.sessionID,
			"timestamp": time.Now().Unix(),
		})
	case *events.Disconnected:
		c.mu.Lock()
		c.isConnected = false
		c.lastActivity = time.Now()
		c.mu.Unlock()

		c.logger.Info().Msg("Disconnected from WhatsApp")
		c.sendWebhookEvent("disconnected", map[string]interface{}{
			"session_id": c.sessionID,
			"timestamp": time.Now().Unix(),
		})
	case *events.Message:
		c.mu.Lock()
		c.messagesReceived++
		c.lastActivity = time.Now()
		c.mu.Unlock()

		c.logger.Info().Str("from", v.Info.Sender.String()).Msg("Received message")
		c.sendWebhookEvent("message", map[string]interface{}{
			"session_id": c.sessionID,
			"message_id": v.Info.ID,
			"from":       v.Info.Sender.String(),
			"chat":       v.Info.Chat.String(),
			"timestamp":  v.Info.Timestamp.Unix(),
			"message":    v.Message,
		})
	case *events.QR:
		c.logger.Info().Msg("QR code received")
		c.sendWebhookEvent("qr", map[string]interface{}{
			"session_id": c.sessionID,
			"qr_code":    v.Codes,
			"timestamp":  time.Now().Unix(),
		})
	case *events.PairSuccess:
		c.logger.Info().Msg("Paired successfully")
		c.sendWebhookEvent("pair_success", map[string]interface{}{
			"session_id": c.sessionID,
			"timestamp":  time.Now().Unix(),
		})
	case *events.StreamError:
		c.logger.Error().Str("code", v.Code).Msg("Stream error")
		c.sendWebhookEvent("stream_error", map[string]interface{}{
			"session_id": c.sessionID,
			"code":       v.Code,
			"timestamp":  time.Now().Unix(),
		})
	case *events.ConnectFailure:
		c.mu.Lock()
		c.reconnections++
		c.mu.Unlock()

		c.logger.Error().Int("reason", int(v.Reason)).Msg("Connection failed")
		c.sendWebhookEvent("connect_failure", map[string]interface{}{
			"session_id": c.sessionID,
			"reason":     int(v.Reason),
			"timestamp":  time.Now().Unix(),
		})
	}
}

// Connect conecta o cliente ao WhatsApp
func (c *MyClient) Connect() error {
	c.logger.Info().Msg("Connecting to WhatsApp")

	if c.IsConnected() {
		return fmt.Errorf("client is already connected")
	}

	return c.client.Connect()
}

// Disconnect desconecta o cliente do WhatsApp
func (c *MyClient) Disconnect() {
	c.logger.Info().Msg("Disconnecting from WhatsApp")
	c.client.Disconnect()

	c.mu.Lock()
	c.isConnected = false
	c.mu.Unlock()
}

// IsConnected verifica se o cliente está conectado
func (c *MyClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isConnected && c.client.IsConnected()
}

// IsLoggedIn verifica se o cliente está logado
func (c *MyClient) IsLoggedIn() bool {
	return c.client.IsLoggedIn()
}

// GetJID retorna o JID do cliente
func (c *MyClient) GetJID() types.JID {
	if c.client.Store.ID == nil {
		return types.EmptyJID
	}
	return *c.client.Store.ID
}

// GetPushName retorna o push name do cliente
func (c *MyClient) GetPushName() string {
	return c.client.Store.PushName
}

// GetStatistics retorna estatísticas do cliente
func (c *MyClient) GetStatistics() *models.SessionStatistics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return &models.SessionStatistics{
		MessagesReceived: int(c.messagesReceived),
		MessagesSent:     int(c.messagesSent),
		Reconnections:    int(c.reconnections),
		LastActivity:     c.lastActivity,
	}
}

// RemoveEventHandler remove um event handler
func (c *MyClient) RemoveEventHandler(id uint32) {
	c.client.RemoveEventHandler(id)
}

// GetClient retorna o cliente whatsmeow subjacente
func (c *MyClient) GetClient() *whatsmeow.Client {
	return c.client
}

// sendWebhookEvent envia um evento para o canal de webhook
func (c *MyClient) sendWebhookEvent(event string, data interface{}) {
	if c.webhookChan == nil {
		return
	}

	webhookEvent := WebhookEvent{
		SessionID: c.sessionID,
		Event:     event,
		Data:      data,
		Timestamp: time.Now(),
	}

	select {
	case c.webhookChan <- webhookEvent:
		c.logger.Debug().Str("event", event).Msg("Webhook event sent")
	default:
		c.logger.Warn().Str("event", event).Msg("Webhook channel full, event dropped")
	}
}

// PairPhone inicia o processo de pareamento por telefone
func (c *MyClient) PairPhone(phoneNumber string) error {
	c.logger.Info().Str("phone", phoneNumber).Msg("Starting phone pairing")

	if c.IsLoggedIn() {
		return fmt.Errorf("client is already logged in")
	}

	// Implementar lógica de pareamento por telefone
	// Isso depende da versão específica do whatsmeow
	return fmt.Errorf("phone pairing not implemented yet")
}

// Logout faz logout do cliente
func (c *MyClient) Logout() error {
	c.logger.Info().Msg("Logging out from WhatsApp")

	if !c.IsLoggedIn() {
		return fmt.Errorf("client is not logged in")
	}

	err := c.client.Logout()
	if err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}

	c.sendWebhookEvent("logout", map[string]interface{}{
		"logged_out_at": time.Now(),
	})

	return nil
}
