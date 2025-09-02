package utils

import (
	"github.com/felipe/zemeow/internal/middleware"
	"github.com/gofiber/fiber/v2"
)

// HasSessionAccess verifica se o usuário tem acesso à sessão especificada
func HasSessionAccess(c *fiber.Ctx, sessionID string) bool {
	authCtx := middleware.GetAuthContext(c)
	if authCtx == nil {
		return false
	}

	// Chave global tem acesso a todas as sessões
	if authCtx.IsGlobalKey {
		return true
	}

	// Chave de sessão específica só tem acesso à própria sessão
	return authCtx.SessionID == sessionID
}

// RequireSessionAccess middleware que verifica acesso à sessão e retorna erro padronizado
func RequireSessionAccess(sessionID string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !HasSessionAccess(c, sessionID) {
			return SendAccessDeniedError(c)
		}
		return c.Next()
	}
}

// GetAuthContext é um wrapper para facilitar o acesso ao contexto de autenticação
func GetAuthContext(c *fiber.Ctx) *middleware.AuthContext {
	return middleware.GetAuthContext(c)
}

// IsGlobalKey verifica se a chave atual é uma chave global
func IsGlobalKey(c *fiber.Ctx) bool {
	authCtx := GetAuthContext(c)
	return authCtx != nil && authCtx.IsGlobalKey
}

// GetSessionID retorna o ID da sessão do contexto de autenticação
func GetSessionID(c *fiber.Ctx) string {
	authCtx := GetAuthContext(c)
	if authCtx == nil {
		return ""
	}
	return authCtx.SessionID
}
