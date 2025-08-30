package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/felipe/zemeow/internal/logger"
)


type AuthHandler struct {
	logger logger.Logger
}


func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		logger: logger.GetWithSession("auth_handler"),
	}
}


type GenerateAPIKeyRequest struct {
	SessionID string `json:"session_id"`
}



func (h *AuthHandler) ValidateAPIKey(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"valid":   true,
		"message": "API validation endpoint",
	})
}



func (h *AuthHandler) GenerateAPIKey(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"api_key": "generated-api-key",
		"message": "API key generation endpoint",
	})
}



func (h *AuthHandler) RevokeAPIKey(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success": true,
		"message": "API key revocation endpoint",
	})
}





func (h *AuthHandler) GetCacheStats(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"cache_stats": "stats",
		"message":     "Cache stats endpoint",
	})
}



func (h *AuthHandler) ClearCache(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Cache clear endpoint",
	})
}



func (h *AuthHandler) extractAPIKey(c *fiber.Ctx) string {

	authHeader := c.Get("Authorization")
	if authHeader != "" {
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			return authHeader[7:]
		}
		return authHeader
	}


	apiKey := c.Get("X-API-Key")
	if apiKey != "" {
		return apiKey
	}


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
