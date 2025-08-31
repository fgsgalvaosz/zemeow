package middleware

import (
	"fmt"
	"strings"
	"time"

	"github.com/felipe/zemeow/internal/db/models"
	"github.com/felipe/zemeow/internal/db/repositories"
	"github.com/felipe/zemeow/internal/logger"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

type AuthContext struct {
	APIKey          string
	IsGlobalKey     bool
	SessionID       string
	HasGlobalAccess bool
}

type SessionInfo struct {
	ID        string `json:"id"`
	SessionID string `json:"session_id"`
	Name      string `json:"name"`
	JID       string `json:"jid"`
	Webhook   string `json:"webhook"`

	Proxy  string `json:"proxy"`
	Events string `json:"events"`
	QRCode string `json:"qrcode"`
}

type AuthMiddleware struct {
	adminAPIKey string
	sessionRepo repositories.SessionRepository
	logger      logger.Logger
}

func NewAuthMiddleware(adminAPIKey string, sessionRepo repositories.SessionRepository) *AuthMiddleware {
	return &AuthMiddleware{
		adminAPIKey: adminAPIKey,
		sessionRepo: sessionRepo,
		logger:      logger.GetWithSession("auth_middleware"),
	}
}

func GetAuthContext(c *fiber.Ctx) *AuthContext {
	if ctx := c.Locals("auth"); ctx != nil {
		return ctx.(*AuthContext)
	}
	return nil
}

func GetSessionInfo(c *fiber.Ctx) *SessionInfo {
	if info := c.Locals("sessioninfo"); info != nil {
		return info.(*SessionInfo)
	}
	return nil
}

func GetSessionID(c *fiber.Ctx) string {

	if sessionID := c.Params("sessionId"); sessionID != "" {
		return sessionID
	}

	if sessionInfo := GetSessionInfo(c); sessionInfo != nil {
		return sessionInfo.SessionID
	}
	return ""
}

func (am *AuthMiddleware) extractAPIKey(c *fiber.Ctx) string {

	if apiKey := c.Get("apikey"); apiKey != "" {
		return apiKey
	}

	if apiKey := c.Get("X-API-Key"); apiKey != "" {
		return apiKey
	}

	token := c.Get("Authorization")
	if token != "" {

		if strings.HasPrefix(token, "Bearer ") {
			return strings.TrimPrefix(token, "Bearer ")
		}
		return token
	}

	return ""
}

func (am *AuthMiddleware) RequireAPIKey() fiber.Handler {
	return func(c *fiber.Ctx) error {

		apiKey := am.extractAPIKey(c)
		if apiKey == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "MISSING_API_KEY",
				"message": "API key is required in 'apikey', 'X-API-Key' or 'Authorization' header",
			})
		}

		if apiKey == am.adminAPIKey {

			authCtx := &AuthContext{
				APIKey:          apiKey,
				IsGlobalKey:     true,
				HasGlobalAccess: true,
			}
			c.Locals("auth", authCtx)
			am.logger.Info().Str("type", "global").Msg("Global authentication successful")
			return c.Next()
		}

		session, err := am.sessionRepo.GetByAPIKey(apiKey)
		if err != nil {
			am.logger.Warn().Err(err).Str("api_key_prefix", apiKey[:min(len(apiKey), 8)]+"...").Msg("Session API key validation failed")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "INVALID_API_KEY",
				"message": "Invalid API key provided",
			})
		}

		authCtx := &AuthContext{
			APIKey:          apiKey,
			IsGlobalKey:     false,
			SessionID:       session.GetSessionID(),
			HasGlobalAccess: false,
		}
		c.Locals("auth", authCtx)
		c.Locals("session", session)

		am.logger.Info().Str("type", "session").Str("session_id", session.GetSessionID()).Str("name", session.Name).Msg("Session authentication successful")
		return c.Next()
	}
}

func (am *AuthMiddleware) RequireGlobalAPIKey() fiber.Handler {
	return func(c *fiber.Ctx) error {

		apiKey := am.extractAPIKey(c)

		if apiKey == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "MISSING_API_KEY",
				"message": "Global API key is required",
			})
		}

		if apiKey != am.adminAPIKey {
			am.logger.Warn().Str("provided_key_prefix", apiKey[:min(len(apiKey), 8)]+"...").Str("expected_key_prefix", am.adminAPIKey[:min(len(am.adminAPIKey), 8)]+"...").Msg("Global access denied - invalid API key")
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "GLOBAL_ACCESS_REQUIRED",
				"message": "Global access required - invalid Global API key",
			})
		}

		c.Locals("auth", &AuthContext{
			APIKey:          apiKey,
			IsGlobalKey:     true,
			HasGlobalAccess: true,
		})

		am.logger.Info().Msg("Global authentication successful")
		return c.Next()
	}
}

func (am *AuthMiddleware) RequireSessionAccess() fiber.Handler {
	return func(c *fiber.Ctx) error {

		auth := GetAuthContext(c)
		if auth == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "AUTHENTICATION_REQUIRED",
				"message": "Authentication required",
			})
		}

		if auth.IsGlobalKey {
			return c.Next()
		}

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

func (am *AuthMiddleware) SessionInfoMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {

		auth := GetAuthContext(c)
		if auth == nil {
			return c.Next()
		}

		if auth.IsGlobalKey {
			return c.Next()
		}

		sessionInterface := c.Locals("session")
		if sessionInterface == nil {
			return c.Next()
		}

		session, ok := sessionInterface.(*models.Session)
		if !ok {
			return c.Next()
		}

		info := &SessionInfo{
			ID:        session.ID.String(),
			SessionID: session.GetSessionID(),
			Name:      session.Name,
		}

		if session.JID != nil {
			info.JID = *session.JID
		}

		if session.WebhookURL != nil {
			info.Webhook = *session.WebhookURL
		}

		if session.ProxyHost != nil && session.ProxyPort != nil {
			info.Proxy = fmt.Sprintf("%s:%d", *session.ProxyHost, *session.ProxyPort)
		}

		if len(session.WebhookEvents) > 0 {
			info.Events = strings.Join(session.WebhookEvents, ",")
		}

		c.Locals("sessioninfo", info)
		return c.Next()
	}
}

func (am *AuthMiddleware) CORSMiddleware() fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization,apikey,X-API-Key",
	})
}

func (am *AuthMiddleware) RateLimiterMiddleware() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        60,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {

			ip := c.IP()
			apiKey := am.extractAPIKey(c)
			if apiKey != "" {
				return ip + ":" + apiKey
			}
			return ip
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   "RATE_LIMIT_EXCEEDED",
				"message": "Too many requests - rate limit exceeded",
			})
		},
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
