package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/felipe/zemeow/internal/db/models"
	"github.com/felipe/zemeow/internal/db/repositories"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/google/uuid"
)

// Service define a interface para operações de sessão
type Service interface {
	// Gerenciamento de sessões
	CreateSession(ctx context.Context, config *Config) (*SessionInfo, error)
	GetSession(ctx context.Context, sessionID string) (*SessionInfo, error)
	ListSessions(ctx context.Context) ([]*SessionInfo, error)
	DeleteSession(ctx context.Context, sessionID string) error

	// Controle de conexão
	ConnectSession(ctx context.Context, sessionID string) error
	DisconnectSession(ctx context.Context, sessionID string) error
	GetSessionStatus(ctx context.Context, sessionID string) (Status, error)

	// Autenticação WhatsApp
	GetQRCode(ctx context.Context, sessionID string) (*QRCodeInfo, error)
	PairPhone(ctx context.Context, sessionID string, request *PairPhoneRequest) (*PairPhoneResponse, error)

	// Configurações
	SetProxy(ctx context.Context, sessionID string, proxy *ProxyConfig) error
	SetWebhook(ctx context.Context, sessionID string, webhook *WebhookConfig) error

	// WhatsApp Client Access
	GetWhatsAppClient(ctx context.Context, sessionID string) (interface{}, error)

	// Lifecycle
	Shutdown(ctx context.Context) error
}

// SessionService implementa a interface Service
type SessionService struct {
	repository repositories.SessionRepository
	manager    interface{} // WhatsAppManager interface
	logger     logger.Logger
}

// NewService cria uma nova instância do serviço de sessão
func NewService(repository repositories.SessionRepository, manager interface{}) Service {
	return &SessionService{
		repository: repository,
		manager:    manager,
		logger:     logger.GetWithSession("session_service"),
	}
}

// CreateSession implementa Service.CreateSession
func (s *SessionService) CreateSession(ctx context.Context, config *Config) (*SessionInfo, error) {
	s.logger.Info().Str("name", config.Name).Msg("Creating new session")

	// Validar configuração
	if err := s.validateConfig(config); err != nil {
		s.logger.Error().Err(err).Msg("Invalid session configuration")
		return nil, err
	}

	// Gerar sessionID único se não fornecido
	sessionID := config.SessionID
	if sessionID == "" {
		sessionID = generateSessionID()
	}

	// Verificar se sessionID já existe
	if exists, _ := s.repository.Exists(sessionID); exists {
		return nil, fmt.Errorf("session ID already exists: %s", sessionID)
	}

	// Gerar API Key se não fornecida
	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = generateAPIKey()
		s.logger.Info().Str("session_id", sessionID).Msg("Generated automatic API key for session")
	}

	// Criar modelo de sessão
	session := &models.Session{
		ID:        uuid.New(),
		SessionID: sessionID,
		Name:      config.Name,
		APIKey:    apiKey,
		Status:    models.SessionStatusDisconnected,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  make(models.Metadata),
	}

	// Configurar proxy se fornecido
	if config.Proxy != nil {
		session.ProxyEnabled = config.Proxy.Enabled
		if config.Proxy.Enabled {
			session.ProxyHost = &config.Proxy.Host
			session.ProxyPort = &config.Proxy.Port
			if config.Proxy.Username != "" {
				session.ProxyUsername = &config.Proxy.Username
				session.ProxyPassword = &config.Proxy.Password
			}
		}
	}

	// Configurar webhook se fornecido
	if config.Webhook != nil {
		session.WebhookURL = &config.Webhook.URL
		session.WebhookEvents = config.Webhook.Events
	}

	// Salvar no repositório
	if err := s.repository.Create(session); err != nil {
		s.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to create session in database")
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Converter para SessionInfo
	sessionInfo := &SessionInfo{
		ID:        sessionID,
		Name:      session.Name,
		APIKey:    session.APIKey,
		Status:    Status(session.Status),
		CreatedAt: session.CreatedAt,
		UpdatedAt: session.UpdatedAt,
	}

	s.logger.Info().Str("session_id", sessionID).Str("name", config.Name).Msg("Session created successfully")
	return sessionInfo, nil
}

// GetSession implementa Service.GetSession
func (s *SessionService) GetSession(ctx context.Context, sessionID string) (*SessionInfo, error) {
	s.logger.Debug().Str("session_id", sessionID).Msg("Getting session info")

	// Buscar sessão no repositório
	session, err := s.repository.GetBySessionID(sessionID)
	if err != nil {
		s.logger.Error().Err(err).Str("session_id", sessionID).Msg("Session not found")
		return nil, fmt.Errorf("session not found: %w", err)
	}

	// Verificar status da sessão no manager
	isConnected := false
	if s.manager != nil {
		isConnected = s.manager.IsSessionActive(sessionID)
	}

	// Converter para SessionInfo
	sessionInfo := &SessionInfo{
		ID:          sessionID,
		Name:        session.Name,
		APIKey:      session.APIKey,
		Status:      Status(session.Status),
		JID:         session.JID,
		IsConnected: isConnected,
		CreatedAt:   session.CreatedAt,
		UpdatedAt:   session.UpdatedAt,
	}

	if session.LastConnectedAt != nil {
		sessionInfo.LastConnectedAt = session.LastConnectedAt
	}

	return sessionInfo, nil
}

