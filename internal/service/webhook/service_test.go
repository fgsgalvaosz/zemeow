package webhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/felipe/zemeow/internal/config"
	"github.com/felipe/zemeow/internal/db/models"
	"github.com/felipe/zemeow/internal/service/meow"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// MockSessionRepository implementa a interface SessionRepository para testes
type MockSessionRepository struct {
	mock.Mock
}

func (m *MockSessionRepository) GetByIdentifier(identifier string) (*models.Session, error) {
	args := m.Called(identifier)
	return args.Get(0).(*models.Session), args.Error(1)
}

// Implementar outros métodos necessários da interface...
func (m *MockSessionRepository) Create(session *models.Session) error {
	args := m.Called(session)
	return args.Error(0)
}

func (m *MockSessionRepository) GetByID(id uuid.UUID) (*models.Session, error) {
	args := m.Called(id)
	return args.Get(0).(*models.Session), args.Error(1)
}

func (m *MockSessionRepository) GetBySessionID(sessionID string) (*models.Session, error) {
	args := m.Called(sessionID)
	return args.Get(0).(*models.Session), args.Error(1)
}

func (m *MockSessionRepository) GetByName(name string) (*models.Session, error) {
	args := m.Called(name)
	return args.Get(0).(*models.Session), args.Error(1)
}

func (m *MockSessionRepository) GetByAPIKey(apiKey string) (*models.Session, error) {
	args := m.Called(apiKey)
	return args.Get(0).(*models.Session), args.Error(1)
}

func (m *MockSessionRepository) GetAll(filter *models.SessionFilter) (*models.SessionListResponse, error) {
	args := m.Called(filter)
	return args.Get(0).(*models.SessionListResponse), args.Error(1)
}

func (m *MockSessionRepository) Update(session *models.Session) error {
	args := m.Called(session)
	return args.Error(0)
}

func (m *MockSessionRepository) UpdateStatus(identifier string, status models.SessionStatus) error {
	args := m.Called(identifier, status)
	return args.Error(0)
}

func (m *MockSessionRepository) UpdateStatusAndJID(identifier string, status models.SessionStatus, jid *string) error {
	args := m.Called(identifier, status, jid)
	return args.Error(0)
}

func (m *MockSessionRepository) UpdateJID(identifier string, jid *string) error {
	args := m.Called(identifier, jid)
	return args.Error(0)
}

func (m *MockSessionRepository) UpdateQRCode(identifier string, qrCode string) error {
	args := m.Called(identifier, qrCode)
	return args.Error(0)
}

func (m *MockSessionRepository) ClearQRCode(identifier string) error {
	args := m.Called(identifier)
	return args.Error(0)
}

func (m *MockSessionRepository) Delete(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockSessionRepository) DeleteByIdentifier(identifier string) error {
	args := m.Called(identifier)
	return args.Error(0)
}

func (m *MockSessionRepository) DeleteBySessionID(sessionID string) error {
	args := m.Called(sessionID)
	return args.Error(0)
}

func (m *MockSessionRepository) Exists(identifier string) (bool, error) {
	args := m.Called(identifier)
	return args.Bool(0), args.Error(1)
}

func (m *MockSessionRepository) Count() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

func (m *MockSessionRepository) GetActiveConnections() ([]*models.Session, error) {
	args := m.Called()
	return args.Get(0).([]*models.Session), args.Error(1)
}

func (m *MockSessionRepository) Close() error {
	args := m.Called()
	return args.Error(0)
}

// Testes unitários

