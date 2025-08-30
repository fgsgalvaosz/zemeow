package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/felipe/zemeow/internal/config"
	"github.com/felipe/zemeow/internal/db/repositories"
	"github.com/felipe/zemeow/internal/logger"
)

// AuthContext representa o contexto de autenticação
type AuthContext struct {
	SessionID string
	Role      string
	IsAdmin   bool
}

// AuthService gerencia autenticação e autorização
type AuthService struct {
	sessionRepo repositories.SessionRepository
	config      *config.Config
	logger      logger.Logger
	apiKeys     map[string]*AuthContext
	mutex       sync.RWMutex
}

// NewAuthService cria uma nova instância do serviço de autenticação
func NewAuthService(sessionRepo repositories.SessionRepository, cfg *config.Config) (*AuthService, error) {
	service := &AuthService{
		sessionRepo: sessionRepo,
		config:      cfg,
		logger:      logger.GetWithSession("auth_service"),
		apiKeys:     make(map[string]*AuthContext),
	}

	// Gerar token de admin padrão
	adminToken := service.generateAPIKey()
	service.apiKeys[adminToken] = &AuthContext{
		SessionID: "admin",
		Role:      "admin",
		IsAdmin:   true,
	}

	service.logger.Info().Str("admin_token", adminToken).Msg("Admin token generated")
	return service, nil
}

// ValidateAPIKey valida uma API Key
func (s *AuthService) ValidateAPIKey(apiKey string) (*AuthContext, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if authCtx, exists := s.apiKeys[apiKey]; exists {
		return authCtx, nil
	}

	return nil, fmt.Errorf("invalid API key")
}

// GenerateAPIKey gera uma nova API Key para uma sessão
func (s *AuthService) GenerateAPIKey(sessionID string) (map[string]interface{}, error) {
	// Mock: Verificação simplificada de sessão

	// Gerar nova API Key
	apiKey := s.generateAPIKey()

	// Armazenar no cache
	s.mutex.Lock()
	s.apiKeys[apiKey] = &AuthContext{
		SessionID: sessionID,
		Role:      "user",
		IsAdmin:   false,
	}
	s.mutex.Unlock()

	response := map[string]interface{}{
		"api_key":    apiKey,
		"session_id": sessionID,
		"expires_in": "never",
		"created_at": time.Now(),
	}

	s.logger.Info().Str("session_id", sessionID).Msg("API key generated")
	return response, nil
}

// RevokeAPIKey revoga uma API Key
func (s *AuthService) RevokeAPIKey(apiKey string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exists := s.apiKeys[apiKey]; !exists {
		return fmt.Errorf("API key not found")
	}

	delete(s.apiKeys, apiKey)
	s.logger.Info().Msg("API key revoked")
	return nil
}

// GetAPIKeyInfo retorna informações sobre uma API Key
func (s *AuthService) GetAPIKeyInfo(apiKey string) (map[string]interface{}, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if authCtx, exists := s.apiKeys[apiKey]; exists {
		return map[string]interface{}{
			"session_id": authCtx.SessionID,
			"role":       authCtx.Role,
			"is_admin":   authCtx.IsAdmin,
			"valid":      true,
		}, nil
	}

	return map[string]interface{}{
		"valid": false,
	}, nil
}

// GetCacheStats retorna estatísticas do cache
func (s *AuthService) GetCacheStats() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return map[string]interface{}{
		"total_keys": len(s.apiKeys),
		"cache_type": "in_memory",
	}
}

// ClearCache limpa o cache de API Keys
func (s *AuthService) ClearCache() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Manter apenas o token de admin
	adminKeys := make(map[string]*AuthContext)
	for key, ctx := range s.apiKeys {
		if ctx.IsAdmin {
			adminKeys[key] = ctx
		}
	}

	s.apiKeys = adminKeys
	s.logger.Info().Msg("API key cache cleared")
}

// generateAPIKey gera uma API Key aleatória
func (s *AuthService) generateAPIKey() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return "zemeow_" + hex.EncodeToString(bytes)
}
