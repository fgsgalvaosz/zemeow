package middleware

import (
	"fmt"
	"strings"
	"time"

	"github.com/felipe/zemeow/internal/db/repositories"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// AuthContext representa o contexto de autenticação
type AuthContext struct {
	APIKey        string
	IsGlobalKey   bool
	SessionID     string
	HasGlobalAccess bool
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
	adminAPIKey    string
	sessionRepo    repositories.SessionRepository
	logger         logger.Logger
}

// NewAuthMiddleware cria um novo middleware de autenticação
func NewAuthMiddleware(adminAPIKey string, sessionRepo repositories.SessionRepository) *AuthMiddleware {
	return &AuthMiddleware{
		adminAPIKey: adminAPIKey,
		sessionRepo: sessionRepo,
		logger:      logger.GetWithSession("auth_middleware"),
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

// extractAPIKey extrai a API key dos headers
func (am *AuthMiddleware) extractAPIKey(c *fiber.Ctx) string {
	// Primeiro tentar header apikey (método preferido)
	if apiKey := c.Get("apikey"); apiKey != "" {
		return apiKey
	}
	
	// Segundo tentar X-API-Key header
	if apiKey := c.Get("X-API-Key"); apiKey != "" {
		return apiKey
	}

	// Terceiro tentar Authorization header
	token := c.Get("Authorization")
	if token != "" {
		// Remover prefixo "Bearer " se presente
		if strings.HasPrefix(token, "Bearer ") {
			return strings.TrimPrefix(token, "Bearer ")
		}
		return token
	}

	return ""
}

// RequireAPIKey middleware que requer autenticação válida (Global ou Session API Key)
func (am *AuthMiddleware) RequireAPIKey() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extrair API key
		apiKey := am.extractAPIKey(c)
		if apiKey == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "MISSING_API_KEY",
				"message": "API key is required in 'apikey', 'X-API-Key' or 'Authorization' header",
			})
		}

		// Verificar se é Global API Key
		if apiKey == am.adminAPIKey {
			// Criar contexto global
			authCtx := &AuthContext{
				APIKey:          apiKey,
				IsGlobalKey:     true,
				HasGlobalAccess: true,
			}
			c.Locals("auth", authCtx)
			am.logger.Info().Str("type", "global").Msg("Global authentication successful")
			return c.Next()
		}

		// Verificar se é Session API Key
		session, err := am.sessionRepo.GetByAPIKey(apiKey)
		if err != nil {
			am.logger.Warn().Err(err).Str("api_key_prefix", apiKey[:min(len(apiKey), 8)]+"...").Msg("Session API key validation failed")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "INVALID_API_KEY",
				"message": "Invalid API key provided",
			})
		}

		// Criar contexto de sessão
		authCtx := &AuthContext{
			APIKey:          apiKey,
			IsGlobalKey:     false,
			SessionID:       session.SessionID,
			HasGlobalAccess: false,
		}
		c.Locals("auth", authCtx)
		c.Locals("session", session)

		am.logger.Info().Str("type", "session").Str("session_id", session.SessionID).Msg("Session authentication successful")
		return c.Next()
	}
}

// RequireGlobalAPIKey middleware que requer privilégios globais
func (am *AuthMiddleware) RequireGlobalAPIKey() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extrair API key
		apiKey := am.extractAPIKey(c)
		if apiKey == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "MISSING_API_KEY",
				"message": "Global API key is required",
			})
		}

		// Validar Global API Key
		if apiKey != am.adminAPIKey {
			am.logger.Warn().Str("provided_key_prefix", apiKey[:min(len(apiKey), 8)]+"...").Msg("Global access denied - invalid API key")
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "GLOBAL_ACCESS_REQUIRED",
				"message": "Global access required - invalid Global API key",
			})
		}

		// Configurar contexto global
		c.Locals("auth", &AuthContext{
			APIKey:          apiKey,
			IsGlobalKey:     true,
			HasGlobalAccess: true,
		})

		am.logger.Info().Msg("Global authentication successful")
		return c.Next()
	}
}

// RequireSessionAccess middleware que verifica acesso à sessão específica
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

		// Global key tem acesso a todas as sessões
		if auth.IsGlobalKey {
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
			if auth.IsGlobalKey {
				sessionID = "global"
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
				if auth.IsGlobalKey {
					return "global"
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

// CORS middleware com suporte aos headers de API key
func (am *AuthMiddleware) CORS() fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-API-Key,apikey",
		ExposeHeaders:    "Content-Length",
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
	})
}