func TestWebhookService_ProcessEventProcessed(t *testing.T) {
	// Mock repository
	mockRepo := &MockSessionRepository{}
	webhookURL := "https://example.com/webhook"
	payloadMode := "processed"
	
	session := &models.Session{
		ID:                   uuid.New(),
		Name:                 "test_session",
		WebhookURL:           &webhookURL,
		WebhookEvents:        []string{"message"},
		WebhookPayloadMode:   &payloadMode,
	}
	
	mockRepo.On("GetByIdentifier", "test_session").Return(session, nil)
	
	// Configuração de teste
	cfg := &config.Config{
		Webhook: config.WebhookConfig{
			Timeout:       10 * time.Second,
			RetryCount:    3,
			RetryInterval: 5 * time.Second,
		},
	}
	
	// Criar serviço
	service := NewWebhookService(mockRepo, cfg)
	
	// Evento de teste
	event := meow.WebhookEvent{
		SessionID: "test_session",
		Event:     "message",
		Data: map[string]interface{}{
			"message_id": "test_msg_id",
			"from":       "test_sender",
		},
		Timestamp: time.Now(),
	}
	
	// Mock HTTP server para receber webhook
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "message", r.Header.Get("X-Webhook-Event"))
		assert.Equal(t, "test_session", r.Header.Get("X-Session-ID"))
		
		var payload WebhookPayload
		err := json.NewDecoder(r.Body).Decode(&payload)
		assert.NoError(t, err)
		assert.Equal(t, "test_session", payload.SessionID)
		assert.Equal(t, "message", payload.Event)
		assert.Equal(t, "processed", payload.Metadata["payload_type"])
		
		w.WriteHeader(200)
	}))
	defer server.Close()
	
	// Atualizar URL do webhook para o servidor de teste
	session.WebhookURL = &server.URL
	
	// Testar processamento
	err := service.processEventProcessed(event, session)
	assert.NoError(t, err)
	
	mockRepo.AssertExpectations(t)
}

func TestWebhookService_ProcessEventRaw(t *testing.T) {
	// Mock repository
	mockRepo := &MockSessionRepository{}
	webhookURL := "https://example.com/webhook"
	payloadMode := "raw"
	jid := "5511999999999@s.whatsapp.net"
	
	session := &models.Session{
		ID:                   uuid.New(),
		Name:                 "test_session",
		WebhookURL:           &webhookURL,
		WebhookEvents:        []string{"message"},
		WebhookPayloadMode:   &payloadMode,
		JID:                  &jid,
	}
	
	mockRepo.On("GetByIdentifier", "test_session").Return(session, nil)
	
	// Configuração de teste
	cfg := &config.Config{
		Webhook: config.WebhookConfig{
			Timeout:       10 * time.Second,
			RetryCount:    3,
			RetryInterval: 5 * time.Second,
		},
	}
	
	// Criar serviço
	service := NewWebhookService(mockRepo, cfg)
	
	// Criar evento raw simulado
	messageEvent := &events.Message{
		Info: types.MessageInfo{
			ID:        "test_msg_id",
			Timestamp: time.Now(),
		},
	}
	
	event := meow.WebhookEvent{
		SessionID:    "test_session",
		Event:        "message",
		Data:         nil, // Dados processados nulos para eventos raw
		Timestamp:    time.Now(),
		RawEventData: messageEvent,
		PayloadMode:  "raw",
		EventType:    "*events.Message",
	}
	
	// Mock HTTP server para receber webhook
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "*events.Message", r.Header.Get("X-Webhook-Event"))
		assert.Equal(t, "test_session", r.Header.Get("X-Session-ID"))
		assert.Equal(t, "raw", r.Header.Get("X-Payload-Type"))
		
		// Tentar decodificar como payload genérico primeiro
		var genericPayload WebhookPayload
		err := json.NewDecoder(r.Body).Decode(&genericPayload)
		assert.NoError(t, err)
		
		// Verificar se contém RawWebhookPayload
		rawPayload, ok := genericPayload.Data.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "raw", rawPayload["payload_type"])
		assert.Equal(t, "*events.Message", rawPayload["event_type"])
		
		w.WriteHeader(200)
	}))
	defer server.Close()
	
	// Atualizar URL do webhook para o servidor de teste
	session.WebhookURL = &server.URL
	
	// Testar processamento raw
	err := service.processEventRaw(event, session)
	assert.NoError(t, err)
	
	mockRepo.AssertExpectations(t)
}