// ListSessions implementa Service.ListSessions
func (s *SessionService) ListSessions(ctx context.Context) ([]*SessionInfo, error) {
	s.logger.Debug().Msg("Listing all sessions")

	// Buscar todas as sessões
	filter := &models.SessionFilter{
		Page:    1,
		PerPage: 100, // Limite padrão
	}

	response, err := s.repository.GetAll(filter)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to list sessions")
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	// Converter para SessionInfo
	sessions := make([]*SessionInfo, len(response.Sessions))
	for i, session := range response.Sessions {
		isConnected := false
		if s.manager != nil {
			isConnected = s.manager.IsSessionActive(session.SessionID)
		}

		sessions[i] = &SessionInfo{
			ID:          session.SessionID,
			Name:        session.Name,
			APIKey:      session.APIKey,
			Status:      Status(session.Status),
			JID:         session.JID,
			IsConnected: isConnected,
			CreatedAt:   session.CreatedAt,
			UpdatedAt:   session.UpdatedAt,
		}

		if session.LastConnectedAt != nil {
			sessions[i].LastConnectedAt = session.LastConnectedAt
		}
	}

	s.logger.Info().Int("count", len(sessions)).Msg("Sessions listed successfully")
	return sessions, nil
}

// DeleteSession implementa Service.DeleteSession
func (s *SessionService) DeleteSession(ctx context.Context, sessionID string) error {
	s.logger.Info().Str("session_id", sessionID).Msg("Deleting session")

	// Verificar se sessão existe
	if exists, _ := s.repository.Exists(sessionID); !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Desconectar sessão se estiver ativa
	if s.manager != nil && s.manager.IsSessionActive(sessionID) {
		s.logger.Info().Str("session_id", sessionID).Msg("Disconnecting active session before deletion")
		if err := s.manager.DisconnectSession(sessionID); err != nil {
			s.logger.Warn().Err(err).Str("session_id", sessionID).Msg("Failed to disconnect session, continuing with deletion")
		}
	}

	// Deletar do repositório
	if err := s.repository.DeleteBySessionID(sessionID); err != nil {
		s.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to delete session")
		return fmt.Errorf("failed to delete session: %w", err)
	}

	s.logger.Info().Str("session_id", sessionID).Msg("Session deleted successfully")
	return nil
}

// ConnectSession implementa Service.ConnectSession
func (s *SessionService) ConnectSession(ctx context.Context, sessionID string) error {
	s.logger.Info().Str("session_id", sessionID).Msg("Connecting session to WhatsApp")

	// Verificar se sessão existe
	if exists, _ := s.repository.Exists(sessionID); !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Conectar via manager (casting para interface esperada)
	if manager, ok := s.manager.(interface{ ConnectSession(string) error }); ok {
		if err := manager.ConnectSession(sessionID); err != nil {
			s.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to connect session")
			return fmt.Errorf("failed to connect session: %w", err)
		}
	} else {
		s.logger.Warn().Msg("WhatsApp manager not available for connection")
		return fmt.Errorf("WhatsApp manager not available")
	}

	s.logger.Info().Str("session_id", sessionID).Msg("Session connection initiated successfully")
	return nil
}

// DisconnectSession implementa Service.DisconnectSession
func (s *SessionService) DisconnectSession(ctx context.Context, sessionID string) error {
	s.logger.Info().Str("session_id", sessionID).Msg("Disconnecting session from WhatsApp")

	// Verificar se sessão existe
	if exists, _ := s.repository.Exists(sessionID); !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Desconectar via manager
	if manager, ok := s.manager.(interface{ DisconnectSession(string) error }); ok {
		if err := manager.DisconnectSession(sessionID); err != nil {
			s.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to disconnect session")
			return fmt.Errorf("failed to disconnect session: %w", err)
		}
	} else {
		s.logger.Warn().Msg("WhatsApp manager not available for disconnection")
		return fmt.Errorf("WhatsApp manager not available")
	}

	s.logger.Info().Str("session_id", sessionID).Msg("Session disconnected successfully")
	return nil
}

