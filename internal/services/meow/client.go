package meow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/felipe/zemeow/internal/models"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/felipe/zemeow/internal/services/message"
	"github.com/google/uuid"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

type WebhookEvent struct {
	SessionID    string      `json:"session_id"`
	Event        string      `json:"event"`
	Data         interface{} `json:"data"`
	Timestamp    time.Time   `json:"timestamp"`
	// Campos adicionais para suporte a payload bruto
	RawEventData interface{} `json:"raw_event_data,omitempty"` // Dados originais do evento whatsmeow
	EventType    string      `json:"event_type,omitempty"`     // Tipo específico do evento (ex: "message", "receipt")
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
	// Send raw event only (zero serialization)
	c.sendWebhookEventRaw(evt)
	
	// Process event locally for persistence and internal handling
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
	case *events.Disconnected:
		c.mu.Lock()
		c.isConnected = false
		c.lastActivity = time.Now()
		c.mu.Unlock()

		c.logger.Info().Msg("Disconnected from WhatsApp")
	case *events.LoggedOut:
		c.logger.Info().Msg("Logged out from WhatsApp")
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
	case *events.QR:
		c.logger.Info().Msg("QR code received")
	case *events.PairSuccess:
		jid := v.ID.String()
		c.logger.Info().Str("jid", jid).Str("business_name", v.BusinessName).Str("platform", v.Platform).Msg("QR Pair Success")

		c.mu.RLock()
		callback := c.onPairSuccess
		c.mu.RUnlock()

		if callback != nil {
			callback(c.sessionID, jid)
		}
	case *events.StreamError:
		c.logger.Error().Str("code", v.Code).Msg("Stream error")
	case *events.ConnectFailure:
		c.mu.Lock()
		c.reconnections++
		c.mu.Unlock()

		c.logger.Error().Int("reason", int(v.Reason)).Msg("Connection failed")
	case *events.Receipt:
		c.logger.Debug().Str("message_id", v.MessageIDs[0]).Str("type", string(v.Type)).Msg("Message receipt")

		if c.messagePersistence != nil {
			err := c.messagePersistence.ProcessReceiptEvent(c.sessionUUID, v)
			if err != nil {
				c.logger.Error().Err(err).Msg("Failed to process receipt event")
			}
		}
	case *events.Presence:
		c.logger.Debug().Str("from", v.From.String()).Msg("Presence update")
	case *events.ChatPresence:
		c.logger.Debug().Str("chat", v.Chat.String()).Str("state", string(v.State)).Msg("Chat presence")
	case *events.UndecryptableMessage:
		c.logger.Warn().Str("from", v.Info.Sender.String()).Str("message_id", v.Info.ID).Msg("Undecryptable message")
	case *events.DeleteForMe:
		c.logger.Info().Str("message_id", v.MessageID).Msg("Message deleted for me")
	case *events.GroupInfo:
		c.logger.Info().Str("group", v.JID.String()).Msg("Group info updated")
	case *events.JoinedGroup:
		c.logger.Info().Str("group", v.JID.String()).Msg("Joined group")
	case *events.Picture:
		c.logger.Info().Str("jid", v.JID.String()).Msg("Profile picture updated")
	case *events.CallOfferNotice:
		c.logger.Info().Str("from", v.From.String()).Str("call_id", v.CallID).Msg("Call offer notice")
	case *events.CallTerminate:
		c.logger.Info().Str("from", v.From.String()).Str("call_id", v.CallID).Msg("Call terminated")
	case *events.Contact:
		c.logger.Info().Str("jid", v.JID.String()).Msg("Contact updated")
	case *events.BlocklistChange:
		c.logger.Info().Msg("Blocklist changed")
	case *events.AppState:
		c.logger.Debug().Msg("App state sync")
	case *events.AppStateSyncComplete:
		c.logger.Info().Msg("App state sync complete")
	case *events.KeepAliveTimeout:
		c.logger.Warn().Msg("Keep alive timeout")
	case *events.KeepAliveRestored:
		c.logger.Info().Msg("Keep alive restored")
	case *events.NewsletterJoin:
		c.logger.Info().Msg("Joined newsletter")
	case *events.NewsletterLeave:
		c.logger.Info().Msg("Left newsletter")
	case *events.HistorySync:
		c.logger.Info().Int("conversations", len(v.Data.Conversations)).Msg("History sync")
	case *events.OfflineSyncPreview:
		c.logger.Info().Msg("Offline sync preview")
	case *events.OfflineSyncCompleted:
		c.logger.Info().Msg("Offline sync completed")
	case *events.TemporaryBan:
		c.logger.Warn().Msg("Temporary ban")
	case *events.ClientOutdated:
		c.logger.Error().Msg("Client outdated")
	case *events.CATRefreshError:
		c.logger.Error().Msg("CAT refresh error")
	case *events.StreamReplaced:
		c.logger.Warn().Msg("Stream replaced")
	case *events.PermanentDisconnect:
		c.logger.Error().Msg("Permanent disconnect")
	case *events.ManualLoginReconnect:
		c.logger.Info().Msg("Manual login reconnect")
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

// sendWebhookEventRaw envia evento com payload bruto da whatsmeow
func (c *MyClient) sendWebhookEventRaw(evt interface{}) {
	if c.webhookChan == nil {
		return
	}

	eventType := c.getEventTypeName(evt)
	
	// Send to webhook channel with raw data
	webhookEvent := WebhookEvent{
		SessionID:    c.sessionID,
		Event:        "raw_event",
		Data:         nil, // Data will be in RawEventData
		Timestamp:    time.Now(),
		RawEventData: evt,
		EventType:    eventType,
	}

	select {
	case c.webhookChan <- webhookEvent:
		c.logger.Debug().Str("event_type", eventType).Msg("Raw webhook event sent")
	default:
		c.logger.Warn().Str("event_type", eventType).Msg("Webhook channel full, raw event dropped")
	}
}

// getEventTypeName returns the name of the event type for identification
func (c *MyClient) getEventTypeName(evt interface{}) string {
	switch evt.(type) {
	case *events.Connected:
		return "connected"
	case *events.Disconnected:
		return "disconnected"
	case *events.LoggedOut:
		return "logged_out"
	case *events.ConnectFailure:
		return "connect_failure"
	case *events.Message:
		return "message"
	case *events.Receipt:
		return "receipt"
	case *events.UndecryptableMessage:
		return "undecryptable_message"
	case *events.DeleteForMe:
		return "delete_for_me"
	case *events.HistorySync:
		return "history_sync"
	case *events.OfflineSyncPreview:
		return "offline_sync_preview"
	case *events.OfflineSyncCompleted:
		return "offline_sync_completed"
	case *events.Presence:
		return "presence"
	case *events.ChatPresence:
		return "chat_presence"
	case *events.QR:
		return "qr"
	case *events.QRScannedWithoutMultidevice:
		return "qr_scanned_without_multidevice"
	case *events.JoinedGroup:
		return "joined_group"
	case *events.GroupInfo:
		return "group_info"
	case *events.CallOffer:
		return "call_offer"
	case *events.CallOfferNotice:
		return "call_offer_notice"
	case *events.CallAccept:
		return "call_accept"
	case *events.CallPreAccept:
		return "call_pre_accept"
	case *events.CallReject:
		return "call_reject"
	case *events.CallTerminate:
		return "call_terminate"
	case *events.CallTransport:
		return "call_transport"
	case *events.CallRelayLatency:
		return "call_relay_latency"
	case *events.UnknownCallEvent:
		return "unknown_call_event"
	case *events.Contact:
		return "contact"
	case *events.Picture:
		return "picture"
	case *events.BlocklistChange:
		return "blocklist"
	case *events.BlocklistChangeAction:
		return "blocklist_change"
	case *events.BusinessName:
		return "business_name"
	case *events.UserAbout:
		return "user_about"
	case *events.UserStatusMute:
		return "user_status_mute"
	case *events.PushName:
		return "push_name"
	case *events.Archive:
		return "archive"
	case *events.Pin:
		return "pin"
	case *events.Mute:
		return "mute"
	case *events.ClearChat:
		return "clear_chat"
	case *events.DeleteChat:
		return "delete_chat"
	case *events.MarkChatAsRead:
		return "mark_chat_as_read"
	case *events.AppState:
		return "app_state"
	case *events.AppStateSyncComplete:
		return "app_state_sync_complete"
	case *events.KeepAliveTimeout:
		return "keepalive_timeout"
	case *events.KeepAliveRestored:
		return "keepalive_restored"
	case *events.NewsletterJoin:
		return "newsletter_join"
	case *events.NewsletterLeave:
		return "newsletter_leave"
	case *events.NewsletterMuteChange:
		return "newsletter_mute_change"
	case *events.NewsletterLiveUpdate:
		return "newsletter_live_update"
	case *events.NewsletterMessageMeta:
		return "newsletter_message_meta"
	case *events.FBMessage:
		return "fb_message"
	case *events.Star:
		return "star"
	case *events.TemporaryBan:
		return "temporary_ban"
	case *events.ClientOutdated:
		return "client_outdated"
	case *events.CATRefreshError:
		return "cat_refresh_error"
	case *events.StreamError:
		return "stream_error"
	case *events.StreamReplaced:
		return "stream_replaced"
	case *events.PermanentDisconnect:
		return "permanent_disconnect"
	case *events.ManualLoginReconnect:
		return "manual_login_reconnect"
	case *events.PrivacySettings:
		return "privacy_settings"
	case *events.UnarchiveChatsSetting:
		return "unarchive_chats_setting"
	case *events.PushNameSetting:
		return "push_name_setting"
	case *events.IdentityChange:
		return "identity_change"
	case *events.LabelEdit:
		return "label_edit"
	case *events.LabelAssociationChat:
		return "label_association_chat"
	case *events.LabelAssociationMessage:
		return "label_association_message"
	case *events.MediaRetry:
		return "media_retry"
	case *events.MediaRetryError:
		return "media_retry_error"
	case *events.DecryptFailMode:
		return "decrypt_fail_mode"
	default:
		return "unknown"
	}
}
