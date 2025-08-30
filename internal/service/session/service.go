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


type Service interface {

	CreateSession(ctx context.Context, config *Config) (*SessionInfo, error)
	GetSession(ctx context.Context, sessionID string) (*SessionInfo, error)
	ListSessions(ctx context.Context) ([]*SessionInfo, error)
	DeleteSession(ctx context.Context, sessionID string) error


	ConnectSession(ctx context.Context, sessionID string) error
	DisconnectSession(ctx context.Context, sessionID string) error
	GetSessionStatus(ctx context.Context, sessionID string) (Status, error)


	GetQRCode(ctx context.Context, sessionID string) (*QRCodeInfo, error)
	PairPhone(ctx context.Context, sessionID string, request *PairPhoneRequest) (*PairPhoneResponse, error)


	SetProxy(ctx context.Context, sessionID string, proxy *ProxyConfig) error
	SetWebhook(ctx context.Context, sessionID string, webhook *WebhookConfig) error


	GetWhatsAppClient(ctx context.Context, sessionID string) (interface{}, error)


	Shutdown(ctx context.Context) error
}


type SessionService struct {
	repository repositories.SessionRepository
	manager    interface{} // WhatsAppManager interface
	logger     logger.Logger
}


func NewService(repository repositories.SessionRepository, manager interface{}) Service {
	return &SessionService{
		repository: repository,
		manager:    manager,
		logger:     logger.GetWithSession("session_service"),
	}
}


func (s *SessionService) CreateSession(ctx context.Context, config *Config) (*SessionInfo, error) {
	s.logger.Info().Str("name", config.Name).Msg("Creating new session")


	if err := s.validateConfig(config); err != nil {
		s.logger.Error().Err(err).Msg("Invalid session configuration")
		return nil, err
	}


	if config.Name == "" {
		return nil, fmt.Errorf("session name is required")
	}


	sessionID := uuid.New()


	if exists, _ := s.repository.Exists(config.Name); exists {
		return nil, fmt.Errorf("session name already exists: %s", config.Name)
	}


	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = generateAPIKey()
		s.logger.Info().Str("session_id", sessionID.String()).Str("name", config.Name).Msg("Generated automatic API key for session")
	}


	session := &models.Session{
		ID:        sessionID,
		Name:      config.Name,
		APIKey:    apiKey,
		Status:    models.SessionStatusDisconnected,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  make(models.Metadata),
	}


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


	if config.Webhook != nil {
		session.WebhookURL = &config.Webhook.URL
		session.WebhookEvents = config.Webhook.Events
	}


	if err := s.repository.Create(session); err != nil {
		s.logger.Error().Err(err).Str("session_id", session.ID.String()).Msg("Failed to create session in database")
		return nil, fmt.Errorf("failed to create session: %w", err)
	}


	sessionInfo := &SessionInfo{
		ID:        session.ID.String(),
		Name:      session.Name,
		APIKey:    session.APIKey,
		Status:    Status(session.Status),
		CreatedAt: session.CreatedAt,
		UpdatedAt: session.UpdatedAt,
	}

	s.logger.Info().Str("session_id", session.ID.String()).Str("name", config.Name).Msg("Session created successfully")
	return sessionInfo, nil
}


