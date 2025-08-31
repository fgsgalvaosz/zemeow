package meow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/felipe/zemeow/internal/db/models"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/felipe/zemeow/internal/service/message"
	"github.com/google/uuid"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

type WebhookEvent struct {
	SessionID string      `json:"session_id"`
	Event     string      `json:"event"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

type QRCodeData struct {
	Code      string    `json:"code"`
	Timeout   int       `json:"timeout"`
	Timestamp time.Time `json:"timestamp"`
}

type ConnectionInfo struct {
	JID          string    `json:"jid"`
	PushName     string    `json:"push_name"`
	BusinessName string    `json:"business_name,omitempty"`
	ConnectedAt  time.Time `json:"connected_at"`
	LastSeen     time.Time `json:"last_seen"`
	BatteryLevel int       `json:"battery_level,omitempty"`
	Plugged      bool      `json:"plugged,omitempty"`
}

type MyClient struct {
	mu            sync.RWMutex
	sessionID     string
	sessionUUID   uuid.UUID
	client        *whatsmeow.Client
	deviceStore   *store.Device
	eventHandlers map[uint32]func(interface{})
	isConnected   bool
	logger        logger.Logger
	webhookChan   chan<- WebhookEvent

	messagePersistence *message.PersistenceService

	onPairSuccess func(sessionID, jid string)

	messagesReceived int64
	messagesSent     int64
	reconnections    int64
	lastActivity     time.Time
}

func NewMyClient(sessionID string, sessionUUID uuid.UUID, deviceStore *store.Device, webhookChan chan<- WebhookEvent, messagePersistence *message.PersistenceService) *MyClient {
	clientLogger := logger.GetWhatsAppLogger(sessionID)

	client := whatsmeow.NewClient(deviceStore, clientLogger)

	myClient := &MyClient{
		sessionID:          sessionID,
		sessionUUID:        sessionUUID,
		client:             client,
		deviceStore:        deviceStore,
		eventHandlers:      make(map[uint32]func(interface{})),
		logger:             logger.GetWithSession(sessionID),
		webhookChan:        webhookChan,
		messagePersistence: messagePersistence,
		lastActivity:       time.Now(),
	}

	myClient.setupDefaultEventHandlers()

	return myClient
}

func (c *MyClient) SetOnPairSuccess(callback func(sessionID, jid string)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onPairSuccess = callback
}

func (c *MyClient) setupDefaultEventHandlers() {

	c.client.AddEventHandler(c.handleEvent)
}

func (c *MyClient) handleEvent(evt interface{}) {
	switch v := evt.(type) {
	case *events.Connected:
		c.mu.Lock()
		c.isConnected = true
		c.lastActivity = time.Now()
		c.mu.Unlock()

		c.logger.Info().Msg("Connected to WhatsApp")

		if c.client.IsLoggedIn() && c.client.Store.ID != nil {
			jid := c.client.Store.ID.String()
			c.logger.Info().Str("jid", jid).Msg("Device already logged in, ensuring JID is updated")

			c.mu.RLock()
			callback := c.onPairSuccess
			c.mu.RUnlock()

			if callback != nil {
				callback(c.sessionID, jid)
			}
		}

		c.sendWebhookEvent("connected", map[string]interface{}{
			"session_id": c.sessionID,
			"timestamp":  time.Now().Unix(),
		})
	case *events.Disconnected:
		c.mu.Lock()
		c.isConnected = false
		c.lastActivity = time.Now()
		c.mu.Unlock()

		c.logger.Info().Msg("Disconnected from WhatsApp")
		c.sendWebhookEvent("disconnected", map[string]interface{}{
			"session_id": c.sessionID,
			"timestamp":  time.Now().Unix(),
		})
	case *events.Message:
		c.mu.Lock()
		c.messagesReceived++
		c.lastActivity = time.Now()
		c.mu.Unlock()

		c.logger.Info().Str("from", v.Info.Sender.String()).Msg("Received message")

		if c.messagePersistence != nil {
			err := c.messagePersistence.ProcessMessageEvent(c.sessionUUID, v)
			if err != nil {
				c.logger.Error().Err(err).Str("message_id", v.Info.ID).Msg("Failed to persist message")
			}
		}

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
		jid := v.ID.String()
		c.logger.Info().Str("jid", jid).Str("business_name", v.BusinessName).Str("platform", v.Platform).Msg("QR Pair Success")

		c.mu.RLock()
		callback := c.onPairSuccess
		c.mu.RUnlock()

		if callback != nil {
			callback(c.sessionID, jid)
		}

		c.sendWebhookEvent("pair_success", map[string]interface{}{
			"session_id":    c.sessionID,
			"jid":           jid,
			"business_name": v.BusinessName,
			"platform":      v.Platform,
			"timestamp":     time.Now().Unix(),
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
	case *events.Receipt:
		c.logger.Debug().Str("message_id", v.MessageIDs[0]).Str("type", string(v.Type)).Msg("Message receipt")

		if c.messagePersistence != nil {
			err := c.messagePersistence.ProcessReceiptEvent(c.sessionUUID, v)
			if err != nil {
				c.logger.Error().Err(err).Msg("Failed to process receipt event")
			}
		}

		c.sendWebhookEvent("receipt", map[string]interface{}{
			"session_id":  c.sessionID,
			"message_ids": v.MessageIDs,
			"chat":        v.Chat.String(),
			"sender":      v.Sender.String(),
			"type":        string(v.Type),
			"timestamp":   v.Timestamp.Unix(),
		})
	case *events.Presence:
		c.logger.Debug().Str("from", v.From.String()).Msg("Presence update")
		c.sendWebhookEvent("presence", map[string]interface{}{
			"session_id": c.sessionID,
			"from":       v.From.String(),
			"timestamp":  time.Now().Unix(),
		})
	case *events.ChatPresence:
		c.logger.Debug().Str("chat", v.Chat.String()).Str("state", string(v.State)).Msg("Chat presence")
		c.sendWebhookEvent("chat_presence", map[string]interface{}{
			"session_id": c.sessionID,
			"chat":       v.Chat.String(),
			"state":      string(v.State),
			"timestamp":  time.Now().Unix(),
		})
	case *events.UndecryptableMessage:
		c.logger.Warn().Str("from", v.Info.Sender.String()).Str("message_id", v.Info.ID).Msg("Undecryptable message")
		c.sendWebhookEvent("undecryptable_message", map[string]interface{}{
			"session_id": c.sessionID,
			"message_id": v.Info.ID,
			"from":       v.Info.Sender.String(),
			"chat":       v.Info.Chat.String(),
			"timestamp":  v.Info.Timestamp.Unix(),
		})
	case *events.GroupInfo:
		c.logger.Info().Str("group", v.JID.String()).Msg("Group info updated")
		c.sendWebhookEvent("group_info", map[string]interface{}{
			"session_id": c.sessionID,
			"group":      v.JID.String(),
			"name":       v.Name,
			"topic":      v.Topic,
			"timestamp":  time.Now().Unix(),
		})
	case *events.JoinedGroup:
		c.logger.Info().Str("group", v.JID.String()).Msg("Joined group")
		c.sendWebhookEvent("joined_group", map[string]interface{}{
			"session_id": c.sessionID,
			"group":      v.JID.String(),
			"timestamp":  time.Now().Unix(),
		})
	case *events.Picture:
		c.logger.Info().Str("jid", v.JID.String()).Msg("Profile picture updated")
		c.sendWebhookEvent("picture", map[string]interface{}{
			"session_id": c.sessionID,
			"jid":        v.JID.String(),
			"picture_id": v.PictureID,
			"timestamp":  time.Now().Unix(),
		})
	case *events.CallOffer:
		c.logger.Info().Str("from", v.BasicCallMeta.From.String()).Str("call_id", v.BasicCallMeta.CallID).Msg("Call offer received")
		c.sendWebhookEvent("call_offer", map[string]interface{}{
			"session_id": c.sessionID,
			"call_id":    v.BasicCallMeta.CallID,
			"from":       v.BasicCallMeta.From.String(),
			"timestamp":  v.BasicCallMeta.Timestamp.Unix(),
		})
	case *events.CallAccept:
		c.logger.Info().Str("from", v.BasicCallMeta.From.String()).Str("call_id", v.BasicCallMeta.CallID).Msg("Call accepted")
		c.sendWebhookEvent("call_accept", map[string]interface{}{
			"session_id": c.sessionID,
			"call_id":    v.BasicCallMeta.CallID,
			"from":       v.BasicCallMeta.From.String(),
			"timestamp":  v.BasicCallMeta.Timestamp.Unix(),
		})
	case *events.CallTerminate:
		c.logger.Info().Str("from", v.BasicCallMeta.From.String()).Str("call_id", v.BasicCallMeta.CallID).Msg("Call terminated")
		c.sendWebhookEvent("call_terminate", map[string]interface{}{
			"session_id": c.sessionID,
			"call_id":    v.BasicCallMeta.CallID,
			"from":       v.BasicCallMeta.From.String(),
			"timestamp":  v.BasicCallMeta.Timestamp.Unix(),
			"reason":     v.Reason,
		})
	case *events.AppState:
		c.logger.Debug().Msg("App state sync")
		c.sendWebhookEvent("app_state", map[string]interface{}{
			"session_id": c.sessionID,
			"timestamp":  time.Now().Unix(),
		})
	case *events.HistorySync:
		c.logger.Info().Int("conversations", len(v.Data.Conversations)).Msg("History sync")
		c.sendWebhookEvent("history_sync", map[string]interface{}{
			"session_id":    c.sessionID,
			"conversations": len(v.Data.Conversations),
			"timestamp":     time.Now().Unix(),
		})

	}
}

func (c *MyClient) Connect() error {
	c.logger.Info().Msg("Connecting to WhatsApp")

	if c.IsConnected() {
		return fmt.Errorf("client is already connected")
	}

	return c.client.Connect()
}

func (c *MyClient) Disconnect() {
	c.logger.Info().Msg("Disconnecting from WhatsApp")
	c.client.Disconnect()

	c.mu.Lock()
	c.isConnected = false
	c.mu.Unlock()
}

func (c *MyClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isConnected && c.client.IsConnected()
}

func (c *MyClient) IsLoggedIn() bool {
	return c.client.IsLoggedIn()
}

func (c *MyClient) GetJID() types.JID {
	if c.client.Store.ID == nil {
		return types.EmptyJID
	}
	return *c.client.Store.ID
}

func (c *MyClient) GetPushName() string {
	return c.client.Store.PushName
}

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

func (c *MyClient) RemoveEventHandler(id uint32) {
	c.client.RemoveEventHandler(id)
}

func (c *MyClient) GetClient() *whatsmeow.Client {
	return c.client
}

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

func (c *MyClient) PairPhone(phoneNumber string) error {
	c.logger.Info().Str("phone", phoneNumber).Msg("Starting phone pairing")

	if c.IsLoggedIn() {
		return fmt.Errorf("client is already logged in")
	}

	return fmt.Errorf("phone pairing not implemented yet")
}

func (c *MyClient) Logout() error {
	c.logger.Info().Msg("Logging out from WhatsApp")

	if !c.IsLoggedIn() {
		return fmt.Errorf("client is not logged in")
	}

	err := c.client.Logout(context.Background())
	if err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}

	c.sendWebhookEvent("logout", map[string]interface{}{
		"logged_out_at": time.Now(),
	})

	return nil
}
