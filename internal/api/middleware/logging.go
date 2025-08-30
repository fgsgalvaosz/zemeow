package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

// LoggingMiddleware gerencia logging de requisições HTTP
type LoggingMiddleware struct {
	logger zerolog.Logger
}

// NewLoggingMiddleware cria um novo middleware de logging
func NewLoggingMiddleware() *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: zerolog.New(nil).With().Str("component", "http_logger").Logger(),
	}
}

// Logger middleware que registra todas as requisições HTTP
func (m *LoggingMiddleware) Logger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Processar requisição
		err := c.Next()

		// Calcular duração
		duration := time.Since(start)

		// Extrair informações da requisição
		method := c.Method()
		path := c.Path()
		status := c.Response().StatusCode()
		ip := c.IP()
		userAgent := c.Get("User-Agent")

		// Extrair sessionID do contexto de autenticação se disponível
		sessionID := ""
		if authCtx := GetAuthContext(c); authCtx != nil {
			sessionID = authCtx.SessionID
		}

		// Determinar nível de log baseado no status
		logEvent := m.logger.Info()
		if status >= 400 && status < 500 {
			logEvent = m.logger.Warn()
		} else if status >= 500 {
			logEvent = m.logger.Error()
		}

		// Log da requisição
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

// RequestID middleware que adiciona um ID único para cada requisição
func (m *LoggingMiddleware) RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Gerar ID único para a requisição
		requestID := generateRequestID()
		
		// Adicionar ao contexto
		c.Locals("request_id", requestID)
		
		// Adicionar ao header de resposta
		c.Set("X-Request-ID", requestID)
		
		return c.Next()
	}
}

// LogWithRequestID cria um logger com request ID
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

// generateRequestID gera um ID único para a requisição
func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + 
		   string(rune(time.Now().UnixNano()%26+65)) + 
		   string(rune(time.Now().UnixNano()%26+65)) + 
		   string(rune(time.Now().UnixNano()%26+65))
}
