package meow

import (
	"testing"
	"time"

	"github.com/felipe/zemeow/internal/logger"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func TestMyClient_GetEventTypeName(t *testing.T) {
	// Inicializar logger para teste
	logger.InitSimple("info", false)
	
	// Criar cliente de teste
	sessionID := "test_session"
	sessionUUID := uuid.New()
	deviceStore := &store.Device{}
	webhookChan := make(chan WebhookEvent, 10)
	
	client := NewMyClient(sessionID, sessionUUID, deviceStore, webhookChan, nil)
	
	// Testar diferentes tipos de eventos
	tests := []struct {
		event    interface{}
		expected string
	}{
		{&events.Message{}, "*events.Message"},
		{&events.Receipt{}, "*events.Receipt"},
		{&events.Connected{}, "*events.Connected"},
		{&events.Disconnected{}, "*events.Disconnected"},
		{&events.QR{}, "*events.QR"},
		{&events.PairSuccess{}, "*events.PairSuccess"},
		{&events.StreamError{}, "*events.StreamError"},
		{&events.ConnectFailure{}, "*events.ConnectFailure"},
		{&events.Presence{}, "*events.Presence"},
		{&events.ChatPresence{}, "*events.ChatPresence"},
		{&events.UndecryptableMessage{}, "*events.UndecryptableMessage"},
		{&events.GroupInfo{}, "*events.GroupInfo"},
		{&events.CallOffer{}, "*events.CallOffer"},
		{&events.CallAccept{}, "*events.CallAccept"},
		{&events.CallTerminate{}, "*events.CallTerminate"},
		{&events.AppState{}, "*events.AppState"},
		{&events.HistorySync{}, "*events.HistorySync"},
		{"unknown_type", "string"}, // Tipo desconhecido
	}
	
	for _, test := range tests {
		result := client.getEventTypeName(test.event)
		assert.Equal(t, test.expected, result, "Event type name mismatch for %T", test.event)
	}
}

func TestMyClient_SendWebhookEventRaw(t *testing.T) {
	// Inicializar logger para teste
	logger.InitSimple("info", false)
	
	// Criar cliente de teste
	sessionID := "test_session"
	sessionUUID := uuid.New()
	deviceStore := &store.Device{}
	webhookChan := make(chan WebhookEvent, 10)
	
	client := NewMyClient(sessionID, sessionUUID, deviceStore, webhookChan, nil)
	
	// Criar evento de teste
	messageEvent := &events.Message{
		Info: types.MessageInfo{
			ID:        "test_msg_id",
			Timestamp: time.Now(),
		},
	}
	
	// Enviar evento raw
	client.sendWebhookEventRaw(messageEvent, "message")
	
	// Verificar se o evento foi enviado
	select {
	case webhookEvent := <-webhookChan:
		assert.Equal(t, sessionID, webhookEvent.SessionID)
		assert.Equal(t, "message", webhookEvent.Event)
		assert.Nil(t, webhookEvent.Data) // Data deve ser nil para eventos raw
		assert.Equal(t, messageEvent, webhookEvent.RawEventData)
		assert.Equal(t, "raw", webhookEvent.PayloadMode)
		assert.Equal(t, "*events.Message", webhookEvent.EventType)
	case <-time.After(1 * time.Second):
		t.Fatal("Webhook event not received within timeout")
	}
}

func TestMyClient_BasicProperties(t *testing.T) {
	// Inicializar logger para teste
	logger.InitSimple("info", false)
	
	// Criar cliente de teste
	sessionID := "test_session"
	sessionUUID := uuid.New()
	deviceStore := &store.Device{}
	webhookChan := make(chan WebhookEvent, 10)
	
	client := NewMyClient(sessionID, sessionUUID, deviceStore, webhookChan, nil)
	
	// Testar propriedades básicas
	assert.Equal(t, sessionID, client.sessionID)
	assert.Equal(t, sessionUUID, client.sessionUUID)
	assert.NotNil(t, client.client)
	assert.NotNil(t, client.logger)
	// Não comparar o canal diretamente devido aos tipos chan vs chan<-
	assert.False(t, client.isConnected)
	assert.Equal(t, int64(0), client.messagesReceived)
	assert.Equal(t, int64(0), client.messagesSent)
	assert.Equal(t, int64(0), client.reconnections)
}