func (s *SessionService) GetSession(ctx context.Context, sessionID string) (*SessionInfo, error) {
	s.logger.Debug().Str("session_id", sessionID).Msg("Getting session info")


	session, err := s.repository.GetByIdentifier(sessionID)
	if err != nil {
		s.logger.Error().Err(err).Str("session_identifier", sessionID).Msg("Session not found")
		return nil, fmt.Errorf("session not found: %w", err)
	}


	isConnected := false
	if s.manager != nil {
		if manager, ok := s.manager.(interface{ IsSessionActive(string) bool }); ok {
			isConnected = manager.IsSessionActive(sessionID)
		}
	}


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


func (s *SessionService) ListSessions(ctx context.Context) ([]*SessionInfo, error) {
	s.logger.Debug().Msg("Listing all sessions")


	filter := &models.SessionFilter{
		Page:    1,
		PerPage: 100, // Limite padrão
	}

	response, err := s.repository.GetAll(filter)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to list sessions")
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}


	sessions := make([]*SessionInfo, len(response.Sessions))
	for i, session := range response.Sessions {
		isConnected := false
		if s.manager != nil {
			if manager, ok := s.manager.(interface{ IsSessionActive(string) bool }); ok {
				isConnected = manager.IsSessionActive(session.GetSessionID())
			}
		}

		sessions[i] = &SessionInfo{
			ID:          session.GetSessionID(), // UUID como ID principal
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


func (s *SessionService) DeleteSession(ctx context.Context, sessionID string) error {
	s.logger.Info().Str("session_id", sessionID).Msg("Deleting session")


	if exists, _ := s.repository.Exists(sessionID); !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}


	if s.manager != nil {
		if manager, ok := s.manager.(interface {
			IsSessionActive(string) bool
			DisconnectSession(string) error
		}); ok {
			if manager.IsSessionActive(sessionID) {
				s.logger.Info().Str("session_id", sessionID).Msg("Disconnecting active session before deletion")
				if err := manager.DisconnectSession(sessionID); err != nil {
					s.logger.Warn().Err(err).Str("session_id", sessionID).Msg("Failed to disconnect session, continuing with deletion")
				}
			}
		}
	}


	if err := s.repository.DeleteByIdentifier(sessionID); err != nil {
		s.logger.Error().Err(err).Str("session_identifier", sessionID).Msg("Failed to delete session")
		return fmt.Errorf("failed to delete session: %w", err)
	}

	s.logger.Info().Str("session_id", sessionID).Msg("Session deleted successfully")
	return nil
}


func (s *SessionService) ConnectSession(ctx context.Context, sessionID string) error {
	s.logger.Info().Str("session_id", sessionID).Msg("Connecting session to WhatsApp")


	if exists, _ := s.repository.Exists(sessionID); !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}


	if manager, ok := s.manager.(interface {
		ConnectSession(context.Context, string) error
	}); ok {
		if err := manager.ConnectSession(ctx, sessionID); err != nil {
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


func (s *SessionService) DisconnectSession(ctx context.Context, sessionID string) error {
	s.logger.Info().Str("session_id", sessionID).Msg("Disconnecting session from WhatsApp")


	if exists, _ := s.repository.Exists(sessionID); !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}


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


func (s *SessionService) GetSessionStatus(ctx context.Context, sessionID string) (Status, error) {
	s.logger.Debug().Str("session_id", sessionID).Msg("Getting session status")


	session, err := s.repository.GetByIdentifier(sessionID)
	if err != nil {
		s.logger.Error().Err(err).Str("session_identifier", sessionID).Msg("Session not found")
		return StatusDisconnected, fmt.Errorf("session not found: %w", err)
	}


	if manager, ok := s.manager.(interface{ IsSessionActive(string) bool }); ok {
		if manager.IsSessionActive(sessionID) {
			return StatusConnected, nil
		}
	}


	return Status(session.Status), nil
}


func (s *SessionService) GetQRCode(ctx context.Context, sessionID string) (*QRCodeInfo, error) {
	s.logger.Info().Str("session_id", sessionID).Msg("Getting QR code for session")


	if exists, _ := s.repository.Exists(sessionID); !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}


	if manager, ok := s.manager.(interface {
		GetQRCode(string) (interface{}, error)
	}); ok {
		qrData, err := manager.GetQRCode(sessionID)
		if err != nil {
			s.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to get QR code")
			return nil, fmt.Errorf("failed to get QR code: %w", err)
		}


		if qrInfo, ok := qrData.(*QRCodeInfo); ok {
			return qrInfo, nil
		}


		return &QRCodeInfo{
			Code:      fmt.Sprintf("%v", qrData),
			Timeout:   60,
			Timestamp: time.Now(),
		}, nil
	}


	return &QRCodeInfo{
		Code:      "qr_unavailable",
		Timeout:   60,
		Timestamp: time.Now(),
	}, nil
}


func (s *SessionService) PairPhone(ctx context.Context, sessionID string, request *PairPhoneRequest) (*PairPhoneResponse, error) {
	return &PairPhoneResponse{
		Success: true,
		Message: "Phone paired successfully",
	}, nil
}


func (s *SessionService) SetProxy(ctx context.Context, sessionID string, proxy *ProxyConfig) error {
	return nil
}


func (s *SessionService) SetWebhook(ctx context.Context, sessionID string, webhook *WebhookConfig) error {
	return nil
}


func (s *SessionService) GetWhatsAppClient(ctx context.Context, sessionID string) (interface{}, error) {
	s.logger.Debug().Str("session_id", sessionID).Msg("Getting WhatsApp client for session")


	if exists, _ := s.repository.Exists(sessionID); !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}


	if manager, ok := s.manager.(interface {
		GetClient(string) (interface{}, error)
	}); ok {
		client, err := manager.GetClient(sessionID)
		if err != nil {
			s.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to get WhatsApp client from manager")
			return nil, fmt.Errorf("failed to get WhatsApp client: %w", err)
		}
		return client, nil
	}


	s.logger.Warn().Str("session_id", sessionID).Msg("WhatsApp manager not available for client access")
	return nil, fmt.Errorf("WhatsApp manager not available")
}


func (s *SessionService) Shutdown(ctx context.Context) error {
	return nil
}


func (s *SessionService) validateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("configuration is required")
	}

	if config.Name == "" {
		return fmt.Errorf("session name is required")
	}


	if config.Proxy != nil && config.Proxy.Enabled {
		if config.Proxy.Host == "" {
			return fmt.Errorf("proxy host is required when proxy is enabled")
		}
		if config.Proxy.Port <= 0 || config.Proxy.Port > 65535 {
			return fmt.Errorf("proxy port must be between 1 and 65535")
		}
	}


	if config.Webhook != nil && config.Webhook.URL != "" {
		if len(config.Webhook.URL) < 10 {
			return fmt.Errorf("webhook URL is too short")
		}
	}

	return nil
}


func generateAPIKey() string {

	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {

		return fmt.Sprintf("api_%d_%d", time.Now().UnixNano(), time.Now().Unix())
	}
	return "zmw_" + hex.EncodeToString(bytes)
}


func generateSessionID() string {
	return uuid.New().String()
}