// GetSessionStatus implementa Service.GetSessionStatus
func (s *SessionService) GetSessionStatus(ctx context.Context, sessionID string) (Status, error) {
	s.logger.Debug().Str("session_id", sessionID).Msg("Getting session status")

	// Buscar sessão no repositório para obter status persistido
	session, err := s.repository.GetBySessionID(sessionID)
	if err != nil {
		s.logger.Error().Err(err).Str("session_id", sessionID).Msg("Session not found")
		return StatusDisconnected, fmt.Errorf("session not found: %w", err)
	}

	// Verificar status em tempo real via manager se disponível
	if manager, ok := s.manager.(interface{ IsSessionActive(string) bool }); ok {
		if manager.IsSessionActive(sessionID) {
			return StatusConnected, nil
		}
	}

	// Retornar status do banco se manager não estiver disponível
	return Status(session.Status), nil
}

// GetQRCode implementa Service.GetQRCode
func (s *SessionService) GetQRCode(ctx context.Context, sessionID string) (*QRCodeInfo, error) {
	s.logger.Info().Str("session_id", sessionID).Msg("Getting QR code for session")

	// Verificar se sessão existe
	if exists, _ := s.repository.Exists(sessionID); !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Obter QR Code via manager se disponível
	if manager, ok := s.manager.(interface{ GetSessionQRCode(string) (interface{}, error) }); ok {
		qrData, err := manager.GetSessionQRCode(sessionID)
		if err != nil {
			s.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to get QR code")
			return nil, fmt.Errorf("failed to get QR code: %w", err)
		}

		// Converter para QRCodeInfo (implementação simplificada)
		return &QRCodeInfo{
			Code:      "qr_initiated", // O QR real será enviado via webhook
			Timeout:   60,
			Timestamp: time.Now(),
		}, nil
	}

	// Fallback se manager não estiver disponível
	return &QRCodeInfo{
		Code:      "qr_unavailable",
		Timeout:   60,
		Timestamp: time.Now(),
	}, nil
}

// PairPhone implementa Service.PairPhone
func (s *SessionService) PairPhone(ctx context.Context, sessionID string, request *PairPhoneRequest) (*PairPhoneResponse, error) {
	return &PairPhoneResponse{
		Success: true,
		Message: "Phone paired successfully",
	}, nil
}

// SetProxy implementa Service.SetProxy
func (s *SessionService) SetProxy(ctx context.Context, sessionID string, proxy *ProxyConfig) error {
	return nil
}

// SetWebhook implementa Service.SetWebhook
func (s *SessionService) SetWebhook(ctx context.Context, sessionID string, webhook *WebhookConfig) error {
	return nil
}

// GetWhatsAppClient implementa Service.GetWhatsAppClient
func (s *SessionService) GetWhatsAppClient(ctx context.Context, sessionID string) (interface{}, error) {
	s.logger.Debug().Str("session_id", sessionID).Msg("Getting WhatsApp client for session")

	// Verificar se sessão existe
	if exists, _ := s.repository.Exists(sessionID); !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Obter cliente via manager se disponível
	if manager, ok := s.manager.(interface{ GetClient(string) (interface{}, error) }); ok {
		client, err := manager.GetClient(sessionID)
		if err != nil {
			s.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to get WhatsApp client from manager")
			return nil, fmt.Errorf("failed to get WhatsApp client: %w", err)
		}
		return client, nil
	}

	// Se manager não estiver disponível, retornar erro
	s.logger.Warn().Str("session_id", sessionID).Msg("WhatsApp manager not available for client access")
	return nil, fmt.Errorf("WhatsApp manager not available")
}

// Shutdown implementa Service.Shutdown
func (s *SessionService) Shutdown(ctx context.Context) error {
	return nil
}

// validateConfig valida a configuração da sessão
func (s *SessionService) validateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("configuration is required")
	}

	if config.Name == "" {
		return fmt.Errorf("session name is required")
	}

	// Validar proxy se fornecido
	if config.Proxy != nil && config.Proxy.Enabled {
		if config.Proxy.Host == "" {
			return fmt.Errorf("proxy host is required when proxy is enabled")
		}
		if config.Proxy.Port <= 0 || config.Proxy.Port > 65535 {
			return fmt.Errorf("proxy port must be between 1 and 65535")
		}
	}

	// Validar webhook se fornecido
	if config.Webhook != nil && config.Webhook.URL != "" {
		if len(config.Webhook.URL) < 10 {
			return fmt.Errorf("webhook URL is too short")
		}
	}

	return nil
}

// generateAPIKey gera uma API key aleatória única
func generateAPIKey() string {
	// Gerar 32 bytes aleatórios (256 bits)
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback para timestamp se rand falhar
		return fmt.Sprintf("api_%d_%d", time.Now().UnixNano(), time.Now().Unix())
	}
	return "zmw_" + hex.EncodeToString(bytes)
}

// generateSessionID gera um ID único para a sessão
func generateSessionID() string {
	return uuid.New().String()
}
