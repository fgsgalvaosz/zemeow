package middleware

import (
	"fmt"
	"strings"
	"time"

	"github.com/felipe/zemeow/internal/db/models"
	"github.com/felipe/zemeow/internal/db/repositories"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/felipe/zemeow/internal/service/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/rs/zerolog"
)

// AuthContext representa o contexto de autenticação
type AuthContext struct {
	APIKey    string
	IsAdmin   bool
	SessionID string
	Session   *models.Session
}

// SessionInfo representa as informações de sessão no contexto
type SessionInfo struct {
	ID        string `json:"id"`
	SessionID string `json:"session_id"`
	Name      string `json:"name"`
	JID       string `json:"jid"`
	Webhook   string `json:"webhook"`
	Token     string `json:"token"`
	Proxy     string `json:"proxy"`
	Events    string `json:"events"`
	QRCode    string `json:"qrcode"`
}

// AuthMiddleware gerencia autenticação e autorização
type AuthMiddleware struct {
	authorizationService *auth.AuthService
	sessionRepo          repositories.SessionRepository
	logger               zerolog.Logger
}

// NewAuthMiddleware cria um novo middleware de autenticação
func NewAuthMiddleware(authService *auth.AuthService) *AuthMiddleware {
	return &AuthMiddleware{
		authorizationService: authService,
		logger:               logger.GetWithSession("auth_middleware"),
	}
}

// GetAuthContext extrai o contexto de autenticação do Fiber context
func GetAuthContext(c *fiber.Ctx) *AuthContext {
	if ctx := c.Locals("auth"); ctx != nil {
		return ctx.(*AuthContext)
	}
	return nil
}

// GetSessionInfo extrai as informações de sessão do contexto
func GetSessionInfo(c *fiber.Ctx) *SessionInfo {
	if info := c.Locals("sessioninfo"); info != nil {
		return info.(*SessionInfo)
	}
	return nil
}

// GetSessionID extrai o sessionID da URL ou do contexto
func GetSessionID(c *fiber.Ctx) string {
	// Primeiro tenta pegar da URL
	if sessionID := c.Params("sessionId"); sessionID != "" {
		return sessionID
	}
	// Depois tenta pegar do contexto de sessão
	if sessionInfo := GetSessionInfo(c); sessionInfo != nil {
		return sessionInfo.SessionID
	}
	return ""
}

// RequireAuth middleware que requer autenticação por token de sessão
func (am *AuthMiddleware) RequireAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Obter token do header ou query parameter
		token := c.Get("Authorization")
		if token == "" {
			token = c.Get("token")
		}
		if token == "" {
			token = c.Query("token")
		}

		// Remove prefix "Bearer " se existir
		if strings.HasPrefix(token, "Bearer ") {
			token = strings.TrimPrefix(token, "Bearer ")
		}

		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "TOKEN_REQUIRED",
				"message": "Authentication token is required",
			})
		}

		// Validar token e obter sessão
		session, err := am.sessionRepo.GetByToken(c.Context(), token)
		if err != nil {
			am.logger.Warn().Err(err).Str("token", token[:10]+"...").Msg("Invalid token provided")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "INVALID_TOKEN",
				"message": "Invalid or expired token",
			})
		}

		if session == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "SESSION_NOT_FOUND",
				"message": "Session not found for provided token",
			})
		}

		// Criar contexto de sessão
		sessionInfo := &SessionInfo{
			ID:        session.ID.String(),
			SessionID: session.SessionID,
			Name:      session.Name,
			Token:     token,
		}

		if session.JID != nil {
			sessionInfo.JID = *session.JID
		}
		if session.WebhookURL != nil {
			sessionInfo.Webhook = *session.WebhookURL
		}
		if len(session.WebhookEvents) > 0 {
			sessionInfo.Events = strings.Join(session.WebhookEvents, ",")
		}

		// Configurar contexto de autenticação
		c.Locals("auth", &AuthContext{
			APIKey:    token,
			IsAdmin:   false, // Sessões normais não são admin
			SessionID: session.SessionID,
			Session:   session,
		})

		// Configurar contexto de sessão (compatibilidade com código legado)
		c.Locals("sessioninfo", sessionInfo)

		am.logger.Info().Str("session_id", session.SessionID).Str("name", session.Name).Msg("Session authenticated successfully")

		return c.Next()
	}
}

