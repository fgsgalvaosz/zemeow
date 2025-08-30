package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/felipe/zemeow/internal/config"
	"github.com/felipe/zemeow/internal/db/repositories"
	"github.com/rs/zerolog"
)

// AuthService gerencia autenticação simples com API Key
type AuthService struct {
	mu         sync.RWMutex
	repository repositories.SessionRepository
	config     *config.Config
	logger     zerolog.Logger
	adminToken string
	apiKeys    map[string]string // apiKey -> sessionID
}

// AuthContext representa o contexto de autenticação
type AuthContext struct {
	SessionID string
	Role      string
	IsAdmin   bool
}

// AuthResponse representa a resposta de autenticação
type AuthResponse struct {
	APIKey    string `json:"api_key"`
	SessionID string `json:"session_id,omitempty"`
	Role      string `json:"role"`
	Valid     bool   `json:"valid"`
}

const (
	// Roles
	RoleAdmin   = "admin"
	RoleSession = "session"
)

// NewAuthService cria uma nova instância do serviço de autenticação simples
func NewAuthService(repository repositories.SessionRepository, config *config.Config) (*AuthService, error) {
	// Gerar admin token se não fornecido
	adminToken := config.Auth.AdminToken
	if adminToken == "" {
		token := make([]byte, 32)
		if _, err := rand.Read(token); err != nil {
			return nil, fmt.Errorf("failed to generate admin token: %w", err)
		}
		adminToken = hex.EncodeToString(token)
	}
	
	service := &AuthService{
		repository: repository,
		config:     config,
		logger:     zerolog.New(nil).With().Str("component", "auth_service").Logger(),
		adminToken: adminToken,
		apiKeys:    make(map[string]string),
	}
	
	service.logger.Info().Str("admin_token", adminToken).Msg("Auth service initialized")
	return service, nil
}

// ValidateAPIKey valida uma API Key
func (s *AuthService) ValidateAPIKey(apiKey string) (*AuthContext, error) {
	// Verificar se é o token de admin
	if apiKey == s.adminToken {
		return &AuthContext{
			SessionID: "admin",
			Role:      RoleAdmin,
			IsAdmin:   true,
		}, nil
	}
	
	s.mu.RLock()
	sessionID, exists := s.apiKeys[apiKey]
	s.mu.RUnlock()
	
	if !exists {
		// Tentar buscar no banco de dados pelo token
		session, err := s.repository.GetByToken(apiKey)
		if err != nil {
			return nil, fmt.Errorf("invalid API key")
		}
		
		// Adicionar ao cache
		s.mu.Lock()
		s.apiKeys[apiKey] = session.SessionID
		s.mu.Unlock()
		
		sessionID = session.SessionID
	}
	
	// Verificar se a sessão ainda existe
	exists, err := s.repository.Exists(sessionID)
	if err != nil || !exists {
		// Remover do cache se não existe mais
		s.mu.Lock()
		delete(s.apiKeys, apiKey)
		s.mu.Unlock()
		return nil, fmt.Errorf("session not found")
	}
	
	return &AuthContext{
		SessionID: sessionID,
		Role:      RoleSession,
		IsAdmin:   false,
	}, nil
}

// GenerateAPIKey gera uma nova API Key para uma sessão
func (s *AuthService) GenerateAPIKey(sessionID string) (*AuthResponse, error) {
	// Verificar se a sessão existe
	session, err := s.repository.GetBySessionID(sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}
	
	// Usar o token da sessão como API Key
	apiKey := session.Token
	
	// Adicionar ao cache
	s.mu.Lock()
	s.apiKeys[apiKey] = sessionID
	s.mu.Unlock()
	
	s.logger.Info().Str("session_id", sessionID).Msg("API key generated")
	
	return &AuthResponse{
		APIKey:    apiKey,
		SessionID: sessionID,
		Role:      RoleSession,
		Valid:     true,
	}, nil
}

// RevokeAPIKey revoga uma API Key
func (s *AuthService) RevokeAPIKey(apiKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	sessionID, exists := s.apiKeys[apiKey]
	if exists {
		delete(s.apiKeys, apiKey)
		s.logger.Info().Str("session_id", sessionID).Msg("API key revoked")
	}
	
	return nil
}

// GetAdminToken retorna o token de admin
func (s *AuthService) GetAdminToken() string {
	return s.adminToken
}

// HasPermission verifica se o contexto tem permissão para uma ação
func (s *AuthService) HasPermission(ctx *AuthContext, action string, resource string) bool {
	// Admin tem todas as permissões
	if ctx.IsAdmin {
		return true
	}
	
	// Sessões só podem acessar seus próprios recursos
	if ctx.Role == RoleSession {
		switch action {
		case "read", "update", "delete":
			// Pode acessar apenas sua própria sessão
			return resource == ctx.SessionID || resource == ""
		case "create":
			// Não pode criar novas sessões
			return false
		default:
			return false
		}
	}
	
	return false
}

// ValidateSessionAccess valida se o contexto pode acessar uma sessão específica
func (s *AuthService) ValidateSessionAccess(ctx *AuthContext, sessionID string) error {
	if ctx.IsAdmin {
		return nil // Admin pode acessar qualquer sessão
	}
	
	if ctx.Role == RoleSession && ctx.SessionID == sessionID {
		return nil // Sessão pode acessar a si mesma
	}
	
	return fmt.Errorf("access denied")
}

// GetAPIKeyInfo retorna informações sobre uma API Key
func (s *AuthService) GetAPIKeyInfo(apiKey string) (map[string]interface{}, error) {
	ctx, err := s.ValidateAPIKey(apiKey)
	if err != nil {
		return nil, err
	}
	
	return map[string]interface{}{
		"session_id": ctx.SessionID,
		"role":       ctx.Role,
		"is_admin":   ctx.IsAdmin,
		"valid":      true,
	}, nil
}

// ClearCache limpa o cache de API Keys
func (s *AuthService) ClearCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.apiKeys = make(map[string]string)
	s.logger.Info().Msg("API key cache cleared")
}

// GetCacheStats retorna estatísticas do cache
func (s *AuthService) GetCacheStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return map[string]interface{}{
		"cached_keys": len(s.apiKeys),
	}
}