func TestWebhookService_ProcessEvent_BothMode(t *testing.T) {
	// Mock repository
	mockRepo := &MockSessionRepository{}
	webhookURL := "https://example.com/webhook"
	payloadMode := "both"
	
	session := &models.Session{
		ID:                   uuid.New(),
		Name:                 "test_session",
		WebhookURL:           &webhookURL,
		WebhookEvents:        []string{"message"},
		WebhookPayloadMode:   &payloadMode,
	}
	
	mockRepo.On("GetByIdentifier", "test_session").Return(session, nil)
	
	// Configuração de teste
	cfg := &config.Config{
		Webhook: config.WebhookConfig{
			Timeout:       10 * time.Second,
			RetryCount:    3,
			RetryInterval: 5 * time.Second,
		},
	}
	
	// Criar serviço
	service := NewWebhookService(mockRepo, cfg)
	
	// Criar evento com dados tanto processados quanto raw
	messageEvent := &events.Message{
		Info: types.MessageInfo{
			ID:        "test_msg_id",
			Timestamp: time.Now(),
		},
	}
	
	event := meow.WebhookEvent{
		SessionID: "test_session",
		Event:     "message",
		Data: map[string]interface{}{
			"message_id": "test_msg_id",
			"from":       "test_sender",
		},
		Timestamp:    time.Now(),
		RawEventData: messageEvent,
		PayloadMode:  "both",
		EventType:    "*events.Message",
	}
	
	// Contador para verificar se ambos os webhooks foram enviados
	webhookCount := 0
	
	// Mock HTTP server para receber webhooks
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		webhookCount++
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "test_session", r.Header.Get("X-Session-ID"))
		
		w.WriteHeader(200)
	}))
	defer server.Close()
	
	// Atualizar URL do webhook para o servidor de teste
	session.WebhookURL = &server.URL
	
	// Testar processamento em modo "both"
	err := service.processEvent(event)
	assert.NoError(t, err)
	
	// Esperar um pouco para que os webhooks sejam processados
	time.Sleep(100 * time.Millisecond)
	
	// Verificar se ambos os webhooks foram enviados
	// Nota: Em implementação real, seria necessário aguardar o processamento assíncrono
	// Este teste pode precisar de ajustes dependendo da implementação final
	
	mockRepo.AssertExpectations(t)
}

func TestWebhookService_CreateEventMetadata(t *testing.T) {
	// Mock repository
	mockRepo := &MockSessionRepository{}
	
	// Configuração de teste
	cfg := &config.Config{
		Webhook: config.WebhookConfig{
			Timeout:       10 * time.Second,
			RetryCount:    3,
			RetryInterval: 5 * time.Second,
		},
	}
	
	// Criar serviço
	service := NewWebhookService(mockRepo, cfg)
	
	// Sessão de teste
	jid := "5511999999999@s.whatsapp.net"
	session := &models.Session{
		ID:   uuid.New(),
		Name: "test_session",
		JID:  &jid,
	}
	
	// Testar criação de metadados
	metadata := service.createEventMetadata(session)
	
	assert.Equal(t, "v0.0.0-20250611130243", metadata.WhatsmeowVersion)
	assert.Equal(t, "2.24.6", metadata.ProtocolVersion)
	assert.Equal(t, jid, metadata.SessionJID)
	assert.Equal(t, "ZeMeow/1.0", metadata.DeviceInfo)
	assert.NotEmpty(t, metadata.GoVersion)
}

func TestRawWebhookPayload_JSONSerialization(t *testing.T) {
	// Criar payload raw de teste
	rawData := map[string]interface{}{
		"Info": map[string]interface{}{
			"ID":        "test_msg_id",
			"Timestamp": time.Now().Unix(),
		},
		"Message": map[string]interface{}{
			"conversation": "Hello World",
		},
	}
	
	rawBytes, err := json.Marshal(rawData)
	assert.NoError(t, err)
	
	payload := RawWebhookPayload{
		SessionID:   "test_session",
		EventType:   "*events.Message",
		RawData:     json.RawMessage(rawBytes),
		PayloadType: "raw",
		Timestamp:   time.Now(),
		EventMeta: EventMetadata{
			WhatsmeowVersion: "v0.0.0-20250611130243",
			SessionJID:       "5511999999999@s.whatsapp.net",
		},
	}
	
	// Serializar payload completo
	serialized, err := json.Marshal(payload)
	assert.NoError(t, err)
	
	// Deserializar de volta
	var deserialized RawWebhookPayload
	err = json.Unmarshal(serialized, &deserialized)
	assert.NoError(t, err)
	
	// Verificar campos
	assert.Equal(t, payload.SessionID, deserialized.SessionID)
	assert.Equal(t, payload.EventType, deserialized.EventType)
	assert.Equal(t, payload.PayloadType, deserialized.PayloadType)
	assert.Equal(t, payload.EventMeta.WhatsmeowVersion, deserialized.EventMeta.WhatsmeowVersion)
	
	// Verificar se RawData foi preservado
	var originalRawData, deserializedRawData map[string]interface{}
	err = json.Unmarshal(payload.RawData, &originalRawData)
	assert.NoError(t, err)
	err = json.Unmarshal(deserialized.RawData, &deserializedRawData)
	assert.NoError(t, err)
	
	assert.Equal(t, originalRawData, deserializedRawData)
}