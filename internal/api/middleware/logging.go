package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)


type LoggingMiddleware struct {
	logger zerolog.Logger
}


func NewLoggingMiddleware() *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: zerolog.New(nil).With().Str("component", "http_logger").Logger(),
	}
}


func (m *LoggingMiddleware) Logger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()


		err := c.Next()


		duration := time.Since(start)


		method := c.Method()
		path := c.Path()
		status := c.Response().StatusCode()
		ip := c.IP()
		userAgent := c.Get("User-Agent")


		sessionID := ""
		if authCtx := GetAuthContext(c); authCtx != nil {
			sessionID = authCtx.SessionID
		}


		logEvent := m.logger.Info()
		if status >= 400 && status < 500 {
			logEvent = m.logger.Warn()
		} else if status >= 500 {
			logEvent = m.logger.Error()
		}


		logEvent.
			Str("method", method).
			Str("path", path).
			Int("status", status).
			Dur("duration", duration).
			Str("ip", ip).
			Str("user_agent", userAgent).
			Str("session_id", sessionID).
			Msg("HTTP request")

		return err
	}
}


func (m *LoggingMiddleware) RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {

		requestID := generateRequestID()
		

		c.Locals("request_id", requestID)
		

		c.Set("X-Request-ID", requestID)
		
		return c.Next()
	}
}


func (m *LoggingMiddleware) LogWithRequestID(c *fiber.Ctx) zerolog.Logger {
	requestID := ""
	if id := c.Locals("request_id"); id != nil {
		requestID = id.(string)
	}
	
	sessionID := ""
	if authCtx := GetAuthContext(c); authCtx != nil {
		sessionID = authCtx.SessionID
	}
	
	return m.logger.With().
		Str("request_id", requestID).
		Str("session_id", sessionID).
		Logger()
}


func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + 
		   string(rune(time.Now().UnixNano()%26+65)) + 
		   string(rune(time.Now().UnixNano()%26+65)) + 
		   string(rune(time.Now().UnixNano()%26+65))
}
