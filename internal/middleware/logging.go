package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/felipe/zemeow/internal/logger"
	"github.com/gofiber/fiber/v2"
)

type LoggingMiddleware struct {
	logger    *logger.ComponentLogger
	headerKey string
}

func NewLoggingMiddleware() *LoggingMiddleware {
	return &LoggingMiddleware{
		logger:    logger.ForComponent("api"),
		headerKey: "X-Request-ID",
	}
}

func (m *LoggingMiddleware) Logger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Generate request ID
		requestID := m.generateRequestID()
		c.Locals("request_id", requestID)
		c.Set(m.headerKey, requestID)

		// Extract session context
		sessionID := m.extractSessionID(c)

		// Create request logger with full context
		reqLogger := logger.ForRequestContext("api", sessionID, requestID)
		requestOp := reqLogger.ForOperation("request")

		// Log request start
		start := time.Now()
		requestOp.Starting().
			Str("method", c.Method()).
			Str("path", c.Path()).
			Str("ip", c.IP()).
			Str("user_agent", c.Get("User-Agent")).
			Msg(logger.GetStandardizedMessage("api", "request", "starting"))

		// Process request
		err := c.Next()

		// Log request completion
		duration := time.Since(start)
		status := c.Response().StatusCode()

		// Determine log level based on status
		var logEvent *logger.OperationLogger
		if status >= 400 {
			logEvent = reqLogger.ForOperation("request")
			if status >= 500 {
				logEvent.Failed("SERVER_ERROR").
					Str("method", c.Method()).
					Str("path", c.Path()).
					Int("status", status).
					Dur("duration", duration).
					Msg(logger.GetStandardizedMessage("api", "request", "failed"))
			} else {
				logEvent.Warn().
					Str("method", c.Method()).
					Str("path", c.Path()).
					Int("status", status).
					Dur("duration", duration).
					Str("status_text", "client_error").
					Msg("HTTP request completed with client error")
			}
		} else {
			requestOp.Success().
				Str("method", c.Method()).
				Str("path", c.Path()).
				Int("status", status).
				Str("status_text", "success").
				Msg(logger.GetStandardizedMessage("api", "request", "success"))
		}

		return err
	}
}

func (m *LoggingMiddleware) RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if request ID already exists
		requestID := c.Get(m.headerKey)
		if requestID == "" {
			requestID = m.generateRequestID()
		}

		c.Locals("request_id", requestID)
		c.Set(m.headerKey, requestID)

		return c.Next()
	}
}

// generateRequestID creates a unique request identifier
func (m *LoggingMiddleware) generateRequestID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return "req_" + hex.EncodeToString(bytes)
}

// extractSessionID extracts session ID from request context
func (m *LoggingMiddleware) extractSessionID(c *fiber.Ctx) string {
	if authCtx := GetAuthContext(c); authCtx != nil {
		return authCtx.SessionID
	}
	return ""
}

func (m *LoggingMiddleware) LogWithRequestID(c *fiber.Ctx) *logger.RequestLogger {
	requestID := ""
	if id := c.Locals("request_id"); id != nil {
		requestID = id.(string)
	}

	sessionID := m.extractSessionID(c)

	return logger.ForRequestContext("api", sessionID, requestID)
}