// RequireAdmin middleware que requer privilégios de administrador
func (am *AuthMiddleware) RequireAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Obter token admin do header
		token := c.Get("Authorization")
		if token == "" {
			token = c.Get("X-Admin-Token")
		}
		if token == "" {
			token = c.Query("admin_token")
		}

		// Remove prefix "Bearer " se existir
		if strings.HasPrefix(token, "Bearer ") {
			token = strings.TrimPrefix(token, "Bearer ")
		}

		// Validar token de admin
		if !am.authorizationService.ValidateAdminToken(token) {
			am.logger.Warn().Str("token", token[:min(10, len(token))]+"...").Msg("Invalid admin token provided")
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "ADMIN_ACCESS_REQUIRED",
				"message": "Administrator access required",
			})
		}

		// Configurar contexto de admin
		c.Locals("auth", &AuthContext{
			APIKey:  token,
			IsAdmin: true,
		})

		am.logger.Info().Msg("Admin access granted")

		return c.Next()
	}
}

// RequireSessionAccess middleware que verifica se o usuário tem acesso à sessão específica
func (am *AuthMiddleware) RequireSessionAccess() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Primeiro verificar se está autenticado
		auth := GetAuthContext(c)
		if auth == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "AUTHENTICATION_REQUIRED",
				"message": "Authentication required",
			})
		}

		// Admin tem acesso a todas as sessões
		if auth.IsAdmin {
			return c.Next()
		}

		// Verificar se o sessionID da URL corresponde à sessão autenticada
		urlSessionID := c.Params("sessionId")
		if urlSessionID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "SESSION_ID_REQUIRED",
				"message": "Session ID is required in URL",
			})
		}

		if auth.SessionID != urlSessionID {
			am.logger.Warn().
				Str("authenticated_session", auth.SessionID).
				Str("requested_session", urlSessionID).
				Msg("Session access denied - session mismatch")
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "SESSION_ACCESS_DENIED",
				"message": "Access denied to this session",
			})
		}

		return c.Next()
	}
}

// min função auxiliar para obter o menor valor
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// CORS middleware
func (am *AuthMiddleware) CORS() fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS,PATCH",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-Admin-Token,token",
		AllowCredentials: false,
		ExposeHeaders:    "Content-Length,Content-Range",
	})
}

// RequestLogger middleware com suporte a sessionID
func (am *AuthMiddleware) RequestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Processar request
		start := time.Now()
		err := c.Next()
		latency := time.Since(start)

		// Obter informações da sessão se disponível
		sessionID := "anonymous"
		if sessionInfo := GetSessionInfo(c); sessionInfo != nil {
			sessionID = sessionInfo.SessionID
		} else if auth := GetAuthContext(c); auth != nil {
			if auth.IsAdmin {
				sessionID = "admin"
			} else if auth.SessionID != "" {
				sessionID = auth.SessionID
			}
		}

		// Log estruturado
		logEvent := am.logger.Info().
			Str("method", c.Method()).
			Str("path", c.Path()).
			Str("session_id", sessionID).
			Int("status", c.Response().StatusCode()).
			Dur("latency", latency).
			Str("ip", c.IP()).
			Str("user_agent", c.Get("User-Agent"))

		if err != nil {
			logEvent = am.logger.Error().Err(err)
		}

		logEvent.Msg("HTTP Request")

		return err
	}
}

// RateLimit middleware com base em sessionID
func (am *AuthMiddleware) RateLimit(max int) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        max,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			// Usar sessionID se disponível, senão IP
			if sessionInfo := GetSessionInfo(c); sessionInfo != nil {
				return fmt.Sprintf("session:%s", sessionInfo.SessionID)
			}
			if auth := GetAuthContext(c); auth != nil {
				if auth.IsAdmin {
					return "admin"
				}
				return fmt.Sprintf("token:%s", auth.APIKey[:min(10, len(auth.APIKey))])
			}
			return fmt.Sprintf("ip:%s", c.IP())
		},
		LimitReached: func(c *fiber.Ctx) error {
			am.logger.Warn().
				Str("ip", c.IP()).
				Str("path", c.Path()).
				Msg("Rate limit exceeded")
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   "RATE_LIMIT_EXCEEDED",
				"message": "Too many requests, please try again later",
			})
		},
	})
}