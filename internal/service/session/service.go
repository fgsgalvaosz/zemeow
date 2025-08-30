package session

import (
	"context"
	"errors"
	"fmt"
	"time"
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

	// Lifecycle
	Shutdown(ctx context.Context) error
}

// SessionService implementa a interface Service
type SessionService struct {
	manager *Manager
}

// NewService cria um novo service de sessão
func NewService() Service {
	return &SessionService{
		manager: NewManager(),
	}
}

// CreateSession implementa Service.CreateSession
func (s *SessionService) CreateSession(ctx context.Context, config *Config) (*SessionInfo, error) {
	// Validar configuração
	if err := s.validateConfig(config); err != nil {
		return nil, err
	}

	// Gerar ID único para a sessão
	sessionID := generateSessionID()

	// Criar sessão no manager
	session, err := s.manager.CreateSession(sessionID, config)
	if err != nil {
		return nil, err
	}

	return session.GetInfo(), nil
}

// GetSession implementa Service.GetSession
func (s *SessionService) GetSession(ctx context.Context, sessionID string) (*SessionInfo, error) {
	session, exists := s.manager.GetSession(sessionID)
	if !exists {
		return nil, ErrSessionNotFound
	}

	return session.GetInfo(), nil
}

// ListSessions implementa Service.ListSessions
func (s *SessionService) ListSessions(ctx context.Context) ([]*SessionInfo, error) {
	sessions := s.manager.ListSessions()
	infos := make([]*SessionInfo, len(sessions))

	for i, session := range sessions {
		infos[i] = session.GetInfo()
	}

	return infos, nil
}

// DeleteSession implementa Service.DeleteSession
func (s *SessionService) DeleteSession(ctx context.Context, sessionID string) error {
	return s.manager.DeleteSession(sessionID)
}

// ConnectSession implementa Service.ConnectSession
func (s *SessionService) ConnectSession(ctx context.Context, sessionID string) error {
	return s.manager.ConnectSession(ctx, sessionID)
}

// DisconnectSession implementa Service.DisconnectSession
func (s *SessionService) DisconnectSession(ctx context.Context, sessionID string) error {
	return s.manager.DisconnectSession(sessionID)
}

// GetSessionStatus implementa Service.GetSessionStatus
func (s *SessionService) GetSessionStatus(ctx context.Context, sessionID string) (Status, error) {
	return s.manager.GetSessionStatus(sessionID)
}

// GetQRCode implementa Service.GetQRCode
func (s *SessionService) GetQRCode(ctx context.Context, sessionID string) (*QRCodeInfo, error) {
	// TODO: Implementar geração de QR Code
	// Isso será implementado na fase de conexão WhatsApp
	return nil, errors.New("not implemented yet")
}

// PairPhone implementa Service.PairPhone
func (s *SessionService) PairPhone(ctx context.Context, sessionID string, request *PairPhoneRequest) (*PairPhoneResponse, error) {
	// TODO: Implementar pareamento por telefone
	// Isso será implementado na fase de conexão WhatsApp
	return nil, errors.New("not implemented yet")
}

// SetProxy implementa Service.SetProxy
func (s *SessionService) SetProxy(ctx context.Context, sessionID string, proxy *ProxyConfig) error {
	session, exists := s.manager.GetSession(sessionID)
	if !exists {
		return ErrSessionNotFound
	}

	session.Config.Proxy = proxy
	return nil
}

// SetWebhook implementa Service.SetWebhook
func (s *SessionService) SetWebhook(ctx context.Context, sessionID string, webhook *WebhookConfig) error {
	session, exists := s.manager.GetSession(sessionID)
	if !exists {
		return ErrSessionNotFound
	}

	session.Config.Webhook = webhook
	return nil
}

// Shutdown implementa Service.Shutdown
func (s *SessionService) Shutdown(ctx context.Context) error {
	return s.manager.Shutdown(ctx)
}

// validateConfig valida a configuração da sessão
func (s *SessionService) validateConfig(config *Config) error {
	if config == nil {
		return ErrInvalidConfig
	}

	if config.Name == "" {
		return errors.New("session name is required")
	}

	if config.Token == "" {
		return errors.New("session token is required")
	}

	return nil
}

// generateSessionID gera um ID único para a sessão
func generateSessionID() string {
	// TODO: Implementar geração de ID único
	// Por enquanto, usar timestamp + random
	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}
