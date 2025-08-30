package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/felipe/zemeow/internal/logger"
)

// AuthHandler gerencia endpoints de autenticação
type AuthHandler struct {
	logger logger.Logger
}

// NewAuthHandler cria uma nova instância do handler de autenticação
func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		logger: logger.GetWithSession("auth_handler"),
	}
}

// GenerateAPIKeyRequest representa uma requisição de geração de API Key
type GenerateAPIKeyRequest struct {
	SessionID string `json:"session_id"`
}

// ValidateAPIKey valida uma API Key
// POST /auth/validate
func (h *AuthHandler) ValidateAPIKey(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"valid":   true,
		"message": "API validation endpoint",
	})
}

// GenerateAPIKey gera uma nova API Key para uma sessão
// POST /auth/generate
func (h *AuthHandler) GenerateAPIKey(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"api_key": "generated-api-key",
		"message": "API key generation endpoint",
	})
}

// RevokeAPIKey revoga uma API Key
// POST /auth/revoke
func (h *AuthHandler) RevokeAPIKey(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success": true,
		"message": "API key revocation endpoint",
	})
}



// GetCacheStats retorna estatísticas do cache de autenticação
// GET /auth/stats
func (h *AuthHandler) GetCacheStats(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"cache_stats": "stats",
		"message":     "Cache stats endpoint",
	})
}

// ClearCache limpa o cache de autenticação
// POST /auth/clear-cache
func (h *AuthHandler) ClearCache(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Cache clear endpoint",
	})
}

// Métodos auxiliares

func (h *AuthHandler) extractAPIKey(c *fiber.Ctx) string {
	// Tentar header Authorization
	authHeader := c.Get("Authorization")
	if authHeader != "" {
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			return authHeader[7:]
		}
		return authHeader
	}

	// Tentar header X-API-Key
	apiKey := c.Get("X-API-Key")
	if apiKey != "" {
		return apiKey
	}

	// Tentar query parameter
	return c.Query("api_key")
}

func (h *AuthHandler) sendError(c *fiber.Ctx, message, code string, status int) error {
	errorResp := fiber.Map{
		"error":   code,
		"message": message,
		"status":  status,
	}

	return c.Status(status).JSON(errorResp)
}
